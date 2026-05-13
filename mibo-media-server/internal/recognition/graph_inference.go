package recognition

import "github.com/atlan/mibo-media-server/internal/database"

type InferenceResult struct {
	AcceptedCandidates []database.RecognitionCandidate
	RejectedReasons    map[string]string
}

func (r InferenceResult) Accepted(candidateKey string) bool {
	for _, candidate := range r.AcceptedCandidates {
		if candidate.CandidateKey == candidateKey {
			return true
		}
	}
	return false
}

func InferConsistentCandidateGraph(candidates []database.RecognitionCandidate, constraints ConstraintResult) InferenceResult {
	result := InferenceResult{RejectedReasons: map[string]string{}}
	byKey := make(map[string]database.RecognitionCandidate, len(candidates))
	for _, candidate := range candidates {
		byKey[candidate.CandidateKey] = candidate
	}
	valid := make(map[string]bool, len(candidates))
	visiting := make(map[string]bool, len(candidates))
	var candidateValid func(string) bool
	candidateValid = func(candidateKey string) bool {
		if accepted, ok := valid[candidateKey]; ok {
			return accepted
		}
		candidate, ok := byKey[candidateKey]
		if !ok {
			return false
		}
		if constraints.IsRejected(candidate.CandidateKey) {
			result.RejectedReasons[candidate.CandidateKey] = constraints.RejectedCandidates[candidate.CandidateKey]
			valid[candidate.CandidateKey] = false
			return false
		}
		if candidate.ParentCandidateKey != "" {
			if visiting[candidate.CandidateKey] {
				result.RejectedReasons[candidate.CandidateKey] = "cyclic parent candidate"
				valid[candidate.CandidateKey] = false
				return false
			}
			visiting[candidate.CandidateKey] = true
			if !candidateValid(candidate.ParentCandidateKey) {
				if result.RejectedReasons[candidate.CandidateKey] == "" {
					result.RejectedReasons[candidate.CandidateKey] = "missing parent candidate"
				}
				valid[candidate.CandidateKey] = false
				visiting[candidate.CandidateKey] = false
				return false
			}
			visiting[candidate.CandidateKey] = false
		}
		valid[candidate.CandidateKey] = true
		return true
	}
	for _, candidate := range candidates {
		if candidateValid(candidate.CandidateKey) {
			result.AcceptedCandidates = append(result.AcceptedCandidates, candidate)
		}
	}
	return result
}
