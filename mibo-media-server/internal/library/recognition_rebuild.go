package library

import (
	"context"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/recognition"
	"gorm.io/gorm"
)

func (s *Service) ResetRecognitionLibraryState(ctx context.Context, libraryID uint) error {
	if libraryID == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		resetRepo := recognition.NewRepository(tx)
		if err := resetRepo.DeleteLibraryManifests(ctx, libraryID); err != nil {
			return err
		}
		var projectedMetadataIDs []uint
		if err := tx.Model(&database.LibraryMetadataProjection{}).
			Where("library_id = ?", libraryID).
			Distinct("metadata_item_id").
			Pluck("metadata_item_id", &projectedMetadataIDs).Error; err != nil {
			return err
		}
		if err := tx.Where("library_id = ?", libraryID).Delete(&database.LibraryMetadataProjection{}).Error; err != nil {
			return err
		}
		var resourceIDs []uint
		if err := tx.Model(&database.ResourceLibraryLink{}).Where("library_id = ?", libraryID).Distinct("resource_id").Pluck("resource_id", &resourceIDs).Error; err != nil {
			return err
		}
		if len(resourceIDs) > 0 {
			if err := tx.Where("resource_id IN ?", resourceIDs).Delete(&database.ResourceMetadataLink{}).Error; err != nil {
				return err
			}
			if err := tx.Where("resource_id IN ?", resourceIDs).Delete(&database.ResourceFile{}).Error; err != nil {
				return err
			}
			if err := tx.Where("resource_id IN ?", resourceIDs).Delete(&database.ResourceLibraryLink{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", resourceIDs).Delete(&database.Resource{}).Error; err != nil {
				return err
			}
		}
		metadataIDs := make([]uint, 0, len(projectedMetadataIDs))
		if len(projectedMetadataIDs) > 0 {
			if err := tx.Model(&database.MetadataItem{}).
				Where("id IN ?", projectedMetadataIDs).
				Where("id NOT IN (?)", tx.Model(&database.ResourceMetadataLink{}).Select("metadata_item_id")).
				Distinct("id").
				Pluck("id", &metadataIDs).Error; err != nil {
				return err
			}
		}
		if len(metadataIDs) > 0 {
			if err := tx.Where("metadata_item_id IN ?", metadataIDs).Delete(&database.MetadataExternalID{}).Error; err != nil {
				return err
			}
			if err := tx.Where("metadata_item_id IN ?", metadataIDs).Delete(&database.MetadataItemImage{}).Error; err != nil {
				return err
			}
			if err := tx.Where("metadata_item_id IN ?", metadataIDs).Delete(&database.MetadataItemPerson{}).Error; err != nil {
				return err
			}
			if err := tx.Where("metadata_item_id IN ?", metadataIDs).Delete(&database.MetadataItemTag{}).Error; err != nil {
				return err
			}
			if err := tx.Where("metadata_item_id IN ?", metadataIDs).Delete(&database.MetadataItemFieldState{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", metadataIDs).Delete(&database.MetadataItem{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) RebuildRecognitionLibraryState(ctx context.Context, libraryID uint) error {
	if libraryID == 0 {
		return nil
	}
	if err := s.ResetRecognitionLibraryState(ctx, libraryID); err != nil {
		return err
	}
	catalogSvc := catalog.NewService(s.db)
	return catalogSvc.RebuildLibraryMetadataProjections(ctx, libraryID)
}
