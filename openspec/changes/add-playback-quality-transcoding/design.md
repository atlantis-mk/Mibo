## Context

The playback page currently receives a `PlaybackSource` and initializes Artplayer with the selected direct URL. Backend playback compatibility is assessed in `internal/playback` using container, video codec, and audio codec metadata from probe results. Secure runtime access to local and OpenList files is mediated through signed access routes, so transcoding must read media through backend-controlled grants rather than exposing raw provider URLs.

The missing piece is a browser-compatible fallback when original quality is not directly playable or when users intentionally choose a lower or different quality. FFmpeg is already configured in the server for subtitle extraction, but there is no long-lived playback transcode session manager, no HLS serving surface, and no frontend quality selector.

## Goals / Non-Goals

**Goals:**
- Add an original-quality default and a playback-page quality selector with 720P, 1080P, 2K, and 4K options when transcoding is available.
- Add an explicit audio repair option that keeps video at original quality when possible and transcodes only unsupported audio to browser-compatible AAC.
- Generate HLS output progressively so playback can start after initial segments are available instead of waiting for a full transcode.
- Ensure every non-original variant uses browser-compatible audio and video formats.
- Keep provider access secure by resolving source files on the backend and serving HLS manifests/segments through authenticated or signed routes.
- Support seeking by restarting or retargeting a transcode session near the requested position when the target segment does not already exist.
- Prefer hardware encoding when configured and available for full video transcodes, while keeping audio-only repair on `-c:v copy`.

**Non-Goals:**
- Persist permanently transcoded libraries or create offline optimized versions.
- Add adaptive bitrate ladders in a single master manifest; each selected quality is a single requested variant.
- Implement multi-audio-track selection, subtitle burn-in, DVR, or live TV transcoding.
- Guarantee frame-perfect seeking for every remote provider; cloud-drive Range behavior can still affect seek latency.

## Decisions

1. Return variant-aware playback metadata from the existing playback contract.

   Extend `PlaybackSource` with available variants and a selected variant. The original variant remains the default. Variant identifiers are stable strings such as `original`, `audio-repair`, `720p`, `1080p`, `2k`, and `4k`. The frontend requests a variant through playback query parameters, and the backend either returns the direct source for `original` or an HLS manifest URL for transcode variants.

   Alternative considered: create a separate preflight endpoint only for variants. Rejected for the first implementation because the play page already needs a single source object and query key; one contract keeps the UI and progress logic simpler.

2. Use HLS for all transcoded variants.

   FFmpeg will generate a playlist and short media segments under a server-managed transcode root. The frontend already has HLS wiring for m3u8 playback, so a transcoded variant can use the same Artplayer type path. Segment duration should default to 3-4 seconds to balance seek responsiveness and request count.

   Alternative considered: pipe fragmented MP4 directly through one HTTP response. Rejected because HLS handles progressive availability, rebuffering, and seek/session restarts more predictably in browser players.

3. Split transcode modes into audio repair and quality transcode.

   Audio repair uses `-c:v copy -c:a aac` whenever the video stream is browser-compatible and only audio fails compatibility. Quality transcode uses a target resolution and bitrate with browser-safe video and audio codecs, e.g. H.264 plus AAC for MP4/HLS segments. When a selected quality is equal to or above source resolution, the backend must avoid upscaling unless the variant is needed for codec compatibility.

   Alternative considered: always full-transcode fallback variants. Rejected because most no-audio cases only need audio conversion and full video transcoding would waste CPU/GPU.

4. Manage transcoding as short-lived playback sessions.

   A session key should include user, file/resource, variant, start position, source identity, and compatibility-relevant metadata. The session manager starts FFmpeg on demand, exposes readiness once the manifest and initial segment exist, tracks last access, and cleans up idle sessions. Only one active producer should own a session directory at a time.

   Alternative considered: spawn FFmpeg per segment request. Rejected because startup cost and remote provider access would make playback and seek behavior too uneven.

5. Keep authorization at the playback and HLS serving boundary.

   Playback APIs still require the authenticated user and library visibility checks. HLS manifest and segment URLs should be short-lived signed URLs or session-scoped routes tied to the user and purpose. FFmpeg source input should be resolved internally from the access layer or from a backend-only signed URL.

   Alternative considered: redirect the browser to provider URLs for transcoded media. Rejected because it bypasses Mibo access control and does not work for generated HLS segments.

6. Hardware encoding is optional and selected by capability detection.

   Add an encoder planner that can choose `h264_videotoolbox`, `h264_nvenc`, `h264_qsv`, `h264_vaapi`, or `libx264` based on configured preference and `ffmpeg -encoders` detection. Audio repair should not use a hardware video encoder because it copies video.

   Alternative considered: expose raw encoder names directly in the UI. Rejected because users should choose quality intent, while the server maps that intent to safe FFmpeg arguments.

## Risks / Trade-offs

- [Remote provider Range or signed URL expiry interrupts FFmpeg] → Resolve source access immediately before session start, refresh OpenList links through the access layer, and surface clear retryable errors when the provider cannot seek or expires early.
- [Full video transcoding cannot keep up with realtime] → Prefer audio repair when possible, support hardware encoders, choose conservative bitrates, and show loading while new segments catch up.
- [Seek to an ungenerated position feels slow] → Cache generated segments briefly, restart FFmpeg from the requested timestamp for misses, and keep segment duration short.
- [Transcode cache grows without bounds] → Enforce per-session idle TTL, max disk usage, and cleanup on process start and scheduled intervals.
- [Browser support differs by platform] → Use conservative Web profiles for generated HLS: H.264 video plus AAC audio, with original direct play still available when compatibility is known.
- [Quality menu suggests variants that cannot improve the source] → Hide or disable upscaling variants by default, while preserving codec-repair variants when needed.

## Migration Plan

1. Add backend variant models and compatibility planning without changing the default `original` playback response.
2. Add HLS session routes and FFmpeg command planning behind feature/config availability checks.
3. Update the frontend play page to render the quality control only when variants are returned.
4. Add integration tests for original fallback behavior, audio repair, quality transcode planning, HLS authorization, and session cleanup.
5. Roll back by disabling transcoding in configuration; original direct playback remains the default path.

## Open Questions

- Should the first implementation expose a manual hardware encoder setting, or only auto-detect with server logs?
- What default transcode root and disk quota should ship for desktop/server deployments?
- Should 2K map to 1440p exactly, or should it preserve cinematic 2048-wide sources as a width-bound profile?
