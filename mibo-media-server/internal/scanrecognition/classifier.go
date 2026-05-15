package scanrecognition

import (
	"sort"
	"strings"
)

const directorySampleLimit = 40

type classifiedVideo struct {
	file   FileInput
	signal VideoSignal
}

type episodeStats struct {
	totalCount             int
	episodicCount          int
	specialCount           int
	seasonCounts           map[int]int
	episodeTitleCounts     map[string]int
	episodeIdentityCounts  map[string]int
	dominantTitleCount     int
	dominantIdentityCount  int
	uniqueEpisodeIdentites int
}

type movieStats struct {
	totalCount          int
	identityCounts      map[string]int
	dominantIdentity    string
	dominantCount       int
	uniqueIdentityCount int
}

func ClassifyTree(tree *Tree) *Tree {
	if tree == nil || tree.Root == nil {
		return tree
	}
	classifyNode(tree.Root, tree.Root)
	return tree
}

func classifyNode(node *DirectoryNode, root *DirectoryNode) {
	if node == nil {
		return
	}
	for _, child := range node.Children {
		classifyNode(child, root)
	}
	if node == root {
		node.Kind = DirectoryKindRoot
		return
	}
	node.Kind = classifyDirectory(node)
}

func classifyDirectory(node *DirectoryNode) DirectoryKind {
	if len(node.DirectVideos) > 0 {
		return classifyDirectVideos(node)
	}
	if hasOnlySeriesChildren(node) {
		return DirectoryKindSeries
	}
	return DirectoryKindUnknown
}

func classifyDirectVideos(node *DirectoryNode) DirectoryKind {
	folderSignal := ParseFolderName(node.Name)
	videos := classifiedVideosForNode(node)
	if len(videos) == 0 {
		return DirectoryKindUnknown
	}
	nfoSignals := parseNodeNFOSignals(node)
	videoSignals := extractVideoSignals(videos)
	if hasNFOConflict(nfoSignals, folderSignal, videoSignals) {
		return DirectoryKindUnknown
	}

	episodeStats := collectEpisodeStats(videos)
	movieStats := collectMovieStats(videos)

	if hasEpisodeNFO(nfoSignals) {
		if folderSignal.HasSeasonMarker && nfosMatchFolderSeason(nfoSignals, folderSignal.Season) {
			return DirectoryKindSeason
		}
		if episodeStats.episodicCount > 0 || len(videos) > 1 {
			return DirectoryKindEpisodeGroup
		}
	}

	episodeKind, episodeConfidence := classifyEpisodeDirectory(folderSignal, episodeStats, len(videos), node)
	movieKind, movieConfidence := classifyMovieDirectory(movieStats, len(videos))

	if hasMovieNFO(nfoSignals) {
		if movieKind == DirectoryKindUnknown {
			if len(videos) == 1 {
				return DirectoryKindMovie
			}
			return DirectoryKindUnknown
		}
		movieConfidence += 2
	}

	if episodeKind != DirectoryKindUnknown && episodeConfidence >= movieConfidence && episodeConfidence >= 4 {
		return episodeKind
	}
	if movieKind != DirectoryKindUnknown && movieConfidence >= 3 {
		return movieKind
	}
	if episodeKind != DirectoryKindUnknown && episodeConfidence >= 3 {
		return episodeKind
	}
	return DirectoryKindUnknown
}

func classifiedVideosForNode(node *DirectoryNode) []classifiedVideo {
	primaryVideos := make([]classifiedVideo, 0, len(node.DirectVideos))
	fallbackVideos := make([]classifiedVideo, 0, len(node.DirectVideos))
	for _, video := range sampledVideos(node.DirectVideos) {
		classified := classifiedVideo{file: video, signal: ParseVideoFilename(video.Path)}
		fallbackVideos = append(fallbackVideos, classified)
		if isPrimaryVideo(video.Path) {
			primaryVideos = append(primaryVideos, classified)
		}
	}
	if len(primaryVideos) > 0 {
		return primaryVideos
	}
	return fallbackVideos
}

func sampledVideos(videos []FileInput) []FileInput {
	if len(videos) <= directorySampleLimit {
		return videos
	}
	indices := map[int]struct{}{}
	for idx := 0; idx < 10; idx++ {
		indices[idx] = struct{}{}
	}
	for idx := len(videos) - 10; idx < len(videos); idx++ {
		indices[idx] = struct{}{}
	}
	step := len(videos) / 20
	if step < 1 {
		step = 1
	}
	for idx := 10; idx < len(videos)-10; idx += step {
		indices[idx] = struct{}{}
	}
	ordered := make([]int, 0, len(indices))
	for idx := range indices {
		ordered = append(ordered, idx)
	}
	sort.Ints(ordered)
	sampled := make([]FileInput, 0, len(ordered))
	for _, idx := range ordered {
		sampled = append(sampled, videos[idx])
	}
	return sampled
}

func extractVideoSignals(videos []classifiedVideo) []VideoSignal {
	signals := make([]VideoSignal, 0, len(videos))
	for _, video := range videos {
		signals = append(signals, video.signal)
	}
	return signals
}

func parseNodeNFOSignals(node *DirectoryNode) []NFOSignal {
	signals := make([]NFOSignal, 0, len(node.Sidecars))
	for _, sidecar := range node.Sidecars {
		signal := ParseNFO(sidecar.SidecarText)
		if signal.Kind == DirectoryKindUnknown {
			continue
		}
		signals = append(signals, signal)
	}
	return signals
}

func hasNFOConflict(nfoSignals []NFOSignal, folderSignal FolderSignal, videoSignals []VideoSignal) bool {
	if hasMovieNFO(nfoSignals) && (hasAnyEpisodeSignal(videoSignals) || folderSignal.HasSeasonMarker || hasEpisodeNFO(nfoSignals)) {
		return true
	}
	return hasEpisodeNFO(nfoSignals) && !folderSignal.HasSeasonMarker && !hasAnyEpisodeSignal(videoSignals) && classifyMovieVideos(videoSignals) != DirectoryKindUnknown
}

func hasMovieNFO(signals []NFOSignal) bool {
	for _, signal := range signals {
		if signal.Kind == DirectoryKindMovie {
			return true
		}
	}
	return false
}

func hasEpisodeNFO(signals []NFOSignal) bool {
	for _, signal := range signals {
		if signal.Kind == DirectoryKindEpisodeGroup || signal.Kind == DirectoryKindSeason {
			return true
		}
	}
	return false
}

func nfosMatchFolderSeason(signals []NFOSignal, season *int) bool {
	if season == nil {
		return true
	}
	for _, signal := range signals {
		if signal.Season != nil && *signal.Season != *season {
			return false
		}
	}
	return true
}

func collectEpisodeStats(videos []classifiedVideo) episodeStats {
	stats := episodeStats{
		totalCount:            len(videos),
		seasonCounts:          map[int]int{},
		episodeTitleCounts:    map[string]int{},
		episodeIdentityCounts: map[string]int{},
	}
	for _, video := range videos {
		signal := video.signal
		if signal.Episode == nil && !signal.IsSpecial {
			continue
		}
		stats.episodicCount++
		if signal.IsSpecial {
			stats.specialCount++
		}
		if signal.Season != nil {
			stats.seasonCounts[*signal.Season]++
		}
		title := bestTitleCandidate(signal.TitleCandidates)
		if title != "" {
			stats.episodeTitleCounts[title]++
		}
		identity := episodeIdentity(signal)
		if identity != "" {
			stats.episodeIdentityCounts[identity]++
		}
	}
	stats.dominantTitleCount = dominantMapCount(stats.episodeTitleCounts)
	stats.dominantIdentityCount = dominantMapCount(stats.episodeIdentityCounts)
	stats.uniqueEpisodeIdentites = len(stats.episodeIdentityCounts)
	return stats
}

func classifyEpisodeDirectory(folderSignal FolderSignal, stats episodeStats, sampleCount int, node *DirectoryNode) (DirectoryKind, int) {
	if sampleCount == 0 || stats.episodicCount == 0 {
		return DirectoryKindUnknown, 0
	}
	if sampleCount == 1 && !folderSignal.HasSeasonMarker && folderSignal.ExpectedEpisodeCount == nil {
		return DirectoryKindUnknown, 0
	}
	confidence := 0
	if stats.episodicCount*100 >= sampleCount*60 {
		confidence += 3
	}
	if stats.episodicCount*100 >= sampleCount*80 {
		confidence++
	}
	if folderSignal.HasSeasonMarker {
		confidence += 2
	}
	if folderSignal.ExpectedEpisodeCount != nil {
		confidence++
	}
	if stats.dominantTitleCount*100 >= stats.episodicCount*60 {
		confidence++
	}
	if stats.uniqueEpisodeIdentites <= 2 && stats.episodicCount >= 2 {
		confidence++
	}

	season := dominantSeason(stats)
	if folderSignal.HasSeasonMarker {
		if videosMatchFolderSeason(extractVideoSignals(classifiedVideosForNode(node)), folderSignal.Season) {
			return DirectoryKindSeason, confidence + 2
		}
		return DirectoryKindUnknown, 0
	}
	if season != nil && stats.episodicCount >= 2 {
		return DirectoryKindEpisodeGroup, confidence + 1
	}
	if folderSignal.ExpectedEpisodeCount != nil || stats.episodicCount*100 >= sampleCount*70 {
		return DirectoryKindEpisodeGroup, confidence
	}
	return DirectoryKindUnknown, 0
}

func dominantSeason(stats episodeStats) *int {
	bestSeason := 0
	bestCount := 0
	for season, count := range stats.seasonCounts {
		if count > bestCount {
			bestSeason = season
			bestCount = count
		}
	}
	if bestCount == 0 {
		return nil
	}
	season := bestSeason
	return &season
}

func collectMovieStats(videos []classifiedVideo) movieStats {
	stats := movieStats{
		totalCount:     len(videos),
		identityCounts: map[string]int{},
	}
	for _, video := range videos {
		identity := movieSignalIdentity(video.signal)
		if identity == "" {
			continue
		}
		stats.identityCounts[identity]++
	}
	stats.uniqueIdentityCount = len(stats.identityCounts)
	stats.dominantIdentity, stats.dominantCount = dominantMapEntry(stats.identityCounts)
	return stats
}

func classifyMovieDirectory(stats movieStats, sampleCount int) (DirectoryKind, int) {
	if sampleCount == 0 || stats.dominantCount == 0 {
		return DirectoryKindUnknown, 0
	}
	confidence := 0
	identityCoverage := len(stats.identityCounts) * 100 / sampleCount
	if identityCoverage >= 80 {
		confidence += 2
	}
	if stats.dominantCount*100 >= sampleCount*60 {
		confidence += 2
	}
	if stats.dominantCount*100 >= sampleCount*85 {
		confidence++
	}
	if stats.uniqueIdentityCount == 1 {
		if sampleCount > 1 {
			return DirectoryKindMovieVersions, confidence + 1
		}
		return DirectoryKindMovie, confidence + 1
	}
	if stats.uniqueIdentityCount > 1 && stats.dominantCount*100 <= sampleCount*60 {
		return DirectoryKindMovieCollection, confidence + 3
	}
	if stats.dominantCount > 1 {
		return DirectoryKindMovieVersions, confidence
	}
	return DirectoryKindMovieCollection, confidence + 2
}

func classifyMovieVideos(signals []VideoSignal) DirectoryKind {
	identities := map[string]struct{}{}
	for _, signal := range signals {
		identity := movieSignalIdentity(signal)
		if identity == "" {
			return DirectoryKindUnknown
		}
		identities[identity] = struct{}{}
	}
	if len(identities) > 1 {
		return DirectoryKindMovieCollection
	}
	if len(signals) > 1 {
		return DirectoryKindMovieVersions
	}
	return DirectoryKindMovie
}

func movieSignalIdentity(signal VideoSignal) string {
	if len(signal.TitleCandidates) == 0 {
		return ""
	}
	return normalizeMovieIdentity(signal.TitleCandidates[0])
}

func episodeIdentity(signal VideoSignal) string {
	title := bestTitleCandidate(signal.TitleCandidates)
	if title == "" {
		return ""
	}
	return normalizeMovieIdentity(title)
}

func bestTitleCandidate(candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}
	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if containsCJK(candidate) && !containsCJK(best) {
			best = candidate
		}
	}
	return best
}

func normalizeMovieIdentity(title string) string {
	words := strings.Fields(strings.ToLower(title))
	filtered := make([]string, 0, len(words))
	for _, word := range words {
		if isMovieVersionNoiseWord(word) {
			continue
		}
		filtered = append(filtered, word)
	}
	return strings.Join(filtered, " ")
}

func isMovieVersionNoiseWord(word string) bool {
	switch word {
	case "extended", "edition", "ultimate", "uncut", "remastered", "imax", "theatrical", "directors", "cut", "v2", "final":
		return true
	case "dc":
		return true
	}
	return strings.HasPrefix(word, "director")
}

func dominantMapCount(values map[string]int) int {
	_, count := dominantMapEntry(values)
	return count
}

func dominantMapEntry(values map[string]int) (string, int) {
	bestKey := ""
	bestCount := 0
	for key, count := range values {
		if count > bestCount {
			bestKey = key
			bestCount = count
		}
	}
	return bestKey, bestCount
}

func hasAnyEpisodeSignal(signals []VideoSignal) bool {
	for _, signal := range signals {
		if signal.Episode != nil || signal.IsSpecial {
			return true
		}
	}
	return false
}

func videosMatchFolderSeason(signals []VideoSignal, season *int) bool {
	if season == nil {
		return true
	}
	for _, signal := range signals {
		if signal.Season != nil && *signal.Season != *season {
			return false
		}
	}
	return true
}

func isPrimaryVideo(filePath string) bool {
	name := strings.ToLower(filenameStem(filePath))
	for _, token := range []string{"sample", "trailer", "preview", "teaser", "featurette", "extras", "behind the scenes", "幕后", "花絮", "ncop", "nced"} {
		if strings.Contains(name, token) {
			return false
		}
	}
	return true
}

func hasOnlySeriesChildren(node *DirectoryNode) bool {
	if len(node.Children) == 0 {
		return false
	}
	for _, child := range node.Children {
		if child.Kind != DirectoryKindSeason {
			return false
		}
	}
	return true
}
