package httpapi

import (
	"errors"
	"net/http"

	"github.com/atlan/mibo-media-server/internal/auth"
)

func (r *Router) handleListAuthSessions(w http.ResponseWriter, req *http.Request) {
	user, token, err := r.authenticateBearerUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	sessions, err := r.auth.ListLoginSessions(req.Context(), user.ID, token)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, sessions)
}

func (r *Router) handleRevokeAuthSession(w http.ResponseWriter, req *http.Request) {
	user, token, err := r.authenticateBearerUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	sessionID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if err := r.auth.RevokeLoginSession(req.Context(), user.ID, sessionID, token); err != nil {
		switch {
		case errors.Is(err, auth.ErrCurrentSession):
			writeError(req.Context(), w, http.StatusBadRequest, err)
		case errors.Is(err, auth.ErrSessionNotFound):
			writeError(req.Context(), w, http.StatusNotFound, err)
		default:
			writeError(req.Context(), w, http.StatusInternalServerError, err)
		}
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, map[string]any{"id": sessionID, "status": "revoked"})
}

func (r *Router) handleRevokeOtherAuthSessions(w http.ResponseWriter, req *http.Request) {
	user, token, err := r.authenticateBearerUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if err := r.auth.RevokeOtherLoginSessions(req.Context(), user.ID, token); err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, map[string]any{"status": "revoked"})
}
