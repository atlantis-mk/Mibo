package catalog

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ManualSeriesRestructureInput struct {
	LibraryID        uint
	RootPath         string
	SeriesTitle      string
	SeasonNumber     *int
	MigrateMetadata  bool
	EpisodeMappings  []ManualSeriesEpisodeMappingInput
	CreatedByUserID  *uint
}

type ManualSeriesEpisodeMappingInput struct {
	SourceItemID     uint
	AssetID          uint
	FileID           uint
	StoragePath      string
	SeasonNumber     *int
	EpisodeNumber    *int
	EpisodeTitle     string
	EpisodePath      string
	EpisodeNumberEnd *int
}

type ManualSeriesRestructureResult struct {
	Series   database.CatalogItem       `json:"series"`
	Seasons  []database.CatalogItem     `json:"seasons"`
	Episodes []database.CatalogItem     `json:"episodes"`
	Mappings []ManualSeriesEpisodeLink  `json:"mappings"`
	Warnings []string                   `json:"warnings,omitempty"`
}

type ManualSeriesEpisodeLink struct {
	SourceItemID  uint   `json:"source_item_id,omitempty"`
	AssetID       uint   `json:"asset_id"`
	FileID        uint   `json:"file_id,omitempty"`
	StoragePath   string `json:"storage_path"`
	EpisodeItemID uint   `json:"episode_item_id"`
	SeasonNumber  int    `json:"season_number"`
	EpisodeNumber int    `json:"episode_number"`
	EpisodeTitle  string `json:"episode_title"`
}

type manualSeriesSource struct {
	SourceItemID uint
	AssetID      uint
	FileID       uint
	StoragePath  string
}

func (s *Service) PreviewManualSeriesRestructure(ctx context.Context, input ManualSeriesRestructureInput) (ManualSeriesRestructureResult, error) {
	rootPath := cleanStoragePath(input.RootPath)
	if input.LibraryID == 0 || rootPath == "" {
		return ManualSeriesRestructureResult{}, errors.New("library_id and root_path are required")
	}
	sources, err := s.loadManualSeriesSources(ctx, s.db.WithContext(ctx), input.LibraryID, rootPath)
	if err != nil {
		return ManualSeriesRestructureResult{}, err
	}
	plan, err := buildManualSeriesPlan(input, sources)
	if err != nil {
		return ManualSeriesRestructureResult{}, err
	}
	return ManualSeriesRestructureResult{Mappings: plan.Mappings, Warnings: plan.Warnings}, nil
}

func (s *Service) ApplyManualSeriesRestructure(ctx context.Context, input ManualSeriesRestructureInput) (ManualSeriesRestructureResult, error) {
	rootPath := cleanStoragePath(input.RootPath)
	if input.LibraryID == 0 || rootPath == "" {
		return ManualSeriesRestructureResult{}, errors.New("library_id and root_path are required")
	}

	var result ManualSeriesRestructureResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		sources, err := s.loadManualSeriesSources(ctx, tx, input.LibraryID, rootPath)
		if err != nil {
			return err
		}
		plan, err := buildManualSeriesPlan(input, sources)
		if err != nil {
			return err
		}
		series, err := s.findOrCreateManualSeries(ctx, tx, input.LibraryID, rootPath, plan.SeriesTitle)
		if err != nil {
			return err
		}
		result.Series = series
		result.Mappings = plan.Mappings
		result.Warnings = plan.Warnings

		seasonsByNumber := make(map[int]database.CatalogItem)
		seasonIDs := make(map[uint]struct{})
		episodeIDs := make(map[uint]struct{})
		sourceItemIDs := make(map[uint]struct{})
		var firstSourceItemID uint
		for idx, mapping := range plan.Mappings {
			season, err := s.findOrCreateManualSeason(ctx, tx, series, mapping.SeasonNumber)
			if err != nil {
				return err
			}
			seasonsByNumber[mapping.SeasonNumber] = season
			seasonIDs[season.ID] = struct{}{}
			episode, err := s.findOrCreateManualEpisode(ctx, tx, series, season, mapping)
			if err != nil {
				return err
			}
			episodeIDs[episode.ID] = struct{}{}
			result.Mappings[idx].EpisodeItemID = episode.ID
			if err := relinkManualSeriesAsset(ctx, tx, mapping.AssetID, episode.ID); err != nil {
				return err
			}
			if mapping.SourceItemID != 0 && mapping.SourceItemID != episode.ID {
				if firstSourceItemID == 0 {
					firstSourceItemID = mapping.SourceItemID
				}
				sourceItemIDs[mapping.SourceItemID] = struct{}{}
				if input.MigrateMetadata {
					if err := copyManualRestructureMetadata(ctx, tx, mapping.SourceItemID, episode.ID, true); err != nil {
						return err
					}
				}
			}
		}
		if input.MigrateMetadata && firstSourceItemID != 0 {
			if err := copyManualRestructureMetadata(ctx, tx, firstSourceItemID, series.ID, false); err != nil {
				return err
			}
		}
		if input.MigrateMetadata {
			if err := ensureManualSeriesImagesFromFirstEpisode(ctx, tx, series.ID); err != nil {
				return err
			}
		}

		if len(sourceItemIDs) > 0 {
			now := time.Now().UTC()
			ids := uintSetToSortedSlice(sourceItemIDs)
			if err := tx.Model(&database.CatalogItem{}).Where("id IN ? AND type = ?", ids, ItemTypeMovie).Updates(map[string]any{"deleted_at": &now, "governance_status": GovernanceManual, "last_canonicalized_at": now}).Error; err != nil {
				return err
			}
		}

		for _, seasonNumber := range sortedIntKeys(seasonsByNumber) {
			result.Seasons = append(result.Seasons, seasonsByNumber[seasonNumber])
		}
		if err := tx.Where("id IN ?", uintSetToSortedSlice(episodeIDs)).Order("parent_index_number asc, index_number asc, id asc").Find(&result.Episodes).Error; err != nil {
			return err
		}

		refreshIDs := append([]uint{series.ID}, uintSetToSortedSlice(seasonIDs)...)
		refreshIDs = append(refreshIDs, uintSetToSortedSlice(episodeIDs)...)
		refreshIDs = append(refreshIDs, uintSetToSortedSlice(sourceItemIDs)...)
		for _, itemID := range refreshIDs {
			if err := s.refreshProjectionWithDB(ctx, tx, ProjectionRefreshRequest{ItemID: itemID}); err != nil {
				return err
			}
		}
		return nil
	})
	return result, err
}

func (s *Service) loadManualSeriesSources(ctx context.Context, db *gorm.DB, libraryID uint, rootPath string) ([]manualSeriesSource, error) {
	likePath := strings.TrimRight(rootPath, "/") + "/%"
	var rows []struct {
		SourceItemID uint
		AssetID      uint
		FileID       uint
		StoragePath  string
	}
	err := db.WithContext(ctx).
		Table("inventory_files").
		Select("catalog_items.id AS source_item_id, media_assets.id AS asset_id, inventory_files.id AS file_id, inventory_files.storage_path AS storage_path").
		Joins("JOIN asset_files ON asset_files.file_id = inventory_files.id").
		Joins("JOIN media_assets ON media_assets.id = asset_files.asset_id AND media_assets.deleted_at IS NULL").
		Joins("LEFT JOIN asset_items ON asset_items.asset_id = media_assets.id AND asset_items.role = ?", inventory.AssetItemRolePrimary).
		Joins("LEFT JOIN catalog_items ON catalog_items.id = asset_items.item_id AND catalog_items.deleted_at IS NULL").
		Where("inventory_files.library_id = ? AND inventory_files.deleted_at IS NULL AND inventory_files.content_class = ?", libraryID, "video").
		Where("inventory_files.storage_path = ? OR inventory_files.storage_path LIKE ?", rootPath, likePath).
		Order("inventory_files.storage_path asc, media_assets.id asc").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	seen := make(map[uint]struct{}, len(rows))
	sources := make([]manualSeriesSource, 0, len(rows))
	for _, row := range rows {
		if row.AssetID == 0 || row.FileID == 0 || strings.TrimSpace(row.StoragePath) == "" {
			continue
		}
		if _, ok := seen[row.AssetID]; ok {
			continue
		}
		seen[row.AssetID] = struct{}{}
		sources = append(sources, manualSeriesSource{SourceItemID: row.SourceItemID, AssetID: row.AssetID, FileID: row.FileID, StoragePath: row.StoragePath})
	}
	if len(sources) == 0 {
		return nil, errors.New("no scanned video assets found under root_path")
	}
	return sources, nil
}

type manualSeriesPlan struct {
	SeriesTitle string
	Mappings    []ManualSeriesEpisodeLink
	Warnings    []string
}

func buildManualSeriesPlan(input ManualSeriesRestructureInput, sources []manualSeriesSource) (manualSeriesPlan, error) {
	rootPath := cleanStoragePath(input.RootPath)
	seriesTitle := strings.TrimSpace(input.SeriesTitle)
	if seriesTitle == "" {
		seriesTitle = path.Base(rootPath)
	}
	mappingByAsset := make(map[uint]ManualSeriesEpisodeMappingInput, len(input.EpisodeMappings))
	mappingByPath := make(map[string]ManualSeriesEpisodeMappingInput, len(input.EpisodeMappings))
	for _, mapping := range input.EpisodeMappings {
		if mapping.AssetID != 0 {
			mappingByAsset[mapping.AssetID] = mapping
		}
		if trimmed := cleanStoragePath(mapping.StoragePath); trimmed != "" {
			mappingByPath[trimmed] = mapping
		}
	}
	defaultSeason := 1
	if input.SeasonNumber != nil && *input.SeasonNumber > 0 {
		defaultSeason = *input.SeasonNumber
	}

	plan := manualSeriesPlan{SeriesTitle: seriesTitle}
	usedSlots := make(map[string]string, len(sources))
	for idx, source := range sources {
		override := mappingByAsset[source.AssetID]
		if override.AssetID == 0 {
			override = mappingByPath[cleanStoragePath(source.StoragePath)]
		}
		seasonNumber := firstPositiveInt(override.SeasonNumber, inferSeasonNumber(rootPath, source.StoragePath), &defaultSeason)
		episodeNumber := firstPositiveInt(override.EpisodeNumber, inferEpisodeNumber(rootPath, source.StoragePath), intPointer(idx+1))
		episodeTitle := strings.TrimSpace(override.EpisodeTitle)
		if episodeTitle == "" {
			episodeTitle = episodeTitleFromStoragePath(source.StoragePath)
		}
		if episodeTitle == "" {
			episodeTitle = fmt.Sprintf("Episode %d", episodeNumber)
		}
		storagePath := cleanStoragePath(source.StoragePath)
		if trimmed := cleanStoragePath(override.StoragePath); trimmed != "" {
			storagePath = trimmed
		}
		slotKey := fmt.Sprintf("%d:%d", seasonNumber, episodeNumber)
		if previous, ok := usedSlots[slotKey]; ok {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("episode slot S%02dE%02d has multiple assets: %s and %s", seasonNumber, episodeNumber, previous, storagePath))
		}
		usedSlots[slotKey] = storagePath
		plan.Mappings = append(plan.Mappings, ManualSeriesEpisodeLink{SourceItemID: source.SourceItemID, AssetID: source.AssetID, FileID: source.FileID, StoragePath: storagePath, SeasonNumber: seasonNumber, EpisodeNumber: episodeNumber, EpisodeTitle: episodeTitle})
	}
	return plan, nil
}

func (s *Service) findOrCreateManualSeries(ctx context.Context, tx *gorm.DB, libraryID uint, rootPath string, title string) (database.CatalogItem, error) {
	var item database.CatalogItem
	err := tx.Where("library_id = ? AND type = ? AND path = ? AND deleted_at IS NULL", libraryID, ItemTypeSeries, rootPath).First(&item).Error
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return database.CatalogItem{}, err
	}
	now := time.Now().UTC()
	item = database.CatalogItem{LibraryID: libraryID, Type: ItemTypeSeries, Path: rootPath, SortKey: title, DisplayOrder: DisplayOrderAired, Title: title, AvailabilityStatus: AvailabilityAvailable, GovernanceStatus: GovernanceManual, CanonicalVersion: 1, LastCanonicalizedAt: &now}
	if err := tx.Create(&item).Error; err != nil {
		return database.CatalogItem{}, err
	}
	if err := tx.Model(&item).Update("root_id", item.ID).Error; err != nil {
		return database.CatalogItem{}, err
	}
	item.RootID = &item.ID
	return item, nil
}

func (s *Service) findOrCreateManualSeason(ctx context.Context, tx *gorm.DB, series database.CatalogItem, seasonNumber int) (database.CatalogItem, error) {
	var item database.CatalogItem
	err := tx.Where("parent_id = ? AND type = ? AND index_number = ? AND deleted_at IS NULL", series.ID, ItemTypeSeason, seasonNumber).First(&item).Error
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return database.CatalogItem{}, err
	}
	now := time.Now().UTC()
	seasonTitle := fmt.Sprintf("Season %d", seasonNumber)
	item = database.CatalogItem{LibraryID: series.LibraryID, Type: ItemTypeSeason, ParentID: &series.ID, RootID: &series.ID, Path: strings.TrimRight(series.Path, "/") + fmt.Sprintf("/season-%02d", seasonNumber), SortKey: fmt.Sprintf("%s S%02d", series.Title, seasonNumber), DisplayOrder: DisplayOrderAired, IndexNumber: &seasonNumber, ParentIndexNumber: &seasonNumber, Title: seasonTitle, AvailabilityStatus: AvailabilityAvailable, GovernanceStatus: GovernanceManual, CanonicalVersion: 1, LastCanonicalizedAt: &now}
	if err := tx.Create(&item).Error; err != nil {
		return database.CatalogItem{}, err
	}
	return item, nil
}

func (s *Service) findOrCreateManualEpisode(ctx context.Context, tx *gorm.DB, series database.CatalogItem, season database.CatalogItem, mapping ManualSeriesEpisodeLink) (database.CatalogItem, error) {
	var item database.CatalogItem
	err := tx.Where("parent_id = ? AND type = ? AND index_number = ? AND deleted_at IS NULL", season.ID, ItemTypeEpisode, mapping.EpisodeNumber).First(&item).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return database.CatalogItem{}, err
	}
	now := time.Now().UTC()
	episodePath := strings.TrimRight(season.Path, "/") + fmt.Sprintf("/episode-%04d", mapping.EpisodeNumber)
	updates := map[string]any{"library_id": series.LibraryID, "type": ItemTypeEpisode, "parent_id": season.ID, "root_id": series.ID, "path": episodePath, "sort_key": fmt.Sprintf("%s S%02dE%02d", series.Title, mapping.SeasonNumber, mapping.EpisodeNumber), "display_order": DisplayOrderAired, "parent_index_number": mapping.SeasonNumber, "index_number": mapping.EpisodeNumber, "title": mapping.EpisodeTitle, "availability_status": AvailabilityAvailable, "governance_status": GovernanceManual, "canonical_version": 1, "last_canonicalized_at": now, "deleted_at": nil}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		item = database.CatalogItem{LibraryID: series.LibraryID, Type: ItemTypeEpisode, ParentID: &season.ID, RootID: &series.ID, Path: episodePath, SortKey: fmt.Sprintf("%s S%02dE%02d", series.Title, mapping.SeasonNumber, mapping.EpisodeNumber), DisplayOrder: DisplayOrderAired, ParentIndexNumber: &mapping.SeasonNumber, IndexNumber: &mapping.EpisodeNumber, Title: mapping.EpisodeTitle, AvailabilityStatus: AvailabilityAvailable, GovernanceStatus: GovernanceManual, CanonicalVersion: 1, LastCanonicalizedAt: &now}
		if err := tx.Create(&item).Error; err != nil {
			return database.CatalogItem{}, err
		}
		return item, nil
	}
	if err := tx.Model(&database.CatalogItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
		return database.CatalogItem{}, err
	}
	if err := tx.First(&item, item.ID).Error; err != nil {
		return database.CatalogItem{}, err
	}
	return item, nil
}

func relinkManualSeriesAsset(ctx context.Context, tx *gorm.DB, assetID uint, episodeID uint) error {
	if assetID == 0 || episodeID == 0 {
		return errors.New("asset id and episode id are required")
	}
	if err := tx.WithContext(ctx).Where("asset_id = ? AND role = ?", assetID, inventory.AssetItemRolePrimary).Delete(&database.AssetItem{}).Error; err != nil {
		return err
	}
	link := database.AssetItem{AssetID: assetID, ItemID: episodeID, Role: inventory.AssetItemRolePrimary, Source: "manual_restructure"}
	return tx.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "asset_id"}, {Name: "item_id"}, {Name: "role"}, {Name: "segment_index"}}, DoUpdates: clause.AssignmentColumns([]string{"source", "updated_at"})}).Create(&link).Error
}

func copyManualRestructureMetadata(ctx context.Context, tx *gorm.DB, sourceItemID uint, targetItemID uint, includeProgress bool) error {
	if sourceItemID == 0 || targetItemID == 0 || sourceItemID == targetItemID {
		return nil
	}
	if err := copyManualRestructureImages(ctx, tx, sourceItemID, targetItemID, includeProgress); err != nil {
		return err
	}
	if err := copyManualRestructureFieldStates(ctx, tx, sourceItemID, targetItemID); err != nil {
		return err
	}
	if err := copyManualRestructureTags(ctx, tx, sourceItemID, targetItemID); err != nil {
		return err
	}
	if err := copyManualRestructurePeople(ctx, tx, sourceItemID, targetItemID); err != nil {
		return err
	}
	if includeProgress {
		if err := migrateManualRestructureUserData(ctx, tx, sourceItemID, targetItemID); err != nil {
			return err
		}
	}
	return nil
}

func copyManualRestructureImages(ctx context.Context, tx *gorm.DB, sourceItemID uint, targetItemID uint, episodeTarget bool) error {
	var rows []database.ItemImage
	if err := tx.WithContext(ctx).Where("item_id = ?", sourceItemID).Order("sort_order asc, id asc").Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		imageType := manualRestructureTargetImageType(row.ImageType, episodeTarget)
		if imageType == "" {
			continue
		}
		shouldSelect, err := manualRestructureImageShouldSelect(ctx, tx, targetItemID, imageType, row.IsSelected)
		if err != nil {
			return err
		}
		var existing int64
		if err := tx.WithContext(ctx).Model(&database.ItemImage{}).Where("item_id = ? AND image_type = ? AND url = ?", targetItemID, imageType, row.URL).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			if shouldSelect {
				if err := selectManualRestructureImage(ctx, tx, targetItemID, imageType, row.URL); err != nil {
					return err
				}
			}
			continue
		}
		copy := database.ItemImage{ItemID: targetItemID, ImageType: imageType, URL: row.URL, SourceID: row.SourceID, Language: row.Language, Width: row.Width, Height: row.Height, IsSelected: shouldSelect, SortOrder: row.SortOrder}
		if err := tx.WithContext(ctx).Create(&copy).Error; err != nil {
			return err
		}
	}
	return nil
}

func manualRestructureTargetImageType(sourceImageType string, episodeTarget bool) string {
	switch strings.TrimSpace(sourceImageType) {
	case "poster":
		if episodeTarget {
			return "still"
		}
		return "poster"
	case "backdrop":
		return "backdrop"
	case "still":
		return "still"
	default:
		if episodeTarget {
			return ""
		}
		return strings.TrimSpace(sourceImageType)
	}
}

func ensureManualSeriesImagesFromFirstEpisode(ctx context.Context, tx *gorm.DB, seriesID uint) error {
	var episode database.CatalogItem
	err := tx.WithContext(ctx).
		Where("root_id = ? AND type = ? AND deleted_at IS NULL", seriesID, ItemTypeEpisode).
		Order("parent_index_number asc, index_number asc, id asc").
		First(&episode).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	var images []database.ItemImage
	if err := tx.WithContext(ctx).Where("item_id = ? AND is_selected = ?", episode.ID, true).Order("sort_order asc, id asc").Find(&images).Error; err != nil {
		return err
	}
	for _, image := range images {
		imageType := manualRestructureSeriesImageTypeFromEpisode(image.ImageType)
		if imageType == "" {
			continue
		}
		shouldSelect, err := manualRestructureImageShouldSelect(ctx, tx, seriesID, imageType, true)
		if err != nil {
			return err
		}
		if !shouldSelect {
			continue
		}
		copy := database.ItemImage{ItemID: seriesID, ImageType: imageType, URL: image.URL, SourceID: image.SourceID, Language: image.Language, Width: image.Width, Height: image.Height, IsSelected: true, SortOrder: image.SortOrder}
		if err := tx.WithContext(ctx).Create(&copy).Error; err != nil {
			return err
		}
	}
	return nil
}

func manualRestructureSeriesImageTypeFromEpisode(sourceImageType string) string {
	switch strings.TrimSpace(sourceImageType) {
	case "still", "poster":
		return "poster"
	case "backdrop":
		return "backdrop"
	default:
		return ""
	}
}

func manualRestructureImageShouldSelect(ctx context.Context, tx *gorm.DB, itemID uint, imageType string, sourceSelected bool) (bool, error) {
	var selectedCount int64
	if err := tx.WithContext(ctx).Model(&database.ItemImage{}).Where("item_id = ? AND image_type = ? AND is_selected = ?", itemID, imageType, true).Count(&selectedCount).Error; err != nil {
		return false, err
	}
	_ = sourceSelected
	return selectedCount == 0, nil
}

func selectManualRestructureImage(ctx context.Context, tx *gorm.DB, itemID uint, imageType string, url string) error {
	if err := tx.WithContext(ctx).Model(&database.ItemImage{}).Where("item_id = ? AND image_type = ?", itemID, imageType).Update("is_selected", false).Error; err != nil {
		return err
	}
	return tx.WithContext(ctx).Model(&database.ItemImage{}).Where("item_id = ? AND image_type = ? AND url = ?", itemID, imageType, url).Update("is_selected", true).Error
}

func copyManualRestructureFieldStates(ctx context.Context, tx *gorm.DB, sourceItemID uint, targetItemID uint) error {
	var rows []database.MetadataFieldState
	if err := tx.WithContext(ctx).Where("item_id = ?", sourceItemID).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if row.FieldKey == "governance_status" {
			continue
		}
		copy := database.MetadataFieldState{ItemID: targetItemID, FieldKey: row.FieldKey, SourceID: row.SourceID, ValueJSON: row.ValueJSON, IsLocked: row.IsLocked, LockReason: row.LockReason, EditedByUserID: row.EditedByUserID, EditedAt: row.EditedAt}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "item_id"}, {Name: "field_key"}}, DoNothing: true}).Create(&copy).Error; err != nil {
			return err
		}
	}
	return nil
}

func copyManualRestructureTags(ctx context.Context, tx *gorm.DB, sourceItemID uint, targetItemID uint) error {
	var rows []database.ItemTag
	if err := tx.WithContext(ctx).Where("item_id = ?", sourceItemID).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		copy := database.ItemTag{ItemID: targetItemID, TagID: row.TagID, SourceID: row.SourceID}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "item_id"}, {Name: "tag_id"}}, DoNothing: true}).Create(&copy).Error; err != nil {
			return err
		}
	}
	return nil
}

func copyManualRestructurePeople(ctx context.Context, tx *gorm.DB, sourceItemID uint, targetItemID uint) error {
	var rows []database.ItemPerson
	if err := tx.WithContext(ctx).Where("item_id = ?", sourceItemID).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		copy := database.ItemPerson{ItemID: targetItemID, PersonID: row.PersonID, Role: row.Role, Character: row.Character, SortOrder: row.SortOrder, SourceID: row.SourceID}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "item_id"}, {Name: "person_id"}, {Name: "role"}}, DoNothing: true}).Create(&copy).Error; err != nil {
			return err
		}
	}
	return nil
}

func migrateManualRestructureUserData(ctx context.Context, tx *gorm.DB, sourceItemID uint, targetItemID uint) error {
	return tx.WithContext(ctx).Model(&database.UserItemData{}).Where("item_id = ?", sourceItemID).Update("item_id", targetItemID).Error
}

func inferSeasonNumber(rootPath string, storagePath string) *int {
	segments := relativePathSegments(rootPath, storagePath)
	if len(segments) >= 2 {
		if value := parseLeadingPositiveInt(segments[0]); value != nil {
			return value
		}
	}
	return nil
}

func inferEpisodeNumber(rootPath string, storagePath string) *int {
	segments := relativePathSegments(rootPath, storagePath)
	for idx := len(segments) - 1; idx >= 0; idx-- {
		name := segments[idx]
		if idx == len(segments)-1 {
			name = strings.TrimSuffix(name, path.Ext(name))
		}
		if value := parseLeadingPositiveInt(name); value != nil {
			return value
		}
	}
	return nil
}

func episodeTitleFromStoragePath(storagePath string) string {
	base := path.Base(storagePath)
	return strings.TrimSpace(strings.TrimSuffix(base, path.Ext(base)))
}

func parseLeadingPositiveInt(input string) *int {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil
	}
	end := 0
	for end < len(trimmed) && trimmed[end] >= '0' && trimmed[end] <= '9' {
		end++
	}
	if end == 0 {
		return nil
	}
	value, err := strconv.Atoi(trimmed[:end])
	if err != nil || value <= 0 {
		return nil
	}
	return &value
}

func firstPositiveInt(values ...*int) int {
	for _, value := range values {
		if value != nil && *value > 0 {
			return *value
		}
	}
	return 1
}

func cleanStoragePath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return path.Clean(trimmed)
}

func relativePathSegments(rootPath string, storagePath string) []string {
	root := strings.TrimRight(cleanStoragePath(rootPath), "/")
	storage := cleanStoragePath(storagePath)
	if root == "" || storage == "" || storage == root {
		return nil
	}
	if !strings.HasPrefix(storage, root+"/") {
		return nil
	}
	return strings.Split(strings.TrimPrefix(storage, root+"/"), "/")
}

func intPointer(value int) *int {
	return &value
}

func uintSetToSortedSlice(values map[uint]struct{}) []uint {
	items := make([]uint, 0, len(values))
	for value := range values {
		items = append(items, value)
	}
	sort.Slice(items, func(i, j int) bool { return items[i] < items[j] })
	return items
}

func sortedIntKeys[T any](values map[int]T) []int {
	keys := make([]int, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	return keys
}
