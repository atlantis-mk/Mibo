package metadata

import (
	"context"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
)

func applyNormalizedExternalIDs(ctx context.Context, catalogSvc *catalog.Service, itemID uint, externalIDs []NormalizedMetadataExternalID, source string, defaultConfidence *float64) error {
	for _, externalID := range externalIDs {
		provider := strings.TrimSpace(externalID.Provider)
		providerType := strings.TrimSpace(externalID.ProviderType)
		value := strings.TrimSpace(externalID.ExternalID)
		if itemID == 0 || provider == "" || providerType == "" || value == "" {
			continue
		}
		confidence := externalID.Confidence
		if confidence == nil {
			confidence = defaultConfidence
		}
		if _, err := catalogSvc.SetExternalID(ctx, catalog.ExternalIDInput{ItemID: itemID, Provider: provider, ProviderType: providerType, ExternalID: value, IsPrimary: externalID.IsPrimary, Source: source, Confidence: confidence}); err != nil {
			return err
		}
		if shouldWriteStableProviderIdentity(provider, providerType, value) {
			if _, err := catalogSvc.SetIdentity(ctx, catalog.IdentityInput{ItemID: itemID, Provider: provider, IdentityType: providerType, IdentityKey: value, Confidence: confidence}); err != nil {
				return err
			}
		}
	}
	return nil
}

func shouldWriteStableProviderIdentity(provider string, providerType string, externalID string) bool {
	return strings.TrimSpace(provider) != "" && strings.TrimSpace(providerType) != "" && strings.TrimSpace(externalID) != ""
}
