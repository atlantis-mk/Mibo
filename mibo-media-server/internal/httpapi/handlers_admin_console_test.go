package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/auth"
	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
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
	handler, authHeader, _ := newAdminConsoleTestServer(t, config.Config{})

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
	if summary.Health.Storage.Status != "not_configured" || summary.Health.Storage.Message == "" {
		t.Fatalf("expected storage not configured, got %#v", summary.Health.Storage)
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

func TestAdminIngestDiagnosticsAndRetry(t *testing.T) {
	handler, authHeader, db := newAdminConsoleTestServer(t, config.Config{})
	ctx := t.Context()
	libraryRecord := database.Library{Name: "Movies", Type: "movie", RootPath: "/media", Status: "active"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	file := database.InventoryFile{LibraryID: libraryRecord.ID, StorageProvider: "local", StoragePath: "/media/Movie.mkv", ContentClass: "video", Status: "available"}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	providerInstanceID := uint(44)
	condition := database.IngestCondition{UnitKey: "inventory_file:1", LibraryID: libraryRecord.ID, InventoryFileID: &file.ID, ConditionType: ingest.ConditionProbed, Status: ingest.ConditionStatusFailed, Reason: "probe_failed", Message: "probe failed", Severity: ingest.SeverityError, ProviderInstanceID: &providerInstanceID}
	if err := db.WithContext(ctx).Create(&condition).Error; err != nil {
		t.Fatalf("create condition: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ingest/diagnostics", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected diagnostics 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode diagnostics envelope: %v", err)
	}
	data, _ := json.Marshal(env.Data)
	var diagnostics ingest.DiagnosticsResult
	if err := json.Unmarshal(data, &diagnostics); err != nil {
		t.Fatalf("decode diagnostics: %v", err)
	}
	if diagnostics.Summary.Failed != 1 || diagnostics.Summary.RetryEligible != 1 || len(diagnostics.Stages) != 1 || diagnostics.Stages[0].StoragePath != file.StoragePath || diagnostics.Stages[0].ProviderInstanceID == nil || *diagnostics.Stages[0].ProviderInstanceID != providerInstanceID {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}

	retryReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ingest/stages/"+strconv.FormatUint(uint64(condition.ID), 10)+"/retry", nil)
	retryReq.Header.Set("Authorization", authHeader)
	retryRec := httptest.NewRecorder()
	handler.ServeHTTP(retryRec, retryReq)
	if retryRec.Code != http.StatusAccepted {
		t.Fatalf("expected retry 202, got %d: %s", retryRec.Code, retryRec.Body.String())
	}
	var dirty database.IngestDirtyUnit
	if err := db.WithContext(ctx).Where("inventory_file_id = ? AND status = ?", file.ID, ingest.DirtyStatusDirty).First(&dirty).Error; err != nil {
		t.Fatalf("expected dirty retry unit: %v", err)
	}
}

func TestAdminIngestDiagnosticsFiltersAndMarksStale(t *testing.T) {
	handler, authHeader, db := newAdminConsoleTestServer(t, config.Config{})
	ctx := t.Context()
	now := time.Now().UTC()
	staleAfter := now.Add(-time.Minute)
	conditions := []database.IngestCondition{
		{UnitKey: "catalog_item:1", LibraryID: 1, ConditionType: ingest.ConditionMetadataMatched, Status: ingest.ConditionStatusReviewRequired, Reason: "no_candidate", Message: "needs review", Severity: ingest.SeverityWarning},
		{UnitKey: "inventory_file:2", LibraryID: 1, ConditionType: ingest.ConditionProbed, Status: ingest.ConditionStatusRunning, Reason: "probing", Message: "still running", Severity: ingest.SeverityInfo, StaleAfter: &staleAfter},
	}
	if err := db.WithContext(ctx).Create(&conditions).Error; err != nil {
		t.Fatalf("create conditions: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ingest/diagnostics?status=review_required", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var diagnostics ingest.DiagnosticsResult
	decodeEnvelopeData(t, rec, &diagnostics)
	if len(diagnostics.Stages) != 1 || diagnostics.Stages[0].Status != ingest.ConditionStatusReviewRequired {
		t.Fatalf("expected filtered review-required diagnostics, got %#v", diagnostics)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/ingest/diagnostics", nil)
	req.Header.Set("Authorization", authHeader)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	decodeEnvelopeData(t, rec, &diagnostics)
	if diagnostics.Summary.Stale != 1 {
		t.Fatalf("expected one stale diagnostic, got %#v", diagnostics)
	}
}

func TestAdminIngestRetryDoesNotDuplicateActiveRunningStage(t *testing.T) {
	handler, authHeader, db := newAdminConsoleTestServer(t, config.Config{})
	ctx := t.Context()
	condition := database.IngestCondition{UnitKey: "inventory_file:1", LibraryID: 1, ConditionType: ingest.ConditionProbed, Status: ingest.ConditionStatusFailed, Reason: "probe_failed", Message: "probe failed", Severity: ingest.SeverityError}
	if err := db.WithContext(ctx).Create(&condition).Error; err != nil {
		t.Fatalf("create condition: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ingest/stages/"+strconv.FormatUint(uint64(condition.ID), 10)+"/retry", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected active retry 202, got %d: %s", rec.Code, rec.Body.String())
	}
	var result ingest.RetryStageResult
	decodeEnvelopeData(t, rec, &result)
	if result.Status != "queued" {
		t.Fatalf("expected retry to queue pending work, got %#v", result)
	}
	var dirtyCount int64
	if err := db.WithContext(ctx).Model(&database.IngestDirtyUnit{}).Count(&dirtyCount).Error; err != nil {
		t.Fatalf("count dirty units: %v", err)
	}
	if dirtyCount != 1 {
		t.Fatalf("expected one dirty retry work item, got %d", dirtyCount)
	}
}

func TestAdminIngestReconcileMarksLibraryScopeDirty(t *testing.T) {
	handler, authHeader, db := newAdminConsoleTestServer(t, config.Config{})
	ctx := t.Context()
	libraryRecord := database.Library{Name: "Movies", Type: "movie", RootPath: "/media", Status: "active"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	body := []byte(`{"library_id":` + strconv.FormatUint(uint64(libraryRecord.ID), 10) + `,"root_path":"/media/Movies"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ingest/reconcile", bytes.NewReader(body))
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	var dirty database.IngestDirtyUnit
	if err := db.WithContext(ctx).Where("library_id = ? AND scope_kind = ? AND root_path = ?", libraryRecord.ID, ingest.ScopeKindLibrary, "/media/Movies").First(&dirty).Error; err != nil {
		t.Fatalf("expected dirty library scope: %v", err)
	}
}

func newAdminConsoleTestServer(t *testing.T, overrides config.Config) (http.Handler, string, *gorm.DB) {
	t.Helper()
	rootPath := t.TempDir()
	cfg := config.Config{
		HTTP:     config.HTTPConfig{Addr: ":8080"},
		Local:    config.LocalStorageConfig{RootPath: rootPath},
		Database: config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")},
		Worker:   config.WorkerConfig{Enabled: true},
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, nil)
	searchSvc := search.NewService(db, librarySvc)
	progressSvc := progress.NewService(db, searchSvc)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	catalogSvc := catalog.NewService(db)
	playbackSvc := playback.NewService(db, registry)
	authHeader := createAuthHeader(t, t.Context(), authSvc)
	handler := New(cfg, db, registry, authSvc, librarySvc, nil, playbackSvc, progressSvc, searchSvc, nil, settingsSvc, catalogSvc)
	return handler, authHeader, db
}
