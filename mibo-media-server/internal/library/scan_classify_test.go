package library

import (
	"reflect"
	"testing"

	"github.com/atlan/mibo-media-server/internal/storage"
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
