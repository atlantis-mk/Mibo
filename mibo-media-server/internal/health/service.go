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
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const failedJobLookbackLimit = 200

const JobKindValidateMediaSource = "validate_media_source"

const healthSettingsCategory = "health"
const ignoredIssueIDsKey = "ignored_issue_ids"

type Service struct {
	db       *gorm.DB
	storage  *providers.Registry
	library  *library.Service
	jobs     *jobs.Service
	adminURL string
}

func NewService(db *gorm.DB, registry *providers.Registry, librarySvc *library.Service, jobsSvc *jobs.Service, adminURL string) *Service {
	return &Service{db: db, storage: registry, library: librarySvc, jobs: jobsSvc, adminURL: strings.TrimRight(strings.TrimSpace(adminURL), "/")}
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
	var jobRows []database.Job
	if err := s.db.WithContext(ctx).
		Where("status = ? AND error_message <> ''", jobs.StatusFailed).
		Order("updated_at desc").
		Limit(failedJobLookbackLimit).
		Find(&jobRows).Error; err != nil {
		return nil, err
	}

	groups := map[string]*issueBuilder{}
	for _, job := range jobRows {
		issue, key, err := s.issueForFailedJob(ctx, job)
		if err != nil {
			return nil, err
		}
		if issue == nil {
			continue
		}
		if _, ok := ignored[issue.ID]; ok {
			continue
		}
		resolved, err := s.issueResolvedAfterFailure(ctx, job, issue)
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
		builder.addJob(job)
	}

	issues := make([]Issue, 0, len(groups))
	for _, builder := range groups {
		builder.finish()
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
	now := time.Now().UTC()
	job := database.Job{
		Kind:        JobKindValidateMediaSource,
		Status:      jobs.StatusCompleted,
		PayloadJSON: fmt.Sprintf(`{"media_source_id":%d}`, mediaSourceID),
		Attempts:    1,
		AvailableAt: now,
		StartedAt:   &now,
		FinishedAt:  &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return s.db.WithContext(ctx).Create(&job).Error
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
		job, err := s.library.QueueLibraryScan(ctx, ref.ID)
		if err != nil {
			return RescanResult{}, err
		}
		result.Jobs = append(result.Jobs, jobRef(job))
	}
	return result, nil
}

func (s *Service) issueForFailedJob(ctx context.Context, job database.Job) (*Issue, string, error) {
	reason, severity := classifyJobFailure(job)
	refs, err := s.affectedForJob(ctx, job)
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
			JobKind:      job.Kind,
			JobStatus:    job.Status,
			PayloadJSON:  job.PayloadJSON,
			ErrorMessage: job.ErrorMessage,
		},
	}
	issue.Title, issue.Message = copyForReason(reason, refs)
	issue.Actions = s.actionsForIssue(reason, refs, job)
	if err := s.addAffectedCounts(ctx, &issue); err != nil {
		return nil, "", err
	}
	key := groupingKey(reason, refs, job)
	issue.ID = issueID(key)
	return &issue, key, nil
}

func classifyJobFailure(job database.Job) (string, string) {
	message := strings.ToLower(job.ErrorMessage)
	if strings.Contains(message, "captcha_invalid") || strings.Contains(message, "captcha_token expired") {
		return ReasonStorageAuthExpired, SeverityBlocking
	}
	return ReasonJobFailedUnknown, SeverityError
}

func (s *Service) affectedForJob(ctx context.Context, job database.Job) (Affected, error) {
	var payload map[string]any
	_ = json.Unmarshal([]byte(job.PayloadJSON), &payload)
	libraryID := uintFromPayload(payload, "library_id")
	fileID := uintFromPayload(payload, "inventory_file_id")
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
	refs := Affected{Jobs: []JobRef{jobRef(job)}}
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
	if err := s.db.WithContext(ctx).Model(&database.CatalogItem{}).Where("library_id IN ? AND deleted_at IS NULL", libraryIDs).Count(&issue.Impact.AffectedCatalogItems).Error; err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&database.InventoryFile{}).Where("library_id IN ?", libraryIDs).Count(&issue.Impact.AffectedFiles).Error
}

func (s *Service) issueResolvedAfterFailure(ctx context.Context, failedJob database.Job, issue *Issue) (bool, error) {
	if issue.ReasonCode != ReasonStorageAuthExpired || len(issue.Affected.Libraries) == 0 {
		return false, nil
	}
	if len(issue.Affected.MediaSources) > 0 {
		validatedSources := 0
		for _, sourceRef := range issue.Affected.MediaSources {
			var count int64
			if err := s.db.WithContext(ctx).Model(&database.Job{}).
				Where("status = ?", jobs.StatusCompleted).
				Where("kind = ?", JobKindValidateMediaSource).
				Where("updated_at > ?", failedJob.UpdatedAt).
				Where("payload_json LIKE ?", `%"media_source_id":`+fmt.Sprint(sourceRef.ID)+`%`).
				Count(&count).Error; err != nil {
				return false, err
			}
			if count == 0 {
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
		if err := s.db.WithContext(ctx).Model(&database.Job{}).
			Where("status = ?", jobs.StatusCompleted).
			Where("kind IN ?", []string{library.JobKindSyncLibrary, library.JobKindTargetedRefresh}).
			Where("updated_at > ?", failedJob.UpdatedAt).
			Where("payload_json LIKE ?", `%"library_id":`+fmt.Sprint(libraryRef.ID)+`%`).
			Count(&count).Error; err != nil {
			return false, err
		}
		if count == 0 {
			return false, nil
		}
	}
	return true, nil
}

func (s *Service) actionsForIssue(reason string, refs Affected, job database.Job) []Action {
	actions := []Action{{Type: ActionViewJob, Label: "查看任务", Description: "查看失败任务的原始错误。", JobID: job.ID}, {Type: ActionIgnoreIssue, Label: "忽略", Description: "不再显示这个健康问题。"}}
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

func (b *issueBuilder) addJob(job database.Job) {
	ref := jobRef(job)
	if !containsJob(b.issue.Affected.Jobs, ref.ID) {
		b.issue.Affected.Jobs = append(b.issue.Affected.Jobs, ref)
	}
	if b.issue.FirstSeenAt == nil || job.CreatedAt.Before(*b.issue.FirstSeenAt) {
		created := job.CreatedAt
		b.issue.FirstSeenAt = &created
	}
	if b.issue.LastSeenAt == nil || job.UpdatedAt.After(*b.issue.LastSeenAt) {
		updated := job.UpdatedAt
		b.issue.LastSeenAt = &updated
		b.issue.LatestJobID = job.ID
		b.issue.TechnicalDetail.JobKind = job.Kind
		b.issue.TechnicalDetail.JobStatus = job.Status
		b.issue.TechnicalDetail.PayloadJSON = job.PayloadJSON
		b.issue.TechnicalDetail.ErrorMessage = job.ErrorMessage
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

func groupingKey(reason string, refs Affected, job database.Job) string {
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
		parts = append(parts, fmt.Sprintf("job:%s", job.Kind))
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

func containsJob(jobs []JobRef, id uint) bool {
	for _, job := range jobs {
		if job.ID == id {
			return true
		}
	}
	return false
}
