package library

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
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
	workflowSvc := s.workflowCapability()
	if workflowSvc == nil {
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
	run, reused, err := workflowSvc.CreateOrReuseRun(ctx, workflow.CreateRunInput{
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
		_, err = workflowSvc.CreateTask(ctx, run, workflow.CreateTaskInput{
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
		_, err = workflowSvc.CreateTask(ctx, run, workflow.CreateTaskInput{
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
	runner.Register(workflow.TaskTypeResolveRecognition, s.RunWorkflowRecognitionResolve)
	runner.Register(workflow.TaskTypeRefreshProjection, s.RunWorkflowCatalogProjectionRefresh)
	runner.Register(workflow.TaskTypeProbeInventory, s.RunWorkflowInventoryProbeBatch)
	runner.Register(workflow.TaskTypeProbeInventoryFile, s.RunWorkflowInventoryFileProbe)
	runner.Register(workflow.TaskTypeMatchMetadata, s.RunWorkflowMetadataMatchBatch)
}

func (s *Service) queueStandaloneWorkflowTask(ctx context.Context, libraryID uint, rootPath string, reason string, taskType string, stage string, taskKeySuffix string, payload any) (database.WorkflowRun, error) {
	workflowSvc := s.workflowCapability()
	if workflowSvc == nil {
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
	run, reused, err := workflowSvc.CreateOrReuseRun(ctx, workflow.CreateRunInput{RunKey: runKey, LibraryID: libraryID, Reason: reason, Priority: 5, ScopeKey: fmt.Sprintf("library:%d", libraryID), Payload: payload})
	if err != nil || reused {
		return run, err
	}
	definition := workflow.DefaultTaskTypeDefinitions()[taskType]
	if stage == "" {
		stage = definition.Stage
	}
	_, err = workflowSvc.CreateTask(ctx, run, workflow.CreateTaskInput{TaskKey: fmt.Sprintf("run:%d:%s", run.ID, taskKeySuffix), TaskType: taskType, Stage: stage, Priority: 5, ScopeKey: run.ScopeKey, Payload: payload, Resources: definition.Resources})
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
	scanMode := scanMode{deferRecognitionResolution: true, rootPath: targetRoot}
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

func (s *Service) RunWorkflowRecognitionResolve(ctx context.Context, task database.WorkflowTask) error {
	var postPayload RecognitionPostResolvePayload
	if err := json.Unmarshal([]byte(task.PayloadJSON), &postPayload); err == nil && len(postPayload.MetadataItemIDs) > 0 {
		return s.RunRecognitionPostResolve(ctx, postPayload)
	}
	var payload RecognitionResolveBatchPayload
	if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("decode recognition resolve workflow payload: %w", err)
	}
	return s.runRecognitionResolveBatch(ctx, payload, task.RunID)
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

func (s *Service) RunWorkflowMetadataMatchBatch(ctx context.Context, task database.WorkflowTask) error {
	var payload MetadataMatchBatchPayload
	if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("decode metadata match workflow payload: %w", err)
	}
	return s.RunMetadataMatchBatch(ctx, payload)
}

func (s *Service) catalogProjectionRefresh(ctx context.Context, libraryID uint, rootPath string) error {
	if libraryID == 0 {
		return fmt.Errorf("library id is required")
	}
	_ = rootPath
	return catalog.NewService(s.db, s.ingestCapability()).RefreshLibraryProjectionScope(ctx, libraryID)
}

func (s *Service) queueWorkflowPostScanTasks(ctx context.Context, runID uint, libraryID uint, rootPath string, mode scanMode, scanPolicy database.LibraryScanPolicy) error {
	workflowSvc := s.workflowCapability()
	if workflowSvc == nil || runID == 0 {
		return s.queuePostScanEnrichment(ctx, libraryID, rootPath, mode, scanPolicy)
	}
	if err := s.queueWorkflowRecognitionResolveTasks(ctx, runID, libraryID, rootPath, mode.recognitionResolveFileIDs); err != nil {
		return err
	}
	if len(mode.recognitionResolveFileIDs) > 0 {
		return nil
	}
	if err := s.queueWorkflowMatchTasks(ctx, runID, libraryID, rootPath, mode.metadataMatchItemIDs); err != nil {
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
	if err := s.queueWorkflowProjectionTask(ctx, runID, libraryID, rootPath); err != nil {
		return err
	}
	return nil
}

func (s *Service) queueWorkflowPostRecognitionResolveTasks(ctx context.Context, runID uint, libraryID uint, rootPath string, fileIDs []uint, metadataItemIDs []uint, scanPolicy database.LibraryScanPolicy) error {
	workflowSvc := s.workflowCapability()
	if workflowSvc == nil || runID == 0 {
		if _, err := s.QueueMetadataMatchBatch(ctx, libraryID, rootPath, metadataItemIDs); err != nil {
			return err
		}
		if (EffectiveLibraryConfig{ScanPolicy: scanPolicy}).InventoryProbeBatchEnabled() {
			if _, err := s.QueueInventoryProbeBatch(ctx, libraryID, rootPath, fileIDs); err != nil {
				return err
			}
		}
		_, err := s.QueueCatalogLibraryProjectionRefresh(ctx, libraryID, rootPath)
		return err
	}
	if err := s.queueWorkflowMatchTasks(ctx, runID, libraryID, rootPath, metadataItemIDs); err != nil {
		return err
	}
	if (EffectiveLibraryConfig{ScanPolicy: scanPolicy}).InventoryProbeBatchEnabled() {
		if err := s.queueWorkflowProbeTasks(ctx, runID, libraryID, rootPath, fileIDs); err != nil {
			return err
		}
	}
	return s.queueWorkflowProjectionTask(ctx, runID, libraryID, rootPath)
}

func (s *Service) queueWorkflowProjectionTask(ctx context.Context, runID uint, libraryID uint, rootPath string) error {
	workflowSvc := s.workflowCapability()
	if workflowSvc == nil {
		return fmt.Errorf("workflow service unavailable")
	}
	taskKey := fmt.Sprintf("run:%d:projection:%s", runID, rootPath)
	exists, err := s.workflowTaskExists(ctx, taskKey)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = workflowSvc.CreateTask(ctx, database.WorkflowRun{ID: runID, LibraryID: libraryID, ScopeKey: fmt.Sprintf("library:%d", libraryID)}, workflow.CreateTaskInput{
		TaskKey:   taskKey,
		TaskType:  workflow.TaskTypeRefreshProjection,
		Stage:     workflow.StageProjection,
		ScopeKey:  fmt.Sprintf("library:%d", libraryID),
		Payload:   map[string]any{"library_id": libraryID, "root_path": rootPath},
		Resources: workflow.DefaultTaskTypeDefinitions()[workflow.TaskTypeRefreshProjection].Resources,
	})
	return err
}

func (s *Service) queueWorkflowProbeTasks(ctx context.Context, runID uint, libraryID uint, rootPath string, fileIDs []uint) error {
	workflowSvc := s.workflowCapability()
	if workflowSvc == nil {
		return fmt.Errorf("workflow service unavailable")
	}
	ids := normalizeUintIDs(fileIDs)
	for start := 0; start < len(ids); start += recognitionResolveScanBatchSize {
		end := start + recognitionResolveScanBatchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := append([]uint(nil), ids[start:end]...)
		taskKey := fmt.Sprintf("run:%d:probe:%s:%d", runID, rootPath, start)
		exists, err := s.workflowTaskExists(ctx, taskKey)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		_, err = workflowSvc.CreateTask(ctx, database.WorkflowRun{ID: runID, LibraryID: libraryID, ScopeKey: fmt.Sprintf("library:%d", libraryID)}, workflow.CreateTaskInput{
			TaskKey:   taskKey,
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

func (s *Service) queueWorkflowMatchTasks(ctx context.Context, runID uint, libraryID uint, rootPath string, metadataItemIDs []uint) error {
	workflowSvc := s.workflowCapability()
	if workflowSvc == nil {
		return fmt.Errorf("workflow service unavailable")
	}
	ids, err := s.filterMetadataMatchableItemIDs(ctx, normalizeUintIDs(metadataItemIDs))
	if err != nil {
		return err
	}
	for start := 0; start < len(ids); start += recognitionResolveScanBatchSize {
		end := start + recognitionResolveScanBatchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := append([]uint(nil), ids[start:end]...)
		taskKey := fmt.Sprintf("run:%d:match:%s:%d", runID, rootPath, start)
		exists, err := s.workflowTaskExists(ctx, taskKey)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		_, err = workflowSvc.CreateTask(ctx, database.WorkflowRun{ID: runID, LibraryID: libraryID, ScopeKey: fmt.Sprintf("library:%d", libraryID)}, workflow.CreateTaskInput{
			TaskKey:   taskKey,
			TaskType:  workflow.TaskTypeMatchMetadata,
			Stage:     workflow.StageMetadataMatch,
			ScopeKey:  fmt.Sprintf("library:%d", libraryID),
			Payload:   MetadataMatchBatchPayload{LibraryID: libraryID, RootPath: rootPath, MetadataItemIDs: batch},
			Resources: workflow.DefaultTaskTypeDefinitions()[workflow.TaskTypeMatchMetadata].Resources,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) queueWorkflowRecognitionResolveTasks(ctx context.Context, runID uint, libraryID uint, rootPath string, fileIDs []uint) error {
	workflowSvc := s.workflowCapability()
	if workflowSvc == nil {
		return fmt.Errorf("workflow service unavailable")
	}
	groups, err := s.recognitionResolveGroups(ctx, fileIDs)
	if err != nil {
		return err
	}
	for _, group := range groups {
		taskKey := fmt.Sprintf("run:%d:resolve-recognition:%s:%d", runID, group.RootPath, group.Start)
		exists, err := s.workflowTaskExists(ctx, taskKey)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		_, err = workflowSvc.CreateTask(ctx, database.WorkflowRun{ID: runID, LibraryID: libraryID, ScopeKey: fmt.Sprintf("library:%d", libraryID)}, workflow.CreateTaskInput{
			TaskKey:   taskKey,
			TaskType:  workflow.TaskTypeResolveRecognition,
			Stage:     workflow.StageMaterialize,
			ScopeKey:  fmt.Sprintf("library:%d", libraryID),
			Payload:   RecognitionResolveBatchPayload{LibraryID: libraryID, RootPath: group.RootPath, FileIDs: group.FileIDs},
			Resources: workflow.DefaultTaskTypeDefinitions()[workflow.TaskTypeResolveRecognition].Resources,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) workflowTaskExists(ctx context.Context, taskKey string) (bool, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&database.WorkflowTask{}).Where("task_key = ?", taskKey).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Service) queueStandaloneRecognitionResolveTasks(ctx context.Context, libraryID uint, rootPath string, fileIDs []uint) error {
	workflowSvc := s.workflowCapability()
	if workflowSvc == nil {
		return fmt.Errorf("workflow service unavailable")
	}
	if libraryID == 0 {
		return fmt.Errorf("library id is required")
	}
	groups, err := s.recognitionResolveGroups(ctx, fileIDs)
	if err != nil {
		return err
	}
	if len(groups) == 0 {
		return nil
	}
	rootPath = strings.TrimSpace(rootPath)
	for _, group := range groups {
		groupRoot := firstNonEmptyString(group.RootPath, rootPath)
		runKey := fmt.Sprintf("library:%d:%s:%s:%s", libraryID, WorkflowReasonManualScan, workflow.TaskTypeResolveRecognition, groupRoot)
		run, reused, err := workflowSvc.CreateOrReuseRun(ctx, workflow.CreateRunInput{RunKey: runKey, LibraryID: libraryID, Reason: WorkflowReasonManualScan, Priority: 5, ScopeKey: fmt.Sprintf("library:%d", libraryID), Payload: RecognitionResolveBatchPayload{LibraryID: libraryID, RootPath: groupRoot, FileIDs: group.FileIDs}})
		if err != nil {
			return err
		}
		if reused {
			continue
		}
		_, err = workflowSvc.CreateTask(ctx, run, workflow.CreateTaskInput{
			TaskKey:   fmt.Sprintf("run:%d:resolve-recognition:%s:%d", run.ID, groupRoot, group.Start),
			TaskType:  workflow.TaskTypeResolveRecognition,
			Stage:     workflow.StageMaterialize,
			Priority:  5,
			ScopeKey:  run.ScopeKey,
			Payload:   RecognitionResolveBatchPayload{LibraryID: libraryID, RootPath: groupRoot, FileIDs: group.FileIDs},
			Resources: workflow.DefaultTaskTypeDefinitions()[workflow.TaskTypeResolveRecognition].Resources,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

type recognitionResolveGroup struct {
	RootPath string
	Start    int
	FileIDs  []uint
}

func (s *Service) recognitionResolveGroups(ctx context.Context, fileIDs []uint) ([]recognitionResolveGroup, error) {
	ids := normalizeUintIDs(fileIDs)
	if len(ids) == 0 {
		return nil, nil
	}
	var files []database.InventoryFile
	for _, batch := range chunkUints(ids, sqliteVariableChunkSize) {
		var partial []database.InventoryFile
		if err := s.db.WithContext(ctx).Where("id IN ?", batch).Find(&partial).Error; err != nil {
			return nil, err
		}
		files = append(files, partial...)
	}
	pathByID := make(map[uint]string, len(files))
	for _, file := range files {
		if file.ID == 0 {
			continue
		}
		pathByID[file.ID] = recognitionResolveGroupRootPath(file.StoragePath)
	}
	grouped := make(map[string][]uint)
	orderedRoots := make([]string, 0)
	for _, id := range ids {
		root := pathByID[id]
		if _, ok := grouped[root]; !ok {
			orderedRoots = append(orderedRoots, root)
		}
		grouped[root] = append(grouped[root], id)
	}
	groups := make([]recognitionResolveGroup, 0)
	for _, root := range orderedRoots {
		groupIDs := grouped[root]
		groups = append(groups, recognitionResolveGroup{RootPath: root, Start: 0, FileIDs: append([]uint(nil), groupIDs...)})
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].RootPath != groups[j].RootPath {
			return groups[i].RootPath < groups[j].RootPath
		}
		return groups[i].Start < groups[j].Start
	})
	return groups, nil
}

func recognitionResolveGroupRootPath(storagePath string) string {
	trimmed := strings.TrimSpace(storagePath)
	if trimmed == "" {
		return ""
	}
	dir := strings.TrimSpace(path.Dir(trimmed))
	if dir == "." {
		return ""
	}
	return dir
}
