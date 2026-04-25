# Phase 13 Research — Legacy Backfill Into Catalog Kernel

**Phase:** 13 — Legacy Backfill Into Catalog Kernel  
**Requirements:** MIGR-01, MIGR-02, MIGR-03  
**Date:** 2026-04-25

## Research Goal

Answer: what must be true to backfill existing legacy `MediaItem` / `MediaFile` / `PlaybackProgress`
data into the catalog kernel safely, repeatedly, and observably before later read and write
cutover phases depend on it.

## No User Context Artifact

- No phase-specific `CONTEXT.md` exists for Phase 13.
- Planning therefore uses ROADMAP.md, REQUIREMENTS.md, Phase 12 outputs, quick-task migration
  research, and current codebase behavior as the authoritative sources.

## Current Codebase Facts

### Catalog and inventory foundations already exist

- `mibo-media-server/internal/database/catalog_models.go` already defines the target logical
  kernel tables: `CatalogItem`, `CatalogExternalID`, `MetadataSource`, `MetadataFieldState`,
  `ItemImage`, `UserItemData`, `ItemRollup`, and `CatalogSearchDocument`.
- `mibo-media-server/internal/database/inventory_models.go` already defines the target playback
  inventory tables: `MediaAsset`, `AssetItem`, `InventoryFile`, `AssetFile`, and `MediaStream`.
- `mibo-media-server/internal/catalog/service.go` already exposes item creation, provider ID
  persistence, metadata evidence persistence, and field-state application helpers.
- `mibo-media-server/internal/inventory/service.go` already exposes idempotent upsert and link
  helpers for `InventoryFile`, `AssetItem`, and `AssetFile`.

### Legacy runtime is still authoritative

- Scans still write legacy `MediaItem` / `MediaFile` rows in `internal/library/scan_run.go` and
  `internal/library/scan_upsert.go`.
- Discovery, browse grouping, progress, playback, metadata matching, and search still read legacy
  rows and legacy projections (`PlaybackProgress`, `SearchDocument`) in packages such as
  `internal/library/query_browse.go`, `internal/progress/service.go`, and `internal/metadata`.
- Legacy show browse semantics are still synthesized at read time by grouping episode rows using
  `ExternalID` or `SeriesTitle` in `groupShowBrowseCandidates`.

### Migration observability groundwork exists but backfill execution does not

- Phase 12 added durable migration settings (`catalog_backfill_completed_at`,
  `catalog_read_enabled`, `legacy_cleanup_completed_at`) in
  `internal/settings/catalog_migration.go`.
- Phase 12 also added catalog projection jobs and worker dispatch.
- There is **no** current legacy-to-catalog backfill service, job kind, run report schema, or
  operator endpoint to trigger and inspect backfill.

### Important legacy semantics to preserve or reinterpret

- Legacy movie rows carry enough direct fields to map one-to-one into `catalog_items(type=movie)`.
- Legacy episode rows only imply series/season hierarchy via `SeriesTitle`, `SeasonNumber`,
  `EpisodeNumber`, and sometimes series-level `ExternalID` values such as `tv:777`.
- Legacy `MediaItem.ExternalID` is overloaded. For episodes it often stores a series-level TMDB TV
  ID, not an episode-level provider ID.
- Legacy `MediaFile` rows already contain the stable identity and probe evidence Phase 13 needs to
  seed `InventoryFile`, `MediaAsset`, `AssetFile`, and future playback selection.
- Legacy `PlaybackProgress` is keyed by `(user_id, media_item_id)` with optional `media_file_id`;
  the target table `UserItemData` is keyed by `(user_id, item_id, asset_id)`.

## Existing Patterns To Reuse

### Queueable worker contracts

- `internal/library/service.go` and `internal/library/service_libraries.go` define the current job
  kind + enqueue helper pattern.
- `internal/worker/worker.go` decodes typed payloads and delegates to a service method.
- `internal/worker/worker_catalog_test.go` proves end-to-end worker dispatch with real sqlite.

**Implication:** Phase 13 should trigger backfill through a durable queued job instead of running
large migration work inline inside an HTTP handler.

### Idempotent persistence in service layers

- `inventory.Service.UpsertFile` uses `clause.OnConflict` on `(storage_provider, storage_path)`.
- `inventory.Service.LinkAssetToItem` and `LinkAssetToFile` also use conflict-based upserts.
- `catalog.Service.SetExternalID` uses `clause.OnConflict` on `(provider, provider_type,
  external_id)`.

**Implication:** Phase 13 should preserve idempotency by reusing service helpers where the existing
unique keys already express the target identity.

### Legacy browse grouping shows where the danger is

- `internal/library/query_browse.go` groups episode rows into virtual shows using
  `ExternalID` first, then `SeriesTitle`.
- This is useful as a fallback heuristic, but it is not sufficient for silent canonical writes when
  data is ambiguous.

**Implication:** Phase 13 should use conservative series grouping precedence and write explicit
report entries for ambiguous or conflicting cases rather than silently merging them.

### Progress upsert pattern

- `internal/progress/service.go` validates the media/file relationship, computes completion, and
  upserts by the user/content key.

**Implication:** Phase 13 should mirror this with catalog IDs, mapping each legacy progress row onto
`UserItemData` using the resolved catalog item and asset.

## Recommended Phase-13 Implementation Shape

1. **Create durable backfill run/report persistence first.**
   - Add explicit run rows and per-entry report rows so every migration attempt is auditable.
   - Report entry kinds must include `success`, `skipped`, `conflict`, `orphan_file`, and
     `duplicate_episode_candidate` to satisfy MIGR-02 directly.

2. **Trigger backfill as a background job.**
   - Add one job kind such as `catalog_backfill_legacy`.
   - Authenticated HTTP should create a run row, enqueue the job, and return `202 Accepted` with a
     run identifier.
   - Separate GET endpoints should list runs and show a full run report.

3. **Split backfill logic by slice, not by layer.**
   - One movie slice: map legacy movie rows into catalog items + assets + files + evidence.
   - One series slice: group legacy episode rows into series/season/episode hierarchy, then attach
     assets/files/evidence.
   - One progress/finalization slice: map legacy progress, refresh projections, and finalize the
     migration state timestamp.

4. **Keep idempotency explicit.**
   - Re-running the backfill must reuse existing catalog rows and inventory links.
   - Report rows are per-run and can duplicate across runs; domain rows must not.
   - Later runs should record `success` again for reused mappings rather than forcing synthetic
     "already migrated" failure semantics.

5. **Do not auto-enable catalog reads.**
   - On successful backfill, update `catalog_backfill_completed_at`.
   - Leave `catalog_read_enabled` unchanged until Phase 16 cutover deliberately enables it.

## Mapping Decisions Required By Phase 13

### Run/report schema

Recommended persistence:

- `CatalogMigrationRun`
  - scope kind: `all` or `library`
  - optional `library_id`
  - `status`: `queued`, `running`, `completed`, `failed`
  - `triggered_by_user_id`
  - `started_at`, `finished_at`, `fatal_error`
  - aggregate counters for each report entry category
- `CatalogMigrationEntry`
  - `run_id`
  - `entry_type`: `success`, `skipped`, `conflict`, `orphan_file`,
    `duplicate_episode_candidate`
  - optional legacy/catalog foreign IDs (`legacy_media_item_id`, `legacy_media_file_id`,
    `catalog_item_id`, `asset_id`, `inventory_file_id`)
  - `library_id`, `storage_path`, `title`, `message`, `details_json`

### Movie mapping

- Legacy `MediaItem(type=movie)` → `catalog_items(type=movie)`.
- Canonical lookup key should be the legacy library + source path + item type.
- Legacy `MediaFile` rows attached to the movie map to:
  - `inventory_files`
  - one `media_assets(asset_type=main)` per playable legacy row
  - `asset_files(role=source, part_index=0)`
  - `asset_items(role=primary, segment_index=0)`
- Legacy artwork URLs map to selected `item_images` with types `poster`, `backdrop`, and `logo`.
- Legacy `metadata_provider`, `external_id`, and `metadata_confidence` should map to both
  `catalog_external_ids` and one provider `metadata_sources` record when present.

### Series/season/episode mapping

Recommended grouping precedence for legacy episode rows:

1. If `metadata_provider` + `external_id` are present and the external ID prefix is a series-level
   TV identity (`tv`, `series`), group by `(library_id, provider, provider_type=series,
   external_id)`.
2. Otherwise fallback to `(library_id, normalized series_title, year)`.
3. If `series_title` is blank and no external ID is available, record a `conflict` entry and do not
   silently invent a series root.

Within a grouped series:

- Series row → `catalog_items(type=series)` using the common directory prefix of member episode
  paths (fallback: first episode directory).
- Season rows → `catalog_items(type=season)` grouped by season number under the series.
- Episode rows → one canonical `catalog_items(type=episode)` per logical season/episode slot.
- If multiple legacy rows claim the same season/episode slot, record
  `duplicate_episode_candidate` entries but link every playable legacy row to assets on the single
  canonical episode row.

### File, asset, and progress mapping

- `MediaFile.StoragePath`, identity, hashes, probe data, size, and timestamps should seed
  `InventoryFile`.
- Phase 13 should carry legacy probe/runtime evidence forward to `MediaAsset` so later playback
  selection has a real baseline.
- `PlaybackProgress` should map to `UserItemData` using the resolved catalog item and the linked
  asset when `media_file_id` can be resolved.

### Reporting rules

- `success`: a legacy row was mapped or reused successfully.
- `skipped`: the row was intentionally not migrated because a required input was missing but the row
  is not structurally contradictory (example: unplayable legacy row without active file).
- `conflict`: the row could not be mapped deterministically.
- `orphan_file`: a legacy `MediaFile` is active but has no active owning `MediaItem`.
- `duplicate_episode_candidate`: two or more legacy rows claim the same logical episode slot.

## Main Risks

1. **Ambiguous series identity risk** — `SeriesTitle` alone is not safe enough for silent merge in
   all cases.
2. **Overloaded provider ID risk** — legacy `ExternalID` semantics differ between movies and TV;
   blindly copying them to the wrong catalog level will create bad future matches.
3. **Idempotency drift risk** — there are no all-encompassing natural keys on `CatalogItem`, so the
   service must query and reuse canonical rows deliberately.
4. **Projection freshness risk** — backfill is not useful unless `item_rollups` and
   `catalog_search_documents` are refreshed for touched libraries before reporting completion.
5. **Long-running job risk** — large libraries should not hold an HTTP request open; the report must
   survive process restarts and expose partial results.

## Validation Architecture

### Fast feedback

- `go test ./internal/catalog -run 'TestLegacyBackfill(Report|Movies|Series|Progress)' -count=1`

### Integration feedback

- `go test ./internal/worker -run 'TestRunOnce.*CatalogBackfill' -count=1`
- `go test ./internal/httpapi -run 'TestCatalogMigrationBackfill' -count=1`

### Full phase regression

- `go test ./internal/catalog ./internal/database ./internal/httpapi ./internal/worker -count=1`

### Required proof points

- Run/report rows persist and aggregate counts by entry type.
- Authenticated operators can enqueue a backfill run and inspect a completed run report.
- Movie backfill creates catalog items, inventory files, assets, asset links, images, external IDs,
  and provider evidence without duplication across reruns.
- Series backfill creates one hierarchy root with reusable season/episode rows, reports duplicate
  slot candidates, and reports orphan files.
- Progress rows migrate to `user_item_data` using resolved catalog item + asset IDs.
- Successful runs refresh projections and update `catalog_backfill_completed_at` without flipping
  `catalog_read_enabled`.

## Architectural Responsibility Map

| Concern | Correct layer | Why |
|---------|---------------|-----|
| Backfill run/report persistence | `internal/database` + `internal/catalog` | report schema is durable data, orchestration belongs to catalog domain |
| Legacy row → catalog/item/asset mapping | `internal/catalog` + `internal/inventory` | this is kernel migration logic, not HTTP or router work |
| Operator trigger and report read APIs | `internal/httpapi` | authenticated request decoding and response mapping belong here |
| Background execution | `internal/worker` + `internal/jobs` | long-running migration must reuse the existing queue model |
| Migration state timestamp update | `internal/settings` via worker | durable cutover flags already live in typed settings |

## Planning Implications

- Phase 13 is backend-only.
- It should be planned as multiple small execution plans: report schema/contracts, operator
  entrypoints, movie slice, series slice, then progress/finalization.
- Plans can run in parallel after the initial contract/report foundation as long as files do not
  overlap.
