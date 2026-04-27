package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/settings"
)

func TestMatchCatalogItemMatchesMovieItem(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 101, "title": "Matched Movie", "original_title": "Matched Movie Original", "release_date": "2024-02-02"}}})
		case "/movie/101":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 101, "title": "Matched Movie", "original_title": "Matched Movie Original", "overview": "Catalog movie overview", "poster_path": "/matched-movie-poster.jpg", "backdrop_path": "/matched-movie-backdrop.jpg", "release_date": "2024-02-02", "runtime": 121, "genres": []map[string]any{}, "credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}}, "images": map[string]any{"logos": []map[string]any{{"file_path": "/matched-movie-logo-en.png", "iso_639_1": "en", "vote_average": 9.0}}}, "videos": map[string]any{"results": []map[string]any{}}})
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
	if _, err := settingsSvc.UpdateMetadataSettings(ctx, settings.UpdateMetadataSettingsInput{TMDB: settings.MetadataProviderInput{APIKey: "catalog-key", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: "1s"}}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}

	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "MovieA", Path: "/movies/MovieA.2024.mkv", SortKey: "MovieA"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	if err := svc.MatchCatalogItem(ctx, item.ID); err != nil {
		t.Fatalf("match catalog item: %v", err)
	}

	var stored database.CatalogItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload catalog item: %v", err)
	}
	if stored.Title != "Matched Movie" || stored.GovernanceStatus != catalog.GovernanceNeedsReview {
		t.Fatalf("unexpected matched catalog item: %#v", stored)
	}
	if stored.Year == nil || *stored.Year != 2024 {
		t.Fatalf("expected year 2024, got %#v", stored.Year)
	}

	var externalID database.CatalogExternalID
	if err := db.WithContext(ctx).Where("item_id = ?", item.ID).First(&externalID).Error; err != nil {
		t.Fatalf("load catalog external id: %v", err)
	}
	if externalID.Provider != "tmdb" || externalID.ProviderType != "movie" || externalID.ExternalID != "movie:101" {
		t.Fatalf("unexpected catalog external id: %#v", externalID)
	}

	var source database.MetadataSource
	if err := db.WithContext(ctx).Where("item_id = ?", item.ID).First(&source).Error; err != nil {
		t.Fatalf("load metadata source: %v", err)
	}
	if source.SourceName != "tmdb" || source.ExternalID != "movie:101" {
		t.Fatalf("unexpected metadata source: %#v", source)
	}

	var images []database.ItemImage
	if err := db.WithContext(ctx).Where("item_id = ?", item.ID).Order("image_type asc, sort_order asc, id asc").Find(&images).Error; err != nil {
		t.Fatalf("load catalog images: %v", err)
	}
	if len(images) != 3 {
		t.Fatalf("expected poster/backdrop/logo images, got %#v", images)
	}
	selectedByType := make(map[string]database.ItemImage, len(images))
	for _, image := range images {
		if image.IsSelected {
			selectedByType[image.ImageType] = image
		}
	}
	if selectedByType["poster"].URL != tmdb.URL+"/images/matched-movie-poster.jpg" {
		t.Fatalf("unexpected poster image: %#v", selectedByType["poster"])
	}
	if selectedByType["backdrop"].URL != tmdb.URL+"/images/matched-movie-backdrop.jpg" {
		t.Fatalf("unexpected backdrop image: %#v", selectedByType["backdrop"])
	}
	if selectedByType["logo"].URL != tmdb.URL+"/images/matched-movie-logo-en.png" {
		t.Fatalf("unexpected logo image: %#v", selectedByType["logo"])
	}
}

func TestMatchCatalogItemPrefersRemoteImagesOverGeneratedCatalogFallback(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 101, "title": "Matched Movie", "original_title": "Matched Movie Original", "release_date": "2024-02-02"}}})
		case "/movie/101":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 101, "title": "Matched Movie", "original_title": "Matched Movie Original", "overview": "Catalog movie overview", "poster_path": "/matched-movie-poster.jpg", "backdrop_path": "/matched-movie-backdrop.jpg", "release_date": "2024-02-02", "runtime": 121, "genres": []map[string]any{}, "credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}}, "images": map[string]any{"logos": []map[string]any{}}, "videos": map[string]any{"results": []map[string]any{}}})
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
	if _, err := settingsSvc.UpdateMetadataSettings(ctx, settings.UpdateMetadataSettingsInput{TMDB: settings.MetadataProviderInput{APIKey: "catalog-key", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: "1s"}}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}

	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "MovieA", Path: "/movies/MovieA.2024.mkv", SortKey: "MovieA"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	if err := db.WithContext(ctx).Create([]database.ItemImage{{ItemID: item.ID, ImageType: "poster", URL: fmt.Sprintf("/api/v1/items/%d/artwork/poster", item.ID), IsSelected: true}, {ItemID: item.ID, ImageType: "backdrop", URL: fmt.Sprintf("/api/v1/items/%d/artwork/backdrop", item.ID), IsSelected: true}}).Error; err != nil {
		t.Fatalf("seed generated fallback images: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	if err := svc.MatchCatalogItem(ctx, item.ID); err != nil {
		t.Fatalf("match catalog item: %v", err)
	}

	var images []database.ItemImage
	if err := db.WithContext(ctx).Where("item_id = ?", item.ID).Order("image_type asc, id asc").Find(&images).Error; err != nil {
		t.Fatalf("load catalog images: %v", err)
	}
	selectedByType := make(map[string]database.ItemImage, len(images))
	for _, image := range images {
		if image.IsSelected {
			selectedByType[image.ImageType] = image
		}
	}
	if selectedByType["poster"].URL != tmdb.URL+"/images/matched-movie-poster.jpg" {
		t.Fatalf("expected remote poster to replace generated fallback, got %#v", selectedByType["poster"])
	}
	if selectedByType["backdrop"].URL != tmdb.URL+"/images/matched-movie-backdrop.jpg" {
		t.Fatalf("expected remote backdrop to replace generated fallback, got %#v", selectedByType["backdrop"])
	}
}

func TestApplyCatalogCandidateReplacesPreviouslySelectedRemoteImages(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/movie/202":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 202, "title": "Updated Match", "original_title": "Updated Match Original", "overview": "Updated overview", "poster_path": "/updated-poster.jpg", "backdrop_path": "/updated-backdrop.jpg", "release_date": "2025-03-02", "runtime": 118, "genres": []map[string]any{}, "credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}}, "images": map[string]any{"logos": []map[string]any{{"file_path": "/updated-logo-en.png", "iso_639_1": "en", "vote_average": 9.0}}}, "videos": map[string]any{"results": []map[string]any{}}})
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
	if _, err := settingsSvc.UpdateMetadataSettings(ctx, settings.UpdateMetadataSettingsInput{TMDB: settings.MetadataProviderInput{APIKey: "catalog-key", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: "1s"}}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}

	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "MovieA", Path: "/movies/MovieA.2024.mkv", SortKey: "MovieA"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	if err := db.WithContext(ctx).Create([]database.ItemImage{
		{ItemID: item.ID, ImageType: "poster", URL: tmdb.URL + "/images/old-poster.jpg", IsSelected: true},
		{ItemID: item.ID, ImageType: "backdrop", URL: tmdb.URL + "/images/old-backdrop.jpg", IsSelected: true},
		{ItemID: item.ID, ImageType: "logo", URL: tmdb.URL + "/images/old-logo.png", IsSelected: true},
	}).Error; err != nil {
		t.Fatalf("seed existing selected images: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	if err := svc.ApplyCatalogCandidate(ctx, item.ID, ApplyCandidateInput{ExternalID: "movie:202"}); err != nil {
		t.Fatalf("apply catalog candidate: %v", err)
	}

	var images []database.ItemImage
	if err := db.WithContext(ctx).Where("item_id = ?", item.ID).Order("image_type asc, sort_order asc, id asc").Find(&images).Error; err != nil {
		t.Fatalf("load catalog images: %v", err)
	}
	selectedByType := make(map[string]database.ItemImage, len(images))
	for _, image := range images {
		if image.IsSelected {
			selectedByType[image.ImageType] = image
		}
	}
	if selectedByType["poster"].URL != tmdb.URL+"/images/updated-poster.jpg" {
		t.Fatalf("expected selected poster to switch to applied candidate, got %#v", selectedByType["poster"])
	}
	if selectedByType["backdrop"].URL != tmdb.URL+"/images/updated-backdrop.jpg" {
		t.Fatalf("expected selected backdrop to switch to applied candidate, got %#v", selectedByType["backdrop"])
	}
	if selectedByType["logo"].URL != tmdb.URL+"/images/updated-logo-en.png" {
		t.Fatalf("expected selected logo to switch to applied candidate, got %#v", selectedByType["logo"])
	}
}

func TestMatchCatalogItemMarksItemUnmatchedWhenSearchReturnsNoResults(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{}})
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
	if _, err := settingsSvc.UpdateMetadataSettings(ctx, settings.UpdateMetadataSettingsInput{TMDB: settings.MetadataProviderInput{APIKey: "catalog-key", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: "1s"}}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}

	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Unknown Movie", Path: "/movies/Unknown.mkv", SortKey: "Unknown Movie"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	if err := svc.MatchCatalogItem(ctx, item.ID); err != nil {
		t.Fatalf("match catalog item: %v", err)
	}

	var stored database.CatalogItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload catalog item: %v", err)
	}
	if stored.GovernanceStatus != catalog.GovernanceUnmatched {
		t.Fatalf("expected unmatched governance state, got %#v", stored)
	}
}

func TestMatchCatalogItemRoutesEpisodeToSeriesRoot(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/tv":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 777, "name": "Matched Show", "original_name": "Matched Show Original", "first_air_date": "2024-01-01"}}})
		case "/tv/777":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 777, "name": "Matched Show", "original_name": "Matched Show Original", "overview": "Series overview", "poster_path": "/matched-show-poster.jpg", "backdrop_path": "/matched-show-backdrop.jpg", "first_air_date": "2024-01-01", "episode_run_time": []int{45}, "seasons": []map[string]any{{"id": 701, "season_number": 1, "name": "Season 1", "overview": "Season overview", "poster_path": "/matched-season-1.jpg"}}, "genres": []map[string]any{}, "credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}}, "images": map[string]any{"logos": []map[string]any{{"file_path": "/matched-show-logo-en.png", "iso_639_1": "en", "vote_average": 9.0}}}, "videos": map[string]any{"results": []map[string]any{}}})
		case "/tv/777/season/1":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 701, "season_number": 1, "name": "Season 1", "overview": "Season overview", "poster_path": "/matched-season-1.jpg", "episodes": []map[string]any{{"id": 1001, "season_number": 1, "episode_number": 1, "name": "Pilot", "air_date": "2024-01-01", "overview": "Pilot overview", "still_path": "/pilot-still.jpg", "runtime": 45}, {"id": 1002, "season_number": 1, "episode_number": 2, "name": "Second Episode", "air_date": "2024-01-08", "overview": "Second overview", "still_path": "/second-still.jpg", "runtime": 47, "crew": []map[string]any{{"id": 9201, "name": "Episode Director", "job": "Director", "department": "Directing", "profile_path": "/director.jpg"}}, "guest_stars": []map[string]any{{"id": 9101, "name": "Guest Actor", "character": "Guest", "profile_path": "/guest.jpg"}}}}})
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
	if _, err := settingsSvc.UpdateMetadataSettings(ctx, settings.UpdateMetadataSettingsInput{TMDB: settings.MetadataProviderInput{APIKey: "catalog-key", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: "1s"}}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}

	catalogSvc := catalog.NewService(db)
	series, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeSeries, Title: "Show A", Path: "/shows/ShowA", SortKey: "Show A"})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", Path: "/shows/ShowA/Season 1", SortKey: "Show A S01", IndexNumber: &seasonNumber, ParentIndexNumber: &seasonNumber})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	episodeOneNumber := 1
	looseEpisodeOne, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeEpisode, ParentID: &series.ID, Title: "Local Episode 1", Path: "/shows/ShowA/ShowA.S01E01.mkv", SortKey: "Show A S01E01", IndexNumber: &episodeOneNumber, ParentIndexNumber: &seasonNumber})
	if err != nil {
		t.Fatalf("create loose episode one: %v", err)
	}
	episodeNumber := 2
	episode, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 2", Path: "/shows/ShowA/Season 1/ShowA.S01E02.mkv", SortKey: "Show A S01E02", IndexNumber: &episodeNumber, ParentIndexNumber: &seasonNumber})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	file := database.InventoryFile{LibraryID: 1, StorageProvider: "local", StoragePath: episode.Path, Status: "available"}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	asset := database.MediaAsset{LibraryID: 1, AssetType: "main", Status: "available", ProbeStatus: "complete"}
	if err := db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create media asset: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: "source", PartIndex: 0}).Error; err != nil {
		t.Fatalf("create asset file link: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.AssetItem{AssetID: asset.ID, ItemID: episode.ID, Role: "primary", SegmentIndex: 0, Source: "scanner"}).Error; err != nil {
		t.Fatalf("create asset item link: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	result, err := svc.MatchCatalogItemWithResult(ctx, episode.ID)
	if err != nil {
		t.Fatalf("match catalog episode via series root: %v", err)
	}
	if result.OriginItemID != episode.ID || result.TargetItemID != series.ID || result.DescendantStatus != "identity_retained" || result.ProviderExternalID != "tv:1002" {
		t.Fatalf("unexpected descendant match result: %#v", result)
	}

	var storedSeries database.CatalogItem
	if err := db.WithContext(ctx).First(&storedSeries, series.ID).Error; err != nil {
		t.Fatalf("reload series: %v", err)
	}
	if storedSeries.Title != "Matched Show" || storedSeries.GovernanceStatus != catalog.GovernanceNeedsReview {
		t.Fatalf("unexpected matched series root: %#v", storedSeries)
	}

	var seriesExternalIDs []database.CatalogExternalID
	if err := db.WithContext(ctx).Where("item_id = ?", series.ID).Find(&seriesExternalIDs).Error; err != nil {
		t.Fatalf("list series external ids: %v", err)
	}
	if len(seriesExternalIDs) != 1 || seriesExternalIDs[0].ProviderType != "tv" || seriesExternalIDs[0].ExternalID != "tv:777" {
		t.Fatalf("unexpected series external ids: %#v", seriesExternalIDs)
	}

	var episodeExternalIDs int64
	if err := db.WithContext(ctx).Model(&database.CatalogExternalID{}).Where("item_id = ?", episode.ID).Count(&episodeExternalIDs).Error; err != nil {
		t.Fatalf("count episode external ids: %v", err)
	}
	if episodeExternalIDs != 1 {
		t.Fatalf("expected existing local episode to gain one descendant identity, got %d rows", episodeExternalIDs)
	}

	var episodes []database.CatalogItem
	if err := db.WithContext(ctx).Where("parent_id = ? AND type = ?", season.ID, catalog.ItemTypeEpisode).Order("index_number asc").Find(&episodes).Error; err != nil {
		t.Fatalf("list synced episodes: %v", err)
	}
	if len(episodes) != 2 {
		t.Fatalf("expected provider sync to create missing episode rows, got %#v", episodes)
	}
	if episodes[0].ID != looseEpisodeOne.ID || episodes[0].IndexNumber == nil || *episodes[0].IndexNumber != 1 || episodes[0].AvailabilityStatus != catalog.AvailabilityMissing {
		t.Fatalf("expected existing loose episode 1 to be reparented and enriched as missing, got %#v", episodes[0])
	}
	if episodes[1].ID != episode.ID || episodes[1].AvailabilityStatus != catalog.AvailabilityAvailable {
		t.Fatalf("expected existing episode with local asset to be reused and stay available, got %#v", episodes[1])
	}

	var seasonExternalID database.CatalogExternalID
	if err := db.WithContext(ctx).Where("item_id = ? AND provider = ? AND provider_type = ?", season.ID, "tmdb", "tv_season").First(&seasonExternalID).Error; err != nil {
		t.Fatalf("load season external id: %v", err)
	}
	if seasonExternalID.ExternalID != "tv:701" {
		t.Fatalf("unexpected season external id: %#v", seasonExternalID)
	}

	var seasonSource database.MetadataSource
	if err := db.WithContext(ctx).Where("item_id = ? AND external_id = ?", season.ID, "tv:701").First(&seasonSource).Error; err != nil {
		t.Fatalf("load season metadata source: %v", err)
	}

	var seasonImages []database.ItemImage
	if err := db.WithContext(ctx).Where("item_id = ?", season.ID).Find(&seasonImages).Error; err != nil {
		t.Fatalf("load season images: %v", err)
	}
	if len(seasonImages) == 0 || seasonImages[0].URL != tmdb.URL+"/images/matched-season-1.jpg" || !seasonImages[0].IsSelected {
		t.Fatalf("unexpected season images: %#v", seasonImages)
	}
	if seasonImages[0].SourceID == nil || *seasonImages[0].SourceID != seasonSource.ID {
		t.Fatalf("expected season image provenance to point at metadata source %d, got %#v", seasonSource.ID, seasonImages[0])
	}

	var firstEpisodeExternalID database.CatalogExternalID
	if err := db.WithContext(ctx).Where("item_id = ? AND provider = ? AND provider_type = ?", episodes[0].ID, "tmdb", "tv_episode").First(&firstEpisodeExternalID).Error; err != nil {
		t.Fatalf("load episode external id: %v", err)
	}
	if firstEpisodeExternalID.ExternalID != "tv:1001" {
		t.Fatalf("unexpected episode external id: %#v", firstEpisodeExternalID)
	}

	var firstEpisodeSource database.MetadataSource
	if err := db.WithContext(ctx).Where("item_id = ? AND external_id = ?", episodes[0].ID, "tv:1001").First(&firstEpisodeSource).Error; err != nil {
		t.Fatalf("load episode metadata source: %v", err)
	}

	var firstEpisodeImages []database.ItemImage
	if err := db.WithContext(ctx).Where("item_id = ?", episodes[0].ID).Find(&firstEpisodeImages).Error; err != nil {
		t.Fatalf("load episode images: %v", err)
	}
	if len(firstEpisodeImages) == 0 || firstEpisodeImages[0].URL != tmdb.URL+"/images/pilot-still.jpg" || !firstEpisodeImages[0].IsSelected {
		t.Fatalf("unexpected episode images: %#v", firstEpisodeImages)
	}
	if firstEpisodeImages[0].SourceID == nil || *firstEpisodeImages[0].SourceID != firstEpisodeSource.ID {
		t.Fatalf("expected episode image provenance to point at metadata source %d, got %#v", firstEpisodeSource.ID, firstEpisodeImages[0])
	}
	if episodes[0].FirstAirDate == nil || !episodes[0].FirstAirDate.Equal(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected episode air date to persist, got %#v", episodes[0].FirstAirDate)
	}

	var episodePeople []struct {
		RelationRole string
		Character    string
		Name         string
		AvatarURL    string
		TMDBPersonID *int
	}
	if err := db.WithContext(ctx).
		Table("item_people").
		Select("item_people.role AS relation_role, item_people.character, people.name, people.avatar_url, people.tmdb_person_id").
		Joins("JOIN people ON people.id = item_people.person_id").
		Where("item_people.item_id = ?", episode.ID).
		Order("item_people.role asc, people.name asc").
		Scan(&episodePeople).Error; err != nil {
		t.Fatalf("load episode people: %v", err)
	}
	if len(episodePeople) != 2 || episodePeople[0].Name != "Guest Actor" || episodePeople[0].Character != "Guest" || episodePeople[0].TMDBPersonID == nil || *episodePeople[0].TMDBPersonID != 9101 || episodePeople[1].Name != "Episode Director" || episodePeople[1].AvatarURL != tmdb.URL+"/images/director.jpg" || episodePeople[1].TMDBPersonID == nil || *episodePeople[1].TMDBPersonID != 9201 {
		t.Fatalf("unexpected episode people: %#v", episodePeople)
	}

	detail, err := catalogSvc.GetItemDetail(ctx, episodes[0].ID)
	if err != nil {
		t.Fatalf("load episode detail: %v", err)
	}
	if len(detail.SourceEvidence) == 0 {
		t.Fatalf("expected episode detail to expose descendant source evidence, got %#v", detail)
	}
	summary, ok := detail.SourceEvidence[0].Summary.(map[string]any)
	if !ok {
		t.Fatalf("expected curated descendant source summary map, got %#v", detail.SourceEvidence[0].Summary)
	}
	for key, want := range map[string]any{
		"matched_title":  "Pilot",
		"air_date":       "2024-01-01",
		"still_path":     "/pilot-still.jpg",
		"series_tmdb_id": float64(777),
	} {
		if got := summary[key]; got != want {
			t.Fatalf("expected episode source summary %q=%#v, got %#v", key, want, got)
		}
	}
}

func TestMatchCatalogEpisodeReportsProviderSlotMissing(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/tv":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 777, "name": "Matched Show", "first_air_date": "2024-01-01"}}})
		case "/tv/777":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 777, "name": "Matched Show", "first_air_date": "2024-01-01", "seasons": []map[string]any{{"id": 701, "season_number": 1, "name": "Season 1"}}, "genres": []map[string]any{}, "credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}}, "images": map[string]any{"logos": []map[string]any{}}, "videos": map[string]any{"results": []map[string]any{}}})
		case "/tv/777/season/1":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 701, "season_number": 1, "name": "Season 1", "episodes": []map[string]any{{"id": 1001, "season_number": 1, "episode_number": 1, "name": "Pilot", "air_date": "2024-01-01"}}})
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
	if _, err := settingsSvc.UpdateMetadataSettings(ctx, settings.UpdateMetadataSettingsInput{TMDB: settings.MetadataProviderInput{APIKey: "catalog-key", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: "1s"}}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}

	catalogSvc := catalog.NewService(db)
	series, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeSeries, Title: "Show A", Path: "/shows/ShowA", SortKey: "Show A"})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", Path: "/shows/ShowA/Season 1", SortKey: "Show A S01", IndexNumber: &seasonNumber, ParentIndexNumber: &seasonNumber})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	episodeNumber := 99
	episode, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Title: "Bad Slot", Path: "/shows/ShowA/Season 1/ShowA.S01E99.mkv", SortKey: "Show A S01E99", IndexNumber: &episodeNumber, ParentIndexNumber: &seasonNumber})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	result, err := svc.MatchCatalogItemWithResult(ctx, episode.ID)
	if err != nil {
		t.Fatalf("match catalog episode: %v", err)
	}
	if result.DescendantStatus != "provider_slot_missing" || result.SeasonNumber == nil || *result.SeasonNumber != 1 || result.EpisodeNumber == nil || *result.EpisodeNumber != 99 {
		t.Fatalf("unexpected provider slot result: %#v", result)
	}
}

func TestRefetchCatalogItemRespectsLockedFields(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/movie/101":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 101, "title": "Provider Title", "original_title": "Provider Original", "overview": "Fresh overview", "release_date": "2024-02-02", "runtime": 121, "genres": []map[string]any{}, "credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}}, "images": map[string]any{"logos": []map[string]any{}}, "videos": map[string]any{"results": []map[string]any{}}})
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
	if _, err := settingsSvc.UpdateMetadataSettings(ctx, settings.UpdateMetadataSettingsInput{TMDB: settings.MetadataProviderInput{APIKey: "catalog-key", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: "1s"}}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}

	catalogSvc := catalog.NewService(db)
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Original Title", Path: "/movies/MovieA.2024.mkv", SortKey: "MovieA"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: item.ID, FieldKey: "title", Value: "Locked Title", Lock: true, LockReason: "user override"}); err != nil {
		t.Fatalf("lock title field: %v", err)
	}
	confidence := 1.0
	if _, err := catalogSvc.SetExternalID(ctx, catalog.ExternalIDInput{ItemID: item.ID, Provider: "tmdb", ProviderType: "movie", ExternalID: "movie:101", IsPrimary: true, Source: "test", Confidence: &confidence}); err != nil {
		t.Fatalf("seed catalog external id: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	if err := svc.RefetchCatalogItem(ctx, item.ID); err != nil {
		t.Fatalf("refetch catalog item: %v", err)
	}

	var stored database.CatalogItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload catalog item: %v", err)
	}
	if stored.Title != "Locked Title" {
		t.Fatalf("expected locked title to remain, got %#v", stored)
	}
	if stored.Overview != "Fresh overview" {
		t.Fatalf("expected unlocked overview to refresh, got %#v", stored)
	}
	if stored.GovernanceStatus != catalog.GovernanceMatched {
		t.Fatalf("expected refetch with canonical identity to stay matched, got %#v", stored)
	}

	var source database.MetadataSource
	if err := db.WithContext(ctx).Where("item_id = ?", item.ID).First(&source).Error; err != nil {
		t.Fatalf("load refreshed metadata source: %v", err)
	}
	if source.ExternalID != "movie:101" || source.SourceName != "tmdb" {
		t.Fatalf("unexpected refreshed metadata source: %#v", source)
	}

	var titleState database.MetadataFieldState
	if err := db.WithContext(ctx).Where("item_id = ? AND field_key = ?", item.ID, "title").First(&titleState).Error; err != nil {
		t.Fatalf("load title field state: %v", err)
	}
	if !titleState.IsLocked {
		t.Fatalf("expected title field to remain locked, got %#v", titleState)
	}
}
