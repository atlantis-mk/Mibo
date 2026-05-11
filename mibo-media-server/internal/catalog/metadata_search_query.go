package catalog

import (
	"context"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

func (s *Service) SearchProjectionItems(ctx context.Context, libraryID uint, query string, typeFilter string, limit int) ([]CatalogListItem, error) {
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
	like := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	if libraryID != 0 {
		var docs []database.LibrarySearchDocument
		db := s.db.WithContext(ctx).Where("library_id = ? AND item_type IN ?", libraryID, allowedTypes)
		if strings.TrimSpace(query) != "" {
			db = db.Where("LOWER(title) LIKE ? OR LOWER(original_title) LIKE ? OR LOWER(people_text) LIKE ? OR LOWER(tags_text) LIKE ? OR LOWER(provider_ids_text) LIKE ? OR LOWER(resource_text) LIKE ?", like, like, like, like, like, like)
		}
		if err := db.Order("title asc").Order("metadata_item_id asc").Limit(limit).Find(&docs).Error; err != nil {
			return nil, err
		}
		projections := make([]database.LibraryMetadataProjection, 0, len(docs))
		for _, doc := range docs {
			projections = append(projections, database.LibraryMetadataProjection{LibraryID: doc.LibraryID, MetadataItemID: doc.MetadataItemID, ItemType: doc.ItemType, Title: doc.Title, Year: doc.Year, AvailabilityStatus: doc.AvailabilityStatus})
		}
		return s.buildProjectionListItems(ctx, projections)
	}
	var rawProjections []database.LibraryMetadataProjection
	db := s.db.WithContext(ctx).
		Model(&database.LibraryMetadataProjection{}).
		Where("hidden = ?", false).
		Where("availability_status = ?", database.ProjectionAvailabilityAvailable).
		Where("item_type IN ?", allowedTypes)
	if strings.TrimSpace(query) != "" {
		db = db.Where("LOWER(title) LIKE ? OR LOWER(sort_title) LIKE ?", like, like)
	}
	if err := db.
		Order("COALESCE(NULLIF(sort_title, ''), title) asc").
		Order("metadata_item_id asc").
		Order("library_id asc").
		Limit(limit * 4).
		Find(&rawProjections).Error; err != nil {
		return nil, err
	}
	projections := make([]database.LibraryMetadataProjection, 0, limit)
	seenMetadataIDs := make(map[uint]struct{}, len(rawProjections))
	for _, projection := range rawProjections {
		if _, seen := seenMetadataIDs[projection.MetadataItemID]; seen {
			continue
		}
		seenMetadataIDs[projection.MetadataItemID] = struct{}{}
		projections = append(projections, projection)
		if len(projections) == limit {
			break
		}
	}
	return s.buildProjectionListItems(ctx, projections)
}
