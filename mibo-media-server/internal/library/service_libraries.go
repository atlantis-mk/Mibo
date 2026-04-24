package library

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

func (s *Service) CreateLibrary(ctx context.Context, input CreateLibraryInput) (database.Library, database.Job, error) {
	if strings.TrimSpace(input.Name) == "" {
		return database.Library{}, database.Job{}, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(input.Type) == "" {
		return database.Library{}, database.Job{}, fmt.Errorf("type is required")
	}
	if input.MediaSourceID == 0 {
		return database.Library{}, database.Job{}, fmt.Errorf("media_source_id is required")
	}
	var source database.MediaSource
	if err := s.db.WithContext(ctx).First(&source, input.MediaSourceID).Error; err != nil {
		return database.Library{}, database.Job{}, err
	}
	rootPath := normalizePath(input.RootPath)
	if rootPath == "/" {
		rootPath = source.RootPath
	}
	rootPath = normalizePathForProvider(source.Provider, rootPath)
	provider, err := s.storage.BuildForSource(source)
	if err != nil {
		return database.Library{}, database.Job{}, err
	}
	if _, err := provider.ResolveStorage(ctx, storage.ResolveStorageRequest{Path: rootPath}); err != nil {
		return database.Library{}, database.Job{}, fmt.Errorf("resolve library root %s: %w", rootPath, err)
	}
	library := database.Library{Name: strings.TrimSpace(input.Name), Type: strings.TrimSpace(strings.ToLower(input.Type)), MediaSourceID: source.ID, RootPath: rootPath, Status: "pending", ScannerEnabled: true}
	if err := s.db.WithContext(ctx).Create(&library).Error; err != nil {
		return database.Library{}, database.Job{}, err
	}
	job, err := s.QueueLibraryScan(ctx, library.ID)
	if err != nil {
		return database.Library{}, database.Job{}, err
	}
	return library, job, nil
}

func (s *Service) QueueTargetedRefresh(ctx context.Context, libraryID uint, rootPath, reason string) (database.Job, error) {
	record, _, provider, err := s.providerForLibrary(ctx, libraryID)
	if err != nil {
		return database.Job{}, err
	}
	if s.jobs == nil {
		return database.Job{}, fmt.Errorf("jobs service unavailable")
	}
	normalizedReason := strings.TrimSpace(strings.ToLower(reason))
	if normalizedReason == "" {
		normalizedReason = "manual"
	}
	targetRoot, err := scopedRefreshRoot(provider.Name(), record.RootPath, rootPath)
	if err != nil {
		return database.Job{}, err
	}
	jobKey := fmt.Sprintf("targeted-refresh:%d:%s:%s", record.ID, targetRoot, normalizedReason)
	return s.jobs.EnqueueUnique(ctx, JobKindTargetedRefresh, jobKey, targetedRefreshPayload{LibraryID: record.ID, RootPath: targetRoot, Reason: normalizedReason})
}

func (s *Service) QueueSearchDocumentReindex(ctx context.Context, mediaItemID uint) (database.Job, error) {
	return s.jobs.EnqueueUnique(ctx, JobKindReindexSearchDocument, fmt.Sprintf("reindex-search-document:%d", mediaItemID), map[string]any{
		"media_item_id": mediaItemID,
	})
}

func (s *Service) QueueLibrarySearchReindex(ctx context.Context, libraryID uint, rootPath string) (database.Job, error) {
	if s.jobs == nil {
		return database.Job{}, fmt.Errorf("jobs service unavailable")
	}
	payload := targetedRefreshPayload{LibraryID: libraryID, RootPath: strings.TrimSpace(rootPath)}
	jobKey := fmt.Sprintf("reindex-library-search:%d:%s", libraryID, payload.RootPath)
	return s.jobs.EnqueueUnique(ctx, JobKindReindexLibrarySearch, jobKey, payload)
}

func (s *Service) providerForLibrary(ctx context.Context, libraryID uint) (database.Library, database.MediaSource, storage.Provider, error) {
	var libraryRecord database.Library
	if err := s.db.WithContext(ctx).First(&libraryRecord, libraryID).Error; err != nil {
		return database.Library{}, database.MediaSource{}, nil, err
	}
	source, provider, err := s.providerForSource(ctx, libraryRecord.MediaSourceID)
	if err != nil {
		return database.Library{}, database.MediaSource{}, nil, err
	}
	return libraryRecord, source, provider, nil
}

func (s *Service) ListLibraries(ctx context.Context) ([]database.Library, error) {
	var libraries []database.Library
	if err := s.db.WithContext(ctx).Order("id asc").Find(&libraries).Error; err != nil {
		return nil, err
	}
	return libraries, nil
}

func (s *Service) ListActiveLibraries(ctx context.Context) ([]database.Library, error) {
	var libraries []database.Library
	if err := s.db.WithContext(ctx).Where("status = ? AND scanner_enabled = ?", "active", true).Order("id asc").Find(&libraries).Error; err != nil {
		return nil, err
	}
	return libraries, nil
}

func (s *Service) DeleteLibrary(ctx context.Context, libraryID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return deleteLibraryRecords(ctx, tx, libraryID)
	})
}

func (s *Service) updateLibraryStatus(ctx context.Context, libraryID uint, status string) error {
	return s.db.WithContext(ctx).Model(&database.Library{}).Where("id = ?", libraryID).Update("status", status).Error
}

func deleteLibraryRecords(ctx context.Context, tx *gorm.DB, libraryID uint) error {
	var record database.Library
	if err := tx.WithContext(ctx).First(&record, libraryID).Error; err != nil {
		return err
	}
	var mediaItemIDs []uint
	if err := tx.WithContext(ctx).Model(&database.MediaItem{}).Where("library_id = ?", libraryID).Pluck("id", &mediaItemIDs).Error; err != nil {
		return err
	}
	if len(mediaItemIDs) > 0 {
		if err := tx.WithContext(ctx).Where("media_item_id IN ?", mediaItemIDs).Delete(&database.PlaybackProgress{}).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Where("media_item_id IN ?", mediaItemIDs).Delete(&database.SearchDocument{}).Error; err != nil {
			return err
		}
	}
	if err := tx.WithContext(ctx).Where("library_id = ?", libraryID).Delete(&database.MediaFile{}).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Where("library_id = ?", libraryID).Delete(&database.MediaItem{}).Error; err != nil {
		return err
	}
	result := tx.WithContext(ctx).Where("id = ?", libraryID).Delete(&database.Library{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func normalizePath(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || trimmed == "/" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "/" + trimmed
	}
	return trimmed
}

func normalizePathForProvider(providerName, input string) string {
	if strings.EqualFold(strings.TrimSpace(providerName), "local") {
		trimmed := strings.TrimSpace(input)
		if trimmed == "" {
			return "/"
		}
		return trimmed
	}
	return normalizePath(input)
}

func scopedRefreshRoot(providerName, libraryRoot, requestedRoot string) (string, error) {
	normalizedLibraryRoot := normalizePathForProvider(providerName, libraryRoot)
	trimmedRequested := strings.TrimSpace(requestedRoot)
	if trimmedRequested == "" {
		return normalizedLibraryRoot, nil
	}
	if strings.EqualFold(strings.TrimSpace(providerName), "local") {
		normalizedRequested := strings.TrimSpace(requestedRoot)
		if normalizedRequested == "" {
			return normalizedLibraryRoot, nil
		}
		cleanLibraryRoot := filepath.Clean(normalizedLibraryRoot)
		cleanRequested := filepath.Clean(normalizedRequested)
		rel, err := filepath.Rel(cleanLibraryRoot, cleanRequested)
		if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
			return "", fmt.Errorf("refresh root %s is outside library root %s", cleanRequested, cleanLibraryRoot)
		}
		return cleanRequested, nil
	}
	normalizedRequested := normalizePath(requestedRoot)
	if normalizedRequested == normalizedLibraryRoot || strings.HasPrefix(normalizedRequested, normalizedLibraryRoot+"/") {
		return normalizedRequested, nil
	}
	return "", fmt.Errorf("refresh root %s is outside library root %s", normalizedRequested, normalizedLibraryRoot)
}
