package worker

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/library"
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
