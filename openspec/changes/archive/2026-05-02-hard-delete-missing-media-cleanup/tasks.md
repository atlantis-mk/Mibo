## 1. Missing Age Tracking

- [x] 1.1 Add schema fields or an equivalent durable marker to track when inventory files and related media first transition to missing.
- [x] 1.2 Update scan missing-mark logic to set the missing timestamp only on transition from available to missing.
- [x] 1.3 Update recovery/upsert logic to clear missing timestamps when media becomes available again.
- [x] 1.4 Add migration/backfill behavior for existing missing rows.

## 2. Cleanup Policy And Scheduling

- [x] 2.1 Add configurable missing cleanup enablement and retention duration settings.
- [x] 2.2 Wire missing cleanup into the existing scheduled cleanup/worker path with support for library-scoped and global runs.
- [x] 2.3 Return cleanup result counts for files, assets, catalog items, and dependent rows removed.
- [x] 2.4 Keep scans responsible for marking missing only; do not hard delete during `sync_library`.

## 3. Hard Delete Graph Cleanup

- [x] 3.1 Implement missing cleanup candidate selection using missing status plus retention cutoff.
- [x] 3.2 Delete inventory, asset, and catalog dependents in explicit dependency order inside transactions.
- [x] 3.3 Delete favorites, playback progress, user item data, manual metadata/governance state, images, tags, people links, external IDs, scanner evidence, streams, and link rows for deleted media.
- [x] 3.4 Hard delete eligible principal `inventory_files`, `media_assets`, and `catalog_items` rows instead of setting `deleted_at`.
- [x] 3.5 Preserve parent catalog items and shared assets that still have available local media descendants or links.
- [x] 3.6 Refresh or queue catalog projection rebuilds for affected libraries/scopes after cleanup.

## 4. Verification

- [x] 4.1 Add tests proving missing rows older than retention are physically removed, not soft deleted.
- [x] 4.2 Add tests proving rows younger than retention are preserved.
- [x] 4.3 Add tests proving favorites, playback/user data, and manual curation/governance records are deleted with the missing graph.
- [x] 4.4 Add tests proving mixed available/missing series or movie versions preserve available graph branches.
- [x] 4.5 Run `go test ./internal/library ./internal/worker ./internal/schedule ./internal/database` from `mibo-media-server/`.
