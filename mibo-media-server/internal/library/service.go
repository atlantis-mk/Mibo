package library

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

type Service struct {
	cfg     config.Config
	db      *gorm.DB
	storage *providers.Registry
	jobs    *jobs.Service
}

type CreateMediaSourceInput struct {
	Provider   string                  `json:"provider"`
	Name       string                  `json:"name"`
	StorageRef string                  `json:"storage_ref"`
	RootPath   string                  `json:"root_path"`
	Config     *providers.SourceConfig `json:"config,omitempty"`
}

type UpdateMediaSourceInput struct {
	Name       string                  `json:"name"`
	StorageRef string                  `json:"storage_ref"`
	RootPath   string                  `json:"root_path"`
	Config     *providers.SourceConfig `json:"config,omitempty"`
}

type MediaSourceView struct {
	ID               uint                       `json:"id"`
	Name             string                     `json:"name"`
	Provider         string                     `json:"provider"`
	StorageRef       string                     `json:"storage_ref"`
	RootPath         string                     `json:"root_path"`
	Config           providers.SourceConfigView `json:"config,omitempty"`
	CapabilitiesJSON string                     `json:"capabilities_json"`
	CreatedAt        string                     `json:"created_at"`
	UpdatedAt        string                     `json:"updated_at"`
}

type CreateLibraryInput struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	MediaSourceID uint   `json:"media_source_id"`
	RootPath      string `json:"root_path"`
}

func NewService(cfg config.Config, db *gorm.DB, registry *providers.Registry, jobs *jobs.Service) *Service {
	return &Service{cfg: cfg, db: db, storage: registry, jobs: jobs}
}

func (s *Service) CreateMediaSource(ctx context.Context, input CreateMediaSourceInput) (database.MediaSource, error) {
	providerName := strings.ToLower(strings.TrimSpace(input.Provider))
	if providerName == "" {
		providerName = "local"
	}
	normalizedConfig, err := providers.NormalizeSourceConfig(providerName, input.Config, s.cfg)
	if err != nil {
		return database.MediaSource{}, err
	}
	provider, err := s.storage.Build(providerName, &normalizedConfig, input.RootPath)
	if err != nil {
		return database.MediaSource{}, err
	}

	name := strings.TrimSpace(input.Name)
	rootPath := normalizePathForProvider(provider.Name(), input.RootPath)
	storageRef := strings.TrimSpace(input.StorageRef)
	if storageRef == "" {
		storageRef = rootPath
	}
	if name == "" {
		return database.MediaSource{}, fmt.Errorf("name is required")
	}

	caps, err := provider.Capabilities(ctx)
	if err != nil {
		return database.MediaSource{}, err
	}

	if _, err := provider.ResolveStorage(ctx, storage.ResolveStorageRequest{Path: rootPath}); err != nil {
		return database.MediaSource{}, fmt.Errorf("resolve storage %s: %w", rootPath, err)
	}

	configJSON, err := providers.MarshalSourceConfig(normalizedConfig)
	if err != nil {
		return database.MediaSource{}, err
	}

	capsJSON, err := json.Marshal(caps)
	if err != nil {
		return database.MediaSource{}, err
	}

	source := database.MediaSource{
		Name:             name,
		Provider:         provider.Name(),
		StorageRef:       storageRef,
		RootPath:         rootPath,
		ConfigJSON:       configJSON,
		CapabilitiesJSON: string(capsJSON),
	}

	if err := s.db.WithContext(ctx).Create(&source).Error; err != nil {
		return database.MediaSource{}, err
	}

	return source, nil
}

func (s *Service) UpdateMediaSource(ctx context.Context, sourceID uint, input UpdateMediaSourceInput) (database.MediaSource, error) {
	var source database.MediaSource
	if err := s.db.WithContext(ctx).First(&source, sourceID).Error; err != nil {
		return database.MediaSource{}, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return database.MediaSource{}, fmt.Errorf("name is required")
	}

	rootPath := input.RootPath
	if strings.TrimSpace(rootPath) == "" {
		rootPath = source.RootPath
	}
	rootPath = normalizePathForProvider(source.Provider, rootPath)

	storageRef := strings.TrimSpace(input.StorageRef)
	if storageRef == "" {
		storageRef = rootPath
	}

	normalizedConfig, err := s.updatedSourceConfig(source, input.Config)
	if err != nil {
		return database.MediaSource{}, err
	}

	provider, err := s.storage.Build(source.Provider, &normalizedConfig, rootPath)
	if err != nil {
		return database.MediaSource{}, err
	}
	if _, err := provider.ResolveStorage(ctx, storage.ResolveStorageRequest{Path: rootPath}); err != nil {
		return database.MediaSource{}, fmt.Errorf("resolve storage %s: %w", rootPath, err)
	}

	caps, err := provider.Capabilities(ctx)
	if err != nil {
		return database.MediaSource{}, err
	}
	capsJSON, err := json.Marshal(caps)
	if err != nil {
		return database.MediaSource{}, err
	}
	configJSON, err := providers.MarshalSourceConfig(normalizedConfig)
	if err != nil {
		return database.MediaSource{}, err
	}

	updates := map[string]any{
		"name":              name,
		"storage_ref":       storageRef,
		"root_path":         rootPath,
		"config_json":       configJSON,
		"capabilities_json": string(capsJSON),
	}
	if err := s.db.WithContext(ctx).Model(&source).Updates(updates).Error; err != nil {
		return database.MediaSource{}, err
	}
	if err := s.db.WithContext(ctx).First(&source, sourceID).Error; err != nil {
		return database.MediaSource{}, err
	}

	return source, nil
}

func (s *Service) ListMediaSources(ctx context.Context) ([]database.MediaSource, error) {
	var sources []database.MediaSource
	if err := s.db.WithContext(ctx).
		Order("id asc").
		Find(&sources).Error; err != nil {
		return nil, err
	}
	return sources, nil
}

func (s *Service) ListMediaSourceViews(ctx context.Context) ([]MediaSourceView, error) {
	sources, err := s.ListMediaSources(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]MediaSourceView, 0, len(sources))
	for _, source := range sources {
		view, err := s.MediaSourceView(source)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) MediaSourceView(source database.MediaSource) (MediaSourceView, error) {
	sourceConfig, err := s.sourceConfigFor(source)
	if err != nil {
		return MediaSourceView{}, err
	}
	return MediaSourceView{
		ID:               source.ID,
		Name:             source.Name,
		Provider:         source.Provider,
		StorageRef:       source.StorageRef,
		RootPath:         source.RootPath,
		Config:           sourceConfig.Sanitized(),
		CapabilitiesJSON: source.CapabilitiesJSON,
		CreatedAt:        source.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        source.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

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

	library := database.Library{
		Name:           strings.TrimSpace(input.Name),
		Type:           strings.TrimSpace(strings.ToLower(input.Type)),
		MediaSourceID:  source.ID,
		RootPath:       rootPath,
		Status:         "pending",
		ScannerEnabled: true,
	}

	if err := s.db.WithContext(ctx).Create(&library).Error; err != nil {
		return database.Library{}, database.Job{}, err
	}

	job, err := s.QueueLibraryScan(ctx, library.ID)
	if err != nil {
		return database.Library{}, database.Job{}, err
	}

	return library, job, nil
}

func (s *Service) providerForSource(ctx context.Context, sourceID uint) (database.MediaSource, storage.Provider, error) {
	var source database.MediaSource
	if err := s.db.WithContext(ctx).First(&source, sourceID).Error; err != nil {
		return database.MediaSource{}, nil, err
	}
	provider, err := s.storage.BuildForSource(source)
	if err != nil {
		return database.MediaSource{}, nil, err
	}
	return source, provider, nil
}

func (s *Service) sourceConfigFor(source database.MediaSource) (providers.SourceConfig, error) {
	parsedConfig, err := providers.ParseSourceConfig(source.ConfigJSON)
	if err != nil {
		return providers.SourceConfig{}, err
	}
	return providers.NormalizeSourceConfig(source.Provider, &parsedConfig, s.cfg)
}

func (s *Service) updatedSourceConfig(source database.MediaSource, input *providers.SourceConfig) (providers.SourceConfig, error) {
	existingConfig, err := s.sourceConfigFor(source)
	if err != nil {
		return providers.SourceConfig{}, err
	}
	if strings.EqualFold(source.Provider, "local") {
		return providers.SourceConfig{}, nil
	}
	if !strings.EqualFold(source.Provider, "openlist") {
		return providers.NormalizeSourceConfig(source.Provider, input, s.cfg)
	}

	mergedConfig := existingConfig
	if mergedConfig.OpenList == nil {
		mergedConfig.OpenList = &providers.OpenListSourceConfig{}
	}
	if input != nil && input.OpenList != nil {
		if value := strings.TrimSpace(input.OpenList.BaseURL); value != "" {
			mergedConfig.OpenList.BaseURL = value
		}
		if value := strings.TrimSpace(input.OpenList.Username); value != "" {
			mergedConfig.OpenList.Username = value
		}
		if value := input.OpenList.Password; strings.TrimSpace(value) != "" {
			mergedConfig.OpenList.Password = value
		}
		if value := strings.TrimSpace(input.OpenList.Token); value != "" {
			mergedConfig.OpenList.Token = value
		}
		if value := strings.TrimSpace(input.OpenList.Timeout); value != "" {
			mergedConfig.OpenList.Timeout = value
		}
		mergedConfig.OpenList.InsecureSkip = input.OpenList.InsecureSkip
	}

	return providers.NormalizeSourceConfig(source.Provider, &mergedConfig, s.cfg)
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
	if err := s.db.WithContext(ctx).
		Order("id asc").
		Find(&libraries).Error; err != nil {
		return nil, err
	}
	return libraries, nil
}

func (s *Service) ListActiveLibraries(ctx context.Context) ([]database.Library, error) {
	var libraries []database.Library
	if err := s.db.WithContext(ctx).
		Where("status = ? AND scanner_enabled = ?", "active", true).
		Order("id asc").
		Find(&libraries).Error; err != nil {
		return nil, err
	}
	return libraries, nil
}

func (s *Service) DeleteLibrary(ctx context.Context, libraryID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return deleteLibraryRecords(ctx, tx, libraryID)
	})
}

func (s *Service) DeleteMediaSource(ctx context.Context, sourceID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var source database.MediaSource
		if err := tx.First(&source, sourceID).Error; err != nil {
			return err
		}

		var libraryIDs []uint
		if err := tx.Model(&database.Library{}).
			Where("media_source_id = ?", sourceID).
			Order("id asc").
			Pluck("id", &libraryIDs).Error; err != nil {
			return err
		}

		for _, libraryID := range libraryIDs {
			if err := deleteLibraryRecords(ctx, tx, libraryID); err != nil {
				return err
			}
		}

		if err := tx.Delete(&database.MediaSource{}, sourceID).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *Service) updateLibraryStatus(ctx context.Context, libraryID uint, status string) error {
	return s.db.WithContext(ctx).
		Model(&database.Library{}).
		Where("id = ?", libraryID).
		Update("status", status).Error
}

func deleteLibraryRecords(ctx context.Context, tx *gorm.DB, libraryID uint) error {
	var record database.Library
	if err := tx.WithContext(ctx).First(&record, libraryID).Error; err != nil {
		return err
	}

	var mediaItemIDs []uint
	if err := tx.WithContext(ctx).
		Model(&database.MediaItem{}).
		Where("library_id = ?", libraryID).
		Pluck("id", &mediaItemIDs).Error; err != nil {
		return err
	}

	if len(mediaItemIDs) > 0 {
		if err := tx.WithContext(ctx).
			Where("media_item_id IN ?", mediaItemIDs).
			Delete(&database.PlaybackProgress{}).Error; err != nil {
			return err
		}
	}

	if err := tx.WithContext(ctx).
		Where("library_id = ?", libraryID).
		Delete(&database.MediaFile{}).Error; err != nil {
		return err
	}

	if err := tx.WithContext(ctx).
		Where("library_id = ?", libraryID).
		Delete(&database.MediaItem{}).Error; err != nil {
		return err
	}

	result := tx.WithContext(ctx).
		Where("id = ?", libraryID).
		Delete(&database.Library{})
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
