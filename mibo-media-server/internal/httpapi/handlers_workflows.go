package httpapi

import (
	"net/http"
	"strconv"
	"time"
)

func (r *Router) handleListWorkflows(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireAdminUser(req); err != nil {
		status := http.StatusUnauthorized
		if err.Error() == "admin access required" {
			status = http.StatusForbidden
		}
		writeError(req.Context(), w, status, err)
		return
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(req.URL.Query().Get("offset"))
	libraryID64, _ := strconv.ParseUint(req.URL.Query().Get("library_id"), 10, 64)
	views, err := r.workflow.ListRuns(req.Context(), limit, offset, req.URL.Query().Get("status"), uint(libraryID64))
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, views)
}

func (r *Router) handleGetWorkflow(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireAdminUser(req); err != nil {
		status := http.StatusUnauthorized
		if err.Error() == "admin access required" {
			status = http.StatusForbidden
		}
		writeError(req.Context(), w, status, err)
		return
	}
	runID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	view, err := r.workflow.GetRunStatus(req.Context(), runID)
	if err != nil {
		writeError(req.Context(), w, http.StatusNotFound, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, view)
}

func (r *Router) handleWorkflowDiagnostics(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireAdminUser(req); err != nil {
		status := http.StatusUnauthorized
		if err.Error() == "admin access required" {
			status = http.StatusForbidden
		}
		writeError(req.Context(), w, status, err)
		return
	}
	diagnostics, err := r.workflow.Diagnostics(req.Context(), time.Now().UTC())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, diagnostics)
}
