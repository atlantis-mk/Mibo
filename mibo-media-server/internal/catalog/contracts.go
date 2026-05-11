package catalog

import (
	"encoding/json"
	"strings"
	"time"
)

// Catalog contracts keep the canonical TV root type as series.

type CatalogSelectedImage struct {
	ImageType string `json:"image_type"`
	URL       string `json:"url"`
	Language  string `json:"language,omitempty"`
	Width     *int   `json:"width,omitempty"`
	Height    *int   `json:"height,omitempty"`
}

type CatalogExternalIdentity struct {
	Provider     string   `json:"provider"`
	ProviderType string   `json:"provider_type"`
	ExternalID   string   `json:"external_id"`
	IsPrimary    bool     `json:"is_primary"`
	Source       string   `json:"source,omitempty"`
	Confidence   *float64 `json:"confidence,omitempty"`
}

type CatalogSourceEvidence struct {
	SourceType           string     `json:"source_type"`
	SourceName           string     `json:"source_name"`
	Language             string     `json:"language,omitempty"`
	ExternalID           string     `json:"external_id,omitempty"`
	MetadataProfileID    *uint      `json:"metadata_profile_id,omitempty"`
	MetadataProfileName  string     `json:"metadata_profile_name,omitempty"`
	ProviderInstanceID   *uint      `json:"provider_instance_id,omitempty"`
	ProviderInstanceName string     `json:"provider_instance_name,omitempty"`
	FallbackSummary      any        `json:"fallback_summary,omitempty"`
	Confidence           *float64   `json:"confidence,omitempty"`
	FetchedAt            time.Time  `json:"fetched_at"`
	ExpiresAt            *time.Time `json:"expires_at,omitempty"`
	Summary              any        `json:"summary,omitempty"`
}

type CatalogFieldState struct {
	FieldKey       string     `json:"field_key"`
	SourceID       *uint      `json:"source_id,omitempty"`
	Value          any        `json:"value,omitempty"`
	IsLocked       bool       `json:"is_locked"`
	LockReason     string     `json:"lock_reason,omitempty"`
	EditedByUserID *uint      `json:"edited_by_user_id,omitempty"`
	EditedAt       *time.Time `json:"edited_at,omitempty"`
}

type CatalogClassificationEvidence struct {
	Kind   string   `json:"kind"`
	Source string   `json:"source,omitempty"`
	Value  string   `json:"value,omitempty"`
	Weight *float64 `json:"weight,omitempty"`
}

type CatalogClassificationAlternative struct {
	Type       string   `json:"type"`
	Role       string   `json:"role,omitempty"`
	TargetKind string   `json:"target_kind,omitempty"`
	TargetKey  string   `json:"target_key,omitempty"`
	Confidence *float64 `json:"confidence,omitempty"`
	Reason     string   `json:"reason,omitempty"`
}

type CatalogClassificationDecision struct {
	ID             uint                               `json:"id"`
	SourcePath     string                             `json:"source_path"`
	DecisionType   string                             `json:"decision_type"`
	Role           string                             `json:"role,omitempty"`
	CandidateType  string                             `json:"candidate_type,omitempty"`
	TargetKind     string                             `json:"target_kind,omitempty"`
	TargetKey      string                             `json:"target_key,omitempty"`
	Status         string                             `json:"status"`
	Confidence     *float64                           `json:"confidence,omitempty"`
	Alternatives   []CatalogClassificationAlternative `json:"alternatives"`
	Evidence       []CatalogClassificationEvidence    `json:"evidence"`
	AffectedFiles  []string                           `json:"affected_files"`
	CorrectionActs []CatalogClassificationCorrection  `json:"correction_actions"`
	Reason         string                             `json:"reason,omitempty"`
	Warnings       []string                           `json:"warnings"`
	CreatedAt      time.Time                          `json:"created_at"`
	UpdatedAt      time.Time                          `json:"updated_at"`
	ResolvedAt     *time.Time                         `json:"resolved_at,omitempty"`
}

type CatalogClassificationCorrection struct {
	Action      string `json:"action"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type CatalogClassificationRuleSummary struct {
	ID              uint   `json:"id"`
	LibraryID       uint   `json:"library_id"`
	Key             string `json:"key"`
	Name            string `json:"name"`
	PathPattern     string `json:"path_pattern"`
	RuleType        string `json:"rule_type"`
	Role            string `json:"role,omitempty"`
	CandidateType   string `json:"candidate_type,omitempty"`
	SeriesTitle     string `json:"series_title,omitempty"`
	SeasonNumber    *int   `json:"season_number,omitempty"`
	NumberingSource string `json:"numbering_source,omitempty"`
	Enabled         bool   `json:"enabled"`
}

type CatalogChildSummary struct {
	ChildCount      int        `json:"child_count"`
	AvailableCount  int        `json:"available_count"`
	MissingCount    int        `json:"missing_count"`
	UnairedCount    int        `json:"unaired_count"`
	PlayedCount     int        `json:"played_count"`
	InProgressCount int        `json:"in_progress_count"`
	LatestAirDate   *time.Time `json:"latest_air_date,omitempty"`
	LatestAddedAt   *time.Time `json:"latest_added_at,omitempty"`
}

type CatalogResourceLink struct {
	MetadataItemID uint   `json:"metadata_item_id"`
	Role         string   `json:"role"`
	SegmentIndex int      `json:"segment_index"`
	StartSeconds *float64 `json:"start_seconds,omitempty"`
	EndSeconds   *float64 `json:"end_seconds,omitempty"`
	Confidence   *float64 `json:"confidence,omitempty"`
	Source       string   `json:"source,omitempty"`
}

type CatalogPersonDetail struct {
	ID        uint   `json:"id,omitempty"`
	Name      string `json:"name"`
	Role      string `json:"role,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type CatalogPersonPageDetail struct {
	ID                 uint                      `json:"id"`
	Name               string                    `json:"name"`
	SortName           string                    `json:"sort_name,omitempty"`
	AvatarURL          string                    `json:"avatar_url,omitempty"`
	Biography          string                    `json:"biography,omitempty"`
	Birthday           *time.Time                `json:"birthday,omitempty"`
	Deathday           *time.Time                `json:"deathday,omitempty"`
	PlaceOfBirth       string                    `json:"place_of_birth,omitempty"`
	KnownForDepartment string                    `json:"known_for_department,omitempty"`
	ExternalIdentities []CatalogExternalIdentity `json:"external_identities,omitempty"`
	RelatedItems       []CatalogListItem         `json:"related_items"`
}

type CatalogTagDetail struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type CatalogEpisodeParentContext struct {
	Series              *CatalogEpisodeSeriesContext `json:"series,omitempty"`
	Season              *CatalogEpisodeSeasonContext `json:"season,omitempty"`
	SeasonNumber        *int                         `json:"season_number,omitempty"`
	EpisodeNumber       *int                         `json:"episode_number,omitempty"`
	EpisodeNumberEnd    *int                         `json:"episode_number_end,omitempty"`
	IncompleteHierarchy bool                         `json:"incomplete_hierarchy"`
}

type CatalogEpisodeSeriesContext struct {
	ID             uint                   `json:"id"`
	Title          string                 `json:"title"`
	SelectedImages []CatalogSelectedImage `json:"selected_images,omitempty"`
}

type CatalogEpisodeSeasonContext struct {
	ID             uint                   `json:"id"`
	Title          string                 `json:"title"`
	Number         *int                   `json:"number,omitempty"`
	SelectedImages []CatalogSelectedImage `json:"selected_images,omitempty"`
}

type CatalogEpisodeShelfItem struct {
	ID                 uint                      `json:"id"`
	LibraryID          uint                      `json:"library_id"`
	Type               string                    `json:"type"`
	Title              string                    `json:"title"`
	Label              string                    `json:"label,omitempty"`
	Overview           string                    `json:"overview,omitempty"`
	SeasonNumber       *int                      `json:"season_number,omitempty"`
	EpisodeNumber      *int                      `json:"episode_number,omitempty"`
	EpisodeNumberEnd   *int                      `json:"episode_number_end,omitempty"`
	RuntimeSeconds     *int                      `json:"runtime_seconds,omitempty"`
	InventoryFileID    uint                      `json:"inventory_file_id,omitempty"`
	AvailabilityStatus string                    `json:"availability_status"`
	GovernanceStatus   string                    `json:"governance_status"`
	ReleaseDate        *time.Time                `json:"release_date,omitempty"`
	FirstAirDate       *time.Time                `json:"first_air_date,omitempty"`
	SelectedImages     []CatalogSelectedImage    `json:"selected_images,omitempty"`
	ExternalIdentities []CatalogExternalIdentity `json:"external_identities,omitempty"`
	Current            bool                      `json:"current"`
	Progress           *CatalogUserProgressState `json:"progress,omitempty"`
}

type CatalogResourceFileSummary struct {
	FileID              uint                        `json:"file_id"`
	Role                string                      `json:"role"`
	PartIndex           int                         `json:"part_index"`
	StorageProvider     string                      `json:"storage_provider"`
	StoragePath         string                      `json:"storage_path,omitempty"`
	StableIdentity      string                      `json:"stable_identity_key,omitempty"`
	SizeBytes           int64                       `json:"size_bytes"`
	Container           string                      `json:"container,omitempty"`
	Status              string                      `json:"status"`
	ModifiedAt          *time.Time                  `json:"modified_at,omitempty"`
	ProviderDiagnostics *CatalogProviderDiagnostics `json:"provider_diagnostics,omitempty"`
}

type CatalogProviderDiagnostics struct {
	StorageProvider    string   `json:"storage_provider,omitempty"`
	AvailableHashKeys  []string `json:"available_hash_keys,omitempty"`
	MetadataIndicators []string `json:"metadata_indicators,omitempty"`
}

type CatalogMediaStreamSummary struct {
	FileID          uint     `json:"file_id"`
	StreamIndex     int      `json:"stream_index"`
	StreamType      string   `json:"stream_type"`
	Codec           string   `json:"codec,omitempty"`
	Profile         string   `json:"profile,omitempty"`
	Level           *int     `json:"level,omitempty"`
	Language        string   `json:"language,omitempty"`
	Title           string   `json:"title,omitempty"`
	Width           *int     `json:"width,omitempty"`
	Height          *int     `json:"height,omitempty"`
	AvgFrameRate    string   `json:"avg_frame_rate,omitempty"`
	RFrameRate      string   `json:"r_frame_rate,omitempty"`
	FieldOrder      string   `json:"field_order,omitempty"`
	ColorSpace      string   `json:"color_space,omitempty"`
	BitDepth        *int     `json:"bit_depth,omitempty"`
	PixelFormat     string   `json:"pixel_format,omitempty"`
	ReferenceFrames *int     `json:"reference_frames,omitempty"`
	Channels        *int     `json:"channels,omitempty"`
	ChannelLayout   string   `json:"channel_layout,omitempty"`
	SampleRate      *int     `json:"sample_rate,omitempty"`
	BitRate         *int64   `json:"bit_rate,omitempty"`
	DurationSeconds *float64 `json:"duration_seconds,omitempty"`
	Default         bool     `json:"default,omitempty"`
	Forced          bool     `json:"forced,omitempty"`
	HearingImpaired bool     `json:"hearing_impaired,omitempty"`
	External        bool     `json:"external,omitempty"`
	URL             string   `json:"url,omitempty"`
	Available       *bool    `json:"available,omitempty"`
}

type CatalogListItem struct {
	ID                 uint                      `json:"id"`
	MetadataItemID     uint                      `json:"metadata_item_id,omitempty"`
	LibraryID          uint                      `json:"library_id"`
	ResourceCount      int                       `json:"resource_count,omitempty"`
	AvailableCount     int                       `json:"available_count,omitempty"`
	MissingCount       int                       `json:"missing_count,omitempty"`
	SourceKind         string                    `json:"source_kind,omitempty"`
	InventoryFileID    *uint                     `json:"inventory_file_id,omitempty"`
	MaturityState      string                    `json:"maturity_state,omitempty"`
	Organizing         bool                      `json:"organizing,omitempty"`
	OrganizingSummary  *CatalogOrganizingSummary `json:"organizing_summary,omitempty"`
	StoragePath        string                    `json:"storage_path,omitempty"`
	Type               string                    `json:"type"`
	Title              string                    `json:"title"`
	OriginalTitle      string                    `json:"original_title,omitempty"`
	SortTitle          string                    `json:"sort_title,omitempty"`
	Overview           string                    `json:"overview,omitempty"`
	Year               *int                      `json:"year,omitempty"`
	RuntimeSeconds     *int                      `json:"runtime_seconds,omitempty"`
	IndexNumber        *int                      `json:"index_number,omitempty"`
	IndexNumberEnd     *int                      `json:"index_number_end,omitempty"`
	ParentIndexNumber  *int                      `json:"parent_index_number,omitempty"`
	EpisodeLabel       string                    `json:"episode_label,omitempty"`
	CommunityRating    *float64                  `json:"community_rating,omitempty"`
	OfficialRating     string                    `json:"official_rating,omitempty"`
	SeriesStatus       string                    `json:"series_status,omitempty"`
	AvailabilityStatus string                    `json:"availability_status"`
	GovernanceStatus   string                    `json:"governance_status"`
	ReleaseDate        *time.Time                `json:"release_date,omitempty"`
	FirstAirDate       *time.Time                `json:"first_air_date,omitempty"`
	LastAirDate        *time.Time                `json:"last_air_date,omitempty"`
	ChildSummary       *CatalogChildSummary      `json:"child_summary,omitempty"`
	SelectedImages     []CatalogSelectedImage    `json:"selected_images,omitempty"`
	ExternalIdentities []CatalogExternalIdentity `json:"external_identities,omitempty"`
}

type CatalogOrganizingSummary struct {
	State      string                       `json:"state"`
	Message    string                       `json:"message"`
	Stage      string                       `json:"stage,omitempty"`
	Severity   string                       `json:"severity,omitempty"`
	Conditions []CatalogOrganizingCondition `json:"conditions,omitempty"`
}

type CatalogOrganizingCondition struct {
	Type     string `json:"type"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
	Message  string `json:"message,omitempty"`
	Severity string `json:"severity,omitempty"`
}

type CatalogItemDetail struct {
	ID                   uint                         `json:"id"`
	MetadataItemID       uint                         `json:"metadata_item_id,omitempty"`
	LibraryID            uint                         `json:"library_id"`
	ResourceCount        int                          `json:"resource_count,omitempty"`
	AvailableCount       int                          `json:"available_count,omitempty"`
	MissingCount         int                          `json:"missing_count,omitempty"`
	Type                 string                       `json:"type"`
	Path                 string                       `json:"path,omitempty"`
	Title                string                       `json:"title"`
	OriginalTitle        string                       `json:"original_title,omitempty"`
	SortTitle            string                       `json:"sort_title,omitempty"`
	Overview             string                       `json:"overview,omitempty"`
	Year                 *int                         `json:"year,omitempty"`
	EndYear              *int                         `json:"end_year,omitempty"`
	RuntimeSeconds       *int                         `json:"runtime_seconds,omitempty"`
	IndexNumber          *int                         `json:"index_number,omitempty"`
	ParentIndexNumber    *int                         `json:"parent_index_number,omitempty"`
	CommunityRating      *float64                     `json:"community_rating,omitempty"`
	OfficialRating       string                       `json:"official_rating,omitempty"`
	SeriesStatus         string                       `json:"series_status,omitempty"`
	AvailabilityStatus   string                       `json:"availability_status"`
	GovernanceStatus     string                       `json:"governance_status"`
	ReleaseDate          *time.Time                   `json:"release_date,omitempty"`
	FirstAirDate         *time.Time                   `json:"first_air_date,omitempty"`
	LastAirDate          *time.Time                   `json:"last_air_date,omitempty"`
	ChildSummary         *CatalogChildSummary         `json:"child_summary,omitempty"`
	SelectedImages       []CatalogSelectedImage       `json:"selected_images,omitempty"`
	ExternalIdentities   []CatalogExternalIdentity    `json:"external_identities,omitempty"`
	Tags                 []CatalogTagDetail           `json:"tags"`
	Genres               []string                     `json:"genres"`
	SourceEvidence       []CatalogSourceEvidence      `json:"source_evidence"`
	FieldStates          []CatalogFieldState          `json:"field_states"`
	Cast                 []CatalogPersonDetail        `json:"cast"`
	Directors            []CatalogPersonDetail        `json:"directors"`
	Seasons              []CatalogSeasonDetail        `json:"seasons"`
	Episodes             []CatalogEpisodeDetail       `json:"episodes"`
	EpisodeContext       *CatalogEpisodeParentContext `json:"episode_context,omitempty"`
	SeriesPlaybackTarget *CatalogSeriesPlaybackTarget `json:"series_playback_target,omitempty"`
	SameSeasonEpisodes   []CatalogEpisodeShelfItem    `json:"same_season_episodes"`
	Resources            []CatalogResourceDetailFull  `json:"resources"`
	RelatedItems         []CatalogListItem            `json:"related_items"`
}

type CatalogSeriesPlaybackTarget struct {
	EpisodeMetadataItemID uint   `json:"episode_metadata_item_id"`
	ResourceID      *uint  `json:"resource_id,omitempty"`
	Title           string `json:"title"`
	Label           string `json:"label,omitempty"`
	SelectionReason string `json:"selection_reason"`
}

type CatalogSeasonDetail struct {
	ID                 uint                      `json:"id"`
	LibraryID          uint                      `json:"library_id"`
	Type               string                    `json:"type"`
	Title              string                    `json:"title"`
	Overview           string                    `json:"overview,omitempty"`
	Year               *int                      `json:"year,omitempty"`
	IndexNumber        *int                      `json:"index_number,omitempty"`
	RuntimeSeconds     *int                      `json:"runtime_seconds,omitempty"`
	InventoryFileID    uint                      `json:"inventory_file_id,omitempty"`
	AvailabilityStatus string                    `json:"availability_status"`
	GovernanceStatus   string                    `json:"governance_status"`
	ChildSummary       *CatalogChildSummary      `json:"child_summary,omitempty"`
	SelectedImages     []CatalogSelectedImage    `json:"selected_images,omitempty"`
	ExternalIdentities []CatalogExternalIdentity `json:"external_identities,omitempty"`
	SourceEvidence     []CatalogSourceEvidence   `json:"source_evidence"`
	FieldStates        []CatalogFieldState       `json:"field_states"`
	Episodes           []CatalogEpisodeDetail    `json:"episodes"`
}

type CatalogEpisodeDetail struct {
	ID                 uint                      `json:"id"`
	LibraryID          uint                      `json:"library_id"`
	Type               string                    `json:"type"`
	Title              string                    `json:"title"`
	Overview           string                    `json:"overview,omitempty"`
	Year               *int                      `json:"year,omitempty"`
	ParentIndexNumber  *int                      `json:"parent_index_number,omitempty"`
	IndexNumber        *int                      `json:"index_number,omitempty"`
	IndexNumberEnd     *int                      `json:"index_number_end,omitempty"`
	AbsoluteNumber     *int                      `json:"absolute_number,omitempty"`
	RuntimeSeconds     *int                      `json:"runtime_seconds,omitempty"`
	InventoryFileID    uint                      `json:"inventory_file_id,omitempty"`
	AvailabilityStatus string                    `json:"availability_status"`
	GovernanceStatus   string                    `json:"governance_status"`
	ReleaseDate        *time.Time                `json:"release_date,omitempty"`
	FirstAirDate       *time.Time                `json:"first_air_date,omitempty"`
	SelectedImages     []CatalogSelectedImage    `json:"selected_images,omitempty"`
	ExternalIdentities []CatalogExternalIdentity `json:"external_identities,omitempty"`
	SourceEvidence     []CatalogSourceEvidence   `json:"source_evidence"`
	FieldStates        []CatalogFieldState       `json:"field_states"`
}

type CatalogResourceDetailFull struct {
	ID              uint                        `json:"id"`
	LibraryID       uint                        `json:"library_id"`
	ResourceType    string                      `json:"resource_type"`
	DisplayName     string                      `json:"display_name,omitempty"`
	Edition         string                      `json:"edition,omitempty"`
	QualityLabel    string                      `json:"quality_label,omitempty"`
	DurationSeconds *float64                    `json:"duration_seconds,omitempty"`
	Status          string                      `json:"status"`
	ProbeStatus     string                      `json:"probe_status"`
	FileIDs         []uint                      `json:"file_ids,omitempty"`
	Files           []CatalogResourceFileSummary `json:"files"`
	Streams         []CatalogMediaStreamSummary `json:"streams"`
	Links           []CatalogResourceLink       `json:"links"`
}

type CatalogResourceDetail struct {
	ID              uint     `json:"id"`
	LibraryID       uint     `json:"library_id,omitempty"`
	ResourceType    string   `json:"resource_type"`
	ResourceShape   string   `json:"resource_shape"`
	DisplayName     string   `json:"display_name,omitempty"`
	Edition         string   `json:"edition,omitempty"`
	QualityLabel    string   `json:"quality_label,omitempty"`
	DurationSeconds *float64 `json:"duration_seconds,omitempty"`
	Status          string   `json:"status"`
	ProbeStatus     string   `json:"probe_status"`
	Role            string   `json:"role"`
	SegmentIndex    int      `json:"segment_index,omitempty"`
	ReviewState     string   `json:"review_state,omitempty"`
}

type CatalogGovernanceWorkspace struct {
	MetadataItemID      uint                               `json:"metadata_item_id"`
	LibraryID           uint                               `json:"library_id"`
	Type                string                             `json:"type"`
	Title               string                             `json:"title"`
	AvailabilityStatus  string                             `json:"availability_status"`
	GovernanceStatus    string                             `json:"governance_status"`
	SelectedImages      []CatalogSelectedImage             `json:"selected_images,omitempty"`
	ImageCandidates     []CatalogSelectedImage             `json:"image_candidates,omitempty"`
	ExternalIdentities  []CatalogExternalIdentity          `json:"external_identities,omitempty"`
	SourceEvidence      []CatalogSourceEvidence            `json:"source_evidence"`
	FieldStates         []CatalogFieldState                `json:"field_states"`
	Resources           []CatalogResourceDetailFull        `json:"resources"`
	Classification      []CatalogClassificationDecision    `json:"classification_decisions"`
	ClassificationRules []CatalogClassificationRuleSummary `json:"classification_rules"`
	RecommendedChildren []CatalogListItem                  `json:"recommended_children"`
	MetadataOperation   any                                `json:"metadata_operation,omitempty"`
}

func ensureCatalogSelectedImages(images []CatalogSelectedImage) []CatalogSelectedImage {
	if images == nil {
		return []CatalogSelectedImage{}
	}
	return images
}

func ensureCatalogExternalIdentities(items []CatalogExternalIdentity) []CatalogExternalIdentity {
	if items == nil {
		return []CatalogExternalIdentity{}
	}
	return items
}

func buildCatalogGenres(tags []CatalogTagDetail) []string {
	if len(tags) == 0 {
		return []string{}
	}
	genres := make([]string, 0, len(tags))
	fallback := make([]string, 0, len(tags))
	seenGenres := make(map[string]struct{}, len(tags))
	seenFallback := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		name := strings.TrimSpace(tag.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seenFallback[key]; !ok {
			fallback = append(fallback, name)
			seenFallback[key] = struct{}{}
		}
		if strings.EqualFold(strings.TrimSpace(tag.Kind), "genre") {
			if _, ok := seenGenres[key]; ok {
				continue
			}
			genres = append(genres, name)
			seenGenres[key] = struct{}{}
		}
	}
	if len(genres) > 0 {
		return genres
	}
	return fallback
}

func ensureCatalogTagDetails(items []CatalogTagDetail) []CatalogTagDetail {
	if items == nil {
		return []CatalogTagDetail{}
	}
	return items
}

func ensureCatalogListItems(items []CatalogListItem) []CatalogListItem {
	if items == nil {
		return []CatalogListItem{}
	}
	return items
}

func ensureCatalogPersonDetails(items []CatalogPersonDetail) []CatalogPersonDetail {
	if items == nil {
		return []CatalogPersonDetail{}
	}
	return items
}

func ensureCatalogSeasonDetails(items []CatalogSeasonDetail) []CatalogSeasonDetail {
	if items == nil {
		return []CatalogSeasonDetail{}
	}
	return items
}

func ensureCatalogEpisodeDetails(items []CatalogEpisodeDetail) []CatalogEpisodeDetail {
	if items == nil {
		return []CatalogEpisodeDetail{}
	}
	return items
}

func ensureCatalogEpisodeShelfItems(items []CatalogEpisodeShelfItem) []CatalogEpisodeShelfItem {
	if items == nil {
		return []CatalogEpisodeShelfItem{}
	}
	return items
}

func ensureCatalogResourceDetails(items []CatalogResourceDetailFull) []CatalogResourceDetailFull {
	if items == nil {
		return []CatalogResourceDetailFull{}
	}
	return items
}

func firstNonZeroUint(values ...uint) uint {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func ensureCatalogResourceFileSummaries(items []CatalogResourceFileSummary) []CatalogResourceFileSummary {
	if items == nil {
		return []CatalogResourceFileSummary{}
	}
	return items
}

func ensureCatalogMediaStreamSummaries(items []CatalogMediaStreamSummary) []CatalogMediaStreamSummary {
	if items == nil {
		return []CatalogMediaStreamSummary{}
	}
	return items
}

func decodeCatalogJSONValue(raw string) (any, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, false
	}
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, false
	}
	return decoded, true
}

func isCatalogScalarJSONValue(value any) bool {
	switch value.(type) {
	case nil, string, float64, bool:
		return true
	default:
		return false
	}
}

func normalizeAvailabilityStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return AvailabilityNoLocalMedia
	}
	return status
}

func normalizeGovernanceStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return GovernancePending
	}
	return status
}
