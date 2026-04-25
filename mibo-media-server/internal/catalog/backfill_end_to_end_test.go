package catalog_test

import (
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/settings"
)

func TestLegacyBackfillEndToEndIdempotent(t *testing.T) {
	t.Parallel()

	fx := newLegacyBackfillFixture(t)
	ctx := fx.ctx

	cleanupAt := time.Date(2026, time.April, 24, 12, 0, 0, 0, time.UTC)
	if _, err := fx.settings.UpdateCatalogMigrationState(ctx, settings.UpdateCatalogMigrationStateInput{
		CatalogReadEnabled:       true,
		LegacyCleanupCompletedAt: &cleanupAt,
	}); err != nil {
		t.Fatalf("seed catalog migration state: %v", err)
	}

	movieRuntimeSeconds := 7200
	movie := database.MediaItem{
		LibraryID:      fx.library.ID,
		Type:           "movie",
		Title:          "Movie A",
		SourcePath:     "/library/movies/movie-a.mkv",
		RuntimeSeconds: &movieRuntimeSeconds,
		MatchStatus:    "matched",
		Status:         "ready",
	}
	if err := fx.db.WithContext(ctx).Create(&movie).Error; err != nil {
		t.Fatalf("create movie: %v", err)
	}

	episodeRuntimeSeconds := 2700
	seasonNumber := 1
	episodeNumber := 1
	episode := database.MediaItem{
		LibraryID:      fx.library.ID,
		Type:           "episode",
		Title:          "Pilot",
		SeriesTitle:    "Show A",
		SourcePath:     "/library/shows/show-a/Season 01/pilot.mkv",
		RuntimeSeconds: &episodeRuntimeSeconds,
		SeasonNumber:   &seasonNumber,
		EpisodeNumber:  &episodeNumber,
		MatchStatus:    "matched",
		MetadataProvider: "tmdb",
		ExternalID:       "tv:show-a",
		Status:           "ready",
	}
	if err := fx.db.WithContext(ctx).Create(&episode).Error; err != nil {
		t.Fatalf("create episode: %v", err)
	}

	movieDuration := 7200.0
	episodeDuration := 2700.0
	movieFile := database.MediaFile{
		LibraryID:         fx.library.ID,
		MediaItemID:       &movie.ID,
		StoragePath:       movie.SourcePath,
		StableIdentityKey: "stable:movie-a",
		ProviderName:      "local",
		ProbeStatus:       "complete",
		DurationSeconds:   &movieDuration,
	}
	episodeFile := database.MediaFile{
		LibraryID:         fx.library.ID,
		MediaItemID:       &episode.ID,
		StoragePath:       episode.SourcePath,
		StableIdentityKey: "stable:show-a:s01e01",
		ProviderName:      "local",
		ProbeStatus:       "complete",
		DurationSeconds:   &episodeDuration,
	}
	for _, file := range []*database.MediaFile{&movieFile, &episodeFile} {
		if err := fx.db.WithContext(ctx).Create(file).Error; err != nil {
			t.Fatalf("create media file %q: %v", file.StoragePath, err)
		}
	}

	movieCompletedAt := time.Date(2026, time.April, 25, 10, 0, 0, 0, time.UTC)
	movieLastPlayedAt := movieCompletedAt.Add(-20 * time.Minute)
	episodeLastPlayedAt := movieCompletedAt.Add(-5 * time.Minute)
	progressRows := []database.PlaybackProgress{
		{
			UserID:          fx.user.ID,
			MediaItemID:     movie.ID,
			MediaFileID:     &movieFile.ID,
			PositionSeconds: 7200,
			DurationSeconds: &movieRuntimeSeconds,
			Watched:         true,
			CompletedAt:     &movieCompletedAt,
			LastPlayedAt:    &movieLastPlayedAt,
		},
		{
			UserID:          fx.user.ID,
			MediaItemID:     episode.ID,
			MediaFileID:     &episodeFile.ID,
			PositionSeconds: 1200,
			DurationSeconds: &episodeRuntimeSeconds,
			Watched:         false,
			LastPlayedAt:    &episodeLastPlayedAt,
		},
	}
	for _, row := range progressRows {
		progress := row
		if err := fx.db.WithContext(ctx).Create(&progress).Error; err != nil {
			t.Fatalf("create playback progress for media item %d: %v", progress.MediaItemID, err)
		}
	}

	firstRun := fx.enqueueLegacyBackfillRun(t, catalog.LegacyBackfillScope{Kind: catalog.LegacyBackfillScopeAll})
	fx.runner.RunOnce(ctx)

	firstReport, err := fx.catalog.GetLegacyBackfillRun(ctx, firstRun.ID)
	if err != nil {
		t.Fatalf("load first run report: %v", err)
	}
	if firstReport.Status != catalog.LegacyBackfillStatusCompleted || firstReport.FinishedAt == nil {
		t.Fatalf("expected completed first run report, got %#v", firstReport)
	}

	stateAfterFirstRun, err := fx.settings.GetCatalogMigrationState(ctx)
	if err != nil {
		t.Fatalf("load catalog migration state after first run: %v", err)
	}
	if stateAfterFirstRun.CatalogBackfillCompletedAt == nil {
		t.Fatal("expected catalog_backfill_completed_at to be set after successful run")
	}
	if !stateAfterFirstRun.CatalogReadEnabled {
		t.Fatal("expected catalog_read_enabled to remain true after successful run")
	}
	if stateAfterFirstRun.LegacyCleanupCompletedAt == nil || !stateAfterFirstRun.LegacyCleanupCompletedAt.Equal(cleanupAt) {
		t.Fatalf("expected legacy_cleanup_completed_at %s, got %#v", cleanupAt, stateAfterFirstRun.LegacyCleanupCompletedAt)
	}

	assertTableCount(t, ctx, fx.db, &database.CatalogItem{}, 4)
	assertTableCount(t, ctx, fx.db, &database.InventoryFile{}, 2)
	assertTableCount(t, ctx, fx.db, &database.MediaAsset{}, 2)
	assertTableCount(t, ctx, fx.db, &database.AssetFile{}, 2)
	assertTableCount(t, ctx, fx.db, &database.AssetItem{}, 2)
	assertTableCount(t, ctx, fx.db, &database.UserItemData{}, 2)

	secondRun := fx.enqueueLegacyBackfillRun(t, catalog.LegacyBackfillScope{Kind: catalog.LegacyBackfillScopeAll})
	fx.runner.RunOnce(ctx)

	secondReport, err := fx.catalog.GetLegacyBackfillRun(ctx, secondRun.ID)
	if err != nil {
		t.Fatalf("load second run report: %v", err)
	}
	if secondReport.Status != catalog.LegacyBackfillStatusCompleted || secondReport.FinishedAt == nil {
		t.Fatalf("expected completed second run report, got %#v", secondReport)
	}

	stateAfterSecondRun, err := fx.settings.GetCatalogMigrationState(ctx)
	if err != nil {
		t.Fatalf("load catalog migration state after second run: %v", err)
	}
	if stateAfterSecondRun.CatalogBackfillCompletedAt == nil {
		t.Fatal("expected catalog_backfill_completed_at to remain set after rerun")
	}
	if !stateAfterSecondRun.CatalogReadEnabled {
		t.Fatal("expected catalog_read_enabled to remain unchanged after rerun")
	}
	if stateAfterSecondRun.LegacyCleanupCompletedAt == nil || !stateAfterSecondRun.LegacyCleanupCompletedAt.Equal(cleanupAt) {
		t.Fatalf("expected legacy_cleanup_completed_at %s after rerun, got %#v", cleanupAt, stateAfterSecondRun.LegacyCleanupCompletedAt)
	}
	if !stateAfterSecondRun.CatalogBackfillCompletedAt.Equal(*stateAfterFirstRun.CatalogBackfillCompletedAt) && stateAfterSecondRun.CatalogBackfillCompletedAt.Before(*stateAfterFirstRun.CatalogBackfillCompletedAt) {
		t.Fatalf("expected rerun completion timestamp to stay monotonic, got first=%s second=%s", stateAfterFirstRun.CatalogBackfillCompletedAt, stateAfterSecondRun.CatalogBackfillCompletedAt)
	}

	assertTableCount(t, ctx, fx.db, &database.CatalogItem{}, 4)
	assertTableCount(t, ctx, fx.db, &database.InventoryFile{}, 2)
	assertTableCount(t, ctx, fx.db, &database.MediaAsset{}, 2)
	assertTableCount(t, ctx, fx.db, &database.AssetFile{}, 2)
	assertTableCount(t, ctx, fx.db, &database.AssetItem{}, 2)
	assertTableCount(t, ctx, fx.db, &database.UserItemData{}, 2)
}

func (fx *legacyBackfillFixture) enqueueLegacyBackfillRun(t *testing.T, scope catalog.LegacyBackfillScope) catalog.LegacyBackfillRun {
	t.Helper()

	run, err := fx.catalog.CreateLegacyBackfillRun(fx.ctx, catalog.CreateLegacyBackfillRunInput{
		Scope:             scope,
		TriggeredByUserID: fx.user.ID,
	})
	if err != nil {
		t.Fatalf("create legacy backfill run: %v", err)
	}

	payload := catalog.LegacyBackfillPayload{RunID: run.ID}
	if scope.Kind == catalog.LegacyBackfillScopeLibrary {
		payload.LibraryID = scope.LibraryID
	}
	if _, err := fx.jobs.Enqueue(fx.ctx, catalog.JobKindLegacyBackfill, payload); err != nil {
		t.Fatalf("enqueue legacy backfill job: %v", err)
	}

	queuedJobs, err := fx.jobs.List(fx.ctx, 10, jobs.StatusQueued, catalog.JobKindLegacyBackfill)
	if err != nil {
		t.Fatalf("list queued backfill jobs: %v", err)
	}
	if len(queuedJobs) == 0 {
		t.Fatal("expected queued legacy backfill job")
	}

	return run
}
