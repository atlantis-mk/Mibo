# Phase 12 Research — Catalog Kernel Contracts & Migration Guards

**Phase:** 12 — Catalog Kernel Contracts & Migration Guards  
**Requirements:** KERN-01, KERN-02, PROD-01  
**Date:** 2026-04-25

## Research Goal

Answer: what must be true before later v3 phases can safely backfill, scan, read, and play from the catalog kernel without a big-bang cutover.

## Current Codebase Facts

### Catalog kernel already exists

- `mibo-media-server/internal/database/catalog_models.go` defines `CatalogItem`, `CatalogExternalID`, `MetadataSource`, `MetadataFieldState`, `ItemImage`, `UserItemData`, `ItemRollup`, and `CatalogSearchDocument`.
- `mibo-media-server/internal/database/inventory_models.go` defines `MediaAsset`, `AssetItem`, `InventoryFile`, `AssetFile`, and `MediaStream`.
- `mibo-media-server/internal/database/database.go` already AutoMigrates both legacy and catalog tables in one startup path.

### Service layer foundations already exist

- `mibo-media-server/internal/catalog/service.go` already freezes key enum strings in code: item types (`movie`, `series`, `season`, `episode`, `extra`), availability states, governance states, metadata source types.
- `mibo-media-server/internal/inventory/service.go` already provides the basic asset/file/item linking semantics with idempotent upserts for `InventoryFile`, `AssetItem`, and `AssetFile`.

### The missing phase-12 pieces are contract and guardrail oriented

- No explicit catalog DTO layer exists yet; current API patterns still commonly serialize `database.*` or legacy service views.
- No dedicated catalog migration state exists in `settings.Service`; current settings categories are only `metadata` and `scan`.
- Projection refresh jobs exist only for legacy search (`reindex_search_document`, `reindex_library_search`) and are wired through `library.Service` + `worker.Runner`.
- Catalog tables exist, but minimum cutover-era composite indexes and boot-compatibility tests are still thin.

## Existing Patterns To Reuse

### Explicit service-owned response/view structs

- `mibo-media-server/internal/library/service.go` exposes `MediaSourceView` instead of returning DB rows as the public contract.
- `mibo-media-server/internal/settings/service.go` exposes `MetadataSettings` and `ScanSettings` as API-safe structs.

**Implication:** Catalog DTOs should live outside `internal/database` and be plain exported structs with `json` tags only.

### Durable settings via category/key rows

- `settings.Service` persists values into `database.SystemSetting` with `(category,key)` uniqueness.
- Upserts use `clause.OnConflict` and parse strings into typed settings in service methods.

**Implication:** Migration flags should reuse `SystemSetting` with a dedicated category, not invent a new table in phase 12.

### Queueable worker contracts

- `library.Service` defines job kind constants and enqueue helpers.
- `worker.Runner.handleJob` switches on those constants and decodes typed payloads.
- `worker_test.go` proves job wiring with real DB rows and `RunOnce`.

**Implication:** Catalog projection refresh entrypoints should follow the same constant + queue helper + worker branch + test pattern.

### Migration safety pattern

- `database.Open` is the only startup migration boundary.
- `catalog_models_test.go` already proves catalog tables appear in a fresh SQLite DB.

**Implication:** Phase 12 should extend startup and migration tests here instead of creating a second migration runner.

## Recommended Phase-12 Implementation Shape

1. **Freeze DTOs first** so later API and frontend cutover phases build against explicit catalog contracts instead of raw `database.CatalogItem` / `database.MediaAsset` rows.
2. **Add durable migration state next** so backfill completion, read enablement, and legacy cleanup status are observable before any cutover work begins.
3. **Define projection refresh entrypoints now** using queueable worker contracts, even if later phases expand the refresh logic. The contract must exist before scanner/metadata/API cutover work depends on it.
4. **Finish with minimum composite indexes and startup compatibility tests** so empty DB boot and legacy DB boot remain safe while catalog constraints tighten.

## Required Contracts To Freeze

### DTOs

Phase 12 should create explicit exported DTOs for:

- `CatalogListItem`
- `CatalogItemDetail`
- `CatalogSeasonDetail`
- `CatalogEpisodeDetail`
- `CatalogAssetDetail`
- `CatalogGovernanceWorkspace`

These DTOs should:

- use `series`, not legacy `show`, for the canonical TV root type
- avoid `gorm` tags and avoid embedding `database.*` structs
- expose availability/governance as explicit string fields using catalog constants
- carry asset/image/source-evidence/field-state data in nested value objects rather than leaking DB-row shape

### Migration state

Durable settings keys required by roadmap + quick-task research:

- `catalog_backfill_completed_at`
- `catalog_read_enabled`
- `legacy_cleanup_completed_at`

Recommended storage:

- category: `catalog_migration`
- typed service methods for parse/serialize
- RFC3339 for timestamps, `true|false` string storage for booleans

### Projection refresh contract

Define queueable catalog entrypoints equivalent to legacy search reindex hooks:

- item-scope refresh entrypoint
- library-scope refresh entrypoint
- worker dispatch branches for both
- tests proving both entrypoints succeed on empty DB and seeded catalog rows

## Constraint And Index Guidance

### Codebase-specific minimums

The roadmap and requirements require minimum indexes for:

- catalog hierarchy reads
- provider identity uniqueness
- metadata field state uniqueness
- asset links
- inventory file uniqueness
- search projections

The existing schema already covers several uniqueness rules:

- `CatalogExternalID`: `idx_catalog_external_identity`
- `MetadataFieldState`: `idx_metadata_field_state_item_field`
- `AssetItem`: `idx_asset_items_asset_item_role_segment`
- `InventoryFile`: `idx_inventory_file_storage_path`
- `AssetFile`: `idx_asset_files_asset_file_role_part`
- `UserItemData`: `idx_user_item_data_user_item_asset`

Phase 12 still needs explicit composite lookup indexes for the cutover paths most likely to be hit by list/detail/season traversal and projection rebuilds.

Recommended additions:

- `catalog_items(library_id, type, availability_status, sort_key)`
- `catalog_items(parent_id, parent_index_number, index_number)`
- `catalog_items(root_id, type, parent_index_number, index_number)`
- `catalog_search_documents(library_id, item_type, availability_status, title)`

### GORM migration research

Context7 GORM docs confirm:

- `AutoMigrate` creates missing tables, indexes, foreign keys, and constraints, but does **not** delete unused columns.
- composite indexes should be defined by assigning the same index name to multiple fields.
- `Migrator().CreateConstraint` / `CreateIndex` is the supported explicit migration path when tags alone are insufficient.
- GORM can disable FK auto-creation globally, but this repo currently does not; phase 12 should preserve the current startup path and add only the minimum safe guards.

### Practical migration takeaway

- Prefer additive indexes and explicit startup tests in phase 12.
- Do **not** attempt destructive cleanup or table removal in phase 12.
- If any foreign key/check constraint proves cross-database fragile, keep the phase-12 scope on composite indexes + boot compatibility and defer stricter cleanup constraints to Phase 20.

## Main Risks

1. **Raw model leakage risk** — later API phases may accidentally serialize `database.CatalogItem` if DTOs are not frozen now.
2. **Unauthorized cutover risk** — a writable `catalog_read_enabled` flag must stay behind authenticated settings endpoints.
3. **Projection drift risk** — later scanner/metadata phases need stable queue entrypoints before they can safely trigger rollup/search refresh.
4. **Migration fragility risk** — new indexes must not break empty startup or legacy-only startup.
5. **Cross-database behavior risk** — SQLite and Postgres differ on some constraint behaviors; phase 12 should verify additive behavior rather than overreaching on destructive DDL.

## Validation Architecture

### Fast feedback

- `go test ./internal/catalog ./internal/settings ./internal/database ./internal/worker -run 'Test(Catalog|DatabaseOpen|RunOnce.*Catalog|CatalogMigration)'`

### Full phase regression

- `go test ./internal/database ./internal/catalog ./internal/inventory ./internal/settings ./internal/httpapi ./internal/worker`

### Required proof points

- DTO contract tests verify JSON field names and prevent raw `database.*` leakage.
- Settings tests verify missing/default parsing, valid writes, invalid timestamp rejection, and durable round-trips.
- Worker tests verify catalog projection jobs are claimable and dispatch correctly.
- Database tests verify empty startup and legacy-database startup both still succeed after index additions.

## Architectural Responsibility Map

| Concern | Correct layer | Why |
|---------|---------------|-----|
| Catalog DTO definitions | `internal/catalog` or API-facing service layer | Keeps DB rows private and reusable across HTTP/API cutover |
| Migration state persistence | `internal/settings` + `internal/httpapi` | Existing durable settings path already owns operator-visible flags |
| Projection queue entrypoints | `internal/library` + `internal/worker` + `internal/catalog` | Matches existing job enqueue/dispatch pattern |
| Index and startup guards | `internal/database` tests and migration boundary | Single source of truth for schema boot behavior |

## Planning Implications

- Phase 12 does **not** need frontend work.
- Phase 12 should produce 3-4 small execution plans, with DTOs, settings, projection jobs, and DB hardening isolated for parallel work where files do not overlap.
- Every plan should keep compatibility with both empty DB startup and legacy DB startup.
