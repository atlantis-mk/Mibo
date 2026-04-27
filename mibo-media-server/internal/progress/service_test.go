package progress

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func TestCatalogProgressUsesItemAndAssetIdentity(t *testing.T) {
	ctx := context.Background()
	service, db := newProgressTestService(t)

	libraryID := uint(1)
	runtimeSeconds := 1800
	item := database.CatalogItem{LibraryID: libraryID, Type: "episode", Title: "Catalog Episode", RuntimeSeconds: &runtimeSeconds, AvailabilityStatus: "available", GovernanceStatus: "matched"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	asset := database.MediaAsset{LibraryID: libraryID, AssetType: "main", Status: "available", ProbeStatus: "complete"}
	if err := db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create media asset: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.AssetItem{AssetID: asset.ID, ItemID: item.ID, Role: "primary", SegmentIndex: 0}).Error; err != nil {
		t.Fatalf("create asset item link: %v", err)
	}

	state, err := service.Update(ctx, 7, UpdateInput{ItemID: item.ID, AssetID: &asset.ID, PositionSeconds: 900, DurationSeconds: intPtr(1800)})
	if err != nil {
		t.Fatalf("update catalog progress: %v", err)
	}
	if state.ItemID != item.ID || state.AssetID == nil || *state.AssetID != asset.ID {
		t.Fatalf("unexpected catalog progress state ids: %#v", state)
	}
	if state.PlayedPercentage == nil || *state.PlayedPercentage != 50 {
		t.Fatalf("expected played percentage 50, got %#v", state.PlayedPercentage)
	}

	reloaded, err := service.GetCatalogState(ctx, 7, item.ID)
	if err != nil {
		t.Fatalf("get catalog state: %v", err)
	}
	if reloaded.ItemID != item.ID || reloaded.AssetID == nil || *reloaded.AssetID != asset.ID {
		t.Fatalf("unexpected reloaded catalog state: %#v", reloaded)
	}

	var stored database.UserItemData
	if err := db.WithContext(ctx).Where("user_id = ? AND item_id = ?", 7, item.ID).First(&stored).Error; err != nil {
		t.Fatalf("load user_item_data: %v", err)
	}
	if stored.PlayCount != 1 || stored.PositionSeconds != 900 {
		t.Fatalf("unexpected stored catalog progress: %#v", stored)
	}
}

func TestCatalogProgressCompletion(t *testing.T) {
	ctx := context.Background()
	service, db := newProgressTestService(t)

	runtimeSeconds := 1200
	item := database.CatalogItem{LibraryID: 1, Type: "movie", Title: "Catalog Movie", RuntimeSeconds: &runtimeSeconds, AvailabilityStatus: "available", GovernanceStatus: "matched"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create catalog item: %v", err)
	}

	state, err := service.Update(ctx, 9, UpdateInput{ItemID: item.ID, PositionSeconds: 1180, DurationSeconds: &runtimeSeconds})
	if err != nil {
		t.Fatalf("update catalog progress: %v", err)
	}
	if !state.Watched || state.CompletedAt == nil || state.LastPlayedAt == nil {
		t.Fatalf("expected completed state, got %#v", state)
	}
}

func newProgressTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	return NewService(db), db
}

func intPtr(value int) *int {
	return &value
}
