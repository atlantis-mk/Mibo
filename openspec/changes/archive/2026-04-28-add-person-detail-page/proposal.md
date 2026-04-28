## Why

Mibo currently exposes cast and director names inside media detail pages, but users cannot open a dedicated person detail experience to understand who someone is or continue browsing their related works. Adding a person-first detail page closes that discovery gap and makes cast metadata materially more useful instead of being a static label list.

## What Changes

- Add a dedicated person detail route and page in the web app with an immersive hero area, biography and basic facts, related works shelves, and external database links.
- Make cast and director cards in catalog media detail pages navigable so users can move from a title to a person detail page.
- Add a catalog person detail API that returns person identity, portrait, biography, birth facts when known, external IDs, and related catalog items grouped for browsing.
- Expand person metadata persistence so catalog people can store richer profile fields sourced from metadata providers instead of only name and avatar.
- Use representative related artwork as the visual backdrop when available, while preserving graceful fallbacks for sparse metadata.

## Capabilities

### New Capabilities
- `person-detail-experience`: Users can open a person detail page that highlights the person, explains who they are, and links into related catalog titles.

### Modified Capabilities

## Impact

- Frontend routing and screens: add a new person detail route, data loader, and page sections in `web/src/routes` and `web/src/features`.
- Existing media detail UI: update person cards in `web/src/features/media/components/standalone-media-detail-specs.tsx` to navigate to the new page.
- Frontend API/query layer: extend `web/src/lib/mibo-api.ts` and `web/src/lib/mibo-query.ts` with person detail contracts and queries.
- Backend catalog API: add `/api/v1/people/{id}` style read endpoints and related contract mapping in `mibo-media-server/internal/httpapi` and `internal/catalog`.
- Backend metadata/data model: extend person storage beyond `name` and `avatar_url` so biography, birth facts, and provider IDs can be persisted and served.
- Tests: add backend contract/query coverage and frontend route/type coverage for person detail browsing.
