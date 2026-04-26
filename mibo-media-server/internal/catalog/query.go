package catalog

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

type CatalogLatestByLibrarySection struct {
	LibraryID   uint              `json:"library_id"`
	LibraryName string            `json:"library_name"`
	Items       []CatalogListItem `json:"items"`
}

func (s *Service) ListLibraryItems(ctx context.Context, libraryID uint, query string, typeFilter string, limit int) ([]CatalogListItem, error) {
	if libraryID == 0 {
		return nil, errors.New("library id is required")
	}
	return s.listItems(ctx, &libraryID, query, typeFilter, limit)
}

func (s *Service) ListItems(ctx context.Context, libraryID uint, query string, typeFilter string, limit int) ([]CatalogListItem, error) {
	var libraryFilter *uint
	if libraryID != 0 {
		libraryFilter = &libraryID
	}
	return s.listItems(ctx, libraryFilter, query, typeFilter, limit)
}

func (s *Service) SearchItems(ctx context.Context, libraryID uint, query string, typeFilter string, limit int) ([]CatalogListItem, error) {
	if strings.TrimSpace(query) == "" {
		return s.ListItems(ctx, libraryID, "", typeFilter, limit)
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	allowedTypes := []string{ItemTypeMovie, ItemTypeSeries}
	switch strings.ToLower(strings.TrimSpace(typeFilter)) {
	case ItemTypeMovie:
		allowedTypes = []string{ItemTypeMovie}
	case ItemTypeSeries, "show":
		allowedTypes = []string{ItemTypeSeries}
	}

	db := s.db.WithContext(ctx).
		Model(&database.CatalogSearchDocument{}).
		Where("item_type IN ?", allowedTypes)
	if libraryID != 0 {
		db = db.Where("library_id = ?", libraryID)
	}
	like := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	db = db.Where("LOWER(title) LIKE ? OR LOWER(original_title) LIKE ? OR LOWER(people_text) LIKE ? OR LOWER(tags_text) LIKE ? OR LOWER(provider_ids_text) LIKE ?", like, like, like, like, like)

	var docs []database.CatalogSearchDocument
	if err := db.Order("title asc").Order("item_id asc").Limit(limit).Find(&docs).Error; err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return []CatalogListItem{}, nil
	}
	itemIDs := make([]uint, 0, len(docs))
	for _, doc := range docs {
		itemIDs = append(itemIDs, doc.ItemID)
	}
	var items []database.CatalogItem
	if err := s.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", itemIDs).Find(&items).Error; err != nil {
		return nil, err
	}
	itemByID := make(map[uint]database.CatalogItem, len(items))
	for _, item := range items {
		itemByID[item.ID] = item
	}
	ordered := make([]database.CatalogItem, 0, len(docs))
	for _, doc := range docs {
		if item, ok := itemByID[doc.ItemID]; ok {
			ordered = append(ordered, item)
		}
	}
	return s.buildCatalogListItems(ctx, ordered)
}

func (s *Service) listItems(ctx context.Context, libraryID *uint, query string, typeFilter string, limit int) ([]CatalogListItem, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	allowedTypes := []string{ItemTypeMovie, ItemTypeSeries}
	switch strings.ToLower(strings.TrimSpace(typeFilter)) {
	case ItemTypeMovie:
		allowedTypes = []string{ItemTypeMovie}
	case ItemTypeSeries, "show":
		allowedTypes = []string{ItemTypeSeries}
	}

	db := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("parent_id IS NULL").
		Where("type IN ?", allowedTypes)
	if libraryID != nil {
		db = db.Where("library_id = ?", *libraryID)
	}
	if trimmedQuery := strings.TrimSpace(query); trimmedQuery != "" {
		like := "%" + strings.ToLower(trimmedQuery) + "%"
		db = db.Where("LOWER(title) LIKE ? OR LOWER(original_title) LIKE ? OR LOWER(sort_title) LIKE ?", like, like, like)
	}

	var items []database.CatalogItem
	if err := db.Order("sort_key asc").Order("title asc").Order("id asc").Limit(limit).Find(&items).Error; err != nil {
		return nil, err
	}
	return s.buildCatalogListItems(ctx, items)
}

func (s *Service) GetItemDetail(ctx context.Context, itemID uint) (CatalogItemDetail, error) {
	item, err := s.loadCatalogItem(ctx, itemID)
	if err != nil {
		return CatalogItemDetail{}, err
	}

	rollups, images, externalIDs, sources, fieldStates, err := s.loadCatalogQueryData(ctx, []uint{item.ID})
	if err != nil {
		return CatalogItemDetail{}, err
	}
	assetsByItem, err := s.loadCatalogAssetsByItem(ctx, []uint{item.ID})
	if err != nil {
		return CatalogItemDetail{}, err
	}

	seasons := []CatalogSeasonDetail{}
	episodes := []CatalogEpisodeDetail{}
	if item.Type == ItemTypeSeries {
		seasons, err = s.ListSeriesSeasons(ctx, item.ID)
		if err != nil {
			return CatalogItemDetail{}, err
		}
	}
	if item.Type == ItemTypeSeason {
		episodes, err = s.buildCatalogEpisodeDetailsForParent(ctx, item.ID)
		if err != nil {
			return CatalogItemDetail{}, err
		}
	}

	return BuildCatalogItemDetail(CatalogItemDetailInput{
		Item:        item,
		Rollup:      rollups[item.ID],
		Images:      images[item.ID],
		ExternalIDs: externalIDs[item.ID],
		Sources:     sources[item.ID],
		FieldStates: fieldStates[item.ID],
		Seasons:     seasons,
		Episodes:    episodes,
		Assets:      assetsByItem[item.ID],
	}), nil
}

func (s *Service) ListSeriesSeasons(ctx context.Context, seriesID uint) ([]CatalogSeasonDetail, error) {
	series, err := s.loadCatalogItem(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	if series.Type != ItemTypeSeries {
		return []CatalogSeasonDetail{}, nil
	}

	var seasons []database.CatalogItem
	if err := s.db.WithContext(ctx).
		Where("parent_id = ? AND type = ? AND deleted_at IS NULL", series.ID, ItemTypeSeason).
		Order("index_number asc").Order("id asc").
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

	rollups, images, externalIDs, sources, fieldStates, err := s.loadCatalogQueryData(ctx, seasonIDs)
	if err != nil {
		return nil, err
	}
	episodesBySeason, err := s.buildCatalogEpisodeDetailsByParent(ctx, seasonIDs)
	if err != nil {
		return nil, err
	}

	result := make([]CatalogSeasonDetail, 0, len(seasons))
	for _, season := range seasons {
		result = append(result, BuildCatalogSeasonDetail(CatalogSeasonDetailInput{
			Item:        season,
			Rollup:      rollups[season.ID],
			Images:      images[season.ID],
			ExternalIDs: externalIDs[season.ID],
			Sources:     sources[season.ID],
			FieldStates: fieldStates[season.ID],
			Episodes:    episodesBySeason[season.ID],
		}))
	}
	return result, nil
}

func (s *Service) GetGovernanceWorkspace(ctx context.Context, itemID uint) (CatalogGovernanceWorkspace, error) {
	item, err := s.loadCatalogItem(ctx, itemID)
	if err != nil {
		return CatalogGovernanceWorkspace{}, err
	}
	_, images, externalIDs, sources, fieldStates, err := s.loadCatalogQueryData(ctx, []uint{item.ID})
	if err != nil {
		return CatalogGovernanceWorkspace{}, err
	}
	assetsByItem, err := s.loadCatalogAssetsByItem(ctx, []uint{item.ID})
	if err != nil {
		return CatalogGovernanceWorkspace{}, err
	}

	children, err := s.ListChildren(ctx, item.ID)
	if err != nil {
		return CatalogGovernanceWorkspace{}, err
	}
	recommendedChildren, err := s.buildCatalogListItems(ctx, children)
	if err != nil {
		return CatalogGovernanceWorkspace{}, err
	}

	return BuildCatalogGovernanceWorkspace(CatalogGovernanceWorkspaceInput{
		Item:                item,
		Images:              images[item.ID],
		ExternalIDs:         externalIDs[item.ID],
		Sources:             sources[item.ID],
		FieldStates:         fieldStates[item.ID],
		Assets:              assetsByItem[item.ID],
		RecommendedChildren: recommendedChildren,
	}), nil
}

func (s *Service) ListRecentlyAdded(ctx context.Context, limit int) ([]CatalogListItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 12
	}
	var items []database.CatalogItem
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("parent_id IS NULL").
		Where("type IN ?", []string{ItemTypeMovie, ItemTypeSeries}).
		Order("created_at desc").Order("id desc").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return s.buildCatalogListItems(ctx, items)
}

func (s *Service) ListLatestByLibrary(ctx context.Context, limit int) ([]CatalogLatestByLibrarySection, error) {
	if limit <= 0 || limit > 50 {
		limit = 12
	}
	var libraries []database.Library
	if err := s.db.WithContext(ctx).
		Where("status = ?", "active").
		Order("name asc").
		Find(&libraries).Error; err != nil {
		return nil, err
	}
	sections := make([]CatalogLatestByLibrarySection, 0, len(libraries))
	for _, library := range libraries {
		var items []database.CatalogItem
		if err := s.db.WithContext(ctx).
			Where("library_id = ? AND deleted_at IS NULL", library.ID).
			Where("parent_id IS NULL").
			Where("type IN ?", []string{ItemTypeMovie, ItemTypeSeries}).
			Order("created_at desc").Order("id desc").
			Limit(limit).
			Find(&items).Error; err != nil {
			return nil, err
		}
		if len(items) == 0 {
			continue
		}
		mapped, err := s.buildCatalogListItems(ctx, items)
		if err != nil {
			return nil, err
		}
		sections = append(sections, CatalogLatestByLibrarySection{
			LibraryID:   library.ID,
			LibraryName: library.Name,
			Items:       mapped,
		})
	}
	return sections, nil
}

func (s *Service) loadCatalogItem(ctx context.Context, itemID uint) (database.CatalogItem, error) {
	var item database.CatalogItem
	err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", itemID).First(&item).Error
	return item, err
}

func (s *Service) IsGovernanceTargetAllowed(ctx context.Context, workspaceItemID uint, targetItemID uint) (bool, error) {
	if workspaceItemID == 0 || targetItemID == 0 {
		return false, nil
	}
	item, err := s.loadCatalogItem(ctx, targetItemID)
	if err != nil {
		return false, err
	}
	for {
		if item.ID == workspaceItemID {
			return true, nil
		}
		if item.ParentID == nil || *item.ParentID == 0 {
			return false, nil
		}
		item, err = s.loadCatalogItem(ctx, *item.ParentID)
		if err != nil {
			return false, err
		}
	}
}

func (s *Service) buildCatalogListItems(ctx context.Context, items []database.CatalogItem) ([]CatalogListItem, error) {
	if len(items) == 0 {
		return []CatalogListItem{}, nil
	}
	itemIDs := make([]uint, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}
	rollups, images, externalIDs, _, _, err := s.loadCatalogQueryData(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	result := make([]CatalogListItem, 0, len(items))
	for _, item := range items {
		result = append(result, BuildCatalogListItem(CatalogListItemInput{
			Item:        item,
			Rollup:      rollups[item.ID],
			Images:      images[item.ID],
			ExternalIDs: externalIDs[item.ID],
		}))
	}
	return result, nil
}

func (s *Service) buildCatalogEpisodeDetailsForParent(ctx context.Context, parentID uint) ([]CatalogEpisodeDetail, error) {
	byParent, err := s.buildCatalogEpisodeDetailsByParent(ctx, []uint{parentID})
	if err != nil {
		return nil, err
	}
	return byParent[parentID], nil
}

func (s *Service) buildCatalogEpisodeDetailsByParent(ctx context.Context, parentIDs []uint) (map[uint][]CatalogEpisodeDetail, error) {
	if len(parentIDs) == 0 {
		return map[uint][]CatalogEpisodeDetail{}, nil
	}
	var episodes []database.CatalogItem
	if err := s.db.WithContext(ctx).
		Where("parent_id IN ? AND type = ? AND deleted_at IS NULL", parentIDs, ItemTypeEpisode).
		Order("parent_id asc").Order("index_number asc").Order("id asc").
		Find(&episodes).Error; err != nil {
		return nil, err
	}
	if len(episodes) == 0 {
		return map[uint][]CatalogEpisodeDetail{}, nil
	}
	itemIDs := make([]uint, 0, len(episodes))
	for _, episode := range episodes {
		itemIDs = append(itemIDs, episode.ID)
	}
	_, images, externalIDs, sources, fieldStates, err := s.loadCatalogQueryData(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	assetsByItem, err := s.loadCatalogAssetsByItem(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[uint][]CatalogEpisodeDetail, len(parentIDs))
	for _, episode := range episodes {
		if episode.ParentID == nil {
			continue
		}
		result[*episode.ParentID] = append(result[*episode.ParentID], BuildCatalogEpisodeDetail(CatalogEpisodeDetailInput{
			Item:        episode,
			Images:      images[episode.ID],
			ExternalIDs: externalIDs[episode.ID],
			Sources:     sources[episode.ID],
			FieldStates: fieldStates[episode.ID],
			Assets:      assetsByItem[episode.ID],
		}))
	}
	return result, nil
}

func (s *Service) loadCatalogQueryData(ctx context.Context, itemIDs []uint) (map[uint]*database.ItemRollup, map[uint][]database.ItemImage, map[uint][]database.CatalogExternalID, map[uint][]database.MetadataSource, map[uint][]database.MetadataFieldState, error) {
	rollups := make(map[uint]*database.ItemRollup, len(itemIDs))
	images := make(map[uint][]database.ItemImage, len(itemIDs))
	externalIDs := make(map[uint][]database.CatalogExternalID, len(itemIDs))
	sources := make(map[uint][]database.MetadataSource, len(itemIDs))
	fieldStates := make(map[uint][]database.MetadataFieldState, len(itemIDs))
	if len(itemIDs) == 0 {
		return rollups, images, externalIDs, sources, fieldStates, nil
	}

	var rollupRows []database.ItemRollup
	if err := s.db.WithContext(ctx).Where("item_id IN ?", itemIDs).Find(&rollupRows).Error; err != nil {
		return nil, nil, nil, nil, nil, err
	}
	for _, row := range rollupRows {
		rowCopy := row
		rollups[row.ItemID] = &rowCopy
	}

	var imageRows []database.ItemImage
	if err := s.db.WithContext(ctx).Where("item_id IN ?", itemIDs).Order("item_id asc, sort_order asc, id asc").Find(&imageRows).Error; err != nil {
		return nil, nil, nil, nil, nil, err
	}
	for _, row := range imageRows {
		images[row.ItemID] = append(images[row.ItemID], row)
	}

	var externalIDRows []database.CatalogExternalID
	if err := s.db.WithContext(ctx).Where("item_id IN ?", itemIDs).Order("item_id asc, is_primary desc, provider asc, provider_type asc, id asc").Find(&externalIDRows).Error; err != nil {
		return nil, nil, nil, nil, nil, err
	}
	for _, row := range externalIDRows {
		externalIDs[row.ItemID] = append(externalIDs[row.ItemID], row)
	}

	var sourceRows []database.MetadataSource
	if err := s.db.WithContext(ctx).Where("item_id IN ?", itemIDs).Order("item_id asc, fetched_at desc, id desc").Find(&sourceRows).Error; err != nil {
		return nil, nil, nil, nil, nil, err
	}
	for _, row := range sourceRows {
		sources[row.ItemID] = append(sources[row.ItemID], row)
	}

	var fieldStateRows []database.MetadataFieldState
	if err := s.db.WithContext(ctx).Where("item_id IN ?", itemIDs).Order("item_id asc, field_key asc, id asc").Find(&fieldStateRows).Error; err != nil {
		return nil, nil, nil, nil, nil, err
	}
	for _, row := range fieldStateRows {
		fieldStates[row.ItemID] = append(fieldStates[row.ItemID], row)
	}

	return rollups, images, externalIDs, sources, fieldStates, nil
}

func (s *Service) loadCatalogAssetsByItem(ctx context.Context, itemIDs []uint) (map[uint][]CatalogAssetDetail, error) {
	result := make(map[uint][]CatalogAssetDetail, len(itemIDs))
	if len(itemIDs) == 0 {
		return result, nil
	}

	var links []database.AssetItem
	if err := s.db.WithContext(ctx).
		Where("item_id IN ?", itemIDs).
		Order("item_id asc, role asc, segment_index asc, id asc").
		Find(&links).Error; err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return result, nil
	}

	assetIDs := make([]uint, 0, len(links))
	assetIDSet := make(map[uint]struct{}, len(links))
	linksByItem := make(map[uint][]database.AssetItem, len(itemIDs))
	for _, link := range links {
		linksByItem[link.ItemID] = append(linksByItem[link.ItemID], link)
		if _, ok := assetIDSet[link.AssetID]; ok {
			continue
		}
		assetIDSet[link.AssetID] = struct{}{}
		assetIDs = append(assetIDs, link.AssetID)
	}

	var assetLinks []database.AssetItem
	if err := s.db.WithContext(ctx).
		Where("asset_id IN ?", assetIDs).
		Order("asset_id asc, role asc, segment_index asc, id asc").
		Find(&assetLinks).Error; err != nil {
		return nil, err
	}
	linksByAsset := make(map[uint][]database.AssetItem, len(assetLinks))
	for _, link := range assetLinks {
		linksByAsset[link.AssetID] = append(linksByAsset[link.AssetID], link)
	}

	var assets []database.MediaAsset
	if err := s.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", assetIDs).Order("id asc").Find(&assets).Error; err != nil {
		return nil, err
	}
	assetByID := make(map[uint]database.MediaAsset, len(assets))
	for _, asset := range assets {
		assetByID[asset.ID] = asset
	}

	var assetFileRows []database.AssetFile
	if err := s.db.WithContext(ctx).Where("asset_id IN ?", assetIDs).Order("asset_id asc, part_index asc, id asc").Find(&assetFileRows).Error; err != nil {
		return nil, err
	}
	fileIDsByAsset := make(map[uint][]uint, len(assetIDs))
	for _, row := range assetFileRows {
		fileIDsByAsset[row.AssetID] = append(fileIDsByAsset[row.AssetID], row.FileID)
	}

	for itemID, itemLinks := range linksByItem {
		assetDetails := make([]CatalogAssetDetail, 0, len(itemLinks))
		seenAssets := make(map[uint]struct{}, len(itemLinks))
		for _, link := range itemLinks {
			if _, ok := seenAssets[link.AssetID]; ok {
				continue
			}
			asset, ok := assetByID[link.AssetID]
			if !ok {
				continue
			}
			assetDetails = append(assetDetails, BuildCatalogAssetDetail(CatalogAssetDetailInput{Asset: asset, Links: linksByAsset[link.AssetID], FileIDs: fileIDsByAsset[link.AssetID]}))
			seenAssets[link.AssetID] = struct{}{}
		}
		sort.SliceStable(assetDetails, func(i, j int) bool {
			if assetDetails[i].Status != assetDetails[j].Status {
				return assetDetails[i].Status < assetDetails[j].Status
			}
			return assetDetails[i].ID < assetDetails[j].ID
		})
		result[itemID] = assetDetails
	}
	return result, nil
}
