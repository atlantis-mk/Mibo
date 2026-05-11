package httpapi

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/atlan/mibo-media-server/internal/progress"
)

type setPreferredResourceInput struct {
	MetadataItemID uint `json:"metadata_item_id"`
	ResourceID     uint `json:"resource_id"`
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
	frameURL, err := r.saveProgressFrame(user.ID, input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if frameURL != "" {
		input.ProgressFrameURL = frameURL
	}

	state, err := r.progress.Update(req.Context(), user.ID, input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, state)
}

func (r *Router) saveProgressFrame(userID uint, input progress.UpdateInput) (string, error) {
	frame := strings.TrimSpace(input.ProgressFrameData)
	if frame == "" {
		return "", nil
	}
	if input.MetadataItemID == 0 {
		return "", fmt.Errorf("metadata_item_id is required for progress frame")
	}

	mediaType, payload, ok := strings.Cut(frame, ",")
	if !ok || !strings.HasPrefix(mediaType, "data:image/") || !strings.Contains(mediaType, ";base64") {
		return "", fmt.Errorf("progress_frame_data must be a base64 image data URL")
	}
	ext := progressFrameExtension(mediaType)
	if ext == "" {
		return "", fmt.Errorf("progress_frame_data must be jpeg, png, or webp")
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("decode progress frame: %w", err)
	}
	if len(decoded) == 0 || len(decoded) > 2*1024*1024 {
		return "", fmt.Errorf("progress frame must be between 1 byte and 2 MiB")
	}

	dir := filepath.Join(r.generatedArtworkRootPath(), "progress", strconvFormatUint(userID), strconvFormatUint(input.MetadataItemID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	baseName := "resource-" + strconvFormatUint(input.ResourceID)
	for _, staleExt := range []string{".jpg", ".jpeg", ".png", ".webp"} {
		_ = os.Remove(filepath.Join(dir, baseName+staleExt))
	}
	path := filepath.Join(dir, baseName+ext)
	if err := os.WriteFile(path, decoded, 0o644); err != nil {
		return "", err
	}

	return "/api/v1/me/progress-frames/" + strconvFormatUint(input.MetadataItemID) + "/" + baseName, nil
}

func (r *Router) handleSetPreferredResource(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	var input setPreferredResourceInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	state, err := r.progress.SetPreferredResource(req.Context(), user.ID, input.MetadataItemID, input.ResourceID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, state)
}

func progressFrameExtension(mediaType string) string {
	switch {
	case strings.HasPrefix(mediaType, "data:image/jpeg"):
		return ".jpg"
	case strings.HasPrefix(mediaType, "data:image/png"):
		return ".png"
	case strings.HasPrefix(mediaType, "data:image/webp"):
		return ".webp"
	default:
		return ""
	}
}
