package httpapi

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
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
	Libraries        int64 `json:"libraries"`
	MediaSources     int64 `json:"media_sources"`
	CatalogItems     int64 `json:"catalog_items"`
	InventoryFiles   int64 `json:"inventory_files"`
	Movies           int64 `json:"movies"`
	Series           int64 `json:"series"`
	Episodes         int64 `json:"episodes"`
	People           int64 `json:"people"`
	ActiveJobs       int64 `json:"active_jobs"`
	FailedJobs       int64 `json:"failed_jobs"`
	Schedules        int64 `json:"schedules"`
	EnabledSchedules int64 `json:"enabled_schedules"`
	Warnings         int64 `json:"warnings"`
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
	summary := adminConsoleSummaryResponse{
		Server: adminConsoleServerSummary{
			Name:            "Mibo",
			Service:         "mibo-media-server",
			Status:          "ok",
			Version:         "unknown",
			UpdateStatus:    "unknown",
			APIAddress:      r.cfg.HTTP.Addr,
			Port:            configuredPort(r.cfg.HTTP.Addr),
			UptimeSeconds:   int64(time.Since(serverStartedAt).Seconds()),
			StorageProvider: configuredStorageProvider(r.cfg),
			StorageRoot:     storageRootPath(r.cfg),
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
				{Name: "hls", Status: boolStatus(r.hls.Enabled()), Message: enabledMessage(r.hls.Enabled())},
			},
		},
		QuickAction: buildAdminConsoleQuickActions(),
		Devices:     []adminConsoleDeviceSummary{},
		Activity:    []adminConsoleActivityEvent{},
		Warnings:    []adminConsoleSectionWarning{},
	}

	if sqlDB, err := r.db.DB(); err != nil {
		summary.Health.Database = adminConsoleSectionStatus{Status: "warning", Message: err.Error()}
		summary.Warnings = append(summary.Warnings, adminConsoleSectionWarning{Section: "database", Message: err.Error()})
	} else if err := sqlDB.PingContext(ctx); err != nil {
		summary.Health.Database = adminConsoleSectionStatus{Status: "warning", Message: err.Error()}
		summary.Warnings = append(summary.Warnings, adminConsoleSectionWarning{Section: "database", Message: err.Error()})
	}

	if provider, err := r.storage.Get(configuredStorageProvider(r.cfg)); err != nil {
		summary.Health.Storage = adminConsoleSectionStatus{Status: "warning", Message: err.Error()}
		summary.Warnings = append(summary.Warnings, adminConsoleSectionWarning{Section: "storage", Message: err.Error()})
	} else if _, err := provider.ResolveStorage(ctx, storage.ResolveStorageRequest{Path: storageRootPath(r.cfg)}); err != nil {
		summary.Health.Storage = adminConsoleSectionStatus{Status: "warning", Message: err.Error()}
		summary.Warnings = append(summary.Warnings, adminConsoleSectionWarning{Section: "storage", Message: err.Error()})
	} else {
		summary.Health.Storage = adminConsoleSectionStatus{Status: "ok", Message: provider.Name()}
	}

	countModel(ctx, r, &database.Library{}, &summary.Media.Libraries, &summary.Warnings, "libraries")
	countModel(ctx, r, &database.MediaSource{}, &summary.Media.MediaSources, &summary.Warnings, "media_sources")
	countModel(ctx, r, &database.CatalogItem{}, &summary.Media.CatalogItems, &summary.Warnings, "catalog_items")
	countModel(ctx, r, &database.InventoryFile{}, &summary.Media.InventoryFiles, &summary.Warnings, "inventory_files")
	countModel(ctx, r, &database.Person{}, &summary.Media.People, &summary.Warnings, "people")
	countWhere(ctx, r, &database.CatalogItem{}, "type = ?", []any{"movie"}, &summary.Media.Movies, &summary.Warnings, "movies")
	countWhere(ctx, r, &database.CatalogItem{}, "type = ?", []any{"series"}, &summary.Media.Series, &summary.Warnings, "series")
	countWhere(ctx, r, &database.CatalogItem{}, "type = ?", []any{"episode"}, &summary.Media.Episodes, &summary.Warnings, "episodes")
	countWhere(ctx, r, &database.Job{}, "status IN ?", []any{[]string{"queued", "running"}}, &summary.Media.ActiveJobs, &summary.Warnings, "active_jobs")
	countWhere(ctx, r, &database.Job{}, "status = ?", []any{"failed"}, &summary.Media.FailedJobs, &summary.Warnings, "failed_jobs")
	countModel(ctx, r, &database.Schedule{}, &summary.Media.Schedules, &summary.Warnings, "schedules")
	countWhere(ctx, r, &database.Schedule{}, "enabled = ?", []any{true}, &summary.Media.EnabledSchedules, &summary.Warnings, "enabled_schedules")
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
	jobs := make([]database.Job, 0, len(libraries))
	for _, libraryRecord := range libraries {
		job, err := r.library.QueueLibraryScan(req.Context(), libraryRecord.ID)
		if err != nil {
			writeError(req.Context(), w, http.StatusBadRequest, err)
			return
		}
		jobs = append(jobs, job)
	}
	writeJSON(req.Context(), w, http.StatusAccepted, map[string]any{"queued": len(jobs), "jobs": jobs})
}

func (r *Router) handleAdminConsoleCatalogConsistency(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, fmt.Errorf("catalog service unavailable"))
		return
	}
	report, err := r.catalog.CheckConsistency(req.Context(), nil)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, report)
}

func (r *Router) handleAdminConsoleRebuildProjections(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, fmt.Errorf("catalog service unavailable"))
		return
	}
	result, err := r.catalog.RebuildDerivedData(req.Context(), nil)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}

func buildAdminConsoleAccessAddresses(req *http.Request) []adminConsoleAccessAddress {
	baseURL := requestBaseURL(req)
	addresses := []adminConsoleAccessAddress{
		{Kind: "local", Label: "本机访问", URL: baseURL, Status: "available", Copyable: true},
	}
	for _, ip := range lanIPv4Addresses() {
		addresses = append(addresses, adminConsoleAccessAddress{Kind: "lan", Label: "局域网访问", URL: replaceURLHost(baseURL, ip), Status: "available", Copyable: true})
	}
	if len(addresses) == 1 {
		addresses = append(addresses, adminConsoleAccessAddress{Kind: "lan", Label: "局域网访问", Status: "unavailable", Message: "未发现可用局域网地址", Copyable: false})
	}
	addresses = append(addresses, adminConsoleAccessAddress{Kind: "remote", Label: "远程访问", Status: "not_configured", Route: "/settings", Message: "未配置", Copyable: false})
	return addresses
}

func buildAdminConsoleQuickActions() []adminConsoleQuickAction {
	return []adminConsoleQuickAction{
		{ID: "open-settings", Label: "打开设置", Description: "进入现有设置区域", Kind: "route", Route: "/settings", Risk: "safe"},
		{ID: "open-libraries", Label: "媒体库管理", Description: "管理媒体库与来源", Kind: "route", Route: "/settings/library", Risk: "safe"},
		{ID: "scan-libraries", Label: "扫描媒体库", Description: "为所有媒体库排队扫描任务", Kind: "mutation", Method: "POST", Endpoint: "/api/v1/admin/console/actions/scan-libraries", Risk: "expensive", Confirm: true},
		{ID: "catalog-consistency", Label: "一致性检查", Description: "检查目录投影和库存关系", Kind: "mutation", Method: "POST", Endpoint: "/api/v1/admin/console/actions/catalog-consistency", Risk: "expensive", Confirm: true},
		{ID: "rebuild-projections", Label: "重建投影", Description: "重建目录派生数据", Kind: "mutation", Method: "POST", Endpoint: "/api/v1/admin/console/actions/rebuild-projections", Risk: "danger", Confirm: true},
		{ID: "open-logs", Label: "查看日志", Description: "日志查看尚未实现", Kind: "unsupported", Disabled: true, DisabledReason: "日志页面尚未实现", Risk: "safe"},
		{ID: "shutdown", Label: "关闭服务器", Description: "服务器生命周期控制尚未实现", Kind: "unsupported", Disabled: true, DisabledReason: "未提供安全关闭接口", Risk: "danger"},
	}
}

func (r *Router) buildAdminConsoleActivity(ctx context.Context, warnings *[]adminConsoleSectionWarning) []adminConsoleActivityEvent {
	events := []adminConsoleActivityEvent{}
	var jobs []database.Job
	if err := r.db.WithContext(ctx).Order("updated_at desc").Limit(6).Find(&jobs).Error; err != nil {
		*warnings = append(*warnings, adminConsoleSectionWarning{Section: "activity", Message: err.Error()})
		return events
	}
	for _, job := range jobs {
		severity := "info"
		if job.Status == "failed" {
			severity = "error"
		} else if job.Status == "running" || job.Status == "queued" {
			severity = "warning"
		}
		message := fmt.Sprintf("%s 任务状态：%s", job.Kind, job.Status)
		if strings.TrimSpace(job.ErrorMessage) != "" {
			message = message + " - " + job.ErrorMessage
		}
		events = append(events, adminConsoleActivityEvent{ID: fmt.Sprintf("job-%d", job.ID), Type: "job", Severity: severity, Message: message, Timestamp: job.UpdatedAt.Format(time.RFC3339)})
	}
	var progressRows []struct {
		ID           uint
		UpdatedAt    time.Time
		Username     string
		Title        string
		PositionSecs int
	}
	if err := r.db.WithContext(ctx).Table("user_item_data").Select("user_item_data.id, user_item_data.updated_at, users.username, catalog_items.title, user_item_data.position_seconds").Joins("left join users on users.id = user_item_data.user_id").Joins("left join catalog_items on catalog_items.id = user_item_data.item_id").Order("user_item_data.updated_at desc").Limit(6).Scan(&progressRows).Error; err != nil {
		*warnings = append(*warnings, adminConsoleSectionWarning{Section: "activity", Message: err.Error()})
		return events
	}
	for _, row := range progressRows {
		events = append(events, adminConsoleActivityEvent{ID: fmt.Sprintf("progress-%d", row.ID), Type: "playback", Severity: "info", Message: "播放进度已更新", User: row.Username, MediaTitle: row.Title, Timestamp: row.UpdatedAt.Format(time.RFC3339)})
	}
	return events
}

func adminConsoleDevicesFromActivity(events []adminConsoleActivityEvent) []adminConsoleDeviceSummary {
	devices := []adminConsoleDeviceSummary{}
	for _, event := range events {
		if event.Device == "" {
			continue
		}
		devices = append(devices, adminConsoleDeviceSummary{ID: event.Device, Name: event.Device, User: event.User, State: event.Type, MediaTitle: event.MediaTitle, LastSeenAt: event.Timestamp})
	}
	return devices
}

func countModel(ctx context.Context, r *Router, model any, target *int64, warnings *[]adminConsoleSectionWarning, section string) {
	if err := r.db.WithContext(ctx).Model(model).Count(target).Error; err != nil {
		*warnings = append(*warnings, adminConsoleSectionWarning{Section: section, Message: err.Error()})
	}
}

func countWhere(ctx context.Context, r *Router, model any, query string, args []any, target *int64, warnings *[]adminConsoleSectionWarning, section string) {
	if err := r.db.WithContext(ctx).Model(model).Where(query, args...).Count(target).Error; err != nil {
		*warnings = append(*warnings, adminConsoleSectionWarning{Section: section, Message: err.Error()})
	}
}

func boolStatus(enabled bool) string {
	if enabled {
		return "ok"
	}
	return "unavailable"
}

func enabledMessage(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func configuredPort(addr string) int {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		trimmed := strings.TrimPrefix(strings.TrimSpace(addr), ":")
		parsed, _ := strconv.Atoi(trimmed)
		return parsed
	}
	parsed, _ := strconv.Atoi(port)
	return parsed
}

func replaceURLHost(baseURL, host string) string {
	scheme := "http://"
	value := strings.TrimPrefix(baseURL, "http://")
	if strings.HasPrefix(baseURL, "https://") {
		scheme = "https://"
		value = strings.TrimPrefix(baseURL, "https://")
	}
	_, port, err := net.SplitHostPort(value)
	if err != nil || port == "" {
		return scheme + host
	}
	return scheme + net.JoinHostPort(host, port)
}

func lanIPv4Addresses() []string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var results []string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch value := addr.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}
			if ip4 := ip.To4(); ip4 != nil {
				results = append(results, ip4.String())
			}
		}
	}
	return results
}
