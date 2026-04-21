# Phase 2: Library & Async Sync Foundation - Context

**Gathered:** 2026-04-21
**Status:** Ready for planning

<domain>
## Phase Boundary

Administrators can connect storage-backed libraries and trust scans/refreshes to run asynchronously without degrading interactive requests. Admins can add media sources (local/NAS/cloud), create libraries bound to sources, trigger scans that queue and process in background, and configure global scheduled refreshes.

</domain>

<decisions>
## Implementation Decisions

### Scan Status UX
- **D-01:** Hybrid approach: library status badge (syncing/active/error) on library cards for quick feedback, plus a Jobs list view accessible from settings for detailed monitoring and retry of failed jobs.

### Scheduled Refresh
- **D-02:** Global refresh interval — one system-wide refresh interval applies to all libraries, not per-library schedules. Simpler config surface while satisfying LIBR-04.

### Scan Behavior
- **D-03:** Merge behavior on rescans — add new items, update changed items, soft-delete items that no longer exist on disk. Preserves playback history. Full rebuild available as explicit admin action only.

### Media Source Config
- **D-04:** Local storage sources need only provider + root path. No additional V1 config (exclude paths, scan depth limits, watch folders). Add if real needs emerge in later phases.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements
- `.planning/ROADMAP.md` — Phase 2 goal, success criteria, and scope boundary
- `.planning/REQUIREMENTS.md` — LIBR-01, LIBR-02, LIBR-03, LIBR-04, CATA-06 requirements
- `.planning/PROJECT.md` — project-level architecture constraints and product direction

### Prior phase context
- `.planning/phases/01-access-platform-boundary/01-CONTEXT.md` — Phase 1 decisions (access boundary, setup flow, StorageProvider abstraction)

### Architecture
- `docs/media-architecture/improved-architecture.md` — keeps OpenList at storage edge and mibo-media-server as the media/business core
- `AGENTS.md` — repo-specific rules for frontend routing and setup/auth alignment

</canonical_refs>

<codebase_context>
## Existing Code Insights

### Reusable Assets
- `mibo-media-server/internal/worker/worker.go`: Existing worker runner that polls jobs every 2 seconds and handles `sync_library`, `match_media_item`, `probe_media_file` job types
- `mibo-media-server/internal/jobs/service.go`: Job queue with Enqueue, EnqueueUnique, Claim, Complete, Fail, Retry — deduplication via `job_key` prevents duplicate scans
- `mibo-media-server/internal/library/service.go`: Library and MediaSource CRUD — CreateLibrary already queues a scan job on creation
- `mibo-media-server/internal/library/scan.go`: Scan logic with merge behavior (upsert items, cleanup missing), fingerprint-based change detection
- `mibo-media-server/internal/storage/provider.go`: StorageProvider interface (List, Get, Link, ResolveStorage, Capabilities) — abstraction already in place
- `web/src/features/app/components/source-drawer.tsx`: Source creation UI with OpenList connection testing
- `web/src/features/app/components/library-drawer.tsx`: Library creation UI with source selection and path picking

### Established Patterns
- Async job pattern: POST returns 202 + job, worker processes async, status updated on library record
- Jobs use `EnqueueUnique` with job_key for deduplication (same library scan won't queue twice)
- Library status field: `pending | syncing | active | error`

### Integration Points
- `mibo-media-server/internal/httpapi/router.go`: `POST /api/v1/libraries/{id}/scan` (queue scan), `GET /api/v1/jobs` (list jobs), `POST /api/v1/jobs/{id}/retry` (retry failed)
- Frontend needs: library status badge on library cards, Jobs list view with retry capability
- Worker scheduling: polls every 2 seconds, configurable via `WorkerConfig.PollInterval`

</codebase_context>

<specifics>
## Specific Ideas

- Jobs list should show all job types (scan, match, probe) with filtering by status
- Global refresh interval config likely lives in settings panel (System Settings)
- Library status badge should reflect current status: "pending" (never scanned), "syncing" (scan in progress), "active" (last scan succeeded), "error" (last scan failed)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 02-library-async-sync-foundation*
*Context gathered: 2026-04-21*
