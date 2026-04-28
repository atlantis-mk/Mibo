## Why

Current library scans only use the video file path, provider metadata, and probe output. Many media folders already contain sidecar subtitle and metadata files that can improve local evidence, preserve external subtitles, and reduce manual correction when filenames are noisy or provider metadata is incomplete.

## What Changes

- Discover supported sidecar files in the same folder as each scanned video.
- Support `.srt` and `.ass` subtitle sidecars as catalog/inventory evidence for the matching video asset.
- Support `.nfo` and `.json` metadata sidecars as scanner evidence that can provide local title/year/series/season/episode hints where safe.
- Record sidecar evidence without overriding locked or manually curated catalog metadata.
- Keep unsupported or malformed sidecar files non-fatal to the scan.

## Capabilities

### New Capabilities
- `sidecar-metadata-files`: Scanner support for local subtitle and metadata sidecar files located next to media files.

### Modified Capabilities

## Impact

- Backend scan pipeline in `mibo-media-server/internal/library`.
- Inventory/catalog evidence records and metadata source payloads.
- Optional playback or detail responses may later expose sidecar subtitles if implemented as associated assets.
- Backend tests for sidecar discovery, parsing, malformed files, and metadata preservation behavior.
