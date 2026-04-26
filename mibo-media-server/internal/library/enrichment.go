package library

import (
	"context"
	"fmt"

	"github.com/atlan/mibo-media-server/internal/catalog"
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

func (s *Service) QueueCatalogItemMatch(ctx context.Context, itemID uint) (database.Job, error) {
	if s.jobs == nil {
		return database.Job{}, fmt.Errorf("jobs service unavailable")
	}
	if itemID == 0 {
		return database.Job{}, fmt.Errorf("catalog item id is required")
	}

	targetID, shouldQueue, err := s.catalogMatchTargetForQueue(ctx, itemID)
	if err != nil {
		return database.Job{}, err
	}
	if !shouldQueue {
		return database.Job{}, nil
	}

	return s.jobs.EnqueueUnique(ctx, JobKindMatchCatalogItem, fmt.Sprintf("match_catalog_item:%d", targetID), map[string]any{
		"item_id": targetID,
	})
}

func (s *Service) catalogMatchTargetForQueue(ctx context.Context, itemID uint) (uint, bool, error) {
	var item database.CatalogItem
	if err := s.db.WithContext(ctx).First(&item, itemID).Error; err != nil {
		return 0, false, fmt.Errorf("load catalog item %d: %w", itemID, err)
	}
	if item.DeletedAt != nil {
		return 0, false, fmt.Errorf("catalog item %d is deleted", item.ID)
	}

	targetID := item.ID
	if item.Type == catalog.ItemTypeSeason || item.Type == catalog.ItemTypeEpisode {
		if item.RootID == nil || *item.RootID == 0 {
			return 0, false, fmt.Errorf("catalog item %d missing root_id", item.ID)
		}
		targetID = *item.RootID
	}

	targetItem := item
	if targetID != item.ID {
		targetItem = database.CatalogItem{}
		if err := s.db.WithContext(ctx).First(&targetItem, targetID).Error; err != nil {
			return 0, false, fmt.Errorf("load catalog match target %d for item %d: %w", targetID, itemID, err)
		}
		if targetItem.DeletedAt != nil {
			return 0, false, fmt.Errorf("catalog item %d is deleted", targetItem.ID)
		}
	}

	return targetID, targetItem.GovernanceStatus == catalog.GovernancePending, nil
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

func (s *Service) QueueInventoryFileProbe(ctx context.Context, inventoryFileID uint, force bool) (database.Job, error) {
	if force {
		var assetIDs []uint
		if err := s.db.WithContext(ctx).
			Model(&database.AssetFile{}).
			Distinct("asset_id").
			Where("file_id = ?", inventoryFileID).
			Pluck("asset_id", &assetIDs).Error; err != nil {
			return database.Job{}, err
		}
		if len(assetIDs) > 0 {
			if err := s.db.WithContext(ctx).
				Model(&database.MediaAsset{}).
				Where("id IN ?", assetIDs).
				Updates(map[string]any{
					"probe_status":           "pending",
					"technical_summary_json": "",
				}).Error; err != nil {
				return database.Job{}, err
			}
		}
	}

	return s.jobs.EnqueueUnique(ctx, JobKindProbeInventoryFile, fmt.Sprintf("probe_inventory_file:%d", inventoryFileID), map[string]any{
		"inventory_file_id": inventoryFileID,
	})
}
