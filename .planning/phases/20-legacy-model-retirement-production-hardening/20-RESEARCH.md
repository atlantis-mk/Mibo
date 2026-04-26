# Phase 20 Research — Legacy Model Retirement & Production Hardening

**Phase:** 20 — Legacy Model Retirement & Production Hardening  
**Requirements:** PROD-02, PROD-03, PROD-04, MIGR-04  
**Date:** 2026-04-25

## Research Goal

Answer: what must change so Mibo can treat the catalog kernel as the only normal
runtime path, keep legacy migration visibility available to operators, and add
the production hardening needed to trust catalog browse/search/playback/progress
in day-to-day use.

## No User Context Artifact

- No phase-specific `CONTEXT.md` exists for Phase 20.
- Planning therefore uses `ROADMAP.md`, `REQUIREMENTS.md`, shipped phase
  summaries, already-written phase plans, and current source behavior as the
  authoritative inputs.

## Current Codebase Facts

### Normal browse/search/progress code still depends on legacy tables

- `mibo-media-server/internal/library/query_browse.go` still falls back to
  `database.MediaItem`, `database.SearchDocument`, and `database.PlaybackProgress`.
- `mibo-media-server/internal/search/service.go` still indexes and searches the
  legacy `search_documents` table built from `database.MediaItem`.
- `mibo-media-server/internal/progress/service.go` still stores state in
  `database.PlaybackProgress` keyed by `media_item_id` and optional
  `media_file_id`, then triggers legacy search reindex work.

### Playback and transport are still visibly legacy in the current tree

- `mibo-media-server/internal/playback/service.go` still loads
  `database.MediaItem` plus child `database.MediaFile` rows.
- `mibo-media-server/internal/httpapi/router.go` still registers
  `/api/v1/media-items/{id}` and `/api/v1/media-files/{id}` playback/file routes.
- Phase 17 planning already defines the intended replacement routes:
  `/api/v1/items/{id}/playback`, `/api/v1/assets/{id}/link`, and
  `/api/v1/inventory-files/{id}/*`.

### Catalog-side projection and inventory foundations already exist

- `database.CatalogItem`, `database.CatalogSearchDocument`, `database.UserItemData`,
  `database.MediaAsset`, `database.AssetItem`, `database.InventoryFile`, and
  `database.AssetFile` are already migrated.
- Phase 12 established catalog projection rebuilds, additive startup index
  backstops, and search-document canonicalization.
- Phase 14 and Phase 15 plans already define the intended catalog-first
  availability, metadata, and projection semantics the runtime should trust.

### Existing startup hardening is only partial

- `mibo-media-server/internal/database/database.go` explicitly backstops four
  catalog indexes, but many catalog-kernel uniqueness guarantees still rely on
  GORM tags alone.
- Unique keys for external identity, field state, asset links, inventory files,
  asset files, media streams, and `user_item_data` already exist in model tags,
  but there is no explicit preflight for duplicate legacy-era rows before
  production startup hardening.

### Operator-visible migration state already exists and should remain

- `internal/settings/catalog_migration.go` persists
  `catalog_backfill_completed_at`, `catalog_read_enabled`, and
  `legacy_cleanup_completed_at` in `SystemSetting` rows.
- `internal/httpapi/handlers_system.go` already exposes `catalog_migration`
  state in `/api/v1/system/info`.
- `internal/httpapi/handlers_catalog_migration.go` already provides authenticated
  backfill queue/list/detail routes. Those are the right compatibility boundary
  to keep while normal browse/playback/progress paths stop touching legacy main
  tables.

## Existing Patterns To Reuse

### 1. Use catalog-owned service helpers plus worker jobs for repair work

- Phase 12 projection refresh and Phase 13 backfill both follow the same shape:
  typed service methods in `internal/catalog`, queue helpers, worker dispatch,
  and focused HTTP/router tests.
- Phase 20 consistency repair should reuse that pattern instead of inventing a
  second operations framework.

### 2. Keep startup hardening additive and explicit

- Phase 12-04 proved the right pattern for schema hardening:
  `database.Open(...)` remains the boundary, `HasIndex/CreateIndex` is the
  additive backstop, and sqlite tests prove fresh, legacy-only, and repeated
  startup behavior.
- Phase 20 should extend that pattern to the remaining catalog-kernel
  uniqueness/index guarantees and fail with actionable errors when duplicate data
  would make hardening unsafe.

### 3. Preserve explainable 200-level runtime failures

- Playback plans and current playback code already treat missing or unsupported
  files as structured `decision` payloads, not opaque 500s.
- Phase 20 legacy removal must preserve that UX while deleting the old media
  route and table dependencies.

### 4. Prefer focused test files over adding more mass to monolith tests

- Existing code already split some newer work into dedicated files like
  `worker_catalog_test.go`, `worker_catalog_backfill_test.go`, and focused router
  tests.
- Phase 20 should continue that pattern for consistency jobs, catalog runtime
  cutover regressions, and startup constraint checks.

## Recommended Phase-20 Implementation Shape

1. **Audit before enforcing.**
   - Add a catalog-kernel consistency audit that detects duplicate identities,
     invalid asset/file links, stale rollups, stale search documents, and
     availability mismatches.

2. **Make repairs first-class jobs.**
   - Queue audit and repair runs through the existing jobs/worker model.
   - Keep the operator surface typed and authenticated.

3. **Only then tighten startup hardening.**
   - Explicitly backstop the remaining unique indexes/constraints in
     `database.Open(...)`.
   - Add preflight duplicate detection so startup fails safely with a concrete
     message instead of partially migrating into a broken state.

4. **Retire legacy runtime code path-by-path.**
   - Browse/search must read catalog projections.
   - Progress must read/write `user_item_data`.
   - Playback must use item/asset/inventory-file identifiers only.
   - Normal HTTP runtime must stop registering legacy media item/file endpoints.

5. **Finish with frontend contract cleanup and operator validation.**
   - Remove lingering `MediaItem` / `MediaFile` client contracts from the web app.
   - Add a repeatable production-check script and a cleanup/runbook document.

## Required Mapping Decisions

### Catalog consistency issue families

Recommended issue codes:

- `duplicate_external_identity`
- `duplicate_inventory_file_path`
- `duplicate_asset_item_role_segment`
- `duplicate_asset_file_role_part`
- `duplicate_user_item_data`
- `missing_primary_source_file`
- `availability_mismatch`
- `missing_item_rollup`
- `missing_catalog_search_document`
- `stale_projection_timestamp`

### Repair surface

Recommended repair modes:

- `availability`
- `rollups`
- `search_documents`
- `projection_refresh`
- `full`

### Legacy compatibility boundary

- Keep authenticated migration/report/status reads under
  `/api/v1/catalog-migration/*` and `/api/v1/settings/catalog-migration`.
- Do **not** keep legacy browse/search/playback/progress as normal runtime
  compatibility paths.
- If a legacy route must survive temporarily, it should be compatibility-only,
  clearly deprecated, and never mutate legacy main-path rows.

## Main Risks

1. **Unsafe hardening risk** — creating unique indexes without preflight
   duplicate detection can break startup on migrated databases.
2. **Hidden legacy dependency risk** — browse/search/progress/playback may look
   catalog-ready while still reading or writing `MediaItem`, `MediaFile`,
   `SearchDocument`, or `PlaybackProgress` internally.
3. **Repair blast-radius risk** — an unscoped consistency repair can mutate more
   libraries than intended.
4. **Compatibility ambiguity risk** — keeping too many legacy endpoints alive
   makes it unclear which path is authoritative.
5. **Frontend drift risk** — the UI can still compile against old `MediaItem`
   contracts even after backend cutover unless client types and query helpers are
   cleaned up deliberately.

## Validation Architecture

### Fast feedback

- `cd mibo-media-server && go test ./internal/catalog -run 'TestCatalogConsistency' -count=1`
- `cd mibo-media-server && go test ./internal/database -run 'TestDatabaseOpenCatalogKernel' -count=1`
- `cd mibo-media-server && go test ./internal/library ./internal/progress ./internal/search -run 'Test(Catalog|UserItemData)' -count=1`
- `cd mibo-media-server && go test ./internal/httpapi -run 'Test(CatalogRuntime|CatalogConsistency)' -count=1`

### Integration feedback

- `cd mibo-media-server && go test ./internal/catalog ./internal/httpapi ./internal/playback ./internal/progress ./internal/search -count=1`

### Full phase regression

- `cd mibo-media-server && go test ./... -count=1`
- `cd web && pnpm typecheck && pnpm build`

### Required proof points

- Catalog consistency audit detects duplicate-key, availability, and projection
  issues deterministically.
- Operators can queue and inspect audit/repair runs through authenticated routes.
- Empty DB startup, legacy DB startup, and repeated startup all pass with the
  hardened catalog constraints/indexes in place.
- Normal browse/search/progress/playback paths no longer depend on legacy main
  tables.
- Frontend contracts and build output no longer depend on legacy `MediaItem` /
  `MediaFile` APIs.

## Architectural Responsibility Map

| Concern | Correct layer | Why |
|---------|---------------|-----|
| Consistency auditing and repair logic | `internal/catalog` | catalog kernel owns integrity rules and projection truth |
| Job queuing and execution | `internal/httpapi` + `internal/worker` | follows the existing authenticated enqueue + worker dispatch pattern |
| Startup constraint/index backstops | `internal/database` | `database.Open(...)` is already the additive migration boundary |
| Catalog-only browse/search/progress runtime | `internal/library`, `internal/search`, `internal/progress` | these packages own the current legacy runtime dependencies being retired |
| Playback route retirement and inventory-only transport | `internal/playback` + `internal/httpapi` | item/asset selection stays in domain code, route registration stays in HTTP |
| Frontend contract cleanup and operator runbook | `web/src/lib/*`, `web/src/features/*`, `docs/media-architecture/*` | client contracts and deployment instructions must move together |

## Planning Implications

- Phase 20 is a mixed backend + frontend hardening phase.
- The safest breakdown is six plans across four waves:
  1. consistency audit contracts,
  2. consistency jobs and operator endpoints,
  3. startup constraint/index hardening,
  4. browse/search/progress legacy retirement,
  5. playback/http legacy retirement,
  6. frontend cleanup plus production validation docs.
- Phase 20 should assume Phases 14-19 have landed their catalog-first write,
  metadata, API, playback, and frontend contracts before execution begins.
