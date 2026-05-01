package library

import (
	"context"
	"fmt"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/schedule"
	"github.com/atlan/mibo-media-server/internal/settings"
	"github.com/atlan/mibo-media-server/internal/storage"
)

type ScheduledJobResult struct {
	LibrariesProcessed   int    `json:"libraries_processed"`
	ItemsChecked         int    `json:"items_checked,omitempty"`
	Failures             int    `json:"failures,omitempty"`
	FilesDeleted         int    `json:"files_deleted,omitempty"`
	AssetsDeleted        int    `json:"assets_deleted,omitempty"`
	CatalogItemsDeleted  int    `json:"catalog_items_deleted,omitempty"`
	DependentRowsDeleted int    `json:"dependent_rows_deleted,omitempty"`
	Summary              string `json:"summary"`
}

func (s *Service) RunScheduledScan(ctx context.Context, due schedule.DueSchedule) (ScheduledJobResult, error) {
	return s.runScheduledLibraryTraversal(ctx, due, "scan", func(ctx context.Context, libraryRecord database.Library, pathRecord database.LibraryPath, provider storage.Provider) error {
		libraryForPath := libraryRecord
		libraryForPath.MediaSourceID = pathRecord.MediaSourceID
		libraryForPath.RootPath = pathRecord.RootPath
		_, err := s.scanLibrary(ctx, provider, libraryForPath, pathRecord.RootPath)
		return err
	})
}

func (s *Service) RunScheduledCleanup(ctx context.Context, due schedule.DueSchedule) (ScheduledJobResult, error) {
	cleanupSettings, err := settings.ResolveCleanupSettings(ctx, s.db, s.cfg.Cleanup)
	if err != nil {
		return ScheduledJobResult{}, err
	}
	if !cleanupSettings.MissingCleanupEnabled {
		libraries, err := s.resolveScheduledLibraries(ctx, due)
		if err != nil {
			return ScheduledJobResult{}, err
		}
		return ScheduledJobResult{LibrariesProcessed: len(libraries), Summary: "missing cleanup disabled"}, nil
	}
	libraries, err := s.resolveScheduledLibraries(ctx, due)
	if err != nil {
		return ScheduledJobResult{}, err
	}
	libraryIDs := make([]uint, 0, len(libraries))
	for _, libraryRecord := range libraries {
		libraryIDs = append(libraryIDs, libraryRecord.ID)
	}
	return s.runMissingMediaCleanupForLibraries(ctx, libraryIDs, "")
}

func (s *Service) RunScheduledInvalidLinkCheck(ctx context.Context, due schedule.DueSchedule) (ScheduledJobResult, error) {
	libraries, err := s.resolveScheduledLibraries(ctx, due)
	if err != nil {
		return ScheduledJobResult{}, err
	}
	result := ScheduledJobResult{LibrariesProcessed: len(libraries)}
	for _, libraryRecord := range libraries {
		config, err := s.EffectiveLibraryConfig(ctx, libraryRecord.ID)
		if err != nil {
			return ScheduledJobResult{}, err
		}
		for _, pathRecord := range config.Paths {
			provider, err := s.providerForLibraryPath(ctx, pathRecord)
			if err != nil {
				return ScheduledJobResult{}, err
			}
			if _, err := provider.ResolveStorage(ctx, storage.ResolveStorageRequest{Path: pathRecord.RootPath}); err != nil {
				result.Failures++
				continue
			}
		}
		var files []database.InventoryFile
		if err := s.db.WithContext(ctx).
			Where("library_id = ? AND deleted_at IS NULL", libraryRecord.ID).
			Order("id asc").
			Find(&files).Error; err != nil {
			return ScheduledJobResult{}, err
		}
		for _, file := range files {
			result.ItemsChecked++
			provider, err := s.providerForInventoryFile(ctx, config, file)
			if err != nil {
				result.Failures++
				continue
			}
			if _, err := provider.ResolveStorage(ctx, storage.ResolveStorageRequest{Path: file.StoragePath}); err != nil {
				result.Failures++
			}
		}
	}
	result.Summary = fmt.Sprintf("checked %d libraries and %d files; %d invalid links found", result.LibrariesProcessed, result.ItemsChecked, result.Failures)
	return result, nil
}

func (s *Service) runScheduledLibraryTraversal(ctx context.Context, due schedule.DueSchedule, label string, fn func(context.Context, database.Library, database.LibraryPath, storage.Provider) error) (ScheduledJobResult, error) {
	libraries, err := s.resolveScheduledLibraries(ctx, due)
	if err != nil {
		return ScheduledJobResult{}, err
	}
	for _, libraryRecord := range libraries {
		config, err := s.EffectiveLibraryConfig(ctx, libraryRecord.ID)
		if err != nil {
			return ScheduledJobResult{}, err
		}
		for _, pathRecord := range config.Paths {
			provider, err := s.providerForLibraryPath(ctx, pathRecord)
			if err != nil {
				return ScheduledJobResult{}, err
			}
			if err := fn(ctx, config.Library, pathRecord, provider); err != nil {
				return ScheduledJobResult{}, err
			}
		}
	}
	return ScheduledJobResult{LibrariesProcessed: len(libraries), Summary: fmt.Sprintf("%s completed for %d libraries", label, len(libraries))}, nil
}

func (s *Service) providerForInventoryFile(ctx context.Context, config EffectiveLibraryConfig, file database.InventoryFile) (storage.Provider, error) {
	for _, pathRecord := range config.Paths {
		if strings.HasPrefix(file.StoragePath, strings.TrimRight(pathRecord.RootPath, "/")+"/") || file.StoragePath == pathRecord.RootPath {
			return s.providerForLibraryPath(ctx, pathRecord)
		}
	}
	return s.providerForLibraryPath(ctx, database.LibraryPath{MediaSourceID: config.Library.MediaSourceID, RootPath: config.Library.RootPath})
}

func (s *Service) resolveScheduledLibraries(ctx context.Context, due schedule.DueSchedule) ([]database.Library, error) {
	switch due.ScopeKind {
	case schedule.ScopeGlobal:
		return s.ListActiveLibraries(ctx)
	case schedule.ScopeLibrary:
		if due.LibraryID == nil || *due.LibraryID == 0 {
			return nil, fmt.Errorf("library scope requires library_id")
		}
		var libraryRecord database.Library
		if err := s.db.WithContext(ctx).
			Where("id = ? AND status = ? AND scanner_enabled = ?", *due.LibraryID, "active", true).
			First(&libraryRecord).Error; err != nil {
			return nil, err
		}
		return []database.Library{libraryRecord}, nil
	default:
		return nil, fmt.Errorf("unsupported schedule scope %q", strings.TrimSpace(string(due.ScopeKind)))
	}
}
