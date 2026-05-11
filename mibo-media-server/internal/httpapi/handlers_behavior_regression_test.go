package httpapi

import (
	"context"
	"encoding/json"
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
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/metadata"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/probe"
	"github.com/atlan/mibo-media-server/internal/progress"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
)

func TestQueueLibraryScanBehavior(t *testing.T) {
	handler, authHeader, db, _, _ := newBehaviorRegressionServer(t)
	libraryRecord := createBehaviorRegressionLibrary(t, db, "Scan Library")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/libraries/"+uintString(libraryRecord.ID)+"/scan", strings.NewReader(`{"mode":"full"}`))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]any
	decodeEnvelopeData(t, rec, &response)
	if queued, _ := response["queued"].(bool); !queued {
		t.Fatalf("expected queued=true, got %#v", response)
	}
	if mode, _ := response["mode"].(string); mode != "full" {
		t.Fatalf("expected full mode, got %#v", response)
	}

	var run database.WorkflowRun
	if err := db.WithContext(t.Context()).Where("library_id = ? AND reason = ?", libraryRecord.ID, library.WorkflowReasonManualScan).First(&run).Error; err != nil {
		t.Fatalf("load workflow run: %v", err)
	}
	if run.Status != workflow.RunStatusQueued {
		t.Fatalf("expected queued workflow run, got %#v", run)
	}

	var tasks []database.WorkflowTask
	if err := db.WithContext(t.Context()).Where("run_id = ?", run.ID).Find(&tasks).Error; err != nil {
		t.Fatalf("load workflow tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one scan task, got %d", len(tasks))
	}
	if tasks[0].TaskType != workflow.TaskTypeScanLibraryPath || tasks[0].Stage != workflow.StageScan {
		t.Fatalf("unexpected workflow task: %#v", tasks[0])
	}
}

func TestCatalogPlaybackBehavior(t *testing.T) {
	handler, authHeader, db, rootPath, catalogSvc := newBehaviorRegressionServer(t)
	libraryRecord, item, resource := seedBehaviorRegressionProjection(t, t.Context(), db, rootPath, catalogSvc)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/items/"+uintString(item.ID)+"/playback?library_id="+uintString(libraryRecord.ID)+"&resource_id="+uintString(resource.ID)+"&client_profile=web",
		nil,
	)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var source playback.PlaybackSource
	decodeEnvelopeData(t, rec, &source)
	if source.MetadataItemID != item.ID || source.ResourceID != resource.ID || !source.Playable {
		t.Fatalf("unexpected playback source: %#v", source)
	}
	if !strings.Contains(source.URL, "/api/v1/inventory-files/") || !strings.HasPrefix(source.URL, "http://") {
		t.Fatalf("expected absolute inventory playback url, got %q", source.URL)
	}
	if source.Decision.SelectedBy != "preferred_resource" {
		t.Fatalf("expected preferred_resource selection, got %#v", source.Decision)
	}
}

func TestGovernanceBehaviorUpdatesFieldAndVisibility(t *testing.T) {
	handler, authHeader, db, rootPath, catalogSvc := newBehaviorRegressionServer(t)
	libraryRecord, item, _ := seedBehaviorRegressionProjection(t, t.Context(), db, rootPath, catalogSvc)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/libraries/"+uintString(libraryRecord.ID)+"/items?type=movie", nil)
	listReq.Header.Set("Authorization", authHeader)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected initial list 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var beforeItems []catalog.CatalogListItem
	decodeEnvelopeData(t, listRec, &beforeItems)
	if len(beforeItems) != 1 || beforeItems[0].Title != "Regression Movie" {
		t.Fatalf("unexpected initial projection items: %#v", beforeItems)
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/items/"+uintString(item.ID)+"/governance/fields?library_id="+uintString(libraryRecord.ID), strings.NewReader(`{"field_key":"title","value":"Renamed Regression Movie"}`))
	updateReq.Header.Set("Authorization", authHeader)
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	handler.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected governance field update 200, got %d: %s", updateRec.Code, updateRec.Body.String())
	}
	var workspace catalog.CatalogGovernanceWorkspace
	decodeEnvelopeData(t, updateRec, &workspace)
	if workspace.Title != "Renamed Regression Movie" {
		t.Fatalf("expected updated workspace title, got %#v", workspace)
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/api/v1/items/"+uintString(item.ID)+"?library_id="+uintString(libraryRecord.ID), nil)
	detailReq.Header.Set("Authorization", authHeader)
	detailRec := httptest.NewRecorder()
	handler.ServeHTTP(detailRec, detailReq)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("expected detail 200, got %d: %s", detailRec.Code, detailRec.Body.String())
	}
	var detail catalog.CatalogItemDetail
	decodeEnvelopeData(t, detailRec, &detail)
	if detail.Title != "Renamed Regression Movie" {
		t.Fatalf("expected detail title to follow governance update, got %#v", detail)
	}

	hideReq := httptest.NewRequest(http.MethodPut, "/api/v1/items/"+uintString(item.ID)+"/governance/projection-visibility", strings.NewReader(`{"library_id":`+uintString(libraryRecord.ID)+`,"hidden":true}`))
	hideReq.Header.Set("Authorization", authHeader)
	hideReq.Header.Set("Content-Type", "application/json")
	hideRec := httptest.NewRecorder()
	handler.ServeHTTP(hideRec, hideReq)
	if hideRec.Code != http.StatusOK {
		t.Fatalf("expected visibility update 200, got %d: %s", hideRec.Code, hideRec.Body.String())
	}

	var projection database.LibraryMetadataProjection
	if err := db.WithContext(t.Context()).Where("library_id = ? AND metadata_item_id = ?", libraryRecord.ID, item.ID).First(&projection).Error; err != nil {
		t.Fatalf("load projection: %v", err)
	}
	if !projection.Hidden {
		t.Fatalf("expected projection hidden after governance visibility update: %#v", projection)
	}

	listAfterReq := httptest.NewRequest(http.MethodGet, "/api/v1/libraries/"+uintString(libraryRecord.ID)+"/items?type=movie", nil)
	listAfterReq.Header.Set("Authorization", authHeader)
	listAfterRec := httptest.NewRecorder()
	handler.ServeHTTP(listAfterRec, listAfterReq)
	if listAfterRec.Code != http.StatusOK {
		t.Fatalf("expected hidden list 200, got %d: %s", listAfterRec.Code, listAfterRec.Body.String())
	}
	var afterItems []catalog.CatalogListItem
	decodeEnvelopeData(t, listAfterRec, &afterItems)
	if len(afterItems) != 0 {
		t.Fatalf("expected hidden item excluded from browse results, got %#v", afterItems)
	}
}

func TestLibraryItemsBehaviorIncludesDiscoveredInventoryEntries(t *testing.T) {
	handler, authHeader, db, _, _ := newBehaviorRegressionServer(t)
	libraryRecord := createBehaviorRegressionLibrary(t, db, "Discover Library")
	file := database.InventoryFile{LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: filepath.Join(libraryRecord.RootPath, "Fresh.Movie.2026.mkv"), ContentClass: "video", Status: "available", ScanState: "discovered"}
	if err := db.WithContext(t.Context()).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/libraries/"+uintString(libraryRecord.ID)+"/items?type=movie", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var items []catalog.CatalogListItem
	decodeEnvelopeData(t, rec, &items)
	if len(items) != 1 {
		t.Fatalf("expected one discovered entry, got %#v", items)
	}
	if items[0].SourceKind != "inventory_file" || items[0].InventoryFileID == nil || *items[0].InventoryFileID != file.ID {
		t.Fatalf("expected inventory-backed browse item, got %#v", items[0])
	}
	if !items[0].Organizing || items[0].MetadataItemID != 0 {
		t.Fatalf("expected discovered item to stay organizing before metadata link, got %#v", items[0])
	}
}

func TestDiscoveryBehaviorReturnsEmptyItemsArray(t *testing.T) {
	handler, authHeader, db, _, _ := newBehaviorRegressionServer(t)
	libraryRecord := createBehaviorRegressionLibrary(t, db, "Empty Discovery Library")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/discovery?library_id="+uintString(libraryRecord.ID)+"&type=show", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Data catalog.BrowseItemsResult `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode discovery response: %v", err)
	}
	if response.Data.Items == nil {
		t.Fatalf("expected empty items array, got nil: %#v", response.Data)
	}
	if len(response.Data.Items) != 0 {
		t.Fatalf("expected zero discovery items, got %#v", response.Data.Items)
	}
	if response.Data.Sort == "" || response.Data.SortDirection == "" {
		t.Fatalf("expected normalized paging metadata, got %#v", response.Data)
	}
	_ = db
}

func TestLibraryItemsBehaviorReplacesDiscoveredEntryAfterMetadataCollapse(t *testing.T) {
	handler, authHeader, db, _, catalogSvc := newBehaviorRegressionServer(t)
	libraryRecord := createBehaviorRegressionLibrary(t, db, "Upgrade Library")
	file := database.InventoryFile{LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: filepath.Join(libraryRecord.RootPath, "Fresh.Movie.2026.mkv"), ContentClass: "video", Status: "available", ScanState: "discovered"}
	if err := db.WithContext(t.Context()).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	initialReq := httptest.NewRequest(http.MethodGet, "/api/v1/libraries/"+uintString(libraryRecord.ID)+"/items?type=movie", nil)
	initialReq.Header.Set("Authorization", authHeader)
	initialRec := httptest.NewRecorder()
	handler.ServeHTTP(initialRec, initialReq)
	if initialRec.Code != http.StatusOK {
		t.Fatalf("expected initial 200, got %d: %s", initialRec.Code, initialRec.Body.String())
	}
	var initialItems []catalog.CatalogListItem
	decodeEnvelopeData(t, initialRec, &initialItems)
	if len(initialItems) != 1 || initialItems[0].SourceKind != "inventory_file" {
		t.Fatalf("expected initial discovered entry, got %#v", initialItems)
	}

	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Fresh Movie", SortTitle: "Fresh Movie", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(t.Context()).Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	resource := database.Resource{StableResourceKey: "resource:fresh-movie", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, DisplayName: "Fresh Movie", Status: "available", ProbeStatus: "ready"}
	if err := db.WithContext(t.Context()).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	now := time.Now().UTC()
	if err := db.WithContext(t.Context()).Create(&database.ResourceFile{ResourceID: resource.ID, InventoryFileID: file.ID, Role: database.ResourceFileRoleSource, PartIndex: 0}).Error; err != nil {
		t.Fatalf("create resource file: %v", err)
	}
	if err := db.WithContext(t.Context()).Create(&database.ResourceLibraryLink{ResourceID: resource.ID, LibraryID: libraryRecord.ID, Status: "available", FirstSeenAt: now, LastSeenAt: now, ReviewState: database.ReviewStateAccepted}).Error; err != nil {
		t.Fatalf("create resource library link: %v", err)
	}
	if err := db.WithContext(t.Context()).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary, ReviewState: database.ReviewStateAccepted}).Error; err != nil {
		t.Fatalf("create resource metadata link: %v", err)
	}
	if _, err := catalogSvc.RebuildLibraryMetadataProjection(t.Context(), libraryRecord.ID, item.ID); err != nil {
		t.Fatalf("rebuild projection: %v", err)
	}

	upgradedReq := httptest.NewRequest(http.MethodGet, "/api/v1/libraries/"+uintString(libraryRecord.ID)+"/items?type=movie", nil)
	upgradedReq.Header.Set("Authorization", authHeader)
	upgradedRec := httptest.NewRecorder()
	handler.ServeHTTP(upgradedRec, upgradedReq)
	if upgradedRec.Code != http.StatusOK {
		t.Fatalf("expected upgraded 200, got %d: %s", upgradedRec.Code, upgradedRec.Body.String())
	}
	var upgradedItems []catalog.CatalogListItem
	decodeEnvelopeData(t, upgradedRec, &upgradedItems)
	if len(upgradedItems) != 1 {
		t.Fatalf("expected one upgraded catalog item, got %#v", upgradedItems)
	}
	if upgradedItems[0].SourceKind == "inventory_file" || upgradedItems[0].MetadataItemID != item.ID {
		t.Fatalf("expected discovered entry to be replaced by catalog item, got %#v", upgradedItems[0])
	}
}

func TestLibraryItemsBehaviorPrefersExistingMetadataCardForSiblingVersion(t *testing.T) {
	handler, authHeader, db, _, catalogSvc := newBehaviorRegressionServer(t)
	libraryRecord := createBehaviorRegressionLibrary(t, db, "Sibling Version Library")
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Sibling Movie", SortTitle: "Sibling Movie", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(t.Context()).Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	now := time.Now().UTC()
	primaryFile := database.InventoryFile{LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: filepath.Join(libraryRecord.RootPath, "Sibling.Movie.1080p.mkv"), ContentClass: "video", Status: "available", ScanState: "classified"}
	versionFile := database.InventoryFile{LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: filepath.Join(libraryRecord.RootPath, "Sibling.Movie.2160p.mkv"), ContentClass: "video", Status: "available", ScanState: "discovered"}
	if err := db.WithContext(t.Context()).Create(&primaryFile).Error; err != nil {
		t.Fatalf("create primary file: %v", err)
	}
	if err := db.WithContext(t.Context()).Create(&versionFile).Error; err != nil {
		t.Fatalf("create version file: %v", err)
	}
	primaryResource := database.Resource{StableResourceKey: "resource:sibling-primary", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, DisplayName: "Sibling Movie 1080p", Status: "available", ProbeStatus: "ready"}
	versionResource := database.Resource{StableResourceKey: "resource:sibling-version", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, DisplayName: "Sibling Movie 2160p", Status: "available", ProbeStatus: "ready", QualityLabel: "2160p"}
	if err := db.WithContext(t.Context()).Create(&primaryResource).Error; err != nil {
		t.Fatalf("create primary resource: %v", err)
	}
	if err := db.WithContext(t.Context()).Create(&versionResource).Error; err != nil {
		t.Fatalf("create version resource: %v", err)
	}
	for _, link := range []database.ResourceFile{{ResourceID: primaryResource.ID, InventoryFileID: primaryFile.ID, Role: database.ResourceFileRoleSource}, {ResourceID: versionResource.ID, InventoryFileID: versionFile.ID, Role: database.ResourceFileRoleSource}} {
		if err := db.WithContext(t.Context()).Create(&link).Error; err != nil {
			t.Fatalf("create resource file: %v", err)
		}
	}
	for _, link := range []database.ResourceLibraryLink{{ResourceID: primaryResource.ID, LibraryID: libraryRecord.ID, Status: "available", FirstSeenAt: now, LastSeenAt: now, ReviewState: database.ReviewStateAccepted}, {ResourceID: versionResource.ID, LibraryID: libraryRecord.ID, Status: "available", FirstSeenAt: now, LastSeenAt: now, ReviewState: database.ReviewStateAccepted}} {
		if err := db.WithContext(t.Context()).Create(&link).Error; err != nil {
			t.Fatalf("create library link: %v", err)
		}
	}
	if err := db.WithContext(t.Context()).Create(&database.ResourceMetadataLink{ResourceID: primaryResource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary, ReviewState: database.ReviewStateAccepted}).Error; err != nil {
		t.Fatalf("create primary metadata link: %v", err)
	}
	if err := db.WithContext(t.Context()).Create(&database.ResourceMetadataLink{ResourceID: versionResource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRoleVersion, ReviewState: database.ReviewStateAccepted}).Error; err != nil {
		t.Fatalf("create version metadata link: %v", err)
	}
	if _, err := catalogSvc.RebuildLibraryMetadataProjection(t.Context(), libraryRecord.ID, item.ID); err != nil {
		t.Fatalf("rebuild projection: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/libraries/"+uintString(libraryRecord.ID)+"/items?type=movie", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var items []catalog.CatalogListItem
	decodeEnvelopeData(t, rec, &items)
	if len(items) != 1 {
		t.Fatalf("expected one metadata card, got %#v", items)
	}
	if items[0].SourceKind == "inventory_file" || items[0].MetadataItemID != item.ID || items[0].ResourceCount != 2 {
		t.Fatalf("expected sibling version to collapse into existing metadata card, got %#v", items[0])
	}
}

func newBehaviorRegressionServer(t *testing.T) (http.Handler, string, *gorm.DB, string, *catalog.Service) {
	t.Helper()
	rootPath := t.TempDir()
	cfg := config.Config{
		HTTP:     config.HTTPConfig{Addr: ":8080"},
		Local:    config.LocalStorageConfig{RootPath: rootPath},
		Database: config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")},
		Worker:   config.WorkerConfig{Enabled: true},
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	workflowSvc := workflow.NewService(db)
	ingestSvc := ingest.NewService(db, workflowSvc)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, nil, ingestSvc, workflowSvc)
	searchSvc := search.NewService(db, librarySvc)
	progressSvc := progress.NewService(db, searchSvc)
	catalogSvc := catalog.NewService(db, ingestSvc)
	playbackSvc := playback.NewService(db, registry)
	metadataSvc := metadata.NewService(db, cfg.Metadata, settingsSvc, searchSvc, ingestSvc)
	handler := New(Dependencies{
		Config:   cfg,
		DB:       db,
		Registry: registry,
		Auth:     authSvc,
		Catalog:  catalogSvc,
		Library:  librarySvc,
		Ingest:   ingestSvc,
		Playback: playbackSvc,
		Progress: progressSvc,
		Search:   searchSvc,
		Metadata: metadataSvc,
		Settings: settingsSvc,
		Workflow: workflowSvc,
	})
	authHeader := loginTestUser(t, authSvc, "behavior-user", "password123")
	return handler, authHeader, db, rootPath, catalogSvc
}

func createBehaviorRegressionLibrary(t *testing.T, db *gorm.DB, name string) database.Library {
	t.Helper()
	ctx := context.Background()
	rootPath := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(rootPath, 0o755); err != nil {
		t.Fatalf("create root path: %v", err)
	}
	source := database.MediaSource{Name: name + " Source", Provider: "local", StorageRef: rootPath, RootPath: rootPath}
	if err := db.WithContext(ctx).Create(&source).Error; err != nil {
		t.Fatalf("create media source: %v", err)
	}
	libraryRecord := database.Library{Name: name, Type: "movies", MediaSourceID: source.ID, RootPath: rootPath, Status: "active", ScannerEnabled: true}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	pathRecord := database.LibraryPath{LibraryID: libraryRecord.ID, MediaSourceID: source.ID, RootPath: rootPath, DisplayName: name, Enabled: true}
	if err := db.WithContext(ctx).Create(&pathRecord).Error; err != nil {
		t.Fatalf("create library path: %v", err)
	}
	return libraryRecord
}

func seedBehaviorRegressionProjection(t *testing.T, ctx context.Context, db *gorm.DB, rootPath string, catalogSvc *catalog.Service) (database.Library, database.MetadataItem, database.Resource) {
	t.Helper()
	source := database.MediaSource{Name: "Regression Source", Provider: "local", StorageRef: rootPath, RootPath: rootPath}
	if err := db.WithContext(ctx).Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	libraryRecord := database.Library{Name: "Regression Library", Type: "movies", MediaSourceID: source.ID, RootPath: rootPath, Status: "active", ScannerEnabled: true}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	pathRecord := database.LibraryPath{LibraryID: libraryRecord.ID, MediaSourceID: source.ID, RootPath: rootPath, DisplayName: libraryRecord.Name, Enabled: true}
	if err := db.WithContext(ctx).Create(&pathRecord).Error; err != nil {
		t.Fatalf("create library path: %v", err)
	}

	runtimeSeconds := 7200
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Regression Movie", SortTitle: "Regression Movie", RuntimeSeconds: &runtimeSeconds, GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	resource := database.Resource{StableResourceKey: "resource:regression", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, DisplayName: "Regression Movie 4K", Status: "available", ProbeStatus: probe.StatusReady, QualityLabel: "2160p"}
	if err := db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	filePath := filepath.Join(rootPath, "regression-movie.mp4")
	if err := os.WriteFile(filePath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
	file := database.InventoryFile{LibraryID: libraryRecord.ID, StorageProvider: "local", StoragePath: filePath, Container: "mp4", Status: "available", SizeBytes: 5}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceFile{ResourceID: resource.ID, InventoryFileID: file.ID, Role: database.ResourceFileRoleSource}).Error; err != nil {
		t.Fatalf("create resource file: %v", err)
	}
	now := time.Now().UTC()
	if err := db.WithContext(ctx).Create(&database.ResourceLibraryLink{ResourceID: resource.ID, LibraryID: libraryRecord.ID, Status: "available", FirstSeenAt: now, LastSeenAt: now}).Error; err != nil {
		t.Fatalf("create library link: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary}).Error; err != nil {
		t.Fatalf("create metadata link: %v", err)
	}
	width := 3840
	height := 2160
	if err := db.WithContext(ctx).Create(&database.MediaStream{FileID: file.ID, StreamIndex: 0, StreamType: "video", Codec: "h264", Width: &width, Height: &height}).Error; err != nil {
		t.Fatalf("create media stream: %v", err)
	}
	if _, err := catalogSvc.RebuildLibraryMetadataProjection(ctx, libraryRecord.ID, item.ID); err != nil {
		t.Fatalf("rebuild library projection: %v", err)
	}
	if _, err := catalogSvc.RebuildMetadataSearchDocument(ctx, item.ID); err != nil {
		t.Fatalf("rebuild metadata search: %v", err)
	}
	if _, err := catalogSvc.RebuildLibrarySearchDocument(ctx, libraryRecord.ID, item.ID); err != nil {
		t.Fatalf("rebuild library search: %v", err)
	}
	return libraryRecord, item, resource
}

func decodeBehaviorEnvelope[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	return value
}
