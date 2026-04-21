package playback

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/probe"
	"github.com/atlan/mibo-media-server/internal/providers"
	"gorm.io/gorm"
)

func TestPlaybackDecisionDirectForCompatibleProfile(t *testing.T) {
	testSvc, itemID := newPlaybackDecisionFixture(t).addMediaItem("Web Direct", []database.MediaFile{{
		StoragePath: filepath.Join(t.TempDir(), "web-direct.mp4"),
		Container:   "mp4",
		ProbeStatus: probe.StatusReady,
		VideoCodec:  "h264",
	}})

	source, err := testSvc.service.GetPlaybackSource(context.Background(), PlaybackRequest{
		MediaItemID:      itemID,
		ClientProfile:    ClientProfileWeb,
		AllowHLSFallback: true,
	})
	if err != nil {
		t.Fatalf("get playback source: %v", err)
	}
	if source.Decision.Kind != "direct" {
		t.Fatalf("decision.kind = %q, want direct", source.Decision.Kind)
	}
	if !source.Playable {
		t.Fatal("expected playable direct result")
	}
	if len(source.Decision.Reasons) == 0 {
		t.Fatal("expected direct decision reasons")
	}
}

func TestPlaybackDecisionFallsBackWhenDirectRejectedAndHLSAllowed(t *testing.T) {
	testSvc, itemID := newPlaybackDecisionFixture(t).addMediaItem("Fallback", []database.MediaFile{{
		StoragePath: filepath.Join(t.TempDir(), "fallback.mkv"),
		Container:   "mkv",
		ProbeStatus: probe.StatusReady,
		VideoCodec:  "hevc",
	}})

	source, err := testSvc.service.GetPlaybackSource(context.Background(), PlaybackRequest{
		MediaItemID:      itemID,
		ClientProfile:    ClientProfileWeb,
		AllowHLSFallback: true,
	})
	if err != nil {
		t.Fatalf("get playback source: %v", err)
	}
	if source.Decision.Kind != "fallback" {
		t.Fatalf("decision.kind = %q, want fallback", source.Decision.Kind)
	}
	if source.Decision.FallbackKind != "hls" {
		t.Fatalf("fallback_kind = %q, want hls", source.Decision.FallbackKind)
	}
	if source.Direct {
		t.Fatal("expected fallback playback to be non-direct")
	}
	if source.Container != "m3u8" {
		t.Fatalf("container = %q, want m3u8", source.Container)
	}
}

func TestPlaybackDecisionReturnsUnplayableWhenNoFallbackAllowed(t *testing.T) {
	testSvc, itemID := newPlaybackDecisionFixture(t).addMediaItem("Unplayable", []database.MediaFile{{
		StoragePath: filepath.Join(t.TempDir(), "unplayable.mkv"),
		Container:   "mkv",
		ProbeStatus: probe.StatusReady,
		VideoCodec:  "hevc",
	}})

	source, err := testSvc.service.GetPlaybackSource(context.Background(), PlaybackRequest{
		MediaItemID:      itemID,
		ClientProfile:    ClientProfileWeb,
		AllowHLSFallback: false,
	})
	if err != nil {
		t.Fatalf("get playback source: %v", err)
	}
	if source.Decision.Kind != "unplayable" {
		t.Fatalf("decision.kind = %q, want unplayable", source.Decision.Kind)
	}
	if source.Playable {
		t.Fatal("expected unplayable result")
	}
	if source.URL != "" {
		t.Fatalf("url = %q, want empty", source.URL)
	}
	if len(source.Decision.Reasons) == 0 {
		t.Fatal("expected unplayable reasons")
	}
}

func TestPlaybackDecisionKeepsCommonProbeMissingMP4Direct(t *testing.T) {
	testSvc, itemID := newPlaybackDecisionFixture(t).addMediaItem("Probe Missing", []database.MediaFile{{
		StoragePath: filepath.Join(t.TempDir(), "probe-missing.mp4"),
		Container:   "mp4",
		ProbeStatus: probe.StatusPending,
	}})

	source, err := testSvc.service.GetPlaybackSource(context.Background(), PlaybackRequest{
		MediaItemID:      itemID,
		ClientProfile:    ClientProfileWeb,
		AllowHLSFallback: true,
	})
	if err != nil {
		t.Fatalf("get playback source: %v", err)
	}
	if source.Decision.Kind != "direct" {
		t.Fatalf("decision.kind = %q, want direct", source.Decision.Kind)
	}
	if !hasDecisionReasonCode(source.Decision.Reasons, "probe_missing_assumed_compatible") {
		t.Fatalf("expected uncertainty reason, got %#v", source.Decision.Reasons)
	}
}

func TestPlaybackDecisionPrefersCompatibleCandidateBeforeQuality(t *testing.T) {
	testSvc, itemID := newPlaybackDecisionFixture(t).addMediaItem("Ranking", []database.MediaFile{
		{
			StoragePath: filepath.Join(t.TempDir(), "ranking-best-quality.mkv"),
			Container:   "mkv",
			ProbeStatus: probe.StatusReady,
			VideoCodec:  "hevc",
			Width:       intPtr(3840),
			Height:      intPtr(2160),
			SizeBytes:   8_000,
		},
		{
			StoragePath: filepath.Join(t.TempDir(), "ranking-compatible.mp4"),
			Container:   "mp4",
			ProbeStatus: probe.StatusReady,
			VideoCodec:  "h264",
			Width:       intPtr(1280),
			Height:      intPtr(720),
			SizeBytes:   2_000,
		},
	})

	source, err := testSvc.service.GetPlaybackSource(context.Background(), PlaybackRequest{
		MediaItemID:      itemID,
		ClientProfile:    ClientProfileWeb,
		AllowHLSFallback: true,
	})
	if err != nil {
		t.Fatalf("get playback source: %v", err)
	}
	if source.Container != "mp4" {
		t.Fatalf("selected container = %q, want mp4", source.Container)
	}
	if source.Decision.Kind != "direct" {
		t.Fatalf("decision.kind = %q, want direct", source.Decision.Kind)
	}
}

type playbackDecisionFixture struct {
	t        *testing.T
	db       *gorm.DB
	service  *Service
	library  database.Library
	rootPath string
	nextPath int
}

func newPlaybackDecisionFixture(t *testing.T) *playbackDecisionFixture {
	t.Helper()

	rootPath := t.TempDir()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		Storage:  config.StorageConfig{Provider: "local"},
		Local:    config.LocalStorageConfig{RootPath: rootPath},
	}
	registry := providers.NewRegistry(cfg)

	source := database.MediaSource{
		Name:       "Local Media",
		Provider:   "local",
		StorageRef: rootPath,
		RootPath:   rootPath,
	}
	if err := db.WithContext(context.Background()).Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}

	libraryRecord := database.Library{
		Name:          "Movies",
		Type:          "movies",
		MediaSourceID: source.ID,
		RootPath:      rootPath,
		Status:        "active",
		ScannerEnabled: true,
	}
	if err := db.WithContext(context.Background()).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}

	return &playbackDecisionFixture{t: t, db: db, service: NewService(db, registry), library: libraryRecord, rootPath: rootPath}
}

func (f *playbackDecisionFixture) addMediaItem(title string, files []database.MediaFile) (*playbackDecisionFixture, uint) {
	f.t.Helper()

	item := database.MediaItem{
		LibraryID:   f.library.ID,
		Type:        "movie",
		Title:       title,
		SourcePath:  filepath.Join(f.rootPath, title),
		MatchStatus: "matched",
		Status:      "ready",
	}
	if err := f.db.WithContext(context.Background()).Create(&item).Error; err != nil {
		f.t.Fatalf("create item: %v", err)
	}

	for i := range files {
		files[i].LibraryID = f.library.ID
		files[i].MediaItemID = &item.ID
		if files[i].StoragePath == "" {
			f.nextPath++
			files[i].StoragePath = filepath.Join(f.rootPath, title+"-"+string(rune('a'+f.nextPath))+".mp4")
		} else {
			files[i].StoragePath = filepath.Join(f.rootPath, filepath.Base(files[i].StoragePath))
		}
		if err := os.WriteFile(files[i].StoragePath, []byte("video"), 0o644); err != nil {
			f.t.Fatalf("write media file: %v", err)
		}
		if err := f.db.WithContext(context.Background()).Create(&files[i]).Error; err != nil {
			f.t.Fatalf("create file: %v", err)
		}
	}

	return f, item.ID
}

func hasDecisionReasonCode(reasons []DecisionReason, want string) bool {
	for _, reason := range reasons {
		if reason.Code == want {
			return true
		}
	}
	return false
}

func intPtr(value int) *int {
	return &value
}
