package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/atlan/mibo-media-server/internal/config"
)

func TestWebAppHandlerServesAssetsAndSPAFallback(t *testing.T) {
	handler := newWebAppHandler(config.WebConfig{}, fstest.MapFS{
		"index.html":    {Data: []byte("<html>Mibo</html>")},
		"assets/app.js": {Data: []byte("console.log('mibo')")},
	})

	assetReq := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	assetRec := httptest.NewRecorder()
	handler.ServeHTTP(assetRec, assetReq)
	if assetRec.Code != http.StatusOK {
		t.Fatalf("expected asset status 200, got %d", assetRec.Code)
	}
	if got := assetRec.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Fatalf("expected immutable asset cache header, got %q", got)
	}

	spaReq := httptest.NewRequest(http.MethodGet, "/libraries/1", nil)
	spaRec := httptest.NewRecorder()
	handler.ServeHTTP(spaRec, spaReq)
	if spaRec.Code != http.StatusOK {
		t.Fatalf("expected spa fallback status 200, got %d", spaRec.Code)
	}
	body, err := io.ReadAll(spaRec.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if !strings.Contains(string(body), "Mibo") {
		t.Fatalf("expected fallback to serve index.html, got %q", body)
	}
}

func TestWebAppHandlerDoesNotFallbackAPIPaths(t *testing.T) {
	handler := newWebAppHandler(config.WebConfig{}, fstest.MapFS{
		"index.html": {Data: []byte("<html>Mibo</html>")},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/missing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected api 404, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "Mibo") {
		t.Fatal("expected api path not to serve SPA index")
	}
}
