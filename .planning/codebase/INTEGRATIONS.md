# External Integrations

**Analysis Date:** 2026-04-21

## APIs & External Services

**Internal app-to-app API:**
- Mibo backend HTTP API - consumed by the frontend for auth, setup, libraries, playback, progress, and metadata workflows
  - SDK/Client: typed `fetch` wrapper in `web/src/lib/mibo-api.ts`
  - Auth: Bearer session token stored under `TOKEN_STORAGE_KEY` in `web/src/lib/client-config.ts`

**Storage provider:**
- OpenList - remote filesystem listing, object metadata lookup, and direct file link generation for media sources
  - SDK/Client: custom HTTP adapter in `mibo-media-server/internal/storage/openlist/adapter.go`
  - Auth: `MIBO_OPENLIST_TOKEN` or `MIBO_OPENLIST_USERNAME` + `MIBO_OPENLIST_PASSWORD` from `mibo-media-server/internal/config/config.go`

**Metadata providers:**
- The Movie Database (TMDB) - active metadata search and detail lookup for media matching in `mibo-media-server/internal/metadata/service.go`
  - SDK/Client: custom `net/http` client in `mibo-media-server/internal/metadata/service.go`
  - Auth: `MIBO_TMDB_API_KEY` from `mibo-media-server/internal/config/config.go`, with runtime overrides persisted by `mibo-media-server/internal/settings/service.go`
- TheTVDB (TVDB) - configuration surface exists, but active API calls are not implemented; settings are exposed in `mibo-media-server/internal/settings/service.go` and `mibo-media-server/internal/httpapi/router.go`
  - SDK/Client: Not detected
  - Auth: `MIBO_TVDB_API_KEY` from `mibo-media-server/internal/config/config.go`, with runtime overrides persisted by `mibo-media-server/internal/settings/service.go`

**Upstream boundary:**
- OpenList upstream checkout in `OpenList/` - present in the workspace but consumed only over HTTP from `mibo-media-server/internal/storage/openlist/adapter.go`; there is no direct code import boundary per `AGENTS.md`
  - SDK/Client: Not applicable
  - Auth: Not applicable to first-party code

## Data Storage

**Databases:**
- SQLite - default application database selected in `mibo-media-server/internal/config/config.go`
  - Connection: `MIBO_DATABASE_DSN`
  - Client: GORM via `mibo-media-server/internal/database/database.go`
- PostgreSQL - optional relational database selected in `mibo-media-server/internal/config/config.go`
  - Connection: `MIBO_DATABASE_DSN`
  - Client: GORM via `mibo-media-server/internal/database/database.go`

**File Storage:**
- OpenList - remote storage provider with link-capable access in `mibo-media-server/internal/storage/openlist/adapter.go`
- Local filesystem - absolute-path constrained storage provider in `mibo-media-server/internal/storage/local/adapter.go`

**Caching:**
- None - no Redis, Memcached, or application cache service was detected in `web/` or `mibo-media-server/`

## Authentication & Identity

**Auth Provider:**
- Custom - username/password auth backed by the application database in `mibo-media-server/internal/auth/service.go`
  - Implementation: passwords hashed with bcrypt and session tokens stored as SHA-256 hashes in `mibo-media-server/internal/auth/service.go` and `mibo-media-server/internal/database/models.go`

## Monitoring & Observability

**Error Tracking:**
- None - no Sentry, Rollbar, or similar service was detected in `web/` or `mibo-media-server/`

**Logs:**
- Backend request and startup logs use the Go standard `log` package in `mibo-media-server/cmd/mibo-media-server/main.go`, `mibo-media-server/internal/app/app.go`, and `mibo-media-server/internal/httpapi/router.go`
- Database logs use GORM warn-level logger setup in `mibo-media-server/internal/database/database.go`

## CI/CD & Deployment

**Hosting:**
- Frontend hosting target is a static bundle produced from `web/package.json`; concrete platform config was not detected in `web/`
- Backend hosting target is a standalone Go server from `mibo-media-server/cmd/mibo-media-server/main.go`; concrete platform config was not detected in `mibo-media-server/`

**CI Pipeline:**
- None detected for first-party apps in `web/` or `mibo-media-server/`
- `OpenList/.github/workflows/*.yml` is upstream-only and outside the first-party deployment surface per `AGENTS.md`

## Environment Configuration

**Required env vars:**
- `VITE_API_BASE_URL` for frontend backend targeting in `web/src/lib/client-config.ts`
- `MIBO_HTTP_ADDR`, `MIBO_HTTP_SHUTDOWN_TIMEOUT`, and `MIBO_CORS_ALLOWED_ORIGINS` in `mibo-media-server/internal/config/config.go`
- `MIBO_STORAGE_PROVIDER`, `MIBO_LOCAL_ROOT_PATH`, `MIBO_OPENLIST_BASE_URL`, `MIBO_OPENLIST_USERNAME`, `MIBO_OPENLIST_PASSWORD`, `MIBO_OPENLIST_TOKEN`, `MIBO_OPENLIST_ROOT_PATH`, `MIBO_OPENLIST_TIMEOUT`, and `MIBO_OPENLIST_INSECURE_SKIP_VERIFY` in `mibo-media-server/internal/config/config.go`
- `MIBO_DATABASE_DRIVER` and `MIBO_DATABASE_DSN` in `mibo-media-server/internal/config/config.go`
- `MIBO_TMDB_API_KEY`, `MIBO_TMDB_BASE_URL`, `MIBO_TMDB_IMAGE_BASE_URL`, `MIBO_TMDB_LANGUAGE`, `MIBO_TMDB_TIMEOUT`, `MIBO_TVDB_API_KEY`, `MIBO_TVDB_BASE_URL`, `MIBO_TVDB_LANGUAGE`, and `MIBO_TVDB_TIMEOUT` in `mibo-media-server/internal/config/config.go`
- `MIBO_FFPROBE_ENABLED`, `MIBO_FFPROBE_PATH`, `MIBO_FFPROBE_TIMEOUT`, `MIBO_WORKER_ENABLED`, and `MIBO_WORKER_POLL_INTERVAL` in `mibo-media-server/internal/config/config.go`

**Secrets location:**
- Process environment for backend startup secrets in `mibo-media-server/internal/config/config.go`
- Database-backed `system_settings` records for metadata API key overrides in `mibo-media-server/internal/settings/service.go` and `mibo-media-server/internal/database/models.go`
- Browser localStorage for frontend session token and API base override in `web/src/lib/client-config.ts` and `web/src/features/app/hooks/use-app-controller.ts`

## Webhooks & Callbacks

**Incoming:**
- None - no webhook or callback endpoints were detected in `mibo-media-server/internal/httpapi/router.go`

**Outgoing:**
- OpenList HTTP calls to `/api/auth/login`, `/api/fs/list`, `/api/fs/get`, and `/api/fs/link` in `mibo-media-server/internal/storage/openlist/adapter.go`
- TMDB HTTP GET requests for search and detail lookup in `mibo-media-server/internal/metadata/service.go`
- Frontend browser requests to `/api/v1/*` backend endpoints in `web/src/lib/mibo-api.ts`

---

*Integration audit: 2026-04-21*
