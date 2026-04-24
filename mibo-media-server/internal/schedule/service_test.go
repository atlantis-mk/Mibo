package schedule

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func newTestService(t *testing.T, now time.Time) (*Service, context.Context, *database.Schedule) {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db, WithNow(func() time.Time { return now }))
	return svc, ctx, nil

}

func TestDailyFrequencyComputesNextRun(t *testing.T) {
	now := time.Date(2026, 4, 24, 10, 30, 0, 0, time.UTC)
	svc, ctx, _ := newTestService(t, now)

	schedule, err := svc.Create(ctx, CreateScheduleInput{
		Name:      "Daily scan",
		Kind:      KindScan,
		ScopeKind: ScopeGlobal,
		Enabled:   true,
		Frequency: FrequencySpec{Kind: FrequencyDaily, TimeOfDay: "15:45"},
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	if schedule.NextRunAt == nil {
		t.Fatalf("expected next run time")
	}
	expected := time.Date(2026, 4, 24, 15, 45, 0, 0, time.UTC)
	if !schedule.NextRunAt.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected, schedule.NextRunAt)
	}

	stored, err := svc.Get(ctx, schedule.ID)
	if err != nil {
		t.Fatalf("reload schedule: %v", err)
	}
	if stored.NextRunAt == nil || !stored.NextRunAt.Equal(expected) {
		t.Fatalf("expected stored next run %s, got %#v", expected, stored.NextRunAt)
	}
	if !stored.Enabled {
		t.Fatalf("expected schedule enabled")
	}
	if stored.LatestRunStatus != "" {
		t.Fatalf("expected empty latest status, got %q", stored.LatestRunStatus)
	}
	if stored.LatestJobID != nil {
		t.Fatalf("expected nil latest job id, got %v", *stored.LatestJobID)
	}
}

func TestValidateScheduleRejectsInvalidCombinations(t *testing.T) {
	now := time.Date(2026, 4, 24, 10, 30, 0, 0, time.UTC)
	svc, ctx, _ := newTestService(t, now)

	cases := []CreateScheduleInput{
		{Name: "Weekly", Kind: KindScan, ScopeKind: ScopeGlobal, Enabled: true, Frequency: FrequencySpec{Kind: FrequencyWeekly, TimeOfDay: "09:30"}},
		{Name: "Monthly", Kind: KindScan, ScopeKind: ScopeGlobal, Enabled: true, Frequency: FrequencySpec{Kind: FrequencyMonthly, TimeOfDay: "09:30"}},
		{Name: "Library missing id", Kind: KindScan, ScopeKind: ScopeLibrary, Enabled: true, Frequency: FrequencySpec{Kind: FrequencyDaily, TimeOfDay: "09:30"}},
		{Name: "Global with id", Kind: KindScan, ScopeKind: ScopeGlobal, LibraryID: uintPtr(7), Enabled: true, Frequency: FrequencySpec{Kind: FrequencyDaily, TimeOfDay: "09:30"}},
		{Name: "Bad kind", Kind: "delete_everything", ScopeKind: ScopeGlobal, Enabled: true, Frequency: FrequencySpec{Kind: FrequencyDaily, TimeOfDay: "09:30"}},
	}

	for _, input := range cases {
		if _, err := svc.Create(ctx, input); err == nil {
			t.Fatalf("expected validation error for %#v", input)
		}
	}
}

func TestListGetAndHistoryProjection(t *testing.T) {
	now := time.Date(2026, 4, 24, 10, 30, 0, 0, time.UTC)
	svc, ctx, _ := newTestService(t, now)

	schedule, err := svc.Create(ctx, CreateScheduleInput{
		Name:      "Trailer sync",
		Kind:      KindTrailerSync,
		ScopeKind: ScopeLibrary,
		LibraryID: uintPtr(42),
		Enabled:   true,
		Frequency: FrequencySpec{Kind: FrequencyWeekly, TimeOfDay: "11:00", Weekday: intPtr(int(time.Monday))},
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	run, err := svc.RecordRunResult(ctx, schedule.ID, RecordRunResultInput{
		Status:       StatusCompleted,
		JobID:        uintPtr(88),
		ErrorSummary: "",
		StartedAt:    now.Add(-5 * time.Minute),
		FinishedAt:   now.Add(-1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("record run result: %v", err)
	}

	if run.ScheduleID != schedule.ID {
		t.Fatalf("expected run to belong to schedule %d, got %d", schedule.ID, run.ScheduleID)
	}

	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("list schedules: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(list))
	}
	if list[0].LatestRunStatus != StatusCompleted {
		t.Fatalf("expected latest status %q, got %q", StatusCompleted, list[0].LatestRunStatus)
	}
	if list[0].LatestJobID == nil || *list[0].LatestJobID != 88 {
		t.Fatalf("expected latest job id 88, got %#v", list[0].LatestJobID)
	}

	detail, err := svc.Get(ctx, schedule.ID)
	if err != nil {
		t.Fatalf("get schedule: %v", err)
	}
	if len(detail.RecentRuns) != 1 {
		t.Fatalf("expected 1 recent run, got %d", len(detail.RecentRuns))
	}

	updated, err := svc.Update(ctx, schedule.ID, UpdateScheduleInput{Enabled: boolPtr(false)})
	if err != nil {
		t.Fatalf("disable schedule: %v", err)
	}
	if updated.Enabled {
		t.Fatalf("expected disabled schedule")
	}
	if updated.NextRunAt != nil {
		t.Fatalf("expected disabled schedule next run cleared, got %v", updated.NextRunAt)
	}
	if len(updated.RecentRuns) != 1 {
		t.Fatalf("expected history retained after disable")
	}
}

func TestScheduleTablesAreMigrated(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	if !db.Migrator().HasTable(&database.Schedule{}) {
		t.Fatalf("expected schedules table to exist")
	}
	if !db.Migrator().HasTable(&database.ScheduleRun{}) {
		t.Fatalf("expected schedule_runs table to exist")
	}
}

func uintPtr(v uint) *uint { return &v }

func intPtr(v int) *int { return &v }

func boolPtr(v bool) *bool { return &v }
