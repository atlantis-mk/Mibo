## Why

Scanned media titles are currently normalized by separate, partially overlapping rules in the library scanner and TMDB matcher, which leaves gaps for website watermarks, release-site tags, and noisy filename tokens. A unified title normalization pipeline will make catalog ingestion, metadata matching, and troubleshooting more reliable while preserving the original filename evidence for review.

## What Changes

- Add a shared backend title normalization capability for scanner and metadata search usage.
- Normalize filename-derived titles by removing common website watermarks, release-site tokens, quality labels, HDR labels, codecs, source labels, streaming platform tags, audio/subtitle markers, years, and release groups.
- Extract year information into structured metadata while removing it from display/search title candidates.
- Preserve original titles and record normalization evidence, including removed tokens and reasons, in scanner metadata payloads.
- Keep technical quality fields such as resolution and codec sourced from ffprobe/probe results rather than filename text.
- Keep existing catalog governance protections so matched, reviewed, locked, or manually edited descriptive fields are not overwritten by rescans.

## Capabilities

### New Capabilities
- `title-normalization-pipeline`: Defines how scanner-derived and metadata-search titles are normalized, how evidence is recorded, and how technical filename noise is handled.

### Modified Capabilities

## Impact

- Affects backend scanner classification in `mibo-media-server/internal/library/scan*.go`.
- Affects TMDB search query construction in `mibo-media-server/internal/metadata/service_matcher.go`.
- Adds or updates backend tests for movie, TV, Chinese filenames, URL/site watermarks, release groups, and noisy technical tokens.
- May add an internal shared package for normalization logic; no public API or frontend contract changes are expected.
