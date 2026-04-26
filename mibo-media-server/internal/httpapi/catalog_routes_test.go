package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/auth"
	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/metadata"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/progress"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
	"gorm.io/gorm"
)

func TestCatalogLibraryItemsRouteUsesCatalogWhenReadEnabled(t *testing.T) {
	router, db, _, _, settingsSvc, catalogSvc, libraryID := newCatalogRouteHarness(t, nil)
	ctx := context.Background()
	if _, err := settingsSvc.UpdateCatalogMigrationState(ctx, settings.UpdateCatalogMigrationStateInput{CatalogReadEnabled: true}); err != nil {
		t.Fatalf("enable catalog reads: %v", err)
	}
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeSeries, Title: "Show A", Path: "/library/ShowA", SortKey: "Show A", AvailabilityStatus: catalog.AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ItemImage{ItemID: item.ID, ImageType: "poster", URL: "/poster.jpg", IsSelected: true}).Error; err != nil {
		t.Fatalf("create selected image: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/libraries/%d/items?type=show", libraryID), nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Data []catalog.CatalogListItem `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Type != catalog.ItemTypeSeries {
		t.Fatalf("unexpected catalog list response: %#v", response.Data)
	}
	if response.Data[0].SelectedImages[0].URL != requestBaseURL(request)+"/poster.jpg" {
		t.Fatalf("expected normalized selected image url, got %#v", response.Data[0].SelectedImages)
	}
}

func TestCatalogLibraryItemsRouteFallsBackToCatalogWhenLegacyEmpty(t *testing.T) {
	router, _, _, _, _, catalogSvc, libraryID := newCatalogRouteHarness(t, nil)
	ctx := context.Background()
	if _, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeMovie, Title: "Movie A", Path: "/library/MovieA.2024.mkv", SortKey: "Movie A", AvailabilityStatus: catalog.AvailabilityAvailable}); err != nil {
		t.Fatalf("create catalog item: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/libraries/%d/items?type=movie", libraryID), nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Data []catalog.CatalogListItem `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Type != catalog.ItemTypeMovie {
		t.Fatalf("expected catalog fallback item, got %#v", response.Data)
	}
}

func TestCatalogDiscoveryRouteUsesCatalogSearchDocumentsWhenReadEnabled(t *testing.T) {
	router, db, authSvc, _, settingsSvc, catalogSvc, libraryID := newCatalogRouteHarness(t, nil)
	ctx := context.Background()
	authHeader := createAuthHeader(t, ctx, authSvc)
	if _, err := settingsSvc.UpdateCatalogMigrationState(ctx, settings.UpdateCatalogMigrationStateInput{CatalogReadEnabled: true}); err != nil {
		t.Fatalf("enable catalog reads: %v", err)
	}
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeMovie, Title: "Movie A", Path: "/library/MovieA.2024.mkv", SortKey: "Movie A", AvailabilityStatus: catalog.AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	if err := catalogSvc.RefreshItemProjection(ctx, item.ID); err != nil {
		t.Fatalf("refresh item projection: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ItemImage{ItemID: item.ID, ImageType: "poster", URL: "/movie-poster.jpg", IsSelected: true}).Error; err != nil {
		t.Fatalf("create selected image: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/discovery?q=Movie", nil)
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			Items []catalog.CatalogListItem `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode discovery response: %v", err)
	}
	if len(response.Data.Items) != 1 || response.Data.Items[0].ID != item.ID {
		t.Fatalf("unexpected discovery items: %#v", response.Data.Items)
	}
}

func TestCatalogDiscoveryRouteFallsBackToCatalogWhenLegacySearchEmpty(t *testing.T) {
	router, _, authSvc, _, _, catalogSvc, libraryID := newCatalogRouteHarness(t, nil)
	ctx := context.Background()
	authHeader := createAuthHeader(t, ctx, authSvc)
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeMovie, Title: "Movie A", Path: "/library/MovieA.2024.mkv", SortKey: "Movie A", AvailabilityStatus: catalog.AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	if err := catalogSvc.RefreshItemProjection(ctx, item.ID); err != nil {
		t.Fatalf("refresh item projection: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/discovery?scope=library&library_id=%d&q=Movie", libraryID), nil)
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Data struct {
			Items []catalog.CatalogListItem `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data.Items) != 1 || response.Data.Items[0].ID != item.ID {
		t.Fatalf("expected catalog search fallback item, got %#v", response.Data.Items)
	}
}

func TestCatalogRecentlyAddedRouteFallsBackWhenLegacyEmpty(t *testing.T) {
	router, _, _, _, _, catalogSvc, libraryID := newCatalogRouteHarness(t, nil)
	ctx := context.Background()
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeMovie, Title: "Movie A", Path: "/library/MovieA.2024.mkv", SortKey: "Movie A", AvailabilityStatus: catalog.AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/home/recently-added?limit=6", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Data []catalog.CatalogListItem `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].ID != item.ID {
		t.Fatalf("expected catalog recently-added fallback, got %#v", response.Data)
	}
}

func TestCatalogLatestByLibraryRouteFallsBackWhenLegacyEmpty(t *testing.T) {
	router, db, authSvc, _, _, catalogSvc, libraryID := newCatalogRouteHarness(t, nil)
	ctx := context.Background()
	authHeader := createAuthHeader(t, ctx, authSvc)
	if err := db.WithContext(ctx).Model(&database.Library{}).Where("id = ?", libraryID).Update("status", "active").Error; err != nil {
		t.Fatalf("activate library: %v", err)
	}
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeMovie, Title: "Movie A", Path: "/library/MovieA.2024.mkv", SortKey: "Movie A", AvailabilityStatus: catalog.AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/home/latest-by-library", nil)
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Data []catalog.CatalogLatestByLibrarySection `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].LibraryID != libraryID || len(response.Data[0].Items) != 1 || response.Data[0].Items[0].ID != item.ID {
		t.Fatalf("expected catalog latest-by-library fallback, got %#v", response.Data)
	}
}

func TestCatalogItemAndGovernanceRoutes(t *testing.T) {
	router, db, authSvc, _, _, catalogSvc, libraryID := newCatalogRouteHarness(t, nil)
	ctx := context.Background()
	authHeader := createAuthHeader(t, ctx, authSvc)

	series, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeSeries, Title: "Show A", Path: "/library/ShowA", SortKey: "Show A", AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernanceNeedsReview})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", Path: "/library/ShowA/Season 1", SortKey: "Show A S01", IndexNumber: &seasonNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: catalog.AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	episodeNumber := 2
	if _, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 2", Path: "/library/ShowA/Season 1/ShowA.S01E02.mkv", SortKey: "Show A S01E02", IndexNumber: &episodeNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: catalog.AvailabilityAvailable}); err != nil {
		t.Fatalf("create episode: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ItemImage{ItemID: series.ID, ImageType: "poster", URL: "/series-poster.jpg", IsSelected: true}).Error; err != nil {
		t.Fatalf("create image: %v", err)
	}
	if _, err := catalogSvc.RecordMetadataSource(ctx, catalog.MetadataSourceInput{ItemID: series.ID, SourceType: catalog.SourceTypeProvider, SourceName: "tmdb", ExternalID: "tv:777", PayloadJSON: `{"title":"Show A"}`}); err != nil {
		t.Fatalf("record metadata source: %v", err)
	}
	if _, err := catalogSvc.SetExternalID(ctx, catalog.ExternalIDInput{ItemID: series.ID, Provider: "tmdb", ProviderType: "tv", ExternalID: "tv:777", IsPrimary: true}); err != nil {
		t.Fatalf("set external id: %v", err)
	}

	detailRecorder := httptest.NewRecorder()
	detailRequest := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/items/%d", series.ID), nil)
	router.ServeHTTP(detailRecorder, detailRequest)
	if detailRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for item detail, got %d body=%s", detailRecorder.Code, detailRecorder.Body.String())
	}
	var detailResponse struct {
		Data catalog.CatalogItemDetail `json:"data"`
	}
	if err := json.Unmarshal(detailRecorder.Body.Bytes(), &detailResponse); err != nil {
		t.Fatalf("decode detail response: %v", err)
	}
	if len(detailResponse.Data.Seasons) != 1 || len(detailResponse.Data.Seasons[0].Episodes) != 1 {
		t.Fatalf("unexpected item detail payload: %#v", detailResponse.Data)
	}

	seasonsRecorder := httptest.NewRecorder()
	seasonsRequest := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/series/%d/seasons", series.ID), nil)
	router.ServeHTTP(seasonsRecorder, seasonsRequest)
	if seasonsRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for series seasons, got %d body=%s", seasonsRecorder.Code, seasonsRecorder.Body.String())
	}
	var seasonsResponse struct {
		Data []catalog.CatalogSeasonDetail `json:"data"`
	}
	if err := json.Unmarshal(seasonsRecorder.Body.Bytes(), &seasonsResponse); err != nil {
		t.Fatalf("decode seasons response: %v", err)
	}
	if len(seasonsResponse.Data) != 1 || len(seasonsResponse.Data[0].Episodes) != 1 {
		t.Fatalf("unexpected series seasons response: %#v", seasonsResponse.Data)
	}

	workspaceRecorder := httptest.NewRecorder()
	workspaceRequest := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/items/%d/governance", series.ID), nil)
	workspaceRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(workspaceRecorder, workspaceRequest)
	if workspaceRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for governance workspace, got %d body=%s", workspaceRecorder.Code, workspaceRecorder.Body.String())
	}
	var workspaceResponse struct {
		Data catalog.CatalogGovernanceWorkspace `json:"data"`
	}
	if err := json.Unmarshal(workspaceRecorder.Body.Bytes(), &workspaceResponse); err != nil {
		t.Fatalf("decode governance response: %v", err)
	}
	if workspaceResponse.Data.ItemID != series.ID || len(workspaceResponse.Data.SourceEvidence) != 1 || len(workspaceResponse.Data.SelectedImages) != 1 {
		t.Fatalf("unexpected governance workspace response: %#v", workspaceResponse.Data)
	}

	updateRecorder := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/items/%d/governance/fields", series.ID), strings.NewReader(`{"field_key":"title","value":"Manual Show","lock":true,"lock_reason":"editor"}`))
	updateRequest.Header.Set("Content-Type", "application/json")
	updateRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(updateRecorder, updateRequest)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for governance field update, got %d body=%s", updateRecorder.Code, updateRecorder.Body.String())
	}
}

func TestCatalogGovernanceAssetLinkCorrectionPreservesWorkspaceState(t *testing.T) {
	router, db, authSvc, _, _, catalogSvc, libraryID := newCatalogRouteHarness(t, nil)
	ctx := context.Background()
	authHeader := createAuthHeader(t, ctx, authSvc)

	series, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeSeries, Title: "Show A", Path: "/library/ShowA", SortKey: "Show A", AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernanceManual})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", Path: "/library/ShowA/Season1", SortKey: "Show A S01", IndexNumber: &seasonNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: catalog.AvailabilityMissing})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: series.ID, FieldKey: "title", Value: "Locked Show A", Lock: true, LockReason: "editor"}); err != nil {
		t.Fatalf("lock title: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ItemImage{ItemID: series.ID, ImageType: "poster", URL: "/series-poster.jpg", IsSelected: true}).Error; err != nil {
		t.Fatalf("create selected image: %v", err)
	}
	asset := database.MediaAsset{LibraryID: libraryID, AssetType: "main", DisplayName: "Season Asset", Status: "available", ProbeStatus: "ready"}
	if err := db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.AssetItem{AssetID: asset.ID, ItemID: series.ID, Role: "primary", SegmentIndex: 0, Source: "scanner"}).Error; err != nil {
		t.Fatalf("link asset to series: %v", err)
	}

	linkRecorder := httptest.NewRecorder()
	linkRequest := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/items/%d/governance/assets/%d/links", series.ID, asset.ID), strings.NewReader(fmt.Sprintf(`{"target_item_id":%d}`, season.ID)))
	linkRequest.Header.Set("Authorization", authHeader)
	linkRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(linkRecorder, linkRequest)
	if linkRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for governance asset relink, got %d body=%s", linkRecorder.Code, linkRecorder.Body.String())
	}
	var linkResponse struct {
		Data catalog.CatalogGovernanceWorkspace `json:"data"`
	}
	if err := json.Unmarshal(linkRecorder.Body.Bytes(), &linkResponse); err != nil {
		t.Fatalf("decode governance relink response: %v", err)
	}
	if len(linkResponse.Data.FieldStates) == 0 || len(linkResponse.Data.SelectedImages) != 1 {
		t.Fatalf("expected field locks and image selections to survive relink, got %#v", linkResponse.Data)
	}
	if len(linkResponse.Data.Assets) != 1 || len(linkResponse.Data.Assets[0].Links) != 2 {
		t.Fatalf("expected asset to link to both current and child item after relink, got %#v", linkResponse.Data.Assets)
	}

	unlinkRecorder := httptest.NewRecorder()
	unlinkRequest := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/items/%d/governance/assets/%d/links/%d", series.ID, asset.ID, series.ID), nil)
	unlinkRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(unlinkRecorder, unlinkRequest)
	if unlinkRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for governance unlink, got %d body=%s", unlinkRecorder.Code, unlinkRecorder.Body.String())
	}
	var unlinkResponse struct {
		Data catalog.CatalogGovernanceWorkspace `json:"data"`
	}
	if err := json.Unmarshal(unlinkRecorder.Body.Bytes(), &unlinkResponse); err != nil {
		t.Fatalf("decode governance unlink response: %v", err)
	}
	var remainingLinks []database.AssetItem
	if err := db.WithContext(ctx).Where("asset_id = ?", asset.ID).Order("item_id asc").Find(&remainingLinks).Error; err != nil {
		t.Fatalf("load remaining asset links: %v", err)
	}
	if len(remainingLinks) != 1 || remainingLinks[0].ItemID != season.ID {
		t.Fatalf("expected database to retain only child link after unlink, got %#v", remainingLinks)
	}
	if len(unlinkResponse.Data.FieldStates) == 0 || len(unlinkResponse.Data.SelectedImages) != 1 {
		t.Fatalf("expected unlink to preserve unrelated workspace state, got %#v", unlinkResponse.Data)
	}
}

func TestCatalogHierarchyRoutesExposeChildrenMissingAndNextUp(t *testing.T) {
	router, db, authSvc, _, _, catalogSvc, libraryID := newCatalogRouteHarness(t, nil)
	ctx := context.Background()
	authHeader := createAuthHeader(t, ctx, authSvc)

	series, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeSeries, Title: "Show A", Path: "/library/ShowA", SortKey: "Show A", AvailabilityStatus: catalog.AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", Path: "/library/ShowA/Season 1", SortKey: "Show A S01", IndexNumber: &seasonNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: catalog.AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	episodeOne := 1
	availableEpisode, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 1", Path: "/library/ShowA/Season 1/ShowA.S01E01.mkv", SortKey: "Show A S01E01", IndexNumber: &episodeOne, ParentIndexNumber: &seasonNumber, AvailabilityStatus: catalog.AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create available episode: %v", err)
	}
	episodeTwo := 2
	nextUpEpisode, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 2", Path: "/library/ShowA/Season 1/ShowA.S01E02.mkv", SortKey: "Show A S01E02", IndexNumber: &episodeTwo, ParentIndexNumber: &seasonNumber, AvailabilityStatus: catalog.AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create next-up episode: %v", err)
	}
	episodeThree := 3
	missingEpisode, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 3", Path: "/library/ShowA/Season 1/ShowA.S01E03.mkv", SortKey: "Show A S01E03", IndexNumber: &episodeThree, ParentIndexNumber: &seasonNumber, AvailabilityStatus: catalog.AvailabilityMissing})
	if err != nil {
		t.Fatalf("create missing episode: %v", err)
	}
	episodeFour := 4
	unairedEpisode, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 4", Path: "/library/ShowA/Season 1/ShowA.S01E04.mkv", SortKey: "Show A S01E04", IndexNumber: &episodeFour, ParentIndexNumber: &seasonNumber, AvailabilityStatus: catalog.AvailabilityUnaired})
	if err != nil {
		t.Fatalf("create unaired episode: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.UserItemData{UserID: 1, ItemID: availableEpisode.ID, PlayCount: 1, CompletedAt: timePtr(time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)), LastPlayedAt: timePtr(time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC))}).Error; err != nil {
		t.Fatalf("seed watched episode: %v", err)
	}

	childrenRecorder := httptest.NewRecorder()
	childrenRequest := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/items/%d/children?type=season", series.ID), nil)
	router.ServeHTTP(childrenRecorder, childrenRequest)
	if childrenRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for children route, got %d body=%s", childrenRecorder.Code, childrenRecorder.Body.String())
	}
	var childrenResponse struct {
		Data []catalog.CatalogListItem `json:"data"`
	}
	if err := json.Unmarshal(childrenRecorder.Body.Bytes(), &childrenResponse); err != nil {
		t.Fatalf("decode children response: %v", err)
	}
	if len(childrenResponse.Data) != 1 || childrenResponse.Data[0].ID != season.ID || childrenResponse.Data[0].Type != catalog.ItemTypeSeason {
		t.Fatalf("unexpected children response: %#v", childrenResponse.Data)
	}

	missingRecorder := httptest.NewRecorder()
	missingRequest := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/series/%d/missing", series.ID), nil)
	router.ServeHTTP(missingRecorder, missingRequest)
	if missingRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for missing route, got %d body=%s", missingRecorder.Code, missingRecorder.Body.String())
	}
	var missingResponse struct {
		Data []catalog.CatalogEpisodeDetail `json:"data"`
	}
	if err := json.Unmarshal(missingRecorder.Body.Bytes(), &missingResponse); err != nil {
		t.Fatalf("decode missing response: %v", err)
	}
	if len(missingResponse.Data) != 1 || missingResponse.Data[0].ID != missingEpisode.ID || missingResponse.Data[0].AvailabilityStatus != catalog.AvailabilityMissing {
		t.Fatalf("unexpected missing response: %#v", missingResponse.Data)
	}

	unairedRecorder := httptest.NewRecorder()
	unairedRequest := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/series/%d/episodes?availability=unaired", series.ID), nil)
	router.ServeHTTP(unairedRecorder, unairedRequest)
	if unairedRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for filtered episode route, got %d body=%s", unairedRecorder.Code, unairedRecorder.Body.String())
	}
	var unairedResponse struct {
		Data []catalog.CatalogEpisodeDetail `json:"data"`
	}
	if err := json.Unmarshal(unairedRecorder.Body.Bytes(), &unairedResponse); err != nil {
		t.Fatalf("decode unaired response: %v", err)
	}
	if len(unairedResponse.Data) != 1 || unairedResponse.Data[0].ID != unairedEpisode.ID || unairedResponse.Data[0].AvailabilityStatus != catalog.AvailabilityUnaired {
		t.Fatalf("unexpected unaired response: %#v", unairedResponse.Data)
	}

	nextUpRecorder := httptest.NewRecorder()
	nextUpRequest := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/series/%d/next-up", series.ID), nil)
	nextUpRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(nextUpRecorder, nextUpRequest)
	if nextUpRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for next-up route, got %d body=%s", nextUpRecorder.Code, nextUpRecorder.Body.String())
	}
	var nextUpResponse struct {
		Data *catalog.CatalogEpisodeDetail `json:"data"`
	}
	if err := json.Unmarshal(nextUpRecorder.Body.Bytes(), &nextUpResponse); err != nil {
		t.Fatalf("decode next-up response: %v", err)
	}
	if nextUpResponse.Data == nil || nextUpResponse.Data.ID != nextUpEpisode.ID {
		t.Fatalf("unexpected next-up response: %#v", nextUpResponse.Data)
	}
}

func TestCatalogMetadataRoutesSearchApplyAndRefetch(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 101, "title": "Movie A", "original_title": "Movie A", "release_date": "2024-02-02"}}})
		case "/movie/101":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 101, "title": "Movie A", "original_title": "Movie A", "overview": "Movie overview", "release_date": "2024-02-02", "runtime": 121, "genres": []map[string]any{}, "credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}}, "images": map[string]any{"logos": []map[string]any{}}, "videos": map[string]any{"results": []map[string]any{}}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer tmdb.Close()

	router, _, authSvc, _, settingsSvc, catalogSvc, libraryID := newCatalogRouteHarness(t, tmdb)
	ctx := context.Background()
	authHeader := createAuthHeader(t, ctx, authSvc)
	if _, err := settingsSvc.UpdateMetadataSettings(ctx, settings.UpdateMetadataSettingsInput{TMDB: settings.MetadataProviderInput{APIKey: "catalog-key", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: "1s"}}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeMovie, Title: "Movie A", Path: "/library/MovieA.2024.mkv", SortKey: "Movie A"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}

	searchRecorder := httptest.NewRecorder()
	searchRequest := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/items/%d/metadata/search", item.ID), strings.NewReader(`{"title":"Movie A"}`))
	searchRequest.Header.Set("Content-Type", "application/json")
	searchRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(searchRecorder, searchRequest)
	if searchRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for metadata search, got %d body=%s", searchRecorder.Code, searchRecorder.Body.String())
	}
	var searchResponse struct {
		Data []metadata.SearchCandidate `json:"data"`
	}
	if err := json.Unmarshal(searchRecorder.Body.Bytes(), &searchResponse); err != nil {
		t.Fatalf("decode search response: %v", err)
	}
	if len(searchResponse.Data) != 1 || searchResponse.Data[0].ExternalID != "movie:101" {
		t.Fatalf("unexpected search response: %#v", searchResponse.Data)
	}

	applyRecorder := httptest.NewRecorder()
	applyRequest := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/items/%d/metadata/apply", item.ID), strings.NewReader(`{"external_id":"movie:101"}`))
	applyRequest.Header.Set("Content-Type", "application/json")
	applyRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(applyRecorder, applyRequest)
	if applyRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for metadata apply, got %d body=%s", applyRecorder.Code, applyRecorder.Body.String())
	}

	refetchRecorder := httptest.NewRecorder()
	refetchRequest := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/items/%d/metadata/refetch", item.ID), nil)
	refetchRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(refetchRecorder, refetchRequest)
	if refetchRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for metadata refetch, got %d body=%s", refetchRecorder.Code, refetchRecorder.Body.String())
	}
}

func TestCatalogProgressRoutesUseItemAndAssetIdentity(t *testing.T) {
	router, db, authSvc, _, _, catalogSvc, libraryID := newCatalogRouteHarness(t, nil)
	ctx := context.Background()
	authHeader := createAuthHeader(t, ctx, authSvc)
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeEpisode, Title: "Episode 2", Path: "/library/ShowA.S01E02.mkv", SortKey: "Episode 2", AvailabilityStatus: catalog.AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	asset := database.MediaAsset{LibraryID: libraryID, AssetType: "main", Status: "available", ProbeStatus: "complete"}
	if err := db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.AssetItem{AssetID: asset.ID, ItemID: item.ID, Role: "primary", SegmentIndex: 0}).Error; err != nil {
		t.Fatalf("create asset item link: %v", err)
	}

	updateRecorder := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(http.MethodPost, "/api/v1/me/progress", strings.NewReader(fmt.Sprintf(`{"item_id":%d,"asset_id":%d,"position_seconds":900,"duration_seconds":1800}`, item.ID, asset.ID)))
	updateRequest.Header.Set("Content-Type", "application/json")
	updateRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(updateRecorder, updateRequest)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for catalog progress update, got %d body=%s", updateRecorder.Code, updateRecorder.Body.String())
	}

	stateRecorder := httptest.NewRecorder()
	stateRequest := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/items/%d/progress", item.ID), nil)
	stateRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(stateRecorder, stateRequest)
	if stateRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for catalog progress state, got %d body=%s", stateRecorder.Code, stateRecorder.Body.String())
	}
	var stateResponse struct {
		Data progress.State `json:"data"`
	}
	if err := json.Unmarshal(stateRecorder.Body.Bytes(), &stateResponse); err != nil {
		t.Fatalf("decode progress response: %v", err)
	}
	if stateResponse.Data.ItemID != item.ID || stateResponse.Data.AssetID == nil || *stateResponse.Data.AssetID != asset.ID {
		t.Fatalf("unexpected catalog progress response: %#v", stateResponse.Data)
	}
}

func TestCatalogPlaybackRoutesUseAssetAndInventoryFileIdentity(t *testing.T) {
	router, db, authSvc, _, _, catalogSvc, libraryID := newCatalogRouteHarness(t, nil)
	ctx := context.Background()
	authHeader := createAuthHeader(t, ctx, authSvc)
	runtimeSeconds := 1800
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeMovie, Title: "Movie A", Path: "/library/MovieA.mp4", SortKey: "Movie A", AvailabilityStatus: catalog.AvailabilityAvailable, RuntimeSeconds: &runtimeSeconds})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	asset := database.MediaAsset{LibraryID: libraryID, AssetType: "main", Status: "available", ProbeStatus: "ready", QualityLabel: "1080p"}
	if err := db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	filePath := filepath.Join(t.TempDir(), "catalog-playback.mp4")
	if err := os.WriteFile(filePath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write inventory file: %v", err)
	}
	file := database.InventoryFile{LibraryID: libraryID, StorageProvider: "local", StoragePath: filePath, Container: "mp4", Status: "available"}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.AssetItem{AssetID: asset.ID, ItemID: item.ID, Role: "primary", SegmentIndex: 0}).Error; err != nil {
		t.Fatalf("create asset item: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: "source", PartIndex: 0}).Error; err != nil {
		t.Fatalf("create asset file: %v", err)
	}
	width := 1280
	height := 720
	if err := db.WithContext(ctx).Create(&database.MediaStream{FileID: file.ID, StreamIndex: 0, StreamType: "video", Codec: "h264", Width: &width, Height: &height}).Error; err != nil {
		t.Fatalf("create media stream: %v", err)
	}

	playbackRecorder := httptest.NewRecorder()
	playbackRequest := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/items/%d/playback?client_profile=web", item.ID), nil)
	playbackRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(playbackRecorder, playbackRequest)
	if playbackRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for catalog playback, got %d body=%s", playbackRecorder.Code, playbackRecorder.Body.String())
	}
	var playbackResponse struct {
		Data playback.PlaybackSource `json:"data"`
	}
	if err := json.Unmarshal(playbackRecorder.Body.Bytes(), &playbackResponse); err != nil {
		t.Fatalf("decode playback response: %v", err)
	}
	if playbackResponse.Data.ItemID != item.ID || playbackResponse.Data.AssetID != asset.ID || playbackResponse.Data.FileID != file.ID {
		t.Fatalf("unexpected catalog playback response: %#v", playbackResponse.Data)
	}

	linkRecorder := httptest.NewRecorder()
	linkRequest := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/assets/%d/link", asset.ID), nil)
	router.ServeHTTP(linkRecorder, linkRequest)
	if linkRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for asset link, got %d body=%s", linkRecorder.Code, linkRecorder.Body.String())
	}
	var linkResponse struct {
		Data playback.FileLink `json:"data"`
	}
	if err := json.Unmarshal(linkRecorder.Body.Bytes(), &linkResponse); err != nil {
		t.Fatalf("decode asset link response: %v", err)
	}
	if linkResponse.Data.AssetID != asset.ID || linkResponse.Data.FileID != file.ID {
		t.Fatalf("unexpected asset link response: %#v", linkResponse.Data)
	}
}

func TestLegacyMediaItemRouteIsRemovedWhenCatalogReadEnabled(t *testing.T) {
	router, db, _, _, settingsSvc, _, libraryID := newCatalogRouteHarness(t, nil)
	ctx := context.Background()
	if _, err := settingsSvc.UpdateCatalogMigrationState(ctx, settings.UpdateCatalogMigrationStateInput{CatalogReadEnabled: true}); err != nil {
		t.Fatalf("enable catalog reads: %v", err)
	}
	item := database.MediaItem{LibraryID: libraryID, Type: "movie", Title: "Legacy Movie", SourcePath: "/library/Legacy.Movie.mkv", MatchStatus: metadata.StatusPending, Status: "ready"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create legacy media item: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-items/%d", item.ID), nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestLegacyMediaFileStreamRouteIsRemovedWhenCatalogReadEnabled(t *testing.T) {
	router, _, _, _, settingsSvc, _, _ := newCatalogRouteHarness(t, nil)
	ctx := context.Background()
	if _, err := settingsSvc.UpdateCatalogMigrationState(ctx, settings.UpdateCatalogMigrationStateInput{CatalogReadEnabled: true}); err != nil {
		t.Fatalf("enable catalog reads: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/media-files/99/stream", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestLegacyMediaWriteAndHierarchyRoutesAreRemovedWhenCatalogReadEnabled(t *testing.T) {
	router, db, _, _, settingsSvc, _, libraryID := newCatalogRouteHarness(t, nil)
	ctx := context.Background()
	if _, err := settingsSvc.UpdateCatalogMigrationState(ctx, settings.UpdateCatalogMigrationStateInput{CatalogReadEnabled: true}); err != nil {
		t.Fatalf("enable catalog reads: %v", err)
	}
	item := database.MediaItem{LibraryID: libraryID, Type: "episode", Title: "Legacy Episode", SeriesTitle: "Legacy Show", SourcePath: "/library/Legacy.Show.S01E01.mkv", MatchStatus: metadata.StatusPending, Status: "ready"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create legacy media item: %v", err)
	}

	seriesRecorder := httptest.NewRecorder()
	seriesRequest := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-items/%d/series-episodes", item.ID), nil)
	router.ServeHTTP(seriesRecorder, seriesRequest)
	if seriesRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for removed series-episodes route, got %d body=%s", seriesRecorder.Code, seriesRecorder.Body.String())
	}

	updateRecorder := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/media-items/%d/metadata", item.ID), strings.NewReader(`{"title":"Manual"}`))
	updateRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(updateRecorder, updateRequest)
	if updateRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for removed metadata write route, got %d body=%s", updateRecorder.Code, updateRecorder.Body.String())
	}
}

func TestCatalogMaintenanceRoutesRebuildAndConsistency(t *testing.T) {
	router, db, authSvc, _, settingsSvc, catalogSvc, libraryID := newCatalogRouteHarness(t, nil)
	ctx := context.Background()
	adminAuthHeader := createAdminAuthHeader(t, ctx, db, authSvc)
	if _, err := settingsSvc.UpdateCatalogMigrationState(ctx, settings.UpdateCatalogMigrationStateInput{CatalogReadEnabled: true}); err != nil {
		t.Fatalf("enable catalog reads: %v", err)
	}
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: libraryID, Type: catalog.ItemTypeMovie, Title: "Movie A", Path: "/library/MovieA.mp4", SortKey: "Movie A", AvailabilityStatus: catalog.AvailabilityMissing})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := db.WithContext(ctx).Where("item_id = ?", item.ID).Delete(&database.ItemRollup{}).Error; err != nil {
		t.Fatalf("delete rollup: %v", err)
	}
	if err := db.WithContext(ctx).Where("item_id = ?", item.ID).Delete(&database.CatalogSearchDocument{}).Error; err != nil {
		t.Fatalf("delete search doc: %v", err)
	}

	consistencyRecorder := httptest.NewRecorder()
	consistencyRequest := httptest.NewRequest(http.MethodGet, "/api/v1/catalog-migration/consistency", nil)
	consistencyRequest.Header.Set("Authorization", adminAuthHeader)
	router.ServeHTTP(consistencyRecorder, consistencyRequest)
	if consistencyRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for consistency report, got %d body=%s", consistencyRecorder.Code, consistencyRecorder.Body.String())
	}
	var consistencyResponse struct {
		Data catalog.ConsistencyReport `json:"data"`
	}
	if err := json.Unmarshal(consistencyRecorder.Body.Bytes(), &consistencyResponse); err != nil {
		t.Fatalf("decode consistency response: %v", err)
	}
	if consistencyResponse.Data.MissingRollupCount == 0 || consistencyResponse.Data.MissingSearchDocumentCount == 0 {
		t.Fatalf("expected consistency report to detect gaps, got %#v", consistencyResponse.Data)
	}

	rebuildRecorder := httptest.NewRecorder()
	rebuildRequest := httptest.NewRequest(http.MethodPost, "/api/v1/catalog-migration/rebuild-projections", nil)
	rebuildRequest.Header.Set("Authorization", adminAuthHeader)
	router.ServeHTTP(rebuildRecorder, rebuildRequest)
	if rebuildRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for rebuild, got %d body=%s", rebuildRecorder.Code, rebuildRecorder.Body.String())
	}
	var rebuildResponse struct {
		Data catalog.RebuildResult `json:"data"`
	}
	if err := json.Unmarshal(rebuildRecorder.Body.Bytes(), &rebuildResponse); err != nil {
		t.Fatalf("decode rebuild response: %v", err)
	}
	if rebuildResponse.Data.ProjectionsRebuilt == 0 {
		t.Fatalf("expected rebuild response to record projection rebuild, got %#v", rebuildResponse.Data)
	}
}

func newCatalogRouteHarness(t *testing.T, tmdb *httptest.Server) (http.Handler, *gorm.DB, *auth.Service, *library.Service, *settings.Service, *catalog.Service, uint) {
	t.Helper()

	rootPath := t.TempDir()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		Storage:  config.StorageConfig{Provider: "local"},
		Local:    config.LocalStorageConfig{RootPath: rootPath},
	}
	if tmdb != nil {
		cfg.Metadata.TMDB = config.TMDBConfig{BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	catalogSvc := catalog.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	searchSvc := search.NewService(db, librarySvc)
	metadataSvc := metadata.NewService(db, cfg.Metadata, settingsSvc, searchSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db, searchSvc), searchSvc, metadataSvc, settingsSvc, catalogSvc)

	ctx := context.Background()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Local", RootPath: rootPath})
	if err != nil {
		t.Fatalf("create media source: %v", err)
	}
	record, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Library", Type: "shows", MediaSourceID: source.ID, RootPath: rootPath})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}

	return router, db, authSvc, librarySvc, settingsSvc, catalogSvc, record.ID
}
