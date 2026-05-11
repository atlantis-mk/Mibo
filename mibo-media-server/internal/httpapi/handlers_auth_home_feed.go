package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/atlan/mibo-media-server/internal/catalog"
)

func (r *Router) handleContinueWatching(w http.ResponseWriter, req *http.Request) {
	r.respondUserCatalogEntries(req, w, func(userID uint, limit int) ([]catalog.CatalogUserItemEntry, error) {
		return r.catalog.ListContinueWatching(req.Context(), userID, limit)
	})
}

func (r *Router) handleRecentlyPlayed(w http.ResponseWriter, req *http.Request) {
	r.respondUserCatalogEntries(req, w, func(userID uint, limit int) ([]catalog.CatalogUserItemEntry, error) {
		return r.catalog.ListRecentlyPlayed(req.Context(), userID, limit)
	})
}

func (r *Router) handleRecentlyAdded(w http.ResponseWriter, req *http.Request) {
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	items, err := r.catalog.ListRecentlyAdded(req.Context(), limit)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	normalizeCatalogItemListArtworkURLs(req, items)
	writeJSON(req.Context(), w, http.StatusOK, items)
}

func (r *Router) handleHomeContentSections(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	sections, err := r.catalog.ListHomeContentSections(req.Context(), limit)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	for idx := range sections {
		normalizeCatalogItemListArtworkURLs(req, sections[idx].Items)
	}
	writeJSON(req.Context(), w, http.StatusOK, sections)
}

func (r *Router) handleHomeMediaOverview(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	previewLimit, _ := strconv.Atoi(req.URL.Query().Get("preview_limit"))
	overview, err := r.catalog.ListHomeMediaOverview(req.Context(), previewLimit)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	for idx := range overview.Sections {
		normalizeCatalogItemListArtworkURLs(req, overview.Sections[idx].Items)
	}
	writeJSON(req.Context(), w, http.StatusOK, overview)
}
