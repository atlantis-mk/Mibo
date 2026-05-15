package scanrecognition

import (
	"path"
	"regexp"
	"strconv"
	"strings"
)

var (
	qualitySignalPattern              = regexp.MustCompile(`(?i)(2160p|1080p|720p|480p|4k|uhd|hdr10\+?|dv|dolby[\s._-]?vision|hevc|x265|h265|avc|x264|h264|web[\s._-]?dl|webrip|bluray|remux)`)
	editionSignalPattern              = regexp.MustCompile(`(?i)(director'?s[\s._-]?cut|extended|unrated|theatrical|imax|criterion|proper|repack)`)
	audioChannelPattern               = regexp.MustCompile(`(?i)\b(?:(?:ddp?|aac|dts|truehd|atmos|eac3|ac3)\s*)?[57](?:\s|\.)1\b`)
	episodePattern                    = regexp.MustCompile(`(?i)^(.*?)[\s._-]+(?:s(\d{1,2})e(\d{1,2})|(\d{1,2})x(\d{1,2}))(?:[\s._-]+.*)?$`)
	episodeOnlyPattern                = regexp.MustCompile(`(?i)^(?:e|ep|episode)[\s._-]*0*(\d{1,3})(?:[\s._-]+.*)?$`)
	embeddedEpisodePattern            = regexp.MustCompile(`(?i)(?:^|[\s._-])(?:e|ep|episode)[\s._-]*0*([1-9]\d{0,2})(?:$|[\s._-])`)
	bracketWebsiteTokenProfilePattern = regexp.MustCompile(`(?i)[\[【(（]([^\]】)）]*(?:网站|www\.|\.com|\.net|\.org)[^\]】)）]*)[\]】)）]`)
	leadingNumericTokenProfilePattern = regexp.MustCompile(`^0*([1-9]\d{0,2})(?:$|[\s._-]+.*)`)
	genericMediaNamePattern           = regexp.MustCompile(`(?i)^(movie|video|feature|main|full|default|film|media|sample)$`)
)

func QualitySignal(input string) string {
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

func SourceTagSignals(input string) []string {
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

func EditionSignal(input string) string {
	match := editionSignalPattern.FindString(strings.TrimSpace(input))
	return strings.TrimSpace(normalizeTitle(match))
}

func ReleaseGroupSignal(input string) string {
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

func WebsiteSignal(input string) string {
	match := bracketWebsiteTokenProfilePattern.FindStringSubmatch(strings.TrimSpace(input))
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func GenericMediaNameSignal(input string) string {
	if genericMediaNamePattern.MatchString(strings.TrimSpace(input)) {
		return strings.TrimSpace(input)
	}
	return ""
}

func CodecSignal(input string) string {
	return codecSignal(input)
}

func AudioSignal(input string) string {
	return audioSignal(input)
}

func SubtitleSignal(input string) string {
	return subtitleSignal(input)
}

func HDRSignal(input string) string {
	return hdrSignal(input)
}

func VideoFileRoleSignal(itemPath string) string {
	segments := strings.Split(strings.Trim(path.Clean(itemPath), "/"), "/")
	for _, segment := range segments {
		candidate := strings.TrimSuffix(path.Base(segment), path.Ext(segment))
		if role := extraTypeSignal(candidate); role != "" {
			return role
		}
	}
	return ""
}

func HasExplicitEpisodeMarker(input string) bool {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return false
	}
	if episodePattern.MatchString(trimmed) || episodeOnlyPattern.MatchString(trimmed) {
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

func WeakEpisodeNumberAllowed(rawTitle string) bool {
	if HasExplicitEpisodeMarker(rawTitle) {
		return true
	}
	normalized := strings.ToLower(normalizeEpisodeIdentifier(rawTitle))
	if audioChannelPattern.MatchString(normalized) {
		return false
	}
	if QualitySignal(rawTitle) != "" && (strings.Contains(normalized, "h 264") || strings.Contains(normalized, "h 265") || strings.Contains(normalized, "x264") || strings.Contains(normalized, "x265") || strings.Contains(normalized, "hevc")) {
		return false
	}
	if yearPattern.MatchString(rawTitle) && (QualitySignal(rawTitle) != "" || strings.Contains(normalized, "web dl") || strings.Contains(normalized, "web rip") || strings.Contains(normalized, "bluray")) {
		return false
	}
	return true
}

func parseOrdinalToken(input string) *int {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil || value <= 0 {
		return nil
	}
	return &value
}

func normalizeEpisodeIdentifier(input string) string {
	replacer := strings.NewReplacer(".", " ", "_", " ", "-", " ", "[", " ", "]", " ", "(", " ", ")", " ", "{", " ", "}", " ")
	return strings.TrimSpace(replacer.Replace(strings.TrimSpace(input)))
}

func looksLikeReleaseGroupToken(input string) bool {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return false
	}
	for _, r := range trimmed {
		if r == ' ' || r == '.' || r == '_' || r == '-' {
			continue
		}
		if r < '0' || (r > '9' && r < 'A') || (r > 'Z' && r < 'a') || r > 'z' {
			return false
		}
	}
	return true
}

func extraTypeSignal(input string) string {
	normalized := strings.ToLower(normalizeEpisodeIdentifier(input))
	switch {
	case containsNormalizedToken(normalized, "sample"):
		return "sample"
	case containsNormalizedToken(normalized, "trailer") || containsNormalizedToken(normalized, "teaser") || strings.Contains(normalized, "预告"):
		return "trailer"
	case containsNormalizedPhrase(normalized, "behind", "the", "scenes") || containsNormalizedPhrase(normalized, "making", "of") || strings.Contains(normalized, "花絮") || strings.Contains(normalized, "幕后"):
		return "behind_the_scenes"
	case containsNormalizedToken(normalized, "featurette") || strings.Contains(normalized, "特典") || strings.Contains(normalized, "特辑"):
		return "featurette"
	case regexp.MustCompile(`(?i)^pv\d*$`).MatchString(strings.ReplaceAll(normalized, " ", "")) || strings.Contains(normalized, "先导"):
		return "preview"
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

func codecSignal(input string) string {
	return firstSignalByPattern(input, regexp.MustCompile(`(?i)(x265|h265|hevc|x264|h264|avc)`))
}

func audioSignal(input string) string {
	return firstSignalByPattern(input, regexp.MustCompile(`(?i)(atmos|truehd|eac3|ac3|ddp?|aac|dts)`))
}

func subtitleSignal(input string) string {
	return firstSignalByPattern(input, regexp.MustCompile(`(?i)(sub(?:bed|s)?|chs|cht|eng|jpn|big5|multi(?:sub|subs)?)`))
}

func hdrSignal(input string) string {
	return firstSignalByPattern(input, regexp.MustCompile(`(?i)(hdr10\+?|dv|dolby[\s._-]?vision|hdr)`))
}

func firstSignalByPattern(input string, pattern *regexp.Regexp) string {
	match := pattern.FindString(strings.TrimSpace(input))
	return strings.TrimSpace(match)
}
