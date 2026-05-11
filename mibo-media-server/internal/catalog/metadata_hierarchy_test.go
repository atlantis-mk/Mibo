package catalog

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestListMetadataSeriesSeasonsReturnsHierarchy(t *testing.T) {
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
	season := database.MetadataItem{ItemType: database.MetadataItemTypeSeason, ContentForm: database.MetadataContentFormStandard, ParentID: &series.ID, RootID: &series.ID, IndexNumber: &seasonNumber, Title: "Season 1", SortTitle: "Season 1", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&season).Error; err != nil {
		t.Fatalf("create season: %v", err)
	}
	episodeNumber := 2
	episode := database.MetadataItem{ItemType: database.MetadataItemTypeEpisode, ContentForm: database.MetadataContentFormStandard, ParentID: &season.ID, RootID: &series.ID, ParentIndexNumber: &seasonNumber, IndexNumber: &episodeNumber, Title: "Episode 2", SortTitle: "Episode 2", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&episode).Error; err != nil {
		t.Fatalf("create episode: %v", err)
	}
	now := time.Now().UTC()
	projections := []database.LibraryMetadataProjection{
		{LibraryID: 7, MetadataItemID: season.ID, ItemType: season.ItemType, ParentID: season.ParentID, RootID: season.RootID, Title: season.Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, ChildCount: 1, AvailableCount: 1, LastProjectedAt: now},
		{LibraryID: 7, MetadataItemID: episode.ID, ItemType: episode.ItemType, ParentID: episode.ParentID, RootID: episode.RootID, Title: episode.Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, ResourceCount: 1, AvailableCount: 1, LastProjectedAt: now},
	}
	if err := db.WithContext(ctx).Create(&projections).Error; err != nil {
		t.Fatalf("create projections: %v", err)
	}

	seasons, err := svc.ListMetadataSeriesSeasons(ctx, series.ID, 7)
	if err != nil {
		t.Fatalf("list seasons: %v", err)
	}
	if len(seasons) != 1 || seasons[0].ID != season.ID || len(seasons[0].Episodes) != 1 || seasons[0].Episodes[0].ID != episode.ID {
		t.Fatalf("unexpected hierarchy: %#v", seasons)
	}
	if seasons[0].AvailabilityStatus != database.ProjectionAvailabilityAvailable || seasons[0].Episodes[0].AvailabilityStatus != database.ProjectionAvailabilityAvailable {
		t.Fatalf("unexpected projection state: %#v", seasons)
	}
}

func TestGetMetadataItemDetailIncludesSeriesHierarchy(t *testing.T) {
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
	season := database.MetadataItem{ItemType: database.MetadataItemTypeSeason, ContentForm: database.MetadataContentFormStandard, ParentID: &series.ID, RootID: &series.ID, IndexNumber: &seasonNumber, Title: "Season 1", SortTitle: "Season 1", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&season).Error; err != nil {
		t.Fatalf("create season: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.LibraryMetadataProjection{LibraryID: 7, MetadataItemID: series.ID, ItemType: series.ItemType, Title: series.Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, ChildCount: 1, LastProjectedAt: time.Now().UTC()}).Error; err != nil {
		t.Fatalf("create projection: %v", err)
	}

	detail, err := svc.GetMetadataItemDetail(ctx, series.ID, 7)
	if err != nil {
		t.Fatalf("get detail: %v", err)
	}
	if len(detail.Seasons) != 1 || detail.Seasons[0].ID != season.ID {
		t.Fatalf("expected metadata seasons in detail, got %#v", detail.Seasons)
	}
}
