package library

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/scanrecognition"
)

const (
	contentShapeEpisodePack         = "episode_pack"
	contentShapeAbsoluteEpisodePack = "absolute_episode_pack"
	contentShapeSeasonFolder        = "season_folder"
	contentShapeFlatEpisodeFolder   = "flat_episode_folder"
	contentShapeSeriesFolder        = "series_folder"
	contentShapeMovieFolder         = "movie_folder"
	contentShapeMovieVersionsFolder = "movie_versions_folder"
	contentShapeMovieCollection     = "movie_collection_folder"
	contentShapeAttachmentGroup     = "attachment_group"
	contentShapeUnknownReview       = "unknown_review"
)

type contentShapeDirectoryPlan struct {
	Shape          string
	Confidence     float64
	ReviewState    string
	SeriesTitle    string
	SeasonNumber   *int
	NumberingMode  string
	MovieWorkKey   string
	AttachmentRole string
	Evidence       map[string]any
	Alternatives   []contentShapePlanAlternative
}

type contentShapePlanRule struct {
	Shape          string         `json:"shape"`
	NumberingMode  string         `json:"numbering_mode,omitempty"`
	SeriesTitle    string         `json:"series_title,omitempty"`
	SeasonNumber   *int           `json:"season_number,omitempty"`
	MovieWorkKey   string         `json:"movie_work_key,omitempty"`
	AttachmentRole string         `json:"attachment_role,omitempty"`
	Rule           map[string]any `json:"rule,omitempty"`
}

type contentShapePlanAlternative struct {
	Shape      string
	Confidence float64
	Reason     string
}

func compileContentShapePlan(profile contentShapeDirectoryProfile) contentShapeDirectoryPlan {
	attachmentOnly := profile.VideoCount > 0 && profile.AttachmentCount == profile.VideoCount
	if attachmentOnly {
		plan := contentShapePlan(profile, contentShapeAttachmentGroup, 0.9, "auto")
		plan.AttachmentRole = "extra"
		return plan
	}
	if profile.VideoCount == 0 {
		if hasContentShapeHint(profile, "season") {
			return contentShapePlan(profile, contentShapeSeriesFolder, 0.8, "auto")
		}
		return contentShapePlan(profile, contentShapeUnknownReview, 0.2, "review_required")
	}
	if profile.SeasonHint != nil && profile.ExplicitEpisodeCount > 0 {
		plan := contentShapePlan(profile, contentShapeSeasonFolder, 0.82, "auto")
		plan.SeasonNumber = profile.SeasonHint
		plan.NumberingMode = "season_episode"
		return plan
	}
	if profile.NonExtraVideoCount == 1 {
		plan := contentShapePlan(profile, contentShapeMovieFolder, 0.75, "auto")
		plan.MovieWorkKey = profile.CommonTitleStem
		return plan
	}
	episodeScore := contentShapeEpisodePackScore(profile)
	movieCollectionScore := contentShapeMovieCollectionScore(profile)
	movieVersionScore := contentShapeMovieVersionScore(profile)
	if episodeScore >= 0.45 && movieCollectionScore >= 0.45 && absFloat64(episodeScore-movieCollectionScore) < 0.15 {
		plan := contentShapePlan(profile, contentShapeUnknownReview, maxFloat64(episodeScore, movieCollectionScore), "review_required")
		plan.Alternatives = append(plan.Alternatives, contentShapePlanAlternative{Shape: contentShapeAbsoluteEpisodePack, Confidence: episodeScore, Reason: "episode sequence evidence competes with independent movie evidence"})
		plan.Alternatives = append(plan.Alternatives, contentShapePlanAlternative{Shape: contentShapeMovieCollection, Confidence: movieCollectionScore, Reason: "movie title/year uniqueness competes with episode sequence evidence"})
		return plan
	}
	if profile.SeasonHint != nil && episodeScore >= 0.6 {
		plan := contentShapePlan(profile, contentShapeSeasonFolder, maxFloat64(0.75, profile.SequenceCoverage), "auto")
		plan.SeasonNumber = profile.SeasonHint
		plan.NumberingMode = "season_episode"
		return plan
	}
	if episodeScore >= 0.82 && movieCollectionScore < 0.5 {
		plan := contentShapePlan(profile, contentShapeAbsoluteEpisodePack, episodeScore, "auto")
		plan.NumberingMode = "absolute"
		return plan
	}
	if profile.ExplicitEpisodeCount > 0 && episodeScore >= 0.65 {
		plan := contentShapePlan(profile, contentShapeEpisodePack, episodeScore, "auto")
		plan.NumberingMode = "episode"
		return plan
	}
	if profile.LeadingNumericCount > 0 && episodeScore >= 0.55 && movieCollectionScore < 0.45 {
		plan := contentShapePlan(profile, contentShapeFlatEpisodeFolder, episodeScore, "auto")
		plan.NumberingMode = "sorted_or_numeric"
		return plan
	}
	if shouldPlanFlatEpisodeFolder(profile, movieCollectionScore) {
		plan := contentShapePlan(profile, contentShapeFlatEpisodeFolder, maxFloat64(0.7, episodeScore), "auto")
		plan.NumberingMode = "sorted_or_numeric"
		return plan
	}
	if movieVersionScore >= 0.7 {
		plan := contentShapePlan(profile, contentShapeMovieVersionsFolder, movieVersionScore, "auto")
		plan.MovieWorkKey = profile.CommonTitleStem
		return plan
	}
	if movieCollectionScore >= 0.65 && episodeScore < 0.55 {
		plan := contentShapePlan(profile, contentShapeMovieCollection, movieCollectionScore, "auto")
		plan.MovieWorkKey = "per-title-year"
		return plan
	}
	if hasContentShapeHint(profile, "season") {
		return contentShapePlan(profile, contentShapeSeriesFolder, 0.65, "auto")
	}
	plan := contentShapePlan(profile, contentShapeUnknownReview, maxFloat64(0.25, movieCollectionScore), "review_required")
	plan.MovieWorkKey = "per-file-title"
	plan.Alternatives = append(plan.Alternatives, contentShapePlanAlternative{Shape: contentShapeMovieCollection, Confidence: maxFloat64(0.75, movieCollectionScore), Reason: "independent per-file movies if manual review accepts this folder"})
	return plan
}

func shouldPlanFlatEpisodeFolder(profile contentShapeDirectoryProfile, movieCollectionScore float64) bool {
	if profile.YearDensity != 0 || profile.NonExtraVideoCount < 2 || profile.VersionEvidenceCount != 0 || profile.SidecarHintCount != 0 || strings.EqualFold(profile.CommonTitleStem, "alpha") {
		return false
	}
	if profile.ExplicitEpisodeCount > 0 && profile.SequenceCoverage >= 0.5 {
		return true
	}
	if profile.TitleUniqueness < 0.5 {
		return false
	}
	if movieCollectionScore >= 0.45 {
		return false
	}
	if strings.TrimSpace(profile.RootPath) != "" && strings.TrimSpace(profile.Path) != "" {
		relative := scanrecognition.RelativePathSegments(profile.RootPath, profile.Path)
		if len(relative) <= 1 {
			return true
		}
		return false
	}
	return true
}

func contentShapeEpisodePackScore(profile contentShapeDirectoryProfile) float64 {
	if profile.NonExtraVideoCount == 0 {
		return 0
	}
	score := profile.SequenceCoverage*0.6 + ratio(profile.ExplicitEpisodeCount+profile.LeadingNumericCount, profile.NonExtraVideoCount)*0.25
	if profile.SeasonHint != nil || hasContentShapeHint(profile, "season") {
		score += 0.1
	}
	if profile.CommonTitleStem != "" && profile.TitleUniqueness <= 0.25 {
		score += 0.05
	}
	score -= profile.YearDensity * 0.35
	if !(profile.SequenceCoverage >= 0.9 && profile.YearDensity == 0) {
		score -= profile.TitleUniqueness * 0.2
	}
	return clamp01(score)
}

func contentShapeMovieCollectionScore(profile contentShapeDirectoryProfile) float64 {
	score := profile.YearDensity*0.45 + profile.TitleUniqueness*0.4
	if profile.TitleYearCount >= 2 {
		score += 0.1
	}
	score -= profile.SequenceCoverage * 0.35
	if profile.SeasonHint != nil || hasContentShapeHint(profile, "season") {
		score -= 0.15
	}
	return clamp01(score)
}

func contentShapeMovieVersionScore(profile contentShapeDirectoryProfile) float64 {
	if profile.NonExtraVideoCount < 2 {
		return 0
	}
	score := ratio(profile.VersionEvidenceCount, profile.NonExtraVideoCount)*0.65 + (1-profile.TitleUniqueness)*0.25
	score -= profile.SequenceCoverage * 0.2
	return clamp01(score)
}

func ratio(numerator int, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func absFloat64(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func contentShapePlan(profile contentShapeDirectoryProfile, shape string, confidence float64, reviewState string) contentShapeDirectoryPlan {
	plan := contentShapeDirectoryPlan{Shape: shape, Confidence: confidence, ReviewState: reviewState, SeriesTitle: contentShapeSeriesTitle(profile), Evidence: contentShapePlanEvidence(profile)}
	if shape != contentShapeUnknownReview {
		plan.Alternatives = append(plan.Alternatives, contentShapePlanAlternative{Shape: contentShapeUnknownReview, Confidence: 1 - confidence, Reason: "requires review when automatic shape evidence is insufficient"})
	}
	return plan
}

func contentShapeSeriesTitle(profile contentShapeDirectoryProfile) string {
	if profile.CommonTitleStem == "" {
		return ""
	}
	return scanrecognition.CleanTitle(profile.CommonTitleStem)
}

func contentShapeDatabasePlan(scope contentShapeScope, profileID uint, plan contentShapeDirectoryPlan) database.ContentShapePlan {
	now := time.Now().UTC()
	confidence := plan.Confidence
	planRuleJSON, _ := json.Marshal(contentShapePlanRule{Shape: plan.Shape, NumberingMode: plan.NumberingMode, SeriesTitle: plan.SeriesTitle, SeasonNumber: plan.SeasonNumber, MovieWorkKey: plan.MovieWorkKey, AttachmentRole: plan.AttachmentRole, Rule: contentShapeCompactRule(plan)})
	evidenceJSON, _ := json.Marshal(plan.Evidence)
	alternativesJSON, _ := json.Marshal(plan.Alternatives)
	return database.ContentShapePlan{ProfileID: profileID, LibraryID: scope.LibraryID, MediaSourceID: scope.MediaSourceID, LibraryPathID: scope.LibraryPathID, StorageProvider: strings.TrimSpace(scope.StorageProvider), RootPath: strings.TrimSpace(scope.RootPath), DirectoryPath: strings.TrimSpace(scope.DirectoryPath), ClassifierVersion: strings.TrimSpace(scope.ClassifierVersion), Fingerprint: strings.TrimSpace(scope.Fingerprint), Shape: strings.TrimSpace(plan.Shape), Confidence: &confidence, ReviewState: strings.TrimSpace(plan.ReviewState), SeriesTitle: strings.TrimSpace(plan.SeriesTitle), SeasonNumber: plan.SeasonNumber, NumberingMode: strings.TrimSpace(plan.NumberingMode), PlanRuleJSON: string(planRuleJSON), EvidenceJSON: string(evidenceJSON), AlternativesJSON: string(alternativesJSON), LastObservedAt: now}
}

func contentShapeCompactRule(plan contentShapeDirectoryPlan) map[string]any {
	switch plan.Shape {
	case contentShapeAbsoluteEpisodePack:
		return map[string]any{"assignment": "episode_number_from_absolute_token"}
	case contentShapeSeasonFolder, contentShapeEpisodePack:
		return map[string]any{"assignment": "episode_number_from_filename", "season_number": plan.SeasonNumber}
	case contentShapeFlatEpisodeFolder:
		return map[string]any{"assignment": "episode_number_from_filename_or_sorted_order"}
	case contentShapeMovieVersionsFolder:
		return map[string]any{"assignment": "same_movie_work_by_common_stem", "movie_work_key": plan.MovieWorkKey}
	case contentShapeMovieCollection:
		return map[string]any{"assignment": "movie_work_by_title_year"}
	case contentShapeAttachmentGroup:
		return map[string]any{"assignment": "attachment_role", "role": plan.AttachmentRole}
	default:
		return map[string]any{"assignment": "review_required"}
	}
}

func contentShapePlanReuseDecision(existing database.ContentShapePlan, nextProfile contentShapeDirectoryProfile, settings contentShapeSettings) (bool, string) {
	if existing.DeletedScope || existing.InvalidatedAt != nil {
		return false, "plan invalidated or deleted"
	}
	if strings.TrimSpace(existing.Shape) == contentShapeUnknownReview || strings.TrimSpace(existing.ReviewState) == "review_required" {
		return false, "review-required plans are not automatically reused"
	}
	if existing.Confidence == nil || *existing.Confidence < settings.PlanReuseConfidenceThreshold {
		return false, "plan confidence below reuse threshold"
	}
	nextPlan := compileContentShapePlan(nextProfile)
	if nextPlan.Shape != existing.Shape {
		return false, "directory delta changed planned shape"
	}
	if nextPlan.Confidence < settings.MediumReviewConfidenceThreshold {
		return false, "directory delta lowered confidence to review range"
	}
	if nextProfile.YearDensity > 0.35 && (existing.Shape == contentShapeAbsoluteEpisodePack || existing.Shape == contentShapeEpisodePack || existing.Shape == contentShapeFlatEpisodeFolder || existing.Shape == contentShapeSeasonFolder) {
		return false, "movie-like evidence conflicts with episode plan"
	}
	return true, "plan rule remains valid for directory delta"
}

func contentShapePlanEvidence(profile contentShapeDirectoryProfile) map[string]any {
	return map[string]any{
		"source":                 "directory_profile",
		"video_count":            profile.VideoCount,
		"non_extra_video_count":  profile.NonExtraVideoCount,
		"attachment_count":       profile.AttachmentCount,
		"explicit_episode_count": profile.ExplicitEpisodeCount,
		"leading_numeric_count":  profile.LeadingNumericCount,
		"sequence_coverage":      profile.SequenceCoverage,
		"sequence_gaps":          profile.SequenceGaps,
		"year_density":           profile.YearDensity,
		"title_uniqueness":       profile.TitleUniqueness,
		"common_title_stem":      profile.CommonTitleStem,
		"version_evidence_count": profile.VersionEvidenceCount,
		"sidecar_hint_count":     profile.SidecarHintCount,
		"category_path_hints":    profile.CategoryPathHints,
	}
}

func contentShapePlanDebugPayload(plan contentShapeDirectoryPlan) map[string]any {
	if strings.TrimSpace(plan.Shape) == "" {
		return nil
	}
	return map[string]any{"shape": plan.Shape, "confidence": plan.Confidence, "review_state": plan.ReviewState, "series_title": plan.SeriesTitle, "numbering_mode": plan.NumberingMode, "movie_work_key": plan.MovieWorkKey, "alternatives": plan.Alternatives}
}

func hasContentShapeHint(profile contentShapeDirectoryProfile, hint string) bool {
	for _, value := range profile.CategoryPathHints {
		if strings.EqualFold(value, hint) {
			return true
		}
	}
	return false
}

func maxFloat64(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}
