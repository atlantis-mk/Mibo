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

func (s *Service) runMatchMetadataItemOperation(ctx context.Context, input MetadataOperationRequest) (MetadataOperationResult, bool, error) {
	startedAt := time.Now().UTC()
	target, plan, err := s.resolveMetadataItemOperationTarget(ctx, input)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	if target.ItemType != database.MetadataItemTypeMovie && target.ItemType != database.MetadataItemTypeSeries {
		return MetadataOperationResult{}, false, nil
	}
	matchability := metadataOperationMatchability(plan, OperationTypeMatch, "")
	if !matchability.Runnable {
		return MetadataOperationResult{}, false, fmt.Errorf("metadata match is not runnable: %s", matchability.Reason)
	}
	mediaType := metadataItemTMDBMediaType(target.ItemType)
	result := metadataItemOperationResult(OperationTypeMatch, input, target, plan)
	if existingResult, ok, err := s.runMatchMetadataItemFromExistingIdentity(ctx, startedAt, target, plan, result, mediaType); err != nil {
		return MetadataOperationResult{}, false, err
	} else if ok {
		return existingResult, true, nil
	}
	searchItem := metadataItemToSearchItem(target, plan.LibraryID)
	queries := buildSearchQueries(searchItem, mediaType)
	var selectedCandidates []NormalizedMetadataCandidate
	var selectedSearchProvider *settings.ResolvedMetadataProviderInstance
	searchAttempts, searchProvider, err := executeMetadataProviderStage(ctx, "search", plan.SearchProviders, func(ctx context.Context, provider settings.ResolvedMetadataProviderInstance) (MetadataProviderAttempt, bool, error) {
		provider = providerWithOperationLanguage(provider, plan)
		switch provider.Record.ProviderType {
		case database.MetadataProviderTypeTMDB:
			candidates, attempt, err := s.executeTMDBSearchStage(ctx, provider, mediaType, queries, searchItem)
			if err != nil {
				s.recordProviderFailure(ctx, provider, err)
				return attempt, false, err
			}
			selectedCandidates = candidates
			return attempt, len(candidates) > 0, nil
		case database.MetadataProviderTypeMetaTube:
			candidates, attempt, err := s.executeMetaTubeSearchStage(ctx, provider, queries, searchItem)
			if err != nil {
				s.recordProviderFailure(ctx, provider, err)
				return attempt, false, err
			}
			selectedCandidates = candidates
			return attempt, len(candidates) > 0, nil
		default:
			return metadataProviderAttemptForProvider("search", provider, ProviderAttemptOutcomeSkippedUnsupported), false, nil
		}
	})
	result.ProviderAttempts = append(result.ProviderAttempts, searchAttempts...)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	selectedSearchProvider = searchProvider
	if len(selectedCandidates) == 0 || selectedSearchProvider == nil || !acceptableAutomatedMatchCandidate(selectedCandidates[0]) {
		result.Status = OperationStatusNoCandidate
		result.GovernanceStatus = governanceStatusForMetadataOperation(OperationTypeMatch, result.Status, 0)
		_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: plan.LibraryID, StartedAt: startedAt})
		return result, true, err
	}
	selectedCandidate := selectedCandidates[0]
	detailProviders := prioritizeProviderInstances(plan.DetailProviders, selectedSearchProvider.Record.Name)
	detail, attempts, err := s.fetchMetadataItemDetail(ctx, plan, detailProviders, selectedCandidate)
	result.ProviderAttempts = append(result.ProviderAttempts, attempts...)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	return s.applyNormalizedMetadataItemDetail(ctx, startedAt, target, plan, result, detail, selectedCandidate, selectedCandidate.Confidence, OperationTypeMatch)
}

func (s *Service) runRefetchMetadataItemOperation(ctx context.Context, input MetadataOperationRequest) (MetadataOperationResult, bool, error) {
	startedAt := time.Now().UTC()
	target, plan, err := s.resolveMetadataItemOperationTarget(ctx, input)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	if target.ItemType != database.MetadataItemTypeMovie && target.ItemType != database.MetadataItemTypeSeries {
		return MetadataOperationResult{}, false, nil
	}
	providerType, preferredInstance, externalID, confidence, err := s.loadMetadataItemRefetchIdentityForPlan(ctx, target.ID, metadataItemTMDBMediaType(target.ItemType), plan)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	if providerType == database.MetadataProviderTypeLocalScan {
		return s.runMetadataItemLocalApply(ctx, startedAt, input, target, plan, preferredInstance, confidence)
	}
	matchability := metadataOperationMatchability(plan, OperationTypeRefetch, providerType)
	if !matchability.Runnable {
		return MetadataOperationResult{}, false, fmt.Errorf("metadata refetch is not runnable: %s", matchability.Reason)
	}
	result := metadataItemOperationResult(OperationTypeRefetch, input, target, plan)
	selectedCandidate := NormalizedMetadataCandidate{Provider: providerType, ProviderType: metadataItemTMDBMediaType(target.ItemType), ExternalID: externalID, Title: target.Title, Confidence: confidence}
	detailProviders := prioritizeProviderInstances(plan.DetailProviders, preferredInstance)
	detail, attempts, err := s.fetchMetadataItemDetail(ctx, plan, detailProviders, selectedCandidate)
	result.ProviderAttempts = append(result.ProviderAttempts, attempts...)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	return s.applyNormalizedMetadataItemDetail(ctx, startedAt, target, plan, result, detail, selectedCandidate, confidence, OperationTypeRefetch)
}

func (s *Service) runMetadataItemLocalApply(ctx context.Context, startedAt time.Time, input MetadataOperationRequest, target database.MetadataItem, plan MetadataExecutionPlan, preferredInstance string, confidence float64) (MetadataOperationResult, bool, error) {
	evidence, err := s.loadMetadataItemLocalScannerEvidence(ctx, target.ID)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	detail, ok := localEvidenceDetail(evidence, metadataItemTypeToCatalogType(target.ItemType))
	result := metadataItemOperationResult(OperationTypeLocalApply, input, target, plan)
	result.ProviderAttempts = append(result.ProviderAttempts, MetadataProviderAttempt{Stage: "local_evidence", ProviderInstanceName: preferredInstance, ProviderType: database.MetadataProviderTypeLocalScan, Outcome: ProviderAttemptOutcomeSuccess, CandidateCount: 1, Selected: true})
	if !ok {
		return MetadataOperationResult{}, false, fmt.Errorf("no parsed scanner sidecar evidence is available for metadata item %d", target.ID)
	}
	selectedCandidate := NormalizedMetadataCandidate{Provider: detail.Provider, ProviderType: detail.ProviderType, ExternalID: detail.ExternalID, Title: detail.Title, Confidence: confidence}
	return s.applyNormalizedMetadataItemDetail(ctx, startedAt, target, plan, result, detail, selectedCandidate, confidence, OperationTypeLocalApply)
}

func (s *Service) runManualApplyMetadataItemOperation(ctx context.Context, input MetadataOperationRequest) (MetadataOperationResult, error) {
	startedAt := time.Now().UTC()
	target, plan, err := s.resolveMetadataItemOperationTarget(ctx, input)
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
	result := metadataItemOperationResult(OperationTypeManualApply, input, target, plan)
	selectedCandidate := NormalizedMetadataCandidate{Provider: providerType, ProviderType: detailProviderType, ExternalID: strings.TrimSpace(input.ManualCandidateExternalID), Title: target.Title, Confidence: 1}
	detail, attempts, err := s.fetchMetadataItemDetail(ctx, plan, plan.DetailProviders, selectedCandidate)
	result.ProviderAttempts = append(result.ProviderAttempts, attempts...)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	if strings.TrimSpace(detail.ExternalID) == "" || strings.TrimSpace(detail.ProviderType) != strings.TrimSpace(detailProviderType) && providerType == database.MetadataProviderTypeMetaTube {
		return MetadataOperationResult{}, fmt.Errorf("metadata detail stage did not select a provider")
	}
	result, _, err = s.applyNormalizedMetadataItemDetail(ctx, startedAt, target, plan, result, detail, selectedCandidate, 1, OperationTypeManualApply)
	return result, err
}

func (s *Service) fetchMetadataItemDetail(ctx context.Context, plan MetadataExecutionPlan, providers []settings.ResolvedMetadataProviderInstance, candidate NormalizedMetadataCandidate) (NormalizedMetadataDetail, []MetadataProviderAttempt, error) {
	var detail NormalizedMetadataDetail
	attempts, _, err := executeMetadataProviderStage(ctx, "detail", providers, func(ctx context.Context, provider settings.ResolvedMetadataProviderInstance) (MetadataProviderAttempt, bool, error) {
		provider = providerWithOperationLanguage(provider, plan)
		switch provider.Record.ProviderType {
		case database.MetadataProviderTypeTMDB:
			if candidate.Provider != database.MetadataProviderTypeTMDB {
				return metadataProviderAttemptForProvider("detail", provider, ProviderAttemptOutcomeSkippedUnsupported), false, nil
			}
			mediaType, tmdbID, err := parseExternalID(candidate.ExternalID)
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
			if candidate.Provider != database.MetadataProviderTypeMetaTube {
				return metadataProviderAttemptForProvider("detail", provider, ProviderAttemptOutcomeSkippedUnsupported), false, nil
			}
			upstreamProvider, upstreamID, err := parseMetaTubeExternalID(candidate.ExternalID)
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
	if err != nil {
		return NormalizedMetadataDetail{}, attempts, err
	}
	if strings.TrimSpace(detail.ExternalID) == "" {
		return NormalizedMetadataDetail{}, attempts, fmt.Errorf("metadata detail stage did not select a provider")
	}
	return detail, attempts, nil
}

func (s *Service) applyNormalizedMetadataItemDetail(ctx context.Context, startedAt time.Time, target database.MetadataItem, plan MetadataExecutionPlan, result MetadataOperationResult, detail NormalizedMetadataDetail, selectedCandidate NormalizedMetadataCandidate, confidence float64, operation string) (MetadataOperationResult, bool, error) {
	status := OperationStatusApplied
	governanceStatus := governanceStatusForMetadataOperation(operation, status, confidence)
	if governanceStatus == catalog.GovernanceNeedsReview {
		status = OperationStatusNeedsReview
	}
	source, err := s.recordNormalizedMetadataItemProviderSource(ctx, target.ID, plan, detail, confidence)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	sourceID := source.ID
	changes := normalizedDetailFieldChanges(target.ID, detail, &sourceID, applyModeForMetadataItemOperation(operation), &confidence)
	changes = append(changes, MetadataFieldChange{ItemID: target.ID, FieldKey: "governance_status", Value: governanceStatus, SourceID: &sourceID, ApplyMode: applyModeForMetadataItemOperation(operation), Confidence: &confidence})
	applied, skipped, err := s.applyMetadataItemFieldChanges(ctx, changes)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	if err := s.applyNormalizedMetadataItemExternalIDs(ctx, target.ID, detail.ExternalIDs, operation, &confidence); err != nil {
		return MetadataOperationResult{}, false, err
	}
	if err := s.applyNormalizedMetadataItemTags(ctx, target.ID, detail.Tags, &sourceID); err != nil {
		return MetadataOperationResult{}, false, err
	}
	applied = append(applied, appliedTagFields(target.ID, detail.Tags, &sourceID, applyModeForMetadataItemOperation(operation), &confidence)...)
	if err := s.applyNormalizedMetadataItemImages(ctx, target.ID, detail.Images, operation == OperationTypeManualApply, &sourceID); err != nil {
		return MetadataOperationResult{}, false, err
	}
	if err := s.applyNormalizedMetadataItemPeople(ctx, target.ID, detail.People, &sourceID); err != nil {
		return MetadataOperationResult{}, false, err
	}
	if target.ItemType == database.MetadataItemTypeSeries && detail.Hierarchy != nil {
		hierarchyResult, err := s.applyNormalizedMetadataItemTVHierarchy(ctx, target, plan, *detail.Hierarchy, governanceStatus, confidence, operation == OperationTypeManualApply)
		if err != nil {
			return MetadataOperationResult{}, false, err
		}
		result.AffectedScope.MetadataItemIDs = appendUniqueUint(result.AffectedScope.MetadataItemIDs, hierarchyResult.AffectedMetadataItemIDs...)
		result.MetadataSourceIDs = appendUniqueUint(result.MetadataSourceIDs, hierarchyResult.MetadataSourceIDs...)
		applied = append(applied, hierarchyResult.AppliedFields...)
		skipped = append(skipped, hierarchyResult.SkippedFields...)
	}
	result.Status = status
	result.GovernanceStatus = governanceStatus
	result.MetadataSourceIDs = appendUniqueUint(result.MetadataSourceIDs, source.ID)
	result.AppliedFields = applied
	result.SkippedFields = skipped
	result.AffectedScope.MetadataItemIDs = appendUniqueUint(result.AffectedScope.MetadataItemIDs, target.ID)
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: plan.LibraryID, SelectedCandidate: selectedCandidate, StartedAt: startedAt})
	return result, true, err
}

func applyModeForMetadataItemOperation(operation string) string {
	switch operation {
	case OperationTypeManualApply:
		return FieldApplyModeManual
	case OperationTypeLocalApply:
		return FieldApplyModeLocal
	default:
		return FieldApplyModeAutomated
	}
}

func (s *Service) runMatchMetadataItemFromExistingIdentity(ctx context.Context, startedAt time.Time, target database.MetadataItem, plan MetadataExecutionPlan, result MetadataOperationResult, mediaType string) (MetadataOperationResult, bool, error) {
	preferredInstance, externalID, confidence, err := s.loadMetadataItemProviderIdentity(ctx, target.ID, database.MetadataProviderTypeTMDB, mediaType)
	if err != nil || strings.TrimSpace(externalID) == "" || confidence < 0.85 {
		return MetadataOperationResult{}, false, nil
	}
	selectedCandidate := NormalizedMetadataCandidate{Provider: database.MetadataProviderTypeTMDB, ProviderType: mediaType, ExternalID: externalID, Title: target.Title, Confidence: confidence}
	detailProviders := prioritizeProviderInstances(plan.DetailProviders, preferredInstance)
	detail, attempts, err := s.fetchMetadataItemDetail(ctx, plan, detailProviders, selectedCandidate)
	result.ProviderAttempts = append(result.ProviderAttempts, attempts...)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	return s.applyNormalizedMetadataItemDetail(ctx, startedAt, target, plan, result, detail, selectedCandidate, confidence, OperationTypeMatch)
}

func (s *Service) loadMetadataItemRefetchIdentityForPlan(ctx context.Context, metadataItemID uint, mediaType string, plan MetadataExecutionPlan) (string, string, string, float64, error) {
	for _, provider := range plan.DetailProviders {
		if !provider.Operational {
			continue
		}
		switch provider.Record.ProviderType {
		case database.MetadataProviderTypeMetaTube:
			preferredInstance, externalID, confidence, err := s.loadMetadataItemProviderIdentity(ctx, metadataItemID, database.MetadataProviderTypeMetaTube, "")
			if err == nil && strings.TrimSpace(externalID) != "" {
				return database.MetadataProviderTypeMetaTube, firstNonEmpty(preferredInstance, provider.Record.Name), externalID, confidence, nil
			}
		case database.MetadataProviderTypeTMDB:
			preferredInstance, externalID, confidence, err := s.loadMetadataItemProviderIdentity(ctx, metadataItemID, database.MetadataProviderTypeTMDB, mediaType)
			if err == nil && strings.TrimSpace(externalID) != "" {
				return database.MetadataProviderTypeTMDB, firstNonEmpty(preferredInstance, provider.Record.Name), externalID, confidence, nil
			}
		case database.MetadataProviderTypeLocalScan:
			if _, err := s.loadMetadataItemLocalScannerEvidence(ctx, metadataItemID); err == nil {
				return database.MetadataProviderTypeLocalScan, provider.Record.Name, "local_scan", 1, nil
			}
		}
	}
	return "", "", "", 0, fmt.Errorf("metadata item %d has no refetch identity", metadataItemID)
}

func (s *Service) loadMetadataItemProviderIdentity(ctx context.Context, metadataItemID uint, provider string, providerType string) (string, string, float64, error) {
	var identity database.MetadataExternalID
	query := s.db.WithContext(ctx).Where("metadata_item_id = ? AND provider = ?", metadataItemID, provider)
	if strings.TrimSpace(providerType) != "" {
		query = query.Where("provider_type = ?", providerType)
	}
	if err := query.Order("is_primary desc, id asc").First(&identity).Error; err != nil {
		return "", "", 0, err
	}
	confidence := 1.0
	if identity.Confidence != nil && *identity.Confidence > 0 {
		confidence = *identity.Confidence
	}
	providerInstanceName := ""
	var source database.MetadataItemSource
	if err := s.db.WithContext(ctx).Where("metadata_item_id = ? AND source_name = ? AND external_id = ?", metadataItemID, provider, strings.TrimSpace(identity.ExternalID)).Order("id desc").First(&source).Error; err == nil {
		providerInstanceName = strings.TrimSpace(source.ProviderInstanceName)
	}
	return providerInstanceName, strings.TrimSpace(identity.ExternalID), confidence, nil
}

func metadataItemOperationResult(operation string, input MetadataOperationRequest, target database.MetadataItem, plan MetadataExecutionPlan) MetadataOperationResult {
	originMetadataID := input.OriginMetadataItemID
	if originMetadataID == 0 {
		originMetadataID = target.ID
	}
	return MetadataOperationResult{Operation: operation, OriginMetadataItemID: originMetadataID, TargetMetadataItemID: target.ID, TargetType: metadataItemTypeToCatalogType(target.ItemType), Plan: metadataExecutionPlanSummary(plan), AffectedScope: MetadataAffectedScope{MetadataItemIDs: []uint{target.ID}, LibraryID: plan.LibraryID, MetadataRootID: target.RootID}}
}

func metadataItemTMDBMediaType(itemType string) string {
	switch strings.TrimSpace(itemType) {
	case database.MetadataItemTypeSeries, database.MetadataItemTypeSeason, database.MetadataItemTypeEpisode:
		return "tv"
	default:
		return "movie"
	}
}
