## Context

`mibo-media-server` already exposes authenticated `/api/v1/me/*` endpoints for user-scoped data such as profile, progress, favorites, and sessions, while server-wide configuration lives under the `settings` service and `system_settings` persistence. There is no equivalent backend capability for per-user preferences, so frontend clients currently have no supported API for account-level settings and would otherwise need to keep important preferences in local-only state.

This change is backend-only. It must create a stable contract for frontend integration without requiring updates in `frontend/`. The design should fit the existing Go service structure, GORM models, and auth flow, and it should be safe for existing users who have never saved settings before.

## Goals / Non-Goals

**Goals:**
- Add an authenticated backend API for reading the current user's settings.
- Add an authenticated backend API for updating the current user's settings.
- Persist settings per user with clear defaults for users who have no saved record yet.
- Define a minimal but useful initial settings schema for frontend integration.
- Keep user-scoped settings clearly separated from admin/server-wide settings.

**Non-Goals:**
- Do not implement `frontend/` UI or client-side consumption in this change.
- Do not migrate existing browser `localStorage` preferences automatically.
- Do not make every stored preference immediately affect runtime playback or UI behavior on the server.
- Do not introduce admin APIs for editing another user's settings in this change.

## Decisions

### 1. Use a dedicated `user_settings` table instead of reusing `system_settings`

The system already uses `system_settings` for server-wide key/value configuration keyed by category and key. Reusing that table for per-user data would require synthetic keys, would blur ownership boundaries, and would make future evolution awkward. A dedicated `user_settings` table keyed by `user_id` keeps the distinction explicit and supports independent validation and migration rules.

The table should store a canonical JSON document plus timestamps, with a unique constraint on `user_id`. This keeps the initial implementation simple while allowing the schema to grow without frequent table migrations.

Alternatives considered:
- Reuse `system_settings` with `user:<id>` prefixes: rejected because it mixes scopes and complicates querying and validation.
- Add many typed columns immediately: rejected because the initial frontend-facing preference set is still likely to evolve.

### 2. Expose the capability at `/api/v1/me/settings`

User settings are part of the authenticated current-user surface, so the API should live beside `/api/v1/me`, `/api/v1/me/favorites`, and other user-specific endpoints. The initial contract should provide:

- `GET /api/v1/me/settings`
- `PUT /api/v1/me/settings`

`GET` returns the fully materialized settings document with defaults applied. `PUT` accepts a settings payload, validates and normalizes it, persists it for the authenticated user, and returns the canonical saved document.

Alternatives considered:
- `/api/v1/settings/user`: rejected because `settings/*` is currently admin/server-oriented.
- `PATCH /api/v1/me/settings`: rejected for the initial version to avoid merge ambiguity and reduce handler complexity.

### 3. Start with a small versioned settings schema oriented around frontend preferences

The initial schema should be intentionally small, frontend-meaningful, and cheap to validate. A recommended first version is:

- `appearance.theme`: `system | light | dark`
- `appearance.locale`: BCP-47 style locale string or empty/default
- `playback.autoplay_next_episode`: boolean
- `playback.prefer_direct_play`: boolean
- `playback.default_subtitle_mode`: `auto | always | never`
- `playback.preferred_audio_language`: language code string or empty
- `playback.preferred_subtitle_language`: language code string or empty

The response should always include all supported fields, even when the user has never customized them, so frontend clients can treat the API as a complete source of truth.

Alternatives considered:
- A free-form settings map with no typed structure: rejected because it weakens validation and makes frontend contracts fragile.
- A much larger schema including every possible preference area: rejected because it increases implementation cost without clear current product need.

### 4. Centralize defaulting, normalization, and validation in the settings service

HTTP handlers should authenticate, decode input, and map service errors to status codes. The settings service should own:

- default settings construction
- normalization of empty strings and casing-sensitive enum values
- validation of allowed enum values and payload shape
- upsert/read behavior

This matches the current backend pattern where route handlers stay thin and domain validation lives in service code.

### 5. Preserve backward safety by materializing defaults on read

Existing users will not have a `user_settings` row. `GET /api/v1/me/settings` must still succeed and return a complete default document. `PUT` should upsert a row atomically for first-time writers. This avoids migrations that need to backfill all users before rollout.

## Risks / Trade-offs

- [JSON document storage reduces relational strictness] -> Mitigation: enforce a typed Go input/output schema and validate before persistence.
- [Initial schema may miss future preference needs] -> Mitigation: keep the stored document versioned and additive so new fields can be introduced safely.
- [Frontend may assume settings immediately affect backend runtime behavior] -> Mitigation: document that this change establishes persistence and API contract first; behavioral consumers can be added separately.
- [Using `PUT` may require clients to send the full document] -> Mitigation: return a full canonical document from `GET`, and define defaults so clients can round-trip safely.

## Migration Plan

1. Add the new `user_settings` model to database AutoMigrate with a uniqueness constraint on `user_id`.
2. Implement service-layer read, validation, normalization, and upsert logic.
3. Register authenticated `/api/v1/me/settings` routes and handlers.
4. Add tests for defaults, persistence, validation failures, auth failures, and user isolation.
5. Roll out without backfill; existing users will receive defaults until they save explicit preferences.

Rollback strategy:
- If the feature must be disabled, stop routing requests to the new endpoints; existing persisted rows can remain unused because they are user-scoped additive data.

## Open Questions

- Should `appearance.locale` be constrained to a strict parser immediately, or treated as a trimmed opaque string in v1?
- Does product want the initial schema to include home-page preferences as part of this first release, or only appearance/playback preferences?
- Should future frontend clients receive a schema version field in the response from day one, or can that be added when the first incompatible shape change appears?
