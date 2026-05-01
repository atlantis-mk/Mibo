package metadata

import "testing"

func TestCleanSearchTitleMatchesScannerNormalization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "release noise", raw: "Movie.Name.2024.1080p.WEB-DL.x265-GROUP", want: "Movie Name"},
		{name: "website watermark", raw: "[www.example.com]Some.Movie.2023.HD1080P", want: "Some Movie"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := cleanSearchTitle(tt.raw); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestBuildSearchQueriesKeepsFallbackVariants(t *testing.T) {
	t.Parallel()

	queries := buildSearchQueries(metadataSearchItem{
		Title:         "Some Movie",
		OriginalTitle: "[www.example.com]Some.Movie.2023.HD1080P",
		SourcePath:    "/movies/Some Movie (2023)/[www.example.com]Some.Movie.2023.HD1080P.mkv",
	}, "movie")

	seen := make(map[string]bool, len(queries))
	for _, query := range queries {
		seen[query.Value] = true
	}
	for _, want := range []string{"[www.example.com]Some.Movie.2023.HD1080P", "Some Movie", "Some Movie (2023)"} {
		if !seen[want] {
			t.Fatalf("expected fallback query %q in %#v", want, queries)
		}
	}
}

func TestBuildSearchQueriesUsesSourceFilenameForMovie(t *testing.T) {
	t.Parallel()

	queries := buildSearchQueries(metadataSearchItem{
		Title:      "我的世界大电影",
		SourcePath: "/movies/www.UIndex.org - Back to the Past 2025 1080p WEB-DL H 264 AAC-HHWEB.mkv",
	}, "movie")

	seen := make(map[string]bool, len(queries))
	for _, query := range queries {
		seen[query.Value] = true
		if query.Value == "我的世界大电影" || query.Value == "电影" {
			t.Fatalf("expected polluted/generic title to be excluded, got %#v", queries)
		}
	}
	if !seen["Back to the Past"] {
		t.Fatalf("expected cleaned filename query, got %#v", queries)
	}
}

func TestBuildSearchQueriesKeepsCatalogTitleWhenItOverlapsFilename(t *testing.T) {
	t.Parallel()

	queries := buildSearchQueries(metadataSearchItem{
		Title:      "Auto MetaTube Movie",
		SourcePath: "/movies/auto-metatube.mkv",
	}, "movie")

	seen := make(map[string]bool, len(queries))
	for _, query := range queries {
		seen[query.Value] = true
	}
	if !seen["Auto MetaTube Movie"] {
		t.Fatalf("expected overlapping catalog title to be retained, got %#v", queries)
	}
}

func TestBuildSearchQueriesSkipsGenericFolderQuery(t *testing.T) {
	t.Parallel()

	queries := buildQueryVariants([]string{"电影", "movie", "Real Title"}, nil)
	if len(queries) != 1 || queries[0].Value != "Real Title" {
		t.Fatalf("expected only concrete query, got %#v", queries)
	}
}

func TestAutomatedMatchRejectsWeakTitleCandidate(t *testing.T) {
	t.Parallel()

	query := matchSearchQuery{Value: "Back to the Past"}
	candidate := scoreMatchCandidate(metadataSearchItem{Title: "Back to the Past"}, "movie", query, searchResult{ID: 1, Title: "A Minecraft Movie", ReleaseDate: "2025-04-04", VoteCount: 1000})
	normalized := NormalizedMetadataCandidate{Title: candidate.result.Title, OriginalTitle: candidate.result.OriginalTitle, Confidence: candidate.confidence, MatchedQuery: candidate.matchedQuery, ReasonSummary: candidate.reasonSummary}
	if acceptableAutomatedMatchCandidate(normalized) {
		t.Fatalf("expected weak title candidate to be rejected, got confidence %.2f with reason %q", normalized.Confidence, normalized.ReasonSummary)
	}
}

func TestCompareNormalizedTitlesTreatsSpacingOnlyDifferenceAsStrong(t *testing.T) {
	t.Parallel()

	if score := compareNormalizedTitles("moviea", "movie a"); score < 0.85 {
		t.Fatalf("expected spacing-only title difference to score strongly, got %.2f", score)
	}
}

func TestAutomatedMatchAcceptsPartialTitleCandidate(t *testing.T) {
	t.Parallel()

	candidate := NormalizedMetadataCandidate{Confidence: 0.72, ReasonSummary: "标题部分匹配，年份完全一致"}
	if !acceptableAutomatedMatchCandidate(candidate) {
		t.Fatalf("expected partial title candidate to be accepted")
	}
}

func TestAutomatedMatchKeepsSingleTokenWeakCandidateForReview(t *testing.T) {
	t.Parallel()

	candidate := NormalizedMetadataCandidate{Title: "Matched Movie", Confidence: 0.21, MatchedQuery: "MovieA", ReasonSummary: "标题弱匹配（query: MovieA)"}
	if !acceptableAutomatedMatchCandidate(candidate) {
		t.Fatalf("expected single-token weak title candidate to remain reviewable")
	}
}
