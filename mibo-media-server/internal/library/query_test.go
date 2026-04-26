package library

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestGetMediaItemSupportsLegacyPeopleJSON(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	ctx := context.Background()
	item := database.MediaItem{
		LibraryID:     1,
		Type:          "movie",
		Title:         "MovieA",
		SourcePath:    "/movies/MovieA.2024.mkv",
		MatchStatus:   "matched",
		Status:        "ready",
		CastJSON:      `["Actor A"]`,
		DirectorsJSON: `["Director A"]`,
	}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	svc := NewService(config.Config{}, db, nil, nil)
	detail, err := svc.GetMediaItem(ctx, item.ID)
	if err != nil {
		t.Fatalf("get media item: %v", err)
	}

	if len(detail.Cast) != 1 || detail.Cast[0].Name != "Actor A" || detail.Cast[0].AvatarURL != "" {
		t.Fatalf("unexpected cast: %#v", detail.Cast)
	}
	if len(detail.Directors) != 1 || detail.Directors[0].Name != "Director A" || detail.Directors[0].Role != "" {
		t.Fatalf("unexpected directors: %#v", detail.Directors)
	}
}

func TestGetMediaItemParsesTrailerDetail(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	ctx := context.Background()
	item := database.MediaItem{
		LibraryID:   1,
		Type:        "movie",
		Title:       "MovieA",
		SourcePath:  "/movies/MovieA.2024.mkv",
		MatchStatus: "matched",
		Status:      "ready",
		TrailerJSON: `{"provider":"tmdb","site":"YouTube","key":"abc123","name":"Official Trailer","type":"Trailer","official":true,"language":"en","watch_url":"https://www.youtube.com/watch?v=abc123","embed_url":"https://www.youtube.com/embed/abc123","thumbnail":"https://img.youtube.com/vi/abc123/hqdefault.jpg"}`,
	}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	svc := NewService(config.Config{}, db, nil, nil)
	detail, err := svc.GetMediaItem(ctx, item.ID)
	if err != nil {
		t.Fatalf("get media item: %v", err)
	}

	if detail.Trailer == nil || detail.Trailer.Key != "abc123" || detail.Trailer.EmbedURL == "" {
		t.Fatalf("unexpected trailer: %#v", detail.Trailer)
	}
}

func TestGetMediaItemReturnsEmptyCollectionsForBlankJSON(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	ctx := context.Background()
	item := database.MediaItem{
		LibraryID:   1,
		Type:        "movie",
		Title:       "MovieA",
		SourcePath:  "/movies/MovieA.2024.mkv",
		MatchStatus: "matched",
		Status:      "ready",
	}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	svc := NewService(config.Config{}, db, nil, nil)
	detail, err := svc.GetMediaItem(ctx, item.ID)
	if err != nil {
		t.Fatalf("get media item: %v", err)
	}

	if detail.Genres == nil || len(detail.Genres) != 0 {
		t.Fatalf("expected empty genres slice, got %#v", detail.Genres)
	}
	if detail.Cast == nil || len(detail.Cast) != 0 {
		t.Fatalf("expected empty cast slice, got %#v", detail.Cast)
	}
	if detail.Directors == nil || len(detail.Directors) != 0 {
		t.Fatalf("expected empty directors slice, got %#v", detail.Directors)
	}
	if detail.Files == nil || len(detail.Files) != 0 {
		t.Fatalf("expected empty files slice, got %#v", detail.Files)
	}
}

func TestListSeriesEpisodesFallsBackToLocalScanData(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	ctx := context.Background()
	items := []database.MediaItem{
		{
			LibraryID:      7,
			Type:           "episode",
			Title:          "启程",
			SeriesTitle:    "灵笼 第一季",
			SeasonNumber:   intPtr(1),
			EpisodeNumber:  intPtr(1),
			PosterURL:      "/poster-s1.jpg",
			BackdropURL:    "/still-s1e1.jpg",
			RuntimeSeconds: intPtr(1500),
			SourcePath:     "/shows/ling-cage-s01e01.mkv",
			Status:         "ready",
		},
		{
			LibraryID:     7,
			Type:          "episode",
			Title:         "重逢",
			SeriesTitle:   "灵笼 第二季",
			SeasonNumber:  intPtr(2),
			EpisodeNumber: intPtr(1),
			SourcePath:    "/shows/ling-cage-s02e01.mkv",
			Status:        "ready",
		},
	}
	for _, item := range items {
		if err := db.WithContext(ctx).Create(&item).Error; err != nil {
			t.Fatalf("create item: %v", err)
		}
	}

	svc := NewService(config.Config{}, db, nil, nil)
	seasons, err := svc.ListSeriesEpisodes(ctx, 1)
	if err != nil {
		t.Fatalf("list series episodes: %v", err)
	}

	if len(seasons) != 1 {
		t.Fatalf("expected only the anchored season, got %#v", seasons)
	}
	if seasons[0].SeasonNumber != 1 || seasons[0].Name != "第 1 季" {
		t.Fatalf("unexpected first season: %#v", seasons[0])
	}
	if len(seasons[0].Episodes) != 1 || seasons[0].Episodes[0].Name != "启程" {
		t.Fatalf("unexpected first season episodes: %#v", seasons[0].Episodes)
	}
	if seasons[0].Episodes[0].StillURL != "/still-s1e1.jpg" {
		t.Fatalf("expected backdrop fallback, got %#v", seasons[0].Episodes[0])
	}
}
