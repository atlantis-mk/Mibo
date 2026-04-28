## Context

The backend scanner currently walks storage directories, filters video files, classifies each video from its path, and writes inventory/catalog records. It records scanner evidence and queues ffprobe, while fallback artwork already checks same-folder image files during probe. Subtitle and metadata sidecar files are currently ignored even when they contain useful local evidence.

The scanner supports both local paths and OpenList-backed storage providers, so the feature must work through the storage provider abstraction instead of direct filesystem access except where local-provider behavior already exposes local paths.

## Goals / Non-Goals

**Goals:**

- Discover `.srt`, `.ass`, `.nfo`, and `.json` sidecar files located in the same folder as a scanned video.
- Associate sidecars with the matching video using deterministic filename rules.
- Preserve sidecar evidence in scanner metadata so later catalog views and governance tools can explain local decisions.
- Use metadata sidecars as safe classification hints before creating or updating catalog rows.
- Keep scans resilient when sidecar files are unreadable, malformed, or too large.

**Non-Goals:**

- Full subtitle playback integration or subtitle selection UI.
- Recursive sidecar discovery outside the video's folder.
- A complete NFO standard implementation for every media manager variant.
- Overriding locked, manual, matched, or needs-review descriptive metadata.
- Parsing subtitle text contents for title or episode inference.

## Decisions

- Add sidecar discovery during `walkDirectory` after listing a directory once.
  - Rationale: the scanner already has all sibling objects for the current directory, avoiding extra list calls per video.
  - Alternative considered: query `provider.Get` for candidate paths per video. This is simpler but scales poorly in folders with many videos.

- Match sidecars by basename and common folder-level metadata names.
  - Video-specific sidecars match `<video-base>.srt`, `<video-base>.ass`, `<video-base>.nfo`, and `<video-base>.json`.
  - Folder-level metadata sidecars match `movie.nfo`, `tvshow.nfo`, `season.nfo`, `metadata.json`, and `info.json` only when there is a single plausible video or the sidecar type naturally applies to the hierarchy level.
  - Rationale: basename matching is predictable and avoids accidentally applying unrelated files.

- Store sidecar evidence in scanner metadata source payloads first.
  - Rationale: the current catalog scan path already records local scanner evidence per item. Extending that payload is lower risk than adding new tables before playback requirements are finalized.
  - Alternative considered: create dedicated subtitle asset rows immediately. This would be useful for playback, but it expands API and player scope beyond the requested scanner capability.

- Parse `.json` as structured metadata and `.nfo` with a small XML-first parser plus conservative text fallback.
  - Rationale: JSON sidecars can be mapped directly. NFO files are often XML, but some are loose text; the scanner should extract only high-confidence fields.
  - Supported fields for hints: title, original title, year, media type, series title, season number, episode number, and external IDs when present.

- Apply sidecar hints only through the existing scanner update and metadata preservation rules.
  - Rationale: existing governance behavior prevents local scans from overwriting curated descriptive fields for matched/manual/locked items.

## Risks / Trade-offs

- Incorrect sidecar association -> Use basename matching by default and limit folder-level sidecars to unambiguous cases.
- Malformed or huge sidecar files slowing scans -> Enforce a small read limit and treat parse failures as non-fatal evidence warnings.
- Provider differences in reading sidecars -> Use `storage.Provider.Get`/`Link` where available and keep local filesystem reads behind provider-resolved paths only when already safe.
- Scope creep into subtitle playback -> Record subtitle sidecars as evidence now and leave player/API exposure to a later change.
