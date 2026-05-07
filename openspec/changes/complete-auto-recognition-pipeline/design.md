## Context

The scanner has moved toward a source-first, shape-first architecture: `inventory_files` stores file facts, `inventory_file_signals` stores reusable filename/path evidence, and `content_shape_profiles/plans/assignments` classify individual directories before catalog materialization. This solves repeated parsing and many single-directory cases, but real media sources often express one work across sibling directories or require parent-directory grouping before the right catalog shape is visible.

Examples that still need a higher-level recognition pass:

- Two sibling release folders each contain one file for the same movie, such as `3.Iron.2004...MiniHD/3.Iron...mkv` and `3-Iron.2004...TAGHD/3-Iron...mkv`.
- A collection directory contains many independent movies, each as either a file or a one-file child folder.
- A series is split as sibling `Show S01`, `Show S02`, or noisy season directories under one parent.
- A parent directory contains attachments or extras that need a likely parent work but are not in the same leaf directory.

The design must preserve the user's constraints: no movie/show library type choice, fast scans, low user involvement, no heavy metadata calls in the fast path, and reuse/cleanup of existing code rather than parallel recognition engines.

## Goals / Non-Goals

**Goals:**

- Add a path-tree work-group compiler above existing content-shape directory plans.
- Detect sibling-directory movie versions and materialize them as one movie with multiple assets.
- Split movie collections into separate movie groups when title/year evidence differs.
- Group series roots across season folders and episode packs.
- Reuse `inventory_file_signals` and existing `content_shape` profiles/plans/assignments as inputs.
- Avoid per-file remote lookups; metadata matching remains asynchronous and work-group scoped.
- Preserve reviewable evidence and scoped correction rules for ambiguous work groups.
- Clean up redundant file-first fallback paths when work-group assignments cover files.

**Non-Goals:**

- Do not reintroduce library media types.
- Do not use TMDB, ffprobe, content hashes, artwork downloads, or media reads for fast work-group recognition.
- Do not redesign catalog item or asset schemas unless existing asset-item semantics cannot represent a group.
- Do not build a new review UI in this change.
- Do not attempt automatic merge of already-created duplicate catalog items outside scanner-owned identities unless it can be done safely and explicitly.

## Decisions

### Decision: Add a path-tree work-group compiler above content-shape plans

The new compiler consumes directory snapshots, child directory profiles/plans, and indexed file signals to create parent-scope work groups. It does not replace `content_shape`; it uses leaf directory plans as evidence and emits higher-level assignments where a parent has enough evidence to override per-leaf materialization.

Conceptual flow:

```text
Directory snapshots + file signals
  -> leaf content_shape plans
  -> parent path-tree work groups
  -> work-group assignments
  -> catalog materialization
```

Rationale:

- Existing `content_shape` is correct for single-directory cases and should be reused.
- Parent-level grouping is required for sibling release folders and series roots.
- Keeping this as a compiler avoids scattering special cases through file-first materialization.

Alternatives considered:

- Fold all logic into `compileContentShapePlan`. Rejected because that function operates on one directory profile and should stay bounded.
- Deduplicate only after TMDB match. Rejected because it delays correctness, still creates duplicate catalog rows, and depends on remote metadata availability.

### Decision: Represent work groups as plan rules plus assignment overrides

Use existing content-shape concepts where possible: a work group has shape, confidence, review state, evidence, alternatives, and assignments. Persist in existing content-shape tables if they can represent parent-scope groups; add a minimal work-group table only if necessary to avoid overloading directory plans.

Rationale:

- The project already has plan, assignment, evidence, and correction-rule patterns.
- Assignment overrides allow a parent plan to say that two files in different child directories belong to one movie work.
- Existing catalog materialization can remain mostly assignment-driven.

### Decision: Sibling movie versions are detected by normalized work key

Sibling one-file child directories SHALL be grouped as movie versions when all likely main files share a normalized title/year work key and their differences are primarily release hints: quality, source, codec, audio, HDR, edition, release group, or container.

Example:

```text
/电影/合集5-2/空房间...3.Iron.2004...MiniHD/3.Iron.2004...MiniHD.mkv
/电影/合集5-2/空房间...3-Iron.2004...TAGHD/3-Iron.2004...TAGHD.mkv
```

Both should produce one movie work key, such as `3 iron:2004`, and two source assets.

Rationale:

- Real release packs often store each version in its own folder.
- The file signal index already exposes title, year, and release-hint differences cheaply.

### Decision: Movie collections split by distinct title/year keys

Parent directories with multiple child files or child folders should become movie collections when title uniqueness and year density are high and episode sequence evidence is low. Each distinct title/year key becomes a separate movie group.

Rationale:

- Common download folders are collection containers, not works.
- Splitting avoids both one giant movie and false episode packs.

### Decision: Series roots group season-like children

A parent directory should become a series work group when children are season folders, episode packs, or sibling directories with normalized series title evidence and compatible season numbers.

Rationale:

- Users should not need to structure all series as `Series/Season N` exactly.
- Current inherited context handles some nested cases but not all sibling season-root forms.

### Decision: Work-group recognition is fast and bounded

The compiler can inspect a bounded parent directory and its immediate children already visited during scan. It can reuse persisted snapshots/profiles and file signals. It must not recursively rescan unrelated subtrees or call remote providers.

Rationale:

- Speed matters more than perfect global inference.
- Parent-level grouping should be O(children + visible files), not O(entire source) for every directory.

### Decision: Metadata matching is queued per work group

After materialization, movie groups queue one movie metadata match and series groups queue one TV metadata match. Episodes and version assets do not trigger per-file remote searches.

Rationale:

- Avoids slow scans and provider rate pressure.
- Keeps remote metadata as enrichment/correction, not fast-path classification.

### Decision: Ambiguous work groups are reviewable and low-touch

When group candidates are close or below threshold, the scanner preserves a review-required decision with alternatives and either creates guarded placeholders for visibility or leaves inventory-only records depending on existing UX requirements. User corrections should save source/path-scoped rules that future scans reuse automatically.

Rationale:

- The user wants minimal participation, not no governance.
- A directory-level correction should solve future files without repeated prompts.

## Risks / Trade-offs

- [Risk] Sibling movie-version grouping may merge unrelated remakes with same title/year-like evidence. -> Mitigate with title/year exactness, release-hint difference checks, and review-required fallback when multiple strong keys exist.
- [Risk] Movie collections with numeric franchise names can look like episode sequences. -> Mitigate with year density, title uniqueness, release-token suppression, and minimum sequence confidence margins.
- [Risk] Parent-level grouping can override correct leaf plans. -> Mitigate by requiring high confidence and preserving leaf plan alternatives in evidence.
- [Risk] Existing duplicate catalog items may remain. -> Mitigate by applying work-group grouping to future scans first; add explicit scanner-owned reconciliation only after identity safety is proven.
- [Risk] More persisted plan state increases invalidation complexity. -> Mitigate with classifier version, parent fingerprint inputs, child plan fingerprints, and scoped invalidation.

## Migration Plan

1. Add in-memory path-tree work-group compilation using existing snapshots, file signals, and content-shape plans.
2. Materialize sibling-directory movie versions and movie collections through assignment overrides without new schema if possible.
3. Persist parent work-group plans and assignments using existing content-shape tables or add a minimal table if existing scope keys are insufficient.
4. Add series-root grouping across season-like sibling directories.
5. Queue metadata matching per work group and verify no per-file remote calls are introduced.
6. Add scoped correction rule support for work-group confirmations.
7. Clean up covered file-first fallback paths and add regression tests.

Rollback strategy:

- Disable work-group compilation and fall back to existing leaf content-shape materialization.
- Keep file signals and leaf content-shape plans intact.
- Ignore any additive work-group state if rollout is disabled.

## Open Questions

- Should parent work groups be persisted in `content_shape_plans` using parent directory paths, or should they get a dedicated `recognition_work_groups` table?
- Should existing duplicate scanner-owned movies with the same future work key be reconciled automatically, or only new scans?
- What exact confidence threshold should be required for sibling-directory movie-version grouping?
