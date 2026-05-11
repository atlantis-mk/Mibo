## 1. Backend Homepage Feed

- [x] 1.1 Add catalog service types for homepage semantic sections, including stable section key, display title, and `CatalogListItem` entries.
- [x] 1.2 Implement a catalog query that groups available, non-hidden movie and series projections into homepage sections ordered for display.
- [x] 1.3 Add an authenticated HTTP route for homepage sections or an equivalent homepage feed endpoint.
- [x] 1.4 Normalize artwork URLs for section items in the HTTP handler.
- [x] 1.5 Add backend tests for movie section, series section, omitted empty sections, hidden/unavailable exclusions, and unauthenticated access.

## 2. Frontend API And Query Chain

- [x] 2.1 Add frontend API types and client method for homepage semantic sections.
- [x] 2.2 Change `homeDataQueryOptions` to stop calling `listLibraries()` and `latestByLibrary()` for homepage rendering.
- [x] 2.3 Change home dashboard data/state from `libraries`, `libraryCount`, and `latestByLibrary` to content sections and derived content counts.
- [x] 2.4 Ensure active ingest polling no longer depends on library status in the homepage query; use an appropriate remaining signal or remove the polling behavior if no reliable home-specific signal exists.

## 3. Frontend Rendering

- [x] 3.1 Replace `LatestLibraryRail` usage with content-section rail rendering.
- [x] 3.2 Update homepage labels and empty-state copy so media libraries are not presented as the primary homepage model.
- [x] 3.3 Preserve existing card navigation, playback, detail links, Continue Watching, hero carousel, and degraded health indicator behavior.
- [x] 3.4 Verify responsive desktop and mobile layout for content-section rails.

## 4. Delete Or Migrate Old Request Chain

- [x] 4.1 Find all remaining frontend usages of `latestByLibrary()` and decide whether each should migrate to homepage sections or a non-home diagnostic/governance path.
- [x] 4.2 Remove the frontend `latestByLibrary()` client method when no product consumer remains.
- [x] 4.3 Remove or rename the backend `/api/v1/home/latest-by-library` route/handler after consumers are migrated.
- [x] 4.4 Remove obsolete latest-by-library homepage state/types/tests.

## 5. Verification

- [x] 5.1 Update frontend home state/regression tests for populated, empty, degraded, continue-watching, and content-section scenarios.
- [x] 5.2 Run `pnpm typecheck` from `web/`.
- [x] 5.3 Run focused backend HTTP/catalog tests for homepage feed behavior from `mibo-media-server/`.
- [x] 5.4 Run `go test ./...` from `mibo-media-server/` if focused tests pass.
