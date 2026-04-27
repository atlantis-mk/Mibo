## 1. Data Contracts And Presentation Helpers

- [x] 1.1 Audit existing homepage, library, search, detail, progress, and latest-by-library DTOs for fields needed by library collages, continue-watching cards, year ranges, count badges, and quick actions.
- [x] 1.2 Replace loose `unknown[]` continue-watching client typing with a catalog-native TypeScript type that exposes the item and progress fields needed by poster cards.
- [x] 1.3 Add or update media presentation helpers for display title, poster/backdrop image, media type, year range/status, badge count priority, and quick-play eligibility.
- [x] 1.4 Add reusable poster-card and horizontal-rail components that support detail navigation, quick play/continue, green count badges, responsive sizing, and keyboard/focus accessibility.

## 2. Homepage Dashboard

- [x] 2.1 Extend homepage data loading to keep full library data, latest-by-library data, and typed continue-watching entries available to sections.
- [x] 2.2 Add a My Media section that renders each library as a clickable multi-poster collage card linked to `/library/$id`.
- [x] 2.3 Render Continue Watching as a horizontal poster rail when in-progress items exist.
- [x] 2.4 Upgrade Latest by Library sections to use shared poster cards, right-arrow section navigation, year ranges, and green count badges.
- [x] 2.5 Add stable loading, error, and empty states for no libraries, no media, empty continue-watching, and empty latest sections.
- [x] 2.6 Verify homepage desktop and mobile layout behavior for vertical page scrolling plus horizontal rail scrolling.

## 3. Favorites

- [x] 3.1 Confirm whether a durable user-scoped favorites model or endpoint already exists in the backend.
- [x] 3.2 If absent, add a user-scoped favorites table with `(user_id, item_id)` uniqueness and catalog item/user foreign keys.
- [x] 3.3 If absent, add authenticated `/api/v1/me/favorites` list, add, and remove handlers with focused backend tests for auth and user isolation.
- [x] 3.4 Add frontend API methods, query keys, and mutation helpers for listing, adding, and removing favorites.
- [x] 3.5 Add a favorites browsing route or surface reachable from the Home/Favorites switch and sidebar.
- [x] 3.6 Add favorite state and favorite/unfavorite actions on detail and applicable card surfaces.

## 4. Navigation Shell

- [x] 4.1 Replace `AppSidebar` documentation sample content with real Mibo navigation for Home, Favorites, Search, Libraries, Settings, and relevant management areas.
- [x] 4.2 Load or receive real library entries for sidebar library navigation without blocking the main page when optional sidebar data is unavailable.
- [x] 4.3 Update the homepage top bar with menu trigger, Mibo brand, Home/Favorites switch, search entry, cast entry, user menu, and settings entry.
- [x] 4.4 Implement the user menu with current username, settings navigation, and logout using existing auth/session behavior.
- [x] 4.5 Implement cast entry behavior as a clear unavailable/coming-soon dialog unless real cast support is available.

## 5. Backend Summary Support

- [x] 5.1 Determine whether existing `child_summary`, progress, and latest-by-library payloads can supply all poster badge counts without backend changes.
- [x] 5.2 If required, add catalog-native summary fields or a homepage aggregate endpoint for representative library posters, unwatched/update counts, and available child counts.
- [x] 5.3 Add focused backend tests for any new summary fields or homepage aggregate endpoint.
- [x] 5.4 Ensure all new backend routes stay on catalog, home, and `/api/v1/me/*` paths rather than retired legacy media routes.

## 6. Integration And Verification

- [x] 6.1 Run `pnpm typecheck` from `web/` and resolve introduced type errors.
- [x] 6.2 Run `pnpm build` from `web/` and resolve introduced build errors.
- [x] 6.3 Run focused backend tests for changed backend packages, or `go test ./...` from `mibo-media-server/` if backend changes are broad.
- [x] 6.4 Manually verify logged-out redirect, populated homepage, no-library homepage, favorites empty/populated states, detail navigation, quick play/continue, and user logout.
- [x] 6.5 Manually verify desktop and mobile viewports for top navigation, sidebar, My Media cards, Continue Watching rail, Latest by Library rails, and cast unavailable messaging.
