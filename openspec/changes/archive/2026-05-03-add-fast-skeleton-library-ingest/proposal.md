## Why

Large library scans still make users wait for catalog projection work before newly discovered videos become browseable. Fast ingest should make playable files visible as soon as storage facts are known, then let classification, probing, metadata matching, artwork, projection, and cleanup converge asynchronously.

## What Changes

- Introduce skeleton ingest semantics for newly discovered video files so scans can persist a stable file-backed visible entry before final movie or episode classification completes.
- Treat `inventory_file` as the durable fast-ingest anchor for discovered content, preserving playback and scan continuity while catalog graph links are refined later.
- Add explicit maturity states for discovered media so clients can distinguish newly discovered, classified, enriched, and review-required content.
- Extend browsing surfaces to show discovered media as media-like cards with "organizing" state instead of exposing a raw file-manager view.
- Move sidecar parsing, artwork selection, detailed catalog classification, probing, metadata matching, and expensive projection refresh work out of the minimal fast-ingest critical path where possible.
- Preserve existing scan correctness by allowing asynchronous classification and reconciliation to upgrade discovered entries into final catalog movie, series, season, and episode graph rows.

## Capabilities

### New Capabilities

- `fast-skeleton-library-ingest`: Defines the user-visible fast ingest contract, discovered media maturity states, and file-anchored upgrade behavior.

### Modified Capabilities

- `media-graph-scanner`: Scanner synchronization requirements change so newly discovered videos can become visible from inventory-backed skeleton records before final catalog projection and enrichment complete.
- `catalog-discovery-sort-filter-contract`: Discovery browse responses must include organizing/discovered entries in library scope when catalog reads are enabled.
- `library-detail-browsing`: Library detail browsing must render discovered entries as media cards with clear organizing state.

## Impact

- Backend scan pipeline in `mibo-media-server/internal/library`, especially `scan_run.go`, catalog write boundaries, enrichment queueing, and missing cleanup ordering.
- Catalog/inventory models and query contracts may need maturity or scan-state fields, or an equivalent query-layer representation based on `inventory_file` state.
- Catalog browse APIs and projections may need to include inventory-backed discovered entries before final catalog graph materialization.
- Frontend library browsing and card components in `web/src/features/library` and shared media card UI need to display organizing states without treating them as final metadata.
- Existing post-scan enrichment jobs for catalog match and inventory probe remain asynchronous and should continue to fail/retry independently from scan completion.
