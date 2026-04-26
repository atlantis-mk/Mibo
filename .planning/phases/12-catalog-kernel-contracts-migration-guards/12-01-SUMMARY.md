---
phase: 12-catalog-kernel-contracts-migration-guards
plan: 01
subsystem: api
tags: [go, catalog, dto, contracts, json, gorm]
requires: []
provides:
  - explicit catalog DTO structs for list, detail, season, episode, asset, and governance responses
  - builder helpers that map catalog and inventory rows into stable JSON contracts
  - contract tests that lock canonical series typing and block raw database field leakage
affects: [catalog-api-cutover, catalog-frontend-cutover, metadata-governance, playback, search]
tech-stack:
  added: []
  patterns: [plain json-tagged DTO structs, builder-based database row mapping, contract-first JSON regression tests]
key-files:
  created:
    - mibo-media-server/internal/catalog/contracts.go
    - mibo-media-server/internal/catalog/contracts_test.go
  modified: []
key-decisions:
  - "Keep catalog contracts in internal/catalog as plain json-tagged DTOs instead of exposing database rows."
  - "Normalize legacy series roots to the canonical series type at the contract boundary and keep nested sections stable for downstream readers."
patterns-established:
  - "Catalog reads should build response DTOs through internal/catalog builders rather than serializing database.* structs directly."
  - "Catalog contract regressions should be caught with JSON-shape tests that assert required keys and forbid DB-only fields."
requirements-completed: [KERN-01]
duration: 8min
completed: 2026-04-25
---

# Phase 12 Plan 01: Catalog DTO Contracts Summary

**Catalog DTO contracts with builder-based database mapping, canonical series normalization, and JSON regression tests for list/detail/season/episode/asset/governance payloads**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-25T04:53:56Z
- **Completed:** 2026-04-25T05:01:35Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Added explicit exported catalog DTOs and nested value objects for stable list/detail/governance contracts.
- Added builder helpers that summarize metadata source payloads and field-state values without leaking raw database row fields.
- Locked JSON shape with regression tests for list, detail, season, episode, asset, and governance workspace payloads.

## Task Commits

Each task was committed atomically:

1. **Task 1: Define explicit catalog DTO structs and nested contract value objects**
   - `4482c75` `test(12-01): add failing catalog DTO contract tests`
   - `2954d25` `feat(12-01): add explicit catalog DTO contracts`
2. **Task 2: Lock DTO JSON shape and mapper behavior with contract tests**
   - `79f5416` `test(12-01): expand catalog JSON contract coverage`
   - `d4d90d3` `feat(12-01): lock catalog contract JSON shape`

_Note: This plan used TDD red/green commits per task. No final docs commit was created because the orchestrator will handle shared planning state._

## Files Created/Modified
- `mibo-media-server/internal/catalog/contracts.go` - Defines exported DTO structs, nested contract objects, and row-to-contract builder helpers.
- `mibo-media-server/internal/catalog/contracts_test.go` - Verifies exported DTO coverage, JSON shape, canonical `series` typing, and absence of raw DB-only fields in marshaled payloads.

## Decisions Made
- Keep DTO definitions and builders in `internal/catalog` so later read phases can import a stable contract layer without depending on `internal/database` JSON shapes.
- Emit normalized `series` contract types and stable nested sections (`assets`, `source_evidence`, `field_states`, hierarchy arrays) at the contract boundary to prevent legacy drift during cutover.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Later catalog API/frontend migration work can import explicit DTOs from `internal/catalog` instead of serializing raw `database.CatalogItem` or `database.MediaAsset` rows.
- Contract tests now fail if future changes reintroduce legacy `show` typing or leak DB-only fields such as `deleted_at`, `payload_json`, or `value_json`.

## Self-Check: PASSED

- Verified summary and contract files exist on disk.
- Verified task commits `4482c75`, `2954d25`, `79f5416`, and `d4d90d3` exist in git history.
