## 1. Storage Metadata Contract

- [x] 1.1 Extend `mibo-media-server/internal/storage.Object` with provider-neutral metadata fields for `Created`, `ObjectType`, `Sign`, `Related`, and sanitized provider diagnostics.
- [x] 1.2 Keep `Sign` and provider diagnostics out of normal JSON output unless explicitly intended for internal/debug use.
- [x] 1.3 Add helper behavior for cloning or sanitizing provider metadata so sensitive fields are not propagated accidentally.

## 2. OpenList Adapter Parsing

- [x] 2.1 Update OpenList list response parsing to capture `created`, `type`, `sign`, safe metadata indicators, and existing provider/hash/thumb fields.
- [x] 2.2 Update OpenList get response parsing to capture `created`, `type`, `sign`, safe metadata indicators, and `related` entries.
- [x] 2.3 Reconstruct full storage paths for related objects from the requested object's parent path and related object names.
- [x] 2.4 Add OpenList adapter tests covering list metadata, get metadata, related object path reconstruction, and missing optional fields.

## 3. Related-File Discovery

- [x] 3.1 Refactor sibling artwork lookup in `internal/probe/artwork.go` to accept related storage objects as an optional first-pass candidate source.
- [x] 3.2 Use related objects from the media file `Get` response before issuing individual sibling artwork `Get` calls.
- [x] 3.3 Preserve current direct candidate path lookup when related files are absent, incomplete, stale, or non-matching.
- [x] 3.4 Ensure related-file optimization does not change artwork priority: non-generated selected artwork, sibling artwork, provider thumbnail poster fallback, then ffmpeg extraction.

## 4. Safe Diagnostics

- [x] 4.1 Identify the existing governance/admin response path best suited for sanitized provider diagnostics.
- [x] 4.2 Add safe diagnostics containing storage provider name, provider-reported driver identity, available hash keys, object type hints, and optional metadata presence indicators.
- [x] 4.3 Ensure raw `sign`, `mount_details`, write/upload flags, and auth-bearing URLs are not exposed in normal catalog, item, playback, or frontend responses.
- [x] 4.4 Add tests proving sensitive OpenList internals are filtered while safe diagnostics remain available where intended.

## 5. Verification

- [x] 5.1 Run focused backend tests for `internal/storage/openlist`.
- [x] 5.2 Run focused backend tests for probe artwork related-file behavior.
- [x] 5.3 Run focused backend tests for governance/admin diagnostics if diagnostics are implemented in an existing package.
- [x] 5.4 Run `go test ./...` from `mibo-media-server/`.
- [x] 5.5 Manually inspect an OpenList-backed item with related artwork files to confirm fewer fallback `Get` probes and unchanged selected artwork ordering.
