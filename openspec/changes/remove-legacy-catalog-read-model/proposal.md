## Why

The resource-first metadata/resource model is now the primary path, but legacy `CatalogItem`/asset read-model fallbacks still keep old ownership semantics alive and make future cleanup risky. Removing the legacy read model now will reduce duplicated behavior, prevent accidental product flows from depending on retired catalog ownership, and complete the architectural cutover started by `redesign-metadata-resource-model`.

## What Changes

- **BREAKING**: Remove product and API dependencies on library-owned `CatalogItem` metadata as a read model.
- Replace remaining series/season/episode detail flows that call legacy catalog item hierarchy endpoints with metadata/resource/projection-backed flows.
- Replace remaining asset-scoped playback/progress/scan-exclusion/restructure paths with resource or inventory-file scoped equivalents.
- Delete old backend query helpers, service methods, handlers, frontend API wrappers, and tests that only assert `CatalogItem.library_id` ownership behavior.
- Keep only bounded development migration/reset behavior; do not preserve compatibility shims for old local databases.

## Capabilities

### New Capabilities
- `legacy-catalog-read-model-retirement`: Defines the removal contract for legacy catalog item, asset, and user-item read-model dependencies after the resource-first cutover.

### Modified Capabilities
- `catalog-data-cutover`: Tighten cutover requirements so retired catalog tables and compatibility read paths are not used by product flows.
- `catalog-api-playback`: Require playback and progress to use metadata/resource identifiers, not asset identifiers.
- `library-detail-browsing`: Require browse/detail hierarchy reads to come from metadata/resource/projection data.
- `catalog-metadata-governance`: Require manual correction flows to operate on metadata items and resources rather than catalog items and assets.
- `favorites-browsing`: Require user data reads/writes to use metadata/resource state without `UserItemData` fallback.

## Impact

- Backend: `internal/catalog`, `internal/httpapi`, `internal/playback`, `internal/progress`, `internal/metadata`, `internal/library`, and database AutoMigrate/model registration.
- Frontend: `web/src/lib/mibo-api.ts`, `web/src/lib/mibo-query.ts`, media detail, episode/series browsing, scan exclusion actions, metadata governance, playback, favorites, home rails.
- Tests: rewrite or delete legacy catalog/asset/user-item tests and add coverage for equivalent metadata/resource/projection behavior.
- Operations: local development reset remains required; old SQLite data is not migrated for this removal.
