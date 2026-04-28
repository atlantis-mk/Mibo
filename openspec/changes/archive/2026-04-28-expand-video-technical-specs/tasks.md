## 1. Data Model

- [x] 1.1 Add nullable detailed technical fields to `database.MediaStream` for profile, level, frame-rate raw values, field order, color space, bit depth, pixel format, reference frames, and stream-specific bitrate support.
- [x] 1.2 Update database migration/index tests to verify existing AutoMigrate coverage still migrates `media_streams` successfully with the new columns.

## 2. Probe And Catalog API

- [x] 2.1 Expand `ffprobeStream` parsing to read detailed video fields from `ffprobe -show_streams` output.
- [x] 2.2 Update `buildInventoryMediaStreams` to persist detailed video attributes and prefer per-stream bitrate when present.
- [x] 2.3 Extend `CatalogMediaStreamSummary` and `buildCatalogMediaStreamSummary` to expose the new optional fields through catalog item detail responses.
- [x] 2.4 Update backend probe and catalog query tests to cover detailed video attributes, sparse probe data, and compatibility with existing compact stream fields.

## 3. Frontend Presentation

- [x] 3.1 Extend `CatalogMediaStreamSummary` in `web/src/lib/mibo-api.ts` with optional detailed video technical fields.
- [x] 3.2 Add frontend formatting helpers for frame rate, interlace state, level display, bit depth, and nullable technical values.
- [x] 3.3 Update the media detail `SpecsSection` video card to render MediaInfo-style label/value rows for each video stream while preserving audio, subtitle, and file summaries.
- [x] 3.4 Ensure sparse stream rows still render useful compact video metadata without empty technical rows.

## 4. Verification

- [x] 4.1 Run focused backend tests for probe, catalog query/contracts, and database migration coverage.
- [x] 4.2 Run frontend typecheck from `web/`.
- [x] 4.3 Manually inspect a catalog media detail response or UI fixture with detailed HEVC-style stream data to confirm the requested fields render correctly.
