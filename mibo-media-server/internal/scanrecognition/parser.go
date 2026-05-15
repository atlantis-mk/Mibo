package scanrecognition

import (
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/atlan/mibo-media-server/internal/titleclean"
)

type VideoSignal struct {
	TitleCandidates []string
	Year            *int
	Season          *int
	Episode         *int
	EpisodeEnd      *int
	EpisodeNumbers  []int
	EpisodeSource   string
	IsSpecial       bool
	SpecialIndex    *int
	ReleaseTokens   []string
	Quality         string
	SourceTags      []string
	Codec           string
	Audio           string
	Subtitle        string
	HDR             string
	Edition         string
	ReleaseGroup    string
	Website         string
	GenericNoise    string
	Role            string
	IsMain          bool
	IsSample        bool
	IsTrailer       bool
	IsExtra         bool
	LeadingNumber   *int
	TitleTokens     []TitleToken
	CleanupEvidence []CleanupEvidence
}

var (
	sxxexxPattern                 = regexp.MustCompile(`(?i)S0*(\d{1,2})E0*(\d{1,3})`)
	separatedSeasonEpisodePattern = regexp.MustCompile(`(?i)S0*(\d{1,2})[ ._-]+E0*(\d{1,3})`)
	xEpisodePattern               = regexp.MustCompile(`(?i)(?:^|[^0-9])0*(\d{1,2})x0*(\d{1,3})(?:[^0-9]|$)`)
	episodeWordPattern            = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(?:EP|Episode)\s*0*(\d{1,3})(?:[^a-z0-9]|$)`)
	episodeMarkerPattern          = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])E0*(\d{1,3})(?:[^a-z0-9]|$)`)
	episodeZeroMarkerPattern      = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])E0+(?:[^a-z0-9]|$)`)
	chineseEpisodePattern         = regexp.MustCompile(`第\s*0*(\d{1,3})\s*集`)
	episodeRangeTokenPattern      = regexp.MustCompile(`(?i)^E0*\d{1,3}-E0*\d{1,3}$`)
	specialEpisodeTokenPattern    = regexp.MustCompile(`(?i)^\+SP$`)
	specialEpisodeIndexPattern    = regexp.MustCompile(`(?i)(?:^|[._\-\s])(SP|OVA|OAD|NCOP|NCED|SPECIAL)0*(\d{1,3})(?:$|[._\-\s])`)
	specialEpisodeMarkerPattern   = regexp.MustCompile(`(?i)(?:^|[._\-\s])(SP|OVA|OAD|NCOP|NCED|SPECIAL)(?:$|[._\-\s])`)
	yearPattern                   = regexp.MustCompile(`(?:19|20)\d{2}`)
	releaseTokenPattern           = regexp.MustCompile(`(?i)\b(2160p|1080p|1080i|720p|480p|4k|hdr|dolby|vision|bluray|brrip|webdl|webrip|remux|x265|x264|h264|h265|hevc|avc|dts|atmos|ac3|aac|ddp(?:[0-9.]+)?|netflix|hdtv|bitstv)\b`)
	numericEpisodePattern         = regexp.MustCompile(`^0*(\d{1,3})$`)
	leadingNumericTokenPattern    = regexp.MustCompile(`^0*([1-9]\d{0,2})(?:$|[\s._-]+.*)`)
	separatorRunsPattern          = regexp.MustCompile(`[._\-\[\](){}]+`)
	whitespaceRunsPattern         = regexp.MustCompile(`\s+`)
)

func ParseVideoFilename(filePath string) VideoSignal {
	stem := filenameStem(filePath)
	normalizedStem := normalizeReleaseText(stem)
	signal := VideoSignal{
		Year:          parseYearPointer(normalizedStem),
		ReleaseTokens: parseReleaseTokens(normalizedStem),
	}

	if seriesPrefix, season, episodeNumbers, ok := parseMultiEpisodeRange(normalizedStem); ok {
		signal.TitleCandidates = titleCandidates(seriesPrefix)
		signal.Season = season
		signal.EpisodeNumbers = append([]int(nil), episodeNumbers...)
		if len(episodeNumbers) > 0 {
			first := episodeNumbers[0]
			last := episodeNumbers[len(episodeNumbers)-1]
			signal.Episode = &first
			if len(episodeNumbers) > 1 {
				signal.EpisodeEnd = &last
			}
		}
		return signal
	}

	if match := sxxexxPattern.FindStringSubmatchIndex(normalizedStem); len(match) > 0 {
		season := parseIntMatch(normalizedStem, match[2], match[3])
		episode := parseIntMatch(normalizedStem, match[4], match[5])
		signal.Season = &season
		signal.Episode = &episode
		signal.EpisodeNumbers = []int{episode}
		signal.TitleCandidates = titleCandidates(normalizedStem[:match[0]])
		return signal
	}

	if match := separatedSeasonEpisodePattern.FindStringSubmatchIndex(normalizedStem); len(match) > 0 {
		season := parseIntMatch(normalizedStem, match[2], match[3])
		episode := parseIntMatch(normalizedStem, match[4], match[5])
		signal.Season = &season
		signal.Episode = &episode
		signal.EpisodeNumbers = []int{episode}
		signal.TitleCandidates = titleCandidates(normalizedStem[:match[0]])
		return signal
	}

	if match := xEpisodePattern.FindStringSubmatchIndex(normalizedStem); len(match) > 0 {
		season := parseIntMatch(normalizedStem, match[2], match[3])
		episode := parseIntMatch(normalizedStem, match[4], match[5])
		signal.Season = &season
		signal.Episode = &episode
		signal.EpisodeNumbers = []int{episode}
		signal.TitleCandidates = titleCandidates(normalizedStem[:match[0]])
		return signal
	}

	if episode, matchEnd, ok := parseEpisodeOnly(normalizedStem); ok {
		if episode == 0 {
			signal.IsSpecial = true
			signal.TitleCandidates = titleCandidates(normalizedStem[:matchEnd])
			return signal
		}
		signal.Episode = &episode
		signal.EpisodeNumbers = []int{episode}
		signal.TitleCandidates = titleCandidates(normalizedStem[:matchEnd])
		return signal
	}

	if match := specialEpisodeIndexPattern.FindStringSubmatchIndex(normalizedStem); len(match) >= 6 {
		index := parseIntMatch(normalizedStem, match[4], match[5])
		signal.IsSpecial = true
		signal.SpecialIndex = &index
		signal.TitleCandidates = titleCandidates(normalizedStem[:match[0]])
		return signal
	}

	if match := episodeZeroMarkerPattern.FindStringIndex(normalizedStem); len(match) > 0 {
		signal.IsSpecial = true
		signal.TitleCandidates = titleCandidates(normalizedStem[:match[0]])
		return signal
	}

	if match := specialEpisodeMarkerPattern.FindStringIndex(normalizedStem); len(match) > 0 {
		signal.IsSpecial = true
		signal.TitleCandidates = titleCandidates(normalizedStem[:match[0]])
		return signal
	}

	signal.TitleCandidates = movieTitleCandidates(normalizedStem, signal.Year)
	return signal
}

func AnalyzeVideoPath(filePath string) VideoSignal {
	signal := ParseVideoFilename(filePath)
	stem := filenameStem(filePath)
	normalized := titleclean.Normalize(titleclean.NormalizeInput{RawTitle: stem})
	signal.CleanupEvidence = cleanupEvidenceFromRemovedTokens(normalized.RemovedTokens)
	removedValues := cleanupEvidenceValueSet(signal.CleanupEvidence)
	signal.TitleTokens = BuildTitleTokens(stem, removedValues)
	signal.Quality = QualitySignal(stem)
	signal.SourceTags = SourceTagSignals(stem)
	signal.Codec = CodecSignal(stem)
	signal.Audio = AudioSignal(stem)
	signal.Subtitle = SubtitleSignal(stem)
	signal.HDR = HDRSignal(stem)
	signal.Edition = EditionSignal(stem)
	signal.ReleaseGroup = ReleaseGroupSignal(stem)
	signal.Website = firstNonEmptyCleanupEvidence(signal.CleanupEvidence, "website", WebsiteSignal(stem))
	signal.GenericNoise = GenericMediaNameSignal(stem)
	signal.Role = VideoFileRoleSignal(filePath)
	signal.IsSample = signal.Role == "sample"
	signal.IsTrailer = signal.Role == "trailer"
	signal.IsExtra = signal.Role != ""
	signal.IsMain = !signal.IsExtra
	if leading := LeadingNumericToken(stem); leading != nil {
		signal.LeadingNumber = leading
		if signal.Episode == nil && WeakEpisodeNumberAllowed(stem) {
			signal.Episode = leading
			signal.EpisodeNumbers = EpisodeNumbersFromPointer(leading)
			signal.EpisodeSource = "leading_numeric"
		}
	}
	if signal.EpisodeSource == "" && (signal.Episode != nil || len(signal.EpisodeNumbers) > 0) && HasExplicitEpisodeMarker(stem) {
		signal.EpisodeSource = "explicit"
	}
	return signal
}

func cleanupEvidenceFromRemovedTokens(tokens []titleclean.RemovedToken) []CleanupEvidence {
	items := make([]CleanupEvidence, 0, len(tokens))
	for _, token := range tokens {
		value := strings.TrimSpace(token.Value)
		reason := strings.TrimSpace(token.Reason)
		if value == "" || reason == "" {
			continue
		}
		items = append(items, CleanupEvidence{Token: value, Reason: reason})
	}
	return items
}

func cleanupEvidenceValueSet(items []CleanupEvidence) map[string]struct{} {
	values := make(map[string]struct{}, len(items))
	for _, item := range items {
		trimmed := strings.ToLower(strings.TrimSpace(item.Token))
		if trimmed == "" {
			continue
		}
		values[trimmed] = struct{}{}
	}
	return values
}

func firstNonEmptyCleanupEvidence(items []CleanupEvidence, reason string, fallback string) string {
	for _, item := range items {
		if strings.TrimSpace(item.Reason) == reason && strings.TrimSpace(item.Token) != "" {
			return strings.TrimSpace(item.Token)
		}
	}
	return strings.TrimSpace(fallback)
}

func filenameStem(filePath string) string {
	base := path.Base(normalizePath(filePath))
	extension := path.Ext(base)
	return strings.TrimSuffix(base, extension)
}

func normalizeReleaseText(input string) string {
	replacer := strings.NewReplacer(
		"WEB-DL", "webdl",
		"Web-DL", "webdl",
		"web-dl", "webdl",
		"WEB.DL", "webdl",
		"web.dl", "webdl",
		"BluRay", "bluray",
		"BLURAY", "bluray",
	)
	return replacer.Replace(input)
}

func parseYearPointer(input string) *int {
	matches := yearPattern.FindAllStringIndex(input, -1)
	if len(matches) == 0 || (len(matches) == 1 && matches[0][0] == 0) {
		return nil
	}
	lastMatch := matches[len(matches)-1]
	value, err := strconv.Atoi(input[lastMatch[0]:lastMatch[1]])
	if err != nil {
		return nil
	}
	return &value
}

func parseReleaseTokens(input string) []string {
	matches := releaseTokenPattern.FindAllString(input, -1)
	if len(matches) == 0 {
		return nil
	}
	tokens := make([]string, 0, len(matches))
	seen := map[string]struct{}{}
	for _, match := range matches {
		token := strings.ToLower(strings.ReplaceAll(match, "-", ""))
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}
	return tokens
}

func parseLeadingNumericToken(input string) *int {
	match := leadingNumericTokenPattern.FindStringSubmatch(strings.TrimSpace(input))
	if len(match) < 2 {
		return nil
	}
	return parseOrdinalToken(match[1])
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

func parseEpisodeOnly(input string) (int, int, bool) {
	for _, pattern := range []*regexp.Regexp{chineseEpisodePattern, episodeWordPattern, episodeMarkerPattern, numericEpisodePattern} {
		matches := pattern.FindStringSubmatchIndex(input)
		if len(matches) < 4 {
			continue
		}
		if !episodeSuffixLooksValid(input[matches[1]:]) {
			continue
		}
		value, err := strconv.Atoi(input[matches[2]:matches[3]])
		if err != nil {
			continue
		}
		return value, matches[0], true
	}
	return 0, 0, false
}

func episodeSuffixLooksValid(input string) bool {
	suffix := strings.TrimSpace(input)
	if suffix == "" {
		return true
	}
	tokens := strings.FieldsFunc(suffix, func(r rune) bool {
		return r == '.' || r == '_' || r == '-' || r == '[' || r == ']' || r == '(' || r == ')' || r == '{' || r == '}' || unicode.IsSpace(r)
	})
	if len(tokens) == 0 {
		return true
	}
	first := tokens[0]
	if yearPattern.MatchString(first) || releaseTokenPattern.MatchString(first) {
		return true
	}
	if episodeRangeTokenPattern.MatchString(first) || specialEpisodeTokenPattern.MatchString(first) {
		return true
	}
	return false
}

func parseIntMatch(input string, start int, end int) int {
	value, _ := strconv.Atoi(input[start:end])
	return value
}

func movieTitleCandidates(input string, year *int) []string {
	cutoff := len(input)
	if year != nil {
		if matches := yearPattern.FindAllStringIndex(input, -1); len(matches) > 0 {
			lastMatch := matches[len(matches)-1]
			cutoff = lastMatch[0]
		}
	} else if match := releaseTokenPattern.FindStringIndex(input); len(match) > 0 {
		cutoff = match[0]
	}
	return titleCandidates(input[:cutoff])
}

func titleCandidates(input string) []string {
	title := normalizeTitle(input)
	if title == "" {
		return nil
	}
	return []string{title}
}

func normalizeTitle(input string) string {
	replaced := separatorRunsPattern.ReplaceAllString(input, " ")
	trimmed := strings.TrimSpace(whitespaceRunsPattern.ReplaceAllString(replaced, " "))
	if trimmed == "" {
		return ""
	}
	words := strings.Fields(trimmed)
	for idx, word := range words {
		words[idx] = normalizeTitleWord(word)
	}
	return strings.Join(words, " ")
}

func normalizeTitleWord(word string) string {
	runes := []rune(strings.ToLower(word))
	for idx, value := range runes {
		if unicode.IsLetter(value) {
			runes[idx] = unicode.ToUpper(value)
			break
		}
	}
	return string(runes)
}
