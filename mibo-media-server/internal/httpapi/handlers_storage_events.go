package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/listener"
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

	_, err = r.listener.RecordStorageEvent(req.Context(), listener.EventIngestInput{
		LibraryID: input.LibraryID,
		Kind:      strings.TrimSpace(strings.ToLower(input.Kind)),
		Path:      validatedPath,
		OldPath:   validatedOldPath,
	})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusAccepted, map[string]any{"queued": true})
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
	cleanRoot := filepath.Clean("/" + strings.TrimLeft(strings.TrimSpace(libraryRoot), "/"))
	cleanCandidate := filepath.Clean("/" + strings.TrimLeft(trimmed, "/"))
	if cleanRoot == string(filepath.Separator) {
		return cleanCandidate, nil
	}
	if cleanCandidate != cleanRoot && !strings.HasPrefix(cleanCandidate, cleanRoot+"/") {
		return "", fmt.Errorf("path %s is outside library root %s", cleanCandidate, cleanRoot)
	}
	return cleanCandidate, nil
}
