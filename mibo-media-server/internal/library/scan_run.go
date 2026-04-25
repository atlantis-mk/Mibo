package library

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
)

func (s *Service) QueueLibraryScan(ctx context.Context, libraryID uint) (database.Job, error) {
	var record database.Library
	if err := s.db.WithContext(ctx).First(&record, libraryID).Error; err != nil {
		return database.Job{}, err
	}
	return s.jobs.EnqueueUnique(ctx, JobKindSyncLibrary, fmt.Sprintf("scan-library-%d", record.ID), map[string]any{"library_id": record.ID, "root_path": record.RootPath})
}

func (s *Service) RunSyncLibrary(ctx context.Context, job database.Job) error {
	type syncLibraryPayload struct {
		LibraryID uint   `json:"library_id"`
		RootPath  string `json:"root_path"`
	}
	var payload syncLibraryPayload
	if err := json.Unmarshal([]byte(job.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("decode sync_library payload: %w", err)
	}
	record, _, provider, err := s.providerForLibrary(ctx, payload.LibraryID)
	if err != nil {
		return err
	}
	rootPath := record.RootPath
	if payload.RootPath != "" {
		rootPath = normalizePath(payload.RootPath)
	}
	if err := s.updateLibraryStatus(ctx, record.ID, "syncing"); err != nil {
		return err
	}
	result, err := s.scanLibrary(ctx, provider, record, rootPath)
	if err != nil {
		_ = s.updateLibraryStatus(ctx, record.ID, "error")
		return err
	}
	if err := s.updateLibraryStatus(ctx, record.ID, "active"); err != nil {
		return err
	}
	if _, err := s.QueueLibrarySearchReindex(ctx, record.ID, rootPath); err != nil {
		return err
	}
	if _, err := s.QueueCatalogLibraryProjectionRefresh(ctx, record.ID, rootPath); err != nil {
		return err
	}
	_ = result
	return nil
}

func (s *Service) RunTargetedRefresh(ctx context.Context, job database.Job) error {
	var payload targetedRefreshPayload
	if err := json.Unmarshal([]byte(job.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("decode targeted_refresh payload: %w", err)
	}
	record, _, provider, err := s.providerForLibrary(ctx, payload.LibraryID)
	if err != nil {
		return err
	}
	rootPath, err := scopedRefreshRoot(provider.Name(), record.RootPath, payload.RootPath)
	if err != nil {
		return err
	}
	if err := s.updateLibraryStatus(ctx, record.ID, "syncing"); err != nil {
		return err
	}
	result, err := s.scanLibraryWithMode(ctx, provider, record, rootPath, scanMode{partial: true, rootPath: rootPath})
	if err != nil {
		_ = s.updateLibraryStatus(ctx, record.ID, "error")
		return err
	}
	if err := s.updateLibraryStatus(ctx, record.ID, "active"); err != nil {
		return err
	}
	if _, err := s.QueueLibrarySearchReindex(ctx, record.ID, rootPath); err != nil {
		return err
	}
	if _, err := s.QueueCatalogLibraryProjectionRefresh(ctx, record.ID, rootPath); err != nil {
		return err
	}
	_ = result
	return nil
}

func (s *Service) scanLibrary(ctx context.Context, provider storage.Provider, library database.Library, rootPath string) (SyncResult, error) {
	return s.scanLibraryWithMode(ctx, provider, library, rootPath, scanMode{})
}

func (s *Service) scanLibraryWithMode(ctx context.Context, provider storage.Provider, library database.Library, rootPath string, mode scanMode) (SyncResult, error) {
	resolved, err := provider.ResolveStorage(ctx, storage.ResolveStorageRequest{Path: rootPath})
	if err != nil {
		return SyncResult{}, fmt.Errorf("resolve library root: %w", err)
	}
	if !resolved.Object.IsDir {
		return SyncResult{}, fmt.Errorf("library root %s is not a directory", rootPath)
	}
	seenFiles := make(map[string]struct{})
	seenItems := make(map[string]struct{})
	result := SyncResult{}
	if err := s.walkDirectory(ctx, provider, library, rootPath, seenFiles, seenItems, &result); err != nil {
		return SyncResult{}, err
	}
	if mode.partial {
		if err := s.cleanupMissingFilesInScope(ctx, library.ID, mode.rootPath, seenFiles); err != nil {
			return SyncResult{}, err
		}
		if err := s.cleanupMissingItemsInScope(ctx, library.ID, mode.rootPath, seenItems); err != nil {
			return SyncResult{}, err
		}
	} else {
		if err := s.cleanupMissingFiles(ctx, library.ID, seenFiles); err != nil {
			return SyncResult{}, err
		}
		if err := s.cleanupMissingItems(ctx, library.ID, seenItems); err != nil {
			return SyncResult{}, err
		}
	}
	return result, nil
}

func (s *Service) walkDirectory(ctx context.Context, provider storage.Provider, library database.Library, dirPath string, seenFiles map[string]struct{}, seenItems map[string]struct{}, result *SyncResult) error {
	result.DirectoriesScanned++
	objects, err := s.listAllDirectoryObjects(ctx, provider, dirPath)
	if err != nil {
		return fmt.Errorf("list directory %s: %w", dirPath, err)
	}
	sort.Slice(objects, func(i, j int) bool { return objects[i].Path < objects[j].Path })
	for _, object := range objects {
		if object.IsDir {
			if err := s.walkDirectory(ctx, provider, library, object.Path, seenFiles, seenItems, result); err != nil {
				return err
			}
			continue
		}
		if !isVideoFile(object.Path) {
			continue
		}
		result.FilesSeen++
		seenFiles[object.Path] = struct{}{}
		classified := classifyMediaFile(library.Type, object)
		seenItems[classified.SourcePath] = struct{}{}
		item, createdItem, err := s.upsertMediaItem(ctx, library.ID, classified)
		if err != nil {
			return err
		}
		if createdItem {
			result.MediaItemsUpserted++
		}
		if item.MatchStatus == "pending" {
			if _, err := s.QueueMediaItemMatch(ctx, item.ID, false); err != nil {
				return err
			}
		}
		fileRecord, createdFile, err := s.upsertMediaFile(ctx, library.ID, item.ID, object)
		if err != nil {
			return err
		}
		if createdFile {
			result.MediaFilesUpserted++
		}
		if fileRecord.ProbeStatus == "pending" {
			if _, err := s.QueueMediaFileProbe(ctx, fileRecord.ID, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) listAllDirectoryObjects(ctx context.Context, provider storage.Provider, dirPath string) ([]storage.Object, error) {
	const pageSize = 1000
	var all []storage.Object
	for page := 1; ; page++ {
		objects, err := provider.List(ctx, storage.ListRequest{Path: dirPath, Refresh: true, Page: page, PerPage: pageSize})
		if err != nil {
			return nil, err
		}
		all = append(all, objects...)
		if len(objects) < pageSize {
			break
		}
	}
	return all, nil
}
