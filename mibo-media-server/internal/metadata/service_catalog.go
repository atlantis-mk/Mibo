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
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/settings"
	"gorm.io/gorm"
)

func (s *Service) MatchCatalogItem(ctx context.Context, itemID uint) error {
	_, err := s.MatchCatalogItemOperation(ctx, itemID)
	return err
}

func (s *Service) MatchCatalogItemOperation(ctx context.Context, itemID uint) (MetadataOperationResult, error) {
	operation, err := s.runMetadataOperation(ctx, MetadataOperationRequest{Operation: OperationTypeMatch, OriginItemID: itemID})
	if err != nil {
		return MetadataOperationResult{}, err
	}
	return operation, nil
}

func (s *Service) SearchCatalogCandidates(ctx context.Context, itemID uint, input ManualSearchInput) ([]SearchCandidate, error) {
	target, err := s.resolveCatalogMatchTarget(ctx, itemID)
	if err != nil {
		return nil, err
	}
	profile, err := s.resolvedCatalogProfile(ctx, target.LibraryID)
	if err != nil {
		return nil, err
	}
	selection, err := s.selectProviderForStage(profile, "search", "")
	if err != nil {
		return nil, err
	}
	if selection.Provider.Record.ID == 0 {
		return nil, fmt.Errorf("当前媒体库没有可用于 search 阶段的远程元数据 provider")
	}
	tmdbCfg := selection.Provider.TMDB
	if language := strings.TrimSpace(profile.PreferredMetadataLanguage); language != "" {
		tmdbCfg.Language = language
	}
	mediaType := catalogTMDBMediaType(target.Type)
	searchItem := catalogItemToSearchItem(target)
	if selection.Provider.Record.ProviderType == database.MetadataProviderTypeMetaTube {
		if mediaType != "movie" {
			return nil, fmt.Errorf("MetaTube provider only supports movie catalog metadata")
		}
		if strings.TrimSpace(input.TMDBID) != "" || strings.TrimSpace(input.TVDBID) != "" {
			return nil, fmt.Errorf("MetaTube search does not support tmdb_id or tvdb_id lookup")
		}
		queries := buildManualSearchQueries(input, searchItem, mediaType)
		if len(queries) == 0 {
			return nil, fmt.Errorf("标题不能为空")
		}
		results, attempt, err := s.executeMetaTubeSearchStage(ctx, selection.Provider, queries, searchItem)
		if err != nil {
			s.recordProviderFailure(ctx, selection.Provider, err)
			return nil, err
		}
		_ = attempt
		return searchCandidatesFromNormalized(results), nil
	}

	if tmdbID := strings.TrimSpace(input.TMDBID); tmdbID != "" {
		id, err := strconv.Atoi(tmdbID)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("tmdb_id 必须是正整数")
		}
		detail, err := s.fetchDetail(ctx, tmdbCfg, mediaType, id)
		if err != nil {
			s.recordProviderFailure(ctx, selection.Provider, err)
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
	providerForSearch := selection.Provider
	providerForSearch.TMDB = tmdbCfg
	results, attempt, err := s.executeTMDBSearchStage(ctx, providerForSearch, mediaType, queries, searchItem)
	if err != nil {
		s.recordProviderFailure(ctx, selection.Provider, err)
		return nil, err
	}
	_ = attempt
	return searchCandidatesFromNormalized(results), nil
}

func (s *Service) ApplyCatalogCandidate(ctx context.Context, itemID uint, input ApplyCandidateInput) error {
	_, err := s.ApplyCatalogCandidateOperation(ctx, itemID, input)
	return err
}

func (s *Service) ApplyCatalogCandidateOperation(ctx context.Context, itemID uint, input ApplyCandidateInput) (MetadataOperationResult, error) {
	operation, err := s.runMetadataOperation(ctx, MetadataOperationRequest{Operation: OperationTypeManualApply, OriginItemID: itemID, ManualCandidateExternalID: input.ExternalID})
	if err != nil {
		return MetadataOperationResult{}, err
	}
	return operation, nil
}

func (s *Service) RefetchCatalogItem(ctx context.Context, itemID uint) error {
	_, err := s.RefetchCatalogItemOperation(ctx, itemID)
	return err
}

func (s *Service) RefetchCatalogItemOperation(ctx context.Context, itemID uint) (MetadataOperationResult, error) {
	operation, err := s.runMetadataOperation(ctx, MetadataOperationRequest{Operation: OperationTypeRefetch, OriginItemID: itemID})
	if err != nil {
		return MetadataOperationResult{}, err
	}
	return operation, nil
}

func (s *Service) loadCatalogMetadataOrigin(ctx context.Context, itemID uint) (database.CatalogItem, error) {
	var item database.CatalogItem
	err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", itemID).First(&item).Error
	return item, err
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

func (s *Service) loadCatalogTMDBIdentity(ctx context.Context, itemID uint, mediaType string) (string, string, float64, error) {
	var identity database.CatalogExternalID
	if err := s.db.WithContext(ctx).
		Where("item_id = ? AND provider = ? AND provider_type = ?", itemID, "tmdb", mediaType).
		Order("is_primary desc, id asc").
		First(&identity).Error; err != nil {
		return "", "", 0, fmt.Errorf("当前 catalog 条目没有可重抓的 TMDB 匹配结果: %w", err)
	}

	confidence := 1.0
	if identity.Confidence != nil && *identity.Confidence > 0 {
		confidence = *identity.Confidence
	}
	providerInstanceName := ""
	var source database.MetadataSource
	if err := s.db.WithContext(ctx).Where("item_id = ? AND external_id = ?", itemID, strings.TrimSpace(identity.ExternalID)).Order("id desc").First(&source).Error; err == nil {
		providerInstanceName = strings.TrimSpace(source.ProviderInstanceName)
	}
	return providerInstanceName, strings.TrimSpace(identity.ExternalID), confidence, nil
}

func (s *Service) loadCatalogMetaTubeIdentity(ctx context.Context, itemID uint) (string, string, float64, error) {
	var identity database.CatalogExternalID
	if err := s.db.WithContext(ctx).
		Where("item_id = ? AND provider = ?", itemID, database.MetadataProviderTypeMetaTube).
		Order("is_primary desc, id asc").
		First(&identity).Error; err != nil {
		return "", "", 0, err
	}
	confidence := 1.0
	if identity.Confidence != nil && *identity.Confidence > 0 {
		confidence = *identity.Confidence
	}
	providerInstanceName := ""
	var source database.MetadataSource
	if err := s.db.WithContext(ctx).Where("item_id = ? AND source_name = ? AND external_id = ?", itemID, database.MetadataProviderTypeMetaTube, strings.TrimSpace(identity.ExternalID)).Order("id desc").First(&source).Error; err == nil {
		providerInstanceName = strings.TrimSpace(source.ProviderInstanceName)
	}
	return providerInstanceName, strings.TrimSpace(identity.ExternalID), confidence, nil
}

func (s *Service) loadCatalogTMDBIdentityOptional(ctx context.Context, itemID uint, mediaType string) (string, bool, error) {
	var identity database.CatalogExternalID
	if err := s.db.WithContext(ctx).
		Where("item_id = ? AND provider = ? AND provider_type = ?", itemID, "tmdb", mediaType).
		Order("is_primary desc, id asc").
		First(&identity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(identity.ExternalID), true, nil
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

func catalogItemToSearchItem(item database.CatalogItem) metadataSearchItem {
	searchItem := metadataSearchItem{
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
	if seasonItem.RootID != nil && *seasonItem.RootID > 0 {
		err = s.db.WithContext(ctx).
			Where("root_id = ? AND type = ? AND parent_index_number = ? AND index_number = ? AND deleted_at IS NULL", *seasonItem.RootID, catalog.ItemTypeEpisode, seasonNumber, episodeNumber).
			Order("id asc").
			First(&episode).Error
		if err == nil {
			updates := map[string]any{"parent_id": seasonItem.ID}
			if episode.RootID == nil || *episode.RootID == 0 {
				updates["root_id"] = *seasonItem.RootID
			}
			if err := s.db.WithContext(ctx).Model(&database.CatalogItem{}).Where("id = ?", episode.ID).Updates(updates).Error; err != nil {
				return database.CatalogItem{}, err
			}
			episode.ParentID = &seasonItem.ID
			if episode.RootID == nil || *episode.RootID == 0 {
				episode.RootID = seasonItem.RootID
			}
			return episode, nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return database.CatalogItem{}, err
		}
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

func (s *Service) syncCatalogPeople(ctx context.Context, itemID uint, cast []library.PersonDetail, directors []library.PersonDetail, sourceID *uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Where("item_id = ?", itemID).Delete(&database.ItemPerson{}).Error; err != nil {
			return err
		}

		rows, err := s.buildCatalogItemPeopleRows(ctx, tx, itemID, cast, directors, sourceID)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.WithContext(ctx).Create(&rows).Error
	})
}

func (s *Service) buildCatalogItemPeopleRows(ctx context.Context, tx *gorm.DB, itemID uint, cast []library.PersonDetail, directors []library.PersonDetail, sourceID *uint) ([]database.ItemPerson, error) {
	rows := make([]database.ItemPerson, 0, len(cast)+len(directors))
	seen := make(map[string]struct{}, len(cast)+len(directors))

	appendRows := func(relationRole string, people []library.PersonDetail) error {
		sortOrder := 0
		for _, person := range people {
			name := strings.TrimSpace(person.Name)
			if name == "" {
				continue
			}
			key := relationRole + "\x00" + name
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}

			record := database.Person{}
			err := loadCatalogPersonRecord(ctx, tx, name, person.TMDBPersonID, &record)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				record = database.Person{Name: name, SortName: strings.ToLower(name), AvatarURL: strings.TrimSpace(person.AvatarURL), TMDBPersonID: person.TMDBPersonID}
				if err := tx.WithContext(ctx).Create(&record).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else {
				updates := map[string]any{}
				if avatarURL := strings.TrimSpace(person.AvatarURL); avatarURL != "" && strings.TrimSpace(record.AvatarURL) != avatarURL {
					updates["avatar_url"] = avatarURL
				}
				if person.TMDBPersonID != nil && *person.TMDBPersonID > 0 && (record.TMDBPersonID == nil || *record.TMDBPersonID != *person.TMDBPersonID) {
					updates["tmdb_person_id"] = *person.TMDBPersonID
				}
				if len(updates) > 0 {
					if err := tx.WithContext(ctx).Model(&database.Person{}).Where("id = ?", record.ID).Updates(updates).Error; err != nil {
						return err
					}
				}
			}

			rows = append(rows, database.ItemPerson{
				ItemID:    itemID,
				PersonID:  record.ID,
				Role:      relationRole,
				Character: strings.TrimSpace(person.Role),
				SortOrder: sortOrder,
				SourceID:  sourceID,
			})
			sortOrder += 1
		}
		return nil
	}

	if err := appendRows("cast", cast); err != nil {
		return nil, err
	}
	if err := appendRows("director", directors); err != nil {
		return nil, err
	}
	return rows, nil
}

func loadCatalogPersonRecord(ctx context.Context, tx *gorm.DB, name string, tmdbPersonID *int, record *database.Person) error {
	if tmdbPersonID != nil && *tmdbPersonID > 0 {
		err := tx.WithContext(ctx).Where("tmdb_person_id = ?", *tmdbPersonID).First(record).Error
		if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	return tx.WithContext(ctx).Where("name = ?", name).First(record).Error
}

func (s *Service) applyCatalogHierarchyFields(ctx context.Context, catalogSvc *catalog.Service, itemID uint, title string, overview string, year *int, runtimeSeconds *int, governanceStatus string, sourceID *uint, confidence *float64) ([]MetadataAppliedField, []MetadataSkippedField, error) {
	applied := make([]MetadataAppliedField, 0, 6)
	skipped := make([]MetadataSkippedField, 0)
	apply := func(fieldKey string, value any) error {
		_, didApply, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: itemID, FieldKey: fieldKey, Value: value, SourceID: sourceID})
		if err != nil {
			return err
		}
		if !didApply {
			skipped = append(skipped, MetadataSkippedField{ItemID: itemID, FieldKey: fieldKey, Reason: "not_applied"})
			return nil
		}
		applied = append(applied, MetadataAppliedField{ItemID: itemID, FieldKey: fieldKey, SourceID: sourceID, ApplyMode: FieldApplyModeAutomated, Confidence: confidence})
		return nil
	}
	if strings.TrimSpace(title) != "" {
		if err := apply("title", strings.TrimSpace(title)); err != nil {
			return nil, nil, err
		}
		if err := apply("sort_title", strings.TrimSpace(title)); err != nil {
			return nil, nil, err
		}
	}
	if strings.TrimSpace(overview) != "" {
		if err := apply("overview", strings.TrimSpace(overview)); err != nil {
			return nil, nil, err
		}
	}
	if year != nil {
		if err := apply("year", *year); err != nil {
			return nil, nil, err
		}
	}
	if runtimeSeconds != nil {
		if err := apply("runtime_seconds", *runtimeSeconds); err != nil {
			return nil, nil, err
		}
	}
	if governanceStatus != "" {
		if err := apply("governance_status", governanceStatus); err != nil {
			return nil, nil, err
		}
	}
	return applied, skipped, nil
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

func (s *Service) syncCatalogHierarchyIdentity(ctx context.Context, itemID uint, profile settings.ResolvedLibraryMetadataProfile, provider settings.ResolvedMetadataProviderInstance, fallback []settings.MetadataExecutionFallbackSummary, providerType string, tmdbID int, confidence float64, payload map[string]any) (database.MetadataSource, error) {
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
	if _, err := catalogSvc.SetIdentity(ctx, catalog.IdentityInput{ItemID: itemID, Provider: "tmdb", IdentityType: providerType, IdentityKey: externalID, Confidence: &confidence}); err != nil {
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
		ItemID:               itemID,
		SourceType:           catalog.SourceTypeProvider,
		SourceName:           provider.Record.ProviderType,
		ExternalID:           externalID,
		MetadataProfileID:    &profile.Profile.ID,
		MetadataProfileName:  profile.Profile.Name,
		ProviderInstanceID:   &provider.Record.ID,
		ProviderInstanceName: provider.Record.Name,
		FallbackSummaryJSON:  mustMarshalFallbackSummary(fallback),
		PayloadJSON:          string(payloadJSON),
		Confidence:           &confidence,
	})
	if err != nil {
		return database.MetadataSource{}, err
	}
	return source, nil
}

func (s *Service) upsertCatalogImageCandidate(ctx context.Context, itemID uint, imageType string, url string, language string, sortOrder int, preferSelected bool, forceSelected bool, sourceID *uint) error {
	trimmedURL := strings.TrimSpace(url)
	trimmedType := strings.TrimSpace(imageType)
	if itemID == 0 || trimmedType == "" || trimmedURL == "" {
		return nil
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sameTypeImages []database.ItemImage
		if preferSelected {
			if err := tx.WithContext(ctx).
				Where("item_id = ? AND image_type = ?", itemID, trimmedType).
				Order("id asc").
				Find(&sameTypeImages).Error; err != nil {
				return err
			}
		}

		var image database.ItemImage
		err := tx.WithContext(ctx).
			Where("item_id = ? AND image_type = ? AND url = ?", itemID, trimmedType, trimmedURL).
			First(&image).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		selectByDefault := false
		if preferSelected {
			shouldPrefer, err := s.shouldPreferCatalogSelectedImage(ctx, tx, itemID, sameTypeImages)
			if err != nil {
				return err
			}
			selectByDefault = forceSelected || shouldPrefer
		}

		now := time.Now().UTC()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if selectByDefault {
				if err := tx.WithContext(ctx).
					Model(&database.ItemImage{}).
					Where("item_id = ? AND image_type = ?", itemID, trimmedType).
					Update("is_selected", false).Error; err != nil {
					return err
				}
			}
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
			if err := tx.WithContext(ctx).
				Model(&database.ItemImage{}).
				Where("item_id = ? AND image_type = ?", itemID, trimmedType).
				Update("is_selected", false).Error; err != nil {
				return err
			}
			updates["is_selected"] = true
		}
		return tx.WithContext(ctx).Model(&database.ItemImage{}).Where("id = ?", image.ID).Updates(updates).Error
	})
}

func (s *Service) shouldPreferCatalogSelectedImage(ctx context.Context, tx *gorm.DB, itemID uint, images []database.ItemImage) (bool, error) {
	scannerSourceIDs := make(map[uint]struct{})
	for _, image := range images {
		if image.IsSelected && image.SourceID != nil {
			scannerSourceIDs[*image.SourceID] = struct{}{}
		}
	}
	if len(scannerSourceIDs) > 0 {
		ids := make([]uint, 0, len(scannerSourceIDs))
		for id := range scannerSourceIDs {
			ids = append(ids, id)
		}
		var sources []database.MetadataSource
		if err := tx.WithContext(ctx).Where("id IN ?", ids).Find(&sources).Error; err != nil {
			return false, err
		}
		scannerSourceIDs = make(map[uint]struct{}, len(sources))
		for _, source := range sources {
			if source.SourceType == catalog.SourceTypeLocalFile && source.SourceName == "scanner" {
				scannerSourceIDs[source.ID] = struct{}{}
			}
		}
	}
	for _, image := range images {
		if !image.IsSelected {
			continue
		}
		if isGeneratedCatalogArtworkURL(itemID, image.URL) {
			continue
		}
		if image.SourceID != nil {
			if _, ok := scannerSourceIDs[*image.SourceID]; ok {
				continue
			}
		}
		return false, nil
	}
	return true, nil
}

func isGeneratedCatalogArtworkURL(itemID uint, rawURL string) bool {
	prefix := fmt.Sprintf("/api/v1/items/%d/artwork/", itemID)
	return strings.HasPrefix(strings.TrimSpace(rawURL), prefix)
}

func tmdbExternalID(id int) string {
	if id <= 0 {
		return ""
	}
	return fmt.Sprintf("tv:%d", id)
}
