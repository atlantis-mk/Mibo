package catalog

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestSearchProjectionItemsUsesLibrarySearchDocuments(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Library Search", SortTitle: "Library Search", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.LibraryMetadataProjection{LibraryID: 7, MetadataItemID: item.ID, ItemType: item.ItemType, Title: item.Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, LastProjectedAt: time.Now().UTC()}).Error; err != nil {
		t.Fatalf("create projection: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.LibrarySearchDocument{LibraryID: 7, MetadataItemID: item.ID, ItemType: item.ItemType, Title: item.Title, ResourceText: "2160p edition", AvailabilityStatus: database.ProjectionAvailabilityAvailable, UpdatedAt: time.Now().UTC()}).Error; err != nil {
		t.Fatalf("create library search doc: %v", err)
	}

	items, err := svc.SearchProjectionItems(ctx, 7, "2160p", "movie", 10)
	if err != nil {
		t.Fatalf("search projection items: %v", err)
	}
	if len(items) != 1 || items[0].MetadataItemID != item.ID {
		t.Fatalf("unexpected search results: %#v", items)
	}
}

func TestBrowseItemsGlobalMoviesUsesProjectionDatasetForTotalAndPaging(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	now := time.Now().UTC()
	items := []database.MetadataItem{
		{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Alpha Movie", SortTitle: "Alpha Movie", GovernanceStatus: database.ReviewStateAccepted},
		{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Bravo Movie", SortTitle: "Bravo Movie", GovernanceStatus: database.ReviewStateAccepted},
		{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Charlie Movie", SortTitle: "Charlie Movie", GovernanceStatus: database.ReviewStateAccepted},
	}
	if err := db.WithContext(ctx).Create(&items).Error; err != nil {
		t.Fatalf("create items: %v", err)
	}
	projections := []database.LibraryMetadataProjection{
		{LibraryID: 7, MetadataItemID: items[0].ID, ItemType: items[0].ItemType, Title: items[0].Title, SortTitle: items[0].SortTitle, AvailabilityStatus: database.ProjectionAvailabilityAvailable, LatestAddedAt: timePtrForBrowseItems(now), LastProjectedAt: now},
		{LibraryID: 8, MetadataItemID: items[0].ID, ItemType: items[0].ItemType, Title: items[0].Title, SortTitle: items[0].SortTitle, AvailabilityStatus: database.ProjectionAvailabilityAvailable, LatestAddedAt: timePtrForBrowseItems(now), LastProjectedAt: now},
		{LibraryID: 7, MetadataItemID: items[1].ID, ItemType: items[1].ItemType, Title: items[1].Title, SortTitle: items[1].SortTitle, AvailabilityStatus: database.ProjectionAvailabilityAvailable, LatestAddedAt: timePtrForBrowseItems(now.Add(-time.Minute)), LastProjectedAt: now},
		{LibraryID: 9, MetadataItemID: items[2].ID, ItemType: items[2].ItemType, Title: items[2].Title, SortTitle: items[2].SortTitle, AvailabilityStatus: database.ProjectionAvailabilityAvailable, LatestAddedAt: timePtrForBrowseItems(now.Add(-2 * time.Minute)), LastProjectedAt: now},
	}
	if err := db.WithContext(ctx).Create(&projections).Error; err != nil {
		t.Fatalf("create projections: %v", err)
	}

	result, err := svc.BrowseItems(ctx, BrowseItemsInput{
		TypeFilter:    "movie",
		WatchedState:  "all",
		OrganizingState: "all",
		Sort:          "title",
		SortDirection: "asc",
		Limit:         2,
		Offset:        0,
	})
	if err != nil {
		t.Fatalf("browse items: %v", err)
	}
	if result.Total != 3 {
		t.Fatalf("expected total 3, got %#v", result)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected first page size 2, got %#v", result.Items)
	}
	if result.Items[0].MetadataItemID != items[0].ID || result.Items[1].MetadataItemID != items[1].ID {
		t.Fatalf("unexpected first page ordering: %#v", result.Items)
	}
	if !result.HasMore {
		t.Fatalf("expected has_more true, got %#v", result)
	}

	secondPage, err := svc.BrowseItems(ctx, BrowseItemsInput{
		TypeFilter:    "movie",
		WatchedState:  "all",
		OrganizingState: "all",
		Sort:          "title",
		SortDirection: "asc",
		Limit:         2,
		Offset:        2,
	})
	if err != nil {
		t.Fatalf("browse second page: %v", err)
	}
	if secondPage.Total != 3 || len(secondPage.Items) != 1 || secondPage.Items[0].MetadataItemID != items[2].ID {
		t.Fatalf("unexpected second page: %#v", secondPage)
	}
	if secondPage.HasMore {
		t.Fatalf("expected last page has_more false, got %#v", secondPage)
	}
}

func timePtrForBrowseItems(value time.Time) *time.Time {
	return &value
}
