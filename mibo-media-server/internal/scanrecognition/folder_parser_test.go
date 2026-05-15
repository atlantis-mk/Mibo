package scanrecognition

import (
	"reflect"
	"testing"
)

func TestParseFolderNameExtractsCombinedSeriesSeasonFolder(t *testing.T) {
	got := ParseFolderName("六尺之下 第一季[全13集][中文字幕].Six.Feet.Under.2001.1080p.WEB-DL.x265.AC3-BitsTV")
	want := FolderSignal{
		TitleCandidates:      []string{"六尺之下", "Six Feet Under"},
		Season:               intPtr(1),
		ExpectedEpisodeCount: intPtr(13),
		Year:                 intPtr(2001),
		ReleaseTokens:        []string{"1080p", "webdl", "x265", "ac3", "bitstv"},
		HasSeasonMarker:      true,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestParseFolderNameTrimsEpisodePackSuffixFromSeriesTitle(t *testing.T) {
	got := ParseFolderName("轮到你了[全20集][中文字幕].Anata.no.Ban.Desu.E01-E20+SP.2019.1080p.WEB-DL.x265.AC3-BitsTV")
	want := FolderSignal{
		TitleCandidates:      []string{"轮到你了", "Anata No Ban Desu"},
		ExpectedEpisodeCount: intPtr(20),
		Year:                 intPtr(2019),
		ReleaseTokens:        []string{"1080p", "webdl", "x265", "ac3", "bitstv"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestParseFolderNameExtractsSplitSeasonFolder(t *testing.T) {
	tests := []struct {
		name string
		path string
		want FolderSignal
	}{
		{
			name: "chinese season",
			path: "第一季",
			want: FolderSignal{Season: intPtr(1), HasSeasonMarker: true},
		},
		{
			name: "english season",
			path: "Season 02",
			want: FolderSignal{Season: intPtr(2), HasSeasonMarker: true},
		},
		{
			name: "series title without season",
			path: "Six Feet Under",
			want: FolderSignal{TitleCandidates: []string{"Six Feet Under"}},
		},
		{
			name: "season folder with release metadata",
			path: "Season 02 1080p WEB-DL",
			want: FolderSignal{Season: intPtr(2), ReleaseTokens: []string{"1080p", "webdl"}, HasSeasonMarker: true},
		},
		{
			name: "part season",
			path: "Part 1",
			want: FolderSignal{Season: intPtr(1), HasSeasonMarker: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseFolderName(tt.path)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expected %#v, got %#v", tt.want, got)
			}
		})
	}
}

func TestParseFolderNamePreservesNumericTitleWithoutReleaseYear(t *testing.T) {
	got := ParseFolderName("1917.1080p.BluRay.x265")
	want := FolderSignal{
		TitleCandidates: []string{"1917"},
		ReleaseTokens:   []string{"1080p", "bluray", "x265"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}
