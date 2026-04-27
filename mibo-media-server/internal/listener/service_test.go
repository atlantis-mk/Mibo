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
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/providers"
	"gorm.io/gorm"
)

func TestMergeWindowUsesOneQueuedIntentPerEvent(t *testing.T) {
	t.Parallel()

	svc, db, record := newListenerTestService(t)
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

	if first.Kind != JobKindApplyStorageEventRefresh || second.Kind != JobKindApplyStorageEventRefresh {
		t.Fatalf("expected %q job kind, got %q and %q", JobKindApplyStorageEventRefresh, first.Kind, second.Kind)
	}
	if first.ID != second.ID {
		t.Fatalf("expected repeated events to reuse one queued listener job, got %d and %d", first.ID, second.ID)
	}

	var jobs []database.Job
	if err := db.WithContext(ctx).Where("kind = ?", JobKindApplyStorageEventRefresh).Order("id asc").Find(&jobs).Error; err != nil {
		t.Fatalf("list listener jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected one debounced listener job, got %d", len(jobs))
	}

	payload := mustDecodeRefreshPayload(t, jobs[0].PayloadJSON)
	if payload.RootPath != filepath.Join(record.RootPath, "Movies") {
		t.Fatalf("expected merged root %q, got %q", filepath.Join(record.RootPath, "Movies"), payload.RootPath)
	}
	if payload.WindowEndsAt.Sub(payload.WindowStartedAt) != 15*time.Second {
		t.Fatalf("expected 15s merge window, got %s", payload.WindowEndsAt.Sub(payload.WindowStartedAt))
	}
	if !jobs[0].AvailableAt.Equal(payload.WindowEndsAt) {
		t.Fatalf("expected available_at to match window end")
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
	if err := db.WithContext(ctx).Model(&database.Job{}).
		Where("kind = ? AND status IN ?", JobKindApplyStorageEventRefresh, []string{jobs.StatusQueued, jobs.StatusRunning}).
		Count(&active).Error; err != nil {
		t.Fatalf("count active listener refresh jobs: %v", err)
	}
	if active != 1 {
		t.Fatalf("expected one active listener refresh job, got %d", active)
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

	var active int64
	if err := db.WithContext(ctx).Model(&database.Job{}).
		Where("kind = ? AND status IN ?", JobKindListenerReconcile, []string{jobs.StatusQueued, jobs.StatusRunning}).
		Count(&active).Error; err != nil {
		t.Fatalf("count active listener reconcile jobs: %v", err)
	}
	if active != 1 {
		t.Fatalf("expected one active listener reconcile job, got %d", active)
	}
}

func TestAncestorPromotionStaysInsideLibraryRoot(t *testing.T) {
	t.Parallel()

	svc, db, record := newListenerTestService(t)
	svc.now = func() time.Time { return time.Date(2026, time.April, 24, 10, 5, 0, 0, time.UTC) }

	ctx := context.Background()
	firstPath := filepath.Join(record.RootPath, "Movies", "Action", "MovieA.2024.mkv")
	secondPath := filepath.Join(record.RootPath, "Movies", "Drama", "MovieB.2024.mkv")
	if _, err := svc.RecordStorageEvent(ctx, EventIngestInput{LibraryID: record.ID, Kind: "update", Path: firstPath}); err != nil {
		t.Fatalf("record first event: %v", err)
	}
	if _, err := svc.RecordStorageEvent(ctx, EventIngestInput{LibraryID: record.ID, Kind: "create", Path: secondPath}); err != nil {
		t.Fatalf("record second event: %v", err)
	}

	var jobs []database.Job
	if err := db.WithContext(ctx).Where("kind = ?", JobKindApplyStorageEventRefresh).Find(&jobs).Error; err != nil {
		t.Fatalf("list listener jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected one promoted listener job, got %d", len(jobs))
	}
	payload := mustDecodeRefreshPayload(t, jobs[0].PayloadJSON)
	if payload.RootPath != filepath.Join(record.RootPath, "Movies") {
		t.Fatalf("expected common ancestor %q, got %q", filepath.Join(record.RootPath, "Movies"), payload.RootPath)
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

	var queued []database.Job
	if err := db.WithContext(ctx).Where("kind = ?", JobKindListenerReconcile).Order("job_key asc").Find(&queued).Error; err != nil {
		t.Fatalf("list reconcile jobs: %v", err)
	}
	if len(queued) != 2 {
		t.Fatalf("expected exactly one future-dated reconcile job per library, got %d", len(queued))
	}
	for _, job := range queued {
		payload := mustDecodeReconcilePayload(t, job.PayloadJSON)
		if job.AvailableAt.Sub(fixedNow) != 6*time.Hour {
			t.Fatalf("expected available_at 6h in the future, got %s", job.AvailableAt.Sub(fixedNow))
		}
		if !job.AvailableAt.Equal(payload.ScheduledFor) {
			t.Fatalf("expected scheduled_for to match available_at")
		}
	}
}

func TestRecordStorageEventFallsBackToFullSyncWhenNormalizationIsUnsafe(t *testing.T) {
	t.Parallel()

	svc, _, record := newListenerTestService(t)
	ctx := context.Background()
	job, err := svc.RecordStorageEvent(ctx, EventIngestInput{LibraryID: record.ID, Kind: "rename", Path: filepath.Join(record.RootPath, "Movies", "MovieA.2024.mkv")})
	if err != nil {
		t.Fatalf("record rename event: %v", err)
	}
	if job.Kind != JobKindApplyStorageEventRefresh {
		t.Fatalf("expected listener job kind, got %q", job.Kind)
	}
	payload := mustDecodeRefreshPayload(t, job.PayloadJSON)
	if !payload.FallbackFullSync {
		t.Fatal("expected unsafe normalization to request full sync fallback")
	}
	if payload.RootPath != record.RootPath {
		t.Fatalf("expected fallback root %q, got %q", record.RootPath, payload.RootPath)
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

	var queued []database.Job
	if err := db.WithContext(ctx).Where("kind IN ?", []string{library.JobKindTargetedRefresh, library.JobKindSyncLibrary}).Order("id asc").Find(&queued).Error; err != nil {
		t.Fatalf("list queued work: %v", err)
	}
	if len(queued) != 1 || queued[0].Kind != library.JobKindTargetedRefresh {
		t.Fatalf("expected one targeted_refresh job, got %#v", queued)
	}
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

	var queued database.Job
	if err := db.WithContext(ctx).Where("kind = ?", library.JobKindSyncLibrary).Order("id desc").First(&queued).Error; err != nil {
		t.Fatalf("load sync job: %v", err)
	}
	if queued.Kind != library.JobKindSyncLibrary {
		t.Fatalf("expected sync_library job, got %q", queued.Kind)
	}
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
	reconcileJob.Status = jobs.StatusRunning
	if err := db.WithContext(ctx).Create(&reconcileJob).Error; err != nil {
		t.Fatalf("store reconcile job: %v", err)
	}

	if err := svc.RunReconcile(ctx, reconcileJob); err != nil {
		t.Fatalf("run reconcile: %v", err)
	}

	var syncJobs []database.Job
	if err := db.WithContext(ctx).Where("kind = ?", library.JobKindSyncLibrary).Find(&syncJobs).Error; err != nil {
		t.Fatalf("list sync jobs: %v", err)
	}
	if len(syncJobs) != 1 {
		t.Fatalf("expected one sync_library job from reconcile, got %d", len(syncJobs))
	}
	var reconcileJobs []database.Job
	if err := db.WithContext(ctx).Where("kind = ? AND status = ?", JobKindListenerReconcile, jobs.StatusQueued).Order("id asc").Find(&reconcileJobs).Error; err != nil {
		t.Fatalf("list reseeded reconcile jobs: %v", err)
	}
	if len(reconcileJobs) != 1 {
		t.Fatalf("expected one future queued reconcile job, got %d", len(reconcileJobs))
	}
	if reconcileJobs[0].AvailableAt.Sub(baseNow) != 6*time.Hour {
		t.Fatalf("expected next reconcile 6h later, got %s", reconcileJobs[0].AvailableAt.Sub(baseNow))
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
	return NewService(db, jobs.NewService(db), nil), db, record
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
	jobsSvc := jobs.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Local", RootPath: storageRoot})
	if err != nil {
		t.Fatalf("create media source: %v", err)
	}
	record, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: libraryRoot})
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
	return NewService(db, jobsSvc, librarySvc), db, record
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
