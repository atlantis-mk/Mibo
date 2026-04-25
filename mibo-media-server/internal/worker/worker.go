package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/listener"
	"github.com/atlan/mibo-media-server/internal/metadata"
	"github.com/atlan/mibo-media-server/internal/probe"
	"github.com/atlan/mibo-media-server/internal/schedule"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
)

type Runner struct {
	cfg      config.WorkerConfig
	jobs     *jobs.Service
	catalog  *catalog.Service
	library  *library.Service
	listener *listener.Service
	metadata *metadata.Service
	probe    *probe.Service
	schedule *schedule.Service
	search   *search.Service
	settings *settings.Service
	interval time.Duration
}

func NewRunner(cfg config.WorkerConfig, jobsSvc *jobs.Service, librarySvc *library.Service, metadataSvc *metadata.Service, probeSvc *probe.Service, settingsSvc *settings.Service, args ...any) *Runner {
	interval := cfg.PollInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}

	runner := &Runner{
		cfg:      cfg,
		jobs:     jobsSvc,
		library:  librarySvc,
		metadata: metadataSvc,
		probe:    probeSvc,
		settings: settingsSvc,
		interval: interval,
	}
	for _, arg := range args {
		if catalogSvc, ok := arg.(*catalog.Service); ok {
			runner.catalog = catalogSvc
		}
		if searchSvc, ok := arg.(*search.Service); ok {
			runner.search = searchSvc
		}
		if scheduleSvc, ok := arg.(*schedule.Service); ok {
			runner.schedule = scheduleSvc
		}
		if listenerSvc, ok := arg.(*listener.Service); ok {
			runner.listener = listenerSvc
		}
	}
	return runner
}

func (r *Runner) Run(ctx context.Context) {
	pollTicker := time.NewTicker(r.interval)
	defer pollTicker.Stop()

	r.runOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTicker.C:
			r.runOnce(ctx)
		}
	}
}

func (r *Runner) getRefreshInterval(ctx context.Context) time.Duration {
	if r.settings != nil {
		scanSettings, err := r.settings.GetScanSettings(ctx)
		if err == nil && scanSettings.RefreshIntervalHours > 0 {
			return time.Duration(scanSettings.RefreshIntervalHours) * time.Hour
		}
	}
	if r.cfg.RefreshIntervalHours > 0 {
		return time.Duration(r.cfg.RefreshIntervalHours) * time.Hour
	}
	return 0
}

func (r *Runner) triggerScheduledScans(ctx context.Context) {
	libraries, err := r.library.ListActiveLibraries(ctx)
	if err != nil {
		log.Printf("worker: failed to list active libraries for scheduled scan: %v", err)
		return
	}
	for _, lib := range libraries {
		if _, err := r.library.QueueLibraryScan(ctx, lib.ID); err != nil {
			log.Printf("worker: scheduled scan triggered for library %d: %v", lib.ID, err)
		} else {
			log.Printf("worker: scheduled scan triggered for library %d", lib.ID)
		}
	}
}

func (r *Runner) RunOnce(ctx context.Context) {
	r.runOnce(ctx)
}

func (r *Runner) runOnce(ctx context.Context) {
	r.enqueueDueSchedules(ctx)
	r.ensureListenerReconcileCoverage(ctx)
	for {
		job, err := r.jobs.ClaimNext(ctx)
		if err != nil {
			if errors.Is(err, jobs.ErrNoAvailableJob) || errors.Is(err, context.Canceled) {
				return
			}
			log.Printf("worker: claim job: %v", err)
			return
		}

		if err := r.handleJob(ctx, job); err != nil {
			log.Printf("worker: job %d (%s) failed: %v", job.ID, job.Kind, err)
			r.markScheduleRunFinished(ctx, job, schedule.StatusFailed, err.Error())
			if failErr := r.jobs.Fail(ctx, job.ID, err); failErr != nil {
				log.Printf("worker: mark job %d failed: %v", job.ID, failErr)
			}
			continue
		}

		if err := r.jobs.Complete(ctx, job.ID); err != nil {
			log.Printf("worker: mark job %d completed: %v", job.ID, err)
		}
		r.markScheduleRunFinished(ctx, job, schedule.StatusCompleted, "completed")
	}
}

func (r *Runner) handleJob(ctx context.Context, job database.Job) error {
	r.markScheduleRunRunning(ctx, job)
	switch job.Kind {
	case library.JobKindSyncLibrary:
		return r.library.RunSyncLibrary(ctx, job)
	case library.JobKindTargetedRefresh:
		return r.library.RunTargetedRefresh(ctx, job)
	case listener.JobKindApplyStorageEventRefresh:
		if r.listener == nil {
			return errors.New("listener service unavailable")
		}
		return r.listener.ApplyStorageEventRefresh(ctx, job)
	case listener.JobKindListenerReconcile:
		if r.listener == nil {
			return errors.New("listener service unavailable")
		}
		return r.listener.RunReconcile(ctx, job)
	case library.JobKindMatchMediaItem:
		var payload struct {
			MediaItemID uint `json:"media_item_id"`
		}
		if err := decodeJobPayload(job.PayloadJSON, &payload); err != nil {
			return err
		}
		return r.metadata.MatchItem(ctx, payload.MediaItemID)
	case library.JobKindRefetchMediaItem:
		var payload struct {
			MediaItemID uint `json:"media_item_id"`
		}
		if err := decodeJobPayload(job.PayloadJSON, &payload); err != nil {
			return err
		}
		return r.metadata.RefetchItem(ctx, payload.MediaItemID)
	case library.JobKindReindexSearchDocument:
		if r.search == nil {
			return errors.New("search service unavailable")
		}
		var payload struct {
			MediaItemID uint `json:"media_item_id"`
		}
		if err := decodeJobPayload(job.PayloadJSON, &payload); err != nil {
			return err
		}
		return r.search.ReindexMediaItem(ctx, payload.MediaItemID)
	case library.JobKindReindexLibrarySearch:
		if r.search == nil {
			return errors.New("search service unavailable")
		}
		var payload struct {
			LibraryID uint   `json:"library_id"`
			RootPath  string `json:"root_path"`
		}
		if err := decodeJobPayload(job.PayloadJSON, &payload); err != nil {
			return err
		}
		return r.search.ReindexLibrary(ctx, payload.LibraryID, payload.RootPath)
	case library.JobKindCatalogRefreshItemProjection:
		if r.catalog == nil {
			return errors.New("catalog service unavailable")
		}
		var payload catalog.ItemProjectionRefreshPayload
		if err := decodeJobPayload(job.PayloadJSON, &payload); err != nil {
			return err
		}
		return r.catalog.RefreshItemProjection(ctx, payload.ItemID)
	case library.JobKindCatalogRefreshLibraryProjection:
		if r.catalog == nil {
			return errors.New("catalog service unavailable")
		}
		var payload catalog.LibraryProjectionRefreshPayload
		if err := decodeJobPayload(job.PayloadJSON, &payload); err != nil {
			return err
		}
		return r.catalog.RefreshLibraryProjection(ctx, payload.LibraryID, payload.RootPath)
	case "probe_media_file":
		var payload struct {
			MediaFileID uint `json:"media_file_id"`
		}
		if err := decodeJobPayload(job.PayloadJSON, &payload); err != nil {
			return err
		}
		return r.probe.ProbeFile(ctx, payload.MediaFileID)
	case schedule.JobKindForSchedule(schedule.KindScan):
		due, err := schedule.ParseSchedulePayload(job.PayloadJSON)
		if err != nil {
			return err
		}
		_, err = r.library.RunScheduledScan(ctx, due)
		return err
	case schedule.JobKindForSchedule(schedule.KindLibraryCleanup):
		due, err := schedule.ParseSchedulePayload(job.PayloadJSON)
		if err != nil {
			return err
		}
		_, err = r.library.RunScheduledCleanup(ctx, due)
		return err
	case schedule.JobKindForSchedule(schedule.KindInvalidLinkCheck):
		due, err := schedule.ParseSchedulePayload(job.PayloadJSON)
		if err != nil {
			return err
		}
		_, err = r.library.RunScheduledInvalidLinkCheck(ctx, due)
		return err
	case schedule.JobKindForSchedule(schedule.KindMetadataRefetch):
		due, err := schedule.ParseSchedulePayload(job.PayloadJSON)
		if err != nil {
			return err
		}
		_, err = r.metadata.RunScheduledMetadataRefetch(ctx, due)
		return err
	case schedule.JobKindForSchedule(schedule.KindTrailerSync):
		due, err := schedule.ParseSchedulePayload(job.PayloadJSON)
		if err != nil {
			return err
		}
		_, err = r.metadata.RunScheduledTrailerSync(ctx, due)
		return err
	case schedule.JobKindForSchedule(schedule.KindArtworkRefresh):
		due, err := schedule.ParseSchedulePayload(job.PayloadJSON)
		if err != nil {
			return err
		}
		_, err = r.metadata.RunScheduledArtworkRefresh(ctx, due)
		return err
	default:
		return errors.New("unsupported job kind: " + job.Kind)
	}
}

func (r *Runner) enqueueDueSchedules(ctx context.Context) {
	if r.schedule == nil {
		return
	}
	if _, err := r.schedule.ClaimDueRuns(ctx, 20); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("worker: claim due schedules: %v", err)
	}
}

func (r *Runner) markScheduleRunRunning(ctx context.Context, job database.Job) {
	if r.schedule == nil {
		return
	}
	if _, err := schedule.ParseSchedulePayload(job.PayloadJSON); err != nil {
		return
	}
	if err := r.schedule.MarkRunRunning(ctx, job.ID); err != nil {
		log.Printf("worker: mark schedule job %d running: %v", job.ID, err)
	}
}

func (r *Runner) markScheduleRunFinished(ctx context.Context, job database.Job, status string, message string) {
	if r.schedule == nil {
		return
	}
	if _, err := schedule.ParseSchedulePayload(job.PayloadJSON); err != nil {
		return
	}
	if err := r.schedule.MarkRunFinished(ctx, schedule.RunTransitionInput{JobID: job.ID, Status: status, Message: message, Finished: time.Now().UTC()}); err != nil {
		log.Printf("worker: mark schedule job %d %s: %v", job.ID, status, err)
	}
}

func (r *Runner) ensureListenerReconcileCoverage(ctx context.Context) {
	if r.listener == nil || r.library == nil {
		return
	}
	libraries, err := r.library.ListActiveLibraries(ctx)
	if err != nil {
		log.Printf("worker: list active libraries for listener reconcile: %v", err)
		return
	}
	if err := r.listener.EnsureReconcileCoverage(ctx, libraries); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("worker: ensure listener reconcile coverage: %v", err)
	}
}

func decodeJobPayload(payload string, out any) error {
	return json.Unmarshal([]byte(payload), out)
}
