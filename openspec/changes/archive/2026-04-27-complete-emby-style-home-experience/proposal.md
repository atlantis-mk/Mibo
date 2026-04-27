## Why

Mibo already has a dark homepage with recently-added hero content, media detail/playback routes, search, settings, and library pages, but it still lacks the complete media-center dashboard behavior users expect from an Emby-style library homepage. Closing these gaps now makes the homepage the primary browsing surface for library entry, recent updates, continuing playback, favorites, search, user actions, and settings.

## What Changes

- Add an Emby-style homepage structure with top navigation, a My Media library entrance wall, Continue Watching, Latest by Library, and Favorites entry points.
- Replace placeholder sidebar content with real Mibo media-center navigation for Home, Favorites, Search, Libraries, Settings, and relevant management areas.
- Add durable favorites browsing and favorite/unfavorite actions if no existing user-scoped favorite model is available.
- Add reusable poster-card presentation for title, poster, year range/status, green count badges, quick play/continue actions, and detail navigation.
- Add homepage media-library cards that use multi-poster collage visuals and route into each library.
- Add cast, search, user, and settings entry points to the app shell; cast may be a clear unsupported/coming-soon action unless playback casting is implemented.
- Keep new product flows catalog-native and avoid building on retired legacy media read routes.

## Capabilities

### New Capabilities

- `homepage-media-library-dashboard`: Defines the homepage dashboard sections for My Media library entries, Continue Watching, and Latest by Library rails.
- `favorites-browsing`: Defines user-scoped favorite persistence, favorite actions, and favorite browsing surfaces.
- `media-card-progress-badges`: Defines poster-card metadata including year ranges, count badges, progress-aware labels, and quick actions.
- `app-navigation-shell`: Defines the real Mibo top-bar and sidebar navigation, including search, user, settings, and cast entry behavior.

### Modified Capabilities

- None.

## Impact

- Frontend: `web/src/features/home`, `web/src/components/app-top-bar.tsx`, `web/src/components/app-sidebar.tsx`, `web/src/components/search-form.tsx`, `web/src/features/library`, `web/src/features/search`, `web/src/lib/media-presentation.ts`, `web/src/lib/mibo-query.ts`, and `web/src/lib/mibo-api.ts`.
- Backend, if needed for durable favorites or count summaries: `mibo-media-server/internal/httpapi`, `mibo-media-server/internal/catalog`, `mibo-media-server/internal/progress`, and `mibo-media-server/internal/database`.
- APIs: may add user-scoped favorites routes under `/api/v1/me/*` and optional catalog-native homepage/card summary fields.
- Verification: frontend `pnpm typecheck` and `pnpm build`; focused backend tests for any new favorites or summary endpoints.
