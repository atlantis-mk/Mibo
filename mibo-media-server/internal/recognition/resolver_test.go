package recognition

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestResolverAppliesHighestPriorityRule(t *testing.T) {
	candidate := database.RecognitionCandidate{ID: 10, CandidateKey: "work:movie:movie:2024", CandidateType: CandidateTypeWork, CanonicalKey: "work:movie:movie:2024"}
	rules := []database.RecognitionRule{
		{ID: 2, RuleKey: "accept", CandidateType: CandidateTypeWork, Action: RuleActionAccept, Priority: 20, Enabled: true, PayloadJSON: candidate.CandidateKey},
		{ID: 1, RuleKey: "split", CandidateType: CandidateTypeWork, Action: RuleActionSplit, Priority: 10, Enabled: true, PayloadJSON: candidate.CandidateKey},
	}
	resolver := NewResolver(rules)
	result := resolver.Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}})
	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeRejected {
		t.Fatalf("expected split rule to reject candidate, got %#v", result.Decisions)
	}
	if len(result.Conflicts) != 1 || result.Conflicts[0].ConflictType != "manual_rule" {
		t.Fatalf("expected manual conflict, got %#v", result.Conflicts)
	}
}

func TestResolverIgnoresNonMatchingRulePayload(t *testing.T) {
	candidate := database.RecognitionCandidate{ID: 10, CandidateKey: "work:movie:movie:2024", CandidateType: CandidateTypeWork}
	rule := database.RecognitionRule{ID: 1, RuleKey: "other", CandidateType: CandidateTypeWork, Action: RuleActionAccept, Priority: 1, Enabled: true, PayloadJSON: "work:movie:other:2024"}
	resolver := NewResolver([]database.RecognitionRule{rule})
	result := resolver.Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}})
	if len(result.Decisions) != 1 || result.Decisions[0].DecisionType == "resolver_rule" {
		t.Fatalf("expected no matching rule decision, got %#v", result.Decisions)
	}
}

func TestResolverAcceptsMovieWorkByTitleYearGate(t *testing.T) {
	fileID := uint(7)
	candidate := database.RecognitionCandidate{ID: 10, CandidateKey: "work:movie:movie:2024", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, CanonicalKey: "work:movie:movie:2024", PrimaryInventoryID: &fileID}
	evidence := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceKey: "title", EvidenceValue: "Movie"}, {InventoryFileID: &fileID, EvidenceKey: "year", EvidenceValue: "2024"}}
	resolver := NewResolver(nil)
	result := resolver.Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}, Evidence: evidence})
	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeAccepted || result.Decisions[0].DecisionType != "resolver_gate" {
		t.Fatalf("expected accepted gate decision, got %#v", result.Decisions)
	}
}

func TestResolverAcceptsHighConfidenceMovieWorkByTitleGate(t *testing.T) {
	fileID := uint(7)
	confidence := 0.75
	candidate := database.RecognitionCandidate{ID: 10, CandidateKey: "work:movie:movie", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, CanonicalKey: "work:movie:movie", PrimaryInventoryID: &fileID, Confidence: &confidence}
	evidence := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceKey: "title", EvidenceValue: "Movie"}}
	resolver := NewResolver(nil)
	result := resolver.Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}, Evidence: evidence})
	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeAccepted || result.Decisions[0].DecisionType != "resolver_gate" {
		t.Fatalf("expected accepted title gate decision, got %#v", result.Decisions)
	}
}

func TestResolverAcceptsEpisodeByTupleGate(t *testing.T) {
	fileID := uint(7)
	candidate := database.RecognitionCandidate{ID: 10, CandidateKey: "episode:show:s01:e02", CandidateType: CandidateTypeEpisode, CanonicalKey: "episode:show:s01:e02", PrimaryInventoryID: &fileID}
	evidence := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceKey: "season_number", EvidenceValue: "1"}, {InventoryFileID: &fileID, EvidenceKey: "episode_number", EvidenceValue: "2"}}
	result := NewResolver(nil).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}, Evidence: evidence})
	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeAccepted {
		t.Fatalf("expected accepted episode gate, got %#v", result.Decisions)
	}
}

func TestResolverAcceptsSeriesAndSeasonByDirectoryContextGates(t *testing.T) {
	fileID := uint(7)
	series := database.RecognitionCandidate{ID: 10, CandidateKey: "work:series:show", CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeries, CanonicalKey: "work:series:show", PrimaryInventoryID: &fileID}
	season := database.RecognitionCandidate{ID: 11, CandidateKey: "work:season:work:series:show:s01", CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeason, CanonicalKey: "work:season:work:series:show:s01", ParentCandidateKey: "work:series:show", PrimaryInventoryID: &fileID}
	evidence := []database.RecognitionEvidence{
		{InventoryFileID: &fileID, EvidenceSource: "content_shape", EvidenceKey: "series_title", EvidenceValue: "Show"},
		{InventoryFileID: &fileID, EvidenceSource: "content_shape", EvidenceKey: "season_number", EvidenceValue: "1"},
	}
	result := NewResolver(nil).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{series, season}, Evidence: evidence})
	if len(result.Decisions) != 2 {
		t.Fatalf("expected series and season decisions, got %#v", result.Decisions)
	}
	for _, decision := range result.Decisions {
		if decision.Outcome != DecisionOutcomeAccepted {
			t.Fatalf("expected accepted series/season gate, got %#v", result.Decisions)
		}
	}
}

func TestResolverAcceptsMovieWorkByDirectoryReductionGate(t *testing.T) {
	fileID := uint(7)
	candidate := database.RecognitionCandidate{ID: 10, CandidateKey: "work:movie:movie:2024", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, CanonicalKey: "work:movie:movie:2024", PrimaryInventoryID: &fileID}
	evidence := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceSource: "directory_reduction", EvidenceKey: "assignment", EvidenceValue: "movie_multi_version"}}
	result := NewResolver(nil).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}, Evidence: evidence})
	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeAccepted {
		t.Fatalf("expected accepted movie by directory reduction, got %#v", result.Decisions)
	}
}

func TestResolverAcceptsEpisodeByDirectoryReductionGate(t *testing.T) {
	fileID := uint(7)
	candidate := database.RecognitionCandidate{ID: 10, CandidateKey: "episode:show:s01:e02", CandidateType: CandidateTypeEpisode, CanonicalKey: "episode:show:s01:e02", PrimaryInventoryID: &fileID}
	evidence := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceSource: "directory_reduction", EvidenceKey: "assignment", EvidenceValue: "episode_multi_version"}}
	result := NewResolver(nil).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}, Evidence: evidence})
	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeAccepted {
		t.Fatalf("expected accepted episode by directory reduction, got %#v", result.Decisions)
	}
}

func TestResolverAcceptsMovieWorkBySingleWorkIdentityGate(t *testing.T) {
	fileID := uint(7)
	candidate := database.RecognitionCandidate{ID: 10, CandidateKey: "work:movie:movie:2024", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, CanonicalKey: "work:movie:movie:2024", PrimaryInventoryID: &fileID}
	evidence := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceSource: "directory_reduction", EvidenceKey: "assignment", EvidenceValue: "single_work_identity"}}
	result := NewResolver(nil).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}, Evidence: evidence})
	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeAccepted {
		t.Fatalf("expected accepted movie by single work identity, got %#v", result.Decisions)
	}
}

func TestResolverAcceptsMovieWorkByMovieCollectionGate(t *testing.T) {
	fileID := uint(7)
	candidate := database.RecognitionCandidate{ID: 10, CandidateKey: "work:movie:movie", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, CanonicalKey: "work:movie:movie", PrimaryInventoryID: &fileID}
	evidence := []database.RecognitionEvidence{
		{InventoryFileID: &fileID, EvidenceKey: "title", EvidenceValue: "Movie"},
		{InventoryFileID: &fileID, EvidenceKey: "episode_number", EvidenceValue: "1"},
		{InventoryFileID: &fileID, EvidenceKey: "episode_source", EvidenceValue: "leading_numeric"},
		{InventoryFileID: &fileID, EvidenceSource: "directory_reduction", EvidenceKey: "assignment", EvidenceValue: "movie_collection"},
	}
	result := NewResolver(nil).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}, Evidence: evidence})
	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeAccepted {
		t.Fatalf("expected accepted movie by movie collection gate, got %#v", result.Decisions)
	}
}

func TestResolverRejectsMovieCollectionGateWithSeasonEvidence(t *testing.T) {
	fileID := uint(7)
	candidate := database.RecognitionCandidate{ID: 10, CandidateKey: "work:movie:show", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, CanonicalKey: "work:movie:show", PrimaryInventoryID: &fileID}
	evidence := []database.RecognitionEvidence{
		{InventoryFileID: &fileID, EvidenceKey: "title", EvidenceValue: "Show"},
		{InventoryFileID: &fileID, EvidenceKey: "season_number", EvidenceValue: "1"},
		{InventoryFileID: &fileID, EvidenceKey: "episode_number", EvidenceValue: "2"},
		{InventoryFileID: &fileID, EvidenceSource: "directory_reduction", EvidenceKey: "assignment", EvidenceValue: "movie_collection"},
	}
	result := NewResolver(nil).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}, Evidence: evidence})
	if len(result.Decisions) != 1 || result.Decisions[0].Outcome == DecisionOutcomeAccepted {
		t.Fatalf("expected movie collection gate to reject season evidence, got %#v", result.Decisions)
	}
}

func TestResolverAcceptsEpisodeBySingleEpisodeIdentityGate(t *testing.T) {
	fileID := uint(7)
	candidate := database.RecognitionCandidate{ID: 10, CandidateKey: "episode:show:s01:e02", CandidateType: CandidateTypeEpisode, CanonicalKey: "episode:show:s01:e02", PrimaryInventoryID: &fileID}
	evidence := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceSource: "directory_reduction", EvidenceKey: "assignment", EvidenceValue: "single_episode_identity"}}
	result := NewResolver(nil).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}, Evidence: evidence})
	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeAccepted {
		t.Fatalf("expected accepted episode by single episode identity, got %#v", result.Decisions)
	}
}

func TestResolverBlocksConflictingExternalIDs(t *testing.T) {
	fileID := uint(7)
	candidate := database.RecognitionCandidate{ID: 10, CandidateKey: "work:movie:movie:2024", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, CanonicalKey: "work:movie:movie:2024", PrimaryInventoryID: &fileID}
	evidence := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceKey: "external_id:tmdb", EvidenceValue: "1"}, {InventoryFileID: &fileID, EvidenceKey: "external_id:tmdb", EvidenceValue: "2"}}
	result := NewResolver(nil).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}, Evidence: evidence})
	if len(result.Conflicts) != 1 || result.Conflicts[0].ConflictType != "external_identity_conflict" {
		t.Fatalf("expected external identity conflict, got %#v", result.Conflicts)
	}
	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeBlockedConflict {
		t.Fatalf("expected blocked conflict decision, got %#v", result.Decisions)
	}
}

func TestResolverProducesFallbackOutcomes(t *testing.T) {
	provisionalConfidence := 0.6
	provisionalReviewConfidence := 0.2
	reviewConfidence := 0.1
	candidates := []database.RecognitionCandidate{
		{ID: 1, CandidateKey: "resource:provisional", CandidateType: CandidateTypePlayableResource, Confidence: &provisionalConfidence},
		{ID: 2, CandidateKey: "resource:still-provisional", CandidateType: CandidateTypePlayableResource, Confidence: &provisionalReviewConfidence},
		{ID: 3, CandidateKey: "resource:review", CandidateType: CandidateTypePlayableResource, Confidence: &reviewConfidence},
		{ID: 4, CandidateKey: "resource:unmatched", CandidateType: CandidateTypePlayableResource},
	}
	result := NewResolver(nil).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: candidates})
	if len(result.Decisions) != 4 {
		t.Fatalf("expected fallback decisions, got %#v", result.Decisions)
	}
	want := map[string]string{"resource:provisional": DecisionOutcomeProvisional, "resource:still-provisional": DecisionOutcomeProvisional, "resource:review": DecisionOutcomeReviewRequired, "resource:unmatched": DecisionOutcomeUnmatched}
	for _, decision := range result.Decisions {
		if decision.Outcome != want[decision.TargetKey] {
			t.Fatalf("unexpected outcome for %s: got %s want %s", decision.TargetKey, decision.Outcome, want[decision.TargetKey])
		}
	}
}

func TestResolverAcceptsDuplicateBinaryCandidate(t *testing.T) {
	candidate := database.RecognitionCandidate{ID: 1, CandidateKey: "duplicate_binary:md5:same", CandidateType: CandidateTypeDuplicateBinary}
	result := NewResolver(nil).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}})
	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeAccepted {
		t.Fatalf("expected duplicate binary accepted, got %#v", result.Decisions)
	}
}

func TestResolverAcceptsVariantAndEditionCandidates(t *testing.T) {
	candidates := []database.RecognitionCandidate{
		{ID: 1, CandidateKey: "variant:2160p:work", CandidateType: CandidateTypeVariant, ParentCandidateKey: "work:movie:movie:2024", VariantKey: "variant:2160p"},
		{ID: 2, CandidateKey: "edition:directors-cut:work", CandidateType: CandidateTypeEdition, ParentCandidateKey: "work:movie:movie:2024", EditionKey: "edition:directors-cut"},
	}
	result := NewResolver(nil).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: candidates})
	if len(result.Decisions) != 2 {
		t.Fatalf("expected two decisions, got %#v", result.Decisions)
	}
	for _, decision := range result.Decisions {
		if decision.Outcome != DecisionOutcomeAccepted {
			t.Fatalf("expected accepted variant/edition decision, got %#v", decision)
		}
	}
}

func TestResolverAcceptsPlayableResourceByDirectoryReductionGate(t *testing.T) {
	fileID := uint(7)
	candidate := database.RecognitionCandidate{ID: 1, CandidateKey: "playable_resource:local:path:/library/Movie.2160p.mkv", CandidateType: CandidateTypePlayableResource, ParentCandidateKey: "work:movie:movie:2024", PrimaryInventoryID: &fileID}
	evidence := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceSource: "directory_reduction", EvidenceKey: candidate.CandidateKey, EvidenceValue: "movie_multi_version"}}
	result := NewResolver(nil).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}, Evidence: evidence})
	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeAccepted {
		t.Fatalf("expected accepted resource by directory reduction, got %#v", result.Decisions)
	}
}

func TestResolverLeavesSupplementalForReviewWithoutParentGate(t *testing.T) {
	candidate := database.RecognitionCandidate{ID: 1, CandidateKey: "supplemental:trailer:resource", CandidateType: CandidateTypeSupplemental, CandidateRole: "trailer"}
	result := NewResolver(nil).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}})
	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeUnmatched {
		t.Fatalf("expected unmatched supplemental without parent evidence, got %#v", result.Decisions)
	}
}

func TestResolverAcceptsSupplementalWithParentGate(t *testing.T) {
	fileID := uint(7)
	candidate := database.RecognitionCandidate{ID: 1, CandidateKey: "supplemental:trailer:resource", CandidateType: CandidateTypeSupplemental, CandidateRole: "trailer", ParentCandidateKey: "work:movie:movie:2024", PrimaryInventoryID: &fileID}
	result := NewResolver(nil).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}})
	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeAccepted {
		t.Fatalf("expected accepted supplemental with parent evidence, got %#v", result.Decisions)
	}
}
