package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

const (
	StatusQueued          = "queued"
	StatusRunning         = "running"
	StatusCancelRequested = "cancel_requested"
	StatusCancelled       = "cancelled"
	StatusCompleted       = "completed"
	StatusFailed          = "failed"
)

var ErrNoAvailableJob = errors.New("no available job")

type Service struct {
	db *gorm.DB
}

var jobKindPriority = map[string]int{
	"sync_library":                       0,
	"targeted_refresh":                   0,
	"listener_reconcile":                 0,
	"apply_storage_event_refresh":        0,
	"missing_media_cleanup":              0,
	"catalog_refresh_item_projection":    1,
	"catalog_refresh_library_projection": 1,
	"catalog_match_batch":                2,
	"inventory_probe_batch":              2,
	"match_catalog_item":                 3,
	"probe_inventory_file":               3,
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Enqueue(ctx context.Context, kind string, payload any) (database.Job, error) {
	return s.EnqueueUnique(ctx, kind, "", payload)
}

func (s *Service) EnqueueUnique(ctx context.Context, kind, jobKey string, payload any) (database.Job, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return database.Job{}, fmt.Errorf("marshal job payload: %w", err)
	}

	if jobKey != "" {
		var existing database.Job
		err := s.db.WithContext(ctx).
			Where("job_key = ? AND status IN ?", jobKey, []string{StatusQueued, StatusRunning}).
			Order("id desc").
			First(&existing).Error
		if err == nil {
			return existing, nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return database.Job{}, err
		}
	}

	job := database.Job{
		JobKey:      jobKey,
		Kind:        kind,
		Status:      StatusQueued,
		PayloadJSON: string(payloadJSON),
		AvailableAt: time.Now().UTC(),
	}

	if err := s.db.WithContext(ctx).Create(&job).Error; err != nil {
		return database.Job{}, err
	}

	return job, nil
}

func (s *Service) List(ctx context.Context, limit int, offset int, status string, kind string) ([]database.Job, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := s.db.WithContext(ctx).Order("created_at desc").Limit(limit).Offset(offset)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if kind != "" {
		query = query.Where("kind = ?", kind)
	}

	var jobs []database.Job
	if err := query.Find(&jobs).Error; err != nil {
		return nil, err
	}

	return jobs, nil
}

func (s *Service) Retry(ctx context.Context, jobID uint) (database.Job, error) {
	var existing database.Job
	if err := s.db.WithContext(ctx).First(&existing, jobID).Error; err != nil {
		return database.Job{}, err
	}
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).
		Model(&database.Job{}).
		Where("id = ?", jobID).
		Updates(map[string]any{
			"status":        StatusQueued,
			"error_message": "",
			"available_at":  now,
			"started_at":    nil,
			"finished_at":   nil,
		}).Error; err != nil {
		return database.Job{}, err
	}

	if err := s.db.WithContext(ctx).First(&existing, jobID).Error; err != nil {
		return database.Job{}, err
	}
	return existing, nil
}

func (s *Service) Cancel(ctx context.Context, jobID uint) (database.Job, error) {
	var existing database.Job
	if err := s.db.WithContext(ctx).First(&existing, jobID).Error; err != nil {
		return database.Job{}, err
	}

	now := time.Now().UTC()
	updates := map[string]any{}
	switch existing.Status {
	case StatusQueued:
		updates["status"] = StatusCancelled
		updates["finished_at"] = now
		updates["error_message"] = "cancelled by administrator"
	case StatusRunning, StatusCancelRequested:
		updates["status"] = StatusCancelRequested
		updates["error_message"] = "cancellation requested by administrator"
	default:
		return database.Job{}, fmt.Errorf("job status %q cannot be cancelled", existing.Status)
	}

	if err := s.db.WithContext(ctx).Model(&database.Job{}).Where("id = ?", jobID).Updates(updates).Error; err != nil {
		return database.Job{}, err
	}
	if err := s.db.WithContext(ctx).First(&existing, jobID).Error; err != nil {
		return database.Job{}, err
	}
	return existing, nil
}

func (s *Service) CancellationRequested(ctx context.Context, jobID uint) (bool, error) {
	var job database.Job
	if err := s.db.WithContext(ctx).Select("status").First(&job, jobID).Error; err != nil {
		return false, err
	}
	return job.Status == StatusCancelRequested || job.Status == StatusCancelled, nil
}

func (s *Service) Cancelled(ctx context.Context, jobID uint) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).
		Model(&database.Job{}).
		Where("id = ? AND status IN ?", jobID, []string{StatusRunning, StatusCancelRequested}).
		Updates(map[string]any{
			"status":        StatusCancelled,
			"finished_at":   now,
			"error_message": "cancelled by administrator",
		}).Error
}

func (s *Service) ClaimNext(ctx context.Context) (database.Job, error) {
	now := time.Now().UTC()
	for range 3 {
		job, err := s.claimNextOnce(ctx, now)
		if errors.Is(err, ErrNoAvailableJob) {
			return database.Job{}, err
		}
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			continue
		}
		return job, err
	}
	return database.Job{}, ErrNoAvailableJob
}

func (s *Service) Complete(ctx context.Context, jobID uint) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).
		Model(&database.Job{}).
		Where("id = ?", jobID).
		Updates(map[string]any{
			"status":        StatusCompleted,
			"finished_at":   now,
			"error_message": "",
		}).Error
}

func (s *Service) Fail(ctx context.Context, jobID uint, err error) error {
	now := time.Now().UTC()
	message := "job failed"
	if err != nil {
		message = err.Error()
	}

	return s.db.WithContext(ctx).
		Model(&database.Job{}).
		Where("id = ?", jobID).
		Updates(map[string]any{
			"status":        StatusFailed,
			"finished_at":   now,
			"error_message": message,
		}).Error
}

func (s *Service) claimNextOnce(ctx context.Context, now time.Time) (database.Job, error) {
	var claimed database.Job

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var job database.Job
		query := tx.
			Where("status = ? AND available_at <= ?", StatusQueued, now).
			Order(jobPriorityOrderClause()).
			Order("available_at asc, id asc")
		if err := query.First(&job).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNoAvailableJob
			}
			return err
		}

		result := tx.Model(&database.Job{}).
			Where("id = ? AND status = ?", job.ID, StatusQueued).
			Updates(map[string]any{
				"status":      StatusRunning,
				"started_at":  now,
				"finished_at": nil,
				"attempts":    gorm.Expr("attempts + 1"),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrDuplicatedKey
		}

		if err := tx.First(&claimed, job.ID).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return database.Job{}, err
	}

	return claimed, nil
}

func jobPriorityOrderClause() string {
	clause := "CASE kind"
	for kind, priority := range jobKindPriority {
		clause += fmt.Sprintf(" WHEN '%s' THEN %d", kind, priority)
	}
	return clause + " ELSE 4 END asc"
}
