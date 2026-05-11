package library

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/recognition"
	"gorm.io/gorm"
)

const evidenceSourceDirectoryReduction = "directory_reduction"

const (
	directoryReductionReviewAttachmentsOnly             = "attachments_only"
	directoryReductionReviewExtrasMixed                 = "extras_mixed"
	directoryReductionReviewSingleWorkWithNoise         = "single_work_with_noise"
	directoryReductionReviewCrossIdentityConflict       = "cross_identity_conflict"
	directoryReductionReviewAmbiguousSeriesVsCollection = "ambiguous_series_vs_collection"
	directoryReductionAssignmentSingleWorkIdentity      = "single_work_identity"
	directoryReductionAssignmentSingleEpisodeIdentity   = "single_episode_identity"
)

type directoryReductionGroup struct {
	ParentKey  string
	VariantKey string
	EditionKey string
	Members    []uint
	Confidence float64
	Assignment string
	TargetKey  string
	Reason     string
	Residual   []string
	ParentPath string
}

type directoryReductionDecision struct {
	ParentPath     string
	Interpretation string
	Status         string
	Confidence     float64
	Reason         string
	Affected       []string
	Residual       []string
	Alternatives   []string
	Evidence       map[string]any
}

type directoryReductionMember struct {
	file       database.InventoryFile
	signal     database.InventoryFileSignal
	parentKey  string
	variantKey string
	editionKey string
}

func directoryReductionContextEvidence(files []database.InventoryFile, signals map[uint]database.InventoryFileSignal) map[uint][]recognition.ContextEvidence {
	if len(files) < 2 {
		return nil
	}
	groups := reduceDirectoryCandidates(files, signals)
	if collection := mixedMovieCollectionContextEvidence(files, signals); len(collection) > 0 {
		return mergeDirectoryContextEvidence(collection, groupsDirectoryContextEvidence(files, groups))
	}
	if len(groups) == 0 {
		decision, ok := inferResidualDirectoryDecision(files, signals)
		if !ok {
			return nil
		}
		if single := singleIdentityResidualContextEvidence(files, signals, decision); len(single) > 0 {
			return single
		}
		return residualDirectoryContextEvidence(files, signals, decision)
	}
	return groupsDirectoryContextEvidence(files, groups)
}

func groupsDirectoryContextEvidence(files []database.InventoryFile, groups []directoryReductionGroup) map[uint][]recognition.ContextEvidence {
	if len(groups) == 0 {
		return nil
	}
	result := make(map[uint][]recognition.ContextEvidence)
	for _, group := range groups {
		payload := map[string]any{
			"reason":         group.Reason,
			"residual_paths": append([]string(nil), group.Residual...),
			"parent_path":    group.ParentPath,
		}
		for _, fileID := range group.Members {
			result[fileID] = append(result[fileID], recognition.ContextEvidence{
				Source:      evidenceSourceDirectoryReduction,
				Assignment:  group.Assignment,
				TargetKey:   group.TargetKey,
				ParentKey:   group.ParentKey,
				VariantKey:  group.VariantKey,
				EditionKey:  group.EditionKey,
				ReviewState: "auto",
				Confidence:  floatPtr(group.Confidence),
				Payload:     payload,
			})
		}
	}
	return result
}

func mixedMovieCollectionContextEvidence(files []database.InventoryFile, signals map[uint]database.InventoryFileSignal) map[uint][]recognition.ContextEvidence {
	movieParents := make(map[string]struct{})
	for _, file := range files {
		if !movieCollectionCandidateFile(file, signals[file.ID]) {
			continue
		}
		if key := movieCollectionParentKey(file, signals[file.ID]); key != "" {
			movieParents[key] = struct{}{}
		}
	}
	if len(movieParents) < 2 {
		return nil
	}
	decision := directoryReductionDecision{ParentPath: commonRecognitionScopePath("", files), Interpretation: pathTreeWorkGroupShapeMovieCollection, Status: scanDecisionStatusProvisional, Confidence: 0.8, Reason: "directory reduction leftover structure looks like an independent movie collection", Alternatives: []string{"movie_multi_version", pathTreeWorkGroupShapeReview}}
	if strings.TrimSpace(decision.ParentPath) == "" && len(files) > 0 {
		decision.ParentPath = path.Dir(strings.TrimSpace(files[0].StoragePath))
	}
	return residualDirectoryContextEvidence(files, signals, decision)
}

func movieCollectionCandidateFile(file database.InventoryFile, signal database.InventoryFileSignal) bool {
	if file.ID == 0 || file.ContentClass != SourceContentClassVideo || !isVideoFile(file.StoragePath) || signal.IsExtra || strings.TrimSpace(signal.Role) != "" && strings.TrimSpace(signal.Role) != "main" {
		return false
	}
	return movieCollectionParentKey(file, signal) != ""
}

func movieCollectionParentKey(file database.InventoryFile, signal database.InventoryFileSignal) string {
	title := strings.TrimSpace(signal.TitleCandidate)
	if title == "" {
		return ""
	}
	if signal.SeasonNumber != nil {
		return ""
	}
	if signal.EpisodeNumber != nil && strings.TrimSpace(signal.EpisodeSource) != "leading_numeric" {
		return ""
	}
	return recognition.MovieWorkKey(recognition.MovieWorkInput{Title: title, Year: signal.Year})
}

func mergeDirectoryContextEvidence(primary map[uint][]recognition.ContextEvidence, secondary map[uint][]recognition.ContextEvidence) map[uint][]recognition.ContextEvidence {
	if len(primary) == 0 {
		return secondary
	}
	for fileID, evidence := range secondary {
		primary[fileID] = append(primary[fileID], evidence...)
	}
	return primary
}

func singleIdentityResidualContextEvidence(files []database.InventoryFile, signals map[uint]database.InventoryFileSignal, decision directoryReductionDecision) map[uint][]recognition.ContextEvidence {
	if strings.TrimSpace(decision.Interpretation) != pathTreeWorkGroupShapeReview {
		return nil
	}
	reviewSubtype := strings.TrimSpace(fmtSprintAny(decision.Evidence["review_subtype"]))
	if reviewSubtype != directoryReductionReviewSingleWorkWithNoise && reviewSubtype != directoryReductionReviewExtrasMixed {
		return nil
	}
	result := make(map[uint][]recognition.ContextEvidence)
	for _, file := range files {
		signal := signals[file.ID]
		if file.ID == 0 || signal.IsExtra || strings.TrimSpace(signal.Role) != "" && strings.TrimSpace(signal.Role) != "main" {
			continue
		}
		parentKey := reductionParentKey(file, signal)
		if parentKey == "" {
			continue
		}
		assignment := directoryReductionAssignmentSingleWorkIdentity
		if strings.HasPrefix(parentKey, recognition.CandidateTypeEpisode+":") {
			assignment = directoryReductionAssignmentSingleEpisodeIdentity
		}
		payload := map[string]any{"reason": decision.Reason, "parent_path": decision.ParentPath, "review_subtype": reviewSubtype, "residual_paths": append([]string(nil), decision.Residual...), "alternatives": append([]string(nil), decision.Alternatives...)}
		result[file.ID] = append(result[file.ID], recognition.ContextEvidence{Source: evidenceSourceDirectoryReduction, Assignment: assignment, TargetKey: strings.TrimSpace(decision.ParentPath), ParentKey: parentKey, ReviewState: "auto", Confidence: floatPtr(0.74), Payload: payload})
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func residualDirectoryContextEvidence(files []database.InventoryFile, signals map[uint]database.InventoryFileSignal, decision directoryReductionDecision) map[uint][]recognition.ContextEvidence {
	result := make(map[uint][]recognition.ContextEvidence)
	for _, file := range files {
		signal := signals[file.ID]
		if file.ID == 0 || signal.IsExtra || strings.TrimSpace(signal.Role) != "" && strings.TrimSpace(signal.Role) != "main" {
			continue
		}
		assignment := strings.TrimSpace(decision.Interpretation)
		if assignment != pathTreeWorkGroupShapeMovieCollection && assignment != pathTreeWorkGroupShapeSeries {
			continue
		}
		parentKey := reductionParentKey(file, signal)
		if assignment == pathTreeWorkGroupShapeMovieCollection {
			parentKey = movieCollectionParentKey(file, signal)
		}
		if parentKey == "" {
			continue
		}
		payload := map[string]any{"reason": decision.Reason, "parent_path": decision.ParentPath, "residual_paths": append([]string(nil), decision.Residual...), "alternatives": append([]string(nil), decision.Alternatives...)}
		result[file.ID] = append(result[file.ID], recognition.ContextEvidence{Source: evidenceSourceDirectoryReduction, Assignment: assignment, TargetKey: strings.TrimSpace(decision.ParentPath), ParentKey: parentKey, ReviewState: "auto", Confidence: floatPtr(decision.Confidence), Payload: payload})
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func fmtSprintAny(value any) string {
	if value == nil {
		return ""
	}
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "<nil>" {
		return ""
	}
	return text
}

func saveDirectoryReductionDecision(ctx context.Context, db *gorm.DB, libraryID uint, decision directoryReductionDecision) error {
	if db == nil || libraryID == 0 || strings.TrimSpace(decision.ParentPath) == "" || strings.TrimSpace(decision.Interpretation) == "" {
		return nil
	}
	confidence := decision.Confidence
	return db.WithContext(ctx).Create(&database.ClassificationDecision{
		LibraryID:         libraryID,
		SourcePath:        strings.TrimSpace(decision.ParentPath),
		DecisionType:      "directory_reduction",
		CandidateType:     strings.TrimSpace(decision.Interpretation),
		TargetKind:        "directory",
		TargetKey:         strings.TrimSpace(decision.ParentPath),
		Status:            firstNonEmptyString(decision.Status, scanDecisionStatusProvisional),
		Confidence:        &confidence,
		AlternativesJSON:  mustJSON(decision.Alternatives),
		EvidenceJSON:      mustJSON(decision.Evidence),
		AffectedFilesJSON: mustJSON(decision.Affected),
		Reason:            strings.TrimSpace(decision.Reason),
	}).Error
}

func directoryReductionScopePath(decision directoryReductionDecision, fallback string) string {
	parentPath := strings.TrimSpace(decision.ParentPath)
	if parentPath == "" {
		return strings.TrimSpace(fallback)
	}
	switch strings.TrimSpace(decision.Interpretation) {
	case pathTreeWorkGroupShapeMovieCollection, pathTreeWorkGroupShapeSeries:
		return parentPath
	default:
		if strings.TrimSpace(fmtSprintAny(decision.Evidence["review_subtype"])) == directoryReductionReviewAmbiguousSeriesVsCollection {
			return parentPath
		}
		return strings.TrimSpace(fallback)
	}
}

func directoryReductionExcludedFileIDs(files []database.InventoryFile, signals map[uint]database.InventoryFileSignal) map[uint]string {
	decision, ok := inferResidualDirectoryDecision(files, signals)
	if !ok || strings.TrimSpace(decision.Interpretation) != pathTreeWorkGroupShapeReview {
		return nil
	}
	subtype := strings.TrimSpace(fmtSprintAny(decision.Evidence["review_subtype"]))
	if subtype != directoryReductionReviewExtrasMixed && subtype != directoryReductionReviewAttachmentsOnly {
		return nil
	}
	excluded := make(map[uint]string)
	for _, file := range files {
		signal := signals[file.ID]
		if file.ID == 0 {
			continue
		}
		if subtype == directoryReductionReviewExtrasMixed && (signal.IsExtra || strings.TrimSpace(signal.Role) != "" && strings.TrimSpace(signal.Role) != "main") {
			excluded[file.ID] = "directory_reduction_extras"
			continue
		}
		if subtype == directoryReductionReviewAttachmentsOnly && (file.ContentClass != SourceContentClassVideo || !isVideoFile(file.StoragePath)) {
			excluded[file.ID] = "directory_reduction_attachments"
		}
	}
	if len(excluded) == 0 {
		return nil
	}
	return excluded
}

func directoryReductionDecisionForFiles(files []database.InventoryFile, signals map[uint]database.InventoryFileSignal) (directoryReductionDecision, bool) {
	groups := reduceDirectoryCandidates(files, signals)
	if len(groups) == 0 {
		return inferResidualDirectoryDecision(files, signals)
	}
	affected := make([]string, 0, len(groups))
	groupKinds := make(map[string]int)
	parentPath := commonRecognitionScopePath("", files)
	if parentPath == "" {
		parentPath = strings.TrimSpace(groups[0].ParentPath)
	}
	confidence := groups[0].Confidence
	residual := append([]string(nil), groups[0].Residual...)
	for _, group := range groups {
		groupKinds[group.Assignment]++
		if group.Confidence < confidence {
			confidence = group.Confidence
		}
		affected = append(affected, memberPathsForGroup(files, group)...)
	}
	sort.Strings(affected)
	if len(groupKinds) == 1 {
		for assignment := range groupKinds {
			return directoryReductionDecision{ParentPath: parentPath, Interpretation: assignment, Status: scanDecisionStatusProvisional, Confidence: confidence, Reason: "directory reduction grouped sibling files before resolver materialization", Affected: affected, Residual: residual, Alternatives: inferReductionAlternatives(assignment), Evidence: map[string]any{"source": evidenceSourceDirectoryReduction, "group_count": len(groups), "residual_paths": residual}}, true
		}
	}
	return directoryReductionDecision{ParentPath: parentPath, Interpretation: pathTreeWorkGroupShapeReview, Status: scanDecisionStatusProvisional, Confidence: 0.52, Reason: "directory reduction found competing grouped interpretations", Affected: affected, Residual: residual, Alternatives: []string{"movie_multi_version", "episode_multi_version", pathTreeWorkGroupShapeMovieCollection, pathTreeWorkGroupShapeSeries}, Evidence: map[string]any{"source": evidenceSourceDirectoryReduction, "group_kinds": groupKinds, "residual_paths": residual}}, true
}

func reduceDirectoryCandidates(files []database.InventoryFile, signals map[uint]database.InventoryFileSignal) []directoryReductionGroup {
	playable := make([]database.InventoryFile, 0, len(files))
	for _, file := range files {
		signal := signals[file.ID]
		if file.ID == 0 || file.ContentClass != SourceContentClassVideo || !isVideoFile(file.StoragePath) || signal.IsExtra || strings.TrimSpace(signal.Role) != "" && strings.TrimSpace(signal.Role) != "main" {
			continue
		}
		playable = append(playable, file)
	}
	if len(playable) < 2 {
		return nil
	}
	byParent := make(map[string][]directoryReductionMember)
	for _, file := range playable {
		signal := signals[file.ID]
		parentKey := reductionParentKey(file, signal)
		if parentKey == "" {
			continue
		}
		byParent[parentKey] = append(byParent[parentKey], directoryReductionMember{file: file, signal: signal, parentKey: parentKey, variantKey: recognition.VariantKey(recognition.VariantInput{Quality: signal.Quality, Codec: signal.Codec, Audio: signal.Audio, Subtitle: signal.Subtitle, HDR: signal.HDR, Container: file.Container, ReleaseGroup: signal.ReleaseGroup}), editionKey: recognition.EditionKey(signal.Edition)})
	}
	groups := make([]directoryReductionGroup, 0)
	for parentKey, members := range byParent {
		if len(members) < 2 {
			continue
		}
		residual := reducedResidualPaths(files, members)
		assignment := "directory_group"
		confidence := 0.84
		reason := "directory reduction grouped sibling files under a shared parent identity"
		if strings.HasPrefix(parentKey, recognition.CandidateTypeEpisode+":") {
			assignment = "episode_multi_version"
			confidence = 0.9
			reason = "directory reduction grouped sibling files under the same episode tuple"
		} else if strings.HasPrefix(parentKey, recognition.CandidateTypeWork+":"+recognition.WorkKindMovie+":") {
			assignment = "movie_multi_version"
			confidence = 0.88
			reason = "directory reduction grouped sibling files under the same movie identity"
		}
		for _, member := range members {
			groups = append(groups, directoryReductionGroup{ParentKey: parentKey, VariantKey: member.variantKey, EditionKey: member.editionKey, Members: []uint{member.file.ID}, Confidence: confidence, Assignment: assignment, TargetKey: parentKey, Reason: reason, Residual: residual, ParentPath: path.Dir(member.file.StoragePath)})
		}
	}
	return groups
}

func inferResidualDirectoryDecision(files []database.InventoryFile, signals map[uint]database.InventoryFileSignal) (directoryReductionDecision, bool) {
	movieParents := make(map[string]struct{})
	episodeParents := make(map[string]struct{})
	affected := make([]string, 0, len(files))
	parentPath := commonRecognitionScopePath("", files)
	for _, file := range files {
		if file.ID == 0 || strings.TrimSpace(file.StoragePath) == "" {
			continue
		}
		if parentPath == "" {
			parentPath = path.Dir(strings.TrimSpace(file.StoragePath))
		}
		affected = append(affected, strings.TrimSpace(file.StoragePath))
		signal := signals[file.ID]
		if signal.IsExtra || strings.TrimSpace(signal.Role) != "" && strings.TrimSpace(signal.Role) != "main" {
			continue
		}
		if key := reductionParentKey(file, signal); strings.HasPrefix(key, recognition.CandidateTypeEpisode+":") {
			episodeParents[key] = struct{}{}
		} else if strings.HasPrefix(key, recognition.CandidateTypeWork+":"+recognition.WorkKindMovie+":") {
			movieParents[key] = struct{}{}
		} else if key := movieCollectionParentKey(file, signal); key != "" {
			movieParents[key] = struct{}{}
		}
	}
	sort.Strings(affected)
	switch {
	case len(episodeParents) >= 2:
		return directoryReductionDecision{ParentPath: parentPath, Interpretation: pathTreeWorkGroupShapeSeries, Status: scanDecisionStatusProvisional, Confidence: 0.82, Reason: "directory reduction leftover structure looks like a series hierarchy", Affected: affected, Alternatives: []string{"episode_multi_version", pathTreeWorkGroupShapeReview}, Evidence: map[string]any{"source": evidenceSourceDirectoryReduction, "episode_parent_count": len(episodeParents)}}, true
	case len(movieParents) >= 2:
		return directoryReductionDecision{ParentPath: parentPath, Interpretation: pathTreeWorkGroupShapeMovieCollection, Status: scanDecisionStatusProvisional, Confidence: 0.8, Reason: "directory reduction leftover structure looks like an independent movie collection", Affected: affected, Alternatives: []string{"movie_multi_version", pathTreeWorkGroupShapeReview}, Evidence: map[string]any{"source": evidenceSourceDirectoryReduction, "movie_parent_count": len(movieParents)}}, true
	case len(movieParents) > 0 && len(episodeParents) > 0:
		return directoryReductionDecision{ParentPath: parentPath, Interpretation: pathTreeWorkGroupShapeReview, Status: scanDecisionStatusProvisional, Confidence: 0.42, Reason: "directory reduction found movie and episode identities mixed in the same directory", Affected: affected, Alternatives: []string{pathTreeWorkGroupShapeMovieCollection, pathTreeWorkGroupShapeSeries}, Evidence: map[string]any{"source": evidenceSourceDirectoryReduction, "review_subtype": directoryReductionReviewCrossIdentityConflict, "movie_parent_count": len(movieParents), "episode_parent_count": len(episodeParents)}}, true
	case len(episodeParents) == 1 && len(movieParents) >= 1:
		return directoryReductionDecision{ParentPath: parentPath, Interpretation: pathTreeWorkGroupShapeReview, Status: scanDecisionStatusProvisional, Confidence: 0.46, Reason: "directory reduction could not disambiguate between a series grouping and an independent movie collection", Affected: affected, Alternatives: []string{pathTreeWorkGroupShapeSeries, pathTreeWorkGroupShapeMovieCollection}, Evidence: map[string]any{"source": evidenceSourceDirectoryReduction, "review_subtype": directoryReductionReviewAmbiguousSeriesVsCollection, "movie_parent_count": len(movieParents), "episode_parent_count": len(episodeParents)}}, true
	case len(movieParents) == 1 && len(affected) > 1:
		subtype, reason := classifySingleWorkNoise(files, signals, movieParents, episodeParents)
		return directoryReductionDecision{ParentPath: parentPath, Interpretation: pathTreeWorkGroupShapeReview, Status: scanDecisionStatusProvisional, Confidence: 0.5, Reason: reason, Affected: affected, Alternatives: []string{"movie_multi_version", pathTreeWorkGroupShapeMovieCollection}, Evidence: map[string]any{"source": evidenceSourceDirectoryReduction, "review_subtype": subtype, "movie_parent_count": len(movieParents), "episode_parent_count": len(episodeParents)}}, true
	case len(movieParents) == 0 && len(episodeParents) == 0 && len(affected) > 0:
		subtype := classifyAttachmentOnlyOrExtras(files, signals)
		evidence := map[string]any{"source": evidenceSourceDirectoryReduction, "review_subtype": subtype}
		if subtype == directoryReductionReviewAttachmentsOnly {
			evidence["artwork_safe"] = true
			evidence["identity_excluded_reason"] = "attachments retained for local artwork evidence only"
		}
		return directoryReductionDecision{ParentPath: parentPath, Interpretation: pathTreeWorkGroupShapeReview, Status: scanDecisionStatusProvisional, Confidence: 0.35, Reason: "directory reduction only found attachments or extras without a stable playable identity", Affected: affected, Alternatives: []string{pathTreeWorkGroupShapeMovieCollection, pathTreeWorkGroupShapeSeries}, Evidence: evidence}, true
	default:
		return directoryReductionDecision{}, false
	}
}

func classifySingleWorkNoise(files []database.InventoryFile, signals map[uint]database.InventoryFileSignal, movieParents map[string]struct{}, episodeParents map[string]struct{}) (string, string) {
	if classifyAttachmentOnlyOrExtras(files, signals) == directoryReductionReviewExtrasMixed {
		return directoryReductionReviewExtrasMixed, "directory reduction found extras mixed around a single playable identity"
	}
	return directoryReductionReviewSingleWorkWithNoise, "directory reduction found extra residual files around a single playable identity"
}

func classifyAttachmentOnlyOrExtras(files []database.InventoryFile, signals map[uint]database.InventoryFileSignal) string {
	hasExtra := false
	hasAttachment := false
	for _, file := range files {
		signal := signals[file.ID]
		if signal.IsExtra || strings.TrimSpace(signal.Role) != "" && strings.TrimSpace(signal.Role) != "main" {
			hasExtra = true
			continue
		}
		if file.ContentClass != SourceContentClassVideo || !isVideoFile(file.StoragePath) {
			hasAttachment = true
		}
	}
	if hasExtra {
		return directoryReductionReviewExtrasMixed
	}
	if hasAttachment {
		return directoryReductionReviewAttachmentsOnly
	}
	return directoryReductionReviewSingleWorkWithNoise
}

func reductionParentKey(file database.InventoryFile, signal database.InventoryFileSignal) string {
	if strings.TrimSpace(signal.TitleCandidate) == "" {
		return ""
	}
	if signal.SeasonNumber != nil && signal.EpisodeNumber != nil {
		return recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: signal.TitleCandidate, SeasonNumber: *signal.SeasonNumber, EpisodeNumber: *signal.EpisodeNumber})
	}
	if signal.EpisodeNumber == nil {
		return recognition.MovieWorkKey(recognition.MovieWorkInput{Title: signal.TitleCandidate, Year: signal.Year})
	}
	return ""
}

func reducedResidualPaths(all []database.InventoryFile, group []directoryReductionMember) []string {
	consumed := make(map[uint]struct{}, len(group))
	for _, member := range group {
		consumed[member.file.ID] = struct{}{}
	}
	residual := make([]string, 0)
	for _, file := range all {
		if _, ok := consumed[file.ID]; ok {
			continue
		}
		residual = append(residual, strings.TrimSpace(file.StoragePath))
	}
	sort.Strings(residual)
	return residual
}

func memberPathsForGroup(all []database.InventoryFile, group directoryReductionGroup) []string {
	if len(group.Members) == 0 {
		return nil
	}
	memberIDs := make(map[uint]struct{}, len(group.Members))
	for _, id := range group.Members {
		memberIDs[id] = struct{}{}
	}
	paths := make([]string, 0, len(group.Members))
	for _, file := range all {
		if _, ok := memberIDs[file.ID]; ok {
			paths = append(paths, strings.TrimSpace(file.StoragePath))
		}
	}
	return paths
}

func inferReductionAlternatives(primary string) []string {
	switch strings.TrimSpace(primary) {
	case "movie_multi_version":
		return []string{pathTreeWorkGroupShapeMovieCollection, pathTreeWorkGroupShapeReview}
	case "episode_multi_version":
		return []string{pathTreeWorkGroupShapeSeries, pathTreeWorkGroupShapeReview}
	default:
		return []string{pathTreeWorkGroupShapeReview}
	}
}
