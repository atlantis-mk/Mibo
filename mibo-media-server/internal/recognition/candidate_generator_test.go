package recognition

import (
	"strings"
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
	assertResourceParent(t, candidates, "playable_resource:local:path:/library/Show/Season 01/Show.S01E02.mkv", "episode:work:season:work:series:show:s01:e02")
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
	assertResourceParent(t, candidates, "playable_resource:local:path:/library/Movie/extras/Trailer.mkv", "")
}

func TestGenerateCandidatesForResourceUsesStableIdentity(t *testing.T) {
	fileID := uint(3)
	unit := RecognitionWorkUnit{ScopePath: "/library/Movie", FolderShape: FolderShapeMovie, Files: []database.InventoryFile{{ID: fileID, StorageProvider: "local", StoragePath: "/library/Movie/Movie.mkv", StableIdentityKey: "stable-movie", ContentClass: "video", Status: "available"}}, FileSignals: map[uint]database.InventoryFileSignal{fileID: {InventoryFileID: &fileID, TitleCandidate: "Movie"}}}

	candidates := GenerateCandidatesForWorkUnit(unit)

	assertCandidate(t, candidates, "playable_resource:local:stable:stable-movie", CandidateTypePlayableResource)
}

func TestGenerateCandidatesForMultiEpisodeResourceIncludesEpisodeKeys(t *testing.T) {
	fileID := uint(4)
	unit := RecognitionWorkUnit{ScopePath: "/library/Show", FolderShape: FolderShapeSeason, Files: []database.InventoryFile{{ID: fileID, StorageProvider: "local", StoragePath: "/library/Show/Show.S01E01-E02.mkv", ContentClass: "video", Status: "available"}}, FileSignals: map[uint]database.InventoryFileSignal{fileID: {InventoryFileID: &fileID, TitleCandidate: "Show", SeasonNumber: intPtrForTest(1), EpisodeNumber: intPtrForTest(1), EpisodeNumbersJSON: `[1,2]`}}}

	candidates := GenerateCandidatesForWorkUnit(unit)

	assertCandidate(t, candidates, "episode:work:season:work:series:show:s01:e01", CandidateTypeEpisode)
	assertCandidate(t, candidates, "episode:work:season:work:series:show:s01:e02", CandidateTypeEpisode)
	resource := candidateByKeyForTest(t, candidates, "playable_resource:local:path:/library/Show/Show.S01E01-E02.mkv")
	if resource.ResourceShape != ResourceKindMultiEpisode || !strings.Contains(resource.EvidenceJSON, "episode_keys") {
		t.Fatalf("expected multi-episode resource evidence, got %#v", resource)
	}
}

func TestGenerateCandidatesUsesSidecarEpisodeIdentity(t *testing.T) {
	fileID := uint(5)
	unit := RecognitionWorkUnit{
		ScopePath: "/library/Show/Season 01",
		FolderShape: FolderShapeSeason,
		Files: []database.InventoryFile{{ID: fileID, StorageProvider: "local", StoragePath: "/library/Show/Season 01/01.mkv", ContentClass: "video", Status: "available"}},
		FileSignals: map[uint]database.InventoryFileSignal{fileID: {InventoryFileID: &fileID, TitleCandidate: "01"}},
		SidecarHints: map[uint][]SidecarHint{fileID: {{SeriesTitle: "Show", SeasonNumber: intPtrForTest(1), EpisodeNumber: intPtrForTest(1)}}},
	}

	candidates := GenerateCandidatesForWorkUnit(unit)

	assertCandidate(t, candidates, "work:series:show", CandidateTypeWork)
	assertCandidate(t, candidates, "work:season:work:series:show:s01", CandidateTypeWork)
	assertCandidate(t, candidates, "episode:work:season:work:series:show:s01:e01", CandidateTypeEpisode)
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

func assertResourceParent(t *testing.T, candidates []database.RecognitionCandidate, key string, parentKey string) {
	t.Helper()
	candidate := candidateByKeyForTest(t, candidates, key)
	wantCanonicalKey := parentKey
	if wantCanonicalKey == "" {
		wantCanonicalKey = key
	}
	if candidate.ParentCandidateKey != parentKey || candidate.CanonicalKey != wantCanonicalKey {
		t.Fatalf("expected resource %s parent/canonical %q, got %#v", key, parentKey, candidate)
	}
}

func candidateByKeyForTest(t *testing.T, candidates []database.RecognitionCandidate, key string) database.RecognitionCandidate {
	t.Helper()
	for _, candidate := range candidates {
		if candidate.CandidateKey == key {
			return candidate
		}
	}
	t.Fatalf("missing candidate %s in %#v", key, candidates)
	return database.RecognitionCandidate{}
}
