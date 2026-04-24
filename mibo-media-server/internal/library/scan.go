package library

import (
	"fmt"
	"regexp"
	"time"
)

var (
	episodePattern = regexp.MustCompile(`(?i)^(.*?)[\s._-]+(?:s(\d{1,2})e(\d{1,2})|(\d{1,2})x(\d{1,2}))(?:[\s._-]+.*)?$`)
	yearPattern    = regexp.MustCompile(`(?i)(?:^|[\s._\-(])((?:19|20)\d{2})(?:$|[\s._\-)])`)
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
	Type          string
	Title         string
	OriginalTitle string
	SeriesTitle   string
	Year          *int
	SeasonNumber  *int
	EpisodeNumber *int
	SourcePath    string
	Status        string
}

type SyncResult struct {
	DirectoriesScanned int `json:"directories_scanned"`
	FilesSeen          int `json:"files_seen"`
	MediaItemsUpserted int `json:"media_items_upserted"`
	MediaFilesUpserted int `json:"media_files_upserted"`
}

type scanMode struct {
	partial  bool
	rootPath string
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
