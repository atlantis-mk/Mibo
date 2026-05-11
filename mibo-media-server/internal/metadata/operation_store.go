package metadata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
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
		DeduplicationKey:      metadataOperationDeduplicationKey(input.Result),
		OriginMetadataItemID:  input.Result.OriginMetadataItemID,
		TargetMetadataItemID:  input.Result.TargetMetadataItemID,
		LibraryID:             input.LibraryID,
		Status:                input.Result.Status,
		GovernanceStatus:      input.Result.GovernanceStatus,
		PlanJSON:              marshalOperationJSON(input.Result.Plan),
		TriggerContextJSON:    marshalOperationJSON(metadataOperationTriggerContext(input.Result.Plan)),
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

func metadataOperationDeduplicationKey(result MetadataOperationResult) string {
	metadataItemID := result.TargetMetadataItemID
	if metadataItemID == 0 {
		metadataItemID = result.OriginMetadataItemID
	}
	if metadataItemID == 0 {
		return ""
	}
	parts := []string{result.Operation, strconv.FormatUint(uint64(metadataItemID), 10), strings.TrimSpace(result.Plan.PreferredMetadataLanguage), strconv.FormatUint(uint64(derefUintForOperation(result.Plan.MetadataProfileID)), 10), strings.TrimSpace(result.Plan.MetadataProfileName)}
	for _, provider := range result.Plan.DetailProviders {
		parts = append(parts, "detail:"+strconv.FormatUint(uint64(provider.ID), 10)+":"+strings.TrimSpace(provider.Name)+":"+strings.TrimSpace(provider.ProviderType))
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(digest[:])
}

func metadataOperationTriggerContext(plan MetadataExecutionPlanSummary) map[string]any {
	return map[string]any{"library_id": plan.LibraryID, "metadata_profile_id": plan.MetadataProfileID, "metadata_profile_name": plan.MetadataProfileName, "preferred_metadata_language": plan.PreferredMetadataLanguage, "preferred_image_language": plan.PreferredImageLanguage}
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
