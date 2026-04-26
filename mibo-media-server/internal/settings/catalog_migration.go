package settings

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

const catalogMigrationCategory = "catalog_migration"

const (
	catalogBackfillCompletedAtKey   = "catalog_backfill_completed_at"
	catalogReadEnabledKey           = "catalog_read_enabled"
	catalogValidationCompletedAtKey = "catalog_validation_completed_at"
	legacyCleanupCompletedAtKey     = "legacy_cleanup_completed_at"
)

type CatalogMigrationState struct {
	CatalogBackfillCompletedAt   *time.Time `json:"catalog_backfill_completed_at"`
	CatalogReadEnabled           bool       `json:"catalog_read_enabled"`
	CatalogValidationCompletedAt *time.Time `json:"catalog_validation_completed_at"`
	LegacyCleanupCompletedAt     *time.Time `json:"legacy_cleanup_completed_at"`
}

type UpdateCatalogMigrationStateInput struct {
	CatalogBackfillCompletedAt   *time.Time `json:"catalog_backfill_completed_at"`
	CatalogReadEnabled           bool       `json:"catalog_read_enabled"`
	CatalogValidationCompletedAt *time.Time `json:"catalog_validation_completed_at"`
	LegacyCleanupCompletedAt     *time.Time `json:"legacy_cleanup_completed_at"`
}

func (s *Service) GetCatalogMigrationState(ctx context.Context) (CatalogMigrationState, error) {
	values, err := s.loadCategoryValues(ctx, catalogMigrationCategory)
	if err != nil {
		return CatalogMigrationState{}, err
	}

	backfillCompletedAt, err := parseCatalogMigrationOptionalTimestamp(values[catalogBackfillCompletedAtKey], catalogBackfillCompletedAtKey)
	if err != nil {
		return CatalogMigrationState{}, err
	}
	readEnabled, hasExplicitReadEnabled, err := parseCatalogMigrationBool(values[catalogReadEnabledKey], catalogReadEnabledKey)
	if err != nil {
		return CatalogMigrationState{}, err
	}
	validationCompletedAt, err := parseCatalogMigrationOptionalTimestamp(values[catalogValidationCompletedAtKey], catalogValidationCompletedAtKey)
	if err != nil {
		return CatalogMigrationState{}, err
	}
	cleanupCompletedAt, err := parseCatalogMigrationOptionalTimestamp(values[legacyCleanupCompletedAtKey], legacyCleanupCompletedAtKey)
	if err != nil {
		return CatalogMigrationState{}, err
	}
	if !hasExplicitReadEnabled {
		readEnabled, err = s.defaultCatalogReadEnabled(ctx, validationCompletedAt, cleanupCompletedAt)
		if err != nil {
			return CatalogMigrationState{}, err
		}
	}

	return CatalogMigrationState{
		CatalogBackfillCompletedAt:   backfillCompletedAt,
		CatalogReadEnabled:           readEnabled,
		CatalogValidationCompletedAt: validationCompletedAt,
		LegacyCleanupCompletedAt:     cleanupCompletedAt,
	}, nil
}

func (s *Service) UpdateCatalogMigrationState(ctx context.Context, input UpdateCatalogMigrationStateInput) (CatalogMigrationState, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := upsertOrDeleteCatalogMigrationTimestamp(ctx, tx, catalogBackfillCompletedAtKey, input.CatalogBackfillCompletedAt); err != nil {
			return err
		}
		if err := upsertOrDeleteCatalogMigrationTimestamp(ctx, tx, catalogValidationCompletedAtKey, input.CatalogValidationCompletedAt); err != nil {
			return err
		}
		if err := upsertOrDeleteCatalogMigrationTimestamp(ctx, tx, legacyCleanupCompletedAtKey, input.LegacyCleanupCompletedAt); err != nil {
			return err
		}
		return upsertCategorySettingWithDB(ctx, tx, catalogMigrationCategory, catalogReadEnabledKey, strconv.FormatBool(input.CatalogReadEnabled), false)
	})
	if err != nil {
		return CatalogMigrationState{}, err
	}

	return s.GetCatalogMigrationState(ctx)
}

func upsertOrDeleteCatalogMigrationTimestamp(ctx context.Context, db *gorm.DB, key string, value *time.Time) error {
	if value == nil {
		return deleteCategorySettingWithDB(ctx, db, catalogMigrationCategory, key)
	}
	return upsertCategorySettingWithDB(ctx, db, catalogMigrationCategory, key, value.UTC().Format(time.RFC3339), false)
}

func parseCatalogMigrationOptionalTimestamp(value, key string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", key, err)
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func parseCatalogMigrationBool(value, key string) (bool, bool, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false, false, nil
	}
	parsed, err := strconv.ParseBool(trimmed)
	if err != nil {
		return false, true, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, true, nil
}

func (s *Service) defaultCatalogReadEnabled(ctx context.Context, validationCompletedAt, cleanupCompletedAt *time.Time) (bool, error) {
	if validationCompletedAt != nil || cleanupCompletedAt != nil {
		return true, nil
	}

	var legacyCount int64
	if err := s.db.WithContext(ctx).Model(&database.MediaItem{}).Where("deleted_at IS NULL").Count(&legacyCount).Error; err != nil {
		return false, err
	}
	if legacyCount > 0 {
		return false, nil
	}

	return false, nil
}
