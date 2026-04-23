# Feature Landscape: Milestone v2 Discovery And Operations

**Domain:** Self-hosted / home-media product discovery and admin operations  
**Project:** Mibo  
**Researched:** 2026-04-23  
**Milestone focus:** search, richer filtering, trailer playback, metadata management, scan listeners, scheduled task management  
**Overall confidence:** MEDIUM-HIGH

## Executive Summary

In mature home-media products, these v2 features split cleanly into two layers: **user-facing discovery** and **admin-facing governance**. Search, filters, and trailers are only as good as the metadata beneath them. Scan listeners, metadata tooling, and scheduled jobs are the operational layer that keeps discovery fresh and trustworthy.

That means Mibo should not treat all six target areas as independent features. In practice, **metadata management + refresh orchestration are the foundation**, while **search + filters are the primary user payoff**. Trailer playback is valuable, but it is downstream of correct item matching and background sync.

The market baseline from Jellyfin, Plex, and Emby is consistent: libraries are typed, metadata is enriched automatically, admins can edit/lock fields, systems run configurable background tasks, and library changes can be monitored in real time where supported. The implication for Mibo is clear: this milestone should feel like a product becoming more complete and operationally reliable, not like a lab for advanced AI discovery.

## Core Feature Model For This Milestone

```text
storage events / scheduled jobs
  → targeted refresh pipeline
  → metadata quality + freshness
  → search index + filter facets + trailer availability
  → better detail pages and faster content discovery

admin governance capabilities
  → user-facing discovery quality
```

## Table Stakes Across The Milestone

| Capability | Why users/admins expect it | Complexity | Milestone stance |
|------------|----------------------------|------------|------------------|
| Product-native search across title/person fields | Every mature media product lets users find items directly instead of browsing only by folder/library | Medium | Must ship |
| Faceted filtering on common metadata | Users expect to narrow large libraries by genre/year/library/watched state at minimum | Medium | Must ship |
| Trailer entry on detail pages when available | Common premium-feeling discovery affordance; absence is acceptable, bad behavior is not | Medium | Ship if metadata match is reliable |
| Admin metadata correction and rematch | Mature products assume automatic metadata will be wrong sometimes | High | Must ship |
| Storage-change-triggered refresh | Users expect new/changed files to appear without babysitting full scans | High | Must ship in conservative form |
| Scheduled background task management | Admins expect long-running maintenance to be visible, configurable, and manually runnable | Medium | Must ship |

## Category 1: Search

### How mature products typically behave

- Search is **global inside the product**, not a file-path lookup.
- It matches **primary title first**, then alternate/original title, then people fields like actor/director.
- Results are grouped or clearly labeled by media type, usually movie vs series first.
- Result quality favors **exact and prefix title matches** over looser person matches.
- Empty states are explicit; users are not expected to guess whether indexing failed.

### Table Stakes

| Feature | Why expected | Complexity | Scope guidance for Mibo |
|---------|--------------|------------|--------------------------|
| Search by title | Baseline behavior in all mature apps | Medium | Exact + prefix + normalized matching |
| Search by actor/director | Common expectation once metadata exists | Medium | Supported only for indexed, normalized people metadata |
| Distinguish movies vs series in results | Prevents ambiguous click paths | Low | Show result type badge/section |
| Stable ranking | Users expect obvious items near top | Medium | Title exact > alt title > cast/director match |
| Highlight matched text | Improves trust in why result appeared | Low | Good fit for milestone |
| Search history / recents | Common convenience feature | Low | Keep local/user-scoped and simple |

### Differentiators Worth Doing

| Feature | Value | Complexity | Recommendation |
|---------|-------|------------|----------------|
| Weighted ranking with title-first bias | Makes results feel “smart” without external search infra | Medium | Yes |
| Unified search API across Web/mobile/TV | Strong long-term payoff for multi-client product | Medium | Yes |
| Lightweight typo tolerance / normalization | Helps with punctuation, spacing, romanization | Medium | Yes, if achievable in existing DB stack |
| Search suggestions from recent queries | Small UX win | Low | Nice if cheap |

### Anti-Features / Scope Boundaries

| Anti-Feature | Why avoid | Do instead |
|--------------|-----------|------------|
| External search engine for v2 | Violates milestone constraint of no new middleware | Use current product DB/indexes first |
| Semantic/vector search | Hard to justify before deterministic search is excellent | Improve ranking and normalization |
| Search across external streaming catalogs | Not aligned with self-hosted personal-media scope | Keep search limited to Mibo-managed media |
| Overbuilt advanced query language | High complexity, low early value | Use simple query + filters |

### Expected milestone scope

**In scope:** title/original title/cast/director search, typed results, sorting, highlighting, recent queries.  
**Out of scope:** natural-language search, “did you mean” service, external index cluster, multi-source federated search.

## Category 2: Rich Filters

### How mature products typically behave

- Filters are usually attached to a library/discovery view, not a separate power-user screen.
- The common baseline is **genre, year, library, watched/unwatched, rating**, then sort.
- Better products preserve filter state while navigating.
- Filter UI is only trusted when the backing metadata is normalized and sparse values are handled clearly.

### Table Stakes

| Feature | Why expected | Complexity | Scope guidance for Mibo |
|---------|--------------|------------|--------------------------|
| Genre filter | Standard browse control | Low | Multi-select if practical; single-select acceptable |
| Year filter | Standard library narrowing | Low | Exact year first; range later |
| Library filter | Essential for multi-library households | Low | Required |
| Watched / unwatched filter | Common personal-media expectation | Low | Required |
| Rating filter | Expected where TMDB/metadata ratings exist | Medium | Prefer threshold or buckets, not freeform |
| Resolution filter | Useful for playback-conscious users | Medium | Only if resolution metadata is already reliable |
| Sort controls | Users expect release/date added/title/rating sorts | Low | Required |

### Differentiators Worth Doing

| Feature | Value | Complexity | Recommendation |
|---------|-------|------------|----------------|
| Sticky filter state per view | Makes large-library browsing feel deliberate | Low | Yes |
| Dynamic facet counts | Helps users understand inventory | Medium | Good if cheap from current data model |
| Region / country filter | Useful for discovery-oriented users | Medium | Only if metadata coverage is strong |
| Compound filtering across discovery surfaces | Strong UX if shared API model exists | Medium | Yes |

### Anti-Features / Scope Boundaries

| Anti-Feature | Why avoid | Do instead |
|--------------|-----------|------------|
| Massive advanced filter builder | Too much UI and query complexity for first release | Ship a small, obvious facet set |
| Filters on weak or sparsely populated metadata | Produces distrust fast | Hide facets until coverage is reliable |
| Per-user saved smart collections in v2 | Valuable, but expands into collection product work | Start with session/user filter persistence only |

### Expected milestone scope

**In scope:** genre, year, region, rating, watched state, library, resolution, sort.  
**Out of scope:** saved smart playlists, nested boolean filter builders, cross-library recommendation logic.

## Category 3: Trailer Playback

### How mature products typically behave

- Trailers are a **detail-page enhancement**, not a separate browsing system.
- Products prefer an **official trailer** when possible and gracefully hide the feature when unavailable.
- Trailers are usually embedded from an external source rather than re-ingested into the user’s own library.
- Language/region mismatch is common, so fallback behavior matters.

### Table Stakes

| Feature | Why expected | Complexity | Scope guidance for Mibo |
|---------|--------------|------------|--------------------------|
| “Watch Trailer” action on detail page | Clear and familiar affordance | Low | Show only when a trailer exists |
| External-source playback | Typical implementation pattern | Medium | Use trusted provider URLs only |
| Prefer official trailer records | Avoid junk clips and fan uploads | Medium | Rank by `official`, `type=Trailer`, locale |
| Graceful absence handling | Many items simply won’t have trailers | Low | Hide CTA instead of broken player |

### Differentiators Worth Doing

| Feature | Value | Complexity | Recommendation |
|---------|-------|------------|----------------|
| Trailer prefetch/sync job | Improves detail-page readiness | Medium | Yes |
| Locale-aware trailer selection | Better UX for multilingual households | Medium | Yes if metadata locale model already exists |
| Multiple trailer choices (official/teaser/clip) | Nice power-user affordance | Medium | Defer unless easy |

### Anti-Features / Scope Boundaries

| Anti-Feature | Why avoid | Do instead |
|--------------|-----------|------------|
| Downloading and storing trailer media in v2 | Adds storage, rights, cleanup, and transcoding complexity | Store references/URLs only |
| Building a trailer catalog page | Low leverage compared with detail-page CTA | Keep trailers attached to item details |
| Treating trailers as guaranteed content | External providers are incomplete | Make trailer support opportunistic |

### Expected milestone scope

**In scope:** fetch trailer metadata from TMDB/external source, choose best candidate, play from item detail page.  
**Out of scope:** trailer downloads, offline trailers, trailer recommendation feeds, trailer transcoding pipeline.

## Category 4: Metadata Management

### How mature products typically behave

- Automatic matching does most of the work, but admins are expected to correct mistakes.
- Manual edits usually **lock fields** so refreshes do not clobber them.
- Rematch and refresh are separate actions: one changes identity/provider mapping, the other re-pulls data.
- Artwork editing is a first-class capability, not an obscure admin hack.

### Table Stakes

| Feature | Why expected | Complexity | Scope guidance for Mibo |
|---------|--------------|------------|--------------------------|
| Edit title/original title/year/summary | Core correction workflow | Medium | Required |
| Edit poster/backdrop | Common artwork correction path | Medium | Required |
| Edit genres/cast/basic credits | Needed for discovery quality | Medium | Required |
| Edit season/episode metadata | Required for TV correctness | High | Limit to key fields first |
| Re-match an item | Standard escape hatch for wrong identity | High | Required |
| Refresh metadata | Standard maintenance action | Medium | Required |
| Lock fields | Expected once manual edits exist | Medium | Required |

### Differentiators Worth Doing

| Feature | Value | Complexity | Recommendation |
|---------|-------|------------|----------------|
| Field-level lock visibility | Builds admin trust | Low | Yes |
| Provider/source attribution in UI | Helps explain where bad metadata came from | Low | Yes |
| Targeted refresh after edit/rematch | Keeps operations fast and safe | Medium | Yes |
| Bulk operations for narrow cases | Useful on large libraries | High | Defer except maybe artwork refresh by library |

### Anti-Features / Scope Boundaries

| Anti-Feature | Why avoid | Do instead |
|--------------|-----------|------------|
| Spreadsheet-style bulk editor | High complexity and high blast radius | Focus on strong single-item editing |
| Raw provider/debug controls for all users | Too technical and error-prone | Keep advanced identity controls admin-only |
| Letting refresh overwrite locked fields | Violates core user expectation | Respect lock semantics everywhere |

### Expected milestone scope

**In scope:** single-item admin editing, artwork replacement, field locks, rematch, refresh, TV season/episode basics.  
**Out of scope:** large-scale bulk editing suite, metadata version history, collaborative moderation workflows.

## Category 5: Storage-Change Scan Listeners

### How mature products typically behave

- Real-time monitoring is treated as a **library freshness accelerator**, not a guarantee that every raw filesystem event maps 1:1 to UI changes.
- Mature products debounce/coalesce events and enqueue **targeted refresh work**, rather than rescanning the world on each event.
- Safe delete handling is conservative because network mounts and cloud-backed storage are noisy.
- Monitoring support is often conditional on storage/filesystem capabilities.

### Table Stakes

| Feature | Why expected | Complexity | Scope guidance for Mibo |
|---------|--------------|------------|--------------------------|
| Detect create/update/delete/move class of changes | Core listener purpose | High | Required, but normalize into refresh intents |
| Trigger targeted refresh jobs | Avoid full-scan behavior | High | Required |
| Debounce bursty events | Necessary for real storage behavior | Medium | Required |
| Conservative delete handling | Prevents accidental item loss | High | Required |
| Visibility that listener is enabled/disabled | Admin expectation | Low | Required |

### Differentiators Worth Doing

| Feature | Value | Complexity | Recommendation |
|---------|-------|------------|----------------|
| Event coalescing by path/library | Better performance and fewer bad refreshes | Medium | Yes |
| Health/status indicators for listener pipelines | Helps admins trust automation | Medium | Yes |
| Fallback polling or scheduled reconciliation | Makes listener failures survivable | Medium | Yes |

### Anti-Features / Scope Boundaries

| Anti-Feature | Why avoid | Do instead |
|--------------|-----------|------------|
| Immediate full rescan on every event | Operationally expensive and noisy | Queue targeted refresh |
| Aggressive delete-on-missing semantics | Dangerous on flaky mounts/cloud backends | Use delayed confirmation / reconciliation |
| Deep OpenList business-logic coupling | Violates architecture constraint | Keep event handling inside `mibo-media-server` abstractions |

### Expected milestone scope

**In scope:** storage-change listener registration, safe event normalization, targeted refresh enqueueing, admin visibility, fallback scheduled reconciliation.  
**Out of scope:** exactly-once event guarantees, cross-provider event unification framework, complex user-defined listener rules.

## Category 6: Scheduled Task Management

### How mature products typically behave

- Scheduled tasks are visible in an admin dashboard with **run-now**, **next run**, **last run**, and **status**.
- Tasks have a small set of useful triggers: interval, daily, weekly, startup.
- Plugins or subsystems may contribute tasks, but operators still expect one control surface.
- Task history matters because background automation is otherwise impossible to trust.

### Table Stakes

| Feature | Why expected | Complexity | Scope guidance for Mibo |
|---------|--------------|------------|--------------------------|
| List all scheduled tasks | Basic operability | Low | Required |
| Enable/disable tasks | Basic control | Low | Required |
| Run task manually | Standard admin workflow | Low | Required |
| Configure interval/schedule | Baseline scheduler behavior | Medium | Required |
| Show next/last run and result | Required for trust | Medium | Required |
| Cover core task types | Users expect scans, metadata refresh, cleanup, trailer sync | Medium | Required |

### Differentiators Worth Doing

| Feature | Value | Complexity | Recommendation |
|---------|-------|------------|----------------|
| Concurrency policy / no-overlap behavior | Prevents duplicate heavy jobs | Medium | Yes |
| Per-task history / logs | High admin value | Medium | Yes |
| Startup trigger and sleep-resume style triggers | Useful for home servers | Medium | Yes if easy |
| Shared task framework for future jobs | Good architecture leverage | Medium | Yes |

### Anti-Features / Scope Boundaries

| Anti-Feature | Why avoid | Do instead |
|--------------|-----------|------------|
| Full workflow automation engine | Massive scope increase | Keep simple periodic tasks |
| Cron-expression-only UX | Too technical for many self-hosters | Offer common schedule presets first |
| Exposing dozens of low-level maintenance jobs initially | Confusing first admin experience | Start with a curated task list |

### Expected milestone scope

**In scope:** scan, metadata refresh, trailer sync, stale link check, artwork refresh, cleanup-style tasks; run now; enable/disable; simple schedules; status/history.  
**Out of scope:** arbitrary DAG workflows, third-party automation marketplace, distributed task orchestration.

## Discovery vs Governance Dependencies

| User-facing capability | Depends on admin/operational capability | Why |
|------------------------|----------------------------------------|-----|
| Search relevance | Metadata quality + rematch/locks | Bad metadata means bad search |
| Filter accuracy | Normalized metadata + scheduled refresh | Facets collapse if data is stale or inconsistent |
| Trailer availability | Correct item identity + trailer sync task | Wrong match means wrong/missing trailer |
| Detail page trust | Artwork edits + locked overrides | Users notice bad posters and summaries immediately |
| Freshly added content appearing in discovery | Scan listeners + scheduled reconciliation | Discovery quality depends on ingestion freshness |
| Stable discovery over time | Task visibility/history | Admins need to know when automation failed |

## Feature Dependencies

```text
stable item identity
  → metadata match/rematch
  → editable metadata + field locks
  → searchable fields + filter facets + trailer source lookup

scan listeners
  → targeted refresh jobs
  → metadata refresh / trailer refresh / index refresh
  → fresher search and browse surfaces

scheduled task management
  → recurring metadata sync / cleanup / trailer sync / reconciliation
  → long-term freshness and operability
```

## MVP Recommendation For This Milestone

Prioritize in this order:

1. **Metadata governance foundation**
   - Single-item edit, artwork change, field locks, rematch, refresh.
   - Why: search, filters, and trailers are all downstream of this.

2. **Search**
   - Title + actor + director, typed results, stable ranking, highlights.
   - Why: biggest immediate user-facing value.

3. **Filters**
   - Genre, year, watched, library, rating, resolution, sort.
   - Why: complements search and upgrades browsing for large libraries.

4. **Scheduled task management**
   - Run now, schedule, status/history for scan/metadata/trailer/cleanup jobs.
   - Why: gives admins visibility and a stable operating surface.

5. **Storage-change scan listeners**
   - Conservative event handling + targeted refresh + reconciliation fallback.
   - Why: freshness matters, but correctness matters more than immediacy.

6. **Trailer playback**
   - Detail-page CTA with provider-backed external playback.
   - Why: good discovery enhancer, but not foundational.

## Recommended Differentiators For Mibo Specifically

These are the best “extra” features for this milestone because they compound well:

1. **Title-first weighted search ranking** instead of generic full-text search.
2. **Field-level metadata locks** that every refresh path respects.
3. **Conservative storage-event → targeted-refresh pipeline** instead of naive auto-scans.
4. **One unified task surface** for scans, metadata, trailers, and cleanup.

## Explicit Scope Guardrails

- Do **not** introduce Elasticsearch/OpenSearch/Meilisearch in v2.
- Do **not** build semantic search or recommendation ML.
- Do **not** download/manage trailer files as first-class library media.
- Do **not** build a bulk metadata operations suite beyond narrow, safe actions.
- Do **not** turn scheduled tasks into a generic workflow engine.
- Do **not** let storage listeners directly mutate library state without going through safe refresh jobs.

## Sources

### Project Context

- `/Users/atlan/Desktop/IdeaProjects/Mibo/.planning/PROJECT.md` — milestone goals, active requirements, constraints. **Confidence: HIGH**

### Official Ecosystem Sources

- Jellyfin Libraries — typed libraries, multi-path libraries, mixed-content warning: https://jellyfin.org/docs/general/server/libraries **Confidence: HIGH**
- Jellyfin Metadata — provider-backed metadata model: https://jellyfin.org/docs/general/server/metadata/ **Confidence: HIGH**
- Jellyfin Tasks — scheduled operations, manual trigger, built-in library/maintenance tasks: https://jellyfin.org/docs/general/server/tasks/ **Confidence: HIGH**
- Jellyfin Quick Start — setup and media-add flow as baseline product expectation: https://jellyfin.org/docs/general/quick-start/ **Confidence: HIGH**
- Plex What is Plex? — auto-cataloging, artwork/info enrichment, device coverage, watch tracking positioning. Last modified Jan 7 2025: https://support.plex.tv/articles/200288286-what-is-plex/ **Confidence: HIGH**
- Plex Edit Details — manual metadata edits, artwork changes, lock semantics. Last modified Mar 17 2026: https://support.plex.tv/articles/201272763-edit-details/ **Confidence: HIGH**
- Plex Your Media — automatic organization and metadata-rich library positioning: https://www.plex.tv/your-media/ **Confidence: MEDIUM**
- Emby Library Setup — library typing, multi-path libraries, direct path ideas, real-time monitoring option: https://emby.media/support/articles/Library-Setup.html **Confidence: HIGH**
- Emby Scheduled Tasks — task triggers, manual runs, admin scheduling surface: https://emby.media/support/articles/Scheduled-Tasks.html **Confidence: HIGH**
- TMDB API videos endpoints — trailer/teaser retrieval fields including `site`, `type`, `official`, `published_at`: /websites/developer_themoviedb_reference via Context7 **Confidence: HIGH**

## Confidence Notes

- **HIGH confidence:** metadata editing/locking, scheduled tasks, real-time monitoring as mature-product norms.
- **MEDIUM-HIGH confidence:** search/filter behavior recommendations; these are consistent with product norms, though official docs are less explicit than metadata/task docs.
- **MEDIUM confidence:** trailer UX expectations; technical source support is strong, but exact product norms vary more by platform.
