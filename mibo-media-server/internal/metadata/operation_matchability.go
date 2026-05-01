package metadata

import (
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/settings"
)

type MetadataOperationMatchability struct {
	Runnable bool
	Reason   string
}

func metadataOperationMatchability(plan MetadataExecutionPlan, operation string, providerType string) MetadataOperationMatchability {
	switch operation {
	case OperationTypeMatch:
		if hasOperationalProvider(plan.SearchProviders) || plan.LocalEvidenceEnabled {
			return MetadataOperationMatchability{Runnable: true}
		}
		return MetadataOperationMatchability{Reason: "no operational search provider or local evidence executor"}
	case OperationTypeRefetch:
		if providerType != "" && hasOperationalProviderType(plan.DetailProviders, providerType) {
			return MetadataOperationMatchability{Runnable: true}
		}
		if plan.LocalEvidenceEnabled {
			return MetadataOperationMatchability{Runnable: true}
		}
		return MetadataOperationMatchability{Reason: "no compatible detail provider or local evidence executor"}
	case OperationTypeManualApply:
		if providerType == "" || hasOperationalProviderType(plan.DetailProviders, providerType) {
			return MetadataOperationMatchability{Runnable: true}
		}
		return MetadataOperationMatchability{Reason: "manual candidate provider is not allowed by detail strategy"}
	case OperationTypeLocalApply:
		if plan.LocalEvidenceEnabled || hasOperationalProviderType(plan.DetailProviders, database.MetadataProviderTypeLocalScan) {
			return MetadataOperationMatchability{Runnable: true}
		}
		return MetadataOperationMatchability{Reason: "local evidence executor is not enabled"}
	default:
		return MetadataOperationMatchability{Reason: "unsupported metadata operation"}
	}
}

func hasOperationalProvider(providers []settings.ResolvedMetadataProviderInstance) bool {
	for _, provider := range providers {
		if provider.Operational {
			return true
		}
	}
	return false
}

func hasOperationalProviderType(providers []settings.ResolvedMetadataProviderInstance, providerType string) bool {
	for _, provider := range providers {
		if provider.Operational && provider.Record.ProviderType == providerType {
			return true
		}
	}
	return false
}
