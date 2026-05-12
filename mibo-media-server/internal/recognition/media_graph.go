package recognition

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

const (
	mediaGroupKindMoviePackage    = "movie_package"
	mediaGroupKindMovieCollection = "movie_collection"
	mediaGroupKindSeriesPackage   = "series_package"
	mediaGroupKindSeasonPackage   = "season_package"
	mediaGroupKindEpisodeRun      = "episode_run"
	mediaGroupKindAttachment      = "attachment"
)

type mediaGraph struct {
	scope           ManifestScope
	groups          []mediaGraphGroup
	classifications []mediaGraphClassification
	filePlans       []mediaGraphFilePlan
	attachments     []mediaGraphAttachment
	groupCount      map[string]int
}

type mediaGraphGroup struct {
	Kind       string         `json:"kind"`
	Key        string         `json:"key"`
	ParentKey  string         `json:"parent_key,omitempty"`
	FileIDs    []uint         `json:"file_ids,omitempty"`
	Confidence float64        `json:"confidence"`
	Payload    map[string]any `json:"payload,omitempty"`
}

type mediaGraphClassification struct {
	GroupKey     string
	GroupKind    string
	ClassifiedAs string
	ReviewState  string
	Confidence   float64
	Reason       string
	Alternatives []string
	Evidence     map[string]any
}

type mediaGraphFilePlan struct {
	File              database.InventoryFile
	Signal            database.InventoryFileSignal
	SidecarHints      []SidecarHint
	ContextEvidence   []ContextEvidence
	ResourceKey       string
	MovieKey          string
	SeriesKey         string
	SeasonKey         string
	EpisodeKeys       []string
	VariantKey        string
	EditionKey        string
	ResourceParentKey string
	ResourceShape     string
	ResourceRole      string
	GroupKind         string
	Confidence        float64
	Accepted          bool
}

type mediaGraphAttachment struct {
	File            database.InventoryFile
	ParentKey       string
	AttachmentKey   string
	Role            string
	AssociationKind string
	Confidence      float64
}

func buildMediaGraphFromInventory(input ManifestBuildInput) mediaGraph {
	scope := input.Scope
	if strings.TrimSpace(scope.ManifestKey) == "" {
		scope.ManifestKey = ManifestKey(Scope{StorageProvider: scope.StorageProvider, RootPath: scope.RootPath, ScopePath: scope.ScopePath}, scope.ClassifierVersion)
	}
	if strings.TrimSpace(scope.Fingerprint) == "" {
		scope.Fingerprint = inventoryManifestFingerprint(input.Files)
	}
	graph := mediaGraph{scope: scope, groupCount: make(map[string]int)}
	plans := make([]mediaGraphFilePlan, 0, len(input.Files))
	seriesGroups := make(map[string]*mediaGraphGroup)
	seasonGroups := make(map[string]*mediaGraphGroup)
	episodeRuns := make(map[string]*mediaGraphGroup)
	movieGroups := make(map[string]*mediaGraphGroup)

	for _, file := range input.Files {
		if !eligibleInventoryFile(file) {
			continue
		}
		resourceKey := PlayableResourceKey(ResourceInput{StorageProvider: file.StorageProvider, StoragePath: file.StoragePath, StableIdentityKey: file.StableIdentityKey})
		if resourceKey == "" {
			continue
		}
		plan := graphFilePlanFromSignals(file, input.FileSignals[file.ID], input.SidecarHints[file.ID], input.ContextEvidence[file.ID], resourceKey)
		plans = append(plans, plan)
		if plan.ResourceRole != "source" {
			continue
		}
		if len(plan.EpisodeKeys) > 0 {
			addMediaGroup(seriesGroups, groupNodeKey(mediaGroupKindSeriesPackage, plan.SeriesKey), mediaGraphGroup{Kind: mediaGroupKindSeriesPackage, Key: groupNodeKey(mediaGroupKindSeriesPackage, plan.SeriesKey), FileIDs: []uint{file.ID}, Confidence: plan.Confidence, Payload: map[string]any{"series_key": plan.SeriesKey}})
			addMediaGroup(seasonGroups, groupNodeKey(mediaGroupKindSeasonPackage, plan.SeasonKey), mediaGraphGroup{Kind: mediaGroupKindSeasonPackage, Key: groupNodeKey(mediaGroupKindSeasonPackage, plan.SeasonKey), ParentKey: groupNodeKey(mediaGroupKindSeriesPackage, plan.SeriesKey), FileIDs: []uint{file.ID}, Confidence: plan.Confidence, Payload: map[string]any{"series_key": plan.SeriesKey, "season_key": plan.SeasonKey}})
			runKey := firstNonEmptyString(plan.SeasonKey, plan.SeriesKey)
			addMediaGroup(episodeRuns, groupNodeKey(mediaGroupKindEpisodeRun, runKey), mediaGraphGroup{Kind: mediaGroupKindEpisodeRun, Key: groupNodeKey(mediaGroupKindEpisodeRun, runKey), ParentKey: groupNodeKey(mediaGroupKindSeasonPackage, plan.SeasonKey), FileIDs: []uint{file.ID}, Confidence: plan.Confidence, Payload: map[string]any{"episode_keys": append([]string(nil), plan.EpisodeKeys...)}})
			continue
		}
		if plan.MovieKey != "" {
			addMediaGroup(movieGroups, groupNodeKey(mediaGroupKindMoviePackage, plan.MovieKey), mediaGraphGroup{Kind: mediaGroupKindMoviePackage, Key: groupNodeKey(mediaGroupKindMoviePackage, plan.MovieKey), FileIDs: []uint{file.ID}, Confidence: plan.Confidence, Payload: map[string]any{"movie_key": plan.MovieKey}})
		}
	}

	graph.filePlans = plans
	graph.appendGroupsFromMap(seriesGroups)
	graph.appendGroupsFromMap(seasonGroups)
	graph.appendGroupsFromMap(episodeRuns)
	graph.appendGroupsFromMap(movieGroups)
	if len(movieGroups) > 1 {
		keys := make([]string, 0, len(movieGroups))
		fileIDs := make([]uint, 0)
		for key, group := range movieGroups {
			keys = append(keys, key)
			fileIDs = append(fileIDs, group.FileIDs...)
		}
		sort.Strings(keys)
		sort.Slice(fileIDs, func(i, j int) bool { return fileIDs[i] < fileIDs[j] })
		graph.groups = append(graph.groups, mediaGraphGroup{
			Kind:       mediaGroupKindMovieCollection,
			Key:        groupNodeKey(mediaGroupKindMovieCollection, strings.TrimSpace(scope.ScopePath)),
			FileIDs:    fileIDs,
			Confidence: 0.78,
			Payload:    map[string]any{"movie_keys": keys},
		})
		graph.groupCount[mediaGroupKindMovieCollection]++
	}
	graph.classifications = classifyMediaGraphGroups(graph)
	acceptedParents := acceptedParentKeysFromClassifications(graph.classifications)
	parentByDir := make(map[string]string)
	for idx := range graph.filePlans {
		plan := &graph.filePlans[idx]
		if plan.ResourceRole != "source" {
			continue
		}
		switch {
		case len(plan.EpisodeKeys) > 0 && acceptedParents[firstEpisodeKey(plan.EpisodeKeys)]:
			plan.GroupKind = mediaGroupKindSeriesPackage
			plan.ResourceParentKey = firstEpisodeKey(plan.EpisodeKeys)
			plan.Accepted = true
			parentByDir[path.Dir(strings.TrimSpace(plan.File.StoragePath))] = plan.ResourceParentKey
		case plan.MovieKey != "" && acceptedParents[plan.MovieKey]:
			plan.GroupKind = mediaGroupKindMoviePackage
			plan.ResourceParentKey = plan.MovieKey
			plan.Accepted = true
			parentByDir[path.Dir(strings.TrimSpace(plan.File.StoragePath))] = plan.ResourceParentKey
		}
	}
	for idx := range graph.filePlans {
		plan := &graph.filePlans[idx]
		if plan.ResourceRole == "source" {
			continue
		}
		parentKey := nearestParentCandidateKey(parentByDir, path.Dir(strings.TrimSpace(plan.File.StoragePath)), scope.RootPath)
		if parentKey == "" {
			parentKey = firstNonEmptyString(plan.MovieKey, firstEpisodeKey(plan.EpisodeKeys))
		}
		if parentKey == "" {
			continue
		}
		plan.GroupKind = mediaGroupKindAttachment
		plan.ResourceParentKey = parentKey
		plan.Accepted = true
		graph.groups = append(graph.groups, mediaGraphGroup{
			Kind:       mediaGroupKindAttachment,
			Key:        groupNodeKey(mediaGroupKindAttachment, plan.ResourceKey),
			ParentKey:  parentKey,
			FileIDs:    []uint{plan.File.ID},
			Confidence: plan.Confidence,
			Payload:    map[string]any{"role": plan.ResourceRole},
		})
		graph.groupCount[mediaGroupKindAttachment]++
	}
	graph.attachments = append(graph.attachments, sidecarAttachmentsFromInput(input, parentByDir, scope.RootPath)...)
	for _, attachment := range graph.attachments {
		graph.groups = append(graph.groups, mediaGraphGroup{
			Kind:       mediaGroupKindAttachment,
			Key:        groupNodeKey(mediaGroupKindAttachment, attachment.AttachmentKey),
			ParentKey:  attachment.ParentKey,
			FileIDs:    []uint{attachment.File.ID},
			Confidence: attachment.Confidence,
			Payload:    map[string]any{"role": attachment.Role, "association": attachment.AssociationKind, "content_class": attachment.File.ContentClass, "storage_path": attachment.File.StoragePath},
		})
		graph.groupCount[mediaGroupKindAttachment]++
	}
	graph.classifications = classifyMediaGraphGroups(graph)
	graph.scope.EvidenceJSON = mustJSON(map[string]any{"groups": graph.groups, "group_counts": graph.groupCount})
	return graph
}

func graphFilePlanFromSignals(file database.InventoryFile, signal database.InventoryFileSignal, sidecarHints []SidecarHint, contextEvidence []ContextEvidence, resourceKey string) mediaGraphFilePlan {
	contextParentKey := firstContextParentKey(contextEvidence)
	contextSeriesKey := firstContextSeriesKey(contextEvidence)
	contextSeasonKey := firstContextSeasonKey(contextEvidence)
	hasEpisodeContext := contextSeriesKey != "" || contextSeasonKey != "" || contextEpisodeParentKey(contextParentKey) != ""
	episodeKeys := episodeCandidateKeys(signal, sidecarHints, contextParentKey)
	seriesKey := ""
	seasonKey := ""
	if len(episodeKeys) > 0 {
		seriesKey = firstNonEmptyString(contextSeriesKey, seriesCandidateKeyFromEpisodeKey(episodeKeys[0]))
		seasonKey = firstNonEmptyString(contextSeasonKey, seasonCandidateKeyFromEpisodeKey(episodeKeys[0]))
	}
	movieKey := firstNonEmptyString(contextMovieParentKey(contextEvidence), movieCandidateKeyWithoutEpisodeContext(hasEpisodeContext, file, signal, sidecarHints))
	role := strings.TrimSpace(signal.Role)
	if role == "" {
		role = "main"
	}
	confidence := 0.6
	resourceShape := ResourceKindSingleFile
	resourceRole := "source"
	if len(episodeKeys) > 0 {
		confidence = 0.82
		if len(episodeKeys) > 1 {
			resourceShape = ResourceKindMultiEpisode
		}
	} else if movieKey != "" {
		confidence = 0.75
	}
	if role != "main" {
		confidence = 0.68
		resourceRole = role
	}
	return mediaGraphFilePlan{
		File:            file,
		Signal:          signal,
		SidecarHints:    sidecarHints,
		ContextEvidence: contextEvidence,
		ResourceKey:     resourceKey,
		MovieKey:        movieKey,
		SeriesKey:       seriesKey,
		SeasonKey:       seasonKey,
		EpisodeKeys:     episodeKeys,
		VariantKey:      firstNonEmptyString(firstContextVariantKey(contextEvidence), variantCandidateKey(file, signal)),
		EditionKey:      firstNonEmptyString(firstContextEditionKey(contextEvidence), editionCandidateKey(signal, sidecarHints)),
		ResourceShape:   resourceShape,
		ResourceRole:    resourceRole,
		Confidence:      confidence,
	}
}

func constructManifestOutput(input ManifestBuildInput) ManifestBuildOutput {
	graph := buildMediaGraphFromInventory(input)
	return constructManifestOutputFromGraph(graph, input)
}

func constructManifestOutputFromGraph(graph mediaGraph, input ManifestBuildInput) ManifestBuildOutput {
	output := ManifestBuildOutput{ManifestScope: graph.scope}
	for _, plan := range graph.filePlans {
		output.Evidence = append(output.Evidence, inventoryEvidence(plan.File, plan.ResourceKey)...)
		for _, resourceFile := range input.ResourceFiles[plan.File.ID] {
			output.Evidence = append(output.Evidence, database.RecognitionEvidence{CandidateID: nil, InventoryFileID: &plan.File.ID, EvidenceKind: evidenceKindLinkedResource, EvidenceSource: evidenceSourceResource, EvidenceKey: plan.ResourceKey, EvidenceValue: mustJSON(resourceFile), Strength: "strong"})
		}
		if signal, ok := input.FileSignals[plan.File.ID]; ok {
			output.Evidence = append(output.Evidence, signalEvidence(plan.File.ID, plan.ResourceKey, signal)...)
		}
		for _, sidecar := range input.SidecarsByFileID[plan.File.ID] {
			output.Evidence = append(output.Evidence, database.RecognitionEvidence{CandidateID: nil, InventoryFileID: &plan.File.ID, EvidenceKind: evidenceKindSidecar, EvidenceSource: evidenceSourceSidecar, EvidenceKey: plan.ResourceKey, EvidenceValue: strings.TrimSpace(sidecar.StoragePath), Strength: "medium", PayloadJSON: mustJSON(sidecar)})
		}
		for _, hint := range input.SidecarHints[plan.File.ID] {
			output.Evidence = append(output.Evidence, sidecarHintEvidence(plan.File.ID, plan.ResourceKey, hint)...)
		}
		for _, context := range input.ContextEvidence[plan.File.ID] {
			output.Evidence = append(output.Evidence, directoryContextEvidence(plan.File.ID, plan.ResourceKey, context)...)
		}
		if reason := strings.TrimSpace(input.ExcludedFileIDs[plan.File.ID]); reason != "" {
			output.Evidence = append(output.Evidence, database.RecognitionEvidence{CandidateID: nil, InventoryFileID: &plan.File.ID, EvidenceKind: evidenceKindScanExclusion, EvidenceSource: evidenceSourceExclusion, EvidenceKey: plan.ResourceKey, EvidenceValue: reason, Strength: "strong"})
		}
		if !plan.Accepted && plan.ResourceRole == "source" && strings.TrimSpace(input.ExcludedFileIDs[plan.File.ID]) == "" {
			continue
		}
		candidate := buildResourceCandidate(plan, input.ExcludedFileIDs[plan.File.ID])
		output.Candidates = append(output.Candidates, candidate)
		if plan.Accepted {
			output.Candidates = append(output.Candidates, groupedCandidatesForPlan(plan)...)
		}
	}
	return output
}

func sidecarAttachmentsFromInput(input ManifestBuildInput, parentByDir map[string]string, rootPath string) []mediaGraphAttachment {
	if len(input.SidecarsByFileID) == 0 {
		return nil
	}
	attachments := make([]mediaGraphAttachment, 0)
	seen := make(map[string]struct{})
	for sourceFileID, sidecars := range input.SidecarsByFileID {
		sourceFile := inventoryFileByID(input.Files, sourceFileID)
		parentKey := nearestParentCandidateKey(parentByDir, path.Dir(strings.TrimSpace(sourceFile.StoragePath)), rootPath)
		for _, sidecar := range sidecars {
			if strings.TrimSpace(sidecar.StoragePath) == "" {
				continue
			}
			attachmentKey := attachmentGraphKey(sidecar)
			if attachmentKey == "" {
				continue
			}
			if _, ok := seen[attachmentKey]; ok {
				continue
			}
			seen[attachmentKey] = struct{}{}
			if parentKey == "" {
				parentKey = nearestParentCandidateKey(parentByDir, path.Dir(strings.TrimSpace(sidecar.StoragePath)), rootPath)
			}
			if parentKey == "" {
				continue
			}
			attachments = append(attachments, mediaGraphAttachment{
				File:            sidecar,
				ParentKey:       parentKey,
				AttachmentKey:   attachmentKey,
				Role:            attachmentRoleForFile(sidecar),
				AssociationKind: "sidecar",
				Confidence:      0.82,
			})
		}
	}
	sort.Slice(attachments, func(i, j int) bool { return attachments[i].AttachmentKey < attachments[j].AttachmentKey })
	return attachments
}

func inventoryFileByID(files []database.InventoryFile, fileID uint) database.InventoryFile {
	for _, file := range files {
		if file.ID == fileID {
			return file
		}
	}
	return database.InventoryFile{}
}

func attachmentGraphKey(file database.InventoryFile) string {
	storagePath := strings.TrimSpace(file.StoragePath)
	if storagePath == "" {
		return ""
	}
	provider := strings.TrimSpace(file.StorageProvider)
	if provider == "" {
		provider = "local"
	}
	if stable := strings.TrimSpace(file.StableIdentityKey); stable != "" {
		return joinKey("attachment", provider, "stable", stable)
	}
	return joinKey("attachment", provider, "path", cleanPathKey(storagePath))
}

func attachmentRoleForFile(file database.InventoryFile) string {
	base := strings.ToLower(strings.TrimSuffix(path.Base(file.StoragePath), path.Ext(file.StoragePath)))
	switch {
	case strings.Contains(base, "poster") || strings.Contains(base, "cover"):
		return "poster"
	case strings.Contains(base, "fanart") || strings.Contains(base, "backdrop"):
		return "fanart"
	case strings.Contains(base, "trailer"):
		return "trailer"
	case strings.Contains(base, "sample"):
		return "sample"
	}
	switch strings.TrimSpace(file.ContentClass) {
	case "text":
		ext := strings.ToLower(path.Ext(file.StoragePath))
		if ext == ".srt" || ext == ".ass" || ext == ".ssa" || ext == ".vtt" {
			return "subtitle"
		}
		return "metadata"
	case "image":
		return "image"
	default:
		return "attachment"
	}
}

func classifyMediaGraphGroups(graph mediaGraph) []mediaGraphClassification {
	classifications := make([]mediaGraphClassification, 0, len(graph.groups))
	for _, group := range graph.groups {
		classification := classifyMediaGraphGroup(graph, group)
		classifications = append(classifications, classification)
	}
	sort.SliceStable(classifications, func(i, j int) bool {
		if classifications[i].GroupKey == classifications[j].GroupKey {
			return classifications[i].ClassifiedAs < classifications[j].ClassifiedAs
		}
		return classifications[i].GroupKey < classifications[j].GroupKey
	})
	return classifications
}

func classifyMediaGraphGroup(graph mediaGraph, group mediaGraphGroup) mediaGraphClassification {
	evidence := mergeGraphPayload(group.Payload, map[string]any{"file_ids": append([]uint(nil), group.FileIDs...)})
	classification := mediaGraphClassification{
		GroupKey:     strings.TrimSpace(group.Key),
		GroupKind:    strings.TrimSpace(group.Kind),
		ClassifiedAs: strings.TrimSpace(group.Kind),
		ReviewState:  database.ReviewStateNeedsReview,
		Confidence:   group.Confidence,
		Reason:       "media graph group needs stronger evidence before materialization",
		Evidence:     evidence,
	}
	switch strings.TrimSpace(group.Kind) {
	case mediaGroupKindSeriesPackage:
		return classifySeriesPackageGroup(graph, group, classification)
	case mediaGroupKindSeasonPackage:
		if group.Confidence >= 0.78 && strings.TrimSpace(group.ParentKey) != "" {
			classification.ReviewState = database.ReviewStateAccepted
			classification.Reason = "season package inherits an accepted series grouping"
		}
		return classification
	case mediaGroupKindEpisodeRun:
		return classifyEpisodeRunGroup(graph, group, classification)
	case mediaGroupKindMoviePackage:
		return classifyMoviePackageGroup(graph, group, classification)
	case mediaGroupKindMovieCollection:
		if len(group.FileIDs) > 1 {
			classification.ReviewState = database.ReviewStateNeedsReview
			classification.Confidence = minFloat(group.Confidence, 0.72)
			classification.Alternatives = []string{mediaGroupKindSeriesPackage, mediaGroupKindMoviePackage}
			classification.Reason = "multiple movie packages in one scope need review unless child movie packages are individually accepted"
		}
		return classification
	case mediaGroupKindAttachment:
		if strings.TrimSpace(group.ParentKey) != "" {
			classification.ReviewState = database.ReviewStateAccepted
			classification.Reason = "attachment group is anchored to an accepted parent resource"
		}
		return classification
	default:
		return classification
	}
}

func classifySeriesPackageGroup(graph mediaGraph, group mediaGraphGroup, classification mediaGraphClassification) mediaGraphClassification {
	strong := 0
	if mediaGroupHasSidecarType(graph, group, WorkKindSeries) || mediaGroupHasSeriesTitleSidecar(graph, group) {
		strong++
	}
	if mediaGroupHasExplicitEpisodeSignal(graph, group) {
		strong++
	}
	if mediaGroupHasSeasonDirectoryEvidence(graph, group) {
		strong++
	}
	if len(group.FileIDs) > 1 && mediaGroupHasSharedSeriesTitle(graph, group) {
		strong++
	}
	if strong > 0 {
		classification.ReviewState = database.ReviewStateAccepted
		classification.Confidence = maxFloat(group.Confidence, 0.82+float64(strong-1)*0.03)
		classification.Reason = "series package has strong episode or series structure evidence"
		return classification
	}
	classification.Alternatives = []string{mediaGroupKindMoviePackage}
	classification.Confidence = minFloat(group.Confidence, 0.6)
	return classification
}

func classifyEpisodeRunGroup(graph mediaGraph, group mediaGraphGroup, classification mediaGraphClassification) mediaGraphClassification {
	episodeNumbers := mediaGroupEpisodeNumbers(graph, group)
	if len(episodeNumbers) > 0 && isMostlyContinuousEpisodeRun(episodeNumbers) {
		classification.ReviewState = database.ReviewStateAccepted
		classification.Confidence = maxFloat(group.Confidence, 0.84)
		classification.Reason = "episode run has stable episode numbering"
		return classification
	}
	if len(group.FileIDs) > 1 {
		classification.ReviewState = database.ReviewStateAccepted
		classification.Confidence = maxFloat(group.Confidence, 0.8)
		classification.Reason = "episode run groups multiple files under a shared series/season"
		return classification
	}
	classification.Alternatives = []string{mediaGroupKindMoviePackage}
	return classification
}

func classifyMoviePackageGroup(graph mediaGraph, group mediaGraphGroup, classification mediaGraphClassification) mediaGraphClassification {
	if mediaGroupHasEpisodeSignal(graph, group) && !mediaGroupOnlyWeakLeadingEpisodeSignal(graph, group) {
		classification.ReviewState = database.ReviewStateNeedsReview
		classification.Confidence = minFloat(group.Confidence, 0.52)
		classification.Alternatives = []string{mediaGroupKindSeriesPackage}
		classification.Reason = "movie package conflicts with episode numbering evidence"
		return classification
	}
	if mediaGroupHasDirectoryAssignment(graph, group, "movie_multi_version", "single_work_identity", "movie_collection") {
		classification.ReviewState = database.ReviewStateAccepted
		classification.Confidence = maxFloat(group.Confidence, 0.82)
		classification.Reason = "movie package is anchored by directory grouping evidence"
		return classification
	}
	if mediaGroupHasSidecarType(graph, group, WorkKindMovie) {
		classification.ReviewState = database.ReviewStateAccepted
		classification.Confidence = maxFloat(group.Confidence, 0.88)
		classification.Reason = "movie package has movie sidecar evidence"
		return classification
	}
	if len(group.FileIDs) == 1 && mediaGroupHasMovieTitleEvidence(graph, group) {
		classification.ReviewState = database.ReviewStateAccepted
		classification.Confidence = maxFloat(group.Confidence, 0.78)
		classification.Reason = "single main video has stable movie title evidence and no episode structure"
		return classification
	}
	classification.Alternatives = []string{mediaGroupKindSeriesPackage, mediaGroupKindMovieCollection}
	classification.Confidence = minFloat(group.Confidence, 0.64)
	return classification
}

func acceptedParentKeysFromClassifications(classifications []mediaGraphClassification) map[string]bool {
	accepted := make(map[string]bool)
	for _, classification := range classifications {
		if strings.TrimSpace(classification.ReviewState) != database.ReviewStateAccepted {
			continue
		}
		if strings.TrimSpace(classification.ClassifiedAs) == mediaGroupKindMoviePackage {
			if key, ok := stringPayloadValue(classification.Evidence, "movie_key"); ok {
				accepted[key] = true
			}
		}
		if strings.TrimSpace(classification.ClassifiedAs) == mediaGroupKindEpisodeRun {
			for _, key := range stringSlicePayloadValue(classification.Evidence, "episode_keys") {
				accepted[key] = true
			}
		}
	}
	return accepted
}

func mediaGroupPlans(graph mediaGraph, group mediaGraphGroup) []mediaGraphFilePlan {
	if len(group.FileIDs) == 0 {
		return nil
	}
	ids := make(map[uint]struct{}, len(group.FileIDs))
	for _, fileID := range group.FileIDs {
		ids[fileID] = struct{}{}
	}
	plans := make([]mediaGraphFilePlan, 0, len(group.FileIDs))
	for _, plan := range graph.filePlans {
		if _, ok := ids[plan.File.ID]; ok {
			plans = append(plans, plan)
		}
	}
	return plans
}

func mediaGroupHasSidecarType(graph mediaGraph, group mediaGraphGroup, mediaType string) bool {
	for _, plan := range mediaGroupPlans(graph, group) {
		for _, hint := range plan.SidecarHints {
			if strings.EqualFold(strings.TrimSpace(hint.MediaType), strings.TrimSpace(mediaType)) {
				return true
			}
		}
	}
	return false
}

func mediaGroupHasSeriesTitleSidecar(graph mediaGraph, group mediaGraphGroup) bool {
	for _, plan := range mediaGroupPlans(graph, group) {
		for _, hint := range plan.SidecarHints {
			if strings.TrimSpace(hint.SeriesTitle) != "" {
				return true
			}
		}
	}
	return false
}

func mediaGroupHasExplicitEpisodeSignal(graph mediaGraph, group mediaGraphGroup) bool {
	for _, plan := range mediaGroupPlans(graph, group) {
		if plan.Signal.SeasonNumber != nil && plan.Signal.EpisodeNumber != nil && strings.TrimSpace(plan.Signal.EpisodeSource) != "leading_numeric" {
			return true
		}
		for _, hint := range plan.SidecarHints {
			if strings.TrimSpace(hint.SeriesTitle) != "" && hint.SeasonNumber != nil && hint.EpisodeNumber != nil {
				return true
			}
		}
	}
	return false
}

func mediaGroupHasEpisodeSignal(graph mediaGraph, group mediaGraphGroup) bool {
	for _, plan := range mediaGroupPlans(graph, group) {
		if plan.Signal.EpisodeNumber != nil || len(plan.EpisodeKeys) > 0 {
			return true
		}
		for _, hint := range plan.SidecarHints {
			if hint.EpisodeNumber != nil {
				return true
			}
		}
	}
	return false
}

func mediaGroupOnlyWeakLeadingEpisodeSignal(graph mediaGraph, group mediaGraphGroup) bool {
	seen := false
	for _, plan := range mediaGroupPlans(graph, group) {
		if plan.Signal.EpisodeNumber == nil {
			continue
		}
		seen = true
		if strings.TrimSpace(plan.Signal.EpisodeSource) != "leading_numeric" || plan.Signal.SeasonNumber != nil {
			return false
		}
	}
	return seen
}

func mediaGroupHasSeasonDirectoryEvidence(graph mediaGraph, group mediaGraphGroup) bool {
	for _, plan := range mediaGroupPlans(graph, group) {
		for _, evidence := range plan.ContextEvidence {
			switch strings.TrimSpace(evidence.Assignment) {
			case "season_identity", "episode_identity", "single_episode_identity", "episode_multi_version":
				return true
			}
			if value, ok := evidence.Payload["season_number"]; ok && strings.TrimSpace(sprintAny(value)) != "" {
				return true
			}
		}
	}
	return false
}

func mediaGroupHasDirectoryAssignment(graph mediaGraph, group mediaGraphGroup, assignments ...string) bool {
	allowed := make(map[string]struct{}, len(assignments))
	for _, assignment := range assignments {
		allowed[strings.TrimSpace(assignment)] = struct{}{}
	}
	for _, plan := range mediaGroupPlans(graph, group) {
		for _, evidence := range plan.ContextEvidence {
			if _, ok := allowed[strings.TrimSpace(evidence.Assignment)]; ok {
				return true
			}
		}
	}
	return false
}

func mediaGroupHasSharedSeriesTitle(graph mediaGraph, group mediaGraphGroup) bool {
	titles := make(map[string]struct{})
	for _, plan := range mediaGroupPlans(graph, group) {
		if title := strings.TrimSpace(plan.Signal.TitleCandidate); title != "" {
			titles[strings.ToLower(title)] = struct{}{}
		}
		for _, hint := range plan.SidecarHints {
			if title := strings.TrimSpace(firstNonEmptyString(hint.SeriesTitle, hint.Title)); title != "" {
				titles[strings.ToLower(title)] = struct{}{}
			}
		}
	}
	return len(titles) == 1
}

func mediaGroupEpisodeNumbers(graph mediaGraph, group mediaGraphGroup) []int {
	numbers := make([]int, 0, len(group.FileIDs))
	seen := make(map[int]struct{})
	for _, plan := range mediaGroupPlans(graph, group) {
		for _, number := range inventorySignalEpisodeNumbers(plan.Signal) {
			if number <= 0 {
				continue
			}
			if _, ok := seen[number]; ok {
				continue
			}
			seen[number] = struct{}{}
			numbers = append(numbers, number)
		}
		if plan.Signal.EpisodeNumber != nil && *plan.Signal.EpisodeNumber > 0 {
			if _, ok := seen[*plan.Signal.EpisodeNumber]; !ok {
				seen[*plan.Signal.EpisodeNumber] = struct{}{}
				numbers = append(numbers, *plan.Signal.EpisodeNumber)
			}
		}
		for _, hint := range plan.SidecarHints {
			if hint.EpisodeNumber == nil || *hint.EpisodeNumber <= 0 {
				continue
			}
			if _, ok := seen[*hint.EpisodeNumber]; ok {
				continue
			}
			seen[*hint.EpisodeNumber] = struct{}{}
			numbers = append(numbers, *hint.EpisodeNumber)
		}
	}
	sort.Ints(numbers)
	return numbers
}

func isMostlyContinuousEpisodeRun(numbers []int) bool {
	if len(numbers) == 0 {
		return false
	}
	if len(numbers) == 1 {
		return true
	}
	for idx := 1; idx < len(numbers); idx++ {
		if numbers[idx] != numbers[idx-1]+1 {
			return false
		}
	}
	return true
}

func mediaGroupHasMovieTitleEvidence(graph mediaGraph, group mediaGraphGroup) bool {
	for _, plan := range mediaGroupPlans(graph, group) {
		if strings.TrimSpace(plan.MovieKey) == "" {
			continue
		}
		if strings.TrimSpace(plan.Signal.TitleCandidate) != "" && movieTitleSignalAllowed(plan.Signal) {
			return true
		}
		for _, hint := range plan.SidecarHints {
			if strings.EqualFold(strings.TrimSpace(hint.MediaType), WorkKindMovie) && strings.TrimSpace(hint.Title) != "" {
				return true
			}
		}
	}
	return false
}

func stringPayloadValue(payload map[string]any, key string) (string, bool) {
	if len(payload) == 0 {
		return "", false
	}
	value, ok := payload[key]
	if !ok {
		return "", false
	}
	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		return trimmed, trimmed != ""
	default:
		trimmed := strings.TrimSpace(sprintAny(value))
		return trimmed, trimmed != ""
	}
}

func stringSlicePayloadValue(payload map[string]any, key string) []string {
	if len(payload) == 0 {
		return nil
	}
	value, ok := payload[key]
	if !ok {
		return nil
	}
	return stringSliceFromAny(value)
}

func stringSliceFromAny(value any) []string {
	switch typed := value.(type) {
	case []string:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if trimmed := strings.TrimSpace(sprintAny(item)); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	default:
		if trimmed := strings.TrimSpace(sprintAny(value)); trimmed != "" {
			return []string{trimmed}
		}
	}
	return nil
}

func appendUniqueStrings(existing []string, values ...string) []string {
	seen := make(map[string]struct{}, len(existing)+len(values))
	result := make([]string, 0, len(existing)+len(values))
	for _, value := range existing {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}

func maxFloat(left float64, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func minFloat(left float64, right float64) float64 {
	if left < right {
		return left
	}
	return right
}

func sprintAny(value any) string {
	return fmt.Sprint(value)
}

func buildResourceCandidate(plan mediaGraphFilePlan, excludedReason string) database.RecognitionCandidate {
	confidence := plan.Confidence
	if confidence <= 0 {
		confidence = 0.6
	}
	evidence := map[string]any{
		"storage_path":     strings.TrimSpace(plan.File.StoragePath),
		"storage_provider": strings.TrimSpace(plan.File.StorageProvider),
		"stable_identity":  strings.TrimSpace(plan.File.StableIdentityKey),
		"hashes_json":      strings.TrimSpace(plan.File.HashesJSON),
		"group_kind":       plan.GroupKind,
	}
	if plan.ResourceShape == ResourceKindMultiEpisode && len(plan.EpisodeKeys) > 1 {
		evidence["episode_keys"] = append([]string(nil), plan.EpisodeKeys...)
	}
	role := plan.ResourceRole
	if role == "" {
		role = "source"
	}
	candidate := database.RecognitionCandidate{
		CandidateKey:       plan.ResourceKey,
		CandidateType:      CandidateTypePlayableResource,
		CandidateRole:      role,
		ParentCandidateKey: plan.ResourceParentKey,
		PrimaryInventoryID: uintPtr(plan.File.ID),
		CanonicalKey:       plan.ResourceParentKey,
		VariantKey:         plan.VariantKey,
		EditionKey:         plan.EditionKey,
		ResourceShape:      firstNonEmptyString(plan.ResourceShape, ResourceKindSingleFile),
		ReviewState:        database.ReviewStatePending,
		Confidence:         &confidence,
		AffectedFilesJSON:  mustJSON([]string{strings.TrimSpace(plan.File.StoragePath)}),
		EvidenceJSON:       mustJSON(evidence),
	}
	if strings.TrimSpace(excludedReason) != "" {
		candidate.ReviewState = database.ReviewStateRejected
		candidate.CandidateRole = "excluded"
	}
	return candidate
}

func groupedCandidatesForPlan(plan mediaGraphFilePlan) []database.RecognitionCandidate {
	candidates := make([]database.RecognitionCandidate, 0, 8)
	primaryFileID := uintPtr(plan.File.ID)
	if plan.MovieKey != "" && len(plan.EpisodeKeys) == 0 {
		confidence := 0.75
		candidates = append(candidates, database.RecognitionCandidate{CandidateKey: plan.MovieKey, CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, PrimaryInventoryID: primaryFileID, CanonicalKey: plan.MovieKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, EvidenceJSON: candidateTitleEvidence(plan.Signal.TitleCandidate, plan.Signal.Year, plan.SidecarHints), AffectedFilesJSON: mustJSON([]string{plan.File.StoragePath})})
		if plan.VariantKey != "" {
			candidates = append(candidates, database.RecognitionCandidate{CandidateKey: joinKey(plan.VariantKey, plan.ResourceKey), CandidateType: CandidateTypeVariant, ParentCandidateKey: plan.MovieKey, PrimaryInventoryID: primaryFileID, CanonicalKey: plan.MovieKey, VariantKey: plan.VariantKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, AffectedFilesJSON: mustJSON([]string{plan.File.StoragePath})})
		}
		if plan.EditionKey != "" {
			candidates = append(candidates, database.RecognitionCandidate{CandidateKey: joinKey(plan.EditionKey, plan.ResourceKey), CandidateType: CandidateTypeEdition, ParentCandidateKey: plan.MovieKey, PrimaryInventoryID: primaryFileID, CanonicalKey: plan.MovieKey, EditionKey: plan.EditionKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, AffectedFilesJSON: mustJSON([]string{plan.File.StoragePath})})
		}
	}
	if len(plan.EpisodeKeys) > 0 {
		confidence := 0.82
		if plan.SeriesKey != "" {
			candidates = append(candidates, database.RecognitionCandidate{CandidateKey: plan.SeriesKey, CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeries, PrimaryInventoryID: primaryFileID, CanonicalKey: plan.SeriesKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, EvidenceJSON: candidateSeriesEvidence(plan.Signal, plan.SidecarHints), AffectedFilesJSON: mustJSON([]string{plan.File.StoragePath})})
		}
		if plan.SeasonKey != "" {
			candidates = append(candidates, database.RecognitionCandidate{CandidateKey: plan.SeasonKey, CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeason, ParentCandidateKey: plan.SeriesKey, PrimaryInventoryID: primaryFileID, CanonicalKey: plan.SeasonKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, EvidenceJSON: candidateSeriesEvidence(plan.Signal, plan.SidecarHints), AffectedFilesJSON: mustJSON([]string{plan.File.StoragePath})})
		}
		for _, episodeKey := range plan.EpisodeKeys {
			candidates = append(candidates, database.RecognitionCandidate{CandidateKey: episodeKey, CandidateType: CandidateTypeEpisode, CandidateRole: WorkKindEpisode, ParentCandidateKey: firstNonEmptyString(plan.SeasonKey, seasonCandidateKeyFromEpisodeKey(episodeKey)), PrimaryInventoryID: primaryFileID, CanonicalKey: episodeKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, EvidenceJSON: candidateTitleEvidence(plan.Signal.TitleCandidate, plan.Signal.Year, plan.SidecarHints), AffectedFilesJSON: mustJSON([]string{plan.File.StoragePath})})
		}
		if plan.VariantKey != "" {
			candidates = append(candidates, database.RecognitionCandidate{CandidateKey: joinKey(plan.VariantKey, plan.ResourceKey), CandidateType: CandidateTypeVariant, ParentCandidateKey: plan.EpisodeKeys[0], PrimaryInventoryID: primaryFileID, CanonicalKey: plan.EpisodeKeys[0], VariantKey: plan.VariantKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, AffectedFilesJSON: mustJSON([]string{plan.File.StoragePath})})
		}
		if plan.EditionKey != "" {
			candidates = append(candidates, database.RecognitionCandidate{CandidateKey: joinKey(plan.EditionKey, plan.ResourceKey), CandidateType: CandidateTypeEdition, ParentCandidateKey: plan.EpisodeKeys[0], PrimaryInventoryID: primaryFileID, CanonicalKey: plan.EpisodeKeys[0], EditionKey: plan.EditionKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, AffectedFilesJSON: mustJSON([]string{plan.File.StoragePath})})
		}
	}
	if plan.ResourceRole != "" && plan.ResourceRole != "source" && plan.ResourceParentKey != "" {
		confidence := 0.8
		candidates = append(candidates, database.RecognitionCandidate{CandidateKey: SupplementalKey(plan.ResourceParentKey, plan.ResourceRole, plan.ResourceKey), CandidateType: CandidateTypeSupplemental, CandidateRole: plan.ResourceRole, ParentCandidateKey: plan.ResourceParentKey, PrimaryInventoryID: primaryFileID, ReviewState: database.ReviewStatePending, Confidence: &confidence, AffectedFilesJSON: mustJSON([]string{plan.File.StoragePath})})
	}
	if hashKey := duplicateBinaryCandidateKey(plan.File); hashKey != "" {
		confidence := 0.98
		candidates = append(candidates, database.RecognitionCandidate{CandidateKey: hashKey, CandidateType: CandidateTypeDuplicateBinary, CandidateRole: "same_binary", PrimaryInventoryID: primaryFileID, ReviewState: database.ReviewStatePending, Confidence: &confidence, AffectedFilesJSON: mustJSON([]string{plan.File.StoragePath})})
	}
	return candidates
}

func addMediaGroup(groups map[string]*mediaGraphGroup, key string, group mediaGraphGroup) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	if existing, ok := groups[key]; ok {
		existing.FileIDs = appendUniqueFileIDs(existing.FileIDs, group.FileIDs...)
		if len(group.Payload) > 0 {
			if existing.Payload == nil {
				existing.Payload = map[string]any{}
			}
			for payloadKey, payloadValue := range group.Payload {
				if payloadKey == "episode_keys" {
					existing.Payload[payloadKey] = appendUniqueStrings(stringSlicePayloadValue(existing.Payload, payloadKey), stringSliceFromAny(payloadValue)...)
					continue
				}
				existing.Payload[payloadKey] = payloadValue
			}
		}
		if group.Confidence > existing.Confidence {
			existing.Confidence = group.Confidence
		}
		return
	}
	group.Key = key
	group.FileIDs = appendUniqueFileIDs(nil, group.FileIDs...)
	groups[key] = &group
}

func (g *mediaGraph) appendGroupsFromMap(groups map[string]*mediaGraphGroup) {
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		group := groups[key]
		g.groups = append(g.groups, *group)
		g.groupCount[group.Kind]++
	}
}

func appendUniqueFileIDs(existing []uint, values ...uint) []uint {
	seen := make(map[uint]struct{}, len(existing)+len(values))
	result := make([]uint, 0, len(existing)+len(values))
	for _, value := range existing {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func nearestParentCandidateKey(parentByDir map[string]string, dirPath string, rootPath string) string {
	current := strings.TrimSpace(dirPath)
	root := strings.TrimSpace(rootPath)
	for current != "" && current != "." {
		if parentKey := strings.TrimSpace(parentByDir[current]); parentKey != "" {
			return parentKey
		}
		if current == root || current == "/" {
			break
		}
		next := path.Dir(current)
		if next == current {
			break
		}
		current = next
	}
	return ""
}

func firstEpisodeKey(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	return strings.TrimSpace(keys[0])
}

func graphNodesFromMediaGraph(graph mediaGraph) []database.MediaGraphNode {
	nodes := make([]database.MediaGraphNode, 0, len(graph.groups)+len(graph.filePlans)+len(graph.attachments))
	for _, group := range graph.groups {
		payload := mergeGraphPayload(group.Payload, map[string]any{"file_ids": append([]uint(nil), group.FileIDs...)})
		nodes = append(nodes, database.MediaGraphNode{
			NodeKey:       strings.TrimSpace(group.Key),
			NodeKind:      strings.TrimSpace(group.Kind),
			ParentNodeKey: strings.TrimSpace(group.ParentKey),
			Confidence:    graphFloatPtr(group.Confidence),
			PayloadJSON:   mustJSON(payload),
		})
	}
	for _, plan := range graph.filePlans {
		payload := map[string]any{
			"storage_path":     strings.TrimSpace(plan.File.StoragePath),
			"storage_provider": strings.TrimSpace(plan.File.StorageProvider),
			"resource_key":     strings.TrimSpace(plan.ResourceKey),
			"group_kind":       strings.TrimSpace(plan.GroupKind),
			"resource_role":    strings.TrimSpace(plan.ResourceRole),
			"resource_shape":   strings.TrimSpace(plan.ResourceShape),
			"episode_keys":     append([]string(nil), plan.EpisodeKeys...),
		}
		fileID := plan.File.ID
		nodes = append(nodes, database.MediaGraphNode{
			NodeKey:         strings.TrimSpace(plan.ResourceKey),
			NodeKind:        "inventory_file",
			ParentNodeKey:   strings.TrimSpace(plan.ResourceParentKey),
			InventoryFileID: &fileID,
			CandidateKey:    strings.TrimSpace(plan.ResourceParentKey),
			Confidence:      graphFloatPtr(plan.Confidence),
			PayloadJSON:     mustJSON(payload),
		})
	}
	for _, attachment := range graph.attachments {
		var fileID *uint
		if attachment.File.ID != 0 {
			fileID = uintPtr(attachment.File.ID)
		}
		payload := map[string]any{
			"storage_path":     strings.TrimSpace(attachment.File.StoragePath),
			"storage_provider": strings.TrimSpace(attachment.File.StorageProvider),
			"content_class":    strings.TrimSpace(attachment.File.ContentClass),
			"role":             strings.TrimSpace(attachment.Role),
			"association":      strings.TrimSpace(attachment.AssociationKind),
		}
		nodes = append(nodes, database.MediaGraphNode{
			NodeKey:         strings.TrimSpace(attachment.AttachmentKey),
			NodeKind:        mediaGroupKindAttachment,
			ParentNodeKey:   strings.TrimSpace(attachment.ParentKey),
			InventoryFileID: fileID,
			CandidateKey:    strings.TrimSpace(attachment.ParentKey),
			Confidence:      graphFloatPtr(attachment.Confidence),
			PayloadJSON:     mustJSON(payload),
		})
	}
	return nodes
}

func graphEdgesFromMediaGraph(graph mediaGraph) []database.MediaGraphEdge {
	edges := make([]database.MediaGraphEdge, 0, len(graph.groups)*2+len(graph.attachments))
	for _, group := range graph.groups {
		for _, fileID := range group.FileIDs {
			edges = append(edges, database.MediaGraphEdge{
				FromNodeKey: strings.TrimSpace(group.Key),
				ToNodeKey:   inventoryFileGraphNodeKey(fileID),
				EdgeKind:    "contains_file",
				Confidence:  graphFloatPtr(group.Confidence),
			})
		}
		if strings.TrimSpace(group.ParentKey) != "" {
			edges = append(edges, database.MediaGraphEdge{
				FromNodeKey: strings.TrimSpace(group.ParentKey),
				ToNodeKey:   strings.TrimSpace(group.Key),
				EdgeKind:    "contains_group",
				Confidence:  graphFloatPtr(group.Confidence),
			})
		}
	}
	for _, plan := range graph.filePlans {
		if plan.File.ID == 0 || strings.TrimSpace(plan.ResourceParentKey) == "" {
			continue
		}
		edges = append(edges, database.MediaGraphEdge{
			FromNodeKey: strings.TrimSpace(plan.ResourceParentKey),
			ToNodeKey:   inventoryFileGraphNodeKey(plan.File.ID),
			EdgeKind:    "materializes_file",
			Confidence:  graphFloatPtr(plan.Confidence),
		})
	}
	for _, attachment := range graph.attachments {
		if strings.TrimSpace(attachment.ParentKey) == "" || strings.TrimSpace(attachment.AttachmentKey) == "" {
			continue
		}
		edges = append(edges, database.MediaGraphEdge{
			FromNodeKey: strings.TrimSpace(attachment.ParentKey),
			ToNodeKey:   strings.TrimSpace(attachment.AttachmentKey),
			EdgeKind:    "has_attachment",
			Confidence:  graphFloatPtr(attachment.Confidence),
			PayloadJSON: mustJSON(map[string]any{"role": attachment.Role, "association": attachment.AssociationKind}),
		})
	}
	return edges
}

func graphClassificationsFromMediaGraph(graph mediaGraph) []database.MediaGraphClassification {
	classifications := make([]database.MediaGraphClassification, 0, len(graph.classifications))
	for _, classification := range graph.classifications {
		confidence := classification.Confidence
		classifications = append(classifications, database.MediaGraphClassification{
			GroupNodeKey:     strings.TrimSpace(classification.GroupKey),
			GroupKind:        strings.TrimSpace(classification.GroupKind),
			ClassifiedAs:     strings.TrimSpace(classification.ClassifiedAs),
			ReviewState:      strings.TrimSpace(classification.ReviewState),
			Confidence:       graphFloatPtr(confidence),
			Reason:           strings.TrimSpace(classification.Reason),
			EvidenceJSON:     mustJSON(classification.Evidence),
			AlternativesJSON: mustJSON(classification.Alternatives),
		})
	}
	return classifications
}

func inventoryFileGraphNodeKey(fileID uint) string {
	if fileID == 0 {
		return ""
	}
	return "inventory_file:" + stringUint(fileID)
}

func mergeGraphPayload(base map[string]any, extras map[string]any) map[string]any {
	if len(base) == 0 && len(extras) == 0 {
		return nil
	}
	merged := make(map[string]any, len(base)+len(extras))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extras {
		merged[key] = value
	}
	return merged
}

func graphFloatPtr(value float64) *float64 {
	return &value
}

func groupNodeKey(kind string, key string) string {
	if strings.TrimSpace(kind) == "" || strings.TrimSpace(key) == "" {
		return strings.TrimSpace(key)
	}
	return "group:" + strings.TrimSpace(kind) + ":" + strings.TrimSpace(key)
}
