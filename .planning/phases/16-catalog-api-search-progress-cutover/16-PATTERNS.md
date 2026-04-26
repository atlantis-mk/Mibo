# Phase 16 Pattern Map

## Target Files And Best Analogs

| Target file | Closest analogs | Reuse guidance |
|-------------|-----------------|----------------|
| `mibo-media-server/internal/catalog/query_items.go` | `internal/catalog/projections.go`, `internal/catalog/contracts.go` | keep query composition in `catalog.Service`; hydrate frozen DTOs with builder functions instead of serializing DB models directly |
| `mibo-media-server/internal/catalog/query_items_test.go` | `internal/catalog/projections_test.go` | use sqlite-backed service tests with seeded catalog tables and direct DTO assertions |
| `mibo-media-server/internal/catalog/governance.go` | `internal/catalog/service.go`, `internal/catalog/backfill.go` | define small input structs, validate early, keep table mutations transaction-scoped |
| `mibo-media-server/internal/catalog/governance_test.go` | `internal/catalog/backfill_report_test.go`, `internal/catalog/projections_test.go` | assert persisted row changes directly from sqlite, not via mocks |
| `mibo-media-server/internal/progress/catalog_progress.go` | `internal/progress/service.go` | preserve progress domain ownership; add catalog-specific methods instead of rewriting the legacy service file wholesale |
| `mibo-media-server/internal/progress/catalog_progress_test.go` | `internal/progress/service_test.go` | test state transitions, completion semantics, and invalid asset rejection with real sqlite rows |
| `mibo-media-server/internal/httpapi/handlers_catalog_items.go` | `internal/httpapi/handlers_catalog_migration.go`, `internal/httpapi/handlers_media.go` | auth first, decode strict JSON, call one service method, return typed DTOs |
| `mibo-media-server/internal/httpapi/handlers_catalog_governance.go` | `internal/httpapi/handlers_catalog_migration.go`, `internal/httpapi/handlers_media.go` | split read routes from mutation routes; keep validation in handlers and business writes in services |
| `mibo-media-server/internal/httpapi/catalog_api_router_test.go` | `internal/httpapi/catalog_migration_backfill_router_test.go` | wire real services, create auth headers through `auth.Service`, and hit the router over `httptest` |

## Concrete Rules To Preserve

1. **Catalog DTO builders are the public boundary.** Use `BuildCatalogListItem`, `BuildCatalogItemDetail`, `BuildCatalogSeasonDetail`, `BuildCatalogEpisodeDetail`, and `BuildCatalogGovernanceWorkspace`.
2. **Scalar-safe provenance stays intact.** `projectCatalogSourceSummary` and `projectCatalogFieldStateValue` intentionally omit raw object blobs from read APIs.
3. **Handlers stay thin.** `handlers_catalog_migration.go` is the preferred pattern: auth check -> request validation -> service call -> `writeJSON`.
4. **Tests use real sqlite, not mocks.** `projections_test.go`, `service_test.go`, and `catalog_migration_backfill_router_test.go` prove the expected style.
5. **Legacy compatibility is preserved by addition, not mutation.** Add catalog routes in `router.go`; do not repoint the legacy `/media-items/*` and `/me/progress` paths in this phase.

## Key Interfaces Already In Place

From `internal/catalog/service.go`:

```go
type ExternalIDInput struct {
    ItemID       uint
    Provider     string
    ProviderType string
    ExternalID   string
    IsPrimary    bool
    Source       string
    Confidence   *float64
}

type ApplyFieldInput struct {
    ItemID         uint
    FieldKey       string
    Value          any
    SourceID       *uint
    Lock           bool
    LockReason     string
    EditedByUserID *uint
    Force          bool
}

func (s *Service) SetExternalID(ctx context.Context, input ExternalIDInput) (database.CatalogExternalID, error)
func (s *Service) ApplyField(ctx context.Context, input ApplyFieldInput) (database.MetadataFieldState, bool, error)
```

From `internal/catalog/contracts.go`:

```go
type CatalogListItem struct { /* frozen public DTO */ }
type CatalogItemDetail struct { /* frozen public DTO */ }
type CatalogSeasonDetail struct { /* frozen public DTO */ }
type CatalogEpisodeDetail struct { /* frozen public DTO */ }
type CatalogGovernanceWorkspace struct { /* frozen public DTO */ }
```

From `internal/database/catalog_models.go` and `inventory_models.go`:

```go
type UserItemData struct {
    UserID          uint
    ItemID          uint
    AssetID         *uint
    PositionSeconds int
    PlayedPercentage *float64
    PlayCount       int
    LastPlayedAt    *time.Time
    CompletedAt     *time.Time
}

type AssetItem struct {
    AssetID      uint
    ItemID       uint
    Role         string
    SegmentIndex int
    StartSeconds *float64
    EndSeconds   *float64
    Confidence   *float64
    Source       string
}
```
