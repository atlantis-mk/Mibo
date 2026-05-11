package httpapi

import (
	"net/http"

	"github.com/atlan/mibo-media-server/internal/playback"
)

func (r *Router) handleGetCatalogPlaybackSource(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	resourceID, err := parseOptionalUintQuery(req, "resource_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	libraryID, err := parseOptionalUintQuery(req, "library_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	clientProfile, err := parseClientProfileQuery(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	userID := user.ID
	source, err := r.playback.GetPlaybackSource(req.Context(), playback.PlaybackRequest{MetadataItemID: itemID, ResourceID: resourceID, LibraryID: libraryID, UserID: &userID, ClientProfile: clientProfile})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	source.URL = buildPlaybackURL(req, source.URL)
	writeJSON(req.Context(), w, http.StatusOK, source)
}

func (r *Router) handleGetInventoryFilePlaybackSource(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	fileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	clientProfile, err := parseClientProfileQuery(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	source, err := r.playback.GetInventoryFilePlaybackSource(req.Context(), fileID, clientProfile)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	source.URL = buildPlaybackURL(req, source.URL)
	writeJSON(req.Context(), w, http.StatusOK, source)
}
