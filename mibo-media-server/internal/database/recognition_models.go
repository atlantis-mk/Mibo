package database

import "time"

type RecognitionManifest struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	ManifestKey       string     `gorm:"size:512;not null;uniqueIndex:idx_recognition_manifest_key;index" json:"manifest_key"`
	LibraryID         uint       `gorm:"not null;index:idx_recognition_manifests_library_state,priority:1" json:"library_id"`
	MediaSourceID     uint       `gorm:"not null;default:0;index" json:"media_source_id"`
	LibraryPathID     *uint      `gorm:"index" json:"library_path_id,omitempty"`
	StorageProvider   string     `gorm:"size:64;not null;index:idx_recognition_manifest_scope,priority:1" json:"storage_provider"`
	RootPath          string     `gorm:"size:2048;not null;index:idx_recognition_manifest_scope,priority:2" json:"root_path"`
	ScopePath         string     `gorm:"size:2048;not null;index:idx_recognition_manifest_scope,priority:3" json:"scope_path"`
	ClassifierVersion string     `gorm:"size:64;not null;index" json:"classifier_version"`
	Fingerprint       string     `gorm:"size:128;not null;index" json:"fingerprint"`
	Status            string     `gorm:"size:64;not null;default:pending;index:idx_recognition_manifests_library_state,priority:2" json:"status"`
	EvidenceJSON      string     `gorm:"type:text" json:"evidence_json"`
	ObservedAt        time.Time  `gorm:"not null;index" json:"observed_at"`
	ResolvedAt        *time.Time `gorm:"index" json:"resolved_at,omitempty"`
	SupersededAt      *time.Time `gorm:"index" json:"superseded_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (RecognitionManifest) TableName() string { return "recognition_manifests" }

type RecognitionCandidate struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	ManifestID         uint       `gorm:"not null;index;uniqueIndex:idx_recognition_candidate_identity,priority:1" json:"manifest_id"`
	CandidateKey       string     `gorm:"size:512;not null;uniqueIndex:idx_recognition_candidate_identity,priority:2;index" json:"candidate_key"`
	CandidateType      string     `gorm:"size:64;not null;index:idx_recognition_candidates_type_state,priority:1" json:"candidate_type"`
	CandidateRole      string     `gorm:"size:64;index" json:"candidate_role"`
	ParentCandidateKey string     `gorm:"size:512;index" json:"parent_candidate_key"`
	TargetMetadataID   *uint      `gorm:"index" json:"target_metadata_id,omitempty"`
	TargetResourceID   *uint      `gorm:"index" json:"target_resource_id,omitempty"`
	PrimaryInventoryID *uint      `gorm:"index" json:"primary_inventory_id,omitempty"`
	CanonicalKey       string     `gorm:"size:512;index" json:"canonical_key"`
	VariantKey         string     `gorm:"size:512;index" json:"variant_key"`
	EditionKey         string     `gorm:"size:512;index" json:"edition_key"`
	ResourceShape      string     `gorm:"size:64;index" json:"resource_shape"`
	ReviewState        string     `gorm:"size:64;not null;default:pending;index:idx_recognition_candidates_type_state,priority:2" json:"review_state"`
	Confidence         *float64   `json:"confidence,omitempty"`
	EvidenceJSON       string     `gorm:"type:text" json:"evidence_json"`
	AlternativesJSON   string     `gorm:"type:text" json:"alternatives_json"`
	AffectedFilesJSON  string     `gorm:"type:text" json:"affected_files_json"`
	MaterializedAt     *time.Time `gorm:"index" json:"materialized_at,omitempty"`
	SupersededAt       *time.Time `gorm:"index" json:"superseded_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (RecognitionCandidate) TableName() string { return "recognition_candidates" }

type RecognitionEvidence struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ManifestID      uint      `gorm:"not null;index:idx_recognition_evidence_manifest_candidate,priority:1" json:"manifest_id"`
	CandidateID     *uint     `gorm:"index:idx_recognition_evidence_manifest_candidate,priority:2" json:"candidate_id,omitempty"`
	InventoryFileID *uint     `gorm:"index" json:"inventory_file_id,omitempty"`
	EvidenceKind    string    `gorm:"size:64;not null;index:idx_recognition_evidence_kind,priority:1" json:"evidence_kind"`
	EvidenceSource  string    `gorm:"size:128;not null;index:idx_recognition_evidence_kind,priority:2" json:"evidence_source"`
	EvidenceKey     string    `gorm:"size:255;index" json:"evidence_key"`
	EvidenceValue   string    `gorm:"size:2048" json:"evidence_value"`
	Strength        string    `gorm:"size:64;index" json:"strength"`
	Confidence      *float64  `json:"confidence,omitempty"`
	PayloadJSON     string    `gorm:"type:text" json:"payload_json"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (RecognitionEvidence) TableName() string { return "recognition_evidence" }

type RecognitionDecision struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	ManifestID       uint       `gorm:"not null;index:idx_recognition_decisions_manifest_status,priority:1" json:"manifest_id"`
	CandidateID      *uint      `gorm:"index" json:"candidate_id,omitempty"`
	DecisionType     string     `gorm:"size:64;not null;index" json:"decision_type"`
	Outcome          string     `gorm:"size:64;not null;index:idx_recognition_decisions_manifest_status,priority:2" json:"outcome"`
	TargetKind       string     `gorm:"size:64;index" json:"target_kind"`
	TargetKey        string     `gorm:"size:512;index" json:"target_key"`
	TargetMetadataID *uint      `gorm:"index" json:"target_metadata_id,omitempty"`
	TargetResourceID *uint      `gorm:"index" json:"target_resource_id,omitempty"`
	Confidence       *float64   `json:"confidence,omitempty"`
	Reason           string     `gorm:"size:1024" json:"reason"`
	EvidenceJSON     string     `gorm:"type:text" json:"evidence_json"`
	AlternativesJSON string     `gorm:"type:text" json:"alternatives_json"`
	ConflictsJSON    string     `gorm:"type:text" json:"conflicts_json"`
	ResolvedAt       *time.Time `gorm:"index" json:"resolved_at,omitempty"`
	SupersededAt     *time.Time `gorm:"index" json:"superseded_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (RecognitionDecision) TableName() string { return "recognition_decisions" }

type RecognitionConflict struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	ManifestID       uint       `gorm:"not null;index:idx_recognition_conflicts_manifest_status,priority:1" json:"manifest_id"`
	CandidateID      *uint      `gorm:"index" json:"candidate_id,omitempty"`
	ConflictKey      string     `gorm:"size:512;not null;index" json:"conflict_key"`
	ConflictType     string     `gorm:"size:64;not null;index" json:"conflict_type"`
	Severity         string     `gorm:"size:64;not null;default:blocking;index" json:"severity"`
	Status           string     `gorm:"size:64;not null;default:open;index:idx_recognition_conflicts_manifest_status,priority:2" json:"status"`
	Reason           string     `gorm:"size:1024" json:"reason"`
	EvidenceJSON     string     `gorm:"type:text" json:"evidence_json"`
	AlternativesJSON string     `gorm:"type:text" json:"alternatives_json"`
	ResolvedAt       *time.Time `gorm:"index" json:"resolved_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (RecognitionConflict) TableName() string { return "recognition_conflicts" }
