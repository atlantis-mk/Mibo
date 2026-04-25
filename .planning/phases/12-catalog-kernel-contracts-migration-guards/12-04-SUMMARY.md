---
phase: 12-catalog-kernel-contracts-migration-guards
plan: 04
subsystem: database
tags: [catalog, database, gorm, sqlite, migration, indexes]
requires:
  - phase: 12-catalog-kernel-contracts-migration-guards
    provides: catalog DTO, migration-state, and projection-job groundwork this startup hardening builds on
provides:
  - Additive composite indexes for catalog hierarchy and projection lookups
  - Explicit startup regression coverage for fresh, legacy-only, and repeated SQLite opens
  - Idempotent database.Open index enforcement that preserves legacy tables during migration
affects: [catalog, backfill, scanner, migration-cutover, database-startup]
tech-stack:
  added: []
  patterns: [additive gorm index enforcement, legacy-compatible startup migration tests, repeated-open idempotency checks]
key-files:
  created:
    - mibo-media-server/internal/database/database_open_test.go
  modified:
    - mibo-media-server/internal/database/catalog_models.go
    - mibo-media-server/internal/database/database.go
    - mibo-media-server/internal/database/catalog_models_test.go
key-decisions:
  - "Define the new lookup indexes in GORM tags and also enforce them explicitly in database.Open so startup remains additive and idempotent across SQLite/Postgres behavior differences."
  - "Seed a legacy-only SQLite schema with representative rows in tests instead of inventing a parallel migration path, so compatibility proof stays anchored to database.Open."
patterns-established:
  - "Catalog migration guards use shared composite index names plus Migrator().CreateIndex as additive backstop."
  - "Startup regression tests prove fresh, legacy-only, and repeated-open behavior against the real database.Open entrypoint."
requirements-completed: [PROD-01]
duration: 5m 57s
completed: 2026-04-25
---

# Phase 12 Plan 04: Catalog migration guard summary

**Composite catalog hierarchy/search indexes now ship with additive startup enforcement and SQLite regression tests that prove fresh, legacy-only, and repeated database opens stay compatible.**

## Performance

- **Duration:** 5m 57s
- **Started:** 2026-04-25T13:41:30+08:00
- **Completed:** 2026-04-25T13:47:27+08:00
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Added the four phase-12 composite lookup indexes for catalog hierarchy traversal and catalog search projection rebuilds.
- Hardened `database.Open(...)` with explicit additive index creation so repeated startup remains idempotent.
- Added regression tests for fresh DB migration, legacy-only DB migration, and repeated-open compatibility while preserving representative legacy rows.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add minimum composite indexes for catalog hierarchy and projection queries** - `e8ca810` (test), `ec84f46` (feat)
2. **Task 2: Prove empty-database and legacy-database startup compatibility after schema hardening** - `7359df2` (test), `e7ef5cf` (feat)

_Note: TDD tasks used RED → GREEN commit pairs._

## Files Created/Modified

- `mibo-media-server/internal/database/catalog_models.go` - shared composite index tags for catalog item and catalog search lookup patterns
- `mibo-media-server/internal/database/database.go` - additive `CreateIndex` backstop for required catalog indexes during startup migration
- `mibo-media-server/internal/database/catalog_models_test.go` - fresh-start migration assertions now cover legacy and catalog tables together
- `mibo-media-server/internal/database/database_open_test.go` - fresh, legacy-only, and repeated-open SQLite migration regression coverage

## Decisions Made

- Used shared GORM index names with explicit priority ordering so composite index column order matches the phase-12 query patterns.
- Kept the migration boundary inside `database.Open(...)` and used `Migrator().CreateIndex(...)` only as an additive safety backstop instead of introducing destructive SQL.
- Seeded a representative legacy schema with rows in tests so repeated-open coverage proves both compatibility and data preservation.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `go test ./internal/database -run 'Test(CatalogKernelTablesAreMigrated|DatabaseOpen.*Catalog)' -count=1` passed.
- `go test ./internal/httpapi -run TestReadyz -count=1` was blocked by an unrelated compile error in `mibo-media-server/internal/library/scan_classify.go` vs `mibo-media-server/internal/library/query_series_grouping.go` (`normalizeSeriesGroupingTitle` redeclared). Per scope-boundary rules this was recorded in `deferred-items.md` and left untouched.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Backfill and cutover work now has the minimum catalog traversal/projection indexes the roadmap called for.
- Startup migration safety is covered for fresh and legacy SQLite databases, with repeated-open assertions guarding future schema tightening.
- Full `TestReadyz` verification should be rerun after the unrelated library compile conflict is resolved outside this plan.

## Known Stubs

None.

## Self-Check: PASSED

- FOUND: `.planning/phases/12-catalog-kernel-contracts-migration-guards/12-04-SUMMARY.md`
- FOUND: `e8ca810`
- FOUND: `ec84f46`
- FOUND: `7359df2`
- FOUND: `e7ef5cf`
