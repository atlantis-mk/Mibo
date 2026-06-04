## Why

The current library ingest path mixes storage traversal, inventory writes, filename signal extraction, directory shape inference, recognition grouping, materialization, enrichment, and projection scheduling across scan and workflow code paths. This causes repeated parsing and inference, excessive database and workflow churn on large libraries, and makes it hard to skip unchanged directories reliably.

This change refactors ingest into a directory-driven incremental pipeline where each stage produces a durable intermediate result that downstream stages consume instead of recomputing.

## What Changes

- Introduce a directory snapshot layer that records provider traversal results, directory fingerprints, visible media counts, and changed/unchanged state.
- Introduce a staged inventory and signal pipeline that batches inventory updates and persists reusable file signals before directory shape planning.
- Promote directory content shape planning to a first-class stage that produces stable directory plans and file assignments for all downstream work.
- Introduce recognition units derived from directory shape plans instead of ad hoc file-parent grouping.
- Refactor materialization to consume recognition units and skip unchanged units by fingerprint.
- Refactor enrichment scheduling so metadata matching, media probing, and projection refresh consume materialization results and are coalesced per run.
- Preserve existing catalog, metadata, resource, recognition manifest, and content shape tables as compatibility surfaces while adding the minimal new state needed for staged execution.
- Keep existing API behavior for creating, scanning, refreshing, browsing, and playing libraries.

## Capabilities

### New Capabilities
- `directory-ingest-pipeline`: Defines the staged directory-driven ingest pipeline, durable stage outputs, change detection, and workflow ordering.
- `recognition-unit-materialization`: Defines recognition units derived from directory shape plans and their materialization semantics.
- `ingest-enrichment-coordination`: Defines how metadata matching, probing, and projection consume staged outputs and avoid duplicate work.

### Modified Capabilities

## Impact

- Affected backend packages: `internal/library`, `internal/recognition`, `internal/scanrecognition`, `internal/inventory`, `internal/workflow`, `internal/catalog`, `internal/metadata`, `internal/probe`, and `internal/database`.
- Database impact: new durable directory snapshot and recognition unit state, plus indexes for stage status, fingerprints, and lookup by library/root/directory.
- Workflow impact: new or refactored task types for snapshot, inventory sync, signal hydration, directory planning, recognition unit materialization, enrichment, and projection coalescing.
- Compatibility impact: existing library creation, manual scan, scheduled scan, targeted refresh, metadata match, probe, and projection APIs remain stable.
- Testing impact: new pipeline unit tests, migration tests, staged workflow tests, unchanged-directory skip tests, and end-to-end scan/materialization tests for movie, movie collection, movie versions, season, flat episode, attachment, and review-required directories.
