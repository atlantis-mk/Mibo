## Why

The network settings page at `/settings/network` currently behaves as a browser-local draft, so administrators cannot reliably configure server access behavior across sessions, browsers, or deployments. Completing this feature makes network configuration an authenticated server setting that can be validated, saved, reloaded, and prepared for runtime integration.

## What Changes

- Add a server-backed network settings model covering local network detection, public access, reverse proxy trust, TLS fields, automatic port mapping preferences, and streaming limits.
- Add authenticated API endpoints for reading and updating network settings.
- Replace the frontend localStorage-only save flow with API-backed loading, validation feedback, save states, and reset-to-current behavior.
- Preserve the existing `/settings/network` route and settings navigation entry while making the page production-meaningful.
- Surface clear guidance for settings that require restart or future runtime wiring, instead of implying that local browser saves change the server immediately.

## Capabilities

### New Capabilities
- `network-settings`: Administrator network configuration for local/remote access, proxy/TLS preferences, port mapping, and remote streaming limits.

### Modified Capabilities

## Impact

- Frontend: `web/src/routes/settings.network.tsx`, `web/src/features/settings/components/network-settings-panel.tsx`, `web/src/lib/mibo-api.ts`, and `web/src/lib/mibo-query.ts`.
- Backend: `mibo-media-server/internal/httpapi/router.go`, settings persistence/service code, and database migration or configuration storage for network settings.
- APIs: new authenticated `/api/v1/settings/network` read/update endpoints.
- Tests: backend API/service tests and frontend typecheck coverage for the network settings client contract.
