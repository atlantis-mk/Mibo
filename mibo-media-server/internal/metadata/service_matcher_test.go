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
	for _, want := range []string{"Some Movie", "[www.example.com]Some.Movie.2023.HD1080P", "Some Movie (2023)"} {
		if !seen[want] {
			t.Fatalf("expected fallback query %q in %#v", want, queries)
		}
	}
}
