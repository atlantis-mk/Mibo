package inventory

import (
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

const (
	MetadataCandidateStrengthStrong = "strong"
	MetadataCandidateStrengthMedium = "medium"
	MetadataCandidateStrengthWeak   = "weak"
)

type MetadataCandidateEvidence struct {
	ExternalIDMatched      bool
	SidecarMatched         bool
	SeriesTitleMatched     bool
	SeasonNumberMatched    bool
	EpisodeNumberMatched   bool
	NormalizedTitleMatched bool
	YearMatched            bool
}

func ClassifyMetadataCandidate(evidence MetadataCandidateEvidence) string {
	if evidence.ExternalIDMatched || evidence.SidecarMatched {
		return MetadataCandidateStrengthStrong
	}
	if evidence.SeriesTitleMatched && evidence.SeasonNumberMatched && evidence.EpisodeNumberMatched {
		return MetadataCandidateStrengthStrong
	}
	if evidence.NormalizedTitleMatched && evidence.YearMatched {
		return MetadataCandidateStrengthMedium
	}
	if evidence.NormalizedTitleMatched {
		return MetadataCandidateStrengthWeak
	}
	return MetadataCandidateStrengthWeak
}

func CandidateReviewState(strength string) string {
	switch strings.TrimSpace(strings.ToLower(strength)) {
	case MetadataCandidateStrengthStrong, MetadataCandidateStrengthMedium:
		return database.ReviewStateAccepted
	default:
		return database.ReviewStateNeedsReview
	}
}
