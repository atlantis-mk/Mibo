package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
	"github.com/atlan/mibo-media-server/internal/workflow"
)

func TestWorkflowStatusEndpoints(t *testing.T) {
	ctx := context.Background()
	handler, authHeader, workflowSvc := newWorkflowHTTPTestServer(t)
	run, _, err := workflowSvc.CreateOrReuseRun(ctx, workflow.CreateRunInput{RunKey: "library:1:manual_scan", LibraryID: 1, Reason: library.WorkflowReasonManualScan, ScopeKey: "library:1"})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if _, err := workflowSvc.CreateTask(ctx, run, workflow.CreateTaskInput{TaskKey: "task:workflow:http", TaskType: workflow.TaskTypeScanLibraryPath, Stage: workflow.StageScan, Resources: map[string]int{workflow.ResourceDBWrite: 1}}); err != nil {
		t.Fatalf("create task: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected list workflows status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+uintString(run.ID), nil)
	detailReq.Header.Set("Authorization", authHeader)
	detailRec := httptest.NewRecorder()
	handler.ServeHTTP(detailRec, detailReq)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("expected get workflow status 200, got %d body=%s", detailRec.Code, detailRec.Body.String())
	}

	diagnosticsReq := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/diagnostics", nil)
	diagnosticsReq.Header.Set("Authorization", authHeader)
	diagnosticsRec := httptest.NewRecorder()
	handler.ServeHTTP(diagnosticsRec, diagnosticsReq)
	if diagnosticsRec.Code != http.StatusOK {
		t.Fatalf("expected diagnostics status 200, got %d body=%s", diagnosticsRec.Code, diagnosticsRec.Body.String())
	}
}

func newWorkflowHTTPTestServer(t *testing.T) (http.Handler, string, *workflow.Service) {
	t.Helper()
	cfg := config.Config{HTTP: config.HTTPConfig{Addr: ":8080"}, Local: config.LocalStorageConfig{RootPath: t.TempDir()}, Database: config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")}, Worker: config.WorkerConfig{Enabled: true}}
	db, err := database.Open(cfg.Database)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	workflowSvc := workflow.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, nil, workflowSvc)
	searchSvc := search.NewService(db, librarySvc)
	progressSvc := progress.NewService(db, searchSvc)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	catalogSvc := catalog.NewService(db)
	playbackSvc := playback.NewService(db, registry)
	authHeader := createAuthHeader(t, context.Background(), authSvc)
	handler := New(cfg, db, registry, authSvc, librarySvc, nil, playbackSvc, progressSvc, searchSvc, nil, settingsSvc, catalogSvc, workflowSvc)
	return handler, authHeader, workflowSvc
}
