package library

import (
	"context"
	"fmt"
	"sort"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/workflow"
)

type MetadataMatchBatchPayload struct {
	LibraryID       uint   `json:"library_id"`
	RootPath        string `json:"root_path,omitempty"`
	MetadataItemIDs []uint `json:"metadata_item_ids"`
}

type InventoryProbeBatchPayload struct {
	LibraryID uint   `json:"library_id"`
	RootPath  string `json:"root_path,omitempty"`
	FileIDs   []uint `json:"file_ids"`
}

type RecognitionResolveBatchPayload struct {
	LibraryID uint   `json:"library_id"`
	RootPath  string `json:"root_path,omitempty"`
	FileIDs   []uint `json:"file_ids"`
	mode      *scanMode
}

type RecognitionPostResolvePayload struct {
	LibraryID       uint   `json:"library_id"`
	RootPath        string `json:"root_path,omitempty"`
	FileIDs         []uint `json:"file_ids,omitempty"`
	MetadataItemIDs []uint `json:"metadata_item_ids,omitempty"`
}

func (s *Service) QueueMetadataMatchBatch(ctx context.Context, libraryID uint, rootPath string, metadataItemIDs []uint) (database.Job, error) {
	ids, err := s.filterMetadataMatchableItemIDs(ctx, normalizeUintIDs(metadataItemIDs))
	if err != nil {
		return database.Job{}, err
	}
	if len(ids) == 0 {
		return database.Job{}, nil
	}
	if s.workflowCapability() != nil {
		run, err := s.queueStandaloneWorkflowTask(ctx, libraryID, rootPath, WorkflowReasonManualScan, workflow.TaskTypeMatchMetadata, workflow.StageMetadataMatch, fmt.Sprintf("match:%s", rootPath), MetadataMatchBatchPayload{LibraryID: libraryID, RootPath: rootPath, MetadataItemIDs: ids})
		return workflowRunCompatibilityJob(run), err
	}
	return database.Job{}, fmt.Errorf("workflow service unavailable")
}

func (s *Service) QueueInventoryProbeBatch(ctx context.Context, libraryID uint, rootPath string, fileIDs []uint) (database.Job, error) {
	ids := normalizeUintIDs(fileIDs)
	if len(ids) == 0 {
		return database.Job{}, nil
	}
	if s.workflowCapability() != nil {
		run, err := s.queueStandaloneWorkflowTask(ctx, libraryID, rootPath, WorkflowReasonProbeInventory, workflow.TaskTypeProbeInventory, workflow.StageProbe, fmt.Sprintf("probe:%s", rootPath), InventoryProbeBatchPayload{LibraryID: libraryID, RootPath: rootPath, FileIDs: ids})
		return workflowRunCompatibilityJob(run), err
	}
	return database.Job{}, fmt.Errorf("workflow service unavailable")
}

func (s *Service) QueueRecognitionResolveBatch(ctx context.Context, libraryID uint, rootPath string, fileIDs []uint) (database.Job, error) {
	ids := normalizeUintIDs(fileIDs)
	if len(ids) == 0 {
		return database.Job{}, nil
	}
	if s.workflowCapability() != nil {
		run, err := s.queueStandaloneWorkflowTask(ctx, libraryID, rootPath, WorkflowReasonManualScan, workflow.TaskTypeResolveRecognition, workflow.StageMaterialize, fmt.Sprintf("resolve-recognition:%s", rootPath), RecognitionResolveBatchPayload{LibraryID: libraryID, RootPath: rootPath, FileIDs: ids})
		return workflowRunCompatibilityJob(run), err
	}
	return database.Job{}, fmt.Errorf("workflow service unavailable")
}

func (s *Service) QueueRecognitionPostResolve(ctx context.Context, libraryID uint, rootPath string, fileIDs []uint, metadataItemIDs []uint) (database.Job, error) {
	normalizedFileIDs := normalizeUintIDs(fileIDs)
	normalizedMetadataItemIDs := normalizeUintIDs(metadataItemIDs)
	if len(normalizedFileIDs) == 0 && len(normalizedMetadataItemIDs) == 0 {
		return database.Job{}, nil
	}
	if s.workflowCapability() != nil {
		payload := RecognitionPostResolvePayload{LibraryID: libraryID, RootPath: rootPath, FileIDs: normalizedFileIDs, MetadataItemIDs: normalizedMetadataItemIDs}
		run, err := s.queueStandaloneWorkflowTask(ctx, libraryID, rootPath, WorkflowReasonManualScan, workflow.TaskTypeResolveRecognition, workflow.StageMaterialize, fmt.Sprintf("post-resolve-recognition:%s", rootPath), payload)
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

func (s *Service) QueueInventoryFileProbe(ctx context.Context, inventoryFileID uint, force bool) (database.Job, error) {
	if inventoryFileID == 0 {
		return database.Job{}, fmt.Errorf("inventory file id is required")
	}
	if force {
		var resourceIDs []uint
		if err := s.db.WithContext(ctx).
			Model(&database.ResourceFile{}).
			Distinct("resource_id").
			Where("inventory_file_id = ?", inventoryFileID).
			Pluck("resource_id", &resourceIDs).Error; err != nil {
			return database.Job{}, err
		}
		if len(resourceIDs) > 0 {
			if err := s.db.WithContext(ctx).
				Model(&database.Resource{}).
				Where("id IN ?", resourceIDs).
				Updates(map[string]any{
					"probe_status":           "pending",
					"technical_summary_json": "",
				}).Error; err != nil {
				return database.Job{}, err
			}
		}
	}
	workflowSvc := s.workflowCapability()
	if workflowSvc == nil {
		return database.Job{}, fmt.Errorf("workflow service unavailable")
	}
	var file database.InventoryFile
	if err := s.db.WithContext(ctx).First(&file, inventoryFileID).Error; err != nil {
		return database.Job{}, err
	}

	run, reused, err := workflowSvc.CreateOrReuseRun(ctx, workflow.CreateRunInput{
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
	_, err = workflowSvc.CreateTask(ctx, run, workflow.CreateTaskInput{
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
