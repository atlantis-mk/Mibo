# Codebase Concerns

**Analysis Date:** 2026-04-21

## Tech Debt

**Frontend application state is concentrated in a few oversized modules:**
- Issue: Core UI state, routing side effects, auth flows, source management, metadata actions, and player orchestration are concentrated in `web/src/features/app/hooks/use-app-controller.ts` (1175 lines), with additional large stateful surfaces in `web/src/components/setup-wizard.tsx` (709 lines), `web/src/components/settings/settings-shell.tsx` (818 lines), and `web/src/features/app/hooks/use-library-data-state.ts` (424 lines).
- Files: `web/src/features/app/hooks/use-app-controller.ts`, `web/src/components/setup-wizard.tsx`, `web/src/components/settings/settings-shell.tsx`, `web/src/features/app/hooks/use-library-data-state.ts`, `web/src/router.tsx`
- Impact: Small product changes require edits across multiple coupled hooks and setup screens, which increases regression risk and makes route/setup/auth behavior drift more likely.
- Fix approach: Split `web/src/features/app/hooks/use-app-controller.ts` by domain (`auth`, `libraries`, `sources`, `metadata`, `playback`), centralize setup/auth persistence around `web/src/lib/client-config.ts`, and reduce cross-hook mutation.

**Backend HTTP surface is implemented as a monolith:**
- Issue: Routing, auth checks, playback URL rewriting, CORS, logging, and error serialization are all implemented inside `mibo-media-server/internal/httpapi/router.go` (1182 lines), while `mibo-media-server/internal/httpapi/router_test.go` (1584 lines) has become the only broad integration safety net.
- Files: `mibo-media-server/internal/httpapi/router.go`, `mibo-media-server/internal/httpapi/router_test.go`
- Impact: Adding or securing a single endpoint requires editing a large shared file, which makes accidental auth or response-shape regressions easy.
- Fix approach: Split `mibo-media-server/internal/httpapi/router.go` by resource area (`auth`, `media sources`, `libraries`, `media items`, `jobs`, `playback`) and apply shared auth middleware instead of per-handler checks.

**Schema evolution relies on startup automigration:**
- Issue: Database changes are applied through `gorm.AutoMigrate` on every startup with no versioned migration history, rollback path, or explicit data backfill layer.
- Files: `mibo-media-server/internal/database/database.go`, `mibo-media-server/internal/database/models.go`
- Impact: Production schema changes are harder to review, audit, and roll back; data-shape changes can become coupled to application boot.
- Fix approach: Introduce explicit migrations and reserve `AutoMigrate` for local development only.

## Known Bugs

**Setup-status fetch failures unlock the app shell:**
- Symptoms: If the setup-status request fails, the router marks the app as initialized and renders the main app instead of keeping the user in setup or showing a blocking error.
- Files: `web/src/router.tsx`
- Trigger: Any network failure or backend error during `createMiboApi({ baseUrl: getStoredApiBaseUrl() }).getSetupStatus()`.
- Workaround: Keep the backend reachable during app bootstrap; otherwise users land in partially initialized screens that fail later.

**Media-detail routes perform duplicate item fetches:**
- Symptoms: Opening `/media/$mediaItemId` triggers one request in the route layer to discover `library_id`, then another request in app state bootstrapping to load the same item again.
- Files: `web/src/router.tsx`, `web/src/features/app/hooks/use-library-data-state.ts`
- Trigger: Visiting any standalone media page.
- Workaround: None in the current implementation; extra latency is paid on every media-detail navigation.

## Security Considerations

**Many mutation and data endpoints are exposed without authentication:**
- Risk: Unauthenticated callers can list or mutate media sources, libraries, jobs, playback links, and media-item data because several handlers never call `requireUser`.
- Files: `mibo-media-server/internal/httpapi/router.go`, especially handlers registered for `/api/v1/media-sources`, `/api/v1/libraries`, `/api/v1/media-items/{id}`, `/api/v1/media-items/{id}/playback`, `/api/v1/media-files/{id}/link`, `/api/v1/jobs`, and `/api/v1/jobs/{id}/retry`
- Current mitigation: Only some routes such as `/api/v1/me/*`, `/api/v1/storage/providers/{provider}/browse`, `/api/v1/storage/openlist/*`, `/api/v1/settings/metadata`, and `/api/v1/media-files/{id}/stream` enforce auth in `mibo-media-server/internal/httpapi/router.go`.
- Recommendations: Apply auth middleware by default, explicitly mark only bootstrap-safe routes (`/healthz`, `/readyz`, `/api/v1/setup/status`, guarded first-user bootstrap if desired) as public, and add endpoint-level authorization tests in `mibo-media-server/internal/httpapi/router_test.go`.

**Authorization model is incomplete even though roles exist:**
- Risk: `database.User` has a `Role` field, but new users are always created as `user` and no handler checks roles; `POST /api/v1/auth/register` remains public after setup.
- Files: `mibo-media-server/internal/database/models.go`, `mibo-media-server/internal/auth/service.go`, `mibo-media-server/internal/httpapi/router.go`
- Current mitigation: None beyond password hashing and session tokens.
- Recommendations: Restrict registration after initial bootstrap, add admin-only guards for settings/source/library/job mutations, and enforce role checks in HTTP handlers or middleware.

**Session tokens and playback tokens are exposed through browser storage and query strings:**
- Risk: Session tokens are stored in `localStorage`, and local playback URLs are rewritten with `access_token` query parameters, which increases leakage risk through browser history, copied URLs, and external tooling.
- Files: `web/src/features/app/hooks/use-auth-state.ts`, `web/src/components/setup-wizard.tsx`, `web/src/router.tsx`, `web/src/lib/client-config.ts`, `mibo-media-server/internal/httpapi/router.go`, `mibo-media-server/internal/httpapi/router_test.go`
- Current mitigation: Backend hashes stored session tokens in `mibo-media-server/internal/auth/service.go`, and stream access still authenticates in `mibo-media-server/internal/httpapi/router.go`.
- Recommendations: Move browser auth to secure cookies or short-lived tokens, stop embedding tokens in playback URLs, and use signed one-time stream URLs for `mibo-media-server/internal/httpapi/router.go`.

**Secrets are flagged but still stored in plaintext:**
- Risk: OpenList credentials and metadata API keys are serialized into the database as raw strings; `IsSecret` only labels records and does not encrypt them.
- Files: `mibo-media-server/internal/library/service.go`, `mibo-media-server/internal/providers/source_config.go`, `mibo-media-server/internal/settings/service.go`, `mibo-media-server/internal/database/models.go`
- Current mitigation: API responses sanitize OpenList config views and mask whether API keys exist.
- Recommendations: Encrypt secret values at rest, separate secret storage from normal settings rows, and avoid persisting third-party passwords in `ConfigJSON` unless unavoidable.

**CORS is permissive by default:**
- Risk: The backend defaults `MIBO_CORS_ALLOWED_ORIGINS` to `*`, which broadens browser access to the HTTP API.
- Files: `mibo-media-server/internal/config/config.go`, `mibo-media-server/internal/httpapi/router.go`
- Current mitigation: None; wildcard is the default.
- Recommendations: Default to explicit origins in non-development environments and fail closed when CORS is not configured.

## Performance Bottlenecks

**Library scans are single-threaded and refresh-heavy:**
- Problem: Scans recurse directory-by-directory, request every page with `Refresh: true`, and perform per-file item upserts, file upserts, metadata-job enqueueing, and probe-job enqueueing in the hot loop.
- Files: `mibo-media-server/internal/library/scan.go`, `mibo-media-server/internal/jobs/service.go`, `mibo-media-server/internal/worker/worker.go`, `mibo-media-server/internal/storage/openlist/adapter.go`
- Cause: `walkDirectory` in `mibo-media-server/internal/library/scan.go` does sequential traversal and `listAllDirectoryObjects` always forces provider refreshes, which is especially expensive against OpenList HTTP calls.
- Improvement path: Add bounded concurrency, avoid unconditional refresh for every page, batch upserts where possible, and separate discovery from metadata/probe scheduling.

**Standalone media navigation duplicates network work:**
- Problem: Media-item routing fetches the same item once to resolve `library_id` and again during normal app-data loading.
- Files: `web/src/router.tsx`, `web/src/features/app/hooks/use-library-data-state.ts`
- Cause: The route layer depends on `libraryId` before rendering `MediaItemPage`, while the controller still bootstraps the selected item separately.
- Improvement path: Make `libraryId` part of the route shape, or pass prefetched item data into `useLibraryDataState` so the second fetch is skipped.

## Fragile Areas

**OpenList integration depends on upstream HTTP contract details:**
- Files: `mibo-media-server/internal/storage/openlist/adapter.go`, `mibo-media-server/internal/httpapi/router_test.go`, `OpenList/`
- Why fragile: The adapter hard-codes `/api/auth/login`, `/api/fs/list`, `/api/fs/get`, and `/api/fs/link`. `OpenList/` is an upstream checkout rather than shared package code, so upstream API changes can silently break Mibo without compiler help.
- Safe modification: Treat `mibo-media-server/internal/storage/openlist/adapter.go` as a compatibility layer, pin against known OpenList versions, and add explicit contract tests for all required endpoints.
- Test coverage: `mibo-media-server/internal/httpapi/router_test.go` and `mibo-media-server/internal/worker/worker_test.go` use mocked OpenList handlers, but there is no real upstream compatibility suite against the checked-out `OpenList/` server.

**Frontend setup/auth persistence is duplicated across screens:**
- Files: `web/src/components/setup-wizard.tsx`, `web/src/features/app/hooks/use-auth-state.ts`, `web/src/features/app/hooks/use-app-controller.ts`, `web/src/router.tsx`, `web/src/lib/client-config.ts`
- Why fragile: Token persistence, API-base persistence, setup refresh, and session clearing are implemented in multiple places with separate effects.
- Safe modification: Centralize browser-session handling in one hook or store and keep route/setup code read-only.
- Test coverage: No automated frontend tests were detected under `web/src`, so persistence regressions are currently caught manually.

## Scaling Limits

**Job execution is effectively single-worker polling:**
- Current capacity: One `worker.Run` loop is started from `mibo-media-server/internal/app/app.go`, and `mibo-media-server/internal/worker/worker.go` processes jobs sequentially.
- Limit: Large scan, metadata, or probe backlogs will queue behind one another and are gated by the poll interval from `mibo-media-server/internal/config/config.go`.
- Scaling path: Support concurrent workers, lease timeouts, and workload-specific queues instead of one sequential polling loop.

**Directory browsing truncates large directory trees:**
- Current capacity: `mibo-media-server/internal/library/browse.go` requests only page 1 with `browsePerPage = 500` and returns that single page.
- Limit: Directories with more than 500 child directories are partially invisible in the browse UI.
- Scaling path: Add pagination to the browse API and UI, or iterate all pages the way `listAllDirectoryObjects` does in `mibo-media-server/internal/library/scan.go`.

## Dependencies at Risk

**OpenList upstream API compatibility:**
- Risk: Mibo depends on specific OpenList endpoint shapes without vendoring a stable client contract.
- Impact: An upstream OpenList upgrade can break source testing, browsing, and link generation in `mibo-media-server/internal/storage/openlist/adapter.go`.
- Migration plan: Introduce adapter contract tests against a pinned OpenList version and treat upgrades to `OpenList/` as explicit compatibility work.

## Missing Critical Features

**No enforced admin/permission model for operational APIs:**
- Problem: The codebase has user roles in `mibo-media-server/internal/database/models.go`, but operational routes do not use them and several sensitive handlers are fully public in `mibo-media-server/internal/httpapi/router.go`.
- Blocks: Secure multi-user deployments, delegated administration, and safe exposure of the backend outside a trusted local network.

## Test Coverage Gaps

**Frontend application flows are untested:**
- What's not tested: Setup flow, auth persistence, route guarding, source creation, playback interactions, and metadata actions in `web/src`.
- Files: `web/package.json`, `web/src/components/setup-wizard.tsx`, `web/src/features/app/hooks/use-app-controller.ts`, `web/src/router.tsx`
- Risk: Regressions in setup/auth/navigation behavior are likely to ship unnoticed because no frontend test runner or test files were detected.
- Priority: High

**Authorization regressions are not comprehensively tested:**
- What's not tested: Public-vs-protected behavior for most library, media-source, playback, and job endpoints in `mibo-media-server/internal/httpapi/router.go`.
- Files: `mibo-media-server/internal/httpapi/router.go`, `mibo-media-server/internal/httpapi/router_test.go`
- Risk: Missing `requireUser` checks can be introduced or persist unnoticed; the current suite explicitly asserts unauthorized access for the storage-provider browse route but not for the wider API surface.
- Priority: High

---

*Concerns audit: 2026-04-21*
