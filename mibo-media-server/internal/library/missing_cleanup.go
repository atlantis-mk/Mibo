package library

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/settings"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
)

type MissingCleanupResult struct {
	FilesDeleted         int
	AssetsDeleted        int
	CatalogItemsDeleted  int
	DependentRowsDeleted int
}

type MissingMediaCleanupPayload struct {
	ScopeKind string `json:"scope_kind"`
	LibraryID *uint  `json:"library_id,omitempty"`
	RootPath  string `json:"root_path,omitempty"`
}

func (s *Service) QueueMissingMediaCleanup(ctx context.Context, payload MissingMediaCleanupPayload) (database.Job, error) {
	scope := strings.TrimSpace(payload.ScopeKind)
	if scope == "" {
		scope = "global"
	}
	jobKey := fmt.Sprintf("missing-media-cleanup:%s", scope)
	if payload.LibraryID != nil && *payload.LibraryID != 0 {
		jobKey = fmt.Sprintf("%s:%d", jobKey, *payload.LibraryID)
	}
	rootPath := strings.TrimSpace(payload.RootPath)
	if rootPath != "" {
		jobKey = fmt.Sprintf("%s:%s", jobKey, rootPath)
	}
	if s.workflow == nil {
		return database.Job{}, fmt.Errorf("workflow service unavailable")
	}
	libraryID := uint(0)
	if payload.LibraryID != nil {
		libraryID = *payload.LibraryID
	}
	if libraryID == 0 {
		var first database.Library
		if err := s.db.WithContext(ctx).Where("deleted_at IS NULL").Order("id asc").First(&first).Error; err == nil {
			libraryID = first.ID
		}
	}
	if libraryID == 0 {
		return database.Job{}, fmt.Errorf("no library available for missing cleanup workflow")
	}
	run, reused, err := s.workflow.CreateOrReuseRun(ctx, workflow.CreateRunInput{
		RunKey:    jobKey,
		LibraryID: libraryID,
		Reason:    WorkflowReasonMissingCleanup,
		Priority:  1,
		ScopeKey:  fmt.Sprintf("cleanup:%s", scope),
		Payload:   MissingMediaCleanupPayload{ScopeKind: scope, LibraryID: payload.LibraryID, RootPath: rootPath},
	})
	if err != nil || reused {
		return workflowRunCompatibilityJob(run), err
	}
	_, err = s.workflow.CreateTask(ctx, run, workflow.CreateTaskInput{
		TaskKey:   fmt.Sprintf("run:%d:cleanup-missing", run.ID),
		TaskType:  workflow.TaskTypeCleanupMissing,
		Stage:     workflow.StageCleanup,
		Priority:  1,
		ScopeKey:  run.ScopeKey,
		Payload:   MissingMediaCleanupPayload{ScopeKind: scope, LibraryID: payload.LibraryID, RootPath: rootPath},
		Resources: workflow.DefaultTaskTypeDefinitions()[workflow.TaskTypeCleanupMissing].Resources,
	})
	return workflowRunCompatibilityJob(run), err
}

func (s *Service) RunMissingMediaCleanupJob(ctx context.Context, payload MissingMediaCleanupPayload) (ScheduledJobResult, error) {
	if strings.TrimSpace(payload.ScopeKind) == "library" {
		if payload.LibraryID == nil || *payload.LibraryID == 0 {
			return ScheduledJobResult{}, fmt.Errorf("library cleanup requires library_id")
		}
		return s.runMissingMediaCleanupForLibraries(ctx, []uint{*payload.LibraryID}, strings.TrimSpace(payload.RootPath))
	}
	libraries, err := s.ListActiveLibraries(ctx)
	if err != nil {
		return ScheduledJobResult{}, err
	}
	libraryIDs := make([]uint, 0, len(libraries))
	for _, libraryRecord := range libraries {
		libraryIDs = append(libraryIDs, libraryRecord.ID)
	}
	return s.runMissingMediaCleanupForLibraries(ctx, libraryIDs, strings.TrimSpace(payload.RootPath))
}

func (s *Service) runMissingMediaCleanupForLibraries(ctx context.Context, libraryIDs []uint, rootPath string) (ScheduledJobResult, error) {
	result := ScheduledJobResult{LibrariesProcessed: len(libraryIDs)}
	for _, libraryID := range libraryIDs {
		config, err := s.EffectiveLibraryConfig(ctx, libraryID)
		if err != nil {
			return ScheduledJobResult{}, err
		}
		paths := config.Paths
		if strings.TrimSpace(rootPath) != "" {
			paths = []database.LibraryPath{{RootPath: rootPath}}
		}
		for _, pathRecord := range paths {
			cleanup, err := s.CleanupMissingMedia(ctx, config.Library.ID, pathRecord.RootPath)
			if err != nil {
				return ScheduledJobResult{}, err
			}
			result.FilesDeleted += cleanup.FilesDeleted
			result.AssetsDeleted += cleanup.AssetsDeleted
			result.CatalogItemsDeleted += cleanup.CatalogItemsDeleted
			result.DependentRowsDeleted += cleanup.DependentRowsDeleted
		}
	}
	result.Summary = fmt.Sprintf("missing cleanup completed for %d libraries; deleted %d files, %d assets, %d catalog items, %d dependent rows", result.LibrariesProcessed, result.FilesDeleted, result.AssetsDeleted, result.CatalogItemsDeleted, result.DependentRowsDeleted)
	return result, nil
}

func (s *Service) CleanupMissingMedia(ctx context.Context, libraryID uint, rootPath string) (MissingCleanupResult, error) {
	if libraryID == 0 {
		return MissingCleanupResult{}, nil
	}
	cleanupSettings, err := settings.ResolveCleanupSettings(ctx, s.db, s.cfg.Cleanup)
	if err != nil {
		return MissingCleanupResult{}, err
	}
	if !cleanupSettings.MissingCleanupEnabled {
		return MissingCleanupResult{}, nil
	}
	retention := time.Duration(cleanupSettings.MissingRetentionSeconds) * time.Second
	if retention < 0 {
		return MissingCleanupResult{}, nil
	}
	cutoff := time.Now().UTC().Add(-retention)
	batchSize := cleanupSettings.MissingCleanupBatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	var result MissingCleanupResult
	for {
		fileIDs, err := s.missingCleanupCandidateFileIDs(ctx, libraryID, rootPath, cutoff, batchSize)
		if err != nil || len(fileIDs) == 0 {
			return result, err
		}
		batchResult, err := s.deleteMissingMediaBatch(ctx, libraryID, rootPath, fileIDs)
		if err != nil {
			return result, err
		}
		result.FilesDeleted += batchResult.FilesDeleted
		result.AssetsDeleted += batchResult.AssetsDeleted
		result.CatalogItemsDeleted += batchResult.CatalogItemsDeleted
		result.DependentRowsDeleted += batchResult.DependentRowsDeleted
	}
}

func (s *Service) missingCleanupCandidateFileIDs(ctx context.Context, libraryID uint, rootPath string, cutoff time.Time, limit int) ([]uint, error) {
	var ids []uint
	query := s.db.WithContext(ctx).
		Model(&database.InventoryFile{}).
		Where("library_id = ? AND status = ? AND missing_since IS NOT NULL AND missing_since <= ?", libraryID, inventory.FileStatusMissing, cutoff).
		Order("missing_since asc, id asc").
		Limit(limit)
	query = applyScopedPathFilter(query, "storage_path", rootPath)
	return ids, query.Pluck("id", &ids).Error
}

func (s *Service) deleteMissingMediaBatch(ctx context.Context, libraryID uint, rootPath string, fileIDs []uint) (MissingCleanupResult, error) {
	var result MissingCleanupResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		assetIDs, err := deletableMissingAssetIDs(ctx, tx, fileIDs)
		if err != nil {
			return err
		}
		itemIDs, err := deletableMissingCatalogItemIDs(ctx, tx, libraryID, assetIDs)
		if err != nil {
			return err
		}

		dependent, err := hardDeleteMissingDependents(ctx, tx, libraryID, fileIDs, assetIDs, itemIDs)
		if err != nil {
			return err
		}
		result.DependentRowsDeleted += dependent

		deleted, err := deleteRows(ctx, tx.Where("id IN ?", fileIDs).Delete(&database.InventoryFile{}))
		if err != nil {
			return err
		}
		result.FilesDeleted += deleted
		if len(assetIDs) > 0 {
			deleted, err = deleteRows(ctx, tx.Where("id IN ?", assetIDs).Delete(&database.MediaAsset{}))
			if err != nil {
				return err
			}
			result.AssetsDeleted += deleted
		}
		if len(itemIDs) > 0 {
			deleted, err = deleteRows(ctx, tx.Where("id IN ?", itemIDs).Delete(&database.CatalogItem{}))
			if err != nil {
				return err
			}
			result.CatalogItemsDeleted += deleted
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	if result.CatalogItemsDeleted > 0 {
		if _, err := s.QueueCatalogLibraryProjectionRefresh(ctx, libraryID, rootPath); err != nil {
			return result, err
		}
	}
	return result, nil
}

func deletableMissingAssetIDs(ctx context.Context, tx *gorm.DB, fileIDs []uint) ([]uint, error) {
	if len(fileIDs) == 0 {
		return nil, nil
	}
	var assetIDs []uint
	if err := tx.WithContext(ctx).
		Table("asset_files").
		Distinct("asset_id").
		Where("file_id IN ?", fileIDs).
		Pluck("asset_id", &assetIDs).Error; err != nil {
		return nil, err
	}
	if len(assetIDs) == 0 {
		return nil, nil
	}
	var keepAssetIDs []uint
	if err := tx.WithContext(ctx).
		Table("asset_files").
		Distinct("asset_files.asset_id").
		Joins("JOIN inventory_files ON inventory_files.id = asset_files.file_id").
		Where("asset_files.asset_id IN ? AND inventory_files.status = ?", assetIDs, inventory.FileStatusAvailable).
		Pluck("asset_files.asset_id", &keepAssetIDs).Error; err != nil {
		return nil, err
	}
	keep := uintSet(keepAssetIDs)
	result := make([]uint, 0, len(assetIDs))
	for _, id := range assetIDs {
		if _, ok := keep[id]; !ok {
			result = append(result, id)
		}
	}
	return result, nil
}

func deletableMissingCatalogItemIDs(ctx context.Context, tx *gorm.DB, libraryID uint, assetIDs []uint) ([]uint, error) {
	if len(assetIDs) == 0 {
		return nil, nil
	}
	var seedIDs []uint
	if err := tx.WithContext(ctx).
		Table("asset_items").
		Distinct("item_id").
		Where("asset_id IN ?", assetIDs).
		Pluck("item_id", &seedIDs).Error; err != nil {
		return nil, err
	}
	if len(seedIDs) == 0 {
		return nil, nil
	}
	selected := make(map[uint]struct{}, len(seedIDs))
	for _, itemID := range seedIDs {
		canDelete, err := canDeleteMissingCatalogItem(ctx, tx, itemID, selected)
		if err != nil {
			return nil, err
		}
		if canDelete {
			selected[itemID] = struct{}{}
		}
	}
	if len(selected) == 0 {
		return nil, nil
	}
	for {
		var parents []database.CatalogItem
		ids := setValues(selected)
		if err := tx.WithContext(ctx).
			Select("id", "parent_id", "availability_status").
			Where("library_id = ? AND id IN ?", libraryID, ids).
			Find(&parents).Error; err != nil {
			return nil, err
		}
		changed := false
		var parentIDs []uint
		for _, item := range parents {
			if item.ParentID != nil {
				parentIDs = append(parentIDs, *item.ParentID)
			}
		}
		for _, parentID := range parentIDs {
			if _, ok := selected[parentID]; ok {
				continue
			}
			canDelete, err := canDeleteMissingCatalogItem(ctx, tx, parentID, selected)
			if err != nil {
				return nil, err
			}
			if canDelete {
				selected[parentID] = struct{}{}
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return setValues(selected), nil
}

func canDeleteMissingCatalogItem(ctx context.Context, tx *gorm.DB, itemID uint, selected map[uint]struct{}) (bool, error) {
	var item database.CatalogItem
	if err := tx.WithContext(ctx).Select("id", "availability_status").First(&item, itemID).Error; err != nil {
		return false, err
	}
	if strings.TrimSpace(item.AvailabilityStatus) != catalog.AvailabilityMissing {
		return false, nil
	}
	var children []database.CatalogItem
	if err := tx.WithContext(ctx).Select("id", "availability_status").Where("parent_id = ?", itemID).Find(&children).Error; err != nil {
		return false, err
	}
	for _, child := range children {
		if _, ok := selected[child.ID]; !ok {
			return false, nil
		}
	}
	var availableAssets int64
	if err := tx.WithContext(ctx).
		Table("asset_items").
		Joins("JOIN media_assets ON media_assets.id = asset_items.asset_id").
		Where("asset_items.item_id = ? AND media_assets.status = ?", itemID, inventory.AssetStatusAvailable).
		Count(&availableAssets).Error; err != nil {
		return false, err
	}
	return availableAssets == 0, nil
}

func hardDeleteMissingDependents(ctx context.Context, tx *gorm.DB, libraryID uint, fileIDs []uint, assetIDs []uint, itemIDs []uint) (int, error) {
	deleted := 0
	deleteAndCount := func(query *gorm.DB) error {
		count, err := deleteRows(ctx, query)
		deleted += count
		return err
	}
	if len(fileIDs) > 0 {
		if err := deleteAndCount(tx.Where("file_id IN ?", fileIDs).Delete(&database.MediaStream{})); err != nil {
			return deleted, err
		}
		if err := deleteAndCount(tx.Where("file_id IN ?", fileIDs).Delete(&database.AssetFile{})); err != nil {
			return deleted, err
		}
		var files []database.InventoryFile
		if err := tx.WithContext(ctx).Select("library_id", "storage_provider", "storage_path", "stable_identity_key").Where("id IN ?", fileIDs).Find(&files).Error; err != nil {
			return deleted, err
		}
		for _, file := range files {
			query := tx.Where("library_id = ? AND storage_provider = ?", file.LibraryID, file.StorageProvider)
			if strings.TrimSpace(file.StableIdentityKey) != "" {
				query = query.Where("stable_identity_key = ?", file.StableIdentityKey)
			} else {
				query = query.Where("storage_path = ?", file.StoragePath)
			}
			if err := deleteAndCount(query.Delete(&database.ScanExclusion{})); err != nil {
				return deleted, err
			}
		}
	}
	if len(assetIDs) > 0 {
		if err := deleteAndCount(tx.Where("asset_id IN ?", assetIDs).Delete(&database.AssetItem{})); err != nil {
			return deleted, err
		}
		if err := deleteAndCount(tx.Where("asset_id IN ?", assetIDs).Delete(&database.AssetFile{})); err != nil {
			return deleted, err
		}
		if err := deleteAndCount(tx.Where("asset_id IN ?", assetIDs).Delete(&database.UserItemData{})); err != nil {
			return deleted, err
		}
	}
	if len(itemIDs) > 0 {
		if err := deleteAndCount(tx.Where("item_id IN ?", itemIDs).Delete(&database.UserItemData{})); err != nil {
			return deleted, err
		}
		if err := deleteAndCount(tx.Where("item_id IN ?", itemIDs).Delete(&database.ItemRollup{})); err != nil {
			return deleted, err
		}
		if err := deleteAndCount(tx.Where("item_id IN ?", itemIDs).Delete(&database.CatalogSearchDocument{})); err != nil {
			return deleted, err
		}
		if err := deleteAndCount(tx.Where("item_id IN ?", itemIDs).Delete(&database.ItemImage{})); err != nil {
			return deleted, err
		}
		if err := deleteAndCount(tx.Where("item_id IN ?", itemIDs).Delete(&database.ItemPerson{})); err != nil {
			return deleted, err
		}
		if err := deleteAndCount(tx.Where("item_id IN ?", itemIDs).Delete(&database.ItemTag{})); err != nil {
			return deleted, err
		}
		if err := deleteAndCount(tx.Where("item_id IN ?", itemIDs).Delete(&database.CatalogExternalID{})); err != nil {
			return deleted, err
		}
		if err := deleteAndCount(tx.Where("item_id IN ?", itemIDs).Delete(&database.CatalogIdentity{})); err != nil {
			return deleted, err
		}
		if err := deleteAndCount(tx.Where("item_id IN ?", itemIDs).Delete(&database.MetadataFieldState{})); err != nil {
			return deleted, err
		}
		if err := deleteAndCount(tx.Where("item_id IN ?", itemIDs).Delete(&database.MetadataSource{})); err != nil {
			return deleted, err
		}
		if err := deleteAndCount(tx.Where("origin_item_id IN ? OR target_item_id IN ?", itemIDs, itemIDs).Delete(&database.MetadataOperation{})); err != nil {
			return deleted, err
		}
		if err := deleteAndCount(tx.Where("item_id IN ?", itemIDs).Delete(&database.AssetItem{})); err != nil {
			return deleted, err
		}
	}
	_ = libraryID
	return deleted, nil
}

func deleteRows(_ context.Context, query *gorm.DB) (int, error) {
	if query.Error != nil {
		return 0, query.Error
	}
	return int(query.RowsAffected), nil
}

func uintSet(ids []uint) map[uint]struct{} {
	result := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if id != 0 {
			result[id] = struct{}{}
		}
	}
	return result
}

func setValues(values map[uint]struct{}) []uint {
	result := make([]uint, 0, len(values))
	for id := range values {
		result = append(result, id)
	}
	return result
}
