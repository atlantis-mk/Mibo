package recognition

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/titleclean"
)

const (
	evidenceKindInventoryFact    = "inventory_fact"
	evidenceKindLinkedResource   = "linked_resource"
	evidenceKindFileSignal       = "file_signal"
	evidenceKindSidecar          = "sidecar"
	evidenceKindDirectoryContext = "directory_context"
	evidenceKindScanExclusion    = "scan_exclusion"

	evidenceSourceInventory    = "inventory_file"
	evidenceSourceResource     = "resource_file"
	evidenceSourceSignal       = "inventory_file_signal"
	evidenceSourceSidecar      = "sidecar_association"
	evidenceSourceContentShape = "content_shape"
	evidenceSourcePathTree     = "path_tree"
	evidenceSourceExclusion    = "scan_exclusion"

	videoContentClass = "video"
)

type ManifestBuildInput struct {
	Scope            ManifestScope
	Files            []database.InventoryFile
	ResourceFiles    map[uint][]database.ResourceFile
	FileSignals      map[uint]database.InventoryFileSignal
	SidecarsByFileID map[uint][]database.InventoryFile
	SidecarHints     map[uint][]SidecarHint
	ContextEvidence  map[uint][]ContextEvidence
	ExcludedFileIDs  map[uint]string
}

type SidecarHint struct {
	Path          string            `json:"path"`
	Extension     string            `json:"extension"`
	ParseStatus   string            `json:"parse_status"`
	MediaType     string            `json:"media_type,omitempty"`
	Title         string            `json:"title,omitempty"`
	OriginalTitle string            `json:"original_title,omitempty"`
	Year          *int              `json:"year,omitempty"`
	SeriesTitle   string            `json:"series_title,omitempty"`
	SeasonNumber  *int              `json:"season_number,omitempty"`
	EpisodeNumber *int              `json:"episode_number,omitempty"`
	ExternalIDs   map[string]string `json:"external_ids,omitempty"`
	Fields        map[string]any    `json:"fields,omitempty"`
}

type ContextEvidence struct {
	Source      string         `json:"source"`
	Assignment  string         `json:"assignment,omitempty"`
	TargetKey   string         `json:"target_key,omitempty"`
	ParentKey   string         `json:"parent_key,omitempty"`
	VariantKey  string         `json:"variant_key,omitempty"`
	EditionKey  string         `json:"edition_key,omitempty"`
	ReviewState string         `json:"review_state,omitempty"`
	Confidence  *float64       `json:"confidence,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
}

type ManifestBuildOutput struct {
	ManifestScope ManifestScope
	Candidates    []database.RecognitionCandidate
	Evidence      []database.RecognitionEvidence
}

func BuildManifestFromInventory(input ManifestBuildInput) ManifestBuildOutput {
	scope := input.Scope
	if strings.TrimSpace(scope.ManifestKey) == "" {
		scope.ManifestKey = ManifestKey(Scope{StorageProvider: scope.StorageProvider, RootPath: scope.RootPath, ScopePath: scope.ScopePath}, scope.ClassifierVersion)
	}
	if strings.TrimSpace(scope.Fingerprint) == "" {
		scope.Fingerprint = inventoryManifestFingerprint(input.Files)
	}
	if scope.ObservedAt.IsZero() {
		scope.ObservedAt = time.Now().UTC()
	}

	output := ManifestBuildOutput{ManifestScope: scope}
	for _, file := range input.Files {
		if !eligibleInventoryFile(file) {
			continue
		}
		resourceKey := PlayableResourceKey(ResourceInput{StorageProvider: file.StorageProvider, StoragePath: file.StoragePath, StableIdentityKey: file.StableIdentityKey})
		if resourceKey == "" {
			continue
		}
		grouped := groupedCandidatesForFile(file, input.FileSignals[file.ID], input.SidecarHints[file.ID], input.ContextEvidence[file.ID], resourceKey)
		parentKey := firstParentCandidateKey(grouped)
		variantKey := firstCandidateTraitKey(grouped, CandidateTypeVariant)
		editionKey := firstCandidateTraitKey(grouped, CandidateTypeEdition)
		confidence := 0.6
		candidate := database.RecognitionCandidate{
			CandidateKey:       resourceKey,
			CandidateType:      CandidateTypePlayableResource,
			CandidateRole:      "source",
			ParentCandidateKey: parentKey,
			PrimaryInventoryID: uintPtr(file.ID),
			CanonicalKey:       parentKey,
			VariantKey:         variantKey,
			EditionKey:         editionKey,
			ResourceShape:      ResourceKindSingleFile,
			ReviewState:        database.ReviewStatePending,
			Confidence:         &confidence,
			AffectedFilesJSON:  mustJSON([]string{strings.TrimSpace(file.StoragePath)}),
			EvidenceJSON:       mustJSON(map[string]any{"storage_path": strings.TrimSpace(file.StoragePath), "storage_provider": strings.TrimSpace(file.StorageProvider), "stable_identity_key": strings.TrimSpace(file.StableIdentityKey), "hashes_json": strings.TrimSpace(file.HashesJSON)}),
		}
		if reason := strings.TrimSpace(input.ExcludedFileIDs[file.ID]); reason != "" {
			candidate.ReviewState = database.ReviewStateRejected
			candidate.CandidateRole = "excluded"
		}
		output.Candidates = append(output.Candidates, candidate)
		output.Evidence = append(output.Evidence, inventoryEvidence(file, resourceKey)...)
		for _, resourceFile := range input.ResourceFiles[file.ID] {
			output.Evidence = append(output.Evidence, database.RecognitionEvidence{CandidateID: nil, InventoryFileID: &file.ID, EvidenceKind: evidenceKindLinkedResource, EvidenceSource: evidenceSourceResource, EvidenceKey: resourceKey, EvidenceValue: mustJSON(resourceFile), Strength: "strong"})
		}
		if signal, ok := input.FileSignals[file.ID]; ok {
			output.Evidence = append(output.Evidence, signalEvidence(file.ID, resourceKey, signal)...)
		}
		for _, sidecar := range input.SidecarsByFileID[file.ID] {
			output.Evidence = append(output.Evidence, database.RecognitionEvidence{CandidateID: nil, InventoryFileID: &file.ID, EvidenceKind: evidenceKindSidecar, EvidenceSource: evidenceSourceSidecar, EvidenceKey: resourceKey, EvidenceValue: strings.TrimSpace(sidecar.StoragePath), Strength: "medium", PayloadJSON: mustJSON(sidecar)})
		}
		for _, hint := range input.SidecarHints[file.ID] {
			output.Evidence = append(output.Evidence, sidecarHintEvidence(file.ID, resourceKey, hint)...)
		}
		for _, contextEvidence := range input.ContextEvidence[file.ID] {
			output.Evidence = append(output.Evidence, directoryContextEvidence(file.ID, resourceKey, contextEvidence)...)
		}
		if reason := strings.TrimSpace(input.ExcludedFileIDs[file.ID]); reason != "" {
			output.Evidence = append(output.Evidence, database.RecognitionEvidence{CandidateID: nil, InventoryFileID: &file.ID, EvidenceKind: evidenceKindScanExclusion, EvidenceSource: evidenceSourceExclusion, EvidenceKey: resourceKey, EvidenceValue: reason, Strength: "strong"})
		}
		output.Candidates = append(output.Candidates, grouped...)
	}
	return output
}

func groupedCandidatesForFile(file database.InventoryFile, signal database.InventoryFileSignal, sidecarHints []SidecarHint, contextEvidence []ContextEvidence, resourceKey string) []database.RecognitionCandidate {
	candidates := make([]database.RecognitionCandidate, 0, 6)
	primaryFileID := uintPtr(file.ID)
	contextParentKey := firstContextParentKey(contextEvidence)
	contextSeriesKey := firstContextSeriesKey(contextEvidence)
	contextSeasonKey := firstContextSeasonKey(contextEvidence)
	hasEpisodeContext := contextSeriesKey != "" || contextSeasonKey != "" || contextEpisodeParentKey(contextParentKey) != ""
	contextVariantKey := firstContextVariantKey(contextEvidence)
	contextEditionKey := firstContextEditionKey(contextEvidence)
	if movieKey := firstNonEmptyString(contextMovieParentKey(contextEvidence), movieCandidateKeyWithoutEpisodeContext(hasEpisodeContext, file, signal, sidecarHints)); movieKey != "" {
		confidence := 0.75
		candidates = append(candidates, database.RecognitionCandidate{CandidateKey: movieKey, CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, PrimaryInventoryID: primaryFileID, CanonicalKey: movieKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, EvidenceJSON: candidateTitleEvidence(signal.TitleCandidate, signal.Year, sidecarHints), AffectedFilesJSON: mustJSON([]string{file.StoragePath})})
		if variantKey := firstNonEmptyString(contextVariantKey, variantCandidateKey(file, signal)); variantKey != "" {
			candidates = append(candidates, database.RecognitionCandidate{CandidateKey: joinKey(variantKey, resourceKey), CandidateType: CandidateTypeVariant, ParentCandidateKey: movieKey, PrimaryInventoryID: primaryFileID, CanonicalKey: movieKey, VariantKey: variantKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, AffectedFilesJSON: mustJSON([]string{file.StoragePath})})
		}
		if editionKey := firstNonEmptyString(contextEditionKey, editionCandidateKey(signal, sidecarHints)); editionKey != "" {
			candidates = append(candidates, database.RecognitionCandidate{CandidateKey: joinKey(editionKey, resourceKey), CandidateType: CandidateTypeEdition, ParentCandidateKey: movieKey, PrimaryInventoryID: primaryFileID, CanonicalKey: movieKey, EditionKey: editionKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, AffectedFilesJSON: mustJSON([]string{file.StoragePath})})
		}
	}
	if episodeKey := firstNonEmptyString(contextEpisodeParentKey(contextParentKey), episodeCandidateKey(signal, sidecarHints)); episodeKey != "" {
		confidence := 0.82
		if seriesKey := firstNonEmptyString(contextSeriesKey, seriesCandidateKeyFromEpisodeKey(episodeKey)); seriesKey != "" {
			candidates = append(candidates, database.RecognitionCandidate{CandidateKey: seriesKey, CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeries, PrimaryInventoryID: primaryFileID, CanonicalKey: seriesKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, EvidenceJSON: candidateSeriesEvidence(signal, sidecarHints), AffectedFilesJSON: mustJSON([]string{file.StoragePath})})
		}
		if seasonKey := firstNonEmptyString(contextSeasonKey, seasonCandidateKeyFromEpisodeKey(episodeKey)); seasonKey != "" {
			candidates = append(candidates, database.RecognitionCandidate{CandidateKey: seasonKey, CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeason, ParentCandidateKey: firstNonEmptyString(contextSeriesKey, seriesCandidateKeyFromEpisodeKey(episodeKey)), PrimaryInventoryID: primaryFileID, CanonicalKey: seasonKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, EvidenceJSON: candidateSeriesEvidence(signal, sidecarHints), AffectedFilesJSON: mustJSON([]string{file.StoragePath})})
		}
		candidates = append(candidates, database.RecognitionCandidate{CandidateKey: episodeKey, CandidateType: CandidateTypeEpisode, CandidateRole: WorkKindEpisode, PrimaryInventoryID: primaryFileID, CanonicalKey: episodeKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, EvidenceJSON: candidateTitleEvidence(signal.TitleCandidate, signal.Year, sidecarHints), AffectedFilesJSON: mustJSON([]string{file.StoragePath})})
		if variantKey := firstNonEmptyString(contextVariantKey, variantCandidateKey(file, signal)); variantKey != "" {
			candidates = append(candidates, database.RecognitionCandidate{CandidateKey: joinKey(variantKey, resourceKey), CandidateType: CandidateTypeVariant, ParentCandidateKey: episodeKey, PrimaryInventoryID: primaryFileID, CanonicalKey: episodeKey, VariantKey: variantKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, AffectedFilesJSON: mustJSON([]string{file.StoragePath})})
		}
		if editionKey := firstNonEmptyString(contextEditionKey, editionCandidateKey(signal, sidecarHints)); editionKey != "" {
			candidates = append(candidates, database.RecognitionCandidate{CandidateKey: joinKey(editionKey, resourceKey), CandidateType: CandidateTypeEdition, ParentCandidateKey: episodeKey, PrimaryInventoryID: primaryFileID, CanonicalKey: episodeKey, EditionKey: editionKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, AffectedFilesJSON: mustJSON([]string{file.StoragePath})})
		}
	}
	if len(candidates) == 0 && contextParentKey != "" {
		confidence := 0.7
		if strings.HasPrefix(contextParentKey, CandidateTypeEpisode+":") {
			if seriesKey := firstNonEmptyString(contextSeriesKey, seriesCandidateKeyFromEpisodeKey(contextParentKey)); seriesKey != "" {
				candidates = append(candidates, database.RecognitionCandidate{CandidateKey: seriesKey, CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeries, PrimaryInventoryID: primaryFileID, CanonicalKey: seriesKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, EvidenceJSON: candidateSeriesEvidence(signal, sidecarHints), AffectedFilesJSON: mustJSON([]string{file.StoragePath})})
			}
			if seasonKey := firstNonEmptyString(contextSeasonKey, seasonCandidateKeyFromEpisodeKey(contextParentKey)); seasonKey != "" {
				candidates = append(candidates, database.RecognitionCandidate{CandidateKey: seasonKey, CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeason, ParentCandidateKey: firstNonEmptyString(contextSeriesKey, seriesCandidateKeyFromEpisodeKey(contextParentKey)), PrimaryInventoryID: primaryFileID, CanonicalKey: seasonKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, EvidenceJSON: candidateSeriesEvidence(signal, sidecarHints), AffectedFilesJSON: mustJSON([]string{file.StoragePath})})
			}
			candidates = append(candidates, database.RecognitionCandidate{CandidateKey: contextParentKey, CandidateType: CandidateTypeEpisode, CandidateRole: WorkKindEpisode, PrimaryInventoryID: primaryFileID, CanonicalKey: contextParentKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, EvidenceJSON: candidateTitleEvidence(signal.TitleCandidate, signal.Year, sidecarHints), AffectedFilesJSON: mustJSON([]string{file.StoragePath})})
		} else if strings.HasPrefix(contextParentKey, CandidateTypeWork+":"+WorkKindMovie+":") {
			candidates = append(candidates, database.RecognitionCandidate{CandidateKey: contextParentKey, CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, PrimaryInventoryID: primaryFileID, CanonicalKey: contextParentKey, ReviewState: database.ReviewStatePending, Confidence: &confidence, EvidenceJSON: candidateTitleEvidence(signal.TitleCandidate, signal.Year, sidecarHints), AffectedFilesJSON: mustJSON([]string{file.StoragePath})})
		}
	}
	if role := strings.TrimSpace(signal.Role); role != "" && role != "main" {
		confidence := 0.8
		parentKey := firstParentCandidateKey(candidates)
		candidates = append(candidates, database.RecognitionCandidate{CandidateKey: SupplementalKey(parentKey, role, resourceKey), CandidateType: CandidateTypeSupplemental, CandidateRole: role, ParentCandidateKey: parentKey, PrimaryInventoryID: primaryFileID, ReviewState: database.ReviewStatePending, Confidence: &confidence, AffectedFilesJSON: mustJSON([]string{file.StoragePath})})
	}
	if hashKey := duplicateBinaryCandidateKey(file); hashKey != "" {
		confidence := 0.98
		candidates = append(candidates, database.RecognitionCandidate{CandidateKey: hashKey, CandidateType: CandidateTypeDuplicateBinary, CandidateRole: "same_binary", PrimaryInventoryID: primaryFileID, ReviewState: database.ReviewStatePending, Confidence: &confidence, AffectedFilesJSON: mustJSON([]string{file.StoragePath})})
	}
	return candidates
}

func candidateTitleEvidence(title string, year *int, sidecarHints []SidecarHint) string {
	for _, hint := range sidecarHints {
		if strings.TrimSpace(hint.Title) != "" {
			return mustJSON(map[string]any{"title": strings.TrimSpace(hint.Title), "year": hint.Year})
		}
		if strings.TrimSpace(hint.SeriesTitle) != "" {
			return mustJSON(map[string]any{"title": strings.TrimSpace(hint.SeriesTitle), "year": hint.Year})
		}
	}
	return mustJSON(map[string]any{"title": strings.TrimSpace(title), "year": year})
}

func candidateSeriesEvidence(signal database.InventoryFileSignal, sidecarHints []SidecarHint) string {
	for _, hint := range sidecarHints {
		if strings.TrimSpace(hint.SeriesTitle) != "" {
			return mustJSON(map[string]any{"title": strings.TrimSpace(hint.SeriesTitle), "year": hint.Year, "season_number": hint.SeasonNumber, "episode_number": hint.EpisodeNumber})
		}
	}
	return mustJSON(map[string]any{"title": strings.TrimSpace(signal.TitleCandidate), "year": signal.Year, "season_number": signal.SeasonNumber, "episode_number": signal.EpisodeNumber})
}

func movieWorkCandidateKey(file database.InventoryFile, signal database.InventoryFileSignal, sidecarHints []SidecarHint) string {
	for _, hint := range sidecarHints {
		if strings.EqualFold(strings.TrimSpace(hint.MediaType), WorkKindMovie) && strings.TrimSpace(hint.Title) != "" {
			return MovieWorkKey(MovieWorkInput{Title: titleclean.MovieWorkTitle(hint.Title), Year: hint.Year})
		}
	}
	if strings.TrimSpace(signal.TitleCandidate) != "" && movieTitleSignalAllowed(signal) {
		return MovieWorkKey(MovieWorkInput{Title: titleclean.MovieWorkTitle(signal.TitleCandidate), Year: signal.Year})
	}
	if strings.TrimSpace(signal.TitleCandidate) != "" && !movieTitleSignalAllowed(signal) {
		return ""
	}
	title := strings.TrimSuffix(path.Base(file.StoragePath), path.Ext(file.StoragePath))
	return MovieWorkKey(MovieWorkInput{Title: titleclean.MovieWorkTitle(title)})
}

func movieTitleSignalAllowed(signal database.InventoryFileSignal) bool {
	if signal.EpisodeNumber == nil {
		return true
	}
	return signal.SeasonNumber == nil && strings.TrimSpace(signal.EpisodeSource) == "leading_numeric"
}

func movieCandidateKeyWithoutEpisodeContext(hasEpisodeContext bool, file database.InventoryFile, signal database.InventoryFileSignal, sidecarHints []SidecarHint) string {
	if hasEpisodeContext {
		return ""
	}
	return movieWorkCandidateKey(file, signal, sidecarHints)
}

func episodeCandidateKey(signal database.InventoryFileSignal, sidecarHints []SidecarHint) string {
	for _, hint := range sidecarHints {
		if strings.TrimSpace(hint.SeriesTitle) != "" && hint.SeasonNumber != nil && hint.EpisodeNumber != nil {
			return EpisodeKey(EpisodeInput{SeriesTitle: hint.SeriesTitle, SeasonNumber: *hint.SeasonNumber, EpisodeNumber: *hint.EpisodeNumber})
		}
	}
	if strings.TrimSpace(signal.TitleCandidate) == "" || signal.SeasonNumber == nil || signal.EpisodeNumber == nil {
		return ""
	}
	return EpisodeKey(EpisodeInput{SeriesTitle: signal.TitleCandidate, SeasonNumber: *signal.SeasonNumber, EpisodeNumber: *signal.EpisodeNumber})
}

func variantCandidateKey(file database.InventoryFile, signal database.InventoryFileSignal) string {
	var sourceTags []string
	_ = json.Unmarshal([]byte(strings.TrimSpace(signal.SourceTagsJSON)), &sourceTags)
	return VariantKey(VariantInput{Quality: signal.Quality, SourceTags: sourceTags, Codec: signal.Codec, Audio: signal.Audio, Subtitle: signal.Subtitle, HDR: signal.HDR, Container: file.Container, ReleaseGroup: signal.ReleaseGroup})
}

func editionCandidateKey(signal database.InventoryFileSignal, sidecarHints []SidecarHint) string {
	if key := EditionKey(signal.Edition); key != "" {
		return key
	}
	for _, hint := range sidecarHints {
		if value, ok := hint.Fields["edition"].(string); ok {
			return EditionKey(value)
		}
	}
	return ""
}

func duplicateBinaryCandidateKey(file database.InventoryFile) string {
	var hashes map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(file.HashesJSON)), &hashes); err != nil {
		return ""
	}
	for _, key := range []string{"md5", "sha1", "sha256"} {
		if value := strings.TrimSpace(hashes[key]); value != "" {
			return DuplicateBinaryKey(key, value)
		}
	}
	return ""
}

func firstParentCandidateKey(candidates []database.RecognitionCandidate) string {
	for _, candidate := range candidates {
		if candidate.CandidateType == CandidateTypeWork || candidate.CandidateType == CandidateTypeEpisode {
			return candidate.CandidateKey
		}
	}
	return ""
}

func firstCandidateTraitKey(candidates []database.RecognitionCandidate, candidateType string) string {
	for _, candidate := range candidates {
		if candidate.CandidateType != candidateType {
			continue
		}
		if candidateType == CandidateTypeVariant && strings.TrimSpace(candidate.VariantKey) != "" {
			return strings.TrimSpace(candidate.VariantKey)
		}
		if candidateType == CandidateTypeEdition && strings.TrimSpace(candidate.EditionKey) != "" {
			return strings.TrimSpace(candidate.EditionKey)
		}
	}
	return ""
}

func firstContextParentKey(items []ContextEvidence) string {
	for _, assignment := range []string{"episode_identity", "single_episode_identity", "movie_multi_version", "single_work_identity", "movie_collection"} {
		for _, item := range items {
			if strings.TrimSpace(item.Assignment) != assignment {
				continue
			}
			if trimmed := strings.TrimSpace(item.ParentKey); trimmed != "" {
				return trimmed
			}
		}
	}
	for _, item := range items {
		if strings.TrimSpace(item.ParentKey) != "" && isPrimaryParentAssignment(item.Assignment) {
			return strings.TrimSpace(item.ParentKey)
		}
		if strings.TrimSpace(item.TargetKey) != "" && (strings.HasPrefix(strings.TrimSpace(item.TargetKey), CandidateTypeWork+":") || strings.HasPrefix(strings.TrimSpace(item.TargetKey), CandidateTypeEpisode+":")) {
			return strings.TrimSpace(item.TargetKey)
		}
	}
	return ""
}

func isPrimaryParentAssignment(assignment string) bool {
	switch strings.TrimSpace(assignment) {
	case "episode_identity", "movie_collection", "movie_multi_version", "single_work_identity", "single_episode_identity":
		return true
	default:
		return false
	}
}

func contextEpisodeParentKey(parentKey string) string {
	parentKey = strings.TrimSpace(parentKey)
	if strings.HasPrefix(parentKey, CandidateTypeEpisode+":") {
		return parentKey
	}
	return ""
}

func firstContextSeriesKey(items []ContextEvidence) string {
	for _, item := range items {
		if strings.TrimSpace(item.Assignment) == "series_identity" && strings.HasPrefix(strings.TrimSpace(item.ParentKey), CandidateTypeWork+":"+WorkKindSeries+":") {
			return strings.TrimSpace(item.ParentKey)
		}
	}
	return ""
}

func firstContextSeasonKey(items []ContextEvidence) string {
	for _, item := range items {
		if strings.TrimSpace(item.Assignment) == "season_identity" && strings.HasPrefix(strings.TrimSpace(item.ParentKey), CandidateTypeWork+":"+WorkKindSeason+":") {
			return strings.TrimSpace(item.ParentKey)
		}
	}
	return ""
}

func contextMovieParentKey(items []ContextEvidence) string {
	if hasEpisodeIdentityContext(items) {
		return ""
	}
	for _, item := range items {
		parentKey := strings.TrimSpace(item.ParentKey)
		if strings.TrimSpace(item.Assignment) == "movie_collection" && strings.HasPrefix(parentKey, CandidateTypeWork+":"+WorkKindMovie+":") {
			return parentKey
		}
	}
	return ""
}

func hasEpisodeIdentityContext(items []ContextEvidence) bool {
	for _, item := range items {
		switch strings.TrimSpace(item.Assignment) {
		case "episode_identity", "single_episode_identity":
			return true
		}
	}
	return false
}

func firstContextVariantKey(items []ContextEvidence) string {
	for _, item := range items {
		if strings.TrimSpace(item.VariantKey) != "" {
			return strings.TrimSpace(item.VariantKey)
		}
	}
	return ""
}

func firstContextEditionKey(items []ContextEvidence) string {
	for _, item := range items {
		if strings.TrimSpace(item.EditionKey) != "" {
			return strings.TrimSpace(item.EditionKey)
		}
	}
	return ""
}

func seriesCandidateKeyFromEpisodeKey(episodeKey string) string {
	trimmed := strings.TrimSpace(episodeKey)
	if !strings.HasPrefix(trimmed, CandidateTypeEpisode+":") {
		return ""
	}
	parts := strings.Split(trimmed, ":")
	if len(parts) < 6 {
		return ""
	}
	for idx := 1; idx < len(parts)-1; idx++ {
		if parts[idx] == WorkKindSeries {
			return joinKey(parts[1 : idx+2]...)
		}
	}
	return ""
}

func seasonCandidateKeyFromEpisodeKey(episodeKey string) string {
	trimmed := strings.TrimSpace(episodeKey)
	if !strings.HasPrefix(trimmed, CandidateTypeEpisode+":") {
		return ""
	}
	parts := strings.Split(trimmed, ":")
	if len(parts) < 3 {
		return ""
	}
	return joinKey(parts[1 : len(parts)-1]...)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func directoryContextEvidence(fileID uint, candidateKey string, context ContextEvidence) []database.RecognitionEvidence {
	source := strings.TrimSpace(context.Source)
	if source == "" {
		source = evidenceSourceContentShape
	}
	strength := "medium"
	if strings.TrimSpace(context.ReviewState) == database.ReviewStateAccepted || strings.TrimSpace(context.ReviewState) == "auto" {
		strength = "strong"
	}
	items := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceKind: evidenceKindDirectoryContext, EvidenceSource: source, EvidenceKey: candidateKey, EvidenceValue: strings.TrimSpace(context.Assignment), Strength: strength, Confidence: context.Confidence, PayloadJSON: mustJSON(context)}}
	add := func(key string, value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		items = append(items, database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: evidenceKindDirectoryContext, EvidenceSource: source, EvidenceKey: key, EvidenceValue: trimmed, Strength: strength, Confidence: context.Confidence, PayloadJSON: mustJSON(context)})
	}
	add("assignment", context.Assignment)
	add("target_key", context.TargetKey)
	add("review_state", context.ReviewState)
	for key, value := range context.Payload {
		switch typed := value.(type) {
		case string:
			add(key, typed)
		case int:
			add(key, stringInt(typed))
		case float64:
			if typed == float64(int(typed)) {
				add(key, stringInt(int(typed)))
			}
		}
	}
	return items
}

func sidecarHintEvidence(fileID uint, candidateKey string, hint SidecarHint) []database.RecognitionEvidence {
	items := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceKind: evidenceKindSidecar, EvidenceSource: evidenceSourceSidecar, EvidenceKey: candidateKey, EvidenceValue: strings.TrimSpace(hint.Path), Strength: "strong", PayloadJSON: mustJSON(hint)}}
	add := func(key string, value string, strength string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		items = append(items, database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: evidenceKindSidecar, EvidenceSource: evidenceSourceSidecar, EvidenceKey: key, EvidenceValue: trimmed, Strength: strength, PayloadJSON: mustJSON(hint)})
	}
	add("parse_status", hint.ParseStatus, "medium")
	add("media_type", hint.MediaType, "strong")
	add("title", hint.Title, "strong")
	add("original_title", hint.OriginalTitle, "medium")
	if hint.Year != nil {
		add("year", stringInt(*hint.Year), "strong")
	}
	add("series_title", hint.SeriesTitle, "strong")
	if hint.SeasonNumber != nil {
		add("season_number", stringInt(*hint.SeasonNumber), "strong")
	}
	if hint.EpisodeNumber != nil {
		add("episode_number", stringInt(*hint.EpisodeNumber), "strong")
	}
	for provider, externalID := range hint.ExternalIDs {
		if strings.TrimSpace(provider) == "" || strings.TrimSpace(externalID) == "" {
			continue
		}
		items = append(items, database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: evidenceKindSidecar, EvidenceSource: evidenceSourceSidecar, EvidenceKey: "external_id:" + strings.TrimSpace(provider), EvidenceValue: strings.TrimSpace(externalID), Strength: "strong", PayloadJSON: mustJSON(hint)})
	}
	if len(hint.Fields) > 0 {
		items = append(items, database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: evidenceKindSidecar, EvidenceSource: evidenceSourceSidecar, EvidenceKey: "fields", EvidenceValue: "sidecar_fields", Strength: "medium", PayloadJSON: mustJSON(hint.Fields)})
	}
	return items
}

func signalEvidence(fileID uint, candidateKey string, signal database.InventoryFileSignal) []database.RecognitionEvidence {
	items := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceKind: evidenceKindFileSignal, EvidenceSource: evidenceSourceSignal, EvidenceKey: candidateKey, EvidenceValue: strings.TrimSpace(signal.TitleCandidate), Strength: "medium", PayloadJSON: mustJSON(signal)}}
	add := func(key string, value string, strength string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		items = append(items, database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: evidenceKindFileSignal, EvidenceSource: evidenceSourceSignal, EvidenceKey: key, EvidenceValue: trimmed, Strength: strength})
	}
	add("title", signal.TitleCandidate, "medium")
	if signal.Year != nil {
		add("year", stringInt(*signal.Year), "medium")
	}
	if signal.SeasonNumber != nil {
		add("season_number", stringInt(*signal.SeasonNumber), "strong")
	}
	if signal.EpisodeNumber != nil {
		add("episode_number", stringInt(*signal.EpisodeNumber), "strong")
	}
	if signal.LeadingNumber != nil {
		add("leading_number", stringInt(*signal.LeadingNumber), "weak")
	}
	add("episode_source", signal.EpisodeSource, "medium")
	add("role", signal.Role, "strong")
	add("quality", signal.Quality, "medium")
	add("codec", signal.Codec, "medium")
	add("audio", signal.Audio, "medium")
	add("subtitle", signal.Subtitle, "medium")
	add("hdr", signal.HDR, "medium")
	add("edition", signal.Edition, "medium")
	add("release_group", signal.ReleaseGroup, "medium")
	if strings.TrimSpace(signal.SourceTagsJSON) != "" && strings.TrimSpace(signal.SourceTagsJSON) != "[]" {
		add("source_tags", signal.SourceTagsJSON, "medium")
	}
	if strings.TrimSpace(signal.EvidenceJSON) != "" && strings.TrimSpace(signal.EvidenceJSON) != "[]" {
		items = append(items, database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: evidenceKindFileSignal, EvidenceSource: evidenceSourceSignal, EvidenceKey: "signal_evidence", EvidenceValue: "filename_signal_evidence", Strength: "medium", PayloadJSON: signal.EvidenceJSON})
	}
	return items
}

func eligibleInventoryFile(file database.InventoryFile) bool {
	return file.ID != 0 && strings.TrimSpace(file.StoragePath) != "" && strings.TrimSpace(file.ContentClass) == videoContentClass && strings.TrimSpace(file.Status) != "missing" && path.Ext(file.StoragePath) != ""
}

func inventoryEvidence(file database.InventoryFile, candidateKey string) []database.RecognitionEvidence {
	items := []database.RecognitionEvidence{
		{InventoryFileID: &file.ID, EvidenceKind: evidenceKindInventoryFact, EvidenceSource: evidenceSourceInventory, EvidenceKey: candidateKey, EvidenceValue: strings.TrimSpace(file.StoragePath), Strength: "strong", PayloadJSON: mustJSON(map[string]any{"size_bytes": file.SizeBytes, "container": file.Container, "status": file.Status})},
	}
	if strings.TrimSpace(file.StableIdentityKey) != "" {
		items = append(items, database.RecognitionEvidence{InventoryFileID: &file.ID, EvidenceKind: evidenceKindInventoryFact, EvidenceSource: evidenceSourceInventory, EvidenceKey: "stable_identity_key", EvidenceValue: strings.TrimSpace(file.StableIdentityKey), Strength: "strong"})
	}
	if strings.TrimSpace(file.HashesJSON) != "" {
		items = append(items, database.RecognitionEvidence{InventoryFileID: &file.ID, EvidenceKind: evidenceKindInventoryFact, EvidenceSource: evidenceSourceInventory, EvidenceKey: "hashes_json", EvidenceValue: strings.TrimSpace(file.HashesJSON), Strength: "strong"})
	}
	return items
}

func inventoryManifestFingerprint(files []database.InventoryFile) string {
	parts := make([]string, 0, len(files))
	for _, file := range files {
		parts = append(parts, strings.Join([]string{strings.TrimSpace(file.StorageProvider), strings.TrimSpace(file.StoragePath), strings.TrimSpace(file.StableIdentityKey), strings.TrimSpace(file.HashesJSON), file.UpdatedAt.UTC().Format(time.RFC3339Nano)}, "\x00"))
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(digest[:])
}

func mustJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func uintPtr(value uint) *uint {
	if value == 0 {
		return nil
	}
	return &value
}

func stringInt(value int) string {
	return strconv.Itoa(value)
}
