---
phase: 03-semantic-catalog-discovery
plan: 01
subsystem: api
tags: [go, catalog, discovery, home-discovery, react]

# Dependency graph
requires:
  - phase: 02-library-async-sync-foundation
    provides: durable media_files/media_items catalog and async scan foundation
provides:
  - library-aware browse inputs with type, year, and sort filters
  - show-grouped catalog discovery rows anchored to media item detail routes
  - consolidated home discovery payload with per-library latest rails
affects: [web-discovery, tv-detail, sidebar-navigation]

# Tech tracking
tech-stack:
  added: []
  patterns: [typed browse input normalization, per-library home discovery rails, show grouping by external identity]

key-files:
  created: []
  modified:
    - mibo-media-server/internal/library/query.go
    - mibo-media-server/internal/httpapi/router.go
    - mibo-media-server/internal/httpapi/router_test.go
    - web/src/lib/mibo-api.ts

key-decisions:
  - "Normalize browse query params to deterministic defaults before querying catalog data."
  - "Represent TV discovery as grouped show cards keyed by external_id with a stable series_title fallback."
  - "Replace the home page's global latest rail contract with latest_by_library while keeping continue watching and recently played cross-library."

patterns-established:
  - "Catalog browse endpoints accept explicit input structs rather than ad hoc parameter lists."
  - "Home discovery payloads stay media-centric and library-first, without provider-facing browse metadata."

requirements-completed: [CATA-02, CATA-03, CATA-05]

# Metrics
duration: 7 min
completed: 2026-04-21
---

# Phase 03 Plan 01: semantic-catalog-discovery Summary

**Library-aware catalog browse filters, grouped show discovery cards, and per-library home rails over durable media_items/media_files data.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-04-21T16:34:18Z
- **Completed:** 2026-04-21T16:42:08Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments
- Added a typed browse query contract with normalized `type`, `year`, `sort`, `scope`, and `limit` handling.
- Changed show discovery to return grouped show cards instead of raw episode rows, using matched external identity first.
- Added a consolidated home discovery payload with cross-library progress rails and per-library latest rails, plus router coverage for both contracts.

## Task Commits

Each task was committed atomically:

1. **Task 1: catalog query layer and library browse filters** - `b2fcf12` (feat)
2. **Task 2: home discovery endpoint and client contract** - `4dd211b` (root feat), `ec63a65` (web feat)
3. **Task 3: HTTP coverage for catalog and home discovery** - `07c9e0e` (test)

**Plan metadata:** pending final docs commit

## Files Created/Modified
- `mibo-media-server/internal/library/query.go` - browse input normalization, show grouping, and per-library latest queries.
- `mibo-media-server/internal/httpapi/router.go` - library browse query parsing and `/api/v1/home/discovery` response assembly.
- `mibo-media-server/internal/httpapi/router_test.go` - integration coverage for filtered browse responses and home discovery payloads.
- `web/src/lib/mibo-api.ts` - typed `HomeDiscovery` client contract and `getHomeDiscovery()` method.

## Decisions Made
- Kept browse validation server-side by normalizing invalid query params to deterministic defaults instead of returning inconsistent behavior.
- Used grouped show cards keyed by `external_id`, falling back to library + `series_title`, so TV discovery is stable without leaking provider concepts.
- Made home discovery authenticated and library-first so the UI can keep personal progress rails while rendering per-library latest sections.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Split Task 2 commits across the root repo and nested `web/` repo**
- **Found during:** Task 2
- **Issue:** `web/src/lib/mibo-api.ts` is not tracked by the root repository because `web/` is its own nested git repository.
- **Fix:** Committed backend changes in the root repo and committed the API client change in the `web/` repo with the same task message.
- **Files modified:** `mibo-media-server/internal/library/query.go`, `mibo-media-server/internal/httpapi/router.go`, `web/src/lib/mibo-api.ts`
- **Verification:** Root/backend tests passed; `pnpm typecheck` passed in `web/`.
- **Committed in:** `4dd211b`, `ec63a65`

---

**2. [Rule 3 - Blocking] Repaired planning metadata manually after `state.advance-plan` could not parse `STATE.md` placeholders**
- **Found during:** Plan metadata update
- **Issue:** The existing `STATE.md` still contained placeholder tokens like `--phase` / `--name`, so `gsd-sdk query state.advance-plan` failed to parse the current position.
- **Fix:** Used the other state handlers that still succeeded, then manually corrected `STATE.md` and `ROADMAP.md` so plan progress reflects Phase 03 Plan 01 completion.
- **Files modified:** `.planning/STATE.md`, `.planning/ROADMAP.md`
- **Verification:** Re-read both files and confirmed Phase 03 now shows plan 1 complete and plan 2 next.
- **Committed in:** `6a90b3d`

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** No product scope change; deviations were limited to commit routing and metadata repair needed to finish execution cleanly.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 01 now provides the browse and home contracts needed by the frontend discovery wave.
- TV season/episode enrichment and cache endpoints remain for later Phase 3 plans.

## Self-Check: PASSED

- Verified summary and key modified backend files exist on disk.
- Verified root task commits `b2fcf12`, `4dd211b`, and `07c9e0e` exist.
- Verified nested `web/` task commit `ec63a65` exists.

---
*Phase: 03-semantic-catalog-discovery*
*Completed: 2026-04-21*
