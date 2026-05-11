package catalog

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestListHomeContentSectionsGroupsMoviesAndSeries(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	now := time.Now().UTC()
	libraryRecord := database.Library{Name: "Mixed", Type: "mixed", RootPath: "/media", Status: "active"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	items := []database.MetadataItem{
		{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Newest Movie", SortTitle: "Newest Movie", GovernanceStatus: database.ReviewStateAccepted},
		{ItemType: database.MetadataItemTypeSeries, ContentForm: database.MetadataContentFormStandard, Title: "Newest Series", SortTitle: "Newest Series", GovernanceStatus: database.ReviewStateAccepted},
		{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Hidden Movie", SortTitle: "Hidden Movie", GovernanceStatus: database.ReviewStateAccepted},
		{ItemType: database.MetadataItemTypeSeries, ContentForm: database.MetadataContentFormStandard, Title: "Unavailable Series", SortTitle: "Unavailable Series", GovernanceStatus: database.ReviewStateAccepted},
	}
	if err := db.WithContext(ctx).Create(&items).Error; err != nil {
		t.Fatalf("create metadata items: %v", err)
	}
	projections := []database.LibraryMetadataProjection{
		{LibraryID: libraryRecord.ID, MetadataItemID: items[0].ID, ItemType: items[0].ItemType, Title: items[0].Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, LatestAddedAt: timePtrForHomeProjection(now), LastProjectedAt: now},
		{LibraryID: libraryRecord.ID, MetadataItemID: items[1].ID, ItemType: items[1].ItemType, Title: items[1].Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, LatestAddedAt: timePtrForHomeProjection(now.Add(-time.Minute)), LastProjectedAt: now},
		{LibraryID: libraryRecord.ID, MetadataItemID: items[2].ID, ItemType: items[2].ItemType, Title: items[2].Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, Hidden: true, LatestAddedAt: timePtrForHomeProjection(now.Add(time.Minute)), LastProjectedAt: now},
		{LibraryID: libraryRecord.ID, MetadataItemID: items[3].ID, ItemType: items[3].ItemType, Title: items[3].Title, AvailabilityStatus: database.ProjectionAvailabilityUnavailable, LatestAddedAt: timePtrForHomeProjection(now.Add(time.Minute)), LastProjectedAt: now},
	}
	if err := db.WithContext(ctx).Create(&projections).Error; err != nil {
		t.Fatalf("create projections: %v", err)
	}

	sections, err := svc.ListHomeContentSections(ctx, 12)
	if err != nil {
		t.Fatalf("list home sections: %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("expected movie and series sections, got %#v", sections)
	}
	if sections[0].Key != "movies" || sections[0].Title != "电影" || len(sections[0].Items) != 1 || sections[0].Items[0].MetadataItemID != items[0].ID {
		t.Fatalf("unexpected movie section: %#v", sections[0])
	}
	if sections[1].Key != "series" || sections[1].Title != "剧集" || len(sections[1].Items) != 1 || sections[1].Items[0].MetadataItemID != items[1].ID {
		t.Fatalf("unexpected series section: %#v", sections[1])
	}
}

func TestListHomeContentSectionsOmitsEmptySections(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sections, err := NewService(db).ListHomeContentSections(context.Background(), 12)
	if err != nil {
		t.Fatalf("list home sections: %v", err)
	}
	if len(sections) != 0 {
		t.Fatalf("expected no sections, got %#v", sections)
	}
}

func TestListHomeMediaOverviewReturnsCountsAndPreviewItems(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	now := time.Now().UTC()
	libraryRecord := database.Library{Name: "Mixed", Type: "mixed", RootPath: "/media", Status: "active"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	items := []database.MetadataItem{
		{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie One", SortTitle: "Movie One", GovernanceStatus: database.ReviewStateAccepted},
		{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie Two", SortTitle: "Movie Two", GovernanceStatus: database.ReviewStateAccepted},
		{ItemType: database.MetadataItemTypeSeries, ContentForm: database.MetadataContentFormStandard, Title: "Series One", SortTitle: "Series One", GovernanceStatus: database.ReviewStateAccepted},
	}
	if err := db.WithContext(ctx).Create(&items).Error; err != nil {
		t.Fatalf("create metadata items: %v", err)
	}
	projections := []database.LibraryMetadataProjection{
		{LibraryID: libraryRecord.ID, MetadataItemID: items[0].ID, ItemType: items[0].ItemType, Title: items[0].Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, LatestAddedAt: timePtrForHomeProjection(now), LastProjectedAt: now},
		{LibraryID: libraryRecord.ID, MetadataItemID: items[1].ID, ItemType: items[1].ItemType, Title: items[1].Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, LatestAddedAt: timePtrForHomeProjection(now.Add(-time.Minute)), LastProjectedAt: now},
		{LibraryID: libraryRecord.ID, MetadataItemID: items[2].ID, ItemType: items[2].ItemType, Title: items[2].Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, LatestAddedAt: timePtrForHomeProjection(now.Add(-2 * time.Minute)), LastProjectedAt: now},
	}
	if err := db.WithContext(ctx).Create(&projections).Error; err != nil {
		t.Fatalf("create projections: %v", err)
	}

	overview, err := svc.ListHomeMediaOverview(ctx, 1)
	if err != nil {
		t.Fatalf("list home media overview: %v", err)
	}
	if len(overview.Sections) != 2 {
		t.Fatalf("expected two sections, got %#v", overview)
	}
	if overview.Sections[0].Key != "movies" || overview.Sections[0].Count != 2 || len(overview.Sections[0].Items) != 1 || overview.Sections[0].Items[0].MetadataItemID != items[0].ID {
		t.Fatalf("unexpected movie overview section: %#v", overview.Sections[0])
	}
	if overview.Sections[1].Key != "series" || overview.Sections[1].Count != 1 || len(overview.Sections[1].Items) != 1 || overview.Sections[1].Items[0].MetadataItemID != items[2].ID {
		t.Fatalf("unexpected series overview section: %#v", overview.Sections[1])
	}
}

func timePtrForHomeProjection(value time.Time) *time.Time {
	return &value
}
