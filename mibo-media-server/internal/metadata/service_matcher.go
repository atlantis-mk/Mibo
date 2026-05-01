package metadata

import (
	"context"
	"path"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/titleclean"
)

type matchSearchQuery struct {
	Value string
	Year  *int
	Label string
}

type scoredMatchCandidate struct {
	result        searchResult
	confidence    float64
	matchedQuery  string
	reasonSummary string
	titleScore    float64
	yearScore     float64
	extraScore    float64
}

type metadataSearchItem struct {
	LibraryID     uint
	Type          string
	Title         string
	OriginalTitle string
	SeriesTitle   string
	Overview      string
	SourcePath    string
	Year          *int
	SeasonNumber  *int
	EpisodeNumber *int
}

func (s *Service) collectSearchCandidates(ctx context.Context, cfg config.TMDBConfig, mediaType string, queries []matchSearchQuery, item metadataSearchItem) ([]scoredMatchCandidate, error) {
	if len(queries) == 0 {
		return nil, nil
	}
	bestByID := make(map[int]scoredMatchCandidate)
	for _, query := range queries {
		if strings.TrimSpace(query.Value) == "" {
			continue
		}
		response, err := s.searchTMDB(ctx, cfg, mediaType, query.Value, query.Year)
		if err != nil {
			return nil, err
		}
		for _, result := range response.Results {
			candidate := scoreMatchCandidate(item, mediaType, query, result)
			current, exists := bestByID[result.ID]
			if !exists || scoredCandidateLess(current, candidate) {
				bestByID[result.ID] = candidate
			}
		}
	}
	results := make([]scoredMatchCandidate, 0, len(bestByID))
	for _, candidate := range bestByID {
		results = append(results, candidate)
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].confidence == results[j].confidence {
			if results[i].titleScore == results[j].titleScore {
				return results[i].result.ID < results[j].result.ID
			}
			return results[i].titleScore > results[j].titleScore
		}
		return results[i].confidence > results[j].confidence
	})
	return results, nil
}

func buildSearchQueries(item metadataSearchItem, mediaType string) []matchSearchQuery {
	titleSources := []string{}
	if sourcePath := strings.TrimSpace(item.SourcePath); sourcePath != "" {
		fileBase := strings.TrimSuffix(path.Base(sourcePath), path.Ext(sourcePath))
		if mediaType == "movie" {
			movieSources := []string{fileBase, path.Base(path.Dir(sourcePath))}
			if titleOverlapsSourceFilename(item.Title, fileBase) {
				movieSources = append(movieSources, item.Title)
			}
			if titleOverlapsSourceFilename(item.OriginalTitle, fileBase) {
				movieSources = append(movieSources, item.OriginalTitle)
			}
			return buildQueryVariants(movieSources, item.Year)
		}
	}
	if mediaType == "tv" {
		titleSources = append(titleSources, item.SeriesTitle)
	}
	titleSources = append(titleSources, item.Title, item.OriginalTitle, item.SeriesTitle)
	if sourcePath := strings.TrimSpace(item.SourcePath); sourcePath != "" {
		fileBase := strings.TrimSuffix(path.Base(sourcePath), path.Ext(sourcePath))
		titleSources = append(titleSources, fileBase, path.Base(path.Dir(sourcePath)))
		if mediaType == "tv" {
			showFolder := path.Base(path.Dir(path.Dir(sourcePath)))
			titleSources = append(titleSources, showFolder)
		}
	}
	return buildQueryVariants(titleSources, item.Year)
}

func buildManualSearchQueries(input ManualSearchInput, item metadataSearchItem, mediaType string) []matchSearchQuery {
	if title := strings.TrimSpace(input.Title); title != "" {
		return buildQueryVariants([]string{title}, input.Year)
	}
	queries := buildSearchQueries(item, mediaType)
	if input.Year == nil {
		return queries
	}
	updated := make([]matchSearchQuery, 0, len(queries))
	for _, query := range queries {
		query.Year = input.Year
		updated = append(updated, query)
	}
	return updated
}

func buildQueryVariants(values []string, year *int) []matchSearchQuery {
	seen := make(map[string]struct{})
	queries := make([]matchSearchQuery, 0, len(values)*2)
	for _, value := range values {
		for _, normalized := range []string{strings.TrimSpace(value), cleanSearchTitle(value)} {
			trimmed := strings.TrimSpace(normalized)
			if trimmed == "" || isGenericSearchQuery(trimmed) {
				continue
			}
			key := strings.ToLower(trimmed)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			queries = append(queries, matchSearchQuery{Value: trimmed, Year: year, Label: trimmed})
		}
	}
	return queries
}

func isGenericSearchQuery(input string) bool {
	normalized := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(input)), " "))
	genericQueries := map[string]struct{}{
		"电影":     {},
		"movies": {},
		"movie":  {},
		"films":  {},
		"film":   {},
		"video":  {},
		"videos": {},
	}
	_, ok := genericQueries[normalized]
	return ok
}

func titleOverlapsSourceFilename(title string, fileBase string) bool {
	titleText := normalizeMatchText(title)
	fileText := normalizeMatchText(fileBase)
	if titleText == "" || fileText == "" {
		return false
	}
	if strings.Contains(strings.ReplaceAll(titleText, " ", ""), strings.ReplaceAll(fileText, " ", "")) || strings.Contains(strings.ReplaceAll(fileText, " ", ""), strings.ReplaceAll(titleText, " ", "")) {
		return true
	}
	return tokenOverlap(significantMatchTokens(titleText), significantMatchTokens(fileText)) > 0
}

func cleanSearchTitle(input string) string {
	return titleclean.Normalize(titleclean.NormalizeInput{RawTitle: input}).Title
}

func scoreMatchCandidate(item metadataSearchItem, mediaType string, query matchSearchQuery, result searchResult) scoredMatchCandidate {
	titleScore := bestTitleScore(query.Value, resultTitles(result, mediaType))
	yearScore, yearReason := scoreResultYear(item.Year, mediaType, result)
	extraScore, extraReason := scoreResultSignals(result)
	confidence := titleScore + yearScore + extraScore
	if confidence < 0.05 {
		confidence = 0.05
	}
	if confidence > 0.99 {
		confidence = 0.99
	}
	reasons := []string{titleScoreSummary(titleScore, query.Value)}
	if yearReason != "" {
		reasons = append(reasons, yearReason)
	}
	if extraReason != "" {
		reasons = append(reasons, extraReason)
	}
	return scoredMatchCandidate{
		result:        result,
		confidence:    confidence,
		matchedQuery:  query.Value,
		reasonSummary: strings.Join(reasons, "，"),
		titleScore:    titleScore,
		yearScore:     yearScore,
		extraScore:    extraScore,
	}
}

func acceptableAutomatedMatchCandidate(candidate NormalizedMetadataCandidate) bool {
	if !strings.Contains(candidate.ReasonSummary, "标题弱匹配") {
		return true
	}
	queryTokens := significantMatchTokens(normalizeMatchText(candidate.MatchedQuery))
	if len(queryTokens) < 2 {
		return true
	}
	candidateTokens := significantMatchTokens(normalizeMatchText(candidate.Title + " " + candidate.OriginalTitle))
	return tokenOverlap(queryTokens, candidateTokens) > 0
}

func significantMatchTokens(input string) []string {
	tokens := splitTokens(input)
	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if _, ok := map[string]struct{}{"a": {}, "an": {}, "the": {}, "to": {}, "of": {}, "and": {}}[token]; ok {
			continue
		}
		result = append(result, token)
	}
	return result
}

func scoredCandidateLess(current, next scoredMatchCandidate) bool {
	if next.confidence != current.confidence {
		return next.confidence > current.confidence
	}
	if next.titleScore != current.titleScore {
		return next.titleScore > current.titleScore
	}
	if next.yearScore != current.yearScore {
		return next.yearScore > current.yearScore
	}
	return next.result.ID < current.result.ID
}

func resultTitles(result searchResult, mediaType string) []string {
	titles := []string{result.Title, result.OriginalTitle, result.Name, result.OriginalName}
	if mediaType == "movie" {
		titles = []string{result.Title, result.OriginalTitle}
	}
	if mediaType == "tv" {
		titles = []string{result.Name, result.OriginalName, result.Title, result.OriginalTitle}
	}
	return titles
}

func bestTitleScore(query string, titles []string) float64 {
	best := 0.0
	normalizedQuery := normalizeMatchText(query)
	if normalizedQuery == "" {
		return best
	}
	for _, title := range titles {
		score := compareNormalizedTitles(normalizedQuery, normalizeMatchText(title))
		if score > best {
			best = score
		}
	}
	return best
}

func compareNormalizedTitles(query, candidate string) float64 {
	if query == "" || candidate == "" {
		return 0
	}
	if query == candidate {
		return 0.9
	}
	if strings.ReplaceAll(query, " ", "") == strings.ReplaceAll(candidate, " ", "") {
		return 0.86
	}
	if strings.Contains(candidate, query) || strings.Contains(query, candidate) {
		return 0.74
	}
	queryTokens := splitTokens(query)
	candidateTokens := splitTokens(candidate)
	if len(queryTokens) == 0 || len(candidateTokens) == 0 {
		return 0
	}
	overlap := tokenOverlap(queryTokens, candidateTokens)
	if overlap >= 1 {
		return 0.82
	}
	if overlap >= 0.66 {
		return 0.68
	}
	if overlap >= 0.5 {
		return 0.56
	}
	if overlap >= 0.34 {
		return 0.42
	}
	return overlap * 0.3
}

func scoreResultYear(targetYear *int, mediaType string, result searchResult) (float64, string) {
	if targetYear == nil {
		return 0, ""
	}
	resultYear := parseYear(result.ReleaseDate)
	if mediaType == "tv" {
		resultYear = parseYear(result.FirstAirDate)
	}
	if resultYear == nil {
		return 0, ""
	}
	delta := *resultYear - *targetYear
	if delta < 0 {
		delta = -delta
	}
	switch delta {
	case 0:
		return 0.16, "年份完全一致"
	case 1:
		return 0.07, "年份接近"
	case 2:
		return -0.03, "年份有偏差"
	default:
		return -0.14, "年份冲突"
	}
}

func scoreResultSignals(result searchResult) (float64, string) {
	if result.VoteCount >= 500 {
		return 0.05, "结果信号较强"
	}
	if result.VoteCount >= 100 {
		return 0.03, "结果信号稳定"
	}
	if result.Popularity >= 20 {
		return 0.02, "结果热度较高"
	}
	return 0, ""
}

func titleScoreSummary(score float64, query string) string {
	switch {
	case score >= 0.75:
		return "标题高度匹配"
	case score >= 0.62:
		return "标题基本匹配"
	case score >= 0.48:
		return "标题部分匹配"
	default:
		return "标题弱匹配（query: " + query + ")"
	}
}

func normalizeMatchText(input string) string {
	trimmed := cleanSearchTitle(input)
	if trimmed == "" {
		trimmed = input
	}
	trimmed = strings.ToLower(strings.TrimSpace(trimmed))
	return strings.Join(strings.Fields(trimmed), " ")
}

func splitTokens(input string) []string {
	if input == "" {
		return nil
	}
	return strings.Fields(input)
}

func tokenOverlap(left, right []string) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	lookup := make(map[string]struct{}, len(right))
	for _, token := range right {
		lookup[token] = struct{}{}
	}
	matches := 0
	seen := make(map[string]struct{}, len(left))
	for _, token := range left {
		if _, exists := seen[token]; exists {
			continue
		}
		seen[token] = struct{}{}
		if _, exists := lookup[token]; exists {
			matches++
		}
	}
	denominator := len(seen)
	if len(lookup) > denominator {
		denominator = len(lookup)
	}
	if denominator == 0 {
		return 0
	}
	return float64(matches) / float64(denominator)
}
