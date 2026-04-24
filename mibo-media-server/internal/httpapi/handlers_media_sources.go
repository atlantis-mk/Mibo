package httpapi

import (
	"net/http"
	"strings"

	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/providers"
)

func (r *Router) handleCreateMediaSource(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	var input library.CreateMediaSourceInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	source, err := r.library.CreateMediaSource(req.Context(), input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	view, err := r.library.MediaSourceView(source)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusCreated, view)
}

func (r *Router) handleBrowseStorageProvider(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	providerName := strings.TrimSpace(req.PathValue("provider"))
	result, err := r.library.BrowseProviderPath(req.Context(), providerName, req.URL.Query().Get("path"))
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleBrowseTemporaryOpenList(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	var input struct {
		Path   string                         `json:"path"`
		Config providers.OpenListSourceConfig `json:"config"`
	}
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	result, err := r.library.BrowseTemporaryOpenListPath(req.Context(), input.Config, input.Path)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleTestTemporaryOpenList(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	var input struct {
		Config providers.OpenListSourceConfig `json:"config"`
	}
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	result, err := r.library.TestTemporaryOpenListConnection(req.Context(), input.Config)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleUpdateMediaSource(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	sourceID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	var input library.UpdateMediaSourceInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	source, err := r.library.UpdateMediaSource(req.Context(), sourceID, input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	view, err := r.library.MediaSourceView(source)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, view)
}

func (r *Router) handleListMediaSources(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	sources, err := r.library.ListMediaSourceViews(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, sources)
}

func (r *Router) handleDeleteMediaSource(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	sourceID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	if err := r.library.DeleteMediaSource(req.Context(), sourceID); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, map[string]any{
		"id":     sourceID,
		"status": "deleted",
		"type":   "media_source",
	})
}

func (r *Router) handleBrowseMediaSource(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	sourceID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	result, err := r.library.BrowseMediaSourcePath(req.Context(), sourceID, req.URL.Query().Get("path"))
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, result)
}
