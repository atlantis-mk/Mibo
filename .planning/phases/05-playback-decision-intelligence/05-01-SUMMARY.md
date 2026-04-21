---
phase: 05-playback-decision-intelligence
plan: 01
subsystem: api
tags: [go, playback, hls, ffprobe, routing]
requires:
  - phase: 04-playback-entry-unified-progress
    provides: authenticated playback entry and canonical progress semantics
provides:
  - explicit `client_profile` playback request validation for `web`, `mobile`, and `tv`
  - profile-aware playback decisions with direct, fallback, and unplayable outcomes
  - compatibility-first file ranking driven by persisted probe data
affects: [05-02, playback-page, hls, media-streaming]
tech-stack:
  added: []
  patterns: [profile-aware playback decisioning, explicit fallback reasons, compatibility-first file selection]
key-files:
  created: [mibo-media-server/internal/playback/profile.go, mibo-media-server/internal/playback/service_test.go]
  modified: [mibo-media-server/internal/playback/service.go, mibo-media-server/internal/httpapi/router.go, mibo-media-server/internal/httpapi/router_test.go]
key-decisions:
  - "Require explicit `client_profile` on playback requests instead of inferring capability from `User-Agent`."
  - "Keep HLS as a fallback mechanism owned by playback policy, not an unconditional router rewrite."
patterns-established:
  - "Playback responses always carry one decision object describing why direct, fallback, or unplayable was selected."
  - "Default file selection filters for profile suitability before applying quality tie-breakers."
requirements-completed: [PLAY-02, PLAY-03]
duration: 28 min
completed: 2026-04-22
---

# Phase 5 Plan 01: Backend explicit client-profile playback contract, probe-aware decision engine, and per-request HLS fallback Summary

**Capability-aware playback selection with explicit profile validation, probe-driven direct-play checks, and HLS fallback/unplayable decisions.**

## Performance

- **Duration:** 28 min
- **Started:** 2026-04-22T05:34:00Z
- **Completed:** 2026-04-22T06:02:00Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Added a typed backend playback contract for client profiles and decision reasons.
- Refactored playback selection to choose profile-compatible direct play first, then explicit HLS fallback, then explicit unplayable results.
- Added regression coverage for missing/invalid profiles, decision kinds, probe-missing optimism, and compatibility-first ranking.

## Task Commits

Implementation was committed in one verified code commit:

1. **Task 1: Lock the explicit client-profile playback contract with failing coverage** - `af1f150` (feat)
2. **Task 2: Implement probe-aware direct, fallback, and unplayable decisioning** - `af1f150` (feat)

**Plan metadata:** pending final docs commit

## Files Created/Modified
- `mibo-media-server/internal/playback/profile.go` - Defines the typed client-profile and playback-decision contract.
- `mibo-media-server/internal/playback/service.go` - Implements profile-aware file ranking and direct/fallback/unplayable decisions.
- `mibo-media-server/internal/playback/service_test.go` - Covers direct, fallback, unplayable, probe-missing, and ranking behavior.
- `mibo-media-server/internal/httpapi/router.go` - Validates `client_profile` and passes fallback capability into the playback service.
- `mibo-media-server/internal/httpapi/router_test.go` - Covers missing/invalid profiles and decision payload behavior.

## Decisions Made
- Used a required query parameter for `client_profile` so the existing playback route shape stayed intact while capability choice became explicit.
- Kept decision ownership in `playback.Service` so the HTTP layer only validates input and reports the selected outcome.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated legacy router tests that still assumed unconditional mkv direct play**
- **Found during:** Task 2 (Implement probe-aware direct, fallback, and unplayable decisioning)
- **Issue:** Existing HTTP playback tests were still written against the old black-box playback behavior and expected direct streaming for `mkv` fixtures that are no longer direct-play candidates for the new `web` profile.
- **Fix:** Aligned the direct-stream fixtures with the shipped profile contract and updated HLS request coverage to include `client_profile=web`.
- **Files modified:** `mibo-media-server/internal/httpapi/router_test.go`
- **Verification:** `cd mibo-media-server && go test ./internal/playback ./internal/httpapi`
- **Committed in:** `af1f150` (part of task commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** The auto-fix kept the existing backend suite aligned with the Phase 5 contract. No scope creep.

## Issues Encountered
- `gsd-sdk` was not executable in this environment, so plan-state updates were applied directly to planning files.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The backend now exposes one stable decision-aware playback response for frontend consumers.
- No blocker found for 05-02 web contract consumption.

## Self-Check

PASSED

- VERIFIED: `cd mibo-media-server && go test ./internal/playback ./internal/httpapi -run 'TestPlayback'`
- VERIFIED: `cd mibo-media-server && go test ./internal/playback ./internal/httpapi`
- FOUND: `af1f150`

---
*Phase: 05-playback-decision-intelligence*
*Completed: 2026-04-22*
