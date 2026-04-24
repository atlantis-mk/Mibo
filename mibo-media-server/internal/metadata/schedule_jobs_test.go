package metadata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/schedule"
	"github.com/atlan/mibo-media-server/internal/settings"
	"gorm.io/gorm"
)

func TestScheduledMetadataRefetchAndTrailerSyncRespectScope(t *testing.T) {
	tmdb := newScheduledTMDBServer(t)
	defer tmdb.Close()

	db, svc := newScheduledMetadataService(t, tmdb.URL)
	ctx := context.Background()
	itemOne := createScheduledItem(t, db, 1, "movie:101")
	_ = createScheduledItem(t, db, 2, "movie:102")

	result, err := svc.RunScheduledMetadataRefetch(ctx, schedule.DueSchedule{Kind: schedule.KindMetadataRefetch, ScopeKind: schedule.ScopeLibrary, LibraryID: uintPtr(1)})
	if err != nil {
		t.Fatalf("metadata refetch: %v", err)
	}
	if result.UpdatedItems != 1 {
		t.Fatalf("expected one updated item, got %#v", result)
	}

	var stored database.MediaItem
	if err := db.WithContext(ctx).First(&stored, itemOne.ID).Error; err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if stored.Title == "Old Title" || stored.TrailerJSON == "" {
		t.Fatalf("expected refetched metadata and trailer, got %#v", stored)
	}

	globalResult, err := svc.RunScheduledTrailerSync(ctx, schedule.DueSchedule{Kind: schedule.KindTrailerSync, ScopeKind: schedule.ScopeGlobal})
	if err != nil {
		t.Fatalf("trailer sync: %v", err)
	}
	if globalResult.ItemsProcessed != 2 {
		t.Fatalf("expected global trailer sync over 2 items, got %#v", globalResult)
	}
	if !json.Valid([]byte(stored.TrailerJSON)) {
		t.Fatalf("expected persisted trailer json to remain normalized, got %s", stored.TrailerJSON)
	}
}

func TestScheduledArtworkRefreshOnlyTouchesArtworkFields(t *testing.T) {
	tmdb := newScheduledTMDBServer(t)
	defer tmdb.Close()

	db, svc := newScheduledMetadataService(t, tmdb.URL)
	ctx := context.Background()
	item := createScheduledItem(t, db, 1, "movie:101")
	originalOverview := item.Overview

	result, err := svc.RunScheduledArtworkRefresh(ctx, schedule.DueSchedule{Kind: schedule.KindArtworkRefresh, ScopeKind: schedule.ScopeLibrary, LibraryID: uintPtr(1)})
	if err != nil {
		t.Fatalf("artwork refresh: %v", err)
	}
	if result.UpdatedItems != 1 {
		t.Fatalf("expected one artwork refresh, got %#v", result)
	}

	var stored database.MediaItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if stored.PosterURL == "" || stored.BackdropURL == "" || stored.LogoURL == "" {
		t.Fatalf("expected artwork fields populated, got %#v", stored)
	}
	if stored.Overview != originalOverview {
		t.Fatalf("expected overview unchanged, got %q want %q", stored.Overview, originalOverview)
	}
}

func newScheduledMetadataService(t *testing.T, tmdbURL string) (*gorm.DB, *Service) {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	settingsSvc := settings.NewService(db, config.MetadataConfig{TMDB: config.TMDBConfig{BaseURL: tmdbURL, ImageBaseURL: tmdbURL + "/images", Language: "en-US", Timeout: time.Second}})
	if _, err := settingsSvc.UpdateMetadataSettings(context.Background(), settings.UpdateMetadataSettingsInput{TMDB: settings.MetadataProviderInput{APIKey: "tmdb-test", BaseURL: tmdbURL, ImageBaseURL: tmdbURL + "/images", Language: "en-US", Timeout: "1s"}}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}
	return db, NewService(db, config.MetadataConfig{}, settingsSvc)
}

func createScheduledItem(t *testing.T, db *gorm.DB, libraryID uint, externalID string) database.MediaItem {
	t.Helper()
	confidence := 0.9
	item := database.MediaItem{LibraryID: libraryID, Type: "movie", Title: "Old Title", Overview: "Original overview", SourcePath: filepath.Join("/library", externalID+".mkv"), MatchStatus: StatusMatched, MetadataProvider: "tmdb", ExternalID: externalID, MetadataConfidence: &confidence, Status: "ready"}
	if err := db.WithContext(context.Background()).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	return item
}

func newScheduledTMDBServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/movie/101", "/movie/102":
			id := 101
			if req.URL.Path == "/movie/102" {
				id = 102
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             id,
				"title":          "Updated Title",
				"original_title": "Updated Title",
				"overview":       "Updated overview",
				"poster_path":    "/poster.jpg",
				"backdrop_path":  "/backdrop.jpg",
				"release_date":   "2024-02-02",
				"runtime":        121,
				"genres":         []map[string]any{{"name": "Action"}},
				"credits":        map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}},
				"images":         map[string]any{"logos": []map[string]any{{"file_path": "/logo.png", "iso_639_1": "en", "vote_average": 9.0}}},
				"videos":         map[string]any{"results": []map[string]any{{"name": "Official Trailer", "key": "abc123", "site": "YouTube", "type": "Trailer", "official": true, "iso_639_1": "en"}}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func uintPtr(v uint) *uint { return &v }
