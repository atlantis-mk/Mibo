# Phase 2: Library & Async Sync Foundation - Research

**Gathered:** 2026-04-21
**Status:** Ready for planning

---

## Research Question

**"What do I need to know to PLAN this phase well?"**

Phase 2 delivers async library scanning and background job infrastructure so admins can trigger scans without blocking requests and configure scheduled refreshes.

---

## Existing Infrastructure

### Job Queue System

The codebase already has a production-quality job queue:

- **`internal/jobs/service.go`**: Complete job lifecycle management
  - `Enqueue(ctx, kind, payload)` — enqueue a job
  - `EnqueueUnique(ctx, kind, jobKey, payload)` — deduplicated enqueue (prevents duplicate scans via `job_key`)
  - `ClaimNext(ctx)` — worker claims next available job (3 retry attempts, uses transaction for atomicity)
  - `Complete(ctx, jobID)` — mark job succeeded
  - `Fail(ctx, jobID, err)` — mark job failed with error message
  - `Retry(ctx, jobID)` — re-queue a failed job for retry
  - `List(ctx, limit)` — list recent jobs

- **Job statuses**: `queued → running → completed` or `queued → running → failed`
- **Deduplication**: `job_key` field prevents duplicate scans (same library won't queue twice)

### Worker System

- **`internal/worker/worker.go`**: Background worker that polls every 2 seconds
  - `Run(ctx)` — infinite loop with ticker
  - `RunOnce(ctx)` — single poll cycle
  - Handles three job types: `sync_library`, `match_media_item`, `probe_media_file`

### Library Scanning

- **`internal/library/scan.go`**: Full scan implementation
  - `QueueLibraryScan(ctx, libraryID)` — enqueues a `sync_library` job
  - `RunSyncLibrary(ctx, job)` — updates library status to "syncing", runs scan, sets "active" or "error"
  - **Merge behavior already implemented**:
    - Add new items (upsert)
    - Update changed items (fingerprint-based detection)
    - Soft-delete missing items (sets `deleted_at`, `status="missing"`)
  - Triggers downstream jobs: `match_media_item` and `probe_media_file` for pending items

### HTTP API Endpoints (all exist)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/libraries` | List libraries |
| `POST` | `/api/v1/libraries` | Create library (auto-queues scan) |
| `GET` | `/api/v1/libraries/{id}` | Get library details |
| `POST` | `/api/v1/libraries/{id}/scan` | Queue manual scan (returns 202 + job) |
| `DELETE` | `/api/v1/libraries/{id}` | Delete library |
| `GET` | `/api/v1/jobs` | List recent jobs |
| `POST` | `/api/v1/jobs/{id}/retry` | Retry failed job |

---

## What's Missing for Phase 2

### Backend Gaps

1. **Global refresh interval configuration**
   - No settings endpoint for `refresh_interval_hours`
   - No `settings` table or service for system-wide config
   - Need: `PUT /api/v1/settings/scan` and `GET /api/v1/settings/scan`

2. **Scheduled scan trigger mechanism**
   - Worker currently only processes on-demand jobs
   - No cron-style scheduled scan based on `refresh_interval_hours`
   - Need: background ticker that enqueues scans for all `active` libraries when interval elapses

3. **Library status in API response**
   - `GET /api/v1/libraries/{id}` returns `database.Library` which has `Status` field
   - But frontend doesn't display it as a badge
   - Status values: `pending | syncing | active | error`

4. **Jobs filtering**
   - `GET /api/v1/jobs` doesn't support `?status=failed&kind=sync_library`
   - Need: query parameter filtering on `status` and `kind`

### Frontend Gaps

1. **Library card status badge**
   - Need to display `syncing/active/error` status on library cards
   - Source drawer already works (`source-drawer.tsx`)

2. **Jobs list view**
   - New page/section to show all jobs with:
     - Status indicators (queued/running/completed/failed)
     - Filtering by status and kind
     - Retry button for failed jobs
   - Accessible from settings area

3. **Global refresh interval UI**
   - Settings panel for system-wide refresh interval
   - Input for hours between automatic scans

---

## Integration Points

### Database Schema

Library has these relevant fields (from `database.Library`):
- `Status string` — `pending | syncing | active | error`
- `ScannerEnabled bool` — could control per-library scan toggle
- `LastScannedAt *time.Time` — timestamp of last scan

### Storage Provider Abstraction

`StorageProvider` interface already in place (`storage/provider.go`):
- `List(ctx, req)` — paginated directory listing
- `ResolveStorage(ctx, req)` — resolve path to storage object
- `Capabilities(ctx)` — returns provider capabilities

This is the abstraction that Phase 1 established and Phase 2 builds on.

### Frontend API Client

- `mibo-api.ts` provides `createMiboApi()` factory
- Library endpoints already have frontend bindings
- Need: Jobs endpoints and status display

---

## Key Technical Decisions

### Scan Behavior (Already Decided)
- Merge on rescans: add new, update changed, soft-delete missing (preserves playback history)
- Fingerprint = `path:size:modified` — detects file changes
- `match_media_item` and `probe_media_file` jobs queued automatically for pending items

### Jobs Deduplication
- `EnqueueUnique` with `job_key` prevents duplicate scans
- Same library scan won't queue twice even if user clicks "Scan" rapidly

### Status Lifecycle
```
pending (created) → syncing (scan starts) → active (scan succeeds) | error (scan fails)
```

### Scheduled Refresh Architecture
- Global interval stored in settings table
- Background worker checks all `active` libraries on interval
- Uses `EnqueueUnique` so manual and scheduled scans deduplicate properly

---

## Validation Architecture

Phase 2 should be verifiable by:

1. **Unit tests**: `go test ./internal/worker -run TestRunOnceProcessesSyncLibraryJob`
2. **Integration**: Create library → verify scan job queued → verify library status transitions `pending → syncing → active`
3. **UI**: Library card shows status badge; Jobs list shows all job types with retry capability

---

## Risk Factors

1. **Large libraries**: Deep directory trees with thousands of files may cause scan jobs to run long
   - Mitigation: Worker processes one job at a time; large scans don't block other work
2. **OpenList latency**: Scanning through OpenList HTTP API may be slow
   - Mitigation: paginated listing (1000 items per page), 2-second worker poll gives backpressure
3. **Concurrent scans**: Multiple libraries scanning simultaneously could strain OpenList
   - Mitigation: Worker processes one job at a time; `EnqueueUnique` prevents duplicate scans per library

---

## Recommendations for Planning

1. **Backend first**: Add settings service, scan scheduling ticker, then jobs filtering
2. **Frontend second**: Status badges, Jobs list, settings panel
3. **Test the happy path**: Create library → auto-scan → verify status transitions → verify media items created
4. **Test error path**: Trigger failed scan (e.g., invalid path) → verify `error` status → verify retry works
