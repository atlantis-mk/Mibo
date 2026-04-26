# Phase 15 Pattern Map

## Purpose

Concrete analogs and contracts for the Phase 15 planner/executor. Use these patterns directly; do not explore the codebase again unless a task explicitly sends you to a new file.

## Target Files And Closest Analogs

| Planned file | Role | Closest analog | Pattern to reuse |
|--------------|------|----------------|------------------|
| `mibo-media-server/internal/catalog/metadata_normalization.go` | catalog-owned normalization helpers | `mibo-media-server/internal/catalog/service.go` | methods on `*catalog.Service` that validate input, run a transaction, and update canonical tables through dedicated helpers |
| `mibo-media-server/internal/catalog/metadata_normalization_test.go` | focused catalog helper tests | `mibo-media-server/internal/catalog/service_test.go` | sqlite temp DB via `database.Open(...)`, assert persisted catalog rows directly |
| `mibo-media-server/internal/catalog/series_governance.go` | hierarchy/evidence upsert helpers | `mibo-media-server/internal/catalog/service.go` + `mibo-media-server/internal/catalog/projections.go` | keep hierarchy traversal and DB queries inside `catalog.Service`, return typed rows, avoid HTTP concerns |
| `mibo-media-server/internal/catalog/series_governance_test.go` | hierarchy idempotence tests | `mibo-media-server/internal/catalog/service_test.go` | seed series/season/episode rows directly and verify reuse/update behavior |
| `mibo-media-server/internal/metadata/catalog_series.go` | series-first TMDB orchestration | `mibo-media-server/internal/metadata/service_match.go` + `mibo-media-server/internal/metadata/service_tmdb.go` | metadata service fetches provider payloads, calculates match confidence, then delegates writes to catalog helpers |
| `mibo-media-server/internal/metadata/catalog_series_test.go` | httptest-backed orchestration tests | `mibo-media-server/internal/metadata/service_test.go` | spin up fake TMDB server, seed DB rows, assert stored provider/source/canonical data |

## Key Code Excerpts

### Catalog canonicalization pattern

From `mibo-media-server/internal/catalog/service.go`:

```go
func (s *Service) RecordMetadataSource(ctx context.Context, input MetadataSourceInput) (database.MetadataSource, error)
func (s *Service) SetExternalID(ctx context.Context, input ExternalIDInput) (database.CatalogExternalID, error)
func (s *Service) ApplyField(ctx context.Context, input ApplyFieldInput) (database.MetadataFieldState, bool, error)
```

**Use for Phase 15:** store raw TMDB payloads through `RecordMetadataSource`, canonical provider identity through `SetExternalID`, and lock-respecting field updates through `ApplyField(... Force:false)`.

### Catalog root + hierarchy pattern

From `mibo-media-server/internal/catalog/service.go`:

```go
func (s *Service) CreateItem(ctx context.Context, input CreateItemInput) (database.CatalogItem, error)
func (s *Service) ListChildren(ctx context.Context, parentID uint) ([]database.CatalogItem, error)
```

**Use for Phase 15:** continue to build rooted `series -> season -> episode` rows through catalog helpers instead of writing child rows from metadata code directly.

### TMDB orchestration pattern

From `mibo-media-server/internal/metadata/service_tmdb.go`:

```go
func (s *Service) searchTMDB(ctx context.Context, cfg config.TMDBConfig, mediaType, query string, year *int) (searchResponse, error)
func (s *Service) findByExternalID(ctx context.Context, cfg config.TMDBConfig, mediaType, externalSource, externalID string) ([]searchResult, error)
func (s *Service) fetchDetail(ctx context.Context, cfg config.TMDBConfig, mediaType string, id int) (detailResponse, error)
func (s *Service) fetchTVSeason(ctx context.Context, cfg config.TMDBConfig, seriesTMDBID int, seasonNumber int) (seasonDetailResponse, error)
```

**Use for Phase 15:** keep TMDB HTTP I/O in `metadata.Service`; do not invent a second client or move request code into `catalog.Service`.

### Existing metadata match pattern

From `mibo-media-server/internal/metadata/service_match.go`:

```go
func (s *Service) MatchItem(ctx context.Context, mediaItemID uint) error
func (s *Service) RefetchItem(ctx context.Context, mediaItemID uint) error
```

**Use for Phase 15:** follow the same high-level sequence:

1. resolve provider config
2. load current DB row(s)
3. fetch provider payload(s)
4. compute confidence / status
5. persist canonical state

But write into catalog tables, not legacy `MediaItem` columns.

### Projection refresh pattern

From `mibo-media-server/internal/catalog/projections.go`:

```go
func (s *Service) RefreshItemProjection(ctx context.Context, itemID uint) error
func (s *Service) RefreshLibraryProjection(ctx context.Context, libraryID uint, rootPath string) error
```

**Use for Phase 15:** run projection refresh after the root-series canonicalization pass finishes, instead of writing `item_rollups` or `catalog_search_documents` inline.

## Rules To Preserve

1. **Catalog writes stay in `internal/catalog`.**
2. **Raw provider payloads stay in `metadata_sources`; normalized values stay in catalog-owned tables.**
3. **Tests use sqlite temp DBs and `httptest` TMDB servers.**
4. **Phase 15 is engine-only; do not add Phase 16/19 API or UI work here.**
5. **If a refresh can overwrite manual state, route it through `ApplyField` with `Force:false`.**
