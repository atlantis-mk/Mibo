## 1. Data Model And Migration

- [x] 1.1 Add database models for operation issues, occurrences, targets, actions, and events.
- [x] 1.2 Add additive migrations and auto-migration coverage for the new operation governance tables.
- [x] 1.3 Add constants and normalization helpers for issue kind, scope kind, lifecycle status, action type, target type, and event type.
- [x] 1.4 Add repository helpers for upserting issues by fingerprint, replacing targets/actions, appending events, and querying issue details.

## 2. Aggregation Engine

- [x] 2.1 Implement an operations issue aggregator that reads active ingest conditions, failed workflow tasks, metadata operations, recognition decisions, and probe failures.
- [x] 2.2 Implement stable fingerprint generation for metadata, classification, probe, workflow, storage, and projection issue categories.
- [x] 2.3 Implement lifecycle reconciliation so observed issues update, missing active facts resolve or remain resolved, and recurring resolved issues reopen.
- [x] 2.4 Add occurrence and target extraction for libraries, media sources, inventory files, resources, metadata items, series, seasons, and episodes.
- [x] 2.5 Add unit tests for duplicate prevention, source occurrence linking, target samples, resolved lifecycle, and reopen lifecycle.

## 3. Episodic Grouping

- [x] 3.1 Add helpers to resolve episode targets to season and series scope through metadata hierarchy and resource links.
- [x] 3.2 Group same-season episode metadata review failures into one season-scoped issue with affected episode/file counts.
- [x] 3.3 Group same-reason multi-season episode failures into one series-scoped issue with per-season target counts.
- [x] 3.4 Add fallback grouping for episode-like files that lack linked series metadata using normalized folder scope and reason.
- [x] 3.5 Add regression tests proving a season with many failing episodes produces one issue while unrelated movies remain separate.

## 4. Issue APIs And Compatibility

- [x] 4.1 Add issue list, detail, event list, and action execution HTTP endpoints under `/api/v1/operations/issues`.
- [x] 4.2 Add server-side filters for status, kind, action type, library, query, page, and page size.
- [x] 4.3 Convert active issues into legacy task-shaped responses for `/api/v1/operations/tasks` while preserving fallback coverage for unmigrated cases.
- [x] 4.4 Update operations overview and pipeline summary counts to prefer issue data where available.
- [x] 4.5 Add API tests for list/detail/action authorization, filtering, pagination, and legacy compatibility responses.

## 5. Governance Actions

- [x] 5.1 Implement issue action planning so each issue includes eligible remediation actions with labels, parameters, and target counts.
- [x] 5.2 Implement grouped retry actions for linked files, metadata items, library scopes, and projection scopes.
- [x] 5.3 Implement metadata candidate and manual-governed actions that update target metadata and refresh linked conditions.
- [x] 5.4 Implement classification accept/correct actions for all linked review-required classification decisions in an issue scope.
- [x] 5.5 Implement resource relink/unlink action hooks using existing metadata governance resource operations.
- [x] 5.6 Implement explicit exclusion and ignore-with-reason actions with confirmation metadata and audit events.
- [x] 5.7 Add action result tests for success, partial failure, audit logging, and issue completion behavior.

## 6. Frontend API Contracts

- [x] 6.1 Add TypeScript types for operation issues, targets, occurrences, actions, action results, and events.
- [x] 6.2 Add authenticated API methods and query options for issue list, issue detail, issue events, and issue action execution.
- [x] 6.3 Preserve existing operations task query consumers during the migration.
- [x] 6.4 Add focused tests for issue API response mapping and query key invalidation.

## 7. Operations Workbench UI

- [x] 7.1 Replace the flat operations manage table with an issue inbox using grouped rows, affected counts, scope labels, severity, lifecycle status, and sample targets.
- [x] 7.2 Add issue filters for status, kind, action type, library, and search with server-side pagination.
- [x] 7.3 Add issue detail surfaces for grouped evidence, targets, occurrences, events, and actions.
- [x] 7.4 Update metadata review UI to support grouped movie, season, and series issue contexts instead of only the first affected item.
- [x] 7.5 Add action execution UX with disabled duplicate execution, progress feedback, per-target results, partial failure display, and post-action refresh.
- [x] 7.6 Add fallback UI for legacy task data or issue API failure.

## 8. Verification

- [x] 8.1 Run backend tests for operations, ingest, metadata, library, recognition, and HTTP API packages.
- [x] 8.2 Run frontend lint, type-check/build, and relevant operations UI tests.
- [x] 8.3 Verify with seeded data that a TV season with repeated episode failures appears as one issue and resolves/reopens correctly.
- [x] 8.4 Verify movie, multi-version movie, extras, probe failure, storage auth failure, and workflow failure cases remain actionable.
- [x] 8.5 Update developer or operations documentation if endpoint contracts or governance workflows changed.
