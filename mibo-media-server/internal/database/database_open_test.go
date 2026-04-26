package database

import (
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDatabaseOpenFreshCatalogDatabaseMigratesLegacyAndCatalogTables(t *testing.T) {
	dbPath := newCatalogDatabasePath(t)
	db := openDatabaseForMigrationAssertions(t, dbPath)

	assertTablesExist(t, db, requiredFreshStartupModels())
	assertCatalogIndexesExist(t, db)
}

func TestDatabaseOpenMigratesLegacyOnlyCatalogDatabase(t *testing.T) {
	dbPath := newCatalogDatabasePath(t)
	seedLegacyOnlyDatabase(t, dbPath)

	db := openDatabaseForMigrationAssertions(t, dbPath)
	assertTablesExist(t, db, requiredFreshStartupModels())
	assertLegacyRowsPreserved(t, db)
	assertCatalogIndexesExist(t, db)
}

func TestDatabaseOpenCatalogMigrationIsIdempotent(t *testing.T) {
	dbPath := newCatalogDatabasePath(t)
	seedLegacyOnlyDatabase(t, dbPath)

	first := openDatabaseForMigrationAssertions(t, dbPath)
	assertCatalogIndexesExist(t, first)

	second := openDatabaseForMigrationAssertions(t, dbPath)
	assertTablesExist(t, second, requiredFreshStartupModels())
	assertLegacyRowsPreserved(t, second)
	assertCatalogIndexesExist(t, second)
}

func newCatalogDatabasePath(t *testing.T) string {
	t.Helper()

	return filepath.Join(t.TempDir(), "mibo.db")
}

func openDatabaseForMigrationAssertions(t *testing.T, dbPath string) *gorm.DB {
	t.Helper()

	db, err := Open(config.DatabaseConfig{Driver: "sqlite", DSN: dbPath})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	return db
}

func seedLegacyOnlyDatabase(t *testing.T, dbPath string) {
	t.Helper()

	legacyDB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open legacy seed database: %v", err)
	}

	legacyModels := []any{
		&MediaSource{},
		&Library{},
		&MediaItem{},
		&MediaFile{},
		&TVSeasonMetadataCache{},
		&TVEpisodeMetadataCache{},
		&Job{},
		&JobActiveIntent{},
		&Schedule{},
		&ScheduleRun{},
		&User{},
		&Session{},
		&PlaybackProgress{},
		&SystemSetting{},
		&SearchHistory{},
		&SearchDocument{},
	}
	if err := legacyDB.AutoMigrate(legacyModels...); err != nil {
		t.Fatalf("migrate legacy-only database: %v", err)
	}

	source := MediaSource{Name: "Legacy Source", Provider: "local", StorageRef: "/media", RootPath: "/media"}
	if err := legacyDB.Create(&source).Error; err != nil {
		t.Fatalf("create legacy media source: %v", err)
	}

	library := Library{Name: "Legacy Library", Type: "movies", MediaSourceID: source.ID, RootPath: "/media/movies"}
	if err := legacyDB.Create(&library).Error; err != nil {
		t.Fatalf("create legacy library: %v", err)
	}

	item := MediaItem{LibraryID: library.ID, Type: "movie", Title: "Legacy Movie", SourcePath: "/media/movies/Legacy.Movie.2024.mkv"}
	if err := legacyDB.Create(&item).Error; err != nil {
		t.Fatalf("create legacy media item: %v", err)
	}

	file := MediaFile{LibraryID: library.ID, MediaItemID: &item.ID, StoragePath: "/media/movies/Legacy.Movie.2024.mkv"}
	if err := legacyDB.Create(&file).Error; err != nil {
		t.Fatalf("create legacy media file: %v", err)
	}

	setting := SystemSetting{Category: "metadata", Key: "language", Value: "en-US"}
	if err := legacyDB.Create(&setting).Error; err != nil {
		t.Fatalf("create legacy system setting: %v", err)
	}

	searchDocument := SearchDocument{MediaItemID: item.ID, LibraryID: library.ID, MediaType: "movie", Title: item.Title}
	if err := legacyDB.Create(&searchDocument).Error; err != nil {
		t.Fatalf("create legacy search document: %v", err)
	}
}

func requiredFreshStartupModels() []any {
	return []any{
		&MediaSource{},
		&Library{},
		&CatalogItem{},
		&CatalogExternalID{},
		&MetadataSource{},
		&MetadataFieldState{},
		&ItemImage{},
		&Person{},
		&ItemPerson{},
		&Tag{},
		&ItemTag{},
		&MediaAsset{},
		&AssetItem{},
		&InventoryFile{},
		&AssetFile{},
		&MediaStream{},
		&UserItemData{},
		&ItemRollup{},
		&CatalogSearchDocument{},
		&MediaItem{},
		&MediaFile{},
		&TVSeasonMetadataCache{},
		&TVEpisodeMetadataCache{},
		&Job{},
		&JobActiveIntent{},
		&Schedule{},
		&ScheduleRun{},
		&User{},
		&Session{},
		&PlaybackProgress{},
		&SystemSetting{},
		&SearchHistory{},
		&SearchDocument{},
	}
}

func assertTablesExist(t *testing.T, db *gorm.DB, models []any) {
	t.Helper()

	for _, model := range models {
		if !db.Migrator().HasTable(model) {
			t.Fatalf("expected table for %T to exist", model)
		}
	}
}

func assertCatalogIndexesExist(t *testing.T, db *gorm.DB) {
	t.Helper()

	requiredIndexes := []struct {
		model any
		name  string
	}{
		{&CatalogItem{}, "idx_catalog_items_library_type_availability_sort"},
		{&CatalogItem{}, "idx_catalog_items_parent_order"},
		{&CatalogItem{}, "idx_catalog_items_root_type_order"},
		{&CatalogSearchDocument{}, "idx_catalog_search_documents_library_type_availability_title"},
		{&CatalogExternalID{}, "idx_catalog_external_identity"},
		{&MetadataFieldState{}, "idx_metadata_field_state_item_field"},
		{&AssetItem{}, "idx_asset_items_item_role"},
		{&AssetItem{}, "idx_asset_items_asset_item_role_segment"},
		{&AssetFile{}, "idx_asset_files_asset_part"},
		{&AssetFile{}, "idx_asset_files_asset_file_role_part"},
		{&InventoryFile{}, "idx_inventory_file_storage_path"},
		{&InventoryFile{}, "idx_inventory_files_library_status_path"},
		{&MediaStream{}, "idx_media_stream_file_index"},
		{&UserItemData{}, "idx_user_item_data_user_item_asset"},
		{&TVSeasonMetadataCache{}, "idx_tv_season_cache_lookup"},
		{&TVEpisodeMetadataCache{}, "idx_tv_episode_cache_lookup"},
		{&PlaybackProgress{}, "idx_user_media_item"},
		{&SystemSetting{}, "idx_system_setting_category_key"},
	}

	for _, index := range requiredIndexes {
		if !db.Migrator().HasIndex(index.model, index.name) {
			t.Fatalf("expected index %q to exist for %T", index.name, index.model)
		}
	}
}

func assertLegacyRowsPreserved(t *testing.T, db *gorm.DB) {
	t.Helper()

	assertModelCount(t, db, &MediaSource{}, 1)
	assertModelCount(t, db, &Library{}, 1)
	assertModelCount(t, db, &MediaItem{}, 1)
	assertModelCount(t, db, &MediaFile{}, 1)
	assertModelCount(t, db, &SystemSetting{}, 1)
	assertModelCount(t, db, &SearchDocument{}, 1)
	assertModelCount(t, db, &CatalogItem{}, 0)

	var item MediaItem
	if err := db.First(&item).Error; err != nil {
		t.Fatalf("load preserved legacy media item: %v", err)
	}
	if item.Title != "Legacy Movie" {
		t.Fatalf("expected preserved legacy title %q, got %q", "Legacy Movie", item.Title)
	}
}

func assertModelCount(t *testing.T, db *gorm.DB, model any, expected int64) {
	t.Helper()

	var count int64
	if err := db.Model(model).Count(&count).Error; err != nil {
		t.Fatalf("count %T rows: %v", model, err)
	}
	if count != expected {
		t.Fatalf("expected %d rows for %T, got %d", expected, model, count)
	}
}
