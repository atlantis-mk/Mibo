## 1. Resolution Unit Model

- [x] 1.1 Add an internal directory metadata resolution payload with library ID, scope path, shape, review state, metadata IDs, resource IDs, and remote-search eligibility fields.
- [x] 1.2 Build resolution units from recognition materialization output and persisted content-shape/directory reduction evidence.
- [x] 1.3 Add focused tests for unit generation from single movie, multipart movie, movie versions, series/season, movie collection, extras-only, and ambiguous directory fixtures.

## 2. Workflow Queueing

- [x] 2.1 Add a workflow task handler for directory-scoped metadata resolution or version the existing metadata match task payload without breaking item-level callers.
- [x] 2.2 Route post-recognition queueing through directory metadata units instead of flat `metadataItemIDs` when directory evidence is available.
- [x] 2.3 Preserve the existing item-level metadata match queue for manual, refetch, legacy, and direct metadata item operations.
- [x] 2.4 Add workflow tests proving episode metadata items are not independently queued for series or season directories.

## 3. Directory Resolution Execution

- [x] 3.1 Implement single movie, multipart movie, and movie-version directory execution so one movie work is resolved and all resources remain linked to it.
- [x] 3.2 Implement series, season, episode-pack, absolute episode-pack, and flat episode directory execution so series is resolved once and episodes are bound from hierarchy/local numbering.
- [x] 3.3 Implement movie collection execution so clear movie identities are handled within one directory scope and local provisional metadata is kept when no search provider exists.
- [x] 3.4 Suppress automatic provider search for attachment-only, extras-only, ambiguous, mixed-conflict, and review-required directories while recording a skip/review outcome.

## 4. Metadata Evidence and Governance

- [x] 4.1 Extend metadata operation evidence to include directory scope, directory shape, skip reason, and per-identity outcomes where relevant.
- [x] 4.2 Ensure local scan evidence, sidecar images, external IDs, and fallback posters still apply during directory-scoped resolution.
- [x] 4.3 Verify manual apply and refetch continue to work against individual metadata items created by directory-scoped resolution.

## 5. Projection and Regression Coverage

- [x] 5.1 Ensure catalog projection refresh runs after successful, skipped, and review-required directory resolution outcomes.
- [x] 5.2 Add end-to-end workflow tests for single movie, movie versions, series folder, season folder, movie collection without search providers, and ambiguous review-required directories.
- [x] 5.3 Run `cd mibo-media-server && go test ./...` and address any workflow, recognition, metadata, or projection regressions.
