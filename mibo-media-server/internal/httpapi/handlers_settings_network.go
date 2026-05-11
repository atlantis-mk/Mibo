package httpapi

import (
	"errors"
	"net/http"

	"github.com/atlan/mibo-media-server/internal/settings"
)

func (r *Router) handleGetNetworkSettings(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.settings == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("settings service unavailable"))
		return
	}
	result, err := r.settings.GetNetworkSettings(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleUpdateNetworkSettings(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.settings == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("settings service unavailable"))
		return
	}
	var input settings.UpdateNetworkSettingsInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	result, err := r.settings.UpdateNetworkSettings(req.Context(), input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}
