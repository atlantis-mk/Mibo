package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/auth"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/metadata"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/progress"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
	"gorm.io/gorm"
)

func testCatalogMigrationSettingsEndpoints(t *testing.T) {
	router, db, authSvc, settingsSvc := newCatalogMigrationTestRouter(t)
	ctx := context.Background()

	t.Run("require auth before get and put", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/settings/catalog-migration", nil)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for unauthenticated get, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		recorder = httptest.NewRecorder()
		request = httptest.NewRequest(http.MethodPut, "/api/v1/settings/catalog-migration", strings.NewReader(`{"catalog_backfill_completed_at":"not-a-timestamp","catalog_read_enabled":true}`))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for unauthenticated put, got %d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	authHeader := createAuthHeader(t, ctx, authSvc)
	backfillAt := time.Date(2026, time.April, 25, 10, 0, 0, 0, time.UTC)
	cleanupAt := time.Date(2026, time.April, 26, 12, 30, 0, 0, time.UTC)
	validationAt := time.Date(2026, time.April, 26, 8, 0, 0, 0, time.UTC)
	if _, err := settingsSvc.UpdateCatalogMigrationState(ctx, settings.UpdateCatalogMigrationStateInput{
		CatalogBackfillCompletedAt:   &backfillAt,
		CatalogReadEnabled:           true,
		CatalogValidationCompletedAt: &validationAt,
		LegacyCleanupCompletedAt:     &cleanupAt,
	}); err != nil {
		t.Fatalf("seed catalog migration state: %v", err)
	}

	t.Run("get returns stored typed state", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/settings/catalog-migration", nil)
		request.Header.Set("Authorization", authHeader)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		var response struct {
			Data settings.CatalogMigrationState `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode catalog migration get response: %v", err)
		}
		assertCatalogMigrationHTTPTimeEqual(t, response.Data.CatalogBackfillCompletedAt, backfillAt)
		if !response.Data.CatalogReadEnabled {
			t.Fatal("expected catalog_read_enabled=true")
		}
		assertCatalogMigrationHTTPTimeEqual(t, response.Data.CatalogValidationCompletedAt, validationAt)
		assertCatalogMigrationHTTPTimeEqual(t, response.Data.LegacyCleanupCompletedAt, cleanupAt)
	})

	t.Run("put validates timestamps and only persists allowed keys", func(t *testing.T) {
		if err := db.WithContext(ctx).Create(&database.SystemSetting{Category: "metadata", Key: "tmdb_api_key", Value: "keep-me", IsSecret: true}).Error; err != nil {
			t.Fatalf("seed metadata setting: %v", err)
		}

		invalidRecorder := httptest.NewRecorder()
		invalidRequest := httptest.NewRequest(http.MethodPut, "/api/v1/settings/catalog-migration", strings.NewReader(`{"catalog_backfill_completed_at":"bad-value","catalog_read_enabled":true}`))
		invalidRequest.Header.Set("Authorization", authHeader)
		invalidRequest.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(invalidRecorder, invalidRequest)
		if invalidRecorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid timestamp, got %d body=%s", invalidRecorder.Code, invalidRecorder.Body.String())
		}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPut, "/api/v1/settings/catalog-migration", strings.NewReader(`{"catalog_backfill_completed_at":"2026-04-27T09:00:00Z","catalog_validation_completed_at":"2026-04-27T10:00:00Z","catalog_read_enabled":false}`))
		request.Header.Set("Authorization", authHeader)
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		var response struct {
			Data settings.CatalogMigrationState `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode catalog migration put response: %v", err)
		}
		if response.Data.CatalogReadEnabled {
			t.Fatal("expected catalog_read_enabled=false after update")
		}
		assertCatalogMigrationHTTPTimeEqual(t, response.Data.CatalogBackfillCompletedAt, time.Date(2026, time.April, 27, 9, 0, 0, 0, time.UTC))
		assertCatalogMigrationHTTPTimeEqual(t, response.Data.CatalogValidationCompletedAt, time.Date(2026, time.April, 27, 10, 0, 0, 0, time.UTC))
		if response.Data.LegacyCleanupCompletedAt != nil {
			t.Fatalf("expected omitted cleanup timestamp to clear, got %v", response.Data.LegacyCleanupCompletedAt)
		}

		var metadataSetting database.SystemSetting
		if err := db.WithContext(ctx).Where("category = ? AND key = ?", "metadata", "tmdb_api_key").First(&metadataSetting).Error; err != nil {
			t.Fatalf("reload metadata setting: %v", err)
		}
		if metadataSetting.Value != "keep-me" || !metadataSetting.IsSecret {
			t.Fatalf("expected unrelated setting to remain untouched, got %#v", metadataSetting)
		}

		var catalogSettings []database.SystemSetting
		if err := db.WithContext(ctx).Where("category = ?", "catalog_migration").Order("key asc").Find(&catalogSettings).Error; err != nil {
			t.Fatalf("reload catalog migration settings: %v", err)
		}
		if len(catalogSettings) != 3 {
			t.Fatalf("expected only three catalog migration rows after clearing cleanup timestamp, got %#v", catalogSettings)
		}
		if catalogSettings[0].Key != "catalog_backfill_completed_at" || catalogSettings[0].Value != "2026-04-27T09:00:00Z" {
			t.Fatalf("unexpected persisted backfill setting: %#v", catalogSettings[0])
		}
		if catalogSettings[1].Key != "catalog_read_enabled" || catalogSettings[1].Value != "false" {
			t.Fatalf("unexpected persisted read setting: %#v", catalogSettings[1])
		}
		if catalogSettings[2].Key != "catalog_validation_completed_at" || catalogSettings[2].Value != "2026-04-27T10:00:00Z" {
			t.Fatalf("unexpected persisted validation setting: %#v", catalogSettings[2])
		}
	})
}

func testCatalogMigrationSystemInfo(t *testing.T) {
	router, _, _, settingsSvc := newCatalogMigrationTestRouter(t)
	ctx := context.Background()
	backfillAt := time.Date(2026, time.April, 25, 10, 0, 0, 0, time.UTC)
	if _, err := settingsSvc.UpdateCatalogMigrationState(ctx, settings.UpdateCatalogMigrationStateInput{
		CatalogBackfillCompletedAt:   &backfillAt,
		CatalogReadEnabled:           true,
		CatalogValidationCompletedAt: &backfillAt,
	}); err != nil {
		t.Fatalf("seed catalog migration state: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/system/info", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Data struct {
			CatalogMigration struct {
				CatalogBackfillCompletedAt   *time.Time `json:"catalog_backfill_completed_at"`
				CatalogReadEnabled           bool       `json:"catalog_read_enabled"`
				CatalogValidationCompletedAt *time.Time `json:"catalog_validation_completed_at"`
				LegacyCleanupCompletedAt     *time.Time `json:"legacy_cleanup_completed_at"`
			} `json:"catalog_migration"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode system info response: %v", err)
	}
	assertCatalogMigrationHTTPTimeEqual(t, response.Data.CatalogMigration.CatalogBackfillCompletedAt, backfillAt)
	if !response.Data.CatalogMigration.CatalogReadEnabled {
		t.Fatal("expected catalog_read_enabled in system info")
	}
	assertCatalogMigrationHTTPTimeEqual(t, response.Data.CatalogMigration.CatalogValidationCompletedAt, backfillAt)
	if response.Data.CatalogMigration.LegacyCleanupCompletedAt != nil {
		t.Fatalf("expected nil cleanup timestamp in system info, got %v", response.Data.CatalogMigration.LegacyCleanupCompletedAt)
	}
	if strings.Contains(recorder.Body.String(), "tmdb_api_key") {
		t.Fatalf("expected system info to avoid unrelated settings leakage, got %s", recorder.Body.String())
	}
}

func newCatalogMigrationTestRouter(t *testing.T) (http.Handler, *gorm.DB, *auth.Service, *settings.Service) {
	t.Helper()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	storageRoot := filepath.Join(t.TempDir(), "storage-root")
	if err := os.MkdirAll(storageRoot, 0o755); err != nil {
		t.Fatalf("create storage root: %v", err)
	}

	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		Storage:  config.StorageConfig{Provider: "local"},
		Local:    config.LocalStorageConfig{RootPath: storageRoot},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settingsSvc), settingsSvc)

	return router, db, authSvc, settingsSvc
}

func assertCatalogMigrationHTTPTimeEqual(t *testing.T, actual *time.Time, expected time.Time) {
	t.Helper()
	if actual == nil {
		t.Fatalf("expected timestamp %s, got nil", expected.Format(time.RFC3339))
	}
	if !actual.Equal(expected) {
		t.Fatalf("expected timestamp %s, got %s", expected.Format(time.RFC3339), actual.Format(time.RFC3339))
	}
}
