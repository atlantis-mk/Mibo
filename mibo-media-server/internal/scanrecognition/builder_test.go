package scanrecognition

import "testing"

func TestBuildTreeKeepsDirectVideosSeparateFromChildVideos(t *testing.T) {
	tree, err := BuildTree(Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/root-movie.mkv", IsVideo: true},
			{ID: 2, Path: "/media/Show/Season 1/Show.S01E01.mkv", IsVideo: true},
			{ID: 3, Path: "/media/Show/Season 1/Show.S01E02.mkv", IsVideo: true},
		},
	})
	if err != nil {
		t.Fatalf("BuildTree returned error: %v", err)
	}

	root := tree.Node("/media")
	if root == nil {
		t.Fatalf("expected root node")
	}
	if len(root.DirectVideos) != 1 || root.DirectVideos[0].ID != 1 {
		t.Fatalf("expected only root video on root node, got %#v", root.DirectVideos)
	}

	season := tree.Node("/media/Show/Season 1")
	if season == nil {
		t.Fatalf("expected season node")
	}
	if len(season.DirectVideos) != 2 {
		t.Fatalf("expected two child videos on season node, got %#v", season.DirectVideos)
	}
	if len(root.Children) != 1 || root.Children[0].Path != "/media/Show" {
		t.Fatalf("expected Show as root child, got %#v", root.Children)
	}
}

func TestBuildTreeAttachesSidecarsToOwningDirectory(t *testing.T) {
	tree, err := BuildTree(Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/Movie/movie.nfo", IsNFO: true, SidecarText: "<movie><title>Movie</title></movie>"},
			{ID: 2, Path: "/media/Movie/Movie.2020.mkv", IsVideo: true},
		},
	})
	if err != nil {
		t.Fatalf("BuildTree returned error: %v", err)
	}

	movie := tree.Node("/media/Movie")
	if movie == nil {
		t.Fatalf("expected movie node")
	}
	if len(movie.Sidecars) != 1 || movie.Sidecars[0].ID != 1 {
		t.Fatalf("expected sidecar on movie directory, got %#v", movie.Sidecars)
	}
	if len(movie.DirectVideos) != 1 || movie.DirectVideos[0].ID != 2 {
		t.Fatalf("expected direct video on movie directory, got %#v", movie.DirectVideos)
	}
}

func TestBuildTreeNormalizesPathsAndIgnoresOutsideRoot(t *testing.T) {
	tree, err := BuildTree(Input{
		RootPath: "/media/Library/",
		Files: []FileInput{
			{ID: 1, Path: "/media/Library/Movies/Alien.mkv", IsVideo: true},
			{ID: 2, Path: "\\media\\Library\\Shows\\Show.S01E01.mkv", IsVideo: true},
			{ID: 3, Path: "/media/Library2/Movie.mkv", IsVideo: true},
			{ID: 4, Path: "/media/Library/../secret.mkv", IsVideo: true},
			{ID: 5, Path: "/media/Library/Movies/../Movies/Escape.mkv", IsVideo: true},
			{ID: 6, Path: "media/Library/Relative.mkv", IsVideo: true},
			{ID: 7, Path: "openlist://media/Library/Remote.mkv", IsVideo: true},
		},
	})
	if err != nil {
		t.Fatalf("BuildTree returned error: %v", err)
	}

	movies := tree.Node("/media/Library/Movies")
	if movies == nil {
		t.Fatalf("expected Movies node")
	}
	if len(movies.DirectVideos) != 1 || movies.DirectVideos[0].ID != 1 {
		t.Fatalf("expected only canonical in-root movie path, got %#v", movies.DirectVideos)
	}
	if tree.Node("/media/Library/Shows") == nil {
		t.Fatalf("expected mixed separator path to be normalized")
	}
	if tree.Node("/media/Library2") != nil {
		t.Fatalf("expected sibling prefix path to be ignored")
	}
	if tree.Node("/media/secret.mkv") != nil || tree.Node("/media") != nil {
		t.Fatalf("expected traversal path outside root to be ignored")
	}
	if tree.Node("/media/Library/Relative.mkv") != nil || tree.Node("/media/Library/Remote.mkv") != nil {
		t.Fatalf("expected relative and scheme-like paths to be ignored")
	}
}

func TestBuildTreeDoesNotAttachOneFileToMultipleBuckets(t *testing.T) {
	tree, err := BuildTree(Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 1, Path: "/media/Movie/Movie.mkv", IsVideo: true, IsNFO: true},
		},
	})
	if err != nil {
		t.Fatalf("BuildTree returned error: %v", err)
	}

	movie := tree.Node("/media/Movie")
	if movie == nil {
		t.Fatalf("expected movie node")
	}
	if len(movie.DirectVideos) != 1 || len(movie.Sidecars) != 0 {
		t.Fatalf("expected ambiguous file to attach only as video, got videos=%#v sidecars=%#v", movie.DirectVideos, movie.Sidecars)
	}
}

func TestBuildTreeReturnsStableChildOrderAndDeduplicatesFiles(t *testing.T) {
	tree, err := BuildTree(Input{
		RootPath: "/media",
		Files: []FileInput{
			{ID: 3, Path: "/media/B/b.mkv", IsVideo: true},
			{ID: 1, Path: "/media/A/a.mkv", IsVideo: true},
			{ID: 1, Path: "/media/A/a.mkv", IsVideo: true},
		},
	})
	if err != nil {
		t.Fatalf("BuildTree returned error: %v", err)
	}

	root := tree.Node("/media")
	if root == nil {
		t.Fatalf("expected root node")
	}
	if len(root.Children) != 2 || root.Children[0].Path != "/media/A" || root.Children[1].Path != "/media/B" {
		t.Fatalf("expected stable child order, got %#v", root.Children)
	}

	aNode := tree.Node("/media/A")
	if aNode == nil {
		t.Fatalf("expected A node")
	}
	if len(aNode.DirectVideos) != 1 {
		t.Fatalf("expected duplicate file to attach once, got %#v", aNode.DirectVideos)
	}
}

func TestBuildTreeRejectsEmptyRoot(t *testing.T) {
	if _, err := BuildTree(Input{}); err == nil {
		t.Fatalf("expected empty root path error")
	}
}
