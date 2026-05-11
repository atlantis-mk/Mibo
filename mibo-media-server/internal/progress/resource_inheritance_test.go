package progress

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestMetadataStateFallsBackForResourceWithoutSpecificProgress(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie", GovernanceStatus: database.ReviewStateAccepted}
	resource := database.Resource{StableResourceKey: "resource:inherit", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, Status: "available"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	lastPlayed := time.Now().UTC()
	if err := db.WithContext(ctx).Create(&database.UserMetadataData{UserID: 7, MetadataItemID: item.ID, PositionSeconds: 321, LastPlayedAt: &lastPlayed}).Error; err != nil {
		t.Fatalf("create metadata data: %v", err)
	}

	state, ok, err := svc.GetMetadataState(ctx, 7, item.ID, resource.ID)
	if err != nil || !ok {
		t.Fatalf("get metadata state: ok=%v err=%v", ok, err)
	}
	if state.PositionSeconds != 321 || state.ResourceID != resource.ID || state.MetadataItemID != item.ID {
		t.Fatalf("unexpected inherited state: %#v", state)
	}
}
