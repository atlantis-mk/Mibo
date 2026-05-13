package recognition

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestInferConsistentCandidateGraphKeepsParentChain(t *testing.T) {
	candidates := []database.RecognitionCandidate{
		{CandidateKey: "work:series:show", CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeries},
		{CandidateKey: "work:season:work:series:show:s01", CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeason, ParentCandidateKey: "work:series:show"},
		{CandidateKey: "episode:work:season:work:series:show:s01:e01", CandidateType: CandidateTypeEpisode, CandidateRole: WorkKindEpisode, ParentCandidateKey: "work:season:work:series:show:s01"},
	}

	result := InferConsistentCandidateGraph(candidates, ConstraintResult{})

	if len(result.AcceptedCandidates) != 3 || !result.Accepted("work:series:show") || !result.Accepted("work:season:work:series:show:s01") || !result.Accepted("episode:work:season:work:series:show:s01:e01") {
		t.Fatalf("expected full parent chain accepted, got %#v", result)
	}
}

func TestInferConsistentCandidateGraphDropsOrphanEpisode(t *testing.T) {
	candidates := []database.RecognitionCandidate{{CandidateKey: "episode:missing:e01", CandidateType: CandidateTypeEpisode, CandidateRole: WorkKindEpisode, ParentCandidateKey: "work:season:missing:s01"}}

	result := InferConsistentCandidateGraph(candidates, ConstraintResult{})

	if result.Accepted("episode:missing:e01") {
		t.Fatalf("expected orphan episode rejected, got %#v", result)
	}
}

func TestInferConsistentCandidateGraphRejectsChildWhenParentConstrained(t *testing.T) {
	candidates := []database.RecognitionCandidate{
		{CandidateKey: "work:series:show", CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeries},
		{CandidateKey: "work:season:work:series:show:s01", CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeason, ParentCandidateKey: "work:series:show"},
	}
	constraints := ConstraintResult{RejectedCandidates: map[string]string{"work:series:show": "test rejection"}, RequiredOutcomes: map[string]string{}}

	result := InferConsistentCandidateGraph(candidates, constraints)

	if result.Accepted("work:series:show") || result.Accepted("work:season:work:series:show:s01") {
		t.Fatalf("expected constrained parent and child rejected, got %#v", result)
	}
}

func TestInferConsistentCandidateGraphRejectsDescendantWhenGrandparentMissing(t *testing.T) {
	candidates := []database.RecognitionCandidate{
		{CandidateKey: "work:season:missing:s01", CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeason, ParentCandidateKey: "work:series:missing"},
		{CandidateKey: "episode:work:season:missing:s01:e01", CandidateType: CandidateTypeEpisode, CandidateRole: WorkKindEpisode, ParentCandidateKey: "work:season:missing:s01"},
	}

	result := InferConsistentCandidateGraph(candidates, ConstraintResult{})

	if result.Accepted("work:season:missing:s01") || result.Accepted("episode:work:season:missing:s01:e01") {
		t.Fatalf("expected missing grandparent to reject parent and descendant, got %#v", result)
	}
}
