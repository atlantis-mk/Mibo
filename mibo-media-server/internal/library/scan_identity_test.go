package library

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	openliststorage "github.com/atlan/mibo-media-server/internal/storage/openlist"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

func TestRunSyncLibraryUsesStableIdentityEvidence(t *testing.T) {
	t.Parallel()

	serverState := &identityServerState{}
	openList := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var body struct {
			Path string `json:"path"`
		}
		_ = json.NewDecoder(req.Body).Decode(&body)

		switch req.URL.Path {
		case "/api/fs/get":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    http.StatusOK,
				"message": "success",
				"data": map[string]any{
					"name":     pathBase(body.Path),
					"is_dir":   body.Path == "/library",
					"size":     0,
					"provider": "alist",
				},
			})
		case "/api/fs/list":
			filePath := serverState.currentPath()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    http.StatusOK,
				"message": "success",
				"data": map[string]any{
					"provider": "alist",
					"content": []map[string]any{{
						"name":      pathBase(filePath),
						"is_dir":    false,
						"size":      2048,
						"modified":  time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC),
						"hash_info": map[string]string{"sha256": "abc123"},
					}},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer openList.Close()

	ctx := context.Background()
	db, svc, libraryRecord := newIdentityScanService(t, openList.URL)

	firstJob := database.Job{PayloadJSON: `{"library_id":1,"root_path":"/library"}`}
	if err := svc.RunSyncLibrary(ctx, firstJob); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	serverState.renameTo("/library/Renamed.Movie.2024.mkv")
	secondJob := database.Job{PayloadJSON: `{"library_id":1,"root_path":"/library"}`}
	if err := svc.RunSyncLibrary(ctx, secondJob); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	var files []database.MediaFile
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).Order("id asc").Find(&files).Error; err != nil {
		t.Fatalf("list media files: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one persisted media file identity, got %d", len(files))
	}
	if files[0].StoragePath != "/library/Renamed.Movie.2024.mkv" {
		t.Fatalf("expected storage path to move with stable identity, got %q", files[0].StoragePath)
	}
	if files[0].DeletedAt != nil {
		t.Fatalf("expected stable identity match to stay active, got deleted_at=%v", files[0].DeletedAt)
	}
	if files[0].StableIdentityKey == "" {
		t.Fatalf("expected stable identity key to be persisted")
	}
}

func TestOpenListAdapterPreservesIdentityEvidence(t *testing.T) {
	t.Parallel()

	openList := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/api/fs/get":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    http.StatusOK,
				"message": "success",
				"data": map[string]any{
					"name":      "MovieA.2024.mkv",
					"is_dir":    false,
					"size":      4096,
					"provider":  "115 Cloud",
					"hash_info": map[string]string{"sha1": "deadbeef"},
					"raw_url":   "https://media.example.test/moviea.mkv",
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer openList.Close()

	adapter := openliststorage.New(config.OpenListConfig{BaseURL: openList.URL, Timeout: time.Second})
	object, err := adapter.Get(context.Background(), storage.GetRequest{Path: "/library/MovieA.2024.mkv"})
	if err != nil {
		t.Fatalf("get object: %v", err)
	}

	if object.Provider != "115 Cloud" {
		t.Fatalf("expected provider evidence, got %q", object.Provider)
	}
	if object.StableIdentity != "" {
		t.Fatalf("expected stable identity to stay empty without trustworthy provider object id, got %q", object.StableIdentity)
	}
	if object.HashInfo["sha1"] != "deadbeef" {
		t.Fatalf("expected hash evidence to be preserved, got %#v", object.HashInfo)
	}
}

type identityServerState struct {
	mu       sync.Mutex
	filePath string
}

func (s *identityServerState) currentPath() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.filePath == "" {
		s.filePath = "/library/MovieA.2024.mkv"
	}
	return s.filePath
}

func (s *identityServerState) renameTo(next string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.filePath = next
}

func newIdentityScanService(t *testing.T, openListURL string) (*gorm.DB, *Service, database.Library) {
	t.Helper()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{OpenList: config.OpenListConfig{BaseURL: openListURL, RootPath: "/library", Timeout: time.Second}}
	registry := providers.NewRegistry(cfg)
	jobsSvc := jobs.NewService(db)
	svc := NewService(cfg, db, registry, jobsSvc)

	ctx := context.Background()
	source, err := svc.CreateMediaSource(ctx, CreateMediaSourceInput{Provider: "openlist", Name: "Test Source", RootPath: "/library"})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	libraryRecord, _, err := svc.CreateLibrary(ctx, CreateLibraryInput{Name: "Identity", Type: "movies", MediaSourceID: source.ID, RootPath: "/library"})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}

	return db, svc, libraryRecord
}

func pathBase(value string) string {
	if value == "" || value == "/" {
		return "root"
	}
	return path.Base(value)
}
