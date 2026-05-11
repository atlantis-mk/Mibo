package library

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
)

func mustWriteFixtureFile(t *testing.T, filePath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("create fixture dir: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("fixture"), 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
}

func TestQueueLibraryWorkflowCreatesPathScopedScanTasks(t *testing.T) {
	ctx, db, svc := newWorkflowScanHarness(t)
	libraryRecord := createWorkflowScanLibrary(t, ctx, svc, "Movies", LibraryTypeAuto)

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

func TestRunRecognitionResolveBatchSkipsMissingInventoryFiles(t *testing.T) {
	ctx, _, svc := newWorkflowScanHarness(t)
	libraryRecord := createWorkflowScanLibrary(t, ctx, svc, "Movies", LibraryTypeAuto)

	err := svc.RunRecognitionResolveBatch(ctx, RecognitionResolveBatchPayload{
		LibraryID: libraryRecord.ID,
		RootPath:  libraryRecord.RootPath,
		FileIDs:   []uint{999999},
	})
	if err != nil {
		t.Fatalf("expected missing inventory file to be skipped, got %v", err)
	}
}

func TestRunWorkflowScanLibraryPathQueuesRecognitionResolveTasksBeforeProjection(t *testing.T) {
	ctx, db, svc := newWorkflowScanHarness(t)
	libraryRecord := createWorkflowScanLibrary(t, ctx, svc, "Movies", LibraryTypeAuto)
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

	var resolveTasks int64
	if err := db.WithContext(ctx).Model(&database.WorkflowTask{}).Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeResolveRecognition).Count(&resolveTasks).Error; err != nil {
		t.Fatalf("count recognition resolve tasks: %v", err)
	}
	if resolveTasks != 1 {
		t.Fatalf("expected scan to queue one follow-up resolve task, got %d", resolveTasks)
	}
	var manifestCount int64
	if err := db.WithContext(ctx).Model(&database.RecognitionManifest{}).Where("library_id = ?", libraryRecord.ID).Count(&manifestCount).Error; err != nil {
		t.Fatalf("count recognition manifests: %v", err)
	}
	if manifestCount != 0 {
		t.Fatalf("expected scan to defer recognition materialization, got %d manifests", manifestCount)
	}
	var projectionTasks int64
	if err := db.WithContext(ctx).Model(&database.WorkflowTask{}).Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeRefreshProjection).Count(&projectionTasks).Error; err != nil {
		t.Fatalf("count projection tasks: %v", err)
	}
	if projectionTasks != 0 {
		t.Fatalf("expected projection refresh to wait for recognition resolve, got %d tasks", projectionTasks)
	}

	var resolveTask database.WorkflowTask
	if err := db.WithContext(ctx).Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeResolveRecognition).First(&resolveTask).Error; err != nil {
		t.Fatalf("load resolve task: %v", err)
	}
	if err := svc.RunWorkflowRecognitionResolve(ctx, resolveTask); err != nil {
		t.Fatalf("run workflow recognition resolve task: %v", err)
	}
	if err := db.WithContext(ctx).Model(&database.RecognitionManifest{}).Where("library_id = ?", libraryRecord.ID).Count(&manifestCount).Error; err != nil {
		t.Fatalf("count recognition manifests after resolve: %v", err)
	}
	if manifestCount == 0 {
		t.Fatalf("expected resolve task to persist recognition manifest")
	}
	var item database.MetadataItem
	if err := db.WithContext(ctx).Where("title = ?", "Movie A").First(&item).Error; err != nil {
		t.Fatalf("expected resolve task to materialize metadata through recognition resolver: %v", err)
	}
	if err := db.WithContext(ctx).Model(&database.WorkflowTask{}).Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeRefreshProjection).Count(&projectionTasks).Error; err != nil {
		t.Fatalf("count projection tasks after resolve: %v", err)
	}
	if projectionTasks != 1 {
		t.Fatalf("expected one projection task after resolve, got %d", projectionTasks)
	}
}

func TestRunWorkflowScanLibraryPathAcceptsScopedSubdirectory(t *testing.T) {
	ctx, db, svc := newWorkflowScanHarness(t)
	libraryRecord := createWorkflowScanLibrary(t, ctx, svc, "Movies", LibraryTypeAuto)
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

	var resolveTask database.WorkflowTask
	if err := db.WithContext(ctx).Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeResolveRecognition).First(&resolveTask).Error; err != nil {
		t.Fatalf("load resolve task: %v", err)
	}
	if err := svc.RunWorkflowRecognitionResolve(ctx, resolveTask); err != nil {
		t.Fatalf("run workflow recognition resolve task: %v", err)
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
	libraryRecord := createWorkflowScanLibrary(t, ctx, svc, "Movies", LibraryTypeAuto)

	job, err := svc.QueueLibraryScanWithReason(ctx, libraryRecord.ID, WorkflowReasonManualScan)
	if err != nil {
		t.Fatalf("queue scan: %v", err)
	}
	if job.ID == 0 || job.Kind != JobKindSyncLibrary || job.Status != workflow.RunStatusQueued {
		t.Fatalf("unexpected compatibility job: %#v", job)
	}
}

func TestRunInventoryProbeBatchUsesInjectedExecutor(t *testing.T) {
	_, _, svc := newWorkflowScanHarness(t)
	var probed []uint
	svc.SetInventoryProbeExecutor(func(ctx context.Context, fileID uint) error {
		probed = append(probed, fileID)
		return nil
	})

	err := svc.RunInventoryProbeBatch(context.Background(), InventoryProbeBatchPayload{FileIDs: []uint{4, 2, 4, 0, 1}})
	if err != nil {
		t.Fatalf("run probe batch: %v", err)
	}
	if !reflect.DeepEqual(probed, []uint{1, 2, 4}) {
		t.Fatalf("unexpected probed ids: %#v", probed)
	}
}

func TestRunWorkflowInventoryFileProbeUsesBatchExecutor(t *testing.T) {
	_, _, svc := newWorkflowScanHarness(t)
	var probed []uint
	svc.SetInventoryProbeExecutor(func(ctx context.Context, fileID uint) error {
		probed = append(probed, fileID)
		return nil
	})

	err := svc.RunWorkflowInventoryFileProbe(context.Background(), database.WorkflowTask{LibraryID: 7, PayloadJSON: `{"inventory_file_id":42}`})
	if err != nil {
		t.Fatalf("run single file probe workflow: %v", err)
	}
	if !reflect.DeepEqual(probed, []uint{42}) {
		t.Fatalf("unexpected probed ids: %#v", probed)
	}
}

func TestRunMetadataMatchBatchUsesInjectedExecutor(t *testing.T) {
	ctx, db, svc := newWorkflowScanHarness(t)
	rows := []database.MetadataItem{
		{ID: 1, ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie One", SortTitle: "Movie One", SortKey: "work:movie:movie-one", GovernanceStatus: database.ReviewStatePending},
		{ID: 3, ItemType: database.MetadataItemTypeSeries, ContentForm: database.MetadataContentFormStandard, Title: "Show Three", SortTitle: "Show Three", SortKey: "work:series:show-three", GovernanceStatus: database.ReviewStatePending},
		{ID: 9, ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie Nine", SortTitle: "Movie Nine", SortKey: "work:movie:movie-nine", GovernanceStatus: database.ReviewStatePending},
	}
	for idx := range rows {
		if err := db.WithContext(ctx).Create(&rows[idx]).Error; err != nil {
			t.Fatalf("seed metadata item: %v", err)
		}
	}
	var matched []uint
	svc.SetMetadataMatchExecutor(func(ctx context.Context, metadataItemID uint, libraryID uint) error {
		matched = append(matched, metadataItemID)
		return nil
	})

	err := svc.RunMetadataMatchBatch(ctx, MetadataMatchBatchPayload{LibraryID: 2, MetadataItemIDs: []uint{9, 3, 9, 0, 1}})
	if err != nil {
		t.Fatalf("run match batch: %v", err)
	}
	if !reflect.DeepEqual(matched, []uint{1, 3, 9}) {
		t.Fatalf("unexpected matched ids: %#v", matched)
	}
}

func TestRunMetadataMatchBatchSkipsUnsupportedEpisodeItems(t *testing.T) {
	ctx, db, svc := newWorkflowScanHarness(t)
	movie := database.MetadataItem{ID: 1, ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie", SortTitle: "Movie", SortKey: "work:movie:movie", GovernanceStatus: database.ReviewStatePending}
	series := database.MetadataItem{ID: 2, ItemType: database.MetadataItemTypeSeries, ContentForm: database.MetadataContentFormStandard, Title: "Show", SortTitle: "Show", SortKey: "work:series:show", GovernanceStatus: database.ReviewStatePending}
	episode := database.MetadataItem{ID: 3, ItemType: database.MetadataItemTypeEpisode, ContentForm: database.MetadataContentFormStandard, Title: "Episode 1", SortTitle: "Episode 1", SortKey: "episode:work:season:work:series:show:s01:e01", GovernanceStatus: database.ReviewStatePending}
	for _, row := range []any{&movie, &series, &episode} {
		if err := db.WithContext(ctx).Create(row).Error; err != nil {
			t.Fatalf("seed metadata item: %v", err)
		}
	}
	var matched []uint
	svc.SetMetadataMatchExecutor(func(ctx context.Context, metadataItemID uint, libraryID uint) error {
		matched = append(matched, metadataItemID)
		return nil
	})

	err := svc.RunMetadataMatchBatch(ctx, MetadataMatchBatchPayload{LibraryID: 2, MetadataItemIDs: []uint{3, 2, 1}})
	if err != nil {
		t.Fatalf("run match batch: %v", err)
	}
	if !reflect.DeepEqual(matched, []uint{1, 2}) {
		t.Fatalf("expected only movie/series ids matched, got %#v", matched)
	}
}

func TestQueueWorkflowPostScanTasksIsIdempotent(t *testing.T) {
	ctx, db, svc := newWorkflowScanHarness(t)
	libraryRecord := createWorkflowScanLibrary(t, ctx, svc, "Movies", LibraryTypeAuto)
	run, _, err := svc.QueueLibraryWorkflow(ctx, QueueWorkflowInput{LibraryID: libraryRecord.ID, Reason: WorkflowReasonManualScan, Priority: 10})
	if err != nil {
		t.Fatalf("queue workflow: %v", err)
	}
	if err := svc.queueWorkflowRecognitionResolveTasks(ctx, run.ID, libraryRecord.ID, libraryRecord.RootPath, []uint{1, 2, 3}); err != nil {
		t.Fatalf("first queue resolve tasks: %v", err)
	}
	if err := svc.queueWorkflowRecognitionResolveTasks(ctx, run.ID, libraryRecord.ID, libraryRecord.RootPath, []uint{1, 2, 3}); err != nil {
		t.Fatalf("second queue resolve tasks: %v", err)
	}
	if err := svc.queueWorkflowMatchTasks(ctx, run.ID, libraryRecord.ID, libraryRecord.RootPath, []uint{11, 12, 13}); err != nil {
		t.Fatalf("first queue match tasks: %v", err)
	}
	if err := svc.queueWorkflowMatchTasks(ctx, run.ID, libraryRecord.ID, libraryRecord.RootPath, []uint{11, 12, 13}); err != nil {
		t.Fatalf("second queue match tasks: %v", err)
	}
	if err := svc.queueWorkflowProbeTasks(ctx, run.ID, libraryRecord.ID, libraryRecord.RootPath, []uint{21, 22, 23}); err != nil {
		t.Fatalf("first queue probe tasks: %v", err)
	}
	if err := svc.queueWorkflowProbeTasks(ctx, run.ID, libraryRecord.ID, libraryRecord.RootPath, []uint{21, 22, 23}); err != nil {
		t.Fatalf("second queue probe tasks: %v", err)
	}
	if err := svc.queueWorkflowProjectionTask(ctx, run.ID, libraryRecord.ID, libraryRecord.RootPath); err != nil {
		t.Fatalf("first queue projection task: %v", err)
	}
	if err := svc.queueWorkflowProjectionTask(ctx, run.ID, libraryRecord.ID, libraryRecord.RootPath); err != nil {
		t.Fatalf("second queue projection task: %v", err)
	}

	var tasks []database.WorkflowTask
	if err := db.WithContext(ctx).Where("run_id = ?", run.ID).Order("id asc").Find(&tasks).Error; err != nil {
		t.Fatalf("load workflow tasks: %v", err)
	}
	counts := map[string]int{}
	for _, task := range tasks {
		counts[task.TaskType]++
	}
	if counts[workflow.TaskTypeResolveRecognition] != 1 {
		t.Fatalf("expected one resolve recognition task, got %d", counts[workflow.TaskTypeResolveRecognition])
	}
	if counts[workflow.TaskTypeMatchMetadata] != 1 {
		t.Fatalf("expected one metadata match task, got %d from tasks %#v", counts[workflow.TaskTypeMatchMetadata], tasks)
	}
	if counts[workflow.TaskTypeProbeInventory] != 1 {
		t.Fatalf("expected one inventory probe task, got %d", counts[workflow.TaskTypeProbeInventory])
	}
	if counts[workflow.TaskTypeRefreshProjection] != 1 {
		t.Fatalf("expected one projection task, got %d", counts[workflow.TaskTypeRefreshProjection])
	}
}

func TestQueueWorkflowRecognitionResolveTasksGroupsFilesByDirectory(t *testing.T) {
	ctx, db, svc := newWorkflowScanHarness(t)
	libraryRecord := createWorkflowScanLibrary(t, ctx, svc, "Movies", LibraryTypeAuto)
	firstDir := filepath.Join(libraryRecord.RootPath, "CollectionA")
	secondDir := filepath.Join(libraryRecord.RootPath, "CollectionB")
	files := make([]database.InventoryFile, 0, recognitionResolveScanBatchSize+2)
	for idx := 0; idx < recognitionResolveScanBatchSize+1; idx++ {
		files = append(files, database.InventoryFile{LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: filepath.Join(firstDir, fmt.Sprintf("Movie %02d.mkv", idx)), ContentClass: "video", Status: "available", ScanState: "discovered"})
	}
	files = append(files, database.InventoryFile{LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: filepath.Join(secondDir, "Movie C.mkv"), ContentClass: "video", Status: "available", ScanState: "discovered"})
	if err := db.WithContext(ctx).Create(&files).Error; err != nil {
		t.Fatalf("create inventory files: %v", err)
	}
	run, _, err := svc.QueueLibraryWorkflow(ctx, QueueWorkflowInput{LibraryID: libraryRecord.ID, Reason: WorkflowReasonManualScan, Priority: 10})
	if err != nil {
		t.Fatalf("queue workflow: %v", err)
	}
	fileIDs := make([]uint, 0, len(files))
	for _, file := range files {
		fileIDs = append(fileIDs, file.ID)
	}
	if err := svc.queueWorkflowRecognitionResolveTasks(ctx, run.ID, libraryRecord.ID, libraryRecord.RootPath, fileIDs); err != nil {
		t.Fatalf("queue resolve tasks: %v", err)
	}

	var tasks []database.WorkflowTask
	if err := db.WithContext(ctx).Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeResolveRecognition).Order("task_key asc").Find(&tasks).Error; err != nil {
		t.Fatalf("load resolve tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected two directory-scoped resolve tasks, got %#v", tasks)
	}
	if !strings.Contains(tasks[0].PayloadJSON, firstDir) || !strings.Contains(tasks[0].PayloadJSON, fmt.Sprintf("\"file_ids\":[%d", files[0].ID)) || !strings.Contains(tasks[0].PayloadJSON, fmt.Sprintf(",%d]", files[recognitionResolveScanBatchSize].ID)) {
		t.Fatalf("expected first task payload to target first directory, got %s", tasks[0].PayloadJSON)
	}
	if !strings.Contains(tasks[1].PayloadJSON, secondDir) {
		t.Fatalf("expected second task payload to target second directory, got %s", tasks[1].PayloadJSON)
	}
}

func TestRegisterWorkflowHandlersRegistersProbeAndMatchHandlers(t *testing.T) {
	ctx, _, svc := newWorkflowScanHarness(t)
	runner := workflow.NewRunner(workflow.NewService(svc.db), workflow.RunnerConfig{Enabled: true})
	svc.SetInventoryProbeExecutor(func(ctx context.Context, fileID uint) error { return nil })
	svc.SetMetadataMatchExecutor(func(ctx context.Context, metadataItemID uint, libraryID uint) error { return nil })
	svc.RegisterWorkflowHandlers(runner)

	probePayload := workflowPayloadJSON(t, InventoryProbeBatchPayload{LibraryID: 1, FileIDs: []uint{5}})
	if err := runner.HandleTaskForTest(ctx, database.WorkflowTask{TaskType: workflow.TaskTypeProbeInventory, PayloadJSON: probePayload}); err != nil {
		t.Fatalf("probe handler unavailable: %v", err)
	}
	matchPayload := workflowPayloadJSON(t, MetadataMatchBatchPayload{LibraryID: 1, MetadataItemIDs: []uint{8}})
	if err := runner.HandleTaskForTest(ctx, database.WorkflowTask{TaskType: workflow.TaskTypeMatchMetadata, PayloadJSON: matchPayload}); err != nil {
		t.Fatalf("match handler unavailable: %v", err)
	}
}

func workflowPayloadJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	return string(encoded)
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
	libraryRecord, _, err := svc.CreateLibrary(ctx, CreateLibraryInput{Name: name, MediaSourceID: source.ID, RootPath: rootPath})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	return libraryRecord
}
