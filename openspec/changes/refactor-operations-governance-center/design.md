## Context

The current operations center derives display rows directly from `IngestCondition`, failed `WorkflowTask`, probe failures, and a small set of recommended actions. This makes the UI useful for diagnostics, but weak for governance:

- Related failures are not grouped. A season with many failing episodes becomes many separate tasks.
- Task IDs are tied to low-level condition IDs or workflow task IDs, so lifecycle is transient and noisy across scans.
- "Mark handled" closes a single condition and often does not represent the actual domain fix.
- Frontend dialogs assume one affected item or file, which does not match series, season, multi-version movie, or batch remediation workflows.

The existing ingest/workflow tables remain the source of operational facts. This change adds a persistent issue layer above them.

## Goals / Non-Goals

**Goals:**

- Persist stable operations issues that aggregate low-level facts by domain scope and reason.
- Collapse repeated episode-level failures into one series/season issue while keeping affected episode/file evidence.
- Provide remediation actions that map to media governance concepts: candidate application, metadata edits, episode numbering correction, classification acceptance, resource relinking, retry, exclusion, and explicit resolution.
- Keep an audit trail of governance actions and allow resolved issues to reopen when the same fingerprint recurs.
- Migrate the operations workbench from a flat task table to an issue inbox without breaking the existing task API immediately.

**Non-Goals:**

- Replace ingest, workflow, recognition, or metadata governance internals.
- Build a full background job orchestration system separate from the existing workflow service.
- Delete historical ingest events or workflow failures.
- Solve every recognition quality issue in this change; the issue system should make those problems governable.

## Decisions

### Add a Persistent Issue Layer

Create operations tables:

- `operation_issues`: stable issue record keyed by fingerprint, with scope, kind, severity, status, timestamps, title, summary, counts, and optional resolved metadata.
- `operation_issue_occurrences`: links to source facts such as ingest conditions, workflow tasks, metadata operations, recognition decisions, and probe/resource failures.
- `operation_issue_targets`: normalized affected entities: library, media source, inventory file, resource, metadata item, series, season, episode.
- `operation_issue_actions`: materialized action descriptors with action type, eligibility, parameters, and labels.
- `operation_issue_events`: audit trail for created, updated, action requested, action succeeded, action failed, resolved, ignored, and reopened.

Alternative considered: continue generating rows on every request. Rejected because it cannot support durable lifecycle, audit, reopening, or stable group resolution.

### Aggregate by Domain Fingerprint

The aggregator computes fingerprints from problem kind, library, domain scope, and reason:

- Series/season metadata issue: `metadata_review:<library_id>:series:<series_id>:season:<season_id|all>:<reason>`
- Movie metadata issue: `metadata_review:<library_id>:movie:<item_id>:<reason>`
- Classification issue: `classification:<library_id>:scope:<series|season|folder|file>:<id-or-normalized-path>:<reason>`
- Probe issue: `probe:<library_id>:resource-or-file:<id>:<reason>`
- Storage/workflow issue: `workflow:<library_id>:source-or-stage:<scope>:<error-class>`

Alternative considered: group by raw error message. Rejected because provider and workflow messages are unstable and can accidentally merge unrelated media.

### Series and Season Are the Default Scope for Episodic Failures

When affected targets include episode metadata items or episode-shaped inventory signals:

- Prefer season scope when all failures belong to one season.
- Prefer series scope when failures span multiple seasons of the same series.
- Fall back to folder/library scope only when no series/season can be inferred.
- Store every affected episode/file as issue targets and representative samples.

Alternative considered: always group at series level. Rejected because large shows need season-sized work units for practical governance.

### Actions Execute Against Issues, Not Bare Conditions

Issue actions resolve or update the underlying facts they own:

- Retry actions mark all linked conditions/files/items dirty as appropriate.
- Metadata candidate actions apply a candidate to the target item or group and then refresh linked conditions.
- Episode numbering actions update episode/season numbering and resource links, then queue relevant projection/match work.
- Classification actions accept or adjust all linked classification decisions in the issue scope.
- Resolve actions require a governance reason and record a user event before closing the issue.

Alternative considered: keep existing `executeOperationsAction(actionID)` strings only. Rejected because one action ID cannot express grouped parameters, partial success, or audit context cleanly.

### Preserve Task API During Migration

Existing `/api/v1/operations/tasks` remains available and can be backed by active issues converted to task-shaped responses. New UI uses issue APIs:

- `GET /api/v1/operations/issues`
- `GET /api/v1/operations/issues/{id}`
- `POST /api/v1/operations/issues/{id}/actions`
- `GET /api/v1/operations/issues/{id}/events`

Alternative considered: replace task API immediately. Rejected because home/settings surfaces already consume tasks and can migrate incrementally.

## Risks / Trade-offs

- Aggregation may hide important per-file detail -> Mitigation: targets and samples remain queryable, and the detail view shows all linked occurrences.
- Fingerprints may be too broad or too narrow -> Mitigation: add targeted tests for series, season, movie, probe, and workflow cases; include versioned fingerprint generation helpers.
- Group actions can partially fail -> Mitigation: action results include per-target success/failure and the issue remains active or in progress until all required targets are governed.
- Resolved issues may reappear after scans -> Mitigation: recurring fingerprints reopen the issue with a `reopened` event instead of duplicating it.
- New tables add migration complexity -> Mitigation: additive migration only; rollback can leave issue tables unused while existing ingest/workflow facts continue to operate.

## Migration Plan

1. Add operation issue database models and migrations.
2. Implement the aggregator in read-only mode and compare generated issues with current task output in tests.
3. Add issue list/detail/action APIs while keeping task APIs unchanged.
4. Back `/operations/tasks` from active issues where possible, falling back to legacy task generation for unmigrated categories.
5. Update frontend workbench to use issue APIs and keep legacy task rendering only for compatibility/debug fallback.
6. Add action execution and audit logging category by category, starting with episodic metadata/classification issues.
7. Remove or simplify legacy task generation after the new issue path covers all current operations categories.

Rollback strategy: disable the aggregator and frontend issue route, fall back to legacy `/operations/tasks`. Since migrations are additive, no destructive rollback is required.

## Open Questions

- Should "ignore" remain visible as a top-level action, or should it be renamed to "exclude from scanning" and moved behind a confirmation flow?
- Should issue assignment/ownership be included now, or deferred until multi-admin workflows become necessary?
- What is the desired retention for resolved issue events beyond the existing ingest event retention?
