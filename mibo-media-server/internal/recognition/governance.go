package recognition

import (
	"context"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

type GovernanceReviewGroup struct {
	Manifest   database.RecognitionManifest    `json:"manifest"`
	Candidates []database.RecognitionCandidate `json:"candidates"`
	Decisions  []database.RecognitionDecision  `json:"decisions"`
	Conflicts  []database.RecognitionConflict  `json:"conflicts"`
	Evidence   []database.RecognitionEvidence  `json:"evidence"`
}

type CorrectionRuleInput struct {
	LibraryID       uint
	MediaSourceID   uint
	StorageProvider string
	ScopePath       string
	RuleType        string
	CandidateType   string
	Action          string
	Priority        int
	PayloadJSON     string
	EvidenceJSON    string
	UserID          *uint
}

func (r *Repository) ApplyCorrectionRule(ctx context.Context, input CorrectionRuleInput) (database.RecognitionRule, error) {
	ruleType := strings.TrimSpace(input.RuleType)
	if ruleType == "" {
		ruleType = "recognition_correction"
	}
	action := strings.TrimSpace(input.Action)
	if action == "" {
		action = RuleActionAccept
	}
	priority := input.Priority
	if priority == 0 {
		priority = 10
	}
	rule := database.RecognitionRule{RuleKey: correctionRuleKey(input, ruleType, action), LibraryID: input.LibraryID, MediaSourceID: input.MediaSourceID, StorageProvider: strings.TrimSpace(input.StorageProvider), ScopePath: strings.TrimSpace(input.ScopePath), RuleType: ruleType, CandidateType: strings.TrimSpace(input.CandidateType), Action: action, Priority: priority, PayloadJSON: strings.TrimSpace(input.PayloadJSON), EvidenceJSON: strings.TrimSpace(input.EvidenceJSON), Enabled: true, CreatedByUserID: input.UserID, UpdatedByUserID: input.UserID}
	return r.UpsertRule(ctx, rule)
}

func correctionRuleKey(input CorrectionRuleInput, ruleType string, action string) string {
	parts := []string{"recognition-rule", stringUint(input.LibraryID), strings.TrimSpace(input.StorageProvider), strings.TrimSpace(input.ScopePath), ruleType, strings.TrimSpace(input.CandidateType), action}
	return strings.Join(parts, ":")
}

func (r *Repository) LoadGovernanceReviewGroups(ctx context.Context, libraryID uint) ([]GovernanceReviewGroup, error) {
	manifests, err := r.ListOpenManifestsForLibrary(ctx, libraryID)
	if err != nil {
		return nil, err
	}
	groups := make([]GovernanceReviewGroup, 0, len(manifests))
	for _, manifest := range manifests {
		graph, err := r.LoadManifestGraph(ctx, manifest.ID)
		if err != nil {
			return nil, err
		}
		if !graphNeedsReview(graph) {
			continue
		}
		groups = append(groups, GovernanceReviewGroup{Manifest: graph.Manifest, Candidates: graph.Candidates, Decisions: graph.Decisions, Conflicts: graph.Conflicts, Evidence: graph.Evidence})
	}
	return groups, nil
}

func graphNeedsReview(graph ManifestGraph) bool {
	if len(graph.Conflicts) > 0 {
		return true
	}
	for _, decision := range graph.Decisions {
		switch decision.Outcome {
		case DecisionOutcomeReviewRequired, DecisionOutcomeBlockedConflict, DecisionOutcomeUnmatched:
			return true
		}
	}
	return false
}
