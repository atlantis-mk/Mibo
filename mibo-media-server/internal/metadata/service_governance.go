package metadata

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
)

type ApplyGovernanceFieldInput struct {
	FieldKey       string
	Value          any
	Lock           bool
	LockReason     string
	EditedByUserID *uint
	Force          bool
}

type SelectGovernanceImageInput struct {
	ImageType      string
	URL            string
	EditedByUserID *uint
}

type CorrectGovernanceEpisodeNumberingInput struct {
	SeasonNumber     int
	EpisodeNumber    int
	EpisodeNumberEnd *int
}

type CreateGovernanceClassificationRuleInput struct {
	Name            string
	Description     string
	PathPattern     string
	RuleType        string
	Role            string
	CandidateType   string
	SeriesTitle     string
	SeasonNumber    *int
	NumberingSource string
	CreatedByUserID *uint
}

type LinkGovernanceAssetInput struct {
	AssetID      uint
	TargetItemID uint
	SourceItemID *uint
	Mode         string
	SegmentIndex int
	StartSeconds *float64
	EndSeconds   *float64
}

type UnlinkGovernanceAssetInput struct {
	AssetID      uint
	TargetItemID uint
}

type governanceEpisodeNumberingEvidence struct {
	OldParentID          *uint `json:"old_parent_id,omitempty"`
	OldRootID            *uint `json:"old_root_id,omitempty"`
	OldParentIndexNumber *int  `json:"old_parent_index_number,omitempty"`
	OldIndexNumber       *int  `json:"old_index_number,omitempty"`
	OldIndexNumberEnd    *int  `json:"old_index_number_end,omitempty"`
	NewParentID          *uint `json:"new_parent_id,omitempty"`
	NewRootID            *uint `json:"new_root_id,omitempty"`
	NewParentIndexNumber *int  `json:"new_parent_index_number,omitempty"`
	NewIndexNumber       *int  `json:"new_index_number,omitempty"`
	NewIndexNumberEnd    *int  `json:"new_index_number_end,omitempty"`
}

type governanceClassificationRuleEvidence struct {
	RuleID          uint   `json:"rule_id"`
	Key             string `json:"key"`
	Name            string `json:"name"`
	PathPattern     string `json:"path_pattern"`
	RuleType        string `json:"rule_type"`
	Role            string `json:"role"`
	CandidateType   string `json:"candidate_type"`
	SeriesTitle     string `json:"series_title"`
	SeasonNumber    *int   `json:"season_number,omitempty"`
	NumberingSource string `json:"numbering_source"`
}

type governanceAssetLinkEvidence struct {
	AssetID      uint     `json:"asset_id"`
	TargetItemID uint     `json:"target_item_id"`
	SourceItemID *uint    `json:"source_item_id,omitempty"`
	Mode         string   `json:"mode,omitempty"`
	SegmentIndex int      `json:"segment_index"`
	StartSeconds *float64 `json:"start_seconds,omitempty"`
	EndSeconds   *float64 `json:"end_seconds,omitempty"`
}

type governanceAssetUnlinkEvidence struct {
	AssetID      uint `json:"asset_id"`
	TargetItemID uint `json:"target_item_id"`
}

func (s *Service) ApplyCatalogGovernanceFieldOperation(ctx context.Context, itemID uint, input ApplyGovernanceFieldInput) (MetadataOperationResult, error) {
	item, plan, result, err := s.governanceOperationBase(ctx, itemID, itemID, OperationTypeManualApply)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	catalogSvc := catalog.NewService(s.db)
	fieldKey := strings.TrimSpace(input.FieldKey)
	_, applied, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: item.ID, FieldKey: fieldKey, Value: input.Value, Lock: input.Lock, LockReason: input.LockReason, EditedByUserID: input.EditedByUserID, Force: input.Force})
	if err != nil {
		return MetadataOperationResult{}, err
	}
	if !applied {
		result.Status = OperationStatusSkipped
		result.SkippedFields = []MetadataSkippedField{{ItemID: item.ID, FieldKey: fieldKey, Reason: "locked"}}
		_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: item.LibraryID, StartedAt: time.Now().UTC()})
		return result, err
	}
	governanceStatus, err := s.applyGovernanceStatusForField(ctx, catalogSvc, item.ID, fieldKey, input.Value, input.Lock, input.EditedByUserID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	result.Status = OperationStatusApplied
	result.GovernanceStatus = governanceStatus
	result.AppliedFields = []MetadataAppliedField{{ItemID: item.ID, FieldKey: fieldKey, ApplyMode: FieldApplyModeManual}}
	if fieldKey != "governance_status" {
		result.AppliedFields = append(result.AppliedFields, MetadataAppliedField{ItemID: item.ID, FieldKey: "governance_status", ApplyMode: FieldApplyModeManual})
	}
	if err := s.acceptClassificationReviewForItem(ctx, item.ID); err != nil {
		return MetadataOperationResult{}, err
	}
	if err := s.refreshMetadataOperationProjectionScope(ctx, result.AffectedScope); err != nil {
		return MetadataOperationResult{}, err
	}
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: plan.LibraryID, StartedAt: time.Now().UTC()})
	return result, err
}

func (s *Service) SelectCatalogGovernanceImageOperation(ctx context.Context, itemID uint, input SelectGovernanceImageInput) (MetadataOperationResult, error) {
	item, plan, result, err := s.governanceOperationBase(ctx, itemID, itemID, OperationTypeManualApply)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	catalogSvc := catalog.NewService(s.db)
	imageType := strings.TrimSpace(input.ImageType)
	if err := catalogSvc.SelectImage(ctx, item.ID, imageType, input.URL); err != nil {
		return MetadataOperationResult{}, err
	}
	if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: item.ID, FieldKey: "governance_status", Value: catalog.GovernanceManual, EditedByUserID: input.EditedByUserID, Force: true}); err != nil {
		return MetadataOperationResult{}, err
	}
	result.Status = OperationStatusApplied
	result.GovernanceStatus = catalog.GovernanceManual
	result.AppliedFields = []MetadataAppliedField{{ItemID: item.ID, FieldKey: "image." + imageType, ApplyMode: FieldApplyModeManual}, {ItemID: item.ID, FieldKey: "governance_status", ApplyMode: FieldApplyModeManual}}
	if err := s.acceptClassificationReviewForItem(ctx, item.ID); err != nil {
		return MetadataOperationResult{}, err
	}
	if err := s.refreshMetadataOperationProjectionScope(ctx, result.AffectedScope); err != nil {
		return MetadataOperationResult{}, err
	}
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: plan.LibraryID, SelectedCandidate: map[string]any{"image_type": imageType, "url": strings.TrimSpace(input.URL)}, StartedAt: time.Now().UTC()})
	return result, err
}

func (s *Service) CorrectCatalogGovernanceEpisodeNumberingOperation(ctx context.Context, episodeID uint, input CorrectGovernanceEpisodeNumberingInput) (MetadataOperationResult, error) {
	before, err := s.loadGovernanceCatalogItem(ctx, episodeID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	catalogSvc := catalog.NewService(s.db)
	updated, err := catalogSvc.CorrectEpisodeNumbering(ctx, catalog.CorrectEpisodeNumberingInput{EpisodeID: episodeID, SeasonNumber: input.SeasonNumber, EpisodeNumber: input.EpisodeNumber, EpisodeNumberEnd: input.EpisodeNumberEnd})
	if err != nil {
		return MetadataOperationResult{}, err
	}
	_, plan, result, err := s.governanceOperationBase(ctx, episodeID, updated.ID, OperationTypeGovernanceEpisodeNumbering)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	result.Status = OperationStatusApplied
	result.GovernanceStatus = strings.TrimSpace(updated.GovernanceStatus)
	result.AppliedFields = []MetadataAppliedField{
		{ItemID: updated.ID, FieldKey: "parent_id", ApplyMode: FieldApplyModeManual},
		{ItemID: updated.ID, FieldKey: "root_id", ApplyMode: FieldApplyModeManual},
		{ItemID: updated.ID, FieldKey: "parent_index_number", ApplyMode: FieldApplyModeManual},
		{ItemID: updated.ID, FieldKey: "index_number", ApplyMode: FieldApplyModeManual},
	}
	if input.EpisodeNumberEnd != nil || before.IndexNumberEnd != nil || updated.IndexNumberEnd != nil {
		result.AppliedFields = append(result.AppliedFields, MetadataAppliedField{ItemID: updated.ID, FieldKey: "index_number_end", ApplyMode: FieldApplyModeManual})
	}
	result.AffectedScope.ItemIDs = appendUniqueUint(result.AffectedScope.ItemIDs, derefUintForOperation(updated.ParentID), derefUintForOperation(updated.RootID), derefUintForOperation(before.ParentID))
	evidence := governanceEpisodeNumberingEvidence{OldParentID: before.ParentID, OldRootID: before.RootID, OldParentIndexNumber: before.ParentIndexNumber, OldIndexNumber: before.IndexNumber, OldIndexNumberEnd: before.IndexNumberEnd, NewParentID: updated.ParentID, NewRootID: updated.RootID, NewParentIndexNumber: updated.ParentIndexNumber, NewIndexNumber: updated.IndexNumber, NewIndexNumberEnd: updated.IndexNumberEnd}
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: plan.LibraryID, SelectedCandidate: evidence, StartedAt: time.Now().UTC()})
	return result, err
}

func (s *Service) CreateCatalogGovernanceClassificationRuleOperation(ctx context.Context, itemID uint, input CreateGovernanceClassificationRuleInput) (MetadataOperationResult, error) {
	item, plan, result, err := s.governanceOperationBase(ctx, itemID, itemID, OperationTypeGovernanceClassificationRule)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	rule, err := catalog.NewService(s.db).CreateClassificationRule(ctx, catalog.ClassificationRuleInput{LibraryID: item.LibraryID, Name: input.Name, Description: input.Description, PathPattern: input.PathPattern, RuleType: input.RuleType, Role: input.Role, CandidateType: input.CandidateType, SeriesTitle: input.SeriesTitle, SeasonNumber: input.SeasonNumber, NumberingSource: input.NumberingSource, CreatedByUserID: input.CreatedByUserID})
	if err != nil {
		return MetadataOperationResult{}, err
	}
	result.Status = OperationStatusApplied
	result.GovernanceStatus = strings.TrimSpace(item.GovernanceStatus)
	result.AppliedFields = []MetadataAppliedField{{ItemID: item.ID, FieldKey: "classification_rules.applied", ApplyMode: FieldApplyModeManual}}
	evidence := governanceClassificationRuleEvidence{RuleID: rule.ID, Key: rule.Key, Name: rule.Name, PathPattern: rule.PathPattern, RuleType: rule.RuleType, Role: rule.Role, CandidateType: rule.CandidateType, SeriesTitle: rule.SeriesTitle, SeasonNumber: rule.SeasonNumber, NumberingSource: rule.NumberingSource}
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: plan.LibraryID, SelectedCandidate: evidence, StartedAt: time.Now().UTC()})
	return result, err
}

func (s *Service) LinkCatalogGovernanceAssetOperation(ctx context.Context, workspaceItemID uint, input LinkGovernanceAssetInput) (MetadataOperationResult, error) {
	workspace, err := s.loadGovernanceCatalogItem(ctx, workspaceItemID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	target, err := s.validateGovernanceTarget(ctx, workspaceItemID, input.TargetItemID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	if input.SourceItemID != nil {
		if _, err := s.validateGovernanceTarget(ctx, workspaceItemID, *input.SourceItemID); err != nil {
			return MetadataOperationResult{}, err
		}
	}
	_, plan, result, err := s.governanceOperationBase(ctx, workspaceItemID, target.ID, OperationTypeGovernanceAssetLink)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	inventorySvc := inventory.NewService(s.db)
	if _, err := inventorySvc.LinkAssetToItem(ctx, inventory.LinkAssetItemInput{AssetID: input.AssetID, ItemID: input.TargetItemID, Role: inventory.AssetItemRolePrimary, SegmentIndex: input.SegmentIndex, StartSeconds: input.StartSeconds, EndSeconds: input.EndSeconds, Source: "governance"}); err != nil {
		return MetadataOperationResult{}, err
	}
	applied := []MetadataAppliedField{{ItemID: input.TargetItemID, FieldKey: "assets.link", ApplyMode: FieldApplyModeManual}}
	if strings.EqualFold(strings.TrimSpace(input.Mode), "move") && input.SourceItemID != nil && *input.SourceItemID != input.TargetItemID {
		if err := inventorySvc.UnlinkAssetFromItem(ctx, input.AssetID, *input.SourceItemID); err != nil {
			return MetadataOperationResult{}, err
		}
		applied = append(applied, MetadataAppliedField{ItemID: *input.SourceItemID, FieldKey: "assets.unlink", ApplyMode: FieldApplyModeManual})
		result.AffectedScope.ItemIDs = appendUniqueUint(result.AffectedScope.ItemIDs, *input.SourceItemID)
	}
	result.Status = OperationStatusApplied
	result.GovernanceStatus = strings.TrimSpace(target.GovernanceStatus)
	result.AppliedFields = applied
	result.AffectedScope.LibraryID = workspace.LibraryID
	evidence := governanceAssetLinkEvidence{AssetID: input.AssetID, TargetItemID: input.TargetItemID, SourceItemID: input.SourceItemID, Mode: strings.TrimSpace(input.Mode), SegmentIndex: input.SegmentIndex, StartSeconds: input.StartSeconds, EndSeconds: input.EndSeconds}
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: plan.LibraryID, SelectedCandidate: evidence, StartedAt: time.Now().UTC()})
	return result, err
}

func (s *Service) UnlinkCatalogGovernanceAssetOperation(ctx context.Context, workspaceItemID uint, input UnlinkGovernanceAssetInput) (MetadataOperationResult, error) {
	workspace, err := s.loadGovernanceCatalogItem(ctx, workspaceItemID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	target, err := s.validateGovernanceTarget(ctx, workspaceItemID, input.TargetItemID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	_, plan, result, err := s.governanceOperationBase(ctx, workspaceItemID, target.ID, OperationTypeGovernanceAssetUnlink)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	if err := inventory.NewService(s.db).UnlinkAssetFromItem(ctx, input.AssetID, input.TargetItemID); err != nil {
		return MetadataOperationResult{}, err
	}
	result.Status = OperationStatusApplied
	result.GovernanceStatus = strings.TrimSpace(target.GovernanceStatus)
	result.AppliedFields = []MetadataAppliedField{{ItemID: input.TargetItemID, FieldKey: "assets.unlink", ApplyMode: FieldApplyModeManual}}
	result.AffectedScope.LibraryID = workspace.LibraryID
	evidence := governanceAssetUnlinkEvidence{AssetID: input.AssetID, TargetItemID: input.TargetItemID}
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: plan.LibraryID, SelectedCandidate: evidence, StartedAt: time.Now().UTC()})
	return result, err
}

func (s *Service) governanceOperationBase(ctx context.Context, originItemID uint, targetItemID uint, operation string) (database.CatalogItem, MetadataExecutionPlan, MetadataOperationResult, error) {
	target, err := s.loadGovernanceCatalogItem(ctx, targetItemID)
	if err != nil {
		return database.CatalogItem{}, MetadataExecutionPlan{}, MetadataOperationResult{}, err
	}
	plan, err := s.resolveMetadataExecutionPlan(ctx, target.LibraryID)
	if err != nil {
		return database.CatalogItem{}, MetadataExecutionPlan{}, MetadataOperationResult{}, err
	}
	result := MetadataOperationResult{Operation: operation, OriginItemID: originItemID, TargetItemID: target.ID, TargetType: target.Type, Plan: metadataExecutionPlanSummary(plan), AffectedScope: MetadataAffectedScope{ItemIDs: []uint{target.ID}, LibraryID: target.LibraryID, RootID: target.RootID}}
	return target, plan, result, nil
}

func (s *Service) loadGovernanceCatalogItem(ctx context.Context, itemID uint) (database.CatalogItem, error) {
	if itemID == 0 {
		return database.CatalogItem{}, errors.New("item id is required")
	}
	var item database.CatalogItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", itemID).First(&item).Error; err != nil {
		return database.CatalogItem{}, err
	}
	return item, nil
}

func (s *Service) validateGovernanceTarget(ctx context.Context, workspaceItemID uint, targetItemID uint) (database.CatalogItem, error) {
	allowed, err := catalog.NewService(s.db).IsGovernanceTargetAllowed(ctx, workspaceItemID, targetItemID)
	if err != nil {
		return database.CatalogItem{}, err
	}
	if !allowed {
		return database.CatalogItem{}, fmt.Errorf("target_item_id 必须是当前治理条目或其后代")
	}
	return s.loadGovernanceCatalogItem(ctx, targetItemID)
}

func (s *Service) applyGovernanceStatusForField(ctx context.Context, catalogSvc *catalog.Service, itemID uint, fieldKey string, value any, lock bool, userID *uint) (string, error) {
	if fieldKey == "governance_status" {
		status, ok := value.(string)
		if !ok || strings.TrimSpace(status) == "" {
			return "", fmt.Errorf("field governance_status requires a string value")
		}
		return strings.TrimSpace(status), nil
	}
	status := catalog.GovernanceManual
	if lock {
		status = catalog.GovernanceLocked
	}
	_, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: itemID, FieldKey: "governance_status", Value: status, EditedByUserID: userID, Force: true})
	return status, err
}

func (s *Service) acceptClassificationReviewForItem(ctx context.Context, itemID uint) error {
	if itemID == 0 {
		return nil
	}
	itemIDs, err := s.catalogItemScopeIDs(ctx, itemID)
	if err != nil {
		return err
	}
	if len(itemIDs) == 0 {
		return nil
	}
	now := time.Now().UTC()
	var fileIDs []uint
	if err := s.db.WithContext(ctx).Model(&database.ClassificationDecision{}).
		Where("item_id IN ? AND inventory_file_id IS NOT NULL AND status IN ?", itemIDs, []string{"provisional", "review_required"}).
		Distinct().
		Pluck("inventory_file_id", &fileIDs).Error; err != nil {
		return err
	}
	if len(fileIDs) == 0 {
		return nil
	}
	if err := s.db.WithContext(ctx).Model(&database.ClassificationDecision{}).
		Where("item_id IN ? AND status IN ?", itemIDs, []string{"provisional", "review_required"}).
		Updates(map[string]any{"status": "accepted", "resolved_at": now, "updated_at": now}).Error; err != nil {
		return err
	}
	if s.ingest == nil {
		return nil
	}
	for _, fileID := range fileIDs {
		if fileID == 0 {
			continue
		}
		if _, err := s.ingest.MarkInventoryFileDirty(ctx, fileID, "classification_metadata_confirmed"); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) catalogItemScopeIDs(ctx context.Context, itemID uint) ([]uint, error) {
	var items []database.CatalogItem
	if err := s.db.WithContext(ctx).
		Select("id").
		Where("id = ? OR parent_id = ? OR root_id = ?", itemID, itemID, itemID).
		Find(&items).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids, nil
}
