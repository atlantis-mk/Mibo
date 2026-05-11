package playback

import (
	"context"
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestResourcePlaybackMissingResourceIsUnplayable(t *testing.T) {
	fixture := newPlaybackDecisionFixture(t)
	ctx := context.Background()
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Missing Movie", GovernanceStatus: database.ReviewStateAccepted}
	if err := fixture.db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	resource := database.Resource{StableResourceKey: "resource:missing-playback", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, Status: "missing"}
	if err := fixture.db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	if err := fixture.db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary}).Error; err != nil {
		t.Fatalf("create metadata link: %v", err)
	}

	source, err := fixture.service.GetPlaybackSource(ctx, PlaybackRequest{MetadataItemID: item.ID, ResourceID: resource.ID, ClientProfile: ClientProfileWeb})
	if err != nil {
		t.Fatalf("get playback: %v", err)
	}
	if source.Playable || source.Decision.Kind != "unplayable" || !hasDecisionReasonCode(source.Decision.Reasons, "no_available_resource") {
		t.Fatalf("expected missing resource unplayable source, got %#v", source)
	}
}
