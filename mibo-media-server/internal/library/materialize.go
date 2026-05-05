package library

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	if err := s.preloadMaterializeBatchFiles(ctx, &batchState, fileIDs); err != nil {
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
	return fmt.Errorf("probe service unavailable for workflow batch")
}

func (s *Service) RunCatalogMatchBatch(ctx context.Context, payload CatalogMatchBatchPayload) error {
	return fmt.Errorf("metadata service unavailable for workflow batch")
}

type materializeBatchState struct {
	scanPolicy          database.LibraryScanPolicy
	subtitlePolicy      database.LibrarySubtitlePolicy
	exclusionRules      []database.ScanExclusionRule
	providersByKey      map[string]materializeProviderBinding
	directorySnapshots  map[string]scanDirectorySnapshot
	filesByID           map[uint]database.InventoryFile
	catalogCache        *catalogScanBatchCache
	classificationCache *classificationDirectoryCache
}

type materializeProviderBinding struct {
	pathRecord database.LibraryPath
	provider   storage.Provider
}

func (s *Service) newMaterializeBatchState(ctx context.Context, config EffectiveLibraryConfig) (materializeBatchState, error) {
	state := materializeBatchState{
		scanPolicy:          config.ScanPolicy,
		subtitlePolicy:      config.SubtitlePolicy,
		providersByKey:      make(map[string]materializeProviderBinding, len(config.Paths)),
		directorySnapshots:  make(map[string]scanDirectorySnapshot),
		filesByID:           make(map[uint]database.InventoryFile),
		catalogCache:        newCatalogScanBatchCache(),
		classificationCache: newClassificationDirectoryCache(),
	}
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
	decisionSnapshot, err := s.directoryShapeSnapshot(ctx, provider, config.Library, snapshot, batchState.exclusionRules, batchState.scanPolicy)
	if err != nil {
		return catalogScanWriteResult{}, err
	}
	libraryForPath := config.Library
	libraryForPath.MediaSourceID = pathRecord.MediaSourceID
	libraryForPath.RootPath = pathRecord.RootPath
	writeResult, _, err := s.materializeObjectFromSnapshotWithDirectorySnapshots(ctx, provider, libraryForPath, object, snapshot, decisionSnapshot, directorySnapshots, batchState.classificationCache, batchState.subtitlePolicy, batchState.catalogCache)
	if err != nil {
		return catalogScanWriteResult{}, err
	}
	if writeResult.Item.ID != 0 {
		s.markProjectionItemDirty(ctx, writeResult.Item.ID, "materialization_completed")
	}
	return writeResult, nil
}

func (s *Service) preloadMaterializeBatchFiles(ctx context.Context, batchState *materializeBatchState, fileIDs []uint) error {
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
		decisionSnapshot, err := s.directoryShapeSnapshot(ctx, provider, config.Library, snapshot, batchState.exclusionRules, batchState.scanPolicy)
		if err != nil {
			return err
		}
		libraryForPath := config.Library
		libraryForPath.MediaSourceID = pathRecord.MediaSourceID
		libraryForPath.RootPath = pathRecord.RootPath
		artifact, ok := preloadCatalogScanArtifact(provider, libraryForPath, object, snapshot, decisionSnapshot, directorySnapshots, batchState.classificationCache)
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

func preloadCatalogScanArtifact(provider storage.Provider, library database.Library, object storage.Object, snapshot scanDirectorySnapshot, decisionSnapshot scanDirectorySnapshot, directorySnapshots map[string]scanDirectorySnapshot, classificationCache *classificationDirectoryCache) (catalogScanArtifact, bool) {
	if classificationCache == nil {
		classificationCache = newClassificationDirectoryCache()
	}
	classificationLibraryType := effectiveVideoLibraryType(library.Type)
	directoryDecision := classificationCache.decision(classificationLibraryType, library.RootPath, decisionSnapshot)
	if shouldSkipTVDirectoryExtra(classificationLibraryType, directoryDecision, object) {
		return catalogScanArtifact{}, false
	}
	directorySummary := classificationCache.summary(classificationLibraryType, library.RootPath, decisionSnapshot)
	inherited := classificationCache.inheritedContext(classificationLibraryType, library.RootPath, object.Path, decisionSnapshot, directoryDecision, directorySnapshots)
	classified := classifyMediaFileWithDirectorySummaryAndContext(classificationLibraryType, library.RootPath, object, snapshot.Path, directoryDecision, decisionSnapshot, directorySummary, inherited)
	artifact, _ := catalogScanArtifactFromObject(provider.Name(), classificationLibraryType, library.RootPath, object, classified)
	return artifact, true
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
		item := database.CatalogItem{
			LibraryID:           input.LibraryID,
			Type:                strings.TrimSpace(input.Type),
			ParentID:            input.ParentID,
			Path:                strings.TrimSpace(input.Path),
			SortKey:             strings.TrimSpace(input.SortKey),
			DisplayOrder:        defaultCatalogState(strings.TrimSpace(input.DisplayOrder), catalog.DisplayOrderAired),
			IndexNumber:         input.IndexNumber,
			IndexNumberEnd:      input.IndexNumberEnd,
			ParentIndexNumber:   input.ParentIndexNumber,
			AbsoluteNumber:      input.AbsoluteNumber,
			Title:               strings.TrimSpace(input.Title),
			OriginalTitle:       strings.TrimSpace(input.OriginalTitle),
			SortTitle:           strings.TrimSpace(input.SortTitle),
			Overview:            input.Overview,
			ReleaseDate:         input.ReleaseDate,
			FirstAirDate:        input.FirstAirDate,
			LastAirDate:         input.LastAirDate,
			Year:                input.Year,
			EndYear:             input.EndYear,
			RuntimeSeconds:      input.RuntimeSeconds,
			CommunityRating:     input.CommunityRating,
			OfficialRating:      strings.TrimSpace(input.OfficialRating),
			SeriesStatus:        strings.TrimSpace(input.SeriesStatus),
			AvailabilityStatus:  defaultCatalogState(input.AvailabilityStatus, catalog.AvailabilityNoLocalMedia),
			GovernanceStatus:    defaultCatalogState(input.GovernanceStatus, catalog.GovernancePending),
			CanonicalVersion:    1,
			LastCanonicalizedAt: func() *time.Time { now := time.Now().UTC(); return &now }(),
		}
		if item.ParentID != nil {
			var parent database.CatalogItem
			if err := tx.First(&parent, *item.ParentID).Error; err != nil {
				return nil, fmt.Errorf("load parent item: %w", err)
			}
			if parent.RootID != nil {
				item.RootID = parent.RootID
			} else {
				item.RootID = &parent.ID
			}
		}
		items = append(items, item)
	}
	if err := tx.CreateInBatches(&items, sqliteWideWriteBatchSize).Error; err != nil {
		return nil, err
	}
	for idx := range items {
		if items[idx].RootID == nil {
			items[idx].RootID = &items[idx].ID
		}
	}
	for _, batch := range chunkCatalogItems(items, sqliteVariableChunkSize) {
		for _, item := range batch {
			if err := tx.Model(&database.CatalogItem{}).Where("id = ?", item.ID).Update("root_id", *item.RootID).Error; err != nil {
				return nil, err
			}
		}
	}
	identities := make([]database.CatalogIdentity, 0, len(items))
	for _, item := range items {
		identityKey, ok := catalog.ScannerIdentityKeyForItem(item)
		if !ok {
			continue
		}
		identities = append(identities, database.CatalogIdentity{ItemID: item.ID, Provider: catalog.IdentityProviderScanner, IdentityType: item.Type, IdentityKey: identityKey, SourcePath: item.Path})
	}
	if len(identities) > 0 {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "provider"}, {Name: "identity_type"}, {Name: "identity_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"item_id", "source_path", "updated_at"}),
		}).CreateInBatches(&identities, sqliteNarrowWriteBatchSize).Error; err != nil {
			return nil, err
		}
	}
	for idx := range items {
		if err := tx.First(&items[idx], items[idx].ID).Error; err != nil {
			return nil, err
		}
	}
	_ = catalogSvc
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

func (s *Service) materializeObjectFromSnapshot(ctx context.Context, provider storage.Provider, library database.Library, object storage.Object, snapshot scanDirectorySnapshot, decisionSnapshot scanDirectorySnapshot, subtitlePolicy database.LibrarySubtitlePolicy, batchCache *catalogScanBatchCache) (catalogScanWriteResult, []string, error) {
	return s.materializeObjectFromSnapshotWithDirectorySnapshots(ctx, provider, library, object, snapshot, decisionSnapshot, nil, nil, subtitlePolicy, batchCache)
}

func (s *Service) materializeObjectFromSnapshotWithDirectorySnapshots(ctx context.Context, provider storage.Provider, library database.Library, object storage.Object, snapshot scanDirectorySnapshot, decisionSnapshot scanDirectorySnapshot, directorySnapshots map[string]scanDirectorySnapshot, classificationCache *classificationDirectoryCache, subtitlePolicy database.LibrarySubtitlePolicy, batchCache *catalogScanBatchCache) (catalogScanWriteResult, []string, error) {
	if classificationCache == nil {
		classificationCache = newClassificationDirectoryCache()
	}
	classificationLibraryType := effectiveVideoLibraryType(library.Type)
	directoryDecision := classificationCache.decision(classificationLibraryType, library.RootPath, decisionSnapshot)
	if shouldSkipTVDirectoryExtra(classificationLibraryType, directoryDecision, object) {
		return catalogScanWriteResult{SkippedReason: "materialization_skipped_tv_extra"}, nil, nil
	}
	directorySummary := classificationCache.summary(classificationLibraryType, library.RootPath, decisionSnapshot)
	inherited := classificationCache.inheritedContext(classificationLibraryType, library.RootPath, object.Path, decisionSnapshot, directoryDecision, directorySnapshots)
	classified := classifyMediaFileWithDirectorySummaryAndContext(classificationLibraryType, library.RootPath, object, snapshot.Path, directoryDecision, decisionSnapshot, directorySummary, inherited)
	artifact, _ := catalogScanArtifactFromObject(provider.Name(), classificationLibraryType, library.RootPath, object, classified)
	if decision := scanDecisionFromDirectoryShape(directoryDecision, artifact); strings.TrimSpace(decision.Type) != "" {
		artifact.Decisions = append(artifact.Decisions, decision)
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
