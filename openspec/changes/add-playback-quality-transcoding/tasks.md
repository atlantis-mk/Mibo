## 1. Playback Variant Contract

- [x] 1.1 Add backend playback variant types for `original`, `audio-repair`, `720p`, `1080p`, `2k`, and `4k`.
- [x] 1.2 Extend `PlaybackSource` JSON with available variants, selected variant, playback mode, and HLS manifest metadata.
- [x] 1.3 Add playback query parsing for requested variant and optional seek/start position.
- [x] 1.4 Update frontend API types and query keys so variant changes refetch playback safely.

## 2. Compatibility and Variant Planning

- [x] 2.1 Extend compatibility planning to distinguish direct-play, audio-only repair, and full video transcode needs.
- [x] 2.2 Implement target quality profiles with max resolution, bitrate defaults, and no-upscale behavior.
- [x] 2.3 Add FFmpeg encoder capability detection for hardware H.264 encoders and CPU fallback.
- [x] 2.4 Add unit tests for audio repair selection, full transcode selection, unavailable FFmpeg behavior, and target quality filtering.

## 3. Transcode Session Manager

- [x] 3.1 Create a playback transcode session manager with session keys, directories, process ownership, last-access tracking, and idle cleanup.
- [x] 3.2 Implement FFmpeg command planning for audio repair using video copy and AAC audio.
- [x] 3.3 Implement FFmpeg command planning for quality transcode using H.264 video, AAC audio, target scaling, and HLS output.
- [x] 3.4 Add startup readiness detection for generated manifests and initial segments.
- [x] 3.5 Add seek handling that reuses cached segments when possible and restarts near the requested timestamp when needed.
- [x] 3.6 Add backend tests for session reuse, readiness, cleanup, seek restart, and FFmpeg command generation.

## 4. HLS Access Routes

- [x] 4.1 Add authenticated or signed routes for transcode manifests and generated HLS segments.
- [x] 4.2 Ensure manifest and segment handlers enforce user/session authorization and do not expose provider URLs or local paths.
- [x] 4.3 Route FFmpeg source reads through secure backend access grants for local and OpenList media.
- [x] 4.4 Add HTTP tests for authorized manifest access, unauthorized segment rejection, and cloud-drive access refresh failures.

## 5. Frontend Playback Controls

- [x] 5.1 Add a quality selector button to the play page controls using the returned variant list.
- [x] 5.2 Default the selector to original quality and show audio repair when offered by the backend.
- [x] 5.3 Switch playback variant by refetching playback metadata and preserving the current position when possible.
- [x] 5.4 Display loading and fallback/error states while a transcode session prepares initial HLS segments.
- [x] 5.5 Update playback mode badges and info panel labels for original, audio repair, and quality transcode modes.

## 6. Verification

- [x] 6.1 Add or update frontend tests for quality menu rendering, option selection, and variant refetch behavior.
- [x] 6.2 Run focused backend tests for playback and HTTP API packages.
- [x] 6.3 Run `go test ./...` in `mibo-media-server`.
- [x] 6.4 Run focused frontend tests for the play feature.
- [ ] 6.5 Manually verify original playback, audio repair playback, and one quality transcode path in the browser.
