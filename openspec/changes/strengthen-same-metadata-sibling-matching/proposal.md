## Why

Current scans are better at deciding whether grouped media is movie or series content, but they still do a weak job deciding whether sibling resources belong to the same metadata identity. This causes duplicate movie or episode identities, weak cross-source reuse, and unnecessary unresolved organizing entries when stronger file-level identity such as `md5` could safely anchor the match.

## What Changes

- Add a sibling-matching capability that runs after work-group classification and decides whether resources belong to the same canonical movie or episode metadata identity.
- Promote file `md5` to a strong cross-source identity signal so the system can quickly recognize the same binary media file even when titles, folders, or sources differ.
- Distinguish canonical work matching from version-trait matching so the scanner can merge alternate encodes, cuts, and release variants under one metadata identity without collapsing unrelated works.
- Isolate extras, samples, trailers, and other supplemental files from automatic primary/version merging.
- Keep weak same-title candidates unresolved and browse-visible until stronger evidence such as provider identity, sidecar identity, episode tuple, or `md5` is available.

## Capabilities

### New Capabilities
- `same-metadata-sibling-matching`: classify sibling and cross-source resources as same-work, same-version-group, same-episode, supplemental, or conflict outcomes before creating or updating metadata links.

### Modified Capabilities
- `source-first-auto-classification`: extend automatic classification so post-group matching can use `md5`, provider identity, sidecar identity, and version traits without blocking fast-path scanning.
- `library-detail-browsing`: ensure organizing entries upgrade cleanly when accepted sibling matches attach new resources to an existing metadata identity or version list.
- `sidecar-metadata-files`: treat sidecar title/year, episode tuple, and external identity hints as primary same-metadata matching evidence in addition to movie-versus-series evidence.

## Impact

- Affected backend areas: `mibo-media-server/internal/library`, `internal/inventory`, `internal/catalog`, and projection refresh paths.
- Affected data flow: scan classification, resource-to-metadata linking, browse upgrade behavior, and asynchronous rematch behavior when file `md5` becomes available.
- Affected product behavior: duplicate organizing cards should reduce for same-media resources, while weak candidates remain conservative and reviewable.
