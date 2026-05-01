package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
)

func TestMetaTubeClientSearchAndDetail(t *testing.T) {
	seenAuth := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		switch r.URL.Path {
		case "/v1/movies/search":
			if got := r.URL.Query().Get("q"); got != "movie title" {
				t.Fatalf("unexpected q: %q", got)
			}
			if got := r.URL.Query().Get("provider"); got != "fanza" {
				t.Fatalf("unexpected provider filter: %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"provider": "fanza", "id": "abc123", "title": "Movie Title", "cover_url": "https://img.example/poster.jpg"}}})
		case "/v1/movies/fanza/abc123":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"provider": "fanza", "id": "abc123", "title": "Movie Title", "summary": "Overview", "runtime": 121, "release_date": "2024-02-02", "actors": []map[string]any{{"name": "Actor"}}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	svc := &Service{}
	cfg := config.MetaTubeConfig{BaseURL: server.URL + "/", Token: "token", UpstreamProviderFilter: "fanza", FallbackEnabled: true, Timeout: time.Second}
	results, err := svc.searchMetaTube(context.Background(), cfg, "movie title")
	if err != nil {
		t.Fatalf("search metatube: %v", err)
	}
	if seenAuth != "Bearer token" {
		t.Fatalf("expected bearer auth header, got %q", seenAuth)
	}
	if len(results) != 1 || results[0].Provider != "fanza" || results[0].ID != "abc123" {
		t.Fatalf("unexpected search results: %#v", results)
	}
	detail, err := svc.fetchMetaTubeDetail(context.Background(), cfg, "fanza", "abc123")
	if err != nil {
		t.Fatalf("fetch metatube detail: %v", err)
	}
	if detail.Title != "Movie Title" || detail.Runtime == nil || *detail.Runtime != 121 || len(detail.Actors) != 1 {
		t.Fatalf("unexpected detail: %#v", detail)
	}
}

func TestMetaTubeClientOmitsAuthWhenTokenMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("expected no authorization header, got %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
	}))
	defer server.Close()

	_, err := (&Service{}).searchMetaTube(context.Background(), config.MetaTubeConfig{BaseURL: server.URL, Timeout: time.Second}, "movie")
	if err != nil {
		t.Fatalf("search metatube without token: %v", err)
	}
}

func TestMetaTubeClientHTTPErrorMapping(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{name: "not found", status: http.StatusNotFound},
		{name: "auth", status: http.StatusUnauthorized},
		{name: "rate limit", status: http.StatusTooManyRequests},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": tc.name})
			}))
			defer server.Close()

			_, err := (&Service{}).searchMetaTube(context.Background(), config.MetaTubeConfig{BaseURL: server.URL, Timeout: time.Second}, "movie")
			var failure providerRequestFailure
			if !errors.As(err, &failure) || failure.StatusCode() != tc.status {
				t.Fatalf("expected mapped status %d, got %#v", tc.status, err)
			}
		})
	}
}
