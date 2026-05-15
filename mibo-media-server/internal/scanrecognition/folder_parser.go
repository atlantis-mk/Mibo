package scanrecognition

import (
	"regexp"
	"strconv"
	"strings"
)

type FolderSignal struct {
	TitleCandidates      []string
	Season               *int
	ExpectedEpisodeCount *int
	Year                 *int
	ReleaseTokens        []string
	HasSeasonMarker      bool
}

var (
	englishSeasonPattern        = regexp.MustCompile(`(?i)(?:^|\b)season\s*0*(\d{1,2})(?:\b|$)`)
	shortSeasonPattern          = regexp.MustCompile(`(?i)(?:^|\b)S0*(\d{1,2})(?:\b|$)`)
	partSeasonPattern           = regexp.MustCompile(`(?i)(?:^|\b)part\s*0*(\d{1,2})(?:\b|$)`)
	chineseSeasonPattern        = regexp.MustCompile(`第\s*([一二三四五六七八九十两零0-9]{1,4})\s*季`)
	expectedEpisodeCountPattern = regexp.MustCompile(`全\s*0*(\d{1,3})\s*集`)
	episodePackSuffixPattern    = regexp.MustCompile(`(?i)(?:^|[ ._-])(E0*\d{1,3}(?:-E0*\d{1,3})?(?:\+SP)?)$`)
	bracketContentPattern       = regexp.MustCompile(`\[[^\]]*\]`)
)

func ParseFolderName(name string) FolderSignal {
	folderName := normalizeReleaseText(strings.TrimSpace(name))
	signal := FolderSignal{
		Year:          parseYearPointer(folderName),
		ReleaseTokens: parseReleaseTokens(folderName),
	}
	if season, ok := parseFolderSeason(folderName); ok {
		signal.Season = &season
		signal.HasSeasonMarker = true
	}
	if episodeCount, ok := parseExpectedEpisodeCount(folderName); ok {
		signal.ExpectedEpisodeCount = &episodeCount
	}
	signal.TitleCandidates = folderTitleCandidates(folderName, signal)
	return signal
}

func parseFolderSeason(input string) (int, bool) {
	for _, pattern := range []*regexp.Regexp{englishSeasonPattern, shortSeasonPattern, partSeasonPattern} {
		matches := pattern.FindStringSubmatch(input)
		if len(matches) < 2 {
			continue
		}
		value, err := strconv.Atoi(matches[1])
		return value, err == nil
	}

	matches := chineseSeasonPattern.FindStringSubmatch(input)
	if len(matches) < 2 {
		return 0, false
	}
	return parseChineseNumber(matches[1])
}

func parseExpectedEpisodeCount(input string) (int, bool) {
	matches := expectedEpisodeCountPattern.FindStringSubmatch(input)
	if len(matches) < 2 {
		return 0, false
	}
	value, err := strconv.Atoi(matches[1])
	return value, err == nil
}

func folderTitleCandidates(input string, signal FolderSignal) []string {
	cleaned := bracketContentPattern.ReplaceAllString(input, " ")
	segments := folderTitleSegments(cleaned, signal)
	candidates := make([]string, 0, len(segments))
	seen := map[string]struct{}{}
	for _, segment := range segments {
		for _, candidate := range splitFolderTitleCandidates(segment) {
			key := strings.ToLower(candidate)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	return candidates
}

func folderTitleSegments(input string, signal FolderSignal) []string {
	if match := folderSeasonMarkerIndex(input); len(match) > 0 {
		segments := []string{trimFolderTitleSegment(input[:match[0]], signal)}
		trailing := trimFolderTitleSegment(input[match[1]:], signal)
		if strings.HasPrefix(strings.TrimSpace(input[match[1]:]), ".") {
			segments = append(segments, trailing)
		}
		return segments
	}
	return []string{trimFolderTitleSegment(input, signal)}
}

func folderSeasonMarkerIndex(input string) []int {
	for _, pattern := range []*regexp.Regexp{chineseSeasonPattern, englishSeasonPattern, shortSeasonPattern, partSeasonPattern} {
		if match := pattern.FindStringIndex(input); len(match) > 0 {
			return match
		}
	}
	return nil
}

func trimFolderTitleSegment(input string, signal FolderSignal) string {
	cutoff := len(input)
	if signal.Year != nil {
		if matches := yearPattern.FindAllStringIndex(input, -1); len(matches) > 0 {
			cutoff = matches[len(matches)-1][0]
		}
	} else if match := releaseTokenPattern.FindStringIndex(input); len(match) > 0 {
		cutoff = match[0]
	}
	segment := strings.Trim(input[:cutoff], " ._-[](){}")
	if match := episodePackSuffixPattern.FindStringIndex(segment); len(match) > 0 {
		segment = strings.TrimSpace(segment[:match[0]])
	}
	return strings.Trim(segment, " ._-[](){}")
}

func splitFolderTitleCandidates(input string) []string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) > 1 && containsCJK(parts[0]) {
		candidates := make([]string, 0, 2)
		if title := normalizeTitle(parts[0]); title != "" {
			candidates = append(candidates, title)
		}
		if title := normalizeTitle(strings.Join(parts[1:], ".")); title != "" {
			candidates = append(candidates, title)
		}
		return candidates
	}
	if title := normalizeTitle(trimmed); title != "" {
		return []string{title}
	}
	return nil
}

func containsCJK(input string) bool {
	for _, value := range input {
		if value >= '一' && value <= '鿿' {
			return true
		}
	}
	return false
}

func parseChineseNumber(input string) (int, bool) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return 0, false
	}
	if value, err := strconv.Atoi(trimmed); err == nil {
		return value, true
	}

	digits := map[rune]int{'零': 0, '一': 1, '二': 2, '两': 2, '三': 3, '四': 4, '五': 5, '六': 6, '七': 7, '八': 8, '九': 9}
	if trimmed == "十" {
		return 10, true
	}
	parts := []rune(trimmed)
	if len(parts) == 2 && parts[0] == '十' {
		value, ok := digits[parts[1]]
		return 10 + value, ok
	}
	if len(parts) == 2 && parts[1] == '十' {
		value, ok := digits[parts[0]]
		return value * 10, ok
	}
	if len(parts) == 3 && parts[1] == '十' {
		tens, tensOK := digits[parts[0]]
		ones, onesOK := digits[parts[2]]
		return tens*10 + ones, tensOK && onesOK
	}
	if len(parts) == 1 {
		value, ok := digits[parts[0]]
		return value, ok
	}
	return 0, false
}
