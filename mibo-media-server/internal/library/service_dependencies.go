package library

import (
	"context"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/workflow"
	"gorm.io/gorm"
)

type serviceDependencies struct {
	storage  *providers.Registry
	workflow *workflow.Service
	ingest   *ingest.Service
}

type executorDependencies struct {
	inventoryProbe inventoryProbeExecutor
	metadataMatch  metadataMatchExecutor
}

func newServiceDependencies(db *gorm.DB, registry *providers.Registry, args ...any) serviceDependencies {
	deps := serviceDependencies{storage: registry}
	for _, arg := range args {
		if ingestSvc, ok := arg.(*ingest.Service); ok {
			deps.ingest = ingestSvc
		}
		if workflowSvc, ok := arg.(*workflow.Service); ok {
			deps.workflow = workflowSvc
		}
	}
	if deps.workflow == nil && db != nil {
		deps.workflow = workflow.NewService(db)
	}
	return deps
}

func newLibraryService(cfg config.Config, db *gorm.DB, deps serviceDependencies) *Service {
	return &Service{cfg: cfg, db: db, storage: deps.storage, workflow: deps.workflow, ingest: deps.ingest}
}

func (s *Service) dependencies() serviceDependencies {
	if s == nil {
		return serviceDependencies{}
	}
	return serviceDependencies{storage: s.storage, workflow: s.workflow, ingest: s.ingest}
}

func (s *Service) withDependencies(deps serviceDependencies) *Service {
	if s == nil {
		return nil
	}
	clone := *s
	clone.storage = deps.storage
	clone.workflow = deps.workflow
	clone.ingest = deps.ingest
	return &clone
}

func (s *Service) runWithWorkflow(ctx context.Context, fn func(*workflow.Service) error) error {
	if s == nil || s.workflow == nil {
		return context.Canceled
	}
	return fn(s.workflow)
}

func (s *Service) storageRegistry() *providers.Registry {
	if s == nil {
		return nil
	}
	return s.storage
}

func (s *Service) workflowCapability() *workflow.Service {
	if s == nil {
		return nil
	}
	return s.workflow
}

func (s *Service) ingestCapability() *ingest.Service {
	if s == nil {
		return nil
	}
	return s.ingest
}

func (s *Service) executors() executorDependencies {
	if s == nil {
		return executorDependencies{}
	}
	return executorDependencies{inventoryProbe: s.inventoryProbeExecutor, metadataMatch: s.metadataMatchExecutor}
}

func (s *Service) inventoryProbeCapability() inventoryProbeExecutor {
	return s.executors().inventoryProbe
}

func (s *Service) metadataMatchCapability() metadataMatchExecutor {
	return s.executors().metadataMatch
}
