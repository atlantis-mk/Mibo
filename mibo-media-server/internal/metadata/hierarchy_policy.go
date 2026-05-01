package metadata

import (
	"context"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/settings"
)

type MetadataHierarchyApplyResult struct {
	AffectedItemIDs   []uint
	MetadataSourceIDs []uint
	AppliedFields     []MetadataAppliedField
	SkippedFields     []MetadataSkippedField
}

func (s *Service) applyNormalizedTVHierarchy(ctx context.Context, seriesItem database.CatalogItem, profile settings.ResolvedLibraryMetadataProfile, provider settings.ResolvedMetadataProviderInstance, hierarchy NormalizedMetadataHierarchy, governanceStatus string, confidence float64, forceSelectImages bool) (MetadataHierarchyApplyResult, error) {
	result := MetadataHierarchyApplyResult{AffectedItemIDs: []uint{seriesItem.ID}}
	catalogSvc := catalog.NewService(s.db)
	fallback := []settings.MetadataExecutionFallbackSummary{{Stage: "hierarchy", Selected: provider.Record.Name}}
	for _, season := range hierarchy.Seasons {
		seasonItem, err := s.findOrCreateCatalogSeasonItem(ctx, catalogSvc, seriesItem, season.SeasonNumber, season.Title)
		if err != nil {
			return MetadataHierarchyApplyResult{}, err
		}
		result.AffectedItemIDs = append(result.AffectedItemIDs, seasonItem.ID)
		seasonSource, err := s.syncCatalogHierarchyIdentity(ctx, seasonItem.ID, profile, provider, fallback, firstNonEmpty(season.ProviderType, "tv_season"), externalIDNumber(season.ExternalID), confidence, map[string]any{"external_id": season.ExternalID, "series_tmdb_id": externalIDNumber(season.SeriesExternalID), "season_number": season.SeasonNumber, "matched_title": season.Title, "poster_path": season.PosterPath})
		if err != nil {
			return MetadataHierarchyApplyResult{}, err
		}
		applied, skipped, err := s.applyCatalogHierarchyFields(ctx, catalogSvc, seasonItem.ID, season.Title, season.Overview, parseYear(season.AirDate), nil, governanceStatus, &seasonSource.ID, &confidence)
		if err != nil {
			return MetadataHierarchyApplyResult{}, err
		}
		result.AppliedFields = append(result.AppliedFields, applied...)
		result.SkippedFields = append(result.SkippedFields, skipped...)
		if airDate := parseProviderDate(season.AirDate); airDate != nil {
			_, didApply, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: seasonItem.ID, FieldKey: "first_air_date", Value: *airDate, SourceID: &seasonSource.ID})
			if err != nil {
				return MetadataHierarchyApplyResult{}, err
			}
			if didApply {
				result.AppliedFields = append(result.AppliedFields, MetadataAppliedField{ItemID: seasonItem.ID, FieldKey: "first_air_date", SourceID: &seasonSource.ID, ApplyMode: FieldApplyModeAutomated, Confidence: &confidence})
			} else {
				result.SkippedFields = append(result.SkippedFields, MetadataSkippedField{ItemID: seasonItem.ID, FieldKey: "first_air_date", Reason: "not_applied"})
			}
		}
		if len(season.ExternalIDs) == 0 {
			season.ExternalIDs = []NormalizedMetadataExternalID{{Provider: "tmdb", ProviderType: firstNonEmpty(season.ProviderType, "tv_season"), ExternalID: season.ExternalID, IsPrimary: true}}
		}
		if err := applyNormalizedExternalIDs(ctx, catalogSvc, seasonItem.ID, season.ExternalIDs, "metadata_match", &confidence); err != nil {
			return MetadataHierarchyApplyResult{}, err
		}
		if seasonSource.ID != 0 {
			result.MetadataSourceIDs = append(result.MetadataSourceIDs, seasonSource.ID)
		}
		if err := s.upsertCatalogImageCandidate(ctx, seasonItem.ID, "poster", season.PosterURL, "", 0, true, forceSelectImages, &seasonSource.ID); err != nil {
			return MetadataHierarchyApplyResult{}, err
		}
		if err := s.applyNormalizedPeople(ctx, seasonItem.ID, season.People, &seasonSource.ID); err != nil {
			return MetadataHierarchyApplyResult{}, err
		}
		for _, episode := range season.Episodes {
			episodeItem, err := s.findOrCreateCatalogEpisodeItem(ctx, catalogSvc, seasonItem, season.SeasonNumber, episode.EpisodeNumber, episode.Title, episode.AirDate)
			if err != nil {
				return MetadataHierarchyApplyResult{}, err
			}
			result.AffectedItemIDs = append(result.AffectedItemIDs, episodeItem.ID)
			episodeSource, err := s.syncCatalogHierarchyIdentity(ctx, episodeItem.ID, profile, provider, fallback, firstNonEmpty(episode.ProviderType, "tv_episode"), externalIDNumber(episode.ExternalID), confidence, map[string]any{"external_id": episode.ExternalID, "series_tmdb_id": externalIDNumber(episode.SeriesExternalID), "season_number": episode.SeasonNumber, "episode_number": episode.EpisodeNumber, "matched_title": episode.Title, "air_date": episode.AirDate, "overview": episode.Overview, "still_path": episode.StillPath})
			if err != nil {
				return MetadataHierarchyApplyResult{}, err
			}
			applied, skipped, err := s.applyCatalogHierarchyFields(ctx, catalogSvc, episodeItem.ID, episode.Title, episode.Overview, parseYear(episode.AirDate), episode.RuntimeSeconds, governanceStatus, &episodeSource.ID, &confidence)
			if err != nil {
				return MetadataHierarchyApplyResult{}, err
			}
			result.AppliedFields = append(result.AppliedFields, applied...)
			result.SkippedFields = append(result.SkippedFields, skipped...)
			if airDate := parseProviderDate(episode.AirDate); airDate != nil {
				_, didApply, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: episodeItem.ID, FieldKey: "first_air_date", Value: *airDate, SourceID: &episodeSource.ID})
				if err != nil {
					return MetadataHierarchyApplyResult{}, err
				}
				if didApply {
					result.AppliedFields = append(result.AppliedFields, MetadataAppliedField{ItemID: episodeItem.ID, FieldKey: "first_air_date", SourceID: &episodeSource.ID, ApplyMode: FieldApplyModeAutomated, Confidence: &confidence})
				} else {
					result.SkippedFields = append(result.SkippedFields, MetadataSkippedField{ItemID: episodeItem.ID, FieldKey: "first_air_date", Reason: "not_applied"})
				}
			}
			if episode.CommunityRating != nil {
				_, didApply, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: episodeItem.ID, FieldKey: "community_rating", Value: *episode.CommunityRating, SourceID: &episodeSource.ID})
				if err != nil {
					return MetadataHierarchyApplyResult{}, err
				}
				if didApply {
					result.AppliedFields = append(result.AppliedFields, MetadataAppliedField{ItemID: episodeItem.ID, FieldKey: "community_rating", SourceID: &episodeSource.ID, ApplyMode: FieldApplyModeAutomated, Confidence: &confidence})
				} else {
					result.SkippedFields = append(result.SkippedFields, MetadataSkippedField{ItemID: episodeItem.ID, FieldKey: "community_rating", Reason: "not_applied"})
				}
			}
			if episodeSource.ID != 0 {
				result.MetadataSourceIDs = append(result.MetadataSourceIDs, episodeSource.ID)
			}
			if err := s.upsertCatalogImageCandidate(ctx, episodeItem.ID, "still", episode.StillURL, "", 0, true, forceSelectImages, &episodeSource.ID); err != nil {
				return MetadataHierarchyApplyResult{}, err
			}
			if err := s.applyNormalizedPeople(ctx, episodeItem.ID, episode.People, &episodeSource.ID); err != nil {
				return MetadataHierarchyApplyResult{}, err
			}
			availability, err := s.resolveCatalogLeafAvailability(ctx, episodeItem.ID, episode.AirDate)
			if err != nil {
				return MetadataHierarchyApplyResult{}, err
			}
			if err := s.updateCatalogAvailability(ctx, episodeItem.ID, availability); err != nil {
				return MetadataHierarchyApplyResult{}, err
			}
		}
		seasonAvailability, err := s.resolveCatalogParentAvailability(ctx, seasonItem.ID)
		if err != nil {
			return MetadataHierarchyApplyResult{}, err
		}
		if err := s.updateCatalogAvailability(ctx, seasonItem.ID, seasonAvailability); err != nil {
			return MetadataHierarchyApplyResult{}, err
		}
	}
	seriesAvailability, err := s.resolveCatalogParentAvailability(ctx, seriesItem.ID)
	if err != nil {
		return MetadataHierarchyApplyResult{}, err
	}
	if err := s.updateCatalogAvailability(ctx, seriesItem.ID, seriesAvailability); err != nil {
		return MetadataHierarchyApplyResult{}, err
	}
	return result, nil
}

func externalIDNumber(value string) int {
	trimmed := strings.TrimSpace(value)
	if idx := strings.LastIndex(trimmed, ":"); idx >= 0 {
		trimmed = trimmed[idx+1:]
	}
	parsed, _ := strconv.Atoi(trimmed)
	return parsed
}
