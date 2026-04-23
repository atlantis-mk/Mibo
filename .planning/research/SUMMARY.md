# Project Research Summary

**Project:** Mibo  
**Domain:** Self-hosted media server discovery and admin operations  
**Researched:** 2026-04-23  
**Confidence:** HIGH

## Recommended Direction

Mibo v2 should stay within the current boundary: **storage provider/OpenList → `mibo-media-server` → JSON APIs → clients**. The right move is not adding external middleware. Search, filtering, metadata governance, trailer selection, listener policy, and scheduling should all live inside `mibo-media-server`, backed by the existing database and worker model.

For stack choices, the strongest recommendation is **SQL-native discovery infrastructure**: SQLite FTS5 by default, PostgreSQL `tsvector` + GIN where Postgres is used, explicit SQL migrations, an app-owned search projection, DB-backed schedule definitions, `robfig/cron/v3` only for schedule triggering, and `fsnotify` only for local-provider directory watching. Trailers should be treated as refreshable metadata references, not downloaded media.

Roadmap-wise, the milestone should be built as **foundation first, discovery second, operations third**. Metadata ownership, projection tables, and query contracts must land before polished search/filter UI. Scheduled jobs and scan listeners should reuse the existing job pipeline instead of becoming parallel execution systems.

## Table Stakes

### Stack decisions
- **Keep `mibo-media-server` as the product brain** — do not push business logic into OpenList.
- **Add an app-owned search projection** — search and facets should not query JSON blobs directly.
- **Use SQLite FTS5 / Postgres full-text search** — enough for v2 without Meilisearch/Elasticsearch.
- **Add explicit SQL migrations** — required for FTS tables, triggers, generated columns, and schedule indexes.
- **Use `robfig/cron/v3` as a trigger layer only** — schedules enqueue jobs; workers do the actual work.
- **Use `fsnotify` only for local storage listeners** — keep reconciliation scans for safety.

### User-facing table stakes
- **Product-native search** across title, original title, cast, and director.
- **Richer filters** for genre, year, library, watched state, rating, resolution, and sort.
- **Trailer CTA on detail pages** when a trusted trailer exists.
- **Typed, stable results** with title-first ranking and clear movie/series labeling.

### Admin/operational table stakes
- **Manual metadata editing** for title, summary, artwork, genres, and credits.
- **Field-level locks** so refreshes and rematches do not overwrite admin decisions.
- **Re-match and refresh actions** as separate flows.
- **Scheduled task management** with enable/disable, run now, next run, last run, and history.
- **Conservative scan listeners** that enqueue targeted refresh jobs, plus fallback reconciliation scans.

### Explicitly defer
- External search middleware.
- Semantic/vector search.
- Trailer downloads, proxying, or transcoding.
- Workflow-engine style scheduling.
- Large-scale bulk metadata editing.

## Suggested Sequencing

### 1. Discovery Foundation
**Why first:** Search, filters, trailers, and metadata editing all depend on clean ownership and indexable data.  
**Deliver:** projection tables, metadata overrides/locks, trailer records, schedule tables, explicit migrations, shared discovery query contract.  
**Key watch-out:** do not keep building on JSON `LIKE` queries or direct metadata overwrites.

### 2. Search + Filters
**Why second:** This is the main user-visible value once the data model is stable.  
**Deliver:** search API, ranking/highlights, search history, filtered browse/search endpoints, shared filter semantics across search and library views.  
**Key watch-out:** browse and search must use one backend contract, especially for watched state, grouping, and media type semantics.

### 3. Metadata Management + Trailers
**Why third:** Discovery quality depends on metadata governance, and trailers are only useful once identity and provider data are trustworthy.  
**Deliver:** admin metadata editor, lock visibility, rematch/refetch flows, TMDB trailer sync, detail-page trailer playback.  
**Key watch-out:** keep provider facts, overrides, and locks separate; invalidate trailer selections when item identity changes.

### 4. Scheduled Jobs
**Why fourth:** Recurring automation should sit on top of working one-off jobs and stable metadata/search flows.  
**Deliver:** schedule CRUD, run-now controls, next/last run visibility, no-overlap policy, job/schedule correlation.  
**Key watch-out:** schedules are definitions, not executions; misfire and overlap policy must be explicit.

### 5. Scan Listeners + Refresh Hardening
**Why fifth:** Listener correctness matters more than immediacy, and it depends on resilient refresh and reindex pipelines already existing.  
**Deliver:** listener service, event dedupe/coalescing, targeted refresh enqueueing, admin visibility, reconciliation fallback.  
**Key watch-out:** listeners must only narrow work scope; they must never mutate canonical media rows directly.

## Watch Out For

1. **Index drift** — search must be a projection of canonical media state, rebuilt from scan/metadata changes and versionable for reindex jobs.
2. **Filter drift** — search, browse, and home surfaces need one discovery contract or users will see inconsistent counts and results.
3. **Metadata overwrite bugs** — manual edits will fail without persisted backend lock/merge semantics.
4. **JSON-as-database creep** — discovery-critical fields need normalized/indexable structures, not more substring hacks.
5. **Trailer staleness** — trailers are volatile external references and must be refreshed when metadata identity or locale changes.
6. **Event storms** — bursty file events can DOS OpenList and the DB without debounce, path coalescing, and concurrency limits.
7. **Scheduler ambiguity** — if schedule state lives only in jobs, restarts and overlaps will behave unpredictably.
8. **Shared worker starvation** — slow maintenance jobs can block freshness unless job classes or per-kind concurrency budgets are added.

## Open Questions

- **SQLite readiness:** confirm FTS5 is enabled in all target builds and test environments.
- **Cross-DB parity:** define how closely SQLite and Postgres search ranking/highlight behavior must match.
- **Watched-state semantics:** lock down item-level vs episode/file-level rules before finalizing filter APIs.
- **Trailer source policy:** decide whether v2 is YouTube-only or includes Vimeo from day one.
- **Schedule UX:** decide how much cron syntax is exposed versus presets/basic forms.
- **Operational visibility:** confirm whether metadata audit/provenance and queue-per-kind telemetry are required in this milestone or the next.

## Sources

- `.planning/research/STACK.md`
- `.planning/research/FEATURES.md`
- `.planning/research/ARCHITECTURE.md`
- `.planning/research/PITFALLS.md`

---
*Research completed: 2026-04-23*  
*Ready for roadmap: yes*
