package catalog

import (
	"context"
	"errors"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Service) RebuildLibraryMetadataProjection(ctx context.Context, libraryID uint, metadataItemID uint) (database.LibraryMetadataProjection, error) {
	if libraryID == 0 || metadataItemID == 0 {
		return database.LibraryMetadataProjection{}, errors.New("library id and metadata item id are required")
	}
	var projection database.LibraryMetadataProjection
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		built, err := buildLibraryMetadataProjection(ctx, tx, libraryID, metadataItemID)
		if err != nil {
			return err
		}
		projection = built
		return tx.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "library_id"}, {Name: "metadata_item_id"}}, DoUpdates: clause.AssignmentColumns([]string{"item_type", "parent_id", "root_id", "title", "sort_title", "year", "availability_status", "resource_count", "available_count", "missing_count", "latest_added_at", "last_projected_at", "updated_at"})}).Create(&projection).Error
	})
	if err != nil {
		return database.LibraryMetadataProjection{}, err
	}
	if err := s.db.WithContext(ctx).Where("library_id = ? AND metadata_item_id = ?", libraryID, metadataItemID).First(&projection).Error; err != nil {
		return database.LibraryMetadataProjection{}, err
	}
	return projection, nil
}

func (s *Service) RebuildResourceMetadataProjections(ctx context.Context, resourceID uint) error {
	if resourceID == 0 {
		return errors.New("resource id is required")
	}
	var pairs []struct {
		LibraryID      uint
		MetadataItemID uint
	}
	if err := s.db.WithContext(ctx).
		Table("resource_library_links").
		Select("resource_library_links.library_id, resource_metadata_links.metadata_item_id").
		Joins("JOIN resource_metadata_links ON resource_metadata_links.resource_id = resource_library_links.resource_id").
		Where("resource_library_links.resource_id = ? AND resource_library_links.deleted_at IS NULL", resourceID).
		Scan(&pairs).Error; err != nil {
		return err
	}
	for _, pair := range pairs {
		if _, err := s.RebuildLibraryMetadataProjection(ctx, pair.LibraryID, pair.MetadataItemID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) RebuildMetadataItemProjections(ctx context.Context, metadataItemID uint) error {
	if metadataItemID == 0 {
		return errors.New("metadata item id is required")
	}
	metadataIDs := []uint{metadataItemID}
	if ancestorIDs, err := s.metadataAncestorIDs(ctx, metadataItemID); err != nil {
		return err
	} else if len(ancestorIDs) > 0 {
		metadataIDs = append(metadataIDs, ancestorIDs...)
	}
	var libraryIDs []uint
	if err := s.db.WithContext(ctx).
		Table("resource_library_links").
		Select("DISTINCT resource_library_links.library_id").
		Joins("JOIN resource_metadata_links ON resource_metadata_links.resource_id = resource_library_links.resource_id").
		Where("resource_metadata_links.metadata_item_id = ? AND resource_library_links.deleted_at IS NULL", metadataItemID).
		Pluck("library_id", &libraryIDs).Error; err != nil {
		return err
	}
	for _, libraryID := range libraryIDs {
		for _, id := range metadataIDs {
			if _, err := s.RebuildLibraryMetadataProjection(ctx, libraryID, id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) metadataAncestorIDs(ctx context.Context, metadataItemID uint) ([]uint, error) {
	if metadataItemID == 0 {
		return nil, nil
	}
	var item database.MetadataItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", metadataItemID).First(&item).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, 2)
	seen := map[uint]struct{}{}
	appendID := func(id *uint) {
		if id == nil || *id == 0 {
			return
		}
		if _, ok := seen[*id]; ok {
			return
		}
		seen[*id] = struct{}{}
		ids = append(ids, *id)
	}
	appendID(item.ParentID)
	appendID(item.RootID)
	return ids, nil
}

func (s *Service) RebuildLibraryMetadataProjections(ctx context.Context, libraryID uint) error {
	if libraryID == 0 {
		return errors.New("library id is required")
	}
	var metadataIDs []uint
	if err := s.db.WithContext(ctx).
		Table("resource_library_links").
		Select("DISTINCT resource_metadata_links.metadata_item_id").
		Joins("JOIN resource_metadata_links ON resource_metadata_links.resource_id = resource_library_links.resource_id").
		Where("resource_library_links.library_id = ? AND resource_library_links.deleted_at IS NULL", libraryID).
		Pluck("metadata_item_id", &metadataIDs).Error; err != nil {
		return err
	}
	var err error
	metadataIDs, err = s.expandMetadataProjectionIDsWithAncestors(ctx, metadataIDs)
	if err != nil {
		return err
	}
	for _, metadataItemID := range metadataIDs {
		if _, err := s.RebuildLibraryMetadataProjection(ctx, libraryID, metadataItemID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) expandMetadataProjectionIDsWithAncestors(ctx context.Context, metadataIDs []uint) ([]uint, error) {
	seen := make(map[uint]struct{}, len(metadataIDs))
	ordered := make([]uint, 0, len(metadataIDs)*3)
	var appendID func(uint) error
	appendID = func(metadataID uint) error {
		if metadataID == 0 {
			return nil
		}
		if _, ok := seen[metadataID]; ok {
			return nil
		}
		seen[metadataID] = struct{}{}
		ordered = append(ordered, metadataID)
		ancestors, err := s.metadataAncestorIDs(ctx, metadataID)
		if err != nil {
			return err
		}
		for _, ancestorID := range ancestors {
			if err := appendID(ancestorID); err != nil {
				return err
			}
		}
		return nil
	}
	for _, metadataID := range metadataIDs {
		if err := appendID(metadataID); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

func buildLibraryMetadataProjection(ctx context.Context, tx *gorm.DB, libraryID uint, metadataItemID uint) (database.LibraryMetadataProjection, error) {
	var item database.MetadataItem
	if err := tx.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", metadataItemID).First(&item).Error; err != nil {
		return database.LibraryMetadataProjection{}, err
	}
	var links []database.ResourceLibraryLink
	if err := tx.WithContext(ctx).
		Joins("JOIN resource_metadata_links ON resource_metadata_links.resource_id = resource_library_links.resource_id").
		Where("resource_library_links.library_id = ? AND resource_metadata_links.metadata_item_id = ? AND resource_library_links.deleted_at IS NULL", libraryID, metadataItemID).
		Find(&links).Error; err != nil {
		return database.LibraryMetadataProjection{}, err
	}
	projection := database.LibraryMetadataProjection{LibraryID: libraryID, MetadataItemID: metadataItemID, ItemType: item.ItemType, ParentID: item.ParentID, RootID: item.RootID, Title: item.Title, SortTitle: item.SortTitle, Year: item.Year, AvailabilityStatus: database.ProjectionAvailabilityUnavailable, LastProjectedAt: time.Now().UTC()}
	for _, link := range links {
		projection.ResourceCount++
		projection.LatestAddedAt = maxTimePtr(projection.LatestAddedAt, &link.FirstSeenAt)
		switch link.Status {
		case "available":
			projection.AvailableCount++
		case "missing":
			projection.MissingCount++
		}
	}
	if projection.AvailableCount > 0 {
		projection.AvailabilityStatus = database.ProjectionAvailabilityAvailable
	} else if projection.MissingCount > 0 {
		projection.AvailabilityStatus = database.ProjectionAvailabilityMissing
	}
	if projection.ResourceCount == 0 {
		if err := applyChildProjectionRollup(ctx, tx, &projection); err != nil {
			return database.LibraryMetadataProjection{}, err
		}
	}
	return projection, nil
}

func applyChildProjectionRollup(ctx context.Context, tx *gorm.DB, projection *database.LibraryMetadataProjection) error {
	if projection == nil || projection.MetadataItemID == 0 {
		return nil
	}
	var children []database.LibraryMetadataProjection
	if err := tx.WithContext(ctx).Where("library_id = ? AND parent_id = ?", projection.LibraryID, projection.MetadataItemID).Find(&children).Error; err != nil {
		return err
	}
	for _, child := range children {
		projection.ChildCount++
		projection.ResourceCount += child.ResourceCount
		projection.AvailableCount += child.AvailableCount
		projection.MissingCount += child.MissingCount
		projection.LatestAddedAt = maxTimePtr(projection.LatestAddedAt, child.LatestAddedAt)
	}
	if projection.AvailableCount > 0 && projection.MissingCount > 0 {
		projection.AvailabilityStatus = database.ProjectionAvailabilityPartial
	} else if projection.AvailableCount > 0 {
		projection.AvailabilityStatus = database.ProjectionAvailabilityAvailable
	} else if projection.MissingCount > 0 {
		projection.AvailabilityStatus = database.ProjectionAvailabilityMissing
	}
	return nil
}

func maxTimePtr(current *time.Time, candidate *time.Time) *time.Time {
	if candidate == nil {
		return current
	}
	if current == nil || candidate.After(*current) {
		value := *candidate
		return &value
	}
	return current
}
