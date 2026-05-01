package httpapi

import (
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/atlan/mibo-media-server/internal/config"
)

func newWebAppHandler(cfg config.WebConfig, embedded fs.FS) http.Handler {
	dist := embedded
	if cfg.DistDir != "" {
		if _, err := os.Stat(filepath.Join(cfg.DistDir, "index.html")); err == nil {
			dist = os.DirFS(cfg.DistDir)
		} else {
			log.Printf("web dist override unavailable path=%s err=%v; using embedded assets", cfg.DistDir, err)
		}
	}

	return &webAppHandler{dist: dist}
}

type webAppHandler struct {
	dist fs.FS
}

func (h *webAppHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/api" || strings.HasPrefix(req.URL.Path, "/api/") {
		writeError(req.Context(), w, http.StatusNotFound, os.ErrNotExist)
		return
	}

	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeError(req.Context(), w, http.StatusMethodNotAllowed, os.ErrPermission)
		return
	}

	filePath := strings.TrimPrefix(path.Clean(req.URL.Path), "/")
	if filePath == "." || filePath == "" {
		filePath = "index.html"
	}

	file, err := h.dist.Open(filePath)
	if err == nil {
		defer file.Close()
		if stat, statErr := file.Stat(); statErr == nil && !stat.IsDir() {
			h.serveFile(w, req, filePath, file, stat)
			return
		}
	}

	index, err := h.dist.Open("index.html")
	if err != nil {
		writeError(req.Context(), w, http.StatusNotFound, err)
		return
	}
	defer index.Close()

	stat, err := index.Stat()
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	h.serveFile(w, req, "index.html", index, stat)
}

func (h *webAppHandler) serveFile(w http.ResponseWriter, req *http.Request, name string, file fs.File, stat fs.FileInfo) {
	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else if name == "index.html" {
		w.Header().Set("Cache-Control", "no-store, max-age=0, must-revalidate")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}

	if contentType := mime.TypeByExtension(path.Ext(name)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	reader, ok := file.(interface {
		Read([]byte) (int, error)
		Seek(int64, int) (int64, error)
	})
	if !ok {
		writeError(req.Context(), w, http.StatusInternalServerError, os.ErrInvalid)
		return
	}

	http.ServeContent(w, req, name, stat.ModTime(), reader)
}
