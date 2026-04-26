# Phase 15 Research — Series-Level Metadata Governance Engine

**Date:** 2026-04-25
**Status:** Complete
**Phase:** 15 — Series-Level Metadata Governance Engine

## Objective

Answer: what needs to be true to plan Phase 15 well, given the current catalog kernel, legacy metadata stack, and the v3 migration ordering.

## Source Artifacts Read

- `.planning/ROADMAP.md`
- `.planning/REQUIREMENTS.md`
- `.planning/STATE.md`
- `.planning/PROJECT.md`
- `.planning/quick/260425-tvg-catalog-kernel-remaining-plan/260425-tvg-CONTEXT.md`
- `.planning/quick/260425-tvg-catalog-kernel-remaining-plan/260425-tvg-RESEARCH.md`
- `.planning/TV-SERIES-METADATA-GOVERNANCE-PLAN.md`
- `.planning/phases/12-catalog-kernel-contracts-migration-guards/12-06-SUMMARY.md`
- `.planning/phases/13-legacy-backfill-into-catalog-kernel/13-01-SUMMARY.md`
- `mibo-media-server/internal/catalog/service.go`
- `mibo-media-server/internal/catalog/projections.go`
- `mibo-media-server/internal/catalog/service_test.go`
- `mibo-media-server/internal/database/catalog_models.go`
- `mibo-media-server/internal/database/inventory_models.go`
- `mibo-media-server/internal/database/models.go`
- `mibo-media-server/internal/metadata/service.go`
- `mibo-media-server/internal/metadata/service_match.go`
- `mibo-media-server/internal/metadata/service_tmdb.go`
- `mibo-media-server/internal/metadata/service_test.go`
- `mibo-media-server/internal/httpapi/router.go`
- `mibo-media-server/internal/httpapi/handlers_media.go`
- `mibo-media-server/internal/library/service.go`
- `mibo-media-server/internal/library/service_libraries.go`
- `mibo-media-server/internal/worker/worker.go`

## Phase Goal And Requirement Mapping

**Goal:** match and refresh metadata from the series root, generating governed seasons and episodes from provider evidence.

**Requirements:**

- `META-01` — match a series catalog item to a provider identity and store provider payloads as source evidence.
- `META-02` — generate or update season and episode catalog items from matched provider data without duplicating local scanned episodes.
- `META-03` — canonicalize provider fields through `metadata_field_states` while preserving locked or manually edited fields.
- `META-04` — normalize images, people, tags, ratings, runtime, and air dates into catalog-owned tables and projections.

## Current Building Blocks

### Catalog layer already exists

- `catalog.Service.CreateItem` already creates rooted hierarchy rows and can create `series`, `season`, and `episode` items.
- `catalog.Service.RecordMetadataSource` already persists raw provider payload snapshots into `metadata_sources`.
- `catalog.Service.SetExternalID` already canonicalizes provider identity with uniqueness on `(provider, provider_type, external_id)`.
- `catalog.Service.ApplyField` already writes `metadata_field_states`, updates canonical `catalog_items` columns, and respects locked fields when `Force` is false.
- `catalog.RefreshItemProjection` / `RefreshLibraryProjection` already rebuild `item_rollups` and `catalog_search_documents`.

### Metadata layer already owns TMDB orchestration

- `metadata.Service` already resolves TMDB config from settings.
- `searchTMDB`, `findByExternalID`, `fetchDetail`, and `fetchTVSeason` already exist and are covered with `httptest`-based tests.
- Existing legacy `MatchItem` / `RefetchItem` flows prove the current orchestration pattern: metadata service fetches TMDB payloads and writes canonical state.

### Schema is additive enough for this phase

- `catalog_items`, `catalog_external_ids`, `metadata_sources`, `metadata_field_states`, `item_images`, `people`, `item_people`, `tags`, `item_tags`, `item_rollups`, and `catalog_search_documents` already exist.
- `media_assets`, `asset_items`, `inventory_files`, and `asset_files` already exist, so episode availability can be derived from asset/file links instead of legacy `MediaFile` state.
- No new dependency or new database system is required for Phase 15 planning.

## Architectural Responsibility Map

| Layer | Owns | Must not own |
|------|------|---------------|
| `internal/metadata` | TMDB search/fetch orchestration, confidence decisions, root-series refresh workflow | raw HTTP handler logic, direct projection document shaping |
| `internal/catalog` | catalog hierarchy upserts, source evidence writes, lock-respecting field writes, images/people/tags normalization | TMDB HTTP calls |
| `internal/library` / jobs / worker | enqueueing and dispatch only when later phases need runtime triggers | canonical metadata mutation details |
| `internal/httpapi` | later-phase transport only | metadata engine internals |

**Planning consequence:** Phase 15 should center on `internal/catalog` helpers plus `internal/metadata` orchestration. It should not spend scope on API/UI surfaces reserved for Phases 16 and 19.

## Recommended Implementation Strategy

1. **Add catalog-owned normalization helpers first.**
   - Introduce dedicated `internal/catalog` helpers for source-scoped image, person, and tag replacement.
   - Replace only rows contributed by the current provider source; preserve manual and local rows from other `source_id` values.

2. **Add hierarchy/evidence helpers before orchestration.**
   - Introduce catalog helper methods to resolve a series root, upsert governed seasons, upsert governed episodes, and reconcile episode availability from asset/file links.
   - Reuse existing scanned or backfilled episode rows by `(library_id, parent_id, index_number)` before creating new provider-only rows.

3. **Implement metadata refresh as a series-root workflow.**
   - Match the catalog series root to TMDB by existing series external ID first, then TMDB/IMDb/TVDB exact lookup where available, then title/year search.
   - Persist a raw `metadata_sources` row for the fetched series detail payload and each fetched season payload.
   - Canonicalize series, season, and episode fields through `catalog.ApplyField(... Force:false)` so locked/manual fields remain authoritative.

4. **Refresh projections after canonicalization, not during piecemeal writes.**
   - Once the root series refresh finishes, rebuild item projections so `item_rollups` and `catalog_search_documents` observe normalized images, tags, people, availability, runtime, rating, and air dates.
   - Keep direct projection SQL inside `catalog.Service`, not `metadata.Service`.

## Irreversible / High-Impact Decisions

### 1. Series root is the canonical metadata anchor

Use the `series` `catalog_item` as the owner of the primary TMDB identity (`provider_type="series"`) and the top-level source evidence. This matches the roadmap goal and avoids re-matching individual episodes as separate metadata roots.

### 2. Provider-generated missing rows need deterministic synthetic paths

When the provider reports a season or episode with no local file-backed path, create a stable synthetic catalog path:

- `catalog://series/{series_id}/season/{season_number}`
- `catalog://series/{series_id}/season/{season_number}/episode/{episode_number}`

This keeps provider-only rows idempotent without inventing fake storage paths, while preserving real scanned/backfilled paths on existing rows.

### 3. Availability comes from asset/file linkage plus air date

Use this precedence:

1. linked active asset + linked active inventory file → `available`
2. no local asset/file and provider air date is in the future → `unaired`
3. no local asset/file and air date is past/blank → `missing`

Do not derive Phase 15 availability from legacy `MediaFile` rows directly.

## Constraints And Pitfalls

### Do not collapse this phase into API work

- Existing HTTP routes are still legacy `media-items` routes.
- The roadmap explicitly places catalog API/governance surface work in Phase 16 and UI rebuild in Phase 19.
- Therefore Phase 15 should ship engine/service behavior, not public endpoint cutover.

### Do not hand-roll a second TMDB client

- Reuse `fetchDetail`, `fetchTVSeason`, `searchTMDB`, and `findByExternalID`.
- Add catalog-series orchestration on top of those helpers instead of creating a parallel provider client package.

### Lock preservation is already solved at the field level

- `catalog.ApplyField` already refuses to overwrite a locked field when `Force` is false.
- The phase should reuse that behavior rather than adding provider-specific lock code in `metadata.Service`.

### Existing people schema is intentionally minimal

- `people` currently stores name/sort_name only.
- `item_people` carries item role, character, sort order, and source linkage.
- This is sufficient for Phase 15’s normalization requirement because the requirement is table normalization, not person provider identity.

### Phase 13 backfill already established series-level identity direction

- Legacy backfill research and plans already treat TV external IDs as series-level identities.
- Phase 15 should continue that rule: series root owns the primary TMDB TV identity; seasons/episodes may receive child provider IDs when the provider payload supplies them.

## Must-Reuse Patterns

- **Catalog writes through helpers, not raw DB structs crossing boundaries.**
- **Provider payloads stay in `metadata_sources`; normalized read models stay in catalog tables.**
- **Projection refresh stays in `catalog.Service`.**
- **`httptest` TMDB servers remain the preferred test harness.**
- **Go tests should target focused package slices and stay under roughly one minute.**

## Proposed Plan Shape

Three execute plans is the smallest full-fidelity split that keeps tasks under the context budget:

1. `15-01` — catalog normalization helpers for images/people/tags.
2. `15-02` — catalog hierarchy/evidence helpers for idempotent series/season/episode governance.
3. `15-03` — metadata service series-first match/refresh orchestration plus projection refresh.

This split keeps the hardest constraints early (source-scoped normalization and hierarchy idempotence) so the orchestration plan can compose proven helpers instead of inventing behavior inline.

## Validation Architecture

### Test Infrastructure

| Property | Value |
|----------|-------|
| Framework | `go test` |
| Config file | none — Go package tests are self-contained |
| Quick run command | `cd mibo-media-server && go test ./internal/catalog ./internal/metadata -count=1` |
| Full suite command | `cd mibo-media-server && go test ./...` |
| Estimated runtime | ~45 seconds for quick run, longer for full suite |

### Sampling Strategy

- After each task commit: run the package-level targeted command for that task.
- After each plan wave: run `cd mibo-media-server && go test ./internal/catalog ./internal/metadata -count=1`.
- Before phase verification: run `cd mibo-media-server && go test ./...`.

### Phase-Specific Assertions To Preserve

- A matched series stores TMDB identity on the series root and raw provider payloads in `metadata_sources`.
- Provider refresh creates missing or unaired season/episode rows without duplicating locally scanned rows.
- Locked canonical fields survive provider refresh unchanged.
- Provider images, people, tags, rating, runtime, and air dates land in catalog-owned tables and projections.

## Research Conclusion

Phase 15 does **not** need new dependencies or a schema redesign. It needs disciplined reuse of the existing catalog kernel:

- put normalization and hierarchy mutation helpers in `internal/catalog`
- keep TMDB orchestration in `internal/metadata`
- preserve locks through existing `metadata_field_states`
- refresh projections only after a full root-series canonicalization pass

That is enough to satisfy `META-01` through `META-04` without bleeding into Phase 16 API work or Phase 19 UI work.
