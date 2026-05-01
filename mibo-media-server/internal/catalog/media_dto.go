package catalog

import (
	"context"
	"strconv"
	"strings"
	"time"
)

const runtimeTickFactor = int64(10000000)

type MediaItemDTO struct {
	Name               string            `json:"Name"`
	ServerID           string            `json:"ServerId,omitempty"`
	ID                 string            `json:"Id"`
	Type               string            `json:"Type"`
	MediaType          string            `json:"MediaType,omitempty"`
	Path               string            `json:"Path,omitempty"`
	SeriesID           string            `json:"SeriesId,omitempty"`
	SeriesName         string            `json:"SeriesName,omitempty"`
	SeasonID           string            `json:"SeasonId,omitempty"`
	SeasonName         string            `json:"SeasonName,omitempty"`
	IndexNumber        *int              `json:"IndexNumber,omitempty"`
	ParentIndexNumber  *int              `json:"ParentIndexNumber,omitempty"`
	ProductionYear     *int              `json:"ProductionYear,omitempty"`
	PremiereDate       string            `json:"PremiereDate,omitempty"`
	EndDate            string            `json:"EndDate,omitempty"`
	Status             string            `json:"Status,omitempty"`
	Overview           string            `json:"Overview,omitempty"`
	CommunityRating    *float64          `json:"CommunityRating,omitempty"`
	OfficialRating     string            `json:"OfficialRating,omitempty"`
	RunTimeTicks       *int64            `json:"RunTimeTicks,omitempty"`
	Genres             []string          `json:"Genres,omitempty"`
	Tags               []string          `json:"Tags,omitempty"`
	ProviderIDs        map[string]string `json:"ProviderIds,omitempty"`
	People             []MediaPersonDTO  `json:"People,omitempty"`
	ImageTags          map[string]string `json:"ImageTags,omitempty"`
	BackdropImageTags  []string          `json:"BackdropImageTags,omitempty"`
	RecursiveItemCount int               `json:"RecursiveItemCount,omitempty"`
	ChildCount         int               `json:"ChildCount,omitempty"`
	MediaSources       []MediaSourceDTO  `json:"MediaSources,omitempty"`
}

type MediaPersonDTO struct {
	Name string `json:"Name"`
	Type string `json:"Type"`
	Role string `json:"Role,omitempty"`
	ID   string `json:"Id,omitempty"`
}

type MediaSourceDTO struct {
	ID                         string           `json:"Id"`
	Path                       string           `json:"Path,omitempty"`
	Protocol                   string           `json:"Protocol"`
	Container                  string           `json:"Container,omitempty"`
	Size                       int64            `json:"Size,omitempty"`
	Name                       string           `json:"Name,omitempty"`
	IsRemote                   bool             `json:"IsRemote"`
	RunTimeTicks               *int64           `json:"RunTimeTicks,omitempty"`
	VideoType                  string           `json:"VideoType,omitempty"`
	DefaultAudioStreamIndex    *int             `json:"DefaultAudioStreamIndex,omitempty"`
	DefaultSubtitleStreamIndex *int             `json:"DefaultSubtitleStreamIndex,omitempty"`
	MediaStreams               []MediaStreamDTO `json:"MediaStreams,omitempty"`
}

type MediaStreamDTO struct {
	Index            int      `json:"Index"`
	Type             string   `json:"Type"`
	Codec            string   `json:"Codec,omitempty"`
	Language         string   `json:"Language,omitempty"`
	Title            string   `json:"Title,omitempty"`
	Width            *int     `json:"Width,omitempty"`
	Height           *int     `json:"Height,omitempty"`
	BitRate          *int64   `json:"BitRate,omitempty"`
	AverageFrameRate *float64 `json:"AverageFrameRate,omitempty"`
	RealFrameRate    *float64 `json:"RealFrameRate,omitempty"`
	Profile          string   `json:"Profile,omitempty"`
	Level            *int     `json:"Level,omitempty"`
	IsInterlaced     bool     `json:"IsInterlaced,omitempty"`
	Channels         *int     `json:"Channels,omitempty"`
	SampleRate       *int     `json:"SampleRate,omitempty"`
	IsDefault        bool     `json:"IsDefault,omitempty"`
	IsForced         bool     `json:"IsForced,omitempty"`
	IsExternal       bool     `json:"IsExternal,omitempty"`
	Path             string   `json:"Path,omitempty"`
}

func (s *Service) GetMediaItemDTO(ctx context.Context, itemID uint, userID *uint) (MediaItemDTO, error) {
	detail, err := s.GetItemDetailForUser(ctx, itemID, userID)
	if err != nil {
		return MediaItemDTO{}, err
	}
	dto := BuildMediaItemDTO(detail)
	if detail.Type == ItemTypeSeason {
		item, err := s.loadCatalogItem(ctx, detail.ID)
		if err != nil {
			return MediaItemDTO{}, err
		}
		if item.ParentID != nil {
			series, err := s.loadCatalogItem(ctx, *item.ParentID)
			if err == nil {
				dto.SeriesID = strconv.FormatUint(uint64(series.ID), 10)
				dto.SeriesName = strings.TrimSpace(series.Title)
			}
		}
	}
	return dto, nil
}

func BuildMediaItemDTO(detail CatalogItemDetail) MediaItemDTO {
	dto := MediaItemDTO{
		Name:              strings.TrimSpace(detail.Title),
		ID:                strconv.FormatUint(uint64(detail.ID), 10),
		Type:              mediaDTOType(detail.Type),
		MediaType:         mediaDTOMediaType(detail.Type),
		Path:              mediaDTOPath(detail),
		ProductionYear:    detail.Year,
		PremiereDate:      mediaDTODate(firstNonNilTime(detail.ReleaseDate, detail.FirstAirDate)),
		EndDate:           mediaDTODate(detail.LastAirDate),
		Status:            strings.TrimSpace(detail.SeriesStatus),
		Overview:          detail.Overview,
		CommunityRating:   detail.CommunityRating,
		OfficialRating:    strings.TrimSpace(detail.OfficialRating),
		RunTimeTicks:      runtimeTicksFromSecondsPtr(detail.RuntimeSeconds),
		Genres:            append([]string(nil), detail.Genres...),
		Tags:              mediaDTOTags(detail.Tags),
		ProviderIDs:       mediaDTOProviderIDs(detail.ExternalIdentities),
		People:            mediaDTOPeople(detail.Cast, detail.Directors),
		ImageTags:         mediaDTOImageTags(detail.SelectedImages),
		BackdropImageTags: mediaDTOBackdropTags(detail.SelectedImages),
		MediaSources:      mediaDTOMediaSources(detail.Assets, detail.RuntimeSeconds),
	}
	if detail.ChildSummary != nil {
		dto.ChildCount = detail.ChildSummary.ChildCount
		dto.RecursiveItemCount = detail.ChildSummary.AvailableCount + detail.ChildSummary.MissingCount + detail.ChildSummary.UnairedCount
	}
	if detail.Type == ItemTypeEpisode && detail.EpisodeContext != nil {
		if detail.EpisodeContext.Series != nil {
			dto.SeriesID = strconv.FormatUint(uint64(detail.EpisodeContext.Series.ID), 10)
			dto.SeriesName = detail.EpisodeContext.Series.Title
		}
		if detail.EpisodeContext.Season != nil {
			dto.SeasonID = strconv.FormatUint(uint64(detail.EpisodeContext.Season.ID), 10)
			dto.SeasonName = detail.EpisodeContext.Season.Title
		}
		if detail.EpisodeContext.SeasonNumber != nil {
			dto.ParentIndexNumber = detail.EpisodeContext.SeasonNumber
		}
		if detail.EpisodeContext.EpisodeNumber != nil {
			dto.IndexNumber = detail.EpisodeContext.EpisodeNumber
		}
	}
	if detail.Type == ItemTypeSeason {
		dto.IndexNumber = detail.IndexNumber
	}
	return dto
}

func mediaDTOType(itemType string) string {
	switch itemType {
	case ItemTypeMovie:
		return "Movie"
	case ItemTypeSeries:
		return "Series"
	case ItemTypeSeason:
		return "Season"
	case ItemTypeEpisode:
		return "Episode"
	default:
		return strings.Title(strings.TrimSpace(itemType))
	}
}

func mediaDTOMediaType(itemType string) string {
	switch itemType {
	case ItemTypeMovie, ItemTypeSeries, ItemTypeEpisode:
		return "Video"
	default:
		return ""
	}
}

func mediaDTOPath(detail CatalogItemDetail) string {
	if detail.Type == ItemTypeEpisode && len(detail.Assets) > 0 && len(detail.Assets[0].Files) > 0 {
		return strings.TrimSpace(detail.Assets[0].Files[0].StoragePath)
	}
	return strings.TrimSpace(detail.Path)
}

func firstNonNilTime(values ...*time.Time) *time.Time {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func mediaDTODate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format("2006-01-02T15:04:05.0000000Z")
}

func runtimeTicksFromSecondsPtr(seconds *int) *int64 {
	if seconds == nil {
		return nil
	}
	value := int64(*seconds) * runtimeTickFactor
	return &value
}

func runtimeTicksFromFloatSeconds(seconds *float64) *int64 {
	if seconds == nil {
		return nil
	}
	value := int64(*seconds * float64(runtimeTickFactor))
	return &value
}

func mediaDTOProviderIDs(identities []CatalogExternalIdentity) map[string]string {
	ids := map[string]string{}
	for _, identity := range identities {
		key := providerIDKey(identity.Provider)
		if key == "" || strings.TrimSpace(identity.ExternalID) == "" {
			continue
		}
		ids[key] = strings.TrimSpace(strings.TrimPrefix(identity.ExternalID, identity.ProviderType+":"))
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
}

func providerIDKey(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "imdb":
		return "Imdb"
	case "tmdb":
		return "Tmdb"
	case "tvdb":
		return "Tvdb"
	default:
		return strings.TrimSpace(provider)
	}
}

func mediaDTOPeople(cast []CatalogPersonDetail, directors []CatalogPersonDetail) []MediaPersonDTO {
	people := make([]MediaPersonDTO, 0, len(directors)+len(cast))
	for _, director := range directors {
		people = append(people, MediaPersonDTO{Name: director.Name, Type: "Director", ID: mediaDTOID(director.ID)})
	}
	for _, actor := range cast {
		people = append(people, MediaPersonDTO{Name: actor.Name, Type: "Actor", Role: actor.Role, ID: mediaDTOID(actor.ID)})
	}
	return people
}

func mediaDTOID(id uint) string {
	if id == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(id), 10)
}

func mediaDTOImageTags(images []CatalogSelectedImage) map[string]string {
	tags := map[string]string{}
	for _, image := range images {
		switch strings.ToLower(strings.TrimSpace(image.ImageType)) {
		case "poster", "primary":
			tags["Primary"] = image.URL
		case "logo":
			tags["Logo"] = image.URL
		case "still":
			tags["Primary"] = image.URL
		}
	}
	if len(tags) == 0 {
		return nil
	}
	return tags
}

func mediaDTOBackdropTags(images []CatalogSelectedImage) []string {
	tags := []string{}
	for _, image := range images {
		if strings.EqualFold(strings.TrimSpace(image.ImageType), "backdrop") {
			tags = append(tags, image.URL)
		}
	}
	return tags
}

func mediaDTOTags(tags []CatalogTagDetail) []string {
	items := []string{}
	for _, tag := range tags {
		if !strings.EqualFold(strings.TrimSpace(tag.Kind), "genre") && strings.TrimSpace(tag.Name) != "" {
			items = append(items, tag.Name)
		}
	}
	return items
}

func mediaDTOMediaSources(assets []CatalogAssetDetail, itemRuntime *int) []MediaSourceDTO {
	sources := make([]MediaSourceDTO, 0, len(assets))
	for _, asset := range assets {
		if len(asset.Files) == 0 {
			continue
		}
		file := asset.Files[0]
		runtimeTicks := runtimeTicksFromFloatSeconds(asset.DurationSeconds)
		if runtimeTicks == nil {
			runtimeTicks = runtimeTicksFromSecondsPtr(itemRuntime)
		}
		source := MediaSourceDTO{ID: strconv.FormatUint(uint64(asset.ID), 10), Path: file.StoragePath, Protocol: "File", Container: file.Container, Size: file.SizeBytes, Name: mediaSourceName(asset), IsRemote: false, RunTimeTicks: runtimeTicks, VideoType: "VideoFile", MediaStreams: mediaDTOStreams(asset.Streams)}
		source.DefaultAudioStreamIndex = defaultStreamIndex(source.MediaStreams, "Audio")
		source.DefaultSubtitleStreamIndex = defaultStreamIndex(source.MediaStreams, "Subtitle")
		sources = append(sources, source)
	}
	return sources
}

func mediaSourceName(asset CatalogAssetDetail) string {
	if strings.TrimSpace(asset.QualityLabel) != "" {
		return strings.TrimSpace(asset.QualityLabel)
	}
	if strings.TrimSpace(asset.DisplayName) != "" {
		return strings.TrimSpace(asset.DisplayName)
	}
	return strings.TrimSpace(asset.AssetType)
}

func mediaDTOStreams(streams []CatalogMediaStreamSummary) []MediaStreamDTO {
	items := make([]MediaStreamDTO, 0, len(streams))
	for _, stream := range streams {
		items = append(items, MediaStreamDTO{Index: stream.StreamIndex, Type: mediaDTOStreamType(stream.StreamType), Codec: stream.Codec, Language: stream.Language, Title: stream.Title, Width: stream.Width, Height: stream.Height, BitRate: stream.BitRate, AverageFrameRate: parseFrameRate(stream.AvgFrameRate), RealFrameRate: parseFrameRate(stream.RFrameRate), Profile: stream.Profile, Level: stream.Level, IsInterlaced: isInterlaced(stream.FieldOrder), Channels: stream.Channels, SampleRate: stream.SampleRate, IsDefault: stream.Default, IsForced: stream.Forced, IsExternal: stream.External, Path: stream.URL})
	}
	return items
}

func mediaDTOStreamType(streamType string) string {
	switch strings.ToLower(strings.TrimSpace(streamType)) {
	case "video":
		return "Video"
	case "audio":
		return "Audio"
	case "subtitle":
		return "Subtitle"
	default:
		return strings.Title(strings.TrimSpace(streamType))
	}
}

func parseFrameRate(value string) *float64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if strings.Contains(trimmed, "/") {
		parts := strings.SplitN(trimmed, "/", 2)
		left, leftErr := strconv.ParseFloat(parts[0], 64)
		right, rightErr := strconv.ParseFloat(parts[1], 64)
		if leftErr == nil && rightErr == nil && right != 0 {
			result := left / right
			return &result
		}
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func isInterlaced(fieldOrder string) bool {
	lower := strings.ToLower(strings.TrimSpace(fieldOrder))
	return lower != "" && lower != "progressive" && lower != "unknown"
}

func defaultStreamIndex(streams []MediaStreamDTO, streamType string) *int {
	for _, stream := range streams {
		if stream.Type == streamType && stream.IsDefault {
			idx := stream.Index
			return &idx
		}
	}
	return nil
}
