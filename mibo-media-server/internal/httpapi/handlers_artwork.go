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
	artworkPath, ok := r.generatedArtworkPath(itemID, kind)
	if !ok {
		writeError(req.Context(), w, http.StatusNotFound, os.ErrNotExist)
		return
	}
	r.serveGeneratedArtwork(w, req, artworkPath)
}

func (r *Router) handleGetProgressFrame(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(req.PathValue("name"))
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("invalid progress frame name"))
		return
	}
	framePath, ok := r.generatedProgressFramePath(user.ID, itemID, name)
	if !ok {
		writeError(req.Context(), w, http.StatusNotFound, os.ErrNotExist)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=300")
	http.ServeFile(w, req, framePath)
}

func (r *Router) serveGeneratedArtwork(w http.ResponseWriter, req *http.Request, artworkPath string) {
	if _, err := os.Stat(artworkPath); err != nil {
		writeError(req.Context(), w, http.StatusNotFound, err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=604800")
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

func (r *Router) generatedArtworkPath(itemID uint, kind string) (string, bool) {
	basePath := filepath.Join(r.generatedArtworkRootPath(), "catalog", strconvFormatUint(itemID), kind)
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp"} {
		candidate := basePath + ext
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}

func (r *Router) generatedProgressFramePath(userID uint, itemID uint, name string) (string, bool) {
	basePath := filepath.Join(r.generatedArtworkRootPath(), "progress", strconvFormatUint(userID), strconvFormatUint(itemID), name)
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp"} {
		candidate := basePath + ext
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}
