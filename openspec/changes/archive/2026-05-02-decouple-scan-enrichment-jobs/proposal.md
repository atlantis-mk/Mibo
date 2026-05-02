## Why

Manual library scans should make source changes visible quickly, especially when files were deleted from OpenList. Today a scan queues large numbers of metadata matching and media probing jobs inline, and the shared FIFO worker queue can delay later scans behind thousands of enrichment jobs.

## What Changes

- Split scan-time enrichment scheduling from the core library synchronization path.
- Make `sync_library` responsible for refreshing storage, reconciling catalog/inventory rows, marking missing files, updating availability, and scheduling projection refreshes.
- Add post-scan batch jobs for catalog metadata matching and inventory media probing so enrichment can run independently after synchronization completes.
- Adjust worker dispatch so library synchronization is not blocked behind older probe or metadata enrichment work.
- Preserve existing metadata matching and probing behavior as eventual post-scan enrichment, not as a prerequisite for scan completion.

## Capabilities

### New Capabilities

### Modified Capabilities

- `media-graph-scanner`: Library scanning will complete core synchronization before post-scan enrichment and must not be blocked by queued probe or metadata match jobs.

## Impact

- Backend job orchestration in `mibo-media-server/internal/jobs`, `internal/worker`, and `internal/library`.
- Scan workflow in `mibo-media-server/internal/library/scan_run.go` and related catalog scan helpers.
- Existing job kinds for `match_catalog_item` and `probe_inventory_file` may be wrapped by new batch job kinds or scheduled through a new post-scan enqueue phase.
- Frontend scan entry points can remain unchanged, but manual scan results should become visible sooner after the scan job completes.
