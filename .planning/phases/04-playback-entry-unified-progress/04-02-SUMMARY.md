---
phase: 04-playback-entry-unified-progress
plan: 02
subsystem: ui
tags: [react, tanstack-router, playback, progress, route-intent]
requires:
  - phase: 04-playback-entry-unified-progress
    provides: authenticated playback entry and canonical media-item progress semantics
provides:
  - shared playback route intent types for media-item, file, and restart entry
  - validated playback route search for explicit fromStart navigation intent
  - a typed controller helper that routes playback entry through one canonical seam
affects: [04-03, playback-page, media-detail-panel, continue-watching]
tech-stack:
  added: []
  patterns: [typed playback route search, canonical playback navigation helper]
key-files:
  created: [web/src/features/app/types/playback-intent.ts]
  modified: [web/src/router.tsx, web/src/features/app/hooks/use-app-controller.ts, web/src/features/app/pages/playback-page.tsx]
key-decisions:
  - "Represent restart intent as validated route search state instead of introducing a second playback route family."
  - "Keep existing detail CTA signatures working by wrapping them around the new typed playback entry helper."
patterns-established:
  - "All frontend playback entry points can flow through one PlaybackEntryIntent contract before navigating to /play routes."
  - "Playback route search normalizes fromStart to a boolean before the playback page consumes it."
requirements-completed: [PLAY-01, PROG-02]
duration: 4min
completed: 2026-04-22
---

# Phase 4 Plan 02: Frontend playback route intent and controller seam Summary

**Typed playback-entry intents with validated `fromStart` route search and one canonical controller seam for standalone playback navigation.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-04-21T20:54:05Z
- **Completed:** 2026-04-21T20:58:30Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Added a reusable playback intent contract for media-item, file override, and restart-from-beginning navigation.
- Validated `fromStart` on both standalone playback routes and passed the normalized intent into `PlaybackPage`.
- Refactored the app controller playback helper to use the shared intent contract while preserving existing detail-page callers.

## Task Commits

Each task was committed atomically:

1. **Task 1: Define reusable playback intent contracts** - `a693318` (feat)
2. **Task 2: Wire router and controller to the canonical playback seam** - `b45bf7f` (feat)

**Plan metadata:** pending final docs commit

## Files Created/Modified
- `web/src/features/app/types/playback-intent.ts` - Shared route-search and navigation intent types for canonical playback entry.
- `web/src/router.tsx` - Boolean route-search validation for `fromStart` on both playback entry routes.
- `web/src/features/app/hooks/use-app-controller.ts` - Typed playback navigation helper and compatibility wrapper for existing CTA consumers.
- `web/src/features/app/pages/playback-page.tsx` - Accepts validated restart intent so forced restart skips resume seeking.

## Decisions Made
- Used route search instead of extra route variants so restart intent stays explicit without fragmenting the standalone playback surface.
- Added `onOpenPlaybackEntry` alongside the existing `onOpenPlayer` adapter so later surfaces can adopt the shared intent contract incrementally.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Extended PlaybackPage to consume validated restart intent**
- **Found during:** Task 2 (Wire router and controller to the canonical playback seam)
- **Issue:** Router could validate and forward `fromStart`, but `PlaybackPage` had no prop or seek guard to receive that intent.
- **Fix:** Added an optional `fromStart` prop and skipped saved-position seeking when restart intent is explicit.
- **Files modified:** `web/src/features/app/pages/playback-page.tsx`
- **Verification:** `cd web && pnpm typecheck && pnpm build`
- **Committed in:** `b45bf7f` (part of task commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** The auto-fix was required to complete the intended route-to-page contract without adding any extra surface area.

## Issues Encountered
- `gsd-sdk` was not available in this environment, so plan-state updates were applied directly to the planning files instead of using the query helpers.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Home/detail surfaces can now adopt `PlaybackEntryIntent` and explicit `fromStart` behavior without redefining playback route shapes.
- No blocker found for 04-03 UI wiring.

## Self-Check

PASSED

- FOUND: `.planning/phases/04-playback-entry-unified-progress/04-02-SUMMARY.md`
- FOUND: `a693318`
- FOUND: `b45bf7f`

---
*Phase: 04-playback-entry-unified-progress*
*Completed: 2026-04-22*
