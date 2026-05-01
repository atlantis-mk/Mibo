package health

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/providers"
	"gorm.io/gorm"
)

func TestSummaryHealthyWithoutFailedJobs(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	summary, err := svc.Summary(ctx)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.Status != OverallStatusHealthy || summary.IssueCount != 0 || len(summary.Issues) != 0 {
		t.Fatalf("unexpected healthy summary: %#v", summary)
	}
}

func TestListIssuesClassifiesStorageAuthExpired(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	db := svc.db
	source := createHealthTestSource(t, db)
	libraryRecord := createHealthTestLibrary(t, db, source.ID)
	createFailedJob(t, db, `{"library_id":`+uintString(libraryRecord.ID)+`}`, "list directory /My Pack/电影: openlist request failed: ErrorCode: 4002 ,Error: captcha_invalid ,ErrorDescription: captcha_token expired")

	issues, err := svc.ListIssues(ctx)
	if err != nil {
		t.Fatalf("list issues: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected one issue, got %d: %#v", len(issues), issues)
	}
	issue := issues[0]
	if issue.ReasonCode != ReasonStorageAuthExpired || issue.Severity != SeverityBlocking {
		t.Fatalf("unexpected issue classification: %#v", issue)
	}
	if !issue.Impact.BlocksScan || !issue.Impact.BlocksHomeVisibility {
		t.Fatalf("expected scan and home visibility impact: %#v", issue.Impact)
	}
	if len(issue.Affected.MediaSources) != 1 || issue.Affected.MediaSources[0].ID != source.ID {
		t.Fatalf("unexpected affected sources: %#v", issue.Affected.MediaSources)
	}
	if issue.Affected.MediaSources[0].AdminURL != "http://openlist.example.test" {
		t.Fatalf("expected source-specific OpenList URL, got %#v", issue.Affected.MediaSources[0])
	}
	if issue.Actions[0].Type != ActionOpenExternalAdmin || issue.Actions[0].Href != "http://openlist.example.test" {
		t.Fatalf("expected OpenList action to use source URL, got %#v", issue.Actions)
	}
	if len(issue.Affected.Libraries) != 1 || issue.Affected.Libraries[0].ID != libraryRecord.ID {
		t.Fatalf("unexpected affected libraries: %#v", issue.Affected.Libraries)
	}
	if !strings.Contains(issue.TechnicalDetail.ErrorMessage, "captcha_invalid") {
		t.Fatalf("expected raw technical detail, got %#v", issue.TechnicalDetail)
	}
	if len(issue.Actions) == 0 {
		t.Fatalf("expected recovery actions")
	}
}

func TestListIssuesGroupsRepeatedFailures(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	db := svc.db
	source := createHealthTestSource(t, db)
	libraryRecord := createHealthTestLibrary(t, db, source.ID)
	createFailedJob(t, db, `{"library_id":`+uintString(libraryRecord.ID)+`}`, "captcha_token expired")
	createFailedJob(t, db, `{"library_id":`+uintString(libraryRecord.ID)+`}`, "captcha_invalid")

	issues, err := svc.ListIssues(ctx)
	if err != nil {
		t.Fatalf("list issues: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected grouped issue, got %d", len(issues))
	}
	if len(issues[0].Affected.Jobs) != 2 {
		t.Fatalf("expected two grouped jobs, got %#v", issues[0].Affected.Jobs)
	}
}

func TestListIssuesHidesStorageAuthFailureAfterSuccessfulRescan(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	db := svc.db
	source := createHealthTestSource(t, db)
	libraryRecord := createHealthTestLibrary(t, db, source.ID)
	failed := createFailedJob(t, db, `{"library_id":`+uintString(libraryRecord.ID)+`}`, "captcha_token expired")
	successAt := failed.UpdatedAt.Add(time.Minute)
	completed := database.Job{Kind: library.JobKindSyncLibrary, Status: jobs.StatusCompleted, PayloadJSON: `{"library_id":` + uintString(libraryRecord.ID) + `}`, Attempts: 1, AvailableAt: successAt, CreatedAt: successAt, UpdatedAt: successAt, FinishedAt: &successAt}
	if err := db.Create(&completed).Error; err != nil {
		t.Fatalf("create completed job: %v", err)
	}

	issues, err := svc.ListIssues(ctx)
	if err != nil {
		t.Fatalf("list issues: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected resolved issue to be hidden, got %#v", issues)
	}
}

func TestListIssuesHidesStorageAuthFailureAfterSuccessfulValidation(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	db := svc.db
	source := createHealthTestSource(t, db)
	libraryRecord := createHealthTestLibrary(t, db, source.ID)
	failed := createFailedJob(t, db, `{"library_id":`+uintString(libraryRecord.ID)+`}`, "captcha_token expired")
	successAt := failed.UpdatedAt.Add(time.Minute)
	completed := database.Job{Kind: JobKindValidateMediaSource, Status: jobs.StatusCompleted, PayloadJSON: `{"media_source_id":` + uintString(source.ID) + `}`, Attempts: 1, AvailableAt: successAt, CreatedAt: successAt, UpdatedAt: successAt, FinishedAt: &successAt}
	if err := db.Create(&completed).Error; err != nil {
		t.Fatalf("create completed validation: %v", err)
	}

	issues, err := svc.ListIssues(ctx)
	if err != nil {
		t.Fatalf("list issues: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected validated issue to be hidden, got %#v", issues)
	}
}

func TestListIssuesFallsBackForUnknownFailure(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	db := svc.db
	createFailedJob(t, db, `{}`, "unexpected worker failure")

	issues, err := svc.ListIssues(ctx)
	if err != nil {
		t.Fatalf("list issues: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected one issue, got %d", len(issues))
	}
	if issues[0].ReasonCode != ReasonJobFailedUnknown || issues[0].Severity != SeverityError {
		t.Fatalf("unexpected fallback issue: %#v", issues[0])
	}
}

func TestIgnoreIssueHidesIssue(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	db := svc.db
	source := createHealthTestSource(t, db)
	libraryRecord := createHealthTestLibrary(t, db, source.ID)
	createFailedJob(t, db, `{"library_id":`+uintString(libraryRecord.ID)+`}`, "captcha_token expired")
	issues, err := svc.ListIssues(ctx)
	if err != nil {
		t.Fatalf("list issues: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected one issue before ignore, got %#v", issues)
	}
	if _, err := svc.IgnoreIssue(ctx, issues[0].ID); err != nil {
		t.Fatalf("ignore issue: %v", err)
	}
	issues, err = svc.ListIssues(ctx)
	if err != nil {
		t.Fatalf("list issues after ignore: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected issue hidden after ignore, got %#v", issues)
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	cfg := config.Config{Database: config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")}}
	db, err := database.Open(cfg.Database)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	jobsSvc := jobs.NewService(db)
	registry := providers.NewRegistry(cfg)
	librarySvc := library.NewService(cfg, db, registry, jobsSvc)
	return NewService(db, registry, librarySvc, jobsSvc, "http://127.0.0.1:5244")
}

func createHealthTestSource(t *testing.T, db *gorm.DB) database.MediaSource {
	t.Helper()
	configJSON, err := json.Marshal(providers.SourceConfig{OpenList: &providers.OpenListSourceConfig{BaseURL: "http://openlist.example.test", Timeout: "15s"}})
	if err != nil {
		t.Fatalf("marshal source config: %v", err)
	}
	source := database.MediaSource{Name: "PikPak", Provider: "openlist", StorageRef: "/", RootPath: "/", ConfigJSON: string(configJSON)}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	return source
}

func createHealthTestLibrary(t *testing.T, db *gorm.DB, mediaSourceID uint) database.Library {
	t.Helper()
	record := database.Library{Name: "电影", Type: "movies", MediaSourceID: mediaSourceID, RootPath: "/My Pack/电影", Status: "error", ScannerEnabled: true}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	return record
}

func createFailedJob(t *testing.T, db *gorm.DB, payloadJSON string, errorMessage string) database.Job {
	t.Helper()
	now := time.Now().UTC()
	job := database.Job{Kind: "targeted_refresh", Status: jobs.StatusFailed, PayloadJSON: payloadJSON, ErrorMessage: errorMessage, Attempts: 1, AvailableAt: now, CreatedAt: now, UpdatedAt: now, FinishedAt: &now}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create failed job: %v", err)
	}
	return job
}

func uintString(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}
