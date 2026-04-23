# Project Pitfalls Research: v2 Product Discovery And Operations

**Project:** Mibo  
**Scope:** search, richer filters, trailer playback, metadata governance, storage-change scan listeners, scheduled task management  
**Researched:** 2026-04-23  
**Overall confidence:** HIGH for repo-specific integration risks, MEDIUM for long-tail operational scaling details

This milestone is not adding isolated features. It is adding a **second system of truth** on top of an existing self-hosted media stack: discovery indexes, editable metadata, external trailer links, event-triggered refresh, and operator-managed schedules. In Mibo, that work must stay inside `mibo-media-server`, with `OpenList` remaining only the storage/access layer.

The main failure mode is not “feature incomplete.” It is **breaking the ownership model that v1 already established**:

- `OpenList` owns file access and path namespace
- `mibo-media-server` owns media semantics, searchability, user state, and jobs
- worker jobs already exist, but are still simple queue jobs backed by the app DB
- metadata already mutates canonical `media_items` rows, and scans can reset metadata when base fields change

That means v2 risks are mostly about **index drift, overwrite rules, event storms, and scheduler ambiguity**.

---

## Suggested Phase Labels For Roadmap Mapping

Use these labels when assigning risks into the milestone roadmap:

1. **Phase 1 — Discovery data model**: searchable/filterable fields, canonical metadata ownership, edit/lock semantics
2. **Phase 2 — Search and filtering engine**: query model, indexing, ranking, result shaping, history
3. **Phase 3 — Metadata and trailers UX**: trailer sourcing, metadata editing, rematch/re-fetch flows, admin controls
4. **Phase 4 — Storage listeners and refresh correctness**: event ingestion, coalescing, targeted refresh, reconciliation
5. **Phase 5 — Scheduled jobs and operations**: schedule definitions, execution policy, observability, backpressure, admin tooling

---

## Critical Risks

### 1) Search index built as a side view, not as a projection of canonical media state

**Risk**  
Search starts as ad hoc SQL over `media_items.title`, `cast_json`, and `directors_json`, then later grows a separate index table/FTS table that is updated inconsistently. Users see titles or people in detail pages that search cannot find, or search results that no longer match current metadata.

**Why It Happens**
- v1 data already exists in denormalized JSON/text columns
- full-text search feels easy to bolt on after browse APIs
- metadata edits, rematches, and scan refreshes all change the same records from different paths
- SQLite/Postgres dual support encourages lowest-common-denominator queries unless indexing is designed explicitly

**Warning Signs**
- after metadata apply/rematch, detail page updates immediately but search misses it
- deleted or reclassified items still appear in search
- highlight snippets come from stale text, not current item fields
- search behavior differs between SQLite and Postgres environments

**Prevention**
- define a single **search document projection** owned by `mibo-media-server`
- rebuild/update that projection from canonical media item changes, not from UI actions
- if using SQLite FTS5, keep it synchronized with explicit triggers or a deterministic rebuild/update path; do not rely on best-effort app writes only
- version the indexed payload so future ranking/filter changes can trigger safe reindex jobs
- treat reindex as an operational job type, not a hidden migration side effect

**Best Phase To Address**  
**Phase 1**, then implementation hardening in **Phase 2**

---

### 2) Mixing browse filters with search filters without defining one query contract

**Risk**  
Search, library browse, homepage discovery, and admin metadata review each invent separate filter semantics for year, watched state, library, type, genre, rating, and resolution. The same filter chip returns different counts or different result sets depending on page.

**Why It Happens**
- current browse already has its own `BrowseMediaItemsInput`
- v2 adds new dimensions not yet modeled consistently in the DB
- watched state is user-specific while most metadata is global
- show-level vs episode-level semantics are easy to blur

**Warning Signs**
- “movie/show” filters behave differently between search and browse
- resolution filter works on files but not grouped shows
- watched/unwatched counts shift depending on whether an item has multiple files/episodes
- frontend adds page-specific filter translation logic

**Prevention**
- define one backend query contract for discovery: scope, media type, global filters, user-state filters, sort, and grouping rules
- decide explicitly which filters are **item-level**, **file-level**, and **user-level**
- compute grouped-show semantics once on the server; do not let each client invent it
- document whether filters apply before or after show grouping
- add golden tests for the same query across browse and search endpoints

**Best Phase To Address**  
**Phase 1**

---

### 3) Treating editable metadata as the new truth without lock/merge semantics

**Risk**  
Admin-edited titles, overviews, posters, or season data are overwritten by later TMDB rematch/re-fetch jobs or by scan-driven metadata resets. The product appears to support manual governance, but the system silently reverts user decisions.

**Why It Happens**
- current metadata apply writes directly into canonical `media_items`
- current scan logic resets metadata when base classification changes
- “re-match” and “refresh metadata” are different intents but can touch the same fields
- field-level locks are not present in the current model

**Warning Signs**
- admins report “my edited title changed back after rescan”
- rematch fixes poster but unexpectedly rewrites overview/cast/title
- locking is implemented only in UI state, not persisted in backend rules
- support asks users not to run scans after editing

**Prevention**
- split metadata into at least three concepts: **detected base facts**, **provider facts**, **admin overrides/locks**
- make field locks enforceable in backend write paths, including scan, auto-match, re-fetch, and manual apply
- store provenance per field or per metadata block: source, updated_at, lock state
- require every metadata mutation path to use one merge policy
- add regression tests covering: scan after edit, rematch after edit, and scheduled refresh after edit

**Best Phase To Address**  
**Phase 1**, with UX work in **Phase 3**

---

### 4) Search/filter model tied too tightly to current denormalized JSON columns

**Risk**  
Genres, cast, directors, region, rating, and trailer-related attributes stay embedded in text blobs (`GenresJSON`, `CastJSON`, `DirectorsJSON`), so filtering and ranking become expensive, brittle, and hard to migrate. The team keeps adding string contains hacks instead of a stable discovery model.

**Why It Happens**
- denormalized JSON was fine for v1 detail rendering
- product-native discovery requires indexed facets, not just display payloads
- avoiding middleware pushes complexity into the app DB schema

**Warning Signs**
- filters rely on `LIKE` over JSON text
- cast/director search matches substrings unpredictably
- facet counts are too slow or disabled
- every new filter requires endpoint-specific parsing code

**Prevention**
- promote discovery-critical fields into index-friendly structures
- keep display JSON if helpful, but do not make it the filtering substrate
- for SQLite, design FTS/facet support intentionally and validate concurrency settings; for large filters, use supporting tables/materialized projections rather than repeated JSON scans
- distinguish canonical metadata storage from query-optimized projections

**Best Phase To Address**  
**Phase 1**

---

### 5) Trailer playback modeled as a permanent media asset instead of a volatile external reference

**Risk**  
Trailers are stored and surfaced like stable local media, but actual trailer sources are external, language/region-specific, can disappear, and may fail embed/playback rules. Users get broken play buttons, stale trailers, or trailers unrelated to the matched title.

**Why It Happens**
- UI wants “play trailer” to feel identical to “play media”
- external metadata providers return links and video descriptors, but not all are durable or embeddable
- rematch and language changes can invalidate previous trailer selection

**Warning Signs**
- trailer button appears often but fails at playback time
- trailers survive metadata rematch even when external ID changes
- app stores raw embed URLs with no source/type/locale metadata
- user cannot tell whether “no trailer” means unavailable, fetch failed, or blocked source

**Prevention**
- treat trailer data as **refreshable metadata**, not as canonical media inventory
- store source/provider, external video key, locale, type, last-validated-at, and availability state
- separate “candidate trailers fetched” from “preferred trailer shown in UI”
- make playback degrade gracefully: unavailable, unsupported source, blocked embed, stale candidate
- attach trailer invalidation to metadata rematch and scheduled refresh flows

**Best Phase To Address**  
**Phase 3**

---

### 6) Storage-change listeners treated as truth instead of as hints into the existing reconciliation model

**Risk**  
New storage listeners directly create/update/delete media rows based on events from the storage layer. Duplicate, delayed, partial, or rename events then corrupt library state, especially when `OpenList` is only an HTTP-accessed storage provider and not the business source of truth.

**Why It Happens**
- event-driven refresh sounds more efficient than scanning
- v1 already has targeted refresh foundations, so it is tempting to let events mutate rows directly
- upstream storage/event streams are often lossy, duplicated, or path-centric

**Warning Signs**
- listener code writes `media_items` or `media_files` directly
- rename/delete/create races produce flicker or duplicate rows
- event handler needs many “fallback to full scan” branches
- missing file reports become hard to reproduce because state depended on event order

**Prevention**
- keep listeners as producers of **targeted refresh jobs**, not direct mutators of canonical media rows
- define event semantics as advisory: they narrow scan scope, they do not assert final truth
- coalesce events by library/root path before enqueuing work
- preserve periodic reconciliation even after listeners ship
- explicitly test rename, partial upload, duplicate delivery, delete-after-create, and out-of-order events

**Best Phase To Address**  
**Phase 4**

---

### 7) Event storms turning targeted refresh into a denial-of-service against OpenList and the app DB

**Risk**  
Directory-level listeners emit bursts, each burst becomes a targeted refresh job, and the worker repeatedly scans the same subtree. The result is self-inflicted load on OpenList, the database, and metadata jobs.

**Why It Happens**
- file operations often emit many events for one logical change
- current job uniqueness is keyed per root/reason for queued/running jobs only
- without cooldown windows, the same subtree is rescanned repeatedly

**Warning Signs**
- many queued `targeted_refresh` jobs for near-identical paths
- OpenList latency spikes during bulk copy/rename operations
- background jobs stay busy but library freshness barely improves
- metadata/probe queues grow because refresh keeps rediscovering the same items

**Prevention**
- add debounce/coalescing windows per library subtree
- merge child-path events upward to the minimal safe root when bursts occur
- track suppression metrics: dropped duplicates, merged paths, deferred jobs
- cap concurrent refresh work separately from probe/metadata jobs
- allow listener pause/catch-up modes during bulk library operations

**Best Phase To Address**  
**Phase 4**, with operational tuning in **Phase 5**

---

### 8) Scheduled job management implemented as “enqueue on cron” without schedule ownership, misfire, or overlap rules

**Risk**  
The app can create recurring scans/refreshes/cleanup tasks, but there is no clear distinction between a schedule definition and a job execution. Disabled schedules still run, restarts skip intended runs, and overlapping runs stack up.

**Why It Happens**
- current job system models one-off jobs well, not recurring policy
- it is tempting to bolt timers directly onto the worker loop
- scan refresh already uses config/settings-driven ticker behavior, which is too limited for v2 admin scheduling

**Warning Signs**
- schedule state is inferred from old jobs instead of stored explicitly
- worker restart changes whether a run happens
- long-running scan and next scheduled scan overlap unpredictably
- admins cannot tell “scheduled”, “due”, “running”, “skipped”, and “last successful run” apart

**Prevention**
- model schedules as first-class records with: enabled flag, cadence, next run, last run, last success, policy, target scope, and owner feature
- enqueue executions from schedule evaluation, not from UI clicks or worker boot side effects
- define overlap policy per schedule type: skip, replace, queue one, or parallelize
- define misfire policy after downtime: catch up once, catch up all, or resume from now
- keep execution history separate from schedule definitions

**Best Phase To Address**  
**Phase 5**

---

### 9) One shared worker lane for user-facing freshness jobs and slow operational maintenance

**Risk**  
Scheduled cover refreshes, metadata retries, dead-link checks, and cleanup jobs share the same execution lane as user-visible scans and targeted refresh. Operations work starves discovery freshness, or discovery floods prevent maintenance from ever completing.

**Why It Happens**
- current worker claims jobs from one shared queue ordered by availability
- job kinds have no built-in priority or resource class
- “simple deployment” biases toward one poller and one queue table

**Warning Signs**
- search/browse remains stale while cleanup jobs run for long periods
- admin runs “refresh all metadata” and normal library updates lag badly
- retry storms from one job kind delay all others
- operators cannot prioritize or pause a noisy job class

**Prevention**
- introduce job classes or priorities before adding many recurring task types
- separate concurrency budgets for refresh, metadata, probe, and maintenance jobs
- expose queue age and backlog per kind in admin UI
- support pause/disable by job class, not only by individual job
- make expensive recurring tasks shardable per library instead of monolithic global jobs

**Best Phase To Address**  
**Phase 5**

---

## Moderate Risks

### 10) Search relevance tuned only for titles, making actor/director search feel broken

**Risk**  
The feature technically supports title/actor/director search, but ranking heavily favors title substring matches and weakly handles people names, alternate/original titles, and show-vs-episode collapsing.

**Why It Happens**
- title-only ranking is easier to ship first
- actor/director data often arrives as arrays/JSON, not first-class searchable entities
- grouped-show results complicate relevance scoring

**Warning Signs**
- searching a well-known actor returns many weak title matches first
- original title search works in details but not in search ranking
- episode-level matches clutter show-level search results

**Prevention**
- define ranking weights intentionally: exact title, prefix title, original title, cast, director, popularity/recency, watch state
- collapse episodes to show-level search documents where appropriate
- keep highlight logic aligned with indexed fields
- test known ambiguous searches before freezing API shape

**Best Phase To Address**  
**Phase 2**

---

### 11) User-state filters accidentally cached or indexed as global state

**Risk**  
Filters like watched/unwatched or continue-watching get mixed into shared discovery caches or search documents, causing one user’s watch state to affect another user’s results.

**Why It Happens**
- product search/filter feels like one unified query surface
- watched status is already derived from progress data, which is separate from media metadata
- denormalized result caching often ignores user dimension at first

**Warning Signs**
- admin and normal user see different counts for “all items” unexpectedly
- watched filters behave differently after login changes
- response caching keys only on query text, not user identity

**Prevention**
- keep user-state joins outside the global search document unless explicitly keyed per user
- separate global facets from user-personalized overlays
- review every cached search/list endpoint for user scoping

**Best Phase To Address**  
**Phase 2**

---

### 12) Metadata governance UI ships before governance auditability exists

**Risk**  
Admins can edit, rematch, lock, and refresh metadata, but the system cannot explain who changed what, when a field became locked, or why a later scheduled job skipped or rewrote it.

**Why It Happens**
- UI editing often ships before durable audit design
- “single-admin household app” feels low-risk until later debugging is needed
- scheduled jobs amplify the cost of invisible changes

**Warning Signs**
- support requires DB inspection to answer “why did this metadata change?”
- a locked field is skipped with no visible reason
- there is no last-updated/source view for metadata blocks

**Prevention**
- add lightweight metadata audit fields/events early
- expose source/provenance and lock reason in admin detail views
- record whether changes came from scan, provider fetch, rematch, manual edit, or schedule

**Best Phase To Address**  
**Phase 3**

---

### 13) Scheduled refreshes and listeners re-triggering metadata work with no item-level idempotency

**Risk**  
The same media item is queued repeatedly for match, trailer sync, or metadata refresh because multiple sources of work exist: scan, targeted refresh, manual rematch, scheduled refresh, and external provider corrections.

**Why It Happens**
- current queue uniqueness is strong for some scan jobs, but item-level enrichment can still duplicate logically
- new features create many more triggers for the same enrichment actions

**Warning Signs**
- same item appears in multiple queued metadata jobs
- provider rate limits are hit during bulk refreshes
- retries and manual actions produce indistinguishable duplicate work

**Prevention**
- define job keys at the item/feature scope where safe
- persist freshness timestamps and reason codes per enrichment type
- skip enqueuing when current state is already fresh enough or locked
- expose deduped vs executed counts in job telemetry

**Best Phase To Address**  
**Phase 3** for data rules, **Phase 5** for operational policy

---

## Phase-Specific Warnings

| Roadmap Phase | Likely Pitfall | Why It Matters In Mibo | Mitigation |
|---|---|---|---|
| Phase 1 — Discovery data model | Search/filter fields piggyback on JSON blobs and mutable display fields | Current `media_items` schema is optimized for browse/detail, not discovery governance | Introduce canonical-vs-projected discovery model and metadata lock semantics before shipping search UI |
| Phase 1 — Discovery data model | Manual edits overwritten by scan/rematch | Current scan logic can reset metadata when base fields change | Add provider facts, admin overrides, and field locks with one merge policy |
| Phase 2 — Search/filter engine | Browse and search drift into separate semantics | Existing browse contract is narrower than planned v2 discovery | Define one query contract and one grouping model across list/search surfaces |
| Phase 2 — Search/filter engine | SQLite/Postgres behavior diverges | Repo supports both DBs; search feature will amplify query differences | Use deterministic search projection and DB-specific tests, especially for FTS/facets |
| Phase 3 — Metadata + trailers | Trailer records become stale after rematch or locale change | Trailer source is external and coupled to metadata IDs | Treat trailers as refreshable metadata with validation state |
| Phase 3 — Metadata + trailers | Governance exists in UI but not backend | Manual edits, re-fetch, rematch, and schedules all mutate metadata | Enforce lock/merge rules server-side and expose provenance |
| Phase 4 — Storage listeners | Events mutate canonical rows directly | `OpenList` is an adapter, not the semantic source of truth | Convert events into targeted refresh jobs only |
| Phase 4 — Storage listeners | Event bursts overwhelm targeted refresh | Current worker/job system is simple and path-based | Add path coalescing, debounce windows, and library-level throttles |
| Phase 5 — Scheduled jobs | Schedules are inferred from jobs | Existing job table models executions, not recurring policy | Create first-class schedules with overlap and misfire policies |
| Phase 5 — Scheduled jobs | Slow maintenance starves freshness | Current worker has one shared claim loop | Add job classes, priorities, and per-kind concurrency budgets |

---

## Practical Roadmap Guidance

1. **Do not start Phase 2 UI-first.** Search/filter UX should follow a Phase 1 contract for canonical metadata ownership and projected discovery fields.
2. **Treat metadata governance as a data-model problem first, an admin-screen problem second.** Otherwise manual edits will not survive scans and schedules.
3. **Listeners should only narrow work scope.** They should never become an alternate ingestion pipeline.
4. **Schedules must be modeled, not implied.** Recurring operations need durable schedule state, overlap policy, and visibility.
5. **Plan one explicit reindex/reprojection path.** v2 will almost certainly need it after schema/ranking changes.

---

## Sources

- `.planning/PROJECT.md` — product goals, architecture boundaries, current v1 capabilities, and v2 constraints. **Confidence: HIGH**
- `mibo-media-server/internal/database/models.go` — current canonical tables and denormalized metadata fields. **Confidence: HIGH**
- `mibo-media-server/internal/library/scan.go` — scan/reset behavior, targeted refresh flow, and enqueue patterns. **Confidence: HIGH**
- `mibo-media-server/internal/library/service.go` — queue uniqueness and targeted refresh job shape. **Confidence: HIGH**
- `mibo-media-server/internal/library/query.go` — current browse contract and grouping logic. **Confidence: HIGH**
- `mibo-media-server/internal/jobs/service.go` — current execution queue semantics and uniqueness limits. **Confidence: HIGH**
- `mibo-media-server/internal/worker/worker.go` — single worker claim loop and current scheduled-scan ticker behavior. **Confidence: HIGH**
- `mibo-media-server/internal/metadata/service.go` — current metadata mutation path and lack of field-level lock semantics. **Confidence: HIGH**
- SQLite documentation on FTS5 external-content tables and synchronization triggers. https://www.sqlite.org/fts5.html **Confidence: HIGH**
- SQLite documentation on WAL/concurrency and optimization pragmas. https://www.sqlite.org/pragma.html#pragma_journal_mode **Confidence: HIGH**
