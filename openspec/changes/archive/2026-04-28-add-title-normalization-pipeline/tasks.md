## 1. Shared Normalizer

- [x] 1.1 Add `mibo-media-server/internal/titleclean` with `NormalizeInput`, `NormalizeResult`, `RemovedToken`, and a normalization version constant.
- [x] 1.2 Implement separator normalization, whitespace cleanup, safe fallback behavior, and structured year extraction.
- [x] 1.3 Implement categorized token removal for website/domain watermarks, quality labels, HDR labels, video codecs, source labels, platform labels, audio markers, subtitle/language markers, and trailing release groups.
- [x] 1.4 Add table-driven unit tests for movie, TV, Chinese, URL/domain watermark, bracketed watermark, dense technical release, release-group, and empty-result fallback cases.

## 2. Scanner Integration

- [x] 2.1 Replace scanner title cleanup in `internal/library/scan_classify.go` with the shared normalizer while preserving existing movie, TV, season, episode, and multi-episode classification behavior.
- [x] 2.2 Keep year extraction wired to catalog scan artifacts using the shared normalizer result.
- [x] 2.3 Extend scanner metadata evidence payloads to include normalization version and removed tokens with reason labels.
- [x] 2.4 Add or update library scanner tests for noisy movie filenames, noisy TV filenames, Chinese season/episode filenames with embedded website watermarks, and matched-item governance preservation.

## 3. Metadata Search Integration

- [x] 3.1 Update `internal/metadata/service_matcher.go` search title cleanup to use the shared normalizer for filename-derived and title-derived query variants.
- [x] 3.2 Preserve useful fallback query variants from original title, source path base name, parent folder, and TV series folder so matching remains resilient when normalization is overly aggressive.
- [x] 3.3 Add metadata matcher tests proving scanner and matcher remove equivalent release noise and website watermarks.

## 4. Technical Metadata Boundaries

- [x] 4.1 Verify normalization does not populate or override width, height, video codec, audio codec, subtitle, or stream fields from filename tokens.
- [x] 4.2 Add regression coverage or assertions around existing probe-owned technical metadata behavior if integration touches related code paths.

## 5. Verification

- [x] 5.1 Run focused backend tests for title normalization, library scanning, and metadata matching.
- [x] 5.2 Run `go test ./...` from `mibo-media-server/` and address any failures caused by this change.
- [x] 5.3 Confirm OpenSpec status reports the change as apply-ready after tasks are created.
