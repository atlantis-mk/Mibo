---
phase: 12-catalog-kernel-contracts-migration-guards
plan: 03
subsystem: api
tags: [catalog, worker, jobs, projections, sqlite, gorm]
requires:
  - phase: 12-catalog-kernel-contracts-migration-guards
    provides: catalog DTO contracts and migration settings groundwork used by projection jobs
provides:
  - Queueable catalog item and library projection refresh contracts
  - Worker dispatch for catalog projection jobs with empty and seeded DB coverage
  - Scan entrypoint hooks that enqueue catalog library projection refreshes alongside legacy reindex jobs
affects: [catalog, scanner, metadata, worker, migration-cutover]
tech-stack:
  added: []
  patterns: [typed worker payloads, targeted projection rebuilds, parallel legacy-and-catalog refresh queuing]
key-files:
  created:
    - mibo-media-server/internal/catalog/projections.go
    - mibo-media-server/internal/catalog/projections_test.go
    - mibo-media-server/internal/worker/worker_catalog_test.go
  modified:
    - mibo-media-server/internal/library/service.go
    - mibo-media-server/internal/library/service_libraries.go
    - mibo-media-server/internal/library/scan_run.go
    - mibo-media-server/internal/worker/worker.go
    - mibo-media-server/internal/app/app.go
    - mibo-media-server/internal/database/database.go
key-decisions:
  - "Keep catalog projection refresh payloads item-scoped or library-scoped so worker jobs stay auditable and bounded."
  - "Rebuild catalog rollups and search documents inside catalog.Service while legacy search reindex remains queued in parallel during migration."
  - "Use a dedicated worker_catalog_test.go file instead of the already-dirty worker_test.go to preserve unrelated worktree changes."
patterns-established:
  - "Projection jobs follow library constants + queue helper + worker dispatch + catalog service execution."
  - "Scan orchestration now fans out both legacy search reindex work and catalog projection refresh work without cutover replacement."
requirements-completed: [PROD-01]
duration: 7m 21s
completed: 2026-04-25
---

# Phase 12 Plan 03: Catalog projection refresh summary

**Catalog projection refresh jobs now rebuild targeted rollups and catalog search documents while scan flows enqueue library-scope refreshes alongside legacy search reindex work.**

## Performance

- **Duration:** 7m 21s
- **Started:** 2026-04-25T13:27:35+08:00
- **Completed:** 2026-04-25T13:34:56+08:00
- **Tasks:** 2
- **Files modified:** 13

## Accomplishments

- Added `catalog_refresh_item_projection` and `catalog_refresh_library_projection` job kinds, typed payloads, queue helpers, and worker dispatch.
- Implemented catalog projection rebuild logic that safely no-ops on empty databases and deterministically rewrites `item_rollups` plus `catalog_search_documents` for targeted scope.
- Wired sync and targeted scan flows to enqueue catalog library projection refreshes without removing the legacy search reindex path.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add catalog projection refresh contract and worker job wiring** - `a49e156` (test), `7261893` (feat)
2. **Task 2: Queue catalog projection refreshes from library scan entrypoints and prove them with worker tests** - `6521965` (test), `afb99e3` (feat)

## Files Created/Modified

- `mibo-media-server/internal/catalog/projections.go` - typed refresh request payloads plus targeted rollup/search projection rebuild logic
- `mibo-media-server/internal/catalog/projections_test.go` - empty and seeded catalog projection coverage
- `mibo-media-server/internal/worker/worker_catalog_test.go` - worker dispatch, seeded rebuild, and scan queue coverage
- `mibo-media-server/internal/library/service.go` - catalog projection job kind constants
- `mibo-media-server/internal/library/service_libraries.go` - queue helpers for item and library projection refresh jobs
- `mibo-media-server/internal/library/scan_run.go` - sync and targeted refresh queue fan-out to catalog projection jobs
- `mibo-media-server/internal/worker/worker.go` - catalog service plumbing and projection job dispatch
- `mibo-media-server/internal/app/app.go` - production worker wiring for catalog projection service
- `mibo-media-server/internal/catalog/service.go` - catalog service foundation required by the new projection methods in current subagent state
- `mibo-media-server/internal/database/catalog_models.go` - catalog projection table models required by the worker contract in current subagent state
- `mibo-media-server/internal/database/inventory_models.go` - inventory model definitions required by the updated database migration path in current subagent state
- `mibo-media-server/internal/database/database.go` - auto-migration coverage for catalog/inventory projection tables
- `mibo-media-server/internal/library/query_series_grouping.go` - blocking helper to keep current subagent-state library compilation intact during worker verification

## Decisions Made

- Kept projection payloads explicitly scoped to item or library boundaries to satisfy the threat-model requirement for bounded refresh work.
- Rebuilt targeted projection rows in the catalog service instead of adding placeholder handlers so later scanner and metadata phases can depend on real behavior now.
- Preserved unrelated dirty changes by placing new worker coverage in `worker_catalog_test.go` rather than editing the already-dirty `worker_test.go`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Committed missing catalog schema and service foundation from current subagent state**
- **Found during:** Task 1 (Add catalog projection refresh contract and worker job wiring)
- **Issue:** The plan assumed catalog service and projection table models already existed, but the current subagent state still had those files uncommitted, which prevented a reproducible projection implementation.
- **Fix:** Included the existing catalog service, catalog/inventory model definitions, database migration wiring, and a small library compile helper in the task implementation commit so the new projection contract builds and runs from committed history.
- **Files modified:** `mibo-media-server/internal/catalog/service.go`, `mibo-media-server/internal/database/catalog_models.go`, `mibo-media-server/internal/database/inventory_models.go`, `mibo-media-server/internal/database/database.go`, `mibo-media-server/internal/library/query_series_grouping.go`
- **Verification:** `go test ./internal/catalog -run 'TestCatalog.*Projection' -count=1`; `go test ./internal/worker -run 'TestRunOnce.*Catalog' -count=1`
- **Committed in:** `7261893`

**2. [Rule 3 - Blocking] Moved new worker coverage into a dedicated test file to preserve dirty worktree changes**
- **Found during:** Task 1 and Task 2 test implementation
- **Issue:** `mibo-media-server/internal/worker/worker_test.go` already had unrelated dirty changes, but the user explicitly required preserving them while still adding projection coverage.
- **Fix:** Added `mibo-media-server/internal/worker/worker_catalog_test.go` with equivalent coverage for worker dispatch, seeded projection rebuilds, and scan queue fan-out.
- **Files modified:** `mibo-media-server/internal/worker/worker_catalog_test.go`
- **Verification:** `go test ./internal/worker -run 'TestRunOnce.*Catalog' -count=1`
- **Committed in:** `a49e156`, `6521965`

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Both fixes were necessary to make the plan executable in the current subagent state while preserving unrelated dirty worktree changes. No functional scope creep beyond the required projection contract.

## Issues Encountered

- The current subagent state included uncommitted catalog/inventory foundation files that the plan expected to already exist.
- Existing dirty changes in `worker_test.go` and related tracked files required isolating new coverage so task commits stayed focused.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Later scanner, metadata, and read-path migration plans can now enqueue durable catalog projection refresh work before cutover.
- Catalog projection jobs are wired through the existing worker model and are safe against empty startup-era databases.

## Known Stubs

None.

## Self-Check: PASSED

- FOUND: `.planning/phases/12-catalog-kernel-contracts-migration-guards/12-03-SUMMARY.md`
- FOUND: `a49e156`
- FOUND: `7261893`
- FOUND: `6521965`
- FOUND: `afb99e3`
