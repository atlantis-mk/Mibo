package catalog

import (
	"context"
	"errors"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

func (s *Service) ListMetadataSeriesSeasons(ctx context.Context, seriesID uint, libraryID uint) ([]CatalogSeasonDetail, error) {
	if seriesID == 0 {
		return nil, errors.New("metadata series id is required")
	}
	var series database.MetadataItem
	if err := s.db.WithContext(ctx).Where("id = ? AND item_type = ? AND deleted_at IS NULL", seriesID, database.MetadataItemTypeSeries).First(&series).Error; err != nil {
		return nil, err
	}
	var seasons []database.MetadataItem
	if err := s.db.WithContext(ctx).
		Where("parent_id = ? AND item_type = ? AND deleted_at IS NULL", series.ID, database.MetadataItemTypeSeason).
		Order("COALESCE(parent_index_number, 0) asc").
		Order("COALESCE(index_number, 0) asc").
		Order("id asc").
		Find(&seasons).Error; err != nil {
		return nil, err
	}
	if len(seasons) == 0 {
		return []CatalogSeasonDetail{}, nil
	}
	seasonIDs := make([]uint, 0, len(seasons))
	for _, season := range seasons {
		seasonIDs = append(seasonIDs, season.ID)
	}
	var episodes []database.MetadataItem
	if err := s.db.WithContext(ctx).
		Where("parent_id IN ? AND item_type = ? AND deleted_at IS NULL", seasonIDs, database.MetadataItemTypeEpisode).
		Order("COALESCE(parent_index_number, 0) asc").
		Order("COALESCE(index_number, 0) asc").
		Order("id asc").
		Find(&episodes).Error; err != nil {
		return nil, err
	}
	metadataIDs := append([]uint{}, seasonIDs...)
	for _, episode := range episodes {
		metadataIDs = append(metadataIDs, episode.ID)
	}
	projections, err := s.loadMetadataProjections(ctx, metadataIDs, libraryID)
	if err != nil {
		return nil, err
	}
	images, err := s.loadMetadataItemSelectedImages(ctx, metadataIDs)
	if err != nil {
		return nil, err
	}
	externalIDs, err := s.loadMetadataExternalIdentities(ctx, metadataIDs)
	if err != nil {
		return nil, err
	}
	primaryFileIDs, err := s.loadMetadataPrimaryInventoryFileIDs(ctx, metadataIDs, libraryID)
	if err != nil {
		return nil, err
	}
	episodesBySeason := make(map[uint][]CatalogEpisodeDetail, len(seasons))
	for _, episode := range episodes {
		detail := metadataEpisodeDetail(episode, libraryID, projections[episode.ID], images[episode.ID], externalIDs[episode.ID], primaryFileIDs[episode.ID])
		episodesBySeason[derefUintForCatalog(episode.ParentID)] = append(episodesBySeason[derefUintForCatalog(episode.ParentID)], detail)
	}
	items := make([]CatalogSeasonDetail, 0, len(seasons))
	for _, season := range seasons {
		items = append(items, metadataSeasonDetail(season, libraryID, projections[season.ID], images[season.ID], externalIDs[season.ID], episodesBySeason[season.ID]))
	}
	return items, nil
}

func (s *Service) loadMetadataProjections(ctx context.Context, metadataIDs []uint, libraryID uint) (map[uint]database.LibraryMetadataProjection, error) {
	result := make(map[uint]database.LibraryMetadataProjection, len(metadataIDs))
	if len(metadataIDs) == 0 || libraryID == 0 {
		return result, nil
	}
	var projections []database.LibraryMetadataProjection
	if err := s.db.WithContext(ctx).Where("library_id = ? AND metadata_item_id IN ?", libraryID, metadataIDs).Find(&projections).Error; err != nil {
		return nil, err
	}
	for _, projection := range projections {
		result[projection.MetadataItemID] = projection
	}
	return result, nil
}

func metadataSeasonDetail(item database.MetadataItem, libraryID uint, projection database.LibraryMetadataProjection, images []CatalogSelectedImage, externalIDs []CatalogExternalIdentity, episodes []CatalogEpisodeDetail) CatalogSeasonDetail {
	return CatalogSeasonDetail{ID: item.ID, LibraryID: libraryID, Type: metadataItemTypeToCatalogType(item.ItemType), Title: strings.TrimSpace(item.Title), Overview: item.Overview, Year: item.Year, IndexNumber: item.IndexNumber, RuntimeSeconds: item.RuntimeSeconds, AvailabilityStatus: metadataProjectionAvailability(projection), GovernanceStatus: strings.TrimSpace(item.GovernanceStatus), ChildSummary: &CatalogChildSummary{ChildCount: projection.ChildCount, AvailableCount: projection.AvailableCount, MissingCount: projection.MissingCount}, SelectedImages: ensureCatalogSelectedImages(images), ExternalIdentities: ensureCatalogExternalIdentities(externalIDs), SourceEvidence: []CatalogSourceEvidence{}, FieldStates: []CatalogFieldState{}, Episodes: ensureCatalogEpisodeDetails(episodes)}
}

func metadataEpisodeDetail(item database.MetadataItem, libraryID uint, projection database.LibraryMetadataProjection, images []CatalogSelectedImage, externalIDs []CatalogExternalIdentity, inventoryFileID uint) CatalogEpisodeDetail {
	return CatalogEpisodeDetail{ID: item.ID, LibraryID: libraryID, Type: metadataItemTypeToCatalogType(item.ItemType), Title: strings.TrimSpace(item.Title), Overview: item.Overview, Year: item.Year, ParentIndexNumber: item.ParentIndexNumber, IndexNumber: item.IndexNumber, IndexNumberEnd: item.IndexNumberEnd, AbsoluteNumber: item.AbsoluteNumber, RuntimeSeconds: item.RuntimeSeconds, InventoryFileID: inventoryFileID, AvailabilityStatus: metadataProjectionAvailability(projection), GovernanceStatus: strings.TrimSpace(item.GovernanceStatus), ReleaseDate: item.ReleaseDate, FirstAirDate: item.FirstAirDate, SelectedImages: ensureCatalogSelectedImages(images), ExternalIdentities: ensureCatalogExternalIdentities(externalIDs), SourceEvidence: []CatalogSourceEvidence{}, FieldStates: []CatalogFieldState{}}
}

func (s *Service) loadMetadataPrimaryInventoryFileIDs(ctx context.Context, metadataIDs []uint, libraryID uint) (map[uint]uint, error) {
	result := make(map[uint]uint, len(metadataIDs))
	if len(metadataIDs) == 0 || libraryID == 0 {
		return result, nil
	}
	type row struct {
		MetadataItemID  uint
		InventoryFileID uint
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("resource_metadata_links").
		Select("resource_metadata_links.metadata_item_id, resource_files.inventory_file_id").
		Joins("JOIN resource_library_links ON resource_library_links.resource_id = resource_metadata_links.resource_id AND resource_library_links.library_id = ? AND resource_library_links.deleted_at IS NULL", libraryID).
		Joins("JOIN resource_files ON resource_files.resource_id = resource_metadata_links.resource_id AND resource_files.role = ?", database.ResourceFileRoleSource).
		Where("resource_metadata_links.metadata_item_id IN ?", metadataIDs).
		Order("resource_metadata_links.segment_index asc, resource_files.part_index asc, resource_files.id asc").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if _, exists := result[row.MetadataItemID]; !exists {
			result[row.MetadataItemID] = row.InventoryFileID
		}
	}
	return result, nil
}

func metadataProjectionAvailability(projection database.LibraryMetadataProjection) string {
	if strings.TrimSpace(projection.AvailabilityStatus) == "" {
		return database.ProjectionAvailabilityUnavailable
	}
	return strings.TrimSpace(projection.AvailabilityStatus)
}

func derefUintForCatalog(value *uint) uint {
	if value == nil {
		return 0
	}
	return *value
}
