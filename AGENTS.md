# Mibo Agent Notes

## Boundaries
- Repo root is not a runnable workspace package or git repo. Work from package roots: `web/`, `mibo-media-server/`, and only touch `OpenList/` when the task is explicitly about upstream OpenList.
- `OpenList/` is a full upstream checkout with its own `.git`; `mibo-media-server` talks to OpenList over HTTP in `mibo-media-server/internal/storage/openlist/adapter.go`, not by importing code from `OpenList/`.
- `web/` is the frontend. Real entrypoints are `src/main.tsx` -> `src/router.tsx` -> `src/App.tsx`; most product behavior still lives in the large `src/App.tsx`.
- `mibo-media-server/` is the custom backend. Startup is `cmd/mibo-media-server/main.go`; service wiring is `internal/app/app.go`; HTTP routes are registered in `internal/httpapi/router.go`.
- Root `package.json` only provides the `shadcn` CLI for local tooling; do not treat it as the frontend app manifest.

## Commands
- Frontend commands run from `web/` and use `pnpm` (`web/pnpm-lock.yaml` is the real lockfile): `pnpm dev`, `pnpm typecheck`, `pnpm build`.
- `pnpm lint` currently fails on pre-existing `react-hooks/set-state-in-effect` and `react-refresh/only-export-components` issues in files like `src/App.tsx`, `src/router.tsx`, and several `src/components/ui/*`. Do not assume a lint failure is caused by your change unless you touched those files.
- Frontend formatting is Prettier, not ESLint autofix: `pnpm format` or `pnpm exec prettier --write <file>`. Config uses no semicolons, double quotes, and the Tailwind plugin.
- Backend commands run from `mibo-media-server/`: `go run ./cmd/mibo-media-server`, `go test ./...`.
- Focused backend checks: `go test ./internal/httpapi -run TestReadyz` and `go test ./internal/worker -run TestRunOnceProcessesSyncLibraryJob`.

## Runtime Quirks
- Backend defaults to `MIBO_STORAGE_PROVIDER=openlist` and `MIBO_OPENLIST_BASE_URL=http://127.0.0.1:5244`. A bare `go run ./cmd/mibo-media-server` expects a live OpenList server unless you override env.
- To use the repo's sample media instead of OpenList, start the backend with `MIBO_STORAGE_PROVIDER=local` and `MIBO_LOCAL_ROOT_PATH=/Users/atlan/Desktop/IdeaProjects/Mibo/demo-media`.
- The local storage adapter only accepts absolute paths and rejects anything outside `MIBO_LOCAL_ROOT_PATH`.
- For local manual testing in this workspace, use app login `admin` / `admin123`.
- Frontend API base defaults to `http://127.0.0.1:8080` via `VITE_API_BASE_URL`, then is overridden from localStorage key `mibo-web-api-base-url`.
- The router redirects to `/setup` until `/api/v1/setup/status` reports `can_enter_app=true`; if you change setup/auth flow, keep `web/src/router.tsx`, `web/src/components/setup-wizard.tsx`, and `web/src/lib/client-config.ts` aligned.
- Setup and source forms intentionally seed absolute demo paths under `demo-media/`; those defaults appear in both `web/src/App.tsx` and `web/src/components/setup-wizard.tsx`.

## UI And Generated Files
- `web/components.json` is shadcn-based with style `radix-nova`; generated UI primitives live under `web/src/components/ui`, and the project uses the `@/*` alias.
- Ignore generated/local state when tracing behavior or editing: `web/dist/`, `mibo-media-server/data/`, and `mibo-media-server/tmp/`.

## Tests
- Backend tests are self-contained: they spin up `httptest` OpenList/TMDB servers and fake `ffprobe` binaries, so `go test ./...` does not need real OpenList, TMDB, or `ffprobe` installed.
