# Phase 14 Research — Scanner Writes Catalog Assets

**Phase:** 14 — Scanner Writes Catalog Assets  
**Requirements:** SCAN-01, SCAN-02, SCAN-03  
**Date:** 2026-04-25

## Research Goal

Answer: what must be true for `internal/library` scans to stop writing legacy
`MediaItem` / `MediaFile` rows and instead persist new file discoveries directly into the
catalog kernel as `catalog_items`, `inventory_files`, `media_assets`, `asset_files`, and
`asset_items` without losing rescan, probe, or delete semantics.

## No User Context Artifact

- No phase-specific `CONTEXT.md` exists for Phase 14.
- Planning therefore uses ROADMAP.md, REQUIREMENTS.md, the quick migration plan,
  Phase 12/13 outputs, and current codebase behavior as the authoritative sources.

## Current Codebase Facts

### Scanner still writes only legacy rows

- `mibo-media-server/internal/library/scan_run.go` walks storage objects, classifies each
  video file, then calls `upsertMediaItem` and `upsertMediaFile` from
  `internal/library/scan_upsert.go`.
- The scan loop still queues `match_media_item` and `probe_media_file`, so current refresh
  behavior depends on legacy tables and legacy job payloads.
- Missing-file cleanup still soft-deletes legacy `MediaItem` / `MediaFile` rows via
  `cleanupMissingItems*` and `cleanupMissingFiles*`.

### Catalog + inventory foundations already exist for direct writes

- `internal/catalog/service.go` already provides `CreateItem`, `SetExternalID`,
  `RecordMetadataSource`, and `ApplyField` with canonical item/governance constants.
- `internal/inventory/service.go` already provides idempotent `UpsertFile`,
  `CreateAsset`, `LinkAssetToItem`, and `LinkAssetToFile` helpers.
- `internal/inventory/service_test.go` already proves `asset_items` can represent one file
  linked to multiple episodes using `role="multi_episode_part"` and ordered
  `segment_index` values.

### Projection refresh already understands catalog rows

- Phase 12 added `catalog_refresh_item_projection` and
  `catalog_refresh_library_projection` queue/worker contracts.
- `internal/catalog/projections.go` rebuilds `item_rollups` and
  `catalog_search_documents` from `catalog_items`, so scanner write-cutover only needs to
  keep queueing catalog projection refreshes after changes.

### Probe still targets legacy `MediaFile`

- `internal/probe/service.go` only exposes `ProbeFile(ctx, mediaFileID uint)` and writes
  codec/runtime results back to `database.MediaFile` plus legacy `MediaItem.runtime_seconds`.
- `internal/library/enrichment.go` only exposes `QueueMediaFileProbe`, which enqueues the
  hard-coded `probe_media_file` job kind.
- Because Phase 14 removes legacy scan writes, it also needs an inventory-file probe path
  that updates `media_assets` and `media_streams` instead of legacy rows.

### Identity and rescan patterns are worth preserving, but not their legacy write target

- `internal/library/scan_identity_test.go` already proves stable identity evidence should
  move the persisted file row across rename/move without creating duplicates.
- `internal/library/scan_reconcile.go` shows the current fallback reconciliation logic is
  about preserving playback continuity across renamed or replaced files, not about legacy
  DTO shape. The continuity goal remains valuable even after the write target changes.

## Existing Patterns To Reuse

### 1. Queue + worker contracts stay in the existing jobs pipeline

- `internal/library/service.go` defines job kind constants.
- `internal/library/enrichment.go` queues probe jobs through `jobs.EnqueueUnique`.
- `internal/worker/worker.go` decodes typed payloads and dispatches to a service method.

**Implication:** Phase 14 should add a new inventory probe payload/job kind rather than
probing inline in the scan loop.

### 2. Idempotent persistence should stay at the catalog/inventory service boundary

- `inventory.Service.UpsertFile` already uses `OnConflict` on
  `(storage_provider, storage_path)`.
- `inventory.Service.LinkAssetToItem` and `LinkAssetToFile` already use conflict-based
  upserts.
- `catalog.Service.SetExternalID` already upserts provider identities.

**Implication:** scanner write helpers should compose these services rather than hand-write
SQL in `scan_run.go`.

### 3. Classification is already separated from persistence

- `internal/library/scan_classify.go` contains filename and path parsing rules.
- `internal/library/scan_classify_test.go` already proves season-folder and filename-based
  episode inference.

**Implication:** extend classification to carry richer scan artifacts (episode slots,
version hints, canonical hierarchy keys) before switching persistence.

### 4. Catalog projection refresh belongs after mutation, not during mutation

- `RunSyncLibrary` and `RunTargetedRefresh` already queue search reindex and catalog
  projection refresh only after scan traversal finishes.

**Implication:** Phase 14 should preserve that shape: write rows during traversal, then
queue library-scope projection refresh once per run.

## Recommended Phase-14 Implementation Shape

1. **Create a catalog-first scan writer boundary before changing the scan loop.**
   - Add helper(s) in `internal/library` that accept a normalized scan artifact and write
     catalog + inventory rows.
   - Keep `scan_run.go` orchestration thin; persistence belongs in dedicated helpers.

2. **Switch scan traversal to call the new writer and stop creating legacy rows.**
   - `RunSyncLibrary` and `RunTargetedRefresh` should no longer call
     `upsertMediaItem`, `upsertMediaFile`, `QueueMediaItemMatch`, or
     `QueueMediaFileProbe`.
   - New scans should create pending-governance catalog rows plus local evidence.

3. **Make episode hierarchy keys canonical and file-independent.**
   - Episode catalog rows cannot key off raw file path because one logical episode can have
     multiple files (versions) and one file can represent multiple episodes.
   - Use canonical synthetic hierarchy paths for series/season/episode rows; keep
     `inventory_files.storage_path` as the file-specific boundary.

4. **Model multi-episode and multi-version as asset-link semantics, not duplicate items.**
   - One multi-episode file should create one `media_asset` linked to multiple episode
     items via `asset_items(role="multi_episode_part")`.
   - A second file for the same episode slot should create another `media_asset` linked to
     the same episode item via `asset_items(role="version")` instead of creating a second
     episode item.

5. **Replace legacy probe writes with inventory-file probe writes.**
   - Probe jobs should target `inventory_files`, rebuild `media_streams`, and update linked
     `media_assets` probe/runtime summary fields.
   - Do not write new probe data into legacy `MediaFile` / `MediaItem` rows.

6. **Delete semantics must update availability, not remove governed catalog metadata.**
   - When a previously seen file disappears, keep `catalog_items`, `metadata_sources`, and
     existing asset/link rows.
   - Mark files/assets unavailable or missing, then recompute leaf item availability from
     remaining available assets.

## Required Mapping Decisions

### Canonical hierarchy keys

Recommended keying strategy:

- **Movie item path:** keep the scan-classified `SourcePath` as the item path for single-file
  movies in this phase.
- **Series item path:** canonical series directory root under the library, or the nearest
  non-season ancestor when scanning an episode file.
- **Season item path:** `{series_path}/season-{season:02d}`.
- **Episode item path:** `{series_path}/season-{season:02d}/episode-{episode:04d}`.

This makes episode items stable across multi-version files and rescans.

### Local evidence contract

Every new scanner-created catalog item should get a `metadata_sources` row with:

- `source_type="local_file"`
- `source_name="scanner"`
- `payload_json` containing only compact allowlisted fields such as:
  - `storage_path`
  - `stable_identity_key`
  - `provider_name`
  - `hashes_json`
  - `detected_title`
  - `series_title`
  - `season_number`
  - `episode_numbers`

### Governance + availability defaults

- New scanner-created movie/series/season/episode rows should default to
  `governance_status="pending"`.
- Leaf items with at least one available linked asset should default to
  `availability_status="available"`.
- Missing-file cleanup should move leaf items to `availability_status="missing"` when no
  available linked asset remains.

### Multi-episode linking rules

- Parse contiguous filename ranges such as `S01E01-E02` and `S01E01E02` into explicit
  episode slot lists.
- Create one `media_asset` for the scanned file.
- Link that asset to each affected episode using
  `asset_items(role="multi_episode_part")` with ascending `segment_index` starting at 1.

### Multi-version rules

- If another available asset already links to the same episode slot but points at a different
  `inventory_file`, create a new `media_asset` with `asset_type="version"`.
- Link version assets with `asset_items(role="version")`.
- Keep the first discovered playable asset as the `primary` link unless later phases add a
  richer version-selection policy.

### Delete / reappearance rules

- Missing scans should mark `inventory_files.status="missing"` and linked
  `media_assets.status="missing"`; they should not soft-delete `catalog_items`.
- If a file reappears with the same `stable_identity_key`, reuse the existing
  `inventory_file` row and its linked asset IDs while updating the current `storage_path`.

## Main Risks

1. **Hierarchy duplication risk** — episode items keyed by raw file path would duplicate when a
   second version appears.
2. **Filename spoofing / ambiguity risk** — scanner-generated hierarchy must stay bounded to
   library-scoped normalized paths and explicit season/episode parsing rules.
3. **Probe regression risk** — removing legacy `MediaFile` writes without replacing probe input
   would strand runtime/stream metadata.
4. **Availability drift risk** — deleting rows instead of marking them missing would destroy the
   governed metadata that later API and UI phases depend on.
5. **Transition visibility risk** — new scans will populate only catalog tables until later API
   cutover phases switch reads, so Phase 14 must keep data correct even if current UI still
   reads legacy models.

## Validation Architecture

### Fast feedback

- `go test ./internal/library -run 'Test(ScanCatalogWriter|ClassifyMediaFile|RunSyncLibrary.*Catalog)' -count=1`
- `go test ./internal/probe -run 'TestProbeInventoryFile' -count=1`

### Integration feedback

- `go test ./internal/worker -run 'TestRunOnce.*ProbeInventoryFile|TestRunSyncLibrary.*Catalog' -count=1`

### Full phase regression

- `go test ./internal/library ./internal/inventory ./internal/probe ./internal/worker -count=1`

### Required proof points

- Scan writer tests prove movies and episodes create catalog/inventory rows without new legacy
  `MediaItem` / `MediaFile` inserts.
- Classification tests prove multi-episode ranges and existing single-episode patterns resolve
  deterministically.
- Probe tests prove inventory-file jobs create/update `media_streams` and linked
  `media_assets`.
- Rescan/delete tests prove missing files only change availability/status and never delete
  governed catalog metadata.

## Architectural Responsibility Map

| Concern | Correct layer | Why |
|---------|---------------|-----|
| Filename/path parsing | `internal/library/scan_classify.go` | parsing rules already live here and should stay separate from persistence |
| Catalog-first scan persistence | `internal/library` using `internal/catalog` + `internal/inventory` services | scanning owns orchestration, catalog/inventory own row semantics |
| Inventory-file probe execution | `internal/probe` + `internal/worker` | ffprobe integration is already isolated here |
| Projection refresh | `internal/library` queue helper + `internal/catalog` worker target | existing phase-12 projection path already owns rollup/search rebuilds |
| Missing-file availability updates | `internal/library` scan cleanup helpers | scan traversal owns knowledge of what was seen vs missing |

## Planning Implications

- Phase 14 is backend-only.
- The safest breakdown is four sequential plans: scan-writer foundation, direct scan write
  integration, inventory-file probe migration, then delete/availability semantics.
- Later phases should treat Phase 14 as the point where the catalog kernel becomes the only
  writer for newly scanned content, even though legacy reads remain until API/playback/frontend
  cutover.
