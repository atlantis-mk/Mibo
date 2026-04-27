## Why

The current library detail page already provides a dark browsing surface with a media-library title, basic discovery controls, and a poster grid. Compared with an Emby-style library detail page, it still behaves more like a limited search result page: it only requests a capped item set, does not show the true library total, does not support title sort direction, does not expose tabbed browsing dimensions, and does not reuse the poster card behavior that already supports year ranges and green count badges.

Completing this page will make media-library browsing feel like a real media-center library: users can enter a category such as `2019 动漫`, switch between useful browsing dimensions, scan posters with accurate metadata, sort and filter deliberately, and jump through large grids quickly.

## What Changes

- Upgrade `web/src/features/library/index.tsx` from a capped grid into a full library detail browsing experience with total counts, pagination or incremental loading, title sorting, sort direction, and responsive layout behavior.
- Reuse shared poster-card presentation so library cards show poster, title, year/year range, green count badges, and quick actions consistently with homepage/favorites cards.
- Add an Emby-style library toolbar with view tabs for content, recommendations, trailers, favorites, genres, tags, platforms, episodes, and folders where data is available; unsupported dimensions must render clear empty or coming-soon states instead of dead controls.
- Add filter and more-action controls as compact toolbar actions while preserving advanced discovery inputs for desktop and mobile.
- Add a right-side alphabetical/character quick index for title-sorted poster grids and support jumping to sections in long libraries.
- Align the top navigation shell for library detail with the rest of the app by adding home/menu/title on the left and search, cast, user, settings, and optional Mibo upgrade/promo entry on the right.
- Fix frontend/backend discovery contract gaps so library filters and sorting used by the UI actually affect returned catalog results.
- Keep all new product flows catalog-native and avoid retired `/api/v1/media-items/*` or `/api/v1/media-files/*` routes.

## Capabilities

### New Capabilities

- `library-detail-browsing`: Defines the full media-library detail browsing page with complete counts, grid loading, title sorting, sort direction, and reusable poster cards.
- `library-detail-view-dimensions`: Defines tabbed library dimensions for content, recommendations, trailers, favorites, genres, tags, platforms, episodes, and folders with bounded unsupported states.
- `library-detail-alpha-index`: Defines the alphabetical/character index and section-jump behavior for title-sorted library grids.
- `catalog-discovery-sort-filter-contract`: Defines the catalog discovery query contract needed by the library detail UI, including total counts, sort direction, limit/offset or cursor loading, and applied filters.

### Modified Capabilities

- `app-navigation-shell`: Extend library detail top-bar behavior to match the app shell entries already used on the homepage.
- `media-card-progress-badges`: Reuse existing card badge/year-range rules on the library detail grid instead of maintaining a bespoke card implementation.

## Impact

- Frontend: `web/src/features/library/index.tsx`, `web/src/features/discovery/controls.tsx`, `web/src/components/media-poster-card.tsx`, `web/src/components/app-top-bar.tsx`, `web/src/components/app-sidebar.tsx`, `web/src/lib/media-presentation.ts`, `web/src/lib/mibo-api.ts`, and `web/src/lib/mibo-query.ts`.
- Backend: `mibo-media-server/internal/httpapi/handlers_search.go`, `mibo-media-server/internal/httpapi/handlers_libraries.go`, `mibo-media-server/internal/catalog/query.go`, `mibo-media-server/internal/library/query.go`, and `mibo-media-server/internal/library/query_browse.go` if the discovery contract needs total counts, offsets, sort direction, or catalog-native facet data.
- APIs: extend `/api/v1/discovery` or add a catalog-native library browse endpoint that returns `items`, `total`, paging metadata, applied sort/filter state, and optional facet summaries.
- Verification: frontend `pnpm typecheck` and `pnpm build`; focused backend tests for discovery sort/filter/paging behavior and any new facet endpoints.
