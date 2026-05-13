package library

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
)

func TestOpenListScanRecognizesMoviesAndTVDirectories(t *testing.T) {
	if os.Getenv("MIBO_OPENLIST_SCAN_TEST") != "1" {
		t.Skip("set MIBO_OPENLIST_SCAN_TEST=1 to scan the configured OpenList server")
	}

	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	cfg := config.Config{Worker: config.WorkerConfig{Enabled: true}}
	registry := providers.NewRegistry(cfg)
	workflowSvc := workflow.NewService(db)
	svc := NewService(cfg, db, registry, nil, ingest.NewService(db), workflowSvc)

	baseURL := integrationEnv("MIBO_OPENLIST_BASE_URL", "http://10.0.0.4:5244")
	username := integrationEnv("MIBO_OPENLIST_USERNAME", "admin")
	password := integrationEnv("MIBO_OPENLIST_PASSWORD", "admin123")
	moviesPath := integrationEnv("MIBO_OPENLIST_MOVIES_PATH", "/电影")
	tvPath := integrationEnv("MIBO_OPENLIST_TV_PATH", "/电视剧")

	source, err := svc.CreateMediaSource(ctx, CreateMediaSourceInput{
		Provider:   "openlist",
		Name:       "OpenList Integration",
		StorageRef: "/",
		RootPath:   "/",
		Config: &providers.SourceConfig{OpenList: &providers.OpenListSourceConfig{
			BaseURL:  baseURL,
			Username: username,
			Password: password,
			Timeout:  "15s",
		}},
	})
	if err != nil {
		t.Fatalf("create openlist media source: %v", err)
	}
	libraryRecord, _, err := svc.CreateLibrary(ctx, CreateLibraryInput{Name: "OpenList Auto", MediaSourceID: source.ID, RootPath: moviesPath})
	if err != nil {
		t.Fatalf("create movies library: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.LibraryPath{LibraryID: libraryRecord.ID, MediaSourceID: source.ID, RootPath: tvPath, DisplayName: "TV", Enabled: true}).Error; err != nil {
		t.Fatalf("create tv library path: %v", err)
	}

	if err := runOpenListScanAndRecognition(ctx, t, db, svc, libraryRecord.ID, moviesPath); err != nil {
		t.Fatalf("scan movies path %s: %v", moviesPath, err)
	}
	if err := runOpenListScanAndRecognition(ctx, t, db, svc, libraryRecord.ID, tvPath); err != nil {
		t.Fatalf("scan tv path %s: %v", tvPath, err)
	}
	if err := catalog.NewService(db, ingest.NewService(db)).RefreshLibraryProjectionScope(ctx, libraryRecord.ID); err != nil {
		t.Fatalf("refresh catalog projection: %v", err)
	}

	assertOpenListScanRecognizedType(ctx, t, db, libraryRecord.ID, moviesPath, database.MetadataItemTypeMovie, true)
	assertOpenListScanRecognizedType(ctx, t, db, libraryRecord.ID, tvPath, database.MetadataItemTypeSeries, false)
	assertOpenListScanRecognizedType(ctx, t, db, libraryRecord.ID, tvPath, database.MetadataItemTypeEpisode, true)
}

func runOpenListScanAndRecognition(ctx context.Context, t *testing.T, db *gorm.DB, svc *Service, libraryID uint, rootPath string) error {
	t.Helper()
	run, _, err := svc.QueueLibraryWorkflow(ctx, QueueWorkflowInput{LibraryID: libraryID, RootPath: rootPath, Reason: WorkflowReasonManualScan, Priority: 10})
	if err != nil {
		return err
	}
	var scanTask database.WorkflowTask
	if err := db.WithContext(ctx).Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeScanLibraryPath).First(&scanTask).Error; err != nil {
		return fmt.Errorf("load scan task: %w", err)
	}
	if err := svc.RunWorkflowScanLibraryPath(ctx, scanTask); err != nil {
		return err
	}
	var resolveTasks []database.WorkflowTask
	if err := db.WithContext(ctx).Where("run_id = ? AND task_type = ?", run.ID, workflow.TaskTypeResolveRecognition).Order("id asc").Find(&resolveTasks).Error; err != nil {
		return fmt.Errorf("load recognition resolve tasks: %w", err)
	}
	if len(resolveTasks) == 0 {
		return fmt.Errorf("no recognition resolve tasks were queued")
	}
	for _, task := range resolveTasks {
		if err := svc.RunWorkflowRecognitionResolve(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

func assertOpenListScanRecognizedType(ctx context.Context, t *testing.T, db *gorm.DB, libraryID uint, rootPath string, itemType string, requireAvailableProjection bool) {
	t.Helper()
	var fileCount int64
	if err := db.WithContext(ctx).Model(&database.InventoryFile{}).Where("library_id = ? AND storage_path LIKE ? AND content_class = ? AND deleted_at IS NULL", libraryID, strings.TrimRight(rootPath, "/")+"/%", SourceContentClassVideo).Count(&fileCount).Error; err != nil {
		t.Fatalf("count scanned inventory files under %s: %v", rootPath, err)
	}
	if fileCount == 0 {
		t.Fatalf("expected OpenList scan to discover video files under %s", rootPath)
	}
	var itemCount int64
	if err := db.WithContext(ctx).Model(&database.MetadataItem{}).Where("item_type = ? AND deleted_at IS NULL", itemType).Count(&itemCount).Error; err != nil {
		t.Fatalf("count recognized metadata %s items: %v", itemType, err)
	}
	if itemCount == 0 {
		t.Fatalf("expected OpenList scan to recognize at least one %s from %s; %s", itemType, rootPath, openListRecognitionDiagnostics(ctx, db, libraryID, rootPath))
	}
	if !requireAvailableProjection {
		return
	}
	var projectionCount int64
	if err := db.WithContext(ctx).Model(&database.LibraryMetadataProjection{}).Where("library_id = ? AND item_type = ? AND availability_status = ?", libraryID, itemType, database.ProjectionAvailabilityAvailable).Count(&projectionCount).Error; err != nil {
		t.Fatalf("count available %s projections: %v", itemType, err)
	}
	if projectionCount == 0 {
		t.Fatalf("expected OpenList scan to project at least one available %s from %s; %s", itemType, rootPath, openListRecognitionDiagnostics(ctx, db, libraryID, rootPath))
	}
}

func openListRecognitionDiagnostics(ctx context.Context, db *gorm.DB, libraryID uint, rootPath string) string {
	var itemCounts []struct {
		ItemType string
		Count    int64
	}
	_ = db.WithContext(ctx).Model(&database.MetadataItem{}).Select("item_type, count(*) as count").Group("item_type").Scan(&itemCounts).Error
	var projectionCounts []struct {
		ItemType           string
		AvailabilityStatus string
		Count              int64
	}
	_ = db.WithContext(ctx).Model(&database.LibraryMetadataProjection{}).Select("item_type, availability_status, count(*) as count").Where("library_id = ?", libraryID).Group("item_type, availability_status").Scan(&projectionCounts).Error
	var files []database.InventoryFile
	_ = db.WithContext(ctx).Where("library_id = ? AND storage_path LIKE ? AND content_class = ? AND deleted_at IS NULL", libraryID, strings.TrimRight(rootPath, "/")+"/%", SourceContentClassVideo).Order("storage_path asc").Limit(5).Find(&files).Error
	filePaths := make([]string, 0, len(files))
	for _, file := range files {
		filePaths = append(filePaths, file.StoragePath)
	}
	return fmt.Sprintf("metadata_counts=%#v projection_counts=%#v sample_files=%#v", itemCounts, projectionCounts, filePaths)
}

func integrationEnv(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
