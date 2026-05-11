package metadata

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"gorm.io/gorm"
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
	MetadataItemID uint
	LibraryID      uint
	ImageType      string
	URL            string
	EditedByUserID *uint
}

type LinkGovernanceResourceInput struct {
	ResourceID           uint
	TargetMetadataItemID uint
	SourceMetadataItemID *uint
	LibraryID            uint
	Mode                 string
	Role                 string
	SegmentIndex         int
	StartSeconds         *float64
	EndSeconds           *float64
}

type UnlinkGovernanceResourceInput struct {
	ResourceID     uint
	MetadataItemID uint
	Role           string
	SegmentIndex   int
	LibraryID      uint
}

type UpdateGovernanceResourceLinkInput struct {
	ResourceID     uint
	MetadataItemID uint
	LibraryID      uint
	Role           string
	SegmentIndex   int
	NewRole        string
	ReviewState    string
}

type MergeGovernanceMetadataInput struct {
	SourceMetadataItemID uint
	TargetMetadataItemID uint
	LibraryID            uint
}

type SplitGovernanceMetadataInput struct {
	SourceMetadataItemID uint
	TargetMetadataItemID uint
	ResourceIDs          []uint
	LibraryID            uint
}

type SetGovernanceProjectionVisibilityInput struct {
	MetadataItemID uint
	LibraryID      uint
	Hidden         bool
}

type ApplyGovernanceMetadataFieldInput struct {
	MetadataItemID uint
	LibraryID      uint
	FieldKey       string
	Value          any
	Lock           bool
	LockReason     string
	EditedByUserID *uint
	Force          bool
}

type governanceResourceLinkEvidence struct {
	ResourceID           uint     `json:"resource_id"`
	TargetMetadataItemID uint     `json:"target_metadata_item_id"`
	SourceMetadataItemID *uint    `json:"source_metadata_item_id,omitempty"`
	Mode                 string   `json:"mode,omitempty"`
	Role                 string   `json:"role"`
	SegmentIndex         int      `json:"segment_index"`
	StartSeconds         *float64 `json:"start_seconds,omitempty"`
	EndSeconds           *float64 `json:"end_seconds,omitempty"`
}

type governanceResourceUnlinkEvidence struct {
	ResourceID     uint   `json:"resource_id"`
	MetadataItemID uint   `json:"metadata_item_id"`
	Role           string `json:"role"`
	SegmentIndex   int    `json:"segment_index"`
}

type governanceResourceLinkUpdateEvidence struct {
	ResourceID     uint   `json:"resource_id"`
	MetadataItemID uint   `json:"metadata_item_id"`
	OldRole        string `json:"old_role"`
	NewRole        string `json:"new_role"`
	SegmentIndex   int    `json:"segment_index"`
	OldReviewState string `json:"old_review_state"`
	NewReviewState string `json:"new_review_state"`
}

type governanceMetadataMergeEvidence struct {
	SourceMetadataItemID uint `json:"source_metadata_item_id"`
	TargetMetadataItemID uint `json:"target_metadata_item_id"`
}

type governanceMetadataSplitEvidence struct {
	SourceMetadataItemID uint   `json:"source_metadata_item_id"`
	TargetMetadataItemID uint   `json:"target_metadata_item_id"`
	ResourceIDs          []uint `json:"resource_ids"`
}

type governanceProjectionVisibilityEvidence struct {
	MetadataItemID uint `json:"metadata_item_id"`
	LibraryID      uint `json:"library_id"`
	Hidden         bool `json:"hidden"`
}

func (s *Service) LinkGovernanceResourceOperation(ctx context.Context, input LinkGovernanceResourceInput) (MetadataOperationResult, error) {
	resource, target, plan, err := s.governanceResourceOperationBase(ctx, input.ResourceID, input.TargetMetadataItemID, input.LibraryID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	role := database.NormalizeResourceLinkRole(input.Role)
	if _, err := inventory.NewService(s.db).LinkResourceToMetadata(ctx, inventory.LinkResourceMetadataInput{ResourceID: resource.ID, MetadataItemID: target.ID, Role: role, SegmentIndex: input.SegmentIndex, StartSeconds: input.StartSeconds, EndSeconds: input.EndSeconds, Source: "governance", ReviewState: database.ReviewStateAccepted}); err != nil {
		return MetadataOperationResult{}, err
	}
	affected := []uint{target.ID}
	applied := []MetadataAppliedField{{ItemID: target.ID, FieldKey: "resources.link", ApplyMode: FieldApplyModeManual}}
	if strings.EqualFold(strings.TrimSpace(input.Mode), "move") && input.SourceMetadataItemID != nil && *input.SourceMetadataItemID != target.ID {
		if err := inventory.NewService(s.db).UnlinkResourceFromMetadata(ctx, resource.ID, *input.SourceMetadataItemID, role, input.SegmentIndex); err != nil {
			return MetadataOperationResult{}, err
		}
		affected = appendUniqueUint(affected, *input.SourceMetadataItemID)
		applied = append(applied, MetadataAppliedField{ItemID: *input.SourceMetadataItemID, FieldKey: "resources.unlink", ApplyMode: FieldApplyModeManual})
	}
	if err := s.refreshGovernanceResourceMetadata(ctx, resource.ID, plan.LibraryID, affected...); err != nil {
		return MetadataOperationResult{}, err
	}
	result := newGovernanceOperationResult(GovernanceOperationResultInput{Operation: OperationTypeGovernanceResourceLink, OriginMetadataItemID: derefUintForOperation(input.SourceMetadataItemID), TargetMetadataItemID: target.ID, TargetType: target.ItemType, Status: OperationStatusApplied, GovernanceStatus: strings.TrimSpace(target.GovernanceStatus), Plan: plan, MetadataItemIDs: affected, LibraryID: plan.LibraryID, MetadataRootID: target.RootID, AppliedFields: applied})
	evidence := governanceResourceLinkEvidence{ResourceID: resource.ID, TargetMetadataItemID: target.ID, SourceMetadataItemID: input.SourceMetadataItemID, Mode: strings.TrimSpace(input.Mode), Role: role, SegmentIndex: input.SegmentIndex, StartSeconds: input.StartSeconds, EndSeconds: input.EndSeconds}
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: plan.LibraryID, SelectedCandidate: evidence, StartedAt: time.Now().UTC()})
	return result, err
}

func (s *Service) UnlinkGovernanceResourceOperation(ctx context.Context, input UnlinkGovernanceResourceInput) (MetadataOperationResult, error) {
	resource, target, plan, err := s.governanceResourceOperationBase(ctx, input.ResourceID, input.MetadataItemID, input.LibraryID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	role := database.NormalizeResourceLinkRole(input.Role)
	if err := inventory.NewService(s.db).UnlinkResourceFromMetadata(ctx, resource.ID, target.ID, role, input.SegmentIndex); err != nil {
		return MetadataOperationResult{}, err
	}
	if err := s.refreshGovernanceResourceMetadata(ctx, resource.ID, plan.LibraryID, target.ID); err != nil {
		return MetadataOperationResult{}, err
	}
	result := newGovernanceOperationResult(GovernanceOperationResultInput{Operation: OperationTypeGovernanceResourceUnlink, OriginMetadataItemID: target.ID, TargetMetadataItemID: target.ID, TargetType: target.ItemType, Status: OperationStatusApplied, GovernanceStatus: strings.TrimSpace(target.GovernanceStatus), Plan: plan, MetadataItemIDs: []uint{target.ID}, LibraryID: plan.LibraryID, MetadataRootID: target.RootID, AppliedFields: []MetadataAppliedField{{ItemID: target.ID, FieldKey: "resources.unlink", ApplyMode: FieldApplyModeManual}}})
	evidence := governanceResourceUnlinkEvidence{ResourceID: resource.ID, MetadataItemID: target.ID, Role: role, SegmentIndex: input.SegmentIndex}
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: plan.LibraryID, SelectedCandidate: evidence, StartedAt: time.Now().UTC()})
	return result, err
}

func (s *Service) UpdateGovernanceResourceLinkOperation(ctx context.Context, input UpdateGovernanceResourceLinkInput) (MetadataOperationResult, error) {
	resource, target, plan, err := s.governanceResourceOperationBase(ctx, input.ResourceID, input.MetadataItemID, input.LibraryID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	oldRole := database.NormalizeResourceLinkRole(input.Role)
	newRole := database.NormalizeResourceLinkRole(input.NewRole)
	newReviewState := database.NormalizeReviewState(input.ReviewState)
	var link database.ResourceMetadataLink
	if err := s.db.WithContext(ctx).Where("resource_id = ? AND metadata_item_id = ? AND role = ? AND segment_index = ?", resource.ID, target.ID, oldRole, input.SegmentIndex).First(&link).Error; err != nil {
		return MetadataOperationResult{}, err
	}
	oldReviewState := link.ReviewState
	link.Role = newRole
	link.ReviewState = newReviewState
	link.Source = "governance"
	if err := s.db.WithContext(ctx).Save(&link).Error; err != nil {
		return MetadataOperationResult{}, err
	}
	if err := s.refreshGovernanceResourceMetadata(ctx, resource.ID, plan.LibraryID, target.ID); err != nil {
		return MetadataOperationResult{}, err
	}
	result := newGovernanceOperationResult(GovernanceOperationResultInput{Operation: OperationTypeGovernanceResourceLinkUpdate, OriginMetadataItemID: target.ID, TargetMetadataItemID: target.ID, TargetType: target.ItemType, Status: OperationStatusApplied, GovernanceStatus: strings.TrimSpace(target.GovernanceStatus), Plan: plan, MetadataItemIDs: []uint{target.ID}, LibraryID: plan.LibraryID, MetadataRootID: target.RootID, AppliedFields: []MetadataAppliedField{{ItemID: target.ID, FieldKey: "resources.role", ApplyMode: FieldApplyModeManual}, {ItemID: target.ID, FieldKey: "resources.review_state", ApplyMode: FieldApplyModeManual}}})
	evidence := governanceResourceLinkUpdateEvidence{ResourceID: resource.ID, MetadataItemID: target.ID, OldRole: oldRole, NewRole: newRole, SegmentIndex: input.SegmentIndex, OldReviewState: oldReviewState, NewReviewState: newReviewState}
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: plan.LibraryID, SelectedCandidate: evidence, StartedAt: time.Now().UTC()})
	return result, err
}

func (s *Service) MergeGovernanceMetadataOperation(ctx context.Context, input MergeGovernanceMetadataInput) (MetadataOperationResult, error) {
	if input.SourceMetadataItemID == 0 || input.TargetMetadataItemID == 0 || input.SourceMetadataItemID == input.TargetMetadataItemID {
		return MetadataOperationResult{}, errors.New("distinct source and target metadata item ids are required")
	}
	source, target, plan, err := s.governanceMetadataOperationBase(ctx, input.SourceMetadataItemID, input.TargetMetadataItemID, input.LibraryID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := moveMetadataIdentityRows(ctx, tx, input.SourceMetadataItemID, input.TargetMetadataItemID); err != nil {
			return err
		}
		now := time.Now().UTC()
		return tx.WithContext(ctx).Model(&database.MetadataItem{}).Where("id = ?", input.SourceMetadataItemID).Updates(map[string]any{"deleted_at": now, "updated_at": now}).Error
	})
	if err != nil {
		return MetadataOperationResult{}, err
	}
	if err := s.cleanupRetiredMetadataIdentity(ctx, input.SourceMetadataItemID); err != nil {
		return MetadataOperationResult{}, err
	}
	if err := s.refreshGovernanceMetadataItems(ctx, plan.LibraryID, input.TargetMetadataItemID); err != nil {
		return MetadataOperationResult{}, err
	}
	result := newGovernanceOperationResult(GovernanceOperationResultInput{Operation: OperationTypeGovernanceMetadataMerge, OriginMetadataItemID: source.ID, TargetMetadataItemID: target.ID, TargetType: target.ItemType, Status: OperationStatusApplied, GovernanceStatus: strings.TrimSpace(target.GovernanceStatus), Plan: plan, MetadataItemIDs: []uint{source.ID, target.ID}, LibraryID: plan.LibraryID, MetadataRootID: target.RootID, AppliedFields: []MetadataAppliedField{{ItemID: target.ID, FieldKey: "metadata.merge", ApplyMode: FieldApplyModeManual}}})
	evidence := governanceMetadataMergeEvidence{SourceMetadataItemID: source.ID, TargetMetadataItemID: target.ID}
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: plan.LibraryID, SelectedCandidate: evidence, StartedAt: time.Now().UTC()})
	return result, err
}

func (s *Service) SplitGovernanceMetadataOperation(ctx context.Context, input SplitGovernanceMetadataInput) (MetadataOperationResult, error) {
	if input.SourceMetadataItemID == 0 || input.TargetMetadataItemID == 0 || input.SourceMetadataItemID == input.TargetMetadataItemID || len(input.ResourceIDs) == 0 {
		return MetadataOperationResult{}, errors.New("source metadata item id, target metadata item id, and resource ids are required")
	}
	source, target, plan, err := s.governanceMetadataOperationBase(ctx, input.SourceMetadataItemID, input.TargetMetadataItemID, input.LibraryID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	resourceIDs := appendUniqueUint(nil, input.ResourceIDs...)
	err = s.db.WithContext(ctx).Model(&database.ResourceMetadataLink{}).Where("metadata_item_id = ? AND resource_id IN ?", source.ID, resourceIDs).Update("metadata_item_id", target.ID).Error
	if err != nil {
		return MetadataOperationResult{}, err
	}
	if err := s.refreshGovernanceMetadataItems(ctx, plan.LibraryID, source.ID, target.ID); err != nil {
		return MetadataOperationResult{}, err
	}
	result := newGovernanceOperationResult(GovernanceOperationResultInput{Operation: OperationTypeGovernanceMetadataSplit, OriginMetadataItemID: source.ID, TargetMetadataItemID: target.ID, TargetType: target.ItemType, Status: OperationStatusApplied, GovernanceStatus: strings.TrimSpace(target.GovernanceStatus), Plan: plan, MetadataItemIDs: []uint{source.ID, target.ID}, LibraryID: plan.LibraryID, MetadataRootID: target.RootID, AppliedFields: []MetadataAppliedField{{ItemID: target.ID, FieldKey: "metadata.split", ApplyMode: FieldApplyModeManual}}})
	evidence := governanceMetadataSplitEvidence{SourceMetadataItemID: source.ID, TargetMetadataItemID: target.ID, ResourceIDs: resourceIDs}
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: plan.LibraryID, SelectedCandidate: evidence, StartedAt: time.Now().UTC()})
	return result, err
}

func (s *Service) SetGovernanceProjectionVisibilityOperation(ctx context.Context, input SetGovernanceProjectionVisibilityInput) (MetadataOperationResult, error) {
	if input.MetadataItemID == 0 || input.LibraryID == 0 {
		return MetadataOperationResult{}, errors.New("metadata item id and library id are required")
	}
	var item database.MetadataItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", input.MetadataItemID).First(&item).Error; err != nil {
		return MetadataOperationResult{}, err
	}
	plan, err := s.resolveMetadataExecutionPlan(ctx, input.LibraryID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	if err := catalog.NewService(s.db).RefreshLibraryProjection(ctx, input.LibraryID, input.MetadataItemID); err != nil {
		return MetadataOperationResult{}, err
	}
	if err := s.db.WithContext(ctx).Model(&database.LibraryMetadataProjection{}).Where("library_id = ? AND metadata_item_id = ?", input.LibraryID, input.MetadataItemID).Update("hidden", input.Hidden).Error; err != nil {
		return MetadataOperationResult{}, err
	}
	result := newGovernanceOperationResult(GovernanceOperationResultInput{Operation: OperationTypeGovernanceProjectionVisibility, OriginMetadataItemID: item.ID, TargetMetadataItemID: item.ID, TargetType: item.ItemType, Status: OperationStatusApplied, GovernanceStatus: strings.TrimSpace(item.GovernanceStatus), Plan: plan, MetadataItemIDs: []uint{item.ID}, LibraryID: input.LibraryID, MetadataRootID: item.RootID, AppliedFields: []MetadataAppliedField{{ItemID: item.ID, FieldKey: "projection.hidden", ApplyMode: FieldApplyModeManual}}})
	evidence := governanceProjectionVisibilityEvidence{MetadataItemID: item.ID, LibraryID: input.LibraryID, Hidden: input.Hidden}
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: input.LibraryID, SelectedCandidate: evidence, StartedAt: time.Now().UTC()})
	return result, err
}

func (s *Service) ApplyGovernanceMetadataFieldOperation(ctx context.Context, input ApplyGovernanceMetadataFieldInput) (MetadataOperationResult, error) {
	if input.MetadataItemID == 0 || strings.TrimSpace(input.FieldKey) == "" {
		return MetadataOperationResult{}, errors.New("metadata item id and field key are required")
	}
	var item database.MetadataItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", input.MetadataItemID).First(&item).Error; err != nil {
		return MetadataOperationResult{}, err
	}
	libraryID := input.LibraryID
	if libraryID == 0 {
		libraryID = firstGovernanceMetadataLibraryID(ctx, s.db, item.ID)
	}
	if libraryID == 0 {
		return MetadataOperationResult{}, errors.New("library id is required for metadata field governance")
	}
	plan, err := s.resolveMetadataExecutionPlan(ctx, libraryID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	fieldKey := strings.TrimSpace(input.FieldKey)
	applied, skipped, err := s.applyMetadataItemFieldChanges(ctx, []MetadataFieldChange{{ItemID: item.ID, FieldKey: fieldKey, Value: input.Value, ApplyMode: FieldApplyModeManual, Force: input.Force}})
	if err != nil {
		return MetadataOperationResult{}, err
	}
	if len(skipped) > 0 {
		result := newGovernanceOperationResult(GovernanceOperationResultInput{Operation: OperationTypeManualApply, OriginMetadataItemID: item.ID, TargetMetadataItemID: item.ID, TargetType: item.ItemType, Status: OperationStatusSkipped, GovernanceStatus: strings.TrimSpace(item.GovernanceStatus), Plan: plan, MetadataItemIDs: []uint{item.ID}, LibraryID: libraryID, MetadataRootID: item.RootID, SkippedFields: skipped})
		_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: libraryID, StartedAt: time.Now().UTC()})
		return result, err
	}
	if input.Lock {
		if err := s.db.WithContext(ctx).Model(&database.MetadataItemFieldState{}).Where("metadata_item_id = ? AND field_key = ? AND locale = ?", item.ID, fieldKey, "").Updates(map[string]any{"is_locked": true, "lock_reason": strings.TrimSpace(input.LockReason), "edited_by_user_id": input.EditedByUserID, "edited_at": time.Now().UTC()}).Error; err != nil {
			return MetadataOperationResult{}, err
		}
	}
	if err := s.refreshGovernanceMetadataItems(ctx, libraryID, item.ID); err != nil {
		return MetadataOperationResult{}, err
	}
	result := newGovernanceOperationResult(GovernanceOperationResultInput{Operation: OperationTypeManualApply, OriginMetadataItemID: item.ID, TargetMetadataItemID: item.ID, TargetType: item.ItemType, Status: OperationStatusApplied, GovernanceStatus: strings.TrimSpace(item.GovernanceStatus), Plan: plan, MetadataItemIDs: []uint{item.ID}, LibraryID: libraryID, MetadataRootID: item.RootID, AppliedFields: applied})
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: libraryID, StartedAt: time.Now().UTC()})
	return result, err
}

func (s *Service) SelectGovernanceMetadataImageOperation(ctx context.Context, input SelectGovernanceImageInput) (MetadataOperationResult, error) {
	imageType := strings.TrimSpace(input.ImageType)
	url := strings.TrimSpace(input.URL)
	if input.MetadataItemID == 0 || imageType == "" || url == "" {
		return MetadataOperationResult{}, errors.New("metadata item id, image type, and url are required")
	}
	var item database.MetadataItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", input.MetadataItemID).First(&item).Error; err != nil {
		return MetadataOperationResult{}, err
	}
	libraryID := input.LibraryID
	if libraryID == 0 {
		resolvedLibraryID, err := s.resolveMetadataOperationLibraryID(ctx, item.ID)
		if err != nil {
			return MetadataOperationResult{}, err
		}
		libraryID = resolvedLibraryID
	}
	plan, err := s.resolveMetadataExecutionPlan(ctx, libraryID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	if err := s.db.WithContext(ctx).Model(&database.MetadataItemImage{}).Where("metadata_item_id = ? AND image_type = ?", item.ID, imageType).Update("is_selected", false).Error; err != nil {
		return MetadataOperationResult{}, err
	}
	row := database.MetadataItemImage{MetadataItemID: item.ID, ImageType: imageType, URL: url, IsSelected: true}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return MetadataOperationResult{}, err
	}
	applied, skipped, err := s.applyMetadataItemFieldChanges(ctx, []MetadataFieldChange{{ItemID: item.ID, FieldKey: "governance_status", Value: catalog.GovernanceManual, ApplyMode: FieldApplyModeManual, Force: true}})
	if err != nil {
		return MetadataOperationResult{}, err
	}
	applied = append(applied, MetadataAppliedField{ItemID: item.ID, FieldKey: "image." + imageType, ApplyMode: FieldApplyModeManual})
	if err := s.refreshGovernanceMetadataItems(ctx, libraryID, item.ID); err != nil {
		return MetadataOperationResult{}, err
	}
	result := newGovernanceOperationResult(GovernanceOperationResultInput{Operation: OperationTypeManualApply, OriginMetadataItemID: item.ID, TargetMetadataItemID: item.ID, TargetType: item.ItemType, Status: OperationStatusApplied, GovernanceStatus: catalog.GovernanceManual, Plan: plan, MetadataItemIDs: []uint{item.ID}, LibraryID: libraryID, MetadataRootID: item.RootID, AppliedFields: applied, SkippedFields: skipped})
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: libraryID, SelectedCandidate: map[string]any{"image_type": imageType, "url": url}, StartedAt: time.Now().UTC()})
	return result, err
}

func (s *Service) governanceResourceOperationBase(ctx context.Context, resourceID uint, metadataItemID uint, libraryID uint) (database.Resource, database.MetadataItem, MetadataExecutionPlan, error) {
	if resourceID == 0 || metadataItemID == 0 {
		return database.Resource{}, database.MetadataItem{}, MetadataExecutionPlan{}, errors.New("resource id and metadata item id are required")
	}
	var resource database.Resource
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", resourceID).First(&resource).Error; err != nil {
		return database.Resource{}, database.MetadataItem{}, MetadataExecutionPlan{}, err
	}
	var item database.MetadataItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", metadataItemID).First(&item).Error; err != nil {
		return database.Resource{}, database.MetadataItem{}, MetadataExecutionPlan{}, err
	}
	if libraryID == 0 {
		if err := s.db.WithContext(ctx).Model(&database.ResourceLibraryLink{}).Where("resource_id = ? AND deleted_at IS NULL", resourceID).Order("library_id asc").Limit(1).Pluck("library_id", &libraryID).Error; err != nil {
			return database.Resource{}, database.MetadataItem{}, MetadataExecutionPlan{}, err
		}
	}
	if libraryID == 0 {
		return database.Resource{}, database.MetadataItem{}, MetadataExecutionPlan{}, errors.New("library id is required for resource governance")
	}
	plan, err := s.resolveMetadataExecutionPlan(ctx, libraryID)
	if err != nil {
		return database.Resource{}, database.MetadataItem{}, MetadataExecutionPlan{}, err
	}
	return resource, item, plan, nil
}

func (s *Service) governanceMetadataOperationBase(ctx context.Context, sourceMetadataItemID uint, targetMetadataItemID uint, libraryID uint) (database.MetadataItem, database.MetadataItem, MetadataExecutionPlan, error) {
	var source database.MetadataItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", sourceMetadataItemID).First(&source).Error; err != nil {
		return database.MetadataItem{}, database.MetadataItem{}, MetadataExecutionPlan{}, err
	}
	var target database.MetadataItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", targetMetadataItemID).First(&target).Error; err != nil {
		return database.MetadataItem{}, database.MetadataItem{}, MetadataExecutionPlan{}, err
	}
	if libraryID == 0 {
		libraryID = firstGovernanceMetadataLibraryID(ctx, s.db, sourceMetadataItemID, targetMetadataItemID)
	}
	if libraryID == 0 {
		return database.MetadataItem{}, database.MetadataItem{}, MetadataExecutionPlan{}, errors.New("library id is required for metadata governance")
	}
	plan, err := s.resolveMetadataExecutionPlan(ctx, libraryID)
	if err != nil {
		return database.MetadataItem{}, database.MetadataItem{}, MetadataExecutionPlan{}, err
	}
	return source, target, plan, nil
}

func (s *Service) refreshGovernanceResourceMetadata(ctx context.Context, resourceID uint, fallbackLibraryID uint, metadataItemIDs ...uint) error {
	catalogSvc := catalog.NewService(s.db)
	if err := catalogSvc.RebuildResourceMetadataProjections(ctx, resourceID); err != nil {
		return err
	}
	var libraryIDs []uint
	if err := s.db.WithContext(ctx).Model(&database.ResourceLibraryLink{}).Where("resource_id = ? AND deleted_at IS NULL", resourceID).Distinct().Pluck("library_id", &libraryIDs).Error; err != nil {
		return err
	}
	libraryIDs = appendUniqueUint(libraryIDs, fallbackLibraryID)
	for _, metadataItemID := range appendUniqueUint(nil, metadataItemIDs...) {
		if _, err := catalogSvc.RebuildMetadataSearchDocument(ctx, metadataItemID); err != nil {
			return err
		}
		for _, libraryID := range libraryIDs {
			if _, err := catalogSvc.RebuildLibraryMetadataProjection(ctx, libraryID, metadataItemID); err != nil {
				return err
			}
			if _, err := catalogSvc.RebuildLibrarySearchDocument(ctx, libraryID, metadataItemID); err != nil {
				return err
			}
		}
		if err := catalogSvc.RebuildMetadataItemProjections(ctx, metadataItemID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) refreshGovernanceMetadataItems(ctx context.Context, fallbackLibraryID uint, metadataItemIDs ...uint) error {
	catalogSvc := catalog.NewService(s.db)
	for _, metadataItemID := range appendUniqueUint(nil, metadataItemIDs...) {
		if _, err := catalogSvc.RebuildMetadataSearchDocument(ctx, metadataItemID); err != nil {
			return err
		}
		if err := catalogSvc.RebuildMetadataItemProjections(ctx, metadataItemID); err != nil {
			return err
		}
		for _, libraryID := range metadataItemLibraryIDs(ctx, s.db, fallbackLibraryID, metadataItemID) {
			if _, err := catalogSvc.RebuildLibraryMetadataProjection(ctx, libraryID, metadataItemID); err != nil {
				return err
			}
			if _, err := catalogSvc.RebuildLibrarySearchDocument(ctx, libraryID, metadataItemID); err != nil {
				return err
			}
		}
	}
	return nil
}

func firstGovernanceMetadataLibraryID(ctx context.Context, db *gorm.DB, metadataItemIDs ...uint) uint {
	for _, metadataItemID := range metadataItemIDs {
		libraryIDs := metadataItemLibraryIDs(ctx, db, 0, metadataItemID)
		if len(libraryIDs) > 0 {
			return libraryIDs[0]
		}
	}
	return 0
}

func metadataItemLibraryIDs(ctx context.Context, db *gorm.DB, fallbackLibraryID uint, metadataItemID uint) []uint {
	libraryIDs := make([]uint, 0)
	_ = db.WithContext(ctx).Table("resource_library_links").Select("DISTINCT resource_library_links.library_id").Joins("JOIN resource_metadata_links ON resource_metadata_links.resource_id = resource_library_links.resource_id").Where("resource_metadata_links.metadata_item_id = ? AND resource_library_links.deleted_at IS NULL", metadataItemID).Pluck("library_id", &libraryIDs).Error
	return appendUniqueUint(libraryIDs, fallbackLibraryID)
}

func moveMetadataIdentityRows(ctx context.Context, tx *gorm.DB, sourceID uint, targetID uint) error {
	if err := tx.WithContext(ctx).Model(&database.ResourceMetadataLink{}).Where("metadata_item_id = ?", sourceID).Update("metadata_item_id", targetID).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Model(&database.MetadataItemSource{}).Where("metadata_item_id = ?", sourceID).Update("metadata_item_id", targetID).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Model(&database.MetadataExternalID{}).Where("metadata_item_id = ?", sourceID).Update("metadata_item_id", targetID).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Model(&database.MetadataItemFieldState{}).Where("metadata_item_id = ?", sourceID).Update("metadata_item_id", targetID).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Model(&database.MetadataItemImage{}).Where("metadata_item_id = ?", sourceID).Update("metadata_item_id", targetID).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Model(&database.MetadataItemPerson{}).Where("metadata_item_id = ?", sourceID).Update("metadata_item_id", targetID).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Model(&database.MetadataItemTag{}).Where("metadata_item_id = ?", sourceID).Update("metadata_item_id", targetID).Error; err != nil {
		return err
	}
	if err := mergeUserMetadataData(ctx, tx, sourceID, targetID); err != nil {
		return err
	}
	return tx.WithContext(ctx).Model(&database.UserResourceData{}).Where("metadata_item_id = ?", sourceID).Update("metadata_item_id", targetID).Error
}

func (s *Service) cleanupRetiredMetadataIdentity(ctx context.Context, metadataItemID uint) error {
	return s.db.WithContext(ctx).Where("metadata_item_id = ?", metadataItemID).Delete(&database.LibraryMetadataProjection{}).Error
}

func mergeUserMetadataData(ctx context.Context, tx *gorm.DB, sourceID uint, targetID uint) error {
	var rows []database.UserMetadataData
	if err := tx.WithContext(ctx).Where("metadata_item_id = ?", sourceID).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		var target database.UserMetadataData
		err := tx.WithContext(ctx).Where("user_id = ? AND metadata_item_id = ?", row.UserID, targetID).First(&target).Error
		if err == gorm.ErrRecordNotFound {
			row.ID = 0
			row.MetadataItemID = targetID
			if err := tx.WithContext(ctx).Create(&row).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		updates := map[string]any{"favorite": target.Favorite || row.Favorite, "play_count": target.PlayCount + row.PlayCount, "updated_at": time.Now().UTC()}
		if row.PositionSeconds > target.PositionSeconds {
			updates["position_seconds"] = row.PositionSeconds
			updates["played_percentage"] = row.PlayedPercentage
			updates["progress_frame_url"] = row.ProgressFrameURL
		}
		if row.PreferredResourceID != nil {
			updates["preferred_resource_id"] = row.PreferredResourceID
		}
		if newerTimePtr(row.LastPlayedAt, target.LastPlayedAt) {
			updates["last_played_at"] = row.LastPlayedAt
		}
		if newerTimePtr(row.CompletedAt, target.CompletedAt) {
			updates["completed_at"] = row.CompletedAt
		}
		if err := tx.WithContext(ctx).Model(&database.UserMetadataData{}).Where("id = ?", target.ID).Updates(updates).Error; err != nil {
			return err
		}
	}
	return tx.WithContext(ctx).Where("metadata_item_id = ?", sourceID).Delete(&database.UserMetadataData{}).Error
}

func newerTimePtr(candidate *time.Time, current *time.Time) bool {
	return candidate != nil && (current == nil || candidate.After(*current))
}
