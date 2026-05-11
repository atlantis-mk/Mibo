package httpapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/workflow"
)

func (r *Router) buildAdminConsoleActivity(ctx context.Context, warnings *[]adminConsoleSectionWarning) []adminConsoleActivityEvent {
	events := []adminConsoleActivityEvent{}
	var runs []database.WorkflowRun
	if err := r.db.WithContext(ctx).Order("updated_at desc").Limit(6).Find(&runs).Error; err != nil {
		*warnings = append(*warnings, adminConsoleSectionWarning{Section: "activity", Message: err.Error()})
		return events
	}
	for _, run := range runs {
		severity := "info"
		if run.Status == workflow.RunStatusFailed {
			severity = "error"
		} else if run.Status == workflow.RunStatusRunning || run.Status == workflow.RunStatusQueued {
			severity = "warning"
		}
		message := fmt.Sprintf("%s workflow 状态：%s", run.Reason, run.Status)
		if strings.TrimSpace(run.ErrorMessage) != "" {
			message = message + " - " + run.ErrorMessage
		}
		events = append(events, adminConsoleActivityEvent{ID: fmt.Sprintf("workflow-%d", run.ID), Type: "workflow", Severity: severity, Message: message, Timestamp: run.UpdatedAt.Format(time.RFC3339)})
	}
	var progressRows []struct {
		ID           uint
		UpdatedAt    time.Time
		Username     string
		Title        string
		PositionSecs int
	}
	if r.db.Migrator().HasTable("user_metadata_data") {
		if err := r.db.WithContext(ctx).Table("user_metadata_data").Select("user_metadata_data.id, user_metadata_data.updated_at, users.username, metadata_items.title, user_metadata_data.position_seconds").Joins("left join users on users.id = user_metadata_data.user_id").Joins("left join metadata_items on metadata_items.id = user_metadata_data.metadata_item_id").Order("user_metadata_data.updated_at desc").Limit(6).Scan(&progressRows).Error; err != nil {
			*warnings = append(*warnings, adminConsoleSectionWarning{Section: "activity", Message: err.Error()})
			return events
		}
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
