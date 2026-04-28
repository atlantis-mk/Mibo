package openlist

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/storage"
)

func TestListCapturesSafeMetadataAndFiltersSensitiveJSON(t *testing.T) {
	t.Parallel()

	created := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/fs/list" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		writeOpenListTestEnvelope(t, w, map[string]any{
			"provider": "AliyundriveOpen",
			"content": []map[string]any{{
				"name":          "Movie A.mkv",
				"is_dir":        false,
				"size":          42,
				"created":       created.Format(time.RFC3339),
				"modified":      created.Add(time.Hour).Format(time.RFC3339),
				"type":          2,
				"sign":          "secret-sign",
				"thumb":         " https://cdn.example.test/movie-thumb.jpg ",
				"hash_info":     map[string]string{"sha1": "abc"},
				"mount_details": map[string]any{"driver": "secret"},
			}},
		})
	}))
	t.Cleanup(server.Close)

	adapter := New(config.OpenListConfig{BaseURL: server.URL, Timeout: time.Second})
	objects, err := adapter.List(context.Background(), storage.ListRequest{Path: "/Movies"})
	if err != nil {
		t.Fatalf("list objects: %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}
	object := objects[0]
	if object.Created == nil || !object.Created.Equal(created) {
		t.Fatalf("expected created time to be preserved, got %#v", object.Created)
	}
	if object.ObjectType != "2" || object.Sign != "secret-sign" || object.ProviderMeta["has_mount_details"] != "true" {
		t.Fatalf("expected metadata to be captured, got %#v", object)
	}
	encoded, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal object: %v", err)
	}
	encodedText := string(encoded)
	if strings.Contains(encodedText, "secret-sign") || strings.Contains(encodedText, "mount_details") || strings.Contains(encodedText, "ProviderMeta") {
		t.Fatalf("expected sensitive metadata filtered from JSON, got %s", encodedText)
	}
}

func TestListPreservesThumbnailURL(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/fs/list" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		writeOpenListTestEnvelope(t, w, map[string]any{
			"provider": "AliyundriveOpen",
			"content": []map[string]any{{
				"name":      "Movie A.mkv",
				"is_dir":    false,
				"size":      42,
				"thumb":     " https://cdn.example.test/movie-thumb.jpg ",
				"hash_info": map[string]string{"sha1": "abc"},
			}},
		})
	}))
	t.Cleanup(server.Close)

	adapter := New(config.OpenListConfig{BaseURL: server.URL, Timeout: time.Second})
	objects, err := adapter.List(context.Background(), storage.ListRequest{Path: "/Movies"})
	if err != nil {
		t.Fatalf("list objects: %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}
	if objects[0].ThumbnailURL != "https://cdn.example.test/movie-thumb.jpg" {
		t.Fatalf("expected thumbnail URL to be preserved, got %#v", objects[0])
	}
}

func TestGetPreservesThumbnailURL(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/fs/get" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		writeOpenListTestEnvelope(t, w, map[string]any{
			"name":     "Movie A.mkv",
			"is_dir":   false,
			"size":     42,
			"raw_url":  "https://cdn.example.test/movie.mkv",
			"thumb":    "https://cdn.example.test/movie-thumb.jpg",
			"provider": "AliyundriveOpen",
		})
	}))
	t.Cleanup(server.Close)

	adapter := New(config.OpenListConfig{BaseURL: server.URL, Timeout: time.Second})
	object, err := adapter.Get(context.Background(), storage.GetRequest{Path: "/Movies/Movie A.mkv"})
	if err != nil {
		t.Fatalf("get object: %v", err)
	}
	if object.ThumbnailURL != "https://cdn.example.test/movie-thumb.jpg" {
		t.Fatalf("expected thumbnail URL to be preserved, got %#v", object)
	}
}

func TestGetCapturesRelatedObjectsWithReconstructedPaths(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/fs/get" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		writeOpenListTestEnvelope(t, w, map[string]any{
			"name":     "Movie A.mkv",
			"is_dir":   false,
			"size":     42,
			"type":     "video",
			"provider": "AliyundriveOpen",
			"related": []map[string]any{{
				"name":      "cover.jpg",
				"is_dir":    false,
				"size":      12,
				"raw_url":   "https://cdn.example.test/cover.jpg",
				"hash_info": map[string]string{"md5": "def"},
			}},
		})
	}))
	t.Cleanup(server.Close)

	adapter := New(config.OpenListConfig{BaseURL: server.URL, Timeout: time.Second})
	object, err := adapter.Get(context.Background(), storage.GetRequest{Path: "/Movies/Movie A.mkv"})
	if err != nil {
		t.Fatalf("get object: %v", err)
	}
	if object.ObjectType != "video" || object.ProviderMeta["has_related"] != "true" {
		t.Fatalf("expected get metadata, got %#v", object)
	}
	if len(object.Related) != 1 {
		t.Fatalf("expected 1 related object, got %#v", object.Related)
	}
	if object.Related[0].Path != "/Movies/cover.jpg" || object.Related[0].RawURL != "https://cdn.example.test/cover.jpg" {
		t.Fatalf("expected reconstructed related object, got %#v", object.Related[0])
	}
}

func TestGetMissingOptionalMetadataRemainsCompatible(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeOpenListTestEnvelope(t, w, map[string]any{"name": "Movie A.mkv", "is_dir": false})
	}))
	t.Cleanup(server.Close)

	adapter := New(config.OpenListConfig{BaseURL: server.URL, Timeout: time.Second})
	object, err := adapter.Get(context.Background(), storage.GetRequest{Path: "/Movies/Movie A.mkv"})
	if err != nil {
		t.Fatalf("get object: %v", err)
	}
	if object.Path != "/Movies/Movie A.mkv" || object.ProviderMeta != nil || object.Related != nil {
		t.Fatalf("expected missing metadata to stay empty, got %#v", object)
	}
}

func writeOpenListTestEnvelope(t *testing.T, w http.ResponseWriter, data any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"code": http.StatusOK, "data": data}); err != nil {
		t.Fatalf("write response: %v", err)
	}
}
