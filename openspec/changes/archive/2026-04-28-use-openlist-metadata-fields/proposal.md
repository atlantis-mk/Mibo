## Why

OpenList exposes more file metadata than Mibo currently uses, including same-directory related files, provider identity, hashes, object type, creation time, signatures, mount details, and folder display metadata. Mibo can use the safe, provider-neutral portions of that data to improve scan accuracy, reduce extra OpenList calls, and provide better diagnostics without coupling catalog behavior to OpenList internals.

## What Changes

- Capture additional OpenList object metadata in the storage provider contract where it has clear cross-provider value: creation time, object type, sign token, related files, and provider diagnostics.
- Use `/api/fs/get` related-file metadata to discover sibling artwork and sidecar candidates before issuing repeated individual `Get` probes.
- Preserve and expose provider/hash metadata for diagnostics and future governance while avoiding sensitive mount detail leakage in normal user-facing APIs.
- Keep thumbnail artwork behavior as poster-only fallback from `use-openlist-thumb-artwork`; this change focuses on the remaining OpenList fields and their safe use.
- Do not make Mibo depend on OpenList-only fields for core correctness; missing fields must degrade to existing behavior.

## Capabilities

### New Capabilities

- `openlist-metadata-utilization`: Defines how Mibo captures and uses additional OpenList metadata for related-file discovery, diagnostics, identity hints, and safe fallback behavior.

### Modified Capabilities

- `catalog-governance-actions`: Extend diagnostics available to governance/repair flows with provider/hash/source metadata without changing governance action semantics.

## Impact

- Affects backend storage object contract in `mibo-media-server/internal/storage/provider.go`.
- Affects OpenList response parsing in `mibo-media-server/internal/storage/openlist/adapter.go`.
- Affects artwork and sidecar discovery paths in `mibo-media-server/internal/probe` and potentially `internal/library` scanning helpers.
- May add internal diagnostic fields to backend response models, limited to admin/debug contexts.
- Adds backend tests for OpenList parsing, related-file fallback ordering, missing-field compatibility, and sensitive metadata filtering.
