package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/library"
)

func (r *Router) handleCreateLibrary(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	var input library.CreateLibraryInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	libraryRecord, job, err := r.library.CreateLibrary(req.Context(), input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusCreated, map[string]any{
		"library": libraryRecord,
		"job":     job,
	})
}

func (r *Router) handleListLibraries(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	libraries, err := r.library.ListLibraries(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, libraries)
}

func (r *Router) handleGetLibrary(w http.ResponseWriter, req *http.Request) {
	libraryID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	record, err := r.library.GetLibrary(req.Context(), libraryID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, record)
}

func (r *Router) handleDeleteLibrary(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	libraryID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	if err := r.library.DeleteLibrary(req.Context(), libraryID); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, map[string]any{
		"id":     libraryID,
		"status": "deleted",
		"type":   "library",
	})
}

func (r *Router) handleQueueLibraryScan(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	libraryID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	job, err := r.library.QueueLibraryScan(req.Context(), libraryID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusAccepted, job)
}

func (r *Router) handleListLibraryItems(w http.ResponseWriter, req *http.Request) {
	libraryID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if r.shouldServeCatalogLibraryItems(req.Context()) {
		limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
		items, err := r.catalog.ListLibraryItems(req.Context(), libraryID, strings.TrimSpace(req.URL.Query().Get("q")), strings.TrimSpace(req.URL.Query().Get("type")), limit)
		if err != nil {
			writeError(req.Context(), w, http.StatusInternalServerError, err)
			return
		}
		normalizeCatalogListItemsArtworkURLs(req, items)
		writeJSON(req.Context(), w, http.StatusOK, items)
		return
	}
	user, _ := r.optionalUser(req)
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	browseInput := library.BrowseMediaItemsInput{
		LibraryID:  libraryID,
		Scope:      library.BrowseScopeLibrary,
		Query:      strings.TrimSpace(req.URL.Query().Get("q")),
		TypeFilter: normalizeBrowseTypeFilter(req.URL.Query().Get("type")),
		Genre:      strings.TrimSpace(req.URL.Query().Get("genre")),
		Region:     strings.TrimSpace(req.URL.Query().Get("region")),
		Year:       library.ParseBrowseYear(req.URL.Query().Get("year")),
		MinRating:  parseBrowseRating(req.URL.Query().Get("min_rating")),
		Watched:    normalizeWatchedStateFilter(req.URL.Query().Get("watched_state")),
		Sort:       normalizeBrowseSort(req.URL.Query().Get("sort")),
		Limit:      limit,
	}

	var userID *uint
	if user != nil {
		userID = &user.ID
	}
	items, err := r.library.DiscoverMediaItems(req.Context(), userID, browseInput)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	out := make([]library.DiscoveryItem, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	legacy := make([]any, 0, len(out))
	for _, item := range out {
		mediaItem := item.Item
		normalizeMediaItemArtworkURLs(req, &mediaItem)
		legacy = append(legacy, mediaItem)
	}
	writeJSON(req.Context(), w, http.StatusOK, legacy)
}

func (r *Router) handleGetMediaItem(w http.ResponseWriter, req *http.Request) {
	if r.rejectLegacyMediaEndpoint(req, w, "/api/v1/items/"+req.PathValue("id")) {
		return
	}
	mediaItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	item, err := r.library.GetMediaItem(req.Context(), mediaItemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	normalizeMediaItemDetailArtworkURLs(req, &item)

	writeJSON(req.Context(), w, http.StatusOK, item)
}

func normalizeBrowseTypeFilter(value string) library.BrowseTypeFilter {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(library.BrowseTypeFilterMovie):
		return library.BrowseTypeFilterMovie
	case string(library.BrowseTypeFilterShow):
		return library.BrowseTypeFilterShow
	default:
		return library.BrowseTypeFilterAll
	}
}

func normalizeBrowseSort(value string) library.BrowseSort {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(library.BrowseSortTitle):
		return library.BrowseSortTitle
	case string(library.BrowseSortYear):
		return library.BrowseSortYear
	case string(library.BrowseSortWatchStatus):
		return library.BrowseSortWatchStatus
	default:
		return library.BrowseSortRecent
	}
}

func normalizeWatchedStateFilter(value string) library.WatchedStateFilter {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(library.WatchedStateFilterUnwatched):
		return library.WatchedStateFilterUnwatched
	case string(library.WatchedStateFilterInProgress):
		return library.WatchedStateFilterInProgress
	case string(library.WatchedStateFilterWatched):
		return library.WatchedStateFilterWatched
	default:
		return library.WatchedStateFilterAll
	}
}

func parseBrowseRating(value string) *float64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return nil
	}
	return &parsed
}
