package metadata

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func (s *Service) ListTVSeasons(ctx context.Context, seriesTMDBID int) ([]TVSeasonMetadata, error) {
	if seriesTMDBID <= 0 {
		return nil, fmt.Errorf("tmdb_id 必须是正整数")
	}
	tmdbCfg, err := s.tmdbConfig(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(tmdbCfg.APIKey) == "" {
		return nil, fmt.Errorf("tmdb 未配置，无法加载剧集季信息")
	}

	cached, err := s.lookupSeasonCache(ctx, seriesTMDBID, tmdbCfg.Language)
	if err != nil {
		return nil, err
	}
	if len(cached) == 0 || seasonCachesStale(cached) {
		detail, err := s.fetchDetail(ctx, tmdbCfg, "tv", seriesTMDBID)
		if err != nil {
			return nil, err
		}
		if err := s.upsertSeasonCache(ctx, seriesTMDBID, tmdbCfg.Language, seasonSummariesToCacheRows(detail.Seasons), time.Now().UTC()); err != nil {
			return nil, err
		}
		cached, err = s.lookupSeasonCache(ctx, seriesTMDBID, tmdbCfg.Language)
		if err != nil {
			return nil, err
		}
	}

	seasons := make([]TVSeasonMetadata, 0, len(cached))
	for _, row := range cached {
		seasons = append(seasons, TVSeasonMetadata{
			SeasonNumber:   row.SeasonNumber,
			Name:           row.Name,
			Overview:       row.Overview,
			PosterURL:      imageURL(tmdbCfg, row.PosterPath),
			RuntimeSeconds: row.RuntimeSeconds,
		})
	}
	return seasons, nil
}

func (s *Service) ListSeasonEpisodes(ctx context.Context, seriesTMDBID int, seasonNumber int, libraryID *uint) ([]TVEpisodeMetadata, error) {
	if seriesTMDBID <= 0 {
		return nil, fmt.Errorf("tmdb_id 必须是正整数")
	}
	if seasonNumber < 0 {
		return nil, fmt.Errorf("season_number 不能为负数")
	}
	tmdbCfg, err := s.tmdbConfig(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(tmdbCfg.APIKey) == "" {
		return nil, fmt.Errorf("tmdb 未配置，无法加载剧集集信息")
	}

	seasonRow, found, err := s.lookupSeasonCacheRow(ctx, seriesTMDBID, seasonNumber, tmdbCfg.Language)
	if err != nil {
		return nil, err
	}
	episodeRows, err := s.lookupEpisodeCache(ctx, seriesTMDBID, seasonNumber, tmdbCfg.Language)
	if err != nil {
		return nil, err
	}
	if !found || cacheIsStale(seasonRow.FetchedAt) || len(episodeRows) == 0 || episodeCachesStale(episodeRows) {
		seasonDetail, err := s.fetchTVSeason(ctx, tmdbCfg, seriesTMDBID, seasonNumber)
		if err != nil {
			return nil, err
		}
		fetchedAt := time.Now().UTC()
		if err := s.upsertSeasonCache(ctx, seriesTMDBID, tmdbCfg.Language, []database.TVSeasonMetadataCache{seasonDetailToCacheRow(seriesTMDBID, tmdbCfg.Language, seasonDetail, fetchedAt)}, fetchedAt); err != nil {
			return nil, err
		}
		if err := s.upsertEpisodeCache(ctx, seriesTMDBID, tmdbCfg.Language, seasonNumber, seasonDetail.Episodes, fetchedAt); err != nil {
			return nil, err
		}
		episodeRows, err = s.lookupEpisodeCache(ctx, seriesTMDBID, seasonNumber, tmdbCfg.Language)
		if err != nil {
			return nil, err
		}
	}

	episodeMediaItemIDs, err := s.lookupEpisodeMediaItemIDs(ctx, seriesTMDBID, seasonNumber, libraryID)
	if err != nil {
		return nil, err
	}

	episodes := make([]TVEpisodeMetadata, 0, len(episodeRows))
	for _, row := range episodeRows {
		episodes = append(episodes, TVEpisodeMetadata{
			MediaItemID:    episodeMediaItemIDs[row.EpisodeNumber],
			SeasonNumber:   row.SeasonNumber,
			EpisodeNumber:  row.EpisodeNumber,
			Name:           row.Name,
			Overview:       row.Overview,
			StillURL:       imageURL(tmdbCfg, row.StillPath),
			RuntimeSeconds: row.RuntimeSeconds,
		})
	}
	return episodes, nil
}

func (s *Service) lookupEpisodeMediaItemIDs(ctx context.Context, seriesTMDBID int, seasonNumber int, libraryID *uint) (map[int]*uint, error) {
	query := s.db.WithContext(ctx).Where("external_id = ? AND season_number = ? AND deleted_at IS NULL", fmt.Sprintf("tv:%d", seriesTMDBID), seasonNumber)
	if libraryID != nil {
		query = query.Where("library_id = ?", *libraryID)
	}
	var items []database.MediaItem
	if err := query.Order("id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	result := make(map[int]*uint, len(items))
	for _, item := range items {
		if item.EpisodeNumber == nil {
			continue
		}
		if _, exists := result[*item.EpisodeNumber]; exists {
			continue
		}
		id := uint(item.ID)
		result[*item.EpisodeNumber] = &id
	}
	return result, nil
}

func (s *Service) refreshTVMetadataCaches(ctx context.Context, cfg config.TMDBConfig, item database.MediaItem, seriesTMDBID int) error {
	fetchedAt := time.Now().UTC()
	seriesDetail, err := s.fetchDetail(ctx, cfg, "tv", seriesTMDBID)
	if err != nil {
		return err
	}
	if err := s.upsertSeasonCache(ctx, seriesTMDBID, cfg.Language, seasonSummariesToCacheRows(seriesDetail.Seasons), fetchedAt); err != nil {
		return err
	}
	if item.SeasonNumber == nil || *item.SeasonNumber < 0 {
		return nil
	}
	seasonDetail, err := s.fetchTVSeason(ctx, cfg, seriesTMDBID, *item.SeasonNumber)
	if err != nil {
		return err
	}
	if err := s.upsertSeasonCache(ctx, seriesTMDBID, cfg.Language, []database.TVSeasonMetadataCache{seasonDetailToCacheRow(seriesTMDBID, cfg.Language, seasonDetail, fetchedAt)}, fetchedAt); err != nil {
		return err
	}
	return s.upsertEpisodeCache(ctx, seriesTMDBID, cfg.Language, *item.SeasonNumber, seasonDetail.Episodes, fetchedAt)
}

func (s *Service) lookupSeasonCache(ctx context.Context, seriesTMDBID int, language string) ([]database.TVSeasonMetadataCache, error) {
	var rows []database.TVSeasonMetadataCache
	err := s.db.WithContext(ctx).Where("series_tmdb_id = ? AND language = ?", seriesTMDBID, strings.TrimSpace(language)).Order("season_number asc, id asc").Find(&rows).Error
	return rows, err
}

func (s *Service) lookupSeasonCacheRow(ctx context.Context, seriesTMDBID int, seasonNumber int, language string) (database.TVSeasonMetadataCache, bool, error) {
	var row database.TVSeasonMetadataCache
	err := s.db.WithContext(ctx).Where("series_tmdb_id = ? AND season_number = ? AND language = ?", seriesTMDBID, seasonNumber, strings.TrimSpace(language)).First(&row).Error
	if err == nil {
		return row, true, nil
	}
	if err == gorm.ErrRecordNotFound {
		return database.TVSeasonMetadataCache{}, false, nil
	}
	return database.TVSeasonMetadataCache{}, false, err
}

func (s *Service) lookupEpisodeCache(ctx context.Context, seriesTMDBID int, seasonNumber int, language string) ([]database.TVEpisodeMetadataCache, error) {
	var rows []database.TVEpisodeMetadataCache
	err := s.db.WithContext(ctx).Where("series_tmdb_id = ? AND season_number = ? AND language = ?", seriesTMDBID, seasonNumber, strings.TrimSpace(language)).Order("episode_number asc, id asc").Find(&rows).Error
	return rows, err
}

func (s *Service) upsertSeasonCache(ctx context.Context, seriesTMDBID int, language string, rows []database.TVSeasonMetadataCache, fetchedAt time.Time) error {
	for _, row := range rows {
		row.SeriesTMDBID = seriesTMDBID
		row.Language = strings.TrimSpace(language)
		row.FetchedAt = fetchedAt
		if err := s.db.WithContext(ctx).Where(database.TVSeasonMetadataCache{SeriesTMDBID: row.SeriesTMDBID, SeasonNumber: row.SeasonNumber, Language: row.Language}).Assign(map[string]any{
			"name":            row.Name,
			"overview":        row.Overview,
			"poster_path":     row.PosterPath,
			"runtime_seconds": row.RuntimeSeconds,
			"payload_json":    row.PayloadJSON,
			"fetched_at":      row.FetchedAt,
		}).FirstOrCreate(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) upsertEpisodeCache(ctx context.Context, seriesTMDBID int, language string, seasonNumber int, episodes []seasonEpisodeResponse, fetchedAt time.Time) error {
	for _, episode := range episodes {
		payloadJSON, err := marshalPayload(episode)
		if err != nil {
			return err
		}
		row := database.TVEpisodeMetadataCache{
			SeriesTMDBID:   seriesTMDBID,
			SeasonNumber:   seasonNumber,
			EpisodeNumber:  episode.EpisodeNumber,
			Language:       strings.TrimSpace(language),
			Name:           episode.Name,
			Overview:       episode.Overview,
			StillPath:      episode.StillPath,
			RuntimeSeconds: runtimeSecondsFromMinutes(episode.Runtime),
			PayloadJSON:    payloadJSON,
			FetchedAt:      fetchedAt,
		}
		if err := s.db.WithContext(ctx).Where(database.TVEpisodeMetadataCache{SeriesTMDBID: row.SeriesTMDBID, SeasonNumber: row.SeasonNumber, EpisodeNumber: row.EpisodeNumber, Language: row.Language}).Assign(map[string]any{
			"name":            row.Name,
			"overview":        row.Overview,
			"still_path":      row.StillPath,
			"runtime_seconds": row.RuntimeSeconds,
			"payload_json":    row.PayloadJSON,
			"fetched_at":      row.FetchedAt,
		}).FirstOrCreate(&row).Error; err != nil {
			return err
		}
	}
	return nil
}
