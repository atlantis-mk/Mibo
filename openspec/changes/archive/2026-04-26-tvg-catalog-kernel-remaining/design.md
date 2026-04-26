## Context

`mibo-media-server/internal/catalog` already contains the first slice of the new catalog kernel, including canonical DTO builders in `contracts.go`, projection refresh logic, and legacy backfill entrypoints. However, the running product still reads and writes primarily through legacy `MediaItem`, `MediaFile`, TMDB season helpers, and media-item-based progress/playback routes. The referenced implementation plan lays out the remaining cutover work across data migration, scanning, metadata governance, APIs, playback, frontend migration, and final cleanup.

The repo structure creates a clear split: backend work lives under `mibo-media-server/internal/*`, while the frontend still centers on `web/src/lib/mibo-api.ts`, `web/src/lib/mibo-query.ts`, and route behavior in `web/src/App.tsx`. The change must preserve incremental rollout and support empty databases, already-migrated databases, and legacy-populated databases without requiring a one-shot schema replacement.

## Goals / Non-Goals

**Goals:**
- Finish the catalog-kernel cutover using staged, idempotent migration and read/write switches rather than a flag day rewrite.
- Establish stable catalog contracts for list/detail/governance/playback flows around `series`, `season`, `episode`, `movie`, `extra`, `media_assets`, and `inventory_files`.
- Keep backfill, scan, metadata, API, playback, and frontend changes aligned so the product can move from legacy reads to catalog reads safely.
- Define the operational guardrails needed for rollout: migration state, projection rebuilds, consistency checks, and legacy cleanup gates.

**Non-Goals:**
- Replacing OpenList integration or changing storage-provider boundaries.
- Redesigning setup/auth flows unrelated to catalog cutover.
- Solving every provider-specific metadata edge case beyond exposing them for governance and safe retry.
- Delivering a complete UI redesign; this change is about semantic migration and governance capabilities.

## Decisions

### Decision: Treat the remaining work as one coordinated cutover with capability-specific specs
The proposal groups the work into five capabilities so specs can separate data, metadata, API/playback, frontend, and operational guarantees while still supporting one implementation change. This keeps the change large enough to describe the real migration while avoiding a single vague spec that mixes incompatible concerns.

Alternatives considered:
- One monolithic capability spec: rejected because it would blur backend, frontend, and operational requirements and be difficult to review.
- One change per phase: rejected for now because the user supplied a single remaining-work plan and asked for one new change with all artifacts generated together.

### Decision: Use catalog item and asset semantics as the only target contract for new behavior
Existing DTOs in `internal/catalog/contracts.go` already normalize around catalog items, selected images, field states, source evidence, and assets. The remaining design assumes all new ingest, metadata, API, playback, and UI behavior converges on these semantics rather than extending legacy `MediaItem` behavior again.

Alternatives considered:
- Keep dual first-class contracts for both catalog and legacy models: rejected because it prolongs divergence and makes the frontend migration harder.
- Map catalog behavior back into legacy shapes indefinitely: rejected because it blocks series-first governance and asset/version-aware playback.

### Decision: Roll out in four transition layers: backfill, cut new writes, cut reads, then remove legacy paths
The safest path is to backfill existing data first, then move scanners/metadata writes to catalog, then switch API and UI reads, and only then remove legacy writes and indexes. This matches the referenced plan and fits the repository’s current mixed state, where backfill and catalog DTO groundwork already exist but handlers and frontend types remain legacy.

Alternatives considered:
- Read-switch before new writes: rejected because new scans would keep creating divergence during rollout.
- Full cutover in a single release: rejected because empty-db, legacy-db, and partially migrated-db cases need different safeguards.

### Decision: Model TV governance at the series root and derive season/episode structure from provider evidence plus local assets
The metadata engine should match at the `series` level, persist source evidence and field locks there and on descendants, and then generate or update `season` and `episode` catalog items from provider detail. Local episode assets must merge into that hierarchy by season and episode number instead of remaining standalone legacy items.

Alternatives considered:
- Continue matching each episode independently: rejected because it duplicates work and does not support full-season completeness or missing-episode visibility.
- Generate only local episodes and ignore provider-only missing episodes: rejected because governance needs to expose gaps explicitly.

### Decision: Keep API DTOs explicit and decouple them from GORM models during the cutover
The new requirements should insist on DTO-based responses for catalog list/detail/governance/playback contracts. This is already the direction suggested by `internal/catalog/contracts.go` and prevents accidental leakage of legacy fields such as `series_title`, `source_path`, or `match_status` into the migrated frontend.

Alternatives considered:
- Serialize database models directly for speed: rejected because it ties API behavior to migration internals and complicates legacy cleanup.

### Decision: Treat playback as item-to-asset resolution, not media-item-to-file lookup
Playback must move from legacy `MediaItem`/`MediaFile` selection to catalog item, asset, asset-file, and inventory-file resolution. This allows multiple versions, multi-episode files, asset availability decisions, and progress keyed by item/asset instead of the legacy file-centric contract.

Alternatives considered:
- Preserve `media_file` as the playback primitive and alias assets to files: rejected because it cannot represent multi-link assets and keeps progress/playback coupled to legacy storage shape.

### Decision: Use migration state markers and consistency jobs as release gates
The existing backfill support indicates migration is operationally significant. The change should therefore require explicit state such as backfill completion, read-enable gates, cleanup completion, plus rebuild and consistency-check commands for projections and search documents before legacy cleanup.

Alternatives considered:
- Infer readiness from table contents alone: rejected because partial backfills and retry states are too ambiguous.

## Risks / Trade-offs

- [Scope spans backend and frontend] -> Mitigation: split requirements by capability and stage tasks by backend-first rollout order.
- [Existing mixed catalog/legacy code can drift during implementation] -> Mitigation: require explicit cutover markers and define when legacy writes become fallback-only.
- [SQLite/Postgres differences can break uniqueness and migration assumptions] -> Mitigation: specify behavioral outcomes, idempotent backfill, and consistency checks instead of relying only on implicit ORM behavior.
- [Provider season numbering and local file evidence may conflict] -> Mitigation: expose conflicts as governance states instead of forcing lossy auto-merges.
- [Frontend currently assumes every item has local-file fields] -> Mitigation: require catalog-aware empty states and asset-aware playback entry before read cutover.
- [Large cutover can be hard to roll back] -> Mitigation: keep rollback at the read-switch level; backfill and catalog writes are additive until cleanup gates are met.

## Migration Plan

1. Finalize catalog contracts, projection refresh hooks, indexes, and migration-state settings.
2. Complete and verify idempotent backfill from legacy tables into catalog, asset, inventory, metadata, and progress tables.
3. Switch scan and metadata jobs to create and update catalog structures while leaving legacy reads available.
4. Move list/detail/search/governance/playback APIs to catalog DTOs and item/asset semantics.
5. Migrate the frontend to catalog item types, season/episode views, governance panels, and asset-aware playback.
6. Enable catalog reads by default, run rebuild and consistency checks, then retire legacy write/read paths after validation.

Rollback strategy:
- Before legacy cleanup, rollback is done by disabling catalog reads and routing UI/API consumers back to legacy responses while preserving additive catalog data.
- After cleanup gates are crossed, rollback is limited to repair/rebuild operations rather than restoring legacy writes.

## Open Questions

- Which migration-state values already exist in the database schema, and which still need to be added?
- Whether the current partial backfill implementation already covers all legacy image/external-id/progress mappings or still needs requirement-level expansion.
- Whether legacy route compatibility should return bridged catalog payloads or explicit deprecation responses during the frontend migration window.
- How much of the governance UI can ship in the first cut without making the initial apply scope too large for one implementation pass.
