---
phase: 12-catalog-kernel-contracts-migration-guards
reviewed: 2026-04-25T06:37:38Z
depth: standard
files_reviewed: 22
files_reviewed_list:
  - mibo-media-server/internal/app/app.go
  - mibo-media-server/internal/catalog/contracts.go
  - mibo-media-server/internal/catalog/contracts_test.go
  - mibo-media-server/internal/catalog/projections.go
  - mibo-media-server/internal/catalog/projections_test.go
  - mibo-media-server/internal/catalog/service.go
  - mibo-media-server/internal/database/catalog_models.go
  - mibo-media-server/internal/database/catalog_models_test.go
  - mibo-media-server/internal/database/database.go
  - mibo-media-server/internal/database/database_open_test.go
  - mibo-media-server/internal/httpapi/catalog_migration_router_test.go
  - mibo-media-server/internal/httpapi/handlers_system.go
  - mibo-media-server/internal/httpapi/router.go
  - mibo-media-server/internal/httpapi/router_test.go
  - mibo-media-server/internal/library/scan_run.go
  - mibo-media-server/internal/library/service.go
  - mibo-media-server/internal/library/service_libraries.go
  - mibo-media-server/internal/settings/catalog_migration.go
  - mibo-media-server/internal/settings/catalog_migration_test.go
  - mibo-media-server/internal/settings/service.go
  - mibo-media-server/internal/worker/worker.go
  - mibo-media-server/internal/worker/worker_catalog_test.go
findings:
  critical: 0
  warning: 3
  info: 0
  total: 3
status: issues_found
---

# Phase 12: Code Review Report

**Reviewed:** 2026-04-25T06:37:38Z
**Depth:** standard
**Files Reviewed:** 22
**Status:** issues_found

## Summary

Reviewed the Phase 12 backend implementation with extra focus on the 12-05 and 12-06 gap-closure changes in catalog contract projection and projection rebuild persistence.

The previously verified raw-JSON contract leak and search-document canonicalization gap are fixed in the current implementation, and the focused Phase 12 test slices for `internal/catalog`, `internal/settings`, `internal/database`, `internal/httpapi`, and `internal/worker` all pass in the current worktree. Remaining issues are narrower migration-safety problems around rollup canonicalization and error classification.

## Warnings

### WR-01: Projection rollups still skip blank migration-era availability values

**File:** `mibo-media-server/internal/catalog/projections.go:186-193`
**Issue:** `buildItemRollups()` still switches on `strings.TrimSpace(descendant.AvailabilityStatus)` instead of the catalog normalization helper. A legacy leaf row with blank availability now gets written to `catalog_search_documents` as `no_local_media`, but the same rebuild leaves `ItemRollup.MissingCount` unchanged because the blank value falls through the switch. That creates an inconsistent migration state where child summaries under-report missing content while the search projection reports the canonical availability.
**Fix:** Normalize before counting so rollups and search documents use the same contract.

```go
switch normalizeAvailabilityStatus(descendant.AvailabilityStatus) {
case AvailabilityAvailable:
    rollup.AvailableCount++
case AvailabilityUnaired:
    rollup.UnairedCount++
case AvailabilityMissing, AvailabilityNoLocalMedia:
    rollup.MissingCount++
}
```

### WR-02: The new projection tests do not guard the remaining rollup canonicalization gap

**File:** `mibo-media-server/internal/catalog/projections_test.go:34-140`
**Issue:** The new 12-06 tests prove canonical `catalog_search_documents` output for a blank root item, but they never seed a blank descendant leaf and assert the rebuilt rollup counts. That is why the rollup inconsistency above can still ship even though the focused projection suite is green.
**Fix:** Add a regression that refreshes a library with at least one blank-availability leaf item and assert both `catalog_search_documents.availability_status == no_local_media` and the parent rollup's `MissingCount` increment.

### WR-03: Catalog-migration settings handlers still misclassify server faults as client errors

**File:** `mibo-media-server/internal/httpapi/handlers_system.go:131-155`
**Issue:** `handleGetCatalogMigrationSettings()` and `handleUpdateCatalogMigrationSettings()` return `400 Bad Request` for any `settings.Service` failure. That is correct for JSON decode/validation failures, but wrong for corrupt persisted state or database errors from `GetCatalogMigrationState()` / `UpdateCatalogMigrationState()`. Operators will see a misleading client error for a server-side fault.
**Fix:** Keep `400` for request decoding/validation only, return `500` for settings-service failures, and add a router test that seeds an invalid stored catalog-migration value.

```go
result, err := r.settings.GetCatalogMigrationState(req.Context())
if err != nil {
    writeError(req.Context(), w, http.StatusInternalServerError, err)
    return
}
```

---

_Reviewed: 2026-04-25T06:37:38Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
