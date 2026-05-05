package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
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

func TestAuthSessionsRequireBearerToken(t *testing.T) {
	handler, _ := newAuthSessionsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthSessionsListAndRevokeThroughHTTP(t *testing.T) {
	handler, authSvc := newAuthSessionsTestServer(t)

	if _, err := authSvc.Register(t.Context(), "alice", "password123"); err != nil {
		t.Fatalf("register alice: %v", err)
	}
	first, err := authSvc.Login(t.Context(), "alice", "password123")
	if err != nil {
		t.Fatalf("login first session: %v", err)
	}
	secondReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"alice","password":"password123"}`))
	secondReq.Header.Set("Content-Type", "application/json")
	secondReq.Header.Set("User-Agent", "Mozilla/5.0 (iPhone)")
	secondReq.RemoteAddr = "192.0.2.10:4567"
	secondRec := httptest.NewRecorder()
	handler.ServeHTTP(secondRec, secondReq)
	if secondRec.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d: %s", secondRec.Code, secondRec.Body.String())
	}
	secondToken := decodeLoginToken(t, secondRec)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
	listReq.Header.Set("Authorization", "Bearer "+secondToken)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	sessions := decodeAuthSessionList(t, listRec)
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d: %#v", len(sessions), sessions)
	}
	var current, other loginSessionResponse
	for _, session := range sessions {
		if session.IsCurrent {
			current = session
		} else {
			other = session
		}
	}
	if current.ID == 0 || other.ID == 0 {
		t.Fatalf("expected current and other sessions, got %#v", sessions)
	}
	if current.DeviceName != "iPhone" || current.RemoteAddr != "192.0.2.10" {
		t.Fatalf("expected captured login metadata, got %#v", current)
	}

	revokeCurrentReq := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/"+uintString(current.ID), nil)
	revokeCurrentReq.Header.Set("Authorization", "Bearer "+secondToken)
	revokeCurrentRec := httptest.NewRecorder()
	handler.ServeHTTP(revokeCurrentRec, revokeCurrentReq)
	if revokeCurrentRec.Code != http.StatusBadRequest {
		t.Fatalf("expected current revoke 400, got %d: %s", revokeCurrentRec.Code, revokeCurrentRec.Body.String())
	}

	revokeOtherReq := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/"+uintString(other.ID), nil)
	revokeOtherReq.Header.Set("Authorization", "Bearer "+secondToken)
	revokeOtherRec := httptest.NewRecorder()
	handler.ServeHTTP(revokeOtherRec, revokeOtherReq)
	if revokeOtherRec.Code != http.StatusOK {
		t.Fatalf("expected revoke 200, got %d: %s", revokeOtherRec.Code, revokeOtherRec.Body.String())
	}
	if _, err := authSvc.Authenticate(t.Context(), first.Token); err == nil {
		t.Fatal("expected first token to be revoked")
	}
	if _, err := authSvc.Authenticate(t.Context(), secondToken); err != nil {
		t.Fatalf("current token should remain valid: %v", err)
	}
}

func newAuthSessionsTestServer(t *testing.T) (http.Handler, *auth.Service) {
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
	handler := New(cfg, db, registry, authSvc, librarySvc, nil, playbackSvc, progressSvc, searchSvc, nil, settingsSvc, catalogSvc)
	return handler, authSvc
}

type loginSessionResponse struct {
	ID         uint   `json:"id"`
	UserAgent  string `json:"user_agent"`
	RemoteAddr string `json:"remote_addr"`
	DeviceName string `json:"device_name"`
	ClientType string `json:"client_type"`
	IsCurrent  bool   `json:"is_current"`
}

func decodeAuthSessionList(t *testing.T, rec *httptest.ResponseRecorder) []loginSessionResponse {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	var sessions []loginSessionResponse
	if err := json.Unmarshal(data, &sessions); err != nil {
		t.Fatalf("decode sessions: %v", err)
	}
	return sessions
}

func decodeLoginToken(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode login result: %v", err)
	}
	return result.Token
}

func uintString(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}
