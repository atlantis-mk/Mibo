package library

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/settings"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

func (s *Service) CreateLibrary(ctx context.Context, input CreateLibraryInput) (database.Library, database.Job, error) {
	if strings.TrimSpace(input.Name) == "" {
		return database.Library{}, database.Job{}, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(input.Type) == "" {
		return database.Library{}, database.Job{}, fmt.Errorf("type is required")
	}
	if input.MediaSourceID == 0 {
		return database.Library{}, database.Job{}, fmt.Errorf("media_source_id is required")
	}
	var source database.MediaSource
	if err := s.db.WithContext(ctx).First(&source, input.MediaSourceID).Error; err != nil {
		return database.Library{}, database.Job{}, err
	}
	rootPath := normalizePath(input.RootPath)
	if rootPath == "/" {
		rootPath = source.RootPath
	}
	rootPath = normalizePathForProvider(source.Provider, rootPath)
	provider, err := s.storage.BuildForSource(source)
	if err != nil {
		return database.Library{}, database.Job{}, err
	}
	if _, err := provider.ResolveStorage(ctx, storage.ResolveStorageRequest{Path: rootPath}); err != nil {
		return database.Library{}, database.Job{}, fmt.Errorf("resolve library root %s: %w", rootPath, err)
	}
	library := database.Library{Name: strings.TrimSpace(input.Name), Type: normalizeLibraryType(input.Type), MediaSourceID: source.ID, RootPath: rootPath, Status: "pending", ScannerEnabled: true}
	if err := s.db.WithContext(ctx).Create(&library).Error; err != nil {
		return database.Library{}, database.Job{}, err
	}
	if err := s.db.WithContext(ctx).Create(&database.LibraryPath{LibraryID: library.ID, MediaSourceID: source.ID, RootPath: rootPath, DisplayName: library.Name, Enabled: true}).Error; err != nil {
		return database.Library{}, database.Job{}, err
	}
	if err := database.EnsureLibraryPolicyDefaults(s.db.WithContext(ctx), library.ID); err != nil {
		return database.Library{}, database.Job{}, err
	}
	if input.Scan != nil {
		if _, err := s.UpdateLibraryScanPolicy(ctx, library.ID, *input.Scan); err != nil {
			return database.Library{}, database.Job{}, err
		}
	}
	if input.Metadata != nil {
		if _, err := s.UpdateLibraryMetadataPolicy(ctx, library.ID, *input.Metadata); err != nil {
			return database.Library{}, database.Job{}, err
		}
	}
	if input.MetadataStrategy != nil {
		if _, err := settings.NewService(s.db, s.cfg.Metadata).UpdateLibraryMetadataStrategy(ctx, library.ID, *input.MetadataStrategy); err != nil {
			return database.Library{}, database.Job{}, err
		}
	}
	if input.Playback != nil {
		if _, err := s.UpdateLibraryPlaybackPolicy(ctx, library.ID, *input.Playback); err != nil {
			return database.Library{}, database.Job{}, err
		}
	}
	if input.Subtitle != nil {
		if _, err := s.UpdateLibrarySubtitlePolicy(ctx, library.ID, *input.Subtitle); err != nil {
			return database.Library{}, database.Job{}, err
		}
	}
	if input.ScanExclusionRules != nil {
		if _, err := s.ReplaceLibraryScanExclusionRules(ctx, library.ID, input.ScanExclusionRules, nil); err != nil {
			return database.Library{}, database.Job{}, err
		}
	}
	job, err := s.QueueLibraryScan(ctx, library.ID)
	if err != nil {
		return database.Library{}, database.Job{}, err
	}
	return library, job, nil
}

func (s *Service) QueueTargetedRefresh(ctx context.Context, libraryID uint, rootPath, reason string) (database.Job, error) {
	config, err := s.EffectiveLibraryConfig(ctx, libraryID)
	if err != nil {
		return database.Job{}, err
	}
	if s.jobs == nil {
		return database.Job{}, fmt.Errorf("jobs service unavailable")
	}
	normalizedReason := strings.TrimSpace(strings.ToLower(reason))
	if normalizedReason == "" {
		normalizedReason = "manual"
	}
	_, _, targetRoot, err := s.scopedRefreshPath(ctx, config, rootPath)
	if err != nil {
		return database.Job{}, err
	}
	jobKey := fmt.Sprintf("targeted-refresh:%d:%s:%s", config.Library.ID, targetRoot, normalizedReason)
	return s.jobs.EnqueueUnique(ctx, JobKindTargetedRefresh, jobKey, targetedRefreshPayload{LibraryID: config.Library.ID, RootPath: targetRoot, Reason: normalizedReason})
}

func (s *Service) QueueCatalogItemProjectionRefresh(ctx context.Context, itemID uint) (database.Job, error) {
	if s.jobs == nil {
		return database.Job{}, fmt.Errorf("jobs service unavailable")
	}
	if itemID == 0 {
		return database.Job{}, fmt.Errorf("item id is required")
	}
	payload := catalog.ItemProjectionRefreshPayload{ItemID: itemID}
	jobKey := fmt.Sprintf("catalog-refresh-item-projection:%d", itemID)
	return s.jobs.EnqueueUnique(ctx, JobKindCatalogRefreshItemProjection, jobKey, payload)
}

func (s *Service) QueueCatalogLibraryProjectionRefresh(ctx context.Context, libraryID uint, rootPath string) (database.Job, error) {
	if s.jobs == nil {
		return database.Job{}, fmt.Errorf("jobs service unavailable")
	}
	if libraryID == 0 {
		return database.Job{}, fmt.Errorf("library id is required")
	}
	payload := catalog.LibraryProjectionRefreshPayload{LibraryID: libraryID, RootPath: strings.TrimSpace(rootPath)}
	jobKey := fmt.Sprintf("catalog-refresh-library-projection:%d:%s", libraryID, payload.RootPath)
	return s.jobs.EnqueueUnique(ctx, JobKindCatalogRefreshLibraryProjection, jobKey, payload)
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

func deleteLibraryDependentRecords(ctx context.Context, tx *gorm.DB, libraryID uint) error {
	queries := []string{
		`DELETE FROM job_active_intents WHERE job_id IN (SELECT id FROM jobs WHERE payload_json LIKE '%"library_id":' || CAST(? AS TEXT) || ',%' OR payload_json LIKE '%"library_id":' || CAST(? AS TEXT) || '}%' OR payload_json IN (SELECT '{"inventory_file_id":' || CAST(id AS TEXT) || '}' FROM inventory_files WHERE library_id = ?) OR payload_json IN (SELECT '{"item_id":' || CAST(id AS TEXT) || '}' FROM catalog_items WHERE library_id = ?))`,
		`DELETE FROM jobs WHERE payload_json LIKE '%"library_id":' || CAST(? AS TEXT) || ',%' OR payload_json LIKE '%"library_id":' || CAST(? AS TEXT) || '}%' OR payload_json IN (SELECT '{"inventory_file_id":' || CAST(id AS TEXT) || '}' FROM inventory_files WHERE library_id = ?) OR payload_json IN (SELECT '{"item_id":' || CAST(id AS TEXT) || '}' FROM catalog_items WHERE library_id = ?)`,
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
		`DELETE FROM media_streams WHERE file_id IN (SELECT id FROM inventory_files WHERE library_id = ?)`,
		`DELETE FROM asset_files WHERE asset_id IN (SELECT id FROM media_assets WHERE library_id = ?) OR file_id IN (SELECT id FROM inventory_files WHERE library_id = ?)`,
		`DELETE FROM asset_items WHERE asset_id IN (SELECT id FROM media_assets WHERE library_id = ?) OR item_id IN (SELECT id FROM catalog_items WHERE library_id = ?)`,
		`DELETE FROM user_item_data WHERE item_id IN (SELECT id FROM catalog_items WHERE library_id = ?) OR asset_id IN (SELECT id FROM media_assets WHERE library_id = ?)`,
		`DELETE FROM catalog_external_ids WHERE item_id IN (SELECT id FROM catalog_items WHERE library_id = ?)`,
		`DELETE FROM catalog_identities WHERE item_id IN (SELECT id FROM catalog_items WHERE library_id = ?)`,
		`DELETE FROM metadata_operations WHERE library_id = ? OR origin_item_id IN (SELECT id FROM catalog_items WHERE library_id = ?) OR target_item_id IN (SELECT id FROM catalog_items WHERE library_id = ?)`,
		`DELETE FROM metadata_field_states WHERE item_id IN (SELECT id FROM catalog_items WHERE library_id = ?)`,
		`DELETE FROM metadata_sources WHERE item_id IN (SELECT id FROM catalog_items WHERE library_id = ?)`,
		`DELETE FROM item_images WHERE item_id IN (SELECT id FROM catalog_items WHERE library_id = ?)`,
		`DELETE FROM item_people WHERE item_id IN (SELECT id FROM catalog_items WHERE library_id = ?)`,
		`DELETE FROM item_tags WHERE item_id IN (SELECT id FROM catalog_items WHERE library_id = ?)`,
		`DELETE FROM item_rollups WHERE item_id IN (SELECT id FROM catalog_items WHERE library_id = ?)`,
		`DELETE FROM catalog_search_documents WHERE library_id = ? OR item_id IN (SELECT id FROM catalog_items WHERE library_id = ?)`,
	}
	args := [][]any{
		{libraryID, libraryID, libraryID, libraryID},
		{libraryID, libraryID, libraryID, libraryID},
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
		{libraryID, libraryID},
		{libraryID, libraryID},
		{libraryID, libraryID},
		{libraryID},
		{libraryID},
		{libraryID, libraryID, libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID},
		{libraryID, libraryID},
	}
	for i, query := range queries {
		if err := tx.WithContext(ctx).Exec(query, args[i]...).Error; err != nil {
			return err
		}
	}
	if err := deleteLegacyLibraryRecords(ctx, tx, libraryID); err != nil {
		return err
	}

	modelDeletes := []struct {
		model any
		where string
		args  []any
	}{
		{&database.Schedule{}, "library_id = ?", []any{libraryID}},
		{&database.StorageIndexEntry{}, "library_id = ?", []any{libraryID}},
		{&database.MediaAsset{}, "library_id = ?", []any{libraryID}},
		{&database.InventoryFile{}, "library_id = ?", []any{libraryID}},
		{&database.CatalogItem{}, "library_id = ?", []any{libraryID}},
	}
	for _, deletion := range modelDeletes {
		if err := tx.WithContext(ctx).Unscoped().Where(deletion.where, deletion.args...).Delete(deletion.model).Error; err != nil {
			return err
		}
	}

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

func deleteLegacyLibraryRecords(ctx context.Context, tx *gorm.DB, libraryID uint) error {
	if tx.Migrator().HasTable("playback_progresses") && tx.Migrator().HasTable("media_items") && tx.Migrator().HasTable("media_files") {
		if err := tx.WithContext(ctx).Exec(`DELETE FROM playback_progresses WHERE media_item_id IN (SELECT id FROM media_items WHERE library_id = ?) OR media_file_id IN (SELECT id FROM media_files WHERE library_id = ?)`, libraryID, libraryID).Error; err != nil {
			return err
		}
	}
	if tx.Migrator().HasTable("media_files") {
		if err := tx.WithContext(ctx).Exec(`DELETE FROM media_files WHERE library_id = ?`, libraryID).Error; err != nil {
			return err
		}
	}
	if tx.Migrator().HasTable("media_items") {
		if err := tx.WithContext(ctx).Exec(`DELETE FROM media_items WHERE library_id = ?`, libraryID).Error; err != nil {
			return err
		}
	}
	return nil
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
