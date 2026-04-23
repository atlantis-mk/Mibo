# Architecture Research: Mibo v2 Product Discovery And Operations

**Scope:** milestone v2 integration architecture only  
**Project:** Mibo  
**Researched:** 2026-04-23  
**Overall confidence:** HIGH

## Integration Points

### Non-negotiable boundary

Keep the existing boundary exactly as:

`OpenList / local storage -> storage.Provider -> mibo-media-server -> JSON API -> web-new/mobile/TV`

Do **not** move search, metadata editing, trailer logic, scheduling, or listener policy into OpenList. The current code already puts storage behind `internal/storage/provider.go`, business orchestration behind `mibo-media-server`, and clients behind `web-new/src/lib/mibo-api.ts`; v2 should extend that shape, not bypass it.

### Where each new feature plugs in

| Feature | Primary integration point | Why |
|---|---|---|
| Product-native search | New app-owned search read model in `mibo-media-server`, fed from catalog changes and metadata changes | Current `internal/search/service.go` is only a stub; search must run on Mibo-owned semantic data, not live storage |
| Richer filtering | Extend library/query read path, not storage listing | Current browse flow already serves DB-backed `MediaItem` reads via `internal/library/query.go` |
| Trailer playback | Extend metadata domain with trailer fetch/cache, expose in media detail API, play as external/embed source from client | TMDB exposes movie and TV videos APIs; Mibo should cache chosen trailer metadata server-side |
| Metadata management | Extend metadata service with overrides, locks, manual re-match, and provider re-fetch jobs | Current metadata flow already supports search/apply/rematch; v2 needs edit governance, not a parallel metadata stack |
| Storage-change scan listeners | Extend existing `/api/v1/storage-events` + targeted refresh flow with listener policy, dedupe, and safety guards | Current router already validates event paths and queues `targeted_refresh` |
| Scheduled job management | Add first-class schedule definitions above existing DB-backed jobs table and worker runner | Current jobs are execution-only; there is no durable recurring schedule model yet |

### Cross-feature integration rule

All new features should converge on the same catalog lifecycle:

1. **scan / targeted refresh** updates `media_items` and `media_files`
2. **metadata enrichment / overrides** updates semantic fields
3. **search index projection** rebuilds searchable documents and facets
4. **query APIs** read only from Mibo DB models/read models
5. **clients** never query OpenList directly

## New Components

### Backend: new services/modules

| Component | Type | Responsibility | Notes |
|---|---|---|---|
| `internal/search/service.go` real implementation | **New** | Search across title/original title/series title/cast/directors; ranking; highlighting; recent searches | Replace current stub returning `planned` |
| `internal/search/indexer.go` or equivalent projection module | **New** | Build/update search documents whenever media metadata changes | Should run from worker jobs, not request path |
| `internal/trailers/service.go` (or `metadata/trailers.go`) | **New** | Fetch, normalize, rank, and cache trailer candidates from TMDB/external video sources | Keep provider fetch logic close to metadata, but trailer cache separate from raw media item fields |
| `internal/metadata/admin.go` | **New** | Apply manual edits, manage locks, reset overrides, re-match/re-fetch selected fields | This is product governance logic, distinct from provider lookup |
| `internal/scheduler/service.go` | **New** | CRUD recurring schedules, compute next-run times, enqueue execution jobs | Do not overload `jobs.Service` with schedule definitions |
| `internal/listeners/service.go` | **New** | Accept storage change events, debounce/coalesce, decide targeted refresh vs full scan fallback | Wrap current router-level event normalization into a domain service |

### Backend: new persistence/read-model tables

| Table | Purpose | Key fields / indexes |
|---|---|---|
| `search_documents` | App-owned denormalized search record per browsable media entity | `media_item_id unique`, `library_id`, `media_type`, searchable text blob/vector, `year`, `region`, `community_rating`, `watched_state`, `resolution_bucket`; indexes on `library_id`, `media_type`, `year`, `watched_state`, `resolution_bucket` |
| `media_item_overrides` | Manual admin overrides separate from provider-owned values | `media_item_id unique`; editable title/original_title/year/overview/poster/backdrop/genres/cast/directors/season_episode fields |
| `media_item_locks` or lock fields on overrides | Lock specific fields from provider refresh | Either boolean columns per field or compact JSON lock map; index by `media_item_id` |
| `media_item_trailers` | Cached trailer candidates and selected default trailer | `media_item_id`, `provider`, `external_video_id`, `site`, `type`, `language`, `url/embed_url`, `official`, `published_at`; index by `media_item_id` |
| `scheduled_jobs` | Recurring job definitions | `kind`, `cron/interval`, `enabled`, `payload_json`, `next_run_at`, `last_run_at`; indexes on `enabled`, `next_run_at` |
| `search_history` | Per-user recent searches | `user_id`, `query`, `scope`, `created_at`; index on `(user_id, created_at)` |
| `library_event_inbox` or reuse `jobs` with listener job keys | Event dedupe/coalescing buffer for storage change notifications | If a dedicated table is used: `library_id`, `root_path`, `event_kind`, `dedupe_key`, `received_at`, `status`; unique on `dedupe_key` |

### Backend: new job kinds

| Job kind | Purpose |
|---|---|
| `reindex_search_document` / `reindex_library_search` | Refresh search projection after scan/metadata edits |
| `sync_media_trailers` | Fetch/update trailer candidates for an item or library scope |
| `refresh_media_metadata` | Re-fetch provider metadata without full rescan |
| `apply_storage_event_refresh` | Coalesced listener-driven targeted refresh job |
| `run_scheduled_job` | Indirection from scheduler definition to concrete execution |
| `cleanup_library` / `validate_links` / `refresh_artwork` | Scheduled operations requested by milestone |

### Frontend: new surfaces in `web-new/`

| Surface | Type | Responsibility |
|---|---|---|
| Search page / global search overlay | **New** | Query input, grouped results, highlights, history |
| Filter bar + filter drawer | **New** | Library/home/search filters for genre/year/region/rating/watched/library/resolution |
| Trailer player panel on media detail | **New** | Show trailer availability and play trailer without entering main playback flow |
| Metadata management panel in settings or media detail admin section | **New** | Edit metadata, lock fields, trigger re-match/re-fetch |
| Scheduled jobs management page | **New** | Create/edit/enable/disable schedules, see last/next run |
| Job execution monitor improvements | **New** | Separate schedule definitions from execution history/failures |

## Modified Components

### Backend: existing modules to extend

| Component | Status | Required change |
|---|---|---|
| `internal/httpapi/router.go` | **Modified** | Add search, filter, trailer, metadata-admin, schedule-management, and listener-management endpoints |
| `internal/app/app.go` | **Modified** | Wire new search/trailer/scheduler/listener services into HTTP + worker graph |
| `internal/database/models.go` | **Modified** | Add new tables/fields/indexes for search docs, overrides, locks, trailers, schedules, optional event inbox |
| `internal/database/database.go` | **Modified** | AutoMigrate new tables, plus manual DB-specific migration hooks for full-text/index setup |
| `internal/worker/worker.go` | **Modified** | Dispatch new job kinds; add schedule tick loop separate from current scan ticker |
| `internal/jobs/service.go` | **Modified** | Support richer filtering by `kind/status/source`, maybe execution metadata, but keep it execution-focused |
| `internal/library/query.go` | **Modified** | Expand browse input to support genre/year/region/rating/watched/library/resolution filters and search-backed result ordering |
| `internal/library/scan.go` | **Modified** | After upsert/cleanup, enqueue search reindex and possibly trailer/metadata follow-up when identities materially change |
| `internal/library/service.go` | **Modified** | Queue event-driven refresh through listener service instead of direct router decisions only |
| `internal/metadata/service.go` | **Modified** | Merge provider metadata with overrides/locks; expose re-fetch-by-field semantics; publish trailer sync jobs when external IDs change |
| `internal/progress/service.go` | **Modified** | Expose watch-state aggregates usable by filters/search facets |

### Frontend: existing modules to extend

| Component | Status | Required change |
|---|---|---|
| `web-new/src/lib/mibo-api.ts` | **Modified** | Add typed methods for search, filters, trailers, metadata CRUD/locks, schedule CRUD, job history |
| `web-new/src/lib/mibo-query.ts` | **Modified** | Add query keys/options for search results, filter facets, trailers, schedules, jobs |
| `web-new/src/features/media/index.tsx` | **Modified** | Load trailer and admin metadata state; invalidate detail/search projections after edits |
| `web-new/src/features/media/components/standalone-media-detail.tsx` | **Modified** | Render trailer CTA/player, metadata admin controls, lock state, re-match/re-fetch actions |
| `web-new/src/features/library/index.tsx` | **Modified** | Add filter controls and filtered result loading |
| `web-new/src/features/home/index.tsx` | **Modified** | Optionally surface global discovery filters/search entry |
| `web-new/src/features/settings/index.tsx` | **Modified** | Add schedule and operations tabs instead of current placeholder “通知与任务” content |
| `web-new/src/features/settings/components/library-management-panel.tsx` | **Modified** | Add scan-listener settings and links to job/schedule controls |
| `web-new/src/components/search-form.tsx` | **Modified** | Replace placeholder docs-style input with real Mibo search navigation |
| `web-new/src/components/app-sidebar.tsx` | **Modified** | Replace sample navigation with product navigation and search entrypoint |

## Schema And Index Changes

### Recommended approach

Use **portable relational tables as the source of truth** and add **DB-specific full-text acceleration as an optional projection layer**:

- **SQLite:** official docs confirm FTS5 supports full-text search, ranking, and `highlight()`/`snippet()` helpers.  
- **PostgreSQL:** official docs confirm native full-text search with `tsvector`/`tsquery` and ranking.

That means:

1. Keep canonical search content in `search_documents`
2. Add DB-specific search acceleration in migrations:
   - SQLite: FTS5 virtual table over external content or synchronized projection
   - PostgreSQL: generated/maintained `tsvector` + GIN index
3. Keep facet filters in normal indexed columns so filters behave consistently on both DBs

### Minimal field additions needed for v2 filtering

| Field source | Needed for |
|---|---|
| `genres` normalized out of `genres_json` or duplicated into search projection | genre filter |
| `year` | year filter |
| region / origin country | region filter |
| provider/community rating | rating filter |
| derived watched state from `playback_progress` | watched/unwatched filter |
| `library_id` | library filter |
| derived resolution bucket from `media_files.width/height` | resolution filter |

### Important migration note

Current `database.Open()` only uses `AutoMigrate`. That is not enough for FTS virtual tables, `tsvector` indexes, or advanced trigger/projection setup. v2 needs an explicit migration layer for search acceleration and possibly for projection triggers.

## API Surface Changes

### New read APIs

- `GET /api/v1/search?q=&type=&library_id=&genre=&year=&region=&rating_min=&watched=&resolution=&page=`
- `GET /api/v1/search/history`
- `GET /api/v1/media-items/{id}/trailers`
- `GET /api/v1/filter-options` or scoped facet endpoints if needed
- `GET /api/v1/scheduled-jobs`
- `GET /api/v1/scheduled-jobs/{id}/runs`

### New write/admin APIs

- `POST /api/v1/search/history`
- `PUT /api/v1/media-items/{id}/metadata/overrides`
- `PUT /api/v1/media-items/{id}/metadata/locks`
- `POST /api/v1/media-items/{id}/metadata/refetch`
- `POST /api/v1/media-items/{id}/trailers/sync`
- `POST /api/v1/scheduled-jobs`
- `PATCH /api/v1/scheduled-jobs/{id}`
- `POST /api/v1/scheduled-jobs/{id}/run`
- `POST /api/v1/storage-listeners/test-event` or keep using `/api/v1/storage-events` with richer server-side policy

### Existing APIs to extend

- `GET /api/v1/libraries/{id}/items` → add richer filter params
- `GET /api/v1/media-items/{id}` → include trailer summary, metadata override state, lock state, editable/admin flags
- `GET /api/v1/jobs` → add schedule/run correlation fields

## Data Flow

### 1. Search and filtering flow

1. Scan or metadata change updates `media_items` / `media_files`
2. Worker enqueues `reindex_search_document`
3. Search projection computes searchable text + facets + watch-state snapshot
4. Client calls `/api/v1/search` or filtered library endpoint
5. Search service returns grouped hits, highlight fragments, and filter-aware ranking

**Key rule:** search reads from projection tables, not from `MediaItem` raw JSON columns directly on every request.

### 2. Trailer flow

1. `media_item.external_id` becomes available or changes
2. Worker enqueues `sync_media_trailers`
3. Trailer service calls TMDB videos API, normalizes candidates, caches preferred trailer
4. `GET /api/v1/media-items/{id}` and `/trailers` include trailer metadata
5. Frontend opens embedded/external trailer player

**Key rule:** trailer playback is discovery media, not primary playback; do not route it through `playback.Service` unless Mibo later proxies trailer streams.

### 3. Metadata management flow

1. Admin opens metadata editor from media detail/settings
2. Client reads current provider values + override values + lock state
3. Admin saves overrides or lock changes via metadata-admin API
4. Metadata service persists overrides, applies merged view to detail reads, and enqueues search reindex
5. If admin requests re-match/re-fetch, worker fetches fresh provider data but respects locks

**Key rule:** keep provider data and human overrides separate; never overwrite the only copy of provider-derived facts.

### 4. Storage-listener flow

1. External storage notifier or internal bridge posts event to Mibo
2. Listener service validates library scope, coalesces duplicate events, decides targeted root
3. Listener service enqueues `apply_storage_event_refresh`
4. Worker runs targeted refresh using existing `scanLibraryWithMode(...partial=true...)`
5. Scan output fans out into metadata/probe/search updates

**Key rule:** listeners are just another ingestion trigger; they must reuse the existing targeted refresh pipeline.

### 5. Scheduled job flow

1. Admin creates schedule definition in `scheduled_jobs`
2. Scheduler loop picks due schedules and enqueues concrete execution jobs into existing `jobs` table
3. Worker executes concrete job kind (`sync_library`, `refresh_media_metadata`, `sync_media_trailers`, etc.)
4. Execution status stays in `jobs`; schedule definition keeps next/last-run metadata

**Key rule:** recurring schedule definitions and one-off job executions must stay separate.

## Recommended Build Order

### 1. Catalog extension foundation

Build first:

- new schema for search projection, trailers, metadata overrides/locks, schedules
- explicit migration path beyond `AutoMigrate`
- worker dispatch support for new job kinds
- API/client type additions

**Why first:** every v2 feature except basic UI polish depends on durable new data models.

### 2. Search + filter backend and read APIs

Build second:

- implement `internal/search/service.go`
- add `search_documents` projection and reindex jobs
- extend library browse/filter query inputs
- add frontend search page + filter bar

**Why second:** this is the main user-facing milestone value and depends only on foundation, not on schedules/listeners.

### 3. Trailer sync + detail-page playback surface

Build third:

- trailer service and cache table
- `/media-items/{id}/trailers` API
- media detail trailer UI
- optional manual sync action

**Why third:** it leverages existing metadata external IDs and is relatively isolated.

### 4. Metadata management and lock-aware re-match

Build fourth:

- overrides and lock model
- metadata-admin APIs
- merge logic in detail/search projections
- admin UI on media detail/settings

**Why fourth:** search and trailers should already exist so edits immediately affect discovery surfaces; locks must be in place before large-scale automated re-fetch.

### 5. Scheduled job management

Build fifth:

- `scheduled_jobs` model and scheduler service
- schedule CRUD UI
- support metadata refresh, trailer sync, scan, cleanup, and link validation schedules

**Why fifth:** recurring automation should sit on top of already-working one-off job handlers.

### 6. Storage-change listeners hardening

Build sixth:

- listener service abstraction over `/api/v1/storage-events`
- dedupe/coalescing policy
- settings/admin controls for listener enablement and fallback behavior
- event-to-targeted-refresh metrics/logging

**Why sixth:** current targeted refresh already exists; listener safety is mostly an ingestion hardening phase after search/metadata projections are resilient.

## Dependency-Aware Phase Graph

```text
foundation schema/migrations
  -> search projection + filter APIs
  -> trailer cache + trailer UI
  -> metadata overrides/locks
  -> scheduled job definitions
  -> storage listener hardening

storage listeners
  -> targeted refresh
  -> metadata/probe refresh
  -> search reindex

metadata overrides/locks
  -> lock-aware refetch
  -> reliable scheduled metadata jobs
```

## Architecture Recommendations

- **Use one new read-model family, not one table per screen.** Search, filters, and highlight snippets should come from a shared projection.
- **Keep trailer logic adjacent to metadata, not playback.** Trailer selection depends on metadata providers; main playback depends on media files and storage links.
- **Separate provider facts from human edits.** Overrides + locks are mandatory if admins can edit metadata without fighting the next refresh.
- **Separate schedules from job executions.** `jobs` is already the execution queue/history; add `scheduled_jobs` instead of encoding recurrence inside `jobs`.
- **Treat listeners as a trigger, not a scanner rewrite.** Reuse `targeted_refresh` and existing path validation.

## Sources

- `.planning/PROJECT.md` — HIGH
- `.planning/codebase/ARCHITECTURE.md` — HIGH
- `.planning/codebase/INTEGRATIONS.md` — HIGH
- `.planning/codebase/STACK.md` — HIGH
- `mibo-media-server/internal/storage/provider.go` — HIGH
- `mibo-media-server/internal/app/app.go` — HIGH
- `mibo-media-server/internal/httpapi/router.go` — HIGH
- `mibo-media-server/internal/library/service.go` — HIGH
- `mibo-media-server/internal/library/scan.go` — HIGH
- `mibo-media-server/internal/library/query.go` — HIGH
- `mibo-media-server/internal/library/enrichment.go` — HIGH
- `mibo-media-server/internal/jobs/service.go` — HIGH
- `mibo-media-server/internal/worker/worker.go` — HIGH
- `mibo-media-server/internal/metadata/service.go` — HIGH
- `mibo-media-server/internal/database/models.go` — HIGH
- `web-new/src/lib/mibo-api.ts` — HIGH
- `web-new/src/lib/mibo-query.ts` — HIGH
- `web-new/src/features/media/index.tsx` — HIGH
- `web-new/src/features/media/components/standalone-media-detail.tsx` — HIGH
- `web-new/src/features/library/index.tsx` — HIGH
- `web-new/src/features/settings/index.tsx` — HIGH
- SQLite FTS5 docs: https://sqlite.org/fts5.html — HIGH
- PostgreSQL full-text search docs: https://www.postgresql.org/docs/current/textsearch-intro.html — HIGH
- TMDB movie videos API: https://developer.themoviedb.org/reference/movie-videos — HIGH
- TMDB TV series videos API: https://developer.themoviedb.org/reference/tv-series-videos — HIGH
