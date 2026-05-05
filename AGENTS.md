# Mibo Agent Notes

## Boundaries
- Repo root is not a runnable workspace package or git repo. Work from package roots: `web/`, `mibo-media-server/`, and only touch `OpenList/` when the task is explicitly about upstream OpenList.
- `OpenList/` is a full upstream checkout with its own `.git`; `mibo-media-server` talks to OpenList over HTTP in `mibo-media-server/internal/storage/openlist/adapter.go`, not by importing code from `OpenList/`.
- `web/` is the frontend. It is a Vite SPA with entrypoints `src/main.tsx` -> `src/router.tsx`; product behavior lives under `src/features/` with shared UI in `src/components/`.
- `mibo-media-server/` is the custom backend. Startup is `cmd/mibo-media-server/main.go`; service wiring is `internal/app/app.go`; HTTP routes are registered in `internal/httpapi/router.go`.
- Root `package.json` only provides the `shadcn` CLI for local tooling; do not treat it as the frontend app manifest.

## Commands
- Frontend commands run from `web/` and use `pnpm` (`web/pnpm-lock.yaml` is the real lockfile): `pnpm dev`, `pnpm typecheck`, `pnpm build`.
- `pnpm lint` may report pre-existing React Hooks or React Refresh issues in migrated UI files. Do not assume a lint failure is caused by your change unless you touched those files.
- Frontend formatting is Prettier, not ESLint autofix: `pnpm format` or `pnpm exec prettier --write <file>`. Config uses no semicolons, double quotes, and the Tailwind plugin.
- Backend commands run from `mibo-media-server/`: `go run ./cmd/mibo-media-server`, `go test ./...`.
- Focused backend checks: `go test ./internal/httpapi -run TestReadyz` and `go test ./internal/worker -run TestRunOnceProcessesSyncLibraryJob`.

## Runtime Quirks
- Storage is configured through user-added media sources; the backend no longer probes a global default storage provider on startup or readiness checks.
- To use the repo's sample media with a local media source, set `MIBO_LOCAL_ROOT_PATH=/Users/atlan/Desktop/IdeaProjects/Mibo/demo-media` before starting the backend.
- The local storage adapter only accepts absolute paths and rejects anything outside `MIBO_LOCAL_ROOT_PATH`.
- For local manual testing in this workspace, use app login `admin` / `admin123`.
- Frontend API base defaults to `http://127.0.0.1:8080` via `VITE_API_BASE_URL`, then is overridden from localStorage key `mibo-web-api-base-url`.
- Catalog reads no longer treat mere catalog presence as a successful cutover gate. Without an explicit `catalog_read_enabled=true`, they default to enabled only after `catalog_validation_completed_at` or legacy cleanup has been recorded; legacy-only databases still stay disabled.
- When catalog reads are enabled, legacy `/api/v1/media-items/*` and `/api/v1/media-files/*` paths are retired compatibility routes; use `/api/v1/items/*`, `/api/v1/assets/*`, and `/api/v1/inventory-files/*` instead.
- Admin maintenance endpoints exist for rollout verification: `GET /api/v1/catalog-migration/consistency` and `POST /api/v1/catalog-migration/rebuild-projections`.
- Recommended rollout check: run `GET /api/v1/catalog-migration/consistency`, confirm zero drift or an understood sample set, then run `POST /api/v1/catalog-migration/rebuild-projections` before rechecking if drift is reported.
- Governance repair endpoints now include item-scoped asset correction paths: `POST /api/v1/items/{id}/governance/assets/{asset_id}/links` and `DELETE /api/v1/items/{id}/governance/assets/{asset_id}/links/{target_item_id}`. They are intentionally bounded to the current workspace item and its descendants.
- Legacy fallback expectation: once `catalog_read_enabled` is on, old media read routes should be treated as bounded retirement shims that can return `410 Gone`; do not build new product flows on top of `/api/v1/media-items/*` or `/api/v1/media-files/*`.
- The router redirects to `/setup` until `/api/v1/setup/status` reports `can_enter_app=true`; if you change setup/auth flow, keep `web/src/router.tsx`, `web/src/features/setup/index.tsx`, and `web/src/lib/setup-gate.ts` aligned.
- Setup and source forms intentionally seed absolute demo paths under `demo-media/`; keep those defaults aligned in setup and media-source settings UI.

## UI And Generated Files
- `web/components.json` is shadcn-based with style `radix-nova`; generated UI primitives live under `web/src/components/ui`, and the project supports both `#/*` and `@/*` aliases.
- Ignore generated/local state when tracing behavior or editing: `web/dist/`, `mibo-media-server/data/`, and `mibo-media-server/tmp/`.

## Tests
- Backend tests are self-contained: they spin up `httptest` OpenList/TMDB servers and fake `ffprobe` binaries, so `go test ./...` does not need real OpenList, TMDB, or `ffprobe` installed.
