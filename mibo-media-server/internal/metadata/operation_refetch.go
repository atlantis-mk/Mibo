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

func (s *Service) runRefetchMetadataOperation(ctx context.Context, input MetadataOperationRequest) (MetadataOperationResult, bool, error) {
	startedAt := time.Now().UTC()
	origin, err := s.loadCatalogMetadataOrigin(ctx, input.OriginItemID)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	target, err := s.resolveCatalogMatchTarget(ctx, input.OriginItemID)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	if target.Type != catalog.ItemTypeMovie && target.Type != catalog.ItemTypeSeries {
		return MetadataOperationResult{}, false, nil
	}
	plan, err := s.resolveMetadataExecutionPlan(ctx, target.LibraryID)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	providerType, preferredInstance, externalID, confidence, err := s.loadRefetchIdentityForPlan(ctx, target.ID, catalogTMDBMediaType(target.Type), plan)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	if providerType == database.MetadataProviderTypeLocalScan {
		return s.runLocalScanRefetchMetadataOperation(ctx, startedAt, origin, target, plan, preferredInstance)
	}
	matchability := metadataOperationMatchability(plan, OperationTypeRefetch, providerType)
	if !matchability.Runnable {
		return MetadataOperationResult{}, false, fmt.Errorf("metadata refetch is not runnable: %s", matchability.Reason)
	}
	result := MetadataOperationResult{Operation: OperationTypeRefetch, OriginItemID: origin.ID, TargetItemID: target.ID, TargetType: target.Type, Plan: metadataExecutionPlanSummary(plan), AffectedScope: MetadataAffectedScope{ItemIDs: []uint{target.ID}, LibraryID: target.LibraryID, RootID: target.RootID}}
	var detail NormalizedMetadataDetail
	detailProviders := prioritizeProviderInstances(plan.DetailProviders, preferredInstance)
	detailAttempts, _, err := executeMetadataProviderStage(ctx, "detail", detailProviders, func(ctx context.Context, provider settings.ResolvedMetadataProviderInstance) (MetadataProviderAttempt, bool, error) {
		provider = providerWithOperationLanguage(provider, plan)
		if provider.Record.ProviderType != providerType {
			return metadataProviderAttemptForProvider("detail", provider, ProviderAttemptOutcomeSkippedUnsupported), false, nil
		}
		switch provider.Record.ProviderType {
		case database.MetadataProviderTypeTMDB:
			mediaType, tmdbID, err := parseExternalID(externalID)
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
			upstreamProvider, upstreamID, err := parseMetaTubeExternalID(externalID)
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
		return MetadataOperationResult{}, false, err
	}
	if strings.TrimSpace(detail.ExternalID) == "" {
		return MetadataOperationResult{}, false, fmt.Errorf("metadata detail stage did not select a provider")
	}
	if target.Type == catalog.ItemTypeSeries && detail.Provider == database.MetadataProviderTypeTMDB && detail.ProviderType == "tv" {
		detailProvider := detailProviderForResult(plan.DetailProviders, detail.Provider)
		if detailProvider.Record.ID != 0 {
			detailProvider = providerWithOperationLanguage(detailProvider, plan)
			if err := s.completeTMDBTVHierarchy(ctx, detailProvider, &detail); err != nil {
				s.recordProviderFailure(ctx, detailProvider, err)
				return MetadataOperationResult{}, false, err
			}
			result.ProviderAttempts = append(result.ProviderAttempts, hierarchyProviderAttempt(detailProvider, detail.Hierarchy))
		}
	}
	selectedCandidate := NormalizedMetadataCandidate{Provider: detail.Provider, ProviderType: detail.ProviderType, ExternalID: detail.ExternalID, Title: detail.Title, Confidence: confidence}
	return s.applyNormalizedRefetchDetail(ctx, startedAt, target, plan, result, detail, selectedCandidate, confidence)
}

func (s *Service) runLocalScanRefetchMetadataOperation(ctx context.Context, startedAt time.Time, origin database.CatalogItem, target database.CatalogItem, plan MetadataExecutionPlan, preferredInstance string) (MetadataOperationResult, bool, error) {
	provider := detailProviderForResult(plan.DetailProviders, database.MetadataProviderTypeLocalScan)
	if provider.Record.ID == 0 {
		return MetadataOperationResult{}, false, fmt.Errorf("metadata local refetch is not runnable: local_scan provider is not available")
	}
	if strings.TrimSpace(preferredInstance) != "" {
		for _, candidate := range plan.DetailProviders {
			if candidate.Record.ProviderType == database.MetadataProviderTypeLocalScan && candidate.Record.Name == preferredInstance {
				provider = candidate
				break
			}
		}
	}
	result := MetadataOperationResult{Operation: OperationTypeRefetch, OriginItemID: origin.ID, TargetItemID: target.ID, TargetType: target.Type, Plan: metadataExecutionPlanSummary(plan), AffectedScope: MetadataAffectedScope{ItemIDs: []uint{target.ID}, LibraryID: target.LibraryID, RootID: target.RootID}}
	evidence, err := s.loadLocalScannerEvidence(ctx, target.ID)
	if err != nil {
		result.ProviderAttempts = append(result.ProviderAttempts, localEvidenceProviderAttempt(provider, false))
		return MetadataOperationResult{}, false, err
	}
	detail, ok := localEvidenceDetail(evidence, target.Type)
	result.ProviderAttempts = append(result.ProviderAttempts, localEvidenceProviderAttempt(provider, ok))
	if !ok {
		return MetadataOperationResult{}, false, fmt.Errorf("no parsed scanner sidecar evidence is available for item %d", target.ID)
	}
	confidence := 1.0
	governanceStatus := governanceStatusForMetadataOperation(OperationTypeLocalApply, OperationStatusApplied, confidence)
	if governanceStatus == "" {
		governanceStatus = catalog.GovernanceMatched
	}
	source, err := s.recordNormalizedLocalScanSource(ctx, target, plan, provider, evidence, detail, confidence)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	sourceID := source.ID
	changes := normalizedDetailFieldChanges(target.ID, detail, &sourceID, FieldApplyModeLocal, &confidence)
	changes = append(changes, MetadataFieldChange{ItemID: target.ID, FieldKey: "governance_status", Value: governanceStatus, SourceID: &sourceID, ApplyMode: FieldApplyModeLocal, Confidence: &confidence})
	applied, skipped, err := s.applyMetadataFieldChanges(ctx, changes)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	if err := applyNormalizedExternalIDs(ctx, catalog.NewService(s.db), target.ID, detail.ExternalIDs, "local_scan", &confidence); err != nil {
		return MetadataOperationResult{}, false, err
	}
	if err := s.applyNormalizedImages(ctx, target.ID, detail.Images, false, &sourceID); err != nil {
		return MetadataOperationResult{}, false, err
	}
	result.Status = OperationStatusApplied
	result.GovernanceStatus = governanceStatus
	result.MetadataSourceIDs = []uint{source.ID}
	result.AppliedFields = applied
	result.SkippedFields = skipped
	if err := s.refreshMetadataOperationProjectionScope(ctx, result.AffectedScope); err != nil {
		return MetadataOperationResult{}, false, err
	}
	selectedCandidate := NormalizedMetadataCandidate{Provider: detail.Provider, ProviderType: detail.ProviderType, ExternalID: detail.ExternalID, Title: detail.Title, Confidence: confidence}
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: target.LibraryID, SelectedCandidate: selectedCandidate, StartedAt: startedAt})
	return result, true, err
}

func (s *Service) applyNormalizedRefetchDetail(ctx context.Context, startedAt time.Time, target database.CatalogItem, plan MetadataExecutionPlan, result MetadataOperationResult, detail NormalizedMetadataDetail, selectedCandidate NormalizedMetadataCandidate, confidence float64) (MetadataOperationResult, bool, error) {
	status := OperationStatusApplied
	governanceStatus := governanceStatusForMetadataOperation(OperationTypeRefetch, status, confidence)
	if governanceStatus == catalog.GovernanceNeedsReview {
		status = OperationStatusNeedsReview
	}
	source, err := s.recordNormalizedProviderSource(ctx, target, plan, detail, confidence)
	if err != nil {
		return MetadataOperationResult{}, true, err
	}
	sourceID := source.ID
	changes := normalizedDetailFieldChanges(target.ID, detail, &sourceID, FieldApplyModeAutomated, &confidence)
	changes = append(changes, MetadataFieldChange{ItemID: target.ID, FieldKey: "governance_status", Value: governanceStatus, SourceID: &sourceID, ApplyMode: FieldApplyModeAutomated, Confidence: &confidence})
	applied, skipped, err := s.applyMetadataFieldChanges(ctx, changes)
	if err != nil {
		return MetadataOperationResult{}, true, err
	}
	if err := applyNormalizedExternalIDs(ctx, catalog.NewService(s.db), target.ID, detail.ExternalIDs, "metadata_refetch", &confidence); err != nil {
		return MetadataOperationResult{}, true, err
	}
	if err := s.applyNormalizedTags(ctx, target.ID, detail.Tags, &sourceID); err != nil {
		return MetadataOperationResult{}, true, err
	}
	applied = append(applied, appliedTagFields(target.ID, detail.Tags, &sourceID, FieldApplyModeAutomated, &confidence)...)
	if err := s.applyNormalizedImages(ctx, target.ID, detail.Images, false, &sourceID); err != nil {
		return MetadataOperationResult{}, true, err
	}
	if err := s.applyNormalizedPeople(ctx, target.ID, detail.People, &sourceID); err != nil {
		return MetadataOperationResult{}, true, err
	}
	if target.Type == catalog.ItemTypeSeries && detail.Hierarchy != nil {
		hierarchyProvider := detailProviderForResult(plan.DetailProviders, detail.Provider)
		if hierarchyProvider.Record.ID != 0 {
			hierarchyResult, err := s.applyNormalizedTVHierarchy(ctx, target, resolvedProfileFromPlan(plan), hierarchyProvider, *detail.Hierarchy, governanceStatus, confidence, false)
			if err != nil {
				return MetadataOperationResult{}, true, err
			}
			result.AffectedScope.ItemIDs = appendUniqueUint(result.AffectedScope.ItemIDs, hierarchyResult.AffectedItemIDs...)
			result.MetadataSourceIDs = appendUniqueUint(result.MetadataSourceIDs, hierarchyResult.MetadataSourceIDs...)
			applied = append(applied, hierarchyResult.AppliedFields...)
			skipped = append(skipped, hierarchyResult.SkippedFields...)
		}
	}
	result.Status = status
	result.GovernanceStatus = governanceStatus
	result.MetadataSourceIDs = appendUniqueUint(result.MetadataSourceIDs, source.ID)
	result.AppliedFields = applied
	result.SkippedFields = skipped
	if err := s.refreshMetadataOperationProjectionScope(ctx, result.AffectedScope); err != nil {
		return MetadataOperationResult{}, true, err
	}
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: target.LibraryID, SelectedCandidate: selectedCandidate, StartedAt: startedAt})
	return result, true, err
}

func (s *Service) loadRefetchIdentity(ctx context.Context, itemID uint, mediaType string) (string, string, string, float64, error) {
	preferredInstance, externalID, confidence, err := s.loadCatalogTMDBIdentity(ctx, itemID, mediaType)
	if err == nil && strings.TrimSpace(externalID) != "" {
		return database.MetadataProviderTypeTMDB, preferredInstance, externalID, confidence, nil
	}
	preferredInstance, externalID, confidence, err = s.loadCatalogMetaTubeIdentity(ctx, itemID)
	if err == nil && strings.TrimSpace(externalID) != "" {
		return database.MetadataProviderTypeMetaTube, preferredInstance, externalID, confidence, nil
	}
	var localSource database.MetadataSource
	if err := s.db.WithContext(ctx).Where("item_id = ? AND source_type = ? AND source_name = ?", itemID, catalog.SourceTypeLocalFile, "scanner").Order("fetched_at desc, id desc").First(&localSource).Error; err == nil {
		return database.MetadataProviderTypeLocalScan, "", "local_scan", 1, nil
	}
	return "", "", "", 0, fmt.Errorf("当前 catalog 条目没有可重抓的匹配结果")
}

func (s *Service) loadRefetchIdentityForPlan(ctx context.Context, itemID uint, mediaType string, plan MetadataExecutionPlan) (string, string, string, float64, error) {
	for _, provider := range plan.DetailProviders {
		if !provider.Operational {
			continue
		}
		switch provider.Record.ProviderType {
		case database.MetadataProviderTypeMetaTube:
			preferredInstance, externalID, confidence, err := s.loadCatalogMetaTubeIdentity(ctx, itemID)
			if err == nil && strings.TrimSpace(externalID) != "" {
				return database.MetadataProviderTypeMetaTube, firstNonEmpty(preferredInstance, provider.Record.Name), externalID, confidence, nil
			}
		case database.MetadataProviderTypeTMDB:
			preferredInstance, externalID, confidence, err := s.loadCatalogTMDBIdentity(ctx, itemID, mediaType)
			if err == nil && strings.TrimSpace(externalID) != "" {
				return database.MetadataProviderTypeTMDB, firstNonEmpty(preferredInstance, provider.Record.Name), externalID, confidence, nil
			}
		case database.MetadataProviderTypeLocalScan:
			if s.hasLocalScannerEvidence(ctx, itemID) {
				return database.MetadataProviderTypeLocalScan, provider.Record.Name, "local_scan", 1, nil
			}
		}
	}
	return s.loadRefetchIdentity(ctx, itemID, mediaType)
}
