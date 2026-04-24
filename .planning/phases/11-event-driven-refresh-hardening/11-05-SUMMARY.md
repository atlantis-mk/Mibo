---
phase: 11-event-driven-refresh-hardening
plan: 05
subsystem: backend-database-listener
tags: [go, gorm, sqlite, jobs, listener, reconciliation]

requires:
  - phase: 11-event-driven-refresh-hardening
    provides: Listener debounce, reconcile coverage, worker dispatch, and OpenList root validation from plans 11-01 through 11-04.
provides:
  - Durable active listener intent guard table keyed by library-level refresh and reconcile intent keys.
  - Atomic storage-event refresh creation that coalesces concurrent bursts into one active listener job.
  - Atomic reconcile coverage seeding that keeps one queued/running reconcile job per library.
affects: [listener, jobs, worker, database]

tech-stack:
  added: []
  patterns:
    - GORM upsert with unique intent guard rows for active listener work.
    - Listener job history remains append-compatible because active uniqueness is separate from job_key.

key-files:
  created:
    - .planning/phases/11-event-driven-refresh-hardening/11-05-SUMMARY.md
  modified:
    - mibo-media-server/internal/database/models.go
    - mibo-media-server/internal/database/database.go
    - mibo-media-server/internal/listener/service.go
    - mibo-media-server/internal/listener/service_test.go

key-decisions:
  - "Use JobActiveIntent.IntentKey for active listener uniqueness instead of making database.Job.JobKey globally unique."
  - "Keep listener refresh and reconcile guards keyed at library scope so completed historical jobs can remain in the jobs table."

patterns-established:
  - "Active listener intents: insert/update a unique guard row before creating or reusing queued/running listener jobs."
  - "Historical job compatibility: active duplicate prevention belongs in JobActiveIntent, not Job.JobKey uniqueness."

requirements-completed: [LIST-03, LIST-04]

duration: 9min
completed: 2026-04-24
---

# Phase 11 Plan 05: Atomic Active Listener Intent Guards Summary

**Durable active-intent guards now serialize concurrent listener refresh and reconcile creation without making historical job keys unique.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-04-24T10:24:30Z
- **Completed:** 2026-04-24T10:33:36Z
- **Tasks:** 2/2 completed
- **Files modified:** 5

## Accomplishments

- Added concurrent regressions proving 20 duplicate storage events leave one active `apply_storage_event_refresh` row.
- Added concurrent regressions proving 20 reconcile coverage calls leave one active `listener_reconcile` row.
- Added `JobActiveIntent` with a unique `intent_key` and AutoMigrate registration.
- Updated listener refresh, reconcile coverage, and reconcile reseeding paths to upsert and maintain active intent guard rows before creating or reusing jobs.
- Preserved `database.Job.JobKey` as a non-unique indexed field so completed historical listener jobs remain possible.

## Task Commits

1. **Task 1: Add concurrent listener duplicate regressions** — `4c4d521` (test)
2. **Task 2: Implement atomic active-intent guards for listener jobs** — `2b1f020` (feat)

## Files Created/Modified

- `mibo-media-server/internal/database/models.go` — Added `JobActiveIntent` model with unique `IntentKey`.
- `mibo-media-server/internal/database/database.go` — AutoMigrates the active intent guard table after `Job`.
- `mibo-media-server/internal/listener/service.go` — Serializes listener refresh and reconcile active job creation through guard upserts.
- `mibo-media-server/internal/listener/service_test.go` — Adds concurrent storage-event and reconcile coverage regressions.
- `.planning/phases/11-event-driven-refresh-hardening/11-05-SUMMARY.md` — Documents plan execution and verification.

## Decisions Made

- Kept `database.Job.JobKey` non-unique (`gorm:"size:255;index"`) because completed listener history must remain reusable across future windows.
- Used one durable active-intent row per library refresh/reconcile intent (`listener-refresh-active:<library_id>`, `listener-reconcile-active:<library_id>`) to serialize concurrent active job creation.

## Verification

| Command | Result |
|---|---|
| `go test ./internal/listener -run 'Test.*Concurrent.*Intent'` | PASS — TDD GREEN after guard implementation |
| `go test ./internal/listener -run 'Test.*(Concurrent|Merge|Ancestor|Reconcile)'` | PASS |
| `go test ./internal/worker -run 'Test.*Reconcile'` | PASS |
| `go test ./...` | PASS |

## TDD Notes

- RED: `go test ./internal/listener -run 'Test.*Concurrent.*Intent'` failed against the existing check-then-insert flow with SQLite busy errors and duplicate reconcile rows.
- GREEN: The same test passed after adding the active-intent guard table and guarded listener writes.

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Phase 11 gap closure is complete for LIST-03/LIST-04 concurrency hardening. The listener pipeline now has programmed regression coverage for concurrent duplicate refresh and reconcile seeding.

## Self-Check: PASSED

- Found summary and all modified source/test files.
- Found task commits `4c4d521` and `2b1f020` in git history.
- Required listener, worker, and full backend verification commands passed.

---
*Phase: 11-event-driven-refresh-hardening*
*Completed: 2026-04-24*
