package httpapi

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/progress"
	"github.com/atlan/mibo-media-server/internal/search"
)

func (r *Router) handleGetMediaItemArtwork(w http.ResponseWriter, req *http.Request) {
	if r.rejectLegacyMediaEndpoint(req, w, "/api/v1/items/"+req.PathValue("id")) {
		return
	}
	mediaItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	kind := normalizeArtworkKind(req.PathValue("kind"))
	if kind == "" {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("artwork kind must be poster or backdrop"))
		return
	}
	artworkPath := filepath.Join(r.generatedArtworkRootPath(), strconvFormatUint(mediaItemID), kind+".jpg")
	r.serveGeneratedArtwork(w, req, artworkPath)
}

func (r *Router) handleGetCatalogItemArtwork(w http.ResponseWriter, req *http.Request) {
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	kind := normalizeArtworkKind(req.PathValue("kind"))
	if kind == "" {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("artwork kind must be poster or backdrop"))
		return
	}
	artworkPath := filepath.Join(r.generatedArtworkRootPath(), "catalog", strconvFormatUint(itemID), kind+".jpg")
	r.serveGeneratedArtwork(w, req, artworkPath)
}

func (r *Router) serveGeneratedArtwork(w http.ResponseWriter, req *http.Request, artworkPath string) {
	if _, err := os.Stat(artworkPath); err != nil {
		writeError(req.Context(), w, http.StatusNotFound, err)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeFile(w, req, artworkPath)
}

func normalizeMediaItemArtworkURLs(req *http.Request, item *database.MediaItem) {
	if item == nil {
		return
	}
	item.PosterURL = buildAssetURL(req, item.PosterURL)
	item.BackdropURL = buildAssetURL(req, item.BackdropURL)
	item.LogoURL = buildAssetURL(req, item.LogoURL)
}

func normalizeMediaItemDetailArtworkURLs(req *http.Request, item *library.MediaItemDetail) {
	if item == nil {
		return
	}
	normalizeMediaItemArtworkURLs(req, &item.MediaItem)
}

func normalizeMediaItemSliceArtworkURLs(req *http.Request, items []database.MediaItem) {
	for idx := range items {
		normalizeMediaItemArtworkURLs(req, &items[idx])
	}
}

func normalizeDiscoveryItemsArtworkURLs(req *http.Request, items []library.DiscoveryItem) {
	for idx := range items {
		normalizeMediaItemArtworkURLs(req, &items[idx].Item)
	}
}

func normalizeSearchResultsArtworkURLs(req *http.Request, items []search.Result) {
	for idx := range items {
		normalizeMediaItemArtworkURLs(req, &items[idx].Item)
	}
}

func normalizeLatestByLibraryArtworkURLs(req *http.Request, sections []library.LatestByLibrarySection) {
	for idx := range sections {
		normalizeMediaItemSliceArtworkURLs(req, sections[idx].Items)
	}
}

func normalizeProgressEntriesArtworkURLs(req *http.Request, entries []progress.Entry) {
	for idx := range entries {
		normalizeMediaItemArtworkURLs(req, &entries[idx].MediaItem)
	}
}

func buildAssetURL(req *http.Request, rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" || !strings.HasPrefix(trimmed, "/") {
		return trimmed
	}
	return requestBaseURL(req) + trimmed
}

func normalizeArtworkKind(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "poster", "backdrop":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func strconvFormatUint(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}

func (r *Router) generatedArtworkRootPath() string {
	trimmed := strings.TrimSpace(r.cfg.FFmpeg.ArtworkRootPath)
	if trimmed != "" {
		return trimmed
	}
	return filepath.Join("tmp", "artwork")
}
