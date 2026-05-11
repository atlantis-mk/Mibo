package database

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

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

func TestRepairSQLiteContentShapePlanScopeIndexRebuildsStaleDefinition(t *testing.T) {
	db := openConstraintTestDB(t)

	if err := db.Exec(`
		CREATE TABLE content_shape_plans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			library_id INTEGER NOT NULL,
			storage_provider TEXT NOT NULL,
			root_path TEXT NOT NULL,
			directory_path TEXT NOT NULL,
			classifier_version TEXT NOT NULL,
			review_state TEXT,
			deleted_scope BOOLEAN
		)
	`).Error; err != nil {
		t.Fatalf("create content_shape_plans table: %v", err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX idx_content_shape_plan_scope ON content_shape_plans(storage_provider, root_path, directory_path, classifier_version)`).Error; err != nil {
		t.Fatalf("create stale content_shape_plan_scope index: %v", err)
	}

	if err := repairSQLiteConflictIndexes(db); err != nil {
		t.Fatalf("repair sqlite conflict indexes: %v", err)
	}
	exists, unique, columns, err := sqliteIndexDefinition(db, "content_shape_plans", "idx_content_shape_plan_scope")
	if err != nil {
		t.Fatalf("load repaired sqlite index definition: %v", err)
	}
	if !exists {
		t.Fatal("expected repaired idx_content_shape_plan_scope to exist")
	}
	if !unique {
		t.Fatal("expected repaired idx_content_shape_plan_scope to remain unique")
	}
	expected := []string{"library_id", "storage_provider", "root_path", "directory_path", "classifier_version"}
	if !sameStringSlice(columns, expected) {
		t.Fatalf("expected repaired idx_content_shape_plan_scope columns %v, got %v", expected, columns)
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
