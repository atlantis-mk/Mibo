---
phase: 03-semantic-catalog-discovery
plan: 02
subsystem: api
tags: [go, gorm, tmdb, react, typescript]

# Dependency graph
requires:
  - phase: 02-library-async-sync-foundation
    provides: durable media item records, worker-backed metadata flow, and HTTP API foundations
provides:
  - persistent TMDB season and episode cache tables keyed by language and series coordinates
  - TV season and episode endpoints for season-first detail navigation
  - series-aware media detail fields for reusing /media/$mediaItemId as the only detail route
affects: [03-03, tv-detail-ui, catalog-discovery]

# Tech tracking
tech-stack:
  added: []
  patterns: [read-through TMDB caching, sanitized TV DTOs, series-aware detail payloads]

key-files:
  created: []
  modified:
    - mibo-media-server/internal/database/models.go
    - mibo-media-server/internal/database/database.go
    - mibo-media-server/internal/metadata/service.go
    - mibo-media-server/internal/httpapi/router.go
    - mibo-media-server/internal/library/query.go
    - mibo-media-server/internal/metadata/service_test.go
    - mibo-media-server/internal/httpapi/router_test.go
    - web/src/lib/mibo-api.ts

key-decisions:
  - "Cache TV season summaries from TMDB show detail and episode rows from season detail responses."
  - "Expose series_tmdb_id and default_season_number on /media item detail instead of adding a new TV detail route."

patterns-established:
  - "TMDB TV cache rows are keyed by series id, season/episode coordinates, and language."
  - "TV API responses expose only Mibo-owned DTO fields while raw TMDB payloads stay server-side in cache tables."

requirements-completed: [CATA-03, CATA-04]

# Metrics
duration: 11 min
completed: 2026-04-21
---

# Phase 03 Plan 02: semantic-catalog-discovery Summary

**Persistent TMDB TV season and episode caching with season endpoints and series-aware `/media/$mediaItemId` detail fields.**

## Performance

- **Duration:** 11 min
- **Started:** 2026-04-21T16:47:43Z
- **Completed:** 2026-04-21T16:59:19Z
- **Tasks:** 3
- **Files modified:** 8

## Accomplishments
- Added dedicated database-backed TMDB season and episode caches with language-aware read-through refresh logic.
- Exposed `GET /api/v1/tv/{tmdb_id}/seasons` and `GET /api/v1/tv/{tmdb_id}/seasons/{n}/episodes` with sanitized season/episode DTOs.
- Extended media detail responses so the existing `/media/$mediaItemId` route can anchor season-first TV navigation.

## Task Commits

Each task was committed atomically:

1. **Task 1: persistent TMDB TV cache models and service helpers** - `bc75d53` (feat)
2. **Task 2: TV season and episode API contracts** - `2ba611f` (backend feat), `web@f690b18` (frontend feat)
3. **Task 3: series-aware detail payload and coverage** - `60011fa` (backend feat), `web@a62d7d0` (frontend feat)

## Files Created/Modified
- `mibo-media-server/internal/database/models.go` - Adds season and episode TMDB cache tables keyed by series coordinates and language.
- `mibo-media-server/internal/database/database.go` - Migrates the new TV cache tables automatically.
- `mibo-media-server/internal/metadata/service.go` - Implements read-through cache lookup, TMDB fetch, upsert, and sanitized TV DTOs.
- `mibo-media-server/internal/httpapi/router.go` - Registers TV season and episode endpoints.
- `mibo-media-server/internal/library/query.go` - Adds series-level detail fields to item detail responses.
- `mibo-media-server/internal/metadata/service_test.go` - Verifies season cache population and episode cache reuse.
- `mibo-media-server/internal/httpapi/router_test.go` - Verifies TV endpoint responses and sanitized payload shape.
- `web/src/lib/mibo-api.ts` - Adds typed TV endpoint helpers and series-aware detail fields for the SPA.

## Decisions Made
- Cached season lists from the TMDB TV detail response and populated episode rows from the TMDB season detail response to avoid repeated client-driven metadata fetches.
- Kept `/media/$mediaItemId` as the only detail route and surfaced `series_tmdb_id`, `series_title_display`, and `default_season_number` so the frontend can hydrate season-first navigation from the existing item detail flow.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Registered new TV cache tables in AutoMigrate**
- **Found during:** Task 1
- **Issue:** Adding cache models without migrating them would leave read-through caching unusable at runtime.
- **Fix:** Updated `mibo-media-server/internal/database/database.go` to AutoMigrate the new season and episode cache tables.
- **Files modified:** `mibo-media-server/internal/database/database.go`
- **Verification:** `go test ./internal/metadata`
- **Committed in:** `bc75d53`

**2. [Rule 2 - Missing Critical] Extended the frontend media detail type for new TV fields**
- **Found during:** Task 3
- **Issue:** The backend detail payload added series-specific fields, but the typed web client would have required unsafe casts to consume them later.
- **Fix:** Added `series_tmdb_id`, `series_title_display`, and `default_season_number` to `web/src/lib/mibo-api.ts`.
- **Files modified:** `web/src/lib/mibo-api.ts`
- **Verification:** `cd web && pnpm typecheck`
- **Committed in:** `web@a62d7d0`

---

**Total deviations:** 2 auto-fixed (2 missing critical)
**Impact on plan:** Both fixes were required to make the planned backend/frontend contract usable in the real workspace. No scope creep.

## Issues Encountered
- The workspace contains a nested `web/.git` repository even though init context reported no `sub_repos`, so frontend task changes were committed in the `web/` repo separately from the root/backend commits.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Backend TV metadata contracts, caching behavior, and detail payload fields are ready for the Phase 3 frontend season selector and episode grid work.
- Frontend API types are aligned with the new backend contracts, so the remaining UI plan can focus on wiring and presentation.

## Self-Check: PASSED

- Found `.planning/phases/03-semantic-catalog-discovery/03-02-SUMMARY.md`
- Verified root commits `bc75d53`, `2ba611f`, and `60011fa`
- Verified web repo commits `f690b18` and `a62d7d0`
