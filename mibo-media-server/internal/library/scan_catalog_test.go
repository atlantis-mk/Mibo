package library

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/storage"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
)

func TestRunSyncLibraryWritesCatalogRows(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	showsRoot := filepath.Join(rootPath, "shows")
	mustWriteFixtureFile(t, filepath.Join(moviesRoot, "Movie A (2024)", "Movie.A.2024.mkv"))
	mustWriteFixtureFile(t, filepath.Join(showsRoot, "Show One", "Season 1", "Show.One.S01E02.mkv"))

	movieLibrary := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	showLibrary := createDirectScanLibrary(t, ctx, svc, "Shows", "shows", showsRoot)

	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(movieLibrary.ID, movieLibrary.RootPath)); err != nil {
		t.Fatalf("run movie sync: %v", err)
	}
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(showLibrary.ID, showLibrary.RootPath)); err != nil {
		t.Fatalf("run show sync: %v", err)
	}

	assertTableCount(t, ctx, db, &database.CatalogItem{}, 4)
	assertTableCount(t, ctx, db, &database.InventoryFile{}, 2)
	assertTableCount(t, ctx, db, &database.MediaAsset{}, 2)
	assertTableCount(t, ctx, db, &database.AssetItem{}, 2)
	assertTableCount(t, ctx, db, &database.AssetFile{}, 2)

	var projectionJobs int64
	if err := db.WithContext(ctx).
		Model(&database.WorkflowTask{}).
		Where("task_type = ?", workflow.TaskTypeRefreshProjection).
		Count(&projectionJobs).Error; err != nil {
		t.Fatalf("count catalog projection refresh tasks: %v", err)
	}
	if projectionJobs != 2 {
		t.Fatalf("expected one catalog projection refresh per scan, got %d", projectionJobs)
	}

	var matchJobs []database.Job
	if err := db.WithContext(ctx).
		Where("kind = ?", JobKindMatchCatalogItem).
		Order("id asc").
		Find(&matchJobs).Error; err != nil {
		t.Fatalf("list catalog match jobs: %v", err)
	}
	if len(matchJobs) != 0 {
		t.Fatalf("expected scan to defer per-item catalog match jobs to batches, got %#v", matchJobs)
	}
	var batchJobs []database.WorkflowTask
	if err := db.WithContext(ctx).
		Where("task_type = ?", workflow.TaskTypeMatchMetadata).
		Order("id asc").
		Find(&batchJobs).Error; err != nil {
		t.Fatalf("list catalog match batch tasks: %v", err)
	}
	if len(batchJobs) != 2 {
		t.Fatalf("expected one catalog match batch per scan, got %#v", batchJobs)
	}
}

func TestScanPublishesDiscoveredInventoryBeforeCatalogMaterialization(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	moviePath := filepath.Join(rootPath, "Fast.Movie.2024.mkv")
	mustWriteFixtureFile(t, moviePath)
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", LibraryTypeMovies, rootPath)
	provider, err := svc.providerForLibraryPath(ctx, database.LibraryPath{MediaSourceID: libraryRecord.MediaSourceID, RootPath: rootPath})
	if err != nil {
		t.Fatalf("provider for library path: %v", err)
	}

	if _, err := svc.scanLibrary(ctx, provider, libraryRecord, rootPath); err != nil {
		t.Fatalf("scan library: %v", err)
	}
	var file database.InventoryFile
	if err := db.WithContext(ctx).Where("storage_path = ?", moviePath).First(&file).Error; err != nil {
		t.Fatalf("load inventory file: %v", err)
	}
	if file.ScanState != inventory.FileScanStateReviewRequired {
		t.Fatalf("expected scan to upgrade discovered file to review-required state, got %#v", file)
	}

	if err := db.WithContext(ctx).Where("file_id = ?", file.ID).Delete(&database.AssetFile{}).Error; err != nil {
		t.Fatalf("simulate pre-materialization unlink: %v", err)
	}
	if err := db.WithContext(ctx).Model(&database.InventoryFile{}).Where("id = ?", file.ID).Update("scan_state", inventory.FileScanStateDiscovered).Error; err != nil {
		t.Fatalf("simulate discovered state: %v", err)
	}
	result, err := catalog.NewService(db).BrowseItems(ctx, catalog.BrowseItemsInput{LibraryID: libraryRecord.ID, TypeFilter: "all", Limit: 20})
	if err != nil {
		t.Fatalf("browse items: %v", err)
	}
	found := false
	for _, item := range result.Items {
		if item.InventoryFileID != nil && *item.InventoryFileID == file.ID && item.SourceKind == "inventory_file" && item.MaturityState == inventory.FileScanStateDiscovered {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected discovered inventory-backed browse entry, got %#v", result.Items)
	}
}

func TestBulkDiscoveredInventoryPersistsProviderThumbnail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	provider := &countingMaterializeProvider{}
	libraryRecord := database.Library{ID: 1, Name: "Movies", Type: LibraryTypeMovies, RootPath: "/library", Status: "active", ScannerEnabled: true}
	candidates := []discoveredInventoryCandidate{{
		object: storage.Object{
			Name:         "Movie.mkv",
			Path:         "/library/Movie.mkv",
			Size:         100,
			ThumbnailURL: "https://cdn.example.test/movie-thumb.jpg",
		},
		container: "mkv",
	}}

	files, err := svc.bulkUpsertDiscoveredInventoryFiles(ctx, provider, libraryRecord, candidates)
	if err != nil {
		t.Fatalf("bulk upsert discovered inventory: %v", err)
	}
	file := files["local\x00/library/Movie.mkv"]
	if file.ThumbnailURL != "https://cdn.example.test/movie-thumb.jpg" {
		t.Fatalf("expected provider thumbnail to persist, got %#v", file)
	}
}

func TestRunCatalogMaterializeBatchReusesDirectorySnapshotWithinBatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	provider := &countingMaterializeProvider{objectsByPath: map[string][]storage.Object{}}
	registry := providers.NewRegistry(config.Config{})
	registryTestSwapLocal(registry, provider)
	svc := NewService(config.Config{}, db, registry, nil)

	rootPath := "/library"
	provider.objectsByPath[rootPath] = []storage.Object{{Name: filepath.Base(rootPath), Path: rootPath, IsDir: true}}
	videoPaths := []string{
		filepath.ToSlash(filepath.Join(rootPath, "folder", "one.mkv")),
		filepath.ToSlash(filepath.Join(rootPath, "folder", "two.mkv")),
		filepath.ToSlash(filepath.Join(rootPath, "folder", "three.mkv")),
	}
	dirPath := filepath.ToSlash(filepath.Join(rootPath, "folder"))
	provider.objectsByPath[dirPath] = []storage.Object{
		{Name: "one.mkv", Path: videoPaths[0], Size: 10, Provider: provider.Name()},
		{Name: "two.mkv", Path: videoPaths[1], Size: 11, Provider: provider.Name()},
		{Name: "three.mkv", Path: videoPaths[2], Size: 12, Provider: provider.Name()},
	}

	source := database.MediaSource{Name: "Test", Provider: "local", RootPath: rootPath}
	if err := db.WithContext(ctx).Create(&source).Error; err != nil {
		t.Fatalf("create media source: %v", err)
	}
	libraryRecord := database.Library{Name: "Movies", Type: LibraryTypeMovies, MediaSourceID: source.ID, RootPath: rootPath, Status: "active", ScannerEnabled: true}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.LibraryPath{LibraryID: libraryRecord.ID, MediaSourceID: source.ID, RootPath: rootPath, DisplayName: libraryRecord.Name, Enabled: true}).Error; err != nil {
		t.Fatalf("create library path: %v", err)
	}
	if err := database.EnsureLibraryPolicyDefaults(db.WithContext(ctx), libraryRecord.ID); err != nil {
		t.Fatalf("ensure library policy defaults: %v", err)
	}

	fileIDs := make([]uint, 0, len(videoPaths))
	for _, videoPath := range videoPaths {
		file, err := inventory.NewService(db).UpsertFile(ctx, inventory.UpsertFileInput{
			LibraryID:       libraryRecord.ID,
			StorageProvider: provider.Name(),
			StoragePath:     videoPath,
			Container:       "mkv",
			ContentClass:    SourceContentClassVideo,
			Status:          inventory.FileStatusAvailable,
			ScanState:       inventory.FileScanStateDiscovered,
		})
		if err != nil {
			t.Fatalf("upsert inventory file %s: %v", videoPath, err)
		}
		fileIDs = append(fileIDs, file.ID)
	}

	if err := svc.RunCatalogMaterializeBatch(ctx, CatalogMaterializeBatchPayload{LibraryID: libraryRecord.ID, RootPath: rootPath, FileIDs: fileIDs}); err != nil {
		t.Fatalf("run catalog materialize batch: %v", err)
	}
	if got := provider.listCount(dirPath); got != 1 {
		t.Fatalf("expected one directory list for %s, got %d", dirPath, got)
	}
	if got := provider.refreshFlags(dirPath); len(got) != 1 || got[0] {
		t.Fatalf("expected materialize relist for %s to avoid refresh, got %#v", dirPath, got)
	}
	if got := provider.listCount(rootPath); got != 0 {
		t.Fatalf("expected no materialize relist for root %s, got %d", rootPath, got)
	}
}

func TestListAllDirectoryObjectsRefreshesOnlyFirstPage(t *testing.T) {
	t.Parallel()

	provider := &countingMaterializeProvider{objectsByPath: map[string][]storage.Object{}}
	svc := NewService(config.Config{}, nil, nil, nil)
	dirPath := "/library"
	provider.objectsByPath[dirPath] = make([]storage.Object, 1001)
	for i := range provider.objectsByPath[dirPath] {
		provider.objectsByPath[dirPath][i] = storage.Object{Name: fmt.Sprintf("file-%04d.mkv", i), Path: fmt.Sprintf("%s/file-%04d.mkv", dirPath, i), Provider: provider.Name()}
	}

	objects, err := svc.listAllDirectoryObjects(context.Background(), provider, dirPath, true)
	if err != nil {
		t.Fatalf("list all directory objects: %v", err)
	}
	if len(objects) != 1001 {
		t.Fatalf("expected 1001 objects, got %d", len(objects))
	}
	if got := provider.refreshFlags(dirPath); len(got) != 2 || !got[0] || got[1] {
		t.Fatalf("expected first page refresh only for %s, got %#v", dirPath, got)
	}
}

func TestScanUpgradesDiscoveredFileIntoEpisodeHierarchyWithFileAnchor(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	episodePath := filepath.Join(rootPath, "Show One", "Season 1", "Show.One.S01E02.mkv")
	mustWriteFixtureFile(t, episodePath)
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Shows", LibraryTypeShows, rootPath)
	provider, err := svc.providerForLibraryPath(ctx, database.LibraryPath{MediaSourceID: libraryRecord.MediaSourceID, RootPath: rootPath})
	if err != nil {
		t.Fatalf("provider for library path: %v", err)
	}

	if _, err := svc.scanLibrary(ctx, provider, libraryRecord, rootPath); err != nil {
		t.Fatalf("scan library: %v", err)
	}
	var file database.InventoryFile
	if err := db.WithContext(ctx).Where("storage_path = ?", episodePath).First(&file).Error; err != nil {
		t.Fatalf("load inventory file: %v", err)
	}
	if file.ScanState != inventory.FileScanStateClassified {
		t.Fatalf("expected classified file state, got %#v", file)
	}
	var episode database.CatalogItem
	if err := db.WithContext(ctx).Where("library_id = ? AND type = ?", libraryRecord.ID, catalog.ItemTypeEpisode).First(&episode).Error; err != nil {
		t.Fatalf("load episode: %v", err)
	}
	var count int64
	if err := db.WithContext(ctx).
		Table("asset_files").
		Joins("JOIN asset_items ON asset_items.asset_id = asset_files.asset_id").
		Where("asset_files.file_id = ? AND asset_items.item_id = ?", file.ID, episode.ID).
		Count(&count).Error; err != nil {
		t.Fatalf("count file anchor links: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected episode graph to preserve inventory file anchor, got %d links", count)
	}
}

type countingMaterializeProvider struct {
	mu            sync.Mutex
	objectsByPath map[string][]storage.Object
	listCalls     map[string]int
	refreshByPath map[string][]bool
}

func (p *countingMaterializeProvider) Name() string { return "local" }

func (p *countingMaterializeProvider) List(_ context.Context, req storage.ListRequest) ([]storage.Object, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.listCalls == nil {
		p.listCalls = make(map[string]int)
	}
	if p.refreshByPath == nil {
		p.refreshByPath = make(map[string][]bool)
	}
	p.listCalls[req.Path]++
	p.refreshByPath[req.Path] = append(p.refreshByPath[req.Path], req.Refresh)
	objects := p.objectsByPath[req.Path]
	cloned := make([]storage.Object, len(objects))
	copy(cloned, objects)
	page := req.Page
	if page <= 0 {
		page = 1
	}
	perPage := req.PerPage
	if perPage <= 0 {
		return cloned, nil
	}
	start := (page - 1) * perPage
	if start >= len(cloned) {
		return []storage.Object{}, nil
	}
	end := start + perPage
	if end > len(cloned) {
		end = len(cloned)
	}
	return cloned[start:end], nil
}

func (p *countingMaterializeProvider) Get(_ context.Context, req storage.GetRequest) (storage.Object, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, object := range p.objectsByPath[req.Path] {
		if object.Path == req.Path {
			return object, nil
		}
	}
	if objects := p.objectsByPath[req.Path]; len(objects) > 0 {
		return objects[0], nil
	}
	return storage.Object{Name: filepath.Base(req.Path), Path: req.Path, IsDir: true, Provider: p.Name()}, nil
}

func (p *countingMaterializeProvider) Link(context.Context, storage.LinkRequest) (storage.LinkResult, error) {
	return storage.LinkResult{}, storage.ErrNotImplemented
}

func (p *countingMaterializeProvider) ResolveStorage(ctx context.Context, req storage.ResolveStorageRequest) (storage.ResolvedStorage, error) {
	object, err := p.Get(ctx, storage.GetRequest{Path: req.Path})
	if err != nil {
		return storage.ResolvedStorage{}, err
	}
	if object.Path == "" {
		object.Path = req.Path
	}
	if object.Name == "" {
		object.Name = filepath.Base(req.Path)
	}
	object.IsDir = true
	object.Provider = p.Name()
	return storage.ResolvedStorage{Provider: p.Name(), Path: req.Path, Object: object, Caps: storage.Capabilities{CanList: true, CanGet: true}}, nil
}

func (p *countingMaterializeProvider) Capabilities(context.Context) (storage.Capabilities, error) {
	return storage.Capabilities{CanList: true, CanGet: true}, nil
}

func (p *countingMaterializeProvider) listCount(path string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.listCalls[path]
}

func (p *countingMaterializeProvider) refreshFlags(path string) []bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	flags := p.refreshByPath[path]
	cloned := make([]bool, len(flags))
	copy(cloned, flags)
	return cloned
}

func registryTestSwapLocal(registry *providers.Registry, provider storage.Provider) {
	if registry == nil {
		return
	}
	registryValue := reflect.ValueOf(registry).Elem()
	localField := registryValue.FieldByName("local")
	reflect.NewAt(localField.Type(), unsafe.Pointer(localField.UnsafeAddr())).Elem().Set(reflect.ValueOf(provider))
}

func TestScanGroupsDiscoveredMovieVersionsByFileAnchors(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	movieDir := filepath.Join(rootPath, "Inception")
	firstPath := filepath.Join(movieDir, "Inception.1080p.mkv")
	secondPath := filepath.Join(movieDir, "Inception.2160p.mkv")
	mustWriteFixtureFile(t, firstPath)
	mustWriteFixtureFile(t, secondPath)
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", LibraryTypeMovies, rootPath)
	provider, err := svc.providerForLibraryPath(ctx, database.LibraryPath{MediaSourceID: libraryRecord.MediaSourceID, RootPath: rootPath})
	if err != nil {
		t.Fatalf("provider for library path: %v", err)
	}

	if _, err := svc.scanLibrary(ctx, provider, libraryRecord, rootPath); err != nil {
		t.Fatalf("scan library: %v", err)
	}
	var movieCount int64
	if err := db.WithContext(ctx).Model(&database.CatalogItem{}).Where("library_id = ? AND type = ?", libraryRecord.ID, catalog.ItemTypeMovie).Count(&movieCount).Error; err != nil {
		t.Fatalf("count movies: %v", err)
	}
	if movieCount != 1 {
		t.Fatalf("expected one movie for discovered versions, got %d", movieCount)
	}
	var files []database.InventoryFile
	if err := db.WithContext(ctx).Where("storage_path IN ?", []string{firstPath, secondPath}).Order("storage_path asc").Find(&files).Error; err != nil {
		t.Fatalf("load files: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected two inventory files, got %#v", files)
	}
	for _, file := range files {
		if file.ScanState != inventory.FileScanStateReviewRequired && file.ScanState != inventory.FileScanStateClassified {
			t.Fatalf("unexpected file scan state: %#v", file)
		}
		var count int64
		if err := db.WithContext(ctx).Model(&database.AssetFile{}).Where("file_id = ?", file.ID).Count(&count).Error; err != nil {
			t.Fatalf("count asset file links: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected file %d to keep one asset anchor link, got %d", file.ID, count)
		}
	}
}

func TestRunSyncLibraryPersistsHashtagTags(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	mustWriteFixtureFile(t, filepath.Join(moviesRoot, "Movie.Name.#IMAX.#国语.2024.mkv"))

	movieLibrary := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(movieLibrary.ID, movieLibrary.RootPath)); err != nil {
		t.Fatalf("run movie sync: %v", err)
	}

	var movie database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeMovie).First(&movie).Error; err != nil {
		t.Fatalf("load movie: %v", err)
	}
	if movie.Title != "Movie Name" {
		t.Fatalf("expected hashtags removed from title, got %q", movie.Title)
	}

	var rows []struct {
		Kind string
		Name string
	}
	if err := db.WithContext(ctx).
		Table("item_tags").
		Select("tags.kind, tags.name").
		Joins("JOIN tags ON tags.id = item_tags.tag_id").
		Where("item_tags.item_id = ?", movie.ID).
		Order("tags.name asc").
		Scan(&rows).Error; err != nil {
		t.Fatalf("load hashtag tags: %v", err)
	}
	if len(rows) != 2 || rows[0].Kind != "hashtag" || rows[0].Name != "IMAX" || rows[1].Kind != "hashtag" || rows[1].Name != "国语" {
		t.Fatalf("unexpected hashtag tags: %#v", rows)
	}

	var source database.MetadataSource
	if err := db.WithContext(ctx).Where("item_id = ? AND source_name = ?", movie.ID, "scanner").First(&source).Error; err != nil {
		t.Fatalf("load scanner source: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(source.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode scanner payload: %v", err)
	}
	if _, ok := payload["hashtag_tags"].([]any); !ok {
		t.Fatalf("expected hashtag_tags evidence in %#v", payload)
	}
}

func TestRunSyncLibraryGroupsFlatTVFolderIntoSingleSeries(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	showsRoot := filepath.Join(rootPath, "shows")
	showDir := filepath.Join(showsRoot, "灵笼第二季")
	mustWriteFixtureFile(t, filepath.Join(showDir, "灵笼 第二季.S02E01.mp4"))
	mustWriteFixtureFile(t, filepath.Join(showDir, "Incarnation.S02E02.mp4"))
	mustWriteFixtureFile(t, filepath.Join(showDir, "第03集.mp4"))

	showLibrary := createDirectScanLibrary(t, ctx, svc, "Shows", "shows", showsRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(showLibrary.ID, showLibrary.RootPath)); err != nil {
		t.Fatalf("run show sync: %v", err)
	}

	var series []database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeSeries).Order("id asc").Find(&series).Error; err != nil {
		t.Fatalf("load series: %v", err)
	}
	if len(series) != 1 {
		t.Fatalf("expected one series for flat TV folder, got %#v", series)
	}

	var episodes []database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeEpisode).Order("index_number asc").Find(&episodes).Error; err != nil {
		t.Fatalf("load episodes: %v", err)
	}
	if len(episodes) != 3 {
		t.Fatalf("expected three episodes, got %#v", episodes)
	}
	for idx, episode := range episodes {
		expectedEpisode := idx + 1
		if episode.RootID == nil || *episode.RootID != series[0].ID || episode.ParentIndexNumber == nil || *episode.ParentIndexNumber != 2 || episode.IndexNumber == nil || *episode.IndexNumber != expectedEpisode {
			t.Fatalf("unexpected episode hierarchy at %d: %#v series=%#v", idx, episode, series[0])
		}
	}

	var assets int64
	if err := db.WithContext(ctx).Model(&database.MediaAsset{}).Count(&assets).Error; err != nil {
		t.Fatalf("count assets: %v", err)
	}
	if assets != 3 {
		t.Fatalf("expected one asset per flat folder episode, got %d", assets)
	}
}

func TestRunSyncLibraryInheritsSeriesDirectoryForNestedVideos(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	showsRoot := filepath.Join(rootPath, "shows")
	showDir := filepath.Join(showsRoot, "灵笼")
	mustWriteFixtureFile(t, filepath.Join(showDir, "Season 1", "01.mp4"))
	mustWriteFixtureFile(t, filepath.Join(showDir, "OVA", "alpha.mp4"))
	mustWriteFixtureFile(t, filepath.Join(showDir, "OVA", "beta.mp4"))

	showLibrary := createDirectScanLibrary(t, ctx, svc, "Shows", "shows", showsRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(showLibrary.ID, showLibrary.RootPath)); err != nil {
		t.Fatalf("run show sync: %v", err)
	}

	var series []database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeSeries).Find(&series).Error; err != nil {
		t.Fatalf("load series: %v", err)
	}
	if len(series) != 1 || series[0].Title != "灵笼" {
		t.Fatalf("expected nested videos to stay under one inherited series, got %#v", series)
	}

	var episodes []database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeEpisode).Order("index_number asc").Find(&episodes).Error; err != nil {
		t.Fatalf("load episodes: %v", err)
	}
	if len(episodes) != 2 {
		t.Fatalf("expected two inherited episodes, got %#v", episodes)
	}
	for idx, episode := range episodes {
		expectedEpisode := idx + 1
		if episode.RootID == nil || *episode.RootID != series[0].ID || episode.ParentIndexNumber == nil || *episode.ParentIndexNumber != 1 || episode.IndexNumber == nil || *episode.IndexNumber != expectedEpisode {
			t.Fatalf("unexpected inherited episode at %d: %#v series=%#v", idx, episode, series[0])
		}
	}
}

func TestRunSyncLibraryGroupsNoisySeasonDirectoriesUnderSeries(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	showsRoot := filepath.Join(rootPath, "shows")
	showDir := filepath.Join(showsRoot, "魔幻手机 (2008)")
	mustWriteFixtureFile(t, filepath.Join(showDir, "第 1 季 - 1080p H265 10bit AAC", "魔幻手机 S01E01 - 第 1 集.mp4"))
	mustWriteFixtureFile(t, filepath.Join(showDir, "第 2 季 - 2160p WEB-DL HEVC DDP 2Audios", "魔幻手机2：傻妞归来 S02E01 - 第 1 集 - 2160p WEB-DL HEVC DDP 2Audios.mp4"))

	showLibrary := createDirectScanLibrary(t, ctx, svc, "Shows", "shows", showsRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(showLibrary.ID, showLibrary.RootPath)); err != nil {
		t.Fatalf("run show sync: %v", err)
	}

	var series []database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeSeries).Find(&series).Error; err != nil {
		t.Fatalf("load series: %v", err)
	}
	if len(series) != 1 || series[0].Title != "魔幻手机" {
		t.Fatalf("expected one normalized series, got %#v", series)
	}

	var seasons []database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeSeason).Order("index_number asc").Find(&seasons).Error; err != nil {
		t.Fatalf("load seasons: %v", err)
	}
	if len(seasons) != 2 {
		t.Fatalf("expected two seasons, got %#v", seasons)
	}
	for idx, season := range seasons {
		expected := idx + 1
		if season.ParentID == nil || *season.ParentID != series[0].ID || season.IndexNumber == nil || *season.IndexNumber != expected {
			t.Fatalf("unexpected season hierarchy at %d: %#v series=%#v", idx, season, series[0])
		}
	}

	var episodes []database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeEpisode).Order("parent_index_number asc, index_number asc").Find(&episodes).Error; err != nil {
		t.Fatalf("load episodes: %v", err)
	}
	if len(episodes) != 2 {
		t.Fatalf("expected two episodes, got %#v", episodes)
	}
	for idx, episode := range episodes {
		expectedSeason := idx + 1
		if episode.RootID == nil || *episode.RootID != series[0].ID || episode.ParentIndexNumber == nil || *episode.ParentIndexNumber != expectedSeason || episode.IndexNumber == nil || *episode.IndexNumber != 1 {
			t.Fatalf("unexpected episode hierarchy at %d: %#v series=%#v", idx, episode, series[0])
		}
	}
}

func TestRunSyncLibraryInfersUnnamedTVEpisodesAfterSkippingExtras(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	showsRoot := filepath.Join(rootPath, "shows")
	showDir := filepath.Join(showsRoot, "Unsorted Show")
	mustWriteFixtureFile(t, filepath.Join(showDir, "trailer.mp4"))
	mustWriteFixtureFile(t, filepath.Join(showDir, "behind-the-scenes.mkv"))
	mustWriteFixtureFile(t, filepath.Join(showDir, "alpha.mkv"))
	mustWriteFixtureFile(t, filepath.Join(showDir, "episode.mkv"))

	showLibrary := createDirectScanLibrary(t, ctx, svc, "Shows", "shows", showsRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(showLibrary.ID, showLibrary.RootPath)); err != nil {
		t.Fatalf("run show sync: %v", err)
	}

	var series []database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeSeries).Find(&series).Error; err != nil {
		t.Fatalf("load series: %v", err)
	}
	if len(series) != 1 {
		t.Fatalf("expected one series, got %#v", series)
	}

	var episodes []database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeEpisode).Order("index_number asc").Find(&episodes).Error; err != nil {
		t.Fatalf("load episodes: %v", err)
	}
	if len(episodes) != 2 {
		t.Fatalf("expected only non-extra files as episodes, got %#v", episodes)
	}
	for idx, episode := range episodes {
		expected := idx + 1
		if episode.IndexNumber == nil || *episode.IndexNumber != expected || episode.ParentIndexNumber == nil || *episode.ParentIndexNumber != 1 {
			t.Fatalf("unexpected fallback episode %d: %#v", idx, episode)
		}
	}

	var movies int64
	if err := db.WithContext(ctx).Model(&database.CatalogItem{}).Where("type = ?", catalog.ItemTypeMovie).Count(&movies).Error; err != nil {
		t.Fatalf("count movie pollution: %v", err)
	}
	if movies != 0 {
		t.Fatalf("expected skipped TV extras not to create movie items, got %d", movies)
	}
}

func TestRunCatalogMaterializeBatchSkipsTVExtrasWithoutFailing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	provider := &countingMaterializeProvider{objectsByPath: map[string][]storage.Object{}}
	registry := providers.NewRegistry(config.Config{})
	registryTestSwapLocal(registry, provider)
	svc := NewService(config.Config{}, db, registry, nil)

	rootPath := "/shows"
	showPath := "/shows/Unsorted Show"
	trailerPath := "/shows/Unsorted Show/trailer.mp4"
	provider.objectsByPath[showPath] = []storage.Object{
		{Name: "alpha.mkv", Path: "/shows/Unsorted Show/alpha.mkv", Size: 100, Provider: provider.Name()},
		{Name: "episode.mkv", Path: "/shows/Unsorted Show/episode.mkv", Size: 100, Provider: provider.Name()},
		{Name: "trailer.mp4", Path: trailerPath, Size: 100, Provider: provider.Name()},
	}

	source := database.MediaSource{Name: "Test", Provider: "local", RootPath: rootPath}
	if err := db.WithContext(ctx).Create(&source).Error; err != nil {
		t.Fatalf("create media source: %v", err)
	}
	libraryRecord := database.Library{Name: "Shows", Type: LibraryTypeShows, MediaSourceID: source.ID, RootPath: rootPath, Status: "active", ScannerEnabled: true}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.LibraryPath{LibraryID: libraryRecord.ID, MediaSourceID: source.ID, RootPath: rootPath, DisplayName: libraryRecord.Name, Enabled: true}).Error; err != nil {
		t.Fatalf("create library path: %v", err)
	}
	if err := database.EnsureLibraryPolicyDefaults(db.WithContext(ctx), libraryRecord.ID); err != nil {
		t.Fatalf("ensure library policy defaults: %v", err)
	}

	file, err := inventory.NewService(db).UpsertFile(ctx, inventory.UpsertFileInput{
		LibraryID:       libraryRecord.ID,
		StorageProvider: provider.Name(),
		StoragePath:     trailerPath,
		Container:       "mp4",
		ContentClass:    SourceContentClassVideo,
		Status:          inventory.FileStatusAvailable,
		ScanState:       inventory.FileScanStateDiscovered,
	})
	if err != nil {
		t.Fatalf("upsert trailer file: %v", err)
	}

	if err := svc.RunCatalogMaterializeBatch(ctx, CatalogMaterializeBatchPayload{LibraryID: libraryRecord.ID, RootPath: rootPath, FileIDs: []uint{file.ID}}); err != nil {
		t.Fatalf("run catalog materialize batch: %v", err)
	}

	var movies int64
	if err := db.WithContext(ctx).Model(&database.CatalogItem{}).Where("type = ?", catalog.ItemTypeMovie).Count(&movies).Error; err != nil {
		t.Fatalf("count movie pollution: %v", err)
	}
	if movies != 0 {
		t.Fatalf("expected skipped TV extra not to create movie items, got %d", movies)
	}
}

func TestRunSyncLibraryMaterializesNumericEpisodeWithQualityToken(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	packRoot := filepath.Join(rootPath, "My Pack")
	showDir := filepath.Join(packRoot, "电视剧", "我的山与海.2160p")
	for episode := 1; episode <= 30; episode++ {
		mustWriteFixtureFile(t, filepath.Join(showDir, fmt.Sprintf("%02d.2160p.HD国语中字无水印[最新电影www.5266ys.com].mkv", episode)))
	}

	libraryRecord := createDirectScanLibrary(t, ctx, svc, "My Pack", LibraryTypeAuto, packRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run sync: %v", err)
	}

	var file database.InventoryFile
	path25 := filepath.ToSlash(filepath.Join(packRoot, "电视剧", "我的山与海.2160p", "25.2160p.HD国语中字无水印[最新电影www.5266ys.com].mkv"))
	if err := db.WithContext(ctx).Where("storage_path = ?", path25).First(&file).Error; err != nil {
		t.Fatalf("load episode 25 inventory file: %v", err)
	}
	if file.ScanState != inventory.FileScanStateReviewRequired {
		t.Fatalf("expected episode 25 to be classified/reviewable, got %#v", file)
	}

	var linkCount int64
	if err := db.WithContext(ctx).Model(&database.AssetFile{}).Where("file_id = ?", file.ID).Count(&linkCount).Error; err != nil {
		t.Fatalf("count episode 25 asset links: %v", err)
	}
	if linkCount != 1 {
		t.Fatalf("expected episode 25 asset link, got %d", linkCount)
	}

	var episodeItem database.CatalogItem
	if err := db.WithContext(ctx).
		Joins("JOIN asset_items ON asset_items.item_id = catalog_items.id").
		Joins("JOIN asset_files ON asset_files.asset_id = asset_items.asset_id").
		Where("asset_files.file_id = ? AND catalog_items.type = ?", file.ID, catalog.ItemTypeEpisode).
		First(&episodeItem).Error; err != nil {
		t.Fatalf("load episode 25 catalog item: %v", err)
	}
	if episodeItem.ParentIndexNumber == nil || *episodeItem.ParentIndexNumber != 1 || episodeItem.IndexNumber == nil || *episodeItem.IndexNumber != 25 {
		t.Fatalf("expected S01E25 catalog item, got %#v", episodeItem)
	}
}

func TestRunWorkflowCatalogMaterializeDistinguishesMaterializePayload(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	packRoot := filepath.Join(rootPath, "My Pack")
	showDir := filepath.Join(packRoot, "电视剧", "我的山与海.2160p")
	for episode := 1; episode <= 30; episode++ {
		mustWriteFixtureFile(t, filepath.Join(showDir, fmt.Sprintf("%02d.2160p.HD国语中字无水印[最新电影www.5266ys.com].mkv", episode)))
	}

	libraryRecord := createDirectScanLibrary(t, ctx, svc, "My Pack", LibraryTypeAuto, packRoot)
	path25 := filepath.ToSlash(filepath.Join(packRoot, "电视剧", "我的山与海.2160p", "25.2160p.HD国语中字无水印[最新电影www.5266ys.com].mkv"))
	file, err := inventory.NewService(db).UpsertFile(ctx, inventory.UpsertFileInput{LibraryID: libraryRecord.ID, StorageProvider: "local", StoragePath: path25, Container: "mkv", ContentClass: SourceContentClassVideo, Status: inventory.FileStatusAvailable, ScanState: inventory.FileScanStateDiscovered})
	if err != nil {
		t.Fatalf("upsert inventory file: %v", err)
	}
	payloadJSON, err := json.Marshal(CatalogMaterializeBatchPayload{LibraryID: libraryRecord.ID, RootPath: libraryRecord.RootPath, FileIDs: []uint{file.ID}})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}

	if err := svc.RunWorkflowCatalogMaterialize(ctx, database.WorkflowTask{LibraryID: libraryRecord.ID, PayloadJSON: string(payloadJSON)}); err != nil {
		t.Fatalf("run workflow materialize: %v", err)
	}

	var linkCount int64
	if err := db.WithContext(ctx).Model(&database.AssetFile{}).Where("file_id = ?", file.ID).Count(&linkCount).Error; err != nil {
		t.Fatalf("count asset links: %v", err)
	}
	if linkCount != 1 {
		t.Fatalf("expected workflow materialize to create asset link, got %d", linkCount)
	}
}

func TestRunSyncLibraryGroupsMovieFolderVersionsIntoSingleMovie(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	movieDir := filepath.Join(moviesRoot, "Inception (2010)")
	mustWriteFixtureFile(t, filepath.Join(movieDir, "Inception.1080p.mkv"))
	mustWriteFixtureFile(t, filepath.Join(movieDir, "Inception.2160p.mkv"))

	movieLibrary := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(movieLibrary.ID, movieLibrary.RootPath)); err != nil {
		t.Fatalf("run movie sync: %v", err)
	}

	var movies []database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeMovie).Find(&movies).Error; err != nil {
		t.Fatalf("load movies: %v", err)
	}
	if len(movies) != 1 {
		t.Fatalf("expected one movie for folder versions, got %#v", movies)
	}
	if movies[0].Path != movieDir {
		t.Fatalf("expected movie path to use folder %q, got %q", movieDir, movies[0].Path)
	}

	var assets []database.MediaAsset
	if err := db.WithContext(ctx).Where("library_id = ?", movieLibrary.ID).Order("asset_type asc, id asc").Find(&assets).Error; err != nil {
		t.Fatalf("load assets: %v", err)
	}
	if len(assets) != 2 {
		t.Fatalf("expected two assets for movie versions, got %#v", assets)
	}
	roles := map[string]bool{}
	for _, asset := range assets {
		var link database.AssetItem
		if err := db.WithContext(ctx).Where("asset_id = ? AND item_id = ?", asset.ID, movies[0].ID).First(&link).Error; err != nil {
			t.Fatalf("load asset link: %v", err)
		}
		roles[link.Role] = true
	}
	if !roles[inventory.AssetItemRolePrimary] || !roles[inventory.AssetItemRoleVersion] {
		t.Fatalf("expected primary and version roles for movie folder versions, got %#v", roles)
	}
}

func TestRunSyncLibraryAssociatesMovieExtrasWithMovieFolder(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	movieDir := filepath.Join(moviesRoot, "Inception (2010)")
	mustWriteFixtureFile(t, filepath.Join(movieDir, "Inception.mkv"))
	mustWriteFixtureFile(t, filepath.Join(movieDir, "trailer.mp4"))
	mustWriteFixtureFile(t, filepath.Join(movieDir, "behind-the-scenes.mkv"))

	movieLibrary := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(movieLibrary.ID, movieLibrary.RootPath)); err != nil {
		t.Fatalf("run movie sync: %v", err)
	}

	var movies []database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeMovie).Find(&movies).Error; err != nil {
		t.Fatalf("load movies: %v", err)
	}
	if len(movies) != 1 {
		t.Fatalf("expected one movie for main and extras, got %#v", movies)
	}

	var assets []database.MediaAsset
	if err := db.WithContext(ctx).Where("library_id = ?", movieLibrary.ID).Find(&assets).Error; err != nil {
		t.Fatalf("load assets: %v", err)
	}
	if len(assets) != 3 {
		t.Fatalf("expected main plus two extra assets, got %#v", assets)
	}
	types := map[string]bool{}
	roles := map[string]bool{}
	for _, asset := range assets {
		types[asset.AssetType] = true
		var link database.AssetItem
		if err := db.WithContext(ctx).Where("asset_id = ? AND item_id = ?", asset.ID, movies[0].ID).First(&link).Error; err != nil {
			t.Fatalf("load asset link: %v", err)
		}
		roles[link.Role] = true
	}
	if !types[inventory.AssetTypeMain] || !types[inventory.AssetTypeTrailer] || !types[inventory.AssetTypeExtra] {
		t.Fatalf("expected main, trailer, and extra asset types, got %#v", types)
	}
	if !roles[inventory.AssetItemRolePrimary] || !roles[inventory.AssetItemRoleTrailer] || !roles[inventory.AssetItemRoleExtra] {
		t.Fatalf("expected primary, trailer, and extra roles, got %#v", roles)
	}
}

func TestRunSyncLibraryMixedClassifiesSingleNonExtraVideoAsMovie(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	mixedRoot := filepath.Join(rootPath, "mixed")
	movieDir := filepath.Join(mixedRoot, "Inception (2010)")
	mustWriteFixtureFile(t, filepath.Join(movieDir, "Inception.mkv"))
	mustWriteFixtureFile(t, filepath.Join(movieDir, "trailer.mp4"))
	mustWriteFixtureFile(t, filepath.Join(movieDir, "behind-the-scenes.mkv"))
	mustWriteFixtureFile(t, filepath.Join(movieDir, "sample.mp4"))
	mustWriteFixtureFile(t, filepath.Join(movieDir, "featurette.mkv"))
	mustWriteFixtureFile(t, filepath.Join(movieDir, "interview.mkv"))
	mustWriteFixtureFile(t, filepath.Join(movieDir, "deleted scene.mkv"))

	mixedLibrary := createDirectScanLibrary(t, ctx, svc, "Mixed", "mixed", mixedRoot)
	if mixedLibrary.Type != LibraryTypeMixed {
		t.Fatalf("expected normalized mixed library type, got %q", mixedLibrary.Type)
	}
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(mixedLibrary.ID, mixedLibrary.RootPath)); err != nil {
		t.Fatalf("run mixed sync: %v", err)
	}

	var movies []database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeMovie).Find(&movies).Error; err != nil {
		t.Fatalf("load movies: %v", err)
	}
	if len(movies) != 1 {
		t.Fatalf("expected one movie for mixed folder with extras, got %#v", movies)
	}
	if movies[0].Path != movieDir {
		t.Fatalf("expected mixed movie path to use folder %q, got %q", movieDir, movies[0].Path)
	}

	var seriesCount int64
	if err := db.WithContext(ctx).Model(&database.CatalogItem{}).Where("type = ?", catalog.ItemTypeSeries).Count(&seriesCount).Error; err != nil {
		t.Fatalf("count series: %v", err)
	}
	if seriesCount != 0 {
		t.Fatalf("expected no series for one non-extra mixed folder, got %d", seriesCount)
	}

	var assets []database.MediaAsset
	if err := db.WithContext(ctx).Where("library_id = ?", mixedLibrary.ID).Find(&assets).Error; err != nil {
		t.Fatalf("load assets: %v", err)
	}
	if len(assets) != 7 {
		t.Fatalf("expected main plus six extra assets, got %#v", assets)
	}
	roles := map[string]bool{}
	for _, asset := range assets {
		var link database.AssetItem
		if err := db.WithContext(ctx).Where("asset_id = ? AND item_id = ?", asset.ID, movies[0].ID).First(&link).Error; err != nil {
			t.Fatalf("load mixed movie asset link: %v", err)
		}
		roles[link.Role] = true
	}
	if !roles[inventory.AssetItemRolePrimary] || !roles[inventory.AssetItemRoleTrailer] || !roles[inventory.AssetItemRoleExtra] {
		t.Fatalf("expected primary, trailer, and extra roles, got %#v", roles)
	}
}

func TestRunSyncLibraryMixedClassifiesMultipleNonExtraVideosAsSeries(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	mixedRoot := filepath.Join(rootPath, "mixed")
	showDir := filepath.Join(mixedRoot, "Unsorted Show")
	mustWriteFixtureFile(t, filepath.Join(showDir, "trailer.mp4"))
	mustWriteFixtureFile(t, filepath.Join(showDir, "behind-the-scenes.mkv"))
	mustWriteFixtureFile(t, filepath.Join(showDir, "Unsorted.Show.S01E01.mkv"))
	mustWriteFixtureFile(t, filepath.Join(showDir, "Unsorted.Show.S01E02.mkv"))

	mixedLibrary := createDirectScanLibrary(t, ctx, svc, "Mixed", "mixed", mixedRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(mixedLibrary.ID, mixedLibrary.RootPath)); err != nil {
		t.Fatalf("run mixed sync: %v", err)
	}

	var series []database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeSeries).Find(&series).Error; err != nil {
		t.Fatalf("load series: %v", err)
	}
	if len(series) != 1 {
		t.Fatalf("expected one series for mixed multi-video folder, got %#v", series)
	}

	var episodes []database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeEpisode).Order("index_number asc").Find(&episodes).Error; err != nil {
		t.Fatalf("load episodes: %v", err)
	}
	if len(episodes) != 2 {
		t.Fatalf("expected only non-extra files as mixed episodes, got %#v", episodes)
	}
	for idx, episode := range episodes {
		expected := idx + 1
		if episode.IndexNumber == nil || *episode.IndexNumber != expected || episode.ParentIndexNumber == nil || *episode.ParentIndexNumber != 1 {
			t.Fatalf("unexpected mixed fallback episode %d: %#v", idx, episode)
		}
	}

	var movies int64
	if err := db.WithContext(ctx).Model(&database.CatalogItem{}).Where("type = ?", catalog.ItemTypeMovie).Count(&movies).Error; err != nil {
		t.Fatalf("count movies: %v", err)
	}
	if movies != 0 {
		t.Fatalf("expected no movies for mixed multi-video folder, got %d", movies)
	}
}

func TestRunSyncLibraryAutoClassifiesMovieVersionsAsOneMovie(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	movieDir := filepath.Join(rootPath, "Blade Runner (1982)")
	mustWriteFixtureFile(t, filepath.Join(movieDir, "Blade Runner 1080p.mkv"))
	mustWriteFixtureFile(t, filepath.Join(movieDir, "Blade Runner 2160p.mkv"))

	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Auto", LibraryTypeAuto, rootPath)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run auto sync: %v", err)
	}

	var movies []database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeMovie).Find(&movies).Error; err != nil {
		t.Fatalf("load movies: %v", err)
	}
	if len(movies) != 1 {
		t.Fatalf("expected one movie for auto movie versions, got %#v", movies)
	}
	var assets int64
	if err := db.WithContext(ctx).Model(&database.MediaAsset{}).Where("library_id = ?", libraryRecord.ID).Count(&assets).Error; err != nil {
		t.Fatalf("count assets: %v", err)
	}
	if assets != 2 {
		t.Fatalf("expected two assets for movie versions, got %d", assets)
	}
}

func TestRunSyncLibraryMarksLowConfidenceAutoClassificationForReview(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	ambiguousDir := filepath.Join(rootPath, "Ambiguous Pack")
	mustWriteFixtureFile(t, filepath.Join(ambiguousDir, "alpha.mkv"))
	mustWriteFixtureFile(t, filepath.Join(ambiguousDir, "beta.mkv"))

	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Auto", LibraryTypeAuto, rootPath)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run auto sync: %v", err)
	}

	var movie database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeMovie).First(&movie).Error; err != nil {
		t.Fatalf("load auto movie: %v", err)
	}
	if movie.GovernanceStatus != catalog.GovernanceNeedsReview {
		t.Fatalf("expected low-confidence auto decision to need review, got %#v", movie)
	}
}

func TestRunSyncLibraryMixedIgnoresExcludedFilesForDirectoryShape(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	mixedRoot := filepath.Join(rootPath, "mixed")
	movieDir := filepath.Join(mixedRoot, "Ambiguous Folder")
	mainPath := filepath.Join(movieDir, "Main Feature.mkv")
	ignoredPath := filepath.Join(movieDir, "Second Feature.mkv")
	mustWriteFixtureFile(t, mainPath)
	mustWriteFixtureFile(t, ignoredPath)

	mixedLibrary := createDirectScanLibrary(t, ctx, svc, "Mixed", "mixed", mixedRoot)
	if err := db.WithContext(ctx).Create(&database.ScanExclusion{LibraryID: mixedLibrary.ID, StorageProvider: "local", StoragePath: ignoredPath, Reason: ScanExclusionReasonWrongImport, Enabled: true}).Error; err != nil {
		t.Fatalf("create scan exclusion: %v", err)
	}
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(mixedLibrary.ID, mixedLibrary.RootPath)); err != nil {
		t.Fatalf("run mixed sync: %v", err)
	}

	var movies []database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeMovie).Find(&movies).Error; err != nil {
		t.Fatalf("load movies: %v", err)
	}
	if len(movies) != 1 {
		t.Fatalf("expected ignored mixed sibling to leave one movie, got %#v", movies)
	}

	var episodes int64
	if err := db.WithContext(ctx).Model(&database.CatalogItem{}).Where("type = ?", catalog.ItemTypeEpisode).Count(&episodes).Error; err != nil {
		t.Fatalf("count episodes: %v", err)
	}
	if episodes != 0 {
		t.Fatalf("expected no episodes after ignored sibling is excluded from shape, got %d", episodes)
	}
}

func TestRunSyncLibraryAppliesMovieFolderMetadataToGroupedVersions(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	movieDir := filepath.Join(moviesRoot, "Inception (2010)")
	mustWriteFixtureFile(t, filepath.Join(movieDir, "Inception.1080p.mkv"))
	mustWriteFixtureFile(t, filepath.Join(movieDir, "Inception.2160p.mkv"))
	mustWriteFixtureTextFile(t, filepath.Join(movieDir, "movie.json"), `{"title":"盗梦空间","year":2010}`)

	movieLibrary := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(movieLibrary.ID, movieLibrary.RootPath)); err != nil {
		t.Fatalf("run movie sync: %v", err)
	}

	var movie database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeMovie).First(&movie).Error; err != nil {
		t.Fatalf("load movie: %v", err)
	}
	if movie.Title != "盗梦空间" || movie.Year == nil || *movie.Year != 2010 {
		t.Fatalf("expected folder metadata to apply to grouped movie, got %#v", movie)
	}

	var source database.MetadataSource
	if err := db.WithContext(ctx).Where("item_id = ? AND source_name = ?", movie.ID, "scanner").First(&source).Error; err != nil {
		t.Fatalf("load scanner source: %v", err)
	}
	if !strings.Contains(source.PayloadJSON, "movie.json") {
		t.Fatalf("expected movie folder sidecar evidence, got %s", source.PayloadJSON)
	}
}

func TestQueueCatalogItemMatchDeduplicatesEpisodeHierarchyToSeriesRoot(t *testing.T) {
	t.Parallel()

	ctx, db, svc, libraryRecord := newScanCatalogWriterHarness(t)
	catalogSvc := catalog.NewService(db)

	series, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryRecord.ID, Type: catalog.ItemTypeSeries, Title: "Show A", Path: "/library/Show A", SortKey: "Show A", AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernancePending})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryRecord.ID, Type: catalog.ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", Path: "/library/Show A/season-01", SortKey: "Show A S01", IndexNumber: &seasonNumber, AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernancePending})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	episodeNumber := 2
	episode, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryRecord.ID, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 2", Path: "/library/Show A/season-01/episode-02", SortKey: "Show A S01E02", IndexNumber: &episodeNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernancePending})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}

	jobFromSeason, err := svc.QueueCatalogItemMatch(ctx, season.ID)
	if err != nil {
		t.Fatalf("queue season catalog match: %v", err)
	}
	jobFromEpisode, err := svc.QueueCatalogItemMatch(ctx, episode.ID)
	if err != nil {
		t.Fatalf("queue episode catalog match: %v", err)
	}
	if jobFromSeason.ID == 0 || jobFromEpisode.ID == 0 || jobFromSeason.ID != jobFromEpisode.ID {
		t.Fatalf("expected season and episode queue to dedupe to the same job, got season=%#v episode=%#v", jobFromSeason, jobFromEpisode)
	}

	var queued []database.WorkflowTask
	if err := db.WithContext(ctx).Where("task_type = ?", workflow.TaskTypeMatchMetadata).Find(&queued).Error; err != nil {
		t.Fatalf("list catalog match tasks: %v", err)
	}
	if len(queued) != 1 {
		t.Fatalf("expected one queued catalog match job, got %#v", queued)
	}

	var payload struct {
		ItemIDs []uint `json:"item_ids"`
	}
	if err := json.Unmarshal([]byte(queued[0].PayloadJSON), &payload); err != nil {
		t.Fatalf("decode task payload: %v", err)
	}
	if len(payload.ItemIDs) != 1 || payload.ItemIDs[0] != series.ID {
		t.Fatalf("expected queued catalog match to target series root %d, got %#v", series.ID, payload.ItemIDs)
	}
}

func TestRunSyncLibraryOrdersExplicitDuplicateEpisodeNamesAsSeparateEpisodes(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	showsRoot := filepath.Join(rootPath, "shows")
	mustWriteFixtureFile(t, filepath.Join(showsRoot, "Show One", "Season 1", "Show.One.S01E02.mkv"))
	mustWriteFixtureFile(t, filepath.Join(showsRoot, "Show One", "Season 1", "Show.One.S01E02.Directors.Cut.mkv"))

	showLibrary := createDirectScanLibrary(t, ctx, svc, "Shows", "shows", showsRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(showLibrary.ID, showLibrary.RootPath)); err != nil {
		t.Fatalf("run show sync: %v", err)
	}

	var episodeCount int64
	if err := db.WithContext(ctx).
		Model(&database.CatalogItem{}).
		Where("library_id = ? AND type = ?", showLibrary.ID, catalog.ItemTypeEpisode).
		Count(&episodeCount).Error; err != nil {
		t.Fatalf("count episode catalog items: %v", err)
	}
	if episodeCount != 2 {
		t.Fatalf("expected explicit duplicate episode names to follow sorted directory order, got %d", episodeCount)
	}

	assertTableCount(t, ctx, db, &database.InventoryFile{}, 2)
	assertTableCount(t, ctx, db, &database.MediaAsset{}, 2)
	assertTableCount(t, ctx, db, &database.AssetItem{}, 2)
	assertTableCount(t, ctx, db, &database.AssetFile{}, 2)

	var assets []database.MediaAsset
	if err := db.WithContext(ctx).
		Where("library_id = ?", showLibrary.ID).
		Order("id asc").
		Find(&assets).Error; err != nil {
		t.Fatalf("list media assets: %v", err)
	}
	if len(assets) != 2 {
		t.Fatalf("expected two assets, got %#v", assets)
	}
	if assets[0].AssetType != "main" {
		t.Fatalf("expected first asset to remain main, got %#v", assets[0])
	}
	if assets[1].AssetType != "main" {
		t.Fatalf("expected sorted duplicate episode names to create a second main asset, got %#v", assets[1])
	}

	var assetItems []database.AssetItem
	if err := db.WithContext(ctx).
		Order("asset_id asc, id asc").
		Find(&assetItems).Error; err != nil {
		t.Fatalf("list asset items: %v", err)
	}
	if len(assetItems) != 2 {
		t.Fatalf("expected two asset-item links, got %#v", assetItems)
	}
	if assetItems[0].Role != "primary" {
		t.Fatalf("expected first asset-item link to stay primary, got %#v", assetItems[0])
	}
	if assetItems[1].Role != "primary" {
		t.Fatalf("expected second sorted episode link to use primary role, got %#v", assetItems[1])
	}
}

func TestRunSyncLibraryMarksMissingInventoryWithoutDeletingCatalogItem(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	filePath := filepath.Join(moviesRoot, "Movie A (2024)", "Movie.A.2024.mkv")
	mustWriteFixtureFile(t, filePath)

	movieLibrary := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(movieLibrary.ID, movieLibrary.RootPath)); err != nil {
		t.Fatalf("run initial movie sync: %v", err)
	}
	if err := os.Remove(filePath); err != nil {
		t.Fatalf("remove scanned movie file: %v", err)
	}
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(movieLibrary.ID, movieLibrary.RootPath)); err != nil {
		t.Fatalf("run missing-file sync: %v", err)
	}

	assertTableCount(t, ctx, db, &database.CatalogItem{}, 1)
	assertTableCount(t, ctx, db, &database.InventoryFile{}, 1)
	assertTableCount(t, ctx, db, &database.MediaAsset{}, 1)
	assertTableCount(t, ctx, db, &database.AssetItem{}, 1)
	assertTableCount(t, ctx, db, &database.AssetFile{}, 1)
	assertTableCount(t, ctx, db, &database.MetadataSource{}, 1)

	var item database.CatalogItem
	if err := db.WithContext(ctx).
		Where("library_id = ? AND type = ?", movieLibrary.ID, catalog.ItemTypeMovie).
		First(&item).Error; err != nil {
		t.Fatalf("load movie catalog item: %v", err)
	}
	if item.AvailabilityStatus != catalog.AvailabilityMissing {
		t.Fatalf("expected missing movie availability after delete, got %#v", item)
	}
	if item.DeletedAt != nil {
		t.Fatalf("expected movie catalog item to remain undeleted, got deleted_at=%v", item.DeletedAt)
	}

	var file database.InventoryFile
	if err := db.WithContext(ctx).
		Where("library_id = ?", movieLibrary.ID).
		First(&file).Error; err != nil {
		t.Fatalf("load inventory file: %v", err)
	}
	if file.Status != "missing" {
		t.Fatalf("expected missing inventory status after delete, got %#v", file)
	}
	if file.DeletedAt != nil {
		t.Fatalf("expected inventory file to remain undeleted, got deleted_at=%v", file.DeletedAt)
	}

	var asset database.MediaAsset
	if err := db.WithContext(ctx).
		Where("library_id = ?", movieLibrary.ID).
		First(&asset).Error; err != nil {
		t.Fatalf("load media asset: %v", err)
	}
	if asset.Status != "missing" {
		t.Fatalf("expected missing asset status after delete, got %#v", asset)
	}
	if asset.DeletedAt != nil {
		t.Fatalf("expected media asset to remain undeleted, got deleted_at=%v", asset.DeletedAt)
	}
}

func TestRunSyncLibraryMarksMissingBeforeEnrichmentRuns(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	moviePath := filepath.Join(moviesRoot, "Movie A (2024)", "Movie.A.2024.mkv")
	mustWriteFixtureFile(t, moviePath)
	movieLibrary := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)

	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(movieLibrary.ID, movieLibrary.RootPath)); err != nil {
		t.Fatalf("run initial sync: %v", err)
	}
	if err := os.Remove(moviePath); err != nil {
		t.Fatalf("remove movie: %v", err)
	}
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(movieLibrary.ID, movieLibrary.RootPath)); err != nil {
		t.Fatalf("run delete sync: %v", err)
	}

	var file database.InventoryFile
	if err := db.WithContext(ctx).Where("storage_path = ?", moviePath).First(&file).Error; err != nil {
		t.Fatalf("load inventory file: %v", err)
	}
	if file.Status != inventory.FileStatusMissing {
		t.Fatalf("expected missing inventory before enrichment runs, got %#v", file)
	}
	var item database.CatalogItem
	if err := db.WithContext(ctx).Where("library_id = ? AND type = ?", movieLibrary.ID, catalog.ItemTypeMovie).First(&item).Error; err != nil {
		t.Fatalf("load catalog item: %v", err)
	}
	if item.AvailabilityStatus != catalog.AvailabilityMissing {
		t.Fatalf("expected missing catalog availability before enrichment runs, got %#v", item)
	}

	assertJobCount(t, ctx, db, JobKindMatchCatalogItem, 0)
	assertJobCount(t, ctx, db, JobKindProbeInventoryFile, 0)
}

func TestRunSyncLibraryKeepsEpisodeAvailableWhenAnotherVersionRemains(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	showsRoot := filepath.Join(rootPath, "shows")
	firstPath := filepath.Join(showsRoot, "Show One", "Season 1", "Show.One.S01E02.mkv")
	secondPath := filepath.Join(showsRoot, "Show One", "Season 1", "Show.One.S01E02.Directors.Cut.mkv")
	mustWriteFixtureFile(t, firstPath)
	mustWriteFixtureFile(t, secondPath)

	showLibrary := createDirectScanLibrary(t, ctx, svc, "Shows", "shows", showsRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(showLibrary.ID, showLibrary.RootPath)); err != nil {
		t.Fatalf("run initial show sync: %v", err)
	}
	if err := os.Remove(firstPath); err != nil {
		t.Fatalf("remove first version file: %v", err)
	}
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(showLibrary.ID, showLibrary.RootPath)); err != nil {
		t.Fatalf("run version delete sync: %v", err)
	}

	var episode database.CatalogItem
	if err := db.WithContext(ctx).
		Where("library_id = ? AND type = ?", showLibrary.ID, catalog.ItemTypeEpisode).
		First(&episode).Error; err != nil {
		t.Fatalf("load episode catalog item: %v", err)
	}
	if episode.AvailabilityStatus != catalog.AvailabilityAvailable {
		t.Fatalf("expected episode to stay available while another version remains, got %#v", episode)
	}

	var files []database.InventoryFile
	if err := db.WithContext(ctx).
		Where("library_id = ?", showLibrary.ID).
		Order("storage_path asc").
		Find(&files).Error; err != nil {
		t.Fatalf("list inventory files: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected two inventory files to remain recorded, got %#v", files)
	}
	if files[0].Status != "missing" && files[1].Status != "missing" {
		t.Fatalf("expected deleted version to be marked missing, got %#v", files)
	}
	if files[0].Status != "available" && files[1].Status != "available" {
		t.Fatalf("expected surviving version to remain available, got %#v", files)
	}

	var assets []database.MediaAsset
	if err := db.WithContext(ctx).
		Where("library_id = ?", showLibrary.ID).
		Order("id asc").
		Find(&assets).Error; err != nil {
		t.Fatalf("list media assets: %v", err)
	}
	if len(assets) != 2 {
		t.Fatalf("expected two media assets to remain recorded, got %#v", assets)
	}
	if assets[0].Status != "missing" && assets[1].Status != "missing" {
		t.Fatalf("expected one version asset to be marked missing, got %#v", assets)
	}
	if assets[0].Status != "available" && assets[1].Status != "available" {
		t.Fatalf("expected one version asset to remain available, got %#v", assets)
	}
}

func TestRunSyncLibraryReusesStableIdentityCatalogRowsOnRename(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, svc, libraryRecord := newIdentityScanService(t)
	provider := &stableIdentityProvider{objects: [][]storage.Object{
		{{Name: "MovieA.2024.mkv", Path: "/library/MovieA.2024.mkv", Size: 2048, StableIdentity: "provider-object-1", Modified: timePtr(time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC))}},
		{{Name: "Renamed.Movie.2024.mkv", Path: "/library/Renamed.Movie.2024.mkv", Size: 2048, StableIdentity: "provider-object-1", Modified: timePtr(time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC))}},
	}}

	if _, err := svc.scanLibrary(ctx, provider, libraryRecord, "/library"); err != nil {
		t.Fatalf("run initial scan: %v", err)
	}

	var firstFile database.InventoryFile
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).First(&firstFile).Error; err != nil {
		t.Fatalf("load first inventory file: %v", err)
	}
	var firstAsset database.MediaAsset
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).First(&firstAsset).Error; err != nil {
		t.Fatalf("load first media asset: %v", err)
	}
	var firstItem database.CatalogItem
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).First(&firstItem).Error; err != nil {
		t.Fatalf("load first catalog item: %v", err)
	}

	if _, err := svc.scanLibrary(ctx, provider, libraryRecord, "/library"); err != nil {
		t.Fatalf("run rename scan: %v", err)
	}

	assertTableCount(t, ctx, db, &database.InventoryFile{}, 1)
	assertTableCount(t, ctx, db, &database.MediaAsset{}, 1)
	assertTableCount(t, ctx, db, &database.AssetFile{}, 1)
	assertTableCount(t, ctx, db, &database.CatalogItem{}, 1)
	assertTableCount(t, ctx, db, &database.AssetItem{}, 1)

	var file database.InventoryFile
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).First(&file).Error; err != nil {
		t.Fatalf("reload inventory file: %v", err)
	}
	if file.ID != firstFile.ID {
		t.Fatalf("expected stable identity rename to reuse inventory file id %d, got %d", firstFile.ID, file.ID)
	}
	if file.StoragePath != "/library/Renamed.Movie.2024.mkv" {
		t.Fatalf("expected reused inventory file to move to renamed path, got %#v", file)
	}

	var asset database.MediaAsset
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).First(&asset).Error; err != nil {
		t.Fatalf("reload media asset: %v", err)
	}
	if asset.ID != firstAsset.ID {
		t.Fatalf("expected stable identity rename to reuse asset id %d, got %d", firstAsset.ID, asset.ID)
	}
	if asset.Status != "available" {
		t.Fatalf("expected reused asset to stay available after rename, got %#v", asset)
	}

	var item database.CatalogItem
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).First(&item).Error; err != nil {
		t.Fatalf("reload catalog item: %v", err)
	}
	if item.ID != firstItem.ID {
		t.Fatalf("expected stable identity rename to reuse catalog item id %d, got %d", firstItem.ID, item.ID)
	}
	if item.Path != "/library/Renamed.Movie.2024.mkv" {
		t.Fatalf("expected reused catalog item path to update, got %#v", item)
	}
}

func TestRunSyncLibraryScansEnabledLibraryPathsOnly(t *testing.T) {
	rootPath := t.TempDir()
	pathA := filepath.Join(rootPath, "a")
	pathB := filepath.Join(rootPath, "b")
	pathC := filepath.Join(rootPath, "c")
	mustWriteFixtureFile(t, filepath.Join(pathA, "Movie A (2020).mkv"))
	mustWriteFixtureFile(t, filepath.Join(pathB, "Movie B (2021).mkv"))
	mustWriteFixtureFile(t, filepath.Join(pathC, "Movie C (2022).mkv"))
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	source, err := svc.CreateMediaSource(ctx, CreateMediaSourceInput{Provider: "local", Name: "Local", RootPath: rootPath})
	if err != nil {
		t.Fatalf("create media source: %v", err)
	}
	libraryRecord, _, err := svc.CreateLibrary(ctx, CreateLibraryInput{Name: "Movies", Type: LibraryTypeMovies, MediaSourceID: source.ID, RootPath: pathA})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	enabledPath, err := svc.AddLibraryPath(ctx, libraryRecord.ID, LibraryPathInput{MediaSourceID: source.ID, RootPath: pathC})
	if err != nil {
		t.Fatalf("add enabled path: %v", err)
	}
	if enabledPath.RootPath != pathC {
		t.Fatalf("unexpected enabled path: %#v", enabledPath)
	}
	falseValue := false
	if _, err := svc.AddLibraryPath(ctx, libraryRecord.ID, LibraryPathInput{MediaSourceID: source.ID, RootPath: pathB, Enabled: &falseValue}); err != nil {
		t.Fatalf("add disabled path: %v", err)
	}
	if err := svc.RunSyncLibrary(ctx, database.Job{PayloadJSON: fmt.Sprintf(`{"library_id":%d}`, libraryRecord.ID)}); err != nil {
		t.Fatalf("run sync library: %v", err)
	}

	var files []database.InventoryFile
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).Order("storage_path asc").Find(&files).Error; err != nil {
		t.Fatalf("list inventory files: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected two files from enabled paths, got %#v", files)
	}
	for _, file := range files {
		if strings.Contains(file.StoragePath, string(filepath.Separator)+"b"+string(filepath.Separator)) {
			t.Fatalf("disabled path file was scanned: %#v", files)
		}
	}
}

func TestScanPolicyIgnoresExtensionsAndPreservesManualExclusionPrecedence(t *testing.T) {
	rootPath := t.TempDir()
	filePath := filepath.Join(rootPath, "Movie A (2020).mkv")
	mustWriteFixtureFile(t, filePath)
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", LibraryTypeMovies, rootPath)
	if err := db.WithContext(ctx).Model(&database.LibraryScanPolicy{}).Where("library_id = ?", libraryRecord.ID).Updates(map[string]any{"scanner_enabled": true, "realtime_monitor_enabled": true, "scheduled_refresh_enabled": true, "refresh_interval_hours": 24, "ignore_hidden_files": true, "ignore_file_extensions_json": `[".mkv"]`, "configurable_exclusion_rules": true}).Error; err != nil {
		t.Fatalf("set scan policy: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ScanExclusion{LibraryID: libraryRecord.ID, StorageProvider: "local", StoragePath: filePath, Reason: ScanExclusionReasonWrongImport, Enabled: true}).Error; err != nil {
		t.Fatalf("create scan exclusion: %v", err)
	}
	_, provider, err := svc.providerForSource(ctx, libraryRecord.MediaSourceID)
	if err != nil {
		t.Fatalf("provider for library: %v", err)
	}
	result, err := svc.scanLibrary(ctx, provider, libraryRecord, rootPath)
	if err != nil {
		t.Fatalf("scan library: %v", err)
	}
	if result.ExcludedFilesSkippedByReason[scanExclusionSkipUserExclusion] != 1 {
		t.Fatalf("expected manual exclusion to win over policy ignore, got %#v", result.ExcludedFilesSkippedByReason)
	}
}

func TestScanPolicyCanDisableConfigurableExclusionRules(t *testing.T) {
	rootPath := t.TempDir()
	mustWriteFixtureFile(t, filepath.Join(rootPath, "Movie ad (2020).mkv"))
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", LibraryTypeMovies, rootPath)
	if _, err := svc.CreateScanExclusionRule(ctx, ScanExclusionRuleInput{LibraryID: &libraryRecord.ID, Name: "Scoped ad", RuleType: ScanExclusionRuleTypeFilenameToken, Value: "ad", Reason: ScanExclusionReasonAdvertisement, Enabled: true}); err != nil {
		t.Fatalf("create scoped exclusion rule: %v", err)
	}
	if err := db.WithContext(ctx).Model(&database.LibraryScanPolicy{}).Where("library_id = ?", libraryRecord.ID).Updates(map[string]any{"scanner_enabled": true, "realtime_monitor_enabled": true, "scheduled_refresh_enabled": true, "refresh_interval_hours": 24, "ignore_hidden_files": true, "ignore_file_extensions_json": `[]`, "configurable_exclusion_rules": false}).Error; err != nil {
		t.Fatalf("set scan policy: %v", err)
	}
	_, provider, err := svc.providerForSource(ctx, libraryRecord.MediaSourceID)
	if err != nil {
		t.Fatalf("provider for library: %v", err)
	}
	result, err := svc.scanLibrary(ctx, provider, libraryRecord, rootPath)
	if err != nil {
		t.Fatalf("scan library: %v", err)
	}
	if result.FilesSeen != 1 || result.ExcludedFilesSkipped != 0 {
		t.Fatalf("expected configurable rule to be disabled, got %#v", result)
	}
}

func TestScanPolicyCanDisableInventoryProbeBatchJobs(t *testing.T) {
	rootPath := t.TempDir()
	mustWriteFixtureFile(t, filepath.Join(rootPath, "Movie A (2024).mkv"))
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", LibraryTypeMovies, rootPath)
	if err := db.WithContext(ctx).Model(&database.LibraryScanPolicy{}).Where("library_id = ?", libraryRecord.ID).Update("inventory_probe_batch_enabled", false).Error; err != nil {
		t.Fatalf("disable inventory probe batch: %v", err)
	}

	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run sync library: %v", err)
	}

	assertJobCount(t, ctx, db, JobKindInventoryProbeBatch, 0)
}

func TestScanPolicySkipsFilesBelowMinimumSize(t *testing.T) {
	rootPath := t.TempDir()
	mustWriteFixtureFile(t, filepath.Join(rootPath, "Tiny Movie (2020).mkv"))
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", LibraryTypeMovies, rootPath)
	if err := db.WithContext(ctx).Model(&database.LibraryScanPolicy{}).Where("library_id = ?", libraryRecord.ID).Update("min_file_size_bytes", int64(1024)).Error; err != nil {
		t.Fatalf("set min file size policy: %v", err)
	}
	_, provider, err := svc.providerForSource(ctx, libraryRecord.MediaSourceID)
	if err != nil {
		t.Fatalf("provider for library: %v", err)
	}
	result, err := svc.scanLibrary(ctx, provider, libraryRecord, rootPath)
	if err != nil {
		t.Fatalf("scan library: %v", err)
	}
	if result.FilesSeen != 0 || result.ExcludedFilesSkippedByReason["policy_min_size"] != 1 {
		t.Fatalf("expected tiny file to be skipped by min size policy, got %#v", result)
	}
}

func TestScanPolicyMinimumSizeZeroDoesNotLimit(t *testing.T) {
	rootPath := t.TempDir()
	mustWriteFixtureFile(t, filepath.Join(rootPath, "Tiny Movie (2020).mkv"))
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", LibraryTypeMovies, rootPath)
	if err := db.WithContext(ctx).Model(&database.LibraryScanPolicy{}).Where("library_id = ?", libraryRecord.ID).Update("min_file_size_bytes", int64(0)).Error; err != nil {
		t.Fatalf("set min file size policy: %v", err)
	}
	_, provider, err := svc.providerForSource(ctx, libraryRecord.MediaSourceID)
	if err != nil {
		t.Fatalf("provider for library: %v", err)
	}
	result, err := svc.scanLibrary(ctx, provider, libraryRecord, rootPath)
	if err != nil {
		t.Fatalf("scan library: %v", err)
	}
	if result.FilesSeen != 1 || result.ExcludedFilesSkipped != 0 {
		t.Fatalf("expected zero min size to allow tiny file, got %#v", result)
	}
}

func TestRunSyncLibraryDeduplicatesScannerMetadataSourcesOnRescan(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	moviePath := filepath.Join(moviesRoot, "Movie A (2024)", "Movie.A.2024.mkv")
	mustWriteFixtureFile(t, moviePath)

	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run initial sync: %v", err)
	}
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run rescan: %v", err)
	}

	assertTableCount(t, ctx, db, &database.CatalogItem{}, 1)
	assertTableCount(t, ctx, db, &database.MetadataSource{}, 1)
}

func TestRunSyncLibraryRecordsSubtitleSidecarEvidence(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	movieDir := filepath.Join(moviesRoot, "Movie A")
	moviePath := filepath.Join(movieDir, "Movie A.mkv")
	mustWriteFixtureFile(t, moviePath)
	mustWriteFixtureTextFile(t, filepath.Join(movieDir, "Movie A.srt"), "1\n00:00:01,000 --> 00:00:02,000\nMovie A\n")
	mustWriteFixtureTextFile(t, filepath.Join(movieDir, "Movie A.ass"), "[Script Info]\nTitle: Movie A\n")
	mustWriteFixtureTextFile(t, filepath.Join(movieDir, "Movie A.txt"), "ignored")

	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run sync: %v", err)
	}

	payload := scannerEvidencePayloadForSingleSource(t, ctx, db)
	subtitles, ok := payload["subtitle_sidecars"].([]any)
	if !ok || len(subtitles) != 2 {
		t.Fatalf("expected two subtitle sidecars, got %#v", payload["subtitle_sidecars"])
	}
	for _, raw := range subtitles {
		item, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("unexpected subtitle sidecar payload: %#v", raw)
		}
		if item["association_source"] != "basename" {
			t.Fatalf("expected basename association, got %#v", item)
		}
		if _, hasContent := item["content"]; hasContent {
			t.Fatalf("subtitle evidence must not include content: %#v", item)
		}
	}
}

func TestRunSyncLibraryBindsMovieSubtitleSidecars(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	movieDir := filepath.Join(moviesRoot, "Movie A")
	mustWriteFixtureFile(t, filepath.Join(movieDir, "Movie A.mkv"))
	mustWriteFixtureTextFile(t, filepath.Join(movieDir, "Movie A.srt"), "subtitle dialogue must not affect scan classification")
	mustWriteFixtureTextFile(t, filepath.Join(movieDir, "Movie A.ass"), "[Script Info]\nTitle: Ignored")

	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run sync: %v", err)
	}

	assertTableCount(t, ctx, db, &database.InventoryFile{}, 3)
	assertTableCount(t, ctx, db, &database.AssetFile{}, 3)
	assertTableCount(t, ctx, db, &database.MediaStream{}, 2)

	var subtitleLinks []database.AssetFile
	if err := db.WithContext(ctx).Where("role = ?", inventory.FileRoleSubtitle).Order("id asc").Find(&subtitleLinks).Error; err != nil {
		t.Fatalf("load subtitle links: %v", err)
	}
	if len(subtitleLinks) != 2 {
		t.Fatalf("expected two subtitle asset links, got %#v", subtitleLinks)
	}
	var subtitleFiles []database.InventoryFile
	if err := db.WithContext(ctx).Where("container IN ?", []string{"ass", "srt"}).Find(&subtitleFiles).Error; err != nil {
		t.Fatalf("load subtitle files: %v", err)
	}
	for _, file := range subtitleFiles {
		if file.Status != inventory.FileStatusAvailable {
			t.Fatalf("expected scanned subtitle inventory to remain available after cleanup, got %#v", file)
		}
	}
	var streams []database.MediaStream
	if err := db.WithContext(ctx).Where("stream_type = ?", inventory.MediaStreamTypeSubtitle).Order("codec asc").Find(&streams).Error; err != nil {
		t.Fatalf("load subtitle streams: %v", err)
	}
	if len(streams) != 2 || streams[0].Codec != "ass" || streams[1].Codec != "srt" {
		t.Fatalf("expected ass and srt subtitle streams, got %#v", streams)
	}
	for _, stream := range streams {
		if !strings.Contains(stream.DispositionJSON, `"external":true`) || !strings.Contains(stream.DispositionJSON, `"managed_by":"scanner"`) {
			t.Fatalf("expected scanner-managed external disposition, got %#v", stream)
		}
	}
}

func TestRunSyncLibraryRespectsExternalSubtitleDisabledPolicy(t *testing.T) {
	rootPath := t.TempDir()
	movieDir := filepath.Join(rootPath, "Movie A (2024)")
	mustWriteFixtureFile(t, filepath.Join(movieDir, "Movie A.mkv"))
	mustWriteFixtureTextFile(t, filepath.Join(movieDir, "Movie A.srt"), "subtitle")
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", LibraryTypeMovies, rootPath)
	if err := db.WithContext(ctx).Model(&database.LibrarySubtitlePolicy{}).Where("library_id = ?", libraryRecord.ID).Update("external_sidecars_enabled", false).Error; err != nil {
		t.Fatalf("disable external subtitles: %v", err)
	}
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run sync library: %v", err)
	}
	assertRawTableCount(t, db, "asset_files", "role = ?", 0, inventory.FileRoleSubtitle)
	assertRawTableCount(t, db, "media_streams", "stream_type = ?", 0, inventory.MediaStreamTypeSubtitle)
}

func TestRunSyncLibraryBindsEpisodeSubtitleSidecar(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	showsRoot := filepath.Join(rootPath, "shows")
	episodeDir := filepath.Join(showsRoot, "Show One", "Season 1")
	mustWriteFixtureFile(t, filepath.Join(episodeDir, "Show.One.S01E02.mkv"))
	mustWriteFixtureTextFile(t, filepath.Join(episodeDir, "Show.One.S01E02.ass"), "[Events]\nDialogue: 0,0:00:00.00,0:00:01.00,Default,,0,0,0,,Ignored")

	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Shows", "shows", showsRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run sync: %v", err)
	}

	var episode database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ?", catalog.ItemTypeEpisode).First(&episode).Error; err != nil {
		t.Fatalf("load episode: %v", err)
	}
	if episode.IndexNumber == nil || *episode.IndexNumber != 1 {
		t.Fatalf("subtitle content must not change episode classification, got %#v", episode)
	}
	var stream database.MediaStream
	if err := db.WithContext(ctx).Where("stream_type = ? AND codec = ?", inventory.MediaStreamTypeSubtitle, "ass").First(&stream).Error; err != nil {
		t.Fatalf("load episode subtitle stream: %v", err)
	}
	if !strings.Contains(stream.DispositionJSON, `"external":true`) {
		t.Fatalf("expected external episode subtitle stream, got %#v", stream)
	}
}

func TestRunSyncLibraryReconcilesSubtitleSidecarsOnRescan(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	movieDir := filepath.Join(moviesRoot, "Movie A")
	subtitlePath := filepath.Join(movieDir, "Movie A.srt")
	mustWriteFixtureFile(t, filepath.Join(movieDir, "Movie A.mkv"))
	mustWriteFixtureTextFile(t, subtitlePath, "subtitle")

	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run initial sync: %v", err)
	}
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run idempotent rescan: %v", err)
	}
	assertRawTableCount(t, db, "asset_files", "role = ?", 1, inventory.FileRoleSubtitle)
	assertRawTableCount(t, db, "media_streams", "stream_type = ?", 1, inventory.MediaStreamTypeSubtitle)

	if err := os.Remove(subtitlePath); err != nil {
		t.Fatalf("remove subtitle: %v", err)
	}
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run stale subtitle rescan: %v", err)
	}
	assertRawTableCount(t, db, "asset_files", "role = ?", 0, inventory.FileRoleSubtitle)
	assertRawTableCount(t, db, "media_streams", "stream_type = ?", 0, inventory.MediaStreamTypeSubtitle)
}

func TestRunSyncLibraryUsesJSONSidecarMovieHints(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	movieDir := filepath.Join(moviesRoot, "Noisy Folder")
	moviePath := filepath.Join(movieDir, "video.mkv")
	mustWriteFixtureFile(t, moviePath)
	mustWriteFixtureTextFile(t, filepath.Join(movieDir, "video.json"), `{"title":"Sidecar Movie","original_title":"Original Sidecar Movie","year":2026,"external_ids":{"tmdb":"12345"}}`)

	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run sync: %v", err)
	}

	var item database.CatalogItem
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).First(&item).Error; err != nil {
		t.Fatalf("load catalog item: %v", err)
	}
	if item.Title != "Sidecar Movie" || item.OriginalTitle != "Original Sidecar Movie" || item.Year == nil || *item.Year != 2026 {
		t.Fatalf("expected JSON sidecar hints on movie item, got %#v", item)
	}

	payload := scannerEvidencePayloadForSingleSource(t, ctx, db)
	metadata := metadataSidecarsFromPayload(t, payload)
	if len(metadata) != 1 || metadata[0]["parse_status"] != "parsed" {
		t.Fatalf("expected parsed metadata sidecar, got %#v", metadata)
	}
	hints, ok := metadata[0]["hints"].(map[string]any)
	if !ok || hints["title"] != "Sidecar Movie" {
		t.Fatalf("expected metadata hints in evidence, got %#v", metadata[0])
	}

	var externalID database.CatalogExternalID
	if err := db.WithContext(ctx).Where("item_id = ? AND provider = ? AND provider_type = ?", item.ID, "tmdb", "movie").First(&externalID).Error; err != nil {
		t.Fatalf("load sidecar external id: %v", err)
	}
	if externalID.ExternalID != "movie:12345" || externalID.Source != "scanner" {
		t.Fatalf("expected scanner TMDB external id, got %#v", externalID)
	}
}

func TestRunSyncLibraryPreselectsSiblingArtworkDuringFirstScan(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	svc.cfg.FFmpeg.ArtworkRootPath = filepath.Join(rootPath, "artwork")
	moviesRoot := filepath.Join(rootPath, "movies")
	movieDir := filepath.Join(moviesRoot, "Movie A")
	mustWriteFixtureFile(t, filepath.Join(movieDir, "Movie A.mkv"))
	mustWriteFixtureTextFile(t, filepath.Join(movieDir, "poster.jpg"), "poster-bytes")
	mustWriteFixtureTextFile(t, filepath.Join(movieDir, "backdrop.jpg"), "backdrop-bytes")

	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run sync: %v", err)
	}

	var item database.CatalogItem
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).First(&item).Error; err != nil {
		t.Fatalf("load item: %v", err)
	}
	selected := selectedImagesByType(t, ctx, db, item.ID)
	if selected["poster"].URL != fmt.Sprintf("/api/v1/items/%d/artwork/poster", item.ID) || !selected["poster"].IsSelected {
		t.Fatalf("expected selected scan poster, got %#v", selected["poster"])
	}
	if selected["backdrop"].URL != fmt.Sprintf("/api/v1/items/%d/artwork/backdrop", item.ID) || !selected["backdrop"].IsSelected {
		t.Fatalf("expected selected scan backdrop, got %#v", selected["backdrop"])
	}
	posterBytes, err := os.ReadFile(filepath.Join(svc.cfg.FFmpeg.ArtworkRootPath, "catalog", fmt.Sprint(item.ID), "poster.jpg"))
	if err != nil || string(posterBytes) != "poster-bytes" {
		t.Fatalf("expected copied poster artwork, bytes=%q err=%v", string(posterBytes), err)
	}
}

func TestCatalogScanUsesProviderThumbnailAsProvisionalArtwork(t *testing.T) {
	t.Parallel()

	ctx, db, svc, libraryRecord := newScanCatalogWriterHarness(t)
	artifact := applyCatalogScanArtworkCandidates(nil, catalogScanArtifact{ItemType: catalog.ItemTypeMovie, ItemPath: "movie-a", SourcePath: "/library/Movie A.mkv", Title: "Movie A"}, storage.Object{Path: "/library/Movie A.mkv", ThumbnailURL: "https://cdn.example.test/movie-thumb.jpg"}, scanDirectorySnapshot{})
	result, err := svc.writeCatalogScanMovie(ctx, libraryRecord, artifact)
	if err != nil {
		t.Fatalf("write scan movie: %v", err)
	}

	selected := selectedImagesByType(t, ctx, db, result.Item.ID)
	if selected["poster"].URL != "https://cdn.example.test/movie-thumb.jpg" || !selected["poster"].IsSelected {
		t.Fatalf("expected provider thumbnail poster, got %#v", selected["poster"])
	}
	if result.File.ThumbnailURL != "https://cdn.example.test/movie-thumb.jpg" {
		t.Fatalf("expected inventory thumbnail to be persisted, got %#v", result.File)
	}

	seasonNumber := 1
	episodeArtifact := applyCatalogScanArtworkCandidates(nil, catalogScanArtifact{ItemType: catalog.ItemTypeEpisode, SourcePath: "/library/Show/Season 1/Show.S01E01.mkv", SeriesPath: "show", SeasonPath: "show/season-01", Title: "Pilot", SeriesTitle: "Show", SeasonNumber: &seasonNumber, EpisodeSlots: []catalogEpisodeSlot{{EpisodeNumber: 1, ItemPath: "show/season-01/episode-0001"}}}, storage.Object{Path: "/library/Show/Season 1/Show.S01E01.mkv", ThumbnailURL: "https://cdn.example.test/episode-thumb.jpg"}, scanDirectorySnapshot{})
	episodeResult, err := svc.writeCatalogScanEpisodeHierarchy(ctx, libraryRecord, episodeArtifact)
	if err != nil {
		t.Fatalf("write scan episode: %v", err)
	}
	selected = selectedImagesByType(t, ctx, db, episodeResult.Item.ID)
	if selected["still"].URL != "https://cdn.example.test/episode-thumb.jpg" || !selected["still"].IsSelected {
		t.Fatalf("expected provider thumbnail still, got %#v", selected["still"])
	}

	var series database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ? AND title = ?", catalog.ItemTypeSeries, "Show").First(&series).Error; err != nil {
		t.Fatalf("load series: %v", err)
	}
	selected = selectedImagesByType(t, ctx, db, series.ID)
	if selected["poster"].URL != "https://cdn.example.test/episode-thumb.jpg" || !selected["poster"].IsSelected {
		t.Fatalf("expected episode thumbnail as provisional series poster, got %#v", selected["poster"])
	}
	if selected["backdrop"].URL != "https://cdn.example.test/episode-thumb.jpg" || !selected["backdrop"].IsSelected {
		t.Fatalf("expected episode thumbnail as provisional series backdrop, got %#v", selected["backdrop"])
	}
	if selected["poster"].SourceID == nil || selected["backdrop"].SourceID == nil {
		t.Fatalf("expected provisional series artwork to retain scanner source, got poster=%#v backdrop=%#v", selected["poster"], selected["backdrop"])
	}
}

func TestCatalogScanPreservesExistingSelectedArtwork(t *testing.T) {
	t.Parallel()

	ctx, db, svc, libraryRecord := newScanCatalogWriterHarness(t)
	artifact := catalogScanArtifact{ItemType: catalog.ItemTypeMovie, ItemPath: "movie-a", SourcePath: "/library/Movie A.mkv", Title: "Movie A", ImageCandidates: []catalogScanImageCandidate{{ImageType: "poster", URL: "https://cdn.example.test/scanner-poster.jpg", Source: catalogScanArtworkSourceProviderThumb, Priority: 100, Provisional: true}}}
	result, err := svc.writeCatalogScanMovie(ctx, libraryRecord, artifact)
	if err != nil {
		t.Fatalf("write scan movie: %v", err)
	}
	if err := db.WithContext(ctx).Model(&database.ItemImage{}).Where("item_id = ? AND image_type = ?", result.Item.ID, "poster").Update("is_selected", false).Error; err != nil {
		t.Fatalf("clear scanner poster selection: %v", err)
	}
	remote := database.ItemImage{ItemID: result.Item.ID, ImageType: "poster", URL: "https://image.example.test/remote-poster.jpg", IsSelected: true}
	if err := db.WithContext(ctx).Create(&remote).Error; err != nil {
		t.Fatalf("seed remote poster: %v", err)
	}

	artifact.ImageCandidates = []catalogScanImageCandidate{{ImageType: "poster", URL: "https://cdn.example.test/new-scanner-poster.jpg", Source: catalogScanArtworkSourceProviderThumb, Priority: 100, Provisional: true}}
	if _, err := svc.writeCatalogScanMovie(ctx, libraryRecord, artifact); err != nil {
		t.Fatalf("rescan movie: %v", err)
	}

	selected := selectedImagesByType(t, ctx, db, result.Item.ID)
	if selected["poster"].URL != remote.URL {
		t.Fatalf("expected existing selected remote poster to stay selected, got %#v", selected["poster"])
	}
}

func TestRunSyncLibraryUsesNFOSidecarEpisodeHints(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	showsRoot := filepath.Join(rootPath, "shows")
	episodeDir := filepath.Join(showsRoot, "Downloads")
	episodePath := filepath.Join(episodeDir, "file.mkv")
	mustWriteFixtureFile(t, episodePath)
	mustWriteFixtureTextFile(t, filepath.Join(episodeDir, "file.nfo"), `<episodedetails><title>Pilot</title><showtitle>Sidecar Show</showtitle><season>1</season><episode>2</episode></episodedetails>`)

	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Shows", "shows", showsRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run sync: %v", err)
	}

	var items []database.CatalogItem
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).Order("type asc, id asc").Find(&items).Error; err != nil {
		t.Fatalf("load catalog items: %v", err)
	}
	var episode database.CatalogItem
	for _, item := range items {
		if item.Type == catalog.ItemTypeEpisode {
			episode = item
			break
		}
	}
	if episode.ID == 0 || episode.Title != "Pilot" || episode.IndexNumber == nil || *episode.IndexNumber != 2 || episode.ParentIndexNumber == nil || *episode.ParentIndexNumber != 1 {
		t.Fatalf("expected NFO sidecar episode hints, got items=%#v episode=%#v", items, episode)
	}

	payload := scannerEvidencePayloadForItem(t, ctx, db, episode.ID)
	metadata := metadataSidecarsFromPayload(t, payload)
	if len(metadata) != 1 || metadata[0]["parse_status"] != "parsed" {
		t.Fatalf("expected parsed NFO sidecar evidence, got %#v", metadata)
	}
}

func TestRunSyncLibrarySidecarFailuresAreNonFatal(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	malformedDir := filepath.Join(moviesRoot, "Malformed")
	oversizedDir := filepath.Join(moviesRoot, "Oversized")
	mustWriteFixtureFile(t, filepath.Join(malformedDir, "Malformed.mkv"))
	mustWriteFixtureTextFile(t, filepath.Join(malformedDir, "Malformed.json"), `{not-json`)
	mustWriteFixtureFile(t, filepath.Join(oversizedDir, "Oversized.mkv"))
	mustWriteFixtureTextFile(t, filepath.Join(oversizedDir, "Oversized.json"), strings.Repeat("x", maxSidecarMetadataBytes+1))

	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run sync: %v", err)
	}
	assertTableCount(t, ctx, db, &database.CatalogItem{}, 2)

	var sources []database.MetadataSource
	if err := db.WithContext(ctx).Order("id asc").Find(&sources).Error; err != nil {
		t.Fatalf("load metadata sources: %v", err)
	}
	statuses := make(map[string]bool)
	for _, source := range sources {
		var payload map[string]any
		if err := json.Unmarshal([]byte(source.PayloadJSON), &payload); err != nil {
			t.Fatalf("decode evidence payload: %v", err)
		}
		for _, sidecar := range metadataSidecarsFromPayload(t, payload) {
			statuses[fmt.Sprint(sidecar["parse_status"])] = true
		}
	}
	if !statuses["malformed"] || !statuses["skipped"] {
		t.Fatalf("expected malformed and skipped sidecar statuses, got %#v", statuses)
	}
}

func TestRunSyncLibrarySkipsAmbiguousFolderMetadataSidecar(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	movieDir := filepath.Join(moviesRoot, "Double Feature")
	mustWriteFixtureFile(t, filepath.Join(movieDir, "Movie A.mkv"))
	mustWriteFixtureFile(t, filepath.Join(movieDir, "Movie B.mkv"))
	mustWriteFixtureTextFile(t, filepath.Join(movieDir, "metadata.json"), `{"title":"Wrong Shared Title","year":2026}`)

	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run sync: %v", err)
	}

	var items []database.CatalogItem
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).Order("title asc").Find(&items).Error; err != nil {
		t.Fatalf("load items: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected two movie items, got %#v", items)
	}
	for _, item := range items {
		if item.Title == "Wrong Shared Title" {
			t.Fatalf("ambiguous folder metadata should not apply, got %#v", items)
		}
	}
}

func TestRunSyncLibraryMarksAncestorAvailabilityMissingWhenEpisodesDeleted(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	showsRoot := filepath.Join(rootPath, "shows")
	episodePath := filepath.Join(showsRoot, "Show One", "Season 1", "Show.One.S01E02.mkv")
	mustWriteFixtureFile(t, episodePath)

	showLibrary := createDirectScanLibrary(t, ctx, svc, "Shows", "shows", showsRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(showLibrary.ID, showLibrary.RootPath)); err != nil {
		t.Fatalf("run initial show sync: %v", err)
	}
	if err := os.Remove(episodePath); err != nil {
		t.Fatalf("remove episode file: %v", err)
	}
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(showLibrary.ID, showLibrary.RootPath)); err != nil {
		t.Fatalf("run delete sync: %v", err)
	}

	var items []database.CatalogItem
	if err := db.WithContext(ctx).Where("library_id = ?", showLibrary.ID).Order("id asc").Find(&items).Error; err != nil {
		t.Fatalf("list catalog items: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected series, season, and episode rows, got %#v", items)
	}
	for _, item := range items {
		if item.AvailabilityStatus != catalog.AvailabilityMissing {
			t.Fatalf("expected deleted hierarchy item to become missing, got %#v", item)
		}
	}
	if items[0].Type != catalog.ItemTypeSeries || items[1].Type != catalog.ItemTypeSeason || items[2].Type != catalog.ItemTypeEpisode {
		t.Fatalf("unexpected item ordering for hierarchy: %#v", items)
	}
}

func TestCatalogItemAvailabilityKeepsSeriesAvailableWithRemainingEpisode(t *testing.T) {
	t.Parallel()

	ctx, db, _, libraryRecord := newScanCatalogWriterHarness(t)
	catalogSvc := catalog.NewService(db)
	series, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryRecord.ID, Type: catalog.ItemTypeSeries, Title: "Show", AvailabilityStatus: catalog.AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryRecord.ID, Type: catalog.ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", IndexNumber: &seasonNumber, AvailabilityStatus: catalog.AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	availableEpisodeNumber := 1
	availableEpisode, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryRecord.ID, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 1", IndexNumber: &availableEpisodeNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: catalog.AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create available episode: %v", err)
	}
	asset := database.MediaAsset{LibraryID: libraryRecord.ID, AssetType: inventory.AssetTypeMain, Status: inventory.AssetStatusAvailable, ProbeStatus: "ready"}
	if err := db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.AssetItem{AssetID: asset.ID, ItemID: availableEpisode.ID, Role: inventory.AssetItemRolePrimary, Source: "scanner"}).Error; err != nil {
		t.Fatalf("link asset: %v", err)
	}
	missingEpisodeNumber := 2
	if _, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryRecord.ID, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 2", IndexNumber: &missingEpisodeNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: catalog.AvailabilityMissing}); err != nil {
		t.Fatalf("create missing episode: %v", err)
	}

	seriesAvailability, err := catalogItemAvailabilityStatus(ctx, db, series.ID)
	if err != nil {
		t.Fatalf("series availability: %v", err)
	}
	seasonAvailability, err := catalogItemAvailabilityStatus(ctx, db, season.ID)
	if err != nil {
		t.Fatalf("season availability: %v", err)
	}
	if seriesAvailability != catalog.AvailabilityAvailable || seasonAvailability != catalog.AvailabilityAvailable {
		t.Fatalf("expected remaining available episode to keep ancestors available, got series=%s season=%s", seriesAvailability, seasonAvailability)
	}
}

func TestScanCatalogWriterCreatesMovieKernelRows(t *testing.T) {
	t.Parallel()

	ctx, db, svc, libraryRecord := newScanCatalogWriterHarness(t)
	modifiedAt := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	year := 2024

	result, err := svc.writeCatalogScanMovie(ctx, libraryRecord, catalogScanArtifact{
		ItemType:          catalog.ItemTypeMovie,
		ItemPath:          "/library/Movie A (2024)/movie.mkv",
		SourcePath:        "/library/Movie A (2024)/movie.mkv",
		Title:             "Movie A",
		OriginalTitle:     "Movie.A.2024",
		Year:              &year,
		StorageProvider:   "local",
		StableIdentityKey: "local:movie-a-2024",
		ProviderName:      "scanner-provider",
		HashesJSON:        `{"sha256":"abc123"}`,
		SizeBytes:         4096,
		ModifiedAt:        &modifiedAt,
		Container:         "mkv",
	})
	if err != nil {
		t.Fatalf("write movie artifact: %v", err)
	}

	if result.Item.Type != catalog.ItemTypeMovie {
		t.Fatalf("expected movie item, got %#v", result.Item)
	}
	if result.File.Status != "available" {
		t.Fatalf("expected available file, got %#v", result.File)
	}
	if result.Asset.AssetType != "main" {
		t.Fatalf("expected main asset, got %#v", result.Asset)
	}

	assertCatalogCounts(t, ctx, db, 1, 1, 1, 1, 1, 1)

	var item database.CatalogItem
	if err := db.WithContext(ctx).First(&item, result.Item.ID).Error; err != nil {
		t.Fatalf("reload catalog item: %v", err)
	}
	if item.AvailabilityStatus != catalog.AvailabilityAvailable || item.GovernanceStatus != catalog.GovernancePending {
		t.Fatalf("expected available/pending item, got %#v", item)
	}

	var assetItem database.AssetItem
	if err := db.WithContext(ctx).First(&assetItem, "asset_id = ?", result.Asset.ID).Error; err != nil {
		t.Fatalf("load asset item: %v", err)
	}
	if assetItem.Role != "primary" || assetItem.SegmentIndex != 0 || assetItem.Source != "scanner" {
		t.Fatalf("unexpected asset-item link: %#v", assetItem)
	}

	var assetFile database.AssetFile
	if err := db.WithContext(ctx).First(&assetFile, "asset_id = ?", result.Asset.ID).Error; err != nil {
		t.Fatalf("load asset file: %v", err)
	}
	if assetFile.Role != "source" || assetFile.PartIndex != 0 {
		t.Fatalf("unexpected asset-file link: %#v", assetFile)
	}

	var source database.MetadataSource
	if err := db.WithContext(ctx).First(&source, "item_id = ?", result.Item.ID).Error; err != nil {
		t.Fatalf("load metadata source: %v", err)
	}
	if source.SourceType != catalog.SourceTypeLocalFile || source.SourceName != "scanner" {
		t.Fatalf("unexpected metadata source: %#v", source)
	}
	assertEvidencePayloadKeys(t, source.PayloadJSON, []string{"detected_title", "hashes_json", "provider_name", "stable_identity_key", "storage_path"})
}

func TestScanCatalogWriterPreservesMetadataFieldsOnMovieRescan(t *testing.T) {
	t.Parallel()

	ctx, db, svc, libraryRecord := newScanCatalogWriterHarness(t)
	catalogSvc := catalog.NewService(db)
	initialYear := 2024
	matchedYear := 2025
	rescannedYear := 2026
	artifact := catalogScanArtifact{
		ItemType:        catalog.ItemTypeMovie,
		ItemPath:        "/library/Movie A (2024)/movie.mkv",
		SourcePath:      "/library/Movie A (2024)/movie.mkv",
		Title:           "Movie A",
		OriginalTitle:   "Movie.A.2024",
		Year:            &initialYear,
		StorageProvider: "local",
		SizeBytes:       4096,
		Container:       "mkv",
	}

	result, err := svc.writeCatalogScanMovie(ctx, libraryRecord, artifact)
	if err != nil {
		t.Fatalf("write initial movie artifact: %v", err)
	}
	if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: result.Item.ID, FieldKey: "title", Value: "Matched Movie"}); err != nil {
		t.Fatalf("apply matched title: %v", err)
	}
	if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: result.Item.ID, FieldKey: "original_title", Value: "Matched Original"}); err != nil {
		t.Fatalf("apply matched original title: %v", err)
	}
	if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: result.Item.ID, FieldKey: "year", Value: matchedYear}); err != nil {
		t.Fatalf("apply matched year: %v", err)
	}
	if err := db.WithContext(ctx).Model(&database.CatalogItem{}).Where("id = ?", result.Item.ID).Updates(map[string]any{
		"title":          "Previously Reverted Title",
		"original_title": "Previously.Reverted",
		"year":           2001,
	}).Error; err != nil {
		t.Fatalf("simulate stale scanner overwrite: %v", err)
	}

	artifact.Title = "Movie A Remux"
	artifact.OriginalTitle = "Movie.A.Remux.2026"
	artifact.Year = &rescannedYear
	if _, err := svc.writeCatalogScanMovie(ctx, libraryRecord, artifact); err != nil {
		t.Fatalf("write rescan movie artifact: %v", err)
	}

	var item database.CatalogItem
	if err := db.WithContext(ctx).First(&item, result.Item.ID).Error; err != nil {
		t.Fatalf("reload movie item: %v", err)
	}
	if item.Title != "Matched Movie" || item.OriginalTitle != "Matched Original" || item.Year == nil || *item.Year != matchedYear {
		t.Fatalf("expected metadata fields to survive rescan, got %#v", item)
	}
}

func TestScanCatalogWriterCreatesEpisodeHierarchyWithLocalEvidence(t *testing.T) {
	t.Parallel()

	ctx, db, svc, libraryRecord := newScanCatalogWriterHarness(t)
	modifiedAt := time.Date(2026, 4, 25, 12, 30, 0, 0, time.UTC)
	seasonNumber := 1

	result, err := svc.writeCatalogScanEpisodeHierarchy(ctx, libraryRecord, catalogScanArtifact{
		ItemType:          catalog.ItemTypeEpisode,
		SourcePath:        "/library/Show One/Season 1/Show One.S01E02.mkv",
		SeriesPath:        "show-one",
		SeasonPath:        "show-one/season-01",
		Title:             "Show One S01E02",
		OriginalTitle:     "Show.One.S01E02",
		SeriesTitle:       "Show One",
		SeasonNumber:      &seasonNumber,
		EpisodeSlots:      []catalogEpisodeSlot{{EpisodeNumber: 2, ItemPath: "show-one/season-01/episode-0002"}},
		StorageProvider:   "local",
		StableIdentityKey: "local:show-one-s01e02",
		ProviderName:      "scanner-provider",
		HashesJSON:        `{"sha1":"deadbeef"}`,
		SizeBytes:         8192,
		ModifiedAt:        &modifiedAt,
		Container:         "mkv",
	})
	if err != nil {
		t.Fatalf("write episode artifact: %v", err)
	}

	assertCatalogCounts(t, ctx, db, 3, 1, 1, 1, 1, 2)

	var items []database.CatalogItem
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).Order("id asc").Find(&items).Error; err != nil {
		t.Fatalf("list catalog items: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected series/season/episode rows, got %#v", items)
	}
	for _, item := range items {
		if item.GovernanceStatus != catalog.GovernancePending || item.AvailabilityStatus != catalog.AvailabilityAvailable {
			t.Fatalf("expected pending/available hierarchy row, got %#v", item)
		}
	}
	if items[0].Path != "show-one" || items[1].Path != "show-one/season-01" || items[2].Path != "show-one/season-01/episode-0002" {
		t.Fatalf("unexpected canonical hierarchy paths: %#v", items)
	}
	if items[1].ParentID == nil || *items[1].ParentID != items[0].ID {
		t.Fatalf("expected season parent link, got %#v", items[1])
	}
	if items[2].ParentID == nil || *items[2].ParentID != items[1].ID {
		t.Fatalf("expected episode parent link, got %#v", items[2])
	}

	var source database.MetadataSource
	if err := db.WithContext(ctx).First(&source, "item_id = ?", result.Item.ID).Error; err != nil {
		t.Fatalf("load episode metadata source: %v", err)
	}
	if source.SourceType != catalog.SourceTypeLocalFile || source.SourceName != "scanner" {
		t.Fatalf("unexpected metadata source: %#v", source)
	}
	assertEvidencePayloadKeys(t, source.PayloadJSON, []string{"detected_title", "episode_numbers", "hashes_json", "provider_name", "season_number", "series_title", "stable_identity_key", "storage_path"})

	var payload map[string]any
	if err := json.Unmarshal([]byte(source.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode payload json: %v", err)
	}
	episodes, ok := payload["episode_numbers"].([]any)
	if !ok || len(episodes) != 1 || int(episodes[0].(float64)) != 2 {
		t.Fatalf("expected compact episode evidence, got %#v", payload)
	}
	if payload["series_title"] != "Show One" || int(payload["season_number"].(float64)) != 1 {
		t.Fatalf("expected series and season evidence, got %#v", payload)
	}

	var assetItem database.AssetItem
	if err := db.WithContext(ctx).First(&assetItem, "asset_id = ?", result.Asset.ID).Error; err != nil {
		t.Fatalf("load asset item: %v", err)
	}
	if assetItem.Role != "primary" || assetItem.Source != "scanner" {
		t.Fatalf("unexpected episode asset-item link: %#v", assetItem)
	}
	if result.Item.Path != "show-one/season-01/episode-0002" {
		t.Fatalf("expected episode leaf item, got %#v", result.Item)
	}
}

func TestScanCatalogWriterReusesProviderCreatedDescendantsByHierarchyIdentity(t *testing.T) {
	t.Parallel()

	ctx, db, svc, libraryRecord := newScanCatalogWriterHarness(t)
	catalogSvc := catalog.NewService(db)
	series, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{
		LibraryID:          libraryRecord.ID,
		Type:               catalog.ItemTypeSeries,
		Path:               "show-one",
		SortKey:            "Show One",
		Title:              "Matched Show One",
		AvailabilityStatus: catalog.AvailabilityMissing,
		GovernanceStatus:   catalog.GovernanceMatched,
	})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{
		LibraryID:          libraryRecord.ID,
		Type:               catalog.ItemTypeSeason,
		ParentID:           &series.ID,
		Path:               "show-one/Season 01",
		SortKey:            "Show One S01",
		Title:              "Season 1",
		IndexNumber:        &seasonNumber,
		ParentIndexNumber:  &seasonNumber,
		AvailabilityStatus: catalog.AvailabilityMissing,
		GovernanceStatus:   catalog.GovernanceMatched,
	})
	if err != nil {
		t.Fatalf("create provider season: %v", err)
	}
	episodeNumber := 2
	episode, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{
		LibraryID:          libraryRecord.ID,
		Type:               catalog.ItemTypeEpisode,
		ParentID:           &season.ID,
		Path:               "show-one/Season 01/Episode 02",
		SortKey:            "Show One S01E02",
		Title:              "Matched Episode 2",
		IndexNumber:        &episodeNumber,
		ParentIndexNumber:  &seasonNumber,
		AvailabilityStatus: catalog.AvailabilityMissing,
		GovernanceStatus:   catalog.GovernanceMatched,
	})
	if err != nil {
		t.Fatalf("create provider episode: %v", err)
	}
	if _, err := catalogSvc.SetExternalID(ctx, catalog.ExternalIDInput{ItemID: episode.ID, Provider: "tmdb", ProviderType: "tv_episode", ExternalID: "tv:1002", IsPrimary: true}); err != nil {
		t.Fatalf("seed provider identity: %v", err)
	}
	if _, err := catalogSvc.RecordMetadataSource(ctx, catalog.MetadataSourceInput{ItemID: episode.ID, SourceType: catalog.SourceTypeProvider, SourceName: "tmdb", ExternalID: "tv:1002", PayloadJSON: `{"matched_title":"Matched Episode 2"}`}); err != nil {
		t.Fatalf("seed provider evidence: %v", err)
	}

	modifiedAt := time.Date(2026, 4, 25, 13, 0, 0, 0, time.UTC)
	result, err := svc.writeCatalogScanEpisodeHierarchy(ctx, libraryRecord, catalogScanArtifact{
		ItemType:          catalog.ItemTypeEpisode,
		SourcePath:        "/library/Show One/Season 1/Show One.S01E02.mkv",
		SeriesPath:        "show-one",
		SeasonPath:        "show-one/season-01",
		Title:             "Show One S01E02",
		OriginalTitle:     "Show.One.S01E02",
		SeriesTitle:       "Show One",
		SeasonNumber:      &seasonNumber,
		EpisodeSlots:      []catalogEpisodeSlot{{EpisodeNumber: 2, ItemPath: "show-one/season-01/episode-0002"}},
		StorageProvider:   "local",
		StableIdentityKey: "local:show-one-s01e02",
		ProviderName:      "scanner-provider",
		HashesJSON:        `{"sha1":"deadbeef"}`,
		SizeBytes:         8192,
		ModifiedAt:        &modifiedAt,
		Container:         "mkv",
	})
	if err != nil {
		t.Fatalf("write scan hierarchy: %v", err)
	}

	assertCatalogCounts(t, ctx, db, 3, 1, 1, 1, 1, 3)
	if result.Item.ID != episode.ID {
		t.Fatalf("expected scanner to reuse provider-created episode %d, got %#v", episode.ID, result.Item)
	}

	var reloadedSeason database.CatalogItem
	if err := db.WithContext(ctx).First(&reloadedSeason, season.ID).Error; err != nil {
		t.Fatalf("reload season: %v", err)
	}
	if reloadedSeason.Path != "show-one/season-01" || reloadedSeason.Title != "Season 1" || reloadedSeason.GovernanceStatus != catalog.GovernanceMatched {
		t.Fatalf("expected season path rewrite without governance loss, got %#v", reloadedSeason)
	}

	var reloadedEpisode database.CatalogItem
	if err := db.WithContext(ctx).First(&reloadedEpisode, episode.ID).Error; err != nil {
		t.Fatalf("reload episode: %v", err)
	}
	if reloadedEpisode.Path != "show-one/season-01/episode-0002" || reloadedEpisode.Title != "Matched Episode 2" || reloadedEpisode.AvailabilityStatus != catalog.AvailabilityAvailable || reloadedEpisode.GovernanceStatus != catalog.GovernanceMatched {
		t.Fatalf("expected reused episode to become available while preserving governance, got %#v", reloadedEpisode)
	}

	var externalID database.CatalogExternalID
	if err := db.WithContext(ctx).Where("item_id = ? AND provider = ? AND provider_type = ?", episode.ID, "tmdb", "tv_episode").First(&externalID).Error; err != nil {
		t.Fatalf("reload external id: %v", err)
	}
	if externalID.ExternalID != "tv:1002" {
		t.Fatalf("expected descendant identity to survive scanner reuse, got %#v", externalID)
	}
}

func newScanCatalogWriterHarness(t *testing.T) (context.Context, *gorm.DB, *Service, database.Library) {
	t.Helper()

	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	rootPath := "/library"
	registry := providers.NewRegistry(config.Config{Local: config.LocalStorageConfig{RootPath: rootPath}})
	svc := NewService(config.Config{Local: config.LocalStorageConfig{RootPath: rootPath}}, db, registry, nil)
	libraryRecord := database.Library{Name: "Library", Type: "shows", RootPath: rootPath, Status: "active"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}

	return ctx, db, svc, libraryRecord
}

func selectedImagesByType(t *testing.T, ctx context.Context, db *gorm.DB, itemID uint) map[string]database.ItemImage {
	t.Helper()
	var images []database.ItemImage
	if err := db.WithContext(ctx).Where("item_id = ?", itemID).Order("image_type asc, id asc").Find(&images).Error; err != nil {
		t.Fatalf("load item images: %v", err)
	}
	selected := make(map[string]database.ItemImage, len(images))
	for _, image := range images {
		if image.IsSelected {
			selected[image.ImageType] = image
		}
	}
	return selected
}

func newDirectScanHarness(t *testing.T, rootPath string) (context.Context, *gorm.DB, *Service) {
	t.Helper()

	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{Local: config.LocalStorageConfig{RootPath: rootPath}}
	registry := providers.NewRegistry(cfg)
	svc := NewService(cfg, db, registry, nil)

	return ctx, db, svc
}

func createDirectScanLibrary(t *testing.T, ctx context.Context, svc *Service, name string, libraryType string, rootPath string) database.Library {
	t.Helper()

	source, err := svc.CreateMediaSource(ctx, CreateMediaSourceInput{Provider: "local", Name: fmt.Sprintf("%s Source", name), RootPath: rootPath})
	if err != nil {
		t.Fatalf("create media source %s: %v", name, err)
	}
	libraryRecord, _, err := svc.CreateLibrary(ctx, CreateLibraryInput{Name: name, Type: libraryType, MediaSourceID: source.ID, RootPath: rootPath})
	if err != nil {
		t.Fatalf("create library %s: %v", name, err)
	}
	return libraryRecord
}

func newSyncLibraryJobPayload(libraryID uint, rootPath string) database.Job {
	payloadJSON, err := json.Marshal(map[string]any{"library_id": libraryID, "root_path": rootPath})
	if err != nil {
		panic(err)
	}
	return database.Job{PayloadJSON: string(payloadJSON)}
}

func mustWriteFixtureFile(t *testing.T, filePath string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir fixture dir %s: %v", filepath.Dir(filePath), err)
	}
	if err := os.WriteFile(filePath, []byte("fixture"), 0o644); err != nil {
		t.Fatalf("write fixture file %s: %v", filePath, err)
	}
}

func mustWriteFixtureTextFile(t *testing.T, filePath string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir fixture dir %s: %v", filepath.Dir(filePath), err)
	}
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture file %s: %v", filePath, err)
	}
}

func scannerEvidencePayloadForSingleSource(t *testing.T, ctx context.Context, db *gorm.DB) map[string]any {
	t.Helper()

	var source database.MetadataSource
	if err := db.WithContext(ctx).First(&source).Error; err != nil {
		t.Fatalf("load metadata source: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(source.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode evidence payload: %v", err)
	}
	return payload
}

func scannerEvidencePayloadForItem(t *testing.T, ctx context.Context, db *gorm.DB, itemID uint) map[string]any {
	t.Helper()

	var source database.MetadataSource
	if err := db.WithContext(ctx).Where("item_id = ?", itemID).First(&source).Error; err != nil {
		t.Fatalf("load metadata source: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(source.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode evidence payload: %v", err)
	}
	return payload
}

func metadataSidecarsFromPayload(t *testing.T, payload map[string]any) []map[string]any {
	t.Helper()

	rawItems, ok := payload["metadata_sidecars"].([]any)
	if !ok {
		return nil
	}
	items := make([]map[string]any, 0, len(rawItems))
	for _, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("unexpected metadata sidecar payload: %#v", raw)
		}
		items = append(items, item)
	}
	return items
}

func assertCatalogCounts(t *testing.T, ctx context.Context, db *gorm.DB, itemCount int64, fileCount int64, assetCount int64, assetItemCount int64, assetFileCount int64, metadataSourceCount int64) {
	t.Helper()

	assertTableCount(t, ctx, db, &database.CatalogItem{}, itemCount)
	assertTableCount(t, ctx, db, &database.InventoryFile{}, fileCount)
	assertTableCount(t, ctx, db, &database.MediaAsset{}, assetCount)
	assertTableCount(t, ctx, db, &database.AssetItem{}, assetItemCount)
	assertTableCount(t, ctx, db, &database.AssetFile{}, assetFileCount)
	assertTableCount(t, ctx, db, &database.MetadataSource{}, metadataSourceCount)
}

func assertTableCount(t *testing.T, ctx context.Context, db *gorm.DB, model any, expected int64) {
	t.Helper()

	var actual int64
	if err := db.WithContext(ctx).Model(model).Count(&actual).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	if actual != expected {
		t.Fatalf("expected %d rows for %T, got %d", expected, model, actual)
	}
}

func assertEvidencePayloadKeys(t *testing.T, payloadJSON string, expectedKeys []string) {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("decode payload json: %v", err)
	}
	if len(payload) != len(expectedKeys) {
		t.Fatalf("expected only allowlisted payload keys %v, got %#v", expectedKeys, payload)
	}
	for _, key := range expectedKeys {
		if _, ok := payload[key]; !ok {
			t.Fatalf("expected payload key %q, got %#v", key, payload)
		}
	}
}
