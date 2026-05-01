## Why

The settings user management page currently presents user administration as a placeholder: it only shows the current session user and disables the "新增用户" action because there is no admin-facing user management API. This leaves administrators unable to create accounts for family members or client devices after initial setup.

## What Changes

- Add an authenticated, admin-only user management capability for listing users and creating new users.
- Replace the settings page placeholder data with server-backed users.
- Enable the "新增用户" action with a form for username, password, and role.
- Keep account creation scoped to basic active users; password reset, deletion, disabling, and per-library access control remain future work.
- Preserve existing setup registration behavior where the first registered account becomes an administrator.

## Capabilities

### New Capabilities
- `admin-user-management`: Admin-facing user listing and account creation for the settings user management page.

### Modified Capabilities

## Impact

- Backend API: new admin user routes under `/api/v1/admin/users`.
- Backend auth: admin authorization checks for user management endpoints.
- Frontend API client: typed user list and create-user methods in `web/src/lib/mibo-api.ts`.
- Frontend UI: `web/src/features/settings/components/user-management-panel.tsx` becomes server-backed and enables account creation.
- Tests: backend HTTP/auth coverage for admin-only access and frontend type/build verification.
