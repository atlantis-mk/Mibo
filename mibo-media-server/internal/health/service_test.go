package health

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/workflow"
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
	completed := database.WorkflowRun{RunKey: "test-success", LibraryID: libraryRecord.ID, Reason: library.WorkflowReasonManualScan, Status: workflow.RunStatusCompleted, ScopeKey: fmt.Sprintf("library:%d", libraryRecord.ID), CreatedAt: successAt, UpdatedAt: successAt, FinishedAt: &successAt}
	if err := db.Create(&completed).Error; err != nil {
		t.Fatalf("create completed workflow: %v", err)
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
	completed := database.SystemSetting{Category: healthSettingsCategory, Key: fmt.Sprintf("media_source_%d_validated_at", source.ID), Value: successAt.Format(time.RFC3339Nano)}
	if err := db.Create(&completed).Error; err != nil {
		t.Fatalf("create completed validation setting: %v", err)
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

func TestListIssuesIncludesIngestConditionFailures(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	itemID := uint(9)
	condition := database.IngestCondition{UnitKey: "metadata_item:9", LibraryID: 1, MetadataItemID: &itemID, ConditionType: ingest.ConditionMetadataMatched, Status: ingest.ConditionStatusReviewRequired, Reason: "no_candidate", Message: "Metadata match needed", Severity: ingest.SeverityWarning}
	if err := svc.db.WithContext(ctx).Create(&condition).Error; err != nil {
		t.Fatalf("create ingest condition: %v", err)
	}

	issues, err := svc.ListIssues(ctx)
	if err != nil {
		t.Fatalf("list issues: %v", err)
	}
	if len(issues) != 1 || issues[0].ReasonCode != ReasonIngestReviewRequired || issues[0].Scope != ScopeIngest {
		t.Fatalf("unexpected ingest issue: %#v", issues)
	}
}

func TestIngestEventRetentionRemovesExpiredEvents(t *testing.T) {
	ctx := context.Background()
	svc := ingest.NewService(newTestService(t).db)
	now := time.Now().UTC()
	expired := now.Add(-time.Hour)
	kept := now.Add(time.Hour)
	if _, err := svc.AppendEvent(ctx, database.IngestEvent{UnitKey: "inventory_file:1", LibraryID: 1, EventType: ingest.EventConditionChanged, ExpiresAt: &expired}); err != nil {
		t.Fatalf("append expired event: %v", err)
	}
	if _, err := svc.AppendEvent(ctx, database.IngestEvent{UnitKey: "inventory_file:2", LibraryID: 1, EventType: ingest.EventConditionChanged, ExpiresAt: &kept}); err != nil {
		t.Fatalf("append kept event: %v", err)
	}
	removed, err := svc.RunEventRetention(ctx, now)
	if err != nil {
		t.Fatalf("run retention: %v", err)
	}
	if removed != 1 {
		t.Fatalf("expected one removed event, got %d", removed)
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

func TestIgnoreIssueHidesIngestConditionIssue(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	db := svc.db
	condition := database.IngestCondition{UnitKey: "inventory_file:1", LibraryID: 1, ConditionType: ingest.ConditionProbed, Status: ingest.ConditionStatusFailed, Reason: "probe_failed", Message: "probe failed", Severity: ingest.SeverityError}
	if err := db.WithContext(ctx).Create(&condition).Error; err != nil {
		t.Fatalf("create ingest condition: %v", err)
	}
	issues, err := svc.ListIssues(ctx)
	if err != nil {
		t.Fatalf("list issues: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected one ingest issue before ignore, got %#v", issues)
	}
	if _, err := svc.IgnoreIssue(ctx, issues[0].ID); err != nil {
		t.Fatalf("ignore issue: %v", err)
	}
	issues, err = svc.ListIssues(ctx)
	if err != nil {
		t.Fatalf("list issues after ignore: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected ingest issue hidden after ignore, got %#v", issues)
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	cfg := config.Config{Database: config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")}}
	db, err := database.Open(cfg.Database)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	registry := providers.NewRegistry(cfg)
	librarySvc := library.NewService(cfg, db, registry, nil)
	return NewService(db, registry, librarySvc, "http://127.0.0.1:5244")
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

func createFailedJob(t *testing.T, db *gorm.DB, payloadJSON string, errorMessage string) database.WorkflowTask {
	t.Helper()
	now := time.Now().UTC()
	run := database.WorkflowRun{RunKey: fmt.Sprintf("test-run-%d", now.UnixNano()), LibraryID: libraryIDFromPayload(payloadJSON), Reason: library.WorkflowReasonTargetedRefresh, Status: workflow.RunStatusFailed, ScopeKey: "test", CreatedAt: now, UpdatedAt: now, FinishedAt: &now}
	if run.LibraryID == 0 {
		run.LibraryID = 1
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatalf("create failed workflow run: %v", err)
	}
	task := database.WorkflowTask{RunID: run.ID, LibraryID: run.LibraryID, TaskKey: fmt.Sprintf("test-task-%d", now.UnixNano()), TaskType: workflow.TaskTypeScanLibraryPath, Stage: workflow.StageScan, Status: workflow.TaskStatusFailed, ScopeKey: run.ScopeKey, PayloadJSON: payloadJSON, ErrorMessage: errorMessage, Attempts: 1, AvailableAt: now, CreatedAt: now, UpdatedAt: now, FinishedAt: &now}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create failed workflow task: %v", err)
	}
	return task
}

func libraryIDFromPayload(payloadJSON string) uint {
	var payload map[string]any
	_ = json.Unmarshal([]byte(payloadJSON), &payload)
	if value, ok := payload["library_id"].(float64); ok && value > 0 {
		return uint(value)
	}
	return 0
}

func uintString(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}
