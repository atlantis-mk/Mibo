package httpapi

import "net/http"

func (r *Router) registerAdminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admin/console", r.handleAdminConsoleSummary)
	mux.HandleFunc("GET /api/v1/admin/ingest/diagnostics", r.handleAdminIngestDiagnostics)
	mux.HandleFunc("POST /api/v1/admin/ingest/reconcile", r.handleAdminIngestReconcile)
	mux.HandleFunc("POST /api/v1/admin/ingest/stages/{id}/retry", r.handleAdminRetryIngestStage)
	mux.HandleFunc("POST /api/v1/admin/ingest/stages/{id}/resolve-review", r.handleAdminResolveIngestReviewStage)
	mux.HandleFunc("POST /api/v1/admin/console/actions/scan-libraries", r.handleAdminConsoleScanLibraries)
	mux.HandleFunc("GET /api/v1/admin/users", r.handleListAdminUsers)
	mux.HandleFunc("POST /api/v1/admin/users", r.handleCreateAdminUser)
	mux.HandleFunc("GET /api/v1/admin/logs", r.handleListAdminLogs)
	mux.HandleFunc("GET /api/v1/admin/logs/{name}", r.handleGetAdminLog)
	mux.HandleFunc("GET /api/v1/admin/logs/{name}/download", r.handleDownloadAdminLog)
	mux.HandleFunc("DELETE /api/v1/admin/logs/{name}", r.handleDeleteAdminLog)
}

func (r *Router) registerSettingsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/settings/metadata/providers", r.handleListMetadataProviderInstances)
	mux.HandleFunc("POST /api/v1/settings/metadata/providers", r.handleCreateMetadataProviderInstance)
	mux.HandleFunc("PATCH /api/v1/settings/metadata/providers/{provider_id}", r.handleUpdateMetadataProviderInstance)
	mux.HandleFunc("GET /api/v1/settings/metadata/profiles", r.handleListMetadataProfiles)
	mux.HandleFunc("POST /api/v1/settings/metadata/profiles", r.handleCreateMetadataProfile)
	mux.HandleFunc("PATCH /api/v1/settings/metadata/profiles/{profile_id}", r.handleUpdateMetadataProfile)
	mux.HandleFunc("GET /api/v1/settings/network", r.handleGetNetworkSettings)
	mux.HandleFunc("PUT /api/v1/settings/network", r.handleUpdateNetworkSettings)
	mux.HandleFunc("GET /api/v1/settings/scan", r.handleGetScanSettings)
	mux.HandleFunc("PUT /api/v1/settings/scan", r.handleUpdateScanSettings)
}
