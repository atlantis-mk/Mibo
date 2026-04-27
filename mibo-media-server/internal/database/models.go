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
