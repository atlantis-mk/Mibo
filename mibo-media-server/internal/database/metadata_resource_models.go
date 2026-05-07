package database

import "time"

type MetadataItem struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	ItemType            string     `gorm:"size:32;not null;index;index:idx_metadata_items_type_status_sort,priority:1;index:idx_metadata_items_root_type_order,priority:2" json:"item_type"`
	ContentForm         string     `gorm:"size:64;not null;default:standard;index" json:"content_form"`
	ParentID            *uint      `gorm:"index;index:idx_metadata_items_parent_order,priority:1" json:"parent_id,omitempty"`
	RootID              *uint      `gorm:"index;index:idx_metadata_items_root_type_order,priority:1" json:"root_id,omitempty"`
	SortKey             string     `gorm:"size:512;index;index:idx_metadata_items_type_status_sort,priority:3" json:"sort_key"`
	DisplayOrder        string     `gorm:"size:32;not null;default:aired" json:"display_order"`
	IndexNumber         *int       `gorm:"index;index:idx_metadata_items_parent_order,priority:3;index:idx_metadata_items_root_type_order,priority:4" json:"index_number,omitempty"`
	IndexNumberEnd      *int       `json:"index_number_end,omitempty"`
	ParentIndexNumber   *int       `gorm:"index;index:idx_metadata_items_parent_order,priority:2;index:idx_metadata_items_root_type_order,priority:3" json:"parent_index_number,omitempty"`
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
	GovernanceStatus    string     `gorm:"size:64;not null;default:pending;index;index:idx_metadata_items_type_status_sort,priority:2" json:"governance_status"`
	CanonicalVersion    int        `gorm:"not null;default:1" json:"canonical_version"`
	LastCanonicalizedAt *time.Time `json:"last_canonicalized_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	DeletedAt           *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

type MetadataExternalID struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	MetadataItemID uint      `gorm:"not null;index" json:"metadata_item_id"`
	Provider       string    `gorm:"size:64;not null;uniqueIndex:idx_metadata_external_identity,priority:1" json:"provider"`
	ProviderType   string    `gorm:"size:64;not null;uniqueIndex:idx_metadata_external_identity,priority:2" json:"provider_type"`
	ExternalID     string    `gorm:"size:255;not null;uniqueIndex:idx_metadata_external_identity,priority:3" json:"external_id"`
	IsPrimary      bool      `gorm:"not null;default:false" json:"is_primary"`
	Source         string    `gorm:"size:64" json:"source"`
	Confidence     *float64  `json:"confidence,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type MetadataItemSource struct {
	ID                   uint       `gorm:"primaryKey" json:"id"`
	MetadataItemID       uint       `gorm:"not null;index" json:"metadata_item_id"`
	SourceType           string     `gorm:"size:64;not null;index" json:"source_type"`
	SourceName           string     `gorm:"size:128;not null;index" json:"source_name"`
	Language             string     `gorm:"size:32;index" json:"language"`
	ExternalID           string     `gorm:"size:255;index" json:"external_id"`
	TriggeringLibraryID  *uint      `gorm:"index" json:"triggering_library_id,omitempty"`
	MetadataProfileID    *uint      `gorm:"index" json:"metadata_profile_id,omitempty"`
	MetadataProfileName  string     `gorm:"size:255;index" json:"metadata_profile_name"`
	ProviderInstanceID   *uint      `gorm:"index" json:"provider_instance_id,omitempty"`
	ProviderInstanceName string     `gorm:"size:255;index" json:"provider_instance_name"`
	FallbackSummaryJSON  string     `gorm:"type:text" json:"fallback_summary_json"`
	PayloadJSON          string     `gorm:"type:text" json:"payload_json"`
	EvidenceJSON         string     `gorm:"type:text" json:"evidence_json"`
	Confidence           *float64   `json:"confidence,omitempty"`
	FetchedAt            time.Time  `gorm:"not null;index" json:"fetched_at"`
	ExpiresAt            *time.Time `gorm:"index" json:"expires_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type MetadataItemFieldState struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	MetadataItemID uint       `gorm:"not null;uniqueIndex:idx_metadata_item_field_state_identity,priority:1" json:"metadata_item_id"`
	FieldKey       string     `gorm:"size:128;not null;uniqueIndex:idx_metadata_item_field_state_identity,priority:2" json:"field_key"`
	Locale         string     `gorm:"size:32;not null;default:'';uniqueIndex:idx_metadata_item_field_state_identity,priority:3" json:"locale"`
	SourceID       *uint      `gorm:"index" json:"source_id,omitempty"`
	ValueJSON      string     `gorm:"type:text;not null" json:"value_json"`
	IsLocked       bool       `gorm:"not null;default:false;index" json:"is_locked"`
	LockReason     string     `gorm:"size:255" json:"lock_reason"`
	EditedByUserID *uint      `gorm:"index" json:"edited_by_user_id,omitempty"`
	EditedAt       *time.Time `json:"edited_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type MetadataItemImage struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	MetadataItemID uint      `gorm:"not null;index;index:idx_metadata_item_images_selected,priority:1" json:"metadata_item_id"`
	ImageType      string    `gorm:"size:64;not null;index;index:idx_metadata_item_images_selected,priority:2" json:"image_type"`
	URL            string    `gorm:"size:2048;not null" json:"url"`
	SourceID       *uint     `gorm:"index" json:"source_id,omitempty"`
	Language       string    `gorm:"size:32;index" json:"language"`
	Width          *int      `json:"width,omitempty"`
	Height         *int      `json:"height,omitempty"`
	IsSelected     bool      `gorm:"not null;default:false;index;index:idx_metadata_item_images_selected,priority:3" json:"is_selected"`
	SortOrder      int       `gorm:"not null;default:0;index" json:"sort_order"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type MetadataItemPerson struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	MetadataItemID uint      `gorm:"not null;uniqueIndex:idx_metadata_item_people_identity,priority:1" json:"metadata_item_id"`
	PersonID       uint      `gorm:"not null;uniqueIndex:idx_metadata_item_people_identity,priority:2" json:"person_id"`
	Role           string    `gorm:"size:128;not null;uniqueIndex:idx_metadata_item_people_identity,priority:3" json:"role"`
	Character      string    `gorm:"size:255" json:"character"`
	SortOrder      int       `gorm:"not null;default:0;index" json:"sort_order"`
	SourceID       *uint     `gorm:"index" json:"source_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type MetadataItemTag struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	MetadataItemID uint      `gorm:"not null;uniqueIndex:idx_metadata_item_tags_identity,priority:1" json:"metadata_item_id"`
	TagID          uint      `gorm:"not null;uniqueIndex:idx_metadata_item_tags_identity,priority:2" json:"tag_id"`
	SourceID       *uint     `gorm:"index" json:"source_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Resource struct {
	ID                   uint       `gorm:"primaryKey" json:"id"`
	ResourceType         string     `gorm:"size:64;not null;default:playable;index" json:"resource_type"`
	ResourceShape        string     `gorm:"size:64;not null;default:single_file;index" json:"resource_shape"`
	StableResourceKey    string     `gorm:"size:512;not null;uniqueIndex" json:"stable_resource_key"`
	DisplayName          string     `gorm:"size:512" json:"display_name"`
	Edition              string     `gorm:"size:128" json:"edition"`
	QualityLabel         string     `gorm:"size:128;index" json:"quality_label"`
	DurationSeconds      *float64   `json:"duration_seconds,omitempty"`
	Status               string     `gorm:"size:64;not null;default:available;index" json:"status"`
	MissingSince         *time.Time `gorm:"index" json:"missing_since,omitempty"`
	ProbeStatus          string     `gorm:"size:64;not null;default:pending;index" json:"probe_status"`
	TechnicalSummaryJSON string     `gorm:"type:text" json:"technical_summary_json"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	DeletedAt            *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

type ResourceFile struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ResourceID      uint      `gorm:"not null;uniqueIndex:idx_resource_files_resource_file_role_part,priority:1;index:idx_resource_files_resource_part,priority:1" json:"resource_id"`
	InventoryFileID uint      `gorm:"not null;uniqueIndex:idx_resource_files_resource_file_role_part,priority:2;index" json:"inventory_file_id"`
	Role            string    `gorm:"size:64;not null;uniqueIndex:idx_resource_files_resource_file_role_part,priority:3" json:"role"`
	PartIndex       int       `gorm:"not null;default:0;uniqueIndex:idx_resource_files_resource_file_role_part,priority:4;index:idx_resource_files_resource_part,priority:2" json:"part_index"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ResourceLibraryLink struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	ResourceID   uint       `gorm:"not null;uniqueIndex:idx_resource_library_link_identity,priority:1;index" json:"resource_id"`
	LibraryID    uint       `gorm:"not null;uniqueIndex:idx_resource_library_link_identity,priority:2;index:idx_resource_library_links_library_status,priority:1" json:"library_id"`
	Status       string     `gorm:"size:64;not null;default:available;index:idx_resource_library_links_library_status,priority:2" json:"status"`
	FirstSeenAt  time.Time  `gorm:"not null;index" json:"first_seen_at"`
	LastSeenAt   time.Time  `gorm:"not null;index" json:"last_seen_at"`
	MissingSince *time.Time `gorm:"index" json:"missing_since,omitempty"`
	EvidenceJSON string     `gorm:"type:text" json:"evidence_json"`
	ReviewState  string     `gorm:"size:64;not null;default:accepted;index" json:"review_state"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

type ResourceMetadataLink struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ResourceID     uint      `gorm:"not null;uniqueIndex:idx_resource_metadata_link_identity,priority:1;index" json:"resource_id"`
	MetadataItemID uint      `gorm:"not null;uniqueIndex:idx_resource_metadata_link_identity,priority:2;index;index:idx_resource_metadata_links_item_role,priority:1" json:"metadata_item_id"`
	Role           string    `gorm:"size:64;not null;uniqueIndex:idx_resource_metadata_link_identity,priority:3;index:idx_resource_metadata_links_item_role,priority:2" json:"role"`
	SegmentIndex   int       `gorm:"not null;default:0;uniqueIndex:idx_resource_metadata_link_identity,priority:4" json:"segment_index"`
	StartSeconds   *float64  `json:"start_seconds,omitempty"`
	EndSeconds     *float64  `json:"end_seconds,omitempty"`
	Confidence     *float64  `json:"confidence,omitempty"`
	EvidenceJSON   string    `gorm:"type:text" json:"evidence_json"`
	Source         string    `gorm:"size:64" json:"source"`
	ReviewState    string    `gorm:"size:64;not null;default:accepted;index" json:"review_state"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type LibraryMetadataProjection struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	LibraryID          uint       `gorm:"not null;uniqueIndex:idx_library_metadata_projection_identity,priority:1;index:idx_library_metadata_projections_library_type_availability_title,priority:1" json:"library_id"`
	MetadataItemID     uint       `gorm:"not null;uniqueIndex:idx_library_metadata_projection_identity,priority:2;index" json:"metadata_item_id"`
	ItemType           string     `gorm:"size:32;not null;index:idx_library_metadata_projections_library_type_availability_title,priority:2" json:"item_type"`
	ParentID           *uint      `gorm:"index" json:"parent_id,omitempty"`
	RootID             *uint      `gorm:"index" json:"root_id,omitempty"`
	Title              string     `gorm:"size:512;not null;index:idx_library_metadata_projections_library_type_availability_title,priority:4" json:"title"`
	SortTitle          string     `gorm:"size:512;index" json:"sort_title"`
	Year               *int       `gorm:"index" json:"year,omitempty"`
	AvailabilityStatus string     `gorm:"size:64;not null;default:unavailable;index:idx_library_metadata_projections_library_type_availability_title,priority:3" json:"availability_status"`
	Hidden             bool       `gorm:"not null;default:false;index" json:"hidden"`
	ResourceCount      int        `gorm:"not null;default:0" json:"resource_count"`
	AvailableCount     int        `gorm:"not null;default:0" json:"available_count"`
	MissingCount       int        `gorm:"not null;default:0" json:"missing_count"`
	ChildCount         int        `gorm:"not null;default:0" json:"child_count"`
	LatestAddedAt      *time.Time `gorm:"index" json:"latest_added_at,omitempty"`
	LastProjectedAt    time.Time  `gorm:"not null;index" json:"last_projected_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type MetadataSearchDocument struct {
	MetadataItemID  uint      `gorm:"primaryKey" json:"metadata_item_id"`
	ItemType        string    `gorm:"size:32;not null;index" json:"item_type"`
	ContentForm     string    `gorm:"size:64;not null;index" json:"content_form"`
	Title           string    `gorm:"size:512;not null;index" json:"title"`
	OriginalTitle   string    `gorm:"size:512" json:"original_title"`
	PeopleText      string    `gorm:"type:text" json:"people_text"`
	TagsText        string    `gorm:"type:text" json:"tags_text"`
	ProviderIDsText string    `gorm:"type:text" json:"provider_ids_text"`
	Year            *int      `gorm:"index" json:"year,omitempty"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type LibrarySearchDocument struct {
	LibraryID          uint      `gorm:"not null;primaryKey;index:idx_library_search_documents_library_type_availability_title,priority:1" json:"library_id"`
	MetadataItemID     uint      `gorm:"not null;primaryKey" json:"metadata_item_id"`
	ItemType           string    `gorm:"size:32;not null;index:idx_library_search_documents_library_type_availability_title,priority:2" json:"item_type"`
	Title              string    `gorm:"size:512;not null;index:idx_library_search_documents_library_type_availability_title,priority:4" json:"title"`
	OriginalTitle      string    `gorm:"size:512" json:"original_title"`
	PeopleText         string    `gorm:"type:text" json:"people_text"`
	TagsText           string    `gorm:"type:text" json:"tags_text"`
	ProviderIDsText    string    `gorm:"type:text" json:"provider_ids_text"`
	ResourceText       string    `gorm:"type:text" json:"resource_text"`
	Year               *int      `gorm:"index" json:"year,omitempty"`
	AvailabilityStatus string    `gorm:"size:64;not null;index:idx_library_search_documents_library_type_availability_title,priority:3" json:"availability_status"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type UserMetadataData struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	UserID              uint       `gorm:"not null;uniqueIndex:idx_user_metadata_data_identity,priority:1" json:"user_id"`
	MetadataItemID      uint       `gorm:"not null;uniqueIndex:idx_user_metadata_data_identity,priority:2;index" json:"metadata_item_id"`
	PositionSeconds     int        `gorm:"not null;default:0" json:"position_seconds"`
	PlayedPercentage    *float64   `json:"played_percentage,omitempty"`
	ProgressFrameURL    string     `gorm:"size:1024" json:"progress_frame_url,omitempty"`
	PlayCount           int        `gorm:"not null;default:0" json:"play_count"`
	Favorite            bool       `gorm:"not null;default:false;index" json:"favorite"`
	PreferredResourceID *uint      `gorm:"index" json:"preferred_resource_id,omitempty"`
	LastPlayedAt        *time.Time `gorm:"index" json:"last_played_at,omitempty"`
	CompletedAt         *time.Time `gorm:"index" json:"completed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type UserResourceData struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	UserID           uint       `gorm:"not null;uniqueIndex:idx_user_resource_data_identity,priority:1" json:"user_id"`
	ResourceID       uint       `gorm:"not null;uniqueIndex:idx_user_resource_data_identity,priority:2;index" json:"resource_id"`
	MetadataItemID   uint       `gorm:"not null;uniqueIndex:idx_user_resource_data_identity,priority:3;index" json:"metadata_item_id"`
	PositionSeconds  int        `gorm:"not null;default:0" json:"position_seconds"`
	PlayedPercentage *float64   `json:"played_percentage,omitempty"`
	ProgressFrameURL string     `gorm:"size:1024" json:"progress_frame_url,omitempty"`
	PlayCount        int        `gorm:"not null;default:0" json:"play_count"`
	Preferred        bool       `gorm:"not null;default:false;index" json:"preferred"`
	LastPlayedAt     *time.Time `gorm:"index" json:"last_played_at,omitempty"`
	CompletedAt      *time.Time `gorm:"index" json:"completed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
