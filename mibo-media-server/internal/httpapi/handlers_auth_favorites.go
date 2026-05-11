package httpapi

import (
	"errors"
	"net/http"
	"strconv"
)

func (r *Router) handleListFavorites(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	entries, err := r.catalog.ListFavorites(req.Context(), user.ID, limit)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, entries)
}

func (r *Router) handleAddFavorite(w http.ResponseWriter, req *http.Request) {
	r.handleFavoriteMutation(w, req, true)
}

func (r *Router) handleRemoveFavorite(w http.ResponseWriter, req *http.Request) {
	r.handleFavoriteMutation(w, req, false)
}

func (r *Router) handleFavoriteMutation(w http.ResponseWriter, req *http.Request, favorite bool) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	entry, err := r.catalog.SetFavorite(req.Context(), user.ID, itemID, favorite)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, entry)
}
