package library_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/probe"
	"github.com/atlan/mibo-media-server/internal/providers"
	"gorm.io/gorm"
)

func TestProbeFileReconcilesSingleFallbackCandidate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tempRoot := t.TempDir()
	newPath := filepath.Join(tempRoot, "Renamed.Movie.2024.mkv")
	if err := os.WriteFile(newPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write media file: %v", err)
	}

	db, libraryRecord, probeSvc := newReconcileHarness(t, tempRoot, "7200.8")
	seed := seedFallbackReconcileData(t, ctx, db, libraryRecord.ID, filepath.Join(tempRoot, "MovieA.2024.mkv"), newPath)

	if err := probeSvc.ProbeFile(ctx, seed.newFile.ID); err != nil {
		t.Fatalf("probe file: %v", err)
	}

	var newFile database.MediaFile
	if err := db.WithContext(ctx).First(&newFile, seed.newFile.ID).Error; err != nil {
		t.Fatalf("reload new file: %v", err)
	}
	if newFile.MediaItemID == nil || *newFile.MediaItemID != seed.oldItem.ID {
		t.Fatalf("expected reconciled file to rebind old media item %d, got %#v", seed.oldItem.ID, newFile.MediaItemID)
	}

	var oldFile database.MediaFile
	if err := db.WithContext(ctx).First(&oldFile, seed.oldFile.ID).Error; err != nil {
		t.Fatalf("reload old file: %v", err)
	}
	if oldFile.ReplacedByID == nil || *oldFile.ReplacedByID != newFile.ID {
		t.Fatalf("expected old file to point at replacement %d, got %#v", newFile.ID, oldFile.ReplacedByID)
	}

	var oldItem database.MediaItem
	if err := db.WithContext(ctx).First(&oldItem, seed.oldItem.ID).Error; err != nil {
		t.Fatalf("reload old item: %v", err)
	}
	if oldItem.DeletedAt != nil {
		t.Fatalf("expected old media item to be restored, got deleted_at=%v", oldItem.DeletedAt)
	}
	if oldItem.SourcePath != newPath {
		t.Fatalf("expected old media item source path to move to %q, got %q", newPath, oldItem.SourcePath)
	}

	var progress database.PlaybackProgress
	if err := db.WithContext(ctx).First(&progress, seed.progress.ID).Error; err != nil {
		t.Fatalf("reload progress: %v", err)
	}
	if progress.MediaFileID == nil || *progress.MediaFileID != newFile.ID {
		t.Fatalf("expected playback progress to follow replacement file %d, got %#v", newFile.ID, progress.MediaFileID)
	}
}

func TestProbeFileDoesNotReconcileWithoutDurationData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tempRoot := t.TempDir()
	newPath := filepath.Join(tempRoot, "Renamed.Movie.2024.mkv")
	if err := os.WriteFile(newPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write media file: %v", err)
	}

	db, libraryRecord, probeSvc := newReconcileHarness(t, tempRoot, "")
	seed := seedFallbackReconcileData(t, ctx, db, libraryRecord.ID, filepath.Join(tempRoot, "MovieA.2024.mkv"), newPath)

	if err := probeSvc.ProbeFile(ctx, seed.newFile.ID); err != nil {
		t.Fatalf("probe file: %v", err)
	}

	var newFile database.MediaFile
	if err := db.WithContext(ctx).First(&newFile, seed.newFile.ID).Error; err != nil {
		t.Fatalf("reload new file: %v", err)
	}
	if newFile.MediaItemID == nil || *newFile.MediaItemID != seed.newItem.ID {
		t.Fatalf("expected unreconciled file to stay on provisional media item %d, got %#v", seed.newItem.ID, newFile.MediaItemID)
	}

	var oldFile database.MediaFile
	if err := db.WithContext(ctx).First(&oldFile, seed.oldFile.ID).Error; err != nil {
		t.Fatalf("reload old file: %v", err)
	}
	if oldFile.ReplacedByID != nil {
		t.Fatalf("expected no replacement link without duration, got %#v", oldFile.ReplacedByID)
	}
}

func TestProbeFileMarksAmbiguousFallbackForReview(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tempRoot := t.TempDir()
	newPath := filepath.Join(tempRoot, "Renamed.Movie.2024.mkv")
	if err := os.WriteFile(newPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write media file: %v", err)
	}

	db, libraryRecord, probeSvc := newReconcileHarness(t, tempRoot, "7200.2")
	seed := seedFallbackReconcileData(t, ctx, db, libraryRecord.ID, filepath.Join(tempRoot, "MovieA.2024.mkv"), newPath)
	otherDeletedAt := time.Now().UTC().Add(-time.Minute)
	otherItem := database.MediaItem{LibraryID: libraryRecord.ID, Type: "movie", Title: "MovieB", SourcePath: filepath.Join(tempRoot, "MovieB.2024.mkv"), MatchStatus: "matched", Status: "missing", DeletedAt: &otherDeletedAt}
	if err := db.WithContext(ctx).Create(&otherItem).Error; err != nil {
		t.Fatalf("create other item: %v", err)
	}
	otherFile := database.MediaFile{LibraryID: libraryRecord.ID, MediaItemID: &otherItem.ID, StoragePath: otherItem.SourcePath, SizeBytes: 4096, Fingerprint: "other-fingerprint", ProbeStatus: probe.StatusReady, DurationSeconds: float64Ptr(7200.1), IdentitySource: "provider_evidence", IdentityStatus: "provisional", ReviewStatus: "pending", DeletedAt: &otherDeletedAt}
	if err := db.WithContext(ctx).Create(&otherFile).Error; err != nil {
		t.Fatalf("create other file: %v", err)
	}

	if err := probeSvc.ProbeFile(ctx, seed.newFile.ID); err != nil {
		t.Fatalf("probe file: %v", err)
	}

	var newFile database.MediaFile
	if err := db.WithContext(ctx).First(&newFile, seed.newFile.ID).Error; err != nil {
		t.Fatalf("reload new file: %v", err)
	}
	if newFile.ReviewStatus != "review_needed" {
		t.Fatalf("expected ambiguous fallback to require review, got %q", newFile.ReviewStatus)
	}
	if newFile.MediaItemID == nil || *newFile.MediaItemID != seed.newItem.ID {
		t.Fatalf("expected ambiguous fallback to stay on provisional item %d, got %#v", seed.newItem.ID, newFile.MediaItemID)
	}

	var progress database.PlaybackProgress
	if err := db.WithContext(ctx).First(&progress, seed.progress.ID).Error; err != nil {
		t.Fatalf("reload progress: %v", err)
	}
	if progress.MediaFileID == nil || *progress.MediaFileID != seed.oldFile.ID {
		t.Fatalf("expected prior progress to stay on original file %d, got %#v", seed.oldFile.ID, progress.MediaFileID)
	}
}

type reconcileSeed struct {
	oldItem  database.MediaItem
	newItem  database.MediaItem
	oldFile  database.MediaFile
	newFile  database.MediaFile
	progress database.PlaybackProgress
}

func newReconcileHarness(t *testing.T, rootPath string, ffprobeDuration string) (*gorm.DB, database.Library, *probe.Service) {
	t.Helper()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	ffprobePath := writeFFprobeScript(t, ffprobeDuration)
	cfg := config.Config{
		Local:   config.LocalStorageConfig{RootPath: rootPath},
		FFprobe: config.FFprobeConfig{Enabled: true, Path: ffprobePath, Timeout: 2 * time.Second},
	}
	registry := providers.NewRegistry(cfg)
	jobsSvc := jobs.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	probeSvc := probe.NewService(db, registry, cfg.FFprobe)

	ctx := context.Background()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Local", RootPath: rootPath})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	libraryRecord, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: rootPath})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}

	return db, libraryRecord, probeSvc
}

func seedFallbackReconcileData(t *testing.T, ctx context.Context, db *gorm.DB, libraryID uint, oldPath string, newPath string) reconcileSeed {
	t.Helper()

	now := time.Now().UTC()
	oldItem := database.MediaItem{LibraryID: libraryID, Type: "movie", Title: "MovieA", SourcePath: oldPath, MatchStatus: "matched", Status: "missing", DeletedAt: &now}
	if err := db.WithContext(ctx).Create(&oldItem).Error; err != nil {
		t.Fatalf("create old item: %v", err)
	}
	newItem := database.MediaItem{LibraryID: libraryID, Type: "movie", Title: "MovieA", SourcePath: newPath, MatchStatus: "pending", Status: "ready"}
	if err := db.WithContext(ctx).Create(&newItem).Error; err != nil {
		t.Fatalf("create new item: %v", err)
	}

	oldFile := database.MediaFile{
		LibraryID:       libraryID,
		MediaItemID:     &oldItem.ID,
		StoragePath:     oldPath,
		SizeBytes:       4096,
		Fingerprint:     "old-fingerprint",
		ProbeStatus:     probe.StatusReady,
		DurationSeconds: float64Ptr(7200),
		IdentitySource:  "provider_evidence",
		IdentityStatus:  "provisional",
		ReviewStatus:    "pending",
		DeletedAt:       &now,
	}
	if err := db.WithContext(ctx).Create(&oldFile).Error; err != nil {
		t.Fatalf("create old file: %v", err)
	}

	newFile := database.MediaFile{
		LibraryID:      libraryID,
		MediaItemID:    &newItem.ID,
		StoragePath:    newPath,
		SizeBytes:      4096,
		Fingerprint:    "new-fingerprint",
		ProbeStatus:    probe.StatusPending,
		IdentitySource: "provider_evidence",
		IdentityStatus: "provisional",
		ReviewStatus:   "pending",
	}
	if err := db.WithContext(ctx).Create(&newFile).Error; err != nil {
		t.Fatalf("create new file: %v", err)
	}

	progress := database.PlaybackProgress{UserID: 1, MediaItemID: oldItem.ID, MediaFileID: &oldFile.ID, PositionSeconds: 600}
	if err := db.WithContext(ctx).Create(&progress).Error; err != nil {
		t.Fatalf("create progress: %v", err)
	}

	return reconcileSeed{oldItem: oldItem, newItem: newItem, oldFile: oldFile, newFile: newFile, progress: progress}
}

func writeFFprobeScript(t *testing.T, duration string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ffprobe")
	content := "#!/bin/sh\ncat <<'EOF'\n{\"streams\":[{\"codec_type\":\"video\",\"codec_name\":\"h264\",\"width\":1920,\"height\":1080}],\"format\":{\"duration\":\"" + duration + "\",\"bit_rate\":\"5000000\"}}\nEOF\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write ffprobe script: %v", err)
	}
	return path
}

func float64Ptr(value float64) *float64 {
	return &value
}
