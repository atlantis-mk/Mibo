package metadata

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/library"
)

func imageURL(cfg config.TMDBConfig, imagePath string) string {
	trimmed := strings.TrimSpace(imagePath)
	if trimmed == "" {
		return ""
	}
	return cfg.ImageBaseURL + "/" + strings.TrimLeft(trimmed, "/")
}

func imageLanguages(language string) string {
	trimmed := strings.TrimSpace(language)
	if trimmed == "" {
		return "en,null"
	}
	base := trimmed
	if idx := strings.Index(trimmed, "-"); idx > 0 {
		base = trimmed[:idx]
	}
	if base == trimmed {
		return trimmed + ",null,en"
	}
	return trimmed + "," + base + ",null,en"
}

func pickLogoPath(language string, logos []imageAsset) string {
	if len(logos) == 0 {
		return ""
	}
	trimmed := strings.TrimSpace(language)
	base := trimmed
	if idx := strings.Index(trimmed, "-"); idx > 0 {
		base = trimmed[:idx]
	}
	rank := func(asset imageAsset) int {
		lang := strings.TrimSpace(asset.Language)
		switch {
		case trimmed != "" && lang == trimmed:
			return 0
		case base != "" && lang == base:
			return 1
		case lang == "":
			return 2
		case lang == "en":
			return 3
		default:
			return 4
		}
	}
	best := logos[0]
	bestRank := rank(best)
	for _, logo := range logos[1:] {
		currentRank := rank(logo)
		if currentRank < bestRank || (currentRank == bestRank && logo.VoteAverage > best.VoteAverage) {
			best = logo
			bestRank = currentRank
		}
	}
	return best.FilePath
}

func extractNamedValues(values []namedValue, max int) []string {
	limit := len(values)
	if max > 0 && limit > max {
		limit = max
	}
	result := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		name := strings.TrimSpace(values[i].Name)
		if name != "" {
			result = append(result, name)
		}
	}
	return result
}

func extractCast(detail detailResponse, cfg config.TMDBConfig, max int) []library.PersonDetail {
	limit := len(detail.Credits.Cast)
	if max > 0 && limit > max {
		limit = max
	}
	result := make([]library.PersonDetail, 0, limit)
	for i := 0; i < limit; i++ {
		member := detail.Credits.Cast[i]
		name := strings.TrimSpace(member.Name)
		if name == "" {
			continue
		}
		result = append(result, library.PersonDetail{Name: name, Role: strings.TrimSpace(member.Character), AvatarURL: imageURL(cfg, member.ProfilePath), TMDBPersonID: intPointerIfPositive(member.ID)})
	}
	return result
}

func extractDirectors(detail detailResponse, cfg config.TMDBConfig) []library.PersonDetail {
	if len(detail.Credits.Crew) > 0 {
		result := make([]library.PersonDetail, 0, 4)
		for _, member := range detail.Credits.Crew {
			if member.Job == "Director" || member.Department == "Directing" {
				name := strings.TrimSpace(member.Name)
				if name == "" {
					continue
				}
				result = append(result, library.PersonDetail{Name: name, Role: strings.TrimSpace(member.Job), AvatarURL: imageURL(cfg, member.ProfilePath), TMDBPersonID: intPointerIfPositive(member.ID)})
				if len(result) == 4 {
					return result
				}
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	fallback := extractNamedValues(detail.CreatedBy, 4)
	result := make([]library.PersonDetail, 0, len(fallback))
	for _, name := range fallback {
		result = append(result, library.PersonDetail{Name: name, Role: "Creator"})
	}
	return result
}

func extractEpisodeCast(episode seasonEpisodeResponse, cfg config.TMDBConfig, max int) []library.PersonDetail {
	limit := len(episode.GuestStars)
	if max > 0 && limit > max {
		limit = max
	}
	result := make([]library.PersonDetail, 0, limit)
	for i := 0; i < limit; i++ {
		member := episode.GuestStars[i]
		name := strings.TrimSpace(member.Name)
		if name == "" {
			continue
		}
		result = append(result, library.PersonDetail{Name: name, Role: strings.TrimSpace(member.Character), AvatarURL: imageURL(cfg, member.ProfilePath), TMDBPersonID: intPointerIfPositive(member.ID)})
	}
	return result
}

func extractEpisodeDirectors(episode seasonEpisodeResponse, cfg config.TMDBConfig) []library.PersonDetail {
	result := make([]library.PersonDetail, 0, 2)
	for _, member := range episode.Crew {
		if member.Job != "Director" && member.Department != "Directing" {
			continue
		}
		name := strings.TrimSpace(member.Name)
		if name == "" {
			continue
		}
		result = append(result, library.PersonDetail{Name: name, Role: strings.TrimSpace(member.Job), AvatarURL: imageURL(cfg, member.ProfilePath), TMDBPersonID: intPointerIfPositive(member.ID)})
		if len(result) == 4 {
			return result
		}
	}
	return result
}

func intPointerIfPositive(value int) *int {
	if value <= 0 {
		return nil
	}
	result := value
	return &result
}

func parseProviderDate(input string) *time.Time {
	trimmed := strings.TrimSpace(input)
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

func runtimeFromDetail(detail detailResponse) *int {
	if detail.Runtime != nil && *detail.Runtime > 0 {
		seconds := *detail.Runtime * 60
		return &seconds
	}
	if len(detail.EpisodeRunTime) > 0 && detail.EpisodeRunTime[0] > 0 {
		seconds := detail.EpisodeRunTime[0] * 60
		return &seconds
	}
	return nil
}

func runtimeSecondsFromMinutes(minutes *int) *int {
	if minutes == nil || *minutes <= 0 {
		return nil
	}
	seconds := *minutes * 60
	return &seconds
}

func marshalPayload(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func marshalStringSlice(values []string) (string, error) {
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func marshalPeople(values []library.PersonDetail) (string, error) {
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func marshalTrailer(value *library.TrailerDetail) (string, error) {
	if value == nil {
		return "", nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func selectTrailer(detail detailResponse) *videoAsset {
	playable := make([]videoAsset, 0, len(detail.Videos.Results))
	for _, candidate := range detail.Videos.Results {
		watchURL, embedURL := trailerSiteURLs(candidate.Site, candidate.Key)
		if watchURL == "" || embedURL == "" {
			continue
		}
		playable = append(playable, candidate)
	}
	if len(playable) == 0 {
		return nil
	}

	sort.SliceStable(playable, func(i, j int) bool {
		left := trailerPriority(playable[i])
		right := trailerPriority(playable[j])
		if left != right {
			return left < right
		}
		return false
	})
	best := playable[0]
	return &best
}

func trailerPriority(video videoAsset) int {
	videoType := strings.ToLower(strings.TrimSpace(video.Type))
	switch {
	case video.Official && videoType == "trailer":
		return 0
	case videoType == "trailer":
		return 1
	case videoType == "teaser":
		return 2
	default:
		return 3
	}
}

func buildTrailerDetail(detail detailResponse) *library.TrailerDetail {
	selected := selectTrailer(detail)
	if selected == nil {
		return nil
	}
	watchURL, embedURL := trailerSiteURLs(selected.Site, selected.Key)
	if watchURL == "" || embedURL == "" {
		return nil
	}
	thumbnail := trailerThumbnailURL(selected.Site, selected.Key)
	return &library.TrailerDetail{
		Provider:  "tmdb",
		Site:      strings.TrimSpace(selected.Site),
		Key:       strings.TrimSpace(selected.Key),
		Name:      strings.TrimSpace(selected.Name),
		Type:      strings.TrimSpace(selected.Type),
		Official:  selected.Official,
		Language:  strings.TrimSpace(selected.Language),
		WatchURL:  watchURL,
		EmbedURL:  embedURL,
		Thumbnail: thumbnail,
	}
}

func trailerSiteURLs(site, key string) (string, string) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return "", ""
	}
	switch strings.ToLower(strings.TrimSpace(site)) {
	case "youtube":
		return "https://www.youtube.com/watch?v=" + trimmedKey, "https://www.youtube.com/embed/" + trimmedKey
	default:
		return "", ""
	}
}

func trailerThumbnailURL(site, key string) string {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(site)) {
	case "youtube":
		return "https://img.youtube.com/vi/" + trimmedKey + "/hqdefault.jpg"
	default:
		return ""
	}
}

func parseYear(input string) *int {
	if len(input) < 4 {
		return nil
	}
	value, err := strconv.Atoi(input[:4])
	if err != nil {
		return nil
	}
	return &value
}

func cloneValues(values url.Values) url.Values {
	if values == nil {
		return url.Values{}
	}
	result := make(url.Values, len(values))
	for key, list := range values {
		copied := make([]string, len(list))
		copy(copied, list)
		result[key] = copied
	}
	return result
}

func tmdbMediaType(itemType string) string {
	if itemType == "episode" {
		return "tv"
	}
	return "movie"
}

func searchResultToCandidate(cfg config.TMDBConfig, mediaType string, candidate scoredMatchCandidate) SearchCandidate {
	result := candidate.result
	title := result.Title
	originalTitle := result.OriginalTitle
	releaseDate := result.ReleaseDate
	if mediaType == "tv" {
		title = result.Name
		originalTitle = result.OriginalName
		releaseDate = result.FirstAirDate
	}
	return SearchCandidate{Provider: "tmdb", MediaType: mediaType, ExternalID: mediaType + ":" + strconv.Itoa(result.ID), Title: title, OriginalTitle: originalTitle, Overview: result.Overview, PosterURL: imageURL(cfg, result.PosterPath), BackdropURL: imageURL(cfg, result.BackdropPath), ReleaseDate: releaseDate, Year: parseYear(releaseDate), Confidence: candidate.confidence, MatchedQuery: candidate.matchedQuery, ReasonSummary: candidate.reasonSummary}
}

func detailToCandidate(cfg config.TMDBConfig, mediaType string, detail detailResponse, confidence float64) SearchCandidate {
	title := detail.Title
	originalTitle := detail.OriginalTitle
	releaseDate := detail.ReleaseDate
	if mediaType == "tv" {
		title = detail.Name
		originalTitle = detail.OriginalName
		releaseDate = detail.FirstAirDate
	}
	return SearchCandidate{Provider: "tmdb", MediaType: mediaType, ExternalID: mediaType + ":" + strconv.Itoa(detail.ID), Title: title, OriginalTitle: originalTitle, Overview: detail.Overview, PosterURL: imageURL(cfg, detail.PosterPath), BackdropURL: imageURL(cfg, detail.BackdropPath), ReleaseDate: releaseDate, Year: parseYear(releaseDate), Confidence: confidence}
}

func parseExternalID(value string) (string, int, error) {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("external_id 格式无效")
	}
	mediaType := strings.TrimSpace(parts[0])
	if mediaType != "movie" && mediaType != "tv" {
		return "", 0, fmt.Errorf("external_id 媒体类型无效")
	}
	id, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || id <= 0 {
		return "", 0, fmt.Errorf("external_id 标识无效")
	}
	return mediaType, id, nil
}
