package library

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/storage"
	"github.com/atlan/mibo-media-server/internal/titleclean"
)

func (s *Service) QueueLibraryScan(ctx context.Context, libraryID uint) (database.Job, error) {
	var record database.Library
	if err := s.db.WithContext(ctx).First(&record, libraryID).Error; err != nil {
		return database.Job{}, err
	}
	return s.jobs.EnqueueUnique(ctx, JobKindSyncLibrary, fmt.Sprintf("scan-library-%d", record.ID), map[string]any{"library_id": record.ID})
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
		scanMode := scanMode{}
		pathResult, err := s.scanLibraryWithMode(ctx, provider, libraryForPath, pathRecord.RootPath, &scanMode)
		if err != nil {
			_ = s.updateLibraryStatus(ctx, config.Library.ID, "error")
			return err
		}
		result.add(pathResult)
		if _, err := s.QueueCatalogLibraryProjectionRefresh(ctx, config.Library.ID, pathRecord.RootPath); err != nil {
			return err
		}
		if err := s.queuePostScanEnrichment(ctx, config.Library.ID, pathRecord.RootPath, scanMode); err != nil {
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
	if err := s.updateLibraryStatus(ctx, config.Library.ID, "syncing"); err != nil {
		return err
	}
	libraryForPath := config.Library
	libraryForPath.MediaSourceID = pathRecord.MediaSourceID
	libraryForPath.RootPath = pathRecord.RootPath
	scanMode := scanMode{partial: true, rootPath: rootPath}
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
	if err := s.queuePostScanEnrichment(ctx, config.Library.ID, rootPath, scanMode); err != nil {
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
	r.CatalogItemsSeen += other.CatalogItemsSeen
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

func (s *Service) scanLibrary(ctx context.Context, provider storage.Provider, library database.Library, rootPath string) (SyncResult, error) {
	return s.scanLibraryWithMode(ctx, provider, library, rootPath, &scanMode{})
}

func (s *Service) scanLibraryWithMode(ctx context.Context, provider storage.Provider, library database.Library, rootPath string, mode *scanMode) (SyncResult, error) {
	if err := ctx.Err(); err != nil {
		return SyncResult{}, err
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
	if !scanPolicy.ScannerEnabled {
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
	if err := s.walkDirectory(ctx, provider, library, rootPath, seenFiles, seenItems, &result, exclusionRules, scanPolicy, subtitlePolicy, mode); err != nil {
		return SyncResult{}, err
	}
	if err := s.cleanupMissingCatalog(ctx, library.ID, rootPath, seenFiles); err != nil {
		return SyncResult{}, err
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
	snapshot, err := s.collectDirectorySnapshot(ctx, provider, dirPath)
	if err != nil {
		return err
	}
	objects := snapshot.Objects
	sidecars := snapshot.Sidecars
	decisionSnapshot, err := s.directoryShapeSnapshot(ctx, provider, library, snapshot, exclusionRules, scanPolicy)
	if err != nil {
		return err
	}
	directoryDecision := resolveDirectoryShape(library.Type, library.RootPath, decisionSnapshot)
	sort.Slice(objects, func(i, j int) bool { return objects[i].Path < objects[j].Path })
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
				return err
			}
			continue
		}
		if !isVideoFile(object.Path) {
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
		if shouldSkipTVDirectoryExtra(library.Type, directoryDecision, object) {
			continue
		}
		classified := classifyMediaFileWithDirectoryDecision(library.Type, library.RootPath, object, snapshot.Path, directoryDecision)
		artifact, itemPaths := catalogScanArtifactFromObject(provider.Name(), library.Type, library.RootPath, object, classified)
		if decision := scanDecisionFromDirectoryShape(directoryDecision, artifact); strings.TrimSpace(decision.Type) != "" {
			artifact.Decisions = append(artifact.Decisions, decision)
		}
		artifact = s.applyCatalogScanSidecars(ctx, provider, artifact, sidecars.matchesForVideoWithFolderMetadata(object.Path, artifactAllowsFolderMetadata(snapshot.Path, artifact)), subtitlePolicy)
		artifact = applyCatalogScanArtworkCandidates(provider, artifact, object, snapshot)
		for _, sidecar := range artifact.SubtitleSidecars {
			if strings.TrimSpace(sidecar.Path) != "" {
				seenFiles[sidecar.Path] = struct{}{}
				result.InventoryFilesSeen++
			}
		}
		itemPaths = catalogScanItemPaths(artifact)
		for _, itemPath := range itemPaths {
			seenItems[itemPath] = struct{}{}
		}
		writeResult, err := s.writeCatalogScan(ctx, library, artifact)
		if err != nil {
			return err
		}
		if writeResult.File.ID != 0 {
			result.InventoryFilesSeen++
			if writeResult.Item.ID != 0 {
				result.CatalogItemsSeen++
			}
		}
		if writeResult.Item.ID != 0 {
			mode.recordCatalogMatchCandidate(writeResult.Item.ID)
		}
		if writeResult.File.ID != 0 {
			mode.recordInventoryProbeCandidate(writeResult.File.ID)
		}
	}
	return nil
}

func (s *Service) queuePostScanEnrichment(ctx context.Context, libraryID uint, rootPath string, mode scanMode) error {
	if _, err := s.QueueCatalogMatchBatch(ctx, libraryID, rootPath, mode.catalogMatchItemIDs); err != nil {
		return err
	}
	if _, err := s.QueueInventoryProbeBatch(ctx, libraryID, rootPath, mode.inventoryProbeFileIDs); err != nil {
		return err
	}
	return nil
}

func (s *Service) directoryShapeSnapshot(ctx context.Context, provider storage.Provider, library database.Library, snapshot scanDirectorySnapshot, exclusionRules []database.ScanExclusionRule, scanPolicy database.LibraryScanPolicy) (scanDirectorySnapshot, error) {
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

func shouldSkipTVDirectoryExtra(libraryType string, decision directoryShapeDecision, object storage.Object) bool {
	if (!isTVLibraryType(libraryType) && !isMixedLibraryType(libraryType)) || decision.Shape != directoryShapeFlatEpisodeFolder {
		return false
	}
	return extraTypeSignal(strings.TrimSuffix(path.Base(object.Path), path.Ext(object.Path))) != ""
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

func (s *Service) collectDirectorySnapshot(ctx context.Context, provider storage.Provider, dirPath string) (scanDirectorySnapshot, error) {
	objects, err := s.listAllDirectoryObjects(ctx, provider, dirPath)
	if err != nil {
		return scanDirectorySnapshot{}, fmt.Errorf("list directory %s: %w", dirPath, err)
	}
	return scanDirectorySnapshot{Path: dirPath, Objects: objects, Sidecars: buildSidecarIndex(provider.Name(), objects)}, nil
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
		ObjectType:           strings.TrimSpace(object.ObjectType),
		ProviderMeta:         object.SanitizedProviderMeta(),
		SizeBytes:            object.Size,
		ModifiedAt:           object.Modified,
		Container:            strings.TrimPrefix(strings.ToLower(path.Ext(object.Path)), "."),
		NormalizationVersion: classified.NormalizationVersion,
		RemovedTokens:        append([]titleclean.RemovedToken(nil), classified.RemovedTokens...),
	}

	if classified.Type == "episode" {
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
	artifact.PreferredAssetType, artifact.PreferredAssetRole = movieExtraAssetDisposition(classified.SourcePath)
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

func movieExtraAssetDisposition(sourcePath string) (string, string) {
	switch extraTypeSignal(strings.TrimSuffix(path.Base(sourcePath), path.Ext(sourcePath))) {
	case "trailer":
		return inventory.AssetTypeTrailer, inventory.AssetItemRoleTrailer
	case "sample":
		return inventory.AssetTypeSample, inventory.AssetItemRoleExtra
	case "behind_the_scenes", "featurette", "interview", "deleted_scene":
		return inventory.AssetTypeExtra, inventory.AssetItemRoleExtra
	default:
		return "", ""
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

func (s *Service) listAllDirectoryObjects(ctx context.Context, provider storage.Provider, dirPath string) ([]storage.Object, error) {
	const pageSize = 1000
	var all []storage.Object
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		objects, err := provider.List(ctx, storage.ListRequest{Path: dirPath, Refresh: true, Page: page, PerPage: pageSize})
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
