package settings

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestCatalogMigrationStateDefaultsToZeroValues(t *testing.T) {
	svc := newCatalogMigrationTestService(t)

	state, err := svc.GetCatalogMigrationState(context.Background())
	if err != nil {
		t.Fatalf("get catalog migration state: %v", err)
	}
	if state.CatalogBackfillCompletedAt != nil {
		t.Fatalf("expected nil backfill timestamp, got %v", state.CatalogBackfillCompletedAt)
	}
	if state.CatalogReadEnabled {
		t.Fatal("expected catalog_read_enabled to stay false before validation gate completes")
	}
	if state.CatalogValidationCompletedAt != nil {
		t.Fatalf("expected nil validation timestamp, got %v", state.CatalogValidationCompletedAt)
	}
	if state.LegacyCleanupCompletedAt != nil {
		t.Fatalf("expected nil cleanup timestamp, got %v", state.LegacyCleanupCompletedAt)
	}
}

func TestCatalogMigrationStateDefaultsReadDisabledForLegacyOnlyData(t *testing.T) {
	svc := newCatalogMigrationTestService(t)
	ctx := context.Background()
	legacyItem := database.MediaItem{LibraryID: 1, Type: "movie", Title: "Legacy", SourcePath: "/legacy.mkv", MatchStatus: "pending", Status: "ready"}
	if err := svc.db.WithContext(ctx).Create(&legacyItem).Error; err != nil {
		t.Fatalf("create legacy item: %v", err)
	}

	state, err := svc.GetCatalogMigrationState(ctx)
	if err != nil {
		t.Fatalf("get catalog migration state: %v", err)
	}
	if state.CatalogReadEnabled {
		t.Fatal("expected catalog_read_enabled to stay false for legacy-only data")
	}
}

func TestCatalogMigrationStateDefaultsReadDisabledForCatalogDataWithoutValidation(t *testing.T) {
	svc := newCatalogMigrationTestService(t)
	ctx := context.Background()
	catalogItem := database.CatalogItem{LibraryID: 1, Type: "movie", Title: "Catalog", AvailabilityStatus: "available", GovernanceStatus: "matched"}
	if err := svc.db.WithContext(ctx).Create(&catalogItem).Error; err != nil {
		t.Fatalf("create catalog item: %v", err)
	}

	state, err := svc.GetCatalogMigrationState(ctx)
	if err != nil {
		t.Fatalf("get catalog migration state: %v", err)
	}
	if state.CatalogReadEnabled {
		t.Fatal("expected catalog_read_enabled to stay false until validation completes")
	}
}

func TestCatalogMigrationStateDefaultsReadEnabledAfterValidationCompletes(t *testing.T) {
	svc := newCatalogMigrationTestService(t)
	ctx := context.Background()
	validatedAt := time.Date(2026, time.April, 26, 9, 0, 0, 0, time.UTC)
	if err := svc.db.WithContext(ctx).Create(&database.SystemSetting{
		Category: catalogMigrationCategory,
		Key:      catalogValidationCompletedAtKey,
		Value:    validatedAt.Format(time.RFC3339),
	}).Error; err != nil {
		t.Fatalf("seed validation timestamp: %v", err)
	}

	state, err := svc.GetCatalogMigrationState(ctx)
	if err != nil {
		t.Fatalf("get catalog migration state: %v", err)
	}
	if !state.CatalogReadEnabled {
		t.Fatal("expected catalog_read_enabled to default true after validation completes")
	}
	assertCatalogMigrationTimeEqual(t, state.CatalogValidationCompletedAt, validatedAt)
}

func TestCatalogMigrationStateRoundTripsPersistedValues(t *testing.T) {
	svc := newCatalogMigrationTestService(t)
	ctx := context.Background()
	backfillAt := time.Date(2026, time.April, 25, 10, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	validationAt := time.Date(2026, time.April, 25, 18, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	cleanupAt := time.Date(2026, time.April, 26, 11, 30, 0, 0, time.FixedZone("UTC-3", -3*60*60))

	updated, err := svc.UpdateCatalogMigrationState(ctx, UpdateCatalogMigrationStateInput{
		CatalogBackfillCompletedAt:   &backfillAt,
		CatalogReadEnabled:           true,
		CatalogValidationCompletedAt: &validationAt,
		LegacyCleanupCompletedAt:     &cleanupAt,
	})
	if err != nil {
		t.Fatalf("update catalog migration state: %v", err)
	}

	assertCatalogMigrationTimeEqual(t, updated.CatalogBackfillCompletedAt, backfillAt.UTC())
	if !updated.CatalogReadEnabled {
		t.Fatal("expected catalog_read_enabled to persist true")
	}
	assertCatalogMigrationTimeEqual(t, updated.CatalogValidationCompletedAt, validationAt.UTC())
	assertCatalogMigrationTimeEqual(t, updated.LegacyCleanupCompletedAt, cleanupAt.UTC())

	state, err := svc.GetCatalogMigrationState(ctx)
	if err != nil {
		t.Fatalf("reload catalog migration state: %v", err)
	}
	assertCatalogMigrationTimeEqual(t, state.CatalogBackfillCompletedAt, backfillAt.UTC())
	if !state.CatalogReadEnabled {
		t.Fatal("expected catalog_read_enabled to round-trip true")
	}
	assertCatalogMigrationTimeEqual(t, state.CatalogValidationCompletedAt, validationAt.UTC())
	assertCatalogMigrationTimeEqual(t, state.LegacyCleanupCompletedAt, cleanupAt.UTC())
	assertCatalogMigrationStoredValue(t, svc, catalogMigrationCategory, catalogBackfillCompletedAtKey, backfillAt.UTC().Format(time.RFC3339))
	assertCatalogMigrationStoredValue(t, svc, catalogMigrationCategory, catalogReadEnabledKey, "true")
	assertCatalogMigrationStoredValue(t, svc, catalogMigrationCategory, catalogValidationCompletedAtKey, validationAt.UTC().Format(time.RFC3339))
	assertCatalogMigrationStoredValue(t, svc, catalogMigrationCategory, legacyCleanupCompletedAtKey, cleanupAt.UTC().Format(time.RFC3339))
}

func TestCatalogMigrationStateRejectsInvalidStoredTimestamp(t *testing.T) {
	svc := newCatalogMigrationTestService(t)
	ctx := context.Background()
	if err := svc.db.WithContext(ctx).Create(&database.SystemSetting{
		Category: catalogMigrationCategory,
		Key:      catalogBackfillCompletedAtKey,
		Value:    "not-a-timestamp",
	}).Error; err != nil {
		t.Fatalf("seed invalid system setting: %v", err)
	}

	_, err := svc.GetCatalogMigrationState(ctx)
	if err == nil {
		t.Fatal("expected invalid timestamp to be rejected")
	}
	if got := err.Error(); got != "parse catalog_backfill_completed_at: parsing time \"not-a-timestamp\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"not-a-timestamp\" as \"2006\"" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newCatalogMigrationTestService(t *testing.T) *Service {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	return NewService(db, config.MetadataConfig{})
}

func assertCatalogMigrationTimeEqual(t *testing.T, actual *time.Time, expected time.Time) {
	t.Helper()
	if actual == nil {
		t.Fatalf("expected timestamp %s, got nil", expected.Format(time.RFC3339))
	}
	if !actual.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected.Format(time.RFC3339), actual.Format(time.RFC3339))
	}
	if actual.Location() != time.UTC {
		t.Fatalf("expected UTC timestamp, got %s", actual.Location())
	}
}

func assertCatalogMigrationStoredValue(t *testing.T, svc *Service, category, key, expected string) {
	t.Helper()
	var record database.SystemSetting
	if err := svc.db.WithContext(context.Background()).Where("category = ? AND key = ?", category, key).First(&record).Error; err != nil {
		t.Fatalf("load system setting %s/%s: %v", category, key, err)
	}
	if record.Value != expected {
		t.Fatalf("expected %s/%s=%q, got %q", category, key, expected, record.Value)
	}
	if record.IsSecret {
		t.Fatalf("expected %s/%s to be non-secret", category, key)
	}
}
