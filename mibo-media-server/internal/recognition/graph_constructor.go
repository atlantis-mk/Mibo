package recognition

import (
	"path"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

type GraphConstructInput = ManifestBuildInput

type GraphConstructOutput struct {
	ManifestScope             ManifestScope
	MediaGraphNodes           []database.MediaGraphNode
	MediaGraphEdges           []database.MediaGraphEdge
	MediaGraphClassifications []database.MediaGraphClassification
	Candidates                []database.RecognitionCandidate
	Evidence                  []database.RecognitionEvidence
}

func ConstructGraphFromInventory(input GraphConstructInput) GraphConstructOutput {
	units := BuildRecognitionWorkUnits(input)
	candidates := make([]database.RecognitionCandidate, 0)
	evidence := make([]database.RecognitionEvidence, 0)
	for _, unit := range units {
		evidence = append(evidence, CollectWorkUnitEvidence(unit)...)
		evidence = append(evidence, siblingConsistencyEvidence(unit)...)
		candidates = append(candidates, GenerateCandidatesForWorkUnit(unit)...)
	}
	graph := mediaGraphFromKernelCandidates(input, candidates)
	return GraphConstructOutput{
		ManifestScope:             graph.scope,
		MediaGraphNodes:           graphNodesFromMediaGraph(graph),
		MediaGraphEdges:           graphEdgesFromMediaGraph(graph),
		MediaGraphClassifications: graphClassificationsFromMediaGraph(graph),
		Candidates:                candidates,
		Evidence:                  evidence,
	}
}

func mediaGraphFromKernelCandidates(input GraphConstructInput, candidates []database.RecognitionCandidate) mediaGraph {
	scope := input.Scope
	if strings.TrimSpace(scope.ManifestKey) == "" {
		scope.ManifestKey = ManifestKey(Scope{StorageProvider: scope.StorageProvider, RootPath: scope.RootPath, ScopePath: scope.ScopePath}, scope.ClassifierVersion)
	}
	if strings.TrimSpace(scope.Fingerprint) == "" {
		scope.Fingerprint = inventoryManifestFingerprint(input.Files)
	}
	graph := mediaGraph{scope: scope, groupCount: make(map[string]int)}
	parentByDir := make(map[string]string)
	for _, candidate := range candidates {
		switch strings.TrimSpace(candidate.CandidateType) {
		case CandidateTypePlayableResource:
			plan := mediaGraphFilePlanFromKernelCandidate(input, candidate)
			graph.filePlans = append(graph.filePlans, plan)
			if strings.TrimSpace(plan.ResourceParentKey) != "" && strings.TrimSpace(plan.File.StoragePath) != "" {
				parentByDir[path.Dir(strings.TrimSpace(plan.File.StoragePath))] = plan.ResourceParentKey
			}
		case CandidateTypeWork, CandidateTypeEpisode:
			graph.appendKernelCandidateGroup(candidate)
		}
	}
	graph.appendReviewPackageGroupsForUnparentedResources()
	graph.attachments = append(graph.attachments, sidecarAttachmentsFromInput(input, parentByDir, scope.RootPath)...)
	graph.classifications = classifyMediaGraphGroups(graph)
	graph.scope.EvidenceJSON = mustJSON(map[string]any{"groups": graph.groups, "group_counts": graph.groupCount})
	return graph
}

func mediaGraphFilePlanFromKernelCandidate(input GraphConstructInput, candidate database.RecognitionCandidate) mediaGraphFilePlan {
	var file database.InventoryFile
	if candidate.PrimaryInventoryID != nil {
		file = inventoryFileByID(input.Files, *candidate.PrimaryInventoryID)
	}
	resourceKey := strings.TrimSpace(candidate.CandidateKey)
	parentKey := strings.TrimSpace(candidate.ParentCandidateKey)
	return mediaGraphFilePlan{
		File:              file,
		Signal:            input.FileSignals[file.ID],
		SidecarHints:      input.SidecarHints[file.ID],
		ContextEvidence:   input.ContextEvidence[file.ID],
		ResourceKey:       resourceKey,
		MovieKey:          movieKeyFromKernelCandidate(parentKey),
		SeriesKey:         seriesCandidateKeyFromEpisodeKey(parentKey),
		SeasonKey:         seasonCandidateKeyFromEpisodeKey(parentKey),
		EpisodeKeys:       episodeKeysFromKernelCandidate(parentKey),
		VariantKey:        strings.TrimSpace(candidate.VariantKey),
		EditionKey:        strings.TrimSpace(candidate.EditionKey),
		ResourceParentKey: parentKey,
		ResourceShape:     firstNonEmptyString(strings.TrimSpace(candidate.ResourceShape), ResourceKindSingleFile),
		ResourceRole:      firstNonEmptyString(strings.TrimSpace(candidate.CandidateRole), strings.TrimSpace(input.FileSignals[file.ID].Role), "source"),
		Confidence:        confidenceFromKernelCandidate(candidate),
		Accepted:          parentKey != "",
	}
}

func (g *mediaGraph) appendReviewPackageGroupsForUnparentedResources() {
	for _, plan := range g.filePlans {
		if strings.TrimSpace(plan.ResourceParentKey) != "" || strings.TrimSpace(plan.ResourceRole) != "source" || strings.TrimSpace(plan.ResourceKey) == "" {
			continue
		}
		g.groups = append(g.groups, mediaGraphGroup{Kind: mediaGroupKindMoviePackage, Key: groupNodeKey(mediaGroupKindMoviePackage, plan.ResourceKey), FileIDs: []uint{plan.File.ID}, Confidence: plan.Confidence, Payload: map[string]any{"resource_key": plan.ResourceKey}})
		g.groupCount[mediaGroupKindMoviePackage]++
	}
}

func (g *mediaGraph) appendKernelCandidateGroup(candidate database.RecognitionCandidate) {
	key := strings.TrimSpace(candidate.CandidateKey)
	if key == "" {
		return
	}
	kind := mediaGraphKindForKernelCandidate(candidate)
	if kind == "" {
		return
	}
	group := mediaGraphGroup{Kind: kind, Key: groupNodeKey(kind, key), ParentKey: kernelParentGroupNodeKey(candidate), Confidence: confidenceFromKernelCandidate(candidate), Payload: mediaGraphPayloadForKernelCandidate(candidate)}
	if candidate.PrimaryInventoryID != nil {
		group.FileIDs = []uint{*candidate.PrimaryInventoryID}
	}
	g.groups = append(g.groups, group)
	g.groupCount[kind]++
}

func kernelParentGroupNodeKey(candidate database.RecognitionCandidate) string {
	parentKey := strings.TrimSpace(candidate.ParentCandidateKey)
	if parentKey == "" {
		return ""
	}
	switch {
	case strings.TrimSpace(candidate.CandidateRole) == WorkKindSeason:
		return groupNodeKey(mediaGroupKindSeriesPackage, parentKey)
	case strings.TrimSpace(candidate.CandidateRole) == WorkKindEpisode || strings.TrimSpace(candidate.CandidateType) == CandidateTypeEpisode:
		return groupNodeKey(mediaGroupKindSeasonPackage, parentKey)
	default:
		return ""
	}
}

func mediaGraphKindForKernelCandidate(candidate database.RecognitionCandidate) string {
	switch strings.TrimSpace(candidate.CandidateRole) {
	case WorkKindMovie:
		return mediaGroupKindMoviePackage
	case WorkKindSeries:
		return mediaGroupKindSeriesPackage
	case WorkKindSeason:
		return mediaGroupKindSeasonPackage
	case WorkKindEpisode:
		return mediaGroupKindEpisodeRun
	}
	if strings.TrimSpace(candidate.CandidateType) == CandidateTypeEpisode {
		return mediaGroupKindEpisodeRun
	}
	return ""
}

func mediaGraphPayloadForKernelCandidate(candidate database.RecognitionCandidate) map[string]any {
	key := strings.TrimSpace(candidate.CandidateKey)
	switch strings.TrimSpace(candidate.CandidateRole) {
	case WorkKindMovie:
		return map[string]any{"movie_key": key}
	case WorkKindSeries:
		return map[string]any{"series_key": key}
	case WorkKindSeason:
		return map[string]any{"season_key": key}
	case WorkKindEpisode:
		return map[string]any{"episode_keys": []string{key}}
	}
	if strings.TrimSpace(candidate.CandidateType) == CandidateTypeEpisode {
		return map[string]any{"episode_keys": []string{key}}
	}
	return nil
}

func confidenceFromKernelCandidate(candidate database.RecognitionCandidate) float64 {
	if candidate.Confidence != nil && *candidate.Confidence > 0 {
		return *candidate.Confidence
	}
	switch strings.TrimSpace(candidate.CandidateType) {
	case CandidateTypePlayableResource:
		if strings.TrimSpace(candidate.ParentCandidateKey) == "" {
			return 0.6
		}
		return 0.78
	case CandidateTypeEpisode:
		return 0.82
	case CandidateTypeWork:
		if strings.TrimSpace(candidate.CandidateRole) == WorkKindMovie {
			return 0.75
		}
		return 0.82
	default:
		return 0.6
	}
}

func movieKeyFromKernelCandidate(parentKey string) string {
	if strings.HasPrefix(strings.TrimSpace(parentKey), CandidateTypeWork+":"+WorkKindMovie+":") {
		return strings.TrimSpace(parentKey)
	}
	return ""
}

func episodeKeysFromKernelCandidate(parentKey string) []string {
	if strings.HasPrefix(strings.TrimSpace(parentKey), CandidateTypeEpisode+":") {
		return []string{strings.TrimSpace(parentKey)}
	}
	return nil
}

func siblingConsistencyEvidence(unit RecognitionWorkUnit) []database.RecognitionEvidence {
	if len(unit.Files) < 2 {
		return nil
	}
	title := ""
	year := ""
	for _, file := range unit.Files {
		signal := unit.FileSignals[file.ID]
		if strings.TrimSpace(signal.TitleCandidate) == "" || signal.Year == nil {
			return nil
		}
		currentTitle := strings.TrimSpace(signal.TitleCandidate)
		currentYear := stringInt(*signal.Year)
		if title == "" {
			title = currentTitle
			year = currentYear
			continue
		}
		if currentTitle != title || currentYear != year {
			return nil
		}
	}
	items := make([]database.RecognitionEvidence, 0, len(unit.Files))
	for _, file := range unit.Files {
		fileID := file.ID
		items = append(items, database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: evidenceKindDirectoryContext, EvidenceSource: "work_unit", EvidenceKey: "sibling_consistency", EvidenceValue: title + ":" + year, Strength: "strong"})
	}
	return items
}
