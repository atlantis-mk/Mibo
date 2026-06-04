## Why

The directory-driven ingest pipeline now depends on `content_shape` plans for materialization, but some mature directory-shape heuristics still live in the older `scanrecognition` tree classifier. This split creates two sources of truth: old classification can recognize large movie collections, movie versions, multipart movies, NFO conflicts, and season trees, while the active materialization path can still fall back to review or leave files organizing.

This change migrates the useful shape-detection behavior into `content_shape`, makes it the single planner for ingest materialization, and removes the obsolete tree-classification path once parity is covered by tests.

## What Changes

- Add first-class `content_shape` evidence and planning rules for:
  - large multi-work directories with high title uniqueness and low episode continuity;
  - catalog/numbered-id collections, including JAV-style identifiers;
  - same-work movie version folders after normalizing version/edition/release noise;
  - multipart movie folders with continuous `part`/`cd`/`disc` sequences;
  - token-consensus classification for versions, multipart movies, and episode groups;
  - NFO/sidecar conflict protection that forces review instead of unsafe materialization;
  - primary-video-first planning that ignores trailers, samples, previews, featurettes, and extras when deciding the main folder shape;
  - parent series folders made only of season children.
- Expand `content_shape` evidence payloads, fingerprints, and tests so rescan behavior is stable and stale plans are invalidated when relevant file, signal, or sidecar evidence changes.
- Move any reusable parsing helpers that are still needed by `content_shape` into neutral helper packages or `library`-owned helpers so `content_shape` does not depend on the old tree classifier.
- Remove the legacy `scanrecognition` tree classifier and the `recognition_scan_adapter` materialization path after all required behavior is covered by `content_shape`.
- **BREAKING** for internal implementation only: directory-shape materialization no longer accepts the old tree classifier as an alternate authority.

## Capabilities

### New Capabilities
- `directory-shape-planning`: Defines how media-library scans classify directory shapes, handle ambiguous evidence, and produce materialization-ready plans from `content_shape`.

### Modified Capabilities
- None.

## Impact

- Backend packages:
  - `mibo-media-server/internal/library/content_shape_*`
  - `mibo-media-server/internal/library/scan_run.go`
  - `mibo-media-server/internal/library/recognition_*`
  - `mibo-media-server/internal/scanrecognition`
- Database-facing behavior:
  - `content_shape_profiles`, `content_shape_plans`, recognition candidates, materialized works/resources, and review states.
  - Classifier version bump required to invalidate stale plans.
- Tests:
  - New and migrated coverage in `internal/library`.
  - Retire or relocate old `internal/scanrecognition` classifier tests as `content_shape` tests.
- User-visible behavior:
  - Large mixed media folders should become organized works/resources when evidence is sufficient.
  - Ambiguous or contradictory sidecar evidence should show as review-required rather than remaining indefinitely “organizing”.
