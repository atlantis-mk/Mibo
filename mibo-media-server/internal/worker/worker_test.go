package worker

import (
	"context"
	"encoding/json"
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
	"github.com/atlan/mibo-media-server/internal/metadata"
	"github.com/atlan/mibo-media-server/internal/probe"
	"github.com/atlan/mibo-media-server/internal/providers"
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

	items, err := librarySvc.ListMediaItems(ctx, libraryRecord.ID, "", 20)
	if err != nil {
		t.Fatalf("list media items: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 media items, got %d", len(items))
	}
	if items[0].MatchStatus != metadata.StatusMatched || items[0].Overview == "" || items[0].PosterURL == "" {
		t.Fatalf("expected movie metadata enrichment, got %#v", items[0])
	}
	if items[1].MatchStatus != metadata.StatusMatched || items[1].SeriesTitle != "ShowB" {
		t.Fatalf("expected episode metadata enrichment, got %#v", items[1])
	}

	var files []database.MediaFile
	if err := db.WithContext(ctx).Where("library_id = ? AND deleted_at IS NULL", libraryRecord.ID).Order("storage_path asc").Find(&files).Error; err != nil {
		t.Fatalf("list media files: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 media files, got %d", len(files))
	}
	if files[0].ProbeStatus != probe.StatusReady || files[0].VideoCodec != "h264" || files[0].DurationSeconds == nil || *files[0].DurationSeconds <= 0 {
		t.Fatalf("expected probe enrichment, got %#v", files[0])
	}
	if !listRequests["/movies"] || !listRequests["/movies/ShowB"] {
		t.Fatalf("expected scan to refresh openlist directory cache, got %#v", listRequests)
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
