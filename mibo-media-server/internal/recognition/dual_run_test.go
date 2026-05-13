package recognition

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestCompareRecognitionOutputsReportsDecisionDifferences(t *testing.T) {
	oldOutput := ResolveResult{Decisions: []database.RecognitionDecision{{TargetKey: "work:movie:a", Outcome: DecisionOutcomeReviewRequired}}}
	newOutput := ResolveResult{Decisions: []database.RecognitionDecision{{TargetKey: "work:movie:a", Outcome: DecisionOutcomeAccepted}}}

	diff := CompareRecognitionOutputs(oldOutput, newOutput)

	if len(diff.DecisionDiffs) != 1 || diff.DecisionDiffs[0].TargetKey != "work:movie:a" {
		t.Fatalf("expected decision diff, got %#v", diff)
	}
}

func TestCompareRecognitionOutputsReportsAddedRemovedAndSortedDecisionDiffs(t *testing.T) {
	oldOutput := ResolveResult{Decisions: []database.RecognitionDecision{
		{TargetKey: "work:movie:c", Outcome: DecisionOutcomeAccepted},
		{TargetKey: "work:movie:a", Outcome: DecisionOutcomeReviewRequired},
	}}
	newOutput := ResolveResult{Decisions: []database.RecognitionDecision{
		{TargetKey: "work:movie:b", Outcome: DecisionOutcomeAccepted},
		{TargetKey: "work:movie:a", Outcome: DecisionOutcomeReviewRequired},
	}}

	diff := CompareRecognitionOutputs(oldOutput, newOutput)

	if len(diff.DecisionDiffs) != 2 {
		t.Fatalf("expected added and removed diffs, got %#v", diff)
	}
	if diff.DecisionDiffs[0] != (RecognitionDecisionDiff{TargetKey: "work:movie:b", OldOutcome: "", NewOutcome: DecisionOutcomeAccepted}) {
		t.Fatalf("expected sorted added diff first, got %#v", diff.DecisionDiffs)
	}
	if diff.DecisionDiffs[1] != (RecognitionDecisionDiff{TargetKey: "work:movie:c", OldOutcome: DecisionOutcomeAccepted, NewOutcome: ""}) {
		t.Fatalf("expected sorted removed diff second, got %#v", diff.DecisionDiffs)
	}
}

func TestCompareRecognitionOutputsIgnoresUnchangedDecisions(t *testing.T) {
	oldOutput := ResolveResult{Decisions: []database.RecognitionDecision{{TargetKey: "work:movie:a", Outcome: DecisionOutcomeAccepted}}}
	newOutput := ResolveResult{Decisions: []database.RecognitionDecision{{TargetKey: "work:movie:a", Outcome: DecisionOutcomeAccepted}}}

	diff := CompareRecognitionOutputs(oldOutput, newOutput)

	if len(diff.DecisionDiffs) != 0 {
		t.Fatalf("expected no diffs, got %#v", diff)
	}
}
