## 1. Backend Session Devices

- [x] 1.1 Extend `database.Session` with nullable login metadata fields for user agent, remote address, device name, and client type.
- [x] 1.2 Update login handling to capture request metadata and pass it into auth session creation without storing raw tokens or secrets.
- [x] 1.3 Add auth service methods to list current-user sessions with `is_current`, revoke one non-current session, and revoke all other sessions.
- [x] 1.4 Add HTTP routes and handlers for listing login sessions, revoking one session, and revoking all other sessions under `/api/v1/auth`.
- [x] 1.5 Add backend tests covering session listing, current-session protection, cross-user revocation blocking, and revoked token rejection.

## 2. Frontend Devices Page

- [x] 2.1 Add API client types and methods for login session list and revoke operations.
- [x] 2.2 Add React Query keys/options and mutations for login device management.
- [x] 2.3 Replace the existing `/settings/devices` console-summary device panel with a login session management panel.
- [x] 2.4 Show loading, empty, error, current-session, fallback-metadata, and mutation-pending states.
- [x] 2.5 Add confirmation flows for revoking a session and revoking all other sessions while disabling current-session revocation.
- [x] 2.6 Add or update the user menu entry to navigate to `/settings/devices` when session/device management is available.

## 3. Verification

- [x] 3.1 Run focused backend auth/httpapi tests for session device management.
- [x] 3.2 Run `go test ./...` from `mibo-media-server/`.
- [x] 3.3 Run `pnpm typecheck` from `web/`.
- [x] 3.4 Manually verify `http://localhost:3000/settings/devices` with `admin` / `admin123`, including revoking another session and preserving the current session.
