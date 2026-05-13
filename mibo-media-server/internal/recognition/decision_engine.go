package recognition

import "github.com/atlan/mibo-media-server/internal/database"

const recognitionAcceptThreshold = 0.75

func BuildKernelDecisions(manifest database.RecognitionManifest, inference InferenceResult, constraints ConstraintResult, scores map[string]float64) ResolveResult {
	result := ResolveResult{}
	for _, candidate := range inference.AcceptedCandidates {
		outcome := constraints.RequiredOutcome(candidate.CandidateKey)
		reason := "candidate selected by recognition kernel"
		if outcome == "" {
			if scores[candidate.CandidateKey] >= recognitionAcceptThreshold {
				outcome = DecisionOutcomeAccepted
				reason = "candidate passed hard constraints, graph inference, and score threshold"
			} else {
				outcome = DecisionOutcomeReviewRequired
				reason = "candidate passed hard constraints but score requires review"
			}
		}
		candidateID := candidate.ID
		result.Decisions = append(result.Decisions, database.RecognitionDecision{ManifestID: manifest.ID, CandidateID: &candidateID, DecisionType: "kernel_decision", Outcome: outcome, TargetKind: candidate.CandidateType, TargetKey: candidate.CandidateKey, TargetMetadataID: candidate.TargetMetadataID, TargetResourceID: candidate.TargetResourceID, Confidence: floatPtr(scores[candidate.CandidateKey]), Reason: reason, EvidenceJSON: candidate.EvidenceJSON})
	}
	for key, reason := range inference.RejectedReasons {
		result.Decisions = append(result.Decisions, database.RecognitionDecision{ManifestID: manifest.ID, DecisionType: "kernel_rejection", Outcome: DecisionOutcomeRejected, TargetKey: key, Reason: reason})
	}
	return result
}

func floatPtr(v float64) *float64 { return &v }
