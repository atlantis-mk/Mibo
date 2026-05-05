package metadata

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
)

type MetadataOperationEvidenceInput struct {
	Result            MetadataOperationResult
	LibraryID         uint
	SelectedCandidate any
	StartedAt         time.Time
	FinishedAt        time.Time
}

func (s *Service) recordMetadataOperation(ctx context.Context, input MetadataOperationEvidenceInput) (database.MetadataOperation, error) {
	startedAt := input.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	finishedAt := input.FinishedAt
	if finishedAt.IsZero() {
		finishedAt = time.Now().UTC()
	}
	record := database.MetadataOperation{
		Operation:             input.Result.Operation,
		OriginItemID:          input.Result.OriginItemID,
		TargetItemID:          input.Result.TargetItemID,
		LibraryID:             input.LibraryID,
		Status:                input.Result.Status,
		GovernanceStatus:      input.Result.GovernanceStatus,
		PlanJSON:              marshalOperationJSON(input.Result.Plan),
		AttemptsJSON:          marshalOperationJSON(input.Result.ProviderAttempts),
		SelectedCandidateJSON: marshalOperationJSON(input.SelectedCandidate),
		MetadataSourceIDsJSON: marshalOperationJSON(input.Result.MetadataSourceIDs),
		AppliedFieldsJSON:     marshalOperationJSON(input.Result.AppliedFields),
		SkippedFieldsJSON:     marshalOperationJSON(input.Result.SkippedFields),
		WarningsJSON:          marshalOperationJSON(input.Result.Warnings),
		StartedAt:             startedAt,
		FinishedAt:            &finishedAt,
	}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		return database.MetadataOperation{}, err
	}
	s.markIngestMetadataOutcome(ctx, input.Result, record)
	return record, nil
}

func (s *Service) markIngestMetadataOutcome(ctx context.Context, result MetadataOperationResult, record database.MetadataOperation) {
	if s.ingest == nil || result.TargetItemID == 0 || result.Plan.LibraryID == 0 || !operationAffectsMetadataMatch(result.Operation) {
		return
	}
	status, reason, message := ingestMetadataStatus(result)
	itemIDs := appendUniqueUint([]uint{result.TargetItemID}, result.AffectedScope.ItemIDs...)
	for _, itemID := range itemIDs {
		if itemID == 0 {
			continue
		}
		if _, err := s.ingest.MarkCatalogItemDirty(ctx, itemID, reason); err != nil {
			log.Printf("metadata: mark catalog item %d ingest dirty: %v", itemID, err)
		}
		if _, err := s.ingest.MarkProjectionItemDirty(ctx, itemID, reason); err != nil {
			log.Printf("metadata: mark catalog item %d projection dirty: %v", itemID, err)
		}
		operationID := record.ID
		if _, err := s.ingest.AppendEvent(ctx, database.IngestEvent{UnitKey: "catalog_item:" + strconv.FormatUint(uint64(itemID), 10), LibraryID: result.Plan.LibraryID, CatalogItemID: &itemID, MetadataOperationID: &operationID, ConditionType: ingest.ConditionMetadataMatched, EventType: ingest.EventConditionChanged, Status: status, Reason: reason, Message: message}); err != nil {
			log.Printf("metadata: append ingest metadata event: %v", err)
		}
	}
}

func operationAffectsMetadataMatch(operation string) bool {
	switch operation {
	case OperationTypeMatch, OperationTypeRefetch, OperationTypeManualApply, OperationTypeLocalApply:
		return true
	default:
		return false
	}
}

func ingestMetadataStatus(result MetadataOperationResult) (string, string, string) {
	switch result.Status {
	case OperationStatusApplied:
		if result.GovernanceStatus == "needs_review" {
			return ingest.ConditionStatusReviewRequired, "metadata_needs_review", "Metadata match requires review"
		}
		return ingest.ConditionStatusTrue, "metadata_applied", "Metadata match applied"
	case OperationStatusNoCandidate:
		return ingest.ConditionStatusFalse, "no_candidate", "No acceptable metadata candidate was found"
	case OperationStatusNeedsReview:
		return ingest.ConditionStatusReviewRequired, "metadata_needs_review", "Metadata match requires review"
	case OperationStatusSkipped:
		return ingest.ConditionStatusSkipped, "metadata_skipped", "Metadata matching was skipped"
	case OperationStatusFailed:
		return ingest.ConditionStatusFailed, "metadata_failed", "Metadata matching failed"
	default:
		return ingest.ConditionStatusUnknown, "metadata_unknown", "Metadata matching state is unknown"
	}
}

func marshalOperationJSON(value any) string {
	if value == nil {
		return ""
	}
	data, err := json.Marshal(value)
	if err != nil || string(data) == "null" {
		return ""
	}
	return string(data)
}
