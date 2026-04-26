package metadata

import (
	"context"
	"fmt"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/schedule"
)

type ScheduledJobResult struct {
	ItemsProcessed int    `json:"items_processed"`
	UpdatedItems   int    `json:"updated_items"`
	Summary        string `json:"summary"`
}

func (s *Service) RunScheduledMetadataRefetch(ctx context.Context, due schedule.DueSchedule) (ScheduledJobResult, error) {
	return s.runScheduledItems(ctx, due, "metadata refetch", func(ctx context.Context, item database.MediaItem) error {
		if strings.TrimSpace(item.ExternalID) != "" {
			return s.RefetchItem(ctx, item.ID)
		}
		return s.MatchItem(ctx, item.ID)
	})
}

func (s *Service) RunScheduledTrailerSync(ctx context.Context, due schedule.DueSchedule) (ScheduledJobResult, error) {
	return s.runScheduledItems(ctx, due, "trailer sync", func(ctx context.Context, item database.MediaItem) error {
		if strings.TrimSpace(item.ExternalID) == "" {
			return s.MatchItem(ctx, item.ID)
		}
		return s.RefetchItem(ctx, item.ID)
	})
}

func (s *Service) RunScheduledArtworkRefresh(ctx context.Context, due schedule.DueSchedule) (ScheduledJobResult, error) {
	items, err := s.listScheduledItems(ctx, due)
	if err != nil {
		return ScheduledJobResult{}, err
	}
	tmdbCfg, err := s.tmdbConfig(ctx)
	if err != nil {
		return ScheduledJobResult{}, err
	}
	updated := 0
	for _, item := range items {
		if strings.TrimSpace(item.ExternalID) == "" {
			continue
		}
		mediaType, externalID, err := parseExternalID(item.ExternalID)
		if err != nil {
			return ScheduledJobResult{}, err
		}
		detail, err := s.fetchDetail(ctx, tmdbCfg, mediaType, externalID)
		if err != nil {
			return ScheduledJobResult{}, err
		}
		updates := map[string]any{
			"poster_url":   preferArtworkURL(item.PosterURL, imageURL(tmdbCfg, detail.PosterPath)),
			"backdrop_url": preferArtworkURL(item.BackdropURL, imageURL(tmdbCfg, detail.BackdropPath)),
			"logo_url":     preferArtworkURL(item.LogoURL, imageURL(tmdbCfg, pickLogoPath(tmdbCfg.Language, detail.Images.Logos))),
		}
		if err := s.db.WithContext(ctx).Model(&database.MediaItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
			return ScheduledJobResult{}, err
		}
		updated++
	}
	return ScheduledJobResult{ItemsProcessed: len(items), UpdatedItems: updated, Summary: fmt.Sprintf("artwork refreshed for %d items", updated)}, nil
}

func (s *Service) runScheduledItems(ctx context.Context, due schedule.DueSchedule, label string, fn func(context.Context, database.MediaItem) error) (ScheduledJobResult, error) {
	items, err := s.listScheduledItems(ctx, due)
	if err != nil {
		return ScheduledJobResult{}, err
	}
	updated := 0
	for _, item := range items {
		if err := fn(ctx, item); err != nil {
			return ScheduledJobResult{}, err
		}
		updated++
	}
	return ScheduledJobResult{ItemsProcessed: len(items), UpdatedItems: updated, Summary: fmt.Sprintf("%s completed for %d items", label, updated)}, nil
}

func (s *Service) listScheduledItems(ctx context.Context, due schedule.DueSchedule) ([]database.MediaItem, error) {
	query := s.db.WithContext(ctx).Model(&database.MediaItem{}).Where("deleted_at IS NULL")
	switch due.ScopeKind {
	case schedule.ScopeGlobal:
		// no extra filter
	case schedule.ScopeLibrary:
		if due.LibraryID == nil || *due.LibraryID == 0 {
			return nil, fmt.Errorf("library scope requires library_id")
		}
		query = query.Where("library_id = ?", *due.LibraryID)
	default:
		return nil, fmt.Errorf("unsupported schedule scope %q", due.ScopeKind)
	}
	var items []database.MediaItem
	if err := query.Order("id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
