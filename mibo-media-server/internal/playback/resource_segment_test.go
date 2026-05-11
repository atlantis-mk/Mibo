package playback

import (
	"context"
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestResourcePlaybackReturnsMultiEpisodeSegment(t *testing.T) {
	fixture := newPlaybackDecisionFixture(t)
	ctx := context.Background()
	first := database.MetadataItem{ItemType: database.MetadataItemTypeEpisode, ContentForm: database.MetadataContentFormStandard, Title: "Episode 1", GovernanceStatus: database.ReviewStateAccepted}
	second := database.MetadataItem{ItemType: database.MetadataItemTypeEpisode, ContentForm: database.MetadataContentFormStandard, Title: "Episode 2", GovernanceStatus: database.ReviewStateAccepted}
	if err := fixture.db.WithContext(ctx).Create(&first).Error; err != nil {
		t.Fatalf("create first: %v", err)
	}
	if err := fixture.db.WithContext(ctx).Create(&second).Error; err != nil {
		t.Fatalf("create second: %v", err)
	}
	resource, _ := seedResourcePlaybackCandidate(t, fixture, first.ID, "multi-episode.mp4", database.ResourceLinkRolePrimary, 1280, 720)
	start := 1800.0
	end := 3600.0
	if err := fixture.db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: second.ID, Role: database.ResourceLinkRolePrimary, SegmentIndex: 2, StartSeconds: &start, EndSeconds: &end}).Error; err != nil {
		t.Fatalf("create second segment link: %v", err)
	}

	source, err := fixture.service.GetPlaybackSource(ctx, PlaybackRequest{MetadataItemID: second.ID, ResourceID: resource.ID, ClientProfile: ClientProfileWeb})
	if err != nil {
		t.Fatalf("get playback: %v", err)
	}
	if source.MetadataItemID != second.ID || source.ResourceID != resource.ID || source.SegmentIndex != 2 || source.StartSeconds == nil || *source.StartSeconds != start || source.EndSeconds == nil || *source.EndSeconds != end {
		t.Fatalf("unexpected segment playback source: %#v", source)
	}
}
