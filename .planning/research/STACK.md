# Technology Stack

**Project:** Mibo
**Dimension:** Stack research for household media server
**Researched:** 2026-04-21
**Question:** What should be added or reinforced next for a self-hosted household media server built around a dedicated media business service plus storage-provider adapters?

## Executive Recommendation

The standard 2025 stack for this architecture is **a Go media business service on PostgreSQL, with a PostgreSQL-backed job system, ffprobe/ffmpeg workers, an HTTP storage gateway adapter (OpenList first), React clients, and optional Redis only after real cache/coordination pressure appears**.

For **Mibo specifically**, the prescriptive move is:

1. **Keep Go as the media core**; do not rewrite the backend.
2. **Promote PostgreSQL to the default production database**; keep SQLite only for demo/dev/single-user evaluation.
3. **Add a real background job system now** using **River** on PostgreSQL instead of inventing ad-hoc worker orchestration.
4. **Keep OpenList as a gateway boundary**, not as the business core.
5. **Keep direct-play first, ffmpeg fallback second**.
6. **Add observability now** with OpenTelemetry + Prometheus before scaling worker complexity.
7. **Do not add Redis as the first queue**; introduce Redis later only for cache/pubsub/rate smoothing if profiling justifies it.

---

## Recommended Stack

### Core Media Service

| Technology | Version | Role | Recommendation | Confidence | Why |
|------------|---------|------|----------------|------------|-----|
| Go | 1.24.x runtime, plan upgrade path to 1.25.x | Main media business service | **Keep and reinforce** | MEDIUM | Go is the right fit for always-on self-hosted media services: low memory, simple deployment, strong concurrency for scan/probe/transcode orchestration, and good HTTP/process integration. Mibo already uses Go, so rewriting would burn roadmap time without architectural benefit. |
| net/http | stdlib | HTTP API surface | **Keep** | HIGH | For this product shape, stdlib HTTP is enough. It keeps dependencies low and fits a stable internal service with explicit routing and middleware. |
| OpenList over HTTP | Pin to a tested release in deployment docs | Storage gateway boundary | **Keep as first adapter, not core runtime** | MEDIUM | It solves heterogeneous storage access now. The correct architectural move is to keep it behind `StorageProvider`, so Mibo owns media semantics while OpenList owns file access. |

### Data Layer

| Technology | Version | Purpose | Recommendation | Confidence | Why |
|------------|---------|---------|----------------|------------|-----|
| PostgreSQL | 17.x or 18.x (current docs: 18.3) | Primary production database | **Make default for production** | HIGH | PostgreSQL is the standard durable core for media metadata, user progress, jobs, idempotency, and future search/filter growth. It is a better long-term fit than SQLite once you add real workers, concurrent clients, and job orchestration. |
| SQLite | 3.x via embedded driver | Local dev / demo / tiny installs | **Keep only as non-default lightweight mode** | MEDIUM | SQLite is excellent for demos and very small single-node installs, but it becomes the wrong default once background workers and higher write concurrency are first-class. |
| GORM | v1.31.1 | ORM / migrations / model access | **Keep for now; tighten usage patterns** | MEDIUM | Mibo already uses GORM. The next step is not replacement, but discipline: keep DB access explicit, avoid magical preload sprawl, and separate media-domain services from storage adapters. |
| pgx | v5.9.2 | Native PostgreSQL driver/pool | **Add explicitly for worker/job path** | HIGH | River is built around PostgreSQL workflows, and pgx is the standard Go Postgres transport for high-concurrency service code. Use it where queue/transaction semantics matter. |

### Background Jobs and Worker Orchestration

| Technology | Version | Purpose | Recommendation | Confidence | Why |
|------------|---------|---------|----------------|------------|-----|
| River | v0.35.0 | Persistent PostgreSQL-backed job queue | **Add now** | HIGH | This is the most important missing reinforcement. River gives durable jobs, retries, scheduling, and transactional enqueue on the same PostgreSQL system of record. That matches Mibo’s “API + Worker, simple deployment first” architecture much better than a Redis-first queue. |
| Internal worker processes | in-repo | Scanner / matcher / ffprobe / transcode executors | **Keep, but run them behind River jobs** | HIGH | The architecture already wants slow work off the request path. River should become the orchestration spine; existing worker logic should become typed job handlers instead of bespoke dispatch logic. |

### Playback and Media Processing

| Technology | Version | Purpose | Recommendation | Confidence | Why |
|------------|---------|---------|----------------|------------|-----|
| ffprobe | system package, pin OS package/image version | Media stream inspection | **Reinforce now** | HIGH | Stable playback decisions depend on accurate codec/container/subtitle/audio metadata. `ffprobe` should be a required worker tool, not an optional afterthought. |
| ffmpeg | system package, pin OS package/image version | HLS/transcode fallback | **Keep as fallback path** | HIGH | Household media servers win by direct play first. But ffmpeg is still the standard fallback for incompatible codecs, containers, subtitle burn-in, and bitrate adaptation. |
| HLS output | standard protocol | Fallback streaming format | **Use as the default transcode output** | MEDIUM | HLS remains the safest multi-client fallback for browsers, mobile, and TV surfaces. It reduces client fragmentation when direct links fail. |

### Observability and Operations

| Technology | Version | Purpose | Recommendation | Confidence | Why |
|------------|---------|---------|----------------|------------|-----|
| OpenTelemetry Go | v1.43.0 | Tracing + metrics instrumentation | **Add now** | HIGH | Once API and worker paths split, you need trace continuity across request → enqueue → worker → storage adapter → ffprobe/ffmpeg. OTel is the standard instrumentation layer. |
| Prometheus client_golang | v1.23.2 | Service metrics export | **Add now** | HIGH | Queue depth, scan latency, OpenList call latency, probe duration, transcode failures, and playback-link generation success should all be first-class metrics. Prometheus remains the standard self-hosted metrics stack. |
| Structured logging (current logger or zap/zerolog if upgraded later) | If changing: zap or zerolog latest stable | Correlatable logs | **Reinforce, do not over-rotate** | LOW | The key need is structured logs with request/job IDs. The exact logger matters less than trace/job correlation. |

### Optional Runtime Additions

| Technology | Version | Purpose | When to Add | Confidence | Why |
|------------|---------|---------|-------------|------------|-----|
| Redis | 7.x OSS; Go client `go-redis` v9.18.0 | Cache / pubsub / rate smoothing | **Only after real pressure appears** | HIGH | Redis is useful for ephemeral link caches, hot browse caches, or lightweight fan-out, but it should not be introduced before PostgreSQL jobs and core observability are in place. |
| go-redis | v9.18.0 | Official Go Redis client | **Only if Redis is added** | HIGH | Official client, supports single node, Sentinel, and Cluster. Good fit if Redis becomes necessary later. |

### Web / Multi-Client Access

| Technology | Version | Purpose | Recommendation | Confidence | Why |
|------------|---------|---------|----------------|------------|-----|
| React | 19.2.5 current | Web client foundation | **Keep** | MEDIUM | Mibo already has a React SPA. The near-term problem is not framework choice; it is stabilizing API contracts for Web/mobile/TV. |
| Vite | 8.0.9 current | Frontend build/dev | **Keep** | MEDIUM | Fast local iteration, simple deployment, no strategic reason to migrate. |
| TanStack Router | 1.168.23 | Frontend routing | **Keep** | MEDIUM | Already present and sufficient for app-style navigation. |
| TanStack Query | 5.99.2 | Server-state cache and invalidation | **Add next** | HIGH | Multi-client media UX needs predictable caching and invalidation for libraries, details, continue-watching, playback sessions, and setup state. This is the standard missing layer in the current web stack. |
| Shaka Player | 5.1.1 | Browser playback for direct play + adaptive fallback | **Keep** | HIGH | Shaka supports both DASH and HLS and is a strong default for a browser client that may mix direct links with adaptive manifests. |

---

## Prescriptive Stack Shape for Mibo

### Use this as the default target architecture

#### Backend
- **Go 1.24.x** media service
- **PostgreSQL 17/18** as the default production database
- **GORM** for the existing application data path
- **pgx + River** for durable jobs and worker orchestration
- **OpenTelemetry + Prometheus** for traces and metrics
- **ffprobe + ffmpeg** installed in the runtime image
- **OpenList adapter** as the first `StorageProvider`

#### Frontend
- **React + Vite** stay in place
- Add **TanStack Query** for server-state handling
- Keep **Shaka Player** as the browser playback engine

#### Deployment
- **Single image / single node is fine initially**
- Run **API + worker in one deployment** at first
- Split worker deployment only when scan/probe/transcode contention appears
- Keep **Redis out of the first stabilization phase**

---

## What To Add or Reinforce Next

### 1. Add River-backed jobs immediately
**Priority:** Highest  
**Why now:** The architecture already depends on moving scan, metadata matching, ffprobe, and transcode off the request path. River gives durable retries, scheduling, and transactional enqueue without introducing a second stateful system before it is needed.

### 2. Make PostgreSQL the default production path
**Priority:** High  
**Why now:** Once jobs matter, SQLite becomes a compromise. PostgreSQL gives Mibo one durable source of truth for metadata, jobs, progress, idempotency, and future search growth.

### 3. Instrument the service before scaling complexity
**Priority:** High  
**Why now:** Without traces/metrics, you will not know whether OpenList, DB access, ffprobe, or ffmpeg is the real bottleneck.

### 4. Add TanStack Query to the web app
**Priority:** Medium  
**Why now:** The frontend is currently at risk of ad-hoc fetch/state logic. Query caching and invalidation is the standard way to keep playback progress, continue-watching, and library state coherent.

### 5. Keep the storage boundary strict
**Priority:** High  
**Why now:** The biggest long-term stack mistake would be letting OpenList-specific assumptions leak into media-domain services. Keep `StorageProvider` narrow and capability-driven.

---

## What NOT To Use

| Category | Do Not Choose | Why Not |
|----------|---------------|---------|
| Primary DB | SQLite as the default production database | Fine for demo/dev, wrong as the default once durable jobs and concurrent workers are required. |
| Job system | Redis-first queues (BullMQ-style thinking, ad-hoc Redis workers, or “just use Redis lists”) | Adds another stateful dependency too early and loses River’s strongest advantage: transactional job insertion with PostgreSQL-backed application state. |
| Architecture | Early microservice split | Household media servers do not benefit from early service sprawl. It increases deployment and debugging cost before proven need. |
| Storage strategy | Deep OpenList fork | Violates the intended boundary, raises maintenance cost, and makes future adapter replacement harder. |
| Backend rewrite | Rewriting Go backend into Node/NestJS/Java | No payoff relative to current constraints. The problem is service boundaries and job orchestration, not backend language capability. |
| Playback strategy | Transcode-first pipeline | Wastes CPU, increases latency, and is unnecessary for the normal household media path. Direct play should remain the happy path. |
| Frontend architecture | GraphQL-first rewrite | Adds schema and cache complexity without solving the core problem. Stable REST/JSON endpoints plus TanStack Query are enough here. |

---

## Suggested Package Set

### Go modules to add/reinforce

```bash
# durable jobs
go get github.com/riverqueue/river@v0.35.0
go get github.com/jackc/pgx/v5@v5.9.2

# observability
go get go.opentelemetry.io/otel@v1.43.0
go get github.com/prometheus/client_golang@v1.23.2

# optional later
go get github.com/redis/go-redis/v9@v9.18.0
```

### Web packages to add/reinforce

```bash
pnpm add @tanstack/react-query@5.99.2

# already-aligned core packages in repo
pnpm add @tanstack/react-router@1.168.23 shaka-player@5.1.1
```

---

## Confidence by Recommendation

| Recommendation | Confidence | Notes |
|----------------|------------|-------|
| PostgreSQL as default production DB | HIGH | Backed by official PostgreSQL docs and architectural fit for jobs + concurrency. |
| River for worker orchestration | HIGH | Backed by River docs showing PostgreSQL-backed durable and transactional jobs; strong match to Mibo’s API/worker design. |
| Redis as optional later, not first | HIGH | Backed by Redis docs and the repo’s own architecture goals; useful but premature as the first queue. |
| OpenTelemetry + Prometheus now | HIGH | Backed by official OTel docs and standard self-hosted ops practice. |
| TanStack Query addition | HIGH | Backed by official TanStack docs; directly solves server-state caching/invalidation for multi-client UX. |
| Keep Go/net/http/GORM stack rather than rewrite | MEDIUM | Strong fit to current codebase and service shape; this is architecture judgment plus repo context, not one official vendor doc. |
| Keep OpenList as adapter boundary | MEDIUM | Strongly supported by project docs and system constraints; less about public ecosystem doctrine than about correct local architecture. |

---

## Sources

### Project context
- `/Users/atlan/Desktop/IdeaProjects/Mibo/.planning/PROJECT.md`
- `/Users/atlan/Desktop/IdeaProjects/Mibo/docs/media-architecture/improved-architecture.md`

### Version verification / official docs
- PostgreSQL current docs (18.3): https://www.postgresql.org/docs/current/index.html
- River docs via Context7 (`/riverqueue/river`) — PostgreSQL-backed durable jobs and transactional inserts
- OpenTelemetry Go docs via Context7 (`/open-telemetry/opentelemetry-go`)
- Redis official docs: https://redis.io/docs/latest/operate/oss_and_stack/install/archive/install-redis/
- go-redis docs via Context7 (`/redis/go-redis`)
- TanStack Query docs via Context7 (`/tanstack/query`)
- Shaka Player docs via Context7 (`/shaka-project/shaka-player`)

### Package version checks performed during research
- `go list -m -json ...@latest` for `gorm.io/gorm`, `gorm.io/driver/postgres`, `github.com/jackc/pgx/v5`, `github.com/riverqueue/river`, `go.opentelemetry.io/otel`, `github.com/prometheus/client_golang`, `github.com/redis/go-redis/v9`, `github.com/go-chi/chi/v5`
- `npm view ... version` for `@tanstack/react-query`, `@tanstack/react-router`, `shaka-player`, `hls.js`, `vite`, `react`

## Bottom Line

**Recommended standard stack for Mibo’s next phase:**

**Go media service + PostgreSQL + River jobs + OpenList adapter + ffprobe/ffmpeg workers + OpenTelemetry/Prometheus + React/Vite clients with TanStack Query + Shaka Player.**

That is the most standard, maintainable 2025-era stack for a self-hosted household media server that wants a stable business-service core and a replaceable storage boundary.
