package metadata

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

type MetadataHierarchyApplyResult struct {
	AffectedMetadataItemIDs []uint
	MetadataSourceIDs       []uint
	AppliedFields           []MetadataAppliedField
	SkippedFields           []MetadataSkippedField
}

func (s *Service) applyNormalizedMetadataItemTVHierarchy(ctx context.Context, seriesItem database.MetadataItem, plan MetadataExecutionPlan, hierarchy NormalizedMetadataHierarchy, governanceStatus string, confidence float64, forceSelectImages bool) (MetadataHierarchyApplyResult, error) {
	result := MetadataHierarchyApplyResult{AffectedMetadataItemIDs: []uint{seriesItem.ID}}
	for _, season := range hierarchy.Seasons {
		seasonItem, err := s.findOrCreateMetadataSeasonItem(ctx, seriesItem, season)
		if err != nil {
			return MetadataHierarchyApplyResult{}, err
		}
		result.AffectedMetadataItemIDs = appendUniqueUint(result.AffectedMetadataItemIDs, seasonItem.ID)
		seasonSource, err := s.recordMetadataItemHierarchySource(ctx, seasonItem.ID, plan, firstNonEmpty(season.ProviderType, "tv_season"), season.ExternalID, confidence, map[string]any{"external_id": season.ExternalID, "series_external_id": season.SeriesExternalID, "season_number": season.SeasonNumber, "matched_title": season.Title, "poster_path": season.PosterPath})
		if err != nil {
			return MetadataHierarchyApplyResult{}, err
		}
		result.MetadataSourceIDs = appendUniqueUint(result.MetadataSourceIDs, seasonSource.ID)
		seasonDetail := NormalizedMetadataDetail{Title: season.Title, Overview: season.Overview, FirstAirDate: season.AirDate, Year: parseYear(season.AirDate)}
		changes := normalizedDetailFieldChanges(seasonItem.ID, seasonDetail, &seasonSource.ID, FieldApplyModeAutomated, &confidence)
		changes = append(changes, MetadataFieldChange{ItemID: seasonItem.ID, FieldKey: "governance_status", Value: governanceStatus, SourceID: &seasonSource.ID, ApplyMode: FieldApplyModeAutomated, Confidence: &confidence})
		applied, skipped, err := s.applyMetadataItemFieldChanges(ctx, changes)
		if err != nil {
			return MetadataHierarchyApplyResult{}, err
		}
		result.AppliedFields = append(result.AppliedFields, applied...)
		result.SkippedFields = append(result.SkippedFields, skipped...)
		seasonExternalIDs := season.ExternalIDs
		if len(seasonExternalIDs) == 0 && strings.TrimSpace(season.ExternalID) != "" {
			seasonExternalIDs = []NormalizedMetadataExternalID{{Provider: "tmdb", ProviderType: firstNonEmpty(season.ProviderType, "tv_season"), ExternalID: season.ExternalID, IsPrimary: true}}
		}
		if err := s.applyNormalizedMetadataItemExternalIDs(ctx, seasonItem.ID, seasonExternalIDs, "metadata_hierarchy", &confidence); err != nil {
			return MetadataHierarchyApplyResult{}, err
		}
		if err := s.applyNormalizedMetadataItemImages(ctx, seasonItem.ID, []NormalizedMetadataImage{{ImageType: "poster", URL: season.PosterURL, Selected: true}}, forceSelectImages, &seasonSource.ID); err != nil {
			return MetadataHierarchyApplyResult{}, err
		}
		if err := s.applyNormalizedMetadataItemPeople(ctx, seasonItem.ID, season.People, &seasonSource.ID); err != nil {
			return MetadataHierarchyApplyResult{}, err
		}
		for _, episode := range season.Episodes {
			episodeItem, err := s.findOrCreateMetadataEpisodeItem(ctx, seriesItem, seasonItem, episode)
			if err != nil {
				return MetadataHierarchyApplyResult{}, err
			}
			result.AffectedMetadataItemIDs = appendUniqueUint(result.AffectedMetadataItemIDs, episodeItem.ID)
			episodeSource, err := s.recordMetadataItemHierarchySource(ctx, episodeItem.ID, plan, firstNonEmpty(episode.ProviderType, "tv_episode"), episode.ExternalID, confidence, map[string]any{"external_id": episode.ExternalID, "series_external_id": episode.SeriesExternalID, "season_number": episode.SeasonNumber, "episode_number": episode.EpisodeNumber, "matched_title": episode.Title, "still_path": episode.StillPath})
			if err != nil {
				return MetadataHierarchyApplyResult{}, err
			}
			result.MetadataSourceIDs = appendUniqueUint(result.MetadataSourceIDs, episodeSource.ID)
			episodeDetail := NormalizedMetadataDetail{Title: episode.Title, Overview: episode.Overview, FirstAirDate: episode.AirDate, RuntimeSeconds: episode.RuntimeSeconds, CommunityRating: episode.CommunityRating, Year: parseYear(episode.AirDate)}
			changes := normalizedDetailFieldChanges(episodeItem.ID, episodeDetail, &episodeSource.ID, FieldApplyModeAutomated, &confidence)
			changes = append(changes, MetadataFieldChange{ItemID: episodeItem.ID, FieldKey: "governance_status", Value: governanceStatus, SourceID: &episodeSource.ID, ApplyMode: FieldApplyModeAutomated, Confidence: &confidence})
			applied, skipped, err := s.applyMetadataItemFieldChanges(ctx, changes)
			if err != nil {
				return MetadataHierarchyApplyResult{}, err
			}
			result.AppliedFields = append(result.AppliedFields, applied...)
			result.SkippedFields = append(result.SkippedFields, skipped...)
			if strings.TrimSpace(episode.ExternalID) != "" {
				if err := s.applyNormalizedMetadataItemExternalIDs(ctx, episodeItem.ID, []NormalizedMetadataExternalID{{Provider: "tmdb", ProviderType: firstNonEmpty(episode.ProviderType, "tv_episode"), ExternalID: episode.ExternalID, IsPrimary: true}}, "metadata_hierarchy", &confidence); err != nil {
					return MetadataHierarchyApplyResult{}, err
				}
			}
			if err := s.applyNormalizedMetadataItemImages(ctx, episodeItem.ID, []NormalizedMetadataImage{{ImageType: "still", URL: episode.StillURL, Selected: true}}, forceSelectImages, &episodeSource.ID); err != nil {
				return MetadataHierarchyApplyResult{}, err
			}
			if err := s.applyNormalizedMetadataItemPeople(ctx, episodeItem.ID, episode.People, &episodeSource.ID); err != nil {
				return MetadataHierarchyApplyResult{}, err
			}
		}
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

func (s *Service) findOrCreateMetadataSeasonItem(ctx context.Context, seriesItem database.MetadataItem, season NormalizedMetadataSeason) (database.MetadataItem, error) {
	var item database.MetadataItem
	err := s.db.WithContext(ctx).Where("item_type = ? AND parent_id = ? AND index_number = ? AND deleted_at IS NULL", database.MetadataItemTypeSeason, seriesItem.ID, season.SeasonNumber).First(&item).Error
	if err == nil {
		return item, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return database.MetadataItem{}, err
	}
	title := strings.TrimSpace(season.Title)
	if title == "" {
		title = "Season " + strconv.Itoa(season.SeasonNumber)
	}
	rootID := seriesItem.ID
	indexNumber := season.SeasonNumber
	item = database.MetadataItem{ItemType: database.MetadataItemTypeSeason, ContentForm: seriesItem.ContentForm, ParentID: &seriesItem.ID, RootID: &rootID, Title: title, SortTitle: seriesItem.Title + " S" + zeroPad2(season.SeasonNumber), SortKey: seriesItem.Title + " S" + zeroPad2(season.SeasonNumber), IndexNumber: &indexNumber, GovernanceStatus: database.ReviewStatePending}
	if err := s.db.WithContext(ctx).Create(&item).Error; err != nil {
		return database.MetadataItem{}, err
	}
	return item, nil
}

func (s *Service) findOrCreateMetadataEpisodeItem(ctx context.Context, seriesItem database.MetadataItem, seasonItem database.MetadataItem, episode NormalizedMetadataEpisode) (database.MetadataItem, error) {
	var item database.MetadataItem
	err := s.db.WithContext(ctx).Where("item_type = ? AND parent_id = ? AND index_number = ? AND deleted_at IS NULL", database.MetadataItemTypeEpisode, seasonItem.ID, episode.EpisodeNumber).First(&item).Error
	if err == nil {
		return item, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return database.MetadataItem{}, err
	}
	title := strings.TrimSpace(episode.Title)
	if title == "" {
		title = seriesItem.Title + " S" + zeroPad2(episode.SeasonNumber) + "E" + zeroPad2(episode.EpisodeNumber)
	}
	rootID := seriesItem.ID
	indexNumber := episode.EpisodeNumber
	parentIndexNumber := episode.SeasonNumber
	item = database.MetadataItem{ItemType: database.MetadataItemTypeEpisode, ContentForm: seriesItem.ContentForm, ParentID: &seasonItem.ID, RootID: &rootID, Title: title, SortTitle: seriesItem.Title + " S" + zeroPad2(episode.SeasonNumber) + "E" + zeroPad2(episode.EpisodeNumber), SortKey: seriesItem.Title + " S" + zeroPad2(episode.SeasonNumber) + "E" + zeroPad2(episode.EpisodeNumber), IndexNumber: &indexNumber, ParentIndexNumber: &parentIndexNumber, GovernanceStatus: database.ReviewStatePending}
	if err := s.db.WithContext(ctx).Create(&item).Error; err != nil {
		return database.MetadataItem{}, err
	}
	return item, nil
}

func (s *Service) recordMetadataItemHierarchySource(ctx context.Context, metadataItemID uint, plan MetadataExecutionPlan, providerType string, externalID string, confidence float64, payload map[string]any) (database.MetadataItemSource, error) {
	payloadJSON := marshalOperationJSON(payload)
	source := database.MetadataItemSource{MetadataItemID: metadataItemID, SourceType: catalog.SourceTypeProvider, SourceName: "tmdb", Language: strings.TrimSpace(plan.PreferredMetadataLanguage), ExternalID: strings.TrimSpace(externalID), TriggeringLibraryID: &plan.LibraryID, MetadataProfileID: plan.MetadataProfileID, MetadataProfileName: strings.TrimSpace(plan.MetadataProfileName), PayloadJSON: payloadJSON, Confidence: &confidence, FetchedAt: time.Now().UTC()}
	_ = providerType
	if err := s.db.WithContext(ctx).Create(&source).Error; err != nil {
		return database.MetadataItemSource{}, err
	}
	return source, nil
}

func zeroPad2(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}
