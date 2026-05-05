package database

import "time"

type StorageIndexEntry struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	LibraryID         uint       `gorm:"not null;uniqueIndex:idx_storage_index_identity,priority:1;index:idx_storage_index_library_status_path,priority:1;index:idx_storage_index_stable_identity,priority:1" json:"library_id"`
	StorageProvider   string     `gorm:"size:64;not null;uniqueIndex:idx_storage_index_identity,priority:2;index:idx_storage_index_stable_identity,priority:2" json:"storage_provider"`
	StoragePath       string     `gorm:"size:2048;not null;uniqueIndex:idx_storage_index_identity,priority:3;index:idx_storage_index_library_status_path,priority:3" json:"storage_path"`
	IsDir             bool       `gorm:"not null;index" json:"is_dir"`
	SizeBytes         int64      `gorm:"not null;default:0" json:"size_bytes"`
	ModifiedAt        *time.Time `gorm:"index" json:"modified_at,omitempty"`
	StableIdentityKey string     `gorm:"size:512;index:idx_storage_index_stable_identity,priority:3" json:"stable_identity_key"`
	HashesJSON        string     `gorm:"type:text" json:"hashes_json"`
	ProviderName      string     `gorm:"size:128" json:"provider_name"`
	ObjectType        string     `gorm:"size:128" json:"object_type"`
	ProviderMetaJSON  string     `gorm:"type:text" json:"provider_meta_json"`
	ObservationStatus string     `gorm:"size:32;not null;default:present;index:idx_storage_index_library_status_path,priority:2" json:"observation_status"`
	FirstObservedAt   time.Time  `gorm:"not null;index" json:"first_observed_at"`
	LastObservedAt    time.Time  `gorm:"not null;index" json:"last_observed_at"`
	MissingSince      *time.Time `gorm:"index" json:"missing_since,omitempty"`
	LastError         string     `gorm:"type:text" json:"last_error"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (StorageIndexEntry) TableName() string {
	return "storage_index_entries"
}

type StorageObservationFailure struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	LibraryID       uint      `gorm:"not null;index:idx_storage_observation_failure_library_path,priority:1" json:"library_id"`
	StorageProvider string    `gorm:"size:64;not null;index" json:"storage_provider"`
	StoragePath     string    `gorm:"size:2048;not null;index:idx_storage_observation_failure_library_path,priority:2" json:"storage_path"`
	Reason          string    `gorm:"size:128;not null;index" json:"reason"`
	ErrorMessage    string    `gorm:"type:text" json:"error_message"`
	ObservedAt      time.Time `gorm:"not null;index" json:"observed_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (StorageObservationFailure) TableName() string {
	return "storage_observation_failures"
}

type StorageDirectoryFingerprint struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	LibraryID       uint      `gorm:"not null;uniqueIndex:idx_storage_dir_fingerprint_identity,priority:1;index" json:"library_id"`
	StorageProvider string    `gorm:"size:64;not null;uniqueIndex:idx_storage_dir_fingerprint_identity,priority:2" json:"storage_provider"`
	StoragePath     string    `gorm:"size:2048;not null;uniqueIndex:idx_storage_dir_fingerprint_identity,priority:3" json:"storage_path"`
	Fingerprint     string    `gorm:"size:128;not null" json:"fingerprint"`
	ChildCount      int       `gorm:"not null;default:0" json:"child_count"`
	ObservedAt      time.Time `gorm:"not null;index" json:"observed_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (StorageDirectoryFingerprint) TableName() string {
	return "storage_directory_fingerprints"
}
