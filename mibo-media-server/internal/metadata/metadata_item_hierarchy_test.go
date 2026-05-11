package metadata

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestApplyNormalizedMetadataItemTVHierarchyCreatesGlobalChildren(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db, config.MetadataConfig{}, nil)
	series := database.MetadataItem{ItemType: database.MetadataItemTypeSeries, ContentForm: database.MetadataContentFormStandard, Title: "Show", SortTitle: "Show", GovernanceStatus: database.ReviewStatePending}
	if err := db.WithContext(ctx).Create(&series).Error; err != nil {
		t.Fatalf("create series metadata: %v", err)
	}
	confidence := 0.95
	hierarchy := NormalizedMetadataHierarchy{Seasons: []NormalizedMetadataSeason{{SeasonNumber: 1, Title: "Season 1", ExternalID: "season:10", Episodes: []NormalizedMetadataEpisode{{SeasonNumber: 1, EpisodeNumber: 1, Title: "Pilot", ExternalID: "episode:100"}}}}}

	result, err := svc.applyNormalizedMetadataItemTVHierarchy(ctx, series, MetadataExecutionPlan{LibraryID: 7, PreferredMetadataLanguage: "en"}, hierarchy, database.ReviewStateAccepted, confidence, false)
	if err != nil {
		t.Fatalf("apply hierarchy: %v", err)
	}
	if len(result.AffectedMetadataItemIDs) != 3 {
		t.Fatalf("expected series, season, episode affected IDs, got %#v", result.AffectedMetadataItemIDs)
	}
	var season database.MetadataItem
	if err := db.WithContext(ctx).Where("item_type = ? AND parent_id = ?", database.MetadataItemTypeSeason, series.ID).First(&season).Error; err != nil {
		t.Fatalf("load season: %v", err)
	}
	var episode database.MetadataItem
	if err := db.WithContext(ctx).Where("item_type = ? AND parent_id = ?", database.MetadataItemTypeEpisode, season.ID).First(&episode).Error; err != nil {
		t.Fatalf("load episode: %v", err)
	}
	if episode.RootID == nil || *episode.RootID != series.ID || episode.IndexNumber == nil || *episode.IndexNumber != 1 {
		t.Fatalf("unexpected episode hierarchy metadata: %#v", episode)
	}
	assertMetadataItemPolicyCount(t, ctx, db, &database.MetadataItemSource{}, 2)
	assertMetadataItemPolicyCount(t, ctx, db, &database.MetadataExternalID{}, 2)
}
