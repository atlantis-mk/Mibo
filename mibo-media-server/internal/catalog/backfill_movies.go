package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"gorm.io/gorm"
)

const legacyBackfillSource = "legacy_backfill"

func (s *Service) backfillMovies(ctx context.Context, run database.CatalogMigrationRun) error {
	legacyItems, err := s.listLegacyMovieItems(ctx, run)
	if err != nil {
		return err
	}

	for _, legacyItem := range legacyItems {
		if strings.TrimSpace(legacyItem.SourcePath) == "" {
			if _, err := s.recordLegacyBackfillEntry(ctx, run.ID, LegacyBackfillEntry{
				EntryType:         LegacyBackfillEntryTypeSkipped,
				LibraryID:         uintPtr(legacyItem.LibraryID),
				LegacyMediaItemID: uintPtr(legacyItem.ID),
				Title:             legacyItem.Title,
				Message:           "legacy movie is missing source_path",
			}); err != nil {
				return err
			}
			continue
		}

		legacyFiles, err := s.listLegacyMovieFiles(ctx, legacyItem.ID)
		if err != nil {
			return err
		}
		if len(legacyFiles) == 0 {
			if _, err := s.recordLegacyBackfillEntry(ctx, run.ID, LegacyBackfillEntry{
				EntryType:         LegacyBackfillEntryTypeSkipped,
				LibraryID:         uintPtr(legacyItem.LibraryID),
				LegacyMediaItemID: uintPtr(legacyItem.ID),
				StoragePath:       legacyItem.SourcePath,
				Title:             legacyItem.Title,
				Message:           "legacy movie has no non-deleted playable file",
			}); err != nil {
				return err
			}
			continue
		}

		catalogItem, err := s.upsertLegacyMovieCatalogItem(ctx, legacyItem)
		if err != nil {
			return err
		}
		if err := s.upsertLegacyMovieImages(ctx, catalogItem.ID, legacyItem); err != nil {
			return err
		}
		if err := s.upsertLegacyMovieProviderEvidence(ctx, catalogItem.ID, legacyItem); err != nil {
			return err
		}

		var firstAssetID *uint
		var firstInventoryFileID *uint
		inventorySvc := inventory.NewService(s.db)
		for _, legacyFile := range legacyFiles {
			inventoryFile, err := inventorySvc.UpsertFile(ctx, inventory.UpsertFileInput{
				LibraryID:         legacyItem.LibraryID,
				StorageProvider:   legacyMovieStorageProvider(legacyFile),
				StoragePath:       strings.TrimSpace(legacyFile.StoragePath),
				StableIdentityKey: strings.TrimSpace(legacyFile.StableIdentityKey),
				HashesJSON:        strings.TrimSpace(legacyFile.ProviderHashesJSON),
				SizeBytes:         legacyFile.SizeBytes,
				ModifiedAt:        legacyFile.LastModifiedAt,
				Container:         strings.TrimSpace(legacyFile.Container),
				Status:            inventory.FileStatusAvailable,
			})
			if err != nil {
				return err
			}

			asset, err := s.upsertLegacyMovieAsset(ctx, legacyItem, legacyFile, catalogItem.ID, inventoryFile.ID)
			if err != nil {
				return err
			}
			if _, err := inventorySvc.LinkAssetToFile(ctx, inventory.LinkAssetFileInput{
				AssetID:   asset.ID,
				FileID:    inventoryFile.ID,
				Role:      inventory.FileRoleSource,
				PartIndex: 0,
			}); err != nil {
				return err
			}
			if _, err := inventorySvc.LinkAssetToItem(ctx, inventory.LinkAssetItemInput{
				AssetID:      asset.ID,
				ItemID:       catalogItem.ID,
				Role:         inventory.AssetItemRolePrimary,
				SegmentIndex: 0,
				Source:       "legacy_backfill",
			}); err != nil {
				return err
			}

			if firstAssetID == nil {
				firstAssetID = uintPtr(asset.ID)
			}
			if firstInventoryFileID == nil {
				firstInventoryFileID = uintPtr(inventoryFile.ID)
			}
		}

		detailsJSON, err := json.Marshal(map[string]any{
			"legacy_media_item_id": legacyItem.ID,
			"file_count":           len(legacyFiles),
			"match_status":         strings.TrimSpace(legacyItem.MatchStatus),
		})
		if err != nil {
			return err
		}
		if _, err := s.recordLegacyBackfillEntry(ctx, run.ID, LegacyBackfillEntry{
			EntryType:         LegacyBackfillEntryTypeSuccess,
			LibraryID:         uintPtr(legacyItem.LibraryID),
			LegacyMediaItemID: uintPtr(legacyItem.ID),
			CatalogItemID:     uintPtr(catalogItem.ID),
			AssetID:           firstAssetID,
			InventoryFileID:   firstInventoryFileID,
			StoragePath:       legacyItem.SourcePath,
			Title:             legacyItem.Title,
			Message:           "migrated legacy movie into catalog kernel",
			Details:           detailsJSON,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) listLegacyMovieItems(ctx context.Context, run database.CatalogMigrationRun) ([]database.MediaItem, error) {
	query := s.db.WithContext(ctx).
		Where("type = ?", ItemTypeMovie).
		Where("deleted_at IS NULL")

	if run.ScopeKind == LegacyBackfillScopeLibrary && run.LibraryID != nil {
		query = query.Where("library_id = ?", *run.LibraryID)
	}

	var items []database.MediaItem
	err := query.Order("library_id asc").Order("id asc").Find(&items).Error
	return items, err
}

func (s *Service) listLegacyMovieFiles(ctx context.Context, legacyItemID uint) ([]database.MediaFile, error) {
	var files []database.MediaFile
	err := s.db.WithContext(ctx).
		Where("media_item_id = ?", legacyItemID).
		Where("deleted_at IS NULL").
		Order("id asc").
		Find(&files).Error
	return files, err
}

func (s *Service) upsertLegacyMovieCatalogItem(ctx context.Context, legacyItem database.MediaItem) (database.CatalogItem, error) {
	var item database.CatalogItem
	err := s.db.WithContext(ctx).
		Where("library_id = ? AND type = ? AND path = ? AND deleted_at IS NULL", legacyItem.LibraryID, ItemTypeMovie, strings.TrimSpace(legacyItem.SourcePath)).
		First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.CreateItem(ctx, CreateItemInput{
			LibraryID:          legacyItem.LibraryID,
			Type:               ItemTypeMovie,
			Path:               strings.TrimSpace(legacyItem.SourcePath),
			SortKey:            strings.TrimSpace(legacyItem.Title),
			Title:              strings.TrimSpace(legacyItem.Title),
			OriginalTitle:      strings.TrimSpace(legacyItem.OriginalTitle),
			SortTitle:          strings.TrimSpace(legacyItem.Title),
			Overview:           legacyItem.Overview,
			ReleaseDate:        parseLegacyReleaseDate(legacyItem.ReleaseDate),
			Year:               legacyItem.Year,
			RuntimeSeconds:     legacyItem.RuntimeSeconds,
			CommunityRating:    legacyItem.VoteAverage,
			AvailabilityStatus: AvailabilityAvailable,
			GovernanceStatus:   legacyMatchStatusToGovernanceStatus(legacyItem.MatchStatus),
		})
	}
	if err != nil {
		return database.CatalogItem{}, err
	}

	updates := map[string]any{
		"title":               strings.TrimSpace(legacyItem.Title),
		"original_title":      strings.TrimSpace(legacyItem.OriginalTitle),
		"sort_title":          strings.TrimSpace(legacyItem.Title),
		"sort_key":            strings.TrimSpace(legacyItem.Title),
		"overview":            legacyItem.Overview,
		"release_date":        parseLegacyReleaseDate(legacyItem.ReleaseDate),
		"year":                legacyItem.Year,
		"runtime_seconds":     legacyItem.RuntimeSeconds,
		"community_rating":    legacyItem.VoteAverage,
		"availability_status": AvailabilityAvailable,
		"governance_status":   legacyMatchStatusToGovernanceStatus(legacyItem.MatchStatus),
		"updated_at":          time.Now().UTC(),
	}
	if err := s.db.WithContext(ctx).Model(&database.CatalogItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
		return database.CatalogItem{}, err
	}
	if err := s.db.WithContext(ctx).First(&item, item.ID).Error; err != nil {
		return database.CatalogItem{}, err
	}
	return item, nil
}

func (s *Service) upsertLegacyMovieImages(ctx context.Context, itemID uint, legacyItem database.MediaItem) error {
	imageInputs := []struct {
		imageType string
		url       string
	}{
		{imageType: "poster", url: legacyItem.PosterURL},
		{imageType: "backdrop", url: legacyItem.BackdropURL},
		{imageType: "logo", url: legacyItem.LogoURL},
	}

	for _, imageInput := range imageInputs {
		trimmedURL := strings.TrimSpace(imageInput.url)
		if trimmedURL == "" {
			continue
		}

		var existing []database.ItemImage
		if err := s.db.WithContext(ctx).
			Where("item_id = ? AND image_type = ?", itemID, imageInput.imageType).
			Order("id asc").
			Find(&existing).Error; err != nil {
			return err
		}
		if len(existing) == 0 {
			if err := s.db.WithContext(ctx).Create(&database.ItemImage{
				ItemID:     itemID,
				ImageType:  imageInput.imageType,
				URL:        trimmedURL,
				IsSelected: true,
				SortOrder:  0,
			}).Error; err != nil {
				return err
			}
			continue
		}
		if err := s.db.WithContext(ctx).Model(&database.ItemImage{}).Where("id = ?", existing[0].ID).Updates(map[string]any{
			"url":         trimmedURL,
			"is_selected": true,
			"sort_order":  0,
			"updated_at":  time.Now().UTC(),
		}).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) upsertLegacyMovieProviderEvidence(ctx context.Context, itemID uint, legacyItem database.MediaItem) error {
	provider := strings.TrimSpace(legacyItem.MetadataProvider)
	externalID := strings.TrimSpace(legacyItem.ExternalID)
	if provider == "" || externalID == "" {
		return nil
	}

	if _, err := s.SetExternalID(ctx, ExternalIDInput{
		ItemID:       itemID,
		Provider:     provider,
		ProviderType: "movie",
		ExternalID:   externalID,
		IsPrimary:    true,
		Source:       legacyBackfillSource,
		Confidence:   legacyItem.MetadataConfidence,
	}); err != nil {
		return err
	}

	payloadJSON, err := json.Marshal(map[string]any{
		"legacy_media_item_id": legacyItem.ID,
		"match_status":         strings.TrimSpace(legacyItem.MatchStatus),
		"confidence":           legacyItem.MetadataConfidence,
	})
	if err != nil {
		return err
	}

	var existing database.MetadataSource
	err = s.db.WithContext(ctx).
		Where("item_id = ? AND source_type = ? AND source_name = ? AND external_id = ?", itemID, SourceTypeProvider, provider, externalID).
		First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		_, err = s.RecordMetadataSource(ctx, MetadataSourceInput{
			ItemID:      itemID,
			SourceType:  SourceTypeProvider,
			SourceName:  provider,
			ExternalID:  externalID,
			PayloadJSON: string(payloadJSON),
			Confidence:  legacyItem.MetadataConfidence,
			FetchedAt:   time.Now().UTC(),
		})
		return err
	}
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).Model(&database.MetadataSource{}).Where("id = ?", existing.ID).Updates(map[string]any{
		"payload_json": string(payloadJSON),
		"confidence":   legacyItem.MetadataConfidence,
		"fetched_at":   time.Now().UTC(),
		"updated_at":   time.Now().UTC(),
	}).Error
}

func (s *Service) upsertLegacyMovieAsset(ctx context.Context, legacyItem database.MediaItem, legacyFile database.MediaFile, catalogItemID uint, inventoryFileID uint) (database.MediaAsset, error) {
	var asset database.MediaAsset
	err := s.db.WithContext(ctx).
		Joins("JOIN asset_items ON asset_items.asset_id = media_assets.id").
		Joins("JOIN asset_files ON asset_files.asset_id = media_assets.id").
		Where("media_assets.library_id = ? AND media_assets.asset_type = ?", legacyItem.LibraryID, inventory.AssetTypeMain).
		Where("media_assets.deleted_at IS NULL").
		Where("asset_items.item_id = ? AND asset_items.role = ? AND asset_items.segment_index = ?", catalogItemID, inventory.AssetItemRolePrimary, 0).
		Where("asset_files.file_id = ? AND asset_files.role = ? AND asset_files.part_index = ?", inventoryFileID, inventory.FileRoleSource, 0).
		Order("media_assets.id asc").
		First(&asset).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		inventorySvc := inventory.NewService(s.db)
		return inventorySvc.CreateAsset(ctx, inventory.CreateAssetInput{
			LibraryID:       legacyItem.LibraryID,
			AssetType:       inventory.AssetTypeMain,
			DisplayName:     strings.TrimSpace(legacyItem.Title),
			DurationSeconds: legacyFile.DurationSeconds,
			Status:          inventory.AssetStatusAvailable,
			ProbeStatus:     legacyFileProbeStatus(legacyFile),
			TechnicalSummaryJSON: string(mustJSON(map[string]any{
				"legacy_media_file_id": legacyFile.ID,
				"storage_path":         strings.TrimSpace(legacyFile.StoragePath),
				"container":            strings.TrimSpace(legacyFile.Container),
				"size_bytes":           legacyFile.SizeBytes,
			})),
		})
	}
	if err != nil {
		return database.MediaAsset{}, err
	}

	updates := map[string]any{
		"display_name":           strings.TrimSpace(legacyItem.Title),
		"duration_seconds":       legacyFile.DurationSeconds,
		"status":                 inventory.AssetStatusAvailable,
		"probe_status":           legacyFileProbeStatus(legacyFile),
		"technical_summary_json": string(mustJSON(map[string]any{"legacy_media_file_id": legacyFile.ID, "storage_path": strings.TrimSpace(legacyFile.StoragePath), "container": strings.TrimSpace(legacyFile.Container), "size_bytes": legacyFile.SizeBytes})),
		"updated_at":             time.Now().UTC(),
	}
	if err := s.db.WithContext(ctx).Model(&database.MediaAsset{}).Where("id = ?", asset.ID).Updates(updates).Error; err != nil {
		return database.MediaAsset{}, err
	}
	if err := s.db.WithContext(ctx).First(&asset, asset.ID).Error; err != nil {
		return database.MediaAsset{}, err
	}
	return asset, nil
}

func legacyMovieStorageProvider(file database.MediaFile) string {
	provider := strings.TrimSpace(file.ProviderName)
	if provider == "" {
		return "legacy"
	}
	return provider
}

func legacyFileProbeStatus(file database.MediaFile) string {
	status := strings.TrimSpace(file.ProbeStatus)
	if status == "" {
		return "pending"
	}
	return status
}

func legacyMatchStatusToGovernanceStatus(matchStatus string) string {
	switch strings.TrimSpace(matchStatus) {
	case "matched":
		return GovernanceMatched
	case "needs_review":
		return GovernanceNeedsReview
	case "unmatched", "skipped":
		return GovernanceUnmatched
	default:
		return GovernancePending
	}
}

func parseLegacyReleaseDate(value string) *time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339} {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			parsed = parsed.UTC()
			return &parsed
		}
	}
	return nil
}

func mustJSON(value any) []byte {
	payload, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return payload
}

func uintPtr(value uint) *uint {
	return &value
}
