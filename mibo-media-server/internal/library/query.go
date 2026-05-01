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
	BrowseTypeFilterAll     BrowseTypeFilter = "all"
	BrowseTypeFilterMovie   BrowseTypeFilter = "movie"
	BrowseTypeFilterShow    BrowseTypeFilter = "show"
	BrowseTypeFilterEpisode BrowseTypeFilter = "episode"
)

type SortDirection string

const (
	SortDirectionAsc  SortDirection = "asc"
	SortDirectionDesc SortDirection = "desc"
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

type BrowseItemsInput struct {
	LibraryID     uint
	Scope         BrowseScope
	Query         string
	TypeFilter    BrowseTypeFilter
	Genre         string
	Region        string
	Year          *int
	MinRating     *float64
	Watched       WatchedStateFilter
	Sort          BrowseSort
	SortDirection SortDirection
	Limit         int
	Offset        int
}

type PersonDetail struct {
	Name         string `json:"name"`
	Role         string `json:"role"`
	AvatarURL    string `json:"avatar_url"`
	TMDBPersonID *int   `json:"tmdb_person_id,omitempty"`
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
	CatalogItemsCount   int64               `json:"catalog_items_count"`
	InventoryFilesCount int64               `json:"inventory_files_count"`
	Paths               []LibraryPathView   `json:"paths"`
	Policies            LibraryPoliciesView `json:"policies"`
}

type TrackDetail struct {
	Codec    string `json:"codec"`
	Language string `json:"language"`
	Title    string `json:"title"`
	Channels int    `json:"channels,omitempty"`
}

func NormalizeBrowseItemsInput(input BrowseItemsInput) BrowseItemsInput {
	if input.Scope != BrowseScopeAll {
		input.Scope = BrowseScopeLibrary
	}
	switch input.TypeFilter {
	case BrowseTypeFilterMovie, BrowseTypeFilterShow, BrowseTypeFilterEpisode:
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
	if input.Offset < 0 {
		input.Offset = 0
	}
	switch input.SortDirection {
	case SortDirectionAsc, SortDirectionDesc:
	default:
		input.SortDirection = SortDirectionDesc
		if input.Sort == BrowseSortTitle {
			input.SortDirection = SortDirectionAsc
		}
	}
	return input
}
