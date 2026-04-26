package database

import (
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"gorm.io/gorm"
)

func TestCatalogKernelTablesAreMigrated(t *testing.T) {
	db := openCatalogTestDB(t)

	for _, model := range requiredFreshStartupModels() {
		if !db.Migrator().HasTable(model) {
			t.Fatalf("expected table for %T to exist", model)
		}
	}
}

func TestDatabaseOpenMigratesCatalogIndexes(t *testing.T) {
	db := openCatalogTestDB(t)

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
		{&AssetItem{}, "idx_asset_items_asset_item_role_segment"},
		{&AssetItem{}, "idx_asset_items_item_role"},
		{&InventoryFile{}, "idx_inventory_file_storage_path"},
		{&InventoryFile{}, "idx_inventory_files_library_status_path"},
		{&AssetFile{}, "idx_asset_files_asset_file_role_part"},
		{&AssetFile{}, "idx_asset_files_asset_part"},
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

func openCatalogTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	return db
}
