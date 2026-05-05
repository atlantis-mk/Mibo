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
	assertDetailedVideoStream(t, streams[0])
	if streams[1].StreamIndex != 1 || streams[1].StreamType != "audio" || streams[1].Codec != "flac" {
		t.Fatalf("unexpected audio stream: %#v", streams[1])
	}
	if streams[1].ChannelLayout != "stereo" || streams[1].SampleRate == nil || *streams[1].SampleRate != 48000 || streams[1].BitDepth == nil || *streams[1].BitDepth != 24 {
		t.Fatalf("unexpected detailed audio stream: %#v", streams[1])
	}
	if streams[1].BitRate != nil {
		t.Fatalf("expected audio stream bitrate to remain empty without stream bitrate, got %#v", streams[1].BitRate)
	}
	if streams[2].StreamIndex != 2 || streams[2].StreamType != "subtitle" || streams[2].Codec != "ass" {
		t.Fatalf("unexpected subtitle stream: %#v", streams[2])
	}
	if streams[2].Title != "Chinese Traditional" || streams[2].Language != "zho" || !strings.Contains(streams[2].DispositionJSON, `"default":true`) || !strings.Contains(streams[2].DispositionJSON, `"external":true`) {
		t.Fatalf("unexpected subtitle stream metadata: %#v", streams[2])
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

}

func TestProbeInventoryFileDoesNotOverwriteLockedRuntime(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := newInventoryProbeFixture(t)
	lockedRuntime := 5400
	if err := fixture.db.WithContext(ctx).Create(&database.MetadataFieldState{ItemID: fixture.item.ID, FieldKey: "runtime_seconds", ValueJSON: `5400`, IsLocked: true, LockReason: "operator lock"}).Error; err != nil {
		t.Fatalf("seed locked runtime field: %v", err)
	}
	if err := fixture.db.WithContext(ctx).Model(&database.CatalogItem{}).Where("id = ?", fixture.item.ID).Update("runtime_seconds", lockedRuntime).Error; err != nil {
		t.Fatalf("seed locked runtime value: %v", err)
	}

	service := NewService(fixture.db, fixture.registry, fixture.cfg.FFprobe)
	if err := service.ProbeInventoryFile(ctx, fixture.file.ID); err != nil {
		t.Fatalf("probe inventory file: %v", err)
	}

	var item database.CatalogItem
	if err := fixture.db.WithContext(ctx).First(&item, fixture.item.ID).Error; err != nil {
		t.Fatalf("load catalog item: %v", err)
	}
	if item.RuntimeSeconds == nil || *item.RuntimeSeconds != lockedRuntime {
		t.Fatalf("expected locked runtime_seconds=%d, got %#v", lockedRuntime, item.RuntimeSeconds)
	}
}

func TestBuildInventoryMediaStreamsAllowsSparseVideoMetadata(t *testing.T) {
	t.Parallel()

	streams := buildInventoryMediaStreams(10, ffprobeOutput{Streams: []ffprobeStream{{CodecType: "video", CodecName: "hevc", Width: 3840, Height: 2160}}}, map[string]any{})
	if len(streams) != 1 {
		t.Fatalf("expected 1 stream, got %d", len(streams))
	}
	stream := streams[0]
	if stream.Codec != "hevc" || stream.Width == nil || *stream.Width != 3840 || stream.Height == nil || *stream.Height != 2160 {
		t.Fatalf("expected compact video metadata to be preserved, got %#v", stream)
	}
	if stream.Profile != "" || stream.Level != nil || stream.AvgFrameRate != "" || stream.BitDepth != nil || stream.ReferenceFrames != nil || stream.BitRate != nil {
		t.Fatalf("expected sparse optional technical fields to stay empty, got %#v", stream)
	}
}

func assertDetailedVideoStream(t *testing.T, stream database.MediaStream) {
	t.Helper()

	if stream.Profile != "High" {
		t.Fatalf("expected profile High, got %q", stream.Profile)
	}
	if stream.Level == nil || *stream.Level != 41 {
		t.Fatalf("expected level 41, got %#v", stream.Level)
	}
	if stream.AvgFrameRate != "24000/1001" || stream.RFrameRate != "24000/1001" {
		t.Fatalf("unexpected frame rates: avg=%q r=%q", stream.AvgFrameRate, stream.RFrameRate)
	}
	if stream.FieldOrder != "progressive" || stream.ColorSpace != "bt709" || stream.PixelFormat != "yuv420p10le" {
		t.Fatalf("unexpected color/interlace fields: %#v", stream)
	}
	if stream.BitDepth == nil || *stream.BitDepth != 10 {
		t.Fatalf("expected bit depth 10, got %#v", stream.BitDepth)
	}
	if stream.ReferenceFrames == nil || *stream.ReferenceFrames != 4 {
		t.Fatalf("expected reference frames 4, got %#v", stream.ReferenceFrames)
	}
	if stream.BitRate == nil || *stream.BitRate != 4200000 {
		t.Fatalf("expected stream bitrate 4200000, got %#v", stream.BitRate)
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

func TestProbeInventoryFileUsesSiblingArtworkBeforeFrameExtraction(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := newInventoryProbeFixture(t)
	artworkRoot := filepath.Join(t.TempDir(), "artwork")
	fixture.cfg.FFmpeg = config.FFmpegConfig{Enabled: true, Path: writeFailingInventoryFakeFFmpeg(t), Timeout: time.Second, ArtworkRootPath: artworkRoot}

	posterSource := filepath.Join(filepath.Dir(fixture.file.StoragePath), "cover.jpg")
	backdropSource := filepath.Join(filepath.Dir(fixture.file.StoragePath), "background.png")
	if err := os.WriteFile(posterSource, []byte("local-poster"), 0o644); err != nil {
		t.Fatalf("write local poster: %v", err)
	}
	if err := os.WriteFile(backdropSource, []byte("local-backdrop"), 0o644); err != nil {
		t.Fatalf("write local backdrop: %v", err)
	}

	service := NewService(fixture.db, fixture.registry, fixture.cfg.FFprobe, fixture.cfg.FFmpeg)
	if err := service.ProbeInventoryFile(ctx, fixture.file.ID); err != nil {
		t.Fatalf("probe inventory file: %v", err)
	}

	var images []database.ItemImage
	if err := fixture.db.WithContext(ctx).Where("item_id = ?", fixture.item.ID).Order("image_type asc").Find(&images).Error; err != nil {
		t.Fatalf("load local catalog images: %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("expected local poster and backdrop, got %#v", images)
	}
	for _, image := range images {
		if !image.IsSelected || !strings.HasPrefix(image.URL, "/api/v1/items/") {
			t.Fatalf("expected selected local catalog artwork url, got %#v", image)
		}
	}

	itemID := strings.TrimSpace(strconv.FormatUint(uint64(fixture.item.ID), 10))
	posterBytes, err := os.ReadFile(filepath.Join(artworkRoot, "catalog", itemID, "poster.jpg"))
	if err != nil {
		t.Fatalf("read copied poster: %v", err)
	}
	if string(posterBytes) != "local-poster" {
		t.Fatalf("unexpected poster bytes: %q", string(posterBytes))
	}
	backdropBytes, err := os.ReadFile(filepath.Join(artworkRoot, "catalog", itemID, "backdrop.png"))
	if err != nil {
		t.Fatalf("read copied backdrop: %v", err)
	}
	if string(backdropBytes) != "local-backdrop" {
		t.Fatalf("unexpected backdrop bytes: %q", string(backdropBytes))
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
	if err := fixture.db.WithContext(ctx).Create(&database.ItemImage{
		ItemID:     fixture.item.ID,
		ImageType:  "backdrop",
		URL:        "https://image.tmdb.org/t/p/original/backdrop.jpg",
		IsSelected: true,
	}).Error; err != nil {
		t.Fatalf("seed remote backdrop: %v", err)
	}

	service := NewService(fixture.db, fixture.registry, fixture.cfg.FFprobe, fixture.cfg.FFmpeg)
	if err := service.ProbeInventoryFile(ctx, fixture.file.ID); err != nil {
		t.Fatalf("probe inventory file: %v", err)
	}

	var images []database.ItemImage
	if err := fixture.db.WithContext(ctx).Where("item_id = ?", fixture.item.ID).Order("id asc").Find(&images).Error; err != nil {
		t.Fatalf("load artwork rows: %v", err)
	}
	if len(images) != 2 || images[0].URL != "https://image.tmdb.org/t/p/original/poster.jpg" || images[1].URL != "https://image.tmdb.org/t/p/original/backdrop.jpg" {
		t.Fatalf("expected remote artwork to remain untouched, got %#v", images)
	}
	itemID := strings.TrimSpace(strconv.FormatUint(uint64(fixture.item.ID), 10))
	if _, err := os.Stat(filepath.Join(artworkRoot, "catalog", itemID, "poster.jpg")); !os.IsNotExist(err) {
		t.Fatalf("expected no generated poster file, got err=%v", err)
	}
}

func TestProbeInventoryFileDoesNotWriteStreamsAfterFileDeleted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := newInventoryProbeFixture(t)
	fixture.cfg.FFprobe.Path = writeSlowInventoryFakeFFprobe(t)
	service := NewService(fixture.db, fixture.registry, fixture.cfg.FFprobe)

	errCh := make(chan error, 1)
	go func() {
		errCh <- service.ProbeInventoryFile(ctx, fixture.file.ID)
	}()

	time.Sleep(50 * time.Millisecond)
	if err := fixture.db.WithContext(ctx).Unscoped().Delete(&database.InventoryFile{}, fixture.file.ID).Error; err != nil {
		t.Fatalf("delete inventory file during probe: %v", err)
	}

	if err := <-errCh; err == nil {
		t.Fatalf("expected probe to fail after inventory file deletion")
	}

	var streamCount int64
	if err := fixture.db.WithContext(ctx).Model(&database.MediaStream{}).Where("file_id = ?", fixture.file.ID).Count(&streamCount).Error; err != nil {
		t.Fatalf("count streams: %v", err)
	}
	if streamCount != 0 {
		t.Fatalf("expected no streams for deleted inventory file, got %d", streamCount)
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
	librarySvc := library.NewService(cfg, db, registry, nil)
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
	content := "#!/bin/sh\ncat <<'EOF'\n{\"streams\":[{\"codec_type\":\"video\",\"codec_name\":\"h264\",\"profile\":\"High\",\"level\":41,\"width\":1920,\"height\":1080,\"avg_frame_rate\":\"24000/1001\",\"r_frame_rate\":\"24000/1001\",\"field_order\":\"progressive\",\"bit_rate\":\"4200000\",\"color_space\":\"bt709\",\"bits_per_raw_sample\":\"10\",\"pix_fmt\":\"yuv420p10le\",\"refs\":4},{\"codec_type\":\"audio\",\"codec_name\":\"flac\",\"channels\":2,\"channel_layout\":\"stereo\",\"sample_rate\":\"48000\",\"bits_per_raw_sample\":\"24\",\"tags\":{\"language\":\"jpn\",\"title\":\"Stereo\"}},{\"codec_type\":\"subtitle\",\"codec_name\":\"ass\",\"disposition\":{\"default\":1,\"forced\":0,\"hearing_impaired\":0,\"external\":1},\"tags\":{\"language\":\"zho\",\"title\":\"Chinese Traditional\"}}],\"format\":{\"duration\":\"7260.25\",\"bit_rate\":\"5000000\"}}\nEOF\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake ffprobe: %v", err)
	}
	return path
}

func writeSlowInventoryFakeFFprobe(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ffprobe")
	content := "#!/bin/sh\nsleep 0.2\ncat <<'EOF'\n{\"streams\":[{\"codec_type\":\"video\",\"codec_name\":\"h264\"}],\"format\":{\"duration\":\"10\"}}\nEOF\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write slow fake ffprobe: %v", err)
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

func writeFailingInventoryFakeFFmpeg(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ffmpeg")
	content := "#!/bin/sh\nprintf 'ffmpeg should not run' >&2\nexit 1\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write failing fake ffmpeg: %v", err)
	}
	return path
}
