package progress

import (
	"context"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func (s *Service) updateResource(ctx context.Context, userID uint, input UpdateInput) (State, bool, error) {
	var metadataItem database.MetadataItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", input.MetadataItemID).First(&metadataItem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return State{}, false, nil
		}
		return State{}, true, err
	}
	var link database.ResourceMetadataLink
	if err := s.db.WithContext(ctx).Where("resource_id = ? AND metadata_item_id = ?", input.ResourceID, input.MetadataItemID).First(&link).Error; err != nil {
		return State{}, true, err
	}
	duration := input.DurationSeconds
	if duration == nil {
		duration = metadataItem.RuntimeSeconds
	}
	policy := playbackPolicy{ResumeEnabled: true, MinResumeDurationSeconds: 0, MaxResumePct: 90}
	now := time.Now().UTC()
	var resourceData database.UserResourceData
	err := s.db.WithContext(ctx).Where("user_id = ? AND resource_id = ? AND metadata_item_id = ?", userID, input.ResourceID, input.MetadataItemID).First(&resourceData).Error
	created := false
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return State{}, true, err
		}
		resourceData = database.UserResourceData{UserID: userID, ResourceID: input.ResourceID, MetadataItemID: input.MetadataItemID}
		created = true
	}
	resourceData = mergeResourceProgress(resourceData, input, duration, now, policy)
	if created {
		if err := s.db.WithContext(ctx).Create(&resourceData).Error; err != nil {
			return State{}, true, err
		}
	} else if err := s.db.WithContext(ctx).Save(&resourceData).Error; err != nil {
		return State{}, true, err
	}
	if err := s.upsertMetadataProgress(ctx, userID, input.MetadataItemID, input.ResourceID, resourceData, duration); err != nil {
		return State{}, true, err
	}
	return toResourceState(resourceData, duration), true, nil
}

func (s *Service) GetMetadataState(ctx context.Context, userID uint, metadataItemID uint, resourceID uint) (State, bool, error) {
	var metadataItem database.MetadataItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", metadataItemID).First(&metadataItem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return State{}, false, nil
		}
		return State{}, true, err
	}
	if resourceID != 0 {
		var resourceData database.UserResourceData
		if err := s.db.WithContext(ctx).Where("user_id = ? AND resource_id = ? AND metadata_item_id = ?", userID, resourceID, metadataItemID).First(&resourceData).Error; err == nil {
			state := toResourceState(resourceData, metadataItem.RuntimeSeconds)
			state.PreferredResourceID = &resourceID
			return state, true, nil
		} else if err != gorm.ErrRecordNotFound {
			return State{}, true, err
		}
	}
	var metadataData database.UserMetadataData
	if err := s.db.WithContext(ctx).Where("user_id = ? AND metadata_item_id = ?", userID, metadataItemID).First(&metadataData).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return State{UserID: userID, MetadataItemID: metadataItemID, ResourceID: resourceID, DurationSeconds: metadataItem.RuntimeSeconds, Watched: false}, true, nil
		}
		return State{}, true, err
	}
	return State{UserID: userID, MetadataItemID: metadataItemID, ResourceID: resourceID, PreferredResourceID: metadataData.PreferredResourceID, PositionSeconds: metadataData.PositionSeconds, DurationSeconds: metadataItem.RuntimeSeconds, PlayedPercentage: metadataData.PlayedPercentage, ProgressFrameURL: metadataData.ProgressFrameURL, PlayCount: metadataData.PlayCount, Watched: metadataData.CompletedAt != nil, CompletedAt: metadataData.CompletedAt, LastPlayedAt: metadataData.LastPlayedAt}, true, nil
}

func mergeResourceProgress(data database.UserResourceData, input UpdateInput, duration *int, now time.Time, policy playbackPolicy) database.UserResourceData {
	completed := input.Completed || policy.isCompleted(input.PositionSeconds, duration)
	data.LastPlayedAt = &now
	data.PlayedPercentage = playedPercentage(input.PositionSeconds, duration)
	if input.ProgressFrameURL != "" {
		data.ProgressFrameURL = input.ProgressFrameURL
	}
	if completed {
		data.PositionSeconds = maxInt(data.PositionSeconds, input.PositionSeconds)
		data.PlayCount = maxInt(data.PlayCount, 1)
		data.CompletedAt = &now
		return data
	}
	data.PositionSeconds = maxInt(data.PositionSeconds, input.PositionSeconds)
	data.PlayCount = maxInt(data.PlayCount, 1)
	data.CompletedAt = nil
	return data
}

func (s *Service) upsertMetadataProgress(ctx context.Context, userID uint, metadataItemID uint, resourceID uint, resourceData database.UserResourceData, duration *int) error {
	var metadataData database.UserMetadataData
	err := s.db.WithContext(ctx).Where("user_id = ? AND metadata_item_id = ?", userID, metadataItemID).First(&metadataData).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	created := err == gorm.ErrRecordNotFound
	metadataData.UserID = userID
	metadataData.MetadataItemID = metadataItemID
	metadataData.PreferredResourceID = &resourceID
	metadataData.PositionSeconds = resourceData.PositionSeconds
	metadataData.PlayedPercentage = resourceData.PlayedPercentage
	metadataData.ProgressFrameURL = resourceData.ProgressFrameURL
	metadataData.PlayCount = maxInt(metadataData.PlayCount, resourceData.PlayCount)
	metadataData.LastPlayedAt = resourceData.LastPlayedAt
	metadataData.CompletedAt = resourceData.CompletedAt
	_ = duration
	if created {
		return s.db.WithContext(ctx).Create(&metadataData).Error
	}
	return s.db.WithContext(ctx).Save(&metadataData).Error
}

func toResourceState(data database.UserResourceData, duration *int) State {
	resourceID := data.ResourceID
	return State{UserID: data.UserID, MetadataItemID: data.MetadataItemID, ResourceID: data.ResourceID, PreferredResourceID: &resourceID, PositionSeconds: data.PositionSeconds, DurationSeconds: duration, PlayedPercentage: data.PlayedPercentage, ProgressFrameURL: data.ProgressFrameURL, PlayCount: data.PlayCount, Watched: data.CompletedAt != nil, CompletedAt: data.CompletedAt, LastPlayedAt: data.LastPlayedAt}
}
