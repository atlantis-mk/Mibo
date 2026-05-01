package database

import (
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
)

func TestBackfillLibraryPathsAndPoliciesCreatesDefaults(t *testing.T) {
	db, err := Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	source := MediaSource{Name: "Local", Provider: "local", StorageRef: "/media", RootPath: "/media"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	library := Library{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: "/media/movies", Status: "active", ScannerEnabled: true}
	if err := db.Create(&library).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}

	if err := BackfillLibraryPathsAndPolicies(db); err != nil {
		t.Fatalf("backfill library paths and policies: %v", err)
	}

	var path LibraryPath
	if err := db.Where("library_id = ?", library.ID).First(&path).Error; err != nil {
		t.Fatalf("load library path: %v", err)
	}
	if path.MediaSourceID != source.ID || path.RootPath != "/media/movies" || !path.Enabled {
		t.Fatalf("unexpected path backfill: %#v", path)
	}

	var scan LibraryScanPolicy
	if err := db.Where("library_id = ?", library.ID).First(&scan).Error; err != nil {
		t.Fatalf("load scan policy: %v", err)
	}
	if !scan.ScannerEnabled || !scan.RealtimeMonitorEnabled || scan.RefreshIntervalHours != 24 || !scan.ConfigurableExclusionRules {
		t.Fatalf("unexpected scan defaults: %#v", scan)
	}
	var metadata LibraryMetadataPolicy
	if err := db.Where("library_id = ?", library.ID).First(&metadata).Error; err != nil {
		t.Fatalf("load metadata policy: %v", err)
	}
	if !metadata.LocalMetadataEnabled || metadata.PreferredMetadataLanguage != "" || metadata.PreferredImageLanguage != "" || metadata.MetadataCountryCode != "" {
		t.Fatalf("unexpected metadata defaults: %#v", metadata)
	}
	var playback LibraryPlaybackPolicy
	if err := db.Where("library_id = ?", library.ID).First(&playback).Error; err != nil {
		t.Fatalf("load playback policy: %v", err)
	}
	if !playback.ResumeEnabled || playback.MinResumePct != 5 || playback.MaxResumePct != 90 {
		t.Fatalf("unexpected playback defaults: %#v", playback)
	}
	var subtitle LibrarySubtitlePolicy
	if err := db.Where("library_id = ?", library.ID).First(&subtitle).Error; err != nil {
		t.Fatalf("load subtitle policy: %v", err)
	}
	if !subtitle.ExternalSidecarsEnabled || !subtitle.TolerateUnavailableSubtitles {
		t.Fatalf("unexpected subtitle defaults: %#v", subtitle)
	}
}

func TestBackfillLibraryPathsAndPoliciesPreservesExistingPath(t *testing.T) {
	db, err := Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	source := MediaSource{Name: "Local", Provider: "local", StorageRef: "/media", RootPath: "/media"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	library := Library{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: "/media/movies", Status: "active", ScannerEnabled: true}
	if err := db.Create(&library).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	custom := LibraryPath{LibraryID: library.ID, MediaSourceID: source.ID, RootPath: "/media/custom", Enabled: true}
	if err := db.Create(&custom).Error; err != nil {
		t.Fatalf("create custom path: %v", err)
	}

	if err := BackfillLibraryPathsAndPolicies(db); err != nil {
		t.Fatalf("backfill library paths and policies: %v", err)
	}
	var count int64
	if err := db.Model(&LibraryPath{}).Where("library_id = ?", library.ID).Count(&count).Error; err != nil {
		t.Fatalf("count paths: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected existing path to be preserved without duplicate, got %d", count)
	}
}
