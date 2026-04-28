package probe

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

func TestCatalogFallbackArtworkUsesProviderThumbnailForPosterOnly(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, item := newArtworkFallbackService(t)
	logPath := filepath.Join(t.TempDir(), "ffmpeg.log")
	service.ffmpeg = config.FFmpegConfig{Enabled: true, Path: writeLoggingArtworkFakeFFmpeg(t, logPath), Timeout: time.Second, ArtworkRootPath: filepath.Join(t.TempDir(), "artwork")}
	provider := &artworkProvider{objects: map[string]storage.Object{
		"/Movies/Movie A.mkv": {Name: "Movie A.mkv", Path: "/Movies/Movie A.mkv", ThumbnailURL: "https://cdn.example.test/thumb.jpg"},
	}}

	if err := service.generateCatalogFallbackArtworkForItem(ctx, item.ID, "/Movies/Movie A.mkv", provider, "/tmp/movie.mkv", nil); err != nil {
		t.Fatalf("generate fallback artwork: %v", err)
	}

	images := loadArtworkImages(t, service.db, item.ID)
	selected := selectedArtworkByKind(images)
	if selected[posterArtworkKind] != "https://cdn.example.test/thumb.jpg" {
		t.Fatalf("expected provider thumbnail poster, got %#v", images)
	}
	if !strings.HasPrefix(selected[backdropArtworkKind], "/api/v1/items/") {
		t.Fatalf("expected generated backdrop, got %#v", images)
	}
	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read ffmpeg log: %v", err)
	}
	log := string(logBytes)
	if strings.Contains(log, "poster.jpg") {
		t.Fatalf("expected ffmpeg not to generate poster, log=%q", log)
	}
	if !strings.Contains(log, "backdrop.jpg") {
		t.Fatalf("expected ffmpeg to generate backdrop, log=%q", log)
	}
}

func TestCatalogFallbackArtworkUsesProviderThumbnailAsEpisodeStill(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, item := newArtworkFallbackServiceForType(t, "episode", "/Shows/Season 1/Show S01E01.mkv")
	logPath := filepath.Join(t.TempDir(), "ffmpeg.log")
	service.ffmpeg = config.FFmpegConfig{Enabled: true, Path: writeLoggingArtworkFakeFFmpeg(t, logPath), Timeout: time.Second, ArtworkRootPath: filepath.Join(t.TempDir(), "artwork")}
	provider := &artworkProvider{objects: map[string]storage.Object{
		"/Shows/Season 1/Show S01E01.mkv": {Name: "Show S01E01.mkv", Path: "/Shows/Season 1/Show S01E01.mkv", ThumbnailURL: "https://cdn.example.test/episode-thumb.jpg"},
	}}

	if err := service.generateCatalogFallbackArtworkForItem(ctx, item.ID, "/Shows/Season 1/Show S01E01.mkv", provider, "/tmp/episode.mkv", nil); err != nil {
		t.Fatalf("generate fallback artwork: %v", err)
	}

	images := loadArtworkImages(t, service.db, item.ID)
	selected := selectedArtworkByKind(images)
	if selected[stillArtworkKind] != "https://cdn.example.test/episode-thumb.jpg" {
		t.Fatalf("expected provider thumbnail still, got %#v", images)
	}
	if selected[posterArtworkKind] != "" {
		t.Fatalf("expected no generated episode poster, got %#v", images)
	}
	if !strings.HasPrefix(selected[backdropArtworkKind], "/api/v1/items/") {
		t.Fatalf("expected generated backdrop, got %#v", images)
	}
	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read ffmpeg log: %v", err)
	}
	log := string(logBytes)
	if strings.Contains(log, "poster.jpg") {
		t.Fatalf("expected ffmpeg not to generate episode poster, log=%q", log)
	}
	if !strings.Contains(log, "backdrop.jpg") {
		t.Fatalf("expected ffmpeg to generate episode backdrop, log=%q", log)
	}
}

func TestCatalogFallbackArtworkPreservesNonGeneratedEpisodeStillBeforeProviderThumbnail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, item := newArtworkFallbackServiceForType(t, "episode", "/Shows/Season 1/Show S01E01.mkv")
	if err := service.db.WithContext(ctx).Create(&database.ItemImage{ItemID: item.ID, ImageType: stillArtworkKind, URL: "https://image.tmdb.org/t/p/original/still.jpg", IsSelected: true}).Error; err != nil {
		t.Fatalf("seed remote still: %v", err)
	}
	provider := &artworkProvider{objects: map[string]storage.Object{
		"/Shows/Season 1/Show S01E01.mkv": {Name: "Show S01E01.mkv", Path: "/Shows/Season 1/Show S01E01.mkv", ThumbnailURL: "https://cdn.example.test/episode-thumb.jpg"},
	}}

	if err := service.generateCatalogFallbackArtworkForItem(ctx, item.ID, "/Shows/Season 1/Show S01E01.mkv", provider, "", nil); err != nil {
		t.Fatalf("generate fallback artwork: %v", err)
	}

	selected := selectedArtworkByKind(loadArtworkImages(t, service.db, item.ID))
	if selected[stillArtworkKind] != "https://image.tmdb.org/t/p/original/still.jpg" {
		t.Fatalf("expected remote still to remain selected, got %#v", selected)
	}
}

func TestCatalogFallbackArtworkUsesSiblingPosterBeforeProviderThumbnail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, item := newArtworkFallbackService(t)
	provider := &artworkProvider{objects: map[string]storage.Object{
		"/Movies/Movie A.mkv": {Name: "Movie A.mkv", Path: "/Movies/Movie A.mkv", ThumbnailURL: "https://cdn.example.test/thumb.jpg"},
		"/Movies/cover.jpg":   {Name: "cover.jpg", Path: "/Movies/cover.jpg"},
	}, links: map[string]string{"/Movies/cover.jpg": "https://cdn.example.test/cover.jpg"}}

	if err := service.generateCatalogFallbackArtworkForItem(ctx, item.ID, "/Movies/Movie A.mkv", provider, "", nil); err != nil {
		t.Fatalf("generate fallback artwork: %v", err)
	}

	selected := selectedArtworkByKind(loadArtworkImages(t, service.db, item.ID))
	if selected[posterArtworkKind] != "https://cdn.example.test/cover.jpg" {
		t.Fatalf("expected sibling cover poster, got %#v", selected)
	}
}

func TestCatalogFallbackArtworkPreservesNonGeneratedPosterBeforeProviderThumbnail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, item := newArtworkFallbackService(t)
	if err := service.db.WithContext(ctx).Create(&database.ItemImage{ItemID: item.ID, ImageType: posterArtworkKind, URL: "https://image.tmdb.org/t/p/original/poster.jpg", IsSelected: true}).Error; err != nil {
		t.Fatalf("seed remote poster: %v", err)
	}
	provider := &artworkProvider{objects: map[string]storage.Object{
		"/Movies/Movie A.mkv": {Name: "Movie A.mkv", Path: "/Movies/Movie A.mkv", ThumbnailURL: "https://cdn.example.test/thumb.jpg"},
	}}

	if err := service.generateCatalogFallbackArtworkForItem(ctx, item.ID, "/Movies/Movie A.mkv", provider, "", nil); err != nil {
		t.Fatalf("generate fallback artwork: %v", err)
	}

	selected := selectedArtworkByKind(loadArtworkImages(t, service.db, item.ID))
	if selected[posterArtworkKind] != "https://image.tmdb.org/t/p/original/poster.jpg" {
		t.Fatalf("expected remote poster to remain selected, got %#v", selected)
	}
}

func TestCatalogFallbackArtworkUsesRelatedSiblingBeforeDirectGet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, item := newArtworkFallbackService(t)
	provider := &artworkProvider{objects: map[string]storage.Object{
		"/Movies/Movie A.mkv": {
			Name: "Movie A.mkv",
			Path: "/Movies/Movie A.mkv",
			Related: []storage.Object{{
				Name:   "cover.jpg",
				Path:   "/Movies/cover.jpg",
				RawURL: "https://cdn.example.test/related-cover.jpg",
			}},
		},
	}, getCalls: map[string]int{}}

	if err := service.generateCatalogFallbackArtworkForItem(ctx, item.ID, "/Movies/Movie A.mkv", provider, "", nil); err != nil {
		t.Fatalf("generate fallback artwork: %v", err)
	}

	selected := selectedArtworkByKind(loadArtworkImages(t, service.db, item.ID))
	if selected[posterArtworkKind] != "https://cdn.example.test/related-cover.jpg" {
		t.Fatalf("expected related cover poster, got %#v", selected)
	}
	if provider.getCalls["/Movies/cover.jpg"] != 0 {
		t.Fatalf("expected related cover to avoid direct get, calls=%#v", provider.getCalls)
	}
}

func TestCatalogFallbackArtworkFallsBackWhenRelatedDoesNotMatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, item := newArtworkFallbackService(t)
	provider := &artworkProvider{objects: map[string]storage.Object{
		"/Movies/Movie A.mkv": {Name: "Movie A.mkv", Path: "/Movies/Movie A.mkv", Related: []storage.Object{{Name: "stale.jpg", Path: "/Movies/stale.jpg"}}},
		"/Movies/cover.jpg":   {Name: "cover.jpg", Path: "/Movies/cover.jpg", RawURL: "https://cdn.example.test/direct-cover.jpg"},
	}, getCalls: map[string]int{}}

	if err := service.generateCatalogFallbackArtworkForItem(ctx, item.ID, "/Movies/Movie A.mkv", provider, "", nil); err != nil {
		t.Fatalf("generate fallback artwork: %v", err)
	}

	selected := selectedArtworkByKind(loadArtworkImages(t, service.db, item.ID))
	if selected[posterArtworkKind] != "https://cdn.example.test/direct-cover.jpg" {
		t.Fatalf("expected direct cover fallback, got %#v", selected)
	}
	if provider.getCalls["/Movies/cover.jpg"] == 0 {
		t.Fatalf("expected direct get fallback, calls=%#v", provider.getCalls)
	}
}

func TestCatalogFallbackArtworkBlankProviderThumbnailKeepsFrameExtraction(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, item := newArtworkFallbackService(t)
	logPath := filepath.Join(t.TempDir(), "ffmpeg.log")
	service.ffmpeg = config.FFmpegConfig{Enabled: true, Path: writeLoggingArtworkFakeFFmpeg(t, logPath), Timeout: time.Second, ArtworkRootPath: filepath.Join(t.TempDir(), "artwork")}
	provider := &artworkProvider{objects: map[string]storage.Object{
		"/Movies/Movie A.mkv": {Name: "Movie A.mkv", Path: "/Movies/Movie A.mkv", ThumbnailURL: "   "},
	}}

	if err := service.generateCatalogFallbackArtworkForItem(ctx, item.ID, "/Movies/Movie A.mkv", provider, "/tmp/movie.mkv", nil); err != nil {
		t.Fatalf("generate fallback artwork: %v", err)
	}

	selected := selectedArtworkByKind(loadArtworkImages(t, service.db, item.ID))
	if !strings.HasPrefix(selected[posterArtworkKind], "/api/v1/items/") || !strings.HasPrefix(selected[backdropArtworkKind], "/api/v1/items/") {
		t.Fatalf("expected generated poster and backdrop, got %#v", selected)
	}
	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read ffmpeg log: %v", err)
	}
	log := string(logBytes)
	if !strings.Contains(log, "poster.jpg") || !strings.Contains(log, "backdrop.jpg") {
		t.Fatalf("expected ffmpeg to generate poster and backdrop, log=%q", log)
	}
}

func newArtworkFallbackService(t *testing.T) (*Service, database.CatalogItem) {
	t.Helper()
	return newArtworkFallbackServiceForType(t, "movie", "/Movies/Movie A.mkv")
}

func newArtworkFallbackServiceForType(t *testing.T, itemType string, itemPath string) (*Service, database.CatalogItem) {
	t.Helper()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	item := database.CatalogItem{Type: itemType, Path: itemPath, SortKey: "Fallback Item", DisplayOrder: "aired", Title: "Fallback Item", AvailabilityStatus: "available", GovernanceStatus: "pending"}
	if err := db.WithContext(context.Background()).Create(&item).Error; err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	return NewService(db, nil, config.FFprobeConfig{}), item
}

func loadArtworkImages(t *testing.T, db *gorm.DB, itemID uint) []database.ItemImage {
	t.Helper()

	var images []database.ItemImage
	if err := db.WithContext(context.Background()).Where("item_id = ?", itemID).Order("image_type asc, id asc").Find(&images).Error; err != nil {
		t.Fatalf("load artwork images: %v", err)
	}
	return images
}

func selectedArtworkByKind(images []database.ItemImage) map[string]string {
	selected := make(map[string]string, len(images))
	for _, image := range images {
		if image.IsSelected {
			selected[image.ImageType] = image.URL
		}
	}
	return selected
}

func writeLoggingArtworkFakeFFmpeg(t *testing.T, logPath string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ffmpeg")
	content := "#!/bin/sh\nout=\"\"\nfor arg in \"$@\"; do\n  out=\"$arg\"\ndone\nmkdir -p \"$(dirname \"$out\")\"\nprintf 'fake-artwork' > \"$out\"\nprintf '%s\\n' \"$(basename \"$out\")\" >> \"" + logPath + "\"\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake ffmpeg: %v", err)
	}
	return path
}

type artworkProvider struct {
	objects  map[string]storage.Object
	links    map[string]string
	getCalls map[string]int
}

func (p *artworkProvider) Name() string {
	return "openlist-test"
}

func (p *artworkProvider) List(context.Context, storage.ListRequest) ([]storage.Object, error) {
	return nil, storage.ErrNotImplemented
}

func (p *artworkProvider) Get(_ context.Context, req storage.GetRequest) (storage.Object, error) {
	if p.getCalls != nil {
		p.getCalls[req.Path]++
	}
	object, ok := p.objects[req.Path]
	if !ok {
		return storage.Object{}, errors.New("not found")
	}
	return object, nil
}

func (p *artworkProvider) Link(_ context.Context, req storage.LinkRequest) (storage.LinkResult, error) {
	link, ok := p.links[req.Path]
	if !ok {
		return storage.LinkResult{}, storage.ErrNotImplemented
	}
	return storage.LinkResult{URL: link}, nil
}

func (p *artworkProvider) ResolveStorage(context.Context, storage.ResolveStorageRequest) (storage.ResolvedStorage, error) {
	return storage.ResolvedStorage{}, storage.ErrNotImplemented
}

func (p *artworkProvider) Capabilities(context.Context) (storage.Capabilities, error) {
	return storage.Capabilities{CanGet: true, CanLink: true}, nil
}
