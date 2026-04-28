package catalog

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
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
	SourceType string     `json:"source_type"`
	SourceName string     `json:"source_name"`
	Language   string     `json:"language,omitempty"`
	ExternalID string     `json:"external_id,omitempty"`
	Confidence *float64   `json:"confidence,omitempty"`
	FetchedAt  time.Time  `json:"fetched_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	Summary    any        `json:"summary,omitempty"`
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

type CatalogAssetLink struct {
	ItemID       uint     `json:"item_id"`
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
	AvailabilityStatus string                    `json:"availability_status"`
	GovernanceStatus   string                    `json:"governance_status"`
	ReleaseDate        *time.Time                `json:"release_date,omitempty"`
	FirstAirDate       *time.Time                `json:"first_air_date,omitempty"`
	SelectedImages     []CatalogSelectedImage    `json:"selected_images,omitempty"`
	ExternalIdentities []CatalogExternalIdentity `json:"external_identities,omitempty"`
	Current            bool                      `json:"current"`
	Progress           *CatalogUserProgressState `json:"progress,omitempty"`
}

type CatalogAssetFileSummary struct {
	FileID          uint       `json:"file_id"`
	Role            string     `json:"role"`
	PartIndex       int        `json:"part_index"`
	StorageProvider string     `json:"storage_provider"`
	StoragePath     string     `json:"storage_path,omitempty"`
	StableIdentity  string     `json:"stable_identity_key,omitempty"`
	SizeBytes       int64      `json:"size_bytes"`
	Container       string     `json:"container,omitempty"`
	Status          string     `json:"status"`
	ModifiedAt      *time.Time `json:"modified_at,omitempty"`
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
}

type CatalogListItem struct {
	ID                 uint                      `json:"id"`
	LibraryID          uint                      `json:"library_id"`
	Type               string                    `json:"type"`
	Title              string                    `json:"title"`
	OriginalTitle      string                    `json:"original_title,omitempty"`
	SortTitle          string                    `json:"sort_title,omitempty"`
	Overview           string                    `json:"overview,omitempty"`
	Year               *int                      `json:"year,omitempty"`
	RuntimeSeconds     *int                      `json:"runtime_seconds,omitempty"`
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

type CatalogItemDetail struct {
	ID                   uint                         `json:"id"`
	LibraryID            uint                         `json:"library_id"`
	Type                 string                       `json:"type"`
	Title                string                       `json:"title"`
	OriginalTitle        string                       `json:"original_title,omitempty"`
	SortTitle            string                       `json:"sort_title,omitempty"`
	Overview             string                       `json:"overview,omitempty"`
	Year                 *int                         `json:"year,omitempty"`
	EndYear              *int                         `json:"end_year,omitempty"`
	RuntimeSeconds       *int                         `json:"runtime_seconds,omitempty"`
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
	Assets               []CatalogAssetDetail         `json:"assets"`
	RelatedItems         []CatalogListItem            `json:"related_items"`
}

type CatalogSeriesPlaybackTarget struct {
	EpisodeItemID   uint   `json:"episode_item_id"`
	AssetID         *uint  `json:"asset_id,omitempty"`
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
	AvailabilityStatus string                    `json:"availability_status"`
	GovernanceStatus   string                    `json:"governance_status"`
	ReleaseDate        *time.Time                `json:"release_date,omitempty"`
	FirstAirDate       *time.Time                `json:"first_air_date,omitempty"`
	SelectedImages     []CatalogSelectedImage    `json:"selected_images,omitempty"`
	ExternalIdentities []CatalogExternalIdentity `json:"external_identities,omitempty"`
	SourceEvidence     []CatalogSourceEvidence   `json:"source_evidence"`
	FieldStates        []CatalogFieldState       `json:"field_states"`
	Assets             []CatalogAssetDetail      `json:"assets"`
}

type CatalogAssetDetail struct {
	ID              uint                        `json:"id"`
	LibraryID       uint                        `json:"library_id"`
	AssetType       string                      `json:"asset_type"`
	DisplayName     string                      `json:"display_name,omitempty"`
	Edition         string                      `json:"edition,omitempty"`
	QualityLabel    string                      `json:"quality_label,omitempty"`
	DurationSeconds *float64                    `json:"duration_seconds,omitempty"`
	Status          string                      `json:"status"`
	ProbeStatus     string                      `json:"probe_status"`
	FileIDs         []uint                      `json:"file_ids,omitempty"`
	Files           []CatalogAssetFileSummary   `json:"files"`
	Streams         []CatalogMediaStreamSummary `json:"streams"`
	Links           []CatalogAssetLink          `json:"links"`
}

type CatalogGovernanceWorkspace struct {
	ItemID              uint                            `json:"item_id"`
	LibraryID           uint                            `json:"library_id"`
	Type                string                          `json:"type"`
	Title               string                          `json:"title"`
	AvailabilityStatus  string                          `json:"availability_status"`
	GovernanceStatus    string                          `json:"governance_status"`
	SelectedImages      []CatalogSelectedImage          `json:"selected_images,omitempty"`
	ImageCandidates     []CatalogSelectedImage          `json:"image_candidates,omitempty"`
	ExternalIdentities  []CatalogExternalIdentity       `json:"external_identities,omitempty"`
	SourceEvidence      []CatalogSourceEvidence         `json:"source_evidence"`
	FieldStates         []CatalogFieldState             `json:"field_states"`
	Assets              []CatalogAssetDetail            `json:"assets"`
	RecommendedChildren []CatalogListItem               `json:"recommended_children"`
	MetadataResult      *CatalogMetadataOperationResult `json:"metadata_result,omitempty"`
}

type CatalogMetadataOperationResult struct {
	OriginItemID       uint   `json:"origin_item_id"`
	TargetItemID       uint   `json:"target_item_id"`
	TargetType         string `json:"target_type"`
	Action             string `json:"action"`
	DescendantStatus   string `json:"descendant_status,omitempty"`
	DescendantItemID   *uint  `json:"descendant_item_id,omitempty"`
	SeasonNumber       *int   `json:"season_number,omitempty"`
	EpisodeNumber      *int   `json:"episode_number,omitempty"`
	ProviderExternalID string `json:"provider_external_id,omitempty"`
	Message            string `json:"message,omitempty"`
}

type CatalogListItemInput struct {
	Item        database.CatalogItem
	Rollup      *database.ItemRollup
	Images      []database.ItemImage
	ExternalIDs []database.CatalogExternalID
}

type CatalogItemDetailInput struct {
	Item                 database.CatalogItem
	Rollup               *database.ItemRollup
	Images               []database.ItemImage
	ExternalIDs          []database.CatalogExternalID
	Sources              []database.MetadataSource
	FieldStates          []database.MetadataFieldState
	Cast                 []CatalogPersonDetail
	Directors            []CatalogPersonDetail
	Tags                 []CatalogTagDetail
	Seasons              []CatalogSeasonDetail
	Episodes             []CatalogEpisodeDetail
	EpisodeContext       *CatalogEpisodeParentContext
	SeriesPlaybackTarget *CatalogSeriesPlaybackTarget
	SameSeasonEpisodes   []CatalogEpisodeShelfItem
	Assets               []CatalogAssetDetail
	Related              []CatalogListItem
}

type CatalogSeasonDetailInput struct {
	Item        database.CatalogItem
	Rollup      *database.ItemRollup
	Images      []database.ItemImage
	ExternalIDs []database.CatalogExternalID
	Sources     []database.MetadataSource
	FieldStates []database.MetadataFieldState
	Episodes    []CatalogEpisodeDetail
}

type CatalogEpisodeDetailInput struct {
	Item        database.CatalogItem
	Images      []database.ItemImage
	ExternalIDs []database.CatalogExternalID
	Sources     []database.MetadataSource
	FieldStates []database.MetadataFieldState
	Assets      []CatalogAssetDetail
}

type CatalogAssetDetailInput struct {
	Asset   database.MediaAsset
	Links   []database.AssetItem
	FileIDs []uint
	Files   []CatalogAssetFileSummary
	Streams []CatalogMediaStreamSummary
}

type CatalogEpisodeShelfItemInput struct {
	Episode       CatalogEpisodeDetail
	CurrentItemID uint
	Progress      *CatalogUserProgressState
}

type CatalogGovernanceWorkspaceInput struct {
	Item                database.CatalogItem
	Images              []database.ItemImage
	ExternalIDs         []database.CatalogExternalID
	Sources             []database.MetadataSource
	FieldStates         []database.MetadataFieldState
	Assets              []CatalogAssetDetail
	RecommendedChildren []CatalogListItem
}

func BuildCatalogListItem(input CatalogListItemInput) CatalogListItem {
	item := input.Item
	return CatalogListItem{
		ID:                 item.ID,
		LibraryID:          item.LibraryID,
		Type:               normalizeCatalogType(item.Type),
		Title:              strings.TrimSpace(item.Title),
		OriginalTitle:      strings.TrimSpace(item.OriginalTitle),
		SortTitle:          strings.TrimSpace(item.SortTitle),
		Overview:           item.Overview,
		Year:               item.Year,
		RuntimeSeconds:     item.RuntimeSeconds,
		CommunityRating:    item.CommunityRating,
		OfficialRating:     strings.TrimSpace(item.OfficialRating),
		SeriesStatus:       strings.TrimSpace(item.SeriesStatus),
		AvailabilityStatus: normalizeAvailabilityStatus(item.AvailabilityStatus),
		GovernanceStatus:   normalizeGovernanceStatus(item.GovernanceStatus),
		ReleaseDate:        item.ReleaseDate,
		FirstAirDate:       item.FirstAirDate,
		LastAirDate:        item.LastAirDate,
		ChildSummary:       buildCatalogChildSummary(input.Rollup),
		SelectedImages:     buildCatalogSelectedImages(input.Images),
		ExternalIdentities: buildCatalogExternalIdentities(input.ExternalIDs),
	}
}

func BuildCatalogItemDetail(input CatalogItemDetailInput) CatalogItemDetail {
	item := input.Item
	return CatalogItemDetail{
		ID:                   item.ID,
		LibraryID:            item.LibraryID,
		Type:                 normalizeCatalogType(item.Type),
		Title:                strings.TrimSpace(item.Title),
		OriginalTitle:        strings.TrimSpace(item.OriginalTitle),
		SortTitle:            strings.TrimSpace(item.SortTitle),
		Overview:             item.Overview,
		Year:                 item.Year,
		EndYear:              item.EndYear,
		RuntimeSeconds:       item.RuntimeSeconds,
		CommunityRating:      item.CommunityRating,
		OfficialRating:       strings.TrimSpace(item.OfficialRating),
		SeriesStatus:         strings.TrimSpace(item.SeriesStatus),
		AvailabilityStatus:   normalizeAvailabilityStatus(item.AvailabilityStatus),
		GovernanceStatus:     normalizeGovernanceStatus(item.GovernanceStatus),
		ReleaseDate:          item.ReleaseDate,
		FirstAirDate:         item.FirstAirDate,
		LastAirDate:          item.LastAirDate,
		ChildSummary:         buildCatalogChildSummary(input.Rollup),
		SelectedImages:       buildCatalogSelectedImages(input.Images),
		ExternalIdentities:   buildCatalogExternalIdentities(input.ExternalIDs),
		Tags:                 ensureCatalogTagDetails(input.Tags),
		Genres:               buildCatalogGenres(input.Tags),
		SourceEvidence:       buildCatalogSourceEvidence(input.Sources),
		FieldStates:          buildCatalogFieldStates(input.FieldStates),
		Cast:                 ensureCatalogPersonDetails(input.Cast),
		Directors:            ensureCatalogPersonDetails(input.Directors),
		Seasons:              ensureCatalogSeasonDetails(input.Seasons),
		Episodes:             ensureCatalogEpisodeDetails(input.Episodes),
		EpisodeContext:       input.EpisodeContext,
		SeriesPlaybackTarget: input.SeriesPlaybackTarget,
		SameSeasonEpisodes:   ensureCatalogEpisodeShelfItems(input.SameSeasonEpisodes),
		Assets:               ensureCatalogAssetDetails(input.Assets),
		RelatedItems:         ensureCatalogListItems(input.Related),
	}
}

func BuildCatalogSeasonDetail(input CatalogSeasonDetailInput) CatalogSeasonDetail {
	item := input.Item
	return CatalogSeasonDetail{
		ID:                 item.ID,
		LibraryID:          item.LibraryID,
		Type:               normalizeCatalogType(item.Type),
		Title:              strings.TrimSpace(item.Title),
		Overview:           item.Overview,
		Year:               item.Year,
		IndexNumber:        item.IndexNumber,
		RuntimeSeconds:     item.RuntimeSeconds,
		AvailabilityStatus: normalizeAvailabilityStatus(item.AvailabilityStatus),
		GovernanceStatus:   normalizeGovernanceStatus(item.GovernanceStatus),
		ChildSummary:       buildCatalogChildSummary(input.Rollup),
		SelectedImages:     buildCatalogSelectedImages(input.Images),
		ExternalIdentities: buildCatalogExternalIdentities(input.ExternalIDs),
		SourceEvidence:     buildCatalogSourceEvidence(input.Sources),
		FieldStates:        buildCatalogFieldStates(input.FieldStates),
		Episodes:           ensureCatalogEpisodeDetails(input.Episodes),
	}
}

func BuildCatalogEpisodeDetail(input CatalogEpisodeDetailInput) CatalogEpisodeDetail {
	item := input.Item
	return CatalogEpisodeDetail{
		ID:                 item.ID,
		LibraryID:          item.LibraryID,
		Type:               normalizeCatalogType(item.Type),
		Title:              strings.TrimSpace(item.Title),
		Overview:           item.Overview,
		Year:               item.Year,
		ParentIndexNumber:  item.ParentIndexNumber,
		IndexNumber:        item.IndexNumber,
		IndexNumberEnd:     item.IndexNumberEnd,
		AbsoluteNumber:     item.AbsoluteNumber,
		RuntimeSeconds:     item.RuntimeSeconds,
		AvailabilityStatus: normalizeAvailabilityStatus(item.AvailabilityStatus),
		GovernanceStatus:   normalizeGovernanceStatus(item.GovernanceStatus),
		ReleaseDate:        item.ReleaseDate,
		FirstAirDate:       item.FirstAirDate,
		SelectedImages:     buildCatalogSelectedImages(input.Images),
		ExternalIdentities: buildCatalogExternalIdentities(input.ExternalIDs),
		SourceEvidence:     buildCatalogSourceEvidence(input.Sources),
		FieldStates:        buildCatalogFieldStates(input.FieldStates),
		Assets:             ensureCatalogAssetDetails(input.Assets),
	}
}

func BuildCatalogAssetDetail(input CatalogAssetDetailInput) CatalogAssetDetail {
	asset := input.Asset
	return CatalogAssetDetail{
		ID:              asset.ID,
		LibraryID:       asset.LibraryID,
		AssetType:       strings.TrimSpace(asset.AssetType),
		DisplayName:     strings.TrimSpace(asset.DisplayName),
		Edition:         strings.TrimSpace(asset.Edition),
		QualityLabel:    strings.TrimSpace(asset.QualityLabel),
		DurationSeconds: asset.DurationSeconds,
		Status:          normalizeAvailabilityStatus(asset.Status),
		ProbeStatus:     strings.TrimSpace(asset.ProbeStatus),
		FileIDs:         append([]uint(nil), input.FileIDs...),
		Files:           ensureCatalogAssetFileSummaries(input.Files),
		Streams:         ensureCatalogMediaStreamSummaries(input.Streams),
		Links:           buildCatalogAssetLinks(input.Links),
	}
}

func BuildCatalogEpisodeShelfItem(input CatalogEpisodeShelfItemInput) CatalogEpisodeShelfItem {
	episode := input.Episode
	return CatalogEpisodeShelfItem{
		ID:                 episode.ID,
		LibraryID:          episode.LibraryID,
		Type:               episode.Type,
		Title:              strings.TrimSpace(episode.Title),
		Label:              formatCatalogEpisodeLabel(episode.ParentIndexNumber, episode.IndexNumber, episode.IndexNumberEnd),
		Overview:           episode.Overview,
		SeasonNumber:       episode.ParentIndexNumber,
		EpisodeNumber:      episode.IndexNumber,
		EpisodeNumberEnd:   episode.IndexNumberEnd,
		RuntimeSeconds:     episode.RuntimeSeconds,
		AvailabilityStatus: normalizeAvailabilityStatus(episode.AvailabilityStatus),
		GovernanceStatus:   normalizeGovernanceStatus(episode.GovernanceStatus),
		ReleaseDate:        episode.ReleaseDate,
		FirstAirDate:       episode.FirstAirDate,
		SelectedImages:     ensureCatalogSelectedImages(episode.SelectedImages),
		ExternalIdentities: ensureCatalogExternalIdentities(episode.ExternalIdentities),
		Current:            episode.ID == input.CurrentItemID,
		Progress:           input.Progress,
	}
}

func BuildCatalogEpisodeParentContext(series *database.CatalogItem, season *database.CatalogItem, seriesImages []database.ItemImage, seasonImages []database.ItemImage, episode database.CatalogItem) *CatalogEpisodeParentContext {
	if episode.Type != ItemTypeEpisode {
		return nil
	}
	context := &CatalogEpisodeParentContext{
		SeasonNumber:        episode.ParentIndexNumber,
		EpisodeNumber:       episode.IndexNumber,
		EpisodeNumberEnd:    episode.IndexNumberEnd,
		IncompleteHierarchy: series == nil || season == nil,
	}
	if series != nil {
		context.Series = &CatalogEpisodeSeriesContext{
			ID:             series.ID,
			Title:          strings.TrimSpace(series.Title),
			SelectedImages: buildCatalogSelectedImages(seriesImages),
		}
	}
	if season != nil {
		context.Season = &CatalogEpisodeSeasonContext{
			ID:             season.ID,
			Title:          strings.TrimSpace(season.Title),
			Number:         season.IndexNumber,
			SelectedImages: buildCatalogSelectedImages(seasonImages),
		}
		if context.SeasonNumber == nil {
			context.SeasonNumber = season.IndexNumber
		}
	}
	return context
}

func BuildCatalogGovernanceWorkspace(input CatalogGovernanceWorkspaceInput) CatalogGovernanceWorkspace {
	item := input.Item
	return CatalogGovernanceWorkspace{
		ItemID:              item.ID,
		LibraryID:           item.LibraryID,
		Type:                normalizeCatalogType(item.Type),
		Title:               strings.TrimSpace(item.Title),
		AvailabilityStatus:  normalizeAvailabilityStatus(item.AvailabilityStatus),
		GovernanceStatus:    normalizeGovernanceStatus(item.GovernanceStatus),
		SelectedImages:      buildCatalogSelectedImages(input.Images),
		ImageCandidates:     buildCatalogImageCandidates(input.Images),
		ExternalIdentities:  buildCatalogExternalIdentities(input.ExternalIDs),
		SourceEvidence:      buildCatalogSourceEvidence(input.Sources),
		FieldStates:         buildCatalogFieldStates(input.FieldStates),
		Assets:              ensureCatalogAssetDetails(input.Assets),
		RecommendedChildren: ensureCatalogListItems(input.RecommendedChildren),
	}
}

func buildCatalogSelectedImages(images []database.ItemImage) []CatalogSelectedImage {
	if images == nil {
		return []CatalogSelectedImage{}
	}
	selected := make([]CatalogSelectedImage, 0, len(images))
	for _, image := range images {
		if !image.IsSelected {
			continue
		}
		selected = append(selected, CatalogSelectedImage{
			ImageType: strings.TrimSpace(image.ImageType),
			URL:       strings.TrimSpace(image.URL),
			Language:  strings.TrimSpace(image.Language),
			Width:     image.Width,
			Height:    image.Height,
		})
	}
	return selected
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

func buildCatalogImageCandidates(images []database.ItemImage) []CatalogSelectedImage {
	if images == nil {
		return []CatalogSelectedImage{}
	}
	candidates := make([]CatalogSelectedImage, 0, len(images))
	for _, image := range images {
		candidates = append(candidates, CatalogSelectedImage{
			ImageType: strings.TrimSpace(image.ImageType),
			URL:       strings.TrimSpace(image.URL),
			Language:  strings.TrimSpace(image.Language),
			Width:     image.Width,
			Height:    image.Height,
		})
	}
	return candidates
}

func buildCatalogExternalIdentities(externalIDs []database.CatalogExternalID) []CatalogExternalIdentity {
	if externalIDs == nil {
		return []CatalogExternalIdentity{}
	}
	identities := make([]CatalogExternalIdentity, 0, len(externalIDs))
	for _, externalID := range externalIDs {
		identities = append(identities, CatalogExternalIdentity{
			Provider:     strings.TrimSpace(externalID.Provider),
			ProviderType: strings.TrimSpace(externalID.ProviderType),
			ExternalID:   strings.TrimSpace(externalID.ExternalID),
			IsPrimary:    externalID.IsPrimary,
			Source:       strings.TrimSpace(externalID.Source),
			Confidence:   externalID.Confidence,
		})
	}
	return identities
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

func buildCatalogSourceEvidence(sources []database.MetadataSource) []CatalogSourceEvidence {
	if sources == nil {
		return []CatalogSourceEvidence{}
	}
	evidence := make([]CatalogSourceEvidence, 0, len(sources))
	for _, source := range sources {
		evidence = append(evidence, CatalogSourceEvidence{
			SourceType: strings.TrimSpace(source.SourceType),
			SourceName: strings.TrimSpace(source.SourceName),
			Language:   strings.TrimSpace(source.Language),
			ExternalID: strings.TrimSpace(source.ExternalID),
			Confidence: source.Confidence,
			FetchedAt:  source.FetchedAt,
			ExpiresAt:  source.ExpiresAt,
			Summary:    projectCatalogSourceSummary(source.PayloadJSON),
		})
	}
	return evidence
}

func buildCatalogFieldStates(states []database.MetadataFieldState) []CatalogFieldState {
	if states == nil {
		return []CatalogFieldState{}
	}
	fieldStates := make([]CatalogFieldState, 0, len(states))
	for _, state := range states {
		fieldStates = append(fieldStates, CatalogFieldState{
			FieldKey:       strings.TrimSpace(state.FieldKey),
			SourceID:       state.SourceID,
			Value:          projectCatalogFieldStateValue(state.ValueJSON),
			IsLocked:       state.IsLocked,
			LockReason:     strings.TrimSpace(state.LockReason),
			EditedByUserID: state.EditedByUserID,
			EditedAt:       state.EditedAt,
		})
	}
	return fieldStates
}

func buildCatalogChildSummary(rollup *database.ItemRollup) *CatalogChildSummary {
	if rollup == nil {
		return nil
	}
	return &CatalogChildSummary{
		ChildCount:      rollup.ChildCount,
		AvailableCount:  rollup.AvailableCount,
		MissingCount:    rollup.MissingCount,
		UnairedCount:    rollup.UnairedCount,
		PlayedCount:     rollup.PlayedCount,
		InProgressCount: rollup.InProgressCount,
		LatestAirDate:   rollup.LatestAirDate,
		LatestAddedAt:   rollup.LatestAddedAt,
	}
}

func buildCatalogAssetLinks(links []database.AssetItem) []CatalogAssetLink {
	if links == nil {
		return []CatalogAssetLink{}
	}
	assetLinks := make([]CatalogAssetLink, 0, len(links))
	for _, link := range links {
		assetLinks = append(assetLinks, CatalogAssetLink{
			ItemID:       link.ItemID,
			Role:         strings.TrimSpace(link.Role),
			SegmentIndex: link.SegmentIndex,
			StartSeconds: link.StartSeconds,
			EndSeconds:   link.EndSeconds,
			Confidence:   link.Confidence,
			Source:       strings.TrimSpace(link.Source),
		})
	}
	return assetLinks
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

func ensureCatalogAssetDetails(items []CatalogAssetDetail) []CatalogAssetDetail {
	if items == nil {
		return []CatalogAssetDetail{}
	}
	return items
}

func ensureCatalogAssetFileSummaries(items []CatalogAssetFileSummary) []CatalogAssetFileSummary {
	if items == nil {
		return []CatalogAssetFileSummary{}
	}
	return items
}

func ensureCatalogMediaStreamSummaries(items []CatalogMediaStreamSummary) []CatalogMediaStreamSummary {
	if items == nil {
		return []CatalogMediaStreamSummary{}
	}
	return items
}

func formatCatalogEpisodeLabel(seasonNumber, episodeNumber, episodeNumberEnd *int) string {
	if seasonNumber == nil && episodeNumber == nil {
		return ""
	}
	var builder strings.Builder
	if seasonNumber != nil {
		builder.WriteString("S")
		builder.WriteString(strconv.Itoa(*seasonNumber))
	}
	if episodeNumber != nil {
		if builder.Len() > 0 {
			builder.WriteString(":")
		}
		builder.WriteString("E")
		builder.WriteString(strconv.Itoa(*episodeNumber))
		if episodeNumberEnd != nil && *episodeNumberEnd != *episodeNumber {
			builder.WriteString("-E")
			builder.WriteString(strconv.Itoa(*episodeNumberEnd))
		}
	}
	return builder.String()
}

func projectCatalogSourceSummary(raw string) any {
	decoded, ok := decodeCatalogJSONValue(raw)
	if !ok {
		return nil
	}
	payload, ok := decoded.(map[string]any)
	if !ok {
		return nil
	}

	const scalarSummaryKeys = "title,name,original_title,overview,release_date,first_air_date,last_air_date,runtime,status,media_type,external_id,matched_title,air_date,poster_path,still_path,series_tmdb_id,storage_path,stable_identity_key,provider_name,detected_title,series_title,season_number,episode_number"
	summary := make(map[string]any)
	for _, key := range strings.Split(scalarSummaryKeys, ",") {
		value, exists := payload[key]
		if !exists || !isCatalogScalarJSONValue(value) {
			continue
		}
		summary[key] = value
	}
	if len(summary) == 0 {
		return nil
	}
	return summary
}

func projectCatalogFieldStateValue(raw string) any {
	decoded, ok := decodeCatalogJSONValue(raw)
	if !ok || !isCatalogScalarJSONValue(decoded) {
		return nil
	}
	return decoded
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

func normalizeCatalogType(itemType string) string {
	return strings.TrimSpace(itemType)
}
