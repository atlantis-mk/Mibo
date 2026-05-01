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

func TestCatalogProgressUsesLibraryPlaybackCompletionThreshold(t *testing.T) {
	ctx := context.Background()
	service, db := newProgressTestService(t)
	runtimeSeconds := 1200
	item := database.CatalogItem{LibraryID: 1, Type: "movie", Title: "Catalog Movie", RuntimeSeconds: &runtimeSeconds, AvailabilityStatus: "available", GovernanceStatus: "matched"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	if err := database.EnsureLibraryPolicyDefaults(db, 1); err != nil {
		t.Fatalf("ensure policy defaults: %v", err)
	}
	if err := db.WithContext(ctx).Model(&database.LibraryPlaybackPolicy{}).Where("library_id = ?", 1).Update("max_resume_pct", 50).Error; err != nil {
		t.Fatalf("update playback policy: %v", err)
	}
	state, err := service.Update(ctx, 9, UpdateInput{ItemID: item.ID, PositionSeconds: 600, DurationSeconds: &runtimeSeconds})
	if err != nil {
		t.Fatalf("update catalog progress: %v", err)
	}
	if !state.Watched || state.CompletedAt == nil {
		t.Fatalf("expected policy threshold to mark completed, got %#v", state)
	}
}

func TestCatalogProgressSkipsShortDurationResume(t *testing.T) {
	ctx := context.Background()
	service, db := newProgressTestService(t)
	runtimeSeconds := 120
	item := database.CatalogItem{LibraryID: 1, Type: "movie", Title: "Short", RuntimeSeconds: &runtimeSeconds, AvailabilityStatus: "available", GovernanceStatus: "matched"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	state, err := service.Update(ctx, 9, UpdateInput{ItemID: item.ID, PositionSeconds: 30, DurationSeconds: &runtimeSeconds})
	if err != nil {
		t.Fatalf("update catalog progress: %v", err)
	}
	if state.PositionSeconds != 0 || state.PlayedPercentage != nil {
		t.Fatalf("expected short media resume to be ignored, got %#v", state)
	}
	var count int64
	if err := db.WithContext(ctx).Model(&database.UserItemData{}).Where("user_id = ? AND item_id = ?", 9, item.ID).Count(&count).Error; err != nil {
		t.Fatalf("count progress rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no progress row for short resumable media, got %d", count)
	}
}

func TestGetCatalogStateReturnsEmptyStateWhenProgressMissing(t *testing.T) {
	ctx := context.Background()
	service, db := newProgressTestService(t)

	runtimeSeconds := 1800
	item := database.CatalogItem{LibraryID: 1, Type: "movie", Title: "Unplayed", RuntimeSeconds: &runtimeSeconds, AvailabilityStatus: "available", GovernanceStatus: "matched"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create catalog item: %v", err)
	}

	state, err := service.GetCatalogState(ctx, 9, item.ID)
	if err != nil {
		t.Fatalf("get catalog state: %v", err)
	}
	if state.UserID != 9 || state.ItemID != item.ID || state.PositionSeconds != 0 || state.Watched {
		t.Fatalf("unexpected empty progress state: %#v", state)
	}
	if state.DurationSeconds == nil || *state.DurationSeconds != runtimeSeconds {
		t.Fatalf("expected runtime duration, got %#v", state.DurationSeconds)
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
