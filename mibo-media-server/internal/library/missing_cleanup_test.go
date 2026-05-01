package library

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/providers"
	"gorm.io/gorm"
)

func TestCleanupMissingMediaHardDeletesRowsOlderThanRetention(t *testing.T) {
	ctx, db, svc, libraryRecord := newMissingCleanupHarness(t, 24*time.Hour)
	oldMissing := time.Now().UTC().Add(-48 * time.Hour)

	item, asset, file := seedMissingCleanupGraph(t, ctx, db, libraryRecord.ID, filepath.Join(libraryRecord.RootPath, "gone.mkv"), oldMissing)
	result, err := svc.CleanupMissingMedia(ctx, libraryRecord.ID, libraryRecord.RootPath)
	if err != nil {
		t.Fatalf("cleanup missing media: %v", err)
	}
	if result.FilesDeleted != 1 || result.AssetsDeleted != 1 || result.CatalogItemsDeleted != 1 {
		t.Fatalf("unexpected cleanup result: %#v", result)
	}
	assertMissingCleanupRowGone(t, ctx, db, &database.InventoryFile{}, file.ID)
	assertMissingCleanupRowGone(t, ctx, db, &database.MediaAsset{}, asset.ID)
	assertMissingCleanupRowGone(t, ctx, db, &database.CatalogItem{}, item.ID)
	assertRawTableCount(t, db, "jobs", "kind = ?", 1, JobKindCatalogRefreshLibraryProjection)
}

func TestCleanupMissingMediaPreservesRowsYoungerThanRetention(t *testing.T) {
	ctx, db, svc, libraryRecord := newMissingCleanupHarness(t, 24*time.Hour)
	recentMissing := time.Now().UTC().Add(-time.Hour)

	item, asset, file := seedMissingCleanupGraph(t, ctx, db, libraryRecord.ID, filepath.Join(libraryRecord.RootPath, "recent.mkv"), recentMissing)
	result, err := svc.CleanupMissingMedia(ctx, libraryRecord.ID, libraryRecord.RootPath)
	if err != nil {
		t.Fatalf("cleanup missing media: %v", err)
	}
	if result.FilesDeleted != 0 || result.AssetsDeleted != 0 || result.CatalogItemsDeleted != 0 {
		t.Fatalf("expected no cleanup, got %#v", result)
	}
	assertMissingCleanupRowExists(t, ctx, db, &database.InventoryFile{}, file.ID)
	assertMissingCleanupRowExists(t, ctx, db, &database.MediaAsset{}, asset.ID)
	assertMissingCleanupRowExists(t, ctx, db, &database.CatalogItem{}, item.ID)
}

func TestCleanupMissingMediaDeletesUserAndGovernanceState(t *testing.T) {
	ctx, db, svc, libraryRecord := newMissingCleanupHarness(t, 0)
	missingAt := time.Now().UTC().Add(-time.Hour)

	item, asset, file := seedMissingCleanupGraph(t, ctx, db, libraryRecord.ID, filepath.Join(libraryRecord.RootPath, "curated.mkv"), missingAt)
	person := database.Person{Name: "Actor"}
	tag := database.Tag{Kind: "genre", Name: "Drama"}
	fetchedAt := time.Now().UTC()
	if err := db.WithContext(ctx).Create(&person).Error; err != nil {
		t.Fatalf("create person: %v", err)
	}
	if err := db.WithContext(ctx).Create(&tag).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}
	rows := []any{
		&database.UserItemData{UserID: 1, ItemID: item.ID, AssetID: &asset.ID, PositionSeconds: 42, Favorite: true},
		&database.ItemRollup{ItemID: item.ID, MissingCount: 1},
		&database.CatalogSearchDocument{ItemID: item.ID, LibraryID: libraryRecord.ID, ItemType: catalog.ItemTypeMovie, Title: item.Title, AvailabilityStatus: catalog.AvailabilityMissing},
		&database.ItemImage{ItemID: item.ID, ImageType: "poster", URL: "https://example.test/poster.jpg", IsSelected: true},
		&database.ItemPerson{ItemID: item.ID, PersonID: person.ID, Role: "actor"},
		&database.ItemTag{ItemID: item.ID, TagID: tag.ID},
		&database.CatalogExternalID{ItemID: item.ID, Provider: "tmdb", ProviderType: catalog.ItemTypeMovie, ExternalID: "123", IsPrimary: true},
		&database.CatalogIdentity{ItemID: item.ID, Provider: catalog.IdentityProviderScanner, IdentityType: catalog.ItemTypeMovie, IdentityKey: "movie:curated"},
		&database.MetadataSource{ItemID: item.ID, SourceType: "manual", SourceName: "manual", FetchedAt: fetchedAt},
		&database.MetadataFieldState{ItemID: item.ID, FieldKey: "title", ValueJSON: `"Manual"`, IsLocked: true},
		&database.MetadataOperation{Operation: "manual_apply", OriginItemID: item.ID, TargetItemID: item.ID, LibraryID: libraryRecord.ID, Status: "applied", StartedAt: fetchedAt},
		&database.MediaStream{FileID: file.ID, StreamIndex: 0, StreamType: "video"},
		&database.ScanExclusion{LibraryID: libraryRecord.ID, StorageProvider: file.StorageProvider, StoragePath: file.StoragePath, Reason: "sample", Enabled: true},
	}
	for _, row := range rows {
		if err := db.WithContext(ctx).Create(row).Error; err != nil {
			t.Fatalf("create dependent row %T: %v", row, err)
		}
	}

	result, err := svc.CleanupMissingMedia(ctx, libraryRecord.ID, libraryRecord.RootPath)
	if err != nil {
		t.Fatalf("cleanup missing media: %v", err)
	}
	if result.DependentRowsDeleted == 0 {
		t.Fatalf("expected dependent rows to be deleted, got %#v", result)
	}
	assertTableCount(t, ctx, db, &database.UserItemData{}, 0)
	assertTableCount(t, ctx, db, &database.ItemRollup{}, 0)
	assertTableCount(t, ctx, db, &database.CatalogSearchDocument{}, 0)
	assertTableCount(t, ctx, db, &database.ItemImage{}, 0)
	assertTableCount(t, ctx, db, &database.ItemPerson{}, 0)
	assertTableCount(t, ctx, db, &database.ItemTag{}, 0)
	assertTableCount(t, ctx, db, &database.CatalogExternalID{}, 0)
	assertTableCount(t, ctx, db, &database.CatalogIdentity{}, 0)
	assertTableCount(t, ctx, db, &database.MetadataSource{}, 0)
	assertTableCount(t, ctx, db, &database.MetadataFieldState{}, 0)
	assertTableCount(t, ctx, db, &database.MetadataOperation{}, 0)
	assertTableCount(t, ctx, db, &database.MediaStream{}, 0)
	assertTableCount(t, ctx, db, &database.ScanExclusion{}, 0)
}

func TestCleanupMissingMediaPreservesAvailableBranches(t *testing.T) {
	ctx, db, svc, libraryRecord := newMissingCleanupHarness(t, 0)
	missingAt := time.Now().UTC().Add(-time.Hour)

	series := database.CatalogItem{LibraryID: libraryRecord.ID, Type: catalog.ItemTypeSeries, Title: "Show", SortKey: "show", AvailabilityStatus: catalog.AvailabilityMissing, MissingSince: &missingAt, GovernanceStatus: catalog.GovernancePending}
	if err := db.WithContext(ctx).Create(&series).Error; err != nil {
		t.Fatalf("create series: %v", err)
	}
	season := database.CatalogItem{LibraryID: libraryRecord.ID, Type: catalog.ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", SortKey: "show s1", AvailabilityStatus: catalog.AvailabilityMissing, MissingSince: &missingAt, GovernanceStatus: catalog.GovernancePending}
	if err := db.WithContext(ctx).Create(&season).Error; err != nil {
		t.Fatalf("create season: %v", err)
	}
	missingEpisode := database.CatalogItem{LibraryID: libraryRecord.ID, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Title: "Missing", SortKey: "missing", AvailabilityStatus: catalog.AvailabilityMissing, MissingSince: &missingAt, GovernanceStatus: catalog.GovernancePending}
	availableEpisode := database.CatalogItem{LibraryID: libraryRecord.ID, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Title: "Available", SortKey: "available", AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernancePending}
	if err := db.WithContext(ctx).Create(&missingEpisode).Error; err != nil {
		t.Fatalf("create missing episode: %v", err)
	}
	if err := db.WithContext(ctx).Create(&availableEpisode).Error; err != nil {
		t.Fatalf("create available episode: %v", err)
	}
	missingAsset, missingFile := seedMissingCleanupAssetFile(t, ctx, db, libraryRecord.ID, filepath.Join(libraryRecord.RootPath, "missing.mkv"), missingAt)
	availableAsset, availableFile := seedAvailableCleanupAssetFile(t, ctx, db, libraryRecord.ID, filepath.Join(libraryRecord.RootPath, "available.mkv"))
	seedAssetItem(t, ctx, db, missingAsset.ID, missingEpisode.ID)
	seedAssetItem(t, ctx, db, availableAsset.ID, availableEpisode.ID)

	movie := database.CatalogItem{LibraryID: libraryRecord.ID, Type: catalog.ItemTypeMovie, Title: "Versioned", SortKey: "versioned", AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernancePending}
	if err := db.WithContext(ctx).Create(&movie).Error; err != nil {
		t.Fatalf("create movie: %v", err)
	}
	missingVersionAsset, missingVersionFile := seedMissingCleanupAssetFile(t, ctx, db, libraryRecord.ID, filepath.Join(libraryRecord.RootPath, "movie-missing.mkv"), missingAt)
	availableVersionAsset, availableVersionFile := seedAvailableCleanupAssetFile(t, ctx, db, libraryRecord.ID, filepath.Join(libraryRecord.RootPath, "movie-available.mkv"))
	seedAssetItem(t, ctx, db, missingVersionAsset.ID, movie.ID)
	seedAssetItem(t, ctx, db, availableVersionAsset.ID, movie.ID)

	result, err := svc.CleanupMissingMedia(ctx, libraryRecord.ID, libraryRecord.RootPath)
	if err != nil {
		t.Fatalf("cleanup missing media: %v", err)
	}
	if result.FilesDeleted != 2 || result.AssetsDeleted != 2 || result.CatalogItemsDeleted != 1 {
		t.Fatalf("unexpected cleanup result: %#v", result)
	}
	assertMissingCleanupRowGone(t, ctx, db, &database.CatalogItem{}, missingEpisode.ID)
	assertMissingCleanupRowExists(t, ctx, db, &database.CatalogItem{}, series.ID)
	assertMissingCleanupRowExists(t, ctx, db, &database.CatalogItem{}, season.ID)
	assertMissingCleanupRowExists(t, ctx, db, &database.CatalogItem{}, availableEpisode.ID)
	assertMissingCleanupRowExists(t, ctx, db, &database.CatalogItem{}, movie.ID)
	assertMissingCleanupRowGone(t, ctx, db, &database.InventoryFile{}, missingFile.ID)
	assertMissingCleanupRowGone(t, ctx, db, &database.InventoryFile{}, missingVersionFile.ID)
	assertMissingCleanupRowExists(t, ctx, db, &database.InventoryFile{}, availableFile.ID)
	assertMissingCleanupRowExists(t, ctx, db, &database.InventoryFile{}, availableVersionFile.ID)
}

func newMissingCleanupHarness(t *testing.T, retention time.Duration) (context.Context, *gorm.DB, *Service, database.Library) {
	t.Helper()
	ctx := context.Background()
	rootPath := t.TempDir()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	cfg := config.Config{Local: config.LocalStorageConfig{RootPath: rootPath}, Cleanup: config.CleanupConfig{MissingCleanupEnabled: true, MissingRetention: retention, MissingCleanupBatchSize: 100}}
	svc := NewService(cfg, db, providers.NewRegistry(cfg), jobs.NewService(db))
	source, err := svc.CreateMediaSource(ctx, CreateMediaSourceInput{Provider: "local", Name: "Local", RootPath: rootPath})
	if err != nil {
		t.Fatalf("create media source: %v", err)
	}
	libraryRecord, _, err := svc.CreateLibrary(ctx, CreateLibraryInput{Name: "Movies", Type: LibraryTypeMovies, MediaSourceID: source.ID, RootPath: rootPath})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	return ctx, db, svc, libraryRecord
}

func seedMissingCleanupGraph(t *testing.T, ctx context.Context, db *gorm.DB, libraryID uint, storagePath string, missingAt time.Time) (database.CatalogItem, database.MediaAsset, database.InventoryFile) {
	t.Helper()
	item := database.CatalogItem{LibraryID: libraryID, Type: catalog.ItemTypeMovie, Title: filepath.Base(storagePath), SortKey: filepath.Base(storagePath), AvailabilityStatus: catalog.AvailabilityMissing, MissingSince: &missingAt, GovernanceStatus: catalog.GovernancePending}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	asset, file := seedMissingCleanupAssetFile(t, ctx, db, libraryID, storagePath, missingAt)
	seedAssetItem(t, ctx, db, asset.ID, item.ID)
	return item, asset, file
}

func seedMissingCleanupAssetFile(t *testing.T, ctx context.Context, db *gorm.DB, libraryID uint, storagePath string, missingAt time.Time) (database.MediaAsset, database.InventoryFile) {
	t.Helper()
	asset := database.MediaAsset{LibraryID: libraryID, Status: inventory.AssetStatusMissing, MissingSince: &missingAt, ProbeStatus: "ready"}
	file := database.InventoryFile{LibraryID: libraryID, StorageProvider: "local", StoragePath: storagePath, Status: inventory.FileStatusMissing, MissingSince: &missingAt}
	if err := db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: inventory.FileRoleSource}).Error; err != nil {
		t.Fatalf("create asset file: %v", err)
	}
	return asset, file
}

func seedAvailableCleanupAssetFile(t *testing.T, ctx context.Context, db *gorm.DB, libraryID uint, storagePath string) (database.MediaAsset, database.InventoryFile) {
	t.Helper()
	asset := database.MediaAsset{LibraryID: libraryID, Status: inventory.AssetStatusAvailable, ProbeStatus: "ready"}
	file := database.InventoryFile{LibraryID: libraryID, StorageProvider: "local", StoragePath: storagePath, Status: inventory.FileStatusAvailable}
	if err := db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create available asset: %v", err)
	}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create available file: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: inventory.FileRoleSource}).Error; err != nil {
		t.Fatalf("create available asset file: %v", err)
	}
	return asset, file
}

func seedAssetItem(t *testing.T, ctx context.Context, db *gorm.DB, assetID uint, itemID uint) {
	t.Helper()
	if err := db.WithContext(ctx).Create(&database.AssetItem{AssetID: assetID, ItemID: itemID, Role: inventory.AssetItemRolePrimary}).Error; err != nil {
		t.Fatalf("create asset item: %v", err)
	}
}

func assertMissingCleanupRowGone(t *testing.T, ctx context.Context, db *gorm.DB, model any, id uint) {
	t.Helper()
	var count int64
	if err := db.WithContext(ctx).Model(model).Where("id = ?", id).Count(&count).Error; err != nil {
		t.Fatalf("count missing cleanup row %T: %v", model, err)
	}
	if count != 0 {
		t.Fatalf("expected %T id %d to be hard deleted, found %d rows", model, id, count)
	}
}

func assertMissingCleanupRowExists(t *testing.T, ctx context.Context, db *gorm.DB, model any, id uint) {
	t.Helper()
	var count int64
	if err := db.WithContext(ctx).Model(model).Where("id = ?", id).Count(&count).Error; err != nil {
		t.Fatalf("count missing cleanup row %T: %v", model, err)
	}
	if count != 1 {
		t.Fatalf("expected %T id %d to exist, found %d rows", model, id, count)
	}
}
