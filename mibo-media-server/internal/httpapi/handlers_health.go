package httpapi

import (
	"errors"
	"net/http"
)

func (r *Router) handleHealthSummary(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.health == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("health service unavailable"))
		return
	}
	summary, err := r.health.Summary(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, summary)
}

func (r *Router) handleHealthIssues(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.health == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("health service unavailable"))
		return
	}
	issues, err := r.health.ListIssues(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, issues)
}

func (r *Router) handleValidateMediaSource(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.health == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("health service unavailable"))
		return
	}
	mediaSourceID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	result, err := r.health.ValidateMediaSource(req.Context(), mediaSourceID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleIgnoreHealthIssue(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.health == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("health service unavailable"))
		return
	}
	issueID := req.PathValue("id")
	if issueID == "" {
		writeError(req.Context(), w, http.StatusBadRequest, errors.New("issue id is required"))
		return
	}
	result, err := r.health.IgnoreIssue(req.Context(), issueID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleRescanHealthIssueLibraries(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.health == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("health service unavailable"))
		return
	}
	issueID := req.PathValue("id")
	if issueID == "" {
		writeError(req.Context(), w, http.StatusBadRequest, errors.New("issue id is required"))
		return
	}
	result, err := r.health.RescanIssueLibraries(req.Context(), issueID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusAccepted, result)
}
