package httpapi

import "net/http"

func (r *Router) registerAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", r.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", r.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", r.handleLogout)
	mux.HandleFunc("GET /api/v1/auth/sessions", r.handleListAuthSessions)
	mux.HandleFunc("DELETE /api/v1/auth/sessions/others", r.handleRevokeOtherAuthSessions)
	mux.HandleFunc("DELETE /api/v1/auth/sessions/{id}", r.handleRevokeAuthSession)
	mux.HandleFunc("GET /api/v1/me", r.handleMe)
	mux.HandleFunc("POST /api/v1/me/progress", r.handleUpdateProgress)
	mux.HandleFunc("POST /api/v1/me/preferred-resource", r.handleSetPreferredResource)
	mux.HandleFunc("GET /api/v1/me/progress-frames/{id}/{name}", r.handleGetProgressFrame)
	mux.HandleFunc("GET /api/v1/me/continue-watching", r.handleContinueWatching)
	mux.HandleFunc("GET /api/v1/me/recently-played", r.handleRecentlyPlayed)
	mux.HandleFunc("GET /api/v1/me/favorites", r.handleListFavorites)
	mux.HandleFunc("POST /api/v1/me/favorites/{id}", r.handleAddFavorite)
	mux.HandleFunc("DELETE /api/v1/me/favorites/{id}", r.handleRemoveFavorite)
}

func (r *Router) registerHomeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/home/media-overview", r.handleHomeMediaOverview)
	mux.HandleFunc("GET /api/v1/home/sections", r.handleHomeContentSections)
	mux.HandleFunc("GET /api/v1/home/recently-added", r.handleRecentlyAdded)
}

func (r *Router) registerHealthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health/summary", r.handleHealthSummary)
	mux.HandleFunc("GET /api/v1/health/issues", r.handleHealthIssues)
	mux.HandleFunc("POST /api/v1/health/issues/{id}/ignore", r.handleIgnoreHealthIssue)
	mux.HandleFunc("POST /api/v1/health/issues/{id}/rescan", r.handleRescanHealthIssueLibraries)
	mux.HandleFunc("POST /api/v1/media-sources/{id}/validate", r.handleValidateMediaSource)
}
