package catalog

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func TestLegacyBackfillMovies(t *testing.T) {
	svc, ctx := newTestService(t)

	libraryID := uint(7)
	run, err := svc.createLegacyBackfillRun(ctx, LegacyBackfillScope{Kind: LegacyBackfillScopeLibrary, LibraryID: &libraryID}, 42)
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	confidence := 0.94
	year := 2024
	runtimeSeconds := 7020
	legacyMovie := database.MediaItem{
		LibraryID:          libraryID,
		Type:               "movie",
		Title:              "Movie A",
		OriginalTitle:      "Movie A Original",
		Overview:           "Catalog-ready movie overview",
		PosterURL:          "https://images.example.com/poster.jpg",
		BackdropURL:        "https://images.example.com/backdrop.jpg",
		LogoURL:            "https://images.example.com/logo.png",
		Year:               &year,
		RuntimeSeconds:     &runtimeSeconds,
		SourcePath:         "/library/movies/movie-a.mkv",
		MatchStatus:        "matched",
		MetadataProvider:   "tmdb",
		ExternalID:         "movie:101",
		MetadataConfidence: &confidence,
		Status:             "ready",
	}
	if err := svc.db.WithContext(ctx).Create(&legacyMovie).Error; err != nil {
		t.Fatalf("create legacy movie: %v", err)
	}

	skippedMovie := database.MediaItem{
		LibraryID:   libraryID,
		Type:        "movie",
		Title:       "Movie Without File",
		SourcePath:  "/library/movies/movie-without-file.mkv",
		MatchStatus: "pending",
		Status:      "ready",
	}
	if err := svc.db.WithContext(ctx).Create(&skippedMovie).Error; err != nil {
		t.Fatalf("create skipped movie: %v", err)
	}

	modifiedAt := time.Date(2026, time.April, 25, 6, 0, 0, 0, time.UTC)
	durationSeconds := 7020.5
	legacyFile := database.MediaFile{
		LibraryID:          libraryID,
		MediaItemID:        &legacyMovie.ID,
		StoragePath:        "/library/movies/movie-a.mkv",
		StableIdentityKey:  "stable:movie-a",
		IdentitySource:     "scan",
		IdentityStatus:     "confirmed",
		ProviderName:       "local",
		ProviderHashesJSON: `{"sha256":"abc123"}`,
		ReviewStatus:       "none",
		Container:          "mkv",
		SizeBytes:          987654321,
		ProbeStatus:        "complete",
		DurationSeconds:    &durationSeconds,
		LastModifiedAt:     &modifiedAt,
	}
	if err := svc.db.WithContext(ctx).Create(&legacyFile).Error; err != nil {
		t.Fatalf("create legacy file: %v", err)
	}

	if err := svc.backfillMovies(ctx, run); err != nil {
		t.Fatalf("backfill movies: %v", err)
	}

	finalized, err := svc.finalizeLegacyBackfillRun(ctx, run.ID, LegacyBackfillStatusCompleted, "")
	if err != nil {
		t.Fatalf("finalize run: %v", err)
	}
	if finalized.SuccessCount != 1 || finalized.SkippedCount != 1 {
		t.Fatalf("unexpected finalized counts: %#v", finalized)
	}

	var catalogItems []database.CatalogItem
	if err := svc.db.WithContext(ctx).Where("library_id = ? AND type = ?", libraryID, ItemTypeMovie).Order("id asc").Find(&catalogItems).Error; err != nil {
		t.Fatalf("list catalog items: %v", err)
	}
	if len(catalogItems) != 1 {
		t.Fatalf("expected one catalog movie, got %#v", catalogItems)
	}
	catalogItem := catalogItems[0]
	if catalogItem.Path != legacyMovie.SourcePath {
		t.Fatalf("expected catalog path %q, got %q", legacyMovie.SourcePath, catalogItem.Path)
	}
	if catalogItem.GovernanceStatus != GovernanceMatched {
		t.Fatalf("expected governance status %q, got %q", GovernanceMatched, catalogItem.GovernanceStatus)
	}
	if catalogItem.AvailabilityStatus != AvailabilityAvailable {
		t.Fatalf("expected availability status %q, got %q", AvailabilityAvailable, catalogItem.AvailabilityStatus)
	}

	var inventoryFiles []database.InventoryFile
	if err := svc.db.WithContext(ctx).Find(&inventoryFiles).Error; err != nil {
		t.Fatalf("list inventory files: %v", err)
	}
	if len(inventoryFiles) != 1 {
		t.Fatalf("expected one inventory file, got %#v", inventoryFiles)
	}
	inventoryFile := inventoryFiles[0]
	if inventoryFile.StorageProvider != legacyFile.ProviderName || inventoryFile.StoragePath != legacyFile.StoragePath {
		t.Fatalf("unexpected inventory file: %#v", inventoryFile)
	}
	if inventoryFile.StableIdentityKey != legacyFile.StableIdentityKey {
		t.Fatalf("expected stable identity %q, got %q", legacyFile.StableIdentityKey, inventoryFile.StableIdentityKey)
	}

	var assets []database.MediaAsset
	if err := svc.db.WithContext(ctx).Find(&assets).Error; err != nil {
		t.Fatalf("list assets: %v", err)
	}
	if len(assets) != 1 {
		t.Fatalf("expected one media asset, got %#v", assets)
	}
	asset := assets[0]
	if asset.AssetType != "main" {
		t.Fatalf("expected main asset, got %#v", asset)
	}
	if asset.ProbeStatus != legacyFile.ProbeStatus {
		t.Fatalf("expected probe status %q, got %q", legacyFile.ProbeStatus, asset.ProbeStatus)
	}
	if asset.DurationSeconds == nil || *asset.DurationSeconds != durationSeconds {
		t.Fatalf("expected duration %.1f, got %#v", durationSeconds, asset.DurationSeconds)
	}

	var assetFile database.AssetFile
	if err := svc.db.WithContext(ctx).First(&assetFile).Error; err != nil {
		t.Fatalf("load asset file link: %v", err)
	}
	if assetFile.AssetID != asset.ID || assetFile.FileID != inventoryFile.ID || assetFile.Role != "source" || assetFile.PartIndex != 0 {
		t.Fatalf("unexpected asset file link: %#v", assetFile)
	}

	var assetItem database.AssetItem
	if err := svc.db.WithContext(ctx).First(&assetItem).Error; err != nil {
		t.Fatalf("load asset item link: %v", err)
	}
	if assetItem.AssetID != asset.ID || assetItem.ItemID != catalogItem.ID {
		t.Fatalf("unexpected asset item link ids: %#v", assetItem)
	}
	if assetItem.Role != "primary" || assetItem.SegmentIndex != 0 || assetItem.Source != "legacy_backfill" {
		t.Fatalf("unexpected asset item link values: %#v", assetItem)
	}

	var images []database.ItemImage
	if err := svc.db.WithContext(ctx).Where("item_id = ?", catalogItem.ID).Order("image_type asc").Find(&images).Error; err != nil {
		t.Fatalf("list item images: %v", err)
	}
	if len(images) != 3 {
		t.Fatalf("expected poster/backdrop/logo images, got %#v", images)
	}
	imageURLs := map[string]string{}
	for _, image := range images {
		if !image.IsSelected {
			t.Fatalf("expected image %q to be selected", image.ImageType)
		}
		imageURLs[image.ImageType] = image.URL
	}
	if imageURLs["poster"] != legacyMovie.PosterURL || imageURLs["backdrop"] != legacyMovie.BackdropURL || imageURLs["logo"] != legacyMovie.LogoURL {
		t.Fatalf("unexpected item images: %#v", images)
	}

	var externalIDs []database.CatalogExternalID
	if err := svc.db.WithContext(ctx).Find(&externalIDs).Error; err != nil {
		t.Fatalf("list external ids: %v", err)
	}
	if len(externalIDs) != 1 {
		t.Fatalf("expected one external id, got %#v", externalIDs)
	}
	if externalIDs[0].ItemID != catalogItem.ID || externalIDs[0].Provider != legacyMovie.MetadataProvider || externalIDs[0].ProviderType != "movie" || externalIDs[0].ExternalID != legacyMovie.ExternalID {
		t.Fatalf("unexpected external id: %#v", externalIDs[0])
	}

	var sources []database.MetadataSource
	if err := svc.db.WithContext(ctx).Find(&sources).Error; err != nil {
		t.Fatalf("list metadata sources: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected one metadata source, got %#v", sources)
	}
	source := sources[0]
	if source.ItemID != catalogItem.ID || source.SourceType != SourceTypeProvider || source.SourceName != legacyMovie.MetadataProvider || source.ExternalID != legacyMovie.ExternalID {
		t.Fatalf("unexpected metadata source: %#v", source)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(source.PayloadJSON), &payload); err != nil {
		t.Fatalf("unmarshal metadata source payload: %v", err)
	}
	if payload["legacy_media_item_id"] != float64(legacyMovie.ID) || payload["match_status"] != legacyMovie.MatchStatus || payload["confidence"] != confidence {
		t.Fatalf("unexpected metadata source payload: %#v", payload)
	}

	report, err := svc.GetLegacyBackfillRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("load backfill report: %v", err)
	}
	if len(report.Entries) != 2 {
		t.Fatalf("expected success and skipped entries, got %#v", report.Entries)
	}
	entryTypes := map[string]LegacyBackfillEntry{}
	for _, entry := range report.Entries {
		entryTypes[entry.EntryType] = entry
	}
	if entryTypes[LegacyBackfillEntryTypeSuccess].LegacyMediaItemID == nil || *entryTypes[LegacyBackfillEntryTypeSuccess].LegacyMediaItemID != legacyMovie.ID {
		t.Fatalf("expected success entry for movie %d, got %#v", legacyMovie.ID, entryTypes[LegacyBackfillEntryTypeSuccess])
	}
	if entryTypes[LegacyBackfillEntryTypeSkipped].LegacyMediaItemID == nil || *entryTypes[LegacyBackfillEntryTypeSkipped].LegacyMediaItemID != skippedMovie.ID {
		t.Fatalf("expected skipped entry for movie %d, got %#v", skippedMovie.ID, entryTypes[LegacyBackfillEntryTypeSkipped])
	}
}

func TestLegacyBackfillMoviesIdempotent(t *testing.T) {
	svc, ctx := newTestService(t)

	libraryID := uint(8)
	confidence := 0.88
	year := 2023
	legacyMovie := database.MediaItem{
		LibraryID:          libraryID,
		Type:               "movie",
		Title:              "Movie B",
		SourcePath:         "/library/movies/movie-b.mp4",
		MatchStatus:        "matched",
		MetadataProvider:   "tmdb",
		ExternalID:         "movie:202",
		MetadataConfidence: &confidence,
		Year:               &year,
		Status:             "ready",
	}
	if err := svc.db.WithContext(ctx).Create(&legacyMovie).Error; err != nil {
		t.Fatalf("create legacy movie: %v", err)
	}

	durationSeconds := 5400.0
	legacyFile := database.MediaFile{
		LibraryID:          libraryID,
		MediaItemID:        &legacyMovie.ID,
		StoragePath:        legacyMovie.SourcePath,
		StableIdentityKey:  "stable:movie-b",
		IdentitySource:     "scan",
		IdentityStatus:     "confirmed",
		ProviderName:       "local",
		ProviderHashesJSON: `{"sha256":"def456"}`,
		ReviewStatus:       "none",
		Container:          "mp4",
		SizeBytes:          123456789,
		ProbeStatus:        "complete",
		DurationSeconds:    &durationSeconds,
	}
	if err := svc.db.WithContext(ctx).Create(&legacyFile).Error; err != nil {
		t.Fatalf("create legacy file: %v", err)
	}

	firstRun, err := svc.createLegacyBackfillRun(ctx, LegacyBackfillScope{Kind: LegacyBackfillScopeLibrary, LibraryID: &libraryID}, 7)
	if err != nil {
		t.Fatalf("create first run: %v", err)
	}
	if err := svc.backfillMovies(ctx, firstRun); err != nil {
		t.Fatalf("first backfill: %v", err)
	}
	if _, err := svc.finalizeLegacyBackfillRun(ctx, firstRun.ID, LegacyBackfillStatusCompleted, ""); err != nil {
		t.Fatalf("finalize first run: %v", err)
	}

	secondRun, err := svc.createLegacyBackfillRun(ctx, LegacyBackfillScope{Kind: LegacyBackfillScopeLibrary, LibraryID: &libraryID}, 8)
	if err != nil {
		t.Fatalf("create second run: %v", err)
	}
	if err := svc.backfillMovies(ctx, secondRun); err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	if _, err := svc.finalizeLegacyBackfillRun(ctx, secondRun.ID, LegacyBackfillStatusCompleted, ""); err != nil {
		t.Fatalf("finalize second run: %v", err)
	}

	assertTableRowCount(t, ctx, svc.db, &database.CatalogItem{}, 1)
	assertTableRowCount(t, ctx, svc.db, &database.InventoryFile{}, 1)
	assertTableRowCount(t, ctx, svc.db, &database.MediaAsset{}, 1)
	assertTableRowCount(t, ctx, svc.db, &database.AssetFile{}, 1)
	assertTableRowCount(t, ctx, svc.db, &database.AssetItem{}, 1)

	firstReport, err := svc.GetLegacyBackfillRun(ctx, firstRun.ID)
	if err != nil {
		t.Fatalf("load first report: %v", err)
	}
	secondReport, err := svc.GetLegacyBackfillRun(ctx, secondRun.ID)
	if err != nil {
		t.Fatalf("load second report: %v", err)
	}
	if len(firstReport.Entries) != 1 || len(secondReport.Entries) != 1 {
		t.Fatalf("expected one success entry per run, got first=%#v second=%#v", firstReport.Entries, secondReport.Entries)
	}
	firstSuccess := firstReport.Entries[0]
	secondSuccess := secondReport.Entries[0]
	if firstSuccess.EntryType != LegacyBackfillEntryTypeSuccess || secondSuccess.EntryType != LegacyBackfillEntryTypeSuccess {
		t.Fatalf("expected success entries, got first=%#v second=%#v", firstSuccess, secondSuccess)
	}
	if firstSuccess.CatalogItemID == nil || secondSuccess.CatalogItemID == nil || *firstSuccess.CatalogItemID != *secondSuccess.CatalogItemID {
		t.Fatalf("expected rerun to reuse catalog item id, got first=%#v second=%#v", firstSuccess.CatalogItemID, secondSuccess.CatalogItemID)
	}
	if firstSuccess.InventoryFileID == nil || secondSuccess.InventoryFileID == nil || *firstSuccess.InventoryFileID != *secondSuccess.InventoryFileID {
		t.Fatalf("expected rerun to reuse inventory file id, got first=%#v second=%#v", firstSuccess.InventoryFileID, secondSuccess.InventoryFileID)
	}
	if firstSuccess.AssetID == nil || secondSuccess.AssetID == nil || *firstSuccess.AssetID != *secondSuccess.AssetID {
		t.Fatalf("expected rerun to reuse asset id, got first=%#v second=%#v", firstSuccess.AssetID, secondSuccess.AssetID)
	}
}

func assertTableRowCount(t *testing.T, ctx context.Context, db *gorm.DB, model any, want int64) {
	t.Helper()

	var count int64
	if err := db.WithContext(ctx).Model(model).Count(&count).Error; err != nil {
		t.Fatalf("count rows for %T: %v", model, err)
	}
	if count != want {
		t.Fatalf("expected %d rows for %T, got %d", want, model, count)
	}
}
