## Context

Mibo's current library scan path discovers storage objects, writes inventory files, derives filename signals, infers directory shape, builds recognition manifests, materializes metadata/resources, and schedules metadata match/probe/projection work across several library and workflow functions. The concepts are already present, but the boundaries are soft: downstream stages frequently infer scope, shape, or signals again from file paths and manifest contents.

This design makes directories the durable coordination boundary. A scan run produces reusable stage outputs in order: directory snapshots, inventory facts, file signals, directory shape plans, recognition units, materialization results, enrichment tasks, and one coalesced projection refresh. Each stage consumes upstream persisted results and stage fingerprints instead of listing provider paths or reparsing filenames.

## Goals / Non-Goals

**Goals:**

- Make unchanged directories cheap to rescan by comparing directory and stage fingerprints.
- Ensure each stage enriches the prior stage's output instead of recomputing the same data.
- Replace ad hoc recognition grouping by file parent path with stable recognition units derived from directory shape plans.
- Preserve current user-facing library creation, manual scan, scheduled scan, targeted refresh, browse, playback, metadata match, probe, and projection behavior.
- Keep existing content shape, recognition manifest, metadata, resource, catalog, and ingest event tables usable during migration.
- Reduce workflow task count by coalescing tasks by directory, recognition unit, and final projection scope.

**Non-Goals:**

- Replacing the recognition candidate and materializer algorithms wholesale.
- Changing metadata provider matching semantics beyond when work is scheduled and deduplicated.
- Removing existing recognition manifests, candidates, decisions, or content shape tables in this change.
- Introducing a distributed queue or external dependency.

## Decisions

### Directory snapshots become the first durable stage

Add a directory snapshot model keyed by library, media source, storage provider, root path, directory path, and scanner version. A snapshot stores the provider-facing object summary, visible media paths, child directories, fingerprint, last observed time, and changed/unchanged status for the current scan.

Rationale: provider traversal is the most expensive and least reusable operation. Later stages must not list storage again when the snapshot already describes the directory.

Alternative considered: keep snapshots in memory only during a workflow run. This would improve one run but would not allow repeated scans or targeted refreshes to skip unchanged directories.

### Inventory and signals consume snapshots in batches

Inventory sync consumes changed snapshots and writes `inventory_files`, sidecar associations, scan exclusions, and missing markers in batches. File signal hydration consumes those inventory rows and persists `inventory_file_signals` once per signal version.

Rationale: signal parsing and sidecar interpretation are reusable facts. Directory shape and recognition must consume persisted signals rather than reparsing filenames.

Alternative considered: have directory shape compute signals on demand. This preserves current behavior but repeats expensive parsing and makes stage fingerprints harder to reason about.

### Directory shape is the planning boundary

Directory shape planning consumes snapshots, inventory facts, file signals, and scan rules to produce `content_shape_profiles`, `content_shape_plans`, and `content_shape_assignments`. The shape plan is the authoritative downstream input for whether a directory is a movie folder, movie versions folder, movie collection folder, season folder, episode pack, flat episode folder, attachment group, or review-required directory.

Rationale: this makes the user's desired flow explicit: directories are classified once, then later stages act according to that classification.

Alternative considered: infer folder shape inside recognition work unit construction. That keeps fewer tables involved but hides the decision and makes metadata match and governance repeat the same inference.

### Recognition units are durable, fingerprinted work items

Add recognition units keyed by library/root/scope/unit key. A recognition unit references its source directory plan, shape, file IDs, assignment IDs, parent unit when needed, unit fingerprint, status, and last materialized fingerprint. Recognition manifests remain the persistence format for candidates/evidence/decisions, but they are built from recognition units.

Rationale: materialization can skip units whose fingerprint has already succeeded, and workflow tasks can target stable units rather than transient file ID batches.

Alternative considered: keep only recognition manifests as units. Manifests are useful after candidate construction, but they do not clearly represent pre-materialization scheduling, unchanged-unit skip state, or directory-shape-derived grouping.

### Materialization produces a stage result consumed by enrichment

Materialization consumes recognition units and writes metadata/resources/links. It also records per-unit materialization results: metadata IDs, resource IDs, file IDs, projection scope IDs, remote-search eligibility, probe eligibility, and warnings.

Rationale: metadata matching, probing, and projection should consume materialization output directly and should not rediscover IDs by reloading manifests and files.

Alternative considered: keep building `DirectoryMetadataResolutionPayload` after materialization from manifest/files. That repeats shape lookup and makes enrichment less deterministic.

### Workflow is staged and coalesced

The workflow order becomes snapshot -> inventory sync -> file signals -> directory shape -> recognition unit build -> materialize units -> enrichment -> projection. Tasks use stable keys by directory or unit fingerprint and are coalesced per run. Projection is scheduled once per affected library scope after all materialization/enrichment planning is complete.

Rationale: stage ordering makes dependency boundaries explicit and avoids many small duplicate tasks.

Alternative considered: keep the current scan task and append more post-scan tasks. That is lower-risk short term but leaves repeated data processing and task explosion intact.

## Risks / Trade-offs

- [Risk] New persisted pipeline state can drift from legacy tables. → Mitigation: derive new state from existing inventory/content-shape/recognition tables during migration and add reconciliation tests.
- [Risk] Fingerprint bugs could skip required work. → Mitigation: include scanner version, scan policy, exclusion rules, signal version, visible video paths, stable identity, size, modified time, and relevant sidecar paths in fingerprints; add forced refresh paths.
- [Risk] Workflow migration could interrupt existing queued runs. → Mitigation: support legacy task handlers during rollout and let new runs use the staged pipeline only after schema migration succeeds.
- [Risk] Recognition unit grouping can change metadata/resource identities. → Mitigation: keep canonical recognition keys and materializer sort keys stable; add end-to-end tests for existing movie, series, version, multipart, and collection fixtures.
- [Risk] More tables make debugging harder if not surfaced. → Mitigation: add query helpers and diagnostics for directory snapshot, shape plan, unit, materialization result, and enrichment plan lineage.

## Migration Plan

1. Add directory snapshot, recognition unit, and materialization result models with indexes and migrations.
2. Backfill directory snapshots and recognition units from current `inventory_files`, `content_shape_plans`, and recognition manifests when possible.
3. Implement staged services behind existing library scan APIs while preserving legacy task handlers.
4. Route new scan workflows through the staged pipeline.
5. Compare staged results with current recognition/materialization behavior in tests and selected compatibility fixtures.
6. Remove or simplify legacy ad hoc grouping paths only after staged pipeline tests cover create, manual scan, scheduled scan, targeted refresh, and unchanged refresh.

Rollback strategy: keep legacy scan and recognition handlers callable until the staged pipeline is proven. If staged execution fails, mark the run failed without deleting existing inventory, metadata, resources, or recognition records.

## Open Questions

- Should directory snapshots store full object JSON or only normalized summaries plus fingerprints?
- Should recognition unit file membership be stored as JSON, join rows, or both for query performance?
- Should forced refresh invalidate all downstream fingerprints or only the snapshot and dependent directory/unit rows?
