package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/settings"
)

type providerSelection struct {
	Provider settings.ResolvedMetadataProviderInstance
	Summary  settings.MetadataExecutionFallbackSummary
}

func (s *Service) settingsService() *settings.Service {
	if s.settings != nil {
		return s.settings
	}
	return settings.NewService(s.db, s.fallback)
}

func (s *Service) resolvedCatalogProfile(ctx context.Context, libraryID uint) (settings.ResolvedLibraryMetadataProfile, error) {
	return s.settingsService().ResolveLibraryMetadataProfile(ctx, libraryID)
}

func (s *Service) selectProviderForStage(profile settings.ResolvedLibraryMetadataProfile, stage string, preferredProviderInstanceName string) (*providerSelection, error) {
	providers := profileProvidersForStage(profile, stage)
	if len(providers) == 0 {
		return nil, fmt.Errorf("no configured provider instance is available for metadata stage %q", stage)
	}
	attempted := make([]string, 0, len(providers))
	if preferred := strings.TrimSpace(preferredProviderInstanceName); preferred != "" {
		for _, provider := range providers {
			attempted = append(attempted, provider.Record.Name)
			if provider.Record.Name == preferred {
				return &providerSelection{Provider: provider, Summary: settings.MetadataExecutionFallbackSummary{Stage: stage, Attempted: attempted, Selected: provider.Record.Name}}, nil
			}
		}
	}
	selected := providers[0]
	for _, provider := range providers {
		attempted = append(attempted, provider.Record.Name)
	}
	return &providerSelection{Provider: selected, Summary: settings.MetadataExecutionFallbackSummary{Stage: stage, Attempted: attempted, Selected: selected.Record.Name, UsedFallback: strings.TrimSpace(preferredProviderInstanceName) != "" && selected.Record.Name != strings.TrimSpace(preferredProviderInstanceName)}}, nil
}

func profileProvidersForStage(profile settings.ResolvedLibraryMetadataProfile, stage string) []settings.ResolvedMetadataProviderInstance {
	switch stage {
	case "search":
		return profile.SearchProviders
	case "detail":
		return profile.DetailProviders
	case "image":
		return profile.ImageProviders
	case "people":
		return profile.PeopleProviders
	case "hierarchy":
		return profile.HierarchyProviders
	default:
		return nil
	}
}

func (s *Service) recordProviderFailure(ctx context.Context, provider settings.ResolvedMetadataProviderInstance, err error) {
	if provider.Record.ID == 0 || err == nil {
		return
	}
	updates := map[string]any{"failure_reason": strings.TrimSpace(err.Error()), "updated_at": time.Now().UTC()}
	if failure, ok := err.(providerRequestFailure); ok {
		switch failure.StatusCode() {
		case 401, 403:
			updates["availability_status"] = "unavailable"
		case 429:
			updates["availability_status"] = "cooldown"
			updates["cooldown_until"] = time.Now().UTC().Add(15 * time.Minute)
		}
	}
	_ = s.db.WithContext(ctx).Model(&provider.Record).Updates(updates).Error
}

func mustMarshalFallbackSummary(value []settings.MetadataExecutionFallbackSummary) string {
	if len(value) == 0 {
		return ""
	}
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}
