package library

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func TestInventoryFileSignalsPersistAndReuseUnchangedFile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openSignalTestDB(t)
	modified := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Show/Show.S01E02.1080p.WEB-DL.mkv", StableIdentityKey: "local:show-e02", SizeBytes: 1024, ModifiedAt: &modified, ContentClass: SourceContentClassVideo, Status: "available"}
	scope := inventoryFileSignalScope{LibraryID: 1, StorageProvider: "local", ClassifierVersion: ContentShapeClassifierVersion}
	model := extractFilenameSignalModel(file.StoragePath)
	if err := saveInventoryFileSignals(ctx, db, scope, []inventoryFileSignalInput{{File: file, Model: model}}); err != nil {
		t.Fatalf("save signal: %v", err)
	}

	models, rows, err := loadReusableInventoryFileSignals(ctx, db, scope, []database.InventoryFile{file})
	if err != nil {
		t.Fatalf("load reusable signal: %v", err)
	}
	loaded, ok := models[file.StoragePath]
	if !ok {
		t.Fatalf("expected reusable model for %s", file.StoragePath)
	}
	if loaded.Identity.EpisodeNumber == nil || *loaded.Identity.EpisodeNumber != 2 || loaded.ReleaseHints.Quality == "" {
		t.Fatalf("unexpected loaded model: %#v", loaded)
	}
	if rows[file.StoragePath].FileFingerprint != inventoryFileSignalFingerprint(file, ContentShapeClassifierVersion) {
		t.Fatalf("expected stored fingerprint to match current file")
	}
}

func TestInventoryFileSignalsInvalidateOnFingerprintOrVersionChange(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openSignalTestDB(t)
	modified := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie.2024.mkv", StableIdentityKey: "local:movie", SizeBytes: 1024, ModifiedAt: &modified, ContentClass: SourceContentClassVideo, Status: "available"}
	scope := inventoryFileSignalScope{LibraryID: 1, StorageProvider: "local", ClassifierVersion: ContentShapeClassifierVersion}
	if err := saveInventoryFileSignals(ctx, db, scope, []inventoryFileSignalInput{{File: file, Model: extractFilenameSignalModel(file.StoragePath)}}); err != nil {
		t.Fatalf("save signal: %v", err)
	}

	changedFile := file
	changedFile.SizeBytes = 2048
	models, _, err := loadReusableInventoryFileSignals(ctx, db, scope, []database.InventoryFile{changedFile})
	if err != nil {
		t.Fatalf("load changed signal: %v", err)
	}
	if _, ok := models[file.StoragePath]; ok {
		t.Fatalf("expected changed file fingerprint to invalidate reusable signal")
	}

	changedScope := scope
	changedScope.ClassifierVersion = ContentShapeClassifierVersion + "-next"
	models, _, err = loadReusableInventoryFileSignals(ctx, db, changedScope, []database.InventoryFile{file})
	if err != nil {
		t.Fatalf("load changed version signal: %v", err)
	}
	if _, ok := models[file.StoragePath]; ok {
		t.Fatalf("expected changed classifier version to invalidate reusable signal")
	}
}

func openSignalTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db
}
