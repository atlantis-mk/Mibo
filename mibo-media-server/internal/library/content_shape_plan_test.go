package library

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/storage"
)

func TestCompileContentShapePlanEpisodeShapes(t *testing.T) {
	t.Parallel()

	mixed := buildContentShapeDirectoryProfile("auto", "/library", scanDirectorySnapshot{Path: "/library/Show", Objects: []storage.Object{{Path: "/library/Show/01.mkv"}, {Path: "/library/Show/第002集.mkv"}, {Path: "/library/Show/S01E003.mkv"}, {Path: "/library/Show/004.2160p.mkv"}}}, newFilenameTokenProfileCache())
	plan := compileContentShapePlan(mixed)
	if plan.Shape != contentShapeAbsoluteEpisodePack && plan.Shape != contentShapeEpisodePack && plan.Shape != contentShapeFlatEpisodeFolder {
		t.Fatalf("expected episode plan for mixed naming, got %#v", plan)
	}

	season := 1
	seasonProfile := buildContentShapeDirectoryProfile("auto", "/library", scanDirectorySnapshot{Path: "/library/Show/Season 1", Objects: []storage.Object{{Path: "/library/Show/Season 1/Show.S01E01.mkv"}, {Path: "/library/Show/Season 1/Show.S01E02.mkv"}}}, newFilenameTokenProfileCache())
	seasonPlan := compileContentShapePlan(seasonProfile)
	if seasonPlan.Shape != contentShapeSeasonFolder || seasonPlan.SeasonNumber == nil || *seasonPlan.SeasonNumber != season {
		t.Fatalf("expected season folder plan, got %#v", seasonPlan)
	}

	absolute := buildContentShapeDirectoryProfile("auto", "/library", largeEpisodeShapeSnapshot("/library/Absolute", 1000), newFilenameTokenProfileCache())
	absolutePlan := compileContentShapePlan(absolute)
	if absolutePlan.Shape != contentShapeAbsoluteEpisodePack || absolutePlan.NumberingMode != "absolute" {
		t.Fatalf("expected absolute pack plan, got %#v", absolutePlan)
	}
}

func TestCompileContentShapePlanMovieShapesAndAmbiguity(t *testing.T) {
	t.Parallel()

	versions := buildContentShapeDirectoryProfile("auto", "/library", scanDirectorySnapshot{Path: "/library/Movie", Objects: []storage.Object{{Path: "/library/Movie/Movie.2024.1080p.mkv"}, {Path: "/library/Movie/Movie.2024.2160p.Directors.Cut.mkv"}}}, newFilenameTokenProfileCache())
	versionPlan := compileContentShapePlan(versions)
	if versionPlan.Shape != contentShapeMovieVersionsFolder || versionPlan.MovieWorkKey == "" {
		t.Fatalf("expected movie versions plan, got %#v", versionPlan)
	}

	collection := buildContentShapeDirectoryProfile("auto", "/library", scanDirectorySnapshot{Path: "/library/Movies", Objects: []storage.Object{{Path: "/library/Movies/Alien.1979.mkv"}, {Path: "/library/Movies/Aliens.1986.mkv"}, {Path: "/library/Movies/Heat.1995.mkv"}}}, newFilenameTokenProfileCache())
	collectionPlan := compileContentShapePlan(collection)
	if collectionPlan.Shape != contentShapeMovieCollection {
		t.Fatalf("expected movie collection plan, got %#v", collectionPlan)
	}

	conflictingNonEpisode := contentShapeDirectoryProfile{VideoCount: 10, NonExtraVideoCount: 10, ExplicitEpisodeCount: 4, LeadingNumericCount: 4, SequenceCoverage: 0.7, YearDensity: 0.7, TitleUniqueness: 0.7, TitleYearCount: 7}
	conflictingNonEpisodePlan := compileContentShapePlan(conflictingNonEpisode)
	if conflictingNonEpisodePlan.Shape != contentShapeUnknownReview || conflictingNonEpisodePlan.ReviewState != "review_required" {
		t.Fatalf("expected conflicting non-episode directory to require review, got %#v", conflictingNonEpisodePlan)
	}

	nonEpisode := contentShapeDirectoryProfile{VideoCount: 10, NonExtraVideoCount: 10, SequenceCoverage: 0.1, YearDensity: 0.1, TitleUniqueness: 0.9}
	nonEpisodePlan := compileContentShapePlan(nonEpisode)
	if nonEpisodePlan.Shape != contentShapeUnknownReview || nonEpisodePlan.ReviewState != "review_required" {
		t.Fatalf("expected weak non-episode directory to require review, got %#v", nonEpisodePlan)
	}
}
