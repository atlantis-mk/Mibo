---
status: complete
quick_id: 260424-stv
slug: ffmpeg
completed_at: 2026-04-24T13:02:12Z
---

# Quick Task 260424-stv Summary

## Outcome

Completed scan-time fallback artwork generation for media ingestion: the backend now uses ffmpeg to derive poster/backdrop images from the media file itself, persists backend-owned artwork references when metadata artwork is missing, and serves those images through API URLs the frontend can render directly.

## What Changed

- Extended `internal/probe` so `probe_media_file` can run best-effort ffmpeg artwork extraction alongside ffprobe, even when metadata matching is skipped.
- Added `MIBO_ARTWORK_ROOT_PATH` support through `config.FFmpegConfig.ArtworkRootPath` so generated artwork can be isolated per runtime/test environment.
- Added `GET /api/v1/media-items/{id}/artwork/{kind}` and normalized backend-managed artwork paths into absolute frontend-safe URLs in media detail, discovery, home, latest, and recently-added responses.
- Updated metadata refresh logic to keep existing fallback artwork when TMDB does not provide a poster/backdrop/logo, instead of overwriting those fields with blanks.
- Added worker and router tests covering scan-generated artwork persistence, absolute URL rewriting, and artwork file serving.

## Validation

- `go test ./internal/probe ./internal/metadata ./internal/httpapi ./internal/worker`
- `go test ./...`

## Assumptions

- Generated artwork remains fallback-only; TMDB/manual artwork still wins when present.
- This run did not create a git commit, so the changes are currently uncommitted in the worktree.

## Commit

- Not created in this run.
