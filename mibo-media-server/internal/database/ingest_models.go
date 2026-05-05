package database

import "time"

type IngestDirtyUnit struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	DirtyKey        string     `gorm:"size:512;not null;uniqueIndex" json:"dirty_key"`
	ScopeKind       string     `gorm:"size:64;not null;index:idx_ingest_dirty_units_claim,priority:2;index:idx_ingest_dirty_units_scope,priority:1" json:"scope_kind"`
	LibraryID       uint       `gorm:"not null;index:idx_ingest_dirty_units_library,priority:1;index:idx_ingest_dirty_units_scope,priority:2" json:"library_id"`
	InventoryFileID *uint      `gorm:"index" json:"inventory_file_id,omitempty"`
	CatalogItemID   *uint      `gorm:"index" json:"catalog_item_id,omitempty"`
	RootPath        string     `gorm:"size:2048;index:idx_ingest_dirty_units_scope,priority:3" json:"root_path,omitempty"`
	Reason          string     `gorm:"size:128;not null" json:"reason"`
	Status          string     `gorm:"size:64;not null;default:dirty;index:idx_ingest_dirty_units_claim,priority:1" json:"status"`
	Attempts        int        `gorm:"not null;default:0" json:"attempts"`
	AvailableAt     time.Time  `gorm:"not null;index:idx_ingest_dirty_units_claim,priority:3" json:"available_at"`
	ClaimedAt       *time.Time `gorm:"index" json:"claimed_at,omitempty"`
	LastError       string     `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type IngestCondition struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	UnitKey             string     `gorm:"size:512;not null;uniqueIndex:idx_ingest_conditions_unit_type;index:idx_ingest_conditions_unit_status,priority:1" json:"unit_key"`
	LibraryID           uint       `gorm:"not null;index:idx_ingest_conditions_library_status,priority:1" json:"library_id"`
	InventoryFileID     *uint      `gorm:"index" json:"inventory_file_id,omitempty"`
	CatalogItemID       *uint      `gorm:"index" json:"catalog_item_id,omitempty"`
	ConditionType       string     `gorm:"size:64;not null;uniqueIndex:idx_ingest_conditions_unit_type;index:idx_ingest_conditions_type_status,priority:1" json:"condition_type"`
	Status              string     `gorm:"size:64;not null;index:idx_ingest_conditions_type_status,priority:2;index:idx_ingest_conditions_library_status,priority:2;index:idx_ingest_conditions_unit_status,priority:2" json:"status"`
	Reason              string     `gorm:"size:128;index" json:"reason,omitempty"`
	Message             string     `gorm:"size:1024" json:"message,omitempty"`
	Severity            string     `gorm:"size:64;index" json:"severity,omitempty"`
	Attempts            int        `gorm:"not null;default:0" json:"attempts"`
	JobID               *uint      `gorm:"index" json:"job_id,omitempty"`
	MetadataOperationID *uint      `gorm:"index" json:"metadata_operation_id,omitempty"`
	ProviderInstanceID  *uint      `gorm:"index" json:"provider_instance_id,omitempty"`
	DetailsJSON         string     `gorm:"type:text" json:"details_json,omitempty"`
	LastTransitionAt    *time.Time `gorm:"index" json:"last_transition_at,omitempty"`
	NextAttemptAt       *time.Time `gorm:"index:idx_ingest_conditions_retry_due" json:"next_attempt_at,omitempty"`
	StaleAfter          *time.Time `gorm:"index" json:"stale_after,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type IngestEvent struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	UnitKey             string     `gorm:"size:512;not null;index:idx_ingest_events_unit_created,priority:1" json:"unit_key"`
	LibraryID           uint       `gorm:"not null;index:idx_ingest_events_library_created,priority:1" json:"library_id"`
	InventoryFileID     *uint      `gorm:"index" json:"inventory_file_id,omitempty"`
	CatalogItemID       *uint      `gorm:"index" json:"catalog_item_id,omitempty"`
	ConditionID         *uint      `gorm:"index:idx_ingest_events_condition_created,priority:1" json:"condition_id,omitempty"`
	ConditionType       string     `gorm:"size:64;index" json:"condition_type,omitempty"`
	EventType           string     `gorm:"size:64;not null;index" json:"event_type"`
	Status              string     `gorm:"size:64;index" json:"status,omitempty"`
	Reason              string     `gorm:"size:128;index" json:"reason,omitempty"`
	Message             string     `gorm:"size:1024" json:"message,omitempty"`
	JobID               *uint      `gorm:"index" json:"job_id,omitempty"`
	MetadataOperationID *uint      `gorm:"index" json:"metadata_operation_id,omitempty"`
	ProviderInstanceID  *uint      `gorm:"index" json:"provider_instance_id,omitempty"`
	UserID              *uint      `gorm:"index" json:"user_id,omitempty"`
	DetailsJSON         string     `gorm:"type:text" json:"details_json,omitempty"`
	ExpiresAt           *time.Time `gorm:"index" json:"expires_at,omitempty"`
	CreatedAt           time.Time  `gorm:"index:idx_ingest_events_unit_created,priority:2;index:idx_ingest_events_library_created,priority:2;index:idx_ingest_events_condition_created,priority:2" json:"created_at"`
}
