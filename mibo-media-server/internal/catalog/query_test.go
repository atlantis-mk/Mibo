package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
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
	createPlayableCatalogAsset(t, svc, ctx, episode.ID)
	related, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Related Show", Path: "/shows/Related", SortKey: "Related Show", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create related item: %v", err)
	}
	genre := database.Tag{Kind: "genre", Name: "Drama"}
	topic := database.Tag{Kind: "topic", Name: "Space"}
	if err := svc.db.WithContext(ctx).Create([]*database.Tag{&genre, &topic}).Error; err != nil {
		t.Fatalf("create tags: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create([]database.ItemTag{
		{ItemID: series.ID, TagID: genre.ID},
		{ItemID: series.ID, TagID: topic.ID},
		{ItemID: related.ID, TagID: genre.ID},
	}).Error; err != nil {
		t.Fatalf("link tags: %v", err)
	}
	actor := database.Person{Name: "Actor A", SortName: "actor a", AvatarURL: "https://example.com/actor-a.jpg"}
	director := database.Person{Name: "Director A", SortName: "director a", AvatarURL: "https://example.com/director-a.jpg"}
	if err := svc.db.WithContext(ctx).Create([]*database.Person{&actor, &director}).Error; err != nil {
		t.Fatalf("create people: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create([]database.ItemPerson{{ItemID: series.ID, PersonID: actor.ID, Role: "cast", Character: "Lead", SortOrder: 0}, {ItemID: series.ID, PersonID: director.ID, Role: "director", Character: "Director", SortOrder: 0}}).Error; err != nil {
		t.Fatalf("link people: %v", err)
	}

	items, err := svc.ListLibraryItems(ctx, 1, "Show A", "show", 10)
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
	if len(detail.Cast) != 1 || detail.Cast[0].ID != actor.ID || detail.Cast[0].Name != "Actor A" || detail.Cast[0].Role != "Lead" || detail.Cast[0].AvatarURL != "https://example.com/actor-a.jpg" {
		t.Fatalf("unexpected cast detail: %#v", detail.Cast)
	}
	if len(detail.Directors) != 1 || detail.Directors[0].ID != director.ID || detail.Directors[0].Name != "Director A" || detail.Directors[0].Role != "Director" || detail.Directors[0].AvatarURL != "https://example.com/director-a.jpg" {
		t.Fatalf("unexpected directors detail: %#v", detail.Directors)
	}
	if len(detail.Tags) != 2 || detail.Tags[0].Kind != "genre" || detail.Tags[0].Name != "Drama" || detail.Tags[1].Name != "Space" {
		t.Fatalf("unexpected tags detail: %#v", detail.Tags)
	}
	if len(detail.Genres) != 1 || detail.Genres[0] != "Drama" {
		t.Fatalf("unexpected genres detail: %#v", detail.Genres)
	}
	if len(detail.RelatedItems) == 0 || detail.RelatedItems[0].ID != related.ID {
		t.Fatalf("unexpected related detail: %#v", detail.RelatedItems)
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

func TestBrowseItemsIncludesUnavailableItems(t *testing.T) {
	svc, ctx := newTestService(t)
	available, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Available Movie", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create available item: %v", err)
	}
	missing, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Missing Movie", AvailabilityStatus: AvailabilityMissing})
	if err != nil {
		t.Fatalf("create missing item: %v", err)
	}

	result, err := svc.BrowseItems(ctx, BrowseItemsInput{LibraryID: 1, TypeFilter: "all", Limit: 20})
	if err != nil {
		t.Fatalf("browse items: %v", err)
	}
	if result.Total != 2 || len(result.Items) != 2 {
		t.Fatalf("expected available and missing items in browse result, got %#v", result)
	}
	ids := map[uint]bool{result.Items[0].ID: true, result.Items[1].ID: true}
	if !ids[available.ID] || !ids[missing.ID] {
		t.Fatalf("expected available item %d and missing item %d, got %#v", available.ID, missing.ID, result)
	}
}

func TestBrowseItemsIncludesSeriesWithoutPlayableDescendants(t *testing.T) {
	svc, ctx := newTestService(t)
	orphan, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Orphan Show", Path: "/shows/orphan", SortKey: "Orphan Show", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create orphan series: %v", err)
	}
	playable, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Playable Show", Path: "/shows/playable", SortKey: "Playable Show", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create playable series: %v", err)
	}
	seasonNumber := 1
	season, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeason, ParentID: &playable.ID, Title: "Season 1", Path: "/shows/playable/season-01", SortKey: "Playable Show S01", IndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create playable season: %v", err)
	}
	episodeNumber := 1
	episode, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 1", Path: "/shows/playable/season-01/episode-001", SortKey: "Playable Show S01E01", IndexNumber: &episodeNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create playable episode: %v", err)
	}
	createPlayableCatalogAsset(t, svc, ctx, episode.ID)

	result, err := svc.BrowseItems(ctx, BrowseItemsInput{LibraryID: 1, TypeFilter: ItemTypeSeries, Limit: 20})
	if err != nil {
		t.Fatalf("browse series: %v", err)
	}
	if result.Total != 2 || len(result.Items) != 2 {
		t.Fatalf("expected playable and orphan series, got %#v", result)
	}
	ids := map[uint]bool{result.Items[0].ID: true, result.Items[1].ID: true}
	if !ids[playable.ID] || !ids[orphan.ID] {
		t.Fatalf("expected playable series %d and orphan series %d, got %#v", playable.ID, orphan.ID, result)
	}
}

func TestBrowseItemsIncludesDiscoveredInventoryEntries(t *testing.T) {
	svc, ctx := newTestService(t)
	available, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Available Movie", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create available item: %v", err)
	}
	file := database.InventoryFile{LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/New.Movie.2024.mkv", Container: "mkv", ContentClass: "video", Status: AvailabilityAvailable, ScanState: "discovered"}
	if err := svc.db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}

	result, err := svc.BrowseItems(ctx, BrowseItemsInput{LibraryID: 1, TypeFilter: "all", Sort: "title", SortDirection: "asc", Limit: 20})
	if err != nil {
		t.Fatalf("browse items: %v", err)
	}
	if result.Total != 2 || len(result.Items) != 2 {
		t.Fatalf("expected catalog and discovered entries, got %#v", result)
	}
	if result.Items[0].ID != available.ID || result.Items[0].SourceKind != "catalog" || result.Items[0].MaturityState != "classified" {
		t.Fatalf("unexpected catalog entry: %#v", result.Items[0])
	}
	discovered := result.Items[1]
	if discovered.SourceKind != "inventory_file" || discovered.InventoryFileID == nil || *discovered.InventoryFileID != file.ID || discovered.MaturityState != "discovered" || !discovered.Organizing {
		t.Fatalf("unexpected discovered entry: %#v", discovered)
	}
	if discovered.Title != "New Movie 2024" || discovered.StoragePath != file.StoragePath {
		t.Fatalf("unexpected discovered display fields: %#v", discovered)
	}
	if discovered.OrganizingSummary == nil || discovered.OrganizingSummary.State != "organizing" || discovered.OrganizingSummary.Stage != ingest.ConditionMaterialized {
		t.Fatalf("unexpected discovered organizing summary: %#v", discovered.OrganizingSummary)
	}
}

func TestBrowseItemsUsesDiscoveredInventoryThumbnail(t *testing.T) {
	svc, ctx := newTestService(t)
	file := database.InventoryFile{LibraryID: 1, StorageProvider: "openlist", StoragePath: "/movies/New.Movie.2024.mkv", ThumbnailURL: "https://cdn.example.test/movie-thumb.jpg", Container: "mkv", ContentClass: "video", Status: AvailabilityAvailable, ScanState: "discovered"}
	if err := svc.db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}

	result, err := svc.BrowseItems(ctx, BrowseItemsInput{LibraryID: 1, TypeFilter: "all", Sort: "title", SortDirection: "asc", Limit: 20})
	if err != nil {
		t.Fatalf("browse items: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("expected one discovered entry, got %#v", result)
	}
	images := result.Items[0].SelectedImages
	if len(images) != 1 || images[0].ImageType != "poster" || images[0].URL != "https://cdn.example.test/movie-thumb.jpg" {
		t.Fatalf("expected discovered thumbnail poster, got %#v", images)
	}
}

func TestBrowseItemsUsesDiscoveredInventoryStorageIndexThumbnail(t *testing.T) {
	svc, ctx := newTestService(t)
	file := database.InventoryFile{LibraryID: 1, StorageProvider: "openlist", StoragePath: "/movies/New.Movie.2024.mkv", Container: "mkv", ContentClass: "video", Status: AvailabilityAvailable, ScanState: "discovered"}
	if err := svc.db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.StorageIndexEntry{LibraryID: 1, StorageProvider: "openlist", StoragePath: file.StoragePath, SizeBytes: 100, ObservationStatus: "present", ProviderMetaJSON: `{"thumbnail_url":"https://cdn.example.test/index-thumb.jpg"}`}).Error; err != nil {
		t.Fatalf("create storage index entry: %v", err)
	}

	result, err := svc.BrowseItems(ctx, BrowseItemsInput{LibraryID: 1, TypeFilter: "all", Sort: "title", SortDirection: "asc", Limit: 20})
	if err != nil {
		t.Fatalf("browse items: %v", err)
	}
	images := result.Items[0].SelectedImages
	if len(images) != 1 || images[0].ImageType != "poster" || images[0].URL != "https://cdn.example.test/index-thumb.jpg" {
		t.Fatalf("expected storage index thumbnail poster, got %#v", images)
	}
}

func TestBrowseItemsSearchDoesNotBypassTypeFilter(t *testing.T) {
	svc, ctx := newTestService(t)
	seasonNumber := 1
	episodeNumber := 8
	series, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "High Education", SortKey: "High Education", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	season, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", SortKey: "High Education Season 1", IndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	episode, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 8", OriginalTitle: "Vladimir.S01E08.Against.Interpretation", SortKey: "High Education S01E08", ParentIndexNumber: &seasonNumber, IndexNumber: &episodeNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	if err := svc.RefreshItemProjection(ctx, episode.ID); err != nil {
		t.Fatalf("refresh episode projection: %v", err)
	}

	result, err := svc.BrowseItems(ctx, BrowseItemsInput{LibraryID: 1, Query: "S01E08", TypeFilter: "all", Sort: "title", SortDirection: "desc", Limit: 20})
	if err != nil {
		t.Fatalf("browse items: %v", err)
	}
	for _, item := range result.Items {
		if item.Type == ItemTypeEpisode {
			t.Fatalf("expected type=all search to exclude episodes, got %#v", item)
		}
	}
}

func TestBrowseItemsIncludesCatalogOrganizingSummary(t *testing.T) {
	svc, ctx := newTestService(t)
	movie, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Organizing Movie", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.IngestCondition{UnitKey: "catalog_item:1", LibraryID: 1, CatalogItemID: &movie.ID, ConditionType: ingest.ConditionMetadataMatched, Status: ingest.ConditionStatusReviewRequired, Reason: "no_candidate", Message: "Metadata match needed", Severity: ingest.SeverityWarning}).Error; err != nil {
		t.Fatalf("create ingest condition: %v", err)
	}

	result, err := svc.BrowseItems(ctx, BrowseItemsInput{LibraryID: 1, TypeFilter: "all", Limit: 20})
	if err != nil {
		t.Fatalf("browse items: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("expected one item, got %#v", result)
	}
	summary := result.Items[0].OrganizingSummary
	if summary == nil || summary.State != "review_required" || summary.Stage != ingest.ConditionMetadataMatched || summary.Message != "Metadata match needed" {
		t.Fatalf("unexpected organizing summary: %#v", summary)
	}
	if !result.Items[0].Organizing {
		t.Fatalf("expected organizing flag for review-required summary: %#v", result.Items[0])
	}
}

func TestBrowseItemsTreatsMaterializedPendingEnhancementsAsReady(t *testing.T) {
	svc, ctx := newTestService(t)
	movie, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Fast Movie", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}
	conditions := []database.IngestCondition{
		{UnitKey: "catalog_item:1", LibraryID: 1, CatalogItemID: &movie.ID, ConditionType: ingest.ConditionMaterialized, Status: ingest.ConditionStatusTrue, Reason: "linked", Message: "Media is linked", Severity: ingest.SeverityInfo},
		{UnitKey: "catalog_item:1", LibraryID: 1, CatalogItemID: &movie.ID, ConditionType: ingest.ConditionProbed, Status: ingest.ConditionStatusPending, Reason: "probe_pending", Message: "Media probe is pending", Severity: ingest.SeverityInfo},
		{UnitKey: "catalog_item:1", LibraryID: 1, CatalogItemID: &movie.ID, ConditionType: ingest.ConditionMetadataMatched, Status: ingest.ConditionStatusPending, Reason: "pending", Message: "Metadata matching is pending", Severity: ingest.SeverityInfo},
		{UnitKey: "catalog_item:1", LibraryID: 1, CatalogItemID: &movie.ID, ConditionType: ingest.ConditionProjectionCurrent, Status: ingest.ConditionStatusPending, Reason: "projection_pending", Message: "Catalog projection refresh is pending", Severity: ingest.SeverityInfo},
	}
	if err := svc.db.WithContext(ctx).Create(&conditions).Error; err != nil {
		t.Fatalf("create ingest conditions: %v", err)
	}

	result, err := svc.BrowseItems(ctx, BrowseItemsInput{LibraryID: 1, TypeFilter: "all", Limit: 20})
	if err != nil {
		t.Fatalf("browse items: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("expected one item, got %#v", result)
	}
	item := result.Items[0]
	if item.Organizing {
		t.Fatalf("expected materialized item to be organized while enhancements are pending: %#v", item)
	}
	if item.OrganizingSummary == nil || item.OrganizingSummary.State != "ready" {
		t.Fatalf("expected ready organizing summary, got %#v", item.OrganizingSummary)
	}
}

func TestBrowseItemsIncludesPartialReadyProbeFailureSummary(t *testing.T) {
	svc, ctx := newTestService(t)
	movie, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Playable Movie", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}
	conditions := []database.IngestCondition{
		{UnitKey: "catalog_item:1", LibraryID: 1, CatalogItemID: &movie.ID, ConditionType: ingest.ConditionMaterialized, Status: ingest.ConditionStatusTrue, Reason: "linked", Message: "Media is linked", Severity: ingest.SeverityInfo},
		{UnitKey: "catalog_item:1", LibraryID: 1, CatalogItemID: &movie.ID, ConditionType: ingest.ConditionProbed, Status: ingest.ConditionStatusFailed, Reason: "probe_failed", Message: "Media probe failed", Severity: ingest.SeverityError},
		{UnitKey: "catalog_item:1", LibraryID: 1, CatalogItemID: &movie.ID, ConditionType: ingest.ConditionMetadataMatched, Status: ingest.ConditionStatusTrue, Reason: "matched", Message: "Metadata matched", Severity: ingest.SeverityInfo},
	}
	if err := svc.db.WithContext(ctx).Create(&conditions).Error; err != nil {
		t.Fatalf("create ingest conditions: %v", err)
	}

	result, err := svc.BrowseItems(ctx, BrowseItemsInput{LibraryID: 1, TypeFilter: "all", Limit: 20})
	if err != nil {
		t.Fatalf("browse items: %v", err)
	}
	summary := result.Items[0].OrganizingSummary
	if summary == nil || summary.State != "failed" || summary.Stage != ingest.ConditionProbed || summary.Message != "Media probe failed" {
		t.Fatalf("unexpected failed probe summary: %#v", summary)
	}
}

func TestBrowseItemsSuppressesDiscoveredEntryLinkedToCatalog(t *testing.T) {
	svc, ctx := newTestService(t)
	movie, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Linked Movie", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}
	asset := database.MediaAsset{LibraryID: 1, AssetType: "main", Status: AvailabilityAvailable, ProbeStatus: "pending"}
	if err := svc.db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	file := database.InventoryFile{LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/Linked.Movie.mkv", Container: "mkv", ContentClass: "video", Status: AvailabilityAvailable, ScanState: "classified"}
	if err := svc.db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: "source"}).Error; err != nil {
		t.Fatalf("link asset file: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.AssetItem{AssetID: asset.ID, ItemID: movie.ID, Role: "primary"}).Error; err != nil {
		t.Fatalf("link asset item: %v", err)
	}

	result, err := svc.BrowseItems(ctx, BrowseItemsInput{LibraryID: 1, TypeFilter: "all", Limit: 20})
	if err != nil {
		t.Fatalf("browse items: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].ID != movie.ID || result.Items[0].SourceKind != "catalog" {
		t.Fatalf("expected only catalog-backed result, got %#v", result)
	}
}

func TestBrowseItemsQueryMatchesProjectionSearchDocumentFields(t *testing.T) {
	svc, ctx := newTestService(t)
	movie, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Projection Movie", SortKey: "Projection Movie", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}
	if _, err := svc.SetExternalID(ctx, ExternalIDInput{ItemID: movie.ID, Provider: "tmdb", ProviderType: "movie", ExternalID: "tmdb-999", IsPrimary: true}); err != nil {
		t.Fatalf("set external id: %v", err)
	}

	result, err := svc.BrowseItems(ctx, BrowseItemsInput{LibraryID: 1, Query: "tmdb-999", TypeFilter: "all", Limit: 20})
	if err != nil {
		t.Fatalf("browse items: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].ID != movie.ID {
		t.Fatalf("expected query to match projection-backed provider id text, got %#v", result)
	}
}

func TestBrowseItemsShowsCatalogItemsWithoutProjectionDocument(t *testing.T) {
	svc, ctx := newTestService(t)
	movie, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Visible Without Projection", SortKey: "Visible Without Projection", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}
	if err := svc.db.WithContext(ctx).Where("item_id = ?", movie.ID).Delete(&database.CatalogSearchDocument{}).Error; err != nil {
		t.Fatalf("delete projection document: %v", err)
	}

	result, err := svc.BrowseItems(ctx, BrowseItemsInput{LibraryID: 1, TypeFilter: "all", Limit: 20})
	if err != nil {
		t.Fatalf("browse items: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].ID != movie.ID {
		t.Fatalf("expected catalog item without projection document to remain visible, got %#v", result)
	}
}

func TestBrowseItemsCatalogPrefixPaginationPreservesMergedOrdering(t *testing.T) {
	svc, ctx := newTestService(t)
	alpha, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Alpha Movie", SortKey: "Alpha Movie", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create alpha item: %v", err)
	}
	if _, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Zulu Movie", SortKey: "Zulu Movie", AvailabilityStatus: AvailabilityAvailable}); err != nil {
		t.Fatalf("create zulu item: %v", err)
	}
	file := database.InventoryFile{LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/Mid.Movie.2024.mkv", Container: "mkv", ContentClass: "video", Status: AvailabilityAvailable, ScanState: "discovered"}
	if err := svc.db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}

	result, err := svc.BrowseItems(ctx, BrowseItemsInput{LibraryID: 1, TypeFilter: "all", Sort: "title", SortDirection: "asc", Offset: 1, Limit: 1})
	if err != nil {
		t.Fatalf("browse items: %v", err)
	}
	if result.Total != 3 || len(result.Items) != 1 {
		t.Fatalf("expected one paged result out of three merged entries, got %#v", result)
	}
	item := result.Items[0]
	if item.SourceKind != "inventory_file" || item.InventoryFileID == nil || *item.InventoryFileID != file.ID {
		t.Fatalf("expected discovered entry in merged middle slot, got %#v", item)
	}
	if result.HasMore != true {
		t.Fatalf("expected more merged results after middle slot, got %#v", result)
	}
	if alpha.ID == 0 {
		t.Fatalf("expected seeded catalog item")
	}
}

func TestBrowseItemsOrganizedFilterExcludesItemsStillOrganizing(t *testing.T) {
	svc, ctx := newTestService(t)
	ready, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Ready Movie", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create ready item: %v", err)
	}
	organizing, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Organizing Movie", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create organizing item: %v", err)
	}
	conditions := []database.IngestCondition{
		{UnitKey: "catalog_item:ready", LibraryID: 1, CatalogItemID: &ready.ID, ConditionType: ingest.ConditionMaterialized, Status: ingest.ConditionStatusTrue, Reason: "linked", Message: "Media is linked", Severity: ingest.SeverityInfo},
		{UnitKey: "catalog_item:ready", LibraryID: 1, CatalogItemID: &ready.ID, ConditionType: ingest.ConditionProbed, Status: ingest.ConditionStatusPending, Reason: "probe_pending", Message: "Media probe is pending", Severity: ingest.SeverityInfo},
		{UnitKey: "catalog_item:ready", LibraryID: 1, CatalogItemID: &ready.ID, ConditionType: ingest.ConditionMetadataMatched, Status: ingest.ConditionStatusPending, Reason: "pending", Message: "Metadata matching is pending", Severity: ingest.SeverityInfo},
		{UnitKey: "catalog_item:ready", LibraryID: 1, CatalogItemID: &ready.ID, ConditionType: ingest.ConditionProjectionCurrent, Status: ingest.ConditionStatusPending, Reason: "projection_pending", Message: "Catalog projection refresh is pending", Severity: ingest.SeverityInfo},
		{UnitKey: "catalog_item:organizing", LibraryID: 1, CatalogItemID: &organizing.ID, ConditionType: ingest.ConditionMaterialized, Status: ingest.ConditionStatusPending, Reason: "awaiting_materialization", Message: "Media is waiting for catalog materialization", Severity: ingest.SeverityInfo},
	}
	if err := svc.db.WithContext(ctx).Create(&conditions).Error; err != nil {
		t.Fatalf("create ingest conditions: %v", err)
	}

	result, err := svc.BrowseItems(ctx, BrowseItemsInput{LibraryID: 1, TypeFilter: "all", OrganizingState: "organized", Limit: 20})
	if err != nil {
		t.Fatalf("browse organized items: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].ID != ready.ID {
		t.Fatalf("expected only ready item in organized browse, got %#v", result)
	}
}

func TestBrowseItemsUnorganizedFilterIncludesItemsNeedingWork(t *testing.T) {
	svc, ctx := newTestService(t)
	ready, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Ready Movie", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create ready item: %v", err)
	}
	organizing, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Organizing Movie", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create organizing item: %v", err)
	}
	conditions := []database.IngestCondition{
		{UnitKey: "catalog_item:ready", LibraryID: 1, CatalogItemID: &ready.ID, ConditionType: ingest.ConditionMaterialized, Status: ingest.ConditionStatusTrue, Reason: "linked", Message: "Media is linked", Severity: ingest.SeverityInfo},
		{UnitKey: "catalog_item:ready", LibraryID: 1, CatalogItemID: &ready.ID, ConditionType: ingest.ConditionProbed, Status: ingest.ConditionStatusPending, Reason: "probe_pending", Message: "Media probe is pending", Severity: ingest.SeverityInfo},
		{UnitKey: "catalog_item:ready", LibraryID: 1, CatalogItemID: &ready.ID, ConditionType: ingest.ConditionMetadataMatched, Status: ingest.ConditionStatusPending, Reason: "pending", Message: "Metadata matching is pending", Severity: ingest.SeverityInfo},
		{UnitKey: "catalog_item:ready", LibraryID: 1, CatalogItemID: &ready.ID, ConditionType: ingest.ConditionProjectionCurrent, Status: ingest.ConditionStatusPending, Reason: "projection_pending", Message: "Catalog projection refresh is pending", Severity: ingest.SeverityInfo},
		{UnitKey: "catalog_item:organizing", LibraryID: 1, CatalogItemID: &organizing.ID, ConditionType: ingest.ConditionMaterialized, Status: ingest.ConditionStatusPending, Reason: "awaiting_materialization", Message: "Media is waiting for catalog materialization", Severity: ingest.SeverityInfo},
	}
	if err := svc.db.WithContext(ctx).Create(&conditions).Error; err != nil {
		t.Fatalf("create ingest conditions: %v", err)
	}

	result, err := svc.BrowseItems(ctx, BrowseItemsInput{LibraryID: 1, TypeFilter: "all", OrganizingState: "unorganized", Limit: 20})
	if err != nil {
		t.Fatalf("browse unorganized items: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].ID != organizing.ID {
		t.Fatalf("expected only organizing item in unorganized browse, got %#v", result)
	}
	if !result.Items[0].Organizing {
		t.Fatalf("expected unorganized result to carry organizing flag, got %#v", result.Items[0])
	}
}

func TestGovernanceWorkspaceSurfacesSanitizedProviderDiagnostics(t *testing.T) {
	svc, ctx := newTestService(t)
	item, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Movie A", AvailabilityStatus: AvailabilityAvailable, GovernanceStatus: GovernanceNeedsReview})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	asset := database.MediaAsset{LibraryID: 1, AssetType: "main", Status: AvailabilityAvailable, ProbeStatus: "ready"}
	if err := svc.db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.AssetItem{AssetID: asset.ID, ItemID: item.ID, Role: "primary", SegmentIndex: 0}).Error; err != nil {
		t.Fatalf("link asset item: %v", err)
	}
	modifiedAt := time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC)
	file := database.InventoryFile{
		LibraryID:         1,
		StorageProvider:   "openlist",
		StoragePath:       "/Movies/Movie A.mkv",
		StableIdentityKey: "openlist:movie-a",
		HashesJSON:        `{"sha1":"abc","md5":"def"}`,
		Container:         "mkv",
		Status:            AvailabilityAvailable,
		ModifiedAt:        &modifiedAt,
	}
	if err := svc.db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: "source", PartIndex: 0}).Error; err != nil {
		t.Fatalf("link asset file: %v", err)
	}

	workspace, err := svc.GetGovernanceWorkspace(ctx, item.ID)
	if err != nil {
		t.Fatalf("get governance workspace: %v", err)
	}
	if len(workspace.Assets) != 1 || len(workspace.Assets[0].Files) != 1 {
		t.Fatalf("unexpected workspace assets: %#v", workspace.Assets)
	}
	diagnostics := workspace.Assets[0].Files[0].ProviderDiagnostics
	if diagnostics == nil || diagnostics.StorageProvider != "openlist" || len(diagnostics.AvailableHashKeys) != 2 || diagnostics.AvailableHashKeys[0] != "md5" || diagnostics.AvailableHashKeys[1] != "sha1" {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	encoded, err := json.Marshal(workspace)
	if err != nil {
		t.Fatalf("marshal workspace: %v", err)
	}
	encodedText := string(encoded)
	for _, forbidden := range []string{"sign", "mount_details", "raw_url", "secret"} {
		if strings.Contains(encodedText, forbidden) {
			t.Fatalf("governance workspace exposed sensitive metadata %q in %s", forbidden, encodedText)
		}
	}
}

func TestSeriesPlaybackTargetSelectsProgressFallbackAndNoLocal(t *testing.T) {
	svc, ctx := newTestService(t)
	series, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Show A", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", IndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	episodeOneNumber := 1
	episodeOne, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 1", IndexNumber: &episodeOneNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create episode one: %v", err)
	}
	episodeTwoNumber := 2
	episodeTwo, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 2", IndexNumber: &episodeTwoNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create episode two: %v", err)
	}
	assetOne := createPlayableCatalogAsset(t, svc, ctx, episodeOne.ID)
	assetTwo := createPlayableCatalogAsset(t, svc, ctx, episodeTwo.ID)
	lastPlayed := time.Now().UTC()
	if err := svc.db.WithContext(ctx).Create(&database.UserItemData{UserID: 7, ItemID: episodeTwo.ID, AssetID: &assetTwo.ID, PositionSeconds: 300, LastPlayedAt: &lastPlayed}).Error; err != nil {
		t.Fatalf("create progress: %v", err)
	}

	detail, err := svc.GetItemDetailForUser(ctx, series.ID, uintPtr(7))
	if err != nil {
		t.Fatalf("get series detail with progress: %v", err)
	}
	if detail.SeriesPlaybackTarget == nil || detail.SeriesPlaybackTarget.EpisodeItemID != episodeTwo.ID || detail.SeriesPlaybackTarget.AssetID == nil || *detail.SeriesPlaybackTarget.AssetID != assetTwo.ID || detail.SeriesPlaybackTarget.SelectionReason != "continue" {
		t.Fatalf("unexpected continue target: %#v", detail.SeriesPlaybackTarget)
	}

	if err := svc.db.WithContext(ctx).Where("user_id = ? AND item_id = ?", 7, episodeTwo.ID).Delete(&database.UserItemData{}).Error; err != nil {
		t.Fatalf("delete progress: %v", err)
	}
	detail, err = svc.GetItemDetailForUser(ctx, series.ID, uintPtr(7))
	if err != nil {
		t.Fatalf("get series detail without progress: %v", err)
	}
	if detail.SeriesPlaybackTarget == nil || detail.SeriesPlaybackTarget.EpisodeItemID != episodeOne.ID || detail.SeriesPlaybackTarget.AssetID == nil || *detail.SeriesPlaybackTarget.AssetID != assetOne.ID || detail.SeriesPlaybackTarget.SelectionReason != "first_available" {
		t.Fatalf("unexpected first local target: %#v", detail.SeriesPlaybackTarget)
	}

	missingSeries, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Missing Show", AvailabilityStatus: AvailabilityMissing})
	if err != nil {
		t.Fatalf("create missing series: %v", err)
	}
	missingSeason, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeason, ParentID: &missingSeries.ID, Title: "Season 1", IndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityMissing})
	if err != nil {
		t.Fatalf("create missing season: %v", err)
	}
	if _, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &missingSeason.ID, Title: "Missing Episode", IndexNumber: &episodeOneNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityMissing}); err != nil {
		t.Fatalf("create missing episode: %v", err)
	}
	detail, err = svc.GetItemDetailForUser(ctx, missingSeries.ID, uintPtr(7))
	if err != nil {
		t.Fatalf("get missing series detail: %v", err)
	}
	if detail.SeriesPlaybackTarget != nil {
		t.Fatalf("expected no playback target for missing series, got %#v", detail.SeriesPlaybackTarget)
	}
}

func TestSeriesSeasonsDefaultToLocalPlayableWhileOperationalReadsStayComplete(t *testing.T) {
	svc, ctx := newTestService(t)
	series, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Mixed Show", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonOneNumber := 1
	seasonOne, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", IndexNumber: &seasonOneNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create season one: %v", err)
	}
	seasonTwoNumber := 2
	seasonTwo, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeason, ParentID: &series.ID, Title: "Season 2", IndexNumber: &seasonTwoNumber, AvailabilityStatus: AvailabilityUnaired})
	if err != nil {
		t.Fatalf("create season two: %v", err)
	}
	episodeOneNumber := 1
	localEpisode, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &seasonOne.ID, Title: "Local Episode", IndexNumber: &episodeOneNumber, ParentIndexNumber: &seasonOneNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create local episode: %v", err)
	}
	createPlayableCatalogAsset(t, svc, ctx, localEpisode.ID)
	episodeTwoNumber := 2
	missingEpisode, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &seasonOne.ID, Title: "Missing Episode", IndexNumber: &episodeTwoNumber, ParentIndexNumber: &seasonOneNumber, AvailabilityStatus: AvailabilityMissing})
	if err != nil {
		t.Fatalf("create missing episode: %v", err)
	}
	unairedEpisode, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &seasonTwo.ID, Title: "Unaired Episode", IndexNumber: &episodeOneNumber, ParentIndexNumber: &seasonTwoNumber, AvailabilityStatus: AvailabilityUnaired})
	if err != nil {
		t.Fatalf("create unaired episode: %v", err)
	}

	seasons, err := svc.ListSeriesSeasons(ctx, series.ID)
	if err != nil {
		t.Fatalf("list series seasons: %v", err)
	}
	if len(seasons) != 1 || seasons[0].ID != seasonOne.ID || len(seasons[0].Episodes) != 1 || seasons[0].Episodes[0].ID != localEpisode.ID {
		t.Fatalf("unexpected local-only seasons: %#v", seasons)
	}
	missingEpisodes, err := svc.ListSeriesMissingEpisodes(ctx, series.ID)
	if err != nil {
		t.Fatalf("list missing episodes: %v", err)
	}
	if len(missingEpisodes) != 1 || missingEpisodes[0].ID != missingEpisode.ID {
		t.Fatalf("unexpected missing episodes: %#v", missingEpisodes)
	}
	unairedEpisodes, err := svc.ListSeriesEpisodes(ctx, series.ID, nil, AvailabilityUnaired)
	if err != nil {
		t.Fatalf("list unaired episodes: %v", err)
	}
	if len(unairedEpisodes) != 1 || unairedEpisodes[0].ID != unairedEpisode.ID {
		t.Fatalf("unexpected unaired episodes: %#v", unairedEpisodes)
	}
}

func TestGetPersonDetailReturnsProfileAndOrderedRelatedWorks(t *testing.T) {
	svc, ctx := newTestService(t)
	birthday := time.Date(1988, 5, 4, 0, 0, 0, 0, time.UTC)
	tmdbPersonID := 321
	person := database.Person{
		Name:               "Actor A",
		SortName:           "Actor A",
		AvatarURL:          "https://example.com/actor-a.jpg",
		TMDBPersonID:       &tmdbPersonID,
		IMDBID:             "nm0000321",
		Biography:          "Lead performer.",
		Birthday:           &birthday,
		PlaceOfBirth:       "Seoul",
		KnownForDepartment: "Acting",
	}
	if err := svc.db.WithContext(ctx).Create(&person).Error; err != nil {
		t.Fatalf("create person: %v", err)
	}

	missingYear := 2025
	availableOlderYear := 2021
	availableNewerYear := 2024
	missing, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Missing Movie", Path: "/movies/missing.mkv", SortKey: "Missing Movie", Year: &missingYear, AvailabilityStatus: AvailabilityMissing, GovernanceStatus: GovernanceMatched})
	if err != nil {
		t.Fatalf("create missing item: %v", err)
	}
	availableOlder, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Older Movie", Path: "/movies/older.mkv", SortKey: "Older Movie", Year: &availableOlderYear, AvailabilityStatus: AvailabilityAvailable, GovernanceStatus: GovernanceMatched})
	if err != nil {
		t.Fatalf("create older item: %v", err)
	}
	availableNewer, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Newer Movie", Path: "/movies/newer.mkv", SortKey: "Newer Movie", Year: &availableNewerYear, AvailabilityStatus: AvailabilityAvailable, GovernanceStatus: GovernanceMatched})
	if err != nil {
		t.Fatalf("create newer item: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create([]database.ItemImage{{ItemID: availableNewer.ID, ImageType: "backdrop", URL: "https://example.com/newer-backdrop.jpg", IsSelected: true}}).Error; err != nil {
		t.Fatalf("create related item image: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create([]database.ItemPerson{{ItemID: missing.ID, PersonID: person.ID, Role: "cast", Character: "Guest", SortOrder: 0}, {ItemID: availableOlder.ID, PersonID: person.ID, Role: "cast", Character: "Support", SortOrder: 1}, {ItemID: availableNewer.ID, PersonID: person.ID, Role: "cast", Character: "Lead", SortOrder: 1}}).Error; err != nil {
		t.Fatalf("link related items: %v", err)
	}

	detail, err := svc.GetPersonDetail(ctx, person.ID)
	if err != nil {
		t.Fatalf("get person detail: %v", err)
	}
	if detail.ID != person.ID || detail.Name != person.Name || detail.Biography != person.Biography || detail.Birthday == nil || !detail.Birthday.Equal(birthday) {
		t.Fatalf("unexpected person detail profile: %#v", detail)
	}
	if len(detail.ExternalIdentities) != 2 || detail.ExternalIdentities[0].Provider != "tmdb" || detail.ExternalIdentities[0].ProviderType != "person" || detail.ExternalIdentities[0].ExternalID != "321" || detail.ExternalIdentities[1].Provider != "imdb" {
		t.Fatalf("unexpected external identities: %#v", detail.ExternalIdentities)
	}
	if len(detail.RelatedItems) != 3 || detail.RelatedItems[0].ID != availableNewer.ID || detail.RelatedItems[1].ID != availableOlder.ID || detail.RelatedItems[2].ID != missing.ID {
		t.Fatalf("unexpected related item order: %#v", detail.RelatedItems)
	}
}

func TestListRecentlyAddedIncludesActiveAndSyncingLibraries(t *testing.T) {
	svc, ctx := newTestService(t)
	activeLibrary := database.Library{Name: "Active", Type: "movies", MediaSourceID: 1, RootPath: "/active", Status: "active"}
	syncingLibrary := database.Library{Name: "Syncing", Type: "movies", MediaSourceID: 1, RootPath: "/syncing", Status: "syncing"}
	inactiveLibrary := database.Library{Name: "Inactive", Type: "shows", MediaSourceID: 1, RootPath: "/inactive", Status: "deleted"}
	if err := svc.db.WithContext(ctx).Create([]*database.Library{&activeLibrary, &syncingLibrary, &inactiveLibrary}).Error; err != nil {
		t.Fatalf("create libraries: %v", err)
	}
	activeItem, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: activeLibrary.ID, Type: ItemTypeMovie, Title: "Active Movie", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create active item: %v", err)
	}
	syncingItem, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: syncingLibrary.ID, Type: ItemTypeMovie, Title: "Syncing Movie", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create syncing item: %v", err)
	}
	if _, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: inactiveLibrary.ID, Type: ItemTypeSeries, Title: "Inactive Show", AvailabilityStatus: AvailabilityAvailable}); err != nil {
		t.Fatalf("create inactive item: %v", err)
	}
	if _, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 999, Type: ItemTypeSeries, Title: "Orphan Show", AvailabilityStatus: AvailabilityAvailable}); err != nil {
		t.Fatalf("create orphan item: %v", err)
	}

	items, err := svc.ListRecentlyAdded(ctx, 12)
	if err != nil {
		t.Fatalf("list recently added: %v", err)
	}
	if len(items) != 2 || items[0].ID != syncingItem.ID || items[1].ID != activeItem.ID {
		t.Fatalf("unexpected recently added items: %#v", items)
	}
}

func TestListLatestByLibraryUsesActiveLibrariesAndProjectionLatestOrder(t *testing.T) {
	svc, ctx := newTestService(t)
	alphaLibrary := database.Library{Name: "Alpha", Type: "movies", MediaSourceID: 1, RootPath: "/alpha", Status: "active"}
	betaLibrary := database.Library{Name: "Beta", Type: "movies", MediaSourceID: 1, RootPath: "/beta", Status: "syncing"}
	inactiveLibrary := database.Library{Name: "Gamma", Type: "movies", MediaSourceID: 1, RootPath: "/gamma", Status: "deleted"}
	if err := svc.db.WithContext(ctx).Create([]*database.Library{&alphaLibrary, &betaLibrary, &inactiveLibrary}).Error; err != nil {
		t.Fatalf("create libraries: %v", err)
	}
	alphaOlder, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: alphaLibrary.ID, Type: ItemTypeMovie, Title: "Alpha Older", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create alpha older: %v", err)
	}
	alphaNewer, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: alphaLibrary.ID, Type: ItemTypeMovie, Title: "Alpha Newer", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create alpha newer: %v", err)
	}
	betaItem, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: betaLibrary.ID, Type: ItemTypeMovie, Title: "Beta Movie", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create beta item: %v", err)
	}
	if _, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: inactiveLibrary.ID, Type: ItemTypeMovie, Title: "Gamma Movie", AvailabilityStatus: AvailabilityAvailable}); err != nil {
		t.Fatalf("create inactive item: %v", err)
	}
	if alphaOlder.ID == 0 {
		t.Fatalf("expected seeded alpha older item")
	}

	sections, err := svc.ListLatestByLibrary(ctx, 1)
	if err != nil {
		t.Fatalf("list latest by library: %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("expected two active/syncing library sections, got %#v", sections)
	}
	if sections[0].LibraryID != alphaLibrary.ID || sections[0].LibraryName != alphaLibrary.Name || len(sections[0].Items) != 1 || sections[0].Items[0].ID != alphaNewer.ID {
		t.Fatalf("unexpected alpha section: %#v", sections[0])
	}
	if sections[1].LibraryID != betaLibrary.ID || sections[1].LibraryName != betaLibrary.Name || len(sections[1].Items) != 1 || sections[1].Items[0].ID != betaItem.ID {
		t.Fatalf("unexpected beta section: %#v", sections[1])
	}
}

func TestGetEpisodeItemDetailIncludesContextShelfProgressAndStreams(t *testing.T) {
	svc, ctx := newTestService(t)
	series, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Show A", Path: "/shows/ShowA", SortKey: "Show A", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", Path: "/shows/ShowA/Season 1", SortKey: "Show A S01", IndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	episodeOneNumber := 1
	episodeOne, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 1", Path: "/shows/ShowA/Season 1/ShowA.S01E01.mkv", SortKey: "Show A S01E01", IndexNumber: &episodeOneNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create episode one: %v", err)
	}
	episodeTwoNumber := 2
	episodeTwo, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 2", Path: "/shows/ShowA/Season 1/ShowA.S01E02.mkv", SortKey: "Show A S01E02", IndexNumber: &episodeTwoNumber, ParentIndexNumber: &seasonNumber, RuntimeSeconds: intPtr(1800), AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create episode two: %v", err)
	}
	episodeThreeNumber := 3
	missingSourceEpisode, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 3", Path: "/shows/ShowA/Season 1/ShowA.S01E03.mkv", SortKey: "Show A S01E03", IndexNumber: &episodeThreeNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create missing-source episode: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create([]database.ItemImage{
		{ItemID: series.ID, ImageType: "backdrop", URL: "https://example.com/series.jpg", IsSelected: true},
		{ItemID: season.ID, ImageType: "poster", URL: "https://example.com/season.jpg", IsSelected: true},
		{ItemID: episodeOne.ID, ImageType: "still", URL: "https://example.com/e1.jpg", IsSelected: true},
		{ItemID: episodeTwo.ID, ImageType: "still", URL: "https://example.com/e2.jpg", IsSelected: true},
		{ItemID: missingSourceEpisode.ID, ImageType: "still", URL: "https://example.com/e3.jpg", IsSelected: true},
	}).Error; err != nil {
		t.Fatalf("create images: %v", err)
	}

	episodeOneAsset := createPlayableCatalogAsset(t, svc, ctx, episodeOne.ID)
	episodeOneAsset.DisplayName = "720p"
	if err := svc.db.WithContext(ctx).Save(&episodeOneAsset).Error; err != nil {
		t.Fatalf("update episode one asset: %v", err)
	}

	asset := database.MediaAsset{LibraryID: 1, AssetType: "main", DisplayName: "1080p", Status: AvailabilityAvailable, ProbeStatus: "ready"}
	if err := svc.db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.AssetItem{AssetID: asset.ID, ItemID: episodeTwo.ID, Role: "primary", SegmentIndex: 0}).Error; err != nil {
		t.Fatalf("link asset: %v", err)
	}
	file := database.InventoryFile{LibraryID: 1, StorageProvider: "local", StoragePath: "/shows/ShowA/Season 1/ShowA.S01E02.mkv", SizeBytes: 123456, Container: "mkv", Status: AvailabilityAvailable}
	if err := svc.db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: "source", PartIndex: 0}).Error; err != nil {
		t.Fatalf("link asset file: %v", err)
	}
	width := 1920
	height := 1080
	level := 41
	bitDepth := 10
	referenceFrames := 4
	channels := 6
	bitrate := int64(640000)
	videoBitrate := int64(4200000)
	audioBitDepth := 24
	audioSampleRate := 48000
	if err := svc.db.WithContext(ctx).Create([]database.MediaStream{
		{FileID: file.ID, StreamIndex: 0, StreamType: "video", Codec: "h264", Profile: "High", Level: &level, Width: &width, Height: &height, AvgFrameRate: "24000/1001", RFrameRate: "24000/1001", FieldOrder: "progressive", ColorSpace: "bt709", BitDepth: &bitDepth, PixelFormat: "yuv420p10le", ReferenceFrames: &referenceFrames, BitRate: &videoBitrate},
		{FileID: file.ID, StreamIndex: 1, StreamType: "audio", Codec: "flac", Language: "jpn", Title: "Japanese", Channels: &channels, ChannelLayout: "5.1(side)", SampleRate: &audioSampleRate, BitDepth: &audioBitDepth, BitRate: &bitrate, DispositionJSON: `{"default":true}`},
		{FileID: file.ID, StreamIndex: 2, StreamType: "subtitle", Codec: "ass", Language: "zho", Title: "Chinese Traditional", DispositionJSON: `{"default":true,"forced":false,"external":true,"hearing_impaired":false}`},
	}).Error; err != nil {
		t.Fatalf("create streams: %v", err)
	}
	playedPercentage := 55.5
	if err := svc.db.WithContext(ctx).Create(&database.UserItemData{UserID: 7, ItemID: episodeOne.ID, PositionSeconds: 600, PlayedPercentage: &playedPercentage}).Error; err != nil {
		t.Fatalf("create progress: %v", err)
	}

	detail, err := svc.GetItemDetailForUser(ctx, episodeTwo.ID, uintPtr(7))
	if err != nil {
		t.Fatalf("get episode detail: %v", err)
	}
	if detail.EpisodeContext == nil || detail.EpisodeContext.Series == nil || detail.EpisodeContext.Series.ID != series.ID || detail.EpisodeContext.Season == nil || detail.EpisodeContext.Season.ID != season.ID {
		t.Fatalf("unexpected episode context: %#v", detail.EpisodeContext)
	}
	if detail.EpisodeContext.IncompleteHierarchy {
		t.Fatalf("expected complete hierarchy context: %#v", detail.EpisodeContext)
	}
	if detail.EpisodeContext.EpisodeNumber == nil || *detail.EpisodeContext.EpisodeNumber != episodeTwoNumber || len(detail.EpisodeContext.Series.SelectedImages) != 1 || len(detail.EpisodeContext.Season.SelectedImages) != 1 {
		t.Fatalf("unexpected episode numbering or parent images: %#v", detail.EpisodeContext)
	}
	if len(detail.SameSeasonEpisodes) != 2 || detail.SameSeasonEpisodes[0].ID != episodeOne.ID || detail.SameSeasonEpisodes[0].Progress == nil || detail.SameSeasonEpisodes[0].Progress.PositionSeconds != 600 {
		t.Fatalf("unexpected same-season shelf progress: %#v", detail.SameSeasonEpisodes)
	}
	if !detail.SameSeasonEpisodes[1].Current || detail.SameSeasonEpisodes[1].Label != "S1:E2" || detail.SameSeasonEpisodes[1].Progress != nil {
		t.Fatalf("unexpected current episode shelf state: %#v", detail.SameSeasonEpisodes[1])
	}
	for _, shelfItem := range detail.SameSeasonEpisodes {
		if shelfItem.ID == missingSourceEpisode.ID {
			t.Fatalf("expected missing-source episode to be hidden from same-season shelf: %#v", detail.SameSeasonEpisodes)
		}
	}
	if len(detail.Assets) != 1 || len(detail.Assets[0].Files) != 1 || detail.Assets[0].Files[0].FileID != file.ID || detail.Assets[0].Files[0].Container != "mkv" {
		t.Fatalf("unexpected asset file summaries: %#v", detail.Assets)
	}
	if len(detail.Assets[0].Streams) != 3 || detail.Assets[0].Streams[0].Width == nil || *detail.Assets[0].Streams[0].Width != width || !detail.Assets[0].Streams[1].Default {
		t.Fatalf("unexpected asset stream summaries: %#v", detail.Assets[0].Streams)
	}
	videoStream := detail.Assets[0].Streams[0]
	if videoStream.Profile != "High" || videoStream.Level == nil || *videoStream.Level != level || videoStream.AvgFrameRate != "24000/1001" || videoStream.BitDepth == nil || *videoStream.BitDepth != bitDepth || videoStream.ReferenceFrames == nil || *videoStream.ReferenceFrames != referenceFrames || videoStream.BitRate == nil || *videoStream.BitRate != videoBitrate {
		t.Fatalf("unexpected detailed video stream summary: %#v", videoStream)
	}
	if videoStream.FieldOrder != "progressive" || videoStream.ColorSpace != "bt709" || videoStream.PixelFormat != "yuv420p10le" {
		t.Fatalf("unexpected detailed video stream display fields: %#v", videoStream)
	}
	audioStream := detail.Assets[0].Streams[1]
	if audioStream.ChannelLayout != "5.1(side)" || audioStream.SampleRate == nil || *audioStream.SampleRate != audioSampleRate || audioStream.BitDepth == nil || *audioStream.BitDepth != audioBitDepth || audioStream.Codec != "flac" {
		t.Fatalf("unexpected detailed audio stream summary: %#v", audioStream)
	}
	subtitleStream := detail.Assets[0].Streams[2]
	if subtitleStream.Codec != "ass" || subtitleStream.Title != "Chinese Traditional" || !subtitleStream.Default || subtitleStream.Forced || !subtitleStream.External || subtitleStream.HearingImpaired {
		t.Fatalf("unexpected detailed subtitle stream summary: %#v", subtitleStream)
	}
}

func TestGetEpisodeItemDetailAllowsIncompleteHierarchy(t *testing.T) {
	svc, ctx := newTestService(t)
	seasonNumber := 1
	episodeNumber := 2
	episode, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, Title: "Loose Episode", Path: "/shows/Loose.S01E02.mkv", SortKey: "Loose S01E02", IndexNumber: &episodeNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create loose episode: %v", err)
	}

	detail, err := svc.GetItemDetail(ctx, episode.ID)
	if err != nil {
		t.Fatalf("get episode detail: %v", err)
	}
	if detail.EpisodeContext == nil || !detail.EpisodeContext.IncompleteHierarchy || detail.EpisodeContext.Series != nil || detail.EpisodeContext.Season != nil {
		t.Fatalf("expected incomplete hierarchy context, got %#v", detail.EpisodeContext)
	}
	if len(detail.SameSeasonEpisodes) != 0 {
		t.Fatalf("expected no same-season shelf for loose episode, got %#v", detail.SameSeasonEpisodes)
	}
}

func TestGetItemDetailIncludesExternalSidecarSubtitleSummary(t *testing.T) {
	svc, ctx := newTestService(t)
	movie, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Movie A", Path: "/movies/Movie A.mkv", SortKey: "Movie A", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}
	asset := database.MediaAsset{LibraryID: 1, AssetType: "main", Status: AvailabilityAvailable, ProbeStatus: "ready"}
	if err := svc.db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.AssetItem{AssetID: asset.ID, ItemID: movie.ID, Role: "primary", SegmentIndex: 0}).Error; err != nil {
		t.Fatalf("link asset item: %v", err)
	}
	videoFile := database.InventoryFile{LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/Movie A.mkv", Container: "mkv", Status: AvailabilityAvailable}
	subtitleFile := database.InventoryFile{LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/Movie A.srt", Container: "srt", Status: AvailabilityAvailable}
	if err := svc.db.WithContext(ctx).Create([]*database.InventoryFile{&videoFile, &subtitleFile}).Error; err != nil {
		t.Fatalf("create inventory files: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create([]database.AssetFile{{AssetID: asset.ID, FileID: videoFile.ID, Role: "source"}, {AssetID: asset.ID, FileID: subtitleFile.ID, Role: "subtitle"}}).Error; err != nil {
		t.Fatalf("link asset files: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.MediaStream{FileID: subtitleFile.ID, StreamIndex: 0, StreamType: "subtitle", Codec: "srt", Title: "Movie A", DispositionJSON: `{"external":true,"managed_by":"scanner"}`}).Error; err != nil {
		t.Fatalf("create subtitle stream: %v", err)
	}

	detail, err := svc.GetItemDetail(ctx, movie.ID)
	if err != nil {
		t.Fatalf("get item detail: %v", err)
	}
	if len(detail.Assets) != 1 || len(detail.Assets[0].Files) != 2 || len(detail.Assets[0].Streams) != 1 {
		t.Fatalf("unexpected asset detail: %#v", detail.Assets)
	}
	stream := detail.Assets[0].Streams[0]
	if stream.FileID != subtitleFile.ID || !stream.External || stream.Available == nil || !*stream.Available || !strings.HasPrefix(stream.URL, "/api/v1/inventory-files/") || strings.Contains(stream.URL, "sign") {
		t.Fatalf("expected safe available external subtitle summary, got %#v", stream)
	}
}

func TestUserItemFavoritesAndContinueWatching(t *testing.T) {
	svc, ctx := newTestService(t)
	movie, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Favorite Movie", Path: "/movies/favorite.mkv", SortKey: "Favorite Movie", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}
	show, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Watching Show", Path: "/shows/watching", SortKey: "Watching Show", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create show: %v", err)
	}
	seasonNumber := 1
	season, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeason, ParentID: &show.ID, Title: "Season 1", Path: "/shows/watching/season-1", SortKey: "Watching Show S01", IndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	episodeNumber := 2
	episode, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 2", Path: "/shows/watching/season-1/e02.mkv", SortKey: "Watching Show S01E02", IndexNumber: &episodeNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.ItemImage{ItemID: show.ID, ImageType: "poster", URL: "https://example.com/show-poster.jpg", IsSelected: true}).Error; err != nil {
		t.Fatalf("create show poster: %v", err)
	}

	const userID uint = 7
	favorite, err := svc.SetFavorite(ctx, userID, movie.ID, true)
	if err != nil {
		t.Fatalf("set favorite: %v", err)
	}
	if !favorite.Favorite || favorite.Item.ID != movie.ID {
		t.Fatalf("unexpected favorite response: %#v", favorite)
	}

	favorites, err := svc.ListFavorites(ctx, userID, 10)
	if err != nil {
		t.Fatalf("list favorites: %v", err)
	}
	if len(favorites) != 1 || favorites[0].Item.ID != movie.ID {
		t.Fatalf("unexpected favorites: %#v", favorites)
	}

	lastPlayed := time.Now().UTC()
	if err := svc.db.WithContext(ctx).Create(&database.UserItemData{UserID: userID, ItemID: episode.ID, PositionSeconds: 120, ProgressFrameURL: "/api/v1/me/progress-frames/1/default", LastPlayedAt: &lastPlayed}).Error; err != nil {
		t.Fatalf("create progress: %v", err)
	}
	continueWatching, err := svc.ListContinueWatching(ctx, userID, 10)
	if err != nil {
		t.Fatalf("list continue watching: %v", err)
	}
	if len(continueWatching) != 1 || continueWatching[0].Item.ID != episode.ID || continueWatching[0].PositionSeconds != 120 {
		t.Fatalf("unexpected continue watching: %#v", continueWatching)
	}
	if continueWatching[0].PlayItem == nil || continueWatching[0].PlayItem.ID != episode.ID {
		t.Fatalf("expected episode play item, got %#v", continueWatching[0].PlayItem)
	}
	if continueWatching[0].ProgressFrameURL != "/api/v1/me/progress-frames/1/default" {
		t.Fatalf("expected progress frame url, got %#v", continueWatching[0])
	}
	if continueWatching[0].DisplayItem == nil || continueWatching[0].DisplayItem.ID != show.ID || continueWatching[0].DisplayItem.Type != ItemTypeSeries {
		t.Fatalf("expected series display item, got %#v", continueWatching[0].DisplayItem)
	}
	if len(continueWatching[0].DisplayItem.SelectedImages) != 1 || continueWatching[0].DisplayItem.SelectedImages[0].URL != "https://example.com/show-poster.jpg" {
		t.Fatalf("expected series poster on display item, got %#v", continueWatching[0].DisplayItem.SelectedImages)
	}
	if err := svc.db.WithContext(ctx).Model(&database.CatalogItem{}).Where("id = ?", episode.ID).Update("availability_status", AvailabilityMissing).Error; err != nil {
		t.Fatalf("mark episode missing: %v", err)
	}
	continueWatching, err = svc.ListContinueWatching(ctx, userID, 10)
	if err != nil {
		t.Fatalf("list continue watching after missing: %v", err)
	}
	if len(continueWatching) != 0 {
		t.Fatalf("expected missing item to be hidden from continue watching, got %#v", continueWatching)
	}

	if _, err := svc.SetFavorite(ctx, userID, movie.ID, false); err != nil {
		t.Fatalf("remove favorite: %v", err)
	}
	favorites, err = svc.ListFavorites(ctx, userID, 10)
	if err != nil {
		t.Fatalf("list favorites after remove: %v", err)
	}
	if len(favorites) != 0 {
		t.Fatalf("expected no favorites, got %#v", favorites)
	}
}

func intPtr(value int) *int {
	return &value
}

func uintPtr(value uint) *uint {
	return &value
}

func createPlayableCatalogAsset(t *testing.T, svc *Service, ctx context.Context, itemID uint) database.MediaAsset {
	t.Helper()
	asset := database.MediaAsset{LibraryID: 1, AssetType: "main", Status: AvailabilityAvailable, ProbeStatus: "ready"}
	if err := svc.db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.AssetItem{AssetID: asset.ID, ItemID: itemID, Role: "primary", SegmentIndex: 0}).Error; err != nil {
		t.Fatalf("link asset item: %v", err)
	}
	file := database.InventoryFile{LibraryID: 1, StorageProvider: "local", StoragePath: "/test/playable-" + fmt.Sprint(itemID) + ".mkv", Container: "mkv", Status: AvailabilityAvailable}
	if err := svc.db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: "source", PartIndex: 0}).Error; err != nil {
		t.Fatalf("link asset file: %v", err)
	}
	return asset
}
