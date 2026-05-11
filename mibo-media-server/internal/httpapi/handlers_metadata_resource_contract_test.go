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
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/progress"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
	"gorm.io/gorm"
)

func TestMetadataResourceAPIContracts(t *testing.T) {
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
	playbackSvc := playback.NewService(db, registry)
	handler := New(Dependencies{
		Config:   cfg,
		DB:       db,
		Registry: registry,
		Auth:     authSvc,
		Catalog:  catalogSvc,
		Library:  librarySvc,
		Playback: playbackSvc,
		Progress: progressSvc,
		Search:   searchSvc,
		Settings: settingsSvc,
	})
	authHeader := createAuthHeader(t, ctx, authSvc)
	libraryRecord, item, resource := seedMetadataResourceAPIContract(t, ctx, db, catalogSvc)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/libraries/"+uintString(libraryRecord.ID)+"/items?type=movie", nil)
	listReq.Header.Set("Authorization", authHeader)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected library items 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var listResponse struct {
		Data []catalog.CatalogListItem `json:"data"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&listResponse); err != nil {
		t.Fatalf("decode library items: %v", err)
	}
	if len(listResponse.Data) != 1 || listResponse.Data[0].MetadataItemID != item.ID || listResponse.Data[0].ResourceCount != 1 {
		t.Fatalf("unexpected library items response: %#v", listResponse.Data)
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/api/v1/items/"+uintString(item.ID)+"?library_id="+uintString(libraryRecord.ID), nil)
	detailReq.Header.Set("Authorization", authHeader)
	detailRec := httptest.NewRecorder()
	handler.ServeHTTP(detailRec, detailReq)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("expected item detail 200, got %d: %s", detailRec.Code, detailRec.Body.String())
	}
	var detailResponse struct {
		Data catalog.CatalogItemDetail `json:"data"`
	}
	if err := json.NewDecoder(detailRec.Body).Decode(&detailResponse); err != nil {
		t.Fatalf("decode item detail: %v", err)
	}
	if detailResponse.Data.MetadataItemID != item.ID || len(detailResponse.Data.Resources) != 1 || detailResponse.Data.Resources[0].ID != resource.ID {
		t.Fatalf("unexpected item detail: %#v", detailResponse.Data)
	}

	resourcesReq := httptest.NewRequest(http.MethodGet, "/api/v1/items/"+uintString(item.ID)+"/resources?library_id="+uintString(libraryRecord.ID), nil)
	resourcesReq.Header.Set("Authorization", authHeader)
	resourcesRec := httptest.NewRecorder()
	handler.ServeHTTP(resourcesRec, resourcesReq)
	if resourcesRec.Code != http.StatusOK {
		t.Fatalf("expected resources 200, got %d: %s", resourcesRec.Code, resourcesRec.Body.String())
	}
	var resourcesResponse struct {
		Data []catalog.CatalogResourceDetail `json:"data"`
	}
	if err := json.NewDecoder(resourcesRec.Body).Decode(&resourcesResponse); err != nil {
		t.Fatalf("decode resources: %v", err)
	}
	if len(resourcesResponse.Data) != 1 || resourcesResponse.Data[0].ID != resource.ID {
		t.Fatalf("unexpected resources: %#v", resourcesResponse.Data)
	}

	favoriteReq := httptest.NewRequest(http.MethodPost, "/api/v1/me/favorites/"+uintString(item.ID), nil)
	favoriteReq.Header.Set("Authorization", authHeader)
	favoriteRec := httptest.NewRecorder()
	handler.ServeHTTP(favoriteRec, favoriteReq)
	if favoriteRec.Code != http.StatusOK {
		t.Fatalf("expected favorite 200, got %d: %s", favoriteRec.Code, favoriteRec.Body.String())
	}
	var favoriteCount int64
	if err := db.WithContext(ctx).Model(&database.UserMetadataData{}).Where("metadata_item_id = ? AND favorite = ?", item.ID, true).Count(&favoriteCount).Error; err != nil || favoriteCount != 1 {
		t.Fatalf("expected metadata favorite row, count=%d err=%v", favoriteCount, err)
	}

	searchReq := httptest.NewRequest(http.MethodGet, "/api/v1/discovery?library_id="+uintString(libraryRecord.ID)+"&q=contract&limit=10", nil)
	searchReq.Header.Set("Authorization", authHeader)
	searchRec := httptest.NewRecorder()
	handler.ServeHTTP(searchRec, searchReq)
	if searchRec.Code != http.StatusOK {
		t.Fatalf("expected discovery 200, got %d: %s", searchRec.Code, searchRec.Body.String())
	}
	if !strings.Contains(searchRec.Body.String(), "metadata_item_id") {
		t.Fatalf("expected discovery response to include metadata item semantics: %s", searchRec.Body.String())
	}
}

func seedMetadataResourceAPIContract(t *testing.T, ctx context.Context, db *gorm.DB, catalogSvc *catalog.Service) (database.Library, database.MetadataItem, database.Resource) {
	t.Helper()
	libraryRecord := database.Library{Name: "Contract Movies", Type: "movies", RootPath: "/movies", Status: "active"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Contract Movie", SortTitle: "Contract Movie", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	resource := database.Resource{StableResourceKey: "resource:contract", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, DisplayName: "Contract Movie 4K", Status: "available", ProbeStatus: "ready"}
	if err := db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	now := time.Now().UTC()
	if err := db.WithContext(ctx).Create(&database.ResourceLibraryLink{ResourceID: resource.ID, LibraryID: libraryRecord.ID, Status: "available", FirstSeenAt: now, LastSeenAt: now}).Error; err != nil {
		t.Fatalf("create resource library link: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary}).Error; err != nil {
		t.Fatalf("create resource metadata link: %v", err)
	}
	if _, err := catalogSvc.RebuildLibraryMetadataProjection(ctx, libraryRecord.ID, item.ID); err != nil {
		t.Fatalf("rebuild projection: %v", err)
	}
	if _, err := catalogSvc.RebuildMetadataSearchDocument(ctx, item.ID); err != nil {
		t.Fatalf("rebuild metadata search: %v", err)
	}
	if _, err := catalogSvc.RebuildLibrarySearchDocument(ctx, libraryRecord.ID, item.ID); err != nil {
		t.Fatalf("rebuild library search: %v", err)
	}
	return libraryRecord, item, resource
}
