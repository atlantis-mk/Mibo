---
phase: 10-scheduled-operations-control
plan: 03
subsystem: api
tags: [metadata, tmdb, trailer, artwork, maintenance]
requires:
  - phase: 10-01
    provides: schedule scope and kind contracts
provides:
  - metadata refetch batch executor
  - trailer sync batch executor
  - artwork refresh batch executor
affects: [worker, web]
tech-stack:
  added: []
  patterns: [metadata-owned batch maintenance]
key-files:
  created: [mibo-media-server/internal/metadata/schedule_jobs.go, mibo-media-server/internal/metadata/schedule_jobs_test.go]
  modified: []
key-decisions:
  - "Trailer sync and artwork refresh stay metadata-owned instead of leaking into worker or HTTP layers."
patterns-established:
  - "Scheduled metadata maintenance operates on persisted media items and existing TMDB helpers."
requirements-completed: [SJOB-02, SJOB-03, SJOB-06]
duration: unknown
completed: 2026-04-24
---

# Phase 10 Plan 03: Metadata Maintenance Executors Summary

**Batch metadata refetch, trailer sync, and artwork refresh executors wired through existing metadata ownership paths**

## Performance
- **Duration:** unknown
- **Started:** 2026-04-24T00:00:00Z
- **Completed:** 2026-04-24T00:00:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Added scoped metadata refetch execution using existing match/refetch helpers.
- Preserved the single-trailer persisted contract during trailer sync.
- Added artwork-only refresh logic that updates poster/backdrop/logo fields without overwriting unrelated metadata.

## Task Commits
1. **Task 1-2: metadata schedule executors** - `e42d668` (feat)

## Files Created/Modified
- `mibo-media-server/internal/metadata/schedule_jobs.go` - batch metadata/trailer/artwork executors
- `mibo-media-server/internal/metadata/schedule_jobs_test.go` - scoped regression coverage for metadata schedule work

## Decisions Made
- Kept artwork refresh intentionally narrower than full refetch so the schedule type identity remains meaningful.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Due schedules can now map to all six locked maintenance kinds through backend-owned executors.

## Self-Check: PASSED
