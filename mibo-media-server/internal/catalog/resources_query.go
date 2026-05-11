package catalog

import (
	"context"
	"errors"
	"strings"
)

func (s *Service) ListMetadataItemResources(ctx context.Context, metadataItemID uint, libraryID uint) ([]CatalogResourceDetail, error) {
	if metadataItemID == 0 {
		return nil, errors.New("metadata item id is required")
	}
	type row struct {
		ID              uint
		LibraryID       uint
		ResourceType    string
		ResourceShape   string
		DisplayName     string
		Edition         string
		QualityLabel    string
		DurationSeconds *float64
		Status          string
		ProbeStatus     string
		Role            string
		SegmentIndex    int
		ReviewState     string
	}
	query := s.db.WithContext(ctx).
		Table("resources").
		Select("resources.id, resource_library_links.library_id, resources.resource_type, resources.resource_shape, resources.display_name, resources.edition, resources.quality_label, resources.duration_seconds, resources.status, resources.probe_status, resource_metadata_links.role, resource_metadata_links.segment_index, resource_metadata_links.review_state").
		Joins("JOIN resource_metadata_links ON resource_metadata_links.resource_id = resources.id").
		Joins("LEFT JOIN resource_library_links ON resource_library_links.resource_id = resources.id AND resource_library_links.deleted_at IS NULL").
		Where("resource_metadata_links.metadata_item_id = ?", metadataItemID)
	if libraryID != 0 {
		query = query.Where("resource_library_links.library_id = ?", libraryID)
	}
	var rows []row
	if err := query.Order("resource_metadata_links.segment_index asc").Order("resources.id asc").Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]CatalogResourceDetail, 0, len(rows))
	for _, row := range rows {
		items = append(items, CatalogResourceDetail{ID: row.ID, LibraryID: row.LibraryID, ResourceType: strings.TrimSpace(row.ResourceType), ResourceShape: strings.TrimSpace(row.ResourceShape), DisplayName: strings.TrimSpace(row.DisplayName), Edition: strings.TrimSpace(row.Edition), QualityLabel: strings.TrimSpace(row.QualityLabel), DurationSeconds: row.DurationSeconds, Status: strings.TrimSpace(row.Status), ProbeStatus: strings.TrimSpace(row.ProbeStatus), Role: strings.TrimSpace(row.Role), SegmentIndex: row.SegmentIndex, ReviewState: strings.TrimSpace(row.ReviewState)})
	}
	return items, nil
}
