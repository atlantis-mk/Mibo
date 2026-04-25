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
	"gorm.io/gorm"
)

func TestRunSyncLibraryWritesCatalogRowsWithoutLegacyMediaTables(t *testing.T) {
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
	assertTableCount(t, ctx, db, &database.MediaItem{}, 0)
	assertTableCount(t, ctx, db, &database.MediaFile{}, 0)

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
	assertTableCount(t, ctx, db, &database.MediaItem{}, 0)
	assertTableCount(t, ctx, db, &database.MediaFile{}, 0)

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
