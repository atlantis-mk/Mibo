package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	"github.com/atlan/mibo-media-server/internal/listener"
	"github.com/atlan/mibo-media-server/internal/metadata"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/probe"
	"github.com/atlan/mibo-media-server/internal/progress"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/schedule"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
	"gorm.io/gorm"
)

func TestReadyz(t *testing.T) {
	openList := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    http.StatusOK,
			"message": "success",
			"data": map[string]any{
				"name":   "root",
				"is_dir": true,
				"size":   0,
			},
		})
	}))
	defer openList.Close()

	db, err := database.Open(config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    filepath.Join(t.TempDir(), "mibo.db"),
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		OpenList: config.OpenListConfig{BaseURL: openList.URL, RootPath: "/media"},
		Worker:   config.WorkerConfig{Enabled: true},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settingsSvc), settingsSvc)

	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var body struct {
		RequestID string `json:"request_id"`
		Data      struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.RequestID == "" {
		t.Fatal("expected request_id in response")
	}
	if body.Data.Status != "ready" {
		t.Fatalf("expected ready status, got %q", body.Data.Status)
	}
}

func TestScheduleEndpointsRequireAuthAndValidatePayload(t *testing.T) {
	router, _, _, _ := newScheduleTestRouter(t)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/schedules", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/schedules", strings.NewReader(`{"name":"Bad","kind":"scan","scope_kind":"global","frequency":{"kind":"weekly","time_of_day":"09:00"}}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected auth to be required before validation, got %d", recorder.Code)
	}

	router, authSvc, _, _ := newScheduleTestRouter(t)
	authHeader := createAuthHeader(t, context.Background(), authSvc)
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/schedules", strings.NewReader(`{"name":"Bad","kind":"scan","scope_kind":"global","frequency":{"kind":"weekly","time_of_day":"09:00"}}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed frequency payload, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestScheduleEndpointsCreateRunAndHistory(t *testing.T) {
	router, authSvc, db, jobsSvc := newScheduleTestRouter(t)
	authHeader := createAuthHeader(t, context.Background(), authSvc)

	createRecorder := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", strings.NewReader(`{"name":"Nightly scan","kind":"scan","scope_kind":"global","enabled":true,"frequency":{"kind":"daily","time_of_day":"09:30"}}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createRecorder.Code, createRecorder.Body.String())
	}

	var created struct {
		Data schedule.Schedule `json:"data"`
	}
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Data.ID == 0 || created.Data.NextRunAt == nil {
		t.Fatalf("expected created schedule payload, got %#v", created.Data)
	}

	runRecorder := httptest.NewRecorder()
	runRequest := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/schedules/%d/run", created.Data.ID), nil)
	runRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(runRecorder, runRequest)
	if runRecorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202 for run-now, got %d body=%s", runRecorder.Code, runRecorder.Body.String())
	}

	var queuedJobs []database.Job
	if err := db.WithContext(context.Background()).Order("id desc").Find(&queuedJobs).Error; err != nil {
		t.Fatalf("load jobs: %v", err)
	}
	if len(queuedJobs) == 0 || queuedJobs[0].Kind != schedule.JobKindForSchedule(schedule.KindScan) {
		t.Fatalf("expected queued schedule job, got %#v", queuedJobs)
	}

	historyRecorder := httptest.NewRecorder()
	historyRequest := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/schedules/%d/history", created.Data.ID), nil)
	historyRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(historyRecorder, historyRequest)
	if historyRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for history, got %d body=%s", historyRecorder.Code, historyRecorder.Body.String())
	}

	listRecorder := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/schedules", nil)
	listRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for list, got %d body=%s", listRecorder.Code, listRecorder.Body.String())
	}
	_ = jobsSvc
}

func TestStorageEventEndpointRequiresAuth(t *testing.T) {
	t.Parallel()

	router, _, _, _, libraryID, moviePath := newStorageEventTestRouter(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/storage-events", strings.NewReader(fmt.Sprintf(`{"library_id":%d,"kind":"update","path":%q}`, libraryID, moviePath)))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestStorageEventEndpointEnqueuesListenerRefreshIntent(t *testing.T) {
	t.Parallel()

	for _, kind := range []string{"create", "update", "delete"} {
		kind := kind
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			router, db, authSvc, _, libraryID, moviePath := newStorageEventTestRouter(t)
			authHeader := createAuthHeader(t, context.Background(), authSvc)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/storage-events", strings.NewReader(fmt.Sprintf(`{"library_id":%d,"kind":%q,"path":%q}`, libraryID, kind, moviePath)))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", authHeader)
			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusAccepted {
				t.Fatalf("expected 202, got %d body=%s", recorder.Code, recorder.Body.String())
			}
			acceptedJob := mustDecodeStorageEventResponseJob(t, recorder.Body.Bytes())
			if acceptedJob.Kind != listener.JobKindApplyStorageEventRefresh {
				t.Fatalf("expected response to return listener refresh job, got %q", acceptedJob.Kind)
			}

			var queuedJob database.Job
			if err := db.WithContext(context.Background()).Order("id desc").First(&queuedJob).Error; err != nil {
				t.Fatalf("load queued job: %v", err)
			}
			if queuedJob.Kind != listener.JobKindApplyStorageEventRefresh {
				t.Fatalf("expected listener refresh job, got %q", queuedJob.Kind)
			}
			payload := mustDecodeListenerRefreshPayload(t, queuedJob.PayloadJSON)
			if payload.RootPath != filepath.Join(filepath.Dir(moviePath)) {
				t.Fatalf("expected targeted listener root %q, got %q", filepath.Dir(moviePath), payload.RootPath)
			}
			if payload.FallbackFullSync {
				t.Fatal("expected default storage event handling to stay targeted")
			}
		})
	}
}

func TestStorageEventEndpointAcceptsOpenListRootChildPath(t *testing.T) {
	t.Parallel()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		Storage:  config.StorageConfig{Provider: "openlist"},
		OpenList: config.OpenListConfig{RootPath: "/"},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settingsSvc), settingsSvc)

	ctx := context.Background()
	source := database.MediaSource{Provider: "openlist", Name: "OpenList", RootPath: "/", StorageRef: "/"}
	if err := db.WithContext(ctx).Create(&source).Error; err != nil {
		t.Fatalf("create openlist source: %v", err)
	}
	record := database.Library{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: "/", Status: "active", ScannerEnabled: true}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create openlist root library: %v", err)
	}
	authHeader := createAuthHeader(t, ctx, authSvc)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/storage-events", strings.NewReader(fmt.Sprintf(`{"library_id":%d,"kind":"update","path":"/MovieA.2024.mkv"}`, record.ID)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	acceptedJob := mustDecodeStorageEventResponseJob(t, recorder.Body.Bytes())
	if acceptedJob.Kind != listener.JobKindApplyStorageEventRefresh {
		t.Fatalf("expected response to return listener refresh job, got %q", acceptedJob.Kind)
	}

	var queuedJob database.Job
	if err := db.WithContext(ctx).Order("id desc").First(&queuedJob).Error; err != nil {
		t.Fatalf("load queued job: %v", err)
	}
	if queuedJob.Kind != listener.JobKindApplyStorageEventRefresh {
		t.Fatalf("expected listener refresh job, got %q", queuedJob.Kind)
	}
	payload := mustDecodeListenerRefreshPayload(t, queuedJob.PayloadJSON)
	if payload.RootPath != "/" {
		t.Fatalf("expected root path %q, got %q", "/", payload.RootPath)
	}
	if payload.FallbackFullSync {
		t.Fatal("expected OpenList root child event to stay targeted")
	}
}

func TestStorageEventEndpointRejectsEscapingPath(t *testing.T) {
	t.Parallel()

	router, _, authSvc, _, libraryID, _ := newStorageEventTestRouter(t)
	authHeader := createAuthHeader(t, context.Background(), authSvc)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/storage-events", strings.NewReader(fmt.Sprintf(`{"library_id":%d,"kind":"update","path":"%s"}`, libraryID, filepath.Join(t.TempDir(), "outside", "MovieA.2024.mkv"))))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestStorageEventEndpointFallsBackToListenerFullSyncIntentForUnsupportedKind(t *testing.T) {
	t.Parallel()

	router, db, authSvc, _, libraryID, moviePath := newStorageEventTestRouter(t)
	authHeader := createAuthHeader(t, context.Background(), authSvc)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/storage-events", strings.NewReader(fmt.Sprintf(`{"library_id":%d,"kind":"checksum","path":%q}`, libraryID, moviePath)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	acceptedJob := mustDecodeStorageEventResponseJob(t, recorder.Body.Bytes())
	if acceptedJob.Kind != listener.JobKindApplyStorageEventRefresh {
		t.Fatalf("expected response to return listener fallback job, got %q", acceptedJob.Kind)
	}

	var queuedJob database.Job
	if err := db.WithContext(context.Background()).Order("id desc").First(&queuedJob).Error; err != nil {
		t.Fatalf("load queued job: %v", err)
	}
	if queuedJob.Kind != listener.JobKindApplyStorageEventRefresh {
		t.Fatalf("expected listener fallback job, got %q", queuedJob.Kind)
	}
	payload := mustDecodeListenerRefreshPayload(t, queuedJob.PayloadJSON)
	if !payload.FallbackFullSync {
		t.Fatal("expected unsupported kind to request fallback_full_sync")
	}
}

func TestStorageEventEndpointMoveUsesCommonAncestorIntent(t *testing.T) {
	t.Parallel()

	router, db, authSvc, _, libraryID, moviePath := newStorageEventTestRouter(t)
	authHeader := createAuthHeader(t, context.Background(), authSvc)
	movedPath := filepath.Join(filepath.Dir(moviePath), "Archived", filepath.Base(moviePath))
	if err := os.MkdirAll(filepath.Dir(movedPath), 0o755); err != nil {
		t.Fatalf("create moved dir: %v", err)
	}
	if err := os.WriteFile(movedPath, []byte("movie"), 0o644); err != nil {
		t.Fatalf("write moved movie file: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/storage-events", strings.NewReader(fmt.Sprintf(`{"library_id":%d,"kind":"move","path":%q,"old_path":%q}`, libraryID, movedPath, moviePath)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	acceptedJob := mustDecodeStorageEventResponseJob(t, recorder.Body.Bytes())
	if acceptedJob.Kind != listener.JobKindApplyStorageEventRefresh {
		t.Fatalf("expected response to return listener move job, got %q", acceptedJob.Kind)
	}

	var queuedJob database.Job
	if err := db.WithContext(context.Background()).Order("id desc").First(&queuedJob).Error; err != nil {
		t.Fatalf("load queued job: %v", err)
	}
	payload := mustDecodeListenerRefreshPayload(t, queuedJob.PayloadJSON)
	if payload.RootPath != filepath.Dir(moviePath) {
		t.Fatalf("expected common ancestor %q, got %q", filepath.Dir(moviePath), payload.RootPath)
	}
}

func TestStorageEventEndpointRenameWithMissingOldPathFallsBackToListenerFullSyncIntent(t *testing.T) {
	t.Parallel()

	router, db, authSvc, _, libraryID, moviePath := newStorageEventTestRouter(t)
	authHeader := createAuthHeader(t, context.Background(), authSvc)
	movedPath := filepath.Join(filepath.Dir(moviePath), "Renamed.MovieA.2024.mkv")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/storage-events", strings.NewReader(fmt.Sprintf(`{"library_id":%d,"kind":"rename","path":%q}`, libraryID, movedPath)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	acceptedJob := mustDecodeStorageEventResponseJob(t, recorder.Body.Bytes())
	if acceptedJob.Kind != listener.JobKindApplyStorageEventRefresh {
		t.Fatalf("expected response to return listener rename fallback job, got %q", acceptedJob.Kind)
	}

	var queuedJob database.Job
	if err := db.WithContext(context.Background()).Order("id desc").First(&queuedJob).Error; err != nil {
		t.Fatalf("load queued job: %v", err)
	}
	payload := mustDecodeListenerRefreshPayload(t, queuedJob.PayloadJSON)
	if !payload.FallbackFullSync {
		t.Fatal("expected missing rename old_path to request fallback_full_sync")
	}
}

func TestStorageEventEndpointRejectsEscapingUnsupportedPayload(t *testing.T) {
	t.Parallel()

	router, _, authSvc, _, libraryID, _ := newStorageEventTestRouter(t)
	authHeader := createAuthHeader(t, context.Background(), authSvc)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/storage-events", strings.NewReader(fmt.Sprintf(`{"library_id":%d,"kind":"checksum","path":"%s"}`, libraryID, filepath.Join(t.TempDir(), "outside", "MovieA.2024.mkv"))))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSetupStatus(t *testing.T) {
	tests := []struct {
		name            string
		setup           func(t *testing.T, ctx context.Context, authSvc *auth.Service, librarySvc *library.Service)
		expectCanEnter  bool
		expectInit      bool
		expectUsers     bool
		expectSources   bool
		expectLibraries bool
	}{
		{
			name: "no users keeps hard gate active",
			setup: func(t *testing.T, ctx context.Context, authSvc *auth.Service, librarySvc *library.Service) {
				t.Helper()
			},
			expectCanEnter:  false,
			expectInit:      false,
			expectUsers:     false,
			expectSources:   false,
			expectLibraries: false,
		},
		{
			name: "user only enables soft gate",
			setup: func(t *testing.T, ctx context.Context, authSvc *auth.Service, librarySvc *library.Service) {
				t.Helper()
				if _, err := authSvc.Register(ctx, "admin", "admin123"); err != nil {
					t.Fatalf("register user: %v", err)
				}
			},
			expectCanEnter:  true,
			expectInit:      false,
			expectUsers:     true,
			expectSources:   false,
			expectLibraries: false,
		},
		{
			name: "user plus source plus library completes setup",
			setup: func(t *testing.T, ctx context.Context, authSvc *auth.Service, librarySvc *library.Service) {
				t.Helper()
				if _, err := authSvc.Register(ctx, "admin", "admin123"); err != nil {
					t.Fatalf("register user: %v", err)
				}
			},
			expectCanEnter:  true,
			expectInit:      true,
			expectUsers:     true,
			expectSources:   true,
			expectLibraries: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageRoot := t.TempDir()
			db, err := database.Open(config.DatabaseConfig{
				Driver: "sqlite",
				DSN:    filepath.Join(t.TempDir(), "mibo.db"),
			})
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
			librarySvc := library.NewService(cfg, db, registry, jobsSvc)
			router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settingsSvc), settingsSvc)

			ctx := context.Background()
			if tt.expectLibraries {
				if _, err := authSvc.Register(ctx, "admin", "admin123"); err != nil {
					t.Fatalf("register user: %v", err)
				}
				source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{
					Provider: "local",
					Name:     "Local media",
					RootPath: storageRoot,
				})
				if err != nil {
					t.Fatalf("create source: %v", err)
				}
				if _, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{
					Name:          "Movies",
					Type:          "movies",
					MediaSourceID: source.ID,
					RootPath:      storageRoot,
				}); err != nil {
					t.Fatalf("create library: %v", err)
				}
			} else {
				tt.setup(t, ctx, authSvc, librarySvc)
			}

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil)
			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("setup status code: %d body=%s", recorder.Code, recorder.Body.String())
			}

			var body struct {
				Data struct {
					Initialized     bool `json:"initialized"`
					CanEnterApp     bool `json:"can_enter_app"`
					HasUsers        bool `json:"has_users"`
					HasMediaSources bool `json:"has_media_sources"`
					HasLibraries    bool `json:"has_libraries"`
				} `json:"data"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode setup status: %v", err)
			}

			if body.Data.CanEnterApp != tt.expectCanEnter {
				t.Fatalf("can_enter_app = %v, want %v", body.Data.CanEnterApp, tt.expectCanEnter)
			}
			if body.Data.Initialized != tt.expectInit {
				t.Fatalf("initialized = %v, want %v", body.Data.Initialized, tt.expectInit)
			}
			if body.Data.HasUsers != tt.expectUsers {
				t.Fatalf("has_users = %v, want %v", body.Data.HasUsers, tt.expectUsers)
			}
			if body.Data.HasMediaSources != tt.expectSources {
				t.Fatalf("has_media_sources = %v, want %v", body.Data.HasMediaSources, tt.expectSources)
			}
			if body.Data.HasLibraries != tt.expectLibraries {
				t.Fatalf("has_libraries = %v, want %v", body.Data.HasLibraries, tt.expectLibraries)
			}
		})
	}
}

func TestLibraryItemEndpoints(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 101, "title": "MovieA", "original_title": "MovieA", "release_date": "2024-02-02"}}})
		case "/movie/101":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 101, "title": "MovieA", "original_title": "MovieA", "overview": "Movie overview", "poster_path": "/movie-a.jpg", "backdrop_path": "/movie-a-bg.jpg", "release_date": "2024-02-02", "runtime": 121, "genres": []map[string]any{{"name": "Action"}}, "credits": map[string]any{"cast": []map[string]any{{"name": "Actor A", "character": "Lead", "profile_path": "/actor-a.jpg"}}, "crew": []map[string]any{{"name": "Director A", "job": "Director", "department": "Directing", "profile_path": "/director-a.jpg"}}}})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{}})
		}
	}))
	defer tmdb.Close()

	openList := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var body struct {
			Path string `json:"path"`
		}
		_ = json.NewDecoder(req.Body).Decode(&body)

		switch req.URL.Path {
		case "/api/fs/get":
			isDir := !strings.HasSuffix(body.Path, ".mp4")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    http.StatusOK,
				"message": "success",
				"data":    map[string]any{"name": "movies", "is_dir": isDir, "size": 0},
			})
		case "/api/fs/list":
			content := []map[string]any{}
			if body.Path == "/movies" {
				content = []map[string]any{{"name": "MovieA.2024.mp4", "is_dir": false, "size": 1024}}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    http.StatusOK,
				"message": "success",
				"data":    map[string]any{"content": content},
			})
		case "/api/fs/link":
			_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusOK, "message": "success", "data": map[string]any{"url": "https://media.example.test" + body.Path}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer openList.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		OpenList: config.OpenListConfig{BaseURL: openList.URL, RootPath: "/movies"},
		Metadata: config.MetadataConfig{TMDB: config.TMDBConfig{APIKey: "test-key", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}},
		FFprobe:  config.FFprobeConfig{Enabled: true, Path: writeRouterFakeFFprobe(t), Timeout: 2 * time.Second},
		Worker:   config.WorkerConfig{Enabled: true, PollInterval: time.Millisecond},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	metadataSvc := metadata.NewService(db, cfg.Metadata, settingsSvc)
	progressSvc := progress.NewService(db)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progressSvc, search.NewService(), metadataSvc, settingsSvc)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "openlist", Name: "Home", RootPath: "/movies"})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	createdLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: "/movies"})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	mediaItem := database.MediaItem{
		LibraryID:     createdLibrary.ID,
		Type:          "movie",
		Title:         "MovieA",
		Overview:      "Movie overview",
		GenresJSON:    `["Action"]`,
		CastJSON:      fmt.Sprintf(`[{"name":"Actor A","role":"Lead","avatar_url":%q}]`, tmdb.URL+"/images/actor-a.jpg"),
		DirectorsJSON: fmt.Sprintf(`[{"name":"Director A","role":"Director","avatar_url":%q}]`, tmdb.URL+"/images/director-a.jpg"),
		SourcePath:    "/movies/MovieA.2024.mp4",
		MatchStatus:   metadata.StatusMatched,
		Status:        "ready",
	}
	if err := db.WithContext(ctx).Create(&mediaItem).Error; err != nil {
		t.Fatalf("create media item: %v", err)
	}
	mediaFile := database.MediaFile{
		LibraryID:          createdLibrary.ID,
		MediaItemID:        &mediaItem.ID,
		StoragePath:        mediaItem.SourcePath,
		Container:          "mp4",
		ProbeStatus:        probe.StatusReady,
		VideoCodec:         "h264",
		DurationSeconds:    float64Ptr(7260.25),
		AudioTracksJSON:    `[{"codec":"aac","language":"eng","title":"Stereo","channels":2}]`,
		SubtitleTracksJSON: `[{"codec":"subrip","language":"eng","title":"English"}]`,
	}
	if err := db.WithContext(ctx).Create(&mediaFile).Error; err != nil {
		t.Fatalf("create media file: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/libraries/%d/items", createdLibrary.ID), nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list items status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var listBody struct {
		Data []database.MediaItem `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listBody.Data) != 1 || listBody.Data[0].Title != "MovieA" {
		t.Fatalf("unexpected list response: %#v", listBody.Data)
	}

	request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/libraries/%d", createdLibrary.ID), nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("get library status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var libraryBody struct {
		Data library.LibraryDetail `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &libraryBody); err != nil {
		t.Fatalf("decode library response: %v", err)
	}
	if libraryBody.Data.ID != createdLibrary.ID || libraryBody.Data.MediaItemsCount != 1 || libraryBody.Data.MediaFilesCount != 1 {
		t.Fatalf("unexpected library detail: %#v", libraryBody.Data)
	}

	request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-items/%d", mediaItem.ID), nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("get item status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var itemBody struct {
		Data library.MediaItemDetail `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &itemBody); err != nil {
		t.Fatalf("decode item response: %v", err)
	}
	if itemBody.Data.Title != "MovieA" || len(itemBody.Data.Files) != 1 || len(itemBody.Data.Genres) != 1 || itemBody.Data.Files[0].VideoCodec != "h264" {
		t.Fatalf("unexpected item detail: %#v", itemBody.Data)
	}
	login := registerAndLoginRouterUser(t, ctx, authSvc, "playback-test-user")
	if len(itemBody.Data.Cast) != 1 || itemBody.Data.Cast[0].AvatarURL != tmdb.URL+"/images/actor-a.jpg" || itemBody.Data.Cast[0].Role != "Lead" {
		t.Fatalf("unexpected cast detail: %#v", itemBody.Data.Cast)
	}
	if len(itemBody.Data.Directors) != 1 || itemBody.Data.Directors[0].AvatarURL != tmdb.URL+"/images/director-a.jpg" {
		t.Fatalf("unexpected directors detail: %#v", itemBody.Data.Directors)
	}

	request = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/media-items/%d/match", mediaItem.ID), nil)
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", login.Token))
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("rematch status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-items/%d/playback?client_profile=web", mediaItem.ID), nil)
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", login.Token))
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("playback source status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var playbackBody struct {
		Data playback.PlaybackSource `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &playbackBody); err != nil {
		t.Fatalf("decode playback response: %v", err)
	}
	if !playbackBody.Data.Playable || playbackBody.Data.URL == "" || playbackBody.Data.MediaFileID != mediaFile.ID {
		t.Fatalf("unexpected playback response: %#v", playbackBody.Data)
	}

	request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-files/%d/link", mediaFile.ID), nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("file link status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var linkBody struct {
		Data playback.FileLink `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &linkBody); err != nil {
		t.Fatalf("decode file link response: %v", err)
	}
	if !linkBody.Data.Playable || linkBody.Data.URL == "" {
		t.Fatalf("unexpected file link response: %#v", linkBody.Data)
	}
}

func TestMetadataSettingsEndpoints(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	storageRoot := t.TempDir()
	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		Storage:  config.StorageConfig{Provider: "local"},
		Local:    config.LocalStorageConfig{RootPath: storageRoot},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settingsSvc), settingsSvc)

	ctx := context.Background()
	if _, err := authSvc.Register(ctx, "admin", "admin123"); err != nil {
		t.Fatalf("register user: %v", err)
	}
	login, err := authSvc.Login(ctx, "admin", "admin123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	body := strings.NewReader(`{"tmdb":{"api_key":"tmdb-test-key","base_url":"https://api.themoviedb.org/3","image_base_url":"https://image.tmdb.org/t/p/original","language":"zh-CN","timeout":"12s"},"tvdb":{"api_key":"tvdb-test-key","base_url":"https://api4.thetvdb.com/v4","language":"zh","timeout":"8s"}}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/metadata", body)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("update metadata settings status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/settings/metadata", nil)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("get metadata settings status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Data settings.MetadataSettings `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode metadata settings response: %v", err)
	}
	if !response.Data.TMDB.Configured || !response.Data.TMDB.APIKeyMasked || response.Data.TMDB.Source != "database" {
		t.Fatalf("unexpected tmdb settings: %#v", response.Data.TMDB)
	}
	if !response.Data.TVDB.Configured || !response.Data.TVDB.APIKeyMasked || response.Data.TVDB.Source != "database" {
		t.Fatalf("unexpected tvdb settings: %#v", response.Data.TVDB)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/system/info", nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("system info status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var systemInfo struct {
		Data struct {
			Modules struct {
				Metadata struct {
					Providers struct {
						TMDB struct {
							Configured bool   `json:"configured"`
							Source     string `json:"source"`
						} `json:"tmdb"`
						TVDB struct {
							Configured bool   `json:"configured"`
							Source     string `json:"source"`
						} `json:"tvdb"`
					} `json:"providers"`
				} `json:"metadata"`
			} `json:"modules"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &systemInfo); err != nil {
		t.Fatalf("decode system info response: %v", err)
	}
	if !systemInfo.Data.Modules.Metadata.Providers.TMDB.Configured || systemInfo.Data.Modules.Metadata.Providers.TMDB.Source != "database" {
		t.Fatalf("unexpected system tmdb provider: %#v", systemInfo.Data.Modules.Metadata.Providers.TMDB)
	}
	if !systemInfo.Data.Modules.Metadata.Providers.TVDB.Configured || systemInfo.Data.Modules.Metadata.Providers.TVDB.Source != "database" {
		t.Fatalf("unexpected system tvdb provider: %#v", systemInfo.Data.Modules.Metadata.Providers.TVDB)
	}
}

func TestCatalogMigrationSettingsEndpoints(t *testing.T) {
	testCatalogMigrationSettingsEndpoints(t)
}

func TestCatalogMigrationSystemInfo(t *testing.T) {
	testCatalogMigrationSystemInfo(t)
}

func TestGetMediaItemIncludesTrailerDetail(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{Database: config.DatabaseConfig{Driver: "sqlite"}, Storage: config.StorageConfig{Provider: "local"}, Local: config.LocalStorageConfig{RootPath: t.TempDir()}}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settingsSvc), settingsSvc)

	ctx := context.Background()
	item := database.MediaItem{
		LibraryID:   1,
		Type:        "movie",
		Title:       "MovieA",
		SourcePath:  "/movies/MovieA.2024.mkv",
		MatchStatus: "matched",
		Status:      "ready",
		TrailerJSON: `{"provider":"tmdb","site":"YouTube","key":"abc123","name":"Official Trailer","type":"Trailer","official":true,"language":"en","watch_url":"https://www.youtube.com/watch?v=abc123","embed_url":"https://www.youtube.com/embed/abc123","thumbnail":"https://img.youtube.com/vi/abc123/hqdefault.jpg"}`,
	}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-items/%d", item.ID), nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("get item status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var body struct {
		Data library.MediaItemDetail `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode item response: %v", err)
	}
	if body.Data.Trailer == nil || body.Data.Trailer.Key != "abc123" || body.Data.Trailer.EmbedURL == "" {
		t.Fatalf("unexpected trailer detail: %#v", body.Data.Trailer)
	}
}

func TestGetMediaItemSerializesEmptyCollectionsAsArrays(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{Database: config.DatabaseConfig{Driver: "sqlite"}, Storage: config.StorageConfig{Provider: "local"}, Local: config.LocalStorageConfig{RootPath: t.TempDir()}}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settingsSvc), settingsSvc)

	ctx := context.Background()
	item := database.MediaItem{
		LibraryID:   1,
		Type:        "movie",
		Title:       "MovieA",
		SourcePath:  "/movies/MovieA.2024.mkv",
		MatchStatus: "matched",
		Status:      "ready",
	}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-items/%d", item.ID), nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("get item status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Data library.MediaItemDetail `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode item response: %v", err)
	}

	if response.Data.Genres == nil || len(response.Data.Genres) != 0 {
		t.Fatalf("expected empty genres array, got %#v", response.Data.Genres)
	}
	if response.Data.Cast == nil || len(response.Data.Cast) != 0 {
		t.Fatalf("expected empty cast array, got %#v", response.Data.Cast)
	}
	if response.Data.Directors == nil || len(response.Data.Directors) != 0 {
		t.Fatalf("expected empty directors array, got %#v", response.Data.Directors)
	}
	if response.Data.Files == nil || len(response.Data.Files) != 0 {
		t.Fatalf("expected empty files array, got %#v", response.Data.Files)
	}
}

func TestGetMediaItemOmitsTrailerWhenUnavailable(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{Database: config.DatabaseConfig{Driver: "sqlite"}, Storage: config.StorageConfig{Provider: "local"}, Local: config.LocalStorageConfig{RootPath: t.TempDir()}}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settingsSvc), settingsSvc)

	ctx := context.Background()
	item := database.MediaItem{
		LibraryID:   1,
		Type:        "movie",
		Title:       "MovieB",
		SourcePath:  "/movies/MovieB.2024.mkv",
		MatchStatus: "matched",
		Status:      "ready",
	}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-items/%d", item.ID), nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("get item status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	if strings.Contains(recorder.Body.String(), "\"trailer\"") {
		t.Fatalf("expected trailer field to be omitted, got %s", recorder.Body.String())
	}

	var body struct {
		Data library.MediaItemDetail `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode item response: %v", err)
	}
	if body.Data.Trailer != nil {
		t.Fatalf("expected trailer to be nil, got %#v", body.Data.Trailer)
	}
}

func TestGeneratedArtworkURLsAreAbsoluteAndServed(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	artworkRoot := filepath.Join(t.TempDir(), "artwork")
	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		Storage:  config.StorageConfig{Provider: "local"},
		Local:    config.LocalStorageConfig{RootPath: t.TempDir()},
		FFmpeg:   config.FFmpegConfig{ArtworkRootPath: artworkRoot},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settingsSvc), settingsSvc)

	ctx := context.Background()
	item := database.MediaItem{
		LibraryID:   1,
		Type:        "movie",
		Title:       "MovieC",
		SourcePath:  "/movies/MovieC.2024.mkv",
		MatchStatus: "skipped",
		Status:      "ready",
	}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	posterURL := fmt.Sprintf("/api/v1/media-items/%d/artwork/poster", item.ID)
	backdropURL := fmt.Sprintf("/api/v1/media-items/%d/artwork/backdrop", item.ID)
	if err := db.WithContext(ctx).Model(&database.MediaItem{}).Where("id = ?", item.ID).Updates(map[string]any{"poster_url": posterURL, "backdrop_url": backdropURL}).Error; err != nil {
		t.Fatalf("set generated artwork urls: %v", err)
	}
	artworkDir := filepath.Join(artworkRoot, fmt.Sprintf("%d", item.ID))
	if err := os.MkdirAll(artworkDir, 0o755); err != nil {
		t.Fatalf("create artwork dir: %v", err)
	}
	posterBytes := []byte("poster-image")
	if err := os.WriteFile(filepath.Join(artworkDir, "poster.jpg"), posterBytes, 0o644); err != nil {
		t.Fatalf("write poster art: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-items/%d", item.ID), nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("get item status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var body struct {
		Data library.MediaItemDetail `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode item response: %v", err)
	}
	if body.Data.PosterURL != "http://example.com"+posterURL {
		t.Fatalf("expected absolute poster url, got %q", body.Data.PosterURL)
	}
	if body.Data.BackdropURL != "http://example.com"+backdropURL {
		t.Fatalf("expected absolute backdrop url, got %q", body.Data.BackdropURL)
	}

	artworkRequest := httptest.NewRequest(http.MethodGet, posterURL, nil)
	artworkRecorder := httptest.NewRecorder()
	router.ServeHTTP(artworkRecorder, artworkRequest)
	if artworkRecorder.Code != http.StatusOK {
		t.Fatalf("artwork status: %d body=%s", artworkRecorder.Code, artworkRecorder.Body.String())
	}
	if contentType := artworkRecorder.Header().Get("Content-Type"); !strings.Contains(contentType, "image/jpeg") {
		t.Fatalf("expected image/jpeg, got %q", contentType)
	}
	if string(artworkRecorder.Body.Bytes()) != string(posterBytes) {
		t.Fatalf("unexpected artwork body: %q", artworkRecorder.Body.String())
	}
}

func TestGeneratedCatalogArtworkURLsAreAbsoluteAndServed(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	artworkRoot := filepath.Join(t.TempDir(), "artwork")
	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		Storage:  config.StorageConfig{Provider: "local"},
		Local:    config.LocalStorageConfig{RootPath: t.TempDir()},
		FFmpeg:   config.FFmpegConfig{ArtworkRootPath: artworkRoot},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	catalogSvc := catalog.NewService(db)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settingsSvc), settingsSvc, catalogSvc)

	ctx := context.Background()
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Movie D", Path: "/movies/MovieD.2024.mkv", SortKey: "Movie D", AvailabilityStatus: catalog.AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	posterURL := fmt.Sprintf("/api/v1/items/%d/artwork/poster", item.ID)
	backdropURL := fmt.Sprintf("/api/v1/items/%d/artwork/backdrop", item.ID)
	if err := db.WithContext(ctx).Create([]database.ItemImage{{ItemID: item.ID, ImageType: "poster", URL: posterURL, IsSelected: true}, {ItemID: item.ID, ImageType: "backdrop", URL: backdropURL, IsSelected: true}}).Error; err != nil {
		t.Fatalf("seed catalog images: %v", err)
	}
	artworkDir := filepath.Join(artworkRoot, "catalog", fmt.Sprintf("%d", item.ID))
	if err := os.MkdirAll(artworkDir, 0o755); err != nil {
		t.Fatalf("create catalog artwork dir: %v", err)
	}
	posterBytes := []byte("catalog-poster-image")
	if err := os.WriteFile(filepath.Join(artworkDir, "poster.jpg"), posterBytes, 0o644); err != nil {
		t.Fatalf("write catalog poster art: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/items/%d", item.ID), nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("get catalog item status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var body struct {
		Data catalog.CatalogItemDetail `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode catalog item response: %v", err)
	}
	if len(body.Data.SelectedImages) != 2 {
		t.Fatalf("expected selected images, got %#v", body.Data.SelectedImages)
	}
	selectedByType := map[string]string{}
	for _, image := range body.Data.SelectedImages {
		selectedByType[image.ImageType] = image.URL
	}
	if selectedByType["poster"] != "http://example.com"+posterURL {
		t.Fatalf("expected absolute catalog poster url, got %q", selectedByType["poster"])
	}
	if selectedByType["backdrop"] != "http://example.com"+backdropURL {
		t.Fatalf("expected absolute catalog backdrop url, got %q", selectedByType["backdrop"])
	}

	artworkRequest := httptest.NewRequest(http.MethodGet, posterURL, nil)
	artworkRecorder := httptest.NewRecorder()
	router.ServeHTTP(artworkRecorder, artworkRequest)
	if artworkRecorder.Code != http.StatusOK {
		t.Fatalf("catalog artwork status: %d body=%s", artworkRecorder.Code, artworkRecorder.Body.String())
	}
	if contentType := artworkRecorder.Header().Get("Content-Type"); !strings.Contains(contentType, "image/jpeg") {
		t.Fatalf("expected image/jpeg, got %q", contentType)
	}
	if string(artworkRecorder.Body.Bytes()) != string(posterBytes) {
		t.Fatalf("unexpected catalog artwork body: %q", artworkRecorder.Body.String())
	}
}

func TestManualMetadataSearchEndpoint(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]any{
					{
						"id":             101,
						"title":          "MovieA",
						"original_title": "MovieA",
						"release_date":   "2024-04-20",
						"overview":       "Primary match",
						"poster_path":    "/primary.jpg",
					},
					{
						"id":             102,
						"title":          "MovieA Returns",
						"original_title": "MovieA Returns",
						"release_date":   "2023-05-01",
						"overview":       "Secondary match",
						"poster_path":    "/secondary.jpg",
					},
				},
			})
		case "/movie/101":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             101,
				"title":          "MovieA Official",
				"original_title": "MovieA Official",
				"release_date":   "2024-04-20",
				"overview":       "Updated overview",
				"poster_path":    "/primary.jpg",
				"backdrop_path":  "/backdrop.jpg",
				"genres":         []map[string]any{{"name": "Action"}},
				"credits": map[string]any{
					"cast": []map[string]any{{"name": "Actor A"}},
					"crew": []map[string]any{{"name": "Director A", "job": "Director", "department": "Directing"}},
				},
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

	storageRoot := t.TempDir()
	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		Storage:  config.StorageConfig{Provider: "local"},
		Local:    config.LocalStorageConfig{RootPath: storageRoot},
		Metadata: config.MetadataConfig{TMDB: config.TMDBConfig{APIKey: "test-key", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "zh-CN", Timeout: time.Second}},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	metadataSvc := metadata.NewService(db, cfg.Metadata, settingsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadataSvc, settingsSvc)

	ctx := context.Background()
	if _, err := authSvc.Register(ctx, "admin", "admin123"); err != nil {
		t.Fatalf("register user: %v", err)
	}
	login, err := authSvc.Login(ctx, "admin", "admin123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Movies", RootPath: storageRoot})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	createdLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: storageRoot})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	year := 2024
	item := database.MediaItem{
		LibraryID:   createdLibrary.ID,
		Type:        "movie",
		Title:       "MovieA",
		Year:        &year,
		SourcePath:  filepath.Join(storageRoot, "MovieA.2024.mkv"),
		MatchStatus: metadata.StatusPending,
		Status:      "ready",
	}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create media item: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/media-items/%d/metadata", item.ID), strings.NewReader(`{"title":"Manual"}`))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("metadata update without auth status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/media-items/%d/metadata", item.ID), strings.NewReader(`{"title":""}`))
	req.Header.Set("Authorization", "Bearer "+login.Token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("metadata update validation status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	body := strings.NewReader(`{"title":"MovieA","year":2024}`)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/media-items/%d/metadata/search", item.ID), body)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("metadata search status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Data []metadata.SearchCandidate `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode metadata search response: %v", err)
	}
	if len(response.Data) != 2 {
		t.Fatalf("unexpected result length: %#v", response.Data)
	}
	if response.Data[0].Title != "MovieA" || response.Data[0].Provider != "tmdb" {
		t.Fatalf("unexpected first candidate: %#v", response.Data[0])
	}
	if response.Data[0].Confidence <= response.Data[1].Confidence {
		t.Fatalf("expected primary result first: %#v", response.Data)
	}

	body = strings.NewReader(`{"external_id":"movie:101"}`)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/media-items/%d/metadata/apply", item.ID), body)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("metadata apply status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var applyResponse struct {
		Data library.MediaItemDetail `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &applyResponse); err != nil {
		t.Fatalf("decode metadata apply response: %v", err)
	}
	if applyResponse.Data.Title != "MovieA Official" || applyResponse.Data.MatchStatus == metadata.StatusPending {
		t.Fatalf("unexpected applied item response: %#v", applyResponse.Data)
	}

	body = strings.NewReader(`{"title":"MovieA Manual","original_title":"MovieA Source","year":2025,"overview":"Edited overview","poster_url":"https://images.example.test/poster.jpg","backdrop_url":"https://images.example.test/backdrop.jpg"}`)
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/media-items/%d/metadata", item.ID), body)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("metadata update status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var updateResponse struct {
		Data library.MediaItemDetail `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &updateResponse); err != nil {
		t.Fatalf("decode metadata update response: %v", err)
	}
	if updateResponse.Data.Title != "MovieA Manual" || updateResponse.Data.Overview != "Edited overview" || updateResponse.Data.PosterURL != "https://images.example.test/poster.jpg" {
		t.Fatalf("unexpected updated item response: %#v", updateResponse.Data)
	}

	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/media-items/%d/metadata/refetch", item.ID), nil)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("metadata refetch status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var queuedJob struct {
		Data database.Job `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &queuedJob); err != nil {
		t.Fatalf("decode metadata refetch response: %v", err)
	}
	if queuedJob.Data.Kind != library.JobKindRefetchMediaItem {
		t.Fatalf("unexpected metadata refetch job: %#v", queuedJob.Data)
	}

	file := database.MediaFile{
		LibraryID:   createdLibrary.ID,
		MediaItemID: &item.ID,
		StoragePath: "/movies/MovieA.2024.mkv",
		ProbeStatus: probe.StatusReady,
	}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create media file: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/media-files/%d/probe", file.ID), nil)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("media file reprobe status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &queuedJob); err != nil {
		t.Fatalf("decode media file reprobe response: %v", err)
	}
	if queuedJob.Data.Kind != "probe_media_file" {
		t.Fatalf("unexpected media file reprobe job: %#v", queuedJob.Data)
	}

	var storedFile database.MediaFile
	if err := db.WithContext(ctx).First(&storedFile, file.ID).Error; err != nil {
		t.Fatalf("reload media file: %v", err)
	}
	if storedFile.ProbeStatus != probe.StatusPending {
		t.Fatalf("expected probe status reset to pending, got %q", storedFile.ProbeStatus)
	}
}

func TestAuthAndProgressEndpoints(t *testing.T) {
	openList := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var body struct {
			Path string `json:"path"`
		}
		_ = json.NewDecoder(req.Body).Decode(&body)

		switch req.URL.Path {
		case "/api/fs/get":
			_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusOK, "message": "success", "data": map[string]any{"name": "movies", "is_dir": !strings.HasSuffix(body.Path, ".mkv"), "size": 1024}})
		case "/api/fs/list":
			content := []map[string]any{}
			if body.Path == "/movies" {
				content = []map[string]any{{"name": "MovieA.2024.mkv", "is_dir": false, "size": 1024}}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusOK, "message": "success", "data": map[string]any{"content": content}})
		case "/api/fs/link":
			_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusOK, "message": "success", "data": map[string]any{"url": "https://media.example.test" + body.Path}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer openList.Close()

	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 101, "title": "MovieA", "original_title": "MovieA", "release_date": "2024-02-02"}}})
		case "/movie/101":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 101, "title": "MovieA", "original_title": "MovieA", "overview": "Movie overview", "poster_path": "/movie-a.jpg", "backdrop_path": "/movie-a-bg.jpg", "release_date": "2024-02-02", "runtime": 121, "genres": []map[string]any{{"name": "Action"}}, "credits": map[string]any{"cast": []map[string]any{{"name": "Actor A"}}, "crew": []map[string]any{{"name": "Director A", "job": "Director", "department": "Directing"}}}})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{}})
		}
	}))
	defer tmdb.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		OpenList: config.OpenListConfig{BaseURL: openList.URL, RootPath: "/movies"},
		Metadata: config.MetadataConfig{TMDB: config.TMDBConfig{APIKey: "test-key", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}},
		FFprobe:  config.FFprobeConfig{Enabled: true, Path: writeRouterFakeFFprobe(t), Timeout: 2 * time.Second},
		Worker:   config.WorkerConfig{Enabled: true, PollInterval: time.Millisecond},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	metadataSvc := metadata.NewService(db, cfg.Metadata, settingsSvc)
	progressSvc := progress.NewService(db)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progressSvc, search.NewService(), metadataSvc, settingsSvc)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "openlist", Name: "Home", RootPath: "/movies"})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	createdLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: "/movies"})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	runtimeSeconds := 7260
	mediaItem := database.MediaItem{LibraryID: createdLibrary.ID, Type: "movie", Title: "MovieA", SourcePath: "/movies/MovieA.2024.mkv", MatchStatus: metadata.StatusMatched, Status: "ready", RuntimeSeconds: &runtimeSeconds}
	if err := db.WithContext(ctx).Create(&mediaItem).Error; err != nil {
		t.Fatalf("create media item: %v", err)
	}
	mediaFile := database.MediaFile{LibraryID: createdLibrary.ID, MediaItemID: &mediaItem.ID, StoragePath: mediaItem.SourcePath, Container: "mkv", ProbeStatus: probe.StatusReady, VideoCodec: "h264"}
	if err := db.WithContext(ctx).Create(&mediaFile).Error; err != nil {
		t.Fatalf("create media file: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"username":"alice","password":"password123"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("register status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"alice","password":"password123"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("login status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var loginBody struct {
		Data auth.LoginResult `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &loginBody); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	token := loginBody.Data.Token
	if token == "" {
		t.Fatal("expected session token")
	}

	authHeader := fmt.Sprintf("Bearer %s", token)
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("me without token status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/me/progress", strings.NewReader(fmt.Sprintf(`{"media_item_id":%d,"media_file_id":%d,"position_seconds":180}`, mediaItem.ID, mediaFile.ID)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("update progress status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/me/continue-watching", nil)
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("continue watching status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var continueBody struct {
		Data []progress.Entry `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &continueBody); err != nil {
		t.Fatalf("decode continue response: %v", err)
	}
	if len(continueBody.Data) != 1 || continueBody.Data[0].MediaItem.ID != mediaItem.ID || continueBody.Data[0].PositionSeconds != 180 {
		t.Fatalf("unexpected continue watching response: %#v", continueBody.Data)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-items/%d/progress", mediaItem.ID), nil)
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("item progress status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var stateBody struct {
		Data progress.State `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &stateBody); err != nil {
		t.Fatalf("decode item progress: %v", err)
	}
	if stateBody.Data.PositionSeconds != 180 || stateBody.Data.Watched {
		t.Fatalf("unexpected progress state: %#v", stateBody.Data)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/me/progress", strings.NewReader(fmt.Sprintf(`{"media_item_id":%d,"media_file_id":%d,"position_seconds":7250,"completed":true}`, mediaItem.ID, mediaFile.ID)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("complete progress status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/me/recently-played", nil)
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("recently played status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var recentBody struct {
		Data []progress.Entry `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &recentBody); err != nil {
		t.Fatalf("decode recent response: %v", err)
	}
	if len(recentBody.Data) != 1 || !recentBody.Data[0].Watched {
		t.Fatalf("unexpected recently played response: %#v", recentBody.Data)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("me status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("logout status: %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRecentlyAddedEndpoint(t *testing.T) {
	openList := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var body struct {
			Path string `json:"path"`
		}
		_ = json.NewDecoder(req.Body).Decode(&body)

		switch req.URL.Path {
		case "/api/fs/get":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    http.StatusOK,
				"message": "success",
				"data":    map[string]any{"name": "movies", "is_dir": !strings.HasSuffix(body.Path, ".mkv"), "size": 0},
			})
		case "/api/fs/list":
			content := []map[string]any{}
			if body.Path == "/movies" {
				content = []map[string]any{
					{"name": "MovieA.2024.mkv", "is_dir": false, "size": 1024},
					{"name": "MovieB.2024.mkv", "is_dir": false, "size": 1024},
					{"name": "MovieC.2024.mkv", "is_dir": false, "size": 1024},
					{"name": "MovieD.2024.mkv", "is_dir": false, "size": 1024},
					{"name": "MovieE.2024.mkv", "is_dir": false, "size": 1024},
					{"name": "MovieF.2024.mkv", "is_dir": false, "size": 1024},
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    http.StatusOK,
				"message": "success",
				"data":    map[string]any{"content": content},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer openList.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		OpenList: config.OpenListConfig{BaseURL: openList.URL, RootPath: "/movies"},
		Worker:   config.WorkerConfig{Enabled: true, PollInterval: time.Millisecond},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settingsSvc), settingsSvc)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "openlist", Name: "Home", RootPath: "/movies"})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	createdLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: "/movies"})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}

	items := []database.MediaItem{
		{LibraryID: createdLibrary.ID, Type: "movie", Title: "MovieA", SourcePath: "/movies/MovieA.2024.mkv", MatchStatus: "matched", Status: "ready"},
		{LibraryID: createdLibrary.ID, Type: "movie", Title: "MovieB", SourcePath: "/movies/MovieB.2024.mkv", MatchStatus: "matched", Status: "ready"},
		{LibraryID: createdLibrary.ID, Type: "movie", Title: "MovieC", SourcePath: "/movies/MovieC.2024.mkv", MatchStatus: "matched", Status: "ready"},
		{LibraryID: createdLibrary.ID, Type: "movie", Title: "MovieD", SourcePath: "/movies/MovieD.2024.mkv", MatchStatus: "matched", Status: "ready"},
		{LibraryID: createdLibrary.ID, Type: "movie", Title: "MovieE", SourcePath: "/movies/MovieE.2024.mkv", MatchStatus: "matched", Status: "ready"},
		{LibraryID: createdLibrary.ID, Type: "movie", Title: "MovieF", SourcePath: "/movies/MovieF.2024.mkv", MatchStatus: "matched", Status: "ready"},
	}
	for idx := range items {
		if err := db.WithContext(ctx).Create(&items[idx]).Error; err != nil {
			t.Fatalf("create recently added media item %d: %v", idx, err)
		}
	}

	if err := db.WithContext(ctx).Order("id asc").Find(&items).Error; err != nil {
		t.Fatalf("list media items: %v", err)
	}
	if len(items) != 6 {
		t.Fatalf("expected 6 media items, got %d", len(items))
	}

	baseTime := time.Now().UTC().Add(-6 * time.Hour)
	for index, item := range items {
		createdAt := baseTime.Add(time.Duration(index) * time.Hour)
		if err := db.WithContext(ctx).
			Model(&database.MediaItem{}).
			Where("id = ?", item.ID).
			Update("created_at", createdAt).Error; err != nil {
			t.Fatalf("update media item created_at: %v", err)
		}
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/home/recently-added", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("recently added status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var body struct {
		Data []database.MediaItem `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode recently added response: %v", err)
	}
	if len(body.Data) != 5 {
		t.Fatalf("expected 5 recently added items, got %d", len(body.Data))
	}

	expectedTitles := []string{"MovieF", "MovieE", "MovieD", "MovieC", "MovieB"}
	for index, title := range expectedTitles {
		if body.Data[index].Title != title {
			t.Fatalf("unexpected recently added order at %d: got %q want %q", index, body.Data[index].Title, title)
		}
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/home/recently-added?limit=999", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("recently added fallback status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode fallback recently added response: %v", err)
	}
	if len(body.Data) != 5 {
		t.Fatalf("expected fallback limit to return 5 items, got %d", len(body.Data))
	}
}

func TestRecentlyAddedEndpointGroupsShowEpisodes(t *testing.T) {
	router, db, authSvc, librarySvc, storageRoot := newDeleteTestRouter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	moviesDir := filepath.Join(storageRoot, "recently-added", "Movies")
	showsDir := filepath.Join(storageRoot, "recently-added", "Shows")
	for _, dir := range []string{moviesDir, showsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create recently-added dir: %v", err)
		}
	}

	movieSource, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Recent Movies Source", RootPath: moviesDir})
	if err != nil {
		t.Fatalf("create recent movies source: %v", err)
	}
	showSource, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Recent Shows Source", RootPath: showsDir})
	if err != nil {
		t.Fatalf("create recent shows source: %v", err)
	}

	movieLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Recent Movies", Type: "movies", MediaSourceID: movieSource.ID, RootPath: moviesDir})
	if err != nil {
		t.Fatalf("create recent movies library: %v", err)
	}
	showLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Recent Shows", Type: "shows", MediaSourceID: showSource.ID, RootPath: showsDir})
	if err != nil {
		t.Fatalf("create recent shows library: %v", err)
	}

	_ = registerAndLoginRouterUser(t, ctx, authSvc, "recent-user")
	createdAt := time.Now().UTC()
	year2025 := 2025
	entries := []database.MediaItem{
		{LibraryID: movieLibrary.ID, Type: "movie", Title: "Movie New", SourcePath: filepath.Join(moviesDir, "movie-new.mkv"), MatchStatus: "matched", Status: "ready", CreatedAt: createdAt.Add(-3 * time.Minute), UpdatedAt: createdAt.Add(-3 * time.Minute)},
		{LibraryID: showLibrary.ID, Type: "episode", Title: "灵笼 S02E01", SeriesTitle: "灵笼", Year: &year2025, SeasonNumber: intPtr(2), EpisodeNumber: intPtr(1), SourcePath: filepath.Join(showsDir, "ling-cage-s02e01.mkv"), MatchStatus: "matched", Status: "ready", CreatedAt: createdAt.Add(-1 * time.Minute), UpdatedAt: createdAt.Add(-1 * time.Minute)},
		{LibraryID: showLibrary.ID, Type: "episode", Title: "灵笼 S02E02", SeriesTitle: "灵笼", Year: &year2025, SeasonNumber: intPtr(2), EpisodeNumber: intPtr(2), SourcePath: filepath.Join(showsDir, "ling-cage-s02e02.mkv"), MatchStatus: "matched", Status: "ready", CreatedAt: createdAt.Add(-2 * time.Minute), UpdatedAt: createdAt.Add(-2 * time.Minute)},
	}
	for idx := range entries {
		if err := db.WithContext(ctx).Create(&entries[idx]).Error; err != nil {
			t.Fatalf("create recently-added media item: %v", err)
		}
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/home/recently-added?limit=6", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("recently added grouped status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var body struct {
		Data []database.MediaItem `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode grouped recently added response: %v", err)
	}
	if len(body.Data) != 2 {
		t.Fatalf("expected grouped recently added items, got %#v", body.Data)
	}
	if body.Data[0].Type != "show" || body.Data[0].Title != "灵笼" {
		t.Fatalf("expected grouped show card first, got %#v", body.Data[0])
	}
	if body.Data[1].Type != "movie" || body.Data[1].Title != "Movie New" {
		t.Fatalf("expected movie card second, got %#v", body.Data[1])
	}
}

func TestCatalogBrowseFilters(t *testing.T) {
	router, db, authSvc, librarySvc, storageRoot := newDeleteTestRouter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mediaRoot := filepath.Join(storageRoot, "catalog")
	if err := os.MkdirAll(mediaRoot, 0o755); err != nil {
		t.Fatalf("create catalog root: %v", err)
	}

	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{
		Provider: "local",
		Name:     "Catalog Source",
		RootPath: mediaRoot,
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}

	libraryRecord, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{
		Name:          "Catalog Library",
		Type:          "shows",
		MediaSourceID: source.ID,
		RootPath:      mediaRoot,
	})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	if err := db.WithContext(ctx).Model(&database.Library{}).Where("id = ?", libraryRecord.ID).Update("status", "active").Error; err != nil {
		t.Fatalf("activate library: %v", err)
	}

	login := registerAndLoginRouterUser(t, ctx, authSvc, "catalog-user")
	year2024 := 2024
	year2023 := 2023
	createdAt := time.Now().UTC()

	items := []database.MediaItem{
		{LibraryID: libraryRecord.ID, Type: "movie", Title: "Movie 2023", Year: &year2023, SourcePath: filepath.Join(mediaRoot, "movie-2023.mkv"), MatchStatus: "matched", Status: "ready", CreatedAt: createdAt.Add(-4 * time.Hour), UpdatedAt: createdAt.Add(-4 * time.Hour)},
		{LibraryID: libraryRecord.ID, Type: "movie", Title: "Movie 2024", Year: &year2024, SourcePath: filepath.Join(mediaRoot, "movie-2024.mkv"), MatchStatus: "matched", Status: "ready", CreatedAt: createdAt.Add(-3 * time.Hour), UpdatedAt: createdAt.Add(-3 * time.Hour)},
		{LibraryID: libraryRecord.ID, Type: "episode", Title: "Pilot", SeriesTitle: "Show One", ExternalID: "tmdb:show-1", Year: &year2024, SeasonNumber: intPtr(1), EpisodeNumber: intPtr(1), SourcePath: filepath.Join(mediaRoot, "show-one-s01e01.mkv"), MatchStatus: "matched", Status: "ready", CreatedAt: createdAt.Add(-2 * time.Hour), UpdatedAt: createdAt.Add(-2 * time.Hour)},
		{LibraryID: libraryRecord.ID, Type: "episode", Title: "Episode Two", SeriesTitle: "Show One", ExternalID: "tmdb:show-1", Year: &year2024, SeasonNumber: intPtr(1), EpisodeNumber: intPtr(2), SourcePath: filepath.Join(mediaRoot, "show-one-s01e02.mkv"), MatchStatus: "matched", Status: "ready", CreatedAt: createdAt.Add(-1 * time.Hour), UpdatedAt: createdAt.Add(-1 * time.Hour)},
	}
	for idx := range items {
		if err := db.WithContext(ctx).Create(&items[idx]).Error; err != nil {
			t.Fatalf("create media item %d: %v", idx, err)
		}
	}

	progressRecord := database.PlaybackProgress{
		UserID:          login.User.ID,
		MediaItemID:     items[2].ID,
		PositionSeconds: 120,
		Watched:         true,
		LastPlayedAt:    timePtr(createdAt),
	}
	if err := db.WithContext(ctx).Create(&progressRecord).Error; err != nil {
		t.Fatalf("create progress: %v", err)
	}

	t.Run("filters movies by year", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/libraries/%d/items?type=movie&year=2024&sort=year", libraryRecord.ID), nil)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("list filtered items status: %d body=%s", recorder.Code, recorder.Body.String())
		}

		var body struct {
			Data []database.MediaItem `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode filtered response: %v", err)
		}
		if len(body.Data) != 1 || body.Data[0].Title != "Movie 2024" {
			t.Fatalf("unexpected filtered movies: %#v", body.Data)
		}
	})

	t.Run("groups shows into one discovery row", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/libraries/%d/items?type=show&sort=recent", libraryRecord.ID), nil)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("list grouped shows status: %d body=%s", recorder.Code, recorder.Body.String())
		}

		var body struct {
			Data []database.MediaItem `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode grouped shows response: %v", err)
		}
		if len(body.Data) != 1 {
			t.Fatalf("expected one grouped show, got %#v", body.Data)
		}
		if body.Data[0].Type != "show" || body.Data[0].Title != "Show One" {
			t.Fatalf("unexpected grouped show card: %#v", body.Data[0])
		}
	})

	t.Run("sorts by watch status", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/libraries/%d/items?sort=watch_status", libraryRecord.ID), nil)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("watch status sort status: %d body=%s", recorder.Code, recorder.Body.String())
		}

		var body struct {
			Data []database.MediaItem `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode watch status response: %v", err)
		}
		if len(body.Data) < 2 {
			t.Fatalf("expected multiple items for watch status sort, got %#v", body.Data)
		}
		if body.Data[0].Title == "Show One" {
			t.Fatalf("expected watched show to sort after unwatched items, got %#v", body.Data)
		}
	})
}

func TestDiscoveryEndpointsShareRegionRatingWatchedAndHighlightSemantics(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	storageRoot := filepath.Join(t.TempDir(), "discovery-root")
	if err := os.MkdirAll(storageRoot, 0o755); err != nil {
		t.Fatalf("create storage root: %v", err)
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
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	searchSvc := search.NewService(db, librarySvc)
	metadataSvc := metadata.NewService(db, cfg.Metadata, settingsSvc, searchSvc)
	progressSvc := progress.NewService(db, searchSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progressSvc, searchSvc, metadataSvc, settingsSvc)

	source := database.MediaSource{Name: "Local", Provider: "local", StorageRef: "local", RootPath: storageRoot}
	if err := db.WithContext(ctx).Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	record := database.Library{Name: "Discovery", Type: "movies", MediaSourceID: source.ID, RootPath: storageRoot, Status: "active", ScannerEnabled: true}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}

	login := registerAndLoginRouterUser(t, ctx, authSvc, "discovery-user")
	authHeader := fmt.Sprintf("Bearer %s", login.Token)
	year := 2024
	movieRating := 8.7
	showRating := 7.9
	items := []database.MediaItem{
		{
			LibraryID:     record.ID,
			Type:          "movie",
			Title:         "Tokyo Mystery",
			OriginalTitle: "Tokyo Mystery",
			Overview:      "Actor A investigates a Tokyo case.",
			GenresJSON:    `["Action"]`,
			RegionsJSON:   `["Japan"]`,
			CastJSON:      `[{"name":"Actor A","role":"Lead"}]`,
			DirectorsJSON: `[{"name":"Director A","role":"Director"}]`,
			Year:          &year,
			VoteAverage:   &movieRating,
			SourcePath:    filepath.Join(storageRoot, "tokyo-mystery.mkv"),
			MatchStatus:   "matched",
			Status:        "ready",
		},
		{
			LibraryID:     record.ID,
			Type:          "episode",
			Title:         "Pilot",
			OriginalTitle: "Pilot",
			SeriesTitle:   "Show Detectives",
			Overview:      "Actor A leads the team.",
			GenresJSON:    `["Drama"]`,
			RegionsJSON:   `["United States"]`,
			CastJSON:      `[{"name":"Actor A","role":"Lead"}]`,
			DirectorsJSON: `[{"name":"Director B","role":"Director"}]`,
			Year:          &year,
			VoteAverage:   &showRating,
			SeasonNumber:  intPtr(1),
			EpisodeNumber: intPtr(1),
			SourcePath:    filepath.Join(storageRoot, "show-detectives-s01e01.mkv"),
			MatchStatus:   "matched",
			Status:        "ready",
		},
		{
			LibraryID:     record.ID,
			Type:          "episode",
			Title:         "Second Case",
			OriginalTitle: "Second Case",
			SeriesTitle:   "Show Detectives",
			Overview:      "Actor A returns for case two.",
			GenresJSON:    `["Drama"]`,
			RegionsJSON:   `["United States"]`,
			CastJSON:      `[{"name":"Actor A","role":"Lead"}]`,
			DirectorsJSON: `[{"name":"Director B","role":"Director"}]`,
			Year:          &year,
			VoteAverage:   &showRating,
			SeasonNumber:  intPtr(1),
			EpisodeNumber: intPtr(2),
			SourcePath:    filepath.Join(storageRoot, "show-detectives-s01e02.mkv"),
			MatchStatus:   "matched",
			Status:        "ready",
		},
	}
	for idx := range items {
		if err := db.WithContext(ctx).Create(&items[idx]).Error; err != nil {
			t.Fatalf("create item %d: %v", idx, err)
		}
		if err := searchSvc.ReindexMediaItem(ctx, items[idx].ID); err != nil {
			t.Fatalf("reindex item %d: %v", idx, err)
		}
	}

	if _, err := progressSvc.Update(ctx, login.User.ID, progress.UpdateInput{MediaItemID: items[0].ID, PositionSeconds: 180, DurationSeconds: intPtr(1200)}); err != nil {
		t.Fatalf("mark movie in progress: %v", err)
	}
	if _, err := progressSvc.Update(ctx, login.User.ID, progress.UpdateInput{MediaItemID: items[1].ID, PositionSeconds: 1180, DurationSeconds: intPtr(1200), Completed: true}); err != nil {
		t.Fatalf("mark show watched: %v", err)
	}

	t.Run("region and rating filters stay aligned across discovery and browse", func(t *testing.T) {
		discoveryRecorder := httptest.NewRecorder()
		discoveryRequest := httptest.NewRequest(http.MethodGet, "/api/v1/discovery?scope=library&library_id="+fmt.Sprint(record.ID)+"&region=Japan&min_rating=8", nil)
		discoveryRequest.Header.Set("Authorization", authHeader)
		router.ServeHTTP(discoveryRecorder, discoveryRequest)
		if discoveryRecorder.Code != http.StatusOK {
			t.Fatalf("discovery filter status: %d body=%s", discoveryRecorder.Code, discoveryRecorder.Body.String())
		}

		var discoveryBody struct {
			Data struct {
				Items []library.DiscoveryItem `json:"items"`
			} `json:"data"`
		}
		if err := json.Unmarshal(discoveryRecorder.Body.Bytes(), &discoveryBody); err != nil {
			t.Fatalf("decode discovery filter response: %v", err)
		}
		if len(discoveryBody.Data.Items) != 1 || discoveryBody.Data.Items[0].Item.Title != "Tokyo Mystery" {
			t.Fatalf("unexpected discovery filter result: %#v", discoveryBody.Data.Items)
		}

		browseRecorder := httptest.NewRecorder()
		browseRequest := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/libraries/%d/items?region=Japan&min_rating=8", record.ID), nil)
		browseRequest.Header.Set("Authorization", authHeader)
		router.ServeHTTP(browseRecorder, browseRequest)
		if browseRecorder.Code != http.StatusOK {
			t.Fatalf("browse filter status: %d body=%s", browseRecorder.Code, browseRecorder.Body.String())
		}

		var browseBody struct {
			Data []database.MediaItem `json:"data"`
		}
		if err := json.Unmarshal(browseRecorder.Body.Bytes(), &browseBody); err != nil {
			t.Fatalf("decode browse filter response: %v", err)
		}
		if len(browseBody.Data) != 1 || browseBody.Data[0].Title != "Tokyo Mystery" {
			t.Fatalf("unexpected browse filter result: %#v", browseBody.Data)
		}
	})

	t.Run("watched state filters stay aligned across discovery and browse", func(t *testing.T) {
		discoveryRecorder := httptest.NewRecorder()
		discoveryRequest := httptest.NewRequest(http.MethodGet, "/api/v1/discovery?scope=library&library_id="+fmt.Sprint(record.ID)+"&watched_state=in_progress", nil)
		discoveryRequest.Header.Set("Authorization", authHeader)
		router.ServeHTTP(discoveryRecorder, discoveryRequest)
		if discoveryRecorder.Code != http.StatusOK {
			t.Fatalf("discovery watched filter status: %d body=%s", discoveryRecorder.Code, discoveryRecorder.Body.String())
		}

		var discoveryBody struct {
			Data struct {
				Items []library.DiscoveryItem `json:"items"`
			} `json:"data"`
		}
		if err := json.Unmarshal(discoveryRecorder.Body.Bytes(), &discoveryBody); err != nil {
			t.Fatalf("decode discovery watched response: %v", err)
		}
		if len(discoveryBody.Data.Items) != 1 || discoveryBody.Data.Items[0].Item.Title != "Tokyo Mystery" || discoveryBody.Data.Items[0].WatchedState != "in_progress" {
			t.Fatalf("unexpected discovery watched result: %#v", discoveryBody.Data.Items)
		}

		browseRecorder := httptest.NewRecorder()
		browseRequest := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/libraries/%d/items?watched_state=in_progress", record.ID), nil)
		browseRequest.Header.Set("Authorization", authHeader)
		router.ServeHTTP(browseRecorder, browseRequest)
		if browseRecorder.Code != http.StatusOK {
			t.Fatalf("browse watched filter status: %d body=%s", browseRecorder.Code, browseRecorder.Body.String())
		}

		var browseBody struct {
			Data []database.MediaItem `json:"data"`
		}
		if err := json.Unmarshal(browseRecorder.Body.Bytes(), &browseBody); err != nil {
			t.Fatalf("decode browse watched response: %v", err)
		}
		if len(browseBody.Data) != 1 || browseBody.Data[0].Title != "Tokyo Mystery" {
			t.Fatalf("unexpected browse watched result: %#v", browseBody.Data)
		}
	})

	t.Run("search highlights and media type distinction survive projection-backed reads", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/discovery?scope=library&library_id="+fmt.Sprint(record.ID)+"&q=Actor%20A", nil)
		request.Header.Set("Authorization", authHeader)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("discovery search status: %d body=%s", recorder.Code, recorder.Body.String())
		}

		var body struct {
			Data struct {
				Items []search.Result `json:"items"`
			} `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode discovery search response: %v", err)
		}
		if len(body.Data.Items) != 2 {
			t.Fatalf("expected two discovery search hits, got %#v", body.Data.Items)
		}
		if body.Data.Items[0].Highlight == "" || body.Data.Items[1].Highlight == "" {
			t.Fatalf("expected non-empty highlights, got %#v", body.Data.Items)
		}
		types := []string{body.Data.Items[0].Item.Type, body.Data.Items[1].Item.Type}
		if !(containsString(types, "movie") && containsString(types, "show")) {
			t.Fatalf("expected movie/show distinction, got %#v", types)
		}
	})
}

func TestHomeDiscoveryEndpoint(t *testing.T) {
	router, db, authSvc, librarySvc, storageRoot := newDeleteTestRouter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mediaRoot := filepath.Join(storageRoot, "home")
	moviesDir := filepath.Join(mediaRoot, "Movies")
	showsDir := filepath.Join(mediaRoot, "Shows")
	if err := os.MkdirAll(moviesDir, 0o755); err != nil {
		t.Fatalf("create movies dir: %v", err)
	}
	if err := os.MkdirAll(showsDir, 0o755); err != nil {
		t.Fatalf("create shows dir: %v", err)
	}

	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{
		Provider: "local",
		Name:     "Home Source",
		RootPath: mediaRoot,
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}

	movieLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{
		Name:          "Movies",
		Type:          "movies",
		MediaSourceID: source.ID,
		RootPath:      moviesDir,
	})
	if err != nil {
		t.Fatalf("create movie library: %v", err)
	}
	showLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{
		Name:          "Shows",
		Type:          "shows",
		MediaSourceID: source.ID,
		RootPath:      showsDir,
	})
	if err != nil {
		t.Fatalf("create show library: %v", err)
	}
	if err := db.WithContext(ctx).Model(&database.Library{}).Where("id IN ?", []uint{movieLibrary.ID, showLibrary.ID}).Update("status", "active").Error; err != nil {
		t.Fatalf("activate libraries: %v", err)
	}

	login := registerAndLoginRouterUser(t, ctx, authSvc, "home-user")
	year2024 := 2024
	createdAt := time.Now().UTC()

	movie := database.MediaItem{LibraryID: movieLibrary.ID, Type: "movie", Title: "Library Movie", Year: &year2024, SourcePath: filepath.Join(moviesDir, "library-movie.mkv"), MatchStatus: "matched", Status: "ready", CreatedAt: createdAt.Add(-2 * time.Hour), UpdatedAt: createdAt.Add(-2 * time.Hour)}
	showEpisode := database.MediaItem{LibraryID: showLibrary.ID, Type: "episode", Title: "Pilot", SeriesTitle: "Library Show", ExternalID: "tmdb:library-show", Year: &year2024, SeasonNumber: intPtr(1), EpisodeNumber: intPtr(1), SourcePath: filepath.Join(showsDir, "library-show-s01e01.mkv"), MatchStatus: "matched", Status: "ready", CreatedAt: createdAt.Add(-1 * time.Hour), UpdatedAt: createdAt.Add(-1 * time.Hour)}
	for _, item := range []*database.MediaItem{&movie, &showEpisode} {
		if err := db.WithContext(ctx).Create(item).Error; err != nil {
			t.Fatalf("create home media item: %v", err)
		}
	}

	lastPlayed := createdAt.Add(-30 * time.Minute)
	progressRecords := []database.PlaybackProgress{
		{UserID: login.User.ID, MediaItemID: movie.ID, PositionSeconds: 180, Watched: false, LastPlayedAt: &lastPlayed},
		{UserID: login.User.ID, MediaItemID: showEpisode.ID, PositionSeconds: 2400, Watched: true, LastPlayedAt: timePtr(createdAt)},
	}
	for _, progressRecord := range progressRecords {
		if err := db.WithContext(ctx).Create(&progressRecord).Error; err != nil {
			t.Fatalf("create playback progress: %v", err)
		}
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/home/discovery", nil)
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", login.Token))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("home discovery status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	if strings.Contains(recorder.Body.String(), "root_path") || strings.Contains(recorder.Body.String(), "storage_provider") {
		t.Fatalf("home discovery leaked provider-centric fields: %s", recorder.Body.String())
	}

	var body struct {
		Data homeDiscoveryResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode home discovery response: %v", err)
	}
	if len(body.Data.ContinueWatching) != 1 || body.Data.ContinueWatching[0].MediaItem.ID != movie.ID {
		t.Fatalf("unexpected continue watching payload: %#v", body.Data.ContinueWatching)
	}
	if len(body.Data.RecentlyPlayed) != 2 {
		t.Fatalf("unexpected recently played payload: %#v", body.Data.RecentlyPlayed)
	}
	if len(body.Data.LatestByLibrary) != 2 {
		t.Fatalf("expected two latest-by-library sections, got %#v", body.Data.LatestByLibrary)
	}
	if body.Data.LatestByLibrary[0].LibraryName == "" || len(body.Data.LatestByLibrary[0].Items) == 0 {
		t.Fatalf("unexpected latest-by-library section: %#v", body.Data.LatestByLibrary)
	}
}

func TestLatestByLibraryEndpoint(t *testing.T) {
	router, db, authSvc, librarySvc, storageRoot := newDeleteTestRouter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	login := registerAndLoginRouterUser(t, ctx, authSvc, "latest-libraries")

	moviesDir := filepath.Join(storageRoot, "latest-by-library", "Movies")
	showsDir := filepath.Join(storageRoot, "latest-by-library", "Shows")
	for _, dir := range []string{moviesDir, showsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create latest-by-library dir: %v", err)
		}
	}

	movieSource, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{
		Provider: "local",
		Name:     "Latest Movies Source",
		RootPath: moviesDir,
	})
	if err != nil {
		t.Fatalf("create latest movies source: %v", err)
	}
	showSource, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{
		Provider: "local",
		Name:     "Latest Shows Source",
		RootPath: showsDir,
	})
	if err != nil {
		t.Fatalf("create latest shows source: %v", err)
	}

	movieLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{
		Name:          "Local Movies",
		Type:          "movies",
		MediaSourceID: movieSource.ID,
		RootPath:      moviesDir,
	})
	if err != nil {
		t.Fatalf("create latest movies library: %v", err)
	}
	showLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{
		Name:          "Local Shows",
		Type:          "shows",
		MediaSourceID: showSource.ID,
		RootPath:      showsDir,
	})
	if err != nil {
		t.Fatalf("create latest shows library: %v", err)
	}

	createdAt := time.Now().UTC()
	entries := make([]database.MediaItem, 0, 15)
	for idx := 0; idx < 14; idx++ {
		timestamp := createdAt.Add(-1 * time.Duration(idx) * time.Minute)
		entries = append(entries, database.MediaItem{
			LibraryID:   movieLibrary.ID,
			Type:        "movie",
			Title:       fmt.Sprintf("Movie %02d", idx+1),
			SourcePath:  filepath.Join(moviesDir, fmt.Sprintf("movie-%02d.mkv", idx+1)),
			MatchStatus: "matched",
			Status:      "ready",
			CreatedAt:   timestamp,
			UpdatedAt:   timestamp,
		})
	}
	entries = append(entries, database.MediaItem{
		LibraryID:   showLibrary.ID,
		Type:        "episode",
		Title:       "Pilot",
		SeriesTitle: "Newest Show",
		SourcePath:  filepath.Join(showsDir, "newest-show-s01e01.mkv"),
		MatchStatus: "matched",
		Status:      "ready",
		CreatedAt:   createdAt.Add(-30 * time.Minute),
		UpdatedAt:   createdAt.Add(-30 * time.Minute),
	})
	entries = append(entries, database.MediaItem{
		LibraryID:     showLibrary.ID,
		Type:          "episode",
		Title:         "Episode Two",
		SeriesTitle:   "Newest Show",
		SeasonNumber:  intPtr(1),
		EpisodeNumber: intPtr(2),
		SourcePath:    filepath.Join(showsDir, "newest-show-s01e02.mkv"),
		MatchStatus:   "matched",
		Status:        "ready",
		CreatedAt:     createdAt.Add(-29 * time.Minute),
		UpdatedAt:     createdAt.Add(-29 * time.Minute),
	})
	for idx := range entries {
		if err := db.WithContext(ctx).Create(&entries[idx]).Error; err != nil {
			t.Fatalf("create latest-by-library media item: %v", err)
		}
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/home/latest-by-library", nil)
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", login.Token))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("latest-by-library status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var body struct {
		Data []library.LatestByLibrarySection `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode latest-by-library response: %v", err)
	}
	if len(body.Data) != 2 {
		t.Fatalf("expected two latest-by-library sections, got %#v", body.Data)
	}
	if body.Data[0].LibraryID != movieLibrary.ID || body.Data[0].LibraryName != movieLibrary.Name {
		t.Fatalf("unexpected first latest-by-library section: %#v", body.Data[0])
	}
	if len(body.Data[0].Items) != 12 || body.Data[0].Items[0].Title != "Movie 01" || body.Data[0].Items[11].Title != "Movie 12" {
		t.Fatalf("unexpected movie library items ordering: %#v", body.Data[0].Items)
	}
	if body.Data[1].LibraryID != showLibrary.ID || len(body.Data[1].Items) != 1 {
		t.Fatalf("unexpected show library section: %#v", body.Data[1])
	}
	if body.Data[1].Items[0].Type != "show" || body.Data[1].Items[0].Title != "Newest Show" {
		t.Fatalf("expected grouped show item in latest-by-library, got %#v", body.Data[1].Items[0])
	}
}

func TestPlaybackEndpointRequiresAuth(t *testing.T) {
	router, db, _, librarySvc, storageRoot := newDeleteTestRouter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mediaRoot := filepath.Join(storageRoot, "playback-auth")
	if err := os.MkdirAll(mediaRoot, 0o755); err != nil {
		t.Fatalf("create media root: %v", err)
	}

	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{
		Provider: "local",
		Name:     "Playback Source",
		RootPath: mediaRoot,
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}

	createdLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{
		Name:          "Playback Library",
		Type:          "movies",
		MediaSourceID: source.ID,
		RootPath:      mediaRoot,
	})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}

	mediaItem := database.MediaItem{
		LibraryID:   createdLibrary.ID,
		Type:        "movie",
		Title:       "Playback Auth Movie",
		SourcePath:  filepath.Join(mediaRoot, "playback-auth.mkv"),
		MatchStatus: "matched",
		Status:      "ready",
	}
	if err := db.WithContext(ctx).Create(&mediaItem).Error; err != nil {
		t.Fatalf("create media item: %v", err)
	}

	mediaFile := database.MediaFile{
		LibraryID:   createdLibrary.ID,
		MediaItemID: &mediaItem.ID,
		StoragePath: mediaItem.SourcePath,
		ProbeStatus: "ready",
	}
	if err := db.WithContext(ctx).Create(&mediaFile).Error; err != nil {
		t.Fatalf("create media file: %v", err)
	}
	if err := os.WriteFile(mediaFile.StoragePath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write media file: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-items/%d/playback", mediaItem.ID), nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestPlaybackEndpointRejectsMissingAndInvalidClientProfile(t *testing.T) {
	router, db, authSvc, librarySvc, storageRoot := newDeleteTestRouter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mediaRoot := filepath.Join(storageRoot, "playback-profile")
	if err := os.MkdirAll(mediaRoot, 0o755); err != nil {
		t.Fatalf("create media root: %v", err)
	}

	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{
		Provider: "local",
		Name:     "Playback Source",
		RootPath: mediaRoot,
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}

	createdLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{
		Name:          "Playback Library",
		Type:          "movies",
		MediaSourceID: source.ID,
		RootPath:      mediaRoot,
	})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}

	mediaItem := database.MediaItem{LibraryID: createdLibrary.ID, Type: "movie", Title: "Playback Movie", SourcePath: filepath.Join(mediaRoot, "movie.mp4"), MatchStatus: "matched", Status: "ready"}
	if err := db.WithContext(ctx).Create(&mediaItem).Error; err != nil {
		t.Fatalf("create media item: %v", err)
	}
	mediaFile := database.MediaFile{LibraryID: createdLibrary.ID, MediaItemID: &mediaItem.ID, StoragePath: mediaItem.SourcePath, Container: "mp4", ProbeStatus: probe.StatusReady, VideoCodec: "h264"}
	if err := db.WithContext(ctx).Create(&mediaFile).Error; err != nil {
		t.Fatalf("create media file: %v", err)
	}
	if err := os.WriteFile(mediaFile.StoragePath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write media file: %v", err)
	}

	authHeader := createAuthHeader(t, ctx, authSvc)

	t.Run("missing profile", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-items/%d/playback", mediaItem.ID), nil)
		request.Header.Set("Authorization", authHeader)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("invalid profile", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-items/%d/playback?client_profile=desktop", mediaItem.ID), nil)
		request.Header.Set("Authorization", authHeader)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
	})
}

func TestPlaybackEndpointReturnsDecisionPayloadForFallbackAndUnplayable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("fallback stays 200 with payload", func(t *testing.T) {
		router, authSvc, mediaItemID := newPlaybackDecisionRouter(t, true)
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-items/%d/playback?client_profile=web", mediaItemID), nil)
		request.Header.Set("Authorization", createAuthHeader(t, ctx, authSvc))
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		var body struct {
			Data playback.PlaybackSource `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body.Data.Decision.Kind != "fallback" {
			t.Fatalf("decision.kind = %q, want fallback", body.Data.Decision.Kind)
		}
	})

	t.Run("unplayable stays 200 with payload", func(t *testing.T) {
		router, authSvc, mediaItemID := newPlaybackDecisionRouter(t, false)
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-items/%d/playback?client_profile=web", mediaItemID), nil)
		request.Header.Set("Authorization", createAuthHeader(t, ctx, authSvc))
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		var body struct {
			Data playback.PlaybackSource `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body.Data.Decision.Kind != "unplayable" {
			t.Fatalf("decision.kind = %q, want unplayable", body.Data.Decision.Kind)
		}
	})
}

func TestAdminSourceAndLibraryEndpointsRequireAuth(t *testing.T) {
	router, _, authSvc, librarySvc, storageRoot := newDeleteTestRouter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mediaRoot := filepath.Join(storageRoot, "admin-media")
	if err := os.MkdirAll(mediaRoot, 0o755); err != nil {
		t.Fatalf("create media root: %v", err)
	}

	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{
		Provider: "local",
		Name:     "Existing Source",
		RootPath: mediaRoot,
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}

	createdLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{
		Name:          "Existing Library",
		Type:          "movies",
		MediaSourceID: source.ID,
		RootPath:      mediaRoot,
	})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "create media source", method: http.MethodPost, path: "/api/v1/media-sources", body: fmt.Sprintf(`{"provider":"local","name":"New Source","root_path":%q}`, filepath.Join(storageRoot, "new-source"))},
		{name: "update media source", method: http.MethodPatch, path: fmt.Sprintf("/api/v1/media-sources/%d", source.ID), body: fmt.Sprintf(`{"name":"Updated Source","root_path":%q}`, mediaRoot)},
		{name: "list media sources", method: http.MethodGet, path: "/api/v1/media-sources"},
		{name: "delete media source", method: http.MethodDelete, path: fmt.Sprintf("/api/v1/media-sources/%d", source.ID)},
		{name: "create library", method: http.MethodPost, path: "/api/v1/libraries", body: fmt.Sprintf(`{"name":"New Library","type":"movies","media_source_id":%d,"root_path":%q}`, source.ID, mediaRoot)},
		{name: "list libraries", method: http.MethodGet, path: "/api/v1/libraries"},
		{name: "delete library", method: http.MethodDelete, path: fmt.Sprintf("/api/v1/libraries/%d", createdLibrary.ID)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}

			router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("expected unauthorized status, got %d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}

	authHeader := createAuthHeader(t, ctx, authSvc)

	t.Run("authenticated source and library setup still succeeds", func(t *testing.T) {
		createSourceRecorder := httptest.NewRecorder()
		createSourceRequest := httptest.NewRequest(http.MethodPost, "/api/v1/media-sources", strings.NewReader(fmt.Sprintf(`{"provider":"local","name":"Authed Source","root_path":%q}`, mediaRoot)))
		createSourceRequest.Header.Set("Authorization", authHeader)
		createSourceRequest.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(createSourceRecorder, createSourceRequest)
		if createSourceRecorder.Code != http.StatusCreated {
			t.Fatalf("create source status: %d body=%s", createSourceRecorder.Code, createSourceRecorder.Body.String())
		}

		var createSourceBody struct {
			Data library.MediaSourceView `json:"data"`
		}
		if err := json.Unmarshal(createSourceRecorder.Body.Bytes(), &createSourceBody); err != nil {
			t.Fatalf("decode create source response: %v", err)
		}

		createLibraryRecorder := httptest.NewRecorder()
		createLibraryRequest := httptest.NewRequest(http.MethodPost, "/api/v1/libraries", strings.NewReader(fmt.Sprintf(`{"name":"Authed Library","type":"movies","media_source_id":%d,"root_path":%q}`, createSourceBody.Data.ID, mediaRoot)))
		createLibraryRequest.Header.Set("Authorization", authHeader)
		createLibraryRequest.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(createLibraryRecorder, createLibraryRequest)
		if createLibraryRecorder.Code != http.StatusCreated {
			t.Fatalf("create library status: %d body=%s", createLibraryRecorder.Code, createLibraryRecorder.Body.String())
		}

		listLibrariesRecorder := httptest.NewRecorder()
		listLibrariesRequest := httptest.NewRequest(http.MethodGet, "/api/v1/libraries", nil)
		listLibrariesRequest.Header.Set("Authorization", authHeader)
		router.ServeHTTP(listLibrariesRecorder, listLibrariesRequest)
		if listLibrariesRecorder.Code != http.StatusOK {
			t.Fatalf("list libraries status: %d body=%s", listLibrariesRecorder.Code, listLibrariesRecorder.Body.String())
		}

		var listLibrariesBody struct {
			Data []library.LibraryDetail `json:"data"`
		}
		if err := json.Unmarshal(listLibrariesRecorder.Body.Bytes(), &listLibrariesBody); err != nil {
			t.Fatalf("decode list libraries response: %v", err)
		}
		if len(listLibrariesBody.Data) == 0 {
			t.Fatal("expected authenticated library list to include created library")
		}
	})
}

func TestAdminScanAndJobsEndpointsRequireAuth(t *testing.T) {
	router, db, authSvc, librarySvc, storageRoot := newDeleteTestRouter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mediaRoot := filepath.Join(storageRoot, "jobs-media")
	if err := os.MkdirAll(mediaRoot, 0o755); err != nil {
		t.Fatalf("create media root: %v", err)
	}

	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{
		Provider: "local",
		Name:     "Jobs Source",
		RootPath: mediaRoot,
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}

	createdLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{
		Name:          "Jobs Library",
		Type:          "movies",
		MediaSourceID: source.ID,
		RootPath:      mediaRoot,
	})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}

	failedJob := database.Job{
		Kind:         "sync_library",
		Status:       jobs.StatusFailed,
		PayloadJSON:  `{"library_id":1}`,
		ErrorMessage: "boom",
		AvailableAt:  time.Now().UTC(),
	}
	if err := db.WithContext(ctx).Create(&failedJob).Error; err != nil {
		t.Fatalf("create failed job: %v", err)
	}

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "queue library scan", method: http.MethodPost, path: fmt.Sprintf("/api/v1/libraries/%d/scan", createdLibrary.ID)},
		{name: "list jobs", method: http.MethodGet, path: "/api/v1/jobs"},
		{name: "retry job", method: http.MethodPost, path: fmt.Sprintf("/api/v1/jobs/%d/retry", failedJob.ID)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, nil)
			router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("expected unauthorized status, got %d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}

	authHeader := createAuthHeader(t, ctx, authSvc)
	adminHeader := createAdminAuthHeader(t, ctx, db, authSvc)

	t.Run("authenticated scan and jobs operations still succeed", func(t *testing.T) {
		queueRecorder := httptest.NewRecorder()
		queueRequest := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/libraries/%d/scan", createdLibrary.ID), nil)
		queueRequest.Header.Set("Authorization", authHeader)
		router.ServeHTTP(queueRecorder, queueRequest)
		if queueRecorder.Code != http.StatusAccepted {
			t.Fatalf("queue scan status: %d body=%s", queueRecorder.Code, queueRecorder.Body.String())
		}

		listRecorder := httptest.NewRecorder()
		listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?status=failed", nil)
		listRequest.Header.Set("Authorization", adminHeader)
		router.ServeHTTP(listRecorder, listRequest)
		if listRecorder.Code != http.StatusOK {
			t.Fatalf("list jobs status: %d body=%s", listRecorder.Code, listRecorder.Body.String())
		}

		var listBody struct {
			Data []database.Job `json:"data"`
		}
		if err := json.Unmarshal(listRecorder.Body.Bytes(), &listBody); err != nil {
			t.Fatalf("decode jobs list response: %v", err)
		}
		if len(listBody.Data) != 1 || listBody.Data[0].ID != failedJob.ID {
			t.Fatalf("unexpected filtered jobs payload: %#v", listBody.Data)
		}

		retryRecorder := httptest.NewRecorder()
		retryRequest := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/jobs/%d/retry", failedJob.ID), nil)
		retryRequest.Header.Set("Authorization", adminHeader)
		router.ServeHTTP(retryRecorder, retryRequest)
		if retryRecorder.Code != http.StatusAccepted {
			t.Fatalf("retry job status: %d body=%s", retryRecorder.Code, retryRecorder.Body.String())
		}
	})
}

func TestDeleteLibraryEndpoint(t *testing.T) {
	router, db, authSvc, librarySvc, storageRoot := newDeleteTestRouter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mediaRoot := filepath.Join(storageRoot, "media-root")
	moviesDir := filepath.Join(mediaRoot, "Movies")
	if err := os.MkdirAll(moviesDir, 0o755); err != nil {
		t.Fatalf("create movies dir: %v", err)
	}

	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{
		Provider: "local",
		Name:     "Local Media",
		RootPath: mediaRoot,
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}

	record, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{
		Name:          "Movies",
		Type:          "movies",
		MediaSourceID: source.ID,
		RootPath:      moviesDir,
	})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}

	_, itemID, fileID := seedLibraryData(t, ctx, db, authSvc, record.ID, moviesDir, "MovieA")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/libraries/%d", record.ID), nil)
	request.Header.Set("Authorization", createAuthHeader(t, ctx, authSvc))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("delete library status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	assertDeletedCount(t, ctx, db, &database.Library{}, "id = ?", record.ID, 0)
	assertDeletedCount(t, ctx, db, &database.MediaItem{}, "id = ?", itemID, 0)
	assertDeletedCount(t, ctx, db, &database.MediaFile{}, "id = ?", fileID, 0)
	assertDeletedCount(t, ctx, db, &database.PlaybackProgress{}, "media_item_id = ?", itemID, 0)
	assertDeletedCount(t, ctx, db, &database.MediaSource{}, "id = ?", source.ID, 1)
}

func TestDeleteMediaSourceEndpointCascadesLibraries(t *testing.T) {
	router, db, authSvc, librarySvc, storageRoot := newDeleteTestRouter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mediaRoot := filepath.Join(storageRoot, "media-root")
	moviesDir := filepath.Join(mediaRoot, "Movies")
	showsDir := filepath.Join(mediaRoot, "Shows")
	if err := os.MkdirAll(moviesDir, 0o755); err != nil {
		t.Fatalf("create movies dir: %v", err)
	}
	if err := os.MkdirAll(showsDir, 0o755); err != nil {
		t.Fatalf("create shows dir: %v", err)
	}

	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{
		Provider: "local",
		Name:     "Local Media",
		RootPath: mediaRoot,
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}

	movieLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{
		Name:          "Movies",
		Type:          "movies",
		MediaSourceID: source.ID,
		RootPath:      moviesDir,
	})
	if err != nil {
		t.Fatalf("create movie library: %v", err)
	}
	showLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{
		Name:          "Shows",
		Type:          "shows",
		MediaSourceID: source.ID,
		RootPath:      showsDir,
	})
	if err != nil {
		t.Fatalf("create show library: %v", err)
	}

	_, movieItemID, movieFileID := seedLibraryData(t, ctx, db, authSvc, movieLibrary.ID, moviesDir, "MovieA")
	_, showItemID, showFileID := seedLibraryData(t, ctx, db, authSvc, showLibrary.ID, showsDir, "ShowA")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/media-sources/%d", source.ID), nil)
	request.Header.Set("Authorization", createAuthHeader(t, ctx, authSvc))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("delete source status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	assertDeletedCount(t, ctx, db, &database.MediaSource{}, "id = ?", source.ID, 0)
	assertDeletedCount(t, ctx, db, &database.Library{}, "id IN ?", []uint{movieLibrary.ID, showLibrary.ID}, 0)
	assertDeletedCount(t, ctx, db, &database.MediaItem{}, "id IN ?", []uint{movieItemID, showItemID}, 0)
	assertDeletedCount(t, ctx, db, &database.MediaFile{}, "id IN ?", []uint{movieFileID, showFileID}, 0)
	assertDeletedCount(t, ctx, db, &database.PlaybackProgress{}, "media_item_id IN ?", []uint{movieItemID, showItemID}, 0)
}

func TestBrowseStorageProviderEndpointRequiresAuthAndListsDirectories(t *testing.T) {
	storageRoot := filepath.Join(t.TempDir(), "storage-root")
	mediaRoot := filepath.Join(storageRoot, "Media")
	moviesDir := filepath.Join(mediaRoot, "Movies")
	showsDir := filepath.Join(mediaRoot, "Shows")
	if err := os.MkdirAll(moviesDir, 0o755); err != nil {
		t.Fatalf("create movies dir: %v", err)
	}
	if err := os.MkdirAll(showsDir, 0o755); err != nil {
		t.Fatalf("create shows dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mediaRoot, "ignore.txt"), []byte("demo"), 0o644); err != nil {
		t.Fatalf("write ignore file: %v", err)
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
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settings.NewService(db, cfg.Metadata)), settings.NewService(db, cfg.Metadata))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/storage/providers/local/browse?path="+url.QueryEscape(mediaRoot), nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	authHeader := createAuthHeader(t, ctx, authSvc)

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/storage/providers/local/browse?path="+url.QueryEscape(mediaRoot), nil)
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("browse local status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var body struct {
		Data library.BrowseResult `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode browse response: %v", err)
	}
	if body.Data.CurrentPath != mediaRoot {
		t.Fatalf("unexpected current path: %q", body.Data.CurrentPath)
	}
	if body.Data.ParentPath != storageRoot {
		t.Fatalf("unexpected parent path: %q", body.Data.ParentPath)
	}
	if len(body.Data.Items) != 2 || body.Data.Items[0].Name != "Movies" || body.Data.Items[1].Name != "Shows" {
		t.Fatalf("unexpected browse items: %#v", body.Data.Items)
	}
	if !body.Data.Items[0].IsDir || !body.Data.Items[1].IsDir {
		t.Fatalf("expected directories only, got %#v", body.Data.Items)
	}
}

func TestBrowseMediaSourceEndpointRestrictsToSourceRoot(t *testing.T) {
	openList := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var body struct {
			Path string `json:"path"`
		}
		_ = json.NewDecoder(req.Body).Decode(&body)

		switch req.URL.Path {
		case "/api/fs/get":
			name := filepath.Base(body.Path)
			if body.Path == "/" {
				name = "root"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    http.StatusOK,
				"message": "success",
				"data": map[string]any{
					"name":   name,
					"is_dir": true,
					"size":   0,
				},
			})
		case "/api/fs/list":
			content := []map[string]any{}
			switch body.Path {
			case "/library":
				content = []map[string]any{{"name": "Movies", "is_dir": true, "size": 0}, {"name": "notes.txt", "is_dir": false, "size": 10}}
			case "/library/Movies":
				content = []map[string]any{{"name": "Kids", "is_dir": true, "size": 0}}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    http.StatusOK,
				"message": "success",
				"data":    map[string]any{"content": content},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer openList.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		OpenList: config.OpenListConfig{BaseURL: openList.URL, RootPath: "/library"},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settings.NewService(db, cfg.Metadata)), settings.NewService(db, cfg.Metadata))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "openlist", Name: "OpenList", RootPath: "/library"})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	authHeader := createAuthHeader(t, ctx, authSvc)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-sources/%d/browse?path=%s", source.ID, url.QueryEscape("/library/Movies")), nil)
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("browse source status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var body struct {
		Data library.BrowseResult `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode browse source response: %v", err)
	}
	if body.Data.RootPath != "/library" || body.Data.CurrentPath != "/library/Movies" || body.Data.ParentPath != "/library" {
		t.Fatalf("unexpected browse source paths: %#v", body.Data)
	}
	if len(body.Data.Items) != 1 || body.Data.Items[0].Path != "/library/Movies/Kids" {
		t.Fatalf("unexpected browse source items: %#v", body.Data.Items)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-sources/%d/browse?path=%s", source.ID, url.QueryEscape("/outside")), nil)
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for outside path, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestOpenListSourcesKeepIndependentConfigs(t *testing.T) {
	newOpenListServer := func(label string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if req.URL.Path == "/api/auth/login" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code":    http.StatusOK,
					"message": "success",
					"data":    map[string]any{"token": label + "-token"},
				})
				return
			}

			if got := req.Header.Get("Authorization"); got != label+"-token" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code":    http.StatusUnauthorized,
					"message": "unauthorized",
					"data":    nil,
				})
				return
			}

			var body struct {
				Path string `json:"path"`
			}
			_ = json.NewDecoder(req.Body).Decode(&body)

			switch req.URL.Path {
			case "/api/fs/get":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code":    http.StatusOK,
					"message": "success",
					"data":    map[string]any{"name": filepath.Base(body.Path), "is_dir": true, "size": 0},
				})
			case "/api/fs/list":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code":    http.StatusOK,
					"message": "success",
					"data": map[string]any{
						"content": []map[string]any{{"name": label + "-Movies", "is_dir": true, "size": 0}},
					},
				})
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
	}

	openListA := newOpenListServer("alpha")
	defer openListA.Close()
	openListB := newOpenListServer("beta")
	defer openListB.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{Database: config.DatabaseConfig{Driver: "sqlite"}}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settings.NewService(db, cfg.Metadata)), settings.NewService(db, cfg.Metadata))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	authHeader := createAuthHeader(t, ctx, authSvc)

	createSource := func(name, baseURL string) library.MediaSourceView {
		t.Helper()
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/media-sources", strings.NewReader(fmt.Sprintf(`{"provider":"openlist","name":"%s","root_path":"/media","config":{"openlist":{"base_url":"%s","username":"admin","password":"secret"}}}`, name, baseURL)))
		request.Header.Set("Authorization", authHeader)
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusCreated {
			t.Fatalf("create source status: %d body=%s", recorder.Code, recorder.Body.String())
		}
		var body struct {
			Data library.MediaSourceView `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode create source response: %v", err)
		}
		if body.Data.Config.OpenList == nil || !body.Data.Config.OpenList.HasPassword || body.Data.Config.OpenList.BaseURL != baseURL {
			t.Fatalf("unexpected source config view: %#v", body.Data)
		}
		return body.Data
	}

	sourceA := createSource("Alpha", openListA.URL)
	sourceB := createSource("Beta", openListB.URL)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/storage/openlist/browse", strings.NewReader(fmt.Sprintf(`{"path":"/media","config":{"base_url":"%s","username":"admin","password":"secret"}}`, openListA.URL)))
	request.Header.Set("Authorization", authHeader)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("temporary browse status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	assertBrowse := func(sourceID uint, wantPath string) {
		t.Helper()
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-sources/%d/browse?path=%s", sourceID, url.QueryEscape("/media")), nil)
		request.Header.Set("Authorization", authHeader)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("browse source status: %d body=%s", recorder.Code, recorder.Body.String())
		}
		var body struct {
			Data library.BrowseResult `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode browse response: %v", err)
		}
		if len(body.Data.Items) != 1 || body.Data.Items[0].Path != wantPath {
			t.Fatalf("unexpected browse items: %#v", body.Data.Items)
		}
	}

	assertBrowse(sourceA.ID, "/media/alpha-Movies")
	assertBrowse(sourceB.ID, "/media/beta-Movies")
}

func TestTemporaryOpenListTestEndpoint(t *testing.T) {
	openList := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if req.URL.Path == "/api/auth/login" {
			var body struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			_ = json.NewDecoder(req.Body).Decode(&body)
			if body.Username != "admin" || body.Password != "secret" {
				_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusUnauthorized, "message": "bad credentials"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusOK, "message": "success", "data": map[string]any{"token": "test-token"}})
			return
		}

		if req.Header.Get("Authorization") != "test-token" {
			_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusUnauthorized, "message": "unauthorized"})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusOK, "message": "success", "data": map[string]any{"name": "root", "is_dir": true, "size": 0}})
	}))
	defer openList.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{Database: config.DatabaseConfig{Driver: "sqlite"}}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settings.NewService(db, cfg.Metadata)), settings.NewService(db, cfg.Metadata))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	authHeader := createAuthHeader(t, ctx, authSvc)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/storage/openlist/test", strings.NewReader(fmt.Sprintf(`{"config":{"base_url":"%s","username":"admin","password":"secret"}}`, openList.URL)))
	request.Header.Set("Authorization", authHeader)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("test openlist status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var body struct {
		Data library.OpenListTestResult `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode test response: %v", err)
	}
	if body.Data.Status != "ok" || body.Data.Provider != "openlist" || body.Data.RootPath != "/" {
		t.Fatalf("unexpected test response: %#v", body.Data)
	}
}

func TestUpdateMediaSourcePreservesOpenListPasswordWhenBlank(t *testing.T) {
	openList := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if req.URL.Path == "/api/auth/login" {
			var body struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			_ = json.NewDecoder(req.Body).Decode(&body)
			if body.Username != "admin" || body.Password != "secret" {
				_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusUnauthorized, "message": "bad credentials"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusOK, "message": "success", "data": map[string]any{"token": "demo-token"}})
			return
		}

		if req.Header.Get("Authorization") != "demo-token" {
			_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusUnauthorized, "message": "unauthorized"})
			return
		}

		var body struct {
			Path string `json:"path"`
		}
		_ = json.NewDecoder(req.Body).Decode(&body)
		switch req.URL.Path {
		case "/api/fs/get":
			_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusOK, "message": "success", "data": map[string]any{"name": filepath.Base(body.Path), "is_dir": true, "size": 0}})
		case "/api/fs/list":
			_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusOK, "message": "success", "data": map[string]any{"content": []map[string]any{{"name": "Movies", "is_dir": true, "size": 0}}}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer openList.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{Database: config.DatabaseConfig{Driver: "sqlite"}}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settings.NewService(db, cfg.Metadata)), settings.NewService(db, cfg.Metadata))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{
		Provider: "openlist",
		Name:     "Original",
		RootPath: "/media",
		Config: &providers.SourceConfig{OpenList: &providers.OpenListSourceConfig{
			BaseURL:  openList.URL,
			Username: "admin",
			Password: "secret",
		}},
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/media-sources/%d", source.ID), strings.NewReader(fmt.Sprintf(`{"name":"Updated","root_path":"/media/new","config":{"openlist":{"base_url":"%s","username":"admin","password":""}}}`, openList.URL)))
	request.Header.Set("Authorization", createAuthHeader(t, ctx, authSvc))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("update source status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var body struct {
		Data library.MediaSourceView `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if body.Data.Name != "Updated" || body.Data.RootPath != "/media/new" || body.Data.Config.OpenList == nil || !body.Data.Config.OpenList.HasPassword {
		t.Fatalf("unexpected updated source view: %#v", body.Data)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-sources/%d/browse?path=%s", source.ID, url.QueryEscape("/media/new")), nil)
	request.Header.Set("Authorization", createAuthHeader(t, ctx, authSvc))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("browse updated source status: %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestLocalPlaybackStreamEndpoint(t *testing.T) {
	mediaRoot := filepath.Join(t.TempDir(), "demo-media")
	movieDir := filepath.Join(mediaRoot, "Movies")
	if err := os.MkdirAll(movieDir, 0o755); err != nil {
		t.Fatalf("create movie dir: %v", err)
	}
	moviePath := filepath.Join(movieDir, "MovieA.2024.mp4")
	movieBytes := []byte("local-stream-payload")
	if err := os.WriteFile(moviePath, movieBytes, 0o644); err != nil {
		t.Fatalf("write movie file: %v", err)
	}

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		Storage:  config.StorageConfig{Provider: "local"},
		Local:    config.LocalStorageConfig{RootPath: mediaRoot},
		FFprobe:  config.FFprobeConfig{Enabled: false},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	playbackSvc := playback.NewService(db, registry)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playbackSvc, progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settings.NewService(db, cfg.Metadata)), settings.NewService(db, cfg.Metadata))
	server := httptest.NewServer(router)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Local Media", RootPath: mediaRoot})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	createdLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: movieDir})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	mediaItem := database.MediaItem{LibraryID: createdLibrary.ID, Type: "movie", Title: "MovieA", SourcePath: moviePath, MatchStatus: "matched", Status: "ready"}
	if err := db.WithContext(ctx).Create(&mediaItem).Error; err != nil {
		t.Fatalf("create media item: %v", err)
	}
	mediaFile := database.MediaFile{LibraryID: createdLibrary.ID, MediaItemID: &mediaItem.ID, StoragePath: moviePath, Container: "mp4", ProbeStatus: probe.StatusReady, VideoCodec: "h264"}
	if err := db.WithContext(ctx).Create(&mediaFile).Error; err != nil {
		t.Fatalf("create media file: %v", err)
	}

	registeredUser, err := authSvc.Register(ctx, "alice", "password123")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	if registeredUser.ID == 0 {
		t.Fatal("expected created user id")
	}
	loginResult, err := authSvc.Login(ctx, "alice", "password123")
	if err != nil {
		t.Fatalf("login user: %v", err)
	}
	authHeader := fmt.Sprintf("Bearer %s", loginResult.Token)

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/media-items/%d/playback?client_profile=web", server.URL, mediaItem.ID), nil)
	if err != nil {
		t.Fatalf("build playback request: %v", err)
	}
	request.Header.Set("Authorization", authHeader)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request playback source: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("playback source status: %d body=%s", response.StatusCode, string(body))
	}

	var playbackBody struct {
		Data playback.PlaybackSource `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&playbackBody); err != nil {
		t.Fatalf("decode playback source: %v", err)
	}
	if !strings.HasPrefix(playbackBody.Data.URL, fmt.Sprintf("%s/api/v1/media-files/%d/stream?", server.URL, mediaFile.ID)) {
		t.Fatalf("unexpected playback url: %s", playbackBody.Data.URL)
	}
	if !strings.Contains(playbackBody.Data.URL, "access_token="+loginResult.Token) {
		t.Fatalf("expected playback url to embed access token, got %s", playbackBody.Data.URL)
	}

	streamResponse, err := http.Get(playbackBody.Data.URL)
	if err != nil {
		t.Fatalf("request local stream: %v", err)
	}
	defer streamResponse.Body.Close()
	if streamResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(streamResponse.Body)
		t.Fatalf("stream status: %d body=%s", streamResponse.StatusCode, string(body))
	}
	streamedBytes, err := io.ReadAll(streamResponse.Body)
	if err != nil {
		t.Fatalf("read stream body: %v", err)
	}
	if string(streamedBytes) != string(movieBytes) {
		t.Fatalf("unexpected stream payload: got %q want %q", string(streamedBytes), string(movieBytes))
	}
}

func TestOpenListPlaybackStreamEndpoint(t *testing.T) {
	mediaPayload := []byte("openlist-stream-payload")
	var openList *httptest.Server
	openList = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if req.URL.Path == "/raw/MovieA.2024.mp4" {
			w.Header().Set("Content-Type", "video/mp4")
			w.Header().Set("Accept-Ranges", "bytes")
			_, _ = w.Write(mediaPayload)
			return
		}

		var body struct {
			Path string `json:"path"`
		}
		_ = json.NewDecoder(req.Body).Decode(&body)

		switch req.URL.Path {
		case "/api/fs/get":
			isDir := !strings.HasSuffix(body.Path, ".mp4")
			data := map[string]any{"name": "movies", "is_dir": isDir, "size": len(mediaPayload)}
			if !isDir {
				data["raw_url"] = openList.URL + "/raw/MovieA.2024.mp4"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusOK, "message": "success", "data": data})
		case "/api/fs/list":
			content := []map[string]any{}
			if body.Path == "/movies" {
				content = []map[string]any{{"name": "MovieA.2024.mp4", "is_dir": false, "size": len(mediaPayload)}}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusOK, "message": "success", "data": map[string]any{"content": content}})
		case "/api/fs/link":
			_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusOK, "message": "success", "data": map[string]any{"url": openList.URL + "/raw/MovieA.2024.mp4"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer openList.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		Storage:  config.StorageConfig{Provider: "openlist"},
		OpenList: config.OpenListConfig{BaseURL: openList.URL, RootPath: "/movies"},
		FFprobe:  config.FFprobeConfig{Enabled: false},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	playbackSvc := playback.NewService(db, registry)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playbackSvc, progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settings.NewService(db, cfg.Metadata)), settings.NewService(db, cfg.Metadata))
	server := httptest.NewServer(router)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "openlist", Name: "OpenList Media", RootPath: "/movies"})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	createdLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: "/movies"})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	mediaItem := database.MediaItem{LibraryID: createdLibrary.ID, Type: "movie", Title: "MovieA", SourcePath: "/movies/MovieA.2024.mp4", MatchStatus: "matched", Status: "ready"}
	if err := db.WithContext(ctx).Create(&mediaItem).Error; err != nil {
		t.Fatalf("create media item: %v", err)
	}
	mediaFile := database.MediaFile{LibraryID: createdLibrary.ID, MediaItemID: &mediaItem.ID, StoragePath: mediaItem.SourcePath, Container: "mp4", ProbeStatus: probe.StatusReady, VideoCodec: "h264"}
	if err := db.WithContext(ctx).Create(&mediaFile).Error; err != nil {
		t.Fatalf("create media file: %v", err)
	}

	registeredUser, err := authSvc.Register(ctx, "bob", "password123")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	if registeredUser.ID == 0 {
		t.Fatal("expected created user id")
	}
	loginResult, err := authSvc.Login(ctx, "bob", "password123")
	if err != nil {
		t.Fatalf("login user: %v", err)
	}
	authHeader := fmt.Sprintf("Bearer %s", loginResult.Token)

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/media-items/%d/playback?client_profile=web", server.URL, mediaItem.ID), nil)
	if err != nil {
		t.Fatalf("build playback request: %v", err)
	}
	request.Header.Set("Authorization", authHeader)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request playback source: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("playback source status: %d body=%s", response.StatusCode, string(body))
	}

	var playbackBody struct {
		Data playback.PlaybackSource `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&playbackBody); err != nil {
		t.Fatalf("decode playback source: %v", err)
	}
	if !strings.HasPrefix(playbackBody.Data.URL, fmt.Sprintf("%s/api/v1/media-files/%d/stream?", server.URL, mediaFile.ID)) {
		t.Fatalf("unexpected playback url: %s", playbackBody.Data.URL)
	}

	streamResponse, err := http.Get(playbackBody.Data.URL)
	if err != nil {
		t.Fatalf("request proxied openlist stream: %v", err)
	}
	defer streamResponse.Body.Close()
	if streamResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(streamResponse.Body)
		t.Fatalf("stream status: %d body=%s", streamResponse.StatusCode, string(body))
	}
	if streamResponse.Header.Get("Content-Type") != "video/mp4" {
		t.Fatalf("unexpected proxied content-type: %s", streamResponse.Header.Get("Content-Type"))
	}
	streamedBytes, err := io.ReadAll(streamResponse.Body)
	if err != nil {
		t.Fatalf("read stream body: %v", err)
	}
	if string(streamedBytes) != string(mediaPayload) {
		t.Fatalf("unexpected proxied stream payload: got %q want %q", string(streamedBytes), string(mediaPayload))
	}
}

func TestLocalPlaybackReturnsHLSPlaylist(t *testing.T) {
	mediaRoot := filepath.Join(t.TempDir(), "demo-media")
	movieDir := filepath.Join(mediaRoot, "Movies")
	if err := os.MkdirAll(movieDir, 0o755); err != nil {
		t.Fatalf("create movie dir: %v", err)
	}
	moviePath := filepath.Join(movieDir, "MovieA.2024.mkv")
	if err := os.WriteFile(moviePath, []byte("local-hls-payload"), 0o644); err != nil {
		t.Fatalf("write movie file: %v", err)
	}

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	hlsRoot := filepath.Join(t.TempDir(), "hls")
	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		Storage:  config.StorageConfig{Provider: "local"},
		Local:    config.LocalStorageConfig{RootPath: mediaRoot},
		FFprobe:  config.FFprobeConfig{Enabled: false},
		FFmpeg:   config.FFmpegConfig{Enabled: true, Path: writeRouterFakeFFmpeg(t), Timeout: 2 * time.Second},
		HLS:      config.HLSConfig{Enabled: true, RootPath: hlsRoot, SegmentDuration: 6, CleanupAge: time.Hour},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settings.NewService(db, cfg.Metadata)), settings.NewService(db, cfg.Metadata))
	server := httptest.NewServer(router)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Local Media", RootPath: mediaRoot})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	createdLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: movieDir})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	mediaItem := database.MediaItem{LibraryID: createdLibrary.ID, Type: "movie", Title: "MovieA", SourcePath: moviePath, MatchStatus: "matched", Status: "ready"}
	if err := db.WithContext(ctx).Create(&mediaItem).Error; err != nil {
		t.Fatalf("create media item: %v", err)
	}
	mediaFile := database.MediaFile{LibraryID: createdLibrary.ID, MediaItemID: &mediaItem.ID, StoragePath: moviePath, Container: "mkv", ProbeStatus: probe.StatusReady, VideoCodec: "h264"}
	if err := db.WithContext(ctx).Create(&mediaFile).Error; err != nil {
		t.Fatalf("create media file: %v", err)
	}

	loginResult := registerAndLoginRouterUser(t, ctx, authSvc, "hls-local")
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/media-items/%d/playback?client_profile=web", server.URL, mediaItem.ID), nil)
	if err != nil {
		t.Fatalf("build playback request: %v", err)
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", loginResult.Token))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request playback source: %v", err)
	}
	defer response.Body.Close()

	var playbackBody struct {
		Data playback.PlaybackSource `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&playbackBody); err != nil {
		t.Fatalf("decode playback source: %v", err)
	}
	if playbackBody.Data.Container != "m3u8" {
		t.Fatalf("expected hls container, got %s", playbackBody.Data.Container)
	}
	if !strings.HasPrefix(playbackBody.Data.URL, fmt.Sprintf("%s/api/v1/media-files/%d/hls/index.m3u8?", server.URL, mediaFile.ID)) {
		t.Fatalf("unexpected hls playback url: %s", playbackBody.Data.URL)
	}

	playlistResponse, err := http.Get(playbackBody.Data.URL)
	if err != nil {
		t.Fatalf("request hls playlist: %v", err)
	}
	defer playlistResponse.Body.Close()
	if playlistResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(playlistResponse.Body)
		t.Fatalf("playlist status: %d body=%s", playlistResponse.StatusCode, string(body))
	}
	if contentType := playlistResponse.Header.Get("Content-Type"); !strings.Contains(contentType, "application/vnd.apple.mpegurl") {
		t.Fatalf("unexpected playlist content-type: %s", contentType)
	}
	playlistBody, err := io.ReadAll(playlistResponse.Body)
	if err != nil {
		t.Fatalf("read playlist body: %v", err)
	}
	if !strings.Contains(string(playlistBody), "segment_000.ts") {
		t.Fatalf("unexpected playlist body: %s", string(playlistBody))
	}

	segmentResponse, err := http.Get(fmt.Sprintf("%s/api/v1/media-files/%d/hls/segment_000.ts?access_token=%s", server.URL, mediaFile.ID, loginResult.Token))
	if err != nil {
		t.Fatalf("request hls segment: %v", err)
	}
	defer segmentResponse.Body.Close()
	if segmentResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(segmentResponse.Body)
		t.Fatalf("segment status: %d body=%s", segmentResponse.StatusCode, string(body))
	}
	segmentBody, err := io.ReadAll(segmentResponse.Body)
	if err != nil {
		t.Fatalf("read segment body: %v", err)
	}
	if string(segmentBody) != "fake-hls-segment" {
		t.Fatalf("unexpected segment payload: %q", string(segmentBody))
	}
}

func TestOpenListPlaybackReturnsHLSPlaylist(t *testing.T) {
	mediaPayload := []byte("openlist-hls-source")
	var openList *httptest.Server
	openList = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if req.URL.Path == "/raw/MovieA.2024.mkv" {
			w.Header().Set("Content-Type", "video/mp4")
			_, _ = w.Write(mediaPayload)
			return
		}

		var body struct {
			Path string `json:"path"`
		}
		_ = json.NewDecoder(req.Body).Decode(&body)

		switch req.URL.Path {
		case "/api/fs/get":
			isDir := !strings.HasSuffix(body.Path, ".mkv")
			data := map[string]any{"name": "movies", "is_dir": isDir, "size": len(mediaPayload)}
			if !isDir {
				data["raw_url"] = openList.URL + "/raw/MovieA.2024.mkv"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusOK, "message": "success", "data": data})
		case "/api/fs/list":
			content := []map[string]any{}
			if body.Path == "/movies" {
				content = []map[string]any{{"name": "MovieA.2024.mkv", "is_dir": false, "size": len(mediaPayload)}}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusOK, "message": "success", "data": map[string]any{"content": content}})
		case "/api/fs/link":
			_ = json.NewEncoder(w).Encode(map[string]any{"code": http.StatusOK, "message": "success", "data": map[string]any{"url": openList.URL + "/raw/MovieA.2024.mkv"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer openList.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	hlsRoot := filepath.Join(t.TempDir(), "hls")
	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		Storage:  config.StorageConfig{Provider: "openlist"},
		OpenList: config.OpenListConfig{BaseURL: openList.URL, RootPath: "/movies"},
		FFprobe:  config.FFprobeConfig{Enabled: false},
		FFmpeg:   config.FFmpegConfig{Enabled: true, Path: writeRouterFakeFFmpeg(t), Timeout: 2 * time.Second},
		HLS:      config.HLSConfig{Enabled: true, RootPath: hlsRoot, SegmentDuration: 6, CleanupAge: time.Hour},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settings.NewService(db, cfg.Metadata)), settings.NewService(db, cfg.Metadata))
	server := httptest.NewServer(router)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "openlist", Name: "OpenList Media", RootPath: "/movies"})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	createdLibrary, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: "/movies"})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	mediaItem := database.MediaItem{LibraryID: createdLibrary.ID, Type: "movie", Title: "MovieA", SourcePath: "/movies/MovieA.2024.mkv", MatchStatus: "matched", Status: "ready"}
	if err := db.WithContext(ctx).Create(&mediaItem).Error; err != nil {
		t.Fatalf("create media item: %v", err)
	}
	mediaFile := database.MediaFile{LibraryID: createdLibrary.ID, MediaItemID: &mediaItem.ID, StoragePath: mediaItem.SourcePath, Container: "mkv", ProbeStatus: probe.StatusReady, VideoCodec: "h264"}
	if err := db.WithContext(ctx).Create(&mediaFile).Error; err != nil {
		t.Fatalf("create media file: %v", err)
	}

	loginResult := registerAndLoginRouterUser(t, ctx, authSvc, "hls-openlist")
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/media-items/%d/playback?client_profile=web", server.URL, mediaItem.ID), nil)
	if err != nil {
		t.Fatalf("build playback request: %v", err)
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", loginResult.Token))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request playback source: %v", err)
	}
	defer response.Body.Close()

	var playbackBody struct {
		Data playback.PlaybackSource `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&playbackBody); err != nil {
		t.Fatalf("decode playback source: %v", err)
	}
	if !strings.HasPrefix(playbackBody.Data.URL, fmt.Sprintf("%s/api/v1/media-files/%d/hls/index.m3u8?", server.URL, mediaFile.ID)) {
		t.Fatalf("unexpected openlist hls playback url: %s", playbackBody.Data.URL)
	}

	playlistResponse, err := http.Get(playbackBody.Data.URL)
	if err != nil {
		t.Fatalf("request hls playlist: %v", err)
	}
	defer playlistResponse.Body.Close()
	if playlistResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(playlistResponse.Body)
		t.Fatalf("playlist status: %d body=%s", playlistResponse.StatusCode, string(body))
	}
	playlistBody, err := io.ReadAll(playlistResponse.Body)
	if err != nil {
		t.Fatalf("read playlist body: %v", err)
	}
	if !strings.Contains(string(playlistBody), "segment_000.ts") {
		t.Fatalf("unexpected openlist playlist body: %s", string(playlistBody))
	}
}

func writeRouterFakeFFprobe(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ffprobe")
	content := "#!/bin/sh\ncat <<'EOF'\n{\"streams\":[{\"codec_type\":\"video\",\"codec_name\":\"h264\",\"width\":1920,\"height\":1080}],\"format\":{\"duration\":\"7260.25\",\"bit_rate\":\"5000000\"}}\nEOF\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake ffprobe: %v", err)
	}
	return path
}

func writeRouterFakeFFmpeg(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ffmpeg")
	content := "#!/bin/sh\nsegment_pattern=\"\"\nplaylist_path=\"\"\nprev=\"\"\nfor arg in \"$@\"; do\n  if [ \"$prev\" = \"segment\" ]; then\n    segment_pattern=\"$arg\"\n    prev=\"\"\n    continue\n  fi\n  if [ \"$arg\" = \"-hls_segment_filename\" ]; then\n    prev=\"segment\"\n    continue\n  fi\n  playlist_path=\"$arg\"\ndone\nsegment_file=$(printf \"$segment_pattern\" 0)\nmkdir -p \"$(dirname \"$playlist_path\")\"\nmkdir -p \"$(dirname \"$segment_file\")\"\ncat <<'EOF' > \"$playlist_path\"\n#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:6\n#EXT-X-MEDIA-SEQUENCE:0\n#EXTINF:6.0,\nsegment_000.ts\n#EXT-X-ENDLIST\nEOF\nprintf 'fake-hls-segment' > \"$segment_file\"\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake ffmpeg: %v", err)
	}
	return path
}

func registerAndLoginRouterUser(t *testing.T, ctx context.Context, authSvc *auth.Service, username string) auth.LoginResult {
	t.Helper()
	registeredUser, err := authSvc.Register(ctx, username, "password123")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	if registeredUser.ID == 0 {
		t.Fatal("expected created user id")
	}
	loginResult, err := authSvc.Login(ctx, username, "password123")
	if err != nil {
		t.Fatalf("login user: %v", err)
	}
	return loginResult
}

func newDeleteTestRouter(t *testing.T) (http.Handler, *gorm.DB, *auth.Service, *library.Service, string) {
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
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settings.NewService(db, cfg.Metadata)), settings.NewService(db, cfg.Metadata))

	return router, db, authSvc, librarySvc, storageRoot
}

func newPlaybackDecisionRouter(t *testing.T, enableHLS bool) (http.Handler, *auth.Service, uint) {
	t.Helper()

	storageRoot := filepath.Join(t.TempDir(), "playback-decision-root")
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
		FFmpeg:   config.FFmpegConfig{Enabled: enableHLS, Path: writeRouterFakeFFmpeg(t), Timeout: 2 * time.Second},
		HLS:      config.HLSConfig{Enabled: enableHLS, RootPath: filepath.Join(t.TempDir(), "hls"), SegmentDuration: 6, CleanupAge: time.Hour},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settings.NewService(db, cfg.Metadata)), settings.NewService(db, cfg.Metadata))

	ctx := context.Background()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Playback Decision Source", RootPath: storageRoot})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	record, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: storageRoot})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}

	mediaPath := filepath.Join(storageRoot, "decision.mkv")
	if err := os.WriteFile(mediaPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write media file: %v", err)
	}

	item := database.MediaItem{LibraryID: record.ID, Type: "movie", Title: "Decision Movie", SourcePath: mediaPath, MatchStatus: "matched", Status: "ready"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create media item: %v", err)
	}
	file := database.MediaFile{LibraryID: record.ID, MediaItemID: &item.ID, StoragePath: mediaPath, Container: "mkv", ProbeStatus: probe.StatusReady, VideoCodec: "hevc"}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create media file: %v", err)
	}

	return router, authSvc, item.ID
}

func seedLibraryData(t *testing.T, ctx context.Context, db *gorm.DB, authSvc *auth.Service, libraryID uint, rootDir, name string) (uint, uint, uint) {
	t.Helper()

	user, err := authSvc.Register(ctx, fmt.Sprintf("%s-user", strings.ToLower(name)), "password123")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	item := database.MediaItem{
		LibraryID:   libraryID,
		Type:        "movies",
		Title:       name,
		SourcePath:  filepath.Join(rootDir, name),
		MatchStatus: "matched",
		Status:      "ready",
	}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create media item: %v", err)
	}

	file := database.MediaFile{
		LibraryID:   libraryID,
		MediaItemID: &item.ID,
		StoragePath: filepath.Join(rootDir, name+".mkv"),
		ProbeStatus: "ready",
	}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create media file: %v", err)
	}

	progressRecord := database.PlaybackProgress{
		UserID:          user.ID,
		MediaItemID:     item.ID,
		MediaFileID:     &file.ID,
		PositionSeconds: 120,
	}
	if err := db.WithContext(ctx).Create(&progressRecord).Error; err != nil {
		t.Fatalf("create progress: %v", err)
	}

	return user.ID, item.ID, file.ID
}

func createAuthHeader(t *testing.T, ctx context.Context, authSvc *auth.Service) string {
	t.Helper()

	username := fmt.Sprintf("user-%d", time.Now().UnixNano())
	if _, err := authSvc.Register(ctx, username, "password123"); err != nil {
		t.Fatalf("register auth user: %v", err)
	}
	loginResult, err := authSvc.Login(ctx, username, "password123")
	if err != nil {
		t.Fatalf("login auth user: %v", err)
	}
	return fmt.Sprintf("Bearer %s", loginResult.Token)
}

func createAdminAuthHeader(t *testing.T, ctx context.Context, db *gorm.DB, authSvc *auth.Service) string {
	t.Helper()

	username := fmt.Sprintf("admin-%d", time.Now().UnixNano())
	user, err := authSvc.Register(ctx, username, "password123")
	if err != nil {
		t.Fatalf("register admin auth user: %v", err)
	}
	if err := db.WithContext(ctx).Model(&database.User{}).Where("id = ?", user.ID).Update("role", "admin").Error; err != nil {
		t.Fatalf("promote admin auth user: %v", err)
	}
	loginResult, err := authSvc.Login(ctx, username, "password123")
	if err != nil {
		t.Fatalf("login admin auth user: %v", err)
	}
	return fmt.Sprintf("Bearer %s", loginResult.Token)
}

func newStorageEventTestRouter(t *testing.T) (http.Handler, *gorm.DB, *auth.Service, *library.Service, uint, string) {
	t.Helper()

	mediaRoot := filepath.Join(t.TempDir(), "storage-events-root")
	movieDir := filepath.Join(mediaRoot, "Movies")
	if err := os.MkdirAll(movieDir, 0o755); err != nil {
		t.Fatalf("create movie dir: %v", err)
	}
	moviePath := filepath.Join(movieDir, "MovieA.2024.mkv")
	if err := os.WriteFile(moviePath, []byte("movie"), 0o644); err != nil {
		t.Fatalf("write movie file: %v", err)
	}

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		Storage:  config.StorageConfig{Provider: "local"},
		Local:    config.LocalStorageConfig{RootPath: mediaRoot},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settingsSvc), settingsSvc)

	ctx := context.Background()
	source, err := librarySvc.CreateMediaSource(ctx, library.CreateMediaSourceInput{Provider: "local", Name: "Local", RootPath: mediaRoot})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	record, _, err := librarySvc.CreateLibrary(ctx, library.CreateLibraryInput{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: movieDir})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}

	return router, db, authSvc, librarySvc, record.ID, moviePath
}

func newScheduleTestRouter(t *testing.T) (http.Handler, *auth.Service, *gorm.DB, *jobs.Service) {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	cfg := config.Config{Database: config.DatabaseConfig{Driver: "sqlite"}, Storage: config.StorageConfig{Provider: "local"}, Local: config.LocalStorageConfig{RootPath: t.TempDir()}, Worker: config.WorkerConfig{Enabled: true, PollInterval: time.Millisecond}}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	searchSvc := search.NewService(db)
	metadataSvc := metadata.NewService(db, cfg.Metadata, settingsSvc, searchSvc)
	scheduleSvc := schedule.NewService(db, schedule.WithJobs(jobsSvc))
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db, searchSvc), searchSvc, metadataSvc, settingsSvc, scheduleSvc)
	return router, authSvc, db, jobsSvc
}

func intPtr(value int) *int {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func mustDecodeListenerRefreshPayload(t *testing.T, raw string) listenerPayloadView {
	t.Helper()
	var payload listenerPayloadView
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode listener payload: %v", err)
	}
	return payload
}

func mustDecodeStorageEventResponseJob(t *testing.T, raw []byte) database.Job {
	t.Helper()
	var response struct {
		Data database.Job `json:"data"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatalf("decode storage event response: %v", err)
	}
	return response.Data
}

type listenerPayloadView struct {
	LibraryID        uint      `json:"library_id"`
	RootPath         string    `json:"root_path"`
	FallbackFullSync bool      `json:"fallback_full_sync"`
	WindowStartedAt  time.Time `json:"window_started_at"`
	WindowEndsAt     time.Time `json:"window_ends_at"`
}

func assertDeletedCount(t *testing.T, ctx context.Context, db *gorm.DB, model any, query string, value any, want int64) {
	t.Helper()

	var count int64
	if err := db.WithContext(ctx).Model(model).Where(query, value).Count(&count).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	if count != want {
		t.Fatalf("unexpected count for %T with %q: got %d want %d", model, query, count, want)
	}
}

func TestTVMetadataEndpoints(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/tv/777":
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
		case "/tv/777/season/1":
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

	storageRoot := t.TempDir()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite"},
		Storage:  config.StorageConfig{Provider: "local"},
		Local:    config.LocalStorageConfig{RootPath: storageRoot},
		Metadata: config.MetadataConfig{
			TMDB: config.TMDBConfig{BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second},
		},
	}
	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	showLibrary := database.Library{Name: "Shows", Type: "tvshows", RootPath: filepath.Join(storageRoot, "shows"), Status: "active", ScannerEnabled: true}
	if err := db.WithContext(context.Background()).Create(&showLibrary).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	episodeOne := 1
	tvMatchedItem := database.MediaItem{
		LibraryID:     showLibrary.ID,
		Type:          "episode",
		Title:         "Pilot",
		SeriesTitle:   "Show A",
		ExternalID:    "tv:777",
		SeasonNumber:  &episodeOne,
		EpisodeNumber: &episodeOne,
		SourcePath:    filepath.Join(storageRoot, "shows", "show-a-s01e01.mkv"),
		MatchStatus:   "matched",
		Status:        "ready",
	}
	if err := db.WithContext(context.Background()).Create(&tvMatchedItem).Error; err != nil {
		t.Fatalf("create media item: %v", err)
	}
	localSeasonTwo := 2
	localEpisodeOne := 1
	localEpisodeTwo := 2
	localEpisodeA := database.MediaItem{
		LibraryID:     showLibrary.ID,
		Type:          "episode",
		Title:         "坠落",
		SeriesTitle:   "本地剧 第一季",
		SeasonNumber:  &episodeOne,
		EpisodeNumber: &localEpisodeOne,
		BackdropURL:   "/local-s1e1.jpg",
		SourcePath:    filepath.Join(storageRoot, "shows", "local-show-s01e01.mkv"),
		Status:        "ready",
	}
	localEpisodeB := database.MediaItem{
		LibraryID:     showLibrary.ID,
		Type:          "episode",
		Title:         "觉醒",
		SeriesTitle:   "本地剧 第二季",
		SeasonNumber:  &localSeasonTwo,
		EpisodeNumber: &localEpisodeTwo,
		PosterURL:     "/local-s2.jpg",
		SourcePath:    filepath.Join(storageRoot, "shows", "local-show-s02e02.mkv"),
		Status:        "ready",
	}
	if err := db.WithContext(context.Background()).Create(&localEpisodeA).Error; err != nil {
		t.Fatalf("create local episode a: %v", err)
	}
	if err := db.WithContext(context.Background()).Create(&localEpisodeB).Error; err != nil {
		t.Fatalf("create local episode b: %v", err)
	}
	if _, err := settingsSvc.UpdateMetadataSettings(context.Background(), settings.UpdateMetadataSettingsInput{
		TMDB: settings.MetadataProviderInput{
			APIKey:       "router-tv-key",
			BaseURL:      tmdb.URL,
			ImageBaseURL: tmdb.URL + "/images",
			Language:     "en-US",
			Timeout:      "1s",
		},
	}); err != nil {
		t.Fatalf("update metadata settings: %v", err)
	}
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	router := New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playback.NewService(db, registry), progress.NewService(db), search.NewService(), metadata.NewService(db, cfg.Metadata, settingsSvc), settingsSvc)

	t.Run("list seasons", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/tv/777/seasons", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("seasons status: %d body=%s", recorder.Code, recorder.Body.String())
		}

		var body struct {
			Data []struct {
				SeasonNumber int    `json:"season_number"`
				Name         string `json:"name"`
				Overview     string `json:"overview"`
				PosterURL    string `json:"poster_url"`
			} `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode seasons response: %v", err)
		}
		if len(body.Data) != 1 || body.Data[0].SeasonNumber != 1 || body.Data[0].Name != "Season 1" {
			t.Fatalf("unexpected seasons payload: %#v", body.Data)
		}
		if body.Data[0].PosterURL != tmdb.URL+"/images/season-1.jpg" {
			t.Fatalf("unexpected poster url: %q", body.Data[0].PosterURL)
		}
	})

	t.Run("list episodes", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/tv/777/seasons/1/episodes", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("episodes status: %d body=%s", recorder.Code, recorder.Body.String())
		}

		var body struct {
			Data []struct {
				MediaItemID    *uint  `json:"media_item_id"`
				SeasonNumber   int    `json:"season_number"`
				EpisodeNumber  int    `json:"episode_number"`
				Name           string `json:"name"`
				Overview       string `json:"overview"`
				StillURL       string `json:"still_url"`
				RuntimeSeconds *int   `json:"runtime_seconds"`
				PayloadJSON    string `json:"payload_json"`
			} `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode episodes response: %v", err)
		}
		if len(body.Data) != 1 || body.Data[0].EpisodeNumber != 1 || body.Data[0].Name != "Pilot" {
			t.Fatalf("unexpected episodes payload: %#v", body.Data)
		}
		if body.Data[0].StillURL != tmdb.URL+"/images/pilot-still.jpg" {
			t.Fatalf("unexpected still url: %q", body.Data[0].StillURL)
		}
		if body.Data[0].RuntimeSeconds == nil || *body.Data[0].RuntimeSeconds != 2880 {
			t.Fatalf("unexpected runtime seconds: %#v", body.Data[0].RuntimeSeconds)
		}
		if body.Data[0].MediaItemID == nil || *body.Data[0].MediaItemID == 0 {
			t.Fatalf("expected episode media_item_id, got %#v", body.Data[0].MediaItemID)
		}
		if body.Data[0].PayloadJSON != "" {
			t.Fatalf("expected sanitized payload without raw tmdb json, got %#v", body.Data[0])
		}
	})

	t.Run("list local series episodes", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media-items/%d/series-episodes", localEpisodeA.ID), nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("local series episodes status: %d body=%s", recorder.Code, recorder.Body.String())
		}

		var body struct {
			Data []struct {
				SeasonNumber int    `json:"season_number"`
				Name         string `json:"name"`
				PosterURL    string `json:"poster_url"`
				Episodes     []struct {
					MediaItemID   uint   `json:"media_item_id"`
					EpisodeNumber int    `json:"episode_number"`
					Name          string `json:"name"`
					StillURL      string `json:"still_url"`
				} `json:"episodes"`
			} `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode local series response: %v", err)
		}
		if len(body.Data) != 1 || body.Data[0].SeasonNumber != 1 {
			t.Fatalf("unexpected local seasons payload: %#v", body.Data)
		}
		if body.Data[0].Episodes[0].StillURL != requestBaseURL(request)+"/local-s1e1.jpg" {
			t.Fatalf("unexpected local still url: %#v", body.Data[0].Episodes[0])
		}
		if body.Data[0].PosterURL != "" {
			t.Fatalf("unexpected local poster url: %#v", body.Data[0])
		}
	})
}
