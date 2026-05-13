package recognition

import "github.com/atlan/mibo-media-server/internal/database"

func ScoreCandidates(candidates []database.RecognitionCandidate, evidence []database.RecognitionEvidence) map[string]float64 {
	evidenceByFile := evidenceKeysByFile(evidence)
	scores := make(map[string]float64, len(candidates))
	for _, candidate := range candidates {
		score := 0.25
		if candidate.PrimaryInventoryID != nil {
			keys := evidenceByFile[*candidate.PrimaryInventoryID]
			if keys["title"] {
				score += 0.20
			}
			if keys["year"] {
				score += 0.15
			}
			if keys["season_number"] {
				score += 0.15
			}
			if keys["episode_number"] {
				score += 0.15
			}
			if keys["folder_shape"] {
				score += 0.20
			}
		}
		if score > 1.0 {
			score = 1.0
		}
		scores[candidate.CandidateKey] = score
	}
	return scores
}
