package library

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
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

type SyncResult struct {
	DirectoriesScanned           int            `json:"directories_scanned"`
	FilesSeen                    int            `json:"files_seen"`
	MetadataItemsSeen            int            `json:"metadata_items_seen"`
	InventoryFilesSeen           int            `json:"inventory_files_seen"`
	ExcludedFilesSkipped         int            `json:"excluded_files_skipped"`
	ExcludedFilesSkippedByReason map[string]int `json:"excluded_files_skipped_by_reason,omitempty"`
}

type scanMode struct {
	partial                    bool
	rootPath                   string
	deferRecognitionResolution bool
	metadataMatchItemIDs       []uint
	recognitionResolveFileIDs  []uint
	inventoryProbeFileIDs      []uint
	classificationFileIDs      []uint
	discoveredFiles            map[string]database.InventoryFile
	directorySnapshots         map[string]scanDirectorySnapshot
	decisionSnapshots          map[string]scanDirectorySnapshot
	pathTreeAssignmentsByPath  map[string]pathTreeWorkGroupAssignment
	pendingSiblingMovieFiles   map[string][]pendingSiblingMovieFile
	resolvedDirectories        map[string]struct{}
	skippedDirectories         map[string]error
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

func (m *scanMode) recordRecognitionResolveCandidate(fileID uint) {
	if m == nil || fileID == 0 {
		return
	}
	m.recognitionResolveFileIDs = append(m.recognitionResolveFileIDs, fileID)
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

func (m *scanMode) markDirectoryResolved(path string) {
	if m == nil {
		return
	}
	key := strings.TrimSpace(path)
	if key == "" {
		return
	}
	if m.resolvedDirectories == nil {
		m.resolvedDirectories = make(map[string]struct{})
	}
	m.resolvedDirectories[key] = struct{}{}
}

func (m *scanMode) directoryResolved(path string) bool {
	if m == nil || m.resolvedDirectories == nil {
		return false
	}
	_, ok := m.resolvedDirectories[strings.TrimSpace(path)]
	return ok
}

func (m *scanMode) recordMetadataMatchCandidate(itemID uint) {
	if m == nil || itemID == 0 {
		return
	}
	m.metadataMatchItemIDs = append(m.metadataMatchItemIDs, itemID)
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
