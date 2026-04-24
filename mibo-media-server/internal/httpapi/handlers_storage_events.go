package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

func (r *Router) handleStorageEvent(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	var input struct {
		LibraryID uint   `json:"library_id"`
		Kind      string `json:"kind"`
		Path      string `json:"path"`
		OldPath   string `json:"old_path"`
	}
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if input.LibraryID == 0 {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("library_id is required"))
		return
	}
	validatedPath, validatedOldPath, err := r.validateStorageEventPaths(req.Context(), input.LibraryID, input.Path, input.OldPath)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	eventRoot, fallbackToFullSync, err := normalizeStorageEventRoot(strings.TrimSpace(strings.ToLower(input.Kind)), validatedPath, validatedOldPath)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	var job database.Job
	if fallbackToFullSync {
		job, err = r.library.QueueLibraryScan(req.Context(), input.LibraryID)
	} else {
		job, err = r.library.QueueTargetedRefresh(req.Context(), input.LibraryID, eventRoot, "storage_event")
	}
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusAccepted, job)
}

func (r *Router) validateStorageEventPaths(ctx context.Context, libraryID uint, currentPath string, oldPath string) (string, string, error) {
	var record database.Library
	if err := r.db.WithContext(ctx).First(&record, libraryID).Error; err != nil {
		return "", "", err
	}
	var source database.MediaSource
	if err := r.db.WithContext(ctx).First(&source, record.MediaSourceID).Error; err != nil {
		return "", "", err
	}

	validatedCurrent, err := validateStorageEventPath(source.Provider, record.RootPath, currentPath)
	if err != nil {
		return "", "", err
	}
	validatedOld, err := validateStorageEventPath(source.Provider, record.RootPath, oldPath)
	if err != nil {
		return "", "", err
	}
	return validatedCurrent, validatedOld, nil
}

func validateStorageEventPath(providerName string, libraryRoot string, candidate string) (string, error) {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" {
		return "", nil
	}
	if strings.EqualFold(strings.TrimSpace(providerName), "local") {
		cleanRoot := filepath.Clean(strings.TrimSpace(libraryRoot))
		cleanCandidate := filepath.Clean(trimmed)
		rel, err := filepath.Rel(cleanRoot, cleanCandidate)
		if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
			return "", fmt.Errorf("path %s is outside library root %s", cleanCandidate, cleanRoot)
		}
		return cleanCandidate, nil
	}
	cleanRoot := path.Clean("/" + strings.TrimLeft(strings.TrimSpace(libraryRoot), "/"))
	cleanCandidate := path.Clean("/" + strings.TrimLeft(trimmed, "/"))
	if cleanCandidate != cleanRoot && !strings.HasPrefix(cleanCandidate, cleanRoot+"/") {
		return "", fmt.Errorf("path %s is outside library root %s", cleanCandidate, cleanRoot)
	}
	return cleanCandidate, nil
}

func normalizeStorageEventRoot(kind string, currentPath string, oldPath string) (string, bool, error) {
	cleanCurrent := strings.TrimSpace(currentPath)
	cleanOld := strings.TrimSpace(oldPath)
	switch kind {
	case "create", "update", "delete":
		if cleanCurrent == "" {
			return "", false, fmt.Errorf("path is required")
		}
		return targetedEventRoot(cleanCurrent), false, nil
	case "move", "rename":
		if cleanCurrent == "" || cleanOld == "" {
			return "", true, nil
		}
		return targetedEventRoot(commonAncestorPath(cleanOld, cleanCurrent)), false, nil
	case "":
		return "", false, fmt.Errorf("kind is required")
	default:
		return "", true, nil
	}
}

func commonAncestorPath(left string, right string) string {
	leftClean := filepath.Clean(strings.TrimSpace(left))
	rightClean := filepath.Clean(strings.TrimSpace(right))
	leftParts := strings.Split(leftClean, string(filepath.Separator))
	rightParts := strings.Split(rightClean, string(filepath.Separator))
	shared := make([]string, 0, min(len(leftParts), len(rightParts)))
	for idx := 0; idx < len(leftParts) && idx < len(rightParts); idx++ {
		if leftParts[idx] != rightParts[idx] {
			break
		}
		shared = append(shared, leftParts[idx])
	}
	if len(shared) == 0 {
		return string(filepath.Separator)
	}
	joined := filepath.Join(shared...)
	if strings.HasPrefix(strings.TrimSpace(left), "/") || strings.HasPrefix(strings.TrimSpace(right), "/") {
		if !strings.HasPrefix(joined, string(filepath.Separator)) {
			joined = string(filepath.Separator) + joined
		}
	}
	return joined
}

func targetedEventRoot(value string) string {
	clean := filepath.Clean(strings.TrimSpace(value))
	if clean == "." || clean == "" {
		return clean
	}
	if ext := filepath.Ext(clean); ext != "" {
		return filepath.Dir(clean)
	}
	return clean
}
