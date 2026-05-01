package jobs

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func TestClaimNextPrioritizesSyncBeforeOlderEnrichment(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openTestDB(t)
	svc := NewService(db)
	baseTime := time.Now().UTC().Add(-time.Hour)

	seedJob(t, ctx, db, database.Job{Kind: "probe_inventory_file", Status: StatusQueued, PayloadJSON: `{"inventory_file_id":1}`, AvailableAt: baseTime})
	seedJob(t, ctx, db, database.Job{Kind: "match_catalog_item", Status: StatusQueued, PayloadJSON: `{"item_id":1}`, AvailableAt: baseTime.Add(time.Second)})
	syncJob := seedJob(t, ctx, db, database.Job{Kind: "sync_library", Status: StatusQueued, PayloadJSON: `{"library_id":1}`, AvailableAt: baseTime.Add(time.Minute)})

	claimed, err := svc.ClaimNext(ctx)
	if err != nil {
		t.Fatalf("claim next: %v", err)
	}
	if claimed.ID != syncJob.ID {
		t.Fatalf("expected sync job %d to be claimed first, got %#v", syncJob.ID, claimed)
	}
}

func TestClaimNextKeepsFIFOWithinPriority(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openTestDB(t)
	svc := NewService(db)
	baseTime := time.Now().UTC().Add(-time.Hour)

	first := seedJob(t, ctx, db, database.Job{Kind: "targeted_refresh", Status: StatusQueued, PayloadJSON: `{"library_id":1}`, AvailableAt: baseTime})
	seedJob(t, ctx, db, database.Job{Kind: "sync_library", Status: StatusQueued, PayloadJSON: `{"library_id":2}`, AvailableAt: baseTime.Add(time.Second)})

	claimed, err := svc.ClaimNext(ctx)
	if err != nil {
		t.Fatalf("claim next: %v", err)
	}
	if claimed.ID != first.ID {
		t.Fatalf("expected earliest same-priority job %d, got %#v", first.ID, claimed)
	}
}

func TestCancelQueuedJobMarksCancelled(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openTestDB(t)
	svc := NewService(db)
	job := seedJob(t, ctx, db, database.Job{Kind: "sync_library", Status: StatusQueued, PayloadJSON: `{"library_id":1}`, AvailableAt: time.Now().UTC()})

	cancelled, err := svc.Cancel(ctx, job.ID)
	if err != nil {
		t.Fatalf("cancel job: %v", err)
	}
	if cancelled.Status != StatusCancelled {
		t.Fatalf("expected status %q, got %q", StatusCancelled, cancelled.Status)
	}
	if cancelled.FinishedAt == nil {
		t.Fatalf("expected finished_at to be set")
	}
}

func TestCancelRunningJobRequestsCancellation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openTestDB(t)
	svc := NewService(db)
	job := seedJob(t, ctx, db, database.Job{Kind: "sync_library", Status: StatusRunning, PayloadJSON: `{"library_id":1}`, AvailableAt: time.Now().UTC()})

	requested, err := svc.Cancel(ctx, job.ID)
	if err != nil {
		t.Fatalf("cancel running job: %v", err)
	}
	if requested.Status != StatusCancelRequested {
		t.Fatalf("expected status %q, got %q", StatusCancelRequested, requested.Status)
	}
	ok, err := svc.CancellationRequested(ctx, job.ID)
	if err != nil {
		t.Fatalf("check cancellation: %v", err)
	}
	if !ok {
		t.Fatalf("expected cancellation requested")
	}
	if err := svc.Cancelled(ctx, job.ID); err != nil {
		t.Fatalf("mark cancelled: %v", err)
	}
	var updated database.Job
	if err := db.WithContext(ctx).First(&updated, job.ID).Error; err != nil {
		t.Fatalf("reload job: %v", err)
	}
	if updated.Status != StatusCancelled {
		t.Fatalf("expected status %q, got %q", StatusCancelled, updated.Status)
	}
}

func TestCancelCompletedJobFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openTestDB(t)
	svc := NewService(db)
	job := seedJob(t, ctx, db, database.Job{Kind: "sync_library", Status: StatusCompleted, PayloadJSON: `{"library_id":1}`, AvailableAt: time.Now().UTC()})

	if _, err := svc.Cancel(ctx, job.ID); err == nil {
		t.Fatalf("expected completed job cancellation to fail")
	}
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	return db
}

func seedJob(t *testing.T, ctx context.Context, db *gorm.DB, job database.Job) database.Job {
	t.Helper()
	if err := db.WithContext(ctx).Create(&job).Error; err != nil {
		t.Fatalf("seed job: %v", err)
	}
	return job
}
