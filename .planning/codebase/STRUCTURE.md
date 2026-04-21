# Codebase Structure

**Analysis Date:** 2026-04-21

## Directory Layout

```text
[project-root]/
├── web/                     # Owned frontend SPA package
│   ├── src/                 # Frontend source code
│   ├── components.json      # shadcn/radix-nova project config
│   ├── package.json         # Frontend manifest and scripts
│   └── vite.config.ts       # Frontend build and alias config
├── mibo-media-server/       # Owned backend Go service package
│   ├── cmd/                 # Executable entrypoint
│   ├── internal/            # Backend implementation packages
│   ├── go.mod               # Backend module definition
│   └── README.md            # Backend package notes
├── OpenList/                # Upstream boundary checkout, not owned implementation
├── demo-media/              # Local sample media for manual runs
├── docs/                    # Product docs and architecture notes
├── .planning/codebase/      # Generated codebase map documents
├── AGENTS.md                # Repo-specific agent operating notes
└── package.json             # Root tooling manifest for shadcn CLI only
```

## Directory Purposes

**`web/`:**
- Purpose: Hold the browser client package.
- Contains: Vite config in `web/vite.config.ts`, TypeScript config in `web/tsconfig.json`, app manifest in `web/package.json`, and source code in `web/src/`.
- Key files: `web/package.json`, `web/components.json`, `web/vite.config.ts`, `web/src/main.tsx`, `web/src/router.tsx`.

**`web/src/`:**
- Purpose: Hold all frontend source files.
- Contains: route bootstrap files, feature modules, shared components, hooks, CSS, and API/client utilities.
- Key files: `web/src/main.tsx`, `web/src/router.tsx`, `web/src/App.tsx`, `web/src/index.css`, `web/src/lib/mibo-api.ts`.

**`web/src/features/app/`:**
- Purpose: Hold product-specific app behavior.
- Contains: route pages in `web/src/features/app/pages/`, controller/state hooks in `web/src/features/app/hooks/`, shared feature constants in `web/src/features/app/constants.ts`, and feature shells/dialogs in `web/src/features/app/components/`.
- Key files: `web/src/features/app/hooks/use-app-controller.ts`, `web/src/features/app/components/app-root.tsx`, `web/src/features/app/pages/home-page.tsx`.

**`web/src/components/`:**
- Purpose: Hold shared non-feature frontend components.
- Contains: app chrome such as `web/src/components/app-sidebar.tsx`, setup flow in `web/src/components/setup-wizard.tsx`, settings panels in `web/src/components/settings/`, and shadcn primitives in `web/src/components/ui/`.
- Key files: `web/src/components/setup-wizard.tsx`, `web/src/components/app-sidebar.tsx`, `web/src/components/settings/settings-shell.tsx`.

**`web/src/components/ui/`:**
- Purpose: Hold reusable UI primitives added from shadcn.
- Contains: source components such as `web/src/components/ui/button.tsx`, `web/src/components/ui/dialog.tsx`, `web/src/components/ui/sidebar.tsx`, and `web/src/components/ui/empty.tsx`.
- Key files: `web/src/components/ui/sidebar.tsx`, `web/src/components/ui/button.tsx`, `web/src/components/ui/sonner.tsx`.

**`web/src/lib/`:**
- Purpose: Hold shared frontend utilities and backend boundary code.
- Contains: the API client in `web/src/lib/mibo-api.ts`, config persistence in `web/src/lib/client-config.ts`, and helpers in `web/src/lib/utils.ts`.
- Key files: `web/src/lib/mibo-api.ts`, `web/src/lib/client-config.ts`, `web/src/lib/utils.ts`.

**`mibo-media-server/`:**
- Purpose: Hold the backend Go module.
- Contains: executable entrypoint under `mibo-media-server/cmd/`, implementation packages under `mibo-media-server/internal/`, runtime data under `mibo-media-server/data/`, and local logs/tmp state under `mibo-media-server/tmp/`.
- Key files: `mibo-media-server/go.mod`, `mibo-media-server/cmd/mibo-media-server/main.go`, `mibo-media-server/internal/app/app.go`.

**`mibo-media-server/internal/`:**
- Purpose: Hold private backend packages.
- Contains: bootstrapping, HTTP handlers, database code, domain services, providers, worker logic, and storage adapters.
- Key files: `mibo-media-server/internal/httpapi/router.go`, `mibo-media-server/internal/library/service.go`, `mibo-media-server/internal/database/models.go`, `mibo-media-server/internal/worker/worker.go`.

**`mibo-media-server/internal/storage/`:**
- Purpose: Hold provider interface and concrete storage backends.
- Contains: the provider contract in `mibo-media-server/internal/storage/provider.go`, local adapter in `mibo-media-server/internal/storage/local/adapter.go`, and OpenList adapter in `mibo-media-server/internal/storage/openlist/adapter.go`.
- Key files: `mibo-media-server/internal/storage/provider.go`, `mibo-media-server/internal/storage/local/adapter.go`, `mibo-media-server/internal/storage/openlist/adapter.go`.

**`mibo-media-server/internal/providers/`:**
- Purpose: Hold storage provider registry and persisted source-config normalization.
- Contains: provider registry construction in `mibo-media-server/internal/providers/registry.go` and source config parsing/sanitizing in `mibo-media-server/internal/providers/source_config.go`.
- Key files: `mibo-media-server/internal/providers/registry.go`, `mibo-media-server/internal/providers/source_config.go`.

**`OpenList/`:**
- Purpose: Hold an upstream checkout boundary only.
- Contains: OpenList's own Go module, server code, assets, and its own `.git/` history.
- Key files: document this only as an external system boundary; do not place owned Mibo implementation here.

**`demo-media/`:**
- Purpose: Hold sample media trees for local manual testing.
- Contains: local filesystem content consumed when `MIBO_STORAGE_PROVIDER=local` and `MIBO_LOCAL_ROOT_PATH` points here.
- Key files: not code; use as runtime fixture data only.

## Key File Locations

**Entry Points:**
- `web/src/main.tsx`: Browser bootstrap and provider installation.
- `web/src/router.tsx`: SPA route tree and setup gating.
- `web/src/App.tsx`: Legacy wrapper that adapts `AppRouteState` into `AppRoot`.
- `mibo-media-server/cmd/mibo-media-server/main.go`: Backend process entry.
- `mibo-media-server/internal/app/app.go`: Backend service graph and server startup.

**Configuration:**
- `package.json`: Root tooling manifest; use for shadcn CLI only, not product runtime.
- `web/package.json`: Frontend scripts and dependencies.
- `web/components.json`: shadcn/radix-nova config and alias definitions.
- `web/vite.config.ts`: `@` alias and Vite plugins.
- `web/tsconfig.json`: frontend path mapping for `@/*`.
- `mibo-media-server/internal/config/config.go`: backend env configuration.
- `mibo-media-server/go.mod`: backend module dependencies.

**Core Logic:**
- `web/src/features/app/hooks/use-app-controller.ts`: main frontend orchestration layer.
- `web/src/features/app/hooks/use-library-data-state.ts`: frontend data loading and route-aware selection.
- `web/src/features/app/hooks/use-playback-state.ts`: frontend playback/progress loop.
- `web/src/lib/mibo-api.ts`: frontend HTTP contract layer.
- `mibo-media-server/internal/httpapi/router.go`: backend endpoint layer.
- `mibo-media-server/internal/library/service.go`: media source and library lifecycle.
- `mibo-media-server/internal/library/scan.go`: scanning/classification/job enqueue logic.
- `mibo-media-server/internal/metadata/service.go`: TMDB matching and manual apply flow.
- `mibo-media-server/internal/playback/service.go`: playback source/link selection.
- `mibo-media-server/internal/probe/service.go`: ffprobe enrichment.
- `mibo-media-server/internal/progress/service.go`: user playback progress.

**Testing:**
- `mibo-media-server/internal/httpapi/router_test.go`: backend HTTP integration coverage.
- `mibo-media-server/internal/worker/worker_test.go`: worker/job execution coverage.
- `mibo-media-server/internal/metadata/service_test.go`: metadata service coverage.
- `web/`: No frontend test config or test directories detected.

## Naming Conventions

**Files:**
- Frontend route/page files use kebab-case with role suffixes: `web/src/features/app/pages/home-page.tsx`, `web/src/features/app/components/browse-app-shell.tsx`, `web/src/features/app/hooks/use-auth-state.ts`.
- Frontend shared UI primitives use lowercase kebab-case names matching the component: `web/src/components/ui/button.tsx`, `web/src/components/ui/sidebar.tsx`.
- Backend packages use lowercase domain directories with a primary `service.go` where applicable: `mibo-media-server/internal/auth/service.go`, `mibo-media-server/internal/progress/service.go`.
- Backend executables live under `cmd/<binary-name>/main.go`: `mibo-media-server/cmd/mibo-media-server/main.go`.

**Directories:**
- Frontend domain directories group by responsibility: `web/src/features/app/components/`, `web/src/features/app/hooks/`, `web/src/features/app/pages/`.
- Backend directories group by bounded context under `mibo-media-server/internal/`: `auth/`, `library/`, `metadata/`, `playback/`, `progress/`, `settings/`, `storage/`, `worker/`.
- Storage implementations live under provider-specific subdirectories: `mibo-media-server/internal/storage/local/`, `mibo-media-server/internal/storage/openlist/`.

## Where to Add New Code

**New Frontend Feature:**
- Primary code: add feature-specific hooks, pages, and components under `web/src/features/app/`.
- Route wiring: add or update paths in `web/src/router.tsx`.
- Shared API calls/types: extend `web/src/lib/mibo-api.ts`.
- Tests: no frontend test location is established; if adding tests, keep them inside `web/` and align with package-local tooling rather than the repo root.

**New Frontend Component/Module:**
- Feature-specific implementation: `web/src/features/app/components/`.
- Shared product component: `web/src/components/`.
- Shared settings UI: `web/src/components/settings/`.
- Reusable primitive sourced from shadcn: `web/src/components/ui/` and update `web/components.json`-compatible imports.

**New Backend Endpoint or Workflow:**
- HTTP handler and route registration: `mibo-media-server/internal/httpapi/router.go`.
- Domain/service logic: matching package under `mibo-media-server/internal/`, such as `internal/library/` or `internal/settings/`.
- Shared persistence model updates: `mibo-media-server/internal/database/models.go`.
- Background execution: enqueue in `mibo-media-server/internal/jobs/service.go` and dispatch in `mibo-media-server/internal/worker/worker.go`.

**New Storage Provider or Adapter:**
- Provider contract additions: `mibo-media-server/internal/storage/provider.go`.
- Adapter implementation: new directory beside `mibo-media-server/internal/storage/local/` and `mibo-media-server/internal/storage/openlist/`.
- Provider registration and config normalization: `mibo-media-server/internal/providers/registry.go` and `mibo-media-server/internal/providers/source_config.go`.

**Utilities:**
- Frontend shared helpers: `web/src/lib/` or `web/src/hooks/` if hook-based.
- Backend shared helpers: keep them close to the owning domain package inside `mibo-media-server/internal/` instead of creating a broad shared misc package.

## Special Directories

**`web/dist/`:**
- Purpose: built frontend assets.
- Generated: Yes.
- Committed: No; treat as build output.

**`mibo-media-server/data/`:**
- Purpose: local backend runtime database files such as `mibo-media-server/data/mibo.db`.
- Generated: Yes.
- Committed: No; treat as local runtime state.

**`mibo-media-server/tmp/`:**
- Purpose: local logs, smoke-test databases, and temporary runtime artifacts.
- Generated: Yes.
- Committed: No; treat as disposable local state.

**`web/node_modules/` and `node_modules/`:**
- Purpose: installed package dependencies.
- Generated: Yes.
- Committed: No.

**`.planning/codebase/`:**
- Purpose: generated architecture and quality mapping documents for GSD workflows.
- Generated: Yes.
- Committed: Yes when workflow requires updated planning docs.

**`OpenList/`:**
- Purpose: external upstream checkout boundary.
- Generated: No.
- Committed: Yes as vendored/upstream source, but do not use it as the location for owned Mibo feature work unless a task explicitly targets upstream OpenList.

---

*Structure analysis: 2026-04-21*
