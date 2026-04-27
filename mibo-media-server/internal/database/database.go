package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(cfg config.DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Driver {
	case "sqlite":
		if err := ensureSQLiteDir(cfg.DSN); err != nil {
			return nil, err
		}
		dialector = sqlite.Open(cfg.DSN)
	case "postgres":
		dialector = postgres.Open(cfg.DSN)
	default:
		return nil, fmt.Errorf("unsupported database driver %q", cfg.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.New(log.New(os.Stdout, "", log.LstdFlags), logger.Config{
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
		}),
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
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
	); err != nil {
		return nil, err
	}

	if err := validateCatalogKernelUniqueness(db); err != nil {
		return nil, err
	}

	if err := ensureCatalogKernelIndexes(db); err != nil {
		return nil, err
	}

	return db, nil
}

func ensureCatalogKernelIndexes(db *gorm.DB) error {
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
		if db.Migrator().HasIndex(index.model, index.name) {
			continue
		}

		if err := db.Migrator().CreateIndex(index.model, index.name); err != nil {
			return fmt.Errorf("create index %s: %w", index.name, err)
		}
	}

	return nil
}

func validateCatalogKernelUniqueness(db *gorm.DB) error {
	checks := []struct {
		label      string
		table      string
		where      string
		columns    []string
		groupBy    string
		selectExpr string
	}{
		{
			label:      "catalog external identity",
			table:      "catalog_external_ids",
			columns:    []string{"provider", "provider_type", "external_id"},
			groupBy:    "provider, provider_type, external_id",
			selectExpr: "provider || '|' || provider_type || '|' || external_id",
		},
		{
			label:      "metadata field state",
			table:      "metadata_field_states",
			columns:    []string{"item_id", "field_key"},
			groupBy:    "item_id, field_key",
			selectExpr: "CAST(item_id AS TEXT) || '|' || field_key",
		},
		{
			label:      "asset item link",
			table:      "asset_items",
			columns:    []string{"asset_id", "item_id", "role", "segment_index"},
			groupBy:    "asset_id, item_id, role, segment_index",
			selectExpr: "CAST(asset_id AS TEXT) || '|' || CAST(item_id AS TEXT) || '|' || role || '|' || CAST(segment_index AS TEXT)",
		},
		{
			label:      "asset file link",
			table:      "asset_files",
			columns:    []string{"asset_id", "file_id", "role", "part_index"},
			groupBy:    "asset_id, file_id, role, part_index",
			selectExpr: "CAST(asset_id AS TEXT) || '|' || CAST(file_id AS TEXT) || '|' || role || '|' || CAST(part_index AS TEXT)",
		},
		{
			label:      "inventory file storage path",
			table:      "inventory_files",
			columns:    []string{"storage_provider", "storage_path"},
			where:      "deleted_at IS NULL",
			groupBy:    "storage_provider, storage_path",
			selectExpr: "storage_provider || '|' || storage_path",
		},
		{
			label:      "media stream index",
			table:      "media_streams",
			columns:    []string{"file_id", "stream_index"},
			groupBy:    "file_id, stream_index",
			selectExpr: "CAST(file_id AS TEXT) || '|' || CAST(stream_index AS TEXT)",
		},
		{
			label:      "user item data identity",
			table:      "user_item_data",
			columns:    []string{"user_id", "item_id", "asset_id"},
			groupBy:    "user_id, item_id, asset_id",
			selectExpr: "CAST(user_id AS TEXT) || '|' || CAST(item_id AS TEXT) || '|' || COALESCE(CAST(asset_id AS TEXT), 'null')",
		},
		{
			label:      "system setting category key",
			table:      "system_settings",
			columns:    []string{"category", "key"},
			groupBy:    "category, key",
			selectExpr: "category || '|' || key",
		},
	}

	for _, check := range checks {
		if !db.Migrator().HasTable(check.table) {
			continue
		}
		var duplicates []string
		query := db.Table(check.table).Select(check.selectExpr)
		if strings.TrimSpace(check.where) != "" {
			query = query.Where(check.where)
		}
		if err := query.Group(check.groupBy).Having("COUNT(*) > 1").Limit(3).Scan(&duplicates).Error; err != nil {
			return fmt.Errorf("check %s duplicates: %w", check.label, err)
		}
		if len(duplicates) > 0 {
			return fmt.Errorf("duplicate-prone %s rows block startup; sample keys: %s", check.label, strings.Join(duplicates, ", "))
		}
	}

	return nil
}

func ensureSQLiteDir(dsn string) error {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" || trimmed == ":memory:" || strings.HasPrefix(trimmed, "file:") {
		return nil
	}

	dir := filepath.Dir(trimmed)
	if dir == "." || dir == "" {
		return nil
	}

	return os.MkdirAll(dir, 0o755)
}
