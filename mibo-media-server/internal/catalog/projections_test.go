package catalog

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func TestCatalogRefreshItemProjectionNoRows(t *testing.T) {
	svc, ctx := newTestService(t)

	if err := svc.RefreshItemProjection(ctx, 999); err != nil {
		t.Fatalf("refresh item projection: %v", err)
	}

	assertProjectionCounts(t, ctx, svc.db, 0, 0)
}

func TestCatalogRefreshLibraryProjectionNoRows(t *testing.T) {
	svc, ctx := newTestService(t)

	if err := svc.RefreshLibraryProjection(ctx, 42, "/library"); err != nil {
		t.Fatalf("refresh library projection: %v", err)
	}

	assertProjectionCounts(t, ctx, svc.db, 0, 0)
}

func TestCatalogRefreshLibraryProjectionRebuildsTargetedRows(t *testing.T) {
	svc, ctx := newProjectionTestService(t)

	series := seedCatalogProjectionItem(t, ctx, svc.db, database.CatalogItem{
		LibraryID:          7,
		Type:               ItemTypeSeries,
		Path:               "/library/Show A",
		SortKey:            "Show A",
		DisplayOrder:       DisplayOrderAired,
		Title:              "Show A",
		AvailabilityStatus: "",
		GovernanceStatus:   GovernancePending,
	})
	seasonNumber := 1
	season := seedCatalogProjectionItem(t, ctx, svc.db, database.CatalogItem{
		LibraryID:          7,
		Type:               ItemTypeSeason,
		ParentID:           &series.ID,
		RootID:             &series.ID,
		Path:               "/library/Show A/Season 1",
		SortKey:            "Show A S01",
		DisplayOrder:       DisplayOrderAired,
		ParentIndexNumber:  &seasonNumber,
		IndexNumber:        &seasonNumber,
		Title:              "Season 1",
		AvailabilityStatus: AvailabilityAvailable,
		GovernanceStatus:   GovernancePending,
	})
	episodeOne := 1
	episodeTwo := 2
	airDate := time.Date(2024, time.January, 10, 0, 0, 0, 0, time.UTC)
	episodeOneItem := seedCatalogProjectionItem(t, ctx, svc.db, database.CatalogItem{
		LibraryID:          7,
		Type:               ItemTypeEpisode,
		ParentID:           &season.ID,
		RootID:             &series.ID,
		Path:               "/library/Show A/Season 1/Show A S01E01.mkv",
		SortKey:            "Show A S01E01",
		DisplayOrder:       DisplayOrderAired,
		ParentIndexNumber:  &seasonNumber,
		IndexNumber:        &episodeOne,
		Title:              "Episode 1",
		FirstAirDate:       &airDate,
		AvailabilityStatus: AvailabilityAvailable,
		GovernanceStatus:   GovernancePending,
	})
	episodeTwoItem := seedCatalogProjectionItem(t, ctx, svc.db, database.CatalogItem{
		LibraryID:          7,
		Type:               ItemTypeEpisode,
		ParentID:           &season.ID,
		RootID:             &series.ID,
		Path:               "/library/Show A/Season 1/Show A S01E02.mkv",
		SortKey:            "Show A S01E02",
		DisplayOrder:       DisplayOrderAired,
		ParentIndexNumber:  &seasonNumber,
		IndexNumber:        &episodeTwo,
		Title:              "Episode 2",
		AvailabilityStatus: AvailabilityMissing,
		GovernanceStatus:   GovernancePending,
	})

	if err := svc.db.WithContext(ctx).Create(&database.ItemRollup{ItemID: series.ID, ChildCount: 99, AvailableCount: 99, MissingCount: 99, UpdatedAt: time.Now().Add(-time.Hour)}).Error; err != nil {
		t.Fatalf("seed stale rollup: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.CatalogSearchDocument{ItemID: series.ID, LibraryID: 7, ItemType: ItemTypeSeries, Title: "stale", AvailabilityStatus: AvailabilityMissing, UpdatedAt: time.Now().Add(-time.Hour)}).Error; err != nil {
		t.Fatalf("seed stale search document: %v", err)
	}

	if err := svc.RefreshLibraryProjection(ctx, 7, "/library/Show A"); err != nil {
		t.Fatalf("refresh library projection: %v", err)
	}

	assertProjectionCounts(t, ctx, svc.db, 4, 4)
	assertRollup(t, ctx, svc.db, series.ID, 3, 1, 1)
	assertRollup(t, ctx, svc.db, season.ID, 2, 1, 1)
	assertCatalogSearchDocument(t, ctx, svc.db, series.ID, 7, ItemTypeSeries, "Show A", AvailabilityNoLocalMedia)
	assertCatalogSearchDocument(t, ctx, svc.db, season.ID, 7, ItemTypeSeason, "Season 1", AvailabilityAvailable)
	assertCatalogSearchDocument(t, ctx, svc.db, episodeOneItem.ID, 7, ItemTypeEpisode, "Episode 1", AvailabilityAvailable)
	assertCatalogSearchDocument(t, ctx, svc.db, episodeTwoItem.ID, 7, ItemTypeEpisode, "Episode 2", AvailabilityMissing)
}

func TestCatalogRefreshItemProjectionNormalizesBlankAvailability(t *testing.T) {
	svc, ctx := newProjectionTestService(t)

	item := seedCatalogProjectionItem(t, ctx, svc.db, database.CatalogItem{
		LibraryID:          9,
		Type:               ItemTypeMovie,
		Path:               "/library/Movie A.mkv",
		SortKey:            "Movie A",
		DisplayOrder:       DisplayOrderAired,
		Title:              "Movie A",
		AvailabilityStatus: "",
		GovernanceStatus:   GovernancePending,
	})

	if err := svc.db.WithContext(ctx).Create(&database.CatalogSearchDocument{ItemID: item.ID, LibraryID: 9, ItemType: ItemTypeMovie, Title: "stale", AvailabilityStatus: AvailabilityMissing, UpdatedAt: time.Now().Add(-time.Hour)}).Error; err != nil {
		t.Fatalf("seed stale search document: %v", err)
	}

	if err := svc.RefreshItemProjection(ctx, item.ID); err != nil {
		t.Fatalf("refresh item projection: %v", err)
	}

	assertProjectionCounts(t, ctx, svc.db, 1, 1)
	assertRollup(t, ctx, svc.db, item.ID, 0, 0, 0)
	assertCatalogSearchDocument(t, ctx, svc.db, item.ID, 9, ItemTypeMovie, "Movie A", AvailabilityNoLocalMedia)
}

func TestCatalogRefreshLibraryProjectionBatchesLargeScopes(t *testing.T) {
	svc, ctx := newProjectionTestService(t)

	for i := 0; i < projectionSQLBatchSize+25; i++ {
		seedCatalogProjectionItem(t, ctx, svc.db, database.CatalogItem{
			LibraryID:          11,
			Type:               ItemTypeMovie,
			Path:               "/library/Movie Batch " + time.Unix(int64(i), 0).UTC().Format("150405"),
			SortKey:            "Movie Batch",
			DisplayOrder:       DisplayOrderAired,
			Title:              "Movie Batch",
			AvailabilityStatus: AvailabilityAvailable,
			GovernanceStatus:   GovernancePending,
		})
	}

	if err := svc.RefreshLibraryProjection(ctx, 11, ""); err != nil {
		t.Fatalf("refresh large library projection: %v", err)
	}

	assertProjectionCounts(t, ctx, svc.db, projectionSQLBatchSize+25, projectionSQLBatchSize+25)
}

func newProjectionTestService(t *testing.T) (*Service, context.Context) {
	t.Helper()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	return NewService(db), context.Background()
}

func seedCatalogProjectionItem(t *testing.T, ctx context.Context, db *gorm.DB, item database.CatalogItem) database.CatalogItem {
	t.Helper()

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

func assertProjectionCounts(t *testing.T, ctx context.Context, db *gorm.DB, wantRollups, wantDocs int64) {
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
		t.Fatalf("count search docs: %v", err)
	}
	if docCount != wantDocs {
		t.Fatalf("expected %d catalog search docs, got %d", wantDocs, docCount)
	}
}

func assertRollup(t *testing.T, ctx context.Context, db *gorm.DB, itemID uint, wantChildren, wantAvailable, wantMissing int) {
	t.Helper()

	var rollup database.ItemRollup
	if err := db.WithContext(ctx).First(&rollup, "item_id = ?", itemID).Error; err != nil {
		t.Fatalf("load rollup for item %d: %v", itemID, err)
	}
	if rollup.ChildCount != wantChildren || rollup.AvailableCount != wantAvailable || rollup.MissingCount != wantMissing {
		t.Fatalf("unexpected rollup for item %d: %#v", itemID, rollup)
	}
}

func assertCatalogSearchDocument(t *testing.T, ctx context.Context, db *gorm.DB, itemID uint, libraryID uint, itemType string, title string, availabilityStatus string) {
	t.Helper()

	var doc database.CatalogSearchDocument
	if err := db.WithContext(ctx).First(&doc, "item_id = ?", itemID).Error; err != nil {
		t.Fatalf("load search document for item %d: %v", itemID, err)
	}
	if doc.LibraryID != libraryID || doc.ItemType != itemType || doc.Title != title || doc.AvailabilityStatus != availabilityStatus {
		t.Fatalf("unexpected catalog search document: %#v", doc)
	}
}
