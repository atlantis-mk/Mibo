package library

import (
	"context"
	"strings"
)

func (s *Service) RunRecognitionResolveBatch(ctx context.Context, payload RecognitionResolveBatchPayload) error {
	return s.runRecognitionResolveBatch(ctx, payload, 0)
}

func (s *Service) runRecognitionResolveBatch(ctx context.Context, payload RecognitionResolveBatchPayload, followupRunID uint) error {
	config, err := s.EffectiveLibraryConfig(ctx, payload.LibraryID)
	if err != nil {
		return err
	}
	fileIDs := normalizeUintIDs(payload.FileIDs)
	if len(fileIDs) == 0 {
		return nil
	}
	rootPath := recognitionResolveBatchRootPath(config, payload.RootPath)
	libraryForPath := config.Library
	libraryForPath.RootPath = rootPath
	result, err := s.runRecognitionMaterializeBatchByFileIDs(ctx, libraryForPath, rootPath, fileIDs, payload.mode)
	if err != nil {
		return err
	}
	if followupRunID != 0 {
		if err := s.queueWorkflowPostRecognitionResolveTasks(ctx, followupRunID, config.Library.ID, rootPath, fileIDs, result.MetadataIDs, config.ScanPolicy); err != nil {
			return err
		}
		return nil
	}
	if _, err := s.QueueRecognitionPostResolve(ctx, config.Library.ID, rootPath, fileIDs, result.MetadataIDs); err != nil {
		return err
	}
	return nil
}

func recognitionResolveBatchRootPath(config EffectiveLibraryConfig, rootPath string) string {
	trimmed := strings.TrimSpace(rootPath)
	if trimmed != "" {
		return trimmed
	}
	return config.Library.RootPath
}
