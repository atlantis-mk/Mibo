package library

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"gorm.io/gorm"
)

type catalogScanWriteResult struct {
	Item  database.CatalogItem
	File  database.InventoryFile
	Asset database.MediaAsset
}

func (s *Service) writeCatalogScan(ctx context.Context, library database.Library, artifact catalogScanArtifact) (catalogScanWriteResult, error) {
	if strings.TrimSpace(artifact.ItemType) == catalog.ItemTypeEpisode || len(artifact.EpisodeSlots) > 0 {
		return s.writeCatalogScanEpisodeHierarchy(ctx, library, artifact)
	}
	return s.writeCatalogScanMovie(ctx, library, artifact)
}

func (s *Service) writeCatalogScanMovie(ctx context.Context, library database.Library, artifact catalogScanArtifact) (catalogScanWriteResult, error) {
	if strings.TrimSpace(artifact.ItemPath) == "" {
		artifact.ItemPath = artifact.SourcePath
	}
	if strings.TrimSpace(artifact.ItemType) == "" {
		artifact.ItemType = catalog.ItemTypeMovie
	}

	var result catalogScanWriteResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		catalogSvc := catalog.NewService(tx)
		inventorySvc := inventory.NewService(tx)

		item, err := createOrReuseCatalogItem(ctx, tx, catalogSvc, catalog.CreateItemInput{
			LibraryID:          library.ID,
			Type:               artifact.ItemType,
			Path:               artifact.ItemPath,
			SortKey:            defaultCatalogSortKey(artifact.Title, artifact.ItemPath),
			Title:              defaultCatalogTitle(artifact.Title, artifact.SourcePath),
			OriginalTitle:      strings.TrimSpace(artifact.OriginalTitle),
			Year:               artifact.Year,
			AvailabilityStatus: catalog.AvailabilityAvailable,
			GovernanceStatus:   catalog.GovernancePending,
		})
		if err != nil {
			return err
		}

		file, err := upsertCatalogScanFile(ctx, tx, inventorySvc, library.ID, artifact)
		if err != nil {
			return err
		}
		asset, err := createOrReuseCatalogScanAsset(ctx, tx, inventorySvc, library.ID, file.ID, artifact, inventory.AssetTypeMain)
		if err != nil {
			return err
		}
		if _, err := inventorySvc.LinkAssetToItem(ctx, inventory.LinkAssetItemInput{AssetID: asset.ID, ItemID: item.ID, Role: inventory.AssetItemRolePrimary, SegmentIndex: 0, Source: "scanner"}); err != nil {
			return err
		}

		if _, err := catalogSvc.RecordMetadataSource(ctx, catalog.MetadataSourceInput{
			ItemID:      item.ID,
			SourceType:  catalog.SourceTypeLocalFile,
			SourceName:  "scanner",
			PayloadJSON: buildCatalogScanEvidencePayload(artifact, nil),
			FetchedAt:   time.Now().UTC(),
		}); err != nil {
			return err
		}

		result = catalogScanWriteResult{Item: item, File: file, Asset: asset}
		return nil
	})
	if err != nil {
		return catalogScanWriteResult{}, err
	}
	return result, nil
}

func (s *Service) writeCatalogScanEpisodeHierarchy(ctx context.Context, library database.Library, artifact catalogScanArtifact) (catalogScanWriteResult, error) {
	if strings.TrimSpace(artifact.ItemType) == "" {
		artifact.ItemType = catalog.ItemTypeEpisode
	}
	if strings.TrimSpace(artifact.SeriesPath) == "" {
		return catalogScanWriteResult{}, errors.New("series path is required")
	}
	if artifact.SeasonNumber == nil {
		return catalogScanWriteResult{}, errors.New("season number is required")
	}
	if len(artifact.EpisodeSlots) == 0 {
		return catalogScanWriteResult{}, errors.New("at least one episode slot is required")
	}

	slots := append([]catalogEpisodeSlot(nil), artifact.EpisodeSlots...)
	sort.Slice(slots, func(i, j int) bool { return slots[i].EpisodeNumber < slots[j].EpisodeNumber })

	var result catalogScanWriteResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		catalogSvc := catalog.NewService(tx)
		inventorySvc := inventory.NewService(tx)

		seriesItem, err := createOrReuseCatalogItem(ctx, tx, catalogSvc, catalog.CreateItemInput{
			LibraryID:          library.ID,
			Type:               catalog.ItemTypeSeries,
			Path:               artifact.SeriesPath,
			SortKey:            defaultCatalogSortKey(artifact.SeriesTitle, artifact.SeriesPath),
			Title:              defaultCatalogTitle(artifact.SeriesTitle, artifact.SeriesPath),
			AvailabilityStatus: catalog.AvailabilityAvailable,
			GovernanceStatus:   catalog.GovernancePending,
		})
		if err != nil {
			return err
		}

		seasonTitle := fmt.Sprintf("Season %d", *artifact.SeasonNumber)
		seasonItem, err := createOrReuseCatalogItem(ctx, tx, catalogSvc, catalog.CreateItemInput{
			LibraryID:          library.ID,
			Type:               catalog.ItemTypeSeason,
			ParentID:           &seriesItem.ID,
			Path:               artifact.SeasonPath,
			SortKey:            fmt.Sprintf("%s S%02d", defaultCatalogTitle(artifact.SeriesTitle, artifact.SeriesPath), *artifact.SeasonNumber),
			Title:              seasonTitle,
			IndexNumber:        artifact.SeasonNumber,
			AvailabilityStatus: catalog.AvailabilityAvailable,
			GovernanceStatus:   catalog.GovernancePending,
		})
		if err != nil {
			return err
		}

		episodeItems := make([]database.CatalogItem, 0, len(slots))
		episodeNumbers := make([]int, 0, len(slots))
		for _, slot := range slots {
			episodeNumber := slot.EpisodeNumber
			episodeTitle := artifact.Title
			if strings.TrimSpace(episodeTitle) == "" {
				episodeTitle = fmt.Sprintf("%s S%02dE%02d", defaultCatalogTitle(artifact.SeriesTitle, artifact.SeriesPath), *artifact.SeasonNumber, episodeNumber)
			}
			episodeItem, err := createOrReuseCatalogItem(ctx, tx, catalogSvc, catalog.CreateItemInput{
				LibraryID:          library.ID,
				Type:               catalog.ItemTypeEpisode,
				ParentID:           &seasonItem.ID,
				Path:               slot.ItemPath,
				SortKey:            fmt.Sprintf("%s S%02dE%02d", defaultCatalogTitle(artifact.SeriesTitle, artifact.SeriesPath), *artifact.SeasonNumber, episodeNumber),
				Title:              episodeTitle,
				OriginalTitle:      strings.TrimSpace(artifact.OriginalTitle),
				Year:               artifact.Year,
				IndexNumber:        &episodeNumber,
				ParentIndexNumber:  artifact.SeasonNumber,
				AvailabilityStatus: catalog.AvailabilityAvailable,
				GovernanceStatus:   catalog.GovernancePending,
			})
			if err != nil {
				return err
			}
			episodeItems = append(episodeItems, episodeItem)
			if _, err := catalogSvc.RecordMetadataSource(ctx, catalog.MetadataSourceInput{
				ItemID:      episodeItem.ID,
				SourceType:  catalog.SourceTypeLocalFile,
				SourceName:  "scanner",
				PayloadJSON: buildCatalogScanEvidencePayload(artifact, episodeNumbersWithAppend(episodeNumbers, episodeNumber)),
				FetchedAt:   time.Now().UTC(),
			}); err != nil {
				return err
			}
			result.Item = episodeItem
			episodeNumbers = append(episodeNumbers, episodeNumber)
		}

		file, err := upsertCatalogScanFile(ctx, tx, inventorySvc, library.ID, artifact)
		if err != nil {
			return err
		}

		assetType, role := catalogScanAssetDisposition(ctx, tx, file.ID, episodeItems)
		asset, err := createOrReuseCatalogScanAsset(ctx, tx, inventorySvc, library.ID, file.ID, artifact, assetType)
		if err != nil {
			return err
		}
		for idx, episodeItem := range episodeItems {
			segmentIndex := 0
			currentRole := role
			if len(episodeItems) > 1 {
				currentRole = inventory.AssetItemRoleMultiEpisodePart
				segmentIndex = idx + 1
			}
			if _, err := inventorySvc.LinkAssetToItem(ctx, inventory.LinkAssetItemInput{
				AssetID:      asset.ID,
				ItemID:       episodeItem.ID,
				Role:         currentRole,
				SegmentIndex: segmentIndex,
				Source:       "scanner",
			}); err != nil {
				return err
			}
		}

		result.File = file
		result.Asset = asset
		return nil
	})
	if err != nil {
		return catalogScanWriteResult{}, err
	}
	return result, nil
}

func upsertCatalogScanFile(ctx context.Context, tx *gorm.DB, inventorySvc *inventory.Service, libraryID uint, artifact catalogScanArtifact) (database.InventoryFile, error) {
	storagePath := strings.TrimSpace(artifact.SourcePath)
	if storagePath == "" {
		return database.InventoryFile{}, errors.New("source path is required")
	}
	storageProvider := strings.TrimSpace(artifact.StorageProvider)
	if storageProvider == "" {
		storageProvider = "local"
	}
	container := strings.TrimSpace(artifact.Container)
	if container == "" {
		container = strings.TrimPrefix(strings.ToLower(path.Ext(storagePath)), ".")
	}
	if reused, reusedExisting, err := reuseInventoryFileByStableIdentity(ctx, tx, libraryID, storageProvider, storagePath, container, artifact); err != nil {
		return database.InventoryFile{}, err
	} else if reusedExisting {
		return reused, nil
	}
	return inventorySvc.UpsertFile(ctx, inventory.UpsertFileInput{
		LibraryID:         libraryID,
		StorageProvider:   storageProvider,
		StoragePath:       storagePath,
		StableIdentityKey: strings.TrimSpace(artifact.StableIdentityKey),
		HashesJSON:        strings.TrimSpace(artifact.HashesJSON),
		SizeBytes:         artifact.SizeBytes,
		ModifiedAt:        artifact.ModifiedAt,
		Container:         container,
		Status:            inventory.FileStatusAvailable,
	})
}

func reuseInventoryFileByStableIdentity(ctx context.Context, db *gorm.DB, libraryID uint, storageProvider string, storagePath string, container string, artifact catalogScanArtifact) (database.InventoryFile, bool, error) {
	stableIdentityKey := strings.TrimSpace(artifact.StableIdentityKey)
	if stableIdentityKey == "" {
		return database.InventoryFile{}, false, nil
	}
	var file database.InventoryFile
	err := db.WithContext(ctx).
		Where("library_id = ? AND storage_provider = ? AND stable_identity_key = ? AND deleted_at IS NULL", libraryID, storageProvider, stableIdentityKey).
		Order("id asc").
		First(&file).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.InventoryFile{}, false, nil
	}
	if err != nil {
		return database.InventoryFile{}, false, err
	}
	updates := map[string]any{
		"storage_path": storagePath,
		"hashes_json":  strings.TrimSpace(artifact.HashesJSON),
		"size_bytes":   artifact.SizeBytes,
		"modified_at":  artifact.ModifiedAt,
		"container":    container,
		"status":       inventory.FileStatusAvailable,
		"deleted_at":   nil,
	}
	if err := db.WithContext(ctx).Model(&database.InventoryFile{}).Where("id = ?", file.ID).Updates(updates).Error; err != nil {
		return database.InventoryFile{}, false, err
	}
	if err := db.WithContext(ctx).First(&file, file.ID).Error; err != nil {
		return database.InventoryFile{}, false, err
	}
	return file, true, nil
}

func createOrReuseCatalogScanAsset(ctx context.Context, tx *gorm.DB, inventorySvc *inventory.Service, libraryID uint, fileID uint, artifact catalogScanArtifact, assetType string) (database.MediaAsset, error) {
	asset, err := findCatalogScanAssetForFile(ctx, tx, fileID)
	if err == nil {
		updates := map[string]any{
			"asset_type":   strings.TrimSpace(assetType),
			"display_name": defaultCatalogTitle(artifact.Title, artifact.SourcePath),
			"status":       inventory.AssetStatusAvailable,
			"deleted_at":   nil,
		}
		if err := tx.WithContext(ctx).Model(&database.MediaAsset{}).Where("id = ?", asset.ID).Updates(updates).Error; err != nil {
			return database.MediaAsset{}, err
		}
		if err := tx.WithContext(ctx).First(&asset, asset.ID).Error; err != nil {
			return database.MediaAsset{}, err
		}
		return asset, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return database.MediaAsset{}, err
	}

	asset, err = inventorySvc.CreateAsset(ctx, inventory.CreateAssetInput{
		LibraryID:   libraryID,
		AssetType:   assetType,
		DisplayName: defaultCatalogTitle(artifact.Title, artifact.SourcePath),
		Status:      inventory.AssetStatusAvailable,
	})
	if err != nil {
		return database.MediaAsset{}, err
	}
	if _, err := inventorySvc.LinkAssetToFile(ctx, inventory.LinkAssetFileInput{AssetID: asset.ID, FileID: fileID, Role: inventory.FileRoleSource, PartIndex: 0}); err != nil {
		return database.MediaAsset{}, err
	}
	return asset, nil
}

func findCatalogScanAssetForFile(ctx context.Context, tx *gorm.DB, fileID uint) (database.MediaAsset, error) {
	var asset database.MediaAsset
	err := tx.WithContext(ctx).
		Joins("JOIN asset_files ON asset_files.asset_id = media_assets.id").
		Where("asset_files.file_id = ? AND asset_files.role = ? AND asset_files.part_index = 0", fileID, inventory.FileRoleSource).
		Order("media_assets.id asc").
		First(&asset).Error
	return asset, err
}

func catalogScanAssetDisposition(ctx context.Context, tx *gorm.DB, fileID uint, episodeItems []database.CatalogItem) (string, string) {
	if len(episodeItems) > 1 {
		return inventory.AssetTypeMain, inventory.AssetItemRoleMultiEpisodePart
	}
	if len(episodeItems) == 1 && hasOtherCatalogAssetForEpisode(ctx, tx, episodeItems[0].ID, fileID) {
		return inventory.AssetTypeVersion, inventory.AssetItemRoleVersion
	}
	return inventory.AssetTypeMain, inventory.AssetItemRolePrimary
}

func hasOtherCatalogAssetForEpisode(ctx context.Context, tx *gorm.DB, itemID uint, fileID uint) bool {
	var count int64
	err := tx.WithContext(ctx).
		Model(&database.AssetItem{}).
		Joins("JOIN media_assets ON media_assets.id = asset_items.asset_id").
		Joins("JOIN asset_files ON asset_files.asset_id = asset_items.asset_id").
		Where("asset_items.item_id = ?", itemID).
		Where("asset_files.role = ? AND asset_files.part_index = 0", inventory.FileRoleSource).
		Where("asset_files.file_id <> ?", fileID).
		Where("media_assets.deleted_at IS NULL").
		Count(&count).Error
	return err == nil && count > 0
}

func createOrReuseCatalogItem(ctx context.Context, tx *gorm.DB, catalogSvc *catalog.Service, input catalog.CreateItemInput) (database.CatalogItem, error) {
	pathValue := strings.TrimSpace(input.Path)
	if pathValue == "" {
		return database.CatalogItem{}, errors.New("catalog item path is required")
	}

	var item database.CatalogItem
	err := tx.WithContext(ctx).Where("library_id = ? AND path = ? AND deleted_at IS NULL", input.LibraryID, pathValue).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return catalogSvc.CreateItem(ctx, input)
	}
	if err != nil {
		return database.CatalogItem{}, err
	}

	updates := map[string]any{
		"parent_id":             input.ParentID,
		"sort_key":              defaultCatalogSortKey(input.SortKey, pathValue),
		"title":                 defaultCatalogTitle(input.Title, pathValue),
		"original_title":        strings.TrimSpace(input.OriginalTitle),
		"year":                  input.Year,
		"index_number":          input.IndexNumber,
		"parent_index_number":   input.ParentIndexNumber,
		"availability_status":   defaultCatalogState(input.AvailabilityStatus, catalog.AvailabilityAvailable),
		"governance_status":     defaultCatalogState(input.GovernanceStatus, catalog.GovernancePending),
		"deleted_at":            nil,
		"last_canonicalized_at": time.Now().UTC(),
	}
	if err := tx.WithContext(ctx).Model(&database.CatalogItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
		return database.CatalogItem{}, err
	}
	if err := tx.WithContext(ctx).First(&item, item.ID).Error; err != nil {
		return database.CatalogItem{}, err
	}
	return item, nil
}

func buildCatalogScanEvidencePayload(artifact catalogScanArtifact, episodeNumbers []int) string {
	payload := map[string]any{
		"storage_path":        strings.TrimSpace(artifact.SourcePath),
		"stable_identity_key": strings.TrimSpace(artifact.StableIdentityKey),
		"provider_name":       strings.TrimSpace(artifact.ProviderName),
		"hashes_json":         strings.TrimSpace(artifact.HashesJSON),
		"detected_title":      strings.TrimSpace(artifact.Title),
	}
	if strings.TrimSpace(artifact.SeriesTitle) != "" {
		payload["series_title"] = strings.TrimSpace(artifact.SeriesTitle)
	}
	if artifact.SeasonNumber != nil {
		payload["season_number"] = *artifact.SeasonNumber
	}
	if len(episodeNumbers) > 0 {
		payload["episode_numbers"] = episodeNumbers
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func defaultCatalogTitle(title string, fallbackPath string) string {
	if strings.TrimSpace(title) != "" {
		return strings.TrimSpace(title)
	}
	base := strings.TrimSuffix(path.Base(strings.TrimSpace(fallbackPath)), path.Ext(strings.TrimSpace(fallbackPath)))
	if strings.TrimSpace(base) != "" && base != "." && base != "/" {
		return cleanTitle(base)
	}
	return strings.TrimSpace(fallbackPath)
}

func defaultCatalogSortKey(sortKey string, fallback string) string {
	if strings.TrimSpace(sortKey) != "" {
		return strings.TrimSpace(sortKey)
	}
	return defaultCatalogTitle(fallback, fallback)
}

func defaultCatalogState(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func primaryOrSegmentRole(slotCount int) string {
	if slotCount > 1 {
		return inventory.AssetItemRoleMultiEpisodePart
	}
	return inventory.AssetItemRolePrimary
}

func episodeNumbersWithAppend(existing []int, value int) []int {
	numbers := append([]int(nil), existing...)
	numbers = append(numbers, value)
	return numbers
}

func canonicalSeriesPath(seriesTitle string) string {
	cleaned := strings.TrimSpace(cleanTitle(seriesTitle))
	if cleaned == "" {
		return "series"
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(cleaned) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	normalized := strings.Trim(builder.String(), "-")
	if normalized == "" {
		return "series"
	}
	return normalized
}

func canonicalEpisodeItemPath(seasonPath string, episodeNumber int) string {
	return fmt.Sprintf("%s/episode-%04d", seasonPath, episodeNumber)
}
