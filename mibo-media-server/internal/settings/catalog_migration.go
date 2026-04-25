package settings

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

const catalogMigrationCategory = "catalog_migration"

const (
	catalogBackfillCompletedAtKey = "catalog_backfill_completed_at"
	catalogReadEnabledKey         = "catalog_read_enabled"
	legacyCleanupCompletedAtKey   = "legacy_cleanup_completed_at"
)

type CatalogMigrationState struct {
	CatalogBackfillCompletedAt *time.Time `json:"catalog_backfill_completed_at"`
	CatalogReadEnabled         bool       `json:"catalog_read_enabled"`
	LegacyCleanupCompletedAt   *time.Time `json:"legacy_cleanup_completed_at"`
}

type UpdateCatalogMigrationStateInput struct {
	CatalogBackfillCompletedAt *time.Time `json:"catalog_backfill_completed_at"`
	CatalogReadEnabled         bool       `json:"catalog_read_enabled"`
	LegacyCleanupCompletedAt   *time.Time `json:"legacy_cleanup_completed_at"`
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
	readEnabled, err := parseCatalogMigrationBool(values[catalogReadEnabledKey], catalogReadEnabledKey)
	if err != nil {
		return CatalogMigrationState{}, err
	}
	cleanupCompletedAt, err := parseCatalogMigrationOptionalTimestamp(values[legacyCleanupCompletedAtKey], legacyCleanupCompletedAtKey)
	if err != nil {
		return CatalogMigrationState{}, err
	}

	return CatalogMigrationState{
		CatalogBackfillCompletedAt: backfillCompletedAt,
		CatalogReadEnabled:         readEnabled,
		LegacyCleanupCompletedAt:   cleanupCompletedAt,
	}, nil
}

func (s *Service) UpdateCatalogMigrationState(ctx context.Context, input UpdateCatalogMigrationStateInput) (CatalogMigrationState, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := upsertOrDeleteCatalogMigrationTimestamp(ctx, tx, catalogBackfillCompletedAtKey, input.CatalogBackfillCompletedAt); err != nil {
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

func parseCatalogMigrationBool(value, key string) (bool, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(trimmed)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}
