package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/ingest"
)

func (r *Router) handleAdminIngestDiagnostics(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireAdminUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.ingest == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, fmt.Errorf("ingest service unavailable"))
		return
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	result, err := r.ingest.Diagnostics(req.Context(), ingest.DiagnosticsInput{Status: strings.TrimSpace(req.URL.Query().Get("status")), Limit: limit})
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleAdminRetryIngestStage(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireAdminUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.ingest == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, fmt.Errorf("ingest service unavailable"))
		return
	}
	conditionID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	userID := user.ID
	result, err := r.ingest.RetryStage(req.Context(), conditionID, &userID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusAccepted, result)
}

func (r *Router) handleAdminResolveIngestReviewStage(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireAdminUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.ingest == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, fmt.Errorf("ingest service unavailable"))
		return
	}
	conditionID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	userID := user.ID
	result, err := r.ingest.ResolveReviewStage(req.Context(), conditionID, &userID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}

type adminIngestReconcileInput struct {
	LibraryID uint   `json:"library_id"`
	RootPath  string `json:"root_path"`
	Reason    string `json:"reason"`
}

func (r *Router) handleAdminIngestReconcile(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireAdminUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.ingest == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, fmt.Errorf("ingest service unavailable"))
		return
	}
	var input adminIngestReconcileInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = "admin_reconcile"
	}
	unit, err := r.ingest.MarkLibraryScopeDirty(req.Context(), input.LibraryID, input.RootPath, reason)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusAccepted, unit)
}
