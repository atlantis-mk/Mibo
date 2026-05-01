## Why

The current metadata configuration model mixes runtime provider selection, library overrides, legacy booleans, and a synthetic `migrated-default-local-only` profile into overlapping data paths. That makes local sidecar metadata feel like a special mode instead of a first-class metadata source, and it leaves library metadata behavior difficult to reason about or migrate safely.

## What Changes

- Introduce a first-class metadata runtime model where provider instances, including a system-managed local scan provider, are the only executable metadata sources.
- Replace library-level metadata booleans and priority remnants with an explicit per-library metadata strategy built from ordered provider instances per stage.
- Reposition metadata profiles as optional reusable templates instead of special runtime-only objects that can short-circuit provider selection.
- Remove the long-term need for `migrated-default-local-only`, `local_only`, `force_local_only`, and legacy library metadata provider-enable/priority flags.
- Ensure scanner-discovered `.nfo` and `.json` evidence remains available to metadata operations through the local scan provider without requiring manual provider configuration.
- **BREAKING**: library metadata management APIs and UI flows will move from profile-plus-policy semantics to strategy/template semantics.

## Capabilities

### New Capabilities
- `metadata-provider-runtime-model`: Defines system-managed metadata provider types, the built-in local scan provider instance, per-stage provider capabilities, library metadata strategies, and optional template application.

### Modified Capabilities
- `library-source-policies`: Replace legacy library metadata policy booleans and provider priority semantics with explicit library metadata strategy reads and writes in library management APIs and UI.
- `sidecar-metadata-files`: Clarify that parsed local sidecar metadata evidence is recorded for safe reuse by the built-in local scan metadata provider instead of relying on special local-only profile behavior.

## Impact

- Backend settings and metadata runtime in `mibo-media-server/internal/settings`, `internal/metadata`, `internal/library`, and related HTTP handlers.
- Database models and migrations for metadata provider instances, library metadata strategy persistence, and retirement of legacy compatibility fields.
- Frontend metadata settings and library settings flows in `web/src/features/settings` and `web/src/lib/mibo-api.ts`.
- Metadata governance provenance, migration defaults, and operator-facing configuration semantics.
