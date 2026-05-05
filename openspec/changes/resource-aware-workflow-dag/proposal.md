## Why

Library ingest currently runs through a global job queue where most non-probe work is effectively serialized, so one large library can block newly added libraries and downstream processing. A resource-aware workflow DAG is needed now to improve throughput, fairness, observability, cancellation, and recovery as scans, catalog materialization, probing, and metadata matching become larger and more expensive.

## What Changes

- Introduce workflow runs for library ingest and refresh work, scoped to a library and reason.
- Replace monolithic library scan execution with workflow tasks connected by explicit dependencies.
- Add a scheduler that selects ready tasks by dependency state, per-library safety, priority, fairness, FIFO order, and resource budgets.
- Add task leases and recovery so crashed workers do not leave work permanently running.
- Add configurable resource budgets for IO, database writes, ffprobe, metadata provider calls, OpenList HTTP, and CPU-heavy work.
- Split large scan and enrichment operations into bounded batches so small libraries and high-priority tasks are not starved by large libraries.
- Preserve existing user-facing scan triggers while changing backend execution to workflow-backed orchestration.

## Capabilities

### New Capabilities
- `resource-aware-workflow-scheduler`: Defines workflow runs, DAG tasks, dependency-aware scheduling, resource budgets, per-library safety, fairness, leases, cancellation, and recovery.

### Modified Capabilities
- `media-graph-scanner`: Library scans become workflow-backed and batch-oriented instead of executing the full scan as one global job.
- `fast-skeleton-library-ingest`: Initial library ingest must return quickly after creating a workflow run while background workflow tasks populate catalog and inventory data.
- `metadata-operation-pipeline`: Metadata matching work must be schedulable under resource budgets and dependency constraints rather than relying only on globally serialized jobs.
- `detailed-video-technical-specs`: Video probing work must participate in workflow scheduling while retaining ffprobe concurrency controls.
- `storage-change-indexing`: Storage change refresh work must map to scoped workflow runs or tasks without blocking unrelated libraries.

## Impact

- Backend database schema: new workflow run/task/dependency/lease/resource budget tables or equivalent migrations.
- Backend services: `internal/jobs`, `internal/worker`, `internal/library`, `internal/catalog`, `internal/probe`, `internal/metadata`, `internal/listener`, `internal/schedule`.
- Runtime behavior: scans, refreshes, materialization, probing, metadata matching, and cleanup become dependency-aware and resource-limited.
- Configuration: new worker scheduler/resource budget settings with safe defaults for SQLite and local development.
- HTTP/API: existing scan and job endpoints should remain compatible, with workflow visibility added or mapped where useful.
- Tests: scheduler selection, dependency unlocking, per-library mutual exclusion, fairness, lease recovery, cancellation, and existing scan behavior.
