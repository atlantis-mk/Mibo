## 1. Data Model And Parsing Foundation

- [x] 1.1 Add persistent database models for `live_tv_sources` and `live_tv_channels` and include them in backend migration/AutoMigrate coverage.
- [x] 1.2 Define typed backend contracts for Live TV source records, refresh status, normalized channel records, and playback payloads.
- [x] 1.3 Implement remote playlist fetch and format detection for `.m3u` and `.txt` Live TV source URLs.
- [x] 1.4 Implement parser and normalization logic that converts supported M3U and TXT entries into a shared channel model.

## 2. Source Management And Refresh APIs

- [x] 2.1 Implement a Live TV service layer for source CRUD, validation, refresh execution, and channel upsert/replace behavior.
- [x] 2.2 Register authenticated/admin HTTP routes for listing, creating, updating, deleting, and refreshing Live TV sources.
- [x] 2.3 Implement handlers that return canonical source records, validation failures, and observable refresh status/error details.

## 3. Channel Listing And Playback

- [x] 3.1 Implement an authenticated channel listing API with basic source/group/query filtering over normalized channel records.
- [x] 3.2 Implement a Live TV playback endpoint that returns a backend-owned playback payload for an imported channel.
- [x] 3.3 Implement a backend-controlled Live TV stream proxy endpoint that opens the upstream stream URL and relays the response safely.

## 4. Frontend Live TV Integration

- [x] 4.1 Extend `frontend/src/lib/mibo-api.ts` and query helpers with Live TV source, refresh, channel list, and playback API methods.
- [x] 4.2 Replace placeholder source management and channel empty-state actions in `frontend/src/features/settings/components/live-tv-settings-panel.tsx` with backend-backed workflows.
- [x] 4.3 Add a lightweight Live TV playback entry in the frontend that can launch imported channels without depending on catalog item playback state.

## 5. Verification

- [x] 5.1 Add backend tests covering source validation, parser normalization for `.m3u` and `.txt`, refresh persistence, and authorization behavior.
- [x] 5.2 Add backend tests covering channel listing, playback payload resolution, and stream proxy failure handling.
- [x] 5.3 Add frontend tests for Live TV settings interactions or API wiring where the current test setup supports them.
- [x] 5.4 Run the relevant frontend and backend test suites for the new Live TV flow.
