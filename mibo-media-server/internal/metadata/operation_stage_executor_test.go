package metadata

import (
	"context"
	"errors"
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/settings"
)

func TestExecuteMetadataProviderStageRecordsFallbackOutcomes(t *testing.T) {
	primary := settings.ResolvedMetadataProviderInstance{Record: database.MetadataProviderInstance{ID: 1, Name: "primary", ProviderType: database.MetadataProviderTypeTMDB, Enabled: false}, Configured: true, Operational: false}
	empty := settings.ResolvedMetadataProviderInstance{Record: database.MetadataProviderInstance{ID: 2, Name: "empty", ProviderType: database.MetadataProviderTypeTMDB, Enabled: true}, Configured: true, Operational: true}
	secondary := settings.ResolvedMetadataProviderInstance{Record: database.MetadataProviderInstance{ID: 3, Name: "secondary", ProviderType: database.MetadataProviderTypeTMDB, Enabled: true}, Configured: true, Operational: true}

	attempts, selected, err := executeMetadataProviderStage(context.Background(), "search", []settings.ResolvedMetadataProviderInstance{primary, empty, secondary}, func(_ context.Context, provider settings.ResolvedMetadataProviderInstance) (MetadataProviderAttempt, bool, error) {
		if provider.Record.Name == "empty" {
			return metadataProviderAttemptForProvider("search", provider, ProviderAttemptOutcomeNoResult), false, nil
		}
		attempt := metadataProviderAttemptForProvider("search", provider, ProviderAttemptOutcomeSuccess)
		attempt.CandidateCount = 1
		return attempt, true, nil
	})
	if err != nil {
		t.Fatalf("execute provider stage: %v", err)
	}
	if selected == nil || selected.Record.Name != "secondary" {
		t.Fatalf("expected secondary provider selected, got %#v", selected)
	}
	if len(attempts) != 3 || attempts[0].Outcome != ProviderAttemptOutcomeSkippedUnavailable || attempts[1].Outcome != ProviderAttemptOutcomeNoResult || attempts[2].Outcome != ProviderAttemptOutcomeSuccess || !attempts[2].Selected {
		t.Fatalf("unexpected attempts: %#v", attempts)
	}
}

func TestExecuteMetadataProviderStageReturnsRetryableFailureAttempt(t *testing.T) {
	provider := settings.ResolvedMetadataProviderInstance{Record: database.MetadataProviderInstance{ID: 1, Name: "primary", ProviderType: database.MetadataProviderTypeTMDB, Enabled: true}, Configured: true, Operational: true}
	attempts, selected, err := executeMetadataProviderStage(context.Background(), "search", []settings.ResolvedMetadataProviderInstance{provider}, func(_ context.Context, provider settings.ResolvedMetadataProviderInstance) (MetadataProviderAttempt, bool, error) {
		err := errors.New("temporary upstream failure")
		return metadataProviderFailureAttempt("search", provider, err), false, err
	})
	if err == nil {
		t.Fatalf("expected stage error")
	}
	if selected != nil {
		t.Fatalf("expected no selected provider, got %#v", selected)
	}
	if len(attempts) != 1 || attempts[0].Outcome != ProviderAttemptOutcomeFailedRetryable || attempts[0].ErrorClass != "error" {
		t.Fatalf("unexpected failure attempts: %#v", attempts)
	}
}
