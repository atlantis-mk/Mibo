package storageindex

import (
	"fmt"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestPlannerClassifiesCreateUpdateDeleteAndDirectoryDelete(t *testing.T) {
	now := time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)
	modified := now.Add(-time.Hour)
	planner := NewPlanner()
	planner.now = func() time.Time { return now }
	result := planner.Plan(PlanInput{
		LibraryID:   1,
		LibraryRoot: "/library",
		Previous: []database.StorageIndexEntry{
			entry("/library/changed.mkv", false, 100, modified, ""),
			entry("/library/deleted.mkv", false, 100, modified, ""),
			entry("/library/deleted-dir", true, 0, modified, ""),
		},
		Current: []database.StorageIndexEntry{
			entry("/library/changed.mkv", false, 200, modified, ""),
			entry("/library/new.mkv", false, 100, modified, ""),
		},
	})
	assertChangeKinds(t, result.Changes, map[string]int{ChangeKindUpdate: 1, ChangeKindCreate: 1, ChangeKindDelete: 1, ChangeKindDirectoryDelete: 1})
	if len(result.Plans) != 1 || result.Plans[0].RootPath != "/library" {
		t.Fatalf("expected coalesced library refresh plan, got %#v", result.Plans)
	}
}

func TestPlannerDetectsStableIdentityMove(t *testing.T) {
	now := time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)
	planner := NewPlanner()
	planner.now = func() time.Time { return now }
	result := planner.Plan(PlanInput{
		LibraryID:   1,
		LibraryRoot: "/library",
		Previous:    []database.StorageIndexEntry{entry("/library/Old.mkv", false, 100, now.Add(-time.Hour), "stable-1")},
		Current:     []database.StorageIndexEntry{entry("/library/New.mkv", false, 100, now.Add(-time.Hour), "stable-1")},
	})
	assertChangeKinds(t, result.Changes, map[string]int{ChangeKindMove: 1})
	move := result.Changes[0]
	if !move.StableMatch || move.OldPath != "/library/Old.mkv" || move.Path != "/library/New.mkv" {
		t.Fatalf("unexpected move change: %#v", move)
	}
}

func TestPlannerFallsBackToDeleteCreateWithoutStableIdentity(t *testing.T) {
	now := time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)
	planner := NewPlanner()
	planner.now = func() time.Time { return now }
	result := planner.Plan(PlanInput{
		LibraryID:   1,
		LibraryRoot: "/library",
		Previous:    []database.StorageIndexEntry{entry("/library/Old.mkv", false, 100, now.Add(-time.Hour), "")},
		Current:     []database.StorageIndexEntry{entry("/library/New.mkv", false, 100, now.Add(-time.Hour), "")},
	})
	assertChangeKinds(t, result.Changes, map[string]int{ChangeKindCreate: 1, ChangeKindDelete: 1})
}

func TestPlannerDefersUnstableFiles(t *testing.T) {
	now := time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)
	planner := NewPlanner()
	planner.now = func() time.Time { return now }
	unstable := entry("/library/New.mkv", false, 100, now.Add(-5*time.Second), "")
	unstable.FirstObservedAt = now.Add(-5 * time.Second)
	unstable.LastObservedAt = now.Add(-5 * time.Second)
	result := planner.Plan(PlanInput{LibraryID: 1, LibraryRoot: "/library", Current: []database.StorageIndexEntry{unstable}})
	if len(result.Changes) != 1 || !result.Changes[0].Deferred || len(result.Plans) != 0 {
		t.Fatalf("expected deferred change without plan, got %#v", result)
	}
}

func TestPlannerFallsBackToFullSyncForDispersedChanges(t *testing.T) {
	now := time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)
	planner := NewPlanner()
	planner.now = func() time.Time { return now }
	planner.maxScopedPaths = 3
	current := make([]database.StorageIndexEntry, 0, 4)
	for idx := 0; idx < 4; idx++ {
		current = append(current, entry(fmt.Sprintf("/library/dir-%d/Movie.mkv", idx), false, 100, now.Add(-time.Hour), ""))
	}
	result := planner.Plan(PlanInput{LibraryID: 1, LibraryRoot: "/library", Current: current})
	if len(result.Plans) != 1 || !result.Plans[0].FullSync || result.Plans[0].RootPath != "/library" {
		t.Fatalf("expected full sync fallback, got %#v", result.Plans)
	}
}

func entry(path string, isDir bool, size int64, modified time.Time, stable string) database.StorageIndexEntry {
	first := modified.Add(-time.Minute)
	last := modified
	return database.StorageIndexEntry{
		LibraryID:         1,
		StorageProvider:   "local",
		StoragePath:       path,
		IsDir:             isDir,
		SizeBytes:         size,
		ModifiedAt:        &modified,
		StableIdentityKey: stable,
		ObservationStatus: ObservationStatusPresent,
		FirstObservedAt:   first,
		LastObservedAt:    last,
	}
}

func assertChangeKinds(t *testing.T, changes []Change, expected map[string]int) {
	t.Helper()
	actual := make(map[string]int)
	for _, change := range changes {
		actual[change.Kind]++
	}
	if len(actual) != len(expected) {
		t.Fatalf("expected change kinds %#v, got %#v", expected, actual)
	}
	for kind, count := range expected {
		if actual[kind] != count {
			t.Fatalf("expected %d %s changes, got %#v", count, kind, actual)
		}
	}
}
