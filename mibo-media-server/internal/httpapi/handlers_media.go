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

func (r *Router) handleListMediaItemSeriesEpisodes(w http.ResponseWriter, req *http.Request) {
	if r.rejectLegacyMediaEndpoint(req, w, "/api/v1/series/"+req.PathValue("id")+"/seasons") {
		return
	}
	mediaItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	seasons, err := r.library.ListSeriesEpisodes(req.Context(), mediaItemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	for seasonIdx := range seasons {
		seasons[seasonIdx].PosterURL = buildAssetURL(req, seasons[seasonIdx].PosterURL)
		for episodeIdx := range seasons[seasonIdx].Episodes {
			seasons[seasonIdx].Episodes[episodeIdx].StillURL = buildAssetURL(req, seasons[seasonIdx].Episodes[episodeIdx].StillURL)
		}
	}

	writeJSON(req.Context(), w, http.StatusOK, seasons)
}

func (r *Router) handleGetMediaItemProgress(w http.ResponseWriter, req *http.Request) {
	if r.rejectLegacyMediaEndpoint(req, w, "/api/v1/items/"+req.PathValue("id")) {
		return
	}
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
	if r.rejectLegacyMediaEndpoint(req, w, "/api/v1/items/"+req.PathValue("id")+"/match") {
		return
	}
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
	if r.rejectLegacyMediaEndpoint(req, w, "/api/v1/items/"+req.PathValue("id")+"/metadata/refetch") {
		return
	}
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

func (r *Router) handleQueueMediaFileProbe(w http.ResponseWriter, req *http.Request) {
	if r.rejectLegacyMediaEndpoint(req, w, "/api/v1/inventory-files/{id}/probe") {
		return
	}
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	mediaFileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	job, err := r.library.QueueMediaFileProbe(req.Context(), mediaFileID, true)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusAccepted, job)
}

func (r *Router) handleSearchMediaItemMetadata(w http.ResponseWriter, req *http.Request) {
	if r.rejectLegacyMediaEndpoint(req, w, "/api/v1/items/"+req.PathValue("id")+"/metadata/search") {
		return
	}
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
	if r.rejectLegacyMediaEndpoint(req, w, "/api/v1/items/"+req.PathValue("id")+"/governance/fields") {
		return
	}
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
	normalizeMediaItemDetailArtworkURLs(req, &item)

	writeJSON(req.Context(), w, http.StatusOK, item)
}

func (r *Router) handleApplyMediaItemMetadata(w http.ResponseWriter, req *http.Request) {
	if r.rejectLegacyMediaEndpoint(req, w, "/api/v1/items/"+req.PathValue("id")+"/metadata/apply") {
		return
	}
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
	normalizeMediaItemDetailArtworkURLs(req, &item)

	writeJSON(req.Context(), w, http.StatusOK, item)
}
