package metadata

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/atlan/mibo-media-server/internal/settings"
)

type metadataStageExecuteFunc func(context.Context, settings.ResolvedMetadataProviderInstance) (MetadataProviderAttempt, bool, error)

func executeMetadataProviderStage(ctx context.Context, stage string, providers []settings.ResolvedMetadataProviderInstance, execute metadataStageExecuteFunc) ([]MetadataProviderAttempt, *settings.ResolvedMetadataProviderInstance, error) {
	attempts := make([]MetadataProviderAttempt, 0, len(providers))
	for _, provider := range providers {
		if !provider.Operational {
			attempts = append(attempts, metadataProviderAttemptForProvider(stage, provider, ProviderAttemptOutcomeSkippedUnavailable))
			continue
		}
		attempt, selected, err := execute(ctx, provider)
		if attempt.Stage == "" {
			attempt.Stage = stage
		}
		if attempt.ProviderInstanceID == 0 {
			attempt.ProviderInstanceID = provider.Record.ID
			attempt.ProviderInstanceName = provider.Record.Name
			attempt.ProviderType = provider.Record.ProviderType
		}
		attempt.Selected = selected
		attempts = append(attempts, attempt)
		if err != nil {
			return attempts, nil, err
		}
		if selected {
			selectedProvider := provider
			return attempts, &selectedProvider, nil
		}
	}
	return attempts, nil, nil
}

func metadataProviderAttemptForProvider(stage string, provider settings.ResolvedMetadataProviderInstance, outcome string) MetadataProviderAttempt {
	return MetadataProviderAttempt{Stage: stage, ProviderInstanceID: provider.Record.ID, ProviderInstanceName: provider.Record.Name, ProviderType: provider.Record.ProviderType, Outcome: outcome}
}

func metadataProviderFailureAttempt(stage string, provider settings.ResolvedMetadataProviderInstance, err error) MetadataProviderAttempt {
	attempt := metadataProviderAttemptForProvider(stage, provider, ProviderAttemptOutcomeFailedRetryable)
	if err == nil {
		return attempt
	}
	attempt.ErrorMessage = err.Error()
	var failure providerRequestFailure
	if errors.As(err, &failure) {
		attempt.StatusCode = failure.StatusCode()
		switch failure.StatusCode() {
		case 401, 403:
			attempt.ErrorClass = "auth"
			attempt.Outcome = ProviderAttemptOutcomeFailedTerminal
		case 429:
			attempt.ErrorClass = "rate_limit"
			attempt.Outcome = ProviderAttemptOutcomeFailedRetryable
		default:
			attempt.ErrorClass = "http"
		}
		return attempt
	}
	if errors.Is(err, context.DeadlineExceeded) || strings.Contains(strings.ToLower(err.Error()), "timeout") {
		attempt.ErrorClass = "timeout"
		return attempt
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		attempt.ErrorClass = "timeout"
		return attempt
	}
	attempt.ErrorClass = "error"
	return attempt
}
