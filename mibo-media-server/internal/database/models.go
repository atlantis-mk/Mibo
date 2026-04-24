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

type MediaItem struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	LibraryID          uint       `gorm:"not null;index" json:"library_id"`
	Type               string     `gorm:"size:64;not null;index" json:"type"`
	Title              string     `gorm:"size:512;not null;index" json:"title"`
	OriginalTitle      string     `gorm:"size:512" json:"original_title"`
	SeriesTitle        string     `gorm:"size:512;index" json:"series_title"`
	Overview           string     `gorm:"type:text" json:"overview"`
	PosterURL          string     `gorm:"size:2048" json:"poster_url"`
	LogoURL            string     `gorm:"size:2048" json:"logo_url"`
	BackdropURL        string     `gorm:"size:2048" json:"backdrop_url"`
	GenresJSON         string     `gorm:"type:text" json:"genres_json"`
	CastJSON           string     `gorm:"type:text" json:"cast_json"`
	DirectorsJSON      string     `gorm:"type:text" json:"directors_json"`
	Year               *int       `json:"year,omitempty"`
	ReleaseDate        string     `gorm:"size:32" json:"release_date"`
	RuntimeSeconds     *int       `json:"runtime_seconds,omitempty"`
	SeasonNumber       *int       `json:"season_number,omitempty"`
	EpisodeNumber      *int       `json:"episode_number,omitempty"`
	SourcePath         string     `gorm:"size:1024;not null" json:"source_path"`
	MatchStatus        string     `gorm:"size:64;not null;default:pending;index" json:"match_status"`
	MetadataProvider   string     `gorm:"size:64" json:"metadata_provider"`
	ExternalID         string     `gorm:"size:128" json:"external_id"`
	MetadataConfidence *float64   `json:"metadata_confidence,omitempty"`
	Status             string     `gorm:"size:64;not null;default:pending" json:"status"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DeletedAt          *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

type MediaFile struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	LibraryID          uint       `gorm:"not null;index" json:"library_id"`
	MediaItemID        *uint      `gorm:"index" json:"media_item_id,omitempty"`
	StoragePath        string     `gorm:"size:1024;not null" json:"storage_path"`
	StableIdentityKey  string     `gorm:"size:512;index:idx_media_file_stable_identity" json:"stable_identity_key"`
	IdentitySource     string     `gorm:"size:64;not null;default:none;index" json:"identity_source"`
	IdentityStatus     string     `gorm:"size:64;not null;default:provisional;index" json:"identity_status"`
	ProviderName       string     `gorm:"size:255" json:"provider_name"`
	ProviderHashesJSON string     `gorm:"type:text" json:"provider_hashes_json"`
	ReplacedByID       *uint      `gorm:"index" json:"replaced_by_id,omitempty"`
	ReviewStatus       string     `gorm:"size:64;not null;default:none;index" json:"review_status"`
	ReviewReason       string     `gorm:"size:255" json:"review_reason"`
	Container          string     `gorm:"size:64" json:"container"`
	SizeBytes          int64      `gorm:"not null;default:0" json:"size_bytes"`
	Fingerprint        string     `gorm:"size:255" json:"fingerprint"`
	ProbeStatus        string     `gorm:"size:64;not null;default:pending;index" json:"probe_status"`
	ProbeError         string     `gorm:"type:text" json:"probe_error"`
	DurationSeconds    *float64   `json:"duration_seconds,omitempty"`
	BitRate            *int64     `json:"bit_rate,omitempty"`
	Width              *int       `json:"width,omitempty"`
	Height             *int       `json:"height,omitempty"`
	VideoCodec         string     `gorm:"size:128" json:"video_codec"`
	AudioTracksJSON    string     `gorm:"type:text" json:"audio_tracks_json"`
	SubtitleTracksJSON string     `gorm:"type:text" json:"subtitle_tracks_json"`
	LastModifiedAt     *time.Time `json:"last_modified_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DeletedAt          *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

type TVSeasonMetadataCache struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	SeriesTMDBID   int       `gorm:"not null;uniqueIndex:idx_tv_season_cache_lookup" json:"series_tmdb_id"`
	SeasonNumber   int       `gorm:"not null;uniqueIndex:idx_tv_season_cache_lookup" json:"season_number"`
	Language       string    `gorm:"size:32;not null;uniqueIndex:idx_tv_season_cache_lookup" json:"language"`
	Name           string    `gorm:"size:512" json:"name"`
	Overview       string    `gorm:"type:text" json:"overview"`
	PosterPath     string    `gorm:"size:2048" json:"poster_path"`
	RuntimeSeconds *int      `json:"runtime_seconds,omitempty"`
	PayloadJSON    string    `gorm:"type:text" json:"-"`
	FetchedAt      time.Time `gorm:"not null;index" json:"fetched_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type TVEpisodeMetadataCache struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	SeriesTMDBID   int       `gorm:"not null;uniqueIndex:idx_tv_episode_cache_lookup" json:"series_tmdb_id"`
	SeasonNumber   int       `gorm:"not null;uniqueIndex:idx_tv_episode_cache_lookup" json:"season_number"`
	EpisodeNumber  int       `gorm:"not null;uniqueIndex:idx_tv_episode_cache_lookup" json:"episode_number"`
	Language       string    `gorm:"size:32;not null;uniqueIndex:idx_tv_episode_cache_lookup" json:"language"`
	Name           string    `gorm:"size:512" json:"name"`
	Overview       string    `gorm:"type:text" json:"overview"`
	StillPath      string    `gorm:"size:2048" json:"still_path"`
	RuntimeSeconds *int      `json:"runtime_seconds,omitempty"`
	PayloadJSON    string    `gorm:"type:text" json:"-"`
	FetchedAt      time.Time `gorm:"not null;index" json:"fetched_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
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
	ExpiresAt  time.Time  `gorm:"not null;index" json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type PlaybackProgress struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	UserID          uint       `gorm:"not null;uniqueIndex:idx_user_media_item" json:"user_id"`
	MediaItemID     uint       `gorm:"not null;uniqueIndex:idx_user_media_item;index" json:"media_item_id"`
	MediaFileID     *uint      `gorm:"index" json:"media_file_id,omitempty"`
	PositionSeconds int        `gorm:"not null;default:0" json:"position_seconds"`
	DurationSeconds *int       `json:"duration_seconds,omitempty"`
	Watched         bool       `gorm:"not null;default:false;index" json:"watched"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	LastPlayedAt    *time.Time `gorm:"index" json:"last_played_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
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
