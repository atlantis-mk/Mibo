package httpapi

import (
	"errors"
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
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	items, err := r.catalog.ListLibraryItems(req.Context(), libraryID, strings.TrimSpace(req.URL.Query().Get("q")), strings.TrimSpace(req.URL.Query().Get("type")), limit)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	normalizeCatalogListItemsArtworkURLs(req, items)
	writeJSON(req.Context(), w, http.StatusOK, items)
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
