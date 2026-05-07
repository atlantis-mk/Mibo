## Why

Large source-first video directories with hundreds or thousands of episodes are currently classified from the file outward, causing repeated filename parsing, directory summary work, and batch-level reclassification even when the directory's content shape is obvious. Mibo needs a shape-first scanner that can recognize directory content forms once, reuse that decision across batches and rescans, and then materialize files from a stable assignment plan.

## What Changes

- Introduce a content shape scanner that profiles each directory before per-file catalog projection.
- Add persistent content shape indexes and plan rules so unchanged directories and small deltas can reuse prior classification decisions.
- Compile directory-level plans for high-confidence episode packs, season folders, flat episode folders, movie folders, movie version folders, movie collections, and ambiguous/review groups.
- Materialize files from directory plan assignments instead of repeatedly running full movie-vs-episode classification for each file in high-confidence groups.
- Preserve inventory-first behavior: scans still record storage facts before final catalog semantics, and low-confidence outcomes remain reviewable instead of silently polluting catalog rows.
- Keep implementation phased, but require the complete architecture to be delivered: temporary plan fast path, persistent shape index, incremental reuse, review handling, and regression coverage.

## Capabilities

### New Capabilities
- `content-shape-index`: Persistent and incremental directory content shape profiling, plan compilation, and plan assignment reuse.

### Modified Capabilities
- `media-graph-scanner`: Scanner grouping changes from file-first classification with directory hints to directory-shape-first plan compilation with per-file assignment materialization.
- `fast-video-classification`: Fast classification must reuse shape plans and avoid repeated full file classification in high-confidence directory groups.
- `filename-signal-classification`: Filename signal extraction must support cheap one-pass token profiles used by directory shape profiling.
- `source-first-auto-classification`: Source-first scanning must keep automatic semantics while surfacing directory-level low-confidence and review-required decisions.

## Impact

- Backend scanner and materialization paths under `mibo-media-server/internal/library/`, especially directory snapshots, filename signal extraction, catalog materialization batches, and classification decision persistence.
- Database schema for directory shape profiles, plan rules, fingerprints, classifier versions, confidence, review state, and assignment metadata.
- Catalog projection behavior for series, season, episode, movie, movie-version, movie-collection, attachment, and ambiguous directory groups.
- Existing scan exclusion, sidecar, artwork preselection, metadata enrichment, inventory probing, missing cleanup, and projection refresh jobs must continue to work after plan-based materialization.
- Tests must cover large mixed-naming episode packs, Season 1/Season 2 directories, absolute episode ranges, movie collections, movie versions, attachments, incremental additions, directory changes, and classifier version invalidation.
