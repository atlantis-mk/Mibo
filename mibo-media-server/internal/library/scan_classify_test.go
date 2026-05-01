package library

import (
	"encoding/json"
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
		"behind-the-scenes":     "behind_the_scenes",
		"sample":                "sample",
		"featurette":            "featurette",
		"interview":             "interview",
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
