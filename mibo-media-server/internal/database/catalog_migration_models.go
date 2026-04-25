package database

import "time"

type CatalogMigrationRun struct {
	ID                             uint       `gorm:"primaryKey" json:"id"`
	ScopeKind                      string     `gorm:"size:32;not null;index" json:"scope_kind"`
	LibraryID                      *uint      `gorm:"index" json:"library_id,omitempty"`
	Status                         string     `gorm:"size:32;not null;index" json:"status"`
	TriggeredByUserID              uint       `gorm:"not null;index" json:"triggered_by_user_id"`
	FatalError                     string     `gorm:"type:text" json:"fatal_error"`
	SuccessCount                   int        `gorm:"not null;default:0" json:"success_count"`
	SkippedCount                   int        `gorm:"not null;default:0" json:"skipped_count"`
	ConflictCount                  int        `gorm:"not null;default:0" json:"conflict_count"`
	OrphanFileCount                int        `gorm:"not null;default:0" json:"orphan_file_count"`
	DuplicateEpisodeCandidateCount int        `gorm:"not null;default:0" json:"duplicate_episode_candidate_count"`
	StartedAt                      *time.Time `gorm:"index" json:"started_at,omitempty"`
	FinishedAt                     *time.Time `gorm:"index" json:"finished_at,omitempty"`
	CreatedAt                      time.Time  `json:"created_at"`
	UpdatedAt                      time.Time  `json:"updated_at"`
}

type CatalogMigrationEntry struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	RunID             uint      `gorm:"not null;index:idx_catalog_migration_entries_run_sort,priority:1" json:"run_id"`
	EntryType         string    `gorm:"size:64;not null;index:idx_catalog_migration_entries_run_sort,priority:2;index" json:"entry_type"`
	LibraryID         *uint     `gorm:"index:idx_catalog_migration_entries_run_sort,priority:3;index" json:"library_id,omitempty"`
	LegacyMediaItemID *uint     `gorm:"index:idx_catalog_migration_entries_run_sort,priority:4;index" json:"legacy_media_item_id,omitempty"`
	LegacyMediaFileID *uint     `gorm:"index:idx_catalog_migration_entries_run_sort,priority:5;index" json:"legacy_media_file_id,omitempty"`
	CatalogItemID     *uint     `gorm:"index" json:"catalog_item_id,omitempty"`
	AssetID           *uint     `gorm:"index" json:"asset_id,omitempty"`
	InventoryFileID   *uint     `gorm:"index" json:"inventory_file_id,omitempty"`
	StoragePath       string    `gorm:"size:2048;index" json:"storage_path"`
	Title             string    `gorm:"size:512;index" json:"title"`
	Message           string    `gorm:"type:text" json:"message"`
	DetailsJSON       string    `gorm:"type:text" json:"details_json"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
