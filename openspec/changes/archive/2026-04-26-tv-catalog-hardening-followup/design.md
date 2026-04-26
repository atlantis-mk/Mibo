## Context

The existing TV planning document describes a greenfield direction where shows, seasons, and episodes are governed as first-class entities with separate metadata evidence, asset linkage, and user/query projections. The current codebase is no longer at the old `MediaItem`-only starting point: it already has `catalog_items`, `catalog_external_ids`, `metadata_sources`, `metadata_field_states`, `item_images`, `media_assets`, `asset_items`, `inventory_files`, `user_item_data`, and catalog-backed HTTP routes.

That means the right next step is not a schema restart. The right next step is to close the behavioral gap between:

1. the TV-first catalog model described in `.planning/TV-SERIES-METADATA-GOVERNANCE-PLAN.md`
2. the partially completed incremental cutover represented by `openspec/changes/tvg-catalog-kernel-remaining`

The audit result is:

- scan write cutover is implemented
- catalog playback and progress are implemented
- catalog governance pages exist
- many frontend screens still depend on compatibility adapters
- some TV hierarchy and governance behaviors remain incomplete
- final operational hardening and legacy retirement are still pending

## Goals / Non-Goals

**Goals**

- Make TV hierarchy operations first-class over the existing catalog graph.
- Make catalog metadata governance complete at the descendant level for seasons and episodes.
- Remove the remaining catalog-to-legacy presentation shims from primary frontend flows.
- Strengthen operational safety so catalog reads can become the stable default and legacy routes can be bounded or removed.

**Non-Goals**

- Replacing the current catalog schema with an all-new greenfield model.
- Reworking storage-provider integration or OpenList boundaries.
- Implementing every future TV feature from the planning document, such as full alternate ordering support, broad NFO support, or every local artwork convention.

## Decisions

### Decision: Use the current catalog kernel as the permanent base

The existing `catalog_items` plus asset-link model already captures the core separation described in the TV plan: logical items, metadata evidence, playable assets, and user/query projections. This follow-up will converge behavior on that model instead of introducing a second replacement schema.

Alternatives considered:

- Start a new greenfield item graph again: rejected because it would duplicate already-landed work and create another migration wave.
- Continue indefinitely with compatibility wrappers: rejected because it preserves ambiguity and keeps TV behavior half-migrated.

### Decision: Treat remaining work as four convergence tracks

The comparison shows four distinct unfinished areas that can be planned and implemented together:

1. TV hierarchy and metadata completeness
2. frontend contract cleanup
3. governance action completeness
4. cutover hardening and legacy retirement

This keeps the change focused on real gaps rather than repeating already-complete tasks.

### Decision: Descendant season and episode metadata must be governable, not only derivable

Series-root matching is the correct authority point for TV provider matching, but the resulting season and episode nodes must carry enough durable identity and evidence to support governance, conflict review, and API responses without falling back to legacy assumptions.

This means season and episode rows should expose stable provider identity, evidence snapshots, artwork candidates/selection, and hierarchy-derived availability semantics where applicable.

### Decision: Frontend catalog migration is only complete when primary views stop depending on legacy presentation shapes

Catalog-backed pages that immediately convert results back into `MediaItem` or `MediaItemDetail` compatibility objects are still semantically coupled to the old model. The remaining work should treat those adapters as temporary compatibility debt and remove them from search, library, detail, and governance-entry flows.

### Decision: Legacy retirement is gated by runtime safety, not by task completion alone

The existing task list marks most cutover steps complete, but the system should not treat legacy cleanup as done until:

- catalog reads are stable by default
- backend and frontend verification suites pass together
- projection rebuild and consistency operations cover the final read model
- legacy endpoints have explicit retirement or fallback behavior
- database-level safety constraints are strong enough for production operation

## Workstreams

### 1. TV Hierarchy And Metadata Completeness

- audit outcome for the current repository is now explicit: descendant rows already exist, but the remaining gaps are scanner reuse of provider-created descendants, descendant artwork provenance linkage, curated descendant evidence summaries, and public catalog-native child/missing/next-up reads
- complete season/episode provider identity and evidence behavior
- ensure season/episode artwork candidates and selected images are durable and queryable
- add hierarchy-native read APIs where current flows still depend on compatibility or implicit grouping
- add TV convenience endpoints for missing and next-up behaviors where they are still absent

### 2. Frontend Contract Cleanup

- replace compatibility mapping in search, library, media detail, and governance entry points with catalog-native view models
- keep route-level and query-level behavior rooted in catalog DTOs
- make hierarchy, availability, and asset state visible without placeholder legacy fields

### 3. Governance Action Completeness

- extend governance from informational asset display to actionable asset-link correction and hierarchy review
- surface mismatches between provider hierarchy and local asset linkage in a way users can resolve
- preserve evidence and lock semantics while enabling these corrections

### 4. Cutover Hardening And Legacy Retirement

- strengthen foreign keys and uniqueness where the current schema is still permissive
- complete rebuild/consistency coverage for availability, search, and rollups
- enable catalog reads by default once validation gates pass
- bound or remove remaining legacy read/write paths and document recovery behavior

## Risks / Trade-offs

- Removing frontend compatibility adapters may expose places where catalog DTOs still lack a few convenience fields.
- Tightening database constraints may surface latent duplicate or malformed catalog rows in test fixtures or migrated data.
- Adding governance correction actions increases UX scope, so the implementation should stay focused on the minimum useful repair flows.
- Legacy endpoint retirement must remain reversible until catalog-read-default behavior is validated end-to-end.

## Migration / Rollout Plan

1. Finish descendant metadata completeness and hierarchy-native APIs.
2. Move primary frontend surfaces to catalog-native contracts.
3. Add governance actions for asset-link and hierarchy correction.
4. Tighten constraints, rebuild coverage, and validation gates.
5. Enable catalog reads by default.
6. Retire or isolate the remaining legacy routes and document operational recovery steps.

Rollback strategy:

- before legacy cleanup, rollback remains the catalog-read switch plus compatibility endpoints
- after cleanup gates pass, rollback should rely on rebuild/repair operations instead of reintroducing legacy write paths
