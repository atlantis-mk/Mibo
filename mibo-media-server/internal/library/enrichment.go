package library

import (
	"context"
	"fmt"

	"github.com/atlan/mibo-media-server/internal/database"
)

func (s *Service) QueueMediaItemMatch(ctx context.Context, mediaItemID uint, force bool) (database.Job, error) {
	if force {
		if err := s.db.WithContext(ctx).
			Model(&database.MediaItem{}).
			Where("id = ?", mediaItemID).
			Updates(map[string]any{
				"match_status":        "pending",
				"metadata_provider":   "",
				"external_id":         "",
				"metadata_confidence": nil,
			}).Error; err != nil {
			return database.Job{}, err
		}
	}

	return s.jobs.EnqueueUnique(ctx, JobKindMatchMediaItem, fmt.Sprintf("match_media_item:%d", mediaItemID), map[string]any{
		"media_item_id": mediaItemID,
	})
}

func (s *Service) QueueMediaItemMetadataRefetch(ctx context.Context, mediaItemID uint) (database.Job, error) {
	return s.jobs.EnqueueUnique(ctx, JobKindRefetchMediaItem, fmt.Sprintf("refetch_media_item:%d", mediaItemID), map[string]any{
		"media_item_id": mediaItemID,
	})
}

func (s *Service) QueueMediaFileProbe(ctx context.Context, mediaFileID uint, force bool) (database.Job, error) {
	if force {
		if err := s.db.WithContext(ctx).
			Model(&database.MediaFile{}).
			Where("id = ?", mediaFileID).
			Updates(map[string]any{
				"probe_status": "pending",
				"probe_error":  "",
			}).Error; err != nil {
			return database.Job{}, err
		}
	}

	return s.jobs.EnqueueUnique(ctx, "probe_media_file", fmt.Sprintf("probe_media_file:%d", mediaFileID), map[string]any{
		"media_file_id": mediaFileID,
	})
}
