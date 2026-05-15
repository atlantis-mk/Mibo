package library

import (
	"context"
	"fmt"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
)

func (s *Service) contentShapeCachePlanForDirectory(ctx context.Context, config EffectiveLibraryConfig, pathRecord database.LibraryPath, provider storage.Provider, snapshot scanDirectorySnapshot, batchState *recognitionBatchState) (contentShapeDirectoryPlan, error) {
	if batchState == nil {
		return contentShapeDirectoryPlan{}, nil
	}
	key := keyForShapeAssignments(provider, pathRecord.RootPath, snapshot.Path)
	if plan, ok := batchState.shapePlansByDir[key]; ok {
		if batchState.shapeCounters != nil {
			batchState.shapeCounters.PlanReuses++
		}
		return plan, nil
	}
	settings := contentShapeSettingsFromConfig(s.cfg)
	if s.db == nil {
		if batchState.shapeCounters != nil {
			batchState.shapeCounters.DirectoryProfileBuilds++
			batchState.shapeCounters.PlanCompiles++
		}
		profile := buildContentShapeDirectoryProfile(effectiveVideoLibraryType(config.Library.Type), pathRecord.RootPath, snapshot, batchState.tokenProfileCache)
		plan := compileContentShapePlan(profile)
		batchState.shapePlansByDir[key] = plan
		return plan, nil
	}
	scope := contentShapeScope{LibraryID: config.Library.ID, MediaSourceID: pathRecord.MediaSourceID, StorageProvider: strings.TrimSpace(provider.Name()), RootPath: strings.TrimSpace(pathRecord.RootPath), DirectoryPath: strings.TrimSpace(snapshot.Path), ClassifierVersion: settings.ClassifierVersion}
	if pathRecord.ID != 0 {
		pathID := pathRecord.ID
		scope.LibraryPathID = &pathID
	}
	profileRecord, builtProfile, profileReused, err := loadOrBuildContentShapeProfileWithBuilt(ctx, s.db, scope, snapshot, batchState.scanPolicy, batchState.exclusionRules, batchState.tokenProfileCache)
	if err != nil {
		return contentShapeDirectoryPlan{}, err
	}
	if !profileReused && batchState.shapeCounters != nil {
		batchState.shapeCounters.DirectoryProfileBuilds++
	}
	scope = contentShapeScopeFromProfile(profileRecord)
	planRecord, reusedPlan, err := loadReusableContentShapePlan(ctx, s.db, scope)
	if err != nil {
		return contentShapeDirectoryPlan{}, err
	}
	if reusedPlan {
		return s.reuseCachedContentShapePlan(ctx, key, snapshot, profileRecord, planRecord, batchState)
	}
	return s.compileCachedContentShapePlan(ctx, key, scope, snapshot, builtProfile, profileRecord, batchState)
}

func (s *Service) reuseCachedContentShapePlan(ctx context.Context, key string, snapshot scanDirectorySnapshot, profileRecord database.ContentShapeProfile, planRecord database.ContentShapePlan, batchState *recognitionBatchState) (contentShapeDirectoryPlan, error) {
	if batchState != nil && batchState.shapeCounters != nil {
		batchState.shapeCounters.PlanReuses++
	}
	plan := contentShapePlanFromRecord(planRecord)
	visiblePaths := contentShapeVisibleVideoPaths(snapshot)
	rows, err := loadReusableContentShapeAssignments(ctx, s.db, planRecord.ID, visiblePaths)
	if err != nil {
		return contentShapeDirectoryPlan{}, err
	}
	if len(rows) == len(visiblePaths) {
		batchState.shapeAssignmentsByDir[key] = contentShapeAssignmentsFromRecords(rows)
	} else {
		assignments := generateContentShapeAssignments(plan, snapshot, batchState.tokenProfileCache)
		if err := saveContentShapeAssignments(ctx, s.db, contentShapeScopeFromProfile(profileRecord), profileRecord.ID, planRecord.ID, assignments); err != nil {
			return contentShapeDirectoryPlan{}, err
		}
		batchState.shapeAssignmentsByDir[key] = contentShapeAssignmentsByPath(assignments)
	}
	batchState.shapePlansByDir[key] = plan
	return plan, nil
}

func (s *Service) compileCachedContentShapePlan(ctx context.Context, key string, scope contentShapeScope, snapshot scanDirectorySnapshot, builtProfile contentShapeDirectoryProfile, profileRecord database.ContentShapeProfile, batchState *recognitionBatchState) (contentShapeDirectoryPlan, error) {
	if batchState != nil && batchState.shapeCounters != nil {
		batchState.shapeCounters.PlanCompiles++
	}
	plan := compileContentShapePlan(builtProfile)
	planRow := contentShapeDatabasePlan(scope, profileRecord.ID, plan)
	if err := saveContentShapePlan(ctx, s.db, &planRow); err != nil {
		return contentShapeDirectoryPlan{}, err
	}
	planRecord, reusedPlan, err := loadReusableContentShapePlan(ctx, s.db, scope)
	if err != nil {
		return contentShapeDirectoryPlan{}, err
	}
	if !reusedPlan {
		return contentShapeDirectoryPlan{}, fmt.Errorf("reload content shape plan for %s", scope.DirectoryPath)
	}
	assignments := generateContentShapeAssignments(plan, snapshot, batchState.tokenProfileCache)
	if err := saveContentShapeAssignments(ctx, s.db, scope, profileRecord.ID, planRecord.ID, assignments); err != nil {
		return contentShapeDirectoryPlan{}, err
	}
	batchState.shapeAssignmentsByDir[key] = contentShapeAssignmentsByPath(assignments)
	if err := saveContentShapeReviewDecision(ctx, s.db, scope, plan, assignments); err != nil {
		return contentShapeDirectoryPlan{}, err
	}
	batchState.shapePlansByDir[key] = plan
	return plan, nil
}

func keyForShapeAssignments(provider storage.Provider, rootPath string, directoryPath string) string {
	return strings.TrimSpace(provider.Name()) + "\x00" + strings.TrimSpace(rootPath) + "\x00" + strings.TrimSpace(directoryPath)
}

func keyForDecisionSnapshot(provider storage.Provider, rootPath string, directoryPath string) string {
	return strings.TrimSpace(provider.Name()) + "\x00" + strings.TrimSpace(rootPath) + "\x00" + strings.TrimSpace(directoryPath)
}

func (s *Service) recognitionDecisionSnapshot(ctx context.Context, provider storage.Provider, library database.Library, snapshot scanDirectorySnapshot, batchState *recognitionBatchState) (scanDirectorySnapshot, error) {
	if batchState == nil {
		return s.filteredScanSnapshot(ctx, provider, library, snapshot, nil, database.LibraryScanPolicy{})
	}
	key := keyForDecisionSnapshot(provider, library.RootPath, snapshot.Path)
	if cached, ok := batchState.decisionSnapshots[key]; ok {
		return cached, nil
	}
	decisionSnapshot, err := s.filteredScanSnapshot(ctx, provider, library, snapshot, batchState.exclusionRules, batchState.scanPolicy)
	if err != nil {
		return scanDirectorySnapshot{}, err
	}
	batchState.decisionSnapshots[key] = decisionSnapshot
	return decisionSnapshot, nil
}
