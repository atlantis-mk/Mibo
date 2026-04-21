# Roadmap: Mibo

## Overview

This roadmap turns Mibo from a working prototype into a stable media platform by hardening the access boundary first, then making ingestion reliable, promoting files into a semantic catalog, exposing product-grade playback and progress APIs, improving playback decisions, and finally adding resilient identity and incremental refresh.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Access & Platform Boundary** - Users enter the app through a stable authenticated boundary that hides storage-provider details. Completed 2026-04-21.
- [x] **Phase 2: Library & Async Sync Foundation** - Admins connect storage and run scans as background work instead of blocking requests. Completed 2026-04-22.
- [x] **Phase 3: Semantic Catalog & Discovery** - Users browse a semantic media catalog rather than raw files. Completed 2026-04-21.
- [x] **Phase 4: Playback Entry & Unified Progress** - Users can start playback and resume consistently across clients. (completed 2026-04-21)
- [x] **Phase 5: Playback Decision Intelligence** - Playback selection becomes capability-aware with direct-play-first behavior. Completed 2026-04-22.
- [ ] **Phase 6: Stable Identity & Incremental Refresh** - Libraries stay fresh and resilient when files change over time.

## Phase Details

### Phase 1: Access & Platform Boundary
**Goal**: Users can initialize Mibo, sign in, and rely on one stable media API boundary while storage implementation details stay hidden behind `mibo-media-server`.
**Depends on**: Nothing (first phase)
**Requirements**: ACCS-01, ACCS-02, ACCS-03, CATA-01
**Success Criteria** (what must be TRUE):
  1. An administrator can complete setup and reach the main application flow without manual backend intervention.
  2. A user can sign in once and continue using protected media APIs through a persistent authenticated session.
  3. Web clients can use one stable HTTP media API shape that is suitable to keep for later mobile and TV clients.
  4. Client-visible media APIs stay media-centric and do not expose OpenList-specific concepts or payloads.
**Plans**: 2 (`01-PLAN-01`, `01-PLAN-02`)
**UI hint**: yes

### Phase 2: Library & Async Sync Foundation
**Goal**: Administrators can connect storage-backed libraries and trust scans/refreshes to run asynchronously without degrading interactive requests.
**Depends on**: Phase 1
**Requirements**: LIBR-01, LIBR-02, LIBR-03, LIBR-04, CATA-06
**Success Criteria** (what must be TRUE):
  1. An administrator can add media sources backed by local storage, NAS-style paths, or cloud-backed storage exposed through the provider boundary.
  2. An administrator can create a library, bind it to a source and root path, and save that configuration successfully.
  3. When an administrator triggers a scan, the app shows that work is queued and processed asynchronously rather than hanging the request.
  4. Scheduled refreshes can be configured so library updates continue happening without manual rescans.
**Plans**: 3 plans

Plans:
- [x] 02-01-PLAN.md — Backend async scan settings, scheduled refresh, and jobs filtering contracts
- [x] 02-02-PLAN.md — Web admin source/library flow, status badges, jobs monitoring, and refresh controls
- [x] 02-03-PLAN.md — Close the auth-boundary verification gap for admin source/library/scan/jobs endpoints

### Phase 3: Semantic Catalog & Discovery
**Goal**: Users can explore a durable media catalog organized as movies and shows with useful metadata and library-aware discovery.
**Depends on**: Phase 2
**Requirements**: CATA-02, CATA-03, CATA-04, CATA-05
**Success Criteria** (what must be TRUE):
  1. Newly scanned files appear in a trackable catalog instead of existing only as transient scan output.
  2. TV content appears as series, seasons, and episodes, and films appear as standalone media items.
  3. Users see posters, summaries, and core item details instead of only raw filenames.
  4. Users can browse by library, filter, search, and open a media detail page for a chosen item.
**Plans**: 3 (`03-PLAN-01`, `03-PLAN-02`, `03-PLAN-03`)
**UI hint**: yes

### Phase 4: Playback Entry & Unified Progress
**Goal**: Users can start playback from catalog surfaces and have resume state persist through one client-facing progress model.
**Depends on**: Phase 3
**Requirements**: PLAY-01, PROG-01, PROG-02
**Success Criteria** (what must be TRUE):
  1. A user can open a media detail page, request playback, and receive a playback entry that works for the current client.
  2. A user's in-progress playback position is saved durably while watching.
  3. The same user can leave playback on one client and resume from the saved position on another client through the same API model.
**Plans**: 4 plans
 
Plans:
- [x] 04-01-PLAN.md — Backend playback auth and canonical progress merge semantics
- [x] 04-02-PLAN.md — Frontend playback route intent contract and controller seam
- [x] 04-03-PLAN.md — Home/detail/playback UI wiring for resume and restart behavior
- [x] 04-04-PLAN.md — Manual end-to-end playback/progress verification
**UI hint**: yes

### Phase 5: Playback Decision Intelligence
**Goal**: Playback becomes more reliable across device types by choosing the best available path using media facts and explicit fallback behavior.
**Depends on**: Phase 4
**Requirements**: PLAY-02, PLAY-03
**Success Criteria** (what must be TRUE):
  1. When a client can direct play a title, the playback response prefers a direct path.
  2. When direct play is not viable, the playback response provides a clear fallback path instead of failing ambiguously.
  3. Playback choices improve when richer stream facts are available from probe data.
**Plans**: 2 plans

Plans:
- [x] 05-01-PLAN.md — Backend explicit client-profile playback contract, probe-aware decision engine, and per-request HLS fallback
- [x] 05-02-PLAN.md — Web typed playback contract consumption and decision-aware playback page behavior

### Phase 6: Stable Identity & Incremental Refresh
**Goal**: Libraries remain accurate over time as files move, rename, or change, without relying on full rescans for every update.
**Depends on**: Phase 5
**Requirements**: SYNC-01, SYNC-02, SYNC-03
**Success Criteria** (what must be TRUE):
  1. Renaming, moving, or remounting files does not easily create duplicate media entries or lose the user's playback progress.
  2. Routine library changes can be picked up through incremental refresh instead of requiring a full scan every time.
  3. Storage change events can safely trigger targeted refresh or resync work.
**Plans**: 4 plans

Plans:
- [ ] 06-01-PLAN.md — Stable identity evidence contract and scan ingestion that stops path-first primary matching
- [ ] 06-02-PLAN.md — Conservative size+duration fallback reconciliation with ambiguity quarantine
- [ ] 06-03-PLAN.md — Targeted incremental refresh jobs and subtree-safe partial scan behavior
- [ ] 06-04-PLAN.md — Authenticated storage-event intake that enqueues safe refresh work

## Progress

**Execution Order:**
Phases execute in numeric order: 2 → 2.1 → 2.2 → 3 → 3.1 → 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Access & Platform Boundary | 2/2 | Complete | 2026-04-21 |
| 2. Library & Async Sync Foundation | 3/3 | Complete | 2026-04-22 |
| 3. Semantic Catalog & Discovery | 3/3 | Complete | 2026-04-21 |
| 4. Playback Entry & Unified Progress | 4/4 | Complete   | 2026-04-21 |
| 5. Playback Decision Intelligence | 2/2 | Complete | 2026-04-22 |
| 6. Stable Identity & Incremental Refresh | 0/TBD | Not started | - |
