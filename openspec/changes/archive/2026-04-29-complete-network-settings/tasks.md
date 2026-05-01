## 1. Backend Settings Contract

- [x] 1.1 Add network settings request/response types to `mibo-media-server/internal/settings` with defaults matching the current `/settings/network` form.
- [x] 1.2 Persist network settings in the existing `system_settings` table under a `network` category, including secret handling for certificate password values.
- [x] 1.3 Validate IP/CIDR lists, port ranges, enum values, TLS fields, and streaming limit fields before saving.
- [x] 1.4 Return status metadata that distinguishes saved configuration from settings that are active only after restart or future runtime wiring.

## 2. Backend API

- [x] 2.1 Register authenticated `GET /api/v1/settings/network` and `PUT /api/v1/settings/network` routes.
- [x] 2.2 Implement handlers that decode payloads, call the settings service, map validation failures to `400`, and reject unauthenticated requests.
- [x] 2.3 Add backend tests for default reads, successful persistence, validation failures, secret masking/clearing, and unauthorized access.

## 3. Frontend API Integration

- [x] 3.1 Add network settings types and `getNetworkSettings`/`updateNetworkSettings` client methods in `web/src/lib/mibo-api.ts`.
- [x] 3.2 Add React Query keys and query options for network settings in `web/src/lib/mibo-query.ts`.
- [x] 3.3 Convert textarea values to and from API arrays for local networks and remote IP filters.

## 4. Network Settings UI

- [x] 4.1 Replace `localStorage` loading and saving in `NetworkSettingsPanel` with server query and mutation state.
- [x] 4.2 Show loading, save progress, success, and error states while preserving the user's draft on failed saves.
- [x] 4.3 Add clear runtime guidance for restart-required fields and configuration-only automatic port mapping.
- [x] 4.4 Ensure certificate password updates use masked/configured state and explicit clear behavior rather than echoing secrets.

## 5. Verification

- [x] 5.1 Run focused backend tests for settings and HTTP API behavior.
- [x] 5.2 Run `pnpm typecheck` from `web/`.
- [x] 5.3 Manually verify `http://localhost:3000/settings/network` loads, saves, reloads persisted values, and displays validation errors.
