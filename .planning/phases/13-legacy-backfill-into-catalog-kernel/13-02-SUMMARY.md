---
phase: 13-legacy-backfill-into-catalog-kernel
plan: 02
subsystem: api
tags: [go, catalog, jobs, worker, auth, migration]
requires:
  - phase: 13-legacy-backfill-into-catalog-kernel
    provides: durable backfill run/report contracts and typed catalog migration DTOs
provides:
  - authenticated trigger and report endpoints for legacy catalog backfill runs
  - worker dispatch for queued catalog_backfill_legacy jobs
  - active job-key reuse for all-library and library-scoped backfill queue requests
affects: [catalog-backfill, worker, operations, migration-reporting]
tech-stack:
  added: []
  patterns: [typed backfill worker payloads, auth-first admin migration handlers, job-key dedupe before new run creation]
key-files:
  created:
    - mibo-media-server/internal/worker/worker_catalog_backfill_test.go
    - mibo-media-server/internal/httpapi/handlers_catalog_migration.go
    - mibo-media-server/internal/httpapi/catalog_migration_backfill_router_test.go
  modified:
    - mibo-media-server/internal/catalog/backfill.go
    - mibo-media-server/internal/worker/worker.go
    - mibo-media-server/internal/app/app.go
    - mibo-media-server/internal/httpapi/router.go
key-decisions:
  - "Reuse queued or running backfill jobs by job_key before creating a duplicate operator-visible run."
  - "Return typed LegacyBackfillRun DTOs from queue/list/detail APIs instead of exposing raw jobs rows to operators."
  - "Validate library-scoped worker payloads against the persisted run scope before advancing run lifecycle state."
patterns-established:
  - "Backfill control-plane endpoints live in internal/httpapi while execution remains in catalog service + worker."
  - "Operator-triggered migration jobs should round-trip through the existing jobs.EnqueueUnique flow with stable scope keys."
requirements-completed: [MIGR-01, MIGR-02]
duration: 12 min
completed: 2026-04-25
---

# Phase 13 Plan 02: Legacy backfill trigger and worker dispatch summary

**Authenticated legacy backfill trigger/report APIs now queue durable catalog migration runs through the worker and reuse active scope-specific jobs instead of spawning duplicate work.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-04-25T07:23:58Z
- **Completed:** 2026-04-25T07:36:28Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Added worker dispatch for `catalog_backfill_legacy` so queued backfill jobs now execute through the existing jobs/worker pipeline.
- Added authenticated `POST /api/v1/catalog-migration/backfill`, `GET /api/v1/catalog-migration/runs`, and `GET /api/v1/catalog-migration/runs/{id}` endpoints returning typed backfill run/report DTOs.
- Reused active scope-specific jobs before creating new work so repeated all-library or per-library triggers collapse onto the existing queued/running backfill.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add worker dispatch for queued legacy backfill runs**
   - `b140138` `test(13-02): add failing legacy backfill worker coverage`
   - `31ca211` `feat(13-02): dispatch queued legacy backfill jobs`
2. **Task 2: Add authenticated backfill trigger and report endpoints**
   - `97eab92` `test(13-02): add failing catalog backfill endpoint coverage`
   - `82fb01b` `feat(13-02): add catalog backfill trigger and report APIs`

## Files Created/Modified

- `mibo-media-server/internal/catalog/backfill.go` - Aligns the queued backfill payload with the HTTP/worker contract and adds run lifecycle execution helpers.
- `mibo-media-server/internal/worker/worker.go` - Dispatches `catalog_backfill_legacy` jobs to the catalog service.
- `mibo-media-server/internal/worker/worker_catalog_backfill_test.go` - Proves queued backfill jobs complete and advance durable run state.
- `mibo-media-server/internal/app/app.go` - Passes the catalog service into HTTP router wiring.
- `mibo-media-server/internal/httpapi/router.go` - Registers authenticated backfill trigger and report routes plus catalog service injection.
- `mibo-media-server/internal/httpapi/handlers_catalog_migration.go` - Implements queue/list/detail handlers, scope validation, and active-job reuse.
- `mibo-media-server/internal/httpapi/catalog_migration_backfill_router_test.go` - Covers auth, queueing, dedupe, and run/report reads.

## Decisions Made

- Reused active queued/running jobs by `job_key` before creating a new run so operators do not accidentally fork duplicate backfill work for the same scope.
- Returned `LegacyBackfillRun` DTOs from the trigger endpoint as well as list/detail endpoints so operators inspect one consistent report shape instead of mixing run and job schemas.
- Kept the handler responsible for auth, scope validation, and job enqueueing while the catalog service owns run lifecycle transitions used by the worker.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added a catalog backfill execution entrypoint and aligned the queued payload contract**
- **Found during:** Task 1 (Add worker dispatch for queued legacy backfill runs)
- **Issue:** The worker branch had no catalog service entrypoint to delegate to, and the committed `LegacyBackfillPayload` shape did not match the plan interface needed by the HTTP trigger and worker tests.
- **Fix:** Updated `catalog.LegacyBackfillPayload` to `{run_id, library_id}`, added `RunLegacyBackfill`, and validated worker payload scope against the persisted run before finalizing the run lifecycle.
- **Files modified:** `mibo-media-server/internal/catalog/backfill.go`
- **Verification:** `cd mibo-media-server && go test ./internal/worker -run 'TestRunOnce.*CatalogBackfill' -count=1`
- **Committed in:** `31ca211`

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** The fix was required to make worker dispatch executable and to keep the HTTP/worker payload contract consistent with the plan. No extra product scope was added.

## Issues Encountered

- `mibo-media-server/internal/app/app.go`, `mibo-media-server/internal/httpapi/router.go`, and `mibo-media-server/internal/worker/worker.go` already had unrelated dirty-worktree edits, so task commits were staged with snapshot/reverse-patch isolation to avoid clobbering user changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Later movie and series backfill plans can now reuse durable operator-triggered runs instead of inventing one-off execution paths.
- Operators already have authenticated visibility into queued, running, completed, and failed run records before the data-mapping slices land in 13-03 through 13-05.

## Self-Check: PASSED

- FOUND: `.planning/phases/13-legacy-backfill-into-catalog-kernel/13-02-SUMMARY.md`
- FOUND: `b140138`
- FOUND: `31ca211`
- FOUND: `97eab92`
- FOUND: `82fb01b`
