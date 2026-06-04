## Why

The media library operations center currently surfaces low-level ingest conditions and workflow failures as separate tasks, which causes episodic content to explode into one error per episode and leaves many movie/series failures with only a generic ignore-or-confirm path. Operators need a governance-oriented workflow that groups related failures, offers domain-specific remediation actions, and records when an issue has actually been handled.

## What Changes

- Introduce persistent operations issues that group related conditions, workflow failures, recognition decisions, and metadata operations into stable governance work items.
- Aggregate episodic failures at the series or season scope when the affected files/episodes belong to the same show, while preserving per-episode evidence for inspection.
- Replace single-object task handling with issue-level remediation actions such as metadata candidate application, episode numbering correction, classification acceptance, resource relinking, retry, exclusion, and mark-as-governed.
- Track governance lifecycle explicitly: active, in progress, resolved, reopened, and ignored-with-reason.
- Add issue action audit events so completed governance work is explainable and repeat scans can reopen only genuinely recurring issues.
- Update the frontend operations center to present issue groups, affected counts, samples, action menus, and grouped metadata/classification workspaces instead of a flat task table.
- Keep existing task APIs compatible during migration, backed by the new issue model where possible.

## Capabilities

### New Capabilities

- `operations-governance-issues`: Persistent operations issue model, aggregation, lifecycle, audit events, and issue list/detail APIs.
- `operations-governance-actions`: Domain-specific remediation actions for metadata, classification, resource linkage, retry, exclusion, and marking an issue as governed.
- `episodic-governance-grouping`: Series/season-aware grouping rules that collapse repeated episode-level failures into one actionable issue with retained evidence.
- `operations-governance-workbench`: Frontend workbench behavior for issue grouping, filtering, samples, grouped action execution, and post-action refresh.

### Modified Capabilities

- None. No archived baseline specs exist for the current operations center.

## Impact

- Backend: `mibo-media-server/internal/operations`, `internal/ingest`, `internal/library`, `internal/metadata`, `internal/recognition`, database models/migrations, and HTTP handlers under `internal/httpapi`.
- Frontend: `frontend/src/features/operations`, operations query/API contracts in `frontend/src/lib/mibo-api.ts` and `frontend/src/lib/mibo-query.ts`, and related metadata governance dialogs.
- APIs: add issue-oriented operations endpoints and action execution contracts; preserve existing `/api/v1/operations/tasks` during transition.
- Data: add persistent operations issue, occurrence, target, action, and event tables; no destructive migration of ingest/workflow history.
- Tests: add backend aggregation/action tests and frontend workbench behavior tests, especially around season/series grouping and resolved/reopened lifecycle.
