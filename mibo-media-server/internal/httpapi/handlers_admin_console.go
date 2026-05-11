package httpapi

import (
	"net/http"
	"time"

	"github.com/atlan/mibo-media-server/internal/library"
)

var serverStartedAt = time.Now()

type adminConsoleSummaryResponse struct {
	Server      adminConsoleServerSummary    `json:"server"`
	Access      adminConsoleAccessSummary    `json:"access"`
	Media       adminConsoleMediaSummary     `json:"media"`
	Health      adminConsoleHealthSummary    `json:"health"`
	Devices     []adminConsoleDeviceSummary  `json:"devices"`
	QuickAction []adminConsoleQuickAction    `json:"quick_actions"`
	Activity    []adminConsoleActivityEvent  `json:"activity"`
	Warnings    []adminConsoleSectionWarning `json:"warnings"`
}

type adminConsoleServerSummary struct {
	Name            string `json:"name"`
	Service         string `json:"service"`
	Status          string `json:"status"`
	Version         string `json:"version"`
	UpdateStatus    string `json:"update_status"`
	APIAddress      string `json:"api_address"`
	Port            int    `json:"port"`
	UptimeSeconds   int64  `json:"uptime_seconds"`
	StorageProvider string `json:"storage_provider"`
	StorageRoot     string `json:"storage_root"`
	DatabaseDriver  string `json:"database_driver"`
}

type adminConsoleAccessAddress struct {
	Kind     string `json:"kind"`
	Label    string `json:"label"`
	URL      string `json:"url,omitempty"`
	Status   string `json:"status"`
	Route    string `json:"route,omitempty"`
	Message  string `json:"message,omitempty"`
	Copyable bool   `json:"copyable"`
}

type adminConsoleAccessSummary struct {
	Addresses []adminConsoleAccessAddress `json:"addresses"`
}

type adminConsoleMediaSummary struct {
	Libraries        int64                     `json:"libraries"`
	MediaSources     int64                     `json:"media_sources"`
	MetadataItems    int64                     `json:"metadata_items"`
	InventoryFiles   int64                     `json:"inventory_files"`
	Movies           int64                     `json:"movies"`
	Series           int64                     `json:"series"`
	Episodes         int64                     `json:"episodes"`
	People           int64                     `json:"people"`
	ActiveJobs       int64                     `json:"active_jobs"`
	FailedJobs       int64                     `json:"failed_jobs"`
	Schedules        int64                     `json:"schedules"`
	EnabledSchedules int64                     `json:"enabled_schedules"`
	Warnings         int64                     `json:"warnings"`
	Ingest           adminConsoleIngestSummary `json:"ingest"`
}

type adminConsoleIngestSummary struct {
	Organizing     int `json:"organizing"`
	Failed         int `json:"failed"`
	Stale          int `json:"stale"`
	ReviewRequired int `json:"review_required"`
	RetryEligible  int `json:"retry_eligible"`
}

type adminConsoleHealthSummary struct {
	Database adminConsoleSectionStatus  `json:"database"`
	Storage  adminConsoleSectionStatus  `json:"storage"`
	Modules  []adminConsoleModuleStatus `json:"modules"`
}

type adminConsoleSectionStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type adminConsoleModuleStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type adminConsoleDeviceSummary struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ClientType string `json:"client_type,omitempty"`
	User       string `json:"user,omitempty"`
	State      string `json:"state,omitempty"`
	MediaTitle string `json:"media_title,omitempty"`
	LastSeenAt string `json:"last_seen_at"`
}

type adminConsoleQuickAction struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Description    string `json:"description"`
	Kind           string `json:"kind"`
	Route          string `json:"route,omitempty"`
	Method         string `json:"method,omitempty"`
	Endpoint       string `json:"endpoint,omitempty"`
	Disabled       bool   `json:"disabled"`
	DisabledReason string `json:"disabled_reason,omitempty"`
	Risk           string `json:"risk"`
	Confirm        bool   `json:"confirm"`
}

type adminConsoleActivityEvent struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	User       string `json:"user,omitempty"`
	Device     string `json:"device,omitempty"`
	MediaTitle string `json:"media_title,omitempty"`
	Timestamp  string `json:"timestamp"`
}

type adminConsoleSectionWarning struct {
	Section string `json:"section"`
	Message string `json:"message"`
}

func (r *Router) handleAdminConsoleSummary(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	ctx := req.Context()
	summary := r.newAdminConsoleSummary(req)
	r.enrichAdminConsoleHealth(ctx, &summary)
	r.enrichAdminConsoleMedia(ctx, &summary)
	summary.Media.Warnings = int64(len(summary.Warnings)) + summary.Media.FailedJobs
	summary.Activity = r.buildAdminConsoleActivity(ctx, &summary.Warnings)
	summary.Devices = adminConsoleDevicesFromActivity(summary.Activity)

	writeJSON(ctx, w, http.StatusOK, summary)
}

func (r *Router) handleAdminConsoleScanLibraries(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	libraries, err := r.library.ListLibraries(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	queued := 0
	for _, libraryRecord := range libraries {
		if _, err := r.library.QueueLibraryScanWithReason(req.Context(), libraryRecord.ID, library.WorkflowReasonManualScan); err != nil {
			writeError(req.Context(), w, http.StatusBadRequest, err)
			return
		}
		queued++
	}
	writeJSON(req.Context(), w, http.StatusAccepted, map[string]any{"queued": queued})
}
