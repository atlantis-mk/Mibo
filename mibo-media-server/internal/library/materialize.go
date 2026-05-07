package library

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

const (
	sqliteVariableChunkSize    = 400
	sqliteWideWriteBatchSize   = 25
	sqliteMediumWriteBatchSize = 50
	sqliteNarrowWriteBatchSize = 100
)

func (s *Service) RunCatalogMaterializeBatch(ctx context.Context, payload CatalogMaterializeBatchPayload) error {
	config, err := s.EffectiveLibraryConfig(ctx, payload.LibraryID)
	if err != nil {
		return err
	}
	fileIDs := normalizeUintIDs(payload.FileIDs)
	if len(fileIDs) == 0 {
		return nil
	}
	batchState, err := s.newMaterializeBatchState(ctx, config)
	if err != nil {
		return err
	}
	if payload.mode != nil {
		for pathKey, snapshot := range payload.mode.directorySnapshots {
			if strings.TrimSpace(pathKey) == "" {
				continue
			}
			batchState.directorySnapshots[pathKey] = snapshot
		}
	}
	directorySnapshots := make(map[string]scanDirectorySnapshot)
	for pathKey, snapshot := range batchState.directorySnapshots {
		directorySnapshots[pathKey] = snapshot
	}
	if err := s.preloadMaterializeBatchFiles(ctx, config, &batchState, fileIDs); err != nil {
		return err
	}
	if err := s.preloadMaterializeBatchCatalogData(ctx, config, &batchState, fileIDs, directorySnapshots); err != nil {
		return err
	}
	if err := s.materializeBatchCreateMissingCatalogData(ctx, config, &batchState); err != nil {
		return err
	}
	materializedFileIDs := make([]uint, 0, len(fileIDs))
	materializedItemIDs := make([]uint, 0, len(fileIDs))
	materializeDirtyReasons := make(map[uint]string, len(fileIDs))
	materializeEvents := make([]database.IngestEvent, 0, len(fileIDs))
	var materializeErrors []error
	for _, fileID := range fileIDs {
		result, err := s.materializeInventoryFile(ctx, config, batchState, fileID, directorySnapshots)
		if err != nil {
			materializeDirtyReasons[fileID] = "materialization_failed"
			materializeEvents = append(materializeEvents, database.IngestEvent{UnitKey: inventoryFileUnitKey(fileID), LibraryID: config.Library.ID, InventoryFileID: &fileID, ConditionType: ingest.ConditionMaterialized, EventType: ingest.EventConditionChanged, Status: ingest.ConditionStatusFailed, Reason: "materialization_failed", Message: err.Error()})
			materializeErrors = append(materializeErrors, fmt.Errorf("materialize inventory file %d: %w", fileID, err))
			continue
		}
		if result.File.ID != 0 {
			materializedFileIDs = append(materializedFileIDs, result.File.ID)
			if result.File.ID != fileID {
				materializedFileIDs = append(materializedFileIDs, fileID)
			}
		}
		if result.Item.ID != 0 {
			materializedItemIDs = append(materializedItemIDs, result.Item.ID)
		}
		if strings.TrimSpace(result.SkippedReason) != "" {
			reason := strings.TrimSpace(result.SkippedReason)
			materializeDirtyReasons[fileID] = reason
			materializeEvents = append(materializeEvents, database.IngestEvent{UnitKey: inventoryFileUnitKey(fileID), LibraryID: config.Library.ID, InventoryFileID: &fileID, ConditionType: ingest.ConditionMaterialized, EventType: ingest.EventConditionChanged, Status: ingest.ConditionStatusTrue, Reason: reason, Message: "Catalog materialization skipped"})
			continue
		}
		if result.File.ID == 0 && result.Item.ID == 0 {
			err := fmt.Errorf("catalog materialization produced no catalog output")
			materializeDirtyReasons[fileID] = "materialization_failed"
			materializeEvents = append(materializeEvents, database.IngestEvent{UnitKey: inventoryFileUnitKey(fileID), LibraryID: config.Library.ID, InventoryFileID: &fileID, ConditionType: ingest.ConditionMaterialized, EventType: ingest.EventConditionChanged, Status: ingest.ConditionStatusFailed, Reason: "materialization_empty", Message: err.Error()})
			materializeErrors = append(materializeErrors, fmt.Errorf("materialize inventory file %d: %w", fileID, err))
			continue
		}
		materializeDirtyReasons[fileID] = "materialization_completed"
		materializeEvents = append(materializeEvents, database.IngestEvent{UnitKey: inventoryFileUnitKey(fileID), LibraryID: config.Library.ID, InventoryFileID: &fileID, ConditionType: ingest.ConditionMaterialized, EventType: ingest.EventConditionChanged, Status: ingest.ConditionStatusTrue, Reason: "materialization_completed", Message: "Catalog materialization completed"})
	}
	if err := s.persistPathTreeWorkGroupAssignments(ctx, &batchState); err != nil {
		return err
	}
	if err := s.flushMaterializeIngestBatch(ctx, materializeDirtyReasons, materializeEvents); err != nil {
		return err
	}
	if len(materializedFileIDs) == 0 && len(materializedItemIDs) == 0 {
		return errors.Join(materializeErrors...)
	}
	rootPath := strings.TrimSpace(payload.RootPath)
	if rootPath == "" {
		rootPath = config.Library.RootPath
	}
	if _, err := s.QueueCatalogPostMaterializeBatch(ctx, config.Library.ID, rootPath, materializedFileIDs, materializedItemIDs); err != nil {
		return err
	}
	return errors.Join(materializeErrors...)
}

func (s *Service) RunCatalogPostMaterializeBatch(ctx context.Context, payload CatalogPostMaterializeBatchPayload) error {
	config, err := s.EffectiveLibraryConfig(ctx, payload.LibraryID)
	if err != nil {
		return err
	}
	rootPath := strings.TrimSpace(payload.RootPath)
	if rootPath == "" {
		rootPath = config.Library.RootPath
	}
	itemIDs := normalizeUintIDs(payload.ItemIDs)
	fileIDs := normalizeUintIDs(payload.FileIDs)
	if len(itemIDs) > 0 {
		if _, err := s.QueueCatalogMatchBatch(ctx, config.Library.ID, rootPath, itemIDs); err != nil {
			return err
		}
	}
	scanPolicy, err := loadScanPolicy(ctx, s.db, config.Library.ID)
	if err != nil {
		return err
	}
	if len(fileIDs) > 0 && scanPolicy.InventoryProbeBatchEnabled {
		if _, err := s.QueueInventoryProbeBatch(ctx, config.Library.ID, rootPath, fileIDs); err != nil {
			return err
		}
	}
	if _, err := s.QueueCatalogLibraryProjectionRefresh(ctx, config.Library.ID, rootPath); err != nil {
		return err
	}
	s.markProjectionLibraryDirty(ctx, config.Library.ID, rootPath, "materialization_completed")
	return nil
}

func (s *Service) RunInventoryProbeBatch(ctx context.Context, payload InventoryProbeBatchPayload) error {
	if s.inventoryProbeExecutor == nil {
		return fmt.Errorf("probe executor unavailable for workflow batch")
	}
	for _, fileID := range normalizeUintIDs(payload.FileIDs) {
		if err := s.inventoryProbeExecutor(ctx, fileID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) RunCatalogMatchBatch(ctx context.Context, payload CatalogMatchBatchPayload) error {
	if s.catalogMatchExecutor == nil {
		return fmt.Errorf("metadata match executor unavailable for workflow batch")
	}
	for _, itemID := range normalizeUintIDs(payload.ItemIDs) {
		if err := s.catalogMatchExecutor(ctx, itemID); err != nil {
			return err
		}
	}
	return nil
}

type materializeBatchState struct {
	scanPolicy                database.LibraryScanPolicy
	subtitlePolicy            database.LibrarySubtitlePolicy
	exclusionRules            []database.ScanExclusionRule
	providersByKey            map[string]materializeProviderBinding
	directorySnapshots        map[string]scanDirectorySnapshot
	decisionSnapshots         map[string]scanDirectorySnapshot
	filesByID                 map[uint]database.InventoryFile
	catalogCache              *catalogScanBatchCache
	tokenProfileCache         *filenameTokenProfileCache
	shapePlansByDir           map[string]contentShapeDirectoryPlan
	shapeAssignmentsByDir     map[string]map[string]contentShapeFileAssignment
	pathTreeAssignmentsByPath map[string]pathTreeWorkGroupAssignment
	shapeCounters             *contentShapeCounters
	indexedSignalsByPath      map[string]filenameSignalModel
}

type materializeProviderBinding struct {
	pathRecord database.LibraryPath
	provider   storage.Provider
}

func (s *Service) newMaterializeBatchState(ctx context.Context, config EffectiveLibraryConfig) (materializeBatchState, error) {
	state := materializeBatchState{
		shapeCounters:             &contentShapeCounters{MaterializationBatches: 1},
		scanPolicy:                config.ScanPolicy,
		subtitlePolicy:            config.SubtitlePolicy,
		providersByKey:            make(map[string]materializeProviderBinding, len(config.Paths)),
		directorySnapshots:        make(map[string]scanDirectorySnapshot),
		decisionSnapshots:         make(map[string]scanDirectorySnapshot),
		filesByID:                 make(map[uint]database.InventoryFile),
		catalogCache:              newCatalogScanBatchCache(),
		tokenProfileCache:         newFilenameTokenProfileCache(),
		shapePlansByDir:           make(map[string]contentShapeDirectoryPlan),
		shapeAssignmentsByDir:     make(map[string]map[string]contentShapeFileAssignment),
		pathTreeAssignmentsByPath: make(map[string]pathTreeWorkGroupAssignment),
		indexedSignalsByPath:      make(map[string]filenameSignalModel),
	}
	state.tokenProfileCache.counters = state.shapeCounters
	if state.scanPolicy.ConfigurableExclusionRules {
		rules, err := s.enabledScanExclusionRules(ctx, config.Library.ID)
		if err != nil {
			return materializeBatchState{}, err
		}
		state.exclusionRules = rules
	}
	paths := append([]database.LibraryPath(nil), config.Paths...)
	sort.Slice(paths, func(i, j int) bool {
		return len(strings.TrimSpace(paths[i].RootPath)) > len(strings.TrimSpace(paths[j].RootPath))
	})
	for _, pathRecord := range paths {
		provider, err := s.providerForLibraryPath(ctx, pathRecord)
		if err != nil {
			return materializeBatchState{}, err
		}
		key := strings.TrimSpace(provider.Name()) + "\x00" + strings.TrimSpace(pathRecord.RootPath)
		state.providersByKey[key] = materializeProviderBinding{pathRecord: pathRecord, provider: provider}
	}
	return state, nil
}

func (s *Service) materializeInventoryFile(ctx context.Context, config EffectiveLibraryConfig, batchState materializeBatchState, fileID uint, directorySnapshots map[string]scanDirectorySnapshot) (catalogScanWriteResult, error) {
	file, ok := batchState.filesByID[fileID]
	if !ok {
		return catalogScanWriteResult{}, fmt.Errorf("inventory file %d not preloaded", fileID)
	}
	if file.Status != inventory.FileStatusAvailable || file.ContentClass != SourceContentClassVideo {
		return catalogScanWriteResult{SkippedReason: "materialization_skipped_not_materializable"}, nil
	}
	pathRecord, provider, err := s.materializeProviderForFile(config, batchState, file)
	if err != nil {
		return catalogScanWriteResult{}, err
	}
	object, snapshot, err := s.materializeSnapshotForFile(ctx, provider, file, directorySnapshots)
	if err != nil {
		return catalogScanWriteResult{}, err
	}
	decisionSnapshot, err := s.materializeDecisionSnapshot(ctx, provider, config.Library, snapshot, &batchState)
	if err != nil {
		return catalogScanWriteResult{}, err
	}
	shapePlan, err := s.contentShapePlanForMaterializeDirectory(ctx, config, pathRecord, provider, decisionSnapshot, &batchState)
	if err != nil {
		return catalogScanWriteResult{}, err
	}
	libraryForPath := config.Library
	libraryForPath.MediaSourceID = pathRecord.MediaSourceID
	libraryForPath.RootPath = pathRecord.RootPath
	writeResult, _, err := s.materializeObjectFromSnapshotWithDirectorySnapshotsAndPlan(ctx, provider, libraryForPath, object, snapshot, decisionSnapshot, directorySnapshots, batchState.tokenProfileCache, shapePlan, batchState.shapeAssignmentsByDir[keyForShapeAssignments(provider, pathRecord.RootPath, decisionSnapshot.Path)], batchState.pathTreeAssignmentsByPath, batchState.subtitlePolicy, batchState.catalogCache)
	if err != nil {
		return catalogScanWriteResult{}, err
	}
	if writeResult.Item.ID != 0 {
		s.markProjectionItemDirty(ctx, writeResult.Item.ID, "materialization_completed")
	}
	return writeResult, nil
}

func (s *Service) preloadMaterializeBatchFiles(ctx context.Context, config EffectiveLibraryConfig, batchState *materializeBatchState, fileIDs []uint) error {
	if batchState == nil || len(fileIDs) == 0 {
		return nil
	}
	var files []database.InventoryFile
	for _, batch := range chunkUints(fileIDs, sqliteVariableChunkSize) {
		var partial []database.InventoryFile
		if err := s.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", batch).Find(&partial).Error; err != nil {
			return err
		}
		files = append(files, partial...)
	}
	for _, file := range files {
		batchState.filesByID[file.ID] = file
	}
	if err := s.hydrateMaterializeFileSignals(ctx, batchState, files); err != nil {
		return err
	}
	rules, err := loadPathTreeClassificationRules(ctx, s.db, config.Library.ID)
	if err != nil {
		return err
	}
	batchState.pathTreeAssignmentsByPath = compilePathTreeAssignmentsFromFiles(files, "", batchState.indexedSignalsByPath, batchState.tokenProfileCache)
	for storagePath, assignment := range applyPathTreeClassificationRules(files, rules, batchState.indexedSignalsByPath, batchState.tokenProfileCache) {
		if batchState.pathTreeAssignmentsByPath == nil {
			batchState.pathTreeAssignmentsByPath = make(map[string]pathTreeWorkGroupAssignment)
		}
		batchState.pathTreeAssignmentsByPath[storagePath] = assignment
	}
	if err := s.persistPathTreeWorkGroupPlans(ctx, batchState, config, files); err != nil {
		return err
	}
	if err := s.persistPathTreeReviewDecisions(ctx, batchState, config, files); err != nil {
		return err
	}
	return nil
}

func loadPathTreeClassificationRules(ctx context.Context, db *gorm.DB, libraryID uint) ([]database.ClassificationRule, error) {
	if db == nil || libraryID == 0 {
		return nil, nil
	}
	var rules []database.ClassificationRule
	if err := db.WithContext(ctx).Where("library_id = ? AND rule_type = ? AND enabled = ?", libraryID, pathTreeWorkGroupRuleType, true).Order("id asc").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func (s *Service) persistPathTreeReviewDecisions(ctx context.Context, batchState *materializeBatchState, config EffectiveLibraryConfig, files []database.InventoryFile) error {
	if s == nil || s.db == nil || batchState == nil {
		return nil
	}
	reviews := compileAmbiguousPathTreeReviewAssignmentsFromFiles(files, batchState.indexedSignalsByPath, batchState.tokenProfileCache)
	if len(reviews) == 0 {
		return nil
	}
	byParent := make(map[string]map[string]pathTreeWorkGroupAssignment)
	for storagePath, assignment := range reviews {
		parentPath := pathTreeAssignmentParentPath(assignment, storagePath)
		if byParent[parentPath] == nil {
			byParent[parentPath] = make(map[string]pathTreeWorkGroupAssignment)
		}
		byParent[parentPath][storagePath] = assignment
	}
	for parentPath, assignments := range byParent {
		if err := savePathTreeReviewDecision(ctx, s.db, config.Library.ID, parentPath, assignments, "path-tree work group candidates require review"); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) persistPathTreeWorkGroupPlans(ctx context.Context, batchState *materializeBatchState, config EffectiveLibraryConfig, files []database.InventoryFile) error {
	if s == nil || s.db == nil || batchState == nil || len(batchState.pathTreeAssignmentsByPath) == 0 {
		return nil
	}
	settings := contentShapeSettingsFromConfig(s.cfg)
	assignmentsByParent := make(map[string]map[string]pathTreeWorkGroupAssignment)
	for storagePath, assignment := range batchState.pathTreeAssignmentsByPath {
		parentPath := pathTreeAssignmentParentPath(assignment, storagePath)
		if parentPath == "" {
			continue
		}
		if assignmentsByParent[parentPath] == nil {
			assignmentsByParent[parentPath] = make(map[string]pathTreeWorkGroupAssignment)
		}
		assignmentsByParent[parentPath][storagePath] = assignment
	}
	if len(assignmentsByParent) == 0 {
		return nil
	}
	providerByFilePath := make(map[string]string, len(files))
	for _, file := range files {
		providerByFilePath[strings.TrimSpace(file.StoragePath)] = strings.TrimSpace(file.StorageProvider)
	}
	for parentPath, assignments := range assignmentsByParent {
		providerName := ""
		for storagePath := range assignments {
			providerName = providerByFilePath[storagePath]
			break
		}
		pathRecord, ok := materializePathRecordForStoragePath(config, parentPath)
		if !ok {
			continue
		}
		scope := contentShapeScope{LibraryID: config.Library.ID, MediaSourceID: pathRecord.MediaSourceID, StorageProvider: providerName, RootPath: pathRecord.RootPath, DirectoryPath: parentPath, ClassifierVersion: settings.ClassifierVersion, Fingerprint: pathTreePersistedFingerprint(parentPath, assignments, settings.ClassifierVersion, batchState.scanPolicy, batchState.exclusionRules)}
		if pathRecord.ID != 0 {
			pathID := pathRecord.ID
			scope.LibraryPathID = &pathID
		}
		profile := contentShapeDatabaseProfile(scope, contentShapeDirectoryProfile{Path: parentPath, VideoCount: len(assignments), NonExtraVideoCount: len(assignments), TitleUniqueCount: len(assignments), TitleUniqueness: 1})
		profile.Fingerprint = scope.Fingerprint
		if err := saveContentShapeProfile(ctx, s.db, &profile); err != nil {
			return err
		}
		profileRecord, _, err := loadReusableContentShapeProfile(ctx, s.db, scope)
		if err != nil {
			return err
		}
		plan := pathTreeContentShapePlanForAssignments(parentPath, assignments)
		planRow := contentShapeDatabasePlan(scope, profileRecord.ID, plan)
		if err := saveContentShapePlan(ctx, s.db, &planRow); err != nil {
			return err
		}
		_, reusedPlan, err := loadReusableContentShapePlan(ctx, s.db, scope)
		if err != nil {
			return err
		}
		if !reusedPlan {
			return fmt.Errorf("reload path-tree work-group plan for %s", parentPath)
		}
	}
	return nil
}

func (s *Service) persistPathTreeWorkGroupAssignments(ctx context.Context, batchState *materializeBatchState) error {
	if s == nil || s.db == nil || batchState == nil || len(batchState.pathTreeAssignmentsByPath) == 0 {
		return nil
	}
	assignmentsByParent := make(map[string]map[string]pathTreeWorkGroupAssignment)
	for storagePath, assignment := range batchState.pathTreeAssignmentsByPath {
		parentPath := pathTreeAssignmentParentPath(assignment, storagePath)
		if parentPath == "" {
			continue
		}
		if assignmentsByParent[parentPath] == nil {
			assignmentsByParent[parentPath] = make(map[string]pathTreeWorkGroupAssignment)
		}
		assignmentsByParent[parentPath][storagePath] = assignment
	}
	settings := contentShapeSettingsFromConfig(s.cfg)
	for parentPath, assignments := range assignmentsByParent {
		var plan database.ContentShapePlan
		if err := s.db.WithContext(ctx).Where("library_id = ? AND directory_path = ? AND classifier_version = ? AND deleted_scope = ? AND invalidated_at IS NULL", batchState.filesByAnyAssignmentLibraryID(assignments), parentPath, settings.ClassifierVersion, false).First(&plan).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}
		scope := contentShapeScopeFromPlan(plan)
		if err := saveContentShapeAssignments(ctx, s.db, scope, plan.ProfileID, plan.ID, contentShapeAssignmentsFromPathTree(assignments)); err != nil {
			return err
		}
	}
	return nil
}

func (state *materializeBatchState) filesByAnyAssignmentLibraryID(assignments map[string]pathTreeWorkGroupAssignment) uint {
	if state == nil {
		return 0
	}
	for storagePath := range assignments {
		for _, file := range state.filesByID {
			if strings.TrimSpace(file.StoragePath) == strings.TrimSpace(storagePath) {
				return file.LibraryID
			}
		}
	}
	return 0
}

func pathTreeAssignmentParentPath(assignment pathTreeWorkGroupAssignment, storagePath string) string {
	if parent, ok := assignment.Evidence["parent_path"].(string); ok && strings.TrimSpace(parent) != "" {
		return strings.TrimSpace(parent)
	}
	return path.Dir(path.Dir(strings.TrimSpace(storagePath)))
}

func materializePathRecordForStoragePath(config EffectiveLibraryConfig, storagePath string) (database.LibraryPath, bool) {
	for _, pathRecord := range config.Paths {
		root := strings.TrimSpace(pathRecord.RootPath)
		if root != "" && (strings.TrimSpace(storagePath) == root || strings.HasPrefix(strings.TrimSpace(storagePath), root+"/")) {
			return pathRecord, true
		}
	}
	return database.LibraryPath{}, false
}

func pathTreePersistedFingerprint(parentPath string, assignments map[string]pathTreeWorkGroupAssignment, classifierVersion string, scanPolicy database.LibraryScanPolicy, exclusionRules []database.ScanExclusionRule) string {
	parts := []string{"parent=" + strings.TrimSpace(parentPath), "classifier=" + strings.TrimSpace(classifierVersion), contentShapeScanPolicyFingerprint(scanPolicy), contentShapeExclusionFingerprint(exclusionRules)}
	keys := make([]string, 0, len(assignments))
	for storagePath := range assignments {
		keys = append(keys, storagePath)
	}
	sort.Strings(keys)
	for _, storagePath := range keys {
		assignment := assignments[storagePath]
		parts = append(parts, strings.Join([]string{storagePath, assignment.AssignmentType, assignment.TargetKey, fmt.Sprintf("%.3f", assignment.Confidence)}, "|"))
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(digest[:])
}

func savePathTreeReviewDecision(ctx context.Context, db *gorm.DB, libraryID uint, parentPath string, assignments map[string]pathTreeWorkGroupAssignment, reason string) error {
	if db == nil || len(assignments) == 0 {
		return nil
	}
	affected := make([]string, 0, len(assignments))
	evidence := make([]map[string]any, 0, len(assignments))
	confidence := 1.0
	for storagePath, assignment := range assignments {
		affected = append(affected, storagePath)
		if assignment.Confidence > 0 && assignment.Confidence < confidence {
			confidence = assignment.Confidence
		}
		evidence = append(evidence, map[string]any{"storage_path": storagePath, "assignment_type": assignment.AssignmentType, "target_key": assignment.TargetKey, "evidence": assignment.Evidence})
	}
	if confidence == 1.0 {
		confidence = 0.5
	}
	sort.Strings(affected)
	return db.WithContext(ctx).Create(&database.ClassificationDecision{LibraryID: libraryID, SourcePath: strings.TrimSpace(parentPath), DecisionType: "path_tree_work_group", CandidateType: pathTreeWorkGroupShapeReview, TargetKind: "work_group", TargetKey: strings.TrimSpace(parentPath), Status: scanDecisionStatusReviewRequired, Confidence: &confidence, EvidenceJSON: mustJSON(evidence), AffectedFilesJSON: mustJSON(affected), AlternativesJSON: mustJSON([]string{pathTreeWorkGroupShapeMovieVersionGroup, pathTreeWorkGroupShapeMovieCollection, pathTreeWorkGroupShapeSeries}), Reason: strings.TrimSpace(reason)}).Error
}

func (s *Service) hydrateMaterializeFileSignals(ctx context.Context, batchState *materializeBatchState, files []database.InventoryFile) error {
	if batchState == nil || len(files) == 0 {
		return nil
	}
	settings := contentShapeSettingsFromConfig(s.cfg)
	filesByProvider := make(map[string][]database.InventoryFile)
	for _, file := range files {
		if file.ID == 0 || file.Status != inventory.FileStatusAvailable || file.ContentClass != SourceContentClassVideo || !isVideoFile(file.StoragePath) {
			continue
		}
		provider := strings.TrimSpace(file.StorageProvider)
		if provider == "" || strings.TrimSpace(file.StoragePath) == "" {
			continue
		}
		filesByProvider[provider] = append(filesByProvider[provider], file)
	}
	for provider, providerFiles := range filesByProvider {
		scope := inventoryFileSignalScope{LibraryID: providerFiles[0].LibraryID, StorageProvider: provider, ClassifierVersion: settings.ClassifierVersion}
		models, _, err := loadReusableInventoryFileSignals(ctx, s.db, scope, providerFiles)
		if err != nil {
			return err
		}
		hydrateFilenameTokenCacheFromSignals(batchState.tokenProfileCache, models)
		for storagePath, model := range models {
			batchState.indexedSignalsByPath[storagePath] = model
		}
		missing := make([]inventoryFileSignalInput, 0)
		for _, file := range providerFiles {
			storagePath := strings.TrimSpace(file.StoragePath)
			if _, ok := models[storagePath]; ok {
				continue
			}
			model := filenameTokenProfileForPath(batchState.tokenProfileCache, storagePath)
			missing = append(missing, inventoryFileSignalInput{File: file, Model: model})
			batchState.indexedSignalsByPath[storagePath] = model
		}
		if err := saveInventoryFileSignals(ctx, s.db, scope, missing); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) preloadMaterializeBatchCatalogData(ctx context.Context, config EffectiveLibraryConfig, batchState *materializeBatchState, fileIDs []uint, directorySnapshots map[string]scanDirectorySnapshot) error {
	if batchState == nil || batchState.catalogCache == nil {
		return nil
	}
	if err := s.preloadMaterializeBatchAssets(ctx, batchState, fileIDs); err != nil {
		return err
	}
	plannedItems := make(map[string]catalog.CreateItemInput)
	paths := make([]string, 0, len(fileIDs)*3)
	seenPaths := make(map[string]struct{}, len(fileIDs)*3)
	for _, fileID := range fileIDs {
		file, ok := batchState.filesByID[fileID]
		if !ok || file.Status != inventory.FileStatusAvailable || file.ContentClass != SourceContentClassVideo {
			continue
		}
		pathRecord, provider, err := s.materializeProviderForFile(config, *batchState, file)
		if err != nil {
			return err
		}
		object, snapshot, err := s.materializeSnapshotForFile(ctx, provider, file, directorySnapshots)
		if err != nil {
			return err
		}
		decisionSnapshot, err := s.materializeDecisionSnapshot(ctx, provider, config.Library, snapshot, batchState)
		if err != nil {
			return err
		}
		shapePlan, err := s.contentShapePlanForMaterializeDirectory(ctx, config, pathRecord, provider, decisionSnapshot, batchState)
		if err != nil {
			return err
		}
		libraryForPath := config.Library
		libraryForPath.MediaSourceID = pathRecord.MediaSourceID
		libraryForPath.RootPath = pathRecord.RootPath
		artifact, ok := materializePlannedCatalogArtifact(provider, libraryForPath, object, batchState.tokenProfileCache, shapePlan, batchState.shapeAssignmentsByDir[keyForShapeAssignments(provider, pathRecord.RootPath, decisionSnapshot.Path)], batchState.pathTreeAssignmentsByPath)
		if !ok {
			continue
		}
		for _, input := range plannedCatalogCreateInputsForArtifact(libraryForPath, artifact) {
			key := plannedCatalogItemKey(input)
			if key == "" {
				continue
			}
			plannedItems[key] = input
		}
		for _, itemPath := range catalogScanItemPaths(artifact) {
			trimmed := strings.TrimSpace(itemPath)
			if trimmed == "" {
				continue
			}
			if _, exists := seenPaths[trimmed]; exists {
				continue
			}
			seenPaths[trimmed] = struct{}{}
			paths = append(paths, trimmed)
		}
	}
	if len(paths) == 0 {
		return nil
	}
	var items []database.CatalogItem
	for _, batch := range chunkStrings(paths, sqliteVariableChunkSize) {
		var partial []database.CatalogItem
		if err := s.db.WithContext(ctx).
			Where("library_id = ? AND deleted_at IS NULL AND path IN ?", config.Library.ID, batch).
			Find(&partial).Error; err != nil {
			return err
		}
		items = append(items, partial...)
	}
	for _, item := range items {
		batchState.catalogCache.rememberItem(item)
		delete(plannedItems, plannedCatalogItemKey(catalog.CreateItemInput{LibraryID: item.LibraryID, Path: item.Path}))
	}
	for _, input := range plannedItems {
		batchState.catalogCache.rememberPlannedItem(input)
	}
	return nil
}

func (s *Service) preloadMaterializeBatchAssets(ctx context.Context, batchState *materializeBatchState, fileIDs []uint) error {
	if batchState == nil || batchState.catalogCache == nil || len(fileIDs) == 0 {
		return nil
	}
	var rows []struct {
		FileID uint
		database.MediaAsset
	}
	for _, batch := range chunkUints(fileIDs, sqliteVariableChunkSize) {
		var partial []struct {
			FileID uint
			database.MediaAsset
		}
		if err := s.db.WithContext(ctx).
			Table("media_assets").
			Select("asset_files.file_id, media_assets.*").
			Joins("JOIN asset_files ON asset_files.asset_id = media_assets.id").
			Where("asset_files.file_id IN ? AND asset_files.role = ? AND asset_files.part_index = 0 AND media_assets.deleted_at IS NULL", batch, inventory.FileRoleSource).
			Scan(&partial).Error; err != nil {
			return err
		}
		rows = append(rows, partial...)
	}
	for _, row := range rows {
		batchState.catalogCache.rememberAsset(row.FileID, row.MediaAsset)
	}
	for _, fileID := range fileIDs {
		if _, ok := batchState.catalogCache.asset(fileID); ok {
			continue
		}
		file, exists := batchState.filesByID[fileID]
		if !exists || file.LibraryID == 0 {
			continue
		}
		batchState.catalogCache.rememberPlannedAsset(fileID, inventory.CreateAssetInput{LibraryID: file.LibraryID, AssetType: inventory.AssetTypeMain, DisplayName: path.Base(strings.TrimSpace(file.StoragePath)), Status: inventory.AssetStatusAvailable})
	}
	return nil
}

func materializePlannedCatalogArtifact(provider storage.Provider, library database.Library, object storage.Object, tokenCache *filenameTokenProfileCache, shapePlan contentShapeDirectoryPlan, shapeAssignments map[string]contentShapeFileAssignment, pathTreeAssignments map[string]pathTreeWorkGroupAssignment) (catalogScanArtifact, bool) {
	classificationLibraryType := effectiveVideoLibraryType(library.Type)
	assignment, hasAssignment := shapeAssignments[strings.TrimSpace(object.Path)]
	if hasAssignment && assignment.AssignmentType == contentShapeAssignmentSkip {
		return catalogScanArtifact{}, false
	}
	pathTreeAssignment := pathTreeAssignments[strings.TrimSpace(object.Path)]
	classified, planned := plannedClassifiedMediaForMaterialize(object, tokenCache, shapePlan, assignment, hasAssignment, pathTreeAssignment)
	if !planned {
		return catalogScanArtifact{}, false
	}
	artifact, _ := catalogScanArtifactFromObject(provider.Name(), classificationLibraryType, library.RootPath, object, classified)
	applyPathTreeWorkGroupAssignment(&artifact, pathTreeAssignments[strings.TrimSpace(object.Path)])
	return artifact, true
}

func plannedClassifiedMediaForMaterialize(object storage.Object, tokenCache *filenameTokenProfileCache, shapePlan contentShapeDirectoryPlan, assignment contentShapeFileAssignment, hasAssignment bool, pathTreeAssignment pathTreeWorkGroupAssignment) (classifiedMedia, bool) {
	if strings.TrimSpace(pathTreeAssignment.StoragePath) != "" {
		return classifiedMedia{Type: "movie", Title: cleanTitle(path.Base(strings.TrimSpace(pathTreeAssignment.TargetKey))), SourcePath: object.Path, Status: "ready", FilenameSignals: filenameTokenProfileForPath(tokenCache, object.Path)}, true
	}
	if hasAssignment {
		return classifiedMediaFromContentShapeAssignment(shapePlan, assignment, object, tokenCache)
	}
	if classified, planned := classifiedMediaFromContentShapePlan(shapePlan, object, tokenCache); planned {
		return classified, true
	}
	return classifiedMedia{}, false
}

func (s *Service) contentShapePlanForMaterializeDirectory(ctx context.Context, config EffectiveLibraryConfig, pathRecord database.LibraryPath, provider storage.Provider, snapshot scanDirectorySnapshot, batchState *materializeBatchState) (contentShapeDirectoryPlan, error) {
	if batchState == nil {
		return contentShapeDirectoryPlan{}, nil
	}
	key := keyForShapeAssignments(provider, pathRecord.RootPath, snapshot.Path)
	if plan, ok := batchState.shapePlansByDir[key]; ok {
		if batchState.shapeCounters != nil {
			batchState.shapeCounters.PlanReuses++
		}
		return plan, nil
	}
	settings := contentShapeSettingsFromConfig(s.cfg)
	if s.db == nil {
		if batchState.shapeCounters != nil {
			batchState.shapeCounters.DirectoryProfileBuilds++
			batchState.shapeCounters.PlanCompiles++
		}
		profile := buildContentShapeDirectoryProfile(effectiveVideoLibraryType(config.Library.Type), pathRecord.RootPath, snapshot, batchState.tokenProfileCache)
		plan := compileContentShapePlan(profile)
		batchState.shapePlansByDir[key] = plan
		return plan, nil
	}
	scope := contentShapeScope{LibraryID: config.Library.ID, MediaSourceID: pathRecord.MediaSourceID, StorageProvider: strings.TrimSpace(provider.Name()), RootPath: strings.TrimSpace(pathRecord.RootPath), DirectoryPath: strings.TrimSpace(snapshot.Path), ClassifierVersion: settings.ClassifierVersion}
	if pathRecord.ID != 0 {
		pathID := pathRecord.ID
		scope.LibraryPathID = &pathID
	}
	profileRecord, builtProfile, profileReused, err := loadOrBuildContentShapeProfileWithBuilt(ctx, s.db, scope, snapshot, batchState.scanPolicy, batchState.exclusionRules, batchState.tokenProfileCache)
	if err != nil {
		return contentShapeDirectoryPlan{}, err
	}
	if !profileReused && batchState.shapeCounters != nil {
		batchState.shapeCounters.DirectoryProfileBuilds++
	}
	scope = contentShapeScopeFromProfile(profileRecord)
	planRecord, reusedPlan, err := loadReusableContentShapePlan(ctx, s.db, scope)
	if err != nil {
		return contentShapeDirectoryPlan{}, err
	}
	if reusedPlan {
		if batchState.shapeCounters != nil {
			batchState.shapeCounters.PlanReuses++
		}
		plan := contentShapePlanFromRecord(planRecord)
		visiblePaths := contentShapeVisibleVideoPaths(snapshot)
		rows, err := loadReusableContentShapeAssignments(ctx, s.db, planRecord.ID, visiblePaths)
		if err != nil {
			return contentShapeDirectoryPlan{}, err
		}
		if len(rows) == len(visiblePaths) {
			batchState.shapeAssignmentsByDir[key] = contentShapeAssignmentsFromRecords(rows)
		} else {
			assignments := generateContentShapeAssignments(plan, snapshot, batchState.tokenProfileCache)
			if err := saveContentShapeAssignments(ctx, s.db, scope, profileRecord.ID, planRecord.ID, assignments); err != nil {
				return contentShapeDirectoryPlan{}, err
			}
			batchState.shapeAssignmentsByDir[key] = contentShapeAssignmentsByPath(assignments)
		}
		batchState.shapePlansByDir[key] = plan
		return plan, nil
	}
	if batchState.shapeCounters != nil {
		batchState.shapeCounters.PlanCompiles++
	}
	plan := compileContentShapePlan(builtProfile)
	planRow := contentShapeDatabasePlan(scope, profileRecord.ID, plan)
	if err := saveContentShapePlan(ctx, s.db, &planRow); err != nil {
		return contentShapeDirectoryPlan{}, err
	}
	planRecord, reusedPlan, err = loadReusableContentShapePlan(ctx, s.db, scope)
	if err != nil {
		return contentShapeDirectoryPlan{}, err
	}
	if !reusedPlan {
		return contentShapeDirectoryPlan{}, fmt.Errorf("reload content shape plan for %s", scope.DirectoryPath)
	}
	assignments := generateContentShapeAssignments(plan, snapshot, batchState.tokenProfileCache)
	if err := saveContentShapeAssignments(ctx, s.db, scope, profileRecord.ID, planRecord.ID, assignments); err != nil {
		return contentShapeDirectoryPlan{}, err
	}
	batchState.shapeAssignmentsByDir[key] = contentShapeAssignmentsByPath(assignments)
	if err := saveContentShapeReviewDecision(ctx, s.db, scope, plan, assignments); err != nil {
		return contentShapeDirectoryPlan{}, err
	}
	batchState.shapePlansByDir[key] = plan
	return plan, nil
}

func keyForShapeAssignments(provider storage.Provider, rootPath string, directoryPath string) string {
	return strings.TrimSpace(provider.Name()) + "\x00" + strings.TrimSpace(rootPath) + "\x00" + strings.TrimSpace(directoryPath)
}

func keyForDecisionSnapshot(provider storage.Provider, rootPath string, directoryPath string) string {
	return strings.TrimSpace(provider.Name()) + "\x00" + strings.TrimSpace(rootPath) + "\x00" + strings.TrimSpace(directoryPath)
}

func (s *Service) materializeDecisionSnapshot(ctx context.Context, provider storage.Provider, library database.Library, snapshot scanDirectorySnapshot, batchState *materializeBatchState) (scanDirectorySnapshot, error) {
	if batchState == nil {
		return s.filteredScanSnapshot(ctx, provider, library, snapshot, nil, database.LibraryScanPolicy{})
	}
	key := keyForDecisionSnapshot(provider, library.RootPath, snapshot.Path)
	if cached, ok := batchState.decisionSnapshots[key]; ok {
		return cached, nil
	}
	decisionSnapshot, err := s.filteredScanSnapshot(ctx, provider, library, snapshot, batchState.exclusionRules, batchState.scanPolicy)
	if err != nil {
		return scanDirectorySnapshot{}, err
	}
	batchState.decisionSnapshots[key] = decisionSnapshot
	return decisionSnapshot, nil
}

func (s *Service) materializeBatchCreateMissingCatalogData(ctx context.Context, config EffectiveLibraryConfig, batchState *materializeBatchState) error {
	if batchState == nil || batchState.catalogCache == nil {
		return nil
	}
	if err := s.materializeBatchCreateMissingAssets(ctx, batchState); err != nil {
		return err
	}
	return s.materializeBatchCreateMissingItems(ctx, config, batchState)
}

func (s *Service) materializeBatchCreateMissingAssets(ctx context.Context, batchState *materializeBatchState) error {
	if batchState == nil || batchState.catalogCache == nil || len(batchState.catalogCache.plannedAssets) == 0 {
		return nil
	}
	fileIDs := make([]uint, 0, len(batchState.catalogCache.plannedAssets))
	assets := make([]database.MediaAsset, 0, len(batchState.catalogCache.plannedAssets))
	for fileID, input := range batchState.catalogCache.plannedAssets {
		fileIDs = append(fileIDs, fileID)
		assets = append(assets, database.MediaAsset{
			LibraryID:   input.LibraryID,
			AssetType:   strings.TrimSpace(input.AssetType),
			DisplayName: strings.TrimSpace(input.DisplayName),
			Status:      inventory.AssetStatusAvailable,
			ProbeStatus: "pending",
		})
	}
	createdAssetIDs := make([]uint, 0, len(assets))
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.CreateInBatches(&assets, sqliteMediumWriteBatchSize).Error; err != nil {
			return err
		}
		links := make([]inventory.LinkAssetFileInput, 0, len(assets))
		for idx, asset := range assets {
			createdAssetIDs = append(createdAssetIDs, asset.ID)
			links = append(links, inventory.LinkAssetFileInput{AssetID: asset.ID, FileID: fileIDs[idx], Role: inventory.FileRoleSource, PartIndex: 0})
		}
		return inventory.NewService(tx).BulkLinkAssetToFiles(ctx, links)
	}); err != nil {
		return err
	}
	for idx, asset := range assets {
		batchState.catalogCache.rememberAsset(fileIDs[idx], asset)
	}
	batchState.catalogCache.plannedAssets = map[uint]inventory.CreateAssetInput{}
	_ = createdAssetIDs
	return nil
}

func (s *Service) materializeBatchCreateMissingItems(ctx context.Context, config EffectiveLibraryConfig, batchState *materializeBatchState) error {
	if batchState == nil || batchState.catalogCache == nil || len(batchState.catalogCache.plannedItems) == 0 {
		return nil
	}
	planned := make([]catalog.CreateItemInput, 0, len(batchState.catalogCache.plannedItems))
	for _, input := range batchState.catalogCache.plannedItems {
		planned = append(planned, input)
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		catalogSvc := catalog.NewService(tx, s.ingest)
		stages := []string{catalog.ItemTypeSeries, catalog.ItemTypeSeason, catalog.ItemTypeMovie, catalog.ItemTypeEpisode}
		for _, itemType := range stages {
			stageInputs := make([]catalog.CreateItemInput, 0)
			for _, input := range planned {
				if strings.TrimSpace(input.Type) != itemType {
					continue
				}
				if _, ok := batchState.catalogCache.item(input.LibraryID, input.Path); ok {
					continue
				}
				resolved := input
				if parentPath := plannedCatalogParentPath(input); parentPath != "" {
					parent, ok := batchState.catalogCache.item(input.LibraryID, parentPath)
					if !ok || parent.ID == 0 {
						continue
					}
					resolved.ParentID = &parent.ID
				}
				stageInputs = append(stageInputs, resolved)
			}
			if len(stageInputs) == 0 {
				continue
			}
			createdItems, err := createCatalogItemsWithoutProjection(ctx, tx, catalogSvc, stageInputs)
			if err != nil {
				return err
			}
			for _, item := range createdItems {
				batchState.catalogCache.rememberItem(item)
			}
		}
		return nil
	})
}

func plannedCatalogParentPath(input catalog.CreateItemInput) string {
	pathValue := strings.TrimSpace(input.Path)
	if pathValue == "" {
		return ""
	}
	switch strings.TrimSpace(input.Type) {
	case catalog.ItemTypeSeason:
		return path.Dir(pathValue)
	case catalog.ItemTypeEpisode:
		return path.Dir(pathValue)
	default:
		return ""
	}
}

func createCatalogItemsWithoutProjection(ctx context.Context, tx *gorm.DB, catalogSvc *catalog.Service, inputs []catalog.CreateItemInput) ([]database.CatalogItem, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	items := make([]database.CatalogItem, 0, len(inputs))
	for _, input := range inputs {
		created, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{
			LibraryID:          input.LibraryID,
			Type:               input.Type,
			ParentID:           input.ParentID,
			Path:               input.Path,
			SortKey:            input.SortKey,
			DisplayOrder:       input.DisplayOrder,
			IndexNumber:        input.IndexNumber,
			IndexNumberEnd:     input.IndexNumberEnd,
			ParentIndexNumber:  input.ParentIndexNumber,
			AbsoluteNumber:     input.AbsoluteNumber,
			Title:              input.Title,
			OriginalTitle:      input.OriginalTitle,
			SortTitle:          input.SortTitle,
			Overview:           input.Overview,
			ReleaseDate:        input.ReleaseDate,
			FirstAirDate:       input.FirstAirDate,
			LastAirDate:        input.LastAirDate,
			Year:               input.Year,
			EndYear:            input.EndYear,
			RuntimeSeconds:     input.RuntimeSeconds,
			CommunityRating:    input.CommunityRating,
			OfficialRating:     input.OfficialRating,
			SeriesStatus:       input.SeriesStatus,
			AvailabilityStatus: input.AvailabilityStatus,
			GovernanceStatus:   input.GovernanceStatus,
			ProjectionMode:     catalog.ProjectionModeDeferred,
		})
		if err != nil {
			return nil, err
		}
		if identityKey, ok := catalog.ScannerIdentityKeyForItem(created); ok {
			if _, err := catalogSvc.SetIdentity(ctx, catalog.IdentityInput{ItemID: created.ID, Provider: catalog.IdentityProviderScanner, IdentityType: created.Type, IdentityKey: identityKey, SourcePath: created.Path}); err != nil {
				return nil, err
			}
		}
		items = append(items, created)
	}
	return items, nil
}

func chunkUints(values []uint, size int) [][]uint {
	if len(values) == 0 {
		return nil
	}
	if size <= 0 {
		size = len(values)
	}
	chunks := make([][]uint, 0, (len(values)+size-1)/size)
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[start:end])
	}
	return chunks
}

func chunkStrings(values []string, size int) [][]string {
	if len(values) == 0 {
		return nil
	}
	if size <= 0 {
		size = len(values)
	}
	chunks := make([][]string, 0, (len(values)+size-1)/size)
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[start:end])
	}
	return chunks
}

func chunkCatalogItems(values []database.CatalogItem, size int) [][]database.CatalogItem {
	if len(values) == 0 {
		return nil
	}
	if size <= 0 {
		size = len(values)
	}
	chunks := make([][]database.CatalogItem, 0, (len(values)+size-1)/size)
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[start:end])
	}
	return chunks
}

func plannedCatalogCreateInputsForArtifact(library database.Library, artifact catalogScanArtifact) []catalog.CreateItemInput {
	if strings.TrimSpace(artifact.ItemType) == catalog.ItemTypeEpisode || len(artifact.EpisodeSlots) > 0 {
		if strings.TrimSpace(artifact.SeriesPath) == "" || artifact.SeasonNumber == nil {
			return nil
		}
		inputs := []catalog.CreateItemInput{{
			LibraryID:          library.ID,
			Type:               catalog.ItemTypeSeries,
			Path:               artifact.SeriesPath,
			SortKey:            defaultCatalogSortKey(artifact.SeriesTitle, artifact.SeriesPath),
			Title:              defaultCatalogTitle(artifact.SeriesTitle, artifact.SeriesPath),
			AvailabilityStatus: catalog.AvailabilityAvailable,
			GovernanceStatus:   governanceStatusForScanArtifact(artifact),
		}, {
			LibraryID:          library.ID,
			Type:               catalog.ItemTypeSeason,
			Path:               artifact.SeasonPath,
			SortKey:            fmt.Sprintf("%s S%02d", defaultCatalogTitle(artifact.SeriesTitle, artifact.SeriesPath), *artifact.SeasonNumber),
			Title:              fmt.Sprintf("Season %d", *artifact.SeasonNumber),
			IndexNumber:        artifact.SeasonNumber,
			AvailabilityStatus: catalog.AvailabilityAvailable,
			GovernanceStatus:   governanceStatusForScanArtifact(artifact),
		}}
		for _, slot := range artifact.EpisodeSlots {
			episodeNumber := slot.EpisodeNumber
			episodeTitle := artifact.Title
			if strings.TrimSpace(episodeTitle) == "" {
				episodeTitle = fmt.Sprintf("%s S%02dE%02d", defaultCatalogTitle(artifact.SeriesTitle, artifact.SeriesPath), *artifact.SeasonNumber, episodeNumber)
			}
			inputs = append(inputs, catalog.CreateItemInput{
				LibraryID:          library.ID,
				Type:               catalog.ItemTypeEpisode,
				Path:               slot.ItemPath,
				SortKey:            fmt.Sprintf("%s S%02dE%02d", defaultCatalogTitle(artifact.SeriesTitle, artifact.SeriesPath), *artifact.SeasonNumber, episodeNumber),
				Title:              episodeTitle,
				OriginalTitle:      strings.TrimSpace(artifact.OriginalTitle),
				Year:               artifact.Year,
				IndexNumber:        &episodeNumber,
				ParentIndexNumber:  artifact.SeasonNumber,
				AvailabilityStatus: catalog.AvailabilityAvailable,
				GovernanceStatus:   governanceStatusForScanArtifact(artifact),
			})
		}
		return inputs
	}
	return []catalog.CreateItemInput{{
		LibraryID:          library.ID,
		Type:               artifact.ItemType,
		Path:               artifact.ItemPath,
		SortKey:            defaultCatalogSortKey(artifact.Title, artifact.ItemPath),
		Title:              defaultCatalogTitle(artifact.Title, artifact.SourcePath),
		OriginalTitle:      strings.TrimSpace(artifact.OriginalTitle),
		Year:               artifact.Year,
		AvailabilityStatus: catalog.AvailabilityAvailable,
		GovernanceStatus:   governanceStatusForScanArtifact(artifact),
	}}
}

func plannedCatalogItemKey(input catalog.CreateItemInput) string {
	if input.LibraryID == 0 || strings.TrimSpace(input.Path) == "" {
		return ""
	}
	return fmt.Sprintf("%d\x00%s", input.LibraryID, strings.TrimSpace(input.Path))
}

func (s *Service) flushMaterializeIngestBatch(ctx context.Context, dirtyReasons map[uint]string, events []database.IngestEvent) error {
	if s.ingest == nil {
		return nil
	}
	grouped := make(map[string][]uint)
	for fileID, reason := range dirtyReasons {
		grouped[strings.TrimSpace(reason)] = append(grouped[strings.TrimSpace(reason)], fileID)
	}
	for reason, fileIDs := range grouped {
		if err := s.ingest.MarkInventoryFilesDirty(ctx, fileIDs, reason); err != nil {
			return err
		}
	}
	if err := s.ingest.AppendEvents(ctx, events); err != nil {
		return err
	}
	return nil
}

func (s *Service) materializeProviderForFile(config EffectiveLibraryConfig, batchState materializeBatchState, file database.InventoryFile) (database.LibraryPath, storage.Provider, error) {
	paths := append([]database.LibraryPath(nil), config.Paths...)
	sort.Slice(paths, func(i, j int) bool {
		return len(strings.TrimSpace(paths[i].RootPath)) > len(strings.TrimSpace(paths[j].RootPath))
	})
	for _, pathRecord := range paths {
		key := strings.TrimSpace(file.StorageProvider) + "\x00" + strings.TrimSpace(pathRecord.RootPath)
		binding, ok := batchState.providersByKey[key]
		if !ok {
			continue
		}
		if !isPathWithinRoot(file.StorageProvider, binding.pathRecord.RootPath, file.StoragePath) {
			continue
		}
		return binding.pathRecord, binding.provider, nil
	}
	return database.LibraryPath{}, nil, fmt.Errorf("no library path provider for inventory file %d", file.ID)
}

func (s *Service) materializeSnapshotForFile(ctx context.Context, provider storage.Provider, file database.InventoryFile, directorySnapshots map[string]scanDirectorySnapshot) (storage.Object, scanDirectorySnapshot, error) {
	dirPath := path.Dir(file.StoragePath)
	snapshot, ok := directorySnapshots[dirPath]
	if !ok {
		var err error
		snapshot, err = s.collectDirectorySnapshot(ctx, provider, dirPath, false)
		if err != nil {
			return storage.Object{}, scanDirectorySnapshot{}, err
		}
		if directorySnapshots != nil {
			directorySnapshots[dirPath] = snapshot
		}
	}
	for _, object := range snapshot.Objects {
		if object.Path == file.StoragePath && !object.IsDir {
			return object, snapshot, nil
		}
	}
	object := storage.Object{Path: file.StoragePath, Name: path.Base(file.StoragePath), Size: file.SizeBytes, Modified: file.ModifiedAt, StableIdentity: file.StableIdentityKey, Provider: file.StorageProvider}
	return object, snapshot, nil
}

func (s *Service) materializeObjectFromSnapshotWithDirectorySnapshotsAndPlan(ctx context.Context, provider storage.Provider, library database.Library, object storage.Object, snapshot scanDirectorySnapshot, decisionSnapshot scanDirectorySnapshot, directorySnapshots map[string]scanDirectorySnapshot, tokenCache *filenameTokenProfileCache, shapePlan contentShapeDirectoryPlan, shapeAssignments map[string]contentShapeFileAssignment, pathTreeAssignments map[string]pathTreeWorkGroupAssignment, subtitlePolicy database.LibrarySubtitlePolicy, batchCache *catalogScanBatchCache) (catalogScanWriteResult, []string, error) {
	classificationLibraryType := effectiveVideoLibraryType(library.Type)
	assignment, hasAssignment := shapeAssignments[strings.TrimSpace(object.Path)]
	pathTreeAssignment := pathTreeAssignments[strings.TrimSpace(object.Path)]
	classified, planned := plannedClassifiedMediaForMaterialize(object, tokenCache, shapePlan, assignment, hasAssignment, pathTreeAssignment)
	if !planned {
		return catalogScanWriteResult{SkippedReason: "materialization_skipped_unplanned"}, nil, nil
	}
	artifact, _ := catalogScanArtifactFromObject(provider.Name(), classificationLibraryType, library.RootPath, object, classified)
	applyPathTreeWorkGroupAssignment(&artifact, pathTreeAssignments[strings.TrimSpace(object.Path)])
	if planned {
		if !hasAssignment {
			assignment = contentShapeAssignmentForObject(shapePlan, object, tokenCache)
		}
		applyContentShapePlanEvidence(&artifact, shapePlan, assignment)
	}
	if decision := scanDecisionFromAttachmentRole(object.Path, artifact); strings.TrimSpace(decision.Type) != "" {
		artifact.Decisions = append(artifact.Decisions, decision)
	}
	artifact = s.applyCatalogScanSidecars(ctx, provider, artifact, snapshot.Sidecars.matchesForVideoWithFolderMetadata(object.Path, artifactAllowsFolderMetadata(snapshot.Path, artifact)), subtitlePolicy)
	artifact = applyCatalogScanArtworkCandidates(provider, artifact, object, snapshot)
	sidecarPaths := make([]string, 0, len(artifact.SubtitleSidecars))
	for _, sidecar := range artifact.SubtitleSidecars {
		if strings.TrimSpace(sidecar.Path) != "" {
			sidecarPaths = append(sidecarPaths, sidecar.Path)
		}
	}
	writeResult, err := s.writeCatalogScanWithCache(ctx, library, artifact, batchCache)
	if err != nil {
		return catalogScanWriteResult{}, nil, err
	}
	return writeResult, sidecarPaths, nil
}

func applyContentShapePlanEvidence(artifact *catalogScanArtifact, shapePlan contentShapeDirectoryPlan, assignment contentShapeFileAssignment) {
	if artifact == nil {
		return
	}
	artifact.ContentShapePlan = contentShapePlanDebugPayload(shapePlan)
	artifact.ContentShapeAssignment = map[string]any{"assignment_type": assignment.AssignmentType, "target_key": assignment.TargetKey, "review_state": assignment.ReviewState, "confidence": assignment.Confidence}
	if shapePlan.Confidence < contentShapeHighConfidenceThreshold || shapePlan.ReviewState == "review_required" {
		artifact.Decisions = append(artifact.Decisions, scanDecision{Type: "content_shape_guarded_placeholder", CandidateType: artifact.ItemType, Status: scanDecisionStatusReviewRequired, Confidence: &shapePlan.Confidence, Reason: "content shape uncertain; materialized guarded local placeholder", Evidence: []scanDecisionEvidence{{Kind: "directory_plan", Source: "content_shape", Value: shapePlan.Shape}, {Kind: "placeholder", Source: "content_shape", Value: artifact.ItemType}}})
		artifact.ContentShapeAssignment["placeholder"] = artifact.ItemType
		artifact.ContentShapeAssignment["placeholder_reason"] = "uncertain_content_shape"
		return
	}
	artifact.Decisions = append(artifact.Decisions, scanDecision{Type: "accepted", CandidateType: artifact.ItemType, Status: "accepted", Confidence: &shapePlan.Confidence, Reason: "content shape plan assignment", Evidence: []scanDecisionEvidence{{Kind: "directory_plan", Source: "content_shape", Value: shapePlan.Shape}}})
}

func applyPathTreeWorkGroupAssignment(artifact *catalogScanArtifact, assignment pathTreeWorkGroupAssignment) {
	if artifact == nil || strings.TrimSpace(assignment.StoragePath) == "" || strings.TrimSpace(assignment.TargetKey) == "" {
		return
	}
	if assignment.AssignmentType == pathTreeAssignmentVersion {
		artifact.ItemType = catalog.ItemTypeMovie
		artifact.ItemPath = strings.TrimSpace(assignment.TargetKey)
		artifact.PreferredAssetType = inventory.AssetTypeVersion
		artifact.PreferredAssetRole = inventory.AssetItemRoleVersion
		applyPathTreeRuleTitle(artifact, assignment)
	} else if assignment.AssignmentType == pathTreeAssignmentMovie {
		artifact.ItemType = catalog.ItemTypeMovie
		artifact.ItemPath = strings.TrimSpace(assignment.TargetKey)
		applyPathTreeRuleTitle(artifact, assignment)
	} else if assignment.AssignmentType == pathTreeAssignmentEpisode && assignment.SeasonNumber != nil && assignment.EpisodeNumber != nil && strings.TrimSpace(assignment.SeriesTitle) != "" {
		artifact.ItemType = catalog.ItemTypeEpisode
		artifact.SeriesTitle = strings.TrimSpace(assignment.SeriesTitle)
		artifact.SeriesPath = canonicalSeriesPath(assignment.SeriesTitle)
		artifact.SeasonNumber = assignment.SeasonNumber
		artifact.SeasonPath = fmt.Sprintf("%s/season-%02d", artifact.SeriesPath, *assignment.SeasonNumber)
		artifact.EpisodeSlots = []catalogEpisodeSlot{{EpisodeNumber: *assignment.EpisodeNumber, ItemPath: canonicalEpisodeItemPath(artifact.SeasonPath, *assignment.EpisodeNumber)}}
	}
	artifact.ContentShapeAssignment = mergeStringAnyMaps(artifact.ContentShapeAssignment, map[string]any{"path_tree_assignment_type": assignment.AssignmentType, "path_tree_target_key": assignment.TargetKey, "path_tree_review_state": assignment.ReviewState, "path_tree_confidence": assignment.Confidence})
	confidence := assignment.Confidence
	artifact.Decisions = append(artifact.Decisions, scanDecision{Type: "path_tree_work_group", CandidateType: artifact.ItemType, TargetKind: "work_group", TargetKey: assignment.TargetKey, Status: "accepted", Confidence: &confidence, Reason: "path-tree work group assignment", Evidence: []scanDecisionEvidence{{Kind: "work_group", Source: "path_tree", Value: assignment.AssignmentType}, {Kind: "target_key", Source: "path_tree", Value: assignment.TargetKey}}})
}

func applyPathTreeRuleTitle(artifact *catalogScanArtifact, assignment pathTreeWorkGroupAssignment) {
	if artifact == nil || len(assignment.Evidence) == 0 {
		return
	}
	if title, ok := assignment.Evidence["title"].(string); ok && strings.TrimSpace(title) != "" {
		artifact.Title = strings.TrimSpace(title)
	}
	if yearValue, ok := assignment.Evidence["year"].(int); ok && yearValue > 0 {
		artifact.Year = &yearValue
	}
}

func mergeStringAnyMaps(base map[string]any, extra map[string]any) map[string]any {
	if base == nil {
		base = make(map[string]any, len(extra))
	}
	for key, value := range extra {
		base[key] = value
	}
	return base
}
