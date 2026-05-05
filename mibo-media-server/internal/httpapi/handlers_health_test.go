package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/auth"
	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/health"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/progress"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
)

func TestHealthIssuesExposeTechnicalDetails(t *testing.T) {
	handler, authHeader, db, _ := newHealthTestServer(t)
	source := createHTTPHealthSource(t, db)
	libraryRecord := createHTTPHealthLibrary(t, db, source.ID)
	createHTTPFailedJob(t, db, `{"library_id":`+uintString(libraryRecord.ID)+`}`, "openlist request failed: ErrorCode: 4002 ,Error: captcha_invalid ,ErrorDescription: captcha_token expired")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/issues", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	issues := decodeHealthIssues(t, rec)
	if len(issues) != 1 {
		t.Fatalf("expected one issue, got %#v", issues)
	}
	if issues[0].ReasonCode != health.ReasonStorageAuthExpired || issues[0].TechnicalDetail.ErrorMessage == "" {
		t.Fatalf("expected classified issue with technical detail: %#v", issues[0])
	}
}

func TestHealthSummaryEmptyWhenHealthy(t *testing.T) {
	handler, authHeader, _, _ := newHealthTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/summary", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	summary := decodeHealthSummary(t, rec)
	if summary.Status != health.OverallStatusHealthy || summary.IssueCount != 0 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestValidateMediaSource(t *testing.T) {
	handler, authHeader, db, root := newHealthTestServer(t)
	source := database.MediaSource{Name: "Local", Provider: "local", StorageRef: "/", RootPath: root}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/media-sources/"+uintString(source.ID)+"/validate", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	result := decodeValidationResult(t, rec)
	if result.Status != "ok" || result.MediaSourceID != source.ID {
		t.Fatalf("unexpected validation result: %#v", result)
	}
}

func TestRescanHealthIssueLibraries(t *testing.T) {
	handler, authHeader, db, _ := newHealthTestServer(t)
	source := createHTTPHealthSource(t, db)
	libraryRecord := createHTTPHealthLibrary(t, db, source.ID)
	createHTTPFailedJob(t, db, `{"library_id":`+uintString(libraryRecord.ID)+`}`, "captcha_token expired")
	issues := requestHealthIssues(t, handler, authHeader)
	if len(issues) != 1 {
		t.Fatalf("expected one issue, got %#v", issues)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/health/issues/"+issues[0].ID+"/rescan", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	result := decodeRescanResult(t, rec)
	if len(result.Jobs) != 1 || result.Jobs[0].Kind != library.JobKindSyncLibrary {
		t.Fatalf("unexpected rescan result: %#v", result)
	}
}

func TestIgnoreHealthIssue(t *testing.T) {
	handler, authHeader, db, _ := newHealthTestServer(t)
	source := createHTTPHealthSource(t, db)
	libraryRecord := createHTTPHealthLibrary(t, db, source.ID)
	createHTTPFailedJob(t, db, `{"library_id":`+uintString(libraryRecord.ID)+`}`, "captcha_token expired")
	issues := requestHealthIssues(t, handler, authHeader)
	if len(issues) != 1 {
		t.Fatalf("expected one issue, got %#v", issues)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/health/issues/"+issues[0].ID+"/ignore", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	issues = requestHealthIssues(t, handler, authHeader)
	if len(issues) != 0 {
		t.Fatalf("expected ignored issue hidden, got %#v", issues)
	}
}

func newHealthTestServer(t *testing.T) (http.Handler, string, *gorm.DB, string) {
	t.Helper()
	root := t.TempDir()
	cfg := config.Config{HTTP: config.HTTPConfig{Addr: ":8080"}, Local: config.LocalStorageConfig{RootPath: root}, OpenList: config.OpenListConfig{BaseURL: "http://127.0.0.1:5244"}, Database: config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")}, Worker: config.WorkerConfig{Enabled: true}}
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
	healthSvc := health.NewService(db, registry, librarySvc, cfg.OpenList.BaseURL)
	authHeader := loginTestUser(t, authSvc, "health-user", "password123")
	handler := New(cfg, db, registry, authSvc, librarySvc, nil, playbackSvc, progressSvc, searchSvc, nil, settingsSvc, catalogSvc, healthSvc)
	return handler, authHeader, db, root
}

func createHTTPHealthSource(t *testing.T, db *gorm.DB) database.MediaSource {
	t.Helper()
	source := database.MediaSource{Name: "PikPak", Provider: "openlist", StorageRef: "/", RootPath: "/"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	return source
}

func createHTTPHealthLibrary(t *testing.T, db *gorm.DB, mediaSourceID uint) database.Library {
	t.Helper()
	record := database.Library{Name: "电影", Type: "movies", MediaSourceID: mediaSourceID, RootPath: "/My Pack/电影", Status: "error", ScannerEnabled: true}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	return record
}

func createHTTPFailedJob(t *testing.T, db *gorm.DB, payloadJSON string, errorMessage string) database.WorkflowTask {
	t.Helper()
	now := time.Now().UTC()
	run := database.WorkflowRun{RunKey: fmt.Sprintf("http-health-test-%d", now.UnixNano()), LibraryID: libraryIDFromPayload(payloadJSON), Reason: library.WorkflowReasonTargetedRefresh, Status: workflow.RunStatusFailed, ScopeKey: "test", CreatedAt: now, UpdatedAt: now, FinishedAt: &now}
	if run.LibraryID == 0 {
		run.LibraryID = 1
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatalf("create failed workflow run: %v", err)
	}
	task := database.WorkflowTask{RunID: run.ID, LibraryID: run.LibraryID, TaskKey: fmt.Sprintf("http-health-task-%d", now.UnixNano()), TaskType: workflow.TaskTypeScanLibraryPath, Stage: workflow.StageScan, Status: workflow.TaskStatusFailed, ScopeKey: run.ScopeKey, PayloadJSON: payloadJSON, ErrorMessage: errorMessage, Attempts: 1, AvailableAt: now, CreatedAt: now, UpdatedAt: now, FinishedAt: &now}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create failed workflow task: %v", err)
	}
	return task
}

func libraryIDFromPayload(payloadJSON string) uint {
	var payload map[string]any
	_ = json.Unmarshal([]byte(payloadJSON), &payload)
	if value, ok := payload["library_id"].(float64); ok && value > 0 {
		return uint(value)
	}
	return 0
}

func requestHealthIssues(t *testing.T, handler http.Handler, authHeader string) []health.Issue {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/issues", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	return decodeHealthIssues(t, rec)
}

func decodeHealthIssues(t *testing.T, rec *httptest.ResponseRecorder) []health.Issue {
	t.Helper()
	var result []health.Issue
	decodeEnvelopeData(t, rec, &result)
	return result
}

func decodeHealthSummary(t *testing.T, rec *httptest.ResponseRecorder) health.Summary {
	t.Helper()
	var result health.Summary
	decodeEnvelopeData(t, rec, &result)
	return result
}

func decodeValidationResult(t *testing.T, rec *httptest.ResponseRecorder) health.ValidationResult {
	t.Helper()
	var result health.ValidationResult
	decodeEnvelopeData(t, rec, &result)
	return result
}

func decodeRescanResult(t *testing.T, rec *httptest.ResponseRecorder) health.RescanResult {
	t.Helper()
	var result health.RescanResult
	decodeEnvelopeData(t, rec, &result)
	return result
}

func decodeEnvelopeData(t *testing.T, rec *httptest.ResponseRecorder, target any) {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("decode data: %v", err)
	}
}
