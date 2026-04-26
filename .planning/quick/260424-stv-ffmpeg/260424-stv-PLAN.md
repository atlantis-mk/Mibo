# Quick Task 260424-stv: 扫描入库时应该使用ffmpeg获取背景图和封面图，这样如果没有元数据时还能显示这两张图片

## Plan Summary

Use the existing scan -> `probe_media_file` pipeline to generate fallback poster/backdrop artwork from the media file itself, then expose those generated assets through backend-owned URLs so home, discovery, and detail surfaces can still render images when TMDB/manual metadata is absent. Keep metadata artwork as the higher-priority source and only rely on ffmpeg-generated artwork when a field would otherwise be blank.

## Tasks

### 1. Add fallback artwork extraction to the probe pipeline
- files: `mibo-media-server/internal/probe/service.go`, `mibo-media-server/internal/config/config.go`, `mibo-media-server/internal/app/app.go`, `mibo-media-server/internal/worker/worker_test.go`
- action: Extend `probe.Service` to accept and use `config.FFmpegConfig`, resolve the same probe target already used for `ffprobe`, and run ffmpeg after a successful probe to derive poster/backdrop fallback artifacts for the owning media item. Persist backend-managed artwork references instead of filesystem paths so scan jobs stay deterministic without request-host context.
- verify: A scanned file with no matched metadata finishes probing with generated fallback artwork references, and scan/probe jobs still complete when ffmpeg is disabled or unavailable.
- done: Probe jobs can derive fallback poster/backdrop artwork from media files during library ingestion.

### 2. Serve generated artwork through API-owned URLs
- files: `mibo-media-server/internal/httpapi/router.go`, `mibo-media-server/internal/httpapi/handlers_media.go`, `mibo-media-server/internal/httpapi/handlers_libraries.go`, `mibo-media-server/internal/httpapi/handlers_auth.go`, `mibo-media-server/internal/httpapi/handlers_search.go`
- action: Add authenticated poster/backdrop endpoints for generated artwork and centralize a helper that rewrites backend-managed artwork references into absolute URLs in media item responses. Apply that helper anywhere `database.MediaItem` values are returned to the web client so the frontend receives usable image URLs in both dev and deployed setups.
- verify: `GET /api/v1/media-items/{id}`, discovery responses, and home/latest payloads return usable `poster_url`/`backdrop_url` values after scan-generated artwork is present.
- done: Frontend-facing media responses can render generated artwork without depending on TMDB/manual metadata.

### 3. Preserve artwork precedence and lock behavior with tests
- files: `mibo-media-server/internal/metadata/service_match.go`, `mibo-media-server/internal/worker/worker_test.go`, `mibo-media-server/internal/httpapi/router_test.go`
- action: Keep TMDB/manual artwork as the preferred source, only backfill poster/backdrop when metadata fields are blank, and add fake-ffmpeg/ffprobe tests that cover unmatched scans plus API delivery of generated assets.
- verify: Focused Go tests for worker scan/probe and router artwork serving pass, and metadata-driven paths still win when real artwork exists.
- done: Fallback artwork behavior is covered end to end and does not regress metadata-backed items.

## Assumptions

- Generated artwork is fallback-only; if TMDB or manual metadata provides poster/backdrop URLs, those remain authoritative.
- Because `poster_url` and `backdrop_url` are consumed directly by the frontend, backend-managed artwork must be surfaced as request-qualified URLs rather than raw local paths.
- Prefer reusing the existing ffmpeg/ffprobe configuration and fake-ffmpeg test patterns instead of introducing a separate artwork service or a new config surface unless implementation proves it necessary.
