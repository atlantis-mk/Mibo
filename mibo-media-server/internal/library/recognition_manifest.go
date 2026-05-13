package library

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/recognition"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

func (s *Service) persistRecognitionManifestForFiles(ctx context.Context, library database.Library, files []database.InventoryFile, scopePath string) (database.RecognitionManifest, error) {
	if len(files) == 0 {
		return database.RecognitionManifest{}, nil
	}
	settings := contentShapeSettingsFromConfig(s.cfg)
	storageProvider := strings.TrimSpace(files[0].StorageProvider)
	if storageProvider == "" {
		storageProvider = "local"
	}
	rootPath := strings.TrimSpace(library.RootPath)
	if rootPath == "" {
		rootPath = path.Dir(files[0].StoragePath)
	}
	if strings.TrimSpace(scopePath) == "" {
		scopePath = rootPath
	}
	signalScope := inventoryFileSignalScope{LibraryID: library.ID, StorageProvider: storageProvider, ClassifierVersion: settings.ClassifierVersion}
	_, fileSignals, err := loadReusableInventoryFileSignals(ctx, s.db, signalScope, files)
	if err != nil {
		return database.RecognitionManifest{}, err
	}
	indexedSignals := signalsByFileID(fileSignals)
	contentShapeEvidence, err := s.recognitionContentShapeContextEvidence(ctx, library, files, indexedSignals)
	if err != nil {
		return database.RecognitionManifest{}, err
	}
	pathTreeEvidence := recognitionPathTreeContextEvidence(files, library.RootPath, indexedSignals)
	sidecarHints, sidecarsByFileID, err := s.recognitionSidecarInputs(ctx, library, files)
	if err != nil {
		return database.RecognitionManifest{}, err
	}
	if decision, ok := directoryReductionDecisionForFiles(files, indexedSignals); ok {
		scopePath = firstNonEmptyString(directoryReductionScopePath(decision, scopePath), scopePath)
		if err := saveDirectoryReductionDecision(ctx, s.db, library.ID, decision); err != nil {
			return database.RecognitionManifest{}, err
		}
	}
	contextEvidence := mergeRecognitionContextEvidence(
		directoryReductionContextEvidence(files, indexedSignals),
		contentShapeEvidence,
		pathTreeEvidence,
	)
	excludedFileIDs := directoryReductionExcludedFileIDs(files, indexedSignals)
	input := recognition.GraphConstructInput{
		Scope:            recognition.ManifestScope{LibraryID: library.ID, MediaSourceID: library.MediaSourceID, StorageProvider: storageProvider, RootPath: rootPath, ScopePath: scopePath, ClassifierVersion: settings.ClassifierVersion},
		Files:            files,
		FileSignals:      indexedSignals,
		SidecarsByFileID: sidecarsByFileID,
		SidecarHints:     sidecarHints,
		ContextEvidence:  contextEvidence,
		ExcludedFileIDs:  excludedFileIDs,
	}
	output := recognition.ConstructGraphFromInventory(input)
	repo := recognition.NewRepository(s.db)
	manifest, err := repo.UpsertManifest(ctx, output.ManifestScope)
	if err != nil {
		return database.RecognitionManifest{}, err
	}
	for idx := range output.Candidates {
		output.Candidates[idx].ManifestID = manifest.ID
	}
	for idx := range output.Evidence {
		output.Evidence[idx].ManifestID = manifest.ID
	}
	for idx := range output.MediaGraphNodes {
		output.MediaGraphNodes[idx].ManifestID = manifest.ID
	}
	for idx := range output.MediaGraphEdges {
		output.MediaGraphEdges[idx].ManifestID = manifest.ID
	}
	for idx := range output.MediaGraphClassifications {
		output.MediaGraphClassifications[idx].ManifestID = manifest.ID
	}
	if err := repo.SaveMediaGraph(ctx, manifest.ID, output.MediaGraphNodes, output.MediaGraphEdges, output.MediaGraphClassifications); err != nil {
		return database.RecognitionManifest{}, err
	}
	if err := repo.SaveCandidates(ctx, output.Candidates); err != nil {
		return database.RecognitionManifest{}, err
	}
	if err := repo.SaveEvidence(ctx, output.Evidence); err != nil {
		return database.RecognitionManifest{}, err
	}
	return manifest, nil
}

func mergeRecognitionContextEvidence(groups ...map[uint][]recognition.ContextEvidence) map[uint][]recognition.ContextEvidence {
	var merged map[uint][]recognition.ContextEvidence
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		if merged == nil {
			merged = make(map[uint][]recognition.ContextEvidence)
		}
		for fileID, items := range group {
			if len(items) == 0 {
				continue
			}
			merged[fileID] = append(merged[fileID], items...)
		}
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

func (s *Service) recognitionSidecarInputs(ctx context.Context, library database.Library, files []database.InventoryFile) (map[uint][]recognition.SidecarHint, map[uint][]database.InventoryFile, error) {
	if len(files) == 0 || s.db == nil {
		return nil, nil, nil
	}
	rowsByFileID, err := loadInventorySidecarSignals(ctx, s.db, library.ID, strings.TrimSpace(files[0].StorageProvider), files)
	if err != nil {
		return nil, nil, err
	}
	if len(rowsByFileID) == 0 {
		return nil, nil, nil
	}
	hintsByFileID := make(map[uint][]recognition.SidecarHint)
	sidecarsByFileID := make(map[uint][]database.InventoryFile)
	for _, file := range files {
		if file.ID == 0 {
			continue
		}
		rows := rowsByFileID[file.ID]
		if len(rows) == 0 {
			continue
		}
		hintsByFileID[file.ID] = sidecarHintsFromSignals(rows)
		for _, row := range rows {
			sidecarFile := database.InventoryFile{
				LibraryID:       library.ID,
				MediaSourceID:   library.MediaSourceID,
				StorageProvider: strings.TrimSpace(file.StorageProvider),
				StoragePath:     strings.TrimSpace(row.SidecarPath),
				ContentClass:    classifySourceObject(row.SidecarPath),
				Status:          inventory.FileStatusAvailable,
			}
			if existing, ok, err := loadInventoryFileByStoragePath(ctx, s.db, library.ID, strings.TrimSpace(file.StorageProvider), row.SidecarPath); err != nil {
				return nil, nil, err
			} else if ok {
				sidecarFile = existing
			}
			sidecarsByFileID[file.ID] = append(sidecarsByFileID[file.ID], sidecarFile)
		}
	}
	if len(hintsByFileID) == 0 {
		hintsByFileID = nil
	}
	if len(sidecarsByFileID) == 0 {
		sidecarsByFileID = nil
	}
	return hintsByFileID, sidecarsByFileID, nil
}

func loadInventoryFileByStoragePath(ctx context.Context, db *gorm.DB, libraryID uint, storageProvider string, storagePath string) (database.InventoryFile, bool, error) {
	if db == nil || libraryID == 0 || strings.TrimSpace(storagePath) == "" {
		return database.InventoryFile{}, false, nil
	}
	provider := strings.TrimSpace(storageProvider)
	if provider == "" {
		provider = "local"
	}
	var file database.InventoryFile
	err := db.WithContext(ctx).Where("library_id = ? AND storage_provider = ? AND storage_path = ?", libraryID, provider, strings.TrimSpace(storagePath)).First(&file).Error
	if err == nil {
		return file, true, nil
	}
	if err == gorm.ErrRecordNotFound {
		return database.InventoryFile{}, false, nil
	}
	return database.InventoryFile{}, false, err
}

func mergeMaps(base map[string]any, extras map[string]any) map[string]any {
	if len(base) == 0 && len(extras) == 0 {
		return nil
	}
	merged := make(map[string]any, len(base)+len(extras))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extras {
		merged[key] = value
	}
	return merged
}

func (s *Service) recognitionContentShapeContextEvidence(ctx context.Context, library database.Library, files []database.InventoryFile, indexedSignals map[uint]database.InventoryFileSignal) (map[uint][]recognition.ContextEvidence, error) {
	if len(files) == 0 || s.db == nil {
		return nil, nil
	}
	settings := contentShapeSettingsFromConfig(s.cfg)
	cache := newFilenameTokenProfileCache()
	if err := hydrateRecognitionFilenameTokenCache(ctx, s.db, library, settings.ClassifierVersion, files, cache); err != nil {
		return nil, err
	}
	objects := make([]storage.Object, 0, len(files))
	assignmentsByPath := make(map[string]contentShapeFileAssignment, len(files))
	for _, file := range files {
		storagePath := strings.TrimSpace(file.StoragePath)
		if file.ID == 0 || storagePath == "" || file.Status != inventory.FileStatusAvailable || file.ContentClass != SourceContentClassVideo || !isVideoFile(storagePath) {
			continue
		}
		objects = append(objects, storage.Object{Path: storagePath})
		model := filenameTokenProfileForPath(cache, storagePath)
		signal := indexedSignals[file.ID]
		title := strings.TrimSpace(signal.TitleCandidate)
		if title == "" {
			title = strings.TrimSpace(model.Identity.TitleCandidate)
		}
		year := signal.Year
		if year == nil {
			year = model.Identity.Year
		}
		model.Identity.TitleCandidate = title
		model.Identity.Year = year
		model.Identity.SeasonNumber = firstNonNilInt(signal.SeasonNumber, model.Identity.SeasonNumber, model.PathHints.SeasonNumber)
		model.Identity.EpisodeNumber = firstNonNilInt(signal.EpisodeNumber, model.Identity.EpisodeNumber)
		assignment := recognitionContentShapeAssignmentForModel(library.RootPath, storagePath, model)
		if strings.TrimSpace(assignment.AssignmentType) == "" {
			continue
		}
		assignmentsByPath[storagePath] = assignment
	}
	if len(assignmentsByPath) == 0 {
		return nil, nil
	}
	parentDir := commonRecognitionScopePath(library.RootPath, files)
	plan := contentShapeDirectoryPlan{
		Shape:       recognitionContentShapePlanShape(assignmentsByPath),
		Confidence:  0.86,
		ReviewState: "auto",
		Evidence:    map[string]any{"source": "content_shape"},
	}
	if parentDir != "" {
		plan.Evidence["parent_path"] = parentDir
	}
	return recognitionContextEvidenceFromAssignments(files, contentShapeAssignmentsByPath(contentShapeAssignmentsFromMap(assignmentsByPath)), plan, "content_shape"), nil
}

func hydrateRecognitionFilenameTokenCache(ctx context.Context, db *gorm.DB, library database.Library, classifierVersion string, files []database.InventoryFile, cache *filenameTokenProfileCache) error {
	if db == nil || cache == nil || len(files) == 0 {
		return nil
	}
	storageProvider := strings.TrimSpace(files[0].StorageProvider)
	if storageProvider == "" {
		storageProvider = "local"
	}
	models, _, err := loadReusableInventoryFileSignals(ctx, db, inventoryFileSignalScope{LibraryID: library.ID, StorageProvider: storageProvider, ClassifierVersion: classifierVersion}, files)
	if err != nil {
		return err
	}
	hydrateFilenameTokenCacheFromSignals(cache, models)
	return nil
}

func contentShapeAssignmentsFromMap(assignments map[string]contentShapeFileAssignment) []contentShapeFileAssignment {
	if len(assignments) == 0 {
		return nil
	}
	paths := make([]string, 0, len(assignments))
	for storagePath := range assignments {
		paths = append(paths, storagePath)
	}
	sort.Strings(paths)
	items := make([]contentShapeFileAssignment, 0, len(paths))
	for _, storagePath := range paths {
		items = append(items, assignments[storagePath])
	}
	return items
}

func recognitionContentShapePlanShape(assignments map[string]contentShapeFileAssignment) string {
	for _, assignment := range assignments {
		if assignment.AssignmentType == contentShapeAssignmentEpisode {
			return contentShapeSeasonFolder
		}
	}
	return contentShapeUnknownReview
}

func recognitionContentShapeAssignmentForModel(libraryRoot string, storagePath string, model filenameSignalModel) contentShapeFileAssignment {
	if strings.TrimSpace(model.PathHints.SeriesTitle) == "" {
		model.PathHints.SeriesTitle = tvSeriesTitleFromPath(libraryRoot, storagePath)
	}
	if model.PathHints.SeasonNumber == nil {
		model.PathHints.SeasonNumber = tvSeasonFromPath(libraryRoot, storagePath)
	}
	seriesTitle := contentShapeEpisodeSeriesTitle(contentShapeDirectoryPlan{}, storage.Object{Path: storagePath}, model)
	seasonNumber := firstNonNilInt(model.Identity.SeasonNumber, model.PathHints.SeasonNumber, tvSeasonFromPath(libraryRoot, storagePath))
	episodeNumber := firstNonNilInt(model.Identity.EpisodeNumber, parseEpisodeNumberFromTitle(strings.TrimSuffix(path.Base(storagePath), path.Ext(storagePath)), seriesTitle))
	if strings.TrimSpace(seriesTitle) == "" || seasonNumber == nil {
		return contentShapeFileAssignment{}
	}
	if episodeNumber == nil && weakEpisodeNumberAllowed(strings.TrimSuffix(path.Base(storagePath), path.Ext(storagePath))) {
		episodeNumber = model.Identity.LeadingNumber
	}
	if episodeNumber == nil || *episodeNumber <= 0 {
		return contentShapeFileAssignment{}
	}
	return contentShapeFileAssignment{
		StoragePath:    storagePath,
		AssignmentType: contentShapeAssignmentEpisode,
		TargetKey:      canonicalSeriesPath(seriesTitle),
		SeriesTitle:    seriesTitle,
		SeasonNumber:   seasonNumber,
		EpisodeNumber:  episodeNumber,
		Confidence:     0.86,
		ReviewState:    "auto",
		Evidence:       map[string]any{"source": "content_shape", "shape": contentShapeSeasonFolder},
	}
}

func recognitionPathTreeContextEvidence(files []database.InventoryFile, libraryRoot string, indexedSignals map[uint]database.InventoryFileSignal) map[uint][]recognition.ContextEvidence {
	if len(files) == 0 {
		return nil
	}
	indexedModels := make(map[string]filenameSignalModel, len(indexedSignals))
	for _, file := range files {
		signal, ok := indexedSignals[file.ID]
		if !ok {
			continue
		}
		model := filenameSignalModel{
			Identity: filenameIdentitySignals{
				TitleCandidate: strings.TrimSpace(signal.TitleCandidate),
				Year:           signal.Year,
				SeasonNumber:   signal.SeasonNumber,
				EpisodeNumber:  signal.EpisodeNumber,
				EpisodeSource:  strings.TrimSpace(signal.EpisodeSource),
			},
			RoleHints: filenameRoleHints{
				Role:    strings.TrimSpace(signal.Role),
				IsExtra: signal.IsExtra,
			},
			PathHints: filenamePathHints{
				SeasonNumber: tvSeasonFromPath(libraryRoot, file.StoragePath),
				SeriesTitle:  tvSeriesTitleFromPath(libraryRoot, file.StoragePath),
			},
			ReleaseHints: filenameReleaseHints{
				Quality:      strings.TrimSpace(signal.Quality),
				Codec:        strings.TrimSpace(signal.Codec),
				Audio:        strings.TrimSpace(signal.Audio),
				Subtitle:     strings.TrimSpace(signal.Subtitle),
				HDR:          strings.TrimSpace(signal.HDR),
				Edition:      strings.TrimSpace(signal.Edition),
				ReleaseGroup: strings.TrimSpace(signal.ReleaseGroup),
			},
		}
		indexedModels[strings.TrimSpace(file.StoragePath)] = model
	}
	assignments := compilePathTreeAssignmentsFromFiles(files, libraryRoot, indexedModels, nil)
	if len(assignments) == 0 {
		return nil
	}
	return recognitionContextEvidenceFromAssignments(files, contentShapeAssignmentsByPath(contentShapeAssignmentsFromPathTree(assignments)), contentShapeDirectoryPlan{Shape: pathTreeContentShapePlanForAssignments(commonRecognitionScopePath(libraryRoot, files), assignments).Shape, Confidence: 0.88, ReviewState: "auto"}, "path_tree")
}

func recognitionContextEvidenceFromAssignments(files []database.InventoryFile, assignments map[string]contentShapeFileAssignment, plan contentShapeDirectoryPlan, source string) map[uint][]recognition.ContextEvidence {
	if len(files) == 0 || len(assignments) == 0 {
		return nil
	}
	result := make(map[uint][]recognition.ContextEvidence)
	for _, file := range files {
		assignment, ok := assignments[strings.TrimSpace(file.StoragePath)]
		if !ok {
			continue
		}
		evidence := recognitionContextEvidenceForAssignment(assignment, plan, source)
		if len(evidence) == 0 {
			continue
		}
		result[file.ID] = append(result[file.ID], evidence...)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func recognitionContextEvidenceForAssignment(assignment contentShapeFileAssignment, plan contentShapeDirectoryPlan, source string) []recognition.ContextEvidence {
	reviewState := strings.TrimSpace(assignment.ReviewState)
	if reviewState == "" {
		reviewState = firstNonEmptyString(plan.ReviewState, "auto")
	}
	confidence := assignment.Confidence
	if confidence <= 0 {
		confidence = plan.Confidence
	}
	switch assignment.AssignmentType {
	case contentShapeAssignmentEpisode:
		if strings.TrimSpace(assignment.SeriesTitle) == "" || assignment.SeasonNumber == nil || assignment.EpisodeNumber == nil {
			return nil
		}
		seriesKey := recognition.SeriesWorkKey(assignment.SeriesTitle)
		seasonKey := recognition.SeasonWorkKey(assignment.SeriesTitle, *assignment.SeasonNumber)
		episodeKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: assignment.SeriesTitle, SeasonNumber: *assignment.SeasonNumber, EpisodeNumber: *assignment.EpisodeNumber})
		payload := map[string]any{"series_title": assignment.SeriesTitle}
		return []recognition.ContextEvidence{
			{Source: source, Assignment: "series_identity", TargetKey: assignment.TargetKey, ParentKey: seriesKey, ReviewState: reviewState, Confidence: floatPtr(confidence), Payload: payload},
			{Source: source, Assignment: "season_identity", TargetKey: assignment.TargetKey, ParentKey: seasonKey, ReviewState: reviewState, Confidence: floatPtr(confidence), Payload: map[string]any{"series_title": assignment.SeriesTitle, "season_number": *assignment.SeasonNumber}},
			{Source: source, Assignment: "episode_identity", TargetKey: assignment.TargetKey, ParentKey: episodeKey, ReviewState: reviewState, Confidence: floatPtr(confidence), Payload: map[string]any{"series_title": assignment.SeriesTitle, "season_number": *assignment.SeasonNumber, "episode_number": *assignment.EpisodeNumber}},
		}
	case contentShapeAssignmentVersion:
		if strings.TrimSpace(assignment.TargetKey) == "" {
			return nil
		}
		return []recognition.ContextEvidence{{Source: source, Assignment: "movie_version", TargetKey: assignment.TargetKey, ParentKey: strings.TrimSpace(assignment.TargetKey), ReviewState: reviewState, Confidence: floatPtr(confidence)}}
	case contentShapeAssignmentMovie:
		if strings.TrimSpace(assignment.TargetKey) == "" {
			return nil
		}
		return []recognition.ContextEvidence{{Source: source, Assignment: "movie_collection", TargetKey: assignment.TargetKey, ParentKey: strings.TrimSpace(assignment.TargetKey), ReviewState: reviewState, Confidence: floatPtr(confidence)}}
	default:
		return nil
	}
}

func signalsByFileID(rows map[string]database.InventoryFileSignal) map[uint]database.InventoryFileSignal {
	if len(rows) == 0 {
		return nil
	}
	result := make(map[uint]database.InventoryFileSignal, len(rows))
	for _, row := range rows {
		if row.InventoryFileID == nil || *row.InventoryFileID == 0 {
			continue
		}
		result[*row.InventoryFileID] = row
	}
	return result
}

func (s *Service) resolveRecognitionManifest(ctx context.Context, manifestID uint) (recognition.MaterializeResult, error) {
	var result recognition.MaterializeResult
	if manifestID == 0 {
		return result, nil
	}
	repo := recognition.NewRepository(s.db)
	graph, err := repo.LoadManifestGraph(ctx, manifestID)
	if err != nil {
		return result, err
	}
	rules, err := repo.LoadEnabledRules(ctx, graph.Manifest.LibraryID, graph.Manifest.StorageProvider, graph.Manifest.ScopePath)
	if err != nil {
		return result, err
	}
	resolver := recognition.NewResolver(rules)
	resolved := resolver.Resolve(graph)
	if err := repo.ReplaceDecisionsAndConflicts(ctx, graph.Manifest.ID, resolved.Decisions, resolved.Conflicts); err != nil {
		return result, err
	}
	materializer := recognition.NewMaterializer(s.db)
	metadataResult, err := materializer.MaterializeMetadata(ctx, graph, resolved.Decisions)
	if err != nil {
		return result, err
	}
	resourceResult, err := materializer.MaterializeResources(ctx, graph, resolved.Decisions)
	if err != nil {
		return result, err
	}
	return result.Merge(metadataResult).Merge(resourceResult), nil
}

func (s *Service) markReviewRequiredInventoryFromManifest(ctx context.Context, manifestID uint) error {
	if manifestID == 0 {
		return nil
	}
	repo := recognition.NewRepository(s.db)
	graph, err := repo.LoadManifestGraph(ctx, manifestID)
	if err != nil {
		return err
	}
	fileIDs := make(map[uint]struct{})
	for _, decision := range graph.Decisions {
		if decision.Outcome != recognition.DecisionOutcomeReviewRequired && decision.Outcome != recognition.DecisionOutcomeBlockedConflict && decision.Outcome != recognition.DecisionOutcomeUnmatched {
			continue
		}
		for _, candidate := range graph.Candidates {
			if candidate.ID == 0 || decision.CandidateID == nil || *decision.CandidateID != candidate.ID || candidate.PrimaryInventoryID == nil {
				continue
			}
			fileIDs[*candidate.PrimaryInventoryID] = struct{}{}
		}
	}
	for fileID := range fileIDs {
		if err := s.db.WithContext(ctx).Model(&database.InventoryFile{}).Where("id = ?", fileID).Update("scan_state", inventory.FileScanStateReviewRequired).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) runRecognitionMaterializeBatch(ctx context.Context, library database.Library, rootPath string, files []database.InventoryFile, mode *scanMode) (recognition.MaterializeResult, error) {
	var result recognition.MaterializeResult
	files = materializableRecognitionFiles(files)
	if len(files) == 0 {
		return result, nil
	}
	if mode != nil && mode.deferRecognitionResolution {
		for _, file := range files {
			mode.recordRecognitionResolveCandidate(file.ID)
		}
		return result, nil
	}
	scopePath := commonRecognitionScopePath(rootPath, files)
	manifest, err := s.persistRecognitionManifestForFiles(ctx, library, files, scopePath)
	if err != nil {
		return result, err
	}
	result, err = s.resolveRecognitionManifest(ctx, manifest.ID)
	if err != nil {
		return result, err
	}
	if err := s.markReviewRequiredInventoryFromManifest(ctx, manifest.ID); err != nil {
		return result, err
	}
	if err := s.recordRecognitionMaterializationCompletion(ctx, library.ID, files, result); err != nil {
		return result, err
	}
	if err := s.applyRecognitionFallbackPoster(ctx, files, result.MetadataIDs); err != nil {
		return result, err
	}
	for _, metadataID := range result.MetadataIDs {
		mode.recordMetadataMatchCandidate(metadataID)
	}
	for _, file := range files {
		mode.recordInventoryProbeCandidate(file.ID)
	}
	return result, nil
}

func (s *Service) runRecognitionMaterializeBatchByFileIDs(ctx context.Context, library database.Library, rootPath string, fileIDs []uint, mode *scanMode) (recognition.MaterializeResult, error) {
	ids := normalizeUintIDs(fileIDs)
	if len(ids) == 0 {
		return recognition.MaterializeResult{}, nil
	}
	var files []database.InventoryFile
	for _, batch := range chunkUints(ids, sqliteVariableChunkSize) {
		var partial []database.InventoryFile
		if err := s.db.WithContext(ctx).Where("id IN ?", batch).Find(&partial).Error; err != nil {
			return recognition.MaterializeResult{}, err
		}
		files = append(files, partial...)
	}
	return s.runRecognitionMaterializeBatch(ctx, library, rootPath, files, mode)
}

func materializableRecognitionFiles(files []database.InventoryFile) []database.InventoryFile {
	result := make([]database.InventoryFile, 0, len(files))
	for _, file := range files {
		if file.ID == 0 || file.Status != inventory.FileStatusAvailable || file.ContentClass != SourceContentClassVideo || !isVideoFile(file.StoragePath) {
			continue
		}
		result = append(result, file)
	}
	return result
}

func commonRecognitionScopePath(rootPath string, files []database.InventoryFile) string {
	trimmedRoot := strings.TrimSpace(rootPath)
	if len(files) == 0 {
		return trimmedRoot
	}
	scope := path.Dir(strings.TrimSpace(files[0].StoragePath))
	for _, file := range files[1:] {
		scope = commonPathPrefix(scope, path.Dir(strings.TrimSpace(file.StoragePath)))
	}
	if trimmedRoot != "" && (scope == "." || !strings.HasPrefix(scope, trimmedRoot)) {
		return trimmedRoot
	}
	if scope == "." || scope == "" {
		return trimmedRoot
	}
	return scope
}

func commonPathPrefix(left string, right string) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" || right == "" {
		return ""
	}
	for left != "." && left != "/" {
		if right == left || strings.HasPrefix(right, left+"/") {
			return left
		}
		left = path.Dir(left)
	}
	if strings.HasPrefix(right, "/") {
		return "/"
	}
	return ""
}

func (s *Service) recordRecognitionMaterializationCompletion(ctx context.Context, libraryID uint, files []database.InventoryFile, result recognition.MaterializeResult) error {
	ingestSvc := s.ingestCapability()
	if ingestSvc == nil || len(files) == 0 {
		return nil
	}
	fileIDs := make([]uint, 0, len(files))
	events := make([]database.IngestEvent, 0, len(files))
	var resourceID *uint
	if len(result.ResourceIDs) > 0 {
		value := result.ResourceIDs[0]
		resourceID = &value
	}
	for _, file := range files {
		if file.ID == 0 {
			continue
		}
		fileIDs = append(fileIDs, file.ID)
		event := database.IngestEvent{UnitKey: inventoryFileUnitKey(file.ID), LibraryID: libraryID, InventoryFileID: &file.ID, ConditionType: ingest.ConditionMaterialized, EventType: ingest.EventConditionChanged, Status: ingest.ConditionStatusTrue, Reason: "recognition_materialization_completed", Message: "Recognition resolver materialization completed"}
		if resourceID != nil {
			event.ResourceID = resourceID
		}
		events = append(events, event)
	}
	if len(fileIDs) > 0 {
		if err := ingestSvc.MarkInventoryFilesDirty(ctx, fileIDs, "recognition_materialization_completed"); err != nil {
			return err
		}
	}
	if len(events) > 0 {
		if err := ingestSvc.AppendEvents(ctx, events); err != nil {
			return fmt.Errorf("record recognition materialization events: %w", err)
		}
	}
	return nil
}

func (s *Service) applyRecognitionFallbackPoster(ctx context.Context, files []database.InventoryFile, metadataIDs []uint) error {
	if s.db == nil || len(metadataIDs) == 0 {
		return nil
	}
	thumbnailByFileID := make(map[uint]string, len(files))
	for _, file := range files {
		if file.ID == 0 {
			continue
		}
		if trimmed := strings.TrimSpace(file.ThumbnailURL); trimmed != "" {
			thumbnailByFileID[file.ID] = trimmed
		}
	}
	if len(thumbnailByFileID) == 0 {
		return nil
	}
	targets, err := s.metadataFallbackPosterTargets(ctx, metadataIDs)
	if err != nil {
		return err
	}
	for _, target := range targets {
		if target.MetadataItemID == 0 || target.InventoryFileID == 0 {
			continue
		}
		posterURL := strings.TrimSpace(thumbnailByFileID[target.InventoryFileID])
		if posterURL == "" {
			continue
		}
		if target.HasPoster {
			continue
		}
		row := database.MetadataItemImage{MetadataItemID: target.MetadataItemID, ImageType: "poster", URL: posterURL, IsSelected: true, SortOrder: 0}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

type metadataFallbackPosterTarget struct {
	MetadataItemID  uint
	InventoryFileID uint
	HasPoster       bool
}

func (s *Service) metadataFallbackPosterTargets(ctx context.Context, metadataIDs []uint) ([]metadataFallbackPosterTarget, error) {
	ids := normalizeUintIDs(metadataIDs)
	if s.db == nil || len(ids) == 0 {
		return nil, nil
	}
	type targetRow struct {
		MetadataItemID  uint
		InventoryFileID uint
		PosterImageID   *uint
	}
	rows := make([]targetRow, 0, len(ids))
	if err := s.db.WithContext(ctx).
		Table("resource_metadata_links").
		Select("resource_metadata_links.metadata_item_id, resource_files.inventory_file_id, metadata_item_images.id as poster_image_id").
		Joins("JOIN resource_files ON resource_files.resource_id = resource_metadata_links.resource_id AND resource_files.role = ?", database.ResourceFileRoleSource).
		Joins("LEFT JOIN metadata_item_images ON metadata_item_images.metadata_item_id = resource_metadata_links.metadata_item_id AND metadata_item_images.image_type = ?", "poster").
		Where("resource_metadata_links.metadata_item_id IN ?", ids).
		Order("resource_metadata_links.metadata_item_id asc, resource_files.id asc").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]metadataFallbackPosterTarget, 0, len(rows))
	seen := make(map[uint]struct{}, len(rows))
	for _, row := range rows {
		if row.MetadataItemID == 0 || row.InventoryFileID == 0 {
			continue
		}
		if _, ok := seen[row.MetadataItemID]; ok {
			continue
		}
		seen[row.MetadataItemID] = struct{}{}
		result = append(result, metadataFallbackPosterTarget{MetadataItemID: row.MetadataItemID, InventoryFileID: row.InventoryFileID, HasPoster: row.PosterImageID != nil && *row.PosterImageID != 0})
	}
	return result, nil
}

func (s *Service) BackfillRecognitionFallbackPosters(ctx context.Context) (int64, error) {
	if s.db == nil {
		return 0, nil
	}
	type backfillRow struct {
		MetadataItemID uint
		ThumbnailURL   string
	}
	rows := make([]backfillRow, 0)
	if err := s.db.WithContext(ctx).
		Table("resource_metadata_links").
		Select("resource_metadata_links.metadata_item_id, inventory_files.thumbnail_url").
		Joins("JOIN resource_files ON resource_files.resource_id = resource_metadata_links.resource_id AND resource_files.role = ?", database.ResourceFileRoleSource).
		Joins("JOIN inventory_files ON inventory_files.id = resource_files.inventory_file_id").
		Joins("LEFT JOIN metadata_item_images ON metadata_item_images.metadata_item_id = resource_metadata_links.metadata_item_id AND metadata_item_images.image_type = ?", "poster").
		Where("metadata_item_images.id IS NULL").
		Where("inventory_files.thumbnail_url <> ''").
		Order("resource_metadata_links.metadata_item_id asc, resource_files.id asc").
		Scan(&rows).Error; err != nil {
		return 0, err
	}
	created := int64(0)
	seen := make(map[uint]struct{}, len(rows))
	for _, row := range rows {
		if row.MetadataItemID == 0 || strings.TrimSpace(row.ThumbnailURL) == "" {
			continue
		}
		if _, ok := seen[row.MetadataItemID]; ok {
			continue
		}
		seen[row.MetadataItemID] = struct{}{}
		image := database.MetadataItemImage{MetadataItemID: row.MetadataItemID, ImageType: "poster", URL: strings.TrimSpace(row.ThumbnailURL), IsSelected: true, SortOrder: 0}
		if err := s.db.WithContext(ctx).Create(&image).Error; err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}
