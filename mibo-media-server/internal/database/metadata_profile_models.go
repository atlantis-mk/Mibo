package database

import "time"

const (
	MetadataProviderTypeTMDB      = "tmdb"
	MetadataProviderTypeTVDB      = "tvdb"
	MetadataProviderTypeMetaTube  = "metatube"
	MetadataProviderTypeLocalScan = "local_scan"

	MetadataProviderAvailabilityAvailable   = "available"
	MetadataProviderAvailabilityUnavailable = "unavailable"
	MetadataProviderAvailabilityCooldown    = "cooldown"

	BuiltInLocalScanProviderInstanceName = "local_scan"
)

type MetadataProviderInstance struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	Name               string     `gorm:"size:255;not null;uniqueIndex" json:"name"`
	ProviderType       string     `gorm:"size:64;not null;index" json:"provider_type"`
	Enabled            bool       `gorm:"not null;default:true;index" json:"enabled"`
	AvailabilityStatus string     `gorm:"size:64;not null;default:available;index" json:"availability_status"`
	FailureReason      string     `gorm:"type:text" json:"failure_reason"`
	CooldownUntil      *time.Time `gorm:"index" json:"cooldown_until,omitempty"`
	ConfigJSON         string     `gorm:"type:text" json:"config_json"`
	SystemManaged      bool       `gorm:"not null;default:false;index" json:"system_managed"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type MetadataProfile struct {
	ID                        uint      `gorm:"primaryKey" json:"id"`
	Name                      string    `gorm:"size:255;not null;uniqueIndex" json:"name"`
	Description               string    `gorm:"type:text" json:"description"`
	SearchProvidersJSON       string    `gorm:"type:text" json:"search_providers_json"`
	DetailProvidersJSON       string    `gorm:"type:text" json:"detail_providers_json"`
	ImageProvidersJSON        string    `gorm:"type:text" json:"image_providers_json"`
	PeopleProvidersJSON       string    `gorm:"type:text" json:"people_providers_json"`
	HierarchyProvidersJSON    string    `gorm:"type:text" json:"hierarchy_providers_json"`
	PreferredMetadataLanguage string    `gorm:"size:32" json:"preferred_metadata_language"`
	PreferredImageLanguage    string    `gorm:"size:32" json:"preferred_image_language"`
	FallbackEnabled           bool      `gorm:"not null;default:true" json:"fallback_enabled"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

type LibraryMetadataStrategy struct {
	ID                        uint      `gorm:"primaryKey" json:"id"`
	LibraryID                 uint      `gorm:"not null;uniqueIndex" json:"library_id"`
	MetadataProfileID         *uint     `gorm:"index" json:"metadata_profile_id,omitempty"`
	SearchProvidersJSON       string    `gorm:"type:text" json:"search_providers_json"`
	DetailProvidersJSON       string    `gorm:"type:text" json:"detail_providers_json"`
	ImageProvidersJSON        string    `gorm:"type:text" json:"image_providers_json"`
	PeopleProvidersJSON       string    `gorm:"type:text" json:"people_providers_json"`
	HierarchyProvidersJSON    string    `gorm:"type:text" json:"hierarchy_providers_json"`
	PreferredMetadataLanguage string    `gorm:"size:32" json:"preferred_metadata_language"`
	PreferredImageLanguage    string    `gorm:"size:32" json:"preferred_image_language"`
	MetadataCountryCode       string    `gorm:"size:16" json:"metadata_country_code"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

func (LibraryMetadataStrategy) TableName() string {
	return "library_metadata_strategies"
}
