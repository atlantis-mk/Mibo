---
phase: 09-trailer-discovery-playback
plan: "09"
subsystem: ui
tags: [tmdb, trailer, react-query, detail-modal, youtube, go, sqlite]
requires:
  - phase: 07-metadata-governance-matching
    provides: metadata ownership and match/refetch job flow
  - phase: 08-native-search-discovery-filters
    provides: typed web API client and detail query patterns
provides:
  - server-selected TMDB trailer metadata persisted on media items
  - media detail trailer contract with hidden-entry semantics when unavailable
  - detail-area trailer entry with in-page modal playback
affects: [metadata, media-detail, playback, tmdb, worker]
tech-stack:
  added: []
  patterns:
    - server-side single trailer selection exposed through existing detail contract
    - detail-page modal playback wired from typed React Query detail data
key-files:
  created:
    - web/src/features/media/components/standalone-media-detail-trailer-dialog.tsx
  modified:
    - mibo-media-server/internal/database/models.go
    - mibo-media-server/internal/metadata/service.go
    - mibo-media-server/internal/metadata/service_tmdb.go
    - mibo-media-server/internal/metadata/service_helpers.go
    - mibo-media-server/internal/metadata/service_match.go
    - mibo-media-server/internal/library/query.go
    - mibo-media-server/internal/library/query_detail.go
    - mibo-media-server/internal/metadata/service_test.go
    - mibo-media-server/internal/library/query_test.go
    - mibo-media-server/internal/httpapi/router_test.go
    - web/src/lib/mibo-api.ts
    - web/src/features/media/components/standalone-media-detail.tsx
    - web/src/features/media/components/standalone-media-detail-hero.tsx
    - web/src/features/media/components/standalone-media-detail-specs.tsx
key-decisions:
  - "Keep trailer discovery metadata-driven: the frontend consumes one persisted trailer result from GET /api/v1/media-items/{id}."
  - "Use SpecsSection as the formal trailer entry point and remove the hero placeholder from the primary interaction path."
  - "Play trailers inside a detail-page dialog so closing playback always returns users to the same detail context."
patterns-established:
  - "Trailer contract pattern: backend normalizes one playable trailer and omits the field entirely when unavailable."
  - "Detail playback pattern: feature-local modal state layered on existing detail query data without extra trailer fetches."
requirements-completed: [TRLR-01, TRLR-02, TRLR-03, TRLR-04]
duration: 4 min
completed: 2026-04-24
---

# Phase 9 Plan 09: Trailer Discovery & Playback Summary

**TMDB trailer metadata now syncs into Mibo-owned detail data, and users can open a single selected trailer from the detail info area in an in-page modal player.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-04-24T02:27:22Z
- **Completed:** 2026-04-24T02:31:38Z
- **Tasks:** 6
- **Files modified:** 15

## Accomplishments

- Completed the backend trailer contract already in progress by validating TMDB `videos` fetch, single-result selection, persistence, and detail projection coverage.
- Added typed frontend trailer support, moved the formal entry into the detail/specs area, and removed the hero placeholder from the formal path.
- Implemented in-page trailer playback with dialog state tied to the existing detail query, plus no-trailer regression coverage for hidden-entry semantics.

## Task Commits

Each task was intended to be committed atomically, but commit creation was blocked by environment limitations:

1. **Task 9-01-01: Trailer selection and normalization** - `unavailable` (git CLI unavailable)
2. **Task 9-01-02: Trailer persistence and detail projection** - `unavailable` (git CLI unavailable)
3. **Task 9-02-01: Typed detail trailer client and specs entry** - `unavailable` (git CLI unavailable)
4. **Task 9-02-02: In-page trailer playback modal and hero demotion** - `unavailable` (git CLI unavailable)
5. **Task 9-03-01: Trailer regression coverage** - `unavailable` (git CLI unavailable)
6. **Task 9-03-02: Integrated validation and UX hardening** - `unavailable` (git CLI unavailable)

**Plan metadata:** `unavailable` (git CLI unavailable)

## Files Created/Modified

- `mibo-media-server/internal/database/models.go` - persisted the selected trailer JSON on media items.
- `mibo-media-server/internal/metadata/service.go` - extended TMDB detail decoding to include `videos` payloads.
- `mibo-media-server/internal/metadata/service_tmdb.go` - requested `videos` alongside existing TMDB appended responses.
- `mibo-media-server/internal/metadata/service_helpers.go` - implemented playable trailer selection, URL normalization, and thumbnail generation.
- `mibo-media-server/internal/metadata/service_match.go` - stored normalized trailer detail during metadata match/refetch flows.
- `mibo-media-server/internal/library/query.go` - added typed trailer detail to the media item detail contract.
- `mibo-media-server/internal/library/query_detail.go` - projected persisted trailer JSON into detail responses while omitting invalid/unavailable data.
- `mibo-media-server/internal/metadata/service_test.go` - covered trailer sync, selection priority, and empty fallback behavior.
- `mibo-media-server/internal/library/query_test.go` - covered trailer parsing on detail reads.
- `mibo-media-server/internal/httpapi/router_test.go` - covered detail serialization with and without trailer payloads.
- `web/src/lib/mibo-api.ts` - added typed trailer shape to the web detail contract.
- `web/src/features/media/components/standalone-media-detail.tsx` - added feature-local trailer dialog state and rendering.
- `web/src/features/media/components/standalone-media-detail-hero.tsx` - removed the hero trailer placeholder from the formal interaction path.
- `web/src/features/media/components/standalone-media-detail-specs.tsx` - added the formal trailer entry card in the detail info area.
- `web/src/features/media/components/standalone-media-detail-trailer-dialog.tsx` - implemented in-page modal trailer playback.

## Decisions Made

- Followed the locked Phase 9 direction exactly: detail-area entry, in-page modal playback, metadata-driven sync, and one selected trailer result.
- Kept the frontend on the existing `mibo-api.ts` + React Query path instead of adding any detail-page trailer fetch.
- Added HTTP-level no-trailer regression coverage so omitted trailer fields remain part of the contract, not just the persistence layer.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The workspace does not provide the `git` CLI, so per-task commits and the final metadata commit could not be created.
- Backend trailer contract work already existed locally in-progress; it aligned with Plan 01 and was preserved, verified, and completed instead of being replaced.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Trailer discovery and playback now have a stable app-owned backend/frontend contract ready for follow-on polish or future refresh scheduling work.
- No functional blockers remain for Phase 9 goals; only git/commit metadata is missing due the environment limitation.

## Self-Check: PASSED

- Summary file exists at `.planning/phases/09-trailer-discovery-playback/09-SUMMARY.md`.
- Commit hashes remain unavailable because the environment does not provide the `git` CLI.

---
*Phase: 09-trailer-discovery-playback*
*Completed: 2026-04-24*
