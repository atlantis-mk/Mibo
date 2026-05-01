package catalog

import "testing"

func TestBuildMediaItemDTOMovieIncludesMediaSourcesAndStreams(t *testing.T) {
	runtimeSeconds := 8880
	bitRate := int64(8000000)
	width := 1920
	height := 1080
	channels := 6
	detail := CatalogItemDetail{
		ID:             12345,
		Type:           ItemTypeMovie,
		Path:           "/movies/Inception (2010)",
		Title:          "Inception",
		Year:           mediaDTOIntPtr(2010),
		RuntimeSeconds: &runtimeSeconds,
		ExternalIdentities: []CatalogExternalIdentity{{Provider: "tmdb", ProviderType: "movie", ExternalID: "movie:27205"}, {
			Provider: "imdb", ProviderType: "movie", ExternalID: "tt1375666",
		}},
		Assets: []CatalogAssetDetail{{
			ID:           77,
			AssetType:    "main",
			QualityLabel: "1080p - HEVC",
			Files:        []CatalogAssetFileSummary{{StoragePath: "/movies/Inception (2010)/Inception.mkv", Container: "mkv", SizeBytes: 12345678900}},
			Streams: []CatalogMediaStreamSummary{{StreamIndex: 0, StreamType: "video", Codec: "hevc", Width: &width, Height: &height, BitRate: &bitRate, AvgFrameRate: "24000/1001"}, {
				StreamIndex: 1, StreamType: "audio", Codec: "aac", Language: "eng", Channels: &channels, Default: true,
			}},
		}},
	}

	dto := BuildMediaItemDTO(detail)
	if dto.Type != "Movie" || dto.MediaType != "Video" || dto.ProviderIDs["Tmdb"] != "27205" || dto.ProviderIDs["Imdb"] != "tt1375666" {
		t.Fatalf("unexpected movie dto identity fields: %#v", dto)
	}
	if dto.RunTimeTicks == nil || *dto.RunTimeTicks != 88800000000 {
		t.Fatalf("unexpected runtime ticks: %#v", dto.RunTimeTicks)
	}
	if len(dto.MediaSources) != 1 || dto.MediaSources[0].Container != "mkv" || dto.MediaSources[0].Size != 12345678900 {
		t.Fatalf("unexpected media source: %#v", dto.MediaSources)
	}
	if dto.MediaSources[0].DefaultAudioStreamIndex == nil || *dto.MediaSources[0].DefaultAudioStreamIndex != 1 {
		t.Fatalf("expected default audio index 1, got %#v", dto.MediaSources[0].DefaultAudioStreamIndex)
	}
	if len(dto.MediaSources[0].MediaStreams) != 2 || dto.MediaSources[0].MediaStreams[0].AverageFrameRate == nil {
		t.Fatalf("unexpected streams: %#v", dto.MediaSources[0].MediaStreams)
	}
}

func TestBuildMediaItemDTOEpisodeIncludesParentContext(t *testing.T) {
	seasonNumber := 1
	episodeNumber := 2
	detail := CatalogItemDetail{
		ID:    102,
		Type:  ItemTypeEpisode,
		Title: "Pilot",
		EpisodeContext: &CatalogEpisodeParentContext{
			Series:        &CatalogEpisodeSeriesContext{ID: 100, Title: "Breaking Bad"},
			Season:        &CatalogEpisodeSeasonContext{ID: 101, Title: "Season 1"},
			SeasonNumber:  &seasonNumber,
			EpisodeNumber: &episodeNumber,
		},
		Assets: []CatalogAssetDetail{{ID: 7, AssetType: "main", Files: []CatalogAssetFileSummary{{StoragePath: "/tv/Breaking Bad/Season 1/S01E02.mkv", Container: "mkv"}}}},
	}

	dto := BuildMediaItemDTO(detail)
	if dto.Type != "Episode" || dto.SeriesID != "100" || dto.SeasonID != "101" || dto.IndexNumber == nil || *dto.IndexNumber != 2 || dto.ParentIndexNumber == nil || *dto.ParentIndexNumber != 1 {
		t.Fatalf("unexpected episode dto: %#v", dto)
	}
	if dto.Path != "/tv/Breaking Bad/Season 1/S01E02.mkv" {
		t.Fatalf("expected episode path from media source, got %q", dto.Path)
	}
}

func TestGetMediaItemDTOSeasonIncludesSeriesContext(t *testing.T) {
	svc, ctx := newTestService(t)
	series, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Breaking Bad", Path: "/tv/Breaking Bad"})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", Path: "/tv/Breaking Bad/Season 1", IndexNumber: &seasonNumber})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}

	dto, err := svc.GetMediaItemDTO(ctx, season.ID, nil)
	if err != nil {
		t.Fatalf("get season dto: %v", err)
	}
	if dto.Type != "Season" || dto.SeriesID != "1" || dto.SeriesName != "Breaking Bad" || dto.IndexNumber == nil || *dto.IndexNumber != 1 {
		t.Fatalf("unexpected season dto: %#v", dto)
	}
}

func TestBuildMediaItemDTOToleratesSparseStreams(t *testing.T) {
	detail := CatalogItemDetail{ID: 1, Type: ItemTypeMovie, Title: "Sparse", Assets: []CatalogAssetDetail{{ID: 2, Files: []CatalogAssetFileSummary{{StoragePath: "/movies/Sparse.mkv"}}, Streams: []CatalogMediaStreamSummary{{StreamIndex: 0, StreamType: "video"}}}}}
	dto := BuildMediaItemDTO(detail)
	if len(dto.MediaSources) != 1 || len(dto.MediaSources[0].MediaStreams) != 1 || dto.MediaSources[0].MediaStreams[0].Type != "Video" {
		t.Fatalf("unexpected sparse dto: %#v", dto)
	}
}

func mediaDTOIntPtr(value int) *int { return &value }
