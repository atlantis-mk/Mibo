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
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/storageindex"
	"gorm.io/gorm"
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
	library *library.Service
	storage *providers.Registry
	index   *storageindex.Service
	planner *storageindex.Planner
	now     func() time.Time
}

func NewService(db *gorm.DB, _ any, librarySvc *library.Service, args ...any) *Service {
	svc := &Service{db: db, library: librarySvc, index: storageindex.NewService(db), planner: storageindex.NewPlanner(), now: func() time.Time { return time.Now().UTC() }}
	for _, arg := range args {
		if registry, ok := arg.(*providers.Registry); ok {
			svc.storage = registry
		}
	}
	return svc
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
	if enabled, err := s.realtimeRefreshEnabled(ctx, record.ID); err != nil {
		return database.Job{}, err
	} else if !enabled {
		return database.Job{}, nil
	}

	payload, err := buildStorageEventPayload(record, input, s.now())
	if err != nil {
		return database.Job{}, err
	}
	if err := s.recordStorageEventIndexHint(ctx, record, input); err != nil {
		return database.Job{}, err
	}
	return s.applyRefreshPayload(ctx, payload)
}

func (s *Service) recordStorageEventIndexHint(ctx context.Context, record database.Library, input EventIngestInput) error {
	if s.index == nil {
		return nil
	}
	var source database.MediaSource
	if err := s.db.WithContext(ctx).First(&source, record.MediaSourceID).Error; err != nil {
		return err
	}
	provider := strings.TrimSpace(source.Provider)
	kind := strings.TrimSpace(strings.ToLower(input.Kind))
	currentPath := strings.TrimSpace(input.Path)
	oldPath := strings.TrimSpace(input.OldPath)
	if (kind == "move" || kind == "rename") && oldPath != "" {
		if _, err := s.index.Find(ctx, record.ID, provider, oldPath); err == nil {
			if _, err := s.index.MarkMissing(ctx, record.ID, provider, oldPath); err != nil {
				return err
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	switch kind {
	case "create", "update", "move", "rename":
		if currentPath == "" {
			return nil
		}
		_, err := s.index.UpsertPresent(ctx, storageindex.ObservationInput{LibraryID: record.ID, StorageProvider: provider, StoragePath: currentPath})
		return err
	case "delete":
		if currentPath == "" {
			return nil
		}
		if _, err := s.index.Find(ctx, record.ID, provider, currentPath); err == nil {
			_, err = s.index.MarkMissing(ctx, record.ID, provider, currentPath)
			return err
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	return nil
}

func (s *Service) EnsureReconcileCoverage(ctx context.Context, libraries []database.Library) error {
	if s.db == nil {
		return errors.New("listener database unavailable")
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
	_, err = s.applyRefreshPayload(ctx, payload)
	return err
}

func (s *Service) applyRefreshPayload(ctx context.Context, payload storageEventRefreshPayload) (database.Job, error) {
	if s.library == nil {
		return database.Job{}, errors.New("listener library service unavailable")
	}
	if payload.LibraryID == 0 {
		return database.Job{}, fmt.Errorf("listener refresh payload missing library_id")
	}
	if payload.FallbackFullSync {
		return s.library.QueueLibraryScanWithReason(ctx, payload.LibraryID, library.WorkflowReasonStorageRefresh)
	}
	return s.library.QueueTargetedRefresh(ctx, payload.LibraryID, payload.RootPath, payload.Reason)
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
	if _, err := s.library.QueueLibraryScanWithReason(ctx, payload.LibraryID, library.WorkflowReasonStorageRefresh); err != nil {
		return err
	}

	return nil
}

func (s *Service) ensureReconcileCoverageForLibrary(ctx context.Context, record database.Library) error {
	return nil
}

func (s *Service) realtimeRefreshEnabled(ctx context.Context, libraryID uint) (bool, error) {
	if s.library == nil {
		return true, nil
	}
	config, err := s.library.EffectiveLibraryConfig(ctx, libraryID)
	if err != nil {
		return false, err
	}
	return config.ScanPolicy.RealtimeMonitorEnabled, nil
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
		Status:      "queued",
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
