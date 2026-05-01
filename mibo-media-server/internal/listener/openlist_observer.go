package listener

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
	"github.com/atlan/mibo-media-server/internal/storageindex"
)

const openListObserverPollInterval = 5 * time.Minute

func (s *Service) StartOpenListObserver(ctx context.Context) {
	if s == nil || s.db == nil || s.storage == nil || s.index == nil || s.planner == nil {
		return
	}
	ticker := time.NewTicker(openListObserverPollInterval)
	defer ticker.Stop()
	s.pollOpenListLibraries(ctx, false)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.pollOpenListLibraries(ctx, false)
		}
	}
}

func (s *Service) pollOpenListLibraries(ctx context.Context, refresh bool) {
	libraries, sources, err := s.openListLibraries(ctx)
	if err != nil {
		log.Printf("listener: list openlist libraries: %v", err)
		return
	}
	for _, libraryRecord := range libraries {
		source := sources[libraryRecord.MediaSourceID]
		provider, err := s.storage.BuildForSource(source)
		if err != nil {
			log.Printf("listener: build openlist provider library=%d: %v", libraryRecord.ID, err)
			continue
		}
		if err := s.PollLibraryWithProvider(ctx, libraryRecord, provider, refresh); err != nil {
			log.Printf("listener: poll openlist library=%d: %v", libraryRecord.ID, err)
		}
	}
}

func (s *Service) PollLibraryWithProvider(ctx context.Context, libraryRecord database.Library, provider storage.Provider, refresh bool) error {
	if s.index == nil || s.planner == nil {
		return fmt.Errorf("storage index planner unavailable")
	}
	if provider == nil {
		return fmt.Errorf("storage provider unavailable")
	}
	previous, err := s.index.ListScoped(ctx, libraryRecord.ID, libraryRecord.RootPath)
	if err != nil {
		return err
	}
	if _, err := s.index.ObserveTree(ctx, storageindex.ObserveTreeInput{LibraryID: libraryRecord.ID, StorageProvider: provider.Name(), RootPath: libraryRecord.RootPath, Provider: provider, Refresh: refresh}); err != nil {
		return err
	}
	current, err := s.index.ListScoped(ctx, libraryRecord.ID, libraryRecord.RootPath)
	if err != nil {
		return err
	}
	result := s.planner.Plan(storageindex.PlanInput{LibraryID: libraryRecord.ID, LibraryRoot: libraryRecord.RootPath, Previous: previous, Current: current})
	for _, plan := range result.Plans {
		if err := s.enqueueRefreshPlan(ctx, plan); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) enqueueRefreshPlan(ctx context.Context, plan storageindex.RefreshPlan) error {
	if s.library == nil {
		return fmt.Errorf("library service unavailable")
	}
	if plan.FullSync {
		_, err := s.library.QueueLibraryScan(ctx, plan.LibraryID)
		return err
	}
	_, err := s.library.QueueTargetedRefresh(ctx, plan.LibraryID, plan.RootPath, defaultString(plan.Reason, "storage_index_diff"))
	return err
}

func (s *Service) openListLibraries(ctx context.Context) ([]database.Library, map[uint]database.MediaSource, error) {
	var sources []database.MediaSource
	if err := s.db.WithContext(ctx).Where("provider = ?", "openlist").Find(&sources).Error; err != nil {
		return nil, nil, err
	}
	if len(sources) == 0 {
		return nil, map[uint]database.MediaSource{}, nil
	}
	sourceByID := make(map[uint]database.MediaSource, len(sources))
	sourceIDs := make([]uint, 0, len(sources))
	for _, source := range sources {
		sourceByID[source.ID] = source
		sourceIDs = append(sourceIDs, source.ID)
	}
	var libraries []database.Library
	if err := s.db.WithContext(ctx).
		Where("media_source_id IN ? AND status = ? AND scanner_enabled = ?", sourceIDs, "active", true).
		Order("id asc").
		Find(&libraries).Error; err != nil {
		return nil, nil, err
	}
	return libraries, sourceByID, nil
}

func defaultString(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
