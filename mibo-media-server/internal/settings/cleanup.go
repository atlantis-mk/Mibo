package settings

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"gorm.io/gorm"
)

const cleanupCategory = "cleanup"

const (
	cleanupMissingEnabledKey   = "missing_cleanup_enabled"
	cleanupMissingRetentionKey = "missing_retention"
	cleanupMissingBatchSizeKey = "missing_cleanup_batch_size"
)

type CleanupSettings struct {
	MissingCleanupEnabled   bool   `json:"missing_cleanup_enabled"`
	MissingRetention        string `json:"missing_retention"`
	MissingRetentionSeconds int64  `json:"missing_retention_seconds"`
	MissingCleanupBatchSize int    `json:"missing_cleanup_batch_size"`
	CanRun                  bool   `json:"can_run"`
	Warning                 string `json:"warning"`
}

type UpdateCleanupSettingsInput struct {
	MissingCleanupEnabled   bool  `json:"missing_cleanup_enabled"`
	MissingRetentionSeconds int64 `json:"missing_retention_seconds"`
	MissingCleanupBatchSize int   `json:"missing_cleanup_batch_size"`
}

func (s *Service) GetCleanupSettings(ctx context.Context, fallback config.CleanupConfig) (CleanupSettings, error) {
	return ResolveCleanupSettings(ctx, s.db, fallback)
}

func (s *Service) UpdateCleanupSettings(ctx context.Context, fallback config.CleanupConfig, input UpdateCleanupSettingsInput) (CleanupSettings, error) {
	if input.MissingRetentionSeconds < 0 {
		return CleanupSettings{}, fmt.Errorf("missing_retention_seconds must be greater than or equal to 0")
	}
	if input.MissingCleanupBatchSize < 1 || input.MissingCleanupBatchSize > 1000 {
		return CleanupSettings{}, fmt.Errorf("missing_cleanup_batch_size must be between 1 and 1000")
	}
	retention := time.Duration(input.MissingRetentionSeconds) * time.Second
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		values := map[string]string{
			cleanupMissingEnabledKey:   strconv.FormatBool(input.MissingCleanupEnabled),
			cleanupMissingRetentionKey: retention.String(),
			cleanupMissingBatchSizeKey: strconv.Itoa(input.MissingCleanupBatchSize),
		}
		for key, value := range values {
			if err := upsertCategorySettingWithDB(ctx, tx, cleanupCategory, key, value, false); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return CleanupSettings{}, err
	}
	return s.GetCleanupSettings(ctx, fallback)
}

func ResolveCleanupSettings(ctx context.Context, db *gorm.DB, fallback config.CleanupConfig) (CleanupSettings, error) {
	values, err := loadCategoryValuesWithDB(ctx, db, cleanupCategory)
	if err != nil {
		return CleanupSettings{}, err
	}
	enabled := parseBoolValue(values[cleanupMissingEnabledKey], fallback.MissingCleanupEnabled)
	retention := fallback.MissingRetention
	if parsed, err := time.ParseDuration(values[cleanupMissingRetentionKey]); err == nil && parsed >= 0 {
		retention = parsed
	}
	batchSize := fallback.MissingCleanupBatchSize
	if parsed, err := strconv.Atoi(values[cleanupMissingBatchSizeKey]); err == nil && parsed >= 1 && parsed <= 1000 {
		batchSize = parsed
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return CleanupSettings{
		MissingCleanupEnabled:   enabled,
		MissingRetention:        retention.String(),
		MissingRetentionSeconds: int64(retention / time.Second),
		MissingCleanupBatchSize: batchSize,
		CanRun:                  enabled,
		Warning:                 "缺失媒体硬删除会永久删除目录、资产、库存、播放进度、收藏和人工治理数据。",
	}, nil
}
