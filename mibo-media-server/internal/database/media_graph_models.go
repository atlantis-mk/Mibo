package database

import "time"

type InventorySidecarSignal struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	InventoryFileID   *uint      `gorm:"index" json:"inventory_file_id,omitempty"`
	LibraryID         uint       `gorm:"not null;index:idx_inventory_sidecar_signals_library_state,priority:1" json:"library_id"`
	StorageProvider   string     `gorm:"size:64;not null;uniqueIndex:idx_inventory_sidecar_signal_identity,priority:1" json:"storage_provider"`
	VideoStoragePath  string     `gorm:"size:2048;not null;uniqueIndex:idx_inventory_sidecar_signal_identity,priority:2;index" json:"video_storage_path"`
	SidecarPath       string     `gorm:"size:2048;not null;uniqueIndex:idx_inventory_sidecar_signal_identity,priority:3;index" json:"sidecar_path"`
	ParentPath        string     `gorm:"size:2048;not null;index" json:"parent_path"`
	Extension         string     `gorm:"size:32;not null;index" json:"extension"`
	AssociationSource string     `gorm:"size:64;not null" json:"association_source"`
	ParseStatus       string     `gorm:"size:64;not null;index" json:"parse_status"`
	MediaType         string     `gorm:"size:64;index" json:"media_type"`
	Title             string     `gorm:"size:512;index" json:"title"`
	OriginalTitle     string     `gorm:"size:512" json:"original_title"`
	Year              *int       `gorm:"index" json:"year,omitempty"`
	SeriesTitle       string     `gorm:"size:512;index" json:"series_title"`
	SeasonNumber      *int       `gorm:"index" json:"season_number,omitempty"`
	EpisodeNumber     *int       `gorm:"index" json:"episode_number,omitempty"`
	ExternalIDsJSON   string     `gorm:"type:text" json:"external_ids_json"`
	FieldsJSON        string     `gorm:"type:text" json:"fields_json"`
	ParseError        string     `gorm:"size:1024" json:"parse_error"`
	FileFingerprint   string     `gorm:"size:128;not null;index" json:"file_fingerprint"`
	InvalidatedAt     *time.Time `gorm:"index" json:"invalidated_at,omitempty"`
	LastObservedAt    time.Time  `gorm:"not null;index" json:"last_observed_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (InventorySidecarSignal) TableName() string { return "inventory_sidecar_signals" }

type MediaGraphNode struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ManifestID      uint      `gorm:"not null;index:idx_media_graph_node_identity,priority:1" json:"manifest_id"`
	NodeKey         string    `gorm:"size:512;not null;uniqueIndex:idx_media_graph_node_identity,priority:2;index" json:"node_key"`
	NodeKind        string    `gorm:"size:64;not null;index:idx_media_graph_nodes_kind_state,priority:1" json:"node_kind"`
	ParentNodeKey   string    `gorm:"size:512;index" json:"parent_node_key"`
	InventoryFileID *uint     `gorm:"index" json:"inventory_file_id,omitempty"`
	CandidateKey    string    `gorm:"size:512;index" json:"candidate_key"`
	Confidence      *float64  `json:"confidence,omitempty"`
	PayloadJSON     string    `gorm:"type:text" json:"payload_json"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (MediaGraphNode) TableName() string { return "media_graph_nodes" }

type MediaGraphEdge struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ManifestID  uint      `gorm:"not null;index:idx_media_graph_edge_identity,priority:1" json:"manifest_id"`
	FromNodeKey string    `gorm:"size:512;not null;uniqueIndex:idx_media_graph_edge_identity,priority:2;index" json:"from_node_key"`
	ToNodeKey   string    `gorm:"size:512;not null;uniqueIndex:idx_media_graph_edge_identity,priority:3;index" json:"to_node_key"`
	EdgeKind    string    `gorm:"size:64;not null;uniqueIndex:idx_media_graph_edge_identity,priority:4;index" json:"edge_kind"`
	Confidence  *float64  `json:"confidence,omitempty"`
	PayloadJSON string    `gorm:"type:text" json:"payload_json"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (MediaGraphEdge) TableName() string { return "media_graph_edges" }

type MediaGraphClassification struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	ManifestID       uint       `gorm:"not null;index:idx_media_graph_classification_identity,priority:1" json:"manifest_id"`
	GroupNodeKey     string     `gorm:"size:512;not null;uniqueIndex:idx_media_graph_classification_identity,priority:2;index" json:"group_node_key"`
	GroupKind        string     `gorm:"size:64;not null;index:idx_media_graph_classifications_state,priority:1" json:"group_kind"`
	ClassifiedAs     string     `gorm:"size:64;not null;index:idx_media_graph_classifications_state,priority:2" json:"classified_as"`
	ReviewState      string     `gorm:"size:64;not null;default:pending;index:idx_media_graph_classifications_state,priority:3" json:"review_state"`
	Confidence       *float64   `json:"confidence,omitempty"`
	Reason           string     `gorm:"size:1024" json:"reason"`
	EvidenceJSON     string     `gorm:"type:text" json:"evidence_json"`
	AlternativesJSON string     `gorm:"type:text" json:"alternatives_json"`
	SupersededAt     *time.Time `gorm:"index" json:"superseded_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (MediaGraphClassification) TableName() string { return "media_graph_classifications" }
