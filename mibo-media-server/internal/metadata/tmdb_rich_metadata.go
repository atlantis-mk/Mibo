package metadata

import (
	"fmt"
	"strings"
)

func communityRatingFromDetail(detail detailResponse) *float64 {
	if detail.VoteAverage <= 0 {
		return nil
	}
	rating := detail.VoteAverage
	return &rating
}

func officialRatingFromDetail(language string, mediaType string, detail detailResponse) string {
	if strings.TrimSpace(mediaType) == "tv" {
		return pickContentRating(language, detail.ContentRatings.Results)
	}
	return pickReleaseDateCertification(language, detail.ReleaseDates.Results)
}

func tmdbExternalIDs(mediaType string, primaryExternalID string, detail detailResponse) []NormalizedMetadataExternalID {
	ids := []NormalizedMetadataExternalID{{Provider: "tmdb", ProviderType: mediaType, ExternalID: primaryExternalID, IsPrimary: true}}
	if imdbID := strings.TrimSpace(detail.ExternalIDs.IMDbID); imdbID != "" {
		ids = append(ids, NormalizedMetadataExternalID{Provider: "imdb", ProviderType: mediaType, ExternalID: imdbID})
	}
	if detail.ExternalIDs.TVDBID > 0 {
		ids = append(ids, NormalizedMetadataExternalID{Provider: "tvdb", ProviderType: mediaType, ExternalID: fmt.Sprintf("%d", detail.ExternalIDs.TVDBID)})
	}
	if wikidataID := strings.TrimSpace(detail.ExternalIDs.WikidataID); wikidataID != "" {
		ids = append(ids, NormalizedMetadataExternalID{Provider: "wikidata", ProviderType: mediaType, ExternalID: wikidataID})
	}
	return ids
}

func tmdbSeasonExternalIDs(primaryExternalID string, detail seasonDetailResponse) []NormalizedMetadataExternalID {
	ids := []NormalizedMetadataExternalID{{Provider: "tmdb", ProviderType: "tv_season", ExternalID: primaryExternalID, IsPrimary: true}}
	if imdbID := strings.TrimSpace(detail.ExternalIDs.IMDbID); imdbID != "" {
		ids = append(ids, NormalizedMetadataExternalID{Provider: "imdb", ProviderType: "tv_season", ExternalID: imdbID})
	}
	if tvdbID := detail.ExternalIDs.TVDBID; tvdbID > 0 {
		ids = append(ids, NormalizedMetadataExternalID{Provider: "tvdb", ProviderType: "tv_season", ExternalID: fmt.Sprintf("%d", tvdbID)})
	}
	if wikidataID := strings.TrimSpace(detail.ExternalIDs.WikidataID); wikidataID != "" {
		ids = append(ids, NormalizedMetadataExternalID{Provider: "wikidata", ProviderType: "tv_season", ExternalID: wikidataID})
	}
	return ids
}

func communityRatingFromEpisode(episode seasonEpisodeResponse) *float64 {
	if episode.VoteAverage <= 0 {
		return nil
	}
	rating := episode.VoteAverage
	return &rating
}

func tmdbTags(kind string, values []namedValue) []NormalizedMetadataTag {
	result := make([]NormalizedMetadataTag, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		name := strings.TrimSpace(value.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, NormalizedMetadataTag{Kind: kind, Name: name})
	}
	return result
}

func tmdbKeywordValues(response keywordsResponse) []namedValue {
	if len(response.Keywords) > 0 {
		return response.Keywords
	}
	return response.Results
}

func pickReleaseDateCertification(language string, regions []releaseDateRegion) string {
	preferred := preferredCertificationRegions(language)
	for _, region := range preferred {
		for _, candidate := range regions {
			if !strings.EqualFold(strings.TrimSpace(candidate.Region), region) {
				continue
			}
			if certification := firstReleaseDateCertification(candidate.ReleaseDates); certification != "" {
				return certification
			}
		}
	}
	for _, candidate := range regions {
		if certification := firstReleaseDateCertification(candidate.ReleaseDates); certification != "" {
			return certification
		}
	}
	return ""
}

func firstReleaseDateCertification(values []releaseDateCertification) string {
	for _, value := range values {
		if certification := strings.TrimSpace(value.Certification); certification != "" {
			return certification
		}
	}
	return ""
}

func pickContentRating(language string, ratings []contentRating) string {
	preferred := preferredCertificationRegions(language)
	for _, region := range preferred {
		for _, candidate := range ratings {
			if strings.EqualFold(strings.TrimSpace(candidate.Region), region) {
				if rating := strings.TrimSpace(candidate.Rating); rating != "" {
					return rating
				}
			}
		}
	}
	for _, candidate := range ratings {
		if rating := strings.TrimSpace(candidate.Rating); rating != "" {
			return rating
		}
	}
	return ""
}

func preferredCertificationRegions(language string) []string {
	regions := []string{}
	trimmed := strings.TrimSpace(language)
	if idx := strings.LastIndex(trimmed, "-"); idx >= 0 && idx < len(trimmed)-1 {
		regions = append(regions, strings.ToUpper(trimmed[idx+1:]))
	}
	regions = appendUniqueStringFold(regions, "US")
	return regions
}
