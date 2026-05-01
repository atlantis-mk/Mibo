package metadata

import "github.com/atlan/mibo-media-server/internal/catalog"

func governanceStatusForMetadataOperation(operation string, status string, confidence float64) string {
	switch operation {
	case OperationTypeManualApply:
		return catalog.GovernanceManual
	case OperationTypeLocalApply:
		if status == OperationStatusApplied {
			return catalog.GovernanceMatched
		}
	case OperationTypeMatch, OperationTypeRefetch:
		switch status {
		case OperationStatusNoCandidate:
			return catalog.GovernanceUnmatched
		case OperationStatusNeedsReview:
			return catalog.GovernanceNeedsReview
		case OperationStatusApplied:
			if confidence > 0 && confidence < 0.85 {
				return catalog.GovernanceNeedsReview
			}
			return catalog.GovernanceMatched
		}
	}
	return ""
}
