package recognition

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

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
			result := NewResolver(nil).Resolve(ManifestGraph{Manifest: database.RecognitionManifest{ID: 1}, Candidates: output.Candidates, Evidence: output.Evidence})
			assertGoldenDecisions(t, fixture, result)
		})
	}
}

func TestConstructGraphFromInventoryBuildsKernelSeasonTopology(t *testing.T) {
	var seasonFixture recognitionGoldenFixture
	for _, fixture := range goldenRecognitionFixtures() {
		if fixture.Name == "standard season folder" {
			seasonFixture = fixture
			break
		}
	}
	output := ConstructGraphFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, RootPath: seasonFixture.LibraryRoot, ScopePath: seasonFixture.LibraryRoot, StorageProvider: "local"}, Files: seasonFixture.Files, FileSignals: signalsByGoldenFixtureFileID(seasonFixture)})

	seriesGroup := groupNodeKey(mediaGroupKindSeriesPackage, SeriesWorkKey("Show"))
	seasonGroup := groupNodeKey(mediaGroupKindSeasonPackage, SeasonWorkKey("Show", 1))
	episodeGroup := groupNodeKey(mediaGroupKindEpisodeRun, EpisodeKey(EpisodeInput{SeriesTitle: "Show", SeasonNumber: 1, EpisodeNumber: 1}))
	if !hasGraphEdge(output.MediaGraphEdges, seriesGroup, seasonGroup, "contains_group") {
		t.Fatalf("expected series group to contain season group, edges=%#v", output.MediaGraphEdges)
	}
	if !hasGraphEdge(output.MediaGraphEdges, seasonGroup, episodeGroup, "contains_group") {
		t.Fatalf("expected season group to contain episode group, edges=%#v", output.MediaGraphEdges)
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

func hasGraphEdge(edges []database.MediaGraphEdge, from string, to string, edgeKind string) bool {
	for _, edge := range edges {
		if edge.FromNodeKey == from && edge.ToNodeKey == to && edge.EdgeKind == edgeKind {
			return true
		}
	}
	return false
}
