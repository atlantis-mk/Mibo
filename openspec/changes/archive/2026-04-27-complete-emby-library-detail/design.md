## Context

The current library detail route is `/_app/library/$id`, implemented by `web/src/features/library/index.tsx`. It loads `api.getLibrary(libraryId)` and `api.discoverMedia({ scope: 'library', library_id, ...filters, limit: 50 })`, then renders a custom poster grid.

Several Emby-like behaviors already exist elsewhere but are not used here. `web/src/components/media-poster-card.tsx` supports green count badges through `getMediaCardBadgeCount()`, poster/title/year presentation, quick play, and favorite actions. `web/src/lib/media-presentation.ts` has `formatMediaCardYearRange()`, which handles series year ranges. The homepage top bar already has examples for search, cast, user menu, settings, and logout. Favorites are also available through the existing favorites route and API client methods.

The main gaps are structural and contractual. The library detail page shows only the number of loaded items, not the total library count. It hardcodes `limit: 50`. It has a sort field but no sort direction. The current `/api/v1/discovery` handler parses rich filters but routes catalog reads through `r.catalog.ListItems` or `r.catalog.SearchItems`, so many discovery inputs such as genre, region, year, min rating, watched state, and sort can be ignored depending on the catalog path.

## Goals / Non-Goals

**Goals:**

- Show a complete, accurate library browse experience with real total counts such as `共 79 项`.
- Support title sorting with ascending and descending direction.
- Support enough paging or incremental loading to browse beyond the first 50 items.
- Reuse shared poster-card presentation for count badges, year ranges, quick play, and favorite state.
- Add top view tabs for Emby-like dimensions while only enabling dimensions backed by current data or clear empty/coming-soon states.
- Add a right-side alphabetical/character index for fast jumping in large title-sorted grids.
- Align library detail top navigation with the app shell entries used on the homepage.
- Make backend discovery behavior match the UI controls exposed on the page.

**Non-Goals:**

- Do not implement a real paid `Emby Premiere` clone. A Mibo-specific promo/upgrade button may be a placeholder or omitted unless product requirements exist.
- Do not implement real Chromecast/AirPlay device discovery as part of this change; cast can use the existing unavailable dialog pattern.
- Do not build new flows on retired legacy media read routes.
- Do not require complete metadata for tags, platforms, trailers, or recommendations before shipping the base library page; unsupported dimensions should be bounded.
- Do not redesign media detail or playback pages beyond navigation links from the library page.

## Decisions

### Decision: Extend the existing library route

Implement the upgraded experience in `web/src/features/library/index.tsx` rather than creating a parallel route.

Rationale: The route already owns library ID parsing, auth handling, data loading, and basic grid rendering. Extending it preserves existing navigation and avoids duplicated route state.

Alternative considered: Add a separate `/library/$id/browse` route. This was rejected because it increases routing complexity without a user-visible benefit.

### Decision: Reuse `MediaPosterCard` for grid items

The library grid should use the shared `MediaPosterCard` or a small variant of it instead of the current bespoke article markup.

Rationale: The shared card already contains the missing green badge behavior, quick play actions, and year range helpers. Reusing it prevents drift across homepage, favorites, search, and library screens.

Alternative considered: Copy badge and year-range code into the library page. This was rejected because it creates duplicate presentation logic.

### Decision: Add sort direction to the discovery contract

The UI should maintain `sort` and `sortDirection` state, with title sorting defaulting to ascending and a toggle for descending. The backend should accept a safe `sort_direction=asc|desc` or equivalent field and apply it consistently.

Rationale: Sorting only by field is insufficient for Emby parity and users need deterministic control in large libraries.

Alternative considered: Reverse items client-side. This was rejected for paged results because client reversal only works for fully loaded datasets.

### Decision: Return total and page metadata from catalog discovery

The browse response should include `items`, `total`, and either `limit/offset` or cursor metadata. The frontend should display total count from the server, not from the currently loaded slice.

Rationale: `items.length` is wrong when the backend caps results. Accurate totals and paging are prerequisites for `共 79 项` and long-library browsing.

Alternative considered: Use `LibraryDetail.media_items_count`. This can be a fallback but may not reflect active filters or grouped show-level rows.

### Decision: Build tabs incrementally with bounded states

Tabs should be present for the requested dimensions, but each tab must either load real data or show a clear state explaining that the dimension has no data or is not yet connected.

Rationale: This gives users the expected navigation model without pretending unsupported data exists.

Alternative considered: Hide unsupported tabs. This was rejected because the requested comparison explicitly calls out the missing dimensions and hiding them makes the gap invisible.

### Decision: Generate alpha index from loaded title sections first

The right-side index can initially be generated from the current title-sorted item list and jump to rendered section anchors. If server paging is used, the index should either cover loaded sections only or be upgraded with server-side title buckets.

Rationale: A client-side section index is enough for the first implementation and avoids a larger facet API unless needed.

Alternative considered: Add server-side alpha buckets immediately. This may be useful later, but it is not required to provide jump behavior for loaded grids.

## Risks / Trade-offs

- Discovery contract changes can affect search and other consumers; keep new fields optional and preserve existing default behavior.
- Filtering and grouping semantics may differ between catalog items and legacy media items; prefer catalog-native item list behavior and avoid retired read routes.
- A right-side index on mobile can crowd the grid; hide it or collapse it behind a compact control on small viewports.
- Showing unsupported tabs can disappoint users; copy must be explicit and avoid broken-looking empty pages.
- Accurate totals for grouped shows may require backend work if the current catalog count is raw items rather than browse rows.
- Client-side alpha sections only represent loaded items if paging is partial; document or avoid index display until enough items are loaded.

## Migration Plan

1. Audit existing library, discovery, favorites, progress, and catalog DTOs for fields needed by tabs, total counts, paging, sort direction, badge counts, and facet labels.
2. Extend the TypeScript discovery filter state and API client to include sort direction and paging metadata.
3. Extend or replace backend discovery response so `/api/v1/discovery` returns accurate totals and applies the filters/sorts already parsed by `discoveryInputFromRequest`.
4. Refactor the library grid to use shared poster-card presentation and favorite/play actions.
5. Add the library toolbar with tabs, sort controls, filter action, more-action menu, and clear empty states for unsupported dimensions.
6. Add title-section grouping and right-side alpha/character jump navigation for title-sorted grids.
7. Align the library top bar with homepage shell actions: home/menu/title, search, cast, user menu, settings, and optional Mibo promo action.
8. Add backend and frontend tests for discovery contract changes, then run typecheck/build.

Rollback strategy: keep the old grid behavior available through default discovery parameters. If backend paging or total count changes need rollback, the frontend can temporarily fall back to loaded-item counts and hide alpha indexing while preserving the rest of the page.

## Open Questions

- Should the page display all Emby-style tabs immediately with empty states, or only the tabs backed by real data plus disabled placeholders?
- Should `播出平台` map to metadata networks/providers, production companies, file/source provider, or a future normalized catalog facet?
- Should `标签` use metadata keywords, user-defined tags, or both?
- Should the green badge count prioritize unwatched episodes, available child count, or total child count on the library page?
- Should `文件夹` browse physical storage paths, catalog folder groups, or source adapter directories?
