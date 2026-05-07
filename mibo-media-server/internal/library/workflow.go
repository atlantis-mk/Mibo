package library

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/workflow"
)

const (
	WorkflowReasonManualScan      = "manual_scan"
	WorkflowReasonCreateLibrary   = "library_created"
	WorkflowReasonTargetedRefresh = "targeted_refresh"
	WorkflowReasonScheduledScan   = "scheduled_scan"
	WorkflowReasonStorageRefresh  = "storage_refresh"
	WorkflowReasonProbeInventory  = "probe_inventory"
	WorkflowReasonMissingCleanup  = "missing_cleanup"
)

type QueueWorkflowInput struct {
	LibraryID uint
	Reason    string
	RootPath  string
	Priority  int
}

type scanWorkflowPayload struct {
	LibraryID uint   `json:"library_id"`
	RootPath  string `json:"root_path,omitempty"`
	Reason    string `json:"reason"`
}

type inventoryFileProbeWorkflowPayload struct {
	InventoryFileID uint `json:"inventory_file_id"`
}

func (s *Service) QueueLibraryWorkflow(ctx context.Context, input QueueWorkflowInput) (database.WorkflowRun, bool, error) {
	if s.workflow == nil {
		return database.WorkflowRun{}, false, fmt.Errorf("workflow service unavailable")
	}
	if input.LibraryID == 0 {
		return database.WorkflowRun{}, false, fmt.Errorf("library id is required")
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = WorkflowReasonManualScan
	}
	rootPath := strings.TrimSpace(input.RootPath)
	runKey := fmt.Sprintf("library:%d:%s", input.LibraryID, reason)
	if rootPath != "" {
		runKey = fmt.Sprintf("%s:%s", runKey, rootPath)
	}
	run, reused, err := s.workflow.CreateOrReuseRun(ctx, workflow.CreateRunInput{
		RunKey:    runKey,
		LibraryID: input.LibraryID,
		Reason:    reason,
		Priority:  input.Priority,
		ScopeKey:  fmt.Sprintf("library:%d", input.LibraryID),
		Payload: map[string]any{
			"library_id": input.LibraryID,
			"root_path":  rootPath,
			"reason":     reason,
		},
	})
	if err != nil {
		return database.WorkflowRun{}, false, err
	}
	if reused {
		return run, true, nil
	}
	if strings.TrimSpace(rootPath) != "" {
		_, err = s.workflow.CreateTask(ctx, run, workflow.CreateTaskInput{
			TaskKey:   fmt.Sprintf("run:%d:scan:%s", run.ID, rootPath),
			TaskType:  workflow.TaskTypeScanLibraryPath,
			Stage:     workflow.StageScan,
			Priority:  input.Priority,
			ScopeKey:  run.ScopeKey,
			Payload:   scanWorkflowPayload{LibraryID: input.LibraryID, RootPath: rootPath, Reason: reason},
			Resources: workflow.DefaultTaskTypeDefinitions()[workflow.TaskTypeScanLibraryPath].Resources,
		})
		if err != nil {
			return database.WorkflowRun{}, false, err
		}
		return run, false, nil
	}

	config, err := s.EffectiveLibraryConfig(ctx, input.LibraryID)
	if err != nil {
		return database.WorkflowRun{}, false, err
	}
	for _, pathRecord := range config.Paths {
		if !pathRecord.Enabled || pathRecord.DeletedAt != nil {
			continue
		}
		_, err = s.workflow.CreateTask(ctx, run, workflow.CreateTaskInput{
			TaskKey:   fmt.Sprintf("run:%d:scan:%s", run.ID, pathRecord.RootPath),
			TaskType:  workflow.TaskTypeScanLibraryPath,
			Stage:     workflow.StageScan,
			Priority:  input.Priority,
			ScopeKey:  run.ScopeKey,
			Payload:   scanWorkflowPayload{LibraryID: input.LibraryID, RootPath: pathRecord.RootPath, Reason: reason},
			Resources: workflow.DefaultTaskTypeDefinitions()[workflow.TaskTypeScanLibraryPath].Resources,
		})
		if err != nil {
			return database.WorkflowRun{}, false, err
		}
	}
	return run, false, nil
}

func (s *Service) RegisterWorkflowHandlers(runner *workflow.Runner) {
	if runner == nil {
		return
	}
	runner.Register(workflow.TaskTypeScanLibraryPath, s.RunWorkflowScanLibraryPath)
	runner.Register(workflow.TaskTypeMaterializeCatalog, s.RunWorkflowCatalogMaterialize)
	runner.Register(workflow.TaskTypeRefreshProjection, s.RunWorkflowCatalogProjectionRefresh)
	runner.Register(workflow.TaskTypeProbeInventory, s.RunWorkflowInventoryProbeBatch)
	runner.Register(workflow.TaskTypeProbeInventoryFile, s.RunWorkflowInventoryFileProbe)
	runner.Register(workflow.TaskTypeMatchMetadata, s.RunWorkflowCatalogMatchBatch)
}

func (s *Service) queueStandaloneWorkflowTask(ctx context.Context, libraryID uint, rootPath string, reason string, taskType string, stage string, taskKeySuffix string, payload any) (database.WorkflowRun, error) {
	if s.workflow == nil {
		return database.WorkflowRun{}, fmt.Errorf("workflow service unavailable")
	}
	if libraryID == 0 {
		return database.WorkflowRun{}, fmt.Errorf("library id is required")
	}
	if strings.TrimSpace(reason) == "" {
		reason = WorkflowReasonManualScan
	}
	rootPath = strings.TrimSpace(rootPath)
	runKey := fmt.Sprintf("library:%d:%s:%s:%s", libraryID, reason, taskType, rootPath)
	run, reused, err := s.workflow.CreateOrReuseRun(ctx, workflow.CreateRunInput{RunKey: runKey, LibraryID: libraryID, Reason: reason, Priority: 5, ScopeKey: fmt.Sprintf("library:%d", libraryID), Payload: payload})
	if err != nil || reused {
		return run, err
	}
	definition := workflow.DefaultTaskTypeDefinitions()[taskType]
	if stage == "" {
		stage = definition.Stage
	}
	_, err = s.workflow.CreateTask(ctx, run, workflow.CreateTaskInput{TaskKey: fmt.Sprintf("run:%d:%s", run.ID, taskKeySuffix), TaskType: taskType, Stage: stage, Priority: 5, ScopeKey: run.ScopeKey, Payload: payload, Resources: definition.Resources})
	return run, err
}

func (s *Service) RunWorkflowScanLibraryPath(ctx context.Context, task database.WorkflowTask) error {
	var payload scanWorkflowPayload
	if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("decode scan workflow payload: %w", err)
	}
	if payload.LibraryID == 0 {
		return fmt.Errorf("workflow scan library id is required")
	}
	config, err := s.EffectiveLibraryConfig(ctx, payload.LibraryID)
	if err != nil {
		return err
	}
	pathRecord, provider, targetRoot, err := s.scopedRefreshPath(ctx, config, payload.RootPath)
	if err != nil {
		return err
	}
	if err := s.updateLibraryStatus(ctx, config.Library.ID, "syncing"); err != nil {
		return err
	}
	s.markLibraryScopeDirty(ctx, config.Library.ID, targetRoot, "workflow_scan_started")
	libraryForPath := config.Library
	libraryForPath.MediaSourceID = pathRecord.MediaSourceID
	libraryForPath.RootPath = pathRecord.RootPath
	scanMode := scanMode{deferCatalogMaterialization: true, rootPath: targetRoot}
	if _, err := s.scanLibraryWithMode(ctx, provider, libraryForPath, targetRoot, &scanMode); err != nil {
		_ = s.updateLibraryStatus(ctx, config.Library.ID, "error")
		return err
	}
	if err := s.queueWorkflowPostScanTasks(ctx, task.RunID, config.Library.ID, targetRoot, scanMode, config.ScanPolicy); err != nil {
		_ = s.updateLibraryStatus(ctx, config.Library.ID, "error")
		return err
	}
	return s.updateLibraryStatus(ctx, config.Library.ID, "active")
}

func (s *Service) RunWorkflowCatalogMaterialize(ctx context.Context, task database.WorkflowTask) error {
	var postPayload CatalogPostMaterializeBatchPayload
	if err := json.Unmarshal([]byte(task.PayloadJSON), &postPayload); err == nil && len(postPayload.ItemIDs) > 0 {
		return s.RunCatalogPostMaterializeBatch(ctx, postPayload)
	}
	var payload CatalogMaterializeBatchPayload
	if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("decode materialize workflow payload: %w", err)
	}
	return s.RunCatalogMaterializeBatch(ctx, payload)
}

func (s *Service) RunWorkflowCatalogProjectionRefresh(ctx context.Context, task database.WorkflowTask) error {
	var payload struct {
		LibraryID uint   `json:"library_id"`
		RootPath  string `json:"root_path"`
	}
	if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("decode projection workflow payload: %w", err)
	}
	return s.catalogProjectionRefresh(ctx, payload.LibraryID, payload.RootPath)
}

func (s *Service) RunWorkflowInventoryProbeBatch(ctx context.Context, task database.WorkflowTask) error {
	var payload InventoryProbeBatchPayload
	if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("decode probe workflow payload: %w", err)
	}
	return s.RunInventoryProbeBatch(ctx, payload)
}

func (s *Service) RunWorkflowInventoryFileProbe(ctx context.Context, task database.WorkflowTask) error {
	var payload inventoryFileProbeWorkflowPayload
	if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("decode inventory file probe workflow payload: %w", err)
	}
	if payload.InventoryFileID == 0 {
		return fmt.Errorf("inventory file id is required")
	}
	return s.RunInventoryProbeBatch(ctx, InventoryProbeBatchPayload{LibraryID: task.LibraryID, FileIDs: []uint{payload.InventoryFileID}})
}

func (s *Service) RunWorkflowCatalogMatchBatch(ctx context.Context, task database.WorkflowTask) error {
	var payload CatalogMatchBatchPayload
	if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("decode metadata match workflow payload: %w", err)
	}
	return s.RunCatalogMatchBatch(ctx, payload)
}

func (s *Service) catalogProjectionRefresh(ctx context.Context, libraryID uint, rootPath string) error {
	if libraryID == 0 {
		return fmt.Errorf("library id is required")
	}
	return catalog.NewService(s.db, s.ingest).RefreshLibraryProjection(ctx, libraryID, rootPath)
}

func (s *Service) queueWorkflowPostScanTasks(ctx context.Context, runID uint, libraryID uint, rootPath string, mode scanMode, scanPolicy database.LibraryScanPolicy) error {
	if s.workflow == nil || runID == 0 {
		return s.queuePostScanEnrichment(ctx, libraryID, rootPath, mode, scanPolicy)
	}
	if err := s.queueWorkflowMaterializeTasks(ctx, runID, libraryID, rootPath, mode.catalogMaterializeFileIDs); err != nil {
		return err
	}
	if err := s.queueWorkflowMatchTasks(ctx, runID, libraryID, rootPath, mode.catalogMatchItemIDs); err != nil {
		return err
	}
	if scanPolicy.InventoryProbeBatchEnabled {
		if err := s.queueWorkflowProbeTasks(ctx, runID, libraryID, rootPath, mode.inventoryProbeFileIDs); err != nil {
			return err
		}
		if err := s.queueWorkflowProbeTasks(ctx, runID, libraryID, rootPath, mode.classificationFileIDs); err != nil {
			return err
		}
	}
	if _, err := s.workflow.CreateTask(ctx, database.WorkflowRun{ID: runID, LibraryID: libraryID, ScopeKey: fmt.Sprintf("library:%d", libraryID)}, workflow.CreateTaskInput{
		TaskKey:   fmt.Sprintf("run:%d:projection:%s", runID, rootPath),
		TaskType:  workflow.TaskTypeRefreshProjection,
		Stage:     workflow.StageProjection,
		ScopeKey:  fmt.Sprintf("library:%d", libraryID),
		Payload:   map[string]any{"library_id": libraryID, "root_path": rootPath},
		Resources: workflow.DefaultTaskTypeDefinitions()[workflow.TaskTypeRefreshProjection].Resources,
	}); err != nil {
		return err
	}
	return nil
}

func (s *Service) queueWorkflowProbeTasks(ctx context.Context, runID uint, libraryID uint, rootPath string, fileIDs []uint) error {
	ids := normalizeUintIDs(fileIDs)
	for start := 0; start < len(ids); start += catalogMaterializeScanBatchSize {
		end := start + catalogMaterializeScanBatchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := append([]uint(nil), ids[start:end]...)
		_, err := s.workflow.CreateTask(ctx, database.WorkflowRun{ID: runID, LibraryID: libraryID, ScopeKey: fmt.Sprintf("library:%d", libraryID)}, workflow.CreateTaskInput{
			TaskKey:   fmt.Sprintf("run:%d:probe:%s:%d", runID, rootPath, start),
			TaskType:  workflow.TaskTypeProbeInventory,
			Stage:     workflow.StageProbe,
			ScopeKey:  fmt.Sprintf("library:%d", libraryID),
			Payload:   InventoryProbeBatchPayload{LibraryID: libraryID, RootPath: rootPath, FileIDs: batch},
			Resources: workflow.DefaultTaskTypeDefinitions()[workflow.TaskTypeProbeInventory].Resources,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) queueWorkflowMatchTasks(ctx context.Context, runID uint, libraryID uint, rootPath string, itemIDs []uint) error {
	ids := normalizeUintIDs(itemIDs)
	for start := 0; start < len(ids); start += catalogMaterializeScanBatchSize {
		end := start + catalogMaterializeScanBatchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := append([]uint(nil), ids[start:end]...)
		_, err := s.workflow.CreateTask(ctx, database.WorkflowRun{ID: runID, LibraryID: libraryID, ScopeKey: fmt.Sprintf("library:%d", libraryID)}, workflow.CreateTaskInput{
			TaskKey:   fmt.Sprintf("run:%d:match:%s:%d", runID, rootPath, start),
			TaskType:  workflow.TaskTypeMatchMetadata,
			Stage:     workflow.StageMetadataMatch,
			ScopeKey:  fmt.Sprintf("library:%d", libraryID),
			Payload:   CatalogMatchBatchPayload{LibraryID: libraryID, RootPath: rootPath, ItemIDs: batch},
			Resources: workflow.DefaultTaskTypeDefinitions()[workflow.TaskTypeMatchMetadata].Resources,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) queueWorkflowMaterializeTasks(ctx context.Context, runID uint, libraryID uint, rootPath string, fileIDs []uint) error {
	ids := normalizeUintIDs(fileIDs)
	for start := 0; start < len(ids); start += catalogMaterializeScanBatchSize {
		end := start + catalogMaterializeScanBatchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := append([]uint(nil), ids[start:end]...)
		_, err := s.workflow.CreateTask(ctx, database.WorkflowRun{ID: runID, LibraryID: libraryID, ScopeKey: fmt.Sprintf("library:%d", libraryID)}, workflow.CreateTaskInput{
			TaskKey:   fmt.Sprintf("run:%d:materialize:%s:%d", runID, rootPath, start),
			TaskType:  workflow.TaskTypeMaterializeCatalog,
			Stage:     workflow.StageMaterialize,
			ScopeKey:  fmt.Sprintf("library:%d", libraryID),
			Payload:   CatalogMaterializeBatchPayload{LibraryID: libraryID, RootPath: rootPath, FileIDs: batch},
			Resources: workflow.DefaultTaskTypeDefinitions()[workflow.TaskTypeMaterializeCatalog].Resources,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
