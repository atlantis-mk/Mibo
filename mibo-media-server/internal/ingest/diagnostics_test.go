package ingest

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/workflow"
)

func TestDiagnosticsEnrichesClassificationReviewMessageWithDirectoryReduction(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	svc := NewService(db, workflow.NewService(db))
	library := database.Library{Name: "library", Type: "auto", MediaSourceID: 1, RootPath: "/library", Status: "active"}
	if err := db.WithContext(ctx).Create(&library).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	file := database.InventoryFile{LibraryID: library.ID, MediaSourceID: 1, StorageProvider: "local", StoragePath: "/library/Movie.2024.2160p.mkv", ContentClass: "video", Status: "available"}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ClassificationDecision{LibraryID: library.ID, InventoryFileID: &file.ID, SourcePath: "/library", DecisionType: "directory_reduction", CandidateType: "movie_multi_version", TargetKind: "directory", TargetKey: "/library", Status: "provisional", Reason: "directory reduction grouped sibling files before resolver materialization", EvidenceJSON: `{"review_subtype":"single_work_with_noise"}`}).Error; err != nil {
		t.Fatalf("create directory reduction decision: %v", err)
	}
	condition := database.IngestCondition{UnitKey: "inventory_file:1", LibraryID: library.ID, InventoryFileID: &file.ID, ConditionType: ConditionReviewRequired, Status: ConditionStatusReviewRequired, Reason: "classification_needs_review", Message: "Classification requires review", Severity: SeverityWarning}
	if err := db.WithContext(ctx).Create(&condition).Error; err != nil {
		t.Fatalf("create condition: %v", err)
	}
	result, err := svc.Diagnostics(ctx, DiagnosticsInput{Status: ConditionStatusReviewRequired, Limit: 10})
	if err != nil {
		t.Fatalf("diagnostics: %v", err)
	}
	if len(result.Stages) != 1 {
		t.Fatalf("expected one diagnostic stage, got %#v", result.Stages)
	}
	message := result.Stages[0].Message
	if !strings.Contains(message, "directory reduction: movie_multi_version") {
		t.Fatalf("expected enriched message, got %q", message)
	}
	if !strings.Contains(message, "subtype: single_work_with_noise") {
		t.Fatalf("expected subtype in enriched message, got %q", message)
	}
}
