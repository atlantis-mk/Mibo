package httpapi

import (
	"net/http"

	"github.com/atlan/mibo-media-server/internal/schedule"
)

type scheduleMutationInput struct {
	Name      string                 `json:"name"`
	Kind      string                 `json:"kind"`
	ScopeKind schedule.ScopeKind     `json:"scope_kind"`
	LibraryID *uint                  `json:"library_id,omitempty"`
	Enabled   *bool                  `json:"enabled,omitempty"`
	Frequency schedule.FrequencySpec `json:"frequency"`
}

type scheduleToggleInput struct {
	Enabled bool `json:"enabled"`
}

func (r *Router) handleListSchedules(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	schedules, err := r.schedule.List(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, schedules)
}

func (r *Router) handleGetSchedule(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	id, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	scheduleRecord, err := r.schedule.Get(req.Context(), id)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, scheduleRecord)
}

func (r *Router) handleCreateSchedule(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	var input scheduleMutationInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	created, err := r.schedule.Create(req.Context(), schedule.CreateScheduleInput{Name: input.Name, Kind: input.Kind, ScopeKind: input.ScopeKind, LibraryID: input.LibraryID, Enabled: enabled, Frequency: input.Frequency})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusCreated, created)
}

func (r *Router) handleUpdateSchedule(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	id, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input scheduleMutationInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	updated, err := r.schedule.Update(req.Context(), id, schedule.UpdateScheduleInput{Name: &input.Name, Kind: &input.Kind, ScopeKind: &input.ScopeKind, LibraryID: input.LibraryID, Enabled: input.Enabled, Frequency: &input.Frequency})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, updated)
}

func (r *Router) handleToggleSchedule(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	id, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input scheduleToggleInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	updated, err := r.schedule.SetEnabled(req.Context(), id, input.Enabled)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, updated)
}

func (r *Router) handleRunScheduleNow(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	id, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	result, err := r.schedule.RunNow(req.Context(), id)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusAccepted, result)
}

func (r *Router) handleListScheduleHistory(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	id, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	history, err := r.schedule.ListHistory(req.Context(), id, 20)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, history)
}
