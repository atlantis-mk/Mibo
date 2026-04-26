package metadata

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func (s *Service) tmdbConfig(ctx context.Context) (config.TMDBConfig, error) {
	if s.settings == nil {
		return s.fallback.TMDB, nil
	}
	resolved, _, err := s.settings.ResolveTMDBConfig(ctx)
	if err != nil {
		return config.TMDBConfig{}, err
	}
	return resolved, nil
}

func (s *Service) MatchItem(ctx context.Context, mediaItemID uint) error {
	tmdbCfg, err := s.tmdbConfig(ctx)
	if err != nil {
		return err
	}

	var item database.MediaItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", mediaItemID).First(&item).Error; err != nil {
		return err
	}

	if strings.TrimSpace(tmdbCfg.APIKey) == "" {
		return s.db.WithContext(ctx).Model(&database.MediaItem{}).Where("id = ?", item.ID).Updates(map[string]any{"match_status": StatusSkipped}).Error
	}

	mediaType := tmdbMediaType(item.Type)
	searchMatch, err := s.searchBestMatch(ctx, tmdbCfg, item, mediaType)
	if err != nil {
		return err
	}
	if searchMatch == nil {
		return s.db.WithContext(ctx).Model(&database.MediaItem{}).Where("id = ?", item.ID).Updates(map[string]any{
			"match_status":        StatusUnmatched,
			"metadata_provider":   "",
			"external_id":         "",
			"metadata_confidence": nil,
		}).Error
	}

	detail, err := s.fetchDetail(ctx, tmdbCfg, mediaType, searchMatch.result.ID)
	if err != nil {
		return err
	}
	episodeMatch, err := s.resolveTVEpisodeMatch(ctx, tmdbCfg, item, mediaType, detail.ID)
	if err != nil {
		return err
	}
	return s.applyDetail(ctx, item, tmdbCfg, mediaType, detail, searchMatch.confidence, episodeMatch)
}

func (s *Service) ApplyCandidate(ctx context.Context, mediaItemID uint, input ApplyCandidateInput) error {
	tmdbCfg, err := s.tmdbConfig(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(tmdbCfg.APIKey) == "" {
		return fmt.Errorf("tmdb 未配置，无法应用元数据")
	}

	var item database.MediaItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", mediaItemID).First(&item).Error; err != nil {
		return err
	}

	mediaType, id, err := parseExternalID(input.ExternalID)
	if err != nil {
		return err
	}
	detail, err := s.fetchDetail(ctx, tmdbCfg, mediaType, id)
	if err != nil {
		return err
	}

	confidence := 1.0
	episodeMatch, err := s.resolveTVEpisodeMatch(ctx, tmdbCfg, item, mediaType, detail.ID)
	if err != nil {
		return err
	}

	return s.applyDetail(ctx, item, tmdbCfg, mediaType, detail, confidence, episodeMatch)
}

func (s *Service) UpdateManualMetadata(ctx context.Context, mediaItemID uint, input ManualMetadataInput) error {
	var item database.MediaItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", mediaItemID).First(&item).Error; err != nil {
		return err
	}

	updates := map[string]any{
		"title":          strings.TrimSpace(input.Title),
		"original_title": strings.TrimSpace(input.OriginalTitle),
		"overview":       strings.TrimSpace(input.Overview),
		"poster_url":     strings.TrimSpace(input.PosterURL),
		"backdrop_url":   strings.TrimSpace(input.BackdropURL),
		"year":           input.Year,
	}

	if err := s.db.WithContext(ctx).Model(&database.MediaItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
		return err
	}
	if s.search != nil {
		return s.search.ReindexMediaItem(ctx, item.ID)
	}
	return nil
}

func (s *Service) RefetchItem(ctx context.Context, mediaItemID uint) error {
	tmdbCfg, err := s.tmdbConfig(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(tmdbCfg.APIKey) == "" {
		return fmt.Errorf("tmdb 未配置，无法重抓元数据")
	}

	var item database.MediaItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", mediaItemID).First(&item).Error; err != nil {
		return err
	}

	if provider := strings.TrimSpace(item.MetadataProvider); provider != "" && provider != "tmdb" {
		return fmt.Errorf("metadata_provider 不支持重抓")
	}
	if strings.TrimSpace(item.ExternalID) == "" {
		return fmt.Errorf("当前条目没有可重抓的匹配结果")
	}

	mediaType, externalID, err := parseExternalID(item.ExternalID)
	if err != nil {
		return err
	}
	detail, err := s.fetchDetail(ctx, tmdbCfg, mediaType, externalID)
	if err != nil {
		return err
	}

	confidence := 1.0
	if item.MetadataConfidence != nil && *item.MetadataConfidence > 0 {
		confidence = *item.MetadataConfidence
	}
	episodeMatch, err := s.resolveTVEpisodeMatch(ctx, tmdbCfg, item, mediaType, externalID)
	if err != nil {
		return err
	}
	if err := s.applyDetail(ctx, item, tmdbCfg, mediaType, detail, confidence, episodeMatch); err != nil {
		return err
	}
	if mediaType == "tv" {
		if err := s.refreshTVMetadataCaches(ctx, tmdbCfg, item, externalID); err != nil {
			return err
		}
	}
	return nil
}

type tvEpisodeMatch struct {
	seasonFound  bool
	episodeFound bool
	season       seasonDetailResponse
	episode      seasonEpisodeResponse
}

func (s *Service) resolveTVEpisodeMatch(ctx context.Context, tmdbCfg config.TMDBConfig, item database.MediaItem, mediaType string, seriesTMDBID int) (*tvEpisodeMatch, error) {
	if mediaType != "tv" || item.Type != "episode" || item.SeasonNumber == nil || item.EpisodeNumber == nil {
		return nil, nil
	}
	seasonDetail, err := s.fetchTVSeason(ctx, tmdbCfg, seriesTMDBID, *item.SeasonNumber)
	if err != nil {
		var tmdbFailure tmdbRequestFailure
		if errors.As(err, &tmdbFailure) && tmdbFailure.StatusCode() == 404 {
			return &tvEpisodeMatch{}, nil
		}
		return nil, err
	}
	match := &tvEpisodeMatch{seasonFound: true, season: seasonDetail}
	for _, episode := range seasonDetail.Episodes {
		if episode.EpisodeNumber == *item.EpisodeNumber {
			match.episodeFound = true
			match.episode = episode
			break
		}
	}
	return match, nil
}

func (s *Service) applyDetail(ctx context.Context, item database.MediaItem, tmdbCfg config.TMDBConfig, mediaType string, detail detailResponse, confidence float64, episodeMatch *tvEpisodeMatch) error {
	status := StatusMatched
	if confidence < 0.85 {
		status = StatusNeedsReview
	}

	genresJSON, err := marshalStringSlice(extractNamedValues(detail.Genres, 8))
	if err != nil {
		return err
	}
	castJSON, err := marshalPeople(extractCast(detail, tmdbCfg, 8))
	if err != nil {
		return err
	}
	directorsJSON, err := marshalPeople(extractDirectors(detail, tmdbCfg))
	if err != nil {
		return err
	}
	regionsJSON, err := marshalStringSlice(extractCountryValues(detail.ProductionCountries, 8))
	if err != nil {
		return err
	}
	trailerJSON, err := marshalTrailer(buildTrailerDetail(detail))
	if err != nil {
		return err
	}

	title := item.Title
	originalTitle := item.OriginalTitle
	seriesTitle := item.SeriesTitle
	overview := detail.Overview
	releaseDate := detail.ReleaseDate
	runtimeSeconds := runtimeFromDetail(detail)
	if mediaType == "movie" {
		if strings.TrimSpace(detail.Title) != "" {
			title = detail.Title
		}
		if strings.TrimSpace(detail.OriginalTitle) != "" {
			originalTitle = detail.OriginalTitle
		}
	} else {
		if strings.TrimSpace(detail.Name) != "" {
			seriesTitle = detail.Name
		}
		if originalTitle == "" && strings.TrimSpace(detail.OriginalName) != "" {
			originalTitle = detail.OriginalName
		}
		if releaseDate == "" {
			releaseDate = detail.FirstAirDate
		}
		if item.Type == "episode" && episodeMatch != nil {
			if !episodeMatch.seasonFound || !episodeMatch.episodeFound {
				status = StatusNeedsReview
			}
			if episodeMatch.episodeFound {
				if strings.TrimSpace(episodeMatch.episode.Name) != "" {
					title = episodeMatch.episode.Name
				}
				if strings.TrimSpace(episodeMatch.episode.Overview) != "" {
					overview = episodeMatch.episode.Overview
				}
				if strings.TrimSpace(episodeMatch.episode.AirDate) != "" {
					releaseDate = episodeMatch.episode.AirDate
				}
				if runtime := runtimeSecondsFromMinutes(episodeMatch.episode.Runtime); runtime != nil {
					runtimeSeconds = runtime
				}
			}
		}
	}
	posterURL := preferArtworkURL(item.PosterURL, imageURL(tmdbCfg, detail.PosterPath))
	backdropCandidate := imageURL(tmdbCfg, detail.BackdropPath)
	if mediaType == "tv" && item.Type == "episode" && episodeMatch != nil && episodeMatch.episodeFound {
		backdropCandidate = imageURL(tmdbCfg, episodeMatch.episode.StillPath)
	}
	backdropURL := preferArtworkURL(item.BackdropURL, backdropCandidate)
	logoURL := preferArtworkURL(item.LogoURL, imageURL(tmdbCfg, pickLogoPath(tmdbCfg.Language, detail.Images.Logos)))
	year := parseYear(releaseDate)

	if err := s.db.WithContext(ctx).Model(&database.MediaItem{}).Where("id = ?", item.ID).Updates(map[string]any{
		"title":               title,
		"original_title":      originalTitle,
		"series_title":        seriesTitle,
		"overview":            overview,
		"poster_url":          posterURL,
		"logo_url":            logoURL,
		"backdrop_url":        backdropURL,
		"genres_json":         genresJSON,
		"regions_json":        regionsJSON,
		"cast_json":           castJSON,
		"directors_json":      directorsJSON,
		"vote_average":        detail.VoteAverage,
		"year":                year,
		"release_date":        releaseDate,
		"runtime_seconds":     runtimeSeconds,
		"trailer_json":        trailerJSON,
		"match_status":        status,
		"metadata_provider":   "tmdb",
		"external_id":         mediaType + ":" + strconv.Itoa(detail.ID),
		"metadata_confidence": confidence,
	}).Error; err != nil {
		return err
	}
	if s.search != nil {
		return s.search.ReindexMediaItem(ctx, item.ID)
	}
	return nil
}

func extractCountryValues(values []countryValue, limit int) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if name := strings.TrimSpace(value.Name); name != "" {
			result = append(result, name)
		}
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func preferArtworkURL(current, candidate string) string {
	trimmedCandidate := strings.TrimSpace(candidate)
	if trimmedCandidate != "" {
		return trimmedCandidate
	}
	return strings.TrimSpace(current)
}

func (s *Service) SearchCandidates(ctx context.Context, mediaItemID uint, input ManualSearchInput) ([]SearchCandidate, error) {
	tmdbCfg, err := s.tmdbConfig(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(tmdbCfg.APIKey) == "" {
		return nil, fmt.Errorf("tmdb 未配置，无法搜索元数据")
	}

	var item database.MediaItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", mediaItemID).First(&item).Error; err != nil {
		return nil, err
	}

	mediaType := tmdbMediaType(item.Type)
	if tmdbID := strings.TrimSpace(input.TMDBID); tmdbID != "" {
		id, err := strconv.Atoi(tmdbID)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("tmdb_id 必须是正整数")
		}
		detail, err := s.fetchDetail(ctx, tmdbCfg, mediaType, id)
		if err != nil {
			return nil, err
		}
		candidate := detailToCandidate(tmdbCfg, mediaType, detail, 1)
		candidate.MatchedQuery = "TMDB ID " + tmdbID
		candidate.ReasonSummary = "通过 TMDB ID 精确定位"
		return []SearchCandidate{candidate}, nil
	}
	if imdbID := strings.TrimSpace(input.IMDbID); imdbID != "" {
		results, err := s.findByExternalID(ctx, tmdbCfg, mediaType, "imdb_id", imdbID)
		if err != nil {
			return nil, err
		}
		candidates := make([]SearchCandidate, 0, len(results))
		for _, result := range results {
			candidate := searchResultToCandidate(tmdbCfg, mediaType, scoredMatchCandidate{result: result, confidence: 1, matchedQuery: "IMDb ID " + imdbID, reasonSummary: "通过 IMDb ID 精确定位"})
			candidates = append(candidates, candidate)
		}
		return candidates, nil
	}
	if tvdbID := strings.TrimSpace(input.TVDBID); tvdbID != "" {
		results, err := s.findByExternalID(ctx, tmdbCfg, mediaType, "tvdb_id", tvdbID)
		if err != nil {
			return nil, err
		}
		candidates := make([]SearchCandidate, 0, len(results))
		for _, result := range results {
			candidate := searchResultToCandidate(tmdbCfg, mediaType, scoredMatchCandidate{result: result, confidence: 1, matchedQuery: "TVDB ID " + tvdbID, reasonSummary: "通过 TVDB ID 精确定位"})
			candidates = append(candidates, candidate)
		}
		return candidates, nil
	}

	queries := buildManualSearchQueries(input, item, mediaType)
	if len(queries) == 0 {
		return nil, fmt.Errorf("标题不能为空")
	}
	return s.searchCandidates(ctx, tmdbCfg, mediaType, queries, item)
}
