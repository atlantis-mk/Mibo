package worker

import (
	"context"
	"testing"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/jobs"
)

func TestRunOnceProcessesCatalogBackfillJob(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, jobsSvc, runner := newCatalogProjectionRunner(t)
	catalogSvc := catalog.NewService(db)
	libraryID := uint(42)

	run, err := catalogSvc.CreateLegacyBackfillRun(ctx, catalog.CreateLegacyBackfillRunInput{
		Scope:             catalog.LegacyBackfillScope{Kind: catalog.LegacyBackfillScopeLibrary, LibraryID: &libraryID},
		TriggeredByUserID: 7,
	})
	if err != nil {
		t.Fatalf("create legacy backfill run: %v", err)
	}

	job, err := jobsSvc.Enqueue(ctx, catalog.JobKindLegacyBackfill, catalog.LegacyBackfillPayload{
		RunID:     run.ID,
		LibraryID: &libraryID,
	})
	if err != nil {
		t.Fatalf("enqueue backfill job: %v", err)
	}

	runner.RunOnce(ctx)

	assertJobCompleted(t, ctx, db, job.ID)

	storedRun, err := catalogSvc.GetLegacyBackfillRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("load legacy backfill run: %v", err)
	}
	if storedRun.Status != catalog.LegacyBackfillStatusCompleted {
		t.Fatalf("expected run status %q, got %q", catalog.LegacyBackfillStatusCompleted, storedRun.Status)
	}
	if storedRun.StartedAt == nil || storedRun.FinishedAt == nil {
		t.Fatalf("expected run timestamps set, got started_at=%v finished_at=%v", storedRun.StartedAt, storedRun.FinishedAt)
	}
	if storedRun.SuccessCount != 0 || storedRun.SkippedCount != 0 || storedRun.ConflictCount != 0 || storedRun.OrphanFileCount != 0 || storedRun.DuplicateEpisodeCandidateCount != 0 {
		t.Fatalf("expected zero aggregate counts for empty backfill, got %#v", storedRun)
	}

	queuedJobs, err := jobsSvc.List(ctx, 10, jobs.StatusQueued, catalog.JobKindLegacyBackfill)
	if err != nil {
		t.Fatalf("list queued backfill jobs: %v", err)
	}
	if len(queuedJobs) != 0 {
		t.Fatalf("expected no queued backfill jobs after run, got %#v", queuedJobs)
	}
}
