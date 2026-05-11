package health

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/storage"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const failedWorkflowLookbackLimit = 200

const healthSettingsCategory = "health"
const ignoredIssueIDsKey = "ignored_issue_ids"

type Service struct {
	db       *gorm.DB
	storage  *providers.Registry
	library  *library.Service
	adminURL string
}

func NewService(db *gorm.DB, registry *providers.Registry, librarySvc *library.Service, adminURL string) *Service {
	return &Service{db: db, storage: registry, library: librarySvc, adminURL: strings.TrimRight(strings.TrimSpace(adminURL), "/")}
}

func (s *Service) Summary(ctx context.Context) (Summary, error) {
	issues, err := s.ListIssues(ctx)
	if err != nil {
		return Summary{}, err
	}
	result := Summary{Status: OverallStatusHealthy, IssueCount: len(issues), Issues: issues}
	for _, issue := range issues {
		switch issue.Severity {
		case SeverityBlocking:
			result.BlockingCount++
		case SeverityError:
			result.ErrorCount++
		case SeverityWarning:
			result.WarningCount++
		}
	}
	switch {
	case result.BlockingCount > 0:
		result.Status = OverallStatusBlocking
	case result.ErrorCount > 0:
		result.Status = OverallStatusError
	case result.WarningCount > 0:
		result.Status = OverallStatusWarning
	}
	return result, nil
}

func (s *Service) ListIssues(ctx context.Context) ([]Issue, error) {
	ignored, err := s.ignoredIssueIDs(ctx)
	if err != nil {
		return nil, err
	}
	var failedTasks []database.WorkflowTask
	if err := s.db.WithContext(ctx).
		Where("status = ? AND error_message <> ''", workflow.TaskStatusFailed).
		Order("updated_at desc").
		Limit(failedWorkflowLookbackLimit).
		Find(&failedTasks).Error; err != nil {
		return nil, err
	}

	groups := map[string]*issueBuilder{}
	if err := s.addIngestConditionIssues(ctx, groups); err != nil {
		return nil, err
	}
	for _, task := range failedTasks {
		issue, key, err := s.issueForFailedWorkflowTask(ctx, task)
		if err != nil {
			return nil, err
		}
		if issue == nil {
			continue
		}
		if _, ok := ignored[issue.ID]; ok {
			continue
		}
		resolved, err := s.issueResolvedAfterFailure(ctx, task, issue)
		if err != nil {
			return nil, err
		}
		if resolved {
			continue
		}
		builder := groups[key]
		if builder == nil {
			copyIssue := *issue
			builder = &issueBuilder{issue: copyIssue}
			groups[key] = builder
		}
		builder.addWorkflowTask(task)
	}

	issues := make([]Issue, 0, len(groups))
	for _, builder := range groups {
		builder.finish()
		if _, ok := ignored[builder.issue.ID]; ok {
			continue
		}
		issues = append(issues, builder.issue)
	}
	sort.SliceStable(issues, func(i, j int) bool {
		left := severityRank(issues[i].Severity)
		right := severityRank(issues[j].Severity)
		if left != right {
			return left < right
		}
		return issues[i].ID < issues[j].ID
	})
	return issues, nil
}

func (s *Service) addIngestConditionIssues(ctx context.Context, groups map[string]*issueBuilder) error {
	var conditions []database.IngestCondition
	if err := s.db.WithContext(ctx).
		Where("status IN ? OR stale_after <= ?", []string{ingest.ConditionStatusFailed, ingest.ConditionStatusReviewRequired}, time.Now().UTC()).
		Order("updated_at desc").
		Limit(100).
		Find(&conditions).Error; err != nil {
		return err
	}
	for _, condition := range conditions {
		issue := ingestConditionIssue(condition)
		if issue == nil {
			continue
		}
		key := issue.ReasonCode + ":" + condition.ConditionType
		builder := groups[key]
		if builder == nil {
			builder = &issueBuilder{issue: *issue}
			groups[key] = builder
		}
		builder.issue.Impact.AffectedFiles++
		if condition.MetadataItemID != nil {
			builder.issue.Impact.AffectedMetadataItems++
		}
	}
	return nil
}

func ingestConditionIssue(condition database.IngestCondition) *Issue {
	now := time.Now().UTC()
	reason := ReasonIngestStageFailed
	title := "媒体整理阶段失败"
	severity := SeverityError
	if condition.Status == ingest.ConditionStatusReviewRequired {
		reason = ReasonIngestReviewRequired
		title = "媒体整理需要人工确认"
		severity = SeverityWarning
	} else if condition.StaleAfter != nil && !condition.StaleAfter.After(now) {
		reason = ReasonIngestStageStale
		title = "媒体整理阶段已过期"
		severity = SeverityWarning
	} else if condition.Status != ingest.ConditionStatusFailed {
		return nil
	}
	message := strings.TrimSpace(condition.Message)
	if message == "" {
		message = condition.Reason
	}
	if message == "" {
		message = condition.ConditionType
	}
	return &Issue{ID: "ingest:" + reason + ":" + condition.ConditionType, Severity: severity, ReasonCode: reason, Scope: ScopeIngest, Title: title, Message: message, Impact: Impact{BlocksHomeVisibility: condition.ConditionType == ingest.ConditionProjectionCurrent}, TechnicalDetail: TechnicalDetail{ErrorMessage: message}, FirstSeenAt: condition.LastTransitionAt, LastSeenAt: &condition.UpdatedAt}
}

func (s *Service) IgnoreIssue(ctx context.Context, issueID string) (IgnoreResult, error) {
	issueID = strings.TrimSpace(issueID)
	if issueID == "" {
		return IgnoreResult{}, fmt.Errorf("issue id is required")
	}
	ignored, err := s.ignoredIssueIDs(ctx)
	if err != nil {
		return IgnoreResult{}, err
	}
	ignored[issueID] = struct{}{}
	ids := make([]string, 0, len(ignored))
	for id := range ignored {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	body, err := json.Marshal(ids)
	if err != nil {
		return IgnoreResult{}, err
	}
	record := database.SystemSetting{Category: healthSettingsCategory, Key: ignoredIssueIDsKey, Value: string(body)}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "category"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&record).Error; err != nil {
		return IgnoreResult{}, err
	}
	return IgnoreResult{IssueID: issueID, Status: "ignored"}, nil
}

func (s *Service) ignoredIssueIDs(ctx context.Context) (map[string]struct{}, error) {
	var value string
	if err := s.db.WithContext(ctx).Model(&database.SystemSetting{}).
		Where("category = ? AND key = ?", healthSettingsCategory, ignoredIssueIDsKey).
		Select("value").
		Scan(&value).Error; err != nil {
		return nil, err
	}
	var ids []string
	if strings.TrimSpace(value) != "" {
		if err := json.Unmarshal([]byte(value), &ids); err != nil {
			return nil, err
		}
	}
	result := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			result[id] = struct{}{}
		}
	}
	return result, nil
}

func (s *Service) ValidateMediaSource(ctx context.Context, mediaSourceID uint) (ValidationResult, error) {
	var source database.MediaSource
	if err := s.db.WithContext(ctx).First(&source, mediaSourceID).Error; err != nil {
		return ValidationResult{}, err
	}
	provider, err := s.storage.BuildForSource(source)
	if err != nil {
		return ValidationResult{}, err
	}
	rootPath := strings.TrimSpace(source.RootPath)
	if rootPath == "" {
		rootPath = "/"
	}
	if _, err := provider.ResolveStorage(ctx, storage.ResolveStorageRequest{Path: rootPath}); err != nil {
		return ValidationResult{MediaSourceID: source.ID, Status: "error", Message: err.Error()}, nil
	}
	if err := s.recordSuccessfulValidation(ctx, source.ID); err != nil {
		return ValidationResult{}, err
	}
	return ValidationResult{MediaSourceID: source.ID, Status: "ok", Message: "媒体源连接验证成功。"}, nil
}

func (s *Service) recordSuccessfulValidation(ctx context.Context, mediaSourceID uint) error {
	record := database.SystemSetting{Category: healthSettingsCategory, Key: fmt.Sprintf("media_source_%d_validated_at", mediaSourceID), Value: time.Now().UTC().Format(time.RFC3339Nano)}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "category"}, {Name: "key"}}, DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"})}).Create(&record).Error
}

func (s *Service) RescanIssueLibraries(ctx context.Context, issueID string) (RescanResult, error) {
	issues, err := s.ListIssues(ctx)
	if err != nil {
		return RescanResult{}, err
	}
	var selected *Issue
	for idx := range issues {
		if issues[idx].ID == issueID {
			selected = &issues[idx]
			break
		}
	}
	if selected == nil {
		return RescanResult{}, gorm.ErrRecordNotFound
	}
	result := RescanResult{IssueID: issueID}
	for _, ref := range selected.Affected.Libraries {
		job, err := s.library.QueueLibraryScanWithReason(ctx, ref.ID, library.WorkflowReasonManualScan)
		if err != nil {
			return RescanResult{}, err
		}
		result.Jobs = append(result.Jobs, jobRef(job))
	}
	return result, nil
}

func (s *Service) issueForFailedWorkflowTask(ctx context.Context, task database.WorkflowTask) (*Issue, string, error) {
	reason, severity := classifyWorkflowFailure(task)
	refs, err := s.affectedForWorkflowTask(ctx, task)
	if err != nil {
		return nil, "", err
	}
	issue := Issue{
		Severity:   severity,
		ReasonCode: reason,
		Scope:      issueScope(refs),
		Impact:     impactForReason(reason),
		Affected:   refs,
		TechnicalDetail: TechnicalDetail{
			JobKind:      task.TaskType,
			JobStatus:    task.Status,
			PayloadJSON:  task.PayloadJSON,
			ErrorMessage: task.ErrorMessage,
		},
	}
	issue.Title, issue.Message = copyForReason(reason, refs)
	issue.Actions = s.actionsForIssue(reason, refs, task)
	if err := s.addAffectedCounts(ctx, &issue); err != nil {
		return nil, "", err
	}
	key := groupingKey(reason, refs, task)
	issue.ID = issueID(key)
	return &issue, key, nil
}

func classifyWorkflowFailure(task database.WorkflowTask) (string, string) {
	message := strings.ToLower(task.ErrorMessage)
	if strings.Contains(message, "captcha_invalid") || strings.Contains(message, "captcha_token expired") {
		return ReasonStorageAuthExpired, SeverityBlocking
	}
	return ReasonJobFailedUnknown, SeverityError
}

func (s *Service) affectedForWorkflowTask(ctx context.Context, task database.WorkflowTask) (Affected, error) {
	var payload map[string]any
	_ = json.Unmarshal([]byte(task.PayloadJSON), &payload)
	libraryID := uintFromPayload(payload, "library_id")
	fileID := uintFromPayload(payload, "inventory_file_id")
	if fileID == 0 {
		fileID = firstUintFromPayloadArray(payload, "file_ids")
	}
	if libraryID == 0 {
		libraryID = task.LibraryID
	}
	if libraryID == 0 && fileID > 0 {
		var file database.InventoryFile
		if err := s.db.WithContext(ctx).First(&file, fileID).Error; err == nil {
			libraryID = file.LibraryID
		} else if err != nil && err != gorm.ErrRecordNotFound {
			return Affected{}, err
		}
	}

	var libraries []database.Library
	if libraryID > 0 {
		if err := s.db.WithContext(ctx).Where("id = ?", libraryID).Find(&libraries).Error; err != nil {
			return Affected{}, err
		}
	}
	sourceIDs := map[uint]struct{}{}
	refs := Affected{Jobs: []JobRef{workflowTaskRef(task)}}
	for _, record := range libraries {
		refs.Libraries = append(refs.Libraries, libraryRef(record))
		if record.MediaSourceID > 0 {
			sourceIDs[record.MediaSourceID] = struct{}{}
		}
	}
	if len(sourceIDs) > 0 {
		ids := make([]uint, 0, len(sourceIDs))
		for id := range sourceIDs {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		var sources []database.MediaSource
		if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&sources).Error; err != nil {
			return Affected{}, err
		}
		for _, source := range sources {
			refs.MediaSources = append(refs.MediaSources, s.mediaSourceRef(source))
		}
	}
	return refs, nil
}

func (s *Service) addAffectedCounts(ctx context.Context, issue *Issue) error {
	if len(issue.Affected.Libraries) == 0 {
		return nil
	}
	libraryIDs := make([]uint, 0, len(issue.Affected.Libraries))
	for _, ref := range issue.Affected.Libraries {
		libraryIDs = append(libraryIDs, ref.ID)
	}
	if s.db.Migrator().HasTable(&database.LibraryMetadataProjection{}) {
		if err := s.db.WithContext(ctx).Model(&database.LibraryMetadataProjection{}).Where("library_id IN ? AND hidden = ?", libraryIDs, false).Count(&issue.Impact.AffectedMetadataItems).Error; err != nil {
			return err
		}
	}
	return s.db.WithContext(ctx).Model(&database.InventoryFile{}).Where("library_id IN ?", libraryIDs).Count(&issue.Impact.AffectedFiles).Error
}

func (s *Service) issueResolvedAfterFailure(ctx context.Context, failedTask database.WorkflowTask, issue *Issue) (bool, error) {
	if issue.ReasonCode != ReasonStorageAuthExpired || len(issue.Affected.Libraries) == 0 {
		return false, nil
	}
	if len(issue.Affected.MediaSources) > 0 {
		validatedSources := 0
		for _, sourceRef := range issue.Affected.MediaSources {
			validatedAt, err := s.validationTime(ctx, sourceRef.ID)
			if err != nil {
				return false, err
			}
			if validatedAt == nil || !validatedAt.After(failedTask.UpdatedAt) {
				continue
			}
			validatedSources++
		}
		if validatedSources == len(issue.Affected.MediaSources) {
			return true, nil
		}
	}
	for _, libraryRef := range issue.Affected.Libraries {
		var count int64
		if err := s.db.WithContext(ctx).Model(&database.WorkflowRun{}).
			Where("status = ?", workflow.RunStatusCompleted).
			Where("updated_at > ?", failedTask.UpdatedAt).
			Where("library_id = ?", libraryRef.ID).
			Count(&count).Error; err != nil {
			return false, err
		}
		if count == 0 {
			return false, nil
		}
	}
	return true, nil
}

func (s *Service) actionsForIssue(reason string, refs Affected, task database.WorkflowTask) []Action {
	actions := []Action{{Type: ActionViewJob, Label: "查看任务", Description: "查看失败 workflow task 的原始错误。", JobID: task.ID}, {Type: ActionIgnoreIssue, Label: "忽略", Description: "不再显示这个健康问题。"}}
	if reason == ReasonStorageAuthExpired {
		if len(refs.MediaSources) > 0 {
			source := refs.MediaSources[0]
			action := Action{Type: ActionOpenExternalAdmin, Label: "打开 OpenList", Description: "在 OpenList 中重新完成 PikPak 登录或验证码验证。", MediaSourceID: source.ID}
			if source.Provider == "openlist" && source.AdminURL != "" {
				action.Href = source.AdminURL
			}
			actions = append([]Action{action}, actions...)
			actions = append(actions, Action{Type: ActionValidateMediaSource, Label: "验证连接", Description: "完成外部验证后，检查媒体源是否可访问。", MediaSourceID: source.ID})
		}
		libraryIDs := make([]uint, 0, len(refs.Libraries))
		for _, ref := range refs.Libraries {
			libraryIDs = append(libraryIDs, ref.ID)
		}
		if len(libraryIDs) > 0 {
			actions = append(actions, Action{Type: ActionRescanAffectedLibrary, Label: "重新扫描受影响媒体库", Description: "验证恢复后重新扫描这些媒体库。", LibraryIDs: libraryIDs})
		}
	}
	return actions
}

type issueBuilder struct {
	issue Issue
}

func (b *issueBuilder) addWorkflowTask(task database.WorkflowTask) {
	ref := workflowTaskRef(task)
	if !containsJob(b.issue.Affected.Jobs, ref.ID) {
		b.issue.Affected.Jobs = append(b.issue.Affected.Jobs, ref)
	}
	if b.issue.FirstSeenAt == nil || task.CreatedAt.Before(*b.issue.FirstSeenAt) {
		created := task.CreatedAt
		b.issue.FirstSeenAt = &created
	}
	if b.issue.LastSeenAt == nil || task.UpdatedAt.After(*b.issue.LastSeenAt) {
		updated := task.UpdatedAt
		b.issue.LastSeenAt = &updated
		b.issue.LatestJobID = task.ID
		b.issue.TechnicalDetail.JobKind = task.TaskType
		b.issue.TechnicalDetail.JobStatus = task.Status
		b.issue.TechnicalDetail.PayloadJSON = task.PayloadJSON
		b.issue.TechnicalDetail.ErrorMessage = task.ErrorMessage
	}
}

func (b *issueBuilder) finish() {
	sort.SliceStable(b.issue.Affected.Jobs, func(i, j int) bool { return b.issue.Affected.Jobs[i].ID > b.issue.Affected.Jobs[j].ID })
}

func copyForReason(reason string, refs Affected) (string, string) {
	switch reason {
	case ReasonStorageAuthExpired:
		return "PikPak 登录验证已过期", "OpenList/PikPak 的登录或验证码验证已过期，受影响媒体库暂时无法刷新。内容没有丢失，重新完成验证后可验证连接并重新扫描。"
	default:
		if len(refs.Libraries) > 0 {
			return "媒体库任务失败", "最近的后台任务失败，相关媒体库可能需要处理。"
		}
		return "后台任务失败", "最近的后台任务失败，请查看技术详情。"
	}
}

func impactForReason(reason string) Impact {
	if reason == ReasonStorageAuthExpired {
		return Impact{BlocksScan: true, BlocksHomeVisibility: true}
	}
	return Impact{}
}

func issueScope(refs Affected) string {
	if len(refs.MediaSources) > 0 {
		return ScopeMediaSource
	}
	if len(refs.Libraries) > 0 {
		return ScopeLibrary
	}
	if len(refs.Jobs) > 0 {
		return ScopeJob
	}
	return ScopeGlobal
}

func groupingKey(reason string, refs Affected, task database.WorkflowTask) string {
	parts := []string{reason, issueScope(refs)}
	if len(refs.MediaSources) > 0 {
		parts = append(parts, fmt.Sprintf("source:%d", refs.MediaSources[0].ID))
	}
	if len(refs.Libraries) > 0 {
		ids := make([]uint, 0, len(refs.Libraries))
		for _, ref := range refs.Libraries {
			ids = append(ids, ref.ID)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for _, id := range ids {
			parts = append(parts, fmt.Sprintf("library:%d", id))
		}
	}
	if len(parts) == 2 {
		parts = append(parts, fmt.Sprintf("task:%s", task.TaskType))
	}
	return strings.Join(parts, ":")
}

func issueID(key string) string {
	sum := sha1.Sum([]byte(key))
	return "issue_" + hex.EncodeToString(sum[:])[:12]
}

func severityRank(severity string) int {
	switch severity {
	case SeverityBlocking:
		return 0
	case SeverityError:
		return 1
	case SeverityWarning:
		return 2
	default:
		return 3
	}
}

func uintFromPayload(payload map[string]any, key string) uint {
	value, ok := payload[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		if typed > 0 {
			return uint(typed)
		}
	case int:
		if typed > 0 {
			return uint(typed)
		}
	}
	return 0
}

func firstUintFromPayloadArray(payload map[string]any, key string) uint {
	value, ok := payload[key]
	if !ok {
		return 0
	}
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return 0
	}
	if typed, ok := items[0].(float64); ok && typed > 0 {
		return uint(typed)
	}
	return 0
}

func (s *Service) validationTime(ctx context.Context, mediaSourceID uint) (*time.Time, error) {
	var value string
	if err := s.db.WithContext(ctx).Model(&database.SystemSetting{}).
		Where("category = ? AND key = ?", healthSettingsCategory, fmt.Sprintf("media_source_%d_validated_at", mediaSourceID)).
		Select("value").
		Scan(&value).Error; err != nil {
		return nil, err
	}
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (s *Service) mediaSourceRef(source database.MediaSource) MediaSourceRef {
	ref := MediaSourceRef{ID: source.ID, Name: source.Name, Provider: source.Provider, RootPath: source.RootPath}
	if strings.EqualFold(source.Provider, "openlist") {
		ref.OpenListURL = s.openListURLForSource(source)
		ref.AdminURL = ref.OpenListURL
	}
	return ref
}

func (s *Service) openListURLForSource(source database.MediaSource) string {
	parsed, err := providers.ParseSourceConfig(source.ConfigJSON)
	if err == nil && parsed.OpenList != nil {
		if value := strings.TrimRight(strings.TrimSpace(parsed.OpenList.BaseURL), "/"); value != "" {
			return value
		}
	}
	return s.adminURL
}

func libraryRef(record database.Library) LibraryRef {
	return LibraryRef{ID: record.ID, Name: record.Name, Type: record.Type, Status: record.Status, MediaSourceID: record.MediaSourceID, RootPath: record.RootPath}
}

func jobRef(job database.Job) JobRef {
	return JobRef{ID: job.ID, Kind: job.Kind, Status: job.Status, Attempts: job.Attempts, CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt, FinishedAt: job.FinishedAt, PayloadJSON: job.PayloadJSON}
}

func workflowTaskRef(task database.WorkflowTask) JobRef {
	return JobRef{ID: task.ID, Kind: task.TaskType, Status: task.Status, Attempts: task.Attempts, CreatedAt: task.CreatedAt, UpdatedAt: task.UpdatedAt, FinishedAt: task.FinishedAt, PayloadJSON: task.PayloadJSON}
}

func containsJob(jobs []JobRef, id uint) bool {
	for _, job := range jobs {
		if job.ID == id {
			return true
		}
	}
	return false
}
