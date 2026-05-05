package metadata

import "github.com/atlan/mibo-media-server/internal/settings"

const (
	OperationTypeMatch                              = "match"
	OperationTypeRefetch                            = "refetch"
	OperationTypeManualApply                        = "manual_apply"
	OperationTypeLocalApply                         = "local_apply"
	OperationTypeGovernanceEpisodeNumbering         = "governance_episode_numbering"
	OperationTypeGovernanceClassificationRule       = "governance_classification_rule"
	OperationTypeGovernanceClassificationCorrection = "governance_classification_correction"
	OperationTypeGovernanceAssetLink                = "governance_asset_link"
	OperationTypeGovernanceAssetUnlink              = "governance_asset_unlink"

	OperationStatusApplied     = "applied"
	OperationStatusNoCandidate = "no_candidate"
	OperationStatusNeedsReview = "needs_review"
	OperationStatusSkipped     = "skipped"
	OperationStatusFailed      = "failed"

	FieldApplyModeAutomated = "automated"
	FieldApplyModeManual    = "manual"
	FieldApplyModeLocal     = "local"
	FieldApplyModeSystem    = "system"

	ProviderAttemptOutcomeSuccess            = "success"
	ProviderAttemptOutcomeNoResult           = "no_result"
	ProviderAttemptOutcomeSkippedUnavailable = "skipped_unavailable"
	ProviderAttemptOutcomeSkippedUnsupported = "skipped_unsupported"
	ProviderAttemptOutcomeFailedRetryable    = "failed_retryable"
	ProviderAttemptOutcomeFailedTerminal     = "failed_terminal"
)

type MetadataOperationRequest struct {
	Operation                 string
	OriginItemID              uint
	TargetItemID              uint
	ManualCandidateExternalID string
	PreferredProviderInstance string
	Force                     bool
}

type MetadataOperationResult struct {
	Operation         string
	OriginItemID      uint
	TargetItemID      uint
	TargetType        string
	Status            string
	GovernanceStatus  string
	Plan              MetadataExecutionPlanSummary
	ProviderAttempts  []MetadataProviderAttempt
	MetadataSourceIDs []uint
	AppliedFields     []MetadataAppliedField
	SkippedFields     []MetadataSkippedField
	AffectedScope     MetadataAffectedScope
	Warnings          []MetadataOperationWarning
}

type MetadataOperationResponse struct {
	Operation         string                       `json:"operation"`
	OriginItemID      uint                         `json:"origin_item_id"`
	TargetItemID      uint                         `json:"target_item_id"`
	TargetType        string                       `json:"target_type"`
	Status            string                       `json:"status"`
	GovernanceStatus  string                       `json:"governance_status,omitempty"`
	Plan              MetadataExecutionPlanSummary `json:"plan"`
	ProviderAttempts  []MetadataProviderAttempt    `json:"provider_attempts,omitempty"`
	MetadataSourceIDs []uint                       `json:"metadata_source_ids,omitempty"`
	AppliedFields     []MetadataAppliedField       `json:"applied_fields,omitempty"`
	SkippedFields     []MetadataSkippedField       `json:"skipped_fields,omitempty"`
	AffectedScope     MetadataAffectedScope        `json:"affected_scope"`
	Warnings          []MetadataOperationWarning   `json:"warnings,omitempty"`
}

type MetadataExecutionPlan struct {
	LibraryID                 uint
	StrategyID                uint
	MetadataProfileID         *uint
	MetadataProfileName       string
	PreferredMetadataLanguage string
	PreferredImageLanguage    string
	SearchProviders           []settings.ResolvedMetadataProviderInstance
	DetailProviders           []settings.ResolvedMetadataProviderInstance
	ImageProviders            []settings.ResolvedMetadataProviderInstance
	PeopleProviders           []settings.ResolvedMetadataProviderInstance
	HierarchyProviders        []settings.ResolvedMetadataProviderInstance
	LocalEvidenceEnabled      bool
}

type MetadataExecutionPlanSummary struct {
	LibraryID                 uint
	StrategyID                uint
	MetadataProfileID         *uint
	MetadataProfileName       string
	PreferredMetadataLanguage string
	PreferredImageLanguage    string
	SearchProviders           []MetadataPlanProviderSummary
	DetailProviders           []MetadataPlanProviderSummary
	ImageProviders            []MetadataPlanProviderSummary
	PeopleProviders           []MetadataPlanProviderSummary
	HierarchyProviders        []MetadataPlanProviderSummary
	LocalEvidenceEnabled      bool
}

type MetadataPlanProviderSummary struct {
	ID                 uint
	Name               string
	ProviderType       string
	Enabled            bool
	Configured         bool
	Operational        bool
	AvailabilityStatus string
	CooldownUntil      *string
}

type MetadataProviderAttempt struct {
	Stage                string
	ProviderInstanceID   uint
	ProviderInstanceName string
	ProviderType         string
	Outcome              string
	ErrorClass           string
	ErrorMessage         string
	StatusCode           int
	CandidateCount       int
	Selected             bool
}

type NormalizedMetadataCandidate struct {
	Provider      string
	ProviderType  string
	ExternalID    string
	Title         string
	OriginalTitle string
	Overview      string
	ReleaseDate   string
	Year          *int
	PosterURL     string
	BackdropURL   string
	Confidence    float64
	MatchedQuery  string
	ReasonSummary string
}

type NormalizedMetadataDetail struct {
	Provider        string
	ProviderType    string
	ExternalID      string
	Title           string
	OriginalTitle   string
	Overview        string
	ReleaseDate     string
	FirstAirDate    string
	LastAirDate     string
	Year            *int
	RuntimeSeconds  *int
	CommunityRating *float64
	OfficialRating  string
	SeriesStatus    string
	Tags            []NormalizedMetadataTag
	Images          []NormalizedMetadataImage
	People          []NormalizedMetadataPerson
	ExternalIDs     []NormalizedMetadataExternalID
	Hierarchy       *NormalizedMetadataHierarchy
}

type NormalizedMetadataTag struct {
	Kind string
	Name string
}

type NormalizedMetadataImage struct {
	ImageType string
	URL       string
	Language  string
	Width     *int
	Height    *int
	SortOrder int
	Selected  bool
}

type NormalizedMetadataPerson struct {
	Name         string
	Role         string
	Character    string
	SortOrder    int
	AvatarURL    string
	TMDBPersonID *int
	IMDBID       string
}

type NormalizedMetadataExternalID struct {
	Provider     string
	ProviderType string
	ExternalID   string
	IsPrimary    bool
	Confidence   *float64
}

type NormalizedMetadataHierarchy struct {
	Seasons []NormalizedMetadataSeason
}

type NormalizedMetadataSeason struct {
	SeriesExternalID string
	ProviderType     string
	ExternalID       string
	SeasonNumber     int
	Title            string
	Overview         string
	AirDate          string
	PosterURL        string
	PosterPath       string
	People           []NormalizedMetadataPerson
	ExternalIDs      []NormalizedMetadataExternalID
	Episodes         []NormalizedMetadataEpisode
}

type NormalizedMetadataEpisode struct {
	SeriesExternalID string
	ProviderType     string
	ExternalID       string
	SeasonNumber     int
	EpisodeNumber    int
	Title            string
	Overview         string
	AirDate          string
	RuntimeSeconds   *int
	CommunityRating  *float64
	StillURL         string
	StillPath        string
	People           []NormalizedMetadataPerson
}

type MetadataAppliedField struct {
	ItemID     uint
	FieldKey   string
	SourceID   *uint
	ApplyMode  string
	Confidence *float64
}

type MetadataSkippedField struct {
	ItemID   uint
	FieldKey string
	Reason   string
}

type MetadataOperationWarning struct {
	Code    string
	Message string
}

type MetadataAffectedScope struct {
	ItemIDs   []uint
	LibraryID uint
	RootID    *uint
}
