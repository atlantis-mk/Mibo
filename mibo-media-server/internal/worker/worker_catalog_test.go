package worker

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/metadata"
	"github.com/atlan/mibo-media-server/internal/probe"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
	"gorm.io/gorm"
)

func TestRunOnceProcessesCatalogItemProjectionJob(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, jobsSvc, runner := newCatalogProjectionRunner(t)
	item := seedCatalogItem(t, ctx, db, seedCatalogItemInput{LibraryID: 7, Type: catalog.ItemTypeMovie, Path: "/library/movie-a", Title: "Movie A", AvailabilityStatus: catalog.AvailabilityAvailable})

	job, err := jobsSvc.Enqueue(ctx, library.JobKindCatalogRefreshItemProjection, catalog.ItemProjectionRefreshPayload{ItemID: item.ID})
	if err != nil {
		t.Fatalf("enqueue item projection job: %v", err)
	}

	runner.RunOnce(ctx)

	assertJobCompleted(t, ctx, db, job.ID)
	assertCatalogSearchDocExists(t, ctx, db, item.ID, item.LibraryID)
	assertCatalogProjectionCounts(t, ctx, db, 1, 1)
}

func TestRunOnceProcessesCatalogLibraryProjectionJobOnEmptyCatalog(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, jobsSvc, runner := newCatalogProjectionRunner(t)

	job, err := jobsSvc.Enqueue(ctx, library.JobKindCatalogRefreshLibraryProjection, catalog.LibraryProjectionRefreshPayload{LibraryID: 22, RootPath: "/library"})
	if err != nil {
		t.Fatalf("enqueue library projection job: %v", err)
	}

	runner.RunOnce(ctx)

	assertJobCompleted(t, ctx, db, job.ID)
	assertCatalogProjectionCounts(t, ctx, db, 0, 0)
}

func TestRunSyncLibraryQueuesCatalogLibraryProjectionRefresh(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, jobsSvc, librarySvc, libraryRecord := newScanProjectionFixture(t)
	payload, err := json.Marshal(map[string]any{"library_id": libraryRecord.ID, "root_path": libraryRecord.RootPath})
	if err != nil {
		t.Fatalf("marshal sync payload: %v", err)
	}

	if err := librarySvc.RunSyncLibrary(ctx, database.Job{PayloadJSON: string(payload)}); err != nil {
		t.Fatalf("run sync library: %v", err)
	}

	assertQueuedJobKind(t, ctx, jobsSvc, library.JobKindReindexLibrarySearch)
	assertQueuedJobKind(t, ctx, jobsSvc, library.JobKindCatalogRefreshLibraryProjection)
}

func TestRunTargetedRefreshQueuesCatalogLibraryProjectionRefresh(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mediaRoot := filepath.Join(t.TempDir(), "media-root")
	showDir := filepath.Join(mediaRoot, "Shows", "Show A")
	if err := os.MkdirAll(showDir, 0o755); err != nil {
		t.Fatalf("create show dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(showDir, "Show A S01E01.mkv"), []byte("episode"), 0o644); err != nil {
		t.Fatalf("write media file: %v", err)
	}

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	jobsSvc := jobs.NewService(db)
	cfg := config.Config{Local: config.LocalStorageConfig{RootPath: mediaRoot}}
	registry := providers.NewRegistry(cfg)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)

	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Local", RootPath: mediaRoot})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	libraryRecord, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Shows", Type: "shows", MediaSourceID: source.ID, RootPath: filepath.Join(mediaRoot, "Shows")})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	if err := db.WithContext(ctx).Where("kind IN ?", []string{library.JobKindSyncLibrary, library.JobKindReindexLibrarySearch, library.JobKindCatalogRefreshLibraryProjection, library.JobKindMatchMediaItem, "probe_media_file"}).Delete(&database.Job{}).Error; err != nil {
		t.Fatalf("clear bootstrap jobs: %v", err)
	}

	job, err := librarySvc.QueueTargetedRefresh(ctx, libraryRecord.ID, showDir, "storage_event")
	if err != nil {
		t.Fatalf("queue targeted refresh: %v", err)
	}
	if err := librarySvc.RunTargetedRefresh(ctx, job); err != nil {
		t.Fatalf("run targeted refresh: %v", err)
	}

	assertQueuedJobKind(t, ctx, jobsSvc, library.JobKindReindexLibrarySearch)
	assertQueuedJobKind(t, ctx, jobsSvc, library.JobKindCatalogRefreshLibraryProjection)
}

func TestRunOnceProcessesCatalogLibraryProjectionJobRebuildsSeededScope(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, jobsSvc, runner := newCatalogProjectionRunner(t)
	series := seedCatalogItem(t, ctx, db, seedCatalogItemInput{LibraryID: 13, Type: catalog.ItemTypeSeries, Path: "/library/Show A", Title: "Show A", AvailabilityStatus: catalog.AvailabilityAvailable})
	season := seedCatalogItem(t, ctx, db, seedCatalogItemInput{LibraryID: 13, Type: catalog.ItemTypeSeason, Path: "/library/Show A/Season 1", ParentID: &series.ID, RootID: &series.ID, Title: "Season 1", AvailabilityStatus: catalog.AvailabilityAvailable})
	episodeNumber := 1
	airDate := time.Date(2024, time.January, 10, 0, 0, 0, 0, time.UTC)
	seedCatalogItem(t, ctx, db, seedCatalogItemInput{LibraryID: 13, Type: catalog.ItemTypeEpisode, Path: "/library/Show A/Season 1/Show A S01E01.mkv", ParentID: &season.ID, RootID: &series.ID, Title: "Episode 1", AvailabilityStatus: catalog.AvailabilityAvailable, FirstAirDate: &airDate})
	seedCatalogItem(t, ctx, db, seedCatalogItemInput{LibraryID: 13, Type: catalog.ItemTypeEpisode, Path: "/library/Show A/Season 1/Show A S01E02.mkv", ParentID: &season.ID, RootID: &series.ID, Title: "Episode 2", AvailabilityStatus: catalog.AvailabilityMissing})

	if err := db.WithContext(ctx).Create(&database.ItemRollup{ItemID: series.ID, ChildCount: 99, AvailableCount: 99, MissingCount: 99, UpdatedAt: time.Now().Add(-time.Hour)}).Error; err != nil {
		t.Fatalf("seed stale rollup: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.CatalogSearchDocument{ItemID: series.ID, LibraryID: 13, ItemType: catalog.ItemTypeSeries, Title: "stale", AvailabilityStatus: catalog.AvailabilityMissing, UpdatedAt: time.Now().Add(-time.Hour)}).Error; err != nil {
		t.Fatalf("seed stale search document: %v", err)
	}

	job, err := jobsSvc.Enqueue(ctx, library.JobKindCatalogRefreshLibraryProjection, catalog.LibraryProjectionRefreshPayload{LibraryID: 13, RootPath: "/library/Show A"})
	if err != nil {
		t.Fatalf("enqueue library projection job: %v", err)
	}

	runner.RunOnce(ctx)

	assertJobCompleted(t, ctx, db, job.ID)
	assertCatalogProjectionCounts(t, ctx, db, 4, 4)
	assertCatalogSearchDocExists(t, ctx, db, series.ID, 13)

	var rollup database.ItemRollup
	if err := db.WithContext(ctx).First(&rollup, "item_id = ?", series.ID).Error; err != nil {
		t.Fatalf("load series rollup: %v", err)
	}
	if rollup.ChildCount != 3 || rollup.AvailableCount != 1 || rollup.MissingCount != 1 || rollup.LatestAirDate == nil || !rollup.LatestAirDate.Equal(airDate) {
		t.Fatalf("unexpected rebuilt rollup: %#v", rollup)
	}

	_ = episodeNumber
}

func newCatalogProjectionRunner(t *testing.T) (*gorm.DB, *jobs.Service, *Runner) {
	t.Helper()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	jobsSvc := jobs.NewService(db)
	runner := NewRunner(config.WorkerConfig{}, jobsSvc, nil, nil, nil, nil, catalog.NewService(db))
	return db, jobsSvc, runner
}

type seedCatalogItemInput struct {
	LibraryID          uint
	Type               string
	Path               string
	ParentID           *uint
	RootID             *uint
	Title              string
	AvailabilityStatus string
	FirstAirDate       *time.Time
	ReleaseDate        *time.Time
}

func seedCatalogItem(t *testing.T, ctx context.Context, db *gorm.DB, input seedCatalogItemInput) database.CatalogItem {
	t.Helper()

	item := database.CatalogItem{
		LibraryID:          input.LibraryID,
		Type:               input.Type,
		ParentID:           input.ParentID,
		RootID:             input.RootID,
		Path:               input.Path,
		SortKey:            input.Title,
		DisplayOrder:       catalog.DisplayOrderAired,
		Title:              input.Title,
		AvailabilityStatus: input.AvailabilityStatus,
		GovernanceStatus:   catalog.GovernancePending,
		FirstAirDate:       input.FirstAirDate,
		ReleaseDate:        input.ReleaseDate,
	}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	if item.RootID == nil {
		item.RootID = &item.ID
		if err := db.WithContext(ctx).Model(&database.CatalogItem{}).Where("id = ?", item.ID).Update("root_id", item.ID).Error; err != nil {
			t.Fatalf("set root id: %v", err)
		}
	}
	return item
}

func assertJobCompleted(t *testing.T, ctx context.Context, db *gorm.DB, jobID uint) {
	t.Helper()

	var job database.Job
	if err := db.WithContext(ctx).First(&job, jobID).Error; err != nil {
		t.Fatalf("load job %d: %v", jobID, err)
	}
	if job.Status != jobs.StatusCompleted {
		t.Fatalf("expected job %d completed, got %q", jobID, job.Status)
	}
}

func assertCatalogSearchDocExists(t *testing.T, ctx context.Context, db *gorm.DB, itemID uint, libraryID uint) {
	t.Helper()

	var doc database.CatalogSearchDocument
	if err := db.WithContext(ctx).First(&doc, "item_id = ?", itemID).Error; err != nil {
		t.Fatalf("load catalog search doc: %v", err)
	}
	if doc.ItemID != itemID || doc.LibraryID != libraryID {
		t.Fatalf("unexpected catalog search doc: %#v", doc)
	}
}

func assertCatalogProjectionCounts(t *testing.T, ctx context.Context, db *gorm.DB, wantRollups, wantDocs int64) {
	t.Helper()

	var rollupCount int64
	if err := db.WithContext(ctx).Model(&database.ItemRollup{}).Count(&rollupCount).Error; err != nil {
		t.Fatalf("count rollups: %v", err)
	}
	if rollupCount != wantRollups {
		t.Fatalf("expected %d rollups, got %d", wantRollups, rollupCount)
	}

	var docCount int64
	if err := db.WithContext(ctx).Model(&database.CatalogSearchDocument{}).Count(&docCount).Error; err != nil {
		t.Fatalf("count catalog search docs: %v", err)
	}
	if docCount != wantDocs {
		t.Fatalf("expected %d catalog search docs, got %d", wantDocs, docCount)
	}
}

func newScanProjectionFixture(t *testing.T) (*gorm.DB, *jobs.Service, *library.Service, database.Library) {
	t.Helper()

	ctx := context.Background()
	mediaRoot := filepath.Join(t.TempDir(), "media-root")
	if err := os.MkdirAll(mediaRoot, 0o755); err != nil {
		t.Fatalf("create media root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mediaRoot, "Movie A.2024.mkv"), []byte("movie"), 0o644); err != nil {
		t.Fatalf("write media file: %v", err)
	}

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, config.MetadataConfig{})
	searchSvc := search.NewService(db)
	ffprobePath := writeFakeFFprobe(t)
	cfg := config.Config{Local: config.LocalStorageConfig{RootPath: mediaRoot}, FFprobe: config.FFprobeConfig{Enabled: true, Path: ffprobePath, Timeout: time.Second}}
	registry := providers.NewRegistry(cfg)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	_ = metadata.NewService(db, cfg.Metadata, settingsSvc, searchSvc)
	_ = probe.NewService(db, registry, cfg.FFprobe)

	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Local", RootPath: mediaRoot})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	libraryRecord, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: mediaRoot})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	if err := db.WithContext(ctx).Where("kind IN ?", []string{library.JobKindSyncLibrary, library.JobKindReindexLibrarySearch, library.JobKindCatalogRefreshLibraryProjection, library.JobKindMatchMediaItem, "probe_media_file"}).Delete(&database.Job{}).Error; err != nil {
		t.Fatalf("clear bootstrap jobs: %v", err)
	}

	return db, jobsSvc, librarySvc, libraryRecord
}

func assertQueuedJobKind(t *testing.T, ctx context.Context, jobsSvc *jobs.Service, kind string) {
	t.Helper()

	queued, err := jobsSvc.List(ctx, 50, jobs.StatusQueued, kind)
	if err != nil {
		t.Fatalf("list pending jobs: %v", err)
	}
	for _, job := range queued {
		if job.Kind == kind {
			return
		}
	}
	t.Fatalf("expected queued job kind %q, got %#v", kind, queued)
}
