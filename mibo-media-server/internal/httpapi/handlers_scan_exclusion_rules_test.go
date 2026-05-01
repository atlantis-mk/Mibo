package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
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
