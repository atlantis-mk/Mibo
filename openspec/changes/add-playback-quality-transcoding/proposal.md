## Why

Mibo can currently hand browser clients a direct playback URL, but direct playback is not enough for common cloud-drive media where the container plays while the audio or video codec does not. Users need an explicit quality selector and a compatible fallback path so playback can remain smooth, audible, and browser-safe without requiring pre-converted files.

## What Changes

- Add a playback-page quality control that defaults to original quality and offers selectable targets such as 720P, 1080P, 2K, and 4K when transcoding is available.
- Add an audio compatibility repair option for cases where the original video is otherwise direct-playable but the audio codec is not supported by the browser.
- Extend playback source responses so the frontend can request a chosen playback variant and understand whether it is original direct play, audio-only repair, or quality transcoding.
- Add backend FFmpeg-based HLS transcoding sessions that generate browser-compatible video and audio streams for the selected target profile.
- Support low-latency start by streaming generated HLS segments while transcoding continues instead of waiting for the full media item to finish.
- Preserve secure provider access by resolving cloud-drive or local files through backend access grants rather than exposing raw provider URLs to the browser.
- Provide session lifecycle, seek handling, cache cleanup, and clear fallback behavior when FFmpeg or a requested hardware encoder is unavailable.

## Capabilities

### New Capabilities
- `playback-quality-selection`: Browser playback quality and compatibility option selection, including original-quality defaults and user-selectable transcode profiles.
- `playback-transcoding`: Backend-managed FFmpeg transcoding sessions that produce browser-compatible HLS variants for audio repair and selected quality targets.

### Modified Capabilities
- None.

## Impact

- Affected frontend code in `frontend/src/features/play`, playback API/query wiring in `frontend/src/lib/mibo-api.ts` and `frontend/src/lib/mibo-query.ts`, and related UI state for variant selection.
- Affected backend code in `mibo-media-server/internal/playback`, `mibo-media-server/internal/httpapi`, access-provider integration, FFmpeg configuration, and session cleanup.
- New HTTP API surface for listing playback variants, requesting a variant, serving HLS manifests, and serving generated media segments.
- New runtime dependency on configured FFmpeg for non-original variants, with optional hardware encoder detection for full video transcodes.
- New tests covering browser compatibility decisions, variant selection, HLS session startup, audio repair, quality transcode command planning, seek/session behavior, authorization, and frontend controls.
