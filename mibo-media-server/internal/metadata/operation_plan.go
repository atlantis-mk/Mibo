package metadata

import (
	"context"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/settings"
)

func (s *Service) resolveMetadataExecutionPlan(ctx context.Context, libraryID uint) (MetadataExecutionPlan, error) {
	profile, err := s.resolvedCatalogProfile(ctx, libraryID)
	if err != nil {
		return MetadataExecutionPlan{}, err
	}
	var strategy database.LibraryMetadataStrategy
	if err := s.db.WithContext(ctx).Where("library_id = ?", libraryID).First(&strategy).Error; err != nil {
		return MetadataExecutionPlan{}, err
	}
	plan := MetadataExecutionPlan{
		LibraryID:                 libraryID,
		StrategyID:                strategy.ID,
		MetadataProfileID:         strategy.MetadataProfileID,
		MetadataProfileName:       strings.TrimSpace(profile.Profile.Name),
		PreferredMetadataLanguage: strings.TrimSpace(profile.PreferredMetadataLanguage),
		PreferredImageLanguage:    strings.TrimSpace(profile.PreferredImageLanguage),
		SearchProviders:           profile.SearchProviders,
		DetailProviders:           profile.DetailProviders,
		ImageProviders:            profile.ImageProviders,
		PeopleProviders:           profile.PeopleProviders,
		HierarchyProviders:        profile.HierarchyProviders,
		LocalEvidenceEnabled:      hasProviderType(profile.DetailProviders, database.MetadataProviderTypeLocalScan),
	}
	return plan, nil
}

func metadataExecutionPlanSummary(plan MetadataExecutionPlan) MetadataExecutionPlanSummary {
	return MetadataExecutionPlanSummary{
		LibraryID:                 plan.LibraryID,
		StrategyID:                plan.StrategyID,
		MetadataProfileID:         plan.MetadataProfileID,
		MetadataProfileName:       plan.MetadataProfileName,
		PreferredMetadataLanguage: plan.PreferredMetadataLanguage,
		PreferredImageLanguage:    plan.PreferredImageLanguage,
		SearchProviders:           metadataPlanProviderSummaries(plan.SearchProviders),
		DetailProviders:           metadataPlanProviderSummaries(plan.DetailProviders),
		ImageProviders:            metadataPlanProviderSummaries(plan.ImageProviders),
		PeopleProviders:           metadataPlanProviderSummaries(plan.PeopleProviders),
		HierarchyProviders:        metadataPlanProviderSummaries(plan.HierarchyProviders),
		LocalEvidenceEnabled:      plan.LocalEvidenceEnabled,
	}
}

func metadataPlanProviderSummaries(providers []settings.ResolvedMetadataProviderInstance) []MetadataPlanProviderSummary {
	items := make([]MetadataPlanProviderSummary, 0, len(providers))
	for _, provider := range providers {
		items = append(items, MetadataPlanProviderSummary{ID: provider.Record.ID, Name: provider.Record.Name, ProviderType: provider.Record.ProviderType, Enabled: provider.Record.Enabled, Configured: provider.Configured, Operational: provider.Operational, AvailabilityStatus: provider.Record.AvailabilityStatus, CooldownUntil: formatPlanTime(provider.Record.CooldownUntil)})
	}
	return items
}

func formatPlanTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

func hasProviderType(providers []settings.ResolvedMetadataProviderInstance, providerType string) bool {
	for _, provider := range providers {
		if provider.Record.ProviderType == providerType {
			return true
		}
	}
	return false
}
