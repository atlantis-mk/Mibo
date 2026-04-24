package metadata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
)

func TestMatchItemUsesDatabaseTMDBConfig(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			if req.URL.Query().Get("api_key") != "db-test-key" {
				t.Fatalf("expected db api key, got %q", req.URL.Query().Get("api_key"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 101, "title": "MovieA", "original_title": "MovieA", "release_date": "2024-02-02"}}})
		case "/movie/101":
			if req.URL.Query().Get("api_key") != "db-test-key" {
				t.Fatalf("expected db api key, got %q", req.URL.Query().Get("api_key"))
			}
			if got := req.URL.Query().Get("append_to_response"); got != "credits,images,videos" {
				t.Fatalf("expected images in append_to_response, got %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 101, "title": "MovieA", "original_title": "MovieA", "overview": "Movie overview", "poster_path": "/movie-a.jpg", "backdrop_path": "/movie-a-bg.jpg", "release_date": "2024-02-02", "runtime": 121, "genres": []map[string]any{{"name": "Action"}}, "credits": map[string]any{"cast": []map[string]any{{"name": "Actor A", "character": "Lead", "profile_path": "/actor-a.jpg"}}, "crew": []map[string]any{{"name": "Director A", "job": "Director", "department": "Directing", "profile_path": "/director-a.jpg"}}}, "images": map[string]any{"logos": []map[string]any{{"file_path": "/movie-a-logo-zh.png", "iso_639_1": "zh", "vote_average": 4.5}, {"file_path": "/movie-a-logo-en.png", "iso_639_1": "en", "vote_average": 8.0}}}, "videos": map[string]any{"results": []map[string]any{{"name": "Official Trailer", "key": "abc123", "site": "YouTube", "type": "Trailer", "official": true, "iso_639_1": "en"}}}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer tmdb.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	ctx := context.Background()
	settingsSvc := settings.NewService(db, config.MetadataConfig{TMDB: config.TMDBConfig{BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}})
	if _, err := settingsSvc.UpdateMetadataSettings(ctx, settings.UpdateMetadataSettingsInput{
		TMDB: settings.MetadataProviderInput{
			APIKey:       "db-test-key",
			BaseURL:      tmdb.URL,
			ImageBaseURL: tmdb.URL + "/images",
			Language:     "en-US",
			Timeout:      "1s",
		},
	}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}

	item := database.MediaItem{LibraryID: 1, Type: "movie", Title: "MovieA", OriginalTitle: "MovieA", SourcePath: "/movies/MovieA.2024.mkv", MatchStatus: StatusPending, Status: "ready"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	if err := svc.MatchItem(ctx, item.ID); err != nil {
		t.Fatalf("match item: %v", err)
	}

	var stored database.MediaItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if stored.MatchStatus != StatusMatched || stored.MetadataProvider != "tmdb" || stored.Overview == "" {
		t.Fatalf("unexpected matched item: %#v", stored)
	}
	if stored.LogoURL != tmdb.URL+"/images/movie-a-logo-en.png" {
		t.Fatalf("unexpected logo url: %q", stored.LogoURL)
	}
	if !strings.Contains(stored.CastJSON, tmdb.URL+"/images/actor-a.jpg") || !strings.Contains(stored.CastJSON, "Lead") {
		t.Fatalf("unexpected cast json: %s", stored.CastJSON)
	}
	if !strings.Contains(stored.DirectorsJSON, tmdb.URL+"/images/director-a.jpg") || !strings.Contains(stored.DirectorsJSON, "Director") {
		t.Fatalf("unexpected directors json: %s", stored.DirectorsJSON)
	}
	if !strings.Contains(stored.TrailerJSON, "abc123") || !strings.Contains(stored.TrailerJSON, "youtube.com/embed/abc123") {
		t.Fatalf("unexpected trailer json: %s", stored.TrailerJSON)
	}
}

func TestMatchItemSelectsBestPlayableTrailer(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 101, "title": "MovieA", "original_title": "MovieA", "release_date": "2024-02-02"}}})
		case "/movie/101":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             101,
				"title":          "MovieA",
				"original_title": "MovieA",
				"overview":       "Movie overview",
				"poster_path":    "/movie-a.jpg",
				"backdrop_path":  "/movie-a-bg.jpg",
				"release_date":   "2024-02-02",
				"runtime":        121,
				"genres":         []map[string]any{{"name": "Action"}},
				"credits":        map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}},
				"images":         map[string]any{"logos": []map[string]any{}},
				"videos": map[string]any{"results": []map[string]any{
					{"name": "Featurette", "key": "skip-me", "site": "Vimeo", "type": "Featurette", "official": true},
					{"name": "Teaser", "key": "teaser-key", "site": "YouTube", "type": "Teaser", "official": true},
					{"name": "Trailer", "key": "trailer-key", "site": "YouTube", "type": "Trailer", "official": false},
					{"name": "Official Trailer", "key": "official-key", "site": "YouTube", "type": "Trailer", "official": true},
				}},
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

	ctx := context.Background()
	settingsSvc := settings.NewService(db, config.MetadataConfig{TMDB: config.TMDBConfig{BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}})
	if _, err := settingsSvc.UpdateMetadataSettings(ctx, settings.UpdateMetadataSettingsInput{TMDB: settings.MetadataProviderInput{APIKey: "db-test-key", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: "1s"}}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}

	item := database.MediaItem{LibraryID: 1, Type: "movie", Title: "MovieA", SourcePath: "/movies/MovieA.2024.mkv", MatchStatus: StatusPending, Status: "ready"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	if err := svc.MatchItem(ctx, item.ID); err != nil {
		t.Fatalf("match item: %v", err)
	}

	var stored database.MediaItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if !strings.Contains(stored.TrailerJSON, "official-key") || strings.Contains(stored.TrailerJSON, "teaser-key") {
		t.Fatalf("unexpected selected trailer json: %s", stored.TrailerJSON)
	}
}

func TestMatchItemLeavesTrailerEmptyWhenNoPlayableCandidateExists(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 101, "title": "MovieA", "original_title": "MovieA", "release_date": "2024-02-02"}}})
		case "/movie/101":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             101,
				"title":          "MovieA",
				"original_title": "MovieA",
				"overview":       "Movie overview",
				"poster_path":    "/movie-a.jpg",
				"backdrop_path":  "/movie-a-bg.jpg",
				"release_date":   "2024-02-02",
				"runtime":        121,
				"genres":         []map[string]any{{"name": "Action"}},
				"credits":        map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}},
				"images":         map[string]any{"logos": []map[string]any{}},
				"videos":         map[string]any{"results": []map[string]any{{"name": "Featurette", "key": "vimeo-only", "site": "Vimeo", "type": "Trailer", "official": true}}},
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

	ctx := context.Background()
	settingsSvc := settings.NewService(db, config.MetadataConfig{TMDB: config.TMDBConfig{BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}})
	if _, err := settingsSvc.UpdateMetadataSettings(ctx, settings.UpdateMetadataSettingsInput{TMDB: settings.MetadataProviderInput{APIKey: "db-test-key", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: "1s"}}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}

	item := database.MediaItem{LibraryID: 1, Type: "movie", Title: "MovieA", SourcePath: "/movies/MovieA.2024.mkv", MatchStatus: StatusPending, Status: "ready"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	if err := svc.MatchItem(ctx, item.ID); err != nil {
		t.Fatalf("match item: %v", err)
	}

	var stored database.MediaItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if stored.TrailerJSON != "" {
		t.Fatalf("expected empty trailer json, got %s", stored.TrailerJSON)
	}
}

func TestMatchItemPersistsRegionsAndRatingIntoSearchProjection(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 101, "title": "MovieA", "original_title": "MovieA", "release_date": "2024-02-02"}}})
		case "/movie/101":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":                   101,
				"title":                "MovieA",
				"original_title":       "MovieA",
				"overview":             "Movie overview",
				"poster_path":          "/movie-a.jpg",
				"backdrop_path":        "/movie-a-bg.jpg",
				"release_date":         "2024-02-02",
				"runtime":              121,
				"vote_average":         8.7,
				"production_countries": []map[string]any{{"name": "Japan"}, {"name": "United States"}},
				"genres":               []map[string]any{{"name": "Action"}},
				"credits":              map[string]any{"cast": []map[string]any{{"name": "Actor A"}}, "crew": []map[string]any{{"name": "Director A", "job": "Director", "department": "Directing"}}},
				"images":               map[string]any{"logos": []map[string]any{}},
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

	ctx := context.Background()
	settingsSvc := settings.NewService(db, config.MetadataConfig{TMDB: config.TMDBConfig{BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}})
	if _, err := settingsSvc.UpdateMetadataSettings(ctx, settings.UpdateMetadataSettingsInput{
		TMDB: settings.MetadataProviderInput{
			APIKey:       "db-test-key",
			BaseURL:      tmdb.URL,
			ImageBaseURL: tmdb.URL + "/images",
			Language:     "en-US",
			Timeout:      "1s",
		},
	}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}

	item := database.MediaItem{LibraryID: 1, Type: "movie", Title: "MovieA", OriginalTitle: "MovieA", SourcePath: "/movies/MovieA.2024.mkv", MatchStatus: StatusPending, Status: "ready"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	searchSvc := search.NewService(db)
	svc := NewService(db, config.MetadataConfig{}, settingsSvc, searchSvc)
	if err := svc.MatchItem(ctx, item.ID); err != nil {
		t.Fatalf("match item: %v", err)
	}

	var stored database.MediaItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if !strings.Contains(stored.RegionsJSON, "Japan") || !strings.Contains(stored.RegionsJSON, "United States") {
		t.Fatalf("expected regions_json to include countries, got %s", stored.RegionsJSON)
	}
	if stored.VoteAverage == nil || *stored.VoteAverage != 8.7 {
		t.Fatalf("expected vote_average to persist, got %#v", stored.VoteAverage)
	}

	var doc database.SearchDocument
	if err := db.WithContext(ctx).First(&doc, "media_item_id = ?", item.ID).Error; err != nil {
		t.Fatalf("load search document: %v", err)
	}
	if !strings.Contains(doc.SearchCountriesText, "Japan") || doc.VoteAverage == nil || *doc.VoteAverage != 8.7 {
		t.Fatalf("unexpected search document: %#v", doc)
	}
}

func TestSearchCandidatesReturnsHelpfulTMDBAuthError(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status_code":    7,
			"status_message": "Invalid API key: You must be granted a valid key.",
			"success":        false,
		})
	}))
	defer tmdb.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	ctx := context.Background()
	settingsSvc := settings.NewService(db, config.MetadataConfig{TMDB: config.TMDBConfig{BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}})
	if _, err := settingsSvc.UpdateMetadataSettings(ctx, settings.UpdateMetadataSettingsInput{
		TMDB: settings.MetadataProviderInput{
			APIKey:       "bad-key",
			BaseURL:      tmdb.URL,
			ImageBaseURL: tmdb.URL + "/images",
			Language:     "en-US",
			Timeout:      "1s",
		},
	}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}

	item := database.MediaItem{LibraryID: 1, Type: "movie", Title: "MovieA", SourcePath: "/movies/MovieA.2024.mkv", MatchStatus: StatusPending, Status: "ready"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	_, err = svc.SearchCandidates(ctx, item.ID, ManualSearchInput{Title: "MovieA"})
	if err == nil {
		t.Fatal("expected auth error")
	}
	if !strings.Contains(err.Error(), "TMDB 认证失败") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMatchItemSupportsTMDBBearerToken(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if got := req.Header.Get("Authorization"); got != "Bearer eyJ.test.token" {
			t.Fatalf("expected bearer token, got %q", got)
		}
		if req.URL.Query().Get("api_key") != "" {
			t.Fatalf("expected no api_key query, got %q", req.URL.Query().Get("api_key"))
		}

		switch req.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 101, "title": "MovieA", "original_title": "MovieA", "release_date": "2024-02-02"}}})
		case "/movie/101":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 101, "title": "MovieA", "original_title": "MovieA", "overview": "Movie overview", "poster_path": "/movie-a.jpg", "backdrop_path": "/movie-a-bg.jpg", "release_date": "2024-02-02", "runtime": 121, "genres": []map[string]any{{"name": "Action"}}, "credits": map[string]any{"cast": []map[string]any{{"name": "Actor A"}}, "crew": []map[string]any{{"name": "Director A", "job": "Director", "department": "Directing"}}}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer tmdb.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	ctx := context.Background()
	settingsSvc := settings.NewService(db, config.MetadataConfig{TMDB: config.TMDBConfig{BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}})
	if _, err := settingsSvc.UpdateMetadataSettings(ctx, settings.UpdateMetadataSettingsInput{
		TMDB: settings.MetadataProviderInput{
			APIKey:       "eyJ.test.token",
			BaseURL:      tmdb.URL,
			ImageBaseURL: tmdb.URL + "/images",
			Language:     "en-US",
			Timeout:      "1s",
		},
	}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}

	item := database.MediaItem{LibraryID: 1, Type: "movie", Title: "MovieA", OriginalTitle: "MovieA", SourcePath: "/movies/MovieA.2024.mkv", MatchStatus: StatusPending, Status: "ready"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	if err := svc.MatchItem(ctx, item.ID); err != nil {
		t.Fatalf("match item: %v", err)
	}
}

func TestListTVSeasonsCachesSeasonMetadata(t *testing.T) {
	tvDetailRequests := 0
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/tv/777":
			tvDetailRequests++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":   777,
				"name": "Show A",
				"seasons": []map[string]any{{
					"id":            701,
					"season_number": 1,
					"name":          "Season 1",
					"overview":      "Season overview",
					"poster_path":   "/season-1.jpg",
				}},
				"credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}},
				"images":  map[string]any{"logos": []map[string]any{}},
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

	ctx := context.Background()
	settingsSvc := settings.NewService(db, config.MetadataConfig{TMDB: config.TMDBConfig{BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}})
	if _, err := settingsSvc.UpdateMetadataSettings(ctx, settings.UpdateMetadataSettingsInput{
		TMDB: settings.MetadataProviderInput{
			APIKey:       "season-cache-key",
			BaseURL:      tmdb.URL,
			ImageBaseURL: tmdb.URL + "/images",
			Language:     "en-US",
			Timeout:      "1s",
		},
	}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	seasons, err := svc.ListTVSeasons(ctx, 777)
	if err != nil {
		t.Fatalf("list tv seasons: %v", err)
	}
	if len(seasons) != 1 || seasons[0].SeasonNumber != 1 || seasons[0].Name != "Season 1" {
		t.Fatalf("unexpected seasons: %#v", seasons)
	}
	if seasons[0].PosterURL != tmdb.URL+"/images/season-1.jpg" {
		t.Fatalf("unexpected poster url: %q", seasons[0].PosterURL)
	}

	var cached []database.TVSeasonMetadataCache
	if err := db.WithContext(ctx).Order("season_number asc").Find(&cached).Error; err != nil {
		t.Fatalf("load cached seasons: %v", err)
	}
	if len(cached) != 1 || cached[0].SeriesTMDBID != 777 || cached[0].Language != "en-US" {
		t.Fatalf("unexpected cached season rows: %#v", cached)
	}

	if _, err := svc.ListTVSeasons(ctx, 777); err != nil {
		t.Fatalf("list tv seasons from cache: %v", err)
	}
	if tvDetailRequests != 1 {
		t.Fatalf("expected single tv detail request, got %d", tvDetailRequests)
	}
}

func TestListSeasonEpisodesReusesEpisodeCache(t *testing.T) {
	seasonRequests := 0
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/tv/777/season/1":
			seasonRequests++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":            701,
				"season_number": 1,
				"name":          "Season 1",
				"overview":      "Season overview",
				"poster_path":   "/season-1.jpg",
				"episodes": []map[string]any{{
					"id":             1001,
					"season_number":  1,
					"episode_number": 1,
					"name":           "Pilot",
					"overview":       "Episode overview",
					"still_path":     "/pilot-still.jpg",
					"runtime":        48,
				}},
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

	ctx := context.Background()
	settingsSvc := settings.NewService(db, config.MetadataConfig{TMDB: config.TMDBConfig{BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}})
	if _, err := settingsSvc.UpdateMetadataSettings(ctx, settings.UpdateMetadataSettingsInput{
		TMDB: settings.MetadataProviderInput{
			APIKey:       "episode-cache-key",
			BaseURL:      tmdb.URL,
			ImageBaseURL: tmdb.URL + "/images",
			Language:     "en-US",
			Timeout:      "1s",
		},
	}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	episodes, err := svc.ListSeasonEpisodes(ctx, 777, 1, nil)
	if err != nil {
		t.Fatalf("list season episodes: %v", err)
	}
	if len(episodes) != 1 || episodes[0].EpisodeNumber != 1 || episodes[0].Name != "Pilot" {
		t.Fatalf("unexpected episodes: %#v", episodes)
	}
	if episodes[0].StillURL != tmdb.URL+"/images/pilot-still.jpg" {
		t.Fatalf("unexpected still url: %q", episodes[0].StillURL)
	}
	if episodes[0].RuntimeSeconds == nil || *episodes[0].RuntimeSeconds != 2880 {
		t.Fatalf("unexpected runtime seconds: %#v", episodes[0].RuntimeSeconds)
	}

	var cachedSeason database.TVSeasonMetadataCache
	if err := db.WithContext(ctx).First(&cachedSeason).Error; err != nil {
		t.Fatalf("load cached season: %v", err)
	}
	var cachedEpisodes []database.TVEpisodeMetadataCache
	if err := db.WithContext(ctx).Order("episode_number asc").Find(&cachedEpisodes).Error; err != nil {
		t.Fatalf("load cached episodes: %v", err)
	}
	if cachedSeason.SeriesTMDBID != 777 || len(cachedEpisodes) != 1 || cachedEpisodes[0].EpisodeNumber != 1 {
		t.Fatalf("unexpected cache rows: season=%#v episodes=%#v", cachedSeason, cachedEpisodes)
	}

	if _, err := svc.ListSeasonEpisodes(ctx, 777, 1, nil); err != nil {
		t.Fatalf("list season episodes from cache: %v", err)
	}
	if seasonRequests != 1 {
		t.Fatalf("expected single season request, got %d", seasonRequests)
	}
}

func TestMatchItemSupportsMetadataRefetch(t *testing.T) {
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
				"genres":         []map[string]any{{"name": "Action"}, {"name": "Drama"}},
				"credits": map[string]any{
					"cast": []map[string]any{{"name": "Actor A", "character": "Lead"}},
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

	ctx := context.Background()
	settingsSvc := settings.NewService(db, config.MetadataConfig{TMDB: config.TMDBConfig{BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}})
	if _, err := settingsSvc.UpdateMetadataSettings(ctx, settings.UpdateMetadataSettingsInput{
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

	confidence := 0.91
	year := 2024
	item := database.MediaItem{
		LibraryID:          1,
		Type:               "movie",
		Title:              "MovieA Stale",
		OriginalTitle:      "MovieA Stale",
		Overview:           "Old overview",
		PosterURL:          "https://old.example/poster.jpg",
		BackdropURL:        "https://old.example/backdrop.jpg",
		Year:               &year,
		SourcePath:         "/movies/MovieA.2024.mkv",
		MatchStatus:        StatusMatched,
		MetadataProvider:   "tmdb",
		ExternalID:         "movie:101",
		MetadataConfidence: &confidence,
		Status:             "ready",
	}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	if err := svc.RefetchItem(ctx, item.ID); err != nil {
		t.Fatalf("refetch item: %v", err)
	}

	var stored database.MediaItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if stored.Title != "MovieA Refetched" || stored.Overview != "Fresh metadata overview" {
		t.Fatalf("unexpected refetched item: %#v", stored)
	}
	if stored.PosterURL != tmdb.URL+"/images/movie-a-refetched.jpg" || stored.BackdropURL != tmdb.URL+"/images/movie-a-refetched-bg.jpg" {
		t.Fatalf("unexpected refetched artwork: poster=%q backdrop=%q", stored.PosterURL, stored.BackdropURL)
	}
	if stored.MetadataConfidence == nil || *stored.MetadataConfidence != confidence {
		t.Fatalf("expected metadata confidence to be preserved, got %#v", stored.MetadataConfidence)
	}
}
