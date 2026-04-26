package library

import (
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

type BrowseScope string

const (
	BrowseScopeLibrary BrowseScope = "library"
	BrowseScopeAll     BrowseScope = "all"
)

type BrowseTypeFilter string

const (
	BrowseTypeFilterAll   BrowseTypeFilter = "all"
	BrowseTypeFilterMovie BrowseTypeFilter = "movie"
	BrowseTypeFilterShow  BrowseTypeFilter = "show"
)

type BrowseSort string

const (
	BrowseSortRecent      BrowseSort = "recent"
	BrowseSortTitle       BrowseSort = "title"
	BrowseSortYear        BrowseSort = "year"
	BrowseSortWatchStatus BrowseSort = "watch_status"
)

type WatchedStateFilter string

const (
	WatchedStateFilterAll        WatchedStateFilter = "all"
	WatchedStateFilterUnwatched  WatchedStateFilter = "unwatched"
	WatchedStateFilterInProgress WatchedStateFilter = "in_progress"
	WatchedStateFilterWatched    WatchedStateFilter = "watched"
)

type BrowseMediaItemsInput struct {
	LibraryID  uint
	Scope      BrowseScope
	Query      string
	TypeFilter BrowseTypeFilter
	Genre      string
	Region     string
	Year       *int
	MinRating  *float64
	Watched    WatchedStateFilter
	Sort       BrowseSort
	Limit      int
}

type browseCandidate struct {
	Item      database.MediaItem
	WatchRank int
}

type showDiscoveryGroup struct {
	Anchor         database.MediaItem
	Display        database.MediaItem
	WatchRank      int
	Representative int
}

type DiscoveryItem struct {
	Item         database.MediaItem `json:"item"`
	WatchedState string             `json:"watched_state"`
}

type PersonDetail struct {
	Name      string `json:"name"`
	Role      string `json:"role"`
	AvatarURL string `json:"avatar_url"`
}

type TrailerDetail struct {
	Provider  string `json:"provider"`
	Site      string `json:"site"`
	Key       string `json:"key"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Official  bool   `json:"official"`
	Language  string `json:"language"`
	WatchURL  string `json:"watch_url"`
	EmbedURL  string `json:"embed_url"`
	Thumbnail string `json:"thumbnail"`
}

type LibraryDetail struct {
	database.Library
	MediaItemsCount int64 `json:"media_items_count"`
	MediaFilesCount int64 `json:"media_files_count"`
}

type MediaItemDetail struct {
	database.MediaItem
	SeriesTMDBID        *int              `json:"series_tmdb_id,omitempty"`
	SeriesTitleDisplay  string            `json:"series_title_display"`
	DefaultSeasonNumber *int              `json:"default_season_number,omitempty"`
	Genres              []string          `json:"genres"`
	Cast                []PersonDetail    `json:"cast"`
	Directors           []PersonDetail    `json:"directors"`
	Trailer             *TrailerDetail    `json:"trailer,omitempty"`
	Files               []MediaFileDetail `json:"files"`
}

type SeriesSeasonDetail struct {
	SeasonNumber   int                   `json:"season_number"`
	Name           string                `json:"name"`
	Overview       string                `json:"overview"`
	PosterURL      string                `json:"poster_url"`
	RuntimeSeconds *int                  `json:"runtime_seconds,omitempty"`
	Episodes       []SeriesEpisodeDetail `json:"episodes"`
}

type SeriesEpisodeDetail struct {
	MediaItemID    uint   `json:"media_item_id"`
	SeasonNumber   int    `json:"season_number"`
	EpisodeNumber  int    `json:"episode_number"`
	Name           string `json:"name"`
	AirDate        string `json:"air_date,omitempty"`
	Overview       string `json:"overview"`
	StillURL       string `json:"still_url"`
	RuntimeSeconds *int   `json:"runtime_seconds,omitempty"`
}

type MediaFileDetail struct {
	database.MediaFile
	AudioTracks    []TrackDetail `json:"audio_tracks"`
	SubtitleTracks []TrackDetail `json:"subtitle_tracks"`
}

type TrackDetail struct {
	Codec    string `json:"codec"`
	Language string `json:"language"`
	Title    string `json:"title"`
	Channels int    `json:"channels,omitempty"`
}

type LatestByLibrarySection struct {
	LibraryID   uint                 `json:"library_id"`
	LibraryName string               `json:"library_name"`
	Items       []database.MediaItem `json:"items"`
}

func NormalizeBrowseMediaItemsInput(input BrowseMediaItemsInput) BrowseMediaItemsInput {
	if input.Scope != BrowseScopeAll {
		input.Scope = BrowseScopeLibrary
	}
	switch input.TypeFilter {
	case BrowseTypeFilterMovie, BrowseTypeFilterShow:
	default:
		input.TypeFilter = BrowseTypeFilterAll
	}
	switch input.Sort {
	case BrowseSortTitle, BrowseSortYear, BrowseSortWatchStatus:
	default:
		input.Sort = BrowseSortRecent
	}
	switch input.Watched {
	case WatchedStateFilterUnwatched, WatchedStateFilterInProgress, WatchedStateFilterWatched:
	default:
		input.Watched = WatchedStateFilterAll
	}
	if input.Year != nil && *input.Year <= 0 {
		input.Year = nil
	}
	if input.MinRating != nil {
		if *input.MinRating < 0 {
			zero := 0.0
			input.MinRating = &zero
		}
		if *input.MinRating > 10 {
			input.MinRating = nil
		}
	}
	input.Query = strings.TrimSpace(input.Query)
	input.Genre = strings.TrimSpace(input.Genre)
	input.Region = strings.TrimSpace(input.Region)
	if input.Limit < 0 || input.Limit > 200 {
		input.Limit = 50
	}
	return input
}
