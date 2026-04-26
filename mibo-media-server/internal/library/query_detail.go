package library

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

func (s *Service) GetMediaItem(ctx context.Context, mediaItemID uint) (MediaItemDetail, error) {
	var item database.MediaItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", mediaItemID).First(&item).Error; err != nil {
		return MediaItemDetail{}, err
	}
	var files []database.MediaFile
	if err := s.db.WithContext(ctx).Where("media_item_id = ? AND deleted_at IS NULL", mediaItemID).Order("storage_path asc").Find(&files).Error; err != nil {
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
	trailer, err := parseTrailerDetail(item.TrailerJSON)
	if err != nil {
		return MediaItemDetail{}, err
	}
	return MediaItemDetail{MediaItem: item, SeriesTMDBID: tmdbSeriesIDFromExternalID(item.ExternalID), SeriesTitleDisplay: seriesTitleDisplay(item), DefaultSeasonNumber: defaultSeasonNumber(item), Genres: genres, Cast: cast, Directors: directors, Trailer: trailer, Files: parsedFiles}, nil
}

func tmdbSeriesIDFromExternalID(value string) *int {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 2)
	if len(parts) != 2 || parts[0] != "tv" {
		return nil
	}
	id, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || id <= 0 {
		return nil
	}
	return &id
}

func seriesTitleDisplay(item database.MediaItem) string {
	if seriesTitle := strings.TrimSpace(item.SeriesTitle); seriesTitle != "" {
		return seriesTitle
	}
	return strings.TrimSpace(item.Title)
}

func defaultSeasonNumber(item database.MediaItem) *int {
	if item.SeasonNumber != nil && *item.SeasonNumber >= 0 {
		season := *item.SeasonNumber
		return &season
	}
	return nil
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
		result = append(result, MediaFileDetail{MediaFile: file, AudioTracks: audioTracks, SubtitleTracks: subtitleTracks})
	}
	return result, nil
}

func parseStringList(input string) ([]string, error) {
	if input == "" {
		return []string{}, nil
	}
	var values []string
	if err := json.Unmarshal([]byte(input), &values); err != nil {
		return nil, err
	}
	if values == nil {
		return []string{}, nil
	}
	return values, nil
}

func parsePeopleList(input string) ([]PersonDetail, error) {
	if input == "" {
		return []PersonDetail{}, nil
	}
	var people []PersonDetail
	if err := json.Unmarshal([]byte(input), &people); err == nil {
		if people == nil {
			return []PersonDetail{}, nil
		}
		return people, nil
	}
	var names []string
	if err := json.Unmarshal([]byte(input), &names); err != nil {
		return nil, err
	}
	if names == nil {
		return []PersonDetail{}, nil
	}
	people = make([]PersonDetail, 0, len(names))
	for _, name := range names {
		people = append(people, PersonDetail{Name: name})
	}
	return people, nil
}

func parseTrackList(input string) ([]TrackDetail, error) {
	if input == "" {
		return []TrackDetail{}, nil
	}
	var values []TrackDetail
	if err := json.Unmarshal([]byte(input), &values); err != nil {
		return nil, err
	}
	if values == nil {
		return []TrackDetail{}, nil
	}
	return values, nil
}

func parseTrailerDetail(input string) (*TrailerDetail, error) {
	if input == "" {
		return nil, nil
	}
	var trailer TrailerDetail
	if err := json.Unmarshal([]byte(input), &trailer); err != nil {
		return nil, err
	}
	if strings.TrimSpace(trailer.Key) == "" || strings.TrimSpace(trailer.EmbedURL) == "" {
		return nil, nil
	}
	return &trailer, nil
}
