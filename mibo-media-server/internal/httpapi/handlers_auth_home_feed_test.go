package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/auth"
	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/progress"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
	"gorm.io/gorm"
)

func TestHomeContentSectionsRequireAuthentication(t *testing.T) {
	handler, _, _ := newHomeFeedTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/home/sections", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHomeContentSectionsReturnSemanticSections(t *testing.T) {
	handler, db, authSvc := newHomeFeedTestServer(t)
	authHeader := createAuthHeader(t, t.Context(), authSvc)
	now := time.Now().UTC()
	libraryRecord := database.Library{Name: "Mixed", Type: "mixed", RootPath: "/media", Status: "active"}
	if err := db.WithContext(t.Context()).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	items := []database.MetadataItem{
		{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Home Movie", SortTitle: "Home Movie", GovernanceStatus: database.ReviewStateAccepted},
		{ItemType: database.MetadataItemTypeSeries, ContentForm: database.MetadataContentFormStandard, Title: "Home Series", SortTitle: "Home Series", GovernanceStatus: database.ReviewStateAccepted},
		{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Hidden Movie", SortTitle: "Hidden Movie", GovernanceStatus: database.ReviewStateAccepted},
	}
	if err := db.WithContext(t.Context()).Create(&items).Error; err != nil {
		t.Fatalf("create items: %v", err)
	}
	if err := db.WithContext(t.Context()).Create(&database.MetadataItemImage{MetadataItemID: items[0].ID, ImageType: "poster", URL: "/api/v1/artwork/poster.jpg", IsSelected: true}).Error; err != nil {
		t.Fatalf("create image: %v", err)
	}
	projections := []database.LibraryMetadataProjection{
		{LibraryID: libraryRecord.ID, MetadataItemID: items[0].ID, ItemType: items[0].ItemType, Title: items[0].Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, LatestAddedAt: timePtr(now), LastProjectedAt: now},
		{LibraryID: libraryRecord.ID, MetadataItemID: items[1].ID, ItemType: items[1].ItemType, Title: items[1].Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, LatestAddedAt: timePtr(now.Add(-time.Minute)), LastProjectedAt: now},
		{LibraryID: libraryRecord.ID, MetadataItemID: items[2].ID, ItemType: items[2].ItemType, Title: items[2].Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, Hidden: true, LatestAddedAt: timePtr(now.Add(time.Minute)), LastProjectedAt: now},
	}
	if err := db.WithContext(t.Context()).Create(&projections).Error; err != nil {
		t.Fatalf("create projections: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/home/sections", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	sections := decodeHomeSections(t, rec)
	if len(sections) != 2 {
		t.Fatalf("expected two sections, got %#v", sections)
	}
	if sections[0].Key != "movies" || sections[0].Title != "电影" || len(sections[0].Items) != 1 || sections[0].Items[0].Title != "Home Movie" {
		t.Fatalf("unexpected movie section: %#v", sections[0])
	}
	if sections[0].Items[0].SelectedImages[0].URL != "http://example.com/api/v1/artwork/poster.jpg" {
		t.Fatalf("expected normalized artwork URL, got %#v", sections[0].Items[0].SelectedImages)
	}
	if sections[1].Key != "series" || sections[1].Title != "剧集" || len(sections[1].Items) != 1 || sections[1].Items[0].Title != "Home Series" {
		t.Fatalf("unexpected series section: %#v", sections[1])
	}
}

func TestHomeMediaOverviewReturnsCountsAndNormalizedArtwork(t *testing.T) {
	handler, db, authSvc := newHomeFeedTestServer(t)
	authHeader := createAuthHeader(t, t.Context(), authSvc)
	now := time.Now().UTC()
	libraryRecord := database.Library{Name: "Mixed", Type: "mixed", RootPath: "/media", Status: "active"}
	if err := db.WithContext(t.Context()).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	items := []database.MetadataItem{
		{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Home Movie 1", SortTitle: "Home Movie 1", GovernanceStatus: database.ReviewStateAccepted},
		{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Home Movie 2", SortTitle: "Home Movie 2", GovernanceStatus: database.ReviewStateAccepted},
		{ItemType: database.MetadataItemTypeSeries, ContentForm: database.MetadataContentFormStandard, Title: "Home Series", SortTitle: "Home Series", GovernanceStatus: database.ReviewStateAccepted},
	}
	if err := db.WithContext(t.Context()).Create(&items).Error; err != nil {
		t.Fatalf("create items: %v", err)
	}
	if err := db.WithContext(t.Context()).Create(&database.MetadataItemImage{MetadataItemID: items[0].ID, ImageType: "poster", URL: "/api/v1/artwork/poster.jpg", IsSelected: true}).Error; err != nil {
		t.Fatalf("create image: %v", err)
	}
	projections := []database.LibraryMetadataProjection{
		{LibraryID: libraryRecord.ID, MetadataItemID: items[0].ID, ItemType: items[0].ItemType, Title: items[0].Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, LatestAddedAt: timePtr(now), LastProjectedAt: now},
		{LibraryID: libraryRecord.ID, MetadataItemID: items[1].ID, ItemType: items[1].ItemType, Title: items[1].Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, LatestAddedAt: timePtr(now.Add(-time.Minute)), LastProjectedAt: now},
		{LibraryID: libraryRecord.ID, MetadataItemID: items[2].ID, ItemType: items[2].ItemType, Title: items[2].Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, LatestAddedAt: timePtr(now.Add(-2 * time.Minute)), LastProjectedAt: now},
	}
	if err := db.WithContext(t.Context()).Create(&projections).Error; err != nil {
		t.Fatalf("create projections: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/home/media-overview?preview_limit=1", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	overview := decodeHomeMediaOverview(t, rec)
	if len(overview.Sections) != 2 {
		t.Fatalf("expected two sections, got %#v", overview)
	}
	if overview.Sections[0].Key != "movies" || overview.Sections[0].Count != 2 || len(overview.Sections[0].Items) != 1 {
		t.Fatalf("unexpected movie overview section: %#v", overview.Sections[0])
	}
	if overview.Sections[0].Items[0].SelectedImages[0].URL != "http://example.com/api/v1/artwork/poster.jpg" {
		t.Fatalf("expected normalized artwork URL, got %#v", overview.Sections[0].Items[0].SelectedImages)
	}
	if overview.Sections[1].Key != "series" || overview.Sections[1].Count != 1 || len(overview.Sections[1].Items) != 1 {
		t.Fatalf("unexpected series overview section: %#v", overview.Sections[1])
	}
}

func newHomeFeedTestServer(t *testing.T) (http.Handler, *gorm.DB, *auth.Service) {
	t.Helper()
	cfg := config.Config{
		HTTP:     config.HTTPConfig{Addr: ":8080"},
		Local:    config.LocalStorageConfig{RootPath: t.TempDir()},
		Database: config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")},
		Worker:   config.WorkerConfig{Enabled: true},
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, nil)
	searchSvc := search.NewService(db, librarySvc)
	handler := New(Dependencies{
		Config:   cfg,
		DB:       db,
		Registry: registry,
		Auth:     authSvc,
		Catalog:  catalog.NewService(db),
		Library:  librarySvc,
		Playback: playback.NewService(db, registry),
		Progress: progress.NewService(db, searchSvc),
		Search:   searchSvc,
		Settings: settings.NewService(db, cfg.Metadata),
	})
	return handler, db, authSvc
}

func decodeHomeSections(t *testing.T, rec *httptest.ResponseRecorder) []catalog.HomeContentSection {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	var sections []catalog.HomeContentSection
	if err := json.Unmarshal(data, &sections); err != nil {
		t.Fatalf("decode sections: %v", err)
	}
	return sections
}

func decodeHomeMediaOverview(t *testing.T, rec *httptest.ResponseRecorder) catalog.HomeMediaOverview {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	var overview catalog.HomeMediaOverview
	if err := json.Unmarshal(data, &overview); err != nil {
		t.Fatalf("decode overview: %v", err)
	}
	return overview
}
