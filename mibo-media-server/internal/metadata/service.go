package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/settings"
	"gorm.io/gorm"
)

const (
	StatusPending     = "pending"
	StatusMatched     = "matched"
	StatusNeedsReview = "needs_review"
	StatusUnmatched   = "unmatched"
	StatusSkipped     = "skipped"
	tmdbCacheTTL      = 7 * 24 * time.Hour
)

type Service struct {
	db       *gorm.DB
	fallback config.MetadataConfig
	settings *settings.Service
}

type ManualSearchInput struct {
	Title  string `json:"title"`
	Year   *int   `json:"year"`
	IMDbID string `json:"imdb_id"`
	TMDBID string `json:"tmdb_id"`
	TVDBID string `json:"tvdb_id"`
}

type ApplyCandidateInput struct {
	ExternalID string `json:"external_id"`
}

type SearchCandidate struct {
	Provider      string  `json:"provider"`
	MediaType     string  `json:"media_type"`
	ExternalID    string  `json:"external_id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	Overview      string  `json:"overview"`
	PosterURL     string  `json:"poster_url"`
	BackdropURL   string  `json:"backdrop_url"`
	ReleaseDate   string  `json:"release_date"`
	Year          *int    `json:"year,omitempty"`
	Confidence    float64 `json:"confidence"`
}

type tmdbErrorResponse struct {
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
	Success       bool   `json:"success"`
}

type searchResponse struct {
	Results []searchResult `json:"results"`
}

type searchResult struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	Name          string `json:"name"`
	OriginalTitle string `json:"original_title"`
	OriginalName  string `json:"original_name"`
	ReleaseDate   string `json:"release_date"`
	FirstAirDate  string `json:"first_air_date"`
	Overview      string `json:"overview"`
	PosterPath    string `json:"poster_path"`
	BackdropPath  string `json:"backdrop_path"`
}

type detailResponse struct {
	ID             int             `json:"id"`
	Runtime        *int            `json:"runtime"`
	EpisodeRunTime []int           `json:"episode_run_time"`
	Title          string          `json:"title"`
	Name           string          `json:"name"`
	OriginalTitle  string          `json:"original_title"`
	OriginalName   string          `json:"original_name"`
	Overview       string          `json:"overview"`
	PosterPath     string          `json:"poster_path"`
	BackdropPath   string          `json:"backdrop_path"`
	ReleaseDate    string          `json:"release_date"`
	FirstAirDate   string          `json:"first_air_date"`
	Genres         []namedValue    `json:"genres"`
	Seasons        []seasonSummary `json:"seasons"`
	CreatedBy      []namedValue    `json:"created_by"`
	Credits        creditsResponse `json:"credits"`
	Images         imagesResponse  `json:"images"`
}

type seasonSummary struct {
	ID           int    `json:"id"`
	SeasonNumber int    `json:"season_number"`
	Name         string `json:"name"`
	Overview     string `json:"overview"`
	PosterPath   string `json:"poster_path"`
}

type seasonDetailResponse struct {
	ID           int                     `json:"id"`
	SeasonNumber int                     `json:"season_number"`
	Name         string                  `json:"name"`
	Overview     string                  `json:"overview"`
	PosterPath   string                  `json:"poster_path"`
	Episodes     []seasonEpisodeResponse `json:"episodes"`
}

type seasonEpisodeResponse struct {
	ID            int    `json:"id"`
	SeasonNumber  int    `json:"season_number"`
	EpisodeNumber int    `json:"episode_number"`
	Name          string `json:"name"`
	Overview      string `json:"overview"`
	StillPath     string `json:"still_path"`
	Runtime       *int   `json:"runtime"`
}

type TVSeasonMetadata struct {
	SeasonNumber   int    `json:"season_number"`
	Name           string `json:"name"`
	Overview       string `json:"overview"`
	PosterURL      string `json:"poster_url"`
	RuntimeSeconds *int   `json:"runtime_seconds,omitempty"`
}

type TVEpisodeMetadata struct {
	SeasonNumber   int    `json:"season_number"`
	EpisodeNumber  int    `json:"episode_number"`
	Name           string `json:"name"`
	Overview       string `json:"overview"`
	StillURL       string `json:"still_url"`
	RuntimeSeconds *int   `json:"runtime_seconds,omitempty"`
}

type imagesResponse struct {
	Logos []imageAsset `json:"logos"`
}

type imageAsset struct {
	FilePath    string  `json:"file_path"`
	Language    string  `json:"iso_639_1"`
	VoteAverage float64 `json:"vote_average"`
}

type namedValue struct {
	Name string `json:"name"`
}

type creditsResponse struct {
	Cast []castMember `json:"cast"`
	Crew []crewMember `json:"crew"`
}

type castMember struct {
	Name        string `json:"name"`
	Character   string `json:"character"`
	ProfilePath string `json:"profile_path"`
}

type crewMember struct {
	Name        string `json:"name"`
	Job         string `json:"job"`
	Department  string `json:"department"`
	ProfilePath string `json:"profile_path"`
}

func NewService(db *gorm.DB, cfg config.MetadataConfig, settingsSvc *settings.Service) *Service {
	return &Service{
		db:       db,
		fallback: cfg,
		settings: settingsSvc,
	}
}

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
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", mediaItemID).
		First(&item).Error; err != nil {
		return err
	}

	if strings.TrimSpace(tmdbCfg.APIKey) == "" {
		return s.db.WithContext(ctx).
			Model(&database.MediaItem{}).
			Where("id = ?", item.ID).
			Updates(map[string]any{
				"match_status": StatusSkipped,
			}).Error
	}

	mediaType := tmdbMediaType(item.Type)
	query := defaultQuery(item, mediaType)

	searchResult, confidence, err := s.searchBestMatch(ctx, tmdbCfg, mediaType, query, item.Year)
	if err != nil {
		return err
	}
	if searchResult == nil {
		return s.db.WithContext(ctx).
			Model(&database.MediaItem{}).
			Where("id = ?", item.ID).
			Updates(map[string]any{
				"match_status":        StatusUnmatched,
				"metadata_provider":   "",
				"external_id":         "",
				"metadata_confidence": nil,
			}).Error
	}

	detail, err := s.fetchDetail(ctx, tmdbCfg, mediaType, searchResult.ID)
	if err != nil {
		return err
	}

	return s.applyDetail(ctx, item, tmdbCfg, mediaType, detail, confidence)
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
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", mediaItemID).
		First(&item).Error; err != nil {
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
	query := defaultQuery(item, mediaType)
	if query != "" {
		confidence = calculateConfidence(mediaType, query, item.Year, searchResult{
			ID:            detail.ID,
			Title:         detail.Title,
			Name:          detail.Name,
			OriginalTitle: detail.OriginalTitle,
			OriginalName:  detail.OriginalName,
			ReleaseDate:   detail.ReleaseDate,
			FirstAirDate:  detail.FirstAirDate,
			Overview:      detail.Overview,
			PosterPath:    detail.PosterPath,
			BackdropPath:  detail.BackdropPath,
		})
	}

	return s.applyDetail(ctx, item, tmdbCfg, mediaType, detail, confidence)
}

func (s *Service) applyDetail(ctx context.Context, item database.MediaItem, tmdbCfg config.TMDBConfig, mediaType string, detail detailResponse, confidence float64) error {
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

	title := item.Title
	originalTitle := item.OriginalTitle
	seriesTitle := item.SeriesTitle
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
	}

	return s.db.WithContext(ctx).
		Model(&database.MediaItem{}).
		Where("id = ?", item.ID).
		Updates(map[string]any{
			"title":               title,
			"original_title":      originalTitle,
			"series_title":        seriesTitle,
			"overview":            detail.Overview,
			"poster_url":          imageURL(tmdbCfg, detail.PosterPath),
			"logo_url":            imageURL(tmdbCfg, pickLogoPath(tmdbCfg.Language, detail.Images.Logos)),
			"backdrop_url":        imageURL(tmdbCfg, detail.BackdropPath),
			"genres_json":         genresJSON,
			"cast_json":           castJSON,
			"directors_json":      directorsJSON,
			"release_date":        releaseDate,
			"runtime_seconds":     runtimeSeconds,
			"match_status":        status,
			"metadata_provider":   "tmdb",
			"external_id":         mediaType + ":" + strconv.Itoa(detail.ID),
			"metadata_confidence": confidence,
		}).Error
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
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", mediaItemID).
		First(&item).Error; err != nil {
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
		return []SearchCandidate{candidate}, nil
	}

	query := strings.TrimSpace(input.Title)
	if query == "" {
		query = defaultQuery(item, mediaType)
	}
	if query == "" {
		return nil, fmt.Errorf("标题不能为空")
	}

	year := input.Year
	if year == nil {
		year = item.Year
	}

	return s.searchCandidates(ctx, tmdbCfg, mediaType, query, year)
}

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

func (s *Service) ListSeasonEpisodes(ctx context.Context, seriesTMDBID int, seasonNumber int) ([]TVEpisodeMetadata, error) {
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

	episodes := make([]TVEpisodeMetadata, 0, len(episodeRows))
	for _, row := range episodeRows {
		episodes = append(episodes, TVEpisodeMetadata{
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

func (s *Service) searchBestMatch(ctx context.Context, cfg config.TMDBConfig, mediaType, query string, year *int) (*searchResult, float64, error) {
	if strings.TrimSpace(query) == "" {
		return nil, 0, nil
	}

	params := url.Values{}
	params.Set("query", query)
	params.Set("language", cfg.Language)
	if year != nil {
		if mediaType == "movie" {
			params.Set("year", strconv.Itoa(*year))
		} else {
			params.Set("first_air_date_year", strconv.Itoa(*year))
		}
	}

	var response searchResponse
	if err := s.request(ctx, cfg, path.Join("search", mediaType), params, &response); err != nil {
		return nil, 0, err
	}

	var best *searchResult
	bestConfidence := 0.0
	for i := range response.Results {
		candidate := &response.Results[i]
		confidence := calculateConfidence(mediaType, query, year, *candidate)
		if confidence > bestConfidence {
			best = candidate
			bestConfidence = confidence
		}
	}

	return best, bestConfidence, nil
}

func (s *Service) searchCandidates(ctx context.Context, cfg config.TMDBConfig, mediaType, query string, year *int) ([]SearchCandidate, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}

	params := url.Values{}
	params.Set("query", query)
	params.Set("language", cfg.Language)
	if year != nil {
		if mediaType == "movie" {
			params.Set("year", strconv.Itoa(*year))
		} else {
			params.Set("first_air_date_year", strconv.Itoa(*year))
		}
	}

	var response searchResponse
	if err := s.request(ctx, cfg, path.Join("search", mediaType), params, &response); err != nil {
		return nil, err
	}

	type scoredCandidate struct {
		result     searchResult
		confidence float64
	}

	scored := make([]scoredCandidate, 0, len(response.Results))
	for _, candidate := range response.Results {
		title := strings.TrimSpace(candidate.Title)
		if mediaType == "tv" {
			title = strings.TrimSpace(candidate.Name)
		}
		if title == "" {
			continue
		}
		scored = append(scored, scoredCandidate{
			result:     candidate,
			confidence: calculateConfidence(mediaType, query, year, candidate),
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].confidence == scored[j].confidence {
			return scored[i].result.ID < scored[j].result.ID
		}
		return scored[i].confidence > scored[j].confidence
	})

	if len(scored) > 8 {
		scored = scored[:8]
	}

	results := make([]SearchCandidate, 0, len(scored))
	for _, candidate := range scored {
		results = append(results, searchResultToCandidate(cfg, mediaType, candidate.result, candidate.confidence))
	}

	return results, nil
}

func (s *Service) fetchDetail(ctx context.Context, cfg config.TMDBConfig, mediaType string, id int) (detailResponse, error) {
	params := url.Values{}
	params.Set("language", cfg.Language)
	params.Set("append_to_response", "credits,images")
	params.Set("include_image_language", imageLanguages(cfg.Language))

	var detail detailResponse
	if err := s.request(ctx, cfg, path.Join(mediaType, strconv.Itoa(id)), params, &detail); err != nil {
		return detailResponse{}, err
	}
	return detail, nil
}

func (s *Service) fetchTVSeason(ctx context.Context, cfg config.TMDBConfig, seriesTMDBID int, seasonNumber int) (seasonDetailResponse, error) {
	params := url.Values{}
	params.Set("language", cfg.Language)

	var detail seasonDetailResponse
	if err := s.request(ctx, cfg, path.Join("tv", strconv.Itoa(seriesTMDBID), "season", strconv.Itoa(seasonNumber)), params, &detail); err != nil {
		return seasonDetailResponse{}, err
	}
	return detail, nil
}

func (s *Service) lookupSeasonCache(ctx context.Context, seriesTMDBID int, language string) ([]database.TVSeasonMetadataCache, error) {
	var rows []database.TVSeasonMetadataCache
	err := s.db.WithContext(ctx).
		Where("series_tmdb_id = ? AND language = ?", seriesTMDBID, strings.TrimSpace(language)).
		Order("season_number asc, id asc").
		Find(&rows).Error
	return rows, err
}

func (s *Service) lookupSeasonCacheRow(ctx context.Context, seriesTMDBID int, seasonNumber int, language string) (database.TVSeasonMetadataCache, bool, error) {
	var row database.TVSeasonMetadataCache
	err := s.db.WithContext(ctx).
		Where("series_tmdb_id = ? AND season_number = ? AND language = ?", seriesTMDBID, seasonNumber, strings.TrimSpace(language)).
		First(&row).Error
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
	err := s.db.WithContext(ctx).
		Where("series_tmdb_id = ? AND season_number = ? AND language = ?", seriesTMDBID, seasonNumber, strings.TrimSpace(language)).
		Order("episode_number asc, id asc").
		Find(&rows).Error
	return rows, err
}

func (s *Service) upsertSeasonCache(ctx context.Context, seriesTMDBID int, language string, rows []database.TVSeasonMetadataCache, fetchedAt time.Time) error {
	for _, row := range rows {
		row.SeriesTMDBID = seriesTMDBID
		row.Language = strings.TrimSpace(language)
		row.FetchedAt = fetchedAt
		if err := s.db.WithContext(ctx).Where(database.TVSeasonMetadataCache{
			SeriesTMDBID: row.SeriesTMDBID,
			SeasonNumber: row.SeasonNumber,
			Language:     row.Language,
		}).Assign(map[string]any{
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
		if err := s.db.WithContext(ctx).Where(database.TVEpisodeMetadataCache{
			SeriesTMDBID:  row.SeriesTMDBID,
			SeasonNumber:  row.SeasonNumber,
			EpisodeNumber: row.EpisodeNumber,
			Language:      row.Language,
		}).Assign(map[string]any{
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

func (s *Service) request(ctx context.Context, cfg config.TMDBConfig, endpoint string, params url.Values, out any) error {
	params = cloneValues(params)
	useBearerToken := looksLikeTMDBBearerToken(cfg.APIKey)
	if !useBearerToken {
		params.Set("api_key", cfg.APIKey)
	}

	requestURL := cfg.BaseURL + "/" + strings.TrimLeft(endpoint, "/")
	if encoded := params.Encode(); encoded != "" {
		requestURL += "?" + encoded
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	if useBearerToken {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.APIKey))
	}

	resp, err := (&http.Client{Timeout: cfg.Timeout}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		return tmdbRequestError(resp.StatusCode, body)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func looksLikeTMDBBearerToken(value string) bool {
	trimmed := strings.TrimSpace(value)
	return strings.HasPrefix(trimmed, "eyJ") || strings.Count(trimmed, ".") >= 2
}

func tmdbRequestError(statusCode int, body []byte) error {
	var response tmdbErrorResponse
	if len(body) > 0 {
		_ = json.Unmarshal(body, &response)
	}

	message := strings.TrimSpace(response.StatusMessage)
	switch statusCode {
	case http.StatusUnauthorized:
		if message == "" {
			message = "API Key 无效或已失效"
		}
		return fmt.Errorf("TMDB 认证失败，请检查 API Key: %s", message)
	case http.StatusForbidden:
		if message == "" {
			message = "请求被 TMDB 拒绝"
		}
		return fmt.Errorf("TMDB 请求被拒绝: %s", message)
	case http.StatusTooManyRequests:
		if message == "" {
			message = "请求过于频繁"
		}
		return fmt.Errorf("TMDB 触发限流: %s", message)
	default:
		if message != "" {
			return fmt.Errorf("TMDB 请求失败(%d): %s", statusCode, message)
		}
		return fmt.Errorf("TMDB 请求失败(%d)", statusCode)
	}
}

func imageURL(cfg config.TMDBConfig, imagePath string) string {
	trimmed := strings.TrimSpace(imagePath)
	if trimmed == "" {
		return ""
	}
	return cfg.ImageBaseURL + "/" + strings.TrimLeft(trimmed, "/")
}

func imageLanguages(language string) string {
	trimmed := strings.TrimSpace(language)
	if trimmed == "" {
		return "en,null"
	}
	base := trimmed
	if idx := strings.Index(trimmed, "-"); idx > 0 {
		base = trimmed[:idx]
	}
	if base == trimmed {
		return trimmed + ",null,en"
	}
	return trimmed + "," + base + ",null,en"
}

func pickLogoPath(language string, logos []imageAsset) string {
	if len(logos) == 0 {
		return ""
	}

	trimmed := strings.TrimSpace(language)
	base := trimmed
	if idx := strings.Index(trimmed, "-"); idx > 0 {
		base = trimmed[:idx]
	}

	rank := func(asset imageAsset) int {
		lang := strings.TrimSpace(asset.Language)
		switch {
		case trimmed != "" && lang == trimmed:
			return 0
		case base != "" && lang == base:
			return 1
		case lang == "":
			return 2
		case lang == "en":
			return 3
		default:
			return 4
		}
	}

	best := logos[0]
	bestRank := rank(best)
	for _, logo := range logos[1:] {
		currentRank := rank(logo)
		if currentRank < bestRank || (currentRank == bestRank && logo.VoteAverage > best.VoteAverage) {
			best = logo
			bestRank = currentRank
		}
	}

	return best.FilePath
}

func calculateConfidence(mediaType, query string, year *int, result searchResult) float64 {
	target := normalizeString(query)
	candidate := normalizeString(result.Title)
	if mediaType == "tv" {
		candidate = normalizeString(result.Name)
	}

	confidence := 0.4
	if target == candidate {
		confidence = 0.9
	} else if strings.Contains(candidate, target) || strings.Contains(target, candidate) {
		confidence = 0.75
	}

	if year != nil {
		resultYear := parseYear(result.ReleaseDate)
		if mediaType == "tv" {
			resultYear = parseYear(result.FirstAirDate)
		}
		if resultYear != nil && *resultYear == *year {
			confidence += 0.08
		}
	}

	if confidence > 0.99 {
		confidence = 0.99
	}
	return confidence
}

func extractNamedValues(values []namedValue, max int) []string {
	limit := len(values)
	if max > 0 && limit > max {
		limit = max
	}
	result := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		name := strings.TrimSpace(values[i].Name)
		if name != "" {
			result = append(result, name)
		}
	}
	return result
}

func extractCast(detail detailResponse, cfg config.TMDBConfig, max int) []library.PersonDetail {
	limit := len(detail.Credits.Cast)
	if max > 0 && limit > max {
		limit = max
	}
	result := make([]library.PersonDetail, 0, limit)
	for i := 0; i < limit; i++ {
		member := detail.Credits.Cast[i]
		name := strings.TrimSpace(member.Name)
		if name == "" {
			continue
		}
		result = append(result, library.PersonDetail{
			Name:      name,
			Role:      strings.TrimSpace(member.Character),
			AvatarURL: imageURL(cfg, member.ProfilePath),
		})
	}
	return result
}

func extractDirectors(detail detailResponse, cfg config.TMDBConfig) []library.PersonDetail {
	if len(detail.Credits.Crew) > 0 {
		result := make([]library.PersonDetail, 0, 4)
		for _, member := range detail.Credits.Crew {
			if member.Job == "Director" || member.Department == "Directing" {
				name := strings.TrimSpace(member.Name)
				if name == "" {
					continue
				}
				result = append(result, library.PersonDetail{
					Name:      name,
					Role:      strings.TrimSpace(member.Job),
					AvatarURL: imageURL(cfg, member.ProfilePath),
				})
				if len(result) == 4 {
					return result
				}
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	fallback := extractNamedValues(detail.CreatedBy, 4)
	result := make([]library.PersonDetail, 0, len(fallback))
	for _, name := range fallback {
		result = append(result, library.PersonDetail{Name: name, Role: "Creator"})
	}
	return result
}

func runtimeFromDetail(detail detailResponse) *int {
	if detail.Runtime != nil && *detail.Runtime > 0 {
		seconds := *detail.Runtime * 60
		return &seconds
	}
	if len(detail.EpisodeRunTime) > 0 && detail.EpisodeRunTime[0] > 0 {
		seconds := detail.EpisodeRunTime[0] * 60
		return &seconds
	}
	return nil
}

func runtimeSecondsFromMinutes(minutes *int) *int {
	if minutes == nil || *minutes <= 0 {
		return nil
	}
	seconds := *minutes * 60
	return &seconds
}

func cacheIsStale(fetchedAt time.Time) bool {
	if fetchedAt.IsZero() {
		return true
	}
	return time.Since(fetchedAt) > tmdbCacheTTL
}

func seasonCachesStale(rows []database.TVSeasonMetadataCache) bool {
	for _, row := range rows {
		if cacheIsStale(row.FetchedAt) {
			return true
		}
	}
	return false
}

func episodeCachesStale(rows []database.TVEpisodeMetadataCache) bool {
	for _, row := range rows {
		if cacheIsStale(row.FetchedAt) {
			return true
		}
	}
	return false
}

func seasonSummariesToCacheRows(summaries []seasonSummary) []database.TVSeasonMetadataCache {
	rows := make([]database.TVSeasonMetadataCache, 0, len(summaries))
	for _, season := range summaries {
		payloadJSON, err := marshalPayload(season)
		if err != nil {
			continue
		}
		rows = append(rows, database.TVSeasonMetadataCache{
			SeasonNumber: season.SeasonNumber,
			Name:         season.Name,
			Overview:     season.Overview,
			PosterPath:   season.PosterPath,
			PayloadJSON:  payloadJSON,
		})
	}
	return rows
}

func seasonDetailToCacheRow(seriesTMDBID int, language string, detail seasonDetailResponse, fetchedAt time.Time) database.TVSeasonMetadataCache {
	payloadJSON, _ := marshalPayload(detail)
	return database.TVSeasonMetadataCache{
		SeriesTMDBID: seriesTMDBID,
		SeasonNumber: detail.SeasonNumber,
		Language:     strings.TrimSpace(language),
		Name:         detail.Name,
		Overview:     detail.Overview,
		PosterPath:   detail.PosterPath,
		PayloadJSON:  payloadJSON,
		FetchedAt:    fetchedAt,
	}
}

func marshalPayload(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func marshalStringSlice(values []string) (string, error) {
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func marshalPeople(values []library.PersonDetail) (string, error) {
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func normalizeString(input string) string {
	lower := strings.ToLower(strings.TrimSpace(input))
	replacer := strings.NewReplacer(".", " ", "_", " ", "-", " ")
	return strings.Join(strings.Fields(replacer.Replace(lower)), " ")
}

func parseYear(input string) *int {
	if len(input) < 4 {
		return nil
	}
	value, err := strconv.Atoi(input[:4])
	if err != nil {
		return nil
	}
	return &value
}

func cloneValues(values url.Values) url.Values {
	if values == nil {
		return url.Values{}
	}
	result := make(url.Values, len(values))
	for key, list := range values {
		copied := make([]string, len(list))
		copy(copied, list)
		result[key] = copied
	}
	return result
}

func tmdbMediaType(itemType string) string {
	if itemType == "episode" {
		return "tv"
	}
	return "movie"
}

func defaultQuery(item database.MediaItem, mediaType string) string {
	if mediaType == "tv" && strings.TrimSpace(item.SeriesTitle) != "" {
		return item.SeriesTitle
	}
	return item.Title
}

func searchResultToCandidate(cfg config.TMDBConfig, mediaType string, result searchResult, confidence float64) SearchCandidate {
	title := result.Title
	originalTitle := result.OriginalTitle
	releaseDate := result.ReleaseDate
	if mediaType == "tv" {
		title = result.Name
		originalTitle = result.OriginalName
		releaseDate = result.FirstAirDate
	}

	return SearchCandidate{
		Provider:      "tmdb",
		MediaType:     mediaType,
		ExternalID:    mediaType + ":" + strconv.Itoa(result.ID),
		Title:         title,
		OriginalTitle: originalTitle,
		Overview:      result.Overview,
		PosterURL:     imageURL(cfg, result.PosterPath),
		BackdropURL:   imageURL(cfg, result.BackdropPath),
		ReleaseDate:   releaseDate,
		Year:          parseYear(releaseDate),
		Confidence:    confidence,
	}
}

func detailToCandidate(cfg config.TMDBConfig, mediaType string, detail detailResponse, confidence float64) SearchCandidate {
	title := detail.Title
	originalTitle := detail.OriginalTitle
	releaseDate := detail.ReleaseDate
	if mediaType == "tv" {
		title = detail.Name
		originalTitle = detail.OriginalName
		releaseDate = detail.FirstAirDate
	}

	return SearchCandidate{
		Provider:      "tmdb",
		MediaType:     mediaType,
		ExternalID:    mediaType + ":" + strconv.Itoa(detail.ID),
		Title:         title,
		OriginalTitle: originalTitle,
		Overview:      detail.Overview,
		PosterURL:     imageURL(cfg, detail.PosterPath),
		BackdropURL:   imageURL(cfg, detail.BackdropPath),
		ReleaseDate:   releaseDate,
		Year:          parseYear(releaseDate),
		Confidence:    confidence,
	}
}

func parseExternalID(value string) (string, int, error) {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("external_id 格式无效")
	}
	mediaType := strings.TrimSpace(parts[0])
	if mediaType != "movie" && mediaType != "tv" {
		return "", 0, fmt.Errorf("external_id 媒体类型无效")
	}
	id, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || id <= 0 {
		return "", 0, fmt.Errorf("external_id 标识无效")
	}
	return mediaType, id, nil
}
