## Why

Mibo already exposes a Live TV settings page in the frontend, but the current experience is only a placeholder and cannot persist live source configuration or play channels through the backend. We need a first real Live TV backend contract now so the existing UI can evolve from local-only mock state into a supported workflow for importing IPTV source lists and watching channels.

## What Changes

- Add backend support for managing remote Live TV playlist sources from the admin settings surface.
- Support importing and refreshing channel data from remote `.m3u` and `.txt` IPTV source URLs.
- Normalize parsed source entries into a stable channel model that the frontend can browse without knowing the original playlist format.
- Expose authenticated APIs to list sources, refresh them, and browse imported channels.
- Add a backend playback path for Live TV channels that proxies upstream stream URLs instead of exposing provider URLs directly to the client.
- Update the frontend Live TV settings page to use backend APIs for source management and channel browsing instead of `localStorage` placeholders.
- Explicitly defer XMLTV/EPG ingestion, DVR/recording workflows, HDHomeRun tuner discovery, and advanced recording execution to future changes.

## Capabilities

### New Capabilities
- `live-tv-sources`: Admin-managed Live TV source ingestion for remote `.m3u` and `.txt` playlists, including channel normalization and refresh workflows.
- `live-tv-playback`: Authenticated Live TV channel playback that resolves imported channel streams through a backend-controlled proxy path.

### Modified Capabilities
- None.

## Impact

- Affected backend code in `mibo-media-server/internal/httpapi`, `mibo-media-server/internal/settings`, and new Live TV domain/service/database layers.
- New persistence for Live TV sources and imported channels, plus parsing logic for multiple playlist formats.
- New authenticated/admin API surface for source CRUD, refresh, channel listing, and channel playback.
- Affected frontend settings code in `frontend/src/features/settings/components/live-tv-settings-panel.tsx` and related API/query wiring.
- New tests covering source validation, parsing, refresh behavior, authorization, channel listing, and playback proxy behavior.
