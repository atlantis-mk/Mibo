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
)

const (
	JobKindApplyStorageEventRefresh = "apply_storage_event_refresh"
	JobKindListenerReconcile        = "listener_reconcile"
	mergeWindow                     = 15 * time.Second
	defaultReconcileInterval        = 6 * time.Hour
)

type EventIngestInput struct {
	LibraryID uint   `json:"library_id"`
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	OldPath   string `json:"old_path"`
}

type storageEventRefreshPayload struct {
	LibraryID         uint      `json:"library_id"`
	RootPath          string    `json:"root_path"`
	FallbackFullSync  bool      `json:"fallback_full_sync"`
	Reason            string    `json:"reason"`
	WindowStartedAt   time.Time `json:"window_started_at"`
	WindowEndsAt      time.Time `json:"window_ends_at"`
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

	jobKey := refreshJobKey(payload.LibraryID, payload.RootPath, payload.FallbackFullSync)
	job, err := createQueuedJob(jobKey, JobKindApplyStorageEventRefresh, payload, payload.WindowEndsAt)
	if err != nil {
		return database.Job{}, err
	}
	if err := s.db.WithContext(ctx).Create(&job).Error; err != nil {
		return database.Job{}, err
	}
	return job, nil
}

func (s *Service) EnsureReconcileCoverage(ctx context.Context, libraries []database.Library) error {
	if s.db == nil {
		return errors.New("listener database unavailable")
	}
	for _, record := range libraries {
		jobKey := reconcileJobKey(record.ID)
		var existing database.Job
		err := s.db.WithContext(ctx).
			Where("job_key = ? AND kind = ? AND status IN ?", jobKey, JobKindListenerReconcile, []string{jobs.StatusQueued, jobs.StatusRunning}).
			Order("id desc").
			First(&existing).Error
		if err == nil {
			continue
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		scheduledFor := s.now().Add(defaultReconcileInterval)
		payload := reconcilePayload{LibraryID: record.ID, Reason: "listener_reconcile", ScheduledFor: scheduledFor}
		job, err := createQueuedJob(jobKey, JobKindListenerReconcile, payload, scheduledFor)
		if err != nil {
			return err
		}
		if err := s.db.WithContext(ctx).Create(&job).Error; err != nil {
			return err
		}
	}
	return nil
}

func refreshJobKey(libraryID uint, rootPath string, fallback bool) string {
	mode := "targeted"
	if fallback {
		mode = "fallback"
	}
	return fmt.Sprintf("listener-refresh:%d:%s:%s", libraryID, mode, strings.TrimSpace(rootPath))
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
