package database

import "time"

type ContentShapeProfile struct {
	ID                   uint       `gorm:"primaryKey" json:"id"`
	LibraryID            uint       `gorm:"not null;index;uniqueIndex:idx_content_shape_profile_identity,priority:1;index:idx_content_shape_profiles_library_state,priority:1" json:"library_id"`
	MediaSourceID        uint       `gorm:"not null;index" json:"media_source_id"`
	LibraryPathID        *uint      `gorm:"index" json:"library_path_id,omitempty"`
	StorageProvider      string     `gorm:"size:64;not null;uniqueIndex:idx_content_shape_profile_identity,priority:2" json:"storage_provider"`
	RootPath             string     `gorm:"size:2048;not null;uniqueIndex:idx_content_shape_profile_identity,priority:3" json:"root_path"`
	DirectoryPath        string     `gorm:"size:2048;not null;uniqueIndex:idx_content_shape_profile_identity,priority:4;index" json:"directory_path"`
	ClassifierVersion    string     `gorm:"size:64;not null;uniqueIndex:idx_content_shape_profile_identity,priority:5;index" json:"classifier_version"`
	Fingerprint          string     `gorm:"size:128;not null;index" json:"fingerprint"`
	VideoCount           int        `gorm:"not null;default:0" json:"video_count"`
	NonExtraVideoCount   int        `gorm:"not null;default:0" json:"non_extra_video_count"`
	AttachmentCount      int        `gorm:"not null;default:0" json:"attachment_count"`
	ExplicitEpisodeCount int        `gorm:"not null;default:0" json:"explicit_episode_count"`
	LeadingNumericCount  int        `gorm:"not null;default:0" json:"leading_numeric_count"`
	SequenceCoverage     *float64   `json:"sequence_coverage,omitempty"`
	YearDensity          *float64   `json:"year_density,omitempty"`
	TitleUniqueness      *float64   `json:"title_uniqueness,omitempty"`
	CommonTitleStem      string     `gorm:"size:512" json:"common_title_stem"`
	SeasonHint           string     `gorm:"size:128" json:"season_hint"`
	SidecarHintsJSON     string     `gorm:"type:text" json:"sidecar_hints_json"`
	Confidence           *float64   `json:"confidence,omitempty"`
	ReviewState          string     `gorm:"size:64;not null;default:auto;index:idx_content_shape_profiles_library_state,priority:2" json:"review_state"`
	EvidenceJSON         string     `gorm:"type:text" json:"evidence_json"`
	DeletedScope         bool       `gorm:"not null;default:false;index:idx_content_shape_profiles_library_state,priority:3" json:"deleted_scope"`
	InvalidatedAt        *time.Time `gorm:"index" json:"invalidated_at,omitempty"`
	LastObservedAt       time.Time  `gorm:"not null;index" json:"last_observed_at"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

func (ContentShapeProfile) TableName() string { return "content_shape_profiles" }

type ContentShapePlan struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	ProfileID         uint       `gorm:"not null;index" json:"profile_id"`
	LibraryID         uint       `gorm:"not null;index;uniqueIndex:idx_content_shape_plan_scope,priority:1;index:idx_content_shape_plans_library_state,priority:1" json:"library_id"`
	MediaSourceID     uint       `gorm:"not null;index" json:"media_source_id"`
	LibraryPathID     *uint      `gorm:"index" json:"library_path_id,omitempty"`
	StorageProvider   string     `gorm:"size:64;not null;uniqueIndex:idx_content_shape_plan_scope,priority:2" json:"storage_provider"`
	RootPath          string     `gorm:"size:2048;not null;uniqueIndex:idx_content_shape_plan_scope,priority:3" json:"root_path"`
	DirectoryPath     string     `gorm:"size:2048;not null;uniqueIndex:idx_content_shape_plan_scope,priority:4;index" json:"directory_path"`
	ClassifierVersion string     `gorm:"size:64;not null;uniqueIndex:idx_content_shape_plan_scope,priority:5;index" json:"classifier_version"`
	Fingerprint       string     `gorm:"size:128;not null;index" json:"fingerprint"`
	Shape             string     `gorm:"size:64;not null;index" json:"shape"`
	Confidence        *float64   `json:"confidence,omitempty"`
	ReviewState       string     `gorm:"size:64;not null;default:auto;index:idx_content_shape_plans_library_state,priority:2" json:"review_state"`
	SeriesTitle       string     `gorm:"size:512" json:"series_title"`
	SeasonNumber      *int       `gorm:"index" json:"season_number,omitempty"`
	NumberingMode     string     `gorm:"size:64" json:"numbering_mode"`
	PlanRuleJSON      string     `gorm:"type:text" json:"plan_rule_json"`
	ExceptionsJSON    string     `gorm:"type:text" json:"exceptions_json"`
	EvidenceJSON      string     `gorm:"type:text" json:"evidence_json"`
	AlternativesJSON  string     `gorm:"type:text" json:"alternatives_json"`
	DeletedScope      bool       `gorm:"not null;default:false;index:idx_content_shape_plans_library_state,priority:3" json:"deleted_scope"`
	InvalidatedAt     *time.Time `gorm:"index" json:"invalidated_at,omitempty"`
	LastObservedAt    time.Time  `gorm:"not null;index" json:"last_observed_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (ContentShapePlan) TableName() string { return "content_shape_plans" }

type ContentShapeAssignment struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	PlanID            uint       `gorm:"not null;index" json:"plan_id"`
	ProfileID         uint       `gorm:"not null;index" json:"profile_id"`
	LibraryID         uint       `gorm:"not null;index;index:idx_content_shape_assignments_library_state,priority:1" json:"library_id"`
	MediaSourceID     uint       `gorm:"not null;index" json:"media_source_id"`
	LibraryPathID     *uint      `gorm:"index" json:"library_path_id,omitempty"`
	InventoryFileID   *uint      `gorm:"index" json:"inventory_file_id,omitempty"`
	StorageProvider   string     `gorm:"size:64;not null;uniqueIndex:idx_content_shape_assignment_file,priority:1" json:"storage_provider"`
	RootPath          string     `gorm:"size:2048;not null;index" json:"root_path"`
	DirectoryPath     string     `gorm:"size:2048;not null;index" json:"directory_path"`
	StoragePath       string     `gorm:"size:2048;not null;uniqueIndex:idx_content_shape_assignment_file,priority:2" json:"storage_path"`
	ClassifierVersion string     `gorm:"size:64;not null;index" json:"classifier_version"`
	AssignmentType    string     `gorm:"size:64;not null;index" json:"assignment_type"`
	TargetKey         string     `gorm:"size:1024;index" json:"target_key"`
	SeriesTitle       string     `gorm:"size:512" json:"series_title"`
	SeasonNumber      *int       `gorm:"index" json:"season_number,omitempty"`
	EpisodeNumber     *int       `gorm:"index" json:"episode_number,omitempty"`
	AbsoluteNumber    *int       `gorm:"index" json:"absolute_number,omitempty"`
	AssetRole         string     `gorm:"size:64;index" json:"asset_role"`
	Confidence        *float64   `json:"confidence,omitempty"`
	ReviewState       string     `gorm:"size:64;not null;default:auto;index:idx_content_shape_assignments_library_state,priority:2" json:"review_state"`
	EvidenceJSON      string     `gorm:"type:text" json:"evidence_json"`
	AlternativesJSON  string     `gorm:"type:text" json:"alternatives_json"`
	DeletedScope      bool       `gorm:"not null;default:false;index:idx_content_shape_assignments_library_state,priority:3" json:"deleted_scope"`
	InvalidatedAt     *time.Time `gorm:"index" json:"invalidated_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (ContentShapeAssignment) TableName() string { return "content_shape_assignments" }
