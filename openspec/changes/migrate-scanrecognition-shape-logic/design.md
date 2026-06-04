## Context

The scan pipeline has already moved toward directory-scoped planning: snapshots feed inventory, file signals, `content_shape` profiles/plans, recognition units, and materialization. The remaining architectural mismatch is that the older `scanrecognition` package still contains a tree classifier with mature directory-shape heuristics, and some legacy recognition adapter code can still build materialization candidates from that old classifier.

The active ingest path should have one authority for directory shape: `content_shape`. The old classifier's useful ideas should become native profile evidence, plan rules, assignment behavior, and tests. After parity is established, the old tree classifier and recognition adapter path can be removed.

## Goals / Non-Goals

**Goals:**

- Make `content_shape` the sole authority for scan-time directory shape and materialization planning.
- Migrate all valuable old classifier behavior into `content_shape`, including movie collections, movie versions, multipart movies, token-consensus decisions, NFO conflict protection, primary-video-first decisions, and season-parent inference.
- Preserve low-level filename/folder parsing helpers only where they remain useful, but move or isolate them so `content_shape` does not depend on the old tree classifier.
- Delete obsolete tree-classification and scan-recognition materialization paths after replacement behavior is tested.
- Bump classifier versions and fingerprints so existing stale plans are regenerated.

**Non-Goals:**

- Replacing metadata provider matching, probing, playback, projection, or frontend presentation models.
- Changing public API response shapes except for improved organization/review outcomes.
- Keeping a compatibility mode that can route materialization through the old tree classifier.

## Decisions

### `content_shape` owns final shape planning

All directory-shape decisions used by materialization will be represented as `contentShapeDirectoryProfile`, `contentShapeDirectoryPlan`, and `contentShapeFileAssignment` data. Recognition units and materialization will consume these outputs only.

Alternative considered: call `scanrecognition.ClassifyTree` as a fallback when `content_shape` returns `unknown_review`. This would reduce short-term work, but it would preserve two sources of truth and make future review/materialization behavior harder to reason about.

### Migrate heuristics as evidence, not as a copied tree model

Old classifier behavior will be decomposed into profile fields and plan helpers:

- identity distribution for same-work versus multi-work folders;
- normalized version identity that removes version, edition, release, quality, and codec noise;
- multipart group/part coverage with continuous-sequence checks;
- token consensus across sibling filenames;
- primary video counts that exclude trailers/samples/extras from shape decisions;
- sidecar/NFO shape hints and conflict flags;
- child directory shape summaries for series parent inference.

Alternative considered: copy the classifier's directory-node structs into `library`. That would move code without simplifying the pipeline.

### Ambiguity becomes explicit review

Contradictory evidence must produce a review-required `content_shape` plan with reason/evidence. Examples include movie NFO inside season-like directories, episode NFO beside movie-like filenames, broken multipart sequences, and mixed high-confidence movie/episode evidence.

Alternative considered: pick the higher score. That risks creating wrong works/resources, which is harder to repair than review-required plans.

### Deletion happens after parity tests pass

The old classifier package can be removed only after each migrated behavior has focused `content_shape` tests and at least one integration/materialization test for representative shapes. Any parsing helpers still needed must be moved before deleting the package.

Alternative considered: delete first and repair compile errors. That makes it too easy to lose subtle behavior.

## Risks / Trade-offs

- [Risk] Migrated thresholds may not exactly match old behavior. -> Mitigation: port old classifier fixtures into `content_shape` tests before deleting old tests.
- [Risk] Some helpers in `scanrecognition` are still used outside the classifier. -> Mitigation: split parsing helpers from classifier code first, then delete only tree-classification and adapter code.
- [Risk] New rules could over-classify collections and skip review. -> Mitigation: keep conflict rules ahead of collection rules and require explicit evidence for high-impact shapes.
- [Risk] Existing cached plans may continue showing old behavior. -> Mitigation: bump `ContentShapeClassifierVersion` and include sidecar/signal/inventory inputs in fingerprints.
- [Risk] Removing the legacy adapter may break hidden tests around recognition candidates. -> Mitigation: compare candidate keys, resource shapes, review states, and materialized outputs before removal.

## Migration Plan

1. Inventory all current references to `scanrecognition` and separate parser/helper use from tree-classifier use.
2. Add missing `content_shape` profile fields for identity distribution, version identity, multipart coverage, token consensus, primary-video count, sidecar/NFO hints, conflict flags, and child shape summaries.
3. Port old classifier behavior tests into `internal/library` as `content_shape` profile/plan/assignment/materialization tests.
4. Implement plan rules in conservative order: conflict review, explicit sidecar hints, multipart, versions, episodic/season, series parent, movie collection, attachment group.
5. Update assignments and recognition-unit construction for multipart resources, movie versions, per-catalog-id collections, and per-file-title collections.
6. Bump classifier version and verify stale plans are invalidated.
7. Remove old tree-classifier APIs, `recognition_scan_adapter` fallback/materialization path, and obsolete tests after replacement coverage passes.
8. Run backend tests and targeted rescan diagnostics against large movie collection, movie versions, multipart movie, season, and NFO-conflict fixtures.

Rollback strategy: before deleting old code, keep the migration in small commits so the removal step can be reverted independently. After deletion, rollback means reverting the removal commit and lowering only the route selection, not restoring two active authorities.

## Open Questions

- Should parsed NFO/sidecar shape hints be persisted as inventory signal rows or folded into `content_shape` profiles only?
- Should token-consensus helpers live under `internal/library` or a neutral parsing package if future importers need them?
- Should review-required plans expose a user-facing reason code for “conflicting sidecar evidence” and “broken multipart sequence” in the library UI?
