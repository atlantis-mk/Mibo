package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/settings"
)

func (s *Service) runMatchMetadataOperation(ctx context.Context, input MetadataOperationRequest) (MetadataOperationResult, bool, error) {
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
	matchability := metadataOperationMatchability(plan, OperationTypeMatch, "")
	if !matchability.Runnable {
		return MetadataOperationResult{}, false, fmt.Errorf("metadata match is not runnable: %s", matchability.Reason)
	}
	if !hasOperationalProvider(plan.SearchProviders) && plan.LocalEvidenceEnabled && !s.hasLocalScannerEvidence(ctx, target.ID) {
		return MetadataOperationResult{}, false, fmt.Errorf("metadata match is not runnable: no operational search provider or local evidence for item")
	}

	result := MetadataOperationResult{Operation: OperationTypeMatch, OriginItemID: origin.ID, TargetItemID: target.ID, TargetType: target.Type, Plan: metadataExecutionPlanSummary(plan), AffectedScope: MetadataAffectedScope{ItemIDs: []uint{target.ID}, LibraryID: target.LibraryID, RootID: target.RootID}}
	mediaType := catalogTMDBMediaType(target.Type)
	if existingResult, ok, err := s.runMatchFromExistingIdentity(ctx, startedAt, origin, target, plan, result, mediaType); err != nil {
		return MetadataOperationResult{}, false, err
	} else if ok {
		return existingResult, true, nil
	}
	searchItem := catalogItemToSearchItem(target)
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
	if len(selectedCandidates) == 0 || selectedSearchProvider == nil {
		result.Status = OperationStatusNoCandidate
		result.GovernanceStatus = governanceStatusForMetadataOperation(OperationTypeMatch, result.Status, 0)
		if err := s.applyCatalogGovernanceStatus(ctx, target.ID, result.GovernanceStatus); err != nil {
			return MetadataOperationResult{}, false, err
		}
		_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: target.LibraryID, StartedAt: startedAt})
		return result, true, err
	}

	selectedCandidate := selectedCandidates[0]
	if !acceptableAutomatedMatchCandidate(selectedCandidate) {
		result.Status = OperationStatusNoCandidate
		result.GovernanceStatus = governanceStatusForMetadataOperation(OperationTypeMatch, result.Status, 0)
		if err := s.applyCatalogGovernanceStatus(ctx, target.ID, result.GovernanceStatus); err != nil {
			return MetadataOperationResult{}, false, err
		}
		_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: target.LibraryID, StartedAt: startedAt, SelectedCandidate: selectedCandidate})
		return result, true, err
	}
	detailProviders := prioritizeProviderInstances(plan.DetailProviders, selectedSearchProvider.Record.Name)
	var detail NormalizedMetadataDetail
	detailAttempts, _, err := executeMetadataProviderStage(ctx, "detail", detailProviders, func(ctx context.Context, provider settings.ResolvedMetadataProviderInstance) (MetadataProviderAttempt, bool, error) {
		provider = providerWithOperationLanguage(provider, plan)
		switch provider.Record.ProviderType {
		case database.MetadataProviderTypeTMDB:
			if selectedCandidate.Provider != database.MetadataProviderTypeTMDB {
				return metadataProviderAttemptForProvider("detail", provider, ProviderAttemptOutcomeSkippedUnsupported), false, nil
			}
			candidateMediaType, tmdbID, err := parseExternalID(selectedCandidate.ExternalID)
			if err != nil {
				return metadataProviderFailureAttempt("detail", provider, err), false, err
			}
			normalized, attempt, err := s.executeTMDBDetailStage(ctx, provider, candidateMediaType, tmdbID)
			if err != nil {
				s.recordProviderFailure(ctx, provider, err)
				return attempt, false, err
			}
			detail = normalized
			return attempt, true, nil
		case database.MetadataProviderTypeMetaTube:
			if selectedCandidate.Provider != database.MetadataProviderTypeMetaTube {
				return metadataProviderAttemptForProvider("detail", provider, ProviderAttemptOutcomeSkippedUnsupported), false, nil
			}
			upstreamProvider, upstreamID, err := parseMetaTubeExternalID(selectedCandidate.ExternalID)
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
	if detail.Provider == database.MetadataProviderTypeTMDB && detail.ProviderType == "tv" {
		provider := providerWithOperationLanguage(*selectedSearchProvider, plan)
		if err := s.completeTMDBTVHierarchy(ctx, provider, &detail); err != nil {
			s.recordProviderFailure(ctx, provider, err)
			return MetadataOperationResult{}, false, err
		}
		result.ProviderAttempts = append(result.ProviderAttempts, hierarchyProviderAttempt(provider, detail.Hierarchy))
	}

	return s.applyNormalizedMatchDetail(ctx, startedAt, target, plan, result, detail, selectedCandidate, selectedCandidate.Confidence)
}

func (s *Service) runMatchFromExistingIdentity(ctx context.Context, startedAt time.Time, origin database.CatalogItem, target database.CatalogItem, plan MetadataExecutionPlan, result MetadataOperationResult, mediaType string) (MetadataOperationResult, bool, error) {
	preferredInstance, externalID, confidence, err := s.loadCatalogTMDBIdentity(ctx, target.ID, mediaType)
	if err != nil || strings.TrimSpace(externalID) == "" {
		return MetadataOperationResult{}, false, nil
	}
	if confidence > 0 && confidence < 0.85 {
		return MetadataOperationResult{}, false, nil
	}
	_, tmdbID, err := parseExternalID(externalID)
	if err != nil {
		return MetadataOperationResult{}, false, nil
	}
	var detail NormalizedMetadataDetail
	detailProviders := prioritizeProviderInstances(plan.DetailProviders, preferredInstance)
	detailAttempts, _, err := executeMetadataProviderStage(ctx, "detail", detailProviders, func(ctx context.Context, provider settings.ResolvedMetadataProviderInstance) (MetadataProviderAttempt, bool, error) {
		provider = providerWithOperationLanguage(provider, plan)
		if provider.Record.ProviderType != database.MetadataProviderTypeTMDB {
			return metadataProviderAttemptForProvider("detail", provider, ProviderAttemptOutcomeSkippedUnsupported), false, nil
		}
		normalized, attempt, err := s.executeTMDBDetailStage(ctx, provider, mediaType, tmdbID)
		if err != nil {
			s.recordProviderFailure(ctx, provider, err)
			return attempt, false, err
		}
		detail = normalized
		return attempt, true, nil
	})
	result.ProviderAttempts = append(result.ProviderAttempts, detailAttempts...)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	if strings.TrimSpace(detail.ExternalID) == "" {
		return MetadataOperationResult{}, false, nil
	}
	if detail.Provider == database.MetadataProviderTypeTMDB && detail.ProviderType == "tv" {
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
	return s.applyNormalizedMatchDetail(ctx, startedAt, target, plan, result, detail, selectedCandidate, confidence)
}

func (s *Service) applyNormalizedMatchDetail(ctx context.Context, startedAt time.Time, target database.CatalogItem, plan MetadataExecutionPlan, result MetadataOperationResult, detail NormalizedMetadataDetail, selectedCandidate NormalizedMetadataCandidate, confidence float64) (MetadataOperationResult, bool, error) {
	status := OperationStatusApplied
	governanceStatus := governanceStatusForMetadataOperation(OperationTypeMatch, status, confidence)
	if governanceStatus == catalog.GovernanceNeedsReview {
		status = OperationStatusNeedsReview
	}
	source, err := s.recordNormalizedProviderSource(ctx, target, plan, detail, confidence)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	sourceID := source.ID
	changes := normalizedDetailFieldChanges(target.ID, detail, &sourceID, FieldApplyModeAutomated, &confidence)
	changes = append(changes, MetadataFieldChange{ItemID: target.ID, FieldKey: "governance_status", Value: governanceStatus, SourceID: &sourceID, ApplyMode: FieldApplyModeAutomated, Confidence: &confidence})
	applied, skipped, err := s.applyMetadataFieldChanges(ctx, changes)
	if err != nil {
		return MetadataOperationResult{}, false, err
	}
	if err := applyNormalizedExternalIDs(ctx, catalog.NewService(s.db), target.ID, detail.ExternalIDs, "metadata_match", &confidence); err != nil {
		return MetadataOperationResult{}, false, err
	}
	if err := s.applyNormalizedTags(ctx, target.ID, detail.Tags, &sourceID); err != nil {
		return MetadataOperationResult{}, false, err
	}
	applied = append(applied, appliedTagFields(target.ID, detail.Tags, &sourceID, FieldApplyModeAutomated, &confidence)...)
	if err := s.applyNormalizedImages(ctx, target.ID, detail.Images, false, &sourceID); err != nil {
		return MetadataOperationResult{}, false, err
	}
	if err := s.applyNormalizedPeople(ctx, target.ID, detail.People, &sourceID); err != nil {
		return MetadataOperationResult{}, false, err
	}
	if target.Type == catalog.ItemTypeSeries && detail.Hierarchy != nil {
		hierarchyProvider := detailProviderForResult(plan.DetailProviders, detail.Provider)
		if hierarchyProvider.Record.ID != 0 {
			hierarchyResult, err := s.applyNormalizedTVHierarchy(ctx, target, resolvedProfileFromPlan(plan), hierarchyProvider, *detail.Hierarchy, governanceStatus, confidence, false)
			if err != nil {
				return MetadataOperationResult{}, false, err
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
		return MetadataOperationResult{}, false, err
	}
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: target.LibraryID, SelectedCandidate: selectedCandidate, StartedAt: startedAt})
	return result, true, err
}

func providerExternalID(providerType string, id int) string {
	if id <= 0 {
		return ""
	}
	providerType = strings.TrimSpace(providerType)
	if providerType == "" {
		providerType = "tv"
	}
	return fmt.Sprintf("%s:%d", providerType, id)
}

func (s *Service) completeTMDBTVHierarchy(ctx context.Context, provider settings.ResolvedMetadataProviderInstance, detail *NormalizedMetadataDetail) error {
	if detail == nil || detail.Hierarchy == nil {
		return nil
	}
	seriesID := externalIDNumber(detail.ExternalID)
	if seriesID <= 0 {
		return nil
	}
	for index := range detail.Hierarchy.Seasons {
		season := &detail.Hierarchy.Seasons[index]
		season.SeriesExternalID = detail.ExternalID
		if season.SeasonNumber <= 0 {
			continue
		}
		seasonDetail, err := s.fetchTVSeason(ctx, provider.TMDB, seriesID, season.SeasonNumber)
		if err != nil {
			return err
		}
		season.AirDate = strings.TrimSpace(seasonDetail.AirDate)
		season.ExternalIDs = tmdbSeasonExternalIDs(season.ExternalID, seasonDetail)
		for index, person := range extractSeasonCast(seasonDetail, provider.TMDB, 8) {
			season.People = append(season.People, NormalizedMetadataPerson{Name: person.Name, Role: "actor", Character: person.Role, SortOrder: index, AvatarURL: person.AvatarURL, TMDBPersonID: person.TMDBPersonID})
		}
		for index, person := range extractSeasonDirectors(seasonDetail, provider.TMDB) {
			season.People = append(season.People, NormalizedMetadataPerson{Name: person.Name, Role: "director", Character: person.Role, SortOrder: index, AvatarURL: person.AvatarURL, TMDBPersonID: person.TMDBPersonID})
		}
		for _, episode := range seasonDetail.Episodes {
			runtimeSeconds := runtimeSecondsFromMinutes(episode.Runtime)
			normalizedEpisode := NormalizedMetadataEpisode{SeriesExternalID: detail.ExternalID, ProviderType: "tv_episode", ExternalID: tmdbExternalID(episode.ID), SeasonNumber: episode.SeasonNumber, EpisodeNumber: episode.EpisodeNumber, Title: strings.TrimSpace(episode.Name), Overview: strings.TrimSpace(episode.Overview), AirDate: strings.TrimSpace(episode.AirDate), RuntimeSeconds: runtimeSeconds, CommunityRating: communityRatingFromEpisode(episode), StillURL: imageURL(provider.TMDB, episode.StillPath), StillPath: strings.TrimSpace(episode.StillPath)}
			for index, person := range extractEpisodeCast(episode, provider.TMDB, 12) {
				normalizedEpisode.People = append(normalizedEpisode.People, NormalizedMetadataPerson{Name: person.Name, Role: "actor", Character: person.Role, SortOrder: index, AvatarURL: person.AvatarURL, TMDBPersonID: person.TMDBPersonID})
			}
			for index, person := range extractEpisodeDirectors(episode, provider.TMDB) {
				normalizedEpisode.People = append(normalizedEpisode.People, NormalizedMetadataPerson{Name: person.Name, Role: "director", Character: person.Role, SortOrder: index, AvatarURL: person.AvatarURL, TMDBPersonID: person.TMDBPersonID})
			}
			season.Episodes = append(season.Episodes, normalizedEpisode)
		}
	}
	return nil
}

func detailProviderForResult(providers []settings.ResolvedMetadataProviderInstance, providerType string) settings.ResolvedMetadataProviderInstance {
	for _, provider := range providers {
		if provider.Record.ProviderType == providerType {
			return provider
		}
	}
	return settings.ResolvedMetadataProviderInstance{}
}

func resolvedProfileFromPlan(plan MetadataExecutionPlan) settings.ResolvedLibraryMetadataProfile {
	return settings.ResolvedLibraryMetadataProfile{Profile: database.MetadataProfile{ID: derefUintForOperation(plan.MetadataProfileID), Name: plan.MetadataProfileName}}
}

func (s *Service) hasLocalScannerEvidence(ctx context.Context, itemID uint) bool {
	_, err := s.loadLocalScannerEvidence(ctx, itemID)
	return err == nil
}

func (s *Service) recordNormalizedProviderSource(ctx context.Context, item database.CatalogItem, plan MetadataExecutionPlan, detail NormalizedMetadataDetail, confidence float64) (database.MetadataSource, error) {
	payload, err := json.Marshal(map[string]any{"provider": detail.Provider, "provider_type": detail.ProviderType, "external_id": detail.ExternalID, "confidence": confidence, "matched_title": detail.Title})
	if err != nil {
		return database.MetadataSource{}, err
	}
	var providerInstanceID *uint
	providerInstanceName := ""
	for _, provider := range plan.DetailProviders {
		if provider.Record.ProviderType == detail.Provider {
			providerInstanceID = &provider.Record.ID
			providerInstanceName = provider.Record.Name
			break
		}
	}
	return catalog.NewService(s.db).RecordMetadataSource(ctx, catalog.MetadataSourceInput{ItemID: item.ID, SourceType: catalog.SourceTypeProvider, SourceName: detail.Provider, ExternalID: detail.ExternalID, MetadataProfileID: plan.MetadataProfileID, MetadataProfileName: plan.MetadataProfileName, ProviderInstanceID: providerInstanceID, ProviderInstanceName: providerInstanceName, PayloadJSON: string(payload), Confidence: &confidence})
}

func (s *Service) recordNormalizedLocalScanSource(ctx context.Context, item database.CatalogItem, plan MetadataExecutionPlan, provider settings.ResolvedMetadataProviderInstance, evidence LocalScannerEvidence, detail NormalizedMetadataDetail, confidence float64) (database.MetadataSource, error) {
	payload, err := json.Marshal(map[string]any{"scanner_source_id": evidence.Source.ID, "sidecars": evidence.Sidecars, "external_ids": detail.ExternalIDs, "matched_title": detail.Title})
	if err != nil {
		return database.MetadataSource{}, err
	}
	providerID := provider.Record.ID
	return catalog.NewService(s.db).RecordMetadataSource(ctx, catalog.MetadataSourceInput{ItemID: item.ID, SourceType: catalog.SourceTypeProvider, SourceName: database.MetadataProviderTypeLocalScan, MetadataProfileID: plan.MetadataProfileID, MetadataProfileName: plan.MetadataProfileName, ProviderInstanceID: &providerID, ProviderInstanceName: provider.Record.Name, PayloadJSON: string(payload), Confidence: &confidence})
}

func providerWithOperationLanguage(provider settings.ResolvedMetadataProviderInstance, plan MetadataExecutionPlan) settings.ResolvedMetadataProviderInstance {
	if language := strings.TrimSpace(plan.PreferredMetadataLanguage); language != "" {
		provider.TMDB.Language = language
	}
	return provider
}

func prioritizeProviderInstances(providers []settings.ResolvedMetadataProviderInstance, preferredName string) []settings.ResolvedMetadataProviderInstance {
	preferredName = strings.TrimSpace(preferredName)
	if preferredName == "" || len(providers) < 2 {
		return providers
	}
	ordered := make([]settings.ResolvedMetadataProviderInstance, 0, len(providers))
	for _, provider := range providers {
		if provider.Record.Name == preferredName {
			ordered = append(ordered, provider)
		}
	}
	for _, provider := range providers {
		if provider.Record.Name != preferredName {
			ordered = append(ordered, provider)
		}
	}
	return ordered
}
