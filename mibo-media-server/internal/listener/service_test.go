package listener

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
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
		payload := decodeReconcilePayload(t, job.PayloadJSON)
		if job.AvailableAt.Sub(fixedNow) != 6*time.Hour {
			t.Fatalf("expected available_at 6h in the future, got %s", job.AvailableAt.Sub(fixedNow))
		}
		if !job.AvailableAt.Equal(payload.ScheduledFor) {
			t.Fatalf("expected scheduled_for to match available_at")
		}
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

func mustDecodeRefreshPayload(t *testing.T, raw string) storageEventRefreshPayload {
	t.Helper()
	var payload storageEventRefreshPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode refresh payload: %v", err)
	}
	return payload
}

func decodeReconcilePayload(t *testing.T, raw string) reconcilePayload {
	t.Helper()
	var payload reconcilePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode reconcile payload: %v", err)
	}
	return payload
}
