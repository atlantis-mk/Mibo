package inventory_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"gorm.io/gorm"
)

func TestAssetLinksSupportMultiEpisodeFiles(t *testing.T) {
	db, ctx := newTestDB(t)
	catalogSvc := catalog.NewService(db)
	inventorySvc := inventory.NewService(db)

	series, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeSeries, Title: "Firefly"})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonIndex := 1
	season, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", IndexNumber: &seasonIndex})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	firstIndex := 1
	first, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Title: "Serenity", IndexNumber: &firstIndex})
	if err != nil {
		t.Fatalf("create first episode: %v", err)
	}
	secondIndex := 2
	second, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Title: "The Train Job", IndexNumber: &secondIndex})
	if err != nil {
		t.Fatalf("create second episode: %v", err)
	}

	file, err := inventorySvc.UpsertFile(ctx, inventory.UpsertFileInput{LibraryID: 1, StorageProvider: "local", StoragePath: "/tv/Firefly/S01E01-E02.mkv", StableIdentityKey: "local:/tv/Firefly/S01E01-E02.mkv", SizeBytes: 1024})
	if err != nil {
		t.Fatalf("upsert file: %v", err)
	}
	asset, err := inventorySvc.CreateAsset(ctx, inventory.CreateAssetInput{LibraryID: 1, AssetType: inventory.AssetTypeMain, DisplayName: "S01E01-E02"})
	if err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if _, err := inventorySvc.LinkAssetToFile(ctx, inventory.LinkAssetFileInput{AssetID: asset.ID, FileID: file.ID, Role: inventory.FileRoleSource}); err != nil {
		t.Fatalf("link asset file: %v", err)
	}
	if _, err := inventorySvc.LinkAssetToItem(ctx, inventory.LinkAssetItemInput{AssetID: asset.ID, ItemID: first.ID, Role: inventory.AssetItemRoleMultiEpisodePart, SegmentIndex: 1}); err != nil {
		t.Fatalf("link first episode: %v", err)
	}
	if _, err := inventorySvc.LinkAssetToItem(ctx, inventory.LinkAssetItemInput{AssetID: asset.ID, ItemID: second.ID, Role: inventory.AssetItemRoleMultiEpisodePart, SegmentIndex: 2}); err != nil {
		t.Fatalf("link second episode: %v", err)
	}

	var links []database.AssetItem
	if err := db.WithContext(ctx).
		Where("asset_id = ?", asset.ID).
		Order("segment_index asc").
		Order("id asc").
		Find(&links).Error; err != nil {
		t.Fatalf("list asset items: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected two episode links, got %#v", links)
	}
	if links[0].ItemID != first.ID || links[0].SegmentIndex != 1 || links[1].ItemID != second.ID || links[1].SegmentIndex != 2 {
		t.Fatalf("unexpected episode link order: %#v", links)
	}
}

func TestUpsertFileRefreshesInventoryRecord(t *testing.T) {
	db, ctx := newTestDB(t)
	svc := inventory.NewService(db)

	first, err := svc.UpsertFile(ctx, inventory.UpsertFileInput{LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/A.mkv", SizeBytes: 100})
	if err != nil {
		t.Fatalf("upsert first file: %v", err)
	}
	second, err := svc.UpsertFile(ctx, inventory.UpsertFileInput{LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/A.mkv", ThumbnailURL: "https://cdn.example.test/thumb.jpg", SizeBytes: 200, Container: "mkv"})
	if err != nil {
		t.Fatalf("upsert second file: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected same inventory file id, got %d and %d", first.ID, second.ID)
	}
	if second.SizeBytes != 200 || second.Container != "mkv" || second.ThumbnailURL != "https://cdn.example.test/thumb.jpg" {
		t.Fatalf("expected refreshed file metadata, got %#v", second)
	}
}

func TestUpsertInventoryFileUsesMediaSourceProviderPathIdentity(t *testing.T) {
	db, ctx := newTestDB(t)
	svc := inventory.NewService(db)

	first, err := svc.UpsertInventoryFile(ctx, inventory.UpsertInventoryFileInput{MediaSourceID: 1, StorageProvider: "local", StoragePath: "/movies/A.mkv", SizeBytes: 100})
	if err != nil {
		t.Fatalf("upsert first inventory file: %v", err)
	}
	second, err := svc.UpsertInventoryFile(ctx, inventory.UpsertInventoryFileInput{MediaSourceID: 1, StorageProvider: "local", StoragePath: "/movies/A.mkv", SizeBytes: 200, Container: "mkv"})
	if err != nil {
		t.Fatalf("upsert second inventory file: %v", err)
	}
	otherSource, err := svc.UpsertInventoryFile(ctx, inventory.UpsertInventoryFileInput{MediaSourceID: 2, StorageProvider: "local", StoragePath: "/movies/A.mkv", SizeBytes: 300})
	if err != nil {
		t.Fatalf("upsert other source inventory file: %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("expected same media source/path to reuse id, got %d and %d", first.ID, second.ID)
	}
	if second.SizeBytes != 200 || second.Container != "mkv" {
		t.Fatalf("expected refreshed file metadata, got %#v", second)
	}
	if otherSource.ID == second.ID {
		t.Fatal("expected different media source to create a distinct inventory file")
	}
	if otherSource.MediaSourceID != 2 || otherSource.LibraryID != 0 {
		t.Fatalf("expected media-source-owned inventory file without library ownership, got %#v", otherSource)
	}
}

func TestBulkUpsertFilesHandlesSQLiteVariableLimit(t *testing.T) {
	db, ctx := newTestDB(t)
	svc := inventory.NewService(db)

	inputs := make([]inventory.UpsertFileInput, 0, 1200)
	for i := 0; i < 1200; i++ {
		inputs = append(inputs, inventory.UpsertFileInput{
			LibraryID:         1,
			StorageProvider:   "local",
			StoragePath:       fmt.Sprintf("/movies/Movie %04d.mkv", i),
			StableIdentityKey: fmt.Sprintf("local:/movies/Movie %04d.mkv", i),
			SizeBytes:         int64(1000 + i),
			Container:         "mkv",
		})
	}

	result, err := svc.BulkUpsertFiles(ctx, inputs)
	if err != nil {
		t.Fatalf("bulk upsert files: %v", err)
	}
	if len(result.FilesByStoragePath) != len(inputs) {
		t.Fatalf("expected %d files in result, got %d", len(inputs), len(result.FilesByStoragePath))
	}
}

func TestUpsertResourceAndLinkFiles(t *testing.T) {
	db, ctx := newTestDB(t)
	svc := inventory.NewService(db)

	videoFile, err := svc.UpsertInventoryFile(ctx, inventory.UpsertInventoryFileInput{MediaSourceID: 1, StorageProvider: "local", StoragePath: "/movies/A.mkv", ContentClass: "video"})
	if err != nil {
		t.Fatalf("upsert video file: %v", err)
	}
	subtitleFile, err := svc.UpsertInventoryFile(ctx, inventory.UpsertInventoryFileInput{MediaSourceID: 1, StorageProvider: "local", StoragePath: "/movies/A.srt", ContentClass: "subtitle", Container: "srt"})
	if err != nil {
		t.Fatalf("upsert subtitle file: %v", err)
	}

	resource, err := svc.UpsertResource(ctx, inventory.UpsertResourceInput{StableResourceKey: "resource:movie:a", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, DisplayName: "Movie A"})
	if err != nil {
		t.Fatalf("upsert resource: %v", err)
	}
	updated, err := svc.UpsertResource(ctx, inventory.UpsertResourceInput{StableResourceKey: "resource:movie:a", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeMultiPart, DisplayName: "Movie A Remux", QualityLabel: "4K"})
	if err != nil {
		t.Fatalf("update resource: %v", err)
	}
	if resource.ID != updated.ID {
		t.Fatalf("expected stable resource key to reuse resource id, got %d and %d", resource.ID, updated.ID)
	}
	if updated.ResourceShape != database.ResourceShapeMultiPart || updated.QualityLabel != "4K" {
		t.Fatalf("expected resource update, got %#v", updated)
	}

	firstLink, err := svc.LinkResourceToFile(ctx, inventory.LinkResourceFileInput{ResourceID: updated.ID, InventoryFileID: videoFile.ID, Role: database.ResourceFileRoleSource, PartIndex: 0})
	if err != nil {
		t.Fatalf("link source file: %v", err)
	}
	secondLink, err := svc.LinkResourceToFile(ctx, inventory.LinkResourceFileInput{ResourceID: updated.ID, InventoryFileID: videoFile.ID, Role: database.ResourceFileRoleSource, PartIndex: 0})
	if err != nil {
		t.Fatalf("relink source file: %v", err)
	}
	if firstLink.ID != secondLink.ID {
		t.Fatalf("expected resource file link to be idempotent, got %d and %d", firstLink.ID, secondLink.ID)
	}
	if _, err := svc.LinkResourceToFile(ctx, inventory.LinkResourceFileInput{ResourceID: updated.ID, InventoryFileID: subtitleFile.ID, Role: database.ResourceFileRoleSubtitle}); err != nil {
		t.Fatalf("link subtitle file: %v", err)
	}

	var links []database.ResourceFile
	if err := db.WithContext(ctx).Where("resource_id = ?", updated.ID).Order("role asc").Find(&links).Error; err != nil {
		t.Fatalf("load resource files: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected two resource file links, got %#v", links)
	}
}

func TestAttachResourceToLibraryUpdatesSeenAndMissingState(t *testing.T) {
	db, ctx := newTestDB(t)
	svc := inventory.NewService(db)

	resource, err := svc.UpsertResource(ctx, inventory.UpsertResourceInput{StableResourceKey: "resource:movie:library-link"})
	if err != nil {
		t.Fatalf("upsert resource: %v", err)
	}
	firstSeen := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	link, err := svc.AttachResourceToLibrary(ctx, inventory.AttachResourceLibraryInput{ResourceID: resource.ID, LibraryID: 10, ObservedAt: &firstSeen, ReviewState: database.ReviewStateAccepted})
	if err != nil {
		t.Fatalf("attach resource to library: %v", err)
	}
	secondSeen := firstSeen.Add(2 * time.Hour)
	updated, err := svc.AttachResourceToLibrary(ctx, inventory.AttachResourceLibraryInput{ResourceID: resource.ID, LibraryID: 10, ObservedAt: &secondSeen, Status: inventory.AssetStatusAvailable})
	if err != nil {
		t.Fatalf("refresh resource library link: %v", err)
	}
	if link.ID != updated.ID {
		t.Fatalf("expected attach to be idempotent, got %d and %d", link.ID, updated.ID)
	}
	if !updated.FirstSeenAt.Equal(firstSeen) || !updated.LastSeenAt.Equal(secondSeen) {
		t.Fatalf("expected first seen preserved and last seen refreshed, got %#v", updated)
	}
	if updated.MissingSince != nil || updated.Status != inventory.AssetStatusAvailable {
		t.Fatalf("expected available link without missing state, got %#v", updated)
	}

	missingAt := secondSeen.Add(time.Hour)
	missing, err := svc.MarkResourceLibraryMissing(ctx, resource.ID, 10, missingAt)
	if err != nil {
		t.Fatalf("mark resource library missing: %v", err)
	}
	if missing.Status != inventory.AssetStatusMissing || missing.MissingSince == nil || !missing.MissingSince.Equal(missingAt) {
		t.Fatalf("expected missing state, got %#v", missing)
	}
}

func TestLinkResourceToMetadataIsIdempotentAndUnlinkable(t *testing.T) {
	db, ctx := newTestDB(t)
	svc := inventory.NewService(db)

	resource, err := svc.UpsertResource(ctx, inventory.UpsertResourceInput{StableResourceKey: "resource:movie:metadata-link"})
	if err != nil {
		t.Fatalf("upsert resource: %v", err)
	}
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie", GovernanceStatus: database.ReviewStatePending}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	confidence := 0.95
	link, err := svc.LinkResourceToMetadata(ctx, inventory.LinkResourceMetadataInput{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary, Confidence: &confidence, EvidenceJSON: `{"source":"test"}`, ReviewState: database.ReviewStateAccepted})
	if err != nil {
		t.Fatalf("link resource metadata: %v", err)
	}
	updatedConfidence := 0.75
	updated, err := svc.LinkResourceToMetadata(ctx, inventory.LinkResourceMetadataInput{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary, Confidence: &updatedConfidence, EvidenceJSON: `{"source":"updated"}`, ReviewState: database.ReviewStateNeedsReview})
	if err != nil {
		t.Fatalf("update resource metadata link: %v", err)
	}
	if link.ID != updated.ID {
		t.Fatalf("expected metadata link to be idempotent, got %d and %d", link.ID, updated.ID)
	}
	if updated.Confidence == nil || *updated.Confidence != updatedConfidence || updated.ReviewState != database.ReviewStateNeedsReview {
		t.Fatalf("expected updated metadata link fields, got %#v", updated)
	}

	if err := svc.UnlinkResourceFromMetadata(ctx, resource.ID, item.ID, database.ResourceLinkRolePrimary, 0); err != nil {
		t.Fatalf("unlink resource metadata: %v", err)
	}
	var count int64
	if err := db.WithContext(ctx).Model(&database.ResourceMetadataLink{}).Where("resource_id = ? AND metadata_item_id = ?", resource.ID, item.ID).Count(&count).Error; err != nil {
		t.Fatalf("count metadata links: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected metadata link to be removed, got %d", count)
	}
}

func TestResourceGraphSupportsSameBasenameCrossLibraryVersionsAndSegments(t *testing.T) {
	db, ctx := newTestDB(t)
	svc := inventory.NewService(db)

	firstFile, err := svc.UpsertInventoryFile(ctx, inventory.UpsertInventoryFileInput{MediaSourceID: 1, StorageProvider: "local", StoragePath: "/library-a/Movie.mkv", ContentClass: "video"})
	if err != nil {
		t.Fatalf("upsert first file: %v", err)
	}
	secondFile, err := svc.UpsertInventoryFile(ctx, inventory.UpsertInventoryFileInput{MediaSourceID: 1, StorageProvider: "local", StoragePath: "/library-b/Movie.mkv", ContentClass: "video"})
	if err != nil {
		t.Fatalf("upsert second file: %v", err)
	}
	if firstFile.ID == secondFile.ID {
		t.Fatal("expected same basename in different paths to create distinct inventory files")
	}

	resource, err := svc.UpsertResource(ctx, inventory.UpsertResourceInput{StableResourceKey: "resource:movie:shared", ResourceShape: database.ResourceShapeMultiPart})
	if err != nil {
		t.Fatalf("upsert shared resource: %v", err)
	}
	if _, err := svc.AttachResourceToLibrary(ctx, inventory.AttachResourceLibraryInput{ResourceID: resource.ID, LibraryID: 1}); err != nil {
		t.Fatalf("attach first library: %v", err)
	}
	if _, err := svc.AttachResourceToLibrary(ctx, inventory.AttachResourceLibraryInput{ResourceID: resource.ID, LibraryID: 2}); err != nil {
		t.Fatalf("attach second library: %v", err)
	}
	if err := svc.BulkLinkResourceToFiles(ctx, []inventory.LinkResourceFileInput{
		{ResourceID: resource.ID, InventoryFileID: firstFile.ID, Role: database.ResourceFileRoleSource, PartIndex: 1},
		{ResourceID: resource.ID, InventoryFileID: secondFile.ID, Role: database.ResourceFileRoleSource, PartIndex: 2},
	}); err != nil {
		t.Fatalf("bulk link resource files: %v", err)
	}
	var memberships int64
	if err := db.WithContext(ctx).Model(&database.ResourceLibraryLink{}).Where("resource_id = ?", resource.ID).Count(&memberships).Error; err != nil {
		t.Fatalf("count memberships: %v", err)
	}
	if memberships != 2 {
		t.Fatalf("expected cross-library membership, got %d", memberships)
	}

	movie := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie", GovernanceStatus: database.ReviewStatePending}
	firstEpisode := database.MetadataItem{ItemType: database.MetadataItemTypeEpisode, ContentForm: database.MetadataContentFormStandard, Title: "Episode 1", GovernanceStatus: database.ReviewStatePending}
	secondEpisode := database.MetadataItem{ItemType: database.MetadataItemTypeEpisode, ContentForm: database.MetadataContentFormStandard, Title: "Episode 2", GovernanceStatus: database.ReviewStatePending}
	if err := db.WithContext(ctx).Create([]*database.MetadataItem{&movie, &firstEpisode, &secondEpisode}).Error; err != nil {
		t.Fatalf("create metadata items: %v", err)
	}
	if _, err := svc.LinkResourceToMetadata(ctx, inventory.LinkResourceMetadataInput{ResourceID: resource.ID, MetadataItemID: movie.ID, Role: database.ResourceLinkRoleVersion, ReviewState: database.ReviewStateAccepted}); err != nil {
		t.Fatalf("link movie version: %v", err)
	}
	if _, err := svc.LinkResourceToMetadata(ctx, inventory.LinkResourceMetadataInput{ResourceID: resource.ID, MetadataItemID: firstEpisode.ID, Role: database.ResourceLinkRolePrimary, SegmentIndex: 1}); err != nil {
		t.Fatalf("link first episode segment: %v", err)
	}
	if _, err := svc.LinkResourceToMetadata(ctx, inventory.LinkResourceMetadataInput{ResourceID: resource.ID, MetadataItemID: secondEpisode.ID, Role: database.ResourceLinkRolePrimary, SegmentIndex: 2}); err != nil {
		t.Fatalf("link second episode segment: %v", err)
	}
	var links []database.ResourceMetadataLink
	if err := db.WithContext(ctx).Where("resource_id = ?", resource.ID).Order("segment_index asc").Find(&links).Error; err != nil {
		t.Fatalf("load metadata links: %v", err)
	}
	if len(links) != 3 {
		t.Fatalf("expected version and multi-episode links, got %#v", links)
	}

	if state := inventory.CandidateReviewState(inventory.ClassifyMetadataCandidate(inventory.MetadataCandidateEvidence{NormalizedTitleMatched: true})); state != database.ReviewStateNeedsReview {
		t.Fatalf("expected weak same-name candidate to require review, got %q", state)
	}
}

func newTestDB(t *testing.T) (*gorm.DB, context.Context) {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	return db, context.Background()
}
