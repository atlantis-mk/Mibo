# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository structure

- `web/` is the React frontend. The app entry is `web/src/main.tsx`, routing is defined in `web/src/router.tsx`, and feature code is organized under `web/src/features/`.
- `mibo-media-server/` is the custom Go backend. The server entrypoint is `mibo-media-server/cmd/mibo-media-server/main.go`, app/service wiring lives in `mibo-media-server/internal/app/app.go`, and HTTP routes are registered in `mibo-media-server/internal/httpapi/router.go`.
- `OpenList/` is a separate upstream checkout with its own git repo. Do not treat it as part of the main app unless the task explicitly targets OpenList.
- The repository root is a coordination layer, not the main frontend workspace. Run frontend commands from `web/` and backend commands from `mibo-media-server/`.
- The repo root does not contain the app's frontend package manifest; `web/package.json` is the real frontend workspace, and the root only carries auxiliary tooling.

## Common commands

### Frontend (`web/`)

```bash
pnpm dev
pnpm build
pnpm typecheck
pnpm lint
pnpm format
pnpm test
pnpm exec vitest run src/lib/mibo-query.test.ts
pnpm exec vitest run src/lib/mibo-query.test.ts -t "query key"
pnpm exec prettier --write src/features/settings/pages.tsx
```

### Backend (`mibo-media-server/`)

```bash
go run ./cmd/mibo-media-server
go test ./...
go test ./internal/httpapi -run TestReadyz
go test ./internal/library -run TestWorkflow
go test ./internal/library -run TestContentShape
```

### Full-stack build / embedded web bundle (repo root)

```bash
./scripts/build-with-web.sh
```

This script builds the frontend into `web/dist-static` and copies it into `mibo-media-server/internal/webui/dist` for embedding.

## Frontend architecture

- The frontend is a Vite + React 19 SPA with TanStack Router and TanStack React Query. `web/src/main.tsx` mounts `RouterProvider` inside `AppQueryProvider` and renders a global Sonner toaster.
- Routing is code-defined in `web/src/router.tsx`, not file-based. There are three main route shells:
  - authenticated app routes under `app-layout` for the end-user experience (`/`, `/library/:id`, `/media/:id`, `/person/:id`, `/favorites`, `/search`)
  - authenticated `/settings` routes for the admin/settings workspace
  - standalone `/play/:id`, `/login`, and `/setup` routes
- Auth and setup gating are enforced in router `beforeLoad` hooks through `requireCanEnterApp`, `requireSetupAccess`, and the auth hydration wait path. If you change setup or login flow, keep `web/src/router.tsx`, `web/src/lib/setup-gate.ts`, and `web/src/features/setup/index.tsx` aligned.
- `web/src/stores/auth-store.ts` is the main global client state. It uses Zustand `persist` to store the bearer token and current user in local storage, plus a `hasHydrated` flag used by the router before protected redirects.
- Most application data is server state managed through React Query. Query keys and reusable `queryOptions` factories live in `web/src/lib/mibo-query.ts`; feature code usually consumes these factories instead of defining ad hoc fetch logic everywhere.
- API access is centralized in `web/src/lib/mibo-api.ts`. `createMiboApi` is the typed client boundary, `getApiBaseUrl()` reads `VITE_API_BASE_URL`, and unauthorized responses clear the session and redirect to `/login`.
- UI is feature-first under `web/src/features/`. Shared UI primitives live under `web/src/components/ui/`, and the project uses shadcn with the `radix-nova` preset from `web/components.json`. Path aliases `@/*` and `#/*` both resolve to `web/src`.
- The settings area is effectively its own sub-application. `web/src/features/settings/sections.ts` defines the left-nav/section registry, while `web/src/features/settings/pages.tsx` lazy-loads the concrete settings panels. When adding or moving settings pages, keep the router, section registry, and lazy page exports in sync.
- The sidebar/app shell is shared UI, but product behavior is mostly concentrated in feature folders such as `home`, `library`, `media`, `play`, `search`, `jobs`, `schedules`, `health`, and `metadata-governance`.

## Backend architecture

- `mibo-media-server` starts in `cmd/mibo-media-server/main.go`, loads config, builds the application container in `internal/app/app.go`, and then runs the HTTP server plus background workers/listeners.
- `internal/app/app.go` is the composition root. It opens the database, builds the provider registry, constructs services such as auth/catalog/library/metadata/playback/search/settings/health, wires workflow handlers, and passes them into `httpapi.New(...)`.
- `internal/httpapi/router.go` is the HTTP surface area. It registers health/setup/auth endpoints, admin console endpoints, library/media-source/settings endpoints, catalog and playback routes, governance routes, schedules, workflows, and the embedded web UI handler.
- The backend is service-oriented by domain. The most important package boundaries are:
  - `internal/library`: library management, scans, classification, cleanup, and scan policy behavior
  - `internal/catalog`: catalog item read models and item/person/playback-facing domain operations
  - `internal/metadata`: metadata provider orchestration, governance, matching, field/image policies
  - `internal/playback`: playback source/link resolution
  - `internal/probe`: ffprobe-driven technical metadata enrichment
  - `internal/schedule`, `internal/workflow`, `internal/jobs`: background execution and orchestration
  - `internal/listener`: local/OpenList storage event observers
  - `internal/settings`: persisted application settings and metadata profile config
  - `internal/providers` plus `internal/storage/*`: storage/provider registry and concrete backends
- The backend serves the frontend from embedded static files via `internal/webui/embed.go`, and `internal/httpapi/router.go` mounts that embedded app at `/`.

## Runtime and integration notes

- The frontend talks to the Go backend over `/api/v1/*`. `VITE_API_BASE_URL` can override the API origin for frontend development.
- For local manual testing in this workspace, use app login `admin` / `admin123`.
- For local development against the sample media in this repo, set `MIBO_LOCAL_ROOT_PATH=/Users/atlan/Desktop/IdeaProjects/Mibo/demo-media` before starting the backend.
- The local storage adapter only accepts absolute paths and rejects anything outside `MIBO_LOCAL_ROOT_PATH`.
- For local development, the frontend API base defaults to `http://127.0.0.1:8080` and can also be overridden from local storage key `mibo-web-api-base-url`.
- For local development, the backend README documents environment variables for HTTP, database, OpenList, TMDB, ffprobe, worker, and workflow configuration.
- The backend integrates with OpenList over HTTP through provider/storage layers; it does not import application code from the sibling `OpenList/` checkout.
- Current product flow is catalog-first. New frontend/backend work should prefer catalog endpoints such as `/api/v1/items/*`, `/api/v1/assets/*`, and `/api/v1/inventory-files/*` instead of building new behavior on legacy media item/file routes.
- Background processing is split between schedule-driven work and workflow DAG execution. `internal/app/app.go` configures workflow resource budgets differently for SQLite versus other databases, then starts the workflow runner and storage listeners when workers are enabled.
- Ignore generated/local state when tracing behavior or editing: `web/dist/`, `mibo-media-server/data/`, and `mibo-media-server/tmp/`.

## Testing notes

- Frontend tests use Vitest and currently live close to feature/lib modules under `web/src/`.
- Frontend formatting is Prettier-based, not ESLint autofix; use `pnpm format` or `pnpm exec prettier --write <file>` when needed.
- Backend tests are spread across domain packages under `mibo-media-server/internal/...` and are designed to run with `go test ./...` without requiring real OpenList, TMDB, or ffprobe installations.
- When validating backend behavior, prefer targeted package tests first because the repo has broad test coverage across `httpapi`, `library`, `catalog`, `metadata`, `playback`, `probe`, `schedule`, and `workflow`.
- `pnpm lint` may report pre-existing React Hooks or React Refresh issues in migrated UI files; do not assume every lint failure was introduced by the current change.
