package library

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestEffectiveLibraryConfigUsesDefaultPoliciesAndCompatibilityPath(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	source := database.MediaSource{Name: "Local", Provider: "local", StorageRef: "/media", RootPath: "/media"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	libraryRecord := database.Library{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: "/media/movies", Status: "active", ScannerEnabled: true}
	if err := db.Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)

	effective, err := svc.EffectiveLibraryConfig(context.Background(), libraryRecord.ID)
	if err != nil {
		t.Fatalf("resolve effective config: %v", err)
	}
	if len(effective.Paths) != 1 || effective.Paths[0].RootPath != "/media/movies" || effective.Paths[0].MediaSourceID != source.ID {
		t.Fatalf("unexpected effective paths: %#v", effective.Paths)
	}
	if !effective.ScanPolicy.ScannerEnabled || !effective.MetadataPolicy.LocalMetadataEnabled || !effective.PlaybackPolicy.ResumeEnabled || !effective.SubtitlePolicy.ExternalSidecarsEnabled {
		t.Fatalf("unexpected default policies: %#v %#v %#v %#v", effective.ScanPolicy, effective.MetadataPolicy, effective.PlaybackPolicy, effective.SubtitlePolicy)
	}
}

func TestEffectiveLibraryConfigUsesEnabledLibraryPaths(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	source := database.MediaSource{Name: "Local", Provider: "local", StorageRef: "/media", RootPath: "/media"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	libraryRecord := database.Library{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: "/media/movies", Status: "active", ScannerEnabled: true}
	if err := db.Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	paths := []database.LibraryPath{
		{LibraryID: libraryRecord.ID, MediaSourceID: source.ID, RootPath: "/media/a", Enabled: true},
		{LibraryID: libraryRecord.ID, MediaSourceID: source.ID, RootPath: "/media/b", Enabled: false},
		{LibraryID: libraryRecord.ID, MediaSourceID: source.ID, RootPath: "/media/c", Enabled: true},
	}
	for _, path := range paths {
		if err := db.Create(&path).Error; err != nil {
			t.Fatalf("create path: %v", err)
		}
	}
	if err := db.Model(&database.LibraryPath{}).Where("library_id = ? AND root_path = ?", libraryRecord.ID, "/media/b").Update("enabled", false).Error; err != nil {
		t.Fatalf("disable path: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)

	effective, err := svc.EffectiveLibraryConfig(context.Background(), libraryRecord.ID)
	if err != nil {
		t.Fatalf("resolve effective config: %v", err)
	}
	if len(effective.Paths) != 2 {
		t.Fatalf("expected two enabled paths, got %#v", effective.Paths)
	}
	if effective.Paths[0].RootPath != "/media/a" || effective.Paths[1].RootPath != "/media/c" {
		t.Fatalf("unexpected enabled paths: %#v", effective.Paths)
	}
}
