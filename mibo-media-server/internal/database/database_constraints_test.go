package database

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestValidateCatalogKernelUniquenessRejectsDuplicateAssetItems(t *testing.T) {
	db := openConstraintTestDB(t)

	if err := db.Exec(`
		CREATE TABLE asset_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			asset_id INTEGER NOT NULL,
			item_id INTEGER NOT NULL,
			role TEXT NOT NULL,
			segment_index INTEGER NOT NULL
		)
	`).Error; err != nil {
		t.Fatalf("create asset_items table: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO asset_items (asset_id, item_id, role, segment_index) VALUES
		(1, 2, 'primary', 0),
		(1, 2, 'primary', 0)
	`).Error; err != nil {
		t.Fatalf("seed duplicate asset_items rows: %v", err)
	}

	err := validateCatalogKernelUniqueness(db)
	if err == nil {
		t.Fatal("expected duplicate asset item rows to be rejected")
	}
	if !strings.Contains(err.Error(), "asset item link") {
		t.Fatalf("expected asset item duplicate error, got %v", err)
	}
}

func TestValidateCatalogKernelUniquenessIgnoresSoftDeletedInventoryDuplicates(t *testing.T) {
	db := openConstraintTestDB(t)

	if err := db.Exec(`
		CREATE TABLE inventory_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			storage_provider TEXT NOT NULL,
			storage_path TEXT NOT NULL,
			deleted_at DATETIME NULL
		)
	`).Error; err != nil {
		t.Fatalf("create inventory_files table: %v", err)
	}
	deletedAt := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC).Format("2006-01-02 15:04:05")
	if err := db.Exec(`
		INSERT INTO inventory_files (storage_provider, storage_path, deleted_at) VALUES
		('local', '/media/show.mkv', NULL),
		('local', '/media/show.mkv', ?)
	`, deletedAt).Error; err != nil {
		t.Fatalf("seed inventory_files rows: %v", err)
	}

	if err := validateCatalogKernelUniqueness(db); err != nil {
		t.Fatalf("expected soft-deleted duplicate inventory path to be ignored, got %v", err)
	}
}

func openConstraintTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "constraints.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open constraint test database: %v", err)
	}
	return db
}
