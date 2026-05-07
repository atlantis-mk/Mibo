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
	tmdbAPIKeyKey                     = "tmdb_api_key"
	tmdbBaseURLKey                    = "tmdb_base_url"
	tmdbImageBaseURLKey               = "tmdb_image_base_url"
	tmdbLanguageKey                   = "tmdb_language"
	tmdbTimeoutKey                    = "tmdb_timeout"
	tvdbAPIKeyKey                     = "tvdb_api_key"
	tvdbBaseURLKey                    = "tvdb_base_url"
	tvdbLanguageKey                   = "tvdb_language"
	tvdbTimeoutKey                    = "tvdb_timeout"
	metatubeTokenKey                  = "metatube_token"
	metatubeBaseURLKey                = "metatube_base_url"
	metatubeUpstreamProviderFilterKey = "metatube_upstream_provider_filter"
	metatubeFallbackEnabledKey        = "metatube_fallback_enabled"
	metatubeTimeoutKey                = "metatube_timeout"
)

const refreshIntervalKey = "refresh_interval_hours"

type Service struct {
	db       *gorm.DB
	fallback config.MetadataConfig
}

type MetadataProviderSettings struct {
	Configured             bool   `json:"configured"`
	APIKeyMasked           bool   `json:"api_key_masked"`
	BaseURL                string `json:"base_url"`
	ImageBaseURL           string `json:"image_base_url,omitempty"`
	Language               string `json:"language"`
	Timeout                string `json:"timeout"`
	Source                 string `json:"source"`
	Implementation         string `json:"implementation"`
	UpstreamProviderFilter string `json:"upstream_provider_filter,omitempty"`
	FallbackEnabled        bool   `json:"fallback_enabled,omitempty"`
}

type MetadataProviderInput struct {
	APIKey                 string `json:"api_key"`
	ClearAPIKey            bool   `json:"clear_api_key"`
	BaseURL                string `json:"base_url"`
	ImageBaseURL           string `json:"image_base_url"`
	Language               string `json:"language"`
	Timeout                string `json:"timeout"`
	UpstreamProviderFilter string `json:"upstream_provider_filter"`
	FallbackEnabled        *bool  `json:"fallback_enabled,omitempty"`
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

func (s *Service) ResolveTMDBConfig(ctx context.Context) (config.TMDBConfig, string, error) {
	values, err := s.loadCategoryValues(ctx, metadataCategory)
	if err != nil {
		return config.TMDBConfig{}, "none", err
	}
	resolved := s.fallback.TMDB
	source := sourceForValue(values[tmdbAPIKeyKey], resolved.APIKey)
	applyStringOverride(&resolved.APIKey, values[tmdbAPIKeyKey])
	applyStringOverride(&resolved.BaseURL, values[tmdbBaseURLKey])
	applyStringOverride(&resolved.ImageBaseURL, values[tmdbImageBaseURLKey])
	applyStringOverride(&resolved.Language, values[tmdbLanguageKey])
	applyDurationOverride(&resolved.Timeout, values[tmdbTimeoutKey])
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

func deleteCategorySettingWithDB(ctx context.Context, db *gorm.DB, category, key string) error {
	return db.WithContext(ctx).
		Where("category = ? AND key = ?", category, key).
		Delete(&database.SystemSetting{}).Error
}

func (s *Service) loadCategoryValues(ctx context.Context, category string) (map[string]string, error) {
	return loadCategoryValuesWithDB(ctx, s.db, category)
}

func loadCategoryValuesWithDB(ctx context.Context, db *gorm.DB, category string) (map[string]string, error) {
	var records []database.SystemSetting
	if err := db.WithContext(ctx).
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
