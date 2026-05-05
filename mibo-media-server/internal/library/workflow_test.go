package library

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
)

func TestQueueLibraryWorkflowCreatesPathScopedScanTasks(t *testing.T) {
	ctx, db, svc := newWorkflowScanHarness(t)
	libraryRecord := createWorkflowScanLibrary(t, ctx, svc, "Movies", LibraryTypeMovies)

	run, reused, err := svc.QueueLibraryWorkflow(ctx, QueueWorkflowInput{LibraryID: libraryRecord.ID, Reason: WorkflowReasonManualScan, Priority: 10})
	if err != nil {
		t.Fatalf("queue workflow: %v", err)
	}
	if reused {
		t.Fatalf("expected new workflow run")
	}
	var tasks []database.WorkflowTask
	if err := db.WithContext(ctx).Where("run_id = ?", run.ID).Find(&tasks).Error; err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one path-scoped scan task, got %d", len(tasks))
	}
	if tasks[0].TaskType != workflow.TaskTypeScanLibraryPath || tasks[0].Stage != workflow.StageScan {
		t.Fatalf("unexpected scan task: %#v", tasks[0])
	}
}

func TestRunWorkflowScanLibraryPathCreatesMaterializeAndProjectionTasks(t *testing.T) {
	ctx, db, svc := newWorkflowScanHarness(t)
	libraryRecord := createWorkflowScanLibrary(t, ctx, svc, "Movies", LibraryTypeMovies)
	mustWriteFixtureFile(t, filepath.Join(libraryRecord.RootPath, "Movie A.2024.mkv"))
	run, _, err := svc.QueueLibraryWorkflow(ctx, QueueWorkflowInput{LibraryID: libraryRecord.ID, Reason: WorkflowReasonManualScan, Priority: 10})
	if err != nil {
		t.Fatalf("queue workflow: %v", err)
	}
	var task database.WorkflowTask
	if err := db.WithContext(ctx).Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeScanLibraryPath).First(&task).Error; err != nil {
		t.Fatalf("load scan task: %v", err)
	}
	if err := svc.RunWorkflowScanLibraryPath(ctx, task); err != nil {
		t.Fatalf("run workflow scan task: %v", err)
	}

	var materializeTasks int64
	if err := db.WithContext(ctx).Model(&database.WorkflowTask{}).Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeMaterializeCatalog).Count(&materializeTasks).Error; err != nil {
		t.Fatalf("count materialize tasks: %v", err)
	}
	if materializeTasks != 1 {
		t.Fatalf("expected one materialize task, got %d", materializeTasks)
	}
	var projectionTasks int64
	if err := db.WithContext(ctx).Model(&database.WorkflowTask{}).Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeRefreshProjection).Count(&projectionTasks).Error; err != nil {
		t.Fatalf("count projection tasks: %v", err)
	}
	if projectionTasks != 1 {
		t.Fatalf("expected one projection task, got %d", projectionTasks)
	}
}

func TestRunWorkflowScanLibraryPathAcceptsScopedSubdirectory(t *testing.T) {
	ctx, db, svc := newWorkflowScanHarness(t)
	libraryRecord := createWorkflowScanLibrary(t, ctx, svc, "Movies", LibraryTypeMovies)
	scopedRoot := filepath.Join(libraryRecord.RootPath, "Movie Pack")
	mustWriteFixtureFile(t, filepath.Join(scopedRoot, "Movie A.2024.mkv"))
	run, _, err := svc.QueueLibraryWorkflow(ctx, QueueWorkflowInput{LibraryID: libraryRecord.ID, RootPath: scopedRoot, Reason: WorkflowReasonTargetedRefresh, Priority: 10})
	if err != nil {
		t.Fatalf("queue workflow: %v", err)
	}
	var task database.WorkflowTask
	if err := db.WithContext(ctx).Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeScanLibraryPath).First(&task).Error; err != nil {
		t.Fatalf("load scan task: %v", err)
	}
	if err := svc.RunWorkflowScanLibraryPath(ctx, task); err != nil {
		t.Fatalf("run workflow scoped scan task: %v", err)
	}

	var projectionTask database.WorkflowTask
	if err := db.WithContext(ctx).Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeRefreshProjection).First(&projectionTask).Error; err != nil {
		t.Fatalf("load projection task: %v", err)
	}
	if !strings.Contains(projectionTask.PayloadJSON, scopedRoot) {
		t.Fatalf("expected projection task to keep scoped root %q, got %s", scopedRoot, projectionTask.PayloadJSON)
	}
}

func TestQueueLibraryScanWithReasonReturnsWorkflowCompatibilityJob(t *testing.T) {
	ctx, _, svc := newWorkflowScanHarness(t)
	libraryRecord := createWorkflowScanLibrary(t, ctx, svc, "Movies", LibraryTypeMovies)

	job, err := svc.QueueLibraryScanWithReason(ctx, libraryRecord.ID, WorkflowReasonManualScan)
	if err != nil {
		t.Fatalf("queue scan: %v", err)
	}
	if job.ID == 0 || job.Kind != JobKindSyncLibrary || job.Status != workflow.RunStatusQueued {
		t.Fatalf("unexpected compatibility job: %#v", job)
	}
}

func newWorkflowScanHarness(t *testing.T) (context.Context, *gorm.DB, *Service) {
	t.Helper()
	ctx := context.Background()
	rootPath := t.TempDir()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	cfg := config.Config{Local: config.LocalStorageConfig{RootPath: rootPath}, Worker: config.WorkerConfig{Enabled: true}}
	registry := providers.NewRegistry(cfg)
	workflowSvc := workflow.NewService(db)
	svc := NewService(cfg, db, registry, nil, ingest.NewService(db), workflowSvc)
	return ctx, db, svc
}

func createWorkflowScanLibrary(t *testing.T, ctx context.Context, svc *Service, name string, libraryType string) database.Library {
	t.Helper()
	rootPath := filepath.Join(svc.cfg.Local.RootPath, name)
	if err := os.MkdirAll(rootPath, 0o755); err != nil {
		t.Fatalf("create library root: %v", err)
	}
	source, err := svc.CreateMediaSource(ctx, CreateMediaSourceInput{Provider: "local", Name: fmt.Sprintf("%s Source", name), RootPath: svc.cfg.Local.RootPath})
	if err != nil {
		t.Fatalf("create media source: %v", err)
	}
	libraryRecord, _, err := svc.CreateLibrary(ctx, CreateLibraryInput{Name: name, Type: libraryType, MediaSourceID: source.ID, RootPath: rootPath})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	return libraryRecord
}
