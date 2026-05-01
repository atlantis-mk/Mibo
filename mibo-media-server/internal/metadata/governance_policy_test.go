package metadata

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestGovernanceStatusForMetadataOperation(t *testing.T) {
	tests := []struct {
		name       string
		operation  string
		status     string
		confidence float64
		want       string
	}{
		{name: "high confidence match", operation: OperationTypeMatch, status: OperationStatusApplied, confidence: 0.95, want: catalog.GovernanceMatched},
		{name: "low confidence match", operation: OperationTypeMatch, status: OperationStatusApplied, confidence: 0.7, want: catalog.GovernanceNeedsReview},
		{name: "no candidate", operation: OperationTypeMatch, status: OperationStatusNoCandidate, want: catalog.GovernanceUnmatched},
		{name: "manual apply", operation: OperationTypeManualApply, status: OperationStatusApplied, confidence: 1, want: catalog.GovernanceManual},
		{name: "skipped", operation: OperationTypeMatch, status: OperationStatusSkipped, want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := governanceStatusForMetadataOperation(tc.operation, tc.status, tc.confidence); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestApplyMetadataFieldChangesRecordsSourceAttribution(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Original", Path: "/movies/original.mkv", SortKey: "Original"})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	source, err := catalog.NewService(db).RecordMetadataSource(ctx, catalog.MetadataSourceInput{ItemID: item.ID, SourceType: catalog.SourceTypeProvider, SourceName: "tmdb", ExternalID: "movie:1", PayloadJSON: `{}`, FetchedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("record metadata source: %v", err)
	}
	if _, _, err := NewService(db, config.MetadataConfig{}, nil).applyMetadataFieldChanges(ctx, []MetadataFieldChange{{ItemID: item.ID, FieldKey: "overview", Value: "Provider Overview", SourceID: &source.ID, ApplyMode: FieldApplyModeAutomated}}); err != nil {
		t.Fatalf("apply field changes: %v", err)
	}
	var state database.MetadataFieldState
	if err := db.WithContext(ctx).Where("item_id = ? AND field_key = ?", item.ID, "overview").First(&state).Error; err != nil {
		t.Fatalf("load field state: %v", err)
	}
	if state.SourceID == nil || *state.SourceID != source.ID {
		t.Fatalf("expected source attribution %d, got %#v", source.ID, state)
	}
}
