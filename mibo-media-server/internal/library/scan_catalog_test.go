package library

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

func TestRunSyncLibraryWritesCatalogRows(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	showsRoot := filepath.Join(rootPath, "shows")
	mustWriteFixtureFile(t, filepath.Join(moviesRoot, "Movie A (2024)", "Movie.A.2024.mkv"))
	mustWriteFixtureFile(t, filepath.Join(showsRoot, "Show One", "Season 1", "Show.One.S01E02.mkv"))

	movieLibrary := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	showLibrary := createDirectScanLibrary(t, ctx, svc, "Shows", "shows", showsRoot)

	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(movieLibrary.ID, movieLibrary.RootPath)); err != nil {
		t.Fatalf("run movie sync: %v", err)
	}
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(showLibrary.ID, showLibrary.RootPath)); err != nil {
		t.Fatalf("run show sync: %v", err)
	}

	assertTableCount(t, ctx, db, &database.CatalogItem{}, 4)
	assertTableCount(t, ctx, db, &database.InventoryFile{}, 2)
	assertTableCount(t, ctx, db, &database.MediaAsset{}, 2)
	assertTableCount(t, ctx, db, &database.AssetItem{}, 2)
	assertTableCount(t, ctx, db, &database.AssetFile{}, 2)

	var projectionJobs int64
	if err := db.WithContext(ctx).
		Model(&database.Job{}).
		Where("kind = ?", JobKindCatalogRefreshLibraryProjection).
		Count(&projectionJobs).Error; err != nil {
		t.Fatalf("count catalog projection refresh jobs: %v", err)
	}
	if projectionJobs != 2 {
		t.Fatalf("expected one catalog projection refresh per scan, got %d", projectionJobs)
	}

	var matchJobs []database.Job
	if err := db.WithContext(ctx).
		Where("kind = ?", JobKindMatchCatalogItem).
		Order("id asc").
		Find(&matchJobs).Error; err != nil {
		t.Fatalf("list catalog match jobs: %v", err)
	}
	if len(matchJobs) != 2 {
		t.Fatalf("expected one catalog match job per canonical movie/series target, got %#v", matchJobs)
	}
}

func TestQueueCatalogItemMatchDeduplicatesEpisodeHierarchyToSeriesRoot(t *testing.T) {
	t.Parallel()

	ctx, db, svc, libraryRecord := newScanCatalogWriterHarness(t)
	catalogSvc := catalog.NewService(db)

	series, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryRecord.ID, Type: catalog.ItemTypeSeries, Title: "Show A", Path: "/library/Show A", SortKey: "Show A", AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernancePending})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryRecord.ID, Type: catalog.ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", Path: "/library/Show A/season-01", SortKey: "Show A S01", IndexNumber: &seasonNumber, AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernancePending})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	episodeNumber := 2
	episode, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryRecord.ID, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 2", Path: "/library/Show A/season-01/episode-02", SortKey: "Show A S01E02", IndexNumber: &episodeNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernancePending})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}

	jobFromSeason, err := svc.QueueCatalogItemMatch(ctx, season.ID)
	if err != nil {
		t.Fatalf("queue season catalog match: %v", err)
	}
	jobFromEpisode, err := svc.QueueCatalogItemMatch(ctx, episode.ID)
	if err != nil {
		t.Fatalf("queue episode catalog match: %v", err)
	}
	if jobFromSeason.ID == 0 || jobFromEpisode.ID == 0 || jobFromSeason.ID != jobFromEpisode.ID {
		t.Fatalf("expected season and episode queue to dedupe to the same job, got season=%#v episode=%#v", jobFromSeason, jobFromEpisode)
	}

	var queued []database.Job
	if err := db.WithContext(ctx).Where("kind = ?", JobKindMatchCatalogItem).Find(&queued).Error; err != nil {
		t.Fatalf("list catalog match jobs: %v", err)
	}
	if len(queued) != 1 {
		t.Fatalf("expected one queued catalog match job, got %#v", queued)
	}

	var payload struct {
		ItemID uint `json:"item_id"`
	}
	if err := json.Unmarshal([]byte(queued[0].PayloadJSON), &payload); err != nil {
		t.Fatalf("decode job payload: %v", err)
	}
	if payload.ItemID != series.ID {
		t.Fatalf("expected queued catalog match to target series root %d, got %d", series.ID, payload.ItemID)
	}
}

func TestRunSyncLibraryCreatesVersionAssetForDuplicateEpisodeSlot(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	showsRoot := filepath.Join(rootPath, "shows")
	mustWriteFixtureFile(t, filepath.Join(showsRoot, "Show One", "Season 1", "Show.One.S01E02.mkv"))
	mustWriteFixtureFile(t, filepath.Join(showsRoot, "Show One", "Season 1", "Show.One.S01E02.Directors.Cut.mkv"))

	showLibrary := createDirectScanLibrary(t, ctx, svc, "Shows", "shows", showsRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(showLibrary.ID, showLibrary.RootPath)); err != nil {
		t.Fatalf("run show sync: %v", err)
	}

	var episodeCount int64
	if err := db.WithContext(ctx).
		Model(&database.CatalogItem{}).
		Where("library_id = ? AND type = ?", showLibrary.ID, catalog.ItemTypeEpisode).
		Count(&episodeCount).Error; err != nil {
		t.Fatalf("count episode catalog items: %v", err)
	}
	if episodeCount != 1 {
		t.Fatalf("expected one canonical episode item for duplicate slot, got %d", episodeCount)
	}

	assertTableCount(t, ctx, db, &database.InventoryFile{}, 2)
	assertTableCount(t, ctx, db, &database.MediaAsset{}, 2)
	assertTableCount(t, ctx, db, &database.AssetItem{}, 2)
	assertTableCount(t, ctx, db, &database.AssetFile{}, 2)

	var assets []database.MediaAsset
	if err := db.WithContext(ctx).
		Where("library_id = ?", showLibrary.ID).
		Order("id asc").
		Find(&assets).Error; err != nil {
		t.Fatalf("list media assets: %v", err)
	}
	if len(assets) != 2 {
		t.Fatalf("expected two assets, got %#v", assets)
	}
	if assets[0].AssetType != "main" {
		t.Fatalf("expected first asset to remain main, got %#v", assets[0])
	}
	if assets[1].AssetType != "version" {
		t.Fatalf("expected duplicate slot to create version asset, got %#v", assets[1])
	}

	var assetItems []database.AssetItem
	if err := db.WithContext(ctx).
		Order("asset_id asc, id asc").
		Find(&assetItems).Error; err != nil {
		t.Fatalf("list asset items: %v", err)
	}
	if len(assetItems) != 2 {
		t.Fatalf("expected two asset-item links, got %#v", assetItems)
	}
	if assetItems[0].Role != "primary" {
		t.Fatalf("expected first asset-item link to stay primary, got %#v", assetItems[0])
	}
	if assetItems[1].Role != "version" {
		t.Fatalf("expected duplicate slot link to use version role, got %#v", assetItems[1])
	}
}

func TestRunSyncLibraryMarksMissingInventoryWithoutDeletingCatalogItem(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	filePath := filepath.Join(moviesRoot, "Movie A (2024)", "Movie.A.2024.mkv")
	mustWriteFixtureFile(t, filePath)

	movieLibrary := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(movieLibrary.ID, movieLibrary.RootPath)); err != nil {
		t.Fatalf("run initial movie sync: %v", err)
	}
	if err := os.Remove(filePath); err != nil {
		t.Fatalf("remove scanned movie file: %v", err)
	}
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(movieLibrary.ID, movieLibrary.RootPath)); err != nil {
		t.Fatalf("run missing-file sync: %v", err)
	}

	assertTableCount(t, ctx, db, &database.CatalogItem{}, 1)
	assertTableCount(t, ctx, db, &database.InventoryFile{}, 1)
	assertTableCount(t, ctx, db, &database.MediaAsset{}, 1)
	assertTableCount(t, ctx, db, &database.AssetItem{}, 1)
	assertTableCount(t, ctx, db, &database.AssetFile{}, 1)
	assertTableCount(t, ctx, db, &database.MetadataSource{}, 1)

	var item database.CatalogItem
	if err := db.WithContext(ctx).
		Where("library_id = ? AND type = ?", movieLibrary.ID, catalog.ItemTypeMovie).
		First(&item).Error; err != nil {
		t.Fatalf("load movie catalog item: %v", err)
	}
	if item.AvailabilityStatus != catalog.AvailabilityMissing {
		t.Fatalf("expected missing movie availability after delete, got %#v", item)
	}
	if item.DeletedAt != nil {
		t.Fatalf("expected movie catalog item to remain undeleted, got deleted_at=%v", item.DeletedAt)
	}

	var file database.InventoryFile
	if err := db.WithContext(ctx).
		Where("library_id = ?", movieLibrary.ID).
		First(&file).Error; err != nil {
		t.Fatalf("load inventory file: %v", err)
	}
	if file.Status != "missing" {
		t.Fatalf("expected missing inventory status after delete, got %#v", file)
	}
	if file.DeletedAt != nil {
		t.Fatalf("expected inventory file to remain undeleted, got deleted_at=%v", file.DeletedAt)
	}

	var asset database.MediaAsset
	if err := db.WithContext(ctx).
		Where("library_id = ?", movieLibrary.ID).
		First(&asset).Error; err != nil {
		t.Fatalf("load media asset: %v", err)
	}
	if asset.Status != "missing" {
		t.Fatalf("expected missing asset status after delete, got %#v", asset)
	}
	if asset.DeletedAt != nil {
		t.Fatalf("expected media asset to remain undeleted, got deleted_at=%v", asset.DeletedAt)
	}
}

func TestRunSyncLibraryKeepsEpisodeAvailableWhenAnotherVersionRemains(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	showsRoot := filepath.Join(rootPath, "shows")
	firstPath := filepath.Join(showsRoot, "Show One", "Season 1", "Show.One.S01E02.mkv")
	secondPath := filepath.Join(showsRoot, "Show One", "Season 1", "Show.One.S01E02.Directors.Cut.mkv")
	mustWriteFixtureFile(t, firstPath)
	mustWriteFixtureFile(t, secondPath)

	showLibrary := createDirectScanLibrary(t, ctx, svc, "Shows", "shows", showsRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(showLibrary.ID, showLibrary.RootPath)); err != nil {
		t.Fatalf("run initial show sync: %v", err)
	}
	if err := os.Remove(firstPath); err != nil {
		t.Fatalf("remove first version file: %v", err)
	}
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(showLibrary.ID, showLibrary.RootPath)); err != nil {
		t.Fatalf("run version delete sync: %v", err)
	}

	var episode database.CatalogItem
	if err := db.WithContext(ctx).
		Where("library_id = ? AND type = ?", showLibrary.ID, catalog.ItemTypeEpisode).
		First(&episode).Error; err != nil {
		t.Fatalf("load episode catalog item: %v", err)
	}
	if episode.AvailabilityStatus != catalog.AvailabilityAvailable {
		t.Fatalf("expected episode to stay available while another version remains, got %#v", episode)
	}

	var files []database.InventoryFile
	if err := db.WithContext(ctx).
		Where("library_id = ?", showLibrary.ID).
		Order("storage_path asc").
		Find(&files).Error; err != nil {
		t.Fatalf("list inventory files: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected two inventory files to remain recorded, got %#v", files)
	}
	if files[0].Status != "missing" && files[1].Status != "missing" {
		t.Fatalf("expected deleted version to be marked missing, got %#v", files)
	}
	if files[0].Status != "available" && files[1].Status != "available" {
		t.Fatalf("expected surviving version to remain available, got %#v", files)
	}

	var assets []database.MediaAsset
	if err := db.WithContext(ctx).
		Where("library_id = ?", showLibrary.ID).
		Order("id asc").
		Find(&assets).Error; err != nil {
		t.Fatalf("list media assets: %v", err)
	}
	if len(assets) != 2 {
		t.Fatalf("expected two media assets to remain recorded, got %#v", assets)
	}
	if assets[0].Status != "missing" && assets[1].Status != "missing" {
		t.Fatalf("expected one version asset to be marked missing, got %#v", assets)
	}
	if assets[0].Status != "available" && assets[1].Status != "available" {
		t.Fatalf("expected one version asset to remain available, got %#v", assets)
	}
}

func TestRunSyncLibraryReusesStableIdentityCatalogRowsOnRename(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, svc, libraryRecord := newIdentityScanService(t)
	provider := &stableIdentityProvider{objects: [][]storage.Object{
		{{Name: "MovieA.2024.mkv", Path: "/library/MovieA.2024.mkv", Size: 2048, StableIdentity: "provider-object-1", Modified: timePtr(time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC))}},
		{{Name: "Renamed.Movie.2024.mkv", Path: "/library/Renamed.Movie.2024.mkv", Size: 2048, StableIdentity: "provider-object-1", Modified: timePtr(time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC))}},
	}}

	if _, err := svc.scanLibrary(ctx, provider, libraryRecord, "/library"); err != nil {
		t.Fatalf("run initial scan: %v", err)
	}

	var firstFile database.InventoryFile
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).First(&firstFile).Error; err != nil {
		t.Fatalf("load first inventory file: %v", err)
	}
	var firstAsset database.MediaAsset
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).First(&firstAsset).Error; err != nil {
		t.Fatalf("load first media asset: %v", err)
	}
	var firstItem database.CatalogItem
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).First(&firstItem).Error; err != nil {
		t.Fatalf("load first catalog item: %v", err)
	}

	if _, err := svc.scanLibrary(ctx, provider, libraryRecord, "/library"); err != nil {
		t.Fatalf("run rename scan: %v", err)
	}

	assertTableCount(t, ctx, db, &database.InventoryFile{}, 1)
	assertTableCount(t, ctx, db, &database.MediaAsset{}, 1)
	assertTableCount(t, ctx, db, &database.AssetFile{}, 1)
	assertTableCount(t, ctx, db, &database.CatalogItem{}, 1)
	assertTableCount(t, ctx, db, &database.AssetItem{}, 1)

	var file database.InventoryFile
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).First(&file).Error; err != nil {
		t.Fatalf("reload inventory file: %v", err)
	}
	if file.ID != firstFile.ID {
		t.Fatalf("expected stable identity rename to reuse inventory file id %d, got %d", firstFile.ID, file.ID)
	}
	if file.StoragePath != "/library/Renamed.Movie.2024.mkv" {
		t.Fatalf("expected reused inventory file to move to renamed path, got %#v", file)
	}

	var asset database.MediaAsset
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).First(&asset).Error; err != nil {
		t.Fatalf("reload media asset: %v", err)
	}
	if asset.ID != firstAsset.ID {
		t.Fatalf("expected stable identity rename to reuse asset id %d, got %d", firstAsset.ID, asset.ID)
	}
	if asset.Status != "available" {
		t.Fatalf("expected reused asset to stay available after rename, got %#v", asset)
	}

	var item database.CatalogItem
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).First(&item).Error; err != nil {
		t.Fatalf("reload catalog item: %v", err)
	}
	if item.ID != firstItem.ID {
		t.Fatalf("expected stable identity rename to reuse catalog item id %d, got %d", firstItem.ID, item.ID)
	}
	if item.Path != "/library/Renamed.Movie.2024.mkv" {
		t.Fatalf("expected reused catalog item path to update, got %#v", item)
	}
}

func TestRunSyncLibraryDeduplicatesScannerMetadataSourcesOnRescan(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	moviePath := filepath.Join(moviesRoot, "Movie A (2024)", "Movie.A.2024.mkv")
	mustWriteFixtureFile(t, moviePath)

	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run initial sync: %v", err)
	}
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run rescan: %v", err)
	}

	assertTableCount(t, ctx, db, &database.CatalogItem{}, 1)
	assertTableCount(t, ctx, db, &database.MetadataSource{}, 1)
}

func TestRunSyncLibraryMarksAncestorAvailabilityMissingWhenEpisodesDeleted(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	showsRoot := filepath.Join(rootPath, "shows")
	episodePath := filepath.Join(showsRoot, "Show One", "Season 1", "Show.One.S01E02.mkv")
	mustWriteFixtureFile(t, episodePath)

	showLibrary := createDirectScanLibrary(t, ctx, svc, "Shows", "shows", showsRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(showLibrary.ID, showLibrary.RootPath)); err != nil {
		t.Fatalf("run initial show sync: %v", err)
	}
	if err := os.Remove(episodePath); err != nil {
		t.Fatalf("remove episode file: %v", err)
	}
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(showLibrary.ID, showLibrary.RootPath)); err != nil {
		t.Fatalf("run delete sync: %v", err)
	}

	var items []database.CatalogItem
	if err := db.WithContext(ctx).Where("library_id = ?", showLibrary.ID).Order("id asc").Find(&items).Error; err != nil {
		t.Fatalf("list catalog items: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected series, season, and episode rows, got %#v", items)
	}
	for _, item := range items {
		if item.AvailabilityStatus != catalog.AvailabilityMissing {
			t.Fatalf("expected deleted hierarchy item to become missing, got %#v", item)
		}
	}
	if items[0].Type != catalog.ItemTypeSeries || items[1].Type != catalog.ItemTypeSeason || items[2].Type != catalog.ItemTypeEpisode {
		t.Fatalf("unexpected item ordering for hierarchy: %#v", items)
	}
}

func TestScanCatalogWriterCreatesMovieKernelRows(t *testing.T) {
	t.Parallel()

	ctx, db, svc, libraryRecord := newScanCatalogWriterHarness(t)
	modifiedAt := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	year := 2024

	result, err := svc.writeCatalogScanMovie(ctx, libraryRecord, catalogScanArtifact{
		ItemType:          catalog.ItemTypeMovie,
		ItemPath:          "/library/Movie A (2024)/movie.mkv",
		SourcePath:        "/library/Movie A (2024)/movie.mkv",
		Title:             "Movie A",
		OriginalTitle:     "Movie.A.2024",
		Year:              &year,
		StorageProvider:   "local",
		StableIdentityKey: "local:movie-a-2024",
		ProviderName:      "scanner-provider",
		HashesJSON:        `{"sha256":"abc123"}`,
		SizeBytes:         4096,
		ModifiedAt:        &modifiedAt,
		Container:         "mkv",
	})
	if err != nil {
		t.Fatalf("write movie artifact: %v", err)
	}

	if result.Item.Type != catalog.ItemTypeMovie {
		t.Fatalf("expected movie item, got %#v", result.Item)
	}
	if result.File.Status != "available" {
		t.Fatalf("expected available file, got %#v", result.File)
	}
	if result.Asset.AssetType != "main" {
		t.Fatalf("expected main asset, got %#v", result.Asset)
	}

	assertCatalogCounts(t, ctx, db, 1, 1, 1, 1, 1, 1)

	var item database.CatalogItem
	if err := db.WithContext(ctx).First(&item, result.Item.ID).Error; err != nil {
		t.Fatalf("reload catalog item: %v", err)
	}
	if item.AvailabilityStatus != catalog.AvailabilityAvailable || item.GovernanceStatus != catalog.GovernancePending {
		t.Fatalf("expected available/pending item, got %#v", item)
	}

	var assetItem database.AssetItem
	if err := db.WithContext(ctx).First(&assetItem, "asset_id = ?", result.Asset.ID).Error; err != nil {
		t.Fatalf("load asset item: %v", err)
	}
	if assetItem.Role != "primary" || assetItem.SegmentIndex != 0 || assetItem.Source != "scanner" {
		t.Fatalf("unexpected asset-item link: %#v", assetItem)
	}

	var assetFile database.AssetFile
	if err := db.WithContext(ctx).First(&assetFile, "asset_id = ?", result.Asset.ID).Error; err != nil {
		t.Fatalf("load asset file: %v", err)
	}
	if assetFile.Role != "source" || assetFile.PartIndex != 0 {
		t.Fatalf("unexpected asset-file link: %#v", assetFile)
	}

	var source database.MetadataSource
	if err := db.WithContext(ctx).First(&source, "item_id = ?", result.Item.ID).Error; err != nil {
		t.Fatalf("load metadata source: %v", err)
	}
	if source.SourceType != catalog.SourceTypeLocalFile || source.SourceName != "scanner" {
		t.Fatalf("unexpected metadata source: %#v", source)
	}
	assertEvidencePayloadKeys(t, source.PayloadJSON, []string{"detected_title", "hashes_json", "provider_name", "stable_identity_key", "storage_path"})
}

func TestScanCatalogWriterPreservesMetadataFieldsOnMovieRescan(t *testing.T) {
	t.Parallel()

	ctx, db, svc, libraryRecord := newScanCatalogWriterHarness(t)
	catalogSvc := catalog.NewService(db)
	initialYear := 2024
	matchedYear := 2025
	rescannedYear := 2026
	artifact := catalogScanArtifact{
		ItemType:        catalog.ItemTypeMovie,
		ItemPath:        "/library/Movie A (2024)/movie.mkv",
		SourcePath:      "/library/Movie A (2024)/movie.mkv",
		Title:           "Movie A",
		OriginalTitle:   "Movie.A.2024",
		Year:            &initialYear,
		StorageProvider: "local",
		SizeBytes:       4096,
		Container:       "mkv",
	}

	result, err := svc.writeCatalogScanMovie(ctx, libraryRecord, artifact)
	if err != nil {
		t.Fatalf("write initial movie artifact: %v", err)
	}
	if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: result.Item.ID, FieldKey: "title", Value: "Matched Movie"}); err != nil {
		t.Fatalf("apply matched title: %v", err)
	}
	if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: result.Item.ID, FieldKey: "original_title", Value: "Matched Original"}); err != nil {
		t.Fatalf("apply matched original title: %v", err)
	}
	if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: result.Item.ID, FieldKey: "year", Value: matchedYear}); err != nil {
		t.Fatalf("apply matched year: %v", err)
	}
	if err := db.WithContext(ctx).Model(&database.CatalogItem{}).Where("id = ?", result.Item.ID).Updates(map[string]any{
		"title":          "Previously Reverted Title",
		"original_title": "Previously.Reverted",
		"year":           2001,
	}).Error; err != nil {
		t.Fatalf("simulate stale scanner overwrite: %v", err)
	}

	artifact.Title = "Movie A Remux"
	artifact.OriginalTitle = "Movie.A.Remux.2026"
	artifact.Year = &rescannedYear
	if _, err := svc.writeCatalogScanMovie(ctx, libraryRecord, artifact); err != nil {
		t.Fatalf("write rescan movie artifact: %v", err)
	}

	var item database.CatalogItem
	if err := db.WithContext(ctx).First(&item, result.Item.ID).Error; err != nil {
		t.Fatalf("reload movie item: %v", err)
	}
	if item.Title != "Matched Movie" || item.OriginalTitle != "Matched Original" || item.Year == nil || *item.Year != matchedYear {
		t.Fatalf("expected metadata fields to survive rescan, got %#v", item)
	}
}

func TestScanCatalogWriterCreatesEpisodeHierarchyWithLocalEvidence(t *testing.T) {
	t.Parallel()

	ctx, db, svc, libraryRecord := newScanCatalogWriterHarness(t)
	modifiedAt := time.Date(2026, 4, 25, 12, 30, 0, 0, time.UTC)
	seasonNumber := 1

	result, err := svc.writeCatalogScanEpisodeHierarchy(ctx, libraryRecord, catalogScanArtifact{
		ItemType:          catalog.ItemTypeEpisode,
		SourcePath:        "/library/Show One/Season 1/Show One.S01E02.mkv",
		SeriesPath:        "show-one",
		SeasonPath:        "show-one/season-01",
		Title:             "Show One S01E02",
		OriginalTitle:     "Show.One.S01E02",
		SeriesTitle:       "Show One",
		SeasonNumber:      &seasonNumber,
		EpisodeSlots:      []catalogEpisodeSlot{{EpisodeNumber: 2, ItemPath: "show-one/season-01/episode-0002"}},
		StorageProvider:   "local",
		StableIdentityKey: "local:show-one-s01e02",
		ProviderName:      "scanner-provider",
		HashesJSON:        `{"sha1":"deadbeef"}`,
		SizeBytes:         8192,
		ModifiedAt:        &modifiedAt,
		Container:         "mkv",
	})
	if err != nil {
		t.Fatalf("write episode artifact: %v", err)
	}

	assertCatalogCounts(t, ctx, db, 3, 1, 1, 1, 1, 1)

	var items []database.CatalogItem
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).Order("id asc").Find(&items).Error; err != nil {
		t.Fatalf("list catalog items: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected series/season/episode rows, got %#v", items)
	}
	for _, item := range items {
		if item.GovernanceStatus != catalog.GovernancePending || item.AvailabilityStatus != catalog.AvailabilityAvailable {
			t.Fatalf("expected pending/available hierarchy row, got %#v", item)
		}
	}
	if items[0].Path != "show-one" || items[1].Path != "show-one/season-01" || items[2].Path != "show-one/season-01/episode-0002" {
		t.Fatalf("unexpected canonical hierarchy paths: %#v", items)
	}
	if items[1].ParentID == nil || *items[1].ParentID != items[0].ID {
		t.Fatalf("expected season parent link, got %#v", items[1])
	}
	if items[2].ParentID == nil || *items[2].ParentID != items[1].ID {
		t.Fatalf("expected episode parent link, got %#v", items[2])
	}

	var source database.MetadataSource
	if err := db.WithContext(ctx).First(&source, "item_id = ?", result.Item.ID).Error; err != nil {
		t.Fatalf("load episode metadata source: %v", err)
	}
	if source.SourceType != catalog.SourceTypeLocalFile || source.SourceName != "scanner" {
		t.Fatalf("unexpected metadata source: %#v", source)
	}
	assertEvidencePayloadKeys(t, source.PayloadJSON, []string{"detected_title", "episode_numbers", "hashes_json", "provider_name", "season_number", "series_title", "stable_identity_key", "storage_path"})

	var payload map[string]any
	if err := json.Unmarshal([]byte(source.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode payload json: %v", err)
	}
	episodes, ok := payload["episode_numbers"].([]any)
	if !ok || len(episodes) != 1 || int(episodes[0].(float64)) != 2 {
		t.Fatalf("expected compact episode evidence, got %#v", payload)
	}
	if payload["series_title"] != "Show One" || int(payload["season_number"].(float64)) != 1 {
		t.Fatalf("expected series and season evidence, got %#v", payload)
	}

	var assetItem database.AssetItem
	if err := db.WithContext(ctx).First(&assetItem, "asset_id = ?", result.Asset.ID).Error; err != nil {
		t.Fatalf("load asset item: %v", err)
	}
	if assetItem.Role != "primary" || assetItem.Source != "scanner" {
		t.Fatalf("unexpected episode asset-item link: %#v", assetItem)
	}
	if result.Item.Path != "show-one/season-01/episode-0002" {
		t.Fatalf("expected episode leaf item, got %#v", result.Item)
	}
}

func TestScanCatalogWriterReusesProviderCreatedDescendantsByHierarchyIdentity(t *testing.T) {
	t.Parallel()

	ctx, db, svc, libraryRecord := newScanCatalogWriterHarness(t)
	catalogSvc := catalog.NewService(db)
	series, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{
		LibraryID:          libraryRecord.ID,
		Type:               catalog.ItemTypeSeries,
		Path:               "show-one",
		SortKey:            "Show One",
		Title:              "Matched Show One",
		AvailabilityStatus: catalog.AvailabilityMissing,
		GovernanceStatus:   catalog.GovernanceMatched,
	})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{
		LibraryID:          libraryRecord.ID,
		Type:               catalog.ItemTypeSeason,
		ParentID:           &series.ID,
		Path:               "show-one/Season 01",
		SortKey:            "Show One S01",
		Title:              "Season 1",
		IndexNumber:        &seasonNumber,
		ParentIndexNumber:  &seasonNumber,
		AvailabilityStatus: catalog.AvailabilityMissing,
		GovernanceStatus:   catalog.GovernanceMatched,
	})
	if err != nil {
		t.Fatalf("create provider season: %v", err)
	}
	episodeNumber := 2
	episode, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{
		LibraryID:          libraryRecord.ID,
		Type:               catalog.ItemTypeEpisode,
		ParentID:           &season.ID,
		Path:               "show-one/Season 01/Episode 02",
		SortKey:            "Show One S01E02",
		Title:              "Matched Episode 2",
		IndexNumber:        &episodeNumber,
		ParentIndexNumber:  &seasonNumber,
		AvailabilityStatus: catalog.AvailabilityMissing,
		GovernanceStatus:   catalog.GovernanceMatched,
	})
	if err != nil {
		t.Fatalf("create provider episode: %v", err)
	}
	if _, err := catalogSvc.SetExternalID(ctx, catalog.ExternalIDInput{ItemID: episode.ID, Provider: "tmdb", ProviderType: "tv_episode", ExternalID: "tv:1002", IsPrimary: true}); err != nil {
		t.Fatalf("seed provider identity: %v", err)
	}
	if _, err := catalogSvc.RecordMetadataSource(ctx, catalog.MetadataSourceInput{ItemID: episode.ID, SourceType: catalog.SourceTypeProvider, SourceName: "tmdb", ExternalID: "tv:1002", PayloadJSON: `{"matched_title":"Matched Episode 2"}`}); err != nil {
		t.Fatalf("seed provider evidence: %v", err)
	}

	modifiedAt := time.Date(2026, 4, 25, 13, 0, 0, 0, time.UTC)
	result, err := svc.writeCatalogScanEpisodeHierarchy(ctx, libraryRecord, catalogScanArtifact{
		ItemType:          catalog.ItemTypeEpisode,
		SourcePath:        "/library/Show One/Season 1/Show One.S01E02.mkv",
		SeriesPath:        "show-one",
		SeasonPath:        "show-one/season-01",
		Title:             "Show One S01E02",
		OriginalTitle:     "Show.One.S01E02",
		SeriesTitle:       "Show One",
		SeasonNumber:      &seasonNumber,
		EpisodeSlots:      []catalogEpisodeSlot{{EpisodeNumber: 2, ItemPath: "show-one/season-01/episode-0002"}},
		StorageProvider:   "local",
		StableIdentityKey: "local:show-one-s01e02",
		ProviderName:      "scanner-provider",
		HashesJSON:        `{"sha1":"deadbeef"}`,
		SizeBytes:         8192,
		ModifiedAt:        &modifiedAt,
		Container:         "mkv",
	})
	if err != nil {
		t.Fatalf("write scan hierarchy: %v", err)
	}

	assertCatalogCounts(t, ctx, db, 3, 1, 1, 1, 1, 2)
	if result.Item.ID != episode.ID {
		t.Fatalf("expected scanner to reuse provider-created episode %d, got %#v", episode.ID, result.Item)
	}

	var reloadedSeason database.CatalogItem
	if err := db.WithContext(ctx).First(&reloadedSeason, season.ID).Error; err != nil {
		t.Fatalf("reload season: %v", err)
	}
	if reloadedSeason.Path != "show-one/season-01" || reloadedSeason.Title != "Season 1" || reloadedSeason.GovernanceStatus != catalog.GovernanceMatched {
		t.Fatalf("expected season path rewrite without governance loss, got %#v", reloadedSeason)
	}

	var reloadedEpisode database.CatalogItem
	if err := db.WithContext(ctx).First(&reloadedEpisode, episode.ID).Error; err != nil {
		t.Fatalf("reload episode: %v", err)
	}
	if reloadedEpisode.Path != "show-one/season-01/episode-0002" || reloadedEpisode.Title != "Matched Episode 2" || reloadedEpisode.AvailabilityStatus != catalog.AvailabilityAvailable || reloadedEpisode.GovernanceStatus != catalog.GovernanceMatched {
		t.Fatalf("expected reused episode to become available while preserving governance, got %#v", reloadedEpisode)
	}

	var externalID database.CatalogExternalID
	if err := db.WithContext(ctx).Where("item_id = ? AND provider = ? AND provider_type = ?", episode.ID, "tmdb", "tv_episode").First(&externalID).Error; err != nil {
		t.Fatalf("reload external id: %v", err)
	}
	if externalID.ExternalID != "tv:1002" {
		t.Fatalf("expected descendant identity to survive scanner reuse, got %#v", externalID)
	}
}

func newScanCatalogWriterHarness(t *testing.T) (context.Context, *gorm.DB, *Service, database.Library) {
	t.Helper()

	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	rootPath := "/library"
	registry := providers.NewRegistry(config.Config{Local: config.LocalStorageConfig{RootPath: rootPath}})
	svc := NewService(config.Config{Local: config.LocalStorageConfig{RootPath: rootPath}}, db, registry, jobs.NewService(db))
	libraryRecord := database.Library{Name: "Library", Type: "shows", RootPath: rootPath, Status: "active"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}

	return ctx, db, svc, libraryRecord
}

func newDirectScanHarness(t *testing.T, rootPath string) (context.Context, *gorm.DB, *Service) {
	t.Helper()

	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{Local: config.LocalStorageConfig{RootPath: rootPath}}
	registry := providers.NewRegistry(cfg)
	svc := NewService(cfg, db, registry, jobs.NewService(db))

	return ctx, db, svc
}

func createDirectScanLibrary(t *testing.T, ctx context.Context, svc *Service, name string, libraryType string, rootPath string) database.Library {
	t.Helper()

	source, err := svc.CreateMediaSource(ctx, CreateMediaSourceInput{Provider: "local", Name: fmt.Sprintf("%s Source", name), RootPath: rootPath})
	if err != nil {
		t.Fatalf("create media source %s: %v", name, err)
	}
	libraryRecord, _, err := svc.CreateLibrary(ctx, CreateLibraryInput{Name: name, Type: libraryType, MediaSourceID: source.ID, RootPath: rootPath})
	if err != nil {
		t.Fatalf("create library %s: %v", name, err)
	}
	return libraryRecord
}

func newSyncLibraryJobPayload(libraryID uint, rootPath string) database.Job {
	payloadJSON, err := json.Marshal(map[string]any{"library_id": libraryID, "root_path": rootPath})
	if err != nil {
		panic(err)
	}
	return database.Job{PayloadJSON: string(payloadJSON)}
}

func mustWriteFixtureFile(t *testing.T, filePath string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir fixture dir %s: %v", filepath.Dir(filePath), err)
	}
	if err := os.WriteFile(filePath, []byte("fixture"), 0o644); err != nil {
		t.Fatalf("write fixture file %s: %v", filePath, err)
	}
}

func assertCatalogCounts(t *testing.T, ctx context.Context, db *gorm.DB, itemCount int64, fileCount int64, assetCount int64, assetItemCount int64, assetFileCount int64, metadataSourceCount int64) {
	t.Helper()

	assertTableCount(t, ctx, db, &database.CatalogItem{}, itemCount)
	assertTableCount(t, ctx, db, &database.InventoryFile{}, fileCount)
	assertTableCount(t, ctx, db, &database.MediaAsset{}, assetCount)
	assertTableCount(t, ctx, db, &database.AssetItem{}, assetItemCount)
	assertTableCount(t, ctx, db, &database.AssetFile{}, assetFileCount)
	assertTableCount(t, ctx, db, &database.MetadataSource{}, metadataSourceCount)
}

func assertTableCount(t *testing.T, ctx context.Context, db *gorm.DB, model any, expected int64) {
	t.Helper()

	var actual int64
	if err := db.WithContext(ctx).Model(model).Count(&actual).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	if actual != expected {
		t.Fatalf("expected %d rows for %T, got %d", expected, model, actual)
	}
}

func assertEvidencePayloadKeys(t *testing.T, payloadJSON string, expectedKeys []string) {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("decode payload json: %v", err)
	}
	if len(payload) != len(expectedKeys) {
		t.Fatalf("expected only allowlisted payload keys %v, got %#v", expectedKeys, payload)
	}
	for _, key := range expectedKeys {
		if _, ok := payload[key]; !ok {
			t.Fatalf("expected payload key %q, got %#v", key, payload)
		}
	}
}
