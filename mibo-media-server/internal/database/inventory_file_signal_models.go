package database

import "time"

type InventoryFileSignal struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	InventoryFileID    *uint      `gorm:"index" json:"inventory_file_id,omitempty"`
	LibraryID          uint       `gorm:"not null;index;index:idx_inventory_file_signals_library_state,priority:1" json:"library_id"`
	StorageProvider    string     `gorm:"size:64;not null;uniqueIndex:idx_inventory_file_signal_identity,priority:1" json:"storage_provider"`
	StoragePath        string     `gorm:"size:2048;not null;uniqueIndex:idx_inventory_file_signal_identity,priority:2;index" json:"storage_path"`
	ParentPath         string     `gorm:"size:2048;not null;index" json:"parent_path"`
	Basename           string     `gorm:"size:512;not null" json:"basename"`
	Extension          string     `gorm:"size:64;not null;index" json:"extension"`
	ClassifierVersion  string     `gorm:"size:64;not null;uniqueIndex:idx_inventory_file_signal_identity,priority:3;index" json:"classifier_version"`
	FileFingerprint    string     `gorm:"size:128;not null;index" json:"file_fingerprint"`
	TitleCandidate     string     `gorm:"size:512;index" json:"title_candidate"`
	Year               *int       `gorm:"index" json:"year,omitempty"`
	SeasonNumber       *int       `gorm:"index" json:"season_number,omitempty"`
	EpisodeNumber      *int       `gorm:"index" json:"episode_number,omitempty"`
	LeadingNumber      *int       `gorm:"index" json:"leading_number,omitempty"`
	EpisodeSource      string     `gorm:"size:64;index" json:"episode_source"`
	Role               string     `gorm:"size:64;index" json:"role"`
	IsExtra            bool       `gorm:"not null;default:false;index" json:"is_extra"`
	Quality            string     `gorm:"size:128;index" json:"quality"`
	Codec              string     `gorm:"size:128;index" json:"codec"`
	Audio              string     `gorm:"size:128" json:"audio"`
	Subtitle           string     `gorm:"size:128" json:"subtitle"`
	HDR                string     `gorm:"size:128" json:"hdr"`
	Edition            string     `gorm:"size:128" json:"edition"`
	ReleaseGroup       string     `gorm:"size:128" json:"release_group"`
	SourceTagsJSON     string     `gorm:"type:text" json:"source_tags_json"`
	EpisodeNumbersJSON string     `gorm:"type:text" json:"episode_numbers_json"`
	TitleTokensJSON    string     `gorm:"type:text" json:"title_tokens_json"`
	ModelJSON          string     `gorm:"type:text" json:"model_json"`
	EvidenceJSON       string     `gorm:"type:text" json:"evidence_json"`
	InvalidatedAt      *time.Time `gorm:"index" json:"invalidated_at,omitempty"`
	LastObservedAt     time.Time  `gorm:"not null;index" json:"last_observed_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (InventoryFileSignal) TableName() string { return "inventory_file_signals" }
