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
	if err := s.cleanupMissingCatalog(ctx, library.ID, rootPath, seenFiles); err != nil {
		return SyncResult{}, err
	}
	_ = seenFiles
	_ = seenItems
	_ = mode
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
		classified := classifyMediaFile(library.Type, library.RootPath, object)
		artifact, itemPaths := catalogScanArtifactFromObject(provider.Name(), object, classified)
		for _, itemPath := range itemPaths {
			seenItems[itemPath] = struct{}{}
		}
		writeResult, err := s.writeCatalogScan(ctx, library, artifact)
		if err != nil {
			return err
		}
		if writeResult.File.ID != 0 {
			result.InventoryFilesSeen++
			if writeResult.Item.ID != 0 {
				result.CatalogItemsSeen++
			}
		}
		if writeResult.Item.ID != 0 {
			if _, err := s.QueueCatalogItemMatch(ctx, writeResult.Item.ID); err != nil {
				return err
			}
		}
		if writeResult.File.ID != 0 {
			if _, err := s.QueueInventoryFileProbe(ctx, writeResult.File.ID, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func catalogScanArtifactFromObject(storageProvider string, object storage.Object, classified classifiedMedia) (catalogScanArtifact, []string) {
	artifact := catalogScanArtifact{
		SourcePath:        object.Path,
		Title:             classified.Title,
		OriginalTitle:     classified.OriginalTitle,
		SeriesTitle:       classified.SeriesTitle,
		Year:              classified.Year,
		SeasonNumber:      classified.SeasonNumber,
		StorageProvider:   strings.TrimSpace(storageProvider),
		StableIdentityKey: strings.TrimSpace(object.StableIdentity),
		ProviderName:      strings.TrimSpace(object.Provider),
		HashesJSON:        encodeHashInfo(object.HashInfo),
		SizeBytes:         object.Size,
		ModifiedAt:        object.Modified,
		Container:         strings.TrimPrefix(strings.ToLower(path.Ext(object.Path)), "."),
	}

	if classified.Type == "episode" {
		artifact.ItemType = catalog.ItemTypeEpisode
		artifact.SeriesPath = canonicalSeriesPath(classified.SeriesTitle)
		if classified.SeasonNumber != nil {
			artifact.SeasonPath = fmt.Sprintf("%s/season-%02d", artifact.SeriesPath, *classified.SeasonNumber)
		}
		episodeNumbers := append([]int(nil), classified.EpisodeNumbers...)
		if len(episodeNumbers) == 0 && classified.EpisodeNumber != nil {
			episodeNumbers = append(episodeNumbers, *classified.EpisodeNumber)
		}
		itemPaths := make([]string, 0, len(episodeNumbers)+2)
		if artifact.SeriesPath != "" {
			itemPaths = append(itemPaths, artifact.SeriesPath)
		}
		if artifact.SeasonPath != "" {
			itemPaths = append(itemPaths, artifact.SeasonPath)
		}
		for _, episodeNumber := range episodeNumbers {
			itemPath := canonicalEpisodeItemPath(artifact.SeasonPath, episodeNumber)
			artifact.EpisodeSlots = append(artifact.EpisodeSlots, catalogEpisodeSlot{EpisodeNumber: episodeNumber, ItemPath: itemPath})
			itemPaths = append(itemPaths, itemPath)
		}
		return artifact, itemPaths
	}

	artifact.ItemType = catalog.ItemTypeMovie
	artifact.ItemPath = classified.SourcePath
	return artifact, []string{artifact.ItemPath}
}

func encodeHashInfo(hashInfo map[string]string) string {
	if len(hashInfo) == 0 {
		return ""
	}
	encoded, err := json.Marshal(hashInfo)
	if err != nil {
		return ""
	}
	return string(encoded)
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
