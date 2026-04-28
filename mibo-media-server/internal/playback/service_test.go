package playback

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog/seriesplayback"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/probe"
	"github.com/atlan/mibo-media-server/internal/providers"
	"gorm.io/gorm"
)

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

func TestCatalogPlaybackResolvesSeriesToPlayableEpisode(t *testing.T) {
	fixture := newPlaybackDecisionFixture(t)
	seasonNumber := 1
	episodeOneNumber := 1
	episodeTwoNumber := 2
	series := database.CatalogItem{LibraryID: fixture.library.ID, Type: "series", Title: "Catalog Series", AvailabilityStatus: "available", GovernanceStatus: "matched"}
	if err := fixture.db.WithContext(context.Background()).Create(&series).Error; err != nil {
		fixture.t.Fatalf("create series: %v", err)
	}
	season := database.CatalogItem{LibraryID: fixture.library.ID, Type: "season", ParentID: &series.ID, Title: "Season 1", IndexNumber: &seasonNumber, AvailabilityStatus: "available", GovernanceStatus: "matched"}
	if err := fixture.db.WithContext(context.Background()).Create(&season).Error; err != nil {
		fixture.t.Fatalf("create season: %v", err)
	}
	first := database.CatalogItem{LibraryID: fixture.library.ID, Type: "episode", ParentID: &season.ID, Title: "Episode 1", ParentIndexNumber: &seasonNumber, IndexNumber: &episodeOneNumber, AvailabilityStatus: "available", GovernanceStatus: "matched"}
	second := database.CatalogItem{LibraryID: fixture.library.ID, Type: "episode", ParentID: &season.ID, Title: "Episode 2", ParentIndexNumber: &seasonNumber, IndexNumber: &episodeTwoNumber, AvailabilityStatus: "available", GovernanceStatus: "matched"}
	for _, item := range []*database.CatalogItem{&first, &second} {
		if err := fixture.db.WithContext(context.Background()).Create(item).Error; err != nil {
			fixture.t.Fatalf("create episode: %v", err)
		}
	}
	createPlayablePlaybackAsset(t, fixture, first.ID, "series-episode-one.mp4")
	secondAsset, secondFile := createPlayablePlaybackAsset(t, fixture, second.ID, "series-episode-two.mp4")
	userID := uint(7)
	lastPlayed := time.Now().UTC()
	if err := fixture.db.WithContext(context.Background()).Create(&database.UserItemData{UserID: userID, ItemID: second.ID, AssetID: &secondAsset.ID, PositionSeconds: 300, LastPlayedAt: &lastPlayed}).Error; err != nil {
		fixture.t.Fatalf("create progress: %v", err)
	}
	target, err := seriesplayback.Select(context.Background(), fixture.db, series.ID, &userID)
	if err != nil {
		fixture.t.Fatalf("select series playback target: %v", err)
	}
	if target == nil || target.EpisodeID != second.ID {
		fixture.t.Fatalf("unexpected series playback target: %#v", target)
	}

	source, err := fixture.service.GetPlaybackSource(context.Background(), PlaybackRequest{ItemID: series.ID, UserID: &userID, ClientProfile: ClientProfileWeb})
	if err != nil {
		fixture.t.Fatalf("get series playback source: %v", err)
	}
	if source.ItemID != second.ID || source.AssetID != secondAsset.ID || source.FileID != secondFile.ID || !source.Playable {
		fixture.t.Fatalf("unexpected resolved series playback source: %#v", source)
	}
}

func TestCatalogPlaybackReturnsUnplayableForSeriesWithoutLocalEpisodes(t *testing.T) {
	fixture := newPlaybackDecisionFixture(t)
	seasonNumber := 1
	episodeNumber := 1
	series := database.CatalogItem{LibraryID: fixture.library.ID, Type: "series", Title: "Missing Series", AvailabilityStatus: "missing", GovernanceStatus: "matched"}
	if err := fixture.db.WithContext(context.Background()).Create(&series).Error; err != nil {
		fixture.t.Fatalf("create series: %v", err)
	}
	season := database.CatalogItem{LibraryID: fixture.library.ID, Type: "season", ParentID: &series.ID, Title: "Season 1", IndexNumber: &seasonNumber, AvailabilityStatus: "missing", GovernanceStatus: "matched"}
	if err := fixture.db.WithContext(context.Background()).Create(&season).Error; err != nil {
		fixture.t.Fatalf("create season: %v", err)
	}
	missing := database.CatalogItem{LibraryID: fixture.library.ID, Type: "episode", ParentID: &season.ID, Title: "Missing Episode", ParentIndexNumber: &seasonNumber, IndexNumber: &episodeNumber, AvailabilityStatus: "missing", GovernanceStatus: "matched"}
	if err := fixture.db.WithContext(context.Background()).Create(&missing).Error; err != nil {
		fixture.t.Fatalf("create missing episode: %v", err)
	}

	source, err := fixture.service.GetPlaybackSource(context.Background(), PlaybackRequest{ItemID: series.ID, ClientProfile: ClientProfileWeb})
	if err != nil {
		fixture.t.Fatalf("get series playback source: %v", err)
	}
	if source.Playable || source.Decision.Kind != "unplayable" || !hasDecisionReasonCode(source.Decision.Reasons, "series_has_no_playable_episode") {
		fixture.t.Fatalf("expected unplayable series decision, got %#v", source)
	}
}

type playbackDecisionFixture struct {
	t        *testing.T
	db       *gorm.DB
	service  *Service
	library  database.Library
	rootPath string
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

func hasDecisionReasonCode(reasons []DecisionReason, want string) bool {
	for _, reason := range reasons {
		if reason.Code == want {
			return true
		}
	}
	return false
}

func createPlayablePlaybackAsset(t *testing.T, fixture *playbackDecisionFixture, itemID uint, name string) (database.MediaAsset, database.InventoryFile) {
	t.Helper()
	asset := database.MediaAsset{LibraryID: fixture.library.ID, AssetType: "main", Status: "available", ProbeStatus: probe.StatusReady}
	if err := fixture.db.WithContext(context.Background()).Create(&asset).Error; err != nil {
		t.Fatalf("create media asset: %v", err)
	}
	filePath := filepath.Join(fixture.rootPath, name)
	if err := os.WriteFile(filePath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write inventory file: %v", err)
	}
	file := database.InventoryFile{LibraryID: fixture.library.ID, StorageProvider: "local", StoragePath: filePath, Container: "mp4", Status: "available"}
	if err := fixture.db.WithContext(context.Background()).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	if err := fixture.db.WithContext(context.Background()).Create(&database.AssetItem{AssetID: asset.ID, ItemID: itemID, Role: "primary", SegmentIndex: 0}).Error; err != nil {
		t.Fatalf("create asset item: %v", err)
	}
	if err := fixture.db.WithContext(context.Background()).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: "source", PartIndex: 0}).Error; err != nil {
		t.Fatalf("create asset file: %v", err)
	}
	width := 1280
	height := 720
	if err := fixture.db.WithContext(context.Background()).Create(&database.MediaStream{FileID: file.ID, StreamIndex: 0, StreamType: "video", Codec: "h264", Width: &width, Height: &height}).Error; err != nil {
		t.Fatalf("create media stream: %v", err)
	}
	return asset, file
}
