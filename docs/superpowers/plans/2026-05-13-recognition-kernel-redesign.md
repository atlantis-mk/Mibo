# Recognition Kernel Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the media recognition kernel and algorithm while preserving the existing scan, workflow, materialization, and catalog projection shells.

**Architecture:** Keep `internal/library` as the orchestration boundary and keep `internal/recognition.Repository`, `ManifestGraph`, `RecognitionCandidate`, `RecognitionEvidence`, `RecognitionDecision`, and `Materializer` as the persistence/materialization contracts. Introduce a layered kernel inside `internal/recognition`: evidence collection, work-unit building, candidate generation, hard constraints, graph inference, scoring, decision explanation, and dual-run evaluation.

**Tech Stack:** Go, GORM, SQLite-backed tests, existing Mibo `database`, `library`, `recognition`, `catalog`, and `workflow` packages.

---

## Current Contracts To Preserve

The rewrite must preserve these outer-shell seams:

- `mibo-media-server/internal/library/scan_run.go` remains responsible for scanning storage and writing `InventoryFile` rows.
- `mibo-media-server/internal/library/workflow.go` keeps the scan -> recognition resolve -> metadata match/probe -> projection sequencing. In particular, `queueWorkflowPostScanTasks` must continue to queue recognition tasks before projection when recognition file IDs exist, and `queueWorkflowPostRecognitionResolveTasks` must queue projection after recognition materialization.
- `mibo-media-server/internal/library/recognition_manifest.go` remains the entrypoint from library orchestration into recognition manifest persistence.
- `mibo-media-server/internal/recognition/repository.go` remains the only place that writes recognition manifests, graph rows, candidates, evidence, decisions, and conflicts.
- `mibo-media-server/internal/recognition/materializer.go` remains the boundary that turns accepted decisions into metadata/resource rows.
- `mibo-media-server/internal/catalog/metadata_projection.go` remains responsible for projection rebuilds after materialization.
- Existing database model contracts in `mibo-media-server/internal/database/recognition_models.go` and `mibo-media-server/internal/database/media_graph_models.go` should be reused before adding new tables.

## File Structure

Create focused recognition-kernel files rather than expanding `manifest_builder.go` further:

- Create `mibo-media-server/internal/recognition/evidence_collector.go`: normalizes inventory facts, reusable file signals, sidecars, and context evidence into file-level evidence.
- Create `mibo-media-server/internal/recognition/work_unit_builder.go`: groups files into directory-level `RecognitionWorkUnit` values and classifies folder shape.
- Create `mibo-media-server/internal/recognition/candidate_generator.go`: emits possible work, episode, resource, variant, edition, collection, and extra candidates from work units.
- Create `mibo-media-server/internal/recognition/constraints.go`: prunes or demotes invalid candidates and records reasons.
- Create `mibo-media-server/internal/recognition/graph_inference.go`: chooses globally consistent candidate sets across siblings and parent-child relationships.
- Create `mibo-media-server/internal/recognition/scoring.go`: computes lightweight scores only after hard constraints and graph inference.
- Create `mibo-media-server/internal/recognition/decision_engine.go`: converts inferred/scored candidates into decisions and explanations.
- Create `mibo-media-server/internal/recognition/eval_fixtures_test.go`: shared golden fixtures and metrics helpers for recognition evaluation.
- Modify `mibo-media-server/internal/recognition/graph_constructor.go`: replace the direct builder call with the new kernel pipeline while preserving `ConstructGraphFromInventory` signature.
- Modify `mibo-media-server/internal/recognition/manifest_builder.go`: keep key/evidence helper functions that are still useful; move new orchestration to the new files.
- Modify `mibo-media-server/internal/recognition/resolver.go`: keep manual rule handling, but let the new decision engine supply default accepted/review/blocked/unmatched outcomes.
- Modify `mibo-media-server/internal/library/recognition_manifest.go`: keep scan-side input loading and repository persistence, but pass normalized inputs into the new kernel.
- Modify `mibo-media-server/internal/library/workflow.go`: only if needed to preserve stage boundaries or add dual-run configuration; do not change the phase order.
- Modify tests in `mibo-media-server/internal/library` and `mibo-media-server/internal/recognition` to lock the new behavior.

## Task 1: Add Golden Evaluation Fixtures

**Files:**
- Create: `mibo-media-server/internal/recognition/eval_fixtures_test.go`
- Test: `mibo-media-server/internal/recognition/eval_fixtures_test.go`

- [ ] **Step 1: Write the golden fixture types and expectations**

Create `mibo-media-server/internal/recognition/eval_fixtures_test.go` with:

```go
package recognition

import (
	"strings"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
)

type recognitionGoldenFixture struct {
	Name             string
	LibraryRoot      string
	Files            []database.InventoryFile
	Signals          map[uint]database.InventoryFileSignal
	SidecarsByFileID map[uint][]database.InventoryFile
	Expected         recognitionGoldenExpectation
}

type recognitionGoldenExpectation struct {
	AcceptedCandidateKeys []string
	ReviewCandidateKeys   []string
	BlockedCandidateKeys  []string
	UnmatchedCandidateKeys []string
	RequiredEvidenceKeys  []string
}

func goldenRecognitionFixtures() []recognitionGoldenFixture {
	modified := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	uintPtr := func(v uint) *uint { return &v }
	intPtr := func(v int) *int { return &v }
	confidence := func(v float64) *float64 { return &v }

	return []recognitionGoldenFixture{
		{
			Name:        "single movie folder",
			LibraryRoot: "/library",
			Files: []database.InventoryFile{{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie A (2024)/Movie A.2024.1080p.mkv", StableIdentityKey: "local:movie-a", ContentClass: "video", Status: "available", SizeBytes: 1024, ModifiedAt: &modified}},
			Signals: map[uint]database.InventoryFileSignal{1: {InventoryFileID: uintPtr(1), StoragePath: "/library/Movie A (2024)/Movie A.2024.1080p.mkv", TitleCandidate: "Movie A", Year: intPtr(2024), Quality: "1080p", Confidence: confidence(0.90)}},
			Expected: recognitionGoldenExpectation{AcceptedCandidateKeys: []string{"work:movie:movie-a:2024"}, RequiredEvidenceKeys: []string{"title", "year"}},
		},
		{
			Name:        "standard season folder",
			LibraryRoot: "/library",
			Files: []database.InventoryFile{
				{ID: 2, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Show/Season 01/Show.S01E01.mkv", StableIdentityKey: "local:show-e01", ContentClass: "video", Status: "available", SizeBytes: 2048, ModifiedAt: &modified},
				{ID: 3, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Show/Season 01/Show.S01E02.mkv", StableIdentityKey: "local:show-e02", ContentClass: "video", Status: "available", SizeBytes: 2048, ModifiedAt: &modified},
			},
			Signals: map[uint]database.InventoryFileSignal{
				2: {InventoryFileID: uintPtr(2), StoragePath: "/library/Show/Season 01/Show.S01E01.mkv", TitleCandidate: "Show", SeasonNumber: intPtr(1), EpisodeNumber: intPtr(1), Confidence: confidence(0.95)},
				3: {InventoryFileID: uintPtr(3), StoragePath: "/library/Show/Season 01/Show.S01E02.mkv", TitleCandidate: "Show", SeasonNumber: intPtr(1), EpisodeNumber: intPtr(2), Confidence: confidence(0.95)},
			},
			Expected: recognitionGoldenExpectation{AcceptedCandidateKeys: []string{"work:series:show", "work:season:work:series:show:s01", "episode:work:season:work:series:show:s01:e01", "episode:work:season:work:series:show:s01:e02"}, RequiredEvidenceKeys: []string{"season_number", "episode_number"}},
		},
		{
			Name:        "multi version movie folder",
			LibraryRoot: "/library",
			Files: []database.InventoryFile{
				{ID: 4, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie B (2024)/Movie B.2024.1080p.mkv", StableIdentityKey: "local:movie-b-1080p", ContentClass: "video", Status: "available", SizeBytes: 4096, ModifiedAt: &modified},
				{ID: 5, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie B (2024)/Movie B.2024.2160p.mkv", StableIdentityKey: "local:movie-b-2160p", ContentClass: "video", Status: "available", SizeBytes: 8192, ModifiedAt: &modified},
			},
			Signals: map[uint]database.InventoryFileSignal{
				4: {InventoryFileID: uintPtr(4), StoragePath: "/library/Movie B (2024)/Movie B.2024.1080p.mkv", TitleCandidate: "Movie B", Year: intPtr(2024), Quality: "1080p", Confidence: confidence(0.91)},
				5: {InventoryFileID: uintPtr(5), StoragePath: "/library/Movie B (2024)/Movie B.2024.2160p.mkv", TitleCandidate: "Movie B", Year: intPtr(2024), Quality: "2160p", Confidence: confidence(0.91)},
			},
			Expected: recognitionGoldenExpectation{AcceptedCandidateKeys: []string{"work:movie:movie-b:2024"}, RequiredEvidenceKeys: []string{"title", "year", "sibling_consistency"}},
		},
		{
			Name:        "extra does not become main work",
			LibraryRoot: "/library",
			Files: []database.InventoryFile{{ID: 6, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie C (2024)/extras/Movie C Trailer.mkv", StableIdentityKey: "local:movie-c-trailer", ContentClass: "video", Status: "available", SizeBytes: 512, ModifiedAt: &modified}},
			Signals: map[uint]database.InventoryFileSignal{6: {InventoryFileID: uintPtr(6), StoragePath: "/library/Movie C (2024)/extras/Movie C Trailer.mkv", TitleCandidate: "Movie C Trailer", Role: "trailer", Year: intPtr(2024), Confidence: confidence(0.60)}},
			Expected: recognitionGoldenExpectation{ReviewCandidateKeys: []string{"playable_resource:local:path:/library/Movie C (2024)/extras/Movie C Trailer.mkv"}, RequiredEvidenceKeys: []string{"role"}},
		},
	}
}

func assertGoldenDecisions(t *testing.T, fixture recognitionGoldenFixture, result ResolveResult) {
	t.Helper()
	actual := make(map[string]string)
	for _, decision := range result.Decisions {
		actual[strings.TrimSpace(decision.TargetKey)] = strings.TrimSpace(decision.Outcome)
	}
	for _, key := range fixture.Expected.AcceptedCandidateKeys {
		if actual[key] != DecisionOutcomeAccepted {
			t.Fatalf("%s: expected %s accepted, got %q from %#v", fixture.Name, key, actual[key], result.Decisions)
		}
	}
	for _, key := range fixture.Expected.ReviewCandidateKeys {
		if actual[key] != DecisionOutcomeReviewRequired {
			t.Fatalf("%s: expected %s review_required, got %q from %#v", fixture.Name, key, actual[key], result.Decisions)
		}
	}
	for _, key := range fixture.Expected.BlockedCandidateKeys {
		if actual[key] != DecisionOutcomeBlockedConflict {
			t.Fatalf("%s: expected %s blocked_conflict, got %q from %#v", fixture.Name, key, actual[key], result.Decisions)
		}
	}
	for _, key := range fixture.Expected.UnmatchedCandidateKeys {
		if actual[key] != DecisionOutcomeUnmatched {
			t.Fatalf("%s: expected %s unmatched, got %q from %#v", fixture.Name, key, actual[key], result.Decisions)
		}
	}
}
```

- [ ] **Step 2: Run the new fixture file test package**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run TestDoesNotExist
```

Expected: package compiles and reports no tests to run, or passes existing package tests. If it fails, fix compile errors in the fixture helpers before continuing.

- [ ] **Step 3: Commit**

```bash
git add mibo-media-server/internal/recognition/eval_fixtures_test.go
git commit -m "test: add recognition golden fixtures"
```

## Task 2: Introduce Work Unit Model

**Files:**
- Create: `mibo-media-server/internal/recognition/work_unit_builder.go`
- Create: `mibo-media-server/internal/recognition/work_unit_builder_test.go`

- [ ] **Step 1: Write failing work-unit tests**

Create `mibo-media-server/internal/recognition/work_unit_builder_test.go` with:

```go
package recognition

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestBuildRecognitionWorkUnitsGroupsSeasonSiblings(t *testing.T) {
	files := []database.InventoryFile{
		{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Show/Season 01/Show.S01E01.mkv", ContentClass: "video", Status: "available"},
		{ID: 2, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Show/Season 01/Show.S01E02.mkv", ContentClass: "video", Status: "available"},
	}
	input := ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, RootPath: "/library", ScopePath: "/library/Show/Season 01", StorageProvider: "local"}, Files: files}

	units := BuildRecognitionWorkUnits(input)

	if len(units) != 1 {
		t.Fatalf("expected one season work unit, got %#v", units)
	}
	if units[0].FolderShape != FolderShapeSeason || units[0].ScopePath != "/library/Show/Season 01" {
		t.Fatalf("unexpected work unit: %#v", units[0])
	}
	if len(units[0].Files) != 2 {
		t.Fatalf("expected both files in the same unit, got %#v", units[0].Files)
	}
}

func TestBuildRecognitionWorkUnitsKeepsExtrasSeparate(t *testing.T) {
	files := []database.InventoryFile{
		{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie (2024)/Movie.2024.mkv", ContentClass: "video", Status: "available"},
		{ID: 2, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie (2024)/extras/Trailer.mkv", ContentClass: "video", Status: "available"},
	}
	input := ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, RootPath: "/library", ScopePath: "/library/Movie (2024)", StorageProvider: "local"}, Files: files}

	units := BuildRecognitionWorkUnits(input)

	if len(units) != 2 {
		t.Fatalf("expected main movie and extra units, got %#v", units)
	}
	if units[0].FolderShape != FolderShapeMovie || units[1].FolderShape != FolderShapeExtra {
		t.Fatalf("expected movie then extra units, got %#v", units)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run 'TestBuildRecognitionWorkUnits' -count=1
```

Expected: FAIL because `RecognitionWorkUnit`, folder-shape constants, and `BuildRecognitionWorkUnits` are not defined.

- [ ] **Step 3: Implement the minimal work-unit builder**

Create `mibo-media-server/internal/recognition/work_unit_builder.go` with:

```go
package recognition

import (
	"path"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

const (
	FolderShapeMovie     = "movie_folder"
	FolderShapeSeries    = "series_root"
	FolderShapeSeason    = "season_folder"
	FolderShapeCollection = "collection_folder"
	FolderShapeExtra     = "extra_folder"
	FolderShapeMixed     = "mixed_folder"
)

type RecognitionWorkUnit struct {
	ScopePath       string
	FolderShape     string
	Files           []database.InventoryFile
	FileSignals     map[uint]database.InventoryFileSignal
	SidecarsByFileID map[uint][]database.InventoryFile
	ContextEvidence map[uint][]ContextEvidence
}

func BuildRecognitionWorkUnits(input ManifestBuildInput) []RecognitionWorkUnit {
	groups := make(map[string][]database.InventoryFile)
	for _, file := range input.Files {
		if strings.TrimSpace(file.StoragePath) == "" || strings.TrimSpace(file.ContentClass) != videoContentClass {
			continue
		}
		scope := workUnitScopePath(input.Scope.RootPath, file.StoragePath)
		groups[scope] = append(groups[scope], file)
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	units := make([]RecognitionWorkUnit, 0, len(keys))
	for _, key := range keys {
		files := append([]database.InventoryFile(nil), groups[key]...)
		sort.Slice(files, func(i, j int) bool { return files[i].StoragePath < files[j].StoragePath })
		units = append(units, RecognitionWorkUnit{ScopePath: key, FolderShape: inferFolderShape(key, files), Files: files, FileSignals: input.FileSignals, SidecarsByFileID: input.SidecarsByFileID, ContextEvidence: input.ContextEvidence})
	}
	return units
}

func workUnitScopePath(rootPath string, storagePath string) string {
	dir := path.Dir(strings.TrimSpace(storagePath))
	base := strings.ToLower(path.Base(dir))
	if base == "extras" || base == "extra" || base == "samples" || base == "sample" || base == "trailers" || base == "trailer" {
		return dir
	}
	return dir
}

func inferFolderShape(scopePath string, files []database.InventoryFile) string {
	base := strings.ToLower(path.Base(scopePath))
	if base == "extras" || base == "extra" || base == "samples" || base == "sample" || base == "trailers" || base == "trailer" {
		return FolderShapeExtra
	}
	if strings.HasPrefix(base, "season ") || strings.HasPrefix(base, "season.") || strings.HasPrefix(base, "s0") || strings.HasPrefix(base, "s1") {
		return FolderShapeSeason
	}
	if len(files) > 1 {
		return FolderShapeMixed
	}
	return FolderShapeMovie
}
```

- [ ] **Step 4: Run work-unit tests**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run 'TestBuildRecognitionWorkUnits' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mibo-media-server/internal/recognition/work_unit_builder.go mibo-media-server/internal/recognition/work_unit_builder_test.go
git commit -m "feat: add recognition work units"
```

## Task 3: Add Evidence Collection Boundary

**Files:**
- Create: `mibo-media-server/internal/recognition/evidence_collector.go`
- Create: `mibo-media-server/internal/recognition/evidence_collector_test.go`

- [ ] **Step 1: Write failing evidence collector test**

Create `mibo-media-server/internal/recognition/evidence_collector_test.go` with:

```go
package recognition

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestCollectWorkUnitEvidenceKeepsSignalSidecarAndContextEvidence(t *testing.T) {
	fileID := uint(7)
	unit := RecognitionWorkUnit{
		ScopePath:   "/library/Show/Season 01",
		FolderShape: FolderShapeSeason,
		Files:       []database.InventoryFile{{ID: fileID, StoragePath: "/library/Show/Season 01/Show.S01E02.mkv", ContentClass: "video", Status: "available"}},
		FileSignals: map[uint]database.InventoryFileSignal{fileID: {InventoryFileID: &fileID, TitleCandidate: "Show", SeasonNumber: intPtrForTest(1), EpisodeNumber: intPtrForTest(2)}},
		ContextEvidence: map[uint][]ContextEvidence{fileID: {{Source: "path_tree", Assignment: "season_folder"}}},
	}

	evidence := CollectWorkUnitEvidence(unit)

	if !hasEvidence(evidence, fileID, "title") || !hasEvidence(evidence, fileID, "season_number") || !hasEvidence(evidence, fileID, "episode_number") || !hasEvidence(evidence, fileID, "folder_shape") {
		t.Fatalf("expected normalized evidence, got %#v", evidence)
	}
}

func intPtrForTest(v int) *int { return &v }

func hasEvidence(items []database.RecognitionEvidence, fileID uint, key string) bool {
	for _, item := range items {
		if item.InventoryFileID != nil && *item.InventoryFileID == fileID && item.EvidenceKey == key {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run test to verify failure**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run TestCollectWorkUnitEvidenceKeepsSignalSidecarAndContextEvidence -count=1
```

Expected: FAIL because `CollectWorkUnitEvidence` is not defined.

- [ ] **Step 3: Implement evidence collection**

Create `mibo-media-server/internal/recognition/evidence_collector.go` with:

```go
package recognition

import (
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

func CollectWorkUnitEvidence(unit RecognitionWorkUnit) []database.RecognitionEvidence {
	items := make([]database.RecognitionEvidence, 0, len(unit.Files)*8)
	for _, file := range unit.Files {
		fileID := file.ID
		items = append(items, inventoryEvidence(file)...)
		if signal, ok := unit.FileSignals[file.ID]; ok {
			items = append(items, signalEvidence(fileID, "file_signal", signal)...)
		}
		for _, hint := range unit.ContextEvidence[file.ID] {
			items = append(items, contextEvidenceItem(fileID, hint))
		}
		items = append(items, database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: evidenceKindDirectoryContext, EvidenceSource: "work_unit", EvidenceKey: "folder_shape", EvidenceValue: strings.TrimSpace(unit.FolderShape), Strength: "strong"})
	}
	return items
}

func contextEvidenceItem(fileID uint, hint ContextEvidence) database.RecognitionEvidence {
	return database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: evidenceKindDirectoryContext, EvidenceSource: strings.TrimSpace(hint.Source), EvidenceKey: firstNonEmptyRecognitionString(hint.Assignment, hint.TargetKey, "context"), EvidenceValue: firstNonEmptyRecognitionString(hint.TargetKey, hint.Assignment, hint.ReviewState), Strength: "medium", PayloadJSON: mustJSON(hint)}
}

func firstNonEmptyRecognitionString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
```

- [ ] **Step 4: Run evidence tests**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run TestCollectWorkUnitEvidenceKeepsSignalSidecarAndContextEvidence -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mibo-media-server/internal/recognition/evidence_collector.go mibo-media-server/internal/recognition/evidence_collector_test.go
git commit -m "feat: collect recognition work unit evidence"
```

## Task 4: Add Candidate Generator

**Files:**
- Create: `mibo-media-server/internal/recognition/candidate_generator.go`
- Create: `mibo-media-server/internal/recognition/candidate_generator_test.go`

- [ ] **Step 1: Write failing candidate generator tests**

Create `mibo-media-server/internal/recognition/candidate_generator_test.go` with:

```go
package recognition

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestGenerateCandidatesForSeasonWorkUnitCreatesSeriesSeasonEpisodes(t *testing.T) {
	fileID := uint(1)
	unit := RecognitionWorkUnit{ScopePath: "/library/Show/Season 01", FolderShape: FolderShapeSeason, Files: []database.InventoryFile{{ID: fileID, StorageProvider: "local", StoragePath: "/library/Show/Season 01/Show.S01E02.mkv", ContentClass: "video", Status: "available"}}, FileSignals: map[uint]database.InventoryFileSignal{fileID: {InventoryFileID: &fileID, TitleCandidate: "Show", SeasonNumber: intPtrForTest(1), EpisodeNumber: intPtrForTest(2)}}}

	candidates := GenerateCandidatesForWorkUnit(unit)

	assertCandidate(t, candidates, "work:series:show", CandidateTypeWork)
	assertCandidate(t, candidates, "work:season:work:series:show:s01", CandidateTypeWork)
	assertCandidate(t, candidates, "episode:work:season:work:series:show:s01:e02", CandidateTypeEpisode)
}

func TestGenerateCandidatesForExtraDoesNotCreateMovieWork(t *testing.T) {
	fileID := uint(2)
	unit := RecognitionWorkUnit{ScopePath: "/library/Movie/extras", FolderShape: FolderShapeExtra, Files: []database.InventoryFile{{ID: fileID, StorageProvider: "local", StoragePath: "/library/Movie/extras/Trailer.mkv", ContentClass: "video", Status: "available"}}, FileSignals: map[uint]database.InventoryFileSignal{fileID: {InventoryFileID: &fileID, TitleCandidate: "Trailer", Role: "trailer"}}}

	candidates := GenerateCandidatesForWorkUnit(unit)

	for _, candidate := range candidates {
		if candidate.CandidateType == CandidateTypeWork && candidate.CandidateRole == WorkKindMovie {
			t.Fatalf("extra folder must not create movie work candidate: %#v", candidates)
		}
	}
	assertCandidate(t, candidates, "playable_resource:local:path:/library/Movie/extras/Trailer.mkv", CandidateTypePlayableResource)
}

func assertCandidate(t *testing.T, candidates []database.RecognitionCandidate, key string, candidateType string) {
	t.Helper()
	for _, candidate := range candidates {
		if candidate.CandidateKey == key && candidate.CandidateType == candidateType {
			return
		}
	}
	t.Fatalf("missing candidate %s/%s in %#v", candidateType, key, candidates)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run 'TestGenerateCandidates' -count=1
```

Expected: FAIL because `GenerateCandidatesForWorkUnit` is not defined.

- [ ] **Step 3: Implement candidate generation**

Create `mibo-media-server/internal/recognition/candidate_generator.go` with:

```go
package recognition

import (
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

func GenerateCandidatesForWorkUnit(unit RecognitionWorkUnit) []database.RecognitionCandidate {
	candidates := make([]database.RecognitionCandidate, 0, len(unit.Files)*5)
	seen := make(map[string]struct{})
	add := func(candidate database.RecognitionCandidate) {
		key := strings.TrimSpace(candidate.CandidateKey)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidate.CandidateKey = key
		if candidate.CanonicalKey == "" {
			candidate.CanonicalKey = key
		}
		candidates = append(candidates, candidate)
	}

	for _, file := range unit.Files {
		fileID := file.ID
		signal := unit.FileSignals[file.ID]
		seriesTitle := strings.TrimSpace(signal.TitleCandidate)
		if unit.FolderShape != FolderShapeExtra && signal.SeasonNumber != nil && signal.EpisodeNumber != nil && seriesTitle != "" {
			seriesKey := WorkKey(WorkInput{Kind: WorkKindSeries, Title: seriesTitle})
			seasonKey := WorkKey(WorkInput{Kind: WorkKindSeason, Title: seriesTitle, ParentKey: seriesKey, SeasonNumber: signal.SeasonNumber})
			episodeKey := EpisodeKey(EpisodeInput{SeriesTitle: seriesTitle, SeasonNumber: *signal.SeasonNumber, EpisodeNumber: *signal.EpisodeNumber, SeasonKey: seasonKey})
			add(database.RecognitionCandidate{CandidateKey: seriesKey, CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeries, PrimaryInventoryID: &fileID, EvidenceJSON: mustJSON(map[string]any{"title": seriesTitle})})
			add(database.RecognitionCandidate{CandidateKey: seasonKey, CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeason, ParentCandidateKey: seriesKey, PrimaryInventoryID: &fileID, EvidenceJSON: mustJSON(map[string]any{"title": seriesTitle, "season_number": *signal.SeasonNumber})})
			add(database.RecognitionCandidate{CandidateKey: episodeKey, CandidateType: CandidateTypeEpisode, CandidateRole: WorkKindEpisode, ParentCandidateKey: seasonKey, PrimaryInventoryID: &fileID, EvidenceJSON: mustJSON(map[string]any{"title": seriesTitle, "season_number": *signal.SeasonNumber, "episode_number": *signal.EpisodeNumber})})
		} else if unit.FolderShape != FolderShapeExtra && seriesTitle != "" {
			add(database.RecognitionCandidate{CandidateKey: WorkKey(WorkInput{Kind: WorkKindMovie, Title: seriesTitle, Year: signal.Year}), CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, PrimaryInventoryID: &fileID, EvidenceJSON: mustJSON(map[string]any{"title": seriesTitle, "year": signal.Year})})
		}
		add(database.RecognitionCandidate{CandidateKey: PlayableResourceKey(file.StorageProvider, file.StoragePath), CandidateType: CandidateTypePlayableResource, ParentCandidateKey: firstParentCandidateKey(candidates), PrimaryInventoryID: &fileID, ResourceShape: resourceShapeForSignal(signal), EvidenceJSON: mustJSON(map[string]any{"storage_path": file.StoragePath, "role": signal.Role})})
	}
	return candidates
}

func resourceShapeForSignal(signal database.InventoryFileSignal) string {
	if len(inventorySignalEpisodeNumbers(signal)) > 1 {
		return ResourceKindMultiEpisode
	}
	return ResourceKindSingleFile
}
```

- [ ] **Step 4: Run candidate tests**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run 'TestGenerateCandidates' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mibo-media-server/internal/recognition/candidate_generator.go mibo-media-server/internal/recognition/candidate_generator_test.go
git commit -m "feat: generate recognition candidates from work units"
```

## Task 5: Add Hard Constraint Pruning

**Files:**
- Create: `mibo-media-server/internal/recognition/constraints.go`
- Create: `mibo-media-server/internal/recognition/constraints_test.go`

- [ ] **Step 1: Write failing constraint tests**

Create `mibo-media-server/internal/recognition/constraints_test.go` with:

```go
package recognition

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestApplyHardConstraintsRejectsMovieCandidateWhenEpisodeEvidenceExists(t *testing.T) {
	fileID := uint(1)
	candidates := []database.RecognitionCandidate{
		{CandidateKey: "work:movie:show:0", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, PrimaryInventoryID: &fileID},
		{CandidateKey: "episode:work:season:work:series:show:s01:e02", CandidateType: CandidateTypeEpisode, CandidateRole: WorkKindEpisode, PrimaryInventoryID: &fileID},
	}
	evidence := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceKey: "season_number", EvidenceValue: "1"}, {InventoryFileID: &fileID, EvidenceKey: "episode_number", EvidenceValue: "2"}}

	result := ApplyHardConstraints(candidates, evidence)

	if !result.IsRejected("work:movie:show:0") || result.IsRejected("episode:work:season:work:series:show:s01:e02") {
		t.Fatalf("expected movie rejected and episode kept, got %#v", result)
	}
}

func TestApplyHardConstraintsDemotesExtraResourceToReview(t *testing.T) {
	fileID := uint(2)
	candidates := []database.RecognitionCandidate{{CandidateKey: "playable_resource:local:path:/library/Movie/extras/Trailer.mkv", CandidateType: CandidateTypePlayableResource, PrimaryInventoryID: &fileID}}
	evidence := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceKey: "role", EvidenceValue: "trailer"}}

	result := ApplyHardConstraints(candidates, evidence)

	if result.RequiredOutcome("playable_resource:local:path:/library/Movie/extras/Trailer.mkv") != DecisionOutcomeReviewRequired {
		t.Fatalf("expected trailer resource review_required, got %#v", result)
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run 'TestApplyHardConstraints' -count=1
```

Expected: FAIL because `ApplyHardConstraints` is not defined.

- [ ] **Step 3: Implement constraints**

Create `mibo-media-server/internal/recognition/constraints.go` with:

```go
package recognition

import (
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

type ConstraintResult struct {
	RejectedCandidates map[string]string
	RequiredOutcomes   map[string]string
}

func (r ConstraintResult) IsRejected(candidateKey string) bool {
	_, ok := r.RejectedCandidates[strings.TrimSpace(candidateKey)]
	return ok
}

func (r ConstraintResult) RequiredOutcome(candidateKey string) string {
	return r.RequiredOutcomes[strings.TrimSpace(candidateKey)]
}

func ApplyHardConstraints(candidates []database.RecognitionCandidate, evidence []database.RecognitionEvidence) ConstraintResult {
	result := ConstraintResult{RejectedCandidates: map[string]string{}, RequiredOutcomes: map[string]string{}}
	evidenceByFile := evidenceKeysByFile(evidence)
	for _, candidate := range candidates {
		if candidate.PrimaryInventoryID == nil {
			continue
		}
		keys := evidenceByFile[*candidate.PrimaryInventoryID]
		if candidate.CandidateType == CandidateTypeWork && candidate.CandidateRole == WorkKindMovie && keys["season_number"] && keys["episode_number"] {
			result.RejectedCandidates[candidate.CandidateKey] = "episode evidence excludes movie work candidate"
		}
		if candidate.CandidateType == CandidateTypePlayableResource && (keys["role:trailer"] || keys["role:sample"] || keys["role:extra"]) {
			result.RequiredOutcomes[candidate.CandidateKey] = DecisionOutcomeReviewRequired
		}
	}
	return result
}

func evidenceKeysByFile(items []database.RecognitionEvidence) map[uint]map[string]bool {
	result := make(map[uint]map[string]bool)
	for _, item := range items {
		if item.InventoryFileID == nil {
			continue
		}
		fileID := *item.InventoryFileID
		if result[fileID] == nil {
			result[fileID] = make(map[string]bool)
		}
		key := strings.TrimSpace(item.EvidenceKey)
		value := strings.ToLower(strings.TrimSpace(item.EvidenceValue))
		result[fileID][key] = true
		if key == "role" && value != "" {
			result[fileID]["role:"+value] = true
		}
	}
	return result
}
```

- [ ] **Step 4: Run constraint tests**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run 'TestApplyHardConstraints' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mibo-media-server/internal/recognition/constraints.go mibo-media-server/internal/recognition/constraints_test.go
git commit -m "feat: prune recognition candidates with hard constraints"
```

## Task 6: Add Graph Inference

**Files:**
- Create: `mibo-media-server/internal/recognition/graph_inference.go`
- Create: `mibo-media-server/internal/recognition/graph_inference_test.go`

- [ ] **Step 1: Write failing graph inference tests**

Create `mibo-media-server/internal/recognition/graph_inference_test.go` with:

```go
package recognition

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestInferConsistentCandidateGraphKeepsParentChain(t *testing.T) {
	candidates := []database.RecognitionCandidate{
		{CandidateKey: "work:series:show", CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeries},
		{CandidateKey: "work:season:work:series:show:s01", CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeason, ParentCandidateKey: "work:series:show"},
		{CandidateKey: "episode:work:season:work:series:show:s01:e01", CandidateType: CandidateTypeEpisode, CandidateRole: WorkKindEpisode, ParentCandidateKey: "work:season:work:series:show:s01"},
	}

	result := InferConsistentCandidateGraph(candidates, ConstraintResult{})

	if len(result.AcceptedCandidates) != 3 || !result.Accepted("work:series:show") || !result.Accepted("work:season:work:series:show:s01") || !result.Accepted("episode:work:season:work:series:show:s01:e01") {
		t.Fatalf("expected full parent chain accepted, got %#v", result)
	}
}

func TestInferConsistentCandidateGraphDropsOrphanEpisode(t *testing.T) {
	candidates := []database.RecognitionCandidate{{CandidateKey: "episode:missing:e01", CandidateType: CandidateTypeEpisode, CandidateRole: WorkKindEpisode, ParentCandidateKey: "work:season:missing:s01"}}

	result := InferConsistentCandidateGraph(candidates, ConstraintResult{})

	if result.Accepted("episode:missing:e01") {
		t.Fatalf("expected orphan episode rejected, got %#v", result)
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run 'TestInferConsistentCandidateGraph' -count=1
```

Expected: FAIL because `InferConsistentCandidateGraph` is not defined.

- [ ] **Step 3: Implement graph inference**

Create `mibo-media-server/internal/recognition/graph_inference.go` with:

```go
package recognition

import "github.com/atlan/mibo-media-server/internal/database"

type InferenceResult struct {
	AcceptedCandidates []database.RecognitionCandidate
	RejectedReasons    map[string]string
}

func (r InferenceResult) Accepted(candidateKey string) bool {
	for _, candidate := range r.AcceptedCandidates {
		if candidate.CandidateKey == candidateKey {
			return true
		}
	}
	return false
}

func InferConsistentCandidateGraph(candidates []database.RecognitionCandidate, constraints ConstraintResult) InferenceResult {
	result := InferenceResult{RejectedReasons: map[string]string{}}
	byKey := make(map[string]database.RecognitionCandidate, len(candidates))
	for _, candidate := range candidates {
		byKey[candidate.CandidateKey] = candidate
	}
	for _, candidate := range candidates {
		if constraints.IsRejected(candidate.CandidateKey) {
			result.RejectedReasons[candidate.CandidateKey] = constraints.RejectedCandidates[candidate.CandidateKey]
			continue
		}
		if candidate.ParentCandidateKey != "" {
			if _, ok := byKey[candidate.ParentCandidateKey]; !ok {
				result.RejectedReasons[candidate.CandidateKey] = "missing parent candidate"
				continue
			}
		}
		result.AcceptedCandidates = append(result.AcceptedCandidates, candidate)
	}
	return result
}
```

- [ ] **Step 4: Run graph inference tests**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run 'TestInferConsistentCandidateGraph' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mibo-media-server/internal/recognition/graph_inference.go mibo-media-server/internal/recognition/graph_inference_test.go
git commit -m "feat: infer consistent recognition candidate graphs"
```

## Task 7: Add Lightweight Scoring

**Files:**
- Create: `mibo-media-server/internal/recognition/scoring.go`
- Create: `mibo-media-server/internal/recognition/scoring_test.go`

- [ ] **Step 1: Write failing scoring test**

Create `mibo-media-server/internal/recognition/scoring_test.go` with:

```go
package recognition

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestScoreCandidatesUsesEvidenceAfterConstraints(t *testing.T) {
	fileID := uint(1)
	candidates := []database.RecognitionCandidate{{CandidateKey: "work:movie:movie-a:2024", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, PrimaryInventoryID: &fileID}}
	evidence := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceKey: "title", Strength: "medium"}, {InventoryFileID: &fileID, EvidenceKey: "year", Strength: "medium"}, {InventoryFileID: &fileID, EvidenceKey: "folder_shape", Strength: "strong"}}

	scores := ScoreCandidates(candidates, evidence)

	if scores["work:movie:movie-a:2024"] < 0.70 {
		t.Fatalf("expected confident movie score, got %#v", scores)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run TestScoreCandidatesUsesEvidenceAfterConstraints -count=1
```

Expected: FAIL because `ScoreCandidates` is not defined.

- [ ] **Step 3: Implement scoring**

Create `mibo-media-server/internal/recognition/scoring.go` with:

```go
package recognition

import "github.com/atlan/mibo-media-server/internal/database"

func ScoreCandidates(candidates []database.RecognitionCandidate, evidence []database.RecognitionEvidence) map[string]float64 {
	evidenceByFile := evidenceKeysByFile(evidence)
	scores := make(map[string]float64, len(candidates))
	for _, candidate := range candidates {
		score := 0.25
		if candidate.PrimaryInventoryID != nil {
			keys := evidenceByFile[*candidate.PrimaryInventoryID]
			if keys["title"] {
				score += 0.20
			}
			if keys["year"] {
				score += 0.15
			}
			if keys["season_number"] {
				score += 0.15
			}
			if keys["episode_number"] {
				score += 0.15
			}
			if keys["folder_shape"] {
				score += 0.20
			}
		}
		if score > 1.0 {
			score = 1.0
		}
		scores[candidate.CandidateKey] = score
	}
	return scores
}
```

- [ ] **Step 4: Run scoring tests**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run TestScoreCandidatesUsesEvidenceAfterConstraints -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mibo-media-server/internal/recognition/scoring.go mibo-media-server/internal/recognition/scoring_test.go
git commit -m "feat: score recognition candidates after inference"
```

## Task 8: Add Decision Engine

**Files:**
- Create: `mibo-media-server/internal/recognition/decision_engine.go`
- Create: `mibo-media-server/internal/recognition/decision_engine_test.go`
- Modify: `mibo-media-server/internal/recognition/resolver.go`

- [ ] **Step 1: Write failing decision engine tests**

Create `mibo-media-server/internal/recognition/decision_engine_test.go` with:

```go
package recognition

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestBuildKernelDecisionsAcceptsHighConfidenceCandidate(t *testing.T) {
	candidate := database.RecognitionCandidate{ID: 1, CandidateKey: "work:movie:movie-a:2024", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie}
	result := BuildKernelDecisions(database.RecognitionManifest{ID: 10}, InferenceResult{AcceptedCandidates: []database.RecognitionCandidate{candidate}}, ConstraintResult{}, map[string]float64{candidate.CandidateKey: 0.85})

	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeAccepted || result.Decisions[0].Reason == "" {
		t.Fatalf("expected accepted explained decision, got %#v", result)
	}
}

func TestBuildKernelDecisionsSendsCloseOrLowConfidenceToReview(t *testing.T) {
	candidate := database.RecognitionCandidate{ID: 1, CandidateKey: "work:movie:ambiguous:0", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie}
	result := BuildKernelDecisions(database.RecognitionManifest{ID: 10}, InferenceResult{AcceptedCandidates: []database.RecognitionCandidate{candidate}}, ConstraintResult{}, map[string]float64{candidate.CandidateKey: 0.55})

	if len(result.Decisions) != 1 || result.Decisions[0].Outcome != DecisionOutcomeReviewRequired {
		t.Fatalf("expected review_required decision, got %#v", result)
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run 'TestBuildKernelDecisions' -count=1
```

Expected: FAIL because `BuildKernelDecisions` is not defined.

- [ ] **Step 3: Implement decision engine**

Create `mibo-media-server/internal/recognition/decision_engine.go` with:

```go
package recognition

import "github.com/atlan/mibo-media-server/internal/database"

const recognitionAcceptThreshold = 0.75

func BuildKernelDecisions(manifest database.RecognitionManifest, inference InferenceResult, constraints ConstraintResult, scores map[string]float64) ResolveResult {
	result := ResolveResult{}
	for _, candidate := range inference.AcceptedCandidates {
		outcome := constraints.RequiredOutcome(candidate.CandidateKey)
		reason := "candidate selected by recognition kernel"
		if outcome == "" {
			if scores[candidate.CandidateKey] >= recognitionAcceptThreshold {
				outcome = DecisionOutcomeAccepted
				reason = "candidate passed hard constraints, graph inference, and score threshold"
			} else {
				outcome = DecisionOutcomeReviewRequired
				reason = "candidate passed hard constraints but score requires review"
			}
		}
		candidateID := candidate.ID
		result.Decisions = append(result.Decisions, database.RecognitionDecision{ManifestID: manifest.ID, CandidateID: &candidateID, DecisionType: "kernel_decision", Outcome: outcome, TargetKind: candidate.CandidateType, TargetKey: candidate.CandidateKey, TargetMetadataID: candidate.TargetMetadataID, TargetResourceID: candidate.TargetResourceID, Confidence: floatPtr(scores[candidate.CandidateKey]), Reason: reason, EvidenceJSON: candidate.EvidenceJSON})
	}
	for key, reason := range inference.RejectedReasons {
		result.Decisions = append(result.Decisions, database.RecognitionDecision{ManifestID: manifest.ID, DecisionType: "kernel_rejection", Outcome: DecisionOutcomeRejected, TargetKey: key, Reason: reason})
	}
	return result
}

func floatPtr(v float64) *float64 { return &v }
```

- [ ] **Step 4: Run decision tests**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run 'TestBuildKernelDecisions' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mibo-media-server/internal/recognition/decision_engine.go mibo-media-server/internal/recognition/decision_engine_test.go
git commit -m "feat: decide recognition outcomes from kernel inference"
```

## Task 9: Wire Kernel Behind ConstructGraphFromInventory

**Files:**
- Modify: `mibo-media-server/internal/recognition/graph_constructor.go`
- Create: `mibo-media-server/internal/recognition/kernel_pipeline_test.go`

- [ ] **Step 1: Write failing kernel pipeline test**

Create `mibo-media-server/internal/recognition/kernel_pipeline_test.go` with:

```go
package recognition

import "testing"

func TestConstructGraphFromInventoryUsesKernelLayersForGoldenFixtures(t *testing.T) {
	for _, fixture := range goldenRecognitionFixtures() {
		t.Run(fixture.Name, func(t *testing.T) {
			output := ConstructGraphFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, RootPath: fixture.LibraryRoot, ScopePath: fixture.LibraryRoot, StorageProvider: "local"}, Files: fixture.Files, FileSignals: signalsByGoldenFixtureFileID(fixture)})
			if len(output.Candidates) == 0 || len(output.Evidence) == 0 {
				t.Fatalf("expected candidates and evidence, got %#v", output)
			}
			for _, key := range fixture.Expected.RequiredEvidenceKeys {
				if !hasAnyEvidenceKey(output.Evidence, key) {
					t.Fatalf("expected evidence key %s in %#v", key, output.Evidence)
				}
			}
		})
	}
}

func signalsByGoldenFixtureFileID(fixture recognitionGoldenFixture) map[uint]database.InventoryFileSignal {
	return fixture.Signals
}

func hasAnyEvidenceKey(items []database.RecognitionEvidence, key string) bool {
	for _, item := range items {
		if item.EvidenceKey == key {
			return true
		}
	}
	return false
}
```

Add the missing import to the test file:

```go
import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)
```

- [ ] **Step 2: Run test to verify current pipeline gap**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run TestConstructGraphFromInventoryUsesKernelLayersForGoldenFixtures -count=1
```

Expected: FAIL until the constructor is wired to collect work-unit evidence and candidates consistently for all fixtures.

- [ ] **Step 3: Replace graph constructor orchestration**

Modify `mibo-media-server/internal/recognition/graph_constructor.go` to:

```go
package recognition

import "github.com/atlan/mibo-media-server/internal/database"

type GraphConstructInput = ManifestBuildInput

type GraphConstructOutput struct {
	ManifestScope             ManifestScope
	MediaGraphNodes           []database.MediaGraphNode
	MediaGraphEdges           []database.MediaGraphEdge
	MediaGraphClassifications []database.MediaGraphClassification
	Candidates                []database.RecognitionCandidate
	Evidence                  []database.RecognitionEvidence
}

func ConstructGraphFromInventory(input GraphConstructInput) GraphConstructOutput {
	units := BuildRecognitionWorkUnits(input)
	candidates := make([]database.RecognitionCandidate, 0)
	evidence := make([]database.RecognitionEvidence, 0)
	for _, unit := range units {
		evidence = append(evidence, CollectWorkUnitEvidence(unit)...)
		candidates = append(candidates, GenerateCandidatesForWorkUnit(unit)...)
	}
	graph := mediaGraphFromKernelCandidates(input, candidates)
	return GraphConstructOutput{ManifestScope: input.Scope, MediaGraphNodes: graphNodesFromMediaGraph(graph), MediaGraphEdges: graphEdgesFromMediaGraph(graph), MediaGraphClassifications: graphClassificationsFromMediaGraph(graph), Candidates: candidates, Evidence: evidence}
}

func mediaGraphFromKernelCandidates(input GraphConstructInput, candidates []database.RecognitionCandidate) mediaGraph {
	graph := mediaGraph{scope: input.Scope}
	for _, candidate := range candidates {
		graph.nodes = append(graph.nodes, mediaGraphNode{key: candidate.CandidateKey, kind: candidate.CandidateType, parentKey: candidate.ParentCandidateKey, fileID: candidate.PrimaryInventoryID})
		if candidate.ParentCandidateKey != "" {
			graph.edges = append(graph.edges, mediaGraphEdge{fromKey: candidate.ParentCandidateKey, toKey: candidate.CandidateKey, relation: "contains"})
		}
	}
	return graph
}
```

If the internal `mediaGraph`, `mediaGraphNode`, or `mediaGraphEdge` helper names differ, adapt this step by reusing the existing graph helper types in `manifest_builder.go`; do not change the public `ConstructGraphFromInventory` signature.

- [ ] **Step 4: Run recognition package tests**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -count=1
```

Expected: PASS. Existing resolver/materializer tests should still pass.

- [ ] **Step 5: Commit**

```bash
git add mibo-media-server/internal/recognition/graph_constructor.go mibo-media-server/internal/recognition/kernel_pipeline_test.go
git commit -m "feat: route recognition graph construction through kernel"
```

## Task 10: Integrate Kernel Decisions With Resolver

**Files:**
- Modify: `mibo-media-server/internal/recognition/resolver.go`
- Modify: `mibo-media-server/internal/recognition/resolver_test.go`

- [ ] **Step 1: Add resolver regression test for kernel outcomes**

Append to `mibo-media-server/internal/recognition/resolver_test.go`:

```go
func TestResolverUsesKernelDecisionWhenNoManualRuleMatches(t *testing.T) {
	fileID := uint(7)
	candidate := database.RecognitionCandidate{ID: 10, CandidateKey: "work:movie:movie:2024", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, CanonicalKey: "work:movie:movie:2024", PrimaryInventoryID: &fileID}
	evidence := []database.RecognitionEvidence{{InventoryFileID: &fileID, EvidenceKey: "title", EvidenceValue: "Movie"}, {InventoryFileID: &fileID, EvidenceKey: "year", EvidenceValue: "2024"}, {InventoryFileID: &fileID, EvidenceKey: "folder_shape", EvidenceValue: FolderShapeMovie}}

	result := NewResolver(nil).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: []database.RecognitionCandidate{candidate}, Evidence: evidence})

	if len(result.Decisions) != 1 || result.Decisions[0].DecisionType != "kernel_decision" || result.Decisions[0].Outcome != DecisionOutcomeAccepted {
		t.Fatalf("expected accepted kernel decision, got %#v", result.Decisions)
	}
}
```

- [ ] **Step 2: Run resolver test to verify failure**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run TestResolverUsesKernelDecisionWhenNoManualRuleMatches -count=1
```

Expected: FAIL because `Resolver.Resolve` still uses the old local gate fallback.

- [ ] **Step 3: Update resolver default path**

Modify `Resolver.Resolve` in `mibo-media-server/internal/recognition/resolver.go` so manual rules and blocking conflicts remain first, then unmatched candidates are passed through the kernel helpers:

```go
func (r *Resolver) Resolve(graph ManifestGraph) ResolveResult {
	result := ResolveResult{Conflicts: detectBlockingConflicts(graph)}
	manualCandidateKeys := make(map[string]struct{})
	remaining := make([]database.RecognitionCandidate, 0, len(graph.Candidates))
	for _, candidate := range graph.Candidates {
		if conflict, ok := blockingConflictForCandidate(candidate, result.Conflicts); ok {
			result.Decisions = append(result.Decisions, database.RecognitionDecision{ManifestID: graph.Manifest.ID, CandidateID: uintPtr(candidate.ID), DecisionType: "resolver_conflict", Outcome: DecisionOutcomeBlockedConflict, TargetKind: candidate.CandidateType, TargetKey: candidate.CandidateKey, Confidence: candidate.Confidence, Reason: conflict.Reason, ConflictsJSON: mustJSON(conflict)})
			manualCandidateKeys[candidate.CandidateKey] = struct{}{}
			continue
		}
		if rule, ok := r.matchingRule(candidate); ok {
			decision, conflict := decisionFromRule(graph.Manifest.ID, candidate, rule)
			if conflict.ConflictKey != "" {
				result.Conflicts = append(result.Conflicts, conflict)
			}
			result.Decisions = append(result.Decisions, decision)
			manualCandidateKeys[candidate.CandidateKey] = struct{}{}
			continue
		}
		remaining = append(remaining, candidate)
	}
	constraints := ApplyHardConstraints(remaining, graph.Evidence)
	inference := InferConsistentCandidateGraph(remaining, constraints)
	scores := ScoreCandidates(inference.AcceptedCandidates, graph.Evidence)
	kernelResult := BuildKernelDecisions(graph.Manifest, inference, constraints, scores)
	result.Decisions = append(result.Decisions, kernelResult.Decisions...)
	result.Conflicts = append(result.Conflicts, kernelResult.Conflicts...)
	_ = manualCandidateKeys
	return result
}
```

After this change, remove unused old helper functions only if the compiler confirms they are unused in the package.

- [ ] **Step 4: Run resolver tests**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run 'TestResolver' -count=1
```

Expected: PASS. If old tests assert `resolver_gate`, update them only when the kernel decision preserves the same outcome and improves explanation.

- [ ] **Step 5: Commit**

```bash
git add mibo-media-server/internal/recognition/resolver.go mibo-media-server/internal/recognition/resolver_test.go
git commit -m "feat: resolve recognition decisions through kernel"
```

## Task 11: Preserve Library Manifest Persistence Boundary

**Files:**
- Modify: `mibo-media-server/internal/library/recognition_manifest.go`
- Modify: `mibo-media-server/internal/library/recognition_manifest_test.go`

- [ ] **Step 1: Add manifest integration regression test**

Append to `mibo-media-server/internal/library/recognition_manifest_test.go`:

```go
func TestPersistRecognitionManifestForFilesPersistsKernelCandidatesAndEvidence(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	library := database.Library{ID: 1, MediaSourceID: 1, StorageProvider: "local", RootPath: "/library"}
	if err := db.Create(&library).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	file := database.InventoryFile{ID: 1, LibraryID: library.ID, StorageProvider: "local", StoragePath: "/library/Movie A (2024)/Movie A.2024.mkv", StableIdentityKey: "local:movie-a", ContentClass: SourceContentClassVideo, Status: "available"}
	if err := db.Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}

	manifest, err := svc.persistRecognitionManifestForFiles(ctx, library, []database.InventoryFile{file}, library.RootPath)
	if err != nil {
		t.Fatalf("persist recognition manifest: %v", err)
	}
	var candidates []database.RecognitionCandidate
	if err := db.Where("manifest_id = ?", manifest.ID).Find(&candidates).Error; err != nil {
		t.Fatalf("load candidates: %v", err)
	}
	var evidence []database.RecognitionEvidence
	if err := db.Where("manifest_id = ?", manifest.ID).Find(&evidence).Error; err != nil {
		t.Fatalf("load evidence: %v", err)
	}
	if len(candidates) == 0 || len(evidence) == 0 {
		t.Fatalf("expected kernel candidates and evidence, candidates=%#v evidence=%#v", candidates, evidence)
	}
}
```

- [ ] **Step 2: Run manifest test**

Run:

```bash
cd mibo-media-server && go test ./internal/library -run TestPersistRecognitionManifestForFilesPersistsKernelCandidatesAndEvidence -count=1
```

Expected: PASS if `ConstructGraphFromInventory` remains behind the existing persistence entrypoint. If it fails, fix `recognition_manifest.go` without moving SQL out of `recognition.Repository`.

- [ ] **Step 3: Keep entrypoint small**

Refactor `persistRecognitionManifestForFiles` only enough to make the input preparation readable:

```go
input := recognition.GraphConstructInput{
	Scope:            recognition.ManifestScope{LibraryID: library.ID, MediaSourceID: library.MediaSourceID, StorageProvider: storageProvider, RootPath: rootPath, ScopePath: scopePath, ClassifierVersion: settings.ClassifierVersion},
	Files:            files,
	FileSignals:      indexedSignals,
	SidecarsByFileID: sidecarsByFileID,
	SidecarHints:     sidecarHints,
	ContextEvidence:  contextEvidence,
	ExcludedFileIDs:  excludedFileIDs,
}
```

Do not add workflow, projection, or materialization logic to this method.

- [ ] **Step 4: Run library recognition tests**

Run:

```bash
cd mibo-media-server && go test ./internal/library -run 'TestPersistRecognitionManifest|TestApplyRecognitionFallbackPoster' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mibo-media-server/internal/library/recognition_manifest.go mibo-media-server/internal/library/recognition_manifest_test.go
git commit -m "refactor: keep recognition manifest persistence boundary"
```

## Task 12: Preserve Workflow And Projection Shells

**Files:**
- Modify: `mibo-media-server/internal/library/workflow_test.go`
- Modify: `mibo-media-server/internal/catalog/metadata_projection_test.go` if projection contract needs a regression case

- [ ] **Step 1: Strengthen workflow stage-order test**

In `mibo-media-server/internal/library/workflow_test.go`, update `TestRunWorkflowScanLibraryPathQueuesRecognitionResolveTasksBeforeProjection` to assert task stages in order after running scan:

```go
var tasks []database.WorkflowTask
if err := db.WithContext(ctx).Where("run_id = ?", run.ID).Order("stage asc, id asc").Find(&tasks).Error; err != nil {
	t.Fatalf("load workflow tasks: %v", err)
}
seenRecognition := false
for _, task := range tasks {
	if task.TaskType == workflow.TaskTypeResolveRecognition {
		seenRecognition = true
	}
	if task.TaskType == workflow.TaskTypeRefreshProjection && !seenRecognition {
		t.Fatalf("projection task was queued before recognition resolve: %#v", tasks)
	}
}
if !seenRecognition {
	t.Fatalf("expected recognition resolve task before projection, got %#v", tasks)
}
```

- [ ] **Step 2: Run workflow shell test**

Run:

```bash
cd mibo-media-server && go test ./internal/library -run 'TestRunWorkflowScanLibraryPathQueuesRecognitionResolveTasksBeforeProjection|TestRunRecognitionResolveBatchSkipsMissingInventoryFiles' -count=1
```

Expected: PASS. If it fails because the new kernel changed task enqueue behavior, fix the recognition integration rather than weakening the test.

- [ ] **Step 3: Run projection tests**

Run:

```bash
cd mibo-media-server && go test ./internal/catalog -run 'Test.*Projection' -count=1
```

Expected: PASS. If a projection test fails, inspect materialized metadata/resource links from `Materializer` before changing projection code.

- [ ] **Step 4: Commit**

```bash
git add mibo-media-server/internal/library/workflow_test.go mibo-media-server/internal/catalog/metadata_projection_test.go
git commit -m "test: preserve recognition workflow projection ordering"
```

## Task 13: Add Dual-Run Comparison Harness

**Files:**
- Create: `mibo-media-server/internal/recognition/dual_run.go`
- Create: `mibo-media-server/internal/recognition/dual_run_test.go`

- [ ] **Step 1: Write failing dual-run comparison test**

Create `mibo-media-server/internal/recognition/dual_run_test.go` with:

```go
package recognition

import "testing"

func TestCompareRecognitionOutputsReportsDecisionDifferences(t *testing.T) {
	oldOutput := ResolveResult{Decisions: []database.RecognitionDecision{{TargetKey: "work:movie:a", Outcome: DecisionOutcomeReviewRequired}}}
	newOutput := ResolveResult{Decisions: []database.RecognitionDecision{{TargetKey: "work:movie:a", Outcome: DecisionOutcomeAccepted}}}

	diff := CompareRecognitionOutputs(oldOutput, newOutput)

	if len(diff.DecisionDiffs) != 1 || diff.DecisionDiffs[0].TargetKey != "work:movie:a" {
		t.Fatalf("expected decision diff, got %#v", diff)
	}
}
```

Add the missing import:

```go
import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)
```

- [ ] **Step 2: Run test to verify failure**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run TestCompareRecognitionOutputsReportsDecisionDifferences -count=1
```

Expected: FAIL because `CompareRecognitionOutputs` is not defined.

- [ ] **Step 3: Implement comparison harness**

Create `mibo-media-server/internal/recognition/dual_run.go` with:

```go
package recognition

import "github.com/atlan/mibo-media-server/internal/database"

type RecognitionOutputDiff struct {
	DecisionDiffs []RecognitionDecisionDiff
}

type RecognitionDecisionDiff struct {
	TargetKey  string
	OldOutcome string
	NewOutcome string
}

func CompareRecognitionOutputs(oldOutput ResolveResult, newOutput ResolveResult) RecognitionOutputDiff {
	oldByKey := decisionsByTargetKey(oldOutput.Decisions)
	newByKey := decisionsByTargetKey(newOutput.Decisions)
	diff := RecognitionOutputDiff{}
	for key, newDecision := range newByKey {
		oldDecision := oldByKey[key]
		if oldDecision.Outcome != newDecision.Outcome {
			diff.DecisionDiffs = append(diff.DecisionDiffs, RecognitionDecisionDiff{TargetKey: key, OldOutcome: oldDecision.Outcome, NewOutcome: newDecision.Outcome})
		}
	}
	return diff
}

func decisionsByTargetKey(decisions []database.RecognitionDecision) map[string]database.RecognitionDecision {
	result := make(map[string]database.RecognitionDecision, len(decisions))
	for _, decision := range decisions {
		result[decision.TargetKey] = decision
	}
	return result
}
```

- [ ] **Step 4: Run dual-run tests**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition -run TestCompareRecognitionOutputsReportsDecisionDifferences -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mibo-media-server/internal/recognition/dual_run.go mibo-media-server/internal/recognition/dual_run_test.go
git commit -m "feat: compare recognition engine outputs"
```

## Task 14: Full Regression And Acceptance Check

**Files:**
- No new files unless a failing test exposes a kernel integration bug.

- [ ] **Step 1: Run recognition tests**

Run:

```bash
cd mibo-media-server && go test ./internal/recognition/... -count=1
```

Expected: PASS.

- [ ] **Step 2: Run library tests**

Run:

```bash
cd mibo-media-server && go test ./internal/library/... -count=1
```

Expected: PASS.

- [ ] **Step 3: Run catalog tests**

Run:

```bash
cd mibo-media-server && go test ./internal/catalog/... -count=1
```

Expected: PASS.

- [ ] **Step 4: Run backend package tests**

Run:

```bash
cd mibo-media-server && go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 5: Check acceptance criteria manually**

Confirm these are true before calling the work done:

- Recognition core has explicit layers: work unit, evidence, candidate, constraint, inference, score, decision.
- Workflow scan -> recognition -> projection order is unchanged.
- Golden fixtures cover movie, season, multi-version, and extra/trailer cases.
- Decisions include explanations through `Reason`, `EvidenceJSON`, rejected reasons, or conflicts.
- New recognition output can be compared against old output through the dual-run harness.
- Materializer and projection tests pass without moving projection logic into recognition.

- [ ] **Step 6: Commit final fixes**

```bash
git add mibo-media-server/internal/recognition mibo-media-server/internal/library mibo-media-server/internal/catalog
git commit -m "test: validate recognition kernel redesign"
```

## Risks And Mitigations

- Baseline too weak: keep Task 1 first and do not tune scoring until fixtures describe expected outcomes.
- Rewrite too large: land each layer behind tests and preserve `ConstructGraphFromInventory` as the public recognition build seam.
- Projection regression: treat projection failures as materializer contract regressions unless projection itself is demonstrably wrong.
- Review-required rate does not improve: keep explanations explicit first; tune constraints/scoring only after golden and real-library comparisons show why review was triggered.
- Old helper removal causes regressions: remove helpers only when package tests and compiler confirm they are unused.

## Self-Review

- Spec coverage: the plan covers baseline/golden fixtures, work units, evidence, candidate generation, hard constraints, graph inference, scoring, decision output, manifest persistence, workflow/projection preservation, dual-run comparison, and full Go validation.
- Placeholder scan: no `TBD`, `TODO`, or unbounded “handle edge cases” steps remain. The only adaptive note is constrained to reusing existing private graph helper names if their exact local names differ during implementation.
- Type consistency: public model names match existing code: `ManifestBuildInput`, `ManifestScope`, `ManifestGraph`, `RecognitionCandidate`, `RecognitionEvidence`, `RecognitionDecision`, `ResolveResult`, `CandidateTypeWork`, `CandidateTypeEpisode`, `CandidateTypePlayableResource`, `WorkKindMovie`, `WorkKindSeries`, `WorkKindSeason`, `WorkKindEpisode`, and decision outcome constants.
