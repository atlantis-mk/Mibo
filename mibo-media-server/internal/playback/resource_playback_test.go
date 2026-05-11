package playback

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/probe"
)

func TestResourcePlaybackDirectFromMetadataItem(t *testing.T) {
	fixture := newPlaybackDecisionFixture(t)
	ctx := context.Background()
	runtimeSeconds := 7200
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Resource Movie", RuntimeSeconds: &runtimeSeconds, GovernanceStatus: database.ReviewStateAccepted}
	if err := fixture.db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	resource := database.Resource{StableResourceKey: "resource:playback", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, DisplayName: "Resource Movie 4K", Status: "available", ProbeStatus: probe.StatusReady, QualityLabel: "2160p"}
	if err := fixture.db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	filePath := filepath.Join(fixture.rootPath, "resource-movie.mp4")
	if err := os.WriteFile(filePath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write inventory file: %v", err)
	}
	file := database.InventoryFile{LibraryID: fixture.library.ID, StorageProvider: "local", StoragePath: filePath, Container: "mp4", Status: "available"}
	if err := fixture.db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	if err := fixture.db.WithContext(ctx).Create(&database.ResourceFile{ResourceID: resource.ID, InventoryFileID: file.ID, Role: database.ResourceFileRoleSource}).Error; err != nil {
		t.Fatalf("create resource file: %v", err)
	}
	if err := fixture.db.WithContext(ctx).Create(&database.ResourceLibraryLink{ResourceID: resource.ID, LibraryID: fixture.library.ID, Status: "available", FirstSeenAt: item.CreatedAt, LastSeenAt: item.CreatedAt}).Error; err != nil {
		t.Fatalf("create library link: %v", err)
	}
	if err := fixture.db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary}).Error; err != nil {
		t.Fatalf("create metadata link: %v", err)
	}
	width := 3840
	height := 2160
	if err := fixture.db.WithContext(ctx).Create(&database.MediaStream{FileID: file.ID, StreamIndex: 0, StreamType: "video", Codec: "h264", Width: &width, Height: &height}).Error; err != nil {
		t.Fatalf("create stream: %v", err)
	}

	source, err := fixture.service.GetPlaybackSource(ctx, PlaybackRequest{MetadataItemID: item.ID, ResourceID: resource.ID, LibraryID: fixture.library.ID, ClientProfile: ClientProfileWeb})
	if err != nil {
		t.Fatalf("get playback source: %v", err)
	}
	if source.MetadataItemID != item.ID || source.ResourceID != resource.ID || source.FileID != file.ID || !source.Playable || source.Decision.SelectedBy != "preferred_resource" {
		t.Fatalf("unexpected resource playback source: %#v", source)
	}
	if source.URL != "/api/v1/inventory-files/"+uintStringForPlayback(file.ID)+"/stream" {
		t.Fatalf("expected local playback proxy url, got %q", source.URL)
	}
}

func uintStringForPlayback(value uint) string {
	return fmt.Sprintf("%d", value)
}
