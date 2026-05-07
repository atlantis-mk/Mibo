package library

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
	"github.com/atlan/mibo-media-server/internal/titleclean"
)

func generateContentShapeAssignmentsFromPersistedRule(planRecord database.ContentShapePlan, snapshot scanDirectorySnapshot, tokenCache *filenameTokenProfileCache) []contentShapeFileAssignment {
	return generateContentShapeAssignments(contentShapePlanFromRecord(planRecord), snapshot, tokenCache)
}

func contentShapePlanFromRecord(planRecord database.ContentShapePlan) contentShapeDirectoryPlan {
	var rule contentShapePlanRule
	confidence := 0.0
	if planRecord.Confidence != nil {
		confidence = *planRecord.Confidence
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(planRecord.PlanRuleJSON)), &rule); err != nil || strings.TrimSpace(rule.Shape) == "" {
		return contentShapeDirectoryPlan{Shape: planRecord.Shape, Confidence: confidence, ReviewState: planRecord.ReviewState, SeriesTitle: planRecord.SeriesTitle, SeasonNumber: planRecord.SeasonNumber, NumberingMode: planRecord.NumberingMode, MovieWorkKey: ""}
	}
	return contentShapeDirectoryPlan{Shape: rule.Shape, Confidence: confidence, ReviewState: planRecord.ReviewState, SeriesTitle: firstNonEmptyString(rule.SeriesTitle, planRecord.SeriesTitle), SeasonNumber: firstNonNilInt(rule.SeasonNumber, planRecord.SeasonNumber), NumberingMode: firstNonEmptyString(rule.NumberingMode, planRecord.NumberingMode), MovieWorkKey: rule.MovieWorkKey, AttachmentRole: rule.AttachmentRole}
}

func firstNonNilInt(values ...*int) *int {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

const (
	contentShapeAssignmentEpisode    = "episode"
	contentShapeAssignmentMovie      = "movie"
	contentShapeAssignmentVersion    = "movie_version"
	contentShapeAssignmentAttachment = "attachment"
	contentShapeAssignmentReview     = "review_required"
	contentShapeAssignmentSkip       = "skip"
)

type contentShapeFileAssignment struct {
	StoragePath    string
	AssignmentType string
	TargetKey      string
	SeriesTitle    string
	SeasonNumber   *int
	EpisodeNumber  *int
	AbsoluteNumber *int
	AssetRole      string
	Confidence     float64
	ReviewState    string
	Evidence       map[string]any
}

func generateContentShapeAssignments(plan contentShapeDirectoryPlan, snapshot scanDirectorySnapshot, tokenCache *filenameTokenProfileCache) []contentShapeFileAssignment {
	objects := visibleVideoObjects(snapshot.Objects)
	episodeOrder := 0
	assignments := make([]contentShapeFileAssignment, 0, len(objects))
	for _, object := range objects {
		model := filenameTokenProfileForPath(tokenCache, object.Path)
		assignment := contentShapeFileAssignment{StoragePath: strings.TrimSpace(object.Path), Confidence: plan.Confidence, ReviewState: plan.ReviewState, Evidence: map[string]any{"source": "directory_plan_assignment", "plan_shape": plan.Shape, "filename_token_profile": model.Evidence}}
		if model.RoleHints.IsExtra {
			if contentShapeShouldSkipAttachment(plan, snapshot, object) {
				assignment.AssignmentType = contentShapeAssignmentSkip
				assignment.ReviewState = "skipped"
				assignment.TargetKey = path.Base(object.Path)
				assignments = append(assignments, assignment)
				continue
			}
			assignment.AssignmentType = contentShapeAssignmentAttachment
			assignment.AssetRole = firstNonEmptyString(model.RoleHints.Role, plan.AttachmentRole, "extra")
			assignment.TargetKey = plan.SeriesTitle
			assignments = append(assignments, assignment)
			continue
		}
		switch plan.Shape {
		case contentShapeEpisodePack, contentShapeSeasonFolder, contentShapeFlatEpisodeFolder, contentShapeAbsoluteEpisodePack:
			episodeOrder++
			assignment.AssignmentType = contentShapeAssignmentEpisode
			assignment.SeriesTitle = contentShapeEpisodeSeriesTitle(plan, object, model)
			assignment.SeasonNumber = firstNonNilInt(model.Identity.SeasonNumber, model.PathHints.SeasonNumber, plan.SeasonNumber)
			episode := model.Identity.EpisodeNumber
			if episode == nil {
				rawTitle := strings.TrimSuffix(path.Base(object.Path), path.Ext(object.Path))
				if parsed := parseEpisodeNumberFromTitle(rawTitle, assignment.SeriesTitle); parsed != nil && weakEpisodeNumberAllowed(rawTitle) {
					episode = parsed
				}
			}
			if episode == nil && (plan.Shape == contentShapeSeasonFolder || plan.Shape == contentShapeFlatEpisodeFolder || plan.Shape == contentShapeEpisodePack) {
				value := episodeOrder
				episode = &value
			}
			assignment.EpisodeNumber = episode
			if plan.Shape == contentShapeAbsoluteEpisodePack {
				assignment.AbsoluteNumber = episode
			}
			assignment.TargetKey = episodeAssignmentTarget(plan, assignment)
			if episode == nil {
				assignment.AssignmentType = contentShapeAssignmentReview
				assignment.ReviewState = "review_required"
			}
		case contentShapeMovieVersionsFolder:
			assignment.AssignmentType = contentShapeAssignmentVersion
			assignment.TargetKey = firstNonEmptyString(plan.MovieWorkKey, normalizeVersionCompareTitle(model.Identity.TitleCandidate))
		case contentShapeMovieFolder, contentShapeMovieCollection:
			assignment.AssignmentType = contentShapeAssignmentMovie
			assignment.TargetKey = movieAssignmentTarget(plan, model)
		default:
			assignment.AssignmentType = contentShapeAssignmentReview
			assignment.ReviewState = "review_required"
			assignment.TargetKey = path.Base(object.Path)
		}
		assignments = append(assignments, assignment)
	}
	return assignments
}

func contentShapeShouldSkipAttachment(plan contentShapeDirectoryPlan, snapshot scanDirectorySnapshot, object storage.Object) bool {
	if strings.TrimSpace(videoFileRoleSignal(object.Path)) == "" {
		return false
	}
	switch strings.TrimSpace(plan.Shape) {
	case contentShapeMovieFolder, contentShapeMovieVersionsFolder, contentShapeMovieCollection:
		return false
	case contentShapeEpisodePack, contentShapeSeasonFolder, contentShapeFlatEpisodeFolder, contentShapeAbsoluteEpisodePack:
		return true
	case contentShapeUnknownReview:
		return contentShapeHasNonExtraVideoSibling(snapshot, object)
	default:
		return false
	}
}

func contentShapeHasNonExtraVideoSibling(snapshot scanDirectorySnapshot, object storage.Object) bool {
	for _, sibling := range snapshot.Objects {
		if sibling.IsDir || !isVideoFile(sibling.Path) || strings.TrimSpace(sibling.Path) == strings.TrimSpace(object.Path) {
			continue
		}
		model := filenameTokenProfileForPath(nil, sibling.Path)
		if !model.RoleHints.IsExtra {
			return true
		}
	}
	return false
}

func contentShapeEpisodeSeriesTitle(plan contentShapeDirectoryPlan, object storage.Object, model filenameSignalModel) string {
	if title := strings.TrimSpace(model.PathHints.SeriesTitle); title != "" {
		return primarySeriesTitleFromGroup(normalizeSeriesGroupingTitle(title))
	}
	if title := tvSeriesTitleFromPath("", object.Path); title != "" {
		return primarySeriesTitleFromGroup(normalizeSeriesGroupingTitle(title))
	}
	if title := strings.TrimSpace(plan.SeriesTitle); title != "" {
		return normalizeSeriesGroupingTitle(title)
	}
	if plan.SeasonNumber != nil {
		return normalizeSeriesGroupingTitle(path.Base(path.Dir(path.Dir(object.Path))))
	}
	return normalizeSeriesGroupingTitle(path.Base(path.Dir(object.Path)))
}

func contentShapeAssignmentsByPath(assignments []contentShapeFileAssignment) map[string]contentShapeFileAssignment {
	if len(assignments) == 0 {
		return nil
	}
	byPath := make(map[string]contentShapeFileAssignment, len(assignments))
	for _, assignment := range assignments {
		storagePath := strings.TrimSpace(assignment.StoragePath)
		if storagePath == "" {
			continue
		}
		byPath[storagePath] = assignment
	}
	return byPath
}

func contentShapeAssignmentsFromRecords(records []database.ContentShapeAssignment) map[string]contentShapeFileAssignment {
	if len(records) == 0 {
		return nil
	}
	byPath := make(map[string]contentShapeFileAssignment, len(records))
	for _, record := range records {
		confidence := 0.0
		if record.Confidence != nil {
			confidence = *record.Confidence
		}
		byPath[strings.TrimSpace(record.StoragePath)] = contentShapeFileAssignment{
			StoragePath:    strings.TrimSpace(record.StoragePath),
			AssignmentType: strings.TrimSpace(record.AssignmentType),
			TargetKey:      strings.TrimSpace(record.TargetKey),
			SeriesTitle:    strings.TrimSpace(record.SeriesTitle),
			SeasonNumber:   record.SeasonNumber,
			EpisodeNumber:  record.EpisodeNumber,
			AbsoluteNumber: record.AbsoluteNumber,
			AssetRole:      strings.TrimSpace(record.AssetRole),
			Confidence:     confidence,
			ReviewState:    strings.TrimSpace(record.ReviewState),
		}
	}
	return byPath
}

func contentShapeVisibleVideoPaths(snapshot scanDirectorySnapshot) []string {
	objects := visibleVideoObjects(snapshot.Objects)
	paths := make([]string, 0, len(objects))
	for _, object := range objects {
		trimmed := strings.TrimSpace(object.Path)
		if trimmed == "" {
			continue
		}
		paths = append(paths, trimmed)
	}
	return paths
}

func classifiedMediaFromContentShapeAssignment(plan contentShapeDirectoryPlan, assignment contentShapeFileAssignment, object storage.Object, tokenCache *filenameTokenProfileCache) (classifiedMedia, bool) {
	model := filenameTokenProfileForPath(tokenCache, object.Path)
	normalized := titleclean.Normalize(titleclean.NormalizeInput{RawTitle: strings.TrimSuffix(path.Base(object.Path), path.Ext(object.Path))})
	if assignment.AssignmentType == contentShapeAssignmentSkip {
		return classifiedMedia{}, false
	}
	if (plan.Confidence < contentShapeHighConfidenceThreshold || plan.ReviewState == "review_required") && assignment.AssignmentType != contentShapeAssignmentEpisode {
		title := contentShapeMovieTitle(plan, assignment, object, model)
		if strings.TrimSpace(title) == "" {
			title = strings.TrimSuffix(path.Base(object.Path), path.Ext(object.Path))
		}
		return classifiedMedia{Type: "movie", Title: title, OriginalTitle: strings.TrimSuffix(path.Base(object.Path), path.Ext(object.Path)), Year: model.Identity.Year, SourcePath: object.Path, Status: "ready", NormalizationVersion: normalized.NormalizationVersion, RemovedTokens: normalized.RemovedTokens, Tags: normalized.Tags, FilenameSignals: model}, true
	}
	switch assignment.AssignmentType {
	case contentShapeAssignmentEpisode:
		if assignment.EpisodeNumber == nil {
			return classifiedMedia{}, false
		}
		seasonNumber := assignment.SeasonNumber
		if seasonNumber == nil {
			defaultSeason := 1
			seasonNumber = &defaultSeason
		}
		seriesTitle := firstNonEmptyString(assignment.SeriesTitle, plan.SeriesTitle, cleanTitle(path.Base(path.Dir(object.Path))))
		if strings.TrimSpace(seriesTitle) == "" {
			return classifiedMedia{}, false
		}
		episodeNumber := *assignment.EpisodeNumber
		title, episodeNumbers := contentShapeEpisodeTitleAndNumbers(seriesTitle, *seasonNumber, episodeNumber, model)
		return classifiedMedia{Type: "episode", Title: title, OriginalTitle: strings.TrimSuffix(path.Base(object.Path), path.Ext(object.Path)), SeriesTitle: seriesTitle, SeasonNumber: seasonNumber, EpisodeNumber: &episodeNumber, EpisodeNumbers: episodeNumbers, SourcePath: object.Path, Status: "ready", NormalizationVersion: normalized.NormalizationVersion, RemovedTokens: normalized.RemovedTokens, Tags: normalized.Tags, FilenameSignals: model}, true
	case contentShapeAssignmentMovie, contentShapeAssignmentVersion, contentShapeAssignmentAttachment, contentShapeAssignmentReview:
		title := contentShapeMovieTitle(plan, assignment, object, model)
		if strings.TrimSpace(title) == "" {
			title = strings.TrimSuffix(path.Base(object.Path), path.Ext(object.Path))
		}
		return classifiedMedia{Type: "movie", Title: title, OriginalTitle: strings.TrimSuffix(path.Base(object.Path), path.Ext(object.Path)), Year: model.Identity.Year, SourcePath: object.Path, Status: "ready", NormalizationVersion: normalized.NormalizationVersion, RemovedTokens: normalized.RemovedTokens, Tags: normalized.Tags, FilenameSignals: model}, true
	default:
		return classifiedMediaFromContentShapePlan(plan, object, tokenCache)
	}
}

func visibleVideoObjects(objects []storage.Object) []storage.Object {
	videos := make([]storage.Object, 0, len(objects))
	for _, object := range objects {
		if !object.IsDir && isVideoFile(object.Path) {
			videos = append(videos, object)
		}
	}
	sort.Slice(videos, func(i, j int) bool { return strings.TrimSpace(videos[i].Path) < strings.TrimSpace(videos[j].Path) })
	return videos
}

func episodeAssignmentTarget(plan contentShapeDirectoryPlan, assignment contentShapeFileAssignment) string {
	if assignment.AbsoluteNumber != nil {
		return fmt.Sprintf("%s:absolute:%04d", plan.SeriesTitle, *assignment.AbsoluteNumber)
	}
	season := 1
	if assignment.SeasonNumber != nil {
		season = *assignment.SeasonNumber
	}
	if assignment.EpisodeNumber == nil {
		return strings.TrimSpace(plan.SeriesTitle)
	}
	return fmt.Sprintf("%s:s%02d:e%04d", plan.SeriesTitle, season, *assignment.EpisodeNumber)
}

func movieAssignmentTarget(plan contentShapeDirectoryPlan, model filenameSignalModel) string {
	if plan.Shape == contentShapeMovieFolder && plan.MovieWorkKey != "" {
		return plan.MovieWorkKey
	}
	title := normalizeVersionCompareTitle(model.Identity.TitleCandidate)
	if model.Identity.Year != nil {
		return fmt.Sprintf("%s:%d", title, *model.Identity.Year)
	}
	return title
}

func classifiedMediaFromContentShapePlan(plan contentShapeDirectoryPlan, object storage.Object, tokenCache *filenameTokenProfileCache) (classifiedMedia, bool) {
	if strings.TrimSpace(plan.Shape) == "" {
		return classifiedMedia{}, false
	}
	model := filenameTokenProfileForPath(tokenCache, object.Path)
	if plan.Confidence < contentShapeHighConfidenceThreshold || plan.ReviewState == "review_required" {
		title := contentShapeMovieTitle(plan, contentShapeFileAssignment{}, object, model)
		if strings.TrimSpace(title) == "" {
			title = strings.TrimSuffix(path.Base(object.Path), path.Ext(object.Path))
		}
		return classifiedMedia{Type: "movie", Title: title, OriginalTitle: strings.TrimSuffix(path.Base(object.Path), path.Ext(object.Path)), Year: model.Identity.Year, SourcePath: object.Path, Status: "ready", FilenameSignals: model}, true
	}
	if plan.Shape == contentShapeMovieVersionsFolder || plan.Shape == contentShapeMovieCollection {
		title := contentShapeMovieTitle(plan, contentShapeFileAssignment{}, object, model)
		return classifiedMedia{Type: "movie", Title: title, OriginalTitle: strings.TrimSuffix(path.Base(object.Path), path.Ext(object.Path)), Year: model.Identity.Year, SourcePath: object.Path, Status: "ready", FilenameSignals: model}, strings.TrimSpace(title) != ""
	}
	if plan.Shape == contentShapeAttachmentGroup {
		title := firstNonEmptyString(plan.SeriesTitle, cleanTitle(plan.MovieWorkKey), cleanTitle(path.Base(path.Dir(object.Path))))
		if strings.TrimSpace(title) == "" {
			return classifiedMedia{}, false
		}
		return classifiedMedia{Type: "movie", Title: title, OriginalTitle: strings.TrimSuffix(path.Base(object.Path), path.Ext(object.Path)), SourcePath: object.Path, Status: "ready", FilenameSignals: model}, true
	}
	if plan.Shape != contentShapeEpisodePack && plan.Shape != contentShapeSeasonFolder && plan.Shape != contentShapeFlatEpisodeFolder && plan.Shape != contentShapeAbsoluteEpisodePack {
		return classifiedMedia{}, false
	}
	assignment := contentShapeAssignmentForObject(plan, object, tokenCache)
	if assignment.AssignmentType != contentShapeAssignmentEpisode || assignment.EpisodeNumber == nil {
		return classifiedMedia{}, false
	}
	seasonNumber := assignment.SeasonNumber
	if seasonNumber == nil {
		defaultSeason := 1
		seasonNumber = &defaultSeason
	}
	seriesTitle := firstNonEmptyString(assignment.SeriesTitle, plan.SeriesTitle, cleanTitle(path.Base(path.Dir(object.Path))))
	if strings.TrimSpace(seriesTitle) == "" {
		return classifiedMedia{}, false
	}
	episodeNumber := *assignment.EpisodeNumber
	title, episodeNumbers := contentShapeEpisodeTitleAndNumbers(seriesTitle, *seasonNumber, episodeNumber, model)
	return classifiedMedia{Type: "episode", Title: title, OriginalTitle: strings.TrimSuffix(path.Base(object.Path), path.Ext(object.Path)), SeriesTitle: seriesTitle, SeasonNumber: seasonNumber, EpisodeNumber: &episodeNumber, EpisodeNumbers: episodeNumbers, SourcePath: object.Path, Status: "ready", FilenameSignals: model}, true
}

func contentShapeMovieTitle(plan contentShapeDirectoryPlan, assignment contentShapeFileAssignment, object storage.Object, model filenameSignalModel) string {
	rawTitle := strings.TrimSuffix(path.Base(object.Path), path.Ext(object.Path))
	if strings.TrimSpace(model.ReleaseHints.GenericNoise) != "" || isGenericMediaName(rawTitle) {
		if parent := cleanTitle(path.Base(path.Dir(object.Path))); strings.TrimSpace(parent) != "" {
			return parent
		}
	}
	return firstNonEmptyString(model.Identity.TitleCandidate, assignment.SeriesTitle, plan.SeriesTitle, cleanTitle(plan.MovieWorkKey), cleanTitle(path.Base(path.Dir(object.Path))))
}

func contentShapeEpisodeTitleAndNumbers(seriesTitle string, seasonNumber int, episodeNumber int, model filenameSignalModel) (string, []int) {
	episodeNumbers := []int{episodeNumber}
	if len(model.Identity.EpisodeNumbers) > 0 && model.Identity.EpisodeNumber != nil && *model.Identity.EpisodeNumber == episodeNumber {
		episodeNumbers = append([]int(nil), model.Identity.EpisodeNumbers...)
	}
	title := fmt.Sprintf("%s S%02dE%02d", seriesTitle, seasonNumber, episodeNumber)
	if len(episodeNumbers) > 1 {
		title = fmt.Sprintf("%s S%02dE%02d-E%02d", seriesTitle, seasonNumber, episodeNumbers[0], episodeNumbers[len(episodeNumbers)-1])
	}
	return title, episodeNumbers
}

func contentShapeAssignmentForObject(plan contentShapeDirectoryPlan, object storage.Object, tokenCache *filenameTokenProfileCache) contentShapeFileAssignment {
	assignments := generateContentShapeAssignments(plan, scanDirectorySnapshot{Path: path.Dir(object.Path), Objects: []storage.Object{object}}, tokenCache)
	if len(assignments) == 0 {
		return contentShapeFileAssignment{}
	}
	return assignments[0]
}
