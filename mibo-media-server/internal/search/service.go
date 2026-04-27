package search

import (
	"context"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

type HistoryEntry struct {
	ID           uint      `json:"id"`
	Query        string    `json:"query"`
	TypeFilter   string    `json:"type_filter"`
	Genre        string    `json:"genre"`
	Region       string    `json:"region"`
	Year         *int      `json:"year,omitempty"`
	MinRating    *float64  `json:"min_rating,omitempty"`
	WatchedState string    `json:"watched_state"`
	Sort         string    `json:"sort"`
	LastUsedAt   time.Time `json:"last_used_at"`
}

func NewService(args ...any) *Service {
	service := &Service{}
	for _, arg := range args {
		if db, ok := arg.(*gorm.DB); ok {
			service.db = db
		}
	}
	return service
}

func (s *Service) Status() map[string]any {
	return map[string]any{
		"status":            "active",
		"history":           "persistent",
		"sqlite_fts5_ready": s.db != nil,
	}
}

func (s *Service) ListHistory(ctx context.Context, userID uint, limit int) ([]HistoryEntry, error) {
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	var rows []database.SearchHistory
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("last_used_at desc, id desc").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	entries := make([]HistoryEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, HistoryEntry{
			ID:           row.ID,
			Query:        row.Query,
			TypeFilter:   row.TypeFilter,
			Genre:        row.Genre,
			Region:       row.Region,
			Year:         row.Year,
			MinRating:    row.MinRating,
			WatchedState: row.WatchedState,
			Sort:         row.Sort,
			LastUsedAt:   row.LastUsedAt,
		})
	}
	return entries, nil
}
