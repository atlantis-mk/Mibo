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

	return []recognitionGoldenFixture{
		{
			Name:        "single movie folder",
			LibraryRoot: "/library",
			Files: []database.InventoryFile{{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie A (2024)/Movie A.2024.1080p.mkv", StableIdentityKey: "local:movie-a", ContentClass: "video", Status: "available", SizeBytes: 1024, ModifiedAt: &modified}},
			Signals: map[uint]database.InventoryFileSignal{1: {InventoryFileID: uintPtr(1), StoragePath: "/library/Movie A (2024)/Movie A.2024.1080p.mkv", TitleCandidate: "Movie A", Year: intPtr(2024), Quality: "1080p"}},
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
				2: {InventoryFileID: uintPtr(2), StoragePath: "/library/Show/Season 01/Show.S01E01.mkv", TitleCandidate: "Show", SeasonNumber: intPtr(1), EpisodeNumber: intPtr(1)},
				3: {InventoryFileID: uintPtr(3), StoragePath: "/library/Show/Season 01/Show.S01E02.mkv", TitleCandidate: "Show", SeasonNumber: intPtr(1), EpisodeNumber: intPtr(2)},
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
				4: {InventoryFileID: uintPtr(4), StoragePath: "/library/Movie B (2024)/Movie B.2024.1080p.mkv", TitleCandidate: "Movie B", Year: intPtr(2024), Quality: "1080p"},
				5: {InventoryFileID: uintPtr(5), StoragePath: "/library/Movie B (2024)/Movie B.2024.2160p.mkv", TitleCandidate: "Movie B", Year: intPtr(2024), Quality: "2160p"},
			},
			Expected: recognitionGoldenExpectation{AcceptedCandidateKeys: []string{"work:movie:movie-b:2024"}, RequiredEvidenceKeys: []string{"title", "year", "sibling_consistency"}},
		},
		{
			Name:        "extra does not become main work",
			LibraryRoot: "/library",
			Files: []database.InventoryFile{{ID: 6, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie C (2024)/extras/Movie C Trailer.mkv", ContentClass: "video", Status: "available", SizeBytes: 512, ModifiedAt: &modified}},
			Signals: map[uint]database.InventoryFileSignal{6: {InventoryFileID: uintPtr(6), StoragePath: "/library/Movie C (2024)/extras/Movie C Trailer.mkv", TitleCandidate: "Movie C Trailer", Role: "trailer", Year: intPtr(2024)}},
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
