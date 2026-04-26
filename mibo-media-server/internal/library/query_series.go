package library

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

func (s *Service) ListSeriesEpisodes(ctx context.Context, mediaItemID uint) ([]SeriesSeasonDetail, error) {
	var anchor database.MediaItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", mediaItemID).First(&anchor).Error; err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(anchor.Type), "show") && !strings.EqualFold(strings.TrimSpace(anchor.Type), "episode") {
		return []SeriesSeasonDetail{}, nil
	}

	anchorKey := browseShowKey(anchor)
	query := s.db.WithContext(ctx).Where("library_id = ? AND type = ? AND deleted_at IS NULL", anchor.LibraryID, "episode")
	if externalID := strings.TrimSpace(anchor.ExternalID); externalID != "" {
		query = query.Where("external_id = ?", externalID)
	} else {
		seriesTitle := strings.TrimSpace(anchor.SeriesTitle)
		if seriesTitle == "" {
			seriesTitle = strings.TrimSpace(anchor.Title)
		}
		queryToken := normalizeSeriesGroupingTitle(seriesTitle)
		query = query.Where("LOWER(series_title) LIKE ? OR LOWER(title) LIKE ?", likeToken(queryToken), likeToken(queryToken))
	}

	var items []database.MediaItem
	if err := query.Order("season_number asc, episode_number asc, id asc").Find(&items).Error; err != nil {
		return nil, err
	}

	filtered := make([]database.MediaItem, 0, len(items))
	for _, item := range items {
		if browseShowKey(item) == anchorKey {
			filtered = append(filtered, item)
		}
	}
	if len(filtered) == 0 {
		return []SeriesSeasonDetail{}, nil
	}

	seasonsByNumber := make(map[int]*SeriesSeasonDetail)
	seasonNumbers := make([]int, 0, len(filtered))
	for _, item := range filtered {
		seasonNumber := normalizeOrdinal(item.SeasonNumber)
		season, exists := seasonsByNumber[seasonNumber]
		if !exists {
			season = &SeriesSeasonDetail{
				SeasonNumber: seasonNumber,
				Name:         formatLocalSeasonName(seasonNumber),
				PosterURL:    strings.TrimSpace(item.PosterURL),
				Episodes:     []SeriesEpisodeDetail{},
			}
			seasonsByNumber[seasonNumber] = season
			seasonNumbers = append(seasonNumbers, seasonNumber)
		}
		if season.PosterURL == "" {
			season.PosterURL = strings.TrimSpace(item.PosterURL)
		}
		if season.RuntimeSeconds == nil && item.RuntimeSeconds != nil {
			runtime := *item.RuntimeSeconds
			season.RuntimeSeconds = &runtime
		}
		season.Episodes = append(season.Episodes, SeriesEpisodeDetail{
			MediaItemID:    item.ID,
			SeasonNumber:   seasonNumber,
			EpisodeNumber:  normalizeOrdinal(item.EpisodeNumber),
			Name:           strings.TrimSpace(item.Title),
			AirDate:        strings.TrimSpace(item.ReleaseDate),
			Overview:       strings.TrimSpace(item.Overview),
			StillURL:       localSeriesEpisodeStillURL(item),
			RuntimeSeconds: item.RuntimeSeconds,
		})
	}

	sort.Ints(seasonNumbers)
	seasons := make([]SeriesSeasonDetail, 0, len(seasonNumbers))
	for _, seasonNumber := range seasonNumbers {
		season := seasonsByNumber[seasonNumber]
		sort.SliceStable(season.Episodes, func(i, j int) bool {
			if season.Episodes[i].EpisodeNumber != season.Episodes[j].EpisodeNumber {
				return season.Episodes[i].EpisodeNumber < season.Episodes[j].EpisodeNumber
			}
			return season.Episodes[i].MediaItemID < season.Episodes[j].MediaItemID
		})
		seasons = append(seasons, *season)
	}

	return seasons, nil
}

func formatLocalSeasonName(seasonNumber int) string {
	if seasonNumber <= 0 {
		return "未识别季"
	}
	return "第 " + strconv.Itoa(seasonNumber) + " 季"
}

func localSeriesEpisodeStillURL(item database.MediaItem) string {
	if backdropURL := strings.TrimSpace(item.BackdropURL); backdropURL != "" {
		return backdropURL
	}
	return strings.TrimSpace(item.PosterURL)
}
