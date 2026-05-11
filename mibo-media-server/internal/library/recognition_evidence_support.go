package library

import (
	"context"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

type recognitionBatchState struct {
	scanPolicy                database.LibraryScanPolicy
	subtitlePolicy            database.LibrarySubtitlePolicy
	exclusionRules            []database.ScanExclusionRule
	directorySnapshots        map[string]scanDirectorySnapshot
	decisionSnapshots         map[string]scanDirectorySnapshot
	tokenProfileCache         *filenameTokenProfileCache
	shapePlansByDir           map[string]contentShapeDirectoryPlan
	shapeAssignmentsByDir     map[string]map[string]contentShapeFileAssignment
	pathTreeAssignmentsByPath map[string]pathTreeWorkGroupAssignment
	shapeCounters             *contentShapeCounters
	indexedSignalsByPath      map[string]filenameSignalModel
}

func loadPathTreeClassificationRules(ctx context.Context, db *gorm.DB, libraryID uint) ([]database.ClassificationRule, error) {
	if db == nil || libraryID == 0 {
		return nil, nil
	}
	var rules []database.ClassificationRule
	err := db.WithContext(ctx).Where("library_id = ? AND rule_type = ? AND enabled = ?", libraryID, pathTreeWorkGroupRuleType, true).Order("id asc").Find(&rules).Error
	return rules, err
}
