package metadata

import (
	"context"
	"errors"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	governanceClassificationActionMovieVersions     = "movie_versions"
	governanceClassificationActionIndependentMovies = "independent_movies"
)

type ApplyGovernanceClassificationCorrectionInput struct {
	Action   string
	RootPath string
	Title    string
}

type governanceMovieVersionSource struct {
	AssetID      uint
	FileID       uint
	SourceItemID uint
	StoragePath  string
}

type governanceClassificationCorrectionEvidence struct {
	Action        string `json:"action"`
	MovieItemID   uint   `json:"movie_item_id,omitempty"`
	MovieItemIDs  []uint `json:"movie_item_ids,omitempty"`
	SourceItemIDs []uint `json:"source_item_ids"`
	AssetIDs      []uint `json:"asset_ids"`
	FileIDs       []uint `json:"file_ids"`
	RootPath      string `json:"root_path"`
}

func (s *Service) ApplyCatalogGovernanceClassificationCorrectionOperation(ctx context.Context, itemID uint, input ApplyGovernanceClassificationCorrectionInput) (MetadataOperationResult, error) {
	action := strings.TrimSpace(input.Action)
	if action != governanceClassificationActionMovieVersions && action != governanceClassificationActionIndependentMovies {
		return MetadataOperationResult{}, errors.New("unsupported classification correction action")
	}
	origin, err := s.loadGovernanceCatalogItem(ctx, itemID)
	if err != nil {
		return MetadataOperationResult{}, err
	}
	plan, err := s.resolveMetadataExecutionPlan(ctx, origin.LibraryID)
	if err != nil {
		return MetadataOperationResult{}, err
	}

	var movie database.CatalogItem
	var movies []database.CatalogItem
	var affectedIDs []uint
	var evidence governanceClassificationCorrectionEvidence
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		root, scopeItems, err := loadGovernanceSeriesScope(ctx, tx, origin)
		if err != nil {
			return err
		}
		rootPath := strings.TrimSpace(input.RootPath)
		if rootPath == "" {
			rootPath = strings.TrimSpace(root.Path)
		}
		sources, err := loadGovernanceMovieVersionSources(ctx, tx, root.LibraryID, rootPath, scopeItems)
		if err != nil {
			return err
		}
		if action == governanceClassificationActionMovieVersions {
			movie, err = findOrCreateGovernanceMovieVersionItem(ctx, tx, root, rootPath, input.Title)
			if err != nil {
				return err
			}
			movies = []database.CatalogItem{movie}
			if err := relinkGovernanceMovieVersionAssets(ctx, tx, movie.ID, sources); err != nil {
				return err
			}
			if err := copyGovernanceMovieVersionImages(ctx, tx, movie.ID, scopeItems); err != nil {
				return err
			}
		} else {
			movies, err = findOrCreateGovernanceIndependentMovieItems(ctx, tx, root, input.Title, sources)
			if err != nil {
				return err
			}
			if err := relinkGovernanceIndependentMovieAssets(ctx, tx, movies, sources); err != nil {
				return err
			}
		}

		now := time.Now().UTC()
		scopeIDs := catalogItemIDs(scopeItems)
		if err := tx.WithContext(ctx).Model(&database.CatalogItem{}).Where("id IN ?", scopeIDs).Updates(map[string]any{"deleted_at": &now, "governance_status": catalog.GovernanceManual, "last_canonicalized_at": now}).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Where("item_id IN ?", scopeIDs).Delete(&database.CatalogSearchDocument{}).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Where("item_id IN ?", scopeIDs).Delete(&database.ItemRollup{}).Error; err != nil {
			return err
		}
		if err := acceptGovernanceClassificationDecisions(ctx, tx, scopeIDs, sources, now); err != nil {
			return err
		}
		movieIDs := catalogItemIDs(movies)
		affectedIDs = appendUniqueUint(movieIDs, scopeIDs...)
		evidence = governanceClassificationCorrectionEvidence{Action: action, MovieItemIDs: movieIDs, SourceItemIDs: scopeIDs, AssetIDs: governanceSourceAssetIDs(sources), FileIDs: governanceSourceFileIDs(sources), RootPath: rootPath}
		if len(movieIDs) > 0 {
			evidence.MovieItemID = movieIDs[0]
			movie = movies[0]
		}
		return nil
	})
	if err != nil {
		return MetadataOperationResult{}, err
	}

	result := MetadataOperationResult{Operation: OperationTypeGovernanceClassificationCorrection, OriginItemID: origin.ID, TargetItemID: movie.ID, TargetType: movie.Type, Status: OperationStatusApplied, GovernanceStatus: strings.TrimSpace(movie.GovernanceStatus), Plan: metadataExecutionPlanSummary(plan), AffectedScope: MetadataAffectedScope{ItemIDs: affectedIDs, LibraryID: movie.LibraryID, RootID: movie.RootID}, AppliedFields: []MetadataAppliedField{{ItemID: movie.ID, FieldKey: "classification." + action, ApplyMode: FieldApplyModeManual}}}
	for _, assetID := range evidence.AssetIDs {
		result.AppliedFields = append(result.AppliedFields, MetadataAppliedField{ItemID: movie.ID, FieldKey: "assets." + uintStringForEvidence(assetID), ApplyMode: FieldApplyModeManual})
	}
	if err := s.refreshMetadataOperationProjectionScope(ctx, MetadataAffectedScope{ItemIDs: []uint{movie.ID}}); err != nil {
		return MetadataOperationResult{}, err
	}
	if s.ingest != nil {
		for _, fileID := range evidence.FileIDs {
			if _, err := s.ingest.MarkInventoryFileDirty(ctx, fileID, "classification_corrected_"+action); err != nil {
				return MetadataOperationResult{}, err
			}
		}
	}
	_, err = s.recordMetadataOperation(ctx, MetadataOperationEvidenceInput{Result: result, LibraryID: movie.LibraryID, SelectedCandidate: evidence, StartedAt: time.Now().UTC()})
	return result, err
}

func loadGovernanceSeriesScope(ctx context.Context, tx *gorm.DB, item database.CatalogItem) (database.CatalogItem, []database.CatalogItem, error) {
	rootID := item.ID
	if item.Type != catalog.ItemTypeSeries && item.RootID != nil && *item.RootID != 0 {
		rootID = *item.RootID
	}
	var root database.CatalogItem
	if err := tx.WithContext(ctx).Where("id = ? AND type = ? AND deleted_at IS NULL", rootID, catalog.ItemTypeSeries).First(&root).Error; err != nil {
		return database.CatalogItem{}, nil, errors.New("classification correction requires a series or episode under a series")
	}
	var items []database.CatalogItem
	if err := tx.WithContext(ctx).Where("library_id = ? AND deleted_at IS NULL AND (id = ? OR root_id = ?)", root.LibraryID, root.ID, root.ID).Order("id asc").Find(&items).Error; err != nil {
		return database.CatalogItem{}, nil, err
	}
	if len(items) == 0 {
		return database.CatalogItem{}, nil, errors.New("series scope is empty")
	}
	return root, items, nil
}

func loadGovernanceMovieVersionSources(ctx context.Context, tx *gorm.DB, libraryID uint, rootPath string, scopeItems []database.CatalogItem) ([]governanceMovieVersionSource, error) {
	var rows []governanceMovieVersionSource
	rootPath = strings.TrimRight(strings.TrimSpace(rootPath), "/")
	if rootPath == "" {
		return nil, errors.New("movie root_path is required")
	}
	likePath := rootPath + "/%"
	if err := tx.WithContext(ctx).
		Table("inventory_files").
		Select("media_assets.id AS asset_id, COALESCE(asset_items.item_id, 0) AS source_item_id, inventory_files.id AS file_id, inventory_files.storage_path").
		Joins("JOIN asset_files ON asset_files.file_id = inventory_files.id AND asset_files.role = ?", inventory.FileRoleSource).
		Joins("JOIN media_assets ON media_assets.id = asset_files.asset_id AND media_assets.deleted_at IS NULL").
		Joins("LEFT JOIN asset_items ON asset_items.asset_id = media_assets.id").
		Where("inventory_files.library_id = ? AND inventory_files.deleted_at IS NULL AND inventory_files.content_class = ?", libraryID, "video").
		Where("inventory_files.storage_path = ? OR inventory_files.storage_path LIKE ?", rootPath, likePath).
		Order("inventory_files.storage_path asc, asset_items.asset_id asc").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 && len(scopeItems) > 0 {
		if err := tx.WithContext(ctx).
			Table("asset_items").
			Select("asset_items.asset_id, asset_items.item_id AS source_item_id, asset_files.file_id, inventory_files.storage_path").
			Joins("JOIN media_assets ON media_assets.id = asset_items.asset_id AND media_assets.deleted_at IS NULL").
			Joins("JOIN asset_files ON asset_files.asset_id = media_assets.id AND asset_files.role = ?", inventory.FileRoleSource).
			Joins("JOIN inventory_files ON inventory_files.id = asset_files.file_id AND inventory_files.deleted_at IS NULL AND inventory_files.content_class = ?", "video").
			Where("asset_items.item_id IN ? AND inventory_files.library_id = ?", catalogItemIDs(scopeItems), libraryID).
			Order("inventory_files.storage_path asc, asset_items.asset_id asc").
			Scan(&rows).Error; err != nil {
			return nil, err
		}
	}
	seen := make(map[uint]struct{}, len(rows))
	sources := make([]governanceMovieVersionSource, 0, len(rows))
	for _, row := range rows {
		if row.AssetID == 0 || row.FileID == 0 || strings.TrimSpace(row.StoragePath) == "" {
			continue
		}
		if _, ok := seen[row.AssetID]; ok {
			continue
		}
		seen[row.AssetID] = struct{}{}
		sources = append(sources, row)
	}
	if len(sources) == 0 {
		return nil, errors.New("no video assets found in series scope")
	}
	return sources, nil
}

func findOrCreateGovernanceMovieVersionItem(ctx context.Context, tx *gorm.DB, root database.CatalogItem, inputRootPath string, inputTitle string) (database.CatalogItem, error) {
	rootPath := strings.TrimSpace(inputRootPath)
	if rootPath == "" {
		rootPath = strings.TrimSpace(root.Title)
	}
	title := strings.TrimSpace(inputTitle)
	if title == "" {
		title = strings.TrimSpace(root.Title)
	}
	if title == "" {
		title = path.Base(rootPath)
	}
	return findOrCreateGovernanceMovieItem(ctx, tx, root.LibraryID, rootPath, title)
}

func relinkGovernanceMovieVersionAssets(ctx context.Context, tx *gorm.DB, movieID uint, sources []governanceMovieVersionSource) error {
	if len(sources) == 0 {
		return errors.New("at least one source asset is required")
	}
	assetIDs := governanceSourceAssetIDs(sources)
	if err := tx.WithContext(ctx).Where("asset_id IN ?", assetIDs).Delete(&database.AssetItem{}).Error; err != nil {
		return err
	}
	for idx, source := range sources {
		assetType := inventory.AssetTypeVersion
		role := inventory.AssetItemRoleVersion
		if idx == 0 {
			assetType = inventory.AssetTypeMain
			role = inventory.AssetItemRolePrimary
		}
		if err := tx.WithContext(ctx).Model(&database.MediaAsset{}).Where("id = ?", source.AssetID).Updates(map[string]any{"asset_type": assetType, "updated_at": time.Now().UTC()}).Error; err != nil {
			return err
		}
		link := database.AssetItem{AssetID: source.AssetID, ItemID: movieID, Role: role, SegmentIndex: 0, Source: "governance"}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "asset_id"}, {Name: "item_id"}, {Name: "role"}, {Name: "segment_index"}}, DoUpdates: clause.AssignmentColumns([]string{"source", "updated_at"})}).Create(&link).Error; err != nil {
			return err
		}
	}
	return nil
}

func findOrCreateGovernanceIndependentMovieItems(ctx context.Context, tx *gorm.DB, root database.CatalogItem, inputTitle string, sources []governanceMovieVersionSource) ([]database.CatalogItem, error) {
	movies := make([]database.CatalogItem, 0, len(sources))
	for idx, source := range sources {
		storagePath := strings.TrimSpace(source.StoragePath)
		if storagePath == "" {
			return nil, errors.New("movie storage_path is required")
		}
		title := path.Base(storagePath)
		if idx == 0 && strings.TrimSpace(inputTitle) != "" {
			title = strings.TrimSpace(inputTitle)
		}
		movie, err := findOrCreateGovernanceMovieItem(ctx, tx, root.LibraryID, storagePath, title)
		if err != nil {
			return nil, err
		}
		movies = append(movies, movie)
	}
	return movies, nil
}

func findOrCreateGovernanceMovieItem(ctx context.Context, tx *gorm.DB, libraryID uint, moviePath string, title string) (database.CatalogItem, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = path.Base(moviePath)
	}
	var movie database.CatalogItem
	err := tx.WithContext(ctx).Where("library_id = ? AND type = ? AND path = ? AND deleted_at IS NULL", libraryID, catalog.ItemTypeMovie, moviePath).First(&movie).Error
	if err == nil {
		updates := map[string]any{"parent_id": nil, "root_id": movie.ID, "title": title, "sort_key": title, "availability_status": catalog.AvailabilityAvailable, "governance_status": catalog.GovernanceManual, "deleted_at": nil, "last_canonicalized_at": time.Now().UTC()}
		if err := tx.WithContext(ctx).Model(&database.CatalogItem{}).Where("id = ?", movie.ID).Updates(updates).Error; err != nil {
			return database.CatalogItem{}, err
		}
		return movie, tx.WithContext(ctx).First(&movie, movie.ID).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return database.CatalogItem{}, err
	}
	now := time.Now().UTC()
	movie = database.CatalogItem{LibraryID: libraryID, Type: catalog.ItemTypeMovie, Path: moviePath, SortKey: title, DisplayOrder: catalog.DisplayOrderAired, Title: title, AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernanceManual, CanonicalVersion: 1, LastCanonicalizedAt: &now}
	if err := tx.WithContext(ctx).Create(&movie).Error; err != nil {
		return database.CatalogItem{}, err
	}
	if err := tx.WithContext(ctx).Model(&movie).Update("root_id", movie.ID).Error; err != nil {
		return database.CatalogItem{}, err
	}
	movie.RootID = &movie.ID
	return movie, nil
}

func relinkGovernanceIndependentMovieAssets(ctx context.Context, tx *gorm.DB, movies []database.CatalogItem, sources []governanceMovieVersionSource) error {
	if len(movies) == 0 || len(movies) != len(sources) {
		return errors.New("independent movie correction requires one movie per source asset")
	}
	assetIDs := governanceSourceAssetIDs(sources)
	if err := tx.WithContext(ctx).Where("asset_id IN ?", assetIDs).Delete(&database.AssetItem{}).Error; err != nil {
		return err
	}
	for idx, source := range sources {
		if err := tx.WithContext(ctx).Model(&database.MediaAsset{}).Where("id = ?", source.AssetID).Updates(map[string]any{"asset_type": inventory.AssetTypeMain, "updated_at": time.Now().UTC()}).Error; err != nil {
			return err
		}
		link := database.AssetItem{AssetID: source.AssetID, ItemID: movies[idx].ID, Role: inventory.AssetItemRolePrimary, SegmentIndex: 0, Source: "governance"}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "asset_id"}, {Name: "item_id"}, {Name: "role"}, {Name: "segment_index"}}, DoUpdates: clause.AssignmentColumns([]string{"source", "updated_at"})}).Create(&link).Error; err != nil {
			return err
		}
	}
	return nil
}

func copyGovernanceMovieVersionImages(ctx context.Context, tx *gorm.DB, movieID uint, scopeItems []database.CatalogItem) error {
	scopeIDs := catalogItemIDs(scopeItems)
	if movieID == 0 || len(scopeIDs) == 0 {
		return nil
	}
	var images []database.ItemImage
	if err := tx.WithContext(ctx).
		Where("item_id IN ? AND image_type IN ? AND is_selected = ?", scopeIDs, []string{"poster", "still", "backdrop"}, true).
		Order("sort_order asc, id asc").
		Find(&images).Error; err != nil {
		return err
	}
	for _, image := range images {
		imageType := governanceMovieVersionTargetImageType(image.ImageType)
		if imageType == "" {
			continue
		}
		shouldSelect, err := governanceMovieVersionImageShouldSelect(ctx, tx, movieID, imageType)
		if err != nil {
			return err
		}
		if !shouldSelect {
			continue
		}
		var existing int64
		if err := tx.WithContext(ctx).Model(&database.ItemImage{}).Where("item_id = ? AND image_type = ? AND url = ?", movieID, imageType, image.URL).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			if err := selectGovernanceMovieVersionImage(ctx, tx, movieID, imageType, image.URL); err != nil {
				return err
			}
			continue
		}
		copy := database.ItemImage{ItemID: movieID, ImageType: imageType, URL: image.URL, SourceID: image.SourceID, Language: image.Language, Width: image.Width, Height: image.Height, IsSelected: true, SortOrder: image.SortOrder}
		if err := tx.WithContext(ctx).Create(&copy).Error; err != nil {
			return err
		}
	}
	return nil
}

func governanceMovieVersionTargetImageType(imageType string) string {
	switch strings.TrimSpace(imageType) {
	case "poster", "still":
		return "poster"
	case "backdrop":
		return "backdrop"
	default:
		return ""
	}
}

func governanceMovieVersionImageShouldSelect(ctx context.Context, tx *gorm.DB, itemID uint, imageType string) (bool, error) {
	var selectedCount int64
	if err := tx.WithContext(ctx).Model(&database.ItemImage{}).Where("item_id = ? AND image_type = ? AND is_selected = ?", itemID, imageType, true).Count(&selectedCount).Error; err != nil {
		return false, err
	}
	return selectedCount == 0, nil
}

func selectGovernanceMovieVersionImage(ctx context.Context, tx *gorm.DB, itemID uint, imageType string, url string) error {
	if err := tx.WithContext(ctx).Model(&database.ItemImage{}).Where("item_id = ? AND image_type = ?", itemID, imageType).Update("is_selected", false).Error; err != nil {
		return err
	}
	return tx.WithContext(ctx).Model(&database.ItemImage{}).Where("item_id = ? AND image_type = ? AND url = ?", itemID, imageType, url).Update("is_selected", true).Error
}

func acceptGovernanceClassificationDecisions(ctx context.Context, tx *gorm.DB, scopeIDs []uint, sources []governanceMovieVersionSource, now time.Time) error {
	fileIDs := governanceSourceFileIDs(sources)
	assetIDs := governanceSourceAssetIDs(sources)
	query := tx.WithContext(ctx).Model(&database.ClassificationDecision{}).Where("status IN ?", []string{"provisional", "review_required"}).Where("item_id IN ?", scopeIDs)
	if len(fileIDs) > 0 && len(assetIDs) > 0 {
		query = tx.WithContext(ctx).Model(&database.ClassificationDecision{}).Where("status IN ?", []string{"provisional", "review_required"}).Where("item_id IN ? OR inventory_file_id IN ? OR asset_id IN ?", scopeIDs, fileIDs, assetIDs)
	}
	return query.Updates(map[string]any{"status": "accepted", "resolved_at": now, "updated_at": now}).Error
}

func catalogItemIDs(items []database.CatalogItem) []uint {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func governanceSourceAssetIDs(sources []governanceMovieVersionSource) []uint {
	ids := make([]uint, 0, len(sources))
	for _, source := range sources {
		ids = append(ids, source.AssetID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func governanceSourceFileIDs(sources []governanceMovieVersionSource) []uint {
	ids := make([]uint, 0, len(sources))
	for _, source := range sources {
		ids = append(ids, source.FileID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func uintStringForEvidence(value uint) string {
	if value == 0 {
		return "0"
	}
	const digits = "0123456789"
	buf := make([]byte, 0, 10)
	for value > 0 {
		buf = append(buf, digits[value%10])
		value /= 10
	}
	for left, right := 0, len(buf)-1; left < right; left, right = left+1, right-1 {
		buf[left], buf[right] = buf[right], buf[left]
	}
	return string(buf)
}
