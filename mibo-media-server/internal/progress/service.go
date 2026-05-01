package progress

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

type UpdateInput struct {
	ItemID          uint  `json:"item_id,omitempty"`
	AssetID         *uint `json:"asset_id,omitempty"`
	PositionSeconds int   `json:"position_seconds"`
	DurationSeconds *int  `json:"duration_seconds,omitempty"`
	Completed       bool  `json:"completed"`
}

type State struct {
	UserID           uint       `json:"user_id"`
	ItemID           uint       `json:"item_id,omitempty"`
	AssetID          *uint      `json:"asset_id,omitempty"`
	PositionSeconds  int        `json:"position_seconds"`
	DurationSeconds  *int       `json:"duration_seconds,omitempty"`
	PlayedPercentage *float64   `json:"played_percentage,omitempty"`
	PlayCount        int        `json:"play_count,omitempty"`
	Watched          bool       `json:"watched"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	LastPlayedAt     *time.Time `json:"last_played_at,omitempty"`
}

func NewService(db *gorm.DB, args ...any) *Service {
	_ = args
	return &Service{db: db}
}

func (s *Service) Status() string {
	return "active"
}

func (s *Service) Update(ctx context.Context, userID uint, input UpdateInput) (State, error) {
	return s.updateCatalog(ctx, userID, input)
}

func (s *Service) updateCatalog(ctx context.Context, userID uint, input UpdateInput) (State, error) {
	if input.ItemID == 0 {
		return State{}, fmt.Errorf("item_id is required")
	}
	if input.PositionSeconds < 0 {
		input.PositionSeconds = 0
	}

	var item database.CatalogItem
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", input.ItemID).
		First(&item).Error; err != nil {
		return State{}, err
	}
	if input.AssetID != nil {
		var assetLink database.AssetItem
		if err := s.db.WithContext(ctx).
			Where("item_id = ? AND asset_id = ?", input.ItemID, *input.AssetID).
			First(&assetLink).Error; err != nil {
			return State{}, fmt.Errorf("invalid asset_id for catalog item")
		}
	}

	duration := input.DurationSeconds
	if duration == nil {
		duration = item.RuntimeSeconds
	}
	policy, err := s.playbackPolicy(ctx, item.LibraryID)
	if err != nil {
		return State{}, err
	}
	if !input.Completed && !policy.shouldRecordResume(duration) {
		return State{UserID: userID, ItemID: input.ItemID, AssetID: input.AssetID, DurationSeconds: duration, Watched: false}, nil
	}

	var data database.UserItemData
	lookup := s.db.WithContext(ctx).Where("user_id = ? AND item_id = ?", userID, input.ItemID)
	if input.AssetID == nil {
		lookup = lookup.Where("asset_id IS NULL")
	} else {
		lookup = lookup.Where("asset_id = ?", *input.AssetID)
	}
	err = lookup.First(&data).Error
	created := false
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return State{}, err
		}
		data = database.UserItemData{UserID: userID, ItemID: input.ItemID, AssetID: input.AssetID}
		created = true
	}

	now := time.Now().UTC()
	data = mergeCatalogProgress(data, input, duration, now, policy)
	if created {
		if err := s.db.WithContext(ctx).Create(&data).Error; err != nil {
			return State{}, err
		}
	} else {
		if err := s.db.WithContext(ctx).Save(&data).Error; err != nil {
			return State{}, err
		}
	}

	return toCatalogState(data, duration), nil
}

func mergeCatalogProgress(data database.UserItemData, input UpdateInput, duration *int, now time.Time, policy playbackPolicy) database.UserItemData {
	completed := input.Completed || policy.isCompleted(input.PositionSeconds, duration)
	data.LastPlayedAt = &now
	data.PlayedPercentage = playedPercentage(input.PositionSeconds, duration)
	if input.AssetID != nil {
		data.AssetID = input.AssetID
	}

	switch {
	case completed:
		data.PositionSeconds = maxInt(data.PositionSeconds, input.PositionSeconds)
		data.PlayCount = maxInt(data.PlayCount, 1)
		data.CompletedAt = &now
	case data.CompletedAt != nil:
		data.PositionSeconds = input.PositionSeconds
		data.PlayCount = maxInt(data.PlayCount, 1)
		data.CompletedAt = nil
	default:
		data.PositionSeconds = maxInt(data.PositionSeconds, input.PositionSeconds)
		data.PlayCount = maxInt(data.PlayCount, 1)
		data.CompletedAt = nil
	}

	return data
}

type playbackPolicy struct {
	ResumeEnabled            bool
	MinResumePct             int
	MaxResumePct             int
	MinResumeDurationSeconds int
}

func (s *Service) playbackPolicy(ctx context.Context, libraryID uint) (playbackPolicy, error) {
	if err := database.EnsureLibraryPolicyDefaults(s.db.WithContext(ctx), libraryID); err != nil {
		return playbackPolicy{}, err
	}
	var policy database.LibraryPlaybackPolicy
	if err := s.db.WithContext(ctx).Where("library_id = ?", libraryID).First(&policy).Error; err != nil {
		return playbackPolicy{}, err
	}
	return playbackPolicy{ResumeEnabled: policy.ResumeEnabled, MinResumePct: policy.MinResumePct, MaxResumePct: policy.MaxResumePct, MinResumeDurationSeconds: policy.MinResumeDurationSeconds}, nil
}

func (p playbackPolicy) shouldRecordResume(duration *int) bool {
	if !p.ResumeEnabled {
		return false
	}
	if duration == nil || *duration <= 0 {
		return true
	}
	return *duration >= p.MinResumeDurationSeconds
}

func (p playbackPolicy) isCompleted(positionSeconds int, durationSeconds *int) bool {
	if durationSeconds == nil || *durationSeconds <= 0 {
		return false
	}
	threshold := *durationSeconds * p.MaxResumePct / 100
	if threshold <= 0 {
		threshold = *durationSeconds * 90 / 100
	}
	return positionSeconds >= threshold
}

func (s *Service) GetCatalogState(ctx context.Context, userID, itemID uint) (State, error) {
	var item database.CatalogItem
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", itemID).
		First(&item).Error; err != nil {
		return State{}, err
	}

	var data database.UserItemData
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND item_id = ?", userID, itemID).
		Order("last_played_at desc, id desc").
		First(&data).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return State{UserID: userID, ItemID: itemID, DurationSeconds: item.RuntimeSeconds, Watched: false}, nil
		}
		return State{}, err
	}
	return toCatalogState(data, item.RuntimeSeconds), nil
}

func toCatalogState(data database.UserItemData, duration *int) State {
	return State{
		UserID:           data.UserID,
		ItemID:           data.ItemID,
		AssetID:          data.AssetID,
		PositionSeconds:  data.PositionSeconds,
		DurationSeconds:  duration,
		PlayedPercentage: data.PlayedPercentage,
		PlayCount:        data.PlayCount,
		Watched:          data.CompletedAt != nil,
		CompletedAt:      data.CompletedAt,
		LastPlayedAt:     data.LastPlayedAt,
	}
}

func isCompleted(positionSeconds int, durationSeconds *int) bool {
	if durationSeconds == nil || *durationSeconds <= 0 {
		return false
	}
	threshold := *durationSeconds - 30
	if threshold < *durationSeconds*95/100 {
		threshold = *durationSeconds * 95 / 100
	}
	return positionSeconds >= threshold
}

func chooseDuration(existing, incoming *int) *int {
	if incoming != nil && *incoming > 0 {
		return incoming
	}
	if existing != nil && *existing > 0 {
		return existing
	}
	return nil
}

func playedPercentage(positionSeconds int, durationSeconds *int) *float64 {
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

func maxInt(left, right int) int {
	if right > left {
		return right
	}
	return left
}
