package catalog

import (
	"context"
	"errors"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

type CatalogUserProgressState struct {
	UserID           uint       `json:"user_id"`
	ItemID           uint       `json:"item_id"`
	AssetID          *uint      `json:"asset_id,omitempty"`
	PositionSeconds  int        `json:"position_seconds"`
	DurationSeconds  *int       `json:"duration_seconds,omitempty"`
	PlayedPercentage *float64   `json:"played_percentage,omitempty"`
	PlayCount        int        `json:"play_count,omitempty"`
	Watched          bool       `json:"watched"`
	Favorite         bool       `json:"favorite"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	LastPlayedAt     *time.Time `json:"last_played_at,omitempty"`
}

type CatalogUserItemEntry struct {
	CatalogUserProgressState
	Item CatalogListItem `json:"item"`
}

func (s *Service) ListContinueWatching(ctx context.Context, userID uint, limit int) ([]CatalogUserItemEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows []database.UserItemData
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND item_id > 0", userID).
		Where("last_played_at IS NOT NULL AND completed_at IS NULL AND position_seconds > 0").
		Order("last_played_at desc").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	return s.buildUserItemEntries(ctx, rows)
}

func (s *Service) ListFavorites(ctx context.Context, userID uint, limit int) ([]CatalogUserItemEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 200
	}

	var rows []database.UserItemData
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND item_id > 0 AND favorite = ?", userID, true).
		Order("updated_at desc").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	return s.buildUserItemEntries(ctx, rows)
}

func (s *Service) SetFavorite(ctx context.Context, userID, itemID uint, favorite bool) (CatalogUserItemEntry, error) {
	if itemID == 0 {
		return CatalogUserItemEntry{}, errors.New("item_id is required")
	}

	var item database.CatalogItem
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", itemID).
		First(&item).Error; err != nil {
		return CatalogUserItemEntry{}, err
	}

	var row database.UserItemData
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND item_id = ? AND asset_id IS NULL", userID, itemID).
		First(&row).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return CatalogUserItemEntry{}, err
		}
		row = database.UserItemData{UserID: userID, ItemID: itemID}
	}

	row.Favorite = favorite
	if row.ID == 0 {
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return CatalogUserItemEntry{}, err
		}
	} else if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return CatalogUserItemEntry{}, err
	}

	entries, err := s.buildUserItemEntries(ctx, []database.UserItemData{row})
	if err != nil {
		return CatalogUserItemEntry{}, err
	}
	if len(entries) == 0 {
		return CatalogUserItemEntry{}, gorm.ErrRecordNotFound
	}
	return entries[0], nil
}

func (s *Service) buildUserItemEntries(ctx context.Context, rows []database.UserItemData) ([]CatalogUserItemEntry, error) {
	if len(rows) == 0 {
		return []CatalogUserItemEntry{}, nil
	}

	itemIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		itemIDs = append(itemIDs, row.ItemID)
	}

	var items []database.CatalogItem
	if err := s.db.WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL", itemIDs).
		Find(&items).Error; err != nil {
		return nil, err
	}
	itemByID := make(map[uint]database.CatalogItem, len(items))
	orderedItems := make([]database.CatalogItem, 0, len(rows))
	for _, item := range items {
		itemByID[item.ID] = item
	}
	for _, row := range rows {
		if item, ok := itemByID[row.ItemID]; ok {
			orderedItems = append(orderedItems, item)
		}
	}

	listItems, err := s.buildCatalogListItems(ctx, orderedItems)
	if err != nil {
		return nil, err
	}
	listItemByID := make(map[uint]CatalogListItem, len(listItems))
	for _, item := range listItems {
		listItemByID[item.ID] = item
	}

	entries := make([]CatalogUserItemEntry, 0, len(rows))
	for _, row := range rows {
		item, ok := listItemByID[row.ItemID]
		if !ok {
			continue
		}
		entries = append(entries, CatalogUserItemEntry{
			CatalogUserProgressState: catalogUserProgressState(row, item.RuntimeSeconds),
			Item:                     item,
		})
	}

	return entries, nil
}

func catalogUserProgressState(row database.UserItemData, duration *int) CatalogUserProgressState {
	return CatalogUserProgressState{
		UserID:           row.UserID,
		ItemID:           row.ItemID,
		AssetID:          row.AssetID,
		PositionSeconds:  row.PositionSeconds,
		DurationSeconds:  duration,
		PlayedPercentage: row.PlayedPercentage,
		PlayCount:        row.PlayCount,
		Watched:          row.CompletedAt != nil,
		Favorite:         row.Favorite,
		CompletedAt:      row.CompletedAt,
		LastPlayedAt:     row.LastPlayedAt,
	}
}
