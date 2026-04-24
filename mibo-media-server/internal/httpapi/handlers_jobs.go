package httpapi

import (
	"net/http"
	"strconv"
)

func (r *Router) handleListJobs(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	status := req.URL.Query().Get("status")
	kind := req.URL.Query().Get("kind")
	jobList, err := r.jobs.List(req.Context(), limit, status, kind)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, jobList)
}

func (r *Router) handleRetryJob(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
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
