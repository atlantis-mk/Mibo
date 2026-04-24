package listener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/library"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	JobKindApplyStorageEventRefresh = "apply_storage_event_refresh"
	JobKindListenerReconcile        = "listener_reconcile"
	mergeWindow                     = 15 * time.Second
	defaultReconcileInterval        = 6 * time.Hour
	refreshActiveIntentPrefix       = "listener-refresh-active"
	reconcileActiveIntentPrefix     = "listener-reconcile-active"
)

type EventIngestInput struct {
	LibraryID uint   `json:"library_id"`
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	OldPath   string `json:"old_path"`
}

type storageEventRefreshPayload struct {
	LibraryID        uint      `json:"library_id"`
	RootPath         string    `json:"root_path"`
	FallbackFullSync bool      `json:"fallback_full_sync"`
	Reason           string    `json:"reason"`
	WindowStartedAt  time.Time `json:"window_started_at"`
	WindowEndsAt     time.Time `json:"window_ends_at"`
}

type reconcilePayload struct {
	LibraryID    uint      `json:"library_id"`
	Reason       string    `json:"reason"`
	ScheduledFor time.Time `json:"scheduled_for"`
}

type Service struct {
	db      *gorm.DB
	jobs    *jobs.Service
	library *library.Service
	now     func() time.Time
}

func NewService(db *gorm.DB, jobsSvc *jobs.Service, librarySvc *library.Service) *Service {
	return &Service{db: db, jobs: jobsSvc, library: librarySvc, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) RecordStorageEvent(ctx context.Context, input EventIngestInput) (database.Job, error) {
	if s.db == nil {
		return database.Job{}, errors.New("listener database unavailable")
	}
	if input.LibraryID == 0 {
		return database.Job{}, fmt.Errorf("library_id is required")
	}

	var record database.Library
	if err := s.db.WithContext(ctx).First(&record, input.LibraryID).Error; err != nil {
		return database.Job{}, err
	}

	payload, err := buildStorageEventPayload(record, input, s.now())
	if err != nil {
		return database.Job{}, err
	}

	var stored database.Job
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		intentKey := refreshActiveIntentKey(payload.LibraryID)
		if _, err := upsertActiveIntent(tx, intentKey, JobKindApplyStorageEventRefresh); err != nil {
			return err
		}

		var existing []database.Job
		if err := tx.
			Where("kind = ? AND status IN ?", JobKindApplyStorageEventRefresh, []string{jobs.StatusQueued, jobs.StatusRunning}).
			Order("id asc").
			Find(&existing).Error; err != nil {
			return err
		}

		active := make([]database.Job, 0, len(existing))
		for _, job := range existing {
			current, err := decodeRefreshPayload(job.PayloadJSON)
			if err != nil {
				return err
			}
			if current.LibraryID != payload.LibraryID || !current.WindowEndsAt.After(payload.WindowStartedAt) {
				continue
			}
			active = append(active, job)
			payload = mergeRefreshPayload(record.RootPath, payload, current)
		}

		payload.WindowEndsAt = payload.WindowStartedAt.Add(mergeWindow)
		jobKey := refreshJobKey(payload.LibraryID, payload.RootPath, payload.FallbackFullSync)
		if len(active) == 0 {
			created, err := createQueuedJob(jobKey, JobKindApplyStorageEventRefresh, payload, payload.WindowEndsAt)
			if err != nil {
				return err
			}
			if err := tx.Create(&created).Error; err != nil {
				return err
			}
			if err := updateActiveIntentJob(tx, intentKey, created.ID); err != nil {
				return err
			}
			stored = created
			return nil
		}

		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal listener job payload: %w", err)
		}
		keeper := active[0]
		if err := tx.Model(&database.Job{}).
			Where("id = ?", keeper.ID).
			Updates(map[string]any{"job_key": jobKey, "payload_json": string(payloadJSON), "available_at": payload.WindowEndsAt.UTC()}).Error; err != nil {
			return err
		}
		if len(active) > 1 {
			ids := make([]uint, 0, len(active)-1)
			for _, duplicate := range active[1:] {
				ids = append(ids, duplicate.ID)
			}
			if err := tx.Where("id IN ?", ids).Delete(&database.Job{}).Error; err != nil {
				return err
			}
		}
		if err := tx.First(&stored, keeper.ID).Error; err != nil {
			return err
		}
		if err := updateActiveIntentJob(tx, intentKey, stored.ID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return database.Job{}, err
	}
	return stored, nil
}

func (s *Service) EnsureReconcileCoverage(ctx context.Context, libraries []database.Library) error {
	if s.db == nil {
		return errors.New("listener database unavailable")
	}
	for _, record := range libraries {
		if err := s.ensureReconcileCoverageForLibrary(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ApplyStorageEventRefresh(ctx context.Context, job database.Job) error {
	if s.library == nil {
		return errors.New("listener library service unavailable")
	}
	payload, err := decodeRefreshPayload(job.PayloadJSON)
	if err != nil {
		return err
	}
	if payload.LibraryID == 0 {
		return fmt.Errorf("listener refresh payload missing library_id")
	}
	if payload.FallbackFullSync {
		_, err = s.library.QueueLibraryScan(ctx, payload.LibraryID)
		return err
	}
	_, err = s.library.QueueTargetedRefresh(ctx, payload.LibraryID, payload.RootPath, payload.Reason)
	return err
}

func (s *Service) RunReconcile(ctx context.Context, job database.Job) error {
	if s.library == nil {
		return errors.New("listener library service unavailable")
	}
	if s.db == nil {
		return errors.New("listener database unavailable")
	}
	payload, err := decodeReconcilePayload(job.PayloadJSON)
	if err != nil {
		return err
	}
	if payload.LibraryID == 0 {
		return fmt.Errorf("listener reconcile payload missing library_id")
	}
	if _, err := s.library.QueueLibraryScan(ctx, payload.LibraryID); err != nil {
		return err
	}

	nextScheduledFor := s.now().Add(defaultReconcileInterval)
	nextPayload := reconcilePayload{LibraryID: payload.LibraryID, Reason: payload.Reason, ScheduledFor: nextScheduledFor}
	jobKey := reconcileJobKey(payload.LibraryID)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		intentKey := reconcileActiveIntentKey(payload.LibraryID)
		if _, err := upsertActiveIntent(tx, intentKey, JobKindListenerReconcile); err != nil {
			return err
		}

		var existing database.Job
		err := tx.
			Where("job_key = ? AND kind = ? AND status = ? AND id <> ?", jobKey, JobKindListenerReconcile, jobs.StatusQueued, job.ID).
			Order("id desc").
			First(&existing).Error
		if err == nil {
			payloadJSON, err := json.Marshal(nextPayload)
			if err != nil {
				return fmt.Errorf("marshal listener reconcile payload: %w", err)
			}
			if err := tx.Model(&database.Job{}).
				Where("id = ?", existing.ID).
				Updates(map[string]any{"payload_json": string(payloadJSON), "available_at": nextScheduledFor.UTC(), "job_key": jobKey}).Error; err != nil {
				return err
			}
			return updateActiveIntentJob(tx, intentKey, existing.ID)
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		queued, err := createQueuedJob(jobKey, JobKindListenerReconcile, nextPayload, nextScheduledFor)
		if err != nil {
			return err
		}
		if err := tx.Create(&queued).Error; err != nil {
			return err
		}
		return updateActiveIntentJob(tx, intentKey, queued.ID)
	})
}

func (s *Service) ensureReconcileCoverageForLibrary(ctx context.Context, record database.Library) error {
	jobKey := reconcileJobKey(record.ID)
	intentKey := reconcileActiveIntentKey(record.ID)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := upsertActiveIntent(tx, intentKey, JobKindListenerReconcile); err != nil {
			return err
		}

		var existing []database.Job
		if err := tx.
			Where("job_key = ? AND kind = ? AND status IN ?", jobKey, JobKindListenerReconcile, []string{jobs.StatusQueued, jobs.StatusRunning}).
			Order("id desc").
			Find(&existing).Error; err != nil {
			return err
		}
		if len(existing) > 0 {
			keeper := existing[0]
			if len(existing) > 1 {
				ids := make([]uint, 0, len(existing)-1)
				for _, duplicate := range existing[1:] {
					ids = append(ids, duplicate.ID)
				}
				if err := tx.Where("id IN ?", ids).Delete(&database.Job{}).Error; err != nil {
					return err
				}
			}
			return updateActiveIntentJob(tx, intentKey, keeper.ID)
		}

		scheduledFor := s.now().Add(defaultReconcileInterval)
		payload := reconcilePayload{LibraryID: record.ID, Reason: "listener_reconcile", ScheduledFor: scheduledFor}
		job, err := createQueuedJob(jobKey, JobKindListenerReconcile, payload, scheduledFor)
		if err != nil {
			return err
		}
		if err := tx.Create(&job).Error; err != nil {
			return err
		}
		return updateActiveIntentJob(tx, intentKey, job.ID)
	})
}

func refreshJobKey(libraryID uint, rootPath string, fallback bool) string {
	mode := "targeted"
	if fallback {
		mode = "fallback"
	}
	return fmt.Sprintf("listener-refresh:%d:%s:%s", libraryID, mode, strings.TrimSpace(rootPath))
}

func refreshActiveIntentKey(libraryID uint) string {
	return fmt.Sprintf("%s:%d", refreshActiveIntentPrefix, libraryID)
}

func reconcileActiveIntentKey(libraryID uint) string {
	return fmt.Sprintf("%s:%d", reconcileActiveIntentPrefix, libraryID)
}

func upsertActiveIntent(tx *gorm.DB, intentKey string, kind string) (database.JobActiveIntent, error) {
	intent := database.JobActiveIntent{IntentKey: intentKey, Kind: kind}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "intent_key"}},
		DoUpdates: clause.Assignments(map[string]any{"kind": kind}),
	}).Create(&intent).Error; err != nil {
		return database.JobActiveIntent{}, err
	}
	if err := tx.Where("intent_key = ?", intentKey).First(&intent).Error; err != nil {
		return database.JobActiveIntent{}, err
	}
	return intent, nil
}

func updateActiveIntentJob(tx *gorm.DB, intentKey string, jobID uint) error {
	return tx.Model(&database.JobActiveIntent{}).
		Where("intent_key = ?", intentKey).
		Update("job_id", jobID).Error
}

func reconcileJobKey(libraryID uint) string {
	return fmt.Sprintf("listener-reconcile:%d", libraryID)
}

func createQueuedJob(jobKey string, kind string, payload any, availableAt time.Time) (database.Job, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return database.Job{}, fmt.Errorf("marshal listener job payload: %w", err)
	}
	return database.Job{
		JobKey:      jobKey,
		Kind:        kind,
		Status:      jobs.StatusQueued,
		PayloadJSON: string(payloadJSON),
		AvailableAt: availableAt.UTC(),
	}, nil
}

func buildStorageEventPayload(record database.Library, input EventIngestInput, now time.Time) (storageEventRefreshPayload, error) {
	kind := strings.TrimSpace(strings.ToLower(input.Kind))
	if kind == "" {
		return storageEventRefreshPayload{}, fmt.Errorf("kind is required")
	}
	rootPath, fallback, err := normalizeStorageEventRoot(record.RootPath, kind, input.Path, input.OldPath)
	if err != nil {
		return storageEventRefreshPayload{}, err
	}
	windowStartedAt := now.UTC()
	return storageEventRefreshPayload{
		LibraryID:        record.ID,
		RootPath:         rootPath,
		FallbackFullSync: fallback,
		Reason:           "storage_event",
		WindowStartedAt:  windowStartedAt,
		WindowEndsAt:     windowStartedAt.Add(mergeWindow),
	}, nil
}

func mergeRefreshPayload(libraryRoot string, base storageEventRefreshPayload, other storageEventRefreshPayload) storageEventRefreshPayload {
	if other.WindowStartedAt.Before(base.WindowStartedAt) {
		base.WindowStartedAt = other.WindowStartedAt
	}
	base.FallbackFullSync = base.FallbackFullSync || other.FallbackFullSync
	if base.FallbackFullSync {
		base.RootPath = cleanPath(libraryRoot)
		return base
	}
	base.RootPath = commonAncestorPath(base.RootPath, other.RootPath, libraryRoot)
	base.RootPath = targetedEventRoot(base.RootPath, libraryRoot)
	return base
}

func decodeRefreshPayload(raw string) (storageEventRefreshPayload, error) {
	var payload storageEventRefreshPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return storageEventRefreshPayload{}, fmt.Errorf("decode listener refresh payload: %w", err)
	}
	return payload, nil
}

func decodeReconcilePayload(raw string) (reconcilePayload, error) {
	var payload reconcilePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return reconcilePayload{}, fmt.Errorf("decode listener reconcile payload: %w", err)
	}
	return payload, nil
}

func normalizeStorageEventRoot(libraryRoot string, kind string, currentPath string, oldPath string) (string, bool, error) {
	cleanLibraryRoot := cleanPath(libraryRoot)
	cleanCurrent := cleanPath(currentPath)
	cleanOld := cleanPath(oldPath)

	switch kind {
	case "create", "update", "delete":
		if cleanCurrent == "" {
			return "", false, fmt.Errorf("path is required")
		}
		return targetedEventRoot(cleanCurrent, cleanLibraryRoot), false, nil
	case "move", "rename":
		if cleanCurrent == "" || cleanOld == "" {
			return cleanLibraryRoot, true, nil
		}
		ancestor := commonAncestorPath(cleanOld, cleanCurrent, cleanLibraryRoot)
		return targetedEventRoot(ancestor, cleanLibraryRoot), false, nil
	default:
		return cleanLibraryRoot, true, nil
	}
}

func cleanPath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	clean := filepath.Clean(trimmed)
	if clean == "." {
		return ""
	}
	return clean
}

func commonAncestorPath(left string, right string, libraryRoot string) string {
	cleanLeft := cleanPath(left)
	cleanRight := cleanPath(right)
	cleanRoot := cleanPath(libraryRoot)
	if cleanLeft == "" || cleanRight == "" {
		return cleanRoot
	}
	shared := cleanRoot
	leftParts := strings.Split(cleanLeft, string(filepath.Separator))
	if strings.HasPrefix(cleanLeft, string(filepath.Separator)) {
		leftParts = leftParts[1:]
	}
	rightParts := strings.Split(cleanRight, string(filepath.Separator))
	if strings.HasPrefix(cleanRight, string(filepath.Separator)) {
		rightParts = rightParts[1:]
	}
	rootParts := strings.Split(cleanRoot, string(filepath.Separator))
	if strings.HasPrefix(cleanRoot, string(filepath.Separator)) {
		rootParts = rootParts[1:]
	}
	sharedParts := make([]string, 0, min(len(leftParts), len(rightParts)))
	for idx := 0; idx < len(leftParts) && idx < len(rightParts); idx++ {
		if leftParts[idx] != rightParts[idx] {
			break
		}
		sharedParts = append(sharedParts, leftParts[idx])
	}
	if len(sharedParts) < len(rootParts) {
		return cleanRoot
	}
	shared = filepath.Join(append([]string{string(filepath.Separator)}, sharedParts...)...)
	if cleanRoot != "" {
		shared = clampWithinLibraryRoot(shared, cleanRoot)
	}
	return shared
}

func clampWithinLibraryRoot(candidate string, libraryRoot string) string {
	cleanCandidate := cleanPath(candidate)
	cleanRoot := cleanPath(libraryRoot)
	if cleanRoot == "" {
		return cleanCandidate
	}
	if cleanCandidate == "" || cleanCandidate == string(filepath.Separator) {
		return cleanRoot
	}
	rel, err := filepath.Rel(cleanRoot, cleanCandidate)
	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return cleanRoot
	}
	return cleanCandidate
}

func targetedEventRoot(value string, libraryRoot string) string {
	clean := clampWithinLibraryRoot(value, libraryRoot)
	if clean == "" {
		return clampWithinLibraryRoot(libraryRoot, libraryRoot)
	}
	if ext := filepath.Ext(clean); ext != "" {
		return clampWithinLibraryRoot(filepath.Dir(clean), libraryRoot)
	}
	return clean
}
