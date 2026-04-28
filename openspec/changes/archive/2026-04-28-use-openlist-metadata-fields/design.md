## Context

OpenList `/api/fs/list` and `/api/fs/get` expose object metadata beyond the fields Mibo currently consumes. The useful fields fall into three groups: stable provider-neutral metadata (`created`, `type`, `hash_info`, `provider`), discovery helpers (`related`, `thumb`, `raw_url`), and OpenList-specific or potentially sensitive metadata (`sign`, `mount_details`, folder `readme`/`header`, write/upload flags). Mibo already has storage-provider abstractions and catalog/probe fallback flows, so the change should extend those boundaries rather than adding OpenList-specific branches in catalog logic.

## Goals / Non-Goals

**Goals:**

- Capture OpenList metadata in provider-neutral storage objects when it has clear value outside OpenList.
- Use `related` files from `/api/fs/get` to reduce repeated sibling `Get` probes for artwork and future sidecar discovery.
- Preserve missing-field compatibility so other providers and older OpenList responses continue to work.
- Keep sensitive or unstable OpenList details out of normal catalog and frontend contracts by default.
- Make provider/hash/source diagnostics available to bounded admin or governance workflows.

**Non-Goals:**

- This change does not add upload, write, or direct-upload features.
- This change does not expose raw `mount_details` or signed path tokens to general frontend consumers.
- This change does not replace Mibo's extension-based media classification with OpenList `type` alone.
- This change does not make OpenList metadata authoritative over TMDB/manual metadata or existing selected artwork.
- This change does not cache or proxy remote provider URLs.

## Decisions

### Extend `storage.Object` with safe metadata fields

Add provider-neutral fields for `Created`, `ObjectType`, `Sign`, `Related`, and an opaque `ProviderMetadata`/diagnostic map with allowlisted keys only. `HashInfo`, `Provider`, `RawURL`, and `ThumbnailURL` remain first-class fields. `Sign` should be parsed for completeness but must not be persisted or emitted through normal catalog APIs unless a future signed-link flow explicitly needs it.

Alternative considered: store every OpenList field in a raw JSON blob. That is flexible but leaks provider-specific shape into business logic and makes sensitive-field filtering harder to enforce.

### Represent `related` as storage objects

Parse `/api/fs/get.related` into `[]storage.Object` so artwork and sidecar discovery can use the same sibling candidate logic regardless of whether siblings came from a directory list, related metadata, or direct `Get` calls. Related object paths must be reconstructed from the parent directory and related object name because OpenList related entries carry object fields but not full Mibo storage paths.

Alternative considered: expose related as raw OpenList structs. That would make OpenList-specific logic leak into probe/library packages and would block future providers from offering similar related-file hints.

### Use related files as an optimization, not a correctness dependency

Sibling artwork lookup should first inspect the already available related set when present. If no matching related object exists, or related is absent, Mibo must keep the existing direct candidate `Get` fallback. This avoids regressions for providers that omit `related`, list incomplete related entries, or return stale metadata.

Alternative considered: replace direct sibling probing entirely with related metadata. That is faster but too risky because related availability is provider and endpoint dependent.

### Treat OpenList object `type` as an auxiliary hint

Persist or carry `ObjectType` as a diagnostic/classification hint, but keep Mibo's extension-based media classification as the source of truth for scan inclusion. The hint can be used in tests, debugging, or future classification fallback only when extensions are ambiguous.

Alternative considered: trust OpenList `type` for media classification. That would couple Mibo's media model to OpenList utility constants and could diverge from Mibo's own supported media rules.

### Filter sensitive provider diagnostics

Provider diagnostics should be allowlisted and bounded: provider name, hash keys, object type, created/modified timestamps, and whether optional fields were present are safe. Raw `mount_details`, sign tokens, auth-bearing URLs, and write/upload flags should not be exposed in normal item responses. Admin/debug endpoints may expose sanitized summaries only.

Alternative considered: expose all OpenList metadata to the frontend for maximum visibility. That increases accidental credential/path leakage risk and creates unstable UI dependencies on OpenList internals.

## Risks / Trade-offs

- Related metadata may be missing or stale -> keep current direct `Get` fallback and test both paths.
- Additional storage fields can broaden API contracts unintentionally -> keep new fields internal unless explicitly needed in admin/debug output.
- Sign tokens and mount details may contain sensitive data -> parse sparingly, do not persist raw values, and add tests for sanitization.
- Object type values are OpenList-defined integers -> document them as hints and avoid hard requirements on exact values outside adapter tests.
- Parsing related files increases response handling complexity -> centralize mapping in the OpenList adapter and reuse existing storage object shape.

## Migration Plan

No database migration is required for the first implementation. Existing inventory rows already store provider and hash data where relevant. New metadata fields are used in-memory for discovery and diagnostics; any later persistence should be proposed separately with a migration and retention policy.

Rollback is code-level: remove the additional parsing and related-file lookup, and the existing direct `Get` fallback continues to work.

## Open Questions

- Should sanitized provider diagnostics be surfaced through an existing governance endpoint or a dedicated admin/debug endpoint?
- Should sidecar subtitle/NFO discovery be implemented in this change or planned as a follow-up after related-file plumbing exists?
- Should OpenList `created` be stored in inventory records in the future, or remain an in-memory/provider diagnostic field for now?
