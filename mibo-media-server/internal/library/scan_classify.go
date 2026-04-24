package library

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
)

func classifyMediaFile(libraryType string, object storage.Object) classifiedMedia {
	fileName := path.Base(object.Path)
	ext := path.Ext(fileName)
	rawTitle := strings.TrimSuffix(fileName, ext)
	normalizedTitle := cleanTitle(rawTitle)
	if groups := episodePattern.FindStringSubmatch(rawTitle); len(groups) > 0 {
		seriesTitle := cleanTitle(groups[1])
		season, episode := parseEpisodeNumbers(groups[2], groups[3], groups[4], groups[5])
		title := fmt.Sprintf("%s S%02dE%02d", seriesTitle, *season, *episode)
		return classifiedMedia{Type: "episode", Title: title, OriginalTitle: rawTitle, SeriesTitle: seriesTitle, SeasonNumber: season, EpisodeNumber: episode, SourcePath: object.Path, Status: "ready"}
	}
	year := parseYear(rawTitle)
	title := normalizedTitle
	if libraryType == "tv" || libraryType == "tvshows" || libraryType == "shows" {
		title = titleFromPath(object.Path)
	}
	return classifiedMedia{Type: "movie", Title: title, OriginalTitle: rawTitle, Year: year, SourcePath: object.Path, Status: "ready"}
}

func isVideoFile(itemPath string) bool {
	_, ok := videoExtensions[strings.ToLower(path.Ext(itemPath))]
	return ok
}

func parseEpisodeNumbers(seasonLeft, episodeLeft, seasonRight, episodeRight string) (*int, *int) {
	seasonValue := seasonLeft
	episodeValue := episodeLeft
	if seasonValue == "" {
		seasonValue = seasonRight
		episodeValue = episodeRight
	}
	season, _ := strconv.Atoi(seasonValue)
	episode, _ := strconv.Atoi(episodeValue)
	return &season, &episode
}

func parseYear(input string) *int {
	match := yearPattern.FindStringSubmatch(input)
	if len(match) < 2 {
		return nil
	}
	value, err := strconv.Atoi(match[1])
	if err != nil {
		return nil
	}
	return &value
}

func titleFromPath(itemPath string) string {
	parent := path.Base(path.Dir(itemPath))
	if parent == "/" || parent == "." || parent == "" {
		return cleanTitle(strings.TrimSuffix(path.Base(itemPath), path.Ext(itemPath)))
	}
	return cleanTitle(parent)
}

func cleanTitle(input string) string {
	replacer := strings.NewReplacer(".", " ", "_", " ")
	cleaned := replacer.Replace(strings.TrimSpace(input))
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	cleaned = yearPattern.ReplaceAllString(cleaned, " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	cleaned = strings.Trim(cleaned, "- ")
	if cleaned == "" {
		return strings.TrimSpace(input)
	}
	return cleaned
}

func mediaItemBaseChanged(item database.MediaItem, classified classifiedMedia) bool {
	return item.Type != classified.Type || !equalIntPointers(item.Year, classified.Year) || !equalIntPointers(item.SeasonNumber, classified.SeasonNumber) || !equalIntPointers(item.EpisodeNumber, classified.EpisodeNumber)
}

func resetMediaItemMetadata(item *database.MediaItem) {
	item.Overview = ""
	item.PosterURL = ""
	item.BackdropURL = ""
	item.GenresJSON = ""
	item.CastJSON = ""
	item.DirectorsJSON = ""
	item.ReleaseDate = ""
	item.RuntimeSeconds = nil
	item.MetadataProvider = ""
	item.ExternalID = ""
	item.MetadataConfidence = nil
}

func resetMediaFileProbe(file *database.MediaFile) {
	file.ProbeError = ""
	file.DurationSeconds = nil
	file.BitRate = nil
	file.Width = nil
	file.Height = nil
	file.VideoCodec = ""
	file.AudioTracksJSON = ""
	file.SubtitleTracksJSON = ""
}

func hasMatchedMetadata(item database.MediaItem) bool {
	return strings.TrimSpace(item.MetadataProvider) != "" || strings.TrimSpace(item.ExternalID) != ""
}

func equalIntPointers(left, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
