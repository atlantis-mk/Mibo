## 1. Inventory And Test Mapping

- [x] 1.1 Inventory all production and test references to `internal/scanrecognition`, separating parser/helper usage from tree-classifier and recognition-adapter usage.
- [x] 1.2 Map each old classifier fixture to a target `content_shape` profile, plan, assignment, recognition-unit, or materialization test.
- [x] 1.3 Add failing `content_shape` tests for large movie collections, catalog-id collections, movie versions, multipart movies, token-consensus cases, sidecar conflicts, primary-video filtering, and season-only series parents.
- [x] 1.4 Add regression coverage for the current large directory case that previously produced `unknown_review` and left files organizing.

## 2. Shared Parsing And Evidence Inputs

- [x] 2.1 Move reusable filename/folder parsing helpers that are still needed out of the old tree-classifier path or duplicate them as `library`-owned helpers.
- [x] 2.2 Extend `contentShapeDirectoryProfile` with normalized movie identity distribution, dominant identity counts, version identity counts, primary video counts, supplemental video counts, multipart group coverage, token-consensus evidence, sidecar shape hints, conflict flags, and child shape summaries.
- [x] 2.3 Ensure `content_shape` fingerprints include all relevant inventory, file signal, scan policy, exclusion, sidecar/NFO, and child-shape inputs.
- [x] 2.4 Persist diagnostic evidence fields needed to explain selected shapes and review-required conflicts.

## 3. Planning Rules

- [x] 3.1 Implement conflict-first planning for contradictory movie/episode sidecar, filename, and folder evidence.
- [x] 3.2 Implement movie version planning from normalized same-work identity and version/edition/release residual evidence.
- [x] 3.3 Implement multipart movie planning with same-work grouping, continuous part ordering, duplicate/missing part rejection, and safe review fallback.
- [x] 3.4 Implement token-consensus planning for version folders, multipart folders, and episodic folders when direct parsing is weak.
- [x] 3.5 Implement primary-video-first shape decisions and attachment-group planning for extras-only directories.
- [x] 3.6 Implement season-only child aggregation so parents with only season-folder children become series folders.
- [x] 3.7 Keep and harden large unique-title and catalog-id movie collection rules so they run only after stronger conflict, episodic, multipart, and version rules.

## 4. Assignments And Materialization

- [x] 4.1 Update `content_shape` assignments for movie versions so variants/resources attach to one movie work instead of becoming separate works.
- [x] 4.2 Update multipart assignments so part files materialize as one multipart playable resource with stable part ordering.
- [x] 4.3 Update collection assignments for per-file-title and per-catalog-id work keys, including duplicate-title and duplicate-catalog safeguards.
- [x] 4.4 Ensure supplemental videos discovered during primary-video-first planning attach as trailers, samples, extras, or review-required attachments instead of shaping the main work.
- [x] 4.5 Verify recognition-unit construction consumes only `content_shape` plan/assignment outputs for all migrated shapes.

## 5. Legacy Path Removal

- [x] 5.1 Remove production calls that use `scanrecognition.ClassifyTree` or old directory-node kinds to decide materialization shape.
- [x] 5.2 Remove or rewrite `recognition_scan_adapter` code that builds candidates from the old tree classifier.
- [x] 5.3 Delete obsolete tree-classifier code and tests after equivalent `content_shape` tests pass.
- [x] 5.4 Keep only parser/helper code that remains actively used, with package ownership adjusted so no deleted classifier dependency remains.
- [x] 5.5 Bump `ContentShapeClassifierVersion` and update any fixtures or snapshots that assert classifier version values.

## 6. Verification

- [x] 6.1 Run focused backend tests for `internal/library` content shape, assignment, recognition-unit, and materialization behavior.
- [x] 6.2 Run migrated parity tests that cover every removed old classifier fixture.
- [x] 6.3 Run `cd mibo-media-server && go test ./...`.
- [x] 6.4 Run a local scan/rescan diagnostic against representative fixtures: large movie collection, catalog-id collection, movie versions, multipart movie, episodic folder, season parent, extras-only folder, and NFO conflict.
- [x] 6.5 Confirm existing organizing-state scenarios now either materialize successfully or expose review-required conflict reasons.
