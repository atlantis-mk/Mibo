package library

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/atlan/mibo-media-server/internal/storage"
	"github.com/atlan/mibo-media-server/internal/titleclean"
)

func TestClassifyMediaFileParsesMultiEpisodeRange(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("shows", "/library", storage.Object{Path: "/library/Show One/Season 1/Show.One.S01E01-E02.mkv"})
	if classified.Type != "episode" {
		t.Fatalf("expected episode classification, got %#v", classified)
	}
	if classified.SeriesTitle != "Show One" {
		t.Fatalf("expected series title from folder fallback, got %q", classified.SeriesTitle)
	}
	if classified.SeasonNumber == nil || *classified.SeasonNumber != 1 {
		t.Fatalf("expected season number 1, got %#v", classified.SeasonNumber)
	}

	field := reflect.ValueOf(classified).FieldByName("EpisodeNumbers")
	if !field.IsValid() {
		t.Fatalf("expected classifyMediaFile to expose EpisodeNumbers for multi-episode ranges")
	}
	if !field.CanInterface() {
		t.Fatalf("expected EpisodeNumbers field to be readable")
	}
	episodeNumbers, ok := field.Interface().([]int)
	if !ok {
		t.Fatalf("expected EpisodeNumbers to be []int, got %T", field.Interface())
	}
	if !reflect.DeepEqual(episodeNumbers, []int{1, 2}) {
		t.Fatalf("expected ordered multi-episode slots [1 2], got %#v", episodeNumbers)
	}
	if classified.Title != "Show One S01E01-E02" {
		t.Fatalf("expected multi-episode normalized title, got %q", classified.Title)
	}
	if classified.SourcePath != "/library/Show One/Season 1/Show.One.S01E01-E02.mkv" {
		t.Fatalf("expected source path to be preserved, got %q", classified.SourcePath)
	}
}

func TestClassifyMediaFileCleansMultiEpisodePromoTail(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("shows", "/library", storage.Object{Path: "/library/黑袍纠察队/Season 5/黑袍纠察队.The.Boys.S05E01-02.6v电影 地址发布页 www.6v123.net 收藏不迷路.mkv"})
	if classified.Type != "episode" {
		t.Fatalf("expected episode classification, got %#v", classified)
	}
	if classified.SeriesTitle != "黑袍纠察队" {
		t.Fatalf("expected series title from folder fallback, got %q", classified.SeriesTitle)
	}
	if classified.Title != "黑袍纠察队 S05E01-E02" {
		t.Fatalf("expected clean multi-episode title, got %q", classified.Title)
	}
	if !reflect.DeepEqual(classified.EpisodeNumbers, []int{1, 2}) {
		t.Fatalf("expected ordered episode range [1 2], got %#v", classified.EpisodeNumbers)
	}
}

func TestClassifyMediaFileInfersEpisodeFromSeasonFolder(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("shows", "/library", storage.Object{Path: "/library/Show One/Season 1/01.mkv"})
	if classified.Type != "episode" {
		t.Fatalf("expected episode classification, got %#v", classified)
	}
	if classified.SeriesTitle != "Show One" {
		t.Fatalf("expected series title from show folder, got %#v", classified.SeriesTitle)
	}
	if classified.SeasonNumber == nil || *classified.SeasonNumber != 1 || classified.EpisodeNumber == nil || *classified.EpisodeNumber != 1 {
		t.Fatalf("expected S01E01 from path fallback, got season=%v episode=%v", classified.SeasonNumber, classified.EpisodeNumber)
	}
	if classified.Title != "Show One S01E01" {
		t.Fatalf("expected normalized episode title, got %q", classified.Title)
	}
}

func TestClassifyFlatMixedFolderUsesFallbackOrderForWeakLeadingNumbers(t *testing.T) {
	t.Parallel()

	snapshot := scanDirectorySnapshot{
		Path: "/library/Pack",
		Objects: []storage.Object{
			{Path: "/library/Pack/1-bdg01-suzanna-egals-11_hq.mp4.mp4"},
			{Path: "/library/Pack/1-bdg03-akiho-nishimura-egals-13_hq.mp4.mp4"},
		},
	}
	decision := resolveDirectoryShape("mixed", "/library", snapshot)
	classified := classifyMediaFileWithDirectoryDecision("mixed", "/library", snapshot.Objects[1], snapshot.Path, decision)

	if classified.Type != "episode" {
		t.Fatalf("expected episode classification, got %#v", classified)
	}
	if classified.SeriesTitle != "Pack" {
		t.Fatalf("expected series title from flat folder, got %q", classified.SeriesTitle)
	}
	if classified.SeasonNumber == nil || *classified.SeasonNumber != 1 || classified.EpisodeNumber == nil || *classified.EpisodeNumber != 2 {
		t.Fatalf("expected sorted fallback episode 2, got season=%v episode=%v", classified.SeasonNumber, classified.EpisodeNumber)
	}
	if classified.Title != "Pack S01E02" {
		t.Fatalf("expected fallback title, got %q", classified.Title)
	}
}

func TestResolveDirectoryShapeGroupsMovieVersions(t *testing.T) {
	t.Parallel()

	snapshot := scanDirectorySnapshot{
		Path: "/library/Movie A",
		Objects: []storage.Object{
			{Path: "/library/Movie A/Movie.A.2024.1080p.mkv"},
			{Path: "/library/Movie A/Movie.A.2024.2160p.Directors.Cut.mkv"},
			{Path: "/library/Movie A/trailer.mp4"},
		},
	}
	decision := resolveDirectoryShape("mixed", "/library", snapshot)
	if decision.Shape != directoryShapeMovieFolder {
		t.Fatalf("expected movie version grouping, got %#v", decision)
	}
}

func TestResolveDirectoryShapeKeepsIndependentMovieFilesReviewable(t *testing.T) {
	t.Parallel()

	snapshot := scanDirectorySnapshot{
		Path: "/My Pack/电影",
		Objects: []storage.Object{
			{Path: "/My Pack/电影/Avatar.Fire.And.Ash.2025.MA.x264.WEB-DL.1080p-Jaskier.mkv"},
			{Path: "/My Pack/电影/The.Lychee.Road.2025.1080p.WEB-DL.H264.AAC-QuickIO.mp4"},
			{Path: "/My Pack/电影/The.Super.Mario.Galaxy.Movie.2026.1080p.WEB-RIP.x265.10Bit.HEVC.Eng.DD.5.1+Sub.ViTO.mkv"},
		},
	}
	decision := resolveDirectoryShape("mixed", "/My Pack", snapshot)
	if decision.Shape != directoryShapeMixedFolder {
		t.Fatalf("expected independent movie files to stay reviewable/mixed, got %#v", decision)
	}
}

func TestResolveDirectoryShapeDoesNotTreatMovieCodecNumbersAsEpisodeEvidence(t *testing.T) {
	t.Parallel()

	snapshot := scanDirectorySnapshot{
		Path: "/My Pack/电影",
		Objects: []storage.Object{
			{Path: "/My Pack/电影/Avatar.Fire.And.Ash.2025.MA.x264.WEB-DL.1080p-Jaskier.mkv"},
			{Path: "/My Pack/电影/The.Lychee.Road.2025.1080p.WEB-DL.H264.AAC-QuickIO.mp4"},
			{Path: "/My Pack/电影/The.Super.Mario.Galaxy.Movie.2026.1080p.WEB-RIP.x265.10Bit.HEVC.Eng.DD.5.1+Sub.ViTO.mkv"},
		},
	}
	decision := resolveDirectoryShape("mixed", "/My Pack", snapshot)
	if decision.Shape == directoryShapeFlatEpisodeFolder {
		t.Fatalf("expected movie files not to form episode folder, got %#v", decision)
	}
	for _, object := range snapshot.Objects {
		signals := resolveFilenameSignals("mixed", "/My Pack", object)
		if signals.EpisodeNumber != nil || len(signals.EpisodeNumbers) > 0 {
			t.Fatalf("expected no episode evidence for %s, got %#v", object.Path, signals)
		}
	}
}

func TestClassifyMediaFileWithDirectorySummaryKeepsNumericEpisodeWithQuality(t *testing.T) {
	t.Parallel()

	snapshot := scanDirectorySnapshot{
		Path: "/My Pack/电视剧/我的山与海.2160p",
	}
	for episode := 1; episode <= 30; episode++ {
		snapshot.Objects = append(snapshot.Objects, storage.Object{Path: fmt.Sprintf("/My Pack/电视剧/我的山与海.2160p/%02d.2160p.HD国语中字无水印[最新电影www.5266ys.com].mkv", episode)})
	}
	decision := resolveDirectoryShape("mixed", "/My Pack", snapshot)
	classified := classifyMediaFileWithDirectorySummary("mixed", "/My Pack", snapshot.Objects[24], snapshot.Path, decision, snapshot, buildScanDirectorySummary("mixed", "/My Pack", snapshot))
	if classified.Type != "episode" {
		t.Fatalf("expected numeric quality filename to classify as episode, got %#v decision=%#v", classified, decision)
	}
	if classified.SeriesTitle != "我的山与海" {
		t.Fatalf("expected clean series title, got %q", classified.SeriesTitle)
	}
	if classified.SeasonNumber == nil || *classified.SeasonNumber != 1 || classified.EpisodeNumber == nil || *classified.EpisodeNumber != 25 {
		season := 0
		if classified.SeasonNumber != nil {
			season = *classified.SeasonNumber
		}
		episode := 0
		if classified.EpisodeNumber != nil {
			episode = *classified.EpisodeNumber
		}
		t.Fatalf("expected S01E25, got S%02dE%02d", season, episode)
	}
}

func TestClassifyMediaFileRecognizesEmbeddedEpisodeToken(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("mixed", "/My Pack", storage.Object{Path: "/My Pack/电视剧/Hold.A.Court.Now.2026.EP01-26.HD1080P.X264.AAC.Mandarin.CHS.XLYS/Hold.A.Court.Now.2026.EP25.HD1080P.X264.AAC.Mandarin.CHS.XLYS.mkv"})
	if classified.Type != "episode" {
		t.Fatalf("expected embedded EP token to classify as episode, got %#v", classified)
	}
	if classified.EpisodeNumber == nil || *classified.EpisodeNumber != 25 {
		t.Fatalf("expected episode 25, got %#v", classified.EpisodeNumber)
	}
	if classified.SeasonNumber == nil || *classified.SeasonNumber != 1 {
		t.Fatalf("expected default season 1, got %#v", classified.SeasonNumber)
	}
}

func TestClassifyMediaFileInfersEpisodeFromNoisySeasonFolder(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("shows", "/library", storage.Object{Path: "/library/魔幻手机 (2008)/第 2 季 - 2160p WEB-DL HEVC DDP 2Audios/魔幻手机2：傻妞归来 S02E01 - 第 1 集 - 2160p WEB-DL HEVC DDP 2Audios.mp4"})
	if classified.Type != "episode" {
		t.Fatalf("expected episode classification, got %#v", classified)
	}
	if classified.SeriesTitle != "魔幻手机" {
		t.Fatalf("expected series title from show folder, got %#v", classified.SeriesTitle)
	}
	if classified.SeasonNumber == nil || *classified.SeasonNumber != 2 || classified.EpisodeNumber == nil || *classified.EpisodeNumber != 1 {
		t.Fatalf("expected S02E01 from noisy season folder, got season=%v episode=%v", classified.SeasonNumber, classified.EpisodeNumber)
	}
	if classified.Title != "魔幻手机 S02E01" {
		t.Fatalf("expected normalized episode title, got %q", classified.Title)
	}
}

func TestClassifyMediaFileExtractsSeriesFromEmbeddedSeasonDirectory(t *testing.T) {
	t.Parallel()

	path := "/library/BEEF (2023) S02 (1080p NF WEB-DL x265 10bit EAC3 Atmos 5.1 Silence)/BEEF (2023) - S02E08 - It Will Stay This Way and You Will Obey (1080p NF WEB-DL x265 Silence).mkv"
	classified := classifyMediaFile("shows", "/library", storage.Object{Path: path})
	if classified.Type != "episode" {
		t.Fatalf("expected episode classification, got %#v", classified)
	}
	if classified.SeriesTitle != "BEEF" {
		t.Fatalf("expected clean series title from embedded season directory, got %q", classified.SeriesTitle)
	}
	if classified.Title != "BEEF S02E08" {
		t.Fatalf("expected normalized episode title, got %q", classified.Title)
	}
	if classified.SeasonNumber == nil || *classified.SeasonNumber != 2 || classified.EpisodeNumber == nil || *classified.EpisodeNumber != 8 {
		t.Fatalf("expected S02E08 metadata, got season=%v episode=%v", classified.SeasonNumber, classified.EpisodeNumber)
	}
}

func TestClassifyMediaFilePrefersChineseTitleFromNoisyBilingualFolder(t *testing.T) {
	t.Parallel()

	path := "/library/【高清剧集网发布 www.TTHDTT.com】高等教欲[全8集][简繁英字幕].Vladimir.S01.2160p.NF.WEB-DL.DDP.5.1.Atmos.HDR10.H.265-BlackTV/Vladimir.S01E08.Against.Interpretation.2160p.NF.WEB-DL.DDP.5.1.Atmos.HDR10.H.265-BlackTV.mkv"
	classified := classifyMediaFile("shows", "/library", storage.Object{Path: path})
	if classified.Type != "episode" {
		t.Fatalf("expected episode classification, got %#v", classified)
	}
	if classified.SeriesTitle != "高等教欲" {
		t.Fatalf("expected Chinese series title from noisy bilingual folder, got %q", classified.SeriesTitle)
	}
	if classified.Title != "高等教欲 S01E08" {
		t.Fatalf("expected normalized episode title, got %q", classified.Title)
	}
}

func TestClassifyMediaFileUsesSeriesFolderForMatchedEpisode(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("shows", "/library", storage.Object{Path: "/library/灵笼/第二季/灵笼 第二季.S02E03.mp4"})
	if classified.Type != "episode" {
		t.Fatalf("expected episode classification, got %#v", classified)
	}
	if classified.SeriesTitle != "灵笼" {
		t.Fatalf("expected series title to normalize by folder, got %q", classified.SeriesTitle)
	}
	if classified.SeasonNumber == nil || *classified.SeasonNumber != 2 || classified.EpisodeNumber == nil || *classified.EpisodeNumber != 3 {
		t.Fatalf("expected S02E03 metadata, got season=%v episode=%v", classified.SeasonNumber, classified.EpisodeNumber)
	}
}

func TestClassifyMediaFileUsesSeriesFolderWhenLibraryTypeIsMovies(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("movies", "/library", storage.Object{Path: "/library/灵笼 第二季/灵笼 第二季[www.4KHDR.CN]Incarnation.S02E01.2025.2160p.WEB-DL.H264.AAC-4KHDR世界.mp4"})
	if classified.Type != "episode" {
		t.Fatalf("expected episode classification, got %#v", classified)
	}
	if classified.SeriesTitle != "灵笼 第二季" {
		t.Fatalf("expected series title from folder fallback, got %q", classified.SeriesTitle)
	}
	if classified.Title != "灵笼 第二季 S02E01" {
		t.Fatalf("expected normalized title from folder fallback, got %q", classified.Title)
	}
}

func TestClassifyMediaFileStripsMovieReleaseNoise(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("movies", "/library", storage.Object{Path: "/library/Movie.Name.2024.1080p.WEB-DL.x265-GROUP.mkv"})
	if classified.Type != "movie" {
		t.Fatalf("expected movie classification, got %#v", classified)
	}
	if classified.Title != "Movie Name" {
		t.Fatalf("expected cleaned movie title, got %q", classified.Title)
	}
	if classified.Year == nil || *classified.Year != 2024 {
		t.Fatalf("expected release year 2024, got %#v", classified.Year)
	}
}

func TestClassifyMediaFileExtractsHashtagTags(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("movies", "/library", storage.Object{Path: "/library/Movie.Name.#IMAX.#国语.2024.1080p.mkv"})
	if classified.Type != "movie" {
		t.Fatalf("expected movie classification, got %#v", classified)
	}
	if classified.Title != "Movie Name" {
		t.Fatalf("expected hashtags removed from normalized title, got %q", classified.Title)
	}
	if !reflect.DeepEqual(classified.Tags, []string{"IMAX", "国语"}) {
		t.Fatalf("expected hashtag tags, got %#v", classified.Tags)
	}
}

func TestClassifyMediaFileStripsWebsiteWatermarkAndTechnicalNoise(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("movies", "/library", storage.Object{Path: "/library/[www.example.com]Some.Movie.2023.HD1080P.mkv"})
	if classified.Type != "movie" {
		t.Fatalf("expected movie classification, got %#v", classified)
	}
	if classified.Title != "Some Movie" {
		t.Fatalf("expected website watermark and quality noise to be removed, got %q", classified.Title)
	}
	if classified.Year == nil || *classified.Year != 2023 {
		t.Fatalf("expected extracted year 2023, got %#v", classified.Year)
	}
	if classified.NormalizationVersion != titleclean.NormalizationVersion || len(classified.RemovedTokens) == 0 {
		t.Fatalf("expected normalization evidence, got version=%q tokens=%#v", classified.NormalizationVersion, classified.RemovedTokens)
	}
}

func TestClassifyMediaFileStripsNoisyTVFilename(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("shows", "/library", storage.Object{Path: "/library/Show Name/Season 1/Show.Name.S01E02.1080p.NF.WEB-DL.DDP5.1.Atmos.x264-GROUP.mkv"})
	if classified.Type != "episode" {
		t.Fatalf("expected episode classification, got %#v", classified)
	}
	if classified.SeriesTitle != "Show Name" || classified.Title != "Show Name S01E02" {
		t.Fatalf("expected normalized noisy episode title, got title=%q series=%q", classified.Title, classified.SeriesTitle)
	}
	if classified.SeasonNumber == nil || *classified.SeasonNumber != 1 || classified.EpisodeNumber == nil || *classified.EpisodeNumber != 2 {
		t.Fatalf("expected S01E02 metadata, got season=%v episode=%v", classified.SeasonNumber, classified.EpisodeNumber)
	}
}

func TestCatalogScanEvidencePayloadIncludesNormalizationEvidence(t *testing.T) {
	t.Parallel()

	payloadJSON := buildCatalogScanEvidencePayload(catalogScanArtifact{
		SourcePath:           "/library/Movie.Name.2024.2160p.WEB-DL.x265.mkv",
		Title:                "Movie Name",
		NormalizationVersion: titleclean.NormalizationVersion,
		RemovedTokens:        []titleclean.RemovedToken{{Value: "2024", Reason: "year"}, {Value: "2160p", Reason: "quality"}},
	}, nil)
	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["normalization_version"] != titleclean.NormalizationVersion {
		t.Fatalf("expected normalization version in payload, got %#v", payload)
	}
	removed, ok := payload["removed_tokens"].([]any)
	if !ok || len(removed) != 2 {
		t.Fatalf("expected removed tokens in payload, got %#v", payload["removed_tokens"])
	}
}

func TestCatalogScanEvidenceKeepsFilenameHintsSeparateFromTechnicalMetadata(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("movies", "/library", storage.Object{Path: "/library/Movie.Name.2024.1080p.WEB-DL.DDP5.1.x265-GROUP.mkv"})
	payloadJSON := buildCatalogScanEvidencePayload(catalogScanArtifact{
		SourcePath:           classified.SourcePath,
		Title:                classified.Title,
		NormalizationVersion: classified.NormalizationVersion,
		RemovedTokens:        classified.RemovedTokens,
		FilenameSignals:      classified.FilenameSignals,
	}, nil)
	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if _, ok := payload["filename_release_hints"]; !ok {
		t.Fatalf("expected filename release hints in evidence payload, got %#v", payload)
	}
	for _, key := range []string{"width", "height", "video_codec", "audio_layout", "audio_channels", "subtitles"} {
		if _, ok := payload[key]; ok {
			t.Fatalf("expected filename hints not to populate authoritative technical key %q: %#v", key, payload)
		}
	}
}

func TestCatalogScanEvidenceIncludesFilenameAndDirectorySummaryEvidence(t *testing.T) {
	t.Parallel()

	summary := scanDirectorySummary{Path: "/library/Show", SnapshotDerived: true}
	classified := classifyMediaFile("shows", "/library", storage.Object{Path: "/library/Show/Season 1/Show.S01E02.1080p.mkv"})
	classified.FilenameSignals.Evidence = append(classified.FilenameSignals.Evidence, directorySummaryEvidence(summary)...)
	payloadJSON := buildCatalogScanEvidencePayload(catalogScanArtifact{SourcePath: classified.SourcePath, Title: classified.Title, FilenameSignals: classified.FilenameSignals}, nil)
	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	evidence, ok := payload["filename_signal_evidence"].([]any)
	if !ok || len(evidence) == 0 {
		t.Fatalf("expected filename signal evidence, got %#v", payload)
	}
	foundDirectorySummary := false
	for _, raw := range evidence {
		item, _ := raw.(map[string]any)
		if item["source"] == "directory_summary" {
			foundDirectorySummary = true
		}
	}
	if !foundDirectorySummary {
		t.Fatalf("expected directory summary evidence, got %#v", evidence)
	}
}

func TestClassifyMediaFileUsesParentFolderForGenericMovieName(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("movies", "/library", storage.Object{Path: "/library/Dune Part Two (2024)/movie.mkv"})
	if classified.Type != "movie" {
		t.Fatalf("expected movie classification, got %#v", classified)
	}
	if classified.Title != "Dune Part Two" {
		t.Fatalf("expected parent folder movie title, got %q", classified.Title)
	}
}

func TestClassifyMediaFileInfersEpisodeFromSeriesPrefixAndSeasonFolder(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("shows", "/library", storage.Object{Path: "/library/Show One/Season 1/Show One - 03.mkv"})
	if classified.Type != "episode" {
		t.Fatalf("expected episode classification, got %#v", classified)
	}
	if classified.SeriesTitle != "Show One" {
		t.Fatalf("expected series title from show folder, got %q", classified.SeriesTitle)
	}
	if classified.SeasonNumber == nil || *classified.SeasonNumber != 1 || classified.EpisodeNumber == nil || *classified.EpisodeNumber != 3 {
		t.Fatalf("expected S01E03 metadata, got season=%v episode=%v", classified.SeasonNumber, classified.EpisodeNumber)
	}
}

func TestClassifyMediaFileInfersChineseEpisodeFromSeasonFolder(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("shows", "/library", storage.Object{Path: "/library/灵笼/第一季/第03集.mp4"})
	if classified.Type != "episode" {
		t.Fatalf("expected episode classification, got %#v", classified)
	}
	if classified.SeriesTitle != "灵笼" {
		t.Fatalf("expected normalized series title, got %q", classified.SeriesTitle)
	}
	if classified.SeasonNumber == nil || *classified.SeasonNumber != 1 || classified.EpisodeNumber == nil || *classified.EpisodeNumber != 3 {
		t.Fatalf("expected S01E03 metadata, got season=%v episode=%v", classified.SeasonNumber, classified.EpisodeNumber)
	}
}

func TestClassifyMediaFileFindsSeasonDirectoryAboveImmediateParent(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("shows", "/library", storage.Object{Path: "/library/Show One/Season 1/Disc 1/03.mkv"})
	if classified.Type != "episode" {
		t.Fatalf("expected episode classification, got %#v", classified)
	}
	if classified.SeriesTitle != "Show One" {
		t.Fatalf("expected series title from ancestor folder, got %q", classified.SeriesTitle)
	}
	if classified.SeasonNumber == nil || *classified.SeasonNumber != 1 || classified.EpisodeNumber == nil || *classified.EpisodeNumber != 3 {
		t.Fatalf("expected S01E03 metadata, got season=%v episode=%v", classified.SeasonNumber, classified.EpisodeNumber)
	}
}

func TestExtraTypeSignalUsesBoundedTokens(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"trailer":               "trailer",
		"teaser":                "trailer",
		"预告片":                   "trailer",
		"behind-the-scenes":     "behind_the_scenes",
		"幕后花絮":                  "behind_the_scenes",
		"sample":                "sample",
		"featurette":            "featurette",
		"特典":                    "featurette",
		"interview":             "interview",
		"PV01":                  "preview",
		"先导预览":                  "preview",
		"deleted scene":         "deleted_scene",
		"Movie - deleted scene": "deleted_scene",
	}
	for input, expected := range cases {
		if actual := extraTypeSignal(input); actual != expected {
			t.Fatalf("expected %q to resolve to %q, got %q", input, expected, actual)
		}
	}

	for _, input := range []string{"Sampler", "Trailerpark Story", "Featurettesque", "Interviewed Hero"} {
		if actual := extraTypeSignal(input); actual != "" {
			t.Fatalf("expected %q not to resolve as an extra, got %q", input, actual)
		}
	}
}

func TestVideoFileRoleSignalUsesPathSegments(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"/library/Movie A/Trailers/Movie A Teaser.mkv":  "trailer",
		"/library/Show/Season 1/PV01.mp4":               "preview",
		"/library/Movie A/extras/behind the scenes.mkv": "behind_the_scenes",
		"/library/Movie A/Movie A 2024.mkv":             "",
	}
	for input, expected := range cases {
		if actual := videoFileRoleSignal(input); actual != expected {
			t.Fatalf("expected %q to resolve to %q, got %q", input, expected, actual)
		}
	}
}

func TestResolveFilenameSignalsBuildsEpisodeAndMovieCandidates(t *testing.T) {
	t.Parallel()

	signals := resolveFilenameSignals("auto", "/library", storage.Object{Path: "/library/Show/Season 1/Show.S01E02.2024.1080p.mkv"})
	if len(signals.Candidates) < 2 {
		t.Fatalf("expected competing episode and movie candidates, got %#v", signals.Candidates)
	}
	var hasEpisode, hasMovie bool
	for _, candidate := range signals.Candidates {
		switch candidate.Type {
		case scanDecisionCandidateEpisode:
			hasEpisode = true
			if candidate.EpisodeNumber == nil || *candidate.EpisodeNumber != 2 {
				t.Fatalf("expected episode candidate for episode 2, got %#v", candidate)
			}
		case scanDecisionCandidateMovie:
			hasMovie = true
		}
	}
	if !hasEpisode || !hasMovie {
		t.Fatalf("expected episode and movie alternatives, got %#v", signals.Candidates)
	}
}

func TestResolveFilenameSignalsBuildsMovieCandidate(t *testing.T) {
	t.Parallel()

	signals := resolveFilenameSignals("auto", "/library", storage.Object{Path: "/library/Movie Name (2024)/Movie.Name.2024.2160p.mkv"})
	if len(signals.Candidates) == 0 || signals.Candidates[0].Type != scanDecisionCandidateMovie {
		t.Fatalf("expected movie candidate, got %#v", signals.Candidates)
	}
	if signals.Candidates[0].Confidence < 0.7 {
		t.Fatalf("expected year-backed movie confidence, got %#v", signals.Candidates[0])
	}
}

func TestFastCandidatesIncludeAttachmentRolesAndMovieVersion(t *testing.T) {
	t.Parallel()

	roles := map[string]string{
		"/library/Movie/trailer.mkv":    scanDecisionRoleTrailer,
		"/library/Movie/sample.mkv":     scanDecisionRoleSample,
		"/library/Movie/PV01.mp4":       "preview",
		"/library/Movie/featurette.mkv": scanDecisionRoleExtra,
	}
	for input, wantRole := range roles {
		signals := resolveFilenameSignals("auto", "/library", storage.Object{Path: input})
		if len(signals.Candidates) == 0 || signals.Candidates[0].Type != scanDecisionCandidateAttachment || signals.Candidates[0].Role != wantRole {
			t.Fatalf("expected attachment candidate role %q for %s, got %#v", wantRole, input, signals.Candidates)
		}
	}

	signals := resolveFilenameSignals("auto", "/library", storage.Object{Path: "/library/Movie.2024.2160p.Directors.Cut-GROUP.mkv"})
	foundVersion := false
	for _, candidate := range signals.Candidates {
		if candidate.Type == scanDecisionCandidateMovieVersion {
			foundVersion = true
		}
	}
	if !foundVersion {
		t.Fatalf("expected movie version candidate, got %#v", signals.Candidates)
	}
}

func TestFastDecisionStatusUsesConservativeThresholds(t *testing.T) {
	t.Parallel()

	if got := classifyFastDecisionStatus(0.9, nil, defaultFastClassificationThresholds); got != scanDecisionStatusConfirmedFast {
		t.Fatalf("expected high confidence confirmed, got %q", got)
	}
	if got := classifyFastDecisionStatus(0.7, nil, defaultFastClassificationThresholds); got != scanDecisionStatusProvisional {
		t.Fatalf("expected medium confidence provisional, got %q", got)
	}
	closeConfidence := 0.78
	if got := classifyFastDecisionStatus(0.86, []scanDecisionAlternative{{Type: scanDecisionCandidateEpisode, Confidence: &closeConfidence}}, defaultFastClassificationThresholds); got != scanDecisionStatusReviewRequired {
		t.Fatalf("expected close alternative review required, got %q", got)
	}
	if got := classifyFastDecisionStatus(0.4, nil, defaultFastClassificationThresholds); got != scanDecisionStatusReviewRequired {
		t.Fatalf("expected low confidence review required, got %q", got)
	}
}

func TestScanDecisionAlternativesPreserveCandidateConflicts(t *testing.T) {
	t.Parallel()

	candidates := []fastClassificationCandidate{
		{Type: scanDecisionCandidateMovie, Role: scanDecisionRoleMain, Confidence: 0.78, Reason: "movie evidence"},
		{Type: scanDecisionCandidateEpisode, Role: scanDecisionRoleMain, Confidence: 0.72, Reason: "episode evidence"},
		{Type: scanDecisionCandidateMovieVersion, Role: scanDecisionRoleMain, Confidence: 0.62, Reason: "version evidence"},
		{Type: scanDecisionCandidateIndependentMovie, Role: scanDecisionRoleMain, Confidence: 0.55, Reason: "independent movie evidence"},
		{Type: scanDecisionCandidateAttachment, Role: scanDecisionRoleTrailer, Confidence: 0.5, Reason: "attachment evidence"},
	}
	alternatives := scanDecisionAlternativesFromCandidates(candidates, 0)
	if len(alternatives) != 4 {
		t.Fatalf("expected four alternatives, got %#v", alternatives)
	}
	wantTypes := []string{scanDecisionCandidateEpisode, scanDecisionCandidateMovieVersion, scanDecisionCandidateIndependentMovie, scanDecisionCandidateAttachment}
	for idx, wantType := range wantTypes {
		if alternatives[idx].Type != wantType {
			t.Fatalf("expected alternative %d type %q, got %#v", idx, wantType, alternatives)
		}
	}
}

func TestFilenameEvidenceSummariesConvertToDecisionEvidence(t *testing.T) {
	t.Parallel()

	summaries := []filenameEvidenceSummary{
		{Kind: filenameSignalKindQuality, Source: "filename", Value: "2160P", Reason: filenameSignalReasonQualityNoise},
		{Kind: filenameSignalKindAudio, Source: "filename", Value: "DDP5.1", Reason: filenameSignalReasonSuppressWeakEpisodeNumber},
		{Kind: "", Source: "filename", Value: "ignored"},
	}

	evidence := filenameEvidenceSummariesToScanDecisionEvidence(summaries)
	if !reflect.DeepEqual(evidence, []scanDecisionEvidence{
		{Kind: filenameSignalKindQuality, Source: "filename", Value: "2160P"},
		{Kind: filenameSignalKindAudio, Source: "filename", Value: "DDP5.1"},
	}) {
		t.Fatalf("unexpected decision evidence: %#v", evidence)
	}
}

func TestExtractFilenameSignalModelCoversReleaseAndRoleSignals(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		path          string
		wantTitle     string
		wantYear      int
		wantSeason    int
		wantEpisode   int
		wantEpisodeTo int
		wantRole      string
		wantQuality   string
		wantWebsite   string
		wantGroup     string
	}{
		{name: "dense movie", path: "/library/Dune.Part.Two.2024.2160p.UHD.BluRay.DV.TrueHD.Atmos.7.1.x265-GROUP.mkv", wantTitle: "Dune Part Two", wantYear: 2024, wantQuality: "2160P UHD BLURAY DV X265", wantGroup: "GROUP"},
		{name: "tv release", path: "/library/Show/Season 1/Show.Name.S01E02.1080p.WEB-DL.DDP5.1.x264-GROUP.mkv", wantTitle: "Show Name", wantSeason: 1, wantEpisode: 2, wantQuality: "1080P WEB-DL X264", wantGroup: "GROUP"},
		{name: "chinese episode", path: "/library/灵笼/第一季/第03集.mp4", wantTitle: "第03集", wantEpisode: 3},
		{name: "multi episode", path: "/library/Show/Show.S01E01-E02.mkv", wantTitle: "Show", wantSeason: 1, wantEpisode: 1, wantEpisodeTo: 2},
		{name: "url watermark", path: "/library/[www.example.com]Some.Movie.2023.HD1080P.mkv", wantTitle: "Some Movie", wantYear: 2023, wantQuality: "1080P", wantWebsite: "www.example.com"},
		{name: "trailer role", path: "/library/Movie/Trailers/Movie.Teaser.mkv", wantTitle: "Movie Teaser", wantRole: "trailer"},
		{name: "sample role", path: "/library/Movie/sample.mkv", wantTitle: "sample", wantRole: "sample"},
		{name: "pv role", path: "/library/Show/PV01.mp4", wantTitle: "PV01", wantRole: "preview"},
		{name: "featurette role", path: "/library/Movie/featurette.mkv", wantTitle: "featurette", wantRole: "featurette"},
		{name: "extra role", path: "/library/Movie/extras/behind the scenes.mkv", wantTitle: "behind the scenes", wantRole: "behind_the_scenes"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := extractFilenameSignalModel(tt.path)
			if model.Identity.TitleCandidate != tt.wantTitle {
				t.Fatalf("expected title %q, got %#v", tt.wantTitle, model)
			}
			if tt.wantYear > 0 && (model.Identity.Year == nil || *model.Identity.Year != tt.wantYear) {
				t.Fatalf("expected year %d, got %#v", tt.wantYear, model.Identity.Year)
			}
			if tt.wantSeason > 0 && (model.Identity.SeasonNumber == nil || *model.Identity.SeasonNumber != tt.wantSeason) {
				t.Fatalf("expected season %d, got %#v", tt.wantSeason, model.Identity.SeasonNumber)
			}
			if tt.wantEpisode > 0 && (model.Identity.EpisodeNumber == nil || *model.Identity.EpisodeNumber != tt.wantEpisode) {
				t.Fatalf("expected episode %d, got %#v", tt.wantEpisode, model.Identity.EpisodeNumber)
			}
			if tt.wantEpisodeTo > 0 && (model.Identity.EpisodeEnd == nil || *model.Identity.EpisodeEnd != tt.wantEpisodeTo) {
				t.Fatalf("expected episode end %d, got %#v", tt.wantEpisodeTo, model.Identity.EpisodeEnd)
			}
			if tt.wantRole != "" && model.RoleHints.Role != tt.wantRole {
				t.Fatalf("expected role %q, got %#v", tt.wantRole, model.RoleHints)
			}
			if tt.wantQuality != "" && model.ReleaseHints.Quality != tt.wantQuality {
				t.Fatalf("expected quality %q, got %#v", tt.wantQuality, model.ReleaseHints)
			}
			if tt.wantWebsite != "" && model.ReleaseHints.Website != tt.wantWebsite {
				t.Fatalf("expected website %q, got %#v", tt.wantWebsite, model.ReleaseHints)
			}
			if tt.wantGroup != "" && model.ReleaseHints.ReleaseGroup != tt.wantGroup {
				t.Fatalf("expected release group %q, got %#v", tt.wantGroup, model.ReleaseHints)
			}
		})
	}
}

func TestAudioChannelTokensSuppressWeakEpisodeInference(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"/library/Movie.Name.5.1.1080p.WEB-DL.mkv",
		"/library/Movie.Name.7.1.1080p.WEB-DL.mkv",
		"/library/Movie.Name.DDP5.1.1080p.WEB-DL.mkv",
		"/library/Movie.Name.TrueHD.Atmos.7.1.2160p.mkv",
	} {
		signals := resolveFilenameSignals("auto", "/library", storage.Object{Path: path})
		if signals.EpisodeNumber != nil || len(signals.EpisodeNumbers) > 0 {
			t.Fatalf("expected audio channel token not to create episode evidence for %s: %#v", path, signals)
		}
		foundSuppression := false
		for _, evidence := range signals.Model.Evidence {
			if evidence.Kind == filenameSignalKindAntiMisclassification && evidence.Reason == filenameSignalReasonSuppressWeakEpisodeNumber {
				foundSuppression = true
			}
		}
		if !foundSuppression {
			t.Fatalf("expected anti-misclassification evidence for %s: %#v", path, signals.Model.Evidence)
		}
	}
}

func TestQualityAndCodecTokensDoNotCreateEpisodeEvidence(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("movies", "/library", storage.Object{Path: "/library/Movie.Name.2160p.1080p.x264.x265.H.264.HEVC.mkv"})
	if classified.Type != "movie" {
		t.Fatalf("expected movie classification, got %#v", classified)
	}
	if classified.Title != "Movie Name" {
		t.Fatalf("expected quality and codec tokens removed from title, got %q", classified.Title)
	}
	signals := resolveFilenameSignals("auto", "/library", storage.Object{Path: "/library/Movie.Name.2160p.1080p.x264.x265.H.264.HEVC.mkv"})
	if signals.EpisodeNumber != nil || len(signals.EpisodeNumbers) > 0 {
		t.Fatalf("expected no episode evidence from quality/codecs, got %#v", signals)
	}
	if signals.Model.ReleaseHints.Quality == "" || signals.Model.ReleaseHints.Codec == "" {
		t.Fatalf("expected quality and codec release hints, got %#v", signals.Model.ReleaseHints)
	}
}

func TestTitleCleanRemovedTokensRemainFilenameSignalEvidence(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("movies", "/library", storage.Object{Path: "/library/Movie.Name.2024.2160p.WEB-DL.x265.mkv"})
	if classified.Title != "Movie Name" {
		t.Fatalf("expected cleaned title, got %q", classified.Title)
	}
	if len(classified.FilenameSignals.CleanupEvidence) == 0 || len(classified.FilenameSignals.Evidence) == 0 {
		t.Fatalf("expected cleanup and filename signal evidence, got %#v", classified.FilenameSignals)
	}
	foundQuality := false
	for _, evidence := range classified.FilenameSignals.Evidence {
		if evidence.Kind == filenameSignalKindQuality && evidence.Value != "" {
			foundQuality = true
		}
	}
	if !foundQuality {
		t.Fatalf("expected removed quality token preserved as filename evidence, got %#v", classified.FilenameSignals.Evidence)
	}
}

func TestMovieReleaseTokensDoNotLookNumericOrTitleLike(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("movies", "/library", storage.Object{Path: "/library/Movie.Name.2024.2160p.UHD.WEB-DL.DDP5.1.TrueHD.Atmos.x265.HEVC.CHS-TEAM.mkv"})
	if classified.Type != "movie" {
		t.Fatalf("expected movie classification, got %#v", classified)
	}
	if classified.Title != "Movie Name" {
		t.Fatalf("expected release tokens stripped from title, got %q", classified.Title)
	}
	signals := resolveFilenameSignals("auto", "/library", storage.Object{Path: classified.SourcePath})
	if signals.EpisodeNumber != nil || len(signals.EpisodeNumbers) > 0 {
		t.Fatalf("expected release tokens not to create episode evidence, got %#v", signals)
	}
	if signals.Model.ReleaseHints.Quality == "" || signals.Model.ReleaseHints.Audio == "" || signals.Model.ReleaseHints.Codec == "" || signals.Model.ReleaseHints.Subtitle == "" {
		t.Fatalf("expected audio, quality, codec, source, and subtitle hints, got %#v", signals.Model.ReleaseHints)
	}
}

func TestClassifyMediaFileStripsAudioSubtitleCompositeAndMixedCaseReleaseGroup(t *testing.T) {
	t.Parallel()

	classified := classifyMediaFile("mixed", "/My Pack", storage.Object{Path: "/My Pack/电影/The.Super.Mario.Galaxy.Movie.2026.1080p.WEB-RIP.x265.10Bit.HEVC.Eng.DD.5.1+Sub.ViTO.mkv"})
	if classified.Type != "movie" {
		t.Fatalf("expected movie classification, got %#v", classified)
	}
	if classified.Title != "The Super Mario Galaxy Movie" {
		t.Fatalf("expected cleaned movie title, got %q", classified.Title)
	}
}

func TestBuildScanDirectorySummaryCapturesSiblingLayouts(t *testing.T) {
	t.Parallel()

	flat := buildScanDirectorySummary("mixed", "/library", scanDirectorySnapshot{Path: "/library/Show", Objects: []storage.Object{{Path: "/library/Show/01.mkv"}, {Path: "/library/Show/02.mkv"}, {Path: "/library/Show/03.mkv"}}})
	if flat.VideoCount != 3 || flat.LikelyMainCount != 3 || !reflect.DeepEqual(flat.NumericSequence, []int{1, 2, 3}) {
		t.Fatalf("expected flat numeric episode summary, got %#v", flat)
	}

	explicit := buildScanDirectorySummary("mixed", "/library", scanDirectorySnapshot{Path: "/library/Show", Objects: []storage.Object{{Path: "/library/Show/Show.S01E01.mkv"}, {Path: "/library/Show/Show.S01E02.mkv"}}})
	if explicit.ExplicitEpisodeCount != 2 || explicit.CommonTitleStem != "show s01e01" && explicit.CommonTitleStem != "show s01e02" {
		t.Fatalf("expected explicit episode sibling summary, got %#v", explicit)
	}

	versions := buildScanDirectorySummary("mixed", "/library", scanDirectorySnapshot{Path: "/library/Movie", Objects: []storage.Object{{Path: "/library/Movie/Movie.2024.1080p.mkv"}, {Path: "/library/Movie/Movie.2024.2160p.Directors.Cut.mkv"}, {Path: "/library/Movie/sample.mkv"}}})
	if versions.LikelyMainCount != 2 || versions.AttachmentCount != 1 || versions.VersionEvidenceCount < 2 {
		t.Fatalf("expected movie version and attachment summary, got %#v", versions)
	}

	independent := buildScanDirectorySummary("mixed", "/library", scanDirectorySnapshot{Path: "/library/Movies", Objects: []storage.Object{{Path: "/library/Movies/Alien.1979.mkv"}, {Path: "/library/Movies/Aliens.1986.mkv"}}})
	if independent.TitleYearMovieCount != 2 || independent.LikelyMainCount != 2 {
		t.Fatalf("expected independent movie summary, got %#v", independent)
	}
}

func TestScanModeCachesDirectorySummaryPerSnapshotDirectory(t *testing.T) {
	t.Parallel()

	mode := &scanMode{}
	snapshot := scanDirectorySnapshot{Path: "/library/Show", Objects: []storage.Object{{Path: "/library/Show/01.mkv"}, {Path: "/library/Show/02.mkv"}}}
	first := mode.directorySummary("mixed", "/library", snapshot)
	second := mode.directorySummary("mixed", "/library", scanDirectorySnapshot{Path: "/library/Show", Objects: []storage.Object{{Path: "/library/Show/03.mkv"}}})
	if first.VideoCount != 2 || second.VideoCount != 2 {
		t.Fatalf("expected cached summary to be reused, first=%#v second=%#v", first, second)
	}
	if len(mode.directorySummaries) != 1 {
		t.Fatalf("expected one cached summary, got %#v", mode.directorySummaries)
	}
}

func TestBuildScanDirectorySummaryHandlesLargeSiblingDirectory(t *testing.T) {
	t.Parallel()

	objects := make([]storage.Object, 0, 150)
	for episode := 1; episode <= 99; episode++ {
		objects = append(objects, storage.Object{Path: fmt.Sprintf("/library/Show/Show.S01E%02d.mkv", episode)})
	}
	summary := buildScanDirectorySummary("mixed", "/library", scanDirectorySnapshot{Path: "/library/Show", Objects: objects})
	if summary.VideoCount != 99 || summary.LikelyMainCount != 99 || summary.ExplicitEpisodeCount != 99 {
		t.Fatalf("expected large sibling summary counts, got %#v", summary)
	}
	if len(summary.NumericSequence) != 99 || summary.NumericSequence[0] != 1 || summary.NumericSequence[98] != 99 {
		t.Fatalf("expected ordered large episode sequence, got %#v", summary.NumericSequence)
	}
}
