package metadata

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestLocalScannerEvidenceReaderAndCandidates(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Original", Path: "/movies/original.mkv", SortKey: "Original"})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	payload, err := json.Marshal(map[string]any{
		"metadata_sidecars": []map[string]any{
			{"path": "/movies/original.nfo", "parse_status": "parsed", "hints": map[string]any{"title": "Sidecar Title", "original_title": "Sidecar Original", "year": 2026}, "external_ids": map[string]any{"tmdb": "123"}},
			{"path": "/movies/bad.nfo", "parse_status": "failed", "hints": map[string]any{"title": "Ignored"}},
		},
		"image_candidates": []map[string]any{{"image_type": "poster", "path": "/movies/poster.jpg", "source": "scanner", "priority": 1, "provisional": true}},
		"external_ids":     []map[string]any{{"provider": "metatube", "provider_type": "fanza", "external_id": "metatube:fanza:abc"}},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if _, err := catalog.NewService(db).RecordMetadataSource(ctx, catalog.MetadataSourceInput{ItemID: item.ID, SourceType: catalog.SourceTypeLocalFile, SourceName: "scanner", PayloadJSON: string(payload), FetchedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("record scanner source: %v", err)
	}

	evidence, err := NewService(db, config.MetadataConfig{}, nil).loadLocalScannerEvidence(ctx, item.ID)
	if err != nil {
		t.Fatalf("load scanner evidence: %v", err)
	}
	if len(evidence.Sidecars) != 1 || len(evidence.Images) != 1 || len(evidence.ExternalIDs) != 1 {
		t.Fatalf("unexpected scanner evidence: %#v", evidence)
	}
	candidates := localEvidenceCandidates(evidence, "movie")
	if len(candidates) != 2 {
		t.Fatalf("expected sidecar and scanner external-id candidates, got %#v", candidates)
	}
	detail, ok := localEvidenceDetail(evidence, catalog.ItemTypeMovie)
	if !ok || detail.Title != "Sidecar Title" || detail.OriginalTitle != "Sidecar Original" || detail.Year == nil || *detail.Year != 2026 || len(detail.ExternalIDs) != 2 {
		t.Fatalf("unexpected local evidence detail: %#v", detail)
	}
}

func TestLoadLocalScannerEvidenceRejectsMalformedPayload(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Original", Path: "/movies/original.mkv", SortKey: "Original"})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	if _, err := catalog.NewService(db).RecordMetadataSource(ctx, catalog.MetadataSourceInput{ItemID: item.ID, SourceType: catalog.SourceTypeLocalFile, SourceName: "scanner", PayloadJSON: "{malformed", FetchedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("record scanner source: %v", err)
	}
	if _, err := NewService(db, config.MetadataConfig{}, nil).loadLocalScannerEvidence(ctx, item.ID); err == nil {
		t.Fatalf("expected malformed scanner payload error")
	}
}
