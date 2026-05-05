## Why

Fast skeleton ingest made newly discovered files visible quickly, but the follow-up scan, materialization, probing, metadata, and projection work is still coordinated through chained jobs whose payloads and failures do not clearly describe each media item's organizing progress. Users need honest per-media organizing state, and administrators need file/stage-level diagnostics and retry controls without reverse-engineering failed jobs.

## What Changes

- Introduce a dirty-driven ingest control plane anchored on `inventory_file` facts rather than a linear job chain as the only source of truth.
- Add condition snapshots that describe current organizing state per discovered file and stage, including materialization, probing, metadata matching, projection freshness, and review requirements.
- Add a reconciler that processes only dirty inventory files or dirty library scopes, derives missing/stale work from database facts, updates conditions, and dispatches necessary existing stage executors.
- Add event/journal records for important ingest stage transitions and failures so administrators can inspect what happened and when.
- Add user-facing organizing summaries for library browsing/cards so clients can display progress, partial readiness, and review-required states without deep joins or job-payload inference.
- Add administrator diagnostics for ingest failures/stale conditions and stage-level retry actions.
- Preserve current fast-ingest behavior: scan remains bounded to storage discovery and inventory persistence; expensive materialization, probe, metadata, and projection work remain asynchronous.

## Capabilities

### New Capabilities

- `ingest-reconciler-conditions`: Defines the dirty-driven ingest reconciliation model, condition semantics, event journal, user organizing summaries, and admin diagnostics/retry behavior.

### Modified Capabilities

- `media-graph-scanner`: Scanner synchronization must mark discovered and changed inventory files dirty for reconciliation instead of directly relying on chained enrichment payloads as the only continuation mechanism.
- `library-detail-browsing`: Library detail browsing must expose organizing progress summaries derived from ingest conditions for inventory-backed and catalog-backed media cards.
- `admin-console-dashboard`: Admin surfaces must include ingest organizing health, failed/stale stage diagnostics, and stage-level retry entry points.

## Impact

- Backend: `mibo-media-server/internal/library`, `internal/jobs`, `internal/worker`, `internal/catalog`, `internal/metadata`, `internal/probe`, `internal/health`, and `internal/httpapi`.
- Database: new durable state for dirty ingest units, condition snapshots, and event/journal records; indexes are required for dirty claiming, stage status queries, and admin diagnostics.
- APIs: new or extended responses for library organizing state and admin ingest diagnostics/retry actions.
- Frontend: library cards and admin/health surfaces need to render organizing progress, stage failures, review-required states, and retry affordances.
- Operations: existing job endpoints remain useful for raw worker inspection, but ingest diagnostics become the primary product-facing view for media organizing failures.
