## Why

The settings navigation already exposes `/settings/devices`, but users do not yet have a complete page for reviewing and managing login devices. Completing this flow improves account security by making active sessions visible and revocable from the web UI.

## What Changes

- Add a login device management experience at `http://localhost:3000/settings/devices` for authenticated users.
- Show the current user's login sessions with device/client metadata, last activity, creation time, expiration, and current-device indication.
- Allow users to revoke individual non-current sessions and revoke all other sessions from the page.
- Add backend session/device read and revoke endpoints backed by the existing auth session store.
- Preserve existing logout behavior for the current session.

## Capabilities

### New Capabilities
- `login-device-management`: Covers listing and managing authenticated login devices/sessions from the settings devices page.

### Modified Capabilities
- `app-navigation-shell`: Clarify that session-relevant user actions can include navigation to login device management.

## Impact

- Frontend: `web/src/features/settings`, route handling for `/settings/devices`, API client types/methods, query keys, and settings page UI.
- Backend: auth service session queries/revocation, HTTP routes under `/api/v1/auth`, request client metadata capture during login, and session serialization.
- Database: may require additive nullable session metadata fields for user agent, remote address, device name/client type, and timestamps already not present.
- Tests: backend auth/httpapi tests and focused frontend typecheck/build coverage for the new settings device page.
