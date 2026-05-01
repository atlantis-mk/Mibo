package metadata

import (
	"context"
	"encoding/json"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
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
	return record, nil
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
