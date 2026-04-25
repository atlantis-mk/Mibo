package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"gorm.io/gorm"
)

type legacySeriesGroup struct {
	key           string
	fallbackKey   string
	libraryID     uint
	seriesTitle   string
	year          *int
	provider      string
	externalID    string
	representative database.MediaItem
	items         []legacySeriesGroupItem
}

type legacySeriesGroupItem struct {
	item  database.MediaItem
	files []database.MediaFile
}

func (s *Service) backfillSeries(ctx context.Context, run database.CatalogMigrationRun) error {
	legacyItems, err := s.listLegacyEpisodeItems(ctx, run)
	if err != nil {
		return err
	}

	groups := map[string]*legacySeriesGroup{}
	fallbackAliases := map[string]string{}
	groupOrder := make([]string, 0)

	for _, legacyItem := range legacyItems {
		identity := classifyLegacySeriesIdentity(legacyItem)
		if identity.providerKey == "" && identity.fallbackKey == "" {
			if _, err := s.recordLegacyBackfillEntry(ctx, run.ID, LegacyBackfillEntry{
				EntryType:         LegacyBackfillEntryTypeConflict,
				LibraryID:         uintPtr(legacyItem.LibraryID),
				LegacyMediaItemID: uintPtr(legacyItem.ID),
				StoragePath:       strings.TrimSpace(legacyItem.SourcePath),
				Title:             strings.TrimSpace(legacyItem.Title),
				Message:           "legacy episode is missing series identity",
				Details:           mustJSON(map[string]any{"reason": "missing_series_identity"}),
			}); err != nil {
				return err
			}
			continue
		}

		if legacyItem.SeasonNumber == nil || legacyItem.EpisodeNumber == nil || *legacyItem.SeasonNumber <= 0 || *legacyItem.EpisodeNumber <= 0 {
			if _, err := s.recordLegacyBackfillEntry(ctx, run.ID, LegacyBackfillEntry{
				EntryType:         LegacyBackfillEntryTypeConflict,
				LibraryID:         uintPtr(legacyItem.LibraryID),
				LegacyMediaItemID: uintPtr(legacyItem.ID),
				StoragePath:       strings.TrimSpace(legacyItem.SourcePath),
				Title:             strings.TrimSpace(legacyItem.Title),
				Message:           "legacy episode is missing season or episode number",
				Details:           mustJSON(map[string]any{"reason": "missing_episode_slot"}),
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
				StoragePath:       strings.TrimSpace(legacyItem.SourcePath),
				Title:             strings.TrimSpace(legacyItem.Title),
				Message:           "legacy episode has no non-deleted playable file",
			}); err != nil {
				return err
			}
			continue
		}

		groupKey := identity.providerKey
		if groupKey == "" {
			groupKey = identity.fallbackKey
			if alias := fallbackAliases[groupKey]; alias != "" {
				groupKey = alias
			}
		} else if identity.fallbackKey != "" {
			if fallbackGroup, ok := groups[identity.fallbackKey]; ok && identity.fallbackKey != groupKey {
				if existingGroup, exists := groups[groupKey]; exists {
					existingGroup.items = append(existingGroup.items, fallbackGroup.items...)
					if betterLegacySeriesRepresentative(fallbackGroup.representative, existingGroup.representative) {
						existingGroup.representative = fallbackGroup.representative
					}
					if existingGroup.seriesTitle == "" {
						existingGroup.seriesTitle = fallbackGroup.seriesTitle
					}
					groupOrder = dedupeLegacySeriesOrder(rekeyLegacySeriesGroup(groupOrder, identity.fallbackKey, groupKey))
				} else {
					groupOrder = rekeyLegacySeriesGroup(groupOrder, identity.fallbackKey, groupKey)
					fallbackGroup.key = groupKey
					groups[groupKey] = fallbackGroup
				}
				delete(groups, identity.fallbackKey)
				fallbackAliases[identity.fallbackKey] = groupKey
			}
			if alias := fallbackAliases[identity.fallbackKey]; alias != "" {
				groupKey = alias
			}
		}
		group, ok := groups[groupKey]
		if !ok {
			group = &legacySeriesGroup{
				key:            groupKey,
				fallbackKey:    identity.fallbackKey,
				libraryID:      legacyItem.LibraryID,
				seriesTitle:    firstNonEmpty(strings.TrimSpace(legacyItem.SeriesTitle), strings.TrimSpace(legacyItem.Title)),
				year:           legacyItem.Year,
				provider:       strings.TrimSpace(legacyItem.MetadataProvider),
				externalID:     strings.TrimSpace(legacyItem.ExternalID),
				representative: legacyItem,
				items:          make([]legacySeriesGroupItem, 0),
			}
			groups[groupKey] = group
			groupOrder = append(groupOrder, groupKey)
		}
		if identity.providerKey != "" {
			group.provider = strings.TrimSpace(legacyItem.MetadataProvider)
			group.externalID = strings.TrimSpace(legacyItem.ExternalID)
			if identity.fallbackKey != "" {
				fallbackAliases[identity.fallbackKey] = groupKey
				group.fallbackKey = identity.fallbackKey
			}
		}
		if betterLegacySeriesRepresentative(legacyItem, group.representative) {
			group.representative = legacyItem
		}
		if title := strings.TrimSpace(legacyItem.SeriesTitle); title != "" {
			group.seriesTitle = title
		}
		group.items = append(group.items, legacySeriesGroupItem{item: legacyItem, files: legacyFiles})
	}

	sort.Strings(groupOrder)
	for _, groupKey := range groupOrder {
		group := groups[groupKey]
		if err := s.processLegacySeriesGroup(ctx, run, group); err != nil {
			return err
		}
	}

	return s.recordLegacyOrphanFiles(ctx, run)
}

func (s *Service) listLegacyEpisodeItems(ctx context.Context, run database.CatalogMigrationRun) ([]database.MediaItem, error) {
	query := s.db.WithContext(ctx).
		Where("type = ?", ItemTypeEpisode).
		Where("deleted_at IS NULL")

	if run.ScopeKind == LegacyBackfillScopeLibrary && run.LibraryID != nil {
		query = query.Where("library_id = ?", *run.LibraryID)
	}

	var items []database.MediaItem
	err := query.Order("library_id asc").Order("id asc").Find(&items).Error
	return items, err
}

func (s *Service) processLegacySeriesGroup(ctx context.Context, run database.CatalogMigrationRun, group *legacySeriesGroup) error {
	seriesItem, err := s.upsertLegacySeriesCatalogItem(ctx, group)
	if err != nil {
		return err
	}
	if err := s.upsertLegacyMovieImages(ctx, seriesItem.ID, group.representative); err != nil {
		return err
	}
	if err := s.upsertLegacySeriesProviderEvidence(ctx, seriesItem.ID, group.representative); err != nil {
		return err
	}

	seasonBuckets := map[int][]legacySeriesGroupItem{}
	seasonOrder := make([]int, 0)
	for _, groupedItem := range group.items {
		seasonNumber := *groupedItem.item.SeasonNumber
		if _, ok := seasonBuckets[seasonNumber]; !ok {
			seasonOrder = append(seasonOrder, seasonNumber)
		}
		seasonBuckets[seasonNumber] = append(seasonBuckets[seasonNumber], groupedItem)
	}
	sort.Ints(seasonOrder)

	for _, seasonNumber := range seasonOrder {
		seasonItem, err := s.upsertLegacySeasonCatalogItem(ctx, seriesItem, seasonNumber, seasonBuckets[seasonNumber])
		if err != nil {
			return err
		}
		if err := s.processLegacySeasonEpisodes(ctx, run, seriesItem, seasonItem, seasonBuckets[seasonNumber]); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) processLegacySeasonEpisodes(ctx context.Context, run database.CatalogMigrationRun, seriesItem database.CatalogItem, seasonItem database.CatalogItem, groupedItems []legacySeriesGroupItem) error {
	episodeBuckets := map[int][]legacySeriesGroupItem{}
	episodeOrder := make([]int, 0)
	for _, groupedItem := range groupedItems {
		episodeNumber := *groupedItem.item.EpisodeNumber
		if _, ok := episodeBuckets[episodeNumber]; !ok {
			episodeOrder = append(episodeOrder, episodeNumber)
		}
		episodeBuckets[episodeNumber] = append(episodeBuckets[episodeNumber], groupedItem)
	}
	sort.Ints(episodeOrder)

	inventorySvc := inventory.NewService(s.db)
	for _, episodeNumber := range episodeOrder {
		slotCandidates := episodeBuckets[episodeNumber]
		sort.SliceStable(slotCandidates, func(i, j int) bool {
			return slotCandidates[i].item.ID < slotCandidates[j].item.ID
		})
		canonical := slotCandidates[0]
		for _, candidate := range slotCandidates[1:] {
			if betterLegacySeriesRepresentative(candidate.item, canonical.item) {
				canonical = candidate
			}
		}

		episodeItem, err := s.upsertLegacyEpisodeCatalogItem(ctx, seasonItem, canonical.item)
		if err != nil {
			return err
		}

		if len(slotCandidates) > 1 {
			for _, candidate := range slotCandidates {
				if candidate.item.ID == canonical.item.ID {
					continue
				}
				// duplicate_episode_candidate entries preserve every non-canonical slot claimant.
				details, err := json.Marshal(map[string]any{
					"canonical_episode_item_id": episodeItem.ID,
					"canonical_legacy_media_item_id": canonical.item.ID,
					"season_number":              *candidate.item.SeasonNumber,
					"episode_number":             *candidate.item.EpisodeNumber,
				})
				if err != nil {
					return err
				}
				if _, err := s.recordLegacyBackfillEntry(ctx, run.ID, LegacyBackfillEntry{
					EntryType:         LegacyBackfillEntryTypeDuplicateEpisodeCandidate,
					LibraryID:         uintPtr(candidate.item.LibraryID),
					LegacyMediaItemID: uintPtr(candidate.item.ID),
					CatalogItemID:     uintPtr(episodeItem.ID),
					StoragePath:       strings.TrimSpace(candidate.item.SourcePath),
					Title:             strings.TrimSpace(candidate.item.Title),
					Message:           "multiple legacy episodes claim the same canonical slot",
					Details:           details,
				}); err != nil {
					return err
				}
			}
		}

		for _, candidate := range slotCandidates {
			var firstAssetID *uint
			var firstInventoryFileID *uint
			for _, legacyFile := range candidate.files {
				inventoryFile, err := inventorySvc.UpsertFile(ctx, inventory.UpsertFileInput{
					LibraryID:         candidate.item.LibraryID,
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

				asset, err := s.upsertLegacyEpisodeAsset(ctx, candidate.item, legacyFile, episodeItem.ID, inventoryFile.ID)
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
					ItemID:       episodeItem.ID,
					Role:         inventory.AssetItemRolePrimary,
					SegmentIndex: 0,
					Source:       legacyBackfillSource,
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
				"series_item_id":  seriesItem.ID,
				"season_item_id":  seasonItem.ID,
				"episode_item_id": episodeItem.ID,
				"file_count":      len(candidate.files),
			})
			if err != nil {
				return err
			}
			if _, err := s.recordLegacyBackfillEntry(ctx, run.ID, LegacyBackfillEntry{
				EntryType:         LegacyBackfillEntryTypeSuccess,
				LibraryID:         uintPtr(candidate.item.LibraryID),
				LegacyMediaItemID: uintPtr(candidate.item.ID),
				CatalogItemID:     uintPtr(episodeItem.ID),
				AssetID:           firstAssetID,
				InventoryFileID:   firstInventoryFileID,
				StoragePath:       strings.TrimSpace(candidate.item.SourcePath),
				Title:             strings.TrimSpace(candidate.item.Title),
				Message:           "migrated legacy episode into catalog hierarchy",
				Details:           detailsJSON,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service) upsertLegacySeriesCatalogItem(ctx context.Context, group *legacySeriesGroup) (database.CatalogItem, error) {
	var item database.CatalogItem
	if group.provider != "" && group.externalID != "" {
		err := s.db.WithContext(ctx).
			Table("catalog_items").
			Joins("JOIN catalog_external_ids ON catalog_external_ids.item_id = catalog_items.id").
			Where("catalog_items.library_id = ? AND catalog_items.type = ? AND catalog_items.deleted_at IS NULL", group.libraryID, ItemTypeSeries).
			Where("catalog_external_ids.provider = ? AND catalog_external_ids.provider_type = ? AND catalog_external_ids.external_id = ?", group.provider, "series", group.externalID).
			Order("catalog_items.id asc").
			First(&item).Error
		if err == nil {
			return s.updateLegacySeriesCatalogItem(ctx, item, group)
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return database.CatalogItem{}, err
		}
	}

	seriesPath := legacySeriesCommonPath(group.items)
	err := s.db.WithContext(ctx).
		Where("library_id = ? AND type = ? AND path = ? AND deleted_at IS NULL", group.libraryID, ItemTypeSeries, seriesPath).
		First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		created, err := s.CreateItem(ctx, CreateItemInput{
			LibraryID:          group.libraryID,
			Type:               ItemTypeSeries,
			Path:               seriesPath,
			SortKey:            group.seriesTitle,
			Title:              group.seriesTitle,
			OriginalTitle:      group.seriesTitle,
			SortTitle:          group.seriesTitle,
			Overview:           group.representative.Overview,
			ReleaseDate:        parseLegacyReleaseDate(group.representative.ReleaseDate),
			Year:               group.year,
			RuntimeSeconds:     group.representative.RuntimeSeconds,
			AvailabilityStatus: AvailabilityAvailable,
			GovernanceStatus:   legacyMatchStatusToGovernanceStatus(group.representative.MatchStatus),
		})
		if err != nil {
			return database.CatalogItem{}, err
		}
		return created, nil
	}
	if err != nil {
		return database.CatalogItem{}, err
	}

	return s.updateLegacySeriesCatalogItem(ctx, item, group)
}

func (s *Service) updateLegacySeriesCatalogItem(ctx context.Context, item database.CatalogItem, group *legacySeriesGroup) (database.CatalogItem, error) {
	updates := map[string]any{
		"path":                legacySeriesCommonPath(group.items),
		"title":               group.seriesTitle,
		"original_title":      group.seriesTitle,
		"sort_title":          group.seriesTitle,
		"sort_key":            group.seriesTitle,
		"overview":            group.representative.Overview,
		"release_date":        parseLegacyReleaseDate(group.representative.ReleaseDate),
		"year":                group.year,
		"runtime_seconds":     group.representative.RuntimeSeconds,
		"availability_status": AvailabilityAvailable,
		"governance_status":   legacyMatchStatusToGovernanceStatus(group.representative.MatchStatus),
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

func (s *Service) upsertLegacySeasonCatalogItem(ctx context.Context, seriesItem database.CatalogItem, seasonNumber int, groupedItems []legacySeriesGroupItem) (database.CatalogItem, error) {
	var item database.CatalogItem
	err := s.db.WithContext(ctx).
		Where("library_id = ? AND type = ? AND parent_id = ? AND index_number = ? AND deleted_at IS NULL", seriesItem.LibraryID, ItemTypeSeason, seriesItem.ID, seasonNumber).
		First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		path := legacySeasonCommonPath(groupedItems)
		if path == "" {
			path = seriesItem.Path
		}
		created, err := s.CreateItem(ctx, CreateItemInput{
			LibraryID:          seriesItem.LibraryID,
			Type:               ItemTypeSeason,
			ParentID:           uintPtr(seriesItem.ID),
			Path:               path,
			SortKey:            fmt.Sprintf("season-%04d", seasonNumber),
			Title:              fmt.Sprintf("Season %d", seasonNumber),
			SortTitle:          fmt.Sprintf("Season %d", seasonNumber),
			IndexNumber:        intPtr(seasonNumber),
			AvailabilityStatus: AvailabilityAvailable,
			GovernanceStatus:   legacyMatchStatusToGovernanceStatus(bestLegacySeriesItem(groupedItems).MatchStatus),
		})
		if err != nil {
			return database.CatalogItem{}, err
		}
		return created, nil
	}
	if err != nil {
		return database.CatalogItem{}, err
	}

	if err := s.db.WithContext(ctx).Model(&database.CatalogItem{}).Where("id = ?", item.ID).Updates(map[string]any{
		"path":                firstNonEmpty(legacySeasonCommonPath(groupedItems), seriesItem.Path),
		"title":               fmt.Sprintf("Season %d", seasonNumber),
		"sort_title":          fmt.Sprintf("Season %d", seasonNumber),
		"sort_key":            fmt.Sprintf("season-%04d", seasonNumber),
		"availability_status": AvailabilityAvailable,
		"governance_status":   legacyMatchStatusToGovernanceStatus(bestLegacySeriesItem(groupedItems).MatchStatus),
		"updated_at":          time.Now().UTC(),
	}).Error; err != nil {
		return database.CatalogItem{}, err
	}
	if err := s.db.WithContext(ctx).First(&item, item.ID).Error; err != nil {
		return database.CatalogItem{}, err
	}
	return item, nil
}

func (s *Service) upsertLegacyEpisodeCatalogItem(ctx context.Context, seasonItem database.CatalogItem, legacyItem database.MediaItem) (database.CatalogItem, error) {
	var item database.CatalogItem
	err := s.db.WithContext(ctx).
		Where("library_id = ? AND type = ? AND parent_id = ? AND parent_index_number = ? AND index_number = ? AND deleted_at IS NULL", legacyItem.LibraryID, ItemTypeEpisode, seasonItem.ID, *legacyItem.SeasonNumber, *legacyItem.EpisodeNumber).
		First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		created, err := s.CreateItem(ctx, CreateItemInput{
			LibraryID:          legacyItem.LibraryID,
			Type:               ItemTypeEpisode,
			ParentID:           uintPtr(seasonItem.ID),
			Path:               strings.TrimSpace(legacyItem.SourcePath),
			SortKey:            fmt.Sprintf("episode-%04d", *legacyItem.EpisodeNumber),
			DisplayOrder:       DisplayOrderAired,
			IndexNumber:        intPtr(*legacyItem.EpisodeNumber),
			ParentIndexNumber:  intPtr(*legacyItem.SeasonNumber),
			Title:              strings.TrimSpace(legacyItem.Title),
			OriginalTitle:      strings.TrimSpace(legacyItem.Title),
			SortTitle:          strings.TrimSpace(legacyItem.Title),
			Overview:           legacyItem.Overview,
			ReleaseDate:        parseLegacyReleaseDate(legacyItem.ReleaseDate),
			Year:               legacyItem.Year,
			RuntimeSeconds:     legacyItem.RuntimeSeconds,
			AvailabilityStatus: AvailabilityAvailable,
			GovernanceStatus:   legacyMatchStatusToGovernanceStatus(legacyItem.MatchStatus),
		})
		if err != nil {
			return database.CatalogItem{}, err
		}
		return created, nil
	}
	if err != nil {
		return database.CatalogItem{}, err
	}

	if err := s.db.WithContext(ctx).Model(&database.CatalogItem{}).Where("id = ?", item.ID).Updates(map[string]any{
		"path":                strings.TrimSpace(legacyItem.SourcePath),
		"title":               strings.TrimSpace(legacyItem.Title),
		"original_title":      strings.TrimSpace(legacyItem.Title),
		"sort_title":          strings.TrimSpace(legacyItem.Title),
		"sort_key":            fmt.Sprintf("episode-%04d", *legacyItem.EpisodeNumber),
		"overview":            legacyItem.Overview,
		"release_date":        parseLegacyReleaseDate(legacyItem.ReleaseDate),
		"year":                legacyItem.Year,
		"runtime_seconds":     legacyItem.RuntimeSeconds,
		"availability_status": AvailabilityAvailable,
		"governance_status":   legacyMatchStatusToGovernanceStatus(legacyItem.MatchStatus),
		"updated_at":          time.Now().UTC(),
	}).Error; err != nil {
		return database.CatalogItem{}, err
	}
	if err := s.db.WithContext(ctx).First(&item, item.ID).Error; err != nil {
		return database.CatalogItem{}, err
	}
	return item, nil
}

func (s *Service) upsertLegacySeriesProviderEvidence(ctx context.Context, itemID uint, legacyItem database.MediaItem) error {
	provider := strings.TrimSpace(legacyItem.MetadataProvider)
	externalID := strings.TrimSpace(legacyItem.ExternalID)
	if provider == "" || externalID == "" || !isLegacySeriesProviderID(externalID) {
		return nil
	}

	if _, err := s.SetExternalID(ctx, ExternalIDInput{
		ItemID:       itemID,
		Provider:     provider,
		ProviderType: "series",
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

func (s *Service) upsertLegacyEpisodeAsset(ctx context.Context, legacyItem database.MediaItem, legacyFile database.MediaFile, episodeItemID uint, inventoryFileID uint) (database.MediaAsset, error) {
	var asset database.MediaAsset
	err := s.db.WithContext(ctx).
		Joins("JOIN asset_items ON asset_items.asset_id = media_assets.id").
		Joins("JOIN asset_files ON asset_files.asset_id = media_assets.id").
		Where("media_assets.library_id = ? AND media_assets.asset_type = ?", legacyItem.LibraryID, inventory.AssetTypeMain).
		Where("media_assets.deleted_at IS NULL").
		Where("asset_items.item_id = ? AND asset_items.role = ? AND asset_items.segment_index = ?", episodeItemID, inventory.AssetItemRolePrimary, 0).
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
				"legacy_media_item_id": legacyItem.ID,
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

	if err := s.db.WithContext(ctx).Model(&database.MediaAsset{}).Where("id = ?", asset.ID).Updates(map[string]any{
		"display_name":           strings.TrimSpace(legacyItem.Title),
		"duration_seconds":       legacyFile.DurationSeconds,
		"status":                 inventory.AssetStatusAvailable,
		"probe_status":           legacyFileProbeStatus(legacyFile),
		"technical_summary_json": string(mustJSON(map[string]any{"legacy_media_item_id": legacyItem.ID, "legacy_media_file_id": legacyFile.ID, "storage_path": strings.TrimSpace(legacyFile.StoragePath), "container": strings.TrimSpace(legacyFile.Container), "size_bytes": legacyFile.SizeBytes})),
		"updated_at":             time.Now().UTC(),
	}).Error; err != nil {
		return database.MediaAsset{}, err
	}
	if err := s.db.WithContext(ctx).First(&asset, asset.ID).Error; err != nil {
		return database.MediaAsset{}, err
	}
	return asset, nil
}

func (s *Service) recordLegacyOrphanFiles(ctx context.Context, run database.CatalogMigrationRun) error {
	query := s.db.WithContext(ctx).
		Table("media_files").
		Select("media_files.*").
		Joins("LEFT JOIN media_items ON media_items.id = media_files.media_item_id AND media_items.deleted_at IS NULL").
		Where("media_files.deleted_at IS NULL").
		Where("media_items.id IS NULL")

	if run.ScopeKind == LegacyBackfillScopeLibrary && run.LibraryID != nil {
		query = query.Where("media_files.library_id = ?", *run.LibraryID)
	}

	var files []database.MediaFile
	if err := query.Order("media_files.library_id asc").Order("media_files.id asc").Find(&files).Error; err != nil {
		return err
	}

	for _, file := range files {
		// orphan_file entries surface active legacy files with no active owning media item.
		details, err := json.Marshal(map[string]any{
			"provider_name":      strings.TrimSpace(file.ProviderName),
			"stable_identity_key": strings.TrimSpace(file.StableIdentityKey),
		})
		if err != nil {
			return err
		}
		if _, err := s.recordLegacyBackfillEntry(ctx, run.ID, LegacyBackfillEntry{
			EntryType:         LegacyBackfillEntryTypeOrphanFile,
			LibraryID:         uintPtr(file.LibraryID),
			LegacyMediaFileID: uintPtr(file.ID),
			StoragePath:       strings.TrimSpace(file.StoragePath),
			Message:           "legacy media file has no active owning media item",
			Details:           details,
		}); err != nil {
			return err
		}
	}

	return nil
}

type legacySeriesIdentity struct {
	providerKey string
	fallbackKey string
}

func classifyLegacySeriesIdentity(item database.MediaItem) legacySeriesIdentity {
	provider := strings.TrimSpace(item.MetadataProvider)
	externalID := strings.TrimSpace(item.ExternalID)
	seriesTitle := strings.TrimSpace(item.SeriesTitle)
	identity := legacySeriesIdentity{}
	if provider != "" && externalID != "" && isLegacySeriesProviderID(externalID) {
		identity.providerKey = fmt.Sprintf("provider:%d:%s:%s", item.LibraryID, strings.ToLower(provider), strings.ToLower(externalID))
	}
	if seriesTitle != "" {
		year := ""
		if item.Year != nil {
			year = strconv.Itoa(*item.Year)
		}
		identity.fallbackKey = fmt.Sprintf("title:%d:%s:%s", item.LibraryID, normalizeLegacySeriesTitle(seriesTitle), year)
	}
	return identity
}

func isLegacySeriesProviderID(externalID string) bool {
	lower := strings.ToLower(strings.TrimSpace(externalID))
	return strings.HasPrefix(lower, "tv:") || strings.HasPrefix(lower, "series:")
}

func normalizeLegacySeriesTitle(title string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(title)), " "))
}

func betterLegacySeriesRepresentative(candidate, current database.MediaItem) bool {
	if strings.TrimSpace(candidate.MetadataProvider) != "" && strings.TrimSpace(current.MetadataProvider) == "" && isLegacySeriesProviderID(candidate.ExternalID) {
		return true
	}
	if strings.TrimSpace(candidate.PosterURL) != "" && strings.TrimSpace(current.PosterURL) == "" {
		return true
	}
	if strings.TrimSpace(candidate.BackdropURL) != "" && strings.TrimSpace(current.BackdropURL) == "" {
		return true
	}
	if strings.TrimSpace(candidate.LogoURL) != "" && strings.TrimSpace(current.LogoURL) == "" {
		return true
	}
	if strings.TrimSpace(candidate.Overview) != "" && strings.TrimSpace(current.Overview) == "" {
		return true
	}
	if current.Year == nil && candidate.Year != nil {
		return true
	}
	return candidate.ID < current.ID
}

func legacySeriesCommonPath(items []legacySeriesGroupItem) string {
	paths := make([]string, 0, len(items))
	for _, item := range items {
		if path := strings.TrimSpace(item.item.SourcePath); path != "" {
			paths = append(paths, path)
		}
	}
	return commonDirectoryPrefix(paths)
}

func legacySeasonCommonPath(items []legacySeriesGroupItem) string {
	paths := make([]string, 0, len(items))
	for _, item := range items {
		if path := strings.TrimSpace(item.item.SourcePath); path != "" {
			paths = append(paths, path)
		}
	}
	return commonDirectoryPrefix(paths)
}

func commonDirectoryPrefix(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	dirs := make([][]string, 0, len(paths))
	for _, value := range paths {
		cleaned := filepath.Clean(filepath.Dir(strings.TrimSpace(value)))
		if cleaned == "." || cleaned == "" {
			cleaned = filepath.Clean(strings.TrimSpace(value))
		}
		parts := strings.Split(filepath.ToSlash(cleaned), "/")
		dirs = append(dirs, parts)
	}
	common := dirs[0]
	for _, parts := range dirs[1:] {
		limit := minInt(len(common), len(parts))
		idx := 0
		for idx < limit && common[idx] == parts[idx] {
			idx++
		}
		common = common[:idx]
		if len(common) == 0 {
			break
		}
	}
	if len(common) == 0 {
		return filepath.ToSlash(filepath.Clean(filepath.Dir(strings.TrimSpace(paths[0]))))
	}
	joined := strings.Join(common, "/")
	if strings.HasPrefix(filepath.ToSlash(strings.TrimSpace(paths[0])), "/") && !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	return filepath.ToSlash(filepath.Clean(joined))
}

func bestLegacySeriesItem(items []legacySeriesGroupItem) database.MediaItem {
	best := items[0].item
	for _, item := range items[1:] {
		if betterLegacySeriesRepresentative(item.item, best) {
			best = item.item
		}
	}
	return best
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func intPtr(value int) *int {
	return &value
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func rekeyLegacySeriesGroup(order []string, fromKey string, toKey string) []string {
	for idx, key := range order {
		if key == fromKey {
			order[idx] = toKey
			break
		}
	}
	return order
}

func dedupeLegacySeriesOrder(order []string) []string {
	filtered := order[:0]
	seen := map[string]struct{}{}
	for _, key := range order {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, key)
	}
	return filtered
}
