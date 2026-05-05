## 1. Backend Diagnostics Model

- [x] 1.1 Define health diagnostic contracts for issue severity, reason codes, scopes, impacts, affected references, technical details, and recovery action descriptors.
- [x] 1.2 Add a backend diagnostics service that derives active issues from libraries, media sources, recent failed jobs, and schedule runs without changing existing job storage.
- [x] 1.3 Implement issue grouping so repeated failed jobs for the same root cause, media source, and affected libraries produce one active issue with latest failure context.
- [x] 1.4 Add classifiers for OpenList/PikPak captcha or authentication expiration and generic unknown job failures.
- [x] 1.5 Add unit tests for classification, grouping, affected-library derivation, and healthy empty diagnostics.

## 2. Backend APIs And Recovery Hooks

- [x] 2.1 Add authenticated health diagnostics endpoints for global summaries and issue listings.
- [x] 2.2 Include enough references in diagnostics responses for frontend links to affected media sources, libraries, and jobs.
- [x] 2.3 Add or expose a media source validation action that verifies provider connectivity after external repair.
- [x] 2.4 Add an affected-library rescan action or compose existing scan endpoints so recovery can re-scan all libraries referenced by an issue.
- [x] 2.5 Add backend API tests covering health listing, technical detail exposure, validation success, validation failure, and affected-library rescan behavior.

## 3. Frontend API And State Integration

- [x] 3.1 Add TypeScript types and client methods for health summaries, issue details, scopes, impacts, and action descriptors.
- [x] 3.2 Add React Query options and invalidation behavior for health diagnostics and recovery actions.
- [x] 3.3 Map reason codes to user-facing Chinese copy while preserving expandable technical details.
- [x] 3.4 Ensure health data loading failures degrade gracefully without blocking existing home, library, or settings rendering.

## 4. Health Center UI

- [x] 4.1 Add a Health Center route and navigation entry for authenticated users.
- [x] 4.2 Render active issues grouped by severity with blocking issues first.
- [x] 4.3 Show affected media sources, libraries, counts when available, latest failure time, and related job references.
- [x] 4.4 Add expandable technical detail panels for job kind, job status, payload context, and raw error text.
- [x] 4.5 Render supported recovery actions including opening OpenList, validating media source connectivity, re-scanning affected libraries, and viewing related jobs.
- [x] 4.6 Add frontend tests or focused component coverage for populated, empty, blocking, warning, and action-loading states.

## 5. Global And Settings Health Surfaces

- [x] 5.1 Add global health indicators in the app shell when active blocking or error issues exist.
- [x] 5.2 Mark affected media libraries in the sidebar and link them to the relevant issue or Health Center context.
- [x] 5.3 Update media source and library settings cards to show user-friendly health summaries and issue links.
- [x] 5.4 Preserve existing raw job list workflows as the technical troubleshooting fallback.

## 6. Homepage Degraded And Empty States

- [x] 6.1 Update home data loading to include diagnostics needed to distinguish no setup, no scan results, and health-blocked visibility.
- [x] 6.2 Replace the misleading empty state when libraries or scanned content exist but blocking health issues hide displayable catalog items.
- [x] 6.3 Add a degraded banner or compact indicator when some content is visible but active health issues affect other libraries or sources.
- [x] 6.4 Ensure homepage actions link to Health Center issue details or the relevant recovery flow.
- [x] 6.5 Add frontend coverage for empty setup state, normal populated state, fully health-blocked state, and partially degraded state.

## 7. Verification

- [x] 7.1 Run backend tests from `mibo-media-server/` with `go test ./...`.
- [x] 7.2 Run frontend typecheck from `web/` with `pnpm typecheck`.
- [x] 7.3 Manually verify the OpenList/PikPak captcha-expired scenario shows a Health Center blocking issue and a health-aware home empty state.
- [x] 7.4 Manually verify validating a repaired media source and re-scanning affected libraries clears or updates the active issue.
