package database

import "time"

type WorkflowRun struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	RunKey       string     `gorm:"size:512;not null;index" json:"run_key"`
	LibraryID    uint       `gorm:"not null;index:idx_workflow_runs_library_status,priority:1" json:"library_id"`
	Reason       string     `gorm:"size:128;not null;index" json:"reason"`
	Status       string     `gorm:"size:64;not null;index:idx_workflow_runs_status_priority_created,priority:1;index:idx_workflow_runs_library_status,priority:2" json:"status"`
	Priority     int        `gorm:"not null;default:0;index:idx_workflow_runs_status_priority_created,priority:2" json:"priority"`
	ScopeKey     string     `gorm:"size:512;not null;index" json:"scope_key"`
	PayloadJSON  string     `gorm:"type:text" json:"payload_json"`
	ErrorMessage string     `gorm:"type:text" json:"error_message"`
	StartedAt    *time.Time `gorm:"index" json:"started_at,omitempty"`
	FinishedAt   *time.Time `gorm:"index" json:"finished_at,omitempty"`
	CancelledAt  *time.Time `gorm:"index" json:"cancelled_at,omitempty"`
	CreatedAt    time.Time  `gorm:"index:idx_workflow_runs_status_priority_created,priority:3" json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (WorkflowRun) TableName() string {
	return "workflow_runs"
}

type WorkflowTask struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	RunID           uint       `gorm:"not null;index:idx_workflow_tasks_run_status,priority:1;index:idx_workflow_tasks_ready,priority:4" json:"run_id"`
	LibraryID       uint       `gorm:"not null;index:idx_workflow_tasks_library_status,priority:1" json:"library_id"`
	TaskKey         string     `gorm:"size:512;not null;uniqueIndex" json:"task_key"`
	TaskType        string     `gorm:"size:128;not null;index" json:"task_type"`
	Stage           string     `gorm:"size:128;not null;index:idx_workflow_tasks_stage_status,priority:1" json:"stage"`
	Status          string     `gorm:"size:64;not null;index:idx_workflow_tasks_ready,priority:1;index:idx_workflow_tasks_run_status,priority:2;index:idx_workflow_tasks_library_status,priority:2;index:idx_workflow_tasks_stage_status,priority:2" json:"status"`
	Priority        int        `gorm:"not null;default:0;index:idx_workflow_tasks_ready,priority:2" json:"priority"`
	ScopeKey        string     `gorm:"size:512;not null;index" json:"scope_key"`
	PayloadJSON     string     `gorm:"type:text" json:"payload_json"`
	ResourceJSON    string     `gorm:"type:text" json:"resource_json"`
	BlockedBy       int        `gorm:"not null;default:0;index" json:"blocked_by"`
	Attempts        int        `gorm:"not null;default:0" json:"attempts"`
	MaxAttempts     int        `gorm:"not null;default:3" json:"max_attempts"`
	AvailableAt     time.Time  `gorm:"not null;index:idx_workflow_tasks_ready,priority:3" json:"available_at"`
	LeaseOwner      string     `gorm:"size:255;index:idx_workflow_tasks_lease,priority:2" json:"lease_owner"`
	LeaseUntil      *time.Time `gorm:"index:idx_workflow_tasks_lease,priority:1" json:"lease_until,omitempty"`
	ErrorMessage    string     `gorm:"type:text" json:"error_message"`
	ResourceWaitKey string     `gorm:"size:128;index" json:"resource_wait_key"`
	StartedAt       *time.Time `gorm:"index" json:"started_at,omitempty"`
	FinishedAt      *time.Time `gorm:"index" json:"finished_at,omitempty"`
	CreatedAt       time.Time  `gorm:"index:idx_workflow_tasks_ready,priority:5" json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (WorkflowTask) TableName() string {
	return "workflow_tasks"
}

type WorkflowTaskDependency struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TaskID          uint      `gorm:"not null;uniqueIndex:idx_workflow_task_dependencies_pair,priority:1;index:idx_workflow_task_dependencies_task,priority:1" json:"task_id"`
	DependsOnTaskID uint      `gorm:"not null;uniqueIndex:idx_workflow_task_dependencies_pair,priority:2;index:idx_workflow_task_dependencies_depends,priority:1" json:"depends_on_task_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (WorkflowTaskDependency) TableName() string {
	return "workflow_task_dependencies"
}

type WorkflowTaskLease struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TaskID    uint      `gorm:"not null;uniqueIndex;index:idx_workflow_task_leases_owner_until,priority:3" json:"task_id"`
	Owner     string    `gorm:"size:255;not null;index:idx_workflow_task_leases_owner_until,priority:1" json:"owner"`
	LeaseUntil time.Time `gorm:"not null;index:idx_workflow_task_leases_owner_until,priority:2;index" json:"lease_until"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (WorkflowTaskLease) TableName() string {
	return "workflow_task_leases"
}

type WorkflowResourceBudget struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ResourceKey    string    `gorm:"size:128;not null;uniqueIndex" json:"resource_key"`
	MaxConcurrency int       `gorm:"not null;default:1" json:"max_concurrency"`
	Enabled        bool      `gorm:"not null;default:true;index" json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (WorkflowResourceBudget) TableName() string {
	return "workflow_resource_budgets"
}

type WorkflowResourceUsage struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ResourceKey string    `gorm:"size:128;not null;uniqueIndex:idx_workflow_resource_usage_resource_task,priority:1;index:idx_workflow_resource_usage_resource,priority:1" json:"resource_key"`
	TaskID      uint      `gorm:"not null;uniqueIndex:idx_workflow_resource_usage_resource_task,priority:2;index" json:"task_id"`
	RunID       uint      `gorm:"not null;index" json:"run_id"`
	LibraryID   uint      `gorm:"not null;index" json:"library_id"`
	Units       int       `gorm:"not null;default:1" json:"units"`
	LeaseUntil  time.Time `gorm:"not null;index:idx_workflow_resource_usage_resource,priority:2;index" json:"lease_until"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (WorkflowResourceUsage) TableName() string {
	return "workflow_resource_usages"
}
