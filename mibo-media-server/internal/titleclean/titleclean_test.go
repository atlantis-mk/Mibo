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
		{name: "multi episode promo tail", raw: "黑袍纠察队.The.Boys.S05E01-02.6v电影 地址发布页 www.6v123.net 收藏不迷路", title: "黑袍纠察队 The Boys"},
		{name: "chinese release watermark and tags", raw: "【高清剧集网发布 www.DDHDTV.com】魔幻手机[全42集][国语配音+中文字幕].Magic.Mobile.Phone.2008.WEB-DL.1080p.H265.AAC-Huawei", title: "魔幻手机 Magic Mobile Phone", year: 2008},
		{name: "fps flac multi audio tail", raw: "飞驰人生3.Pegasus.3.2026.2160p.60fps.FLAC.5Audios", title: "飞驰人生3 Pegasus 3", year: 2026},
		{name: "web bit depth yts watermark", raw: "Peaky.Blinders.The.Immortal.Man.2026.2160p.4K.WEB.x265.10bit.AAC5.1-[YTS.BZ].mkv", title: "Peaky Blinders The Immortal Man", year: 2026},
		{name: "numeric title version and ddp71", raw: "M3GAN.2.0.2025.BluRay.1080p.x265.10bit.DDP7.1.-SSDSSE.mkv", title: "M3GAN 2.0", year: 2025},
		{name: "split h264 codec", raw: "Back to the Past 2025 1080p WEB-DL H 264 AAC-HHWEB.mkv", title: "Back to the Past", year: 2025},
		{name: "short release tag after year", raw: "Avatar.Fire.And.Ash.2025.MA.x264.WEB-DL.1080p-Jaskier.mkv", title: "Avatar Fire And Ash", year: 2025},
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

func TestNormalizeExtractsHashtagTags(t *testing.T) {
	t.Parallel()

	result := Normalize(NormalizeInput{RawTitle: "Movie.Name.#IMAX #国语.2024.1080p"})
	if result.Title != "Movie Name" {
		t.Fatalf("expected hashtag tags removed from title, got %q", result.Title)
	}
	if len(result.Tags) != 2 || result.Tags[0] != "IMAX" || result.Tags[1] != "国语" {
		t.Fatalf("expected extracted hashtag tags, got %#v", result.Tags)
	}
	wantHashtagEvidence := map[string]bool{"#IMAX": false, "#国语": false}
	for _, token := range result.RemovedTokens {
		if token.Reason == "hashtag" {
			wantHashtagEvidence[token.Value] = true
		}
	}
	for token, seen := range wantHashtagEvidence {
		if !seen {
			t.Fatalf("expected hashtag removal evidence for %q in %#v", token, result.RemovedTokens)
		}
	}
}
