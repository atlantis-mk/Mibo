## Context

The active backend already persists directory snapshots, inventory files, filename signals, `content_shape` profiles/plans/assignments, recognition units, and directory metadata resolution payloads. The remaining mismatch is that the current `content_shape` plan describes a single directory's direct contents, while metadata materialization often needs a higher semantic boundary.

The failure mode discussed for `暗影蜘蛛侠 Spider-Noir (2026)` is representative: each version folder (`1080p彩版`, `4K彩版`, `4K黑白版`) contains the same `S01E01-S01E08` episode set and can be classified as an `episode_pack`, but the metadata root is the parent directory. The parent has no direct videos and current child-shape inference only recognizes child names like `Season 1` or `extras`, so the pipeline lacks a durable representation for "this parent is the root of one series with versioned episode packs."

The implementation should preserve useful existing behavior from:

- `content_shape_profile.go`: filename token residual/cancellation via `contentShapeTokenConsensus`, primary-video filtering, sidecar evidence, child directory hints.
- `content_shape_plan.go`: conservative ordered planning and review fallbacks.
- `content_shape_assignment.go`: per-file assignments for episode, movie, version, multipart, attachment, and review.
- `recognition_unit_stage.go`: grouping assignments into units consumed by materialization.
- `directory_metadata_resolution.go` and materialization support: directory-scoped metadata resolution.
- `recognition_directory_reduction.go`: residual sibling grouping ideas, but not as the final authority once scope root decisions exist.

## Goals / Non-Goals

**Goals:**

- Separate "leaf directory shape" from "metadata scope root" so bottom folders can remain `episode_pack` while a parent scope is selected as the series root.
- Use token residual/cancellation as the primary bottom-level method for direct sibling videos, not just as a weak auxiliary signal.
- Walk upward from leaf clusters by a bounded depth and evaluate candidate scope roots using identity purity, content coverage, layout explainability, boundary evidence, and attachment handling.
- Persist scope root decisions with stable fingerprints and evidence so recognition units, materialization, directory metadata resolution, and diagnostics consume one authoritative scope.
- Avoid duplicate processing by claiming all files under a selected scope and suppressing redundant per-file or per-leaf materialization inside that scope.
- Treat trailers, samples, extras, specials, featurettes, and other supplemental folders as attachments that can belong to a parent scope without reducing main identity purity.
- Clean up or bypass old single-directory-root assumptions and residual grouping paths after the new scope decision stage is authoritative.

**Non-Goals:**

- Replacing metadata provider search, provider detail enrichment, playback, projection, or frontend detail presentation models.
- Requiring full-library batch completion before producing scope decisions; the design must work during normal incremental scanning.
- Making every unknown parent layout automatic; ambiguous mixed scopes remain review-required.
- Removing low-level filename/folder parsing helpers that are still useful to the active pipeline.

## Decisions

### Leaf classification stays local and direct-file only

Leaf classification will classify a directory from its direct video children only. It will not decide whether the directory is the metadata root. A leaf summary will include shape, dominant identity, title evidence, season set, episode set, part set, version signature, attachment summary, confidence, review state, and residual-token evidence.

The existing `contentShapeTokenConsensus` behavior will be promoted into a clearer leaf-classifier responsibility:

1. Parse primary video filename/path signals.
2. Split filename titles into comparable tokens.
3. Remove tokens common to all primary videos in the folder.
4. Interpret residual tokens as episode markers, multipart markers, version markers, distinct movie titles, or conflicts.
5. Fall back to existing score-based evidence only when residual evidence is insufficient.

Alternative considered: let upward scope detection reparse every file. That would duplicate parser behavior and make leaf tests less useful.

### Metadata scope root is a separate persisted decision

A new scope decision stage will consume leaf summaries and child summaries rather than replacing `content_shape` plans. It will persist a record conceptually shaped like:

```json
{
  "scope_path": "/.../暗影蜘蛛侠 Spider-Noir (2026)",
  "root_kind": "series",
  "layout": "versioned_episode_packs",
  "identity_key": "series:spider-noir",
  "confidence": 0.94,
  "children": [
    {"path": "1080p彩版", "role": "version", "version_key": "1080p-color"},
    {"path": "4K彩版", "role": "version", "version_key": "2160p-color"},
    {"path": "4K黑白版", "role": "version", "version_key": "2160p-bw"}
  ]
}
```

`directory_shape` remains useful for leaf-level planning, but final recognition/materialization should prefer the scope decision when present.

Alternative considered: add only a `series_versions_folder` shape. That would solve the immediate case but would continue mixing leaf shape and scope root layout, and it would not generalize to split episode packs, attachment-only sibling folders, or movie versions spread across child directories.

### Upward reduce selects "complete, pure, explainable, bounded" scopes

For each changed leaf cluster, the stage will inspect parent candidates up to a configured bounded depth, initially 4 levels or until the library root. A candidate is scored by:

- **Identity purity**: one dominant movie or series identity explains the main children.
- **Coverage gain**: the parent adds versions, seasons, parts, attachments, or complementary episode ranges without adding unrelated works.
- **Layout explainability**: child roles match known layouts such as `series/season_directories`, `series/versioned_episode_packs`, `series/split_episode_packs`, `movie/version_directories`, `movie/multipart_parts`, `movie_collection`, or `attachment_orphan_review`.
- **Boundary evidence**: the current directory name matches the dominant identity, while the parent looks like a library root, source/share folder, category folder, or contains multiple unrelated identities.
- **Attachment neutrality**: attachment child folders are excluded from main purity scoring but retained in the final scope.

The selected scope is the highest candidate that remains pure and explainable before the parent boundary would reduce purity or add unrelated identities.

Alternative considered: always choose the lowest confident leaf. That keeps materialization simple but misses versioned series and parent-level attachments.

### Attachments are first-class child roles

Leaf directories and files identified as trailers, samples, extras, featurettes, behind-the-scenes clips, interviews, deleted scenes, NCOP/NCED, and similar supplemental media will be summarized as attachments. They must not create movie/series metadata by themselves unless configured sidecar evidence says they are special episodes. Scope detection will attach them to the nearest compatible main scope or produce `attachment_orphan_review` if no main scope exists.

Ambiguous labels such as `SP`, `OVA`, `Specials`, and `番外` require stronger episode evidence before becoming episodes; otherwise they are attachment/review candidates.

Alternative considered: count every video child in scope purity. That incorrectly turns normal movie/series roots with trailers into mixed-review scopes.

### Scope claims prevent duplicate materialization

When a scope decision is accepted, the pipeline will record the inventory files covered by that scope fingerprint. Subsequent leaf scans inside the same unchanged scope will not enqueue independent recognition/materialization tasks. Partial refreshes that touch one child folder will recompute the affected leaf and its ancestor scope candidates, then update or invalidate the scope claim.

Alternative considered: continue materializing each leaf and rely on metadata merge. That creates duplicate local metadata, noisy governance, and harder rollback.

### Old root inference paths are retired after parity

The new scope decision becomes the authoritative source for final metadata root selection. Existing direct-directory `content_shape` plans remain as leaf summaries and backward-compatible evidence, but downstream code that treats a leaf plan as the final metadata root must be removed or gated behind missing scope decisions only during migration. The residual directory reduction path can be retained only as a temporary fallback until scope decision parity tests cover its useful cases.

Alternative considered: keep both systems active indefinitely. That preserves inconsistent sources of truth and makes debugging directory outcomes harder.

## Risks / Trade-offs

- [Risk] Scope detection may over-group unrelated sibling folders that share a weak title. -> Mitigation: require identity purity, compatible child layouts, and boundary evidence; fall back to review when title/episode/movie evidence conflicts.
- [Risk] Partial scans may see only one child version and choose a lower scope. -> Mitigation: use persisted directory snapshots and child summaries for siblings under the candidate parent, not only files observed in the current scan callback.
- [Risk] Attachment labels vary across languages and fandom naming conventions. -> Mitigation: keep a conservative attachment vocabulary, require episode evidence for ambiguous specials, and expose review evidence.
- [Risk] New persisted scope decisions may leave stale content shape and recognition rows. -> Mitigation: bump classifier/scope versions, include child summaries in fingerprints, and invalidate covered recognition units on scope changes.
- [Risk] Removing residual grouping too early could regress movie version or multipart behavior. -> Mitigation: port representative residual grouping fixtures into leaf/scope tests before deleting or disabling that path.
- [Risk] Scope task claiming can hide files if coverage is computed incorrectly. -> Mitigation: persist covered file IDs/paths in evidence, add diagnostics for orphan leaf clusters, and test deletion/addition/invalidation flows.

## Migration Plan

1. Introduce leaf summary types and repositories without changing materialization behavior.
2. Refactor bottom-level `content_shape` planning so token residual/cancellation evidence is explicit, persisted, and testable as leaf classification output.
3. Add scope decision types, fingerprints, repositories, and diagnostics.
4. Build upward reduce logic using persisted snapshots, leaf summaries, and bounded ancestor traversal.
5. Add scope claim handling to prevent duplicate materialization inside an accepted scope.
6. Adapt recognition unit construction and directory metadata resolution to consume scope decisions when present.
7. Port tests for existing movie folder, movie versions, multipart, movie collection, episode pack, season folder, attachment group, and residual directory reduction behavior.
8. Add new integration fixtures for versioned series roots, split episode packs, parent-level attachments, orphan attachment review, partial refresh, and boundary detection.
9. Bump classifier/scope versions and invalidate stale derived rows.
10. Remove or disable obsolete single-directory-root and residual grouping code paths once parity tests pass.

Rollback strategy: keep scope decision consumption behind a versioned stage boundary during rollout. If materialization regressions appear, disable scope-decision consumption while preserving leaf summaries and diagnostics, then re-enable after the faulty reducer rule is corrected.

## Open Questions

- Should scope decisions be stored in a new table or as an extension of `recognition_units`/`content_shape_plans`? A new table is cleaner, but integration with existing repositories may be faster if staged carefully.
- Should attachment vocabularies be configurable per library, or start as system defaults with later user overrides?
- How should manual governance override a wrong scope decision: by pinning `scope_path`, forcing review, or editing child roles?
- What is the first production-safe scope version string and should it share the existing `ContentShapeClassifierVersion` or use a separate `MetadataScopeClassifierVersion`?
