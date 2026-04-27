## Why

The current media detail page has the base immersive layout, but several Emby-style details are either missing, non-interactive, or represented by technical/governance data instead of user-facing media metadata. Closing these gaps will make the detail page feel like a complete playback surface rather than an admin-oriented catalog view.

## What Changes

- Add richer hero metadata for rating, year ranges, official rating, genres, series status, and season/episode summaries where data exists.
- Replace placeholder-looking hero actions with usable watched, favorite, play, and more-menu behavior, while moving management actions out of the primary consumer action row.
- Add an Emby-style season selector that focuses the episode shelf on one season at a time and separates Specials from numbered seasons.
- Improve episode cards with `Sx:Ey` labeling and user-facing availability/progress signals.
- Add related/recommended media shelves using poster cards and existing badge/year-range behavior.
- Rework the bottom information area so user-readable metadata and external database links are primary, while technical asset/governance details are secondary.
- Ensure detail-page top navigation entries are actual app entries or clearly marked unavailable actions instead of decorative icons.

## Capabilities

### New Capabilities
- `immersive-media-detail`: Defines the media detail page presentation, actions, season/specials shelves, related shelves, and user-facing metadata hierarchy.

### Modified Capabilities
- `catalog-api-playback`: Item detail responses must expose the catalog-backed metadata needed by the immersive detail page, including display ratings, ratings certificates, year ranges, tags/genres, external identities, child summaries, and related media candidates.
- `app-navigation-shell`: The detail-page top bar must use real navigation/menu/cast/search/user/settings semantics consistent with the app shell.

## Impact

- Frontend: `web/src/features/media/*`, `web/src/lib/media-presentation.ts`, `web/src/lib/mibo-api.ts`, and reusable media card/top-bar UI as needed.
- Backend: catalog detail DTOs and queries under `mibo-media-server/internal/catalog` and HTTP API responses if required to expose missing metadata.
- Data: uses existing catalog fields where possible; may require projecting existing tags/external identities into detail responses rather than adding new persisted tables.
- Tests: frontend typecheck/build and focused backend catalog contract/query tests for any DTO changes.
