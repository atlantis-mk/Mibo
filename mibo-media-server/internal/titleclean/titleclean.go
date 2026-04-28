package titleclean

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const NormalizationVersion = "titleclean-v2"

type NormalizeInput struct {
	RawTitle string
}

type NormalizeResult struct {
	Title                string
	Year                 *int
	RemovedTokens        []RemovedToken
	NormalizationVersion string
}

type RemovedToken struct {
	Value  string `json:"value"`
	Reason string `json:"reason"`
}

var (
	yearTokenPattern         = regexp.MustCompile(`^(?:19|20)\d{2}$`)
	episodeCodePattern       = regexp.MustCompile(`(?i)^s\d{1,2}e\d{1,3}(?:e\d{1,3})?$`)
	multiEpisodeRangePattern = regexp.MustCompile(`(?i)^(.*?)([\s._-]+s\d{1,2}e\d{1,3}(?:-e?\d{1,3}|e\d{1,3})(?:[\s._-]+.*)?)$`)
	bracketedWebsitePattern  = regexp.MustCompile(`(?i)[\[【(]\s*((?:https?://)?(?:www\.)?[a-z0-9][a-z0-9-]*(?:\.[a-z0-9][a-z0-9-]*)+[^\]】)]*)\s*[\]】)]`)
	standaloneWebsitePattern = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?[a-z0-9][a-z0-9-]*(?:\.[a-z0-9][a-z0-9-]*)*\.(?:com|net|org|cn|tv|io)\b`)
)

func Normalize(input NormalizeInput) NormalizeResult {
	raw := strings.TrimSpace(input.RawTitle)
	result := NormalizeResult{NormalizationVersion: NormalizationVersion}
	if raw == "" {
		return result
	}

	cleanableRaw := raw
	if prefix, removed, ok := stripMultiEpisodeRange(raw); ok {
		cleanableRaw = prefix
		result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: removed, Reason: "episode_range"})
	}

	withoutWebsites, websiteTokens := removeWebsiteTokens(cleanableRaw)
	for _, token := range websiteTokens {
		result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: token, Reason: "website"})
	}

	normalized := normalizeSeparators(withoutWebsites)
	tokens := strings.Fields(normalized)
	kept := make([]string, 0, len(tokens))
	for idx := 0; idx < len(tokens); idx++ {
		token := strings.Trim(tokens[idx], "-_.()[]{}【】")
		if token == "" {
			continue
		}
		if yearTokenPattern.MatchString(token) {
			if result.Year == nil {
				if year, err := strconv.Atoi(token); err == nil {
					result.Year = &year
				}
			}
			result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: token, Reason: "year"})
			continue
		}
		if idx+1 < len(tokens) {
			next := strings.Trim(tokens[idx+1], "-_.()[]{}【】")
			if reason, ok := classifyPair(token, next); ok {
				result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: token + " " + next, Reason: reason})
				idx++
				continue
			}
		}
		if reason, ok := classifyToken(token); ok {
			result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: token, Reason: reason})
			continue
		}
		kept = append(kept, token)
	}

	kept, removedGroups := stripTrailingReleaseGroups(kept)
	for _, token := range removedGroups {
		result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: token, Reason: "release_group"})
	}

	title := strings.TrimSpace(strings.Join(kept, " "))
	title = strings.Join(strings.Fields(title), " ")
	if unusableTitle(title) {
		title = strings.TrimSpace(raw)
	}
	result.Title = title
	return result
}

func stripMultiEpisodeRange(input string) (string, string, bool) {
	match := multiEpisodeRangePattern.FindStringSubmatch(strings.TrimSpace(input))
	if len(match) < 3 {
		return input, "", false
	}
	prefix := strings.TrimSpace(match[1])
	removed := strings.TrimSpace(match[2])
	if prefix == "" || removed == "" {
		return input, "", false
	}
	return prefix, strings.TrimLeft(removed, " ._-"), true
}

func removeWebsiteTokens(input string) (string, []string) {
	var removed []string
	withoutBracketed := bracketedWebsitePattern.ReplaceAllStringFunc(input, func(match string) string {
		groups := bracketedWebsitePattern.FindStringSubmatch(match)
		if len(groups) > 1 {
			removed = append(removed, strings.TrimSpace(groups[1]))
		}
		return " "
	})
	withoutStandalone := standaloneWebsitePattern.ReplaceAllStringFunc(withoutBracketed, func(match string) string {
		trimmed := strings.TrimSpace(match)
		if looksLikeWebsite(trimmed) {
			removed = append(removed, trimmed)
			return " "
		}
		return match
	})
	return withoutStandalone, removed
}

func looksLikeWebsite(input string) bool {
	lower := strings.ToLower(strings.TrimSpace(input))
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "www.") || strings.HasSuffix(lower, ".com") || strings.HasSuffix(lower, ".net") || strings.HasSuffix(lower, ".org") || strings.HasSuffix(lower, ".cn") || strings.HasSuffix(lower, ".tv") || strings.HasSuffix(lower, ".io")
}

func normalizeSeparators(input string) string {
	replacer := strings.NewReplacer(
		".", " ",
		"_", " ",
		"-", " ",
		"[", " ",
		"]", " ",
		"(", " ",
		")", " ",
		"{", " ",
		"}", " ",
		"【", " ",
		"】", " ",
	)
	return strings.Join(strings.Fields(replacer.Replace(strings.TrimSpace(input))), " ")
}

func classifyPair(left, right string) (string, bool) {
	switch normalizeToken(left) + " " + normalizeToken(right) {
	case "web dl", "blu ray", "dolby vision", "dual audio":
		if normalizeToken(left) == "dolby" {
			return "hdr", true
		}
		if normalizeToken(left) == "dual" {
			return "audio", true
		}
		return "source", true
	case "ddp5 1", "aac5 1", "aac2 0":
		return "audio", true
	}
	return "", false
}

func classifyToken(input string) (string, bool) {
	normalized := normalizeToken(input)
	if normalized == "" {
		return "", false
	}
	if _, ok := map[string]struct{}{"2160p": {}, "1080p": {}, "720p": {}, "480p": {}, "4k": {}, "8k": {}, "hd": {}, "uhd": {}, "hd1080p": {}}[normalized]; ok {
		return "quality", true
	}
	if _, ok := map[string]struct{}{"hdr": {}, "hdr10": {}, "hdr10plus": {}, "dv": {}, "dovi": {}}[normalized]; ok {
		return "hdr", true
	}
	if _, ok := map[string]struct{}{"x264": {}, "x265": {}, "h264": {}, "h265": {}, "hevc": {}, "avc": {}}[normalized]; ok {
		return "video_codec", true
	}
	if _, ok := map[string]struct{}{"bluray": {}, "bdrip": {}, "brrip": {}, "bdrmux": {}, "remux": {}, "webrip": {}, "webdl": {}, "hdtv": {}, "uhdrip": {}, "dl": {}}[normalized]; ok {
		return "source", true
	}
	if _, ok := map[string]struct{}{"nf": {}, "netflix": {}, "amzn": {}, "amazon": {}, "dsnp": {}, "disney": {}, "hmax": {}, "max": {}, "hulu": {}, "atvp": {}}[normalized]; ok {
		return "platform", true
	}
	if _, ok := map[string]struct{}{"atmos": {}, "dts": {}, "dtshd": {}, "truehd": {}, "aac": {}, "aac20": {}, "aac51": {}, "ddp": {}, "ddp5": {}, "ddp51": {}, "ac3": {}, "eac3": {}, "51": {}, "71": {}}[normalized]; ok {
		return "audio", true
	}
	if _, ok := map[string]struct{}{"multi": {}, "multisub": {}, "multisubs": {}, "sub": {}, "subs": {}, "subbed": {}, "dub": {}, "dubbed": {}, "chs": {}, "cht": {}, "eng": {}, "jpn": {}, "gb": {}, "big5": {}}[normalized]; ok {
		return "subtitle", true
	}
	if _, ok := map[string]struct{}{"proper": {}, "repack": {}, "extended": {}, "unrated": {}, "limited": {}, "yts": {}, "rarbg": {}, "wiki": {}, "mkv": {}, "mp4": {}}[normalized]; ok {
		return "release_group", true
	}
	if strings.HasPrefix(normalized, "4khdr") {
		return "release_group", true
	}
	return "", false
}

func normalizeToken(input string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(input)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func stripTrailingReleaseGroups(tokens []string) ([]string, []string) {
	kept := append([]string(nil), tokens...)
	var removed []string
	for len(kept) > 0 {
		candidate := strings.TrimSpace(kept[len(kept)-1])
		if !looksLikeReleaseGroupToken(candidate) {
			break
		}
		removed = append(removed, candidate)
		kept = kept[:len(kept)-1]
	}
	return kept, removed
}

func looksLikeReleaseGroupToken(input string) bool {
	trimmed := strings.TrimSpace(input)
	if len(trimmed) < 3 || containsNonASCII(trimmed) {
		return false
	}
	if episodeCodePattern.MatchString(trimmed) {
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

func containsNonASCII(input string) bool {
	for _, r := range input {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}

func unusableTitle(input string) bool {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return true
	}
	return len([]rune(trimmed)) < 2
}
