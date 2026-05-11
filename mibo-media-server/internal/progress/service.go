package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

type UpdateInput struct {
	MetadataItemID    uint   `json:"metadata_item_id,omitempty"`
	ResourceID        uint   `json:"resource_id,omitempty"`
	PositionSeconds   int    `json:"position_seconds"`
	DurationSeconds   *int   `json:"duration_seconds,omitempty"`
	Completed         bool   `json:"completed"`
	ProgressFrameURL  string `json:"progress_frame_url,omitempty"`
	ProgressFrameData string `json:"progress_frame_data,omitempty"`
}

type State struct {
	UserID              uint       `json:"user_id"`
	MetadataItemID      uint       `json:"metadata_item_id,omitempty"`
	ResourceID          uint       `json:"resource_id,omitempty"`
	PreferredResourceID *uint      `json:"preferred_resource_id,omitempty"`
	PositionSeconds     int        `json:"position_seconds"`
	DurationSeconds     *int       `json:"duration_seconds,omitempty"`
	PlayedPercentage    *float64   `json:"played_percentage,omitempty"`
	ProgressFrameURL    string     `json:"progress_frame_url,omitempty"`
	PlayCount           int        `json:"play_count,omitempty"`
	Watched             bool       `json:"watched"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	LastPlayedAt        *time.Time `json:"last_played_at,omitempty"`
}

func NewService(db *gorm.DB, args ...any) *Service {
	_ = args
	return &Service{db: db}
}

func (s *Service) Status() string {
	return "active"
}

func (s *Service) Update(ctx context.Context, userID uint, input UpdateInput) (State, error) {
	if input.MetadataItemID != 0 && input.ResourceID != 0 {
		if state, ok, err := s.updateResource(ctx, userID, input); err != nil {
			return State{}, err
		} else if ok {
			return state, nil
		}
	}
	return State{}, fmt.Errorf("metadata_item_id and resource_id are required for progress updates")
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

func (s *Service) GetMetadataItemState(ctx context.Context, userID, metadataItemID uint) (State, error) {
	if state, ok, err := s.GetMetadataState(ctx, userID, metadataItemID, 0); err != nil {
		return State{}, err
	} else if ok {
		return state, nil
	}
	return State{}, fmt.Errorf("metadata item %d not found", metadataItemID)
}

func (s *Service) SetPreferredResource(ctx context.Context, userID, metadataItemID, resourceID uint) (State, error) {
	if metadataItemID == 0 || resourceID == 0 {
		return State{}, fmt.Errorf("metadata_item_id and resource_id are required")
	}
	var metadataItem database.MetadataItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", metadataItemID).First(&metadataItem).Error; err != nil {
		return State{}, err
	}
	var link database.ResourceMetadataLink
	if err := s.db.WithContext(ctx).Where("resource_id = ? AND metadata_item_id = ?", resourceID, metadataItemID).First(&link).Error; err != nil {
		return State{}, err
	}
	var metadataData database.UserMetadataData
	err := s.db.WithContext(ctx).Where("user_id = ? AND metadata_item_id = ?", userID, metadataItemID).First(&metadataData).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return State{}, err
	}
	created := err == gorm.ErrRecordNotFound
	metadataData.UserID = userID
	metadataData.MetadataItemID = metadataItemID
	metadataData.PreferredResourceID = &resourceID
	if created {
		if err := s.db.WithContext(ctx).Create(&metadataData).Error; err != nil {
			return State{}, err
		}
	} else if err := s.db.WithContext(ctx).Save(&metadataData).Error; err != nil {
		return State{}, err
	}
	state, err := s.GetMetadataItemState(ctx, userID, metadataItemID)
	if err != nil {
		return State{}, err
	}
	return state, nil
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
