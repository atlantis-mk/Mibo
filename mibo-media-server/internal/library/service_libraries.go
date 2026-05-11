package library

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/settings"
	"github.com/atlan/mibo-media-server/internal/storage"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
)

func (s *Service) CreateLibrary(ctx context.Context, input CreateLibraryInput) (database.Library, database.Job, error) {
	if input.MediaSourceID == 0 {
		return database.Library{}, database.Job{}, fmt.Errorf("media_source_id is required")
	}
	source, provider, rootPath, err := s.resolveLibraryCreateSource(ctx, input.MediaSourceID, input.RootPath)
	if err != nil {
		return database.Library{}, database.Job{}, err
	}
	library, err := s.createLibraryRecord(ctx, source, provider, rootPath, strings.TrimSpace(input.Name))
	if err != nil {
		return database.Library{}, database.Job{}, err
	}
	if err := s.applyCreateLibraryInput(ctx, library.ID, input); err != nil {
		return database.Library{}, database.Job{}, err
	}
	job, err := s.QueueLibraryScanWithReason(ctx, library.ID, WorkflowReasonCreateLibrary)
	if err != nil {
		if strings.Contains(err.Error(), "workflow service unavailable") {
			return library, database.Job{}, nil
		}
		return database.Library{}, database.Job{}, err
	}
	return library, job, nil
}

func (s *Service) resolveLibraryCreateSource(ctx context.Context, mediaSourceID uint, requestedRoot string) (database.MediaSource, storage.Provider, string, error) {
	var source database.MediaSource
	if err := s.db.WithContext(ctx).First(&source, mediaSourceID).Error; err != nil {
		return database.MediaSource{}, nil, "", err
	}
	rootPath := normalizePath(requestedRoot)
	if rootPath == "/" {
		rootPath = source.RootPath
	}
	rootPath = normalizePathForProvider(source.Provider, rootPath)
	provider, err := s.storageRegistry().BuildForSource(source)
	if err != nil {
		return database.MediaSource{}, nil, "", err
	}
	if _, err := provider.ResolveStorage(ctx, storage.ResolveStorageRequest{Path: rootPath}); err != nil {
		return database.MediaSource{}, nil, "", fmt.Errorf("resolve library root %s: %w", rootPath, err)
	}
	return source, provider, rootPath, nil
}

func (s *Service) createLibraryRecord(ctx context.Context, source database.MediaSource, provider storage.Provider, rootPath string, requestedName string) (database.Library, error) {
	name := requestedName
	if name == "" {
		name = deriveLibraryNameFromPath(rootPath, source.Name)
	}
	pendingProbe := SourceProbeSummary{Status: SourceProbeStatusPending, Classes: emptySourceProbeClassCounts(), MaxObjects: defaultSourceProbeMaxObjects, MaxDepth: defaultSourceProbeMaxDepth}
	library := database.Library{Name: name, Type: LibraryTypeAuto, MediaSourceID: source.ID, RootPath: rootPath, Status: "pending", ScannerEnabled: true, ProbeStatus: pendingProbe.Status, ProbeSummaryJSON: encodeSourceProbeSummary(pendingProbe)}
	if err := s.db.WithContext(ctx).Create(&library).Error; err != nil {
		return database.Library{}, err
	}
	pathRecord := database.LibraryPath{LibraryID: library.ID, MediaSourceID: source.ID, RootPath: rootPath, DisplayName: library.Name, Enabled: true}
	if err := s.db.WithContext(ctx).Create(&pathRecord).Error; err != nil {
		return database.Library{}, err
	}
	if err := database.EnsureLibraryPolicyDefaults(s.db.WithContext(ctx), library.ID); err != nil {
		return database.Library{}, err
	}
	s.probeLibrarySourceAsync(library.ID, provider, rootPath)
	return library, nil
}

func (s *Service) probeLibrarySourceAsync(libraryID uint, provider storage.Provider, rootPath string) {
	go func() {
		probe := s.ProbeSource(context.Background(), provider, rootPath)
		_ = s.db.WithContext(context.Background()).Model(&database.Library{}).Where("id = ?", libraryID).Updates(map[string]any{
			"probe_status":       probe.Status,
			"probe_summary_json": encodeSourceProbeSummary(probe),
		}).Error
	}()
}

func (s *Service) applyCreateLibraryInput(ctx context.Context, libraryID uint, input CreateLibraryInput) error {
	if input.Scan != nil {
		if _, err := s.UpdateLibraryScanPolicy(ctx, libraryID, *input.Scan); err != nil {
			return err
		}
	}
	if input.Metadata != nil {
		if _, err := s.UpdateLibraryMetadataPolicy(ctx, libraryID, *input.Metadata); err != nil {
			return err
		}
	}
	if input.MetadataStrategy != nil {
		if _, err := settings.NewService(s.db, s.cfg.Metadata).UpdateLibraryMetadataStrategy(ctx, libraryID, *input.MetadataStrategy); err != nil {
			return err
		}
	}
	if input.Playback != nil {
		if _, err := s.UpdateLibraryPlaybackPolicy(ctx, libraryID, *input.Playback); err != nil {
			return err
		}
	}
	if input.Subtitle != nil {
		if _, err := s.UpdateLibrarySubtitlePolicy(ctx, libraryID, *input.Subtitle); err != nil {
			return err
		}
	}
	if input.ScanExclusionRules != nil {
		if _, err := s.ReplaceLibraryScanExclusionRules(ctx, libraryID, input.ScanExclusionRules, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) QueueLibraryScanWithReason(ctx context.Context, libraryID uint, reason string) (database.Job, error) {
	if s.workflowCapability() != nil {
		run, _, err := s.QueueLibraryWorkflow(ctx, QueueWorkflowInput{LibraryID: libraryID, Reason: reason, Priority: 10})
		return workflowRunCompatibilityJob(run), err
	}
	return database.Job{}, fmt.Errorf("workflow service unavailable")
}

func workflowRunCompatibilityJob(run database.WorkflowRun) database.Job {
	now := time.Now().UTC()
	return database.Job{ID: run.ID, JobKey: run.RunKey, Kind: JobKindSyncLibrary, Status: run.Status, PayloadJSON: run.PayloadJSON, ErrorMessage: run.ErrorMessage, AvailableAt: now, CreatedAt: run.CreatedAt, UpdatedAt: run.UpdatedAt}
}

func deriveLibraryNameFromPath(rootPath string, fallback string) string {
	trimmed := strings.Trim(strings.TrimSpace(rootPath), "/")
	if trimmed == "" {
		trimmed = strings.TrimSpace(fallback)
	}
	if trimmed == "" {
		return "媒体来源"
	}
	parts := strings.FieldsFunc(trimmed, func(r rune) bool { return r == '/' || r == '\\' })
	if len(parts) == 0 {
		return trimmed
	}
	return parts[len(parts)-1]
}

func (s *Service) QueueTargetedRefresh(ctx context.Context, libraryID uint, rootPath, reason string) (database.Job, error) {
	config, err := s.EffectiveLibraryConfig(ctx, libraryID)
	if err != nil {
		return database.Job{}, err
	}
	normalizedReason := strings.TrimSpace(strings.ToLower(reason))
	if normalizedReason == "" {
		normalizedReason = "manual"
	}
	_, _, targetRoot, err := s.scopedRefreshPath(ctx, config, rootPath)
	if err != nil {
		return database.Job{}, err
	}
	if s.hasActiveScanWorkflow(ctx, config.Library.ID) {
		return database.Job{}, nil
	}
	s.markLibraryScopeDirty(ctx, config.Library.ID, targetRoot, "targeted_refresh_queued")
	if s.workflowCapability() != nil {
		run, _, err := s.QueueLibraryWorkflow(ctx, QueueWorkflowInput{LibraryID: config.Library.ID, RootPath: targetRoot, Reason: WorkflowReasonTargetedRefresh, Priority: 8})
		return workflowRunCompatibilityJob(run), err
	}
	return database.Job{}, fmt.Errorf("workflow service unavailable")
}

func (s *Service) hasActiveScanWorkflow(ctx context.Context, libraryID uint) bool {
	if s.db == nil || libraryID == 0 {
		return false
	}
	var count int64
	err := s.db.WithContext(ctx).Model(&database.WorkflowRun{}).
		Where("library_id = ? AND status IN ? AND reason IN ?", libraryID, []string{workflow.RunStatusQueued, workflow.RunStatusRunning}, []string{WorkflowReasonCreateLibrary, WorkflowReasonManualScan, WorkflowReasonTargetedRefresh, WorkflowReasonScheduledScan, WorkflowReasonStorageRefresh}).
		Count(&count).Error
	return err == nil && count > 0
}

func (s *Service) QueueCatalogLibraryProjectionRefresh(ctx context.Context, libraryID uint, rootPath string) (database.Job, error) {
	if s.workflowCapability() != nil {
		run, err := s.queueStandaloneWorkflowTask(ctx, libraryID, rootPath, WorkflowReasonManualScan, workflow.TaskTypeRefreshProjection, workflow.StageProjection, fmt.Sprintf("projection:%s", rootPath), map[string]any{"library_id": libraryID, "root_path": rootPath})
		return workflowRunCompatibilityJob(run), err
	}
	if libraryID == 0 {
		return database.Job{}, fmt.Errorf("library id is required")
	}
	return database.Job{}, fmt.Errorf("workflow service unavailable")
}

func (s *Service) ListLibraries(ctx context.Context) ([]database.Library, error) {
	var libraries []database.Library
	if err := s.db.WithContext(ctx).Order("id asc").Find(&libraries).Error; err != nil {
		return nil, err
	}
	return libraries, nil
}

func (s *Service) ListLibraryDetails(ctx context.Context) ([]LibraryDetail, error) {
	libraries, err := s.ListLibraries(ctx)
	if err != nil {
		return nil, err
	}
	details := make([]LibraryDetail, 0, len(libraries))
	for _, record := range libraries {
		detail, err := s.GetLibrary(ctx, record.ID)
		if err != nil {
			return nil, err
		}
		details = append(details, detail)
	}
	return details, nil
}

func (s *Service) ListActiveLibraries(ctx context.Context) ([]database.Library, error) {
	var libraries []database.Library
	if err := s.db.WithContext(ctx).Where("status = ? AND scanner_enabled = ?", "active", true).Order("id asc").Find(&libraries).Error; err != nil {
		return nil, err
	}
	return libraries, nil
}

func (s *Service) DeleteLibrary(ctx context.Context, libraryID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return deleteLibraryRecords(ctx, tx, libraryID)
	})
}

func (s *Service) updateLibraryStatus(ctx context.Context, libraryID uint, status string) error {
	return s.db.WithContext(ctx).Model(&database.Library{}).Where("id = ?", libraryID).Update("status", status).Error
}

func deleteLibraryRecords(ctx context.Context, tx *gorm.DB, libraryID uint) error {
	var record database.Library
	if err := tx.WithContext(ctx).First(&record, libraryID).Error; err != nil {
		return err
	}
	if err := deleteLibraryDependentRecords(ctx, tx, libraryID); err != nil {
		return err
	}
	result := tx.WithContext(ctx).Where("id = ?", libraryID).Delete(&database.Library{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

type libraryDeleteQuery struct {
	query string
	args  []any
}

func deleteLibraryDependentRecords(ctx context.Context, tx *gorm.DB, libraryID uint) error {
	if err := cancelRunningLibraryJobs(ctx, tx, libraryID); err != nil {
		return err
	}
	if err := execLibraryDeleteQueries(ctx, tx, libraryDeleteQueries(libraryID)); err != nil {
		return err
	}
	if err := deleteLibraryModels(ctx, tx, libraryID); err != nil {
		return err
	}
	return execCleanupQueries(ctx, tx)
}

func libraryDeleteQueries(libraryID uint) []libraryDeleteQuery {
	queries := []string{
		`DELETE FROM content_shape_assignments WHERE library_id = ?`,
		`DELETE FROM content_shape_plans WHERE library_id = ?`,
		`DELETE FROM content_shape_profiles WHERE library_id = ?`,
		`DELETE FROM job_active_intents WHERE job_id IN (SELECT id FROM jobs WHERE payload_json LIKE '%"library_id":' || CAST(? AS TEXT) || ',%' OR payload_json LIKE '%"library_id":' || CAST(? AS TEXT) || '}%' OR payload_json IN (SELECT '{"inventory_file_id":' || CAST(id AS TEXT) || '}' FROM inventory_files WHERE library_id = ?))`,
		`DELETE FROM jobs WHERE status NOT IN ('running', 'cancel_requested') AND (payload_json LIKE '%"library_id":' || CAST(? AS TEXT) || ',%' OR payload_json LIKE '%"library_id":' || CAST(? AS TEXT) || '}%' OR payload_json IN (SELECT '{"inventory_file_id":' || CAST(id AS TEXT) || '}' FROM inventory_files WHERE library_id = ?))`,
		`DELETE FROM workflow_task_dependencies WHERE task_id IN (SELECT id FROM workflow_tasks WHERE library_id = ?) OR depends_on_task_id IN (SELECT id FROM workflow_tasks WHERE library_id = ?)`,
		`DELETE FROM workflow_task_leases WHERE task_id IN (SELECT id FROM workflow_tasks WHERE library_id = ?)`,
		`DELETE FROM workflow_resource_usages WHERE library_id = ? OR task_id IN (SELECT id FROM workflow_tasks WHERE library_id = ?)`,
		`DELETE FROM workflow_tasks WHERE library_id = ?`,
		`DELETE FROM workflow_runs WHERE library_id = ?`,
		`DELETE FROM schedule_runs WHERE schedule_id IN (SELECT id FROM schedules WHERE library_id = ?)`,
		`DELETE FROM library_scan_policies WHERE library_id = ?`,
		`DELETE FROM scan_exclusion_rules WHERE library_id = ?`,
		`DELETE FROM scan_exclusions WHERE library_id = ?`,
		`DELETE FROM library_metadata_policies WHERE library_id = ?`,
		`DELETE FROM library_metadata_strategies WHERE library_id = ?`,
		`DELETE FROM library_playback_policies WHERE library_id = ?`,
		`DELETE FROM library_subtitle_policies WHERE library_id = ?`,
		`DELETE FROM library_paths WHERE library_id = ?`,
		`DELETE FROM storage_observation_failures WHERE library_id = ?`,
		`DELETE FROM ingest_events WHERE library_id = ?`,
		`DELETE FROM ingest_conditions WHERE library_id = ?`,
		`DELETE FROM ingest_dirty_units WHERE library_id = ?`,
		`DELETE FROM media_streams WHERE file_id IN (SELECT id FROM inventory_files WHERE library_id = ?)`,
		`DELETE FROM resource_files WHERE inventory_file_id IN (SELECT id FROM inventory_files WHERE library_id = ?)`,
		`DELETE FROM resource_library_links WHERE library_id = ?`,
		`DELETE FROM user_resource_data WHERE resource_id IN (SELECT id FROM resources WHERE id NOT IN (SELECT DISTINCT resource_id FROM resource_files) AND id NOT IN (SELECT DISTINCT resource_id FROM resource_library_links))`,
		`UPDATE user_metadata_data SET preferred_resource_id = NULL WHERE preferred_resource_id IN (SELECT id FROM resources WHERE id NOT IN (SELECT DISTINCT resource_id FROM resource_files) AND id NOT IN (SELECT DISTINCT resource_id FROM resource_library_links))`,
		`DELETE FROM resource_metadata_links WHERE resource_id IN (SELECT id FROM resources WHERE id NOT IN (SELECT DISTINCT resource_id FROM resource_files) AND id NOT IN (SELECT DISTINCT resource_id FROM resource_library_links))`,
		`DELETE FROM resources WHERE id NOT IN (SELECT DISTINCT resource_id FROM resource_files) AND id NOT IN (SELECT DISTINCT resource_id FROM resource_library_links)`,
		`DELETE FROM library_metadata_projections WHERE library_id = ?`,
		`DELETE FROM library_search_documents WHERE library_id = ?`,
		`DELETE FROM metadata_operations WHERE library_id = ?`,
	}
	args := [][]any{
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID, libraryID, libraryID},
		{libraryID, libraryID, libraryID},
		{libraryID, libraryID},
		{libraryID},
		{libraryID, libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{},
		{},
		{},
		{},
		{libraryID},
		{libraryID},
		{libraryID},
	}
	result := make([]libraryDeleteQuery, 0, len(queries))
	for i, query := range queries {
		result = append(result, libraryDeleteQuery{query: query, args: args[i]})
	}
	return result
}

func execLibraryDeleteQueries(ctx context.Context, tx *gorm.DB, queries []libraryDeleteQuery) error {
	for _, query := range queries {
		if err := tx.WithContext(ctx).Exec(query.query, query.args...).Error; err != nil {
			return err
		}
	}
	return nil
}

func deleteLibraryModels(ctx context.Context, tx *gorm.DB, libraryID uint) error {
	modelDeletes := []struct {
		model any
		where string
		args  []any
	}{
		{&database.Schedule{}, "library_id = ?", []any{libraryID}},
		{&database.StorageIndexEntry{}, "library_id = ?", []any{libraryID}},
		{&database.InventoryFile{}, "library_id = ?", []any{libraryID}},
	}
	for _, deletion := range modelDeletes {
		if err := tx.WithContext(ctx).Unscoped().Where(deletion.where, deletion.args...).Delete(deletion.model).Error; err != nil {
			return err
		}
	}
	return nil
}

func execCleanupQueries(ctx context.Context, tx *gorm.DB) error {
	cleanupQueries := []string{
		`DELETE FROM media_streams WHERE file_id NOT IN (SELECT id FROM inventory_files)`,
		`DELETE FROM people WHERE id NOT IN (SELECT DISTINCT person_id FROM item_people)`,
		`DELETE FROM tags WHERE id NOT IN (SELECT DISTINCT tag_id FROM item_tags)`,
	}
	for _, query := range cleanupQueries {
		if err := tx.WithContext(ctx).Exec(query).Error; err != nil {
			return err
		}
	}
	return nil
}

func cancelRunningLibraryJobs(ctx context.Context, tx *gorm.DB, libraryID uint) error {
	now := time.Now().UTC()
	return tx.WithContext(ctx).Exec(
		`UPDATE jobs SET status = 'cancel_requested', error_message = 'library deleted; cancellation requested', updated_at = ? WHERE status = 'running' AND (payload_json LIKE '%"library_id":' || CAST(? AS TEXT) || ',%' OR payload_json LIKE '%"library_id":' || CAST(? AS TEXT) || '}%' OR payload_json IN (SELECT '{"inventory_file_id":' || CAST(id AS TEXT) || '}' FROM inventory_files WHERE library_id = ?))`,
		now,
		libraryID,
		libraryID,
		libraryID,
	).Error
}

func normalizePath(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || trimmed == "/" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "/" + trimmed
	}
	return trimmed
}

func normalizePathForProvider(providerName, input string) string {
	if strings.EqualFold(strings.TrimSpace(providerName), "local") {
		trimmed := strings.TrimSpace(input)
		if trimmed == "" {
			return "/"
		}
		return trimmed
	}
	return normalizePath(input)
}

func scopedRefreshRoot(providerName, libraryRoot, requestedRoot string) (string, error) {
	normalizedLibraryRoot := normalizePathForProvider(providerName, libraryRoot)
	trimmedRequested := strings.TrimSpace(requestedRoot)
	if trimmedRequested == "" {
		return normalizedLibraryRoot, nil
	}
	if strings.EqualFold(strings.TrimSpace(providerName), "local") {
		normalizedRequested := strings.TrimSpace(requestedRoot)
		if normalizedRequested == "" {
			return normalizedLibraryRoot, nil
		}
		cleanLibraryRoot := filepath.Clean(normalizedLibraryRoot)
		cleanRequested := filepath.Clean(normalizedRequested)
		rel, err := filepath.Rel(cleanLibraryRoot, cleanRequested)
		if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
			return "", fmt.Errorf("refresh root %s is outside library root %s", cleanRequested, cleanLibraryRoot)
		}
		return cleanRequested, nil
	}
	normalizedRequested := normalizePath(requestedRoot)
	if normalizedRequested == normalizedLibraryRoot || strings.HasPrefix(normalizedRequested, normalizedLibraryRoot+"/") {
		return normalizedRequested, nil
	}
	return "", fmt.Errorf("refresh root %s is outside library root %s", normalizedRequested, normalizedLibraryRoot)
}
