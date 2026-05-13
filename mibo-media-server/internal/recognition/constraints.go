package recognition

import (
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

type ConstraintResult struct {
	RejectedCandidates map[string]string
	RequiredOutcomes   map[string]string
}

func (r ConstraintResult) IsRejected(candidateKey string) bool {
	_, ok := r.RejectedCandidates[strings.TrimSpace(candidateKey)]
	return ok
}

func (r ConstraintResult) RequiredOutcome(candidateKey string) string {
	return r.RequiredOutcomes[strings.TrimSpace(candidateKey)]
}

func ApplyHardConstraints(candidates []database.RecognitionCandidate, evidence []database.RecognitionEvidence) ConstraintResult {
	result := ConstraintResult{RejectedCandidates: map[string]string{}, RequiredOutcomes: map[string]string{}}
	evidenceByFile := evidenceKeysByFile(evidence)
	for _, candidate := range candidates {
		if candidate.PrimaryInventoryID == nil {
			continue
		}
		keys := evidenceByFile[*candidate.PrimaryInventoryID]
		if candidate.CandidateType == CandidateTypeWork && candidate.CandidateRole == WorkKindMovie && keys["season_number"] && keys["episode_number"] {
			result.RejectedCandidates[candidate.CandidateKey] = "episode evidence excludes movie work candidate"
		}
		if candidate.CandidateType == CandidateTypePlayableResource && (keys["role:trailer"] || keys["role:sample"] || keys["role:extra"]) {
			result.RequiredOutcomes[candidate.CandidateKey] = DecisionOutcomeReviewRequired
		}
	}
	return result
}

func evidenceKeysByFile(items []database.RecognitionEvidence) map[uint]map[string]bool {
	result := make(map[uint]map[string]bool)
	for _, item := range items {
		if item.InventoryFileID == nil {
			continue
		}
		fileID := *item.InventoryFileID
		if result[fileID] == nil {
			result[fileID] = make(map[string]bool)
		}
		key := strings.TrimSpace(item.EvidenceKey)
		value := strings.ToLower(strings.TrimSpace(item.EvidenceValue))
		result[fileID][key] = true
		if key == "role" && value != "" {
			result[fileID]["role:"+value] = true
		}
	}
	return result
}
