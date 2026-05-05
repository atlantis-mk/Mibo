package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

type ScopeKind string

const (
	ScopeGlobal  ScopeKind = "global"
	ScopeLibrary ScopeKind = "library"
)

const (
	KindScan             = "scan"
	KindLibraryCleanup   = "library_cleanup"
	KindInvalidLinkCheck = "invalid_link_check"

	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

type Service struct {
	db         *gorm.DB
	dispatcher Dispatcher
	now        func() time.Time
}

type Dispatcher func(context.Context, DueSchedule) (database.Job, error)

type Option func(*Service)

func WithNow(fn func() time.Time) Option {
	return func(s *Service) {
		if fn != nil {
			s.now = fn
		}
	}
}

func NewService(db *gorm.DB, opts ...Option) *Service {
	svc := &Service{db: db, now: func() time.Time { return time.Now().UTC() }}
	for _, opt := range opts {
		if opt != nil {
			opt(svc)
		}
	}
	return svc
}

func WithJobs(_ any) Option {
	return func(s *Service) {
	}
}

func WithDispatcher(dispatcher Dispatcher) Option {
	return func(s *Service) {
		s.dispatcher = dispatcher
	}
}

type CreateScheduleInput struct {
	Name      string        `json:"name"`
	Kind      string        `json:"kind"`
	ScopeKind ScopeKind     `json:"scope_kind"`
	LibraryID *uint         `json:"library_id,omitempty"`
	Enabled   bool          `json:"enabled"`
	Frequency FrequencySpec `json:"frequency"`
}

type UpdateScheduleInput struct {
	Name      *string        `json:"name,omitempty"`
	Kind      *string        `json:"kind,omitempty"`
	ScopeKind *ScopeKind     `json:"scope_kind,omitempty"`
	LibraryID *uint          `json:"library_id,omitempty"`
	Enabled   *bool          `json:"enabled,omitempty"`
	Frequency *FrequencySpec `json:"frequency,omitempty"`
}

type RecordRunResultInput struct {
	Status       string
	JobID        *uint
	ErrorSummary string
	StartedAt    time.Time
	FinishedAt   time.Time
}

type DueSchedule struct {
	ID        uint
	Kind      string
	ScopeKind ScopeKind
	LibraryID *uint
}

type Schedule struct {
	ID                  uint          `json:"id"`
	Name                string        `json:"name"`
	Kind                string        `json:"kind"`
	ScopeKind           ScopeKind     `json:"scope_kind"`
	LibraryID           *uint         `json:"library_id,omitempty"`
	Frequency           FrequencySpec `json:"frequency"`
	Enabled             bool          `json:"enabled"`
	NextRunAt           *time.Time    `json:"next_run_at,omitempty"`
	LatestRunStatus     string        `json:"latest_run_status"`
	LatestRunMessage    string        `json:"latest_run_message"`
	LatestJobID         *uint         `json:"latest_job_id,omitempty"`
	LatestRunStartedAt  *time.Time    `json:"latest_run_started_at,omitempty"`
	LatestRunFinishedAt *time.Time    `json:"latest_run_finished_at,omitempty"`
	RecentRuns          []ScheduleRun `json:"recent_runs,omitempty"`
	CreatedAt           time.Time     `json:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at"`
}

type ScheduleRun struct {
	ID           uint       `json:"id"`
	ScheduleID   uint       `json:"schedule_id"`
	Status       string     `json:"status"`
	JobID        *uint      `json:"job_id,omitempty"`
	Job          *JobDetail `json:"job,omitempty"`
	ErrorSummary string     `json:"error_summary"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type JobDetail struct {
	ID           uint       `json:"id"`
	JobKey       string     `json:"job_key"`
	Kind         string     `json:"kind"`
	Status       string     `json:"status"`
	PayloadJSON  string     `json:"payload_json"`
	ErrorMessage string     `json:"error_message"`
	Attempts     int        `json:"attempts"`
	AvailableAt  time.Time  `json:"available_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type RunNowResult struct {
	Run ScheduleRun  `json:"run"`
	Job database.Job `json:"job"`
}

type RunTransitionInput struct {
	JobID    uint
	Status   string
	Message  string
	Finished time.Time
}

func (s *Service) Create(ctx context.Context, input CreateScheduleInput) (Schedule, error) {
	if err := validateCreateInput(input); err != nil {
		return Schedule{}, err
	}

	record := database.Schedule{
		Name:          strings.TrimSpace(input.Name),
		Kind:          normalizeKind(input.Kind),
		ScopeKind:     string(input.ScopeKind),
		LibraryID:     input.LibraryID,
		FrequencyKind: string(input.Frequency.Kind),
		TimeOfDay:     strings.TrimSpace(input.Frequency.TimeOfDay),
		Weekday:       input.Frequency.Weekday,
		DayOfMonth:    input.Frequency.DayOfMonth,
		Enabled:       input.Enabled,
	}
	if record.Enabled {
		nextRun, err := input.Frequency.NextRunFrom(s.now())
		if err != nil {
			return Schedule{}, err
		}
		record.NextRunAt = &nextRun
	}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		return Schedule{}, err
	}
	return s.Get(ctx, record.ID)
}

func (s *Service) Update(ctx context.Context, id uint, input UpdateScheduleInput) (Schedule, error) {
	var record database.Schedule
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&record).Error; err != nil {
		return Schedule{}, err
	}

	merged := CreateScheduleInput{
		Name:      record.Name,
		Kind:      record.Kind,
		ScopeKind: ScopeKind(record.ScopeKind),
		LibraryID: record.LibraryID,
		Enabled:   record.Enabled,
		Frequency: FrequencySpec{Kind: FrequencyKind(record.FrequencyKind), TimeOfDay: record.TimeOfDay, Weekday: record.Weekday, DayOfMonth: record.DayOfMonth},
	}
	if input.Name != nil {
		merged.Name = *input.Name
	}
	if input.Kind != nil {
		merged.Kind = *input.Kind
	}
	if input.ScopeKind != nil {
		merged.ScopeKind = *input.ScopeKind
	}
	if input.Enabled != nil {
		merged.Enabled = *input.Enabled
	}
	if input.Frequency != nil {
		merged.Frequency = *input.Frequency
	}
	if input.LibraryID != nil || merged.ScopeKind == ScopeGlobal {
		merged.LibraryID = input.LibraryID
	}
	if err := validateCreateInput(merged); err != nil {
		return Schedule{}, err
	}

	updates := map[string]any{
		"name":           strings.TrimSpace(merged.Name),
		"kind":           normalizeKind(merged.Kind),
		"scope_kind":     string(merged.ScopeKind),
		"library_id":     merged.LibraryID,
		"frequency_kind": string(merged.Frequency.Kind),
		"time_of_day":    strings.TrimSpace(merged.Frequency.TimeOfDay),
		"weekday":        merged.Frequency.Weekday,
		"day_of_month":   merged.Frequency.DayOfMonth,
		"enabled":        merged.Enabled,
	}
	if merged.Enabled {
		nextRun, err := merged.Frequency.NextRunFrom(s.now())
		if err != nil {
			return Schedule{}, err
		}
		updates["next_run_at"] = nextRun
	} else {
		updates["next_run_at"] = nil
	}

	if err := s.db.WithContext(ctx).Model(&database.Schedule{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return Schedule{}, err
	}
	return s.Get(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Schedule, error) {
	var records []database.Schedule
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL").Order("id asc").Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]Schedule, 0, len(records))
	for _, record := range records {
		out = append(out, projectSchedule(record, nil))
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, id uint) (Schedule, error) {
	var record database.Schedule
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&record).Error; err != nil {
		return Schedule{}, err
	}
	var runs []database.ScheduleRun
	if err := s.db.WithContext(ctx).Where("schedule_id = ?", id).Order("created_at desc").Limit(10).Find(&runs).Error; err != nil {
		return Schedule{}, err
	}
	return projectSchedule(record, runs), nil
}

func (s *Service) SetEnabled(ctx context.Context, id uint, enabled bool) (Schedule, error) {
	return s.Update(ctx, id, UpdateScheduleInput{Enabled: &enabled})
}

func (s *Service) ListHistory(ctx context.Context, scheduleID uint, limit int) ([]ScheduleRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var runs []database.ScheduleRun
	if err := s.db.WithContext(ctx).Where("schedule_id = ?", scheduleID).Order("created_at desc").Limit(limit).Find(&runs).Error; err != nil {
		return nil, err
	}
	jobsByID, err := s.jobsByRunJobID(ctx, runs)
	if err != nil {
		return nil, err
	}
	projected := make([]ScheduleRun, 0, len(runs))
	for _, run := range runs {
		projected = append(projected, projectRun(run, jobsByID))
	}
	return projected, nil
}

func (s *Service) RunNow(ctx context.Context, scheduleID uint) (RunNowResult, error) {
	var schedule database.Schedule
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", scheduleID).First(&schedule).Error; err != nil {
		return RunNowResult{}, err
	}
	payload := map[string]any{
		"schedule_id": schedule.ID,
		"kind":        schedule.Kind,
		"scope_kind":  schedule.ScopeKind,
	}
	if schedule.LibraryID != nil {
		payload["library_id"] = *schedule.LibraryID
	}
	var job database.Job
	var err error
	if s.dispatcher == nil {
		err = fmt.Errorf("schedule dispatcher unavailable")
	} else {
		job, err = s.dispatcher(ctx, DueSchedule{ID: schedule.ID, Kind: schedule.Kind, ScopeKind: ScopeKind(schedule.ScopeKind), LibraryID: schedule.LibraryID})
	}
	if err != nil {
		return RunNowResult{}, err
	}
	run, err := s.RecordRunResult(ctx, schedule.ID, RecordRunResultInput{
		Status:    StatusQueued,
		JobID:     &job.ID,
		StartedAt: s.now(),
	})
	if err != nil {
		return RunNowResult{}, err
	}
	return RunNowResult{Run: run, Job: job}, nil
}

func (s *Service) ClaimDueRuns(ctx context.Context, limit int) ([]RunNowResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	now := s.now()
	var due []database.Schedule
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL AND enabled = ? AND next_run_at IS NOT NULL AND next_run_at <= ?", true, now).
		Order("next_run_at asc, id asc").
		Limit(limit).
		Find(&due).Error; err != nil {
		return nil, err
	}
	results := make([]RunNowResult, 0, len(due))
	for _, record := range due {
		if strings.TrimSpace(record.LatestRunStatus) == StatusQueued || strings.TrimSpace(record.LatestRunStatus) == StatusRunning {
			continue
		}
		result, err := s.RunNow(ctx, record.ID)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *Service) MarkRunRunning(ctx context.Context, jobID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var run database.ScheduleRun
		if err := tx.Where("job_id = ?", jobID).Order("id desc").First(&run).Error; err != nil {
			return err
		}
		now := s.now()
		if err := tx.Model(&database.ScheduleRun{}).Where("id = ?", run.ID).Updates(map[string]any{"status": StatusRunning, "started_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(&database.Schedule{}).Where("id = ?", run.ScheduleID).Updates(map[string]any{"latest_run_status": StatusRunning, "latest_job_id": jobID, "latest_run_started_at": now}).Error
	})
}

func (s *Service) MarkRunFinished(ctx context.Context, input RunTransitionInput) error {
	if !isValidRunStatus(input.Status) {
		return fmt.Errorf("invalid run status %q", input.Status)
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var run database.ScheduleRun
		if err := tx.Where("job_id = ?", input.JobID).Order("id desc").First(&run).Error; err != nil {
			return err
		}
		finished := input.Finished
		if finished.IsZero() {
			finished = s.now()
		}
		message := strings.TrimSpace(input.Message)
		if message == StatusCompleted && strings.TrimSpace(run.ErrorSummary) != "" {
			message = strings.TrimSpace(run.ErrorSummary)
		}
		if err := tx.Model(&database.ScheduleRun{}).Where("id = ?", run.ID).Updates(map[string]any{"status": input.Status, "error_summary": message, "finished_at": finished}).Error; err != nil {
			return err
		}
		return tx.Model(&database.Schedule{}).Where("id = ?", run.ScheduleID).Updates(map[string]any{"latest_run_status": input.Status, "latest_run_message": message, "latest_job_id": input.JobID, "latest_run_finished_at": finished}).Error
	})
}

func JobKindForSchedule(scheduleKind string) string {
	return "schedule_" + normalizeKind(scheduleKind)
}

func ParseSchedulePayload(payload string) (DueSchedule, error) {
	var decoded struct {
		ScheduleID uint   `json:"schedule_id"`
		Kind       string `json:"kind"`
		ScopeKind  string `json:"scope_kind"`
		LibraryID  *uint  `json:"library_id"`
	}
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		return DueSchedule{}, err
	}
	return DueSchedule{ID: decoded.ScheduleID, Kind: decoded.Kind, ScopeKind: ScopeKind(decoded.ScopeKind), LibraryID: decoded.LibraryID}, nil
}

func (s *Service) RecordRunResult(ctx context.Context, scheduleID uint, input RecordRunResultInput) (ScheduleRun, error) {
	if !isValidRunStatus(input.Status) {
		return ScheduleRun{}, fmt.Errorf("invalid run status %q", input.Status)
	}
	var run database.ScheduleRun
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record database.Schedule
		if err := tx.Where("id = ? AND deleted_at IS NULL", scheduleID).First(&record).Error; err != nil {
			return err
		}
		run = database.ScheduleRun{
			ScheduleID:   scheduleID,
			Status:       input.Status,
			JobID:        input.JobID,
			ErrorSummary: strings.TrimSpace(input.ErrorSummary),
			StartedAt:    timePtr(input.StartedAt),
			FinishedAt:   timePtr(input.FinishedAt),
		}
		if err := tx.Create(&run).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"latest_run_status":      input.Status,
			"latest_run_message":     strings.TrimSpace(input.ErrorSummary),
			"latest_job_id":          input.JobID,
			"latest_run_started_at":  timePtr(input.StartedAt),
			"latest_run_finished_at": timePtr(input.FinishedAt),
		}
		if record.Enabled {
			nextRun, err := FrequencySpec{Kind: FrequencyKind(record.FrequencyKind), TimeOfDay: record.TimeOfDay, Weekday: record.Weekday, DayOfMonth: record.DayOfMonth}.NextRunFrom(maxTime(input.FinishedAt, s.now()))
			if err != nil {
				return err
			}
			updates["next_run_at"] = nextRun
		} else {
			updates["next_run_at"] = nil
		}
		return tx.Model(&database.Schedule{}).Where("id = ?", scheduleID).Updates(updates).Error
	})
	if err != nil {
		return ScheduleRun{}, err
	}
	return projectRun(run, nil), nil
}

func validateCreateInput(input CreateScheduleInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if !isValidKind(input.Kind) {
		return fmt.Errorf("unsupported schedule kind %q", input.Kind)
	}
	switch input.ScopeKind {
	case ScopeGlobal:
		if input.LibraryID != nil {
			return fmt.Errorf("global scope cannot include library_id")
		}
	case ScopeLibrary:
		if input.LibraryID == nil || *input.LibraryID == 0 {
			return fmt.Errorf("library scope requires library_id")
		}
	default:
		return fmt.Errorf("unsupported scope kind %q", input.ScopeKind)
	}
	return input.Frequency.Validate()
}

func isValidKind(kind string) bool {
	switch normalizeKind(kind) {
	case KindScan, KindLibraryCleanup, KindInvalidLinkCheck:
		return true
	default:
		return false
	}
}

func normalizeKind(kind string) string {
	return strings.TrimSpace(strings.ToLower(kind))
}

func isValidRunStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case StatusQueued, StatusRunning, StatusCompleted, StatusFailed, "cancel_requested", "cancelled":
		return true
	default:
		return false
	}
}

func projectSchedule(record database.Schedule, runs []database.ScheduleRun) Schedule {
	projected := Schedule{
		ID:                  record.ID,
		Name:                record.Name,
		Kind:                record.Kind,
		ScopeKind:           ScopeKind(record.ScopeKind),
		LibraryID:           record.LibraryID,
		Frequency:           FrequencySpec{Kind: FrequencyKind(record.FrequencyKind), TimeOfDay: record.TimeOfDay, Weekday: record.Weekday, DayOfMonth: record.DayOfMonth},
		Enabled:             record.Enabled,
		NextRunAt:           record.NextRunAt,
		LatestRunStatus:     record.LatestRunStatus,
		LatestRunMessage:    record.LatestRunMessage,
		LatestJobID:         record.LatestJobID,
		LatestRunStartedAt:  record.LatestRunStartedAt,
		LatestRunFinishedAt: record.LatestRunFinishedAt,
		CreatedAt:           record.CreatedAt,
		UpdatedAt:           record.UpdatedAt,
	}
	if len(runs) > 0 {
		projected.RecentRuns = make([]ScheduleRun, 0, len(runs))
		for _, run := range runs {
			projected.RecentRuns = append(projected.RecentRuns, projectRun(run, nil))
		}
	}
	return projected
}

func projectRun(record database.ScheduleRun, jobsByID map[uint]database.Job) ScheduleRun {
	projected := ScheduleRun{ID: record.ID, ScheduleID: record.ScheduleID, Status: record.Status, JobID: record.JobID, ErrorSummary: record.ErrorSummary, StartedAt: record.StartedAt, FinishedAt: record.FinishedAt, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
	if record.JobID != nil && jobsByID != nil {
		if job, ok := jobsByID[*record.JobID]; ok {
			projected.Job = projectJobDetail(job)
		}
	}
	return projected
}

func (s *Service) jobsByRunJobID(ctx context.Context, runs []database.ScheduleRun) (map[uint]database.Job, error) {
	ids := make([]uint, 0, len(runs))
	seen := make(map[uint]struct{})
	for _, run := range runs {
		if run.JobID == nil {
			continue
		}
		if _, ok := seen[*run.JobID]; ok {
			continue
		}
		seen[*run.JobID] = struct{}{}
		ids = append(ids, *run.JobID)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	var records []database.Job
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&records).Error; err != nil {
		return nil, err
	}
	jobsByID := make(map[uint]database.Job, len(records))
	for _, record := range records {
		jobsByID[record.ID] = record
	}
	return jobsByID, nil
}

func projectJobDetail(record database.Job) *JobDetail {
	return &JobDetail{ID: record.ID, JobKey: record.JobKey, Kind: record.Kind, Status: record.Status, PayloadJSON: record.PayloadJSON, ErrorMessage: record.ErrorMessage, Attempts: record.Attempts, AvailableAt: record.AvailableAt, StartedAt: record.StartedAt, FinishedAt: record.FinishedAt, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	v := value.UTC()
	return &v
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
