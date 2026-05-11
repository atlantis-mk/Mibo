package catalog

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestGetMetadataItemDetailReturnsProjectionAndResources(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Detail Movie", SortTitle: "Detail Movie", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	resource := database.Resource{StableResourceKey: "resource:detail", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, DisplayName: "Detail Movie 4K", Status: "available", ProbeStatus: "ready"}
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
	if err := db.WithContext(ctx).Create(&database.LibraryMetadataProjection{LibraryID: 7, MetadataItemID: item.ID, ItemType: item.ItemType, Title: item.Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, ResourceCount: 1, AvailableCount: 1, LastProjectedAt: seen}).Error; err != nil {
		t.Fatalf("create projection: %v", err)
	}

	detail, err := svc.GetMetadataItemDetail(ctx, item.ID, 7)
	if err != nil {
		t.Fatalf("get detail: %v", err)
	}
	if detail.MetadataItemID != item.ID || detail.ResourceCount != 1 || len(detail.Resources) != 1 || detail.Resources[0].ID != resource.ID {
		t.Fatalf("unexpected metadata detail: %#v", detail)
	}
}
