---
phase: 14-scanner-writes-catalog-assets
plan: 01
subsystem: scanner
tags: [catalog, inventory, scanner, sqlite, gorm]
requires:
  - phase: 14-scanner-writes-catalog-assets
    provides: phase plan and validation contract for scanner catalog-write cutover
provides:
  - Catalog-first scan writer helpers for movie and episode hierarchy artifacts
  - Direct regression coverage for movie kernel writes and episode hierarchy writes
  - Library service wiring for shared catalog and inventory services
affects: [scanner, catalog, inventory]
tech-stack:
  added: []
  patterns: [catalog-first write boundary, local-file evidence sources, direct kernel-write regression tests]
key-files:
  created:
    - mibo-media-server/internal/library/scan_catalog.go
    - mibo-media-server/internal/library/scan_catalog_test.go
  modified:
    - mibo-media-server/internal/library/service.go
    - mibo-media-server/internal/library/scan.go
key-decisions:
  - "Keep the new scan writer boundary inside the library package so later scan-loop cutover can switch callers without rediscovering persistence rules."
  - "Record only compact scanner local-file evidence in metadata_sources and avoid raw provider blobs."
  - "Create or reuse catalog hierarchy rows by canonical path while keeping asset and file creation inside the same write flow."
patterns-established:
  - "Scanner kernel writes now follow catalog item creation/reuse -> inventory file upsert -> asset/file/item links -> metadata source evidence."
requirements-completed: [SCAN-01, SCAN-02]
completed: 2026-04-25
---

# Phase 14 Plan 01: Catalog-first scan writer boundary summary

Implemented the first Phase 14 cutover step by adding reusable catalog-first scan writer helpers and locking the expected movie and episode hierarchy write shape in focused tests.

## Accomplishments

- Added `catalog` and `inventory` service wiring to `library.Service` through `catalog.NewService(db)` and `inventory.NewService(db)`.
- Added scan-writer contract types in `scan.go` for direct catalog write artifacts and episode slot lists.
- Implemented `writeCatalogScanMovie` and `writeCatalogScanEpisodeHierarchy` in `scan_catalog.go`.
- Added regression tests covering movie writes, episode hierarchy creation, scanner asset links, and allowlisted local evidence payloads.

## Verification

- `cd mibo-media-server && go test ./internal/library -run 'TestScanCatalogWriter' -count=1`
- `cd mibo-media-server && go test ./internal/library -count=1`

## Next Readiness

- The scan loop can now be switched to this boundary in `14-02` without introducing catalog/inventory write semantics at the same time.
- Episode hierarchy creation already uses canonical paths and scanner-owned local evidence, which reduces risk for the cutover wave.
