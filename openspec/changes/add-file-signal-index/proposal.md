## Why

Mibo already separates inventory facts from catalog projection and has a content-shape planner, but filename signal extraction is still mostly runtime-local and can be repeated across large scans, materialization batches, and rescans. A persistent file signal index will make first-scan grouping more explicit, speed up unchanged rescans, and let the existing content-shape scanner reuse durable per-file evidence instead of rebuilding token profiles repeatedly.

## What Changes

- Add a persistent inventory file signal index keyed by storage provider, storage path, classifier version, and file fingerprint.
- Reuse existing filename signal extraction logic instead of introducing a parallel parser.
- Feed content-shape directory profiles from persisted file signals when available, with runtime extraction retained only as fallback during migration or cache misses.
- Preserve existing `inventory_files`, `content_shape_profiles`, `content_shape_plans`, and `content_shape_assignments` semantics while making their inputs more durable and reusable.
- Tighten low-confidence content-shape behavior so uncertain plans preserve review evidence and avoid silently committing unrelated movie or episode catalog pollution.
- Clean up redundant repeated parsing paths after the signal index covers scan and materialization workflows.

## Capabilities

### New Capabilities
- `inventory-file-signal-index`: Persistent per-file filename/path signal indexing and reuse for scanner profiling, planning, and incremental rescans.

### Modified Capabilities
- `filename-signal-classification`: Filename signal extraction results must be reusable through a persisted index when file facts and classifier version are unchanged.
- `media-graph-scanner`: Scanner grouping must use durable file signal evidence before catalog writes and must avoid low-confidence semantic pollution.
- `fast-video-classification`: Fast classification must reuse indexed signals and avoid repeated parsing in high-cardinality directories.
- `source-first-auto-classification`: Automatic classification must continue without media library type selection while preserving low-confidence outcomes for review.

## Impact

- Backend database models and migrations for a new inventory file signal table.
- Scanner and materialization paths under `mibo-media-server/internal/library/`, especially filename token profile extraction, content-shape profile construction, content-shape plan reuse, and assignment materialization.
- Inventory service integration where file facts are upserted and looked up.
- Existing scan exclusions, subtitle binding, artwork preselection, probe scheduling, metadata matching, missing cleanup, and projection refresh must remain compatible.
- Tests must cover signal reuse, classifier version invalidation, file fingerprint invalidation, large directory profile construction, and low-confidence review behavior.
