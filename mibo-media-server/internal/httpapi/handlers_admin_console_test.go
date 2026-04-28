package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/auth"
	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/progress"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
	"gorm.io/gorm"
)

func TestAdminConsoleSummarySuccess(t *testing.T) {
	handler, authHeader, db := newAdminConsoleTestServer(t, config.Config{})
	ctx := t.Context()
	if err := db.WithContext(ctx).Create(&database.MediaSource{Name: "Local", Provider: "local", StorageRef: "local", RootPath: "/media"}).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.Library{Name: "Movies", Type: "movie", MediaSourceID: 1, RootPath: "/media", Status: "active", ScannerEnabled: true}).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.CatalogItem{LibraryID: 1, Type: "movie", Title: "Example", AvailabilityStatus: "available"}).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/console", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	var summary adminConsoleSummaryResponse
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summary.Media.Libraries != 1 || summary.Media.CatalogItems != 1 || summary.Media.Movies != 1 {
		t.Fatalf("unexpected media summary: %#v", summary.Media)
	}
	if summary.Health.Database.Status != "ok" || summary.Health.Storage.Status != "ok" {
		t.Fatalf("unexpected health summary: %#v", summary.Health)
	}
	if len(summary.QuickAction) == 0 {
		t.Fatalf("expected quick actions")
	}
}

func TestAdminConsoleSummaryReturnsPartialWarnings(t *testing.T) {
	overrides := config.Config{Storage: config.StorageConfig{Provider: "missing"}}
	handler, authHeader, _ := newAdminConsoleTestServer(t, overrides)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/console", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected partial 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	data, _ := json.Marshal(env.Data)
	var summary adminConsoleSummaryResponse
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summary.Health.Storage.Status != "warning" || len(summary.Warnings) == 0 {
		t.Fatalf("expected storage warning, got %#v", summary)
	}
}

func TestAdminConsoleSummaryRequiresAuthentication(t *testing.T) {
	handler, _, _ := newAdminConsoleTestServer(t, config.Config{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/console", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func newAdminConsoleTestServer(t *testing.T, overrides config.Config) (http.Handler, string, *gorm.DB) {
	t.Helper()
	rootPath := t.TempDir()
	cfg := config.Config{
		HTTP:     config.HTTPConfig{Addr: ":8080"},
		Storage:  config.StorageConfig{Provider: "local"},
		Local:    config.LocalStorageConfig{RootPath: rootPath},
		Database: config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")},
		Worker:   config.WorkerConfig{Enabled: true},
	}
	if overrides.Storage.Provider != "" {
		cfg.Storage = overrides.Storage
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	searchSvc := search.NewService(db, librarySvc)
	progressSvc := progress.NewService(db, searchSvc)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	catalogSvc := catalog.NewService(db)
	playbackSvc := playback.NewService(db, registry)
	authHeader := createAuthHeader(t, t.Context(), authSvc)
	handler := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playbackSvc, progressSvc, searchSvc, nil, settingsSvc, catalogSvc)
	return handler, authHeader, db
}
