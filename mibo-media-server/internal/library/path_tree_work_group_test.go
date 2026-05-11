package library

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
)

func TestNormalizedMovieWorkKeySuppressesReleaseHints(t *testing.T) {
	cases := []struct {
		name string
		path string
		want string
	}{
		{name: "dot release", path: "/movies/3.Iron.2004.1080p.Bluray.x265.10bit.DDP.5.1-MiniHD.mkv", want: "3 iron:2004"},
		{name: "hyphen release", path: "/movies/3-Iron.2004.1080p.BluRay.DTS.x265-10bit-TAGHD.mkv", want: "3 iron:2004"},
		{name: "chinese prefix folder", path: "/电影/合集5-2/空房间[简繁英字幕].3.Iron.2004.1080p.Bluray.x265.10bit.DDP.5.1-MiniHD/3.Iron.2004.1080p.Bluray.x265.10bit.DDP.5.1-MiniHD.mkv", want: "3 iron:2004"},
		{name: "edition and source", path: "/movies/Heat.1995.Directors.Cut.2160p.UHD.BluRay.TrueHD.Atmos.x265-GROUP.mkv", want: "heat:1995"},
		{name: "codec audio quality", path: "/movies/Alien.1979.1080p.BluRay.DTS-HD.MA.5.1.x264-Example.mkv", want: "alien:1979"},
		{name: "28 days later minihd", path: "/电影/合集5-1/惊变28天[繁英字幕].28.Days.Later.2002.BluRay.1080p.x265.10bit-MiniHD/28.Days.Later.2002.BluRay.1080p.x265.10bit-MiniHD.mkv", want: "28 days later:2002"},
		{name: "28 days later xiaomi", path: "/电影/合集5-1/惊变28天[中文字幕].28.Days.Later.2002.BluRay.1080p.DTS-HD.MA5.1.x265.10bit-Xiaomi/28.Days.Later.2002.BluRay.1080p.DTS-HD.MA5.1.x265.10bit-Xiaomi.mkv", want: "28 days later:2002"},
		{name: "ammonite hdma", path: "/电影/合集5-1/菊石[中英字幕].Ammonite.2020.BluRay.1080p.DTS-HDMA5.1.x265.10bit-Xiaomi/Ammonite.2020.BluRay.1080p.DTS-HDMA5.1.x265.10bit-Xiaomi.mkv", want: "ammonite:2020"},
		{name: "con air two audio", path: "/movies/ConAir.1997.BluRay.1080p.x265.2Audio-MiniHD.mkv", want: "conair:1997"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			key := normalizedMovieWorkKeyFromSignal(extractFilenameSignalModel(tt.path))
			if key.Normalized != tt.want {
				t.Fatalf("expected %q, got %q from title %q", tt.want, key.Normalized, key.Title)
			}
		})
	}
}

func TestNormalizedMovieWorkKeyKeepsMaterialDifferences(t *testing.T) {
	first := normalizedMovieWorkKeyFromSignal(extractFilenameSignalModel("/movies/Alien.1979.1080p.BluRay.x265.mkv"))
	second := normalizedMovieWorkKeyFromSignal(extractFilenameSignalModel("/movies/Aliens.1986.1080p.BluRay.x265.mkv"))
	if first.Normalized == "" || second.Normalized == "" {
		t.Fatalf("expected keys, got %q and %q", first.Normalized, second.Normalized)
	}
	if first.Normalized == second.Normalized {
		t.Fatalf("expected distinct keys, both were %q", first.Normalized)
	}
}

func TestCompileSiblingMovieVersionAssignmentsFromFiles(t *testing.T) {
	files := []database.InventoryFile{
		{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/合集5-2/空房间[简繁英字幕].3.Iron.2004.1080p.Bluray.x265.10bit.DDP.5.1-MiniHD/3.Iron.2004.1080p.Bluray.x265.10bit.DDP.5.1-MiniHD.mkv", ContentClass: SourceContentClassVideo, Status: inventory.FileStatusAvailable},
		{ID: 2, LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/合集5-2/空房间[简繁英字幕].3-Iron.2004.1080p.BluRay.DTS.x265-10bit-TAGHD/3-Iron.2004.1080p.BluRay.DTS.x265-10bit-TAGHD.mkv", ContentClass: SourceContentClassVideo, Status: inventory.FileStatusAvailable},
	}
	assignments := compileSiblingMovieVersionAssignmentsFromFiles(files, nil, newFilenameTokenProfileCache())
	if len(assignments) != 2 {
		t.Fatalf("expected two assignments, got %#v", assignments)
	}
	var target string
	for _, file := range files {
		assignment := assignments[file.StoragePath]
		if assignment.AssignmentType != pathTreeAssignmentVersion {
			t.Fatalf("expected version assignment for %s, got %#v", file.StoragePath, assignment)
		}
		if target == "" {
			target = assignment.TargetKey
		}
		if assignment.TargetKey != target {
			t.Fatalf("expected shared target %q, got %q", target, assignment.TargetKey)
		}
	}
	if target != "/movies/合集5-2/3 Iron (2004)" {
		t.Fatalf("unexpected target key %q", target)
	}
}

func TestCompileSiblingMovieVersionAssignmentsSkipsDifferentMovies(t *testing.T) {
	files := []database.InventoryFile{
		{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/collection/Alien.1979.1080p.BluRay.x265/Alien.1979.1080p.BluRay.x265.mkv", ContentClass: SourceContentClassVideo, Status: inventory.FileStatusAvailable},
		{ID: 2, LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/collection/Aliens.1986.1080p.BluRay.x265/Aliens.1986.1080p.BluRay.x265.mkv", ContentClass: SourceContentClassVideo, Status: inventory.FileStatusAvailable},
	}
	assignments := compileSiblingMovieVersionAssignmentsFromFiles(files, nil, newFilenameTokenProfileCache())
	if len(assignments) != 0 {
		t.Fatalf("expected no assignments, got %#v", assignments)
	}
}

func TestCompileMovieCollectionAssignmentsFromFiles(t *testing.T) {
	files := []database.InventoryFile{
		{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/collection/Alien.1979.mkv", ContentClass: SourceContentClassVideo, Status: inventory.FileStatusAvailable},
		{ID: 2, LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/collection/Aliens.1986/Aliens.1986.1080p.BluRay.x265.mkv", ContentClass: SourceContentClassVideo, Status: inventory.FileStatusAvailable},
		{ID: 3, LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/collection/Heat.1995.mkv", ContentClass: SourceContentClassVideo, Status: inventory.FileStatusAvailable},
	}
	assignments := compileMovieCollectionAssignmentsFromFiles(files, nil, newFilenameTokenProfileCache())
	if len(assignments) != 3 {
		t.Fatalf("expected three assignments, got %#v", assignments)
	}
	for _, file := range files {
		assignment := assignments[file.StoragePath]
		if assignment.AssignmentType != pathTreeAssignmentMovie || assignment.TargetKey == "" {
			t.Fatalf("expected movie collection assignment for %s, got %#v", file.StoragePath, assignment)
		}
	}
}

func TestCompilePathTreeMovieAssignmentsPreservesVersionsInsideCollections(t *testing.T) {
	files := []database.InventoryFile{
		{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/collection/Heat.1995.1080p/Heat.1995.1080p.BluRay.x265.mkv", ContentClass: SourceContentClassVideo, Status: inventory.FileStatusAvailable},
		{ID: 2, LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/collection/Heat.1995.2160p/Heat.1995.2160p.UHD.BluRay.x265.mkv", ContentClass: SourceContentClassVideo, Status: inventory.FileStatusAvailable},
		{ID: 3, LibraryID: 1, StorageProvider: "local", StoragePath: "/movies/collection/Alien.1979/Alien.1979.1080p.BluRay.x265.mkv", ContentClass: SourceContentClassVideo, Status: inventory.FileStatusAvailable},
	}
	assignments := compilePathTreeMovieAssignmentsFromFiles(files, nil, newFilenameTokenProfileCache())
	if len(assignments) != 3 {
		t.Fatalf("expected three assignments, got %#v", assignments)
	}
	if assignments[files[0].StoragePath].AssignmentType != pathTreeAssignmentVersion || assignments[files[1].StoragePath].AssignmentType != pathTreeAssignmentVersion {
		t.Fatalf("expected Heat versions inside collection, got %#v", assignments)
	}
	if assignments[files[2].StoragePath].AssignmentType != pathTreeAssignmentMovie {
		t.Fatalf("expected Alien to remain independent movie, got %#v", assignments[files[2].StoragePath])
	}
}
