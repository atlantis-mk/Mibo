package inventory_test

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
)

func TestClassifyMetadataCandidateStrength(t *testing.T) {
	tests := []struct {
		name     string
		evidence inventory.MetadataCandidateEvidence
		want     string
	}{
		{name: "external id", evidence: inventory.MetadataCandidateEvidence{ExternalIDMatched: true}, want: inventory.MetadataCandidateStrengthStrong},
		{name: "sidecar", evidence: inventory.MetadataCandidateEvidence{SidecarMatched: true}, want: inventory.MetadataCandidateStrengthStrong},
		{name: "episode hierarchy", evidence: inventory.MetadataCandidateEvidence{SeriesTitleMatched: true, SeasonNumberMatched: true, EpisodeNumberMatched: true}, want: inventory.MetadataCandidateStrengthStrong},
		{name: "title year", evidence: inventory.MetadataCandidateEvidence{NormalizedTitleMatched: true, YearMatched: true}, want: inventory.MetadataCandidateStrengthMedium},
		{name: "same name", evidence: inventory.MetadataCandidateEvidence{NormalizedTitleMatched: true}, want: inventory.MetadataCandidateStrengthWeak},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inventory.ClassifyMetadataCandidate(tt.evidence); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestCandidateReviewState(t *testing.T) {
	if got := inventory.CandidateReviewState(inventory.MetadataCandidateStrengthStrong); got != database.ReviewStateAccepted {
		t.Fatalf("expected strong candidate accepted, got %q", got)
	}
	if got := inventory.CandidateReviewState(inventory.MetadataCandidateStrengthMedium); got != database.ReviewStateAccepted {
		t.Fatalf("expected medium candidate accepted, got %q", got)
	}
	if got := inventory.CandidateReviewState(inventory.MetadataCandidateStrengthWeak); got != database.ReviewStateNeedsReview {
		t.Fatalf("expected weak candidate needs review, got %q", got)
	}
}
