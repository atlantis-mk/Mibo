package database

import "time"

type MediaAsset struct {
	ID                   uint       `gorm:"primaryKey" json:"id"`
	LibraryID            uint       `gorm:"not null;index" json:"library_id"`
	AssetType            string     `gorm:"size:64;not null;default:main;index" json:"asset_type"`
	DisplayName          string     `gorm:"size:512" json:"display_name"`
	Edition              string     `gorm:"size:128" json:"edition"`
	QualityLabel         string     `gorm:"size:128;index" json:"quality_label"`
	DurationSeconds      *float64   `json:"duration_seconds,omitempty"`
	Status               string     `gorm:"size:64;not null;default:available;index" json:"status"`
	ProbeStatus          string     `gorm:"size:64;not null;default:pending;index" json:"probe_status"`
	TechnicalSummaryJSON string     `gorm:"type:text" json:"technical_summary_json"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	DeletedAt            *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

type AssetItem struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	AssetID      uint      `gorm:"not null;uniqueIndex:idx_asset_items_asset_item_role_segment" json:"asset_id"`
	ItemID       uint      `gorm:"not null;uniqueIndex:idx_asset_items_asset_item_role_segment;index;index:idx_asset_items_item_role,priority:1" json:"item_id"`
	Role         string    `gorm:"size:64;not null;uniqueIndex:idx_asset_items_asset_item_role_segment;index:idx_asset_items_item_role,priority:2" json:"role"`
	SegmentIndex int       `gorm:"not null;default:0;uniqueIndex:idx_asset_items_asset_item_role_segment" json:"segment_index"`
	StartSeconds *float64  `json:"start_seconds,omitempty"`
	EndSeconds   *float64  `json:"end_seconds,omitempty"`
	Confidence   *float64  `json:"confidence,omitempty"`
	Source       string    `gorm:"size:64" json:"source"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type InventoryFile struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	LibraryID         uint       `gorm:"not null;index;index:idx_inventory_files_library_status_path,priority:1" json:"library_id"`
	StorageProvider   string     `gorm:"size:64;not null;uniqueIndex:idx_inventory_file_storage_path" json:"storage_provider"`
	StoragePath       string     `gorm:"size:2048;not null;uniqueIndex:idx_inventory_file_storage_path;index:idx_inventory_files_library_status_path,priority:3" json:"storage_path"`
	StableIdentityKey string     `gorm:"size:512;index" json:"stable_identity_key"`
	HashesJSON        string     `gorm:"type:text" json:"hashes_json"`
	SizeBytes         int64      `gorm:"not null;default:0" json:"size_bytes"`
	ModifiedAt        *time.Time `gorm:"index" json:"modified_at,omitempty"`
	Container         string     `gorm:"size:64;index" json:"container"`
	Status            string     `gorm:"size:64;not null;default:available;index;index:idx_inventory_files_library_status_path,priority:2" json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

func (InventoryFile) TableName() string {
	return "inventory_files"
}

type AssetFile struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	AssetID   uint      `gorm:"not null;uniqueIndex:idx_asset_files_asset_file_role_part;index:idx_asset_files_asset_part,priority:1" json:"asset_id"`
	FileID    uint      `gorm:"not null;uniqueIndex:idx_asset_files_asset_file_role_part;index" json:"file_id"`
	Role      string    `gorm:"size:64;not null;uniqueIndex:idx_asset_files_asset_file_role_part" json:"role"`
	PartIndex int       `gorm:"not null;default:0;uniqueIndex:idx_asset_files_asset_file_role_part;index:idx_asset_files_asset_part,priority:2" json:"part_index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MediaStream struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	FileID          uint      `gorm:"not null;uniqueIndex:idx_media_stream_file_index;index" json:"file_id"`
	StreamIndex     int       `gorm:"not null;uniqueIndex:idx_media_stream_file_index" json:"stream_index"`
	StreamType      string    `gorm:"size:64;not null;index" json:"stream_type"`
	Codec           string    `gorm:"size:128;index" json:"codec"`
	Profile         string    `gorm:"size:128" json:"profile,omitempty"`
	Level           *int      `json:"level,omitempty"`
	Language        string    `gorm:"size:32;index" json:"language"`
	Title           string    `gorm:"size:255" json:"title"`
	Width           *int      `json:"width,omitempty"`
	Height          *int      `json:"height,omitempty"`
	AvgFrameRate    string    `gorm:"size:32" json:"avg_frame_rate,omitempty"`
	RFrameRate      string    `gorm:"size:32" json:"r_frame_rate,omitempty"`
	FieldOrder      string    `gorm:"size:64" json:"field_order,omitempty"`
	ColorSpace      string    `gorm:"size:64" json:"color_space,omitempty"`
	BitDepth        *int      `json:"bit_depth,omitempty"`
	PixelFormat     string    `gorm:"size:64" json:"pixel_format,omitempty"`
	ReferenceFrames *int      `json:"reference_frames,omitempty"`
	Channels        *int      `json:"channels,omitempty"`
	ChannelLayout   string    `gorm:"size:64" json:"channel_layout,omitempty"`
	SampleRate      *int      `json:"sample_rate,omitempty"`
	BitRate         *int64    `json:"bit_rate,omitempty"`
	DurationSeconds *float64  `json:"duration_seconds,omitempty"`
	DispositionJSON string    `gorm:"type:text" json:"disposition_json"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
