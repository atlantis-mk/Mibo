package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func (s *Service) MatchCatalogItem(ctx context.Context, itemID uint) error {
	tmdbCfg, err := s.tmdbConfig(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(tmdbCfg.APIKey) == "" {
		return fmt.Errorf("tmdb 未配置，无法匹配 catalog 元数据")
	}

	target, err := s.resolveCatalogMatchTarget(ctx, itemID)
	if err != nil {
		return err
	}

	mediaType := catalogTMDBMediaType(target.Type)
	searchItem := catalogItemToSearchItem(target)
	searchMatch, err := s.searchBestMatch(ctx, tmdbCfg, searchItem, mediaType)
	if err != nil {
		return err
	}
	if searchMatch == nil {
		return s.applyCatalogGovernanceStatus(ctx, target.ID, catalog.GovernanceUnmatched)
	}

	detail, err := s.fetchDetail(ctx, tmdbCfg, mediaType, searchMatch.result.ID)
	if err != nil {
		return err
	}

	return s.applyCatalogDetail(ctx, target, tmdbCfg, mediaType, detail, searchMatch.confidence)
}

func (s *Service) SearchCatalogCandidates(ctx context.Context, itemID uint, input ManualSearchInput) ([]SearchCandidate, error) {
	tmdbCfg, err := s.tmdbConfig(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(tmdbCfg.APIKey) == "" {
		return nil, fmt.Errorf("tmdb 未配置，无法搜索 catalog 元数据")
	}

	target, err := s.resolveCatalogMatchTarget(ctx, itemID)
	if err != nil {
		return nil, err
	}
	mediaType := catalogTMDBMediaType(target.Type)
	searchItem := catalogItemToSearchItem(target)

	if tmdbID := strings.TrimSpace(input.TMDBID); tmdbID != "" {
		id, err := strconv.Atoi(tmdbID)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("tmdb_id 必须是正整数")
		}
		detail, err := s.fetchDetail(ctx, tmdbCfg, mediaType, id)
		if err != nil {
			return nil, err
		}
		candidate := detailToCandidate(tmdbCfg, mediaType, detail, 1)
		candidate.MatchedQuery = "TMDB ID " + tmdbID
		candidate.ReasonSummary = "通过 TMDB ID 精确定位"
		return []SearchCandidate{candidate}, nil
	}

	queries := buildManualSearchQueries(input, searchItem, mediaType)
	if len(queries) == 0 {
		return nil, fmt.Errorf("标题不能为空")
	}
	return s.searchCandidates(ctx, tmdbCfg, mediaType, queries, searchItem)
}

func (s *Service) ApplyCatalogCandidate(ctx context.Context, itemID uint, input ApplyCandidateInput) error {
	tmdbCfg, err := s.tmdbConfig(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(tmdbCfg.APIKey) == "" {
		return fmt.Errorf("tmdb 未配置，无法应用 catalog 元数据")
	}

	target, err := s.resolveCatalogMatchTarget(ctx, itemID)
	if err != nil {
		return err
	}
	mediaType, id, err := parseExternalID(input.ExternalID)
	if err != nil {
		return err
	}
	detail, err := s.fetchDetail(ctx, tmdbCfg, mediaType, id)
	if err != nil {
		return err
	}
	return s.applyCatalogDetail(ctx, target, tmdbCfg, mediaType, detail, 1)
}

func (s *Service) RefetchCatalogItem(ctx context.Context, itemID uint) error {
	tmdbCfg, err := s.tmdbConfig(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(tmdbCfg.APIKey) == "" {
		return fmt.Errorf("tmdb 未配置，无法重抓 catalog 元数据")
	}

	target, err := s.resolveCatalogMatchTarget(ctx, itemID)
	if err != nil {
		return err
	}

	mediaType := catalogTMDBMediaType(target.Type)
	externalID, confidence, err := s.loadCatalogTMDBIdentity(ctx, target.ID, mediaType)
	if err != nil {
		return err
	}
	_, tmdbID, err := parseExternalID(externalID)
	if err != nil {
		return err
	}

	detail, err := s.fetchDetail(ctx, tmdbCfg, mediaType, tmdbID)
	if err != nil {
		return err
	}
	return s.applyCatalogDetail(ctx, target, tmdbCfg, mediaType, detail, confidence)
}

func (s *Service) resolveCatalogMatchTarget(ctx context.Context, itemID uint) (database.CatalogItem, error) {
	var item database.CatalogItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", itemID).First(&item).Error; err != nil {
		return database.CatalogItem{}, err
	}

	if item.Type == catalog.ItemTypeSeason || item.Type == catalog.ItemTypeEpisode {
		if item.RootID == nil || *item.RootID == 0 {
			return database.CatalogItem{}, fmt.Errorf("catalog item %d 缺少 root_id，无法回溯到 series", item.ID)
		}
		var root database.CatalogItem
		if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", *item.RootID).First(&root).Error; err != nil {
			return database.CatalogItem{}, err
		}
		return root, nil
	}

	return item, nil
}

func (s *Service) loadCatalogTMDBIdentity(ctx context.Context, itemID uint, mediaType string) (string, float64, error) {
	var identity database.CatalogExternalID
	if err := s.db.WithContext(ctx).
		Where("item_id = ? AND provider = ? AND provider_type = ?", itemID, "tmdb", mediaType).
		Order("is_primary desc, id asc").
		First(&identity).Error; err != nil {
		return "", 0, fmt.Errorf("当前 catalog 条目没有可重抓的 TMDB 匹配结果: %w", err)
	}

	confidence := 1.0
	if identity.Confidence != nil && *identity.Confidence > 0 {
		confidence = *identity.Confidence
	}
	return strings.TrimSpace(identity.ExternalID), confidence, nil
}

func (s *Service) applyCatalogDetail(ctx context.Context, item database.CatalogItem, tmdbCfg config.TMDBConfig, mediaType string, detail detailResponse, confidence float64) error {
	status := catalog.GovernanceMatched
	if confidence < 0.85 {
		status = catalog.GovernanceNeedsReview
	}

	title := strings.TrimSpace(detail.Title)
	originalTitle := strings.TrimSpace(detail.OriginalTitle)
	releaseDate := strings.TrimSpace(detail.ReleaseDate)
	if mediaType == "tv" {
		title = strings.TrimSpace(detail.Name)
		originalTitle = strings.TrimSpace(detail.OriginalName)
		releaseDate = strings.TrimSpace(detail.FirstAirDate)
	}
	year := parseYear(releaseDate)
	runtimeSeconds := runtimeFromDetail(detail)
	externalID := fmt.Sprintf("%s:%d", mediaType, detail.ID)

	catalogSvc := catalog.NewService(s.db)
	if title != "" {
		if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: item.ID, FieldKey: "title", Value: title}); err != nil {
			return err
		}
		if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: item.ID, FieldKey: "sort_title", Value: title}); err != nil {
			return err
		}
	}
	if originalTitle != "" {
		if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: item.ID, FieldKey: "original_title", Value: originalTitle}); err != nil {
			return err
		}
	}
	if overview := strings.TrimSpace(detail.Overview); overview != "" {
		if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: item.ID, FieldKey: "overview", Value: overview}); err != nil {
			return err
		}
	}
	if year != nil {
		if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: item.ID, FieldKey: "year", Value: *year}); err != nil {
			return err
		}
	}
	if runtimeSeconds != nil {
		if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: item.ID, FieldKey: "runtime_seconds", Value: *runtimeSeconds}); err != nil {
			return err
		}
	}
	if err := s.applyCatalogGovernanceStatus(ctx, item.ID, status); err != nil {
		return err
	}
	if _, err := catalogSvc.SetExternalID(ctx, catalog.ExternalIDInput{
		ItemID:       item.ID,
		Provider:     "tmdb",
		ProviderType: mediaType,
		ExternalID:   externalID,
		IsPrimary:    true,
		Source:       "metadata_match",
		Confidence:   &confidence,
	}); err != nil {
		return err
	}

	payloadJSON, err := json.Marshal(map[string]any{
		"media_type":    mediaType,
		"external_id":   externalID,
		"confidence":    confidence,
		"matched_title": title,
	})
	if err != nil {
		return err
	}
	source, err := catalogSvc.RecordMetadataSource(ctx, catalog.MetadataSourceInput{
		ItemID:      item.ID,
		SourceType:  catalog.SourceTypeProvider,
		SourceName:  "tmdb",
		ExternalID:  externalID,
		PayloadJSON: string(payloadJSON),
		Confidence:  &confidence,
	})
	if err != nil {
		return err
	}
	if err := s.syncCatalogDetailImages(ctx, item.ID, tmdbCfg, detail, &source.ID); err != nil {
		return err
	}
	if mediaType == "tv" && item.Type == catalog.ItemTypeSeries {
		if err := s.syncCatalogSeriesHierarchy(ctx, item, tmdbCfg, detail.ID, status, confidence, detail.Seasons); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) applyCatalogGovernanceStatus(ctx context.Context, itemID uint, status string) error {
	_, _, err := catalog.NewService(s.db).ApplyField(ctx, catalog.ApplyFieldInput{
		ItemID:   itemID,
		FieldKey: "governance_status",
		Value:    status,
	})
	return err
}

func catalogTMDBMediaType(itemType string) string {
	switch strings.TrimSpace(itemType) {
	case catalog.ItemTypeSeries, catalog.ItemTypeSeason, catalog.ItemTypeEpisode:
		return "tv"
	default:
		return "movie"
	}
}

func catalogItemToSearchItem(item database.CatalogItem) database.MediaItem {
	searchItem := database.MediaItem{
		LibraryID:     item.LibraryID,
		Type:          item.Type,
		Title:         strings.TrimSpace(item.Title),
		OriginalTitle: strings.TrimSpace(item.OriginalTitle),
		Overview:      item.Overview,
		Year:          item.Year,
		SourcePath:    strings.TrimSpace(item.Path),
	}
	if item.Type == catalog.ItemTypeSeries || item.Type == catalog.ItemTypeSeason || item.Type == catalog.ItemTypeEpisode {
		searchItem.SeriesTitle = strings.TrimSpace(item.Title)
	}
	if item.Type == catalog.ItemTypeSeason {
		searchItem.SeasonNumber = item.IndexNumber
	}
	if item.Type == catalog.ItemTypeEpisode {
		searchItem.SeasonNumber = item.ParentIndexNumber
		searchItem.EpisodeNumber = item.IndexNumber
	}
	return searchItem
}

func (s *Service) syncCatalogSeriesHierarchy(ctx context.Context, seriesItem database.CatalogItem, tmdbCfg config.TMDBConfig, seriesTMDBID int, governanceStatus string, confidence float64, seasons []seasonSummary) error {
	catalogSvc := catalog.NewService(s.db)
	for _, seasonSummary := range seasons {
		seasonItem, err := s.findOrCreateCatalogSeasonItem(ctx, catalogSvc, seriesItem, seasonSummary.SeasonNumber, seasonSummary.Name)
		if err != nil {
			return err
		}
		if err := s.applyCatalogHierarchyFields(ctx, catalogSvc, seasonItem.ID, seasonSummary.Name, seasonSummary.Overview, nil, nil, governanceStatus); err != nil {
			return err
		}
		seasonSource, err := s.syncCatalogHierarchyIdentity(ctx, seasonItem.ID, "tv_season", seasonSummary.ID, confidence, map[string]any{
			"media_type":     "tv_season",
			"external_id":    tmdbExternalID(seasonSummary.ID),
			"series_tmdb_id": seriesTMDBID,
			"season_number":  seasonSummary.SeasonNumber,
			"matched_title":  strings.TrimSpace(seasonSummary.Name),
			"poster_path":    strings.TrimSpace(seasonSummary.PosterPath),
		})
		if err != nil {
			return err
		}
		if err := s.upsertCatalogImageCandidate(ctx, seasonItem.ID, "poster", imageURL(tmdbCfg, seasonSummary.PosterPath), "", 0, true, &seasonSource.ID); err != nil {
			return err
		}

		seasonDetail, err := s.fetchTVSeason(ctx, tmdbCfg, seriesTMDBID, seasonSummary.SeasonNumber)
		if err != nil {
			return err
		}
		for _, episode := range seasonDetail.Episodes {
			releaseDate := strings.TrimSpace(episode.AirDate)
			runtimeSeconds := runtimeSecondsFromMinutes(episode.Runtime)
			episodeItem, err := s.findOrCreateCatalogEpisodeItem(ctx, catalogSvc, seasonItem, seasonSummary.SeasonNumber, episode.EpisodeNumber, episode.Name, releaseDate)
			if err != nil {
				return err
			}
			if err := s.applyCatalogHierarchyFields(ctx, catalogSvc, episodeItem.ID, episode.Name, episode.Overview, parseYear(releaseDate), runtimeSeconds, governanceStatus); err != nil {
				return err
			}
			episodeSource, err := s.syncCatalogHierarchyIdentity(ctx, episodeItem.ID, "tv_episode", episode.ID, confidence, map[string]any{
				"media_type":     "tv_episode",
				"external_id":    tmdbExternalID(episode.ID),
				"series_tmdb_id": seriesTMDBID,
				"season_number":  seasonSummary.SeasonNumber,
				"episode_number": episode.EpisodeNumber,
				"matched_title":  strings.TrimSpace(episode.Name),
				"air_date":       releaseDate,
				"still_path":     strings.TrimSpace(episode.StillPath),
			})
			if err != nil {
				return err
			}
			if err := s.upsertCatalogImageCandidate(ctx, episodeItem.ID, "still", imageURL(tmdbCfg, episode.StillPath), "", 0, true, &episodeSource.ID); err != nil {
				return err
			}
			availability, err := s.resolveCatalogLeafAvailability(ctx, episodeItem.ID, releaseDate)
			if err != nil {
				return err
			}
			if err := s.updateCatalogAvailability(ctx, episodeItem.ID, availability); err != nil {
				return err
			}
		}

		seasonAvailability, err := s.resolveCatalogParentAvailability(ctx, seasonItem.ID)
		if err != nil {
			return err
		}
		if err := s.updateCatalogAvailability(ctx, seasonItem.ID, seasonAvailability); err != nil {
			return err
		}
	}

	seriesAvailability, err := s.resolveCatalogParentAvailability(ctx, seriesItem.ID)
	if err != nil {
		return err
	}
	return s.updateCatalogAvailability(ctx, seriesItem.ID, seriesAvailability)
}

func (s *Service) findOrCreateCatalogSeasonItem(ctx context.Context, catalogSvc *catalog.Service, seriesItem database.CatalogItem, seasonNumber int, title string) (database.CatalogItem, error) {
	var season database.CatalogItem
	err := s.db.WithContext(ctx).
		Where("parent_id = ? AND type = ? AND index_number = ? AND deleted_at IS NULL", seriesItem.ID, catalog.ItemTypeSeason, seasonNumber).
		First(&season).Error
	if err == nil {
		return season, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		var zero database.CatalogItem
		return zero, err
	}
	seasonNumberCopy := seasonNumber
	seasonPath := strings.TrimRight(seriesItem.Path, "/") + fmt.Sprintf("/Season %02d", seasonNumber)
	return catalogSvc.CreateItem(ctx, catalog.CreateItemInput{
		LibraryID:          seriesItem.LibraryID,
		Type:               catalog.ItemTypeSeason,
		ParentID:           &seriesItem.ID,
		Path:               seasonPath,
		SortKey:            fmt.Sprintf("%s S%02d", strings.TrimSpace(seriesItem.Title), seasonNumber),
		Title:              firstNonEmptyCatalogValue(strings.TrimSpace(title), fmt.Sprintf("Season %d", seasonNumber)),
		IndexNumber:        &seasonNumberCopy,
		ParentIndexNumber:  &seasonNumberCopy,
		AvailabilityStatus: catalog.AvailabilityMissing,
		GovernanceStatus:   governanceOrPending(""),
	})
}

func (s *Service) findOrCreateCatalogEpisodeItem(ctx context.Context, catalogSvc *catalog.Service, seasonItem database.CatalogItem, seasonNumber int, episodeNumber int, title string, releaseDate string) (database.CatalogItem, error) {
	var episode database.CatalogItem
	err := s.db.WithContext(ctx).
		Where("parent_id = ? AND type = ? AND index_number = ? AND deleted_at IS NULL", seasonItem.ID, catalog.ItemTypeEpisode, episodeNumber).
		First(&episode).Error
	if err == nil {
		return episode, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		var zero database.CatalogItem
		return zero, err
	}
	seasonNumberCopy := seasonNumber
	episodeNumberCopy := episodeNumber
	episodePath := strings.TrimRight(seasonItem.Path, "/") + fmt.Sprintf("/Episode %02d", episodeNumber)
	availability, err := s.resolveCatalogLeafAvailability(ctx, 0, releaseDate)
	if err != nil {
		return database.CatalogItem{}, err
	}
	return catalogSvc.CreateItem(ctx, catalog.CreateItemInput{
		LibraryID:          seasonItem.LibraryID,
		Type:               catalog.ItemTypeEpisode,
		ParentID:           &seasonItem.ID,
		Path:               episodePath,
		SortKey:            fmt.Sprintf("%s E%02d", strings.TrimSpace(seasonItem.Title), episodeNumber),
		Title:              firstNonEmptyCatalogValue(strings.TrimSpace(title), fmt.Sprintf("Episode %d", episodeNumber)),
		IndexNumber:        &episodeNumberCopy,
		ParentIndexNumber:  &seasonNumberCopy,
		AvailabilityStatus: availability,
		GovernanceStatus:   governanceOrPending(""),
	})
}

func (s *Service) applyCatalogHierarchyFields(ctx context.Context, catalogSvc *catalog.Service, itemID uint, title string, overview string, year *int, runtimeSeconds *int, governanceStatus string) error {
	if strings.TrimSpace(title) != "" {
		if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: itemID, FieldKey: "title", Value: strings.TrimSpace(title)}); err != nil {
			return err
		}
		if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: itemID, FieldKey: "sort_title", Value: strings.TrimSpace(title)}); err != nil {
			return err
		}
	}
	if strings.TrimSpace(overview) != "" {
		if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: itemID, FieldKey: "overview", Value: strings.TrimSpace(overview)}); err != nil {
			return err
		}
	}
	if year != nil {
		if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: itemID, FieldKey: "year", Value: *year}); err != nil {
			return err
		}
	}
	if runtimeSeconds != nil {
		if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: itemID, FieldKey: "runtime_seconds", Value: *runtimeSeconds}); err != nil {
			return err
		}
	}
	if governanceStatus != "" {
		if _, _, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: itemID, FieldKey: "governance_status", Value: governanceStatus}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) resolveCatalogLeafAvailability(ctx context.Context, itemID uint, releaseDate string) (string, error) {
	if itemID != 0 {
		var availableAssets int64
		if err := s.db.WithContext(ctx).
			Table("asset_items").
			Joins("JOIN media_assets ON media_assets.id = asset_items.asset_id").
			Where("asset_items.item_id = ?", itemID).
			Where("media_assets.deleted_at IS NULL AND media_assets.status = ?", "available").
			Count(&availableAssets).Error; err != nil {
			return "", err
		}
		if availableAssets > 0 {
			return catalog.AvailabilityAvailable, nil
		}
	}
	if strings.TrimSpace(releaseDate) != "" {
		if parsed, err := time.Parse("2006-01-02", releaseDate); err == nil && parsed.After(time.Now().UTC()) {
			return catalog.AvailabilityUnaired, nil
		}
	}
	return catalog.AvailabilityMissing, nil
}

func (s *Service) resolveCatalogParentAvailability(ctx context.Context, parentID uint) (string, error) {
	var children []database.CatalogItem
	if err := s.db.WithContext(ctx).
		Where("parent_id = ? AND deleted_at IS NULL", parentID).
		Order("id asc").
		Find(&children).Error; err != nil {
		return "", err
	}
	if len(children) == 0 {
		return catalog.AvailabilityNoLocalMedia, nil
	}
	hasUnaired := false
	for _, child := range children {
		switch strings.TrimSpace(child.AvailabilityStatus) {
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

func (s *Service) updateCatalogAvailability(ctx context.Context, itemID uint, availability string) error {
	return s.db.WithContext(ctx).
		Model(&database.CatalogItem{}).
		Where("id = ?", itemID).
		Update("availability_status", availability).Error
}

func governanceOrPending(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return catalog.GovernancePending
	}
	return trimmed
}

func firstNonEmptyCatalogValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (s *Service) syncCatalogDetailImages(ctx context.Context, itemID uint, tmdbCfg config.TMDBConfig, detail detailResponse, sourceID *uint) error {
	if err := s.upsertCatalogImageCandidate(ctx, itemID, "poster", imageURL(tmdbCfg, detail.PosterPath), "", 0, true, sourceID); err != nil {
		return err
	}
	if err := s.upsertCatalogImageCandidate(ctx, itemID, "backdrop", imageURL(tmdbCfg, detail.BackdropPath), "", 0, true, sourceID); err != nil {
		return err
	}
	bestLogoPath := pickLogoPath(tmdbCfg.Language, detail.Images.Logos)
	for idx, logo := range detail.Images.Logos {
		if err := s.upsertCatalogImageCandidate(ctx, itemID, "logo", imageURL(tmdbCfg, logo.FilePath), strings.TrimSpace(logo.Language), idx, strings.TrimSpace(logo.FilePath) == strings.TrimSpace(bestLogoPath), sourceID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) syncCatalogHierarchyIdentity(ctx context.Context, itemID uint, providerType string, tmdbID int, confidence float64, payload map[string]any) (database.MetadataSource, error) {
	if itemID == 0 || tmdbID <= 0 {
		return database.MetadataSource{}, nil
	}
	externalID := tmdbExternalID(tmdbID)
	catalogSvc := catalog.NewService(s.db)
	if _, err := catalogSvc.SetExternalID(ctx, catalog.ExternalIDInput{
		ItemID:       itemID,
		Provider:     "tmdb",
		ProviderType: providerType,
		ExternalID:   externalID,
		IsPrimary:    true,
		Source:       "metadata_match",
		Confidence:   &confidence,
	}); err != nil {
		return database.MetadataSource{}, err
	}
	if payload == nil {
		payload = map[string]any{}
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return database.MetadataSource{}, err
	}
	source, err := catalogSvc.RecordMetadataSource(ctx, catalog.MetadataSourceInput{
		ItemID:      itemID,
		SourceType:  catalog.SourceTypeProvider,
		SourceName:  "tmdb",
		ExternalID:  externalID,
		PayloadJSON: string(payloadJSON),
		Confidence:  &confidence,
	})
	if err != nil {
		return database.MetadataSource{}, err
	}
	return source, nil
}

func (s *Service) upsertCatalogImageCandidate(ctx context.Context, itemID uint, imageType string, url string, language string, sortOrder int, preferSelected bool, sourceID *uint) error {
	trimmedURL := strings.TrimSpace(url)
	trimmedType := strings.TrimSpace(imageType)
	if itemID == 0 || trimmedType == "" || trimmedURL == "" {
		return nil
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var image database.ItemImage
		err := tx.WithContext(ctx).
			Where("item_id = ? AND image_type = ? AND url = ?", itemID, trimmedType, trimmedURL).
			First(&image).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		selectByDefault := false
		if preferSelected {
			var selectedCount int64
			if err := tx.WithContext(ctx).
				Model(&database.ItemImage{}).
				Where("item_id = ? AND image_type = ? AND is_selected = ?", itemID, trimmedType, true).
				Count(&selectedCount).Error; err != nil {
				return err
			}
			selectByDefault = selectedCount == 0
		}

		now := time.Now().UTC()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.WithContext(ctx).Create(&database.ItemImage{
				ItemID:     itemID,
				ImageType:  trimmedType,
				URL:        trimmedURL,
				SourceID:   sourceID,
				Language:   strings.TrimSpace(language),
				IsSelected: selectByDefault,
				SortOrder:  sortOrder,
			}).Error
		}

		updates := map[string]any{
			"source_id":  sourceID,
			"language":   strings.TrimSpace(language),
			"sort_order": sortOrder,
			"updated_at": now,
		}
		if selectByDefault && !image.IsSelected {
			updates["is_selected"] = true
		}
		return tx.WithContext(ctx).Model(&database.ItemImage{}).Where("id = ?", image.ID).Updates(updates).Error
	})
}

func tmdbExternalID(id int) string {
	if id <= 0 {
		return ""
	}
	return fmt.Sprintf("tv:%d", id)
}
