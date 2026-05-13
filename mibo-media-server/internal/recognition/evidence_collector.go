package recognition

import (
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

func CollectWorkUnitEvidence(unit RecognitionWorkUnit) []database.RecognitionEvidence {
	items := make([]database.RecognitionEvidence, 0, len(unit.Files)*8)
	for _, file := range unit.Files {
		fileID := file.ID
		items = append(items, inventoryEvidence(file, "work_unit")...)
		if signal, ok := unit.FileSignals[file.ID]; ok {
			items = append(items, signalEvidence(fileID, "file_signal", signal)...)
		}
		for _, sidecar := range unit.SidecarsByFileID[file.ID] {
			items = append(items, database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: evidenceKindSidecar, EvidenceSource: evidenceSourceSidecar, EvidenceKey: "sidecar", EvidenceValue: strings.TrimSpace(sidecar.StoragePath), Strength: "medium", PayloadJSON: mustJSON(sidecar)})
		}
		for _, hint := range unit.SidecarHints[file.ID] {
			items = append(items, sidecarHintEvidence(fileID, "work_unit", hint)...)
		}
		for _, hint := range unit.ContextEvidence[file.ID] {
			items = append(items, directoryContextEvidence(fileID, "work_unit", hint)...)
		}
		if reason := strings.TrimSpace(unit.ExcludedFileIDs[file.ID]); reason != "" {
			items = append(items, database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: evidenceKindScanExclusion, EvidenceSource: evidenceSourceExclusion, EvidenceKey: "scan_exclusion", EvidenceValue: reason, Strength: "strong"})
		}
		items = append(items, database.RecognitionEvidence{InventoryFileID: &fileID, EvidenceKind: evidenceKindDirectoryContext, EvidenceSource: "work_unit", EvidenceKey: "folder_shape", EvidenceValue: strings.TrimSpace(unit.FolderShape), Strength: "strong"})
	}
	return items
}
