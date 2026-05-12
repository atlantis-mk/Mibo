package library

import (
	"time"

	"github.com/atlan/mibo-media-server/internal/storage"
)

type scanGraph struct {
	RootPath  string
	Nodes     []scanGraphNode
	Edges     []scanGraphEdge
	Decisions []scanDecision
}

type scanGraphNode struct {
	ID         string
	Kind       string
	Path       string
	ParentPath string
	Object     storage.Object
	Attrs      map[string]string
}

type scanGraphEdge struct {
	FromID string
	ToID   string
	Kind   string
	Attrs  map[string]string
}

type scanDecision struct {
	Type          string
	TargetKind    string
	TargetKey     string
	Role          string
	CandidateType string
	Status        string
	Confidence    *float64
	Alternatives  []scanDecisionAlternative
	Evidence      []scanDecisionEvidence
	EvidenceRefs  []string
	Reason        string
	Warnings      []string
	CreatedAt     time.Time
}

type scanDecisionAlternative struct {
	Type       string
	Role       string
	TargetKind string
	TargetKey  string
	Confidence *float64
	Reason     string
}

type scanDecisionEvidence struct {
	Kind   string
	Source string
	Value  string
	Weight *float64
}

const (
	scanGraphNodeDirectory      = "directory"
	scanGraphNodeFile           = "file"
	scanGraphNodeSidecar        = "sidecar"
	scanGraphNodeCandidateWork  = "candidate_work"
	scanGraphNodeCandidateAsset = "candidate_asset"

	scanGraphEdgeContains          = "contains"
	scanGraphEdgeSameBasename      = "same_basename"
	scanGraphEdgeSidecarOf         = "sidecar_of"
	scanGraphEdgeBelongsToGroup    = "belongs_to_group"
	scanGraphEdgeCandidateMainFile = "candidate_main_file"
	scanGraphEdgeCandidateExtra    = "candidate_extra_file"
	scanGraphEdgeCandidateVersion  = "candidate_version"
	scanGraphEdgeCandidateEpisode  = "candidate_episode_slot"

	scanDecisionSeriesGroup = "series_group"
	scanDecisionMovieGroup  = "movie_group"
	scanDecisionAssetLink   = "asset_link"
	scanDecisionEpisodeSlot = "episode_slot"

	scanDecisionRoleMain              = "main"
	scanDecisionRoleTrailer           = "trailer"
	scanDecisionRoleExtra             = "extra"
	scanDecisionRoleSample            = "sample"
	scanDecisionRoleUnknownAttachment = "unknown_attachment"

	scanDecisionCandidateMovie            = "movie"
	scanDecisionCandidateEpisode          = "episode"
	scanDecisionCandidateAttachment       = "attachment"
	scanDecisionCandidateMovieVersion     = "movie_version"
	scanDecisionCandidateIndependentMovie = "independent_movie"

	scanDecisionStatusConfirmedFast  = "confirmed_fast"
	scanDecisionStatusProvisional    = "provisional"
	scanDecisionStatusReviewRequired = "review_required"
)

var defaultFastClassificationThresholds = fastClassificationThresholds{
	Confirmed: 0.80,
	Review:    0.50,
	Margin:    0.15,
}

type fastClassificationThresholds struct {
	Confirmed float64
	Review    float64
	Margin    float64
}

func classifyFastDecisionStatus(confidence float64, alternatives []scanDecisionAlternative, thresholds fastClassificationThresholds) string {
	if thresholds.Confirmed <= 0 {
		thresholds = defaultFastClassificationThresholds
	}
	if confidence < thresholds.Review {
		return scanDecisionStatusReviewRequired
	}
	for _, alternative := range alternatives {
		if alternative.Confidence == nil {
			continue
		}
		if confidence-*alternative.Confidence < thresholds.Margin {
			return scanDecisionStatusReviewRequired
		}
	}
	if confidence >= thresholds.Confirmed {
		return scanDecisionStatusConfirmedFast
	}
	return scanDecisionStatusProvisional
}

func floatPtr(value float64) *float64 {
	return &value
}
