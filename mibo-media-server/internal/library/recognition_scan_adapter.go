package library

import (
	"encoding/xml"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/recognition"
	"github.com/atlan/mibo-media-server/internal/scanrecognition"
)

const scanRecognitionEvidenceSource = "scanrecognition"

type scanRecognitionManifestOutput struct {
	Candidates []database.RecognitionCandidate
	Evidence   []database.RecognitionEvidence
}

func buildScanRecognitionManifestOutput(files []database.InventoryFile, rootPath string, sidecarHints map[uint][]recognition.SidecarHint) scanRecognitionManifestOutput {
	input := scanRecognitionInput(files, rootPath, sidecarHints)
	tree, err := scanrecognition.BuildTree(input)
	if err != nil || tree == nil || tree.Root == nil {
		return scanRecognitionManifestOutput{}
	}
	scanrecognition.ClassifyTree(tree)
	return scanRecognitionOutput(files, tree, sidecarHints)
}

func scanRecognitionInput(files []database.InventoryFile, rootPath string, sidecarHints map[uint][]recognition.SidecarHint) scanrecognition.Input {
	input := scanrecognition.Input{RootPath: rootPath}
	for _, file := range files {
		if !isRecognitionVideoFile(file) || extractFilenameSignalModel(file.StoragePath).VideoSignal.IsExtra {
			continue
		}
		input.Files = append(input.Files, scanrecognition.FileInput{ID: file.ID, Path: file.StoragePath, StorageProvider: file.StorageProvider, StableIdentityKey: file.StableIdentityKey, SizeBytes: file.SizeBytes, IsVideo: true})
		for _, hint := range sidecarHints[file.ID] {
			if sidecarText := syntheticScanNFO(hint); sidecarText != "" {
				input.Files = append(input.Files, scanrecognition.FileInput{Path: hint.Path, IsNFO: true, SidecarText: sidecarText})
			}
		}
	}
	return input
}

func scanRecognitionOutput(files []database.InventoryFile, tree *scanrecognition.Tree, sidecarHints map[uint][]recognition.SidecarHint) scanRecognitionManifestOutput {
	builder := scanRecognitionOutputBuilder{tree: tree, sidecarHints: sidecarHints, candidatesByKey: map[string]database.RecognitionCandidate{}, parentByDirectory: map[string]string{}}
	for _, file := range files {
		if !isRecognitionVideoFile(file) {
			continue
		}
		builder.addEvidence(file)
		if signal := extractFilenameSignalModel(file.StoragePath); signal.VideoSignal.IsExtra {
			continue
		}
		builder.addMainVideo(file)
	}
	for _, file := range files {
		if !isRecognitionVideoFile(file) {
			continue
		}
		signal := extractFilenameSignalModel(file.StoragePath)
		if !signal.VideoSignal.IsExtra {
			continue
		}
		builder.addSupplemental(file, signal)
	}
	return builder.output()
}

type scanRecognitionOutputBuilder struct {
	tree              *scanrecognition.Tree
	sidecarHints      map[uint][]recognition.SidecarHint
	candidatesByKey   map[string]database.RecognitionCandidate
	evidence          []database.RecognitionEvidence
	parentByDirectory map[string]string
}

func (b *scanRecognitionOutputBuilder) addMainVideo(file database.InventoryFile) {
	node := b.nodeForFile(file)
	if node == nil {
		return
	}
	effectiveKind := b.effectiveDirectoryKind(node)
	if node.Kind == scanrecognition.DirectoryKindRoot && effectiveKind != scanrecognition.DirectoryKindRoot {
		switch effectiveKind {
		case scanrecognition.DirectoryKindSeason, scanrecognition.DirectoryKindEpisodeGroup:
			b.addEpisodeVideo(file, node)
		case scanrecognition.DirectoryKindMovie, scanrecognition.DirectoryKindMovieVersions, scanrecognition.DirectoryKindMovieCollection:
			b.addMovieVideo(file, node, true)
		case scanrecognition.DirectoryKindUnknown, scanrecognition.DirectoryKindAmbiguous:
			if len(b.sidecarHints[file.ID]) == 0 {
				b.addMovieVideo(file, node, true)
			}
		}
		return
	}
	switch effectiveKind {
	case scanrecognition.DirectoryKindRoot:
		b.addRootVideo(file, node)
	case scanrecognition.DirectoryKindMovie, scanrecognition.DirectoryKindMovieVersions:
		b.addMovieVideo(file, node, len(node.Children) > 0)
	case scanrecognition.DirectoryKindMovieCollection:
		b.addMovieVideo(file, node, true)
	case scanrecognition.DirectoryKindSeason, scanrecognition.DirectoryKindEpisodeGroup:
		b.addEpisodeVideo(file, node)
	case scanrecognition.DirectoryKindUnknown, scanrecognition.DirectoryKindAmbiguous:
		if len(b.sidecarHints[file.ID]) == 0 {
			b.addMovieVideo(file, node, true)
		}
	}
}

func (b *scanRecognitionOutputBuilder) effectiveDirectoryKind(node *scanrecognition.DirectoryNode) scanrecognition.DirectoryKind {
	if node == nil {
		return scanrecognition.DirectoryKindUnknown
	}
	if node.Kind != scanrecognition.DirectoryKindRoot {
		return node.Kind
	}
	if len(node.DirectVideos) == 0 || strings.TrimSpace(node.Path) == "/" {
		return node.Kind
	}
	input := scanrecognition.Input{RootPath: path.Dir(node.Path)}
	for _, video := range node.DirectVideos {
		input.Files = append(input.Files, video)
	}
	for _, sidecar := range node.Sidecars {
		input.Files = append(input.Files, sidecar)
	}
	tree, err := scanrecognition.BuildTree(input)
	if err != nil || tree == nil {
		return node.Kind
	}
	scanrecognition.ClassifyTree(tree)
	if classifiedNode := tree.Node(node.Path); classifiedNode != nil {
		classified := classifiedNode.Kind
		if classified != scanrecognition.DirectoryKindRoot {
			return classified
		}
	}
	return node.Kind
}

func (b *scanRecognitionOutputBuilder) addRootVideo(file database.InventoryFile, node *scanrecognition.DirectoryNode) {
	videoSignal := scanrecognition.ParseVideoFilename(file.StoragePath)
	folderSignal := scanrecognition.ParseFolderName(node.Name)
	if videoSignal.Episode != nil && folderSignal.HasSeasonMarker {
		b.addEpisodeVideo(file, node)
		return
	}
	b.addMovieVideo(file, node, true)
}

func (b *scanRecognitionOutputBuilder) addMovieVideo(file database.InventoryFile, node *scanrecognition.DirectoryNode, perFile bool) {
	effectiveKind := b.effectiveDirectoryKind(node)
	model := extractFilenameSignalModel(file.StoragePath)
	hint := firstMovieSidecarHint(b.sidecarHints[file.ID])
	folderSignal := scanrecognition.ParseFolderName(node.Name)
	title := firstNonEmptyString(hint.Title, preferredMovieTitleCandidate(model, !perFile), firstScanTitle(folderSignal.TitleCandidates), path.Base(path.Dir(file.StoragePath)))
	year := firstNonNilInt(hint.Year, preferredMovieYearCandidate(model, !perFile))
	if !perFile {
		title = firstNonEmptyString(hint.Title, firstScanTitle(folderSignal.TitleCandidates), preferredMovieTitleCandidate(model, false), path.Base(path.Dir(file.StoragePath)))
		year = firstNonNilInt(hint.Year, commonMovieYear(node.DirectVideos), preferredMovieYearCandidate(model, true))
	}
	displayTitle := movieDisplayTitle(hint.Title, model, node, effectiveKind, title)
	movieKey := recognition.MovieWorkKey(recognition.MovieWorkInput{Title: title, Year: year})
	if movieKey == "" {
		return
	}
	confidence := 0.9
	b.addCandidate(database.RecognitionCandidate{CandidateKey: movieKey, CandidateType: recognition.CandidateTypeWork, CandidateRole: recognition.WorkKindMovie, CanonicalKey: movieKey, ReviewState: database.ReviewStatePending, Confidence: float64Ptr(confidence), EvidenceJSON: mustJSON(map[string]any{"source": scanRecognitionEvidenceSource, "directory_kind": effectiveKind, "title": displayTitle, "year": year, "work_title": title})})
	b.parentByDirectory[path.Dir(file.StoragePath)] = movieKey
	resourceKey := recognition.PlayableResourceKey(recognition.ResourceInput{StorageProvider: file.StorageProvider, StoragePath: file.StoragePath, StableIdentityKey: file.StableIdentityKey})
	if resourceKey == "" {
		return
	}
	variantKey := recognition.VariantKey(recognition.VariantInput{Quality: model.VideoSignal.Quality, SourceTags: model.VideoSignal.SourceTags, Codec: model.VideoSignal.Codec, Audio: model.VideoSignal.Audio, Subtitle: model.VideoSignal.Subtitle, HDR: model.VideoSignal.HDR, Container: file.Container, ReleaseGroup: model.VideoSignal.ReleaseGroup})
	editionKey := recognition.EditionKey(model.VideoSignal.Edition)
	b.addCandidate(database.RecognitionCandidate{CandidateKey: resourceKey, CandidateType: recognition.CandidateTypePlayableResource, CandidateRole: "source", ParentCandidateKey: movieKey, PrimaryInventoryID: uintPtr(file.ID), CanonicalKey: movieKey, VariantKey: variantKey, EditionKey: editionKey, ResourceShape: recognition.ResourceKindSingleFile, ReviewState: database.ReviewStatePending, Confidence: float64Ptr(confidence), EvidenceJSON: mustJSON(map[string]any{"source": scanRecognitionEvidenceSource, "directory_kind": effectiveKind, "path": file.StoragePath})})
	b.addTraitCandidates(movieKey, resourceKey, file, variantKey, editionKey, confidence)
}

func (b *scanRecognitionOutputBuilder) addEpisodeVideo(file database.InventoryFile, node *scanrecognition.DirectoryNode) {
	effectiveKind := b.effectiveDirectoryKind(node)
	videoSignal := scanrecognition.ParseVideoFilename(file.StoragePath)
	folderSignal := scanrecognition.ParseFolderName(node.Name)
	hint := firstEpisodeSidecarHint(b.sidecarHints[file.ID])
	seriesTitle := b.seriesTitle(node, videoSignal, folderSignal, hint)
	seasonNumber := firstPositiveInt(hint.SeasonNumber, videoSignal.Season, folderSignal.Season)
	if seasonNumber == 0 && effectiveKind == scanrecognition.DirectoryKindEpisodeGroup {
		seasonNumber = 1
	}
	episodeNumber := firstPositiveInt(hint.EpisodeNumber, videoSignal.Episode)
	if episodeNumber == 0 {
		episodeNumber = b.syntheticEpisodeNumber(node, file, videoSignal, folderSignal, effectiveKind)
	}
	if seriesTitle == "" || seasonNumber == 0 || episodeNumber == 0 {
		return
	}
	seriesKey := recognition.SeriesWorkKey(seriesTitle)
	seasonKey := recognition.SeasonWorkKey(seriesTitle, seasonNumber)
	episodeKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: seriesTitle, SeasonNumber: seasonNumber, EpisodeNumber: episodeNumber})
	if seriesKey == "" || seasonKey == "" || episodeKey == "" {
		return
	}
	displayTitle := seriesDisplayTitle(node, folderSignal, hint, seriesTitle)
	confidence := 0.9
	b.addCandidate(database.RecognitionCandidate{CandidateKey: seriesKey, CandidateType: recognition.CandidateTypeWork, CandidateRole: recognition.WorkKindSeries, CanonicalKey: seriesKey, ReviewState: database.ReviewStatePending, Confidence: float64Ptr(confidence), EvidenceJSON: mustJSON(map[string]any{"source": scanRecognitionEvidenceSource, "directory_kind": effectiveKind, "title": displayTitle, "series_title": seriesTitle})})
	b.addCandidate(database.RecognitionCandidate{CandidateKey: seasonKey, CandidateType: recognition.CandidateTypeWork, CandidateRole: recognition.WorkKindSeason, ParentCandidateKey: seriesKey, CanonicalKey: seasonKey, ReviewState: database.ReviewStatePending, Confidence: float64Ptr(confidence), EvidenceJSON: mustJSON(map[string]any{"source": scanRecognitionEvidenceSource, "directory_kind": effectiveKind, "title": displayTitle, "series_title": seriesTitle, "season_number": seasonNumber})})
	b.addCandidate(database.RecognitionCandidate{CandidateKey: episodeKey, CandidateType: recognition.CandidateTypeEpisode, CandidateRole: recognition.WorkKindEpisode, ParentCandidateKey: seasonKey, CanonicalKey: episodeKey, ReviewState: database.ReviewStatePending, Confidence: float64Ptr(confidence), EvidenceJSON: mustJSON(map[string]any{"source": scanRecognitionEvidenceSource, "title": displayTitle, "directory_kind": effectiveKind, "series_title": seriesTitle, "season_number": seasonNumber, "episode_number": episodeNumber})})
	b.parentByDirectory[path.Dir(file.StoragePath)] = episodeKey
	resourceKey := recognition.PlayableResourceKey(recognition.ResourceInput{StorageProvider: file.StorageProvider, StoragePath: file.StoragePath, StableIdentityKey: file.StableIdentityKey})
	if resourceKey == "" {
		return
	}
	b.addCandidate(database.RecognitionCandidate{CandidateKey: resourceKey, CandidateType: recognition.CandidateTypePlayableResource, CandidateRole: "source", ParentCandidateKey: episodeKey, PrimaryInventoryID: uintPtr(file.ID), CanonicalKey: episodeKey, ResourceShape: recognition.ResourceKindSingleFile, ReviewState: database.ReviewStatePending, Confidence: float64Ptr(confidence), EvidenceJSON: mustJSON(map[string]any{"source": scanRecognitionEvidenceSource, "directory_kind": effectiveKind, "path": file.StoragePath})})
	fileID := file.ID
	b.evidence = append(b.evidence,
		database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: "directory_context", EvidenceSource: "content_shape", EvidenceKey: "series_title", EvidenceValue: seriesTitle, Strength: "strong", PayloadJSON: mustJSON(map[string]any{"source": scanRecognitionEvidenceSource})},
		database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: "directory_context", EvidenceSource: "content_shape", EvidenceKey: "season_number", EvidenceValue: strconv.Itoa(seasonNumber), Strength: "strong", PayloadJSON: mustJSON(map[string]any{"source": scanRecognitionEvidenceSource})},
	)
}

func (b *scanRecognitionOutputBuilder) syntheticEpisodeNumber(node *scanrecognition.DirectoryNode, file database.InventoryFile, signal scanrecognition.VideoSignal, folderSignal scanrecognition.FolderSignal, kind scanrecognition.DirectoryKind) int {
	if node == nil {
		return 0
	}
	if kind != scanrecognition.DirectoryKindEpisodeGroup && kind != scanrecognition.DirectoryKindSeason {
		return 0
	}
	base := 0
	if folderSignal.ExpectedEpisodeCount != nil && *folderSignal.ExpectedEpisodeCount > 0 {
		base = *folderSignal.ExpectedEpisodeCount
	} else {
		base = b.maxPositiveEpisodeNumber(node)
	}
	if signal.IsSpecial && signal.SpecialIndex != nil && *signal.SpecialIndex > 0 {
		return base + *signal.SpecialIndex
	}
	residualOffset := b.residualEpisodeOffset(node, file.StoragePath)
	if residualOffset == 0 {
		return 0
	}
	return base + b.maxSpecialIndex(node) + residualOffset
}

func (b *scanRecognitionOutputBuilder) maxPositiveEpisodeNumber(node *scanrecognition.DirectoryNode) int {
	maxEpisode := 0
	for _, video := range node.DirectVideos {
		signal := scanrecognition.ParseVideoFilename(video.Path)
		if signal.Episode != nil && *signal.Episode > maxEpisode {
			maxEpisode = *signal.Episode
		}
	}
	return maxEpisode
}

func (b *scanRecognitionOutputBuilder) maxSpecialIndex(node *scanrecognition.DirectoryNode) int {
	maxIndex := 0
	for _, video := range node.DirectVideos {
		signal := scanrecognition.ParseVideoFilename(video.Path)
		if signal.SpecialIndex != nil && *signal.SpecialIndex > maxIndex {
			maxIndex = *signal.SpecialIndex
		}
	}
	return maxIndex
}

func (b *scanRecognitionOutputBuilder) residualEpisodeOffset(node *scanrecognition.DirectoryNode, storagePath string) int {
	target := strings.TrimSpace(storagePath)
	if target == "" {
		return 0
	}
	offset := 0
	for _, video := range node.DirectVideos {
		signal := scanrecognition.ParseVideoFilename(video.Path)
		if signal.Episode != nil && *signal.Episode > 0 {
			continue
		}
		if signal.SpecialIndex != nil && *signal.SpecialIndex > 0 {
			continue
		}
		offset++
		if strings.TrimSpace(video.Path) == target {
			return offset
		}
	}
	return 0
}

func (b *scanRecognitionOutputBuilder) addSupplemental(file database.InventoryFile, signal filenameSignalModel) {
	parentKey := b.parentByDirectory[path.Dir(file.StoragePath)]
	if parentKey == "" {
		return
	}
	resourceKey := recognition.PlayableResourceKey(recognition.ResourceInput{StorageProvider: file.StorageProvider, StoragePath: file.StoragePath, StableIdentityKey: file.StableIdentityKey})
	if resourceKey == "" {
		return
	}
	confidence := 0.8
	b.addCandidate(database.RecognitionCandidate{CandidateKey: recognition.SupplementalKey(parentKey, signal.RoleHints.Role, resourceKey), CandidateType: recognition.CandidateTypeSupplemental, CandidateRole: signal.RoleHints.Role, ParentCandidateKey: parentKey, PrimaryInventoryID: uintPtr(file.ID), ReviewState: database.ReviewStatePending, Confidence: float64Ptr(confidence), EvidenceJSON: mustJSON(map[string]any{"source": scanRecognitionEvidenceSource, "role": signal.RoleHints.Role, "path": file.StoragePath})})
}

func (b *scanRecognitionOutputBuilder) addTraitCandidates(parentKey string, resourceKey string, file database.InventoryFile, variantKey string, editionKey string, confidence float64) {
	if variantKey != "" {
		b.addCandidate(database.RecognitionCandidate{CandidateKey: variantKey + ":" + resourceKey, CandidateType: recognition.CandidateTypeVariant, ParentCandidateKey: parentKey, PrimaryInventoryID: uintPtr(file.ID), CanonicalKey: parentKey, VariantKey: variantKey, ReviewState: database.ReviewStatePending, Confidence: float64Ptr(confidence), AffectedFilesJSON: mustJSON([]string{file.StoragePath})})
	}
	if editionKey != "" {
		b.addCandidate(database.RecognitionCandidate{CandidateKey: editionKey + ":" + resourceKey, CandidateType: recognition.CandidateTypeEdition, ParentCandidateKey: parentKey, PrimaryInventoryID: uintPtr(file.ID), CanonicalKey: parentKey, EditionKey: editionKey, ReviewState: database.ReviewStatePending, Confidence: float64Ptr(confidence), AffectedFilesJSON: mustJSON([]string{file.StoragePath})})
	}
}

func (b *scanRecognitionOutputBuilder) addEvidence(file database.InventoryFile) {
	node := b.nodeForFile(file)
	kind := scanrecognition.DirectoryKindUnknown
	if node != nil {
		kind = b.effectiveDirectoryKind(node)
	}
	fileID := file.ID
	b.evidence = append(b.evidence, database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: "directory_context", EvidenceSource: scanRecognitionEvidenceSource, EvidenceKey: strings.TrimSpace(file.StoragePath), EvidenceValue: string(kind), Strength: "strong", PayloadJSON: mustJSON(map[string]any{"path": file.StoragePath, "directory_kind": kind})})
	b.addSidecarEvidence(fileID, b.sidecarHints[file.ID])
}

func (b *scanRecognitionOutputBuilder) addSidecarEvidence(fileID uint, hints []recognition.SidecarHint) {
	for _, hint := range hints {
		if hint.SeriesTitle != "" {
			b.evidence = append(b.evidence, database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: "sidecar", EvidenceSource: "sidecar_association", EvidenceKey: "series_title", EvidenceValue: hint.SeriesTitle, Strength: "strong", PayloadJSON: mustJSON(hint)})
		}
		if hint.Title != "" {
			b.evidence = append(b.evidence, database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: "sidecar", EvidenceSource: "sidecar_association", EvidenceKey: "title", EvidenceValue: hint.Title, Strength: "medium", PayloadJSON: mustJSON(hint)})
		}
		for key, value := range hint.ExternalIDs {
			if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
				continue
			}
			b.evidence = append(b.evidence, database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: "sidecar", EvidenceSource: "sidecar_association", EvidenceKey: "external_id:" + strings.TrimSpace(key), EvidenceValue: strings.TrimSpace(value), Strength: "strong", PayloadJSON: mustJSON(hint)})
		}
	}
}

func (b *scanRecognitionOutputBuilder) addCandidate(candidate database.RecognitionCandidate) {
	key := strings.TrimSpace(candidate.CandidateKey)
	if key == "" {
		return
	}
	if _, ok := b.candidatesByKey[key]; ok {
		return
	}
	candidate.CandidateKey = key
	b.candidatesByKey[key] = candidate
}

func (b *scanRecognitionOutputBuilder) nodeForFile(file database.InventoryFile) *scanrecognition.DirectoryNode {
	if b.tree == nil {
		return nil
	}
	return b.tree.Node(path.Dir(strings.TrimSpace(file.StoragePath)))
}

func (b *scanRecognitionOutputBuilder) seriesTitle(node *scanrecognition.DirectoryNode, videoSignal scanrecognition.VideoSignal, folderSignal scanrecognition.FolderSignal, hint recognition.SidecarHint) string {
	if recognition.SeriesWorkKey(hint.SeriesTitle) != "" {
		return hint.SeriesTitle
	}
	if title := bestSeriesFolderTitle(folderSignal, videoSignal.TitleCandidates); title != "" {
		return title
	}
	if folderSignal.HasSeasonMarker {
		if parent := b.tree.Node(path.Dir(node.Path)); parent != nil && parent.Kind != scanrecognition.DirectoryKindRoot && recognition.SeriesWorkKey(parent.Name) != "" {
			return parent.Name
		}
		if title := strings.TrimSpace(path.Base(path.Dir(node.Path))); recognition.SeriesWorkKey(title) != "" {
			return title
		}
	}
	if (videoSignal.Episode != nil && *videoSignal.Episode > 0) || videoSignal.IsSpecial || videoSignal.SpecialIndex != nil {
		if title := firstKeyableSeriesTitle(videoSignal.TitleCandidates); title != "" {
			return title
		}
	}
	if title := firstKeyableSeriesTitle(videoSignal.TitleCandidates); title != "" {
		return title
	}
	return firstNonEmptyString(hint.SeriesTitle, bestSeriesFolderTitle(folderSignal, videoSignal.TitleCandidates), firstScanTitle(videoSignal.TitleCandidates))
}

func (b *scanRecognitionOutputBuilder) output() scanRecognitionManifestOutput {
	result := scanRecognitionManifestOutput{Evidence: b.evidence}
	for _, candidate := range b.candidatesByKey {
		result.Candidates = append(result.Candidates, candidate)
	}
	sort.Slice(result.Candidates, func(i, j int) bool { return result.Candidates[i].CandidateKey < result.Candidates[j].CandidateKey })
	return result
}

func buildScanRecognitionDecisions(candidates []database.RecognitionCandidate) []database.RecognitionDecision {
	decisions := make([]database.RecognitionDecision, 0, len(candidates))
	for _, candidate := range candidates {
		outcome := database.ReviewStateAccepted
		if candidate.CandidateType == recognition.CandidateTypePlayableResource || candidate.CandidateType == recognition.CandidateTypeEpisode || candidate.CandidateType == recognition.CandidateTypeWork {
			outcome = database.ReviewStateAccepted
		}
		decision := database.RecognitionDecision{ManifestID: candidate.ManifestID, CandidateID: uintPtr(candidate.ID), DecisionType: "scanrecognition_outcome", Outcome: outcome, TargetKind: candidate.CandidateType, TargetKey: candidate.CandidateKey, TargetMetadataID: candidate.TargetMetadataID, TargetResourceID: candidate.TargetResourceID, Confidence: candidate.Confidence, Reason: "scan recognition accepted candidate", EvidenceJSON: candidate.EvidenceJSON, AlternativesJSON: candidate.AlternativesJSON}
		decisions = append(decisions, decision)
	}
	return decisions
}

func syntheticScanNFO(hint recognition.SidecarHint) string {
	if !strings.EqualFold(strings.TrimSpace(hint.Extension), ".nfo") || !strings.EqualFold(strings.TrimSpace(hint.ParseStatus), "parsed") {
		return ""
	}
	root := syntheticNFORoot(hint)
	if root == "" {
		return ""
	}
	document := syntheticNFODocument{XMLName: xml.Name{Local: root}, Title: firstNonEmptyString(hint.Title, hint.SeriesTitle), Year: hint.Year, Season: hint.SeasonNumber, Episode: hint.EpisodeNumber}
	if hint.ExternalIDs != nil {
		document.TMDBID = hint.ExternalIDs["tmdb"]
		document.IMDbID = hint.ExternalIDs["imdb"]
	}
	encoded, err := xml.Marshal(document)
	if err != nil {
		return ""
	}
	return string(encoded)
}

type syntheticNFODocument struct {
	XMLName xml.Name
	Title   string `xml:"title,omitempty"`
	Year    *int   `xml:"year,omitempty"`
	Season  *int   `xml:"season,omitempty"`
	Episode *int   `xml:"episode,omitempty"`
	TMDBID  string `xml:"tmdbid,omitempty"`
	IMDbID  string `xml:"imdbid,omitempty"`
}

func syntheticNFORoot(hint recognition.SidecarHint) string {
	mediaType := strings.ToLower(strings.TrimSpace(hint.MediaType))
	switch mediaType {
	case recognition.WorkKindMovie:
		return "movie"
	case recognition.WorkKindSeries:
		return "tvshow"
	case recognition.WorkKindSeason:
		return "season"
	case recognition.WorkKindEpisode:
		return "episodedetails"
	}
	if hint.EpisodeNumber != nil || hint.SeasonNumber != nil && hint.SeriesTitle != "" {
		return "episodedetails"
	}
	return ""
}

func firstMovieSidecarHint(hints []recognition.SidecarHint) recognition.SidecarHint {
	for _, hint := range hints {
		if strings.EqualFold(strings.TrimSpace(hint.MediaType), recognition.WorkKindMovie) {
			return hint
		}
	}
	return recognition.SidecarHint{}
}

func firstEpisodeSidecarHint(hints []recognition.SidecarHint) recognition.SidecarHint {
	for _, hint := range hints {
		if hint.EpisodeNumber != nil || hint.SeasonNumber != nil || strings.EqualFold(strings.TrimSpace(hint.MediaType), recognition.WorkKindEpisode) {
			return hint
		}
	}
	return recognition.SidecarHint{}
}

func firstScanTitle(candidates []string) string {
	for _, candidate := range candidates {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstKeyableSeriesTitle(candidates []string) string {
	for _, candidate := range candidates {
		if scanrecognition.GenericMediaNameSignal(candidate) != "" {
			continue
		}
		if recognition.SeriesWorkKey(candidate) != "" {
			return strings.TrimSpace(candidate)
		}
	}
	return ""
}

func bestSeriesFolderTitle(folderSignal scanrecognition.FolderSignal, videoCandidates []string) string {
	if len(folderSignal.TitleCandidates) == 0 {
		return ""
	}
	videoTitle := firstKeyableSeriesTitle(videoCandidates)
	videoNormalized := normalizeVersionCompareTitle(firstScanTitle(videoCandidates))
	if videoNormalized != "" {
		for _, candidate := range folderSignal.TitleCandidates {
			trimmed := strings.TrimSpace(candidate)
			if trimmed == "" || scanrecognition.GenericMediaNameSignal(trimmed) != "" {
				continue
			}
			normalized := normalizeVersionCompareTitle(trimmed)
			if normalized == videoNormalized {
				return trimSeriesSeasonSuffix(trimmed)
			}
			if strings.Contains(normalized, videoNormalized) {
				if videoTitle != "" {
					return trimSeriesSeasonSuffix(videoTitle)
				}
				return trimSeriesSeasonSuffix(trimmed)
			}
		}
	}
	return trimSeriesSeasonSuffix(firstKeyableSeriesTitle(folderSignal.TitleCandidates))
}

func trimSeriesSeasonSuffix(title string) string {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return ""
	}
	parsed := scanrecognition.ParseFolderName(trimmed)
	if parsed.Season == nil {
		return trimmed
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return trimmed
	}
	last := strings.ToLower(parts[len(parts)-1])
	if strings.HasPrefix(last, "s") && len(last) > 1 {
		return strings.TrimSpace(strings.Join(parts[:len(parts)-1], " "))
	}
	return trimmed
}

func movieDisplayTitle(hintTitle string, model filenameSignalModel, node *scanrecognition.DirectoryNode, kind scanrecognition.DirectoryKind, fallback string) string {
	if title := strings.TrimSpace(hintTitle); title != "" {
		return title
	}
	if title := movieFolderDisplayTitle(model, node, kind); title != "" {
		return title
	}
	return strings.TrimSpace(fallback)
}

func movieFolderDisplayTitle(model filenameSignalModel, node *scanrecognition.DirectoryNode, kind scanrecognition.DirectoryKind) string {
	if node == nil {
		return ""
	}
	switch kind {
	case scanrecognition.DirectoryKindMovie, scanrecognition.DirectoryKindMovieVersions, scanrecognition.DirectoryKindMovieCollection:
	default:
		return ""
	}
	folderName := path.Base(strings.TrimSpace(model.RawPathData.Directory))
	if node.Kind == scanrecognition.DirectoryKindRoot && scanrecognition.GenericMediaNameSignal(folderName) != "" {
		return ""
	}
	folderSignal := scanrecognition.ParseFolderName(folderName)
	if folderSignal.Season != nil {
		return ""
	}
	// Validate against all folder candidates, but display the first one so localized
	// names like "辣身舞" can be shown while "Dirty Dancing" remains the work key.
	if movieFolderTitleCandidate(model) == "" {
		return ""
	}
	return strings.TrimSpace(firstScanTitle(folderSignal.TitleCandidates))
}

func seriesDisplayTitle(node *scanrecognition.DirectoryNode, folderSignal scanrecognition.FolderSignal, hint recognition.SidecarHint, fallback string) string {
	if title := strings.TrimSpace(firstNonEmptyString(hint.SeriesTitle, hint.Title)); title != "" {
		return title
	}
	if title := firstScanTitle(folderSignal.TitleCandidates); title != "" {
		return title
	}
	if folderSignal.HasSeasonMarker && node != nil {
		parentTitle := strings.TrimSpace(path.Base(path.Dir(node.Path)))
		if parentTitle != "" && parentTitle != "." && parentTitle != "/" {
			return parentTitle
		}
	}
	return strings.TrimSpace(fallback)
}

func commonMovieYear(files []scanrecognition.FileInput) *int {
	var common *int
	for _, file := range files {
		signal := scanrecognition.ParseVideoFilename(file.Path)
		if signal.Year == nil {
			return nil
		}
		if common == nil {
			year := *signal.Year
			common = &year
			continue
		}
		if *common != *signal.Year {
			return nil
		}
	}
	return common
}

func firstPositiveInt(values ...*int) int {
	for _, value := range values {
		if value != nil && *value > 0 {
			return *value
		}
	}
	return 0
}

func isRecognitionVideoFile(file database.InventoryFile) bool {
	return file.ID != 0 && strings.TrimSpace(file.StoragePath) != "" && isVideoFile(file.StoragePath)
}

func uintPtr(value uint) *uint {
	result := value
	return &result
}

func float64Ptr(value float64) *float64 {
	result := value
	return &result
}
