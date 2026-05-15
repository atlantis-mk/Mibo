package library

import "github.com/atlan/mibo-media-server/internal/database"

type recognitionBatchState struct {
	scanPolicy                database.LibraryScanPolicy
	subtitlePolicy            database.LibrarySubtitlePolicy
	exclusionRules            []database.ScanExclusionRule
	directorySnapshots        map[string]scanDirectorySnapshot
	decisionSnapshots         map[string]scanDirectorySnapshot
	tokenProfileCache         *filenameTokenProfileCache
	shapePlansByDir           map[string]contentShapeDirectoryPlan
	shapeAssignmentsByDir     map[string]map[string]contentShapeFileAssignment
	shapeCounters             *contentShapeCounters
	indexedSignalsByPath      map[string]filenameSignalModel
}
