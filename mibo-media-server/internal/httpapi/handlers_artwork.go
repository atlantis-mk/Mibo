package httpapi

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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
