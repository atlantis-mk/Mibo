package inventory_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"gorm.io/gorm"
)

func TestAssetLinksSupportMultiEpisodeFiles(t *testing.T) {
	db, ctx := newTestDB(t)
	catalogSvc := catalog.NewService(db)
	inventorySvc := inventory.NewService(db)

	series, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeSeries, Title: "Firefly"})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonIndex := 1
	season, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", IndexNumber: &seasonIndex})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	firstIndex := 1
	first, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Title: "Serenity", IndexNumber: &firstIndex})
	if err != nil {
		t.Fatalf("create first episode: %v", err)
	}
	secondIndex := 2
	second, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Title: "The Train Job", IndexNumber: &secondIndex})
	if err != nil {
		t.Fatalf("create second episode: %v", err)
	}

	file, err := inventorySvc.UpsertFile(ctx, inventory.UpsertFileInput{LibraryID: 1, StorageProvider: "local", StoragePath: "/tv/Firefly/S01E01-E02.mkv", StableIdentityKey: "local:/tv/Firefly/S01E01-E02.mkv", SizeBytes: 1024})
	if err != nil {
		t.Fatalf("upsert file: %v", err)
	}
	asset, err := inventorySvc.CreateAsset(ctx, inventory.CreateAssetInput{LibraryID: 1, AssetType: inventory.AssetTypeMain, DisplayName: "S01E01-E02"})
	if err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if _, err := inventorySvc.LinkAssetToFile(ctx, inventory.LinkAssetFileInput{AssetID: asset.ID, FileID: file.ID, Role: inventory.FileRoleSource}); err != nil {
		t.Fatalf("link asset file: %v", err)
	}
	if _, err := inventorySvc.LinkAssetToItem(ctx, inventory.LinkAssetItemInput{AssetID: asset.ID, ItemID: first.ID, Role: inventory.AssetItemRoleMultiEpisodePart, SegmentIndex: 1}); err != nil {
		t.Fatalf("link first episode: %v", err)
	}
	if _, err := inventorySvc.LinkAssetToItem(ctx, inventory.LinkAssetItemInput{AssetID: asset.ID, ItemID: second.ID, Role: inventory.AssetItemRoleMultiEpisodePart, SegmentIndex: 2}); err != nil {
		t.Fatalf("link second episode: %v", err)
	}

	links, err := inventorySvc.ListAssetItems(ctx, asset.ID)
	if err != nil {
		t.Fatalf("list asset items: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected two episode links, got %#v", links)
	}
	if links[0].ItemID != first.ID || links[0].SegmentIndex != 1 || links[1].ItemID != second.ID || links[1].SegmentIndex != 2 {
		t.Fatalf("unexpected episode link order: %#v", links)
	}
}

func TestUpsertFileRefreshesInventoryRecord(t *testing.T) {
	db, ctx := newTestDB(t)
	svc := inventory.NewService(db)

	first, err := svc.UpsertFile(ctx, inventory.UpsertFileInput{LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/A.mkv", SizeBytes: 100})
	if err != nil {
		t.Fatalf("upsert first file: %v", err)
	}
	second, err := svc.UpsertFile(ctx, inventory.UpsertFileInput{LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/A.mkv", SizeBytes: 200, Container: "mkv"})
	if err != nil {
		t.Fatalf("upsert second file: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected same inventory file id, got %d and %d", first.ID, second.ID)
	}
	if second.SizeBytes != 200 || second.Container != "mkv" {
		t.Fatalf("expected refreshed file metadata, got %#v", second)
	}
}

func newTestDB(t *testing.T) (*gorm.DB, context.Context) {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	return db, context.Background()
}
