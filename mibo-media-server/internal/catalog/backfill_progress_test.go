package catalog_test

import (
	"context"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/settings"
	"github.com/atlan/mibo-media-server/internal/worker"
	"gorm.io/gorm"
)

func TestLegacyBackfillProgress(t *testing.T) {
	t.Parallel()

	fx := newLegacyBackfillFixture(t)
	ctx := fx.ctx

	runtimeSeconds := 1000
	legacyItem := database.MediaItem{
		LibraryID:      fx.library.ID,
		Type:           "movie",
		Title:          "Movie With Progress",
		SourcePath:     "/library/movies/movie-with-progress.mkv",
		RuntimeSeconds: &runtimeSeconds,
		MatchStatus:    "matched",
		Status:         "ready",
	}
	if err := fx.db.WithContext(ctx).Create(&legacyItem).Error; err != nil {
		t.Fatalf("create legacy media item: %v", err)
	}

	durationSecondsFloat := 1000.0
	legacyFile := database.MediaFile{
		LibraryID:         fx.library.ID,
		MediaItemID:       &legacyItem.ID,
		StoragePath:       legacyItem.SourcePath,
		StableIdentityKey: "stable:movie-with-progress",
		ProviderName:      "local",
		ProbeStatus:       "complete",
		DurationSeconds:   &durationSecondsFloat,
	}
	if err := fx.db.WithContext(ctx).Create(&legacyFile).Error; err != nil {
		t.Fatalf("create legacy media file: %v", err)
	}

	catalogItem, err := fx.catalog.CreateItem(ctx, catalog.CreateItemInput{
		LibraryID:          fx.library.ID,
		Type:               catalog.ItemTypeMovie,
		Path:               legacyItem.SourcePath,
		SortKey:            legacyItem.Title,
		Title:              legacyItem.Title,
		AvailabilityStatus: catalog.AvailabilityAvailable,
		GovernanceStatus:   catalog.GovernanceMatched,
		RuntimeSeconds:     &runtimeSeconds,
	})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}

	inventoryFile, err := fx.inventory.UpsertFile(ctx, inventory.UpsertFileInput{
		LibraryID:         fx.library.ID,
		StorageProvider:   "local",
		StoragePath:       legacyFile.StoragePath,
		StableIdentityKey: legacyFile.StableIdentityKey,
		Status:            inventory.FileStatusAvailable,
	})
	if err != nil {
		t.Fatalf("upsert inventory file: %v", err)
	}

	asset, err := fx.inventory.CreateAsset(ctx, inventory.CreateAssetInput{
		LibraryID:       fx.library.ID,
		AssetType:       inventory.AssetTypeMain,
		DisplayName:     legacyItem.Title,
		DurationSeconds: &durationSecondsFloat,
		Status:          inventory.AssetStatusAvailable,
		ProbeStatus:     legacyFile.ProbeStatus,
	})
	if err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if _, err := fx.inventory.LinkAssetToFile(ctx, inventory.LinkAssetFileInput{
		AssetID:   asset.ID,
		FileID:    inventoryFile.ID,
		Role:      inventory.FileRoleSource,
		PartIndex: 0,
	}); err != nil {
		t.Fatalf("link asset to file: %v", err)
	}
	if _, err := fx.inventory.LinkAssetToItem(ctx, inventory.LinkAssetItemInput{
		AssetID:      asset.ID,
		ItemID:       catalogItem.ID,
		Role:         inventory.AssetItemRolePrimary,
		SegmentIndex: 0,
		Source:       "legacy_backfill",
	}); err != nil {
		t.Fatalf("link asset to item: %v", err)
	}

	completedAt := time.Date(2026, time.April, 25, 9, 30, 0, 0, time.UTC)
	lastPlayedAt := completedAt.Add(-15 * time.Minute)
	durationSecondsInt := 1000
	legacyProgress := database.PlaybackProgress{
		UserID:          fx.user.ID,
		MediaItemID:     legacyItem.ID,
		MediaFileID:     &legacyFile.ID,
		PositionSeconds: 960,
		DurationSeconds: &durationSecondsInt,
		Watched:         true,
		CompletedAt:     &completedAt,
		LastPlayedAt:    &lastPlayedAt,
	}
	if err := fx.db.WithContext(ctx).Create(&legacyProgress).Error; err != nil {
		t.Fatalf("create legacy playback progress: %v", err)
	}

	run, err := fx.catalog.CreateLegacyBackfillRun(ctx, catalog.CreateLegacyBackfillRunInput{
		Scope:             catalog.LegacyBackfillScope{Kind: catalog.LegacyBackfillScopeLibrary, LibraryID: &fx.library.ID},
		TriggeredByUserID: fx.user.ID,
	})
	if err != nil {
		t.Fatalf("create backfill run: %v", err)
	}

	assertTableCount(t, ctx, fx.db, &database.UserItemData{}, 0)
	assertTableCount(t, ctx, fx.db, &database.ItemRollup{}, 1)
	assertTableCount(t, ctx, fx.db, &database.CatalogSearchDocument{}, 1)

	if err := fx.catalog.RunLegacyBackfill(ctx, catalog.LegacyBackfillPayload{RunID: run.ID, LibraryID: &fx.library.ID}); err != nil {
		t.Fatalf("run legacy backfill: %v", err)
	}

	var userItemData database.UserItemData
	if err := fx.db.WithContext(ctx).First(&userItemData).Error; err != nil {
		t.Fatalf("load user_item_data: %v", err)
	}
	if userItemData.UserID != fx.user.ID || userItemData.ItemID != catalogItem.ID {
		t.Fatalf("unexpected user item data ids: %#v", userItemData)
	}
	if userItemData.AssetID == nil || *userItemData.AssetID != asset.ID {
		t.Fatalf("expected asset id %d, got %#v", asset.ID, userItemData.AssetID)
	}
	if userItemData.PositionSeconds != legacyProgress.PositionSeconds {
		t.Fatalf("expected position %d, got %d", legacyProgress.PositionSeconds, userItemData.PositionSeconds)
	}
	if userItemData.PlayCount != 1 {
		t.Fatalf("expected play_count=1, got %#v", userItemData)
	}
	if userItemData.PlayedPercentage == nil || math.Abs(*userItemData.PlayedPercentage-96) > 0.001 {
		t.Fatalf("expected played_percentage near 96, got %#v", userItemData.PlayedPercentage)
	}
	if userItemData.LastPlayedAt == nil || !userItemData.LastPlayedAt.Equal(lastPlayedAt) {
		t.Fatalf("expected last_played_at %s, got %#v", lastPlayedAt, userItemData.LastPlayedAt)
	}
	if userItemData.CompletedAt == nil || !userItemData.CompletedAt.Equal(completedAt) {
		t.Fatalf("expected completed_at %s, got %#v", completedAt, userItemData.CompletedAt)
	}

	assertTableCount(t, ctx, fx.db, &database.ItemRollup{}, 1)
	assertTableCount(t, ctx, fx.db, &database.CatalogSearchDocument{}, 1)

	report, err := fx.catalog.GetLegacyBackfillRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("load backfill run: %v", err)
	}
	if report.Status != catalog.LegacyBackfillStatusCompleted {
		t.Fatalf("expected run status %q, got %#v", catalog.LegacyBackfillStatusCompleted, report)
	}
}

func TestLegacyBackfillProgressResolvesDuplicateEpisodeCandidates(t *testing.T) {
	t.Parallel()

	fx := newLegacyBackfillFixture(t)
	ctx := fx.ctx

	year := 2024
	seasonNumber := 1
	episodeNumber := 1
	runtimeSeconds := 1800
	canonical := database.MediaItem{
		LibraryID:        fx.library.ID,
		Type:             catalog.ItemTypeEpisode,
		Title:            "Pilot",
		SeriesTitle:      "Duplicate Progress Show",
		SourcePath:       "/library/shows/duplicate-progress/canonical/pilot.mkv",
		SeasonNumber:     &seasonNumber,
		EpisodeNumber:    &episodeNumber,
		Year:             &year,
		RuntimeSeconds:   &runtimeSeconds,
		MatchStatus:      "matched",
		MetadataProvider: "tmdb",
		ExternalID:       "tv:999",
		Status:           "ready",
	}
	duplicate := database.MediaItem{
		LibraryID:      fx.library.ID,
		Type:           catalog.ItemTypeEpisode,
		Title:          "Pilot",
		SeriesTitle:    "Duplicate Progress Show",
		SourcePath:     "/library/shows/duplicate-progress/duplicate/pilot.mkv",
		SeasonNumber:   &seasonNumber,
		EpisodeNumber:  &episodeNumber,
		Year:           &year,
		RuntimeSeconds: &runtimeSeconds,
		MatchStatus:    "matched",
		Status:         "ready",
	}
	for _, item := range []*database.MediaItem{&canonical, &duplicate} {
		if err := fx.db.WithContext(ctx).Create(item).Error; err != nil {
			t.Fatalf("create legacy episode %q: %v", item.SourcePath, err)
		}
	}

	durationSeconds := 1800.0
	canonicalFile := database.MediaFile{LibraryID: fx.library.ID, MediaItemID: &canonical.ID, StoragePath: canonical.SourcePath, StableIdentityKey: "stable:duplicate-progress:canonical", ProviderName: "local", ProbeStatus: "complete", DurationSeconds: &durationSeconds}
	duplicateFile := database.MediaFile{LibraryID: fx.library.ID, MediaItemID: &duplicate.ID, StoragePath: duplicate.SourcePath, StableIdentityKey: "stable:duplicate-progress:duplicate", ProviderName: "local", ProbeStatus: "complete", DurationSeconds: &durationSeconds}
	for _, file := range []*database.MediaFile{&canonicalFile, &duplicateFile} {
		if err := fx.db.WithContext(ctx).Create(file).Error; err != nil {
			t.Fatalf("create legacy media file %q: %v", file.StoragePath, err)
		}
	}

	completedAt := time.Date(2026, time.April, 25, 11, 0, 0, 0, time.UTC)
	legacyProgress := database.PlaybackProgress{
		UserID:          fx.user.ID,
		MediaItemID:     duplicate.ID,
		MediaFileID:     &duplicateFile.ID,
		PositionSeconds: 600,
		DurationSeconds: &runtimeSeconds,
		Watched:         false,
		CompletedAt:     &completedAt,
	}
	if err := fx.db.WithContext(ctx).Create(&legacyProgress).Error; err != nil {
		t.Fatalf("create duplicate playback progress: %v", err)
	}

	run, err := fx.catalog.CreateLegacyBackfillRun(ctx, catalog.CreateLegacyBackfillRunInput{
		Scope:             catalog.LegacyBackfillScope{Kind: catalog.LegacyBackfillScopeLibrary, LibraryID: &fx.library.ID},
		TriggeredByUserID: fx.user.ID,
	})
	if err != nil {
		t.Fatalf("create backfill run: %v", err)
	}

	if err := fx.catalog.RunLegacyBackfill(ctx, catalog.LegacyBackfillPayload{RunID: run.ID, LibraryID: &fx.library.ID}); err != nil {
		t.Fatalf("run legacy backfill: %v", err)
	}

	var episodes []database.CatalogItem
	if err := fx.db.WithContext(ctx).Where("library_id = ? AND type = ?", fx.library.ID, catalog.ItemTypeEpisode).Order("id asc").Find(&episodes).Error; err != nil {
		t.Fatalf("list canonical catalog episodes: %v", err)
	}
	if len(episodes) != 1 {
		t.Fatalf("expected one canonical episode after duplicate collapse, got %#v", episodes)
	}

	var userItemData []database.UserItemData
	if err := fx.db.WithContext(ctx).Order("id asc").Find(&userItemData).Error; err != nil {
		t.Fatalf("list user item data: %v", err)
	}
	if len(userItemData) != 1 {
		t.Fatalf("expected one migrated progress row, got %#v", userItemData)
	}
	if userItemData[0].ItemID != episodes[0].ID || userItemData[0].AssetID == nil {
		t.Fatalf("expected duplicate progress to map onto canonical episode %#v, got %#v", episodes[0], userItemData[0])
	}

	report, err := fx.catalog.GetLegacyBackfillRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("load backfill run: %v", err)
	}
	foundDuplicateEntry := false
	foundProgressSuccess := false
	for _, entry := range report.Entries {
		if entry.EntryType == catalog.LegacyBackfillEntryTypeDuplicateEpisodeCandidate && entry.LegacyMediaItemID != nil && *entry.LegacyMediaItemID == duplicate.ID {
			foundDuplicateEntry = true
		}
		if entry.EntryType == catalog.LegacyBackfillEntryTypeSuccess && entry.LegacyMediaItemID != nil && *entry.LegacyMediaItemID == duplicate.ID && entry.AssetID != nil {
			foundProgressSuccess = true
		}
	}
	if !foundDuplicateEntry || !foundProgressSuccess {
		t.Fatalf("expected duplicate candidate and migrated progress entries, got %#v", report.Entries)
	}
}

type legacyBackfillFixture struct {
	ctx       context.Context
	db        *gorm.DB
	catalog   *catalog.Service
	inventory *inventory.Service
	jobs      *jobs.Service
	settings  *settings.Service
	runner    *worker.Runner
	library   database.Library
	user      database.User
}

func newLegacyBackfillFixture(t *testing.T) *legacyBackfillFixture {
	t.Helper()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	ctx := context.Background()
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, config.MetadataConfig{})
	catalogSvc := catalog.NewService(db)
	fixture := &legacyBackfillFixture{
		ctx:       ctx,
		db:        db,
		catalog:   catalogSvc,
		inventory: inventory.NewService(db),
		jobs:      jobsSvc,
		settings:  settingsSvc,
		runner:    worker.NewRunner(config.WorkerConfig{}, jobsSvc, nil, nil, nil, settingsSvc, catalogSvc),
	}

	mediaSource := database.MediaSource{
		Name:       "Local Source",
		Provider:   "local",
		StorageRef: "local",
		RootPath:   "/library",
	}
	if err := db.WithContext(ctx).Create(&mediaSource).Error; err != nil {
		t.Fatalf("create media source: %v", err)
	}

	fixture.library = database.Library{
		Name:          "Legacy Library",
		Type:          "mixed",
		MediaSourceID: mediaSource.ID,
		RootPath:      "/library",
		Status:        "active",
	}
	if err := db.WithContext(ctx).Create(&fixture.library).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}

	fixture.user = database.User{Username: "admin", PasswordHash: "hash", Role: "admin"}
	if err := db.WithContext(ctx).Create(&fixture.user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	return fixture
}

func assertTableCount(t *testing.T, ctx context.Context, db *gorm.DB, model any, want int64) {
	t.Helper()

	var count int64
	if err := db.WithContext(ctx).Model(model).Count(&count).Error; err != nil {
		t.Fatalf("count rows for %T: %v", model, err)
	}
	if count != want {
		t.Fatalf("expected %d rows for %T, got %d", want, model, count)
	}
}
