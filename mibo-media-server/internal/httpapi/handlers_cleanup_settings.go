package httpapi

import (
	"errors"
	"net/http"

	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/settings"
)

func (r *Router) handleGetCleanupSettings(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.settings == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("settings service unavailable"))
		return
	}
	result, err := r.settings.GetCleanupSettings(req.Context(), r.cfg.Cleanup)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleUpdateCleanupSettings(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireAdminUser(req); err != nil {
		writeError(req.Context(), w, http.StatusForbidden, err)
		return
	}
	if r.settings == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("settings service unavailable"))
		return
	}
	var input settings.UpdateCleanupSettingsInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	result, err := r.settings.UpdateCleanupSettings(req.Context(), r.cfg.Cleanup, input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleRunMissingMediaCleanup(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireAdminUser(req); err != nil {
		writeError(req.Context(), w, http.StatusForbidden, err)
		return
	}
	if r.settings == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("settings service unavailable"))
		return
	}
	cleanupSettings, err := r.settings.GetCleanupSettings(req.Context(), r.cfg.Cleanup)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	if !cleanupSettings.MissingCleanupEnabled {
		writeError(req.Context(), w, http.StatusBadRequest, errors.New("missing cleanup is disabled"))
		return
	}
	_, err = r.library.QueueMissingMediaCleanup(req.Context(), library.MissingMediaCleanupPayload{ScopeKind: "global"})
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusAccepted, map[string]any{"queued": true})
}
