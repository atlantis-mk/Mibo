package workflow

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
)

type Handler func(context.Context, database.WorkflowTask) error

type RunnerConfig struct {
	Enabled       bool
	PollInterval  time.Duration
	LeaseDuration time.Duration
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
	ticker := time.NewTicker(r.config.PollInterval)
	defer ticker.Stop()
	r.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.runOnce(ctx)
		}
	}
}

func (r *Runner) runOnce(ctx context.Context) {
	if _, err := r.service.RecoverExpiredLeases(ctx, time.Now().UTC()); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("workflow: recover expired leases: %v", err)
	}
	var wg sync.WaitGroup
	defer wg.Wait()
	for {
		task, err := r.service.ClaimNextTask(ctx, ClaimInput{Owner: r.config.Owner, LeaseDuration: r.config.LeaseDuration})
		if err != nil {
			if errors.Is(err, ErrNoReadyTask) || errors.Is(err, context.Canceled) {
				return
			}
			log.Printf("workflow: claim task: %v", err)
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.finishTask(ctx, task, r.handleTask(ctx, task))
		}()
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
