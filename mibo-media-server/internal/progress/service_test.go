package progress

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/search"
	"gorm.io/gorm"
)

func TestProgressKeepsFurthestUnfinishedPosition(t *testing.T) {
	ctx := context.Background()
	service, db, item, firstFile, _ := newProgressTestService(t)

	state, err := service.Update(ctx, 42, UpdateInput{
		MediaItemID:     item.ID,
		MediaFileID:     &firstFile.ID,
		PositionSeconds: 480,
		DurationSeconds: intPtr(1200),
	})
	if err != nil {
		t.Fatalf("initial update: %v", err)
	}
	if state.PositionSeconds != 480 {
		t.Fatalf("initial position = %d, want 480", state.PositionSeconds)
	}

	state, err = service.Update(ctx, 42, UpdateInput{
		MediaItemID:     item.ID,
		PositionSeconds: 300,
		DurationSeconds: intPtr(1100),
	})
	if err != nil {
		t.Fatalf("stale update: %v", err)
	}

	if state.PositionSeconds != 480 {
		t.Fatalf("position_seconds = %d, want 480", state.PositionSeconds)
	}
	if state.MediaFileID == nil || *state.MediaFileID != firstFile.ID {
		t.Fatalf("media_file_id = %v, want %d", state.MediaFileID, firstFile.ID)
	}
	if state.Watched {
		t.Fatal("expected unfinished state to stay unwatched")
	}

	persisted := loadProgressRecord(t, ctx, db, 42, item.ID)
	if persisted.PositionSeconds != 480 {
		t.Fatalf("persisted position_seconds = %d, want 480", persisted.PositionSeconds)
	}
}

func TestProgressCompletionDominatesCanonicalStateAndDiscoveryRails(t *testing.T) {
	ctx := context.Background()
	service, _, item, firstFile, _ := newProgressTestService(t)

	_, err := service.Update(ctx, 7, UpdateInput{
		MediaItemID:     item.ID,
		MediaFileID:     &firstFile.ID,
		PositionSeconds: 540,
		DurationSeconds: intPtr(1200),
	})
	if err != nil {
		t.Fatalf("seed unfinished progress: %v", err)
	}

	state, err := service.Update(ctx, 7, UpdateInput{
		MediaItemID:     item.ID,
		PositionSeconds: 1180,
		DurationSeconds: intPtr(1200),
		Completed:       true,
	})
	if err != nil {
		t.Fatalf("complete progress: %v", err)
	}

	if !state.Watched {
		t.Fatal("expected watched=true after explicit completion")
	}
	if state.CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
	if state.LastPlayedAt == nil {
		t.Fatal("expected last_played_at to be set")
	}

	continueWatching, err := service.ContinueWatching(ctx, 7, 10)
	if err != nil {
		t.Fatalf("continue watching: %v", err)
	}
	if len(continueWatching) != 0 {
		t.Fatalf("continue watching = %#v, want empty after completion", continueWatching)
	}

	recentlyPlayed, err := service.RecentlyPlayed(ctx, 7, 10)
	if err != nil {
		t.Fatalf("recently played: %v", err)
	}
	if len(recentlyPlayed) != 1 {
		t.Fatalf("recently played len = %d, want 1", len(recentlyPlayed))
	}
	if !recentlyPlayed[0].Watched {
		t.Fatal("expected recently played entry to remain watched")
	}
	if recentlyPlayed[0].CompletedAt == nil {
		t.Fatal("expected recently played entry to retain completed_at")
	}
}

func TestProgressWatchedStateReindexesSearchDocument(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	source := database.MediaSource{Name: "Local", Provider: "local", StorageRef: "local", RootPath: t.TempDir()}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	libraryRecord := database.Library{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: source.RootPath, Status: "active"}
	if err := db.Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	runtimeSeconds := 1200
	item := database.MediaItem{
		LibraryID:      libraryRecord.ID,
		Type:           "movie",
		Title:          "Progress Reindex Movie",
		RuntimeSeconds: &runtimeSeconds,
		SourcePath:     filepath.Join(source.RootPath, "progress-reindex.mkv"),
		MatchStatus:    "matched",
		Status:         "ready",
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	searchSvc := search.NewService(db)
	if err := searchSvc.ReindexMediaItem(ctx, item.ID); err != nil {
		t.Fatalf("seed search document: %v", err)
	}
	service := NewService(db, searchSvc)

	var seeded database.SearchDocument
	if err := db.WithContext(ctx).First(&seeded, "media_item_id = ?", item.ID).Error; err != nil {
		t.Fatalf("load seeded document: %v", err)
	}

	if _, err := service.Update(ctx, 7, UpdateInput{MediaItemID: item.ID, PositionSeconds: 180, DurationSeconds: intPtr(1200)}); err != nil {
		t.Fatalf("set in-progress state: %v", err)
	}
	var inProgress database.SearchDocument
	if err := db.WithContext(ctx).First(&inProgress, "media_item_id = ?", item.ID).Error; err != nil {
		t.Fatalf("load in-progress document: %v", err)
	}
	if !inProgress.UpdatedAt.After(seeded.UpdatedAt) {
		t.Fatalf("expected in-progress update to refresh search document timestamp: before=%s after=%s", seeded.UpdatedAt, inProgress.UpdatedAt)
	}

	if _, err := service.Update(ctx, 7, UpdateInput{MediaItemID: item.ID, PositionSeconds: 1180, DurationSeconds: intPtr(1200), Completed: true}); err != nil {
		t.Fatalf("set watched state: %v", err)
	}
	var watched database.SearchDocument
	if err := db.WithContext(ctx).First(&watched, "media_item_id = ?", item.ID).Error; err != nil {
		t.Fatalf("load watched document: %v", err)
	}
	if !watched.UpdatedAt.After(inProgress.UpdatedAt) {
		t.Fatalf("expected watched update to refresh search document timestamp: in-progress=%s watched=%s", inProgress.UpdatedAt, watched.UpdatedAt)
	}

	state, err := service.GetState(ctx, 7, item.ID)
	if err != nil {
		t.Fatalf("load progress state: %v", err)
	}
	if !state.Watched {
		t.Fatal("expected watched state after completion")
	}
}

func newProgressTestService(t *testing.T) (*Service, *gorm.DB, database.MediaItem, database.MediaFile, database.MediaFile) {
	t.Helper()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	source := database.MediaSource{Name: "Local", Provider: "local", StorageRef: "local", RootPath: t.TempDir()}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}

	library := database.Library{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: source.RootPath, Status: "active"}
	if err := db.Create(&library).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}

	runtimeSeconds := 1200
	item := database.MediaItem{
		LibraryID:      library.ID,
		Type:           "movie",
		Title:          "Progress Test Movie",
		RuntimeSeconds: &runtimeSeconds,
		SourcePath:     filepath.Join(source.RootPath, "progress-test.mkv"),
		MatchStatus:    "matched",
		Status:         "ready",
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	firstFile := database.MediaFile{LibraryID: library.ID, MediaItemID: &item.ID, StoragePath: filepath.Join(source.RootPath, "progress-test-a.mkv"), ProbeStatus: "ready"}
	secondFile := database.MediaFile{LibraryID: library.ID, MediaItemID: &item.ID, StoragePath: filepath.Join(source.RootPath, "progress-test-b.mkv"), ProbeStatus: "ready"}
	for _, file := range []*database.MediaFile{&firstFile, &secondFile} {
		if err := db.Create(file).Error; err != nil {
			t.Fatalf("create file: %v", err)
		}
	}

	return NewService(db), db, item, firstFile, secondFile
}

func loadProgressRecord(t *testing.T, ctx context.Context, db *gorm.DB, userID, mediaItemID uint) database.PlaybackProgress {
	t.Helper()

	var progress database.PlaybackProgress
	if err := db.WithContext(ctx).Where("user_id = ? AND media_item_id = ?", userID, mediaItemID).First(&progress).Error; err != nil {
		t.Fatalf("load progress: %v", err)
	}
	return progress
}

func intPtr(value int) *int {
	return &value
}

func timeCloseToNow(t *testing.T, value *time.Time) {
	t.Helper()
	if value == nil {
		t.Fatal("expected timestamp to be set")
	}
	if time.Since(*value) > 5*time.Second {
		t.Fatalf("timestamp %s too old", value.UTC())
	}
}
