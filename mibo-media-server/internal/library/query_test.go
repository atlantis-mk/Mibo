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
