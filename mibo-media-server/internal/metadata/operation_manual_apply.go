package metadata

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/settings"
)

func (s *Service) runManualApplyMetadataOperation(ctx context.Context, input MetadataOperationRequest) (MetadataOperationResult, error) {
	startedAt := time.Now().UTC()
	target, err := s.resolveCatalogMatchTarget(ctx, input.OriginItemID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	plan, err := s.resolveMetadataExecutionPlan(ctx, target.LibraryID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	providerType, detailProviderType, err := manualCandidateProviderTypes(input.ManualCandidateExternalID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	matchability := metadataOperationMatchability(plan, OperationTypeManualApply, providerType)
	if !matchability.Runnable {
		return MetadataOperationResult{}, fmt.Errorf("metadata manual apply is not runnable: %s", matchability.Reason)
	}
	result := MetadataOperationResult{Operation: OperationTypeManualApply, OriginItemID: input.OriginItemID, TargetItemID: target.ID, TargetType: target.Type, Plan: metadataExecutionPlanSummary(plan), AffectedScope: MetadataAffectedScope{ItemIDs: []uint{target.ID}, LibraryID: target.LibraryID, RootID: target.RootID}}
	var detail NormalizedMetadataDetail
	detailAttempts, _, err := executeMetadataProviderStage(ctx, "detail", plan.DetailProviders, func(ctx context.Context, provider settings.ResolvedMetadataProviderInstance) (MetadataProviderAttempt, bool, error) {
		provider = providerWithOperationLanguage(provider, plan)
		if provider.Record.ProviderType != providerType {
			return metadataProviderAttemptForProvider("detail", provider, ProviderAttemptOutcomeSkippedUnsupported), false, nil
		}
		switch provider.Record.ProviderType {
		case database.MetadataProviderTypeTMDB:
			mediaType, tmdbID, err := parseExternalID(input.ManualCandidateExternalID)
			if err != nil {
				return metadataProviderFailureAttempt("detail", provider, err), false, err
			}
			normalized, attempt, err := s.executeTMDBDetailStage(ctx, provider, mediaType, tmdbID)
			if err != nil {
				s.recordProviderFailure(ctx, provider, err)
				return attempt, false, err
			}
			detail = normalized
			return attempt, true, nil
		case database.MetadataProviderTypeMetaTube:
			upstreamProvider, upstreamID, err := parseMetaTubeExternalID(input.ManualCandidateExternalID)
			if err != nil {
				return metadataProviderFailureAttempt("detail", provider, err), false, err
			}
			normalized, attempt, err := s.executeMetaTubeDetailStage(ctx, provider, upstreamProvider, upstreamID)
			if err != nil {
				s.recordProviderFailure(ctx, provider, err)
				return attempt, false, err
			}
			detail = normalized
			return attempt, true, nil
		default:
			return metadataProviderAttemptForProvider("detail", provider, ProviderAttemptOutcomeSkippedUnsupported), false, nil
		}
	})
	result.ProviderAttempts = append(result.ProviderAttempts, detailAttempts...)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	if strings.TrimSpace(detail.ExternalID) == "" || strings.TrimSpace(detail.ProviderType) != strings.TrimSpace(detailProviderType) && providerType == database.MetadataProviderTypeMetaTube {
		return MetadataOperationResult{}, fmt.Errorf("metadata detail stage did not select a provider")
	}
	selectedCandidate := NormalizedMetadataCandidate{Provider: detail.Provider, ProviderType: detail.ProviderType, ExternalID: detail.ExternalID, Title: detail.Title, Confidence: 1}
	return s.applyNormalizedManualDetail(ctx, startedAt, target, plan, result, detail, selectedCandidate)
}

func (s *Service) applyNormalizedManualDetail(ctx context.Context, startedAt time.Time, target database.CatalogItem, plan MetadataExecutionPlan, result MetadataOperationResult, detail NormalizedMetadataDetail, selectedCandidate NormalizedMetadataCandidate) (MetadataOperationResult, error) {
	confidence := 1.0
	governanceStatus := governanceStatusForMetadataOperation(OperationTypeManualApply, OperationStatusApplied, confidence)
	source, err := s.recordNormalizedProviderSource(ctx, target, plan, detail, confidence)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	sourceID := source.ID
	changes := normalizedDetailFieldChanges(target.ID, detail, &sourceID, FieldApplyModeManual, &confidence)
	changes = append(changes, MetadataFieldChange{ItemID: target.ID, FieldKey: "governance_status", Value: governanceStatus, SourceID: &sourceID, ApplyMode: FieldApplyModeManual, Confidence: &confidence})
	applied, skipped, err := s.applyMetadataFieldChanges(ctx, changes)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	if err := applyNormalizedExternalIDs(ctx, catalog.NewService(s.db), target.ID, detail.ExternalIDs, "metadata_match", &confidence); err != nil {
		return MetadataOperationResult{}, err
	}
	if err := s.applyNormalizedTags(ctx, target.ID, detail.Tags, &sourceID); err != nil {
		return MetadataOperationResult{}, err
	}
	applied = append(applied, appliedTagFields(target.ID, detail.Tags, &sourceID, FieldApplyModeManual, &confidence)...)
	if err := s.applyNormalizedImages(ctx, target.ID, detail.Images, true, &sourceID); err != nil {
		return MetadataOperationResult{}, err
	}
	if err := s.applyNormalizedPeople(ctx, target.ID, detail.People, &sourceID); err != nil {
		return MetadataOperationResult{}, err
	}
	result.Status = OperationStatusApplied
	result.GovernanceStatus = governanceStatus
	result.MetadataSourceIDs = []uint{source.ID}
	result.AppliedFields = applied
	result.SkippedFields = skipped
	if err := s.refreshMetadataOperationProjectionScope(ctx, result.AffectedScope); err != nil {
		return MetadataOperationResult{}, err
	}
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: target.LibraryID, SelectedCandidate: selectedCandidate, StartedAt: startedAt})
	return result, err
}

func manualCandidateProviderTypes(externalID string) (string, string, error) {
	externalID = strings.TrimSpace(externalID)
	if strings.HasPrefix(externalID, database.MetadataProviderTypeMetaTube+":") {
		upstreamProvider, _, err := parseMetaTubeExternalID(externalID)
		return database.MetadataProviderTypeMetaTube, upstreamProvider, err
	}
	mediaType, _, err := parseExternalID(externalID)
	return database.MetadataProviderTypeTMDB, mediaType, err
}
