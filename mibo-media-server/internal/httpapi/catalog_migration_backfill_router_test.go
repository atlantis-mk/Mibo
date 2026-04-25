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

func TestCatalogMigrationBackfillRequiresAuth(t *testing.T) {
	t.Parallel()

	router, _, _, _, _, _ := newCatalogMigrationBackfillTestRouter(t)
	tests := []struct {
		name   string
		method string
		url    string
		body   string
	}{
		{name: "queue", method: http.MethodPost, url: "/api/v1/catalog-migration/backfill", body: `{}`},
		{name: "list runs", method: http.MethodGet, url: "/api/v1/catalog-migration/runs"},
		{name: "get run", method: http.MethodGet, url: "/api/v1/catalog-migration/runs/1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
			if tc.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestQueueCatalogLegacyBackfill(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	router, db, authSvc, jobsSvc, catalogSvc, libraryID := newCatalogMigrationBackfillTestRouter(t)
	authHeader, user := createBackfillAuthHeader(t, ctx, authSvc)

	queueAll := func() catalog.LegacyBackfillRun {
		return queueCatalogLegacyBackfillRequest(t, router, authHeader, `{}`)
	}

	allRun := queueAll()
	if allRun.Status != catalog.LegacyBackfillStatusQueued {
		t.Fatalf("expected queued all-libraries run, got %#v", allRun)
	}
	if allRun.Scope.Kind != catalog.LegacyBackfillScopeAll || allRun.Scope.LibraryID != nil {
		t.Fatalf("expected all-libraries scope, got %#v", allRun.Scope)
	}
	if allRun.TriggeredByUserID != user.ID {
		t.Fatalf("expected triggered_by_user_id=%d, got %d", user.ID, allRun.TriggeredByUserID)
	}
	assertCatalogBackfillQueuedJob(t, ctx, db, jobsSvc, "catalog-backfill-legacy:all", allRun.ID, nil)

	reusedRun := queueAll()
	if reusedRun.ID != allRun.ID {
		t.Fatalf("expected duplicate all-libraries queue to reuse run %d, got %d", allRun.ID, reusedRun.ID)
	}
	assertCatalogBackfillJobCount(t, ctx, db, 1, "catalog-backfill-legacy:all")

	libraryRun := queueCatalogLegacyBackfillRequest(t, router, authHeader, fmt.Sprintf(`{"library_id":%d}`, libraryID))
	if libraryRun.Status != catalog.LegacyBackfillStatusQueued {
		t.Fatalf("expected queued library run, got %#v", libraryRun)
	}
	if libraryRun.Scope.Kind != catalog.LegacyBackfillScopeLibrary || libraryRun.Scope.LibraryID == nil || *libraryRun.Scope.LibraryID != libraryID {
		t.Fatalf("expected library scope %d, got %#v", libraryID, libraryRun.Scope)
	}
	assertCatalogBackfillQueuedJob(t, ctx, db, jobsSvc, fmt.Sprintf("catalog-backfill-legacy:library:%d", libraryID), libraryRun.ID, &libraryID)

	storedRun, err := catalogSvc.GetLegacyBackfillRun(ctx, libraryRun.ID)
	if err != nil {
		t.Fatalf("load queued library run: %v", err)
	}
	if storedRun.Status != catalog.LegacyBackfillStatusQueued {
		t.Fatalf("expected stored run queued, got %#v", storedRun)
	}
}

func TestCatalogMigrationBackfillRunReads(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	router, db, authSvc, _, catalogSvc, libraryID := newCatalogMigrationBackfillTestRouter(t)
	authHeader, user := createBackfillAuthHeader(t, ctx, authSvc)

	olderRun, err := catalogSvc.CreateLegacyBackfillRun(ctx, catalog.CreateLegacyBackfillRunInput{
		Scope:             catalog.LegacyBackfillScope{Kind: catalog.LegacyBackfillScopeLibrary, LibraryID: &libraryID},
		TriggeredByUserID: user.ID,
	})
	if err != nil {
		t.Fatalf("create older run: %v", err)
	}
	newerRun, err := catalogSvc.CreateLegacyBackfillRun(ctx, catalog.CreateLegacyBackfillRunInput{
		Scope:             catalog.LegacyBackfillScope{Kind: catalog.LegacyBackfillScopeAll},
		TriggeredByUserID: user.ID,
	})
	if err != nil {
		t.Fatalf("create newer run: %v", err)
	}

	finishedAt := time.Now().UTC()
	if err := db.WithContext(ctx).Model(&database.CatalogMigrationRun{}).Where("id = ?", olderRun.ID).Updates(map[string]any{
		"status":        catalog.LegacyBackfillStatusCompleted,
		"success_count": 1,
		"finished_at":   finishedAt,
	}).Error; err != nil {
		t.Fatalf("update older run: %v", err)
	}
	if err := db.WithContext(ctx).Model(&database.CatalogMigrationRun{}).Where("id = ?", newerRun.ID).Updates(map[string]any{
		"status":                            catalog.LegacyBackfillStatusCompleted,
		"success_count":                     2,
		"conflict_count":                    1,
		"duplicate_episode_candidate_count": 1,
		"finished_at":                       finishedAt,
	}).Error; err != nil {
		t.Fatalf("update newer run: %v", err)
	}

	legacyItemTen := uint(10)
	legacyItemTwenty := uint(20)
	legacyFileFive := uint(5)
	otherLibraryID := uint(libraryID + 1)
	entries := []database.CatalogMigrationEntry{
		{RunID: newerRun.ID, EntryType: catalog.LegacyBackfillEntryTypeSuccess, LibraryID: &libraryID, LegacyMediaItemID: &legacyItemTwenty, Message: "success-late"},
		{RunID: newerRun.ID, EntryType: catalog.LegacyBackfillEntryTypeConflict, LibraryID: &libraryID, LegacyMediaItemID: &legacyItemTen, Message: "conflict"},
		{RunID: newerRun.ID, EntryType: catalog.LegacyBackfillEntryTypeDuplicateEpisodeCandidate, LibraryID: &otherLibraryID, LegacyMediaFileID: &legacyFileFive, Message: "duplicate"},
		{RunID: newerRun.ID, EntryType: catalog.LegacyBackfillEntryTypeSuccess, LibraryID: &libraryID, LegacyMediaItemID: &legacyItemTen, Message: "success-early"},
	}
	for _, entry := range entries {
		if err := db.WithContext(ctx).Create(&entry).Error; err != nil {
			t.Fatalf("seed entry: %v", err)
		}
	}

	listRecorder := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/catalog-migration/runs", nil)
	listRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for runs list, got %d body=%s", listRecorder.Code, listRecorder.Body.String())
	}

	var listResponse struct {
		Data []catalog.LegacyBackfillRun `json:"data"`
	}
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("decode runs list response: %v", err)
	}
	if len(listResponse.Data) < 2 {
		t.Fatalf("expected at least two runs, got %#v", listResponse.Data)
	}
	if listResponse.Data[0].ID != newerRun.ID || listResponse.Data[1].ID != olderRun.ID {
		t.Fatalf("expected newest-first runs [%d, %d], got %#v", newerRun.ID, olderRun.ID, listResponse.Data)
	}

	detailRecorder := httptest.NewRecorder()
	detailRequest := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/catalog-migration/runs/%d", newerRun.ID), nil)
	detailRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(detailRecorder, detailRequest)
	if detailRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for run detail, got %d body=%s", detailRecorder.Code, detailRecorder.Body.String())
	}

	var detailResponse struct {
		Data catalog.LegacyBackfillRun `json:"data"`
	}
	if err := json.Unmarshal(detailRecorder.Body.Bytes(), &detailResponse); err != nil {
		t.Fatalf("decode run detail response: %v", err)
	}
	if detailResponse.Data.ID != newerRun.ID {
		t.Fatalf("expected detail for run %d, got %#v", newerRun.ID, detailResponse.Data)
	}
	if got := len(detailResponse.Data.Entries); got != 4 {
		t.Fatalf("expected four detail entries, got %#v", detailResponse.Data.Entries)
	}
	gotMessages := []string{
		detailResponse.Data.Entries[0].Message,
		detailResponse.Data.Entries[1].Message,
		detailResponse.Data.Entries[2].Message,
		detailResponse.Data.Entries[3].Message,
	}
	wantMessages := []string{"conflict", "duplicate", "success-early", "success-late"}
	for idx, want := range wantMessages {
		if gotMessages[idx] != want {
			t.Fatalf("expected detail messages %v, got %v", wantMessages, gotMessages)
		}
	}
}

func newCatalogMigrationBackfillTestRouter(t *testing.T) (http.Handler, *gorm.DB, *auth.Service, *jobs.Service, *catalog.Service, uint) {
	t.Helper()

	storageRoot := filepath.Join(t.TempDir(), "storage-root")
	if err := os.MkdirAll(storageRoot, 0o755); err != nil {
		t.Fatalf("create storage root: %v", err)
	}

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		Storage:  config.StorageConfig{Provider: "local"},
		Local:    config.LocalStorageConfig{RootPath: storageRoot},
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
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Local", RootPath: storageRoot})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	record, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: storageRoot})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	if err := db.WithContext(ctx).Where("kind = ?", library.JobKindSyncLibrary).Delete(&database.Job{}).Error; err != nil {
		t.Fatalf("clear bootstrap sync job: %v", err)
	}

	return router, db, authSvc, jobsSvc, catalogSvc, record.ID
}

func createBackfillAuthHeader(t *testing.T, ctx context.Context, authSvc *auth.Service) (string, database.User) {
	t.Helper()

	username := fmt.Sprintf("backfill-user-%d", time.Now().UnixNano())
	user, err := authSvc.Register(ctx, username, "password123")
	if err != nil {
		t.Fatalf("register auth user: %v", err)
	}
	loginResult, err := authSvc.Login(ctx, username, "password123")
	if err != nil {
		t.Fatalf("login auth user: %v", err)
	}
	return fmt.Sprintf("Bearer %s", loginResult.Token), user
}

func queueCatalogLegacyBackfillRequest(t *testing.T, router http.Handler, authHeader string, body string) catalog.LegacyBackfillRun {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/catalog-migration/backfill", strings.NewReader(body))
	request.Header.Set("Authorization", authHeader)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Data catalog.LegacyBackfillRun `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode queue response: %v", err)
	}
	return response.Data
}

func assertCatalogBackfillQueuedJob(t *testing.T, ctx context.Context, db *gorm.DB, jobsSvc *jobs.Service, jobKey string, runID uint, libraryID *uint) {
	t.Helper()

	assertCatalogBackfillJobCount(t, ctx, db, 1, jobKey)
	queuedJobs, err := jobsSvc.List(ctx, 10, jobs.StatusQueued, catalog.JobKindLegacyBackfill)
	if err != nil {
		t.Fatalf("list queued backfill jobs: %v", err)
	}
	for _, job := range queuedJobs {
		if job.JobKey != jobKey {
			continue
		}
		var payload catalog.LegacyBackfillPayload
		if err := json.Unmarshal([]byte(job.PayloadJSON), &payload); err != nil {
			t.Fatalf("decode queued job payload: %v", err)
		}
		if payload.RunID != runID {
			t.Fatalf("expected payload run id %d, got %#v", runID, payload)
		}
		switch {
		case libraryID == nil && payload.LibraryID != nil:
			t.Fatalf("expected nil payload library id, got %#v", payload)
		case libraryID != nil && (payload.LibraryID == nil || *payload.LibraryID != *libraryID):
			t.Fatalf("expected payload library id %d, got %#v", *libraryID, payload)
		}
		return
	}
	t.Fatalf("expected queued backfill job with key %q, got %#v", jobKey, queuedJobs)
}

func assertCatalogBackfillJobCount(t *testing.T, ctx context.Context, db *gorm.DB, want int64, jobKey string) {
	t.Helper()

	var count int64
	if err := db.WithContext(ctx).Model(&database.Job{}).Where("job_key = ? AND status IN ?", jobKey, []string{jobs.StatusQueued, jobs.StatusRunning}).Count(&count).Error; err != nil {
		t.Fatalf("count backfill jobs for %q: %v", jobKey, err)
	}
	if count != want {
		t.Fatalf("expected %d active jobs for %q, got %d", want, jobKey, count)
	}
}
