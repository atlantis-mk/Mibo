package library

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestContentShapeReviewDecisionAndScopedCorrectionRule(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	scope := testContentShapeScope(ContentShapeClassifierVersion, "/library/Ambiguous")
	plan := contentShapeDirectoryPlan{Shape: contentShapeUnknownReview, Confidence: 0.51, ReviewState: "review_required", Evidence: map[string]any{"source": "directory_profile"}, Alternatives: []contentShapePlanAlternative{{Shape: contentShapeAbsoluteEpisodePack, Confidence: 0.5}, {Shape: contentShapeMovieCollection, Confidence: 0.49}}}
	assignments := []contentShapeFileAssignment{{StoragePath: "/library/Ambiguous/001.mkv", AssignmentType: contentShapeAssignmentReview, ReviewState: "review_required"}}
	if err := saveContentShapeReviewDecision(ctx, db, scope, plan, assignments); err != nil {
		t.Fatalf("save review decision: %v", err)
	}
	var decision database.ClassificationDecision
	if err := db.WithContext(ctx).Where("library_id = ? AND decision_type = ?", scope.LibraryID, "content_shape_plan").First(&decision).Error; err != nil {
		t.Fatalf("load review decision: %v", err)
	}
	if decision.Status != "provisional" || decision.AffectedFilesJSON == "" || decision.AlternativesJSON == "" {
		t.Fatalf("expected provisional decision with evidence, got %#v", decision)
	}
	if err := saveContentShapeCorrectionRule(ctx, db, scope, "Confirm absolute pack", contentShapeAbsoluteEpisodePack, map[string]any{"source": "user_scoped_rule"}, nil); err != nil {
		t.Fatalf("save correction rule: %v", err)
	}
	var rule database.ClassificationRule
	if err := db.WithContext(ctx).Where("library_id = ? AND rule_type = ?", scope.LibraryID, "content_shape_directory").First(&rule).Error; err != nil {
		t.Fatalf("load correction rule: %v", err)
	}
	if rule.PathPattern != scope.DirectoryPath || rule.CandidateType != contentShapeAbsoluteEpisodePack || !rule.Enabled {
		t.Fatalf("expected scoped correction rule, got %#v", rule)
	}
}
