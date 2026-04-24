package library

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

func (s *Service) GetLibrary(ctx context.Context, libraryID uint) (LibraryDetail, error) {
	var detail LibraryDetail
	if err := s.db.WithContext(ctx).First(&detail.Library, libraryID).Error; err != nil {
		return LibraryDetail{}, err
	}
	if err := s.db.WithContext(ctx).Model(&database.MediaItem{}).Where("library_id = ? AND deleted_at IS NULL", libraryID).Count(&detail.MediaItemsCount).Error; err != nil {
		return LibraryDetail{}, err
	}
	if err := s.db.WithContext(ctx).Model(&database.MediaFile{}).Where("library_id = ? AND deleted_at IS NULL", libraryID).Count(&detail.MediaFilesCount).Error; err != nil {
		return LibraryDetail{}, err
	}
	return detail, nil
}

func (s *Service) ListMediaItems(ctx context.Context, libraryID uint, mediaType string, limit int) ([]database.MediaItem, error) {
	input := BrowseMediaItemsInput{LibraryID: libraryID, Scope: BrowseScopeLibrary, TypeFilter: BrowseTypeFilterAll, Sort: BrowseSortTitle, Limit: limit}
	if strings.EqualFold(strings.TrimSpace(mediaType), "movie") {
		input.TypeFilter = BrowseTypeFilterMovie
	}
	if strings.EqualFold(strings.TrimSpace(mediaType), "show") || strings.EqualFold(strings.TrimSpace(mediaType), "episode") {
		input.TypeFilter = BrowseTypeFilterShow
	}
	return s.BrowseMediaItems(ctx, input)
}

func (s *Service) BrowseMediaItems(ctx context.Context, input BrowseMediaItemsInput) ([]database.MediaItem, error) {
	items, err := s.DiscoverMediaItems(ctx, nil, input)
	if err != nil {
		return nil, err
	}
	result := make([]database.MediaItem, 0, len(items))
	for _, item := range items {
		result = append(result, item.Item)
	}
	return result, nil
}

func (s *Service) DiscoverMediaItems(ctx context.Context, userID *uint, input BrowseMediaItemsInput) ([]DiscoveryItem, error) {
	input = NormalizeBrowseMediaItemsInput(input)
	items, err := s.loadDiscoveryMediaItems(ctx, input)
	if err != nil {
		return nil, err
	}
	watchRanks, watchedStates, err := s.loadWatchRanks(ctx, userID)
	if err != nil {
		return nil, err
	}
	candidates := make([]browseCandidate, 0, len(items))
	for _, item := range items {
		if !matchesDiscoveryFilters(item, watchedStates[item.ID], input, shouldUseSearchDocuments(input)) {
			continue
		}
		candidates = append(candidates, browseCandidate{Item: item, WatchRank: watchRanks[item.ID]})
	}
	movies, shows := splitBrowseCandidates(candidates)
	sortBrowseCandidates(movies, input.Sort)
	sortBrowseCandidates(shows, input.Sort)
	groupedShows := groupShowBrowseCandidates(shows)
	result := make([]browseCandidate, 0, len(movies)+len(groupedShows))
	result = append(result, movies...)
	for _, group := range groupedShows {
		result = append(result, browseCandidate{Item: group.Display, WatchRank: group.WatchRank})
	}
	sortBrowseCandidates(result, input.Sort)
	itemsLimit := len(result)
	if input.Limit > 0 {
		itemsLimit = min(input.Limit, len(result))
	}
	itemsOut := make([]database.MediaItem, 0, itemsLimit)
	statesOut := make([]string, 0, itemsLimit)
	for _, candidate := range result {
		itemsOut = append(itemsOut, candidate.Item)
		statesOut = append(statesOut, watchedStates[candidate.Item.ID])
		if input.Limit > 0 && len(itemsOut) >= input.Limit {
			break
		}
	}
	response := make([]DiscoveryItem, 0, len(itemsOut))
	for idx, item := range itemsOut {
		response = append(response, DiscoveryItem{Item: item, WatchedState: statesOut[idx]})
	}
	return response, nil
}

func (s *Service) loadDiscoveryMediaItems(ctx context.Context, input BrowseMediaItemsInput) ([]database.MediaItem, error) {
	if shouldUseSearchDocuments(input) {
		items, err := s.loadDiscoveryMediaItemsFromDocuments(ctx, input)
		if err != nil {
			return nil, err
		}
		if len(items) > 0 {
			return items, nil
		}
	}
	return s.loadDiscoveryMediaItemsLegacy(ctx, input)
}

func (s *Service) loadDiscoveryMediaItemsLegacy(ctx context.Context, input BrowseMediaItemsInput) ([]database.MediaItem, error) {
	query := s.db.WithContext(ctx).Model(&database.MediaItem{}).Where("deleted_at IS NULL")
	if input.Scope == BrowseScopeLibrary {
		query = query.Where("library_id = ?", input.LibraryID)
	}
	if input.Year != nil {
		query = query.Where("year = ?", *input.Year)
	}
	if input.TypeFilter == BrowseTypeFilterMovie {
		query = query.Where("type = ?", "movie")
	}
	if input.TypeFilter == BrowseTypeFilterShow {
		query = query.Where("type = ?", "episode")
	}
	if input.MinRating != nil {
		query = query.Where("vote_average IS NOT NULL AND vote_average >= ?", *input.MinRating)
	}
	var items []database.MediaItem
	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) loadDiscoveryMediaItemsFromDocuments(ctx context.Context, input BrowseMediaItemsInput) ([]database.MediaItem, error) {
	query := s.db.WithContext(ctx).Model(&database.SearchDocument{})
	if input.Scope == BrowseScopeLibrary {
		query = query.Where("library_id = ?", input.LibraryID)
	}
	if input.Year != nil {
		query = query.Where("year = ?", *input.Year)
	}
	if input.TypeFilter != BrowseTypeFilterAll {
		query = query.Where("media_type = ?", string(input.TypeFilter))
	}
	if input.MinRating != nil {
		query = query.Where("vote_average IS NOT NULL AND vote_average >= ?", *input.MinRating)
	}
	if genre := strings.TrimSpace(input.Genre); genre != "" {
		query = query.Where("LOWER(search_genres_text) LIKE ?", likeToken(genre))
	}
	if region := strings.TrimSpace(input.Region); region != "" {
		query = query.Where("LOWER(search_countries_text) LIKE ?", likeToken(region))
	}
	if text := strings.TrimSpace(input.Query); text != "" {
		like := likeToken(text)
		query = query.Where("LOWER(title) LIKE ? OR LOWER(original_title) LIKE ? OR LOWER(series_title) LIKE ? OR LOWER(search_people_text) LIKE ? OR LOWER(overview) LIKE ?", like, like, like, like, like)
	}

	var ids []uint
	if err := query.Pluck("media_item_id", &ids).Error; err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}

	var items []database.MediaItem
	if err := s.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	order := make(map[uint]int, len(ids))
	for idx, id := range ids {
		order[id] = idx
	}
	sort.SliceStable(items, func(i, j int) bool {
		return order[items[i].ID] < order[items[j].ID]
	})
	return items, nil
}

func (s *Service) ListRecentlyAdded(ctx context.Context, limit int) ([]database.MediaItem, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	var items []database.MediaItem
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL").Order("created_at desc, id desc").Limit(limit).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) ListLatestByLibrary(ctx context.Context, limit int) ([]LatestByLibrarySection, error) {
	if limit <= 0 || limit > 50 {
		limit = 12
	}
	libraries, err := s.ListActiveLibraries(ctx)
	if err != nil {
		return nil, err
	}
	sections := make([]LatestByLibrarySection, 0, len(libraries))
	for _, libraryRecord := range libraries {
		items, err := s.BrowseMediaItems(ctx, BrowseMediaItemsInput{LibraryID: libraryRecord.ID, Scope: BrowseScopeLibrary, TypeFilter: BrowseTypeFilterAll, Sort: BrowseSortRecent, Limit: limit})
		if err != nil {
			return nil, err
		}
		sections = append(sections, LatestByLibrarySection{LibraryID: libraryRecord.ID, LibraryName: libraryRecord.Name, Items: items})
	}
	return sections, nil
}

func (s *Service) ListAllLatestMediaByLibrary(ctx context.Context) ([]LatestByLibrarySection, error) {
	libraries, err := s.ListLibraries(ctx)
	if err != nil {
		return nil, err
	}
	sections := make([]LatestByLibrarySection, 0, len(libraries))
	for _, libraryRecord := range libraries {
		var items []database.MediaItem
		if err := s.db.WithContext(ctx).Where("library_id = ? AND deleted_at IS NULL", libraryRecord.ID).Order("created_at desc, id desc").Find(&items).Error; err != nil {
			return nil, err
		}
		sections = append(sections, LatestByLibrarySection{LibraryID: libraryRecord.ID, LibraryName: libraryRecord.Name, Items: items})
	}
	return sections, nil
}

func (s *Service) loadWatchRanks(ctx context.Context, userID *uint) (map[uint]int, map[uint]string, error) {
	type playbackRankRow struct {
		MediaItemID uint
		WatchRank   int
		Watched     int
		Position    int
	}
	var rows []playbackRankRow
	query := s.db.WithContext(ctx).Model(&database.PlaybackProgress{})
	if userID != nil && *userID > 0 {
		query = query.Where("user_id = ?", *userID)
	}
	if err := query.Select(`media_item_id,
			MIN(CASE
				WHEN watched = 0 AND position_seconds > 0 THEN 1
				WHEN watched = 1 THEN 2
				ELSE 0
			END) AS watch_rank,
			MAX(CASE WHEN watched = 1 THEN 1 ELSE 0 END) AS watched,
			MAX(position_seconds) AS position`).Group("media_item_id").Scan(&rows).Error; err != nil {
		return nil, nil, err
	}
	ranks := make(map[uint]int, len(rows))
	states := make(map[uint]string, len(rows))
	for _, row := range rows {
		ranks[row.MediaItemID] = row.WatchRank
		switch {
		case row.Watched > 0:
			states[row.MediaItemID] = string(WatchedStateFilterWatched)
		case row.Position > 0:
			states[row.MediaItemID] = string(WatchedStateFilterInProgress)
		default:
			states[row.MediaItemID] = string(WatchedStateFilterUnwatched)
		}
	}
	return ranks, states, nil
}

func matchesDiscoveryFilters(item database.MediaItem, watchedState string, input BrowseMediaItemsInput, skipMetadataFilters bool) bool {
	if !skipMetadataFilters {
		if input.Genre != "" && !stringListContains(item.GenresJSON, input.Genre) {
			return false
		}
		if input.Region != "" && !stringListContains(item.RegionsJSON, input.Region) {
			return false
		}
	}
	if input.Watched != WatchedStateFilterAll {
		state := watchedState
		if state == "" {
			state = string(WatchedStateFilterUnwatched)
		}
		if state != string(input.Watched) {
			return false
		}
	}
	if skipMetadataFilters || input.Query == "" {
		return true
	}
	query := strings.ToLower(strings.TrimSpace(input.Query))
	for _, field := range searchableFields(item) {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func shouldUseSearchDocuments(input BrowseMediaItemsInput) bool {
	return strings.TrimSpace(input.Query) != "" || strings.TrimSpace(input.Genre) != "" || strings.TrimSpace(input.Region) != "" || input.MinRating != nil || input.Year != nil || input.TypeFilter != BrowseTypeFilterAll
}

func likeToken(value string) string {
	return "%" + strings.ToLower(strings.TrimSpace(value)) + "%"
}

func stringListContains(input string, target string) bool {
	values, err := parseStringList(input)
	if err != nil {
		return false
	}
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}

func searchableFields(item database.MediaItem) []string {
	fields := []string{
		item.Title,
		item.OriginalTitle,
		item.SeriesTitle,
	}
	if people, err := parsePeopleList(item.CastJSON); err == nil {
		for _, person := range people {
			fields = append(fields, person.Name)
		}
	}
	if people, err := parsePeopleList(item.DirectorsJSON); err == nil {
		for _, person := range people {
			fields = append(fields, person.Name)
		}
	}
	return fields
}

func splitBrowseCandidates(candidates []browseCandidate) ([]browseCandidate, []browseCandidate) {
	movies := make([]browseCandidate, 0, len(candidates))
	shows := make([]browseCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Item.Type == "movie" {
			movies = append(movies, candidate)
			continue
		}
		shows = append(shows, candidate)
	}
	return movies, shows
}

func sortBrowseCandidates(candidates []browseCandidate, sortBy BrowseSort) {
	sort.SliceStable(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		switch sortBy {
		case BrowseSortRecent:
			if !left.Item.CreatedAt.Equal(right.Item.CreatedAt) {
				return left.Item.CreatedAt.After(right.Item.CreatedAt)
			}
		case BrowseSortYear:
			leftYear, rightYear := normalizeYear(left.Item.Year), normalizeYear(right.Item.Year)
			if leftYear != rightYear {
				return leftYear > rightYear
			}
		case BrowseSortWatchStatus:
			if left.WatchRank != right.WatchRank {
				return left.WatchRank < right.WatchRank
			}
		}
		leftTitle := browseDisplayTitle(left.Item)
		rightTitle := browseDisplayTitle(right.Item)
		if leftTitle != rightTitle {
			return leftTitle < rightTitle
		}
		return left.Item.ID < right.Item.ID
	})
}

func groupShowBrowseCandidates(candidates []browseCandidate) []showDiscoveryGroup {
	groups := make([]showDiscoveryGroup, 0)
	groupIndex := make(map[string]int)
	for _, candidate := range candidates {
		key := browseShowKey(candidate.Item)
		if idx, ok := groupIndex[key]; ok {
			group := groups[idx]
			if browseAnchorLess(candidate.Item, group.Anchor) {
				group.Anchor = candidate.Item
			}
			if betterShowMetadata(candidate.Item, group.Display) {
				group.Display = candidate.Item
			}
			if candidate.WatchRank < group.WatchRank {
				group.WatchRank = candidate.WatchRank
			}
			groups[idx] = group
			continue
		}
		groupIndex[key] = len(groups)
		groups = append(groups, showDiscoveryGroup{Anchor: candidate.Item, Display: candidate.Item, WatchRank: candidate.WatchRank, Representative: len(groups)})
	}
	for idx := range groups {
		group := groups[idx]
		display := group.Anchor
		seriesTitle := strings.TrimSpace(group.Display.SeriesTitle)
		if seriesTitle == "" {
			seriesTitle = strings.TrimSpace(group.Anchor.SeriesTitle)
		}
		if seriesTitle != "" {
			display.Title = seriesTitle
			display.OriginalTitle = seriesTitle
			display.SeriesTitle = seriesTitle
		}
		display.Type = string(BrowseTypeFilterShow)
		display.SeasonNumber = nil
		display.EpisodeNumber = nil
		if strings.TrimSpace(group.Display.Overview) != "" {
			display.Overview = group.Display.Overview
		}
		if strings.TrimSpace(group.Display.PosterURL) != "" {
			display.PosterURL = group.Display.PosterURL
		}
		if strings.TrimSpace(group.Display.BackdropURL) != "" {
			display.BackdropURL = group.Display.BackdropURL
		}
		if strings.TrimSpace(group.Display.LogoURL) != "" {
			display.LogoURL = group.Display.LogoURL
		}
		if display.Year == nil {
			display.Year = group.Display.Year
		}
		groups[idx].Display = display
	}
	return groups
}

func browseDisplayTitle(item database.MediaItem) string {
	title := strings.TrimSpace(item.Title)
	if strings.EqualFold(item.Type, string(BrowseTypeFilterShow)) {
		if seriesTitle := strings.TrimSpace(item.SeriesTitle); seriesTitle != "" {
			title = seriesTitle
		}
	}
	if title == "" {
		title = strings.TrimSpace(item.SeriesTitle)
	}
	if title == "" {
		title = strings.TrimSpace(item.SourcePath)
	}
	return strings.ToLower(title)
}

func browseShowKey(item database.MediaItem) string {
	if externalID := strings.TrimSpace(item.ExternalID); externalID != "" {
		return "external:" + externalID
	}
	seriesTitle := strings.TrimSpace(item.SeriesTitle)
	if seriesTitle == "" {
		seriesTitle = strings.TrimSpace(item.Title)
	}
	return fmt.Sprintf("library:%d:series:%s", item.LibraryID, strings.ToLower(seriesTitle))
}

func browseAnchorLess(left, right database.MediaItem) bool {
	leftSeason, rightSeason := normalizeOrdinal(left.SeasonNumber), normalizeOrdinal(right.SeasonNumber)
	if leftSeason != rightSeason {
		return leftSeason < rightSeason
	}
	leftEpisode, rightEpisode := normalizeOrdinal(left.EpisodeNumber), normalizeOrdinal(right.EpisodeNumber)
	if leftEpisode != rightEpisode {
		return leftEpisode < rightEpisode
	}
	return left.ID < right.ID
}

func betterShowMetadata(candidate, current database.MediaItem) bool {
	if strings.TrimSpace(current.Overview) == "" && strings.TrimSpace(candidate.Overview) != "" {
		return true
	}
	if strings.TrimSpace(current.PosterURL) == "" && strings.TrimSpace(candidate.PosterURL) != "" {
		return true
	}
	if strings.TrimSpace(current.BackdropURL) == "" && strings.TrimSpace(candidate.BackdropURL) != "" {
		return true
	}
	if strings.TrimSpace(current.SeriesTitle) == "" && strings.TrimSpace(candidate.SeriesTitle) != "" {
		return true
	}
	return false
}

func normalizeYear(year *int) int {
	if year == nil {
		return -1
	}
	return *year
}

func normalizeOrdinal(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func ParseBrowseYear(value string) *int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	year, err := strconv.Atoi(trimmed)
	if err != nil || year <= 0 {
		return nil
	}
	return &year
}
