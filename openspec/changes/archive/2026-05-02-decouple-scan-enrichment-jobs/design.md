## Context

Mibo currently treats manual library scanning as a combined synchronization and enrichment workflow. `sync_library` refreshes storage listings, writes catalog and inventory rows, marks missing files, and also enqueues per-item metadata matching and per-file ffprobe jobs while walking the library. Those follow-up jobs are separate job records, but they share the same FIFO queue as scans, so a large scan can leave thousands of probe and match jobs ahead of a later manual scan.

The user-visible problem is that deleting files from OpenList and clicking scan can leave the library page unchanged until the new scan job reaches the front of the queue and completes. Core synchronization should be the fast, high-priority result of a manual scan; metadata matching and technical probing are enrichment work that can complete afterward.

## Goals / Non-Goals

**Goals:**

- Make `sync_library` complete core synchronization without waiting for metadata matching or media probing work.
- Schedule metadata matching and inventory probing as post-scan batch enrichment work.
- Ensure manual scan jobs are not blocked behind older enrichment jobs in the worker queue.
- Preserve existing enrichment outcomes: catalog items still get matched and inventory files still get probed eventually.
- Keep frontend scan APIs unchanged.

**Non-Goals:**

- Changing catalog classification rules, metadata provider behavior, or ffprobe extraction logic.
- Physically deleting missing catalog or inventory records during scan.
- Adding new frontend controls for enrichment queues in this change.
- Replacing the jobs table or introducing an external queue dependency.

## Decisions

### Prioritize sync jobs over enrichment jobs in the existing queue

The worker will claim available jobs using a deterministic priority order rather than pure FIFO. `sync_library`, targeted refresh, listener reconciliation, and projection refreshes should run before metadata matching and probe enrichment. This keeps manual scan results responsive without changing the public scan API or requiring separate worker processes.

Alternative considered: create separate queues or worker binaries for scan and enrichment. That would provide stronger isolation, but it is more operationally expensive and unnecessary for the current single-process app.

### Capture enrichment candidates during scan and enqueue batch jobs after synchronization

The scan loop will collect catalog item IDs and inventory file IDs that need enrichment. After missing cleanup and availability updates complete, it will enqueue one catalog match batch job and one inventory probe batch job for that scan scope. The scan job may still enqueue projection refreshes because projection freshness is part of making the synchronized catalog visible.

Alternative considered: continue enqueuing one job per item/file inline, but lower their priority. This improves scan queue latency but still creates large job table fan-out during the critical scan path.

### Batch jobs fan out or process bounded chunks

Batch enrichment jobs will process IDs in bounded chunks. Catalog matching can run sequentially inside the batch to respect metadata provider rate limits. Inventory probing can reuse the existing probe worker concurrency behavior by either enqueueing bounded probe jobs from the batch or processing a limited number directly with the existing probe service. The implementation should prefer minimal reuse of existing per-item handlers and avoid duplicating matching/probing logic.

Alternative considered: make one huge batch job process all probes directly. That reduces job count but risks long-running jobs that are harder to retry and can monopolize the worker.

### Make scan completion independent from enrichment completion

`sync_library` will be marked completed once storage reconciliation, missing cleanup, availability updates, and projection-refresh enqueueing are successful. Failure in post-scan enrichment must not retroactively fail the scan job. Enrichment jobs should expose their own failures through normal job status.

Alternative considered: make scan wait for batch jobs. That preserves a single all-inclusive scan result but repeats the current latency problem.

## Risks / Trade-offs

- Batch enrichment job fails after scan completes → The library content is still synchronized, and the failed batch can be retried independently.
- Lower-priority enrichment takes longer during repeated scans → This is acceptable because content visibility and deletion cleanup are prioritized; worker status still exposes backlog.
- Duplicate enrichment scheduling across repeated scans → Use unique job keys scoped by library/path and enqueue time, or deduplicate IDs inside batch payloads to avoid waste.
- Large payloads if thousands of IDs are stored in one job → Use bounded batches or job payload filters such as library ID/root path plus a “needs enrichment” status where practical.
- Existing running jobs remain stuck after process restart → This change does not introduce recovery, but priority ordering should apply to newly queued jobs; stale running job recovery can be handled separately if needed.

## Migration Plan

1. Add new batch enrichment job kind constants and worker handlers.
2. Change scan code to collect enrichment candidates and enqueue batch jobs after core sync work.
3. Adjust job claiming order to prioritize synchronization before enrichment.
4. Keep old `match_catalog_item` and `probe_inventory_file` handlers for compatibility with already queued jobs.
5. Deploy without database migration unless the chosen implementation adds fields for priority or enrichment state.
6. Roll back by restoring FIFO claim order and inline per-item enqueueing; old and new enrichment jobs should remain independently retryable.

## Open Questions

- Should inventory probe batch jobs directly call the probe service or fan out existing `probe_inventory_file` jobs in bounded chunks?
- Should repeated manual scans deduplicate active enrichment batches by library ID only, or by library ID plus root path?
