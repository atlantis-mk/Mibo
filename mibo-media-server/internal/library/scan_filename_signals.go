package library

import (
	"path"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/scanrecognition"
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

	filenameSignalReasonQualityNoise              = "quality_noise"
	filenameSignalReasonSourceNoise               = "source_noise"
	filenameSignalReasonCodecNoise                = "codec_noise"
	filenameSignalReasonAudioNoise                = "audio_noise"
	filenameSignalReasonSubtitleNoise             = "subtitle_noise"
	filenameSignalReasonHDRNoise                  = "hdr_noise"
	filenameSignalReasonEditionHint               = "edition_hint"
	filenameSignalReasonReleaseGroupHint          = "release_group_hint"
	filenameSignalReasonYearHint                  = "year_hint"
	filenameSignalReasonEpisodeMarker             = "episode_marker"
	filenameSignalReasonRoleHint                  = "role_hint"
	filenameSignalReasonRemovedFromTitle          = "removed_from_title"
	filenameSignalReasonSuppressWeakEpisodeNumber = "suppress_weak_episode_number"
	filenameSignalReasonDirectoryTitleHint        = "directory_title_hint"
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
	LeadingNumber  *int
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
	RawPathData filenameRawPathData
	// VideoSignal is the authoritative scanrecognition result for this path.
	VideoSignal scanrecognition.VideoSignal
	TitleTokens []filenameTitleToken
	// Identity mirrors selected VideoSignal fields for cache/persistence compatibility.
	Identity filenameIdentitySignals
	// ReleaseHints mirrors selected VideoSignal release metadata for legacy readers.
	ReleaseHints filenameReleaseHints
	// RoleHints mirrors selected VideoSignal role flags for legacy readers.
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
	videoSignal := scanrecognition.AnalyzeVideoPath(cleanPath)
	folderSignal := scanrecognition.ParseFolderName(path.Base(path.Dir(cleanPath)))
	model := filenameSignalModel{
		RawPathData: filenameRawPathData{Path: cleanPath, Directory: path.Dir(cleanPath), Basename: fileName, Extension: ext, Segments: strings.Split(strings.Trim(cleanPath, "/"), "/")},
		VideoSignal: videoSignal,
		Identity:    filenameIdentitySignals{TitleCandidate: firstScanTitle(videoSignal.TitleCandidates), Year: firstNonNilInt(videoSignal.Year, folderSignal.Year)},
		ReleaseHints: filenameReleaseHints{
			Quality:      videoSignal.Quality,
			SourceTags:   append([]string(nil), videoSignal.SourceTags...),
			Codec:        videoSignal.Codec,
			Audio:        videoSignal.Audio,
			Subtitle:     videoSignal.Subtitle,
			HDR:          videoSignal.HDR,
			Edition:      videoSignal.Edition,
			ReleaseGroup: videoSignal.ReleaseGroup,
			Website:      videoSignal.Website,
			GenericNoise: videoSignal.GenericNoise,
		},
		RoleHints:       filenameRoleHints{Role: videoSignal.Role},
		CleanupEvidence: filenameCleanupEvidenceFromSignal(videoSignal.CleanupEvidence),
	}
	model.RoleHints.IsSample = videoSignal.IsSample
	model.RoleHints.IsTrailer = videoSignal.IsTrailer
	model.RoleHints.IsExtra = videoSignal.IsExtra
	model.RoleHints.IsMain = videoSignal.IsMain
	model.TitleTokens = filenameTitleTokensFromSignal(videoSignal.TitleTokens)
	model.Identity.SeasonNumber = firstNonNilInt(videoSignal.Season, folderSignal.Season)
	model.Identity.EpisodeNumber = videoSignal.Episode
	model.Identity.EpisodeEnd = videoSignal.EpisodeEnd
	model.Identity.EpisodeNumbers = append([]int(nil), videoSignal.EpisodeNumbers...)
	model.Identity.LeadingNumber = videoSignal.LeadingNumber
	model.Identity.EpisodeSource = videoSignal.EpisodeSource
	model.PathHints.SeasonNumber = scanrecognition.SeasonFromPath("", cleanPath)
	model.PathHints.SeriesTitle = scanrecognition.SeriesTitleFromPath("", cleanPath)
	model.Evidence = filenameEvidenceSummariesFromModel(model, rawTitle)
	return model
}

func syncFilenameSignalModel(storagePath string, model *filenameSignalModel) {
	if model == nil {
		return
	}
	pathValue := strings.TrimSpace(storagePath)
	if pathValue == "" {
		pathValue = strings.TrimSpace(model.RawPathData.Path)
	}
	if pathValue != "" {
		model.VideoSignal = scanrecognition.AnalyzeVideoPath(pathValue)
	}
	model.RoleHints.Role = model.VideoSignal.Role
	model.RoleHints.IsSample = model.VideoSignal.IsSample
	model.RoleHints.IsTrailer = model.VideoSignal.IsTrailer
	model.RoleHints.IsExtra = model.VideoSignal.IsExtra
	model.RoleHints.IsMain = model.VideoSignal.IsMain
	model.ReleaseHints.Quality = model.VideoSignal.Quality
	model.ReleaseHints.SourceTags = append([]string(nil), model.VideoSignal.SourceTags...)
	model.ReleaseHints.Codec = model.VideoSignal.Codec
	model.ReleaseHints.Audio = model.VideoSignal.Audio
	model.ReleaseHints.Subtitle = model.VideoSignal.Subtitle
	model.ReleaseHints.HDR = model.VideoSignal.HDR
	model.ReleaseHints.Edition = model.VideoSignal.Edition
	model.ReleaseHints.ReleaseGroup = model.VideoSignal.ReleaseGroup
	model.ReleaseHints.Website = model.VideoSignal.Website
	model.ReleaseHints.GenericNoise = model.VideoSignal.GenericNoise
	if len(model.CleanupEvidence) == 0 {
		model.CleanupEvidence = filenameCleanupEvidenceFromSignal(model.VideoSignal.CleanupEvidence)
	}
	if len(model.TitleTokens) == 0 {
		model.TitleTokens = filenameTitleTokensFromSignal(model.VideoSignal.TitleTokens)
	}
	if strings.TrimSpace(model.Identity.TitleCandidate) == "" {
		model.Identity.TitleCandidate = firstScanTitle(model.VideoSignal.TitleCandidates)
	}
	model.Identity.Year = firstNonNilInt(model.Identity.Year, model.VideoSignal.Year)
	model.Identity.SeasonNumber = firstNonNilInt(model.Identity.SeasonNumber, model.VideoSignal.Season)
	model.Identity.EpisodeNumber = firstNonNilInt(model.Identity.EpisodeNumber, model.VideoSignal.Episode)
	if model.Identity.EpisodeEnd == nil {
		model.Identity.EpisodeEnd = model.VideoSignal.EpisodeEnd
	}
	if len(model.Identity.EpisodeNumbers) == 0 {
		model.Identity.EpisodeNumbers = append([]int(nil), model.VideoSignal.EpisodeNumbers...)
	}
	if model.Identity.LeadingNumber == nil {
		model.Identity.LeadingNumber = model.VideoSignal.LeadingNumber
	}
	if strings.TrimSpace(model.Identity.EpisodeSource) == "" {
		model.Identity.EpisodeSource = model.VideoSignal.EpisodeSource
	}
}

func filenameCleanupEvidenceFromSignal(items []scanrecognition.CleanupEvidence) []filenameCleanupEvidence {
	converted := make([]filenameCleanupEvidence, 0, len(items))
	for _, item := range items {
		converted = append(converted, filenameCleanupEvidence{Token: item.Token, Reason: item.Reason})
	}
	return converted
}

func filenameTitleTokensFromSignal(items []scanrecognition.TitleToken) []filenameTitleToken {
	converted := make([]filenameTitleToken, 0, len(items))
	for _, item := range items {
		converted = append(converted, filenameTitleToken{Value: item.Value, Kept: item.Kept})
	}
	return converted
}

func filenameSignalTitleCandidate(model filenameSignalModel) string {
	return strings.TrimSpace(firstNonEmptyString(firstScanTitle(model.VideoSignal.TitleCandidates), model.Identity.TitleCandidate))
}

func movieFolderTitleCandidate(model filenameSignalModel) string {
	return movieFolderTitleCandidateFromParts(model.VideoSignal, model.RawPathData, model.Identity.TitleCandidate, model.Identity.Year)
}

func movieFolderTitleCandidateFromParts(videoSignal scanrecognition.VideoSignal, rawPathData filenameRawPathData, fallbackTitle string, fallbackYear *int) string {
	folderName := strings.TrimSpace(path.Base(strings.TrimSpace(rawPathData.Directory)))
	if folderName == "" || folderName == "." || folderName == "/" {
		return ""
	}
	folderSignal := scanrecognition.ParseFolderName(folderName)
	if folderSignal.Season != nil {
		return ""
	}
	folderTitle := bestFolderTitleCandidate(folderSignal.TitleCandidates, videoSignal.TitleCandidates, fallbackTitle)
	if folderTitle == "" {
		return ""
	}
	fileTitle := strings.TrimSpace(firstNonEmptyString(firstScanTitle(videoSignal.TitleCandidates), fallbackTitle))
	folderNormalized := normalizeVersionCompareTitle(folderTitle)
	fileNormalized := normalizeVersionCompareTitle(fileTitle)
	if folderNormalized != "" && fileNormalized != "" && folderNormalized == fileNormalized {
		return folderTitle
	}
	folderYear := folderSignal.Year
	fileYear := firstNonNilInt(videoSignal.Year, fallbackYear)
	if folderYear != nil && fileYear != nil && *folderYear == *fileYear {
		return folderTitle
	}
	rawTitle := strings.TrimSpace(strings.TrimSuffix(rawPathData.Basename, rawPathData.Extension))
	if strings.TrimSpace(videoSignal.GenericNoise) != "" || strings.TrimSpace(scanrecognition.GenericMediaNameSignal(rawTitle)) != "" {
		return folderTitle
	}
	return ""
}

func bestFolderTitleCandidate(folderCandidates []string, fileCandidates []string, fallbackTitle string) string {
	fileNormalized := normalizeVersionCompareTitle(strings.TrimSpace(firstNonEmptyString(firstScanTitle(fileCandidates), fallbackTitle)))
	if fileNormalized != "" {
		for _, candidate := range folderCandidates {
			trimmed := strings.TrimSpace(candidate)
			if trimmed == "" {
				continue
			}
			if normalizeVersionCompareTitle(trimmed) == fileNormalized {
				return trimmed
			}
		}
	}
	return strings.TrimSpace(firstScanTitle(folderCandidates))
}

func preferredMovieTitleCandidate(model filenameSignalModel, preferFolder bool) string {
	if preferFolder {
		if folderTitle := movieFolderTitleCandidate(model); folderTitle != "" {
			return folderTitle
		}
	}
	if title := filenameSignalTitleCandidate(model); title != "" {
		return title
	}
	if !preferFolder {
		return movieFolderTitleCandidate(model)
	}
	return ""
}

func preferredMovieYearCandidate(model filenameSignalModel, preferFolder bool) *int {
	if preferFolder {
		folderSignal := scanrecognition.ParseFolderName(path.Base(strings.TrimSpace(model.RawPathData.Directory)))
		if folderTitle := movieFolderTitleCandidate(model); folderTitle != "" && folderSignal.Year != nil {
			return firstNonNilInt(folderSignal.Year, model.VideoSignal.Year, model.Identity.Year)
		}
	}
	return firstNonNilInt(model.VideoSignal.Year, model.Identity.Year)
}

func filenameSignalYear(model filenameSignalModel) *int {
	return firstNonNilInt(model.VideoSignal.Year, model.Identity.Year)
}

func filenameSignalSeasonNumber(model filenameSignalModel) *int {
	return firstNonNilInt(model.VideoSignal.Season, model.Identity.SeasonNumber)
}

func filenameSignalEpisodeNumber(model filenameSignalModel) *int {
	return firstNonNilInt(model.VideoSignal.Episode, model.Identity.EpisodeNumber)
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
	if !scanrecognition.WeakEpisodeNumberAllowed(rawTitle) {
		items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindAntiMisclassification, Source: "filename", Value: rawTitle, Reason: filenameSignalReasonSuppressWeakEpisodeNumber})
	}
	for _, token := range model.TitleTokens {
		if !token.Kept {
			items = append(items, filenameEvidenceSummary{Kind: filenameSignalKindAntiMisclassification, Source: "filename", Value: token.Value, Reason: filenameSignalReasonSuppressWeakEpisodeNumber})
		}
	}
	return items
}
