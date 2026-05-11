## Why

The authenticated homepage still treats media libraries as the primary discovery axis even though Mibo is moving toward source-first scanning and content-shape-driven catalog organization. This makes the product model inconsistent: users see library buckets on the main page while the scanner and catalog increasingly classify content by what it is rather than where it came from.

## What Changes

- Replace the homepage's latest-by-library feed with homepage sections grouped by user-facing content shape, such as movies, series, and other supported catalog shapes.
- Stop the homepage data query from requesting library lists and `/api/v1/home/latest-by-library` for normal rendering.
- Add a backend homepage feed contract that returns ready-to-render semantic sections rather than library-scoped sections.
- Keep media libraries available for settings, source management, diagnostics, and explicit library detail views, but remove them as the homepage's primary discovery structure.
- Preserve Continue Watching and recently-added hero behavior while deriving homepage summary counts from content sections or recently-added catalog items instead of library records.
- **BREAKING**: The homepage frontend will no longer depend on the `latest_by_library` response shape for rendering; any internal consumers that still need library-grouped recents must call a non-home diagnostics/governance path or be migrated.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `homepage-media-library-dashboard`: Replace media-library-first homepage requirements with content-shape homepage sections and remove the homepage request chain that fetches libraries and latest-by-library data.

## Impact

- Frontend homepage state, query options, section rendering, empty-state copy, and tests under `web/src/features/home/` and `web/src/lib/mibo-query.ts`.
- Frontend API client types and methods in `web/src/lib/mibo-api.ts`.
- Backend home feed handlers/routes in `mibo-media-server/internal/httpapi/` and catalog query logic in `mibo-media-server/internal/catalog/`.
- OpenSpec requirement contract for the authenticated homepage.
