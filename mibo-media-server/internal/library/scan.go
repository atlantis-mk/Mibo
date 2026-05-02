package library

import (
	"fmt"
	"regexp"
	"strings"
	"time"

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
	ItemType             string
	ItemPath             string
	SourcePath           string
	SeriesPath           string
	SeasonPath           string
	Title                string
	OriginalTitle        string
	SeriesTitle          string
	Year                 *int
	Tags                 []string
	SeasonNumber         *int
	EpisodeSlots         []catalogEpisodeSlot
	StorageProvider      string
	StableIdentityKey    string
	ProviderName         string
	HashesJSON           string
	ObjectType           string
	ProviderMeta         map[string]string
	SizeBytes            int64
	ModifiedAt           *time.Time
	Container            string
	PreferredAssetType   string
	PreferredAssetRole   string
	NormalizationVersion string
	RemovedTokens        []titleclean.RemovedToken
	FilenameSignals      filenameSignalModel
	SubtitleSidecars     []catalogScanSidecar
	MetadataSidecars     []catalogScanMetadataSidecar
	ImageCandidates      []catalogScanImageCandidate
	ExternalIDs          []catalogScanExternalID
	Decisions            []scanDecision
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
	partial               bool
	rootPath              string
	catalogMatchItemIDs   []uint
	inventoryProbeFileIDs []uint
	classificationFileIDs []uint
	directorySummaries    map[string]scanDirectorySummary
}

func (m *scanMode) directorySummary(libraryType string, libraryRoot string, snapshot scanDirectorySnapshot) scanDirectorySummary {
	if m == nil {
		return buildScanDirectorySummary(libraryType, libraryRoot, snapshot)
	}
	if m.directorySummaries == nil {
		m.directorySummaries = make(map[string]scanDirectorySummary)
	}
	key := strings.TrimSpace(snapshot.Path)
	if summary, ok := m.directorySummaries[key]; ok {
		return summary
	}
	summary := buildScanDirectorySummary(libraryType, libraryRoot, snapshot)
	m.directorySummaries[key] = summary
	return summary
}

func (m *scanMode) recordCatalogMatchCandidate(itemID uint) {
	if m == nil || itemID == 0 {
		return
	}
	m.catalogMatchItemIDs = append(m.catalogMatchItemIDs, itemID)
}

func (m *scanMode) recordInventoryProbeCandidate(fileID uint) {
	if m == nil || fileID == 0 {
		return
	}
	m.inventoryProbeFileIDs = append(m.inventoryProbeFileIDs, fileID)
}

func (m *scanMode) recordClassificationValidationCandidate(fileID uint) {
	if m == nil || fileID == 0 {
		return
	}
	m.classificationFileIDs = append(m.classificationFileIDs, fileID)
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
