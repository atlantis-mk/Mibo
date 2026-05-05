package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func TestClaimNextTaskAllowsDifferentLibrariesAndBlocksSameLibrary(t *testing.T) {
	ctx := context.Background()
	svc, db := newTestService(t)
	if err := svc.EnsureResourceBudgets(ctx, map[string]int{ResourceDBWrite: 2}); err != nil {
		t.Fatalf("ensure budgets: %v", err)
	}
	firstRun := createTestRunWithKey(t, ctx, svc, "library:1:manual_scan", 1)
	secondRun := createTestRunWithKey(t, ctx, svc, "library:2:manual_scan", 2)
	first := createSchedulerTask(t, ctx, svc, firstRun, "task:library:1:first", map[string]int{ResourceDBWrite: 1})
	sameLibrary := createSchedulerTask(t, ctx, svc, firstRun, "task:library:1:second", map[string]int{ResourceDBWrite: 1})
	differentLibrary := createSchedulerTask(t, ctx, svc, secondRun, "task:library:2:first", map[string]int{ResourceDBWrite: 1})

	claimed, err := svc.ClaimNextTask(ctx, ClaimInput{Owner: "worker-1", LeaseDuration: time.Minute})
	if err != nil {
		t.Fatalf("claim first: %v", err)
	}
	if claimed.ID != first.ID {
		t.Fatalf("expected first FIFO task %d, got %d", first.ID, claimed.ID)
	}
	claimed, err = svc.ClaimNextTask(ctx, ClaimInput{Owner: "worker-2", LeaseDuration: time.Minute})
	if err != nil {
		t.Fatalf("claim different library: %v", err)
	}
	if claimed.ID != differentLibrary.ID {
		t.Fatalf("expected different library task %d, got %d", differentLibrary.ID, claimed.ID)
	}

	var same database.WorkflowTask
	if err := db.First(&same, sameLibrary.ID).Error; err != nil {
		t.Fatalf("load same-library task: %v", err)
	}
	if same.Status != TaskStatusQueued {
		t.Fatalf("expected same-library task to remain queued, got %#v", same)
	}
}

func TestClaimNextTaskRespectsResourceBudgetAndReleasesOnComplete(t *testing.T) {
	ctx := context.Background()
	svc, db := newTestService(t)
	if err := svc.EnsureResourceBudgets(ctx, map[string]int{ResourceFFprobe: 1, ResourceDBWrite: 2}); err != nil {
		t.Fatalf("ensure budgets: %v", err)
	}
	firstRun := createTestRunWithKey(t, ctx, svc, "library:1:manual_scan", 1)
	secondRun := createTestRunWithKey(t, ctx, svc, "library:2:manual_scan", 2)
	first := createSchedulerTask(t, ctx, svc, firstRun, "task:ffprobe:first", map[string]int{ResourceFFprobe: 1})
	second := createSchedulerTask(t, ctx, svc, secondRun, "task:ffprobe:second", map[string]int{ResourceFFprobe: 1})

	claimed, err := svc.ClaimNextTask(ctx, ClaimInput{Owner: "worker-1", LeaseDuration: time.Minute})
	if err != nil {
		t.Fatalf("claim first: %v", err)
	}
	if claimed.ID != first.ID {
		t.Fatalf("expected first ffprobe task %d, got %d", first.ID, claimed.ID)
	}
	_, err = svc.ClaimNextTask(ctx, ClaimInput{Owner: "worker-2", LeaseDuration: time.Minute})
	if !errors.Is(err, ErrNoReadyTask) {
		t.Fatalf("expected no ready task while ffprobe exhausted, got %v", err)
	}

	var waiting database.WorkflowTask
	if err := db.First(&waiting, second.ID).Error; err != nil {
		t.Fatalf("load waiting task: %v", err)
	}
	if waiting.ResourceWaitKey != ResourceFFprobe {
		t.Fatalf("expected resource wait %q, got %q", ResourceFFprobe, waiting.ResourceWaitKey)
	}
	if err := svc.CompleteTask(ctx, first.ID); err != nil {
		t.Fatalf("complete first: %v", err)
	}
	claimed, err = svc.ClaimNextTask(ctx, ClaimInput{Owner: "worker-2", LeaseDuration: time.Minute})
	if err != nil {
		t.Fatalf("claim second after release: %v", err)
	}
	if claimed.ID != second.ID {
		t.Fatalf("expected second ffprobe task %d, got %d", second.ID, claimed.ID)
	}
}

func TestClaimNextTaskSkipsBlockedDependencies(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)
	if err := svc.EnsureResourceBudgets(ctx, map[string]int{ResourceDBWrite: 1}); err != nil {
		t.Fatalf("ensure budgets: %v", err)
	}
	run := createTestRun(t, ctx, svc)
	first := createSchedulerTask(t, ctx, svc, run, "task:dependency:first", map[string]int{ResourceDBWrite: 1})
	blocked, err := svc.CreateTask(ctx, run, CreateTaskInput{TaskKey: "task:dependency:blocked", TaskType: TaskTypeMaterializeCatalog, Stage: StageMaterialize, Resources: map[string]int{ResourceDBWrite: 1}, DependsOnTaskIDs: []uint{first.ID}})
	if err != nil {
		t.Fatalf("create blocked task: %v", err)
	}
	claimed, err := svc.ClaimNextTask(ctx, ClaimInput{Owner: "worker-1", LeaseDuration: time.Minute})
	if err != nil {
		t.Fatalf("claim first: %v", err)
	}
	if claimed.ID != first.ID {
		t.Fatalf("expected first task %d, got %d", first.ID, claimed.ID)
	}
	if err := svc.CompleteTask(ctx, first.ID); err != nil {
		t.Fatalf("complete first: %v", err)
	}
	claimed, err = svc.ClaimNextTask(ctx, ClaimInput{Owner: "worker-2", LeaseDuration: time.Minute})
	if err != nil {
		t.Fatalf("claim unblocked: %v", err)
	}
	if claimed.ID != blocked.ID {
		t.Fatalf("expected unblocked task %d, got %d", blocked.ID, claimed.ID)
	}
}

func TestDefaultTaskTypesAreIsolatedCopies(t *testing.T) {
	definitions := DefaultTaskTypeDefinitions()
	definitions[TaskTypeProbeInventory].Resources[ResourceFFprobe] = 99
	if DefaultTaskTypeDefinitions()[TaskTypeProbeInventory].Resources[ResourceFFprobe] == 99 {
		t.Fatalf("expected default definitions to be copied")
	}
	if DefaultSQLiteResourceBudgets()[ResourceDBWrite] != 1 {
		t.Fatalf("expected sqlite db_write budget to default to 1")
	}
}

func createSchedulerTask(t *testing.T, ctx context.Context, svc *Service, run database.WorkflowRun, taskKey string, resources map[string]int) database.WorkflowTask {
	t.Helper()
	task, err := svc.CreateTask(ctx, run, CreateTaskInput{TaskKey: taskKey, TaskType: TaskTypeDiscoverStorage, Stage: StageScan, Resources: resources})
	if err != nil {
		t.Fatalf("create scheduler task: %v", err)
	}
	return task
}

var _ *gorm.DB
