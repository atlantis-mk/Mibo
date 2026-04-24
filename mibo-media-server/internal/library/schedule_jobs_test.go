package library

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/schedule"
	"gorm.io/gorm"
)

func TestScheduledScanRespectsScope(t *testing.T) {
	ctx := context.Background()
	_, svc, firstLibrary, secondLibrary := newScheduledLibraryService(t)

	globalResult, err := svc.RunScheduledScan(ctx, schedule.DueSchedule{Kind: schedule.KindScan, ScopeKind: schedule.ScopeGlobal})
	if err != nil {
		t.Fatalf("run global scheduled scan: %v", err)
	}
	if globalResult.LibrariesProcessed != 2 {
		t.Fatalf("expected 2 libraries processed, got %d", globalResult.LibrariesProcessed)
	}

	singleResult, err := svc.RunScheduledScan(ctx, schedule.DueSchedule{Kind: schedule.KindScan, ScopeKind: schedule.ScopeLibrary, LibraryID: &firstLibrary.ID})
	if err != nil {
		t.Fatalf("run single-library scheduled scan: %v", err)
	}
	if singleResult.LibrariesProcessed != 1 {
		t.Fatalf("expected 1 library processed, got %d", singleResult.LibrariesProcessed)
	}
	if secondLibrary.ID == firstLibrary.ID {
		t.Fatal("expected distinct libraries")
	}
}

func TestScheduledCleanupAndInvalidLinkChecks(t *testing.T) {
	ctx := context.Background()
	db, svc, firstLibrary, _ := newScheduledLibraryService(t)

	cleanupResult, err := svc.RunScheduledCleanup(ctx, schedule.DueSchedule{Kind: schedule.KindLibraryCleanup, ScopeKind: schedule.ScopeLibrary, LibraryID: &firstLibrary.ID})
	if err != nil {
		t.Fatalf("run scheduled cleanup: %v", err)
	}
	if cleanupResult.LibrariesProcessed != 1 {
		t.Fatalf("expected cleanup to process 1 library, got %d", cleanupResult.LibrariesProcessed)
	}

	missingPath := filepath.Join(firstLibrary.RootPath, "missing-file.mkv")
	file := database.MediaFile{LibraryID: firstLibrary.ID, StoragePath: missingPath}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create media file: %v", err)
	}

	invalidResult, err := svc.RunScheduledInvalidLinkCheck(ctx, schedule.DueSchedule{Kind: schedule.KindInvalidLinkCheck, ScopeKind: schedule.ScopeLibrary, LibraryID: &firstLibrary.ID})
	if err != nil {
		t.Fatalf("run invalid link check: %v", err)
	}
	if invalidResult.Failures == 0 {
		t.Fatalf("expected invalid link failures, got %#v", invalidResult)
	}

	var libraryRecord database.Library
	if err := db.WithContext(ctx).First(&libraryRecord, firstLibrary.ID).Error; err != nil {
		t.Fatalf("reload library: %v", err)
	}
	if libraryRecord.Status != "active" {
		t.Fatalf("expected invalid-link check not to mutate library status, got %q", libraryRecord.Status)
	}
}

func newScheduledLibraryService(t *testing.T) (*gorm.DB, *Service, database.Library, database.Library) {
	t.Helper()
	mediaRoot := t.TempDir()
	firstRoot := filepath.Join(mediaRoot, "MoviesA")
	secondRoot := filepath.Join(mediaRoot, "MoviesB")
	if err := os.MkdirAll(firstRoot, 0o755); err != nil {
		t.Fatalf("mkdir first root: %v", err)
	}
	if err := os.MkdirAll(secondRoot, 0o755); err != nil {
		t.Fatalf("mkdir second root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(firstRoot, "MovieA.2024.mkv"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write first media file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secondRoot, "MovieB.2024.mkv"), []byte("b"), 0o644); err != nil {
		t.Fatalf("write second media file: %v", err)
	}

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	cfg := config.Config{Local: config.LocalStorageConfig{RootPath: mediaRoot}}
	svc := NewService(cfg, db, providers.NewRegistry(cfg), jobs.NewService(db))

	ctx := context.Background()
	firstSource := database.MediaSource{Name: "LocalA", Provider: "local", RootPath: firstRoot, StorageRef: firstRoot}
	secondSource := database.MediaSource{Name: "LocalB", Provider: "local", RootPath: secondRoot, StorageRef: secondRoot}
	if err := db.WithContext(ctx).Create(&firstSource).Error; err != nil {
		t.Fatalf("create first source: %v", err)
	}
	if err := db.WithContext(ctx).Create(&secondSource).Error; err != nil {
		t.Fatalf("create second source: %v", err)
	}
	firstLibrary := database.Library{Name: "First", Type: "movies", MediaSourceID: firstSource.ID, RootPath: firstRoot, Status: "active", ScannerEnabled: true}
	secondLibrary := database.Library{Name: "Second", Type: "movies", MediaSourceID: secondSource.ID, RootPath: secondRoot, Status: "active", ScannerEnabled: true}
	if err := db.WithContext(ctx).Create(&firstLibrary).Error; err != nil {
		t.Fatalf("create first library: %v", err)
	}
	if err := db.WithContext(ctx).Create(&secondLibrary).Error; err != nil {
		t.Fatalf("create second library: %v", err)
	}

	return db, svc, firstLibrary, secondLibrary
}
