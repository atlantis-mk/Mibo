package metadata

import (
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/settings"
)

func providerWithOperationLanguage(provider settings.ResolvedMetadataProviderInstance, plan MetadataExecutionPlan) settings.ResolvedMetadataProviderInstance {
	if strings.TrimSpace(plan.PreferredMetadataLanguage) != "" {
		provider.TMDB.Language = strings.TrimSpace(plan.PreferredMetadataLanguage)
	}
	return provider
}

func prioritizeProviderInstances(providers []settings.ResolvedMetadataProviderInstance, preferredName string) []settings.ResolvedMetadataProviderInstance {
	preferredName = strings.TrimSpace(preferredName)
	if preferredName == "" || len(providers) < 2 {
		return providers
	}
	ordered := make([]settings.ResolvedMetadataProviderInstance, 0, len(providers))
	for _, provider := range providers {
		if provider.Record.Name == preferredName {
			ordered = append(ordered, provider)
		}
	}
	for _, provider := range providers {
		if provider.Record.Name != preferredName {
			ordered = append(ordered, provider)
		}
	}
	return ordered
}

func manualCandidateProviderTypes(externalID string) (string, string, error) {
	trimmed := strings.TrimSpace(externalID)
	if strings.HasPrefix(trimmed, "metatube:") {
		return database.MetadataProviderTypeMetaTube, "", nil
	}
	mediaType, _, err := parseExternalID(trimmed)
	if err != nil {
		return "", "", err
	}
	return database.MetadataProviderTypeTMDB, mediaType, nil
}

func catalogTMDBMediaType(itemType string) string {
	switch strings.TrimSpace(itemType) {
	case database.MetadataItemTypeSeries, database.MetadataItemTypeSeason, database.MetadataItemTypeEpisode:
		return "tv"
	default:
		return "movie"
	}
}

func providerExternalID(mediaType string, id int) string {
	return strings.TrimSpace(mediaType) + ":" + strconv.Itoa(id)
}

func tmdbExternalID(id int) string {
	return strconv.Itoa(id)
}
