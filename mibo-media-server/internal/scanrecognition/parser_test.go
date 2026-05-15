package scanrecognition

import (
	"reflect"
	"testing"
)

func TestParseVideoFilenameExtractsEpisodeSignals(t *testing.T) {
	tests := []struct {
		name string
		path string
		want VideoSignal
	}{
		{
			name: "sxxexx",
			path: "/media/Shows/Six.Feet.Under.S01E01.2001.1080p.WEB-DL.x265.mkv",
			want: VideoSignal{
				TitleCandidates: []string{"Six Feet Under"},
				Year:            intPtr(2001),
				Season:          intPtr(1),
				Episode:         intPtr(1),
				EpisodeNumbers:  []int{1},
				ReleaseTokens:   []string{"1080p", "webdl", "x265"},
			},
		},
		{
			name: "one x episode",
			path: "/media/Shows/Show.Name.2x03.720p.H264.mkv",
			want: VideoSignal{
				TitleCandidates: []string{"Show Name"},
				Season:          intPtr(2),
				Episode:         intPtr(3),
				EpisodeNumbers:  []int{3},
				ReleaseTokens:   []string{"720p", "h264"},
			},
		},
		{
			name: "separated season episode marker",
			path: "/media/Shows/Space.Force.S02.E1.THE.INQUIRY.1080p.Netflix.WEB-DL.H265.DDP5.1-SeeWEB.mp4",
			want: VideoSignal{
				TitleCandidates: []string{"Space Force"},
				Season:          intPtr(2),
				Episode:         intPtr(1),
				EpisodeNumbers:  []int{1},
				ReleaseTokens:   []string{"1080p", "netflix", "webdl", "h265", "ddp5.1"},
			},
		},
		{
			name: "episode marker without season",
			path: "/media/Shows/第01集.mkv",
			want: VideoSignal{
				Episode:        intPtr(1),
				EpisodeNumbers: []int{1},
			},
		},
		{
			name: "bare numeric episode",
			path: "/media/Shows/01.mkv",
			want: VideoSignal{
				Episode:        intPtr(1),
				EpisodeNumbers: []int{1},
			},
		},
		{
			name: "special episode marker",
			path: "/media/电视剧/轮到你了[全20集][中文字幕].Anata.no.Ban.Desu.E01-E20+SP.2019.1080p.WEB-DL.x265.AC3-BitsTV/Anata.no.Ban.Desu.SP.2019.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv",
			want: VideoSignal{
				TitleCandidates: []string{"Anata No Ban Desu"},
				Year:            intPtr(2019),
				IsSpecial:       true,
				ReleaseTokens:   []string{"1080p", "webdl", "x265", "ac3", "bitstv"},
			},
		},
		{
			name: "episode word marker",
			path: "/media/Shows/Show.Name.EP03.1080p.WEB-DL.mkv",
			want: VideoSignal{
				TitleCandidates: []string{"Show Name"},
				Episode:         intPtr(3),
				EpisodeNumbers:  []int{3},
				ReleaseTokens:   []string{"1080p", "webdl"},
			},
		},
		{
			name: "episode marker with hdtv suffix",
			path: "/media/Shows/ViuTV.940920.E01.HDTV.1080i.H264-EntTV.ts",
			want: VideoSignal{
				TitleCandidates: []string{"Viutv 940920"},
				Episode:         intPtr(1),
				EpisodeNumbers:  []int{1},
				ReleaseTokens:   []string{"hdtv", "1080i", "h264"},
			},
		},
		{
			name: "ova special marker",
			path: "/media/Anime/Show.Name.OVA.1080p.BluRay.mkv",
			want: VideoSignal{
				TitleCandidates: []string{"Show Name"},
				IsSpecial:       true,
				ReleaseTokens:   []string{"1080p", "bluray"},
			},
		},
		{
			name: "indexed special marker",
			path: "/media/Shows/Mouse.SP02.The.Predator.1080p.WEB-DL.mkv",
			want: VideoSignal{
				TitleCandidates: []string{"Mouse"},
				IsSpecial:       true,
				SpecialIndex:    intPtr(2),
				ReleaseTokens:   []string{"1080p", "webdl"},
			},
		},
		{
			name: "episode zero is special",
			path: "/media/Shows/Mouse.E00.Restart.Highlight.Clips.1080p.WEB-DL.mkv",
			want: VideoSignal{
				TitleCandidates: []string{"Mouse"},
				IsSpecial:       true,
				ReleaseTokens:   []string{"1080p", "webdl"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseVideoFilename(tt.path)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expected %#v, got %#v", tt.want, got)
			}
		})
	}
}

func TestParseVideoFilenameExtractsMovieSignals(t *testing.T) {
	tests := []struct {
		name string
		path string
		want VideoSignal
	}{
		{
			name: "movie title year and release",
			path: "/media/Movies/Movie.Name.2001.1080p.WEB-DL.x265.mkv",
			want: VideoSignal{
				TitleCandidates: []string{"Movie Name"},
				Year:            intPtr(2001),
				ReleaseTokens:   []string{"1080p", "webdl", "x265"},
			},
		},
		{
			name: "unicode title",
			path: "/media/Movies/六尺之下.2001.1080p.WEB-DL.mkv",
			want: VideoSignal{
				TitleCandidates: []string{"六尺之下"},
				Year:            intPtr(2001),
				ReleaseTokens:   []string{"1080p", "webdl"},
			},
		},
		{
			name: "numeric movie title is not bare episode when year exists",
			path: "/media/Movies/1917.2019.1080p.BluRay.x265.mkv",
			want: VideoSignal{
				TitleCandidates: []string{"1917"},
				Year:            intPtr(2019),
				ReleaseTokens:   []string{"1080p", "bluray", "x265"},
			},
		},
		{
			name: "numeric movie title without release year",
			path: "/media/Movies/1917.1080p.BluRay.x265.mkv",
			want: VideoSignal{
				TitleCandidates: []string{"1917"},
				ReleaseTokens:   []string{"1080p", "bluray", "x265"},
			},
		},
		{
			name: "episode-looking token inside movie title",
			path: "/media/Movies/The.E01.Report.2020.1080p.WEB-DL.mkv",
			want: VideoSignal{
				TitleCandidates: []string{"The E01 Report"},
				Year:            intPtr(2020),
				ReleaseTokens:   []string{"1080p", "webdl"},
			},
		},
		{
			name: "episode-range suffix in folder title is ignored for series extraction",
			path: "/media/电视剧/轮到你了[全20集][中文字幕].Anata.no.Ban.Desu.E01-E20+SP.2019.1080p.WEB-DL.x265.AC3-BitsTV/Anata.no.Ban.Desu.E01.2019.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv",
			want: VideoSignal{
				TitleCandidates: []string{"Anata No Ban Desu"},
				Year:            intPtr(2019),
				Episode:         intPtr(1),
				EpisodeNumbers:  []int{1},
				ReleaseTokens:   []string{"1080p", "webdl", "x265", "ac3", "bitstv"},
			},
		},
		{
			name: "standalone special episode keeps series title",
			path: "/media/电视剧/轮到你了[全20集][中文字幕].Anata.no.Ban.Desu.E01-E20+SP.2019.1080p.WEB-DL.x265.AC3-BitsTV/Anata.no.Ban.Desu.SP.2019.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv",
			want: VideoSignal{
				TitleCandidates: []string{"Anata No Ban Desu"},
				Year:            intPtr(2019),
				IsSpecial:       true,
				ReleaseTokens:   []string{"1080p", "webdl", "x265", "ac3", "bitstv"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseVideoFilename(tt.path)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expected %#v, got %#v", tt.want, got)
			}
		})
	}
}

func intPtr(value int) *int {
	return &value
}
