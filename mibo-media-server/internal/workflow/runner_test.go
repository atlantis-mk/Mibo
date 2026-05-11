package workflow

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func TestRunnerProcessesDifferentLibrariesConcurrentlyWhenResourcesAllow(t *testing.T) {
	ctx := context.Background()
	svc, db := newTestService(t)
	if err := svc.EnsureResourceBudgets(ctx, map[string]int{ResourceDBWrite: 2}); err != nil {
		t.Fatalf("ensure budgets: %v", err)
	}
	firstRun := createTestRunWithKey(t, ctx, svc, "library:1:manual_scan", 1)
	secondRun := createTestRunWithKey(t, ctx, svc, "library:2:manual_scan", 2)
	first := createSchedulerTask(t, ctx, svc, firstRun, "task:runner:library:1", map[string]int{ResourceDBWrite: 1})
	second := createSchedulerTask(t, ctx, svc, secondRun, "task:runner:library:2", map[string]int{ResourceDBWrite: 1})

	runner := NewRunner(svc, RunnerConfig{Enabled: true, LeaseDuration: time.Minute, MaxConcurrent: 2, Owner: "test-runner"})
	started := make(chan uint, 2)
	release := make(chan struct{})
	runner.Register(TaskTypeDiscoverStorage, func(ctx context.Context, task database.WorkflowTask) error {
		started <- task.ID
		<-release
		return nil
	})

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go runner.Run(runCtx)

	seen := map[uint]bool{}
	deadline := time.After(time.Second)
	for len(seen) < 2 {
		select {
		case taskID := <-started:
			seen[taskID] = true
		case <-deadline:
			t.Fatalf("expected both library tasks to start concurrently, seen=%v", seen)
		}
	}
	if !seen[first.ID] || !seen[second.ID] {
		t.Fatalf("expected tasks %d and %d to start, seen=%v", first.ID, second.ID, seen)
	}
	close(release)
	deadline = time.After(time.Second)
	for _, taskID := range []uint{first.ID, second.ID} {
		for {
			var task database.WorkflowTask
			if err := db.First(&task, taskID).Error; err != nil {
				t.Fatalf("load task: %v", err)
			}
			if task.Status == TaskStatusCompleted {
				break
			}
			select {
			case <-deadline:
				t.Fatalf("task %d did not complete, status=%s", taskID, task.Status)
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
	}

	assertTaskStatus(t, db, first.ID, TaskStatusCompleted)
	assertTaskStatus(t, db, second.ID, TaskStatusCompleted)
}

func TestRunnerHeartbeatRenewsLeaseWhileTaskRuns(t *testing.T) {
	ctx := context.Background()
	svc, db := newTestService(t)
	run := createTestRun(t, ctx, svc)
	task := createSchedulerTask(t, ctx, svc, run, "task:runner:heartbeat", nil)

	runner := NewRunner(svc, RunnerConfig{Enabled: true, LeaseDuration: 80 * time.Millisecond, PollInterval: 10 * time.Millisecond, MaxConcurrent: 1, Owner: "test-runner"})
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	runner.Register(TaskTypeDiscoverStorage, func(ctx context.Context, task database.WorkflowTask) error {
		started <- struct{}{}
		<-release
		return nil
	})

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go runner.Run(runCtx)

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("task did not start")
	}

	var initialLease database.WorkflowTaskLease
	deadline := time.After(time.Second)
	for {
		if err := db.Where("task_id = ?", task.ID).First(&initialLease).Error; err == nil {
			break
		}
		select {
		case <-deadline:
			t.Fatal("initial lease not created")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	time.Sleep(60 * time.Millisecond)

	var renewedLease database.WorkflowTaskLease
	if err := db.Where("task_id = ?", task.ID).First(&renewedLease).Error; err != nil {
		t.Fatalf("load renewed lease: %v", err)
	}
	if !renewedLease.LeaseUntil.After(initialLease.LeaseUntil) {
		t.Fatalf("expected lease renewal after %s, got %s", initialLease.LeaseUntil, renewedLease.LeaseUntil)
	}

	close(release)
	deadline = time.After(time.Second)
	for {
		var current database.WorkflowTask
		if err := db.First(&current, task.ID).Error; err != nil {
			t.Fatalf("load task: %v", err)
		}
		if current.Status == TaskStatusCompleted {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("task did not complete, status=%s", current.Status)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestRunnerFailsTaskWhenExecutionTimesOut(t *testing.T) {
	ctx := context.Background()
	svc, db := newTestService(t)
	run := createTestRun(t, ctx, svc)
	task := createSchedulerTask(t, ctx, svc, run, "task:runner:timeout", nil)

	runner := NewRunner(svc, RunnerConfig{Enabled: true, LeaseDuration: time.Second, TaskTimeout: 50 * time.Millisecond, PollInterval: 10 * time.Millisecond, MaxConcurrent: 1, Owner: "test-runner"})
	started := make(chan struct{}, 1)
	runner.Register(TaskTypeDiscoverStorage, func(ctx context.Context, task database.WorkflowTask) error {
		started <- struct{}{}
		<-ctx.Done()
		return ctx.Err()
	})

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go runner.Run(runCtx)

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("task did not start")
	}

	deadline := time.After(time.Second)
	for {
		var current database.WorkflowTask
		if err := db.First(&current, task.ID).Error; err != nil {
			t.Fatalf("load task: %v", err)
		}
		if current.Status == TaskStatusFailed {
			if !strings.Contains(current.ErrorMessage, context.DeadlineExceeded.Error()) {
				t.Fatalf("expected timeout error message, got %q", current.ErrorMessage)
			}
			break
		}
		select {
		case <-deadline:
			t.Fatalf("task did not fail, status=%s", current.Status)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestRunnerRegisterPanicsOnDuplicateTaskType(t *testing.T) {
	runner := NewRunner(nil, RunnerConfig{})
	runner.Register(TaskTypeDiscoverStorage, func(ctx context.Context, task database.WorkflowTask) error { return nil })

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected duplicate registration panic")
		}
		message, ok := recovered.(string)
		if !ok || !strings.Contains(message, TaskTypeDiscoverStorage) {
			t.Fatalf("unexpected panic: %#v", recovered)
		}
	}()

	runner.Register(TaskTypeDiscoverStorage, func(ctx context.Context, task database.WorkflowTask) error { return nil })
}

func assertTaskStatus(t *testing.T, db *gorm.DB, taskID uint, status string) {
	t.Helper()
	var task database.WorkflowTask
	if err := db.First(&task, taskID).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	if task.Status != status {
		t.Fatalf("expected task %d status %q, got %q", taskID, status, task.Status)
	}
}
