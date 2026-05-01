package library

import (
	"path"
	"regexp"
	"strings"

	"github.com/atlan/mibo-media-server/internal/storage"
)

var (
	qualitySignalPattern = regexp.MustCompile(`(?i)(2160p|1080p|720p|480p|4k|uhd|hdr10\+?|dv|dolby[\s._-]?vision|hevc|x265|h265|avc|x264|h264|web[\s._-]?dl|webrip|bluray|remux)`)
	editionSignalPattern = regexp.MustCompile(`(?i)(director'?s[\s._-]?cut|extended|unrated|theatrical|imax|criterion|proper|repack)`)
)

type filenameSignals struct {
	RawTitle          string
	TitleCandidate    string
	YearCandidate     *int
	SeasonNumber      *int
	EpisodeNumber     *int
	EpisodeNumberEnd  *int
	EpisodeNumbers    []int
	EpisodeSource     string
	QualityLabel      string
	Edition           string
	ReleaseGroup      string
	SourceTags        []string
	ExtraType         string
	IsSample          bool
	IsTrailer         bool
	IsExtra           bool
	ClassificationRef classifiedMedia
}

func resolveFilenameSignals(libraryType string, libraryRoot string, object storage.Object) filenameSignals {
	fileName := path.Base(object.Path)
	rawTitle := strings.TrimSuffix(fileName, path.Ext(fileName))
	classified := classifyMediaFile(libraryType, libraryRoot, object)
	signals := filenameSignals{
		RawTitle:          rawTitle,
		TitleCandidate:    classified.Title,
		YearCandidate:     classified.Year,
		SeasonNumber:      classified.SeasonNumber,
		EpisodeNumber:     classified.EpisodeNumber,
		EpisodeNumbers:    append([]int(nil), classified.EpisodeNumbers...),
		EpisodeSource:     explicitEpisodeSource(rawTitle, classified),
		QualityLabel:      qualitySignal(rawTitle),
		Edition:           editionSignal(rawTitle),
		ReleaseGroup:      releaseGroupSignal(rawTitle),
		SourceTags:        sourceTagSignals(rawTitle),
		ExtraType:         extraTypeSignal(rawTitle),
		ClassificationRef: classified,
	}
	if signals.EpisodeNumber == nil {
		if episode := parseEpisodeNumberFromTitle(rawTitle, classified.SeriesTitle); episode != nil {
		signals.EpisodeNumber = episode
		signals.EpisodeNumbers = episodeNumbersFromPointer(episode)
		signals.EpisodeSource = "filename"
		}
	}
	if len(signals.EpisodeNumbers) > 1 {
		last := signals.EpisodeNumbers[len(signals.EpisodeNumbers)-1]
		signals.EpisodeNumberEnd = &last
	}
	signals.IsSample = signals.ExtraType == "sample"
	signals.IsTrailer = signals.ExtraType == "trailer"
	signals.IsExtra = signals.ExtraType != ""
	return signals
}

func explicitEpisodeSource(rawTitle string, classified classifiedMedia) string {
	if classified.EpisodeNumber == nil && len(classified.EpisodeNumbers) == 0 {
		return ""
	}
	if hasExplicitEpisodeMarker(rawTitle) {
		return "explicit"
	}
	return "weak"
}

func qualitySignal(input string) string {
	matches := qualitySignalPattern.FindAllString(strings.TrimSpace(input), -1)
	if len(matches) == 0 {
		return ""
	}
	seen := make(map[string]struct{}, len(matches))
	parts := make([]string, 0, len(matches))
	for _, match := range matches {
		normalized := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(match, ".", ""), "_", "-"))
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		parts = append(parts, normalized)
	}
	return strings.Join(parts, " ")
}

func editionSignal(input string) string {
	match := editionSignalPattern.FindString(strings.TrimSpace(input))
	return strings.TrimSpace(cleanTitle(match))
}

func releaseGroupSignal(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	idx := strings.LastIndex(trimmed, "-")
	if idx < 0 || idx == len(trimmed)-1 {
		return ""
	}
	candidate := strings.TrimSpace(trimmed[idx+1:])
	if looksLikeReleaseGroupToken(candidate) {
		return candidate
	}
	return ""
}

func sourceTagSignals(input string) []string {
	matches := qualitySignalPattern.FindAllString(strings.TrimSpace(input), -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(matches))
	tags := make([]string, 0, len(matches))
	for _, match := range matches {
		tag := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(match, ".", ""), "_", "-"))
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	return tags
}

func extraTypeSignal(input string) string {
	normalized := strings.ToLower(normalizeEpisodeIdentifier(input))
	switch {
	case containsNormalizedToken(normalized, "sample"):
		return "sample"
	case containsNormalizedToken(normalized, "trailer") || strings.Contains(normalized, "预告"):
		return "trailer"
	case containsNormalizedPhrase(normalized, "behind", "the", "scenes") || containsNormalizedPhrase(normalized, "making", "of"):
		return "behind_the_scenes"
	case containsNormalizedToken(normalized, "featurette"):
		return "featurette"
	case containsNormalizedToken(normalized, "interview"):
		return "interview"
	case containsNormalizedPhrase(normalized, "deleted", "scene"):
		return "deleted_scene"
	default:
		return ""
	}
}

func containsNormalizedToken(normalized string, token string) bool {
	for _, field := range strings.Fields(normalized) {
		if field == token {
			return true
		}
	}
	return false
}

func containsNormalizedPhrase(normalized string, tokens ...string) bool {
	fields := strings.Fields(normalized)
	if len(tokens) == 0 || len(fields) < len(tokens) {
		return false
	}
	for idx := 0; idx <= len(fields)-len(tokens); idx++ {
		matched := true
		for tokenIdx, token := range tokens {
			if fields[idx+tokenIdx] != token {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}
