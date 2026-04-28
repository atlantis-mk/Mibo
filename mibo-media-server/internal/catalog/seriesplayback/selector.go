package seriesplayback

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

const (
	itemTypeSeries  = "series"
	itemTypeSeason  = "season"
	itemTypeEpisode = "episode"

	availabilityAvailable = "available"
)

type Target struct {
	EpisodeID uint
	AssetID   *uint
	Title     string
	Label     string
	Reason    string
}

type playableEpisode struct {
	Item    database.CatalogItem
	AssetID uint
}

func Select(ctx context.Context, db *gorm.DB, seriesID uint, userID *uint) (*Target, error) {
	episodes, err := LoadPlayableEpisodes(ctx, db, seriesID)
	if err != nil {
		return nil, err
	}
	if len(episodes) == 0 {
		return nil, nil
	}

	if userID != nil && *userID > 0 {
		progressTarget, err := selectInProgressTarget(ctx, db, episodes, *userID)
		if err != nil {
			return nil, err
		}
		if progressTarget != nil {
			return progressTarget, nil
		}
	}

	episode := episodes[0]
	assetID := episode.AssetID
	return buildTarget(episode.Item, &assetID, "first_available"), nil
}

func LoadPlayableEpisodeIDs(ctx context.Context, db *gorm.DB, seriesID uint) (map[uint]struct{}, error) {
	episodes, err := LoadPlayableEpisodes(ctx, db, seriesID)
	if err != nil {
		return nil, err
	}
	ids := make(map[uint]struct{}, len(episodes))
	for _, episode := range episodes {
		ids[episode.Item.ID] = struct{}{}
	}
	return ids, nil
}

func LoadPlayableEpisodes(ctx context.Context, db *gorm.DB, seriesID uint) ([]playableEpisode, error) {
	var series database.CatalogItem
	if err := db.WithContext(ctx).Where("id = ? AND type = ? AND deleted_at IS NULL", seriesID, itemTypeSeries).First(&series).Error; err != nil {
		return nil, err
	}

	var rows []struct {
		ID                 uint
		LibraryID          uint
		Type               string
		ParentID           *uint
		RootID             *uint
		Path               string
		SortKey            string
		DisplayOrder       string
		IndexNumber        *int
		IndexNumberEnd     *int
		ParentIndexNumber  *int
		AbsoluteNumber     *int
		Title              string
		OriginalTitle      string
		SortTitle          string
		Overview           string
		Year               *int
		EndYear            *int
		RuntimeSeconds     *int
		CommunityRating    *float64
		OfficialRating     string
		SeriesStatus       string
		AvailabilityStatus string
		GovernanceStatus   string
		AssetID            uint
	}
	if err := db.WithContext(ctx).
		Table("catalog_items AS episodes").
		Select(`episodes.id, episodes.library_id, episodes.type, episodes.parent_id, episodes.root_id, episodes.path, episodes.sort_key, episodes.display_order, episodes.index_number, episodes.index_number_end, episodes.parent_index_number, episodes.absolute_number, episodes.title, episodes.original_title, episodes.sort_title, episodes.overview, episodes.year, episodes.end_year, episodes.runtime_seconds, episodes.community_rating, episodes.official_rating, episodes.series_status, episodes.availability_status, episodes.governance_status, asset_items.asset_id`).
		Joins("JOIN catalog_items AS seasons ON seasons.id = episodes.parent_id AND seasons.type = ? AND seasons.parent_id = ? AND seasons.deleted_at IS NULL", itemTypeSeason, series.ID).
		Joins("JOIN asset_items ON asset_items.item_id = episodes.id").
		Joins("JOIN media_assets ON media_assets.id = asset_items.asset_id AND media_assets.status = ? AND media_assets.deleted_at IS NULL", availabilityAvailable).
		Joins("JOIN asset_files ON asset_files.asset_id = media_assets.id AND asset_files.role = ?", "source").
		Joins("JOIN inventory_files ON inventory_files.id = asset_files.file_id AND inventory_files.status = ? AND inventory_files.deleted_at IS NULL", availabilityAvailable).
		Where("episodes.type = ? AND episodes.availability_status = ? AND episodes.deleted_at IS NULL", itemTypeEpisode, availabilityAvailable).
		Order("COALESCE(episodes.parent_index_number, seasons.index_number, 0) asc").
		Order("COALESCE(episodes.index_number, 0) asc").
		Order("episodes.id asc").
		Order("CASE WHEN media_assets.asset_type = 'main' THEN 0 ELSE 1 END asc").
		Order("media_assets.id asc").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]playableEpisode, 0, len(rows))
	seenEpisodes := make(map[uint]struct{}, len(rows))
	for _, row := range rows {
		if _, ok := seenEpisodes[row.ID]; ok {
			continue
		}
		seenEpisodes[row.ID] = struct{}{}
		result = append(result, playableEpisode{
			Item: database.CatalogItem{
				ID:                 row.ID,
				LibraryID:          row.LibraryID,
				Type:               row.Type,
				ParentID:           row.ParentID,
				RootID:             row.RootID,
				Path:               row.Path,
				SortKey:            row.SortKey,
				DisplayOrder:       row.DisplayOrder,
				IndexNumber:        row.IndexNumber,
				IndexNumberEnd:     row.IndexNumberEnd,
				ParentIndexNumber:  row.ParentIndexNumber,
				AbsoluteNumber:     row.AbsoluteNumber,
				Title:              row.Title,
				OriginalTitle:      row.OriginalTitle,
				SortTitle:          row.SortTitle,
				Overview:           row.Overview,
				Year:               row.Year,
				EndYear:            row.EndYear,
				RuntimeSeconds:     row.RuntimeSeconds,
				CommunityRating:    row.CommunityRating,
				OfficialRating:     row.OfficialRating,
				SeriesStatus:       row.SeriesStatus,
				AvailabilityStatus: row.AvailabilityStatus,
				GovernanceStatus:   row.GovernanceStatus,
			},
			AssetID: row.AssetID,
		})
	}
	return result, nil
}

func selectInProgressTarget(ctx context.Context, db *gorm.DB, episodes []playableEpisode, userID uint) (*Target, error) {
	episodeByID := make(map[uint]playableEpisode, len(episodes))
	episodeIDs := make([]uint, 0, len(episodes))
	for _, episode := range episodes {
		episodeByID[episode.Item.ID] = episode
		episodeIDs = append(episodeIDs, episode.Item.ID)
	}

	var rows []database.UserItemData
	if err := db.WithContext(ctx).
		Where("user_id = ? AND item_id IN ? AND completed_at IS NULL AND position_seconds > 0", userID, episodeIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	sort.SliceStable(rows, func(i, j int) bool {
		left := rows[i].UpdatedAt
		right := rows[j].UpdatedAt
		if rows[i].LastPlayedAt != nil {
			left = *rows[i].LastPlayedAt
		}
		if rows[j].LastPlayedAt != nil {
			right = *rows[j].LastPlayedAt
		}
		return left.After(right)
	})
	for _, row := range rows {
		episode, ok := episodeByID[row.ItemID]
		if !ok {
			continue
		}
		assetID := episode.AssetID
		return buildTarget(episode.Item, &assetID, "continue"), nil
	}
	return nil, nil
}

func buildTarget(item database.CatalogItem, assetID *uint, reason string) *Target {
	return &Target{
		EpisodeID: item.ID,
		AssetID:   assetID,
		Title:     strings.TrimSpace(item.Title),
		Label:     formatEpisodeLabel(item.ParentIndexNumber, item.IndexNumber, item.IndexNumberEnd),
		Reason:    reason,
	}
}

func formatEpisodeLabel(seasonNumber *int, episodeNumber *int, episodeNumberEnd *int) string {
	if seasonNumber == nil && episodeNumber == nil {
		return ""
	}
	season := "?"
	if seasonNumber != nil {
		season = fmt.Sprintf("%d", *seasonNumber)
	}
	episode := "?"
	if episodeNumber != nil {
		episode = fmt.Sprintf("%d", *episodeNumber)
	}
	label := fmt.Sprintf("S%s:E%s", season, episode)
	if episodeNumberEnd != nil && episodeNumber != nil && *episodeNumberEnd != *episodeNumber {
		label += fmt.Sprintf("-E%d", *episodeNumberEnd)
	}
	return label
}
