package httpapi

import (
	"net/http"
	"strconv"

	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/progress"
)

func (r *Router) handleRegister(w http.ResponseWriter, req *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	user, err := r.auth.Register(req.Context(), input.Username, input.Password)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusCreated, user)
}

func (r *Router) handleLogin(w http.ResponseWriter, req *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	result, err := r.auth.Login(req.Context(), input.Username, input.Password)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleLogout(w http.ResponseWriter, req *http.Request) {
	token, err := bearerToken(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if err := r.auth.Logout(req.Context(), token); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, map[string]any{"status": "logged_out"})
}

func (r *Router) handleMe(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, user)
}

func (r *Router) handleUpdateProgress(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	var input progress.UpdateInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	state, err := r.progress.Update(req.Context(), user.ID, input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, state)
}

func (r *Router) handleContinueWatching(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	entries, err := r.progress.ContinueWatching(req.Context(), user.ID, limit)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, entries)
}

func (r *Router) handleRecentlyPlayed(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	entries, err := r.progress.RecentlyPlayed(req.Context(), user.ID, limit)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, entries)
}

func (r *Router) handleRecentlyAdded(w http.ResponseWriter, req *http.Request) {
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	if r.shouldServeCatalogLibraryItems(req.Context()) && r.catalog != nil {
		items, err := r.catalog.ListRecentlyAdded(req.Context(), limit)
		if err != nil {
			writeError(req.Context(), w, http.StatusInternalServerError, err)
			return
		}
		normalizeCatalogListItemsArtworkURLs(req, items)
		writeJSON(req.Context(), w, http.StatusOK, items)
		return
	}
	items, err := r.library.ListRecentlyAdded(req.Context(), limit)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	if len(items) == 0 && r.catalog != nil {
		catalogItems, err := r.catalog.ListRecentlyAdded(req.Context(), limit)
		if err != nil {
			writeError(req.Context(), w, http.StatusInternalServerError, err)
			return
		}
		normalizeCatalogListItemsArtworkURLs(req, catalogItems)
		writeJSON(req.Context(), w, http.StatusOK, catalogItems)
		return
	}
	normalizeMediaItemSliceArtworkURLs(req, items)
	writeJSON(req.Context(), w, http.StatusOK, items)
}

func (r *Router) handleLatestByLibrary(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.shouldServeCatalogLibraryItems(req.Context()) && r.catalog != nil {
		sections, err := r.catalog.ListLatestByLibrary(req.Context(), 12)
		if err != nil {
			writeError(req.Context(), w, http.StatusInternalServerError, err)
			return
		}
		for idx := range sections {
			normalizeCatalogListItemsArtworkURLs(req, sections[idx].Items)
		}
		writeJSON(req.Context(), w, http.StatusOK, sections)
		return
	}

	sections, err := r.library.ListAllLatestMediaByLibrary(req.Context(), 12)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	if !hasLatestByLibraryItems(sections) && r.catalog != nil {
		catalogSections, err := r.catalog.ListLatestByLibrary(req.Context(), 12)
		if err != nil {
			writeError(req.Context(), w, http.StatusInternalServerError, err)
			return
		}
		for idx := range catalogSections {
			normalizeCatalogListItemsArtworkURLs(req, catalogSections[idx].Items)
		}
		writeJSON(req.Context(), w, http.StatusOK, catalogSections)
		return
	}
	normalizeLatestByLibraryArtworkURLs(req, sections)

	writeJSON(req.Context(), w, http.StatusOK, sections)
}

func hasLatestByLibraryItems(sections []library.LatestByLibrarySection) bool {
	for _, section := range sections {
		if len(section.Items) > 0 {
			return true
		}
	}
	return false
}

func (r *Router) handleHomeDiscovery(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	continueWatching, err := r.progress.ContinueWatching(req.Context(), user.ID, 20)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	recentlyPlayed, err := r.progress.RecentlyPlayed(req.Context(), user.ID, 20)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	latestByLibrary, err := r.library.ListLatestByLibrary(req.Context(), 12)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	normalizeProgressEntriesArtworkURLs(req, continueWatching)
	normalizeProgressEntriesArtworkURLs(req, recentlyPlayed)
	normalizeLatestByLibraryArtworkURLs(req, latestByLibrary)

	writeJSON(req.Context(), w, http.StatusOK, homeDiscoveryResponse{
		ContinueWatching: continueWatching,
		RecentlyPlayed:   recentlyPlayed,
		LatestByLibrary:  latestByLibrary,
	})
}
