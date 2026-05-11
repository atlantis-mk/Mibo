package database

import "time"

type MetadataOperation struct {
	ID                    uint       `gorm:"primaryKey" json:"id"`
	Operation             string     `gorm:"size:64;not null;index" json:"operation"`
	DeduplicationKey      string     `gorm:"size:512;index" json:"deduplication_key"`
	OriginMetadataItemID  uint       `gorm:"index" json:"origin_metadata_item_id"`
	TargetMetadataItemID  uint       `gorm:"index" json:"target_metadata_item_id"`
	LibraryID             uint       `gorm:"not null;index" json:"library_id"`
	Status                string     `gorm:"size:64;not null;index" json:"status"`
	GovernanceStatus      string     `gorm:"size:64;index" json:"governance_status"`
	PlanJSON              string     `gorm:"type:text" json:"plan_json"`
	TriggerContextJSON    string     `gorm:"type:text" json:"trigger_context_json"`
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
