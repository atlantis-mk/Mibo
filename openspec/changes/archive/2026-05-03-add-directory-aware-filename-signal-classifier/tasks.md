## 1. Filename Signal Model

- [x] 1.1 Define internal filename signal structures for raw path data, title tokens, identity signals, release hints, role hints, cleanup evidence, path hints, and lightweight evidence summaries
- [x] 1.2 Add signal kinds and reason constants for quality, source, codec, audio, subtitle, HDR, edition, release group, year, episode marker, role, title cleanup, and anti-misclassification evidence
- [x] 1.3 Add focused tests for signal model serialization or evidence conversion where existing decision evidence contracts require stable output

## 2. Signal Extraction

- [x] 2.1 Implement a filename/path signal extractor that runs before title cleanup and consumes only path strings, basenames, extensions, and path segments
- [x] 2.2 Move or reuse existing episode, year, season-directory, quality, edition, audio, role, website, release-group, and generic noise rules as signal probes instead of final semantic decisions
- [x] 2.3 Preserve filename-derived release hints while building cleaned title candidates from the remaining title tokens
- [x] 2.4 Add tests for dense movie release names, TV release names, Chinese episode markers, multi-episode ranges, URL watermarks, release groups, trailers, samples, PVs, featurettes, and extras

## 3. Anti-Misclassification Behavior

- [x] 3.1 Ensure audio channel tokens such as `5.1`, `7.1`, `DDP5.1`, and `TrueHD.Atmos.7.1` suppress weak numeric episode inference
- [x] 3.2 Ensure quality and codec tokens such as `2160p`, `1080p`, `x264`, `x265`, `H.264`, and `HEVC` are removed from title views and cannot create episode-number evidence
- [x] 3.3 Ensure filename-derived technical hints are not written into authoritative stream or playback technical metadata fields
- [x] 3.4 Add regression tests for movie filenames containing audio, quality, codec, source, and subtitle tokens that previously could look numeric or title-like

## 4. Directory Summary Cache

- [x] 4.1 Define per-directory summary structures for video counts, likely-main counts, attachment counts, explicit episode counts, numeric sequence evidence, title-year movie evidence, common title stems, version evidence, and season-directory hints
- [x] 4.2 Build directory summaries from scan traversal snapshots or already-listed directory entries without extra recursive storage probing
- [x] 4.3 Reuse cached directory summaries across files in the same scanned directory during classification
- [x] 4.4 Add tests for flat numeric episode folders, explicit episode sibling groups, movie-version folders, independent movies in one directory, and mixed attachment/main directories

## 5. Candidate Generation And Resolution

- [x] 5.1 Update fast candidate generation to consume filename signals and directory summaries rather than matching raw filename strings directly in decision code
- [x] 5.2 Generate movie, episode, trailer, sample, preview, extra, and version candidates with lightweight evidence summaries
- [x] 5.3 Apply conservative confidence thresholds so strong cheap evidence can confirm decisions, medium evidence remains provisional, and conflicting evidence becomes review-required
- [x] 5.4 Preserve alternatives when movie, episode, version, independent-movie, or attachment candidates conflict
- [x] 5.5 Add tests for high-confidence fast decisions, provisional decisions, and review-required decisions caused by cheap evidence conflicts

## 6. Scanner And Title Normalization Integration

- [x] 6.1 Integrate filename signal extraction into scanner classification before title cleanup and catalog projection
- [x] 6.2 Align `titleclean` usage with preserved signals so removed tokens remain available as filename-derived evidence
- [x] 6.3 Include filename signal evidence and directory summary evidence in resolver decisions or scanner metadata evidence without requiring a new frontend flow
- [x] 6.4 Preserve existing governance protections for matched, reviewed, locked, or manual catalog fields during rescans

## 7. Performance Guardrails

- [x] 7.1 Verify fast classification uses no ffprobe, provider calls, media-content reads, file hashing, artwork retrieval, or additional recursive source analysis
- [x] 7.2 Add or update tests proving directory summaries are built once per scan snapshot directory and reused across sibling classifications
- [x] 7.3 Add benchmark or focused test coverage for large sibling directories if the existing test structure supports it

## 8. Verification

- [x] 8.1 Run focused backend tests for title cleaning, filename signal extraction, scan classification, directory summaries, and governance decision evidence
- [x] 8.2 Run `go test ./...` from `mibo-media-server/`
- [x] 8.3 Run frontend typecheck only if backend API contracts or generated frontend types are changed
- [x] 8.4 Manually verify demo-media or representative fixtures for one-file movies, explicit episodes, flat numbered episodes, movie versions, trailers/extras/samples, audio-channel false positives, and ambiguous review outcomes
