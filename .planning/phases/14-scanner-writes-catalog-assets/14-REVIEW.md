---
phase: 14-scanner-writes-catalog-assets
reviewed: 2026-04-25T10:28:02Z
depth: standard
files_reviewed: 28
files_reviewed_list:
  - .planning/phases/14-scanner-writes-catalog-assets/14-01-PLAN.md
  - .planning/phases/14-scanner-writes-catalog-assets/14-01-SUMMARY.md
  - .planning/phases/14-scanner-writes-catalog-assets/14-02-PLAN.md
  - .planning/phases/14-scanner-writes-catalog-assets/14-02-SUMMARY.md
  - .planning/phases/14-scanner-writes-catalog-assets/14-03-PLAN.md
  - .planning/phases/14-scanner-writes-catalog-assets/14-03-SUMMARY.md
  - .planning/phases/14-scanner-writes-catalog-assets/14-04-PLAN.md
  - .planning/phases/14-scanner-writes-catalog-assets/14-04-SUMMARY.md
  - mibo-media-server/internal/library/scan.go
  - mibo-media-server/internal/library/service.go
  - mibo-media-server/internal/library/scan_classify.go
  - mibo-media-server/internal/library/scan_run.go
  - mibo-media-server/internal/library/scan_catalog.go
  - mibo-media-server/internal/library/enrichment.go
  - mibo-media-server/internal/library/service_libraries.go
  - mibo-media-server/internal/library/scan_upsert.go
  - mibo-media-server/internal/probe/service.go
  - mibo-media-server/internal/worker/worker.go
  - mibo-media-server/internal/inventory/service.go
  - mibo-media-server/internal/catalog/service.go
  - mibo-media-server/internal/catalog/projections.go
  - mibo-media-server/internal/database/inventory_models.go
  - mibo-media-server/internal/database/catalog_models.go
  - mibo-media-server/internal/library/scan_classify_test.go
  - mibo-media-server/internal/library/scan_catalog_test.go
  - mibo-media-server/internal/library/scan_identity_test.go
  - mibo-media-server/internal/probe/service_inventory_test.go
  - mibo-media-server/internal/worker/worker_catalog_scan_test.go
findings:
  critical: 0
  warning: 3
  info: 0
  total: 3
status: issues_found
---

# Phase 14: Code Review Report

**Reviewed:** 2026-04-25T10:28:02Z
**Depth:** standard
**Files Reviewed:** 28
**Status:** issues_found

## Summary

Reviewed the Phase 14 scanner/catalog cutover, including scan classification, catalog/inventory writes, probe queueing, worker dispatch, targeted tests, and phase summaries. The targeted Go tests pass, but I found three correctness risks: rename rescans can duplicate catalog items, scanner local evidence grows duplicates on every rescan, and parent series/season availability can remain stale after all leaf files disappear.

## Warnings

### WR-01: Stable-identity rename can duplicate catalog items and keep stale item links alive

**File:** `mibo-media-server/internal/library/scan_catalog.go:34-36,46-69,300-349`
**Issue:** Movie item identity is still keyed by `artifact.SourcePath`, while stable-identity reuse only applies to `inventory_files`. On a rename/move rescan, `reuseInventoryFileByStableIdentity` reuses the existing file row, but `writeCatalogScanMovie` creates a new `catalog_items` row for the new path and then links the reused asset to that new item without removing the old link. The existing rename test (`mibo-media-server/internal/library/scan_catalog_test.go:263-315`) only checks file/asset reuse, so this duplicate-item regression is currently untested.
**Fix:** Reuse the existing catalog item when a reused file/asset is found, or make movie item identity canonical and path-independent before linking the asset.

```go
reusedFile, reused := reuseInventoryFileByStableIdentity(...)
if reused {
    item, err := findCatalogItemLinkedToFileOrAsset(tx, reusedFile.ID)
    if err == nil {
        // refresh mutable fields, but do not create a second item
        return relinkOrUpdateExistingItem(tx, item, artifact)
    }
}

item, err := createOrReuseCatalogItem(...)
```

### WR-02: Rescans append duplicate scanner `metadata_sources` rows

**File:** `mibo-media-server/internal/library/scan_catalog.go:73-79,169-175`; `mibo-media-server/internal/catalog/service.go:193-218`
**Issue:** Both scan writers call `RecordMetadataSource` on every scan, and `RecordMetadataSource` always inserts a new row. Repeated scans of unchanged content therefore accumulate duplicate `local_file/scanner` evidence for the same item. That leaks into API `source_evidence` output and causes unbounded row growth for routine rescans.
**Fix:** Upsert the scanner-owned metadata source per item instead of always inserting a new row.

```go
var existing database.MetadataSource
err := tx.Where(
    "item_id = ? AND source_type = ? AND source_name = ? AND external_id = ?",
    item.ID,
    catalog.SourceTypeLocalFile,
    "scanner",
    "",
).First(&existing).Error

if errors.Is(err, gorm.ErrRecordNotFound) {
    _, err = catalogSvc.RecordMetadataSource(ctx, input)
} else {
    err = tx.Model(&existing).Updates(map[string]any{
        "payload_json": input.PayloadJSON,
        "fetched_at":   time.Now().UTC(),
        "updated_at":   time.Now().UTC(),
    }).Error
}
```

### WR-03: Series/season `availability_status` can stay permanently stale after deletes

**File:** `mibo-media-server/internal/library/scan_catalog.go:114-138,266-281,582-613`; `mibo-media-server/internal/catalog/projections.go:234-246`
**Issue:** Series and season rows are created with `availability_status="available"`, but cleanup only recomputes availability for items returned by `scopedCatalogItemIDs`, which are items with direct `asset_items` links. Parent series/season rows have no direct asset links, so after all episode files disappear they keep their old `available` state, and projection refresh then copies that stale status into `catalog_search_documents`.
**Fix:** After recomputing leaf availability, also recalculate ancestor availability from child/item rollups, or derive parent availability during projection refresh instead of persisting a stale parent flag.

```go
leafAndAncestorIDs := collectLeafAndAncestorIDs(tx, libraryID, rootPath)
for _, itemID := range leafAndAncestorIDs {
    availability, err := rollupCatalogAvailability(tx, itemID)
    if err != nil {
        return err
    }
    if err := tx.Model(&database.CatalogItem{}).
        Where("id = ?", itemID).
        Update("availability_status", availability).Error; err != nil {
        return err
    }
}
```

---

_Reviewed: 2026-04-25T10:28:02Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
