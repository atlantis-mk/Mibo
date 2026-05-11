package httpapi

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/atlan/mibo-media-server/internal/settings"
)

func (r *Router) handleGetScanSettings(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.settings == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("settings service unavailable"))
		return
	}
	result, err := r.settings.GetScanSettings(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleUpdateScanSettings(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.settings == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("settings service unavailable"))
		return
	}
	var input settings.UpdateScanSettingsInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if input.RefreshIntervalHours <= 0 || input.RefreshIntervalHours > 720 {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("refresh_interval_hours must be between 1 and 720"))
		return
	}
	result, err := r.settings.UpdateScanSettings(req.Context(), input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}
