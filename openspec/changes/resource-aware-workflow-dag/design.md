## Context

Mibo backend work currently uses `internal/jobs` plus `internal/worker.Runner`. Jobs are stored in a global `jobs` table, selected by kind priority and FIFO, then dispatched to library, catalog, probe, metadata, listener, and schedule services. Only single-file ffprobe work has a dedicated concurrency path; most scan, materialization, projection, matching, cleanup, and listener refresh work is globally serialized.

This protects SQLite and same-library writes, but it underuses independent resources. A large library scan can block a later library scan and its downstream tasks even when the libraries do not share data. The existing scan path also embeds a multi-stage workflow inside `sync_library`: scan storage, reconcile inventory/catalog skeletons, queue projection refresh, queue materialization, queue probe batches, and queue metadata matching. The implicit workflow makes progress, fairness, cancellation, and recovery difficult.

## Goals / Non-Goals

**Goals:**
- Move library ingest and refresh work to explicit workflow runs and DAG tasks.
- Allow unrelated libraries to progress concurrently without permitting conflicting same-library stages.
- Schedule by dependency readiness, FIFO fairness, priority, and resource budgets rather than only by global job kind.
- Split large work into bounded batches so one large library cannot monopolize workers.
- Preserve existing scan triggers and user-facing behavior while replacing backend orchestration.
- Add lease-based recovery for crashed workers and cancellation that applies to an entire workflow run.
- Keep safe defaults for SQLite and local development while allowing higher concurrency on larger deployments.

**Non-Goals:**
- Replacing storage providers or scanner classification semantics.
- Rewriting metadata provider behavior beyond scheduler integration and resource limiting.
- Changing frontend browse contracts as part of the scheduler migration.
- Making all tasks fully parallel; dependency order and library safety remain required.
- Introducing an external queue service such as Redis, NATS, or Kafka for the first implementation.

## Decisions

### Decision: Add `internal/workflow` Instead Of Expanding `internal/jobs`

Create a new backend domain package for workflow orchestration. It owns workflow run/task models, DAG dependencies, scheduler selection, resource budgets, leases, cancellation, and lifecycle transitions. Existing `internal/jobs` remains for compatibility and for jobs not migrated in the first phase.

Alternatives considered:
- Expand `internal/jobs`: smaller initial diff, but the current job model lacks scope, dependency, lease, and resource concepts. Adding all of that to jobs would blur simple queued jobs with workflow DAG tasks.
- Replace jobs entirely in one step: cleaner end state, but higher migration risk and unnecessary for non-scan background work initially.

### Decision: Persist Workflow Runs, Tasks, Dependencies, Leases, And Resource Budgets

Add durable workflow tables rather than keeping scheduler state in memory. Required records include runs, tasks, task dependencies, task leases, and configured resource budgets. Tasks store `library_id`, `scope_key`, `task_type`, `stage`, `status`, `priority`, `payload_json`, `available_at`, timestamps, attempt count, `lease_until`, and error state.

Alternatives considered:
- Derive DAG from jobs payload JSON: avoids migrations but makes scheduler queries fragile and expensive.
- In-memory workflow graph: faster but loses recovery and does not support process restarts.

### Decision: Use Resource Budgets As The Primary Concurrency Control

Each task type declares required resources such as `db_write`, `local_disk_io`, `openlist_http`, `ffprobe`, `metadata_api`, and `cpu_heavy`. The scheduler claims tasks only when all required resources have available capacity. Stage concurrency can be represented as optional resources such as `stage_scan` or `stage_metadata_match` but must not be the only throttle.

Alternatives considered:
- Fixed stage pools only: simpler and maps to user language, but wastes capacity when tasks in the same stage consume different resources.
- One global worker count: easy to configure, but cannot prevent ffprobe/API/DB bottlenecks independently.

### Decision: Enforce Same-Library Safety In The Scheduler

The scheduler MUST prevent incompatible tasks for the same `library_id` from running concurrently. The first implementation should use conservative same-library mutual exclusion for mutating workflow tasks. Later refinements may allow declared-compatible same-library tasks, such as independent read-only diagnostics.

Alternatives considered:
- Let service methods handle their own locks: spreads concurrency rules across domains and makes queue fairness harder.
- Allow same-library stage overlap when DAG dependencies permit it: potentially faster, but risky while scanner/catalog/projection writes share state.

### Decision: Batch Large Work Into Bounded Tasks

Discovery, catalog materialization, probe, metadata match, and projection work should operate on bounded batches. Batch tasks should carry cursors or ID lists and enqueue successor tasks until the stage is complete. Default batch sizes should favor responsiveness and SQLite safety over maximum raw throughput.

Alternatives considered:
- Keep one task per whole library stage: simpler but still lets a large library monopolize a worker for a long time.
- One task per file/item for every stage: best fairness but too much queue overhead for large scans.

### Decision: Keep Existing API Triggers And Map Them To Workflow Runs

Endpoints and services that currently queue `sync_library`, targeted refresh, scheduled scans, or storage event refreshes should create workflow runs and return a compatible accepted response. Where job IDs are still expected, compatibility may create a lightweight bridge job or expose a workflow-backed status object through existing job listing until the UI is migrated.

Alternatives considered:
- Break API responses to return only workflow IDs: cleaner but unnecessary user-facing churn.
- Hide workflows completely behind jobs: preserves APIs but prevents useful workflow visibility.

### Decision: Lease Claimed Tasks And Requeue Expired Leases

Task claims write a lease owner and `lease_until`. Workers renew leases for long tasks. If the server exits or a worker crashes, expired leases become claimable again. Task handlers must be idempotent or bounded by upsert semantics because a task can run more than once after lease expiry.

Alternatives considered:
- Rely on process lifetime and `running` status: current approach can strand tasks after crashes.
- Use DB advisory locks only: not portable across SQLite/Postgres and insufficient for recovery metadata.

## Risks / Trade-offs

- More scheduler complexity -> keep workflow initially scoped to library ingest/refresh and cover scheduler selection with focused tests.
- SQLite write contention under higher concurrency -> default `db_write` budget to 1 for SQLite and keep batch sizes conservative.
- Duplicate execution after lease expiry -> require task handlers to be idempotent and use existing upsert/reconcile semantics.
- Same-library mutual exclusion may limit maximum throughput -> start conservative, then add compatibility declarations after scan/projection writes are proven safe.
- API compatibility may be awkward during migration -> provide explicit mapping from workflow run/task status to existing job list responses where needed.
- Resource configuration can be confusing -> ship named safe presets and document defaults in config.

## Migration Plan

1. Add workflow database models and migrations while leaving existing jobs intact.
2. Implement workflow service methods for creating runs, adding tasks/dependencies, claiming tasks, completing/failing tasks, cancellation, lease renewal, and expired lease recovery.
3. Add resource registry and default task type definitions for scan discovery, materialization, projection refresh, probe batches, metadata match batches, cleanup, and storage refresh.
4. Add workflow runner alongside the existing worker runner. Initially enable it for new library scan workflows while jobs continue handling unmigrated work.
5. Change manual scan, create-library scan, scheduled scan, and storage refresh queueing to create workflow runs for library-scoped scan work.
6. Gradually move post-scan materialization, probing, metadata matching, and projection refresh from job dispatch to workflow tasks.
7. Add admin/status visibility for workflow runs and tasks or map them into existing job APIs.
8. Once migrated behavior is stable, retire redundant job kinds or keep them as compatibility shims.

Rollback strategy: keep old job handlers and queueing paths behind configuration until workflow scanning is validated. If workflow execution is disabled, scan triggers fall back to existing job queue behavior.

## Open Questions

- Should workflow visibility be added to the existing `/api/v1/jobs` response first, or should a dedicated `/api/v1/workflows` endpoint be introduced immediately?
- What default resource budgets should be used for Postgres deployments versus SQLite deployments?
- Which same-library task pairs can safely overlap after the conservative first implementation?
