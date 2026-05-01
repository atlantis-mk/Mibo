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
	"github.com/atlan/mibo-media-server/internal/jobs"
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
	"github.com/atlan/mibo-media-server/internal/worker"
	"gorm.io/gorm"
)

type App struct {
	cfg      config.Config
	db       *gorm.DB
	server   *http.Server
	worker   *worker.Runner
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
	jobsSvc := jobs.NewService(db)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	catalogSvc := catalog.NewService(db)
	if _, err := catalogSvc.BackfillScannerIdentities(ctx); err != nil {
		return nil, fmt.Errorf("backfill scanner identities: %w", err)
	}
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	listenerSvc := listener.NewService(db, jobsSvc, librarySvc, registry)
	probeSvc := probe.NewService(db, registry, cfg.FFprobe, cfg.FFmpeg)
	playbackSvc := playback.NewService(db, registry)
	searchSvc := search.NewService(db, librarySvc)
	metadataSvc := metadata.NewService(db, cfg.Metadata, settingsSvc, searchSvc)
	healthSvc := health.NewService(db, registry, librarySvc, jobsSvc, cfg.OpenList.BaseURL)
	catalogSvc.SetPersonProfileRefresher(metadataSvc)
	scheduleSvc := schedule.NewService(db, schedule.WithJobs(jobsSvc))
	progressSvc := progress.NewService(db, searchSvc)
	workerRunner := worker.NewRunner(cfg.Worker, jobsSvc, librarySvc, metadataSvc, probeSvc, settingsSvc, catalogSvc, searchSvc, scheduleSvc, listenerSvc)

	handler := httpapi.New(cfg, db, registry, authSvc, librarySvc, jobsSvc, playbackSvc, progressSvc, searchSvc, metadataSvc, settingsSvc, catalogSvc, scheduleSvc, listenerSvc, healthSvc)

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{
		cfg:      cfg,
		db:       db,
		server:   server,
		worker:   workerRunner,
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
		go a.worker.Run(ctx)
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
