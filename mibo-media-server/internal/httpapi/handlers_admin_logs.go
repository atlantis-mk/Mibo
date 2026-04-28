package httpapi

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const adminLogDir = "data/logs"

type adminLogFile struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	SizeBytes  int64     `json:"size_bytes"`
	Kind       string    `json:"kind"`
}

type adminLogContent struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func (r *Router) handleListAdminLogs(w http.ResponseWriter, req *http.Request) {
	if !r.requireAdmin(w, req) {
		return
	}

	entries, err := os.ReadDir(adminLogDir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(req.Context(), w, http.StatusOK, []adminLogFile{})
			return
		}
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	logs := make([]adminLogFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isLogFileName(entry.Name()) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		logs = append(logs, adminLogFile{
			Name:       entry.Name(),
			ModifiedAt: info.ModTime(),
			SizeBytes:  info.Size(),
			Kind:       classifyLogFile(entry.Name()),
		})
	}

	sort.Slice(logs, func(i, j int) bool {
		return logs[i].ModifiedAt.After(logs[j].ModifiedAt)
	})

	writeJSON(req.Context(), w, http.StatusOK, logs)
}

func (r *Router) handleGetAdminLog(w http.ResponseWriter, req *http.Request) {
	if !r.requireAdmin(w, req) {
		return
	}

	path, name, ok := resolveLogPath(req.PathValue("name"))
	if !ok {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("invalid log file name"))
		return
	}

	content, err := os.ReadFile(path)
	if err != nil {
		status := http.StatusInternalServerError
		if os.IsNotExist(err) {
			status = http.StatusNotFound
		}
		writeError(req.Context(), w, status, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, adminLogContent{
		Name:    name,
		Content: string(content),
	})
}

func (r *Router) handleDownloadAdminLog(w http.ResponseWriter, req *http.Request) {
	if !r.requireAdmin(w, req) {
		return
	}

	path, name, ok := resolveLogPath(req.PathValue("name"))
	if !ok {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("invalid log file name"))
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	http.ServeFile(w, req, path)
}

func (r *Router) handleDeleteAdminLog(w http.ResponseWriter, req *http.Request) {
	if !r.requireAdmin(w, req) {
		return
	}

	path, name, ok := resolveLogPath(req.PathValue("name"))
	if !ok {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("invalid log file name"))
		return
	}

	if err := os.Remove(path); err != nil {
		status := http.StatusInternalServerError
		if os.IsNotExist(err) {
			status = http.StatusNotFound
		}
		writeError(req.Context(), w, status, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, map[string]string{
		"name":   name,
		"status": "deleted",
	})
}

func (r *Router) requireAdmin(w http.ResponseWriter, req *http.Request) bool {
	if _, err := r.requireAdminUser(req); err != nil {
		status := http.StatusUnauthorized
		if err.Error() == "admin access required" {
			status = http.StatusForbidden
		}
		writeError(req.Context(), w, status, err)
		return false
	}
	return true
}

func resolveLogPath(name string) (string, string, bool) {
	cleanName := filepath.Base(strings.TrimSpace(name))
	if cleanName == "." || cleanName == "" || cleanName != name || !isLogFileName(cleanName) {
		return "", "", false
	}
	return filepath.Join(adminLogDir, cleanName), cleanName, true
}

func isLogFileName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	return strings.HasSuffix(lower, ".txt") || strings.HasSuffix(lower, ".log")
}

func classifyLogFile(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "ffmpeg") || strings.Contains(lower, "transcode"):
		return "转码"
	case strings.Contains(lower, "server"):
		return "服务器"
	default:
		return "日志"
	}
}
