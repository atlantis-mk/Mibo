---
phase: 06-stable-identity-incremental-refresh
plan: "03"
subsystem: api
tags: [go, worker, jobs, incremental-refresh, scan]
requires:
  - phase: 06-stable-identity-incremental-refresh
    provides: conservative stable-identity scan and post-probe reconciliation
provides:
  - targeted refresh job enqueueing by library path and reason
  - partial scan cleanup confined to the requested subtree
  - worker dispatch coverage for full sync and targeted refresh coexistence
affects: [storage-events, worker, sync-jobs]
tech-stack:
  added: []
  patterns: [scoped partial scan mode, targeted refresh job deduplication]
key-files:
  created: []
  modified:
    - mibo-media-server/internal/library/service.go
    - mibo-media-server/internal/library/scan.go
    - mibo-media-server/internal/worker/worker.go
    - mibo-media-server/internal/worker/worker_test.go
key-decisions:
  - "Deduplicate targeted refresh jobs by normalized library-scoped root path plus refresh reason."
  - "Partial refreshes reuse the scan engine but only soft-delete missing rows inside the requested subtree."
patterns-established:
  - "Targeted refresh is additive alongside full-library sync, not a replacement for scheduled/manual rescans."
  - "Worker dispatch routes both sync_library and targeted_refresh through the same library scan service with different cleanup scope."
requirements-completed: [SYNC-02]
duration: 14min
completed: 2026-04-21
---

# Phase 6 Plan 03: Targeted incremental refresh jobs Summary

**Targeted refresh jobs now dedupe by normalized subtree, reuse the scan engine safely, and leave unrelated library rows untouched.**

## Performance

- **Duration:** 14 min
- **Started:** 2026-04-21T22:57:18Z
- **Completed:** 2026-04-21T23:05:30Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Added a first-class `targeted_refresh` job path with normalized library-scoped roots and deduped job keys.
- Split full-library cleanup from partial subtree cleanup so incremental refreshes cannot soft-delete unrelated rows.
- Preserved existing `sync_library` worker behavior while adding dedicated targeted refresh queue and dispatch coverage.

## Task Commits

1. **Task 1: Add targeted refresh job contracts and scoped scan mode** - `b29296f` (test), `0ec8df6` (feat)
2. **Task 2: Preserve scheduled/manual sync compatibility while adding incremental refresh** - `b246abc` (test)

## Files Created/Modified
- `mibo-media-server/internal/library/service.go` - Added targeted refresh enqueue helper and library-scoped path normalization.
- `mibo-media-server/internal/library/scan.go` - Added partial scan mode with subtree-only missing cleanup.
- `mibo-media-server/internal/worker/worker.go` - Added targeted refresh dispatch alongside the existing full sync path.
- `mibo-media-server/internal/worker/worker_test.go` - Added queue dedupe, subtree-safe refresh, and targeted refresh worker dispatch coverage.

## Decisions Made
- Deduplication keys include normalized target root plus refresh reason so repeated storage triggers collapse safely.
- Partial refresh reuses the existing scan pipeline but switches cleanup scope from whole-library to subtree-local only.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- `internal/jobs/service.go` already provided the required `EnqueueUnique` behavior, so targeted refresh work could be layered on without changing the queue implementation.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- The API boundary can now translate storage events into safe targeted refresh jobs without widening scan cleanup scope.
- Scheduled/manual full-library sync remains intact for fallback scenarios and broader maintenance runs.

## Self-Check: PASSED
