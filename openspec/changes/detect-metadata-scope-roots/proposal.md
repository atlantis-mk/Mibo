## Why

The current scan recognition pipeline plans each directory mostly from its direct files or direct child directory names, so it can classify leaf folders such as `4K彩版` as `episode_pack` while missing that their parent is the real metadata root for a versioned series. This causes directory semantics, recognition units, metadata resolution, and review evidence to depend on local folder guesses instead of the complete media scope.

## What Changes

- Introduce leaf directory classification as a first-class stage that classifies the innermost video-containing folder from direct sibling video signals using token residual/cancellation, primary-video filtering, sidecar hints, and conservative review fallbacks.
- Introduce metadata scope root detection that starts from leaf clusters, walks upward a bounded number of levels, compares sibling child clusters, and selects the highest directory that is complete, pure, explainable, and bounded by a classification/source/library boundary.
- Represent scope decisions separately from leaf directory shape, including `root_kind`, `layout`, `scope_path`, child roles, attachment roles, confidence, and evidence.
- Support layouts that current leaf-only shape planning cannot express, including versioned episode packs, season directories, split episode packs, movie version directories, multipart movie scopes, movie collections, and attachment-only/orphan-review scopes.
- Treat trailers, samples, extras, specials, featurettes, and other supplemental folders as attachments that do not damage scope purity, while still surfacing orphan or ambiguous attachment groups for review.
- Replace downstream materialization and metadata-resolution task creation with scope-level tasks once a scope root is claimed, preventing duplicate per-file or per-leaf processing inside the same metadata scope.
- Remove or retire obsolete code paths that infer final metadata roots from single-directory `content_shape` plans alone or from residual directory reduction after scope root detection becomes authoritative.
- Bump relevant classifier/scope versions and fingerprints so stale content shape plans, recognition units, and directory metadata resolution payloads are regenerated.

## Capabilities

### New Capabilities
- `leaf-directory-shape-classification`: Classifies direct video-containing folders using file signals, token residuals, primary/attachment filtering, and review-safe shape decisions.
- `metadata-scope-root-detection`: Finds the metadata top-level directory by upward reducing leaf clusters into scope decisions with root kind, layout, child roles, and boundary evidence.

### Modified Capabilities
- None. There are no archived specs under `openspec/specs/`; this change introduces new capabilities while replacing implementation behavior in the active scan recognition pipeline.

## Impact

- Backend scan pipeline: `mibo-media-server/internal/library/scan_run.go`, `recognition_planning.go`, `content_shape_profile.go`, `content_shape_plan.go`, `content_shape_assignment.go`, `recognition_unit_stage.go`, `recognition_unit_materialization_support.go`, `directory_metadata_resolution.go`, and related repositories/models.
- Recognition/materialization: recognition units, resource grouping, metadata item materialization, resource metadata links, attachment links, and directory metadata resolution payloads.
- Database model/versioning: new or extended persisted rows for leaf cluster summaries and scope decisions, plus classifier/scope fingerprint version bumps.
- Cleanup: retire single-directory-root assumptions and legacy residual grouping paths that conflict with scope-level root detection.
- Tests: add focused unit tests for leaf token residual classification, upward scope selection, attachment filtering, versioned series roots, split seasons, movie versions, multipart movies, mixed review boundaries, and integration tests for scan-to-materialization behavior.
