## Why

Media library freshness currently depends on manual or scheduled scans plus an external storage-event endpoint. This misses the desired behavior for local and OpenList-backed libraries: file creation, deletion, modification, and movement should be detected automatically while preserving the scanner as the source of truth for catalog and inventory writes.

## What Changes

- Add a persistent storage index that records the last observed provider snapshot for library paths, including path, directory flag, size, modified time, stable identity, hash evidence, and observation status.
- Add a diff planner that compares provider observations against the storage index and produces bounded refresh plans for create, update, delete, move, rename, and uncertain changes.
- Add provider-specific change sources that feed the same planning path:
  - Local libraries use file-system events plus periodic reconciliation.
  - OpenList libraries use polling snapshot diffs plus periodic reconciliation.
  - Existing `POST /api/v1/storage-events` remains supported as an external hint source.
- Route all planned changes through existing listener jobs, targeted refresh jobs, scanner writes, probe jobs, and catalog projection refreshes instead of directly mutating catalog state from events.
- Replace the earlier watcher-only design with storage-index-driven detection and planning; watcher events become hints, not the authoritative state model.

## Capabilities

### New Capabilities
- `storage-change-indexing`: Persistent storage snapshots, diff planning, and provider change detection for automatically refreshing library inventory and catalog availability.

### Modified Capabilities
- None.

## Impact

- Backend database schema gains storage index and change planning state tables.
- Backend services gain storage index, diff planning, and change source orchestration in domain packages under `mibo-media-server/internal/`.
- Local storage may require a file-system watcher dependency and local file identity extraction where supported.
- OpenList storage gains polling-based observation using existing `List`/`Get` provider capabilities without modifying upstream OpenList.
- Worker processing gains new jobs or extends listener jobs to run observations, reconcile storage indexes, and enqueue targeted refreshes.
- Admin or settings APIs may expose listener/index status for debugging and rollout visibility.
