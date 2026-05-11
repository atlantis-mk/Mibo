package titleclean

import (
	"fmt"
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
	Tags                 []string
	RemovedTokens        []RemovedToken
	NormalizationVersion string
}

type RemovedToken struct {
	Value  string `json:"value"`
	Reason string `json:"reason"`
}

var (
	yearTokenPattern          = regexp.MustCompile(`^(?:19|20)\d{2}$`)
	episodeCodePattern        = regexp.MustCompile(`(?i)^s\d{1,2}e\d{1,3}(?:e\d{1,3})?$`)
	multiEpisodeRangePattern  = regexp.MustCompile(`(?i)^(.*?)([\s._-]+s\d{1,2}e\d{1,3}(?:-e?\d{1,3}|e\d{1,3})(?:[\s._-]+.*)?)$`)
	standaloneWebsitePattern  = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?[a-z0-9][a-z0-9-]*(?:\.[a-z0-9][a-z0-9-]*)*\.(?:com|net|org|cn|tv|io)\b`)
	bracketedTokenPattern     = regexp.MustCompile(`[\[【(]([^\]】)]*)[\]】)]`)
	hashtagTokenPattern       = regexp.MustCompile(`(^|[\s._\-\[【(])#([\pL\pN_][\pL\pN_-]*)`)
	hyphenReleasePattern      = regexp.MustCompile(`(?i)(.*(?:\b(?:19|20)\d{2}\b|\b(?:2160p|1080p|720p|480p|web[-._ ]?dl|web[-._ ]?rip|blu[-._ ]?ray|h\.?26[45]|x26[45]|hevc|aac|ddp|ac3|eac3)\b).*)-([a-z0-9][a-z0-9._-]{2,})$`)
	fullEpisodeCountPattern   = regexp.MustCompile(`^全\d+集$`)
	fpsTokenPattern           = regexp.MustCompile(`^\d{2,3}fps$`)
	bitDepthTokenPattern      = regexp.MustCompile(`^\d{1,2}bit$`)
	multiAudioTokenPattern    = regexp.MustCompile(`^\d+audios?$`)
	audioSubtitleTokenPattern = regexp.MustCompile(`^(?:[257]1|ddp[257]1|aac[257]1|truehd[257]1|dd[257]1)(?:sub|subs|subtitle|subtitles)$`)
	audioCodecChannelPattern  = regexp.MustCompile(`^(?:dts(?:hd)?(?:ma)?|hdma|ma|truehdatmos|truehd|ddp|dd|aac|ac3|eac3)(?:20|51|71)$`)
	movieWorkQualityPattern   = regexp.MustCompile(`(?i)(2160p|1080p|720p|480p|4k|uhd|hdr10\+?|dv|dolby[\s._-]?vision|hevc|x265|h265|avc|x264|h264|web[\s._-]?dl|webrip|bluray|remux)`)
	movieWorkEditionPattern   = regexp.MustCompile(`(?i)(director'?s[\s._-]?cut|extended|unrated|theatrical|imax|criterion|proper|repack)`)
	movieWorkAudioPattern     = regexp.MustCompile(`(?i)\b(?:(?:ddp?|aac|dts|truehd|atmos|eac3|ac3)\s*)?[57](?:\s|\.)1\b`)
)

func Normalize(input NormalizeInput) NormalizeResult {
	raw := strings.TrimSpace(input.RawTitle)
	result := NormalizeResult{NormalizationVersion: NormalizationVersion}
	if raw == "" {
		return result
	}

	cleanableRaw := raw
	withoutHashtags, tags := removeHashtagTokens(cleanableRaw)
	cleanableRaw = withoutHashtags
	for _, tag := range tags {
		result.Tags = append(result.Tags, tag)
		result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: "#" + tag, Reason: "hashtag"})
	}
	if prefix, removed, ok := stripMultiEpisodeRange(cleanableRaw); ok {
		cleanableRaw = prefix
		result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: removed, Reason: "episode_range"})
	}
	if prefix, removed, ok := stripHyphenReleaseGroup(cleanableRaw); ok {
		cleanableRaw = prefix
		result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: removed, Reason: "release_group"})
	}
	withoutWebsites, websiteTokens := removeWebsiteTokens(cleanableRaw)
	for _, token := range websiteTokens {
		result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: token, Reason: "website"})
	}

	withoutBracketedMetadata, bracketedTokens := removeBracketedMetadataTokens(withoutWebsites)
	for _, token := range bracketedTokens {
		result.RemovedTokens = append(result.RemovedTokens, token)
	}

	normalized := normalizeSeparators(withoutBracketedMetadata)
	tokens := strings.Fields(normalized)
	kept := make([]string, 0, len(tokens))
	seenReleaseMetadata := false
	for idx := 0; idx < len(tokens); idx++ {
		token := strings.Trim(tokens[idx], "-_.()[]{}【】")
		if token == "" {
			continue
		}
		if boundary, ok := classifyReleaseTailBoundary(tokens, idx); ok {
			result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: strings.Join(cleanTokenWindow(tokens, idx, boundary), " "), Reason: "release_tail"})
			result.RemovedTokens = append(result.RemovedTokens, releaseTailRemovedTokens(tokens, idx)...)
			if yearTokenPattern.MatchString(token) && result.Year == nil {
				if year, err := strconv.Atoi(token); err == nil {
					result.Year = &year
				}
			}
			break
		}
		if yearTokenPattern.MatchString(token) {
			seenReleaseMetadata = true
			if result.Year == nil {
				if year, err := strconv.Atoi(token); err == nil {
					result.Year = &year
				}
			}
			result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: token, Reason: "year"})
			continue
		}
		if seenReleaseMetadata {
			if count, ok := classifyAudioTokenSequence(tokens, idx); ok {
				result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: strings.Join(cleanTokenWindow(tokens, idx, count), " "), Reason: "audio"})
				idx += count - 1
				continue
			}
		}
		if idx+1 < len(tokens) {
			next := strings.Trim(tokens[idx+1], "-_.()[]{}【】")
			if idx+2 < len(tokens) {
				third := strings.Trim(tokens[idx+2], "-_.()[]{}【】")
				if reason, ok := classifyTriple(token, next, third); ok {
					seenReleaseMetadata = true
					result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: token + " " + next + " " + third, Reason: reason})
					idx += 2
					continue
				}
			}
			if reason, ok := classifyPair(token, next); ok {
				seenReleaseMetadata = true
				result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: token + " " + next, Reason: reason})
				idx++
				continue
			}
		}
		if reason, ok := classifyToken(token); ok {
			seenReleaseMetadata = true
			result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: token, Reason: reason})
			continue
		}
		if seenReleaseMetadata && idx > 0 {
			previous := strings.Trim(tokens[idx-1], "-_.()[]{}【】")
			if looksLikeAudioTailToken(token, previous) {
				result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: token, Reason: "audio"})
				continue
			}
		}
		if seenReleaseMetadata && looksLikeShortReleaseToken(token) {
			result.RemovedTokens = append(result.RemovedTokens, RemovedToken{Value: token, Reason: "release_group"})
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

func MovieWorkTitle(input string) string {
	cleaned := Normalize(NormalizeInput{RawTitle: strings.TrimSpace(input)}).Title
	cleaned = movieWorkQualityPattern.ReplaceAllString(cleaned, " ")
	cleaned = movieWorkEditionPattern.ReplaceAllString(cleaned, " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if cleaned == "" {
		cleaned = strings.TrimSpace(input)
	}
	parts := strings.Fields(strings.NewReplacer(".", " ", "-", " ", "_", " ").Replace(cleaned))
	kept := parts[:0]
	for idx, part := range parts {
		if suppressMovieWorkToken(part) {
			continue
		}
		if idx > 0 && movieWorkAudioTailToken(part, parts[idx-1]) {
			continue
		}
		kept = append(kept, part)
	}
	if len(kept) == 0 {
		return strings.Join(strings.Fields(cleaned), " ")
	}
	return strings.Join(kept, " ")
}

func NormalizeMovieWorkTitle(input string) string {
	return strings.ToLower(MovieWorkTitle(input))
}

func removeHashtagTokens(input string) (string, []string) {
	seen := map[string]struct{}{}
	tags := []string{}
	cleaned := hashtagTokenPattern.ReplaceAllStringFunc(input, func(match string) string {
		groups := hashtagTokenPattern.FindStringSubmatch(match)
		if len(groups) < 3 {
			return match
		}
		tag := strings.TrimSpace(groups[2])
		key := strings.ToLower(tag)
		if tag != "" {
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				tags = append(tags, tag)
			}
		}
		return groups[1]
	})
	return cleaned, tags
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

func stripHyphenReleaseGroup(input string) (string, string, bool) {
	match := hyphenReleasePattern.FindStringSubmatch(strings.TrimSpace(input))
	if len(match) < 3 {
		return input, "", false
	}
	prefix := strings.TrimSpace(match[1])
	removed := strings.TrimSpace(match[2])
	if prefix == "" || removed == "" {
		return input, "", false
	}
	return prefix, removed, true
}

func removeWebsiteTokens(input string) (string, []string) {
	var removed []string
	withoutBracketed := bracketedTokenPattern.ReplaceAllStringFunc(input, func(match string) string {
		groups := bracketedTokenPattern.FindStringSubmatch(match)
		if len(groups) > 1 {
			content := strings.TrimSpace(groups[1])
			if standaloneWebsitePattern.MatchString(content) {
				removed = append(removed, content)
				return " "
			}
		}
		return match
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

func removeBracketedMetadataTokens(input string) (string, []RemovedToken) {
	var removed []RemovedToken
	withoutBracketed := bracketedTokenPattern.ReplaceAllStringFunc(input, func(match string) string {
		groups := bracketedTokenPattern.FindStringSubmatch(match)
		if len(groups) < 2 {
			return match
		}
		content := strings.TrimSpace(groups[1])
		if reason, ok := classifyBracketedMetadata(content); ok {
			removed = append(removed, RemovedToken{Value: content, Reason: reason})
			return " "
		}
		return match
	})
	return withoutBracketed, removed
}

func classifyBracketedMetadata(input string) (string, bool) {
	normalized := normalizeToken(input)
	if fullEpisodeCountPattern.MatchString(strings.TrimSpace(input)) || fullEpisodeCountPattern.MatchString(normalized) {
		return "episode_count", true
	}
	if strings.Contains(input, "字幕") || strings.Contains(input, "配音") || strings.Contains(input, "中字") || strings.Contains(input, "国语") {
		return "subtitle", true
	}
	if looksLikeReleaseGroupToken(normalized) || isKnownReleaseGroup(normalized) {
		return "release_group", true
	}
	return "", false
}

func looksLikeWebsite(input string) bool {
	lower := strings.ToLower(strings.TrimSpace(input))
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "www.") || strings.HasSuffix(lower, ".com") || strings.HasSuffix(lower, ".net") || strings.HasSuffix(lower, ".org") || strings.HasSuffix(lower, ".cn") || strings.HasSuffix(lower, ".tv") || strings.HasSuffix(lower, ".io")
}

func normalizeSeparators(input string) string {
	input = preserveNumericVersionDots(input)
	replacer := strings.NewReplacer(
		".", " ",
		"\x00", ".",
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

func preserveNumericVersionDots(input string) string {
	runes := []rune(input)
	for idx, r := range runes {
		if r != '.' || idx == 0 || idx >= len(runes)-1 {
			continue
		}
		left, right := runes[idx-1], runes[idx+1]
		if !unicode.IsDigit(left) || !unicode.IsDigit(right) {
			continue
		}
		leftBoundary := idx == 1 || isSeparatorRune(runes[idx-2])
		rightBoundary := idx+2 >= len(runes) || isSeparatorRune(runes[idx+2])
		if leftBoundary && rightBoundary {
			runes[idx] = '\x00'
		}
	}
	return string(runes)
}

func isSeparatorRune(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune("._-[](){}【】", r)
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
	case "h 264", "h 265", "x 264", "x 265":
		return "video_codec", true
	}
	return "", false
}

func classifyReleaseTailBoundary(tokens []string, start int) (int, bool) {
	if start < 0 || start >= len(tokens) {
		return 0, false
	}
	current := strings.Trim(tokens[start], "-_.()[]{}【】")
	if current == "" {
		return 0, false
	}
	if yearTokenPattern.MatchString(current) {
		if start+1 < len(tokens) && startsReleaseTail(tokens, start+1) {
			return len(tokens) - start, true
		}
		return 0, false
	}
	if start > 0 && startsReleaseTail(tokens, start) {
		return len(tokens) - start, true
	}
	return 0, false
}

func startsReleaseTail(tokens []string, start int) bool {
	if start < 0 || start >= len(tokens) {
		return false
	}
	current := strings.Trim(tokens[start], "-_.()[]{}【】")
	if current == "" {
		return false
	}
	if reason, ok := classifyToken(current); ok {
		switch reason {
		case "quality", "source", "video_codec", "hdr", "platform":
			return true
		}
	}
	if start+1 < len(tokens) {
		next := strings.Trim(tokens[start+1], "-_.()[]{}【】")
		if reason, ok := classifyPair(current, next); ok {
			switch reason {
			case "source", "video_codec", "hdr":
				return true
			}
		}
	}
	return false
}

func releaseTailRemovedTokens(tokens []string, start int) []RemovedToken {
	removed := make([]RemovedToken, 0, len(tokens)-start)
	for idx := start; idx < len(tokens); idx++ {
		token := strings.Trim(tokens[idx], "-_.()[]{}【】")
		if token == "" {
			continue
		}
		if yearTokenPattern.MatchString(token) {
			removed = append(removed, RemovedToken{Value: token, Reason: "year"})
			continue
		}
		if count, ok := classifyAudioTokenSequence(tokens, idx); ok {
			removed = append(removed, RemovedToken{Value: strings.Join(cleanTokenWindow(tokens, idx, count), " "), Reason: "audio"})
			idx += count - 1
			continue
		}
		if idx+1 < len(tokens) {
			next := strings.Trim(tokens[idx+1], "-_.()[]{}【】")
			if idx+2 < len(tokens) {
				third := strings.Trim(tokens[idx+2], "-_.()[]{}【】")
				if reason, ok := classifyTriple(token, next, third); ok {
					removed = append(removed, RemovedToken{Value: token + " " + next + " " + third, Reason: reason})
					idx += 2
					continue
				}
			}
			if reason, ok := classifyPair(token, next); ok {
				removed = append(removed, RemovedToken{Value: token + " " + next, Reason: reason})
				idx++
				continue
			}
		}
		if reason, ok := classifyToken(token); ok {
			removed = append(removed, RemovedToken{Value: token, Reason: reason})
			continue
		}
		if looksLikeReleaseGroupToken(token) || looksLikeShortReleaseToken(token) {
			removed = append(removed, RemovedToken{Value: token, Reason: "release_group"})
		}
	}
	return removed
}

func classifyAudioTokenSequence(tokens []string, start int) (int, bool) {
	max := 5
	if remaining := len(tokens) - start; remaining < max {
		max = remaining
	}
	for count := max; count > 0; count-- {
		combined := normalizedTokenWindow(tokens, start, count)
		if audioCodecChannelPattern.MatchString(combined) {
			return count, true
		}
	}
	return 0, false
}

func normalizedTokenWindow(tokens []string, start int, count int) string {
	var builder strings.Builder
	for _, token := range cleanTokenWindow(tokens, start, count) {
		builder.WriteString(normalizeToken(token))
	}
	return builder.String()
}

func cleanTokenWindow(tokens []string, start int, count int) []string {
	window := make([]string, 0, count)
	for idx := start; idx < len(tokens) && idx < start+count; idx++ {
		window = append(window, strings.Trim(tokens[idx], "-_.()[]{}【】"))
	}
	return window
}

func classifyTriple(left, middle, right string) (string, bool) {
	combined := normalizeToken(left) + " " + normalizeToken(middle) + " " + normalizeToken(right)
	switch combined {
	case "dd 5 1sub", "dd 7 1sub", "ddp 5 1sub", "ddp 7 1sub", "truehd 5 1sub", "truehd 7 1sub", "aac 5 1sub", "aac 7 1sub":
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
	if _, ok := map[string]struct{}{"bluray": {}, "bdrip": {}, "brrip": {}, "bdrmux": {}, "remux": {}, "web": {}, "webrip": {}, "webdl": {}, "hdtv": {}, "uhdrip": {}, "dl": {}}[normalized]; ok {
		return "source", true
	}
	if _, ok := map[string]struct{}{"nf": {}, "netflix": {}, "amzn": {}, "amazon": {}, "dsnp": {}, "disney": {}, "hmax": {}, "max": {}, "hulu": {}, "atvp": {}}[normalized]; ok {
		return "platform", true
	}
	if _, ok := map[string]struct{}{"atmos": {}, "dts": {}, "dtshd": {}, "hdma5": {}, "hdma7": {}, "hdma51": {}, "hdma71": {}, "truehd": {}, "truehd5": {}, "truehd7": {}, "truehd51": {}, "truehd71": {}, "aac": {}, "aac20": {}, "aac51": {}, "ddp": {}, "ddp5": {}, "ddp51": {}, "ddp71": {}, "ac3": {}, "eac3": {}, "51": {}, "71": {}}[normalized]; ok {
		return "audio", true
	}
	if _, ok := map[string]struct{}{"ma5": {}, "ma7": {}, "ma51": {}, "ma71": {}}[normalized]; ok {
		return "audio", true
	}
	if _, ok := map[string]struct{}{"flac": {}, "opus": {}, "pcm": {}, "lpcm": {}}[normalized]; ok {
		return "audio", true
	}
	if fpsTokenPattern.MatchString(normalized) {
		return "frame_rate", true
	}
	if bitDepthTokenPattern.MatchString(normalized) {
		return "video_codec", true
	}
	if multiAudioTokenPattern.MatchString(normalized) {
		return "audio", true
	}
	if audioSubtitleTokenPattern.MatchString(normalized) {
		return "audio", true
	}
	if _, ok := map[string]struct{}{"multi": {}, "multisub": {}, "multisubs": {}, "sub": {}, "subs": {}, "subbed": {}, "dub": {}, "dubbed": {}, "chs": {}, "cht": {}, "eng": {}, "jpn": {}, "gb": {}, "big5": {}}[normalized]; ok {
		return "subtitle", true
	}
	if _, ok := map[string]struct{}{"proper": {}, "repack": {}, "extended": {}, "unrated": {}, "limited": {}, "mkv": {}, "mp4": {}}[normalized]; ok {
		return "release_group", true
	}
	if isKnownReleaseGroup(normalized) {
		return "release_group", true
	}
	if strings.HasPrefix(normalized, "4khdr") {
		return "release_group", true
	}
	return "", false
}

func isKnownReleaseGroup(normalized string) bool {
	_, ok := map[string]struct{}{"yts": {}, "ytsbz": {}, "rarbg": {}, "wiki": {}}[normalized]
	return ok
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
		if len(kept) == 1 && hasMixedCaseLetters(candidate) {
			break
		}
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
	if hasUpper && hasLower && upperRuneCount(trimmed) >= 2 && !strings.ContainsAny(trimmed, " ") {
		return true
	}
	return hasUpper && !hasLower
}

func upperRuneCount(input string) int {
	count := 0
	for _, r := range input {
		if r >= 'A' && r <= 'Z' {
			count++
		}
	}
	return count
}

func hasMixedCaseLetters(input string) bool {
	hasUpper := false
	hasLower := false
	for _, r := range input {
		if r >= 'A' && r <= 'Z' {
			hasUpper = true
		}
		if r >= 'a' && r <= 'z' {
			hasLower = true
		}
	}
	return hasUpper && hasLower
}

func looksLikeShortReleaseToken(input string) bool {
	trimmed := strings.TrimSpace(input)
	if len(trimmed) < 2 || len(trimmed) > 4 || containsNonASCII(trimmed) {
		return false
	}
	for _, r := range trimmed {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

func looksLikeAudioTailToken(input string, previous string) bool {
	trimmed := strings.TrimSpace(input)
	prev := normalizeToken(previous)
	if trimmed != "1" || prev == "" {
		return false
	}
	return strings.HasPrefix(prev, "ma") || strings.HasPrefix(prev, "hdma") || prev == "dts" || prev == "hd" || prev == "ddp" || prev == "dd" || prev == "aac" || prev == "eac3" || prev == "truehd"
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

func suppressMovieWorkToken(token string) bool {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return true
	}
	lower := strings.ToLower(trimmed)
	if movieWorkQualityPattern.MatchString(trimmed) || movieWorkAudioPattern.MatchString(lower) {
		return true
	}
	if reason, ok := classifyToken(trimmed); ok {
		switch reason {
		case "quality", "hdr", "video_codec", "source", "platform", "audio", "subtitle", "frame_rate", "release_group":
			return true
		}
	}
	if strings.HasPrefix(lower, "ma") && len(lower) > 2 {
		if _, err := fmt.Sscanf(lower, "ma%d", new(int)); err == nil {
			return true
		}
	}
	if _, err := fmt.Sscanf(lower, "%dx%d", new(int), new(int)); err == nil {
		return true
	}
	return false
}

func movieWorkAudioTailToken(token string, previous string) bool {
	lower := strings.ToLower(strings.TrimSpace(token))
	prev := strings.ToLower(strings.TrimSpace(previous))
	if lower == "" || prev == "" {
		return false
	}
	if _, err := fmt.Sscanf(lower, "%d", new(int)); err != nil {
		return false
	}
	return strings.HasPrefix(prev, "ma") || prev == "dts" || prev == "hd" || prev == "ddp" || prev == "dd" || prev == "aac" || prev == "eac3"
}
