package library

import (
	"context"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

func (s *Service) GetLibrary(ctx context.Context, libraryID uint) (LibraryDetail, error) {
	var detail LibraryDetail
	if err := s.db.WithContext(ctx).First(&detail.Library, libraryID).Error; err != nil {
		return LibraryDetail{}, err
	}
	if err := s.db.WithContext(ctx).Model(&database.CatalogItem{}).
		Where("library_id = ? AND deleted_at IS NULL", libraryID).
		Where("parent_id IS NULL").
		Count(&detail.CatalogItemsCount).Error; err != nil {
		return LibraryDetail{}, err
	}
	if err := s.db.WithContext(ctx).Model(&database.InventoryFile{}).Where("library_id = ? AND deleted_at IS NULL", libraryID).Count(&detail.InventoryFilesCount).Error; err != nil {
		return LibraryDetail{}, err
	}
	return detail, nil
}

func ParseBrowseYear(value string) *int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	year, err := strconv.Atoi(trimmed)
	if err != nil || year <= 0 {
		return nil
	}
	return &year
}
