package metadata

import (
	"context"
	"strings"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/settings"
)

func (s *Service) executeTMDBSearchStage(ctx context.Context, provider settings.ResolvedMetadataProviderInstance, mediaType string, queries []matchSearchQuery, item metadataSearchItem) ([]NormalizedMetadataCandidate, MetadataProviderAttempt, error) {
	attempt := metadataProviderAttemptForProvider("search", provider, ProviderAttemptOutcomeNoResult)
	candidates, err := s.collectSearchCandidates(ctx, provider.TMDB, mediaType, queries, item)
	if err != nil {
		attempt = metadataProviderFailureAttempt("search", provider, err)
		return nil, attempt, err
	}
	attempt.CandidateCount = len(candidates)
	if len(candidates) == 0 {
		return nil, attempt, nil
	}
	attempt.Outcome = ProviderAttemptOutcomeSuccess
	attempt.Selected = true
	results := make([]NormalizedMetadataCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		searchCandidate := searchResultToCandidate(provider.TMDB, mediaType, candidate)
		results = append(results, NormalizedMetadataCandidate{Provider: searchCandidate.Provider, ProviderType: searchCandidate.MediaType, ExternalID: searchCandidate.ExternalID, Title: searchCandidate.Title, OriginalTitle: searchCandidate.OriginalTitle, Overview: searchCandidate.Overview, ReleaseDate: searchCandidate.ReleaseDate, Year: searchCandidate.Year, PosterURL: searchCandidate.PosterURL, BackdropURL: searchCandidate.BackdropURL, Confidence: searchCandidate.Confidence, MatchedQuery: searchCandidate.MatchedQuery, ReasonSummary: searchCandidate.ReasonSummary})
	}
	return results, attempt, nil
}

func (s *Service) executeTMDBDetailStage(ctx context.Context, provider settings.ResolvedMetadataProviderInstance, mediaType string, tmdbID int) (NormalizedMetadataDetail, MetadataProviderAttempt, error) {
	attempt := metadataProviderAttemptForProvider("detail", provider, ProviderAttemptOutcomeSuccess)
	detail, err := s.fetchDetail(ctx, provider.TMDB, mediaType, tmdbID)
	if err != nil {
		attempt = metadataProviderFailureAttempt("detail", provider, err)
		return NormalizedMetadataDetail{}, attempt, err
	}
	return normalizeTMDBDetail(provider.TMDB, mediaType, detail), attempt, nil
}

func normalizeTMDBDetail(cfg config.TMDBConfig, mediaType string, detail detailResponse) NormalizedMetadataDetail {
	title := strings.TrimSpace(detail.Title)
	originalTitle := strings.TrimSpace(detail.OriginalTitle)
	releaseDate := strings.TrimSpace(detail.ReleaseDate)
	if mediaType == "tv" {
		title = strings.TrimSpace(detail.Name)
		originalTitle = strings.TrimSpace(detail.OriginalName)
		releaseDate = strings.TrimSpace(detail.FirstAirDate)
	}
	externalID := providerExternalID(mediaType, detail.ID)
	normalized := NormalizedMetadataDetail{Provider: "tmdb", ProviderType: mediaType, ExternalID: externalID, Title: title, OriginalTitle: originalTitle, Overview: strings.TrimSpace(detail.Overview), ReleaseDate: releaseDate, FirstAirDate: strings.TrimSpace(detail.FirstAirDate), LastAirDate: strings.TrimSpace(detail.LastAirDate), Year: parseYear(releaseDate), RuntimeSeconds: runtimeFromDetail(detail), CommunityRating: communityRatingFromDetail(detail), OfficialRating: officialRatingFromDetail(cfg.Language, mediaType, detail), SeriesStatus: strings.TrimSpace(detail.Status), ExternalIDs: tmdbExternalIDs(mediaType, externalID, detail)}
	normalized.Tags = append(normalized.Tags, tmdbTags("genre", detail.Genres)...)
	normalized.Tags = append(normalized.Tags, tmdbTags("keyword", tmdbKeywordValues(detail.Keywords))...)
	normalized.Images = append(normalized.Images, NormalizedMetadataImage{ImageType: "poster", URL: imageURL(cfg, detail.PosterPath), Selected: true})
	normalized.Images = append(normalized.Images, NormalizedMetadataImage{ImageType: "backdrop", URL: imageURL(cfg, detail.BackdropPath), Selected: true})
	if logoPath := pickLogoPath(cfg.Language, detail.Images.Logos); logoPath != "" {
		normalized.Images = append(normalized.Images, NormalizedMetadataImage{ImageType: "logo", URL: imageURL(cfg, logoPath), Selected: true})
	}
	for index, person := range extractCast(detail, cfg, 8) {
		normalized.People = append(normalized.People, NormalizedMetadataPerson{Name: person.Name, Role: "actor", Character: person.Role, SortOrder: index, AvatarURL: person.AvatarURL, TMDBPersonID: person.TMDBPersonID})
	}
	for index, person := range extractDirectors(detail, cfg) {
		normalized.People = append(normalized.People, NormalizedMetadataPerson{Name: person.Name, Role: "director", Character: person.Role, SortOrder: index, AvatarURL: person.AvatarURL, TMDBPersonID: person.TMDBPersonID})
	}
	if mediaType == "tv" {
		hierarchy := NormalizedMetadataHierarchy{Seasons: make([]NormalizedMetadataSeason, 0, len(detail.Seasons))}
		for _, season := range detail.Seasons {
			hierarchy.Seasons = append(hierarchy.Seasons, NormalizedMetadataSeason{SeriesExternalID: externalID, ProviderType: "tv_season", ExternalID: tmdbExternalID(season.ID), SeasonNumber: season.SeasonNumber, Title: strings.TrimSpace(season.Name), Overview: strings.TrimSpace(season.Overview), PosterURL: imageURL(cfg, season.PosterPath), PosterPath: strings.TrimSpace(season.PosterPath)})
		}
		normalized.Hierarchy = &hierarchy
	}
	return normalized
}

func hierarchyProviderAttempt(provider settings.ResolvedMetadataProviderInstance, hierarchy *NormalizedMetadataHierarchy) MetadataProviderAttempt {
	attempt := metadataProviderAttemptForProvider("hierarchy", provider, ProviderAttemptOutcomeNoResult)
	if hierarchy != nil && len(hierarchy.Seasons) > 0 {
		attempt.Outcome = ProviderAttemptOutcomeSuccess
		attempt.CandidateCount = len(hierarchy.Seasons)
		attempt.Selected = true
	}
	return attempt
}
