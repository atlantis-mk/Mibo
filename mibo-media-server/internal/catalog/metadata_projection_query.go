package catalog

import (
	"context"
	"errors"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

func (s *Service) ListLibraryProjectionItems(ctx context.Context, libraryID uint, query string, typeFilter string, limit int) ([]CatalogListItem, error) {
	if libraryID == 0 {
		return nil, errors.New("library id is required")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	allowedTypes := []string{database.MetadataItemTypeMovie, database.MetadataItemTypeSeries}
	switch strings.ToLower(strings.TrimSpace(typeFilter)) {
	case database.MetadataItemTypeMovie:
		allowedTypes = []string{database.MetadataItemTypeMovie}
	case database.MetadataItemTypeSeries, "show":
		allowedTypes = []string{database.MetadataItemTypeSeries}
	}
	db := s.db.WithContext(ctx).
		Model(&database.LibraryMetadataProjection{}).
		Where("library_id = ? AND hidden = ?", libraryID, false).
		Where("item_type IN ?", allowedTypes).
		Where("availability_status = ?", database.ProjectionAvailabilityAvailable)
	if trimmedQuery := strings.TrimSpace(query); trimmedQuery != "" {
		like := "%" + strings.ToLower(trimmedQuery) + "%"
		db = db.Where("LOWER(title) LIKE ? OR LOWER(sort_title) LIKE ?", like, like)
	}
	var projections []database.LibraryMetadataProjection
	if err := db.Order("COALESCE(NULLIF(sort_title, ''), title) asc").Order("metadata_item_id asc").Limit(limit).Find(&projections).Error; err != nil {
		return nil, err
	}
	return s.buildProjectionListItems(ctx, projections)
}

func (s *Service) buildProjectionListItems(ctx context.Context, projections []database.LibraryMetadataProjection) ([]CatalogListItem, error) {
	items := make([]CatalogListItem, 0, len(projections))
	if len(projections) == 0 {
		return items, nil
	}
	metadataIDs := make([]uint, 0, len(projections))
	for _, projection := range projections {
		metadataIDs = append(metadataIDs, projection.MetadataItemID)
	}
	imagesByItem, err := s.loadMetadataItemSelectedImages(ctx, metadataIDs)
	if err != nil {
		return nil, err
	}
	identitiesByItem, err := s.loadMetadataExternalIdentities(ctx, metadataIDs)
	if err != nil {
		return nil, err
	}
	for _, projection := range projections {
		items = append(items, CatalogListItem{ID: projection.MetadataItemID, MetadataItemID: projection.MetadataItemID, LibraryID: projection.LibraryID, Type: metadataItemTypeToCatalogType(projection.ItemType), Title: projection.Title, SortTitle: projection.SortTitle, Year: projection.Year, AvailabilityStatus: projection.AvailabilityStatus, GovernanceStatus: database.ReviewStateAccepted, ResourceCount: projection.ResourceCount, AvailableCount: projection.AvailableCount, MissingCount: projection.MissingCount, ChildSummary: &CatalogChildSummary{ChildCount: projection.ChildCount, AvailableCount: projection.AvailableCount, MissingCount: projection.MissingCount}, SelectedImages: imagesByItem[projection.MetadataItemID], ExternalIdentities: identitiesByItem[projection.MetadataItemID]})
	}
	return items, nil
}

func (s *Service) loadMetadataItemSelectedImages(ctx context.Context, metadataIDs []uint) (map[uint][]CatalogSelectedImage, error) {
	result := make(map[uint][]CatalogSelectedImage, len(metadataIDs))
	if len(metadataIDs) == 0 {
		return result, nil
	}
	var images []database.MetadataItemImage
	if err := s.db.WithContext(ctx).Where("metadata_item_id IN ? AND is_selected = ?", metadataIDs, true).Order("sort_order asc, id asc").Find(&images).Error; err != nil {
		return nil, err
	}
	for _, image := range images {
		result[image.MetadataItemID] = append(result[image.MetadataItemID], CatalogSelectedImage{ImageType: image.ImageType, URL: image.URL, Language: image.Language, Width: image.Width, Height: image.Height})
	}
	return result, nil
}

func (s *Service) loadMetadataItemImages(ctx context.Context, metadataIDs []uint) (map[uint][]CatalogSelectedImage, error) {
	result := make(map[uint][]CatalogSelectedImage, len(metadataIDs))
	if len(metadataIDs) == 0 {
		return result, nil
	}
	var images []database.MetadataItemImage
	if err := s.db.WithContext(ctx).Where("metadata_item_id IN ?", metadataIDs).Order("sort_order asc, id asc").Find(&images).Error; err != nil {
		return nil, err
	}
	for _, image := range images {
		result[image.MetadataItemID] = append(result[image.MetadataItemID], CatalogSelectedImage{ImageType: image.ImageType, URL: image.URL, Language: image.Language, Width: image.Width, Height: image.Height})
	}
	return result, nil
}

func selectedMetadataImages(images []CatalogSelectedImage) []CatalogSelectedImage {
	if images == nil {
		return []CatalogSelectedImage{}
	}
	selected := make([]CatalogSelectedImage, 0, len(images))
	seenTypes := make(map[string]struct{}, len(images))
	for _, image := range images {
		imageType := strings.TrimSpace(image.ImageType)
		if strings.TrimSpace(image.URL) == "" {
			continue
		}
		if _, ok := seenTypes[imageType]; ok {
			continue
		}
		seenTypes[imageType] = struct{}{}
		selected = append(selected, image)
	}
	return selected
}

func (s *Service) loadMetadataExternalIdentities(ctx context.Context, metadataIDs []uint) (map[uint][]CatalogExternalIdentity, error) {
	result := make(map[uint][]CatalogExternalIdentity, len(metadataIDs))
	if len(metadataIDs) == 0 {
		return result, nil
	}
	var externalIDs []database.MetadataExternalID
	if err := s.db.WithContext(ctx).Where("metadata_item_id IN ?", metadataIDs).Order("is_primary desc, id asc").Find(&externalIDs).Error; err != nil {
		return nil, err
	}
	for _, externalID := range externalIDs {
		result[externalID.MetadataItemID] = append(result[externalID.MetadataItemID], CatalogExternalIdentity{Provider: externalID.Provider, ProviderType: externalID.ProviderType, ExternalID: externalID.ExternalID, IsPrimary: externalID.IsPrimary, Confidence: externalID.Confidence})
	}
	return result, nil
}

func metadataItemTypeToCatalogType(itemType string) string {
	switch strings.TrimSpace(itemType) {
	case database.MetadataItemTypeSeries:
		return ItemTypeSeries
	case database.MetadataItemTypeSeason:
		return ItemTypeSeason
	case database.MetadataItemTypeEpisode:
		return ItemTypeEpisode
	default:
		return ItemTypeMovie
	}
}
