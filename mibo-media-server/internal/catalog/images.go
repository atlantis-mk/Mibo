package catalog

import (
	"context"
	"errors"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func (s *Service) SelectImage(ctx context.Context, itemID uint, imageType string, url string) error {
	if itemID == 0 {
		return errors.New("item id is required")
	}
	imageType = strings.TrimSpace(imageType)
	url = strings.TrimSpace(url)
	if imageType == "" || url == "" {
		return errors.New("image type and url are required")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var images []database.ItemImage
		if err := tx.Where("item_id = ? AND image_type = ?", itemID, imageType).Order("id asc").Find(&images).Error; err != nil {
			return err
		}
		found := false
		for _, image := range images {
			isSelected := strings.TrimSpace(image.URL) == url
			if isSelected {
				found = true
			}
			if err := tx.Model(&database.ItemImage{}).Where("id = ?", image.ID).Update("is_selected", isSelected).Error; err != nil {
				return err
			}
		}
		if !found {
			return gorm.ErrRecordNotFound
		}
		return s.refreshProjectionWithDB(ctx, tx, ProjectionRefreshRequest{ItemID: itemID})
	})
}
