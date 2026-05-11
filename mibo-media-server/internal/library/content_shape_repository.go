package library

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type contentShapeScope struct {
	LibraryID         uint
	MediaSourceID     uint
	LibraryPathID     *uint
	StorageProvider   string
	RootPath          string
	DirectoryPath     string
	ClassifierVersion string
	Fingerprint       string
}

func contentShapeScopeFromProfile(profile database.ContentShapeProfile) contentShapeScope {
	return contentShapeScope{LibraryID: profile.LibraryID, MediaSourceID: profile.MediaSourceID, LibraryPathID: profile.LibraryPathID, StorageProvider: profile.StorageProvider, RootPath: profile.RootPath, DirectoryPath: profile.DirectoryPath, ClassifierVersion: profile.ClassifierVersion, Fingerprint: profile.Fingerprint}
}

func contentShapeScopeFromPlan(plan database.ContentShapePlan) contentShapeScope {
	return contentShapeScope{LibraryID: plan.LibraryID, MediaSourceID: plan.MediaSourceID, LibraryPathID: plan.LibraryPathID, StorageProvider: plan.StorageProvider, RootPath: plan.RootPath, DirectoryPath: plan.DirectoryPath, ClassifierVersion: plan.ClassifierVersion, Fingerprint: plan.Fingerprint}
}

func loadReusableContentShapeProfile(ctx context.Context, db *gorm.DB, scope contentShapeScope) (database.ContentShapeProfile, bool, error) {
	var profile database.ContentShapeProfile
	err := db.WithContext(ctx).
		Where("library_id = ? AND storage_provider = ? AND root_path = ? AND directory_path = ? AND classifier_version = ? AND fingerprint = ? AND deleted_scope = ? AND invalidated_at IS NULL", scope.LibraryID, strings.TrimSpace(scope.StorageProvider), strings.TrimSpace(scope.RootPath), strings.TrimSpace(scope.DirectoryPath), strings.TrimSpace(scope.ClassifierVersion), strings.TrimSpace(scope.Fingerprint), false).
		First(&profile).Error
	if err == nil {
		return profile, true, nil
	}
	if err == gorm.ErrRecordNotFound {
		return database.ContentShapeProfile{}, false, nil
	}
	return database.ContentShapeProfile{}, false, err
}

func loadReusableContentShapePlan(ctx context.Context, db *gorm.DB, scope contentShapeScope) (database.ContentShapePlan, bool, error) {
	var plan database.ContentShapePlan
	err := db.WithContext(ctx).
		Where("library_id = ? AND storage_provider = ? AND root_path = ? AND directory_path = ? AND classifier_version = ? AND fingerprint = ? AND deleted_scope = ? AND invalidated_at IS NULL", scope.LibraryID, strings.TrimSpace(scope.StorageProvider), strings.TrimSpace(scope.RootPath), strings.TrimSpace(scope.DirectoryPath), strings.TrimSpace(scope.ClassifierVersion), strings.TrimSpace(scope.Fingerprint), false).
		First(&plan).Error
	if err == nil {
		return plan, true, nil
	}
	if err == gorm.ErrRecordNotFound {
		return database.ContentShapePlan{}, false, nil
	}
	return database.ContentShapePlan{}, false, err
}

func saveContentShapeProfile(ctx context.Context, db *gorm.DB, profile *database.ContentShapeProfile) error {
	if profile == nil {
		return nil
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "library_id"}, {Name: "storage_provider"}, {Name: "root_path"}, {Name: "directory_path"}, {Name: "classifier_version"}},
		DoUpdates: clause.AssignmentColumns([]string{"media_source_id", "library_path_id", "fingerprint", "video_count", "non_extra_video_count", "attachment_count", "explicit_episode_count", "leading_numeric_count", "sequence_coverage", "year_density", "title_uniqueness", "common_title_stem", "season_hint", "sidecar_hints_json", "confidence", "review_state", "evidence_json", "deleted_scope", "invalidated_at", "last_observed_at", "updated_at"}),
	}).Create(profile).Error
}

func saveContentShapePlan(ctx context.Context, db *gorm.DB, plan *database.ContentShapePlan) error {
	if plan == nil {
		return nil
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "library_id"}, {Name: "storage_provider"}, {Name: "root_path"}, {Name: "directory_path"}, {Name: "classifier_version"}},
		DoUpdates: clause.AssignmentColumns([]string{"profile_id", "media_source_id", "library_path_id", "fingerprint", "shape", "confidence", "review_state", "series_title", "season_number", "numbering_mode", "plan_rule_json", "exceptions_json", "evidence_json", "alternatives_json", "deleted_scope", "invalidated_at", "last_observed_at", "updated_at"}),
	}).Create(plan).Error
}

func saveContentShapeAssignments(ctx context.Context, db *gorm.DB, scope contentShapeScope, profileID uint, planID uint, assignments []contentShapeFileAssignment) error {
	if len(assignments) == 0 {
		return nil
	}
	rows := make([]database.ContentShapeAssignment, 0, len(assignments))
	for _, assignment := range assignments {
		confidence := assignment.Confidence
		evidenceJSON := mustJSON(assignment.Evidence)
		rows = append(rows, database.ContentShapeAssignment{PlanID: planID, ProfileID: profileID, LibraryID: scope.LibraryID, MediaSourceID: scope.MediaSourceID, LibraryPathID: scope.LibraryPathID, StorageProvider: strings.TrimSpace(scope.StorageProvider), RootPath: strings.TrimSpace(scope.RootPath), DirectoryPath: strings.TrimSpace(scope.DirectoryPath), StoragePath: strings.TrimSpace(assignment.StoragePath), ClassifierVersion: strings.TrimSpace(scope.ClassifierVersion), AssignmentType: strings.TrimSpace(assignment.AssignmentType), TargetKey: strings.TrimSpace(assignment.TargetKey), SeriesTitle: strings.TrimSpace(assignment.SeriesTitle), SeasonNumber: assignment.SeasonNumber, EpisodeNumber: assignment.EpisodeNumber, AbsoluteNumber: assignment.AbsoluteNumber, AssetRole: strings.TrimSpace(assignment.AssetRole), Confidence: &confidence, ReviewState: strings.TrimSpace(assignment.ReviewState), EvidenceJSON: evidenceJSON})
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "storage_provider"}, {Name: "storage_path"}},
		DoUpdates: clause.AssignmentColumns([]string{"plan_id", "profile_id", "library_id", "media_source_id", "library_path_id", "root_path", "directory_path", "classifier_version", "assignment_type", "target_key", "series_title", "season_number", "episode_number", "absolute_number", "asset_role", "confidence", "review_state", "evidence_json", "deleted_scope", "invalidated_at", "updated_at"}),
	}).CreateInBatches(&rows, 100).Error
}

func loadReusableContentShapeAssignments(ctx context.Context, db *gorm.DB, planID uint, storagePaths []string) ([]database.ContentShapeAssignment, error) {
	if planID == 0 || len(storagePaths) == 0 {
		return nil, nil
	}
	paths := make([]string, 0, len(storagePaths))
	seen := make(map[string]struct{}, len(storagePaths))
	for _, storagePath := range storagePaths {
		trimmed := strings.TrimSpace(storagePath)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		paths = append(paths, trimmed)
	}
	if len(paths) == 0 {
		return nil, nil
	}
	var rows []database.ContentShapeAssignment
	if err := db.WithContext(ctx).
		Where("plan_id = ? AND deleted_scope = ? AND invalidated_at IS NULL AND storage_path IN ?", planID, false, paths).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func saveContentShapeReviewDecision(ctx context.Context, db *gorm.DB, scope contentShapeScope, plan contentShapeDirectoryPlan, assignments []contentShapeFileAssignment) error {
	if plan.ReviewState != "review_required" {
		return nil
	}
	confidence := plan.Confidence
	affected := make([]string, 0, len(assignments))
	for _, assignment := range assignments {
		if strings.TrimSpace(assignment.StoragePath) != "" {
			affected = append(affected, assignment.StoragePath)
		}
	}
	return db.WithContext(ctx).Create(&database.ClassificationDecision{LibraryID: scope.LibraryID, SourcePath: strings.TrimSpace(scope.DirectoryPath), DecisionType: "content_shape_plan", CandidateType: strings.TrimSpace(plan.Shape), TargetKind: "directory", TargetKey: strings.TrimSpace(scope.DirectoryPath), Status: "provisional", Confidence: &confidence, AlternativesJSON: mustJSON(plan.Alternatives), EvidenceJSON: mustJSON(plan.Evidence), AffectedFilesJSON: mustJSON(affected), Reason: "content shape plan remains provisional for resolver follow-up"}).Error
}

func saveContentShapeCorrectionRule(ctx context.Context, db *gorm.DB, scope contentShapeScope, ruleName string, shape string, payload map[string]any, createdByUserID *uint) error {
	key := strings.Join([]string{"content-shape", fmt.Sprint(scope.LibraryID), strings.TrimSpace(scope.StorageProvider), strings.TrimSpace(scope.DirectoryPath), strings.TrimSpace(shape)}, ":")
	rule := database.ClassificationRule{LibraryID: scope.LibraryID, Key: key, Name: strings.TrimSpace(ruleName), PathPattern: strings.TrimSpace(scope.DirectoryPath), RuleType: "content_shape_directory", CandidateType: strings.TrimSpace(shape), PayloadJSON: mustJSON(payload), Enabled: true, CreatedByUserID: createdByUserID}
	if rule.Name == "" {
		rule.Name = "Content shape correction"
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key"}}, DoUpdates: clause.AssignmentColumns([]string{"name", "path_pattern", "rule_type", "candidate_type", "payload_json", "enabled", "updated_at"})}).Create(&rule).Error
}

func invalidateContentShapeScope(ctx context.Context, db *gorm.DB, scope contentShapeScope, now time.Time) error {
	return markContentShapeScopeDeleted(ctx, db, now, "library_id = ? AND storage_provider = ? AND root_path = ? AND directory_path = ?", scope.LibraryID, strings.TrimSpace(scope.StorageProvider), strings.TrimSpace(scope.RootPath), strings.TrimSpace(scope.DirectoryPath))
}
