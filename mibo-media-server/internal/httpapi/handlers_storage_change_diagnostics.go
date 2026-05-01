package httpapi

import (
	"errors"
	"net/http"
)

func (r *Router) handleStorageChangeDiagnostics(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.listener == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("listener service unavailable"))
		return
	}
	diagnostics, err := r.listener.Diagnostics(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, diagnostics)
}
