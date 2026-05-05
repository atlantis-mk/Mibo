## 1. Workflow Data Model

- [x] 1.1 Add workflow database models for runs, tasks, task dependencies, task leases, resource budgets, and resource usage under `internal/database`.
- [x] 1.2 Add AutoMigrate coverage and indexes for claim queries, dependency checks, run status queries, resource usage, and expired lease recovery.
- [x] 1.3 Define workflow status constants and task lifecycle transitions in a new `internal/workflow` package.
- [x] 1.4 Add focused database tests for model migration, required indexes, and basic persistence behavior.

## 2. Workflow Service Core

- [x] 2.1 Implement workflow run creation and active-run reuse for library-scoped scan, targeted refresh, scheduled scan, and storage-change refresh reasons.
- [x] 2.2 Implement task creation with dependency edges, task payload JSON encoding, priority, stage, scope key, and available time.
- [x] 2.3 Implement completion, failure, retry, skip, cancellation, and supersede transitions for runs and tasks.
- [x] 2.4 Implement expired lease recovery and lease renewal APIs.
- [x] 2.5 Add unit tests for run creation, DAG dependency unlocking, lifecycle transitions, cancellation, supersede, and lease recovery.

## 3. Resource-Aware Scheduler

- [x] 3.1 Define task type registry with default resource requirements for discovery, materialization, projection, probing, metadata matching, cleanup, and storage refresh.
- [x] 3.2 Add configuration defaults for resource budgets, including conservative SQLite defaults and higher configurable limits for non-SQLite deployments.
- [x] 3.3 Implement scheduler claim logic for ready tasks, dependency satisfaction, FIFO ordering, priority, per-run fairness, resource budget availability, and same-library mutual exclusion.
- [x] 3.4 Implement resource reservation and release tied to task lease and task finish transitions.
- [x] 3.5 Add scheduler tests for cross-library concurrency, same-library exclusion, FIFO behavior, resource exhaustion, fairness across large and small runs, and budget release on completion/failure.

## 4. Workflow Runner

- [x] 4.1 Add a workflow runner that polls and claims workflow tasks independently from the existing jobs runner.
- [x] 4.2 Add dispatch handlers for workflow task types and cancellation checks for long-running task handlers.
- [x] 4.3 Wire the workflow service and runner in `internal/app/app.go` while leaving existing jobs runner active for unmigrated job kinds.
- [x] 4.4 Add integration tests proving two libraries can progress concurrently when resource budgets allow.

## 5. Scanner Workflow Migration

- [x] 5.1 Add library workflow creation methods that convert manual scan, create-library scan, scheduled scan, and targeted refresh requests into workflow runs.
- [x] 5.2 Split scanner discovery and core synchronization into bounded workflow tasks with batch cursors or bounded root scopes.
- [x] 5.3 Convert catalog materialization and post-materialization scheduling to dependent workflow tasks.
- [x] 5.4 Convert catalog projection refresh scheduling to dependent workflow tasks.
- [x] 5.5 Preserve existing scan endpoint response behavior and add compatibility mapping where callers expect job-like status.
- [x] 5.6 Add scanner workflow tests covering skeleton visibility, missing file reconciliation, batch continuation, and independent cross-library execution.

## 6. Enrichment Workflow Migration

- [x] 6.1 Convert inventory probe batches to workflow tasks that use ffprobe, disk-read, CPU, and database-write resources.
- [x] 6.2 Convert metadata match batches to workflow tasks that use metadata API and database-write resources.
- [x] 6.3 Keep metadata operation evidence and projection refresh semantics intact when matching runs from workflow tasks.
- [x] 6.4 Add tests proving probe backlog does not block scan discovery and metadata provider delays do not block skeleton visibility.

## 7. Storage And Schedule Integration

- [x] 7.1 Route storage change planner output to scoped workflow runs or tasks instead of directly enqueueing globally blocking refresh jobs.
- [x] 7.2 Route scheduled library scans to workflow runs while preserving schedule run status updates.
- [x] 7.3 Preserve listener reconcile and non-library global jobs on the legacy jobs runner until explicitly migrated.
- [x] 7.4 Add tests for storage event refresh workflow creation, scheduled workflow status updates, and legacy job coexistence.

## 8. Observability And Compatibility

- [x] 8.1 Add workflow run and task status query methods with stage counts, error summaries, resource waits, and recent task details.
- [x] 8.2 Expose workflow status through existing job compatibility responses or add a dedicated workflow status API, based on final implementation choice.
- [x] 8.3 Add admin/diagnostic coverage for active resource budgets, active leases, blocked tasks, and expired lease recovery counts.
- [x] 8.4 Add API tests for scan trigger compatibility and workflow visibility.

## 9. Verification And Rollout

- [x] 9.1 Add feature/config fallback so scan triggers can use legacy jobs if workflow execution is disabled during rollout.
- [x] 9.2 Run focused backend tests for workflow, worker, library scan, probe, metadata, storage listener, and schedule packages.
- [x] 9.3 Run `go test ./...` from `mibo-media-server/` and resolve failures caused by the change.
- [x] 9.4 Document default resource budget behavior and operational tuning notes for local SQLite and larger deployments.
