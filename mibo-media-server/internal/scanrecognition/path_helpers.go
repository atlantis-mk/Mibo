package scanrecognition

import (
	"path"
	"strings"

	"github.com/atlan/mibo-media-server/internal/titleclean"
)

type TitleToken struct {
	Value string
	Kept  bool
}

type CleanupEvidence struct {
	Token  string
	Reason string
}

func CleanTitle(input string) string {
	return strings.TrimSpace(titleclean.NormalizeMovieWorkTitle(strings.TrimSpace(input)))
}

func NormalizeSeriesGroupingTitle(input string) string {
	cleaned := CleanTitle(input)
	if cleaned == "" {
		return ""
	}
	return strings.Join(strings.Fields(cleaned), " ")
}

func PrimarySeriesTitleFromGroup(input string) string {
	title := CleanTitle(input)
	if title == "" {
		return ""
	}
	if primary := leadingNonASCIITitle(title); primary != "" {
		return primary
	}
	return title
}

func RelativePathSegments(rootPath string, itemPath string) []string {
	normalizedRoot := strings.TrimSpace(path.Clean(rootPath))
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

func SeriesTitleFromPath(rootPath string, itemPath string) string {
	segments := RelativePathSegments(rootPath, itemPath)
	if len(segments) < 2 {
		return ""
	}
	seriesIndex := len(segments) - 2
	if seasonIndex := seasonDirectoryIndex(segments); seasonIndex >= 0 {
		if title := firstTitle(ParseFolderName(segments[seasonIndex]).TitleCandidates); title != "" {
			return title
		}
		if seasonIndex > 0 {
			seriesIndex = seasonIndex - 1
		}
	}
	if seriesIndex < 0 || seriesIndex >= len(segments)-1 {
		return ""
	}
	return CleanTitle(segments[seriesIndex])
}

func SeasonFromPath(rootPath string, itemPath string) *int {
	segments := RelativePathSegments(rootPath, itemPath)
	if len(segments) < 2 {
		return nil
	}
	if seasonIndex := seasonDirectoryIndex(segments); seasonIndex >= 0 {
		return ParseFolderName(segments[seasonIndex]).Season
	}
	return nil
}

func LeadingNumericToken(rawTitle string) *int {
	match := leadingNumericTokenProfilePattern.FindStringSubmatch(strings.TrimSpace(rawTitle))
	if len(match) < 2 {
		return nil
	}
	return parseOrdinalToken(match[1])
}

func EpisodeNumbersFromPointer(episode *int) []int {
	if episode == nil || *episode <= 0 {
		return nil
	}
	return []int{*episode}
}

func TokenizeTitle(rawTitle string) []string {
	tokens := strings.Fields(strings.NewReplacer(".", " ", "_", " ", "-", " ").Replace(rawTitle))
	if len(tokens) == 0 {
		return nil
	}
	return tokens
}

func BuildTitleTokens(rawTitle string, removedValues map[string]struct{}) []TitleToken {
	rawTokens := TokenizeTitle(rawTitle)
	if len(rawTokens) == 0 {
		return nil
	}
	items := make([]TitleToken, 0, len(rawTokens))
	for _, token := range rawTokens {
		trimmed := strings.TrimSpace(token)
		if trimmed == "" {
			continue
		}
		_, removed := removedValues[strings.ToLower(trimmed)]
		items = append(items, TitleToken{Value: trimmed, Kept: !removed && !SuppressedTitleToken(trimmed)})
	}
	return items
}

func SuppressedTitleToken(token string) bool {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return true
	}
	lower := strings.ToLower(trimmed)
	if QualitySignal(trimmed) != "" || strings.Contains(lower, "atmos") || strings.Contains(lower, "truehd") || strings.Contains(lower, "eac3") || strings.Contains(lower, "ac3") || strings.Contains(lower, "ddp") || strings.Contains(lower, "aac") || strings.Contains(lower, "dts") {
		return true
	}
	switch lower {
	case "trailer", "sample", "preview", "featurette", "extra", "extras", "behind", "scenes", "pv":
		return true
	default:
		return false
	}
}

func seasonDirectoryIndex(segments []string) int {
	if len(segments) < 2 {
		return -1
	}
	for idx := len(segments) - 2; idx >= 0; idx-- {
		if ParseFolderName(segments[idx]).Season != nil {
			return idx
		}
	}
	return -1
}

func firstTitle(candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}
	return strings.TrimSpace(candidates[0])
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
