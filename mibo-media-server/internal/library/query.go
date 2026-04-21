package library

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

type BrowseScope string

const (
	BrowseScopeLibrary BrowseScope = "library"
	BrowseScopeAll     BrowseScope = "all"
)

type BrowseTypeFilter string

const (
	BrowseTypeFilterAll   BrowseTypeFilter = "all"
	BrowseTypeFilterMovie BrowseTypeFilter = "movie"
	BrowseTypeFilterShow  BrowseTypeFilter = "show"
)

type BrowseSort string

const (
	BrowseSortRecent      BrowseSort = "recent"
	BrowseSortTitle       BrowseSort = "title"
	BrowseSortYear        BrowseSort = "year"
	BrowseSortWatchStatus BrowseSort = "watch_status"
)

type BrowseMediaItemsInput struct {
	LibraryID  uint
	Scope      BrowseScope
	TypeFilter BrowseTypeFilter
	Year       *int
	Sort       BrowseSort
	Limit      int
}

type browseCandidate struct {
	Item      database.MediaItem
	WatchRank int
}

type showDiscoveryGroup struct {
	Anchor         database.MediaItem
	Display        database.MediaItem
	WatchRank      int
	Representative int
}

func NormalizeBrowseMediaItemsInput(input BrowseMediaItemsInput) BrowseMediaItemsInput {
	if input.Scope != BrowseScopeAll {
		input.Scope = BrowseScopeLibrary
	}

	switch input.TypeFilter {
	case BrowseTypeFilterMovie, BrowseTypeFilterShow:
	default:
		input.TypeFilter = BrowseTypeFilterAll
	}

	switch input.Sort {
	case BrowseSortTitle, BrowseSortYear, BrowseSortWatchStatus:
	default:
		input.Sort = BrowseSortRecent
	}

	if input.Year != nil && *input.Year <= 0 {
		input.Year = nil
	}

	if input.Limit <= 0 || input.Limit > 200 {
		input.Limit = 50
	}

	return input
}

type PersonDetail struct {
	Name      string `json:"name"`
	Role      string `json:"role"`
	AvatarURL string `json:"avatar_url"`
}

type LibraryDetail struct {
	database.Library
	MediaItemsCount int64 `json:"media_items_count"`
	MediaFilesCount int64 `json:"media_files_count"`
}

type MediaItemDetail struct {
	database.MediaItem
	Genres    []string          `json:"genres"`
	Cast      []PersonDetail    `json:"cast"`
	Directors []PersonDetail    `json:"directors"`
	Files     []MediaFileDetail `json:"files"`
}

type MediaFileDetail struct {
	database.MediaFile
	AudioTracks    []TrackDetail `json:"audio_tracks"`
	SubtitleTracks []TrackDetail `json:"subtitle_tracks"`
}

type TrackDetail struct {
	Codec    string `json:"codec"`
	Language string `json:"language"`
	Title    string `json:"title"`
	Channels int    `json:"channels,omitempty"`
}

func (s *Service) GetLibrary(ctx context.Context, libraryID uint) (LibraryDetail, error) {
	var detail LibraryDetail
	if err := s.db.WithContext(ctx).First(&detail.Library, libraryID).Error; err != nil {
		return LibraryDetail{}, err
	}

	if err := s.db.WithContext(ctx).
		Model(&database.MediaItem{}).
		Where("library_id = ? AND deleted_at IS NULL", libraryID).
		Count(&detail.MediaItemsCount).Error; err != nil {
		return LibraryDetail{}, err
	}
	if err := s.db.WithContext(ctx).
		Model(&database.MediaFile{}).
		Where("library_id = ? AND deleted_at IS NULL", libraryID).
		Count(&detail.MediaFilesCount).Error; err != nil {
		return LibraryDetail{}, err
	}

	return detail, nil
}

func (s *Service) ListMediaItems(ctx context.Context, libraryID uint, mediaType string, limit int) ([]database.MediaItem, error) {
	input := BrowseMediaItemsInput{
		LibraryID:  libraryID,
		Scope:      BrowseScopeLibrary,
		TypeFilter: BrowseTypeFilterAll,
		Sort:       BrowseSortTitle,
		Limit:      limit,
	}
	if strings.EqualFold(strings.TrimSpace(mediaType), "movie") {
		input.TypeFilter = BrowseTypeFilterMovie
	}
	if strings.EqualFold(strings.TrimSpace(mediaType), "show") || strings.EqualFold(strings.TrimSpace(mediaType), "episode") {
		input.TypeFilter = BrowseTypeFilterShow
	}
	return s.BrowseMediaItems(ctx, input)
}

func (s *Service) BrowseMediaItems(ctx context.Context, input BrowseMediaItemsInput) ([]database.MediaItem, error) {
	input = NormalizeBrowseMediaItemsInput(input)

	query := s.db.WithContext(ctx).Model(&database.MediaItem{}).
		Where("deleted_at IS NULL")
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

	var items []database.MediaItem
	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}

	watchRanks, err := s.loadWatchRanks(ctx)
	if err != nil {
		return nil, err
	}

	candidates := make([]browseCandidate, 0, len(items))
	for _, item := range items {
		candidates = append(candidates, browseCandidate{
			Item:      item,
			WatchRank: watchRanks[item.ID],
		})
	}

	movies, shows := splitBrowseCandidates(candidates)
	sortBrowseCandidates(movies, input.Sort)
	sortBrowseCandidates(shows, input.Sort)

	groupedShows := groupShowBrowseCandidates(shows)
	result := make([]browseCandidate, 0, len(movies)+len(groupedShows))
	result = append(result, movies...)
	for _, group := range groupedShows {
		result = append(result, browseCandidate{
			Item:      group.Display,
			WatchRank: group.WatchRank,
		})
	}

	sortBrowseCandidates(result, input.Sort)

	itemsOut := make([]database.MediaItem, 0, min(input.Limit, len(result)))
	for _, candidate := range result {
		itemsOut = append(itemsOut, candidate.Item)
		if len(itemsOut) >= input.Limit {
			break
		}
	}

	return itemsOut, nil
}

func (s *Service) ListRecentlyAdded(ctx context.Context, limit int) ([]database.MediaItem, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}

	var items []database.MediaItem
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("created_at desc, id desc").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, err
	}

	return items, nil
}

func (s *Service) loadWatchRanks(ctx context.Context) (map[uint]int, error) {
	type playbackRankRow struct {
		MediaItemID uint
		WatchRank   int
	}

	var rows []playbackRankRow
	if err := s.db.WithContext(ctx).
		Model(&database.PlaybackProgress{}).
		Select(`media_item_id,
			MIN(CASE
				WHEN watched = 0 AND position_seconds > 0 THEN 1
				WHEN watched = 1 THEN 2
				ELSE 0
			END) AS watch_rank`).
		Group("media_item_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	ranks := make(map[uint]int, len(rows))
	for _, row := range rows {
		ranks[row.MediaItemID] = row.WatchRank
	}
	return ranks, nil
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
		groups = append(groups, showDiscoveryGroup{
			Anchor:         candidate.Item,
			Display:        candidate.Item,
			WatchRank:      candidate.WatchRank,
			Representative: len(groups),
		})
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

func (s *Service) GetMediaItem(ctx context.Context, mediaItemID uint) (MediaItemDetail, error) {
	var item database.MediaItem
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", mediaItemID).
		First(&item).Error; err != nil {
		return MediaItemDetail{}, err
	}

	var files []database.MediaFile
	if err := s.db.WithContext(ctx).
		Where("media_item_id = ? AND deleted_at IS NULL", mediaItemID).
		Order("storage_path asc").
		Find(&files).Error; err != nil {
		return MediaItemDetail{}, err
	}

	parsedFiles, err := buildMediaFileDetails(files)
	if err != nil {
		return MediaItemDetail{}, err
	}

	genres, err := parseStringList(item.GenresJSON)
	if err != nil {
		return MediaItemDetail{}, err
	}
	cast, err := parsePeopleList(item.CastJSON)
	if err != nil {
		return MediaItemDetail{}, err
	}
	directors, err := parsePeopleList(item.DirectorsJSON)
	if err != nil {
		return MediaItemDetail{}, err
	}

	return MediaItemDetail{
		MediaItem: item,
		Genres:    genres,
		Cast:      cast,
		Directors: directors,
		Files:     parsedFiles,
	}, nil
}

func buildMediaFileDetails(files []database.MediaFile) ([]MediaFileDetail, error) {
	result := make([]MediaFileDetail, 0, len(files))
	for _, file := range files {
		audioTracks, err := parseTrackList(file.AudioTracksJSON)
		if err != nil {
			return nil, err
		}
		subtitleTracks, err := parseTrackList(file.SubtitleTracksJSON)
		if err != nil {
			return nil, err
		}
		result = append(result, MediaFileDetail{
			MediaFile:      file,
			AudioTracks:    audioTracks,
			SubtitleTracks: subtitleTracks,
		})
	}
	return result, nil
}

func parseStringList(input string) ([]string, error) {
	if input == "" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal([]byte(input), &values); err != nil {
		return nil, err
	}
	return values, nil
}

func parsePeopleList(input string) ([]PersonDetail, error) {
	if input == "" {
		return nil, nil
	}

	var people []PersonDetail
	if err := json.Unmarshal([]byte(input), &people); err == nil {
		return people, nil
	}

	var names []string
	if err := json.Unmarshal([]byte(input), &names); err != nil {
		return nil, err
	}

	people = make([]PersonDetail, 0, len(names))
	for _, name := range names {
		people = append(people, PersonDetail{Name: name})
	}
	return people, nil
}

func parseTrackList(input string) ([]TrackDetail, error) {
	if input == "" {
		return nil, nil
	}
	var values []TrackDetail
	if err := json.Unmarshal([]byte(input), &values); err != nil {
		return nil, err
	}
	return values, nil
}
