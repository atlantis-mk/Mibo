package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	bulkSQLChunkSize = 400
	bulkWriteSize    = 50

	DirtyStatusDirty     = "dirty"
	DirtyStatusClaimed   = "claimed"
	DirtyStatusCompleted = "completed"
	DirtyStatusFailed    = "failed"

	ScopeKindInventoryFile     = "inventory_file"
	ScopeKindCatalogItem       = "catalog_item"
	ScopeKindLibrary           = "library"
	ScopeKindProjectionItem    = "projection_item"
	ScopeKindProjectionLibrary = "projection_library"

	ConditionVisible           = "visible"
	ConditionMaterialized      = "materialized"
	ConditionProbed            = "probed"
	ConditionMetadataMatched   = "metadata_matched"
	ConditionProjectionCurrent = "projection_current"
	ConditionReviewRequired    = "review_required"

	ConditionStatusUnknown        = "unknown"
	ConditionStatusPending        = "pending"
	ConditionStatusRunning        = "running"
	ConditionStatusTrue           = "true"
	ConditionStatusFalse          = "false"
	ConditionStatusSkipped        = "skipped"
	ConditionStatusFailed         = "failed"
	ConditionStatusReviewRequired = "review_required"

	SeverityInfo    = "info"
	SeverityWarning = "warning"
	SeverityError   = "error"

	EventConditionChanged = "condition_changed"
	EventRetryRequested   = "retry_requested"
	EventDirtyClaimed     = "dirty_claimed"
	EventDirtyFailed      = "dirty_failed"

	DefaultEventRetention = 30 * 24 * time.Hour

	jobKindCatalogMatchBatch               = "catalog_match_batch"
	jobKindCatalogMaterializeBatch         = "catalog_materialize_batch"
	jobKindInventoryProbeBatch             = "inventory_probe_batch"
	jobKindCatalogRefreshItemProjection    = "catalog_refresh_item_projection"
	jobKindCatalogRefreshLibraryProjection = "catalog_refresh_library_projection"
)

type Service struct {
	db       *gorm.DB
	workflow *workflow.Service
	now      func() time.Time
}

type ReconcileResult struct {
	Claimed   int
	Processed int
	Failed    int
	RetryDue  int
}

type conditionInput struct {
	UnitKey             string
	LibraryID           uint
	InventoryFileID     *uint
	CatalogItemID       *uint
	ConditionType       string
	Status              string
	Reason              string
	Message             string
	Severity            string
	Attempts            int
	JobID               *uint
	MetadataOperationID *uint
	ProviderInstanceID  *uint
	DetailsJSON         string
	NextAttemptAt       *time.Time
	StaleAfter          *time.Time
}

type CatalogMatchBatchPayload struct {
	LibraryID uint   `json:"library_id"`
	RootPath  string `json:"root_path,omitempty"`
	ItemIDs   []uint `json:"item_ids"`
}

type InventoryProbeBatchPayload struct {
	LibraryID uint   `json:"library_id"`
	RootPath  string `json:"root_path,omitempty"`
	FileIDs   []uint `json:"file_ids"`
}

type CatalogMaterializeBatchPayload struct {
	LibraryID uint   `json:"library_id"`
	RootPath  string `json:"root_path,omitempty"`
	FileIDs   []uint `json:"file_ids"`
}

type itemProjectionRefreshPayload struct {
	ItemID uint `json:"item_id"`
}

type libraryProjectionRefreshPayload struct {
	LibraryID uint   `json:"library_id"`
	RootPath  string `json:"root_path"`
}

func NewService(db *gorm.DB, args ...any) *Service {
	service := &Service{db: db, now: func() time.Time { return time.Now().UTC() }}
	for _, arg := range args {
		if workflowSvc, ok := arg.(*workflow.Service); ok {
			service.workflow = workflowSvc
		}
	}
	if service.workflow == nil && db != nil {
		service.workflow = workflow.NewService(db)
	}
	return service
}

func (s *Service) MarkInventoryFileDirty(ctx context.Context, inventoryFileID uint, reason string) (database.IngestDirtyUnit, error) {
	if inventoryFileID == 0 {
		return database.IngestDirtyUnit{}, errors.New("inventory file id is required")
	}
	var file database.InventoryFile
	if err := s.db.WithContext(ctx).Where("id = ?", inventoryFileID).First(&file).Error; err != nil {
		return database.IngestDirtyUnit{}, err
	}
	fileID := file.ID
	return s.upsertDirty(ctx, database.IngestDirtyUnit{DirtyKey: inventoryFileUnitKey(file.ID), ScopeKind: ScopeKindInventoryFile, LibraryID: file.LibraryID, InventoryFileID: &fileID, RootPath: strings.TrimSpace(file.StoragePath), Reason: normalizeReason(reason), Status: DirtyStatusDirty, AvailableAt: s.now()})
}

func (s *Service) MarkCatalogItemDirty(ctx context.Context, itemID uint, reason string) (database.IngestDirtyUnit, error) {
	if itemID == 0 {
		return database.IngestDirtyUnit{}, errors.New("catalog item id is required")
	}
	var item database.CatalogItem
	if err := s.db.WithContext(ctx).Where("id = ?", itemID).First(&item).Error; err != nil {
		return database.IngestDirtyUnit{}, err
	}
	currentItemID := item.ID
	return s.upsertDirty(ctx, database.IngestDirtyUnit{DirtyKey: catalogItemUnitKey(item.ID), ScopeKind: ScopeKindCatalogItem, LibraryID: item.LibraryID, CatalogItemID: &currentItemID, RootPath: strings.TrimSpace(item.Path), Reason: normalizeReason(reason), Status: DirtyStatusDirty, AvailableAt: s.now()})
}

func (s *Service) MarkLibraryScopeDirty(ctx context.Context, libraryID uint, rootPath string, reason string) (database.IngestDirtyUnit, error) {
	if libraryID == 0 {
		return database.IngestDirtyUnit{}, errors.New("library id is required")
	}
	rootPath = strings.TrimSpace(rootPath)
	return s.upsertDirty(ctx, database.IngestDirtyUnit{DirtyKey: fmt.Sprintf("library:%d:%s", libraryID, rootPath), ScopeKind: ScopeKindLibrary, LibraryID: libraryID, RootPath: rootPath, Reason: normalizeReason(reason), Status: DirtyStatusDirty, AvailableAt: s.now()})
}

func (s *Service) MarkProjectionItemDirty(ctx context.Context, itemID uint, reason string) (database.IngestDirtyUnit, error) {
	if itemID == 0 {
		return database.IngestDirtyUnit{}, errors.New("catalog item id is required")
	}
	var item database.CatalogItem
	if err := s.db.WithContext(ctx).Where("id = ?", itemID).First(&item).Error; err != nil {
		return database.IngestDirtyUnit{}, err
	}
	currentItemID := item.ID
	return s.upsertDirty(ctx, database.IngestDirtyUnit{DirtyKey: fmt.Sprintf("projection_item:%d", item.ID), ScopeKind: ScopeKindProjectionItem, LibraryID: item.LibraryID, CatalogItemID: &currentItemID, RootPath: strings.TrimSpace(item.Path), Reason: normalizeReason(reason), Status: DirtyStatusDirty, AvailableAt: s.now()})
}

func (s *Service) MarkProjectionLibraryDirty(ctx context.Context, libraryID uint, rootPath string, reason string) (database.IngestDirtyUnit, error) {
	if libraryID == 0 {
		return database.IngestDirtyUnit{}, errors.New("library id is required")
	}
	rootPath = strings.TrimSpace(rootPath)
	return s.upsertDirty(ctx, database.IngestDirtyUnit{DirtyKey: fmt.Sprintf("projection_library:%d:%s", libraryID, rootPath), ScopeKind: ScopeKindProjectionLibrary, LibraryID: libraryID, RootPath: rootPath, Reason: normalizeReason(reason), Status: DirtyStatusDirty, AvailableAt: s.now()})
}

func (s *Service) AppendEvent(ctx context.Context, event database.IngestEvent) (database.IngestEvent, error) {
	if event.UnitKey == "" {
		return database.IngestEvent{}, errors.New("unit key is required")
	}
	if event.LibraryID == 0 {
		return database.IngestEvent{}, errors.New("library id is required")
	}
	if strings.TrimSpace(event.EventType) == "" {
		event.EventType = EventConditionChanged
	}
	if event.ExpiresAt == nil {
		expires := s.now().Add(DefaultEventRetention)
		event.ExpiresAt = &expires
	}
	if err := s.db.WithContext(ctx).Create(&event).Error; err != nil {
		return database.IngestEvent{}, err
	}
	return event, nil
}

func (s *Service) MarkInventoryFilesDirty(ctx context.Context, fileIDs []uint, reason string) error {
	ids := uniqueUintIDs(fileIDs)
	if len(ids) == 0 {
		return nil
	}
	var files []database.InventoryFile
	for _, batch := range chunkUintIDs(ids, bulkSQLChunkSize) {
		var partial []database.InventoryFile
		if err := s.db.WithContext(ctx).Where("id IN ?", batch).Find(&partial).Error; err != nil {
			return err
		}
		files = append(files, partial...)
	}
	now := s.now()
	units := make([]database.IngestDirtyUnit, 0, len(files))
	for _, file := range files {
		fileID := file.ID
		units = append(units, database.IngestDirtyUnit{DirtyKey: inventoryFileUnitKey(file.ID), ScopeKind: ScopeKindInventoryFile, LibraryID: file.LibraryID, InventoryFileID: &fileID, RootPath: strings.TrimSpace(file.StoragePath), Reason: normalizeReason(reason), Status: DirtyStatusDirty, AvailableAt: now})
	}
	return s.bulkUpsertDirty(ctx, units)
}

func (s *Service) AppendEvents(ctx context.Context, events []database.IngestEvent) error {
	if len(events) == 0 {
		return nil
	}
	now := s.now()
	for idx := range events {
		if strings.TrimSpace(events[idx].UnitKey) == "" {
			return errors.New("unit key is required")
		}
		if events[idx].LibraryID == 0 {
			return errors.New("library id is required")
		}
		if strings.TrimSpace(events[idx].EventType) == "" {
			events[idx].EventType = EventConditionChanged
		}
		if events[idx].ExpiresAt == nil {
			expires := now.Add(DefaultEventRetention)
			events[idx].ExpiresAt = &expires
		}
	}
	return s.db.WithContext(ctx).CreateInBatches(&events, bulkWriteSize).Error
}

func (s *Service) PruneExpiredEvents(ctx context.Context, now time.Time) (int64, error) {
	result := s.db.WithContext(ctx).Where("expires_at IS NOT NULL AND expires_at <= ?", now.UTC()).Delete(&database.IngestEvent{})
	return result.RowsAffected, result.Error
}

func (s *Service) ClaimDirty(ctx context.Context, limit int) ([]database.IngestDirtyUnit, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	now := s.now()
	var claimed []database.IngestDirtyUnit
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []database.IngestDirtyUnit
		if err := tx.Where("status = ? AND available_at <= ?", DirtyStatusDirty, now).
			Order("available_at asc, id asc").
			Limit(limit).
			Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		ids := make([]uint, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ID)
		}
		if err := tx.Model(&database.IngestDirtyUnit{}).
			Where("id IN ? AND status = ?", ids, DirtyStatusDirty).
			Updates(map[string]any{"status": DirtyStatusClaimed, "claimed_at": now, "attempts": gorm.Expr("attempts + 1"), "last_error": ""}).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", ids).Order("available_at asc, id asc").Find(&claimed).Error
	})
	return claimed, err
}

func (s *Service) ReconcileOnce(ctx context.Context, limit int) (ReconcileResult, error) {
	if err := s.markRetryDueConditionsDirty(ctx, limit); err != nil {
		return ReconcileResult{}, err
	}
	claimed, err := s.ClaimDirty(ctx, limit)
	if err != nil {
		return ReconcileResult{}, err
	}
	result := ReconcileResult{Claimed: len(claimed)}
	for _, unit := range claimed {
		if err := s.reconcileDirtyUnit(ctx, unit); err != nil {
			result.Failed++
			if updateErr := s.markDirtyFailed(ctx, unit.ID, err); updateErr != nil {
				return result, updateErr
			}
			continue
		}
		result.Processed++
		if err := s.markDirtyCompleted(ctx, unit.ID); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (s *Service) markRetryDueConditionsDirty(ctx context.Context, limit int) error {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var conditions []database.IngestCondition
	if err := s.db.WithContext(ctx).
		Where("next_attempt_at IS NOT NULL AND next_attempt_at <= ?", s.now()).
		Where("status IN ?", []string{ConditionStatusPending, ConditionStatusFailed}).
		Order("next_attempt_at asc, id asc").
		Limit(limit).
		Find(&conditions).Error; err != nil {
		return err
	}
	for _, condition := range conditions {
		reason := "retry_due_" + condition.ConditionType
		if condition.InventoryFileID != nil {
			if _, err := s.MarkInventoryFileDirty(ctx, *condition.InventoryFileID, reason); err != nil {
				return err
			}
		} else if condition.CatalogItemID != nil {
			if condition.ConditionType == ConditionProjectionCurrent {
				if _, err := s.MarkProjectionItemDirty(ctx, *condition.CatalogItemID, reason); err != nil {
					return err
				}
			} else if _, err := s.MarkCatalogItemDirty(ctx, *condition.CatalogItemID, reason); err != nil {
				return err
			}
		} else if _, err := s.MarkLibraryScopeDirty(ctx, condition.LibraryID, "", reason); err != nil {
			return err
		}
		if err := s.db.WithContext(ctx).Model(&database.IngestCondition{}).Where("id = ?", condition.ID).Update("next_attempt_at", nil).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) upsertDirty(ctx context.Context, unit database.IngestDirtyUnit) (database.IngestDirtyUnit, error) {
	if strings.TrimSpace(unit.DirtyKey) == "" {
		return database.IngestDirtyUnit{}, errors.New("dirty key is required")
	}
	if unit.LibraryID == 0 {
		return database.IngestDirtyUnit{}, errors.New("library id is required")
	}
	if strings.TrimSpace(unit.ScopeKind) == "" {
		return database.IngestDirtyUnit{}, errors.New("scope kind is required")
	}
	if unit.AvailableAt.IsZero() {
		unit.AvailableAt = s.now()
	}
	unit.Reason = normalizeReason(unit.Reason)
	unit.Status = DirtyStatusDirty
	var stored database.IngestDirtyUnit
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "dirty_key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"scope_kind":        unit.ScopeKind,
			"library_id":        unit.LibraryID,
			"inventory_file_id": unit.InventoryFileID,
			"catalog_item_id":   unit.CatalogItemID,
			"root_path":         strings.TrimSpace(unit.RootPath),
			"reason":            unit.Reason,
			"status":            DirtyStatusDirty,
			"available_at":      unit.AvailableAt,
			"claimed_at":        nil,
			"last_error":        "",
			"updated_at":        s.now(),
		}),
	}).Create(&unit).Error
	if err != nil {
		return database.IngestDirtyUnit{}, err
	}
	if err := s.db.WithContext(ctx).Where("dirty_key = ?", unit.DirtyKey).First(&stored).Error; err != nil {
		return database.IngestDirtyUnit{}, err
	}
	return stored, nil
}

func (s *Service) bulkUpsertDirty(ctx context.Context, units []database.IngestDirtyUnit) error {
	if len(units) == 0 {
		return nil
	}
	now := s.now()
	for idx := range units {
		if strings.TrimSpace(units[idx].DirtyKey) == "" {
			return errors.New("dirty key is required")
		}
		if units[idx].LibraryID == 0 {
			return errors.New("library id is required")
		}
		if strings.TrimSpace(units[idx].ScopeKind) == "" {
			return errors.New("scope kind is required")
		}
		if units[idx].AvailableAt.IsZero() {
			units[idx].AvailableAt = now
		}
		units[idx].Reason = normalizeReason(units[idx].Reason)
		units[idx].Status = DirtyStatusDirty
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "dirty_key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"scope_kind":        gorm.Expr("excluded.scope_kind"),
			"library_id":        gorm.Expr("excluded.library_id"),
			"inventory_file_id": gorm.Expr("excluded.inventory_file_id"),
			"catalog_item_id":   gorm.Expr("excluded.catalog_item_id"),
			"root_path":         gorm.Expr("excluded.root_path"),
			"reason":            gorm.Expr("excluded.reason"),
			"status":            DirtyStatusDirty,
			"available_at":      gorm.Expr("excluded.available_at"),
			"claimed_at":        nil,
			"last_error":        "",
			"updated_at":        now,
		}),
	}).CreateInBatches(&units, bulkWriteSize).Error
}

func chunkUintIDs(values []uint, size int) [][]uint {
	if len(values) == 0 {
		return nil
	}
	if size <= 0 {
		size = len(values)
	}
	chunks := make([][]uint, 0, (len(values)+size-1)/size)
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[start:end])
	}
	return chunks
}

func uniqueUintIDs(ids []uint) []uint {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func (s *Service) reconcileDirtyUnit(ctx context.Context, unit database.IngestDirtyUnit) error {
	switch unit.ScopeKind {
	case ScopeKindInventoryFile:
		if unit.InventoryFileID == nil || *unit.InventoryFileID == 0 {
			return errors.New("dirty inventory unit missing inventory_file_id")
		}
		return s.reconcileInventoryFile(ctx, *unit.InventoryFileID, unit.Reason)
	case ScopeKindCatalogItem:
		if unit.CatalogItemID == nil || *unit.CatalogItemID == 0 {
			return errors.New("dirty catalog unit missing catalog_item_id")
		}
		return s.reconcileCatalogItem(ctx, *unit.CatalogItemID, unit.Reason)
	case ScopeKindProjectionItem:
		if unit.CatalogItemID == nil || *unit.CatalogItemID == 0 {
			return errors.New("dirty projection unit missing catalog_item_id")
		}
		return s.reconcileProjectionItem(ctx, *unit.CatalogItemID)
	case ScopeKindLibrary:
		return s.expandLibraryScope(ctx, unit)
	case ScopeKindProjectionLibrary:
		return s.reconcileProjectionLibrary(ctx, unit.LibraryID, unit.RootPath)
	default:
		return fmt.Errorf("unsupported ingest dirty scope kind %q", unit.ScopeKind)
	}
}

func (s *Service) reconcileInventoryFile(ctx context.Context, fileID uint, reason string) error {
	var file database.InventoryFile
	var itemIDs []uint
	var targets []database.CatalogItem
	var conditions []conditionInput
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", fileID).First(&file).Error; err != nil {
			return err
		}
		unitKey := inventoryFileUnitKey(file.ID)
		linkedItems, err := linkedCatalogItemIDs(tx, file.ID)
		if err != nil {
			return err
		}
		assetIDs, err := linkedAssetIDs(tx, file.ID)
		if err != nil {
			return err
		}
		metadataTargets, err := metadataTargetsForItems(tx, linkedItems)
		if err != nil {
			return err
		}
		derivedConditions := s.inventoryFileConditions(tx, file, unitKey, linkedItems, assetIDs, metadataTargets)
		for _, condition := range derivedConditions {
			if err := s.setCondition(ctx, tx, condition); err != nil {
				return err
			}
		}
		itemIDs = linkedItems
		targets = metadataTargets
		conditions = derivedConditions
		return nil
	})
	if err != nil {
		return err
	}
	return s.dispatchInventoryFileWork(ctx, file, itemIDs, targets, conditions, reason)
}

func (s *Service) reconcileCatalogItem(ctx context.Context, itemID uint, reason string) error {
	var item database.CatalogItem
	var targets []database.CatalogItem
	var conditions []conditionInput
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", itemID).First(&item).Error; err != nil {
			return err
		}
		unitKey := catalogItemUnitKey(item.ID)
		metadataTargets, err := metadataTargetsForItems(tx, []uint{item.ID})
		if err != nil {
			return err
		}
		derivedConditions := s.catalogItemConditions(tx, item, unitKey, metadataTargets)
		for _, condition := range derivedConditions {
			if err := s.setCondition(ctx, tx, condition); err != nil {
				return err
			}
		}
		targets = metadataTargets
		conditions = derivedConditions
		return nil
	})
	if err != nil {
		return err
	}
	return s.dispatchCatalogItemWork(ctx, item, targets, conditions, reason)
}

func (s *Service) reconcileProjectionItem(ctx context.Context, itemID uint) error {
	var item database.CatalogItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", itemID).First(&item).Error; err != nil {
		return err
	}
	if err := s.reconcileCatalogItem(ctx, item.ID, "projection_refresh"); err != nil {
		return err
	}
	job, err := s.queueItemProjectionRefresh(ctx, item.ID)
	if err != nil {
		return err
	}
	return s.markProjectionRefreshQueued(ctx, item, job)
}

func (s *Service) markProjectionRefreshQueued(ctx context.Context, item database.CatalogItem, job database.Job) error {
	if job.ID == 0 {
		return nil
	}
	itemID := item.ID
	condition := conditionInput{UnitKey: catalogItemUnitKey(item.ID), LibraryID: item.LibraryID, CatalogItemID: &itemID, ConditionType: ConditionProjectionCurrent, Status: ConditionStatusPending, Reason: "projection_refresh_queued", Message: "Catalog projection refresh is queued", Severity: SeverityInfo, JobID: &job.ID}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.setCondition(ctx, tx, condition)
	})
}

func (s *Service) reconcileProjectionLibrary(ctx context.Context, libraryID uint, rootPath string) error {
	if libraryID == 0 {
		return errors.New("library id is required")
	}
	return s.queueLibraryProjectionRefresh(ctx, libraryID, rootPath)
}

func (s *Service) expandLibraryScope(ctx context.Context, unit database.IngestDirtyUnit) error {
	query := s.db.WithContext(ctx).Model(&database.InventoryFile{}).
		Where("library_id = ? AND deleted_at IS NULL AND content_class = ?", unit.LibraryID, "video")
	if strings.TrimSpace(unit.RootPath) != "" {
		query = query.Where("storage_path = ? OR storage_path LIKE ?", unit.RootPath, strings.TrimRight(unit.RootPath, "/")+"/%")
	}
	var fileIDs []uint
	if err := query.Order("id asc").Pluck("id", &fileIDs).Error; err != nil {
		return err
	}
	for _, fileID := range fileIDs {
		if _, err := s.MarkInventoryFileDirty(ctx, fileID, unit.Reason); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) dispatchInventoryFileWork(ctx context.Context, file database.InventoryFile, itemIDs []uint, targets []database.CatalogItem, conditions []conditionInput, reason string) error {
	if s.workflow == nil || file.ID == 0 || file.DeletedAt != nil || strings.TrimSpace(file.Status) == "missing" || strings.TrimSpace(file.ContentClass) != "video" {
		return nil
	}
	if shouldDispatchCondition(reason, conditions, ConditionMaterialized) && len(itemIDs) == 0 {
		job, err := s.queueCatalogMaterializeBatch(ctx, file.LibraryID, file.StoragePath, []uint{file.ID})
		return s.attachDispatchJob(ctx, conditions, ConditionMaterialized, job, err)
	}
	if shouldDispatchCondition(reason, conditions, ConditionProbed) && len(itemIDs) > 0 {
		job, err := s.queueInventoryProbeBatch(ctx, file.LibraryID, file.StoragePath, []uint{file.ID})
		return s.attachDispatchJob(ctx, conditions, ConditionProbed, job, err)
	}
	if shouldDispatchCondition(reason, conditions, ConditionMetadataMatched) {
		job, err := s.queueCatalogMatchBatch(ctx, file.LibraryID, file.StoragePath, targetIDs(targets))
		return s.attachDispatchJob(ctx, conditions, ConditionMetadataMatched, job, err)
	}
	if shouldDispatchCondition(reason, conditions, ConditionProjectionCurrent) {
		for _, target := range targets {
			job, err := s.queueItemProjectionRefresh(ctx, target.ID)
			if err := s.attachDispatchJob(ctx, conditions, ConditionProjectionCurrent, job, err); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) dispatchCatalogItemWork(ctx context.Context, item database.CatalogItem, targets []database.CatalogItem, conditions []conditionInput, reason string) error {
	if s.workflow == nil || item.ID == 0 || item.DeletedAt != nil {
		return nil
	}
	if shouldDispatchCondition(reason, conditions, ConditionMetadataMatched) {
		job, err := s.queueCatalogMatchBatch(ctx, item.LibraryID, item.Path, targetIDs(targets))
		return s.attachDispatchJob(ctx, conditions, ConditionMetadataMatched, job, err)
	}
	if shouldDispatchCondition(reason, conditions, ConditionProjectionCurrent) {
		for _, target := range targets {
			job, err := s.queueItemProjectionRefresh(ctx, target.ID)
			if err := s.attachDispatchJob(ctx, conditions, ConditionProjectionCurrent, job, err); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) queueCatalogMaterializeBatch(ctx context.Context, libraryID uint, rootPath string, fileIDs []uint) (database.Job, error) {
	ids := normalizeUintIDs(fileIDs)
	if s.workflow != nil && len(ids) > 0 {
		run, err := s.queueWorkflowTask(ctx, libraryID, rootPath, workflow.TaskTypeMaterializeCatalog, workflow.StageMaterialize, CatalogMaterializeBatchPayload{LibraryID: libraryID, RootPath: rootPath, FileIDs: ids})
		return workflowCompatibilityJob(run), err
	}
	if len(ids) == 0 {
		return database.Job{}, nil
	}
	return database.Job{}, fmt.Errorf("workflow service unavailable")
}

func (s *Service) queueInventoryProbeBatch(ctx context.Context, libraryID uint, rootPath string, fileIDs []uint) (database.Job, error) {
	ids := normalizeUintIDs(fileIDs)
	if s.workflow != nil && len(ids) > 0 {
		run, err := s.queueWorkflowTask(ctx, libraryID, rootPath, workflow.TaskTypeProbeInventory, workflow.StageProbe, InventoryProbeBatchPayload{LibraryID: libraryID, RootPath: rootPath, FileIDs: ids})
		return workflowCompatibilityJob(run), err
	}
	if len(ids) == 0 {
		return database.Job{}, nil
	}
	return database.Job{}, fmt.Errorf("workflow service unavailable")
}

func (s *Service) queueCatalogMatchBatch(ctx context.Context, libraryID uint, rootPath string, itemIDs []uint) (database.Job, error) {
	ids := normalizeUintIDs(itemIDs)
	if s.workflow != nil && len(ids) > 0 {
		run, err := s.queueWorkflowTask(ctx, libraryID, rootPath, workflow.TaskTypeMatchMetadata, workflow.StageMetadataMatch, CatalogMatchBatchPayload{LibraryID: libraryID, RootPath: rootPath, ItemIDs: ids})
		return workflowCompatibilityJob(run), err
	}
	if len(ids) == 0 {
		return database.Job{}, nil
	}
	return database.Job{}, fmt.Errorf("workflow service unavailable")
}

func (s *Service) queueItemProjectionRefresh(ctx context.Context, itemID uint) (database.Job, error) {
	if s.workflow != nil && itemID != 0 {
		var item database.CatalogItem
		if err := s.db.WithContext(ctx).First(&item, itemID).Error; err != nil {
			return database.Job{}, err
		}
		run, err := s.queueWorkflowTask(ctx, item.LibraryID, item.Path, workflow.TaskTypeRefreshProjection, workflow.StageProjection, itemProjectionRefreshPayload{ItemID: itemID})
		return workflowCompatibilityJob(run), err
	}
	if itemID == 0 {
		return database.Job{}, nil
	}
	return database.Job{}, fmt.Errorf("workflow service unavailable")
}

func (s *Service) queueLibraryProjectionRefresh(ctx context.Context, libraryID uint, rootPath string) error {
	if s.workflow != nil && libraryID != 0 {
		_, err := s.queueWorkflowTask(ctx, libraryID, rootPath, workflow.TaskTypeRefreshProjection, workflow.StageProjection, libraryProjectionRefreshPayload{LibraryID: libraryID, RootPath: strings.TrimSpace(rootPath)})
		return err
	}
	if libraryID == 0 {
		return nil
	}
	return fmt.Errorf("workflow service unavailable")
}

func (s *Service) queueWorkflowTask(ctx context.Context, libraryID uint, rootPath string, taskType string, stage string, payload any) (database.WorkflowRun, error) {
	if s.workflow == nil {
		return database.WorkflowRun{}, fmt.Errorf("workflow service unavailable")
	}
	if libraryID == 0 {
		return database.WorkflowRun{}, nil
	}
	rootPath = strings.TrimSpace(rootPath)
	runKey := fmt.Sprintf("ingest:%d:%s:%s", libraryID, taskType, rootPath)
	run, reused, err := s.workflow.CreateOrReuseRun(ctx, workflow.CreateRunInput{RunKey: runKey, LibraryID: libraryID, Reason: "ingest_dispatch", Priority: 5, ScopeKey: fmt.Sprintf("library:%d", libraryID), Payload: payload})
	if err != nil || reused {
		return run, err
	}
	definition := workflow.DefaultTaskTypeDefinitions()[taskType]
	if stage == "" {
		stage = definition.Stage
	}
	_, err = s.workflow.CreateTask(ctx, run, workflow.CreateTaskInput{TaskKey: fmt.Sprintf("run:%d:%s:%s", run.ID, taskType, rootPath), TaskType: taskType, Stage: stage, Priority: 5, ScopeKey: run.ScopeKey, Payload: payload, Resources: definition.Resources})
	return run, err
}

func workflowCompatibilityJob(run database.WorkflowRun) database.Job {
	now := time.Now().UTC()
	return database.Job{ID: run.ID, JobKey: run.RunKey, Kind: run.Reason, Status: run.Status, PayloadJSON: run.PayloadJSON, ErrorMessage: run.ErrorMessage, AvailableAt: now, CreatedAt: run.CreatedAt, UpdatedAt: run.UpdatedAt}
}

func (s *Service) attachDispatchJob(ctx context.Context, conditions []conditionInput, conditionType string, job database.Job, err error) error {
	if err != nil || job.ID == 0 {
		return err
	}
	for _, condition := range conditions {
		if condition.ConditionType != conditionType || condition.UnitKey == "" {
			continue
		}
		return s.db.WithContext(ctx).Model(&database.IngestCondition{}).
			Where("unit_key = ? AND condition_type = ?", condition.UnitKey, condition.ConditionType).
			Update("job_id", job.ID).Error
	}
	return nil
}

func conditionHasStatus(conditions []conditionInput, conditionType string, statuses ...string) bool {
	statusSet := make(map[string]struct{}, len(statuses))
	for _, status := range statuses {
		statusSet[status] = struct{}{}
	}
	for _, condition := range conditions {
		if condition.ConditionType != conditionType {
			continue
		}
		_, ok := statusSet[condition.Status]
		return ok
	}
	return false
}

func shouldDispatchStage(reason string, conditionType string) bool {
	reason = strings.TrimSpace(reason)
	if !strings.HasPrefix(reason, "admin_retry_") && !strings.HasPrefix(reason, "retry_due_") && reason != "projection_refresh" {
		return true
	}
	suffix := strings.TrimPrefix(strings.TrimPrefix(reason, "admin_retry_"), "retry_due_")
	if reason == "projection_refresh" {
		suffix = ConditionProjectionCurrent
	}
	if suffix == "projection" {
		suffix = ConditionProjectionCurrent
	}
	return suffix == conditionType
}

func shouldDispatchCondition(reason string, conditions []conditionInput, conditionType string) bool {
	if !shouldDispatchStage(reason, conditionType) {
		return false
	}
	if conditionHasStatus(conditions, conditionType, ConditionStatusPending) {
		return true
	}
	reason = strings.TrimSpace(reason)
	if strings.HasPrefix(reason, "admin_retry_") || strings.HasPrefix(reason, "retry_due_") {
		return conditionHasStatus(conditions, conditionType, ConditionStatusFailed)
	}
	return false
}

func targetIDs(targets []database.CatalogItem) []uint {
	ids := make([]uint, 0, len(targets))
	for _, target := range targets {
		ids = append(ids, target.ID)
	}
	return ids
}

func normalizeUintIDs(ids []uint) []uint {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func (s *Service) inventoryFileConditions(tx *gorm.DB, file database.InventoryFile, unitKey string, itemIDs []uint, assetIDs []uint, targets []database.CatalogItem) []conditionInput {
	fileID := file.ID
	base := conditionInput{UnitKey: unitKey, LibraryID: file.LibraryID, InventoryFileID: &fileID}
	conditions := make([]conditionInput, 0, 6)
	if file.DeletedAt != nil || strings.TrimSpace(file.Status) == "missing" {
		projectionStatus := ConditionStatusPending
		projectionReason := "missing_visibility"
		projectionMessage := "Projection must converge with missing media state"
		projectionSeverity := SeverityWarning
		if strings.TrimSpace(file.ContentClass) != "video" {
			projectionStatus = ConditionStatusSkipped
			projectionReason = "unsupported_content_class"
			projectionMessage = "Projection is not required"
			projectionSeverity = SeverityInfo
		}
		conditions = append(conditions,
			base.with(ConditionVisible, ConditionStatusFalse, "missing", "Source file is missing", SeverityWarning),
			base.with(ConditionMaterialized, ConditionStatusFalse, "missing", "Materialization is blocked because the source file is missing", SeverityWarning),
			base.with(ConditionProbed, ConditionStatusSkipped, "missing", "Probe is skipped because the source file is missing", SeverityInfo),
			base.with(ConditionMetadataMatched, ConditionStatusSkipped, "missing", "Metadata matching is skipped because the source file is missing", SeverityInfo),
			base.with(ConditionProjectionCurrent, projectionStatus, projectionReason, projectionMessage, projectionSeverity),
			base.with(ConditionReviewRequired, ConditionStatusFalse, "not_required", "No review is required for missing media", SeverityInfo),
		)
		return conditions
	}
	if strings.TrimSpace(file.ContentClass) != "video" {
		conditions = append(conditions,
			base.with(ConditionVisible, ConditionStatusSkipped, "unsupported_content_class", "File is not video content", SeverityInfo),
			base.with(ConditionMaterialized, ConditionStatusSkipped, "unsupported_content_class", "Catalog materialization is skipped", SeverityInfo),
			base.with(ConditionProbed, ConditionStatusSkipped, "unsupported_content_class", "Probe is skipped", SeverityInfo),
			base.with(ConditionMetadataMatched, ConditionStatusSkipped, "unsupported_content_class", "Metadata matching is skipped", SeverityInfo),
			base.with(ConditionProjectionCurrent, ConditionStatusSkipped, "unsupported_content_class", "Projection is not required", SeverityInfo),
			base.with(ConditionReviewRequired, ConditionStatusFalse, "not_required", "No review is required", SeverityInfo),
		)
		return conditions
	}
	conditions = append(conditions, base.with(ConditionVisible, ConditionStatusTrue, "available", "Discovered media is visible", SeverityInfo))
	if len(itemIDs) == 0 {
		conditions = append(conditions,
			base.with(ConditionMaterialized, ConditionStatusPending, "awaiting_materialization", "Media is waiting for catalog materialization", SeverityInfo),
			base.with(ConditionProbed, ConditionStatusPending, "awaiting_materialization", "Media probe will run after materialization", SeverityInfo),
			base.with(ConditionMetadataMatched, ConditionStatusPending, "awaiting_materialization", "Metadata matching will run after materialization", SeverityInfo),
			base.with(ConditionProjectionCurrent, ConditionStatusSkipped, "inventory_visible", "Discovered inventory entry does not require catalog projection yet", SeverityInfo),
		)
	} else {
		conditions = append(conditions,
			base.with(ConditionMaterialized, ConditionStatusTrue, "linked", "Media is linked to catalog", SeverityInfo),
			s.probeCondition(tx, base, assetIDs),
			s.metadataCondition(tx, base, targets),
			s.projectionCondition(tx, base, targets),
		)
	}
	conditions = append(conditions, s.reviewCondition(tx, base, file.ID, targets))
	return conditions
}

func (s *Service) catalogItemConditions(tx *gorm.DB, item database.CatalogItem, unitKey string, targets []database.CatalogItem) []conditionInput {
	itemID := item.ID
	base := conditionInput{UnitKey: unitKey, LibraryID: item.LibraryID, CatalogItemID: &itemID}
	conditions := []conditionInput{
		base.with(ConditionVisible, visibleStatus(item.AvailabilityStatus), visibleReason(item.AvailabilityStatus), visibleMessage(item.AvailabilityStatus), visibleSeverity(item.AvailabilityStatus)),
		base.with(ConditionMaterialized, ConditionStatusTrue, "catalog_item_exists", "Catalog item exists", SeverityInfo),
		s.metadataCondition(tx, base, targets),
		s.projectionCondition(tx, base, targets),
	}
	conditions = append(conditions, s.reviewCondition(tx, base, 0, targets))
	return conditions
}

func (s *Service) probeCondition(tx *gorm.DB, base conditionInput, assetIDs []uint) conditionInput {
	if len(assetIDs) == 0 {
		return base.with(ConditionProbed, ConditionStatusPending, "awaiting_asset", "Media probe is waiting for an asset", SeverityInfo)
	}
	var assets []database.MediaAsset
	if err := tx.Where("id IN ? AND deleted_at IS NULL", assetIDs).Find(&assets).Error; err != nil || len(assets) == 0 {
		return base.with(ConditionProbed, ConditionStatusUnknown, "asset_lookup_failed", "Probe status cannot be determined", SeverityWarning)
	}
	ready := 0
	unavailable := 0
	failed := 0
	for _, asset := range assets {
		switch strings.TrimSpace(asset.ProbeStatus) {
		case "ready":
			ready++
		case "unavailable":
			unavailable++
		case "error":
			failed++
		}
	}
	switch {
	case failed > 0:
		return base.with(ConditionProbed, ConditionStatusFailed, "probe_failed", "Media probe failed", SeverityError)
	case ready == len(assets):
		return base.with(ConditionProbed, ConditionStatusTrue, "ready", "Media probe completed", SeverityInfo)
	case ready+unavailable == len(assets):
		return base.with(ConditionProbed, ConditionStatusSkipped, "unavailable", "Media probe is unavailable for this source", SeverityWarning)
	default:
		return base.with(ConditionProbed, ConditionStatusPending, "probe_pending", "Media probe is pending", SeverityInfo)
	}
}

func (s *Service) metadataCondition(tx *gorm.DB, base conditionInput, targets []database.CatalogItem) conditionInput {
	if len(targets) == 0 {
		return base.with(ConditionMetadataMatched, ConditionStatusPending, "awaiting_materialization", "Metadata matching is waiting for catalog materialization", SeverityInfo)
	}
	matched := 0
	skipped := 0
	pending := 0
	for _, item := range targets {
		switch strings.TrimSpace(item.GovernanceStatus) {
		case "matched", "manual", "locked":
			matched++
		case "needs_review":
			return base.with(ConditionMetadataMatched, ConditionStatusReviewRequired, "needs_review", "Metadata match requires review", SeverityWarning)
		case "unmatched":
			if !s.hasMetadataNoCandidateOperation(tx, item.ID) || !s.hasRunnableMetadataSearchStrategy(tx, item.LibraryID) {
				skipped++
				continue
			}
			return base.with(ConditionMetadataMatched, ConditionStatusFalse, "no_candidate", "No acceptable metadata candidate was found", SeverityWarning)
		case "":
			pending++
		case "pending":
			pending++
		default:
			skipped++
		}
	}
	switch {
	case matched == len(targets):
		return base.with(ConditionMetadataMatched, ConditionStatusTrue, "matched", "Metadata is matched", SeverityInfo)
	case pending > 0:
		return base.with(ConditionMetadataMatched, ConditionStatusPending, "pending", "Metadata matching is pending", SeverityInfo)
	case skipped == len(targets):
		return base.with(ConditionMetadataMatched, ConditionStatusSkipped, "not_required", "Metadata matching is not required", SeverityInfo)
	default:
		return base.with(ConditionMetadataMatched, ConditionStatusUnknown, "unknown", "Metadata state is unknown", SeverityWarning)
	}
}

func (s *Service) projectionCondition(tx *gorm.DB, base conditionInput, targets []database.CatalogItem) conditionInput {
	if len(targets) == 0 {
		return base.with(ConditionProjectionCurrent, ConditionStatusSkipped, "inventory_visible", "Catalog projection is not required yet", SeverityInfo)
	}
	ids := make([]uint, 0, len(targets))
	for _, target := range targets {
		ids = append(ids, target.ID)
	}
	var count int64
	if err := tx.Model(&database.CatalogSearchDocument{}).Where("item_id IN ?", ids).Count(&count).Error; err != nil {
		return base.with(ConditionProjectionCurrent, ConditionStatusUnknown, "projection_lookup_failed", "Projection state cannot be determined", SeverityWarning)
	}
	if count == int64(len(ids)) {
		return base.with(ConditionProjectionCurrent, ConditionStatusTrue, "current", "Catalog projection is current", SeverityInfo)
	}
	return base.with(ConditionProjectionCurrent, ConditionStatusPending, "projection_pending", "Catalog projection refresh is pending", SeverityInfo)
}

func (s *Service) reviewCondition(tx *gorm.DB, base conditionInput, fileID uint, targets []database.CatalogItem) conditionInput {
	for _, item := range targets {
		itemID := item.ID
		itemBase := base
		itemBase.CatalogItemID = &itemID
		switch strings.TrimSpace(item.GovernanceStatus) {
		case "needs_review":
			return itemBase.with(ConditionReviewRequired, ConditionStatusReviewRequired, "metadata_needs_review", "Metadata match requires review", SeverityWarning)
		case "unmatched":
			if !s.hasMetadataNoCandidateOperation(tx, item.ID) || !s.hasRunnableMetadataSearchStrategy(tx, item.LibraryID) {
				continue
			}
			return itemBase.with(ConditionReviewRequired, ConditionStatusReviewRequired, "metadata_no_candidate", "No acceptable metadata candidate was found", SeverityWarning)
		}
	}
	if fileID != 0 {
		var count int64
		if err := tx.Model(&database.ClassificationDecision{}).
			Where("classification_decisions.inventory_file_id = ? AND classification_decisions.status IN ?", fileID, []string{"provisional", "review_required"}).
			Where("NOT EXISTS (?)", tx.Model(&database.CatalogItem{}).
				Select("1").
				Where("catalog_items.id = classification_decisions.item_id").
				Where("catalog_items.deleted_at IS NULL").
				Where("catalog_items.governance_status IN ?", []string{"matched", "manual", "locked"})).
			Count(&count).Error; err == nil && count > 0 {
			return base.with(ConditionReviewRequired, ConditionStatusReviewRequired, "classification_needs_review", "Classification requires review", SeverityWarning)
		}
	}
	return base.with(ConditionReviewRequired, ConditionStatusFalse, "not_required", "No review is required", SeverityInfo)
}

func (s *Service) hasMetadataNoCandidateOperation(tx *gorm.DB, itemID uint) bool {
	if itemID == 0 {
		return false
	}
	var count int64
	if err := tx.Model(&database.MetadataOperation{}).
		Where("target_item_id = ? AND operation = ? AND status = ?", itemID, "match", "no_candidate").
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func (s *Service) hasRunnableMetadataSearchStrategy(tx *gorm.DB, libraryID uint) bool {
	if libraryID == 0 {
		return false
	}
	var strategy database.LibraryMetadataStrategy
	if err := tx.Where("library_id = ?", libraryID).First(&strategy).Error; err != nil {
		return errors.Is(err, gorm.ErrRecordNotFound)
	}
	providerIDs := uintListFromJSON(strategy.SearchProvidersJSON)
	if len(providerIDs) == 0 {
		return false
	}
	var providers []database.MetadataProviderInstance
	if err := tx.Where("id IN ?", providerIDs).Find(&providers).Error; err != nil {
		return false
	}
	now := time.Now().UTC()
	for _, provider := range providers {
		if metadataSearchProviderRunnable(provider, now) {
			return true
		}
	}
	return false
}

func metadataSearchProviderRunnable(provider database.MetadataProviderInstance, now time.Time) bool {
	if !provider.Enabled || !metadataProviderAvailabilityActive(provider, now) {
		return false
	}
	switch strings.TrimSpace(provider.ProviderType) {
	case database.MetadataProviderTypeTMDB:
		return metadataProviderConfigValue(provider.ConfigJSON, "tmdb_api_key") != ""
	case database.MetadataProviderTypeMetaTube:
		return metadataProviderConfigValue(provider.ConfigJSON, "metatube_base_url") != ""
	default:
		return false
	}
}

func metadataProviderAvailabilityActive(provider database.MetadataProviderInstance, now time.Time) bool {
	status := strings.TrimSpace(provider.AvailabilityStatus)
	if status == "" || status == database.MetadataProviderAvailabilityAvailable {
		return provider.CooldownUntil == nil || provider.CooldownUntil.Before(now)
	}
	if status == database.MetadataProviderAvailabilityCooldown {
		return provider.CooldownUntil != nil && provider.CooldownUntil.Before(now)
	}
	return false
}

func metadataProviderConfigValue(raw string, key string) string {
	var values map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &values); err != nil {
		return ""
	}
	return strings.TrimSpace(values[key])
}

func uintListFromJSON(raw string) []uint {
	var values []uint
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &values); err != nil {
		return []uint{}
	}
	return values
}

func (s *Service) setCondition(ctx context.Context, tx *gorm.DB, input conditionInput) error {
	if input.UnitKey == "" || input.LibraryID == 0 || input.ConditionType == "" {
		return errors.New("condition requires unit key, library id, and type")
	}
	now := s.now()
	var existing database.IngestCondition
	err := tx.WithContext(ctx).Where("unit_key = ? AND condition_type = ?", input.UnitKey, input.ConditionType).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	changed := errors.Is(err, gorm.ErrRecordNotFound) || existing.Status != input.Status || existing.Reason != input.Reason || existing.Message != input.Message || existing.Severity != input.Severity
	transitionAt := existing.LastTransitionAt
	if changed {
		transitionAt = &now
	}
	condition := database.IngestCondition{
		ID:                  existing.ID,
		UnitKey:             input.UnitKey,
		LibraryID:           input.LibraryID,
		InventoryFileID:     input.InventoryFileID,
		CatalogItemID:       input.CatalogItemID,
		ConditionType:       input.ConditionType,
		Status:              defaultString(input.Status, ConditionStatusUnknown),
		Reason:              strings.TrimSpace(input.Reason),
		Message:             strings.TrimSpace(input.Message),
		Severity:            defaultString(input.Severity, SeverityInfo),
		Attempts:            input.Attempts,
		JobID:               input.JobID,
		MetadataOperationID: input.MetadataOperationID,
		ProviderInstanceID:  input.ProviderInstanceID,
		DetailsJSON:         strings.TrimSpace(input.DetailsJSON),
		LastTransitionAt:    transitionAt,
		NextAttemptAt:       input.NextAttemptAt,
		StaleAfter:          input.StaleAfter,
	}
	if condition.ID == 0 {
		if err := tx.WithContext(ctx).Create(&condition).Error; err != nil {
			return err
		}
	} else if err := tx.WithContext(ctx).Save(&condition).Error; err != nil {
		return err
	}
	if !changed {
		return nil
	}
	expires := now.Add(DefaultEventRetention)
	event := database.IngestEvent{UnitKey: condition.UnitKey, LibraryID: condition.LibraryID, InventoryFileID: condition.InventoryFileID, CatalogItemID: condition.CatalogItemID, ConditionID: &condition.ID, ConditionType: condition.ConditionType, EventType: EventConditionChanged, Status: condition.Status, Reason: condition.Reason, Message: condition.Message, JobID: condition.JobID, MetadataOperationID: condition.MetadataOperationID, ProviderInstanceID: condition.ProviderInstanceID, DetailsJSON: condition.DetailsJSON, ExpiresAt: &expires}
	return tx.WithContext(ctx).Create(&event).Error
}

func (s *Service) markDirtyCompleted(ctx context.Context, unitID uint) error {
	return s.db.WithContext(ctx).Model(&database.IngestDirtyUnit{}).Where("id = ?", unitID).Updates(map[string]any{"status": DirtyStatusCompleted, "claimed_at": nil, "last_error": ""}).Error
}

func (s *Service) markDirtyFailed(ctx context.Context, unitID uint, err error) error {
	message := "reconcile failed"
	if err != nil {
		message = err.Error()
	}
	return s.db.WithContext(ctx).Model(&database.IngestDirtyUnit{}).Where("id = ?", unitID).Updates(map[string]any{"status": DirtyStatusFailed, "claimed_at": nil, "last_error": message}).Error
}

func linkedCatalogItemIDs(tx *gorm.DB, fileID uint) ([]uint, error) {
	var ids []uint
	err := tx.Table("asset_items").Distinct("asset_items.item_id").
		Joins("JOIN asset_files ON asset_files.asset_id = asset_items.asset_id").
		Joins("JOIN media_assets ON media_assets.id = asset_items.asset_id AND media_assets.deleted_at IS NULL").
		Joins("JOIN catalog_items ON catalog_items.id = asset_items.item_id AND catalog_items.deleted_at IS NULL").
		Where("asset_files.file_id = ?", fileID).
		Order("asset_items.item_id asc").
		Pluck("asset_items.item_id", &ids).Error
	return ids, err
}

func linkedAssetIDs(tx *gorm.DB, fileID uint) ([]uint, error) {
	var ids []uint
	err := tx.Model(&database.AssetFile{}).Distinct("asset_id").Where("file_id = ?", fileID).Order("asset_id asc").Pluck("asset_id", &ids).Error
	return ids, err
}

func metadataTargetsForItems(tx *gorm.DB, itemIDs []uint) ([]database.CatalogItem, error) {
	if len(itemIDs) == 0 {
		return nil, nil
	}
	var items []database.CatalogItem
	if err := tx.Where("id IN ? AND deleted_at IS NULL", itemIDs).Find(&items).Error; err != nil {
		return nil, err
	}
	targetIDs := make(map[uint]struct{}, len(items))
	for _, item := range items {
		switch strings.TrimSpace(item.Type) {
		case "movie", "series":
			targetIDs[item.ID] = struct{}{}
		case "season", "episode":
			if item.RootID != nil && *item.RootID != 0 {
				targetIDs[*item.RootID] = struct{}{}
			}
		}
	}
	if len(targetIDs) == 0 {
		return nil, nil
	}
	ids := make([]uint, 0, len(targetIDs))
	for id := range targetIDs {
		ids = append(ids, id)
	}
	var targets []database.CatalogItem
	if err := tx.Where("id IN ? AND deleted_at IS NULL", ids).Order("id asc").Find(&targets).Error; err != nil {
		return nil, err
	}
	return targets, nil
}

func inventoryFileUnitKey(fileID uint) string {
	return fmt.Sprintf("inventory_file:%d", fileID)
}

func catalogItemUnitKey(itemID uint) string {
	return fmt.Sprintf("catalog_item:%d", itemID)
}

func (input conditionInput) with(conditionType string, status string, reason string, message string, severity string) conditionInput {
	input.ConditionType = conditionType
	input.Status = status
	input.Reason = reason
	input.Message = message
	input.Severity = severity
	return input
}

func normalizeReason(reason string) string {
	if reason = strings.TrimSpace(reason); reason != "" {
		return reason
	}
	return "unspecified"
}

func defaultString(value string, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return fallback
}

func visibleStatus(availability string) string {
	if strings.TrimSpace(availability) == "available" {
		return ConditionStatusTrue
	}
	return ConditionStatusFalse
}

func visibleReason(availability string) string {
	if strings.TrimSpace(availability) == "available" {
		return "available"
	}
	return "not_available"
}

func visibleMessage(availability string) string {
	if strings.TrimSpace(availability) == "available" {
		return "Catalog media is visible"
	}
	return "Catalog media is not available"
}

func visibleSeverity(availability string) string {
	if strings.TrimSpace(availability) == "available" {
		return SeverityInfo
	}
	return SeverityWarning
}
