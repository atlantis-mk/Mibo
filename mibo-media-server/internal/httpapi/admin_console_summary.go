package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/workflow"
)

func (r *Router) newAdminConsoleSummary(req *http.Request) adminConsoleSummaryResponse {
	return adminConsoleSummaryResponse{
		Server: adminConsoleServerSummary{
			Name:            "Mibo",
			Service:         "mibo-media-server",
			Status:          "ok",
			Version:         "unknown",
			UpdateStatus:    "unknown",
			APIAddress:      r.cfg.HTTP.Addr,
			Port:            configuredPort(r.cfg.HTTP.Addr),
			UptimeSeconds:   int64(time.Since(serverStartedAt).Seconds()),
			StorageProvider: "暂无",
			StorageRoot:     "",
			DatabaseDriver:  r.cfg.Database.Driver,
		},
		Access: adminConsoleAccessSummary{Addresses: buildAdminConsoleAccessAddresses(req)},
		Health: adminConsoleHealthSummary{
			Database: adminConsoleSectionStatus{Status: "ok", Message: r.cfg.Database.Driver},
			Storage:  adminConsoleSectionStatus{Status: "unknown"},
			Modules: []adminConsoleModuleStatus{
				{Name: "auth", Status: "ok"},
				{Name: "library", Status: "ok"},
				{Name: "jobs", Status: "ok"},
				{Name: "worker", Status: boolStatus(r.cfg.Worker.Enabled), Message: enabledMessage(r.cfg.Worker.Enabled)},
				{Name: "metadata", Status: "ok"},
				{Name: "playback", Status: "ok"},
			},
		},
		QuickAction: buildAdminConsoleQuickActions(),
		Devices:     []adminConsoleDeviceSummary{},
		Activity:    []adminConsoleActivityEvent{},
		Warnings:    []adminConsoleSectionWarning{},
	}
}

func (r *Router) enrichAdminConsoleHealth(ctx context.Context, summary *adminConsoleSummaryResponse) {
	if summary == nil {
		return
	}
	if sqlDB, err := r.db.DB(); err != nil {
		summary.Health.Database = adminConsoleSectionStatus{Status: "warning", Message: err.Error()}
		summary.Warnings = append(summary.Warnings, adminConsoleSectionWarning{Section: "database", Message: err.Error()})
	} else if err := sqlDB.PingContext(ctx); err != nil {
		summary.Health.Database = adminConsoleSectionStatus{Status: "warning", Message: err.Error()}
		summary.Warnings = append(summary.Warnings, adminConsoleSectionWarning{Section: "database", Message: err.Error()})
	}
	if providers, err := configuredMediaSourceProviders(ctx, r); err != nil {
		summary.Health.Storage = adminConsoleSectionStatus{Status: "warning", Message: err.Error()}
		summary.Warnings = append(summary.Warnings, adminConsoleSectionWarning{Section: "storage", Message: err.Error()})
	} else if len(providers) == 0 {
		summary.Health.Storage = adminConsoleSectionStatus{Status: "not_configured", Message: "暂无"}
	} else {
		providerSummary := strings.Join(providers, ", ")
		summary.Server.StorageProvider = providerSummary
		summary.Health.Storage = adminConsoleSectionStatus{Status: "ok", Message: providerSummary}
	}
}

func (r *Router) enrichAdminConsoleMedia(ctx context.Context, summary *adminConsoleSummaryResponse) {
	if summary == nil {
		return
	}
	countModel(ctx, r, &database.Library{}, &summary.Media.Libraries, &summary.Warnings, "libraries")
	countModel(ctx, r, &database.MediaSource{}, &summary.Media.MediaSources, &summary.Warnings, "media_sources")
	countModel(ctx, r, &database.LibraryMetadataProjection{}, &summary.Media.MetadataItems, &summary.Warnings, "metadata_items")
	countModel(ctx, r, &database.InventoryFile{}, &summary.Media.InventoryFiles, &summary.Warnings, "inventory_files")
	countModel(ctx, r, &database.Person{}, &summary.Media.People, &summary.Warnings, "people")
	countWhere(ctx, r, &database.LibraryMetadataProjection{}, "item_type = ? AND hidden = ?", []any{database.MetadataItemTypeMovie, false}, &summary.Media.Movies, &summary.Warnings, "movies")
	countWhere(ctx, r, &database.LibraryMetadataProjection{}, "item_type = ? AND hidden = ?", []any{database.MetadataItemTypeSeries, false}, &summary.Media.Series, &summary.Warnings, "series")
	countWhere(ctx, r, &database.LibraryMetadataProjection{}, "item_type = ? AND hidden = ?", []any{database.MetadataItemTypeEpisode, false}, &summary.Media.Episodes, &summary.Warnings, "episodes")
	countWhere(ctx, r, &database.WorkflowRun{}, "status IN ?", []any{[]string{workflow.RunStatusQueued, workflow.RunStatusRunning}}, &summary.Media.ActiveJobs, &summary.Warnings, "active_workflows")
	countWhere(ctx, r, &database.WorkflowRun{}, "status = ?", []any{workflow.RunStatusFailed}, &summary.Media.FailedJobs, &summary.Warnings, "failed_workflows")
	countModel(ctx, r, &database.Schedule{}, &summary.Media.Schedules, &summary.Warnings, "schedules")
	countWhere(ctx, r, &database.Schedule{}, "enabled = ?", []any{true}, &summary.Media.EnabledSchedules, &summary.Warnings, "enabled_schedules")
	if r.ingest != nil {
		if diagnostics, err := r.ingest.Diagnostics(ctx, ingest.DiagnosticsInput{Status: "all", Limit: 500}); err == nil {
			summary.Media.Ingest = adminConsoleIngestSummary{Organizing: diagnostics.Summary.Organizing, Failed: diagnostics.Summary.Failed, Stale: diagnostics.Summary.Stale, ReviewRequired: diagnostics.Summary.ReviewRequired, RetryEligible: diagnostics.Summary.RetryEligible}
		} else {
			summary.Warnings = append(summary.Warnings, adminConsoleSectionWarning{Section: "ingest", Message: err.Error()})
		}
	}
}

func configuredMediaSourceProviders(ctx context.Context, r *Router) ([]string, error) {
	providers := []string{}
	if err := r.db.WithContext(ctx).Model(&database.MediaSource{}).Distinct("provider").Order("provider asc").Pluck("provider", &providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}

func countModel(ctx context.Context, r *Router, model any, target *int64, warnings *[]adminConsoleSectionWarning, section string) {
	if !r.db.Migrator().HasTable(model) {
		return
	}
	if err := r.db.WithContext(ctx).Model(model).Count(target).Error; err != nil {
		*warnings = append(*warnings, adminConsoleSectionWarning{Section: section, Message: err.Error()})
	}
}

func countWhere(ctx context.Context, r *Router, model any, query string, args []any, target *int64, warnings *[]adminConsoleSectionWarning, section string) {
	if !r.db.Migrator().HasTable(model) {
		return
	}
	if err := r.db.WithContext(ctx).Model(model).Where(query, args...).Count(target).Error; err != nil {
		*warnings = append(*warnings, adminConsoleSectionWarning{Section: section, Message: err.Error()})
	}
}
