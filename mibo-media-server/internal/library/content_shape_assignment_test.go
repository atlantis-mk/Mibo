package library

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
)

func TestContentShapeAssignmentsReuseAbsolutePlanForAddedEpisode(t *testing.T) {
	t.Parallel()

	snapshot := scanDirectorySnapshot{Path: "/library/Show", Objects: []storage.Object{{Path: "/library/Show/001.mkv"}, {Path: "/library/Show/002.mkv"}, {Path: "/library/Show/003.mkv"}}}
	profile := buildContentShapeDirectoryProfile("auto", "/library", snapshot, newFilenameTokenProfileCache())
	plan := compileContentShapePlan(profile)
	record := contentShapeDatabasePlan(testContentShapeScope(ContentShapeClassifierVersion, snapshot.Path), 1, plan)
	reused := generateContentShapeAssignmentsFromPersistedRule(record, scanDirectorySnapshot{Path: snapshot.Path, Objects: append(snapshot.Objects, storage.Object{Path: "/library/Show/004.mkv"})}, newFilenameTokenProfileCache())
	if len(reused) != 4 || reused[3].AbsoluteNumber == nil || *reused[3].AbsoluteNumber != 4 {
		t.Fatalf("expected added episode assignment from persisted rule, got %#v", reused)
	}
}

func TestContentShapePlanFromRecordPreservesPersistedRuleFields(t *testing.T) {
	t.Parallel()

	season := 2
	confidence := 0.88
	record := database.ContentShapePlan{
		Shape:         contentShapeSeasonFolder,
		Confidence:    &confidence,
		ReviewState:   "auto",
		SeriesTitle:   "Fallback Series",
		SeasonNumber:  &season,
		NumberingMode: "season_episode",
		PlanRuleJSON:  `{"shape":"season_folder","series_title":"Persisted Series","season_number":2,"numbering_mode":"season_episode"}`,
	}
	plan := contentShapePlanFromRecord(record)
	if plan.Shape != contentShapeSeasonFolder || plan.SeriesTitle != "Persisted Series" || plan.SeasonNumber == nil || *plan.SeasonNumber != 2 {
		t.Fatalf("expected persisted rule fields to hydrate plan, got %#v", plan)
	}
}

func TestContentShapePersistedPlanAndAssignmentsSavedAndReused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	state := recognitionBatchState{scanPolicy: database.LibraryScanPolicy{IgnoreHiddenFiles: true}, tokenProfileCache: newFilenameTokenProfileCache(), shapePlansByDir: make(map[string]contentShapeDirectoryPlan), shapeAssignmentsByDir: make(map[string]map[string]contentShapeFileAssignment), shapeCounters: &contentShapeCounters{}}
	state.tokenProfileCache.counters = state.shapeCounters
	svc := NewService(config.Config{}, db, nil, nil)
	cfg := EffectiveLibraryConfig{Library: database.Library{ID: 1, Type: "auto", RootPath: "/library"}}
	pathRecord := database.LibraryPath{ID: 7, LibraryID: 1, MediaSourceID: 1, RootPath: "/library"}
	provider := staticNameProvider{name: "local"}
	snapshot := largeEpisodeShapeSnapshot("/library/Show", 5)

	first, err := svc.contentShapePlanForRecognitionDirectory(ctx, cfg, pathRecord, provider, snapshot, &state)
	if err != nil {
		t.Fatalf("first content shape plan: %v", err)
	}
	if first.Shape == "" {
		t.Fatalf("expected compiled plan, got %#v", first)
	}
	var profileCount int64
	if err := db.WithContext(ctx).Model(&database.ContentShapeProfile{}).Count(&profileCount).Error; err != nil {
		t.Fatalf("count profiles: %v", err)
	}
	if profileCount != 1 {
		t.Fatalf("expected one persisted profile, got %d", profileCount)
	}
	var planCount int64
	if err := db.WithContext(ctx).Model(&database.ContentShapePlan{}).Count(&planCount).Error; err != nil {
		t.Fatalf("count plans: %v", err)
	}
	if planCount != 1 {
		t.Fatalf("expected one persisted plan, got %d", planCount)
	}
	var assignmentCount int64
	if err := db.WithContext(ctx).Model(&database.ContentShapeAssignment{}).Count(&assignmentCount).Error; err != nil {
		t.Fatalf("count assignments: %v", err)
	}
	if assignmentCount != int64(len(snapshot.Objects)) {
		t.Fatalf("expected %d persisted assignments, got %d", len(snapshot.Objects), assignmentCount)
	}

	state.shapePlansByDir = make(map[string]contentShapeDirectoryPlan)
	second, err := svc.contentShapePlanForRecognitionDirectory(ctx, cfg, pathRecord, provider, snapshot, &state)
	if err != nil {
		t.Fatalf("second content shape plan: %v", err)
	}
	if second.Shape != first.Shape || second.NumberingMode != first.NumberingMode {
		t.Fatalf("expected persisted plan reuse, first=%#v second=%#v", first, second)
	}
	counters := state.shapeCounters.snapshot()
	if counters.PlanCompiles != 1 || counters.PlanReuses != 1 {
		t.Fatalf("expected one compile and one reuse, got %#v", counters)
	}
}

func TestContentShapePersistedReviewDecisionNotDuplicatedOnReuse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	state := recognitionBatchState{scanPolicy: database.LibraryScanPolicy{IgnoreHiddenFiles: true}, tokenProfileCache: newFilenameTokenProfileCache(), shapePlansByDir: make(map[string]contentShapeDirectoryPlan), shapeAssignmentsByDir: make(map[string]map[string]contentShapeFileAssignment), shapeCounters: &contentShapeCounters{}}
	state.tokenProfileCache.counters = state.shapeCounters
	svc := NewService(config.Config{}, db, nil, nil)
	cfg := EffectiveLibraryConfig{Library: database.Library{ID: 1, Type: "auto", RootPath: "/library"}}
	pathRecord := database.LibraryPath{ID: 9, LibraryID: 1, MediaSourceID: 1, RootPath: "/library"}
	provider := staticNameProvider{name: "local"}
	snapshot := scanDirectorySnapshot{Path: "/library/Ambiguous", Objects: []storage.Object{{Path: "/library/Ambiguous/001.mkv"}, {Path: "/library/Ambiguous/002.mkv"}}}
	scope := testContentShapeScope(ContentShapeClassifierVersion, snapshot.Path)
	pathID := pathRecord.ID
	scope.LibraryPathID = &pathID
	profile, _, err := loadOrBuildContentShapeProfile(ctx, db, scope, snapshot, state.scanPolicy, nil, state.tokenProfileCache)
	if err != nil {
		t.Fatalf("build profile: %v", err)
	}
	scope = contentShapeScopeFromProfile(profile)
	reviewPlan := contentShapeDirectoryPlan{Shape: contentShapeUnknownReview, Confidence: 0.51, ReviewState: "review_required", Evidence: map[string]any{"source": "directory_profile"}, Alternatives: []contentShapePlanAlternative{{Shape: contentShapeAbsoluteEpisodePack, Confidence: 0.5}, {Shape: contentShapeMovieCollection, Confidence: 0.49}}}
	planRow := contentShapeDatabasePlan(scope, profile.ID, reviewPlan)
	if err := saveContentShapePlan(ctx, db, &planRow); err != nil {
		t.Fatalf("save review plan: %v", err)
	}
	persistedPlan, reused, err := loadReusableContentShapePlan(ctx, db, scope)
	if err != nil || !reused {
		t.Fatalf("load persisted review plan: reused=%t err=%v", reused, err)
	}
	assignments := generateContentShapeAssignmentsFromPersistedRule(persistedPlan, snapshot, state.tokenProfileCache)
	if err := saveContentShapeAssignments(ctx, db, scope, profile.ID, persistedPlan.ID, assignments); err != nil {
		t.Fatalf("save review assignments: %v", err)
	}
	if err := saveContentShapeReviewDecision(ctx, db, scope, reviewPlan, assignments); err != nil {
		t.Fatalf("save review decision: %v", err)
	}
	var firstCount int64
	if err := db.WithContext(ctx).Model(&database.ClassificationDecision{}).Where("library_id = ? AND decision_type = ?", cfg.Library.ID, "content_shape_plan").Count(&firstCount).Error; err != nil {
		t.Fatalf("count first review decisions: %v", err)
	}
	if firstCount != 1 {
		t.Fatalf("expected one seeded review decision, got %d", firstCount)
	}

	if _, err := svc.contentShapePlanForRecognitionDirectory(ctx, cfg, pathRecord, provider, snapshot, &state); err != nil {
		t.Fatalf("reuse review plan: %v", err)
	}
	state.shapePlansByDir = make(map[string]contentShapeDirectoryPlan)
	if _, err := svc.contentShapePlanForRecognitionDirectory(ctx, cfg, pathRecord, provider, snapshot, &state); err != nil {
		t.Fatalf("reuse persisted review plan after cache clear: %v", err)
	}
	var secondCount int64
	if err := db.WithContext(ctx).Model(&database.ClassificationDecision{}).Where("library_id = ? AND decision_type = ?", cfg.Library.ID, "content_shape_plan").Count(&secondCount).Error; err != nil {
		t.Fatalf("count second review decisions: %v", err)
	}
	if secondCount != 1 {
		t.Fatalf("expected persisted review plan reuse to avoid duplicate decisions, got %d", secondCount)
	}
}

func TestContentShapePlanReuseRejectsDeletedOrConflictingDeltas(t *testing.T) {
	t.Parallel()

	confidence := 0.95
	settings := contentShapeSettings{PlanReuseConfidenceThreshold: 0.85, MediumReviewConfidenceThreshold: 0.65}
	existing := database.ContentShapePlan{Shape: contentShapeAbsoluteEpisodePack, Confidence: &confidence, ReviewState: "auto"}
	deletedProfile := buildContentShapeDirectoryProfile("auto", "/library", scanDirectorySnapshot{Path: "/library/Show", Objects: []storage.Object{{Path: "/library/Show/001.mkv"}, {Path: "/library/Show/003.mkv"}}}, newFilenameTokenProfileCache())
	if ok, _ := contentShapePlanReuseDecision(existing, deletedProfile, settings); !ok {
		t.Fatalf("expected simple deletion/gap to keep reusable absolute plan")
	}
	conflictProfile := buildContentShapeDirectoryProfile("auto", "/library", scanDirectorySnapshot{Path: "/library/Show", Objects: []storage.Object{{Path: "/library/Show/001.mkv"}, {Path: "/library/Show/002.mkv"}, {Path: "/library/Show/Inception.2010.mkv"}, {Path: "/library/Show/Heat.1995.mkv"}}}, newFilenameTokenProfileCache())
	if ok, reason := contentShapePlanReuseDecision(existing, conflictProfile, settings); ok {
		t.Fatalf("expected conflicting movie-like files to invalidate reuse, reason=%s profile=%#v", reason, conflictProfile)
	}
}

func TestContentShapeAssignmentsStableAcrossBatches(t *testing.T) {
	t.Parallel()

	snapshot := scanDirectorySnapshot{Path: "/library/Show", Objects: []storage.Object{{Path: "/library/Show/001.mkv"}, {Path: "/library/Show/002.mkv"}, {Path: "/library/Show/003.mkv"}, {Path: "/library/Show/004.mkv"}}}
	plan := compileContentShapePlan(buildContentShapeDirectoryProfile("auto", "/library", snapshot, newFilenameTokenProfileCache()))
	firstBatch := generateContentShapeAssignments(plan, scanDirectorySnapshot{Path: snapshot.Path, Objects: snapshot.Objects[:2]}, newFilenameTokenProfileCache())
	secondBatch := generateContentShapeAssignments(plan, scanDirectorySnapshot{Path: snapshot.Path, Objects: snapshot.Objects[2:]}, newFilenameTokenProfileCache())
	if firstBatch[0].TargetKey == "" || firstBatch[1].TargetKey == "" || secondBatch[0].TargetKey == "" || secondBatch[1].TargetKey == "" {
		t.Fatalf("expected stable target keys, first=%#v second=%#v", firstBatch, secondBatch)
	}
	if firstBatch[0].TargetKey == secondBatch[0].TargetKey {
		t.Fatalf("expected distinct episode target keys across batches, first=%#v second=%#v", firstBatch, secondBatch)
	}
}

func TestContentShapeLargeEpisodeDirectoryPlanReusedAcrossBatches(t *testing.T) {
	t.Parallel()

	state := recognitionBatchState{tokenProfileCache: newFilenameTokenProfileCache(), shapePlansByDir: make(map[string]contentShapeDirectoryPlan), shapeAssignmentsByDir: make(map[string]map[string]contentShapeFileAssignment), shapeCounters: &contentShapeCounters{}}
	state.tokenProfileCache.counters = state.shapeCounters
	svc := NewService(config.Config{}, nil, nil, nil)
	config := EffectiveLibraryConfig{Library: database.Library{ID: 1, Type: "auto", RootPath: "/library"}}
	pathRecord := database.LibraryPath{LibraryID: 1, MediaSourceID: 1, RootPath: "/library"}
	provider := staticNameProvider{name: "local"}
	snapshot := largeEpisodeShapeSnapshot("/library/Show", 1000)
	first, err := svc.contentShapePlanForRecognitionDirectory(context.Background(), config, pathRecord, provider, snapshot, &state)
	if err != nil {
		t.Fatalf("compile first plan: %v", err)
	}
	second, err := svc.contentShapePlanForRecognitionDirectory(context.Background(), config, pathRecord, provider, scanDirectorySnapshot{Path: snapshot.Path, Objects: snapshot.Objects[:25]}, &state)
	if err != nil {
		t.Fatalf("reuse second plan: %v", err)
	}
	if len(state.shapePlansByDir) != 1 {
		t.Fatalf("expected one cached directory plan, got %#v", state.shapePlansByDir)
	}
	if first.Shape != second.Shape || first.Shape != contentShapeAbsoluteEpisodePack {
		t.Fatalf("expected reused absolute plan, first=%#v second=%#v", first, second)
	}
	counters := state.shapeCounters.snapshot()
	if counters.DirectoryProfileBuilds != 1 || counters.PlanCompiles != 1 || counters.PlanReuses != 1 {
		t.Fatalf("expected one profile build/compile and one reuse, got %#v", counters)
	}
}

func TestContentShapeFlatEpisodeAssignmentsRemainDistinctWhenReused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	state := recognitionBatchState{scanPolicy: database.LibraryScanPolicy{IgnoreHiddenFiles: true}, tokenProfileCache: newFilenameTokenProfileCache(), shapePlansByDir: make(map[string]contentShapeDirectoryPlan), shapeAssignmentsByDir: make(map[string]map[string]contentShapeFileAssignment), shapeCounters: &contentShapeCounters{}}
	state.tokenProfileCache.counters = state.shapeCounters
	svc := NewService(config.Config{}, db, nil, nil)
	cfg := EffectiveLibraryConfig{Library: database.Library{ID: 1, Type: "auto", RootPath: "/library"}}
	pathRecord := database.LibraryPath{ID: 11, LibraryID: 1, MediaSourceID: 1, RootPath: "/library"}
	provider := staticNameProvider{name: "local"}
	snapshot := scanDirectorySnapshot{Path: "/library/FlatShow", Objects: []storage.Object{{Path: "/library/FlatShow/Alpha.mkv"}, {Path: "/library/FlatShow/Beta.mkv"}, {Path: "/library/FlatShow/Gamma.mkv"}}}

	plan, err := svc.contentShapePlanForRecognitionDirectory(ctx, cfg, pathRecord, provider, snapshot, &state)
	if err != nil {
		t.Fatalf("build flat plan: %v", err)
	}
	plan.Shape = contentShapeFlatEpisodeFolder
	plan.NumberingMode = "sorted_or_numeric"
	key := keyForShapeAssignments(provider, pathRecord.RootPath, snapshot.Path)
	assignments := generateContentShapeAssignments(plan, snapshot, state.tokenProfileCache)
	state.shapePlansByDir[key] = plan
	state.shapeAssignmentsByDir[key] = contentShapeAssignmentsByPath(assignments)

	alpha := state.shapeAssignmentsByDir[key]["/library/FlatShow/Alpha.mkv"]
	beta := state.shapeAssignmentsByDir[key]["/library/FlatShow/Beta.mkv"]
	if alpha.EpisodeNumber == nil || beta.EpisodeNumber == nil {
		t.Fatalf("expected episode numbers for flat folder assignments, alpha=%#v beta=%#v", alpha, beta)
	}
	if *alpha.EpisodeNumber == *beta.EpisodeNumber {
		t.Fatalf("expected distinct episode numbers from directory-level assignments, alpha=%#v beta=%#v", alpha, beta)
	}
	if alpha.TargetKey == "" || beta.TargetKey == "" {
		t.Fatalf("expected stable episode target keys from reused assignments, alpha=%#v beta=%#v", alpha, beta)
	}
}

type staticNameProvider struct{ name string }

func (p staticNameProvider) Name() string { return p.name }

func (p staticNameProvider) List(ctx context.Context, req storage.ListRequest) ([]storage.Object, error) {
	return nil, storage.ErrNotImplemented
}

func (p staticNameProvider) Get(ctx context.Context, req storage.GetRequest) (storage.Object, error) {
	return storage.Object{}, storage.ErrNotImplemented
}

func (p staticNameProvider) Link(ctx context.Context, req storage.LinkRequest) (storage.LinkResult, error) {
	return storage.LinkResult{}, storage.ErrNotImplemented
}

func (p staticNameProvider) ResolveStorage(ctx context.Context, req storage.ResolveStorageRequest) (storage.ResolvedStorage, error) {
	return storage.ResolvedStorage{}, storage.ErrNotImplemented
}

func (p staticNameProvider) Capabilities(ctx context.Context) (storage.Capabilities, error) {
	return storage.Capabilities{}, nil
}
