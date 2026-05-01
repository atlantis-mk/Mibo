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
	if err := configureTestTMDBProvider(ctx, settingsSvc, tmdb.URL, "catalog-key"); err != nil {
		t.Fatalf("configure tmdb provider instance: %v", err)
	}

	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "MovieA", Path: "/movies/MovieA.2024.mkv", SortKey: "MovieA"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	if _, err := svc.MatchCatalogItemOperation(ctx, item.ID); err != nil {
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

func TestMatchCatalogItemOperationDocumentsAutomatedTMDBMovieBaseline(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 202, "title": "Baseline Movie", "original_title": "Baseline Original", "release_date": "2025-03-04", "vote_count": 800}}})
		case "/movie/202":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 202, "title": "Baseline Movie", "original_title": "Baseline Original", "overview": "Baseline overview", "poster_path": "/baseline-poster.jpg", "backdrop_path": "/baseline-backdrop.jpg", "release_date": "2025-03-04", "runtime": 99, "genres": []map[string]any{}, "credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}}, "images": map[string]any{"logos": []map[string]any{}}, "videos": map[string]any{"results": []map[string]any{}}})
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
	if err := configureTestTMDBProvider(ctx, settingsSvc, tmdb.URL, "catalog-key"); err != nil {
		t.Fatalf("configure tmdb provider instance: %v", err)
	}

	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Baseline Movie 2025", Path: "/movies/Baseline.Movie.2025.mkv", SortKey: "Baseline Movie"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}

	operation, err := NewService(db, config.MetadataConfig{}, settingsSvc).MatchCatalogItemOperation(ctx, item.ID)
	if err != nil {
		t.Fatalf("match catalog item: %v", err)
	}
	if operation.Operation != OperationTypeMatch || operation.OriginItemID != item.ID || operation.TargetItemID != item.ID || operation.TargetType != catalog.ItemTypeMovie || operation.Status != OperationStatusApplied {
		t.Fatalf("unexpected match operation: %#v", operation)
	}

	var stored database.CatalogItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload catalog item: %v", err)
	}
	if stored.Title != "Baseline Movie" || stored.OriginalTitle != "Baseline Original" || stored.Overview != "Baseline overview" {
		t.Fatalf("unexpected stored metadata: %#v", stored)
	}
	if stored.Year == nil || *stored.Year != 2025 || stored.RuntimeSeconds == nil || *stored.RuntimeSeconds != 99*60 {
		t.Fatalf("unexpected year/runtime: %#v", stored)
	}
	if stored.GovernanceStatus != catalog.GovernanceMatched {
		t.Fatalf("expected high-confidence movie to be matched, got %q", stored.GovernanceStatus)
	}

	var source database.MetadataSource
	if err := db.WithContext(ctx).Where("item_id = ? AND source_type = ? AND source_name = ?", item.ID, catalog.SourceTypeProvider, "tmdb").First(&source).Error; err != nil {
		t.Fatalf("load provider metadata source: %v", err)
	}
	if source.ExternalID != "movie:202" || source.ProviderInstanceName != database.MigratedDefaultTMDBProviderInstanceName {
		t.Fatalf("unexpected provider source: %#v", source)
	}

	var identity database.CatalogIdentity
	if err := db.WithContext(ctx).Where("item_id = ? AND provider = ? AND identity_type = ? AND identity_key = ?", item.ID, "tmdb", "movie", "movie:202").First(&identity).Error; err != nil {
		t.Fatalf("load provider identity: %v", err)
	}
}

func TestMatchCatalogItemSyncsTMDBMovieRichMetadata(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 303, "title": "Rich Movie", "release_date": "2026-04-05", "vote_count": 1000}}})
		case "/movie/303":
			appendToResponse := req.URL.Query().Get("append_to_response")
			if appendToResponse != "credits,images,videos,keywords,release_dates,external_ids" {
				t.Fatalf("unexpected movie append_to_response: %q", appendToResponse)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 303, "title": "Rich Movie", "overview": "Rich overview", "release_date": "2026-04-05", "runtime": 101, "vote_average": 8.4, "genres": []map[string]any{{"name": "Drama"}, {"name": "Drama"}, {"name": "Sci-Fi"}}, "keywords": map[string]any{"keywords": []map[string]any{{"name": "Space"}, {"name": "space"}, {"name": "Mystery"}}}, "release_dates": map[string]any{"results": []map[string]any{{"iso_3166_1": "US", "release_dates": []map[string]any{{"certification": "PG-13"}}}}}, "external_ids": map[string]any{"imdb_id": "tt3033030", "wikidata_id": "Q303"}, "credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}}, "images": map[string]any{"logos": []map[string]any{}}, "videos": map[string]any{"results": []map[string]any{}}})
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
	if err := configureTestTMDBProvider(ctx, settingsSvc, tmdb.URL, "catalog-key"); err != nil {
		t.Fatalf("configure tmdb provider instance: %v", err)
	}
	catalogSvc := catalog.NewService(db)
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Rich Movie", Path: "/movies/Rich.Movie.2026.mkv", SortKey: "Rich Movie"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}

	operation, err := NewService(db, config.MetadataConfig{}, settingsSvc).MatchCatalogItemOperation(ctx, item.ID)
	if err != nil {
		t.Fatalf("match catalog item: %v", err)
	}
	if !operationHasAppliedField(operation, "community_rating") || !operationHasAppliedField(operation, "official_rating") || !operationHasAppliedField(operation, "tags.genre") || !operationHasAppliedField(operation, "tags.keyword") {
		t.Fatalf("expected rich applied fields, got %#v", operation.AppliedFields)
	}

	var stored database.CatalogItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if stored.CommunityRating == nil || *stored.CommunityRating != 8.4 || stored.OfficialRating != "PG-13" {
		t.Fatalf("unexpected rich rating fields: %#v", stored)
	}

	detail, err := catalogSvc.GetItemDetail(ctx, item.ID)
	if err != nil {
		t.Fatalf("load detail: %v", err)
	}
	if len(detail.Genres) != 2 || detail.Genres[0] != "Drama" || detail.Genres[1] != "Sci-Fi" {
		t.Fatalf("unexpected genres: %#v", detail.Genres)
	}
	if !catalogTagsContain(detail.Tags, "keyword", "Space") || !catalogTagsContain(detail.Tags, "keyword", "Mystery") {
		t.Fatalf("expected keyword tags, got %#v", detail.Tags)
	}

	var imdbIdentity database.CatalogIdentity
	if err := db.WithContext(ctx).Where("item_id = ? AND provider = ? AND identity_type = ? AND identity_key = ?", item.ID, "imdb", "movie", "tt3033030").First(&imdbIdentity).Error; err != nil {
		t.Fatalf("load imdb identity: %v", err)
	}
}

func TestMatchCatalogItemUsesExistingExternalIDForDetail(t *testing.T) {
	searchCalled := false
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			searchCalled = true
			w.WriteHeader(http.StatusInternalServerError)
		case "/movie/101":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 101, "title": "Known Movie", "original_title": "Known Movie Original", "overview": "Known overview", "poster_path": "/known-poster.jpg", "backdrop_path": "/known-backdrop.jpg", "release_date": "2024-02-02", "runtime": 121, "genres": []map[string]any{}, "credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}}, "images": map[string]any{"logos": []map[string]any{}}, "videos": map[string]any{"results": []map[string]any{}}})
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
	if err := configureTestTMDBProvider(ctx, settingsSvc, tmdb.URL, "catalog-key"); err != nil {
		t.Fatalf("configure tmdb provider instance: %v", err)
	}
	catalogSvc := catalog.NewService(db)
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "MovieA", Path: "/movies/MovieA.2024.mkv", SortKey: "MovieA"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	confidence := 1.0
	if _, err := catalogSvc.SetExternalID(ctx, catalog.ExternalIDInput{ItemID: item.ID, Provider: "tmdb", ProviderType: "movie", ExternalID: "movie:101", IsPrimary: true, Source: "scanner", Confidence: &confidence}); err != nil {
		t.Fatalf("seed scanner external id: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	if _, err := svc.MatchCatalogItemOperation(ctx, item.ID); err != nil {
		t.Fatalf("match catalog item: %v", err)
	}
	if searchCalled {
		t.Fatalf("expected existing external id to skip remote search")
	}
	var stored database.CatalogItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if stored.Title != "Known Movie" {
		t.Fatalf("expected detail fetch to apply known movie, got %#v", stored)
	}
}

func TestMatchCatalogItemIgnoresLowConfidenceExistingExternalID(t *testing.T) {
	searchCalled := false
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			searchCalled = true
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 202, "title": "Back to the Past", "original_title": "Back to the Past", "release_date": "2025-02-02"}}})
		case "/movie/999":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 999, "title": "A Minecraft Movie", "release_date": "2025-04-04", "genres": []map[string]any{}, "credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}}, "images": map[string]any{"logos": []map[string]any{}}, "videos": map[string]any{"results": []map[string]any{}}})
		case "/movie/202":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 202, "title": "Back to the Past", "original_title": "Back to the Past", "overview": "Matched overview", "release_date": "2025-02-02", "runtime": 100, "genres": []map[string]any{}, "credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}}, "images": map[string]any{"logos": []map[string]any{}}, "videos": map[string]any{"results": []map[string]any{}}})
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
	if err := configureTestTMDBProvider(ctx, settingsSvc, tmdb.URL, "catalog-key"); err != nil {
		t.Fatalf("configure tmdb provider instance: %v", err)
	}
	catalogSvc := catalog.NewService(db)
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Back to the Past", Path: "/movies/Back to the Past 2025 1080p WEB-DL H 264 AAC-HHWEB.mkv", SortKey: "Back to the Past"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	confidence := 0.23
	if _, err := catalogSvc.SetExternalID(ctx, catalog.ExternalIDInput{ItemID: item.ID, Provider: "tmdb", ProviderType: "movie", ExternalID: "movie:999", IsPrimary: true, Source: "metadata_match", Confidence: &confidence}); err != nil {
		t.Fatalf("seed low-confidence external id: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	if _, err := svc.MatchCatalogItemOperation(ctx, item.ID); err != nil {
		t.Fatalf("match catalog item: %v", err)
	}
	if !searchCalled {
		t.Fatalf("expected low-confidence existing external id to be ignored and search to run")
	}
	var stored database.CatalogItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if stored.Title != "Back to the Past" {
		t.Fatalf("expected rematch to avoid stale minecraft title, got %#v", stored)
	}
}

func configureTestTMDBProvider(ctx context.Context, settingsSvc *settings.Service, baseURL string, apiKey string) error {
	enabled := true
	_, err := settingsSvc.UpsertMetadataProviderInstance(ctx, 0, settings.UpdateMetadataProviderInstanceInput{
		Name:               database.MigratedDefaultTMDBProviderInstanceName,
		ProviderType:       database.MetadataProviderTypeTMDB,
		Enabled:            &enabled,
		AvailabilityStatus: database.MetadataProviderAvailabilityAvailable,
		TMDB: &settings.MetadataProviderInput{
			APIKey:       apiKey,
			BaseURL:      baseURL,
			ImageBaseURL: baseURL + "/images",
			Language:     "en-US",
			Timeout:      "1s",
		},
	})
	return err
}

func TestMetaTubeCatalogSearchApplyAndRefetch(t *testing.T) {
	requestCounts := map[string]int{}
	metatube := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		requestCounts[req.URL.Path]++
		switch req.URL.Path {
		case "/v1/movies/search":
			if got := req.URL.Query().Get("provider"); got != "fanza" {
				t.Fatalf("unexpected metatube provider filter: %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"provider": "fanza", "id": "abc123", "title": "MetaTube Movie", "release_date": "2024-02-02", "cover_url": metatubeImageURL(req, "/poster.jpg")}}})
		case "/v1/movies/fanza/abc123":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"provider": "fanza", "id": "abc123", "title": "MetaTube Movie", "original_title": "MetaTube Original", "summary": "MetaTube overview", "release_date": "2024-02-02", "runtime": 121, "genres": []string{"Drama"}, "director": "Director One", "actors": []map[string]any{{"name": "Actor One", "role": "Lead"}}, "cover_url": metatubeImageURL(req, "/poster.jpg"), "backdrop_url": metatubeImageURL(req, "/backdrop.jpg"), "fallback": map[string]any{"used": false}}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer metatube.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	settingsSvc := settings.NewService(db, config.MetadataConfig{})
	enabled := true
	provider, err := settingsSvc.UpsertMetadataProviderInstance(ctx, 0, settings.UpdateMetadataProviderInstanceInput{Name: "metatube", ProviderType: database.MetadataProviderTypeMetaTube, Enabled: &enabled, AvailabilityStatus: database.MetadataProviderAvailabilityAvailable, MetaTube: &settings.MetadataProviderInput{APIKey: "token", BaseURL: metatube.URL, UpstreamProviderFilter: "fanza", Timeout: "1s"}})
	if err != nil {
		t.Fatalf("configure metatube provider: %v", err)
	}
	if err := database.EnsureLibraryPolicyDefaults(db, 1); err != nil {
		t.Fatalf("ensure library policy defaults: %v", err)
	}
	if _, err := settingsSvc.UpdateLibraryMetadataStrategy(ctx, 1, settings.UpdateLibraryMetadataStrategyInput{SearchProviderIDs: []uint{provider.ID}, DetailProviderIDs: []uint{provider.ID}, ImageProviderIDs: []uint{provider.ID}, PeopleProviderIDs: []uint{provider.ID}}); err != nil {
		t.Fatalf("update metadata strategy: %v", err)
	}

	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "MetaTube Movie", Path: "/movies/metatube.mkv", SortKey: "MetaTube Movie"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	candidates, err := svc.SearchCatalogCandidates(ctx, item.ID, ManualSearchInput{Title: "MetaTube Movie"})
	if err != nil {
		t.Fatalf("search metatube candidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0].Provider != "metatube" || candidates[0].ExternalID != "metatube:fanza:abc123" {
		t.Fatalf("unexpected metatube candidates: %#v", candidates)
	}
	if _, err := svc.ApplyCatalogCandidateOperation(ctx, item.ID, ApplyCandidateInput{ExternalID: candidates[0].ExternalID}); err != nil {
		t.Fatalf("apply metatube candidate: %v", err)
	}

	var externalIDs []database.CatalogExternalID
	if err := db.WithContext(ctx).Where("item_id = ?", item.ID).Order("provider asc").Find(&externalIDs).Error; err != nil {
		t.Fatalf("load external ids: %v", err)
	}
	if len(externalIDs) != 1 || externalIDs[0].Provider != "metatube" || externalIDs[0].ProviderType != "fanza" || externalIDs[0].ExternalID != "metatube:fanza:abc123" {
		t.Fatalf("unexpected metatube external identities: %#v", externalIDs)
	}
	if _, err := catalog.NewService(db).SetExternalID(ctx, catalog.ExternalIDInput{ItemID: item.ID, Provider: "tmdb", ProviderType: "movie", ExternalID: "movie:999", IsPrimary: false, Source: "test"}); err != nil {
		t.Fatalf("add tmdb identity: %v", err)
	}
	var source database.MetadataSource
	if err := db.WithContext(ctx).Where("item_id = ? AND source_name = ?", item.ID, "metatube").First(&source).Error; err != nil {
		t.Fatalf("load metatube metadata source: %v", err)
	}
	if source.ProviderInstanceName != "metatube" || source.ExternalID != "metatube:fanza:abc123" || source.PayloadJSON == "" {
		t.Fatalf("unexpected metatube source: %#v", source)
	}

	if _, err := svc.RefetchCatalogItemOperation(ctx, item.ID); err != nil {
		t.Fatalf("refetch metatube catalog item: %v", err)
	}
	if requestCounts["/v1/movies/fanza/abc123"] < 2 {
		t.Fatalf("expected refetch to call metatube detail again, counts: %#v", requestCounts)
	}

	missing, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Missing", Path: "/movies/missing.mkv", SortKey: "Missing"})
	if err != nil {
		t.Fatalf("create missing catalog item: %v", err)
	}
	if _, err := svc.RefetchCatalogItemOperation(ctx, missing.ID); err == nil {
		t.Fatalf("expected missing MetaTube identity refetch error")
	}
}

func TestMatchCatalogItemOperationDocumentsAutomatedMetaTubeMovieBaseline(t *testing.T) {
	requestCounts := map[string]int{}
	metatube := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		requestCounts[req.URL.Path]++
		switch req.URL.Path {
		case "/v1/movies/search":
			if got := req.URL.Query().Get("provider"); got != "fanza" {
				t.Fatalf("unexpected metatube provider filter: %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"provider": "fanza", "id": "auto123", "title": "Auto MetaTube Movie", "release_date": "2024-05-06", "cover_url": metatubeImageURL(req, "/auto-poster.jpg")}}})
		case "/v1/movies/fanza/auto123":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"provider": "fanza", "id": "auto123", "title": "Auto MetaTube Movie", "original_title": "Auto MetaTube Original", "summary": "Auto MetaTube overview", "release_date": "2024-05-06", "runtime": 88, "director": "Director Two", "actors": []map[string]any{{"name": "Actor Two", "role": "Lead"}}, "cover_url": metatubeImageURL(req, "/auto-poster.jpg"), "backdrop_url": metatubeImageURL(req, "/auto-backdrop.jpg")}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer metatube.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	settingsSvc := settings.NewService(db, config.MetadataConfig{})
	enabled := true
	provider, err := settingsSvc.UpsertMetadataProviderInstance(ctx, 0, settings.UpdateMetadataProviderInstanceInput{Name: "metatube-auto", ProviderType: database.MetadataProviderTypeMetaTube, Enabled: &enabled, AvailabilityStatus: database.MetadataProviderAvailabilityAvailable, MetaTube: &settings.MetadataProviderInput{BaseURL: metatube.URL, UpstreamProviderFilter: "fanza", Timeout: "1s"}})
	if err != nil {
		t.Fatalf("configure metatube provider: %v", err)
	}
	if err := database.EnsureLibraryPolicyDefaults(db, 1); err != nil {
		t.Fatalf("ensure library policy defaults: %v", err)
	}
	if _, err := settingsSvc.UpdateLibraryMetadataStrategy(ctx, 1, settings.UpdateLibraryMetadataStrategyInput{SearchProviderIDs: []uint{provider.ID}, DetailProviderIDs: []uint{provider.ID}, ImageProviderIDs: []uint{provider.ID}, PeopleProviderIDs: []uint{provider.ID}}); err != nil {
		t.Fatalf("update metadata strategy: %v", err)
	}

	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Auto MetaTube Movie", Path: "/movies/auto-metatube.mkv", SortKey: "Auto MetaTube Movie"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}

	operation, err := NewService(db, config.MetadataConfig{}, settingsSvc).MatchCatalogItemOperation(ctx, item.ID)
	if err != nil {
		t.Fatalf("match metatube movie: %v", err)
	}
	if operation.Operation != OperationTypeMatch || operation.OriginItemID != item.ID || operation.TargetItemID != item.ID || operation.TargetType != catalog.ItemTypeMovie || operation.Status != OperationStatusApplied {
		t.Fatalf("unexpected metatube match operation: %#v", operation)
	}
	if requestCounts["/v1/movies/search"] < 1 || requestCounts["/v1/movies/fanza/auto123"] != 1 {
		t.Fatalf("unexpected metatube request counts: %#v", requestCounts)
	}

	var stored database.CatalogItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload catalog item: %v", err)
	}
	if stored.Title != "Auto MetaTube Movie" || stored.OriginalTitle != "Auto MetaTube Original" || stored.Overview != "Auto MetaTube overview" || stored.RuntimeSeconds == nil || *stored.RuntimeSeconds != 88*60 {
		t.Fatalf("unexpected metatube stored item: %#v", stored)
	}
	if stored.GovernanceStatus != catalog.GovernanceMatched {
		t.Fatalf("expected metatube auto match to mark item matched, got %q", stored.GovernanceStatus)
	}

	var source database.MetadataSource
	if err := db.WithContext(ctx).Where("item_id = ? AND source_name = ?", item.ID, database.MetadataProviderTypeMetaTube).First(&source).Error; err != nil {
		t.Fatalf("load metatube metadata source: %v", err)
	}
	if source.ProviderInstanceName != "metatube-auto" || source.ExternalID != "metatube:fanza:auto123" {
		t.Fatalf("unexpected metatube source: %#v", source)
	}
}

func metatubeImageURL(req *http.Request, path string) string {
	return "http://" + req.Host + path
}

func TestRefetchCatalogItemUsesLocalScanEvidence(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	settingsSvc := settings.NewService(db, config.MetadataConfig{})
	providers, err := settingsSvc.ListMetadataProviderInstances(ctx)
	if err != nil {
		t.Fatalf("list metadata providers: %v", err)
	}
	localScanID := uint(0)
	for _, provider := range providers {
		if provider.ProviderType == database.MetadataProviderTypeLocalScan {
			localScanID = provider.ID
			break
		}
	}
	if localScanID == 0 {
		t.Fatalf("expected local scan provider id")
	}
	if err := database.EnsureLibraryPolicyDefaults(db, 1); err != nil {
		t.Fatalf("ensure library policy defaults: %v", err)
	}
	strategy, err := settingsSvc.GetLibraryMetadataStrategy(ctx, 1)
	if err != nil {
		t.Fatalf("get metadata strategy: %v", err)
	}
	_, err = settingsSvc.UpdateLibraryMetadataStrategy(ctx, 1, settings.UpdateLibraryMetadataStrategyInput{
		TemplateProfileID:         strategy.TemplateProfileID,
		SearchProviderIDs:         []uint{},
		DetailProviderIDs:         []uint{localScanID},
		ImageProviderIDs:          []uint{},
		PeopleProviderIDs:         []uint{},
		HierarchyProviderIDs:      []uint{},
		PreferredMetadataLanguage: "zh-CN",
	})
	if err != nil {
		t.Fatalf("update metadata strategy: %v", err)
	}

	catalogSvc := catalog.NewService(db)
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Original Movie", Path: "/movies/Original.Movie.2024.mkv", SortKey: "Original Movie"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	if _, applied, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: item.ID, FieldKey: "title", Value: "Locked Movie", Lock: true, LockReason: "baseline lock"}); err != nil {
		t.Fatalf("lock title field: %v", err)
	} else if !applied {
		t.Fatalf("expected title lock to apply")
	}
	payloadJSON, err := json.Marshal(map[string]any{
		"metadata_sidecars": []map[string]any{{
			"path":         "/movies/Original.Movie.2024.nfo",
			"parse_status": "parsed",
			"hints": map[string]any{
				"title":          "Sidecar Movie",
				"original_title": "Sidecar Original",
				"year":           2024,
			},
			"external_ids": map[string]any{"tmdb": "456"},
		}},
	})
	if err != nil {
		t.Fatalf("marshal scanner payload: %v", err)
	}
	if _, err := catalogSvc.RecordMetadataSource(ctx, catalog.MetadataSourceInput{ItemID: item.ID, SourceType: catalog.SourceTypeLocalFile, SourceName: "scanner", PayloadJSON: string(payloadJSON), FetchedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("record scanner metadata source: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	if _, err := svc.RefetchCatalogItemOperation(ctx, item.ID); err != nil {
		t.Fatalf("refetch catalog item with local scan: %v", err)
	}

	var stored database.CatalogItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload catalog item: %v", err)
	}
	if stored.Title != "Locked Movie" || stored.OriginalTitle != "Sidecar Original" {
		t.Fatalf("expected local scan metadata to update title fields, got %#v", stored)
	}
	if stored.Year == nil || *stored.Year != 2024 {
		t.Fatalf("expected local scan year 2024, got %#v", stored.Year)
	}
	var source database.MetadataSource
	if err := db.WithContext(ctx).Where("item_id = ? AND source_name = ?", item.ID, database.MetadataProviderTypeLocalScan).Order("id desc").First(&source).Error; err != nil {
		t.Fatalf("load local scan metadata source: %v", err)
	}
	if source.ProviderInstanceName != database.BuiltInLocalScanProviderInstanceName {
		t.Fatalf("expected local scan provider provenance, got %#v", source)
	}
}

func TestMatchCatalogItemUsesLibraryMetadataLanguagePolicy(t *testing.T) {
	seenLanguages := map[string]bool{}
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		seenLanguages[req.URL.Query().Get("language")] = true
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 101, "title": "Matched Movie", "release_date": "2024-02-02"}}})
		case "/movie/101":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 101, "title": "Matched Movie", "release_date": "2024-02-02", "runtime": 121, "genres": []map[string]any{}, "credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}}, "images": map[string]any{"logos": []map[string]any{}}, "videos": map[string]any{"results": []map[string]any{}}})
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
	if err := configureTestTMDBProvider(ctx, settingsSvc, tmdb.URL, "catalog-key"); err != nil {
		t.Fatalf("configure tmdb provider instance: %v", err)
	}
	if err := database.EnsureLibraryPolicyDefaults(db, 1); err != nil {
		t.Fatalf("ensure policy defaults: %v", err)
	}
	strategy, err := settingsSvc.GetLibraryMetadataStrategy(ctx, 1)
	if err != nil {
		t.Fatalf("get metadata strategy: %v", err)
	}
	if _, err := settingsSvc.UpdateLibraryMetadataStrategy(ctx, 1, settings.UpdateLibraryMetadataStrategyInput{TemplateProfileID: strategy.TemplateProfileID, SearchProviderIDs: strategy.SearchProviderIDs, DetailProviderIDs: strategy.DetailProviderIDs, ImageProviderIDs: strategy.ImageProviderIDs, PeopleProviderIDs: strategy.PeopleProviderIDs, HierarchyProviderIDs: strategy.HierarchyProviderIDs, PreferredMetadataLanguage: "zh-CN", PreferredImageLanguage: strategy.PreferredImageLanguage, MetadataCountryCode: strategy.MetadataCountryCode}); err != nil {
		t.Fatalf("set metadata strategy language: %v", err)
	}
	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "MovieA", Path: "/movies/MovieA.2024.mkv", SortKey: "MovieA"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	if _, err := NewService(db, config.MetadataConfig{}, settingsSvc).MatchCatalogItemOperation(ctx, item.ID); err != nil {
		t.Fatalf("match catalog item: %v", err)
	}
	if !seenLanguages["zh-CN"] {
		t.Fatalf("expected TMDB requests to use zh-CN, saw %#v", seenLanguages)
	}
}

func TestMatchCatalogItemRespectsDisabledTMDBPolicy(t *testing.T) {
	called := false
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer tmdb.Close()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	settingsSvc := settings.NewService(db, config.MetadataConfig{TMDB: config.TMDBConfig{BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}})
	if err := configureTestTMDBProvider(ctx, settingsSvc, tmdb.URL, "catalog-key"); err != nil {
		t.Fatalf("configure tmdb provider instance: %v", err)
	}
	if err := database.EnsureLibraryPolicyDefaults(db, 1); err != nil {
		t.Fatalf("ensure policy defaults: %v", err)
	}
	strategy, err := settingsSvc.GetLibraryMetadataStrategy(ctx, 1)
	if err != nil {
		t.Fatalf("get metadata strategy: %v", err)
	}
	if _, err := settingsSvc.UpdateLibraryMetadataStrategy(ctx, 1, settings.UpdateLibraryMetadataStrategyInput{TemplateProfileID: 0, SearchProviderIDs: []uint{}, DetailProviderIDs: strategy.DetailProviderIDs, ImageProviderIDs: strategy.ImageProviderIDs, PeopleProviderIDs: strategy.PeopleProviderIDs, HierarchyProviderIDs: strategy.HierarchyProviderIDs, PreferredMetadataLanguage: strategy.PreferredMetadataLanguage, PreferredImageLanguage: strategy.PreferredImageLanguage, MetadataCountryCode: strategy.MetadataCountryCode}); err != nil {
		t.Fatalf("disable search providers in metadata strategy: %v", err)
	}
	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "MovieA", Path: "/movies/MovieA.2024.mkv", SortKey: "MovieA"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	if _, err := NewService(db, config.MetadataConfig{}, settingsSvc).MatchCatalogItemOperation(ctx, item.ID); err == nil {
		t.Fatalf("expected match to fail when search stage has no configured provider")
	}
	if called {
		t.Fatalf("expected empty search strategy to avoid TMDB calls")
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
	if err := configureTestTMDBProvider(ctx, settingsSvc, tmdb.URL, "catalog-key"); err != nil {
		t.Fatalf("configure tmdb provider instance: %v", err)
	}

	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "MovieA", Path: "/movies/MovieA.2024.mkv", SortKey: "MovieA"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	if err := db.WithContext(ctx).Create([]database.ItemImage{{ItemID: item.ID, ImageType: "poster", URL: fmt.Sprintf("/api/v1/items/%d/artwork/poster", item.ID), IsSelected: true}, {ItemID: item.ID, ImageType: "backdrop", URL: fmt.Sprintf("/api/v1/items/%d/artwork/backdrop", item.ID), IsSelected: true}}).Error; err != nil {
		t.Fatalf("seed generated fallback images: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	if _, err := svc.MatchCatalogItemOperation(ctx, item.ID); err != nil {
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
	if err := configureTestTMDBProvider(ctx, settingsSvc, tmdb.URL, "catalog-key"); err != nil {
		t.Fatalf("configure tmdb provider instance: %v", err)
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
	if _, err := svc.ApplyCatalogCandidateOperation(ctx, item.ID, ApplyCandidateInput{ExternalID: "movie:202"}); err != nil {
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
	if err := configureTestTMDBProvider(ctx, settingsSvc, tmdb.URL, "catalog-key"); err != nil {
		t.Fatalf("configure tmdb provider instance: %v", err)
	}

	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Unknown Movie", Path: "/movies/Unknown.mkv", SortKey: "Unknown Movie"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}

	svc := NewService(db, config.MetadataConfig{}, settingsSvc)
	if _, err := svc.MatchCatalogItemOperation(ctx, item.ID); err != nil {
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
	if err := configureTestTMDBProvider(ctx, settingsSvc, tmdb.URL, "catalog-key"); err != nil {
		t.Fatalf("configure tmdb provider instance: %v", err)
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
	operation, err := svc.MatchCatalogItemOperation(ctx, episode.ID)
	if err != nil {
		t.Fatalf("match catalog episode via series root: %v", err)
	}
	if operation.Operation != OperationTypeMatch || operation.OriginItemID != episode.ID || operation.TargetItemID != series.ID || operation.TargetType != catalog.ItemTypeSeries || operation.Status != OperationStatusNeedsReview {
		t.Fatalf("unexpected descendant match operation: %#v", operation)
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
	var firstEpisodeIdentity database.CatalogIdentity
	if err := db.WithContext(ctx).Where("item_id = ? AND provider = ? AND identity_type = ?", episodes[0].ID, "tmdb", "tv_episode").First(&firstEpisodeIdentity).Error; err != nil {
		t.Fatalf("load episode catalog identity: %v", err)
	}
	if firstEpisodeIdentity.IdentityKey != "tv:1001" {
		t.Fatalf("unexpected episode catalog identity: %#v", firstEpisodeIdentity)
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

func TestMatchCatalogItemOperationDocumentsAutomatedTMDBSeriesHierarchyBaseline(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/tv":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 778, "name": "Baseline Show", "original_name": "Baseline Show Original", "first_air_date": "2024-01-01", "vote_count": 900}}})
		case "/tv/778":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 778, "name": "Baseline Show", "original_name": "Baseline Show Original", "overview": "Series baseline overview", "poster_path": "/show-poster.jpg", "backdrop_path": "/show-backdrop.jpg", "first_air_date": "2024-01-01", "episode_run_time": []int{42}, "seasons": []map[string]any{{"id": 1701, "season_number": 1, "name": "Season 1", "overview": "Season baseline overview", "poster_path": "/season-poster.jpg"}}, "genres": []map[string]any{}, "credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}}, "images": map[string]any{"logos": []map[string]any{}}, "videos": map[string]any{"results": []map[string]any{}}})
		case "/tv/778/season/1":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1701, "season_number": 1, "name": "Season 1", "overview": "Season baseline overview", "poster_path": "/season-poster.jpg", "episodes": []map[string]any{{"id": 2001, "season_number": 1, "episode_number": 1, "name": "Pilot", "air_date": "2024-01-01", "overview": "Pilot baseline overview", "still_path": "/pilot.jpg", "runtime": 42}}})
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
	if err := configureTestTMDBProvider(ctx, settingsSvc, tmdb.URL, "catalog-key"); err != nil {
		t.Fatalf("configure tmdb provider instance: %v", err)
	}

	catalogSvc := catalog.NewService(db)
	series, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeSeries, Title: "Baseline Show", Path: "/shows/Baseline Show", SortKey: "Baseline Show"})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}

	operation, err := NewService(db, config.MetadataConfig{}, settingsSvc).MatchCatalogItemOperation(ctx, series.ID)
	if err != nil {
		t.Fatalf("match series: %v", err)
	}
	if operation.Operation != OperationTypeMatch || operation.OriginItemID != series.ID || operation.TargetItemID != series.ID || operation.TargetType != catalog.ItemTypeSeries || operation.Status != OperationStatusApplied {
		t.Fatalf("unexpected series match operation: %#v", operation)
	}

	var storedSeries database.CatalogItem
	if err := db.WithContext(ctx).First(&storedSeries, series.ID).Error; err != nil {
		t.Fatalf("reload series: %v", err)
	}
	if storedSeries.Title != "Baseline Show" || storedSeries.GovernanceStatus != catalog.GovernanceMatched || storedSeries.RuntimeSeconds == nil || *storedSeries.RuntimeSeconds != 42*60 {
		t.Fatalf("unexpected stored series: %#v", storedSeries)
	}

	var season database.CatalogItem
	if err := db.WithContext(ctx).Where("root_id = ? AND type = ? AND index_number = ?", series.ID, catalog.ItemTypeSeason, 1).First(&season).Error; err != nil {
		t.Fatalf("load provider-created season: %v", err)
	}
	if season.Title != "Season 1" || season.AvailabilityStatus != catalog.AvailabilityMissing {
		t.Fatalf("unexpected season descendant: %#v", season)
	}

	var episode database.CatalogItem
	if err := db.WithContext(ctx).Where("root_id = ? AND type = ? AND parent_index_number = ? AND index_number = ?", series.ID, catalog.ItemTypeEpisode, 1, 1).First(&episode).Error; err != nil {
		t.Fatalf("load provider-created episode: %v", err)
	}
	if episode.Title != "Pilot" || episode.AvailabilityStatus != catalog.AvailabilityMissing || episode.RuntimeSeconds == nil || *episode.RuntimeSeconds != 42*60 {
		t.Fatalf("unexpected episode descendant: %#v", episode)
	}

	var externalIDs []database.CatalogExternalID
	if err := db.WithContext(ctx).Where("item_id IN ?", []uint{series.ID, season.ID, episode.ID}).Order("provider_type asc").Find(&externalIDs).Error; err != nil {
		t.Fatalf("load hierarchy external ids: %v", err)
	}
	if len(externalIDs) != 3 {
		t.Fatalf("expected series, season, and episode external ids, got %#v", externalIDs)
	}
}

func TestMatchCatalogItemSyncsTMDBTVRichMetadata(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/tv":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 779, "name": "Rich Show", "first_air_date": "2026-01-01", "vote_count": 1000}}})
		case "/tv/779":
			appendToResponse := req.URL.Query().Get("append_to_response")
			if appendToResponse != "credits,images,videos,keywords,content_ratings,external_ids" {
				t.Fatalf("unexpected tv append_to_response: %q", appendToResponse)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 779, "name": "Rich Show", "original_name": "Rich Show Original", "overview": "Rich series overview", "first_air_date": "2026-01-01", "last_air_date": "2026-03-01", "status": "Ended", "episode_run_time": []int{50}, "vote_average": 7.7, "genres": []map[string]any{{"name": "Drama"}}, "keywords": map[string]any{"results": []map[string]any{{"name": "Detective"}}}, "content_ratings": map[string]any{"results": []map[string]any{{"iso_3166_1": "US", "rating": "TV-MA"}}}, "external_ids": map[string]any{"imdb_id": "tt779779", "tvdb_id": 7791, "wikidata_id": "Q779"}, "seasons": []map[string]any{{"id": 2701, "season_number": 1, "name": "Season 1", "overview": "Season rich overview", "poster_path": "/season-rich.jpg"}}, "credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}}, "images": map[string]any{"logos": []map[string]any{}}, "videos": map[string]any{"results": []map[string]any{}}})
		case "/tv/779/season/1":
			if req.URL.Query().Get("append_to_response") != "credits,images,external_ids" {
				t.Fatalf("unexpected season append_to_response: %q", req.URL.Query().Get("append_to_response"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 2701, "season_number": 1, "name": "Season 1", "air_date": "2026-01-01", "overview": "Season rich overview", "poster_path": "/season-rich.jpg", "external_ids": map[string]any{"tvdb_id": 27011}, "credits": map[string]any{"cast": []map[string]any{{"id": 9301, "name": "Season Actor", "character": "Lead", "profile_path": "/season-actor.jpg"}}, "crew": []map[string]any{}}, "episodes": []map[string]any{{"id": 3001, "season_number": 1, "episode_number": 1, "name": "Pilot", "air_date": "2026-01-01", "overview": "Pilot rich overview", "still_path": "/pilot-rich.jpg", "runtime": 50, "vote_average": 8.1}}})
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
	if err := configureTestTMDBProvider(ctx, settingsSvc, tmdb.URL, "catalog-key"); err != nil {
		t.Fatalf("configure tmdb provider instance: %v", err)
	}
	catalogSvc := catalog.NewService(db)
	series, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeSeries, Title: "Rich Show", Path: "/shows/Rich Show", SortKey: "Rich Show"})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}

	operation, err := NewService(db, config.MetadataConfig{}, settingsSvc).MatchCatalogItemOperation(ctx, series.ID)
	if err != nil {
		t.Fatalf("match series: %v", err)
	}
	for _, field := range []string{"community_rating", "official_rating", "series_status", "last_air_date", "tags.genre", "tags.keyword"} {
		if !operationHasAppliedField(operation, field) {
			t.Fatalf("expected applied field %q in %#v", field, operation.AppliedFields)
		}
	}

	var storedSeries database.CatalogItem
	if err := db.WithContext(ctx).First(&storedSeries, series.ID).Error; err != nil {
		t.Fatalf("reload series: %v", err)
	}
	if storedSeries.CommunityRating == nil || *storedSeries.CommunityRating != 7.7 || storedSeries.OfficialRating != "TV-MA" || storedSeries.SeriesStatus != "Ended" || storedSeries.LastAirDate == nil {
		t.Fatalf("unexpected rich series fields: %#v", storedSeries)
	}

	var season database.CatalogItem
	if err := db.WithContext(ctx).Where("root_id = ? AND type = ? AND index_number = ?", series.ID, catalog.ItemTypeSeason, 1).First(&season).Error; err != nil {
		t.Fatalf("load season: %v", err)
	}
	if season.FirstAirDate == nil {
		t.Fatalf("expected season first air date, got %#v", season)
	}
	var seasonPersonCount int64
	if err := db.WithContext(ctx).Model(&database.ItemPerson{}).Where("item_id = ?", season.ID).Count(&seasonPersonCount).Error; err != nil {
		t.Fatalf("count season people: %v", err)
	}
	if seasonPersonCount != 1 {
		t.Fatalf("expected season person, got %d", seasonPersonCount)
	}

	var episode database.CatalogItem
	if err := db.WithContext(ctx).Where("root_id = ? AND type = ? AND parent_index_number = ? AND index_number = ?", series.ID, catalog.ItemTypeEpisode, 1, 1).First(&episode).Error; err != nil {
		t.Fatalf("load episode: %v", err)
	}
	if episode.CommunityRating == nil || *episode.CommunityRating != 8.1 {
		t.Fatalf("expected episode community rating, got %#v", episode)
	}

	detail, err := catalogSvc.GetItemDetail(ctx, series.ID)
	if err != nil {
		t.Fatalf("load series detail: %v", err)
	}
	if !catalogTagsContain(detail.Tags, "genre", "Drama") || !catalogTagsContain(detail.Tags, "keyword", "Detective") {
		t.Fatalf("expected rich series tags, got %#v", detail.Tags)
	}
	var imdbIdentity database.CatalogIdentity
	if err := db.WithContext(ctx).Where("item_id = ? AND provider = ? AND identity_type = ? AND identity_key = ?", series.ID, "imdb", "tv", "tt779779").First(&imdbIdentity).Error; err != nil {
		t.Fatalf("load series imdb identity: %v", err)
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
	if err := configureTestTMDBProvider(ctx, settingsSvc, tmdb.URL, "catalog-key"); err != nil {
		t.Fatalf("configure tmdb provider instance: %v", err)
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
	operation, err := svc.MatchCatalogItemOperation(ctx, episode.ID)
	if err != nil {
		t.Fatalf("match catalog episode: %v", err)
	}
	if operation.Operation != OperationTypeMatch || operation.OriginItemID != episode.ID || operation.TargetItemID != series.ID || operation.TargetType != catalog.ItemTypeSeries || operation.Status != OperationStatusNeedsReview {
		t.Fatalf("unexpected provider slot operation: %#v", operation)
	}
}

func operationHasAppliedField(operation MetadataOperationResult, fieldKey string) bool {
	for _, field := range operation.AppliedFields {
		if field.FieldKey == fieldKey {
			return true
		}
	}
	return false
}

func catalogTagsContain(tags []catalog.CatalogTagDetail, kind string, name string) bool {
	for _, tag := range tags {
		if tag.Kind == kind && tag.Name == name {
			return true
		}
	}
	return false
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
	if err := configureTestTMDBProvider(ctx, settingsSvc, tmdb.URL, "catalog-key"); err != nil {
		t.Fatalf("configure tmdb provider instance: %v", err)
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
	if _, err := svc.RefetchCatalogItemOperation(ctx, item.ID); err != nil {
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
