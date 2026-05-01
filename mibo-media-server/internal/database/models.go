package database

import "time"

type MediaSource struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Name             string    `gorm:"size:255;not null" json:"name"`
	Provider         string    `gorm:"size:64;not null;index" json:"provider"`
	StorageRef       string    `gorm:"size:512;not null" json:"storage_ref"`
	RootPath         string    `gorm:"size:1024;not null" json:"root_path"`
	ConfigJSON       string    `gorm:"type:text" json:"-"`
	CapabilitiesJSON string    `gorm:"type:text" json:"capabilities_json"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Library struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Name           string    `gorm:"size:255;not null" json:"name"`
	Type           string    `gorm:"size:64;not null;index" json:"type"`
	MediaSourceID  uint      `gorm:"not null;index" json:"media_source_id"`
	RootPath       string    `gorm:"size:1024;not null" json:"root_path"`
	Status         string    `gorm:"size:64;not null;default:active" json:"status"`
	ScannerEnabled bool      `gorm:"not null;default:true" json:"scanner_enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type LibraryPath struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	LibraryID     uint       `gorm:"not null;index;uniqueIndex:idx_library_paths_library_source_path,priority:1" json:"library_id"`
	MediaSourceID uint       `gorm:"not null;index;uniqueIndex:idx_library_paths_library_source_path,priority:2" json:"media_source_id"`
	RootPath      string     `gorm:"size:1024;not null;uniqueIndex:idx_library_paths_library_source_path,priority:3" json:"root_path"`
	DisplayName   string     `gorm:"size:255" json:"display_name"`
	Enabled       bool       `gorm:"not null;default:true;index" json:"enabled"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

func (LibraryPath) TableName() string {
	return "library_paths"
}

type LibraryScanPolicy struct {
	ID                         uint      `gorm:"primaryKey" json:"id"`
	LibraryID                  uint      `gorm:"not null;uniqueIndex" json:"library_id"`
	ScannerEnabled             bool      `gorm:"not null;default:true" json:"scanner_enabled"`
	RealtimeMonitorEnabled     bool      `gorm:"not null;default:true" json:"realtime_monitor_enabled"`
	ScheduledRefreshEnabled    bool      `gorm:"not null;default:true" json:"scheduled_refresh_enabled"`
	RefreshIntervalHours       int       `gorm:"not null;default:24" json:"refresh_interval_hours"`
	IgnoreHiddenFiles          bool      `gorm:"not null;default:true" json:"ignore_hidden_files"`
	IgnoreFileExtensionsJSON   string    `gorm:"type:text" json:"ignore_file_extensions_json"`
	MinFileSizeBytes           int64     `gorm:"not null;default:0" json:"min_file_size_bytes"`
	SampleIgnoreSizeBytes      int64     `gorm:"not null;default:0" json:"sample_ignore_size_bytes"`
	ConfigurableExclusionRules bool      `gorm:"not null;default:true" json:"configurable_exclusion_rules"`
	CreatedAt                  time.Time `json:"created_at"`
	UpdatedAt                  time.Time `json:"updated_at"`
}

type LibraryMetadataPolicy struct {
	ID                        uint      `gorm:"primaryKey" json:"id"`
	LibraryID                 uint      `gorm:"not null;uniqueIndex" json:"library_id"`
	PreferredMetadataLanguage string    `gorm:"size:32" json:"preferred_metadata_language"`
	PreferredImageLanguage    string    `gorm:"size:32" json:"preferred_image_language"`
	MetadataCountryCode       string    `gorm:"size:16" json:"metadata_country_code"`
	LocalMetadataEnabled      bool      `gorm:"not null;default:true" json:"local_metadata_enabled"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

type LibraryPlaybackPolicy struct {
	ID                       uint      `gorm:"primaryKey" json:"id"`
	LibraryID                uint      `gorm:"not null;uniqueIndex" json:"library_id"`
	ResumeEnabled            bool      `gorm:"not null;default:true" json:"resume_enabled"`
	MinResumePct             int       `gorm:"not null;default:5" json:"min_resume_pct"`
	MaxResumePct             int       `gorm:"not null;default:90" json:"max_resume_pct"`
	MinResumeDurationSeconds int       `gorm:"not null;default:300" json:"min_resume_duration_seconds"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type LibrarySubtitlePolicy struct {
	ID                             uint      `gorm:"primaryKey" json:"id"`
	LibraryID                      uint      `gorm:"not null;uniqueIndex" json:"library_id"`
	ExternalSidecarsEnabled        bool      `gorm:"not null;default:true" json:"external_sidecars_enabled"`
	PreferredLanguagesJSON         string    `gorm:"type:text" json:"preferred_languages_json"`
	RequirePerfectMatch            bool      `gorm:"not null;default:false" json:"require_perfect_match"`
	SaveWithMedia                  bool      `gorm:"not null;default:false" json:"save_with_media"`
	TolerateUnavailableSubtitles   bool      `gorm:"not null;default:true" json:"tolerate_unavailable_subtitles"`
	SkipIfEmbeddedSubtitlesPresent bool      `gorm:"not null;default:false" json:"skip_if_embedded_subtitles_present"`
	SkipIfAudioTrackMatches        bool      `gorm:"not null;default:false" json:"skip_if_audio_track_matches"`
	CreatedAt                      time.Time `json:"created_at"`
	UpdatedAt                      time.Time `json:"updated_at"`
}

type Job struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	JobKey       string     `gorm:"size:255;index" json:"job_key"`
	Kind         string     `gorm:"size:128;not null;index" json:"kind"`
	Status       string     `gorm:"size:64;not null;index" json:"status"`
	PayloadJSON  string     `gorm:"type:text" json:"payload_json"`
	ErrorMessage string     `gorm:"type:text" json:"error_message"`
	Attempts     int        `gorm:"not null;default:0" json:"attempts"`
	AvailableAt  time.Time  `gorm:"not null;index" json:"available_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type JobActiveIntent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	IntentKey string    `gorm:"size:255;not null;uniqueIndex" json:"intent_key"`
	Kind      string    `gorm:"size:128;not null;index" json:"kind"`
	JobID     uint      `gorm:"not null;index" json:"job_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Schedule struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	Name                string     `gorm:"size:255;not null" json:"name"`
	Kind                string     `gorm:"size:64;not null;index" json:"kind"`
	ScopeKind           string     `gorm:"size:32;not null;index" json:"scope_kind"`
	LibraryID           *uint      `gorm:"index" json:"library_id,omitempty"`
	FrequencyKind       string     `gorm:"size:32;not null" json:"frequency_kind"`
	TimeOfDay           string     `gorm:"size:5;not null" json:"time_of_day"`
	Weekday             *int       `json:"weekday,omitempty"`
	DayOfMonth          *int       `json:"day_of_month,omitempty"`
	Enabled             bool       `gorm:"not null;default:true;index" json:"enabled"`
	NextRunAt           *time.Time `gorm:"index" json:"next_run_at,omitempty"`
	LatestRunStatus     string     `gorm:"size:64" json:"latest_run_status"`
	LatestRunMessage    string     `gorm:"type:text" json:"latest_run_message"`
	LatestJobID         *uint      `gorm:"index" json:"latest_job_id,omitempty"`
	LatestRunStartedAt  *time.Time `json:"latest_run_started_at,omitempty"`
	LatestRunFinishedAt *time.Time `json:"latest_run_finished_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	DeletedAt           *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

type ScheduleRun struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	ScheduleID   uint       `gorm:"not null;index" json:"schedule_id"`
	Status       string     `gorm:"size:64;not null;index" json:"status"`
	JobID        *uint      `gorm:"index" json:"job_id,omitempty"`
	ErrorSummary string     `gorm:"type:text" json:"error_summary"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:128;not null;uniqueIndex" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Role         string    `gorm:"size:64;not null;default:user" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Session struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"not null;index" json:"user_id"`
	TokenHash  string     `gorm:"size:128;not null;uniqueIndex" json:"-"`
	UserAgent  *string    `gorm:"type:text" json:"user_agent,omitempty"`
	RemoteAddr *string    `gorm:"size:255" json:"remote_addr,omitempty"`
	DeviceName *string    `gorm:"size:255" json:"device_name,omitempty"`
	ClientType *string    `gorm:"size:128" json:"client_type,omitempty"`
	ExpiresAt  time.Time  `gorm:"not null;index" json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type SystemSetting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Category  string    `gorm:"size:64;not null;uniqueIndex:idx_system_setting_category_key" json:"category"`
	Key       string    `gorm:"size:128;not null;uniqueIndex:idx_system_setting_category_key" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	IsSecret  bool      `gorm:"not null;default:false" json:"is_secret"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SearchHistory struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       uint       `gorm:"not null;index:idx_search_history_user_used_at" json:"user_id"`
	Query        string     `gorm:"size:512;not null" json:"query"`
	TypeFilter   string     `gorm:"size:32" json:"type_filter"`
	Genre        string     `gorm:"size:128" json:"genre"`
	Region       string     `gorm:"size:128" json:"region"`
	Year         *int       `json:"year,omitempty"`
	MinRating    *float64   `json:"min_rating,omitempty"`
	WatchedState string     `gorm:"size:32" json:"watched_state"`
	Sort         string     `gorm:"size:32" json:"sort"`
	LastUsedAt   time.Time  `gorm:"not null;index:idx_search_history_user_used_at" json:"last_used_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}
