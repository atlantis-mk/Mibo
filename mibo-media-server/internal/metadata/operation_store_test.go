package metadata

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestRecordMetadataOperationPersistsAttemptEvidence(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db, config.MetadataConfig{}, nil)
	result := MetadataOperationResult{
		Operation:        OperationTypeMatch,
		OriginItemID:     1,
		TargetItemID:     1,
		TargetType:       "movie",
		Status:           OperationStatusApplied,
		GovernanceStatus: StatusMatched,
		Plan: MetadataExecutionPlanSummary{LibraryID: 7, SearchProviders: []MetadataPlanProviderSummary{{ID: 10, Name: "primary", ProviderType: database.MetadataProviderTypeTMDB, Operational: true}}},
		ProviderAttempts: []MetadataProviderAttempt{
			{Stage: "search", ProviderInstanceID: 10, ProviderInstanceName: "primary", ProviderType: database.MetadataProviderTypeTMDB, Outcome: ProviderAttemptOutcomeSuccess, CandidateCount: 1, Selected: true},
			{Stage: "search", ProviderInstanceID: 11, ProviderInstanceName: "empty", ProviderType: database.MetadataProviderTypeTMDB, Outcome: ProviderAttemptOutcomeNoResult, CandidateCount: 0},
			{Stage: "detail", ProviderInstanceID: 12, ProviderInstanceName: "cooldown", ProviderType: database.MetadataProviderTypeTMDB, Outcome: ProviderAttemptOutcomeSkippedUnavailable, ErrorClass: "cooldown"},
			{Stage: "detail", ProviderInstanceID: 13, ProviderInstanceName: "bad-auth", ProviderType: database.MetadataProviderTypeTMDB, Outcome: ProviderAttemptOutcomeFailedTerminal, ErrorClass: "auth", StatusCode: 401, ErrorMessage: "unauthorized"},
		},
		MetadataSourceIDs: []uint{101, 102},
		AppliedFields:    []MetadataAppliedField{{ItemID: 1, FieldKey: "title", ApplyMode: FieldApplyModeAutomated}},
		SkippedFields:    []MetadataSkippedField{{ItemID: 1, FieldKey: "overview", Reason: "locked"}},
		Warnings:         []MetadataOperationWarning{{Code: "fallback", Message: "used fallback"}},
	}

	record, err := svc.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: 7, SelectedCandidate: NormalizedMetadataCandidate{Provider: "tmdb", ExternalID: "movie:1"}, StartedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("record metadata operation: %v", err)
	}

	var stored database.MetadataOperation
	if err := db.WithContext(ctx).First(&stored, record.ID).Error; err != nil {
		t.Fatalf("load operation evidence: %v", err)
	}
	for name, value := range map[string]string{
		"plan":                stored.PlanJSON,
		"attempts":            stored.AttemptsJSON,
		"selected candidate":  stored.SelectedCandidateJSON,
		"metadata source ids": stored.MetadataSourceIDsJSON,
		"applied fields":      stored.AppliedFieldsJSON,
		"skipped fields":      stored.SkippedFieldsJSON,
		"warnings":            stored.WarningsJSON,
	} {
		if value == "" {
			t.Fatalf("expected %s evidence to be persisted: %#v", name, stored)
		}
	}
	if stored.Operation != OperationTypeMatch || stored.Status != OperationStatusApplied || stored.GovernanceStatus != StatusMatched || stored.LibraryID != 7 {
		t.Fatalf("unexpected operation evidence metadata: %#v", stored)
	}
}
