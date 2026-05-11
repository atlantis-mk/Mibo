package workflow

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

type Handler func(context.Context, database.WorkflowTask) error

type RunnerConfig struct {
	Enabled       bool
	PollInterval  time.Duration
	LeaseDuration time.Duration
	TaskTimeout   time.Duration
	MaxConcurrent int
	Owner         string
}

type Runner struct {
	service  *Service
	config   RunnerConfig
	handlers map[string]Handler
}

func (r *Runner) Service() *Service {
	if r == nil {
		return nil
	}
	return r.service
}

func NewRunner(service *Service, config RunnerConfig) *Runner {
	if config.PollInterval <= 0 {
		config.PollInterval = 2 * time.Second
	}
	if config.LeaseDuration <= 0 {
		config.LeaseDuration = time.Minute
	}
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 4
	}
	if config.Owner == "" {
		config.Owner = fmt.Sprintf("workflow-runner-%d", time.Now().UnixNano())
	}
	return &Runner{service: service, config: config, handlers: map[string]Handler{}}
}

func (r *Runner) Register(taskType string, handler Handler) {
	if taskType == "" || handler == nil {
		return
	}
	if _, exists := r.handlers[taskType]; exists {
		panic(fmt.Sprintf("workflow handler already registered for task type %q", taskType))
	}
	r.handlers[taskType] = handler
}

func (r *Runner) HandleTaskForTest(ctx context.Context, task database.WorkflowTask) error {
	return r.handleTask(ctx, task)
}

func (r *Runner) Run(ctx context.Context) {
	if r == nil || r.service == nil || !r.config.Enabled {
		return
	}
	for workerIndex := 0; workerIndex < r.config.MaxConcurrent; workerIndex++ {
		go r.runWorker(ctx, workerIndex)
	}
	<-ctx.Done()
}

func (r *Runner) runWorker(ctx context.Context, workerIndex int) {
	owner := fmt.Sprintf("%s-%d", r.config.Owner, workerIndex+1)
	ticker := time.NewTicker(r.config.PollInterval)
	defer ticker.Stop()
	r.runOnce(ctx, owner)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.runOnce(ctx, owner)
		}
	}
}

func (r *Runner) runOnce(ctx context.Context, owner string) {
	if _, err := r.service.RecoverExpiredLeases(ctx, time.Now().UTC()); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("workflow: recover expired leases: %v", err)
	}
	for {
		task, err := r.service.ClaimNextTask(ctx, ClaimInput{Owner: owner, LeaseDuration: r.config.LeaseDuration})
		if err != nil {
			if errors.Is(err, ErrNoReadyTask) || errors.Is(err, context.Canceled) {
				return
			}
			log.Printf("workflow: claim task: %v", err)
			return
		}
		r.finishTask(ctx, task, r.handleTaskWithLease(ctx, owner, task))
	}
}

func (r *Runner) handleTask(ctx context.Context, task database.WorkflowTask) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	handler := r.handlers[task.TaskType]
	if handler == nil {
		return fmt.Errorf("workflow handler unavailable for task type %q", task.TaskType)
	}
	return handler(ctx, task)
}

func (r *Runner) handleTaskWithLease(ctx context.Context, owner string, task database.WorkflowTask) error {
	taskCtx := ctx
	var taskCancel context.CancelFunc
	if r.config.TaskTimeout > 0 {
		taskCtx, taskCancel = context.WithTimeout(ctx, r.config.TaskTimeout)
	} else {
		taskCtx, taskCancel = context.WithCancel(ctx)
	}
	defer taskCancel()
	leaseCtx, leaseCancel := context.WithCancel(taskCtx)
	defer leaseCancel()
	go r.heartbeatLease(leaseCtx, owner, task.ID)
	err := r.handleTask(taskCtx, task)
	if err != nil {
		return err
	}
	if timeoutErr := taskCtx.Err(); errors.Is(timeoutErr, context.DeadlineExceeded) {
		return timeoutErr
	}
	return nil
}

func (r *Runner) heartbeatLease(ctx context.Context, owner string, taskID uint) {
	interval := r.config.LeaseDuration / 2
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			leaseUntil := time.Now().UTC().Add(r.config.LeaseDuration)
			if err := r.service.RenewLease(ctx, taskID, owner, leaseUntil); err != nil {
				if !errors.Is(err, context.Canceled) && !errors.Is(err, gorm.ErrRecordNotFound) {
					log.Printf("workflow: renew lease for task %d: %v", taskID, err)
				}
				return
			}
		}
	}
}

func (r *Runner) finishTask(ctx context.Context, task database.WorkflowTask, err error) {
	if err != nil {
		log.Printf("workflow: task %d (%s) failed: %v", task.ID, task.TaskType, err)
		if failErr := r.service.FailTask(ctx, task.ID, err); failErr != nil {
			log.Printf("workflow: mark task %d failed: %v", task.ID, failErr)
		}
		return
	}
	if err := r.service.CompleteTask(ctx, task.ID); err != nil {
		log.Printf("workflow: mark task %d completed: %v", task.ID, err)
	}
}
