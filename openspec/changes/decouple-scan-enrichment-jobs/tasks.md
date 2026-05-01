## 1. Job Priority

- [x] 1.1 Add deterministic job priority ordering so sync, targeted refresh, listener reconciliation, and projection refresh jobs are claimed before match/probe enrichment jobs.
- [x] 1.2 Update job service tests to verify a queued `sync_library` job is claimed before older queued `probe_inventory_file` and `match_catalog_item` jobs.
- [x] 1.3 Keep FIFO ordering within the same priority class.

## 2. Post-Scan Enrichment Jobs

- [x] 2.1 Add batch job kind constants for catalog matching and inventory probing enrichment.
- [x] 2.2 Define compact payloads for enrichment batches, including library ID, optional root path, and bounded ID lists or scoped selection criteria.
- [x] 2.3 Add worker handlers that process catalog match batches through existing metadata matching logic.
- [x] 2.4 Add worker handlers that process inventory probe batches through existing probe logic or bounded fan-out to existing probe jobs.
- [x] 2.5 Keep existing per-item `match_catalog_item` and `probe_inventory_file` handlers working for already queued jobs.

## 3. Scan Workflow

- [x] 3.1 Change `sync_library` scan state to collect catalog item IDs and inventory file IDs that need enrichment instead of enqueueing per-item enrichment inline.
- [x] 3.2 Enqueue post-scan catalog match and inventory probe batches only after core synchronization, missing cleanup, availability updates, and projection refresh scheduling succeed.
- [x] 3.3 Ensure enrichment enqueue failures are reported consistently without corrupting completed catalog/inventory reconciliation.
- [x] 3.4 Preserve existing projection refresh enqueueing so browse APIs see scan results promptly.

## 4. Verification

- [x] 4.1 Add or update backend tests proving deleted files become missing during `sync_library` completion without waiting for match/probe work.
- [x] 4.2 Add or update worker tests proving enrichment batch failure does not change a completed scan job to failed.
- [x] 4.3 Run `go test ./internal/jobs ./internal/worker ./internal/library` from `mibo-media-server/`.
- [x] 4.4 Manually verify that clicking scan on an OpenList-backed library queues and completes `sync_library` ahead of existing enrichment backlog.
