## Why

Mibo needs a dedicated admin console that gives operators a fast, information-dense view of server health, access addresses, media library status, recent activity, and management entry points. Today those operational concerns are scattered across product screens or backend endpoints, which makes routine maintenance and troubleshooting slower than it needs to be.

## What Changes

- Add an admin console dashboard page with a fixed management sidebar, top page actions, server status summary, activity timeline, and management shortcut sections.
- Show server overview information including server name, version, update status, API port, storage provider, database status, uptime, and media counts.
- Show access addresses for local, LAN, and remote access, with clear unavailable or unconfigured states.
- Provide quick actions for common operations such as scanning the media library, viewing logs, running catalog consistency checks, rebuilding projections, and opening settings.
- Show recent activity such as playback starts and stops, user/device activity, scan events, transcode events, setup or source changes, and operational warnings.
- Provide grouped navigation and management entries for users, media libraries, metadata, network, transcoding, database, scheduled tasks, logs, plugins, connected devices, downloads, camera uploads, DLNA, and advanced maintenance.
- Add responsive behavior so the console remains usable on desktop and mobile while preserving the established Mibo visual language.

## Capabilities

### New Capabilities

- `admin-console-dashboard`: Defines the Mibo admin console experience, including layout, server status, access addresses, activity timeline, quick actions, management entry points, and device-related entries.

### Modified Capabilities

None.

## Impact

- Frontend: likely affects `web/src/router.tsx`, `web/src/App.tsx` or extracted dashboard components, shared navigation/sidebar components, API client types, and UI composition under `web/src/components/`.
- Backend: may require lightweight read-only summary endpoints if existing APIs do not already expose version, server status, address, activity, and count information in a dashboard-friendly shape.
- APIs: may add dashboard summary and activity endpoints under the current `/api/v1` surface without changing legacy media routes.
- Operations: surfaces existing maintenance endpoints such as catalog consistency and projection rebuilds as console actions instead of hiding them behind manual API calls.
