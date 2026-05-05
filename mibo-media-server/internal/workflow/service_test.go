package workflow

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func TestCreateOrReuseRunReusesActiveRun(t *testing.T) {
	ctx := context.Background()
	svc, db := newTestService(t)

	run, reused, err := svc.CreateOrReuseRun(ctx, CreateRunInput{RunKey: "library:1:manual_scan", LibraryID: 1, Reason: "manual_scan", Priority: 5})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if reused {
		t.Fatalf("first run should not be reused")
	}
	second, reused, err := svc.CreateOrReuseRun(ctx, CreateRunInput{RunKey: "library:1:manual_scan", LibraryID: 1, Reason: "manual_scan", Priority: 5})
	if err != nil {
		t.Fatalf("reuse run: %v", err)
	}
	if !reused || second.ID != run.ID {
		t.Fatalf("expected active run reuse, got reused=%v second=%#v run=%#v", reused, second, run)
	}

	if err := db.Model(&database.WorkflowRun{}).Where("id = ?", run.ID).Update("status", RunStatusCompleted).Error; err != nil {
		t.Fatalf("complete run: %v", err)
	}
	third, reused, err := svc.CreateOrReuseRun(ctx, CreateRunInput{RunKey: "library:1:manual_scan", LibraryID: 1, Reason: "manual_scan", Priority: 5})
	if err != nil {
		t.Fatalf("create after completion: %v", err)
	}
	if reused || third.ID == run.ID {
		t.Fatalf("expected new run after completion, got reused=%v third=%#v", reused, third)
	}
}

func TestTaskCompletionUnlocksDependencies(t *testing.T) {
	ctx := context.Background()
	svc, db := newTestService(t)
	run := createTestRun(t, ctx, svc)

	first, err := svc.CreateTask(ctx, run, CreateTaskInput{TaskKey: "task:first", TaskType: "discover", Stage: "scan"})
	if err != nil {
		t.Fatalf("create first task: %v", err)
	}
	second, err := svc.CreateTask(ctx, run, CreateTaskInput{TaskKey: "task:second", TaskType: "materialize", Stage: "materialize", DependsOnTaskIDs: []uint{first.ID}})
	if err != nil {
		t.Fatalf("create second task: %v", err)
	}
	if second.Status != TaskStatusBlocked || second.BlockedBy != 1 {
		t.Fatalf("expected blocked dependent, got %#v", second)
	}

	started, err := svc.StartTask(ctx, first.ID, "worker-1", time.Now().UTC().Add(time.Minute))
	if err != nil {
		t.Fatalf("start first task: %v", err)
	}
	if started.Status != TaskStatusRunning {
		t.Fatalf("expected running first task, got %q", started.Status)
	}
	if err := svc.CompleteTask(ctx, first.ID); err != nil {
		t.Fatalf("complete first task: %v", err)
	}

	var unlocked database.WorkflowTask
	if err := db.First(&unlocked, second.ID).Error; err != nil {
		t.Fatalf("load dependent: %v", err)
	}
	if unlocked.Status != TaskStatusQueued || unlocked.BlockedBy != 0 {
		t.Fatalf("expected queued unlocked task, got %#v", unlocked)
	}
}

func TestCancelAndSupersedeRunUpdateActiveTasks(t *testing.T) {
	ctx := context.Background()
	svc, db := newTestService(t)
	run := createTestRun(t, ctx, svc)
	task, err := svc.CreateTask(ctx, run, CreateTaskInput{TaskKey: "task:cancel", TaskType: "discover", Stage: "scan"})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := svc.CancelRun(ctx, run.ID, "cancelled by test"); err != nil {
		t.Fatalf("cancel run: %v", err)
	}
	assertRunAndTaskStatus(t, db, run.ID, task.ID, RunStatusCancelled, TaskStatusCancelled)

	run = createTestRunWithKey(t, ctx, svc, "library:2:manual_scan", 2)
	task, err = svc.CreateTask(ctx, run, CreateTaskInput{TaskKey: "task:supersede", TaskType: "discover", Stage: "scan"})
	if err != nil {
		t.Fatalf("create superseded task: %v", err)
	}
	if err := svc.SupersedeRun(ctx, run.ID, "newer run exists"); err != nil {
		t.Fatalf("supersede run: %v", err)
	}
	assertRunAndTaskStatus(t, db, run.ID, task.ID, RunStatusSuperseded, TaskStatusSuperseded)
}

func TestRenewAndRecoverExpiredLeases(t *testing.T) {
	ctx := context.Background()
	svc, db := newTestService(t)
	run := createTestRun(t, ctx, svc)
	task, err := svc.CreateTask(ctx, run, CreateTaskInput{TaskKey: "task:lease", TaskType: "discover", Stage: "scan"})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	leaseUntil := time.Now().UTC().Add(time.Minute)
	if _, err := svc.StartTask(ctx, task.ID, "worker-1", leaseUntil); err != nil {
		t.Fatalf("start task: %v", err)
	}
	newLeaseUntil := leaseUntil.Add(time.Minute)
	if err := svc.RenewLease(ctx, task.ID, "worker-1", newLeaseUntil); err != nil {
		t.Fatalf("renew lease: %v", err)
	}

	var lease database.WorkflowTaskLease
	if err := db.Where("task_id = ?", task.ID).First(&lease).Error; err != nil {
		t.Fatalf("load lease: %v", err)
	}
	if !lease.LeaseUntil.Equal(newLeaseUntil) {
		t.Fatalf("expected renewed lease %s, got %s", newLeaseUntil, lease.LeaseUntil)
	}

	recovered, err := svc.RecoverExpiredLeases(ctx, newLeaseUntil.Add(time.Second))
	if err != nil {
		t.Fatalf("recover leases: %v", err)
	}
	if recovered != 1 {
		t.Fatalf("expected one recovered lease, got %d", recovered)
	}
	var recoveredTask database.WorkflowTask
	if err := db.First(&recoveredTask, task.ID).Error; err != nil {
		t.Fatalf("load recovered task: %v", err)
	}
	if recoveredTask.Status != TaskStatusRetrying || recoveredTask.LeaseUntil != nil || recoveredTask.LeaseOwner != "" {
		t.Fatalf("unexpected recovered task: %#v", recoveredTask)
	}
}

func TestFailAndRetryTask(t *testing.T) {
	ctx := context.Background()
	svc, db := newTestService(t)
	run := createTestRun(t, ctx, svc)
	task, err := svc.CreateTask(ctx, run, CreateTaskInput{TaskKey: "task:retry", TaskType: "discover", Stage: "scan"})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if _, err := svc.StartTask(ctx, task.ID, "worker-1", time.Now().UTC().Add(time.Minute)); err != nil {
		t.Fatalf("start task: %v", err)
	}
	if err := svc.FailTask(ctx, task.ID, errors.New("boom")); err != nil {
		t.Fatalf("fail task: %v", err)
	}
	if err := svc.RetryTask(ctx, task.ID, time.Now().UTC()); err != nil {
		t.Fatalf("retry task: %v", err)
	}
	var retry database.WorkflowTask
	if err := db.First(&retry, task.ID).Error; err != nil {
		t.Fatalf("load retry task: %v", err)
	}
	if retry.Status != TaskStatusRetrying || retry.ErrorMessage != "" {
		t.Fatalf("unexpected retry task: %#v", retry)
	}
}

func newTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	return NewService(db), db
}

func createTestRun(t *testing.T, ctx context.Context, svc *Service) database.WorkflowRun {
	t.Helper()
	return createTestRunWithKey(t, ctx, svc, "library:1:manual_scan", 1)
}

func createTestRunWithKey(t *testing.T, ctx context.Context, svc *Service, key string, libraryID uint) database.WorkflowRun {
	t.Helper()
	run, _, err := svc.CreateOrReuseRun(ctx, CreateRunInput{RunKey: key, LibraryID: libraryID, Reason: "manual_scan", Priority: 1})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	return run
}

func assertRunAndTaskStatus(t *testing.T, db *gorm.DB, runID uint, taskID uint, runStatus string, taskStatus string) {
	t.Helper()
	var run database.WorkflowRun
	if err := db.First(&run, runID).Error; err != nil {
		t.Fatalf("load run: %v", err)
	}
	if run.Status != runStatus {
		t.Fatalf("expected run status %q, got %q", runStatus, run.Status)
	}
	var task database.WorkflowTask
	if err := db.First(&task, taskID).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	if task.Status != taskStatus {
		t.Fatalf("expected task status %q, got %q", taskStatus, task.Status)
	}
}
