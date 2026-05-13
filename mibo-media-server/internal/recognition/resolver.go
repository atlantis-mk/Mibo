package recognition

import (
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

const (
	RuleActionAccept     = "accept"
	RuleActionMerge      = "merge"
	RuleActionSplit      = "split"
	RuleActionReject     = "reject"
	RuleActionClassifyAs = "classify_as"

	DecisionOutcomeAccepted        = "accepted"
	DecisionOutcomeProvisional     = "provisional"
	DecisionOutcomeReviewRequired  = "review_required"
	DecisionOutcomeBlockedConflict = "blocked_conflict"
	DecisionOutcomeUnmatched       = "unmatched"
	DecisionOutcomeRejected        = "rejected"
)

type Resolver struct {
	rules []database.RecognitionRule
}

func NewResolver(rules []database.RecognitionRule) *Resolver {
	return &Resolver{rules: sortedRules(rules)}
}

type ResolveResult struct {
	Decisions []database.RecognitionDecision
	Conflicts []database.RecognitionConflict
}

func (r *Resolver) Resolve(graph ManifestGraph) ResolveResult {
	result := ResolveResult{Conflicts: detectBlockingConflicts(graph)}
	manualAccepted := make([]database.RecognitionCandidate, 0)
	manualCandidateKeys := make(map[string]struct{})
	remaining := make([]database.RecognitionCandidate, 0, len(graph.Candidates))
	for _, candidate := range graph.Candidates {
		if conflict, ok := blockingConflictForCandidate(candidate, result.Conflicts); ok {
			result.Decisions = append(result.Decisions, database.RecognitionDecision{ManifestID: graph.Manifest.ID, CandidateID: uintPtr(candidate.ID), DecisionType: "resolver_conflict", Outcome: DecisionOutcomeBlockedConflict, TargetKind: candidate.CandidateType, TargetKey: candidate.CandidateKey, Confidence: candidate.Confidence, Reason: conflict.Reason, ConflictsJSON: mustJSON(conflict)})
			continue
		}
		if rule, ok := r.matchingRule(candidate); ok {
			decision, conflict := decisionFromRule(graph.Manifest.ID, candidate, rule)
			if conflict.ConflictKey != "" {
				result.Conflicts = append(result.Conflicts, conflict)
			}
			result.Decisions = append(result.Decisions, decision)
			manualCandidateKeys[candidate.CandidateKey] = struct{}{}
			if decision.Outcome == DecisionOutcomeAccepted {
				manualAccepted = append(manualAccepted, candidate)
			}
			continue
		}
		remaining = append(remaining, candidate)
	}
	constraints := ApplyHardConstraints(remaining, graph.Evidence)
	allowMovieCollectionLeadingEpisodes(remaining, graph.Evidence, constraints)
	inferenceCandidates := append([]database.RecognitionCandidate(nil), manualAccepted...)
	inferenceCandidates = append(inferenceCandidates, remaining...)
	inference := InferConsistentCandidateGraph(inferenceCandidates, constraints)
	scores := ScoreCandidates(inference.AcceptedCandidates, graph.Evidence)
	applyCandidateConfidence(scores, inference.AcceptedCandidates)
	kernelResult := BuildKernelDecisions(graph.Manifest, inference, constraints, scores)
	for _, decision := range kernelResult.Decisions {
		if _, ok := manualCandidateKeys[decision.TargetKey]; ok {
			continue
		}
		result.Decisions = append(result.Decisions, decision)
	}
	result.Conflicts = append(result.Conflicts, kernelResult.Conflicts...)
	return result
}

func allowMovieCollectionLeadingEpisodes(candidates []database.RecognitionCandidate, evidence []database.RecognitionEvidence, constraints ConstraintResult) {
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.CandidateType) != CandidateTypeWork || strings.TrimSpace(candidate.CandidateRole) != WorkKindMovie {
			continue
		}
		if !hasDirectoryReductionAssignment(candidate, evidence, "movie_collection") || !hasWeakLeadingEpisodeEvidence(candidate, evidence) || !movieCollectionEvidenceAllowsMovie(candidate, evidence) {
			continue
		}
		delete(constraints.RejectedCandidates, strings.TrimSpace(candidate.CandidateKey))
	}
}

func applyCandidateConfidence(scores map[string]float64, candidates []database.RecognitionCandidate) {
	for _, candidate := range candidates {
		if candidate.Confidence == nil || *candidate.Confidence <= scores[candidate.CandidateKey] {
			continue
		}
		scores[candidate.CandidateKey] = *candidate.Confidence
	}
}

func fallbackDecision(manifestID uint, candidate database.RecognitionCandidate) database.RecognitionDecision {
	outcome := DecisionOutcomeProvisional
	reason := "candidate remains provisional pending more resolver evidence"
	if candidate.SupersededAt != nil {
		outcome = "superseded"
		reason = "candidate was superseded by newer manifest evidence"
	} else if candidate.Confidence == nil {
		outcome = DecisionOutcomeUnmatched
		reason = "candidate has no usable confidence or acceptance evidence"
	} else if *candidate.Confidence < 0.2 {
		outcome = DecisionOutcomeReviewRequired
		reason = "candidate has only weak evidence and requires resolver review"
	}
	return database.RecognitionDecision{ManifestID: manifestID, CandidateID: uintPtr(candidate.ID), DecisionType: "resolver_outcome", Outcome: outcome, TargetKind: candidate.CandidateType, TargetKey: candidate.CandidateKey, TargetMetadataID: candidate.TargetMetadataID, TargetResourceID: candidate.TargetResourceID, Confidence: candidate.Confidence, Reason: reason, EvidenceJSON: candidate.EvidenceJSON, AlternativesJSON: candidate.AlternativesJSON}
}

func detectBlockingConflicts(graph ManifestGraph) []database.RecognitionConflict {
	conflicts := make([]database.RecognitionConflict, 0)
	byFile := make(map[uint]map[string]map[string]struct{})
	for _, evidence := range graph.Evidence {
		if evidence.InventoryFileID == nil {
			continue
		}
		key := strings.TrimSpace(evidence.EvidenceKey)
		if !isConflictEvidenceKey(key) || strings.TrimSpace(evidence.EvidenceValue) == "" {
			continue
		}
		fileID := *evidence.InventoryFileID
		if byFile[fileID] == nil {
			byFile[fileID] = make(map[string]map[string]struct{})
		}
		if byFile[fileID][key] == nil {
			byFile[fileID][key] = make(map[string]struct{})
		}
		byFile[fileID][key][strings.TrimSpace(evidence.EvidenceValue)] = struct{}{}
	}
	for fileID, valuesByKey := range byFile {
		for key, values := range valuesByKey {
			if len(values) <= 1 {
				continue
			}
			conflicts = append(conflicts, database.RecognitionConflict{ManifestID: graph.Manifest.ID, ConflictKey: "file:" + stringUint(fileID) + ":" + key, ConflictType: conflictTypeForEvidenceKey(key), Severity: "blocking", Status: "open", Reason: "conflicting " + key + " evidence for inventory file", EvidenceJSON: mustJSON(values)})
		}
	}
	return conflicts
}

func blockingConflictForCandidate(candidate database.RecognitionCandidate, conflicts []database.RecognitionConflict) (database.RecognitionConflict, bool) {
	if candidate.PrimaryInventoryID == nil {
		return database.RecognitionConflict{}, false
	}
	prefix := "file:" + stringUint(*candidate.PrimaryInventoryID) + ":"
	for _, conflict := range conflicts {
		if strings.TrimSpace(conflict.Severity) == "blocking" && strings.HasPrefix(strings.TrimSpace(conflict.ConflictKey), prefix) {
			return conflict, true
		}
	}
	return database.RecognitionConflict{}, false
}

func isConflictEvidenceKey(key string) bool {
	return strings.HasPrefix(key, "external_id:") || key == "year" || key == "media_type" || key == "episode_number" || key == "season_number"
}

func conflictTypeForEvidenceKey(key string) string {
	if strings.HasPrefix(key, "external_id:") {
		return "external_identity_conflict"
	}
	return strings.TrimSpace(key) + "_conflict"
}

func stringUint(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}

func acceptedByLocalGate(candidate database.RecognitionCandidate, evidence []database.RecognitionEvidence) (bool, string) {
	switch strings.TrimSpace(candidate.CandidateType) {
	case CandidateTypeDuplicateBinary:
		return true, "same binary hash evidence anchors duplicate candidate"
	case CandidateTypeEpisode:
		if hasExternalIDEvidence(candidate, evidence) {
			return true, "external identity evidence anchors episode identity"
		}
		if hasDirectoryReductionAssignment(candidate, evidence, "episode_multi_version") {
			return true, "directory reduction grouped sibling files under the same episode identity"
		}
		if hasDirectoryReductionAssignment(candidate, evidence, "single_episode_identity") {
			return true, "directory reduction isolated a single episode identity from directory noise"
		}
		if strings.TrimSpace(candidate.CanonicalKey) != "" && hasEvidenceKeys(candidate, evidence, "season_number", "episode_number") {
			return true, "series season episode tuple anchors episode identity"
		}
	case CandidateTypeWork:
		if hasExternalIDEvidence(candidate, evidence) {
			return true, "external identity evidence anchors work identity"
		}
		if strings.TrimSpace(candidate.CandidateRole) == WorkKindSeries && strings.TrimSpace(candidate.CanonicalKey) != "" && hasEvidenceKeys(candidate, evidence, "series_title") {
			return true, "series title evidence anchors series identity"
		}
		if strings.TrimSpace(candidate.CandidateRole) == WorkKindSeason && strings.TrimSpace(candidate.CanonicalKey) != "" && hasEvidenceKeys(candidate, evidence, "series_title", "season_number") {
			return true, "series title and season number evidence anchor season identity"
		}
		if strings.TrimSpace(candidate.CandidateRole) == WorkKindMovie && strings.TrimSpace(candidate.CanonicalKey) != "" && hasDirectoryReductionAssignment(candidate, evidence, "movie_collection") && hasEvidenceKeys(candidate, evidence, "title") && movieCollectionEvidenceAllowsMovie(candidate, evidence) {
			return true, "directory reduction identified an independent movie within a movie collection folder"
		}
		if strings.TrimSpace(candidate.CandidateRole) == WorkKindMovie && hasDirectoryReductionAssignment(candidate, evidence, "movie_multi_version") {
			return true, "directory reduction grouped sibling files under the same movie identity"
		}
		if strings.TrimSpace(candidate.CandidateRole) == WorkKindMovie && hasDirectoryReductionAssignment(candidate, evidence, "single_work_identity") {
			return true, "directory reduction isolated a single movie identity from directory noise"
		}
		if strings.TrimSpace(candidate.CandidateRole) == WorkKindMovie && strings.TrimSpace(candidate.CanonicalKey) != "" && hasEvidenceKeys(candidate, evidence, "title", "year") {
			return true, "normalized movie title and year evidence anchors work identity"
		}
		if strings.TrimSpace(candidate.CandidateRole) == WorkKindMovie && strings.TrimSpace(candidate.CanonicalKey) != "" && hasEvidenceKeys(candidate, evidence, "title") && !hasAnyEvidenceKeys(candidate, evidence, "season_number", "episode_number") && candidate.Confidence != nil && *candidate.Confidence >= 0.75 {
			return true, "high-confidence movie title evidence anchors work identity"
		}
	case CandidateTypeVariant, CandidateTypeEdition:
		if hasDirectoryReductionSource(candidate, evidence) && strings.TrimSpace(candidate.ParentCandidateKey) != "" && (strings.TrimSpace(candidate.VariantKey) != "" || strings.TrimSpace(candidate.EditionKey) != "") {
			return true, "directory reduction identified a version trait attached to the parent identity"
		}
		if strings.TrimSpace(candidate.ParentCandidateKey) != "" && (strings.TrimSpace(candidate.VariantKey) != "" || strings.TrimSpace(candidate.EditionKey) != "") {
			return true, "variant or edition traits attach to parent work candidate"
		}
	case CandidateTypePlayableResource:
		if hasDirectoryReductionSource(candidate, evidence) && strings.TrimSpace(candidate.ParentCandidateKey) != "" && candidate.PrimaryInventoryID != nil {
			return true, "directory reduction attached the playable resource to an accepted grouped identity"
		}
		if strings.TrimSpace(candidate.ParentCandidateKey) != "" && candidate.PrimaryInventoryID != nil {
			return true, "playable resource attaches to accepted parent identity candidate"
		}
	case CandidateTypeSupplemental:
		if strings.TrimSpace(candidate.ParentCandidateKey) != "" && candidate.PrimaryInventoryID != nil && strings.TrimSpace(candidate.CandidateRole) != "" {
			return true, "supplemental resource attaches to accepted parent identity candidate"
		}
	}
	return false, ""
}

func decisionFromGate(manifestID uint, candidate database.RecognitionCandidate, reason string, outcome string) database.RecognitionDecision {
	return database.RecognitionDecision{ManifestID: manifestID, CandidateID: uintPtr(candidate.ID), DecisionType: "resolver_gate", Outcome: outcome, TargetKind: candidate.CandidateType, TargetKey: candidate.CandidateKey, TargetMetadataID: candidate.TargetMetadataID, TargetResourceID: candidate.TargetResourceID, Confidence: candidate.Confidence, Reason: reason, EvidenceJSON: candidate.EvidenceJSON, AlternativesJSON: candidate.AlternativesJSON}
}

func hasExternalIDEvidence(candidate database.RecognitionCandidate, evidence []database.RecognitionEvidence) bool {
	for _, item := range evidence {
		if !evidenceBelongsToCandidate(candidate, item) {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(item.EvidenceKey), "external_id:") && strings.TrimSpace(item.EvidenceValue) != "" {
			return true
		}
	}
	return false
}

func hasEvidenceKeys(candidate database.RecognitionCandidate, evidence []database.RecognitionEvidence, keys ...string) bool {
	needed := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		needed[key] = struct{}{}
	}
	for _, item := range evidence {
		if !evidenceBelongsToCandidate(candidate, item) {
			continue
		}
		delete(needed, strings.TrimSpace(item.EvidenceKey))
	}
	return len(needed) == 0
}

func hasAnyEvidenceKeys(candidate database.RecognitionCandidate, evidence []database.RecognitionEvidence, keys ...string) bool {
	needed := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		needed[key] = struct{}{}
	}
	for _, item := range evidence {
		if !evidenceBelongsToCandidate(candidate, item) {
			continue
		}
		if _, ok := needed[strings.TrimSpace(item.EvidenceKey)]; ok {
			return true
		}
	}
	return false
}

func movieCollectionEvidenceAllowsMovie(candidate database.RecognitionCandidate, evidence []database.RecognitionEvidence) bool {
	for _, item := range evidence {
		if !evidenceBelongsToCandidate(candidate, item) {
			continue
		}
		switch strings.TrimSpace(item.EvidenceKey) {
		case "season_number":
			return false
		case "episode_number":
			if !hasWeakLeadingEpisodeEvidence(candidate, evidence) {
				return false
			}
		}
	}
	return true
}

func hasWeakLeadingEpisodeEvidence(candidate database.RecognitionCandidate, evidence []database.RecognitionEvidence) bool {
	for _, item := range evidence {
		if !evidenceBelongsToCandidate(candidate, item) {
			continue
		}
		if strings.TrimSpace(item.EvidenceKey) == "episode_source" && strings.TrimSpace(item.EvidenceValue) == "leading_numeric" {
			return true
		}
	}
	return false
}

func hasDirectoryReductionAssignment(candidate database.RecognitionCandidate, evidence []database.RecognitionEvidence, assignment string) bool {
	for _, item := range evidence {
		if !evidenceBelongsToCandidate(candidate, item) {
			continue
		}
		if strings.TrimSpace(item.EvidenceSource) == "directory_reduction" && strings.TrimSpace(item.EvidenceKey) == "assignment" && strings.TrimSpace(item.EvidenceValue) == strings.TrimSpace(assignment) {
			return true
		}
	}
	return false
}

func hasDirectoryReductionSource(candidate database.RecognitionCandidate, evidence []database.RecognitionEvidence) bool {
	for _, item := range evidence {
		if evidenceBelongsToCandidate(candidate, item) && strings.TrimSpace(item.EvidenceSource) == "directory_reduction" {
			return true
		}
	}
	return false
}

func evidenceBelongsToCandidate(candidate database.RecognitionCandidate, evidence database.RecognitionEvidence) bool {
	if candidate.PrimaryInventoryID != nil && evidence.InventoryFileID != nil && *candidate.PrimaryInventoryID == *evidence.InventoryFileID {
		return true
	}
	key := strings.TrimSpace(evidence.EvidenceKey)
	return key != "" && (key == strings.TrimSpace(candidate.CandidateKey) || key == strings.TrimSpace(candidate.CanonicalKey) || key == strings.TrimSpace(candidate.ParentCandidateKey))
}

func (r *Resolver) matchingRule(candidate database.RecognitionCandidate) (database.RecognitionRule, bool) {
	for _, rule := range r.rules {
		if !rule.Enabled {
			continue
		}
		if strings.TrimSpace(rule.CandidateType) != "" && strings.TrimSpace(rule.CandidateType) != strings.TrimSpace(candidate.CandidateType) {
			continue
		}
		payload := strings.TrimSpace(rule.PayloadJSON)
		candidateKeyMatches := strings.TrimSpace(candidate.CandidateKey) != "" && strings.Contains(payload, strings.TrimSpace(candidate.CandidateKey))
		canonicalKeyMatches := strings.TrimSpace(candidate.CanonicalKey) != "" && strings.Contains(payload, strings.TrimSpace(candidate.CanonicalKey))
		if payload != "" && !candidateKeyMatches && !canonicalKeyMatches {
			continue
		}
		return rule, true
	}
	return database.RecognitionRule{}, false
}

func decisionFromRule(manifestID uint, candidate database.RecognitionCandidate, rule database.RecognitionRule) (database.RecognitionDecision, database.RecognitionConflict) {
	outcome := DecisionOutcomeAccepted
	reason := "manual resolver rule accepted candidate"
	switch strings.TrimSpace(rule.Action) {
	case RuleActionSplit, RuleActionReject:
		outcome = DecisionOutcomeRejected
		reason = "manual resolver rule rejects or splits candidate"
	case RuleActionClassifyAs, RuleActionMerge, RuleActionAccept:
		outcome = DecisionOutcomeAccepted
	}
	decision := database.RecognitionDecision{ManifestID: manifestID, CandidateID: uintPtr(candidate.ID), DecisionType: "resolver_rule", Outcome: outcome, TargetKind: candidate.CandidateType, TargetKey: candidate.CandidateKey, TargetMetadataID: candidate.TargetMetadataID, TargetResourceID: candidate.TargetResourceID, Confidence: candidate.Confidence, Reason: reason, EvidenceJSON: rule.EvidenceJSON, AlternativesJSON: candidate.AlternativesJSON}
	if outcome != DecisionOutcomeRejected {
		return decision, database.RecognitionConflict{}
	}
	conflict := database.RecognitionConflict{ManifestID: manifestID, CandidateID: uintPtr(candidate.ID), ConflictKey: "rule:" + strings.TrimSpace(rule.RuleKey), ConflictType: "manual_rule", Severity: "blocking", Status: "open", Reason: reason, EvidenceJSON: rule.EvidenceJSON}
	return decision, conflict
}

func sortedRules(rules []database.RecognitionRule) []database.RecognitionRule {
	items := append([]database.RecognitionRule(nil), rules...)
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && ruleLess(items[j], items[j-1]); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
	return items
}

func ruleLess(left database.RecognitionRule, right database.RecognitionRule) bool {
	if left.Priority != right.Priority {
		return left.Priority < right.Priority
	}
	return left.ID < right.ID
}
