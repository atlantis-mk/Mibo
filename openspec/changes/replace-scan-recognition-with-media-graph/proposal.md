## Why

The current scan path still makes movie-vs-series decisions too early from file-level signals and then tries to repair them with directory, sidecar, recognition, and projection follow-up code. This has produced broken TV ingestion in real libraries: episode candidates exist, but series/season metadata and resource links do not reliably materialize.

Now that Mibo is development-reset oriented and already resource-first, the faster and more accurate path is to replace the overlapping scan recognition paths with one directory-first media graph pipeline.

## What Changes

- **BREAKING** Replace scan-time single-file recognition with a media graph pipeline that collects inventory facts, builds local work groups, classifies groups, resolves identities, and materializes catalog metadata only from accepted graph decisions.
- Add durable media graph records for directories, files, work groups, evidence, decisions, and resource relationships.
- Classify movie, movie collection, movie version, series, season, episode run, multi-episode resource, and supplemental video roles from group evidence instead of isolated filename guesses.
- Treat filename parsing, sidecars, content-shape planning, path-tree grouping, hashes, and external IDs as evidence providers only.
- Materialize TV content as `series -> season -> episode -> resource` and movie content as `movie -> resource/version/supplemental` from graph decisions.
- Limit metadata matching to supported work-level targets: `movie` and `series`.
- Mark ambiguous groups for review instead of silently creating incorrect movie or episode metadata.
- Remove or rewrite old final-decision code paths that directly create/link movie or episode metadata from scan helpers, content-shape/path-tree assignments, or sibling-matching fallbacks.

## Capabilities

### New Capabilities

- `media-graph-recognition`: Defines the replacement scanner recognition pipeline, graph group types, decision rules, materialization behavior, review handling, and cleanup expectations for old recognition paths.

### Modified Capabilities

- `media-graph-scanner`: Tighten requirements so scanner output is a durable media graph and final catalog writes come only from graph decisions.
- `mixed-content-library`: Replace library-type and per-file automatic classification behavior with group-first movie-vs-series classification.
- `metadata-operation-pipeline`: Restrict automatic metadata matching to graph-materialized `movie` and `series` targets and prevent episode/season match jobs from failing workflows.
- `tv-hierarchy-metadata-completion`: Require TV hierarchy metadata to be created from graph decisions, not from orphan episode candidates.

## Impact

- Backend scan and recognition code under `mibo-media-server/internal/library`, `mibo-media-server/internal/recognition`, `mibo-media-server/internal/catalog`, and related workflow/materialization paths.
- Database schema for persisted media graph nodes, edges, evidence, group decisions, and review state.
- Workflow behavior for scan, resolve, materialize, projection refresh, and metadata match scheduling.
- Tests and fixtures for movie folders, movie versions, movie collections, TV season folders, flat TV folders, multi-episode files, extras, ambiguous folders, and cleanup of old recognition behavior.
- Local development data must be reset or migrated because old orphan episode/movie fallback rows are not compatible with the replacement decision model.
