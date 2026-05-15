package scanrecognition

import (
	"fmt"
	"testing"
)

func TestClassifyTreeRecognizesSplitSeasonFolder(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/电视剧/六尺之下/第一季/01.mkv", IsVideo: true},
			{ID: 2, Path: "/media/电视剧/六尺之下/第一季/02.mkv", IsVideo: true},
		},
	})

	season := tree.Node("/media/电视剧/六尺之下/第一季")
	if season.Kind != DirectoryKindSeason {
		t.Fatalf("expected season folder, got %q", season.Kind)
	}

	series := tree.Node("/media/电视剧/六尺之下")
	if series.Kind != DirectoryKindSeries {
		t.Fatalf("expected parent series folder, got %q", series.Kind)
	}
}

func TestClassifyTreeRecognizesCombinedSeasonFolder(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/电视剧/六尺之下 第一季[全13集][中文字幕].Six.Feet.Under.2001.1080p.WEB-DL.x265.AC3-BitsTV/Six.Feet.Under.S01E01.2001.1080p.WEB-DL.x265.AC3-BitsTV.mkv", IsVideo: true},
			{ID: 2, Path: "/media/电视剧/六尺之下 第一季[全13集][中文字幕].Six.Feet.Under.2001.1080p.WEB-DL.x265.AC3-BitsTV/Six.Feet.Under.S01E02.2001.1080p.WEB-DL.x265.AC3-BitsTV.mkv", IsVideo: true},
		},
	})

	season := tree.Node("/media/电视剧/六尺之下 第一季[全13集][中文字幕].Six.Feet.Under.2001.1080p.WEB-DL.x265.AC3-BitsTV")
	if season.Kind != DirectoryKindSeason {
		t.Fatalf("expected combined folder to classify as season, got %q", season.Kind)
	}
}

func TestClassifyTreeRecognizesEpisodeGroupWithoutSeasonFolder(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/电视剧/六尺之下/Six.Feet.Under.S01E01.2001.1080p.WEB-DL.x265.mkv", IsVideo: true},
			{ID: 2, Path: "/media/电视剧/六尺之下/Six.Feet.Under.S01E02.2001.1080p.WEB-DL.x265.mkv", IsVideo: true},
		},
	})

	episodeGroup := tree.Node("/media/电视剧/六尺之下")
	if episodeGroup.Kind != DirectoryKindEpisodeGroup {
		t.Fatalf("expected episode group, got %q", episodeGroup.Kind)
	}
}

func TestClassifyTreeRecognizesEpisodeGroupWithExpectedCountFolder(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/电视剧/轮到你了[全20集][中文字幕].Anata.no.Ban.Desu.E01-E20+SP.2019.1080p.WEB-DL.x265.AC3-BitsTV/Anata.no.Ban.Desu.E01.2019.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv", IsVideo: true},
			{ID: 2, Path: "/media/电视剧/轮到你了[全20集][中文字幕].Anata.no.Ban.Desu.E01-E20+SP.2019.1080p.WEB-DL.x265.AC3-BitsTV/Anata.no.Ban.Desu.E02.2019.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv", IsVideo: true},
		},
	})

	episodeGroup := tree.Node("/media/电视剧/轮到你了[全20集][中文字幕].Anata.no.Ban.Desu.E01-E20+SP.2019.1080p.WEB-DL.x265.AC3-BitsTV")
	if episodeGroup.Kind != DirectoryKindEpisodeGroup {
		t.Fatalf("expected episode-count folder to classify as episode group, got %q", episodeGroup.Kind)
	}
}

func TestClassifyTreeRecognizesSingleEpisodePackFolderAsEpisodeGroup(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/电视剧/轮到你了[全20集][中文字幕].Anata.no.Ban.Desu.E01-E20+SP.2019.1080p.WEB-DL.x265.AC3-BitsTV/Anata.no.Ban.Desu.E01.2019.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv", IsVideo: true},
		},
	})

	episodeGroup := tree.Node("/media/电视剧/轮到你了[全20集][中文字幕].Anata.no.Ban.Desu.E01-E20+SP.2019.1080p.WEB-DL.x265.AC3-BitsTV")
	if episodeGroup.Kind != DirectoryKindEpisodeGroup {
		t.Fatalf("expected single-file episode pack folder to classify as episode group, got %q", episodeGroup.Kind)
	}
}

func TestClassifyTreeRecognizesMostlyEpisodicFolderWithNoiseVideo(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/Show/Season 1/Show.S01E01.mkv", IsVideo: true},
			{ID: 2, Path: "/media/Show/Season 1/Show.S01E02.mkv", IsVideo: true},
			{ID: 3, Path: "/media/Show/Season 1/Show.Trailer.mkv", IsVideo: true},
		},
	})

	season := tree.Node("/media/Show/Season 1")
	if season.Kind != DirectoryKindSeason {
		t.Fatalf("expected mostly episodic folder with trailer noise to classify as season, got %q", season.Kind)
	}
}

func TestClassifyTreeDoesNotPromoteEpisodeGroupChildToSeries(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/Anthology/Show.One/Show.One.S01E01.mkv", IsVideo: true},
			{ID: 2, Path: "/media/Anthology/Show.One/Show.One.S01E02.mkv", IsVideo: true},
		},
	})

	anthology := tree.Node("/media/Anthology")
	if anthology.Kind == DirectoryKindSeries {
		t.Fatalf("expected episode group child not to promote parent to series")
	}
}

func TestClassifyTreeRejectsSeasonFolderWithMixedVideoSeasons(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/Show/Season 1/Show.S01E01.mkv", IsVideo: true},
			{ID: 2, Path: "/media/Show/Season 1/Show.S02E01.mkv", IsVideo: true},
		},
	})

	season := tree.Node("/media/Show/Season 1")
	if season.Kind == DirectoryKindSeason {
		t.Fatalf("expected mixed video seasons not to classify as season")
	}
}

func TestClassifyTreeHandlesNilTrees(t *testing.T) {
	if ClassifyTree(nil) != nil {
		t.Fatalf("expected nil tree to stay nil")
	}
	if got := ClassifyTree(&Tree{}); got.Root != nil {
		t.Fatalf("expected empty tree to stay empty")
	}
}

func TestClassifyTreeRecognizesSingleMovieDirectory(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/Movies/Inception (2010)/Inception.2010.1080p.BluRay.x265.mkv", IsVideo: true},
		},
	})

	movie := tree.Node("/media/Movies/Inception (2010)")
	if movie.Kind != DirectoryKindMovie {
		t.Fatalf("expected movie directory, got %q", movie.Kind)
	}
}

func TestClassifyTreeRecognizesMovieVersions(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/Movies/Movie A (2020)/Movie.A.2020.1080p.BluRay.x265.mkv", IsVideo: true},
			{ID: 2, Path: "/media/Movies/Movie A (2020)/Movie.A.2020.2160p.BluRay.x265.mkv", IsVideo: true},
		},
	})

	movie := tree.Node("/media/Movies/Movie A (2020)")
	if movie.Kind != DirectoryKindMovieVersions {
		t.Fatalf("expected movie versions directory, got %q", movie.Kind)
	}
}

func TestClassifyTreeRecognizesMovieVersionsWithEditionNoise(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/Movies/Movie A (2020)/Movie.A.2020.1080p.BluRay.x265.mkv", IsVideo: true},
			{ID: 2, Path: "/media/Movies/Movie A (2020)/Movie.A.Directors.Cut.2020.2160p.REMUX.mkv", IsVideo: true},
		},
	})

	movie := tree.Node("/media/Movies/Movie A (2020)")
	if movie.Kind != DirectoryKindMovieVersions {
		t.Fatalf("expected movie versions directory with edition noise, got %q", movie.Kind)
	}
}

func TestClassifyTreeRecognizesEpisodeWordPatternFolder(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/Anime/Show Name Part 1/Show.Name.EP01.1080p.mkv", IsVideo: true},
			{ID: 2, Path: "/media/Anime/Show Name Part 1/Show.Name.EP02.1080p.mkv", IsVideo: true},
			{ID: 3, Path: "/media/Anime/Show Name Part 1/Show.Name.OVA.1080p.mkv", IsVideo: true},
		},
	})

	season := tree.Node("/media/Anime/Show Name Part 1")
	if season.Kind != DirectoryKindSeason {
		t.Fatalf("expected EP folder to classify as season, got %q", season.Kind)
	}
}

func TestClassifyTreeRecognizesLargeMovieCollectionBySampling(t *testing.T) {
	files := make([]FileInput, 0, 80)
	for idx := 0; idx < 80; idx++ {
		files = append(files, FileInput{
			ID:     uint(idx + 1),
			Path:   fmt.Sprintf("/media/Collections/Set/Movie.%02d.2020.1080p.BluRay.mkv", idx),
			IsVideo: true,
		})
	}
	tree := buildClassifyTree(t, Input{RootPath: "/media", Files: files})

	collection := tree.Node("/media/Collections/Set")
	if collection.Kind != DirectoryKindMovieCollection {
		t.Fatalf("expected large sampled folder to classify as movie collection, got %q", collection.Kind)
	}
}

func TestClassifyTreeRecognizesMovieCollection(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/Collections/SciFi/Movie.A.2020.1080p.BluRay.x265.mkv", IsVideo: true},
			{ID: 2, Path: "/media/Collections/SciFi/Movie.B.2021.1080p.BluRay.x265.mkv", IsVideo: true},
		},
	})

	collection := tree.Node("/media/Collections/SciFi")
	if collection.Kind != DirectoryKindMovieCollection {
		t.Fatalf("expected movie collection directory, got %q", collection.Kind)
	}
}

func TestClassifyTreeDoesNotTreatSingleSxxExxMovieLikeFileAsEpisodeGroup(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/Movies/Odd.Movie/Odd.Movie.S01E01.2020.1080p.BluRay.x265.mkv", IsVideo: true},
		},
	})

	movie := tree.Node("/media/Movies/Odd.Movie")
	if movie.Kind == DirectoryKindEpisodeGroup || movie.Kind == DirectoryKindSeason {
		t.Fatalf("expected single SxxExx-like movie file not to classify as episodic, got %q", movie.Kind)
	}
}

func TestClassifyTreeDoesNotTreatSameTitleDifferentYearsAsCollection(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/Movies/Movie A/Movie.A.2020.1080p.BluRay.x265.mkv", IsVideo: true},
			{ID: 2, Path: "/media/Movies/Movie A/Movie.A.2021.2160p.BluRay.x265.mkv", IsVideo: true},
		},
	})

	movie := tree.Node("/media/Movies/Movie A")
	if movie.Kind == DirectoryKindMovieCollection {
		t.Fatalf("expected same title with different years not to classify as collection")
	}
}

func TestClassifyTreeUsesMovieNFOForWeakMovieFilename(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/Movies/Inception/file.mkv", IsVideo: true},
			{ID: 2, Path: "/media/Movies/Inception/movie.nfo", IsNFO: true, SidecarText: `<movie><title>Inception</title><year>2010</year></movie>`},
		},
	})

	movie := tree.Node("/media/Movies/Inception")
	if movie.Kind != DirectoryKindMovie {
		t.Fatalf("expected movie NFO to classify weak movie filename as movie, got %q", movie.Kind)
	}
}

func TestClassifyTreeUsesEpisodeNFOForWeakSeasonFolder(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/Shows/Six Feet Under/Season 1/file.mkv", IsVideo: true},
			{ID: 2, Path: "/media/Shows/Six Feet Under/Season 1/file.nfo", IsNFO: true, SidecarText: `<episodedetails><season>1</season><episode>1</episode></episodedetails>`},
		},
	})

	season := tree.Node("/media/Shows/Six Feet Under/Season 1")
	if season.Kind != DirectoryKindSeason {
		t.Fatalf("expected episode NFO to classify weak season folder as season, got %q", season.Kind)
	}
}

func TestClassifyTreeKeepsConflictingNFOUnknown(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/Shows/Six Feet Under/Season 1/Six.Feet.Under.S01E01.mkv", IsVideo: true},
			{ID: 2, Path: "/media/Shows/Six Feet Under/Season 1/movie.nfo", IsNFO: true, SidecarText: `<movie><title>Six Feet Under</title><year>2001</year></movie>`},
		},
	})

	season := tree.Node("/media/Shows/Six Feet Under/Season 1")
	if season.Kind != DirectoryKindUnknown {
		t.Fatalf("expected conflicting movie NFO and episode filename evidence to stay unknown, got %q", season.Kind)
	}
}

func TestClassifyTreeKeepsMovieNFOSeasonMarkerConflictUnknown(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/Shows/Six Feet Under/Season 1/file.mkv", IsVideo: true},
			{ID: 2, Path: "/media/Shows/Six Feet Under/Season 1/movie.nfo", IsNFO: true, SidecarText: `<movie><title>Six Feet Under</title><year>2001</year></movie>`},
		},
	})

	season := tree.Node("/media/Shows/Six Feet Under/Season 1")
	if season.Kind != DirectoryKindUnknown {
		t.Fatalf("expected movie NFO and season folder evidence to stay unknown, got %q", season.Kind)
	}
}

func TestClassifyTreeKeepsEpisodeNFOAndMovieFilenameConflictUnknown(t *testing.T) {
	tree := buildClassifyTree(t, Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/Movies/Inception/Inception.2010.1080p.BluRay.x265.mkv", IsVideo: true},
			{ID: 2, Path: "/media/Movies/Inception/movie.nfo", IsNFO: true, SidecarText: `<episodedetails><season>1</season><episode>1</episode></episodedetails>`},
		},
	})

	movie := tree.Node("/media/Movies/Inception")
	if movie.Kind != DirectoryKindUnknown {
		t.Fatalf("expected episode NFO and movie filename evidence to stay unknown, got %q", movie.Kind)
	}
}

func buildClassifyTree(t *testing.T, input Input) *Tree {
	t.Helper()
	tree, err := BuildTree(input)
	if err != nil {
		t.Fatalf("BuildTree returned error: %v", err)
	}
	ClassifyTree(tree)
	return tree
}
