## Context

Mibo has already moved to source-first libraries, durable inventory files, resource-first catalog projection, and metadata operations rooted on `movie` and `series`. The remaining scanner problem is architectural: several modules still make final movie-vs-TV decisions from partial context and then pass those decisions downstream as if they were facts.

The current behavior fails in real TV libraries because file-level recognition can emit many episode candidates while series and season candidates remain provisional or unmapped. The downstream catalog then sees orphan episodes, movie fallback links, and no visible series hierarchy. Repairing each symptom keeps the code split across multiple shallow decision paths.

This design replaces that model with one deep scanner recognition module: inventory facts feed a media graph, the graph produces group decisions, and only graph decisions materialize catalog metadata.

## Goals / Non-Goals

**Goals:**

- Classify local media by directory/work group before deciding movie, series, season, episode, version, collection, or supplemental semantics.
- Make filename parsing, sidecars, content-shape, path-tree, hashes, and provider IDs evidence providers instead of final materializers.
- Materialize movies and TV hierarchies from one idempotent graph resolver.
- Keep scan fast by avoiding remote metadata, full media probing, and artwork work in the synchronous classification path.
- Make ambiguous cases visible as review state without publishing wrong movie or orphan episode metadata.
- Remove old scan-time final decision code after the replacement path covers movie and TV ingestion.

**Non-Goals:**

- Preserve old development database recognition rows, orphan episode rows, or movie fallback links.
- Add user-selected movie/show library types.
- Solve every anime absolute numbering, anthology, concert, sports, music, photo, or document case in the first implementation.
- Require episode-level provider matching before local TV browse works.
- Build the full review UI in the first implementation, beyond storing review-required decisions and exposing enough state for governance endpoints.

## Decisions

### Decision: Introduce a scanner-owned media graph

Create a scanner recognition package that owns graph construction and group classification. The graph contains:

- Directory and file nodes from inventory.
- Work group nodes such as `movie_package`, `movie_collection`, `series_package`, `season_package`, `episode_run`, and `ambiguous_group`.
- Resource candidate nodes such as `main_video`, `version`, `multi_part`, `multi_episode`, and `supplemental`.
- Edges such as `contains`, `belongs_to`, `version_of`, `supplemental_of`, `subtitle_of`, and `sidecar_of`.
- Evidence rows for parser tokens, directory structure, sibling runs, sidecars, external IDs, hashes, and manual rules.

Alternative considered: continue using `recognition_candidates` directly as the graph. Rejected as the primary interface because those rows already carry resolver semantics and make it too easy for file-level candidates to bypass group classification.

### Decision: Keep inventory and parser outputs, replace final decision ownership

The existing inventory collection, source listing, exclusion rules, basic filename signal extraction, and sidecar parsing should survive where they are reliable. Their outputs must be converted into graph evidence. Code that directly creates or links final catalog metadata from those signals must be deleted or rewritten.

Alternative considered: wrap old final decisions and feed them into graph materialization. Rejected because it preserves the same ordering bug under a new name.

### Decision: Classify groups with gates and conflicts

Graph classification uses explicit gates:

- TV gates: `SxxEyy`, `1x02`, `第01集`, `EP02`, season directory plus ordered numeric files, repeated episode slots under one work directory, TV sidecar identity, or manual rule.
- Movie gates: one main video with title/year evidence, movie sidecar identity, directory/file title agreement, or manual rule.
- Version gates: same work identity with release/quality/source/edition differences and no episode slot conflict.
- Collection gates: multiple independent movie packages under one parent without shared episode or version evidence.
- Supplemental gates: bounded trailer/sample/extra/featurette/interview/deleted-scene tokens, extras folders, or sidecar roles.
- Review gates: hard conflicts in external IDs, incompatible movie/episode evidence for the same main file, incompatible episode slots, or multiple high-confidence parent groups.

Alternative considered: a single weighted score threshold. Rejected because this domain needs hard blockers: an episode and a movie cannot both own the same main resource without review.

### Decision: Materializer consumes graph decisions only

The materializer applies accepted graph decisions into the catalog:

- TV: `series -> season -> episode -> resource links`
- Movie: `movie -> resource links`
- Versions: multiple resources or link roles under one movie
- Supplemental: resource links with supplemental role under a parent work
- Multi-episode: one resource linked to all accepted episode slots with segment/order metadata
- Ambiguous: inventory-backed visibility and review state, no normal movie/episode publication

Projection refresh must use affected resource links and refresh ancestors for TV descendants.

Alternative considered: let graph decisions call existing materialization helpers that also accept raw file candidates. Rejected because the interface would still permit old bypasses.

### Decision: Metadata matching targets work roots

Automatic metadata matching will be queued only for `movie` and `series` roots. Episodes and seasons inherit hierarchy enrichment from series operations and are skipped if they reach a batch unexpectedly.

Alternative considered: make episode match operations succeed as no-ops. Rejected as the main strategy because queueing unsupported work hides caller mistakes and wastes workflow capacity.

### Decision: Clean up old recognition modules during implementation

Implementation is not complete until old final recognition paths are gone from scan flow. Remaining code must be clearly categorized as:

- Inventory fact collection.
- Signal or sidecar evidence extraction.
- Media graph construction.
- Graph classification and resolver decisions.
- Catalog materialization and projection.
- Governance/manual rule handling.

Tests that only prove the old final decision path should be deleted or replaced with graph fixture tests.

Alternative considered: leave old paths disabled behind flags. Rejected because this workspace is development-reset oriented and the old paths have already made scanner behavior hard to reason about.

## Risks / Trade-offs

- [Risk] The refactor touches scan, recognition, catalog, workflow, and metadata matching together. Mitigation: implement vertical slices in this order: graph schema, movie package materialization, TV season folder materialization, version/supplemental handling, cleanup.
- [Risk] New graph tables add persistence complexity. Mitigation: keep indexed columns for stable group keys, type, status, library/source scope, affected inventory IDs, and store detailed evidence payloads as JSON where direct querying is not required.
- [Risk] Conservative review behavior may show fewer final catalog items initially. Mitigation: keep inventory-backed discovered visibility and make review-required decisions queryable.
- [Risk] Removing old paths can expose edge cases previously handled accidentally. Mitigation: capture current real library failures as fixtures and add broad directory-level tests for the replacement behavior.
- [Risk] Local data reset is disruptive. Mitigation: document reset as part of the change, and provide a targeted rebuild command or maintenance endpoint for development libraries.

## Migration Plan

1. Add media graph schema and Go models for graph nodes, edges, evidence, decisions, and decision-file mappings.
2. Build graph construction from existing `inventory_files`, filename signals, sidecars, and directory summaries.
3. Implement group classification gates for single movie, movie versions, movie collection, standard TV, season folder with numeric files, flat episode folder, multi-episode files, and supplementals.
4. Implement graph decision persistence and idempotent materialization to `MetadataItem`, `Resource`, `ResourceMetadataLink`, and library projections.
5. Wire scan workflows to build, classify, materialize, and refresh projections through graph decisions.
6. Restrict metadata match scheduling and execution to `movie` and `series`.
7. Delete or rewrite old final decision paths from scan classification, sidecar application, content-shape/path-tree materialization, sibling matching fallbacks, and raw recognition manifest materialization.
8. Add fixtures and tests for real failure shapes, including `Show/Season 1/01.mkv`, `Show/Season 1/02.mkv`, movie folders, multi-version movie folders, independent movie folders, extras, and ambiguous mixed folders.
9. Reset local development data or run a targeted rebuild for affected libraries, then rescan and validate counts for series, seasons, episodes, resource links, projections, and workflow status.

Rollback before data reset is a normal branch revert. After local reset, rollback means recreating data with the previous code if needed; no production migration guarantee is required for this change.

## Open Questions

- Should graph evidence be mostly normalized rows, JSON payloads with selected indexed columns, or a hybrid from the first implementation?
- Should ambiguous inventory-backed visibility use existing projection maturity fields or a new review projection type?
- Should manual graph rules be added in the first implementation, or only after automatic graph materialization is stable?
