package httpapi

import (
	"errors"
	"net/http"

	"github.com/atlan/mibo-media-server/internal/library"
)

func (r *Router) handleQueueInventoryFileProbe(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	fileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	_, err = r.library.QueueInventoryFileProbe(req.Context(), fileID, true)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusAccepted, map[string]any{"queued": true})
}

func (r *Router) handleMarkInventoryFileScanExclusion(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	fileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	r.markScanExclusion(w, req, library.MarkScanExclusionInput{InventoryFileID: fileID, UserID: &user.ID})
}

func (r *Router) handlePreviewInventoryFileFilenameExclusion(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	fileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	r.previewFilenameExclusion(w, req, library.FilenameExclusionTargetInput{InventoryFileID: fileID})
}

func (r *Router) handleCreateInventoryFileFilenameExclusionRule(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	fileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var body scanExclusionMarkInput
	if err := decodeJSON(req, &body); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	rule, err := r.library.CreateFilenameExclusionRule(req.Context(), library.CreateFilenameExclusionRuleInput{FilenameExclusionTargetInput: library.FilenameExclusionTargetInput{InventoryFileID: fileID}, Reason: body.Reason, UserID: &user.ID})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, rule)
}

func (r *Router) previewFilenameExclusion(w http.ResponseWriter, req *http.Request, input library.FilenameExclusionTargetInput) {
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	preview, err := r.library.PreviewFilenameExclusion(req.Context(), input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, preview)
}

func (r *Router) markScanExclusion(w http.ResponseWriter, req *http.Request, input library.MarkScanExclusionInput) {
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	var body scanExclusionMarkInput
	if err := decodeJSON(req, &body); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	input.Reason = body.Reason
	exclusion, err := r.library.MarkScanExclusion(req.Context(), input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, exclusion)
}
