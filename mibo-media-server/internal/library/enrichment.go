package library

import (
	"context"
	"fmt"
	"sort"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/workflow"
)

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
	mode      *scanMode
}

type CatalogPostMaterializeBatchPayload struct {
	LibraryID uint   `json:"library_id"`
	RootPath  string `json:"root_path,omitempty"`
	FileIDs   []uint `json:"file_ids,omitempty"`
	ItemIDs   []uint `json:"item_ids,omitempty"`
}

func (s *Service) QueueCatalogItemMatch(ctx context.Context, itemID uint) (database.Job, error) {
	if itemID == 0 {
		return database.Job{}, fmt.Errorf("catalog item id is required")
	}

	targetID, shouldQueue, err := s.catalogMatchTargetForQueue(ctx, itemID)
	if err != nil {
		return database.Job{}, err
	}
	if !shouldQueue {
		return database.Job{}, nil
	}
	if s.workflow != nil {
		var item database.CatalogItem
		if err := s.db.WithContext(ctx).First(&item, targetID).Error; err != nil {
			return database.Job{}, err
		}
		run, err := s.queueStandaloneWorkflowTask(ctx, item.LibraryID, item.Path, WorkflowReasonManualScan, workflow.TaskTypeMatchMetadata, workflow.StageMetadataMatch, fmt.Sprintf("match-item:%d", targetID), CatalogMatchBatchPayload{LibraryID: item.LibraryID, RootPath: item.Path, ItemIDs: []uint{targetID}})
		return workflowRunCompatibilityJob(run), err
	}
	return database.Job{}, fmt.Errorf("workflow service unavailable")
}

func (s *Service) QueueCatalogMatchBatch(ctx context.Context, libraryID uint, rootPath string, itemIDs []uint) (database.Job, error) {
	ids := normalizeUintIDs(itemIDs)
	if len(ids) == 0 {
		return database.Job{}, nil
	}
	if s.workflow != nil {
		run, err := s.queueStandaloneWorkflowTask(ctx, libraryID, rootPath, WorkflowReasonManualScan, workflow.TaskTypeMatchMetadata, workflow.StageMetadataMatch, fmt.Sprintf("match:%s", rootPath), CatalogMatchBatchPayload{LibraryID: libraryID, RootPath: rootPath, ItemIDs: ids})
		return workflowRunCompatibilityJob(run), err
	}
	return database.Job{}, fmt.Errorf("workflow service unavailable")
}

func (s *Service) QueueInventoryProbeBatch(ctx context.Context, libraryID uint, rootPath string, fileIDs []uint) (database.Job, error) {
	ids := normalizeUintIDs(fileIDs)
	if len(ids) == 0 {
		return database.Job{}, nil
	}
	if s.workflow != nil {
		run, err := s.queueStandaloneWorkflowTask(ctx, libraryID, rootPath, WorkflowReasonProbeInventory, workflow.TaskTypeProbeInventory, workflow.StageProbe, fmt.Sprintf("probe:%s", rootPath), InventoryProbeBatchPayload{LibraryID: libraryID, RootPath: rootPath, FileIDs: ids})
		return workflowRunCompatibilityJob(run), err
	}
	return database.Job{}, fmt.Errorf("workflow service unavailable")
}

func (s *Service) QueueCatalogMaterializeBatch(ctx context.Context, libraryID uint, rootPath string, fileIDs []uint) (database.Job, error) {
	ids := normalizeUintIDs(fileIDs)
	if len(ids) == 0 {
		return database.Job{}, nil
	}
	if s.workflow != nil {
		run, err := s.queueStandaloneWorkflowTask(ctx, libraryID, rootPath, WorkflowReasonManualScan, workflow.TaskTypeMaterializeCatalog, workflow.StageMaterialize, fmt.Sprintf("materialize:%s", rootPath), CatalogMaterializeBatchPayload{LibraryID: libraryID, RootPath: rootPath, FileIDs: ids})
		return workflowRunCompatibilityJob(run), err
	}
	return database.Job{}, fmt.Errorf("workflow service unavailable")
}

func (s *Service) QueueCatalogPostMaterializeBatch(ctx context.Context, libraryID uint, rootPath string, fileIDs []uint, itemIDs []uint) (database.Job, error) {
	normalizedFileIDs := normalizeUintIDs(fileIDs)
	normalizedItemIDs := normalizeUintIDs(itemIDs)
	if len(normalizedFileIDs) == 0 && len(normalizedItemIDs) == 0 {
		return database.Job{}, nil
	}
	if s.workflow != nil {
		payload := CatalogPostMaterializeBatchPayload{LibraryID: libraryID, RootPath: rootPath, FileIDs: normalizedFileIDs, ItemIDs: normalizedItemIDs}
		run, err := s.queueStandaloneWorkflowTask(ctx, libraryID, rootPath, WorkflowReasonManualScan, workflow.TaskTypeMaterializeCatalog, workflow.StageMaterialize, fmt.Sprintf("post-materialize:%s", rootPath), payload)
		return workflowRunCompatibilityJob(run), err
	}
	return database.Job{}, fmt.Errorf("workflow service unavailable")
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

func (s *Service) catalogMatchTargetForQueue(ctx context.Context, itemID uint) (uint, bool, error) {
	var item database.CatalogItem
	if err := s.db.WithContext(ctx).First(&item, itemID).Error; err != nil {
		return 0, false, fmt.Errorf("load catalog item %d: %w", itemID, err)
	}
	if item.DeletedAt != nil {
		return 0, false, fmt.Errorf("catalog item %d is deleted", item.ID)
	}

	targetID := item.ID
	if item.Type == catalog.ItemTypeSeason || item.Type == catalog.ItemTypeEpisode {
		if item.RootID == nil || *item.RootID == 0 {
			return 0, false, fmt.Errorf("catalog item %d missing root_id", item.ID)
		}
		targetID = *item.RootID
	}

	targetItem := item
	if targetID != item.ID {
		targetItem = database.CatalogItem{}
		if err := s.db.WithContext(ctx).First(&targetItem, targetID).Error; err != nil {
			return 0, false, fmt.Errorf("load catalog match target %d for item %d: %w", targetID, itemID, err)
		}
		if targetItem.DeletedAt != nil {
			return 0, false, fmt.Errorf("catalog item %d is deleted", targetItem.ID)
		}
	}

	return targetID, targetItem.GovernanceStatus == catalog.GovernancePending, nil
}
func (s *Service) QueueInventoryFileProbe(ctx context.Context, inventoryFileID uint, force bool) (database.Job, error) {
	if inventoryFileID == 0 {
		return database.Job{}, fmt.Errorf("inventory file id is required")
	}
	if force {
		var assetIDs []uint
		if err := s.db.WithContext(ctx).
			Model(&database.AssetFile{}).
			Distinct("asset_id").
			Where("file_id = ?", inventoryFileID).
			Pluck("asset_id", &assetIDs).Error; err != nil {
			return database.Job{}, err
		}
		if len(assetIDs) > 0 {
			if err := s.db.WithContext(ctx).
				Model(&database.MediaAsset{}).
				Where("id IN ?", assetIDs).
				Updates(map[string]any{
					"probe_status":           "pending",
					"technical_summary_json": "",
				}).Error; err != nil {
				return database.Job{}, err
			}
		}
	}
	if s.workflow == nil {
		return database.Job{}, fmt.Errorf("workflow service unavailable")
	}
	var file database.InventoryFile
	if err := s.db.WithContext(ctx).First(&file, inventoryFileID).Error; err != nil {
		return database.Job{}, err
	}

	run, reused, err := s.workflow.CreateOrReuseRun(ctx, workflow.CreateRunInput{
		RunKey:    fmt.Sprintf("inventory-file:%d:probe", inventoryFileID),
		LibraryID: file.LibraryID,
		Reason:    WorkflowReasonProbeInventory,
		Priority:  10,
		ScopeKey:  fmt.Sprintf("inventory-file:%d", inventoryFileID),
		Payload:   inventoryFileProbeWorkflowPayload{InventoryFileID: inventoryFileID},
	})
	if err != nil || reused {
		return workflowRunCompatibilityJob(run), err
	}
	_, err = s.workflow.CreateTask(ctx, run, workflow.CreateTaskInput{
		TaskKey:   fmt.Sprintf("run:%d:probe-file:%d", run.ID, inventoryFileID),
		TaskType:  workflow.TaskTypeProbeInventoryFile,
		Stage:     workflow.StageProbe,
		Priority:  10,
		ScopeKey:  run.ScopeKey,
		Payload:   inventoryFileProbeWorkflowPayload{InventoryFileID: inventoryFileID},
		Resources: workflow.DefaultTaskTypeDefinitions()[workflow.TaskTypeProbeInventoryFile].Resources,
	})
	return workflowRunCompatibilityJob(run), err
}
