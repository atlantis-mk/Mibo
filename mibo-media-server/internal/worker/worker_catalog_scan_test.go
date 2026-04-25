package worker

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/probe"
	"github.com/atlan/mibo-media-server/internal/providers"
	"gorm.io/gorm"
)

func TestRunOnceProcessesProbeInventoryFileJob(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := newInventoryProbeRunnerFixture(t)

	job, err := fixture.librarySvc.QueueInventoryFileProbe(ctx, fixture.file.ID, false)
	if err != nil {
		t.Fatalf("queue inventory probe job: %v", err)
	}
	if job.Kind != "probe_inventory_file" {
		t.Fatalf("expected job kind %q, got %q", "probe_inventory_file", job.Kind)
	}
	var payload struct {
		InventoryFileID uint `json:"inventory_file_id"`
	}
	if err := json.Unmarshal([]byte(job.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode job payload: %v", err)
	}
	if payload.InventoryFileID != fixture.file.ID {
		t.Fatalf("expected inventory_file_id=%d, got %d", fixture.file.ID, payload.InventoryFileID)
	}

	fixture.runner.RunOnce(ctx)

	assertJobCompleted(t, ctx, fixture.db, job.ID)

	var asset database.MediaAsset
	if err := fixture.db.WithContext(ctx).First(&asset, fixture.asset.ID).Error; err != nil {
		t.Fatalf("load asset: %v", err)
	}
	if asset.ProbeStatus != probe.StatusReady {
		t.Fatalf("expected asset probe status %q, got %q", probe.StatusReady, asset.ProbeStatus)
	}
	if asset.DurationSeconds == nil || *asset.DurationSeconds <= 0 {
		t.Fatalf("expected asset duration to be set, got %#v", asset.DurationSeconds)
	}

	var streamCount int64
	if err := fixture.db.WithContext(ctx).Model(&database.MediaStream{}).Where("file_id = ?", fixture.file.ID).Count(&streamCount).Error; err != nil {
		t.Fatalf("count streams: %v", err)
	}
	if streamCount != 3 {
		t.Fatalf("expected 3 media_stream rows, got %d", streamCount)
	}
}

type inventoryProbeRunnerFixture struct {
	db         *gorm.DB
	librarySvc *library.Service
	runner     *Runner
	asset      database.MediaAsset
	file       database.InventoryFile
}

func newInventoryProbeRunnerFixture(t *testing.T) inventoryProbeRunnerFixture {
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
		FFprobe: config.FFprobeConfig{Enabled: true, Path: writeFakeFFprobe(t), Timeout: time.Second},
		Worker:  config.WorkerConfig{Enabled: true, PollInterval: time.Millisecond},
	}
	registry := providers.NewRegistry(cfg)
	jobsSvc := jobs.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	probeSvc := probe.NewService(db, registry, cfg.FFprobe)
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
		ProbeStatus: probe.StatusPending,
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

	runner := NewRunner(cfg.Worker, jobsSvc, librarySvc, nil, probeSvc, nil)
	return inventoryProbeRunnerFixture{db: db, librarySvc: librarySvc, runner: runner, asset: asset, file: file}
}
