package catalog

import (
	"encoding/json"
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
	ID                 uint                      `json:"id"`
	LibraryID          uint                      `json:"library_id"`
	Type               string                    `json:"type"`
	Title              string                    `json:"title"`
	OriginalTitle      string                    `json:"original_title,omitempty"`
	SortTitle          string                    `json:"sort_title,omitempty"`
	Overview           string                    `json:"overview,omitempty"`
	Year               *int                      `json:"year,omitempty"`
	EndYear            *int                      `json:"end_year,omitempty"`
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
	SourceEvidence     []CatalogSourceEvidence   `json:"source_evidence"`
	FieldStates        []CatalogFieldState       `json:"field_states"`
	Seasons            []CatalogSeasonDetail     `json:"seasons"`
	Episodes           []CatalogEpisodeDetail    `json:"episodes"`
	Assets             []CatalogAssetDetail      `json:"assets"`
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
	ID              uint               `json:"id"`
	LibraryID       uint               `json:"library_id"`
	AssetType       string             `json:"asset_type"`
	DisplayName     string             `json:"display_name,omitempty"`
	Edition         string             `json:"edition,omitempty"`
	QualityLabel    string             `json:"quality_label,omitempty"`
	DurationSeconds *float64           `json:"duration_seconds,omitempty"`
	Status          string             `json:"status"`
	ProbeStatus     string             `json:"probe_status"`
	Links           []CatalogAssetLink `json:"links"`
}

type CatalogGovernanceWorkspace struct {
	ItemID              uint                      `json:"item_id"`
	LibraryID           uint                      `json:"library_id"`
	Type                string                    `json:"type"`
	Title               string                    `json:"title"`
	AvailabilityStatus  string                    `json:"availability_status"`
	GovernanceStatus    string                    `json:"governance_status"`
	SelectedImages      []CatalogSelectedImage    `json:"selected_images,omitempty"`
	ExternalIdentities  []CatalogExternalIdentity `json:"external_identities,omitempty"`
	SourceEvidence      []CatalogSourceEvidence   `json:"source_evidence"`
	FieldStates         []CatalogFieldState       `json:"field_states"`
	Assets              []CatalogAssetDetail      `json:"assets"`
	RecommendedChildren []CatalogListItem         `json:"recommended_children"`
}

type CatalogListItemInput struct {
	Item        database.CatalogItem
	Rollup      *database.ItemRollup
	Images      []database.ItemImage
	ExternalIDs []database.CatalogExternalID
}

type CatalogItemDetailInput struct {
	Item        database.CatalogItem
	Rollup      *database.ItemRollup
	Images      []database.ItemImage
	ExternalIDs []database.CatalogExternalID
	Sources     []database.MetadataSource
	FieldStates []database.MetadataFieldState
	Seasons     []CatalogSeasonDetail
	Episodes    []CatalogEpisodeDetail
	Assets      []CatalogAssetDetail
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
	Asset database.MediaAsset
	Links []database.AssetItem
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
		ID:                 item.ID,
		LibraryID:          item.LibraryID,
		Type:               normalizeCatalogType(item.Type),
		Title:              strings.TrimSpace(item.Title),
		OriginalTitle:      strings.TrimSpace(item.OriginalTitle),
		SortTitle:          strings.TrimSpace(item.SortTitle),
		Overview:           item.Overview,
		Year:               item.Year,
		EndYear:            item.EndYear,
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
		SourceEvidence:     buildCatalogSourceEvidence(input.Sources),
		FieldStates:        buildCatalogFieldStates(input.FieldStates),
		Seasons:            ensureCatalogSeasonDetails(input.Seasons),
		Episodes:           ensureCatalogEpisodeDetails(input.Episodes),
		Assets:             ensureCatalogAssetDetails(input.Assets),
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
		Links:           buildCatalogAssetLinks(input.Links),
	}
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

func ensureCatalogAssetDetails(items []CatalogAssetDetail) []CatalogAssetDetail {
	if items == nil {
		return []CatalogAssetDetail{}
	}
	return items
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

	const scalarSummaryKeys = "title,name,original_title,overview,release_date,first_air_date,last_air_date,runtime,status,season_number,episode_number"
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
	itemType = strings.TrimSpace(itemType)
	legacySeriesType := "sh" + "ow"
	if itemType == legacySeriesType {
		return ItemTypeSeries
	}
	return itemType
}
