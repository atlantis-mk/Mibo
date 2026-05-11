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
	MetadataItemID   uint       `json:"metadata_item_id"`
	PositionSeconds  int        `json:"position_seconds"`
	DurationSeconds  *int       `json:"duration_seconds,omitempty"`
	PlayedPercentage *float64   `json:"played_percentage,omitempty"`
	ProgressFrameURL string     `json:"progress_frame_url,omitempty"`
	PlayCount        int        `json:"play_count,omitempty"`
	Watched          bool       `json:"watched"`
	Favorite         bool       `json:"favorite"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	LastPlayedAt     *time.Time `json:"last_played_at,omitempty"`
}

type CatalogUserItemEntry struct {
	CatalogUserProgressState
	Item        CatalogListItem  `json:"item"`
	DisplayItem *CatalogListItem `json:"display_item,omitempty"`
	PlayItem    *CatalogListItem `json:"play_item,omitempty"`
}

func (s *Service) ListContinueWatching(ctx context.Context, userID uint, limit int) ([]CatalogUserItemEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var rows []database.UserMetadataData
	if err := s.db.WithContext(ctx).
		Joins("JOIN metadata_items ON metadata_items.id = user_metadata_data.metadata_item_id").
		Where("user_metadata_data.user_id = ? AND user_metadata_data.metadata_item_id > 0", userID).
		Where("metadata_items.deleted_at IS NULL").
		Where("last_played_at IS NOT NULL AND completed_at IS NULL AND position_seconds > 0").
		Order("user_metadata_data.last_played_at desc").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return s.buildUserMetadataEntries(ctx, rows)
}

func (s *Service) ListRecentlyPlayed(ctx context.Context, userID uint, limit int) ([]CatalogUserItemEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows []database.UserMetadataData
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND metadata_item_id > 0", userID).
		Where("last_played_at IS NOT NULL").
		Order("last_played_at desc").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return s.buildUserMetadataEntries(ctx, rows)
}

func (s *Service) ListFavorites(ctx context.Context, userID uint, limit int) ([]CatalogUserItemEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 200
	}
	var metadataRows []database.UserMetadataData
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND favorite = ?", userID, true).
		Order("updated_at desc").
		Limit(limit).
		Find(&metadataRows).Error; err != nil {
		return nil, err
	}
	return s.buildUserMetadataEntries(ctx, metadataRows)
}

func (s *Service) SetFavorite(ctx context.Context, userID, metadataItemID uint, favorite bool) (CatalogUserItemEntry, error) {
	if metadataItemID == 0 {
		return CatalogUserItemEntry{}, errors.New("metadata item id is required")
	}
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", metadataItemID).First(&database.MetadataItem{}).Error; err != nil {
		return CatalogUserItemEntry{}, err
	}
	var row database.UserMetadataData
	err := s.db.WithContext(ctx).Where("user_id = ? AND metadata_item_id = ?", userID, metadataItemID).First(&row).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return CatalogUserItemEntry{}, err
		}
		row = database.UserMetadataData{UserID: userID, MetadataItemID: metadataItemID}
	}
	row.Favorite = favorite
	if row.ID == 0 {
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return CatalogUserItemEntry{}, err
		}
	} else if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return CatalogUserItemEntry{}, err
	}

	entries, err := s.buildUserMetadataEntries(ctx, []database.UserMetadataData{row})
	if err != nil {
		return CatalogUserItemEntry{}, err
	}
	if len(entries) == 0 {
		return CatalogUserItemEntry{}, gorm.ErrRecordNotFound
	}
	return entries[0], nil
}

func (s *Service) buildUserMetadataEntries(ctx context.Context, rows []database.UserMetadataData) ([]CatalogUserItemEntry, error) {
	if len(rows) == 0 {
		return []CatalogUserItemEntry{}, nil
	}
	metadataIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		metadataIDs = append(metadataIDs, row.MetadataItemID)
	}
	var projections []database.LibraryMetadataProjection
	if err := s.db.WithContext(ctx).Where("metadata_item_id IN ? AND hidden = ?", metadataIDs, false).Order("library_id asc").Find(&projections).Error; err != nil {
		return nil, err
	}
	projectionByMetadataID := make(map[uint]database.LibraryMetadataProjection, len(projections))
	for _, projection := range projections {
		if _, exists := projectionByMetadataID[projection.MetadataItemID]; !exists {
			projectionByMetadataID[projection.MetadataItemID] = projection
		}
	}
	orderedProjections := make([]database.LibraryMetadataProjection, 0, len(rows))
	for _, row := range rows {
		if projection, ok := projectionByMetadataID[row.MetadataItemID]; ok {
			orderedProjections = append(orderedProjections, projection)
		}
	}
	listItems, err := s.buildProjectionListItems(ctx, orderedProjections)
	if err != nil {
		return nil, err
	}
	itemByID := make(map[uint]CatalogListItem, len(listItems))
	for _, item := range listItems {
		itemByID[item.MetadataItemID] = item
	}
	entries := make([]CatalogUserItemEntry, 0, len(rows))
	for _, row := range rows {
		item, ok := itemByID[row.MetadataItemID]
		if !ok {
			continue
		}
		entries = append(entries, CatalogUserItemEntry{CatalogUserProgressState: catalogUserMetadataProgressState(row), Item: item})
	}
	return entries, nil
}

func catalogUserMetadataProgressState(row database.UserMetadataData) CatalogUserProgressState {
	return CatalogUserProgressState{UserID: row.UserID, MetadataItemID: row.MetadataItemID, PositionSeconds: row.PositionSeconds, PlayedPercentage: row.PlayedPercentage, ProgressFrameURL: row.ProgressFrameURL, PlayCount: row.PlayCount, Watched: row.CompletedAt != nil, Favorite: row.Favorite, CompletedAt: row.CompletedAt, LastPlayedAt: row.LastPlayedAt}
}
