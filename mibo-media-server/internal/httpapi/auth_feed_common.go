package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
)

type catalogUserEntryLister func(userID uint, limit int) ([]catalog.CatalogUserItemEntry, error)

func (r *Router) respondUserCatalogEntries(req *http.Request, w http.ResponseWriter, list catalogUserEntryLister) {
	entries, err := r.listUserCatalogEntries(req, list)
	if err != nil {
		writeCatalogEntryListError(req, w, err)
		return
	}
	normalizeUserItemEntryURLs(req, entries)
	writeJSON(req.Context(), w, http.StatusOK, entries)
}

func (r *Router) listUserCatalogEntries(req *http.Request, list catalogUserEntryLister) ([]catalog.CatalogUserItemEntry, error) {
	user, err := r.requireUser(req)
	if err != nil {
		return nil, err
	}
	if r.catalog == nil {
		return nil, errors.New("catalog service unavailable")
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	return list(user.ID, limit)
}

func writeCatalogEntryListError(req *http.Request, w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	if strings.Contains(message, "unauthorized") || strings.Contains(message, "missing bearer token") || strings.Contains(message, "invalid token") {
		status = http.StatusUnauthorized
	}
	writeError(req.Context(), w, status, err)
}

func normalizeUserItemEntryURLs(req *http.Request, entries []catalog.CatalogUserItemEntry) {
	for idx := range entries {
		entries[idx].ProgressFrameURL = buildAssetURL(req, entries[idx].ProgressFrameURL)
		normalizeCatalogItemSummaryArtworkURLs(req, &entries[idx].Item)
		if entries[idx].DisplayItem != nil {
			normalizeCatalogItemSummaryArtworkURLs(req, entries[idx].DisplayItem)
		}
		if entries[idx].PlayItem != nil {
			normalizeCatalogItemSummaryArtworkURLs(req, entries[idx].PlayItem)
		}
	}
}
