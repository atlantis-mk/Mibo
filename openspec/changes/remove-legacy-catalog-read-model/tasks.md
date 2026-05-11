## 1. Inventory Remaining Legacy Dependencies

- [x] 1.1 List backend references to `CatalogItem`, `MediaAsset`, `AssetItem`, and `UserItemData` and classify each as product flow, test fixture, or removable dead code.
- [x] 1.2 List frontend references to `CatalogAssetDetail`, `asset_id`, `getCatalogItem`, `listCatalogSeriesSeasons`, scan-exclusion item actions, and catalog-named query helpers.
- [x] 1.3 Confirm replacement resource/projection endpoints needed before deleting each legacy caller.

## 2. Metadata Hierarchy And Detail Replacement

- [x] 2.1 Add or finalize backend metadata hierarchy query methods for series, seasons, episodes, parent context, same-season shelves, and local-only availability filtering.
- [x] 2.2 Add HTTP endpoints or extend existing metadata item detail endpoints to return hierarchy data without `ListSeriesSeasons`.
- [x] 2.3 Update frontend media detail and episode shelves to consume metadata hierarchy/resource projection data instead of `listCatalogSeriesSeasons`.
- [x] 2.4 Add tests for series detail, episode detail, same-season shelves, local-only shelves, and incomplete hierarchy behavior.

## 3. Playback, Progress, Favorites, And Home Cleanup

- [x] 3.1 Remove backend asset-selected playback fallback and ensure playback accepts only metadata item plus optional resource ID for normal product playback.
- [x] 3.2 Remove `UserItemData` fallback from progress writes, progress reads, continue watching, favorites, and home rails.
- [x] 3.3 Update frontend API types and UI consumers to remove `asset_id` and `CatalogAssetDetail` from normal playback/progress/favorites/home flows.
- [x] 3.4 Add tests for resource progress aggregation, continue watching, favorites, home latest/continue rails, and default resource playback without legacy state.

## 4. Governance And File-Level Actions

- [x] 4.1 Replace manual movie-version, independent-movie, and episode-sequence restructure payloads from asset IDs to resource IDs.
- [x] 4.2 Update governance workspaces to expose resource files and resource links instead of linked assets where user correction is needed.
- [x] 4.3 Replace scan exclusion and reprobe actions that depend on catalog item or asset IDs with inventory-file or source-path anchored endpoints.
- [x] 4.4 Add governance tests for resource restructure, resource relink/unlink/update, metadata merge/split, projection visibility, scan exclusion, and reprobe.

## 5. Delete Legacy Architecture

- [x] 5.1 Delete backend query helpers, service methods, route handlers, and DTO fields that only support library-owned `CatalogItem` read semantics.
- [x] 5.2 Delete `MediaAsset`, `AssetItem`, and asset-file read-model usage after resource replacements are active.
- [x] 5.3 Delete or stop AutoMigrating retired legacy database models that are no longer referenced by product code or tests.
- [x] 5.4 Delete frontend catalog/asset API wrappers, query keys, presentation helpers, and type fields that no longer have callers.
- [x] 5.5 Delete or rewrite tests that assert `CatalogItem.library_id`, `AssetItem`, `MediaAsset`, or `UserItemData` behavior.

## 6. Verification And Operations

- [x] 6.1 Run focused backend tests for catalog projections, metadata hierarchy, playback, progress, governance, favorites, home, scan exclusion, and scanner/materialization.
- [x] 6.2 Run full backend `go test ./...` from `mibo-media-server/`.
- [x] 6.3 Run frontend `pnpm typecheck` and `pnpm build` from `web/`.
- [x] 6.4 Reset local development data and rescan demo media with `MIBO_LOCAL_ROOT_PATH=/Users/atlan/Desktop/IdeaProjects/Mibo/demo-media`.
- [x] 6.5 Manually verify library browse, item detail, episode shelves, multi-version playback, progress/continue watching, favorites, search, home dashboard, governance corrections, scan exclusion, and reprobe.
- [x] 6.6 Update `AGENTS.md` runtime notes if endpoint names, reset expectations, or removed legacy model guidance changes.
