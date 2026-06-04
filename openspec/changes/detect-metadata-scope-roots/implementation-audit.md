# Implementation Audit

## 1.1 Final-root assumptions from content shape plans

Current scan-to-materialization flow treats a persisted `content_shape_plans` row as the directory-level authority for downstream work:

- `internal/library/recognition_planning.go`
  - `contentShapeCachePlanForDirectory` builds or reuses one `ContentShapePlan` for the scanned `DirectoryPath`.
  - `compileCachedContentShapePlan` persists the plan, generates assignments for the same snapshot, stores review decisions, and caches the plan by provider/root/directory.
  - The plan is local to the scanned directory. Parent scope semantics are not represented.
- `internal/library/recognition_unit_stage.go`
  - `loadOrBuildRecognitionUnitsForPlans` consumes `ContentShapePlan` rows directly.
  - `loadRecognitionUnitStageInput` loads assignments for the plan and inventory files from those assignment paths.
  - `recognitionUnitRecord` writes `RecognitionUnit.DirectoryPath`, `DirectoryShape`, `SourcePlanID`, `SourcePlanFingerprint`, and `Fingerprint` from the plan, making the plan directory the unit scope.
- `internal/library/recognition_manifest.go`
  - `persistRecognitionManifestForUnit` derives manifest `scopePath` from the recognition unit files with `commonRecognitionScopePath`.
  - If the unit has authoritative assignment membership, the manifest evidence is reconstructed from the unit assignments.
  - Without authoritative unit evidence, `directoryReductionDecisionForFiles` may adjust `scopePath`, and content-shape evidence is merged into recognition graph context.
  - Manifest `EvidenceJSON` currently uses `{"scheme":"content_shape"}`.
- `internal/library/directory_metadata_resolution.go`
  - `buildDirectoryMetadataResolutionUnit` uses the manifest scope path as the directory scope.
  - `loadDirectoryResolutionShape` first looks up the latest `ContentShapePlan` for that scope path, then falls back to a `directory_reduction` decision, then path inference.
  - Remote-search eligibility and primary metadata item selection are driven by the resulting shape string.
- `internal/library/workflow.go`
  - Directory metadata resolution units are loaded from recognition units after materialization, so unit scope and shape flow into post-resolution work.
- `internal/catalog/hierarchical_browse.go`
  - Browse grouping reads `ContentShapePlan` rows as directory-shape hints for presentation and should remain evidence-oriented after scope decisions become authoritative.

Implication for this change: once metadata scope decisions exist, recognition units and directory metadata resolution need a scope-decision branch before the current plan-as-root path. Content shape plans remain useful as leaf evidence but cannot be the sole final metadata root when a scope decision claims the leaf.

## 1.2 Recognition directory reduction behaviors to port

`internal/library/recognition_directory_reduction.go` is the existing sibling residual reducer. It currently acts as a competing authority in manifest construction and directory resolution:

- Groups sibling movie versions by shared movie identity and version residuals.
- Groups sibling episode versions by shared episode tuple and variant residuals.
- Detects movie collection or series-like leftover structure from grouped residual files.
- Emits `directory_reduction` context evidence for recognition manifest graph construction.
- Persists `ClassificationDecision` rows with `DecisionType = directory_reduction`.
- Adjusts manifest `scopePath` through `directoryReductionScopePath`.
- Excludes extras and attachment-only files with reasons such as `directory_reduction_extras` and `directory_reduction_attachments`.
- Preserves review subtypes for cross-identity conflict, ambiguous series-vs-collection, single-work noise, extras-mixed, and attachment-only layouts.
- Feeds diagnostics in `internal/ingest/diagnostics.go`.

Behaviors that must be ported before disabling it:

- Movie version sibling grouping.
- Episode version sibling grouping.
- Movie collection detection from distinct title/year identities.
- Series-like grouping from multiple episode parents.
- Mixed movie/episode conflict review.
- Ambiguous series-vs-collection review.
- Attachment-only and extras-mixed review/exclusion evidence.
- Diagnostic review subtype explanations.

After parity tests cover these cases, the reducer should be limited to diagnostics or disabled as a materialization authority whenever metadata scope decisions are enabled.

## 1.3 Version and fingerprint inputs

Current constants and derived fingerprints that must change or be included when leaf/scope logic changes:

- `internal/library/content_shape_config.go`
  - `ContentShapeClassifierVersion = "content-shape-v6"`.
  - The active `contentShapeSettingsFromConfig` exposes this classifier version to profiles, plans, assignments, file signals, recognition manifests, and recognition units.
- `ContentShapeProfile` fingerprint
  - Built by `contentShapeDirectoryFingerprint`.
  - Existing tests show the fingerprint includes classifier version, scan policy/exclusion rules, directory snapshot facts, inventory file fingerprints, filename signal facts, sidecar file fingerprints, and visible video counts.
- `ContentShapePlan`
  - Reuse is keyed by library/provider/root/directory/classifier version plus exact fingerprint.
  - The unique index is library/provider/root/directory/classifier version.
- `ContentShapeAssignment`
  - Assignment rows carry classifier version and evidence JSON; the unique index is provider/path, so save updates existing assignment rows when regenerated.
- `RecognitionUnit`
  - `recognitionUnitFingerprint` includes plan ID, plan fingerprint, shape, classifier version, unit key, parent key, review state, file facts, assignment facts, signal facts, and sidecar facts.
  - `SourcePlanFingerprint` connects the unit to content-shape plan invalidation.
- `RecognitionManifest`
  - `persistRecognitionManifestWithInputs` uses the content-shape classifier version in `ManifestScope`.
  - When called from a recognition unit, the unit fingerprint becomes the manifest fingerprint.
  - Otherwise `newRecognitionFingerprint` currently only hashes provider/path/stable identity key.
- `DirectoryMetadataResolutionPayload`
  - Built from manifest/unit/materialization output and current directory shape lookup.
  - It does not yet have a scope classifier version or scope fingerprint in the payload.

New implementation should introduce a separate metadata scope classifier version, include leaf classifier version and child leaf fingerprints in scope fingerprints, and ensure scope changes invalidate recognition units and directory metadata resolution payloads that were derived from the old content-shape-only root.

## 1.4 Spider-Noir fixture note

Failing layout:

```text
Spider-Noir (2026)/
  1080p彩版/
    Spider-Noir.S01E01.mkv
    ...
    Spider-Noir.S01E08.mkv
  4K彩版/
    Spider-Noir.S01E01.mkv
    ...
    Spider-Noir.S01E08.mkv
  4K黑白版/
    Spider-Noir.S01E01.mkv
    ...
    Spider-Noir.S01E08.mkv
```

Current expected failure:

- Each version child can be classified locally as an `episode_pack`.
- The parent has no direct videos, so current leaf-local planning does not naturally create one parent metadata root.
- Downstream work may materialize each child version directory as its own root or depend on residual reduction rather than a persisted scope decision.

Expected scope outcome:

- Leaf summaries:
  - `1080p彩版`, `4K彩版`, and `4K黑白版` are direct-file-only `episode_pack` leaves.
  - Each leaf has the same dominant series identity and season/episode set `S01E01` through `S01E08`.
  - Each leaf has a distinct version signature from directory/title residuals.
- Metadata scope decision:
  - `scope_path`: `Spider-Noir (2026)`.
  - `root_kind`: `series`.
  - `layout`: `series/versioned_episode_packs`.
  - Child roles:
    - `1080p彩版`: version child, version signature `1080p/color`.
    - `4K彩版`: version child, version signature `2160p/color`.
    - `4K黑白版`: version child, version signature `2160p/black_white`.
  - Covered files: all eight episode files in each version folder.
  - Downstream materialization: one series hierarchy with one episode metadata item per episode number and three resource versions attached to each matching episode item.
