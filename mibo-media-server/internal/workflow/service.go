package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	db       *gorm.DB
	createMu sync.Mutex
}

type CreateRunInput struct {
	RunKey    string
	LibraryID uint
	Reason    string
	Priority  int
	ScopeKey  string
	Payload   any
}

type CreateTaskInput struct {
	TaskKey          string
	TaskType         string
	Stage            string
	Priority         int
	ScopeKey         string
	Payload          any
	Resources        map[string]int
	DependsOnTaskIDs []uint
	AvailableAt      time.Time
	MaxAttempts      int
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CreateOrReuseRun(ctx context.Context, input CreateRunInput) (database.WorkflowRun, bool, error) {
	s.createMu.Lock()
	defer s.createMu.Unlock()

	if input.RunKey == "" {
		return database.WorkflowRun{}, false, errors.New("workflow run key is required")
	}
	if input.LibraryID == 0 {
		return database.WorkflowRun{}, false, errors.New("workflow library id is required")
	}
	if input.Reason == "" {
		return database.WorkflowRun{}, false, errors.New("workflow reason is required")
	}
	if input.ScopeKey == "" {
		input.ScopeKey = fmt.Sprintf("library:%d", input.LibraryID)
	}
	payloadJSON, err := encodeJSON(input.Payload)
	if err != nil {
		return database.WorkflowRun{}, false, err
	}

	var existing database.WorkflowRun
	err = s.db.WithContext(ctx).
		Where("run_key = ? AND status IN ?", input.RunKey, []string{RunStatusQueued, RunStatusRunning}).
		First(&existing).Error
	if err == nil {
		return existing, true, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return database.WorkflowRun{}, false, err
	}

	run := database.WorkflowRun{RunKey: input.RunKey, LibraryID: input.LibraryID, Reason: input.Reason, Status: RunStatusQueued, Priority: input.Priority, ScopeKey: input.ScopeKey, PayloadJSON: payloadJSON}
	if err := s.db.WithContext(ctx).Create(&run).Error; err != nil {
		return database.WorkflowRun{}, false, err
	}
	return run, false, nil
}

func (s *Service) CreateTask(ctx context.Context, run database.WorkflowRun, input CreateTaskInput) (database.WorkflowTask, error) {
	if run.ID == 0 {
		return database.WorkflowTask{}, errors.New("workflow run is required")
	}
	if input.TaskKey == "" {
		return database.WorkflowTask{}, errors.New("workflow task key is required")
	}
	if input.TaskType == "" {
		return database.WorkflowTask{}, errors.New("workflow task type is required")
	}
	if input.Stage == "" {
		return database.WorkflowTask{}, errors.New("workflow stage is required")
	}
	if input.ScopeKey == "" {
		input.ScopeKey = run.ScopeKey
	}
	if input.AvailableAt.IsZero() {
		input.AvailableAt = time.Now().UTC()
	}
	if input.MaxAttempts <= 0 {
		input.MaxAttempts = 3
	}
	payloadJSON, err := encodeJSON(input.Payload)
	if err != nil {
		return database.WorkflowTask{}, err
	}
	resourceJSON, err := encodeJSON(input.Resources)
	if err != nil {
		return database.WorkflowTask{}, err
	}
	status := TaskStatusQueued
	if len(input.DependsOnTaskIDs) > 0 {
		status = TaskStatusBlocked
	}

	var task database.WorkflowTask
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		task = database.WorkflowTask{RunID: run.ID, LibraryID: run.LibraryID, TaskKey: input.TaskKey, TaskType: input.TaskType, Stage: input.Stage, Status: status, Priority: input.Priority, ScopeKey: input.ScopeKey, PayloadJSON: payloadJSON, ResourceJSON: resourceJSON, BlockedBy: len(input.DependsOnTaskIDs), MaxAttempts: input.MaxAttempts, AvailableAt: input.AvailableAt}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		for _, dependencyID := range input.DependsOnTaskIDs {
			if dependencyID == 0 {
				continue
			}
			dep := database.WorkflowTaskDependency{TaskID: task.ID, DependsOnTaskID: dependencyID}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&dep).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return task, err
}

func (s *Service) StartTask(ctx context.Context, taskID uint, owner string, leaseUntil time.Time) (database.WorkflowTask, error) {
	if owner == "" {
		return database.WorkflowTask{}, errors.New("lease owner is required")
	}
	if leaseUntil.IsZero() {
		return database.WorkflowTask{}, errors.New("lease until is required")
	}
	now := time.Now().UTC()
	var task database.WorkflowTask
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&task, taskID).Error; err != nil {
			return err
		}
		if !CanTransitionTask(task.Status, TransitionStart) {
			return fmt.Errorf("task status %q cannot start", task.Status)
		}
		updates := map[string]any{"status": TaskStatusRunning, "attempts": gorm.Expr("attempts + 1"), "started_at": now, "lease_owner": owner, "lease_until": leaseUntil, "error_message": ""}
		if err := tx.Model(&database.WorkflowTask{}).Where("id = ?", taskID).Updates(updates).Error; err != nil {
			return err
		}
		lease := database.WorkflowTaskLease{TaskID: taskID, Owner: owner, LeaseUntil: leaseUntil}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "task_id"}}, DoUpdates: clause.AssignmentColumns([]string{"owner", "lease_until", "updated_at"})}).Create(&lease).Error; err != nil {
			return err
		}
		return tx.First(&task, taskID).Error
	})
	return task, err
}

func (s *Service) CompleteTask(ctx context.Context, taskID uint) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task database.WorkflowTask
		if err := tx.First(&task, taskID).Error; err != nil {
			return err
		}
		if !CanTransitionTask(task.Status, TransitionComplete) {
			return fmt.Errorf("task status %q cannot complete", task.Status)
		}
		if err := tx.Model(&database.WorkflowTask{}).Where("id = ?", taskID).Updates(map[string]any{"status": TaskStatusCompleted, "finished_at": now, "lease_owner": "", "lease_until": nil, "error_message": "", "resource_wait_key": ""}).Error; err != nil {
			return err
		}
		if err := tx.Where("task_id = ?", taskID).Delete(&database.WorkflowTaskLease{}).Error; err != nil {
			return err
		}
		if err := tx.Where("task_id = ?", taskID).Delete(&database.WorkflowResourceUsage{}).Error; err != nil {
			return err
		}
		if err := s.unlockDependents(tx, taskID); err != nil {
			return err
		}
		return s.refreshRunStatus(tx, task.RunID, now)
	})
}

func (s *Service) FailTask(ctx context.Context, taskID uint, cause error) error {
	message := "workflow task failed"
	if cause != nil {
		message = cause.Error()
	}
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task database.WorkflowTask
		if err := tx.First(&task, taskID).Error; err != nil {
			return err
		}
		if !CanTransitionTask(task.Status, TransitionFail) {
			return fmt.Errorf("task status %q cannot fail", task.Status)
		}
		if err := tx.Model(&database.WorkflowTask{}).Where("id = ?", taskID).Updates(map[string]any{"status": TaskStatusFailed, "finished_at": now, "lease_owner": "", "lease_until": nil, "error_message": message}).Error; err != nil {
			return err
		}
		if err := tx.Where("task_id = ?", taskID).Delete(&database.WorkflowTaskLease{}).Error; err != nil {
			return err
		}
		if err := tx.Where("task_id = ?", taskID).Delete(&database.WorkflowResourceUsage{}).Error; err != nil {
			return err
		}
		return s.refreshRunStatus(tx, task.RunID, now)
	})
}

func (s *Service) refreshRunStatus(tx *gorm.DB, runID uint, now time.Time) error {
	if runID == 0 {
		return nil
	}
	var run database.WorkflowRun
	if err := tx.First(&run, runID).Error; err != nil {
		return err
	}
	if IsTerminalRunStatus(run.Status) {
		return nil
	}
	var failed int64
	if err := tx.Model(&database.WorkflowTask{}).Where("run_id = ? AND status = ?", runID, TaskStatusFailed).Count(&failed).Error; err != nil {
		return err
	}
	if failed > 0 {
		return tx.Model(&database.WorkflowRun{}).Where("id = ?", runID).Updates(map[string]any{"status": RunStatusFailed, "finished_at": now, "updated_at": now}).Error
	}
	var active int64
	if err := tx.Model(&database.WorkflowTask{}).Where("run_id = ? AND status IN ?", runID, []string{TaskStatusBlocked, TaskStatusQueued, TaskStatusRunning, TaskStatusRetrying}).Count(&active).Error; err != nil {
		return err
	}
	if active == 0 {
		return tx.Model(&database.WorkflowRun{}).Where("id = ?", runID).Updates(map[string]any{"status": RunStatusCompleted, "finished_at": now, "updated_at": now}).Error
	}
	if run.Status == RunStatusQueued {
		return tx.Model(&database.WorkflowRun{}).Where("id = ?", runID).Updates(map[string]any{"status": RunStatusRunning, "started_at": now, "updated_at": now}).Error
	}
	return nil
}

func (s *Service) RetryTask(ctx context.Context, taskID uint, availableAt time.Time) error {
	if availableAt.IsZero() {
		availableAt = time.Now().UTC()
	}
	return s.db.WithContext(ctx).Model(&database.WorkflowTask{}).Where("id = ? AND status IN ?", taskID, []string{TaskStatusFailed, TaskStatusRunning}).Updates(map[string]any{"status": TaskStatusRetrying, "available_at": availableAt, "finished_at": nil, "lease_owner": "", "lease_until": nil, "error_message": ""}).Error
}

func (s *Service) CancelRun(ctx context.Context, runID uint, message string) error {
	if message == "" {
		message = "workflow cancelled"
	}
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&database.WorkflowRun{}).Where("id = ? AND status IN ?", runID, []string{RunStatusQueued, RunStatusRunning}).Updates(map[string]any{"status": RunStatusCancelled, "cancelled_at": now, "finished_at": now, "error_message": message}).Error; err != nil {
			return err
		}
		return tx.Model(&database.WorkflowTask{}).Where("run_id = ? AND status IN ?", runID, []string{TaskStatusBlocked, TaskStatusQueued, TaskStatusRetrying, TaskStatusRunning}).Updates(map[string]any{"status": TaskStatusCancelled, "finished_at": now, "error_message": message}).Error
	})
}

func (s *Service) SupersedeRun(ctx context.Context, runID uint, message string) error {
	if message == "" {
		message = "workflow superseded"
	}
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&database.WorkflowRun{}).Where("id = ? AND status IN ?", runID, []string{RunStatusQueued, RunStatusRunning}).Updates(map[string]any{"status": RunStatusSuperseded, "finished_at": now, "error_message": message}).Error; err != nil {
			return err
		}
		return tx.Model(&database.WorkflowTask{}).Where("run_id = ? AND status IN ?", runID, []string{TaskStatusBlocked, TaskStatusQueued, TaskStatusRetrying, TaskStatusRunning}).Updates(map[string]any{"status": TaskStatusSuperseded, "finished_at": now, "error_message": message}).Error
	})
}

func (s *Service) RenewLease(ctx context.Context, taskID uint, owner string, leaseUntil time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&database.WorkflowTask{}).Where("id = ? AND status = ? AND lease_owner = ?", taskID, TaskStatusRunning, owner).Updates(map[string]any{"lease_until": leaseUntil})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Model(&database.WorkflowTaskLease{}).Where("task_id = ? AND owner = ?", taskID, owner).Update("lease_until", leaseUntil).Error
	})
}

func (s *Service) RecoverExpiredLeases(ctx context.Context, now time.Time) (int64, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var recovered int64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&database.WorkflowTask{}).Where("status = ? AND lease_until IS NOT NULL AND lease_until < ?", TaskStatusRunning, now).Updates(map[string]any{"status": TaskStatusRetrying, "lease_owner": "", "lease_until": nil, "available_at": now, "resource_wait_key": ""})
		if result.Error != nil {
			return result.Error
		}
		recovered = result.RowsAffected
		if err := tx.Where("lease_until < ?", now).Delete(&database.WorkflowTaskLease{}).Error; err != nil {
			return err
		}
		return tx.Where("lease_until < ?", now).Delete(&database.WorkflowResourceUsage{}).Error
	})
	return recovered, err
}

func (s *Service) unlockDependents(tx *gorm.DB, completedTaskID uint) error {
	var dependencies []database.WorkflowTaskDependency
	if err := tx.Where("depends_on_task_id = ?", completedTaskID).Find(&dependencies).Error; err != nil {
		return err
	}
	for _, dependency := range dependencies {
		if err := tx.Model(&database.WorkflowTask{}).Where("id = ? AND blocked_by > 0", dependency.TaskID).Update("blocked_by", gorm.Expr("blocked_by - 1")).Error; err != nil {
			return err
		}
		if err := tx.Model(&database.WorkflowTask{}).Where("id = ? AND status = ? AND blocked_by = 0", dependency.TaskID, TaskStatusBlocked).Update("status", TaskStatusQueued).Error; err != nil {
			return err
		}
	}
	return nil
}

func encodeJSON(value any) (string, error) {
	if value == nil {
		return "{}", nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal workflow payload: %w", err)
	}
	return string(encoded), nil
}
