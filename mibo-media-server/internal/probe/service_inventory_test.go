package probe

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/providers"
	"gorm.io/gorm"
)

func TestProbeInventoryFileUpdatesAssetsAndStreams(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := newInventoryProbeFixture(t)

	service := NewService(fixture.db, fixture.registry, fixture.cfg.FFprobe)
	prober, ok := any(service).(interface {
		ProbeInventoryFile(context.Context, uint) error
	})
	if !ok {
		t.Fatalf("service does not implement ProbeInventoryFile")
	}

	if err := prober.ProbeInventoryFile(ctx, fixture.file.ID); err != nil {
		t.Fatalf("probe inventory file: %v", err)
	}

	var streams []database.MediaStream
	if err := fixture.db.WithContext(ctx).Order("stream_index asc").Find(&streams, "file_id = ?", fixture.file.ID).Error; err != nil {
		t.Fatalf("load streams: %v", err)
	}
	if len(streams) != 3 {
		t.Fatalf("expected 3 streams, got %d", len(streams))
	}
	if streams[0].StreamIndex != 0 || streams[0].StreamType != "video" || streams[0].Codec != "h264" {
		t.Fatalf("unexpected video stream: %#v", streams[0])
	}
	if streams[1].StreamIndex != 1 || streams[1].StreamType != "audio" || streams[1].Codec != "aac" {
		t.Fatalf("unexpected audio stream: %#v", streams[1])
	}
	if streams[2].StreamIndex != 2 || streams[2].StreamType != "subtitle" || streams[2].Codec != "subrip" {
		t.Fatalf("unexpected subtitle stream: %#v", streams[2])
	}

	var asset database.MediaAsset
	if err := fixture.db.WithContext(ctx).First(&asset, fixture.asset.ID).Error; err != nil {
		t.Fatalf("load asset: %v", err)
	}
	if asset.ProbeStatus != StatusReady {
		t.Fatalf("expected probe status %q, got %q", StatusReady, asset.ProbeStatus)
	}
	if asset.DurationSeconds == nil || *asset.DurationSeconds <= 0 {
		t.Fatalf("expected asset duration to be set, got %#v", asset.DurationSeconds)
	}
	if strings.TrimSpace(asset.TechnicalSummaryJSON) == "" {
		t.Fatalf("expected technical summary json to be populated")
	}
	var summary map[string]any
	if err := json.Unmarshal([]byte(asset.TechnicalSummaryJSON), &summary); err != nil {
		t.Fatalf("technical summary should be valid json: %v", err)
	}
	if strings.Contains(asset.TechnicalSummaryJSON, `"streams"`) || strings.Contains(asset.TechnicalSummaryJSON, `"format"`) {
		t.Fatalf("expected compact technical summary, got raw ffprobe payload: %s", asset.TechnicalSummaryJSON)
	}

	var item database.CatalogItem
	if err := fixture.db.WithContext(ctx).First(&item, fixture.item.ID).Error; err != nil {
		t.Fatalf("load catalog item: %v", err)
	}
	if item.RuntimeSeconds == nil || *item.RuntimeSeconds != 7260 {
		t.Fatalf("expected runtime_seconds=7260, got %#v", item.RuntimeSeconds)
	}

	var legacyCount int64
	if err := fixture.db.WithContext(ctx).Model(&database.MediaFile{}).Count(&legacyCount).Error; err != nil {
		t.Fatalf("count legacy media files: %v", err)
	}
	if legacyCount != 0 {
		t.Fatalf("expected legacy media_files to remain untouched, got %d rows", legacyCount)
	}
}

func TestProbeInventoryFileAllowsSameStreamIndexesAcrossDifferentFiles(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := newInventoryProbeFixture(t)
	service := NewService(fixture.db, fixture.registry, fixture.cfg.FFprobe)
	inventorySvc := inventory.NewService(fixture.db)

	secondPath := filepath.Join(filepath.Dir(fixture.file.StoragePath), "Movie B.2024.mkv")
	if err := os.WriteFile(secondPath, []byte("movie-b"), 0o644); err != nil {
		t.Fatalf("write second media file: %v", err)
	}

	secondItem := database.CatalogItem{
		LibraryID:          fixture.file.LibraryID,
		Type:               "movie",
		Path:               secondPath,
		SortKey:            "Movie B",
		DisplayOrder:       "aired",
		Title:              "Movie B",
		AvailabilityStatus: "available",
		GovernanceStatus:   "pending",
	}
	if err := fixture.db.WithContext(ctx).Create(&secondItem).Error; err != nil {
		t.Fatalf("create second catalog item: %v", err)
	}

	secondAsset, err := inventorySvc.CreateAsset(ctx, inventory.CreateAssetInput{
		LibraryID:   fixture.file.LibraryID,
		AssetType:   inventory.AssetTypeMain,
		DisplayName: "Movie B",
		ProbeStatus: StatusPending,
	})
	if err != nil {
		t.Fatalf("create second asset: %v", err)
	}
	secondFile, err := inventorySvc.UpsertFile(ctx, inventory.UpsertFileInput{
		LibraryID:         fixture.file.LibraryID,
		StorageProvider:   "local",
		StoragePath:       secondPath,
		StableIdentityKey: "stable-movie-b",
		SizeBytes:         int64(len("movie-b")),
		Container:         "mkv",
		Status:            inventory.FileStatusAvailable,
	})
	if err != nil {
		t.Fatalf("create second inventory file: %v", err)
	}
	if _, err := inventorySvc.LinkAssetToItem(ctx, inventory.LinkAssetItemInput{AssetID: secondAsset.ID, ItemID: secondItem.ID, Role: inventory.AssetItemRolePrimary, Source: "scanner"}); err != nil {
		t.Fatalf("link second asset to item: %v", err)
	}
	if _, err := inventorySvc.LinkAssetToFile(ctx, inventory.LinkAssetFileInput{AssetID: secondAsset.ID, FileID: secondFile.ID, Role: inventory.FileRoleSource}); err != nil {
		t.Fatalf("link second asset to file: %v", err)
	}

	if err := service.ProbeInventoryFile(ctx, fixture.file.ID); err != nil {
		t.Fatalf("probe first inventory file: %v", err)
	}
	if err := service.ProbeInventoryFile(ctx, secondFile.ID); err != nil {
		t.Fatalf("probe second inventory file: %v", err)
	}

	var streamCount int64
	if err := fixture.db.WithContext(ctx).Model(&database.MediaStream{}).Count(&streamCount).Error; err != nil {
		t.Fatalf("count streams: %v", err)
	}
	if streamCount != 6 {
		t.Fatalf("expected 6 total media_stream rows across both files, got %d", streamCount)
	}
}

func TestProbeInventoryFileGeneratesCatalogFallbackArtworkWithoutRemoteImages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := newInventoryProbeFixture(t)
	artworkRoot := filepath.Join(t.TempDir(), "artwork")
	fixture.cfg.FFmpeg = config.FFmpegConfig{Enabled: true, Path: writeInventoryFakeFFmpeg(t), Timeout: time.Second, ArtworkRootPath: artworkRoot}

	service := NewService(fixture.db, fixture.registry, fixture.cfg.FFprobe, fixture.cfg.FFmpeg)
	if err := service.ProbeInventoryFile(ctx, fixture.file.ID); err != nil {
		t.Fatalf("probe inventory file: %v", err)
	}

	var images []database.ItemImage
	if err := fixture.db.WithContext(ctx).Where("item_id = ?", fixture.item.ID).Order("image_type asc").Find(&images).Error; err != nil {
		t.Fatalf("load generated catalog images: %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("expected generated poster and backdrop, got %#v", images)
	}
	for _, image := range images {
		if !image.IsSelected {
			t.Fatalf("expected generated image to be selected, got %#v", image)
		}
		if !strings.HasPrefix(image.URL, "/api/v1/items/") {
			t.Fatalf("expected catalog artwork url, got %#v", image)
		}
	}
	itemID := strings.TrimSpace(strconv.FormatUint(uint64(fixture.item.ID), 10))
	posterPath := filepath.Join(artworkRoot, "catalog", itemID, "poster.jpg")
	backdropPath := filepath.Join(artworkRoot, "catalog", itemID, "backdrop.jpg")
	posterBytes, err := os.ReadFile(posterPath)
	if err != nil {
		t.Fatalf("read generated poster: %v", err)
	}
	if string(posterBytes) != "fake-artwork" {
		t.Fatalf("unexpected poster bytes: %q", string(posterBytes))
	}
	if _, err := os.Stat(backdropPath); err != nil {
		t.Fatalf("expected generated backdrop file: %v", err)
	}
}

func TestProbeInventoryFileSkipsCatalogFallbackArtworkWhenRemoteImagesExist(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := newInventoryProbeFixture(t)
	artworkRoot := filepath.Join(t.TempDir(), "artwork")
	fixture.cfg.FFmpeg = config.FFmpegConfig{Enabled: true, Path: writeInventoryFakeFFmpeg(t), Timeout: time.Second, ArtworkRootPath: artworkRoot}

	if err := fixture.db.WithContext(ctx).Create(&database.ItemImage{
		ItemID:     fixture.item.ID,
		ImageType:  "poster",
		URL:        "https://image.tmdb.org/t/p/original/poster.jpg",
		IsSelected: true,
	}).Error; err != nil {
		t.Fatalf("seed remote artwork: %v", err)
	}

	service := NewService(fixture.db, fixture.registry, fixture.cfg.FFprobe, fixture.cfg.FFmpeg)
	if err := service.ProbeInventoryFile(ctx, fixture.file.ID); err != nil {
		t.Fatalf("probe inventory file: %v", err)
	}

	var images []database.ItemImage
	if err := fixture.db.WithContext(ctx).Where("item_id = ?", fixture.item.ID).Order("id asc").Find(&images).Error; err != nil {
		t.Fatalf("load artwork rows: %v", err)
	}
	if len(images) != 1 || images[0].URL != "https://image.tmdb.org/t/p/original/poster.jpg" {
		t.Fatalf("expected remote artwork to remain untouched, got %#v", images)
	}
	itemID := strings.TrimSpace(strconv.FormatUint(uint64(fixture.item.ID), 10))
	if _, err := os.Stat(filepath.Join(artworkRoot, "catalog", itemID, "poster.jpg")); !os.IsNotExist(err) {
		t.Fatalf("expected no generated poster file, got err=%v", err)
	}
}

type inventoryProbeFixture struct {
	cfg      config.Config
	db       *gorm.DB
	registry *providers.Registry
	item     database.CatalogItem
	asset    database.MediaAsset
	file     database.InventoryFile
}

func newInventoryProbeFixture(t *testing.T) inventoryProbeFixture {
	t.Helper()

	mediaRoot := filepath.Join(t.TempDir(), "media-root")
	moviesRoot := filepath.Join(mediaRoot, "Movies")
	filePath := filepath.Join(moviesRoot, "Movie A.2024.mkv")
	if err := os.MkdirAll(moviesRoot, 0o755); err != nil {
		t.Fatalf("create media root: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("movie"), 0o644); err != nil {
		t.Fatalf("write media file: %v", err)
	}

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{
		Local:   config.LocalStorageConfig{RootPath: mediaRoot},
		FFprobe: config.FFprobeConfig{Enabled: true, Path: writeInventoryFakeFFprobe(t), Timeout: time.Second},
	}
	registry := providers.NewRegistry(cfg)
	jobsSvc := jobs.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	inventorySvc := inventory.NewService(db)
	ctx := context.Background()

	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Local", RootPath: mediaRoot})
	if err != nil {
		t.Fatalf("create media source: %v", err)
	}
	libraryRecord, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: moviesRoot})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}

	item := database.CatalogItem{
		LibraryID:          libraryRecord.ID,
		Type:               "movie",
		Path:               filePath,
		SortKey:            "Movie A",
		DisplayOrder:       "aired",
		Title:              "Movie A",
		AvailabilityStatus: "available",
		GovernanceStatus:   "pending",
	}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create catalog item: %v", err)
	}

	asset, err := inventorySvc.CreateAsset(ctx, inventory.CreateAssetInput{
		LibraryID:   libraryRecord.ID,
		AssetType:   inventory.AssetTypeMain,
		DisplayName: "Movie A",
		ProbeStatus: StatusPending,
	})
	if err != nil {
		t.Fatalf("create asset: %v", err)
	}

	file, err := inventorySvc.UpsertFile(ctx, inventory.UpsertFileInput{
		LibraryID:         libraryRecord.ID,
		StorageProvider:   "local",
		StoragePath:       filePath,
		StableIdentityKey: "stable-movie-a",
		SizeBytes:         5,
		Container:         "mkv",
		Status:            inventory.FileStatusAvailable,
	})
	if err != nil {
		t.Fatalf("create inventory file: %v", err)
	}

	if _, err := inventorySvc.LinkAssetToItem(ctx, inventory.LinkAssetItemInput{AssetID: asset.ID, ItemID: item.ID, Role: inventory.AssetItemRolePrimary, Source: "scanner"}); err != nil {
		t.Fatalf("link asset to item: %v", err)
	}
	if _, err := inventorySvc.LinkAssetToFile(ctx, inventory.LinkAssetFileInput{AssetID: asset.ID, FileID: file.ID, Role: inventory.FileRoleSource}); err != nil {
		t.Fatalf("link asset to file: %v", err)
	}

	return inventoryProbeFixture{cfg: cfg, db: db, registry: registry, item: item, asset: asset, file: file}
}

func writeInventoryFakeFFprobe(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ffprobe")
	content := "#!/bin/sh\ncat <<'EOF'\n{\"streams\":[{\"codec_type\":\"video\",\"codec_name\":\"h264\",\"width\":1920,\"height\":1080},{\"codec_type\":\"audio\",\"codec_name\":\"aac\",\"channels\":2,\"tags\":{\"language\":\"eng\",\"title\":\"Stereo\"}},{\"codec_type\":\"subtitle\",\"codec_name\":\"subrip\",\"tags\":{\"language\":\"eng\",\"title\":\"English\"}}],\"format\":{\"duration\":\"7260.25\",\"bit_rate\":\"5000000\"}}\nEOF\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake ffprobe: %v", err)
	}
	return path
}

func writeInventoryFakeFFmpeg(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ffmpeg")
	content := "#!/bin/sh\nout=\"\"\nfor arg in \"$@\"; do\n  out=\"$arg\"\ndone\nmkdir -p \"$(dirname \"$out\")\"\nprintf 'fake-artwork' > \"$out\"\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake ffmpeg: %v", err)
	}
	return path
}
