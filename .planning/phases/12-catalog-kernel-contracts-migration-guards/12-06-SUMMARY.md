---
phase: 12-catalog-kernel-contracts-migration-guards
plan: 06
subsystem: database
tags: [catalog, projections, migration, search, gorm]
requires:
  - phase: 12-catalog-kernel-contracts-migration-guards
    provides: queueable catalog projection refresh contracts and targeted rebuild coverage
provides:
  - Canonicalized search-document rebuilds for legacy `show` rows
  - Defaulted `no_local_media` persistence for blank availability values
  - Targeted regression coverage for mixed legacy and canonical projection rows
affects: [catalog, search, scanner, metadata, migration-cutover]
tech-stack:
  added: []
  patterns: [canonicalize-at-persistence-boundary, focused-projection-regressions]
key-files:
  created: []
  modified:
    - mibo-media-server/internal/catalog/projections.go
    - mibo-media-server/internal/catalog/projections_test.go
key-decisions:
  - "Normalize search-document fields only at projection persistence so existing rollup scope and aggregation behavior stay unchanged."
  - "Cover both library-scope legacy rows and item-scope blank availability rows so mixed migration-era refresh paths stay stable."
patterns-established:
  - "Projection rebuilds must pass legacy catalog values through catalog normalization helpers before writing cutover-facing search documents."
  - "Projection regressions should assert stored `catalog_search_documents` rows, not only in-memory refresh behavior."
requirements-completed: [PROD-01]
duration: 3m
completed: 2026-04-25
---

# Phase 12 Plan 06: Projection canonicalization summary

**Catalog projection rebuilds now persist canonical `series` and `no_local_media` values into `catalog_search_documents` even when source rows still contain legacy or blank migration-era data.**

## Performance

- **Duration:** 3m
- **Started:** 2026-04-25T14:29:29+08:00
- **Completed:** 2026-04-25T14:32:29+08:00
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Added failing regression coverage for a library refresh that starts from a legacy `show` root row and blank availability source data.
- Added item-scope coverage proving blank availability is normalized when rebuilding a targeted search document.
- Updated projection persistence to use the catalog normalization helpers before writing cutover-facing search rows.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add failing regression coverage for legacy type and blank availability rebuilds** - `6a5d497` (test)
2. **Task 2: Normalize search document fields before projection persistence** - `bc87956` (feat)

_Note: This plan followed TDD with a failing regression commit before the implementation fix._

## Files Created/Modified

- `mibo-media-server/internal/catalog/projections_test.go` - regression coverage for legacy `show` type and blank availability refresh behavior
- `mibo-media-server/internal/catalog/projections.go` - canonicalizes persisted search-document type and availability fields during rebuilds
- `.planning/phases/12-catalog-kernel-contracts-migration-guards/12-06-SUMMARY.md` - execution summary for this plan

## Decisions Made

- Normalized only the search-document write boundary because the plan explicitly scoped the fix to persistence, not rollup semantics.
- Kept assertions focused on stored `catalog_search_documents` rows so the Phase 12 verification gap is directly reproduced and closed.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Later scanner, metadata, and API cutover work can rebuild search documents without reintroducing legacy `show` or blank availability values.
- Phase 12 verification gap 2 is now covered by targeted regression tests.

## Known Stubs

None.

## Self-Check: PASSED

- FOUND: `.planning/phases/12-catalog-kernel-contracts-migration-guards/12-06-SUMMARY.md`
- FOUND: `6a5d497`
- FOUND: `bc87956`
