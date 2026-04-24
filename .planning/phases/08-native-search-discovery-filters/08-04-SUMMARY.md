---
phase: 08-native-search-discovery-filters
plan: "04"
subsystem: testing
tags: [go, httpapi, worker, metadata, progress, discovery]
requires:
  - phase: 08-native-search-discovery-filters
    provides: projection-backed discovery reads and lifecycle reindex hooks
provides:
  - focused regression proof for region, rating, watched-state, and highlight behavior
  - worker-level proof for queued search reindex jobs
affects: [phase-08, verification, testing, discovery]
tech-stack:
  added: []
  patterns: [mutation-driven discovery regression tests]
key-files:
  created: []
  modified:
    - mibo-media-server/internal/httpapi/router_test.go
    - mibo-media-server/internal/metadata/service_test.go
    - mibo-media-server/internal/progress/service_test.go
    - mibo-media-server/internal/worker/worker_test.go
key-decisions:
  - Phase 8 closure is proven with mutation-driven tests instead of relying on generic build/test success.
patterns-established:
  - Discovery regression coverage should assert shared browse/search semantics from the public routes down to the worker queue.
requirements-completed: [SRCH-01, SRCH-02, SRCH-03, SRCH-04, SRCH-05, SRCH-06, SRCH-07, SRCH-08, FLTR-01, FLTR-02, FLTR-03, FLTR-04, FLTR-05, FLTR-06]
duration: n/a
completed: 2026-04-24
---

# Phase 8 Plan 04: Discovery Regression Proof Summary

## Outcome

Added focused regression coverage that proves projection-backed discovery stays correct for region, rating, watched-state, highlights, and worker-driven refresh paths.

## Accomplishments

- Added metadata tests proving TMDB country/rating persistence also refreshes the discovery projection.
- Added progress tests proving watched-state transitions reindex the affected discovery document.
- Added router tests asserting discovery and browse stay aligned for `region`, `min_rating`, and `watched_state`, and that search results still expose highlights plus movie/show distinction.
- Added worker tests covering the new search reindex job kinds.

## Validation

- `go test ./internal/metadata ./internal/progress ./internal/httpapi -run 'Test.*(Discovery|Region|Rating|Watched|Highlight)'`
- `go test ./internal/worker ./internal/httpapi`
- `go test ./...`

## Deviations From Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Next Phase Readiness

- Phase 8 has explicit automated proof for the gaps reported in the previous verification pass.
