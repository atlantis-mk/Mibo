package library

import (
	"context"
	"fmt"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/schedule"
	"github.com/atlan/mibo-media-server/internal/storage"
)

type ScheduledJobResult struct {
	LibrariesProcessed int    `json:"libraries_processed"`
	ItemsChecked       int    `json:"items_checked,omitempty"`
	Failures           int    `json:"failures,omitempty"`
	Summary            string `json:"summary"`
}

func (s *Service) RunScheduledScan(ctx context.Context, due schedule.DueSchedule) (ScheduledJobResult, error) {
	return s.runScheduledLibraryTraversal(ctx, due, "scan", func(ctx context.Context, libraryRecord database.Library, provider storage.Provider) error {
		_, err := s.scanLibrary(ctx, provider, libraryRecord, libraryRecord.RootPath)
		return err
	})
}

func (s *Service) RunScheduledCleanup(ctx context.Context, due schedule.DueSchedule) (ScheduledJobResult, error) {
	return s.runScheduledLibraryTraversal(ctx, due, "cleanup", func(ctx context.Context, libraryRecord database.Library, provider storage.Provider) error {
		_, err := s.scanLibrary(ctx, provider, libraryRecord, libraryRecord.RootPath)
		return err
	})
}

func (s *Service) RunScheduledInvalidLinkCheck(ctx context.Context, due schedule.DueSchedule) (ScheduledJobResult, error) {
	libraries, err := s.resolveScheduledLibraries(ctx, due)
	if err != nil {
		return ScheduledJobResult{}, err
	}
	result := ScheduledJobResult{LibrariesProcessed: len(libraries)}
	for _, libraryRecord := range libraries {
		_, _, provider, err := s.providerForLibrary(ctx, libraryRecord.ID)
		if err != nil {
			return ScheduledJobResult{}, err
		}
		if _, err := provider.ResolveStorage(ctx, storage.ResolveStorageRequest{Path: libraryRecord.RootPath}); err != nil {
			result.Failures++
			continue
		}
		var files []database.MediaFile
		if err := s.db.WithContext(ctx).
			Where("library_id = ? AND deleted_at IS NULL", libraryRecord.ID).
			Order("id asc").
			Find(&files).Error; err != nil {
			return ScheduledJobResult{}, err
		}
		for _, file := range files {
			result.ItemsChecked++
			if _, err := provider.ResolveStorage(ctx, storage.ResolveStorageRequest{Path: file.StoragePath}); err != nil {
				result.Failures++
			}
		}
	}
	result.Summary = fmt.Sprintf("checked %d libraries and %d files; %d invalid links found", result.LibrariesProcessed, result.ItemsChecked, result.Failures)
	return result, nil
}

func (s *Service) runScheduledLibraryTraversal(ctx context.Context, due schedule.DueSchedule, label string, fn func(context.Context, database.Library, storage.Provider) error) (ScheduledJobResult, error) {
	libraries, err := s.resolveScheduledLibraries(ctx, due)
	if err != nil {
		return ScheduledJobResult{}, err
	}
	for _, libraryRecord := range libraries {
		_, _, provider, err := s.providerForLibrary(ctx, libraryRecord.ID)
		if err != nil {
			return ScheduledJobResult{}, err
		}
		if err := fn(ctx, libraryRecord, provider); err != nil {
			return ScheduledJobResult{}, err
		}
	}
	return ScheduledJobResult{LibrariesProcessed: len(libraries), Summary: fmt.Sprintf("%s completed for %d libraries", label, len(libraries))}, nil
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
