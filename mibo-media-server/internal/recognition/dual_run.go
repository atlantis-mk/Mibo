package recognition

import (
	"sort"

	"github.com/atlan/mibo-media-server/internal/database"
)

type RecognitionOutputDiff struct {
	DecisionDiffs []RecognitionDecisionDiff
}

type RecognitionDecisionDiff struct {
	TargetKey  string
	OldOutcome string
	NewOutcome string
}

func CompareRecognitionOutputs(oldOutput ResolveResult, newOutput ResolveResult) RecognitionOutputDiff {
	oldByKey := decisionsByTargetKey(oldOutput.Decisions)
	newByKey := decisionsByTargetKey(newOutput.Decisions)
	keys := make([]string, 0, len(oldByKey)+len(newByKey))
	seen := make(map[string]struct{}, len(oldByKey)+len(newByKey))
	for key := range oldByKey {
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	for key := range newByKey {
		if _, ok := seen[key]; ok {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	diff := RecognitionOutputDiff{}
	for _, key := range keys {
		oldDecision := oldByKey[key]
		newDecision := newByKey[key]
		if oldDecision.Outcome != newDecision.Outcome {
			diff.DecisionDiffs = append(diff.DecisionDiffs, RecognitionDecisionDiff{TargetKey: key, OldOutcome: oldDecision.Outcome, NewOutcome: newDecision.Outcome})
		}
	}
	return diff
}

func decisionsByTargetKey(decisions []database.RecognitionDecision) map[string]database.RecognitionDecision {
	result := make(map[string]database.RecognitionDecision, len(decisions))
	for _, decision := range decisions {
		result[decision.TargetKey] = decision
	}
	return result
}
