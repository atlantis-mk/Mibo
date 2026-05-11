package app

import (
	"context"
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
	ingest   *ingest.Service
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
	librarySvc := library.NewService(cfg, db, registry, nil, ingestSvc, workflowSvc)
	listenerSvc := listener.NewService(db, nil, librarySvc, registry)
	probeSvc := probe.NewService(db, registry, cfg.FFprobe, cfg.FFmpeg, ingestSvc)
	playbackSvc := playback.NewService(db, registry)
	searchSvc := search.NewService(db, librarySvc)
	metadataSvc := metadata.NewService(db, cfg.Metadata, settingsSvc, searchSvc, ingestSvc)
	librarySvc.SetInventoryProbeExecutor(probeSvc.ProbeInventoryFile)
	librarySvc.SetMetadataMatchExecutor(func(ctx context.Context, metadataItemID uint, libraryID uint) error {
		_, err := metadataSvc.MatchMetadataItemOperation(ctx, metadataItemID, libraryID)
		return err
	})
	healthSvc := health.NewService(db, registry, librarySvc, cfg.OpenList.BaseURL)
	catalogSvc.SetPersonProfileRefresher(metadataSvc)
	scheduleSvc := schedule.NewService(db, schedule.WithDispatcher(func(ctx context.Context, due schedule.DueSchedule) (database.Job, error) {
		status := schedule.StatusQueued
		var result library.ScheduledJobResult
		var err error
		switch due.Kind {
		case schedule.KindScan:
			result, err = librarySvc.RunScheduledScan(ctx, due)
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
	workflowRunner := workflow.NewRunner(workflowSvc, workflow.RunnerConfig{Enabled: true, PollInterval: cfg.Worker.WorkflowPollInterval, LeaseDuration: cfg.Worker.WorkflowLeaseDuration, TaskTimeout: cfg.Worker.WorkflowTaskTimeout, MaxConcurrent: cfg.Worker.WorkflowMaxConcurrent})
	librarySvc.RegisterWorkflowHandlers(workflowRunner)

	handler := httpapi.New(httpapi.Dependencies{
		Config:   cfg,
		DB:       db,
		Registry: registry,
		Auth:     authSvc,
		Catalog:  catalogSvc,
		Library:  librarySvc,
		Listener: listenerSvc,
		Ingest:   ingestSvc,
		Playback: playbackSvc,
		Progress: progressSvc,
		Search:   searchSvc,
		Metadata: metadataSvc,
		Schedule: scheduleSvc,
		Settings: settingsSvc,
		Health:   healthSvc,
		Workflow: workflowSvc,
	})

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{
		cfg:      cfg,
		db:       db,
		server:   server,
		ingest:   ingestSvc,
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
		go a.runIngestReconciler(ctx)
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

func (a *App) runIngestReconciler(ctx context.Context) {
	if a.ingest == nil {
		return
	}
	interval := a.cfg.Worker.PollInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	runOnce := func() {
		if _, err := a.ingest.ReconcileOnce(ctx, 100); err != nil && ctx.Err() == nil {
			log.Printf("ingest: reconcile dirty work: %v", err)
		}
	}
	runOnce()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runOnce()
		}
	}
}
