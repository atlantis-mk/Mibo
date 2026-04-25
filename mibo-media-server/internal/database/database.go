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
		&CatalogMigrationRun{},
		&CatalogMigrationEntry{},
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
	); err != nil {
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
