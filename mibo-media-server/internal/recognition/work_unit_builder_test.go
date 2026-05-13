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
