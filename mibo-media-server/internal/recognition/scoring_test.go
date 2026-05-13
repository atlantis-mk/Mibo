package recognition

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestScoreCandidatesUsesEvidenceAfterConstraints(t *testing.T) {
	fileID := uint(1)
	candidates := []database.RecognitionCandidate{{CandidateKey: "work:movie:movie-a:2024", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, PrimaryInventoryID: &fileID}}
	evidence := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceKey: "title", Strength: "medium"}, {InventoryFileID: &fileID, EvidenceKey: "year", Strength: "medium"}, {InventoryFileID: &fileID, EvidenceKey: "folder_shape", Strength: "strong"}}

	scores := ScoreCandidates(candidates, evidence)

	if scores["work:movie:movie-a:2024"] < 0.70 {
		t.Fatalf("expected confident movie score, got %#v", scores)
	}
}
