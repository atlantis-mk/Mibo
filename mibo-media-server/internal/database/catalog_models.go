package database

import "time"

type CatalogItem struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	LibraryID           uint       `gorm:"not null;index;index:idx_catalog_items_library_type_availability_sort,priority:1" json:"library_id"`
	Type                string     `gorm:"size:32;not null;index;index:idx_catalog_items_library_type_availability_sort,priority:2;index:idx_catalog_items_root_type_order,priority:2" json:"type"`
	ParentID            *uint      `gorm:"index;index:idx_catalog_items_parent_order,priority:1" json:"parent_id,omitempty"`
	RootID              *uint      `gorm:"index;index:idx_catalog_items_root_type_order,priority:1" json:"root_id,omitempty"`
	Path                string     `gorm:"size:2048;index" json:"path"`
	SortKey             string     `gorm:"size:512;index;index:idx_catalog_items_library_type_availability_sort,priority:4" json:"sort_key"`
	DisplayOrder        string     `gorm:"size:32;not null;default:aired" json:"display_order"`
	IndexNumber         *int       `gorm:"index;index:idx_catalog_items_parent_order,priority:3;index:idx_catalog_items_root_type_order,priority:4" json:"index_number,omitempty"`
	IndexNumberEnd      *int       `json:"index_number_end,omitempty"`
	ParentIndexNumber   *int       `gorm:"index;index:idx_catalog_items_parent_order,priority:2;index:idx_catalog_items_root_type_order,priority:3" json:"parent_index_number,omitempty"`
	AbsoluteNumber      *int       `gorm:"index" json:"absolute_number,omitempty"`
	Title               string     `gorm:"size:512;not null;index" json:"title"`
	OriginalTitle       string     `gorm:"size:512" json:"original_title"`
	SortTitle           string     `gorm:"size:512;index" json:"sort_title"`
	Overview            string     `gorm:"type:text" json:"overview"`
	ReleaseDate         *time.Time `json:"release_date,omitempty"`
	FirstAirDate        *time.Time `gorm:"index" json:"first_air_date,omitempty"`
	LastAirDate         *time.Time `json:"last_air_date,omitempty"`
	Year                *int       `gorm:"index" json:"year,omitempty"`
	EndYear             *int       `json:"end_year,omitempty"`
	RuntimeSeconds      *int       `json:"runtime_seconds,omitempty"`
	CommunityRating     *float64   `json:"community_rating,omitempty"`
	OfficialRating      string     `gorm:"size:64" json:"official_rating"`
	SeriesStatus        string     `gorm:"size:64" json:"series_status"`
	AvailabilityStatus  string     `gorm:"size:64;not null;default:no_local_media;index;index:idx_catalog_items_library_type_availability_sort,priority:3" json:"availability_status"`
	MissingSince        *time.Time `gorm:"index" json:"missing_since,omitempty"`
	GovernanceStatus    string     `gorm:"size:64;not null;default:pending;index" json:"governance_status"`
	CanonicalVersion    int        `gorm:"not null;default:1" json:"canonical_version"`
	LastCanonicalizedAt *time.Time `json:"last_canonicalized_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	DeletedAt           *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

type CatalogExternalID struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ItemID       uint      `gorm:"not null;index" json:"item_id"`
	Provider     string    `gorm:"size:64;not null;uniqueIndex:idx_catalog_external_identity" json:"provider"`
	ProviderType string    `gorm:"size:64;not null;uniqueIndex:idx_catalog_external_identity" json:"provider_type"`
	ExternalID   string    `gorm:"size:255;not null;uniqueIndex:idx_catalog_external_identity" json:"external_id"`
	IsPrimary    bool      `gorm:"not null;default:false" json:"is_primary"`
	Source       string    `gorm:"size:64" json:"source"`
	Confidence   *float64  `json:"confidence,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CatalogIdentity struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ItemID       uint      `gorm:"not null;index" json:"item_id"`
	Provider     string    `gorm:"size:64;not null;uniqueIndex:idx_catalog_identity_key" json:"provider"`
	IdentityType string    `gorm:"size:64;not null;uniqueIndex:idx_catalog_identity_key" json:"identity_type"`
	IdentityKey  string    `gorm:"size:1024;not null;uniqueIndex:idx_catalog_identity_key" json:"identity_key"`
	SourcePath   string    `gorm:"size:2048;index" json:"source_path"`
	Confidence   *float64  `json:"confidence,omitempty"`
	EvidenceJSON string    `gorm:"type:text" json:"evidence_json"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type MetadataSource struct {
	ID                   uint       `gorm:"primaryKey" json:"id"`
	ItemID               uint       `gorm:"not null;index" json:"item_id"`
	SourceType           string     `gorm:"size:64;not null;index" json:"source_type"`
	SourceName           string     `gorm:"size:128;not null;index" json:"source_name"`
	Language             string     `gorm:"size:32;index" json:"language"`
	ExternalID           string     `gorm:"size:255;index" json:"external_id"`
	MetadataProfileID    *uint      `gorm:"index" json:"metadata_profile_id,omitempty"`
	MetadataProfileName  string     `gorm:"size:255;index" json:"metadata_profile_name"`
	ProviderInstanceID   *uint      `gorm:"index" json:"provider_instance_id,omitempty"`
	ProviderInstanceName string     `gorm:"size:255;index" json:"provider_instance_name"`
	FallbackSummaryJSON  string     `gorm:"type:text" json:"fallback_summary_json"`
	PayloadJSON          string     `gorm:"type:text" json:"payload_json"`
	Confidence           *float64   `json:"confidence,omitempty"`
	FetchedAt            time.Time  `gorm:"not null;index" json:"fetched_at"`
	ExpiresAt            *time.Time `gorm:"index" json:"expires_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type MetadataFieldState struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	ItemID         uint       `gorm:"not null;uniqueIndex:idx_metadata_field_state_item_field" json:"item_id"`
	FieldKey       string     `gorm:"size:128;not null;uniqueIndex:idx_metadata_field_state_item_field" json:"field_key"`
	SourceID       *uint      `gorm:"index" json:"source_id,omitempty"`
	ValueJSON      string     `gorm:"type:text;not null" json:"value_json"`
	IsLocked       bool       `gorm:"not null;default:false;index" json:"is_locked"`
	LockReason     string     `gorm:"size:255" json:"lock_reason"`
	EditedByUserID *uint      `gorm:"index" json:"edited_by_user_id,omitempty"`
	EditedAt       *time.Time `json:"edited_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type MetadataOperation struct {
	ID                    uint       `gorm:"primaryKey" json:"id"`
	Operation             string     `gorm:"size:64;not null;index" json:"operation"`
	OriginItemID          uint       `gorm:"not null;index" json:"origin_item_id"`
	TargetItemID          uint       `gorm:"not null;index" json:"target_item_id"`
	LibraryID             uint       `gorm:"not null;index" json:"library_id"`
	Status                string     `gorm:"size:64;not null;index" json:"status"`
	GovernanceStatus      string     `gorm:"size:64;index" json:"governance_status"`
	PlanJSON              string     `gorm:"type:text" json:"plan_json"`
	AttemptsJSON          string     `gorm:"type:text" json:"attempts_json"`
	SelectedCandidateJSON string     `gorm:"type:text" json:"selected_candidate_json"`
	MetadataSourceIDsJSON string     `gorm:"type:text" json:"metadata_source_ids_json"`
	AppliedFieldsJSON     string     `gorm:"type:text" json:"applied_fields_json"`
	SkippedFieldsJSON     string     `gorm:"type:text" json:"skipped_fields_json"`
	WarningsJSON          string     `gorm:"type:text" json:"warnings_json"`
	StartedAt             time.Time  `gorm:"not null;index" json:"started_at"`
	FinishedAt            *time.Time `gorm:"index" json:"finished_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type ItemImage struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ItemID     uint      `gorm:"not null;index" json:"item_id"`
	ImageType  string    `gorm:"size:64;not null;index" json:"image_type"`
	URL        string    `gorm:"size:2048;not null" json:"url"`
	SourceID   *uint     `gorm:"index" json:"source_id,omitempty"`
	Language   string    `gorm:"size:32;index" json:"language"`
	Width      *int      `json:"width,omitempty"`
	Height     *int      `json:"height,omitempty"`
	IsSelected bool      `gorm:"not null;default:false;index" json:"is_selected"`
	SortOrder  int       `gorm:"not null;default:0;index" json:"sort_order"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Person struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	Name               string     `gorm:"size:255;not null;uniqueIndex" json:"name"`
	SortName           string     `gorm:"size:255;index" json:"sort_name"`
	AvatarURL          string     `gorm:"size:2048" json:"avatar_url"`
	TMDBPersonID       *int       `gorm:"index" json:"tmdb_person_id,omitempty"`
	IMDBID             string     `gorm:"size:32" json:"imdb_id"`
	Biography          string     `gorm:"type:text" json:"biography"`
	Birthday           *time.Time `json:"birthday,omitempty"`
	Deathday           *time.Time `json:"deathday,omitempty"`
	PlaceOfBirth       string     `gorm:"size:255" json:"place_of_birth"`
	KnownForDepartment string     `gorm:"size:128" json:"known_for_department"`
	ProfileRefreshedAt *time.Time `gorm:"index" json:"profile_refreshed_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type ItemPerson struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ItemID    uint      `gorm:"not null;uniqueIndex:idx_item_people_item_person_role" json:"item_id"`
	PersonID  uint      `gorm:"not null;uniqueIndex:idx_item_people_item_person_role" json:"person_id"`
	Role      string    `gorm:"size:128;not null;uniqueIndex:idx_item_people_item_person_role" json:"role"`
	Character string    `gorm:"size:255" json:"character"`
	SortOrder int       `gorm:"not null;default:0;index" json:"sort_order"`
	SourceID  *uint     `gorm:"index" json:"source_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Tag struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Kind      string    `gorm:"size:64;not null;uniqueIndex:idx_tags_kind_name" json:"kind"`
	Name      string    `gorm:"size:255;not null;uniqueIndex:idx_tags_kind_name" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ItemTag struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ItemID    uint      `gorm:"not null;uniqueIndex:idx_item_tags_item_tag" json:"item_id"`
	TagID     uint      `gorm:"not null;uniqueIndex:idx_item_tags_item_tag" json:"tag_id"`
	SourceID  *uint     `gorm:"index" json:"source_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserItemData struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	UserID           uint       `gorm:"not null;uniqueIndex:idx_user_item_data_user_item_asset" json:"user_id"`
	ItemID           uint       `gorm:"not null;uniqueIndex:idx_user_item_data_user_item_asset;index" json:"item_id"`
	AssetID          *uint      `gorm:"uniqueIndex:idx_user_item_data_user_item_asset;index" json:"asset_id,omitempty"`
	PositionSeconds  int        `gorm:"not null;default:0" json:"position_seconds"`
	PlayedPercentage *float64   `json:"played_percentage,omitempty"`
	PlayCount        int        `gorm:"not null;default:0" json:"play_count"`
	Favorite         bool       `gorm:"not null;default:false;index" json:"favorite"`
	LastPlayedAt     *time.Time `gorm:"index" json:"last_played_at,omitempty"`
	CompletedAt      *time.Time `gorm:"index" json:"completed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type ItemRollup struct {
	ItemID          uint       `gorm:"primaryKey" json:"item_id"`
	ChildCount      int        `gorm:"not null;default:0" json:"child_count"`
	AvailableCount  int        `gorm:"not null;default:0" json:"available_count"`
	MissingCount    int        `gorm:"not null;default:0" json:"missing_count"`
	UnairedCount    int        `gorm:"not null;default:0" json:"unaired_count"`
	PlayedCount     int        `gorm:"not null;default:0" json:"played_count"`
	InProgressCount int        `gorm:"not null;default:0" json:"in_progress_count"`
	LatestAirDate   *time.Time `json:"latest_air_date,omitempty"`
	LatestAddedAt   *time.Time `json:"latest_added_at,omitempty"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type CatalogSearchDocument struct {
	ItemID             uint      `gorm:"primaryKey" json:"item_id"`
	LibraryID          uint      `gorm:"not null;index;index:idx_catalog_search_documents_library_type_availability_title,priority:1" json:"library_id"`
	ItemType           string    `gorm:"size:32;not null;index;index:idx_catalog_search_documents_library_type_availability_title,priority:2" json:"item_type"`
	Title              string    `gorm:"size:512;not null;index;index:idx_catalog_search_documents_library_type_availability_title,priority:4" json:"title"`
	OriginalTitle      string    `gorm:"size:512" json:"original_title"`
	PeopleText         string    `gorm:"type:text" json:"people_text"`
	TagsText           string    `gorm:"type:text" json:"tags_text"`
	ProviderIDsText    string    `gorm:"type:text" json:"provider_ids_text"`
	Year               *int      `gorm:"index" json:"year,omitempty"`
	OfficialRating     string    `gorm:"size:64;index" json:"official_rating"`
	AvailabilityStatus string    `gorm:"size:64;index;index:idx_catalog_search_documents_library_type_availability_title,priority:3" json:"availability_status"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (CatalogSearchDocument) TableName() string {
	return "catalog_search_documents"
}
