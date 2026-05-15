package scanrecognition

type DirectoryKind string

const (
	DirectoryKindRoot            DirectoryKind = "root"
	DirectoryKindMovie           DirectoryKind = "movie"
	DirectoryKindMovieVersions   DirectoryKind = "movie_versions"
	DirectoryKindMovieCollection DirectoryKind = "movie_collection"
	DirectoryKindSeries          DirectoryKind = "series"
	DirectoryKindSeason          DirectoryKind = "season"
	DirectoryKindEpisodeGroup    DirectoryKind = "episode_group"
	DirectoryKindExtras          DirectoryKind = "extras"
	DirectoryKindUnknown         DirectoryKind = "unknown"
	DirectoryKindAmbiguous       DirectoryKind = "ambiguous"
)

type Input struct {
	RootPath string
	Files    []FileInput
}

type FileInput struct {
	ID                uint
	Path              string
	StorageProvider   string
	StableIdentityKey string
	SizeBytes         int64
	IsVideo           bool
	IsNFO             bool
	SidecarText       string
}

type Tree struct {
	Root  *DirectoryNode
	index map[string]*DirectoryNode
}

type DirectoryNode struct {
	Path         string
	Name         string
	Kind         DirectoryKind
	Children     []*DirectoryNode
	DirectVideos []FileInput
	Sidecars     []FileInput
	parent       *DirectoryNode
}

func (t *Tree) Node(pathValue string) *DirectoryNode {
	if t == nil {
		return nil
	}
	return t.index[normalizePath(pathValue)]
}
