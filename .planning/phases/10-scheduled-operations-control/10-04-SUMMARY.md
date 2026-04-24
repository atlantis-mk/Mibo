---
phase: 10-scheduled-operations-control
plan: 04
subsystem: api
tags: [httpapi, auth, schedule, jobs, backend]
requires:
  - phase: 10-01
    provides: schedule domain contracts
  - phase: 10-02
    provides: library schedule executors
  - phase: 10-03
    provides: metadata schedule executors
provides:
  - authenticated schedule CRUD endpoints
  - run-now endpoint returning async job feedback
  - schedule history and detail routes
affects: [worker, web]
tech-stack:
  added: []
  patterns: [thin schedule handlers, authenticated schedule-first api envelope]
key-files:
  created: [mibo-media-server/internal/httpapi/handlers_schedules.go]
  modified: [mibo-media-server/internal/app/app.go, mibo-media-server/internal/httpapi/router.go, mibo-media-server/internal/httpapi/router_test.go, mibo-media-server/internal/schedule/service.go]
key-decisions:
  - "Router wiring falls back to a default schedule service but accepts injected wiring from app setup."
patterns-established:
  - "Run-now creates schedule runs and returns queued job metadata immediately instead of blocking HTTP requests."
requirements-completed: [SJOB-01, SJOB-02, SJOB-03, SJOB-04, SJOB-05, SJOB-06, SJOB-07, SJOB-08]
duration: unknown
completed: 2026-04-24
---

# Phase 10 Plan 04: Schedule HTTP API Summary

**Authenticated schedule CRUD, history, and run-now APIs exposed over the existing jobs-based execution path**

## Performance
- **Duration:** unknown
- **Started:** 2026-04-24T00:00:00Z
- **Completed:** 2026-04-24T00:00:00Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Extended the schedule service with job-backed run-now orchestration.
- Added authenticated list/create/get/update/toggle/run/history endpoints.
- Added endpoint regression tests for auth, validation, list/history serialization, and run-now behavior.

## Task Commits
1. **Task 1: add run-now orchestration in schedule service** - `571f8b0` (feat)
2. **Task 2: expose authenticated schedule APIs** - `549b19a` (feat)

## Files Created/Modified
- `mibo-media-server/internal/schedule/service.go` - run-now and job payload support
- `mibo-media-server/internal/httpapi/handlers_schedules.go` - schedule CRUD/history handlers
- `mibo-media-server/internal/httpapi/router.go` - schedule route registration
- `mibo-media-server/internal/httpapi/router_test.go` - schedule endpoint tests
- `mibo-media-server/internal/app/app.go` - app wiring for the schedule service

## Decisions Made
- Kept schedule routes separate from `/api/v1/jobs` so clients consume schedule-first contracts directly.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The worker can now consume schedule-managed jobs through authenticated APIs and run-now triggers.

## Self-Check: PASSED
