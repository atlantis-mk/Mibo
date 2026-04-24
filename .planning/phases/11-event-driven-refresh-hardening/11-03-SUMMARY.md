---
phase: 11-event-driven-refresh-hardening
plan: 03
subsystem: infra
tags: [worker, jobs, listener, reconciliation, targeted-refresh]
requires:
  - phase: 11-01
    provides: listener-domain service with refresh and reconcile job contracts
  - phase: 11-02
    provides: storage event ingress routed through listener jobs
provides:
  - worker dispatch for coalesced listener refresh jobs
  - periodic listener reconciliation coverage seeding for active libraries
  - due listener reconciliation execution through the existing queue lifecycle
affects: [backend, worker, listener, library-refresh]
tech-stack:
  added: []
  patterns: [listener-triggered work fans into existing jobs instead of direct scan mutation]
key-files:
  created: []
  modified: [mibo-media-server/internal/worker/worker.go, mibo-media-server/internal/worker/worker_test.go]
key-decisions:
  - "Listener worker branches delegate to the listener service and only enqueue existing scan jobs."
  - "Reconciliation coverage is seeded before normal job claiming so active libraries keep a future fallback scan intent."
patterns-established:
  - "Listener refresh work remains queue-driven: apply_storage_event_refresh becomes targeted_refresh or sync_library work."
  - "Listener reconciliation remains library-scoped and self-reseeding on the six-hour service cadence."
requirements-completed: [LIST-02, LIST-04]
duration: 13min
completed: 2026-04-24
---

# Phase 11 Plan 03: Listener-Aware Worker Dispatch Summary

**Worker dispatch now turns coalesced listener jobs into existing scan queue work and keeps active libraries covered by self-reseeding reconciliation jobs**

## Performance

- **Duration:** 13 min
- **Started:** 2026-04-24T09:00:47Z
- **Completed:** 2026-04-24T09:13:31Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Injected the listener service into the worker runner and dispatched `apply_storage_event_refresh` jobs through `listener.ApplyStorageEventRefresh`.
- Preserved the single queue-driven execution path by asserting listener refresh jobs enqueue `targeted_refresh` or `sync_library`, not direct canonical row mutations.
- Added worker-owned reconciliation seeding before normal queue claims and delegated due `listener_reconcile` jobs through `listener.RunReconcile` for sync fan-out plus future reseeding.

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: storage event worker dispatch coverage** - `14a73a7` (test)
2. **Task 1 GREEN: dispatch storage event listener jobs** - `880b493` (feat)
3. **Task 2 RED: listener reconcile worker coverage** - `310fe55` (test)
4. **Task 2 GREEN: maintain listener reconcile coverage** - `423c04a` (feat)

_Note: Both tasks used TDD test → feat commits._

## Files Created/Modified

- `mibo-media-server/internal/worker/worker.go` - listener dependency injection, refresh dispatch, reconcile coverage seeding, and reconcile dispatch.
- `mibo-media-server/internal/worker/worker_test.go` - regression coverage for storage event fan-out and future-dated reconcile reseeding.

## Decisions Made

- Kept listener-triggered work in the existing worker queue lifecycle, preserving `targeted_refresh` / `sync_library` as the only scan execution paths.
- Kept reconciliation fallback library-scoped and worker-owned instead of introducing a second scheduler, UI workflow, or direct listener mutation path.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Verification

- `cd /root/Mibo/mibo-media-server && go test ./internal/worker -run 'Test.*(StorageEvent|TargetedRefresh|Reconcile)'` — passed.
- `cd /root/Mibo/mibo-media-server && go test ./internal/worker` — passed.

## TDD Gate Compliance

- RED gate commits present: `14a73a7`, `310fe55`.
- GREEN gate commits present after RED commits: `880b493`, `423c04a`.
- No refactor commit was needed.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Threat Flags

None - changes only connected plan-covered listener job dispatch and reconciliation seeding surfaces.

## Next Phase Readiness

- Phase 11 worker integration is complete; listener ingress jobs now reach real scan work through existing queue semantics.
- Periodic reconciliation is self-sustaining for active libraries, satisfying the missed-event fallback requirement.

## Self-Check: PASSED

- Confirmed summary, worker source, and worker test files exist.
- Confirmed task commits exist: `14a73a7`, `880b493`, `310fe55`, `423c04a`.

---
*Phase: 11-event-driven-refresh-hardening*
*Completed: 2026-04-24*
