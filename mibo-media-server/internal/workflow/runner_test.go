package workflow

import (
	"context"
	"sync"
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

	runner := NewRunner(svc, RunnerConfig{Enabled: true, LeaseDuration: time.Minute})
	started := make(chan uint, 2)
	release := make(chan struct{})
	var once sync.Once
	runner.Register(TaskTypeDiscoverStorage, func(ctx context.Context, task database.WorkflowTask) error {
		started <- task.ID
		once.Do(func() {
			go runner.runOnce(ctx)
		})
		<-release
		return nil
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		runner.runOnce(ctx)
	}()

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
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("runner did not finish")
	}

	assertTaskStatus(t, db, first.ID, TaskStatusCompleted)
	assertTaskStatus(t, db, second.ID, TaskStatusCompleted)
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
