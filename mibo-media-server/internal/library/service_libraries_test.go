package library

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/storage"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
)

func TestCreateLibraryRecordDoesNotWaitForSourceProbe(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("unwrap database: %v", err)
	}
	defer sqlDB.Close()

	source := database.MediaSource{Name: "Local", Provider: "local", StorageRef: "local", RootPath: "/media"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}

	provider := newBlockingProbeProvider()
	svc := NewService(config.Config{}, db, providers.NewRegistry(config.Config{}), nil)
	result := make(chan struct {
		library database.Library
		err     error
	}, 1)
	go func() {
		library, err := svc.createLibraryRecord(context.Background(), source, provider, "/media", "Movies")
		result <- struct {
			library database.Library
			err     error
		}{library: library, err: err}
	}()

	select {
	case created := <-result:
		if created.err != nil {
			t.Fatalf("create library record: %v", created.err)
		}
		if created.library.ProbeStatus != SourceProbeStatusPending {
			t.Fatalf("probe status = %q, want pending", created.library.ProbeStatus)
		}
		var stored database.Library
		if err := db.First(&stored, created.library.ID).Error; err != nil {
			t.Fatalf("load library: %v", err)
		}
		if stored.ProbeStatus != SourceProbeStatusPending {
			t.Fatalf("stored probe status = %q, want pending", stored.ProbeStatus)
		}
	case <-time.After(200 * time.Millisecond):
		provider.release()
		t.Fatal("createLibraryRecord waited for source probe")
	}

	<-provider.started
	provider.release()
}

func TestQueueTargetedRefreshSkipsWhileActiveScanWorkflowExists(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	source := database.MediaSource{Name: "Local", Provider: "local", StorageRef: "local", RootPath: "/media"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	library := database.Library{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: "/media", Status: "syncing"}
	if err := db.Create(&library).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	if err := db.Create(&database.LibraryPath{LibraryID: library.ID, MediaSourceID: source.ID, RootPath: "/media", Enabled: true}).Error; err != nil {
		t.Fatalf("create library path: %v", err)
	}
	activeRun := database.WorkflowRun{RunKey: fmt.Sprintf("library:%d:%s", library.ID, WorkflowReasonCreateLibrary), LibraryID: library.ID, Reason: WorkflowReasonCreateLibrary, Status: workflow.RunStatusRunning, ScopeKey: fmt.Sprintf("library:%d", library.ID)}
	if err := db.Create(&activeRun).Error; err != nil {
		t.Fatalf("create active workflow run: %v", err)
	}
	svc := NewService(config.Config{}, db, providers.NewRegistry(config.Config{}), nil)

	job, err := svc.QueueTargetedRefresh(ctx, library.ID, "/media", WorkflowReasonTargetedRefresh)
	if err != nil {
		t.Fatalf("queue targeted refresh: %v", err)
	}
	if job.ID != 0 {
		t.Fatalf("expected no targeted refresh workflow job, got %#v", job)
	}
	var count int64
	if err := db.Model(&database.WorkflowRun{}).Where("library_id = ? AND reason = ?", library.ID, WorkflowReasonTargetedRefresh).Count(&count).Error; err != nil {
		t.Fatalf("count targeted refresh workflows: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no targeted refresh workflows, got %d", count)
	}
}

func TestDeleteLibraryRemovesCatalogInventoryAndJobRecords(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("unwrap database: %v", err)
	}
	defer sqlDB.Close()

	source := database.MediaSource{Name: "Local", Provider: "local", StorageRef: "local", RootPath: "/media"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	library := database.Library{Name: "Delete Me", Type: "movies", MediaSourceID: source.ID, RootPath: "/media/delete", Status: "active", ScannerEnabled: true}
	otherLibrary := database.Library{Name: "Keep Me", Type: "movies", MediaSourceID: source.ID, RootPath: "/media/keep", Status: "active", ScannerEnabled: true}
	if err := db.Create(&library).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	if err := db.Create(&otherLibrary).Error; err != nil {
		t.Fatalf("create other library: %v", err)
	}

	inventoryFile := database.InventoryFile{LibraryID: library.ID, StorageProvider: "local", StoragePath: "/media/delete/movie.mkv", Status: "available"}
	otherInventoryFile := database.InventoryFile{LibraryID: otherLibrary.ID, StorageProvider: "local", StoragePath: "/media/keep/movie.mkv", Status: "available"}
	if err := db.Create(&inventoryFile).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	if err := db.Create(&otherInventoryFile).Error; err != nil {
		t.Fatalf("create other inventory file: %v", err)
	}
	metadataItem := database.MetadataItem{ItemType: "movie", ContentForm: "standard", Title: "Deleted Movie", SortKey: "deleted movie", GovernanceStatus: "accepted"}
	if err := db.Create(&metadataItem).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	deletedResource := database.Resource{StableResourceKey: "resource:delete", DisplayName: "Deleted Movie", Status: "available"}
	sharedResource := database.Resource{StableResourceKey: "resource:shared", DisplayName: "Shared Movie", Status: "available"}
	if err := db.Create(&deletedResource).Error; err != nil {
		t.Fatalf("create deleted resource: %v", err)
	}
	if err := db.Create(&sharedResource).Error; err != nil {
		t.Fatalf("create shared resource: %v", err)
	}
	seed := []any{
		&database.MediaStream{FileID: inventoryFile.ID, StreamIndex: 0, StreamType: "video"},
		&database.MediaStream{FileID: otherInventoryFile.ID, StreamIndex: 0, StreamType: "video"},
		&database.MediaStream{FileID: 999999, StreamIndex: 0, StreamType: "video"},
		&database.ResourceFile{ResourceID: deletedResource.ID, InventoryFileID: inventoryFile.ID, Role: "primary"},
		&database.ResourceLibraryLink{ResourceID: deletedResource.ID, LibraryID: library.ID, Status: "available", FirstSeenAt: time.Now(), LastSeenAt: time.Now()},
		&database.ResourceMetadataLink{ResourceID: deletedResource.ID, MetadataItemID: metadataItem.ID, Role: "primary"},
		&database.ResourceFile{ResourceID: sharedResource.ID, InventoryFileID: inventoryFile.ID, Role: "primary"},
		&database.ResourceFile{ResourceID: sharedResource.ID, InventoryFileID: otherInventoryFile.ID, Role: "primary"},
		&database.ResourceLibraryLink{ResourceID: sharedResource.ID, LibraryID: library.ID, Status: "available", FirstSeenAt: time.Now(), LastSeenAt: time.Now()},
		&database.ResourceLibraryLink{ResourceID: sharedResource.ID, LibraryID: otherLibrary.ID, Status: "available", FirstSeenAt: time.Now(), LastSeenAt: time.Now()},
		&database.ResourceMetadataLink{ResourceID: sharedResource.ID, MetadataItemID: metadataItem.ID, Role: "primary"},
		&database.LibraryMetadataProjection{LibraryID: library.ID, MetadataItemID: metadataItem.ID, ItemType: metadataItem.ItemType, Title: metadataItem.Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, LastProjectedAt: time.Now()},
		&database.LibrarySearchDocument{LibraryID: library.ID, MetadataItemID: metadataItem.ID, ItemType: metadataItem.ItemType, Title: metadataItem.Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, UpdatedAt: time.Now()},
		&database.UserMetadataData{UserID: 1, MetadataItemID: metadataItem.ID, PreferredResourceID: &deletedResource.ID},
		&database.UserResourceData{UserID: 1, ResourceID: deletedResource.ID, MetadataItemID: metadataItem.ID},
		&database.UserResourceData{UserID: 1, ResourceID: sharedResource.ID, MetadataItemID: metadataItem.ID},
		&database.MetadataOperation{Operation: "match", OriginMetadataItemID: 1, TargetMetadataItemID: 1, LibraryID: library.ID, Status: "applied", StartedAt: time.Now()},
		&database.IngestDirtyUnit{DirtyKey: "inventory_file:delete", ScopeKind: "inventory_file", LibraryID: library.ID, InventoryFileID: &inventoryFile.ID, Reason: "test", Status: "dirty", AvailableAt: time.Now()},
		&database.IngestCondition{UnitKey: "inventory_file:delete", LibraryID: library.ID, InventoryFileID: &inventoryFile.ID, ConditionType: "probed", Status: "failed", Reason: "test", Severity: "error"},
		&database.IngestEvent{UnitKey: "inventory_file:delete", LibraryID: library.ID, InventoryFileID: &inventoryFile.ID, EventType: "condition_changed", Status: "failed", Reason: "test"},
		&database.IngestDirtyUnit{DirtyKey: "inventory_file:keep", ScopeKind: "inventory_file", LibraryID: otherLibrary.ID, InventoryFileID: &otherInventoryFile.ID, Reason: "test", Status: "dirty", AvailableAt: time.Now()},
		&database.IngestCondition{UnitKey: "inventory_file:keep", LibraryID: otherLibrary.ID, InventoryFileID: &otherInventoryFile.ID, ConditionType: "probed", Status: "failed", Reason: "test", Severity: "error"},
		&database.IngestEvent{UnitKey: "inventory_file:keep", LibraryID: otherLibrary.ID, InventoryFileID: &otherInventoryFile.ID, EventType: "condition_changed", Status: "failed", Reason: "test"},
		&database.LibraryMetadataStrategy{LibraryID: library.ID},
		&database.ScanExclusion{LibraryID: library.ID, StorageProvider: "local", StoragePath: "/media/delete/ad.mkv", Reason: "advertisement", Enabled: true},
		&database.User{Username: "user", PasswordHash: "hash", Role: "admin"},
	}
	for _, record := range seed {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("seed %T: %v", record, err)
		}
	}
	schedule := database.Schedule{Name: "scan", Kind: "scan", ScopeKind: "library", LibraryID: &library.ID, FrequencyKind: "daily", TimeOfDay: "03:00", Enabled: true}
	if err := db.Create(&schedule).Error; err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	if err := db.Create(&database.ScheduleRun{ScheduleID: schedule.ID, Status: "completed"}).Error; err != nil {
		t.Fatalf("create schedule run: %v", err)
	}
	job := database.Job{Kind: JobKindSyncLibrary, Status: "queued", PayloadJSON: fmt.Sprintf(`{"library_id":%d,"root_path":"/media/delete"}`, library.ID), AvailableAt: time.Now()}
	probeJob := database.Job{Kind: JobKindProbeInventoryFile, Status: "queued", PayloadJSON: fmt.Sprintf(`{"inventory_file_id":%d}`, inventoryFile.ID), AvailableAt: time.Now()}
	runningJob := database.Job{Kind: JobKindSyncLibrary, Status: "running", PayloadJSON: fmt.Sprintf(`{"library_id":%d,"root_path":"/media/delete"}`, library.ID), AvailableAt: time.Now(), StartedAt: timePtr(time.Now())}
	otherJob := database.Job{Kind: JobKindSyncLibrary, Status: "queued", PayloadJSON: fmt.Sprintf(`{"library_id":%d,"root_path":"/media/keep"}`, otherLibrary.ID), AvailableAt: time.Now()}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}
	if err := db.Create(&probeJob).Error; err != nil {
		t.Fatalf("create probe job: %v", err)
	}
	if err := db.Create(&runningJob).Error; err != nil {
		t.Fatalf("create running job: %v", err)
	}
	if err := db.Create(&otherJob).Error; err != nil {
		t.Fatalf("create other job: %v", err)
	}
	if err := db.Create(&database.JobActiveIntent{IntentKey: "sync", Kind: job.Kind, JobID: job.ID}).Error; err != nil {
		t.Fatalf("create active intent: %v", err)
	}
	workflowRun := database.WorkflowRun{RunKey: fmt.Sprintf("library:%d:manual_scan", library.ID), LibraryID: library.ID, Reason: "manual_scan", Status: workflow.RunStatusQueued, ScopeKey: fmt.Sprintf("library:%d", library.ID)}
	otherWorkflowRun := database.WorkflowRun{RunKey: fmt.Sprintf("library:%d:manual_scan", otherLibrary.ID), LibraryID: otherLibrary.ID, Reason: "manual_scan", Status: workflow.RunStatusQueued, ScopeKey: fmt.Sprintf("library:%d", otherLibrary.ID)}
	if err := db.Create(&workflowRun).Error; err != nil {
		t.Fatalf("create workflow run: %v", err)
	}
	if err := db.Create(&otherWorkflowRun).Error; err != nil {
		t.Fatalf("create other workflow run: %v", err)
	}
	workflowTask := database.WorkflowTask{RunID: workflowRun.ID, LibraryID: library.ID, TaskKey: fmt.Sprintf("run:%d:scan", workflowRun.ID), TaskType: workflow.TaskTypeScanLibraryPath, Stage: workflow.StageScan, Status: workflow.TaskStatusRunning, ScopeKey: workflowRun.ScopeKey, AvailableAt: time.Now()}
	dependentWorkflowTask := database.WorkflowTask{RunID: workflowRun.ID, LibraryID: library.ID, TaskKey: fmt.Sprintf("run:%d:projection", workflowRun.ID), TaskType: workflow.TaskTypeRefreshProjection, Stage: workflow.StageProjection, Status: workflow.TaskStatusBlocked, ScopeKey: workflowRun.ScopeKey, BlockedBy: 1, AvailableAt: time.Now()}
	otherWorkflowTask := database.WorkflowTask{RunID: otherWorkflowRun.ID, LibraryID: otherLibrary.ID, TaskKey: fmt.Sprintf("run:%d:scan", otherWorkflowRun.ID), TaskType: workflow.TaskTypeScanLibraryPath, Stage: workflow.StageScan, Status: workflow.TaskStatusQueued, ScopeKey: otherWorkflowRun.ScopeKey, AvailableAt: time.Now()}
	if err := db.Create(&workflowTask).Error; err != nil {
		t.Fatalf("create workflow task: %v", err)
	}
	if err := db.Create(&dependentWorkflowTask).Error; err != nil {
		t.Fatalf("create dependent workflow task: %v", err)
	}
	if err := db.Create(&otherWorkflowTask).Error; err != nil {
		t.Fatalf("create other workflow task: %v", err)
	}
	if err := db.Create(&database.WorkflowTaskDependency{TaskID: dependentWorkflowTask.ID, DependsOnTaskID: workflowTask.ID}).Error; err != nil {
		t.Fatalf("create workflow dependency: %v", err)
	}
	if err := db.Create(&database.WorkflowTaskLease{TaskID: workflowTask.ID, Owner: "test", LeaseUntil: time.Now().Add(time.Minute)}).Error; err != nil {
		t.Fatalf("create workflow lease: %v", err)
	}
	if err := db.Create(&database.WorkflowResourceUsage{ResourceKey: "library_scan", TaskID: workflowTask.ID, RunID: workflowRun.ID, LibraryID: library.ID, Units: 1, LeaseUntil: time.Now().Add(time.Minute)}).Error; err != nil {
		t.Fatalf("create workflow resource usage: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	if err := svc.DeleteLibrary(context.Background(), library.ID); err != nil {
		t.Fatalf("delete library: %v", err)
	}

	assertRawTableCount(t, db, "libraries", "id = ?", 0, library.ID)
	assertRawTableCount(t, db, "inventory_files", "library_id = ?", 0, library.ID)
	assertRawTableCount(t, db, "media_streams", "file_id = ?", 0, inventoryFile.ID)
	assertRawTableCount(t, db, "media_streams", "file_id = 999999", 0)
	assertRawTableCount(t, db, "resource_files", "inventory_file_id = ?", 0, inventoryFile.ID)
	assertRawTableCount(t, db, "resource_library_links", "library_id = ?", 0, library.ID)
	assertRawTableCount(t, db, "resource_metadata_links", "resource_id = ?", 0, deletedResource.ID)
	assertRawTableCount(t, db, "resources", "id = ?", 0, deletedResource.ID)
	assertRawTableCount(t, db, "library_metadata_projections", "library_id = ?", 0, library.ID)
	assertRawTableCount(t, db, "library_search_documents", "library_id = ?", 0, library.ID)
	assertRawTableCount(t, db, "user_resource_data", "resource_id = ?", 0, deletedResource.ID)
	assertRawTableCount(t, db, "user_metadata_data", "preferred_resource_id = ?", 0, deletedResource.ID)
	assertRawTableCount(t, db, "metadata_operations", "library_id = ?", 0, library.ID)
	assertRawTableCount(t, db, "ingest_dirty_units", "library_id = ?", 0, library.ID)
	assertRawTableCount(t, db, "ingest_conditions", "library_id = ?", 0, library.ID)
	assertRawTableCount(t, db, "ingest_events", "library_id = ?", 0, library.ID)
	assertRawTableCount(t, db, "library_metadata_strategies", "library_id = ?", 0, library.ID)
	assertRawTableCount(t, db, "scan_exclusions", "library_id = ?", 0, library.ID)
	assertRawTableCount(t, db, "schedules", "library_id = ?", 0, library.ID)
	assertRawTableCount(t, db, "schedule_runs", "schedule_id = ?", 0, schedule.ID)
	assertRawTableCount(t, db, "jobs", "id IN (?, ?)", 0, job.ID, probeJob.ID)
	assertRawTableCount(t, db, "jobs", "id = ? AND status = ?", 1, runningJob.ID, "cancel_requested")
	assertRawTableCount(t, db, "job_active_intents", "job_id = ?", 0, job.ID)
	assertRawTableCount(t, db, "workflow_runs", "id = ?", 0, workflowRun.ID)
	assertRawTableCount(t, db, "workflow_tasks", "id IN (?, ?)", 0, workflowTask.ID, dependentWorkflowTask.ID)
	assertRawTableCount(t, db, "workflow_task_dependencies", "task_id = ? OR depends_on_task_id = ?", 0, dependentWorkflowTask.ID, workflowTask.ID)
	assertRawTableCount(t, db, "workflow_task_leases", "task_id = ?", 0, workflowTask.ID)
	assertRawTableCount(t, db, "workflow_resource_usages", "task_id = ? OR library_id = ?", 0, workflowTask.ID, library.ID)
	assertRawTableCount(t, db, "libraries", "id = ?", 1, otherLibrary.ID)
	assertRawTableCount(t, db, "inventory_files", "id = ?", 1, otherInventoryFile.ID)
	assertRawTableCount(t, db, "resource_files", "inventory_file_id = ?", 1, otherInventoryFile.ID)
	assertRawTableCount(t, db, "resource_library_links", "library_id = ?", 1, otherLibrary.ID)
	assertRawTableCount(t, db, "resource_metadata_links", "resource_id = ?", 1, sharedResource.ID)
	assertRawTableCount(t, db, "resources", "id = ?", 1, sharedResource.ID)
	assertRawTableCount(t, db, "user_resource_data", "resource_id = ?", 1, sharedResource.ID)
	assertRawTableCount(t, db, "ingest_dirty_units", "library_id = ?", 1, otherLibrary.ID)
	assertRawTableCount(t, db, "ingest_conditions", "library_id = ?", 1, otherLibrary.ID)
	assertRawTableCount(t, db, "ingest_events", "library_id = ?", 1, otherLibrary.ID)
	assertRawTableCount(t, db, "jobs", "id = ?", 1, otherJob.ID)
	assertRawTableCount(t, db, "workflow_runs", "id = ?", 1, otherWorkflowRun.ID)
	assertRawTableCount(t, db, "workflow_tasks", "id = ?", 1, otherWorkflowTask.ID)
}

func assertRawTableCount(t *testing.T, db *gorm.DB, table, where string, expected int64, args ...any) {
	t.Helper()
	var count int64
	if err := db.Table(table).Where(where, args...).Count(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != expected {
		t.Fatalf("%s count = %d, want %d", table, count, expected)
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}

type blockingProbeProvider struct {
	started     chan struct{}
	releaseOnce chan struct{}
}

func newBlockingProbeProvider() *blockingProbeProvider {
	return &blockingProbeProvider{started: make(chan struct{}), releaseOnce: make(chan struct{})}
}

func (p *blockingProbeProvider) Name() string { return "fake" }

func (p *blockingProbeProvider) List(ctx context.Context, req storage.ListRequest) ([]storage.Object, error) {
	select {
	case <-p.started:
	default:
		close(p.started)
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.releaseOnce:
	}
	return []storage.Object{{Path: req.Path + "/movie.mkv"}}, nil
}

func (p *blockingProbeProvider) Get(context.Context, storage.GetRequest) (storage.Object, error) {
	return storage.Object{}, storage.ErrNotImplemented
}

func (p *blockingProbeProvider) Link(context.Context, storage.LinkRequest) (storage.LinkResult, error) {
	return storage.LinkResult{}, storage.ErrNotImplemented
}

func (p *blockingProbeProvider) ResolveStorage(context.Context, storage.ResolveStorageRequest) (storage.ResolvedStorage, error) {
	return storage.ResolvedStorage{}, nil
}

func (p *blockingProbeProvider) Capabilities(context.Context) (storage.Capabilities, error) {
	return storage.Capabilities{CanList: true}, nil
}

func (p *blockingProbeProvider) release() {
	select {
	case <-p.releaseOnce:
	default:
		close(p.releaseOnce)
	}
}
