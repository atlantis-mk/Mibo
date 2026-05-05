package health

import "time"

const (
	OverallStatusHealthy  = "healthy"
	OverallStatusWarning  = "warning"
	OverallStatusError    = "error"
	OverallStatusBlocking = "blocking"

	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityError    = "error"
	SeverityBlocking = "blocking"

	ReasonStorageAuthExpired   = "storage_auth_expired"
	ReasonJobFailedUnknown     = "job_failed_unknown"
	ReasonIngestStageFailed    = "ingest_stage_failed"
	ReasonIngestStageStale     = "ingest_stage_stale"
	ReasonIngestReviewRequired = "ingest_review_required"

	ScopeGlobal      = "global"
	ScopeMediaSource = "media_source"
	ScopeLibrary     = "library"
	ScopeJob         = "job"
	ScopeIngest      = "ingest"
	ScopeDependency  = "dependency"

	ActionOpenExternalAdmin     = "open_external_admin"
	ActionValidateMediaSource   = "validate_media_source"
	ActionRescanAffectedLibrary = "rescan_affected_libraries"
	ActionViewJob               = "view_job"
	ActionIgnoreIssue           = "ignore_issue"
)

type Summary struct {
	Status        string  `json:"status"`
	IssueCount    int     `json:"issue_count"`
	BlockingCount int     `json:"blocking_count"`
	ErrorCount    int     `json:"error_count"`
	WarningCount  int     `json:"warning_count"`
	Issues        []Issue `json:"issues"`
}

type Issue struct {
	ID              string          `json:"id"`
	Severity        string          `json:"severity"`
	ReasonCode      string          `json:"reason_code"`
	Scope           string          `json:"scope"`
	Title           string          `json:"title"`
	Message         string          `json:"message"`
	Impact          Impact          `json:"impact"`
	Affected        Affected        `json:"affected"`
	Actions         []Action        `json:"actions"`
	TechnicalDetail TechnicalDetail `json:"technical_detail"`
	FirstSeenAt     *time.Time      `json:"first_seen_at,omitempty"`
	LastSeenAt      *time.Time      `json:"last_seen_at,omitempty"`
	LatestJobID     uint            `json:"latest_job_id,omitempty"`
}

type Impact struct {
	BlocksScan           bool  `json:"blocks_scan"`
	BlocksHomeVisibility bool  `json:"blocks_home_visibility"`
	BlocksPlayback       bool  `json:"blocks_playback"`
	BlocksMetadata       bool  `json:"blocks_metadata"`
	AffectedCatalogItems int64 `json:"affected_catalog_items"`
	AffectedFiles        int64 `json:"affected_files"`
}

type Affected struct {
	MediaSources []MediaSourceRef `json:"media_sources"`
	Libraries    []LibraryRef     `json:"libraries"`
	Jobs         []JobRef         `json:"jobs"`
}

type MediaSourceRef struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	RootPath    string `json:"root_path"`
	AdminURL    string `json:"admin_url,omitempty"`
	OpenListURL string `json:"openlist_url,omitempty"`
}

type LibraryRef struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	MediaSourceID uint   `json:"media_source_id"`
	RootPath      string `json:"root_path"`
}

type JobRef struct {
	ID          uint       `json:"id"`
	Kind        string     `json:"kind"`
	Status      string     `json:"status"`
	Attempts    int        `json:"attempts"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	PayloadJSON string     `json:"payload_json,omitempty"`
}

type Action struct {
	Type          string `json:"type"`
	Label         string `json:"label"`
	Description   string `json:"description,omitempty"`
	Href          string `json:"href,omitempty"`
	MediaSourceID uint   `json:"media_source_id,omitempty"`
	JobID         uint   `json:"job_id,omitempty"`
	LibraryIDs    []uint `json:"library_ids,omitempty"`
}

type TechnicalDetail struct {
	JobKind      string `json:"job_kind,omitempty"`
	JobStatus    string `json:"job_status,omitempty"`
	PayloadJSON  string `json:"payload_json,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type ValidationResult struct {
	MediaSourceID uint   `json:"media_source_id"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

type RescanResult struct {
	IssueID string   `json:"issue_id"`
	Jobs    []JobRef `json:"jobs"`
}

type IgnoreResult struct {
	IssueID string `json:"issue_id"`
	Status  string `json:"status"`
}
