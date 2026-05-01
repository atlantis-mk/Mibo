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
	Type         string
	TargetKind   string
	TargetKey    string
	Confidence   *float64
	EvidenceRefs []string
	Reason       string
	Warnings     []string
	CreatedAt    time.Time
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
)
