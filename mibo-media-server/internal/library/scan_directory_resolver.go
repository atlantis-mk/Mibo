package library

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/storage"
)

const (
	directoryShapeMovieFolder       = "movie_folder"
	directoryShapeSeriesFolder      = "series_folder"
	directoryShapeSeasonFolder      = "season_folder"
	directoryShapeFlatEpisodeFolder = "flat_episode_folder"
	directoryShapeMixedFolder       = "mixed_folder"
	directoryShapeUnknown           = "unknown"
)

type directoryShapeDecision struct {
	Shape            string
	VideoCount       int
	EpisodeCount     int
	SeasonNumber     *int
	FallbackEpisodes map[string]int
	Confidence       float64
	Reason           string
}

func resolveDirectoryShape(libraryType string, libraryRoot string, snapshot scanDirectorySnapshot) directoryShapeDecision {
	decision := directoryShapeDecision{Shape: directoryShapeUnknown, Reason: "no media evidence"}
	isTVLibrary := isTVLibraryType(libraryType)
	isMovieLibrary := isMovieLibraryType(libraryType)
	isMixedLibrary := isMixedLibraryType(libraryType)

	seasonNumber := parseSeasonDirectoryNumber(path.Base(snapshot.Path))
	if seasonNumber != nil {
		decision.SeasonNumber = seasonNumber
	}

	hasSeasonChild := false
	hasMovieSidecar := false
	hasTVSidecar := false
	seasonCounts := make(map[int]int)
	fallbackCandidates := make([]string, 0)
	for _, object := range snapshot.Objects {
		if object.IsDir {
			if parseSeasonDirectoryNumber(path.Base(object.Path)) != nil {
				hasSeasonChild = true
			}
			continue
		}
		ext := sidecarExtension(object.Path)
		base := sidecarBaseName(object.Path)
		if ext == ".nfo" || ext == ".json" {
			switch strings.ToLower(strings.TrimSpace(base)) {
			case "movie":
				hasMovieSidecar = true
			case "tvshow", "season":
				hasTVSidecar = true
			}
		}
		if !isVideoFile(object.Path) {
			continue
		}
		decision.VideoCount++
		signals := resolveFilenameSignals(libraryType, libraryRoot, object)
		if signals.IsExtra {
			continue
		}
		fallbackCandidates = append(fallbackCandidates, object.Path)
		if signals.SeasonNumber != nil {
			seasonCounts[*signals.SeasonNumber]++
		}
		if signals.EpisodeNumber != nil || len(signals.EpisodeNumbers) > 0 {
			decision.EpisodeCount++
		}
	}
	if (isTVLibrary || isMixedLibrary) && len(fallbackCandidates) > 0 {
		sort.Strings(fallbackCandidates)
		decision.FallbackEpisodes = make(map[string]int, len(fallbackCandidates))
		for idx, objectPath := range fallbackCandidates {
			decision.FallbackEpisodes[objectPath] = idx + 1
		}
	}
	if decision.SeasonNumber == nil && len(seasonCounts) == 1 {
		for seasonNumber := range seasonCounts {
			value := seasonNumber
			decision.SeasonNumber = &value
		}
	}

	switch {
	case decision.VideoCount == 0 && hasSeasonChild:
		decision.Shape = directoryShapeSeriesFolder
		decision.Confidence = 0.85
		decision.Reason = "directory contains season child directories"
	case hasTVSidecar:
		decision.Shape = directoryShapeSeriesFolder
		decision.Confidence = 0.95
		decision.Reason = "directory contains TV metadata sidecar"
	case seasonNumber != nil && decision.VideoCount > 0:
		decision.Shape = directoryShapeSeasonFolder
		decision.Confidence = 0.9
		decision.Reason = "directory name is a season and contains videos"
	case isTVLibrary && decision.VideoCount > 0 && decision.EpisodeCount > 0:
		decision.Shape = directoryShapeFlatEpisodeFolder
		decision.Confidence = 0.8
		decision.Reason = "TV library directory contains episode-like videos"
	case isTVLibrary && decision.VideoCount > 0 && len(fallbackCandidates) > 0:
		decision.Shape = directoryShapeFlatEpisodeFolder
		decision.Confidence = 0.45
		decision.Reason = "TV library directory contains videos without episode numbers; using sorted non-extra files as episode order"
	case isMixedLibrary && len(fallbackCandidates) == 1:
		decision.Shape = directoryShapeMovieFolder
		decision.Confidence = 0.65
		decision.Reason = "mixed library directory contains one non-extra video"
	case isMixedLibrary && len(fallbackCandidates) > 1:
		decision.Shape = directoryShapeFlatEpisodeFolder
		decision.Confidence = 0.55
		decision.Reason = "mixed library directory contains multiple non-extra videos; using sorted files as episode order"
	case hasMovieSidecar && decision.VideoCount > 0:
		decision.Shape = directoryShapeMovieFolder
		decision.Confidence = 0.95
		decision.Reason = "directory contains movie metadata sidecar"
	case isMovieLibrary && decision.VideoCount > 0 && decision.EpisodeCount == 0:
		decision.Shape = directoryShapeMovieFolder
		decision.Confidence = 0.75
		decision.Reason = "movie library directory contains non-episode videos"
	case decision.VideoCount > 0 && decision.EpisodeCount > 0 && decision.EpisodeCount < decision.VideoCount:
		decision.Shape = directoryShapeMixedFolder
		decision.Confidence = 0.5
		decision.Reason = "directory contains mixed episode and non-episode videos"
	case decision.VideoCount > 0:
		decision.Shape = directoryShapeUnknown
		decision.Confidence = 0.3
		decision.Reason = "directory contains videos without enough grouping evidence"
	}

	return decision
}

func classifyMediaFileWithDirectoryDecision(libraryType string, libraryRoot string, object storage.Object, dirPath string, decision directoryShapeDecision) classifiedMedia {
	classified := classifyMediaFile(libraryType, libraryRoot, object)
	if !isTVLibraryType(libraryType) && !isMixedLibraryType(libraryType) {
		return classified
	}
	if decision.Shape != directoryShapeFlatEpisodeFolder && decision.Shape != directoryShapeSeasonFolder {
		return classified
	}

	signals := resolveFilenameSignals(libraryType, libraryRoot, object)
	episodeNumbers := append([]int(nil), signals.EpisodeNumbers...)
	if len(episodeNumbers) == 0 && signals.EpisodeNumber != nil {
		episodeNumbers = append(episodeNumbers, *signals.EpisodeNumber)
	}
	if signals.EpisodeSource != "explicit" && decision.FallbackEpisodes != nil {
		episodeNumbers = nil
	}
	if len(episodeNumbers) == 0 && decision.FallbackEpisodes != nil {
		if fallbackEpisode, ok := decision.FallbackEpisodes[object.Path]; ok {
			episodeNumbers = append(episodeNumbers, fallbackEpisode)
		}
	}
	if len(episodeNumbers) == 0 {
		return classified
	}

	seasonNumber := signals.SeasonNumber
	if seasonNumber == nil {
		seasonNumber = decision.SeasonNumber
	}
	if seasonNumber == nil && decision.Shape == directoryShapeFlatEpisodeFolder {
		defaultSeason := 1
		seasonNumber = &defaultSeason
	}
	if seasonNumber == nil {
		return classified
	}

	seriesTitle := directorySeriesTitle(libraryRoot, object.Path, dirPath, decision)
	if strings.TrimSpace(seriesTitle) == "" {
		return classified
	}
	firstEpisode := episodeNumbers[0]
	title := fmt.Sprintf("%s S%02dE%02d", seriesTitle, *seasonNumber, firstEpisode)
	if len(episodeNumbers) > 1 {
		title = fmt.Sprintf("%s S%02dE%02d-E%02d", seriesTitle, *seasonNumber, firstEpisode, episodeNumbers[len(episodeNumbers)-1])
	}

	classified.Type = "episode"
	classified.Title = title
	classified.SeriesTitle = seriesTitle
	classified.SeasonNumber = seasonNumber
	classified.EpisodeNumber = &firstEpisode
	classified.EpisodeNumbers = episodeNumbers
	return classified
}

func directorySeriesTitle(libraryRoot string, objectPath string, dirPath string, decision directoryShapeDecision) string {
	if title := tvSeriesTitleFromPath(libraryRoot, objectPath); title != "" {
		return title
	}
	if decision.Shape == directoryShapeSeasonFolder {
		return cleanTitle(path.Base(path.Dir(strings.TrimRight(dirPath, "/"))))
	}
	return cleanTitle(path.Base(strings.TrimRight(dirPath, "/")))
}

func scanDecisionFromDirectoryShape(decision directoryShapeDecision, artifact catalogScanArtifact) scanDecision {
	if decision.Shape == "" || decision.Shape == directoryShapeUnknown {
		return scanDecision{}
	}
	confidence := decision.Confidence
	targetKind := artifact.ItemType
	targetKey := artifact.ItemPath
	decisionType := scanDecisionMovieGroup
	if artifact.ItemType == "episode" {
		targetKind = "series"
		targetKey = artifact.SeriesPath
		decisionType = scanDecisionSeriesGroup
	}
	return scanDecision{Type: decisionType, TargetKind: targetKind, TargetKey: targetKey, Confidence: &confidence, Reason: decision.Reason, CreatedAt: time.Now().UTC()}
}
