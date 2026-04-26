package catalog

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestCatalogQueryAPIsReturnDetailAndGovernanceWorkspace(t *testing.T) {
	svc, ctx := newTestService(t)
	series, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Show A", Path: "/shows/ShowA", SortKey: "Show A", AvailabilityStatus: AvailabilityAvailable, GovernanceStatus: GovernanceNeedsReview})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", Path: "/shows/ShowA/Season 1", SortKey: "Show A S01", IndexNumber: &seasonNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	episodeNumber := 2
	episode, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 2", Path: "/shows/ShowA/Season 1/ShowA.S01E02.mkv", SortKey: "Show A S01E02", IndexNumber: &episodeNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	if _, err := svc.RecordMetadataSource(ctx, MetadataSourceInput{ItemID: series.ID, SourceType: SourceTypeProvider, SourceName: "tmdb", ExternalID: "tv:777", PayloadJSON: `{"title":"Show A"}`}); err != nil {
		t.Fatalf("record source: %v", err)
	}
	if _, err := svc.SetExternalID(ctx, ExternalIDInput{ItemID: series.ID, Provider: "tmdb", ProviderType: "tv", ExternalID: "tv:777", IsPrimary: true}); err != nil {
		t.Fatalf("set external id: %v", err)
	}
	if _, _, err := svc.ApplyField(ctx, ApplyFieldInput{ItemID: series.ID, FieldKey: "title", Value: "Show A", Lock: true, LockReason: "manual"}); err != nil {
		t.Fatalf("apply field: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.ItemImage{ItemID: series.ID, ImageType: "poster", URL: "https://example.com/poster.jpg", IsSelected: true}).Error; err != nil {
		t.Fatalf("create image: %v", err)
	}

	items, err := svc.ListLibraryItems(ctx, 1, "show", "show", 10)
	if err != nil {
		t.Fatalf("list library items: %v", err)
	}
	if len(items) != 1 || items[0].ID != series.ID || items[0].Type != ItemTypeSeries {
		t.Fatalf("unexpected library items: %#v", items)
	}

	detail, err := svc.GetItemDetail(ctx, series.ID)
	if err != nil {
		t.Fatalf("get item detail: %v", err)
	}
	if detail.ID != series.ID || len(detail.Seasons) != 1 || len(detail.Seasons[0].Episodes) != 1 || detail.Seasons[0].Episodes[0].ID != episode.ID {
		t.Fatalf("unexpected item detail: %#v", detail)
	}

	seasons, err := svc.ListSeriesSeasons(ctx, series.ID)
	if err != nil {
		t.Fatalf("list series seasons: %v", err)
	}
	if len(seasons) != 1 || len(seasons[0].Episodes) != 1 || seasons[0].Episodes[0].ID != episode.ID {
		t.Fatalf("unexpected seasons payload: %#v", seasons)
	}

	workspace, err := svc.GetGovernanceWorkspace(ctx, series.ID)
	if err != nil {
		t.Fatalf("get governance workspace: %v", err)
	}
	if workspace.ItemID != series.ID || len(workspace.SourceEvidence) != 1 || len(workspace.FieldStates) != 1 || len(workspace.SelectedImages) != 1 || len(workspace.RecommendedChildren) != 1 {
		t.Fatalf("unexpected governance workspace: %#v", workspace)
	}
}
