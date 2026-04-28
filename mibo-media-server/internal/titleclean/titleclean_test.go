package titleclean

import "testing"

func TestNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		raw   string
		title string
		year  int
	}{
		{name: "movie", raw: "Movie.Name.2024.1080p.WEB-DL.x265-GROUP", title: "Movie Name", year: 2024},
		{name: "tv", raw: "Show.Name.S01E02.1080p.NF.WEB-DL.DDP5.1.Atmos.x264-GROUP", title: "Show Name S01E02"},
		{name: "chinese", raw: "灵笼 第二季[www.4KHDR.CN]Incarnation.S02E01.2025.2160p.WEB-DL.H264.AAC-4KHDR世界", title: "灵笼 第二季 Incarnation S02E01", year: 2025},
		{name: "url watermark", raw: "www.example.com.Some.Movie.2023.HD1080P", title: "Some Movie", year: 2023},
		{name: "bracketed watermark", raw: "[www.example.com]Some.Movie.2023.HD1080P", title: "Some Movie", year: 2023},
		{name: "dense technical release", raw: "Dune.Part.Two.2024.2160p.UHD.BluRay.REMUX.HEVC.TrueHD.Atmos-GROUP", title: "Dune Part Two", year: 2024},
		{name: "release group", raw: "Some.Movie.2024-GROUP", title: "Some Movie", year: 2024},
		{name: "empty fallback", raw: "2024.2160p.WEB-DL.x265", title: "2024.2160p.WEB-DL.x265", year: 2024},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := Normalize(NormalizeInput{RawTitle: tt.raw})
			if result.Title != tt.title {
				t.Fatalf("expected title %q, got %q", tt.title, result.Title)
			}
			if result.NormalizationVersion != NormalizationVersion {
				t.Fatalf("expected normalization version %q, got %q", NormalizationVersion, result.NormalizationVersion)
			}
			if tt.year == 0 {
				if result.Year != nil {
					t.Fatalf("expected no year, got %#v", result.Year)
				}
				return
			}
			if result.Year == nil || *result.Year != tt.year {
				t.Fatalf("expected year %d, got %#v", tt.year, result.Year)
			}
		})
	}
}

func TestNormalizeRecordsRemovedTokenReasons(t *testing.T) {
	t.Parallel()

	result := Normalize(NormalizeInput{RawTitle: "[www.example.com]Movie.Name.2024.2160p.WEB-DL.x265-GROUP"})
	wantReasons := map[string]bool{"website": false, "year": false, "quality": false, "source": false, "video_codec": false, "release_group": false}
	for _, token := range result.RemovedTokens {
		if _, ok := wantReasons[token.Reason]; ok {
			wantReasons[token.Reason] = true
		}
	}
	for reason, seen := range wantReasons {
		if !seen {
			t.Fatalf("expected removed token reason %q in %#v", reason, result.RemovedTokens)
		}
	}
}
