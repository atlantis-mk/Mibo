package recognition

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestApplyHardConstraintsRejectsMovieCandidateWhenEpisodeEvidenceExists(t *testing.T) {
	fileID := uint(1)
	candidates := []database.RecognitionCandidate{
		{CandidateKey: "work:movie:show:0", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, PrimaryInventoryID: &fileID},
		{CandidateKey: "episode:work:season:work:series:show:s01:e02", CandidateType: CandidateTypeEpisode, CandidateRole: WorkKindEpisode, PrimaryInventoryID: &fileID},
	}
	evidence := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceKey: "season_number", EvidenceValue: "1"}, {InventoryFileID: &fileID, EvidenceKey: "episode_number", EvidenceValue: "2"}}

	result := ApplyHardConstraints(candidates, evidence)

	if !result.IsRejected("work:movie:show:0") || result.IsRejected("episode:work:season:work:series:show:s01:e02") {
		t.Fatalf("expected movie rejected and episode kept, got %#v", result)
	}
}

func TestApplyHardConstraintsDemotesExtraResourceToReview(t *testing.T) {
	fileID := uint(2)
	candidates := []database.RecognitionCandidate{{CandidateKey: "playable_resource:local:path:/library/Movie/extras/Trailer.mkv", CandidateType: CandidateTypePlayableResource, PrimaryInventoryID: &fileID}}
	evidence := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceKey: "role", EvidenceValue: "trailer"}}

	result := ApplyHardConstraints(candidates, evidence)

	if result.RequiredOutcome("playable_resource:local:path:/library/Movie/extras/Trailer.mkv") != DecisionOutcomeReviewRequired {
		t.Fatalf("expected trailer resource review_required, got %#v", result)
	}
}
