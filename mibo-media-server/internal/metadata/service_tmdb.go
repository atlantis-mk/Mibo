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

	"github.com/atlan/mibo-media-server/internal/config"
)

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
		scored = append(scored, scoredCandidate{result: candidate, confidence: calculateConfidence(mediaType, query, year, candidate)})
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
	params.Set("append_to_response", "credits,images,videos")
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
