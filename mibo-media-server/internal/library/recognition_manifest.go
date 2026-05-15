package library

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/recognition"
	"github.com/atlan/mibo-media-server/internal/scanrecognition"
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
	sidecarHints, sidecarsByFileID, err := s.recognitionSidecarInputs(ctx, library, files)
	if err != nil {
		return database.RecognitionManifest{}, err
	}
	output := buildScanRecognitionManifestOutput(files, rootPath, sidecarHints)
	lockedKindsByFileID := scanRecognitionLockedKindsByFileID(output.Candidates)
	contentShapeEvidence, err := s.recognitionContentShapeContextEvidence(ctx, library, files, indexedSignals, lockedKindsByFileID)
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
	)
	scope := recognition.ManifestScope{LibraryID: library.ID, MediaSourceID: library.MediaSourceID, StorageProvider: storageProvider, RootPath: rootPath, ScopePath: scopePath, ClassifierVersion: settings.ClassifierVersion, Fingerprint: newRecognitionFingerprint(files), EvidenceJSON: mustJSON(map[string]any{"scheme": "scanrecognition"})}
	var manifest database.RecognitionManifest
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := recognition.NewRepository(tx)
		created, err := repo.UpsertManifest(ctx, scope)
		if err != nil {
			return err
		}
		manifest = created
		for idx := range output.Candidates {
			output.Candidates[idx].ManifestID = manifest.ID
		}
		for idx := range output.Evidence {
			output.Evidence[idx].ManifestID = manifest.ID
		}
		graphOutput := recognition.ConstructGraphFromCandidates(recognition.GraphConstructInput{Scope: scope, Files: files, FileSignals: indexedSignals, SidecarsByFileID: sidecarsByFileID, SidecarHints: sidecarHints, ContextEvidence: contextEvidence}, output.Candidates)
		for idx := range graphOutput.MediaGraphNodes {
			graphOutput.MediaGraphNodes[idx].ManifestID = manifest.ID
		}
		for idx := range graphOutput.MediaGraphEdges {
			graphOutput.MediaGraphEdges[idx].ManifestID = manifest.ID
		}
		for idx := range graphOutput.MediaGraphClassifications {
			graphOutput.MediaGraphClassifications[idx].ManifestID = manifest.ID
		}
		if err := repo.ReplaceCandidatesAndEvidence(ctx, manifest.ID, output.Candidates, output.Evidence); err != nil {
			return err
		}
		if err := repo.SaveMediaGraph(ctx, manifest.ID, graphOutput.MediaGraphNodes, graphOutput.MediaGraphEdges, graphOutput.MediaGraphClassifications); err != nil {
			return err
		}
		graph, err := repo.LoadManifestGraph(ctx, manifest.ID)
		if err != nil {
			return err
		}
		decisions := buildScanRecognitionDecisions(graph.Candidates)
		return repo.ReplaceDecisionsAndConflicts(ctx, manifest.ID, decisions, nil)
	}); err != nil {
		return database.RecognitionManifest{}, err
	}
	return manifest, nil
}

func newRecognitionFingerprint(files []database.InventoryFile) string {
	hash := sha256.New()
	for _, file := range files {
		if file.ID == 0 {
			continue
		}
		hash.Write([]byte(strings.TrimSpace(file.StorageProvider)))
		hash.Write([]byte("\x00"))
		hash.Write([]byte(strings.TrimSpace(file.StoragePath)))
		hash.Write([]byte("\x00"))
		hash.Write([]byte(strings.TrimSpace(file.StableIdentityKey)))
		hash.Write([]byte("\x00"))
	}
	return hex.EncodeToString(hash.Sum(nil))
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

type recognitionLockedKind string

const (
	recognitionLockedKindMovie   recognitionLockedKind = "movie"
	recognitionLockedKindEpisode recognitionLockedKind = "episode"
)

func scanRecognitionLockedKindsByFileID(candidates []database.RecognitionCandidate) map[uint]recognitionLockedKind {
	locked := make(map[uint]recognitionLockedKind)
	for _, candidate := range candidates {
		if candidate.PrimaryInventoryID == nil || *candidate.PrimaryInventoryID == 0 {
			continue
		}
		fileID := *candidate.PrimaryInventoryID
		switch {
		case candidate.CandidateType == recognition.CandidateTypeEpisode:
			locked[fileID] = recognitionLockedKindEpisode
		case candidate.CandidateType == recognition.CandidateTypeWork && (candidate.CandidateRole == recognition.WorkKindSeries || candidate.CandidateRole == recognition.WorkKindSeason):
			if locked[fileID] == "" {
				locked[fileID] = recognitionLockedKindEpisode
			}
		case candidate.CandidateType == recognition.CandidateTypeWork && candidate.CandidateRole == recognition.WorkKindMovie:
			if locked[fileID] == "" {
				locked[fileID] = recognitionLockedKindMovie
			}
		}
	}
	return locked
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

func (s *Service) recognitionContentShapeContextEvidence(ctx context.Context, library database.Library, files []database.InventoryFile, indexedSignals map[uint]database.InventoryFileSignal, lockedKindsByFileID map[uint]recognitionLockedKind) (map[uint][]recognition.ContextEvidence, error) {
	if len(files) == 0 || s.db == nil {
		return nil, nil
	}
	settings := contentShapeSettingsFromConfig(s.cfg)
	cache := newFilenameTokenProfileCache()
	if err := hydrateRecognitionFilenameTokenCache(ctx, s.db, library, settings.ClassifierVersion, files, cache); err != nil {
		return nil, err
	}
	assignmentsByPath := make(map[string]contentShapeFileAssignment, len(files))
	for _, file := range files {
		storagePath := strings.TrimSpace(file.StoragePath)
		if file.ID == 0 || storagePath == "" || file.Status != inventory.FileStatusAvailable || file.ContentClass != SourceContentClassVideo || !isVideoFile(storagePath) {
			continue
		}
		model := filenameTokenProfileForPath(cache, storagePath)
		signal := indexedSignals[file.ID]
		title := strings.TrimSpace(signal.TitleCandidate)
		if title == "" {
			title = filenameSignalTitleCandidate(model)
		}
		year := signal.Year
		if year == nil {
			year = filenameSignalYear(model)
		}
		videoSignal := model.VideoSignal
		if strings.TrimSpace(title) != "" && len(videoSignal.TitleCandidates) == 0 {
			videoSignal.TitleCandidates = []string{title}
		}
		videoSignal.Year = firstNonNilInt(signal.Year, videoSignal.Year, year)
		videoSignal.Season = firstNonNilInt(signal.SeasonNumber, videoSignal.Season, model.PathHints.SeasonNumber)
		videoSignal.Episode = firstNonNilInt(signal.EpisodeNumber, videoSignal.Episode)
		assignment := recognitionContentShapeAssignmentForModel(library.RootPath, storagePath, videoSignal, model.PathHints, lockedKindsByFileID[file.ID])
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

func recognitionContentShapeAssignmentForModel(libraryRoot string, storagePath string, videoSignal scanrecognition.VideoSignal, pathHints filenamePathHints, lockedKind recognitionLockedKind) contentShapeFileAssignment {
	if lockedKind == recognitionLockedKindMovie {
		return contentShapeFileAssignment{}
	}
	if strings.TrimSpace(pathHints.SeriesTitle) == "" {
		pathHints.SeriesTitle = scanrecognition.SeriesTitleFromPath(libraryRoot, storagePath)
	}
	if pathHints.SeasonNumber == nil {
		pathHints.SeasonNumber = scanrecognition.SeasonFromPath(libraryRoot, storagePath)
	}
	seriesTitle := contentShapeEpisodeSeriesTitle(contentShapeDirectoryPlan{}, storage.Object{Path: storagePath}, pathHints)
	seasonNumber := firstNonNilInt(videoSignal.Season, pathHints.SeasonNumber, scanrecognition.SeasonFromPath(libraryRoot, storagePath))
	episodeNumber := firstNonNilInt(videoSignal.Episode, scanrecognition.AnalyzeVideoPath(strings.TrimSuffix(path.Base(storagePath), path.Ext(storagePath))+".mkv").Episode)
	if strings.TrimSpace(seriesTitle) == "" || seasonNumber == nil {
		return contentShapeFileAssignment{}
	}
	if episodeNumber == nil && scanrecognition.WeakEpisodeNumberAllowed(strings.TrimSuffix(path.Base(storagePath), path.Ext(storagePath))) {
		episodeNumber = videoSignal.LeadingNumber
	}
	if episodeNumber == nil || *episodeNumber <= 0 {
		return contentShapeFileAssignment{}
	}
	if lockedKind != "" && lockedKind != recognitionLockedKindEpisode {
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
	default:
		return nil
	}
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
	if !hasScanRecognitionDecisions(graph.Decisions) {
		return result, nil
	}
	materializer := recognition.NewMaterializer(s.db)
	metadataResult, err := materializer.MaterializeMetadata(ctx, graph, graph.Decisions)
	if err != nil {
		return result, err
	}
	resourceResult, err := materializer.MaterializeResources(ctx, graph, graph.Decisions)
	if err != nil {
		return result, err
	}
	return result.Merge(metadataResult).Merge(resourceResult), nil
}

func hasScanRecognitionDecisions(decisions []database.RecognitionDecision) bool {
	for _, decision := range decisions {
		if strings.TrimSpace(decision.DecisionType) == "scanrecognition_outcome" {
			return true
		}
	}
	return false
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
