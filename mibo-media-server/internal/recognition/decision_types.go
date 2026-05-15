package recognition

import (
	"strconv"

	"github.com/atlan/mibo-media-server/internal/database"
)

const (
	DecisionOutcomeAccepted        = "accepted"
	DecisionOutcomeProvisional     = "provisional"
	DecisionOutcomeReviewRequired  = "review_required"
	DecisionOutcomeBlockedConflict = "blocked_conflict"
	DecisionOutcomeUnmatched       = "unmatched"
	DecisionOutcomeRejected        = "rejected"
)

type ResolveResult struct {
	Decisions []database.RecognitionDecision
	Conflicts []database.RecognitionConflict
}

func stringUint(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}
