## Context

The current library workflow scans storage, builds inventory files, defers recognition materialization, and then queues follow-up metadata match, probe, and projection tasks. Recognition materialization already uses directory evidence from `content_shape_*`, `scanrecognition`, and `recognition_directory_reduction` to distinguish movie folders, multi-version movie folders, movie collections, series or season folders, attachments, and review-required cases.

The disconnect is after materialization: `RunMetadataMatchBatch` receives metadata item IDs and processes each movie or series item one at a time. This loses the directory-level decision that already explains whether one metadata lookup is enough for a folder, whether a folder contains several independent movie identities, or whether the folder should not be matched automatically.

## Goals / Non-Goals

**Goals:**

- Make automatic post-scan metadata work directory-scoped instead of flat item-scoped.
- Use existing content-shape and recognition evidence as the source of truth for the automatic resolution unit.
- Ensure single movie, multipart movie, and movie-version folders resolve the movie work once and bind all resources to it.
- Ensure series and season folders resolve the series once and bind seasons and episodes from local numbering, sidecars, or hierarchy data.
- Avoid speculative search when no configured search provider exists or when the directory is review-required.
- Preserve manual metadata apply, refetch, local-scan evidence, governance operation records, and catalog projection refresh behavior.

**Non-Goals:**

- Replacing the recognition kernel or content-shape classifier.
- Adding new remote provider types or changing provider plugin protocol contracts.
- Changing public HTTP API surfaces unless implementation discovers an existing UI needs extra diagnostics.
- Implementing a full manual review UI for every ambiguous directory shape.

## Decisions

### 1. Introduce directory metadata resolution units

Create an internal directory resolution payload produced after recognition materialization. The payload should include library ID, scope path, directory shape, review state, materialized work IDs, episode IDs, resource IDs, primary series/work IDs, and flags for local evidence and remote search eligibility.

Rationale: the workflow needs a durable unit that can be queued, retried, tested, and summarized without reconstructing directory semantics from arbitrary metadata IDs.

Alternative considered: keep `match_metadata` tasks but add filters. This would reduce some work but would still force series and multi-version semantics through item-level matching.

### 2. Keep item-level metadata operations for manual and explicit actions

Do not remove `MatchMetadataItemOperation`, manual apply, or refetch. Directory-scoped resolution should be the automatic post-scan path; explicit user actions can continue to target a single metadata item.

Rationale: manual workflows need precise item targeting, and existing metadata governance records are built around item operations.

Alternative considered: replace item matching entirely. This would be a larger migration and risks breaking manual correction workflows.

### 3. Make remote search eligibility explicit

Each directory resolution unit should decide whether remote search is allowed before invoking provider search. Search is allowed when the directory shape is automatically resolved, the profile has an operational search provider, and the unit has a clear work identity. Search is blocked for attachment-only, extras-only, unknown-review, mixed-conflict, and ambiguous folders.

Rationale: this prevents slow no-op tasks and avoids low-confidence matches in ambiguous folders.

Alternative considered: let metadata operation matchability decide after task start. That still creates queue noise and repeated operation records.

### 4. Resolve series at the series scope, not the episode scope

For series, season, episode-pack, and flat episode folders, directory resolution should select or create the series metadata once. Season and episode metadata should be derived from recognized hierarchy, local sidecars, and remote hierarchy detail when available, but individual episodes must not run independent search.

Rationale: most providers search series, not arbitrary episode files, and local SxxExx evidence is stronger for binding episodes than repeated title searches.

Alternative considered: match season or episode items separately when they have titles. This is slower and prone to false positives for generic episode names.

### 5. Treat movie collection folders as multiple identities under one scope

Movie collection folders should produce one directory task containing several movie identities. If a movie identity has local sidecar or external ID evidence, apply that evidence or detail provider as allowed. If no search provider is configured and no local evidence exists, keep the generated local metadata provisional instead of recording failed matches.

Rationale: a collection folder is still one directory decision, but it legitimately contains multiple works.

Alternative considered: one task per movie item. This regresses to item-level behavior and loses the collection-level decision.

## Risks / Trade-offs

- Directory unit generation may miss a shape transition after rescans -> derive units from the same persisted content-shape and recognition materialization records used by the resolver, and add regression tests for rescans.
- Suppressing automatic match can hide a valid remote match for ambiguous folders -> leave governance/review state explicit and keep manual apply/refetch available.
- Series hierarchy application can update many metadata rows from one task -> keep projection scope and operation affected IDs explicit so catalog refresh remains bounded.
- Existing workflow dashboards may still label the stage as `metadata_match` -> preserve task stage names initially while changing payload semantics, or add a new task type with compatibility summaries.
- Directory-scoped tasks may be harder to retry per single item inside a collection -> record per-identity outcomes in the operation evidence and keep manual item operations for corrections.

## Migration Plan

1. Add the directory resolution unit model and create it from recognition materialization results without changing queueing.
2. Add tests proving unit generation for single movie, multipart movie, movie versions, series/season, movie collection, extras, and ambiguous directories.
3. Introduce a directory metadata workflow handler and route post-recognition queueing through it.
4. Keep the existing item-level match task available for manual or legacy callers.
5. Verify projection refresh still runs after directory resolution and after no-op review-required directories.
6. Roll back by routing post-recognition queueing back to `queueWorkflowMatchTasks` while leaving item-level metadata operations untouched.

## Open Questions

- Should directory-scoped automatic resolution use a new workflow task type such as `resolve_directory_metadata`, or reuse `match_metadata` with a versioned payload for UI continuity?
- Should collection folder identities be resolved serially inside one task, or split into child units while retaining a parent collection scope?
- Should local provisional metadata use a new governance status distinct from `unmatched`, or reuse existing local-scan source records plus current governance values?
