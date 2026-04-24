package library

import (
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/providers"
	"gorm.io/gorm"
)

type Service struct {
	cfg     config.Config
	db      *gorm.DB
	storage *providers.Registry
	jobs    *jobs.Service
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
	Name          string `json:"name"`
	Type          string `json:"type"`
	MediaSourceID uint   `json:"media_source_id"`
	RootPath      string `json:"root_path"`
}

type targetedRefreshPayload struct {
	LibraryID uint   `json:"library_id"`
	RootPath  string `json:"root_path"`
	Reason    string `json:"reason"`
}

const (
	JobKindSyncLibrary           = "sync_library"
	JobKindTargetedRefresh       = "targeted_refresh"
	JobKindMatchMediaItem        = "match_media_item"
	JobKindRefetchMediaItem      = "refetch_media_item"
	JobKindReindexSearchDocument = "reindex_search_document"
	JobKindReindexLibrarySearch  = "reindex_library_search"
)

func NewService(cfg config.Config, db *gorm.DB, registry *providers.Registry, jobs *jobs.Service) *Service {
	return &Service{cfg: cfg, db: db, storage: registry, jobs: jobs}
}
