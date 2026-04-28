## Context

Catalog-backed detail pages already receive asset file and stream summaries through `GET /api/v1/items/{id}`. The backend stores stream rows in `media_streams`, populated from `ffprobe -show_format -show_streams` for catalog inventory files. Today the parsed stream model only captures codec type/name, dimensions, channels, language, and title, while the UI renders video streams as a compact multi-line summary.

The desired UI is closer to a MediaInfo technical table for video streams: title, codec, profile, level, resolution, aspect ratio, interlace state, frame rate, bitrate, color space, bit depth, pixel format, and reference frames. Most of those values are available in `ffprobe` stream JSON but are not currently persisted or exposed through the catalog contracts.

## Goals / Non-Goals

**Goals:**

- Persist detailed video stream attributes from `ffprobe` for catalog inventory files when the values are available.
- Expose the attributes as optional catalog stream summary fields without breaking existing catalog detail consumers.
- Render the primary video stream information in a label/value format that matches the requested technical specification style.
- Keep partial data useful: missing attributes should be omitted or shown as unknown only where the existing UI already uses placeholders.
- Preserve existing audio, subtitle, file, playback, and governance behavior.

**Non-Goals:**

- Add a new external MediaInfo dependency or shell out to tools other than the existing `ffprobe` integration.
- Implement transcoding decisions, compatibility scoring, or playback blocking based on the new fields.
- Backfill every existing row automatically outside normal re-probe or scan flows.
- Replace the whole detail page layout or create a separate advanced diagnostics page.

## Decisions

1. Store detailed attributes as typed nullable columns on `media_streams`.

   The fields are stable stream facts and are already associated with one inventory file stream. Typed columns keep catalog queries simple and make API contracts explicit. A generic JSON blob was considered, but it would push formatting and schema drift into the frontend and make tests less precise.

2. Use `ffprobe` stream JSON as the single source of truth.

   The existing probe service already runs `ffprobe` with `-show_streams`, and fields such as `profile`, `level`, `field_order`, `avg_frame_rate`, `r_frame_rate`, `bit_rate`, `color_space`, `bits_per_raw_sample`, `pix_fmt`, and `refs` are available there depending on codec/container. Adding MediaInfo was considered, but it would introduce another runtime dependency and duplicate probe work.

3. Keep all new API fields optional and additive.

   Existing databases, disabled probes, and files with sparse metadata must continue to render. Catalog detail responses should omit unavailable fields rather than fail or synthesize misleading values. This preserves compatibility for frontend code that only needs the current compact fields.

4. Prefer per-stream bitrate when available, with container bitrate as fallback only for display continuity.

   The current catalog stream rows use the format bitrate for every stream. Video technical specs should represent the video stream bitrate when `ffprobe` provides it. If a video stream has no bitrate, the UI may still show the existing value as unknown or fallback depending on the formatter, but the backend should not label container bitrate as stream-specific data.

5. Format derived values in the frontend.

   Aspect ratio, user-facing frame-rate strings, normalized level display, interlace yes/no labels, and bit depth labels are presentation concerns. Persist raw probe values where possible and let the UI render Chinese labels that match the requested format.

## Risks / Trade-offs

- Some codecs/containers omit detailed stream fields -> render partial tables and keep the reprobe action visible so users understand missing data is probe-dependent.
- SQLite AutoMigrate adds nullable columns but does not populate old rows -> existing media will only gain details after re-probe or future scans.
- `avg_frame_rate` and `r_frame_rate` can differ or contain `0/0` -> parse defensively and prefer average frame rate when valid.
- `level` values are codec-specific integers -> display the normalized decimal form where applicable but preserve the raw value when normalization is uncertain.
- Per-stream bitrate may be absent more often than format bitrate -> avoid overstating precision; frontend copy should not imply a missing bitrate is an error.

## Migration Plan

- Add nullable fields to `database.MediaStream`; rely on existing database migration flow to add columns.
- Expand probe parsing and stream row construction for newly probed inventory files.
- Map database fields through catalog contracts and frontend types as optional properties.
- Update the detail UI to use the expanded fields when present while preserving compact behavior for sparse rows.
- Rollback is safe at the API/UI level because the fields are additive; if needed, frontend can ignore them and backend can stop writing them while leaving nullable columns in place.

## Open Questions

- Should existing catalog inventory files be re-probed automatically as part of implementation, or should users trigger reprobe per item/library when they want richer technical data?
- Should audio streams receive similar detailed fields in this change, or remain scoped to the requested video technical format?
