## 1. Backend User Management API

- [x] 1.1 Add an admin authorization helper in `internal/httpapi` that requires a valid session and `role == "admin"`.
- [x] 1.2 Add an auth service method for admin-created users that validates username, password, and role without changing first-user registration behavior.
- [x] 1.3 Add `GET /api/v1/admin/users` to return users without password hashes or session data.
- [x] 1.4 Add `POST /api/v1/admin/users` to create users with role `user` or `admin`.
- [x] 1.5 Map duplicate username and invalid role failures to clear client errors.

## 2. Backend Tests

- [x] 2.1 Add HTTP tests proving anonymous and non-admin users cannot list or create users.
- [x] 2.2 Add HTTP tests proving admins can list users and create ordinary users.
- [x] 2.3 Add HTTP tests for admin creation of another admin, duplicate username rejection, and invalid role rejection.
- [x] 2.4 Run focused backend tests for the affected auth/httpapi packages.

## 3. Frontend API Integration

- [x] 3.1 Add typed `AdminUser` or reuse-safe user response types in `web/src/lib/mibo-api.ts`.
- [x] 3.2 Add `listAdminUsers` and `createAdminUser` API client methods under the existing request boundary.
- [x] 3.3 Add React Query keys/options or local query usage for admin user listing and mutation invalidation.

## 4. Settings UI

- [x] 4.1 Replace the local session-only placeholder list in `user-management-panel.tsx` with server-backed user data.
- [x] 4.2 Enable the "新增用户" action for administrators and add a create-user form with username, password, and role.
- [x] 4.3 Show loading, empty, success, and validation/error states without hiding the existing user detail layout.
- [x] 4.4 Refresh the user list and select the created user after successful creation.

## 5. Verification

- [x] 5.1 Run `go test ./internal/httpapi ./internal/auth` from `mibo-media-server/`.
- [x] 5.2 Run `pnpm typecheck` from `web/`.
- [x] 5.3 Manually verify an admin can open `/settings/users`, create a user, and see it in the list.
