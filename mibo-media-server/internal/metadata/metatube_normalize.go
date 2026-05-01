package metadata

import (
	"context"
	"fmt"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/library"
)

func (s *Service) searchMetaTubeCandidates(ctx context.Context, cfg config.MetaTubeConfig, queries []matchSearchQuery, item metadataSearchItem) ([]SearchCandidate, error) {
	if len(queries) == 0 {
		return nil, nil
	}
	bestByID := map[string]SearchCandidate{}
	for _, query := range queries {
		results, err := s.searchMetaTube(ctx, cfg, query.Value)
		if err != nil {
			return nil, err
		}
		for _, result := range results {
			candidate := metatubeSearchResultToCandidate(result, item, query)
			if strings.TrimSpace(candidate.ExternalID) == "" {
				continue
			}
			current, exists := bestByID[candidate.ExternalID]
			if !exists || candidate.Confidence > current.Confidence {
				bestByID[candidate.ExternalID] = candidate
			}
		}
	}
	candidates := make([]SearchCandidate, 0, len(bestByID))
	for _, candidate := range bestByID {
		candidates = append(candidates, candidate)
	}
	sortSearchCandidates(candidates)
	if len(candidates) > 8 {
		candidates = candidates[:8]
	}
	return candidates, nil
}

func metatubeSearchResultToCandidate(result metatubeSearchResult, item metadataSearchItem, query matchSearchQuery) SearchCandidate {
	title := firstNonEmpty(result.Title, result.Number, result.OriginalTitle)
	originalTitle := firstNonEmpty(result.OriginalTitle, result.Title)
	releaseDate := metatubeReleaseDate(result.ReleaseDate, result.Date, result.Year)
	confidence := bestTitleScore(query.Value, []string{title, originalTitle, result.Number}) + 0.08
	if item.Year != nil && parseYear(releaseDate) != nil && *item.Year == *parseYear(releaseDate) {
		confidence += 0.08
	}
	if confidence < 0.05 {
		confidence = 0.05
	}
	if confidence > 0.99 {
		confidence = 0.99
	}
	upstreamProvider := strings.TrimSpace(result.Provider)
	upstreamID := strings.TrimSpace(firstNonEmpty(result.ID, result.Number))
	return SearchCandidate{Provider: "metatube", MediaType: "movie", ExternalID: metatubeExternalID(upstreamProvider, upstreamID), Title: title, OriginalTitle: originalTitle, Overview: firstNonEmpty(result.Overview, result.Summary), PosterURL: firstNonEmpty(result.PosterURL, result.CoverURL, result.ThumbURL), BackdropURL: result.BackdropURL, ReleaseDate: releaseDate, Year: parseYear(releaseDate), Confidence: confidence, MatchedQuery: query.Value, ReasonSummary: titleScoreSummary(confidence, query.Value)}
}

func metatubeExternalID(upstreamProvider string, upstreamID string) string {
	provider := strings.TrimSpace(upstreamProvider)
	id := strings.TrimSpace(upstreamID)
	if provider == "" || id == "" {
		return ""
	}
	return "metatube:" + provider + ":" + id
}

func parseMetaTubeExternalID(value string) (string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 3)
	if len(parts) != 3 || parts[0] != "metatube" || strings.TrimSpace(parts[1]) == "" || strings.TrimSpace(parts[2]) == "" {
		return "", "", fmt.Errorf("metatube external_id 格式无效")
	}
	return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2]), nil
}

func metatubeReleaseDate(values ...any) string {
	for _, value := range values {
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		case int:
			if typed > 0 {
				return fmt.Sprintf("%04d-01-01", typed)
			}
		}
	}
	return ""
}

func metatubeRuntimeSeconds(detail metatubeDetailResponse) *int {
	minutes := detail.Runtime
	if minutes == nil {
		minutes = detail.Duration
	}
	return runtimeSecondsFromMinutes(minutes)
}

func metatubeCast(detail metatubeDetailResponse, max int) []library.PersonDetail {
	people := detail.Actors
	if len(people) == 0 {
		people = detail.Cast
	}
	limit := len(people)
	if max > 0 && limit > max {
		limit = max
	}
	result := make([]library.PersonDetail, 0, limit)
	for i := 0; i < limit; i++ {
		name := strings.TrimSpace(people[i].Name)
		if name == "" {
			continue
		}
		result = append(result, library.PersonDetail{Name: name, Role: strings.TrimSpace(people[i].Role), AvatarURL: strings.TrimSpace(people[i].AvatarURL)})
	}
	return result
}

func metatubeDirectors(detail metatubeDetailResponse) []library.PersonDetail {
	names := detail.Directors
	if strings.TrimSpace(detail.Director) != "" {
		names = append([]string{detail.Director}, names...)
	}
	result := make([]library.PersonDetail, 0, len(names))
	seen := map[string]struct{}{}
	for _, value := range names {
		name := strings.TrimSpace(value)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, library.PersonDetail{Name: name, Role: "Director"})
	}
	return result
}

func metatubeGenres(detail metatubeDetailResponse) []string {
	values := append([]string{}, detail.Genres...)
	values = append(values, detail.Tags...)
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func metatubeGovernanceStatus(confidence float64) string {
	if confidence < 0.85 {
		return catalog.GovernanceNeedsReview
	}
	return catalog.GovernanceMatched
}
