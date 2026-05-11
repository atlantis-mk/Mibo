package catalog

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestListLibraryProjectionItemsReturnsMetadataIDsAndResourceSummary(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	year := 2026
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Projection Movie", SortTitle: "Projection Movie", Year: &year, GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.LibraryMetadataProjection{LibraryID: 7, MetadataItemID: item.ID, ItemType: item.ItemType, Title: item.Title, SortTitle: item.SortTitle, Year: item.Year, AvailabilityStatus: database.ProjectionAvailabilityAvailable, ResourceCount: 2, AvailableCount: 1, MissingCount: 1, LastProjectedAt: time.Now().UTC()}).Error; err != nil {
		t.Fatalf("create projection: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.MetadataItemImage{MetadataItemID: item.ID, ImageType: "poster", URL: "https://image/poster.jpg", IsSelected: true}).Error; err != nil {
		t.Fatalf("create image: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.MetadataExternalID{MetadataItemID: item.ID, Provider: "tmdb", ProviderType: "movie", ExternalID: "movie:42", IsPrimary: true}).Error; err != nil {
		t.Fatalf("create external id: %v", err)
	}

	items, err := svc.ListLibraryProjectionItems(ctx, 7, "Projection", "movie", 20)
	if err != nil {
		t.Fatalf("list projection items: %v", err)
	}
	if len(items) != 1 || items[0].ID != item.ID || items[0].MetadataItemID != item.ID || items[0].ResourceCount != 2 || len(items[0].SelectedImages) != 1 || len(items[0].ExternalIdentities) != 1 {
		t.Fatalf("unexpected projection items: %#v", items)
	}
}

func TestListLibraryItemsIncludesDiscoveredInventoryEntries(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	year := 2026
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Projection Movie", SortTitle: "Projection Movie", Year: &year, GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.LibraryMetadataProjection{LibraryID: 7, MetadataItemID: item.ID, ItemType: item.ItemType, Title: item.Title, SortTitle: item.SortTitle, Year: item.Year, AvailabilityStatus: database.ProjectionAvailabilityAvailable, ResourceCount: 1, AvailableCount: 1, LastProjectedAt: time.Now().UTC()}).Error; err != nil {
		t.Fatalf("create projection: %v", err)
	}
	file := database.InventoryFile{LibraryID: 7, StorageProvider: "local", StoragePath: "/library/Fresh.Movie.2026.mkv", ContentClass: "video", Status: "available", ScanState: "discovered"}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}

	items, err := svc.ListLibraryItems(ctx, 7, "", "movie", 20)
	if err != nil {
		t.Fatalf("list library items: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected projection and discovered entry, got %#v", items)
	}
	var discovered *CatalogListItem
	for idx := range items {
		if items[idx].SourceKind == "inventory_file" {
			discovered = &items[idx]
			break
		}
	}
	if discovered == nil {
		t.Fatalf("expected discovered inventory entry, got %#v", items)
	}
	if discovered.InventoryFileID == nil || *discovered.InventoryFileID != file.ID {
		t.Fatalf("expected discovered entry to point at inventory file %d, got %#v", file.ID, discovered)
	}
	if discovered.MetadataItemID != 0 || !discovered.Organizing {
		t.Fatalf("expected discovered entry to remain pre-metadata and organizing, got %#v", discovered)
	}
}
