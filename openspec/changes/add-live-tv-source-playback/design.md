## Context

`frontend/src/features/settings/components/live-tv-settings-panel.tsx` already exposes a Live TV management surface, but it is entirely placeholder-driven and persists only advanced form fields in browser `localStorage`. The backend currently has no Live TV domain model, no source persistence, no playlist parsing capability, and no channel playback contract.

At the same time, the backend already has two patterns this change should align with:

- server-managed admin settings and persistence through structured services and database-backed models
- backend-controlled media access and remote URL proxying through the existing playback/access stack

This change crosses both apps. It introduces a new backend capability, a new frontend integration path, new persisted data, and a playback flow that cannot safely reuse the current catalog-only mental model without additional adaptation.

The user input discussed for this change is concrete: the system must support remote IPTV playlist URLs in both `.txt` and `.m3u` forms, using sources similar to `vbskycn/iptv`, and must provide an end-to-end path from source import to playable channel.

## Goals / Non-Goals

**Goals:**
- Add an admin-managed Live TV source model for remote playlist URLs.
- Support importing channels from remote `.txt` and `.m3u` sources.
- Normalize imported channels into a stable backend representation independent of source file format.
- Provide authenticated APIs to create, update, delete, refresh, and list Live TV sources.
- Provide authenticated APIs to browse imported Live TV channels.
- Provide a backend-controlled Live TV playback URL that proxies upstream channel streams.
- Replace frontend placeholder behavior with real backend-backed source and channel workflows.

**Non-Goals:**
- Do not implement XMLTV/EPG ingestion in this change.
- Do not implement DVR, recording schedules, timeshift buffers, or post-processing execution in this change.
- Do not support HDHomeRun or tuner device discovery in this change.
- Do not migrate current advanced Live TV browser `localStorage` fields into server persistence yet.
- Do not force Live TV playback through the existing metadata item or inventory file playback contracts.

## Decisions

### 1. Model Live TV sources and channels as dedicated persisted entities

This change should not store source definitions in `system_settings` key/value rows alone. Source management needs per-record lifecycle, validation state, fetch metadata, timestamps, and one-to-many channel ownership. Likewise, imported channels need stable identity, display fields, and playback targets that survive refresh cycles.

Recommended persistence shape:

- `live_tv_sources`
  - `id`
  - `name`
  - `source_type` (`playlist_url`)
  - `format_hint` (`auto | m3u | txt`)
  - `url`
  - `enabled`
  - `last_refresh_at`
  - `last_refresh_status`
  - `last_refresh_error`
  - timestamps
- `live_tv_channels`
  - `id`
  - `source_id`
  - `source_channel_key`
  - `name`
  - `group_name`
  - `logo_url`
  - `tvg_id`
  - `tvg_name`
  - `stream_url`
  - `raw_attributes_json`
  - `sort_order`
  - `enabled`
  - timestamps

The `source_channel_key` should be deterministic per source refresh, derived from the most stable available combination of fields such as `tvg-id`, channel name, and stream URL. This allows upsert-style refresh behavior instead of blind duplication.

Alternatives considered:
- Store everything as JSON blobs in `system_settings`: rejected because CRUD, refresh tracking, and channel identity become awkward.
- Parse playlists on every request and avoid persistence: rejected because it increases latency, hides refresh failures, and prevents stable channel browsing.

### 2. Support remote URL sources only in v1, with dual parser support for `.m3u` and `.txt`

The user’s immediate need is remote URL ingestion, not local file upload. The source model should therefore accept a remote URL and an optional format hint. The refresh path fetches the remote content, detects or respects the format, and then hands the content to one of two parsers:

- `M3UParser` for extended M3U playlists with `#EXTM3U`, `#EXTINF`, and optional attributes such as `tvg-id`, `tvg-name`, `tvg-logo`, and `group-title`
- `TXTParser` for simpler line-based channel definitions, with support for common `name,url` or grouped text playlist conventions where feasible

Both parsers should emit the same normalized channel structure before persistence.

Alternatives considered:
- Support only `.m3u` initially: rejected because the user explicitly needs both formats.
- Support upload/paste/import flows immediately: rejected to keep the first backend surface small and aligned to the current source examples.

### 3. Make refresh an explicit server action, not implicit on every read

Source import should behave like a managed fetch operation. Creating or updating a source may optionally trigger an initial refresh, but browsing sources/channels should read from persisted state instead of forcing a network fetch every time. This makes failures observable and decouples admin browsing from upstream availability.

Recommended API pattern:

- `GET /api/v1/live-tv/sources`
- `POST /api/v1/live-tv/sources`
- `PATCH /api/v1/live-tv/sources/{id}`
- `DELETE /api/v1/live-tv/sources/{id}`
- `POST /api/v1/live-tv/sources/{id}/refresh`
- `GET /api/v1/live-tv/channels`

`GET /api/v1/live-tv/channels` should support filters such as `source_id`, `group`, `q`, and `enabled`.

Alternatives considered:
- Auto-refresh on every channel list request: rejected because it couples UX to remote latency and failures.
- Background scheduled refresh in v1: rejected because it adds worker/scheduler complexity before the core import path is validated.

### 4. Use a dedicated Live TV playback endpoint and backend proxy contract

Imported channels represent external live streams, not catalog items or inventory files. The current playback page and backend contracts are built around metadata items, resources, and file-backed playback. Reusing that contract directly would create semantic mismatches and unnecessary fake catalog objects.

Instead, add a dedicated Live TV playback contract:

- `GET /api/v1/live-tv/channels/{id}/playback`

This endpoint returns a lightweight playback payload containing channel metadata and a backend URL owned by Mibo. The actual stream request should go through a proxy endpoint under backend control, for example:

- `GET /api/v1/live-tv/channels/{id}/stream`

The stream handler should:
- resolve the persisted upstream stream URL
- fetch or redirect through a backend-controlled path
- preserve relevant response headers when proxying
- avoid exposing raw provider URLs directly to the browser where possible

This aligns with the existing access/proxy philosophy while keeping Live TV independent from inventory-file access grants.

Alternatives considered:
- Return upstream stream URLs directly to the frontend: rejected because it leaks provider URLs and weakens future control over auth, headers, and observability.
- Create fake catalog items/resources for every channel: rejected because it adds domain distortion and unnecessary ingestion complexity.

### 5. Build a lightweight frontend Live TV playback entry instead of forcing the current catalog player contract

The current frontend player can consume a raw URL, but the surrounding route logic expects catalog items or inventory file playback. Live TV should therefore get a dedicated frontend entry that consumes the Live TV playback payload directly and initializes the player without catalog metadata requirements.

The settings page should be updated to:
- load sources from backend APIs
- create/edit/delete sources
- trigger refresh
- render imported channels from backend data
- launch channel playback through a Live TV-specific route or modal

Alternatives considered:
- Keep the settings page read-only and build backend only: rejected because the current user request explicitly asks to complete source addition and playback.
- Force channel playback through the existing `/play` route immediately: rejected because it adds avoidable coupling and likely UI conditionals.

### 6. Treat advanced Live TV settings as deferred and keep them visibly out of backend scope for this change

The existing advanced tab contains fields for buffer limits, guide days, recording folders, and post-processing. Those fields imply later DVR and EPG behavior that this change explicitly does not implement. They should remain outside the backend contract for now to avoid writing incomplete server-side configuration that has no runtime consumer.

Alternatives considered:
- Persist advanced settings now for future use: rejected because it creates configuration debt and suggests runtime support that does not yet exist.

## Risks / Trade-offs

- [TXT playlist formats are inconsistent across providers] -> Mitigation: define and document a supported subset, store parse failures clearly, and keep parser normalization conservative.
- [Upstream live streams may require redirects, unstable origins, or uncommon headers] -> Mitigation: centralize stream proxy logic and surface refresh/playback failures with clear backend error messages.
- [Live TV introduces a second playback pathway beside catalog playback] -> Mitigation: keep payloads structurally similar where helpful, but isolate route and API semantics cleanly.
- [Persisted channel rows can become stale when upstream playlists change frequently] -> Mitigation: provide explicit refresh, source timestamps, and replace/upsert semantics during refresh.
- [Frontend scope may expand if advanced settings are partially wired] -> Mitigation: limit this change to sources, channel listing, and playback only.

## Migration Plan

1. Add database models and migrations/AutoMigrate coverage for Live TV sources and channels.
2. Implement parser and normalization services for `.m3u` and `.txt` remote playlists.
3. Implement source CRUD and refresh APIs.
4. Implement channel list and channel playback APIs.
5. Update the frontend Live TV settings page to consume the new APIs and remove placeholder-only source/channel actions.
6. Add a Live TV playback entry in the frontend and validate end-to-end playback using imported channels.
7. Add tests for persistence, parsing, authorization, refresh, channel listing, and playback proxy behavior.

Rollback strategy:
- If rollout must be reversed, disable the new Live TV routes and frontend integration.
- Persisted source/channel rows can remain as additive unused data until a cleanup migration is chosen later.

## Open Questions

- Which TXT playlist conventions must be supported in v1 beyond basic `name,url`-style entries?
- Should initial source creation trigger an automatic first refresh, or should refresh remain entirely explicit?
- Does the frontend want Live TV playback as a dedicated route, an overlay modal, or a tab within settings for the first iteration?
