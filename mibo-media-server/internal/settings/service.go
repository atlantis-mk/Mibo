package settings

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const metadataCategory = "metadata"
const scanCategory = "scan"

const (
	tmdbAPIKeyKey       = "tmdb_api_key"
	tmdbBaseURLKey      = "tmdb_base_url"
	tmdbImageBaseURLKey = "tmdb_image_base_url"
	tmdbLanguageKey     = "tmdb_language"
	tmdbTimeoutKey      = "tmdb_timeout"
	tvdbAPIKeyKey       = "tvdb_api_key"
	tvdbBaseURLKey      = "tvdb_base_url"
	tvdbLanguageKey     = "tvdb_language"
	tvdbTimeoutKey      = "tvdb_timeout"
)

const refreshIntervalKey = "refresh_interval_hours"

type Service struct {
	db       *gorm.DB
	fallback config.MetadataConfig
}

type MetadataProviderSettings struct {
	Configured     bool   `json:"configured"`
	APIKeyMasked   bool   `json:"api_key_masked"`
	BaseURL        string `json:"base_url"`
	ImageBaseURL   string `json:"image_base_url,omitempty"`
	Language       string `json:"language"`
	Timeout        string `json:"timeout"`
	Source         string `json:"source"`
	Implementation string `json:"implementation"`
}

type MetadataSettings struct {
	TMDB MetadataProviderSettings `json:"tmdb"`
	TVDB MetadataProviderSettings `json:"tvdb"`
}

type MetadataProviderInput struct {
	APIKey       string `json:"api_key"`
	ClearAPIKey  bool   `json:"clear_api_key"`
	BaseURL      string `json:"base_url"`
	ImageBaseURL string `json:"image_base_url"`
	Language     string `json:"language"`
	Timeout      string `json:"timeout"`
}

type UpdateMetadataSettingsInput struct {
	TMDB MetadataProviderInput `json:"tmdb"`
	TVDB MetadataProviderInput `json:"tvdb"`
}

type ScanSettings struct {
	RefreshIntervalHours int `json:"refresh_interval_hours"`
}

type UpdateScanSettingsInput struct {
	RefreshIntervalHours int `json:"refresh_interval_hours"`
}

func NewService(db *gorm.DB, fallback config.MetadataConfig) *Service {
	return &Service{db: db, fallback: fallback}
}

func (s *Service) GetMetadataSettings(ctx context.Context) (MetadataSettings, error) {
	tmdbCfg, tmdbSource, err := s.ResolveTMDBConfig(ctx)
	if err != nil {
		return MetadataSettings{}, err
	}
	tvdbCfg, tvdbSource, err := s.ResolveTVDBConfig(ctx)
	if err != nil {
		return MetadataSettings{}, err
	}

	return MetadataSettings{
		TMDB: MetadataProviderSettings{
			Configured:     strings.TrimSpace(tmdbCfg.APIKey) != "",
			APIKeyMasked:   strings.TrimSpace(tmdbCfg.APIKey) != "",
			BaseURL:        tmdbCfg.BaseURL,
			ImageBaseURL:   tmdbCfg.ImageBaseURL,
			Language:       tmdbCfg.Language,
			Timeout:        tmdbCfg.Timeout.String(),
			Source:         tmdbSource,
			Implementation: "active",
		},
		TVDB: MetadataProviderSettings{
			Configured:     strings.TrimSpace(tvdbCfg.APIKey) != "",
			APIKeyMasked:   strings.TrimSpace(tvdbCfg.APIKey) != "",
			BaseURL:        tvdbCfg.BaseURL,
			Language:       tvdbCfg.Language,
			Timeout:        tvdbCfg.Timeout.String(),
			Source:         tvdbSource,
			Implementation: "planned",
		},
	}, nil
}

func (s *Service) UpdateMetadataSettings(ctx context.Context, input UpdateMetadataSettingsInput) (MetadataSettings, error) {
	if err := s.updateTMDBSettings(ctx, input.TMDB); err != nil {
		return MetadataSettings{}, err
	}
	if err := s.updateTVDBSettings(ctx, input.TVDB); err != nil {
		return MetadataSettings{}, err
	}
	return s.GetMetadataSettings(ctx)
}

func (s *Service) ResolveTMDBConfig(ctx context.Context) (config.TMDBConfig, string, error) {
	resolved := s.fallback.TMDB
	values, err := s.loadCategoryValues(ctx, metadataCategory)
	if err != nil {
		return config.TMDBConfig{}, "none", err
	}
	source := sourceForValue(values[tmdbAPIKeyKey], resolved.APIKey)
	applyStringOverride(&resolved.APIKey, values[tmdbAPIKeyKey])
	applyStringOverride(&resolved.BaseURL, values[tmdbBaseURLKey])
	applyStringOverride(&resolved.ImageBaseURL, values[tmdbImageBaseURLKey])
	applyStringOverride(&resolved.Language, values[tmdbLanguageKey])
	applyDurationOverride(&resolved.Timeout, values[tmdbTimeoutKey])
	return resolved, source, nil
}

func (s *Service) ResolveTVDBConfig(ctx context.Context) (config.TVDBConfig, string, error) {
	resolved := s.fallback.TVDB
	values, err := s.loadCategoryValues(ctx, metadataCategory)
	if err != nil {
		return config.TVDBConfig{}, "none", err
	}
	source := sourceForValue(values[tvdbAPIKeyKey], resolved.APIKey)
	applyStringOverride(&resolved.APIKey, values[tvdbAPIKeyKey])
	applyStringOverride(&resolved.BaseURL, values[tvdbBaseURLKey])
	applyStringOverride(&resolved.Language, values[tvdbLanguageKey])
	applyDurationOverride(&resolved.Timeout, values[tvdbTimeoutKey])
	return resolved, source, nil
}

func (s *Service) GetScanSettings(ctx context.Context) (ScanSettings, error) {
	values, err := s.loadCategoryValues(ctx, scanCategory)
	if err != nil {
		return ScanSettings{RefreshIntervalHours: 24}, nil
	}
	intervalStr := values[refreshIntervalKey]
	if intervalStr == "" {
		return ScanSettings{RefreshIntervalHours: 24}, nil
	}
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval <= 0 {
		return ScanSettings{RefreshIntervalHours: 24}, nil
	}
	return ScanSettings{RefreshIntervalHours: interval}, nil
}

func (s *Service) UpdateScanSettings(ctx context.Context, input UpdateScanSettingsInput) (ScanSettings, error) {
	interval := input.RefreshIntervalHours
	if interval <= 0 {
		return ScanSettings{}, fmt.Errorf("refresh_interval_hours must be greater than 0")
	}
	record := database.SystemSetting{
		Category: scanCategory,
		Key:      refreshIntervalKey,
		Value:    strconv.Itoa(interval),
		IsSecret: false,
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "category"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "is_secret", "updated_at"}),
	}).Create(&record).Error
	if err != nil {
		return ScanSettings{}, err
	}
	return ScanSettings{RefreshIntervalHours: interval}, nil
}

func (s *Service) updateTMDBSettings(ctx context.Context, input MetadataProviderInput) error {
	if err := s.updateSecretSetting(ctx, tmdbAPIKeyKey, input.APIKey, input.ClearAPIKey); err != nil {
		return err
	}
	if err := s.updatePlainSetting(ctx, tmdbBaseURLKey, input.BaseURL); err != nil {
		return err
	}
	if err := s.updatePlainSetting(ctx, tmdbImageBaseURLKey, input.ImageBaseURL); err != nil {
		return err
	}
	if err := s.updatePlainSetting(ctx, tmdbLanguageKey, input.Language); err != nil {
		return err
	}
	return s.updatePlainSetting(ctx, tmdbTimeoutKey, input.Timeout)
}

func (s *Service) updateTVDBSettings(ctx context.Context, input MetadataProviderInput) error {
	if err := s.updateSecretSetting(ctx, tvdbAPIKeyKey, input.APIKey, input.ClearAPIKey); err != nil {
		return err
	}
	if err := s.updatePlainSetting(ctx, tvdbBaseURLKey, input.BaseURL); err != nil {
		return err
	}
	if err := s.updatePlainSetting(ctx, tvdbLanguageKey, input.Language); err != nil {
		return err
	}
	return s.updatePlainSetting(ctx, tvdbTimeoutKey, input.Timeout)
}

func (s *Service) updatePlainSetting(ctx context.Context, key, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return s.deleteSetting(ctx, key)
	}
	return s.upsertSetting(ctx, key, trimmed, false)
}

func (s *Service) updateSecretSetting(ctx context.Context, key, value string, clear bool) error {
	if clear {
		return s.deleteSetting(ctx, key)
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return s.upsertSetting(ctx, key, trimmed, true)
}

func (s *Service) upsertSetting(ctx context.Context, key, value string, secret bool) error {
	return s.upsertCategorySetting(ctx, metadataCategory, key, value, secret)
}

func (s *Service) upsertCategorySetting(ctx context.Context, category, key, value string, secret bool) error {
	return upsertCategorySettingWithDB(ctx, s.db, category, key, value, secret)
}

func upsertCategorySettingWithDB(ctx context.Context, db *gorm.DB, category, key, value string, secret bool) error {
	record := database.SystemSetting{
		Category: category,
		Key:      key,
		Value:    value,
		IsSecret: secret,
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "category"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "is_secret", "updated_at"}),
	}).Create(&record).Error
}

func (s *Service) deleteSetting(ctx context.Context, key string) error {
	return s.deleteCategorySetting(ctx, metadataCategory, key)
}

func (s *Service) deleteCategorySetting(ctx context.Context, category, key string) error {
	return deleteCategorySettingWithDB(ctx, s.db, category, key)
}

func deleteCategorySettingWithDB(ctx context.Context, db *gorm.DB, category, key string) error {
	return db.WithContext(ctx).
		Where("category = ? AND key = ?", category, key).
		Delete(&database.SystemSetting{}).Error
}

func (s *Service) loadCategoryValues(ctx context.Context, category string) (map[string]string, error) {
	var records []database.SystemSetting
	if err := s.db.WithContext(ctx).
		Where("category = ?", category).
		Find(&records).Error; err != nil {
		return nil, err
	}
	values := make(map[string]string, len(records))
	for _, record := range records {
		values[record.Key] = record.Value
	}
	return values, nil
}

func sourceForValue(databaseValue, fallbackValue string) string {
	if strings.TrimSpace(databaseValue) != "" {
		return "database"
	}
	if strings.TrimSpace(fallbackValue) != "" {
		return "env"
	}
	return "none"
}

func applyStringOverride(target *string, value string) {
	if strings.TrimSpace(value) != "" {
		*target = strings.TrimSpace(value)
	}
}

func applyDurationOverride(target *time.Duration, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	parsed, err := time.ParseDuration(trimmed)
	if err == nil {
		*target = parsed
	}
}
