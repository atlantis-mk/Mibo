package recognition

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestCollectWorkUnitEvidenceKeepsSignalSidecarAndContextEvidence(t *testing.T) {
	fileID := uint(7)
	confidence := 0.92
	unit := RecognitionWorkUnit{
		ScopePath:   "/library/Show/Season 01",
		FolderShape: FolderShapeSeason,
		Files:       []database.InventoryFile{{ID: fileID, StoragePath: "/library/Show/Season 01/Show.S01E02.mkv", ContentClass: "video", Status: "available"}},
		FileSignals: map[uint]database.InventoryFileSignal{fileID: {InventoryFileID: &fileID, TitleCandidate: "Show", SeasonNumber: intPtrForTest(1), EpisodeNumber: intPtrForTest(2)}},
		SidecarsByFileID: map[uint][]database.InventoryFile{fileID: {{ID: 8, StoragePath: "/library/Show/Season 01/Show.S01E02.nfo", ContentClass: "metadata", Status: "available"}}},
		ContextEvidence: map[uint][]ContextEvidence{fileID: {{Source: "directory_reduction", Assignment: "episode_multi_version", ReviewState: "auto", Confidence: &confidence}}},
	}

	evidence := CollectWorkUnitEvidence(unit)

	if !hasEvidence(evidence, fileID, "title") || !hasEvidence(evidence, fileID, "season_number") || !hasEvidence(evidence, fileID, "episode_number") || !hasEvidence(evidence, fileID, "sidecar") || !hasEvidence(evidence, fileID, "folder_shape") || !hasContextAssignmentEvidence(evidence, fileID, confidence) {
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

func hasContextAssignmentEvidence(items []database.RecognitionEvidence, fileID uint, confidence float64) bool {
	for _, item := range items {
		if item.InventoryFileID == nil || *item.InventoryFileID != fileID {
			continue
		}
		if item.EvidenceSource == "directory_reduction" && item.EvidenceKey == "assignment" && item.EvidenceValue == "episode_multi_version" && item.Strength == "strong" && item.Confidence != nil && *item.Confidence == confidence {
			return true
		}
	}
	return false
}
