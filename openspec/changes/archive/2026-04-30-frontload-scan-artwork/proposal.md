## Why

First-time library scans make catalog items visible quickly, but artwork often appears much later because selected images currently depend on asynchronous metadata matching or probe/ffmpeg fallback work. Frontloading cheap, deterministic artwork during scan improves perceived readiness without making scans wait on remote providers or media decoding.

## What Changes

- Detect and apply local sibling artwork files such as poster, cover, folder, backdrop, and fanart during the initial scan write path.
- Apply provider-supplied thumbnail URLs as provisional selected artwork when no better selected image exists.
- Preserve remote metadata matching and ffmpeg frame extraction as background enrichment paths that can replace provisional artwork later.
- Use sidecar-provided external IDs during scan so later metadata jobs can skip remote search when possible.
- Do not move ffprobe, ffmpeg extraction, or remote metadata requests into the blocking scan phase.

## Capabilities

### New Capabilities
- `scan-phase-artwork-preselection`: Defines how the scan phase can preselect low-cost artwork and identity hints before asynchronous enrichment completes.

### Modified Capabilities
- `sidecar-metadata-files`: Scanner-provided sidecar external IDs can seed catalog identities early enough for later metadata detail enrichment to avoid search.
- `openlist-metadata-utilization`: Provider thumbnail metadata can be used as provisional selected artwork during initial catalog scan.

## Impact

- Backend scan pipeline in `mibo-media-server/internal/library`, especially catalog scan artifact creation and write logic.
- Catalog image persistence in `mibo-media-server/internal/database` and `mibo-media-server/internal/catalog` only as needed to preserve selected-image semantics.
- Probe fallback artwork in `mibo-media-server/internal/probe` remains asynchronous but must not overwrite higher-quality local or remote selected images.
- Metadata matching in `mibo-media-server/internal/metadata` should benefit from external IDs seeded during scan.
- No frontend API contract changes are expected; existing `selected_images` fields should show artwork earlier.
