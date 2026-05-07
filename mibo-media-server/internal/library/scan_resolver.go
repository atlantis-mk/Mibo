package library

import (
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/storage"
)

var (
	qualitySignalPattern = regexp.MustCompile(`(?i)(2160p|1080p|720p|480p|4k|uhd|hdr10\+?|dv|dolby[\s._-]?vision|hevc|x265|h265|avc|x264|h264|web[\s._-]?dl|webrip|bluray|remux)`)
	editionSignalPattern = regexp.MustCompile(`(?i)(director'?s[\s._-]?cut|extended|unrated|theatrical|imax|criterion|proper|repack)`)
	audioChannelPattern  = regexp.MustCompile(`(?i)\b(?:(?:ddp?|aac|dts|truehd|atmos|eac3|ac3)\s*)?[57](?:\s|\.)1\b`)
)

type filenameSignals struct {
	Model            filenameSignalModel
	RawTitle         string
	TitleCandidate   string
	YearCandidate    *int
	SeasonNumber     *int
	EpisodeNumber    *int
	EpisodeNumberEnd *int
	EpisodeNumbers   []int
	EpisodeSource    string
	QualityLabel     string
	Edition          string
	ReleaseGroup     string
	SourceTags       []string
	ExtraType        string
	IsSample         bool
	IsTrailer        bool
	IsExtra          bool
	Candidates       []fastClassificationCandidate
}

type fastClassificationCandidate struct {
	Type          string
	Role          string
	TargetKind    string
	TargetKey     string
	SeasonNumber  *int
	EpisodeNumber *int
	EpisodeEnd    *int
	Confidence    float64
	Evidence      []scanDecisionEvidence
	Reason        string
}

func resolveFilenameSignals(libraryType string, libraryRoot string, object storage.Object) filenameSignals {
	return resolveFilenameSignalsWithTokenCache(libraryType, libraryRoot, object, nil)
}

func resolveFilenameSignalsWithTokenCache(libraryType string, libraryRoot string, object storage.Object, tokenCache *filenameTokenProfileCache) filenameSignals {
	model := filenameTokenProfileForPath(tokenCache, object.Path)
	rawTitle := strings.TrimSuffix(model.RawPathData.Basename, model.RawPathData.Extension)
	seasonNumber := firstNonNilInt(model.Identity.SeasonNumber, model.PathHints.SeasonNumber, tvSeasonFromPath(libraryRoot, object.Path))
	episodeNumber := model.Identity.EpisodeNumber
	episodeNumbers := append([]int(nil), model.Identity.EpisodeNumbers...)
	episodeEnd := model.Identity.EpisodeEnd
	episodeSource := model.Identity.EpisodeSource
	if episodeNumber == nil {
		if episode := parseEpisodeNumberFromTitle(rawTitle, firstNonEmptyString(model.PathHints.SeriesTitle, tvSeriesTitleFromPath(libraryRoot, object.Path))); episode != nil && weakEpisodeNumberAllowed(rawTitle) {
			episodeNumber = episode
			episodeNumbers = episodeNumbersFromPointer(episode)
			episodeSource = "filename"
		}
	}
	if len(episodeNumbers) > 1 && episodeEnd == nil {
		last := episodeNumbers[len(episodeNumbers)-1]
		episodeEnd = &last
	}
	signals := filenameSignals{
		Model:            model,
		RawTitle:         rawTitle,
		TitleCandidate:   model.Identity.TitleCandidate,
		YearCandidate:    model.Identity.Year,
		SeasonNumber:     seasonNumber,
		EpisodeNumber:    episodeNumber,
		EpisodeNumberEnd: episodeEnd,
		EpisodeNumbers:   episodeNumbers,
		EpisodeSource:    episodeSource,
		QualityLabel:     qualitySignal(rawTitle),
		Edition:          editionSignal(rawTitle),
		ReleaseGroup:     releaseGroupSignal(rawTitle),
		SourceTags:       sourceTagSignals(rawTitle),
		ExtraType:        videoFileRoleSignal(object.Path),
	}
	signals.IsSample = signals.ExtraType == "sample"
	signals.IsTrailer = signals.ExtraType == "trailer"
	signals.IsExtra = signals.ExtraType != ""
	signals.Model.Identity = filenameIdentitySignals{TitleCandidate: signals.TitleCandidate, Year: signals.YearCandidate, SeasonNumber: signals.SeasonNumber, EpisodeNumber: signals.EpisodeNumber, EpisodeEnd: signals.EpisodeNumberEnd, EpisodeNumbers: append([]int(nil), signals.EpisodeNumbers...), EpisodeSource: signals.EpisodeSource}
	signals.Model.ReleaseHints.Quality = signals.QualityLabel
	signals.Model.ReleaseHints.SourceTags = append([]string(nil), signals.SourceTags...)
	signals.Model.ReleaseHints.Edition = signals.Edition
	signals.Model.ReleaseHints.ReleaseGroup = signals.ReleaseGroup
	signals.Model.RoleHints = filenameRoleHints{Role: signals.ExtraType, IsMain: signals.ExtraType == "", IsSample: signals.IsSample, IsTrailer: signals.IsTrailer, IsExtra: signals.IsExtra}
	signals.Model.PathHints = filenamePathHints{SeriesTitle: tvSeriesTitleFromPath(libraryRoot, object.Path), SeasonNumber: tvSeasonFromPath(libraryRoot, object.Path), Role: videoFileRoleSignal(object.Path)}
	signals.Model.Evidence = filenameEvidenceSummaries(signals)
	signals.Candidates = fastCandidatesFromFilenameSignals(signals)
	return signals
}

func filenameEvidenceSummaries(signals filenameSignals) []filenameEvidenceSummary {
	items := make([]filenameEvidenceSummary, 0, 8)
	if strings.TrimSpace(signals.TitleCandidate) != "" {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindTitle, Source: "filename", Value: signals.TitleCandidate, Reason: filenameSignalReasonRemovedFromTitle})
	}
	if signals.YearCandidate != nil {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindYear, Source: "filename", Value: strconv.Itoa(*signals.YearCandidate), Reason: filenameSignalReasonYearHint})
	}
	if strings.TrimSpace(signals.QualityLabel) != "" {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindQuality, Source: "filename", Value: signals.QualityLabel, Reason: filenameSignalReasonQualityNoise})
	}
	if strings.TrimSpace(signals.Edition) != "" {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindEdition, Source: "filename", Value: signals.Edition, Reason: filenameSignalReasonEditionHint})
	}
	if strings.TrimSpace(signals.ReleaseGroup) != "" {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindReleaseGroup, Source: "filename", Value: signals.ReleaseGroup, Reason: filenameSignalReasonReleaseGroupHint})
	}
	if strings.TrimSpace(signals.ExtraType) != "" {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindRole, Source: "path", Value: signals.ExtraType, Reason: filenameSignalReasonRoleHint})
	}
	if signals.EpisodeNumber != nil || len(signals.EpisodeNumbers) > 0 {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindEpisodeMarker, Source: strings.TrimSpace(signals.EpisodeSource), Value: signals.RawTitle, Reason: filenameSignalReasonEpisodeMarker})
	}
	if !weakEpisodeNumberAllowed(signals.RawTitle) {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindAntiMisclassification, Source: "filename", Value: signals.RawTitle, Reason: filenameSignalReasonSuppressWeakEpisodeNumber})
	}
	return items
}

func weakEpisodeNumberAllowed(rawTitle string) bool {
	if hasExplicitEpisodeMarker(rawTitle) {
		return true
	}
	normalized := strings.ToLower(normalizeEpisodeIdentifier(rawTitle))
	if audioChannelPattern.MatchString(normalized) {
		return false
	}
	if qualitySignal(rawTitle) != "" && (strings.Contains(normalized, "h 264") || strings.Contains(normalized, "h 265") || strings.Contains(normalized, "x264") || strings.Contains(normalized, "x265") || strings.Contains(normalized, "hevc")) {
		return false
	}
	if yearPattern.MatchString(rawTitle) && (qualitySignal(rawTitle) != "" || strings.Contains(normalized, "web dl") || strings.Contains(normalized, "web rip") || strings.Contains(normalized, "bluray")) {
		return false
	}
	return true
}

func fastCandidatesFromFilenameSignals(signals filenameSignals) []fastClassificationCandidate {
	candidates := make([]fastClassificationCandidate, 0, 2)
	if signals.IsExtra {
		confidence := 0.9
		role := scanDecisionRoleExtra
		switch signals.ExtraType {
		case "trailer":
			role = scanDecisionRoleTrailer
		case "sample":
			role = scanDecisionRoleSample
		case "preview":
			role = "preview"
		}
		candidates = append(candidates, fastClassificationCandidate{Type: scanDecisionCandidateAttachment, Role: role, Confidence: confidence, Evidence: []scanDecisionEvidence{{Kind: filenameSignalKindRole, Source: "filename", Value: signals.ExtraType}}, Reason: "filename indicates attachment role"})
		return candidates
	}
	if signals.EpisodeNumber != nil || len(signals.EpisodeNumbers) > 0 {
		confidence := 0.72
		if signals.EpisodeSource == "explicit" {
			confidence = 0.9
		}
		candidate := fastClassificationCandidate{Type: scanDecisionCandidateEpisode, Role: scanDecisionRoleMain, SeasonNumber: signals.SeasonNumber, EpisodeNumber: signals.EpisodeNumber, EpisodeEnd: signals.EpisodeNumberEnd, Confidence: confidence, Reason: "filename contains episode evidence"}
		if len(signals.EpisodeNumbers) > 0 {
			first := signals.EpisodeNumbers[0]
			candidate.EpisodeNumber = &first
		}
		candidate.Evidence = append(candidate.Evidence, scanDecisionEvidence{Kind: filenameSignalKindEpisodeMarker, Source: signals.EpisodeSource, Value: signals.RawTitle})
		candidates = append(candidates, candidate)
		if signals.YearCandidate != nil || strings.TrimSpace(signals.QualityLabel) != "" {
			movieConfidence := 0.46
			if signals.YearCandidate != nil {
				movieConfidence = 0.55
			}
			candidates = append(candidates, fastClassificationCandidate{Type: scanDecisionCandidateMovie, Role: scanDecisionRoleMain, Confidence: movieConfidence, Evidence: movieCandidateEvidence(signals), Reason: "filename also contains movie-like release evidence"})
		}
	}
	if signals.EpisodeNumber == nil && len(signals.EpisodeNumbers) == 0 {
		confidence := 0.68
		if signals.YearCandidate != nil {
			confidence = 0.78
		}
		candidates = append(candidates, fastClassificationCandidate{Type: scanDecisionCandidateMovie, Role: scanDecisionRoleMain, Confidence: confidence, Evidence: movieCandidateEvidence(signals), Reason: "filename and path provide movie-like title evidence"})
		if strings.TrimSpace(signals.QualityLabel) != "" || strings.TrimSpace(signals.Edition) != "" || strings.TrimSpace(signals.ReleaseGroup) != "" {
			candidates = append(candidates, fastClassificationCandidate{Type: scanDecisionCandidateMovieVersion, Role: scanDecisionRoleMain, Confidence: 0.62, Evidence: movieCandidateEvidence(signals), Reason: "filename contains movie version release hints"})
		}
	}
	return candidates
}

func scanDecisionAlternativesFromCandidates(candidates []fastClassificationCandidate, selected int) []scanDecisionAlternative {
	alternatives := make([]scanDecisionAlternative, 0, len(candidates))
	for idx, candidate := range candidates {
		if idx == selected {
			continue
		}
		confidence := candidate.Confidence
		alternatives = append(alternatives, scanDecisionAlternative{Type: candidate.Type, Role: candidate.Role, TargetKind: candidate.TargetKind, TargetKey: candidate.TargetKey, Confidence: &confidence, Reason: candidate.Reason})
	}
	return alternatives
}

func movieCandidateEvidence(signals filenameSignals) []scanDecisionEvidence {
	evidence := []scanDecisionEvidence{{Kind: filenameSignalKindTitle, Source: "filename", Value: signals.TitleCandidate}}
	if signals.YearCandidate != nil {
		evidence = append(evidence, scanDecisionEvidence{Kind: filenameSignalKindYear, Source: "filename", Value: strconv.Itoa(*signals.YearCandidate)})
	}
	if strings.TrimSpace(signals.QualityLabel) != "" {
		evidence = append(evidence, scanDecisionEvidence{Kind: filenameSignalKindQuality, Source: "filename", Value: signals.QualityLabel})
	}
	if strings.TrimSpace(signals.Edition) != "" {
		evidence = append(evidence, scanDecisionEvidence{Kind: filenameSignalKindEdition, Source: "filename", Value: signals.Edition})
	}
	return evidence
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

func videoFileRoleSignal(itemPath string) string {
	segments := strings.Split(strings.Trim(path.Clean(itemPath), "/"), "/")
	for _, segment := range segments {
		candidate := strings.TrimSuffix(path.Base(segment), path.Ext(segment))
		if role := extraTypeSignal(candidate); role != "" {
			return role
		}
	}
	return ""
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
