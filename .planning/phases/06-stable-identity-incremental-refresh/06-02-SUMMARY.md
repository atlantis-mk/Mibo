---
phase: 06-stable-identity-incremental-refresh
plan: "02"
subsystem: api
tags: [go, probe, reconciliation, playback-progress, scan]
requires:
  - phase: 06-stable-identity-incremental-refresh
    provides: stable identity evidence fields and provisional fallback candidates
provides:
  - post-probe fallback reconciliation using size and duration
  - ambiguity quarantine for multi-candidate fallback recovery
  - playback progress rebinding on safe continuity recovery
affects: [worker, incremental-refresh, event-intake]
tech-stack:
  added: []
  patterns: [probe-triggered reconciliation, review-needed ambiguity quarantine]
key-files:
  created: [mibo-media-server/internal/library/identity_reconcile_test.go]
  modified:
    - mibo-media-server/internal/probe/service.go
    - mibo-media-server/internal/library/scan.go
key-decisions:
  - "Only a unique size+duration match may reclaim a deleted media identity after probe completes."
  - "Multiple qualifying fallback matches are quarantined with review-needed status instead of guessing."
patterns-established:
  - "Probe completion is the gate for any non-stable-id continuity recovery."
  - "Playback progress follows the replacement file only after a unique high-confidence fallback match."
requirements-completed: [SYNC-01]
duration: 8min
completed: 2026-04-21
---

# Phase 6 Plan 02: Conservative fallback reconciliation and ambiguity quarantine Summary

**Probe-complete fallback reconciliation now restores continuity on unique size+duration matches and quarantines ambiguous candidates for review.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-21T22:48:26Z
- **Completed:** 2026-04-21T22:56:29Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Added probe-triggered reconciliation that reattaches provisional files to the prior media item only after duration is available.
- Restored prior media-item continuity and moved playback progress to the replacement file for safe rename/move recovery.
- Marked multi-candidate fallback matches as `review_needed` while leaving low-confidence cases detached.

## Task Commits

1. **Task 1: Reconcile provisional candidates after probe using D-03 signals** - `2afba28` (test), `d3c6548` (feat)
2. **Task 2: Quarantine ambiguous fallback matches per D-04** - `7f3d0ce` (test), `e6b662a` (feat)

## Files Created/Modified
- `mibo-media-server/internal/library/identity_reconcile_test.go` - Regression coverage for unique-match recovery, no-duration no-op, and ambiguity quarantine.
- `mibo-media-server/internal/probe/service.go` - Triggered fallback reconciliation only after successful probe duration writes.
- `mibo-media-server/internal/library/scan.go` - Added size+duration reconciliation, progress rebinding, and review-needed ambiguity handling.

## Decisions Made
- Size+duration reconciliation only runs after probe persists duration facts.
- Ambiguous fallback matches keep the provisional file separate and preserve prior progress links unchanged.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Worker-facing incremental refresh can now rely on stable/provisional identity states without losing conservative reconciliation behavior.
- Storage-event intake can enqueue targeted work knowing ambiguous fallback matches pause safely for review instead of misbinding continuity.

## Self-Check: PASSED
