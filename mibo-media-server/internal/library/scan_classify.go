package library

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/storage"
	"github.com/atlan/mibo-media-server/internal/titleclean"
)

const (
	LibraryTypeMovies = "movies"
	LibraryTypeShows  = "shows"
	LibraryTypeMixed  = "mixed"
	LibraryTypeAuto   = "auto"
)

func classifyMediaFile(libraryType string, libraryRoot string, object storage.Object) classifiedMedia {
	libraryType = effectiveVideoLibraryType(libraryType)
	filenameModel := extractFilenameSignalModel(object.Path)
	fileName := path.Base(object.Path)
	ext := path.Ext(fileName)
	rawTitle := strings.TrimSuffix(fileName, ext)
	normalized := titleclean.Normalize(titleclean.NormalizeInput{RawTitle: rawTitle})
	normalizedTitle := normalized.Title
	isTVLibrary := isTVLibraryType(libraryType)
	pathSeriesTitle := tvSeriesTitleFromPath(libraryRoot, object.Path)
	shouldTryTVPath := isTVLibrary || pathSeriesTitle != "" || tvSeasonFromPath(libraryRoot, object.Path) != nil
	if seriesPrefix, season, episodeNumbers, ok := parseMultiEpisodeRange(rawTitle); ok {
		seriesTitle := cleanTitle(seriesPrefix)
		if pathSeriesTitle != "" {
			seriesTitle = pathSeriesTitle
		}
		title := fmt.Sprintf("%s S%02dE%02d-E%02d", seriesTitle, *season, episodeNumbers[0], episodeNumbers[len(episodeNumbers)-1])
		firstEpisode := episodeNumbers[0]
		return classifiedMedia{Type: "episode", Title: title, OriginalTitle: rawTitle, SeriesTitle: seriesTitle, SeasonNumber: season, EpisodeNumber: &firstEpisode, EpisodeNumbers: episodeNumbers, SourcePath: object.Path, Status: "ready", NormalizationVersion: normalized.NormalizationVersion, RemovedTokens: normalized.RemovedTokens, Tags: normalized.Tags, FilenameSignals: filenameModel}
	}
	if groups := episodePattern.FindStringSubmatch(rawTitle); len(groups) > 0 {
		seriesTitle := cleanTitle(groups[1])
		if pathSeriesTitle != "" {
			seriesTitle = pathSeriesTitle
		}
		season, episode := parseEpisodeNumbers(groups[2], groups[3], groups[4], groups[5])
		title := fmt.Sprintf("%s S%02dE%02d", seriesTitle, *season, *episode)
		return classifiedMedia{Type: "episode", Title: title, OriginalTitle: rawTitle, SeriesTitle: seriesTitle, SeasonNumber: season, EpisodeNumber: episode, EpisodeNumbers: episodeNumbersFromPointer(episode), SourcePath: object.Path, Status: "ready", NormalizationVersion: normalized.NormalizationVersion, RemovedTokens: normalized.RemovedTokens, Tags: normalized.Tags, FilenameSignals: filenameModel}
	}
	if episode := parseEmbeddedEpisodeNumber(rawTitle); episode != nil && shouldUseEmbeddedEpisodeToken(libraryRoot, object.Path, pathSeriesTitle) {
		season := tvSeasonFromPath(libraryRoot, object.Path)
		if season == nil {
			defaultSeason := 1
			season = &defaultSeason
		}
		seriesTitle := embeddedEpisodeSeriesTitle(libraryRoot, object.Path, pathSeriesTitle)
		if seriesTitle != "" {
			title := fmt.Sprintf("%s S%02dE%02d", seriesTitle, *season, *episode)
			return classifiedMedia{Type: "episode", Title: title, OriginalTitle: rawTitle, SeriesTitle: seriesTitle, SeasonNumber: season, EpisodeNumber: episode, EpisodeNumbers: episodeNumbersFromPointer(episode), SourcePath: object.Path, Status: "ready", NormalizationVersion: normalized.NormalizationVersion, RemovedTokens: normalized.RemovedTokens, Tags: normalized.Tags, FilenameSignals: filenameModel}
		}
	}
	if shouldTryTVPath {
		if classified, ok := classifyTVEpisodeFromPath(libraryRoot, object.Path, rawTitle, pathSeriesTitle, normalized); ok {
			classified.FilenameSignals = filenameModel
			return classified
		}
	}
	year := normalized.Year
	title := normalizedTitle
	if moviePathTitle := movieTitleFromPath(object.Path, rawTitle); moviePathTitle != "" {
		title = moviePathTitle
	}
	if isTVLibrary {
		title = titleFromPath(object.Path)
	}
	return classifiedMedia{Type: "movie", Title: title, OriginalTitle: rawTitle, Year: year, SourcePath: object.Path, Status: "ready", NormalizationVersion: normalized.NormalizationVersion, RemovedTokens: normalized.RemovedTokens, Tags: normalized.Tags, FilenameSignals: filenameModel}
}

func parseEmbeddedEpisodeNumber(input string) *int {
	match := embeddedEpisodePattern.FindStringSubmatch(normalizeEpisodeIdentifier(input))
	if len(match) < 2 {
		return nil
	}
	return parseOrdinalToken(match[1])
}

func shouldUseEmbeddedEpisodeToken(libraryRoot string, itemPath string, pathSeriesTitle string) bool {
	if strings.TrimSpace(pathSeriesTitle) == "" {
		return false
	}
	for _, segment := range relativePathSegments(libraryRoot, itemPath) {
		normalized := strings.ToLower(strings.TrimSpace(segment))
		if normalized == "电视剧" || normalized == "剧集" || normalized == "shows" || normalized == "tv" || normalized == "tv shows" || strings.Contains(normalized, "episode") {
			return true
		}
		if embeddedEpisodePattern.MatchString(normalizeEpisodeIdentifier(segment)) {
			return true
		}
	}
	return false
}

func embeddedEpisodeSeriesTitle(libraryRoot string, itemPath string, pathSeriesTitle string) string {
	segments := relativePathSegments(libraryRoot, itemPath)
	if len(segments) >= 2 {
		parent := path.Base(path.Dir(itemPath))
		cleaned := cleanTitle(embeddedEpisodePattern.ReplaceAllString(parent, " "))
		if cleaned != "" {
			return cleaned
		}
	}
	return strings.TrimSpace(pathSeriesTitle)
}

func parseMultiEpisodeRange(input string) (string, *int, []int, bool) {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(.*?)[\s._-]+s(\d{1,2})e(\d{1,2})-e?(\d{1,2})(?:[\s._-]+.*)?$`),
		regexp.MustCompile(`(?i)^(.*?)[\s._-]+s(\d{1,2})e(\d{1,2})e(\d{1,2})(?:[\s._-]+.*)?$`),
	}
	for _, pattern := range patterns {
		match := pattern.FindStringSubmatch(strings.TrimSpace(input))
		if len(match) < 5 {
			continue
		}
		season, err := strconv.Atoi(match[2])
		if err != nil || season <= 0 {
			continue
		}
		startEpisode, err := strconv.Atoi(match[3])
		if err != nil || startEpisode <= 0 {
			continue
		}
		endEpisode, err := strconv.Atoi(match[4])
		if err != nil || endEpisode <= startEpisode {
			continue
		}
		episodeNumbers := make([]int, 0, endEpisode-startEpisode+1)
		for episode := startEpisode; episode <= endEpisode; episode++ {
			episodeNumbers = append(episodeNumbers, episode)
		}
		return strings.TrimSpace(match[1]), &season, episodeNumbers, true
	}
	return "", nil, nil, false
}

func isVideoFile(itemPath string) bool {
	_, ok := videoExtensions[strings.ToLower(path.Ext(itemPath))]
	return ok
}

func parseEpisodeNumbers(seasonLeft, episodeLeft, seasonRight, episodeRight string) (*int, *int) {
	seasonValue := seasonLeft
	episodeValue := episodeLeft
	if seasonValue == "" {
		seasonValue = seasonRight
		episodeValue = episodeRight
	}
	season, _ := strconv.Atoi(seasonValue)
	episode, _ := strconv.Atoi(episodeValue)
	return &season, &episode
}

func parseYear(input string) *int {
	match := yearPattern.FindStringSubmatch(input)
	if len(match) < 2 {
		return nil
	}
	value, err := strconv.Atoi(match[1])
	if err != nil {
		return nil
	}
	return &value
}

func titleFromPath(itemPath string) string {
	parent := path.Base(path.Dir(itemPath))
	if parent == "/" || parent == "." || parent == "" {
		return cleanTitle(strings.TrimSuffix(path.Base(itemPath), path.Ext(itemPath)))
	}
	return cleanTitle(parent)
}

func movieTitleFromPath(itemPath string, rawTitle string) string {
	cleanedRaw := cleanTitle(rawTitle)
	parent := cleanTitle(path.Base(path.Dir(itemPath)))
	if parent == "" {
		return cleanedRaw
	}
	if cleanedRaw == "" || isGenericMediaName(cleanedRaw) {
		return parent
	}
	return cleanedRaw
}

func isGenericMediaName(input string) bool {
	return genericMediaNamePattern.MatchString(strings.TrimSpace(input))
}

func isTVLibraryType(libraryType string) bool {
	switch normalizeLibraryType(libraryType) {
	case LibraryTypeShows:
		return true
	default:
		return false
	}
}

func isMovieLibraryType(libraryType string) bool {
	switch normalizeLibraryType(libraryType) {
	case LibraryTypeMovies:
		return true
	default:
		return false
	}
}

func isMixedLibraryType(libraryType string) bool {
	return normalizeLibraryType(libraryType) == LibraryTypeMixed
}

func normalizeLibraryType(libraryType string) string {
	switch strings.ToLower(strings.TrimSpace(libraryType)) {
	case "movie", "movies", "films":
		return LibraryTypeMovies
	case "tv", "tvshows", "shows":
		return LibraryTypeShows
	case "mixed", "mixed-content", "mixed_content":
		return LibraryTypeMixed
	case "", "auto", "source", "source-first", "source_first":
		return LibraryTypeAuto
	default:
		return strings.ToLower(strings.TrimSpace(libraryType))
	}
}

func effectiveVideoLibraryType(libraryType string) string {
	if normalizeLibraryType(libraryType) == LibraryTypeAuto {
		return LibraryTypeMixed
	}
	return libraryType
}

func classifyTVEpisodeFromPath(libraryRoot string, itemPath string, rawTitle string, pathSeriesTitle string, normalized titleclean.NormalizeResult) (classifiedMedia, bool) {
	seriesTitle := strings.TrimSpace(pathSeriesTitle)
	if seriesTitle == "" {
		seriesTitle = tvSeriesTitleFromPath(libraryRoot, itemPath)
	}
	if seriesTitle == "" {
		return classifiedMedia{}, false
	}
	season := tvSeasonFromPath(libraryRoot, itemPath)
	if season == nil {
		return classifiedMedia{}, false
	}
	episode := parseEpisodeNumberFromTitle(rawTitle, seriesTitle)
	if episode == nil {
		return classifiedMedia{}, false
	}
	title := fmt.Sprintf("%s S%02dE%02d", seriesTitle, *season, *episode)
	return classifiedMedia{Type: "episode", Title: title, OriginalTitle: rawTitle, SeriesTitle: seriesTitle, SeasonNumber: season, EpisodeNumber: episode, EpisodeNumbers: episodeNumbersFromPointer(episode), SourcePath: itemPath, Status: "ready", NormalizationVersion: normalized.NormalizationVersion, RemovedTokens: normalized.RemovedTokens, Tags: normalized.Tags}, true
}

func hasExplicitEpisodeMarker(input string) bool {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return false
	}
	if _, _, _, ok := parseMultiEpisodeRange(trimmed); ok {
		return true
	}
	if episodePattern.MatchString(trimmed) {
		return true
	}
	normalized := normalizeEpisodeIdentifier(trimmed)
	for _, pattern := range []*regexp.Regexp{episodeOnlyPattern, embeddedEpisodePattern, chineseEpisodePattern} {
		if pattern.MatchString(normalized) {
			return true
		}
	}
	return false
}

func episodeNumbersFromPointer(episode *int) []int {
	if episode == nil || *episode <= 0 {
		return nil
	}
	return []int{*episode}
}

func tvSeriesTitleFromPath(libraryRoot string, itemPath string) string {
	segments := relativePathSegments(libraryRoot, itemPath)
	if len(segments) < 2 {
		return ""
	}
	seriesIndex := len(segments) - 2
	if seasonIndex := tvSeasonDirectoryIndex(segments); seasonIndex >= 0 {
		if title := seriesTitleFromEmbeddedSeasonDirectory(segments[seasonIndex]); title != "" {
			return title
		}
		if seasonIndex > 0 {
			seriesIndex = seasonIndex - 1
		}
	}
	if seriesIndex < 0 || seriesIndex >= len(segments)-1 {
		return ""
	}
	return cleanTitle(segments[seriesIndex])
}

func tvSeasonFromPath(libraryRoot string, itemPath string) *int {
	segments := relativePathSegments(libraryRoot, itemPath)
	if len(segments) < 2 {
		return nil
	}
	if seasonIndex := tvSeasonDirectoryIndex(segments); seasonIndex >= 0 {
		return parseSeasonDirectoryNumber(segments[seasonIndex])
	}
	return nil
}

func tvSeasonDirectoryIndex(segments []string) int {
	if len(segments) < 2 {
		return -1
	}
	for idx := len(segments) - 2; idx >= 0; idx-- {
		if parseSeasonDirectoryNumber(segments[idx]) != nil {
			return idx
		}
	}
	return -1
}

func relativePathSegments(libraryRoot string, itemPath string) []string {
	normalizedRoot := strings.TrimSpace(path.Clean(libraryRoot))
	normalizedPath := strings.TrimSpace(path.Clean(itemPath))
	if normalizedPath == "" || normalizedPath == "." {
		return nil
	}
	if normalizedRoot == "." {
		normalizedRoot = ""
	}
	if normalizedRoot != "" && normalizedRoot != "/" {
		if normalizedPath != normalizedRoot && !strings.HasPrefix(normalizedPath, normalizedRoot+"/") {
			return nil
		}
		normalizedPath = strings.TrimPrefix(normalizedPath, normalizedRoot)
	}
	normalizedPath = strings.TrimPrefix(normalizedPath, "/")
	if normalizedPath == "" {
		return nil
	}
	return strings.Split(normalizedPath, "/")
}

func parseSeasonDirectoryNumber(input string) *int {
	match := seasonDirectoryPattern.FindStringSubmatch(strings.TrimSpace(input))
	if len(match) >= 3 {
		if value := parseOrdinalToken(match[1]); value != nil {
			return value
		}
		if value := parseOrdinalToken(match[2]); value != nil {
			return value
		}
	}
	embedded := embeddedSeasonDirPattern.FindStringSubmatch(strings.TrimSpace(input))
	if len(embedded) < 4 {
		return nil
	}
	for _, group := range []string{embedded[2], embedded[3]} {
		if value := parseOrdinalToken(group); value != nil {
			return value
		}
	}
	return nil
}

func seriesTitleFromEmbeddedSeasonDirectory(input string) string {
	match := embeddedSeasonDirPattern.FindStringSubmatch(strings.TrimSpace(input))
	if len(match) < 4 {
		return ""
	}
	if title := primarySeriesTitleFromGroup(groupWithoutTrailingYear(match[1])); title != "" {
		return title
	}
	return ""
}

func groupWithoutTrailingYear(input string) string {
	return strings.TrimSpace(yearPattern.ReplaceAllString(strings.TrimSpace(input), " "))
}

func primarySeriesTitleFromGroup(input string) string {
	title := cleanTitle(input)
	if title == "" {
		return ""
	}
	if primary := leadingNonASCIITitle(title); primary != "" {
		return primary
	}
	return title
}

func leadingNonASCIITitle(input string) string {
	tokens := strings.Fields(strings.TrimSpace(input))
	if len(tokens) < 2 || !containsNonASCIIForTitle(tokens[0]) {
		return ""
	}
	kept := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if !containsNonASCIIForTitle(token) {
			break
		}
		kept = append(kept, token)
	}
	return strings.Join(kept, " ")
}

func containsNonASCIIForTitle(input string) bool {
	for _, r := range input {
		if r > 127 {
			return true
		}
	}
	return false
}

func parseEpisodeNumberFromTitle(input string, seriesTitle string) *int {
	candidates := []string{normalizeEpisodeIdentifier(input)}
	if withoutSeries := stripSeriesPrefix(candidates[0], seriesTitle); withoutSeries != "" {
		candidates = append(candidates, withoutSeries)
	}
	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		for _, pattern := range []*regexp.Regexp{episodeOnlyPattern, embeddedEpisodePattern, chineseEpisodePattern, numericEpisodePattern, trailingEpisodePattern} {
			match := pattern.FindStringSubmatch(trimmed)
			if len(match) < 2 {
				continue
			}
			if value := parseOrdinalToken(match[1]); value != nil {
				return value
			}
		}
	}
	return nil
}

func normalizeEpisodeIdentifier(input string) string {
	replacer := strings.NewReplacer(".", " ", "_", " ", "-", " ")
	cleaned := replacer.Replace(strings.TrimSpace(input))
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return strings.TrimSpace(cleaned)
}

func stripSeriesPrefix(input string, seriesTitle string) string {
	trimmedInput := strings.TrimSpace(input)
	trimmedSeries := strings.TrimSpace(normalizeEpisodeIdentifier(seriesTitle))
	if trimmedInput == "" || trimmedSeries == "" {
		return trimmedInput
	}
	trimmedInputLower := strings.ToLower(trimmedInput)
	trimmedSeriesLower := strings.ToLower(trimmedSeries)
	if !strings.HasPrefix(trimmedInputLower, trimmedSeriesLower) {
		return trimmedInput
	}
	stripped := strings.TrimSpace(trimmedInput[len(trimmedSeries):])
	stripped = strings.TrimLeft(stripped, "-_. ")
	return strings.TrimSpace(stripped)
}

func parseOrdinalToken(input string) *int {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil
	}
	if value, err := strconv.Atoi(trimmed); err == nil && value > 0 {
		return &value
	}
	value, ok := parseChineseNumber(trimmed)
	if !ok || value <= 0 {
		return nil
	}
	return &value
}

func parseChineseNumber(input string) (int, bool) {
	trimmed := strings.NewReplacer("两", "二", "零", "").Replace(strings.TrimSpace(input))
	if trimmed == "" {
		return 0, false
	}
	values := map[rune]int{'一': 1, '二': 2, '三': 3, '四': 4, '五': 5, '六': 6, '七': 7, '八': 8, '九': 9}
	if !strings.ContainsRune(trimmed, '十') {
		total := 0
		for _, r := range trimmed {
			value, ok := values[r]
			if !ok {
				return 0, false
			}
			total += value
		}
		return total, true
	}
	parts := strings.SplitN(trimmed, "十", 2)
	tens := 1
	if parts[0] != "" {
		value, ok := values[[]rune(parts[0])[0]]
		if !ok {
			return 0, false
		}
		tens = value
	}
	ones := 0
	if len(parts) > 1 && parts[1] != "" {
		value, ok := values[[]rune(parts[1])[0]]
		if !ok {
			return 0, false
		}
		ones = value
	}
	return tens*10 + ones, true
}

func normalizeSeriesGroupingTitle(input string) string {
	cleaned := cleanTitle(input)
	normalized := strings.TrimSpace(trailingSeasonTitlePattern.ReplaceAllString(cleaned, ""))
	normalized = strings.Join(strings.Fields(normalized), " ")
	if normalized == "" {
		return cleaned
	}
	return normalized
}

func cleanTitle(input string) string {
	return titleclean.Normalize(titleclean.NormalizeInput{RawTitle: input}).Title
}

func stripTrailingReleaseGroup(input string) string {
	tokens := strings.Fields(strings.TrimSpace(input))
	for len(tokens) > 0 {
		candidate := strings.Trim(tokens[len(tokens)-1], "-_.()[]{}")
		if !looksLikeReleaseGroupToken(candidate) {
			break
		}
		tokens = tokens[:len(tokens)-1]
	}
	return strings.Join(tokens, " ")
}

func looksLikeReleaseGroupToken(input string) bool {
	trimmed := strings.TrimSpace(input)
	if len(trimmed) < 3 {
		return false
	}
	hasUpper := false
	hasLower := false
	for _, r := range trimmed {
		if r >= 'A' && r <= 'Z' {
			hasUpper = true
		}
		if r >= 'a' && r <= 'z' {
			hasLower = true
		}
	}
	return hasUpper && !hasLower
}

func equalIntPointers(left, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
