## 1. Storage Index Data Model

- [x] 1.1 Add database models and migrations for persistent storage index rows keyed by library, provider, and normalized path.
- [x] 1.2 Add indexes for library/status/path lookup, stable identity lookup, and pending observation queries.
- [x] 1.3 Add storage index service methods to upsert present observations, mark missing observations, load scoped index rows, and record provider observation failures.
- [x] 1.4 Add unit tests for index upsert, missing marking, duplicate prevention, and scoped path queries.

## 2. Provider Observation Support

- [x] 2.1 Extend local storage object mapping to include stable local identity from device and inode where the platform exposes it.
- [x] 2.2 Add provider-neutral observation DTOs that map `storage.Object` data into storage index fields.
- [x] 2.3 Implement OpenList polling observation using existing provider `List`/`Get` behavior without modifying upstream OpenList.
- [x] 2.4 Implement local reconciliation observation by walking local provider directories through the storage provider boundary.
- [x] 2.5 Add tests for OpenList observation mapping, local identity mapping, and provider listing failure handling.

## 3. Diff Planner

- [x] 3.1 Implement planner classification for create, update, delete, directory delete, move/rename, and uncertain changes.
- [x] 3.2 Implement stable-identity move detection and low-confidence fallback to delete-plus-create.
- [x] 3.3 Implement file stability checks that defer scans for actively changing files.
- [x] 3.4 Implement scope planning for file parent, directory scope, common ancestor, and full-library fallback.
- [x] 3.5 Add unit tests covering new files, changed files, deleted files, deleted directories, stable-identity renames, weak heuristic fallback, and dispersed changes.

## 4. Listener and Job Integration

- [x] 4.1 Add jobs or extend listener jobs to run storage observations and planner-generated refresh plans asynchronously.
- [x] 4.2 Route external `POST /api/v1/storage-events` hints through the storage index and planner path while preserving request validation.
- [x] 4.3 Ensure planner output enqueues `QueueTargetedRefresh` or `QueueLibraryScan` instead of directly writing catalog state.
- [x] 4.4 Preserve existing listener debounce behavior and coalesce planner work by library and common ancestor.
- [x] 4.5 Add worker tests proving planned changes enqueue targeted refreshes or full scans as expected.

## 5. Local Change Source

- [x] 5.1 Add a local observer manager that registers active local libraries and handles start, stop, and library changes.
- [x] 5.2 Add recursive local file-system event handling for create, write, remove, and rename hints.
- [x] 5.3 Add fallback behavior when recursive watching fails or OS watcher limits are reached.
- [x] 5.4 Add tests for local event normalization and watcher-unavailable fallback using test doubles where direct watcher tests are impractical.

## 6. OpenList Change Source

- [x] 6.1 Add per-library OpenList polling scheduling with conservative defaults and reconciliation coverage.
- [x] 6.2 Add selective refresh behavior for suspected changed OpenList directories without forcing every poll to refresh upstream cache.
- [x] 6.3 Add tests for polling diff create, update, delete, and stable-identity move scenarios.
- [x] 6.4 Verify OpenList polling does not import or modify code under `OpenList/`.

## 7. Diagnostics and Settings

- [x] 7.1 Add backend status data for observer mode, last successful observation, last reconcile, pending plan count, and recent failure summary per library.
- [x] 7.2 Expose storage change diagnostics through an admin or scan settings API endpoint.
- [x] 7.3 Add frontend display for automatic detection status if scan settings UI already has a suitable placement; otherwise keep API-only diagnostics.
- [x] 7.4 Add tests for diagnostics responses and disabled-observer fallback reporting.

## 8. Verification and Rollout

- [x] 8.1 Add integration tests for create, delete, and stable-identity rename flowing from observation through scan refresh and playback path update.
- [x] 8.2 Add regression tests that scanner cleanup, sidecar handling, metadata matching, probe jobs, and projection refreshes still run through existing scan paths.
- [x] 8.3 Run `go test ./...` from `mibo-media-server/` and fix failures related to this change.
- [x] 8.4 Run frontend typecheck/build only if diagnostics UI or API client code is changed.
- [x] 8.5 Document rollout defaults, rollback by disabling observers, and continued availability of manual and scheduled scans.
