## 1. Backend Catalog Contract

- [x] 1.1 Add an optional series playback target DTO to catalog item detail responses, including episode item ID, optional asset ID, label/title, and selection reason.
- [x] 1.2 Implement a shared series playback target selector that considers only locally playable episodes and prefers unfinished user progress before first available episode ordering.
- [x] 1.3 Populate the series playback target from `GetItemDetailForUser` for series items without changing movie or episode detail responses.
- [x] 1.4 Scope the default consumer series season hierarchy to locally playable episodes and omit seasons with no local playable episodes.
- [x] 1.5 Preserve explicit missing/unaired series reads through missing-episode and availability-filtered operational paths.

## 2. Backend Playback

- [x] 2.1 Resolve catalog playback requests for series items through the shared series playback target selector before asset selection.
- [x] 2.2 Return playback source context for the resolved episode item when a series has a playable target.
- [x] 2.3 Return a clear unplayable decision, not a server error, when a series has no locally playable episode target.

## 3. Frontend Detail Experience

- [x] 3.1 Update frontend API types and media presentation mapping for the optional series playback target.
- [x] 3.2 Route the series detail primary play/continue action to the target episode item and selected asset when present.
- [x] 3.3 Show disabled or unavailable feedback for series with no local playback target instead of navigating to the series item player.
- [x] 3.4 Ensure the default `剧集信息` shelf and displayed counts use only local playable episodes and omit empty seasons.
- [x] 3.5 Keep direct missing or unaired episode detail pages visibly unavailable without changing governance actions.

## 4. Verification

- [x] 4.1 Add backend tests for series playback target selection with unfinished progress, first-local fallback, and no-local target.
- [x] 4.2 Add backend tests for series playback endpoint resolution and unplayable response behavior.
- [x] 4.3 Add backend tests that default series season hierarchy hides missing/unaired episodes while explicit missing reads still return them.
- [x] 4.4 Run `go test ./internal/catalog ./internal/playback ./internal/httpapi` from `mibo-media-server/`.
- [x] 4.5 Run `pnpm typecheck` from `web/`.
