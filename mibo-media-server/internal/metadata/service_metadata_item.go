package metadata

import (
	"context"
	"fmt"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func (s *Service) MatchMetadataItemOperation(ctx context.Context, metadataItemID uint, libraryID uint) (MetadataOperationResult, error) {
	return s.runMetadataOperation(ctx, MetadataOperationRequest{Operation: OperationTypeMatch, OriginMetadataItemID: metadataItemID, TargetMetadataItemID: metadataItemID, LibraryID: libraryID})
}

func (s *Service) RefetchMetadataItemOperation(ctx context.Context, metadataItemID uint, libraryID uint) (MetadataOperationResult, error) {
	return s.runMetadataOperation(ctx, MetadataOperationRequest{Operation: OperationTypeRefetch, OriginMetadataItemID: metadataItemID, TargetMetadataItemID: metadataItemID, LibraryID: libraryID})
}

func (s *Service) ApplyMetadataCandidateOperation(ctx context.Context, metadataItemID uint, libraryID uint, input ApplyCandidateInput) (MetadataOperationResult, error) {
	return s.runMetadataOperation(ctx, MetadataOperationRequest{Operation: OperationTypeManualApply, OriginMetadataItemID: metadataItemID, TargetMetadataItemID: metadataItemID, LibraryID: libraryID, ManualCandidateExternalID: input.ExternalID})
}

func (s *Service) resolveMetadataItemOperationTarget(ctx context.Context, input MetadataOperationRequest) (database.MetadataItem, MetadataExecutionPlan, error) {
	metadataItemID := input.TargetMetadataItemID
	if metadataItemID == 0 {
		metadataItemID = input.OriginMetadataItemID
	}
	if metadataItemID == 0 {
		return database.MetadataItem{}, MetadataExecutionPlan{}, fmt.Errorf("metadata item id is required")
	}
	var item database.MetadataItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", metadataItemID).First(&item).Error; err != nil {
		return database.MetadataItem{}, MetadataExecutionPlan{}, err
	}
	libraryID := input.LibraryID
	if libraryID == 0 {
		resolvedLibraryID, err := s.resolveMetadataOperationLibraryID(ctx, item.ID)
		if err != nil {
			return database.MetadataItem{}, MetadataExecutionPlan{}, err
		}
		libraryID = resolvedLibraryID
	}
	plan, err := s.resolveMetadataExecutionPlan(ctx, libraryID)
	if err != nil {
		return database.MetadataItem{}, MetadataExecutionPlan{}, err
	}
	return item, plan, nil
}

func (s *Service) resolveMetadataOperationLibraryID(ctx context.Context, metadataItemID uint) (uint, error) {
	var link database.ResourceLibraryLink
	err := s.db.WithContext(ctx).
		Joins("JOIN resource_metadata_links ON resource_metadata_links.resource_id = resource_library_links.resource_id").
		Where("resource_metadata_links.metadata_item_id = ?", metadataItemID).
		Where("resource_library_links.deleted_at IS NULL").
		Order("resource_library_links.last_seen_at desc, resource_library_links.id asc").
		First(&link).Error
	if err == nil {
		return link.LibraryID, nil
	}
	if !isRecordNotFound(err) {
		return 0, err
	}
	var projection database.LibraryMetadataProjection
	err = s.db.WithContext(ctx).Where("metadata_item_id = ?", metadataItemID).Order("latest_added_at desc, id asc").First(&projection).Error
	if err == nil {
		return projection.LibraryID, nil
	}
	if !isRecordNotFound(err) {
		return 0, err
	}
	return 0, fmt.Errorf("metadata item %d has no library context", metadataItemID)
}

func metadataItemToSearchItem(item database.MetadataItem, libraryID uint) metadataSearchItem {
	searchItem := metadataSearchItem{LibraryID: libraryID, Type: metadataItemTypeToCatalogType(item.ItemType), Title: strings.TrimSpace(item.Title), OriginalTitle: strings.TrimSpace(item.OriginalTitle), Overview: item.Overview, Year: item.Year}
	if item.ItemType == database.MetadataItemTypeSeries || item.ItemType == database.MetadataItemTypeSeason || item.ItemType == database.MetadataItemTypeEpisode {
		searchItem.SeriesTitle = strings.TrimSpace(item.Title)
	}
	if item.ItemType == database.MetadataItemTypeSeason {
		searchItem.SeasonNumber = item.IndexNumber
	}
	if item.ItemType == database.MetadataItemTypeEpisode {
		searchItem.SeasonNumber = item.ParentIndexNumber
		searchItem.EpisodeNumber = item.IndexNumber
	}
	return searchItem
}

func metadataItemTypeToCatalogType(itemType string) string {
	switch strings.TrimSpace(itemType) {
	case database.MetadataItemTypeSeries:
		return catalog.ItemTypeSeries
	case database.MetadataItemTypeSeason:
		return catalog.ItemTypeSeason
	case database.MetadataItemTypeEpisode:
		return catalog.ItemTypeEpisode
	default:
		return catalog.ItemTypeMovie
	}
}

func isRecordNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
