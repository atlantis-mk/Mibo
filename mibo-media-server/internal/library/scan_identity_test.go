package library

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/storage"
	openliststorage "github.com/atlan/mibo-media-server/internal/storage/openlist"
	"gorm.io/gorm"
)

func TestRunSyncLibraryUsesStableIdentityEvidence(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, svc, libraryRecord := newIdentityScanService(t)
	provider := &stableIdentityProvider{objects: [][]storage.Object{
		{{Name: "MovieA.2024.mkv", Path: "/library/MovieA.2024.mkv", Size: 2048, StableIdentity: "provider-object-1", Modified: timePtr(time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC))}},
		{{Name: "Renamed.Movie.2024.mkv", Path: "/library/Renamed.Movie.2024.mkv", Size: 2048, StableIdentity: "provider-object-1", Modified: timePtr(time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC))}},
	}}

	if _, err := svc.scanLibrary(ctx, provider, libraryRecord, "/library"); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if _, err := svc.scanLibrary(ctx, provider, libraryRecord, "/library"); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	var files []database.InventoryFile
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).Order("id asc").Find(&files).Error; err != nil {
		t.Fatalf("list inventory files: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one persisted inventory file identity, got %d", len(files))
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

func TestRunSyncLibraryUpsertsInventoryFileForSamePathRescan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, svc, libraryRecord := newIdentityScanService(t)
	provider := &stableIdentityProvider{objects: [][]storage.Object{
		{{Name: "MovieA.2024.mkv", Path: "/library/MovieA.2024.mkv", Size: 2048, Provider: "alist", HashInfo: map[string]string{"sha256": "old"}, Modified: timePtr(time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC))}},
		{{Name: "MovieA.2024.mkv", Path: "/library/MovieA.2024.mkv", Size: 4096, Provider: "alist", HashInfo: map[string]string{"sha256": "new"}, Modified: timePtr(time.Date(2026, 4, 22, 11, 0, 0, 0, time.UTC))}},
	}}

	if _, err := svc.scanLibrary(ctx, provider, libraryRecord, "/library"); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if _, err := svc.scanLibrary(ctx, provider, libraryRecord, "/library"); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	var files []database.InventoryFile
	if err := db.WithContext(ctx).Where("library_id = ?", libraryRecord.ID).Order("id asc").Find(&files).Error; err != nil {
		t.Fatalf("list inventory files: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected same-path rescans to update one inventory file row, got %d rows", len(files))
	}
	if files[0].DeletedAt != nil {
		t.Fatalf("expected upserted inventory file to remain active, got deleted_at=%v", files[0].DeletedAt)
	}
	if files[0].SizeBytes != 4096 {
		t.Fatalf("expected rescan to refresh size bytes, got %d", files[0].SizeBytes)
	}
	if files[0].HashesJSON != `{"sha256":"new"}` {
		t.Fatalf("expected rescan to refresh hashes, got %q", files[0].HashesJSON)
	}
}

func TestRunSyncLibraryReusesCatalogItemAcrossRescan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, svc, libraryRecord := newIdentityScanService(t)
	provider := &stableIdentityProvider{objects: [][]storage.Object{{
		{Name: "MovieA.2024.mkv", Path: "/library/MovieA.2024.mkv", Size: 2048, Modified: timePtr(time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC))},
	}}}

	if _, err := svc.scanLibrary(ctx, provider, libraryRecord, "/library"); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	var item database.CatalogItem
	if err := db.WithContext(ctx).
		Where("library_id = ? AND path = ?", libraryRecord.ID, "/library/MovieA.2024.mkv").
		First(&item).Error; err != nil {
		t.Fatalf("load scanned catalog item: %v", err)
	}

	if _, err := svc.scanLibrary(ctx, provider, libraryRecord, "/library"); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	var itemCount int64
	if err := db.WithContext(ctx).Model(&database.CatalogItem{}).Where("library_id = ?", libraryRecord.ID).Count(&itemCount).Error; err != nil {
		t.Fatalf("count catalog items: %v", err)
	}
	if itemCount != 1 {
		t.Fatalf("expected one catalog item across rescans, got %d", itemCount)
	}
	var fileCount int64
	if err := db.WithContext(ctx).Model(&database.InventoryFile{}).Where("library_id = ?", libraryRecord.ID).Count(&fileCount).Error; err != nil {
		t.Fatalf("count inventory files: %v", err)
	}
	if fileCount != 1 {
		t.Fatalf("expected one inventory file across rescans, got %d", fileCount)
	}
}

type stableIdentityProvider struct {
	mu      sync.Mutex
	objects [][]storage.Object
	round   int
}

func (p *stableIdentityProvider) Name() string {
	return "stable-test"
}

func (p *stableIdentityProvider) List(_ context.Context, req storage.ListRequest) ([]storage.Object, error) {
	if strings.TrimSpace(req.Path) != "/library" {
		return nil, nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	index := p.round
	if index >= len(p.objects) {
		index = len(p.objects) - 1
	}
	objects := make([]storage.Object, len(p.objects[index]))
	copy(objects, p.objects[index])
	p.round++
	return objects, nil
}

func (p *stableIdentityProvider) Get(_ context.Context, req storage.GetRequest) (storage.Object, error) {
	if strings.TrimSpace(req.Path) == "/library" {
		return storage.Object{Name: "library", Path: "/library", IsDir: true}, nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, round := range p.objects {
		for _, object := range round {
			if object.Path == req.Path {
				return object, nil
			}
		}
	}
	return storage.Object{}, nil
}

func (p *stableIdentityProvider) Link(context.Context, storage.LinkRequest) (storage.LinkResult, error) {
	return storage.LinkResult{}, storage.ErrNotImplemented
}

func (p *stableIdentityProvider) ResolveStorage(ctx context.Context, req storage.ResolveStorageRequest) (storage.ResolvedStorage, error) {
	object, _ := p.Get(ctx, storage.GetRequest{Path: req.Path})
	return storage.ResolvedStorage{Provider: p.Name(), Path: req.Path, Object: object}, nil
}

func (p *stableIdentityProvider) Capabilities(context.Context) (storage.Capabilities, error) {
	return storage.Capabilities{CanList: true, CanGet: true}, nil
}

func newIdentityScanService(t *testing.T) (*gorm.DB, *Service, database.Library) {
	t.Helper()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	svc := NewService(config.Config{}, db, nil, jobs.NewService(db))

	ctx := context.Background()
	libraryRecord := database.Library{Name: "Identity", Type: "movies", RootPath: "/library", Status: "active"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}

	return db, svc, libraryRecord
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func pathBase(value string) string {
	if value == "" || value == "/" {
		return "root"
	}
	return path.Base(value)
}
