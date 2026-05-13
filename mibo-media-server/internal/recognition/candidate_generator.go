package recognition

import (
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

func GenerateCandidatesForWorkUnit(unit RecognitionWorkUnit) []database.RecognitionCandidate {
	candidates := make([]database.RecognitionCandidate, 0, len(unit.Files)*5)
	seen := make(map[string]struct{})
	add := func(candidate database.RecognitionCandidate) {
		key := strings.TrimSpace(candidate.CandidateKey)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidate.CandidateKey = key
		if candidate.CanonicalKey == "" {
			candidate.CanonicalKey = key
		}
		candidates = append(candidates, candidate)
	}

	for _, file := range unit.Files {
		fileID := file.ID
		signal := unit.FileSignals[file.ID]
		sidecarHints := unit.SidecarHints[file.ID]
		contextEvidence := unit.ContextEvidence[file.ID]
		excludedReason := strings.TrimSpace(unit.ExcludedFileIDs[file.ID])
		parentKey := ""
		episodeKeys := episodeCandidateKeys(signal, sidecarHints, firstContextParentKey(contextEvidence))
		if excludedReason == "" && unit.FolderShape != FolderShapeExtra && len(episodeKeys) > 0 {
			episodeKey := episodeKeys[0]
			seriesKey := firstNonEmptyString(firstContextSeriesKey(contextEvidence), seriesCandidateKeyFromEpisodeKey(episodeKey))
			seasonKey := firstNonEmptyString(firstContextSeasonKey(contextEvidence), seasonCandidateKeyFromEpisodeKey(episodeKey))
			add(database.RecognitionCandidate{CandidateKey: seriesKey, CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeries, PrimaryInventoryID: &fileID, EvidenceJSON: candidateSeriesEvidence(signal, sidecarHints)})
			add(database.RecognitionCandidate{CandidateKey: seasonKey, CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeason, ParentCandidateKey: seriesKey, PrimaryInventoryID: &fileID, EvidenceJSON: candidateSeriesEvidence(signal, sidecarHints)})
			for _, episodeKey := range episodeKeys {
				add(database.RecognitionCandidate{CandidateKey: episodeKey, CandidateType: CandidateTypeEpisode, CandidateRole: WorkKindEpisode, ParentCandidateKey: seasonKey, PrimaryInventoryID: &fileID, EvidenceJSON: candidateTitleEvidence(signal.TitleCandidate, signal.Year, sidecarHints)})
			}
			parentKey = firstEpisodeKey(episodeKeys)
			addVersionTraitCandidates(&add, file, signal, sidecarHints, contextEvidence, parentKey, fileID)
		} else if excludedReason == "" && unit.FolderShape != FolderShapeExtra && !nonMainResourceRole(signal) {
			parentKey = firstNonEmptyString(contextMovieParentKey(contextEvidence), kernelMovieWorkCandidateKey(signal, sidecarHints))
			add(database.RecognitionCandidate{CandidateKey: parentKey, CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, PrimaryInventoryID: &fileID, EvidenceJSON: candidateTitleEvidence(signal.TitleCandidate, signal.Year, sidecarHints)})
			addVersionTraitCandidates(&add, file, signal, sidecarHints, contextEvidence, parentKey, fileID)
		} else if nonMainResourceRole(signal) {
			parentKey = firstParentCandidateKey(candidates)
		}
		resourceEvidence := map[string]any{"storage_path": file.StoragePath, "role": signal.Role}
		if len(episodeKeys) > 1 {
			resourceEvidence["episode_keys"] = episodeKeys
		}
		if excludedReason != "" {
			resourceEvidence["excluded_reason"] = excludedReason
		}
		resourceRole := strings.TrimSpace(signal.Role)
		if excludedReason != "" {
			resourceRole = "excluded"
		}
		add(database.RecognitionCandidate{CandidateKey: PlayableResourceKey(ResourceInput{StorageProvider: file.StorageProvider, StoragePath: file.StoragePath, StableIdentityKey: file.StableIdentityKey}), CandidateType: CandidateTypePlayableResource, CandidateRole: resourceRole, ParentCandidateKey: parentKey, CanonicalKey: parentKey, PrimaryInventoryID: &fileID, ResourceShape: resourceShapeForSignal(signal), EvidenceJSON: mustJSON(resourceEvidence)})
		if nonMainResourceRole(signal) && parentKey != "" {
			add(database.RecognitionCandidate{CandidateKey: SupplementalKey(parentKey, signal.Role, PlayableResourceKey(ResourceInput{StorageProvider: file.StorageProvider, StoragePath: file.StoragePath, StableIdentityKey: file.StableIdentityKey})), CandidateType: CandidateTypeSupplemental, CandidateRole: strings.TrimSpace(signal.Role), ParentCandidateKey: parentKey, PrimaryInventoryID: &fileID})
		}
	}
	return candidates
}

func addVersionTraitCandidates(add *func(database.RecognitionCandidate), file database.InventoryFile, signal database.InventoryFileSignal, sidecarHints []SidecarHint, contextEvidence []ContextEvidence, parentKey string, fileID uint) {
	if strings.TrimSpace(parentKey) == "" {
		return
	}
	resourceKey := PlayableResourceKey(ResourceInput{StorageProvider: file.StorageProvider, StoragePath: file.StoragePath, StableIdentityKey: file.StableIdentityKey})
	confidence := 0.75
	if variantKey := firstNonEmptyString(firstContextVariantKey(contextEvidence), variantCandidateKey(file, signal)); variantKey != "" {
		(*add)(database.RecognitionCandidate{CandidateKey: joinKey(variantKey, resourceKey), CandidateType: CandidateTypeVariant, ParentCandidateKey: parentKey, PrimaryInventoryID: &fileID, CanonicalKey: parentKey, VariantKey: variantKey, Confidence: &confidence})
	}
	if editionKey := firstNonEmptyString(firstContextEditionKey(contextEvidence), editionCandidateKey(signal, sidecarHints)); editionKey != "" {
		(*add)(database.RecognitionCandidate{CandidateKey: joinKey(editionKey, resourceKey), CandidateType: CandidateTypeEdition, ParentCandidateKey: parentKey, PrimaryInventoryID: &fileID, CanonicalKey: parentKey, EditionKey: editionKey, Confidence: &confidence})
	}
}

func nonMainResourceRole(signal database.InventoryFileSignal) bool {
	role := strings.TrimSpace(signal.Role)
	return role != "" && role != "main"
}

func kernelMovieWorkCandidateKey(signal database.InventoryFileSignal, sidecarHints []SidecarHint) string {
	for _, hint := range sidecarHints {
		if strings.EqualFold(strings.TrimSpace(hint.MediaType), WorkKindMovie) && strings.TrimSpace(hint.Title) != "" {
			return MovieWorkKey(MovieWorkInput{Title: hint.Title, Year: hint.Year})
		}
	}
	if strings.TrimSpace(signal.TitleCandidate) == "" || !movieTitleSignalAllowed(signal) {
		return ""
	}
	return MovieWorkKey(MovieWorkInput{Title: signal.TitleCandidate, Year: signal.Year})
}

func candidateEpisodeKeysForSignal(signal database.InventoryFileSignal) []string {
	return candidateEpisodeKeysForSignalAndSidecars(signal, nil)
}

func candidateEpisodeKeysForSignalAndSidecars(signal database.InventoryFileSignal, sidecarHints []SidecarHint) []string {
	if keys := episodeCandidateKeys(signal, sidecarHints, ""); len(keys) > 0 {
		return keys
	}
	if strings.TrimSpace(signal.TitleCandidate) == "" || signal.SeasonNumber == nil {
		return nil
	}
	numbers := inventorySignalEpisodeNumbers(signal)
	if len(numbers) == 0 && signal.EpisodeNumber != nil {
		numbers = []int{*signal.EpisodeNumber}
	}
	keys := make([]string, 0, len(numbers))
	seen := make(map[string]struct{}, len(numbers))
	for _, episodeNumber := range numbers {
		key := EpisodeKey(EpisodeInput{SeriesTitle: signal.TitleCandidate, SeasonNumber: *signal.SeasonNumber, EpisodeNumber: episodeNumber})
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys
}

func firstSidecarSeriesTitle(hints []SidecarHint) string {
	for _, hint := range hints {
		if strings.TrimSpace(hint.SeriesTitle) != "" {
			return strings.TrimSpace(hint.SeriesTitle)
		}
	}
	return ""
}

func firstSidecarSeasonNumber(hints []SidecarHint) *int {
	for _, hint := range hints {
		if hint.SeasonNumber != nil {
			return hint.SeasonNumber
		}
	}
	return nil
}

func firstSidecarEpisodeNumber(hints []SidecarHint) *int {
	for _, hint := range hints {
		if hint.EpisodeNumber != nil {
			return hint.EpisodeNumber
		}
	}
	return nil
}

func firstNonNilIntPtr(values ...*int) *int {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func resourceShapeForSignal(signal database.InventoryFileSignal) string {
	if len(inventorySignalEpisodeNumbers(signal)) > 1 {
		return ResourceKindMultiEpisode
	}
	return ResourceKindSingleFile
}
