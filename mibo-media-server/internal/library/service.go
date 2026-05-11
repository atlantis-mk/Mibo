package library

import (
	"context"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/settings"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
)

type inventoryProbeExecutor func(context.Context, uint) error

type metadataMatchExecutor func(context.Context, uint, uint) error

type Service struct {
	cfg                    config.Config
	db                     *gorm.DB
	storage                *providers.Registry
	workflow               *workflow.Service
	ingest                 *ingest.Service
	inventoryProbeExecutor inventoryProbeExecutor
	metadataMatchExecutor  metadataMatchExecutor
}

type CreateMediaSourceInput struct {
	Provider   string                  `json:"provider"`
	Name       string                  `json:"name"`
	StorageRef string                  `json:"storage_ref"`
	RootPath   string                  `json:"root_path"`
	Config     *providers.SourceConfig `json:"config,omitempty"`
}

type UpdateMediaSourceInput struct {
	Name       string                  `json:"name"`
	StorageRef string                  `json:"storage_ref"`
	RootPath   string                  `json:"root_path"`
	Config     *providers.SourceConfig `json:"config,omitempty"`
}

type MediaSourceView struct {
	ID               uint                       `json:"id"`
	Name             string                     `json:"name"`
	Provider         string                     `json:"provider"`
	StorageRef       string                     `json:"storage_ref"`
	RootPath         string                     `json:"root_path"`
	Config           providers.SourceConfigView `json:"config,omitempty"`
	CapabilitiesJSON string                     `json:"capabilities_json"`
	CreatedAt        string                     `json:"created_at"`
	UpdatedAt        string                     `json:"updated_at"`
}

type CreateLibraryInput struct {
	Name               string                                       `json:"name"`
	MediaSourceID      uint                                         `json:"media_source_id"`
	RootPath           string                                       `json:"root_path"`
	Scan               *LibraryScanPolicyView                       `json:"scan,omitempty"`
	Metadata           *LibraryMetadataPolicyView                   `json:"metadata,omitempty"`
	MetadataStrategy   *settings.UpdateLibraryMetadataStrategyInput `json:"metadata_strategy,omitempty"`
	Playback           *LibraryPlaybackPolicyView                   `json:"playback,omitempty"`
	Subtitle           *LibrarySubtitlePolicyView                   `json:"subtitle,omitempty"`
	ScanExclusionRules []ScanExclusionRuleInput                     `json:"scan_exclusion_rules,omitempty"`
}

type targetedRefreshPayload struct {
	LibraryID uint   `json:"library_id"`
	RootPath  string `json:"root_path"`
	Reason    string `json:"reason"`
}

const (
	JobKindSyncLibrary                     = "sync_library"
	JobKindTargetedRefresh                 = "targeted_refresh"
	JobKindMatchCatalogItem                = "match_catalog_item"
	JobKindProbeInventoryFile              = "probe_inventory_file"
	JobKindMetadataMatchBatch              = "metadata_match_batch"
	JobKindRecognitionResolveBatch         = "recognition_resolve_batch"
	JobKindRecognitionPostResolveBatch     = "recognition_post_resolve_batch"
	JobKindInventoryProbeBatch             = "inventory_probe_batch"
	JobKindCatalogRefreshItemProjection    = "catalog_refresh_item_projection"
	JobKindCatalogRefreshLibraryProjection = "catalog_refresh_library_projection"
)

func NewService(cfg config.Config, db *gorm.DB, registry *providers.Registry, _ any, args ...any) *Service {
	return newLibraryService(cfg, db, newServiceDependencies(db, registry, args...))
}

func (s *Service) SetInventoryProbeExecutor(executor func(context.Context, uint) error) {
	s.inventoryProbeExecutor = executor
}

func (s *Service) SetMetadataMatchExecutor(executor func(context.Context, uint, uint) error) {
	s.metadataMatchExecutor = executor
}
