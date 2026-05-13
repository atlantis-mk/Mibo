package recognition

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestBuildKernelDecisionsAcceptsHighConfidenceCandidate(t *testing.T) {
	candidate := database.RecognitionCandidate{ID: 1, CandidateKey: "work:movie:movie-a:2024", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie}
	result := BuildKernelDecisions(database.RecognitionManifest{ID: 10}, InferenceResult{AcceptedCandidates: []database.RecognitionCandidate{candidate}}, ConstraintResult{}, map[string]float64{candidate.CandidateKey: 0.85})

	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeAccepted || result.Decisions[0].Reason == "" {
		t.Fatalf("expected accepted explained decision, got %#v", result)
	}
}

func TestBuildKernelDecisionsSendsCloseOrLowConfidenceToReview(t *testing.T) {
	candidate := database.RecognitionCandidate{ID: 1, CandidateKey: "work:movie:ambiguous:0", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie}
	result := BuildKernelDecisions(database.RecognitionManifest{ID: 10}, InferenceResult{AcceptedCandidates: []database.RecognitionCandidate{candidate}}, ConstraintResult{}, map[string]float64{candidate.CandidateKey: 0.55})

	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeReviewRequired {
		t.Fatalf("expected review_required decision, got %#v", result)
	}
}
