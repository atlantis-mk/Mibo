package playback

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestResourcePlaybackUsesSelectedResourceSubtitleFiles(t *testing.T) {
	fixture := newPlaybackDecisionFixture(t)
	ctx := context.Background()
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie", GovernanceStatus: database.ReviewStateAccepted}
	if err := fixture.db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	resource, _ := seedResourcePlaybackCandidate(t, fixture, item.ID, "movie-with-sub.mp4", database.ResourceLinkRolePrimary, 1280, 720)
	subtitlePath := filepath.Join(fixture.rootPath, "movie-with-sub.srt")
	if err := os.WriteFile(subtitlePath, []byte("sub"), 0o644); err != nil {
		t.Fatalf("write subtitle: %v", err)
	}
	subtitleFile := database.InventoryFile{LibraryID: fixture.library.ID, StorageProvider: "local", StoragePath: subtitlePath, Container: "srt", ContentClass: "subtitle", Status: "available"}
	if err := fixture.db.WithContext(ctx).Create(&subtitleFile).Error; err != nil {
		t.Fatalf("create subtitle file: %v", err)
	}
	if err := fixture.db.WithContext(ctx).Create(&database.ResourceFile{ResourceID: resource.ID, InventoryFileID: subtitleFile.ID, Role: database.ResourceFileRoleSubtitle}).Error; err != nil {
		t.Fatalf("create subtitle resource file: %v", err)
	}
	if err := fixture.db.WithContext(ctx).Create(&database.MediaStream{FileID: subtitleFile.ID, StreamIndex: 1, StreamType: "subtitle", Codec: "srt", Language: "en", DispositionJSON: `{"external":true}`}).Error; err != nil {
		t.Fatalf("create subtitle stream: %v", err)
	}

	source, err := fixture.service.GetPlaybackSource(ctx, PlaybackRequest{MetadataItemID: item.ID, ResourceID: resource.ID, LibraryID: fixture.library.ID, ClientProfile: ClientProfileWeb})
	if err != nil {
		t.Fatalf("get playback: %v", err)
	}
	if len(source.SubtitleTracks) != 1 || source.SubtitleTracks[0].FileID != subtitleFile.ID || source.SubtitleTracks[0].URL == "" {
		t.Fatalf("unexpected resource subtitles: %#v", source.SubtitleTracks)
	}
}
