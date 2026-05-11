package recognition

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestLoadGovernanceReviewGroupsReturnsReviewDecisions(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	repo := NewRepository(db)
	manifest, err := repo.UpsertManifest(ctx, ManifestScope{ManifestKey: "manifest:review", LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test", Fingerprint: "fp"})
	if err != nil {
		t.Fatalf("upsert manifest: %v", err)
	}
	candidate := database.RecognitionCandidate{ManifestID: manifest.ID, CandidateKey: "candidate", CandidateType: CandidateTypePlayableResource}
	if err := repo.SaveCandidates(ctx, []database.RecognitionCandidate{candidate}); err != nil {
		t.Fatalf("save candidate: %v", err)
	}
	if err := repo.SaveDecisions(ctx, []database.RecognitionDecision{{ManifestID: manifest.ID, DecisionType: "resolver_outcome", Outcome: DecisionOutcomeReviewRequired, TargetKind: CandidateTypePlayableResource, TargetKey: "candidate"}}); err != nil {
		t.Fatalf("save decision: %v", err)
	}
	groups, err := repo.LoadGovernanceReviewGroups(ctx, 1)
	if err != nil {
		t.Fatalf("load groups: %v", err)
	}
	if len(groups) != 1 || len(groups[0].Decisions) != 1 {
		t.Fatalf("expected review group, got %#v", groups)
	}
}

func TestApplyCorrectionRuleWritesResolverRule(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	repo := NewRepository(db)
	rule, err := repo.ApplyCorrectionRule(ctx, CorrectionRuleInput{LibraryID: 1, StorageProvider: "local", ScopePath: "/library/Movie", CandidateType: CandidateTypeWork, Action: RuleActionSplit, PayloadJSON: `{"candidate_key":"work:movie"}`})
	if err != nil {
		t.Fatalf("apply correction: %v", err)
	}
	if rule.ID == 0 || rule.Action != RuleActionSplit || !rule.Enabled {
		t.Fatalf("unexpected rule %#v", rule)
	}
}

func TestManualSplitRuleOverridesAutomaticAcceptance(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	repo := NewRepository(db)
	candidateKey := "work:movie:movie:2024"
	_, err = repo.ApplyCorrectionRule(ctx, CorrectionRuleInput{LibraryID: 1, StorageProvider: "local", ScopePath: "/library/Movie", CandidateType: CandidateTypeWork, Action: RuleActionSplit, PayloadJSON: candidateKey})
	if err != nil {
		t.Fatalf("apply split rule: %v", err)
	}
	rules, err := repo.LoadEnabledRules(ctx, 1, "local", "/library/Movie/File.mkv")
	if err != nil {
		t.Fatalf("load rules: %v", err)
	}
	fileID := uint(1)
	candidate := database.RecognitionCandidate{ID: 1, CandidateKey: candidateKey, CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, CanonicalKey: candidateKey, PrimaryInventoryID: &fileID}
	evidence := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceKey: "title", EvidenceValue: "Movie"}, {InventoryFileID: &fileID, EvidenceKey: "year", EvidenceValue: "2024"}}
	result := NewResolver(rules).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}, Evidence: evidence})
	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeRejected {
		t.Fatalf("expected manual split to override automatic acceptance, got %#v", result.Decisions)
	}
}
