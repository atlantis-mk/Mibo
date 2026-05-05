## Why

Fast video classification now depends on many filename and path rules that mix signal extraction, title cleanup, media-role detection, and movie-vs-episode decisions. This makes rules hard to maintain, risks losing useful filename-derived metadata during cleanup, and can misread release tokens such as audio channels, quality labels, or codecs as title text or weak episode numbers.

## What Changes

- Introduce a cached directory-aware filename signal classification pipeline for fast scans.
- Extract structured filename/path signals before title cleanup, including quality, source, codec, audio, subtitle, release group, role, episode, year, and path hints.
- Treat filename-derived technical and release values as metadata hints and classification evidence, not authoritative ffprobe or provider metadata.
- Build clean title views from preserved signals instead of silently discarding tokens before classification.
- Generate movie, episode, trailer, sample, extra, and version candidates with lightweight evidence summaries.
- Use per-directory summaries built from the current scan snapshot to improve flat episode, independent movie, movie-version, and attachment decisions without extra storage I/O.
- Keep the fast path free of ffprobe, provider calls, hashes, and media file reads; ambiguous or conflicting results remain provisional, reviewable, or eligible for later background refinement.

## Capabilities

### New Capabilities

- `filename-signal-classification`: Extract and use filename-derived metadata hints, lightweight evidence, and cached directory summaries during fast media classification.

### Modified Capabilities

- `fast-video-classification`: Fast classification must consume structured filename signals and directory summaries rather than interleaving raw regex cleanup with final semantic decisions.
- `title-normalization-pipeline`: Title cleanup must preserve filename-derived signals and removed-token evidence before producing cleaned title views.
- `media-graph-scanner`: Scanner projection must retain filename signal evidence and avoid treating release hints as authoritative technical metadata.
- `source-first-auto-classification`: Source-first scanning must remain responsive by using cheap filename signals and cached directory summaries before escalating ambiguous results.

## Impact

- Backend scanner code under `mibo-media-server/internal/library`, especially filename parsing, title cleaning integration, fast candidate generation, directory snapshot use, and decision evidence.
- Backend title normalization under `mibo-media-server/internal/titleclean` where cleanup behavior must align with preserved filename signals.
- Catalog scan artifacts and classification decision evidence that expose filename-derived metadata hints and candidate reasons.
- Backend tests for filename signal extraction, release-token preservation, audio/quality anti-misclassification, directory summaries, flat episode folders, movie versions, independent movies, trailers, samples, extras, and ambiguous review outcomes.
- No new frontend UI is required for the first implementation unless existing governance views need minor contract adjustments to display the new evidence summaries.
