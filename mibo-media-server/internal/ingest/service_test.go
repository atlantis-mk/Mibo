package ingest

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
)

func TestMarkInventoryFileDirtyUpsertsDirtyUnit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	file := seedIngestInventoryFile(t, ctx, db)
	svc := NewService(db)

	first, err := svc.MarkInventoryFileDirty(ctx, file.ID, "scan_discovered")
	if err != nil {
		t.Fatalf("mark first dirty: %v", err)
	}
	second, err := svc.MarkInventoryFileDirty(ctx, file.ID, "scan_refreshed")
	if err != nil {
		t.Fatalf("mark second dirty: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected dirty unit to be upserted, got first=%d second=%d", first.ID, second.ID)
	}
	if second.Reason != "scan_refreshed" || second.Status != DirtyStatusDirty || second.InventoryFileID == nil || *second.InventoryFileID != file.ID {
		t.Fatalf("unexpected dirty unit after upsert: %#v", second)
	}
}

func TestMarkInventoryFilesDirtyAndAppendEventsBatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	first := seedIngestInventoryFile(t, ctx, db)
	second := database.InventoryFile{LibraryID: first.LibraryID, StorageProvider: "local", StoragePath: "/media/second.mkv", ContentClass: "video", Status: "available", Container: "mkv"}
	if err := db.WithContext(ctx).Create(&second).Error; err != nil {
		t.Fatalf("create second file: %v", err)
	}
	svc := NewService(db)

	if err := svc.MarkInventoryFilesDirty(ctx, []uint{first.ID, second.ID, first.ID}, "scan_discovered"); err != nil {
		t.Fatalf("mark inventory files dirty: %v", err)
	}
	events := []database.IngestEvent{
		{UnitKey: "inventory_file:1", LibraryID: first.LibraryID, InventoryFileID: &first.ID, EventType: EventConditionChanged, Reason: "scanner_discovery"},
		{UnitKey: "inventory_file:2", LibraryID: second.LibraryID, InventoryFileID: &second.ID, EventType: EventConditionChanged, Reason: "scanner_discovery"},
	}
	if err := svc.AppendEvents(ctx, events); err != nil {
		t.Fatalf("append events: %v", err)
	}

	var dirtyUnits []database.IngestDirtyUnit
	if err := db.WithContext(ctx).Order("inventory_file_id asc").Find(&dirtyUnits).Error; err != nil {
		t.Fatalf("load dirty units: %v", err)
	}
	if len(dirtyUnits) != 2 {
		t.Fatalf("expected two dirty units, got %#v", dirtyUnits)
	}
	for _, unit := range dirtyUnits {
		if unit.InventoryFileID == nil || unit.Reason != "scan_discovered" || unit.Status != DirtyStatusDirty {
			t.Fatalf("unexpected dirty unit: %#v", unit)
		}
	}
	var eventCount int64
	if err := db.WithContext(ctx).Model(&database.IngestEvent{}).Count(&eventCount).Error; err != nil {
		t.Fatalf("count events: %v", err)
	}
	if eventCount != 2 {
		t.Fatalf("expected two appended events, got %d", eventCount)
	}
}

func TestMarkInventoryFilesDirtyAndAppendEventsHandlesSQLiteVariableLimit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	files := make([]database.InventoryFile, 0, 1200)
	for i := 0; i < 1200; i++ {
		files = append(files, database.InventoryFile{LibraryID: 1, StorageProvider: "local", StoragePath: "/media/bulk-" + strconv.Itoa(i) + ".mkv", ContentClass: "video", Status: "available", Container: "mkv"})
	}
	if err := db.WithContext(ctx).Create(&files).Error; err != nil {
		t.Fatalf("create files: %v", err)
	}
	svc := NewService(db)
	fileIDs := make([]uint, 0, len(files))
	events := make([]database.IngestEvent, 0, len(files))
	for _, file := range files {
		fileID := file.ID
		fileIDs = append(fileIDs, file.ID)
		events = append(events, database.IngestEvent{UnitKey: "inventory_file:" + strconv.Itoa(int(file.ID)), LibraryID: file.LibraryID, InventoryFileID: &fileID, EventType: EventConditionChanged, Reason: "scanner_discovery"})
	}

	if err := svc.MarkInventoryFilesDirty(ctx, fileIDs, "scan_discovered"); err != nil {
		t.Fatalf("mark inventory files dirty: %v", err)
	}
	if err := svc.AppendEvents(ctx, events); err != nil {
		t.Fatalf("append events: %v", err)
	}

	var dirtyCount int64
	if err := db.WithContext(ctx).Model(&database.IngestDirtyUnit{}).Count(&dirtyCount).Error; err != nil {
		t.Fatalf("count dirty units: %v", err)
	}
	if dirtyCount != int64(len(files)) {
		t.Fatalf("expected %d dirty units, got %d", len(files), dirtyCount)
	}
	var eventCount int64
	if err := db.WithContext(ctx).Model(&database.IngestEvent{}).Count(&eventCount).Error; err != nil {
		t.Fatalf("count events: %v", err)
	}
	if eventCount != int64(len(files)) {
		t.Fatalf("expected %d events, got %d", len(files), eventCount)
	}
}

func TestReconcileOnceDerivesConditionsFromFacts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	file := seedIngestInventoryFile(t, ctx, db)
	svc := NewService(db)
	if _, err := svc.MarkInventoryFileDirty(ctx, file.ID, "scan_discovered"); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}

	result, err := svc.ReconcileOnce(ctx, 10)
	if err != nil {
		t.Fatalf("reconcile once: %v", err)
	}
	if result.Claimed != 1 || result.Processed != 1 || result.Failed != 0 {
		t.Fatalf("unexpected reconcile result: %#v", result)
	}

	conditions := loadIngestConditions(t, ctx, db, "inventory_file:1")
	assertCondition(t, conditions, ConditionVisible, ConditionStatusTrue, "available")
	assertCondition(t, conditions, ConditionMaterialized, ConditionStatusPending, "awaiting_materialization")
	assertCondition(t, conditions, ConditionProbed, ConditionStatusPending, "awaiting_materialization")
	assertCondition(t, conditions, ConditionMetadataMatched, ConditionStatusPending, "awaiting_materialization")

	var eventCount int64
	if err := db.WithContext(ctx).Model(&database.IngestEvent{}).Where("unit_key = ?", "inventory_file:1").Count(&eventCount).Error; err != nil {
		t.Fatalf("count events: %v", err)
	}
	if eventCount == 0 {
		t.Fatalf("expected condition transition events")
	}
}

func TestReconcileOnceProcessesOnlyDirtyUnits(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	dirtyFile := seedIngestInventoryFile(t, ctx, db)
	cleanFile := database.InventoryFile{LibraryID: dirtyFile.LibraryID, StorageProvider: "local", StoragePath: "/media/clean.mkv", ContentClass: "video", Status: "available", Container: "mkv"}
	if err := db.WithContext(ctx).Create(&cleanFile).Error; err != nil {
		t.Fatalf("create clean file: %v", err)
	}
	svc := NewService(db)
	if _, err := svc.MarkInventoryFileDirty(ctx, dirtyFile.ID, "scan_discovered"); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}

	if _, err := svc.ReconcileOnce(ctx, 10); err != nil {
		t.Fatalf("reconcile once: %v", err)
	}

	var conditionCount int64
	if err := db.WithContext(ctx).Model(&database.IngestCondition{}).Where("unit_key = ?", "inventory_file:2").Count(&conditionCount).Error; err != nil {
		t.Fatalf("count clean file conditions: %v", err)
	}
	if conditionCount != 0 {
		t.Fatalf("expected clean non-dirty file to be ignored, got %d conditions", conditionCount)
	}
}

func TestReconcileUpdatesDriftedConditionsFromFacts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	file := seedIngestInventoryFile(t, ctx, db)
	stale := database.IngestCondition{UnitKey: "inventory_file:1", LibraryID: file.LibraryID, InventoryFileID: &file.ID, ConditionType: ConditionVisible, Status: ConditionStatusFalse, Reason: "stale", Severity: SeverityError}
	if err := db.WithContext(ctx).Create(&stale).Error; err != nil {
		t.Fatalf("create stale condition: %v", err)
	}
	svc := NewService(db)
	if _, err := svc.MarkInventoryFileDirty(ctx, file.ID, "repair"); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}
	if _, err := svc.ReconcileOnce(ctx, 10); err != nil {
		t.Fatalf("reconcile once: %v", err)
	}

	conditions := loadIngestConditions(t, ctx, db, "inventory_file:1")
	assertCondition(t, conditions, ConditionVisible, ConditionStatusTrue, "available")
}

func TestIngestEventRetention(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	file := seedIngestInventoryFile(t, ctx, db)
	svc := NewService(db)
	expiredAt := time.Now().UTC().Add(-time.Hour)
	keptAt := time.Now().UTC().Add(time.Hour)
	if _, err := svc.AppendEvent(ctx, database.IngestEvent{UnitKey: "inventory_file:1", LibraryID: file.LibraryID, InventoryFileID: &file.ID, EventType: EventConditionChanged, ExpiresAt: &expiredAt}); err != nil {
		t.Fatalf("append expired event: %v", err)
	}
	if _, err := svc.AppendEvent(ctx, database.IngestEvent{UnitKey: "inventory_file:1", LibraryID: file.LibraryID, InventoryFileID: &file.ID, EventType: EventRetryRequested, ExpiresAt: &keptAt}); err != nil {
		t.Fatalf("append kept event: %v", err)
	}
	deleted, err := svc.PruneExpiredEvents(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("prune events: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected one expired event deleted, got %d", deleted)
	}
	var remaining int64
	if err := db.WithContext(ctx).Model(&database.IngestEvent{}).Count(&remaining).Error; err != nil {
		t.Fatalf("count events: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("expected one event to remain, got %d", remaining)
	}
}

func TestReconcileDispatchesMissingMaterializationWork(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	file := seedIngestInventoryFile(t, ctx, db)
	svc := NewService(db)
	if _, err := svc.MarkInventoryFileDirty(ctx, file.ID, "scan_discovered"); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}

	if _, err := svc.ReconcileOnce(ctx, 10); err != nil {
		t.Fatalf("reconcile once: %v", err)
	}

	task := loadTaskByType(t, ctx, db, workflow.TaskTypeMaterializeCatalog)
	var payload CatalogMaterializeBatchPayload
	if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode materialize payload: %v", err)
	}
	if len(payload.FileIDs) != 1 || payload.FileIDs[0] != file.ID {
		t.Fatalf("unexpected materialize payload: %#v", payload)
	}
	conditions := loadIngestConditions(t, ctx, db, inventoryFileUnitKey(file.ID))
	materialized := conditions[ConditionMaterialized]
	if materialized.JobID == nil || *materialized.JobID != task.RunID {
		t.Fatalf("expected materialized condition to reference workflow run %d, got %#v", task.RunID, materialized.JobID)
	}
}

func TestRetryProbeDispatchesOnlyProbeWork(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	file, _ := seedLinkedIngestMovie(t, ctx, db, "error", "matched", false)
	svc := NewService(db)
	if _, err := svc.MarkInventoryFileDirty(ctx, file.ID, "admin_retry_"+ConditionProbed); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}

	if _, err := svc.ReconcileOnce(ctx, 10); err != nil {
		t.Fatalf("reconcile once: %v", err)
	}

	loadTaskByType(t, ctx, db, workflow.TaskTypeProbeInventory)
	assertNoTaskByType(t, ctx, db, workflow.TaskTypeMatchMetadata)
	assertNoTaskByType(t, ctx, db, workflow.TaskTypeMaterializeCatalog)
}

func TestProjectionDirtyScopeDispatchesProjectionRefresh(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	_, item := seedLinkedIngestMovie(t, ctx, db, "ready", "matched", false)
	svc := NewService(db)
	if _, err := svc.MarkProjectionItemDirty(ctx, item.ID, "metadata_applied"); err != nil {
		t.Fatalf("mark projection dirty: %v", err)
	}

	if _, err := svc.ReconcileOnce(ctx, 10); err != nil {
		t.Fatalf("reconcile once: %v", err)
	}

	task := loadTaskByType(t, ctx, db, workflow.TaskTypeRefreshProjection)
	var payload itemProjectionRefreshPayload
	if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode projection payload: %v", err)
	}
	if payload.ItemID != item.ID {
		t.Fatalf("unexpected projection payload: %#v", payload)
	}
	conditions := loadIngestConditions(t, ctx, db, catalogItemUnitKey(item.ID))
	projection := conditions[ConditionProjectionCurrent]
	if projection.Status != ConditionStatusPending || projection.Reason != "projection_refresh_queued" {
		t.Fatalf("expected projection condition queued, got %#v", projection)
	}
	if projection.JobID == nil || *projection.JobID != task.RunID {
		t.Fatalf("expected projection condition to reference workflow run %d, got %#v", task.RunID, projection.JobID)
	}
}

func TestProjectionLibraryDirtyOnlyDispatchesProjectionRefresh(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	file := seedIngestInventoryFile(t, ctx, db)
	svc := NewService(db)
	if _, err := svc.MarkProjectionLibraryDirty(ctx, file.LibraryID, "/media", "projection_refresh"); err != nil {
		t.Fatalf("mark projection library dirty: %v", err)
	}

	if _, err := svc.ReconcileOnce(ctx, 10); err != nil {
		t.Fatalf("reconcile once: %v", err)
	}

	loadTaskByType(t, ctx, db, workflow.TaskTypeRefreshProjection)
	assertNoTaskByType(t, ctx, db, workflow.TaskTypeMaterializeCatalog)
	assertNoTaskByType(t, ctx, db, workflow.TaskTypeProbeInventory)
	assertNoTaskByType(t, ctx, db, workflow.TaskTypeMatchMetadata)
}

func TestRetryLibraryProjectionConditionMarksProjectionScopeDirty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	file := seedIngestInventoryFile(t, ctx, db)
	svc := NewService(db)
	condition := database.IngestCondition{UnitKey: "library:" + strconv.FormatUint(uint64(file.LibraryID), 10), LibraryID: file.LibraryID, ConditionType: ConditionProjectionCurrent, Status: ConditionStatusFailed, Reason: "projection_failed", Severity: SeverityError}
	if err := db.WithContext(ctx).Create(&condition).Error; err != nil {
		t.Fatalf("create projection condition: %v", err)
	}

	if _, err := svc.RetryStage(ctx, condition.ID, nil); err != nil {
		t.Fatalf("retry projection condition: %v", err)
	}

	var dirty database.IngestDirtyUnit
	if err := db.WithContext(ctx).Where("library_id = ? AND scope_kind = ?", file.LibraryID, ScopeKindProjectionLibrary).First(&dirty).Error; err != nil {
		t.Fatalf("expected projection library dirty scope: %v", err)
	}
	if dirty.Reason != "admin_retry_projection" {
		t.Fatalf("unexpected dirty reason: %#v", dirty)
	}
}

func TestRetryDueConditionMarksTargetDirty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	file, _ := seedLinkedIngestMovie(t, ctx, db, "error", "matched", true)
	dueAt := time.Now().UTC().Add(-time.Minute)
	condition := database.IngestCondition{UnitKey: inventoryFileUnitKey(file.ID), LibraryID: file.LibraryID, InventoryFileID: &file.ID, ConditionType: ConditionProbed, Status: ConditionStatusFailed, Reason: "probe_failed", Severity: SeverityError, NextAttemptAt: &dueAt}
	if err := db.WithContext(ctx).Create(&condition).Error; err != nil {
		t.Fatalf("create retry due condition: %v", err)
	}
	svc := NewService(db)

	if _, err := svc.ReconcileOnce(ctx, 10); err != nil {
		t.Fatalf("reconcile once: %v", err)
	}

	loadTaskByType(t, ctx, db, workflow.TaskTypeProbeInventory)
	assertNoTaskByType(t, ctx, db, workflow.TaskTypeMatchMetadata)
	assertNoTaskByType(t, ctx, db, workflow.TaskTypeMaterializeCatalog)
	var stored database.IngestCondition
	if err := db.WithContext(ctx).First(&stored, condition.ID).Error; err != nil {
		t.Fatalf("reload retry due condition: %v", err)
	}
	if stored.NextAttemptAt != nil {
		t.Fatalf("expected retry due next_attempt_at to be cleared, got %#v", stored.NextAttemptAt)
	}
}

func TestNoCandidateRequiresReview(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	file, item := seedLinkedIngestMovie(t, ctx, db, "ready", "unmatched", true)
	seedMetadataNoCandidateOperation(t, ctx, db, item)
	svc := NewService(db)
	if _, err := svc.MarkInventoryFileDirty(ctx, file.ID, "metadata_no_candidate"); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}
	if _, err := svc.ReconcileOnce(ctx, 10); err != nil {
		t.Fatalf("reconcile once: %v", err)
	}

	conditions := loadIngestConditions(t, ctx, db, inventoryFileUnitKey(file.ID))
	assertCondition(t, conditions, ConditionMetadataMatched, ConditionStatusFalse, "no_candidate")
	assertCondition(t, conditions, ConditionReviewRequired, ConditionStatusReviewRequired, "metadata_no_candidate")
	reviewCondition := conditions[ConditionReviewRequired]
	if reviewCondition.CatalogItemID == nil || *reviewCondition.CatalogItemID != item.ID {
		t.Fatalf("expected metadata review condition to reference catalog item %d, got %#v", item.ID, reviewCondition.CatalogItemID)
	}
}

func TestManualGovernanceClearsNoCandidateReview(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	file, item := seedLinkedIngestMovie(t, ctx, db, "ready", "unmatched", true)
	seedMetadataNoCandidateOperation(t, ctx, db, item)
	svc := NewService(db)
	if _, err := svc.MarkInventoryFileDirty(ctx, file.ID, "metadata_no_candidate"); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}
	if _, err := svc.ReconcileOnce(ctx, 10); err != nil {
		t.Fatalf("reconcile no candidate: %v", err)
	}
	conditions := loadIngestConditions(t, ctx, db, inventoryFileUnitKey(file.ID))
	assertCondition(t, conditions, ConditionMetadataMatched, ConditionStatusFalse, "no_candidate")
	assertCondition(t, conditions, ConditionReviewRequired, ConditionStatusReviewRequired, "metadata_no_candidate")

	if err := db.WithContext(ctx).Model(&database.CatalogItem{}).Where("id = ?", item.ID).Update("governance_status", "manual").Error; err != nil {
		t.Fatalf("mark item manual: %v", err)
	}
	if _, err := svc.MarkInventoryFileDirty(ctx, file.ID, "metadata_applied"); err != nil {
		t.Fatalf("mark dirty after manual governance: %v", err)
	}
	if _, err := svc.ReconcileOnce(ctx, 10); err != nil {
		t.Fatalf("reconcile manual governance: %v", err)
	}

	conditions = loadIngestConditions(t, ctx, db, inventoryFileUnitKey(file.ID))
	assertCondition(t, conditions, ConditionMetadataMatched, ConditionStatusTrue, "matched")
	assertCondition(t, conditions, ConditionReviewRequired, ConditionStatusFalse, "not_required")
}

func TestMatchedCatalogItemSuppressesClassificationReview(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	file, item := seedLinkedIngestMovie(t, ctx, db, "ready", "matched", true)
	decision := database.ClassificationDecision{LibraryID: file.LibraryID, InventoryFileID: &file.ID, ItemID: &item.ID, SourcePath: file.StoragePath, DecisionType: "series_group", CandidateType: "episode", TargetKind: "series", TargetKey: "Show", Status: "provisional"}
	if err := db.WithContext(ctx).Create(&decision).Error; err != nil {
		t.Fatalf("create classification decision: %v", err)
	}
	svc := NewService(db)
	if _, err := svc.MarkInventoryFileDirty(ctx, file.ID, "metadata_applied"); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}
	if _, err := svc.ReconcileOnce(ctx, 10); err != nil {
		t.Fatalf("reconcile once: %v", err)
	}

	conditions := loadIngestConditions(t, ctx, db, inventoryFileUnitKey(file.ID))
	assertCondition(t, conditions, ConditionMetadataMatched, ConditionStatusTrue, "matched")
	assertCondition(t, conditions, ConditionReviewRequired, ConditionStatusFalse, "not_required")
}

func TestDiagnosticsSkipsMetadataReviewWithoutCatalogReference(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	file := seedIngestInventoryFile(t, ctx, db)
	condition := database.IngestCondition{
		UnitKey:         inventoryFileUnitKey(file.ID),
		LibraryID:       file.LibraryID,
		InventoryFileID: &file.ID,
		ConditionType:   ConditionReviewRequired,
		Status:          ConditionStatusReviewRequired,
		Reason:          "metadata_no_candidate",
		Severity:        SeverityWarning,
		Message:         "No acceptable metadata candidate was found",
	}
	if err := db.WithContext(ctx).Create(&condition).Error; err != nil {
		t.Fatalf("create invalid metadata review condition: %v", err)
	}
	svc := NewService(db)

	diagnostics, err := svc.Diagnostics(ctx, DiagnosticsInput{})
	if err != nil {
		t.Fatalf("load diagnostics: %v", err)
	}
	if len(diagnostics.Stages) != 0 || diagnostics.Summary.ReviewRequired != 0 {
		t.Fatalf("expected invalid metadata review condition to be hidden, got %#v", diagnostics)
	}
}

func TestDiagnosticsSkipsNonActionableWarnings(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	file := seedIngestInventoryFile(t, ctx, db)
	warning := database.IngestCondition{
		UnitKey:         inventoryFileUnitKey(file.ID),
		LibraryID:       file.LibraryID,
		InventoryFileID: &file.ID,
		ConditionType:   ConditionVisible,
		Status:          ConditionStatusFalse,
		Reason:          "not_available",
		Severity:        SeverityWarning,
		Message:         "Catalog media is not available",
	}
	failed := database.IngestCondition{
		UnitKey:         inventoryFileUnitKey(file.ID),
		LibraryID:       file.LibraryID,
		InventoryFileID: &file.ID,
		ConditionType:   ConditionProbed,
		Status:          ConditionStatusFailed,
		Reason:          "probe_failed",
		Severity:        SeverityError,
		Message:         "Media probe failed",
	}
	if err := db.WithContext(ctx).Create(&[]database.IngestCondition{warning, failed}).Error; err != nil {
		t.Fatalf("create ingest conditions: %v", err)
	}
	svc := NewService(db)

	diagnostics, err := svc.Diagnostics(ctx, DiagnosticsInput{})
	if err != nil {
		t.Fatalf("load diagnostics: %v", err)
	}
	if len(diagnostics.Stages) != 1 || diagnostics.Stages[0].ConditionType != ConditionProbed {
		t.Fatalf("expected only actionable failure, got %#v", diagnostics.Stages)
	}
	if diagnostics.Summary.Failed != 1 {
		t.Fatalf("expected failed summary to count actionable failure, got %#v", diagnostics.Summary)
	}
}

func TestNoCandidateWithoutMetadataOperationDoesNotRequireReview(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	file, _ := seedLinkedIngestMovie(t, ctx, db, "ready", "unmatched", true)
	svc := NewService(db)
	if _, err := svc.MarkInventoryFileDirty(ctx, file.ID, "metadata_no_candidate"); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}
	if _, err := svc.ReconcileOnce(ctx, 10); err != nil {
		t.Fatalf("reconcile once: %v", err)
	}

	conditions := loadIngestConditions(t, ctx, db, inventoryFileUnitKey(file.ID))
	assertCondition(t, conditions, ConditionMetadataMatched, ConditionStatusSkipped, "not_required")
	assertCondition(t, conditions, ConditionReviewRequired, ConditionStatusFalse, "not_required")
	diagnostics, err := svc.Diagnostics(ctx, DiagnosticsInput{})
	if err != nil {
		t.Fatalf("load diagnostics: %v", err)
	}
	if diagnostics.Summary.ReviewRequired != 0 {
		t.Fatalf("expected unmatched item without metadata operation not to require review, got %#v", diagnostics.Summary)
	}
}

func TestNoCandidateWithDisabledSearchStrategyDoesNotRequireReview(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	file, item := seedLinkedIngestMovie(t, ctx, db, "ready", "unmatched", true)
	if err := db.WithContext(ctx).Create(&database.MetadataOperation{LibraryID: item.LibraryID, Operation: "match", TargetItemID: item.ID, Status: "no_candidate"}).Error; err != nil {
		t.Fatalf("create metadata operation: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.LibraryMetadataStrategy{LibraryID: item.LibraryID, SearchProvidersJSON: "[]"}).Error; err != nil {
		t.Fatalf("create empty metadata strategy: %v", err)
	}
	svc := NewService(db)
	if _, err := svc.MarkInventoryFileDirty(ctx, file.ID, "metadata_no_candidate"); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}
	if _, err := svc.ReconcileOnce(ctx, 10); err != nil {
		t.Fatalf("reconcile once: %v", err)
	}

	conditions := loadIngestConditions(t, ctx, db, inventoryFileUnitKey(file.ID))
	assertCondition(t, conditions, ConditionMetadataMatched, ConditionStatusSkipped, "not_required")
	assertCondition(t, conditions, ConditionReviewRequired, ConditionStatusFalse, "not_required")
}

func TestBackfillIngestDirtyScopesMarksLibraryPaths(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openIngestTestDB(t)
	source := database.MediaSource{Name: "Local", Provider: "local", RootPath: "/media"}
	if err := db.WithContext(ctx).Create(&source).Error; err != nil {
		t.Fatalf("create media source: %v", err)
	}
	library := database.Library{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: "/media/Movies"}
	if err := db.WithContext(ctx).Create(&library).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	path := database.LibraryPath{LibraryID: library.ID, MediaSourceID: source.ID, RootPath: library.RootPath, Enabled: true}
	if err := db.WithContext(ctx).Create(&path).Error; err != nil {
		t.Fatalf("create library path: %v", err)
	}
	if err := database.BackfillIngestDirtyScopes(db); err != nil {
		t.Fatalf("backfill dirty scopes: %v", err)
	}

	var unit database.IngestDirtyUnit
	if err := db.WithContext(ctx).Where("scope_kind = ? AND library_id = ?", ScopeKindLibrary, library.ID).First(&unit).Error; err != nil {
		t.Fatalf("load backfilled dirty scope: %v", err)
	}
	if unit.RootPath != library.RootPath || unit.Reason != "startup_backfill" {
		t.Fatalf("unexpected backfilled dirty scope: %#v", unit)
	}
}

func openIngestTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	return db
}

func seedIngestInventoryFile(t *testing.T, ctx context.Context, db *gorm.DB) database.InventoryFile {
	t.Helper()
	source := database.MediaSource{Name: "Local", Provider: "local", RootPath: "/media"}
	if err := db.WithContext(ctx).Create(&source).Error; err != nil {
		t.Fatalf("create media source: %v", err)
	}
	library := database.Library{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: "/media"}
	if err := db.WithContext(ctx).Create(&library).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	file := database.InventoryFile{LibraryID: library.ID, StorageProvider: "local", StoragePath: "/media/Movie.mkv", ContentClass: "video", Status: "available", Container: "mkv"}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	return file
}

func seedLinkedIngestMovie(t *testing.T, ctx context.Context, db *gorm.DB, probeStatus string, governanceStatus string, withProjection bool) (database.InventoryFile, database.CatalogItem) {
	t.Helper()
	file := seedIngestInventoryFile(t, ctx, db)
	asset := database.MediaAsset{LibraryID: file.LibraryID, AssetType: "main", DisplayName: "Movie", Status: "available", ProbeStatus: probeStatus}
	if err := db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	item := database.CatalogItem{LibraryID: file.LibraryID, Type: "movie", Path: file.StoragePath, SortKey: "Movie", DisplayOrder: "aired", Title: "Movie", AvailabilityStatus: "available", GovernanceStatus: governanceStatus}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: "source"}).Error; err != nil {
		t.Fatalf("link asset file: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.AssetItem{AssetID: asset.ID, ItemID: item.ID, Role: "primary", SegmentIndex: 0}).Error; err != nil {
		t.Fatalf("link asset item: %v", err)
	}
	if withProjection {
		doc := database.CatalogSearchDocument{ItemID: item.ID, LibraryID: item.LibraryID, ItemType: item.Type, Title: item.Title, AvailabilityStatus: item.AvailabilityStatus, UpdatedAt: time.Now().UTC()}
		if err := db.WithContext(ctx).Create(&doc).Error; err != nil {
			t.Fatalf("create projection: %v", err)
		}
	}
	return file, item
}

func seedMetadataNoCandidateOperation(t *testing.T, ctx context.Context, db *gorm.DB, item database.CatalogItem) {
	t.Helper()
	operation := database.MetadataOperation{Operation: "match", OriginItemID: item.ID, TargetItemID: item.ID, LibraryID: item.LibraryID, Status: "no_candidate", GovernanceStatus: "unmatched", StartedAt: time.Now().UTC()}
	if err := db.WithContext(ctx).Create(&operation).Error; err != nil {
		t.Fatalf("create metadata operation: %v", err)
	}
}

func loadIngestConditions(t *testing.T, ctx context.Context, db *gorm.DB, unitKey string) map[string]database.IngestCondition {
	t.Helper()
	var rows []database.IngestCondition
	if err := db.WithContext(ctx).Where("unit_key = ?", unitKey).Find(&rows).Error; err != nil {
		t.Fatalf("load ingest conditions: %v", err)
	}
	result := make(map[string]database.IngestCondition, len(rows))
	for _, row := range rows {
		result[row.ConditionType] = row
	}
	return result
}

func assertCondition(t *testing.T, conditions map[string]database.IngestCondition, conditionType string, status string, reason string) {
	t.Helper()
	condition, ok := conditions[conditionType]
	if !ok {
		t.Fatalf("missing condition %q in %#v", conditionType, conditions)
	}
	if condition.Status != status || condition.Reason != reason {
		t.Fatalf("unexpected condition %q: want status=%q reason=%q got %#v", conditionType, status, reason, condition)
	}
}

func loadTaskByType(t *testing.T, ctx context.Context, db *gorm.DB, taskType string) database.WorkflowTask {
	t.Helper()
	var task database.WorkflowTask
	if err := db.WithContext(ctx).Where("task_type = ?", taskType).Order("id desc").First(&task).Error; err != nil {
		t.Fatalf("load workflow task type %s: %v", taskType, err)
	}
	return task
}

func assertNoTaskByType(t *testing.T, ctx context.Context, db *gorm.DB, taskType string) {
	t.Helper()
	var count int64
	if err := db.WithContext(ctx).Model(&database.WorkflowTask{}).Where("task_type = ?", taskType).Count(&count).Error; err != nil {
		t.Fatalf("count workflow task type %s: %v", taskType, err)
	}
	if count != 0 {
		t.Fatalf("expected no %s workflow task, got %d", taskType, count)
	}
}
