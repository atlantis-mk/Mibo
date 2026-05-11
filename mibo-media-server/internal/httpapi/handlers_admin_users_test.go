package httpapi

import (
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
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/progress"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
)

func TestAdminUsersRequireAdmin(t *testing.T) {
	handler, _, userHeader := newAdminUsersTestServer(t)

	for _, tc := range []struct {
		name       string
		method     string
		body       string
		authHeader string
		wantStatus int
	}{
		{name: "anonymous list", method: http.MethodGet, wantStatus: http.StatusUnauthorized},
		{name: "anonymous create", method: http.MethodPost, body: `{"username":"guest","password":"password123","role":"user"}`, wantStatus: http.StatusUnauthorized},
		{name: "non-admin list", method: http.MethodGet, authHeader: userHeader, wantStatus: http.StatusForbidden},
		{name: "non-admin create", method: http.MethodPost, body: `{"username":"guest","password":"password123","role":"user"}`, authHeader: userHeader, wantStatus: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/v1/admin/users", strings.NewReader(tc.body))
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAdminUsersListAndCreateOrdinaryUser(t *testing.T) {
	handler, adminHeader, _ := newAdminUsersTestServer(t)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", strings.NewReader(`{"username":"Viewer","password":"password123","role":"user"}`))
	createReq.Header.Set("Authorization", adminHeader)
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	created := decodeAdminUserResponse(t, createRec)
	if created.Username != "viewer" || created.Role != "user" {
		t.Fatalf("unexpected created user: %#v", created)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	listReq.Header.Set("Authorization", adminHeader)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	users := decodeAdminUserListResponse(t, listRec)
	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d: %#v", len(users), users)
	}
	if users[2].Username != "viewer" || users[2].Role != "user" {
		t.Fatalf("expected viewer user in list, got %#v", users)
	}
}

func TestAdminUsersCreateAdminAndRejectInvalidRequests(t *testing.T) {
	handler, adminHeader, _ := newAdminUsersTestServer(t)

	createAdminReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", strings.NewReader(`{"username":"SecondAdmin","password":"password123","role":"admin"}`))
	createAdminReq.Header.Set("Authorization", adminHeader)
	createAdminRec := httptest.NewRecorder()
	handler.ServeHTTP(createAdminRec, createAdminReq)

	if createAdminRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createAdminRec.Code, createAdminRec.Body.String())
	}
	created := decodeAdminUserResponse(t, createAdminRec)
	if created.Username != "secondadmin" || created.Role != "admin" {
		t.Fatalf("unexpected created admin: %#v", created)
	}

	duplicateReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", strings.NewReader(`{"username":"secondadmin","password":"password123","role":"user"}`))
	duplicateReq.Header.Set("Authorization", adminHeader)
	duplicateRec := httptest.NewRecorder()
	handler.ServeHTTP(duplicateRec, duplicateReq)
	if duplicateRec.Code != http.StatusBadRequest {
		t.Fatalf("expected duplicate 400, got %d: %s", duplicateRec.Code, duplicateRec.Body.String())
	}

	invalidRoleReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", strings.NewReader(`{"username":"operator","password":"password123","role":"owner"}`))
	invalidRoleReq.Header.Set("Authorization", adminHeader)
	invalidRoleRec := httptest.NewRecorder()
	handler.ServeHTTP(invalidRoleRec, invalidRoleReq)
	if invalidRoleRec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid role 400, got %d: %s", invalidRoleRec.Code, invalidRoleRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	listReq.Header.Set("Authorization", adminHeader)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	users := decodeAdminUserListResponse(t, listRec)
	if len(users) != 3 {
		t.Fatalf("expected rejected requests not to create users, got %#v", users)
	}
}

func newAdminUsersTestServer(t *testing.T) (http.Handler, string, string) {
	t.Helper()
	cfg := config.Config{
		HTTP:     config.HTTPConfig{Addr: ":8080"},
		Local:    config.LocalStorageConfig{RootPath: t.TempDir()},
		Database: config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")},
		Worker:   config.WorkerConfig{Enabled: true},
	}
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

	adminHeader := loginTestUser(t, authSvc, "admin-user", "password123")
	userHeader := loginTestUser(t, authSvc, "regular-user", "password123")
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
	return handler, adminHeader, userHeader
}

func loginTestUser(t *testing.T, authSvc *auth.Service, username, password string) string {
	t.Helper()
	if _, err := authSvc.Register(t.Context(), username, password); err != nil {
		t.Fatalf("register %s: %v", username, err)
	}
	login, err := authSvc.Login(t.Context(), username, password)
	if err != nil {
		t.Fatalf("login %s: %v", username, err)
	}
	return "Bearer " + login.Token
}

func decodeAdminUserResponse(t *testing.T, rec *httptest.ResponseRecorder) adminUserResponse {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	var user adminUserResponse
	if err := json.Unmarshal(data, &user); err != nil {
		t.Fatalf("decode user: %v", err)
	}
	return user
}

func decodeAdminUserListResponse(t *testing.T, rec *httptest.ResponseRecorder) []adminUserResponse {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	var users []adminUserResponse
	if err := json.Unmarshal(data, &users); err != nil {
		t.Fatalf("decode users: %v", err)
	}
	return users
}
