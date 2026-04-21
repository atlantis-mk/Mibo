---
phase: 05-playback-decision-intelligence
plan: 02
subsystem: ui
tags: [react, typescript, playback, shaka, api-client]
requires:
  - phase: 05-playback-decision-intelligence
    provides: explicit client-profile playback responses with direct, fallback, and unplayable decisions
provides:
  - typed web playback API support for explicit client profiles and decision metadata
  - playback page handling for direct, fallback, and unplayable backend outcomes
  - truthful fallback link labeling and messaging for standalone playback
affects: [phase-06, playback-page, web-api-client]
tech-stack:
  added: []
  patterns: [decision-aware playback ui, explicit web playback profile, typed playback decision contract]
key-files:
  created: []
  modified: [web/src/lib/mibo-api.ts, web/src/features/app/pages/playback-page.tsx, web/src/features/app/hooks/use-playback-state.ts]
key-decisions:
  - "Make the web client always declare itself as `web` instead of leaving playback capability implicit."
  - "Keep the playback page on the existing route and player boot flow while making fallback and unplayable states truthful."
patterns-established:
  - "Frontend playback requests go through the typed API client with an explicit `clientProfile` option."
  - "Playback UI reads `playbackSource.decision` to decide whether to boot the player or render a state surface."
requirements-completed: [PLAY-02, PLAY-03]
duration: 10 min
completed: 2026-04-22
---

# Phase 5 Plan 02: Web typed playback contract consumption and decision-aware playback page behavior Summary

**Typed web playback requests with explicit `web` capability and playback-page messaging that distinguishes direct, fallback, and unplayable results.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-04-22T06:02:00Z
- **Completed:** 2026-04-22T06:12:00Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Extended the web API client with typed client-profile and playback-decision shapes.
- Updated the standalone playback page to request `clientProfile: "web"` and handle fallback/unplayable outcomes explicitly.
- Kept the rest of the app type-safe by updating the shared playback-state hook to the new API contract.

## Task Commits

Implementation was committed in one verified code commit:

1. **Task 1: Extend the typed web API for explicit client profiles and decision payloads** - `89ad32b` (feat)
2. **Task 2: Make the playback page decision-aware for direct, fallback, and unplayable results** - `89ad32b` (feat)

**Plan metadata:** pending final docs commit

## Files Created/Modified
- `web/src/lib/mibo-api.ts` - Adds `ClientProfile`, `PlaybackDecision`, and the explicit `getPlayback(..., { clientProfile })` contract.
- `web/src/features/app/pages/playback-page.tsx` - Requests `clientProfile: "web"`, renders fallback messaging, and surfaces unplayable decision reasons.
- `web/src/features/app/hooks/use-playback-state.ts` - Adopts the new typed playback request shape for the shared playback state helper.

## Decisions Made
- Preserved the existing playback route and native-vs-Shaka boot logic by continuing to key player selection off the returned `container`.
- Replaced the hardcoded `直链` label only when the backend explicitly selected fallback playback so direct results keep their current affordance.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Updated the shared playback-state hook to the new API signature**
- **Found during:** Task 2 (Make the playback page decision-aware for direct, fallback, and unplayable results)
- **Issue:** `use-playback-state.ts` still depended on the old `getPlayback(mediaItemId, mediaFileId?)` shape, which broke the production build after the typed API contract changed.
- **Fix:** Switched the hook to the new options object and surfaced unplayable responses as user-facing errors instead of opening an empty player.
- **Files modified:** `web/src/features/app/hooks/use-playback-state.ts`
- **Verification:** `cd web && pnpm typecheck && pnpm build`
- **Committed in:** `89ad32b` (part of task commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** The auto-fix kept shared playback entry points aligned with the new typed contract. No scope creep.

## Issues Encountered
- `gsd-sdk` was not executable in this environment, so plan-state updates were applied directly to planning files.
- The frontend work lives in the nested `web/` git repository, so the code commit was created there while planning metadata remains in the root repository.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The web client now consumes the backend playback decision contract without misleading fallback labels.
- No blocker found for Phase 6 planning/execution.

## Self-Check

PASSED

- VERIFIED: `cd web && pnpm typecheck`
- VERIFIED: `cd web && pnpm build`
- FOUND: `89ad32b`

---
*Phase: 05-playback-decision-intelligence*
*Completed: 2026-04-22*
