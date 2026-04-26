package playback

import (
	"context"
	"fmt"
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

func TestCatalogPlaybackDirectFromAssetAndInventoryFile(t *testing.T) {
	fixture := newPlaybackDecisionFixture(t)
	runtimeSeconds := 1800
	item := database.CatalogItem{LibraryID: fixture.library.ID, Type: "episode", Title: "Catalog Episode", RuntimeSeconds: &runtimeSeconds, AvailabilityStatus: "available", GovernanceStatus: "matched"}
	if err := fixture.db.WithContext(context.Background()).Create(&item).Error; err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	asset := database.MediaAsset{LibraryID: fixture.library.ID, AssetType: "main", Status: "available", ProbeStatus: probe.StatusReady}
	if err := fixture.db.WithContext(context.Background()).Create(&asset).Error; err != nil {
		t.Fatalf("create media asset: %v", err)
	}
	filePath := filepath.Join(fixture.rootPath, "catalog-episode.mp4")
	if err := os.WriteFile(filePath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write inventory file: %v", err)
	}
	file := database.InventoryFile{LibraryID: fixture.library.ID, StorageProvider: "local", StoragePath: filePath, Container: "mp4", Status: "available"}
	if err := fixture.db.WithContext(context.Background()).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file row: %v", err)
	}
	if err := fixture.db.WithContext(context.Background()).Create(&database.AssetItem{AssetID: asset.ID, ItemID: item.ID, Role: "primary", SegmentIndex: 0}).Error; err != nil {
		t.Fatalf("create asset item: %v", err)
	}
	if err := fixture.db.WithContext(context.Background()).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: "source", PartIndex: 0}).Error; err != nil {
		t.Fatalf("create asset file: %v", err)
	}
	width := 1280
	height := 720
	if err := fixture.db.WithContext(context.Background()).Create(&database.MediaStream{FileID: file.ID, StreamIndex: 0, StreamType: "video", Codec: "h264", Width: &width, Height: &height}).Error; err != nil {
		t.Fatalf("create video stream: %v", err)
	}

	source, err := fixture.service.GetPlaybackSource(context.Background(), PlaybackRequest{ItemID: item.ID, ClientProfile: ClientProfileWeb})
	if err != nil {
		t.Fatalf("get catalog playback source: %v", err)
	}
	if source.ItemID != item.ID || source.AssetID != asset.ID || source.FileID != file.ID {
		t.Fatalf("unexpected catalog playback ids: %#v", source)
	}
	if source.Decision.Kind != "direct" || !source.Playable || source.URL == "" {
		t.Fatalf("expected direct playable catalog source, got %#v", source)
	}
}

func TestCatalogPlaybackReturnsUnplayableWhenNoAssetAvailable(t *testing.T) {
	fixture := newPlaybackDecisionFixture(t)
	item := database.CatalogItem{LibraryID: fixture.library.ID, Type: "movie", Title: "Missing Catalog Movie", AvailabilityStatus: "missing", GovernanceStatus: "pending"}
	if err := fixture.db.WithContext(context.Background()).Create(&item).Error; err != nil {
		t.Fatalf("create catalog item: %v", err)
	}

	source, err := fixture.service.GetPlaybackSource(context.Background(), PlaybackRequest{ItemID: item.ID, ClientProfile: ClientProfileWeb})
	if err != nil {
		t.Fatalf("get catalog playback source: %v", err)
	}
	if source.Decision.Kind != "unplayable" || source.Playable {
		t.Fatalf("expected unplayable decision, got %#v", source)
	}
	if !hasDecisionReasonCode(source.Decision.Reasons, "no_available_asset") {
		t.Fatalf("expected no_available_asset reason, got %#v", source.Decision.Reasons)
	}
}

func TestCatalogPlaybackHonorsPreferredAssetSelection(t *testing.T) {
	fixture := newPlaybackDecisionFixture(t)
	runtimeSeconds := 1800
	item := database.CatalogItem{LibraryID: fixture.library.ID, Type: "episode", Title: "Catalog Episode", RuntimeSeconds: &runtimeSeconds, AvailabilityStatus: "available", GovernanceStatus: "matched"}
	if err := fixture.db.WithContext(context.Background()).Create(&item).Error; err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	mainAsset := database.MediaAsset{LibraryID: fixture.library.ID, AssetType: "main", Status: "available", ProbeStatus: probe.StatusReady, QualityLabel: "720p"}
	versionAsset := database.MediaAsset{LibraryID: fixture.library.ID, AssetType: "version", Status: "available", ProbeStatus: probe.StatusReady, QualityLabel: "1080p"}
	for _, asset := range []*database.MediaAsset{&mainAsset, &versionAsset} {
		if err := fixture.db.WithContext(context.Background()).Create(asset).Error; err != nil {
			t.Fatalf("create asset: %v", err)
		}
	}
	for idx, asset := range []*database.MediaAsset{&mainAsset, &versionAsset} {
		filePath := filepath.Join(fixture.rootPath, fmt.Sprintf("catalog-version-%d.mp4", idx))
		if err := os.WriteFile(filePath, []byte("video"), 0o644); err != nil {
			t.Fatalf("write inventory file: %v", err)
		}
		file := database.InventoryFile{LibraryID: fixture.library.ID, StorageProvider: "local", StoragePath: filePath, Container: "mp4", Status: "available"}
		if err := fixture.db.WithContext(context.Background()).Create(&file).Error; err != nil {
			t.Fatalf("create inventory file: %v", err)
		}
		if err := fixture.db.WithContext(context.Background()).Create(&database.AssetItem{AssetID: asset.ID, ItemID: item.ID, Role: "primary", SegmentIndex: 0}).Error; err != nil {
			t.Fatalf("create asset item: %v", err)
		}
		if err := fixture.db.WithContext(context.Background()).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: "source", PartIndex: 0}).Error; err != nil {
			t.Fatalf("create asset file: %v", err)
		}
		width := 1280 + idx*640
		height := 720 + idx*360
		if err := fixture.db.WithContext(context.Background()).Create(&database.MediaStream{FileID: file.ID, StreamIndex: 0, StreamType: "video", Codec: "h264", Width: &width, Height: &height}).Error; err != nil {
			t.Fatalf("create media stream: %v", err)
		}
	}

	source, err := fixture.service.GetPlaybackSource(context.Background(), PlaybackRequest{ItemID: item.ID, AssetID: versionAsset.ID, ClientProfile: ClientProfileWeb})
	if err != nil {
		t.Fatalf("get preferred catalog playback source: %v", err)
	}
	if source.AssetID != versionAsset.ID || source.QualityLabel != "1080p" || source.Decision.SelectedBy != "preferred_asset" {
		t.Fatalf("unexpected preferred asset playback source: %#v", source)
	}
}

func TestCatalogPlaybackSupportsMultiEpisodeAssetForLinkedItem(t *testing.T) {
	fixture := newPlaybackDecisionFixture(t)
	seasonNumber := 1
	episodeOneNumber := 1
	episodeTwoNumber := 2
	first := database.CatalogItem{LibraryID: fixture.library.ID, Type: "episode", Title: "Episode 1", ParentIndexNumber: &seasonNumber, IndexNumber: &episodeOneNumber, AvailabilityStatus: "available", GovernanceStatus: "matched"}
	second := database.CatalogItem{LibraryID: fixture.library.ID, Type: "episode", Title: "Episode 2", ParentIndexNumber: &seasonNumber, IndexNumber: &episodeTwoNumber, AvailabilityStatus: "available", GovernanceStatus: "matched"}
	for _, item := range []*database.CatalogItem{&first, &second} {
		if err := fixture.db.WithContext(context.Background()).Create(item).Error; err != nil {
			fixture.t.Fatalf("create catalog item: %v", err)
		}
	}
	asset := database.MediaAsset{LibraryID: fixture.library.ID, AssetType: "main", Status: "available", ProbeStatus: probe.StatusReady}
	if err := fixture.db.WithContext(context.Background()).Create(&asset).Error; err != nil {
		fixture.t.Fatalf("create media asset: %v", err)
	}
	filePath := filepath.Join(fixture.rootPath, "multi-episode.mp4")
	if err := os.WriteFile(filePath, []byte("video"), 0o644); err != nil {
		fixture.t.Fatalf("write inventory file: %v", err)
	}
	file := database.InventoryFile{LibraryID: fixture.library.ID, StorageProvider: "local", StoragePath: filePath, Container: "mp4", Status: "available"}
	if err := fixture.db.WithContext(context.Background()).Create(&file).Error; err != nil {
		fixture.t.Fatalf("create inventory file: %v", err)
	}
	for idx, itemID := range []uint{first.ID, second.ID} {
		if err := fixture.db.WithContext(context.Background()).Create(&database.AssetItem{AssetID: asset.ID, ItemID: itemID, Role: "primary", SegmentIndex: idx}).Error; err != nil {
			fixture.t.Fatalf("create asset item link: %v", err)
		}
	}
	if err := fixture.db.WithContext(context.Background()).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: "source", PartIndex: 0}).Error; err != nil {
		fixture.t.Fatalf("create asset file link: %v", err)
	}
	width := 1280
	height := 720
	if err := fixture.db.WithContext(context.Background()).Create(&database.MediaStream{FileID: file.ID, StreamIndex: 0, StreamType: "video", Codec: "h264", Width: &width, Height: &height}).Error; err != nil {
		fixture.t.Fatalf("create media stream: %v", err)
	}

	source, err := fixture.service.GetPlaybackSource(context.Background(), PlaybackRequest{ItemID: second.ID, ClientProfile: ClientProfileWeb})
	if err != nil {
		fixture.t.Fatalf("get playback source: %v", err)
	}
	if source.ItemID != second.ID || source.AssetID != asset.ID || source.FileID != file.ID || !source.Playable {
		fixture.t.Fatalf("unexpected multi-episode playback source: %#v", source)
	}
}

func TestCatalogPlaybackReturnsUnplayableForMissingInventoryFile(t *testing.T) {
	fixture := newPlaybackDecisionFixture(t)
	item := database.CatalogItem{LibraryID: fixture.library.ID, Type: "movie", Title: "Missing File Movie", AvailabilityStatus: "available", GovernanceStatus: "matched"}
	if err := fixture.db.WithContext(context.Background()).Create(&item).Error; err != nil {
		fixture.t.Fatalf("create catalog item: %v", err)
	}
	asset := database.MediaAsset{LibraryID: fixture.library.ID, AssetType: "main", Status: "available", ProbeStatus: probe.StatusReady}
	if err := fixture.db.WithContext(context.Background()).Create(&asset).Error; err != nil {
		fixture.t.Fatalf("create media asset: %v", err)
	}
	file := database.InventoryFile{LibraryID: fixture.library.ID, StorageProvider: "local", StoragePath: filepath.Join(fixture.rootPath, "missing.mp4"), Container: "mp4", Status: "available"}
	if err := fixture.db.WithContext(context.Background()).Create(&file).Error; err != nil {
		fixture.t.Fatalf("create inventory file: %v", err)
	}
	if err := fixture.db.WithContext(context.Background()).Create(&database.AssetItem{AssetID: asset.ID, ItemID: item.ID, Role: "primary", SegmentIndex: 0}).Error; err != nil {
		fixture.t.Fatalf("create asset item link: %v", err)
	}
	if err := fixture.db.WithContext(context.Background()).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: "source", PartIndex: 0}).Error; err != nil {
		fixture.t.Fatalf("create asset file link: %v", err)
	}

	source, err := fixture.service.GetPlaybackSource(context.Background(), PlaybackRequest{ItemID: item.ID, ClientProfile: ClientProfileWeb})
	if err != nil {
		fixture.t.Fatalf("get playback source: %v", err)
	}
	if source.Playable || source.Decision.Kind != "unplayable" {
		fixture.t.Fatalf("expected unplayable missing-file source, got %#v", source)
	}
	if !hasDecisionReasonCode(source.Decision.Reasons, "source_unavailable") {
		fixture.t.Fatalf("expected source_unavailable reason, got %#v", source.Decision.Reasons)
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
		Name:           "Movies",
		Type:           "movies",
		MediaSourceID:  source.ID,
		RootPath:       rootPath,
		Status:         "active",
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
