package database

import (
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"gorm.io/gorm"
)

func TestDatabaseOpenFreshCatalogDatabaseMigratesCatalogTables(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "mibo.db")
	db, err := Open(config.DatabaseConfig{Driver: "sqlite", DSN: dbPath})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	assertTablesExist(t, db, requiredFreshStartupModels())
	assertCatalogIndexesExist(t, db)
	assertMediaStreamTechnicalColumnsExist(t, db)
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
		&Job{},
		&JobActiveIntent{},
		&Schedule{},
		&ScheduleRun{},
		&User{},
		&Session{},
		&SystemSetting{},
		&SearchHistory{},
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
		{&SystemSetting{}, "idx_system_setting_category_key"},
	}

	for _, index := range requiredIndexes {
		if !db.Migrator().HasIndex(index.model, index.name) {
			t.Fatalf("expected index %q to exist for %T", index.name, index.model)
		}
	}
}

func assertMediaStreamTechnicalColumnsExist(t *testing.T, db *gorm.DB) {
	t.Helper()

	requiredColumns := []string{
		"profile",
		"level",
		"avg_frame_rate",
		"r_frame_rate",
		"field_order",
		"color_space",
		"bit_depth",
		"pixel_format",
		"reference_frames",
		"channel_layout",
		"sample_rate",
		"bit_rate",
	}
	for _, column := range requiredColumns {
		if !db.Migrator().HasColumn(&MediaStream{}, column) {
			t.Fatalf("expected media_streams.%s column to exist", column)
		}
	}
}
