package metadata

import (
	"context"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/settings"
)

func (s *Service) executeMetaTubeSearchStage(ctx context.Context, provider settings.ResolvedMetadataProviderInstance, queries []matchSearchQuery, item metadataSearchItem) ([]NormalizedMetadataCandidate, MetadataProviderAttempt, error) {
	attempt := metadataProviderAttemptForProvider("search", provider, ProviderAttemptOutcomeNoResult)
	candidates, err := s.searchMetaTubeCandidates(ctx, provider.MetaTube, queries, item)
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
		results = append(results, NormalizedMetadataCandidate{Provider: candidate.Provider, ProviderType: candidate.MediaType, ExternalID: candidate.ExternalID, Title: candidate.Title, OriginalTitle: candidate.OriginalTitle, Overview: candidate.Overview, ReleaseDate: candidate.ReleaseDate, Year: candidate.Year, PosterURL: candidate.PosterURL, BackdropURL: candidate.BackdropURL, Confidence: candidate.Confidence, MatchedQuery: candidate.MatchedQuery, ReasonSummary: candidate.ReasonSummary})
	}
	return results, attempt, nil
}

func (s *Service) executeMetaTubeDetailStage(ctx context.Context, provider settings.ResolvedMetadataProviderInstance, upstreamProvider string, upstreamID string) (NormalizedMetadataDetail, MetadataProviderAttempt, error) {
	attempt := metadataProviderAttemptForProvider("detail", provider, ProviderAttemptOutcomeSuccess)
	detail, err := s.fetchMetaTubeDetail(ctx, provider.MetaTube, upstreamProvider, upstreamID)
	if err != nil {
		attempt = metadataProviderFailureAttempt("detail", provider, err)
		return NormalizedMetadataDetail{}, attempt, err
	}
	return normalizeMetaTubeDetail(detail), attempt, nil
}

func normalizeMetaTubeDetail(detail metatubeDetailResponse) NormalizedMetadataDetail {
	releaseDate := metatubeReleaseDate(detail.ReleaseDate, detail.Date, detail.Year)
	externalID := metatubeExternalID(detail.Provider, firstNonEmpty(detail.ID, detail.Number))
	normalized := NormalizedMetadataDetail{Provider: database.MetadataProviderTypeMetaTube, ProviderType: strings.TrimSpace(detail.Provider), ExternalID: externalID, Title: strings.TrimSpace(firstNonEmpty(detail.Title, detail.Number, detail.OriginalTitle)), OriginalTitle: strings.TrimSpace(firstNonEmpty(detail.OriginalTitle, detail.Title)), Overview: strings.TrimSpace(firstNonEmpty(detail.Overview, detail.Summary, detail.Description)), ReleaseDate: releaseDate, Year: parseYear(releaseDate), RuntimeSeconds: metatubeRuntimeSeconds(detail), ExternalIDs: []NormalizedMetadataExternalID{{Provider: database.MetadataProviderTypeMetaTube, ProviderType: strings.TrimSpace(detail.Provider), ExternalID: externalID, IsPrimary: true}}}
	normalized.Images = append(normalized.Images, NormalizedMetadataImage{ImageType: "poster", URL: firstNonEmpty(detail.PosterURL, detail.CoverURL, detail.ThumbURL), Selected: true})
	normalized.Images = append(normalized.Images, NormalizedMetadataImage{ImageType: "backdrop", URL: strings.TrimSpace(detail.BackdropURL), Selected: true})
	for index, imageURL := range detail.Images {
		normalized.Images = append(normalized.Images, NormalizedMetadataImage{ImageType: "poster", URL: strings.TrimSpace(imageURL), SortOrder: index + 1})
	}
	for index, person := range metatubeCast(detail, 8) {
		normalized.People = append(normalized.People, NormalizedMetadataPerson{Name: person.Name, Role: "actor", Character: person.Role, SortOrder: index, AvatarURL: person.AvatarURL})
	}
	for index, person := range metatubeDirectors(detail) {
		normalized.People = append(normalized.People, NormalizedMetadataPerson{Name: person.Name, Role: "director", Character: person.Role, SortOrder: index, AvatarURL: person.AvatarURL})
	}
	return normalized
}
