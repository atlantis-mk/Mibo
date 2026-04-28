## 1. Backend Summary Data

- [x] 1.1 Audit existing backend endpoints and services for server info, library counts, jobs, schedules, catalog consistency, and playback/progress activity that can feed the console.
- [x] 1.2 Add an authenticated dashboard summary handler only for data that cannot be composed cleanly from existing endpoints.
- [x] 1.3 Include server, access, storage, database, media metrics, health, device, quick-action, and activity sections in the summary response with typed response structs.
- [x] 1.4 Return partial warning states for unavailable sections instead of failing the whole summary when non-critical data is missing.
- [x] 1.5 Wire any new backend route under `/api/v1` without using retired legacy media read routes.
- [x] 1.6 Add focused backend tests for the dashboard summary success path, partial-unavailable states, and authentication behavior.

## 2. Frontend API And Queries

- [x] 2.1 Add TypeScript types for the console summary, server status, access addresses, metrics, activity events, device summaries, and quick actions.
- [x] 2.2 Add API client functions and query options for loading console summary data.
- [x] 2.3 Add mutation helpers for supported expensive quick actions such as library scans, catalog consistency checks, or projection rebuilds where existing APIs are available.
- [x] 2.4 Represent unsupported actions and routes with disabled metadata so the UI does not navigate to broken screens.

## 3. Console Route And Navigation

- [x] 3.1 Add the authenticated `/console` route under the existing `_app` route group.
- [x] 3.2 Extend the app sidebar with a highlighted `控制台` entry and grouped management sections for Mibo Web, server, media, devices, and advanced operations.
- [x] 3.3 Preserve existing consumer navigation and library shortcuts while adding admin entries.
- [x] 3.4 Ensure unavailable sidebar entries render disabled or coming-soon states instead of active links.

## 4. Dashboard UI

- [x] 4.1 Build the console page shell with title area, back/navigation affordance, and top-right quick entries for cast/play-to-device, current user, and settings.
- [x] 4.2 Build the server overview card with service status, version/update status, port, uptime, storage, database, module health, and access addresses.
- [x] 4.3 Build metric cards for libraries, media sources, item/file/person/series/episode counts, devices, jobs/scans, and warning/error counts.
- [x] 4.4 Build quick-action controls with confirmation for expensive operations and disabled states for unsupported operations.
- [x] 4.5 Build the activity timeline with icons, severity styling, event descriptions, optional user/device/media fields, timestamps, loading state, and empty state.
- [x] 4.6 Build the management entry grid for users, media library, live TV, network, transcoding, database, conversions, scheduled tasks, logs, plugins, devices, downloads, camera upload, DLNA, and advanced maintenance.
- [x] 4.7 Build the device-related section with connected/recent device summaries where data exists and disabled planned-feature entries where it does not.

## 5. Responsive And Visual Polish

- [x] 5.1 Apply light admin styling with white/gray surfaces, green primary emphasis, yellow warning, red danger, gray unavailable states, and compact information density.
- [x] 5.2 Verify desktop layout keeps the sidebar persistently available and uses dashboard grids effectively.
- [x] 5.3 Verify mobile layout stacks cards, keeps actions tappable, and preserves access to core status and activity sections.
- [x] 5.4 Add loading, full-error, retry, and partial-warning UI states for console data.

## 6. Verification

- [x] 6.1 Run backend focused tests for any new handlers or services.
- [x] 6.2 Run `pnpm typecheck` from `web/`.
- [x] 6.3 Manually verify `/console` after login with the local app credentials and sample/local storage configuration.
- [x] 6.4 Verify unsupported management entries and quick actions cannot trigger broken navigation or unsafe operations.
