package catalog

import (
	"context"
	"errors"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

func (s *Service) GetMetadataItemDetail(ctx context.Context, metadataItemID uint, libraryID uint) (CatalogItemDetail, error) {
	if metadataItemID == 0 {
		return CatalogItemDetail{}, errors.New("metadata item id is required")
	}
	var item database.MetadataItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", metadataItemID).First(&item).Error; err != nil {
		return CatalogItemDetail{}, err
	}
	projection := database.LibraryMetadataProjection{LibraryID: libraryID, MetadataItemID: item.ID, ItemType: item.ItemType, Title: item.Title, SortTitle: item.SortTitle, Year: item.Year, AvailabilityStatus: database.ProjectionAvailabilityUnavailable}
	if libraryID != 0 {
		_ = s.db.WithContext(ctx).Where("library_id = ? AND metadata_item_id = ?", libraryID, item.ID).First(&projection).Error
	}
	imagesByItem, err := s.loadMetadataItemSelectedImages(ctx, []uint{item.ID})
	if err != nil {
		return CatalogItemDetail{}, err
	}
	identitiesByItem, err := s.loadMetadataExternalIdentities(ctx, []uint{item.ID})
	if err != nil {
		return CatalogItemDetail{}, err
	}
	resources, err := s.loadMetadataResourceDetails(ctx, item.ID, libraryID)
	if err != nil {
		return CatalogItemDetail{}, err
	}
	var seasons []CatalogSeasonDetail
	if item.ItemType == database.MetadataItemTypeSeries {
		seasons, err = s.ListMetadataSeriesSeasons(ctx, item.ID, libraryID)
		if err != nil {
			return CatalogItemDetail{}, err
		}
	}
	return CatalogItemDetail{ID: item.ID, MetadataItemID: item.ID, LibraryID: libraryID, Type: metadataItemTypeToCatalogType(item.ItemType), Title: strings.TrimSpace(item.Title), OriginalTitle: strings.TrimSpace(item.OriginalTitle), SortTitle: strings.TrimSpace(item.SortTitle), Overview: item.Overview, Year: item.Year, EndYear: item.EndYear, RuntimeSeconds: item.RuntimeSeconds, IndexNumber: item.IndexNumber, ParentIndexNumber: item.ParentIndexNumber, CommunityRating: item.CommunityRating, OfficialRating: strings.TrimSpace(item.OfficialRating), SeriesStatus: strings.TrimSpace(item.SeriesStatus), AvailabilityStatus: projection.AvailabilityStatus, GovernanceStatus: strings.TrimSpace(item.GovernanceStatus), ReleaseDate: item.ReleaseDate, FirstAirDate: item.FirstAirDate, LastAirDate: item.LastAirDate, ResourceCount: projection.ResourceCount, AvailableCount: projection.AvailableCount, MissingCount: projection.MissingCount, ChildSummary: &CatalogChildSummary{ChildCount: projection.ChildCount, AvailableCount: projection.AvailableCount, MissingCount: projection.MissingCount}, SelectedImages: imagesByItem[item.ID], ExternalIdentities: identitiesByItem[item.ID], Tags: []CatalogTagDetail{}, Genres: []string{}, SourceEvidence: []CatalogSourceEvidence{}, FieldStates: []CatalogFieldState{}, Cast: []CatalogPersonDetail{}, Directors: []CatalogPersonDetail{}, Seasons: ensureCatalogSeasonDetails(seasons), Episodes: []CatalogEpisodeDetail{}, SameSeasonEpisodes: []CatalogEpisodeShelfItem{}, Resources: resources, RelatedItems: []CatalogListItem{}}, nil
}

func (s *Service) loadMetadataResourceDetails(ctx context.Context, metadataItemID uint, libraryID uint) ([]CatalogResourceDetailFull, error) {
	type resourceRow struct {
		ID              uint
		ResourceType    string
		DisplayName     string
		Edition         string
		QualityLabel    string
		DurationSeconds *float64
		Status          string
		ProbeStatus     string
		LibraryID       uint
	}
	query := s.db.WithContext(ctx).
		Table("resources").
		Select("resources.id, resources.resource_type, resources.display_name, resources.edition, resources.quality_label, resources.duration_seconds, resources.status, resources.probe_status, resource_library_links.library_id").
		Joins("JOIN resource_metadata_links ON resource_metadata_links.resource_id = resources.id").
		Joins("JOIN resource_library_links ON resource_library_links.resource_id = resources.id").
		Where("resource_metadata_links.metadata_item_id = ? AND resource_library_links.deleted_at IS NULL", metadataItemID)
	if libraryID != 0 {
		query = query.Where("resource_library_links.library_id = ?", libraryID)
	}
	var rows []resourceRow
	if err := query.Order("resources.id asc").Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]CatalogResourceDetailFull, 0, len(rows))
	for _, row := range rows {
		fileIDs, files, streams, err := s.loadMetadataResourceFileSummaries(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, CatalogResourceDetailFull{ID: row.ID, LibraryID: row.LibraryID, ResourceType: row.ResourceType, DisplayName: strings.TrimSpace(row.DisplayName), Edition: strings.TrimSpace(row.Edition), QualityLabel: strings.TrimSpace(row.QualityLabel), DurationSeconds: row.DurationSeconds, Status: strings.TrimSpace(row.Status), ProbeStatus: strings.TrimSpace(row.ProbeStatus), FileIDs: fileIDs, Files: files, Streams: streams, Links: []CatalogResourceLink{}})
	}
	return items, nil
}

func (s *Service) loadMetadataResourceFileSummaries(ctx context.Context, resourceID uint) ([]uint, []CatalogResourceFileSummary, []CatalogMediaStreamSummary, error) {
	var resourceFiles []database.ResourceFile
	if err := s.db.WithContext(ctx).Where("resource_id = ?", resourceID).Order("role asc, part_index asc, id asc").Find(&resourceFiles).Error; err != nil {
		return nil, nil, nil, err
	}
	if len(resourceFiles) == 0 {
		return []uint{}, []CatalogResourceFileSummary{}, []CatalogMediaStreamSummary{}, nil
	}
	fileIDs := make([]uint, 0, len(resourceFiles))
	for _, file := range resourceFiles {
		fileIDs = append(fileIDs, file.InventoryFileID)
	}
	var inventoryFiles []database.InventoryFile
	if err := s.db.WithContext(ctx).Where("id IN ?", fileIDs).Find(&inventoryFiles).Error; err != nil {
		return nil, nil, nil, err
	}
	inventoryFilesByID := make(map[uint]database.InventoryFile, len(inventoryFiles))
	for _, file := range inventoryFiles {
		inventoryFilesByID[file.ID] = file
	}
	var streams []database.MediaStream
	if err := s.db.WithContext(ctx).Where("file_id IN ?", fileIDs).Order("file_id asc, stream_index asc").Find(&streams).Error; err != nil {
		return nil, nil, nil, err
	}
	streamsByFileID := make(map[uint][]database.MediaStream, len(fileIDs))
	for _, stream := range streams {
		streamsByFileID[stream.FileID] = append(streamsByFileID[stream.FileID], stream)
	}
	fileSummaries, streamSummaries := buildCatalogResourceFileAndStreamSummaries(resourceFiles, inventoryFilesByID, streamsByFileID)
	return fileIDs, fileSummaries, streamSummaries, nil
}
