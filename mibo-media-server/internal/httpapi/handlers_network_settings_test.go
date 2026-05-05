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
	"gorm.io/gorm"
)

func TestNetworkSettingsDefaultsAndPersistence(t *testing.T) {
	handler, authHeader, db := newNetworkSettingsTestServer(t)

	defaults := requestNetworkSettings(t, handler, authHeader, http.MethodGet, "")
	if defaults.LocalHTTPPort != 8096 || defaults.LocalHTTPSPort != 8920 || defaults.RemoteIPFilterMode != "allow" {
		t.Fatalf("unexpected defaults: %#v", defaults)
	}
	if defaults.EffectiveStatus.Source != "defaults" || len(defaults.EffectiveStatus.RestartRequiredFields) == 0 {
		t.Fatalf("expected default status metadata, got %#v", defaults.EffectiveStatus)
	}

	updated := requestNetworkSettings(t, handler, authHeader, http.MethodPut, validNetworkSettingsPayload(`"certificate_password":"secret"`))
	if updated.LocalHTTPPort != 8080 || updated.PublicHTTPSPort != 9443 || updated.ExternalDomain != "media.example.com" {
		t.Fatalf("unexpected updated settings: %#v", updated)
	}
	if !updated.CertificatePassword.Configured || !updated.CertificatePassword.Masked {
		t.Fatalf("expected masked certificate password state, got %#v", updated.CertificatePassword)
	}
	if updated.EffectiveStatus.Source != "database" || updated.EffectiveStatus.AutomaticPortMappingActive {
		t.Fatalf("unexpected effective status: %#v", updated.EffectiveStatus)
	}

	var secret database.SystemSetting
	if err := db.WithContext(t.Context()).Where("category = ? AND key = ?", "network", "certificate_password").First(&secret).Error; err != nil {
		t.Fatalf("load secret setting: %v", err)
	}
	if !secret.IsSecret || secret.Value != "secret" {
		t.Fatalf("expected stored secret, got %#v", secret)
	}

	reloaded := requestNetworkSettings(t, handler, authHeader, http.MethodGet, "")
	if reloaded.LocalIPAddress != "192.168.1.50" || len(reloaded.RemoteIPFilter) != 2 || !reloaded.CertificatePassword.Configured {
		t.Fatalf("expected persisted settings, got %#v", reloaded)
	}
}

func TestNetworkSettingsValidationFailures(t *testing.T) {
	handler, authHeader, _ := newNetworkSettingsTestServer(t)

	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "invalid address", body: validNetworkSettingsPayload(`"local_networks":["not-a-network"]`)},
		{name: "invalid port", body: validNetworkSettingsPayload(`"local_http_port":70000`)},
		{name: "invalid enum", body: validNetworkSettingsPayload(`"secure_connection_mode":"always"`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/network", strings.NewReader(tc.body))
			req.Header.Set("Authorization", authHeader)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestNetworkSettingsClearsCertificatePasswordAndRequiresAuth(t *testing.T) {
	handler, authHeader, db := newNetworkSettingsTestServer(t)
	_ = requestNetworkSettings(t, handler, authHeader, http.MethodPut, validNetworkSettingsPayload(`"certificate_password":"secret"`))

	cleared := requestNetworkSettings(t, handler, authHeader, http.MethodPut, validNetworkSettingsPayload(`"clear_certificate_password":true`))
	if cleared.CertificatePassword.Configured || cleared.CertificatePassword.Masked {
		t.Fatalf("expected cleared password state, got %#v", cleared.CertificatePassword)
	}
	var count int64
	if err := db.WithContext(t.Context()).Model(&database.SystemSetting{}).Where("category = ? AND key = ?", "network", "certificate_password").Count(&count).Error; err != nil {
		t.Fatalf("count secret setting: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected secret setting to be deleted, got count %d", count)
	}

	for _, method := range []string{http.MethodGet, http.MethodPut} {
		req := httptest.NewRequest(method, "/api/v1/settings/network", strings.NewReader(validNetworkSettingsPayload("")))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected %s unauthorized, got %d", method, rec.Code)
		}
	}
}

func newNetworkSettingsTestServer(t *testing.T) (http.Handler, string, *gorm.DB) {
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
	authHeader := loginTestUser(t, authSvc, "network-user", "password123")
	handler := New(cfg, db, registry, authSvc, librarySvc, nil, playbackSvc, progressSvc, searchSvc, nil, settingsSvc, catalogSvc)
	return handler, authHeader, db
}

func requestNetworkSettings(t *testing.T, handler http.Handler, authHeader, method, body string) settings.NetworkSettings {
	t.Helper()
	req := httptest.NewRequest(method, "/api/v1/settings/network", strings.NewReader(body))
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	var result settings.NetworkSettings
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode network settings: %v", err)
	}
	return result
}

func validNetworkSettingsPayload(override string) string {
	fields := []string{
		`"local_networks":["192.168.1.0/24","10.0.0.0/8"]`,
		`"local_ip_address":"192.168.1.50"`,
		`"local_http_port":8080`,
		`"local_https_port":9443`,
		`"allow_remote_access":true`,
		`"remote_ip_filter":["203.0.113.10","198.51.100.0/24"]`,
		`"remote_ip_filter_mode":"block"`,
		`"public_http_port":8080`,
		`"public_https_port":9443`,
		`"external_domain":"media.example.com"`,
		`"trust_proxy_headers":true`,
		`"ssl_certificate_path":"/config/certs/mibo.pfx"`,
		`"secure_connection_mode":"preferred"`,
		`"automatic_port_mapping":true`,
		`"max_video_streams":"4"`,
		`"remote_streaming_bitrate_limit":"8mbps"`,
		`"network_request_protocol":"ipv4"`,
	}
	if strings.TrimSpace(override) != "" {
		name := strings.SplitN(strings.TrimSpace(override), ":", 2)[0]
		for index, field := range fields {
			if strings.HasPrefix(field, name+":") {
				fields[index] = override
				return "{" + strings.Join(fields, ",") + "}"
			}
		}
		fields = append(fields, override)
	}
	return "{" + strings.Join(fields, ",") + "}"
}
