## Context

The backend already has an auth service that can register users and assigns the first user the `admin` role. The settings user management UI does not use this flow because it needs an authenticated administration surface: it currently renders only the current session user and disables account creation.

This change spans backend authorization, API shape, frontend API bindings, and the settings UI. It must preserve setup registration behavior while adding a separate admin-only management path.

## Goals / Non-Goals

**Goals:**
- Provide admin-only APIs to list users and create users.
- Enable the settings user management page to display real users from the database.
- Enable administrators to create basic accounts with username, password, and role.
- Keep validation consistent with existing auth registration rules.
- Add focused backend tests for authorization and creation behavior.

**Non-Goals:**
- No password reset, user deletion, account disabling, session revocation, or audit log UI.
- No per-library or per-media-source access control.
- No public self-service registration changes.
- No migration of existing users beyond using current `users` rows.

## Decisions

1. Add admin user endpoints under `/api/v1/admin/users`.

   Rationale: Existing admin console and log routes already use the `/api/v1/admin/*` namespace. Keeping user management there makes authorization expectations clear and avoids overloading `/api/v1/auth/register`.

   Alternative considered: Reuse `POST /api/v1/auth/register` from the settings page. This was rejected because registration is a setup/public auth concern and does not enforce current-admin authorization.

2. Implement role checks in the HTTP layer with a small reusable admin guard.

   Rationale: User management authorization is route-level policy. Handlers can require a valid session and then reject non-admin users before calling auth/database logic.

   Alternative considered: Add role checks inside `auth.Service.Register`. This was rejected because setup registration still needs to create the first admin without a current user, and mixing caller policy into the service would make the existing flow harder to preserve.

3. Add a dedicated create-user method in `auth.Service` for admin creation only if needed by implementation clarity.

   Rationale: Existing `Register` always computes role from user count and cannot accept an explicit role. Admin creation needs to create either `user` or `admin` after validating allowed roles. This can be a new service method such as `CreateUser(ctx, username, password, role)` while leaving `Register` untouched.

   Alternative considered: Change `Register` to accept a role. This was rejected because it would alter the semantics of setup registration and broaden a public auth primitive.

4. Use React Query from the settings panel for server state and keep API calls in `mibo-api.ts`.

   Rationale: The repo already centralizes frontend requests through `web/src/lib/mibo-api.ts` and uses query helpers for shared server state. The page should invalidate the users query after creation instead of maintaining a local-only list.

   Alternative considered: Call `fetch` directly in the component. This was rejected because it bypasses the project API boundary.

## Risks / Trade-offs

- [Risk] Accidentally exposing user creation to non-admin sessions → Mitigation: add backend tests for missing token, non-admin user, and admin user cases.
- [Risk] Duplicate username database errors surface as low-quality messages → Mitigation: normalize/validate input and map uniqueness failures to a clear bad-request response where practical.
- [Risk] The first-user setup path changes unintentionally → Mitigation: keep `/api/v1/auth/register` behavior unchanged and test admin creation through separate endpoints.
- [Risk] Creating additional admins is a powerful action → Mitigation: require explicit `role` input constrained to `user` or `admin`, and default frontend selection to ordinary user.

## Migration Plan

No database migration is required because the existing `users` table already contains username, password hash, role, and timestamps.

Deploy backend and frontend together. Existing sessions continue to work. If rollback is needed, the disabled placeholder UI can be restored while created users remain valid ordinary `users` table records.
