## 1. Discovery Contract And Data Loading

- [x] 1.1 Audit current `/api/v1/discovery`, `/api/v1/libraries/{id}`, favorites, progress, and catalog list DTOs for library detail needs: total count, filtered count, sort direction, paging, badge count, favorite state, and available facets.
- [x] 1.2 Decide whether to extend `/api/v1/discovery` or add a catalog-native library browse endpoint; keep existing discovery defaults backward-compatible.
- [x] 1.3 Add `sort_direction` support with accepted values `asc` and `desc`; apply it server-side for title, year, recent, and watch-status sorting where meaningful.
- [x] 1.4 Add response metadata for `total`, `limit`, `offset` or cursor, and whether more results are available.
- [x] 1.5 Ensure server-side filtering honors the UI controls already parsed by `discoveryInputFromRequest`: type, genre, region, year, minimum rating, watched state, query, sort, and library scope.
- [x] 1.6 Add focused backend tests for library-scoped discovery total counts, title ascending/descending order, paging beyond 50 items, and applied filters.

## 2. Frontend State And API Types

- [x] 2.1 Extend `DiscoveryFilters` with `sortDirection` and any paging state needed by the library detail page.
- [x] 2.2 Update `DiscoveryQuery` and `discoverMedia()` in `web/src/lib/mibo-api.ts` to send sort direction and paging parameters and receive response metadata.
- [x] 2.3 Update query keys in library detail so sort, direction, filters, active tab, and paging state produce correct cache behavior.
- [x] 2.4 Display server-provided total count in the library header using Chinese copy such as `共 79 项`.
- [x] 2.5 Add incremental loading or pagination controls so libraries larger than the first page are browsable.

## 3. Poster Grid Presentation

- [x] 3.1 Replace the bespoke card markup in `web/src/features/library/index.tsx` with `MediaPosterCard` or a shared grid variant.
- [x] 3.2 Ensure cards show poster, title, `formatMediaCardYearRange()` output, and green count badge from `getMediaCardBadgeCount()`.
- [x] 3.3 Wire quick play/continue and favorite/unfavorite actions where the existing API and card props support them.
- [x] 3.4 Preserve responsive grid density for desktop and mobile; avoid fixed widths that break full-grid layout.
- [x] 3.5 Add empty, loading, and error states that remain useful for both the all-content tab and filtered/unsupported tabs.

## 4. Library Toolbar And View Dimensions

- [x] 4.1 Add top tabs for `节目`, `推荐`, `预告`, `收藏`, `类型`, `标签`, `播出平台`, `集`, and `文件夹`.
- [x] 4.2 Map `节目` to the primary library item grid.
- [x] 4.3 Map `收藏` to user-scoped favorite items filtered to the current library.
- [x] 4.4 Map `集` to episode-level browsing when the library contains series content, using catalog-native episode data.
- [x] 4.5 Add bounded empty or coming-soon states for `推荐`, `预告`, `类型`, `标签`, `播出平台`, and `文件夹` until reliable facet/trailer/folder data is available.
- [x] 4.6 Add compact `筛选` and `更多` toolbar actions; keep advanced controls accessible without consuming excessive vertical space.
- [x] 4.7 Add title sort control and ascending/descending toggle with visible active state.

## 5. Alphabetical / Character Index

- [x] 5.1 Group title-sorted grid items into stable section keys using title initials, numbers, symbols, and CJK-friendly fallback behavior.
- [x] 5.2 Render a right-side vertical quick index on desktop and a compact/mobile-safe alternative on small screens.
- [x] 5.3 Add section anchors and jump behavior without breaking normal page scroll.
- [x] 5.4 Hide or disable the index when the active sort is not title-based or when too few sections exist.
- [x] 5.5 Verify index behavior with Chinese titles, numeric titles, Latin titles, and symbols.

## 6. Navigation Shell Alignment

- [x] 6.1 Update the library detail top-left area with menu trigger, home entry, and current library/category title.
- [x] 6.2 Add top-right search entry, cast entry with existing unavailable-dialog behavior, user menu, and settings entry.
- [x] 6.3 Decide whether to add a Mibo-specific promo/upgrade button; if added, route it to a clear placeholder or product surface rather than copying Emby Premiere semantics.
- [x] 6.4 Keep top-bar actions keyboard-accessible and usable on mobile.

## 7. Verification

- [x] 7.1 Run `pnpm typecheck` from `web/` and fix introduced type errors.
- [x] 7.2 Run `pnpm build` from `web/` and fix introduced build errors.
- [x] 7.3 Run focused backend tests for discovery contract changes, or `go test ./internal/httpapi ./internal/catalog ./internal/library` from `mibo-media-server/` if touched broadly.
- [x] 7.4 Manually verify a library with more than 50 items shows accurate total count and can browse beyond the first page.
- [x] 7.5 Manually verify title ascending/descending sorting, filters, tabs, alpha index, card badges/year ranges, favorite actions, quick play, and top-bar actions on desktop and mobile.
