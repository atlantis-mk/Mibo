# Project Research Summary

**Project:** Mibo
**Domain:** Self-hosted household media server
**Researched:** 2026-04-21
**Confidence:** HIGH

## Executive Summary

Mibo is not a generic file browser and should not evolve like one. The research is consistent across stack, features, architecture, and pitfalls: successful household media servers keep storage access at the edge, move media semantics into the core service, and run expensive work asynchronously. For Mibo, that means keeping OpenList as a storage gateway, making `mibo-media-server` the durable business core, and building the product around a DB-backed media catalog, playback decisioning, and multi-client progress sync.

The recommended approach is a pragmatic monolith with strong internal seams: Go stays as the backend core, PostgreSQL becomes the production default, River becomes the job spine, and React/Vite stays on the frontend with TanStack Query added for server-state discipline. Product work should center on reliable ingestion, semantic media modeling, direct-play-first playback with fallback, and stable progress APIs rather than chasing Plex-scale feature breadth.

The main risks are architectural, not cosmetic: leaking OpenList details into product APIs, using path-based identity, overloading scan flows with metadata/probe work, and treating playback as “just return a URL.” Mitigation is clear from the research: harden the `StorageProvider` boundary first, split scan from enrichment jobs, add stable identity and confidence models early, and instrument the system before adding performance complexity.

## Key Findings

### Recommended Stack

The stack recommendation is unusually clear: reinforce the existing Go + React architecture instead of rewriting it. The backend should become a Go media core on PostgreSQL with River-backed jobs, ffprobe/ffmpeg worker tooling, and OpenTelemetry + Prometheus from the start of the next milestone. OpenList remains the first storage adapter, not the source of product semantics.

On the frontend, React + Vite remain good fits, but the missing discipline is server-state management. TanStack Query should be added next so library views, detail pages, playback state, and progress sync stop depending on ad-hoc fetch/state patterns.

**Core technologies:**
- **Go 1.24.x**: backend core — already the right fit for long-running media orchestration and HTTP services.
- **PostgreSQL 17/18**: primary production DB — needed for durable metadata, progress, and concurrent job orchestration.
- **River + pgx**: background jobs — the recommended durable queue for scan, match, probe, and later transcode work.
- **GORM**: app data access — keep it, but use it more explicitly and avoid domain leakage.
- **OpenList over HTTP**: storage gateway — keep as first adapter while preserving a strict provider boundary.
- **ffprobe / ffmpeg**: media analysis and fallback playback — direct play first, transcode only when necessary.
- **OpenTelemetry + Prometheus**: observability — required before worker complexity and performance tuning grow.
- **React + Vite + TanStack Query + Shaka Player**: web client stack — stable SPA base with proper server-state handling and browser playback.

### Expected Features

The research draws a hard line between table stakes and differentiators. Table stakes are the core personal-media loop: setup, source/library management, scan/refresh, automatic organization into movie/show structures, metadata enrichment, browse/search/details, reliable playback, resume/progress sync, authentication, and an API shape that can support Web now and mobile/TV later. Mibo already has partial coverage in several of these areas, but they must be made reliable enough to feel like product fundamentals rather than prototype behavior.

Differentiation should come from architecture-aligned strengths: unified ingestion across mixed storage, stable identity that survives moves/renames, incremental sync, fast-path scanning with deferred heavy work, direct-play-first playback policy, and a unified progress/history model across clients. The research is equally clear on what to defer: Live TV/DVR, social/watch-party features, aggressive transcoding pipelines, deep OpenList customization, and broad non-video media expansion.

**Must have (table stakes):**
- Setup/admin onboarding — fast path to usable libraries.
- Source and library management — storage selection, path validation, typed libraries.
- Manual + scheduled scan/refresh — users expect newly added media to appear.
- Semantic organization — movies, series, seasons, episodes.
- Metadata enrichment — posters, synopsis, cast/basic details.
- Browse, filter, search, and detail pages — DB-backed, predictable, multi-client friendly.
- Reliable playback — direct link first with graceful fallback.
- Resume playback and progress sync — durable across clients.
- Authentication and household-safe user separation — baseline for real family usage.

**Should have (competitive):**
- Unified storage ingestion across local/NAS/cloud behind adapters.
- Stable file identity and rename/move resilience.
- Fast-path scan pipeline with deferred metadata/probe work.
- Incremental and event-driven sync after full-scan correctness is proven.
- Unified progress/history model for Web, mobile, and TV clients.
- Continue Watching and Recently Added surfaces once semantics and progress are stable.

**Defer (v2+):**
- Live TV / DVR.
- Music / photos / books expansion.
- Watch-party / social features.
- Deep OpenList fork or custom storage stack.
- Always-on pre-transcoding or optimization pipelines.

### Architecture Approach

The architecture recommendation is to keep a single media-core service with explicit internal seams. Clients talk only to Mibo APIs; Mibo owns semantic media data, playback policy, progress, and jobs; OpenList remains a file-access boundary. The system should be built around DB-backed read models, an async worker pipeline, and a stable `StorageProvider` abstraction so future direct adapters remain optional optimizations rather than rewrites.

**Major components:**
1. **API + admin/query services** — own auth, setup, library management, browse/search/detail APIs, and client-facing contracts.
2. **Playback + progress services** — resolve semantic items to playable sessions and manage durable resume/watch state.
3. **Job system + workers** — run scan, classify, metadata match, probe, and later transcode/incremental refresh work asynchronously.
4. **StorageProvider boundary** — normalize file access behind adapters and prevent OpenList leakage into business logic.
5. **Database-backed semantic model** — store libraries, media files, media graph, jobs, and user playback state as the source of truth.

### Critical Pitfalls

The pitfalls research strongly reinforces the architecture guidance: most failures come from weak boundaries and premature shortcuts, not from missing shiny features.

1. **Path-based identity** — treat path as mutable metadata, add stable file identity early, and attach progress to media/version identity instead of scan rows.
2. **Scan loop doing everything** — keep scanning as fast discovery and split match/probe/playback prep into retryable job stages.
3. **Weak metadata confidence model** — separate parsing from matching, store confidence, and plan for manual correction flows.
4. **OpenList details leaking upward** — keep APIs media-centric and normalize all provider output before domain code sees it.
5. **Playback modeled as URL generation** — build a capability-aware decision service with direct play, remux/direct stream, and transcode fallback.
6. **Naive progress sync** — separate playback sessions from canonical resume state and define merge rules for concurrent clients.

## Implications for Roadmap

Based on the combined research, the roadmap should be dependency-first, not UI-first. The correct sequencing is: stabilize boundary and jobs, build the catalog, promote files into semantic media entities, expose stable query APIs, then harden playback/progress, and only then add technical playback intelligence and incremental sync.

### Phase 1: Boundary and Job Foundation
**Rationale:** Everything else depends on a stable storage contract and durable async orchestration.
**Delivers:** Hardened `StorageProvider`, explicit API/worker separation, PostgreSQL production path, River job model, basic observability.
**Addresses:** Source/library management, scheduled/manual sync foundation, operational simplicity.
**Avoids:** OpenList leakage, path-identity mistakes, premature microservice drift.

### Phase 2: Scan Catalog Pipeline
**Rationale:** Mibo needs a trustworthy ingestion backbone before it can improve UX.
**Delivers:** Canonical `sync_library` flow, `media_files` catalog, checkpoints, retries, bounded scan/probe concurrency, fast-path discovery.
**Uses:** PostgreSQL, River, OpenTelemetry, Prometheus, existing Go worker skeleton.
**Implements:** Scanner worker and DB-backed ingestion pipeline.
**Avoids:** Full understanding inside scan loop, unbounded worker pressure.

### Phase 3: Semantic Media Graph and Metadata Confidence
**Rationale:** Browse, detail, search, and playback all depend on stable media semantics, not raw files.
**Delivers:** `media_items / series / seasons / episodes`, metadata matching pipeline, confidence scoring, typed-library correctness, manual-fix hooks.
**Addresses:** Automatic organization, metadata enrichment, detail pages, browse/search quality.
**Avoids:** Wrong-match sprawl, mixed-library hacks, fragile filename assumptions.

### Phase 4: Client-Facing Query, Playback, and Progress Core
**Rationale:** This is the first phase that should feel fully product-grade to end users.
**Delivers:** Home/library/detail/search APIs, direct-play-first playback decision service, stable progress/session model, Continue Watching groundwork, TanStack Query integration on web.
**Addresses:** Reliable playback, resume sync, multi-device API readiness, household-facing home surfaces.
**Avoids:** Playback-as-URL shortcut, web-only API bias, naive last-write-wins progress.

### Phase 5: Probe Intelligence and Incremental Sync
**Rationale:** Add smarter playback and day-2 operational quality only after correctness is established.
**Delivers:** Async ffprobe facts, richer playback capability decisions, targeted rescans, scheduled reconciliation, initial event-driven refresh where safe.
**Addresses:** Better client compatibility, rename/move resilience, faster library freshness.
**Avoids:** Shipping delta sync before correctness, blind playback decisions, unsupported-client surprises.

### Phase 6: Fallback Transcoding and Hotspot Optimization
**Rationale:** Transcoding and direct adapters are valuable, but only after evidence justifies the complexity.
**Delivers:** HLS/remux/transcode fallback, optional worker split, targeted caching or Redis only if metrics prove need, direct adapters only for measured hotspots.
**Addresses:** Harder client/storage scenarios and larger-library robustness.
**Avoids:** Performance theater, premature queue/cache architecture, overbuilt deployment topology.

### Phase Ordering Rationale

- Storage boundary, jobs, and identity come first because every later capability depends on them.
- Semantic modeling must precede polished client experiences; otherwise the UI is built on unstable file semantics.
- Playback and progress belong together because both depend on semantic identity and stable client-facing contracts.
- Probe intelligence, incremental sync, and transcode fallback should be delayed until the core model is correct and observable.
- Performance optimizations should follow metrics, not fear.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3:** metadata confidence/manual-fix design, especially around specials, alternate ordering, and multi-episode edge cases.
- **Phase 4:** playback capability contract for Web/mobile/TV and progress conflict semantics across multiple clients.
- **Phase 5:** event-driven/incremental sync strategy for OpenList-backed storage without sacrificing reconciliation correctness.
- **Phase 6:** transcode policy and hardware acceleration choices for realistic household deployment targets.

Phases with standard patterns (likely skip research-phase):
- **Phase 1:** PostgreSQL + River + observability + strict adapter boundary are well-supported patterns.
- **Phase 2:** async scan/catalog pipelines with bounded worker concurrency are established and already aligned with current architecture.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Strong alignment between project context and official docs for PostgreSQL, River, OTel, Prometheus, and frontend additions. |
| Features | MEDIUM-HIGH | Competitor baselines are clear, but exact milestone cut lines still require product judgment. |
| Architecture | HIGH | Local architecture docs and current code boundaries strongly agree on the recommended direction. |
| Pitfalls | MEDIUM-HIGH | Most pitfalls are reinforced by official Jellyfin/Plex guidance plus clear project-specific risks; progress conflict semantics need some design validation. |

**Overall confidence:** HIGH

### Gaps to Address

- **Stable identity implementation details:** decide what provider-level identity guarantees OpenList can expose and how to reconcile partial support.
- **Metadata correction UX:** research and define the minimal manual-fix/admin workflow before broad rollout of aggressive matching.
- **Playback capability schema:** formalize how clients declare codec/container/subtitle/HDR capabilities.
- **Progress conflict rules:** validate merge behavior for dual-device use, seek/backtrack, and completion semantics.
- **Incremental sync semantics:** define rename/delete/partial-upload behavior before trusting events over reconciliation.
- **Production deployment guidance:** validate recommended Postgres + worker + ffmpeg packaging on Mibo’s expected self-hosted targets.

## Sources

### Primary (HIGH confidence)
- `.planning/PROJECT.md` — product scope, constraints, existing capabilities, and target evolution.
- `docs/media-architecture/improved-architecture.md` — target system boundaries and build order.
- PostgreSQL official docs — production DB recommendation.
- River docs (`/riverqueue/river`) — durable PostgreSQL-backed jobs and transactional enqueue.
- OpenTelemetry Go docs (`/open-telemetry/opentelemetry-go`) — tracing/metrics instrumentation model.
- Prometheus Go client docs — service metrics export.
- Jellyfin docs (libraries, metadata, transcoding, codec support) — feature baseline and operational constraints.

### Secondary (MEDIUM confidence)
- Plex official support/docs — market baseline for table stakes and playback model.
- Navidrome docs — supporting pattern for app-owned scanning/transcoding boundaries.
- Package/version checks captured in `STACK.md` — current ecosystem versions for recommended additions.

### Tertiary (LOW confidence)
- None material to the core recommendation; the main open items are design gaps, not source quality gaps.

---
*Research completed: 2026-04-21*
*Ready for roadmap: yes*
