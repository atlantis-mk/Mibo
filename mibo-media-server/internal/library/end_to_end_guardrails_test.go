package library

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
)

func TestListenerRefreshToScanToProjectionEndToEnd(t *testing.T) {
	ctx, db, svc := newWorkflowScanHarness(t)
	libraryRecord := createWorkflowScanLibrary(t, ctx, svc, "Movies", LibraryTypeAuto)
	moviePath := filepath.Join(libraryRecord.RootPath, "Movie A.2024.mkv")
	mustWriteFixtureFile(t, moviePath)

	run, _, err := svc.QueueLibraryWorkflow(ctx, QueueWorkflowInput{
		LibraryID: libraryRecord.ID,
		RootPath:  libraryRecord.RootPath,
		Reason:    WorkflowReasonStorageRefresh,
		Priority:  10,
	})
	if err != nil {
		t.Fatalf("queue storage refresh workflow: %v", err)
	}

	var scanTask database.WorkflowTask
	if err := db.WithContext(ctx).
		Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeScanLibraryPath).
		First(&scanTask).Error; err != nil {
		t.Fatalf("load scan task: %v", err)
	}
	if err := svc.RunWorkflowScanLibraryPath(ctx, scanTask); err != nil {
		t.Fatalf("run scan task: %v", err)
	}
	if err := runQueuedRecognitionResolveTasks(ctx, db, svc, run.ID); err != nil {
		t.Fatalf("run resolve tasks: %v", err)
	}

	var projectionTask database.WorkflowTask
	if err := db.WithContext(ctx).
		Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeRefreshProjection).
		First(&projectionTask).Error; err != nil {
		t.Fatalf("load projection task: %v", err)
	}
	if err := svc.RunWorkflowCatalogProjectionRefresh(ctx, projectionTask); err != nil {
		t.Fatalf("run projection task: %v", err)
	}

	var item database.MetadataItem
	if err := db.WithContext(ctx).Where("title = ?", "Movie A").First(&item).Error; err != nil {
		t.Fatalf("load metadata item: %v", err)
	}

	projectionSvc := catalog.NewService(db)
	browseItems, err := projectionSvc.ListLibraryProjectionItems(ctx, libraryRecord.ID, "Movie A", "movie", 20)
	if err != nil {
		t.Fatalf("list projection items: %v", err)
	}
	if len(browseItems) != 1 || browseItems[0].MetadataItemID != item.ID {
		t.Fatalf("unexpected projection browse items: %#v", browseItems)
	}

	recentItems, err := projectionSvc.ListRecentlyAdded(ctx, 10)
	if err != nil {
		t.Fatalf("list recently added: %v", err)
	}
	if len(recentItems) == 0 || recentItems[0].MetadataItemID != item.ID {
		t.Fatalf("expected recently added to expose scanned item, got %#v", recentItems)
	}

	sections, err := projectionSvc.ListHomeContentSections(ctx, 10)
	if err != nil {
		t.Fatalf("list home content sections: %v", err)
	}
	if len(sections) != 1 || len(sections[0].Items) != 1 || sections[0].Items[0].MetadataItemID != item.ID {
		t.Fatalf("expected home content sections to expose scanned item, got %#v", sections)
	}
}

func TestTargetedRefreshWorkflowKeepsScopedRootInProjectionTask(t *testing.T) {
	ctx, db, svc := newWorkflowScanHarness(t)
	libraryRecord := createWorkflowScanLibrary(t, ctx, svc, "Movies", LibraryTypeAuto)
	scopedRoot := filepath.Join(libraryRecord.RootPath, "Collection")
	mustWriteFixtureFile(t, filepath.Join(scopedRoot, "Movie B.2024.mkv"))

	run, _, err := svc.QueueLibraryWorkflow(ctx, QueueWorkflowInput{
		LibraryID: libraryRecord.ID,
		RootPath:  scopedRoot,
		Reason:    WorkflowReasonTargetedRefresh,
		Priority:  10,
	})
	if err != nil {
		t.Fatalf("queue targeted refresh workflow: %v", err)
	}

	var scanTask database.WorkflowTask
	if err := db.WithContext(ctx).
		Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeScanLibraryPath).
		First(&scanTask).Error; err != nil {
		t.Fatalf("load scan task: %v", err)
	}
	if err := svc.RunWorkflowScanLibraryPath(ctx, scanTask); err != nil {
		t.Fatalf("run scan task: %v", err)
	}
	if err := runQueuedRecognitionResolveTasks(ctx, db, svc, run.ID); err != nil {
		t.Fatalf("run resolve tasks: %v", err)
	}

	var projectionTask database.WorkflowTask
	if err := db.WithContext(ctx).
		Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeRefreshProjection).
		First(&projectionTask).Error; err != nil {
		t.Fatalf("load projection task: %v", err)
	}
	var payload struct {
		LibraryID uint   `json:"library_id"`
		RootPath  string `json:"root_path"`
	}
	if err := json.Unmarshal([]byte(projectionTask.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode projection payload: %v", err)
	}
	if payload.RootPath != scopedRoot {
		t.Fatalf("expected scoped root %q, got %#v", scopedRoot, payload)
	}
}

func TestGovernanceVisibilityRemovesHomeProjectionAfterRefresh(t *testing.T) {
	ctx, db, svc := newWorkflowScanHarness(t)
	libraryRecord := createWorkflowScanLibrary(t, ctx, svc, "Movies", LibraryTypeAuto)
	moviePath := filepath.Join(libraryRecord.RootPath, "Movie Hidden.2024.mkv")
	mustWriteFixtureFile(t, moviePath)

	run, _, err := svc.QueueLibraryWorkflow(ctx, QueueWorkflowInput{LibraryID: libraryRecord.ID, Reason: WorkflowReasonManualScan, Priority: 10})
	if err != nil {
		t.Fatalf("queue workflow: %v", err)
	}
	var scanTask database.WorkflowTask
	if err := db.WithContext(ctx).
		Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeScanLibraryPath).
		First(&scanTask).Error; err != nil {
		t.Fatalf("load scan task: %v", err)
	}
	if err := svc.RunWorkflowScanLibraryPath(ctx, scanTask); err != nil {
		t.Fatalf("run scan task: %v", err)
	}
	if err := runQueuedRecognitionResolveTasks(ctx, db, svc, run.ID); err != nil {
		t.Fatalf("run resolve tasks: %v", err)
	}
	var projectionTask database.WorkflowTask
	if err := db.WithContext(ctx).
		Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeRefreshProjection).
		First(&projectionTask).Error; err != nil {
		t.Fatalf("load projection task: %v", err)
	}
	if err := svc.RunWorkflowCatalogProjectionRefresh(ctx, projectionTask); err != nil {
		t.Fatalf("run projection task: %v", err)
	}

	var item database.MetadataItem
	if err := db.WithContext(ctx).Where("title = ?", "Movie Hidden").First(&item).Error; err != nil {
		t.Fatalf("load metadata item: %v", err)
	}
	if err := db.WithContext(ctx).
		Model(&database.LibraryMetadataProjection{}).
		Where("library_id = ? AND metadata_item_id = ?", libraryRecord.ID, item.ID).
		Update("hidden", true).Error; err != nil {
		t.Fatalf("hide projection: %v", err)
	}

	projectionSvc := catalog.NewService(db)
	recentItems, err := projectionSvc.ListRecentlyAdded(ctx, 10)
	if err != nil {
		t.Fatalf("list recently added: %v", err)
	}
	for _, entry := range recentItems {
		if entry.MetadataItemID == item.ID {
			t.Fatalf("expected hidden item removed from recently added, got %#v", recentItems)
		}
	}

	sections, err := projectionSvc.ListHomeContentSections(ctx, 10)
	if err != nil {
		t.Fatalf("list home content sections: %v", err)
	}
	for _, section := range sections {
		for _, entry := range section.Items {
			if entry.MetadataItemID == item.ID {
				t.Fatalf("expected hidden item removed from home content sections, got %#v", sections)
			}
		}
	}
}

func TestMaterializedPlaybackStillResolvesAfterProjectionRefresh(t *testing.T) {
	ctx, db, svc := newWorkflowScanHarness(t)
	libraryRecord := createWorkflowScanLibrary(t, ctx, svc, "Movies", LibraryTypeAuto)
	moviePath := filepath.Join(libraryRecord.RootPath, "Movie Playback.2024.mp4")
	if err := os.MkdirAll(filepath.Dir(moviePath), 0o755); err != nil {
		t.Fatalf("create playback fixture dir: %v", err)
	}
	if err := os.WriteFile(moviePath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write playback fixture file: %v", err)
	}

	run, _, err := svc.QueueLibraryWorkflow(ctx, QueueWorkflowInput{LibraryID: libraryRecord.ID, Reason: WorkflowReasonManualScan, Priority: 10})
	if err != nil {
		t.Fatalf("queue workflow: %v", err)
	}
	var scanTask database.WorkflowTask
	if err := db.WithContext(ctx).
		Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeScanLibraryPath).
		First(&scanTask).Error; err != nil {
		t.Fatalf("load scan task: %v", err)
	}
	if err := svc.RunWorkflowScanLibraryPath(ctx, scanTask); err != nil {
		t.Fatalf("run scan task: %v", err)
	}
	if err := runQueuedRecognitionResolveTasks(ctx, db, svc, run.ID); err != nil {
		t.Fatalf("run resolve tasks: %v", err)
	}
	var projectionTask database.WorkflowTask
	if err := db.WithContext(ctx).
		Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeRefreshProjection).
		First(&projectionTask).Error; err != nil {
		t.Fatalf("load projection task: %v", err)
	}
	if err := svc.RunWorkflowCatalogProjectionRefresh(ctx, projectionTask); err != nil {
		t.Fatalf("run projection task: %v", err)
	}

	var item database.MetadataItem
	if err := db.WithContext(ctx).Where("title = ?", "Movie Playback").First(&item).Error; err != nil {
		t.Fatalf("load metadata item: %v", err)
	}
	playbackSvc := newPlaybackServiceForWorkflowHarness(t, db, svc)
	source, err := playbackSvc.GetPlaybackSource(ctx, playbackRequestForLibrary(item.ID, libraryRecord.ID))
	if err != nil {
		t.Fatalf("get playback source: %v", err)
	}
	if !source.Playable || source.MetadataItemID != item.ID || source.URL == "" {
		t.Fatalf("unexpected playback source after projection refresh: %#v", source)
	}
}

func TestScanWorkflowCompletesWithoutRunningMetadataMatchOrProbe(t *testing.T) {
	ctx, db, svc := newWorkflowScanHarness(t)
	libraryRecord := createWorkflowScanLibrary(t, ctx, svc, "TV", LibraryTypeAuto)
	showPath := filepath.Join(libraryRecord.RootPath, "Show", "Season 1", "01.mkv")
	mustWriteFixtureFile(t, showPath)

	svc.SetMetadataMatchExecutor(func(ctx context.Context, metadataItemID uint, libraryID uint) error {
		return fmt.Errorf("metadata match should not be required for scan completion")
	})
	svc.SetInventoryProbeExecutor(func(ctx context.Context, fileID uint) error {
		return fmt.Errorf("inventory probe should not be required for scan completion")
	})

	run, _, err := svc.QueueLibraryWorkflow(ctx, QueueWorkflowInput{LibraryID: libraryRecord.ID, Reason: WorkflowReasonManualScan, Priority: 10})
	if err != nil {
		t.Fatalf("queue workflow: %v", err)
	}
	var scanTask database.WorkflowTask
	if err := db.WithContext(ctx).
		Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeScanLibraryPath).
		First(&scanTask).Error; err != nil {
		t.Fatalf("load scan task: %v", err)
	}
	if err := svc.RunWorkflowScanLibraryPath(ctx, scanTask); err != nil {
		t.Fatalf("run scan task: %v", err)
	}
	if err := runQueuedRecognitionResolveTasks(ctx, db, svc, run.ID); err != nil {
		t.Fatalf("run recognition resolve tasks: %v", err)
	}

	var projectionTask database.WorkflowTask
	if err := db.WithContext(ctx).
		Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeRefreshProjection).
		First(&projectionTask).Error; err != nil {
		t.Fatalf("load projection task: %v", err)
	}
	if err := svc.RunWorkflowCatalogProjectionRefresh(ctx, projectionTask); err != nil {
		t.Fatalf("run projection task: %v", err)
	}

	var seriesCount int64
	if err := db.WithContext(ctx).Model(&database.MetadataItem{}).Where("item_type = ?", database.MetadataItemTypeSeries).Count(&seriesCount).Error; err != nil {
		t.Fatalf("count series items: %v", err)
	}
	if seriesCount == 0 {
		t.Fatalf("expected scan/resolve/projection to publish series metadata before metadata match")
	}
	var matchTasks int64
	if err := db.WithContext(ctx).Model(&database.WorkflowTask{}).Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeMatchMetadata).Count(&matchTasks).Error; err != nil {
		t.Fatalf("count match tasks: %v", err)
	}
	if matchTasks == 0 {
		t.Fatalf("expected metadata match tasks to be queued as follow-up work")
	}
	var probeTasks int64
	if err := db.WithContext(ctx).Model(&database.WorkflowTask{}).Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeProbeInventory).Count(&probeTasks).Error; err != nil {
		t.Fatalf("count probe tasks: %v", err)
	}
	if probeTasks == 0 {
		t.Fatalf("expected probe tasks to be queued as follow-up work")
	}
}

func newPlaybackServiceForWorkflowHarness(t *testing.T, db *gorm.DB, svc *Service) *playback.Service {
	t.Helper()
	return playback.NewService(db, providers.NewRegistry(svc.cfg))
}

func playbackRequestForLibrary(metadataItemID uint, libraryID uint) playback.PlaybackRequest {
	return playback.PlaybackRequest{MetadataItemID: metadataItemID, LibraryID: libraryID, ClientProfile: playback.ClientProfileWeb}
}

func runQueuedRecognitionResolveTasks(ctx context.Context, db *gorm.DB, svc *Service, runID uint) error {
	var tasks []database.WorkflowTask
	if err := db.WithContext(ctx).
		Where("run_id = ? AND task_type = ?", runID, workflow.TaskTypeResolveRecognition).
		Order("id asc").
		Find(&tasks).Error; err != nil {
		return err
	}
	for _, task := range tasks {
		if err := svc.RunWorkflowRecognitionResolve(ctx, task); err != nil {
			return err
		}
	}
	return nil
}
