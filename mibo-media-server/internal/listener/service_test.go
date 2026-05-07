package listener

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
)

func TestMergeWindowUsesOneQueuedIntentPerEvent(t *testing.T) {
	t.Parallel()

	svc, _, record := newListenerTestService(t)
	fixedNow := time.Date(2026, time.April, 24, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }

	ctx := context.Background()
	first, err := svc.RecordStorageEvent(ctx, EventIngestInput{LibraryID: record.ID, Kind: "update", Path: filepath.Join(record.RootPath, "Movies", "MovieA.2024.mkv")})
	if err != nil {
		t.Fatalf("record first event: %v", err)
	}
	second, err := svc.RecordStorageEvent(ctx, EventIngestInput{LibraryID: record.ID, Kind: "delete", Path: filepath.Join(record.RootPath, "Movies", "MovieA.2024.mkv")})
	if err != nil {
		t.Fatalf("record second event: %v", err)
	}

	if first.Kind != library.JobKindSyncLibrary || second.Kind != library.JobKindSyncLibrary {
		t.Fatalf("expected workflow compatibility job kind, got %q and %q", first.Kind, second.Kind)
	}
	if first.ID != second.ID {
		t.Fatalf("expected repeated events to reuse one queued workflow run, got %d and %d", first.ID, second.ID)
	}

	payload := mustDecodeWorkflowPayload(t, first.PayloadJSON)
	if payload.RootPath != filepath.Join(record.RootPath, "Movies") {
		t.Fatalf("expected merged root %q, got %q", filepath.Join(record.RootPath, "Movies"), payload.RootPath)
	}
}

func TestRecordStorageEventConcurrentDuplicatesKeepOneActiveIntent(t *testing.T) {
	t.Parallel()

	svc, db, record := newListenerTestService(t)
	fixedNow := time.Date(2026, time.April, 24, 10, 2, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }

	ctx := context.Background()
	const goroutines = 20
	errCh := make(chan error, goroutines)
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.RecordStorageEvent(ctx, EventIngestInput{LibraryID: record.ID, Kind: "update", Path: filepath.Join(record.RootPath, "Movies", "MovieA.2024.mkv")})
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("record concurrent storage event: %v", err)
		}
	}

	var active int64
	if err := db.WithContext(ctx).Model(&database.WorkflowRun{}).
		Where("reason = ? AND status IN ?", library.WorkflowReasonTargetedRefresh, []string{workflow.RunStatusQueued, workflow.RunStatusRunning}).
		Count(&active).Error; err != nil {
		t.Fatalf("count active listener refresh workflows: %v", err)
	}
	if active != 1 {
		t.Fatalf("expected one active listener refresh workflow, got %d", active)
	}
}

func TestRecordStorageEventUpdatesStorageIndexHint(t *testing.T) {
	t.Parallel()

	svc, db, record := newListenerTestService(t)
	ctx := context.Background()
	pathValue := filepath.Join(record.RootPath, "Movies", "MovieA.2024.mkv")
	if _, err := svc.RecordStorageEvent(ctx, EventIngestInput{LibraryID: record.ID, Kind: "create", Path: pathValue}); err != nil {
		t.Fatalf("record create event: %v", err)
	}
	var entry database.StorageIndexEntry
	if err := db.WithContext(ctx).Where("library_id = ? AND storage_path = ?", record.ID, pathValue).First(&entry).Error; err != nil {
		t.Fatalf("load storage index entry: %v", err)
	}
	if entry.ObservationStatus != "present" || entry.StorageProvider != "local" {
		t.Fatalf("expected present local index hint, got %#v", entry)
	}
	if _, err := svc.RecordStorageEvent(ctx, EventIngestInput{LibraryID: record.ID, Kind: "delete", Path: pathValue}); err != nil {
		t.Fatalf("record delete event: %v", err)
	}
	if err := db.WithContext(ctx).First(&entry, entry.ID).Error; err != nil {
		t.Fatalf("reload storage index entry: %v", err)
	}
	if entry.ObservationStatus != "missing" || entry.MissingSince == nil {
		t.Fatalf("expected missing index hint, got %#v", entry)
	}
}

func TestEnsureReconcileCoverageConcurrentCallsKeepOneActiveIntent(t *testing.T) {
	t.Parallel()

	svc, db, record := newListenerTestService(t)
	fixedNow := time.Date(2026, time.April, 24, 11, 2, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }

	ctx := context.Background()
	const goroutines = 20
	errCh := make(chan error, goroutines)
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- svc.EnsureReconcileCoverage(ctx, []database.Library{record})
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("ensure concurrent reconcile coverage: %v", err)
		}
	}

	assertNoListenerJobs(t, ctx, db)
}

func TestAncestorPromotionStaysInsideLibraryRoot(t *testing.T) {
	t.Parallel()

	svc, _, record := newListenerTestService(t)
	svc.now = func() time.Time { return time.Date(2026, time.April, 24, 10, 5, 0, 0, time.UTC) }

	ctx := context.Background()
	firstPath := filepath.Join(record.RootPath, "Movies", "Action", "MovieA.2024.mkv")
	secondPath := filepath.Join(record.RootPath, "Movies", "Drama", "MovieB.2024.mkv")
	first, err := svc.RecordStorageEvent(ctx, EventIngestInput{LibraryID: record.ID, Kind: "update", Path: firstPath})
	if err != nil {
		t.Fatalf("record first event: %v", err)
	}
	if _, err := svc.RecordStorageEvent(ctx, EventIngestInput{LibraryID: record.ID, Kind: "create", Path: secondPath}); err != nil {
		t.Fatalf("record second event: %v", err)
	}

	payload := mustDecodeWorkflowPayload(t, first.PayloadJSON)
	if payload.RootPath != filepath.Join(record.RootPath, "Movies", "Action") {
		t.Fatalf("expected immediate targeted root %q, got %q", filepath.Join(record.RootPath, "Movies", "Action"), payload.RootPath)
	}
}

func TestReconcileCoverageSchedulesOneFutureJobPerLibrary(t *testing.T) {
	t.Parallel()

	svc, db, first := newListenerTestService(t)
	ctx := context.Background()
	second := database.Library{Name: "Shows", Type: "shows", MediaSourceID: first.MediaSourceID, RootPath: filepath.Join(first.RootPath, "Shows"), Status: "active", ScannerEnabled: true}
	if err := db.WithContext(ctx).Create(&second).Error; err != nil {
		t.Fatalf("create second library: %v", err)
	}

	fixedNow := time.Date(2026, time.April, 24, 11, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }

	if err := svc.EnsureReconcileCoverage(ctx, []database.Library{first, second}); err != nil {
		t.Fatalf("ensure coverage: %v", err)
	}
	if err := svc.EnsureReconcileCoverage(ctx, []database.Library{first, second}); err != nil {
		t.Fatalf("ensure coverage idempotent: %v", err)
	}

	assertNoListenerJobs(t, ctx, db)
}

func TestRecordStorageEventFallsBackToFullSyncWhenNormalizationIsUnsafe(t *testing.T) {
	t.Parallel()

	svc, _, record := newListenerTestService(t)
	ctx := context.Background()
	job, err := svc.RecordStorageEvent(ctx, EventIngestInput{LibraryID: record.ID, Kind: "rename", Path: filepath.Join(record.RootPath, "Movies", "MovieA.2024.mkv")})
	if err != nil {
		t.Fatalf("record rename event: %v", err)
	}
	if job.Kind != library.JobKindSyncLibrary {
		t.Fatalf("expected workflow compatibility job kind, got %q", job.Kind)
	}
	payload := mustDecodeWorkflowPayload(t, job.PayloadJSON)
	if payload.LibraryID != record.ID || payload.Reason != library.WorkflowReasonStorageRefresh {
		t.Fatalf("expected fallback storage refresh payload, got %#v", payload)
	}
}

func TestApplyStorageEventRefreshQueuesExistingTargetedRefreshWork(t *testing.T) {
	t.Parallel()

	svc, db, record := newListenerIntegrationService(t)
	ctx := context.Background()
	listenerJob, err := svc.RecordStorageEvent(ctx, EventIngestInput{LibraryID: record.ID, Kind: "update", Path: filepath.Join(record.RootPath, "Movies", "MovieA.2024.mkv")})
	if err != nil {
		t.Fatalf("record storage event: %v", err)
	}

	if err := svc.ApplyStorageEventRefresh(ctx, listenerJob); err != nil {
		t.Fatalf("apply storage event refresh: %v", err)
	}

	assertWorkflowRefresh(t, ctx, db, library.WorkflowReasonTargetedRefresh)
}

func TestApplyStorageEventRefreshQueuesFullSyncForFallbackIntent(t *testing.T) {
	t.Parallel()

	svc, db, record := newListenerIntegrationService(t)
	ctx := context.Background()
	listenerJob, err := svc.RecordStorageEvent(ctx, EventIngestInput{LibraryID: record.ID, Kind: "rename", Path: filepath.Join(record.RootPath, "Movies", "MovieA.2024.mkv")})
	if err != nil {
		t.Fatalf("record fallback event: %v", err)
	}

	if err := svc.ApplyStorageEventRefresh(ctx, listenerJob); err != nil {
		t.Fatalf("apply fallback refresh: %v", err)
	}

	assertWorkflowRefresh(t, ctx, db, library.WorkflowReasonStorageRefresh)
}

func TestRunReconcileQueuesLibrarySyncAndReseedsNextWindow(t *testing.T) {
	t.Parallel()

	svc, db, record := newListenerIntegrationService(t)
	ctx := context.Background()
	baseNow := time.Date(2026, time.April, 24, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return baseNow }
	reconcileJob, err := createQueuedJob(reconcileJobKey(record.ID), JobKindListenerReconcile, reconcilePayload{LibraryID: record.ID, Reason: "listener_reconcile", ScheduledFor: baseNow}, baseNow.Add(-time.Minute))
	if err != nil {
		t.Fatalf("build reconcile job: %v", err)
	}
	reconcileJob.Status = "running"
	if err := db.WithContext(ctx).Create(&reconcileJob).Error; err != nil {
		t.Fatalf("store reconcile job: %v", err)
	}

	if err := svc.RunReconcile(ctx, reconcileJob); err != nil {
		t.Fatalf("run reconcile: %v", err)
	}

	assertWorkflowRefresh(t, ctx, db, library.WorkflowReasonStorageRefresh)
	assertNoListenerJobs(t, ctx, db)
}

func TestRecordStorageEventSkipsRefreshWhenRealtimePolicyDisabled(t *testing.T) {
	svc, db, record := newListenerIntegrationService(t)
	ctx := context.Background()
	if err := db.WithContext(ctx).Model(&database.LibraryScanPolicy{}).Where("library_id = ?", record.ID).Updates(map[string]any{"scanner_enabled": true, "realtime_monitor_enabled": false, "scheduled_refresh_enabled": true, "refresh_interval_hours": 24, "ignore_hidden_files": true, "ignore_file_extensions_json": `[]`, "configurable_exclusion_rules": true}).Error; err != nil {
		t.Fatalf("disable realtime policy: %v", err)
	}

	job, err := svc.RecordStorageEvent(ctx, EventIngestInput{LibraryID: record.ID, Kind: "update", Path: filepath.Join(record.RootPath, "Movies", "MovieA.2024.mkv")})
	if err != nil {
		t.Fatalf("record storage event: %v", err)
	}
	if job.ID != 0 {
		t.Fatalf("expected no listener job when realtime is disabled, got %#v", job)
	}
	var count int64
	if err := db.WithContext(ctx).Model(&database.Job{}).Where("kind = ?", JobKindApplyStorageEventRefresh).Count(&count).Error; err != nil {
		t.Fatalf("count listener jobs: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no listener refresh jobs, got %d", count)
	}
}

func newListenerTestService(t *testing.T) (*Service, *gorm.DB, database.Library) {
	t.Helper()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	source := database.MediaSource{Name: "Local", Provider: "local", StorageRef: filepath.Join(t.TempDir(), "storage"), RootPath: filepath.Join(t.TempDir(), "storage")}
	if err := db.WithContext(ctx).Create(&source).Error; err != nil {
		t.Fatalf("create media source: %v", err)
	}
	record := database.Library{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: filepath.Join(source.RootPath, "Library"), Status: "active", ScannerEnabled: true}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	if err := database.EnsureLibraryPolicyDefaults(db, record.ID); err != nil {
		t.Fatalf("ensure library policy defaults: %v", err)
	}
	cfg := config.Config{Local: config.LocalStorageConfig{RootPath: source.RootPath}}
	librarySvc := library.NewService(cfg, db, providers.NewRegistry(cfg), nil)
	return NewService(db, nil, librarySvc), db, record
}

func newListenerIntegrationService(t *testing.T) (*Service, *gorm.DB, database.Library) {
	t.Helper()

	storageRoot := filepath.Join(t.TempDir(), "storage")
	libraryRoot := filepath.Join(storageRoot, "Library")
	if err := os.MkdirAll(filepath.Join(libraryRoot, "Movies"), 0o755); err != nil {
		t.Fatalf("create library root: %v", err)
	}

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	cfg := config.Config{Local: config.LocalStorageConfig{RootPath: storageRoot}}
	registry := providers.NewRegistry(cfg)
	librarySvc := library.NewService(cfg, db, registry, nil)
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Local", RootPath: storageRoot})
	if err != nil {
		t.Fatalf("create media source: %v", err)
	}
	record, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", MediaSourceID: source.ID, RootPath: libraryRoot})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	if err := db.WithContext(ctx).Model(&database.Library{}).Where("id = ?", record.ID).Update("status", "active").Error; err != nil {
		t.Fatalf("activate library: %v", err)
	}
	if err := db.WithContext(ctx).Where("1 = 1").Delete(&database.Job{}).Error; err != nil {
		t.Fatalf("clear setup jobs: %v", err)
	}
	record.Status = "active"
	return NewService(db, nil, librarySvc), db, record
}

func mustDecodeWorkflowPayload(t *testing.T, raw string) struct {
	LibraryID uint   `json:"library_id"`
	RootPath  string `json:"root_path"`
	Reason    string `json:"reason"`
} {
	t.Helper()
	var payload struct {
		LibraryID uint   `json:"library_id"`
		RootPath  string `json:"root_path"`
		Reason    string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode workflow payload: %v", err)
	}
	return payload
}

func assertWorkflowRefresh(t *testing.T, ctx context.Context, db *gorm.DB, reason string) {
	t.Helper()
	var count int64
	if err := db.WithContext(ctx).Model(&database.WorkflowRun{}).Where("reason = ?", reason).Count(&count).Error; err != nil {
		t.Fatalf("count workflow refresh runs: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one %s workflow run, got %d", reason, count)
	}
}

func assertNoListenerJobs(t *testing.T, ctx context.Context, db *gorm.DB) {
	t.Helper()
	var count int64
	if err := db.WithContext(ctx).Model(&database.Job{}).
		Where("kind IN ? AND status != ?", []string{JobKindApplyStorageEventRefresh, JobKindListenerReconcile}, "running").
		Count(&count).Error; err != nil {
		t.Fatalf("count listener jobs: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no legacy listener jobs, got %d", count)
	}
}

func mustDecodeRefreshPayload(t *testing.T, raw string) storageEventRefreshPayload {
	t.Helper()
	var payload storageEventRefreshPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode refresh payload: %v", err)
	}
	return payload
}

func mustDecodeReconcilePayload(t *testing.T, raw string) reconcilePayload {
	t.Helper()
	var payload reconcilePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode reconcile payload: %v", err)
	}
	return payload
}
