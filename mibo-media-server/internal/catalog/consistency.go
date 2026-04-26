package catalog

import (
	"context"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

type ConsistencyReport struct {
	LibraryID                   *uint  `json:"library_id,omitempty"`
	CatalogItemCount            int    `json:"catalog_item_count"`
	CatalogAssetCount           int    `json:"catalog_asset_count"`
	MissingRollupCount          int    `json:"missing_rollup_count"`
	MissingSearchDocumentCount  int    `json:"missing_search_document_count"`
	AvailabilityMismatchCount   int    `json:"availability_mismatch_count"`
	AssetFileLinkGapCount       int    `json:"asset_file_link_gap_count"`
	DuplicateExternalIDCount    int    `json:"duplicate_external_id_count"`
	DuplicateAssetLinkCount     int    `json:"duplicate_asset_link_count"`
	DuplicateAssetFileCount     int    `json:"duplicate_asset_file_count"`
	DuplicateInventoryPathCount int    `json:"duplicate_inventory_path_count"`
	SampleItemIDs               []uint `json:"sample_item_ids,omitempty"`
}

type RebuildResult struct {
	LibraryID          *uint `json:"library_id,omitempty"`
	LibrariesProcessed int   `json:"libraries_processed"`
	ItemsUpdated       int   `json:"items_updated"`
	ProjectionsRebuilt int   `json:"projections_rebuilt"`
}

func (s *Service) CheckConsistency(ctx context.Context, libraryID *uint) (ConsistencyReport, error) {
	items, err := s.loadConsistencyItems(ctx, libraryID)
	if err != nil {
		return ConsistencyReport{}, err
	}
	assets, err := s.loadConsistencyAssets(ctx, libraryID)
	if err != nil {
		return ConsistencyReport{}, err
	}
	rollupIDs, docIDs, err := s.loadProjectionPresence(ctx, items)
	if err != nil {
		return ConsistencyReport{}, err
	}
	availableAssetIDsByItem, assetFileCounts, err := s.loadConsistencyRelations(ctx, items, assets)
	if err != nil {
		return ConsistencyReport{}, err
	}

	report := ConsistencyReport{LibraryID: libraryID, CatalogItemCount: len(items), CatalogAssetCount: len(assets), SampleItemIDs: []uint{}}
	childrenByParent := make(map[uint][]database.CatalogItem, len(items))
	itemByID := make(map[uint]database.CatalogItem, len(items))
	for _, item := range items {
		itemByID[item.ID] = item
		if item.ParentID != nil {
			childrenByParent[*item.ParentID] = append(childrenByParent[*item.ParentID], item)
		}
		if _, ok := rollupIDs[item.ID]; !ok {
			report.MissingRollupCount++
			report.SampleItemIDs = appendSampleID(report.SampleItemIDs, item.ID)
		}
		if _, ok := docIDs[item.ID]; !ok {
			report.MissingSearchDocumentCount++
			report.SampleItemIDs = appendSampleID(report.SampleItemIDs, item.ID)
		}
	}
	expectedByID := make(map[uint]string, len(items))
	for _, item := range items {
		expected := expectedConsistencyAvailabilityForID(item.ID, itemByID, childrenByParent, availableAssetIDsByItem, expectedByID)
		if strings.TrimSpace(item.AvailabilityStatus) != expected {
			report.AvailabilityMismatchCount++
			report.SampleItemIDs = appendSampleID(report.SampleItemIDs, item.ID)
		}
	}
	for _, asset := range assets {
		if assetFileCounts[asset.ID] == 0 {
			report.AssetFileLinkGapCount++
		}
	}
	if report.DuplicateExternalIDCount, err = s.countConsistencyDuplicateGroups(ctx, "catalog_external_ids", "provider, provider_type, external_id", ""); err != nil {
		return ConsistencyReport{}, err
	}
	if report.DuplicateAssetLinkCount, err = s.countConsistencyDuplicateGroups(ctx, "asset_items", "asset_id, item_id, role, segment_index", ""); err != nil {
		return ConsistencyReport{}, err
	}
	if report.DuplicateAssetFileCount, err = s.countConsistencyDuplicateGroups(ctx, "asset_files", "asset_id, file_id, role, part_index", ""); err != nil {
		return ConsistencyReport{}, err
	}
	if report.DuplicateInventoryPathCount, err = s.countConsistencyDuplicateGroups(ctx, "inventory_files", "storage_provider, storage_path", "deleted_at IS NULL"); err != nil {
		return ConsistencyReport{}, err
	}

	sort.Slice(report.SampleItemIDs, func(i, j int) bool { return report.SampleItemIDs[i] < report.SampleItemIDs[j] })
	return report, nil
}

func (s *Service) RebuildDerivedData(ctx context.Context, libraryID *uint) (RebuildResult, error) {
	items, err := s.loadConsistencyItems(ctx, libraryID)
	if err != nil {
		return RebuildResult{}, err
	}
	availableAssetIDsByItem, _, err := s.loadConsistencyRelations(ctx, items, nil)
	if err != nil {
		return RebuildResult{}, err
	}
	childrenByParent := make(map[uint][]database.CatalogItem, len(items))
	itemByID := make(map[uint]database.CatalogItem, len(items))
	for _, item := range items {
		itemByID[item.ID] = item
		if item.ParentID != nil {
			childrenByParent[*item.ParentID] = append(childrenByParent[*item.ParentID], item)
		}
	}
	expectedByID := make(map[uint]string, len(items))

	updated := 0
	for _, item := range items {
		expected := expectedConsistencyAvailabilityForID(item.ID, itemByID, childrenByParent, availableAssetIDsByItem, expectedByID)
		if strings.TrimSpace(item.AvailabilityStatus) == expected {
			continue
		}
		if err := s.db.WithContext(ctx).Model(&database.CatalogItem{}).Where("id = ?", item.ID).Update("availability_status", expected).Error; err != nil {
			return RebuildResult{}, err
		}
		updated++
	}

	libraryIDs := make(map[uint]string)
	for _, item := range items {
		if _, ok := libraryIDs[item.LibraryID]; ok {
			continue
		}
		libraryIDs[item.LibraryID] = libraryRootPath(items, item.LibraryID)
	}
	for libraryIDValue, rootPath := range libraryIDs {
		if err := s.RefreshLibraryProjection(ctx, libraryIDValue, rootPath); err != nil {
			return RebuildResult{}, err
		}
	}

	return RebuildResult{LibraryID: libraryID, LibrariesProcessed: len(libraryIDs), ItemsUpdated: updated, ProjectionsRebuilt: len(libraryIDs)}, nil
}

func (s *Service) loadConsistencyItems(ctx context.Context, libraryID *uint) ([]database.CatalogItem, error) {
	query := s.db.WithContext(ctx).Where("deleted_at IS NULL")
	if libraryID != nil && *libraryID != 0 {
		query = query.Where("library_id = ?", *libraryID)
	}
	var items []database.CatalogItem
	if err := query.Order("id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) loadConsistencyAssets(ctx context.Context, libraryID *uint) ([]database.MediaAsset, error) {
	query := s.db.WithContext(ctx).Where("deleted_at IS NULL")
	if libraryID != nil && *libraryID != 0 {
		query = query.Where("library_id = ?", *libraryID)
	}
	var assets []database.MediaAsset
	if err := query.Order("id asc").Find(&assets).Error; err != nil {
		return nil, err
	}
	return assets, nil
}

func (s *Service) loadProjectionPresence(ctx context.Context, items []database.CatalogItem) (map[uint]struct{}, map[uint]struct{}, error) {
	itemIDs := make([]uint, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}
	rollupIDs := make(map[uint]struct{}, len(itemIDs))
	docIDs := make(map[uint]struct{}, len(itemIDs))
	if len(itemIDs) == 0 {
		return rollupIDs, docIDs, nil
	}
	var rollups []database.ItemRollup
	if err := s.db.WithContext(ctx).Where("item_id IN ?", itemIDs).Find(&rollups).Error; err != nil {
		return nil, nil, err
	}
	for _, rollup := range rollups {
		rollupIDs[rollup.ItemID] = struct{}{}
	}
	var docs []database.CatalogSearchDocument
	if err := s.db.WithContext(ctx).Where("item_id IN ?", itemIDs).Find(&docs).Error; err != nil {
		return nil, nil, err
	}
	for _, doc := range docs {
		docIDs[doc.ItemID] = struct{}{}
	}
	return rollupIDs, docIDs, nil
}

func (s *Service) loadConsistencyRelations(ctx context.Context, items []database.CatalogItem, assets []database.MediaAsset) (map[uint]map[uint]struct{}, map[uint]int, error) {
	itemIDs := make([]uint, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}
	availableAssetIDsByItem := make(map[uint]map[uint]struct{}, len(itemIDs))
	assetFileCounts := make(map[uint]int, len(assets))
	if len(itemIDs) == 0 {
		return availableAssetIDsByItem, assetFileCounts, nil
	}
	var links []database.AssetItem
	if err := s.db.WithContext(ctx).Where("item_id IN ?", itemIDs).Find(&links).Error; err != nil {
		return nil, nil, err
	}
	assetIDs := make([]uint, 0, len(links))
	assetIDSet := map[uint]struct{}{}
	for _, link := range links {
		if _, ok := availableAssetIDsByItem[link.ItemID]; !ok {
			availableAssetIDsByItem[link.ItemID] = map[uint]struct{}{}
		}
		if _, ok := assetIDSet[link.AssetID]; !ok {
			assetIDSet[link.AssetID] = struct{}{}
			assetIDs = append(assetIDs, link.AssetID)
		}
	}
	if len(assetIDs) > 0 {
		var assetRows []database.MediaAsset
		if err := s.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", assetIDs).Find(&assetRows).Error; err != nil {
			return nil, nil, err
		}
		statusByAsset := map[uint]string{}
		for _, asset := range assetRows {
			statusByAsset[asset.ID] = strings.TrimSpace(asset.Status)
		}
		for _, link := range links {
			if statusByAsset[link.AssetID] == AvailabilityAvailable {
				availableAssetIDsByItem[link.ItemID][link.AssetID] = struct{}{}
			}
		}
		var fileLinks []database.AssetFile
		if err := s.db.WithContext(ctx).Where("asset_id IN ?", assetIDs).Find(&fileLinks).Error; err != nil {
			return nil, nil, err
		}
		for _, link := range fileLinks {
			assetFileCounts[link.AssetID]++
		}
	}
	return availableAssetIDsByItem, assetFileCounts, nil
}

func expectedConsistencyAvailabilityForID(itemID uint, itemByID map[uint]database.CatalogItem, childrenByParent map[uint][]database.CatalogItem, availableAssetIDsByItem map[uint]map[uint]struct{}, cache map[uint]string) string {
	if value, ok := cache[itemID]; ok {
		return value
	}
	item := itemByID[itemID]
	children := childrenByParent[itemID]
	availableAssetIDs := availableAssetIDsByItem[itemID]
	if len(children) == 0 {
		if len(availableAssetIDs) > 0 {
			cache[itemID] = AvailabilityAvailable
			return AvailabilityAvailable
		}
		if strings.TrimSpace(item.AvailabilityStatus) == AvailabilityUnaired {
			cache[itemID] = AvailabilityUnaired
			return AvailabilityUnaired
		}
		cache[itemID] = AvailabilityMissing
		return AvailabilityMissing
	}
	hasUnaired := false
	for _, child := range children {
		switch expectedConsistencyAvailabilityForID(child.ID, itemByID, childrenByParent, availableAssetIDsByItem, cache) {
		case AvailabilityAvailable:
			cache[itemID] = AvailabilityAvailable
			return AvailabilityAvailable
		case AvailabilityMissing, AvailabilityNoLocalMedia:
			cache[itemID] = AvailabilityMissing
			return AvailabilityMissing
		case AvailabilityUnaired:
			hasUnaired = true
		}
	}
	if hasUnaired {
		cache[itemID] = AvailabilityUnaired
		return AvailabilityUnaired
	}
	cache[itemID] = AvailabilityNoLocalMedia
	return AvailabilityNoLocalMedia
}

func appendSampleID(ids []uint, value uint) []uint {
	for _, id := range ids {
		if id == value {
			return ids
		}
	}
	if len(ids) >= 10 {
		return ids
	}
	return append(ids, value)
}

func libraryRootPath(items []database.CatalogItem, libraryID uint) string {
	rootPath := ""
	for _, item := range items {
		if item.LibraryID != libraryID {
			continue
		}
		trimmedPath := strings.TrimSpace(item.Path)
		if rootPath == "" || (trimmedPath != "" && len(trimmedPath) < len(rootPath)) {
			rootPath = trimmedPath
		}
	}
	return rootPath
}

func (s *Service) countConsistencyDuplicateGroups(ctx context.Context, table string, groupBy string, where string) (int, error) {
	query := s.db.WithContext(ctx).Table(table)
	if strings.TrimSpace(where) != "" {
		query = query.Where(where)
	}
	type duplicateCountRow struct {
		Count int
	}
	var rows []duplicateCountRow
	if err := query.
		Select("COUNT(*) as count").
		Group(groupBy).
		Having("COUNT(*) > 1").
		Scan(&rows).Error; err != nil {
		return 0, err
	}
	total := 0
	for _, row := range rows {
		total += row.Count - 1
	}
	return total, nil
}
