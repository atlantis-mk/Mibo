## Why

The new catalog kernel schema exists, but the product still depends heavily on legacy `MediaItem` and `MediaFile` flows for ingestion, metadata, APIs, playback, and UI. Finishing the cutover now is necessary to unlock series-first metadata governance, asset/version-aware playback, and eventual removal of the legacy model without a destabilizing one-shot rewrite.

## What Changes

- Finalize catalog kernel contracts, indexes, migration guards, and projection refresh behavior needed for production-safe rollout.
- Add idempotent backfill from legacy media tables into catalog, asset, file, metadata, and progress tables with reporting.
- Rebuild scanning and metadata matching so new writes target catalog items, media assets, and inventory files instead of legacy media models.
- Cut APIs, playback selection, search, and progress updates over to catalog-backed DTOs and item/asset semantics.
- Move the frontend and metadata governance UI to catalog item types, season/episode hierarchies, image selection, field locks, and asset-aware playback.
- Retire remaining legacy read/write paths and add consistency, indexing, and migration safeguards for the final production cutover.

## Capabilities

### New Capabilities
- `catalog-data-cutover`: Define the catalog kernel write contracts, legacy backfill behavior, and scanner-driven catalog ingestion required to move stored media data onto the new schema.
- `catalog-metadata-governance`: Define series-first metadata matching, provider-backed season and episode generation, field locking, source evidence, and governance state behavior.
- `catalog-api-playback`: Define catalog-backed list/detail/search/progress APIs and item-to-asset playback selection semantics.
- `catalog-frontend-migration`: Define frontend behavior for catalog-backed lists, details, seasons, playback entry, and governance workflows.
- `catalog-cutover-operations`: Define the final cutover, consistency checks, indexing, and legacy cleanup expectations needed for safe production rollout.

### Modified Capabilities

- None.

## Impact

- Backend packages: `mibo-media-server/internal/catalog`, `internal/inventory`, `internal/library`, `internal/metadata`, `internal/httpapi`, `internal/playback`, `internal/progress`, `internal/search`, and database migrations.
- Frontend packages: `web/src/lib`, `web/src/features`, routes in `web/src/App.tsx`, and governance-related UI flows.
- API contracts for item lists, item detail, series seasons, governance endpoints, playback selection, and progress writes.
- Operational behaviors around migration state, backfill reporting, projection rebuilds, indexes, and legacy table retirement.
