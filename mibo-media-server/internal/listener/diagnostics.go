package listener

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
)

type StorageChangeDiagnostic struct {
	LibraryID                 uint       `json:"library_id"`
	LibraryName               string     `json:"library_name"`
	StorageProvider           string     `json:"storage_provider"`
	ObserverMode              string     `json:"observer_mode"`
	ObserverEnabled           bool       `json:"observer_enabled"`
	LastSuccessfulObservation *time.Time `json:"last_successful_observation,omitempty"`
	LastReconcileAt           *time.Time `json:"last_reconcile_at,omitempty"`
	PendingPlanCount          int64      `json:"pending_plan_count"`
	RecentFailureSummary      string     `json:"recent_failure_summary,omitempty"`
}

func (s *Service) Diagnostics(ctx context.Context) ([]StorageChangeDiagnostic, error) {
	var records []struct {
		database.Library
		Provider string
	}
	if err := s.db.WithContext(ctx).
		Table("libraries").
		Select("libraries.*, media_sources.provider").
		Joins("JOIN media_sources ON media_sources.id = libraries.media_source_id").
		Where("libraries.status = ? AND libraries.scanner_enabled = ?", "active", true).
		Order("libraries.id asc").
		Scan(&records).Error; err != nil {
		return nil, err
	}
	result := make([]StorageChangeDiagnostic, 0, len(records))
	for _, record := range records {
		diag := StorageChangeDiagnostic{
			LibraryID:       record.ID,
			LibraryName:     record.Name,
			StorageProvider: record.Provider,
			ObserverMode:    observerModeForProvider(record.Provider),
			ObserverEnabled: observerEnabledForProvider(record.Provider),
		}
		lastObservation, err := s.lastSuccessfulObservation(ctx, record.ID)
		if err != nil {
			return nil, err
		}
		diag.LastSuccessfulObservation = lastObservation
		diag.LastReconcileAt = lastObservation
		pending, err := s.pendingRefreshCount(ctx, record.ID)
		if err != nil {
			return nil, err
		}
		diag.PendingPlanCount = pending
		failure, err := s.recentFailureSummary(ctx, record.ID)
		if err != nil {
			return nil, err
		}
		diag.RecentFailureSummary = failure
		result = append(result, diag)
	}
	return result, nil
}

func (s *Service) lastSuccessfulObservation(ctx context.Context, libraryID uint) (*time.Time, error) {
	var entry database.StorageIndexEntry
	err := s.db.WithContext(ctx).
		Where("library_id = ? AND observation_status = ?", libraryID, "present").
		Order("last_observed_at desc").
		First(&entry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entry.LastObservedAt, nil
}

func (s *Service) pendingRefreshCount(ctx context.Context, libraryID uint) (int64, error) {
	var count int64
	patterns := []string{
		fmt.Sprintf(`%%"library_id":%d,%%`, libraryID),
		fmt.Sprintf(`%%"library_id":%d}%%`, libraryID),
	}
	err := s.db.WithContext(ctx).Model(&database.WorkflowRun{}).
		Where("status IN ?", []string{workflow.RunStatusQueued, workflow.RunStatusRunning}).
		Where("reason IN ?", []string{"storage_refresh", "targeted_refresh"}).
		Where("payload_json LIKE ? OR payload_json LIKE ?", patterns[0], patterns[1]).
		Count(&count).Error
	return count, err
}

func (s *Service) recentFailureSummary(ctx context.Context, libraryID uint) (string, error) {
	var failure database.StorageObservationFailure
	err := s.db.WithContext(ctx).
		Where("library_id = ?", libraryID).
		Order("observed_at desc, id desc").
		First(&failure).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	if strings.TrimSpace(failure.ErrorMessage) == "" {
		return failure.Reason, nil
	}
	return failure.Reason + ": " + failure.ErrorMessage, nil
}

func observerModeForProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "local":
		return "local_watcher_with_reconcile"
	case "openlist":
		return "openlist_polling_with_reconcile"
	default:
		return "manual_and_scheduled_scan"
	}
}

func observerEnabledForProvider(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "local", "openlist":
		return true
	default:
		return false
	}
}
