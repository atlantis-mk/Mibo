package library

import (
	"context"
	"encoding/json"
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
	"github.com/atlan/mibo-media-server/internal/storageindex"
	"github.com/atlan/mibo-media-server/internal/titleclean"
	"gorm.io/gorm"
)

type discoveredInventoryCandidate struct {
	object    storage.Object
	container string
}

type discoveredStableIdentityBinding struct {
	file      database.InventoryFile
	container string
	hashJSON  string
	object    storage.Object
}

type discoveredStableIdentityUpdate struct {
	file      database.InventoryFile
	container string
	hashJSON  string
	object    storage.Object
	resultKey string
}

const recognitionResolveScanBatchSize = 25

func (s *Service) QueueLibraryScan(ctx context.Context, libraryID uint) (database.Job, error) {
	var record database.Library
	if err := s.db.WithContext(ctx).First(&record, libraryID).Error; err != nil {
		return database.Job{}, err
	}
	s.markLibraryScopeDirty(ctx, record.ID, record.RootPath, "library_scan_queued")
	return database.Job{}, fmt.Errorf("workflow service unavailable")
}

func (s *Service) RunSyncLibrary(ctx context.Context, job database.Job) error {
	type syncLibraryPayload struct {
		LibraryID uint   `json:"library_id"`
		RootPath  string `json:"root_path"`
	}
	var payload syncLibraryPayload
	if err := json.Unmarshal([]byte(job.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("decode sync_library payload: %w", err)
	}
	config, err := s.EffectiveLibraryConfig(ctx, payload.LibraryID)
	if err != nil {
		return err
	}
	if err := s.updateLibraryStatus(ctx, config.Library.ID, "syncing"); err != nil {
		return err
	}
	paths := config.Paths
	if strings.TrimSpace(payload.RootPath) != "" {
		pathRecord, err := config.pathForRoot(payload.RootPath)
		if err != nil {
			_ = s.updateLibraryStatus(ctx, config.Library.ID, "error")
			return err
		}
		paths = []database.LibraryPath{pathRecord}
	}
	for _, pathRecord := range paths {
		s.markLibraryScopeDirty(ctx, config.Library.ID, pathRecord.RootPath, "library_scan_started")
	}
	var result SyncResult
	for _, pathRecord := range paths {
		provider, err := s.providerForLibraryPath(ctx, pathRecord)
		if err != nil {
			_ = s.updateLibraryStatus(ctx, config.Library.ID, "error")
			return err
		}
		libraryForPath := config.Library
		libraryForPath.MediaSourceID = pathRecord.MediaSourceID
		libraryForPath.RootPath = pathRecord.RootPath
		scanMode := scanMode{deferRecognitionResolution: s.cfg.Worker.Enabled}
		pathResult, err := s.scanLibraryWithMode(ctx, provider, libraryForPath, pathRecord.RootPath, &scanMode)
		if err != nil {
			_ = s.updateLibraryStatus(ctx, config.Library.ID, "error")
			return err
		}
		result.add(pathResult)
		if _, err := s.QueueCatalogLibraryProjectionRefresh(ctx, config.Library.ID, pathRecord.RootPath); err != nil {
			return err
		}
		if err := s.queuePostScanEnrichment(ctx, config.Library.ID, pathRecord.RootPath, scanMode, config.ScanPolicy); err != nil {
			return err
		}
	}
	if err := s.updateLibraryStatus(ctx, config.Library.ID, "active"); err != nil {
		return err
	}
	_ = result
	return nil
}

func (s *Service) RunTargetedRefresh(ctx context.Context, job database.Job) error {
	var payload targetedRefreshPayload
	if err := json.Unmarshal([]byte(job.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("decode targeted_refresh payload: %w", err)
	}
	config, err := s.EffectiveLibraryConfig(ctx, payload.LibraryID)
	if err != nil {
		return err
	}
	pathRecord, provider, rootPath, err := s.scopedRefreshPath(ctx, config, payload.RootPath)
	if err != nil {
		return err
	}
	s.markLibraryScopeDirty(ctx, config.Library.ID, rootPath, "targeted_refresh_started")
	if err := s.updateLibraryStatus(ctx, config.Library.ID, "syncing"); err != nil {
		return err
	}
	libraryForPath := config.Library
	libraryForPath.MediaSourceID = pathRecord.MediaSourceID
	libraryForPath.RootPath = pathRecord.RootPath
	scanMode := scanMode{partial: true, rootPath: rootPath, deferRecognitionResolution: s.cfg.Worker.Enabled}
	result, err := s.scanLibraryWithMode(ctx, provider, libraryForPath, rootPath, &scanMode)
	if err != nil {
		_ = s.updateLibraryStatus(ctx, config.Library.ID, "error")
		return err
	}
	if err := s.updateLibraryStatus(ctx, config.Library.ID, "active"); err != nil {
		return err
	}
	if _, err := s.QueueCatalogLibraryProjectionRefresh(ctx, config.Library.ID, rootPath); err != nil {
		return err
	}
	if err := s.queuePostScanEnrichment(ctx, config.Library.ID, rootPath, scanMode, config.ScanPolicy); err != nil {
		return err
	}
	_ = result
	return nil
}

func (r *SyncResult) add(other SyncResult) {
	if r == nil {
		return
	}
	r.DirectoriesScanned += other.DirectoriesScanned
	r.FilesSeen += other.FilesSeen
	r.MetadataItemsSeen += other.MetadataItemsSeen
	r.InventoryFilesSeen += other.InventoryFilesSeen
	r.ExcludedFilesSkipped += other.ExcludedFilesSkipped
	for reason, count := range other.ExcludedFilesSkippedByReason {
		if r.ExcludedFilesSkippedByReason == nil {
			r.ExcludedFilesSkippedByReason = map[string]int{}
		}
		r.ExcludedFilesSkippedByReason[reason] += count
	}
}

func (c EffectiveLibraryConfig) pathForRoot(rootPath string) (database.LibraryPath, error) {
	normalized := normalizePath(rootPath)
	for _, pathRecord := range c.Paths {
		if normalizePath(pathRecord.RootPath) == normalized {
			return pathRecord, nil
		}
	}
	return database.LibraryPath{}, scopedRefreshRootError(rootPath)
}

func (s *Service) providerForLibraryPath(ctx context.Context, pathRecord database.LibraryPath) (storage.Provider, error) {
	_, provider, err := s.providerForSource(ctx, pathRecord.MediaSourceID)
	return provider, err
}

func (s *Service) scopedRefreshPath(ctx context.Context, config EffectiveLibraryConfig, rootPath string) (database.LibraryPath, storage.Provider, string, error) {
	for _, pathRecord := range config.Paths {
		provider, err := s.providerForLibraryPath(ctx, pathRecord)
		if err != nil {
			return database.LibraryPath{}, nil, "", err
		}
		targetRoot, err := scopedRefreshRoot(provider.Name(), pathRecord.RootPath, rootPath)
		if err == nil {
			return pathRecord, provider, targetRoot, nil
		}
	}
	return database.LibraryPath{}, nil, "", scopedRefreshRootError(rootPath)
}

func (s *Service) scanDecisionSnapshot(ctx context.Context, provider storage.Provider, library database.Library, snapshot scanDirectorySnapshot, exclusionRules []database.ScanExclusionRule, scanPolicy database.LibraryScanPolicy, mode *scanMode) (scanDirectorySnapshot, error) {
	if mode != nil {
		if cached, ok := mode.decisionSnapshot(snapshot.Path); ok {
			return cached, nil
		}
	}
	decisionSnapshot, err := s.filteredScanSnapshot(ctx, provider, library, snapshot, exclusionRules, scanPolicy)
	if err != nil {
		return scanDirectorySnapshot{}, err
	}
	if mode != nil {
		mode.recordDecisionSnapshot(decisionSnapshot)
	}
	return decisionSnapshot, nil
}

func (s *Service) canShortCircuitDirectoryMaterialization(ctx context.Context, provider storage.Provider, library database.Library, snapshot scanDirectorySnapshot, exclusionRules []database.ScanExclusionRule, scanPolicy database.LibraryScanPolicy, mode *scanMode) (bool, error) {
	if mode == nil || s.db == nil || mode.partial || !mode.deferRecognitionResolution {
		return false, nil
	}
	if mode.directoryResolved(snapshot.Path) {
		return false, nil
	}
	decisionSnapshot, err := s.scanDecisionSnapshot(ctx, provider, library, snapshot, exclusionRules, scanPolicy, mode)
	if err != nil {
		return false, err
	}
	visiblePaths := contentShapeVisibleVideoPaths(decisionSnapshot)
	if len(visiblePaths) == 0 {
		return false, nil
	}
	if !s.db.Migrator().HasTable("resource_files") {
		return false, nil
	}
	var rows []struct {
		StoragePath string
	}
	if err := s.db.WithContext(ctx).
		Table("inventory_files").
		Select("DISTINCT inventory_files.storage_path").
		Joins("JOIN resource_files ON resource_files.inventory_file_id = inventory_files.id AND resource_files.role = ? AND resource_files.part_index = 0", inventory.FileRoleSource).
		Where("inventory_files.library_id = ? AND inventory_files.deleted_at IS NULL AND inventory_files.status = ? AND inventory_files.storage_provider = ? AND inventory_files.storage_path IN ?", library.ID, inventory.FileStatusAvailable, strings.TrimSpace(provider.Name()), visiblePaths).
		Scan(&rows).Error; err != nil {
		return false, err
	}
	if len(rows) != len(visiblePaths) {
		return false, nil
	}
	mode.markDirectoryResolved(snapshot.Path)
	return true, nil
}

func (s *Service) scanLibrary(ctx context.Context, provider storage.Provider, library database.Library, rootPath string) (SyncResult, error) {
	return s.scanLibraryWithMode(ctx, provider, library, rootPath, nil)
}

func (s *Service) scanLibraryWithMode(ctx context.Context, provider storage.Provider, library database.Library, rootPath string, mode *scanMode) (SyncResult, error) {
	if err := ctx.Err(); err != nil {
		return SyncResult{}, err
	}
	workingMode := mode
	if workingMode == nil {
		workingMode = &scanMode{}
	}
	resolved, err := provider.ResolveStorage(ctx, storage.ResolveStorageRequest{Path: rootPath})
	if err != nil {
		return SyncResult{}, fmt.Errorf("resolve library root: %w", err)
	}
	if !resolved.Object.IsDir {
		return SyncResult{}, fmt.Errorf("library root %s is not a directory", rootPath)
	}
	seenFiles := make(map[string]struct{})
	seenItems := make(map[string]struct{})
	result := SyncResult{}
	scanPolicy, err := loadScanPolicy(ctx, s.db, library.ID)
	if err != nil {
		return SyncResult{}, err
	}
	effectiveConfig := EffectiveLibraryConfig{ScanPolicy: scanPolicy}
	if !effectiveConfig.ScannerEnabled() {
		return result, nil
	}
	subtitlePolicy, err := loadSubtitlePolicy(ctx, s.db, library.ID)
	if err != nil {
		return SyncResult{}, err
	}
	var exclusionRules []database.ScanExclusionRule
	if scanPolicy.ConfigurableExclusionRules {
		exclusionRules, err = s.enabledScanExclusionRules(ctx, library.ID)
		if err != nil {
			return SyncResult{}, err
		}
	}
	if err := s.walkDirectory(ctx, provider, library, rootPath, seenFiles, seenItems, &result, exclusionRules, scanPolicy, subtitlePolicy, workingMode); err != nil {
		return SyncResult{}, err
	}
	if workingMode.allowsMissingCleanup() {
		if err := s.cleanupMissingCatalog(ctx, library.ID, rootPath, seenFiles, workingMode.skippedDirectoryPaths()); err != nil {
			return SyncResult{}, err
		}
	}
	_ = seenFiles
	_ = seenItems
	_ = mode
	return result, nil
}

func (s *Service) walkDirectory(ctx context.Context, provider storage.Provider, library database.Library, dirPath string, seenFiles map[string]struct{}, seenItems map[string]struct{}, result *SyncResult, exclusionRules []database.ScanExclusionRule, scanPolicy database.LibraryScanPolicy, subtitlePolicy database.LibrarySubtitlePolicy, mode *scanMode) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	result.DirectoriesScanned++
	snapshot, err := s.collectDirectorySnapshot(ctx, provider, dirPath, true)
	if err != nil {
		return err
	}
	if mode != nil {
		mode.recordDirectorySnapshot(snapshot)
	}
	objects := snapshot.Objects
	sort.Slice(objects, func(i, j int) bool { return objects[i].Path < objects[j].Path })
	discoveredCandidates := make([]discoveredInventoryCandidate, 0)
	for _, object := range objects {
		if err := ctx.Err(); err != nil {
			return err
		}
		if object.IsDir {
			if shouldSkipByScanPolicy(object, scanPolicy) {
				result.recordExcludedFileSkipped("policy_hidden")
				continue
			}
			if err := s.walkDirectory(ctx, provider, library, object.Path, seenFiles, seenItems, result, exclusionRules, scanPolicy, subtitlePolicy, mode); err != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
				mode.recordSkippedDirectory(object.Path, err)
				s.recordScanDirectoryFailure(ctx, provider, library, object.Path, err)
				continue
			}
			continue
		}
		if !isVideoFile(object.Path) {
			if className := classifySourceObject(object.Path); className != SourceContentClassOther {
				if err := s.upsertNonVideoInventoryFile(ctx, provider, library, object, className); err != nil {
					return err
				}
				result.InventoryFilesSeen++
			}
			continue
		}
		exclusion, err := s.scanExclusionDecisionWithRules(ctx, library, provider.Name(), object, exclusionRules)
		if err != nil {
			return err
		}
		if exclusion.Excluded {
			result.recordExcludedFileSkipped(exclusion.Source)
			continue
		}
		if reason := scanPolicySkipReason(object, scanPolicy); reason != "" {
			result.recordExcludedFileSkipped(reason)
			continue
		}
		result.FilesSeen++
		seenFiles[object.Path] = struct{}{}
		discoveredCandidates = append(discoveredCandidates, discoveredInventoryCandidate{object: object, container: strings.TrimPrefix(strings.ToLower(path.Ext(object.Path)), ".")})
	}
	decisionSnapshot, err := s.scanDecisionSnapshot(ctx, provider, library, snapshot, exclusionRules, scanPolicy, mode)
	if err != nil {
		return err
	}
	if skip, err := s.canShortCircuitDirectoryMaterialization(ctx, provider, library, snapshot, exclusionRules, scanPolicy, mode); err != nil {
		return err
	} else if skip {
		return nil
	}
	if err := s.flushDiscoveredInventoryCandidates(ctx, provider, library, snapshot, decisionSnapshot, exclusionRules, scanPolicy, subtitlePolicy, mode, seenFiles, result, discoveredCandidates); err != nil {
		return err
	}
	if err := s.flushPendingSiblingMovieFiles(ctx, mode, snapshot.Path, seenFiles, result); err != nil {
		return err
	}
	if err := s.flushRecognitionResolveCandidates(ctx, library.ID, rootPathForScanMode(library.RootPath, mode), mode, recognitionResolveScanBatchSize); err != nil {
		return err
	}
	return nil
}

func (s *Service) recordScanDirectoryFailure(ctx context.Context, provider storage.Provider, library database.Library, dirPath string, err error) {
	if err == nil {
		return
	}
	if _, recordErr := storageindex.NewService(s.db).RecordFailure(ctx, storageindex.FailureInput{
		LibraryID:       library.ID,
		StorageProvider: provider.Name(),
		StoragePath:     dirPath,
		Reason:          "scan_directory_skipped",
		Error:           err,
	}); recordErr != nil {
		return
	}
}

func (s *Service) flushDiscoveredInventoryCandidates(ctx context.Context, provider storage.Provider, library database.Library, snapshot scanDirectorySnapshot, decisionSnapshot scanDirectorySnapshot, exclusionRules []database.ScanExclusionRule, scanPolicy database.LibraryScanPolicy, subtitlePolicy database.LibrarySubtitlePolicy, mode *scanMode, seenFiles map[string]struct{}, result *SyncResult, candidates []discoveredInventoryCandidate) error {
	if len(candidates) == 0 {
		return nil
	}
	files, err := s.bulkUpsertDiscoveredInventoryFiles(ctx, provider, library, candidates)
	if err != nil {
		return err
	}
	if mode != nil {
		mode.recordDiscoveredFiles(files)
		mode.mergePathTreeAssignments(compilePathTreeAssignmentsFromFiles(mode.discoveredVideoFilesInSnapshot(decisionSnapshot), library.RootPath, nil, newFilenameTokenProfileCache()))
	}
	batchFiles := make([]database.InventoryFile, 0, len(candidates))
	for _, candidate := range candidates {
		fileKey := strings.TrimSpace(provider.Name()) + "\x00" + strings.TrimSpace(candidate.object.Path)
		file, ok := files[fileKey]
		if !ok {
			return fmt.Errorf("discovered inventory file missing after bulk upsert for %s", candidate.object.Path)
		}
		result.InventoryFilesSeen++
		batchFiles = append(batchFiles, file)
	}
	materializeResult, err := s.runRecognitionMaterializeBatch(ctx, library, rootPathForScanMode(library.RootPath, mode), batchFiles, mode)
	if err != nil {
		return err
	}
	result.MetadataItemsSeen += len(materializeResult.MetadataIDs)
	if mode != nil {
		mode.markDirectoryResolved(snapshot.Path)
	}
	_ = exclusionRules
	_ = scanPolicy
	_ = subtitlePolicy
	_ = seenFiles
	return nil
}

func (s *Service) shouldDelaySiblingMovieMaterialization(ctx context.Context, mode *scanMode, library database.Library, file database.InventoryFile, object storage.Object) bool {
	if mode == nil || mode.deferRecognitionResolution || mode.partial || file.ID == 0 || strings.TrimSpace(file.StoragePath) == "" {
		return false
	}
	childDir := path.Dir(file.StoragePath)
	parentDir := path.Dir(childDir)
	if childDir == strings.TrimSpace(library.RootPath) || parentDir == "." || parentDir == childDir {
		return false
	}
	if rules, err := loadPathTreeClassificationRules(ctx, s.db, library.ID); err == nil && pathTreeFileMatchesAnyRule(file.StoragePath, rules) {
		return true
	}
	if parseSeasonDirectoryNumber(path.Base(childDir)) != nil && strings.TrimSpace(seriesTitleFromEmbeddedSeasonDirectory(path.Base(childDir))) != "" {
		return true
	}
	if mode.pendingSiblingMovieFiles != nil {
		if pending := mode.pendingSiblingMovieFiles[parentDir]; len(pending) > 0 {
			return true
		}
	}
	settings := contentShapeSettingsFromConfig(s.cfg)
	scope := inventoryFileSignalScope{LibraryID: file.LibraryID, StorageProvider: strings.TrimSpace(file.StorageProvider), ClassifierVersion: settings.ClassifierVersion}
	models, _, err := loadReusableInventoryFileSignals(ctx, s.db, scope, []database.InventoryFile{file})
	if err != nil {
		return false
	}
	assignments := compilePathTreeAssignmentsFromFiles([]database.InventoryFile{file}, library.RootPath, models, newFilenameTokenProfileCache())
	_ = assignments
	signal := models[strings.TrimSpace(file.StoragePath)]
	if strings.TrimSpace(signal.Identity.TitleCandidate) == "" {
		signal = extractFilenameSignalModel(object.Path)
	}
	workKey := normalizedMovieWorkKeyFromSignal(signal)
	return strings.TrimSpace(workKey.Normalized) != "" && workKey.Year != nil && signal.Identity.EpisodeNumber == nil && len(signal.Identity.EpisodeNumbers) == 0 && !signal.RoleHints.IsExtra
}

func (m *scanMode) recordPendingSiblingMovieFile(pending pendingSiblingMovieFile) {
	if m == nil || strings.TrimSpace(pending.Object.Path) == "" {
		return
	}
	parentDir := path.Dir(path.Dir(pending.Object.Path))
	if parentDir == "." {
		return
	}
	if m.pendingSiblingMovieFiles == nil {
		m.pendingSiblingMovieFiles = make(map[string][]pendingSiblingMovieFile)
	}
	m.pendingSiblingMovieFiles[parentDir] = append(m.pendingSiblingMovieFiles[parentDir], pending)
}

func (s *Service) flushPendingSiblingMovieFiles(ctx context.Context, mode *scanMode, parentPath string, seenFiles map[string]struct{}, result *SyncResult) error {
	_ = ctx
	_ = mode
	_ = parentPath
	_ = seenFiles
	_ = result
	return nil
}

func (s *Service) flushRecognitionResolveCandidates(ctx context.Context, libraryID uint, rootPath string, mode *scanMode, minBatchSize int) error {
	if mode == nil || !mode.deferRecognitionResolution || len(mode.recognitionResolveFileIDs) == 0 {
		return nil
	}
	_ = ctx
	_ = libraryID
	_ = rootPath
	_ = minBatchSize
	return nil
	/*
		if minBatchSize > 0 && len(mode.recognitionResolveFileIDs) < minBatchSize {
			return nil
		}
		fileIDs := append([]uint(nil), mode.recognitionResolveFileIDs...)
		mode.recognitionResolveFileIDs = nil
		var library database.Library
		if err := s.db.WithContext(ctx).First(&library, libraryID).Error; err != nil {
			return err
		}
		if strings.TrimSpace(rootPath) != "" {
			library.RootPath = strings.TrimSpace(rootPath)
		}
		_, err := s.runRecognitionMaterializeBatchByFileIDs(ctx, library, rootPath, fileIDs, mode)
		return err
	*/
}

func rootPathForScanMode(defaultRoot string, mode *scanMode) string {
	if mode != nil && strings.TrimSpace(mode.rootPath) != "" {
		return mode.rootPath
	}
	return defaultRoot
}

func (s *Service) upsertNonVideoInventoryFile(ctx context.Context, provider storage.Provider, library database.Library, object storage.Object, className string) error {
	container := strings.TrimPrefix(strings.ToLower(path.Ext(object.Path)), ".")
	_, err := inventory.NewService(s.db).UpsertFile(ctx, inventory.UpsertFileInput{
		LibraryID:         library.ID,
		StorageProvider:   provider.Name(),
		StoragePath:       object.Path,
		StableIdentityKey: strings.TrimSpace(object.StableIdentity),
		HashesJSON:        encodeHashInfo(object.HashInfo),
		SizeBytes:         object.Size,
		ModifiedAt:        object.Modified,
		Container:         container,
		ContentClass:      className,
		Status:            inventory.FileStatusAvailable,
	})
	return err
}

func (s *Service) upsertDiscoveredInventoryFile(ctx context.Context, provider storage.Provider, library database.Library, object storage.Object) (database.InventoryFile, error) {
	container := strings.TrimPrefix(strings.ToLower(path.Ext(object.Path)), ".")
	if file, ok, err := reuseInventoryFileByStableIdentity(ctx, s.db, inventoryFileStableIdentityReuseInput{
		LibraryID:         library.ID,
		StorageProvider:   provider.Name(),
		StableIdentityKey: strings.TrimSpace(object.StableIdentity),
		StoragePath:       object.Path,
		HashesJSON:        encodeHashInfo(object.HashInfo),
		ThumbnailURL:      strings.TrimSpace(object.ThumbnailURL),
		SizeBytes:         object.Size,
		ModifiedAt:        object.Modified,
		Container:         container,
		ContentClass:      SourceContentClassVideo,
		Status:            inventory.FileStatusAvailable,
		ScanState:         inventory.FileScanStateDiscovered,
	}); err != nil {
		return database.InventoryFile{}, err
	} else if ok {
		s.markInventoryFileDirty(ctx, file.ID, "scanner_refresh")
		return file, nil
	}
	file, err := inventory.NewService(s.db).UpsertFile(ctx, inventory.UpsertFileInput{
		LibraryID:         library.ID,
		StorageProvider:   provider.Name(),
		StoragePath:       object.Path,
		StableIdentityKey: strings.TrimSpace(object.StableIdentity),
		HashesJSON:        encodeHashInfo(object.HashInfo),
		SizeBytes:         object.Size,
		ModifiedAt:        object.Modified,
		Container:         container,
		ContentClass:      SourceContentClassVideo,
		Status:            inventory.FileStatusAvailable,
		ScanState:         inventory.FileScanStateDiscovered,
	})
	if err != nil {
		return database.InventoryFile{}, err
	}
	if err := s.ensureInventoryFileSignals(ctx, library.ID, provider.Name(), []database.InventoryFile{file}); err != nil {
		return database.InventoryFile{}, err
	}
	s.markInventoryFileDirty(ctx, file.ID, "scanner_discovery")
	return file, nil
}

func (s *Service) bulkUpsertDiscoveredInventoryFiles(ctx context.Context, provider storage.Provider, library database.Library, candidates []discoveredInventoryCandidate) (map[string]database.InventoryFile, error) {
	inputs := make([]inventory.UpsertFileInput, 0, len(candidates))
	result := make(map[string]database.InventoryFile, len(candidates))
	dirtyFileIDs := make([]uint, 0, len(candidates))
	events := make([]database.IngestEvent, 0, len(candidates))
	reuseUpdates := make([]discoveredStableIdentityUpdate, 0, len(candidates))
	reusedByStableIdentity, err := s.preloadDiscoveredInventoryFilesByStableIdentity(ctx, library.ID, provider.Name(), candidates)
	if err != nil {
		return nil, err
	}
	for _, candidate := range candidates {
		object := candidate.object
		if binding, ok := reusedByStableIdentity[strings.TrimSpace(object.StableIdentity)]; ok && binding.file.ID != 0 {
			reuseUpdates = append(reuseUpdates, discoveredStableIdentityUpdate{file: binding.file, container: binding.container, hashJSON: binding.hashJSON, object: binding.object, resultKey: provider.Name() + "\x00" + object.Path})
			continue
		}
		inputs = append(inputs, inventory.UpsertFileInput{LibraryID: library.ID, StorageProvider: provider.Name(), StoragePath: object.Path, StableIdentityKey: strings.TrimSpace(object.StableIdentity), HashesJSON: encodeHashInfo(object.HashInfo), ThumbnailURL: strings.TrimSpace(object.ThumbnailURL), SizeBytes: object.Size, ModifiedAt: object.Modified, Container: candidate.container, ContentClass: SourceContentClassVideo, Status: inventory.FileStatusAvailable, ScanState: inventory.FileScanStateDiscovered})
	}
	if len(inputs) > 0 {
		bulkResult, err := inventory.NewService(s.db).BulkUpsertFiles(ctx, inputs)
		if err != nil {
			return nil, err
		}
		for key, file := range bulkResult.FilesByStoragePath {
			result[key] = file
			dirtyFileIDs = append(dirtyFileIDs, file.ID)
			fileID := file.ID
			events = append(events, database.IngestEvent{UnitKey: inventoryFileUnitKey(file.ID), LibraryID: library.ID, InventoryFileID: &fileID, ConditionType: ingest.ConditionVisible, EventType: ingest.EventConditionChanged, Status: ingest.ConditionStatusPending, Reason: "scanner_discovery", Message: "Inventory file discovered during library scan"})
		}
	}
	if len(reuseUpdates) > 0 {
		reusedFiles, err := s.applyDiscoveredReuseBindings(ctx, reuseUpdates)
		if err != nil {
			return nil, err
		}
		for resultKey, file := range reusedFiles {
			result[resultKey] = file
			dirtyFileIDs = append(dirtyFileIDs, file.ID)
		}
	}
	ingestSvc := s.ingestCapability()
	if len(dirtyFileIDs) > 0 && ingestSvc != nil {
		if err := ingestSvc.MarkInventoryFilesDirty(ctx, dirtyFileIDs, "scanner_discovery"); err != nil {
			return nil, err
		}
	}
	if len(events) > 0 && ingestSvc != nil {
		if err := ingestSvc.AppendEvents(ctx, events); err != nil {
			return nil, err
		}
	}
	filesForSignals := make([]database.InventoryFile, 0, len(result))
	for _, file := range result {
		filesForSignals = append(filesForSignals, file)
	}
	if err := s.ensureInventoryFileSignals(ctx, library.ID, provider.Name(), filesForSignals); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) ensureInventoryFileSignals(ctx context.Context, libraryID uint, storageProvider string, files []database.InventoryFile) error {
	if len(files) == 0 {
		return nil
	}
	settings := contentShapeSettingsFromConfig(s.cfg)
	scope := inventoryFileSignalScope{LibraryID: libraryID, StorageProvider: storageProvider, ClassifierVersion: settings.ClassifierVersion}
	models, _, err := loadReusableInventoryFileSignals(ctx, s.db, scope, files)
	if err != nil {
		return err
	}
	missing := make([]inventoryFileSignalInput, 0)
	for _, file := range files {
		if file.ID == 0 || file.Status != inventory.FileStatusAvailable || file.ContentClass != SourceContentClassVideo || !isVideoFile(file.StoragePath) {
			continue
		}
		if _, ok := models[strings.TrimSpace(file.StoragePath)]; ok {
			continue
		}
		missing = append(missing, inventoryFileSignalInput{File: file, Model: extractFilenameSignalModel(file.StoragePath)})
	}
	return saveInventoryFileSignals(ctx, s.db, scope, missing)
}

func (s *Service) preloadDiscoveredInventoryFilesByStableIdentity(ctx context.Context, libraryID uint, storageProvider string, candidates []discoveredInventoryCandidate) (map[string]discoveredStableIdentityBinding, error) {
	stableIdentityKeys := make([]string, 0, len(candidates))
	bindings := make(map[string]discoveredStableIdentityBinding)
	for _, candidate := range candidates {
		key := strings.TrimSpace(candidate.object.StableIdentity)
		if key == "" {
			continue
		}
		if _, ok := bindings[key]; ok {
			continue
		}
		stableIdentityKeys = append(stableIdentityKeys, key)
		bindings[key] = discoveredStableIdentityBinding{container: candidate.container, hashJSON: encodeHashInfo(candidate.object.HashInfo), object: candidate.object}
	}
	if len(stableIdentityKeys) == 0 {
		return bindings, nil
	}
	files, err := loadInventoryFilesByStableIdentity(ctx, s.db, libraryID, storageProvider, stableIdentityKeys)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		key := strings.TrimSpace(file.StableIdentityKey)
		binding := bindings[key]
		binding.file = file
		bindings[key] = binding
	}
	return bindings, nil
}

func (s *Service) applyDiscoveredReuseBindings(ctx context.Context, updates []discoveredStableIdentityUpdate) (map[string]database.InventoryFile, error) {
	if len(updates) == 0 {
		return nil, nil
	}
	ids := make([]uint, 0, len(updates))
	resultKeyByID := make(map[uint]string, len(updates))
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, update := range updates {
			if update.file.ID == 0 {
				continue
			}
			ids = append(ids, update.file.ID)
			resultKeyByID[update.file.ID] = update.resultKey
			if err := applyInventoryFileStableIdentityReuseUpdate(ctx, tx, update.file.ID, inventoryFileStableIdentityReuseInput{
				LibraryID:         update.file.LibraryID,
				StorageProvider:   update.file.StorageProvider,
				StableIdentityKey: update.file.StableIdentityKey,
				StoragePath:       update.object.Path,
				HashesJSON:        update.hashJSON,
				ThumbnailURL:      strings.TrimSpace(update.object.ThumbnailURL),
				SizeBytes:         update.object.Size,
				ModifiedAt:        update.object.Modified,
				Container:         update.container,
				ContentClass:      SourceContentClassVideo,
				Status:            inventory.FileStatusAvailable,
				ScanState:         inventory.FileScanStateDiscovered,
			}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	var files []database.InventoryFile
	if len(ids) == 0 {
		return map[string]database.InventoryFile{}, nil
	}
	for _, batch := range chunkUints(ids, sqliteVariableChunkSize) {
		var partial []database.InventoryFile
		if err := s.db.WithContext(ctx).Where("id IN ?", batch).Find(&partial).Error; err != nil {
			return nil, err
		}
		files = append(files, partial...)
	}
	result := make(map[string]database.InventoryFile, len(files))
	for _, file := range files {
		result[resultKeyByID[file.ID]] = file
	}
	return result, nil
}

func (s *Service) queuePostScanEnrichment(ctx context.Context, libraryID uint, rootPath string, mode scanMode, scanPolicy database.LibraryScanPolicy) error {
	if len(mode.recognitionResolveFileIDs) > 0 {
		return s.queueStandaloneRecognitionResolveTasks(ctx, libraryID, rootPath, mode.recognitionResolveFileIDs)
	}
	if _, err := s.QueueMetadataMatchBatch(ctx, libraryID, rootPath, mode.metadataMatchItemIDs); err != nil {
		return err
	}
	effectiveConfig := EffectiveLibraryConfig{ScanPolicy: scanPolicy}
	if !effectiveConfig.InventoryProbeBatchEnabled() {
		return nil
	}
	if _, err := s.QueueInventoryProbeBatch(ctx, libraryID, rootPath, mode.inventoryProbeFileIDs); err != nil {
		return err
	}
	if _, err := s.QueueInventoryProbeBatch(ctx, libraryID, rootPath, mode.classificationFileIDs); err != nil {
		return err
	}
	return nil
}

func catalogScanArtifactNeedsClassificationValidation(artifact catalogScanArtifact) bool {
	for _, decision := range artifact.Decisions {
		if decision.Status == scanDecisionStatusProvisional || decision.Status == scanDecisionStatusReviewRequired {
			return true
		}
	}
	return false
}

func (s *Service) filteredScanSnapshot(ctx context.Context, provider storage.Provider, library database.Library, snapshot scanDirectorySnapshot, exclusionRules []database.ScanExclusionRule, scanPolicy database.LibraryScanPolicy) (scanDirectorySnapshot, error) {
	filtered := snapshot
	filtered.Objects = make([]storage.Object, 0, len(snapshot.Objects))
	for _, object := range snapshot.Objects {
		if err := ctx.Err(); err != nil {
			return scanDirectorySnapshot{}, err
		}
		if object.IsDir || !isVideoFile(object.Path) {
			filtered.Objects = append(filtered.Objects, object)
			continue
		}
		exclusion, err := s.scanExclusionDecisionWithRules(ctx, library, provider.Name(), object, exclusionRules)
		if err != nil {
			return scanDirectorySnapshot{}, err
		}
		if exclusion.Excluded || scanPolicySkipReason(object, scanPolicy) != "" {
			continue
		}
		filtered.Objects = append(filtered.Objects, object)
	}
	return filtered, nil
}

func (r *SyncResult) recordExcludedFileSkipped(reason string) {
	if r == nil {
		return
	}
	r.ExcludedFilesSkipped++
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		trimmed = "unknown"
	}
	if r.ExcludedFilesSkippedByReason == nil {
		r.ExcludedFilesSkippedByReason = make(map[string]int)
	}
	r.ExcludedFilesSkippedByReason[trimmed]++
}

func shouldSkipByScanPolicy(object storage.Object, policy database.LibraryScanPolicy) bool {
	if !policy.IgnoreHiddenFiles {
		return false
	}
	return strings.HasPrefix(path.Base(object.Path), ".")
}

func scanPolicySkipReason(object storage.Object, policy database.LibraryScanPolicy) string {
	if shouldSkipByScanPolicy(object, policy) {
		return "policy_hidden"
	}
	ext := strings.ToLower(path.Ext(object.Path))
	for _, ignored := range stringListFromJSON(policy.IgnoreFileExtensionsJSON) {
		if strings.ToLower(strings.TrimSpace(ignored)) == ext {
			return "policy_extension"
		}
	}
	if policy.MinFileSizeBytes > 0 && object.Size >= 0 && object.Size < policy.MinFileSizeBytes {
		return "policy_min_size"
	}
	if policy.SampleIgnoreSizeBytes > 0 && object.Size > 0 && object.Size <= policy.SampleIgnoreSizeBytes && hasAdvertisementToken(object.Path) {
		return "policy_sample"
	}
	return ""
}

func artifactAllowsFolderMetadata(dirPath string, artifact catalogScanArtifact) bool {
	if artifact.ItemType == catalog.ItemTypeEpisode {
		return strings.TrimSpace(artifact.SeriesPath) != "" && strings.TrimSpace(artifact.SeasonPath) != ""
	}
	return artifact.ItemType == catalog.ItemTypeMovie && strings.TrimSpace(artifact.ItemPath) == strings.TrimSpace(dirPath)
}

func (s *Service) collectDirectorySnapshot(ctx context.Context, provider storage.Provider, dirPath string, refresh bool) (scanDirectorySnapshot, error) {
	objects, err := s.listAllDirectoryObjects(ctx, provider, dirPath, refresh)
	if err != nil {
		return scanDirectorySnapshot{}, fmt.Errorf("list directory %s: %w", dirPath, err)
	}
	return scanDirectorySnapshot{Path: dirPath, Objects: objects, Sidecars: buildSidecarIndex(provider.Name(), objects)}, nil
}

func scanDecisionFromAttachmentRole(sourcePath string, artifact catalogScanArtifact) scanDecision {
	extraType := videoFileRoleSignal(sourcePath)
	if strings.TrimSpace(extraType) == "" {
		return scanDecision{}
	}
	confidence := 0.9
	role := scanDecisionRoleExtra
	switch extraType {
	case "trailer":
		role = scanDecisionRoleTrailer
	case "sample":
		role = scanDecisionRoleSample
	}
	return scanDecision{
		Type:          scanDecisionAssetLink,
		TargetKind:    artifact.ItemType,
		TargetKey:     artifact.ItemPath,
		Role:          role,
		CandidateType: scanDecisionCandidateAttachment,
		Status:        scanDecisionStatusConfirmedFast,
		Confidence:    &confidence,
		Evidence: []scanDecisionEvidence{{
			Kind:   "filename_role",
			Source: "path",
			Value:  extraType,
		}},
		Reason:    "video filename or path indicates an attachment role",
		CreatedAt: time.Now().UTC(),
	}
}

func catalogScanArtifactFromObject(storageProvider string, libraryType string, libraryRoot string, object storage.Object, classified classifiedMedia) (catalogScanArtifact, []string) {
	artifact := catalogScanArtifact{
		SourcePath:           object.Path,
		Title:                classified.Title,
		OriginalTitle:        classified.OriginalTitle,
		SeriesTitle:          classified.SeriesTitle,
		Year:                 classified.Year,
		Tags:                 append([]string(nil), classified.Tags...),
		SeasonNumber:         classified.SeasonNumber,
		StorageProvider:      strings.TrimSpace(storageProvider),
		StableIdentityKey:    strings.TrimSpace(object.StableIdentity),
		ProviderName:         strings.TrimSpace(object.Provider),
		HashesJSON:           encodeHashInfo(object.HashInfo),
		ThumbnailURL:         strings.TrimSpace(object.ThumbnailURL),
		ObjectType:           strings.TrimSpace(object.ObjectType),
		ProviderMeta:         object.SanitizedProviderMeta(),
		SizeBytes:            object.Size,
		ModifiedAt:           object.Modified,
		Container:            strings.TrimPrefix(strings.ToLower(path.Ext(object.Path)), "."),
		NormalizationVersion: classified.NormalizationVersion,
		RemovedTokens:        append([]titleclean.RemovedToken(nil), classified.RemovedTokens...),
		FilenameSignals:      classified.FilenameSignals,
	}

	if classified.Type == "episode" {
		if linkRole := movieExtraLinkRole(classified.SourcePath); strings.TrimSpace(linkRole) != "" {
			artifact.ItemType = catalog.ItemTypeMovie
			artifact.ItemPath = movieCatalogItemPath(libraryType, libraryRoot, classified.SourcePath, classified.Title)
			artifact.PreferredLinkRole = linkRole
			return artifact, catalogScanItemPaths(artifact)
		}
		artifact.ItemType = catalog.ItemTypeEpisode
		artifact.SeriesPath = canonicalSeriesPath(classified.SeriesTitle)
		if classified.SeasonNumber != nil {
			artifact.SeasonPath = fmt.Sprintf("%s/season-%02d", artifact.SeriesPath, *classified.SeasonNumber)
		}
		episodeNumbers := append([]int(nil), classified.EpisodeNumbers...)
		if len(episodeNumbers) == 0 && classified.EpisodeNumber != nil {
			episodeNumbers = append(episodeNumbers, *classified.EpisodeNumber)
		}
		for _, episodeNumber := range episodeNumbers {
			itemPath := canonicalEpisodeItemPath(artifact.SeasonPath, episodeNumber)
			artifact.EpisodeSlots = append(artifact.EpisodeSlots, catalogEpisodeSlot{EpisodeNumber: episodeNumber, ItemPath: itemPath})
		}
		return artifact, catalogScanItemPaths(artifact)
	}

	artifact.ItemType = catalog.ItemTypeMovie
	artifact.ItemPath = movieCatalogItemPath(libraryType, libraryRoot, classified.SourcePath, classified.Title)
	artifact.PreferredLinkRole = movieExtraLinkRole(classified.SourcePath)
	return artifact, catalogScanItemPaths(artifact)
}

func movieCatalogItemPath(libraryType string, libraryRoot string, sourcePath string, title string) string {
	if !isMovieLibraryType(libraryType) && !isMixedLibraryType(libraryType) {
		return sourcePath
	}
	segments := relativePathSegments(libraryRoot, sourcePath)
	parentTitle := cleanTitle(path.Base(path.Dir(sourcePath)))
	extraType := extraTypeSignal(strings.TrimSuffix(path.Base(sourcePath), path.Ext(sourcePath)))
	if len(segments) >= 2 && (strings.EqualFold(strings.TrimSpace(parentTitle), strings.TrimSpace(title)) || extraType != "") {
		return path.Dir(sourcePath)
	}
	return sourcePath
}

func movieExtraLinkRole(sourcePath string) string {
	switch extraTypeSignal(strings.TrimSuffix(path.Base(sourcePath), path.Ext(sourcePath))) {
	case "trailer":
		return database.ResourceLinkRoleTrailer
	case "sample":
		return database.ResourceLinkRoleSample
	case "behind_the_scenes", "featurette", "preview", "interview", "deleted_scene":
		return database.ResourceLinkRoleExtra
	default:
		return ""
	}
}

func catalogScanItemPaths(artifact catalogScanArtifact) []string {
	if artifact.ItemType == catalog.ItemTypeEpisode {
		itemPaths := make([]string, 0, len(artifact.EpisodeSlots)+2)
		if artifact.SeriesPath != "" {
			itemPaths = append(itemPaths, artifact.SeriesPath)
		}
		if artifact.SeasonPath != "" {
			itemPaths = append(itemPaths, artifact.SeasonPath)
		}
		for _, slot := range artifact.EpisodeSlots {
			if slot.ItemPath != "" {
				itemPaths = append(itemPaths, slot.ItemPath)
			}
		}
		return itemPaths
	}
	return []string{artifact.ItemPath}
}

func encodeHashInfo(hashInfo map[string]string) string {
	if len(hashInfo) == 0 {
		return ""
	}
	encoded, err := json.Marshal(hashInfo)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func (s *Service) listAllDirectoryObjects(ctx context.Context, provider storage.Provider, dirPath string, refresh bool) ([]storage.Object, error) {
	const pageSize = 1000
	var all []storage.Object
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		objects, err := provider.List(ctx, storage.ListRequest{Path: dirPath, Refresh: refresh && page == 1, Page: page, PerPage: pageSize})
		if err != nil {
			return nil, err
		}
		all = append(all, objects...)
		if len(objects) < pageSize {
			break
		}
	}
	return all, nil
}
