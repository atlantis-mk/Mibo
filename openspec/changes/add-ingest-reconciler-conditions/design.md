## Context

Mibo already separates fast file discovery from expensive catalog materialization, media probing, metadata matching, artwork work, and projection refresh. That split is correct for large libraries, but the continuation path is still encoded mostly as chained jobs and ad-hoc enqueue points. Jobs are useful execution records, yet they are a poor product-facing model for questions like "what is this file waiting on?", "why is this card still organizing?", and "can I retry only probe for these failed files?"

The durable fact that survives catalog graph reshaping is `inventory_file`. Catalog items can be created, merged, linked, reshaped into series/season/episode hierarchies, or marked for review, but the discovered file remains the stable anchor for organizing progress. The new design keeps database facts as the source of truth and adds a dirty-driven reconciliation layer that derives current conditions and missing work from those facts.

## Goals / Non-Goals

**Goals:**

- Give frontend browsing a stable, lightweight organizing summary for discovered and catalog-backed media.
- Give administrators stage-level diagnostics and retry actions for materialization, probe, metadata, projection, and review-required outcomes.
- Avoid a full-library polling reconciler by processing only dirty files/scopes and stale retry candidates.
- Preserve fast ingest: scans should not synchronously perform expensive sidecar parsing, ffprobe, remote metadata, or projection rebuilds.
- Keep `inventory_file` as the ingest anchor while allowing user-facing status to aggregate to catalog cards once item/asset links exist.
- Make the system self-healing: if facts and condition snapshots drift, dirty reconciliation should recompute the condition from facts rather than trusting a linear workflow table as a second source of truth.

**Non-Goals:**

- Replacing the existing `jobs` service or worker loop with an external workflow engine.
- Introducing Kafka, Redis, Temporal, or another queue dependency.
- Making remote metadata providers or ffprobe part of the scan critical path.
- Exposing raw condition tables as generic CRUD APIs.
- Solving every historical repair case in the first iteration; administrator-triggered full reconciliation can exist as a bounded maintenance action.

## Decisions

### Decision: Use dirty-driven reconciliation, not full-library polling

Normal reconciliation SHALL only process dirty inventory files, dirty library/root scopes, and retry-due failed conditions. Scanner, listener, materialization, probe, metadata, governance repair, and projection maintenance paths mark affected units or scopes dirty when they change relevant facts.

Alternatives considered:

- Periodic full-library reconcile: simpler mental model, but large libraries would repeatedly scan stable rows and put unnecessary pressure on SQLite.
- Pure chained jobs: less schema work, but does not provide reliable media-level progress, self-healing, or stage-level retry.

### Decision: Keep database facts as truth and conditions as derived snapshots

Conditions SHALL summarize current ingest state, but they are not authoritative over the underlying facts. Reconciliation derives conditions from `inventory_files`, asset/file links, asset/item links, `media_streams`, metadata operation/source rows, catalog item governance state, and projection freshness markers.

Example conditions:

- `visible`: discovered file can appear in library browsing.
- `materialized`: catalog/asset/file linkage exists or materialization reached review-required.
- `probed`: ffprobe completed, skipped, unavailable, or failed.
- `metadata_matched`: automated metadata applied, no candidate found, skipped, or failed.
- `projection_current`: affected browse/detail projection scope has been refreshed.
- `review_required`: manual review is needed because classification or metadata confidence is insufficient.

Alternatives considered:

- Single `scan_state`: too coarse to represent playable-but-unmatched or classified-but-probe-failed cases.
- Workflow stage table as truth: useful for execution, but vulnerable to drift if underlying facts are changed by migration, manual governance, cleanup, or repair jobs.

### Decision: Model per-stage conditions with Kubernetes-like status semantics

Each condition should have a stable type plus status, reason, message, severity, timestamps, attempts, and optional references to job, item, asset, file, metadata operation, or provider. Status values should support `unknown`, `pending`, `running`, `true`, `false`, `skipped`, `failed`, and `review_required`, or a similarly compact set with equivalent meaning.

The user-facing organizing state is derived from conditions rather than stored independently as a second truth. For example:

- `materialized=pending` -> organizing, "identifying media".
- `probed=running` -> organizing, "analyzing video streams".
- `metadata_matched=false reason=no_candidate` -> needs review, "metadata match needed".
- `materialized=true`, `probed=true/skipped`, `metadata_matched=true/skipped`, `projection_current=true` -> ready.

Alternatives considered:

- A single workflow status enum: easy to display, but cannot represent partial readiness and multiple simultaneous issues.
- Raw job status display: implementation-centric and not stable enough for product UI.

### Decision: Dispatch existing executors through reconciler-selected work

The reconciler should dispatch or reuse the existing materialize, probe, metadata match, and projection executors. It should not duplicate classification, probe, or metadata logic. Jobs remain the execution mechanism and audit trail for worker attempts, but the reconciler decides what work is needed based on facts and conditions.

The first implementation can bridge by creating specialized jobs such as materialize/probe/metadata/projection batches from dirty units. Later iterations may claim stage work directly, but the API contract should be about conditions and diagnostics, not job internals.

Alternatives considered:

- Replace all jobs with stage workers immediately: cleaner eventual architecture, but too large and risky for one change.
- Keep each executor enqueueing the next executor: preserves current coupling and makes admin stage retry ambiguous.

### Decision: Add an event journal for history, not current state

Condition rows answer "what is true now". Event journal rows answer "what happened before now." Events should be append-only for meaningful transitions and errors, with a retention policy such as recent N events per unit plus recent N days globally.

Alternatives considered:

- Conditions only: enough for current UI, weak for admin diagnosis and timeline.
- Full event sourcing: strong auditability, but overkill and too expensive for current Mibo needs.

### Decision: Provide scenario-driven APIs

APIs should model product scenarios instead of exposing low-level tables.

Proposed API shapes:

- Library browsing responses include an organizing summary for each card when available.
- `GET /api/v1/libraries/{id}/organizing` lists organizing and review-required media in that library scope for user-facing progress surfaces.
- `GET /api/v1/admin/ingest/diagnostics` lists failed, stale, running, and review-required ingest units/stages for administrators.
- `POST /api/v1/admin/ingest/stages/{id}/retry` retries one failed or skipped stage.
- Optional maintenance action: `POST /api/v1/admin/ingest/reconcile` marks a bounded library/root scope dirty and starts reconciliation.

Existing `/api/v1/jobs` remains for raw job inspection and compatibility, but ingest diagnostics become the primary media-organizing view.

Alternatives considered:

- Extend `/api/v1/jobs` with media-specific projections: mixes execution records with product diagnosis and still cannot represent current state after jobs are superseded.
- Generic condition CRUD endpoints: violates the business API boundary and leaks storage details.

### Decision: Projection refresh is a dirty scope, not a side effect everywhere

Materialization, probe, metadata, and governance changes should mark projection scopes dirty. A projection reconciler or dirty-scope worker refreshes affected item/library scopes with batching and debounce. This reduces duplicate projection refresh jobs and makes `projection_current` a visible condition.

Alternatives considered:

- Continue enqueueing projection refresh from every executor: simple, but creates duplicate work and unclear freshness state.
- Synchronous projection refresh in each executor: hurts throughput and fast-ingest behavior.

## Risks / Trade-offs

- Additional tables and writes increase SQLite pressure -> Keep reconciliation dirty-driven, use small transactions, add targeted indexes, batch condition updates, and retain probe concurrency limits.
- Conditions can drift from facts if marking dirty is missed -> Provide bounded admin full-reconcile actions and have critical executors mark dirty on completion/failure.
- Event journal can grow without bound -> Add retention or compaction from the first implementation.
- Admin retry may conflict with currently running work -> Retry actions should check condition/job state and either no-op, mark dirty, or enqueue follow-up work rather than duplicate a running stage.
- User-facing progress could become noisy -> Derive concise messages from condition priority and hide technical detail from normal cards.
- Migration may need to initialize conditions for existing libraries -> Backfill by marking active libraries/root scopes dirty and reconciling in bounded batches after migration.

## Migration Plan

1. Add database models and indexes for dirty units/scopes, condition snapshots, and event journal rows.
2. Backfill by marking existing available video inventory files or active library scopes dirty without blocking startup on full reconciliation.
3. Add reconciler service and worker job kind, initially dispatching existing materialize/probe/metadata/projection jobs where needed.
4. Update scan/listener/materialize/probe/metadata/projection completion paths to mark affected units/scopes dirty.
5. Add condition-derived organizing summaries to browse/query contracts while preserving existing fields.
6. Add admin diagnostics and retry endpoints after condition snapshots are reliable.
7. Roll back safely by leaving facts and existing jobs intact; condition/dirty/event tables can be ignored by old code if needed.

## Open Questions

- Should the first implementation include event retention enforcement immediately, or only write events and add retention in a follow-up maintenance task?
- Should metadata `no_candidate` be considered `review_required` for all library types, or only when the library metadata policy requires automated matching?
- What exact stale threshold should mark running/pending conditions as admin-visible warnings?
