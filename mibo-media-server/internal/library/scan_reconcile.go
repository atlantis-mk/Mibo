package library

import (
	"context"
	"errors"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func ReconcileProvisionalMediaFile(ctx context.Context, db *gorm.DB, mediaFileID uint) error {
	if db == nil {
		return nil
	}
	var file database.MediaFile
	if err := db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", mediaFileID).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if file.DurationSeconds == nil || file.IdentityStatus != mediaFileIdentityStatusProvisional {
		return nil
	}
	var candidates []database.MediaFile
	if err := db.WithContext(ctx).Where("library_id = ? AND id <> ? AND deleted_at IS NOT NULL AND media_item_id IS NOT NULL AND replaced_by_id IS NULL AND size_bytes = ? AND duration_seconds IS NOT NULL", file.LibraryID, file.ID, file.SizeBytes).Order("deleted_at desc, id desc").Find(&candidates).Error; err != nil {
		return err
	}
	matches := make([]database.MediaFile, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.DurationSeconds == nil {
			continue
		}
		if durationDelta(*candidate.DurationSeconds, *file.DurationSeconds) <= fallbackDurationToleranceSeconds {
			matches = append(matches, candidate)
		}
	}
	if len(matches) == 0 {
		return db.WithContext(ctx).Model(&database.MediaFile{}).Where("id = ?", file.ID).Updates(map[string]any{"review_status": mediaFileReviewStatusPending, "review_reason": "no_high_confidence_match"}).Error
	}
	if len(matches) > 1 {
		return db.WithContext(ctx).Model(&database.MediaFile{}).Where("id = ?", file.ID).Updates(map[string]any{"review_status": mediaFileReviewStatusNeeded, "review_reason": "ambiguous_size_duration_match"}).Error
	}
	target := matches[0]
	targetMediaItemID := *target.MediaItemID
	currentMediaItemID := file.MediaItemID
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&database.MediaFile{}).Where("id = ?", file.ID).Updates(map[string]any{"media_item_id": targetMediaItemID, "identity_status": mediaFileIdentityStatusReconciled, "review_status": mediaFileReviewStatusNone, "review_reason": ""}).Error; err != nil {
			return err
		}
		if err := tx.Model(&database.MediaFile{}).Where("id = ?", target.ID).Update("replaced_by_id", file.ID).Error; err != nil {
			return err
		}
		if err := tx.Model(&database.PlaybackProgress{}).Where("media_item_id = ? AND media_file_id = ?", targetMediaItemID, target.ID).Update("media_file_id", file.ID).Error; err != nil {
			return err
		}
		if err := tx.Model(&database.MediaItem{}).Where("id = ?", targetMediaItemID).Updates(map[string]any{"source_path": file.StoragePath, "status": "ready", "deleted_at": nil}).Error; err != nil {
			return err
		}
		if currentMediaItemID != nil && *currentMediaItemID != targetMediaItemID {
			var activeCount int64
			if err := tx.Model(&database.MediaFile{}).Where("media_item_id = ? AND deleted_at IS NULL AND id <> ?", *currentMediaItemID, file.ID).Count(&activeCount).Error; err != nil {
				return err
			}
			if activeCount == 0 {
				if err := tx.Model(&database.MediaItem{}).Where("id = ?", *currentMediaItemID).Updates(map[string]any{"status": "missing", "deleted_at": time.Now().UTC()}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
