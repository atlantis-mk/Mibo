package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/metadata"
)

func (r *Router) handleListTVSeasons(w http.ResponseWriter, req *http.Request) {
	if r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("metadata service unavailable"))
		return
	}

	tmdbID, err := parseIntPathValue(req, "tmdb_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	seasons, err := r.metadata.ListTVSeasons(req.Context(), tmdbID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, seasons)
}

func (r *Router) handleListTVSeasonEpisodes(w http.ResponseWriter, req *http.Request) {
	if r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("metadata service unavailable"))
		return
	}

	tmdbID, err := parseIntPathValue(req, "tmdb_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	seasonNumber, err := parseIntPathValue(req, "n")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	var libraryID *uint
	if rawLibraryID := strings.TrimSpace(req.URL.Query().Get("library_id")); rawLibraryID != "" {
		parsedLibraryID, err := strconv.ParseUint(rawLibraryID, 10, 64)
		if err != nil || parsedLibraryID == 0 {
			writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("library_id must be a positive integer"))
			return
		}
		value := uint(parsedLibraryID)
		libraryID = &value
	}

	episodes, err := r.metadata.ListSeasonEpisodes(req.Context(), tmdbID, seasonNumber, libraryID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, episodes)
}

func (r *Router) handleGetMediaItemProgress(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	mediaItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	state, err := r.progress.GetState(req.Context(), user.ID, mediaItemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, state)
}

func (r *Router) handleQueueMediaItemMatch(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	mediaItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	job, err := r.library.QueueMediaItemMatch(req.Context(), mediaItemID, true)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusAccepted, job)
}

func (r *Router) handleQueueMediaItemMetadataRefetch(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	mediaItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	job, err := r.library.QueueMediaItemMetadataRefetch(req.Context(), mediaItemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusAccepted, job)
}

func (r *Router) handleSearchMediaItemMetadata(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("metadata service unavailable"))
		return
	}

	mediaItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	var input metadata.ManualSearchInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if err := validateManualSearchInput(input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	results, err := r.metadata.SearchCandidates(req.Context(), mediaItemID, input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, results)
}

func (r *Router) handleUpdateMediaItemMetadata(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("metadata service unavailable"))
		return
	}

	mediaItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	var input metadata.ManualMetadataInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if err := validateManualMetadataInput(input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	if err := r.metadata.UpdateManualMetadata(req.Context(), mediaItemID, input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	item, err := r.library.GetMediaItem(req.Context(), mediaItemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, item)
}

func (r *Router) handleApplyMediaItemMetadata(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("metadata service unavailable"))
		return
	}

	mediaItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	var input metadata.ApplyCandidateInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(input.ExternalID) == "" {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("external_id is required"))
		return
	}

	if err := r.metadata.ApplyCandidate(req.Context(), mediaItemID, input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	item, err := r.library.GetMediaItem(req.Context(), mediaItemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, item)
}
