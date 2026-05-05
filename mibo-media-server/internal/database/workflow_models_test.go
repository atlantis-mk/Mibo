package database

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
)

func TestWorkflowModelsMigrateAndPersist(t *testing.T) {
	db, err := Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	now := time.Now().UTC()
	run := WorkflowRun{RunKey: "library:1:scan", LibraryID: 1, Reason: "manual_scan", Status: "queued", Priority: 10, ScopeKey: "library:1", PayloadJSON: `{"root_path":"/media"}`}
	if err := db.Create(&run).Error; err != nil {
		t.Fatalf("create workflow run: %v", err)
	}

	first := WorkflowTask{RunID: run.ID, LibraryID: run.LibraryID, TaskKey: "run:1:discover", TaskType: "discover", Stage: "scan", Status: "queued", Priority: 10, ScopeKey: "library:1", PayloadJSON: `{}`, ResourceJSON: `{"db_write":1}`, AvailableAt: now}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first task: %v", err)
	}
	second := WorkflowTask{RunID: run.ID, LibraryID: run.LibraryID, TaskKey: "run:1:materialize", TaskType: "materialize", Stage: "materialize", Status: "blocked", Priority: 10, ScopeKey: "library:1", PayloadJSON: `{}`, ResourceJSON: `{"db_write":1}`, BlockedBy: 1, AvailableAt: now}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("create second task: %v", err)
	}

	if err := db.Create(&WorkflowTaskDependency{TaskID: second.ID, DependsOnTaskID: first.ID}).Error; err != nil {
		t.Fatalf("create dependency: %v", err)
	}
	leaseUntil := now.Add(time.Minute)
	if err := db.Create(&WorkflowTaskLease{TaskID: first.ID, Owner: "test-worker", LeaseUntil: leaseUntil}).Error; err != nil {
		t.Fatalf("create lease: %v", err)
	}
	if err := db.Create(&WorkflowResourceBudget{ResourceKey: "db_write", MaxConcurrency: 1, Enabled: true}).Error; err != nil {
		t.Fatalf("create budget: %v", err)
	}
	if err := db.Create(&WorkflowResourceUsage{ResourceKey: "db_write", TaskID: first.ID, RunID: run.ID, LibraryID: run.LibraryID, Units: 1, LeaseUntil: leaseUntil}).Error; err != nil {
		t.Fatalf("create usage: %v", err)
	}

	var dependencyCount int64
	if err := db.Model(&WorkflowTaskDependency{}).Where("task_id = ?", second.ID).Count(&dependencyCount).Error; err != nil {
		t.Fatalf("count dependencies: %v", err)
	}
	if dependencyCount != 1 {
		t.Fatalf("expected one dependency, got %d", dependencyCount)
	}

	var usage WorkflowResourceUsage
	if err := db.Where("resource_key = ? AND task_id = ?", "db_write", first.ID).First(&usage).Error; err != nil {
		t.Fatalf("load resource usage: %v", err)
	}
	if usage.RunID != run.ID || usage.LibraryID != run.LibraryID || usage.Units != 1 {
		t.Fatalf("unexpected resource usage: %#v", usage)
	}
}

func TestWorkflowModelsEnforceUniqueTaskKeys(t *testing.T) {
	db, err := Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	run := WorkflowRun{RunKey: "library:1:scan", LibraryID: 1, Reason: "manual_scan", Status: "queued", ScopeKey: "library:1"}
	if err := db.Create(&run).Error; err != nil {
		t.Fatalf("create workflow run: %v", err)
	}
	task := WorkflowTask{RunID: run.ID, LibraryID: 1, TaskKey: "task:1", TaskType: "discover", Stage: "scan", Status: "queued", ScopeKey: "library:1", AvailableAt: time.Now().UTC()}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create workflow task: %v", err)
	}
	duplicateTask := WorkflowTask{RunID: run.ID, LibraryID: 1, TaskKey: task.TaskKey, TaskType: "discover", Stage: "scan", Status: "queued", ScopeKey: "library:1", AvailableAt: time.Now().UTC()}
	if err := db.Create(&duplicateTask).Error; err == nil {
		t.Fatalf("expected duplicate task key to fail")
	}
}
