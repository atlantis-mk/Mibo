## 1. Data Model And Service Foundation

- [x] 1.1 Add a `user_settings` database model with a unique `user_id` constraint and include it in AutoMigrate.
- [x] 1.2 Define typed user settings request/response structs and default document construction in `internal/settings`.
- [x] 1.3 Implement service methods to read the current user's settings with defaults materialized when no record exists.
- [x] 1.4 Implement service methods to validate, normalize, and upsert the current user's settings document.

## 2. Authenticated API Surface

- [x] 2.1 Register `GET /api/v1/me/settings` and `PUT /api/v1/me/settings` in the authenticated user routes.
- [x] 2.2 Implement HTTP handlers that authenticate the user, decode requests, call the settings service, and return canonical JSON responses.
- [x] 2.3 Map validation failures to `400 Bad Request` and unauthenticated access to `401 Unauthorized` consistently with existing `/api/v1/me/*` handlers.

## 3. Verification

- [x] 3.1 Add service-level tests for default reads, normalization, validation failures, first-write upsert, and subsequent updates.
- [x] 3.2 Add HTTP handler tests for authenticated read/update success cases and unauthenticated rejection.
- [x] 3.3 Add persistence-isolation tests proving one user's saved settings are not returned to another user.
- [x] 3.4 Run the relevant backend test suite for settings and HTTP API coverage.
