package library

import (
	"context"
	"encoding/json"
	"errors"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

func (s *Service) upsertMediaItem(ctx context.Context, libraryID uint, classified classifiedMedia) (database.MediaItem, bool, error) {
	var item database.MediaItem
	err := s.db.WithContext(ctx).Where("library_id = ? AND source_path = ?", libraryID, classified.SourcePath).First(&item).Error
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
	item.Year = classified.Year
	item.SeasonNumber = classified.SeasonNumber
	item.EpisodeNumber = classified.EpisodeNumber
	item.Status = classified.Status
	item.DeletedAt = nil
	if created || baseChanged || !hasMatchedMetadata(item) {
		item.Title = classified.Title
		item.OriginalTitle = classified.OriginalTitle
		item.SeriesTitle = classified.SeriesTitle
	}
	if baseChanged {
		resetMediaItemMetadata(&item)
		item.Title = classified.Title
		item.OriginalTitle = classified.OriginalTitle
		item.SeriesTitle = classified.SeriesTitle
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
		err := query.Where("library_id = ? AND stable_identity_key = ? AND deleted_at IS NULL", libraryID, identity).First(out).Error
		if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	var pathMatch database.MediaFile
	err := query.Where("library_id = ? AND storage_path = ? AND deleted_at IS NULL", libraryID, object.Path).Order("id desc").First(&pathMatch).Error
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
	if attachMediaItem != nil {
		*attachMediaItem = false
	}
	if retirePathMatches != nil {
		*retirePathMatches = true
	}
	return gorm.ErrRecordNotFound
}

func (s *Service) cleanupMissingFiles(ctx context.Context, libraryID uint, seen map[string]struct{}) error {
	return markMissingRecords(ctx, s.db, &database.MediaFile{}, "library_id = ?", libraryID, "storage_path", seen, map[string]any{"deleted_at": cleanupDeletedAt()})
}

func (s *Service) cleanupMissingFilesInScope(ctx context.Context, libraryID uint, rootPath string, seen map[string]struct{}) error {
	return markMissingRecordsInScope(ctx, s.db, &database.MediaFile{}, libraryID, "storage_path", rootPath, seen, map[string]any{"deleted_at": cleanupDeletedAt()})
}

func (s *Service) stageFallbackCandidate(ctx context.Context, libraryID uint, storagePath string) error {
	return s.db.WithContext(ctx).Model(&database.MediaFile{}).Where("library_id = ? AND storage_path = ? AND deleted_at IS NULL", libraryID, storagePath).Update("deleted_at", cleanupDeletedAt()).Error
}

func (s *Service) cleanupMissingItems(ctx context.Context, libraryID uint, seen map[string]struct{}) error {
	return markMissingRecords(ctx, s.db, &database.MediaItem{}, "library_id = ?", libraryID, "source_path", seen, map[string]any{"deleted_at": cleanupDeletedAt(), "status": "missing"})
}

func (s *Service) cleanupMissingItemsInScope(ctx context.Context, libraryID uint, rootPath string, seen map[string]struct{}) error {
	return markMissingRecordsInScope(ctx, s.db, &database.MediaItem{}, libraryID, "source_path", rootPath, seen, map[string]any{"deleted_at": cleanupDeletedAt(), "status": "missing"})
}

func markMissingRecords(ctx context.Context, db *gorm.DB, model any, baseQuery string, libraryID uint, pathColumn string, seen map[string]struct{}, updates map[string]any) error {
	query := db.WithContext(ctx).Model(model).Where(baseQuery+" AND deleted_at IS NULL", libraryID)
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
	query := db.WithContext(ctx).Model(model).Where("library_id = ? AND deleted_at IS NULL", libraryID)
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
