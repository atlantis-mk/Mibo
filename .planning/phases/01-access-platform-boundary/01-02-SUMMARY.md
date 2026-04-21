---
phase: 01
plan: 02
subsystem: frontend-web
tags:
  - setup
  - routing
  - app-entry
key-files:
  created:
    - web/src/features/app/components/setup-guide-panel.tsx
  modified:
    - web/src/lib/client-config.ts
    - web/src/router.tsx
    - web/src/components/setup-wizard.tsx
    - web/src/features/app/components/browse-app-shell.tsx
    - web/src/features/app/hooks/use-app-controller.ts
key-decisions:
  - Split app entry into a hard gate (`/setup`) and a soft in-app configuration guide.
  - Keep normal client entry centered on `mibo-media-server` APIs rather than OpenList-specific flows.
requirements-completed:
  - ACCS-01
  - ACCS-03
  - CATA-01
duration: 24 min
completed: 2026-04-21T00:00:00Z
---

# Phase 01 Plan 02: Two-Stage Gate UX And Boundary Summary

Implemented the agreed two-stage app entry in the web client: first-time visitors still hit `/setup`, while signed-in admins without media configuration now enter the app boundary and land on a dedicated configuration guide instead of an empty home shell.

## Commits

| Scope | Commit | Description |
|------|--------|-------------|
| web sub-repo | `e0e7496` | Added shared setup-state helpers, router soft/hard gate behavior, and in-app setup guidance surface |

## What Changed

- Added shared setup-state helpers in `web/src/lib/client-config.ts`:
  - `canEnterApp(...)`
  - `isSetupFullyInitialized(...)`
  - `needsSetupGuide(...)`
- Updated `web/src/router.tsx` so the hard gate is still driven by setup status, but app entry now keys off the shared `canEnterApp(...)` rule.
- Updated `web/src/components/setup-wizard.tsx` copy and step framing so media source/library creation is clearly optional after admin creation.
- Added `web/src/features/app/components/setup-guide-panel.tsx` as the soft-gate landing surface.
- Wired `web/src/features/app/components/browse-app-shell.tsx` and `web/src/features/app/hooks/use-app-controller.ts` so `/` shows the config guide when the user can enter the app but media source or library setup is still incomplete.

## Verification

- `pnpm typecheck` -> PASS
- `pnpm build` -> PASS
- Manual check -> PASS: empty instance opens on `/setup`
- Manual check -> PASS: after creating only the admin user and skipping media setup, app entry lands on configuration guidance instead of an empty home shell
- Manual check -> PASS: the user-facing flow remains routed through `mibo-media-server` client APIs and shared setup status helpers, not OpenList-specific product entry flows

## Deviations from Plan

None - plan executed exactly as written.

## Self-Check: PASSED

- Key guide surface exists on disk
- Automated verification commands passed
- Manual hard-gate and soft-gate flows behaved as planned in the browser

## Next Phase Readiness

Phase 1 now has both the backend setup/auth contract and the frontend two-stage gate flow in place. The next phase can build on a stable app-entry boundary instead of re-litigating setup semantics.
