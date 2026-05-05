package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/auth"
	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/metadata"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/progress"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
)

func TestCatalogGovernanceFieldHTTPReturnsMetadataOperation(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{HTTP: config.HTTPConfig{Addr: ":8080"}, Local: config.LocalStorageConfig{RootPath: t.TempDir()}, Database: config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")}}
	db, err := database.Open(cfg.Database)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, nil)
	searchSvc := search.NewService(db, librarySvc)
	progressSvc := progress.NewService(db, searchSvc)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	catalogSvc := catalog.NewService(db)
	ingestSvc := ingest.NewService(db)
	metadataSvc := metadata.NewService(db, cfg.Metadata, settingsSvc, ingestSvc)
	playbackSvc := playback.NewService(db, registry)
	if err := database.EnsureLibraryPolicyDefaults(db, 1); err != nil {
		t.Fatalf("ensure library policy defaults: %v", err)
	}
	if _, err := settingsSvc.UpdateLibraryMetadataStrategy(ctx, 1, settings.UpdateLibraryMetadataStrategyInput{}); err != nil {
		t.Fatalf("update metadata strategy: %v", err)
	}
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Unmatched", Path: "/movies/unmatched.mkv", SortKey: "Unmatched", GovernanceStatus: catalog.GovernanceUnmatched})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	handler := New(cfg, db, registry, authSvc, librarySvc, nil, playbackSvc, progressSvc, searchSvc, metadataSvc, settingsSvc, catalogSvc)
	authHeader := createAuthHeader(t, ctx, authSvc)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/items/"+uintString(item.ID)+"/governance/fields", strings.NewReader(`{"field_key":"title","value":"Manual HTTP Title"}`))
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected governance field 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Data catalog.CatalogGovernanceWorkspace `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode workspace: %v", err)
	}
	workspace := response.Data
	operation, ok := workspace.MetadataOperation.(map[string]any)
	if !ok {
		t.Fatalf("expected metadata_operation object, got %#v", workspace.MetadataOperation)
	}
	if operation["operation"] != metadata.OperationTypeManualApply || operation["status"] != metadata.OperationStatusApplied || operation["governance_status"] != catalog.GovernanceManual {
		t.Fatalf("unexpected metadata operation: %#v", operation)
	}
	if workspace.Title != "Manual HTTP Title" || workspace.GovernanceStatus != catalog.GovernanceManual {
		t.Fatalf("unexpected workspace after governance field update: %#v", workspace)
	}

	var count int64
	if err := db.WithContext(ctx).Model(&database.MetadataOperation{}).Where("target_item_id = ? AND operation = ? AND status = ?", item.ID, metadata.OperationTypeManualApply, metadata.OperationStatusApplied).Count(&count).Error; err != nil {
		t.Fatalf("count metadata operations: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one recorded metadata operation, got %d", count)
	}
}

func TestCatalogMetadataHTTPManualApplyAndRefetch(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 301, "title": "HTTP Movie", "release_date": "2024-01-02", "vote_count": 900}}})
		case "/movie/301":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 301, "title": "HTTP Movie", "overview": "HTTP overview", "release_date": "2024-01-02", "runtime": 100, "genres": []map[string]any{}, "credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}}, "images": map[string]any{"logos": []map[string]any{}}, "videos": map[string]any{"results": []map[string]any{}}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer tmdb.Close()

	ctx := context.Background()
	cfg := config.Config{HTTP: config.HTTPConfig{Addr: ":8080"}, Local: config.LocalStorageConfig{RootPath: t.TempDir()}, Database: config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")}, Metadata: config.MetadataConfig{TMDB: config.TMDBConfig{BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}}}
	db, err := database.Open(cfg.Database)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, nil)
	searchSvc := search.NewService(db, librarySvc)
	progressSvc := progress.NewService(db, searchSvc)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	catalogSvc := catalog.NewService(db)
	metadataSvc := metadata.NewService(db, cfg.Metadata, settingsSvc)
	playbackSvc := playback.NewService(db, registry)
	if err := configureHTTPTestTMDBProvider(ctx, settingsSvc, tmdb.URL); err != nil {
		t.Fatalf("configure tmdb provider: %v", err)
	}
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "HTTP Movie", Path: "/movies/http.mkv", SortKey: "HTTP Movie"})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	handler := New(cfg, db, registry, authSvc, librarySvc, nil, playbackSvc, progressSvc, searchSvc, metadataSvc, settingsSvc, catalogSvc)
	authHeader := createAuthHeader(t, ctx, authSvc)

	searchReq := httptest.NewRequest(http.MethodPost, "/api/v1/items/"+uintString(item.ID)+"/metadata/search", strings.NewReader(`{"title":"HTTP Movie"}`))
	searchReq.Header.Set("Authorization", authHeader)
	searchRec := httptest.NewRecorder()
	handler.ServeHTTP(searchRec, searchReq)
	if searchRec.Code != http.StatusOK {
		t.Fatalf("expected search 200, got %d: %s", searchRec.Code, searchRec.Body.String())
	}

	applyReq := httptest.NewRequest(http.MethodPost, "/api/v1/items/"+uintString(item.ID)+"/metadata/apply", strings.NewReader(`{"external_id":"movie:301"}`))
	applyReq.Header.Set("Authorization", authHeader)
	applyRec := httptest.NewRecorder()
	handler.ServeHTTP(applyRec, applyReq)
	if applyRec.Code != http.StatusOK {
		t.Fatalf("expected apply 200, got %d: %s", applyRec.Code, applyRec.Body.String())
	}

	refetchReq := httptest.NewRequest(http.MethodPost, "/api/v1/items/"+uintString(item.ID)+"/metadata/refetch", nil)
	refetchReq.Header.Set("Authorization", authHeader)
	refetchRec := httptest.NewRecorder()
	handler.ServeHTTP(refetchRec, refetchReq)
	if refetchRec.Code != http.StatusOK {
		t.Fatalf("expected refetch 200, got %d: %s", refetchRec.Code, refetchRec.Body.String())
	}
}

func configureHTTPTestTMDBProvider(ctx context.Context, settingsSvc *settings.Service, baseURL string) error {
	enabled := true
	_, err := settingsSvc.UpsertMetadataProviderInstance(ctx, 0, settings.UpdateMetadataProviderInstanceInput{Name: database.MigratedDefaultTMDBProviderInstanceName, ProviderType: database.MetadataProviderTypeTMDB, Enabled: &enabled, AvailabilityStatus: database.MetadataProviderAvailabilityAvailable, TMDB: &settings.MetadataProviderInput{APIKey: "http-key", BaseURL: baseURL, ImageBaseURL: baseURL + "/images", Language: "en-US", Timeout: "1s"}})
	return err
}
