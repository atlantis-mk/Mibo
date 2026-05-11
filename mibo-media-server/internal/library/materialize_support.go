package library

import (
	"context"
	"fmt"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
)

func (s *Service) RunRecognitionPostResolve(ctx context.Context, payload RecognitionPostResolvePayload) error {
	config, err := s.EffectiveLibraryConfig(ctx, payload.LibraryID)
	if err != nil {
		return err
	}
	rootPath := strings.TrimSpace(payload.RootPath)
	if rootPath == "" {
		rootPath = config.Library.RootPath
	}
	metadataItemIDs := normalizeUintIDs(payload.MetadataItemIDs)
	fileIDs := normalizeUintIDs(payload.FileIDs)
	if len(metadataItemIDs) > 0 {
		if _, err := s.QueueMetadataMatchBatch(ctx, config.Library.ID, rootPath, metadataItemIDs); err != nil {
			return err
		}
	}
	if len(fileIDs) > 0 && config.InventoryProbeBatchEnabled() {
		if _, err := s.QueueInventoryProbeBatch(ctx, config.Library.ID, rootPath, fileIDs); err != nil {
			return err
		}
	}
	if _, err := s.QueueCatalogLibraryProjectionRefresh(ctx, config.Library.ID, rootPath); err != nil {
		return err
	}
	s.markProjectionLibraryDirty(ctx, config.Library.ID, rootPath, "materialization_completed")
	return nil
}

func (s *Service) RunInventoryProbeBatch(ctx context.Context, payload InventoryProbeBatchPayload) error {
	executor := s.inventoryProbeCapability()
	if executor == nil {
		return fmt.Errorf("probe executor unavailable for workflow batch")
	}
	for _, fileID := range normalizeUintIDs(payload.FileIDs) {
		if err := executor(ctx, fileID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) RunMetadataMatchBatch(ctx context.Context, payload MetadataMatchBatchPayload) error {
	executor := s.metadataMatchCapability()
	if executor == nil {
		return fmt.Errorf("metadata match executor unavailable for workflow batch")
	}
	ids, err := s.filterMetadataMatchableItemIDs(ctx, normalizeUintIDs(payload.MetadataItemIDs))
	if err != nil {
		return err
	}
	for _, metadataItemID := range ids {
		if err := executor(ctx, metadataItemID, payload.LibraryID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) filterMetadataMatchableItemIDs(ctx context.Context, ids []uint) ([]uint, error) {
	ids = normalizeUintIDs(ids)
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []struct {
		ID       uint
		ItemType string
	}
	if err := s.db.WithContext(ctx).Model(&database.MetadataItem{}).Select("id, item_type").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	allowed := make(map[uint]struct{}, len(rows))
	for _, row := range rows {
		switch strings.TrimSpace(row.ItemType) {
		case database.MetadataItemTypeMovie, database.MetadataItemTypeSeries:
			allowed[row.ID] = struct{}{}
		}
	}
	filtered := make([]uint, 0, len(allowed))
	for _, id := range ids {
		if _, ok := allowed[id]; ok {
			filtered = append(filtered, id)
		}
	}
	return filtered, nil
}

func (s *Service) hydrateRecognitionFileSignals(ctx context.Context, batchState *recognitionBatchState, files []database.InventoryFile) error {
	if batchState == nil || len(files) == 0 {
		return nil
	}
	settings := contentShapeSettingsFromConfig(s.cfg)
	filesByProvider := make(map[string][]database.InventoryFile)
	for _, file := range files {
		if file.ID == 0 || file.Status != inventory.FileStatusAvailable || file.ContentClass != SourceContentClassVideo || !isVideoFile(file.StoragePath) {
			continue
		}
		provider := strings.TrimSpace(file.StorageProvider)
		if provider == "" || strings.TrimSpace(file.StoragePath) == "" {
			continue
		}
		filesByProvider[provider] = append(filesByProvider[provider], file)
	}
	for provider, providerFiles := range filesByProvider {
		scope := inventoryFileSignalScope{LibraryID: providerFiles[0].LibraryID, StorageProvider: provider, ClassifierVersion: settings.ClassifierVersion}
		models, _, err := loadReusableInventoryFileSignals(ctx, s.db, scope, providerFiles)
		if err != nil {
			return err
		}
		hydrateFilenameTokenCacheFromSignals(batchState.tokenProfileCache, models)
		for storagePath, model := range models {
			batchState.indexedSignalsByPath[storagePath] = model
		}
		missing := make([]inventoryFileSignalInput, 0)
		for _, file := range providerFiles {
			storagePath := strings.TrimSpace(file.StoragePath)
			if _, ok := models[storagePath]; ok {
				continue
			}
			model := filenameTokenProfileForPath(batchState.tokenProfileCache, storagePath)
			missing = append(missing, inventoryFileSignalInput{File: file, Model: model})
			batchState.indexedSignalsByPath[storagePath] = model
		}
		if err := saveInventoryFileSignals(ctx, s.db, scope, missing); err != nil {
			return err
		}
	}
	return nil
}

func chunkUints(values []uint, size int) [][]uint {
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

func chunkStrings(values []string, size int) [][]string {
	if len(values) == 0 {
		return nil
	}
	if size <= 0 {
		size = len(values)
	}
	chunks := make([][]string, 0, (len(values)+size-1)/size)
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[start:end])
	}
	return chunks
}

func (s *Service) flushRecognitionIngestBatch(ctx context.Context, dirtyReasons map[uint]string, events []database.IngestEvent) error {
	ingestSvc := s.ingestCapability()
	if ingestSvc == nil {
		return nil
	}
	grouped := make(map[string][]uint)
	for fileID, reason := range dirtyReasons {
		grouped[strings.TrimSpace(reason)] = append(grouped[strings.TrimSpace(reason)], fileID)
	}
	for reason, fileIDs := range grouped {
		if err := ingestSvc.MarkInventoryFilesDirty(ctx, fileIDs, reason); err != nil {
			return err
		}
	}
	if err := ingestSvc.AppendEvents(ctx, events); err != nil {
		return err
	}
	return nil
}
