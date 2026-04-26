package httpapi

import (
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
	if r.shouldServeCatalogLibraryItems(req.Context()) && r.catalog != nil {
		var items []catalog.CatalogListItem
		if strings.TrimSpace(input.Query) == "" {
			items, err = r.catalog.ListItems(req.Context(), input.LibraryID, "", string(input.TypeFilter), input.Limit)
		} else {
			items, err = r.catalog.SearchItems(req.Context(), input.LibraryID, input.Query, string(input.TypeFilter), input.Limit)
		}
		if err != nil {
			writeError(req.Context(), w, http.StatusInternalServerError, err)
			return
		}
		normalizeCatalogListItemsArtworkURLs(req, items)
		writeJSON(req.Context(), w, http.StatusOK, map[string]any{"items": items})
		return
	}
	if strings.TrimSpace(input.Query) == "" {
		items, err := r.library.DiscoverMediaItems(req.Context(), &user.ID, input)
		if err != nil {
			writeError(req.Context(), w, http.StatusInternalServerError, err)
			return
		}
		normalizeDiscoveryItemsArtworkURLs(req, items)
		writeJSON(req.Context(), w, http.StatusOK, map[string]any{"items": items})
		return
	}
	results, err := r.search.Search(req.Context(), user.ID, input)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	normalizeSearchResultsArtworkURLs(req, results)
	writeJSON(req.Context(), w, http.StatusOK, map[string]any{"items": results})
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

func discoveryInputFromRequest(req *http.Request) (library.BrowseMediaItemsInput, error) {
	query := req.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	var libraryID uint
	if raw := strings.TrimSpace(query.Get("library_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || parsed == 0 {
			return library.BrowseMediaItemsInput{}, fmt.Errorf("library_id must be a positive integer")
		}
		libraryID = uint(parsed)
	}
	input := library.BrowseMediaItemsInput{
		LibraryID:  libraryID,
		Scope:      library.BrowseScopeAll,
		Query:      strings.TrimSpace(query.Get("q")),
		TypeFilter: normalizeBrowseTypeFilter(query.Get("type")),
		Genre:      strings.TrimSpace(query.Get("genre")),
		Region:     strings.TrimSpace(query.Get("region")),
		Year:       library.ParseBrowseYear(query.Get("year")),
		MinRating:  parseBrowseRating(query.Get("min_rating")),
		Watched:    normalizeWatchedStateFilter(query.Get("watched_state")),
		Sort:       normalizeBrowseSort(query.Get("sort")),
		Limit:      limit,
	}
	if query.Get("scope") == "library" {
		input.Scope = library.BrowseScopeLibrary
	}
	return library.NormalizeBrowseMediaItemsInput(input), nil
}
