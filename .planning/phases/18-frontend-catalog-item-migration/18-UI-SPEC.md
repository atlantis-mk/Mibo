# Phase 18 UI Spec — Frontend Catalog Item Migration

**Date:** 2026-04-25
**Phase:** 18 — Frontend Catalog Item Migration
**Status:** Draft
**Requirements:** UI-01, UI-02, UI-03, UI-04

## Goal

Define the user-visible cutover from legacy `MediaItem` / `MediaFile` UI assumptions to catalog-native item, season, episode, asset, and progress contracts without changing the existing product visual language.

This spec covers `/`, `/library/:id`, `/search`, `/media/:id`, and `/play/:id`.

## Dependencies

This UI spec assumes the additive backend routes from Phases 16 and 17 exist before implementation:

- `GET /api/v1/items`
- `GET /api/v1/items/{id}`
- `GET /api/v1/series/{id}/seasons`
- `GET /api/v1/items/{id}/progress`
- `POST /api/v1/me/item-progress`
- `GET /api/v1/items/{id}/playback`

If those routes are not live, the frontend phase stops at additive typing and query wiring only.

## Shared UI Rules

1. Browse surfaces use `CatalogListItem` as the primary card unit.
2. Detail surfaces use `CatalogItemDetail`, `CatalogSeasonDetail`, `CatalogEpisodeDetail`, and `CatalogAssetDetail` directly.
3. Playback and progress use `item_id` plus optional `asset_id` end-to-end.
4. Availability is always visible and never collapsed into a generic “unavailable” label.
5. Selected images come from `selected_images` first; no new heuristics based on `source_path` or legacy series title inference are allowed once a surface is migrated.
6. Existing visual tone stays intact: cinematic hero, rounded cards, glassy overlays, compact badge rows, and large typography.

## Shared Presentation Rules

### Images

- Poster cards prefer selected poster image.
- Hero backgrounds prefer selected backdrop image, then poster image fallback.
- Title logo areas prefer selected logo image, then plain text fallback.

### Availability Labels

- `available` -> `可播放`
- `missing` -> `缺少文件`
- `unaired` -> `未播出`
- `no_local_media` -> `没有本地媒体`
- any other non-available value -> `暂不可用`

Every browse card and detail hero must show one availability badge.

### Type Labels

- `movie` -> `电影`
- `series` -> `剧集`
- `season` -> `季度`
- `episode` -> `剧集`

### Watched State

- Continue watching and progress chips keep the current `未看` / `观看中` / `已看` wording.
- Completion state comes from catalog progress, not from legacy file-level assumptions.

## Screen Specs

### Home `/`

**Purpose:** show recently added catalog items and library rails without legacy show-folder heuristics.

**Data:** top-level `CatalogListItem` browse results plus per-user progress summaries.

**Layout:** keep the current full-screen hero carousel and horizontal library rails.

**Hero slide content**

- Background: selected backdrop or poster.
- Title: item title only.
- Meta row: type, year, optional series status.
- Body: overview or existing generic fallback copy.
- Primary CTA:
  - `available` -> `播放`
  - otherwise -> disabled `暂不可播放`
- Secondary CTA: `详情`
- Status badge: availability badge is always visible near the meta row.

**Library rail cards**

- Poster image, title, year, type badge, library badge, availability badge.
- No derived `series_title` label from source path.
- Cards always deep-link to `/media/:id`.

**Empty state**

- Keep current empty-home shell.
- Copy changes from “没有可轮播的媒体内容” to catalog-neutral wording.

### Library `/library/:id`

**Purpose:** browse a library through catalog list items and explicit availability states.

**Layout:** keep current filter header and responsive poster grid.

**Card content**

- Poster image.
- Title.
- Year.
- Type badge.
- Watched-state badge.
- Availability badge.
- Created-at label.

**Behavior**

- `series` cards open `/media/:id?view=series`.
- `movie` and `episode` cards open `/media/:id`.
- Missing and unaired items are still rendered as normal cards; they are not filtered out or visually hidden.

**Empty state**

- Keep the current card-shell empty state.
- Copy should mention that the library currently has no catalog items matching the filters.

### Search `/search`

**Purpose:** show catalog-native search results with explicit type and availability.

**Layout:** keep current top bar, search history chips, and responsive result card list.

**Result card content**

- Type badge.
- Year.
- Watched state.
- Availability badge.
- Title.
- Optional highlight excerpt.

**Behavior**

- Results follow the same deep-link rules as library cards.
- Search result rendering must not assume every non-movie is a legacy `show` row.

### Detail `/media/:id`

**Purpose:** show canonical catalog item detail and expose playability, hierarchy, and version choices.

**Layout:** preserve the existing standalone detail experience: poster column, hero content, cast, episode rails, metadata/spec panels.

**Hero section**

- Title/logo area uses selected logo if available.
- Meta row uses catalog-native year/date/runtime/rating/status fields.
- Primary action rules:
  - one playable asset -> `播放`
  - resumable progress on selected/default asset -> `继续播放`
  - multiple playable assets -> `选择版本`
  - no playable assets -> disabled `暂不可播放`
- Secondary action rules:
  - resumable progress exists -> `从头播放`
  - metadata/governance actions remain where already present
- Availability badge is visible even when a play button exists.
- A short reason banner appears for non-playable items, using asset or playback reasons where available.

**Progress block**

- Reads catalog progress for the item and selected/default asset.
- Shows percent, elapsed time, and completion text.
- “标记看完” writes catalog progress with `completed: true` and the active asset when one exists.

**Asset section**

- New section below hero or above specs.
- Each asset card shows:
  - `display_name`
  - edition or quality label
  - duration
  - asset status
  - probe status
- Asset card actions:
  - playable asset -> `播放此版本`
  - non-playable asset -> disabled button with inline reason text
- Default asset is visually highlighted.

**Series and season hierarchy**

- Series pages use `CatalogItemDetail` plus `/series/{id}/seasons`.
- Season rail headers show season title/index and child-summary counts when present.
- Episode cards show:
  - episode number
  - title
  - air date
  - runtime
  - availability badge
  - optional still image
- Episode cards link only when the episode item exists.
- `unaired` and `missing` episodes remain visible with their state badges.

**Specs panel**

- Keep current structure.
- Prefer catalog fields and selected asset data over legacy primary-file assumptions whenever the selected asset is known.

### Playback `/play/:id`

**Purpose:** play a catalog item through its default or explicitly selected asset and persist asset-aware progress.

**Route contract**

- Route path stays `/play/:id`.
- Optional search params:
  - `fromStart`
  - `assetId`

**Entry behavior**

- No `assetId` provided:
  - one playable asset -> auto-play it
  - multiple playable assets -> use backend default selection and show which version was chosen
  - no playable assets -> show unplayable screen
- `assetId` provided:
  - request playback for that asset
  - if unplayable, show the asset-specific reason

**Player chrome additions**

- Small badge row near the title shows selected version and availability/playability summary.
- If multiple playable assets exist, provide a lightweight “切换版本” action that returns to detail or opens an in-player selector.

**Unplayable state**

- Full-page error shell keeps current black theater styling.
- Show:
  - item title
  - selected asset label when relevant
  - primary reason text from playback decision
  - secondary CTA back to detail page

**Progress persistence**

- Save progress against `item_id` and selected `asset_id`.
- Resume restores from the matching asset progress.
- Completion writes must stay idempotent.

## Component Ownership

- `web/src/lib/mibo-api.ts`: catalog DTOs and endpoint methods.
- `web/src/lib/mibo-query.ts`: catalog query keys and composed loaders.
- `web/src/lib/media-presentation.ts` or sibling helper: catalog titles, badges, selected-image helpers.
- `web/src/features/home/*`: hero and library-rail browse migration.
- `web/src/features/library/index.tsx`: library browse grid migration.
- `web/src/features/search/index.tsx`: search result migration.
- `web/src/features/media/*`: detail, asset, and series hierarchy migration.
- `web/src/features/play/index.tsx`: asset-aware playback and progress.

## Requirement Mapping

- **UI-01:** home, library, and search render catalog list items and surface availability state.
- **UI-02:** detail renders catalog item fields, selected images, and asset inventory.
- **UI-03:** series view renders catalog season and episode hierarchy with `available` / `missing` / `unaired` / `no_local_media` states visible.
- **UI-04:** playback accepts optional `asset_id`, persists asset-aware progress, and explains unplayable outcomes.

## Acceptance Criteria

1. No migrated browse/detail/playback surface depends on `series_title`, `source_path`, `media_item_id`, or `media_file_id` for its primary catalog UI state.
2. Home, library, and search show availability badges without hiding missing or unaired items.
3. Detail pages expose asset/version information and playability instead of assuming a single primary file.
4. Series pages use catalog hierarchy instead of TMDB-or-local fallback synthesis.
5. Playback works with both default asset selection and explicit `assetId` selection.
6. `cd web && pnpm typecheck && pnpm build` remains green.

## Out Of Scope

- Metadata governance redesign from Phase 19.
- Visual redesign of the shared app shell.
- Removing legacy backend routes.
