package library

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

func (s *Service) CreateMediaSource(ctx context.Context, input CreateMediaSourceInput) (database.MediaSource, error) {
	providerName := strings.ToLower(strings.TrimSpace(input.Provider))
	if providerName == "" {
		providerName = "local"
	}
	normalizedConfig, err := providers.NormalizeSourceConfig(providerName, input.Config, s.cfg)
	if err != nil {
		return database.MediaSource{}, err
	}
	provider, err := s.storageRegistry().Build(providerName, &normalizedConfig, input.RootPath)
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
	source := database.MediaSource{Name: name, Provider: provider.Name(), StorageRef: storageRef, RootPath: rootPath, ConfigJSON: configJSON, CapabilitiesJSON: string(capsJSON)}
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
	provider, err := s.storageRegistry().Build(source.Provider, &normalizedConfig, rootPath)
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
	updates := map[string]any{"name": name, "storage_ref": storageRef, "root_path": rootPath, "config_json": configJSON, "capabilities_json": string(capsJSON)}
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
	if err := s.db.WithContext(ctx).Order("id asc").Find(&sources).Error; err != nil {
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
	return MediaSourceView{ID: source.ID, Name: source.Name, Provider: source.Provider, StorageRef: source.StorageRef, RootPath: source.RootPath, Config: sourceConfig.Sanitized(), CapabilitiesJSON: source.CapabilitiesJSON, CreatedAt: source.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: source.UpdatedAt.UTC().Format(time.RFC3339)}, nil
}

func (s *Service) DeleteMediaSource(ctx context.Context, sourceID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var source database.MediaSource
		if err := tx.First(&source, sourceID).Error; err != nil {
			return err
		}
		var libraryIDs []uint
		if err := tx.Model(&database.Library{}).Where("media_source_id = ?", sourceID).Order("id asc").Pluck("id", &libraryIDs).Error; err != nil {
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

func (s *Service) providerForSource(ctx context.Context, sourceID uint) (database.MediaSource, storage.Provider, error) {
	var source database.MediaSource
	if err := s.db.WithContext(ctx).First(&source, sourceID).Error; err != nil {
		return database.MediaSource{}, nil, err
	}
	provider, err := s.storageRegistry().BuildForSource(source)
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
