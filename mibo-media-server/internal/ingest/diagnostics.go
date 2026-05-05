package ingest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"gorm.io/gorm"
)

const defaultDiagnosticsLimit = 100

type DiagnosticsInput struct {
	Status string
	Limit  int
}

type DiagnosticsResult struct {
	Summary DiagnosticsSummary `json:"summary"`
	Stages  []DiagnosticStage  `json:"stages"`
}

type DiagnosticsSummary struct {
	Organizing     int `json:"organizing"`
	Failed         int `json:"failed"`
	Stale          int `json:"stale"`
	ReviewRequired int `json:"review_required"`
	RetryEligible  int `json:"retry_eligible"`
}

type DiagnosticStage struct {
	ID                  uint       `json:"id"`
	UnitKey             string     `json:"unit_key"`
	LibraryID           uint       `json:"library_id"`
	LibraryName         string     `json:"library_name,omitempty"`
	InventoryFileID     *uint      `json:"inventory_file_id,omitempty"`
	StoragePath         string     `json:"storage_path,omitempty"`
	CatalogItemID       *uint      `json:"catalog_item_id,omitempty"`
	CatalogTitle        string     `json:"catalog_title,omitempty"`
	ConditionType       string     `json:"condition_type"`
	Status              string     `json:"status"`
	Reason              string     `json:"reason,omitempty"`
	Message             string     `json:"message,omitempty"`
	Severity            string     `json:"severity,omitempty"`
	Attempts            int        `json:"attempts"`
	JobID               *uint      `json:"job_id,omitempty"`
	MetadataOperationID *uint      `json:"metadata_operation_id,omitempty"`
	ProviderInstanceID  *uint      `json:"provider_instance_id,omitempty"`
	RetryEligible       bool       `json:"retry_eligible"`
	Stale               bool       `json:"stale"`
	UpdatedAt           time.Time  `json:"updated_at"`
	LastTransitionAt    *time.Time `json:"last_transition_at,omitempty"`
}

type RetryStageResult struct {
	ConditionID uint   `json:"condition_id"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

type ResolveReviewStageResult struct {
	ConditionID uint   `json:"condition_id"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

func (s *Service) Diagnostics(ctx context.Context, input DiagnosticsInput) (DiagnosticsResult, error) {
	limit := input.Limit
	if limit <= 0 || limit > 500 {
		limit = defaultDiagnosticsLimit
	}
	query := s.db.WithContext(ctx).Where("status IN ? OR severity = ?", []string{ConditionStatusPending, ConditionStatusRunning, ConditionStatusFailed, ConditionStatusReviewRequired}, SeverityError)
	status := strings.TrimSpace(input.Status)
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	var conditions []database.IngestCondition
	if err := query.Order("updated_at desc, id desc").Limit(limit).Find(&conditions).Error; err != nil {
		return DiagnosticsResult{}, err
	}
	return s.buildDiagnosticsResult(ctx, conditions)
}

func (s *Service) RetryStage(ctx context.Context, conditionID uint, userID *uint) (RetryStageResult, error) {
	if conditionID == 0 {
		return RetryStageResult{}, errors.New("condition id is required")
	}
	var condition database.IngestCondition
	if err := s.db.WithContext(ctx).First(&condition, conditionID).Error; err != nil {
		return RetryStageResult{}, err
	}
	active, status, err := s.conditionActiveJobStatus(ctx, condition)
	if err != nil {
		return RetryStageResult{}, err
	}
	if active {
		return RetryStageResult{ConditionID: condition.ID, Status: status, Message: "Stage already has active work"}, nil
	}
	if !conditionRetryEligible(condition, conditionStale(condition, s.now())) {
		return RetryStageResult{}, fmt.Errorf("condition %d is not retry eligible", condition.ID)
	}
	if condition.InventoryFileID != nil {
		if _, err := s.MarkInventoryFileDirty(ctx, *condition.InventoryFileID, "admin_retry_"+condition.ConditionType); err != nil {
			return RetryStageResult{}, err
		}
	} else if condition.CatalogItemID != nil {
		if condition.ConditionType == ConditionProjectionCurrent {
			if _, err := s.MarkProjectionItemDirty(ctx, *condition.CatalogItemID, "admin_retry_projection"); err != nil {
				return RetryStageResult{}, err
			}
		} else if _, err := s.MarkCatalogItemDirty(ctx, *condition.CatalogItemID, "admin_retry_"+condition.ConditionType); err != nil {
			return RetryStageResult{}, err
		}
	} else {
		if condition.ConditionType == ConditionProjectionCurrent {
			if _, err := s.MarkProjectionLibraryDirty(ctx, condition.LibraryID, "", "admin_retry_projection"); err != nil {
				return RetryStageResult{}, err
			}
		} else if _, err := s.MarkLibraryScopeDirty(ctx, condition.LibraryID, "", "admin_retry_"+condition.ConditionType); err != nil {
			return RetryStageResult{}, err
		}
	}
	if _, err := s.AppendEvent(ctx, database.IngestEvent{UnitKey: condition.UnitKey, LibraryID: condition.LibraryID, InventoryFileID: condition.InventoryFileID, CatalogItemID: condition.CatalogItemID, ConditionID: &condition.ID, ConditionType: condition.ConditionType, EventType: EventRetryRequested, Status: ConditionStatusPending, Reason: "admin_retry", Message: "Administrator requested ingest stage retry", UserID: userID}); err != nil {
		return RetryStageResult{}, err
	}
	return RetryStageResult{ConditionID: condition.ID, Status: "queued", Message: "Retry queued for affected ingest stage"}, nil
}

func (s *Service) ResolveReviewStage(ctx context.Context, conditionID uint, userID *uint) (ResolveReviewStageResult, error) {
	if conditionID == 0 {
		return ResolveReviewStageResult{}, errors.New("condition id is required")
	}
	var condition database.IngestCondition
	if err := s.db.WithContext(ctx).First(&condition, conditionID).Error; err != nil {
		return ResolveReviewStageResult{}, err
	}
	if condition.ConditionType != ConditionReviewRequired || condition.Status != ConditionStatusReviewRequired {
		return ResolveReviewStageResult{}, fmt.Errorf("condition %d is not a review-required stage", condition.ID)
	}
	reason := strings.TrimSpace(condition.Reason)
	now := s.now()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		switch reason {
		case "classification_needs_review":
			if condition.InventoryFileID == nil || *condition.InventoryFileID == 0 {
				return fmt.Errorf("condition %d is missing inventory file reference", condition.ID)
			}
			fileID := *condition.InventoryFileID
			if err := tx.Model(&database.ClassificationDecision{}).
				Where("inventory_file_id = ? AND status IN ?", fileID, []string{"provisional", "review_required"}).
				Updates(map[string]any{"status": "accepted", "resolved_at": now, "updated_at": now}).Error; err != nil {
				return err
			}
			if err := tx.Model(&database.InventoryFile{}).
				Where("id = ?", fileID).
				Updates(map[string]any{"scan_state": inventory.FileScanStateClassified, "updated_at": now}).Error; err != nil {
				return err
			}
		case "metadata_no_candidate", "metadata_needs_review":
			if condition.CatalogItemID == nil || *condition.CatalogItemID == 0 {
				return fmt.Errorf("condition %d is missing catalog item reference", condition.ID)
			}
			if err := tx.Model(&database.CatalogItem{}).
				Where("id = ?", *condition.CatalogItemID).
				Updates(map[string]any{"governance_status": "manual", "updated_at": now}).Error; err != nil {
				return err
			}
		default:
			return fmt.Errorf("condition %d review reason %q cannot be resolved automatically", condition.ID, reason)
		}
		if err := s.setCondition(ctx, tx, conditionInput{UnitKey: condition.UnitKey, LibraryID: condition.LibraryID, InventoryFileID: condition.InventoryFileID, CatalogItemID: condition.CatalogItemID}.
			with(ConditionReviewRequired, ConditionStatusFalse, "not_required", "No review is required", SeverityInfo)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return ResolveReviewStageResult{}, err
	}
	if _, err := s.AppendEvent(ctx, database.IngestEvent{UnitKey: condition.UnitKey, LibraryID: condition.LibraryID, InventoryFileID: condition.InventoryFileID, CatalogItemID: condition.CatalogItemID, ConditionID: &condition.ID, ConditionType: condition.ConditionType, EventType: EventConditionChanged, Status: ConditionStatusFalse, Reason: "admin_resolved", Message: "Administrator resolved review stage", UserID: userID}); err != nil {
		return ResolveReviewStageResult{}, err
	}
	return ResolveReviewStageResult{ConditionID: condition.ID, Status: "resolved", Message: "Review stage resolved"}, nil
}

func (s *Service) RunEventRetention(ctx context.Context, now time.Time) (int64, error) {
	if now.IsZero() {
		now = s.now()
	}
	return s.PruneExpiredEvents(ctx, now)
}

func (s *Service) buildDiagnosticsResult(ctx context.Context, conditions []database.IngestCondition) (DiagnosticsResult, error) {
	libraryNames, filePaths, itemTitles, err := s.diagnosticReferences(ctx, conditions)
	if err != nil {
		return DiagnosticsResult{}, err
	}
	result := DiagnosticsResult{Stages: make([]DiagnosticStage, 0, len(conditions))}
	now := s.now()
	activeJobIDs, err := s.activeJobIDs(ctx, conditions)
	if err != nil {
		return DiagnosticsResult{}, err
	}
	for _, condition := range conditions {
		if !conditionReportable(condition) {
			continue
		}
		stale := conditionStale(condition, now)
		_, hasActiveJob := activeJobIDs[jobIDValue(condition.JobID)]
		retryEligible := conditionRetryEligible(condition, stale) && !hasActiveJob
		switch condition.Status {
		case ConditionStatusFailed:
			result.Summary.Failed++
		case ConditionStatusReviewRequired:
			result.Summary.ReviewRequired++
		case ConditionStatusPending, ConditionStatusRunning:
			result.Summary.Organizing++
		}
		if condition.ConditionType == ConditionMetadataMatched && condition.Status == ConditionStatusFalse && condition.Reason == "no_candidate" {
			result.Summary.ReviewRequired++
		}
		if stale {
			result.Summary.Stale++
		}
		if retryEligible {
			result.Summary.RetryEligible++
		}
		stage := DiagnosticStage{ID: condition.ID, UnitKey: condition.UnitKey, LibraryID: condition.LibraryID, LibraryName: libraryNames[condition.LibraryID], InventoryFileID: condition.InventoryFileID, CatalogItemID: condition.CatalogItemID, ConditionType: condition.ConditionType, Status: condition.Status, Reason: condition.Reason, Message: condition.Message, Severity: condition.Severity, Attempts: condition.Attempts, JobID: condition.JobID, MetadataOperationID: condition.MetadataOperationID, ProviderInstanceID: condition.ProviderInstanceID, RetryEligible: retryEligible, Stale: stale, UpdatedAt: condition.UpdatedAt, LastTransitionAt: condition.LastTransitionAt}
		if condition.InventoryFileID != nil {
			stage.StoragePath = filePaths[*condition.InventoryFileID]
		}
		if condition.CatalogItemID != nil {
			stage.CatalogTitle = itemTitles[*condition.CatalogItemID]
		}
		result.Stages = append(result.Stages, stage)
	}
	return result, nil
}

func conditionReportable(condition database.IngestCondition) bool {
	if condition.ConditionType != ConditionReviewRequired || condition.Status != ConditionStatusReviewRequired {
		return true
	}
	switch strings.TrimSpace(condition.Reason) {
	case "classification_needs_review":
		return condition.InventoryFileID != nil && *condition.InventoryFileID != 0
	case "metadata_no_candidate", "metadata_needs_review":
		return condition.CatalogItemID != nil && *condition.CatalogItemID != 0
	default:
		return true
	}
}

func (s *Service) conditionActiveJobStatus(ctx context.Context, condition database.IngestCondition) (bool, string, error) {
	return false, "", nil
}

func (s *Service) activeJobIDs(ctx context.Context, conditions []database.IngestCondition) (map[uint]struct{}, error) {
	return map[uint]struct{}{}, nil
}

func isActiveJobStatus(status string) bool {
	return false
}

func jobIDValue(id *uint) uint {
	if id == nil {
		return 0
	}
	return *id
}

func (s *Service) diagnosticReferences(ctx context.Context, conditions []database.IngestCondition) (map[uint]string, map[uint]string, map[uint]string, error) {
	libraryIDs := map[uint]struct{}{}
	fileIDs := map[uint]struct{}{}
	itemIDs := map[uint]struct{}{}
	for _, condition := range conditions {
		libraryIDs[condition.LibraryID] = struct{}{}
		if condition.InventoryFileID != nil {
			fileIDs[*condition.InventoryFileID] = struct{}{}
		}
		if condition.CatalogItemID != nil {
			itemIDs[*condition.CatalogItemID] = struct{}{}
		}
	}
	libraryNames := map[uint]string{}
	if len(libraryIDs) > 0 {
		var libraries []database.Library
		if err := s.db.WithContext(ctx).Where("id IN ?", uintKeys(libraryIDs)).Find(&libraries).Error; err != nil {
			return nil, nil, nil, err
		}
		for _, library := range libraries {
			libraryNames[library.ID] = library.Name
		}
	}
	filePaths := map[uint]string{}
	if len(fileIDs) > 0 {
		var files []database.InventoryFile
		if err := s.db.WithContext(ctx).Where("id IN ?", uintKeys(fileIDs)).Find(&files).Error; err != nil {
			return nil, nil, nil, err
		}
		for _, file := range files {
			filePaths[file.ID] = file.StoragePath
		}
	}
	itemTitles := map[uint]string{}
	if len(itemIDs) > 0 {
		var items []database.CatalogItem
		if err := s.db.WithContext(ctx).Where("id IN ?", uintKeys(itemIDs)).Find(&items).Error; err != nil {
			return nil, nil, nil, err
		}
		for _, item := range items {
			itemTitles[item.ID] = item.Title
		}
	}
	return libraryNames, filePaths, itemTitles, nil
}

func conditionStale(condition database.IngestCondition, now time.Time) bool {
	return condition.StaleAfter != nil && !condition.StaleAfter.After(now)
}

func conditionRetryEligible(condition database.IngestCondition, stale bool) bool {
	if stale {
		return true
	}
	switch condition.Status {
	case ConditionStatusFailed, ConditionStatusReviewRequired, ConditionStatusSkipped:
		return true
	default:
		return false
	}
}

func uintKeys(values map[uint]struct{}) []uint {
	ids := make([]uint, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	return ids
}
