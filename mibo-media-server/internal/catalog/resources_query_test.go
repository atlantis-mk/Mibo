package catalog

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestListMetadataItemResourcesFiltersByLibrary(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie", SortTitle: "Movie", GovernanceStatus: database.ReviewStateAccepted}
	resource := database.Resource{StableResourceKey: "resource:list", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, DisplayName: "Version A", Status: "available", ProbeStatus: "ready"}
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
	if err := db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRoleVersion}).Error; err != nil {
		t.Fatalf("create metadata link: %v", err)
	}

	resources, err := svc.ListMetadataItemResources(ctx, item.ID, 7)
	if err != nil {
		t.Fatalf("list resources: %v", err)
	}
	if len(resources) != 1 || resources[0].ID != resource.ID || resources[0].LibraryID != 7 || resources[0].Role != database.ResourceLinkRoleVersion {
		t.Fatalf("unexpected resources: %#v", resources)
	}
	resources, err = svc.ListMetadataItemResources(ctx, item.ID, 8)
	if err != nil {
		t.Fatalf("list filtered resources: %v", err)
	}
	if len(resources) != 0 {
		t.Fatalf("expected no resources for other library, got %#v", resources)
	}
}
