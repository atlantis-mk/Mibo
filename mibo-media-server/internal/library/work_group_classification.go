package library

import (
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
)

const (
	workGroupKindMovie        = "movie"
	workGroupKindMovieVersion = "movie_version_group"
	workGroupKindSeries       = "series_hierarchy"
	workGroupKindUnresolved   = "unresolved"
)

type workGroupClassification struct {
	Kind            string
	MetadataType    string
	Strength        string
	ReviewState     string
	Accepted        bool
	Conflict        bool
	Confidence      float64
	Reason          string
	Evidence        []scanDecisionEvidence
	Alternatives    []scanDecisionAlternative
	HasWeakFallback bool
}

func classifyWorkGroup(artifact catalogScanArtifact) workGroupClassification {
	seriesEvidence := workGroupSeriesEvidence(artifact)
	movieEvidence := workGroupMovieEvidence(artifact)
	decision := workGroupClassification{
		Kind:         workGroupKindUnresolved,
		MetadataType: "",
		Strength:     inventory.MetadataCandidateStrengthWeak,
		ReviewState:  database.ReviewStateNeedsReview,
		Accepted:     false,
		Confidence:   0.45,
		Reason:       "insufficient grouped evidence for final metadata collapse",
	}
	switch {
	case seriesEvidence.strong && movieEvidence.strong:
		decision.Conflict = true
		decision.Reason = "conflicting grouped movie and series evidence"
		decision.Evidence = append(decision.Evidence, seriesEvidence.evidence...)
		decision.Evidence = append(decision.Evidence, movieEvidence.evidence...)
		decision.Alternatives = []scanDecisionAlternative{{Type: scanDecisionCandidateEpisode, Confidence: floatPtr(seriesEvidence.confidence), Reason: "series evidence present"}, {Type: scanDecisionCandidateMovie, Confidence: floatPtr(movieEvidence.confidence), Reason: "movie evidence present"}}
	case seriesEvidence.strong:
		decision.Kind = workGroupKindSeries
		decision.MetadataType = catalog.ItemTypeEpisode
		decision.Strength = inventory.MetadataCandidateStrengthStrong
		decision.ReviewState = database.ReviewStateAccepted
		decision.Accepted = true
		decision.Confidence = seriesEvidence.confidence
		decision.Reason = seriesEvidence.reason
		decision.Evidence = append(decision.Evidence, seriesEvidence.evidence...)
	case movieEvidence.strong:
		decision.Kind = movieEvidence.kind
		decision.MetadataType = catalog.ItemTypeMovie
		decision.Strength = inventory.MetadataCandidateStrengthStrong
		decision.ReviewState = database.ReviewStateAccepted
		decision.Accepted = true
		decision.Confidence = movieEvidence.confidence
		decision.Reason = movieEvidence.reason
		decision.Evidence = append(decision.Evidence, movieEvidence.evidence...)
	case movieEvidence.weak:
		decision.HasWeakFallback = true
		decision.Reason = movieEvidence.reason
		decision.Evidence = append(decision.Evidence, movieEvidence.evidence...)
		decision.Alternatives = []scanDecisionAlternative{{Type: scanDecisionCandidateMovie, Confidence: floatPtr(movieEvidence.confidence), Reason: "weak movie fallback"}}
	}
	return decision
}

type workGroupSignal struct {
	kind       string
	strong     bool
	weak       bool
	confidence float64
	reason     string
	evidence   []scanDecisionEvidence
}

func workGroupSeriesEvidence(artifact catalogScanArtifact) workGroupSignal {
	evidence := make([]scanDecisionEvidence, 0)
	for _, sidecar := range artifact.MetadataSidecars {
		if sidecar.ParseStatus != "parsed" {
			continue
		}
		if shouldTreatSidecarAsEpisode(sidecar.Hints) || strings.EqualFold(strings.TrimSpace(sidecar.Hints.MediaType), catalog.ItemTypeSeries) {
			evidence = append(evidence, scanDecisionEvidence{Kind: "sidecar_media_type", Source: strings.TrimSpace(sidecar.Extension), Value: firstNonEmptyString(strings.TrimSpace(sidecar.Hints.MediaType), catalog.ItemTypeEpisode)})
			return workGroupSignal{kind: workGroupKindSeries, strong: true, confidence: 0.97, reason: "sidecar metadata indicates series hierarchy", evidence: evidence}
		}
	}
	if strings.TrimSpace(artifact.ItemType) == catalog.ItemTypeEpisode {
		evidence = append(evidence, scanDecisionEvidence{Kind: "item_type", Source: "artifact", Value: catalog.ItemTypeEpisode})
	}
	if strings.TrimSpace(artifact.SeriesTitle) != "" && artifact.SeasonNumber != nil && len(artifact.EpisodeSlots) > 0 {
		evidence = append(evidence,
			scanDecisionEvidence{Kind: "series_title", Source: "group", Value: strings.TrimSpace(artifact.SeriesTitle)},
			scanDecisionEvidence{Kind: "season_number", Source: "group", Value: strconv.Itoa(*artifact.SeasonNumber)},
		)
		return workGroupSignal{kind: workGroupKindSeries, strong: true, confidence: 0.95, reason: "series hierarchy evidence present", evidence: evidence}
	}
	for _, externalID := range artifact.ExternalIDs {
		providerType := strings.TrimSpace(externalID.ProviderType)
		if providerType == "tv" || providerType == "tv_episode" || providerType == "tv_season" {
			evidence = append(evidence, scanDecisionEvidence{Kind: "external_id", Source: strings.TrimSpace(externalID.Provider), Value: providerType})
			return workGroupSignal{kind: workGroupKindSeries, strong: true, confidence: 0.95, reason: "external identity indicates series hierarchy", evidence: evidence}
		}
	}
	if artifact.FilenameSignals.Identity.SeasonNumber != nil && (artifact.FilenameSignals.Identity.EpisodeNumber != nil || len(artifact.FilenameSignals.Identity.EpisodeNumbers) > 0) {
		evidence = append(evidence, filenameEvidenceSummariesToScanDecisionEvidence(artifact.FilenameSignals.Evidence)...)
		return workGroupSignal{kind: workGroupKindSeries, strong: true, confidence: 0.82, reason: "filename episode markers indicate series hierarchy", evidence: evidence}
	}
	if artifact.FilenameSignals.PathHints.SeasonNumber != nil && (artifact.FilenameSignals.Identity.EpisodeNumber != nil || len(artifact.FilenameSignals.Identity.EpisodeNumbers) > 0) {
		evidence = append(evidence, filenameEvidenceSummariesToScanDecisionEvidence(artifact.FilenameSignals.Evidence)...)
		return workGroupSignal{kind: workGroupKindSeries, strong: true, confidence: 0.78, reason: "directory and filename episode markers indicate series hierarchy", evidence: evidence}
	}
	return workGroupSignal{}
}

func workGroupMovieEvidence(artifact catalogScanArtifact) workGroupSignal {
	evidence := make([]scanDecisionEvidence, 0)
	for _, sidecar := range artifact.MetadataSidecars {
		if sidecar.ParseStatus != "parsed" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(sidecar.Hints.MediaType), catalog.ItemTypeMovie) {
			evidence = append(evidence, scanDecisionEvidence{Kind: "sidecar_media_type", Source: strings.TrimSpace(sidecar.Extension), Value: catalog.ItemTypeMovie})
			kind := workGroupKindMovie
			if artifact.PreferredLinkRole == database.ResourceLinkRoleVersion {
				kind = workGroupKindMovieVersion
			}
			return workGroupSignal{kind: kind, strong: true, confidence: 0.9, reason: "sidecar metadata indicates movie work", evidence: evidence}
		}
	}
	for _, externalID := range artifact.ExternalIDs {
		providerType := strings.TrimSpace(externalID.ProviderType)
		if providerType == "movie" {
			evidence = append(evidence, scanDecisionEvidence{Kind: "external_id", Source: strings.TrimSpace(externalID.Provider), Value: providerType})
			kind := workGroupKindMovie
			if artifact.PreferredLinkRole == database.ResourceLinkRoleVersion {
				kind = workGroupKindMovieVersion
			}
			return workGroupSignal{kind: kind, strong: true, confidence: 0.95, reason: "external identity indicates movie", evidence: evidence}
		}
	}
	assignmentType, _ := artifact.ContentShapeAssignment["path_tree_assignment_type"].(string)
	switch strings.TrimSpace(assignmentType) {
	case pathTreeAssignmentMovie:
		evidence = append(evidence, scanDecisionEvidence{Kind: "work_group", Source: "path_tree", Value: assignmentType})
		return workGroupSignal{kind: workGroupKindMovie, strong: true, confidence: 0.9, reason: "path-tree assignment indicates movie work", evidence: evidence}
	case pathTreeAssignmentVersion:
		evidence = append(evidence, scanDecisionEvidence{Kind: "work_group", Source: "path_tree", Value: assignmentType})
		return workGroupSignal{kind: workGroupKindMovieVersion, strong: true, confidence: 0.9, reason: "path-tree assignment indicates movie version group", evidence: evidence}
	}
	shapeAssignment, _ := artifact.ContentShapeAssignment["assignment_type"].(string)
	shapeReview, _ := artifact.ContentShapeAssignment["review_state"].(string)
	if (shapeAssignment == contentShapeAssignmentMovie || shapeAssignment == contentShapeAssignmentVersion) && shapeReview != scanDecisionStatusReviewRequired {
		evidence = append(evidence, scanDecisionEvidence{Kind: "directory_plan", Source: "content_shape", Value: shapeAssignment})
		kind := workGroupKindMovie
		if shapeAssignment == contentShapeAssignmentVersion {
			kind = workGroupKindMovieVersion
		}
		return workGroupSignal{kind: kind, strong: true, confidence: 0.82, reason: "content-shape assignment indicates movie work", evidence: evidence}
	}
	if strings.TrimSpace(artifact.Title) != "" && artifact.Year != nil && artifact.FilenameSignals.Identity.EpisodeNumber == nil && len(artifact.FilenameSignals.Identity.EpisodeNumbers) == 0 {
		evidence = append(evidence, scanDecisionEvidence{Kind: filenameSignalKindTitle, Source: "filename", Value: strings.TrimSpace(artifact.Title)})
		evidence = append(evidence, scanDecisionEvidence{Kind: filenameSignalKindYear, Source: "filename", Value: strconv.Itoa(*artifact.Year)})
		return workGroupSignal{kind: workGroupKindMovie, weak: true, confidence: 0.6, reason: "only weak title and year movie fallback available", evidence: evidence}
	}
	return workGroupSignal{}
}

func floatPtr(value float64) *float64 {
	return &value
}
