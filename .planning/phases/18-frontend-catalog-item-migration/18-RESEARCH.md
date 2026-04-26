# Phase 18 Research — Frontend Catalog Item Migration

**Date:** 2026-04-25
**Phase:** 18 — Frontend Catalog Item Migration
**Requirements:** UI-01, UI-02, UI-03, UI-04

## Research Goal

Answer: what must change in the web client so home, library, search, detail,
series, progress, and playback flows stop depending on legacy `MediaItem` /
`MediaFile` contracts and instead consume catalog-native item, season, episode,
asset, and progress APIs.

## No User Context Artifact

- No phase-specific `CONTEXT.md` exists for Phase 18.
- Research therefore uses `ROADMAP.md`, `REQUIREMENTS.md`, shipped catalog-kernel
  summaries, and the current frontend/backend code paths as the authoritative
  inputs.

## Current Codebase Facts

### Frontend is still fully keyed to legacy media contracts

- `web/src/lib/mibo-api.ts` defines `MediaItem`, `MediaItemDetail`,
  `MediaFile`, `ProgressState`, and `PlaybackSource` with legacy
  `media_item_id` / `media_file_id` fields.
- `web/src/lib/mibo-query.ts` builds query keys around `mediaItemId` and legacy
  progress/detail endpoints.
- `web/src/lib/media-presentation.ts` still normalizes legacy `show` / `episode`
  display state and derives series titles from `series_title` + `source_path`.

### Core user surfaces all call legacy endpoints today

- `web/src/features/home/index.tsx` and `web/src/lib/mibo-query.ts` use
  `recentlyAdded`, `continueWatching`, and `latestByLibrary`, all returning
  legacy browse payloads.
- `web/src/features/library/index.tsx` and `web/src/features/search/index.tsx`
  call `discoverMedia(...)` and render `DiscoveryItem` / `SearchResult`, each
  wrapping `MediaItem`.
- `web/src/features/media/index.tsx` loads `getMediaItem`,
  `getMediaItemProgress`, and `listLocalSeriesEpisodes`, then adapts legacy
  show/episode data into the current detail view.
- `web/src/features/play/index.tsx` requests playback through
  `getPlayback(mediaItemId, { mediaFileId? })` and persists progress with
  `updateProgress({ media_item_id, media_file_id })`.

### Backend catalog DTOs exist, but frontend-cutover routes are not live yet

- `mibo-media-server/internal/catalog/contracts.go` already defines the target
  DTO family: `CatalogListItem`, `CatalogItemDetail`, `CatalogSeasonDetail`,
  `CatalogEpisodeDetail`, `CatalogAssetDetail`, `CatalogGovernanceWorkspace`,
  plus normalized `availability_status`, `governance_status`, selected images,
  child summaries, and asset links.
- `mibo-media-server/internal/httpapi/router.go` still registers only legacy
  frontend-facing browse/detail/progress/playback routes such as
  `/api/v1/libraries/{id}/items`, `/api/v1/discovery`, `/api/v1/media-items/{id}`,
  `/api/v1/media-items/{id}/progress`, and `/api/v1/media-items/{id}/playback`.
- The additive catalog routes recommended in Phase 16 research (`/api/v1/items`,
  `/api/v1/items/{id}`, `/api/v1/series/{id}/seasons`,
  `/api/v1/items/{id}/progress`, `/api/v1/me/item-progress`) are not present yet.
- The catalog playback routes recommended in Phase 17 research
  (`/api/v1/items/{id}/playback`, explicit asset selection, inventory-file
  stream/HLS ids) are also not present yet.

### This phase is intentionally downstream of Phases 16 and 17

- Phase 16 introduces the catalog browse/detail/search/progress APIs the web app
  must consume.
- Phase 17 introduces the catalog item -> asset playback contract the web player
  must use for UI-04.
- Phase 18 planning should therefore create explicit dependencies on those API
  and playback contracts instead of inventing temporary frontend-only shims.

## Recommended Cutover Strategy

### 1. Add catalog-native TypeScript contracts beside legacy ones first

Extend `web/src/lib/mibo-api.ts` with additive frontend types that mirror the
backend catalog DTOs rather than mutating `MediaItem` in place.

Recommended additions:

- `CatalogSelectedImage`
- `CatalogExternalIdentity`
- `CatalogChildSummary`
- `CatalogAssetLink`
- `CatalogAssetDetail`
- `CatalogListItem`
- `CatalogEpisodeDetail`
- `CatalogSeasonDetail`
- `CatalogItemDetail`
- `CatalogProgressState`
- `CatalogPlaybackSource`

This keeps the migration reviewable and lets Phase 18 cut each surface over
deliberately instead of forcing one giant rename.

### 2. Move shared query wiring to catalog-native API helpers

The repo already centralizes data access in `mibo-api.ts` + `mibo-query.ts`.
Follow that boundary.

Recommended additive API methods:

- `listCatalogItems(...)`
- `getCatalogItem(itemId)`
- `listCatalogSeriesSeasons(itemId)`
- `getCatalogItemProgress(itemId)`
- `updateCatalogProgress({ item_id, asset_id, ... })`
- `getCatalogPlayback(itemId, { assetId?, clientProfile })`

Do not place raw `fetch` calls inside `features/home`, `features/library`,
`features/search`, `features/media`, or `features/play`.

### 3. Migrate browse surfaces before detail/playback

Safest order:

1. shared contracts + query keys
2. home / library / search list cards (`CatalogListItem`)
3. detail / series surfaces (`CatalogItemDetail`, `CatalogSeasonDetail`,
   `CatalogEpisodeDetail`, `CatalogAssetDetail`)
4. playback launch + explicit asset/version selection (`CatalogPlaybackSource`)

This follows the user journey from browse -> inspect -> play while keeping
dependencies directional.

### 4. Replace legacy show heuristics with catalog hierarchy truth

The current detail layer uses `buildPresentedMediaItem(...)` and local/TMDB
episode fallback behavior to simulate a series view from legacy rows. Phase 18
should pivot to the catalog truth instead:

- series page comes from `CatalogItemDetail` + `/series/{id}/seasons`
- availability comes from catalog `availability_status` and child summaries
- episode cards should surface `available`, `missing`, `unaired`, and
  `no_local_media` / unavailable states directly instead of collapsing them into
  a single generic state

### 5. Treat playback as item + optional asset selection

`web/src/features/play/index.tsx` currently assumes a single playback URL plus a
legacy `media_file_id`. Phase 18 should consume the Phase 17 contract instead:

- route entry stays `/play/$id` but the underlying request becomes item-based
- explicit version selection should pass `asset_id`
- progress writes should persist `item_id` + `asset_id`
- UI should show a clear disabled / explainable state when no playable asset is
  available

## Architectural Responsibility Map

| Concern | Owning frontend layer | Why |
|--------|------------------------|-----|
| Catalog DTO typing and endpoint methods | `web/src/lib/mibo-api.ts` | single source of truth for backend JSON contracts |
| Shared query keys and composed loaders | `web/src/lib/mibo-query.ts` | current React Query boundary already lives here |
| Browse-card presentation helpers | `web/src/lib/media-presentation.ts` or a sibling catalog-presenter helper | keeps formatting/label logic out of page components |
| Home / library / search surface rendering | `web/src/features/home`, `web/src/features/library`, `web/src/features/search` | feature modules own page composition |
| Detail / series hierarchy rendering | `web/src/features/media` | existing detail route and components already own this flow |
| Playback player and asset-selection UX | `web/src/features/play` plus detail CTA wiring | preserves thin routes and keeps player state local |

## Existing Patterns To Reuse

### Thin route pattern

- `_app.index.tsx`, `_app.library.$id.tsx`, `_app.search.tsx`,
  `_app.media.$id.tsx`, and `play.$id.tsx` already keep routing thin and delegate
  to feature entry components.

### Shared query option pattern

- `homeDataQueryOptions(...)`, `mediaItemDetailQueryOptions(...)`, and
  `mediaItemProgressQueryOptions(...)` show the preferred pattern for composing
  API calls and stable query keys.

### Shared presentation-helper pattern

- `formatMediaCardTitle(...)` and `parseMediaDetailView(...)` demonstrate that
  card/title/view translation logic belongs in `web/src/lib/`, not inline in
  every feature component.

### Mutation invalidation pattern

- `web/src/features/media/index.tsx` invalidates specific query keys after
  rematch/reprobe/progress mutations. Phase 18 should reuse that style for
  catalog progress and playback updates.

## Constraints And Pitfalls

1. **Do not keep shipping legacy IDs through new UI code.** Once a surface moves
   to catalog APIs, it should use `item_id`, optional `asset_id`, and
   inventory-backed playback data end-to-end.
2. **Do not hand-roll fetch logic in feature pages.** Use `mibo-api.ts` and
   `mibo-query.ts` only.
3. **Do not collapse availability states.** `available`, `missing`, `unaired`,
   and `no_local_media` / unavailable states are part of the phase goal.
4. **Do not assume every item has a playable asset.** The detail and playback UI
   must expose explainable non-playable states.
5. **Do not keep legacy show-folder heuristics as the source of truth.** Catalog
   series/season/episode hierarchy should replace `source_path`-driven guesses.
6. **Do not block on frontend lint for phase validation.** Repo notes say
   `pnpm lint` currently fails on unrelated pre-existing issues; use typecheck +
   build as the automated gate.
7. **UI-SPEC is mandatory before planning.** This phase is frontend-heavy and
   the workflow’s UI safety gate is enabled, so planning must stop until a
   `*-UI-SPEC.md` exists.

## Recommended Plan Split

1. **Catalog API typings + shared query/presentation adapters**
   - Add catalog TS types and additive API/query helpers.
2. **Home, library, and search surface migration**
   - Switch browse cards and list pages to `CatalogListItem` data.
3. **Detail and series hierarchy migration**
   - Switch media detail, season rails, and availability messaging to catalog
     detail + hierarchy contracts.
4. **Playback + explicit asset selection migration**
   - Cut player/progress flows over to catalog playback and asset-aware progress.

## Validation Architecture

**Framework:** TypeScript compiler + Vite production build (no dedicated frontend
unit-test harness detected in `web/`)

**Quick commands**

- `cd web && pnpm typecheck`
- `cd web && pnpm build`

**Full phase command**

- `cd web && pnpm typecheck && pnpm build`

**Manual proof after backend Phases 16-17 are available**

- Home renders catalog browse cards from `/api/v1/items`-backed queries.
- Library and search results render catalog titles, selected images, and watched
  state without legacy `MediaItem` assumptions.
- Detail page shows catalog availability states and series seasons/episodes from
  catalog hierarchy.
- Playback launches with item-based playback data and can pass an explicit
  `asset_id` when multiple versions exist.
