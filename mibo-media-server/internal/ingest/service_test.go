package ingest

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
)

func TestReconcileSkipsInventoryProbeWhenBatchPolicyDisabled(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	workflowSvc := workflow.NewService(db)
	svc := NewService(db, workflowSvc)

	source := database.MediaSource{Name: "source", Provider: "local", RootPath: "/media"}
	if err := db.WithContext(ctx).Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	library := database.Library{Name: "library", Type: "auto", MediaSourceID: source.ID, RootPath: "/media", Status: "active"}
	if err := db.WithContext(ctx).Create(&library).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.LibraryScanPolicy{LibraryID: library.ID, ScannerEnabled: true, RealtimeMonitorEnabled: true, ScheduledRefreshEnabled: true, RefreshIntervalHours: 24, IgnoreHiddenFiles: true, IgnoreFileExtensionsJSON: "[]", InventoryProbeBatchEnabled: false, ConfigurableExclusionRules: true}).Error; err != nil {
		t.Fatalf("create scan policy: %v", err)
	}
	if err := db.WithContext(ctx).Model(&database.LibraryScanPolicy{}).Where("library_id = ?", library.ID).Update("inventory_probe_batch_enabled", false).Error; err != nil {
		t.Fatalf("disable inventory probe batch: %v", err)
	}
	file := database.InventoryFile{LibraryID: library.ID, MediaSourceID: source.ID, StorageProvider: source.Provider, StoragePath: "/media/movie.mkv", ContentClass: "video", Status: "available", ScanState: "discovered"}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	resource := database.Resource{StableResourceKey: "resource:movie", ResourceType: "playable", ResourceShape: "single_file", DisplayName: "movie", Status: "available", ProbeStatus: "pending"}
	if err := db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceFile{ResourceID: resource.ID, InventoryFileID: file.ID, Role: "primary"}).Error; err != nil {
		t.Fatalf("create resource file: %v", err)
	}
	now := time.Now().UTC()
	if err := db.WithContext(ctx).Create(&database.ResourceLibraryLink{ResourceID: resource.ID, LibraryID: library.ID, Status: "available", FirstSeenAt: now, LastSeenAt: now}).Error; err != nil {
		t.Fatalf("create resource library link: %v", err)
	}
	item := database.MetadataItem{ItemType: "movie", Title: "Movie", GovernanceStatus: "matched"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: "primary", ReviewState: "accepted"}).Error; err != nil {
		t.Fatalf("create resource metadata link: %v", err)
	}
	if _, err := svc.MarkInventoryFileDirty(ctx, file.ID, "test"); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}

	result, err := svc.ReconcileOnce(ctx, 10)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.Processed != 1 {
		t.Fatalf("expected one processed unit, got %+v", result)
	}
	var count int64
	if err := db.WithContext(ctx).Model(&database.WorkflowTask{}).Where("task_type = ?", workflow.TaskTypeProbeInventory).Count(&count).Error; err != nil {
		t.Fatalf("count probe tasks: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no probe inventory task, got %d", count)
	}
}

func TestReconcileDefersDispatchDuringActiveScanWorkflow(t *testing.T) {
	ctx := context.Background()
	db, svc, library, file := setupDirtyMaterializationFixture(t, ctx)
	now := time.Now().UTC()
	activeRun := database.WorkflowRun{RunKey: fmt.Sprintf("library:%d:manual_scan", library.ID), LibraryID: library.ID, Reason: "manual_scan", Status: workflow.RunStatusQueued, ScopeKey: fmt.Sprintf("library:%d", library.ID), CreatedAt: now, UpdatedAt: now}
	if err := db.WithContext(ctx).Create(&activeRun).Error; err != nil {
		t.Fatalf("create active workflow run: %v", err)
	}
	if _, err := svc.MarkInventoryFileDirty(ctx, file.ID, "scanner_discovery"); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}

	result, err := svc.ReconcileOnce(ctx, 10)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.Processed != 1 {
		t.Fatalf("expected one processed unit, got %+v", result)
	}
	assertWorkflowTaskCount(t, db, workflow.TaskTypeResolveRecognition, 0)
}

func TestReconcileScannerDiscoveryDoesNotDispatchSingleFileRecognition(t *testing.T) {
	ctx := context.Background()
	db, svc, _, file := setupDirtyMaterializationFixture(t, ctx)
	if _, err := svc.MarkInventoryFileDirty(ctx, file.ID, "scanner_discovery"); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}

	result, err := svc.ReconcileOnce(ctx, 10)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.Processed != 1 {
		t.Fatalf("expected one processed unit, got %+v", result)
	}
	assertWorkflowTaskCount(t, db, workflow.TaskTypeResolveRecognition, 0)
}

func TestReconcileScannerDiscoveryStillSkipsAfterScanWorkflowCompleted(t *testing.T) {
	ctx := context.Background()
	db, svc, library, file := setupDirtyMaterializationFixture(t, ctx)
	now := time.Now().UTC()
	completedRun := database.WorkflowRun{RunKey: fmt.Sprintf("library:%d:manual_scan", library.ID), LibraryID: library.ID, Reason: "manual_scan", Status: workflow.RunStatusCompleted, ScopeKey: fmt.Sprintf("library:%d", library.ID), CreatedAt: now, UpdatedAt: now, FinishedAt: &now}
	if err := db.WithContext(ctx).Create(&completedRun).Error; err != nil {
		t.Fatalf("create completed workflow run: %v", err)
	}
	if _, err := svc.MarkInventoryFileDirty(ctx, file.ID, "scanner_discovery"); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}

	result, err := svc.ReconcileOnce(ctx, 10)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.Processed != 1 {
		t.Fatalf("expected one processed unit, got %+v", result)
	}
	assertWorkflowTaskCount(t, db, workflow.TaskTypeResolveRecognition, 0)
}

func TestReconcileRecognitionMaterializationCompletedDoesNotDispatchSingleFileRecognition(t *testing.T) {
	ctx := context.Background()
	db, svc, _, file := setupDirtyMaterializationFixture(t, ctx)
	if _, err := svc.MarkInventoryFileDirty(ctx, file.ID, "recognition_materialization_completed"); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}

	result, err := svc.ReconcileOnce(ctx, 10)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.Processed != 1 {
		t.Fatalf("expected one processed unit, got %+v", result)
	}
	assertWorkflowTaskCount(t, db, workflow.TaskTypeResolveRecognition, 0)
}

func TestReconcileTargetedRefreshQueuedDoesNotDispatchSingleFileRecognition(t *testing.T) {
	ctx := context.Background()
	db, svc, _, file := setupDirtyMaterializationFixture(t, ctx)
	if _, err := svc.MarkInventoryFileDirty(ctx, file.ID, "targeted_refresh_queued"); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}

	result, err := svc.ReconcileOnce(ctx, 10)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.Processed != 1 {
		t.Fatalf("expected one processed unit, got %+v", result)
	}
	assertWorkflowTaskCount(t, db, workflow.TaskTypeResolveRecognition, 0)
}

func TestReconcileAllowsAdminRetryDuringActiveScanWorkflow(t *testing.T) {
	ctx := context.Background()
	db, svc, library, file := setupDirtyMaterializationFixture(t, ctx)
	now := time.Now().UTC()
	activeRun := database.WorkflowRun{RunKey: fmt.Sprintf("library:%d:manual_scan", library.ID), LibraryID: library.ID, Reason: "manual_scan", Status: workflow.RunStatusRunning, ScopeKey: fmt.Sprintf("library:%d", library.ID), CreatedAt: now, UpdatedAt: now, StartedAt: &now}
	if err := db.WithContext(ctx).Create(&activeRun).Error; err != nil {
		t.Fatalf("create active workflow run: %v", err)
	}
	if _, err := svc.MarkInventoryFileDirty(ctx, file.ID, "admin_retry_materialized"); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}

	result, err := svc.ReconcileOnce(ctx, 10)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.Processed != 1 {
		t.Fatalf("expected one processed unit, got %+v", result)
	}
	assertWorkflowTaskCount(t, db, workflow.TaskTypeResolveRecognition, 1)
}

func TestProjectionLibraryDirtyDefersDuringActiveScanWorkflow(t *testing.T) {
	ctx := context.Background()
	db, svc, library, _ := setupDirtyMaterializationFixture(t, ctx)
	now := time.Now().UTC()
	activeRun := database.WorkflowRun{RunKey: fmt.Sprintf("library:%d:manual_scan", library.ID), LibraryID: library.ID, Reason: "manual_scan", Status: workflow.RunStatusRunning, ScopeKey: fmt.Sprintf("library:%d", library.ID), CreatedAt: now, UpdatedAt: now, StartedAt: &now}
	if err := db.WithContext(ctx).Create(&activeRun).Error; err != nil {
		t.Fatalf("create active workflow run: %v", err)
	}
	if _, err := svc.MarkProjectionLibraryDirty(ctx, library.ID, library.RootPath, "materialization_completed"); err != nil {
		t.Fatalf("mark projection dirty: %v", err)
	}

	result, err := svc.ReconcileOnce(ctx, 10)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.Processed != 1 {
		t.Fatalf("expected one processed unit, got %+v", result)
	}
	assertWorkflowTaskCount(t, db, workflow.TaskTypeRefreshProjection, 0)
}

func TestProjectionLibraryDirtyAllowsAdminRetryDuringActiveScanWorkflow(t *testing.T) {
	ctx := context.Background()
	db, svc, library, _ := setupDirtyMaterializationFixture(t, ctx)
	now := time.Now().UTC()
	activeRun := database.WorkflowRun{RunKey: fmt.Sprintf("library:%d:manual_scan", library.ID), LibraryID: library.ID, Reason: "manual_scan", Status: workflow.RunStatusRunning, ScopeKey: fmt.Sprintf("library:%d", library.ID), CreatedAt: now, UpdatedAt: now, StartedAt: &now}
	if err := db.WithContext(ctx).Create(&activeRun).Error; err != nil {
		t.Fatalf("create active workflow run: %v", err)
	}
	if _, err := svc.MarkProjectionLibraryDirty(ctx, library.ID, library.RootPath, "admin_retry_projection"); err != nil {
		t.Fatalf("mark projection dirty: %v", err)
	}

	result, err := svc.ReconcileOnce(ctx, 10)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.Processed != 1 {
		t.Fatalf("expected one processed unit, got %+v", result)
	}
	assertWorkflowTaskCount(t, db, workflow.TaskTypeRefreshProjection, 1)
}

func setupDirtyMaterializationFixture(t *testing.T, ctx context.Context) (*gorm.DB, *Service, database.Library, database.InventoryFile) {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	workflowSvc := workflow.NewService(db)
	svc := NewService(db, workflowSvc)
	source := database.MediaSource{Name: "source", Provider: "local", RootPath: "/media"}
	if err := db.WithContext(ctx).Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	library := database.Library{Name: "library", Type: "auto", MediaSourceID: source.ID, RootPath: "/media", Status: "active"}
	if err := db.WithContext(ctx).Create(&library).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.LibraryScanPolicy{LibraryID: library.ID, ScannerEnabled: true, RealtimeMonitorEnabled: true, ScheduledRefreshEnabled: true, RefreshIntervalHours: 24, IgnoreHiddenFiles: true, IgnoreFileExtensionsJSON: "[]", InventoryProbeBatchEnabled: true, ConfigurableExclusionRules: true}).Error; err != nil {
		t.Fatalf("create scan policy: %v", err)
	}
	file := database.InventoryFile{LibraryID: library.ID, MediaSourceID: source.ID, StorageProvider: source.Provider, StoragePath: "/media/movie.mkv", ContentClass: "video", Status: "available", ScanState: "discovered"}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	return db, svc, library, file
}

func assertWorkflowTaskCount(t *testing.T, db *gorm.DB, taskType string, expected int64) {
	t.Helper()
	var count int64
	if err := db.Model(&database.WorkflowTask{}).Where("task_type = ?", taskType).Count(&count).Error; err != nil {
		t.Fatalf("count workflow tasks: %v", err)
	}
	if count != expected {
		t.Fatalf("expected %d %s task(s), got %d", expected, taskType, count)
	}
}
