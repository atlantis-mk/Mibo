# Phase 16 Research — Catalog API, Search, and Progress Cutover

**Date:** 2026-04-25
**Phase:** 16 — Catalog API, Search, and Progress Cutover
**Requirements:** API-01, API-02, API-03, API-04

## Current State

- Catalog schema, DTO builders, projection refresh jobs, migration settings, and backfill control-plane contracts already exist in `mibo-media-server/internal/catalog`, `internal/database`, and `internal/httpapi` from Phases 12-13.
- TV metadata governance is planned in Phase 15 and introduces catalog-owned normalization, season/episode upserts, and series-first TMDB orchestration; Phase 16 should consume those contracts instead of adding parallel read/write logic.
- The current public app surface is still legacy-first:
  - browse/detail/series reads come from `internal/library/query*.go`
  - search comes from `internal/search/service.go`
  - progress writes `playback_progress` in `internal/progress/service.go`
  - HTTP routes are legacy `/media-items/*`, `/libraries/{id}/items`, `/discovery`, and `/me/progress`
  - frontend types in `web/src/lib/mibo-api.ts` still model `MediaItem`, `MediaFile`, and legacy progress payloads
- `catalog/contracts.go` already defines the target DTO shapes for list/detail/season/episode/asset/governance responses, so Phase 16 should hydrate those contracts rather than invent new API-specific structs.

## Recommended Cutover Strategy

### 1. Add additive catalog endpoints first

Phase 18 is the frontend migration, so Phase 16 should expose catalog-native endpoints without deleting or repointing the legacy routes yet.

Recommended additive routes:

- `GET /api/v1/items`
- `GET /api/v1/items/{id}`
- `GET /api/v1/series/{id}/seasons`
- `GET /api/v1/items/{id}/governance`
- `PATCH /api/v1/items/{id}/field-locks`
- `POST /api/v1/items/{id}/images/select`
- `POST /api/v1/items/{id}/match`
- `POST /api/v1/items/{id}/refresh-metadata`
- `POST /api/v1/assets/{asset_id}/link-item`
- `DELETE /api/v1/assets/{asset_id}/link-item/{item_id}`
- `GET /api/v1/items/{id}/progress`
- `POST /api/v1/me/item-progress`

This preserves the current web app until Phase 18 while still satisfying the phase goal and requirements with stable catalog contracts.

### 2. Put read composition in `internal/catalog`, not `internal/httpapi`

Existing repository patterns are consistent:

- DTO shaping lives in `internal/catalog/contracts.go`
- projection persistence lives in `internal/catalog/projections.go`
- handlers stay thin and call services (`handlers_catalog_migration.go`, `handlers_media.go`)

Phase 16 should follow the same split:

- `internal/catalog/*` owns list/detail/season/governance loading and governance mutations
- `internal/httpapi/*` owns auth, request decode, response mapping, and route registration only

### 3. Use `catalog_search_documents` for search/filter paths

Legacy search uses `search_documents` + `MediaItem`. Phase 16 should search on `catalog_search_documents` and hydrate `CatalogListItem` from `catalog_items`, `item_rollups`, `item_images`, and `catalog_external_ids`.

Implications:

- default browse should return top-level `movie` and `series` units
- search/filter query should match title/original title/people/tags/provider IDs from `catalog_search_documents`
- detail and seasons should come from hierarchy rows in `catalog_items`, not grouped legacy episode rows

### 4. Move progress writes to `user_item_data`

Legacy progress writes `playback_progress` keyed by `(user_id, media_item_id)`.

Phase 16 should add catalog progress keyed by `(user_id, item_id, asset_id)` using `user_item_data`:

- validate `asset_id` belongs to `item_id` through `asset_items`
- store playback position against the logical item plus chosen asset
- expose read/update endpoints by catalog item ID
- keep legacy `/api/v1/me/progress` untouched until the frontend migration phase

### 5. Reuse Phase 15 governance methods for provider-driven mutations

Manual governance should not duplicate provider orchestration logic in `httpapi`.

Phase 16 should call Phase 15 service contracts for provider-driven writes:

- `metadata.MatchCatalogSeries(...)`
- `metadata.RefreshCatalogSeriesMetadata(...)`

Field locks, image selection, and asset links belong in `internal/catalog` because they mutate catalog-owned tables directly.

## Architectural Responsibility Map

| Concern | Owning package | Why |
|--------|----------------|-----|
| Catalog item list/detail/season/governance reads | `internal/catalog` | DTO hydration and hierarchy composition already live there |
| Field lock, image selection, asset link mutations | `internal/catalog` | these write catalog-owned tables directly |
| Provider match/refresh trigger endpoints | `internal/httpapi` calling `internal/metadata` | handlers stay thin; metadata engine owns provider workflows |
| Catalog progress writes/reads | `internal/progress` | preserves the existing domain ownership for progress behavior |
| Public route registration and auth | `internal/httpapi` | matches current router/handler structure |

## Existing Patterns To Reuse

### Catalog service pattern

- `internal/catalog/service.go` — small input structs, validation, transactions, direct GORM writes
- `internal/catalog/projections.go` — query helpers plus projection-building in one package
- `internal/catalog/contracts.go` — all public catalog DTOs and scalar-safe summary projection rules

### HTTP handler pattern

- `internal/httpapi/handlers_catalog_migration.go` — auth-first, service availability checks, strict input validation, typed DTO responses
- `internal/httpapi/handlers_media.go` — path parsing helpers, handler-local request structs, route-to-service delegation

### Test pattern

- `internal/catalog/projections_test.go` — sqlite-backed catalog service tests with real tables
- `internal/progress/service_test.go` — direct package tests asserting state transitions and idempotence
- `internal/httpapi/catalog_migration_backfill_router_test.go` — end-to-end router tests using real service wiring and auth headers

## Constraints And Pitfalls

1. **Do not expose raw GORM rows from handlers.** The frozen contract is `catalog/contracts.go`.
2. **Do not reintroduce legacy virtual show grouping.** Series seasons must come from `catalog_items` hierarchy, not `groupShowBrowseCandidates` or `ListSeriesEpisodes`.
3. **Do not mutate provider evidence in handlers.** Use metadata/catalog services; handlers only decode/validate/route.
4. **Do not repurpose legacy routes yet.** Phase 18 still needs a controlled frontend cutover; additive routes are safer.
5. **Do not accept arbitrary `asset_id` on progress updates.** Validate the asset-item link before writing `user_item_data`.
6. **Do not bypass selected-image semantics.** `item_images` must keep exactly one selected row per `image_type`.
7. **Do not assume projection tables carry user progress.** `item_rollups` and `catalog_search_documents` are catalog/global projections; user progress remains in `user_item_data` and should be joined explicitly when needed.

## Recommended Plan Split

1. Catalog read/query foundation (`internal/catalog`) for browse/detail/series trees
2. Governance workspace + mutation helpers (`internal/catalog`)
3. Catalog progress state (`internal/progress`) backed by `user_item_data`
4. Additive HTTP routes (`internal/httpapi`) exposing items/series/governance/progress APIs

## Validation Architecture

**Framework:** `go test`

**Quick commands**

- `cd mibo-media-server && go test ./internal/catalog -run 'Test(ListItems|GetItemDetail|ListSeriesSeasons|GetGovernanceWorkspace|UpdateFieldLock|SelectImage|LinkAsset|UnlinkAsset)' -count=1`
- `cd mibo-media-server && go test ./internal/progress -run 'Test(UpdateCatalogProgress|GetCatalogProgressState)' -count=1`
- `cd mibo-media-server && go test ./internal/httpapi -run 'TestCatalog(ItemRoutes|GovernanceRoutes|ProgressRoutes)' -count=1`

**Full phase command**

- `cd mibo-media-server && go test ./internal/catalog ./internal/progress ./internal/httpapi -count=1`

**Manual spot-check after execution**

- Authenticated `GET /api/v1/items?library_id=<id>` returns catalog movies/series only
- Authenticated `GET /api/v1/items/{id}` returns catalog detail with selected images/assets
- Authenticated `GET /api/v1/series/{id}/seasons` returns season + episode availability from catalog hierarchy
- Authenticated `POST /api/v1/me/item-progress` followed by `GET /api/v1/items/{id}/progress` round-trips `item_id` / `asset_id`
