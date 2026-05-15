package library

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/scanrecognition"
	"github.com/atlan/mibo-media-server/internal/storage"
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
		if model.VideoSignal.IsExtra {
			if contentShapeShouldSkipAttachment(plan, snapshot, object) {
				assignment.AssignmentType = contentShapeAssignmentSkip
				assignment.ReviewState = "skipped"
				assignment.TargetKey = path.Base(object.Path)
				assignments = append(assignments, assignment)
				continue
			}
			assignment.AssignmentType = contentShapeAssignmentAttachment
			assignment.AssetRole = firstNonEmptyString(model.VideoSignal.Role, plan.AttachmentRole, "extra")
			assignment.TargetKey = plan.SeriesTitle
			assignments = append(assignments, assignment)
			continue
		}
		switch plan.Shape {
		case contentShapeEpisodePack, contentShapeSeasonFolder, contentShapeFlatEpisodeFolder, contentShapeAbsoluteEpisodePack:
			episodeOrder++
			assignment.AssignmentType = contentShapeAssignmentEpisode
			assignment.SeriesTitle = contentShapeEpisodeSeriesTitle(plan, object, model.PathHints)
			assignment.SeasonNumber = firstNonNilInt(model.VideoSignal.Season, model.PathHints.SeasonNumber, plan.SeasonNumber)
			episode := model.VideoSignal.Episode
			if episode == nil {
				rawTitle := strings.TrimSuffix(path.Base(object.Path), path.Ext(object.Path))
				if parsed := scanrecognition.AnalyzeVideoPath(rawTitle + ".mkv").Episode; parsed != nil && scanrecognition.WeakEpisodeNumberAllowed(rawTitle) {
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
			assignment.TargetKey = firstNonEmptyString(plan.MovieWorkKey, normalizeVersionCompareTitle(preferredMovieTitleCandidate(model, true)))
		case contentShapeMovieFolder, contentShapeMovieCollection:
			assignment.AssignmentType = contentShapeAssignmentMovie
			assignment.TargetKey = movieAssignmentTarget(plan, model, true)
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
	if strings.TrimSpace(scanrecognition.VideoFileRoleSignal(object.Path)) == "" {
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
		if !model.VideoSignal.IsExtra {
			return true
		}
	}
	return false
}

func contentShapeEpisodeSeriesTitle(plan contentShapeDirectoryPlan, object storage.Object, pathHints filenamePathHints) string {
	if title := strings.TrimSpace(pathHints.SeriesTitle); title != "" {
		return scanrecognition.PrimarySeriesTitleFromGroup(scanrecognition.NormalizeSeriesGroupingTitle(title))
	}
	if title := scanrecognition.SeriesTitleFromPath("", object.Path); title != "" {
		return scanrecognition.PrimarySeriesTitleFromGroup(scanrecognition.NormalizeSeriesGroupingTitle(title))
	}
	if title := strings.TrimSpace(plan.SeriesTitle); title != "" {
		return scanrecognition.NormalizeSeriesGroupingTitle(title)
	}
	if plan.SeasonNumber != nil {
		return scanrecognition.NormalizeSeriesGroupingTitle(path.Base(path.Dir(path.Dir(object.Path))))
	}
	return scanrecognition.NormalizeSeriesGroupingTitle(path.Base(path.Dir(object.Path)))
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

func movieAssignmentTarget(plan contentShapeDirectoryPlan, model filenameSignalModel, preferFolder bool) string {
	if plan.Shape == contentShapeMovieFolder && plan.MovieWorkKey != "" {
		return plan.MovieWorkKey
	}
	title := normalizeVersionCompareTitle(strings.TrimSpace(preferredMovieTitleCandidate(model, preferFolder)))
	if year := preferredMovieYearCandidate(model, preferFolder); year != nil {
		return fmt.Sprintf("%s:%d", title, *year)
	}
	return title
}

func contentShapeAssignmentForObject(plan contentShapeDirectoryPlan, object storage.Object, tokenCache *filenameTokenProfileCache) contentShapeFileAssignment {
	assignments := generateContentShapeAssignments(plan, scanDirectorySnapshot{Path: path.Dir(object.Path), Objects: []storage.Object{object}}, tokenCache)
	if len(assignments) == 0 {
		return contentShapeFileAssignment{}
	}
	return assignments[0]
}
