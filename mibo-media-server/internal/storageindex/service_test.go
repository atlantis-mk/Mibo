package storageindex

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
)

func newTestService(t *testing.T) (context.Context, *Service, database.Library) {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	source := database.MediaSource{Name: "Local", Provider: "local", StorageRef: "local", RootPath: "/media"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	library := database.Library{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: "/media/movies", Status: "active", ScannerEnabled: true}
	if err := db.Create(&library).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	svc := NewService(db)
	svc.now = func() time.Time { return time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC) }
	return context.Background(), svc, library
}

func TestUpsertPresentCreatesAndUpdatesStorageIndexEntry(t *testing.T) {
	ctx, svc, library := newTestService(t)
	modified := time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC)
	entry, err := svc.UpsertPresent(ctx, ObservationInput{
		LibraryID:         library.ID,
		StorageProvider:   "local",
		StoragePath:       "/media/movies/Movie A.mkv",
		SizeBytes:         1024,
		ModifiedAt:        &modified,
		StableIdentityKey: "dev-1:ino-2",
		Hashes:            map[string]string{"etag": "abc"},
		ProviderName:      "local",
		ObjectType:        "file",
		ProviderMeta:      map[string]string{"device": "1", "inode": "2"},
	})
	if err != nil {
		t.Fatalf("upsert present: %v", err)
	}
	if entry.ObservationStatus != ObservationStatusPresent || entry.StoragePath != "/media/movies/Movie A.mkv" || entry.SizeBytes != 1024 {
		t.Fatalf("unexpected entry: %#v", entry)
	}
	if entry.HashesJSON == "" || entry.ProviderMetaJSON == "" {
		t.Fatalf("expected encoded evidence, got %#v", entry)
	}

	svc.now = func() time.Time { return time.Date(2026, 4, 29, 11, 0, 0, 0, time.UTC) }
	updated, err := svc.UpsertPresent(ctx, ObservationInput{LibraryID: library.ID, StorageProvider: "local", StoragePath: "/media/movies/Movie A.mkv", SizeBytes: 2048})
	if err != nil {
		t.Fatalf("update present: %v", err)
	}
	if updated.ID != entry.ID {
		t.Fatalf("expected upsert to reuse id %d, got %d", entry.ID, updated.ID)
	}
	if updated.SizeBytes != 2048 || !updated.LastObservedAt.Equal(time.Date(2026, 4, 29, 11, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected updated observation, got %#v", updated)
	}

	var count int64
	if err := svc.db.WithContext(ctx).Model(&database.StorageIndexEntry{}).Where("library_id = ?", library.ID).Count(&count).Error; err != nil {
		t.Fatalf("count index entries: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one index row, got %d", count)
	}
}

func TestMarkMissingKeepsStorageIndexEntry(t *testing.T) {
	ctx, svc, library := newTestService(t)
	if _, err := svc.UpsertPresent(ctx, ObservationInput{LibraryID: library.ID, StorageProvider: "local", StoragePath: "/media/movies/Missing.mkv", SizeBytes: 1024}); err != nil {
		t.Fatalf("upsert present: %v", err)
	}
	svc.now = func() time.Time { return time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC) }
	missing, err := svc.MarkMissing(ctx, library.ID, "local", "/media/movies/Missing.mkv")
	if err != nil {
		t.Fatalf("mark missing: %v", err)
	}
	if missing.ObservationStatus != ObservationStatusMissing || missing.MissingSince == nil {
		t.Fatalf("expected missing entry with timestamp, got %#v", missing)
	}
}

func TestListScopedReturnsOnlyDescendantEntries(t *testing.T) {
	ctx, svc, library := newTestService(t)
	paths := []string{
		"/media/movies/A/Movie.mkv",
		"/media/movies/A/Sub/Extra.mkv",
		"/media/movies/B/Movie.mkv",
	}
	for _, path := range paths {
		if _, err := svc.UpsertPresent(ctx, ObservationInput{LibraryID: library.ID, StorageProvider: "local", StoragePath: path}); err != nil {
			t.Fatalf("upsert %s: %v", path, err)
		}
	}
	entries, err := svc.ListScoped(ctx, library.ID, "/media/movies/A")
	if err != nil {
		t.Fatalf("list scoped: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected two scoped entries, got %#v", entries)
	}
	for _, entry := range entries {
		if entry.StoragePath == "/media/movies/B/Movie.mkv" {
			t.Fatalf("unexpected out-of-scope entry: %#v", entry)
		}
	}
}

func TestRecordFailureStoresObservationFailure(t *testing.T) {
	ctx, svc, library := newTestService(t)
	failure, err := svc.RecordFailure(ctx, FailureInput{LibraryID: library.ID, StorageProvider: "openlist", StoragePath: "/media", Reason: "list_failed", Error: errors.New("upstream unavailable")})
	if err != nil {
		t.Fatalf("record failure: %v", err)
	}
	if failure.ID == 0 || failure.ErrorMessage != "upstream unavailable" || failure.Reason != "list_failed" {
		t.Fatalf("unexpected failure: %#v", failure)
	}
}

func TestObservationFromObjectMapsProviderEvidence(t *testing.T) {
	ctx, svc, library := newTestService(t)
	modified := time.Date(2026, 4, 29, 9, 30, 0, 0, time.UTC)
	entry, err := svc.UpsertPresent(ctx, svc.ObservationFromObject(library.ID, "openlist", storage.Object{
		Name:           "Movie.mkv",
		Path:           "/media/Movie.mkv",
		Size:           4096,
		Modified:       &modified,
		StableIdentity: "openlist-stable-1",
		Provider:       "alist",
		HashInfo:       map[string]string{"sha1": "abc"},
		ObjectType:     "file",
		ThumbnailURL:   "https://cdn.example.test/movie-thumb.jpg",
		ProviderMeta:   map[string]string{"has_sign": "true"},
	}))
	if err != nil {
		t.Fatalf("upsert mapped object: %v", err)
	}
	if entry.StorageProvider != "openlist" || entry.StableIdentityKey != "openlist-stable-1" || entry.ProviderName != "alist" || entry.ObjectType != "file" {
		t.Fatalf("unexpected mapped entry: %#v", entry)
	}
	if entry.HashesJSON == "" || !strings.Contains(entry.ProviderMetaJSON, "thumbnail_url") {
		t.Fatalf("expected provider evidence to be encoded, got %#v", entry)
	}
}

func TestObserveTreeWalksProviderAndRecordsFailures(t *testing.T) {
	ctx, svc, library := newTestService(t)
	provider := &fakeProvider{
		name: "openlist",
		objects: map[string]storage.Object{
			"/media":             {Name: "media", Path: "/media", IsDir: true},
			"/media/Movie":       {Name: "Movie", Path: "/media/Movie", IsDir: true},
			"/media/Other":       {Name: "Other", Path: "/media/Other", IsDir: true},
			"/media/Movie/a.mkv": {Name: "a.mkv", Path: "/media/Movie/a.mkv", StableIdentity: "stable-a", Size: 100},
		},
		children: map[string][]storage.Object{
			"/media":       {{Name: "Movie", Path: "/media/Movie", IsDir: true}, {Name: "Other", Path: "/media/Other", IsDir: true}},
			"/media/Movie": {{Name: "a.mkv", Path: "/media/Movie/a.mkv", StableIdentity: "stable-a", Size: 100}},
			"/media/Other": nil,
		},
	}
	entries, err := svc.ObserveTree(ctx, ObserveTreeInput{LibraryID: library.ID, Provider: provider, RootPath: "/media", Refresh: true})
	if err != nil {
		t.Fatalf("observe tree: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("expected root plus three children, got %#v", entries)
	}
	file, err := svc.Find(ctx, library.ID, "openlist", "/media/Movie/a.mkv")
	if err != nil {
		t.Fatalf("find observed file: %v", err)
	}
	if file.StableIdentityKey != "stable-a" {
		t.Fatalf("expected stable identity from provider, got %#v", file)
	}

	provider.failListPath = "/media/Movie"
	if _, err := svc.ObserveTree(ctx, ObserveTreeInput{LibraryID: library.ID, Provider: provider, RootPath: "/media"}); err == nil {
		t.Fatalf("expected list failure")
	}
	var failures []database.StorageObservationFailure
	if err := svc.db.WithContext(ctx).Where("library_id = ? AND reason = ?", library.ID, "list_failed").Find(&failures).Error; err != nil {
		t.Fatalf("load failures: %v", err)
	}
	if len(failures) != 1 || failures[0].StoragePath != "/media/Movie" {
		t.Fatalf("expected recorded list failure, got %#v", failures)
	}
}

func TestObserveTreeSkipsUnchangedSubtrees(t *testing.T) {
	ctx, svc, library := newTestService(t)
	provider := &fakeProvider{
		name: "openlist",
		objects: map[string]storage.Object{
			"/media":             {Name: "media", Path: "/media", IsDir: true},
			"/media/Movie":       {Name: "Movie", Path: "/media/Movie", IsDir: true},
			"/media/Movie/a.mkv": {Name: "a.mkv", Path: "/media/Movie/a.mkv", StableIdentity: "stable-a", Size: 100},
		},
		children: map[string][]storage.Object{
			"/media":       {{Name: "Movie", Path: "/media/Movie", IsDir: true}},
			"/media/Movie": {{Name: "a.mkv", Path: "/media/Movie/a.mkv", StableIdentity: "stable-a", Size: 100}},
		},
	}
	if _, err := svc.ObserveTree(ctx, ObserveTreeInput{LibraryID: library.ID, Provider: provider, RootPath: "/media", Refresh: true, SkipUnchanged: true}); err != nil {
		t.Fatalf("seed observe tree: %v", err)
	}
	provider.listCounts = map[string]int{}
	if _, err := svc.ObserveTree(ctx, ObserveTreeInput{LibraryID: library.ID, Provider: provider, RootPath: "/media", Refresh: true, SkipUnchanged: true}); err != nil {
		t.Fatalf("second observe tree: %v", err)
	}
	if provider.listCounts["/media"] != 1 {
		t.Fatalf("expected root to be listed once, got %#v", provider.listCounts)
	}
	if provider.listCounts["/media/Movie"] != 0 {
		t.Fatalf("expected unchanged child subtree to be skipped, got %#v", provider.listCounts)
	}
}

type fakeProvider struct {
	name         string
	objects      map[string]storage.Object
	children     map[string][]storage.Object
	failListPath string
	listCounts   map[string]int
}

func (p *fakeProvider) Name() string { return p.name }

func (p *fakeProvider) List(_ context.Context, req storage.ListRequest) ([]storage.Object, error) {
	if p.listCounts == nil {
		p.listCounts = make(map[string]int)
	}
	p.listCounts[req.Path]++
	if req.Path == p.failListPath {
		return nil, errors.New("list failed")
	}
	return append([]storage.Object(nil), p.children[req.Path]...), nil
}

func (p *fakeProvider) Get(_ context.Context, req storage.GetRequest) (storage.Object, error) {
	object, ok := p.objects[req.Path]
	if !ok {
		return storage.Object{}, errors.New("not found")
	}
	return object, nil
}

func (p *fakeProvider) Link(context.Context, storage.LinkRequest) (storage.LinkResult, error) {
	return storage.LinkResult{}, storage.ErrNotImplemented
}

func (p *fakeProvider) ResolveStorage(ctx context.Context, req storage.ResolveStorageRequest) (storage.ResolvedStorage, error) {
	object, err := p.Get(ctx, storage.GetRequest{Path: req.Path})
	if err != nil {
		return storage.ResolvedStorage{}, err
	}
	return storage.ResolvedStorage{Provider: p.Name(), Path: req.Path, Object: object}, nil
}

func (p *fakeProvider) Capabilities(context.Context) (storage.Capabilities, error) {
	return storage.Capabilities{CanList: true, CanGet: true}, nil
}
