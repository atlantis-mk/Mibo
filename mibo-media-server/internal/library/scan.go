package library

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
	"github.com/atlan/mibo-media-server/internal/titleclean"
)

var (
	episodePattern             = regexp.MustCompile(`(?i)^(.*?)[\s._-]+(?:s(\d{1,2})e(\d{1,2})|(\d{1,2})x(\d{1,2}))(?:[\s._-]+.*)?$`)
	yearPattern                = regexp.MustCompile(`(?i)(?:^|[\s._\-(])((?:19|20)\d{2})(?:$|[\s._\-)])`)
	seasonDirectoryPattern     = regexp.MustCompile(`(?i)^(?:season|s)[\s._-]*0*(\d{1,2})(?:$|[\s._-]+.*$)|^第?\s*([0-9一二三四五六七八九十两零]+)\s*季(?:$|[\s._-]+.*$)`)
	embeddedSeasonDirPattern   = regexp.MustCompile(`(?i)^(.*?)[\s._-]+(?:season[\s._-]*0*(\d{1,2})|s0*(\d{1,2}))(?:$|[\s._\-(]+.*$)`)
	episodeOnlyPattern         = regexp.MustCompile(`(?i)^(?:e|ep|episode)[\s._-]*0*(\d{1,3})(?:[\s._-]+.*)?$`)
	embeddedEpisodePattern     = regexp.MustCompile(`(?i)(?:^|[\s._-])(?:e|ep|episode)[\s._-]*0*([1-9]\d{0,2})(?:$|[\s._-])`)
	numericEpisodePattern      = regexp.MustCompile(`^0*([1-9]\d{0,2})(?:[\s._-]+.*)?$`)
	chineseEpisodePattern      = regexp.MustCompile(`^第?\s*(\d{1,3})\s*[集话話](?:[\s._-]+.*)?$`)
	trailingEpisodePattern     = regexp.MustCompile(`(?i)^.*?[\s._-]+(?:e|ep|episode)?[\s._-]*0*([1-9]\d{0,2})(?:v\d+)?(?:[\s._-]+.*)?$`)
	trailingSeasonTitlePattern = regexp.MustCompile(`(?i)(?:[\s._-]+(?:season[\s._-]*0*\d{1,2}|s0*\d{1,2})|第\s*[0-9一二三四五六七八九十两零]+\s*季)$`)
	scanNoisePattern           = regexp.MustCompile(`(?i)(?:^|[\s._\-\[(【(])(2160p|1080p|720p|480p|4k|hdr10\+?|dv|dolby[\s._-]?vision|atmos|dts(?:hd)?|truehd|aac\d?(?:\.\d)?|x26[45]|h\.?26[45]|hevc|avc|bluray|blu[\s._-]?ray|bdrip|brrip|bdrmux|remux|web[\s._-]?dl|webrip|hdtv|uhd|nf|amzn|dsnp|hmax|proper|repack|extended|unrated|limited|dual[\s._-]?audio|multi(?:sub|subs)?|sub(?:bed|s)?|dub(?:bed)?|chs|cht|eng|jpn|gb|big5)(?:$|[\s._\-)\]】])`)
	genericMediaNamePattern    = regexp.MustCompile(`(?i)^(movie|video|feature|main|full|default|film|media|sample)$`)
)

var videoExtensions = map[string]struct{}{
	".mp4":  {},
	".mkv":  {},
	".avi":  {},
	".mov":  {},
	".wmv":  {},
	".m4v":  {},
	".ts":   {},
	".m2ts": {},
	".webm": {},
}

const (
	mediaFileIdentitySourceNone             = "none"
	mediaFileIdentitySourceStableIdentity   = "stable_identity"
	mediaFileIdentitySourceProviderEvidence = "provider_evidence"

	mediaFileIdentityStatusExact       = "exact"
	mediaFileIdentityStatusProvisional = "provisional"
	mediaFileIdentityStatusReconciled  = "fallback_reconciled"

	mediaFileReviewStatusNone    = "none"
	mediaFileReviewStatusPending = "pending"
	mediaFileReviewStatusNeeded  = "review_needed"

	fallbackDurationToleranceSeconds = 2.0
)

type classifiedMedia struct {
	Type                 string
	Title                string
	OriginalTitle        string
	SeriesTitle          string
	Year                 *int
	Tags                 []string
	SeasonNumber         *int
	EpisodeNumber        *int
	EpisodeNumbers       []int
	SourcePath           string
	Status               string
	NormalizationVersion string
	RemovedTokens        []titleclean.RemovedToken
	FilenameSignals      filenameSignalModel
}

type catalogEpisodeSlot struct {
	EpisodeNumber int
	ItemPath      string
}

type catalogScanArtifact struct {
	ItemType               string
	ItemPath               string
	SourcePath             string
	SeriesPath             string
	SeasonPath             string
	Title                  string
	OriginalTitle          string
	SeriesTitle            string
	Year                   *int
	Tags                   []string
	SeasonNumber           *int
	EpisodeSlots           []catalogEpisodeSlot
	StorageProvider        string
	StableIdentityKey      string
	ProviderName           string
	HashesJSON             string
	ThumbnailURL           string
	ObjectType             string
	ProviderMeta           map[string]string
	SizeBytes              int64
	ModifiedAt             *time.Time
	Container              string
	PreferredAssetType     string
	PreferredAssetRole     string
	NormalizationVersion   string
	RemovedTokens          []titleclean.RemovedToken
	FilenameSignals        filenameSignalModel
	SubtitleSidecars       []catalogScanSidecar
	MetadataSidecars       []catalogScanMetadataSidecar
	ImageCandidates        []catalogScanImageCandidate
	ExternalIDs            []catalogScanExternalID
	Decisions              []scanDecision
	ContentShapeProfile    map[string]any
	ContentShapePlan       map[string]any
	ContentShapeAssignment map[string]any
}

type catalogScanImageCandidate struct {
	ImageType   string
	URL         string
	Path        string
	Source      string
	Priority    int
	Provisional bool
}

type catalogScanExternalID struct {
	Provider     string
	ProviderType string
	ExternalID   string
	Confidence   *float64
}

type catalogScanSidecar struct {
	Path              string
	Extension         string
	AssociationSource string
	SizeBytes         int64
	ModifiedAt        *time.Time
	StableIdentityKey string
}

type catalogScanMetadataSidecar struct {
	catalogScanSidecar
	ParseStatus string
	Hints       catalogScanMetadataHints
	ExternalIDs map[string]string
}

type catalogScanMetadataHints struct {
	Title         string
	OriginalTitle string
	Year          *int
	MediaType     string
	SeriesTitle   string
	SeasonNumber  *int
	EpisodeNumber *int
}

type SyncResult struct {
	DirectoriesScanned           int            `json:"directories_scanned"`
	FilesSeen                    int            `json:"files_seen"`
	CatalogItemsSeen             int            `json:"catalog_items_seen"`
	InventoryFilesSeen           int            `json:"inventory_files_seen"`
	ExcludedFilesSkipped         int            `json:"excluded_files_skipped"`
	ExcludedFilesSkippedByReason map[string]int `json:"excluded_files_skipped_by_reason,omitempty"`
}

type scanMode struct {
	partial                     bool
	rootPath                    string
	deferCatalogMaterialization bool
	catalogMatchItemIDs         []uint
	catalogMaterializeFileIDs   []uint
	inventoryProbeFileIDs       []uint
	classificationFileIDs       []uint
	discoveredFiles             map[string]database.InventoryFile
	directorySnapshots          map[string]scanDirectorySnapshot
	decisionSnapshots           map[string]scanDirectorySnapshot
	pathTreeAssignmentsByPath   map[string]pathTreeWorkGroupAssignment
	pendingSiblingMovieFiles    map[string][]pendingSiblingMovieFile
	materializedDirectories     map[string]struct{}
	skippedDirectories          map[string]error
}

type pendingSiblingMovieFile struct {
	Provider           storage.Provider
	Library            database.Library
	Object             storage.Object
	Snapshot           scanDirectorySnapshot
	DecisionSnapshot   scanDirectorySnapshot
	DirectorySnapshots map[string]scanDirectorySnapshot
	SubtitlePolicy     database.LibrarySubtitlePolicy
	FileID             uint
}

func (m *scanMode) pathTreeAssignment(storagePath string) pathTreeWorkGroupAssignment {
	if m == nil || len(m.pathTreeAssignmentsByPath) == 0 {
		return pathTreeWorkGroupAssignment{}
	}
	return m.pathTreeAssignmentsByPath[strings.TrimSpace(storagePath)]
}

func (m *scanMode) mergePathTreeAssignments(assignments map[string]pathTreeWorkGroupAssignment) {
	if m == nil || len(assignments) == 0 {
		return
	}
	if m.pathTreeAssignmentsByPath == nil {
		m.pathTreeAssignmentsByPath = make(map[string]pathTreeWorkGroupAssignment, len(assignments))
	}
	for storagePath, assignment := range assignments {
		m.pathTreeAssignmentsByPath[strings.TrimSpace(storagePath)] = assignment
	}
}

func (m *scanMode) allowsMissingCleanup() bool {
	if m == nil {
		return true
	}
	return !m.partial
}

func (m *scanMode) recordCatalogMaterializeCandidate(fileID uint) {
	if m == nil || fileID == 0 {
		return
	}
	m.catalogMaterializeFileIDs = append(m.catalogMaterializeFileIDs, fileID)
}

func (m *scanMode) recordDirectorySnapshot(snapshot scanDirectorySnapshot) {
	if m == nil {
		return
	}
	key := strings.TrimSpace(snapshot.Path)
	if key == "" {
		return
	}
	if m.directorySnapshots == nil {
		m.directorySnapshots = make(map[string]scanDirectorySnapshot)
	}
	if _, ok := m.directorySnapshots[key]; ok {
		return
	}
	m.directorySnapshots[key] = snapshot
}

func (m *scanMode) decisionSnapshot(path string) (scanDirectorySnapshot, bool) {
	if m == nil || m.decisionSnapshots == nil {
		return scanDirectorySnapshot{}, false
	}
	snapshot, ok := m.decisionSnapshots[strings.TrimSpace(path)]
	return snapshot, ok
}

func (m *scanMode) recordDecisionSnapshot(snapshot scanDirectorySnapshot) {
	if m == nil {
		return
	}
	key := strings.TrimSpace(snapshot.Path)
	if key == "" {
		return
	}
	if m.decisionSnapshots == nil {
		m.decisionSnapshots = make(map[string]scanDirectorySnapshot)
	}
	if _, ok := m.decisionSnapshots[key]; ok {
		return
	}
	m.decisionSnapshots[key] = snapshot
}

func (m *scanMode) markDirectoryMaterialized(path string) {
	if m == nil {
		return
	}
	key := strings.TrimSpace(path)
	if key == "" {
		return
	}
	if m.materializedDirectories == nil {
		m.materializedDirectories = make(map[string]struct{})
	}
	m.materializedDirectories[key] = struct{}{}
}

func (m *scanMode) directoryMaterialized(path string) bool {
	if m == nil || m.materializedDirectories == nil {
		return false
	}
	_, ok := m.materializedDirectories[strings.TrimSpace(path)]
	return ok
}

func (m *scanMode) recordCatalogMatchCandidate(itemID uint) {
	if m == nil || itemID == 0 {
		return
	}
	m.catalogMatchItemIDs = append(m.catalogMatchItemIDs, itemID)
}

func (m *scanMode) recordCatalogMatchCandidateForItem(item database.CatalogItem) {
	if m == nil || item.ID == 0 {
		return
	}
	if (item.Type == "season" || item.Type == "episode") && item.RootID != nil && *item.RootID != 0 {
		m.recordCatalogMatchCandidate(*item.RootID)
		return
	}
	m.recordCatalogMatchCandidate(item.ID)
}

func (m *scanMode) recordInventoryProbeCandidate(fileID uint) {
	if m == nil || fileID == 0 {
		return
	}
	m.inventoryProbeFileIDs = append(m.inventoryProbeFileIDs, fileID)
}

func (m *scanMode) recordDiscoveredFiles(files map[string]database.InventoryFile) {
	if m == nil || len(files) == 0 {
		return
	}
	if m.discoveredFiles == nil {
		m.discoveredFiles = make(map[string]database.InventoryFile, len(files))
	}
	for _, file := range files {
		if file.ID == 0 || strings.TrimSpace(file.StoragePath) == "" {
			continue
		}
		m.discoveredFiles[strings.TrimSpace(file.StoragePath)] = file
	}
}

func (m *scanMode) discoveredVideoFilesInSnapshot(snapshot scanDirectorySnapshot) []database.InventoryFile {
	if m == nil || len(m.discoveredFiles) == 0 {
		return nil
	}
	files := make([]database.InventoryFile, 0, len(snapshot.Objects))
	for _, object := range snapshot.Objects {
		if object.IsDir || !isVideoFile(object.Path) {
			continue
		}
		if file, ok := m.discoveredFiles[strings.TrimSpace(object.Path)]; ok {
			files = append(files, file)
		}
	}
	return files
}

func (m *scanMode) recordClassificationValidationCandidate(fileID uint) {
	if m == nil || fileID == 0 {
		return
	}
	m.classificationFileIDs = append(m.classificationFileIDs, fileID)
}

func (m *scanMode) recordSkippedDirectory(path string, err error) {
	if m == nil || err == nil {
		return
	}
	key := strings.TrimSpace(path)
	if key == "" {
		return
	}
	if m.skippedDirectories == nil {
		m.skippedDirectories = make(map[string]error)
	}
	m.skippedDirectories[key] = err
}

func (m *scanMode) skippedDirectoryPaths() []string {
	if m == nil || len(m.skippedDirectories) == 0 {
		return nil
	}
	paths := make([]string, 0, len(m.skippedDirectories))
	for path := range m.skippedDirectories {
		paths = append(paths, path)
	}
	return paths
}

type scanDirectorySnapshot struct {
	Path     string
	Objects  []storage.Object
	Sidecars sidecarIndex
}

func durationDelta(left, right float64) float64 {
	if left >= right {
		return left - right
	}
	return right - left
}

func cleanupDeletedAt() time.Time {
	return time.Now().UTC()
}

func scopedRefreshRootError(root string) error {
	return fmt.Errorf("invalid scoped refresh root: %s", root)
}
