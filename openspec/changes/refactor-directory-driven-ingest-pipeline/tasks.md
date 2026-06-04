## 1. Data Model And Migration

- [x] 1.1 Add directory snapshot models for library/provider/root/directory fingerprints, normalized object summaries, visible media paths, child directories, scan run lineage, and changed state.
- [x] 1.2 Add recognition unit models for unit key, directory shape, source plan, file membership, assignment membership, parent unit, fingerprint, status, and last materialized fingerprint.
- [x] 1.3 Add materialization result models or fields for unit metadata IDs, resource IDs, file IDs, projection IDs, enrichment eligibility, skip reasons, and warnings.
- [x] 1.4 Add indexes for snapshot lookup by library/provider/root/directory/version, recognition unit lookup by library/unit key/status/fingerprint, and enrichment lookup by affected target.
- [x] 1.5 Add migration tests covering fresh schema creation and migration from existing inventory, content shape, recognition, metadata, and resource tables.

## 2. Snapshot Stage

- [x] 2.1 Implement a directory snapshot repository with upsert, load latest, mark unchanged, mark changed, and affected-subtree query methods.
- [x] 2.2 Refactor storage traversal so scan tasks persist snapshots before inventory or recognition work runs.
- [x] 2.3 Compute snapshot fingerprints from normalized object summaries, visible media paths, child directories, scanner version, and relevant provider identity fields.
- [x] 2.4 Implement unchanged-directory skipping with forced refresh override and scan policy/exclusion invalidation inputs.
- [x] 2.5 Add tests for changed, unchanged, deleted, targeted refresh, and forced refresh directory snapshots.

## 3. Inventory And Signal Stages

- [x] 3.1 Implement inventory sync from changed directory snapshots using batched inventory file upserts and missing-file marking.
- [x] 3.2 Ensure sidecar and non-video inventory facts are derived from snapshots without relisting provider contents.
- [x] 3.3 Implement file signal hydration as a distinct stage that loads existing valid signals and only parses missing or invalidated signals.
- [x] 3.4 Add signal stage fingerprints that include classifier version and relevant signal inputs.
- [x] 3.5 Add tests proving inventory sync and signal hydration reuse persisted snapshot and signal data.

## 4. Directory Shape Stage

- [x] 4.1 Refactor content shape planning to consume snapshots, inventory facts, file signals, and scan rules as explicit inputs.
- [x] 4.2 Persist content shape profiles, plans, and assignments as the authoritative directory type and file assignment outputs.
- [x] 4.3 Add invalidation so content shape plans rerun only when snapshot, inventory, signal, scan policy, or exclusion fingerprints change.
- [x] 4.4 Add tests for movie folder, movie versions folder, movie collection folder, season folder, episode pack, flat episode folder, attachment group, and review-required plans.

## 5. Recognition Units

- [x] 5.1 Implement recognition unit construction from content shape plans and assignments.
- [x] 5.2 Map movie folder, movie versions folder, movie collection folder, episodic folder, attachment group, and review-required shapes to stable recognition unit structures.
- [x] 5.3 Compute recognition unit fingerprints from shape plan, assignments, file IDs, file signals, sidecar evidence, and classifier version.
- [x] 5.4 Replace ad hoc recognition resolve grouping with recognition unit scheduling while preserving compatibility helpers for legacy calls.
- [x] 5.5 Add tests for recognition unit grouping, stable unit keys, fingerprint changes, and unchanged unit skips.

## 6. Manifest And Materialization

- [x] 6.1 Build recognition manifests from recognition unit inputs instead of raw file batches.
- [x] 6.2 Preserve existing candidate keys, canonical keys, metadata sort keys, resource keys, and recognition decision behavior.
- [x] 6.3 Refactor materialization to consume recognition units and record per-unit materialization results.
- [x] 6.4 Skip manifest rebuild and materialization when a recognition unit fingerprint matches the last successful materialized fingerprint.
- [x] 6.5 Add tests for movie, movie versions, movie collection, series/season/episode, multi-episode, multipart, supplemental, review-required, and unchanged-unit materialization.

## 7. Enrichment And Projection Coordination

- [x] 7.1 Refactor metadata match scheduling to consume materialization results and directory/unit eligibility instead of recomputing target IDs from manifests.
- [x] 7.2 Refactor probe scheduling to consume materialization file/resource outputs and library probe policy.
- [x] 7.3 Implement enrichment skip records for review-required, attachment-only, extras-only, provider-unavailable, and mixed-conflict units.
- [x] 7.4 Coalesce workflow tasks by normalized enrichment target and retain contributing unit lineage.
- [x] 7.5 Schedule projection refresh once per affected library scope after materialization and enrichment planning complete.
- [x] 7.6 Add tests for duplicate enrichment coalescing, projection coalescing, skip reasons, and remote provider availability checks.

## 8. Workflow Integration

- [x] 8.1 Add staged workflow task types or refactor existing handlers for snapshot, inventory sync, signal hydration, directory shape, recognition unit build, unit materialization, enrichment planning, and projection.
- [x] 8.2 Route library creation scans, manual scans, scheduled scans, storage refreshes, and targeted refreshes through the staged pipeline.
- [x] 8.3 Keep legacy scan/materialization handlers callable during rollout for compatibility and fallback.
- [x] 8.4 Add workflow dependency and task-key tests proving stages run in order and duplicate task creation is avoided.

## 9. Diagnostics And Compatibility

- [x] 9.1 Add internal query helpers that connect a directory snapshot to inventory files, signals, content shape plan, recognition units, materialization results, enrichment tasks, and projection refreshes.
- [x] 9.2 Add diagnostics or admin-facing service methods for directory pipeline lineage and skip reasons.
- [x] 9.3 Verify existing library create, list, browse, media detail, playback, metadata governance, probe, and projection APIs remain compatible.
- [x] 9.4 Update relevant docs or developer notes describing the staged pipeline and durable intermediate outputs.

## 10. End-To-End Verification

- [x] 10.1 Add end-to-end tests for creating and scanning a new library through the staged pipeline.
- [x] 10.2 Add end-to-end tests for repeated unchanged scans that skip downstream stages.
- [x] 10.3 Add end-to-end tests for targeted refresh invalidating only affected snapshots, units, enrichment, and projection scopes.
- [x] 10.4 Run `cd mibo-media-server && go test ./...`.
- [x] 10.5 Run `cd frontend && pnpm test` if frontend contracts or visible behavior change.
