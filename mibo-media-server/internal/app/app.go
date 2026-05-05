package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/atlan/mibo-media-server/internal/auth"
	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/health"
	"github.com/atlan/mibo-media-server/internal/httpapi"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/listener"
	"github.com/atlan/mibo-media-server/internal/metadata"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/probe"
	"github.com/atlan/mibo-media-server/internal/progress"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/schedule"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
)

type App struct {
	cfg      config.Config
	db       *gorm.DB
	server   *http.Server
	workflow *workflow.Runner
	listener *listener.Service
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	db, err := database.Open(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	registry := providers.NewRegistry(cfg)
	authSvc := auth.NewService(db)
	workflowSvc := workflow.NewService(db)
	workflowBudgets := workflow.DefaultServerResourceBudgets()
	if cfg.Database.Driver == "sqlite" {
		workflowBudgets = workflow.DefaultSQLiteResourceBudgets()
	}
	if err := workflowSvc.EnsureResourceBudgets(ctx, workflowBudgets); err != nil {
		return nil, fmt.Errorf("ensure workflow resource budgets: %w", err)
	}
	ingestSvc := ingest.NewService(db, workflowSvc)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	catalogSvc := catalog.NewService(db, ingestSvc)
	if _, err := catalogSvc.BackfillScannerIdentities(ctx); err != nil {
		return nil, fmt.Errorf("backfill scanner identities: %w", err)
	}
	librarySvc := library.NewService(cfg, db, registry, nil, ingestSvc, workflowSvc)
	listenerSvc := listener.NewService(db, nil, librarySvc, registry)
	probeSvc := probe.NewService(db, registry, cfg.FFprobe, cfg.FFmpeg, ingestSvc)
	playbackSvc := playback.NewService(db, registry)
	searchSvc := search.NewService(db, librarySvc)
	metadataSvc := metadata.NewService(db, cfg.Metadata, settingsSvc, searchSvc, ingestSvc)
	healthSvc := health.NewService(db, registry, librarySvc, cfg.OpenList.BaseURL)
	catalogSvc.SetPersonProfileRefresher(metadataSvc)
	scheduleSvc := schedule.NewService(db, schedule.WithDispatcher(func(ctx context.Context, due schedule.DueSchedule) (database.Job, error) {
		status := schedule.StatusQueued
		var result library.ScheduledJobResult
		var err error
		switch due.Kind {
		case schedule.KindScan:
			result, err = librarySvc.RunScheduledScan(ctx, due)
		case schedule.KindLibraryCleanup:
			result, err = librarySvc.RunScheduledCleanup(ctx, due)
		case schedule.KindInvalidLinkCheck:
			result, err = librarySvc.RunScheduledInvalidLinkCheck(ctx, due)
			status = schedule.StatusCompleted
		default:
			return database.Job{}, fmt.Errorf("unsupported schedule kind %q", due.Kind)
		}
		if err != nil {
			return database.Job{}, err
		}
		now := time.Now().UTC()
		return database.Job{ID: due.ID, Kind: schedule.JobKindForSchedule(due.Kind), Status: status, PayloadJSON: result.Summary, AvailableAt: now, CreatedAt: now, UpdatedAt: now}, nil
	}))
	progressSvc := progress.NewService(db, searchSvc)
	workflowRunner := workflow.NewRunner(workflowSvc, workflow.RunnerConfig{Enabled: true, PollInterval: cfg.Worker.WorkflowPollInterval, LeaseDuration: cfg.Worker.WorkflowLeaseDuration})
	librarySvc.RegisterWorkflowHandlers(workflowRunner)
	workflowRunner.Register(workflow.TaskTypeProbeInventory, func(ctx context.Context, task database.WorkflowTask) error {
		var payload library.InventoryProbeBatchPayload
		if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
			return err
		}
		for _, fileID := range payload.FileIDs {
			if fileID == 0 {
				continue
			}
			if err := probeSvc.ProbeInventoryFile(ctx, fileID); err != nil {
				return err
			}
		}
		return nil
	})
	workflowRunner.Register(workflow.TaskTypeMatchMetadata, func(ctx context.Context, task database.WorkflowTask) error {
		var payload library.CatalogMatchBatchPayload
		if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
			return err
		}
		for _, itemID := range payload.ItemIDs {
			if itemID == 0 {
				continue
			}
			if _, err := metadataSvc.MatchCatalogItemOperation(ctx, itemID); err != nil {
				return err
			}
		}
		return nil
	})

	handler := httpapi.New(cfg, db, registry, authSvc, librarySvc, nil, playbackSvc, progressSvc, searchSvc, metadataSvc, settingsSvc, catalogSvc, ingestSvc, scheduleSvc, listenerSvc, healthSvc, workflowSvc)

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{
		cfg:      cfg,
		db:       db,
		server:   server,
		workflow: workflowRunner,
		listener: listenerSvc,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	sqlDB, err := a.db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	go func() {
		log.Printf("mibo-media-server listening on %s", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	if a.cfg.Worker.Enabled {
		go a.workflow.Run(ctx)
		go a.listener.StartLocalObserver(ctx)
		go a.listener.StartOpenListObserver(ctx)
	}

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.HTTP.ShutdownTimeout)
		defer cancel()
		return a.server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
