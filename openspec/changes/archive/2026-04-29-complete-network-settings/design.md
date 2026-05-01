## Context

`/settings/network` already exists in the frontend and presents a comprehensive network form, but it stores values only in `localStorage` and displays a notice that server integration is pending. The backend already has a `settings.Service` backed by `database.SystemSetting` rows and authenticated settings endpoints for metadata and scan settings, which is the closest existing pattern for this work.

The implementation should complete the current page without introducing a separate settings architecture. Network settings should be persisted centrally and exposed through authenticated API endpoints, while runtime-sensitive values such as listening ports and TLS mode should be clearly represented as saved configuration that may require restart or later runtime wiring.

## Goals / Non-Goals

**Goals:**
- Persist network settings on the server and return stable defaults when no settings have been saved.
- Provide authenticated read/update APIs that validate the network settings payload.
- Update the existing network settings UI to load from the API, save through the API, show errors and loading states, and avoid browser-local persistence as the source of truth.
- Keep the settings contract explicit about which values are currently configuration-only versus immediately active.
- Add focused backend tests for defaults, validation, persistence, and auth behavior.

**Non-Goals:**
- Dynamically rebinding the HTTP server listener after a settings save.
- Implementing real UPnP/NAT-PMP port mapping.
- Installing, parsing, or storing uploaded certificate file contents.
- Reworking unrelated localStorage-backed settings panels.

## Decisions

1. Store network settings in the existing `system_settings` table under a new `network` category.
   - Rationale: metadata and scan settings already use this storage model, so the change avoids a new table and keeps settings migration minimal.
   - Alternative considered: a dedicated network settings table. That would add schema overhead without meaningful query or lifecycle benefits because these are singleton server settings.

2. Expose `GET /api/v1/settings/network` and `PUT /api/v1/settings/network` from the existing settings service and router.
   - Rationale: this matches the existing settings API shape and keeps frontend client additions small.
   - Alternative considered: embedding network settings in a broader settings endpoint. That would increase coupling between unrelated settings surfaces.

3. Normalize list-style values as arrays in the API contract, while allowing the UI to keep textarea editing locally.
   - Rationale: arrays are easier to validate and test server-side, while the existing UI can still display one entry per line.
   - Alternative considered: storing newline-delimited strings end to end. That would mirror the current UI but makes validation and future runtime use less precise.

4. Treat certificate paths and password as settings values, not file uploads.
   - Rationale: the current UI only captures a file name/path, and backend certificate installation is outside this change. Password storage must use secret settings semantics if included.
   - Alternative considered: uploading certificate files through the settings endpoint. That requires file storage, permissions, and lifecycle decisions that are not necessary to complete the page.

5. Return an `effective_status` or equivalent metadata for restart/future-runtime warnings rather than pretending every saved field is active immediately.
   - Rationale: administrators need honest feedback for ports, TLS, and port mapping fields; the page can be complete without unsafe runtime rebinding.
   - Alternative considered: hide all inactive fields. That would undercut the existing network page scope and make future runtime integration harder.

## Risks / Trade-offs

- Runtime mismatch between saved settings and active server listeners -> show clear saved-versus-active guidance for fields that do not take effect immediately.
- Invalid CIDR, IP, port, or bitrate inputs blocking administrators -> validate on the server with field-specific error messages and mirror constraints in frontend controls.
- Secret certificate password exposure -> never return the stored password; return only a masked/configured indicator and support clear/update semantics.
- Existing browser-local drafts could be lost -> document that server values are now authoritative and optionally initialize from server defaults rather than importing localStorage.

## Migration Plan

- Add network settings keys to the existing `system_settings` persistence path; no separate table migration is required if the table already exists.
- Deploy backend endpoints before or with the frontend so the page can load from the server.
- Rollback is safe because unused `network` category rows can remain in `system_settings`; older builds will ignore them.

## Open Questions

- Should certificate password support be included in the first implementation, or should the frontend field be disabled until certificate installation is implemented?
- Should the page expose a migration notice for any existing `mibo-web-network-settings` localStorage values, or simply replace them with server defaults?
