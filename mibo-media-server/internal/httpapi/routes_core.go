package httpapi

import "net/http"

func (r *Router) registerSystemRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", r.handleHealth)
	mux.HandleFunc("GET /readyz", r.handleReady)
	mux.HandleFunc("GET /api/v1/system/info", r.handleSystemInfo)
}

func (r *Router) registerSetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/setup/status", r.handleSetupStatus)
}

func (r *Router) registerSearchRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/discovery", r.handleDiscoverMedia)
	mux.HandleFunc("GET /api/v1/search/history", r.handleListSearchHistory)
}

func (r *Router) registerWorkflowRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/workflows", r.handleListWorkflows)
	mux.HandleFunc("GET /api/v1/workflows/{id}", r.handleGetWorkflow)
	mux.HandleFunc("GET /api/v1/workflows/diagnostics", r.handleWorkflowDiagnostics)
}

func (r *Router) registerScheduleRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/schedules", r.handleListSchedules)
	mux.HandleFunc("POST /api/v1/schedules", r.handleCreateSchedule)
	mux.HandleFunc("GET /api/v1/schedules/{id}", r.handleGetSchedule)
	mux.HandleFunc("PATCH /api/v1/schedules/{id}", r.handleUpdateSchedule)
	mux.HandleFunc("POST /api/v1/schedules/{id}/toggle", r.handleToggleSchedule)
	mux.HandleFunc("POST /api/v1/schedules/{id}/run", r.handleRunScheduleNow)
	mux.HandleFunc("GET /api/v1/schedules/{id}/history", r.handleListScheduleHistory)
}
