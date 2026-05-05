package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atlan/mibo-media-server/internal/auth"
	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/progress"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
	"gorm.io/gorm"
)

func TestLibraryPathAndPolicyHTTP(t *testing.T) {
	handler, authSvc, db, root := newLibraryPolicyTestServer(t)
	authHeader := createAuthHeader(t, context.Background(), authSvc)
	source := database.MediaSource{Name: "Local", Provider: "local", StorageRef: root, RootPath: root}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	libraryRecord := database.Library{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: root, Status: "active", ScannerEnabled: true}
	if err := db.Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}

	addReq := httptest.NewRequest(http.MethodPost, "/api/v1/libraries/"+uintString(libraryRecord.ID)+"/paths", strings.NewReader(`{"media_source_id":`+uintString(source.ID)+`,"root_path":"`+filepath.ToSlash(root)+`","display_name":"Primary"}`))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", authHeader)
	addRec := httptest.NewRecorder()
	handler.ServeHTTP(addRec, addReq)
	if addRec.Code != http.StatusCreated {
		t.Fatalf("expected add path 201, got %d: %s", addRec.Code, addRec.Body.String())
	}
	createdPath := decodeLibraryPathView(t, addRec)
	if createdPath.ID == 0 || createdPath.RootPath != filepath.ToSlash(root) || !createdPath.Enabled {
		t.Fatalf("unexpected path response: %#v", createdPath)
	}

	disableReq := httptest.NewRequest(http.MethodPatch, "/api/v1/libraries/"+uintString(libraryRecord.ID)+"/paths/"+uintString(createdPath.ID), strings.NewReader(`{"enabled":false}`))
	disableReq.Header.Set("Content-Type", "application/json")
	disableReq.Header.Set("Authorization", authHeader)
	disableRec := httptest.NewRecorder()
	handler.ServeHTTP(disableRec, disableReq)
	if disableRec.Code != http.StatusOK {
		t.Fatalf("expected disable path 200, got %d: %s", disableRec.Code, disableRec.Body.String())
	}
	if disabled := decodeLibraryPathView(t, disableRec); disabled.Enabled {
		t.Fatalf("expected disabled path, got %#v", disabled)
	}

	policyReq := httptest.NewRequest(http.MethodPut, "/api/v1/libraries/"+uintString(libraryRecord.ID)+"/policies/scan", strings.NewReader(`{"scanner_enabled":true,"realtime_monitor_enabled":false,"scheduled_refresh_enabled":true,"refresh_interval_hours":12,"ignore_hidden_files":true,"ignore_file_extensions":[".txt"],"min_file_size_bytes":1024,"sample_ignore_size_bytes":100,"inventory_probe_batch_enabled":false,"configurable_exclusion_rules":true}`))
	policyReq.Header.Set("Content-Type", "application/json")
	policyReq.Header.Set("Authorization", authHeader)
	policyRec := httptest.NewRecorder()
	handler.ServeHTTP(policyRec, policyReq)
	if policyRec.Code != http.StatusOK {
		t.Fatalf("expected update policy 200, got %d: %s", policyRec.Code, policyRec.Body.String())
	}
	policy := decodeLibraryScanPolicyView(t, policyRec)
	if policy.RealtimeMonitorEnabled || policy.RefreshIntervalHours != 12 || policy.MinFileSizeBytes != 1024 || policy.InventoryProbeBatchEnabled || len(policy.IgnoreFileExtensions) != 1 || policy.IgnoreFileExtensions[0] != ".txt" {
		t.Fatalf("unexpected scan policy: %#v", policy)
	}
}

func TestLibraryPathHTTPRejectsInvalidPath(t *testing.T) {
	handler, authSvc, db, root := newLibraryPolicyTestServer(t)
	authHeader := createAuthHeader(t, context.Background(), authSvc)
	source := database.MediaSource{Name: "Local", Provider: "local", StorageRef: root, RootPath: root}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	libraryRecord := database.Library{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: root, Status: "active", ScannerEnabled: true}
	if err := db.Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/libraries/"+uintString(libraryRecord.ID)+"/paths", strings.NewReader(`{"media_source_id":`+uintString(source.ID)+`,"root_path":"/outside"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid path 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func newLibraryPolicyTestServer(t *testing.T) (http.Handler, *auth.Service, *gorm.DB, string) {
	t.Helper()
	root := t.TempDir()
	cfg := config.Config{HTTP: config.HTTPConfig{Addr: ":8080"}, Local: config.LocalStorageConfig{RootPath: root}, Database: config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")}, Worker: config.WorkerConfig{Enabled: true}}
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
	handler := New(cfg, db, registry, authSvc, librarySvc, nil, playbackSvc, progressSvc, searchSvc, nil, settingsSvc, catalogSvc)
	return handler, authSvc, db, root
}

func decodeLibraryPathView(t *testing.T, rec *httptest.ResponseRecorder) library.LibraryPathView {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	var result library.LibraryPathView
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode path: %v", err)
	}
	return result
}

func decodeLibraryScanPolicyView(t *testing.T, rec *httptest.ResponseRecorder) library.LibraryScanPolicyView {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	var result library.LibraryScanPolicyView
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode policy: %v", err)
	}
	return result
}
