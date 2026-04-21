---
phase: 04-playback-entry-unified-progress
plan: 03
subsystem: ui
tags: [react, tanstack-router, playback, progress, resume, restart]
requires:
  - phase: 04-playback-entry-unified-progress
    provides: authenticated playback entry and canonical media-item progress semantics
  - phase: 04-playback-entry-unified-progress
    provides: typed playback route search and canonical playback navigation intents
provides:
  - continue-watching cards that open the standalone playback page directly
  - detail actions that default to resume, expose explicit restart, and confirm mark-watched
  - playback-page seek defaults that honor watched rows and explicit restart intent
affects: [04-04, playback-page, continue-watching, media-detail]
tech-stack:
  added: []
  patterns: [home-to-playback routing, resumable-detail actions, watched-aware playback seek defaults]
key-files:
  created: []
  modified: [web/src/features/app/components/browse-app-shell.tsx, web/src/features/app/components/browse-panel.tsx, web/src/features/app/components/media-detail-panel.tsx, web/src/features/app/components/standalone-media-detail.tsx, web/src/features/app/pages/playback-page.tsx]
key-decisions:
  - "Route the home continue-watching rail directly to /play so recovery starts playback without an intermediate detail hop."
  - "Only show 从头播放 when unfinished canonical progress exists, while watched titles fall back to 立即播放 and start from zero."
patterns-established:
  - "Detail surfaces use one canonical action set: primary play/resume, optional restart, rematch, and confirmed mark-watched."
  - "PlaybackPage derives its initial seek target from validated route intent plus canonical progress state, not from stale saved position alone."
requirements-completed: [PLAY-01, PROG-01, PROG-02]
duration: 8min
completed: 2026-04-22
---

# Phase 4 Plan 03: Home, detail, and playback resume UX Summary

**Canonical standalone playback entry with home-first resume routing, detail restart controls, and watched-aware player seek behavior.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-21T20:59:31Z
- **Completed:** 2026-04-21T21:07:31Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Routed the home `继续观看` rail straight into the standalone playback page instead of reopening detail first.
- Reworked detail actions around automatic sync semantics with `继续播放` / `立即播放`, conditional `从头播放`, and confirmed `标记看完`.
- Hardened playback-page start position rules so watched rows and explicit restart intents begin from zero without stale resume sync behavior.

## Task Commits

Each task was committed atomically:

1. **Task 1: Rewire home and detail surfaces to canonical play/restart actions** - `web@bb0e4f6` (feat)
2. **Task 2: Make the playback page honor restart-vs-resume semantics** - `web@fd3212a` (feat)

**Plan metadata:** pending final docs commit

## Files Created/Modified
- `web/src/features/app/components/browse-app-shell.tsx` - Passed the typed playback-entry callback into browse and detail surfaces.
- `web/src/features/app/components/browse-panel.tsx` - Sent home continue-watching selections directly to standalone playback.
- `web/src/features/app/components/media-detail-panel.tsx` - Replaced manual save emphasis with resume/restart actions and confirmed mark-watched UI.
- `web/src/features/app/components/standalone-media-detail.tsx` - Matched the standalone detail action row to the canonical resume/restart/watch-complete contract.
- `web/src/features/app/pages/playback-page.tsx` - Derived initial seek and sync baseline from restart intent plus canonical watched state.

## Decisions Made
- Kept `最近播放` and generic catalog cards on their existing detail-first behavior while making `继续观看` the dedicated direct-to-play recovery surface.
- Reused the route-level `fromStart` intent from Plan 02 instead of introducing a second playback route or a manual-save comeback flow.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Reset restart/watched sync baseline to zero**
- **Found during:** Task 2 (Make the playback page honor restart-vs-resume semantics)
- **Issue:** The page still seeded `lastSyncedPositionRef` from the old saved progress row, which would suppress the 15-second auto-sync loop after a watched/default-from-zero or `fromStart=true` entry.
- **Fix:** Introduced a shared initial-playback-position helper and reused it for both seek selection and sync baseline initialization.
- **Files modified:** `web/src/features/app/pages/playback-page.tsx`
- **Verification:** `cd web && pnpm typecheck && pnpm build`
- **Committed in:** `web@fd3212a` (part of task commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** The auto-fix was necessary to keep restart-from-zero entries aligned with the existing automatic progress-sync contract.

## Issues Encountered
- `gsd-sdk` was not executable in this environment, so planning state updates were applied directly to the planning files.
- The implementation work lived in the nested `web/` git repository, so task commits were created there while planning metadata remains in the root repository.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 4 now exposes the intended product playback entry behavior for manual verification in 04-04.
- No blocker found for end-to-end playback/progress verification.

## Self-Check

PASSED

- FOUND: `.planning/phases/04-playback-entry-unified-progress/04-03-SUMMARY.md`
- FOUND: `web@bb0e4f6`
- FOUND: `web@fd3212a`

---
*Phase: 04-playback-entry-unified-progress*
*Completed: 2026-04-22*
