## Why

Missing media records are currently retained indefinitely after files disappear from storage. This prevents accidental data loss, but over time it leaves large amounts of invisible catalog, inventory, asset, playback, favorite, and governance data that the user explicitly wants removed.

## What Changes

- Add scheduled cleanup for media records that remain `missing` beyond a retention window.
- Hard delete eligible missing catalog, asset, inventory, and related rows instead of soft deleting them.
- Cascade cleanup through favorites, playback progress, user item data, manual metadata/governance state, external IDs, images, tags, people links, streams, asset links, and scanner evidence tied to the deleted media graph.
- Treat favorites, playback history, manual match, and manual correction as data to delete together with the missing media graph, not as protection from cleanup.
- Add configuration for retention timing and cleanup scope while keeping manual scans focused on marking missing first.
- Preserve immediate scan behavior: missing records are marked during scan, then hard deleted only by cleanup.

## Capabilities

### New Capabilities

### Modified Capabilities

- `media-graph-scanner`: Missing media records will be eligible for scheduled hard deletion after a retention period, including user and governance data linked to those records.

## Impact

- Backend cleanup logic in `mibo-media-server/internal/library` and scheduled job handling in `internal/schedule` / `internal/worker`.
- Database deletion behavior for `catalog_items`, `inventory_files`, `media_assets`, link tables, metadata/governance tables, playback progress, favorites, and related user data.
- Admin/settings API may need cleanup policy fields for missing retention and enablement.
- Existing soft-delete assumptions must be reviewed where cleanup touches catalog and inventory records.
