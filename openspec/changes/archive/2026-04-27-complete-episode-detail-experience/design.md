## Context

The current catalog detail flow loads `GET /api/v1/items/{id}` for every media type and only loads `GET /api/v1/series/{id}/seasons` when the opened item is a series. Episode cards therefore navigate to `/media/{episodeId}` but the resulting page has no series/season context, no same-season shelf, and no episode-specific layout. The frontend presentation helper `buildPresentedCatalogItem()` currently returns the item unchanged, so the `view` route search value does not create a distinct episode presentation.

The backend already has most durable relationships needed for this work: catalog parent-child hierarchy, root IDs, season and episode index fields, TMDB descendant identities, metadata sources, selected still artwork, `asset_items`, `asset_files`, inventory files, and `media_streams`. The main gap is exposing those relationships through detail-oriented DTOs and mapping them into a consumer detail page rather than an admin/governance-oriented view.

## Goals / Non-Goals

**Goals:**

- Make episode detail pages match the design intent: series title, episode subtitle, still artwork, same-season shelf, episode overview, episode people, and detailed media stream information.
- Keep the existing `/media/$id` and `/play/$id` routes while making the presentation adapt to item type.
- Expose missing episode parent context, sibling shelf data, progress state, and stream metadata through typed catalog APIs.
- Make episode rematch/refetch behavior understandable and repairable when local assets are linked to the wrong season or episode.
- Preserve existing series detail shelves and movie detail behavior.

**Non-Goals:**

- Rebuild the whole detail page into a pixel-perfect Emby clone.
- Add a new metadata provider or replace TMDB matching.
- Implement casting, subscription/Premiere behavior, or external account integrations.
- Replace the player or implement full subtitle/audio switching persistence beyond passing available selection context to playback.
- Introduce new persistence unless an inspected stream attribute cannot be represented by existing `media_streams` or asset/file tables.

## Decisions

1. Extend catalog detail DTOs instead of adding a one-off episode aggregate endpoint.

   Episode detail, series detail, governance, and playback already converge on catalog item identities. Adding typed fields such as parent context, sibling season episodes, and asset media information to the existing catalog contracts keeps frontend loading centralized in `mibo-api.ts`. A new `/episode-detail` aggregate endpoint was considered, but it would duplicate item, hierarchy, progress, and asset loading before there is evidence of a performance bottleneck.

2. Represent episode parent context as a stable API structure.

   The frontend should not infer the series title from source paths or parse `SxxExx` labels. The backend should return explicit context for episode details: root series ID/title/images, season ID/name/number, episode number/range, and sibling season navigation data. This avoids fragile path parsing and supports provider-created missing/unaired descendants.

3. Keep same-season shelves local to the episode page but backed by catalog hierarchy reads.

   When an episode is opened, the page should fetch or receive the containing season's episode list, highlight the current episode, and include progress state when available. Routing to `?view=series` is not required for this interaction. The series page can continue using local selected-season state for full season browsing.

4. Expose media stream summaries through catalog asset detail, not playback URL resolution.

   The design's video/audio/subtitle cards are informational and do not require resolving a playable URL. Reading `media_streams` through catalog asset detail avoids triggering storage link checks just to render details. Playback can still use `/api/v1/items/{id}/playback` to select and validate the actual file.

5. Treat episode matching as a descendant-aware series hierarchy operation.

   TMDB TV metadata is synchronized from the series root, but the user action may originate on an episode. The action should report descendant-specific outcomes: missing identity, season/episode mismatch, provider descendant found, or asset linked to a different descendant. This keeps provider sync centralized while making the episode page actionable.

6. Keep governance corrections bounded to hierarchy and linkage state.

   Fixing an episode number, moving an asset link, or adjusting multi-episode segment links must not overwrite field locks, selected images, source evidence, or external identities. Corrections should use existing governance endpoints where possible and add narrowly scoped descendant repair operations only where the current API cannot express the change.

7. Prefer episode-level people, then fallback gracefully.

   Episode detail should display directors and guest/episode cast when provider data exists. If no episode-level people are available, the UI may fallback to series-level cast/directors with clear presentation, rather than showing empty person cards or misleading placeholder avatars.

## Risks / Trade-offs

- Provider-created missing episodes may have no local asset -> the page must show metadata and disable playback with a clear unavailable state.
- Stream metadata may be incomplete for unprobed files -> render partial technical cards and keep reprobe available instead of blocking detail load.
- Adding progress state to episode shelves can increase query cost -> batch progress lookup by sibling episode IDs and keep empty progress valid.
- Episode matching can still be ambiguous when local filenames are wrong -> surface the mismatch and offer governance repair rather than silently relinking assets.
- Fallback to series people can blur episode-specific credits -> visually prefer episode people and only fallback when the descendant has none.

## Migration Plan

- Add backend DTO fields as additive changes and keep old clients valid.
- Update catalog query tests to assert empty-list and omitted optional-field behavior.
- Update frontend API types and presentation mapping before changing UI rendering.
- Roll out episode detail layout behind item type checks so movie and series pages preserve current behavior.
- Rollback is straightforward: frontend can ignore new fields and backend can keep serving additive response fields.

## Open Questions

- Should audio/subtitle dropdown selections be persisted per user/item immediately, or only passed to the player for the current session?
- Should episode-level people include writers/guest stars as separate rails now, or only cast/directors to match the current people section shape?
- Should the same-season shelf be included directly in item detail responses or loaded through a companion query keyed by parent season ID?
