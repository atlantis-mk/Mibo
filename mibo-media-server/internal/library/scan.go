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

func (s *Service) QueueLibraryScan(ctx context.Context, libraryID uint) (database.Job, error) {
	var record database.Library
	if err := s.db.WithContext(ctx).First(&record, libraryID).Error; err != nil {
		return database.Job{}, err
	}

	return s.jobs.EnqueueUnique(ctx, "sync_library", fmt.Sprintf("scan-library-%d", record.ID), map[string]any{
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

func (s *Service) scanLibrary(ctx context.Context, provider storage.Provider, library database.Library, rootPath string) (SyncResult, error) {
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

	if err := s.cleanupMissingFiles(ctx, library.ID, seenFiles); err != nil {
		return SyncResult{}, err
	}
	if err := s.cleanupMissingItems(ctx, library.ID, seenItems); err != nil {
		return SyncResult{}, err
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
	var file database.MediaFile
	err := s.db.WithContext(ctx).
		Where("library_id = ? AND storage_path = ?", libraryID, object.Path).
		First(&file).Error
	created := false
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return database.MediaFile{}, false, err
		}
		file = database.MediaFile{LibraryID: libraryID, StoragePath: object.Path}
		created = true
	}

	fingerprint := buildFingerprint(object)
	baseChanged := created || file.MediaItemID == nil || *file.MediaItemID != mediaItemID || file.Fingerprint != fingerprint
	file.MediaItemID = &mediaItemID
	file.Container = strings.TrimPrefix(strings.ToLower(path.Ext(object.Path)), ".")
	file.SizeBytes = object.Size
	file.LastModifiedAt = object.Modified
	file.Fingerprint = fingerprint
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

func (s *Service) cleanupMissingFiles(ctx context.Context, libraryID uint, seen map[string]struct{}) error {
	return markMissingRecords(ctx, s.db, &database.MediaFile{}, "library_id = ?", libraryID, "storage_path", seen, map[string]any{
		"deleted_at":    time.Now().UTC(),
		"media_item_id": nil,
	})
}

func (s *Service) cleanupMissingItems(ctx context.Context, libraryID uint, seen map[string]struct{}) error {
	return markMissingRecords(ctx, s.db, &database.MediaItem{}, "library_id = ?", libraryID, "source_path", seen, map[string]any{
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
	parts := []string{object.Path, strconv.FormatInt(object.Size, 10)}
	if object.Modified != nil {
		parts = append(parts, object.Modified.UTC().Format(time.RFC3339Nano))
	}
	return strings.Join(parts, ":")
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
