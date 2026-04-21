package library

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

var (
	episodePattern = regexp.MustCompile(`(?i)^(.*?)[\s._-]+(?:s(\d{1,2})e(\d{1,2})|(\d{1,2})x(\d{1,2}))(?:[\s._-]+.*)?$`)
	yearPattern    = regexp.MustCompile(`(?i)(?:^|[\s._\-(])((?:19|20)\d{2})(?:$|[\s._\-)])`)
)

var videoExtensions = map[string]struct{}{
	".mp4":  {},
	".mkv":  {},
	".avi":  {},
	".mov":  {},
	".wmv":  {},
	".m4v":  {},
	".ts":   {},
	".m2ts": {},
	".webm": {},
}

const (
	mediaFileIdentitySourceNone             = "none"
	mediaFileIdentitySourceStableIdentity   = "stable_identity"
	mediaFileIdentitySourceProviderEvidence = "provider_evidence"

	mediaFileIdentityStatusExact       = "exact"
	mediaFileIdentityStatusProvisional = "provisional"
	mediaFileIdentityStatusReconciled  = "fallback_reconciled"

	mediaFileReviewStatusNone    = "none"
	mediaFileReviewStatusPending = "pending"
	mediaFileReviewStatusNeeded  = "review_needed"

	fallbackDurationToleranceSeconds = 2.0
)

type classifiedMedia struct {
	Type          string
	Title         string
	OriginalTitle string
	SeriesTitle   string
	Year          *int
	SeasonNumber  *int
	EpisodeNumber *int
	SourcePath    string
	Status        string
}

type SyncResult struct {
	DirectoriesScanned int `json:"directories_scanned"`
	FilesSeen          int `json:"files_seen"`
	MediaItemsUpserted int `json:"media_items_upserted"`
	MediaFilesUpserted int `json:"media_files_upserted"`
}

type scanMode struct {
	partial  bool
	rootPath string
}

func (s *Service) QueueLibraryScan(ctx context.Context, libraryID uint) (database.Job, error) {
	var record database.Library
	if err := s.db.WithContext(ctx).First(&record, libraryID).Error; err != nil {
		return database.Job{}, err
	}

	return s.jobs.EnqueueUnique(ctx, JobKindSyncLibrary, fmt.Sprintf("scan-library-%d", record.ID), map[string]any{
		"library_id": record.ID,
		"root_path":  record.RootPath,
	})
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

	record, _, provider, err := s.providerForLibrary(ctx, payload.LibraryID)
	if err != nil {
		return err
	}

	rootPath := record.RootPath
	if payload.RootPath != "" {
		rootPath = normalizePath(payload.RootPath)
	}

	if err := s.updateLibraryStatus(ctx, record.ID, "syncing"); err != nil {
		return err
	}

	result, err := s.scanLibrary(ctx, provider, record, rootPath)
	if err != nil {
		_ = s.updateLibraryStatus(ctx, record.ID, "error")
		return err
	}

	if err := s.updateLibraryStatus(ctx, record.ID, "active"); err != nil {
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

	record, _, provider, err := s.providerForLibrary(ctx, payload.LibraryID)
	if err != nil {
		return err
	}

	rootPath, err := scopedRefreshRoot(provider.Name(), record.RootPath, payload.RootPath)
	if err != nil {
		return err
	}

	if err := s.updateLibraryStatus(ctx, record.ID, "syncing"); err != nil {
		return err
	}

	result, err := s.scanLibraryWithMode(ctx, provider, record, rootPath, scanMode{partial: true, rootPath: rootPath})
	if err != nil {
		_ = s.updateLibraryStatus(ctx, record.ID, "error")
		return err
	}

	if err := s.updateLibraryStatus(ctx, record.ID, "active"); err != nil {
		return err
	}

	_ = result
	return nil
}

func (s *Service) scanLibrary(ctx context.Context, provider storage.Provider, library database.Library, rootPath string) (SyncResult, error) {
	return s.scanLibraryWithMode(ctx, provider, library, rootPath, scanMode{})
}

func (s *Service) scanLibraryWithMode(ctx context.Context, provider storage.Provider, library database.Library, rootPath string, mode scanMode) (SyncResult, error) {
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

	if err := s.walkDirectory(ctx, provider, library, rootPath, seenFiles, seenItems, &result); err != nil {
		return SyncResult{}, err
	}

	if mode.partial {
		// T-06-09: partial refreshes only reconcile missing rows inside the targeted subtree.
		if err := s.cleanupMissingFilesInScope(ctx, library.ID, mode.rootPath, seenFiles); err != nil {
			return SyncResult{}, err
		}
		if err := s.cleanupMissingItemsInScope(ctx, library.ID, mode.rootPath, seenItems); err != nil {
			return SyncResult{}, err
		}
	} else {
		if err := s.cleanupMissingFiles(ctx, library.ID, seenFiles); err != nil {
			return SyncResult{}, err
		}
		if err := s.cleanupMissingItems(ctx, library.ID, seenItems); err != nil {
			return SyncResult{}, err
		}
	}

	return result, nil
}

func (s *Service) walkDirectory(ctx context.Context, provider storage.Provider, library database.Library, dirPath string, seenFiles map[string]struct{}, seenItems map[string]struct{}, result *SyncResult) error {
	result.DirectoriesScanned++

	objects, err := s.listAllDirectoryObjects(ctx, provider, dirPath)
	if err != nil {
		return fmt.Errorf("list directory %s: %w", dirPath, err)
	}

	sort.Slice(objects, func(i, j int) bool {
		return objects[i].Path < objects[j].Path
	})

	for _, object := range objects {
		if object.IsDir {
			if err := s.walkDirectory(ctx, provider, library, object.Path, seenFiles, seenItems, result); err != nil {
				return err
			}
			continue
		}
		if !isVideoFile(object.Path) {
			continue
		}

		result.FilesSeen++
		seenFiles[object.Path] = struct{}{}

		classified := classifyMediaFile(library.Type, object)
		seenItems[classified.SourcePath] = struct{}{}

		item, createdItem, err := s.upsertMediaItem(ctx, library.ID, classified)
		if err != nil {
			return err
		}
		if createdItem {
			result.MediaItemsUpserted++
		}
		if item.MatchStatus == "pending" {
			if _, err := s.QueueMediaItemMatch(ctx, item.ID, false); err != nil {
				return err
			}
		}

		fileRecord, createdFile, err := s.upsertMediaFile(ctx, library.ID, item.ID, object)
		if err != nil {
			return err
		}
		if createdFile {
			result.MediaFilesUpserted++
		}
		if fileRecord.ProbeStatus == "pending" {
			if _, err := s.QueueMediaFileProbe(ctx, fileRecord.ID, false); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service) upsertMediaItem(ctx context.Context, libraryID uint, classified classifiedMedia) (database.MediaItem, bool, error) {
	var item database.MediaItem
	err := s.db.WithContext(ctx).
		Where("library_id = ? AND source_path = ?", libraryID, classified.SourcePath).
		First(&item).Error
	created := false
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return database.MediaItem{}, false, err
		}
		item = database.MediaItem{LibraryID: libraryID, SourcePath: classified.SourcePath}
		created = true
	}
	baseChanged := created || mediaItemBaseChanged(item, classified)

	item.Type = classified.Type
	item.Title = classified.Title
	item.OriginalTitle = classified.OriginalTitle
	item.SeriesTitle = classified.SeriesTitle
	item.Year = classified.Year
	item.SeasonNumber = classified.SeasonNumber
	item.EpisodeNumber = classified.EpisodeNumber
	item.Status = classified.Status
	item.DeletedAt = nil
	if baseChanged {
		resetMediaItemMetadata(&item)
		item.MatchStatus = "pending"
	}
	if item.MatchStatus == "" {
		item.MatchStatus = "pending"
	}

	if created {
		if err := s.db.WithContext(ctx).Create(&item).Error; err != nil {
			return database.MediaItem{}, false, err
		}
		return item, true, nil
	}

	if err := s.db.WithContext(ctx).Save(&item).Error; err != nil {
		return database.MediaItem{}, false, err
	}
	return item, false, nil
}

func (s *Service) upsertMediaFile(ctx context.Context, libraryID, mediaItemID uint, object storage.Object) (database.MediaFile, bool, error) {
	fingerprint := buildFingerprint(object)
	var file database.MediaFile
	attachMediaItem := true
	retirePathMatches := false
	err := s.matchMediaFileForScan(ctx, libraryID, object, fingerprint, &file, &attachMediaItem, &retirePathMatches)
	created := false
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return database.MediaFile{}, false, err
		}
		file = database.MediaFile{LibraryID: libraryID}
		created = true
	}
	if created && retirePathMatches {
		if err := s.stageFallbackCandidate(ctx, libraryID, object.Path); err != nil {
			return database.MediaFile{}, false, err
		}
	}

	baseChanged := created || attachMediaItemChanged(file.MediaItemID, attachMediaItem, mediaItemID) || file.Fingerprint != fingerprint
	file.StoragePath = object.Path
	if attachMediaItem {
		file.MediaItemID = &mediaItemID
	} else {
		file.MediaItemID = nil
	}
	applyObjectIdentityEvidence(&file, object)
	file.Container = strings.TrimPrefix(strings.ToLower(path.Ext(object.Path)), ".")
	file.SizeBytes = object.Size
	file.LastModifiedAt = object.Modified
	file.Fingerprint = fingerprint
	file.ReplacedByID = nil
	file.DeletedAt = nil
	if baseChanged {
		resetMediaFileProbe(&file)
		file.ProbeStatus = "pending"
	}
	if file.ProbeStatus == "" {
		file.ProbeStatus = "pending"
	}

	if created {
		if err := s.db.WithContext(ctx).Create(&file).Error; err != nil {
			return database.MediaFile{}, false, err
		}
		return file, true, nil
	}

	if err := s.db.WithContext(ctx).Save(&file).Error; err != nil {
		return database.MediaFile{}, false, err
	}
	return file, false, nil
}

func (s *Service) matchMediaFileForScan(ctx context.Context, libraryID uint, object storage.Object, fingerprint string, out *database.MediaFile, attachMediaItem *bool, retirePathMatches *bool) error {
	query := s.db.WithContext(ctx)
	if identity := strings.TrimSpace(object.StableIdentity); identity != "" {
		// D-01: exact stable identity is the only scan-time continuity match that may survive a path change.
		err := query.
			Where("library_id = ? AND stable_identity_key = ? AND deleted_at IS NULL", libraryID, identity).
			First(out).Error
		if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	var pathMatch database.MediaFile
	err := query.
		Where("library_id = ? AND storage_path = ? AND deleted_at IS NULL", libraryID, object.Path).
		Order("id desc").
		First(&pathMatch).Error
	if err != nil {
		if attachMediaItem != nil {
			*attachMediaItem = true
		}
		return err
	}

	if pathMatch.Fingerprint == fingerprint {
		if attachMediaItem != nil {
			*attachMediaItem = pathMatch.MediaItemID != nil
		}
		*out = pathMatch
		return nil
	}

	// D-02: without exact stable identity, path is a locator only. A changed object at the same path becomes
	// a provisional fallback candidate instead of reusing the existing continuity-bound media file row.
	if attachMediaItem != nil {
		*attachMediaItem = false
	}
	if retirePathMatches != nil {
		*retirePathMatches = true
	}
	return gorm.ErrRecordNotFound
}

func (s *Service) cleanupMissingFiles(ctx context.Context, libraryID uint, seen map[string]struct{}) error {
	return markMissingRecords(ctx, s.db, &database.MediaFile{}, "library_id = ?", libraryID, "storage_path", seen, map[string]any{
		"deleted_at": time.Now().UTC(),
	})
}

func (s *Service) cleanupMissingFilesInScope(ctx context.Context, libraryID uint, rootPath string, seen map[string]struct{}) error {
	return markMissingRecordsInScope(ctx, s.db, &database.MediaFile{}, libraryID, "storage_path", rootPath, seen, map[string]any{
		"deleted_at": time.Now().UTC(),
	})
}

func (s *Service) stageFallbackCandidate(ctx context.Context, libraryID uint, storagePath string) error {
	return s.db.WithContext(ctx).
		Model(&database.MediaFile{}).
		Where("library_id = ? AND storage_path = ? AND deleted_at IS NULL", libraryID, storagePath).
		Update("deleted_at", time.Now().UTC()).Error
}

func (s *Service) cleanupMissingItems(ctx context.Context, libraryID uint, seen map[string]struct{}) error {
	return markMissingRecords(ctx, s.db, &database.MediaItem{}, "library_id = ?", libraryID, "source_path", seen, map[string]any{
		"deleted_at": time.Now().UTC(),
		"status":     "missing",
	})
}

func (s *Service) cleanupMissingItemsInScope(ctx context.Context, libraryID uint, rootPath string, seen map[string]struct{}) error {
	return markMissingRecordsInScope(ctx, s.db, &database.MediaItem{}, libraryID, "source_path", rootPath, seen, map[string]any{
		"deleted_at": time.Now().UTC(),
		"status":     "missing",
	})
}

func markMissingRecords(ctx context.Context, db *gorm.DB, model any, baseQuery string, libraryID uint, pathColumn string, seen map[string]struct{}, updates map[string]any) error {
	query := db.WithContext(ctx).
		Model(model).
		Where(baseQuery+" AND deleted_at IS NULL", libraryID)

	if len(seen) > 0 {
		paths := make([]string, 0, len(seen))
		for itemPath := range seen {
			paths = append(paths, itemPath)
		}
		query = query.Where(pathColumn+" NOT IN ?", paths)
	}

	return query.Updates(updates).Error
}

func markMissingRecordsInScope(ctx context.Context, db *gorm.DB, model any, libraryID uint, pathColumn string, rootPath string, seen map[string]struct{}, updates map[string]any) error {
	query := db.WithContext(ctx).
		Model(model).
		Where("library_id = ? AND deleted_at IS NULL", libraryID)
	query = applyScopedPathFilter(query, pathColumn, rootPath)

	if len(seen) > 0 {
		paths := make([]string, 0, len(seen))
		for itemPath := range seen {
			paths = append(paths, itemPath)
		}
		query = query.Where(pathColumn+" NOT IN ?", paths)
	}

	return query.Updates(updates).Error
}

func applyScopedPathFilter(query *gorm.DB, pathColumn string, rootPath string) *gorm.DB {
	normalizedRoot := strings.TrimSpace(rootPath)
	if normalizedRoot == "" || normalizedRoot == "/" {
		return query
	}
	trimmedRoot := strings.TrimRight(normalizedRoot, "/")
	if trimmedRoot == "" {
		trimmedRoot = "/"
	}
	if trimmedRoot == "/" {
		return query
	}
	return query.Where("("+pathColumn+" = ? OR "+pathColumn+" LIKE ?)", trimmedRoot, trimmedRoot+"/%")
}

func classifyMediaFile(libraryType string, object storage.Object) classifiedMedia {
	fileName := path.Base(object.Path)
	ext := path.Ext(fileName)
	rawTitle := strings.TrimSuffix(fileName, ext)
	normalizedTitle := cleanTitle(rawTitle)

	if groups := episodePattern.FindStringSubmatch(rawTitle); len(groups) > 0 {
		seriesTitle := cleanTitle(groups[1])
		season, episode := parseEpisodeNumbers(groups[2], groups[3], groups[4], groups[5])
		title := fmt.Sprintf("%s S%02dE%02d", seriesTitle, *season, *episode)
		return classifiedMedia{
			Type:          "episode",
			Title:         title,
			OriginalTitle: rawTitle,
			SeriesTitle:   seriesTitle,
			SeasonNumber:  season,
			EpisodeNumber: episode,
			SourcePath:    object.Path,
			Status:        "ready",
		}
	}

	year := parseYear(rawTitle)
	title := normalizedTitle
	if libraryType == "tv" || libraryType == "tvshows" || libraryType == "shows" {
		title = titleFromPath(object.Path)
	}

	return classifiedMedia{
		Type:          "movie",
		Title:         title,
		OriginalTitle: rawTitle,
		Year:          year,
		SourcePath:    object.Path,
		Status:        "ready",
	}
}

func (s *Service) listAllDirectoryObjects(ctx context.Context, provider storage.Provider, dirPath string) ([]storage.Object, error) {
	const pageSize = 1000

	var all []storage.Object
	for page := 1; ; page++ {
		objects, err := provider.List(ctx, storage.ListRequest{
			Path:    dirPath,
			Refresh: true,
			Page:    page,
			PerPage: pageSize,
		})
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

func buildFingerprint(object storage.Object) string {
	parts := []string{strconv.FormatInt(object.Size, 10)}
	if identity := strings.TrimSpace(object.StableIdentity); identity != "" {
		parts = append(parts, "stable="+identity)
	}
	if provider := strings.TrimSpace(object.Provider); provider != "" {
		parts = append(parts, "provider="+provider)
	}
	if hashInfo := marshalObjectHashInfo(object.HashInfo); hashInfo != "" {
		parts = append(parts, "hashes="+hashInfo)
	}
	if object.Modified != nil {
		parts = append(parts, object.Modified.UTC().Format(time.RFC3339Nano))
	}
	return strings.Join(parts, ":")
}

func applyObjectIdentityEvidence(file *database.MediaFile, object storage.Object) {
	if file == nil {
		return
	}

	file.StableIdentityKey = strings.TrimSpace(object.StableIdentity)
	file.ProviderName = strings.TrimSpace(object.Provider)
	file.ProviderHashesJSON = marshalObjectHashInfo(object.HashInfo)
	file.ReviewReason = ""
	if file.StableIdentityKey != "" {
		file.IdentitySource = mediaFileIdentitySourceStableIdentity
		file.IdentityStatus = mediaFileIdentityStatusExact
		file.ReviewStatus = mediaFileReviewStatusNone
		return
	}
	if file.ProviderName != "" || file.ProviderHashesJSON != "" {
		file.IdentitySource = mediaFileIdentitySourceProviderEvidence
		file.IdentityStatus = mediaFileIdentityStatusProvisional
		file.ReviewStatus = mediaFileReviewStatusPending
		file.ReviewReason = "awaiting_high_confidence_reconciliation"
		return
	}
	file.IdentitySource = mediaFileIdentitySourceNone
	file.IdentityStatus = mediaFileIdentityStatusProvisional
	file.ReviewStatus = mediaFileReviewStatusPending
	file.ReviewReason = "stable_identity_missing"
}

func marshalObjectHashInfo(input map[string]string) string {
	if len(input) == 0 {
		return ""
	}
	keys := make([]string, 0, len(input))
	normalized := make(map[string]string, len(input))
	for key, value := range input {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		normalized[trimmedKey] = trimmedValue
		keys = append(keys, trimmedKey)
	}
	if len(keys) == 0 {
		return ""
	}
	sort.Strings(keys)
	ordered := make(map[string]string, len(keys))
	for _, key := range keys {
		ordered[key] = normalized[key]
	}
	encoded, err := json.Marshal(ordered)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func attachMediaItemChanged(current *uint, attachMediaItem bool, mediaItemID uint) bool {
	if !attachMediaItem {
		return current != nil
	}
	if current == nil {
		return true
	}
	return *current != mediaItemID
}

func ReconcileProvisionalMediaFile(ctx context.Context, db *gorm.DB, mediaFileID uint) error {
	if db == nil {
		return nil
	}

	var file database.MediaFile
	if err := db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", mediaFileID).
		First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if file.DurationSeconds == nil || file.IdentityStatus != mediaFileIdentityStatusProvisional {
		return nil
	}

	var candidates []database.MediaFile
	if err := db.WithContext(ctx).
		Where("library_id = ? AND id <> ? AND deleted_at IS NOT NULL AND media_item_id IS NOT NULL AND replaced_by_id IS NULL AND size_bytes = ? AND duration_seconds IS NOT NULL", file.LibraryID, file.ID, file.SizeBytes).
		Order("deleted_at desc, id desc").
		Find(&candidates).Error; err != nil {
		return err
	}

	matches := make([]database.MediaFile, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.DurationSeconds == nil {
			continue
		}
		if durationDelta(*candidate.DurationSeconds, *file.DurationSeconds) <= fallbackDurationToleranceSeconds {
			matches = append(matches, candidate)
		}
	}
	if len(matches) == 0 {
		return db.WithContext(ctx).
			Model(&database.MediaFile{}).
			Where("id = ?", file.ID).
			Updates(map[string]any{
				"review_status": mediaFileReviewStatusPending,
				"review_reason": "no_high_confidence_match",
			}).Error
	}
	if len(matches) > 1 {
		return db.WithContext(ctx).
			Model(&database.MediaFile{}).
			Where("id = ?", file.ID).
			Updates(map[string]any{
				"review_status": mediaFileReviewStatusNeeded,
				"review_reason": "ambiguous_size_duration_match",
			}).Error
	}

	target := matches[0]
	targetMediaItemID := *target.MediaItemID
	currentMediaItemID := file.MediaItemID
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&database.MediaFile{}).
			Where("id = ?", file.ID).
			Updates(map[string]any{
				"media_item_id":   targetMediaItemID,
				"identity_status": mediaFileIdentityStatusReconciled,
				"review_status":   mediaFileReviewStatusNone,
				"review_reason":   "",
			}).Error; err != nil {
			return err
		}

		if err := tx.Model(&database.MediaFile{}).
			Where("id = ?", target.ID).
			Update("replaced_by_id", file.ID).Error; err != nil {
			return err
		}

		if err := tx.Model(&database.PlaybackProgress{}).
			Where("media_item_id = ? AND media_file_id = ?", targetMediaItemID, target.ID).
			Update("media_file_id", file.ID).Error; err != nil {
			return err
		}

		if err := tx.Model(&database.MediaItem{}).
			Where("id = ?", targetMediaItemID).
			Updates(map[string]any{
				"source_path": file.StoragePath,
				"status":      "ready",
				"deleted_at":  nil,
			}).Error; err != nil {
			return err
		}

		if currentMediaItemID != nil && *currentMediaItemID != targetMediaItemID {
			var activeCount int64
			if err := tx.Model(&database.MediaFile{}).
				Where("media_item_id = ? AND deleted_at IS NULL AND id <> ?", *currentMediaItemID, file.ID).
				Count(&activeCount).Error; err != nil {
				return err
			}
			if activeCount == 0 {
				if err := tx.Model(&database.MediaItem{}).
					Where("id = ?", *currentMediaItemID).
					Updates(map[string]any{
						"status":     "missing",
						"deleted_at": time.Now().UTC(),
					}).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func durationDelta(left, right float64) float64 {
	if left >= right {
		return left - right
	}
	return right - left
}

func isVideoFile(itemPath string) bool {
	_, ok := videoExtensions[strings.ToLower(path.Ext(itemPath))]
	return ok
}

func parseEpisodeNumbers(seasonLeft, episodeLeft, seasonRight, episodeRight string) (*int, *int) {
	seasonValue := seasonLeft
	episodeValue := episodeLeft
	if seasonValue == "" {
		seasonValue = seasonRight
		episodeValue = episodeRight
	}

	season, _ := strconv.Atoi(seasonValue)
	episode, _ := strconv.Atoi(episodeValue)
	return &season, &episode
}

func parseYear(input string) *int {
	match := yearPattern.FindStringSubmatch(input)
	if len(match) < 2 {
		return nil
	}
	value, err := strconv.Atoi(match[1])
	if err != nil {
		return nil
	}
	return &value
}

func titleFromPath(itemPath string) string {
	parent := path.Base(path.Dir(itemPath))
	if parent == "/" || parent == "." || parent == "" {
		return cleanTitle(strings.TrimSuffix(path.Base(itemPath), path.Ext(itemPath)))
	}
	return cleanTitle(parent)
}

func cleanTitle(input string) string {
	replacer := strings.NewReplacer(".", " ", "_", " ")
	cleaned := replacer.Replace(strings.TrimSpace(input))
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	cleaned = yearPattern.ReplaceAllString(cleaned, " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	cleaned = strings.Trim(cleaned, "- ")
	if cleaned == "" {
		return strings.TrimSpace(input)
	}
	return cleaned
}

func mediaItemBaseChanged(item database.MediaItem, classified classifiedMedia) bool {
	return item.Type != classified.Type ||
		item.Title != classified.Title ||
		item.OriginalTitle != classified.OriginalTitle ||
		item.SeriesTitle != classified.SeriesTitle ||
		!equalIntPointers(item.Year, classified.Year) ||
		!equalIntPointers(item.SeasonNumber, classified.SeasonNumber) ||
		!equalIntPointers(item.EpisodeNumber, classified.EpisodeNumber)
}

func resetMediaItemMetadata(item *database.MediaItem) {
	item.Overview = ""
	item.PosterURL = ""
	item.BackdropURL = ""
	item.GenresJSON = ""
	item.CastJSON = ""
	item.DirectorsJSON = ""
	item.ReleaseDate = ""
	item.RuntimeSeconds = nil
	item.MetadataProvider = ""
	item.ExternalID = ""
	item.MetadataConfidence = nil
}

func resetMediaFileProbe(file *database.MediaFile) {
	file.ProbeError = ""
	file.DurationSeconds = nil
	file.BitRate = nil
	file.Width = nil
	file.Height = nil
	file.VideoCodec = ""
	file.AudioTracksJSON = ""
	file.SubtitleTracksJSON = ""
}

func equalIntPointers(left, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
