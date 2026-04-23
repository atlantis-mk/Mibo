# Stack Research: v2 Product Discovery And Operations

**Project:** Mibo  
**Scope:** New stack additions/changes for product-native search, richer filtering, trailer playback, metadata management, storage-change scan listeners, and scheduled job management  
**Researched:** 2026-04-23  
**Overall recommendation:** Keep the current `OpenList -> mibo-media-server -> clients` shape, add SQL-native indexing and in-process scheduling inside `mibo-media-server`, and do **not** introduce external search/queue/scheduler middleware for this milestone.

## Recommended Additions

| Area | Add / Change | Version / Status | Why this fits Mibo | Integration constraints |
|---|---|---|---|---|
| Search + filtering backend | **Add an internal search projection plus SQL-native full-text indexing** | **SQLite FTS5** on current SQLite 3.x deployments; **PostgreSQL `tsvector` + GIN** on current Postgres deployments | Keeps search inside the existing database, preserves simple deployment, and keeps semantic ownership in `mibo-media-server` instead of OpenList or a sidecar search engine | Implement behind `internal/search` so query behavior is shared while SQL differs by driver; use raw SQL migrations for FTS/GIN objects rather than GORM AutoMigrate alone |
| Search query execution | **Keep Go stdlib + GORM for CRUD, use raw SQL for ranking/highlight search queries** | Current repo stack: Go 1.24, `gorm.io/gorm v1.25.12` | FTS ranking, snippets, generated vectors, and virtual tables are not a good fit for ORM-only abstraction | Add a driver-aware search repository; keep API responses stable for Web/mobile/TV |
| Scheduler | **Add `github.com/robfig/cron/v3`** | Pin latest stable **v3.x** | Best fit for in-process schedule parsing, timezone-aware cron support, and wrappers like `Recover` / `SkipIfStillRunning`; scheduler should enqueue into the existing `jobs` table, not run business logic inline | Use scheduler only to create jobs; keep real work in existing worker/job pipeline |
| Local storage listeners | **Add `github.com/fsnotify/fsnotify`** | Pin latest stable **v1.x** | Best lightweight choice for local provider file change detection without adding another service | Use only for `local` provider; watch directories, not files; re-add watches for new subdirs |
| OpenList storage listeners | **Do not add a new library; extend current storage-event ingestion path** | Reuse current `/api/v1/storage-events` + targeted refresh foundation | The current OpenList adapter only exposes HTTP list/get/link semantics; Mibo should keep event normalization and safety rules in `mibo-media-server` | For OpenList-backed libraries, accept pushed events from an upstream notifier if available, but always keep periodic reconciliation scans because remote/network filesystems are not fully reliable for edge-triggered events |
| Trailer metadata | **Extend existing metadata service to call TMDB video endpoints** | No new HTTP client lib required; reuse stdlib `net/http` | Current metadata service already owns TMDB integration, so trailers belong there too | Persist normalized trailer records in Mibo DB; return safe embed/play URLs to clients instead of exposing arbitrary third-party HTML |
| Trailer playback UI | **Do not add a heavyweight player SDK initially** | Reuse current React stack | TMDB trailers are usually external-hosted assets (commonly YouTube); a lazy iframe/webview flow is enough for v2 | Whitelist supported sites (start with YouTube, optionally Vimeo); no proxy/transcode path for trailers in v2 |
| Metadata admin UI | **Reuse existing frontend form/table stack** (`react-hook-form`, `zod`, `@tanstack/react-table`) | Already present in `web-new/package.json` | No need for a new admin framework just to edit metadata and schedules | Keep forms schema-driven and API-first so the same endpoints work for Web/mobile/TV admin surfaces later |
| Migrations discipline | **Add explicit SQL migration steps inside backend startup/migrations package** | Project change, not a new service | FTS virtual tables, triggers, partial indexes, generated columns, and schedule indexes need deterministic SQL | Keep GORM for base tables, but do not rely on AutoMigrate for search/index objects |

## Internal Data Changes

### 1. Search and filter projection

Add an application-owned projection layer instead of querying `media_items` JSON fields directly.

**Recommended tables:**

1. `media_search_documents`
   - `media_item_id` (PK)
   - `library_id`
   - `media_type`
   - `title`
   - `original_title`
   - `series_title`
   - `overview`
   - `search_people_text` (actor/director names flattened for FTS)
   - `search_genres_text`
   - `search_countries_text`
   - `year`
   - `vote_average`
   - `watched_state` (`unwatched` / `in_progress` / `watched`)
   - `max_height`
   - `resolution_bucket` (`sd` / `hd` / `fhd` / `uhd`)
   - `updated_at`

2. `media_item_genres`
   - `media_item_id`
   - `genre`

3. `media_item_people`
   - `media_item_id`
   - `role` (`cast`, `director`)
   - `name`
   - optional `sort_order`

4. `media_item_countries`
   - `media_item_id`
   - `country_code`
   - `country_name`

**Why this shape:**
- Search/highlight wants a denormalized text document.
- Faceted filtering wants indexable scalar rows, not JSON blobs.
- It avoids making OpenList responsible for search semantics.

### 2. Full-text index implementation

**SQLite path (recommended default for current repo default):**
- Create `media_search_fts` as an **FTS5 external-content table** over `media_search_documents`.
- Use FTS5 ranking (`bm25`) and snippets/highlights from SQLite FTS5.
- Preferred tokenizer: `unicode61`; verify compiled FTS5 support at startup/CI.

**PostgreSQL path (keep compatibility with supported Postgres mode):**
- Add a generated `tsvector` column or expression index over the same search document.
- Index with **GIN**.

**Pattern to follow:**
- `media_items` remains the canonical semantic record.
- Search tables are projections rebuilt on scan, metadata match/apply, manual edit, delete, and restore.

### 3. Metadata management data model

Do **not** keep expanding only JSON blobs on `media_items`. For admin editing and lock/re-match behavior, add explicit metadata control structures.

**Recommended additions:**

1. `media_item_metadata_overrides`
   - `media_item_id`
   - `field_key` (`title`, `original_title`, `overview`, `year`, `poster_url`, `backdrop_url`, `genres`, `cast`, `directors`, etc.)
   - `value_json`
   - `locked` (bool)
   - `updated_by_user_id`
   - `updated_at`

2. `media_item_metadata_state`
   - `media_item_id`
   - `provider` (`tmdb`)
   - `external_id`
   - `last_fetched_at`
   - `last_matched_at`
   - `needs_refresh`
   - `match_confidence`
   - `revision`

3. Extend `media_items` with fields needed for new filtering and trailer selection:
   - `original_language`
   - `production_countries_json` or derive from `media_item_countries`
   - `vote_average`
   - `vote_count`

**Why:**
- Manual edits and field locks are operational state, not just provider payload.
- Re-match/re-fetch must know what is user-owned versus provider-owned.

### 4. Trailer storage

Add `media_item_trailers`:
- `media_item_id`
- `provider` (`tmdb`, later `manual`)
- `site` (`YouTube`, `Vimeo`)
- `video_key`
- `type` (`Trailer`, `Teaser`, `Clip`)
- `official` (bool)
- `language`
- `country`
- `published_at`
- `sort_order`
- `status`

**Recommendation:** store `site + video_key + metadata`, not raw embed HTML. Compute embed/play URLs server-side from a whitelist.

### 5. Scheduled job management data model

Current `jobs` is an execution queue/history table, not a schedule definition table. Keep it, and add:

1. `job_schedules`
   - `id`
   - `kind`
   - `target_scope_json` (library IDs, source IDs, global)
   - `payload_json`
   - `cron_expression`
   - `timezone`
   - `enabled`
   - `next_run_at`
   - `last_run_at`
   - `concurrency_policy` (`skip`, `delay`)
   - `created_by_user_id`
   - `updated_at`

2. Extend `jobs`
   - `schedule_id` (nullable)
   - `trigger_kind` (`manual`, `schedule`, `storage_event`, `system`)

This preserves the current worker architecture: schedules enqueue jobs; workers process jobs.

### 6. Search history

Add `user_search_history`:
- `user_id`
- `query`
- `scope`
- `created_at`

Keep it simple; no Redis cache is needed.

## Operational Considerations

### Search + filtering

- **Do indexing inside `mibo-media-server` transactions/workflows.** When scan/match/manual-edit changes a media item, update canonical tables first, then refresh projection rows.
- **Prefer async rebuild for bulk scans, sync rebuild for single-item edits.** Bulk operations should enqueue reindex work; admin edits should feel immediate.
- **Use driver-specific SQL, shared API contract.** `internal/search` should hide SQLite-vs-Postgres differences from HTTP handlers.
- **Do not query JSON text for filters at runtime.** That will get slow and inconsistent as the library grows.

### Storage listeners

- **Local provider:** `fsnotify` is good enough, but it is not recursive; watch directories and add watches dynamically. Its own docs note subdirectories are not watched automatically, and network filesystems are not reliable for notifications.
- **OpenList-backed provider:** keep current `storage-events -> targeted_refresh/full_scan` path as the integration contract. Because Mibo only talks to OpenList through HTTP adapter calls, listener semantics must stay in Mibo, not in an OpenList fork.
- **Always keep a safety reconciliation scan.** Event streams can miss moves, remote renames, network share churn, and offline windows.

### Scheduled jobs

- Use `robfig/cron/v3` only as the **trigger layer**.
- Attach `Recover` and `SkipIfStillRunning`/`DelayIfStillRunning` wrappers so a broken schedule does not flood the queue.
- Persist schedule definitions in DB and rebuild cron entries from DB at startup.
- Do not execute scan/metadata/trailer logic directly in cron callbacks; enqueue the existing jobs instead.

### Trailer playback

- TMDB officially exposes movie and TV video endpoints, and details endpoints support `append_to_response`; use those through the existing metadata service.
- Start with **YouTube-only** if needed for predictability; Vimeo can be a second supported site if present in real data.
- No proxying, downloading, or transcoding trailer media in v2. That would create unnecessary legal, bandwidth, and storage complexity.

### Migrations and testing

- Add startup/integration tests that verify:
  - SQLite build has FTS5 enabled.
  - Projection rebuild is idempotent.
  - Manual metadata locks survive re-match/re-fetch.
  - Schedule reload after restart recreates expected next runs.
  - Storage listener fallbacks produce targeted refresh when safe and full scan when not.

## Do Not Add Yet

| Do not add | Why not for this milestone |
|---|---|
| Meilisearch / Elasticsearch / OpenSearch / Typesense | Violates the “no external middleware” constraint and adds deployment + sync complexity before Mibo has validated its own search model |
| Bleve in `mibo-media-server` | Still creates a second index/storage subsystem to maintain when SQLite/Postgres already cover v2 requirements |
| OpenList’s built-in Bleve/Meilisearch search stack | Wrong ownership boundary: OpenList indexes storage namespace, while Mibo owns semantic media identity, watched state, metadata edits, and cross-client API behavior |
| Redis / NATS / Kafka for scheduling or event fan-out | Not needed; the existing DB-backed jobs model plus in-process cron is enough for v2 |
| Temporal / Airflow / separate scheduler service | Massive overkill for schedule definitions that ultimately enqueue jobs into one Go service |
| New frontend admin framework | Existing React + RHF + Zod + TanStack table stack is already sufficient |
| Heavy trailer player SDK | Trailer playback is external embed playback, not a new core streaming pipeline |
| Deep OpenList fork to add media-business logic | Conflicts with project constraints and would entangle storage access with Mibo-owned product semantics |

## Concrete Implementation Order

1. **Introduce SQL migrations for search/schedule objects.**
2. **Add projection tables + indexes for search/filtering.**
3. **Implement `internal/search` with SQLite FTS5 first, Postgres adapter second.**
4. **Add trailer tables + TMDB video sync in metadata service.**
5. **Add metadata overrides/locks and merge logic.**
6. **Add `job_schedules` + cron enqueuer.**
7. **Add `fsnotify` local watcher and keep OpenList on event-ingest + reconciliation.**

## Sources

### Repo / project context
- `.planning/PROJECT.md`
- `mibo-media-server/go.mod`
- `mibo-media-server/internal/database/models.go`
- `mibo-media-server/internal/database/database.go`
- `mibo-media-server/internal/httpapi/router.go`
- `mibo-media-server/internal/jobs/service.go`
- `mibo-media-server/internal/library/service.go`
- `mibo-media-server/internal/metadata/service.go`
- `mibo-media-server/internal/search/service.go`
- `mibo-media-server/internal/storage/openlist/adapter.go`
- `mibo-media-server/internal/worker/worker.go`
- `web-new/package.json`
- `OpenList/internal/search/bleve/*`
- `OpenList/internal/search/meilisearch/*`

### Documentation verified during research
- SQLite FTS5 docs via Context7 (`/websites/sqlite_docs`) — external-content FTS tables, triggers, ranking/snippet support
- PostgreSQL current docs via Context7 (`/websites/postgresql_current`) — generated `tsvector` and GIN indexing for full-text search
- `robfig/cron/v3` docs via Context7 (`/robfig/cron`) — parser options, recovery, and concurrency wrappers
- TMDB official API docs via Context7 (`/websites/developer_themoviedb_reference`) and official site — movie/TV video endpoints and `append_to_response`
- fsnotify README / docs — directory watching model, non-recursive behavior, and network filesystem caveats

## Bottom Line

For this milestone, the right stack move is **not** “add search infrastructure.” It is **make `mibo-media-server` smarter with SQL-native indexing, explicit metadata-control tables, a DB-backed schedule layer, and provider-aware listeners**.

That keeps deployment simple, respects the current OpenList boundary, and gives Mibo native product search/operations features without introducing external middleware too early.
