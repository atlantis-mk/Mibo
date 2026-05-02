package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atlan/mibo-media-server/internal/auth"
	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/progress"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
	"gorm.io/gorm"
)

func TestScanExclusionRuleCRUDThroughHTTP(t *testing.T) {
	handler, authSvc := newAuthSessionsTestServer(t)
	authHeader := createAuthHeader(t, t.Context(), authSvc)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/scan-exclusion-rules", strings.NewReader(`{"name":"Skip promo","rule_type":"filename_token","value":"promo","reason":"advertisement","enabled":true}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", authHeader)
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	created := decodeScanExclusionRule(t, createRec)
	if created.ID == 0 || created.System || created.Value != "promo" || !created.Enabled {
		t.Fatalf("unexpected created rule: %#v", created)
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/scan-exclusion-rules/"+uintString(created.ID), strings.NewReader(`{"enabled":false}`))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("Authorization", authHeader)
	patchRec := httptest.NewRecorder()
	handler.ServeHTTP(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("expected patch 200, got %d: %s", patchRec.Code, patchRec.Body.String())
	}
	updated := decodeScanExclusionRule(t, patchRec)
	if updated.Enabled || updated.DisabledAt == nil {
		t.Fatalf("expected disabled rule with timestamp, got %#v", updated)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/scan-exclusion-rules/"+uintString(created.ID), nil)
	deleteReq.Header.Set("Authorization", authHeader)
	deleteRec := httptest.NewRecorder()
	handler.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected delete 200, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestScanExclusionRuleHTTPValidationAndSystemDeleteProtection(t *testing.T) {
	handler, authSvc := newAuthSessionsTestServer(t)
	authHeader := createAuthHeader(t, t.Context(), authSvc)

	invalidReq := httptest.NewRequest(http.MethodPost, "/api/v1/scan-exclusion-rules", strings.NewReader(`{"name":"Bad","rule_type":"substring","value":"ad","reason":"advertisement"}`))
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidReq.Header.Set("Authorization", authHeader)
	invalidRec := httptest.NewRecorder()
	handler.ServeHTTP(invalidRec, invalidReq)
	if invalidRec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid create 400, got %d: %s", invalidRec.Code, invalidRec.Body.String())
	}
	invalidScopeReq := httptest.NewRequest(http.MethodPost, "/api/v1/scan-exclusion-rules", strings.NewReader(`{"library_id":9999,"name":"Bad scope","rule_type":"filename_token","value":"promo","reason":"advertisement"}`))
	invalidScopeReq.Header.Set("Content-Type", "application/json")
	invalidScopeReq.Header.Set("Authorization", authHeader)
	invalidScopeRec := httptest.NewRecorder()
	handler.ServeHTTP(invalidScopeRec, invalidScopeReq)
	if invalidScopeRec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid scope create 400, got %d: %s", invalidScopeRec.Code, invalidScopeRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/scan-exclusion-rules", nil)
	listReq.Header.Set("Authorization", authHeader)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	rules := decodeScanExclusionRules(t, listRec)
	var systemRule database.ScanExclusionRule
	for _, rule := range rules {
		if rule.System {
			systemRule = rule
			break
		}
	}
	if systemRule.ID == 0 {
		t.Fatalf("expected seeded system rule in %#v", rules)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/scan-exclusion-rules/"+uintString(systemRule.ID), nil)
	deleteReq.Header.Set("Authorization", authHeader)
	deleteRec := httptest.NewRecorder()
	handler.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusBadRequest {
		t.Fatalf("expected system delete 400, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestFilenameExclusionHTTPFlowAndAuthorization(t *testing.T) {
	handler, authSvc, db := newFilenameExclusionTestServer(t)
	ctx := t.Context()
	authHeader := createAuthHeader(t, ctx, authSvc)
	itemID, otherFileID := seedFilenameExclusionHTTPFixture(t, ctx, db)

	unauthorizedReq := httptest.NewRequest(http.MethodGet, "/api/v1/items/"+uintString(itemID)+"/scan-exclusion-preview", nil)
	unauthorizedRec := httptest.NewRecorder()
	handler.ServeHTTP(unauthorizedRec, unauthorizedReq)
	if unauthorizedRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated preview 401, got %d: %s", unauthorizedRec.Code, unauthorizedRec.Body.String())
	}

	previewReq := httptest.NewRequest(http.MethodGet, "/api/v1/items/"+uintString(itemID)+"/scan-exclusion-preview", nil)
	previewReq.Header.Set("Authorization", authHeader)
	previewRec := httptest.NewRecorder()
	handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("expected preview 200, got %d: %s", previewRec.Code, previewRec.Body.String())
	}
	preview := decodeFilenameExclusionPreview(t, previewRec)
	if preview.NormalizedFilename != "sharedname.mp4" || preview.AffectedCount != 2 {
		t.Fatalf("unexpected preview: %#v", preview)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/items/"+uintString(itemID)+"/filename-exclusion-rule", strings.NewReader(`{"reason":"wrong_import"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", authHeader)
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("expected filename rule create 200, got %d: %s", createRec.Code, createRec.Body.String())
	}
	created := decodeFilenameExclusionRule(t, createRec)
	if created.ID == 0 || !created.Enabled || created.AffectedCount != 2 {
		t.Fatalf("unexpected created filename rule: %#v", created)
	}

	restoreReq := httptest.NewRequest(http.MethodPost, "/api/v1/filename-exclusion-rules/"+uintString(created.ID)+"/restores", strings.NewReader(`{"inventory_file_id":`+uintString(otherFileID)+`}`))
	restoreReq.Header.Set("Content-Type", "application/json")
	restoreReq.Header.Set("Authorization", authHeader)
	restoreRec := httptest.NewRecorder()
	handler.ServeHTTP(restoreRec, restoreReq)
	if restoreRec.Code != http.StatusOK {
		t.Fatalf("expected restore 200, got %d: %s", restoreRec.Code, restoreRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/filename-exclusion-rules?enabled=true", nil)
	listReq.Header.Set("Authorization", authHeader)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	rules := decodeFilenameExclusionRules(t, listRec)
	if len(rules) != 1 || !hasRestoredAffectedFile(rules[0], otherFileID) {
		t.Fatalf("expected restored affected file in list response, got %#v", rules)
	}

	disableReq := httptest.NewRequest(http.MethodPatch, "/api/v1/filename-exclusion-rules/"+uintString(created.ID), strings.NewReader(`{"enabled":false}`))
	disableReq.Header.Set("Content-Type", "application/json")
	disableReq.Header.Set("Authorization", authHeader)
	disableRec := httptest.NewRecorder()
	handler.ServeHTTP(disableRec, disableReq)
	if disableRec.Code != http.StatusOK {
		t.Fatalf("expected rule restore 200, got %d: %s", disableRec.Code, disableRec.Body.String())
	}
	disabled := decodeFilenameExclusionRule(t, disableRec)
	if disabled.Enabled || disabled.DisabledAt == nil {
		t.Fatalf("expected disabled filename rule with timestamp, got %#v", disabled)
	}
}

func newFilenameExclusionTestServer(t *testing.T) (http.Handler, *auth.Service, *gorm.DB) {
	t.Helper()
	cfg := config.Config{HTTP: config.HTTPConfig{Addr: ":8080"}, Storage: config.StorageConfig{Provider: "local"}, Local: config.LocalStorageConfig{RootPath: t.TempDir()}, Database: config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")}, Worker: config.WorkerConfig{Enabled: true}}
	db, err := database.Open(cfg.Database)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	searchSvc := search.NewService(db, librarySvc)
	progressSvc := progress.NewService(db, searchSvc)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	catalogSvc := catalog.NewService(db)
	playbackSvc := playback.NewService(db, registry)
	return New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playbackSvc, progressSvc, searchSvc, nil, settingsSvc, catalogSvc), authSvc, db
}

func seedFilenameExclusionHTTPFixture(t *testing.T, ctx context.Context, db *gorm.DB) (uint, uint) {
	t.Helper()
	source := database.MediaSource{Name: "Fixture Source", Provider: "local", StorageRef: "local", RootPath: "/fixture"}
	if err := db.WithContext(ctx).Create(&source).Error; err != nil {
		t.Fatalf("create media source: %v", err)
	}
	libraryRecord := database.Library{Name: "Fixture", Type: "movies", MediaSourceID: source.ID, RootPath: "/fixture", Status: "active", ScannerEnabled: true}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	itemID, _ := createFilenameExclusionHTTPLinkedFile(t, ctx, db, libraryRecord.ID, "/fixture/Movie A/SharedName.mp4")
	_, otherFileID := createFilenameExclusionHTTPLinkedFile(t, ctx, db, libraryRecord.ID, "/fixture/Movie B/SharedName.mp4")
	return itemID, otherFileID
}

func createFilenameExclusionHTTPLinkedFile(t *testing.T, ctx context.Context, db *gorm.DB, libraryID uint, storagePath string) (uint, uint) {
	t.Helper()
	item := database.CatalogItem{LibraryID: libraryID, Type: catalog.ItemTypeMovie, Title: filepath.Base(storagePath), Path: storagePath, SortKey: storagePath, AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernancePending}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	file := database.InventoryFile{LibraryID: libraryID, StorageProvider: "local", StoragePath: storagePath, SizeBytes: 1024, ContentClass: "video", Status: inventory.FileStatusAvailable}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	asset := database.MediaAsset{LibraryID: libraryID, AssetType: inventory.AssetTypeMain, DisplayName: filepath.Base(storagePath), Status: inventory.AssetStatusAvailable, ProbeStatus: "ready"}
	if err := db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create media asset: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: inventory.FileRoleSource, PartIndex: 0}).Error; err != nil {
		t.Fatalf("create asset file: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.AssetItem{AssetID: asset.ID, ItemID: item.ID, Role: inventory.AssetItemRolePrimary, Source: "scanner"}).Error; err != nil {
		t.Fatalf("create asset item: %v", err)
	}
	return item.ID, file.ID
}

func decodeScanExclusionRule(t *testing.T, rec *httptest.ResponseRecorder) database.ScanExclusionRule {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	var rule database.ScanExclusionRule
	if err := json.Unmarshal(data, &rule); err != nil {
		t.Fatalf("decode rule: %v", err)
	}
	return rule
}

func decodeScanExclusionRules(t *testing.T, rec *httptest.ResponseRecorder) []database.ScanExclusionRule {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	var rules []database.ScanExclusionRule
	if err := json.Unmarshal(data, &rules); err != nil {
		t.Fatalf("decode rules: %v", err)
	}
	return rules
}

func decodeFilenameExclusionPreview(t *testing.T, rec *httptest.ResponseRecorder) library.FilenameExclusionPreview {
	t.Helper()
	var preview library.FilenameExclusionPreview
	decodeScanExclusionEnvelopeData(t, rec, &preview)
	return preview
}

func decodeFilenameExclusionRule(t *testing.T, rec *httptest.ResponseRecorder) library.FilenameExclusionRuleView {
	t.Helper()
	var rule library.FilenameExclusionRuleView
	decodeScanExclusionEnvelopeData(t, rec, &rule)
	return rule
}

func decodeFilenameExclusionRules(t *testing.T, rec *httptest.ResponseRecorder) []library.FilenameExclusionRuleView {
	t.Helper()
	var rules []library.FilenameExclusionRuleView
	decodeScanExclusionEnvelopeData(t, rec, &rules)
	return rules
}

func decodeScanExclusionEnvelopeData(t *testing.T, rec *httptest.ResponseRecorder, target any) {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("decode data: %v", err)
	}
}

func hasRestoredAffectedFile(rule library.FilenameExclusionRuleView, fileID uint) bool {
	for _, file := range rule.AffectedFiles {
		if file.ID == fileID && file.Restored {
			return true
		}
	}
	return false
}
