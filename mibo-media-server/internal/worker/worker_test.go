package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/listener"
	"github.com/atlan/mibo-media-server/internal/metadata"
	"github.com/atlan/mibo-media-server/internal/probe"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/schedule"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
)

func TestRunOnceProcessesSyncLibraryJob(t *testing.T) {
	listRequests := make(map[string]bool)
	openList := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var body struct {
			Path    string `json:"path"`
			Refresh bool   `json:"refresh"`
		}
		_ = json.NewDecoder(req.Body).Decode(&body)

		switch req.URL.Path {
		case "/api/fs/get":
			isDir := !strings.HasSuffix(body.Path, ".mkv")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    http.StatusOK,
				"message": "success",
				"data": map[string]any{
					"name":   pathBase(body.Path),
					"is_dir": isDir,
					"size":   0,
				},
			})
		case "/api/fs/list":
			listRequests[body.Path] = body.Refresh
			content := []map[string]any{}
			switch body.Path {
			case "/movies":
				content = []map[string]any{
					{"name": "MovieA.2024.mkv", "is_dir": false, "size": 1024},
					{"name": "ShowB", "is_dir": true, "size": 0},
				}
			case "/movies/ShowB":
				content = []map[string]any{
					{"name": "ShowB.S01E02.mkv", "is_dir": false, "size": 2048},
					{"name": "poster.jpg", "is_dir": false, "size": 512},
				}
			}

			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    http.StatusOK,
				"message": "success",
				"data": map[string]any{
					"content": content,
				},
			})
		case "/api/fs/link":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    http.StatusOK,
				"message": "success",
				"data":    map[string]any{"url": "https://media.example.test" + body.Path},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer openList.Close()

	tmdb := newTMDBTestServer()
	defer tmdb.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	ffprobePath := writeFakeFFprobe(t)
	cfg := config.Config{
		OpenList: config.OpenListConfig{BaseURL: openList.URL, RootPath: "/movies"},
		Metadata: config.MetadataConfig{TMDB: config.TMDBConfig{APIKey: "test-key", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}},
		FFprobe:  config.FFprobeConfig{Enabled: true, Path: ffprobePath, Timeout: 2 * time.Second},
		Worker:   config.WorkerConfig{Enabled: true, PollInterval: time.Millisecond},
	}

	registry := providers.NewRegistry(cfg)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	metadataSvc := metadata.NewService(db, cfg.Metadata, settingsSvc)
	probeSvc := probe.NewService(db, registry, cfg.FFprobe)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "openlist", Name: "Home Media", RootPath: "/movies"})
	if err != nil {
		t.Fatalf("create media source: %v", err)
	}

	libraryRecord, job, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: "/movies"})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}

	runner := NewRunner(cfg.Worker, jobsSvc, librarySvc, metadataSvc, probeSvc, settingsSvc)
	runner.RunOnce(ctx)

	var storedLibrary database.Library
	if err := db.WithContext(ctx).First(&storedLibrary, libraryRecord.ID).Error; err != nil {
		t.Fatalf("reload library: %v", err)
	}
	if storedLibrary.Status != "active" {
		t.Fatalf("expected library status active, got %q", storedLibrary.Status)
	}

	var storedJob database.Job
	if err := db.WithContext(ctx).First(&storedJob, job.ID).Error; err != nil {
		t.Fatalf("reload job: %v", err)
	}
	if storedJob.Status != jobs.StatusCompleted {
		t.Fatalf("expected job status completed, got %q", storedJob.Status)
	}

	var catalogCount int64
	if err := db.WithContext(ctx).Model(&database.CatalogItem{}).Where("library_id = ? AND deleted_at IS NULL", libraryRecord.ID).Count(&catalogCount).Error; err != nil {
		t.Fatalf("count catalog items: %v", err)
	}
	if catalogCount != 4 {
		t.Fatalf("expected 4 catalog items (movie + series/season/episode), got %d", catalogCount)
	}

	var externalIDCount int64
	if err := db.WithContext(ctx).Model(&database.CatalogExternalID{}).Count(&externalIDCount).Error; err != nil {
		t.Fatalf("count catalog external ids: %v", err)
	}
	if externalIDCount == 0 {
		t.Fatalf("expected auto catalog match to persist external ids after scan")
	}

	var selectedImageCount int64
	if err := db.WithContext(ctx).Model(&database.ItemImage{}).Where("is_selected = ?", true).Count(&selectedImageCount).Error; err != nil {
		t.Fatalf("count selected catalog images: %v", err)
	}
	if selectedImageCount == 0 {
		t.Fatalf("expected auto catalog match to persist selected artwork after scan")
	}

	var legacyItemCount int64
	if err := db.WithContext(ctx).Model(&database.MediaItem{}).Where("library_id = ? AND deleted_at IS NULL", libraryRecord.ID).Count(&legacyItemCount).Error; err != nil {
		t.Fatalf("count legacy media items: %v", err)
	}
	if legacyItemCount != 0 {
		t.Fatalf("expected catalog-first scan to leave legacy media_items empty, got %d", legacyItemCount)
	}

	var legacyFileCount int64
	if err := db.WithContext(ctx).Model(&database.MediaFile{}).Where("library_id = ? AND deleted_at IS NULL", libraryRecord.ID).Count(&legacyFileCount).Error; err != nil {
		t.Fatalf("count legacy media files: %v", err)
	}
	if legacyFileCount != 0 {
		t.Fatalf("expected catalog-first scan to leave legacy media_files empty, got %d", legacyFileCount)
	}

	var files []database.InventoryFile
	if err := db.WithContext(ctx).Where("library_id = ? AND deleted_at IS NULL", libraryRecord.ID).Order("storage_path asc").Find(&files).Error; err != nil {
		t.Fatalf("list inventory files: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 inventory files, got %d", len(files))
	}

	var assets []database.MediaAsset
	if err := db.WithContext(ctx).Where("library_id = ? AND deleted_at IS NULL", libraryRecord.ID).Order("id asc").Find(&assets).Error; err != nil {
		t.Fatalf("list media assets: %v", err)
	}
	if len(assets) != 2 {
		t.Fatalf("expected 2 media assets, got %d", len(assets))
	}
	for _, asset := range assets {
		if asset.ProbeStatus != probe.StatusReady || asset.DurationSeconds == nil || *asset.DurationSeconds <= 0 {
			t.Fatalf("expected inventory probe enrichment on asset, got %#v", asset)
		}
	}
	if !listRequests["/movies"] || !listRequests["/movies/ShowB"] {
		t.Fatalf("expected scan to refresh openlist directory cache, got %#v", listRequests)
	}
}

func TestTargetedRefreshQueuesUniqueJobs(t *testing.T) {
	t.Parallel()

	mediaRoot := filepath.Join(t.TempDir(), "media-root")
	showDir := filepath.Join(mediaRoot, "Movies", "ShowB")
	if err := os.MkdirAll(showDir, 0o755); err != nil {
		t.Fatalf("create media dirs: %v", err)
	}

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{Local: config.LocalStorageConfig{RootPath: mediaRoot}}
	registry := providers.NewRegistry(cfg)
	jobsSvc := jobs.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)

	ctx := context.Background()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Local", RootPath: mediaRoot})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	record, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: filepath.Join(mediaRoot, "Movies")})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}

	first, err := librarySvc.QueueTargetedRefresh(ctx, record.ID, showDir, "storage_event")
	if err != nil {
		t.Fatalf("queue first targeted refresh: %v", err)
	}
	second, err := librarySvc.QueueTargetedRefresh(ctx, record.ID, showDir, "storage_event")
	if err != nil {
		t.Fatalf("queue duplicate targeted refresh: %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("expected duplicate targeted refresh to return same job, got %d and %d", first.ID, second.ID)
	}
	if first.Kind != "targeted_refresh" {
		t.Fatalf("expected targeted_refresh job kind, got %q", first.Kind)
	}
}

func TestRunOnceProcessesMetadataRefetchJob(t *testing.T) {
	t.Skip("legacy metadata refetch job removed after full catalog cutover")
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/movie/101":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             101,
				"title":          "MovieA Refetched",
				"original_title": "MovieA Refetched",
				"overview":       "Fresh metadata overview",
				"poster_path":    "/movie-a-refetched.jpg",
				"backdrop_path":  "/movie-a-refetched-bg.jpg",
				"release_date":   "2024-02-02",
				"runtime":        126,
				"genres":         []map[string]any{{"name": "Action"}},
				"credits": map[string]any{
					"cast": []map[string]any{{"name": "Actor A"}},
					"crew": []map[string]any{{"name": "Director A", "job": "Director", "department": "Directing"}},
				},
				"images": map[string]any{"logos": []map[string]any{}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer tmdb.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{Worker: config.WorkerConfig{Enabled: true, PollInterval: time.Millisecond}}
	registry := providers.NewRegistry(cfg)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, config.MetadataConfig{TMDB: config.TMDBConfig{BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}})
	if _, err := settingsSvc.UpdateMetadataSettings(context.Background(), settings.UpdateMetadataSettingsInput{
		TMDB: settings.MetadataProviderInput{
			APIKey:       "refetch-key",
			BaseURL:      tmdb.URL,
			ImageBaseURL: tmdb.URL + "/images",
			Language:     "en-US",
			Timeout:      "1s",
		},
	}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}

	confidence := 0.93
	item := database.MediaItem{
		LibraryID:          1,
		Type:               "movie",
		Title:              "MovieA Stale",
		SourcePath:         "/movies/MovieA.2024.mkv",
		MatchStatus:        metadata.StatusMatched,
		MetadataProvider:   "tmdb",
		ExternalID:         "movie:101",
		MetadataConfidence: &confidence,
		Status:             "ready",
	}
	if err := db.WithContext(context.Background()).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	job, err := jobsSvc.Enqueue(context.Background(), library.JobKindRefetchMediaItem, map[string]any{"media_item_id": item.ID})
	if err != nil {
		t.Fatalf("enqueue refetch job: %v", err)
	}

	runner := NewRunner(cfg.Worker, jobsSvc, library.NewService(cfg, db, registry, jobsSvc), metadata.NewService(db, config.MetadataConfig{}, settingsSvc), probe.NewService(db, registry, config.FFprobeConfig{}), settingsSvc)
	runner.RunOnce(context.Background())

	var storedJob database.Job
	if err := db.WithContext(context.Background()).First(&storedJob, job.ID).Error; err != nil {
		t.Fatalf("reload job: %v", err)
	}
	if storedJob.Status != jobs.StatusCompleted {
		t.Fatalf("expected refetch job completed, got %q", storedJob.Status)
	}

	var stored database.MediaItem
	if err := db.WithContext(context.Background()).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if stored.Title != "MovieA Refetched" || stored.Overview != "Fresh metadata overview" {
		t.Fatalf("unexpected refetched media item: %#v", stored)
	}
}

func TestRunOnceProcessesSearchReindexJobs(t *testing.T) {
	t.Parallel()

	mediaRoot := filepath.Join(t.TempDir(), "media-root")
	if err := os.MkdirAll(mediaRoot, 0o755); err != nil {
		t.Fatalf("create media root: %v", err)
	}

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{Local: config.LocalStorageConfig{RootPath: mediaRoot}, Worker: config.WorkerConfig{Enabled: true, PollInterval: time.Millisecond}}
	registry := providers.NewRegistry(cfg)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	searchSvc := search.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	metadataSvc := metadata.NewService(db, cfg.Metadata, settingsSvc, searchSvc)
	probeSvc := probe.NewService(db, registry, cfg.FFprobe)
	runner := NewRunner(cfg.Worker, jobsSvc, librarySvc, metadataSvc, probeSvc, settingsSvc, searchSvc)

	ctx := context.Background()
	source := database.MediaSource{Name: "Local", Provider: "local", StorageRef: "local", RootPath: mediaRoot}
	if err := db.WithContext(ctx).Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	record := database.Library{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: mediaRoot, Status: "active", ScannerEnabled: true}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	rating := 8.4
	year := 2024
	items := []database.MediaItem{
		{LibraryID: record.ID, Type: "movie", Title: "MovieA", CastJSON: `[{"name":"Actor A"}]`, GenresJSON: `["Action"]`, RegionsJSON: `["Japan"]`, VoteAverage: &rating, Year: &year, SourcePath: filepath.Join(mediaRoot, "MovieA.2024.mkv"), MatchStatus: metadata.StatusMatched, Status: "ready"},
		{LibraryID: record.ID, Type: "movie", Title: "MovieB", CastJSON: `[{"name":"Actor B"}]`, GenresJSON: `["Drama"]`, RegionsJSON: `["United States"]`, VoteAverage: &rating, Year: &year, SourcePath: filepath.Join(mediaRoot, "MovieB.2024.mkv"), MatchStatus: metadata.StatusMatched, Status: "ready"},
	}
	for idx := range items {
		if err := db.WithContext(ctx).Create(&items[idx]).Error; err != nil {
			t.Fatalf("create media item %d: %v", idx, err)
		}
	}

	documentJob, err := jobsSvc.Enqueue(ctx, library.JobKindReindexSearchDocument, map[string]any{"media_item_id": items[0].ID})
	if err != nil {
		t.Fatalf("enqueue search document reindex: %v", err)
	}
	libraryJob, err := jobsSvc.Enqueue(ctx, library.JobKindReindexLibrarySearch, map[string]any{"library_id": record.ID, "root_path": mediaRoot})
	if err != nil {
		t.Fatalf("enqueue library search reindex: %v", err)
	}

	runner.RunOnce(ctx)

	for _, jobID := range []uint{documentJob.ID, libraryJob.ID} {
		var job database.Job
		if err := db.WithContext(ctx).First(&job, jobID).Error; err != nil {
			t.Fatalf("reload job %d: %v", jobID, err)
		}
		if job.Status != jobs.StatusCompleted {
			t.Fatalf("expected job %d completed, got %q", jobID, job.Status)
		}
	}

	var docs []database.SearchDocument
	if err := db.WithContext(ctx).Order("media_item_id asc").Find(&docs).Error; err != nil {
		t.Fatalf("load search documents: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 search documents, got %#v", docs)
	}
	if docs[0].MediaItemID != items[0].ID || docs[1].MediaItemID != items[1].ID {
		t.Fatalf("unexpected search documents: %#v", docs)
	}
	if docs[0].VoteAverage == nil || *docs[0].VoteAverage != rating || !strings.Contains(docs[0].SearchCountriesText, "Japan") {
		t.Fatalf("unexpected first search document: %#v", docs[0])
	}
}

func TestRunOnceClaimsDueSchedulesAndUpdatesRunHistory(t *testing.T) {
	t.Skip("legacy metadata schedule kinds removed after full catalog cutover")
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.URL.Path != "/movie/101" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":             101,
			"title":          "Artwork Updated",
			"original_title": "Artwork Updated",
			"overview":       "Existing overview",
			"poster_path":    "/poster.jpg",
			"backdrop_path":  "/backdrop.jpg",
			"release_date":   "2024-02-02",
			"runtime":        120,
			"genres":         []map[string]any{{"name": "Action"}},
			"credits":        map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}},
			"images":         map[string]any{"logos": []map[string]any{{"file_path": "/logo.png", "iso_639_1": "en", "vote_average": 9.0}}},
		})
	}))
	defer tmdb.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	cfg := config.Config{Worker: config.WorkerConfig{Enabled: true, PollInterval: time.Millisecond}}
	registry := providers.NewRegistry(cfg)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, config.MetadataConfig{TMDB: config.TMDBConfig{BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}})
	if _, err := settingsSvc.UpdateMetadataSettings(context.Background(), settings.UpdateMetadataSettingsInput{TMDB: settings.MetadataProviderInput{APIKey: "worker-key", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: "1s"}}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}
	searchSvc := search.NewService(db)
	metadataSvc := metadata.NewService(db, cfg.Metadata, settingsSvc, searchSvc)
	scheduleSvc := schedule.NewService(db, schedule.WithJobs(jobsSvc))
	runner := NewRunner(cfg.Worker, jobsSvc, library.NewService(cfg, db, registry, jobsSvc), metadataSvc, probe.NewService(db, registry, config.FFprobeConfig{}), settingsSvc, searchSvc, scheduleSvc)

	confidence := 0.9
	item := database.MediaItem{LibraryID: 1, Type: "movie", Title: "Artwork stale", Overview: "Existing overview", SourcePath: "/library/movie.mkv", MatchStatus: metadata.StatusMatched, MetadataProvider: "tmdb", ExternalID: "movie:101", MetadataConfidence: &confidence, Status: "ready"}
	if err := db.WithContext(context.Background()).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	created, err := scheduleSvc.Create(context.Background(), schedule.CreateScheduleInput{Name: "Cleanup", Kind: schedule.KindLibraryCleanup, ScopeKind: schedule.ScopeGlobal, Enabled: true, Frequency: schedule.FrequencySpec{Kind: schedule.FrequencyDaily, TimeOfDay: "09:00"}})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	if err := db.WithContext(context.Background()).Model(&database.Schedule{}).Where("id = ?", created.ID).Update("next_run_at", past).Error; err != nil {
		t.Fatalf("set due next_run_at: %v", err)
	}

	runner.RunOnce(context.Background())

	var storedSchedule database.Schedule
	if err := db.WithContext(context.Background()).First(&storedSchedule, created.ID).Error; err != nil {
		t.Fatalf("reload schedule: %v", err)
	}
	if storedSchedule.LatestRunStatus != schedule.StatusCompleted {
		t.Fatalf("expected completed schedule status, got %q", storedSchedule.LatestRunStatus)
	}
	if storedSchedule.LatestJobID == nil {
		t.Fatalf("expected latest job id to be set")
	}
	var runs []database.ScheduleRun
	if err := db.WithContext(context.Background()).Where("schedule_id = ?", created.ID).Find(&runs).Error; err != nil {
		t.Fatalf("list schedule runs: %v", err)
	}
	if len(runs) != 1 || runs[0].Status != schedule.StatusCompleted || runs[0].StartedAt == nil || runs[0].FinishedAt == nil {
		t.Fatalf("expected completed run history, got %#v", runs)
	}
	var storedItem database.MediaItem
	if err := db.WithContext(context.Background()).First(&storedItem, item.ID).Error; err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if storedItem.PosterURL == "" || storedItem.LogoURL == "" {
		t.Fatalf("expected artwork refresh to update fields, got %#v", storedItem)
	}
}

func TestRunOnceIgnoresLegacyRefreshIntervalWithoutSchedules(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	cfg := config.Config{Worker: config.WorkerConfig{Enabled: true, PollInterval: time.Millisecond, RefreshIntervalHours: 24}}
	registry := providers.NewRegistry(cfg)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	searchSvc := search.NewService(db)
	runner := NewRunner(cfg.Worker, jobsSvc, library.NewService(cfg, db, registry, jobsSvc), metadata.NewService(db, cfg.Metadata, settingsSvc, searchSvc), probe.NewService(db, registry, config.FFprobeConfig{}), settingsSvc, searchSvc, schedule.NewService(db, schedule.WithJobs(jobsSvc)))

	runner.RunOnce(context.Background())

	var count int64
	if err := db.WithContext(context.Background()).Model(&database.Job{}).Count(&count).Error; err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no legacy auto-scan jobs without schedules, got %d", count)
	}
}

func TestRunOnceGeneratesFallbackArtworkWhenMetadataMissing(t *testing.T) {
	t.Skip("legacy fallback artwork generation removed after full catalog cutover")
	mediaRoot := filepath.Join(t.TempDir(), "media-root")
	if err := os.MkdirAll(mediaRoot, 0o755); err != nil {
		t.Fatalf("create media root: %v", err)
	}
	moviePath := filepath.Join(mediaRoot, "MovieA.2024.mkv")
	if err := os.WriteFile(moviePath, []byte("movie"), 0o644); err != nil {
		t.Fatalf("write movie file: %v", err)
	}

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	artworkRoot := filepath.Join(t.TempDir(), "artwork")
	ffprobePath := writeFakeFFprobe(t)
	ffmpegPath := writeFakeFFmpeg(t)
	cfg := config.Config{
		Local:   config.LocalStorageConfig{RootPath: mediaRoot},
		FFprobe: config.FFprobeConfig{Enabled: true, Path: ffprobePath, Timeout: 2 * time.Second},
		FFmpeg:  config.FFmpegConfig{Enabled: true, Path: ffmpegPath, Timeout: 2 * time.Second, ArtworkRootPath: artworkRoot},
		Worker:  config.WorkerConfig{Enabled: true, PollInterval: time.Millisecond},
	}
	registry := providers.NewRegistry(cfg)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	probeSvc := probe.NewService(db, registry, cfg.FFprobe, cfg.FFmpeg)
	runner := NewRunner(cfg.Worker, jobsSvc, librarySvc, metadata.NewService(db, cfg.Metadata, settingsSvc), probeSvc, settingsSvc)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Local", RootPath: mediaRoot})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	record, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: mediaRoot})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}

	item := database.MediaItem{LibraryID: record.ID, Type: "movie", Title: "MovieA", SourcePath: moviePath, MatchStatus: metadata.StatusSkipped, Status: "ready"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create media item: %v", err)
	}
	file := database.MediaFile{LibraryID: record.ID, MediaItemID: &item.ID, StoragePath: moviePath, Container: "mkv", ProbeStatus: probe.StatusPending}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create media file: %v", err)
	}
	if _, err := jobsSvc.Enqueue(ctx, "probe_media_file", map[string]any{"media_file_id": file.ID}); err != nil {
		t.Fatalf("enqueue legacy probe job: %v", err)
	}

	runner.RunOnce(ctx)

	var storedItem database.MediaItem
	if err := db.WithContext(ctx).First(&storedItem, item.ID).Error; err != nil {
		t.Fatalf("load media item: %v", err)
	}
	wantPosterURL := fmt.Sprintf("/api/v1/media-items/%d/artwork/poster", storedItem.ID)
	wantBackdropURL := fmt.Sprintf("/api/v1/media-items/%d/artwork/backdrop", storedItem.ID)
	if storedItem.PosterURL != wantPosterURL || storedItem.BackdropURL != wantBackdropURL {
		t.Fatalf("expected generated artwork urls, got poster=%q backdrop=%q", storedItem.PosterURL, storedItem.BackdropURL)
	}
	for _, name := range []string{"poster.jpg", "backdrop.jpg"} {
		if _, err := os.Stat(filepath.Join(artworkRoot, fmt.Sprintf("%d", storedItem.ID), name)); err != nil {
			t.Fatalf("expected generated artwork %s: %v", name, err)
		}
	}
}

func TestPartialSyncDoesNotSoftDeleteUnseenLibraryRows(t *testing.T) {
	t.Parallel()

	mediaRoot := filepath.Join(t.TempDir(), "media-root")
	movieDir := filepath.Join(mediaRoot, "Movies")
	showDir := filepath.Join(movieDir, "ShowB")
	if err := os.MkdirAll(showDir, 0o755); err != nil {
		t.Fatalf("create media dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(movieDir, "MovieA.2024.mkv"), []byte("movie"), 0o644); err != nil {
		t.Fatalf("write movie file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(showDir, "ShowB.S01E02.mkv"), []byte("episode"), 0o644); err != nil {
		t.Fatalf("write episode file: %v", err)
	}

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	ffprobePath := writeFakeFFprobe(t)
	cfg := config.Config{
		Local:   config.LocalStorageConfig{RootPath: mediaRoot},
		FFprobe: config.FFprobeConfig{Enabled: true, Path: ffprobePath, Timeout: 2 * time.Second},
		Worker:  config.WorkerConfig{Enabled: true, PollInterval: time.Millisecond},
	}
	registry := providers.NewRegistry(cfg)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	runner := NewRunner(cfg.Worker, jobsSvc, librarySvc, metadata.NewService(db, cfg.Metadata, settingsSvc), probe.NewService(db, registry, cfg.FFprobe), settingsSvc)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Local", RootPath: mediaRoot})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	record, initialJob, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: movieDir})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	runner.RunOnce(ctx)

	if err := os.WriteFile(filepath.Join(showDir, "ShowB.S01E03.mkv"), []byte("episode-2"), 0o644); err != nil {
		t.Fatalf("write new episode: %v", err)
	}
	targetedJob, err := librarySvc.QueueTargetedRefresh(ctx, record.ID, showDir, "storage_event")
	if err != nil {
		t.Fatalf("queue targeted refresh: %v", err)
	}
	runner.RunOnce(ctx)

	var movieFile database.InventoryFile
	if err := db.WithContext(ctx).Where("library_id = ? AND storage_path = ?", record.ID, filepath.Join(movieDir, "MovieA.2024.mkv")).First(&movieFile).Error; err != nil {
		t.Fatalf("load movie inventory file: %v", err)
	}
	if movieFile.DeletedAt != nil {
		t.Fatalf("expected unrelated movie row to stay active after partial sync")
	}

	var jobRecords []database.Job
	if err := db.WithContext(ctx).Where("id IN ?", []uint{initialJob.ID, targetedJob.ID}).Order("id asc").Find(&jobRecords).Error; err != nil {
		t.Fatalf("load jobs: %v", err)
	}
	if len(jobRecords) != 2 || jobRecords[1].Kind != "targeted_refresh" {
		t.Fatalf("expected full sync then targeted refresh jobs, got %#v", jobRecords)
	}
}

func TestRunOnceProcessesTargetedRefreshJob(t *testing.T) {
	t.Parallel()

	mediaRoot := filepath.Join(t.TempDir(), "media-root")
	movieDir := filepath.Join(mediaRoot, "Movies")
	showDir := filepath.Join(movieDir, "ShowB")
	if err := os.MkdirAll(showDir, 0o755); err != nil {
		t.Fatalf("create media dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(movieDir, "MovieA.2024.mkv"), []byte("movie"), 0o644); err != nil {
		t.Fatalf("write movie file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(showDir, "ShowB.S01E02.mkv"), []byte("episode"), 0o644); err != nil {
		t.Fatalf("write episode file: %v", err)
	}

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	ffprobePath := writeFakeFFprobe(t)
	cfg := config.Config{
		Local:   config.LocalStorageConfig{RootPath: mediaRoot},
		FFprobe: config.FFprobeConfig{Enabled: true, Path: ffprobePath, Timeout: 2 * time.Second},
		Worker:  config.WorkerConfig{Enabled: true, PollInterval: time.Millisecond},
	}
	registry := providers.NewRegistry(cfg)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	runner := NewRunner(cfg.Worker, jobsSvc, librarySvc, metadata.NewService(db, cfg.Metadata, settingsSvc), probe.NewService(db, registry, cfg.FFprobe), settingsSvc)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Local", RootPath: mediaRoot})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	record, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: movieDir})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	runner.RunOnce(ctx)

	if err := os.WriteFile(filepath.Join(showDir, "ShowB.S01E03.mkv"), []byte("episode-2"), 0o644); err != nil {
		t.Fatalf("write new episode: %v", err)
	}
	targetedJob, err := librarySvc.QueueTargetedRefresh(ctx, record.ID, showDir, "storage_event")
	if err != nil {
		t.Fatalf("queue targeted refresh: %v", err)
	}
	runner.RunOnce(ctx)

	var storedJob database.Job
	if err := db.WithContext(ctx).First(&storedJob, targetedJob.ID).Error; err != nil {
		t.Fatalf("reload targeted job: %v", err)
	}
	if storedJob.Status != jobs.StatusCompleted {
		t.Fatalf("expected targeted refresh job completed, got %q", storedJob.Status)
	}

	var episodeCount int64
	if err := db.WithContext(ctx).Model(&database.InventoryFile{}).Where("library_id = ? AND deleted_at IS NULL AND storage_path LIKE ?", record.ID, filepath.Join(showDir, "%")).Count(&episodeCount).Error; err != nil {
		t.Fatalf("count episode inventory files: %v", err)
	}
	if episodeCount != 2 {
		t.Fatalf("expected targeted refresh to scan subtree files, got %d", episodeCount)
	}
}

func TestRunOnceProcessesStorageEventRefreshJob(t *testing.T) {
	t.Parallel()

	mediaRoot := filepath.Join(t.TempDir(), "media-root")
	movieDir := filepath.Join(mediaRoot, "Movies")
	showDir := filepath.Join(movieDir, "ShowB")
	if err := os.MkdirAll(showDir, 0o755); err != nil {
		t.Fatalf("create media dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(movieDir, "MovieA.2024.mkv"), []byte("movie"), 0o644); err != nil {
		t.Fatalf("write movie file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(showDir, "ShowB.S01E02.mkv"), []byte("episode"), 0o644); err != nil {
		t.Fatalf("write episode file: %v", err)
	}

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	ffprobePath := writeFakeFFprobe(t)
	cfg := config.Config{
		Local:   config.LocalStorageConfig{RootPath: mediaRoot},
		FFprobe: config.FFprobeConfig{Enabled: true, Path: ffprobePath, Timeout: 2 * time.Second},
		Worker:  config.WorkerConfig{Enabled: true, PollInterval: time.Millisecond},
	}
	registry := providers.NewRegistry(cfg)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	listenerSvc := listener.NewService(db, jobsSvc, librarySvc)
	runner := NewRunner(cfg.Worker, jobsSvc, librarySvc, metadata.NewService(db, cfg.Metadata, settingsSvc), probe.NewService(db, registry, cfg.FFprobe), settingsSvc, listenerSvc)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Local", RootPath: mediaRoot})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	record, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: movieDir})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	runner.RunOnce(ctx)

	targetRoot := filepath.Join(movieDir, "ShowB")
	listenerJob, err := jobsSvc.Enqueue(ctx, listener.JobKindApplyStorageEventRefresh, map[string]any{
		"library_id":         record.ID,
		"root_path":          targetRoot,
		"fallback_full_sync": false,
		"reason":             "storage_event",
		"window_started_at":  time.Now().UTC().Add(-time.Second),
		"window_ends_at":     time.Now().UTC().Add(-time.Millisecond),
	})
	if err != nil {
		t.Fatalf("enqueue listener job: %v", err)
	}

	runner.RunOnce(ctx)

	var storedListenerJob database.Job
	if err := db.WithContext(ctx).First(&storedListenerJob, listenerJob.ID).Error; err != nil {
		t.Fatalf("reload listener job: %v", err)
	}
	if storedListenerJob.Status != jobs.StatusCompleted {
		t.Fatalf("expected listener job completed, got %q", storedListenerJob.Status)
	}

	var queuedJobs []database.Job
	if err := db.WithContext(ctx).Where("kind = ?", library.JobKindTargetedRefresh).Order("id asc").Find(&queuedJobs).Error; err != nil {
		t.Fatalf("list targeted refresh jobs: %v", err)
	}
	if len(queuedJobs) == 0 {
		t.Fatal("expected listener job to enqueue targeted_refresh")
	}
	var payload struct {
		LibraryID uint   `json:"library_id"`
		RootPath  string `json:"root_path"`
	}
	if err := json.Unmarshal([]byte(queuedJobs[len(queuedJobs)-1].PayloadJSON), &payload); err != nil {
		t.Fatalf("decode targeted payload: %v", err)
	}
	if payload.LibraryID != record.ID || payload.RootPath != targetRoot {
		t.Fatalf("unexpected targeted refresh payload: %#v", payload)
	}

	fallbackJob, err := jobsSvc.Enqueue(ctx, listener.JobKindApplyStorageEventRefresh, map[string]any{
		"library_id":         record.ID,
		"root_path":          movieDir,
		"fallback_full_sync": true,
		"reason":             "storage_event",
		"window_started_at":  time.Now().UTC().Add(-time.Second),
		"window_ends_at":     time.Now().UTC().Add(-time.Millisecond),
	})
	if err != nil {
		t.Fatalf("enqueue fallback listener job: %v", err)
	}

	runner.RunOnce(ctx)

	var storedFallbackJob database.Job
	if err := db.WithContext(ctx).First(&storedFallbackJob, fallbackJob.ID).Error; err != nil {
		t.Fatalf("reload fallback listener job: %v", err)
	}
	if storedFallbackJob.Status != jobs.StatusCompleted {
		t.Fatalf("expected fallback listener job completed, got %q", storedFallbackJob.Status)
	}

	var syncJobs []database.Job
	if err := db.WithContext(ctx).Where("kind = ?", library.JobKindSyncLibrary).Order("id asc").Find(&syncJobs).Error; err != nil {
		t.Fatalf("list sync jobs: %v", err)
	}
	if len(syncJobs) < 2 {
		t.Fatalf("expected fallback listener job to enqueue another sync_library, got %#v", syncJobs)
	}
}

func TestRunOnceEnsuresAndReseedsListenerReconcileJobs(t *testing.T) {
	t.Parallel()

	mediaRoot := filepath.Join(t.TempDir(), "media-root")
	movieDir := filepath.Join(mediaRoot, "Movies")
	if err := os.MkdirAll(movieDir, 0o755); err != nil {
		t.Fatalf("create media dir: %v", err)
	}

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{Local: config.LocalStorageConfig{RootPath: mediaRoot}, Worker: config.WorkerConfig{Enabled: true, PollInterval: time.Millisecond}}
	registry := providers.NewRegistry(cfg)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	listenerSvc := listener.NewService(db, jobsSvc, librarySvc)
	runner := NewRunner(cfg.Worker, jobsSvc, librarySvc, metadata.NewService(db, cfg.Metadata, settingsSvc), probe.NewService(db, registry, cfg.FFprobe), settingsSvc, listenerSvc)

	ctx := context.Background()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Local", RootPath: mediaRoot})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	record, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: movieDir})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	if err := db.WithContext(ctx).Model(&database.Library{}).Where("id = ?", record.ID).Update("status", "active").Error; err != nil {
		t.Fatalf("activate library: %v", err)
	}

	runner.RunOnce(ctx)

	var reconcileJobs []database.Job
	if err := db.WithContext(ctx).Where("kind = ?", listener.JobKindListenerReconcile).Order("id asc").Find(&reconcileJobs).Error; err != nil {
		t.Fatalf("list reconcile jobs: %v", err)
	}
	if len(reconcileJobs) != 1 {
		t.Fatalf("expected one listener_reconcile job, got %#v", reconcileJobs)
	}
	if !reconcileJobs[0].AvailableAt.After(time.Now().UTC()) {
		t.Fatalf("expected future-dated reconcile job, got %s", reconcileJobs[0].AvailableAt)
	}

	if err := db.WithContext(ctx).Model(&database.Job{}).Where("id = ?", reconcileJobs[0].ID).Update("available_at", time.Now().UTC().Add(-time.Second)).Error; err != nil {
		t.Fatalf("make reconcile due: %v", err)
	}

	runner.RunOnce(ctx)

	var completed database.Job
	if err := db.WithContext(ctx).First(&completed, reconcileJobs[0].ID).Error; err != nil {
		t.Fatalf("reload original reconcile job: %v", err)
	}
	if completed.Status != jobs.StatusCompleted {
		t.Fatalf("expected reconcile job completed, got %q", completed.Status)
	}

	reconcileJobs = nil
	if err := db.WithContext(ctx).Where("kind = ?", listener.JobKindListenerReconcile).Order("id asc").Find(&reconcileJobs).Error; err != nil {
		t.Fatalf("reload reconcile jobs: %v", err)
	}
	if len(reconcileJobs) != 2 {
		t.Fatalf("expected completed + reseeded reconcile jobs, got %#v", reconcileJobs)
	}
	if reconcileJobs[1].Status != jobs.StatusQueued {
		t.Fatalf("expected reseeded reconcile queued, got %q", reconcileJobs[1].Status)
	}
	if !reconcileJobs[1].AvailableAt.After(time.Now().UTC().Add(5 * time.Hour)) {
		t.Fatalf("expected reseeded reconcile far in future, got %s", reconcileJobs[1].AvailableAt)
	}

	var syncJobs []database.Job
	if err := db.WithContext(ctx).Where("kind = ?", library.JobKindSyncLibrary).Order("id asc").Find(&syncJobs).Error; err != nil {
		t.Fatalf("list sync jobs: %v", err)
	}
	if len(syncJobs) < 2 {
		t.Fatalf("expected reconcile run to queue another sync_library, got %#v", syncJobs)
	}
}

func TestScheduleMarkersIgnoreNonScheduleJobs(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	runner := NewRunner(config.WorkerConfig{}, nil, nil, nil, nil, nil, schedule.NewService(db))
	job := database.Job{
		ID:          7,
		Kind:        listener.JobKindListenerReconcile,
		PayloadJSON: `{"library_id":1,"reason":"listener_reconcile","scheduled_for":"2026-04-24T18:37:14Z"}`,
	}

	var logs bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(originalOutput)

	runner.markScheduleRunRunning(context.Background(), job)
	runner.markScheduleRunFinished(context.Background(), job, schedule.StatusCompleted, "completed")

	if strings.Contains(logs.String(), "mark schedule job") {
		t.Fatalf("expected non-schedule jobs to skip schedule markers, got logs %q", logs.String())
	}
}

func newTMDBTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 101, "title": "MovieA", "original_title": "MovieA", "release_date": "2024-02-02"}}})
		case "/search/tv":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 202, "name": "ShowB", "original_name": "ShowB", "first_air_date": "2021-01-01"}}})
		case "/movie/101":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 101, "title": "MovieA", "original_title": "MovieA", "overview": "Movie overview", "poster_path": "/movie-a.jpg", "backdrop_path": "/movie-a-bg.jpg", "release_date": "2024-02-02", "runtime": 121, "genres": []map[string]any{{"name": "Action"}}, "credits": map[string]any{"cast": []map[string]any{{"name": "Actor A"}}, "crew": []map[string]any{{"name": "Director A", "job": "Director", "department": "Directing"}}}})
		case "/tv/202":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 202, "name": "ShowB", "original_name": "ShowB", "overview": "Show overview", "poster_path": "/show-b.jpg", "backdrop_path": "/show-b-bg.jpg", "first_air_date": "2021-01-01", "episode_run_time": []int{48}, "genres": []map[string]any{{"name": "Drama"}}, "credits": map[string]any{"cast": []map[string]any{{"name": "Actor B"}}, "crew": []map[string]any{{"name": "Director B", "job": "Director", "department": "Directing"}}}})
		case "/tv/202/season/1":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 301, "season_number": 1, "name": "Season 1", "poster_path": "/show-b-s1.jpg", "episodes": []map[string]any{{"id": 1002, "season_number": 1, "episode_number": 2, "name": "Episode 2", "air_date": "2021-01-08", "overview": "Episode overview", "still_path": "/show-b-s1e2.jpg", "runtime": 49}}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func writeFakeFFprobe(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ffprobe")
	content := "#!/bin/sh\ncat <<'EOF'\n{\"streams\":[{\"codec_type\":\"video\",\"codec_name\":\"h264\",\"width\":1920,\"height\":1080},{\"codec_type\":\"audio\",\"codec_name\":\"aac\",\"channels\":2,\"tags\":{\"language\":\"eng\",\"title\":\"Stereo\"}},{\"codec_type\":\"subtitle\",\"codec_name\":\"subrip\",\"tags\":{\"language\":\"eng\",\"title\":\"English\"}}],\"format\":{\"duration\":\"7260.25\",\"bit_rate\":\"5000000\"}}\nEOF\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake ffprobe: %v", err)
	}
	return path
}

func writeFakeFFmpeg(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ffmpeg")
	content := "#!/bin/sh\nout=\"\"\nfor arg in \"$@\"; do\n  out=\"$arg\"\ndone\nmkdir -p \"$(dirname \"$out\")\"\nprintf 'fake-artwork' > \"$out\"\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake ffmpeg: %v", err)
	}
	return path
}

func pathBase(value string) string {
	if value == "" || value == "/" {
		return "root"
	}
	for len(value) > 1 && value[len(value)-1] == '/' {
		value = value[:len(value)-1]
	}
	idx := len(value) - 1
	for idx >= 0 && value[idx] != '/' {
		idx--
	}
	return value[idx+1:]
}
