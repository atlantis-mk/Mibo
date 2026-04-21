# Technology Stack

**Analysis Date:** 2026-04-21

## Languages

**Primary:**
- TypeScript - frontend SPA in `web/src/main.tsx`, `web/src/router.tsx`, and `web/src/lib/mibo-api.ts`
- Go 1.24.0 - backend service in `mibo-media-server/cmd/mibo-media-server/main.go` and `mibo-media-server/internal/app/app.go`

**Secondary:**
- CSS - Tailwind v4 theme and global styles in `web/src/index.css`
- JSON/YAML - project and tool configuration in `web/package.json`, `web/components.json`, `web/pnpm-lock.yaml`, and root `package-lock.json`

## Runtime

**Environment:**
- Node.js - required for the frontend toolchain declared in `web/package.json`; exact version not detected in `web/`
- Go 1.24.0 - declared in `mibo-media-server/go.mod`
- Browser runtime - Vite client build targets modern browser APIs via `web/tsconfig.app.json`

**Package Manager:**
- pnpm - product frontend package manager, indicated by `web/pnpm-lock.yaml`
- npm - root-only tooling lockfile for `shadcn` CLI in `package.json` and `package-lock.json`
- Lockfile: present in `web/pnpm-lock.yaml` and root `package-lock.json`

## Frameworks

**Core:**
- React 19 - UI runtime in `web/package.json` with entrypoint wiring in `web/src/main.tsx`
- Vite 7 - frontend dev server and build pipeline in `web/package.json` and `web/vite.config.ts`
- TanStack Router - client routing in `web/src/router.tsx`
- Tailwind CSS 4 - styling pipeline in `web/package.json` and `web/src/index.css`
- GORM - backend ORM and schema migration layer in `mibo-media-server/internal/database/database.go`
- Go standard `net/http` - backend HTTP stack and route registration in `mibo-media-server/internal/httpapi/router.go`

**Testing:**
- Go `testing` package - backend tests in `mibo-media-server/internal/httpapi/router_test.go`, `mibo-media-server/internal/metadata/service_test.go`, and `mibo-media-server/internal/worker/worker_test.go`
- Frontend-specific test runner - Not detected in `web/`

**Build/Dev:**
- TypeScript 5.9 - frontend typechecking in `web/package.json`, `web/tsconfig.app.json`, and `web/tsconfig.node.json`
- ESLint 9 - frontend linting in `web/eslint.config.js`
- Prettier 3 with Tailwind plugin - frontend formatting per `web/package.json` and repo notes in `AGENTS.md`
- shadcn CLI - UI source generation configured by `web/components.json` and root `package.json`

## Key Dependencies

**Critical:**
- `@tanstack/react-router` - route tree and setup gating in `web/src/router.tsx`
- `react` / `react-dom` - frontend rendering in `web/src/main.tsx`
- `tailwindcss`, `@tailwindcss/vite`, `tw-animate-css` - style system in `web/src/index.css` and `web/vite.config.ts`
- `radix-ui`, `@base-ui/react`, `vaul`, `cmdk`, `sonner` - component primitives used across `web/src/components/ui/*`
- `gorm.io/gorm` - persistence and model migration in `mibo-media-server/internal/database/database.go`
- `github.com/glebarez/sqlite` - default embedded database driver in `mibo-media-server/go.mod` and `mibo-media-server/internal/database/database.go`
- `gorm.io/driver/postgres` - optional production-style database driver in `mibo-media-server/go.mod` and `mibo-media-server/internal/database/database.go`
- `golang.org/x/crypto/bcrypt` - password hashing in `mibo-media-server/internal/auth/service.go`

**Infrastructure:**
- `swiper`, `embla-carousel-react`, and `recharts` - interactive media UI widgets in `web/src/features/app/components/home-hero-carousel.tsx`, `web/src/features/app/components/home-rail.tsx`, and `web/src/components/ui/chart.tsx`
- `next-themes` and `@fontsource-variable/geist` - theming/font setup in `web/src/components/ui/sonner.tsx`, `web/src/components/theme-provider.tsx`, and `web/src/index.css`
- `ffprobe` external binary - media inspection invoked from `mibo-media-server/internal/probe/service.go`

## Configuration

**Environment:**
- Frontend API base uses `VITE_API_BASE_URL` in `web/src/lib/client-config.ts`
- Backend runtime is entirely env-driven in `mibo-media-server/internal/config/config.go`
- Storage selection uses `MIBO_STORAGE_PROVIDER`, with `openlist` default and `local` alternative in `mibo-media-server/internal/config/config.go`
- Database selection uses `MIBO_DATABASE_DRIVER` and `MIBO_DATABASE_DSN` in `mibo-media-server/internal/config/config.go`
- Metadata providers use `MIBO_TMDB_*` and `MIBO_TVDB_*` env vars in `mibo-media-server/internal/config/config.go`
- OpenList access uses `MIBO_OPENLIST_*` env vars in `mibo-media-server/internal/config/config.go`
- `.env*` files were not detected in the workspace; runtime config is read from process env in `mibo-media-server/internal/config/config.go`

**Build:**
- Frontend build config lives in `web/vite.config.ts`, `web/tsconfig.json`, `web/tsconfig.app.json`, and `web/tsconfig.node.json`
- Frontend lint config lives in `web/eslint.config.js`
- shadcn UI generation config lives in `web/components.json`
- Backend dependency and toolchain config lives in `mibo-media-server/go.mod`

## Platform Requirements

**Development:**
- Run frontend commands from `web/` with pnpm per `AGENTS.md` and `web/package.json`
- Run backend commands from `mibo-media-server/` with Go per `AGENTS.md` and `mibo-media-server/go.mod`
- Local media probing requires an `ffprobe` binary reachable by `MIBO_FFPROBE_PATH` in `mibo-media-server/internal/config/config.go`
- Default backend startup expects a reachable OpenList server at `MIBO_OPENLIST_BASE_URL` unless storage is switched to local, per `AGENTS.md` and `mibo-media-server/internal/config/config.go`

**Production:**
- Deployment target is a standalone Go HTTP server exposing `MIBO_HTTP_ADDR` from `mibo-media-server/cmd/mibo-media-server/main.go` and `mibo-media-server/internal/app/app.go`
- Frontend is a static Vite bundle produced by `web/package.json`
- First-party CI/CD or hosting config was not detected in `web/` or `mibo-media-server/`; `OpenList/` contains upstream deployment files but is an external boundary only per `AGENTS.md`

---

*Stack analysis: 2026-04-21*
