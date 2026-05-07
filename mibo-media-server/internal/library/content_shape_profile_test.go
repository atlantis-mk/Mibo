package library

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
)

func TestContentShapeProfilePersistsAndReusesUnchangedDirectory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	snapshot := largeEpisodeShapeSnapshot("/library/Show", 100)
	scope := testContentShapeScope("content-shape-v1", snapshot.Path)
	profile, reused, err := loadOrBuildContentShapeProfile(ctx, db, scope, snapshot, database.LibraryScanPolicy{IgnoreHiddenFiles: true}, nil, newFilenameTokenProfileCache())
	if err != nil || reused {
		t.Fatalf("expected first profile build, reused=%t err=%v", reused, err)
	}
	if profile.VideoCount != 100 || profile.NonExtraVideoCount != 100 || profile.SequenceCoverage == nil || *profile.SequenceCoverage < 1 {
		t.Fatalf("expected persisted large episode evidence, got %#v", profile)
	}
	second, reused, err := loadOrBuildContentShapeProfile(ctx, db, scope, snapshot, database.LibraryScanPolicy{IgnoreHiddenFiles: true}, nil, newFilenameTokenProfileCache())
	if err != nil || !reused {
		t.Fatalf("expected unchanged profile reuse, reused=%t err=%v", reused, err)
	}
	if second.ID != profile.ID {
		t.Fatalf("expected same profile ID, first=%d second=%d", profile.ID, second.ID)
	}
}

func TestContentShapeProfileClassifierAndExclusionInputsInvalidateFingerprint(t *testing.T) {
	t.Parallel()

	snapshot := largeEpisodeShapeSnapshot("/library/Show", 3)
	policy := database.LibraryScanPolicy{IgnoreHiddenFiles: true, ConfigurableExclusionRules: true}
	base := contentShapeDirectoryFingerprint(contentShapeFingerprintInput{LibraryID: 1, StorageProvider: "local", RootPath: "/library", DirectoryPath: snapshot.Path, ClassifierVersion: "content-shape-v1", ScanPolicy: policy, Snapshot: snapshot, VisibleVideoCount: 3})
	changedVersion := contentShapeDirectoryFingerprint(contentShapeFingerprintInput{LibraryID: 1, StorageProvider: "local", RootPath: "/library", DirectoryPath: snapshot.Path, ClassifierVersion: "content-shape-v2", ScanPolicy: policy, Snapshot: snapshot, VisibleVideoCount: 3})
	changedExclusions := contentShapeDirectoryFingerprint(contentShapeFingerprintInput{LibraryID: 1, StorageProvider: "local", RootPath: "/library", DirectoryPath: snapshot.Path, ClassifierVersion: "content-shape-v1", ScanPolicy: policy, ExclusionRules: []database.ScanExclusionRule{{Key: "hide-sample", RuleType: "glob", Value: "*sample*", Reason: "sample", Enabled: true}}, Snapshot: snapshot, VisibleVideoCount: 3})
	if base == changedVersion {
		t.Fatalf("expected classifier version to change fingerprint")
	}
	if base == changedExclusions {
		t.Fatalf("expected exclusion inputs to change fingerprint")
	}
}

func TestContentShapeProfileCapturesMovieCollectionEvidence(t *testing.T) {
	t.Parallel()

	profile := buildContentShapeDirectoryProfile("auto", "/library", scanDirectorySnapshot{Path: "/library/Movies", Objects: []storage.Object{{Path: "/library/Movies/Alien.1979.mkv"}, {Path: "/library/Movies/Aliens.1986.mkv"}, {Path: "/library/Movies/Heat.1995.mkv"}}}, newFilenameTokenProfileCache())
	if profile.VideoCount != 3 || profile.TitleYearCount != 3 || profile.YearDensity != 1 || profile.TitleUniqueness != 1 {
		t.Fatalf("expected movie collection evidence, got %#v", profile)
	}
	if profile.SequenceCoverage != 0 || len(profile.NumericSequence) != 0 {
		t.Fatalf("expected weak episode sequence evidence, got %#v", profile)
	}
}

func largeEpisodeShapeSnapshot(dir string, count int) scanDirectorySnapshot {
	objects := make([]storage.Object, 0, count)
	for i := 1; i <= count; i++ {
		objects = append(objects, storage.Object{Path: dir + "/" + zeroPad3(i) + ".mkv"})
	}
	return scanDirectorySnapshot{Path: dir, Objects: objects}
}

func testContentShapeScope(version string, dir string) contentShapeScope {
	return contentShapeScope{LibraryID: 1, MediaSourceID: 1, StorageProvider: "local", RootPath: "/library", DirectoryPath: dir, ClassifierVersion: version}
}

func zeroPad3(value int) string {
	return fmt.Sprintf("%03d", value)
}
