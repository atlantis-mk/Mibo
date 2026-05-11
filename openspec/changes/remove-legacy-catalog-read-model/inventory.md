## Backend Legacy Dependency Inventory

### Product Flow Dependencies
- `internal/httpapi/handlers_catalog.go`: `handleGetCatalogItem`, `handleListCatalogSeriesSeasons`, scan-exclusion item handlers, and playback/progress calls still expose catalog-item named product routes.
- `internal/catalog/query.go`: `GetItemDetailForUser`, `ListSeriesSeasons`, related item lookup, latest item fallback, selected image mapping, people details, asset detail loading, and user progress enrichment still depend on `database.CatalogItem`, `database.MediaAsset`, and `database.AssetItem`.
- `internal/catalog/user_items.go`: continue watching and favorites still build entries from `database.UserItemData` and `database.CatalogItem` fallback rows.
- `internal/progress/service.go`: legacy progress update/read path still writes and reads `database.UserItemData`, validates `database.CatalogItem`, and optionally validates `database.AssetItem`.
- `internal/playback/service.go`: legacy playback path still resolves `database.CatalogItem`, `database.MediaAsset`, `database.AssetItem`, and asset files. Resource playback still reuses helper structs for stream formatting.
- `internal/metadata/service_governance.go` and `internal/metadata/service_governance_correction.go`: legacy governance field/image/classification/manual restructure operations still load `database.CatalogItem` and, for restructure, write `MediaAsset`/`AssetItem` relationships.
- `internal/library/materialize.go` and scan catalog code: scanner/materialization still contains catalog-kernel writes that must be replaced or deleted once resource graph writes are complete.

### Test Fixture Dependencies
- `internal/catalog/*_test.go`: many tests seed `CatalogItem`, `MediaAsset`, `AssetItem`, and `UserItemData` to verify old detail, query, series playback target, and favorites behavior.
- `internal/metadata/service_governance_test.go`: several tests seed catalog items/assets for legacy governance corrections.
- `internal/playback/*_test.go` and `internal/progress/*_test.go`: both legacy and resource-first coverage exist; legacy cases need deletion or rewrite after replacement.
- `internal/library/*_test.go`: scanner/materialization tests still assert catalog-kernel outputs in places.

### Removable Dead Code Candidates
- Asset-selected playback API handling after all normal playback callers use `resource_id`.
- Asset-link governance handlers already removed; remaining asset governance service methods were removed during the prior cutover cleanup.
- Catalog item detail and series-season helper methods after metadata hierarchy/detail endpoints are active.
- `UserItemData` conversion helpers after continue watching, favorites, and progress read/write are resource-only.

## Frontend Legacy Dependency Inventory

### Product Flow Dependencies
- `web/src/lib/mibo-api.ts`: `CatalogAssetDetail`, `asset_id` fields, `getCatalogItem`, `listCatalogSeriesSeasons`, item scan-exclusion helpers, and catalog-named progress/favorites contracts remain.
- `web/src/lib/mibo-query.ts`: `catalogItemDetailQueryOptions` and `catalogSeriesSeasonsQueryOptions` still back media detail and play page data loads.
- `web/src/features/media/index.tsx`: media detail still starts from `getCatalogItem`, uses `listCatalogSeriesSeasons`, and passes asset-derived scan-exclusion/progress context.
- `web/src/features/media/components/*`: presentation utilities still read `CatalogAssetDetail` for technical detail display and primary asset fallback.
- `web/src/features/play/index.tsx`: play page still loads catalog item detail and series seasons, and includes legacy `asset_id` in progress update when playback returns one.
- `web/src/features/home/home-sections.tsx`, `web/src/features/favorites/index.tsx`, and `web/src/components/media-poster-card.tsx`: compatibility fields such as `asset_id` are still tolerated for keys/progress/action payloads.
- `web/src/features/metadata-governance/*`: manual restructure payloads and asset display panels still use `assetId`/`asset_id` and `CatalogAssetDetail`.
- Scan exclusion UI remains settings-level, but item/card-level scan exclusion actions still target item scan-exclusion helpers.

### Replacement Map
- `getCatalogItem` -> metadata item detail endpoint returning metadata fields, library projection context, parent/sibling hierarchy, resource summary, and technical resource/file summaries.
- `listCatalogSeriesSeasons` -> metadata hierarchy endpoint or expanded metadata detail payload for series/episode shelves.
- `CatalogAssetDetail`/`asset_id` normal playback -> `MetadataResourceDetail`/`resource_id`.
- `UserItemData` progress/favorites -> `UserMetadataData` and `UserResourceData`.
- Item/asset scan exclusion -> inventory-file or source-path anchored scan exclusion endpoints.
- Manual restructure `asset_id` payloads -> resource IDs and resource metadata link operations.

## Endpoint Gaps To Close Before Deletion
- Metadata hierarchy read endpoint for series seasons, same-season siblings, episode parent context, and local-only filtering.
- Resource technical detail endpoint or expanded resource list that carries files/streams currently shown through `CatalogAssetDetail`.
- Inventory-file/source-path scan exclusion preview/apply endpoint for card/detail actions.
- Resource-based manual restructure endpoints for movie versions, independent movies, and episode sequences.
- Resource-only continue watching/favorites/home DTOs with no `asset_id` or `UserItemData` fallback.
