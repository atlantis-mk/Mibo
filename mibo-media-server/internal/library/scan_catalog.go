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
	"gorm.io/gorm/clause"
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

		file, err := upsertCatalogScanFile(ctx, tx, inventorySvc, library.ID, artifact)
		if err != nil {
			return err
		}
		asset, err := createOrReuseCatalogScanAsset(ctx, tx, inventorySvc, library.ID, file.ID, artifact, inventory.AssetTypeMain)
		if err != nil {
			return err
		}
		if err := bindCatalogScanSubtitleSidecars(ctx, tx, inventorySvc, library.ID, asset.ID, artifact); err != nil {
			return err
		}
		item, err := findOrCreateCatalogMovieItem(ctx, tx, catalogSvc, library.ID, asset.ID, artifact)
		if err != nil {
			return err
		}
		if _, err := inventorySvc.LinkAssetToItem(ctx, inventory.LinkAssetItemInput{AssetID: asset.ID, ItemID: item.ID, Role: inventory.AssetItemRolePrimary, SegmentIndex: 0, Source: "scanner"}); err != nil {
			return err
		}

		if err := recordCatalogScanMetadataSource(ctx, tx, catalogSvc, item.ID, buildCatalogScanEvidencePayload(artifact, nil)); err != nil {
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
			if err := recordCatalogScanMetadataSource(ctx, tx, catalogSvc, episodeItem.ID, buildCatalogScanEvidencePayload(artifact, episodeNumbersWithAppend(episodeNumbers, episodeNumber))); err != nil {
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
		if err := bindCatalogScanSubtitleSidecars(ctx, tx, inventorySvc, library.ID, asset.ID, artifact); err != nil {
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

func (s *Service) cleanupMissingCatalog(ctx context.Context, libraryID uint, rootPath string, seen map[string]struct{}) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var files []database.InventoryFile
		fileQuery := tx.WithContext(ctx).
			Where("library_id = ? AND deleted_at IS NULL", libraryID)
		fileQuery = applyScopedPathFilter(fileQuery, "storage_path", rootPath)
		if err := fileQuery.Order("id asc").Find(&files).Error; err != nil {
			return err
		}
		if len(files) == 0 {
			return nil
		}

		missingFileIDs := make([]uint, 0)
		for _, file := range files {
			if _, ok := seen[file.StoragePath]; ok {
				continue
			}
			missingFileIDs = append(missingFileIDs, file.ID)
		}
		if len(missingFileIDs) > 0 {
			if err := tx.WithContext(ctx).
				Model(&database.InventoryFile{}).
				Where("id IN ?", missingFileIDs).
				Updates(map[string]any{"status": inventory.FileStatusMissing, "deleted_at": nil}).Error; err != nil {
				return err
			}
		}

		assetIDs, err := scopedCatalogAssetIDs(ctx, tx, libraryID, rootPath)
		if err != nil {
			return err
		}
		for _, assetID := range assetIDs {
			status, err := catalogAssetAvailabilityStatus(ctx, tx, assetID)
			if err != nil {
				return err
			}
			if err := tx.WithContext(ctx).
				Model(&database.MediaAsset{}).
				Where("id = ?", assetID).
				Updates(map[string]any{"status": status, "deleted_at": nil}).Error; err != nil {
				return err
			}
		}

		itemIDs, err := scopedCatalogItemAndAncestorIDs(ctx, tx, libraryID, rootPath)
		if err != nil {
			return err
		}
		for _, itemID := range itemIDs {
			availability, err := catalogItemAvailabilityStatus(ctx, tx, itemID)
			if err != nil {
				return err
			}
			if err := tx.WithContext(ctx).
				Model(&database.CatalogItem{}).
				Where("id = ?", itemID).
				Update("availability_status", availability).Error; err != nil {
				return err
			}
		}

		return nil
	})
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

func bindCatalogScanSubtitleSidecars(ctx context.Context, tx *gorm.DB, inventorySvc *inventory.Service, libraryID uint, assetID uint, artifact catalogScanArtifact) error {
	currentFileIDs := make([]uint, 0, len(artifact.SubtitleSidecars))
	for _, sidecar := range artifact.SubtitleSidecars {
		file, err := upsertCatalogScanSubtitleFile(ctx, inventorySvc, libraryID, artifact, sidecar)
		if err != nil {
			return err
		}
		currentFileIDs = append(currentFileIDs, file.ID)
		if _, err := inventorySvc.LinkAssetToFile(ctx, inventory.LinkAssetFileInput{AssetID: assetID, FileID: file.ID, Role: inventory.FileRoleSubtitle}); err != nil {
			return err
		}
		if err := upsertCatalogScanSubtitleStream(ctx, tx, file.ID, artifact.SourcePath, sidecar); err != nil {
			return err
		}
	}
	return reconcileCatalogScanSubtitleSidecars(ctx, tx, assetID, currentFileIDs)
}

func upsertCatalogScanSubtitleFile(ctx context.Context, inventorySvc *inventory.Service, libraryID uint, artifact catalogScanArtifact, sidecar catalogScanSidecar) (database.InventoryFile, error) {
	storageProvider := strings.TrimSpace(artifact.StorageProvider)
	if storageProvider == "" {
		storageProvider = "local"
	}
	container := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(sidecar.Extension)), ".")
	if container == "" {
		container = strings.TrimPrefix(strings.ToLower(path.Ext(strings.TrimSpace(sidecar.Path))), ".")
	}
	return inventorySvc.UpsertFile(ctx, inventory.UpsertFileInput{
		LibraryID:         libraryID,
		StorageProvider:   storageProvider,
		StoragePath:       strings.TrimSpace(sidecar.Path),
		StableIdentityKey: strings.TrimSpace(sidecar.StableIdentityKey),
		SizeBytes:         sidecar.SizeBytes,
		ModifiedAt:        sidecar.ModifiedAt,
		Container:         container,
		Status:            inventory.FileStatusAvailable,
	})
}

func upsertCatalogScanSubtitleStream(ctx context.Context, tx *gorm.DB, fileID uint, sourcePath string, sidecar catalogScanSidecar) error {
	codec, title, language := subtitleSidecarStreamMetadata(sourcePath, sidecar)
	dispositionJSON, err := json.Marshal(map[string]any{
		"default":          false,
		"forced":           false,
		"hearing_impaired": false,
		"external":         inventory.MediaStreamDispositionExternalAvailable,
		"external_source":  inventory.MediaStreamDispositionExternalScanner,
		"managed_by":       inventory.MediaStreamDispositionManagedByScanner,
	})
	if err != nil {
		return err
	}
	stream := database.MediaStream{FileID: fileID, StreamIndex: 0, StreamType: inventory.MediaStreamTypeSubtitle, Codec: codec, Language: language, Title: title, DispositionJSON: string(dispositionJSON)}
	return tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "file_id"}, {Name: "stream_index"}},
		DoUpdates: clause.AssignmentColumns([]string{"stream_type", "codec", "language", "title", "disposition_json", "updated_at"}),
	}).Create(&stream).Error
}

func reconcileCatalogScanSubtitleSidecars(ctx context.Context, tx *gorm.DB, assetID uint, currentFileIDs []uint) error {
	var staleLinks []database.AssetFile
	query := tx.WithContext(ctx).Where("asset_id = ? AND role = ?", assetID, inventory.FileRoleSubtitle)
	if len(currentFileIDs) > 0 {
		query = query.Where("file_id NOT IN ?", currentFileIDs)
	}
	if err := query.Find(&staleLinks).Error; err != nil {
		return err
	}
	if len(staleLinks) == 0 {
		return nil
	}
	staleFileIDs := make([]uint, 0, len(staleLinks))
	for _, link := range staleLinks {
		staleFileIDs = append(staleFileIDs, link.FileID)
	}
	if err := tx.WithContext(ctx).Where("asset_id = ? AND role = ? AND file_id IN ?", assetID, inventory.FileRoleSubtitle, staleFileIDs).Delete(&database.AssetFile{}).Error; err != nil {
		return err
	}
	return tx.WithContext(ctx).Where("file_id IN ? AND stream_type = ?", staleFileIDs, inventory.MediaStreamTypeSubtitle).Delete(&database.MediaStream{}).Error
}

func subtitleSidecarStreamMetadata(sourcePath string, sidecar catalogScanSidecar) (string, string, string) {
	extension := strings.ToLower(strings.TrimSpace(sidecar.Extension))
	codec := strings.TrimPrefix(extension, ".")
	if codec == "" {
		codec = strings.TrimPrefix(strings.ToLower(path.Ext(strings.TrimSpace(sidecar.Path))), ".")
	}
	title := strings.TrimSuffix(path.Base(strings.TrimSpace(sidecar.Path)), path.Ext(strings.TrimSpace(sidecar.Path)))
	if strings.TrimSpace(title) == "" {
		title = "External subtitle"
	}
	language := subtitleLanguageFromFilename(sourcePath, sidecar.Path)
	return codec, title, language
}

func subtitleLanguageFromFilename(sourcePath string, subtitlePath string) string {
	subtitleBase := strings.TrimSuffix(path.Base(strings.TrimSpace(subtitlePath)), path.Ext(strings.TrimSpace(subtitlePath)))
	sourceBase := strings.TrimSuffix(path.Base(strings.TrimSpace(sourcePath)), path.Ext(strings.TrimSpace(sourcePath)))
	candidate := strings.TrimPrefix(strings.TrimPrefix(subtitleBase, sourceBase), ".")
	if candidate == subtitleBase || strings.TrimSpace(candidate) == "" {
		parts := strings.FieldsFunc(subtitleBase, func(r rune) bool { return r == '.' || r == '_' || r == '-' || unicode.IsSpace(r) })
		if len(parts) == 0 {
			return ""
		}
		candidate = parts[len(parts)-1]
	}
	switch strings.ToLower(strings.TrimSpace(candidate)) {
	case "en", "eng", "english":
		return "eng"
	case "zh", "zho", "chi", "chs", "cht", "cn", "sc", "tc", "chinese":
		return "zho"
	case "ja", "jpn", "jp", "japanese":
		return "jpn"
	case "ko", "kor", "kr", "korean":
		return "kor"
	case "fr", "fra", "fre", "french":
		return "fra"
	case "de", "deu", "ger", "german":
		return "deu"
	case "es", "spa", "spanish":
		return "spa"
	default:
		if len(candidate) == 3 && isASCIIAlpha(candidate) {
			return strings.ToLower(candidate)
		}
		return ""
	}
}

func isASCIIAlpha(value string) bool {
	for _, r := range value {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return value != ""
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
		err = findCatalogHierarchyItemForScan(ctx, tx, input, &item)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return catalogSvc.CreateItem(ctx, input)
	}
	if err != nil {
		return database.CatalogItem{}, err
	}

	updates := map[string]any{
		"path":                  pathValue,
		"parent_id":             input.ParentID,
		"sort_key":              defaultCatalogSortKey(input.SortKey, pathValue),
		"title":                 defaultCatalogTitle(input.Title, pathValue),
		"original_title":        strings.TrimSpace(input.OriginalTitle),
		"year":                  input.Year,
		"index_number":          input.IndexNumber,
		"parent_index_number":   input.ParentIndexNumber,
		"availability_status":   defaultCatalogState(input.AvailabilityStatus, catalog.AvailabilityAvailable),
		"governance_status":     governanceStatusForScanUpdate(item.GovernanceStatus, input.GovernanceStatus),
		"deleted_at":            nil,
		"last_canonicalized_at": time.Now().UTC(),
	}
	if err := applyCatalogScanMetadataOverrides(ctx, tx, item, updates); err != nil {
		return database.CatalogItem{}, err
	}
	if err := tx.WithContext(ctx).Model(&database.CatalogItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
		return database.CatalogItem{}, err
	}
	if err := tx.WithContext(ctx).First(&item, item.ID).Error; err != nil {
		return database.CatalogItem{}, err
	}
	return item, nil
}

func findOrCreateCatalogMovieItem(ctx context.Context, tx *gorm.DB, catalogSvc *catalog.Service, libraryID uint, assetID uint, artifact catalogScanArtifact) (database.CatalogItem, error) {
	if existing, err := findCatalogItemForAsset(ctx, tx, assetID); err == nil {
		updates := map[string]any{
			"path":                  strings.TrimSpace(artifact.ItemPath),
			"sort_key":              defaultCatalogSortKey(artifact.Title, artifact.ItemPath),
			"title":                 defaultCatalogTitle(artifact.Title, artifact.SourcePath),
			"original_title":        strings.TrimSpace(artifact.OriginalTitle),
			"year":                  artifact.Year,
			"availability_status":   catalog.AvailabilityAvailable,
			"governance_status":     governanceStatusForScanUpdate(existing.GovernanceStatus, catalog.GovernancePending),
			"last_canonicalized_at": time.Now().UTC(),
			"deleted_at":            nil,
		}
		if err := applyCatalogScanMetadataOverrides(ctx, tx, existing, updates); err != nil {
			return database.CatalogItem{}, err
		}
		if err := tx.WithContext(ctx).Model(&database.CatalogItem{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
			return database.CatalogItem{}, err
		}
		if err := tx.WithContext(ctx).First(&existing, existing.ID).Error; err != nil {
			return database.CatalogItem{}, err
		}
		return existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return database.CatalogItem{}, err
	}

	return createOrReuseCatalogItem(ctx, tx, catalogSvc, catalog.CreateItemInput{
		LibraryID:          libraryID,
		Type:               artifact.ItemType,
		Path:               artifact.ItemPath,
		SortKey:            defaultCatalogSortKey(artifact.Title, artifact.ItemPath),
		Title:              defaultCatalogTitle(artifact.Title, artifact.SourcePath),
		OriginalTitle:      strings.TrimSpace(artifact.OriginalTitle),
		Year:               artifact.Year,
		AvailabilityStatus: catalog.AvailabilityAvailable,
		GovernanceStatus:   catalog.GovernancePending,
	})
}

func findCatalogItemForAsset(ctx context.Context, tx *gorm.DB, assetID uint) (database.CatalogItem, error) {
	var item database.CatalogItem
	err := tx.WithContext(ctx).
		Joins("JOIN asset_items ON asset_items.item_id = catalog_items.id").
		Where("asset_items.asset_id = ? AND catalog_items.deleted_at IS NULL", assetID).
		Order("catalog_items.id asc").
		First(&item).Error
	return item, err
}

func applyCatalogScanMetadataOverrides(ctx context.Context, tx *gorm.DB, item database.CatalogItem, updates map[string]any) error {
	if item.ID == 0 || len(updates) == 0 {
		return nil
	}

	if catalogScanShouldPreserveDescriptiveFields(item.GovernanceStatus) {
		if _, ok := updates["title"]; ok {
			updates["title"] = item.Title
		}
		if _, ok := updates["original_title"]; ok {
			updates["original_title"] = item.OriginalTitle
		}
		if _, ok := updates["year"]; ok {
			updates["year"] = item.Year
		}
	}

	fieldKeys := make([]string, 0, 3)
	if _, ok := updates["title"]; ok {
		fieldKeys = append(fieldKeys, "title")
	}
	if _, ok := updates["original_title"]; ok {
		fieldKeys = append(fieldKeys, "original_title")
	}
	if _, ok := updates["year"]; ok {
		fieldKeys = append(fieldKeys, "year")
	}
	if len(fieldKeys) == 0 {
		return nil
	}

	var states []database.MetadataFieldState
	if err := tx.WithContext(ctx).
		Where("item_id = ? AND field_key IN ?", item.ID, fieldKeys).
		Find(&states).Error; err != nil {
		return err
	}
	for _, state := range states {
		switch state.FieldKey {
		case "title", "original_title":
			var value string
			if err := json.Unmarshal([]byte(state.ValueJSON), &value); err != nil {
				return fmt.Errorf("decode catalog field state %s for item %d: %w", state.FieldKey, item.ID, err)
			}
			updates[state.FieldKey] = value
		case "year":
			var value int
			if err := json.Unmarshal([]byte(state.ValueJSON), &value); err != nil {
				return fmt.Errorf("decode catalog field state %s for item %d: %w", state.FieldKey, item.ID, err)
			}
			updates["year"] = value
		}
	}
	return nil
}

func catalogScanShouldPreserveDescriptiveFields(governanceStatus string) bool {
	switch strings.TrimSpace(governanceStatus) {
	case catalog.GovernanceMatched, catalog.GovernanceNeedsReview, catalog.GovernanceLocked, catalog.GovernanceManual:
		return true
	default:
		return false
	}
}

func recordCatalogScanMetadataSource(ctx context.Context, tx *gorm.DB, catalogSvc *catalog.Service, itemID uint, payloadJSON string) error {
	now := time.Now().UTC()
	var existing database.MetadataSource
	err := tx.WithContext(ctx).
		Where("item_id = ? AND source_type = ? AND source_name = ? AND external_id = ? AND language = ?", itemID, catalog.SourceTypeLocalFile, "scanner", "", "").
		First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		_, err = catalogSvc.RecordMetadataSource(ctx, catalog.MetadataSourceInput{
			ItemID:      itemID,
			SourceType:  catalog.SourceTypeLocalFile,
			SourceName:  "scanner",
			PayloadJSON: payloadJSON,
			FetchedAt:   now,
		})
		return err
	}
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Model(&database.MetadataSource{}).Where("id = ?", existing.ID).Updates(map[string]any{
		"payload_json": payloadJSON,
		"fetched_at":   now,
		"updated_at":   now,
	}).Error
}

func buildCatalogScanEvidencePayload(artifact catalogScanArtifact, episodeNumbers []int) string {
	payload := map[string]any{
		"storage_path":        strings.TrimSpace(artifact.SourcePath),
		"stable_identity_key": strings.TrimSpace(artifact.StableIdentityKey),
		"provider_name":       strings.TrimSpace(artifact.ProviderName),
		"hashes_json":         strings.TrimSpace(artifact.HashesJSON),
		"detected_title":      strings.TrimSpace(artifact.Title),
	}
	if strings.TrimSpace(artifact.ObjectType) != "" {
		payload["object_type"] = strings.TrimSpace(artifact.ObjectType)
	}
	if len(artifact.ProviderMeta) > 0 {
		payload["provider_metadata"] = artifact.ProviderMeta
	}
	if strings.TrimSpace(artifact.NormalizationVersion) != "" {
		payload["normalization_version"] = strings.TrimSpace(artifact.NormalizationVersion)
	}
	if len(artifact.RemovedTokens) > 0 {
		payload["removed_tokens"] = artifact.RemovedTokens
	}
	if len(artifact.SubtitleSidecars) > 0 {
		payload["subtitle_sidecars"] = sidecarEvidencePayload(artifact.SubtitleSidecars)
	}
	if len(artifact.MetadataSidecars) > 0 {
		payload["metadata_sidecars"] = metadataSidecarEvidencePayload(artifact.MetadataSidecars)
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

func sidecarEvidencePayload(sidecars []catalogScanSidecar) []map[string]any {
	items := make([]map[string]any, 0, len(sidecars))
	for _, sidecar := range sidecars {
		item := map[string]any{
			"path":               strings.TrimSpace(sidecar.Path),
			"extension":          strings.TrimSpace(sidecar.Extension),
			"association_source": strings.TrimSpace(sidecar.AssociationSource),
		}
		items = append(items, item)
	}
	return items
}

func metadataSidecarEvidencePayload(sidecars []catalogScanMetadataSidecar) []map[string]any {
	items := make([]map[string]any, 0, len(sidecars))
	for _, sidecar := range sidecars {
		item := map[string]any{
			"path":               strings.TrimSpace(sidecar.Path),
			"extension":          strings.TrimSpace(sidecar.Extension),
			"association_source": strings.TrimSpace(sidecar.AssociationSource),
			"parse_status":       strings.TrimSpace(sidecar.ParseStatus),
		}
		if hints := metadataHintsPayload(sidecar.Hints); len(hints) > 0 {
			item["hints"] = hints
		}
		if len(sidecar.ExternalIDs) > 0 {
			item["external_ids"] = sidecar.ExternalIDs
		}
		items = append(items, item)
	}
	return items
}

func metadataHintsPayload(hints catalogScanMetadataHints) map[string]any {
	payload := make(map[string]any)
	if strings.TrimSpace(hints.Title) != "" {
		payload["title"] = strings.TrimSpace(hints.Title)
	}
	if strings.TrimSpace(hints.OriginalTitle) != "" {
		payload["original_title"] = strings.TrimSpace(hints.OriginalTitle)
	}
	if hints.Year != nil {
		payload["year"] = *hints.Year
	}
	if strings.TrimSpace(hints.MediaType) != "" {
		payload["media_type"] = strings.TrimSpace(hints.MediaType)
	}
	if strings.TrimSpace(hints.SeriesTitle) != "" {
		payload["series_title"] = strings.TrimSpace(hints.SeriesTitle)
	}
	if hints.SeasonNumber != nil {
		payload["season_number"] = *hints.SeasonNumber
	}
	if hints.EpisodeNumber != nil {
		payload["episode_number"] = *hints.EpisodeNumber
	}
	return payload
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

func findCatalogHierarchyItemForScan(ctx context.Context, tx *gorm.DB, input catalog.CreateItemInput, target *database.CatalogItem) error {
	if target == nil || input.ParentID == nil || input.IndexNumber == nil {
		return gorm.ErrRecordNotFound
	}
	itemType := strings.TrimSpace(input.Type)
	if itemType != catalog.ItemTypeSeason && itemType != catalog.ItemTypeEpisode {
		return gorm.ErrRecordNotFound
	}
	return tx.WithContext(ctx).
		Where("library_id = ? AND parent_id = ? AND type = ? AND index_number = ? AND deleted_at IS NULL", input.LibraryID, *input.ParentID, itemType, *input.IndexNumber).
		Order("id asc").
		First(target).Error
}

func governanceStatusForScanUpdate(existingStatus string, desiredStatus string) string {
	trimmedDesired := strings.TrimSpace(desiredStatus)
	trimmedExisting := strings.TrimSpace(existingStatus)
	if trimmedDesired == "" || trimmedDesired == catalog.GovernancePending {
		if trimmedExisting != "" {
			return trimmedExisting
		}
	}
	return defaultCatalogState(trimmedDesired, catalog.GovernancePending)
}

func scopedCatalogAssetIDs(ctx context.Context, tx *gorm.DB, libraryID uint, rootPath string) ([]uint, error) {
	var assetIDs []uint
	query := tx.WithContext(ctx).
		Table("asset_files").
		Distinct("asset_files.asset_id").
		Joins("JOIN inventory_files ON inventory_files.id = asset_files.file_id").
		Where("inventory_files.library_id = ? AND inventory_files.deleted_at IS NULL", libraryID)
	query = applyScopedPathFilter(query, "inventory_files.storage_path", rootPath)
	if err := query.Order("asset_files.asset_id asc").Pluck("asset_files.asset_id", &assetIDs).Error; err != nil {
		return nil, err
	}
	return assetIDs, nil
}

func catalogAssetAvailabilityStatus(ctx context.Context, tx *gorm.DB, assetID uint) (string, error) {
	var availableCount int64
	err := tx.WithContext(ctx).
		Table("asset_files").
		Joins("JOIN inventory_files ON inventory_files.id = asset_files.file_id").
		Where("asset_files.asset_id = ? AND asset_files.role = ? AND inventory_files.deleted_at IS NULL AND inventory_files.status = ?", assetID, inventory.FileRoleSource, inventory.FileStatusAvailable).
		Count(&availableCount).Error
	if err != nil {
		return "", err
	}
	if availableCount > 0 {
		return inventory.AssetStatusAvailable, nil
	}
	return inventory.AssetStatusMissing, nil
}

func scopedCatalogItemIDs(ctx context.Context, tx *gorm.DB, libraryID uint, rootPath string) ([]uint, error) {
	var itemIDs []uint
	query := tx.WithContext(ctx).
		Table("asset_items").
		Distinct("asset_items.item_id").
		Joins("JOIN media_assets ON media_assets.id = asset_items.asset_id").
		Joins("JOIN asset_files ON asset_files.asset_id = media_assets.id").
		Joins("JOIN inventory_files ON inventory_files.id = asset_files.file_id").
		Joins("JOIN catalog_items ON catalog_items.id = asset_items.item_id").
		Where("catalog_items.library_id = ? AND catalog_items.deleted_at IS NULL AND inventory_files.deleted_at IS NULL", libraryID)
	query = applyScopedPathFilter(query, "inventory_files.storage_path", rootPath)
	if err := query.Order("asset_items.item_id asc").Pluck("asset_items.item_id", &itemIDs).Error; err != nil {
		return nil, err
	}
	return itemIDs, nil
}

func applyScopedPathFilter(query *gorm.DB, column string, rootPath string) *gorm.DB {
	scope := strings.TrimRight(strings.TrimSpace(rootPath), "/")
	if scope == "" {
		return query
	}
	return query.Where(column+" = ? OR "+column+" LIKE ?", scope, scope+"/%")
}

func scopedCatalogItemAndAncestorIDs(ctx context.Context, tx *gorm.DB, libraryID uint, rootPath string) ([]uint, error) {
	itemIDs, err := scopedCatalogItemIDs(ctx, tx, libraryID, rootPath)
	if err != nil || len(itemIDs) == 0 {
		return itemIDs, err
	}

	var items []database.CatalogItem
	if err := tx.WithContext(ctx).
		Select("id", "parent_id").
		Where("library_id = ? AND deleted_at IS NULL", libraryID).
		Find(&items).Error; err != nil {
		return nil, err
	}
	parentByID := make(map[uint]*uint, len(items))
	for _, item := range items {
		parentByID[item.ID] = item.ParentID
	}

	ids := make(map[uint]struct{}, len(itemIDs))
	for _, itemID := range itemIDs {
		current := itemID
		for {
			if _, exists := ids[current]; exists {
				break
			}
			ids[current] = struct{}{}
			parentID := parentByID[current]
			if parentID == nil {
				break
			}
			current = *parentID
		}
	}

	result := make([]uint, 0, len(ids))
	for id := range ids {
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func catalogItemAvailabilityStatus(ctx context.Context, tx *gorm.DB, itemID uint) (string, error) {
	var item database.CatalogItem
	if err := tx.WithContext(ctx).
		Select("id", "availability_status").
		Where("id = ? AND deleted_at IS NULL", itemID).
		First(&item).Error; err != nil {
		return "", err
	}

	var children []database.CatalogItem
	if err := tx.WithContext(ctx).
		Select("id", "availability_status").
		Where("parent_id = ? AND deleted_at IS NULL", itemID).
		Order("index_number asc, id asc").
		Find(&children).Error; err != nil {
		return "", err
	}
	if len(children) > 0 {
		hasUnaired := false
		for _, child := range children {
			availability, err := catalogItemAvailabilityStatus(ctx, tx, child.ID)
			if err != nil {
				return "", err
			}
			switch strings.TrimSpace(availability) {
			case catalog.AvailabilityAvailable:
				return catalog.AvailabilityAvailable, nil
			case catalog.AvailabilityMissing, catalog.AvailabilityNoLocalMedia:
				return catalog.AvailabilityMissing, nil
			case catalog.AvailabilityUnaired:
				hasUnaired = true
			}
		}
		if hasUnaired {
			return catalog.AvailabilityUnaired, nil
		}
		return catalog.AvailabilityNoLocalMedia, nil
	}

	var availableCount int64
	err := tx.WithContext(ctx).
		Table("asset_items").
		Joins("JOIN media_assets ON media_assets.id = asset_items.asset_id").
		Where("asset_items.item_id = ? AND media_assets.deleted_at IS NULL AND media_assets.status = ?", itemID, inventory.AssetStatusAvailable).
		Count(&availableCount).Error
	if err != nil {
		return "", err
	}
	if availableCount > 0 {
		return catalog.AvailabilityAvailable, nil
	}
	if strings.TrimSpace(item.AvailabilityStatus) == catalog.AvailabilityUnaired {
		return catalog.AvailabilityUnaired, nil
	}
	return catalog.AvailabilityMissing, nil
}
