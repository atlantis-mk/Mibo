package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/search"
	"gorm.io/gorm"
)

type Service struct {
	db     *gorm.DB
	search *search.Service
}

type UpdateInput struct {
	MediaItemID     uint  `json:"media_item_id"`
	MediaFileID     *uint `json:"media_file_id,omitempty"`
	PositionSeconds int   `json:"position_seconds"`
	DurationSeconds *int  `json:"duration_seconds,omitempty"`
	Completed       bool  `json:"completed"`
}

type State struct {
	UserID          uint       `json:"user_id"`
	MediaItemID     uint       `json:"media_item_id"`
	MediaFileID     *uint      `json:"media_file_id,omitempty"`
	PositionSeconds int        `json:"position_seconds"`
	DurationSeconds *int       `json:"duration_seconds,omitempty"`
	Watched         bool       `json:"watched"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	LastPlayedAt    *time.Time `json:"last_played_at,omitempty"`
}

type Entry struct {
	State
	MediaItem database.MediaItem `json:"media_item"`
}

func NewService(db *gorm.DB, args ...any) *Service {
	service := &Service{db: db}
	for _, arg := range args {
		if searchSvc, ok := arg.(*search.Service); ok {
			service.search = searchSvc
		}
	}
	return service
}

func (s *Service) Status() string {
	return "active"
}

func (s *Service) Update(ctx context.Context, userID uint, input UpdateInput) (State, error) {
	if input.MediaItemID == 0 {
		return State{}, fmt.Errorf("media_item_id is required")
	}
	if input.PositionSeconds < 0 {
		input.PositionSeconds = 0
	}

	var item database.MediaItem
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", input.MediaItemID).
		First(&item).Error; err != nil {
		return State{}, err
	}

	if input.MediaFileID != nil {
		var file database.MediaFile
		if err := s.db.WithContext(ctx).
			Where("id = ? AND media_item_id = ? AND deleted_at IS NULL", *input.MediaFileID, input.MediaItemID).
			First(&file).Error; err != nil {
			return State{}, fmt.Errorf("invalid media_file_id for media item")
		}
	}

	duration := input.DurationSeconds
	if duration == nil {
		duration = item.RuntimeSeconds
	}

	var progress database.PlaybackProgress
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND media_item_id = ?", userID, input.MediaItemID).
		First(&progress).Error
	created := false
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return State{}, err
		}
		progress = database.PlaybackProgress{UserID: userID, MediaItemID: input.MediaItemID}
		created = true
	}

	now := time.Now().UTC()
	progress = mergeProgress(progress, input, duration, now)

	if created {
		if err := s.db.WithContext(ctx).Create(&progress).Error; err != nil {
			return State{}, err
		}
	} else {
		if err := s.db.WithContext(ctx).Save(&progress).Error; err != nil {
			return State{}, err
		}
	}
	if s.search != nil {
		if err := s.search.ReindexMediaItem(ctx, input.MediaItemID); err != nil {
			return State{}, err
		}
	}

	return toState(progress), nil
}

func mergeProgress(progress database.PlaybackProgress, input UpdateInput, duration *int, now time.Time) database.PlaybackProgress {
	completed := input.Completed || isCompleted(input.PositionSeconds, duration)

	if input.MediaFileID != nil {
		progress.MediaFileID = input.MediaFileID
	}
	progress.DurationSeconds = chooseDuration(progress.DurationSeconds, duration)
	progress.LastPlayedAt = &now

	switch {
	case completed:
		progress.PositionSeconds = maxInt(progress.PositionSeconds, input.PositionSeconds)
		progress.Watched = true
		progress.CompletedAt = &now
	case progress.Watched:
		progress.PositionSeconds = input.PositionSeconds
		progress.Watched = false
		progress.CompletedAt = nil
	default:
		progress.PositionSeconds = maxInt(progress.PositionSeconds, input.PositionSeconds)
		progress.Watched = false
		progress.CompletedAt = nil
	}

	return progress
}

func (s *Service) GetState(ctx context.Context, userID, mediaItemID uint) (State, error) {
	var progress database.PlaybackProgress
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND media_item_id = ?", userID, mediaItemID).
		First(&progress).Error; err != nil {
		return State{}, err
	}
	return toState(progress), nil
}

func (s *Service) ContinueWatching(ctx context.Context, userID uint, limit int) ([]Entry, error) {
	return s.listEntries(ctx, userID, limit, true)
}

func (s *Service) RecentlyPlayed(ctx context.Context, userID uint, limit int) ([]Entry, error) {
	return s.listEntries(ctx, userID, limit, false)
}

func (s *Service) listEntries(ctx context.Context, userID uint, limit int, onlyContinue bool) ([]Entry, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	query := s.db.WithContext(ctx).
		Model(&database.PlaybackProgress{}).
		Where("user_id = ? AND last_played_at IS NOT NULL", userID)
	if onlyContinue {
		query = query.Where("watched = ? AND position_seconds > 0", false)
	}

	var progresses []database.PlaybackProgress
	if err := query.Order("last_played_at desc").Limit(limit).Find(&progresses).Error; err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(progresses))
	for _, progress := range progresses {
		var item database.MediaItem
		if err := s.db.WithContext(ctx).
			Where("id = ? AND deleted_at IS NULL", progress.MediaItemID).
			First(&item).Error; err != nil {
			return nil, err
		}
		entries = append(entries, Entry{State: toState(progress), MediaItem: item})
	}
	return entries, nil
}

func toState(progress database.PlaybackProgress) State {
	return State{
		UserID:          progress.UserID,
		MediaItemID:     progress.MediaItemID,
		MediaFileID:     progress.MediaFileID,
		PositionSeconds: progress.PositionSeconds,
		DurationSeconds: progress.DurationSeconds,
		Watched:         progress.Watched,
		CompletedAt:     progress.CompletedAt,
		LastPlayedAt:    progress.LastPlayedAt,
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

func maxInt(left, right int) int {
	if right > left {
		return right
	}
	return left
}
