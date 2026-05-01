## Context

`/settings/devices` already exists in the frontend settings navigation and currently renders a device panel based on admin console activity summaries. That data is useful for operational telemetry but does not represent authenticated login sessions, so it cannot answer which devices are signed in or support revoking device access.

The backend auth service already stores sessions with token hashes, user IDs, expiration, creation, update, and last-used timestamps. The session model does not yet store request/device metadata, and the existing login API does not capture client information. The implementation should build on this auth session store rather than creating a separate device registry.

## Goals / Non-Goals

**Goals:**
- Provide a complete authenticated settings page at `/settings/devices` for viewing login sessions.
- Let users revoke individual non-current sessions and revoke all other sessions for their account.
- Mark the current session clearly and prevent accidental self-revocation from device-management actions.
- Capture basic login/session metadata from incoming requests without storing raw tokens or sensitive secrets.
- Keep the API scoped to the current authenticated user.

**Non-Goals:**
- Real-time presence, push updates, or online/offline detection.
- Remote logout of other users by admins.
- Full user-agent parsing with a new external dependency.
- Device trust, MFA, passkeys, or long-term device registration.
- DLNA/cast device management.

## Decisions

1. Use sessions as login devices.

   A login device is represented by an auth session row plus display metadata. This avoids introducing a separate device table before there is a durable device identity model. Alternative considered: create a `devices` table and link sessions to it. That would support long-term per-device history, but it adds identity and deduplication complexity that is unnecessary for session revocation.

2. Add nullable metadata to `database.Session`.

   Store additive fields such as `UserAgent`, `RemoteAddr`, `DeviceName`, and `ClientType` on sessions. Existing sessions can render with fallback labels like `Unknown device` or `Mibo Web`, and GORM auto-migration can add nullable columns without requiring a backfill. Alternative considered: derive all metadata on read from access logs or activity events. That would be incomplete and would not cover quiet active sessions.

3. Keep endpoints under current-user auth scope.

   Add endpoints such as `GET /api/v1/auth/sessions`, `DELETE /api/v1/auth/sessions/{id}`, and `DELETE /api/v1/auth/sessions/others`. Handlers require the bearer token, resolve the current user, and only operate on that user's sessions. Alternative considered: placing routes under `/api/v1/settings/devices`; auth-session semantics are clearer under `/auth` and align with logout.

4. Identify current session by token hash server-side.

   The list response should include `is_current` computed by comparing the request bearer token hash with each session's stored token hash. The raw token hash must not be serialized. Deleting a specific session should reject the current session with a validation error and direct callers to use the existing logout endpoint for current-session logout.

5. Replace the settings devices panel data source.

   The frontend should replace the console summary device list with auth session queries and mutations. This keeps `/settings/devices` focused on login device security while leaving admin console device summaries available in the console page.

## Risks / Trade-offs

- [Risk] User-agent-derived device names may be imprecise -> Mitigation: store raw user agent for audit display and use simple deterministic labels without promising exact device detection.
- [Risk] Existing sessions lack metadata -> Mitigation: render fallback values and rely on new logins to collect richer metadata.
- [Risk] Revoking all other sessions can surprise users -> Mitigation: require an explicit confirmation in the UI and keep the current session active.
- [Risk] Stale pages after revocation can show old rows -> Mitigation: invalidate the session query after successful mutations and show mutation progress/error states.

## Migration Plan

Add nullable columns to the existing `sessions` table through the current database migration/auto-migration path. Deploying the backend first is safe because old clients continue to use login/logout. Rollback is safe because old code ignores extra database columns; rows created with metadata still contain the existing required session fields.

## Open Questions

- Should device metadata prefer `X-Forwarded-For` when the trusted proxy/network settings are finalized, or only `RemoteAddr` for this change?
