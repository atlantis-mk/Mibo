## Why

The current scanner still decides metadata identity while materializing individual files, so scan order, local directory shape, and already-created metadata rows can affect whether same-work resources become versions or duplicates. Because Mibo is still in development and local data can be reset, this is the right time to replace the heuristic, file-by-file recognition path with a manifest-driven resolver and remove the superseded code instead of layering another matcher on top.

## What Changes

- **BREAKING**: Replace file-by-file catalog materialization with an inventory-first recognition pipeline: scans record file facts and signals, then build a recognition manifest before creating metadata/resource links.
- **BREAKING**: Stop using content-shape, path-tree work groups, and same-metadata sibling matching as independent final identity decision engines; keep only the useful parsers/evidence extraction or delete the obsolete paths.
- Introduce a recognition manifest containing work candidates, episode candidates, playable resource candidates, edition/variant evidence, sidecar evidence, hash evidence, conflicts, and resolver decisions.
- Introduce a deterministic identity resolver that evaluates the complete manifest, applies explicit conflict rules, and decides which candidates can auto-create or reuse `MetadataItem`, `Resource`, and `ResourceMetadataLink` records.
- Separate canonical work identity from resource variant identity so copies, encodes, editions/cuts, multi-part resources, multi-episode resources, trailers, extras, and samples are not all collapsed into a single generic version concept.
- Make scan results idempotent and order-independent by deriving stable candidate keys from file facts, sidecars, path context, and normalized signals before materialization.
- Preserve fast scanning by keeping remote metadata, ffprobe, and content hashing out of the initial recognition pass; asynchronous enrichment may trigger resolver re-runs for affected candidates.
- Add explicit cleanup tasks to remove replaced scanner/materializer branches, duplicate tests, legacy fallback creation, and unused helper types so the codebase does not carry two recognition architectures.

## Capabilities

### New Capabilities
- `recognition-manifest-resolver`: Defines manifest construction, identity resolution, conflict handling, idempotent materialization, and cleanup expectations for the new recognition architecture.

### Modified Capabilities
- `source-first-auto-classification`: Replace direct scanner-owned metadata linking with manifest generation and resolver-owned materialization.
- `media-graph-scanner`: Change scan output from immediate per-file catalog writes to inventory facts plus recognition candidates that materialize through the resolver.
- `fast-video-classification`: Keep filename/path signal extraction in the fast path while removing final identity decisions from fast classification helpers.
- `catalog-metadata-governance`: Require resolver decisions, conflicts, and user corrections to be traceable and reusable without preserving legacy matching engines.
- `sidecar-metadata-files`: Treat sidecar metadata as high-priority resolver evidence rather than a separate direct-link path.

## Impact

- Backend scanner and library services under `mibo-media-server/internal/library`, especially scan run, materialization, content-shape assignment, path-tree work grouping, same-metadata sibling matching, and catalog scan linking code.
- Database schema may add manifest/candidate/decision tables or replace existing scanner decision tables if that reduces maintenance cost; development reset is acceptable.
- Projection refresh and catalog read models remain resource/metadata based, but their inputs come from resolver materialization instead of scanner-side link helpers.
- Governance merge/split and correction endpoints must operate on resolver decisions and generate durable resolver rules, not patch around obsolete scanner heuristics.
- Existing OpenSpec changes that introduced content-shape, path-tree, file-signal, and sibling-matching behavior become implementation history; their surviving code must be justified as evidence extraction or removed.
