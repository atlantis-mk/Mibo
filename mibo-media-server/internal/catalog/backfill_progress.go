package catalog

import (
	"context"
	"errors"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Service) backfillProgress(ctx context.Context, run database.CatalogMigrationRun) error {
	progressRows, err := s.listLegacyPlaybackProgress(ctx, run)
	if err != nil {
		return err
	}

	for _, progressRow := range progressRows {
		legacyItem, err := s.loadLegacyProgressMediaItem(ctx, progressRow.MediaItemID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}

		catalogItem, assetID, legacyFileID, err := s.resolveLegacyProgressTarget(ctx, run.ID, legacyItem, progressRow.MediaFileID)
		if err != nil {
			return err
		}
		if assetID == nil {
			if _, err := s.recordLegacyBackfillEntry(ctx, run.ID, LegacyBackfillEntry{
				EntryType:         LegacyBackfillEntryTypeSkipped,
				LibraryID:         uintPtr(legacyItem.LibraryID),
				LegacyMediaItemID: uintPtr(legacyItem.ID),
				LegacyMediaFileID: legacyFileID,
				CatalogItemID:     uintPtr(catalogItem.ID),
				StoragePath:       strings.TrimSpace(legacyItem.SourcePath),
				Title:             strings.TrimSpace(legacyItem.Title),
				Message:           "legacy playback progress has no resolved catalog asset mapping",
				Details:           mustJSON(map[string]any{"user_id": progressRow.UserID}),
			}); err != nil {
				return err
			}
			continue
		}

		userItemData := database.UserItemData{
			UserID:           progressRow.UserID,
			ItemID:           catalogItem.ID,
			AssetID:          assetID,
			PositionSeconds:  progressRow.PositionSeconds,
			PlayedPercentage: legacyPlayedPercentage(progressRow.PositionSeconds, progressDurationSeconds(progressRow, legacyItem)),
			PlayCount:        legacyProgressPlayCount(progressRow),
			LastPlayedAt:     progressRow.LastPlayedAt,
			CompletedAt:      progressRow.CompletedAt,
		}
		if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}, {Name: "asset_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"position_seconds", "played_percentage", "play_count", "last_played_at", "completed_at", "updated_at"}),
		}).Create(&userItemData).Error; err != nil {
			return err
		}

		if _, err := s.recordLegacyBackfillEntry(ctx, run.ID, LegacyBackfillEntry{
			EntryType:         LegacyBackfillEntryTypeSuccess,
			LibraryID:         uintPtr(legacyItem.LibraryID),
			LegacyMediaItemID: uintPtr(legacyItem.ID),
			LegacyMediaFileID: legacyFileID,
			CatalogItemID:     uintPtr(catalogItem.ID),
			AssetID:           assetID,
			StoragePath:       strings.TrimSpace(legacyItem.SourcePath),
			Title:             strings.TrimSpace(legacyItem.Title),
			Message:           "migrated legacy playback progress into catalog user data",
			Details: mustJSON(map[string]any{
				"user_id":              progressRow.UserID,
				"position_seconds":     progressRow.PositionSeconds,
				"played_percentage":    userItemData.PlayedPercentage,
				"play_count":           userItemData.PlayCount,
				"legacy_media_file_id": legacyFileID,
			}),
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) listLegacyPlaybackProgress(ctx context.Context, run database.CatalogMigrationRun) ([]database.PlaybackProgress, error) {
	query := s.db.WithContext(ctx).
		Model(&database.PlaybackProgress{}).
		Order("user_id asc").
		Order("media_item_id asc")

	if run.ScopeKind == LegacyBackfillScopeLibrary && run.LibraryID != nil {
		var mediaItemIDs []uint
		if err := s.db.WithContext(ctx).
			Model(&database.MediaItem{}).
			Where("library_id = ? AND deleted_at IS NULL", *run.LibraryID).
			Pluck("id", &mediaItemIDs).Error; err != nil {
			return nil, err
		}
		if len(mediaItemIDs) == 0 {
			return nil, nil
		}
		query = query.Where("media_item_id IN ?", mediaItemIDs)
	}

	var rows []database.PlaybackProgress
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) loadLegacyProgressMediaItem(ctx context.Context, mediaItemID uint) (database.MediaItem, error) {
	var item database.MediaItem
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", mediaItemID).
		First(&item).Error
	return item, err
}

func (s *Service) resolveLegacyProgressTarget(ctx context.Context, runID uint, legacyItem database.MediaItem, mediaFileID *uint) (database.CatalogItem, *uint, *uint, error) {
	catalogItem, err := s.resolveLegacyProgressCatalogItem(ctx, runID, legacyItem)
	if err != nil {
		return database.CatalogItem{}, nil, nil, err
	}

	if mediaFileID == nil || *mediaFileID == 0 {
		var assetLinks []database.AssetItem
		if err := s.db.WithContext(ctx).
			Where("item_id = ?", catalogItem.ID).
			Order("id asc").
			Limit(2).
			Find(&assetLinks).Error; err != nil {
			return database.CatalogItem{}, nil, nil, err
		}
		if len(assetLinks) == 1 {
			return catalogItem, uintPtr(assetLinks[0].AssetID), nil, nil
		}
		return catalogItem, nil, nil, nil
	}

	var legacyFile database.MediaFile
	if err := s.db.WithContext(ctx).
		Where("id = ? AND media_item_id = ? AND deleted_at IS NULL", *mediaFileID, legacyItem.ID).
		First(&legacyFile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return catalogItem, nil, nil, nil
		}
		return database.CatalogItem{}, nil, nil, err
	}

	var inventoryFile database.InventoryFile
	if err := s.db.WithContext(ctx).
		Where("storage_provider = ? AND storage_path = ? AND deleted_at IS NULL", legacyMovieStorageProvider(legacyFile), strings.TrimSpace(legacyFile.StoragePath)).
		First(&inventoryFile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return catalogItem, nil, uintPtr(legacyFile.ID), nil
		}
		return database.CatalogItem{}, nil, nil, err
	}

	var asset database.MediaAsset
	if err := s.db.WithContext(ctx).
		Model(&database.MediaAsset{}).
		Joins("JOIN asset_items ON asset_items.asset_id = media_assets.id").
		Joins("JOIN asset_files ON asset_files.asset_id = media_assets.id").
		Where("asset_items.item_id = ?", catalogItem.ID).
		Where("asset_files.file_id = ?", inventoryFile.ID).
		Where("media_assets.deleted_at IS NULL").
		Order("media_assets.id asc").
		First(&asset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return catalogItem, nil, uintPtr(legacyFile.ID), nil
		}
		return database.CatalogItem{}, nil, nil, err
	}

	return catalogItem, uintPtr(asset.ID), uintPtr(legacyFile.ID), nil
}

func (s *Service) resolveLegacyProgressCatalogItem(ctx context.Context, runID uint, legacyItem database.MediaItem) (database.CatalogItem, error) {
	var catalogItem database.CatalogItem
	err := s.db.WithContext(ctx).
		Where("library_id = ? AND type = ? AND path = ? AND deleted_at IS NULL", legacyItem.LibraryID, strings.TrimSpace(legacyItem.Type), strings.TrimSpace(legacyItem.SourcePath)).
		Order("id asc").
		First(&catalogItem).Error
	if err == nil {
		return catalogItem, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) || strings.TrimSpace(legacyItem.Type) != ItemTypeEpisode || runID == 0 {
		return database.CatalogItem{}, err
	}

	var entry database.CatalogMigrationEntry
	err = s.db.WithContext(ctx).
		Where("run_id = ? AND legacy_media_item_id = ? AND catalog_item_id IS NOT NULL", runID, legacyItem.ID).
		Where("entry_type IN ?", []string{LegacyBackfillEntryTypeDuplicateEpisodeCandidate, LegacyBackfillEntryTypeSuccess}).
		Order("id asc").
		First(&entry).Error
	if err != nil {
		return database.CatalogItem{}, err
	}
	err = s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", *entry.CatalogItemID).
		First(&catalogItem).Error
	return catalogItem, err
}

func progressDurationSeconds(progress database.PlaybackProgress, legacyItem database.MediaItem) *int {
	if progress.DurationSeconds != nil && *progress.DurationSeconds > 0 {
		return progress.DurationSeconds
	}
	if legacyItem.RuntimeSeconds != nil && *legacyItem.RuntimeSeconds > 0 {
		return legacyItem.RuntimeSeconds
	}
	return nil
}

func legacyPlayedPercentage(positionSeconds int, durationSeconds *int) *float64 {
	if durationSeconds == nil || *durationSeconds <= 0 {
		return nil
	}
	percentage := float64(positionSeconds) / float64(*durationSeconds) * 100
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}
	return &percentage
}

func legacyProgressPlayCount(progress database.PlaybackProgress) int {
	if progress.LastPlayedAt != nil || progress.CompletedAt != nil || progress.PositionSeconds > 0 {
		return 1
	}
	return 0
}
