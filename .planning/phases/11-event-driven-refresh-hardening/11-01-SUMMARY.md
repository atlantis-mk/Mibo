---
phase: 11-event-driven-refresh-hardening
plan: 01
subsystem: backend-listener
tags: [go, gorm, jobs, listener, debounce, reconciliation]

# Dependency graph
requires:
  - phase: 10-scheduled-operations-control
    provides: existing jobs/worker model for background operations
  - phase: 06-stable-identity-incremental-refresh
    provides: targeted refresh and conservative scan reconciliation semantics
provides:
  - listener-domain job contracts for apply_storage_event_refresh and listener_reconcile
  - durable 15-second storage-event coalescing over the existing jobs table
  - six-hour per-library reconciliation coverage seeding
affects: [11-02-storage-event-ingress, 11-03-worker-listener-execution, jobs, library-refresh]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - future-dated queued jobs as listener debounce windows
    - library-bounded safe ancestor promotion for event bursts
    - listener intent converts to existing targeted_refresh or sync_library work only

key-files:
  created:
    - mibo-media-server/internal/listener/service.go
    - mibo-media-server/internal/listener/service_test.go
  modified: []

key-decisions:
  - "Use a fixed 15-second listener merge window persisted in database.Job.available_at."
  - "Use one listener_reconcile queued intent per active library on a six-hour cadence."
  - "Treat unsafe move/rename or unsupported event kinds as full-sync fallback intent instead of guessing a widened targeted root."

patterns-established:
  - "Listener events are persisted as intermediate jobs before scan work is enqueued."
  - "Listener policy remains inside mibo-media-server/internal/listener and does not mutate canonical media rows."

requirements-completed: [LIST-03, LIST-04]

# Metrics
duration: 3min
completed: 2026-04-24
---

# Phase 11 Plan 01: Listener-Domain Refresh Foundation Summary

**Durable listener coalescing with 15-second debounce windows, safe ancestor promotion, and six-hour reconciliation intent per library**

## Performance

- **Duration:** 3 min
- **Started:** 2026-04-24T08:35:26Z
- **Completed:** 2026-04-24T08:39:09Z
- **Tasks:** 2 completed
- **Files modified:** 2 implementation/test files plus planning metadata

## Accomplishments

- Added the `internal/listener` package with stable job kinds, event ingest input, and durable payload contracts for coalesced refresh and reconciliation.
- Locked the listener policy in regression tests: 15-second merge windows, `library_id + normalized_root` coalescing, sibling/nested ancestor promotion, and six-hour reconciliation cadence.
- Implemented persistence on the existing `database.Job` table so listener work queues delayed `apply_storage_event_refresh` jobs and periodic `listener_reconcile` jobs without external infrastructure.
- Added execution methods that convert listener intent into existing `targeted_refresh` or `sync_library` jobs while leaving canonical media mutation to scan/reconcile flows.

## Task Commits

Each TDD task was committed atomically:

1. **Task 1: Define listener job contracts and lock the coalescing policy in tests**
   - `d5c117b` `test(11-01): add failing listener coalescing coverage`
   - `941e118` `feat(11-01): add listener debounce policy foundation`
2. **Task 2: Implement listener service persistence on top of the existing jobs table**
   - `2a8572f` `test(11-01): add failing listener persistence coverage`
   - `6a4fadc` `feat(11-01): persist listener jobs on the existing queue`

**Plan metadata:** committed after this summary with `docs(11-01): complete listener refresh foundation plan`.

## Files Created/Modified

- `mibo-media-server/internal/listener/service.go` - Listener-domain service, job constants, payload contracts, coalescing, reconciliation coverage, and conversion to existing library jobs.
- `mibo-media-server/internal/listener/service_test.go` - Regression coverage for merge windows, ancestor promotion, fallback full sync, queue-only application, and reconcile reseeding.

## Decisions Made

- Used the existing `database.Job` table and future `available_at` timestamps as the durable debounce substrate, avoiding Redis/Kafka/NATS or a second scheduler.
- Kept listener output as intermediate intent (`apply_storage_event_refresh` / `listener_reconcile`) and only later enqueued existing `targeted_refresh` or `sync_library` work.
- Chose conservative fallback for unsafe normalization: when move/rename data is incomplete or an event kind is unsupported, request a full library sync rather than widening guesses beyond safe scope.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. Existing task commits were present and were verified against the plan criteria before creating this summary.

## User Setup Required

None - no external service configuration required.

## Verification

- `cd /root/Mibo/mibo-media-server && go test ./internal/listener -run 'Test.*(Merge|Ancestor|Reconcile)'` — passed.
- `cd /root/Mibo/mibo-media-server && go test ./internal/listener` — passed.
- Confirmed listener code creates listener/library job rows only and does not directly mutate canonical media tables.

## Known Stubs

None.

## TDD Gate Compliance

- RED gate commits exist for both tasks: `d5c117b`, `2a8572f`.
- GREEN gate commits exist after their corresponding RED commits: `941e118`, `6a4fadc`.

## Next Phase Readiness

- Plan 11-02 can route `/api/v1/storage-events` through `listener.RecordStorageEvent` using the service contracts introduced here.
- Plan 11-03 can dispatch `apply_storage_event_refresh` and `listener_reconcile` in the worker by calling `ApplyStorageEventRefresh` and `RunReconcile`.

## Self-Check: PASSED

- Found `mibo-media-server/internal/listener/service.go`.
- Found `mibo-media-server/internal/listener/service_test.go`.
- Found `.planning/phases/11-event-driven-refresh-hardening/11-01-SUMMARY.md`.
- Found task commits `d5c117b`, `941e118`, `2a8572f`, and `6a4fadc` in git history.

---
*Phase: 11-event-driven-refresh-hardening*
*Completed: 2026-04-24*
