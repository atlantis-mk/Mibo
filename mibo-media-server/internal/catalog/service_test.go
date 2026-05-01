package catalog

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestCreateItemBuildsSeriesHierarchy(t *testing.T) {
	svc, ctx := newTestService(t)

	series, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "The Expanse", AvailabilityStatus: AvailabilityNoLocalMedia})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	if series.RootID == nil || *series.RootID != series.ID {
		t.Fatalf("expected series root id to point at itself, got %#v", series.RootID)
	}

	seasonIndex := 1
	season, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", IndexNumber: &seasonIndex})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	if season.RootID == nil || *season.RootID != series.ID {
		t.Fatalf("expected season root id %d, got %#v", series.ID, season.RootID)
	}

	episodeIndex := 1
	episode, err := svc.CreateItem(ctx, CreateItemInput{
		LibraryID:          1,
		Type:               ItemTypeEpisode,
		ParentID:           &season.ID,
		Title:              "Dulcinea",
		IndexNumber:        &episodeIndex,
		ParentIndexNumber:  &seasonIndex,
		AvailabilityStatus: AvailabilityMissing,
		GovernanceStatus:   GovernanceMatched,
	})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	if episode.RootID == nil || *episode.RootID != series.ID {
		t.Fatalf("expected episode root id %d, got %#v", series.ID, episode.RootID)
	}
	if episode.AvailabilityStatus != AvailabilityMissing {
		t.Fatalf("expected missing episode row without local media, got %q", episode.AvailabilityStatus)
	}

	children, err := svc.ListChildren(ctx, season.ID)
	if err != nil {
		t.Fatalf("list children: %v", err)
	}
	if len(children) != 1 || children[0].ID != episode.ID {
		t.Fatalf("unexpected season children: %#v", children)
	}

	var seriesRollup database.ItemRollup
	if err := svc.db.WithContext(ctx).First(&seriesRollup, "item_id = ?", series.ID).Error; err != nil {
		t.Fatalf("load series rollup: %v", err)
	}
	if seriesRollup.ChildCount != 2 {
		t.Fatalf("expected series rollup child count 2, got %#v", seriesRollup)
	}
}

func TestApplyFieldRespectsLockedCanonicalValue(t *testing.T) {
	svc, ctx := newTestService(t)
	item, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Original Title"})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	_, applied, err := svc.ApplyField(ctx, ApplyFieldInput{ItemID: item.ID, FieldKey: "title", Value: "Manual Title", Lock: true, LockReason: "user edit"})
	if err != nil {
		t.Fatalf("apply manual title: %v", err)
	}
	if !applied {
		t.Fatalf("expected manual field to apply")
	}

	_, applied, err = svc.ApplyField(ctx, ApplyFieldInput{ItemID: item.ID, FieldKey: "title", Value: "Provider Title"})
	if err != nil {
		t.Fatalf("apply provider title: %v", err)
	}
	if applied {
		t.Fatalf("expected locked field to reject provider overwrite")
	}

	var reloaded database.CatalogItem
	if err := svc.db.WithContext(ctx).First(&reloaded, item.ID).Error; err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if reloaded.Title != "Manual Title" {
		t.Fatalf("expected locked title to remain, got %q", reloaded.Title)
	}

	var doc database.CatalogSearchDocument
	if err := svc.db.WithContext(ctx).First(&doc, "item_id = ?", item.ID).Error; err != nil {
		t.Fatalf("load search document: %v", err)
	}
	if doc.Title != "Manual Title" {
		t.Fatalf("expected refreshed search document title %q, got %q", "Manual Title", doc.Title)
	}
}

func TestSetExternalIDUpsertsProviderIdentity(t *testing.T) {
	svc, ctx := newTestService(t)
	first, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "First"})
	if err != nil {
		t.Fatalf("create first item: %v", err)
	}
	second, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Second"})
	if err != nil {
		t.Fatalf("create second item: %v", err)
	}

	if _, err := svc.SetExternalID(ctx, ExternalIDInput{ItemID: first.ID, Provider: "tmdb", ProviderType: "tv", ExternalID: "123", IsPrimary: true}); err != nil {
		t.Fatalf("set first external id: %v", err)
	}
	if _, err := svc.SetExternalID(ctx, ExternalIDInput{ItemID: second.ID, Provider: "tmdb", ProviderType: "tv", ExternalID: "123", IsPrimary: true}); err != nil {
		t.Fatalf("move external id: %v", err)
	}

	var count int64
	if err := svc.db.WithContext(ctx).Model(&database.CatalogExternalID{}).Where("provider = ? AND provider_type = ? AND external_id = ?", "tmdb", "tv", "123").Count(&count).Error; err != nil {
		t.Fatalf("count external ids: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one canonical provider identity, got %d", count)
	}
	var externalID database.CatalogExternalID
	if err := svc.db.WithContext(ctx).Where("provider = ? AND provider_type = ? AND external_id = ?", "tmdb", "tv", "123").First(&externalID).Error; err != nil {
		t.Fatalf("load external id: %v", err)
	}
	if externalID.ItemID != second.ID {
		t.Fatalf("expected provider id to point at second item %d, got %d", second.ID, externalID.ItemID)
	}

	var doc database.CatalogSearchDocument
	if err := svc.db.WithContext(ctx).First(&doc, "item_id = ?", second.ID).Error; err != nil {
		t.Fatalf("load refreshed search document: %v", err)
	}
	if !strings.Contains(doc.ProviderIDsText, "tmdb:tv:123") {
		t.Fatalf("expected provider ids text to include canonical external id, got %q", doc.ProviderIDsText)
	}
}

func TestCatalogIdentityUpsertAndReconcile(t *testing.T) {
	svc, ctx := newTestService(t)
	first, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "First", Path: "/tv/Show"})
	if err != nil {
		t.Fatalf("create first item: %v", err)
	}
	second, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Second", Path: "/tv/Show Renamed"})
	if err != nil {
		t.Fatalf("create second item: %v", err)
	}

	identityKey := "library:1:series:/tv/Show"
	if _, err := svc.SetIdentity(ctx, IdentityInput{ItemID: first.ID, Provider: IdentityProviderScanner, IdentityType: IdentityTypeSeries, IdentityKey: identityKey, SourcePath: "/tv/Show"}); err != nil {
		t.Fatalf("set first identity: %v", err)
	}
	if _, err := svc.SetIdentity(ctx, IdentityInput{ItemID: second.ID, Provider: IdentityProviderScanner, IdentityType: IdentityTypeSeries, IdentityKey: identityKey, SourcePath: "/tv/Show Renamed"}); err != nil {
		t.Fatalf("move identity: %v", err)
	}

	item, identity, found, err := svc.ReconcileItemByIdentity(ctx, IdentityInput{ItemID: second.ID, Provider: IdentityProviderScanner, IdentityType: IdentityTypeSeries, IdentityKey: identityKey})
	if err != nil {
		t.Fatalf("reconcile identity: %v", err)
	}
	if !found {
		t.Fatal("expected identity to reconcile")
	}
	if item.ID != second.ID || identity.ItemID != second.ID {
		t.Fatalf("expected identity to point at second item %d, got item=%d identity=%#v", second.ID, item.ID, identity)
	}

	var count int64
	if err := svc.db.WithContext(ctx).Model(&database.CatalogIdentity{}).Where("provider = ? AND identity_type = ? AND identity_key = ?", IdentityProviderScanner, IdentityTypeSeries, identityKey).Count(&count).Error; err != nil {
		t.Fatalf("count identities: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one scanner identity row, got %d", count)
	}
}

func TestBackfillScannerIdentitiesUsesStableItemPath(t *testing.T) {
	svc, ctx := newTestService(t)
	movie, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 7, Type: ItemTypeMovie, Title: "Movie", Path: "/movies/Movie (2024)"})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}
	series, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 7, Type: ItemTypeSeries, Title: "Show", Path: "/tv/Show"})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}

	count, err := svc.BackfillScannerIdentities(ctx)
	if err != nil {
		t.Fatalf("backfill scanner identities: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected two backfilled identities, got %d", count)
	}

	for _, item := range []database.CatalogItem{movie, series} {
		key, ok := ScannerIdentityKeyForItem(item)
		if !ok {
			t.Fatalf("expected scanner identity key for %#v", item)
		}
		resolved, _, err := svc.FindItemByIdentity(ctx, IdentityProviderScanner, item.Type, key)
		if err != nil {
			t.Fatalf("find item by identity %q: %v", key, err)
		}
		if resolved.ID != item.ID {
			t.Fatalf("expected identity %q to resolve item %d, got %d", key, item.ID, resolved.ID)
		}
	}

	if count, err = svc.BackfillScannerIdentities(ctx); err != nil {
		t.Fatalf("repeat backfill scanner identities: %v", err)
	} else if count != 2 {
		t.Fatalf("expected repeat backfill to process two identities without duplicates, got %d", count)
	}
}

func TestApplyFieldSupportsManualAndLockedGovernanceOverrides(t *testing.T) {
	svc, ctx := newTestService(t)
	item, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Movie A"})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	if _, applied, err := svc.ApplyField(ctx, ApplyFieldInput{ItemID: item.ID, FieldKey: "governance_status", Value: GovernanceManual}); err != nil {
		t.Fatalf("apply manual governance status: %v", err)
	} else if !applied {
		t.Fatal("expected manual governance status to apply")
	}
	if _, applied, err := svc.ApplyField(ctx, ApplyFieldInput{ItemID: item.ID, FieldKey: "governance_status", Value: GovernanceLocked, Lock: true, LockReason: "review approved"}); err != nil {
		t.Fatalf("apply locked governance status: %v", err)
	} else if !applied {
		t.Fatal("expected locked governance status to apply")
	}

	var stored database.CatalogItem
	if err := svc.db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if stored.GovernanceStatus != GovernanceLocked {
		t.Fatalf("expected locked governance status, got %#v", stored)
	}

	var state database.MetadataFieldState
	if err := svc.db.WithContext(ctx).Where("item_id = ? AND field_key = ?", item.ID, "governance_status").First(&state).Error; err != nil {
		t.Fatalf("load governance field state: %v", err)
	}
	if !state.IsLocked || state.LockReason != "review approved" {
		t.Fatalf("expected locked governance field state, got %#v", state)
	}
}

func TestCorrectEpisodeNumberingMovesEpisodeWithinSeriesAndDetectsConflict(t *testing.T) {
	svc, ctx := newTestService(t)
	series, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Show A", Path: "/shows/ShowA"})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonOneNumber := 1
	seasonOne, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", IndexNumber: &seasonOneNumber})
	if err != nil {
		t.Fatalf("create season one: %v", err)
	}
	episodeOneNumber := 1
	episode, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &seasonOne.ID, Title: "Episode 1", IndexNumber: &episodeOneNumber, ParentIndexNumber: &seasonOneNumber, GovernanceStatus: GovernanceLocked})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	seasonTwoNumber := 2
	conflict, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeason, ParentID: &series.ID, Title: "Season 2", IndexNumber: &seasonTwoNumber})
	if err != nil {
		t.Fatalf("create season two: %v", err)
	}
	conflictEpisodeNumber := 3
	if _, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &conflict.ID, Title: "Episode 3", IndexNumber: &conflictEpisodeNumber, ParentIndexNumber: &seasonTwoNumber}); err != nil {
		t.Fatalf("create conflict episode: %v", err)
	}

	if _, err := svc.CorrectEpisodeNumbering(ctx, CorrectEpisodeNumberingInput{EpisodeID: episode.ID, SeasonNumber: 2, EpisodeNumber: 3}); err == nil || !strings.Contains(err.Error(), "already occupied") {
		t.Fatalf("expected occupied slot conflict, got %v", err)
	}

	updated, err := svc.CorrectEpisodeNumbering(ctx, CorrectEpisodeNumberingInput{EpisodeID: episode.ID, SeasonNumber: 2, EpisodeNumber: 4})
	if err != nil {
		t.Fatalf("correct episode numbering: %v", err)
	}
	if updated.ParentID == nil || *updated.ParentID != conflict.ID || updated.ParentIndexNumber == nil || *updated.ParentIndexNumber != 2 || updated.IndexNumber == nil || *updated.IndexNumber != 4 {
		t.Fatalf("unexpected corrected episode: %#v", updated)
	}
	if updated.GovernanceStatus != GovernanceLocked {
		t.Fatalf("expected unrelated governance status to be preserved, got %#v", updated)
	}
}

func newTestService(t *testing.T) (*Service, context.Context) {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	return NewService(db), context.Background()
}
