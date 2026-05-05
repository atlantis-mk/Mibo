package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNoReadyTask = errors.New("no ready workflow task")

type ClaimInput struct {
	Owner         string
	LeaseDuration time.Duration
	Now           time.Time
	Limit         int
}

func (s *Service) EnsureResourceBudgets(ctx context.Context, budgets map[string]int) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for resourceKey, maxConcurrency := range budgets {
			if resourceKey == "" || maxConcurrency <= 0 {
				continue
			}
			budget := database.WorkflowResourceBudget{ResourceKey: resourceKey, MaxConcurrency: maxConcurrency, Enabled: true}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "resource_key"}}, DoUpdates: clause.AssignmentColumns([]string{"max_concurrency", "enabled", "updated_at"})}).Create(&budget).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) ClaimNextTask(ctx context.Context, input ClaimInput) (database.WorkflowTask, error) {
	if input.Owner == "" {
		return database.WorkflowTask{}, errors.New("claim owner is required")
	}
	if input.LeaseDuration <= 0 {
		input.LeaseDuration = time.Minute
	}
	if input.Now.IsZero() {
		input.Now = time.Now().UTC()
	}
	if input.Limit <= 0 {
		input.Limit = 100
	}
	leaseUntil := input.Now.Add(input.LeaseDuration)
	var claimed database.WorkflowTask
	claimedTask := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var candidates []database.WorkflowTask
		if err := tx.Where("status IN ? AND blocked_by = 0 AND available_at <= ?", []string{TaskStatusQueued, TaskStatusRetrying}, input.Now).
			Order("priority desc, run_id asc, created_at asc, id asc").
			Limit(input.Limit).
			Find(&candidates).Error; err != nil {
			return err
		}
		for _, candidate := range candidates {
			if s.hasLibraryConflict(tx, candidate) {
				continue
			}
			resources, err := decodeResources(candidate.ResourceJSON)
			if err != nil {
				return err
			}
			ok, waitKey, err := resourcesAvailable(tx, resources)
			if err != nil {
				return err
			}
			if !ok {
				_ = tx.Model(&database.WorkflowTask{}).Where("id = ?", candidate.ID).Update("resource_wait_key", waitKey).Error
				continue
			}
			result := tx.Model(&database.WorkflowTask{}).
				Where("id = ? AND status IN ? AND blocked_by = 0", candidate.ID, []string{TaskStatusQueued, TaskStatusRetrying}).
				Updates(map[string]any{"status": TaskStatusRunning, "attempts": gorm.Expr("attempts + 1"), "started_at": input.Now, "lease_owner": input.Owner, "lease_until": leaseUntil, "resource_wait_key": "", "error_message": ""})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				continue
			}
			if err := reserveResources(tx, candidate, resources, leaseUntil); err != nil {
				return err
			}
			lease := database.WorkflowTaskLease{TaskID: candidate.ID, Owner: input.Owner, LeaseUntil: leaseUntil}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "task_id"}}, DoUpdates: clause.AssignmentColumns([]string{"owner", "lease_until", "updated_at"})}).Create(&lease).Error; err != nil {
				return err
			}
			if err := tx.First(&claimed, candidate.ID).Error; err != nil {
				return err
			}
			claimedTask = true
			return nil
		}
		return nil
	})
	if err != nil {
		return database.WorkflowTask{}, err
	}
	if !claimedTask {
		return database.WorkflowTask{}, ErrNoReadyTask
	}
	return claimed, nil
}

func (s *Service) hasLibraryConflict(tx *gorm.DB, candidate database.WorkflowTask) bool {
	var count int64
	if err := tx.Model(&database.WorkflowTask{}).Where("library_id = ? AND id <> ? AND status = ?", candidate.LibraryID, candidate.ID, TaskStatusRunning).Count(&count).Error; err != nil {
		return true
	}
	return count > 0
}

func resourcesAvailable(tx *gorm.DB, resources map[string]int) (bool, string, error) {
	for resourceKey, requestedUnits := range resources {
		if requestedUnits <= 0 {
			continue
		}
		var budget database.WorkflowResourceBudget
		if err := tx.Where("resource_key = ? AND enabled = ?", resourceKey, true).First(&budget).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, resourceKey, nil
			}
			return false, "", err
		}
		var used int64
		if err := tx.Model(&database.WorkflowResourceUsage{}).Where("resource_key = ?", resourceKey).Select("COALESCE(SUM(units), 0)").Scan(&used).Error; err != nil {
			return false, "", err
		}
		if int(used)+requestedUnits > budget.MaxConcurrency {
			return false, resourceKey, nil
		}
	}
	return true, "", nil
}

func reserveResources(tx *gorm.DB, task database.WorkflowTask, resources map[string]int, leaseUntil time.Time) error {
	for resourceKey, units := range resources {
		if units <= 0 {
			continue
		}
		usage := database.WorkflowResourceUsage{ResourceKey: resourceKey, TaskID: task.ID, RunID: task.RunID, LibraryID: task.LibraryID, Units: units, LeaseUntil: leaseUntil}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "resource_key"}, {Name: "task_id"}}, DoUpdates: clause.AssignmentColumns([]string{"units", "lease_until", "updated_at"})}).Create(&usage).Error; err != nil {
			return err
		}
	}
	return nil
}

func decodeResources(raw string) (map[string]int, error) {
	if raw == "" {
		return nil, nil
	}
	var resources map[string]int
	if err := json.Unmarshal([]byte(raw), &resources); err != nil {
		return nil, fmt.Errorf("decode workflow resources: %w", err)
	}
	return resources, nil
}
