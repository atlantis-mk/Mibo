package playback

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/probe"
)

func TestResourcePlaybackPrefersExplicitUserPreferredResource(t *testing.T) {
	fixture := newPlaybackDecisionFixture(t)
	ctx := context.Background()
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie", GovernanceStatus: database.ReviewStateAccepted}
	if err := fixture.db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	lowResource, _ := seedResourcePlaybackCandidate(t, fixture, item.ID, "low.mp4", database.ResourceLinkRolePrimary, 1280, 720)
	highResource, _ := seedResourcePlaybackCandidate(t, fixture, item.ID, "high.mp4", database.ResourceLinkRoleVersion, 3840, 2160)
	userID := uint(7)
	if err := fixture.db.WithContext(ctx).Create(&database.UserMetadataData{UserID: userID, MetadataItemID: item.ID, PreferredResourceID: &lowResource.ID}).Error; err != nil {
		t.Fatalf("create user metadata data: %v", err)
	}

	source, err := fixture.service.GetPlaybackSource(ctx, PlaybackRequest{MetadataItemID: item.ID, UserID: &userID, ClientProfile: ClientProfileWeb})
	if err != nil {
		t.Fatalf("get playback: %v", err)
	}
	if source.ResourceID != lowResource.ID || source.Decision.SelectedBy != "user_resource_progress" || highResource.ID == 0 {
		t.Fatalf("expected explicit preferred resource, got %#v", source)
	}
}

func seedResourcePlaybackCandidate(t *testing.T, fixture *playbackDecisionFixture, metadataItemID uint, fileName string, role string, width int, height int) (database.Resource, database.InventoryFile) {
	t.Helper()
	ctx := context.Background()
	resource := database.Resource{StableResourceKey: "resource:" + fileName, ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, DisplayName: fileName, Status: "available", ProbeStatus: probe.StatusReady}
	if err := fixture.db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	filePath := filepath.Join(fixture.rootPath, fileName)
	if err := os.WriteFile(filePath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	file := database.InventoryFile{LibraryID: fixture.library.ID, StorageProvider: "local", StoragePath: filePath, Container: "mp4", Status: "available"}
	if err := fixture.db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	if err := fixture.db.WithContext(ctx).Create(&database.ResourceFile{ResourceID: resource.ID, InventoryFileID: file.ID, Role: database.ResourceFileRoleSource}).Error; err != nil {
		t.Fatalf("create resource file: %v", err)
	}
	if err := fixture.db.WithContext(ctx).Create(&database.ResourceLibraryLink{ResourceID: resource.ID, LibraryID: fixture.library.ID, Status: "available", FirstSeenAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()}).Error; err != nil {
		t.Fatalf("create library link: %v", err)
	}
	if err := fixture.db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: metadataItemID, Role: role}).Error; err != nil {
		t.Fatalf("create metadata link: %v", err)
	}
	if err := fixture.db.WithContext(ctx).Create(&database.MediaStream{FileID: file.ID, StreamIndex: 0, StreamType: "video", Codec: "h264", Width: &width, Height: &height}).Error; err != nil {
		t.Fatalf("create stream: %v", err)
	}
	return resource, file
}
