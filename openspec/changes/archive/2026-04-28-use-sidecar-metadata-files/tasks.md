## 1. Scanner Sidecar Discovery

- [x] 1.1 Add sidecar data structures for discovered subtitle and metadata files in `internal/library`.
- [x] 1.2 Build a per-directory sidecar index from `listAllDirectoryObjects` results during `walkDirectory`.
- [x] 1.3 Implement deterministic basename and safe folder-level association rules for `.srt`, `.ass`, `.nfo`, and `.json` files.
- [x] 1.4 Add tests for supported extensions, unsupported extension ignores, basename matching, and ambiguous folder-level metadata skips.

## 2. Metadata Parsing And Hints

- [x] 2.1 Implement bounded reads for sidecar metadata content through the storage provider path available to scans.
- [x] 2.2 Parse JSON sidecars for title, original title, year, media type, series title, season number, episode number, and external IDs.
- [x] 2.3 Parse NFO sidecars with XML-first extraction and conservative fallback handling for high-confidence fields.
- [x] 2.4 Merge parsed sidecar hints into catalog scan artifacts before catalog item writes while preserving existing governance rules.
- [x] 2.5 Add tests for movie JSON hints, episode NFO hints, malformed metadata, and oversized sidecar skips.

## 3. Evidence Recording

- [x] 3.1 Extend scanner metadata payloads to include associated subtitle sidecars with path, extension, and association source.
- [x] 3.2 Extend scanner metadata payloads to include metadata sidecar parse status and extracted local hint fields.
- [x] 3.3 Ensure subtitle sidecar contents are not parsed for classification or stored in scanner evidence.
- [x] 3.4 Add tests verifying sidecar evidence appears on movie and episode metadata sources.

## 4. Verification

- [x] 4.1 Run focused library scanner tests for sidecar discovery and catalog scan writes.
- [x] 4.2 Run `go test ./internal/library ./internal/probe ./internal/catalog` from `mibo-media-server/`.
- [x] 4.3 Update any failing tests or fixtures caused by intentional scanner metadata payload changes.
