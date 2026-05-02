package library

import (
	"path"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/titleclean"
)

const (
	filenameSignalKindQuality               = "quality"
	filenameSignalKindSource                = "source"
	filenameSignalKindCodec                 = "codec"
	filenameSignalKindAudio                 = "audio"
	filenameSignalKindSubtitle              = "subtitle"
	filenameSignalKindHDR                   = "hdr"
	filenameSignalKindEdition               = "edition"
	filenameSignalKindReleaseGroup          = "release_group"
	filenameSignalKindYear                  = "year"
	filenameSignalKindEpisodeMarker         = "episode_marker"
	filenameSignalKindRole                  = "role"
	filenameSignalKindTitleCleanup          = "title_cleanup"
	filenameSignalKindAntiMisclassification = "anti_misclassification"
	filenameSignalKindTitle                 = "title"
	filenameSignalKindPath                  = "path"

	filenameSignalReasonQualityNoise               = "quality_noise"
	filenameSignalReasonSourceNoise                = "source_noise"
	filenameSignalReasonCodecNoise                 = "codec_noise"
	filenameSignalReasonAudioNoise                 = "audio_noise"
	filenameSignalReasonSubtitleNoise              = "subtitle_noise"
	filenameSignalReasonHDRNoise                   = "hdr_noise"
	filenameSignalReasonEditionHint                = "edition_hint"
	filenameSignalReasonReleaseGroupHint           = "release_group_hint"
	filenameSignalReasonYearHint                   = "year_hint"
	filenameSignalReasonEpisodeMarker              = "episode_marker"
	filenameSignalReasonRoleHint                   = "role_hint"
	filenameSignalReasonRemovedFromTitle           = "removed_from_title"
	filenameSignalReasonSuppressWeakEpisodeNumber  = "suppress_weak_episode_number"
	filenameSignalReasonDirectoryTitleHint         = "directory_title_hint"
	filenameSignalReasonWebsiteNoise              = "website_noise"
	filenameSignalReasonGenericNameHint           = "generic_name_hint"
)

type filenameRawPathData struct {
	Path      string
	Directory string
	Basename  string
	Extension string
	Segments  []string
}

type filenameTitleToken struct {
	Value string
	Kept  bool
}

type filenameIdentitySignals struct {
	TitleCandidate string
	Year           *int
	SeasonNumber   *int
	EpisodeNumber  *int
	EpisodeEnd     *int
	EpisodeNumbers []int
	EpisodeSource  string
}

type filenameReleaseHints struct {
	Quality      string
	SourceTags   []string
	Codec        string
	Audio        string
	Subtitle     string
	HDR          string
	Edition      string
	ReleaseGroup string
	Website      string
	GenericNoise string
}

type filenameRoleHints struct {
	Role      string
	IsMain    bool
	IsSample  bool
	IsTrailer bool
	IsExtra   bool
}

type filenameCleanupEvidence struct {
	Token  string
	Reason string
}

type filenamePathHints struct {
	SeriesTitle  string
	SeasonNumber *int
	Role         string
}

type filenameEvidenceSummary struct {
	Kind   string
	Source string
	Value  string
	Reason string
}

type filenameSignalModel struct {
	RawPathData     filenameRawPathData
	TitleTokens     []filenameTitleToken
	Identity        filenameIdentitySignals
	ReleaseHints    filenameReleaseHints
	RoleHints       filenameRoleHints
	CleanupEvidence []filenameCleanupEvidence
	PathHints       filenamePathHints
	Evidence        []filenameEvidenceSummary
}

func extractFilenameSignalModel(itemPath string) filenameSignalModel {
	cleanPath := path.Clean(strings.TrimSpace(itemPath))
	fileName := path.Base(cleanPath)
	ext := path.Ext(fileName)
	rawTitle := strings.TrimSuffix(fileName, ext)
	segments := strings.Split(strings.Trim(cleanPath, "/"), "/")
	normalized := titleclean.Normalize(titleclean.NormalizeInput{RawTitle: rawTitle})
	model := filenameSignalModel{
		RawPathData: filenameRawPathData{Path: cleanPath, Directory: path.Dir(cleanPath), Basename: fileName, Extension: ext, Segments: segments},
		Identity: filenameIdentitySignals{TitleCandidate: normalized.Title, Year: normalized.Year},
		ReleaseHints: filenameReleaseHints{
			Quality:      qualitySignal(rawTitle),
			SourceTags:   sourceTagSignals(rawTitle),
			Codec:        firstSignalByReason(normalized.RemovedTokens, "video_codec"),
			Audio:        firstSignalByReason(normalized.RemovedTokens, "audio"),
			Subtitle:     firstSignalByReason(normalized.RemovedTokens, "subtitle"),
			HDR:          firstSignalByReason(normalized.RemovedTokens, "hdr"),
			Edition:      editionSignal(rawTitle),
			ReleaseGroup: releaseGroupSignal(rawTitle),
			Website:      firstSignalByReason(normalized.RemovedTokens, "website"),
			GenericNoise: genericFilenameNoiseSignal(rawTitle),
		},
		RoleHints:       filenameRoleHints{Role: videoFileRoleSignal(cleanPath)},
		CleanupEvidence: cleanupEvidenceFromRemovedTokens(normalized.RemovedTokens),
	}
	model.RoleHints.IsSample = model.RoleHints.Role == "sample"
	model.RoleHints.IsTrailer = model.RoleHints.Role == "trailer"
	model.RoleHints.IsExtra = model.RoleHints.Role != ""
	model.RoleHints.IsMain = !model.RoleHints.IsExtra
	model.TitleTokens = filenameTitleTokens(rawTitle, model.CleanupEvidence)
	if seriesPrefix, season, episodeNumbers, ok := parseMultiEpisodeRange(rawTitle); ok {
		model.Identity.TitleCandidate = cleanTitle(seriesPrefix)
		model.Identity.SeasonNumber = season
		model.Identity.EpisodeNumbers = append([]int(nil), episodeNumbers...)
		if len(episodeNumbers) > 0 {
			first := episodeNumbers[0]
			last := episodeNumbers[len(episodeNumbers)-1]
			model.Identity.EpisodeNumber = &first
			if len(episodeNumbers) > 1 {
				model.Identity.EpisodeEnd = &last
			}
		}
		model.Identity.EpisodeSource = "explicit"
	} else if groups := episodePattern.FindStringSubmatch(rawTitle); len(groups) > 0 {
		season, episode := parseEpisodeNumbers(groups[2], groups[3], groups[4], groups[5])
		model.Identity.TitleCandidate = cleanTitle(groups[1])
		model.Identity.SeasonNumber = season
		model.Identity.EpisodeNumber = episode
		model.Identity.EpisodeNumbers = episodeNumbersFromPointer(episode)
		model.Identity.EpisodeSource = "explicit"
	} else if episode := parseEmbeddedEpisodeNumber(rawTitle); episode != nil {
		model.Identity.EpisodeNumber = episode
		model.Identity.EpisodeNumbers = episodeNumbersFromPointer(episode)
		model.Identity.EpisodeSource = "explicit"
	} else if match := chineseEpisodePattern.FindStringSubmatch(normalizeEpisodeIdentifier(rawTitle)); len(match) >= 2 {
		if episode := parseOrdinalToken(match[1]); episode != nil {
			model.Identity.EpisodeNumber = episode
			model.Identity.EpisodeNumbers = episodeNumbersFromPointer(episode)
			model.Identity.EpisodeSource = "explicit"
		}
	}
	for idx := len(segments) - 2; idx >= 0; idx-- {
		if season := parseSeasonDirectoryNumber(segments[idx]); season != nil {
			model.PathHints.SeasonNumber = season
			break
		}
	}
	model.Evidence = filenameEvidenceSummariesFromModel(model, rawTitle)
	return model
}

func cleanupEvidenceFromRemovedTokens(tokens []titleclean.RemovedToken) []filenameCleanupEvidence {
	items := make([]filenameCleanupEvidence, 0, len(tokens))
	for _, token := range tokens {
		value := strings.TrimSpace(token.Value)
		reason := strings.TrimSpace(token.Reason)
		if value == "" || reason == "" {
			continue
		}
		items = append(items, filenameCleanupEvidence{Token: value, Reason: reason})
	}
	return items
}

func firstSignalByReason(tokens []titleclean.RemovedToken, reason string) string {
	for _, token := range tokens {
		if strings.TrimSpace(token.Reason) == reason && strings.TrimSpace(token.Value) != "" {
			return strings.TrimSpace(token.Value)
		}
	}
	return ""
}

func genericFilenameNoiseSignal(rawTitle string) string {
	if isGenericMediaName(rawTitle) {
		return strings.TrimSpace(rawTitle)
	}
	return ""
}

func filenameTitleTokens(rawTitle string, removed []filenameCleanupEvidence) []filenameTitleToken {
	removedValues := make(map[string]struct{}, len(removed))
	for _, evidence := range removed {
		removedValues[strings.ToLower(strings.TrimSpace(evidence.Token))] = struct{}{}
	}
	tokens := strings.Fields(normalizeEpisodeIdentifier(rawTitle))
	items := make([]filenameTitleToken, 0, len(tokens))
	for _, token := range tokens {
		trimmed := strings.TrimSpace(token)
		if trimmed == "" {
			continue
		}
		_, removed := removedValues[strings.ToLower(trimmed)]
		items = append(items, filenameTitleToken{Value: trimmed, Kept: !removed})
	}
	return items
}

func (summary filenameEvidenceSummary) scanDecisionEvidence() scanDecisionEvidence {
	return scanDecisionEvidence{Kind: strings.TrimSpace(summary.Kind), Source: strings.TrimSpace(summary.Source), Value: strings.TrimSpace(summary.Value)}
}

func filenameEvidenceSummariesToScanDecisionEvidence(summaries []filenameEvidenceSummary) []scanDecisionEvidence {
	items := make([]scanDecisionEvidence, 0, len(summaries))
	for _, summary := range summaries {
		item := summary.scanDecisionEvidence()
		if item.Kind == "" || item.Source == "" || item.Value == "" {
			continue
		}
		items = append(items, item)
	}
	return items
}

func filenameEvidenceSummariesFromModel(model filenameSignalModel, rawTitle string) []filenameEvidenceSummary {
	items := make([]filenameEvidenceSummary, 0, 10)
	if strings.TrimSpace(model.Identity.TitleCandidate) != "" {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindTitle, Source: "filename", Value: model.Identity.TitleCandidate, Reason: filenameSignalReasonRemovedFromTitle})
	}
	if model.Identity.Year != nil {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindYear, Source: "filename", Value: strconv.Itoa(*model.Identity.Year), Reason: filenameSignalReasonYearHint})
	}
	if strings.TrimSpace(model.ReleaseHints.Quality) != "" {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindQuality, Source: "filename", Value: model.ReleaseHints.Quality, Reason: filenameSignalReasonQualityNoise})
	}
	if strings.TrimSpace(model.ReleaseHints.Codec) != "" {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindCodec, Source: "filename", Value: model.ReleaseHints.Codec, Reason: filenameSignalReasonCodecNoise})
	}
	if strings.TrimSpace(model.ReleaseHints.Audio) != "" {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindAudio, Source: "filename", Value: model.ReleaseHints.Audio, Reason: filenameSignalReasonAudioNoise})
	}
	if strings.TrimSpace(model.ReleaseHints.Subtitle) != "" {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindSubtitle, Source: "filename", Value: model.ReleaseHints.Subtitle, Reason: filenameSignalReasonSubtitleNoise})
	}
	if strings.TrimSpace(model.ReleaseHints.HDR) != "" {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindHDR, Source: "filename", Value: model.ReleaseHints.HDR, Reason: filenameSignalReasonHDRNoise})
	}
	if strings.TrimSpace(model.ReleaseHints.Edition) != "" {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindEdition, Source: "filename", Value: model.ReleaseHints.Edition, Reason: filenameSignalReasonEditionHint})
	}
	if strings.TrimSpace(model.ReleaseHints.ReleaseGroup) != "" {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindReleaseGroup, Source: "filename", Value: model.ReleaseHints.ReleaseGroup, Reason: filenameSignalReasonReleaseGroupHint})
	}
	if strings.TrimSpace(model.ReleaseHints.Website) != "" {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindTitleCleanup, Source: "filename", Value: model.ReleaseHints.Website, Reason: filenameSignalReasonWebsiteNoise})
	}
	if strings.TrimSpace(model.ReleaseHints.GenericNoise) != "" {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindTitleCleanup, Source: "filename", Value: model.ReleaseHints.GenericNoise, Reason: filenameSignalReasonGenericNameHint})
	}
	if strings.TrimSpace(model.RoleHints.Role) != "" {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindRole, Source: "path", Value: model.RoleHints.Role, Reason: filenameSignalReasonRoleHint})
	}
	if model.Identity.EpisodeNumber != nil || len(model.Identity.EpisodeNumbers) > 0 {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindEpisodeMarker, Source: strings.TrimSpace(model.Identity.EpisodeSource), Value: rawTitle, Reason: filenameSignalReasonEpisodeMarker})
	}
	if !weakEpisodeNumberAllowed(rawTitle) {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindAntiMisclassification, Source: "filename", Value: rawTitle, Reason: filenameSignalReasonSuppressWeakEpisodeNumber})
	}
	return items
}
