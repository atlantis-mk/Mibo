---
phase: 10-scheduled-operations-control
plan: 06
subsystem: ui
tags: [react, tanstack-query, route, schedule, workspace]
requires:
  - phase: 10-04
    provides: schedule HTTP APIs
  - phase: 10-05
    provides: stable schedule status and history semantics
provides:
  - typed schedule API/query client helpers
  - dedicated schedules workspace route
  - schedule-first admin list and detail shell
affects: [settings, operations]
tech-stack:
  added: []
  patterns: [typed schedule route workspace, query-driven schedule UI]
key-files:
  created: [web/src/routes/_app.schedules.index.tsx, web/src/features/schedules/index.tsx, web/src/features/schedules/workspace.tsx, web/src/features/schedules/components/schedule-form-dialog.tsx, web/src/features/schedules/components/schedule-list.tsx, web/src/features/schedules/components/schedule-run-history-drawer.tsx]
  modified: [web/src/lib/mibo-api.ts, web/src/lib/mibo-query.ts, web/src/routeTree.gen.ts]
key-decisions:
  - "The dedicated schedules route became the primary management surface rather than a settings-tab expansion."
patterns-established:
  - "Schedule UI loads via typed mibo-api contracts and TanStack Query keys only."
requirements-completed: [SJOB-01, SJOB-02, SJOB-03, SJOB-04, SJOB-05, SJOB-06, SJOB-07, SJOB-08]
duration: unknown
completed: 2026-04-24
---

# Phase 10 Plan 06: Schedules Workspace Summary

**Dedicated schedules route with typed API/query contracts, list management surface, and detail-layer history shell**

## Performance
- **Duration:** unknown
- **Started:** 2026-04-24T00:00:00Z
- **Completed:** 2026-04-24T00:00:00Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- Extended the web API client with typed schedule list/detail/history and mutation methods.
- Added TanStack Query keys/helpers for schedule workspace data.
- Shipped a dedicated `/schedules` workspace with list, edit/create dialog, run-now actions, and history drawer shell.

## Task Commits
1. **Task 1-2: typed client contract and schedules workspace** - `2505823` (feat)

## Files Created/Modified
- `web/src/lib/mibo-api.ts` - typed schedule request/response models and client methods
- `web/src/lib/mibo-query.ts` - schedule query keys and query options
- `web/src/routes/_app.schedules.index.tsx` - dedicated schedules workspace route
- `web/src/features/schedules/*` - workspace, list, form, and history UI
- `web/src/routeTree.gen.ts` - generated route registration updated for the new workspace

## Decisions Made
- Included the form dialog and history drawer during workspace delivery because they were required for a usable schedule-first management surface.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated generated route typing for the new schedules route**
- **Found during:** Task 2 verification
- **Issue:** typecheck failed because the generated TanStack route tree had not yet learned the new `_app.schedules.index` file
- **Fix:** refreshed `web/src/routeTree.gen.ts` to include the schedules workspace route
- **Files modified:** `web/src/routeTree.gen.ts`
- **Verification:** `pnpm typecheck && pnpm build`
- **Committed in:** `2505823`

**Total deviations:** 1 auto-fixed (1 Rule 3)
**Impact on plan:** Required for build correctness; no scope creep.

## Issues Encountered
- Route typing generation had to be refreshed before the new schedule route could typecheck.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The dedicated workspace is in place and ready for the settings summary bridge.

## Self-Check: PASSED
