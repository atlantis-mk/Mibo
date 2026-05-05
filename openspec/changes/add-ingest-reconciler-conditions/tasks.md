## 1. Data Model And Migration

- [x] 1.1 Add database models for ingest dirty units/scopes, ingest condition snapshots, and ingest event journal rows.
- [x] 1.2 Add indexes for dirty claim ordering, inventory-file lookup, library/root filtering, condition type/status filtering, retry-due lookup, and admin diagnostics queries.
- [x] 1.3 Add migration/backfill behavior that marks existing available video inventory files or active library scopes dirty without blocking startup on a full reconciliation pass.
- [x] 1.4 Add unit tests for model migration, unique constraints, dirty upsert behavior, and event retention metadata.

## 2. Reconciler Core

- [x] 2.1 Create an ingest/reconciliation service in the backend domain layer that claims bounded dirty batches and derives conditions from current database facts.
- [x] 2.2 Implement condition derivation for visibility, materialization, probe, metadata match, projection freshness, and review-required states.
- [x] 2.3 Implement dirty marking helpers for inventory files, catalog items, assets, and library/root projection scopes.
- [x] 2.4 Implement event journal append helpers for stage transitions, failures, retries, review-required outcomes, and administrator actions.
- [x] 2.5 Add tests proving reconciliation updates stale or incorrect conditions based on facts rather than trusting previous condition rows.
- [x] 2.6 Add tests proving normal reconciliation processes only dirty work and does not scan all inventory files.

## 3. Worker And Executor Integration

- [x] 3.1 Add worker support for ingest reconciliation jobs or scheduled reconciliation ticks that process dirty batches without starving sync jobs.
- [x] 3.2 Wire scanner discovery, refresh, missing marking, and listener refresh paths to mark affected ingest units or scopes dirty.
- [x] 3.3 Wire catalog materialization completion/failure to update facts, mark affected units dirty, and append condition events.
- [x] 3.4 Wire probe completion/failure/unavailable outcomes to mark affected units dirty and append condition events.
- [x] 3.5 Wire metadata match applied/no-candidate/failed/skipped outcomes to mark affected catalog targets or files dirty and append condition events.
- [x] 3.6 Route projection changes through dirty projection scopes and update projection-current conditions after refresh.
- [x] 3.7 Add integration tests for scan -> dirty reconcile -> materialize/probe/metadata/projection dispatch and convergence.

## 4. User-Facing Organizing State

- [x] 4.1 Extend backend browse/query DTOs with condition-derived organizing summary fields for discovered and catalog-backed media results.
- [x] 4.2 Ensure duplicate suppression still hides inventory-backed discovered cards once catalog-backed results represent the same file or asset.
- [x] 4.3 Add backend API tests for organizing, partially ready, ready, failed, and review-required browse summaries.
- [x] 4.4 Update frontend library detail types and API mapping for organizing summaries.
- [x] 4.5 Update library media cards to render organizing, partial-ready, failed, and review-required badges and concise progress copy.
- [x] 4.6 Limit final catalog-only actions on inventory-only organizing cards while preserving safe playback when a playable source exists.
- [x] 4.7 Add focused frontend tests for organizing card rendering, upgrade to catalog-backed card, and action gating.

## 5. Admin Diagnostics And Retry

- [x] 5.1 Add admin ingest diagnostics service/query that lists failed, stale, running, pending, and review-required conditions with file, library, item, job, provider, and event references.
- [x] 5.2 Add admin API endpoint for ingest diagnostics using a scenario-oriented response shape rather than raw table rows.
- [x] 5.3 Add stage-scoped retry action that marks the affected unit/scope dirty or dispatches follow-up work without duplicating currently running stages.
- [x] 5.4 Add admin API tests for diagnostics filtering, stale-stage detection, retry eligibility, and duplicate-running retry protection.
- [x] 5.5 Extend admin console summary with ingest organizing health counts.
- [x] 5.6 Add frontend admin console entry points and diagnostics UI for failed/stale/review-required ingest stages.
- [x] 5.7 Add frontend tests for diagnostics rendering and retry feedback states.

## 6. Health, Retention, And Maintenance

- [x] 6.1 Integrate ingest condition failures and stale stages into health issue generation without replacing existing failed-job health issues prematurely.
- [x] 6.2 Add a bounded admin maintenance action to mark a library/root scope dirty for full reconciliation.
- [x] 6.3 Implement event journal retention or compaction for old events while preserving current conditions and recent diagnostics.
- [x] 6.4 Add tests for health issue grouping, maintenance reconcile action, and event retention behavior.

## 7. Verification

- [x] 7.1 Run focused backend tests for ingest reconciliation, scanner integration, worker dispatch, probe, metadata, catalog browse, health, and admin diagnostics.
- [x] 7.2 Run full backend test suite with `go test ./...` from `mibo-media-server/`.
- [x] 7.3 Run frontend typecheck and focused library/admin UI tests from `web/`.
- [ ] 7.4 Manually validate a local media scan with sample media: newly discovered files appear quickly, organizing summaries progress, admin diagnostics show failures, and stage retry converges without duplicate cards.
