## Why

Series detail pages currently behave like the series item itself must own a playable asset, so the primary play action can remain unavailable even when local episode files exist. Series season and episode presentation can also include provider-known missing or unaired descendants in the main episode shelf, which makes the page look playable or complete for episodes that do not have local media.

## What Changes

- Make series primary playback resolve to a local playable episode: continue the user's in-progress episode when one exists, otherwise start from the first locally available episode.
- Keep series detail season and episode shelves focused on locally playable episodes by default, while preserving missing and unaired descendants for dedicated missing/operational views.
- Ensure episode counts and season labels shown in the consumer detail page reflect displayed local episodes rather than all provider-known descendants.
- Keep unavailable episode detail behavior intact when a user explicitly opens a missing or unaired episode from governance or missing-episode workflows.

## Capabilities

### New Capabilities

### Modified Capabilities
- `catalog-api-playback`: Series detail and playback APIs must expose a progress-aware playable episode target for series items and local-only episode hierarchy presentation for consumer shelves.
- `immersive-media-detail`: Series detail primary actions and episode shelves must use the playable episode target and hide non-local episodes from the default consumer shelf.
- `tv-hierarchy-metadata-completion`: Provider-known missing and unaired descendants remain durable catalog state, but consumer hierarchy reads must distinguish local-playable presentation from complete operational hierarchy state.

## Impact

- Backend catalog reads in `mibo-media-server/internal/catalog/query.go` and `query_tv.go` need availability-aware hierarchy behavior and a reusable series playback target selection path.
- Playback routing in `mibo-media-server/internal/httpapi/handlers_catalog.go` and `internal/playback` may need to accept a series item by resolving it to the chosen episode item before asset selection.
- Frontend detail presentation in `web/src/features/media/index.tsx`, `web/src/lib/media-presentation.ts`, and `web/src/features/media/components/standalone-media-detail-*` needs to render series play/continue against the selected episode and filter default shelves to local episodes.
- Tests should cover series with mixed available/missing/unaired episodes, in-progress episode continuation, first-local-episode fallback, and preservation of explicit missing/unaired views.
