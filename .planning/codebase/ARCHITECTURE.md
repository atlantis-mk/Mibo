# Architecture

**Analysis Date:** 2026-04-21

## Pattern Overview

**Overall:** Multi-package workspace with a Vite SPA frontend in `web/` and a layered Go HTTP service in `mibo-media-server/`, connected over JSON HTTP.

**Key Characteristics:**
- Keep product code in package roots: `web/` for the UI and `mibo-media-server/` for backend behavior.
- Treat `OpenList/` as an external upstream boundary; integrate through `mibo-media-server/internal/storage/openlist/adapter.go`, not by importing upstream source.
- Centralize frontend orchestration in controller hooks such as `web/src/features/app/hooks/use-app-controller.ts`, then render through shell components in `web/src/features/app/components/`.

## Layers

**Workspace Boundary Layer:**
- Purpose: Separate owned product packages from tooling and upstream checkouts.
- Location: repo root `package.json`, `web/`, `mibo-media-server/`, `OpenList/`.
- Contains: workspace tooling at `package.json`, frontend app code under `web/`, backend app code under `mibo-media-server/`, upstream OpenList checkout under `OpenList/`.
- Depends on: package-local manifests such as `web/package.json` and `mibo-media-server/go.mod`.
- Used by: all local development and mapping workflows.

**Frontend Bootstrap Layer:**
- Purpose: Start the browser app and install global providers.
- Location: `web/src/main.tsx`, `web/src/router.tsx`.
- Contains: React root creation, theme/toast/tooltip providers, route definitions, setup gate checks.
- Depends on: `@tanstack/react-router`, `web/src/lib/client-config.ts`, `web/src/lib/mibo-api.ts`.
- Used by: all browser navigation into `web/src/features/app/pages/*.tsx` and `web/src/components/setup-wizard.tsx`.

**Frontend Feature Composition Layer:**
- Purpose: Map routes to page state, auth state, content state, playback state, and dialog state.
- Location: `web/src/features/app/hooks/use-app-controller.ts`, `web/src/features/app/hooks/use-auth-state.ts`, `web/src/features/app/hooks/use-library-data-state.ts`, `web/src/features/app/hooks/use-playback-state.ts`, `web/src/features/app/hooks/use-source-dialog-state.ts`.
- Contains: API client creation, navigation intents, optimistic UI orchestration, dialog state machines, playback progress syncing.
- Depends on: `web/src/lib/mibo-api.ts`, `web/src/lib/client-config.ts`, TanStack Router navigation, `sonner` toasts.
- Used by: `web/src/features/app/pages/*.tsx`, `web/src/App.tsx`, `web/src/features/app/components/app-root.tsx`.

**Frontend Presentation Layer:**
- Purpose: Render feature state into shells, dialogs, settings panels, and reusable UI primitives.
- Location: `web/src/features/app/components/`, `web/src/components/`, `web/src/components/settings/`, `web/src/components/ui/`.
- Contains: route shells such as `web/src/features/app/components/browse-app-shell.tsx` and `web/src/features/app/components/settings-app-shell.tsx`, app chrome in `web/src/components/app-sidebar.tsx`, setup/auth screens, and shadcn-generated primitives under `web/src/components/ui/`.
- Depends on: controller output types from `web/src/features/app/hooks/use-app-controller.ts` and alias imports configured by `web/components.json` and `web/tsconfig.json`.
- Used by: every route and dialog surface in the SPA.

**Frontend API Boundary Layer:**
- Purpose: Keep browser/server communication in one typed client module.
- Location: `web/src/lib/mibo-api.ts`.
- Contains: request envelope handling, `ApiError`, domain types, and methods for `/api/v1/*` endpoints.
- Depends on: `fetch` and runtime base URL/token inputs.
- Used by: all feature hooks and the setup flow in `web/src/components/setup-wizard.tsx`.

**Backend Bootstrap Layer:**
- Purpose: Load config, open the database, wire services, start HTTP and worker loops.
- Location: `mibo-media-server/cmd/mibo-media-server/main.go`, `mibo-media-server/internal/app/app.go`, `mibo-media-server/internal/config/config.go`, `mibo-media-server/internal/database/database.go`.
- Contains: env-based config loading, GORM auto-migration, service construction, `http.Server`, and worker startup.
- Depends on: `mibo-media-server/internal/*` services and `gorm`.
- Used by: the executable at `mibo-media-server/cmd/mibo-media-server/main.go`.

**Backend HTTP/API Layer:**
- Purpose: Expose product capabilities over REST-style JSON endpoints.
- Location: `mibo-media-server/internal/httpapi/router.go`.
- Contains: route registration, auth checks, request decoding, response envelopes, health/readiness endpoints, media/library/source/playback/job/settings handlers.
- Depends on: service layer packages such as `internal/auth`, `internal/library`, `internal/metadata`, `internal/playback`, `internal/progress`, `internal/settings`, and `internal/jobs`.
- Used by: `web/src/lib/mibo-api.ts` and external HTTP clients.

**Backend Domain Services Layer:**
- Purpose: Implement business workflows behind HTTP handlers and worker jobs.
- Location: `mibo-media-server/internal/auth/service.go`, `internal/library/*.go`, `internal/metadata/service.go`, `internal/playback/service.go`, `internal/probe/service.go`, `internal/progress/service.go`, `internal/settings/service.go`, `internal/jobs/service.go`, `internal/search/service.go`.
- Contains: auth/session flows, media source and library management, scan/classification logic, TMDB metadata matching, playback link selection, ffprobe enrichment, progress tracking, metadata settings persistence, and background job queuing.
- Depends on: `gorm.DB`, the storage provider registry in `mibo-media-server/internal/providers/registry.go`, and cross-service collaboration through `internal/app/app.go` wiring.
- Used by: `mibo-media-server/internal/httpapi/router.go` and `mibo-media-server/internal/worker/worker.go`.

**Backend Storage Adapter Layer:**
- Purpose: Hide filesystem vs OpenList differences behind a common interface.
- Location: `mibo-media-server/internal/storage/provider.go`, `mibo-media-server/internal/storage/local/adapter.go`, `mibo-media-server/internal/storage/openlist/adapter.go`, `mibo-media-server/internal/providers/registry.go`, `mibo-media-server/internal/providers/source_config.go`.
- Contains: provider interface, provider capability metadata, path resolution, local filesystem access, OpenList HTTP calls, and source-config normalization/sanitization.
- Depends on: backend config from `mibo-media-server/internal/config/config.go` and `database.MediaSource` records from `mibo-media-server/internal/database/models.go`.
- Used by: `internal/library`, `internal/playback`, `internal/probe`, `internal/httpapi` readiness checks, and OpenList test/browse flows.

**Backend Persistence Layer:**
- Purpose: Persist libraries, media, jobs, users, sessions, progress, and system settings.
- Location: `mibo-media-server/internal/database/models.go`.
- Contains: GORM models such as `MediaSource`, `Library`, `MediaItem`, `MediaFile`, `Job`, `User`, `Session`, `PlaybackProgress`, and `SystemSetting`.
- Depends on: GORM migrations in `mibo-media-server/internal/database/database.go`.
- Used by: every backend service.

## Data Flow

**Frontend Route Boot and App Entry:**

1. `web/src/main.tsx` mounts the app with `ThemeProvider`, `TooltipProvider`, `RouterProvider`, and `Toaster`.
2. `web/src/router.tsx` checks `/api/v1/setup/status` through `createMiboApi` before letting users enter non-setup routes.
3. A route page such as `web/src/features/app/pages/home-page.tsx` calls `useAppController`, then renders `web/src/features/app/components/app-root.tsx`.
4. `web/src/features/app/components/app-root.tsx` chooses `AuthScreen`, `BrowseAppShell`, or `SettingsAppShell` from controller state.

**Authenticated Library Browsing:**

1. `web/src/features/app/hooks/use-auth-state.ts` stores the session token in `localStorage` under `TOKEN_STORAGE_KEY` from `web/src/lib/client-config.ts`.
2. `web/src/features/app/hooks/use-library-data-state.ts` uses `web/src/lib/mibo-api.ts` to load `/api/v1/me`, `/api/v1/media-sources`, `/api/v1/libraries`, `/api/v1/me/continue-watching`, `/api/v1/me/recently-played`, and `/api/v1/home/recently-added`.
3. Route-specific effects load `/api/v1/libraries/{id}/items`, `/api/v1/media-items/{id}`, and `/api/v1/media-items/{id}/progress` as needed.
4. `web/src/features/app/components/browse-app-shell.tsx` and related child components render shelves, rails, detail panels, and navigation.

**Media Source and Library Creation:**

1. `web/src/components/setup-wizard.tsx` or `web/src/features/app/hooks/use-app-controller.ts` sends source and library forms through `web/src/lib/mibo-api.ts`.
2. `mibo-media-server/internal/httpapi/router.go` delegates to `mibo-media-server/internal/library/service.go`.
3. `library.Service` validates provider config, resolves the target root through `providers.Registry`, persists `MediaSource` and `Library` rows, then queues a `sync_library` job in `mibo-media-server/internal/jobs/service.go`.
4. `mibo-media-server/internal/worker/worker.go` claims the queued job and runs `library.Service.RunSyncLibrary`.

**Library Scan and Metadata Enrichment:**

1. `mibo-media-server/internal/library/scan.go` recursively lists storage objects through a `storage.Provider`.
2. Scan logic classifies files into `MediaItem` and `MediaFile` records, resetting metadata and probe state when base file identity changes.
3. New or changed records queue `match_media_item` and `probe_media_file` jobs through `mibo-media-server/internal/jobs/service.go`.
4. `mibo-media-server/internal/worker/worker.go` dispatches metadata jobs to `mibo-media-server/internal/metadata/service.go` and probe jobs to `mibo-media-server/internal/probe/service.go`.

**Playback Flow:**

1. `web/src/features/app/hooks/use-playback-state.ts` requests `/api/v1/media-items/{id}/playback`.
2. `mibo-media-server/internal/playback/service.go` picks the best `MediaFile`, resolves storage access through the provider registry, and returns either a direct link or `/api/v1/media-files/{id}/stream`.
3. The browser video element in the player dialog reports progress back through `/api/v1/me/progress`.
4. `mibo-media-server/internal/progress/service.go` updates `PlaybackProgress`, then the frontend refreshes continue-watching and recently-played rails.

**State Management:**
- Keep frontend state local to controller/hooks in `web/src/features/app/hooks/*.ts` and browser persistence in `localStorage` via `web/src/lib/client-config.ts`.
- Keep backend state in GORM-backed tables defined in `mibo-media-server/internal/database/models.go`.
- Use the jobs table from `mibo-media-server/internal/jobs/service.go` plus the polling runner in `mibo-media-server/internal/worker/worker.go` for asynchronous work instead of in-request long-running processing.

## Key Abstractions

**App Controller:**
- Purpose: Present one composed view model for each frontend route.
- Examples: `web/src/features/app/hooks/use-app-controller.ts`, `web/src/features/app/components/app-root.tsx`, `web/src/features/app/pages/home-page.tsx`.
- Pattern: Build a single controller hook that merges auth, library, playback, settings, and dialog behaviors into shell props.

**Typed API Client:**
- Purpose: Make browser/backend contracts explicit and shared inside the frontend.
- Examples: `web/src/lib/mibo-api.ts`, `web/src/lib/client-config.ts`.
- Pattern: Define request/response types in the client file, unwrap the backend envelope there, and keep route/hooks code free of raw `fetch` calls.

**Shell + Dialog Composition:**
- Purpose: Separate layout rendering from state orchestration.
- Examples: `web/src/features/app/components/browse-app-shell.tsx`, `web/src/features/app/components/settings-app-shell.tsx`, `web/src/features/app/components/app-dialogs.tsx`.
- Pattern: Pass controller slices into large shell components instead of letting page files hold direct UI logic.

**Storage Provider Interface:**
- Purpose: Make local filesystem and OpenList look the same to domain services.
- Examples: `mibo-media-server/internal/storage/provider.go`, `mibo-media-server/internal/storage/local/adapter.go`, `mibo-media-server/internal/storage/openlist/adapter.go`, `mibo-media-server/internal/providers/registry.go`.
- Pattern: Route domain code through `storage.Provider` and `providers.Registry`; add provider-specific normalization in `internal/providers/source_config.go`.

**Database-Backed Job Queue:**
- Purpose: Run scans, metadata matching, and probing asynchronously.
- Examples: `mibo-media-server/internal/jobs/service.go`, `mibo-media-server/internal/worker/worker.go`, `mibo-media-server/internal/library/scan.go`.
- Pattern: Enqueue job rows in request-time services, then let the worker claim and execute them by `job.Kind`.

**Service-Oriented Backend Modules:**
- Purpose: Keep handlers thin and make workflows reusable from HTTP and worker paths.
- Examples: `mibo-media-server/internal/auth/service.go`, `mibo-media-server/internal/library/service.go`, `mibo-media-server/internal/metadata/service.go`, `mibo-media-server/internal/playback/service.go`, `mibo-media-server/internal/progress/service.go`, `mibo-media-server/internal/settings/service.go`.
- Pattern: Expose a `Service` per domain package with constructor injection from `mibo-media-server/internal/app/app.go`.

## Entry Points

**Frontend Application Entry:**
- Location: `web/src/main.tsx`
- Triggers: Vite loading `web/index.html`.
- Responsibilities: Mount React, install global providers, and hand control to TanStack Router.

**Frontend Route Tree:**
- Location: `web/src/router.tsx`
- Triggers: Browser navigation inside the SPA.
- Responsibilities: Define `/`, `/movies`, `/shows`, `/settings`, `/library/$libraryId`, `/media/$mediaItemId`, and `/setup`; gate non-setup routes by setup status.

**Frontend Legacy Wrapper:**
- Location: `web/src/App.tsx`
- Triggers: Direct component usage outside router-based entry.
- Responsibilities: Adapt `AppRouteState` into `useAppController` and `AppRoot`; keep route-shaped state usable as a component.

**Backend Executable:**
- Location: `mibo-media-server/cmd/mibo-media-server/main.go`
- Triggers: `go run ./cmd/mibo-media-server` or compiled binary startup.
- Responsibilities: Load config, bootstrap `app.New`, and run until signal shutdown.

**Backend Application Bootstrap:**
- Location: `mibo-media-server/internal/app/app.go`
- Triggers: `main.go` construction.
- Responsibilities: Open DB, construct service graph, install HTTP handler, and optionally start the worker loop.

**Backend HTTP Router:**
- Location: `mibo-media-server/internal/httpapi/router.go`
- Triggers: Incoming HTTP requests.
- Responsibilities: Route `/healthz`, `/readyz`, `/api/v1/auth/*`, `/api/v1/setup/status`, `/api/v1/media-sources/*`, `/api/v1/libraries/*`, `/api/v1/media-items/*`, `/api/v1/media-files/*`, `/api/v1/me/*`, `/api/v1/settings/metadata`, `/api/v1/jobs*`, and storage browse/test endpoints.

**Background Worker:**
- Location: `mibo-media-server/internal/worker/worker.go`
- Triggers: `Worker.Enabled` in `mibo-media-server/internal/config/config.go`.
- Responsibilities: Poll queued jobs and dispatch `sync_library`, `match_media_item`, and `probe_media_file` handlers.

## Error Handling

**Strategy:** Return explicit errors from services, translate them into JSON error envelopes in `mibo-media-server/internal/httpapi/router.go`, and surface them as `ApiError` in `web/src/lib/mibo-api.ts`.

**Patterns:**
- Use guard clauses in handlers and hooks, for example auth checks in `mibo-media-server/internal/httpapi/router.go` and `ApiError` branching in `web/src/features/app/hooks/use-app-controller.ts`.
- Mark asynchronous failures in persisted state instead of swallowing them, such as failed jobs in `mibo-media-server/internal/jobs/service.go` and probe status updates in `mibo-media-server/internal/probe/service.go`.

## Cross-Cutting Concerns

**Logging:** Use backend standard-library logging in `mibo-media-server/cmd/mibo-media-server/main.go`, `mibo-media-server/internal/app/app.go`, `mibo-media-server/internal/database/database.go`, and `mibo-media-server/internal/worker/worker.go`.

**Validation:** Validate env config in `mibo-media-server/internal/config/config.go`, source/provider settings in `mibo-media-server/internal/providers/source_config.go` and `mibo-media-server/internal/library/service.go`, auth credentials in `mibo-media-server/internal/auth/service.go`, and frontend form/error handling in `web/src/features/app/hooks/use-app-controller.ts` plus `web/src/components/setup-wizard.tsx`.

**Authentication:** Use Bearer session tokens from `mibo-media-server/internal/auth/service.go`, store them in browser `localStorage` through `web/src/lib/client-config.ts`, require them in authenticated handlers inside `mibo-media-server/internal/httpapi/router.go`, and let `web/src/features/app/hooks/use-auth-state.ts` clear state on 401 responses.

**UI Component System:** Use shadcn/radix-nova source components from `web/src/components/ui/` with aliases defined in `web/components.json`; keep shared product UI in `web/src/components/` and feature-specific UI in `web/src/features/app/components/`.

**Upstream Boundary:** Keep `OpenList/` as a separate upstream checkout with its own `.git`; integrate only through HTTP calls implemented in `mibo-media-server/internal/storage/openlist/adapter.go`.

---

*Architecture analysis: 2026-04-21*
