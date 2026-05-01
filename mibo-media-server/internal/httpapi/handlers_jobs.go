package httpapi

import (
	"net/http"
	"strconv"
)

func (r *Router) handleListJobs(w http.ResponseWriter, req *http.Request) {
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
	status := req.URL.Query().Get("status")
	kind := req.URL.Query().Get("kind")
	jobList, err := r.jobs.List(req.Context(), limit, offset, status, kind)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, jobList)
}

func (r *Router) handleRetryJob(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireAdminUser(req); err != nil {
		status := http.StatusUnauthorized
		if err.Error() == "admin access required" {
			status = http.StatusForbidden
		}
		writeError(req.Context(), w, status, err)
		return
	}

	jobID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	job, err := r.jobs.Retry(req.Context(), jobID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusAccepted, job)
}

func (r *Router) handleCancelJob(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireAdminUser(req); err != nil {
		status := http.StatusUnauthorized
		if err.Error() == "admin access required" {
			status = http.StatusForbidden
		}
		writeError(req.Context(), w, status, err)
		return
	}

	jobID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	job, err := r.jobs.Cancel(req.Context(), jobID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusAccepted, job)
}
