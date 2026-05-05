package workflow

import (
	"context"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
)

type RunStatusView struct {
	Run         database.WorkflowRun       `json:"run"`
	TaskCounts  []TaskStatusCount          `json:"task_counts"`
	ResourceWaits []ResourceWaitCount      `json:"resource_waits"`
	RecentTasks []database.WorkflowTask    `json:"recent_tasks"`
}

type TaskStatusCount struct {
	Stage  string `json:"stage"`
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type ResourceWaitCount struct {
	ResourceKey string `json:"resource_key"`
	Count       int64  `json:"count"`
}

type Diagnostics struct {
	ActiveRuns           int64                            `json:"active_runs"`
	RunningTasks         int64                            `json:"running_tasks"`
	BlockedTasks         int64                            `json:"blocked_tasks"`
	ExpiredLeases        int64                            `json:"expired_leases"`
	ResourceBudgets      []database.WorkflowResourceBudget `json:"resource_budgets"`
	ResourceUsage        []database.WorkflowResourceUsage  `json:"resource_usage"`
}

func (s *Service) ListRuns(ctx context.Context, limit int, offset int, status string, libraryID uint) ([]RunStatusView, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	query := s.db.WithContext(ctx).Order("updated_at desc, id desc").Limit(limit).Offset(offset)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if libraryID != 0 {
		query = query.Where("library_id = ?", libraryID)
	}
	var runs []database.WorkflowRun
	if err := query.Find(&runs).Error; err != nil {
		return nil, err
	}
	views := make([]RunStatusView, 0, len(runs))
	for _, run := range runs {
		view, err := s.GetRunStatus(ctx, run.ID)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) GetRunStatus(ctx context.Context, runID uint) (RunStatusView, error) {
	var run database.WorkflowRun
	if err := s.db.WithContext(ctx).First(&run, runID).Error; err != nil {
		return RunStatusView{}, err
	}
	var taskCounts []TaskStatusCount
	if err := s.db.WithContext(ctx).Model(&database.WorkflowTask{}).Select("stage, status, COUNT(*) AS count").Where("run_id = ?", runID).Group("stage, status").Scan(&taskCounts).Error; err != nil {
		return RunStatusView{}, err
	}
	var waits []ResourceWaitCount
	if err := s.db.WithContext(ctx).Model(&database.WorkflowTask{}).Select("resource_wait_key AS resource_key, COUNT(*) AS count").Where("run_id = ? AND resource_wait_key <> ''", runID).Group("resource_wait_key").Scan(&waits).Error; err != nil {
		return RunStatusView{}, err
	}
	var recent []database.WorkflowTask
	if err := s.db.WithContext(ctx).Where("run_id = ?", runID).Order("updated_at desc, id desc").Limit(20).Find(&recent).Error; err != nil {
		return RunStatusView{}, err
	}
	return RunStatusView{Run: run, TaskCounts: taskCounts, ResourceWaits: waits, RecentTasks: recent}, nil
}

func (s *Service) Diagnostics(ctx context.Context, now time.Time) (Diagnostics, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	diagnostics := Diagnostics{}
	if err := s.db.WithContext(ctx).Model(&database.WorkflowRun{}).Where("status IN ?", []string{RunStatusQueued, RunStatusRunning}).Count(&diagnostics.ActiveRuns).Error; err != nil {
		return Diagnostics{}, err
	}
	if err := s.db.WithContext(ctx).Model(&database.WorkflowTask{}).Where("status = ?", TaskStatusRunning).Count(&diagnostics.RunningTasks).Error; err != nil {
		return Diagnostics{}, err
	}
	if err := s.db.WithContext(ctx).Model(&database.WorkflowTask{}).Where("status = ?", TaskStatusBlocked).Count(&diagnostics.BlockedTasks).Error; err != nil {
		return Diagnostics{}, err
	}
	if err := s.db.WithContext(ctx).Model(&database.WorkflowTaskLease{}).Where("lease_until < ?", now).Count(&diagnostics.ExpiredLeases).Error; err != nil {
		return Diagnostics{}, err
	}
	if err := s.db.WithContext(ctx).Order("resource_key asc").Find(&diagnostics.ResourceBudgets).Error; err != nil {
		return Diagnostics{}, err
	}
	if err := s.db.WithContext(ctx).Order("resource_key asc, task_id asc").Find(&diagnostics.ResourceUsage).Error; err != nil {
		return Diagnostics{}, err
	}
	return diagnostics, nil
}
