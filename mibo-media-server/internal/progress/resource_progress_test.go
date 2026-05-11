package progress

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestUpdateResourceProgressAggregatesMetadataProgress(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	runtimeSeconds := 1000
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie", RuntimeSeconds: &runtimeSeconds, GovernanceStatus: database.ReviewStateAccepted}
	resource := database.Resource{StableResourceKey: "resource:progress", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, Status: "available"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary}).Error; err != nil {
		t.Fatalf("create link: %v", err)
	}

	state, err := svc.Update(ctx, 7, UpdateInput{MetadataItemID: item.ID, ResourceID: resource.ID, PositionSeconds: 950})
	if err != nil {
		t.Fatalf("update resource progress: %v", err)
	}
	if !state.Watched || state.MetadataItemID != item.ID || state.ResourceID != resource.ID {
		t.Fatalf("unexpected state: %#v", state)
	}
	var resourceData database.UserResourceData
	if err := db.WithContext(ctx).Where("user_id = ? AND resource_id = ? AND metadata_item_id = ?", 7, resource.ID, item.ID).First(&resourceData).Error; err != nil {
		t.Fatalf("load resource data: %v", err)
	}
	var metadataData database.UserMetadataData
	if err := db.WithContext(ctx).Where("user_id = ? AND metadata_item_id = ?", 7, item.ID).First(&metadataData).Error; err != nil {
		t.Fatalf("load metadata data: %v", err)
	}
	if metadataData.CompletedAt == nil || metadataData.PreferredResourceID == nil || *metadataData.PreferredResourceID != resource.ID || resourceData.CompletedAt == nil {
		t.Fatalf("expected aggregated completion, metadata=%#v resource=%#v", metadataData, resourceData)
	}
}

func TestSetPreferredResourcePersistsPreference(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie", GovernanceStatus: database.ReviewStateAccepted}
	resource := database.Resource{StableResourceKey: "resource:preferred", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, Status: "available"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary}).Error; err != nil {
		t.Fatalf("create link: %v", err)
	}
	state, err := svc.SetPreferredResource(ctx, 7, item.ID, resource.ID)
	if err != nil {
		t.Fatalf("set preferred resource: %v", err)
	}
	if state.PreferredResourceID == nil || *state.PreferredResourceID != resource.ID {
		t.Fatalf("expected preferred resource in state, got %#v", state)
	}
	var metadataData database.UserMetadataData
	if err := db.WithContext(ctx).Where("user_id = ? AND metadata_item_id = ?", 7, item.ID).First(&metadataData).Error; err != nil {
		t.Fatalf("load metadata data: %v", err)
	}
	if metadataData.PreferredResourceID == nil || *metadataData.PreferredResourceID != resource.ID {
		t.Fatalf("expected persisted preferred resource, got %#v", metadataData)
	}
}
