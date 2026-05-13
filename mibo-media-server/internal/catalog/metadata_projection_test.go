package catalog

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestRebuildLibraryMetadataProjectionFromResourceLinks(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	year := 2026
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie", SortTitle: "Movie", Year: &year, GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	resource := database.Resource{StableResourceKey: "resource:1", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, Status: "available"}
	if err := db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	seen := time.Now().UTC()
	if err := db.WithContext(ctx).Create(&database.ResourceLibraryLink{ResourceID: resource.ID, LibraryID: 7, Status: "available", FirstSeenAt: seen, LastSeenAt: seen}).Error; err != nil {
		t.Fatalf("create library link: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary}).Error; err != nil {
		t.Fatalf("create metadata link: %v", err)
	}

	projection, err := svc.RebuildLibraryMetadataProjection(ctx, 7, item.ID)
	if err != nil {
		t.Fatalf("rebuild projection: %v", err)
	}
	if projection.LibraryID != 7 || projection.MetadataItemID != item.ID || projection.ResourceCount != 1 || projection.AvailableCount != 1 || projection.AvailabilityStatus != database.ProjectionAvailabilityAvailable {
		t.Fatalf("unexpected projection: %#v", projection)
	}
}

func TestRebuildLibraryMetadataProjectionRollsUpChildren(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	series := database.MetadataItem{ItemType: database.MetadataItemTypeSeries, ContentForm: database.MetadataContentFormStandard, Title: "Show", SortTitle: "Show", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&series).Error; err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season := database.MetadataItem{ItemType: database.MetadataItemTypeSeason, ContentForm: database.MetadataContentFormStandard, ParentID: &series.ID, RootID: &series.ID, Title: "Season 1", SortTitle: "Show S01", IndexNumber: &seasonNumber, GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&season).Error; err != nil {
		t.Fatalf("create season: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.LibraryMetadataProjection{LibraryID: 7, MetadataItemID: season.ID, ItemType: season.ItemType, ParentID: season.ParentID, RootID: season.RootID, Title: season.Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, ResourceCount: 2, AvailableCount: 2, LastProjectedAt: time.Now().UTC()}).Error; err != nil {
		t.Fatalf("create child projection: %v", err)
	}

	projection, err := svc.RebuildLibraryMetadataProjection(ctx, 7, series.ID)
	if err != nil {
		t.Fatalf("rebuild series projection: %v", err)
	}
	if projection.ChildCount != 1 || projection.ResourceCount != 2 || projection.AvailableCount != 2 || projection.AvailabilityStatus != database.ProjectionAvailabilityAvailable {
		t.Fatalf("unexpected rollup projection: %#v", projection)
	}
}

func TestRebuildMetadataItemProjectionsRefreshesEpisodeAncestors(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	series := database.MetadataItem{ItemType: database.MetadataItemTypeSeries, ContentForm: database.MetadataContentFormStandard, Title: "Show", SortTitle: "Show", SortKey: "work:series:show", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&series).Error; err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season := database.MetadataItem{ItemType: database.MetadataItemTypeSeason, ContentForm: database.MetadataContentFormStandard, ParentID: &series.ID, RootID: &series.ID, Title: "Season 1", SortTitle: "Show S01", SortKey: "work:season:work:series:show:s01", IndexNumber: &seasonNumber, GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&season).Error; err != nil {
		t.Fatalf("create season: %v", err)
	}
	episodeNumber := 1
	episode := database.MetadataItem{ItemType: database.MetadataItemTypeEpisode, ContentForm: database.MetadataContentFormStandard, ParentID: &season.ID, RootID: &series.ID, ParentIndexNumber: &seasonNumber, IndexNumber: &episodeNumber, Title: "Episode 1", SortTitle: "Episode 1", SortKey: "episode:work:season:work:series:show:s01:e01", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&episode).Error; err != nil {
		t.Fatalf("create episode: %v", err)
	}
	resource := database.Resource{StableResourceKey: "resource:episode", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, Status: "available"}
	if err := db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	seen := time.Now().UTC()
	if err := db.WithContext(ctx).Create(&database.ResourceLibraryLink{ResourceID: resource.ID, LibraryID: 7, Status: "available", FirstSeenAt: seen, LastSeenAt: seen}).Error; err != nil {
		t.Fatalf("create library link: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: episode.ID, Role: database.ResourceLinkRolePrimary}).Error; err != nil {
		t.Fatalf("create metadata link: %v", err)
	}

	if err := svc.RebuildMetadataItemProjections(ctx, episode.ID); err != nil {
		t.Fatalf("rebuild metadata item projections: %v", err)
	}

	var projections []database.LibraryMetadataProjection
	if err := db.WithContext(ctx).Where("library_id = ?", 7).Order("metadata_item_id asc").Find(&projections).Error; err != nil {
		t.Fatalf("load projections: %v", err)
	}
	if len(projections) != 3 {
		t.Fatalf("expected episode plus series/season ancestor projections, got %#v", projections)
	}
}

func TestRebuildLibraryMetadataProjectionsIncludesEpisodeAncestors(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	series := database.MetadataItem{ItemType: database.MetadataItemTypeSeries, ContentForm: database.MetadataContentFormStandard, Title: "Show", SortTitle: "Show", SortKey: "work:series:show", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&series).Error; err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season := database.MetadataItem{ItemType: database.MetadataItemTypeSeason, ContentForm: database.MetadataContentFormStandard, ParentID: &series.ID, RootID: &series.ID, Title: "Season 1", SortTitle: "Show S01", SortKey: "work:season:work:series:show:s01", IndexNumber: &seasonNumber, GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&season).Error; err != nil {
		t.Fatalf("create season: %v", err)
	}
	episodeNumber := 1
	episode := database.MetadataItem{ItemType: database.MetadataItemTypeEpisode, ContentForm: database.MetadataContentFormStandard, ParentID: &season.ID, RootID: &series.ID, ParentIndexNumber: &seasonNumber, IndexNumber: &episodeNumber, Title: "Episode 1", SortTitle: "Episode 1", SortKey: "episode:work:season:work:series:show:s01:e01", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&episode).Error; err != nil {
		t.Fatalf("create episode: %v", err)
	}
	resource := database.Resource{StableResourceKey: "resource:episode", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, Status: "available"}
	if err := db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	seen := time.Now().UTC()
	if err := db.WithContext(ctx).Create(&database.ResourceLibraryLink{ResourceID: resource.ID, LibraryID: 7, Status: "available", FirstSeenAt: seen, LastSeenAt: seen}).Error; err != nil {
		t.Fatalf("create library link: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: episode.ID, Role: database.ResourceLinkRolePrimary}).Error; err != nil {
		t.Fatalf("create metadata link: %v", err)
	}

	if err := svc.RebuildLibraryMetadataProjections(ctx, 7); err != nil {
		t.Fatalf("rebuild library projections: %v", err)
	}

	var projections []database.LibraryMetadataProjection
	if err := db.WithContext(ctx).Where("library_id = ?", 7).Order("item_type asc").Find(&projections).Error; err != nil {
		t.Fatalf("load projections: %v", err)
	}
	if len(projections) != 3 {
		t.Fatalf("expected episode plus season and series projections, got %#v", projections)
	}
	var seriesProjection database.LibraryMetadataProjection
	if err := db.WithContext(ctx).Where("library_id = ? AND metadata_item_id = ?", 7, series.ID).First(&seriesProjection).Error; err != nil {
		t.Fatalf("load series projection: %v", err)
	}
	if seriesProjection.AvailabilityStatus != database.ProjectionAvailabilityAvailable || seriesProjection.AvailableCount != 1 {
		t.Fatalf("expected available series projection from episode rollup, got %#v", seriesProjection)
	}
}

func TestRebuildResourceMetadataProjectionsRefreshesAffectedPairs(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie", SortTitle: "Movie", GovernanceStatus: database.ReviewStateAccepted}
	resource := database.Resource{StableResourceKey: "resource:trigger", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, Status: "available"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	seen := time.Now().UTC()
	if err := db.WithContext(ctx).Create(&database.ResourceLibraryLink{ResourceID: resource.ID, LibraryID: 7, Status: "available", FirstSeenAt: seen, LastSeenAt: seen}).Error; err != nil {
		t.Fatalf("create library link: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary}).Error; err != nil {
		t.Fatalf("create metadata link: %v", err)
	}

	if err := svc.RebuildResourceMetadataProjections(ctx, resource.ID); err != nil {
		t.Fatalf("rebuild by resource: %v", err)
	}
	var projection database.LibraryMetadataProjection
	if err := db.WithContext(ctx).Where("library_id = ? AND metadata_item_id = ?", 7, item.ID).First(&projection).Error; err != nil {
		t.Fatalf("load projection: %v", err)
	}
	if projection.AvailabilityStatus != database.ProjectionAvailabilityAvailable {
		t.Fatalf("unexpected projection: %#v", projection)
	}
}

func TestRebuildLibraryMetadataProjectionSeparatesSameMetadataAcrossLibraries(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Shared", SortTitle: "Shared", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	resource := database.Resource{StableResourceKey: "resource:shared", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, Status: "available"}
	if err := db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	seen := time.Now().UTC()
	for _, libraryID := range []uint{7, 8} {
		if err := db.WithContext(ctx).Create(&database.ResourceLibraryLink{ResourceID: resource.ID, LibraryID: libraryID, Status: "available", FirstSeenAt: seen, LastSeenAt: seen}).Error; err != nil {
			t.Fatalf("create library link: %v", err)
		}
	}
	if err := db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary}).Error; err != nil {
		t.Fatalf("create metadata link: %v", err)
	}
	for _, libraryID := range []uint{7, 8} {
		if _, err := svc.RebuildLibraryMetadataProjection(ctx, libraryID, item.ID); err != nil {
			t.Fatalf("rebuild projection for library %d: %v", libraryID, err)
		}
	}
	var count int64
	if err := db.WithContext(ctx).Model(&database.LibraryMetadataProjection{}).Where("metadata_item_id = ?", item.ID).Count(&count).Error; err != nil {
		t.Fatalf("count projections: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected two library projections for one metadata item, got %d", count)
	}
}

func TestRebuildLibraryMetadataProjectionMissingTransition(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Missing", SortTitle: "Missing", GovernanceStatus: database.ReviewStateAccepted}
	resource := database.Resource{StableResourceKey: "resource:missing", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, Status: "missing"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	seen := time.Now().UTC()
	if err := db.WithContext(ctx).Create(&database.ResourceLibraryLink{ResourceID: resource.ID, LibraryID: 7, Status: "missing", FirstSeenAt: seen, LastSeenAt: seen}).Error; err != nil {
		t.Fatalf("create library link: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary}).Error; err != nil {
		t.Fatalf("create metadata link: %v", err)
	}
	projection, err := svc.RebuildLibraryMetadataProjection(ctx, 7, item.ID)
	if err != nil {
		t.Fatalf("rebuild missing projection: %v", err)
	}
	if projection.AvailabilityStatus != database.ProjectionAvailabilityMissing || projection.MissingCount != 1 {
		t.Fatalf("unexpected missing projection: %#v", projection)
	}
}
