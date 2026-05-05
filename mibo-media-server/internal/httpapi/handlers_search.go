package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/library"
)

func (r *Router) handleDiscoverMedia(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	input, err := discoveryInputFromRequest(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	result, err := r.catalog.BrowseItems(req.Context(), catalog.BrowseItemsInput{
		LibraryID:       input.LibraryID,
		Query:           input.Query,
		TypeFilter:      string(input.TypeFilter),
		Genre:           input.Genre,
		Region:          input.Region,
		Year:            input.Year,
		MinRating:       input.MinRating,
		WatchedState:    string(input.Watched),
		OrganizingState: string(input.Organizing),
		Sort:            string(input.Sort),
		SortDirection:   string(input.SortDirection),
		Limit:           input.Limit,
		Offset:          input.Offset,
		UserID:          user.ID,
	})
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	normalizeCatalogListItemsArtworkURLs(req, result.Items)
	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleListSearchHistory(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	items, err := r.search.ListHistory(req.Context(), user.ID, limit)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, items)
}

func discoveryInputFromRequest(req *http.Request) (library.BrowseItemsInput, error) {
	query := req.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))
	var libraryID uint
	if raw := strings.TrimSpace(query.Get("library_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || parsed == 0 {
			return library.BrowseItemsInput{}, fmt.Errorf("library_id must be a positive integer")
		}
		libraryID = uint(parsed)
	}
	input := library.BrowseItemsInput{
		LibraryID:     libraryID,
		Scope:         library.BrowseScopeAll,
		Query:         strings.TrimSpace(query.Get("q")),
		TypeFilter:    normalizeBrowseTypeFilter(query.Get("type")),
		Genre:         strings.TrimSpace(query.Get("genre")),
		Region:        strings.TrimSpace(query.Get("region")),
		Year:          library.ParseBrowseYear(query.Get("year")),
		MinRating:     parseBrowseRating(query.Get("min_rating")),
		Watched:       normalizeWatchedStateFilter(query.Get("watched_state")),
		Organizing:    normalizeOrganizingStateFilter(query.Get("organizing_state")),
		Sort:          normalizeBrowseSort(query.Get("sort")),
		SortDirection: normalizeSortDirection(query.Get("sort_direction")),
		Limit:         limit,
		Offset:        offset,
	}
	if query.Get("scope") == "library" {
		input.Scope = library.BrowseScopeLibrary
	}
	return library.NormalizeBrowseItemsInput(input), nil
}

func normalizeOrganizingStateFilter(value string) library.OrganizingStateFilter {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(library.OrganizingStateFilterOrganized):
		return library.OrganizingStateFilterOrganized
	case string(library.OrganizingStateFilterUnorganized):
		return library.OrganizingStateFilterUnorganized
	default:
		return library.OrganizingStateFilterAll
	}
}
