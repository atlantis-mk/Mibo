package catalog

import (
	"context"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog/seriesplayback"
	"github.com/atlan/mibo-media-server/internal/database"
)

func (s *Service) ListChildItems(ctx context.Context, parentID uint, typeFilter string, availabilityFilter string) ([]CatalogListItem, error) {
	if _, err := s.loadCatalogItem(ctx, parentID); err != nil {
		return nil, err
	}

	query := s.db.WithContext(ctx).
		Where("parent_id = ? AND deleted_at IS NULL", parentID)
	if normalizedType := normalizeCatalogQueryTypeFilter(typeFilter); normalizedType != "" {
		query = query.Where("type = ?", normalizedType)
	}
	if allowed := normalizeCatalogAvailabilityFilterList(availabilityFilter); len(allowed) > 0 {
		query = query.Where("availability_status IN ?", allowed)
	}

	var items []database.CatalogItem
	if err := query.
		Order("parent_index_number asc").
		Order("index_number asc").
		Order("sort_key asc").
		Order("title asc").
		Order("id asc").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return s.buildCatalogListItems(ctx, items)
}

func (s *Service) ListSeriesEpisodes(ctx context.Context, seriesID uint, seasonNumber *int, availabilityFilter string) ([]CatalogEpisodeDetail, error) {
	series, err := s.loadCatalogItem(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	if series.Type != ItemTypeSeries {
		return []CatalogEpisodeDetail{}, nil
	}

	seasonQuery := s.db.WithContext(ctx).
		Where("parent_id = ? AND type = ? AND deleted_at IS NULL", series.ID, ItemTypeSeason)
	if seasonNumber != nil {
		seasonQuery = seasonQuery.Where("index_number = ?", *seasonNumber)
	}

	var seasons []database.CatalogItem
	if err := seasonQuery.Order("index_number asc").Order("id asc").Find(&seasons).Error; err != nil {
		return nil, err
	}
	if len(seasons) == 0 {
		return []CatalogEpisodeDetail{}, nil
	}

	seasonIDs := make([]uint, 0, len(seasons))
	for _, season := range seasons {
		seasonIDs = append(seasonIDs, season.ID)
	}

	byParent, err := s.buildCatalogEpisodeDetailsByParent(ctx, seasonIDs)
	if err != nil {
		return nil, err
	}
	allowedAvailability := normalizeCatalogAvailabilityFilterSet(availabilityFilter)
	result := make([]CatalogEpisodeDetail, 0)
	for _, season := range seasons {
		for _, episode := range byParent[season.ID] {
			if len(allowedAvailability) > 0 {
				if _, ok := allowedAvailability[strings.TrimSpace(episode.AvailabilityStatus)]; !ok {
					continue
				}
			}
			result = append(result, episode)
		}
	}
	return result, nil
}

func (s *Service) ListSeriesMissingEpisodes(ctx context.Context, seriesID uint) ([]CatalogEpisodeDetail, error) {
	return s.ListSeriesEpisodes(ctx, seriesID, nil, AvailabilityMissing)
}

func (s *Service) GetSeriesNextUp(ctx context.Context, userID uint, seriesID uint) (*CatalogEpisodeDetail, error) {
	target, err := seriesplayback.Select(ctx, s.db, seriesID, &userID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, nil
	}
	detail, err := s.GetItemDetailForUser(ctx, target.EpisodeID, &userID)
	if err != nil {
		return nil, err
	}
	episode := CatalogEpisodeDetail{
		ID:                 detail.ID,
		LibraryID:          detail.LibraryID,
		Type:               detail.Type,
		Title:              detail.Title,
		Overview:           detail.Overview,
		Year:               detail.Year,
		ParentIndexNumber:  detail.EpisodeContext.SeasonNumber,
		IndexNumber:        detail.EpisodeContext.EpisodeNumber,
		IndexNumberEnd:     detail.EpisodeContext.EpisodeNumberEnd,
		RuntimeSeconds:     detail.RuntimeSeconds,
		AvailabilityStatus: detail.AvailabilityStatus,
		GovernanceStatus:   detail.GovernanceStatus,
		ReleaseDate:        detail.ReleaseDate,
		FirstAirDate:       detail.FirstAirDate,
		SelectedImages:     detail.SelectedImages,
		ExternalIdentities: detail.ExternalIdentities,
		SourceEvidence:     detail.SourceEvidence,
		FieldStates:        detail.FieldStates,
		Assets:             detail.Assets,
	}
	return &episode, nil
}

func normalizeCatalogQueryTypeFilter(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "show" {
		return ItemTypeSeries
	}
	return trimmed
}

func normalizeCatalogAvailabilityFilterList(value string) []string {
	set := normalizeCatalogAvailabilityFilterSet(value)
	if len(set) == 0 {
		return nil
	}
	values := make([]string, 0, len(set))
	for _, candidate := range []string{AvailabilityAvailable, AvailabilityMissing, AvailabilityUnaired, AvailabilityNoLocalMedia} {
		if _, ok := set[candidate]; ok {
			values = append(values, candidate)
		}
	}
	return values
}

func normalizeCatalogAvailabilityFilterSet(value string) map[string]struct{} {
	allowed := make(map[string]struct{})
	for _, rawPart := range strings.Split(value, ",") {
		switch strings.TrimSpace(rawPart) {
		case AvailabilityAvailable, AvailabilityMissing, AvailabilityUnaired, AvailabilityNoLocalMedia:
			allowed[strings.TrimSpace(rawPart)] = struct{}{}
		}
	}
	return allowed
}
