package recognition

import (
	"path"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

const (
	FolderShapeMovie      = "movie_folder"
	FolderShapeSeries     = "series_root"
	FolderShapeSeason     = "season_folder"
	FolderShapeCollection = "collection_folder"
	FolderShapeExtra      = "extra_folder"
	FolderShapeMixed      = "mixed_folder"
)

type RecognitionWorkUnit struct {
	ScopePath       string
	FolderShape     string
	Files           []database.InventoryFile
	FileSignals     map[uint]database.InventoryFileSignal
	SidecarsByFileID map[uint][]database.InventoryFile
	SidecarHints     map[uint][]SidecarHint
	ContextEvidence map[uint][]ContextEvidence
	ExcludedFileIDs map[uint]string
}

func BuildRecognitionWorkUnits(input ManifestBuildInput) []RecognitionWorkUnit {
	groups := make(map[string][]database.InventoryFile)
	for _, file := range input.Files {
		if !eligibleInventoryFile(file) {
			continue
		}
		scope := workUnitScopePath(file.StoragePath)
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
		units = append(units, RecognitionWorkUnit{ScopePath: key, FolderShape: inferFolderShape(key, files), Files: files, FileSignals: input.FileSignals, SidecarsByFileID: input.SidecarsByFileID, SidecarHints: input.SidecarHints, ContextEvidence: input.ContextEvidence, ExcludedFileIDs: input.ExcludedFileIDs})
	}
	return units
}

func workUnitScopePath(storagePath string) string {
	return path.Dir(strings.TrimSpace(storagePath))
}

func inferFolderShape(scopePath string, files []database.InventoryFile) string {
	base := strings.ToLower(path.Base(scopePath))
	if base == "extras" || base == "extra" || base == "samples" || base == "sample" || base == "trailers" || base == "trailer" {
		return FolderShapeExtra
	}
	if strings.HasPrefix(base, "season ") || strings.HasPrefix(base, "season.") || strings.HasPrefix(base, "s0") || strings.HasPrefix(base, "s1") {
		return FolderShapeSeason
	}
	return FolderShapeMovie
}
